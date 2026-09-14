// database_migration_bundle_test.go — P3-02:内置数据库迁移 EngineeringPolicy
// 束(catalog/v14/policies/database_migration.yaml)与迁移 harness 夹具
// (test/fixtures/v14/migration/**)经**真实生产接缝**的验收测试:
//
//	parse(P1-02 engineeringpolicy.Parse)
//	→ load(P1-03 artifacts.LoadPolicies,Git 快照装载)
//	→ match(P1-05 engineeringpolicy.Match + P1-04 risk.Derive)
//	→ GateKeys / EvaluateGate(P1-02 门禁语义)
//	→ assurance.Select(A0-03/P1 selector,决策锁 #21)。
//
// 本文件只消费既有生产接缝,不改动任何生产代码。迁移 SQL 夹具在此不被执行
// (无数据库依赖):harness 的执行面归其运行时任务;本文件钉住的是策略束与
// 夹具的确定性契约(PRD §24.2:fresh DB / N-1 upgrade / data integrity /
// repairable defect / verdict 门禁)。
package engineeringpolicy_test

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/artifacts"
	"tally/internal/assurance"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
)

// fixtureTime 与 internal/artifacts 的装载夹具同一纪律:固定签名时间,
// 提交过程零随机、零时钟依赖。
var fixtureTime = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

// migRepoRoot 定位被测仓库根(测试运行目录 = internal/engineeringpolicy)。
func migRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(root, "catalog", "v14", "policies")); err != nil || !fi.IsDir() {
		t.Fatalf("仓库根 %s 下找不到 catalog/v14/policies: %v", root, err)
	}
	return root
}

// migLoadBundle 从允许路径读出内置迁移策略束,经真实 P1-02 解析器解析,
// 并断言其确定性内容契约(PRD §24.2 的三类证据键、schema_migration 触发、
// 无路径轴)。
func migLoadBundle(t *testing.T) engineeringpolicy.Policy {
	t.Helper()
	path := filepath.Join(migRepoRoot(t), "catalog", "v14", "policies", "database_migration.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读策略束 %s: %v", path, err)
	}
	p, err := engineeringpolicy.Parse(data)
	if err != nil {
		t.Fatalf("内置迁移策略束必须通过 P1-02 解析/校验: %v", err)
	}

	if p.ID != "database_migration" {
		t.Fatalf("策略 id = %q", p.ID)
	}
	if p.Enforcement != engineeringpolicy.EnforcementGate {
		t.Fatalf("内置迁移策略必须是 gate(enforcement=%q)", p.Enforcement)
	}
	// AC1:检测迁移任务/风险输入,不用路径轴 —— 路径触碰不产生假置信。
	if !slices.Equal(p.Trigger.RiskSignals, []string{"schema_migration"}) {
		t.Fatalf("trigger.risk_signals = %v,只应声明冻结信号 schema_migration", p.Trigger.RiskSignals)
	}
	if len(p.Trigger.Paths) != 0 {
		t.Fatalf("trigger.paths = %v:路径轴会制造 path-only 假置信,不得声明", p.Trigger.Paths)
	}
	if p.Revision == "" || !strings.HasPrefix(p.Revision, "sha256:") {
		t.Fatalf("Parse 必须填充内容修订哈希,得到 %q", p.Revision)
	}
	// 解析确定性:同内容重复解析得到同一修订。
	again, err := engineeringpolicy.Parse(data)
	if err != nil || again.Revision != p.Revision {
		t.Fatalf("同内容重复解析修订不一致: %q vs %q (err=%v)", again.Revision, p.Revision, err)
	}
	return p
}

// migCommitBundle 把策略束内容提交进临时 git 仓库的生产装载目录
// (.tally/policies),返回仓库目录 —— 用真实 P1-03 Git-first 装载器装载。
func migCommitBundle(t *testing.T, yamlBytes []byte) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init git 仓库: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("取工作区: %v", err)
	}
	const inRepoPath = ".tally/policies/database_migration.yaml"
	if err := os.MkdirAll(filepath.Join(dir, ".tally", "policies"), 0o755); err != nil {
		t.Fatalf("建 .tally/policies: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".tally", "policies", "database_migration.yaml"), yamlBytes, 0o644); err != nil {
		t.Fatalf("写策略束: %v", err)
	}
	if _, err := wt.Add(inRepoPath); err != nil {
		t.Fatalf("暂存 %s: %v", inRepoPath, err)
	}
	sig := &object.Signature{Name: "p3-02-test", Email: "p3-02@test", When: fixtureTime}
	if _, err := wt.Commit("p3-02 migration bundle fixture", &git.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("提交: %v", err)
	}
	return dir
}

