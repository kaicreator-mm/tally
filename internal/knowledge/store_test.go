package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/core"
	"tally/internal/store"
)

// —— 测试替身 ——

// fakeProof 实现 core.VerificationProof。只有 failure 的四道闸能产出真实证据;
// 这里是测试桩,用来验证 Store 对证据参数的要求与有效性检查。
type fakeProof struct {
	at    time.Time
	combo core.VersionCombo
	gates string
}

func (p fakeProof) VerifiedAt() time.Time            { return p.at }
func (p fakeProof) VerifiedCombo() core.VersionCombo { return p.combo }
func (p fakeProof) GateRecord() string               { return p.gates }
func (p fakeProof) Valid() bool                      { return !p.at.IsZero() }

func goodProof(combo core.VersionCombo) fakeProof {
	return fakeProof{
		at:    time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC),
		combo: combo,
		gates: "g1=5/5red g2=5/5green g3=1/1targets g4=2files-ok",
	}
}

// fakeStats 实现 StatLister(模糊召回索引),离线可测。
type fakeStats struct {
	rows []store.FingerprintStat
	err  error
}

func (f *fakeStats) ListFingerprintStatsByFuzzy(_ context.Context, _ string) ([]store.FingerprintStat, error) {
	return f.rows, f.err
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("初始化 git 仓库: %v", err)
	}
	return dir
}

func testFix() core.Fix {
	return core.Fix{
		Repro:   "git checkout abc && make build",
		Summary: "锁定依赖版本,避开上游破坏性变更",
		Changes: []core.FileChange{{Path: "go.mod", Content: "module demo\n"}},
	}
}

const testFP = core.Fingerprint("a3f9c2e1b4d50607")

// —— 入库(写)——

// 验收:入库动作是 git commit,写入 knowledge/<指纹>/ 三件套,不写数据库。
func TestStoreWritesGitCommit(t *testing.T) {
	repo := initRepo(t)
	s := NewStore(repo, &fakeStats{})
	combo := core.VersionCombo{"go": "1.23.0"}

	ref, err := s.Store(context.Background(), testFP, testFix(), goodProof(combo))
	if err != nil {
		t.Fatalf("Store 出错: %v", err)
	}
	if ref == "" || strings.HasPrefix(ref, "000000") {
		t.Errorf("入库应返回真实 git 提交哈希, got %q", ref)
	}

	dir := filepath.Join(repo, "knowledge", string(testFP))
	for _, name := range []string{"fix.md", "repro.sh", "verified.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("缺少入库文件 %s: %v", name, err)
		}
	}

	// verified.json 是机器记录:有验证时刻/版本组合/闸门记录/修复内容,没有 LLM 分析。
	raw, err := os.ReadFile(filepath.Join(dir, "verified.json"))
	if err != nil {
		t.Fatalf("读 verified.json: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("verified.json 不是合法 JSON: %v", err)
	}
	for _, key := range []string{"verified_at", "verified_combo", "gates", "changes"} {
		if _, ok := rec[key]; !ok {
			t.Errorf("verified.json 缺字段 %s", key)
		}
	}
	if strings.Contains(string(raw), "root_cause") || strings.Contains(string(raw), "analysis") {
		t.Errorf("verified.json 不得存 LLM 分析过程: %s", raw)
	}

	// 留痕进了 git(HEAD 非零,提交信息带指纹)
	r, _ := git.PlainOpen(repo)
	head, err := r.Head()
	if err != nil || head.Hash().IsZero() {
		t.Fatalf("入库应产生 git 提交: %v", err)
	}
	if head.Hash().String() != ref {
		t.Errorf("返回的引用 %s 应等于 HEAD %s", ref, head.Hash())
	}
	commit, err := r.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("取提交: %v", err)
	}
	if !strings.Contains(commit.Message, string(testFP)) {
		t.Errorf("提交信息应带指纹: %q", commit.Message)
	}
}

// 验收(铁律 2):无有效验证证据不得入库 —— 零值/ nil 证据被拒。
func TestStoreRejectsInvalidProof(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})

	if _, err := s.Store(context.Background(), testFP, testFix(), nil); !errors.Is(err, ErrInvalidProof) {
		t.Errorf("nil 证据应被拒, got %v", err)
	}
	if _, err := s.Store(context.Background(), testFP, testFix(), fakeProof{}); !errors.Is(err, ErrInvalidProof) {
		t.Errorf("零值证据应被拒, got %v", err)
	}
}

func TestStoreRequiresRepro(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	fix := testFix()
	fix.Repro = ""
	if _, err := s.Store(context.Background(), testFP, fix, goodProof(nil)); err == nil {
		t.Error("缺复现命令不得入库")
	}
}

