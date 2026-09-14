package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"tally/internal/core"
	"tally/internal/gitwriter"
	"tally/internal/store"
)

// Dir 是验证库在 git 仓库里的目录。条目按精确指纹建子目录。
const Dir = "knowledge"

// ErrInvalidProof:写入缺少有效验证证据(铁律 2 的运行时兜底)。
var ErrInvalidProof = errors.New("写入知识库需要有效验证证据")

// StatLister 是 knowledge 对指纹索引的期望(使用方定义):模糊指纹不在知识库建目录
// (只有精确指纹能关联修复),召回线索只能走索引面。*store.Queries 直接满足它。
type StatLister interface {
	ListFingerprintStatsByFuzzy(ctx context.Context, fpFuzzy string) ([]store.FingerprintStat, error)
}

// Store 是已验证修复库的读写入口。
//
// 内容存 git(knowledge/<精确指纹>/),入库就是一次 git commit;所有提交统一经
// internal/gitwriter 单一写边界(PRD v1.3 §23.1:per-repo 串行 + 跨进程锁 + 期望
// 基线校验),本包不直接操作 git worktree/index。索引面(fingerprint_stat)只用于
// 模糊召回,不存修复内容。不存 LLM 分析过程、故障描述、未验证结论(铁律 2)。
type Store struct {
	repoPath string
	stats    StatLister
}

// NewStore 构造知识库入口。repoPath 是 git 仓库根,stats 提供模糊召回索引。
func NewStore(repoPath string, stats StatLister) *Store {
	return &Store{repoPath: repoPath, stats: stats}
}

// verifiedRecord 是 verified.json 的机器记录。字段集合是定死的 ——
// 没有根因分析、没有故障描述、没有 LLM 的过程文字(架构测试与单测双重检查)。
type verifiedRecord struct {
	VerifiedAt    time.Time         `json:"verified_at"`
	VerifiedCombo core.VersionCombo `json:"verified_combo"`
	Gates         string            `json:"gates"`
	Summary       string            `json:"summary"`
	Changes       []core.FileChange `json:"changes"`
}

// Store 把一条已验证修复写进库:写 knowledge/<精确指纹>/ 三件套并 git commit,
// 返回提交哈希(即 fingerprint_stat.fix_ref 记的引用)。提交经 GitWriter 单一写
// 边界(internal/gitwriter)原子落盘、串行提交。
//
// 验证时刻与版本组合一律取自 proof —— 它是"验证过"的唯一凭据:证据无效直接拒绝,
// 未验证的东西进不来(铁律 2)。proof 的具体类型只有 failure 的四道闸产得出
// (字段不导出、构造不导出),本方法只认 core.VerificationProof 接口(业务包互不导入)。
func (s *Store) Store(ctx context.Context, fp core.Fingerprint, fix core.Fix, proof core.VerificationProof) (string, error) {
	if err := checkFingerprint(fp); err != nil {
		return "", err
	}
	if proof == nil || !proof.Valid() {
		return "", ErrInvalidProof
	}
	if strings.TrimSpace(fix.Repro) == "" {
		return "", fmt.Errorf("写入知识库 %s: 缺少复现命令", fp)
	}

	rec := verifiedRecord{
		VerifiedAt:    proof.VerifiedAt().UTC(),
		VerifiedCombo: proof.VerifiedCombo(),
		Gates:         proof.GateRecord(),
		Summary:       fix.Summary,
		Changes:       fix.Changes,
	}
	raw, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化验证记录: %w", err)
	}

	entryDir := Dir + "/" + string(fp)
	files := map[string][]byte{
		entryDir + "/fix.md":        []byte(renderFixMD(fp, fix, rec)),
		entryDir + "/repro.sh":      []byte("#!/bin/sh\n" + strings.TrimSpace(fix.Repro) + "\n"),
		entryDir + "/verified.json": raw,
	}
	msg := "fix: 已验证修复 " + string(fp) + "\n\nverified-at: " + rec.VerifiedAt.Format(time.RFC3339) +
		"\ngates: " + rec.Gates + "\n"
	commit, err := commitFiles(ctx, s.repoPath, files, msg)
	if err != nil {
		return "", fmt.Errorf("已验证修复写入 git: %w", err)
	}
	return commit, nil
}