// TestMigrationBundleLoadsThroughGitLoader:parse → load 全走真实接缝。
// 装载器从 Git 提交树(不是工作区)读 .tally/policies/*.yaml,同一内容必然
// 得到同一修订哈希 —— 内容被换,修订即变(PRD §30 desired/observed 纪律)。
func TestMigrationBundleLoadsThroughGitLoader(t *testing.T) {
	p := migLoadBundle(t)
	dir := migCommitBundle(t, mustReadBundleBytes(t))

	snap, err := artifacts.LoadPolicies(dir, "HEAD")
	if err != nil {
		t.Fatalf("artifacts.LoadPolicies: %v", err)
	}
	if len(snap.Rejected) != 0 {
		t.Fatalf("合法内置束不得被拒收: %+v", snap.Rejected)
	}
	if len(snap.Policies) != 1 {
		t.Fatalf("装载策略数 = %d,应为 1", len(snap.Policies))
	}
	loaded := snap.Policies[0]
	if loaded.Path != ".tally/policies/database_migration.yaml" {
		t.Fatalf("装载路径 = %q", loaded.Path)
	}
	if !reflect.DeepEqual(loaded.Policy, p) {
		t.Fatalf("Git 装载结果与解析结果不一致:\n loaded=%+v\n parsed =%+v", loaded.Policy, p)
	}
	if loaded.Policy.Revision != p.Revision {
		t.Fatalf("装载修订 %q 与解析修订 %q 不同:同内容必须同修订", loaded.Policy.Revision, p.Revision)
	}
}

func mustReadBundleBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(migRepoRoot(t), "catalog", "v14", "policies", "database_migration.yaml"))
	if err != nil {
		t.Fatalf("读策略束: %v", err)
	}
	return data
}

// migMigrationRisk 从夹具的 basis 候选(test/fixtures/v14/migration/risk/)
// 经真实 P1-04 派生 schema_migration 信号集。
func migMigrationRisk(t *testing.T) risk.Result {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(migRepoRoot(t), "test", "fixtures", "v14", "migration", "risk", "schema_migration.basis.txt"))
	if err != nil {
		t.Fatalf("读风险夹具: %v", err)
	}
	var refs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		refs = append(refs, line)
	}
	if len(refs) == 0 {
		t.Fatal("风险夹具必须至少给一个 basis ref")
	}
	res, err := risk.Derive([]risk.Signal{{ID: risk.SignalSchemaMigration, Basis: refs}})
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return res
}

