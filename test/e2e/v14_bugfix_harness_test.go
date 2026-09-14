//go:build integration

// P3-03 — Bug Fix 闭环 E2E 的夹具与测试等价组合(harness)。
//
// 被测纵切与 P2-10 矩阵(v14_closedloop_harness_test.go)完全同型:生产闭环
// 组合根 closedloop.NewLoop 驱动 FakeRuntime 的确定性脚本,事实全部落进真实
// PostgreSQL。差异只在场景语料:本文件消费 P3-01 的 Bug Fix Harness 冻结语料
// (test/fixtures/v14/bugfix:TASK-2410 合同、确定性事实、已知可修复缺陷)与
// 内置束 catalog/v14/policies/bug_fix.yaml,把 PRD §24.1 的缺陷闭环跑成真实
// 执行:
//
//	baseline(含 off-by-one 缺陷)→ 冻结回归协议在修复前确定性失败(复现)
//	→ 束 Git-first 装载 + 真实匹配器命中 + GateKeys 编译
//	→ Agent 候选(一次性 worktree)→ 回归验证 FAIL(可修复)
//	→ 外层 Repair:新 Run + 新一次性 worktree + 修复基物化 + 安全反馈
//	  + reason=rework 取代事实
//	→ 复验 PASS → 同一门禁满足 → accepted(带 basis)
//
// 与 P2-10 harness 的复用关系(同一 package e2e):
//   - clCodegenRunner / clInstruction / clFixCandidateFromWorktree 原样复用
//     (任务无关,其行为本体由 P2-09/P2-10 权威覆盖);
//   - clStore / clGit / clWriteFile / clAssertOwnerClean / clAssertRootEmpty /
//     clRejections 原样复用;
//   - 脚本化 runtime 与组合根按本场景语料重写(bf 前缀),差异如实声明:
//     runtime 是 FakeRuntime 的确定性脚本包装,每个 Execute 轮次向会话的
//     一次性 worktree 写入令牌桶候选内容。
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/closedloop"
	"tally/internal/contextpack"
	"tally/internal/contract"
	"tally/internal/core"
	"tally/internal/engineeringpolicy"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/risk"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

// ---- 常量与夹具标识 ----------------------------------------------------------

const (
	// bfProject / bfTaskID:P3-01 Bug Fix Harness 冻结语料的场景 1
	// (PRD §24.1 低风险功能缺陷:令牌桶边界 off-by-one)。
	bfProject = core.ProjectID("proj-p2410")
	bfTaskID  = "TASK-2410"

	// bfLimiterRel 是候选交付物(脚本写入、冻结命令 grep 的文件);
	// bfLimiterTestRel 是回归检查证据文件(base 已提交)。
	bfLimiterRel     = "internal/limiter/token_bucket.go"
	bfLimiterTestRel = "internal/limiter/token_bucket_test.go"

	// bfBehaviorBinding / bfRegressionBinding 是两个 visible 冻结绑定:
	// AC-01 行为验收与 AC-02 回归检查(fails-before / passes-after)。
	bfBehaviorBinding   = "BEH-1"
	bfRegressionBinding = "REG-1"

	// bfFixedToken 只存在于修复后候选:回归检查(冻结命令)对修复前代码
	// 确定性 exit 1(缺陷在,回归检查失败),对修复后候选 exit 0。
	bfFixedToken = "P2410-BOUNDARY-FIX"

	// bfBugFixPolicyID 是内置束的唯一策略 id。
	bfBugFixPolicyID = "POL-BUG-FIX-REGRESSION"
)

// bfBaselineContent 是修复前基线(缺陷在基线里):`used < capacity+1` 的边界
// 反转让桶被取空后仍放行第 capacity+1 个即时请求 —— 与可修复缺陷夹具的
// 已知失败用例(第 4 次请求 want=reject / buggy got=allow)同一缺陷形态。
func bfBaselineContent() string {
	return `package limiter

// Allow reports whether an immediate request is admitted by the token bucket.
// TASK-2410 defect (pre-fix baseline): the boundary comparison is inverted,
// so after the bucket is emptied one extra immediate request is admitted.
func Allow(used, capacity int) bool {
	return used < capacity+1 // BUG: admits the (capacity+1)-th request
}
`
}