func TestStoreRejectsBadFingerprint(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	if _, err := s.Store(context.Background(), "../escape", testFix(), goodProof(nil)); err == nil {
		t.Error("非法指纹(路径注入)应被拒")
	}
}

// —— 查库:精确指纹 ——

// 验收:精确命中 → 直接返回修复,版本组合一致不标 stale。
func TestLookupExactRoundtrip(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	combo := core.VersionCombo{"go": "1.23.0", "node": "20.0.0"}
	if _, err := s.Store(context.Background(), testFP, testFix(), goodProof(combo)); err != nil {
		t.Fatalf("Store: %v", err)
	}

	hit, ok, err := s.LookupExact(context.Background(), testFP, combo)
	if err != nil || !ok {
		t.Fatalf("精确命中应 ok: ok=%v err=%v", ok, err)
	}
	if hit.Stale {
		t.Error("版本组合一致不应标 stale")
	}
	if hit.Repro != testFix().Repro || hit.Summary != testFix().Summary {
		t.Errorf("修复内容应原样取回: %+v", hit.Fix)
	}
	if len(hit.Changes) != 1 || hit.Changes[0].Path != "go.mod" {
		t.Errorf("改动应原样取回: %+v", hit.Changes)
	}
	if !hit.VerifiedAt.Equal(goodProof(combo).at) {
		t.Errorf("验证时刻应取回: %v", hit.VerifiedAt)
	}
}

// 验收:命中时比对版本组合,不一致标 stale 并降级为线索。
func TestLookupExactComboMismatchStale(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	if _, err := s.Store(context.Background(), testFP, testFix(), goodProof(core.VersionCombo{"go": "1.23.0"})); err != nil {
		t.Fatalf("Store: %v", err)
	}

	hit, ok, err := s.LookupExact(context.Background(), testFP, core.VersionCombo{"go": "1.24.0"})
	if err != nil || !ok {
		t.Fatalf("应命中: ok=%v err=%v", ok, err)
	}
	if !hit.Stale {
		t.Error("版本组合不一致应标 stale,降级为线索")
	}
}

// 当前组合新增了键(验证时的键都还在且相等)→ 不失效:新键不应作废老修复。
func TestLookupExactExtraCurrentKeyNotStale(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	if _, err := s.Store(context.Background(), testFP, testFix(), goodProof(core.VersionCombo{"go": "1.23.0"})); err != nil {
		t.Fatalf("Store: %v", err)
	}
	hit, ok, err := s.LookupExact(context.Background(), testFP, core.VersionCombo{"go": "1.23.0", "node": "22"})
	if err != nil || !ok {
		t.Fatalf("应命中: %v %v", ok, err)
	}
	if hit.Stale {
		t.Error("当前组合仅新增键不应标 stale")
	}
}

func TestLookupExactMissing(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	if _, ok, err := s.LookupExact(context.Background(), testFP, nil); err != nil || ok {
		t.Errorf("未入库应 ok=false 无错误: ok=%v err=%v", ok, err)
	}
}

// —— 老化标记 ——

// 验收:老化标记写进 git(留痕),标记后命中即 stale;记录保留不删。
func TestMarkStale(t *testing.T) {
	repo := initRepo(t)
	s := NewStore(repo, &fakeStats{})
	combo := core.VersionCombo{"go": "1.23.0"}
	if _, err := s.Store(context.Background(), testFP, testFix(), goodProof(combo)); err != nil {
		t.Fatalf("Store: %v", err)
	}

	if _, err := s.MarkStale(context.Background(), testFP, "依赖升级,复现命令不再失败"); err != nil {
		t.Fatalf("MarkStale: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "knowledge", string(testFP), "STALE.md")); err != nil {
		t.Errorf("老化标记应落盘: %v", err)
	}

	hit, ok, err := s.LookupExact(context.Background(), testFP, combo) // 组合其实还一致
	if err != nil || !ok {
		t.Fatalf("应仍命中(记录保留): %v %v", ok, err)
	}
	if !hit.Stale {
		t.Error("被标老化的修复命中时应 stale")
	}
}

func TestMarkStaleMissingEntry(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{})
	if _, err := s.MarkStale(context.Background(), testFP, "x"); err == nil {
		t.Error("不存在的条目不能标老化")
	}
}

// —— GitWriter 单一写边界(P0-19)——

