package closedloop

// integration_test.go 实现 P2-08 的集成级测试:经包内组合根 NewLoop 把全部
// 已落地域件(一次性 worktree 执行适配 + candidateverify 验证适配 + P2-03
// 路由 + P2-04 反馈 + P2-07 预算 + P2-05 取代出口 + P2-06 暴露账面)组成
// 工作闭环,并用 fake 内循环 / fake 验证服务驱动真实边界(真 Git 仓库、真
// 一次性 worktree 物化)走完整 loop。
//
// 覆盖(任务验收逐条):
//   - AC1 首过 PASS:恰好一次执行、一次验证,零修复,Accepted 带 basis;
//   - AC2 修复:可信 FAIL → 分类 repairable → 新 Run + 新候选 + reason=rework
//     取代事实 + 安全反馈进入修复编码上下文 + 全新 worktree 物化继续基 +
//     同一冻结契约复验 → Accepted;
//   - AC3 非可修复(needs_review → requires_human_judgment)升级,不修复;
//     预算耗尽显式 repair_exhausted 升级;
//   - AC4 取消:执行中/验证中取消都不伪造 verdict,半途尝试不入谱系,
//     一次性 worktree 照常清理;复验路径对取消不重试;
//   - AC5 每个终态都带 basis 或类型化 reason;
//   - 暴露守卫:修复在途蓄意披露 → verified 在先也不得 accept,
//     exposure_compromise 升级;
//   - ERROR(verify 端口 error)的有界 infra 复验:不耗修复预算、不伪装成
//     候选失败;非可重试类立即升级。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/verification"
	"tally/internal/workspace"
)

// ---- 固件:任务/契约词表 ----

const (
	loopProject    = core.ProjectID("proj-p2-08")
	loopTask       = "TASK-P2-08"
	loopAcceptance = "AC-1"
	loopBinding    = "vis-core"
	loopGate       = "build"
	// lpHoldoutBinding 是 held-out binding id:出现在任何实现侧通道即泄漏。
	lpHoldoutBinding   = "holdout-secret-core"
	loopRuntime        = "test-runtime"
	loopWidgetFilePath = "pkg/widget.go"
)

// lpHoldoutDigest 是 held-out 内容摘要形态(泄漏扫描用)。
var lpHoldoutDigest = "sha256:" + strings.Repeat("cd", 32)

// lpFixedAt 是测试候选的固化时刻(早于一切暴露/取代时刻,时间规则确定)。
var lpFixedAt = fixedNow.Add(-2 * time.Hour)

// ---- 固件:真实 Git 仓库(一次性 worktree 的检出基线) ----

// lpGitEnv 隔离环境机 git 配置(与 internal/workspace 同纪律)。
func lpGitEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}

func lpGit(t *testing.T, dir string, argv ...string) string {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  lpGitEnv(),
	})
	if err != nil {
		t.Fatalf("git %v: %v", argv, err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("git %v: exit %d: %s", argv, res.ExitCode, res.Stderr)
	}
	return strings.TrimSpace(res.Stdout)
}

// lpRepo 建一个一次性 git 仓库(基线提交 README.md),返回 (repo, base SHA)。
func lpRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	lpGit(t, dir, "init", "-b", "main")
	lpGit(t, dir, "config", "user.name", "tally-test")
	lpGit(t, dir, "config", "user.email", "tally-test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("base readme\n"), 0o644); err != nil {
		t.Fatalf("写基线文件: %v", err)
	}
	lpGit(t, dir, "add", ".")
	lpGit(t, dir, "commit", "-m", "base")
	return dir, lpGit(t, dir, "rev-parse", "HEAD")
}

// ---- 固件:冻结验证契约(visible binding + required gate) ----

func lpContract(t *testing.T) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:   loopTask,
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:        loopAcceptance,
			Statement: "the widget greets deterministically",
			EvidenceBindings: []verification.Binding{{
				ID:               loopBinding,
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityVisible,
				Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "pkg/widget_test.go", Selector: "TestWidget"},
			}},
		}},
		RequiredGates: map[string]verification.Gate{
			loopGate: {Required: true, Command: &verification.CommandSpec{Argv: []string{"go", "build", "./..."}, Dir: "."}},
		},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("冻结契约: %v", err)
	}
	return f
}

// ---- 固件:fake 内循环 CodegenRunner ----

type lpRunStep struct {
	content string                  // 本次尝试固化的候选文件内容
	err     error                   // 非空:进程/基础设施失败
	cancel  context.CancelFunc      // 非空:调用时先取消 ctx(半途产出不采信)
	onRun   func(CodegenRunRequest) // 旁路观察钩子(物化断言/在途暴露记账)
}

type loopRunner struct {
	calls []CodegenRunRequest
	steps []lpRunStep
}