// bfWrongFix* 是修复尝试的确定性脚本内容:形似修复但仍承认超容量请求
// (互不相同的候选内容 ⇒ 互不相同的 ChangeSet 身份),全部不含回归检查
// token ⇒ 冻结回归协议对每个候选都确定性失败。
func bfWrongFix1() string {
	return `package limiter

// Allow reports whether an immediate request is admitted by the token bucket.
// Wrong fix attempt: the comparison is reordered but the off-by-one stays,
// so the fourth immediate request is still admitted after the bucket empties.
func Allow(used, capacity int) bool {
	return used <= capacity // BUG: still admits the (capacity+1)-th request
}
`
}

func bfWrongFix2() string {
	return `package limiter

// Allow reports whether an immediate request is admitted by the token bucket.
// Wrong fix attempt (rework 1): the algebraic rewrite still admits the request
// beyond capacity, because capacity-used is zero exactly at the boundary.
func Allow(used, capacity int) bool {
	return capacity-used >= 0 // BUG: admits the (capacity+1)-th request
}
`
}

func bfWrongFix3() string {
	return `package limiter

// Allow reports whether an immediate request is admitted by the token bucket.
// Wrong fix attempt (rework 2): the added clause re-admits everything, so the
// bucket boundary is still not enforced.
func Allow(used, capacity int) bool {
	return used < capacity || used == capacity // BUG: admits every request
}
`
}

// bfFixedContent 是修复后候选:边界恢复 `used < capacity`(第 4 次请求被拒绝,
// 直到 token 补充),携带唯一回归检查 token。
func bfFixedContent() string {
	return `package limiter

// Allow reports whether an immediate request is admitted by the token bucket.
// TASK-2410 fix (` + bfFixedToken + `): with capacity=3 and no refill elapsed,
// the first three immediate requests are allowed and the fourth is rejected
// until tokens refill.
func Allow(used, capacity int) bool {
	return used < capacity
}
`
}

func bfLimiterTestContent() string {
	return `package limiter

// Regression check fixture for TASK-2410. The frozen protocol (see
// .tally/verifications/TASK-2410.yaml) is executed by the verification
// authority only — never by the implementation side.
`
}

// bfBehaviorArgv 是 AC-01 行为验收的冻结命令(离线、确定性):修复后候选的
// 边界表达式 `return used < capacity` 整行锚定匹配;修复前任何形态(基线或
// 错修复候选)都不匹配 → 非零退出 = 可信 FAIL。
func bfBehaviorArgv() []string {
	return []string{"git", "grep", "--no-index", "-q", `return used < capacity$`, bfLimiterRel}
}

// bfRegressionArgv 是 AC-02 回归检查的冻结命令:grep 唯一修复 token ——
// 同一检查在修复前(基线/缺陷候选)确定性失败、修复后通过。
func bfRegressionArgv() []string {
	return []string{"git", "grep", "--no-index", "-q", bfFixedToken, bfLimiterRel}
}

const bfProjectYAML = `project: proj-p2410
entities:
  - id: limiter
    type: go_package
    path: internal/limiter
`

const bfRequirementMD = `# REQ-2410 — Token bucket boundary defect

The token bucket limiter admits one extra immediate request after the bucket
is emptied; requests beyond capacity must be rejected until tokens refill.

Deterministic reproduction: with capacity=3 and no refill elapsed, the fourth
immediate request is admitted (want: reject).
`

