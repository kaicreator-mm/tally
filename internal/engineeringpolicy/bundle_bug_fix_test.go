// P3-01 内置 Bug Fix EngineeringPolicy 束(catalog/v14/policies/bug_fix.yaml)
// 与 Bug Fix Harness 冻结语料(test/fixtures/v14/bugfix)的权威测试:
//
//   - 束经**真实解析器**(engineeringpolicy.Parse)解析:gate 语义、
//     gate_condition ⊆ requires、内容修订哈希、精确值触发器(P3-01 AC1);
//   - 束经**真实装载器**(internal/artifacts.LoadPolicies)从 Git 提交树的
//     .tally/policies/ 装载,Revision 绑定 Git blob 实际内容(防篡改/防漂移);
//   - 夹具任务(真实 TaskContract + 确定性风险/变更事实)经**真实匹配器**
//     (engineeringpolicy.Match)命中:适用的代码缺陷被要求回归证据
//     (AC2),低风险缺陷不产生任何 L3 要求(AC3,配合 assurance.Select
//     的真实风险规则);
//   - 已知可修复缺陷夹具经**真实门禁语义**(EvaluateGate)证明:修复前
//     gate 不满足(未满足键确定性),修复后满足(AC4)。
//
// 本文件只消费既有生产接缝,不新增任何生产代码(允许路径:束、夹具、
// 本包测试)。辅助符号全部以 bugFix 前缀命名,避免与并行的 P3-02 测试冲突。

package engineeringpolicy_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gopkg.in/yaml.v3"

	"tally/internal/artifacts"
	"tally/internal/assurance"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/taskcontract"
)

// bugFixPolicyID 是内置束的唯一策略 id(catalog/v14/policies/bug_fix.yaml)。
const bugFixPolicyID = "POL-BUG-FIX-REGRESSION"

// bugFixWantGateKeys 是束门禁要求的两条回归证据键(升序)。
var bugFixWantGateKeys = []string{"bug_repro_failed_before", "regression_check_passes_after"}

// bugFixFactsFile 是夹具事实文件(test/fixtures/v14/bugfix/facts)的解码载体:
// 一次匹配的变更轴与风险候选原始事实。
type bugFixFactsFile struct {
	ChangedPaths []string            `yaml:"changed_paths"`
	RiskSignals  []bugFixSignalInput `yaml:"risk_signals"`
}

type bugFixSignalInput struct {
	ID    string   `yaml:"id"`
	Basis []string `yaml:"basis"`
}

// bugFixRepairCase 是已知可修复缺陷夹具里的一行回归检查:want 是修复后
// 期望,buggy_candidate_got 是修复前候选的确定性输出。
type bugFixRepairCase struct {
	Input             string `yaml:"input"`
	Want              string `yaml:"want"`
	BuggyCandidateGot string `yaml:"buggy_candidate_got"`
}

// bugFixRepairableFixture 是已知可修复缺陷夹具的解码载体(AC4)。
type bugFixRepairableFixture struct {
	Task            string `yaml:"task"`
	RequirementRef  string `yaml:"requirement_ref"`
	PolicyID        string `yaml:"policy_id"`
	Defect          string `yaml:"defect"`
	RegressionCheck struct {
		Name  string             `yaml:"name"`
		Cases []bugFixRepairCase `yaml:"cases"`
	} `yaml:"regression_check"`
	PreRepairFulfillments  []engineeringpolicy.RequirementFulfillment `yaml:"pre_repair_fulfillments"`
	PostRepairFulfillments []engineeringpolicy.RequirementFulfillment `yaml:"post_repair_fulfillments"`
}

// bugFixRepoPath 从仓库根解析路径(测试 CWD 是本包目录,向上两级即仓库根)。
func bugFixRepoPath(t *testing.T, parts ...string) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根: %v", err)
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