// 验收(P0-19):入库提交经 gitwriter 单一写边界;内容与 HEAD 完全一致时不产生
// 空提交,重复入库不报错并返回当前 HEAD(引用仍有效)。verified.json 的验证时刻
// 取自 proof(定值),重复入库字节级一致,结果确定。
func TestStoreRepeatedIdenticalWriteReturnsHead(t *testing.T) {
	repo := initRepo(t)
	s := NewStore(repo, &fakeStats{})
	combo := core.VersionCombo{"go": "1.23.0"}

	first, err := s.Store(context.Background(), testFP, testFix(), goodProof(combo))
	if err != nil {
		t.Fatalf("首次 Store: %v", err)
	}
	second, err := s.Store(context.Background(), testFP, testFix(), goodProof(combo))
	if err != nil {
		t.Fatalf("内容一致的重复 Store 不应报错: %v", err)
	}
	if second != first {
		t.Errorf("内容一致的重复入库不应产生新提交: first=%s second=%s", first, second)
	}
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("打开仓库: %v", err)
	}
	head, err := r.Head()
	if err != nil {
		t.Fatalf("取 HEAD: %v", err)
	}
	if second != head.Hash().String() {
		t.Errorf("返回引用 %s 应等于当前 HEAD %s", second, head.Hash())
	}
}

// 验收(P0-19):重复标老化同样经 gitwriter;返回引用等于当前 HEAD 且不出错
// (标记时间带秒级时间戳:同一秒内重复 → 无新提交;跨秒 → 新提交,两者都合法)。
func TestMarkStaleTwice(t *testing.T) {
	repo := initRepo(t)
	s := NewStore(repo, &fakeStats{})
	combo := core.VersionCombo{"go": "1.23.0"}
	if _, err := s.Store(context.Background(), testFP, testFix(), goodProof(combo)); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if _, err := s.MarkStale(context.Background(), testFP, "依赖升级"); err != nil {
		t.Fatalf("MarkStale: %v", err)
	}
	ref, err := s.MarkStale(context.Background(), testFP, "依赖升级")
	if err != nil {
		t.Fatalf("重复 MarkStale 不应报错: %v", err)
	}
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("打开仓库: %v", err)
	}
	head, err := r.Head()
	if err != nil {
		t.Fatalf("取 HEAD: %v", err)
	}
	if ref != head.Hash().String() {
		t.Errorf("返回引用 %s 应等于当前 HEAD %s", ref, head.Hash())
	}
	hit, ok, err := s.LookupExact(context.Background(), testFP, combo)
	if err != nil || !ok {
		t.Fatalf("应仍命中(记录保留): %v %v", ok, err)
	}
	if !hit.Stale {
		t.Error("被标老化的修复命中时应 stale")
	}
}

// 验收(P0-19):并发入库经 gitwriter 的 per-repo 串行全部成功,提交一个不丢、
// 互不覆盖(此前直写 go-git worktree 无串行与基线保障)。
func TestStoreConcurrentWritesSerialize(t *testing.T) {
	repo := initRepo(t)
	s := NewStore(repo, &fakeStats{})
	combo := core.VersionCombo{"go": "1.23.0"}
	const n = 4
	refs := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// 指纹只能是十六进制(reFingerprint),用前缀编号保证互不相同。
			fp := core.Fingerprint(fmt.Sprintf("%08x00000000", i))
			refs[i], errs[i] = s.Store(context.Background(), fp, testFix(), goodProof(combo))
		}(i)
	}
	wg.Wait()
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("并发 Store #%d 失败: %v", i, errs[i])
		}
	}

	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("打开仓库: %v", err)
	}
	head, err := r.Head()
	if err != nil {
		t.Fatalf("取 HEAD: %v", err)
	}
	iter, err := r.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		t.Fatalf("取提交历史: %v", err)
	}
	inHistory := map[string]bool{}
	count := 0
	if err := iter.ForEach(func(c *object.Commit) error {
		count++
		inHistory[c.Hash.String()] = true
		return nil
	}); err != nil {
		t.Fatalf("遍历提交历史: %v", err)
	}
	if count != n {
		t.Errorf("仓库应有 %d 个提交(每个入库恰好一个,无丢失), got %d", n, count)
	}
	for i, ref := range refs {
		if !inHistory[ref] {
			t.Errorf("并发入库 #%d 的提交 %s 不在提交历史中", i, ref)
		}
		dir := filepath.Join(repo, "knowledge", fmt.Sprintf("%08x00000000", i))
		if _, err := os.Stat(filepath.Join(dir, "verified.json")); err != nil {
			t.Errorf("并发入库 #%d 的条目应落盘: %v", i, err)
		}
	}
}

// —— 查库:模糊指纹(只返回线索)——