// bfVerificationYAML 是 TASK-2410 的冻结 VerificationContract:两个 visible
// 绑定(L1 确定性地板,决策锁 21 —— 束不把 held-out 写成默认要求)。
// 命令 grep 是真实测试执行的确定性离线代理(P2-10 同一纪律:非零退出 =
// 可信 FAIL)。
const bfVerificationYAML = `task_id: TASK-2410
revision: 1
acceptances:
  - id: AC-01
    statement: with capacity=3 and no refill elapsed, the first three immediate requests are allowed and the fourth is rejected
    independence_required: none
    evidence_bindings:
      - id: BEH-1
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: test
          path: internal/limiter/token_bucket_test.go
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [git, grep, --no-index, -q, 'return used < capacity$', internal/limiter/token_bucket.go]
  - id: AC-02
    statement: a regression check fails on the pre-fix limiter and passes after the fix
    independence_required: none
    evidence_bindings:
      - id: REG-1
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: test
          path: internal/limiter/token_bucket_test.go
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
          command:
            argv: [git, grep, --no-index, -q, P2410-BOUNDARY-FIX, internal/limiter/token_bucket.go]
`

// ---- P3-01 冻结语料读取(仓库内单一事实源)----------------------------------

// bfRepoRoot 从测试 CWD(test/e2e)解析仓库根。
func bfRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("定位仓库根: %v", err)
	}
	return root
}

func bfReadFixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	p := filepath.Join(append([]string{bfRepoRoot(t)}, parts...)...)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读夹具 %s: %v", p, err)
	}
	return data
}

// bfTaskContract 用真实 TaskContract 解析器读 P3-01 夹具合同(与装进夹具
// 仓库的字节同源)。
func bfTaskContract(t *testing.T) taskcontract.TaskContract {
	t.Helper()
	c, err := taskcontract.ParseFile(filepath.Join(bfRepoRoot(t),
		"test", "fixtures", "v14", "bugfix", "contracts", bfTaskID+".yaml"))
	if err != nil {
		t.Fatalf("夹具合同 %s 必须是合法 TaskContract: %v", bfTaskID, err)
	}
	return c
}

// bfFactsFile 是夹具事实文件(test/fixtures/v14/bugfix/facts)的解码载体:
// 执行期匹配输入的变更轴与风险候选原始事实(P3-01 语料)。
type bfSignalInput struct {
	ID    string   `yaml:"id"`
	Basis []string `yaml:"basis"`
}

type bfFactsFile struct {
	ChangedPaths []string        `yaml:"changed_paths"`
	RiskSignals  []bfSignalInput `yaml:"risk_signals"`
}

func bfLoadFacts(t *testing.T) bfFactsFile {
	t.Helper()
	var f bfFactsFile
	if err := yaml.Unmarshal(bfReadFixture(t, "test", "fixtures", "v14", "bugfix", "facts", bfTaskID+".yaml"), &f); err != nil {
		t.Fatalf("解码夹具事实 %s: %v", bfTaskID, err)
	}
	return f
}

// bfRepairCase / bfRepairFixture 是已知可修复缺陷夹具
// (test/fixtures/v14/bugfix/repairable_failure)的解码载体:regression_check
// cases 里 buggy_candidate_got ≠ want 的行是「已知的修复前失败」;
// pre/post_repair_fulfillments 是修复前后候选在束 requires 键上的满足投影。
type bfRepairCase struct {
	Input             string `yaml:"input"`
	Want              string `yaml:"want"`
	BuggyCandidateGot string `yaml:"buggy_candidate_got"`
}

type bfRepairFixture struct {
	Task            string `yaml:"task"`
	RequirementRef  string `yaml:"requirement_ref"`
	PolicyID        string `yaml:"policy_id"`
	Defect          string `yaml:"defect"`
	RegressionCheck struct {
		Name  string         `yaml:"name"`
		Cases []bfRepairCase `yaml:"cases"`
	} `yaml:"regression_check"`
	PreRepairFulfillments  []engineeringpolicy.RequirementFulfillment `yaml:"pre_repair_fulfillments"`
	PostRepairFulfillments []engineeringpolicy.RequirementFulfillment `yaml:"post_repair_fulfillments"`
}