func (r *loopRunner) RunCodegen(ctx context.Context, req CodegenRunRequest) (*changeset.ChangeSet, error) {
	r.calls = append(r.calls, req)
	if len(r.steps) == 0 {
		return nil, errors.New("loopRunner: 脚本耗尽")
	}
	s := r.steps[0]
	r.steps = r.steps[1:]
	if s.onRun != nil {
		s.onRun(req)
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.err != nil {
		return nil, s.err
	}
	// 生产 runner 契约:候选是冻结不可变快照,ProducerRun = 本次 Run 身份。
	return changeset.New(changeset.Spec{
		Project:      loopProject,
		BaseRevision: req.BaseRevision,
		Files:        []changeset.ChangedFile{{Path: loopWidgetFilePath, Content: s.content}},
		ProducerRun:  string(req.RunID),
		CreatedAt:    lpFixedAt,
	})
}

// ---- 固件:fake 候选验证服务 ----

type lpVerifyStep struct {
	status   verification.FinalStatus
	runID    int64
	bindings []verification.BindingOutcome
	gates    []verification.GateResult
	holdout  []verification.HoldoutVerdictRecord
	// functional 非空:FinalOutcome 携带该功能覆盖报告(P1-06 验收守卫的
	// 事实面);nil = 无观察报告(与真实编排器的 FinalVerified 不同,真实
	// FinalVerified 恒携带完整报告 —— 需要模拟真实形态的测试显式提供)。
	functional *verification.FunctionalCoverageReport
	err        error
	cancel     context.CancelFunc
}

type loopVerification struct {
	calls    []candidateverify.Request
	contract *verification.Frozen // 期望的冻结契约(逐次调用核对 pin 不变)
	steps    []lpVerifyStep
}

func (v *loopVerification) Verify(_ context.Context, req candidateverify.Request) (*verification.FinalOutcome, error) {
	v.calls = append(v.calls, req)
	if v.contract != nil && req.Contract.Hash() != v.contract.Hash() {
		return nil, fmt.Errorf("loopVerification: 收到契约 %s,期望 %s(契约 pin 漂移)", req.Contract.Hash(), v.contract.Hash())
	}
	if len(v.steps) == 0 {
		return nil, errors.New("loopVerification: 脚本耗尽")
	}
	s := v.steps[0]
	v.steps = v.steps[1:]
	if s.cancel != nil {
		s.cancel()
	}
	if s.err != nil {
		return nil, s.err
	}
	out := &verification.FinalOutcome{
		RunID:            s.runID,
		TaskID:           req.Contract.TaskID(),
		ContractRevision: req.Contract.Revision(),
		ChangeSetID:      req.ChangeSet.ID(),
		Status:           s.status,
		Bindings:         s.bindings,
		Gates:            s.gates,
		Holdout:          s.holdout,
	}
	if s.functional != nil {
		out.Functional = *s.functional
	}
	return out, nil
}

// lpFailedVisible 是一条失败 visible binding 观察(触发 repairable 分类)。
func lpFailedVisible() []verification.BindingOutcome {
	return []verification.BindingOutcome{{
		BindingID:    loopBinding,
		AcceptanceID: loopAcceptance,
		Status:       verification.EvidenceStatusFail,
	}}
}

// ---- 固件:组合 ----

type lpFixture struct {
	engine  *Engine
	codegen *loopRunner
	verify  *loopVerification
	base    string
	root    string
}

// lpLoop 组装完整闭环(真实 git 仓库 + 一次性 worktree 表)。
func lpLoop(t *testing.T, codegen *loopRunner, verify *loopVerification, policy RepairPolicy, lineage *memLineage, seed []evidence.Exposure, sink ExposureSink, infraAttempts int) *lpFixture {
	t.Helper()
	repo, base := lpRepo(t)
	root := filepath.Join(t.TempDir(), "wt")
	mgr, err := workspace.New(repo, root)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	frozen := lpContract(t)
	verify.contract = frozen
	engine, err := NewLoop(LoopConfig{
		Codegen:       codegen,
		Verify:        verify,
		Contract:      frozen,
		Runtime:       loopRuntime,
		Worktrees:     workspace.NewRunTable(mgr),
		BaseRevision:  func(context.Context, string) (string, error) { return base, nil },
		Repair:        policy,
		Lineage:       lineage,
		Exposure:      sink,
		Exposures:     seed,
		InfraAttempts: infraAttempts,
		Now:           func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewLoop: %v", err)
	}
	return &lpFixture{engine: engine, codegen: codegen, verify: verify, base: base, root: root}
}

// lpAssertRootEmpty 断言一次性 worktree 根已清空(尝试结束即丢弃沙箱)。
func lpAssertRootEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("读一次性根 %s: %v", root, err)
	}
	if len(entries) != 0 {
		t.Fatalf("一次性根 %s 残留 %d 项,期望已清空", root, len(entries))
	}
}

// lpDefault 组合参数:默认修复策略 + 内存取代出口。
func lpDefault(t *testing.T, codegen *loopRunner, verify *loopVerification) (*lpFixture, *memLineage) {
	t.Helper()
	lineage := &memLineage{}
	policy, err := NewRepairPolicy(DefaultMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy: %v", err)
	}
	return lpLoop(t, codegen, verify, policy, lineage, nil, nil, 0), lineage
}

// ---- AC1:首过 PASS,零修复 ----