// TestMigrationPolicyMatchesRiskFactsNotPaths(AC1):策略由派生风险事实命中,
// 路径-only 输入永不命中;unknown 信号保持 fail-safe 可见。
func TestMigrationPolicyMatchesRiskFactsNotPaths(t *testing.T) {
	p := migLoadBundle(t)

	// 迁移任务/变更事实 → schema_migration → 命中,basis 记录风险轴。
	cov, err := engineeringpolicy.Match([]engineeringpolicy.Policy{p},
		engineeringpolicy.MatchInput{Risk: migMigrationRisk(t)})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(cov.MatchedPolicies) != 1 || cov.MatchedPolicies[0].Policy.ID != p.ID {
		t.Fatalf("schema_migration 事实必须命中迁移策略: %+v", cov.MatchedPolicies)
	}
	if basis := cov.MatchedPolicies[0].Basis; !slices.Equal(basis.RiskSignals, []risk.SignalID{"schema_migration"}) {
		t.Fatalf("命中 basis 风险轴 = %v", basis.RiskSignals)
	}
	if cov.Confidence != engineeringpolicy.ConfidenceFull || cov.HasUnmatchedRisk() {
		t.Fatalf("仅含已声明信号时 confidence=full 且无未覆盖: %s unmatched=%v", cov.Confidence, cov.UnmatchedRiskSignals)
	}

	// path-only 假置信防线:变更触碰迁移链文件本身不触发策略。
	pathOnly, err := engineeringpolicy.Match([]engineeringpolicy.Policy{p},
		engineeringpolicy.MatchInput{
			ChangedPaths: []string{
				"migrations/0003_orders_total_not_null.sql",
				"test/fixtures/v14/migration/migrations/0003_orders_total_not_null.sql",
			},
		})
	if err != nil {
		t.Fatalf("Match(path-only): %v", err)
	}
	if len(pathOnly.MatchedPolicies) != 0 || pathOnly.Confidence != engineeringpolicy.ConfidenceNone {
		t.Fatalf("路径-only 输入不得命中迁移策略(假置信): %+v conf=%s", pathOnly.MatchedPolicies, pathOnly.Confidence)
	}

	// unknown 信号(INV-V14-09):命中迁移策略的同时,未被任何命中策略声明
	// 的 unknown 信号必须原样留在未覆盖清单,绝不静默当作已覆盖。
	mixed, err := risk.Derive([]risk.Signal{
		{ID: risk.SignalSchemaMigration, Basis: []string{"task:TASK-MIG-001"}},
		{ID: risk.SignalID("crypto_key_rotation"), Basis: []string{"task:TASK-MIG-001"}},
	})
	if err != nil {
		t.Fatalf("risk.Derive(unknown): %v", err)
	}
	covUnk, err := engineeringpolicy.Match([]engineeringpolicy.Policy{p},
		engineeringpolicy.MatchInput{Risk: mixed})
	if err != nil {
		t.Fatalf("Match(unknown): %v", err)
	}
	if len(covUnk.MatchedPolicies) != 1 {
		t.Fatalf("schema_migration 事实仍须命中: %+v", covUnk.MatchedPolicies)
	}
	if !slices.Contains(covUnk.UnknownRiskSignals, "crypto_key_rotation") {
		t.Fatalf("unknown 信号必须保真露出: %v", covUnk.UnknownRiskSignals)
	}
	if !covUnk.HasUnmatchedRisk() || !slices.Contains(covUnk.UnmatchedRiskSignals, "crypto_key_rotation") {
		t.Fatalf("未被声明的 unknown 信号必须进入未覆盖清单: %v", covUnk.UnmatchedRiskSignals)
	}
	if covUnk.Confidence != engineeringpolicy.ConfidencePartial {
		t.Fatalf("存在未覆盖信号时 confidence=partial,得到 %s", covUnk.Confidence)
	}
}

// migReadEvidence 读一组证据夹具(evidence/<set>/<key>.txt,内容
// satisfied | unsatisfied)并编译成门禁 fulfillments。夹具键集必须与策略
// requires[] 完全一致 —— AC2 的映射是结构性的,不是巧合。
func migReadEvidence(t *testing.T, p engineeringpolicy.Policy, set string) []engineeringpolicy.RequirementFulfillment {
	t.Helper()
	dir := filepath.Join(migRepoRoot(t), "test", "fixtures", "v14", "migration", "evidence", set)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读证据集 %s: %v", set, err)
	}
	var out []engineeringpolicy.RequirementFulfillment
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		key := strings.TrimSuffix(e.Name(), ".txt")
		if seen[key] {
			t.Fatalf("证据集 %s 键 %q 重复", set, key)
		}
		seen[key] = true
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读证据 %s/%s: %v", set, e.Name(), err)
		}
		fact := strings.TrimSpace(string(data))
		switch fact {
		case "satisfied":
			out = append(out, engineeringpolicy.RequirementFulfillment{Key: key, Satisfied: true})
		case "unsatisfied":
			out = append(out, engineeringpolicy.RequirementFulfillment{Key: key, Satisfied: false})
		default:
			t.Fatalf("证据 %s/%s 内容 %q 非法(只允许 satisfied|unsatisfied)", set, e.Name(), fact)
		}
	}
	slices.SortFunc(out, func(a, b engineeringpolicy.RequirementFulfillment) int {
		return strings.Compare(a.Key, b.Key)
	})
	gotKeys := make([]string, 0, len(out))
	for _, f := range out {
		gotKeys = append(gotKeys, f.Key)
	}
	wantKeys := slices.Clone(p.Requires)
	slices.Sort(wantKeys)
	if !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("证据集 %s 键 %v 与策略 requires %v 不一致(AC2 映射必须一一对应)", set, gotKeys, wantKeys)
	}
	return out
}