func bfLoadRepairFixture(t *testing.T) bfRepairFixture {
	t.Helper()
	var fx bfRepairFixture
	if err := yaml.Unmarshal(bfReadFixture(t, "test", "fixtures", "v14", "bugfix", "repairable_failure", bfTaskID+".yaml"), &fx); err != nil {
		t.Fatalf("解码可修复缺陷夹具 %s: %v", bfTaskID, err)
	}
	return fx
}

// bfAssertFixtureGateParity 对账 P3-01 可修复缺陷夹具与真实闭环执行:
// 夹具的 pre/post 满足投影与真实执行观察投影逐键一致;夹具已知修复前失败
// 恰好是边界用例(第 4 次请求 want=reject / buggy got=allow);夹具指向
// 本任务与本内置束。
func bfAssertFixtureGateParity(t *testing.T, fx bfRepairFixture, pre, post []engineeringpolicy.RequirementFulfillment) {
	t.Helper()
	if fx.Task != bfTaskID || fx.PolicyID != bfBugFixPolicyID {
		t.Fatalf("夹具指向 task=%s policy=%s, 期望 %s / %s", fx.Task, fx.PolicyID, bfTaskID, bfBugFixPolicyID)
	}
	failed := 0
	for _, c := range fx.RegressionCheck.Cases {
		if c.BuggyCandidateGot == c.Want {
			continue
		}
		failed++
		if !strings.Contains(c.Input, "4th immediate request") || c.Want != "reject" || c.BuggyCandidateGot != "allow" {
			t.Fatalf("已知修复前失败用例 = %+v, 期望第 4 次请求 reject/allow 边界", c)
		}
	}
	if failed != 1 {
		t.Fatalf("夹具已知修复前失败 = %d 个, 期望恰好 1", failed)
	}
	toMap := func(fs []engineeringpolicy.RequirementFulfillment) map[string]bool {
		m := make(map[string]bool, len(fs))
		for _, f := range fs {
			m[f.Key] = f.Satisfied
		}
		return m
	}
	if !reflect.DeepEqual(toMap(fx.PreRepairFulfillments), toMap(pre)) {
		t.Fatalf("夹具修复前投影 = %+v, 期望与真实执行一致 %+v", fx.PreRepairFulfillments, pre)
	}
	if !reflect.DeepEqual(toMap(fx.PostRepairFulfillments), toMap(post)) {
		t.Fatalf("夹具修复后投影 = %+v, 期望与真实执行一致 %+v", fx.PostRepairFulfillments, post)
	}
}

// ---- 夹具仓库(bug-fix 场景)--------------------------------------------------

// bfNewRepo 建带 .tally 制品(含 FrozenIntent;P1-01 同源 bundle 装载的
// 第四类权威制品)、内置束与含缺陷基线的真 git 仓库:基线提交的令牌桶就是
// 缺陷本体(回归检查在基线上确定性失败,AC1 的复现前提)。intent 为低风险
// bug_fix(无风险提示):权威风险输入 = 确定性任务事实,P1-02 DeriveRisk。
func bfNewRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	clGit(t, dir, "init", "-b", "main")
	clGit(t, dir, "config", "user.name", "tally-p303-e2e")
	clGit(t, dir, "config", "user.email", "tally-p303-e2e@example.invalid")
	clWriteFile(t, dir, ".tally/project.yaml", bfProjectYAML)
	clWriteFile(t, dir, ".tally/requirements/REQ-2410.md", bfRequirementMD)
	clWriteFile(t, dir, ".tally/intents/"+bfTaskID+".yaml",
		"task_id: "+bfTaskID+"\ntask_type: bug_fix\nsource_ref: REQ-2410 token bucket boundary defect\n")
	// 冻结合同逐字取自 P3-01 夹具目录(单一事实源)。
	clWriteFile(t, dir, ".tally/tasks/"+bfTaskID+".yaml", string(bfReadFixture(t, "test", "fixtures", "v14", "bugfix", "contracts", bfTaskID+".yaml")))
	clWriteFile(t, dir, ".tally/verifications/"+bfTaskID+".yaml", bfVerificationYAML)
	// 内置束按 Git-first 纪律装进项目提交树(与 catalog 同字节)。
	clWriteFile(t, dir, ".tally/policies/bug_fix.yaml", string(bfReadFixture(t, "catalog", "v14", "policies", "bug_fix.yaml")))
	clWriteFile(t, dir, bfLimiterRel, bfBaselineContent())
	clWriteFile(t, dir, bfLimiterTestRel, bfLimiterTestContent())
	clGit(t, dir, "add", ".")
	clGit(t, dir, "commit", "-m", "p3-03 bugfix fixtures")
	return dir
}