// LookupExact 按精确指纹查库。命中时 ok=true。
//
// 命中即比对版本组合:验证时的组合与 current 对不上、或条目被标了老化 / 已退休,
// hit.Stale=true —— 调用方必须把它降级成线索,不能直接当答案(PRD 第五部分「修复会过期」)。
func (s *Store) LookupExact(_ context.Context, fp core.Fingerprint, current core.VersionCombo) (core.FixHit, bool, error) {
	if err := checkFingerprint(fp); err != nil {
		return core.FixHit{}, false, err
	}
	base := filepath.Join(s.repoPath, filepath.FromSlash(Dir+"/"+string(fp)))
	rec, err := readVerifiedRecord(base)
	if errors.Is(err, os.ErrNotExist) {
		return core.FixHit{}, false, nil
	}
	if err != nil {
		return core.FixHit{}, false, fmt.Errorf("读知识条目 %s: %w", fp, err)
	}
	repro, err := readRepro(base)
	if err != nil {
		return core.FixHit{}, false, fmt.Errorf("读复现命令 %s: %w", fp, err)
	}

	hit := core.FixHit{
		Fix:           core.Fix{Repro: repro, Summary: rec.Summary, Changes: rec.Changes},
		VerifiedAt:    rec.VerifiedAt,
		VerifiedCombo: rec.VerifiedCombo,
		Stale:         comboStale(rec.VerifiedCombo, current) || hasStaleMarker(base) || hasRetiredMarker(base),
	}
	return hit, true, nil
}

// LookupFuzzy 按模糊指纹召回线索:只给坐标(精确指纹)与状态,给不出修复内容 ——
// 修复只能由精确指纹关联,这个限制在数据结构层面(Lead 无 Fix 字段)和 SQL 层面
// (模糊指纹关联不进 fix_ref)双重生效。版本组合对不上的线索标 stale。
func (s *Store) LookupFuzzy(ctx context.Context, fp core.Fingerprint, current core.VersionCombo) ([]core.Lead, error) {
	rows, err := s.stats.ListFingerprintStatsByFuzzy(ctx, string(fp))
	if err != nil {
		return nil, fmt.Errorf("召回模糊指纹 %s: %w", fp, err)
	}
	leads := make([]core.Lead, 0, len(rows))
	for _, r := range rows {
		var combo core.VersionCombo
		if len(r.FixVerifiedCombo) > 0 {
			if err := json.Unmarshal(r.FixVerifiedCombo, &combo); err != nil {
				return nil, fmt.Errorf("解析 %s 的验证组合: %w", r.FpExact, err)
			}
		}
		leads = append(leads, core.Lead{
			ExactFP:       core.Fingerprint(r.FpExact),
			Status:        r.Status,
			VerifiedCombo: combo,
			// 没验证过修复(无组合)谈不上失效;验证过但对不上才是 stale。
			Stale: len(combo) > 0 && comboStale(combo, current),
		})
	}
	return leads, nil
}

// Entry 是一条已验证修复的只读浏览视图(界面用)。不含验证证据本身 ——
// 界面只读验证库、绝不写(INV-2);证据只在失败处理流程入库时用。
type Entry struct {
	Fingerprint   string
	Repro         string
	Summary       string
	Gates         string
	VerifiedAt    time.Time
	VerifiedCombo core.VersionCombo
	Changes       []core.FileChange
	MarkedStale   bool // 有 STALE.md:被显式标过「可能过期」
	Retired       bool // 有 RETIRED.md:验证过不再复现,修复已退休(退出使用)
}

// ListVerified 列出库里全部已验证修复条目(只读浏览),按指纹排序。
// 损坏 / 读不了的条目跳过(best-effort,不让一个坏条目挡掉整库浏览)。
func (s *Store) ListVerified(_ context.Context) ([]Entry, error) {
	root := filepath.Join(s.repoPath, filepath.FromSlash(Dir))
	dirs, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil // 还没有任何修复
	}
	if err != nil {
		return nil, fmt.Errorf("读验证库目录: %w", err)
	}
	var out []Entry
	for _, de := range dirs {
		if !de.IsDir() || !reFingerprint.MatchString(de.Name()) {
			continue
		}
		base := filepath.Join(root, de.Name())
		rec, rerr := readVerifiedRecord(base)
		if rerr != nil {
			continue // 跳过损坏条目
		}
		repro, _ := readRepro(base)
		out = append(out, Entry{
			Fingerprint:   de.Name(),
			Repro:         repro,
			Summary:       rec.Summary,
			Gates:         rec.Gates,
			VerifiedAt:    rec.VerifiedAt,
			VerifiedCombo: rec.VerifiedCombo,
			Changes:       rec.Changes,
			MarkedStale:   hasStaleMarker(base),
			Retired:       hasRetiredMarker(base),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fingerprint < out[j].Fingerprint })
	return out, nil
}