// bugFixMustParseBundle 用真实解析器解析内置束,失败即测试失败。
func bugFixMustParseBundle(t *testing.T) engineeringpolicy.Policy {
	t.Helper()
	data, err := os.ReadFile(bugFixRepoPath(t, "catalog", "v14", "policies", "bug_fix.yaml"))
	if err != nil {
		t.Fatalf("读内置束: %v", err)
	}
	p, err := engineeringpolicy.Parse(data)
	if err != nil {
		t.Fatalf("内置束必须通过真实解析器: %v", err)
	}
	return p
}

// bugFixLoadContract 用真实 TaskContract 解析器读夹具任务合同。
func bugFixLoadContract(t *testing.T, id string) taskcontract.TaskContract {
	t.Helper()
	c, err := taskcontract.ParseFile(bugFixRepoPath(t, "test", "fixtures", "v14", "bugfix", "contracts", id+".yaml"))
	if err != nil {
		t.Fatalf("夹具合同 %s 必须是合法 TaskContract: %v", id, err)
	}
	return c
}

// bugFixLoadFacts 读夹具事实文件。
func bugFixLoadFacts(t *testing.T, id string) bugFixFactsFile {
	t.Helper()
	data, err := os.ReadFile(bugFixRepoPath(t, "test", "fixtures", "v14", "bugfix", "facts", id+".yaml"))
	if err != nil {
		t.Fatalf("读夹具事实 %s: %v", id, err)
	}
	var f bugFixFactsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		t.Fatalf("解码夹具事实 %s: %v", id, err)
	}
	return f
}

// bugFixDerive 经真实 P1-04 派生得到风险集(输入顺序无关)。
func bugFixDerive(t *testing.T, facts bugFixFactsFile) risk.Result {
	t.Helper()
	sigs := make([]risk.Signal, 0, len(facts.RiskSignals))
	for _, s := range facts.RiskSignals {
		sigs = append(sigs, risk.Signal{ID: risk.SignalID(s.ID), Basis: s.Basis})
	}
	res, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return res
}

// bugFixReversed 返回倒序拷贝(确定性对拍用)。
func bugFixReversed(xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[len(xs)-1-i] = x
	}
	return out
}

// bugFixInstallBundle 把内置束字节安装进临时 git 仓库的
// .tally/policies/bug_fix.yaml 并提交,返回 (仓库目录, 提交 sha)。
// mutate 非 nil 时先变形内容(篡改对拍用);签名时间固定,夹具确定。
func bugFixInstallBundle(t *testing.T, mutate func([]byte) []byte) (string, string) {
	t.Helper()
	data, err := os.ReadFile(bugFixRepoPath(t, "catalog", "v14", "policies", "bug_fix.yaml"))
	if err != nil {
		t.Fatalf("读内置束: %v", err)
	}
	if mutate != nil {
		data = mutate(data)
	}
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init git 仓库: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("取工作区: %v", err)
	}
	target := filepath.Join(dir, ".tally", "policies", "bug_fix.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("建安装目录: %v", err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatalf("写安装文件: %v", err)
	}
	if _, err := wt.Add(".tally/policies/bug_fix.yaml"); err != nil {
		t.Fatalf("暂存: %v", err)
	}
	sig := &object.Signature{Name: "p3-01-fixture", Email: "p3-01@tally.local", When: time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)}
	h, err := wt.Commit("install bug_fix bundle", &git.CommitOptions{Author: sig, Committer: sig})
	if err != nil {
		t.Fatalf("提交: %v", err)
	}
	return dir, h.String()
}

// ---- AC1:trigger 与 requires 确定性 --------------------------------------