// bfExitCode 在 dir 内执行冻结协议 argv,返回退出码(运行器故障才 fatal;
// 非零退出是可信的确定性观察结果,不是错误)。
func bfExitCode(t *testing.T, dir string, argv []string) int {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: argv,
		Dir:  dir,
		Env:  e2eGitEnv(),
	})
	if err != nil {
		t.Fatalf("运行 %v: %v", argv, err)
	}
	return res.ExitCode
}

// ---- 束:Git-first 装载 + 真实匹配器 + GateKeys 编译 ------------------------

// bfMatchBundle 在闭环执行之前完成执行期策略装配:内置束经真实解析器
// (engineeringpolicy.Parse)与生产装载器(artifacts.LoadPolicies,项目提交
// 树 .tally/policies/)装载,再经真实匹配器(engineeringpolicy.Match)命中
// P3-01 夹具的任务/变更/风险事实。返回命中覆盖与命中策略(门禁求值用)。
func bfMatchBundle(t *testing.T, repo, rev string) (engineeringpolicy.PolicyMatchCoverage, engineeringpolicy.Policy) {
	t.Helper()

	catalog, err := engineeringpolicy.Parse(bfReadFixture(t, "catalog", "v14", "policies", "bug_fix.yaml"))
	if err != nil {
		t.Fatalf("内置束必须通过真实解析器: %v", err)
	}

	ps, err := artifacts.LoadPolicies(repo, rev)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	if len(ps.Rejected) != 0 || len(ps.Policies) != 1 {
		t.Fatalf("策略装载 = %d 份、拒收 %v, 期望恰好 1 份合法束", len(ps.Policies), ps.Rejected)
	}
	loaded := ps.Policies[0]
	if loaded.Path != ".tally/policies/bug_fix.yaml" ||
		loaded.Policy.ID != catalog.ID || loaded.Policy.Revision != catalog.Revision {
		t.Fatalf("装载束 = path %s id %s rev %s, 期望与 catalog 同一策略同一内容修订 %s",
			loaded.Path, loaded.Policy.ID, loaded.Policy.Revision, catalog.Revision)
	}

	// 执行期匹配输入 = P3-01 夹具事实(任务轴 = 合同 requirement_refs;
	// 变更轴 = changed_paths;风险轴 = risk.Derive)。
	facts := bfLoadFacts(t)
	contract := bfTaskContract(t)
	sigs := make([]risk.Signal, 0, len(facts.RiskSignals))
	for _, s := range facts.RiskSignals {
		sigs = append(sigs, risk.Signal{ID: risk.SignalID(s.ID), Basis: s.Basis})
	}
	derived, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	cov, err := engineeringpolicy.Match([]engineeringpolicy.Policy{loaded.Policy}, engineeringpolicy.MatchInput{
		Risk:            derived,
		ChangedPaths:    facts.ChangedPaths,
		RequirementRefs: contract.RequirementRefs,
	})
	if err != nil {
		t.Fatalf("engineeringpolicy.Match: %v", err)
	}
	if cov.HasUnmatchedRisk() {
		t.Fatalf("低风险缺陷夹具不得留下未覆盖风险信号: %v", cov.UnmatchedRiskSignals)
	}
	if cov.Confidence != engineeringpolicy.ConfidenceFull ||
		len(cov.MatchedPolicies) != 1 || cov.MatchedPolicies[0].Policy.ID != bfBugFixPolicyID {
		t.Fatalf("匹配覆盖 = confidence %s policies %+v, 期望 full + 恰好命中 %s",
			cov.Confidence, cov.MatchedPolicies, bfBugFixPolicyID)
	}
	if basis := cov.MatchedPolicies[0].Basis; !reflect.DeepEqual(basis.RequirementRefs, []string{".tally/requirements/REQ-2410.md"}) {
		t.Fatalf("basis.RequirementRefs = %v, 期望需求轴命中 REQ-2410", basis.RequirementRefs)
	}
	return cov, loaded.Policy
}