// MarkStale 把一条修复标为「可能过期」(如依赖升级后,PRD 第五部分):
// 在条目目录写 STALE.md 并 git commit。记录保留不删(事实可追溯),
// 之后命中它一律 stale,降级成线索。
func (s *Store) MarkStale(ctx context.Context, fp core.Fingerprint, reason string) (string, error) {
	if err := checkFingerprint(fp); err != nil {
		return "", err
	}
	base := filepath.Join(s.repoPath, filepath.FromSlash(Dir+"/"+string(fp)))
	if _, err := os.Stat(filepath.Join(base, "verified.json")); err != nil {
		return "", fmt.Errorf("知识条目 %s 不存在: %w", fp, err)
	}
	reason = strings.TrimSpace(reason)
	marker := "# 可能过期\n\n- 标记时间: " + time.Now().UTC().Format(time.RFC3339) + "\n- 原因: " + reason + "\n"
	commit, err := commitFiles(ctx, s.repoPath, map[string][]byte{
		Dir + "/" + string(fp) + "/STALE.md": []byte(marker),
	}, "stale: 标记修复可能过期 "+string(fp)+"\n\nreason: "+reason+"\n")
	if err != nil {
		return "", fmt.Errorf("写老化标记: %w", err)
	}
	return commit, nil
}

// Retire 把一条修复标为「已退休」(PRD 第五部分「修复会过期」):依赖升级后重跑其
// 验证过的复现命令不再复现,修复退出使用。在条目目录写 RETIRED.md 并 git commit,
// 记录保留不删。退休 ≠ STALE(STALE 是组合对不上、降级成线索);退休是「验证过
// 不再复现」。退休后命中它一律按 stale 处理(见 LookupExact)。
func (s *Store) Retire(ctx context.Context, fp core.Fingerprint, reason string) (string, error) {
	if err := checkFingerprint(fp); err != nil {
		return "", err
	}
	base := filepath.Join(s.repoPath, filepath.FromSlash(Dir+"/"+string(fp)))
	if _, err := os.Stat(filepath.Join(base, "verified.json")); err != nil {
		return "", fmt.Errorf("知识条目 %s 不存在: %w", fp, err)
	}
	reason = strings.TrimSpace(reason)
	marker := "# 已退休\n\n- 退休时间: " + time.Now().UTC().Format(time.RFC3339) + "\n- 原因: " + reason + "\n"
	commit, err := commitFiles(ctx, s.repoPath, map[string][]byte{
		Dir + "/" + string(fp) + "/RETIRED.md": []byte(marker),
	}, "retire: 修复退休 "+string(fp)+"\n\nreason: "+reason+"\n")
	if err != nil {
		return "", fmt.Errorf("写退休标记: %w", err)
	}
	return commit, nil
}

// comboStale 判断修复是否可能过期:验证时记录的版本组合与当前对不上即 stale。
// 比对规则:验证时存在的每个键,当前组合里必须也在且值相等;当前新增的键
// 不作废旧修复(新工具纳入版本记录不应让全部历史修复瞬间失效)。
// 验证组合为空说明无法证明仍有效 —— 保守标 stale。
func comboStale(verified, current core.VersionCombo) bool {
	if len(verified) == 0 {
		return true
	}
	for k, v := range verified {
		if cv, ok := current[k]; !ok || cv != v {
			return true
		}
	}
	return false
}

var reFingerprint = regexp.MustCompile(`^[0-9a-f]{8,64}$`)

// checkFingerprint 把条目目录名限制在十六进制指纹内:目录名直接取自指纹,
// 必须防任意路径借此溜进仓库(路径注入)。
func checkFingerprint(fp core.Fingerprint) error {
	if !reFingerprint.MatchString(string(fp)) {
		return fmt.Errorf("非法指纹: %q", string(fp))
	}
	return nil
}

// renderFixMD 把已验证内容渲染成人读的 fix.md。与 verified.json 同源:
// 一个给人看,一个给机器读,两者都不含 LLM 的分析过程。
func renderFixMD(fp core.Fingerprint, fix core.Fix, rec verifiedRecord) string {
	var b strings.Builder
	b.WriteString("# 已验证修复\n\n")
	fmt.Fprintf(&b, "- 精确指纹: %s\n", fp)
	fmt.Fprintf(&b, "- 验证时间: %s\n", rec.VerifiedAt.Format(time.RFC3339))
	combos := make([]string, 0, len(rec.VerifiedCombo))
	for k, v := range rec.VerifiedCombo {
		combos = append(combos, k+"="+v)
	}
	sort.Strings(combos)
	fmt.Fprintf(&b, "- 版本组合: %s\n", strings.Join(combos, ", "))
	b.WriteString("\n## 做了什么改动、为什么\n\n")
	b.WriteString(strings.TrimSpace(fix.Summary) + "\n")
	b.WriteString("\n## 改动\n")
	for _, c := range fix.Changes {
		fmt.Fprintf(&b, "\n### %s\n\n```\n%s\n```\n", c.Path, c.Content)
	}
	return b.String()
}