func TestBugFixBundleIsDeterministicGatePolicy(t *testing.T) {
	p := bugFixMustParseBundle(t)

	if p.ID != bugFixPolicyID {
		t.Fatalf("id = %q,期望 %q", p.ID, bugFixPolicyID)
	}
	if p.Enforcement != engineeringpolicy.EnforcementGate {
		t.Fatalf("内置束必须是 gate 策略,得到 %q", p.Enforcement)
	}
	if !strings.HasPrefix(p.Revision, "sha256:") {
		t.Fatalf("Revision 未填充内容修订哈希: %q", p.Revision)
	}

	// gate_condition ⊆ requires(门禁不得要求未事先告知的条件,PRD §8.3;
	// Validate 已强制,这里显式对账)。
	for _, k := range p.GateCondition.AllOf {
		if !slices.Contains(p.Requires, k) {
			t.Fatalf("gate_condition 键 %q 不在 requires[]", k)
		}
	}
	// GateKeys 编译为升序的两条回归证据键。
	if got := p.GateKeys(); !reflect.DeepEqual(got, bugFixWantGateKeys) {
		t.Fatalf("GateKeys = %v,期望 %v", got, bugFixWantGateKeys)
	}

	// 触发器只用精确值:匹配器是精确值相等(无 glob/正则),触发条目携带
	// 通配元字符只会造成「写了 glob、按字面匹配」的假确定性。
	wantSignals := []string{"auth_symbol_change", "protected_surface_change"}
	wantRefs := []string{".tally/requirements/REQ-2410.md", ".tally/requirements/REQ-2411.md"}
	if !reflect.DeepEqual(p.Trigger.RiskSignals, wantSignals) {
		t.Fatalf("trigger.risk_signals = %v,期望 %v(冻结枚举精确值)", p.Trigger.RiskSignals, wantSignals)
	}
	if !reflect.DeepEqual(p.Trigger.RequirementRefs, wantRefs) {
		t.Fatalf("trigger.requirement_refs = %v,期望 %v(Bug Fix Harness 冻结语料)", p.Trigger.RequirementRefs, wantRefs)
	}
	if len(p.Trigger.Paths) != 0 {
		t.Fatalf("路径轴不用于本束(精确值匹配下路径触发是假覆盖): %v", p.Trigger.Paths)
	}
	for _, entry := range append(append(append([]string{}, p.Trigger.RiskSignals...), p.Trigger.RequirementRefs...), p.Trigger.Paths...) {
		if strings.TrimSpace(entry) == "" || strings.ContainsAny(entry, "*?[\\") {
			t.Fatalf("触发条目 %q 必须是非空精确值,不得含通配元字符", entry)
		}
	}

	if !slices.Contains(p.BundleTags, "bug_fix") {
		t.Fatalf("bundle_tags 应含 bug_fix: %v", p.BundleTags)
	}

	// 同一内容重复解析得到同一 Revision(内容推导,确定性)。
	again := bugFixMustParseBundle(t)
	if again.Revision != p.Revision {
		t.Fatalf("同内容两次解析 Revision 不同:%q vs %q", p.Revision, again.Revision)
	}
}

// ---- 真实装载器:Git-first 装载与 Revision 防篡改 -------------------------