func TestLoopFirstPassAccepted(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalVerified, runID: 41},
	}}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted {
		t.Fatalf("Phase = %s, 期望 accepted", res.Phase)
	}
	if res.Basis == nil || res.Basis.VerifyRun != "41" || res.Basis.Attempt != 1 {
		t.Fatalf("Basis = %+v, 期望 {候选1, 41, 1}(AC5:Accepted 必带依据)", res.Basis)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].Supersedes != "" {
		t.Errorf("Attempts = %+v, 期望恰好一次无取代尝试(AC1:零修复)", res.Attempts)
	}
	if len(fx.codegen.calls) != 1 || len(fx.verify.calls) != 1 {
		t.Errorf("端口调用 = exec %d / verify %d, 期望 1/1", len(fx.codegen.calls), len(fx.verify.calls))
	}
	if len(lineage.facts) != 0 {
		t.Errorf("首过不应有取代事实,得到 %d 条", len(lineage.facts))
	}
	if fx.codegen.calls[0].RepairBase != nil || fx.codegen.calls[0].Feedback != nil {
		t.Errorf("首次尝试不应携带 RepairBase/Feedback:%+v", fx.codegen.calls[0])
	}
	if fx.verify.calls[0].ChangeSet.ID() != res.Basis.ChangeSet {
		t.Errorf("verdict 绑定对象 = %s, 期望被接受候选 %s", fx.verify.calls[0].ChangeSet.ID(), res.Basis.ChangeSet)
	}
	if fx.verify.calls[0].Runtime != loopRuntime {
		t.Errorf("验证请求 runtime = %q, 期望组合根钉住的 %q", fx.verify.calls[0].Runtime, loopRuntime)
	}
	lpAssertRootEmpty(t, fx.root)
}

// ---- AC2:FAIL → 分类 → 修复(新 Run/新候选/取代/复验)→ Accepted ----

func TestLoopRepairThenAccepted(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalFailed, runID: 100, bindings: lpFailedVisible()},
		{status: verification.FinalVerified, runID: 101},
	}}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
		{content: "candidate two"}, // 修复尝试:新 Run、新候选
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("期望 Accepted 带 basis,得到 %s / %+v", res.Phase, res.Basis)
	}
	if res.Basis.Attempt != 2 || res.Basis.VerifyRun != "101" {
		t.Errorf("Basis = %+v, 期望绑定修复候选与复验运行 101", res.Basis)
	}

	// 尝试谱系:父链接 + 取代。
	if len(res.Attempts) != 2 {
		t.Fatalf("Attempts = %d 条, 期望 2", len(res.Attempts))
	}
	a1, a2 := res.Attempts[0], res.Attempts[1]
	if a2.ParentRun != a1.RunID || a2.Supersedes != a1.ChangeSet {
		t.Errorf("修复尝试 = %+v, 期望父链接首次尝试并取代其候选", a2)
	}
	if len(lineage.facts) != 1 {
		t.Fatalf("取代事实 = %d 条, 期望 1", len(lineage.facts))
	}
	if fact := lineage.facts[0]; fact.Old != a1.ChangeSet || fact.New != a2.ChangeSet || fact.Reason != changeset.SupersedeRework || !fact.At.Equal(fixedNow) {
		t.Errorf("取代事实 = %+v, 期望 rework @ 注入时钟", fact)
	}

	// 修复 Run 的请求语义(ADR-0001 §2 + §4)。
	req2 := fx.codegen.calls[1]
	if req2.ParentRun != fx.codegen.calls[0].RunID {
		t.Errorf("修复 Run 父链接 = %s, 期望 %s", req2.ParentRun, fx.codegen.calls[0].RunID)
	}
	if req2.RepairBase == nil || req2.RepairBase.ID() != a1.ChangeSet {
		t.Fatalf("修复请求应携带冻结前候选作为 RepairBase,得到 %v", req2.RepairBase)
	}
	fb := req2.Feedback
	if fb == nil {
		t.Fatal("修复尝试必须携带安全反馈(P2-04 投影是唯一失败材料通道)")
	}
	if fb.Failure != FailureRepairable || fb.TaskID != loopTask || fb.Attempt != 1 || fb.ChangeSet != a1.ChangeSet {
		t.Errorf("反馈身份 = %+v, 期望绑定本次失败候选", fb)
	}
	if fb.Subject.AcceptanceID != loopAcceptance || fb.Subject.Statement != "the widget greets deterministically" {
		t.Errorf("反馈主体 = %+v, 期望冻结契约的 acceptance id + 语义文本", fb.Subject)
	}
	if !strings.Contains(fb.DiagnosticSummary, loopBinding) {
		t.Errorf("诊断摘要应含失败 binding id:%q", fb.DiagnosticSummary)
	}
	if len(fb.Evidence) != 1 || fb.Evidence[0].BindingID != loopBinding || fb.Evidence[0].Path != "pkg/widget_test.go" {
		t.Errorf("可见证据事实 = %+v, 期望失败 binding 的可见定位", fb.Evidence)
	}
	if len(fb.Holdout) != 0 {
		t.Errorf("无 held-out 失败时投影应为空,得到 %+v", fb.Holdout)
	}

	// 复验绑定修复候选,且同一冻结契约(两次调用的 pin 一致由 fake 强制)。
	if fx.verify.calls[1].ChangeSet.ID() != a2.ChangeSet {
		t.Errorf("复验对象 = %s, 期望修复候选 %s", fx.verify.calls[1].ChangeSet.ID(), a2.ChangeSet)
	}
	if len(fx.verify.calls) != 2 {
		t.Errorf("验证调用 = %d 次, 期望 2", len(fx.verify.calls))
	}
	lpAssertRootEmpty(t, fx.root)
}