func readVerifiedRecord(base string) (verifiedRecord, error) {
	b, err := os.ReadFile(filepath.Join(base, "verified.json"))
	if err != nil {
		return verifiedRecord{}, err
	}
	var rec verifiedRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return verifiedRecord{}, fmt.Errorf("解析 verified.json: %w", err)
	}
	return rec, nil
}

func readRepro(base string) (string, error) {
	b, err := os.ReadFile(filepath.Join(base, "repro.sh"))
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if rest, ok := strings.CutPrefix(s, "#!"); ok {
		if _, body, ok := strings.Cut(rest, "\n"); ok {
			s = body
		}
	}
	return strings.TrimSpace(s), nil
}

func hasStaleMarker(base string) bool {
	_, err := os.Stat(filepath.Join(base, "STALE.md"))
	return err == nil
}

func hasRetiredMarker(base string) bool {
	_, err := os.Stat(filepath.Join(base, "RETIRED.md"))
	return err == nil
}

// knowledgeCommitMu 串行化本进程内的知识库 Git 写:读基线(Head)与提交(Commit)
// 必须成对不被本进程其他写入插队。gitwriter 的 per-repo 锁是包内私有,外部无法
// 跨 Head+Commit 持有;跨进程互斥仍由 gitwriter 的跨进程锁 + 下方有限基线重试兜底。
var knowledgeCommitMu sync.Mutex

// commitFiles 经 GitWriter 单一写边界(internal/gitwriter,PRD v1.3 §23.1)在
// repoPath 仓库写入若干文件并创建一个提交,返回提交哈希。入库 = git commit:知识库
// 内容在 git 里,不进数据库表(docs/03-data-model.md)。落盘、暂存、提交与失败回滚
// 全部由 gitwriter 完成,本包不再直接操作 worktree/index。
//
// 基线语义:每次先读当前 HEAD 作为期望基线,gitwriter 在仓库锁内校验基线后才提交。
// 知识内容只由指纹/证据/修复决定、与基线无关,因此基线被并发写入抢先移动时按
// gitwriter 的恢复指引换新基线重试(等效 no-op rebase),有限次后仍冲突才向上报错。
func commitFiles(ctx context.Context, repoPath string, files map[string][]byte, message string) (string, error) {
	w, err := gitwriter.New(repoPath)
	if err != nil {
		return "", fmt.Errorf("定位知识库仓库: %w", err)
	}

	// 确定性文件顺序:同一内容集合总是构成同一提交。
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	updates := make([]gitwriter.FileUpdate, 0, len(names))
	for _, name := range names {
		updates = append(updates, gitwriter.FileUpdate{Path: name, Content: files[name]})
	}

	// 重试只吸收并发基线冲突窗口(跨进程),不吞其他错误。
	const baseAttempts = 3
	knowledgeCommitMu.Lock()
	defer knowledgeCommitMu.Unlock()
	for attempt := 1; ; attempt++ {
		base, err := w.Head(ctx)
		if err != nil {
			return "", fmt.Errorf("读取知识库仓库状态: %w", err)
		}
		res, err := w.Commit(ctx, gitwriter.CommitRequest{
			ExpectedBase: base,
			Message:      message,
			Files:        updates,
			Signature:    &gitwriter.Signature{Name: commitName, Email: commitEmail},
		})
		switch {
		case err == nil:
			return res.Commit, nil
		case errors.Is(err, gitwriter.ErrNoChanges):
			// gitwriter 不产生空提交:内容与 HEAD 完全一致(重复验证/重复标老化)
			// 时不再新建提交,返回当前 HEAD —— 内容已在提交历史里,调用方拿到的
			// 仍是有效的提交引用(fingerprint_stat.fix_ref 语义不变)。
			head, herr := w.Head(ctx)
			if herr != nil {
				return "", fmt.Errorf("读取知识库仓库状态: %w", herr)
			}
			return head, nil
		case errors.Is(err, gitwriter.ErrBaseRevision) && attempt < baseAttempts:
			continue
		default:
			return "", err
		}
	}
}

const (
	commitName  = "tally"
	commitEmail = "tally@localhost"
)