func TestBugFixBundleLoadsThroughRealPolicyLoader(t *testing.T) {
	p := bugFixMustParseBundle(t)

	repoDir, rev := bugFixInstallBundle(t, nil)
	snap, err := artifacts.LoadPolicies(repoDir, rev)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	if len(snap.Rejected) != 0 {
		t.Fatalf("内置束不得被拒收: %v", snap.Rejected)
	}
	if len(snap.Policies) != 1 {
		t.Fatalf("期望恰好装载 1 份策略,得到 %d", len(snap.Policies))
	}
	loaded := snap.Policies[0]
	if loaded.Path != ".tally/policies/bug_fix.yaml" {
		t.Fatalf("装载路径 = %q", loaded.Path)
	}
	if loaded.Policy.ID != p.ID {
		t.Fatalf("装载策略 id = %q,期望 %q", loaded.Policy.ID, p.ID)
	}
	// Revision 绑定 Git blob 实际内容(不是声明出来的)。
	if loaded.Policy.Revision != p.Revision {
		t.Fatalf("装载 Revision = %q,期望与内容推导一致 %q", loaded.Policy.Revision, p.Revision)
	}
	if got := loaded.Policy.GateKeys(); !reflect.DeepEqual(got, bugFixWantGateKeys) {
		t.Fatalf("装载后 GateKeys = %v,期望 %v", got, bugFixWantGateKeys)
	}

	// 策略实质内容被篡改(rationale 文本)⇒ Revision 必须改变(不存在
	// 声明/观测漂移通道)。注意:注释/排版不属于解析后的策略内容,Revision
	// 按设计不变(hash.go),篡改必须落在策略值上。
	tamperedDir, tamperedRev := bugFixInstallBundle(t, func(b []byte) []byte {
		return bytes.Replace(b, []byte("同一回归检查在修复后通过"), []byte("同一回归检查在修复后通过(篡改)"), 1)
	})
	tampered, err := artifacts.LoadPolicies(tamperedDir, tamperedRev)
	if err != nil {
		t.Fatalf("LoadPolicies(篡改): %v", err)
	}
	if len(tampered.Policies) != 1 {
		t.Fatalf("篡改版应照常装载,得到 %d 份策略", len(tampered.Policies))
	}
	if tampered.Policies[0].Policy.Revision == p.Revision {
		t.Fatal("内容变化必须改变内容修订哈希(Revision)")
	}
}

// ---- AC2/AC3 的匹配面:夹具任务经真实匹配器命中 ---------------------------

func TestBugFixBundleMatchesBugFixHarnessFixtures(t *testing.T) {
	policy := bugFixMustParseBundle(t)

	cases := []struct {
		name        string
		taskID      string
		wantMatch   bool
		wantRisk    []risk.SignalID
		wantRefs    []string
		wantConfide engineeringpolicy.MatchConfidence
	}{
		{
			name:        "低风险缺陷经需求轴命中",
			taskID:      "TASK-2410",
			wantMatch:   true,
			wantRisk:    nil,
			wantRefs:    []string{".tally/requirements/REQ-2410.md"},
			wantConfide: engineeringpolicy.ConfidenceFull,
		},
		{
			name:        "认证缺陷经需求与信号双轴命中",
			taskID:      "TASK-2411",
			wantMatch:   true,
			wantRisk:    []risk.SignalID{risk.SignalAuthSymbolChange},
			wantRefs:    []string{".tally/requirements/REQ-2411.md"},
			wantConfide: engineeringpolicy.ConfidenceFull,
		},
		{
			name:        "文档对照任务不命中",
			taskID:      "TASK-2412",
			wantMatch:   false,
			wantConfide: engineeringpolicy.ConfidenceNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contract := bugFixLoadContract(t, tc.taskID)
			facts := bugFixLoadFacts(t, tc.taskID)
			policies := []engineeringpolicy.Policy{policy}

			cov, err := engineeringpolicy.Match(policies, engineeringpolicy.MatchInput{
				Risk:            bugFixDerive(t, facts),
				ChangedPaths:    facts.ChangedPaths,
				RequirementRefs: contract.RequirementRefs,
			})
			if err != nil {
				t.Fatalf("Match: %v", err)
			}

			// 确定性:全部输入倒序重排后结果逐位相同(AC1/AC4)。
			again, err := engineeringpolicy.Match(policies, engineeringpolicy.MatchInput{
				Risk:            bugFixDerive(t, bugFixFactsFile{ChangedPaths: bugFixReversed(facts.ChangedPaths), RiskSignals: bugFixReversedSignals(facts.RiskSignals)}),
				ChangedPaths:    bugFixReversed(facts.ChangedPaths),
				RequirementRefs: bugFixReversed(contract.RequirementRefs),
			})
			if err != nil {
				t.Fatalf("Match(重排): %v", err)
			}
			if !reflect.DeepEqual(cov, again) {
				t.Fatalf("输入顺序不得影响匹配结果:\n首次 %+v\n重排 %+v", cov, again)
			}

			if cov.HasUnmatchedRisk() {
				t.Fatalf("夹具场景不得留下未覆盖风险信号: %v", cov.UnmatchedRiskSignals)
			}
			if cov.Confidence != tc.wantConfide {
				t.Fatalf("confidence = %q,期望 %q", cov.Confidence, tc.wantConfide)
			}

			if !tc.wantMatch {
				if len(cov.MatchedPolicies) != 0 {
					t.Fatalf("对照任务不得命中: %+v", cov.MatchedPolicies)
				}
				return
			}

			// 命中且依据逐轴可对账(AC1)。
			if len(cov.MatchedPolicies) != 1 || cov.MatchedPolicies[0].Policy.ID != bugFixPolicyID {
				t.Fatalf("期望恰好命中 %s,得到 %+v", bugFixPolicyID, cov.MatchedPolicies)
			}
			basis := cov.MatchedPolicies[0].Basis
			if !reflect.DeepEqual(basis.RiskSignals, tc.wantRisk) {
				t.Fatalf("basis.RiskSignals = %v,期望 %v", basis.RiskSignals, tc.wantRisk)
			}
			if !reflect.DeepEqual(basis.RequirementRefs, tc.wantRefs) {
				t.Fatalf("basis.RequirementRefs = %v,期望 %v", basis.RequirementRefs, tc.wantRefs)
			}
			if len(basis.Paths) != 0 {
				t.Fatalf("路径轴未声明,basis 不得记录路径命中: %v", basis.Paths)
			}

			// AC2:命中的代码缺陷被要求回归证据(GateKeys 编译自真实命中策略)。
			if got := cov.MatchedPolicies[0].Policy.GateKeys(); !reflect.DeepEqual(got, bugFixWantGateKeys) {
				t.Fatalf("命中策略 GateKeys = %v,期望 %v", got, bugFixWantGateKeys)
			}
		})
	}
}