// TestMigrationGateRequiresFreshDBUpgradeAndIntegrity(AC2/AC3):
// fresh DB / N-1 升级 / 数据完整性三类 required evidence 与 §24.2 闭环一一对应;
// 修复前(只有 fresh DB 证据)门禁必须拦下 —— fresh DB 单独不放行;
// 修复后三类证据齐备,门禁放行。
func TestMigrationGateRequiresFreshDBUpgradeAndIntegrity(t *testing.T) {
	p := migLoadBundle(t)

	wantKeys := []string{"data_integrity", "fresh_database", "n_minus_1_upgrade"}
	gotKeys := p.GateKeys()
	slices.Sort(gotKeys)
	if !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("GateKeys = %v,应为 §24.2 三类证据键 %v", gotKeys, wantKeys)
	}

	pre, err := p.EvaluateGate(migReadEvidence(t, p, "pre_repair"))
	if err != nil {
		t.Fatalf("EvaluateGate(pre_repair): %v", err)
	}
	if pre.Satisfied {
		t.Fatal("修复前(仅 fresh DB 证据)门禁不得放行:fresh DB 单独是假置信")
	}
	if !slices.Equal(pre.UnmetKeys, []string{"data_integrity", "n_minus_1_upgrade"}) {
		t.Fatalf("修复前未满足键 = %v", pre.UnmetKeys)
	}

	post, err := p.EvaluateGate(migReadEvidence(t, p, "post_repair"))
	if err != nil {
		t.Fatalf("EvaluateGate(post_repair): %v", err)
	}
	if !post.Satisfied || len(post.UnmetKeys) != 0 {
		t.Fatalf("修复后三类证据齐备必须放行: satisfied=%v unmet=%v", post.Satisfied, post.UnmetKeys)
	}
}

// TestMigrationFixtureCarriesRepairableDefect(AC3):夹具包含一个确定性
// 可修复的迁移缺陷(N-1 路径失败、fresh DB 路径假绿),且修复以**新候选
// ChangeSet 内容**并列存放 —— 缺陷文件不被改写(决策锁 8/23),同一修订号位
// 表达 supersession。本测试只钉夹具契约;SQL 的真实执行归 harness 运行时任务。
func TestMigrationFixtureCarriesRepairableDefect(t *testing.T) {
	fx := filepath.Join(migRepoRoot(t), "test", "fixtures", "v14", "migration")

	entries, err := os.ReadDir(filepath.Join(fx, "migrations"))
	if err != nil {
		t.Fatalf("读迁移链目录: %v", err)
	}
	var chain []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			chain = append(chain, e.Name())
		}
	}
	if !slices.Equal(chain, []string{"0001_init.sql", "0002_add_orders.sql", "0003_orders_total_not_null.sql"}) {
		t.Fatalf("迁移链 = %v,应为有序 0001..0003(夹具确定性)", chain)
	}

	read := func(rel ...string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(append([]string{fx}, rel...)...))
		if err != nil {
			t.Fatalf("读夹具 %v: %v", rel, err)
		}
		return string(data)
	}

	defect := read("migrations", "0003_orders_total_not_null.sql")
	repair := read("repair", "0003_orders_total_not_null.sql")
	seed := read("seed", "n_minus_1.sql")

	// 缺陷签名:NOT NULL 且无 DEFAULT —— fresh DB(空表)成功,N-1(有数据)
	// 被 PostgreSQL 确定性拒绝(column contains null values)。
	if !strings.Contains(defect, "ADD COLUMN total_cents BIGINT NOT NULL") {
		t.Fatalf("缺陷迁移缺少 NOT NULL 添加语句:\n%s", defect)
	}
	if strings.Contains(defect, "NOT NULL DEFAULT") {
		t.Fatalf("缺陷迁移不得自带 DEFAULT(否则两条路径都成功,失去区分度):\n%s", defect)
	}
	// N-1 路径在 0002 之后注入真实行,0003 升级的是带数据的库。
	if !strings.Contains(seed, "INSERT INTO orders") {
		t.Fatal("N-1 seed 必须向 orders 注入行,否则 0003 的缺陷不可达")
	}
	// 修复是独立的新候选内容:同一修订号位、append-only supersession、
	// 缺陷文件原样保留。
	if defect == repair {
		t.Fatal("修复候选不得是缺陷文件的复制(必须是修正后的新 ChangeSet 内容)")
	}
	if !strings.Contains(repair, "ADD COLUMN total_cents BIGINT NOT NULL DEFAULT 0") {
		t.Fatalf("修复候选必须用 NOT NULL DEFAULT 同时满足两条路径:\n%s", repair)
	}
	if !strings.Contains(defect, "repair/0003_orders_total_not_null.sql") {
		t.Fatal("缺陷文件必须注明修复以新候选内容(repair/)存放,而非原地改写(锁 8/23)")
	}
}