// ---- 脚本化 FakeRuntime(claim 一律来自真 FakeRuntime)----------------------

// bfStep 是一次 Execute 轮次的脚本(与 clStep 同语义,写入路径换成本场景
// 交付物):write 非空则在会话的一次性 worktree 内写入令牌桶候选;inspect
// 在写入前观察 workspace 与指令(修复基物化断言必须发生在当下)。
type bfStep struct {
	write   string
	err     error
	inspect func(ws, instruction string)
}

type bfScriptedRuntime struct {
	inner agentruntime.Runtime

	mu    sync.Mutex
	steps []bfStep
	instr []string
}

func newBfScriptedRuntime(steps ...bfStep) *bfScriptedRuntime {
	return &bfScriptedRuntime{inner: fake.New(fake.Config{}), steps: steps}
}

func (r *bfScriptedRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.inner.Descriptor() }

func (r *bfScriptedRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &bfScriptedSession{rt: r, inner: s, ws: req.Workspace}, nil
}

// instructions 返回已观测到的迭代指令(按 Execute 顺序的拷贝)。
func (r *bfScriptedRuntime) instructions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.instr...)
}

type bfScriptedSession struct {
	rt    *bfScriptedRuntime
	inner agentruntime.Session
	ws    string
}

func (s *bfScriptedSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.rt.mu.Lock()
	if len(s.rt.steps) == 0 {
		s.rt.mu.Unlock()
		return agentruntime.AgentRunResult{}, fmt.Errorf("bfScriptedRuntime: 脚本耗尽")
	}
	step := s.rt.steps[0]
	s.rt.steps = s.rt.steps[1:]
	s.rt.instr = append(s.rt.instr, req.Instruction)
	s.rt.mu.Unlock()

	if step.inspect != nil {
		step.inspect(s.ws, req.Instruction)
	}
	if step.err != nil {
		return agentruntime.AgentRunResult{}, step.err
	}
	if step.write != "" {
		p := filepath.Join(s.ws, filepath.FromSlash(bfLimiterRel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
		if err := os.WriteFile(p, []byte(step.write), 0o644); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
	}
	return s.inner.Execute(ctx, req)
}

func (s *bfScriptedSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *bfScriptedSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *bfScriptedSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *bfScriptedSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// ---- 配置与测试等价组合(clRunTask 同型)-------------------------------------

type bfConfig struct {
	Repo         string
	WorktreeRoot string
	EnvName      string
	MaxRepair    int // 0 = 缺省 2(决策锁 23)
}

func bfConfigFor(t *testing.T, repo string) bfConfig {
	t.Helper()
	return bfConfig{
		Repo:         repo,
		WorktreeRoot: filepath.Join(t.TempDir(), "wt"),
		EnvName:      "p303-e2e",
	}
}

// bfDeps 是组合的依赖面(runtime 必须注入:FakeRuntime 脚本)。
type bfDeps struct {
	cfg     bfConfig
	queries *store.Queries
	runtime agentruntime.Runtime
}

// bfObservation 是一次闭环执行的可观测面。
type bfObservation struct {
	Result closedloop.Result
	Frozen *verification.Frozen
	Task   taskcontract.TaskContract
	Instrs []string
}

// bfContextPack 与 worker v14ContextPack 同规则(P1-05 compiled-aware):
// contextpack.Select;候选路径过滤换成本场景包路径,visible test 证据路径
// 照常纳入;requires 注入编译权威的执行前 requires 上下文(Prevention,
// 纳入 context_hash)。
func bfContextPack(t *testing.T, repo, base string, task taskcontract.TaskContract, frozen *verification.Frozen, requires contract.RequiresContext) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{"git", "-C", repo, "ls-tree", "-r", "--name-only", "-z", base},
		Env:  e2eGitEnv(),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git ls-tree %s: err=%v res=%+v", base, err, res)
	}
	seen := map[string]bool{}
	var candidates []contextpack.Candidate
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		candidates = append(candidates, contextpack.Candidate{Path: p})
	}
	for _, p := range strings.Split(res.Stdout, "\x00") {
		if p == "" {
			continue
		}
		if p == strings.TrimSuffix(p, "/") && (p == "internal/limiter" || strings.HasPrefix(p, "internal/limiter/")) {
			add(p)
		}
	}
	c := frozen.Contract()
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			if b.VisibilityPolicy == verification.VisibilityVisible &&
				b.Evidence.Type == verification.EvidenceTest && b.Evidence.Path != "" {
				add(b.Evidence.Path)
			}
		}
	}
	out, err := contextpack.Select(contextpack.SelectInput{
		Task:         task,
		Verification: frozen,
		Candidates:   candidates,
		Requires:     requires,
	})
	if err != nil {
		t.Fatalf("contextpack.Select: %v", err)
	}
	return out.Pack.Hash()
}