// bugFixReversedSignals 倒序拷贝信号候选(确定性对拍用)。
func bugFixReversedSignals(in []bugFixSignalInput) []bugFixSignalInput {
	out := make([]bugFixSignalInput, len(in))
	for i, s := range in {
		out[len(in)-1-i] = s
	}
	return out
}

// ---- AC3:低风险缺陷不意外要求 L3 -----------------------------------------

func TestBugFixBundleLowRiskDefectStaysAtL1Floor(t *testing.T) {
	policy := bugFixMustParseBundle(t)

	// 束自身不得把 L3/held-out 写成默认要求:requires 与 gate 全部键都是
	// L1 可满足的确定性回归证据键。
	for _, k := range append(append([]string{}, policy.Requires...), policy.GateCondition.AllOf...) {
		low := strings.ToLower(k)
		if strings.Contains(low, "held") || strings.Contains(low, "l3") {
			t.Errorf("键 %q 疑似把 held-out/L3 写成默认要求(决策锁 21)", k)
		}
	}

	// 低风险缺陷(TASK-2410):空信号集 → RiskMinimal → 维持 L1 默认地板,
	// 决策链里没有任何上调规则。
	lowRisk := bugFixDerive(t, bugFixLoadFacts(t, "TASK-2410"))
	if lowRisk.AboveMinimal() {
		t.Fatalf("低风险夹具不得派生任何风险信号: %+v", lowRisk.All())
	}
	sel, err := assurance.Select(assurance.ClauseInput{ClauseID: "TASK-2410/AC-01", Risk: assurance.RiskMinimal})
	if err != nil {
		t.Fatalf("assurance.Select: %v", err)
	}
	if sel.Required != assurance.LevelL1 {
		t.Fatalf("低风险缺陷必需档 = %s,期望 L1(决策锁 21 默认地板)", sel.Required)
	}
	if !reflect.DeepEqual(sel.Basis.Reasons, []assurance.Reason{assurance.ReasonDefaultFloor}) {
		t.Fatalf("低风险决策链不得含上调理由: %v", sel.Basis.Reasons)
	}

	// 认证缺陷(TASK-2411):信号存在 → 风险规则上调 L3("除非 bake-off/
	// 风险规则如此判定");这是风险规则说了算,不是束写死的。
	authRisk := bugFixDerive(t, bugFixLoadFacts(t, "TASK-2411"))
	if !authRisk.AboveMinimal() {
		t.Fatal("认证缺陷夹具必须派生风险信号")
	}
	// 与 contract.Bind 的 P1 最小映射一致:无未覆盖信号且 AboveMinimal
	// ⇒ RiskElevated(未覆盖面已由匹配测试单独断言为空)。
	sel2, err := assurance.Select(assurance.ClauseInput{ClauseID: "TASK-2411/AC-01", Risk: assurance.RiskElevated})
	if err != nil {
		t.Fatalf("assurance.Select(auth): %v", err)
	}
	if sel2.Required != assurance.LevelL3 {
		t.Fatalf("认证缺陷必需档 = %s,期望 L3(风险规则上调)", sel2.Required)
	}
	if !slices.Contains(sel2.Basis.Reasons, assurance.ReasonRiskUplift) {
		t.Fatalf("上调必须携带 ReasonRiskUplift 依据: %v", sel2.Basis.Reasons)
	}
	// 即便上调 L3,束的门禁键仍是同两条回归证据键:束永不追加 held-out 键。
	if got := policy.GateKeys(); !reflect.DeepEqual(got, bugFixWantGateKeys) {
		t.Fatalf("上调前后 GateKeys 必须不变且无 held-out 键: %v", got)
	}
}