// TestMigrationAssuranceFollowsA003Selector(AC4):迁移条款的必需保证档
// 不在本任务里硬编码,而是走 A0-03 冻结的 P1 selector(assurance.Select);
// 这里按 contract.Bind 的同一最小映射把匹配覆盖+风险事实翻译成风险档后
// 过选择器 —— 迁移永远高于最小风险 → L3 上调;存在未覆盖/unknown 信号时
// fail-safe L3;非迁移(空信号集)维持默认地板 L1,不是 universal held-out。
func TestMigrationAssuranceFollowsA003Selector(t *testing.T) {
	// 与 contract.Bind 相同的 P1 最小风险档翻译(未导出,故在此镜像两行)。
	band := func(cov engineeringpolicy.PolicyMatchCoverage, r risk.Result) assurance.RiskBand {
		switch {
		case cov.HasUnmatchedRisk():
			return assurance.RiskBand("unmatched_risk_signals_present")
		case r.AboveMinimal():
			return assurance.RiskElevated
		default:
			return assurance.RiskMinimal
		}
	}
	p := migLoadBundle(t)

	// 迁移条款(schema_migration 已派生、无未覆盖)→ elevated → L3。
	cov, err := engineeringpolicy.Match([]engineeringpolicy.Policy{p},
		engineeringpolicy.MatchInput{Risk: migMigrationRisk(t)})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	sel, err := assurance.Select(assurance.ClauseInput{ClauseID: "AC-MIG-001", Risk: band(cov, migMigrationRisk(t))})
	if err != nil {
		t.Fatalf("assurance.Select(迁移条款): %v", err)
	}
	if sel.Required != assurance.LevelL3 {
		t.Fatalf("迁移条款必需档 = %s,应为 L3(锁 #21 风险上调)", sel.Required)
	}
	if !slices.Contains(sel.Basis.Reasons, assurance.ReasonRiskUplift) {
		t.Fatalf("决策链缺少 risk_uplift 依据: %v", sel.Basis.Reasons)
	}

	// 存在未覆盖/unknown 信号 → fail-safe 上调 L3,绝不落 L0/L1。经真实
	// Match 覆盖投影派生含 unknown 信号的风险事实,再走同一 band 翻译。
	mixed, err := risk.Derive([]risk.Signal{
		{ID: risk.SignalSchemaMigration, Basis: []string{"task:TASK-MIG-001"}},
		{ID: risk.SignalID("crypto_key_rotation"), Basis: []string{"task:TASK-MIG-001"}},
	})
	if err != nil {
		t.Fatalf("risk.Derive(unknown): %v", err)
	}
	covUnk, err := engineeringpolicy.Match([]engineeringpolicy.Policy{p},
		engineeringpolicy.MatchInput{Risk: mixed})
	if err != nil {
		t.Fatalf("Match(unknown): %v", err)
	}
	if !covUnk.HasUnmatchedRisk() {
		t.Fatalf("unknown 信号未被命中策略声明,必须未覆盖: %v", covUnk.UnmatchedRiskSignals)
	}
	selFail, err := assurance.Select(assurance.ClauseInput{ClauseID: "AC-MIG-001",
		Risk: band(covUnk, mixed)})
	if err != nil {
		t.Fatalf("assurance.Select(未覆盖风险): %v", err)
	}
	if selFail.Required != assurance.LevelL3 || !slices.Contains(selFail.Basis.Reasons, assurance.ReasonUnmatchedRiskFailsafe) {
		t.Fatalf("未覆盖风险必须 fail-safe L3: required=%s reasons=%v", selFail.Required, selFail.Basis.Reasons)
	}

	// 非迁移任务(空信号集)→ 维持默认地板 L1(Lock #21:held-out 按风险
	// 上调,不是 universal 默认)。
	selMin, err := assurance.Select(assurance.ClauseInput{ClauseID: "AC-MIG-001", Risk: band(engineeringpolicy.PolicyMatchCoverage{}, risk.Result{})})
	if err != nil {
		t.Fatalf("assurance.Select(最小风险): %v", err)
	}
	if selMin.Required != assurance.LevelL1 {
		t.Fatalf("最小风险条款必需档 = %s,应为默认地板 L1", selMin.Required)
	}

	// 三种形态都不产生实验档 L2(锁 #21)。
	for _, s := range []assurance.Selection{sel, selFail, selMin} {
		if s.Required == assurance.LevelL2 {
			t.Fatalf("L2 是实验档,不得被选择: %+v", s)
		}
	}
}