// bfRunTask 按 cmd/worker runV14Task 的生产组成执行 TASK-2410 的 v1.4 闭环
// (P1-08 编译权威对齐):同源 bundle 装载(P1-01,DL-04)→ 编译权威(P1-03,
// 执行的第一道门)→ 一次性 worktree 根 → compiled-aware ContextPack(P1-05)
// → 生产事实面(RunStoreSink / NewStoreLineageSink / NewStoreExposureSink)
// → 生产候选验证服务 → closedloop.NewLoop(LoopConfig.Authority 必达:P1-06
// 验收守卫在测试等价组合同样生效)→ Engine.Run。
func bfRunTask(t *testing.T, ctx context.Context, deps bfDeps, taskID string) bfObservation {
	t.Helper()
	cfg := deps.cfg

	bundle, err := artifacts.LoadBundle(cfg.Repo, "HEAD", taskID)
	if err != nil {
		t.Fatalf("装载 %s 的同源执行 bundle(HEAD): %v", cfg.Repo, err)
	}
	if bundle.ProjectID == "" {
		t.Fatalf("提交 %s 缺少 .tally/project.yaml(项目身份必须由 Git 注册表声明)", bundle.SourceCommit)
	}
	task := bundle.Task.Contract
	if task.Profile != taskcontract.ProfileStandard {
		t.Fatalf("task %s profile=%q,非 standard", taskID, task.Profile)
	}
	frozen := bundle.Verification.Frozen

	// 编译权威(P1-03/P1-04,PRD §16 no-bypass):先于任何 runtime 会话与
	// workspace 创建;fail-safe 类型化上抛,不进闭环。
	authority, err := bundle.CompileAuthority()
	if err != nil {
		t.Fatalf("编译 %s 的执行权威: %v", taskID, err)
	}

	mgr, err := workspace.New(cfg.Repo, cfg.WorktreeRoot)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	worktrees := workspace.NewRunTable(mgr)

	packHash := bfContextPack(t, cfg.Repo, bundle.SourceCommit, task, frozen, authority.Compiled.Requires())

	runs := runstore.New(deps.queries)
	wfSink := workflow.NewRunStoreSink(runs)
	verify := closedloop.CandidateVerification(candidateverify.New(candidateverify.Config{
		Repo:         cfg.Repo,
		WorktreeRoot: cfg.WorktreeRoot,
		EnvName:      cfg.EnvName,
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p3-03"},
	}, candidateverify.Deps{Queries: deps.queries}))
	lineage, err := closedloop.NewStoreLineageSink(runs)
	if err != nil {
		t.Fatalf("NewStoreLineageSink: %v", err)
	}
	exposure, err := closedloop.NewStoreExposureSink(runs, frozen.Revision())
	if err != nil {
		t.Fatalf("NewStoreExposureSink: %v", err)
	}
	seeds, err := runs.ExposuresForTask(ctx, bfProject, taskID)
	if err != nil {
		t.Fatalf("读暴露事实种子: %v", err)
	}

	runnerImpl := &clCodegenRunner{
		project:  bfProject,
		task:     task,
		policy:   policy.DefaultExecutionPolicy(),
		profile:  "default-coding",
		runtime:  deps.runtime,
		model:    "test-model",
		behavior: "behavior-1",
		verifRev: frozen.Hash(),
		packHash: packHash,
		requires: authority.Compiled.Requires().Lines(),
		sink:     wfSink,
	}

	repair := closedloop.DefaultRepairPolicy()
	if cfg.MaxRepair != 0 {
		repair, err = closedloop.NewRepairPolicy(cfg.MaxRepair)
		if err != nil {
			t.Fatalf("修复预算非法: %v", err)
		}
	}
	loop, err := closedloop.NewLoop(closedloop.LoopConfig{
		Codegen:  runnerImpl,
		Verify:   verify,
		Contract: frozen,
		// 验收守卫(P1-06/P1-08,DL-07/A-03):编译权威是 accept 权威的输入。
		Authority: authority.Compiled,
		Runtime:   deps.runtime.Descriptor().Name,
		Worktrees: worktrees,
		BaseRevision: func(context.Context, string) (string, error) {
			return bundle.SourceCommit, nil
		},
		Repair:    repair,
		Lineage:   lineage,
		Exposure:  exposure,
		Exposures: seeds,
	})
	if err != nil {
		t.Fatalf("组装闭环: %v", err)
	}
	res, err := loop.Run(ctx, taskID)
	if err != nil {
		t.Fatalf("闭环执行: %v", err)
	}
	return bfObservation{Result: res, Frozen: frozen, Task: task, Instrs: deps.runtime.(*bfScriptedRuntime).instructions()}
}