// ---- AC4:已知可修复缺陷经真实门禁语义 -------------------------------------

func TestBugFixFixtureHasKnownRepairableFailure(t *testing.T) {
	policy := bugFixMustParseBundle(t)

	data, err := os.ReadFile(bugFixRepoPath(t, "test", "fixtures", "v14", "bugfix", "repairable_failure", "TASK-2410.yaml"))
	if err != nil {
		t.Fatalf("读可修复缺陷夹具: %v", err)
	}
	var fx bugFixRepairableFixture
	if err := yaml.Unmarshal(data, &fx); err != nil {
		t.Fatalf("解码可修复缺陷夹具: %v", err)
	}
	if fx.Task != "TASK-2410" || fx.PolicyID != bugFixPolicyID {
		t.Fatalf("夹具指向 task=%s policy=%s,期望 TASK-2410 / %s", fx.Task, fx.PolicyID, bugFixPolicyID)
	}

	// 已知修复前失败必须真实存在于数据:恰好一行 buggy 输出 ≠ want。
	failed := 0
	for _, c := range fx.RegressionCheck.Cases {
		if c.BuggyCandidateGot != c.Want {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("期望夹具恰好含 1 个已知修复前失败,得到 %d", failed)
	}

	// 修复前:复现已确立(第一键满足)但回归检查未过 → gate 不满足,
	// 未满足键确定性列出 → 可修复失败进入外层 Repair(决策锁 23)。
	pre, err := policy.EvaluateGate(fx.PreRepairFulfillments)
	if err != nil {
		t.Fatalf("EvaluateGate(pre): %v", err)
	}
	if pre.Satisfied {
		t.Fatal("修复前候选不得通过门禁")
	}
	if !reflect.DeepEqual(pre.UnmetKeys, []string{"regression_check_passes_after"}) {
		t.Fatalf("修复前 UnmetKeys = %v,期望 [regression_check_passes_after]", pre.UnmetKeys)
	}

	// 修复后:同一回归检查通过 → 同一门禁满足(不放宽、不改条件)。
	post, err := policy.EvaluateGate(fx.PostRepairFulfillments)
	if err != nil {
		t.Fatalf("EvaluateGate(post): %v", err)
	}
	if !post.Satisfied || len(post.UnmetKeys) != 0 {
		t.Fatalf("修复后候选应通过同一门禁: %+v", post)
	}
}