// 修复尝试的 worktree 以基线检出 + 确定性物化冻结前候选(P2-05,ADR-0001 §2)。
func TestLoopRepairWorkspaceMaterializesFrozenBase(t *testing.T) {
	var sawBaseFile, sawWidget, sawReadme string
	var sawRepairBaseID changeset.ChangeSetID
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalFailed, runID: 100, bindings: lpFailedVisible()},
		{status: verification.FinalVerified, runID: 101},
	}}
	fx, _ := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
		{content: "candidate two", onRun: func(req CodegenRunRequest) {
			if req.RepairBase == nil {
				t.Error("修复尝试应携带 RepairBase")
				return
			}
			sawRepairBaseID = req.RepairBase.ID()
			files := req.RepairBase.Files()
			if len(files) != 1 {
				t.Fatalf("RepairBase 文件 = %+v", files)
			}
			// runner 在一次性 worktree 内观察:基线检出 + 前候选内容物化。
			b, err := os.ReadFile(filepath.Join(req.WorkspacePath, filepath.FromSlash(files[0].Path)))
			if err != nil {
				t.Fatalf("读物化文件: %v", err)
			}
			sawWidget = string(b)
			sawBaseFile = files[0].Content
			if b, err := os.ReadFile(filepath.Join(req.WorkspacePath, "README.md")); err == nil {
				sawReadme = string(b)
			}
		}},
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil || res.Phase != PhaseAccepted {
		t.Fatalf("期望 Accepted,得到 %s / %v", res.Phase, err)
	}
	if sawRepairBaseID == "" || sawBaseFile == "" {
		t.Fatal("修复 Run 未观察到冻结前候选")
	}
	if sawWidget != sawBaseFile {
		t.Errorf("物化内容 = %q, 期望冻结候选内容 %q 逐字一致", sawWidget, sawBaseFile)
	}
	if sawReadme != "base readme\n" {
		t.Errorf("基线检出文件 = %q, 期望基线提交内容", sawReadme)
	}
	// 每次尝试是不同的 Run 身份(幂等键)⇒ 不同一次性 worktree。
	if fx.codegen.calls[0].RunID == fx.codegen.calls[1].RunID {
		t.Errorf("两次尝试共享 Run 身份:%s", fx.codegen.calls[0].RunID)
	}
}

// ---- AC3:非可修复(needs_review → requires_human_judgment)升级,不修复 ----

func TestLoopNeedsReviewEscalatesWithoutRepair(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalNeedsReview, runID: 7},
	}}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil {
		t.Fatalf("期望 Escalated,得到 %s", res.Phase)
	}
	if res.Escalation.Reason != EscalationReason(FailureRequiresHuman) {
		t.Errorf("升级原因 = %s, 期望 requires_human_judgment(类型化,AC5)", res.Escalation.Reason)
	}
	if res.Basis != nil {
		t.Error("升级不得携带 basis")
	}
	if len(fx.codegen.calls) != 1 || len(fx.verify.calls) != 1 {
		t.Errorf("调用 = exec %d / verify %d, 期望 1/1(needs_review 不修复)", len(fx.codegen.calls), len(fx.verify.calls))
	}
	if len(lineage.facts) != 0 {
		t.Errorf("非修复路径不应有取代事实,得到 %d 条", len(lineage.facts))
	}
}

// 预算耗尽:显式 repair_exhausted 升级(默认策略 2 次修复,第 3 次被拒)。
func TestLoopRepairExhaustedEscalates(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalFailed, runID: 1, bindings: lpFailedVisible()},
		{status: verification.FinalFailed, runID: 2, bindings: lpFailedVisible()},
		{status: verification.FinalFailed, runID: 3, bindings: lpFailedVisible()},
	}}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "c1"},
		{content: "c2"},
		{content: "c3"},
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationRepairExhausted {
		t.Fatalf("期望 repair_exhausted 升级,得到 %+v", res.Escalation)
	}
	// 默认 MaxRepairAttempts=2:尝试 1 + 修复 2 次,第 3 次修复被预算拒绝。
	if len(res.Attempts) != 3 || len(fx.codegen.calls) != 3 {
		t.Errorf("尝试 = %d / exec 调用 = %d, 期望 3/3", len(res.Attempts), len(fx.codegen.calls))
	}
	if len(lineage.facts) != 2 {
		t.Errorf("取代事实 = %d 条, 期望 2(第 3 次修复未发生)", len(lineage.facts))
	}
	lpAssertRootEmpty(t, fx.root)
}

// ---- AC4:取消是一等终态,不伪造 verdict ----

func TestLoopCancellationDuringExecute(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	verify := &loopVerification{}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "half-done", cancel: cancel},
	}}, verify)

	res, err := fx.engine.Run(ctx, loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseCancelled {
		t.Fatalf("Phase = %s, 期望 cancelled", res.Phase)
	}
	if res.Basis != nil || res.Escalation != nil {
		t.Errorf("取消不得产生 basis/升级:%+v / %+v", res.Basis, res.Escalation)
	}
	if len(res.Attempts) != 0 || len(fx.verify.calls) != 0 || len(lineage.facts) != 0 {
		t.Errorf("半途尝试不得入谱系/触发验证/落取代:attempts=%d verify=%d facts=%d",
			len(res.Attempts), len(fx.verify.calls), len(lineage.facts))
	}
	// 取消后一次性 worktree 照常清理(Terminate 不继承调用方取消)。
	lpAssertRootEmpty(t, fx.root)
}