// ---- PRD §31.1/§31.3:hands-off / 人工介入事件账目 ---------------------------

// bfEventLedger 把终态结果投影为可度量的离散事件账目:
//   - HandsOff:终态 accepted 且带 basis(零人工修码/零手工复制失败/零手工
//     重跑验证 —— 人工查看 diff 不算失败);
//   - AutomaticRepairs:闭环自动完成的外层修复次数(新 Run + 新候选 + 取代
//     事实);内层迭代不进入本账目(两个预算可区分,P2-11 AC2 纪律);
//   - InterventionEvents:升级产生的人工介入事件(人必须裁决;类型化原因,
//     如 repair_exhausted = manual repair/裁决事件)。
type bfEventLedger struct {
	HandsOff           bool
	AutomaticRepairs   int
	InterventionEvents []closedloop.EscalationReason
}

func bfLedger(res closedloop.Result) bfEventLedger {
	led := bfEventLedger{}
	if res.Phase == closedloop.PhaseAccepted && res.Basis != nil {
		led.HandsOff = true
	}
	if n := len(res.Attempts); n > 1 {
		led.AutomaticRepairs = n - 1
	}
	if res.Escalation != nil {
		led.InterventionEvents = append(led.InterventionEvents, res.Escalation.Reason)
	}
	return led
}

// ---- 共享断言辅助 ------------------------------------------------------------

// bfBindingVerdict 读某次验证运行中指定绑定的证据 verdict。
func bfBindingVerdict(t *testing.T, runs *runstore.Store, project core.ProjectID, verifyRunID int64, binding string) evidence.Verdict {
	t.Helper()
	evs, err := runs.EvidenceForRun(context.Background(), project, verifyRunID)
	if err != nil {
		t.Fatalf("EvidenceForRun(%d): %v", verifyRunID, err)
	}
	for _, ev := range evs {
		if ev.BindingID == binding {
			return ev.Verdict
		}
	}
	t.Fatalf("验证运行 %d 没有 %s 的证据行(共 %d 行)", verifyRunID, binding, len(evs))
	return ""
}