// 验收:模糊命中只返回线索(坐标+状态),组合不一致标 stale。
func TestLookupFuzzyLeads(t *testing.T) {
	comboJSON, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	stats := &fakeStats{rows: []store.FingerprintStat{
		{FpExact: "aaaa0000aaaa0000", FpFuzzy: "fuzzy1", Status: "fixed", FixVerifiedCombo: comboJSON},
		{FpExact: "bbbb0000bbbb0000", FpFuzzy: "fuzzy1", Status: "unresolved"},
	}}
	s := NewStore(initRepo(t), stats)

	leads, err := s.LookupFuzzy(context.Background(), "fuzzy1", core.VersionCombo{"go": "1.24.0"})
	if err != nil {
		t.Fatalf("LookupFuzzy: %v", err)
	}
	if len(leads) != 2 {
		t.Fatalf("线索数 = %d", len(leads))
	}
	if leads[0].ExactFP != "aaaa0000aaaa0000" || leads[0].Status != "fixed" {
		t.Errorf("线索应带精确指纹与状态: %+v", leads[0])
	}
	if !leads[0].Stale {
		t.Error("验证组合与当前不一致的线索应标 stale")
	}
	if leads[1].Stale {
		t.Error("无验证组合的线索谈不上失效")
	}
}

func TestLookupFuzzyPropagatesError(t *testing.T) {
	s := NewStore(initRepo(t), &fakeStats{err: errors.New("库不可用")})
	if _, err := s.LookupFuzzy(context.Background(), "fuzzy1", nil); err == nil {
		t.Error("索引出错应向上返回")
	}
}

// —— 机器保证:API 形状 ——

// 验收:查模糊指纹的方法,签名(入参+返回)里不含 Fix 字段 —— 模糊只能当线索。
func TestLookupFuzzySignatureHasNoFixField(t *testing.T) {
	m, ok := reflect.TypeOf(&Store{}).MethodByName("LookupFuzzy")
	if !ok {
		t.Fatal("Store 缺少 LookupFuzzy 方法")
	}
	mt := m.Type
	for i := 0; i < mt.NumIn(); i++ {
		assertNoFixField(t, mt.In(i), "入参")
	}
	for i := 0; i < mt.NumOut(); i++ {
		assertNoFixField(t, mt.Out(i), "返回值")
	}
}

func assertNoFixField(t *testing.T, rt reflect.Type, where string) {
	t.Helper()
	switch rt.Kind() {
	case reflect.Slice, reflect.Array, reflect.Ptr:
		assertNoFixField(t, rt.Elem(), where)
	case reflect.Struct:
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if strings.EqualFold(f.Name, "fix") {
				t.Errorf("模糊指纹查询%s出现 Fix 字段 %q:模糊命中只能当线索", where, f.Name)
			}
			if f.Type.Name() == "Fix" {
				t.Errorf("模糊指纹查询%s字段 %q 的类型是 Fix:线索不得携带修复内容", where, f.Name)
			}
			assertNoFixField(t, f.Type, where)
		}
	}
}

// 验收:入库不存 LLM 分析过程 —— verified.json 的记录结构字段白名单检查。
func TestVerifiedRecordFieldsWhitelist(t *testing.T) {
	allowed := map[string]bool{
		"VerifiedAt": true, "VerifiedCombo": true, "Gates": true, "Summary": true, "Changes": true,
	}
	rt := reflect.TypeOf(verifiedRecord{})
	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Name
		if !allowed[name] {
			t.Errorf("verified.json 记录出现计划外字段 %q:知识库不存 LLM 分析过程/故障描述", name)
		}
	}
}

// comboStale 的判定细则。
func TestComboStale(t *testing.T) {
	cases := []struct {
		name     string
		verified core.VersionCombo
		current  core.VersionCombo
		want     bool
	}{
		{"完全一致", core.VersionCombo{"a": "1"}, core.VersionCombo{"a": "1"}, false},
		{"当前新增键", core.VersionCombo{"a": "1"}, core.VersionCombo{"a": "1", "b": "2"}, false},
		{"值变了", core.VersionCombo{"a": "1"}, core.VersionCombo{"a": "2"}, true},
		{"键没了", core.VersionCombo{"a": "1"}, core.VersionCombo{"b": "1"}, true},
		{"当前为空", core.VersionCombo{"a": "1"}, nil, true},
		{"未记录验证组合", core.VersionCombo{}, core.VersionCombo{"a": "1"}, true},
	}
	for _, c := range cases {
		if got := comboStale(c.verified, c.current); got != c.want {
			t.Errorf("%s: comboStale = %v, want %v", c.name, got, c.want)
		}
	}
}

// 编译期见证:knowledge 暴露的写/查签名与 failure 侧期望(KnowledgeBase)同形。
// 不在这里导入 failure —— 业务包互不导入,用镜像接口守住形状,接线在上层(web/jobs)。
type knowledgeBaseShape interface {
	LookupExact(ctx context.Context, fp core.Fingerprint, current core.VersionCombo) (core.FixHit, bool, error)
	LookupFuzzy(ctx context.Context, fp core.Fingerprint, current core.VersionCombo) ([]core.Lead, error)
	Store(ctx context.Context, fp core.Fingerprint, fix core.Fix, proof core.VerificationProof) (string, error)
}

var _ knowledgeBaseShape = (*Store)(nil)