func TestLoopCancellationDuringVerify(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalVerified, runID: 5, cancel: cancel},
	}}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
	}}, verify)

	res, err := fx.engine.Run(ctx, loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseCancelled {
		t.Fatalf("Phase = %s, 期望 cancelled", res.Phase)
	}
	if res.Basis != nil {
		t.Errorf("取消的验证不得伪造 basis:%+v", res.Basis)
	}
	if len(res.Attempts) != 1 {
		t.Errorf("已完成尝试应保留为事实,得到 %d 条", len(res.Attempts))
	}
	if len(lineage.facts) != 0 {
		t.Errorf("取消不落取代事实,得到 %d 条", len(lineage.facts))
	}
	lpAssertRootEmpty(t, fx.root)
}

// 复验路径取消:ERROR 即使可重试类也绝不重试(取消优先于重试)。
func TestLoopCancelSuppressesInfraRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	verify := &loopVerification{steps: []lpVerifyStep{
		{err: fmt.Errorf("runner down: %w", verification.ErrNoWorkspace), cancel: cancel},
		{status: verification.FinalVerified, runID: 9},
	}}
	fx, _ := lpDefault(t, &loopRunner{steps: []lpRunStep{{content: "candidate one"}}}, verify)

	res, err := fx.engine.Run(ctx, loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseCancelled {
		t.Fatalf("Phase = %s, 期望 cancelled", res.Phase)
	}
	if len(fx.verify.calls) != 1 {
		t.Errorf("取消后不得复验,verify 调用 = %d", len(fx.verify.calls))
	}
}

// ---- 暴露守卫:修复在途蓄意披露 → verified 也不得 accept ----

func TestLoopExposureCompromiseEscalates(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalFailed, runID: 100, bindings: lpFailedVisible()},
		{status: verification.FinalVerified, runID: 101},
	}}
	sink := &memExposureSink{}
	var fx *lpFixture
	codegen := &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
		{content: "candidate two", onRun: func(req CodegenRunRequest) {
			if req.RepairBase == nil || fx == nil {
				return
			}
			// 修复在途蓄意披露(持久通道):先记账取得身份,再继续 ——
			// 这正是「修复期间暴露」的物理位置(P2-06)。
			if _, err := fx.engine.ExposureLedger().RecordExposure(context.Background(), RepairExposureSpec{
				TaskID:     loopTask,
				Binding:    evidence.BindingRef{Project: loopProject, Verification: "verif-r1", BindingID: lpHoldoutBinding},
				Acceptance: loopAcceptance,
				Anchor:     req.RepairBase.ID(),
				Channel:    evidence.ChannelValidationLog,
				Reason:     "intentional diagnostic disclosure during repair",
				ExposedAt:  fixedNow.Add(-time.Hour),
			}); err != nil {
				t.Errorf("在途暴露记账: %v", err)
			}
		}},
	}}
	lineage := &memLineage{}
	policy, err := NewRepairPolicy(DefaultMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy: %v", err)
	}
	fx = lpLoop(t, codegen, verify, policy, lineage, nil, sink, 0)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil {
		t.Fatalf("期望 Escalated,得到 %s", res.Phase)
	}
	if res.Escalation.Reason != EscalationExposureCompromise {
		t.Errorf("升级原因 = %s, 期望 exposure_compromise(INV-V14-07/08)", res.Escalation.Reason)
	}
	if res.Basis != nil {
		t.Error("compromise 谱系不得接受候选:basis 必须为空")
	}
	if len(res.Attempts) != 2 || len(lineage.facts) != 1 {
		t.Errorf("谱系应完整保留:attempts=%d facts=%d", len(res.Attempts), len(lineage.facts))
	}
	if len(sink.specs) != 1 {
		t.Errorf("暴露事实应持久化 1 条,得到 %d", len(sink.specs))
	}
}

// ---- ERROR 语义:有界 infra 复验(不耗修复预算、不伪装成候选失败) ----

func TestLoopInfraRetryThenVerified(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{err: fmt.Errorf("workspace gone: %w", verification.ErrNoWorkspace)},
		{status: verification.FinalVerified, runID: 9},
	}}
	fx, lineage := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("期望 Accepted,得到 %s / %+v", res.Phase, res.Basis)
	}
	if len(fx.verify.calls) != 2 {
		t.Errorf("复验调用 = %d 次, 期望 2(默认有界 2 次)", len(fx.verify.calls))
	}
	if len(fx.codegen.calls) != 1 || len(lineage.facts) != 0 {
		t.Errorf("infra 复验不得触发修复:exec=%d facts=%d", len(fx.codegen.calls), len(lineage.facts))
	}
}

// 非可重试 ERROR(unknown 类):立即升级,绝不重试、绝不伪装成候选失败。
func TestLoopUnknownVerifyErrorEscalatesImmediately(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{err: errors.New("boom")},
		{status: verification.FinalVerified, runID: 9},
	}}
	fx, _ := lpDefault(t, &loopRunner{steps: []lpRunStep{
		{content: "candidate one"},
	}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationInfrastructure {
		t.Fatalf("期望 infrastructure_failure 升级,得到 %+v", res.Escalation)
	}
	if len(fx.verify.calls) != 1 {
		t.Errorf("unknown 类不得复验,verify 调用 = %d", len(fx.verify.calls))
	}
}

// ---- AC5:未知结论 fail-safe(Inconclusive),终态恒带类型化载荷 ----

func TestLoopUnknownStatusEscalatesInconclusive(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{
		{status: verification.FinalStatus("sort-of"), runID: 3},
	}}
	fx, _ := lpDefault(t, &loopRunner{steps: []lpRunStep{{content: "candidate one"}}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationInconclusive {
		t.Fatalf("期望 verification_inconclusive 升级,得到 %+v", res.Escalation)
	}
	if res.Basis != nil {
		t.Error("不可信结论不得产生 basis")
	}
	if len(fx.codegen.calls) != 1 {
		t.Errorf("Inconclusive 不修复,exec 调用 = %d", len(fx.codegen.calls))
	}
}

// ---- 组合根校验 ----

func TestNewLoopConfigValidation(t *testing.T) {
	repo, base := lpRepo(t)
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	table := workspace.NewRunTable(mgr)
	frozen := lpContract(t)
	policy, err := NewRepairPolicy(DefaultMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy: %v", err)
	}
	baseFn := BaseRevisionFunc(func(context.Context, string) (string, error) { return base, nil })

	full := LoopConfig{
		Codegen:      &loopRunner{},
		Verify:       &loopVerification{},
		Contract:     frozen,
		Runtime:      loopRuntime,
		Worktrees:    table,
		BaseRevision: baseFn,
		Repair:       policy,
	}
	if _, err := NewLoop(full); err != nil {
		t.Fatalf("完整配置应可构造: %v", err)
	}
	for name, mutate := range map[string]func(*LoopConfig){
		"missing_codegen":   func(c *LoopConfig) { c.Codegen = nil },
		"missing_verify":    func(c *LoopConfig) { c.Verify = nil },
		"missing_contract":  func(c *LoopConfig) { c.Contract = nil },
		"missing_runtime":   func(c *LoopConfig) { c.Runtime = "  " },
		"missing_worktrees": func(c *LoopConfig) { c.Worktrees = nil },
		"missing_base":      func(c *LoopConfig) { c.BaseRevision = nil },
		"zero_policy":       func(c *LoopConfig) { c.Repair = RepairPolicy{} },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := full
			mutate(&cfg)
			if _, err := NewLoop(cfg); !errors.Is(err, ErrInvalidLoopConfig) {
				t.Fatalf("期望 ErrInvalidLoopConfig,得到 %v", err)
			}
		})
	}
}

// ---- ServiceVerifier 请求守卫与账面纪律 ----

func TestServiceVerifierRequestGuards(t *testing.T) {
	svc := &loopVerification{}
	v, err := NewServiceVerifier(ServiceVerifierConfig{Service: svc, Contract: lpContract(t), Runtime: loopRuntime})
	if err != nil {
		t.Fatalf("NewServiceVerifier: %v", err)
	}
	if _, err := NewServiceVerifier(ServiceVerifierConfig{Contract: lpContract(t), Runtime: loopRuntime}); !errors.Is(err, ErrInvalidVerifier) {
		t.Errorf("缺服务应拒绝构造,得到 %v", err)
	}
	if _, err := NewServiceVerifier(ServiceVerifierConfig{Service: svc, Contract: lpContract(t), Runtime: loopRuntime, InfraAttempts: HardInfraVerifyAttempts + 1}); !errors.Is(err, ErrInvalidVerifier) {
		t.Errorf("复验次数越界应拒绝构造,得到 %v", err)
	}

	cs := candidate(t, "c1")
	if _, err := v.VerifyCandidate(context.Background(), VerifyRequest{TaskID: loopTask, Attempt: 1, ChangeSet: cs.ID()}); !errors.Is(err, ErrInvalidVerifyCall) {
		t.Errorf("缺快照应 ErrInvalidVerifyCall,得到 %v", err)
	}
	if _, err := v.VerifyCandidate(context.Background(), VerifyRequest{TaskID: loopTask, Attempt: 1, ChangeSet: "other", Snapshot: cs}); !errors.Is(err, ErrInvalidVerifyCall) {
		t.Errorf("身份不一致应 ErrInvalidVerifyCall,得到 %v", err)
	}

	// append-only:同候选不同结论拒绝覆盖(verdict 只绑精确候选)。
	svc.steps = []lpVerifyStep{{status: verification.FinalFailed, runID: 1}, {status: verification.FinalVerified, runID: 2}}
	if _, err := v.VerifyCandidate(context.Background(), VerifyRequest{TaskID: loopTask, Attempt: 1, ChangeSet: cs.ID(), Snapshot: cs}); err != nil {
		t.Fatalf("首次验证: %v", err)
	}
	if _, err := v.VerifyCandidate(context.Background(), VerifyRequest{TaskID: loopTask, Attempt: 1, ChangeSet: cs.ID(), Snapshot: cs}); !errors.Is(err, ErrOutcomeAppendOnly) {
		t.Errorf("同候选不同结论应 ErrOutcomeAppendOnly,得到 %v", err)
	}
}

func TestVerdictMappingAndRunRef(t *testing.T) {
	cases := []struct {
		status verification.FinalStatus
		want   Verdict
	}{
		{verification.FinalVerified, VerdictVerified},
		{verification.FinalFailed, VerdictFailed},
		{verification.FinalNeedsReview, VerdictFailed}, // 可信非 VERIFIED 结论:路由分类,不冒充 ERROR
		{verification.FinalStatus(""), VerdictInconclusive},
		{verification.FinalStatus("weird"), VerdictInconclusive},
	}
	for _, tc := range cases {
		if got := verdictForStatus(tc.status); got != tc.want {
			t.Errorf("verdictForStatus(%q) = %s, 期望 %s", tc.status, got, tc.want)
		}
	}
	if verifyRunRef(0) != "" || verifyRunRef(-1) != "" {
		t.Error("无可信运行行 id 不得伪造引用")
	}
	if verifyRunRef(41) != "41" {
		t.Errorf("verifyRunRef(41) = %q, 期望 41", verifyRunRef(41))
	}
}

func TestOutcomeLedgerClassify(t *testing.T) {
	l := newOutcomeLedger()
	if got := l.classify(VerifyFailure{ChangeSet: "missing"}); got != FailureUnknown {
		t.Errorf("缺席结论应按 unknown(fail-safe),得到 %s", got)
	}
	cs := candidate(t, "c1")
	out := &verification.FinalOutcome{RunID: 12, ChangeSetID: cs.ID(), Status: verification.FinalNeedsReview}
	if err := l.record(out); err != nil {
		t.Fatalf("record: %v", err)
	}
	if got := l.classify(VerifyFailure{ChangeSet: cs.ID(), RunRef: "12"}); got != FailureRequiresHuman {
		t.Errorf("needs_review 分类 = %s, 期望 requires_human_judgment", got)
	}
	// 账面事实只增不改:调用方原地改写已交付的结论不得影响已记录事实。
	out.Status = verification.FinalFailed
	if got := l.classify(VerifyFailure{ChangeSet: cs.ID()}); got != FailureRequiresHuman {
		t.Errorf("改写原对象后分类 = %s, 期望仍为 requires_human_judgment", got)
	}
	// 重复记录同一身份 = 纪律违背,拒绝(fail-safe)。
	if err := l.record(&verification.FinalOutcome{RunID: 13, ChangeSetID: cs.ID(), Status: verification.FinalFailed}); !errors.Is(err, ErrOutcomeAppendOnly) {
		t.Errorf("同候选重复记录应 ErrOutcomeAppendOnly,得到 %v", err)
	}
}

// ---- FeedbackSource:从 FinalOutcome + 冻结契约构造安全投影 ----

func TestFeedbackSourceBuildsSafeProjection(t *testing.T) {
	frozen := lpContract(t)
	l := newOutcomeLedger()
	cs := candidate(t, "c1")
	record := verification.HoldoutVerdictRecord{
		TaskID:           loopTask,
		ContractRevision: 1,
		AcceptanceID:     loopAcceptance,
		BindingID:        lpHoldoutBinding,
		CandidateID:      string(cs.ID()),
		EvidenceType:     verification.EvidenceTest,
		ContentDigest:    lpHoldoutDigest,
		Status:           verification.EvidenceStatusFail,
		CompletedAt:      lpFixedAt,
	}
	out := &verification.FinalOutcome{
		RunID:            77,
		ChangeSetID:      cs.ID(),
		ContractRevision: 1,
		Status:           verification.FinalFailed,
		Bindings:         lpFailedVisible(),
		Holdout:          []verification.HoldoutVerdictRecord{record},
	}
	if err := l.record(out); err != nil {
		t.Fatalf("record: %v", err)
	}
	src := &contractFeedbackSource{ledger: l, contract: frozen}

	fb, err := src.RepairFeedback(context.Background(), VerifyFailure{TaskID: loopTask, Attempt: 1, ChangeSet: cs.ID(), RunRef: "77"})
	if err != nil {
		t.Fatalf("RepairFeedback: %v", err)
	}
	if fb.Failure != FailureRepairable || fb.Subject.AcceptanceID != loopAcceptance {
		t.Fatalf("反馈 = %+v", fb)
	}
	if len(fb.Evidence) != 1 || fb.Evidence[0].BindingID != loopBinding {
		t.Errorf("可见证据 = %+v", fb.Evidence)
	}
	if len(fb.Holdout) != 1 {
		t.Fatalf("held-out 投影 = %+v, 期望 1 条", fb.Holdout)
	}
	if !fb.Holdout[0].Outcome.CompletedAt.Equal(lpFixedAt) || fb.Holdout[0].Status != verification.EvidenceStatusFail {
		t.Errorf("held-out 投影条目 = %+v", fb.Holdout[0])
	}
	if !strings.Contains(fb.DiagnosticSummary, loopBinding) || !strings.Contains(fb.DiagnosticSummary, "held-out checks failed: 1") {
		t.Errorf("诊断摘要 = %q", fb.DiagnosticSummary)
	}

	// 泄漏扫描:held-out binding 身份与内容摘要不得出现在任何投影值面。
	blob, err := json.Marshal(fb)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, leak := range []string{lpHoldoutBinding, lpHoldoutDigest} {
		if strings.Contains(string(blob), leak) {
			t.Errorf("投影泄漏 %q:%s", leak, blob)
		}
	}
}

func TestFeedbackSourceGateSubjectAndFailsafes(t *testing.T) {
	frozen := lpContract(t)
	l := newOutcomeLedger()
	cs := candidate(t, "gate-c1")
	gateOut := &verification.FinalOutcome{
		RunID:       8,
		ChangeSetID: cs.ID(),
		Status:      verification.FinalFailed,
		Gates:       []verification.GateResult{{Name: loopGate, Status: verification.VisibleStatusFail}},
	}
	if err := l.record(gateOut); err != nil {
		t.Fatalf("record: %v", err)
	}
	src := &contractFeedbackSource{ledger: l, contract: frozen}

	fb, err := src.RepairFeedback(context.Background(), VerifyFailure{TaskID: loopTask, Attempt: 1, ChangeSet: cs.ID(), RunRef: "8"})
	if err != nil {
		t.Fatalf("gate 失败反馈: %v", err)
	}
	if fb.Subject.Gate != loopGate || fb.Subject.AcceptanceID != "" {
		t.Errorf("gate 主体 = %+v", fb.Subject)
	}

	// 结论缺席 → 拒绝构造(fail-safe 升级,绝不虚构材料)。
	if _, err := src.RepairFeedback(context.Background(), VerifyFailure{TaskID: loopTask, Attempt: 1, ChangeSet: "ghost"}); !errors.Is(err, ErrFeedbackOutcomeUnavailable) {
		t.Errorf("期望 ErrFeedbackOutcomeUnavailable,得到 %v", err)
	}

	// 非 repairable 结论 → Builder 拒绝(修复通道不为升级/infra 材料伪造入口)。
	l2 := newOutcomeLedger()
	cs2 := candidate(t, "nr-c1")
	if err := l2.record(&verification.FinalOutcome{RunID: 9, ChangeSetID: cs2.ID(), Status: verification.FinalNeedsReview}); err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, err := (&contractFeedbackSource{ledger: l2, contract: frozen}).RepairFeedback(context.Background(), VerifyFailure{TaskID: loopTask, Attempt: 1, ChangeSet: cs2.ID(), RunRef: "9"}); !errors.Is(err, ErrFailureNotRepairable) {
		t.Errorf("期望 ErrFailureNotRepairable,得到 %v", err)
	}
}

// ---- runstore 暴露出口适配(字段透传映射) ----

// lpExposureAppender 是 ExposureFactAppender 的 fake:捕获 runstore.ExposureRecord
// 供映射断言(真实 runstore.RecordExposure 的落库/触发器行为归 runstore 测试)。
type lpExposureAppender struct {
	records []runstore.ExposureRecord
}

func (a *lpExposureAppender) RecordExposure(_ context.Context, rec runstore.ExposureRecord) (evidence.Exposure, error) {
	a.records = append(a.records, rec)
	return evidence.NewExposure(evidence.ExposureSpec{
		ID:        "exp-p2-08",
		Binding:   rec.Binding,
		Channel:   rec.Channel,
		ChangeSet: rec.ChangeSet,
		ExposedAt: fixedNow,
	})
}

func TestStoreExposureSinkMapsSpecVerbatim(t *testing.T) {
	appender := &lpExposureAppender{}
	sink, err := NewStoreExposureSink(appender, 1)
	if err != nil {
		t.Fatalf("NewStoreExposureSink: %v", err)
	}
	if _, err := NewStoreExposureSink(nil, 1); !errors.Is(err, ErrInvalidExposureStoreSink) {
		t.Errorf("nil 存储应拒绝构造,得到 %v", err)
	}
	if _, err := NewStoreExposureSink(appender, 0); !errors.Is(err, ErrInvalidExposureStoreSink) {
		t.Errorf("非法 revision 应拒绝构造,得到 %v", err)
	}

	spec := RepairExposureSpec{
		TaskID:     loopTask,
		Binding:    evidence.BindingRef{Project: loopProject, Verification: "verif-r1", BindingID: lpHoldoutBinding},
		Acceptance: loopAcceptance,
		Anchor:     "cs-anchor",
		Channel:    evidence.ChannelValidationLog,
		Reason:     "why",
		ExposedAt:  fixedNow,
	}
	if _, err := sink.RecordExposure(context.Background(), spec); err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}
	if len(appender.records) != 1 {
		t.Fatalf("记录 = %d 条, 期望 1", len(appender.records))
	}
	rec := appender.records[0]
	if rec.Task != loopTask || rec.ContractRevision != 1 ||
		rec.Binding != (evidence.BindingRef{Project: loopProject, Verification: "verif-r1", BindingID: lpHoldoutBinding}) ||
		rec.Acceptance != evidence.AcceptanceID(loopAcceptance) ||
		rec.ChangeSet != evidence.ChangeSetID("cs-anchor") ||
		rec.Channel != evidence.ChannelValidationLog || rec.Reason != "why" || !rec.ExposedAt.Equal(fixedNow) {
		t.Errorf("映射不透传:%+v", rec)
	}
}
