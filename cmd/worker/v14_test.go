package main

// P2-09 的闭环路径接线测试。
//
// 覆盖任务验收:
//   - AC1 v1.4 job 端到端执行闭环:首过 Accepted(真 git 仓库 + 真一次性
//     worktree + 真 candidateverify 服务,`go version` 作 visible binding 命令,
//     全程离线确定性);FAIL→repairable→修复→Accepted(安全反馈进入修复编码
//     上下文、冻结前候选物化为修复基、reason=rework 取代事实);预算耗尽显式
//     升级;内层耗尽/提供方失败分别落 no_candidate / infrastructure 升级;
//   - AC2 v1.3 路径保持可调用:v13 kind 与 legacy kind 不被 v14 覆盖
//     (v13 行为本体由 v13_test.go 原样覆盖);
//   - AC3 无 runtime 名字分支:闭环按注入 runtime 的 Descriptor 归因与
//     能力协商执行,配置字符串只进配置映射面(workerRuntimes 注册表,P4-01);
//   - AC4 租约/回收语义不动:本任务不改 internal/jobs 队列(其租约测试
//     原样跑 green);v14 handler 只是多注册一个 kind;
//   - AC5 job 取消传播到闭环 Engine:验证中取消 → Cancelled 终态,
//     不伪造 verdict,一次性 worktree 照常清理。
//
// 数据库层测试与 v13 同一隔离策略:每测试独立 schema、自跑迁移、cleanup
// DROP;PostgreSQL 不可达即 SKIP(绝不记 PASS)。

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/evidence"
	"tally/internal/exec"
	"tally/internal/intentevidence"
	"tally/internal/jobs"
	"tally/internal/risk"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/runtimes/fake"
)

// ---- 固件:脚本化 runtime(每个会话首 Execute 按脚本写入/观察) ----

// v14ScriptStep 是一个会话首次 Execute 的脚本:inspect 在写入前观察一次性
// worktree 与指令(修复基物化断言必须发生在当下 —— worktree 随尝试结束销毁);
// 非空 content 写入会话 workspace;非空 err 模拟提供方失败。
type v14ScriptStep struct {
	content  string
	err      error
	inspect  func(ws, instruction string)
	finished chan<- struct{} // 非空:写入后关闭(外部观测执行已发生)
}

// v14ScriptedRuntime 包装 fake runtime:每个 Start 产出一个会话,每个会话的
// 每次 Execute 弹出一条脚本;指令与 workspace 留观。
type v14ScriptedRuntime struct {
	inner agentruntime.Runtime

	mu     sync.Mutex
	steps  []v14ScriptStep
	instrs []string
}

func newV14ScriptedRuntime(steps ...v14ScriptStep) *v14ScriptedRuntime {
	return &v14ScriptedRuntime{inner: fake.New(fake.Config{}), steps: steps}
}

func (r *v14ScriptedRuntime) Descriptor() agentruntime.RuntimeDescriptor { return r.inner.Descriptor() }

func (r *v14ScriptedRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := r.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v14ScriptedSession{rt: r, inner: s, ws: req.Workspace}, nil
}

// instructions 返回已观测到的迭代指令(按 Execute 顺序)。
func (r *v14ScriptedRuntime) instructions() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.instrs...)
}

type v14ScriptedSession struct {
	rt    *v14ScriptedRuntime
	inner agentruntime.Session
	ws    string
}

func (s *v14ScriptedSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	s.rt.mu.Lock()
	if len(s.rt.steps) == 0 {
		s.rt.mu.Unlock()
		return agentruntime.AgentRunResult{}, errors.New("v14ScriptedRuntime: 脚本耗尽")
	}
	step := s.rt.steps[0]
	s.rt.steps = s.rt.steps[1:]
	s.rt.instrs = append(s.rt.instrs, req.Instruction)
	s.rt.mu.Unlock()

	if step.inspect != nil {
		step.inspect(s.ws, req.Instruction)
	}
	if step.err != nil {
		return agentruntime.AgentRunResult{}, step.err
	}
	if step.content != "" {
		p := filepath.Join(s.ws, "internal", "widget", "greeting.go")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
		if err := os.WriteFile(p, []byte(step.content), 0o644); err != nil {
			return agentruntime.AgentRunResult{}, err
		}
	}
	if step.finished != nil {
		close(step.finished)
	}
	return s.inner.Execute(ctx, req)
}

func (s *v14ScriptedSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *v14ScriptedSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *v14ScriptedSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *v14ScriptedSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

// ---- 固件:脚本化候选验证服务(closedloop.CandidateVerification 缝) ----

type v14VerifyStep struct {
	status   verification.FinalStatus
	runID    int64
	bindings []verification.BindingOutcome
	// functional 非空:FinalOutcome 携带该功能覆盖报告(P1-06 验收守卫的
	// 事实面)。status=FinalVerified 且 functional 为空时,替身按生产编排器
	// 的同一判定规则合成完整覆盖报告(orchestrator:FinalVerified 蕴含
	// Functional.Complete,每个 acceptance 由其 binding 满足)—— fake 必须
	// 与真实 FinalOutcome 的契约同形,验收守卫才能据实评估。
	functional *verification.FunctionalCoverageReport
	err        error
	cancel     context.CancelFunc // 非空:调用时先取消 ctx(取消的验证产出不构成 verdict)
}

type v14FakeVerify struct {
	mu    sync.Mutex
	steps []v14VerifyStep
	calls []candidateverify.Request
}

func (v *v14FakeVerify) Verify(ctx context.Context, req candidateverify.Request) (*verification.FinalOutcome, error) {
	v.mu.Lock()
	v.calls = append(v.calls, req)
	if len(v.steps) == 0 {
		v.mu.Unlock()
		return nil, errors.New("v14FakeVerify: 脚本耗尽")
	}
	step := v.steps[0]
	v.steps = v.steps[1:]
	v.mu.Unlock()

	if step.cancel != nil {
		step.cancel()
	}
	if step.err != nil {
		return nil, step.err
	}
	out := &verification.FinalOutcome{
		RunID:            step.runID,
		TaskID:           req.Contract.TaskID(),
		ContractRevision: req.Contract.Revision(),
		ChangeSetID:      req.ChangeSet.ID(),
		Status:           step.status,
		Bindings:         step.bindings,
	}
	switch {
	case step.functional != nil:
		out.Functional = *step.functional
	case step.status == verification.FinalVerified:
		out.Functional = v14CompleteFunctional(req.Contract)
	}
	return out, nil
}

// v14CompleteFunctional 合成「全绿」功能覆盖报告:每条 acceptance 由其全部
// binding 满足(与真实编排器对 FinalVerified 的判定一致)。
func v14CompleteFunctional(f *verification.Frozen) verification.FunctionalCoverageReport {
	c := f.Contract()
	rep := verification.FunctionalCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances:      make([]verification.AcceptanceCoverage, 0, len(c.Acceptances)),
		Complete:         true,
	}
	for _, acc := range c.Acceptances {
		ac := verification.AcceptanceCoverage{AcceptanceID: acc.ID, Covered: true}
		for _, b := range acc.EvidenceBindings {
			ac.SatisfiedBy = append(ac.SatisfiedBy, b.ID)
		}
		rep.Acceptances = append(rep.Acceptances, ac)
	}
	return rep
}

// v14FailedVisible 是一条失败 visible binding 观察(触发 repairable 分类)。
func v14FailedVisible() []verification.BindingOutcome {
	return []verification.BindingOutcome{{
		BindingID:    "VIS-1",
		AcceptanceID: "AC-01",
		Status:       verification.EvidenceStatusFail,
	}}
}

// ---- 固件:内存取代事实出口 ----

type v14MemLineage struct {
	mu    sync.Mutex
	facts []changeset.SupersededFact
}

func (l *v14MemLineage) RecordSuperseded(_ context.Context, f changeset.SupersededFact) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.facts = append(l.facts, f)
	return nil
}

// v14ConfigFor 是 v14 配置的测试缺省(共享面沿用 v13 fixture,真仓库)。
func v14ConfigFor(t *testing.T, repo string) v14Config {
	t.Helper()
	return v14Config{v13: v13ConfigFor(t, repo)}
}

// ---- P1-04 fixtures:同源执行 bundle 仓库(冻结意图 + 策略快照) ----

// v14IntentArtifact 渲染一份合法 FrozenIntent 制品字节(Freeze → Render,与
// 生产冻结产物同形态,与 artifacts 包测试同一构造路径)。
func v14IntentArtifact(t *testing.T, taskID string) string {
	t.Helper()
	fi, err := intentevidence.Freeze(intentevidence.Facts{
		TaskID:    taskID,
		TaskType:  intentevidence.TaskTypeBugFix,
		SourceRef: "v14 bundle fixture",
	})
	if err != nil {
		t.Fatalf("冻结意图夹具 %s: %v", taskID, err)
	}
	b, err := fi.Render()
	if err != nil {
		t.Fatalf("渲染意图夹具 %s: %v", taskID, err)
	}
	return string(b)
}

// newV14Repo 在 v13 fixture(TASK-1043 合同 + 冻结验证契约)之上补冻结意图
// 制品:v1.4.1 worker 从同源 bundle(P1-01 LoadBundle)取得执行权威,四类
// 制品缺一不可 —— v14 路径的仓库 fixture 必须是 bundle 完整的。
func newV14Repo(t *testing.T) string {
	t.Helper()
	repo := newV13Repo(t)
	v13WriteRepoFile(t, repo, ".tally/intents/TASK-1043.yaml", v14IntentArtifact(t, "TASK-1043"))
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: 冻结意图制品")
	return repo
}

// newV14HeldRepo:held-out fixture(TASK-1050)+ 冻结意图。
func newV14HeldRepo(t *testing.T) string {
	t.Helper()
	repo := newV13HeldRepo(t)
	v13WriteRepoFile(t, repo, ".tally/intents/TASK-1050.yaml", v14IntentArtifact(t, "TASK-1050"))
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: 冻结意图制品(held-out)")
	return repo
}

// newV14RejectedPolicyRepo:含一份解析失败的策略制品 —— 策略快照 Rejected
// 非空,CompileAuthority 的 fail-safe 裁决(拒收策略阻断编译)必须生效。
func newV14RejectedPolicyRepo(t *testing.T) string {
	t.Helper()
	repo := newV14Repo(t)
	v13WriteRepoFile(t, repo, ".tally/policies/broken.yaml", "")
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: 坏策略制品")
	return repo
}

// v14StartSpy 统计 runtime.Start 调用(P1-04 AC 的 spy:编译成功之前不得有
// 任何会话开启 —— 失败路径零次 Start 是负面证明,成功路径恰好一次)。
type v14StartSpy struct {
	inner  agentruntime.Runtime
	mu     sync.Mutex
	starts int
}

func newV14StartSpy(inner agentruntime.Runtime) *v14StartSpy { return &v14StartSpy{inner: inner} }

func (s *v14StartSpy) Descriptor() agentruntime.RuntimeDescriptor { return s.inner.Descriptor() }

func (s *v14StartSpy) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s.mu.Lock()
	s.starts++
	s.mu.Unlock()
	return s.inner.Start(ctx, req)
}

func (s *v14StartSpy) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.starts
}

// v14AssertRootEmpty 断言一次性根已清空(尝试结束即丢弃沙箱,验证 workspace
// 由 candidateverify 自清)。
func v14AssertRootEmpty(t *testing.T, root string) {
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

// ---- AC1:首过 Accepted(真 candidateverify + 真一次性 worktree) ----

func TestV14ClosedLoopFirstPassAccepted(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	vs := &v13MemVerifySink{}

	// 验证腿用生产组成(真 candidateverify 服务):visible binding 跑
	// `go version`(离线、确定性 exit 0)→ verified;事实落内存 sink。
	verify := candidateverify.New(candidateverify.Config{
		Repo:         repo,
		WorktreeRoot: cfg.v13.WorktreeRoot,
		EnvName:      cfg.v13.EnvName,
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "v14-test"},
	}, candidateverify.Deps{Sink: vs})

	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})
	res, err := runV14Task(context.Background(), v14Deps{
		cfg:     cfg,
		runtime: rt,
		wfSink:  wf,
		verify:  verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}

	// 闭环终态:Accepted 且必带 basis(每个 VERIFIED-like 结果携带依据)。
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%v, 期望 accepted + basis", res.Phase, res.Basis)
	}
	if res.Basis.Attempt != 1 || res.Basis.VerifyRun == "" {
		t.Fatalf("Basis = %+v, 期望尝试 1 + 验证运行引用", res.Basis)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].Supersedes != "" {
		t.Fatalf("尝试谱系 = %+v, 期望恰好一次无取代尝试(首过零修复)", res.Attempts)
	}
	if res.Attempts[0].RunID != closedloop.RunIDFor("TASK-1043", 1) {
		t.Errorf("尝试 RunID = %s, 期望 attempt 幂等键 %s", res.Attempts[0].RunID, closedloop.RunIDFor("TASK-1043", 1))
	}

	// codegen 事实:一次全新 CodegenRun(attempt 推导身份)→ 一个不可变候选。
	if len(wf.begins) != 1 || len(wf.progresses) != 1 || len(wf.finishes) != 1 || len(wf.changeSets) != 1 {
		t.Fatalf("codegen 事实 = %d/%d/%d/%d, 期望 1/1/1/1",
			len(wf.begins), len(wf.progresses), len(wf.finishes), len(wf.changeSets))
	}
	b := wf.begins[0]
	if b.ID != workflow.RunID("TASK-1043/attempt/001") || b.EscalatedFrom != "" {
		t.Errorf("RunBegin 身份 = %s(父 %s), 期望 attempt 幂等键、无父链接", b.ID, b.EscalatedFrom)
	}
	if b.WorkspaceID == "" || b.BaseRevision == "" ||
		!strings.HasPrefix(b.VerificationRevision, "sha256:") ||
		!strings.HasPrefix(b.ContextPackHash, "sha256:") ||
		len(b.CapabilitySnapshot) == 0 {
		t.Errorf("RunBegin 观测坐标不完整: %+v", b)
	}
	if wf.finishes[0].Status != runstore.RunCompleted || wf.finishes[0].ResultPatchHash == "" {
		t.Errorf("RunFinish = %+v, 期望 completed + 候选哈希", wf.finishes[0])
	}
	if wf.changeSets[0].ID() != res.Basis.ChangeSet {
		t.Errorf("候选身份漂移: sink=%s basis=%s", wf.changeSets[0].ID(), res.Basis.ChangeSet)
	}
	if res.Basis.VerifyRun != "1" {
		t.Errorf("VerifyRun = %s, 期望验证 sink 首行引用 1", res.Basis.VerifyRun)
	}

	// 验证事实:真服务跑过 visible binding → pass。
	if len(vs.runs) != 1 || vs.runs[0].Status != verification.RunStatusPass {
		t.Fatalf("VerificationRun = %+v, 期望 pass", vs.runs)
	}

	// AC4/INV-EXEC-01:owner 主 worktree 零污染;一次性根全部回收。
	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// ---- AC1:FAIL → repairable → 修复(新 Run/新候选/取代/复验)→ Accepted ----

func TestV14ClosedLoopRepairThenAccepted(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	lineage := &v14MemLineage{}
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalFailed, runID: 100, bindings: v14FailedVisible()},
		{status: verification.FinalVerified, runID: 101},
	}}

	const first, reworked = "package widget\n", "package widget\n\n// reworked greeting\n"
	var sawMaterialized string
	rt := newV14ScriptedRuntime(
		v14ScriptStep{content: first},
		v14ScriptStep{content: reworked, inspect: func(ws, instruction string) {
			// 修复尝试的 worktree 里,冻结前候选已被确定性物化(写新内容前观察)。
			b, err := os.ReadFile(filepath.Join(ws, "internal", "widget", "greeting.go"))
			if err != nil {
				t.Errorf("修复基未物化: %v", err)
				return
			}
			sawMaterialized = string(b)
			if !strings.Contains(instruction, "Repair context") {
				t.Errorf("修复迭代指令缺安全反馈段: %q", instruction)
			}
		}},
	)

	res, err := runV14Task(context.Background(), v14Deps{
		cfg:     cfg,
		runtime: rt,
		wfSink:  wf,
		verify:  verify,
		lineage: lineage,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}

	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil || res.Basis.Attempt != 2 {
		t.Fatalf("终态 = %s basis=%+v, 期望 accepted@attempt2", res.Phase, res.Basis)
	}
	if res.Basis.VerifyRun != "101" {
		t.Errorf("Basis.VerifyRun = %s, 期望复验运行 101", res.Basis.VerifyRun)
	}
	if len(res.Attempts) != 2 {
		t.Fatalf("尝试 = %d, 期望 2", len(res.Attempts))
	}
	a1, a2 := res.Attempts[0], res.Attempts[1]
	if a2.ParentRun != a1.RunID || a2.Supersedes != a1.ChangeSet {
		t.Errorf("修复尝试 = %+v, 期望父链接首次尝试并取代其候选", a2)
	}

	// 取代事实:reason=rework,追加一次。
	if len(lineage.facts) != 1 {
		t.Fatalf("取代事实 = %d 条, 期望 1", len(lineage.facts))
	}
	if fact := lineage.facts[0]; fact.Old != a1.ChangeSet || fact.New != a2.ChangeSet || fact.Reason != changeset.SupersedeRework {
		t.Errorf("取代事实 = %+v, 期望 a1→a2 rework", fact)
	}

	// 修复基物化 = 冻结前候选内容逐字。
	if sawMaterialized != first {
		t.Errorf("修复基内容 = %q, 期望前候选 %q 逐字一致", sawMaterialized, first)
	}

	// codegen 事实:两次全新 Run,身份 = attempt 幂等键,父链接经谱系槽位。
	if len(wf.begins) != 2 || len(wf.changeSets) != 2 {
		t.Fatalf("codegen 事实 = begins %d / changeSets %d, 期望 2/2", len(wf.begins), len(wf.changeSets))
	}
	if wf.begins[0].ID != workflow.RunID("TASK-1043/attempt/001") ||
		wf.begins[1].ID != workflow.RunID("TASK-1043/attempt/002") ||
		wf.begins[1].EscalatedFrom != workflow.RunID(a1.RunID) {
		t.Errorf("Run 身份/父链接 = %s←%s / %s←%s",
			wf.begins[0].ID, wf.begins[0].EscalatedFrom, wf.begins[1].ID, wf.begins[1].EscalatedFrom)
	}
	if wf.changeSets[0].ID() == wf.changeSets[1].ID() {
		t.Error("修复候选复用了前候选身份(候选必须全新)")
	}

	// 安全反馈边界:失败材料只经 P2-04 投影进入修复编码上下文。
	instrs := rt.instructions()
	if len(instrs) != 2 {
		t.Fatalf("迭代指令 = %d 条, 期望 2", len(instrs))
	}
	if strings.Contains(instrs[0], "Repair context") {
		t.Errorf("首次尝试不应携带修复反馈: %q", instrs[0])
	}
	for _, want := range []string{"AC-01", "widget greets.", "repairable", "VIS-1", "diagnostic summary"} {
		if !strings.Contains(instrs[1], want) {
			t.Errorf("修复反馈缺 allowlist 材料 %q: %q", want, instrs[1])
		}
	}

	// 复验:绑定修复候选,同一冻结契约(pin 逐字不变)。
	if len(verify.calls) != 2 {
		t.Fatalf("验证调用 = %d, 期望 2", len(verify.calls))
	}
	if verify.calls[0].Contract.Hash() != verify.calls[1].Contract.Hash() {
		t.Error("复验契约 pin 漂移(修复绝不放宽冻结契约)")
	}
	if verify.calls[1].ChangeSet.ID() != a2.ChangeSet {
		t.Errorf("复验对象 = %s, 期望修复候选 %s", verify.calls[1].ChangeSet.ID(), a2.ChangeSet)
	}

	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// ---- AC1:预算耗尽显式升级(缺省 RepairPolicy = 2 次修复,决策锁 23) ----

func TestV14ClosedLoopRepairExhausted(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	lineage := &v14MemLineage{}
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalFailed, runID: 1, bindings: v14FailedVisible()},
		{status: verification.FinalFailed, runID: 2, bindings: v14FailedVisible()},
		{status: verification.FinalFailed, runID: 3, bindings: v14FailedVisible()},
	}}
	rt := newV14ScriptedRuntime(
		v14ScriptStep{content: "c1\n"},
		v14ScriptStep{content: "c2\n"},
		v14ScriptStep{content: "c3\n"},
	)

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify, lineage: lineage,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil {
		t.Fatalf("终态 = %s, 期望 escalated", res.Phase)
	}
	if res.Escalation.Reason != closedloop.EscalationRepairExhausted {
		t.Errorf("升级原因 = %s, 期望 repair_exhausted", res.Escalation.Reason)
	}
	// 尝试 1 + 修复 2 次(第 3 次修复被预算拒绝),取代事实恰好 2 条。
	if len(res.Attempts) != 3 || len(wf.changeSets) != 3 {
		t.Errorf("尝试/候选 = %d/%d, 期望 3/3", len(res.Attempts), len(wf.changeSets))
	}
	if len(lineage.facts) != 2 {
		t.Errorf("取代事实 = %d 条, 期望 2", len(lineage.facts))
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// 内层耗尽(无可固定候选)→ no_candidate 升级,不产生候选事实。
func TestV14ClosedLoopNoCandidateEscalates(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	verify := &v14FakeVerify{}
	// 内层预算(缺省 ExecutionPolicy)内全部迭代无产出。
	rt := newV14ScriptedRuntime(
		v14ScriptStep{}, v14ScriptStep{}, v14ScriptStep{}, v14ScriptStep{},
	)

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationNoCandidate {
		t.Fatalf("终态 = %s / %+v, 期望 no_candidate 升级", res.Phase, res.Escalation)
	}
	if len(wf.changeSets) != 0 || len(verify.calls) != 0 {
		t.Errorf("无可验证物时不得产生候选/验证: %d/%d", len(wf.changeSets), len(verify.calls))
	}
	if len(wf.finishes) != 1 || wf.finishes[0].Status != runstore.RunFailed ||
		wf.finishes[0].FailureReason != "iterations_exhausted" {
		t.Fatalf("RunFinish = %+v, 期望 failed/iterations_exhausted", wf.finishes)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// 提供方失败(runtime 会话错误)→ 基础设施升级,绝不伪装成候选失败,
// 一次性 worktree 照常清理(ADR-0001 §3)。
func TestV14ClosedLoopProviderFailureEscalates(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	verify := &v14FakeVerify{}
	rt := newV14ScriptedRuntime(v14ScriptStep{err: errors.New("runtime crashed")})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationInfrastructure {
		t.Fatalf("终态 = %s / %+v, 期望 infrastructure_failure 升级", res.Phase, res.Escalation)
	}
	if len(res.Attempts) != 0 || len(wf.changeSets) != 0 || len(verify.calls) != 0 {
		t.Errorf("失败尝试不得入谱系/触发验证: %d/%d/%d",
			len(res.Attempts), len(wf.changeSets), len(verify.calls))
	}
	if len(wf.finishes) != 1 || wf.finishes[0].Status != runstore.RunFailed ||
		wf.finishes[0].FailureReason != "runtime_error" {
		t.Fatalf("RunFinish = %+v, 期望 failed/runtime_error", wf.finishes)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// ---- AC5:取消传播到闭环 Engine,不伪造 verdict ----

func TestV14ClosedLoopCancellationPropagates(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 验证中取消:即使验证器返回 verified,已取消的产出也不得构成 verdict。
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalVerified, runID: 5, cancel: cancel},
	}}
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(ctx, v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseCancelled {
		t.Fatalf("终态 = %s, 期望 cancelled", res.Phase)
	}
	if res.Basis != nil || res.Escalation != nil {
		t.Errorf("取消不得产生 basis/升级: %+v / %+v", res.Basis, res.Escalation)
	}
	if len(verify.calls) != 1 {
		t.Errorf("取消后不得复验,验证调用 = %d", len(verify.calls))
	}
	// 已完成尝试保留为事实;一次性 worktree 照常清理。
	if len(res.Attempts) != 1 || len(wf.changeSets) != 1 {
		t.Errorf("已完成尝试 = %d(候选 %d), 期望保留为事实", len(res.Attempts), len(wf.changeSets))
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// 启动前已取消:零尝试、零 verdict、零沙箱。
func TestV14ClosedLoopPreCancelled(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rt := newV14ScriptedRuntime()

	res, err := runV14Task(ctx, v14Deps{
		cfg:     cfg,
		runtime: rt,
		wfSink:  &v13MemRunSink{},
		verify:  &v14FakeVerify{},
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseCancelled || len(res.Attempts) != 0 {
		t.Fatalf("终态 = %s attempts=%d, 期望 cancelled/0", res.Phase, len(res.Attempts))
	}
	if len(rt.instructions()) != 0 {
		t.Errorf("取消后不得执行任何迭代: %q", rt.instructions())
	}
}

// ---- AC3:runtime 归因来自注入 runtime 的 Descriptor,不按配置名字分支 ----

func TestV14RuntimeAttributionIsObservedDescriptor(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	cfg.v13.Runtime = "codex" // 配置写 codex,注入面给 fake:闭环不得按名字分支
	wf := &v13MemRunSink{}
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalVerified, runID: 9},
	}}
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("配置 runtime 名不得改变注入 runtime 的执行: %v", err)
	}
	if res.Phase != closedloop.PhaseAccepted {
		t.Fatalf("终态 = %s, 期望 accepted", res.Phase)
	}
	if wf.begins[0].RuntimeName != "fake" {
		t.Errorf("RunBegin.RuntimeName = %s, 期望注入 runtime 的 Descriptor 名", wf.begins[0].RuntimeName)
	}
	if verify.calls[0].Runtime != "fake" {
		t.Errorf("验证归因 runtime = %s, 期望实际执行后端身份", verify.calls[0].Runtime)
	}
}

// ---- AC2:v13 与 legacy kind 保持注册,v14 是新增独立 kind ----

func TestV14KindDistinctFromV13AndLegacy(t *testing.T) {
	legacy := map[jobs.Kind]bool{}
	for kind := range exec.Handlers(exec.Config{}) {
		legacy[kind] = true
	}
	if KindV14Task == KindV13Task || legacy[KindV14Task] {
		t.Fatalf("v14 kind %s 不得覆盖 v13/legacy kind", KindV14Task)
	}
	for _, k := range []jobs.Kind{
		jobs.KindEnvironmentUp, jobs.KindEnvironmentDown, jobs.KindEnvironmentRebuild,
		jobs.KindDeploy, jobs.KindDeployRollback, jobs.KindDiagnose,
	} {
		if !legacy[k] {
			t.Errorf("legacy job %s 不在注册表(AC2)", k)
		}
	}
}

// ---- handler 入队面(payload 校验) ----

func TestV14HandlerPayloadValidation(t *testing.T) {
	_, h := v14Handler(v14Config{}, nil)
	if err := h(context.Background(), json.RawMessage(`not-json`)); err == nil {
		t.Fatal("坏 JSON 应报错")
	}
	if err := h(context.Background(), json.RawMessage(`{"task":"  "}`)); err == nil {
		t.Fatal("缺 task 应报错")
	}
}

// ---- 配置面:修复预算来自配置(缺省 2,越界拒绝) ----

func TestV14RepairPolicyFromConfig(t *testing.T) {
	def, err := v14RepairPolicy(0)
	if err != nil || def.MaxRepairAttempts != closedloop.DefaultMaxRepairAttempts {
		t.Fatalf("缺省修复预算 = %d/%v, 期望 %d", def.MaxRepairAttempts, err, closedloop.DefaultMaxRepairAttempts)
	}
	if p, err := v14RepairPolicy(3); err != nil || p.MaxRepairAttempts != 3 {
		t.Fatalf("显式修复预算 = %d/%v, 期望 3", p.MaxRepairAttempts, err)
	}
	for _, bad := range []int{-1, closedloop.HardMaxRepairAttempts + 1} {
		if _, err := v14RepairPolicy(bad); !errors.Is(err, closedloop.ErrInvalidRepairPolicy) {
			t.Errorf("非法预算 %d 应拒绝构造,得到 %v", bad, err)
		}
	}

	t.Setenv("TALLY_V14_MAX_REPAIR", "3")
	if cfg := v14ConfigFromEnv("repo"); cfg.MaxRepair != 3 {
		t.Errorf("env 覆盖 = %d, 期望 3", cfg.MaxRepair)
	}
	t.Setenv("TALLY_V14_MAX_REPAIR", "bogus")
	if cfg := v14ConfigFromEnv("repo"); cfg.MaxRepair != 0 {
		t.Errorf("非法 env 应落缺省 0,得到 %d", cfg.MaxRepair)
	}
	if cfg := v14ConfigFromEnv("repo"); !strings.Contains(cfg.v13.WorktreeRoot, "tally-v14-worktrees") {
		t.Errorf("v14 缺省一次性根 = %s", cfg.v13.WorktreeRoot)
	}
}

// ---- 修复指令渲染:只含 allowlist 投影 ----

func TestV14InstructionRendersSafeFeedbackOnly(t *testing.T) {
	task := mustV14Task(t)
	fb, err := closedloop.NewFeedbackBuilder("TASK-1043", 1, changeset.ChangeSetID("cs-1"), "100").
		ForAcceptance("AC-01", "widget greets.").
		WithFailure(closedloop.FailureRepairable).
		WithDiagnosticSummary("candidate cs-1 rejected; visible evidence failed: VIS-1").
		AddVisibleEvidence(closedloop.VisibleEvidence{
			AcceptanceID: "AC-01", BindingID: "VIS-1",
			EvidenceType: verification.EvidenceTest, Path: "internal/widget/widget_test.go",
			Status: verification.EvidenceStatusFail,
		}).
		Build()
	if err != nil {
		t.Fatalf("构造反馈: %v", err)
	}
	out := v14Instruction(task, 2, &fb, nil)
	for _, want := range []string{"Repair context", "AC-01", "widget greets.", "repairable", "VIS-1", "attempt/001"} {
		if !strings.Contains(out, want) {
			t.Errorf("修复指令缺 %q:\n%s", want, out)
		}
	}
	// 非修复重试语义保留;修复语义不与重试文案混叠。
	if !strings.Contains(out, "repair attempt 2") {
		t.Errorf("修复指令缺修复说明:\n%s", out)
	}
	plain := v14Instruction(task, 2, nil, nil)
	if strings.Contains(plain, "Repair context") || !strings.Contains(plain, "retry iteration 2") {
		t.Errorf("非修复指令不应携带反馈段:\n%s", plain)
	}
	// 零值 requires 不渲染 Prevention 段(P1-05:零策略/未命中时指令形状不变)。
	if strings.Contains(plain, "Prevention requirements") {
		t.Errorf("零值 requires 不应渲染防御上下文段:\n%s", plain)
	}
}

func mustV14Task(t *testing.T) taskcontract.TaskContract {
	t.Helper()
	repo := newV14Repo(t)
	snap, err := artifacts.Load(repo, "HEAD")
	if err != nil {
		t.Fatalf("装载制品: %v", err)
	}
	task, ok := findV13Task(snap, "TASK-1043")
	if !ok {
		t.Fatal("fixture 缺 TASK-1043")
	}
	return task.Contract
}

// ---- held-out 前提强制(与 v13 同一合同,经 v14 入口) ----

func TestV14HeldOutPrerequisitesEnforced(t *testing.T) {
	repo := newV14HeldRepo(t)
	cfg := v14ConfigFor(t, repo)
	rt := newV14ScriptedRuntime()
	if _, err := runV14Task(context.Background(), v14Deps{
		cfg:     cfg,
		runtime: rt,
		wfSink:  &v13MemRunSink{},
		verify:  &v14FakeVerify{},
	}, "TASK-1050"); err == nil || !strings.Contains(err.Error(), "held_out") {
		t.Fatalf("holdout 通道未接线应拒绝执行: %v", err)
	}
	if len(rt.instructions()) != 0 {
		t.Errorf("拒绝执行后不得开跑任何迭代: %q", rt.instructions())
	}
}

// ---- P1-04:合同编译权威前置门(PRD §16 no-bypass 成为运行时事实) ----

// AC:runtime start spy 证明编译成功之前不会有任何会话开启。负面证明:
// 拒收策略 ⇒ CompileAuthority fail-safe 阻断(类型化 ErrPolicyRejections),
// 零会话、零 codegen 事实、零一次性 worktree —— 编译权威失败是执行前置
// 失败,不是候选失败(不进闭环、不产 verdict、不耗修复预算);bundle 不完整
// (缺 intent)同理被装载哨兵先行阻断。
func TestV14CompileFailureBlocksRuntimeStart(t *testing.T) {
	t.Run("拒收策略阻断编译", func(t *testing.T) {
		repo := newV14RejectedPolicyRepo(t)
		cfg := v14ConfigFor(t, repo)
		spy := newV14StartSpy(newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"}))
		wf := &v13MemRunSink{}

		_, err := runV14Task(context.Background(), v14Deps{
			cfg: cfg, runtime: spy, wfSink: wf, verify: &v14FakeVerify{},
		}, "TASK-1043")
		if !errors.Is(err, artifacts.ErrPolicyRejections) {
			t.Fatalf("拒收策略应类型化阻断(ErrPolicyRejections), got %v", err)
		}
		if n := spy.count(); n != 0 {
			t.Errorf("编译失败后不得开启任何 runtime 会话, starts=%d", n)
		}
		if len(wf.begins) != 0 || len(wf.finishes) != 0 || len(wf.changeSets) != 0 {
			t.Errorf("编译失败不得产生 codegen 事实: begins=%d finishes=%d changeSets=%d",
				len(wf.begins), len(wf.finishes), len(wf.changeSets))
		}
		v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
	})

	t.Run("缺意图制品阻断装载", func(t *testing.T) {
		repo := newV13Repo(t) // 无 .tally/intents ⇒ bundle 不完整
		cfg := v14ConfigFor(t, repo)
		spy := newV14StartSpy(newV14ScriptedRuntime())

		if _, err := runV14Task(context.Background(), v14Deps{
			cfg: cfg, runtime: spy, wfSink: &v13MemRunSink{}, verify: &v14FakeVerify{},
		}, "TASK-1043"); !errors.Is(err, artifacts.ErrIntentNotFound) {
			t.Fatalf("缺 intent 应类型化拒收(ErrIntentNotFound), got %v", err)
		}
		if n := spy.count(); n != 0 {
			t.Errorf("装载失败后不得开启任何 runtime 会话, starts=%d", n)
		}
	})
}

// 编译成功 ⇒ 恰好一次会话开启,且运行事实坐标全部钉在同一 bundle 的 pinned
// 输入上:task/verification 修订与 base 提交与「重装载 + 重编译」的确定性
// 复算逐项一致 —— bundle 身份与编译哈希正是这些坐标的纯函数,可独立复算
// (P1-03 确定性在 worker 事实面的落点)。
func TestV14RunFactsPinCompiledAuthority(t *testing.T) {
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalVerified, runID: 7},
	}}
	spy := newV14StartSpy(newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"}))

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: spy, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseAccepted {
		t.Fatalf("终态 = %s, 期望 accepted", res.Phase)
	}
	if n := spy.count(); n != 1 {
		t.Errorf("编译成功后恰好一次会话开启, starts=%d", n)
	}
	if len(wf.begins) != 1 {
		t.Fatalf("codegen 事实 = %d 条, 期望 1", len(wf.begins))
	}

	// 同一提交重装载重编译:同一权威确定性复现。
	b2, err := artifacts.LoadBundle(repo, "HEAD", "TASK-1043")
	if err != nil {
		t.Fatalf("重装载 bundle: %v", err)
	}
	a2, err := b2.CompileAuthority()
	if err != nil {
		t.Fatalf("重编译权威: %v", err)
	}
	b := wf.begins[0]
	if b.TaskID != b2.TaskID ||
		b.TaskRevision != b2.Task.Contract.Hash ||
		b.VerificationRevision != b2.Verification.Frozen.Hash() ||
		b.BaseRevision != b2.SourceCommit {
		t.Errorf("运行事实坐标漂离编译权威输入: (task %s/%s, taskrev %s/%s, verif %s/%s, base %s/%s)",
			b.TaskID, b2.TaskID,
			b.TaskRevision, b2.Task.Contract.Hash,
			b.VerificationRevision, b2.Verification.Frozen.Hash(),
			b.BaseRevision, b2.SourceCommit)
	}
	if a2.BundleIdentity == "" || a2.Compiled.Hash() == "" {
		t.Errorf("编译身份不完整: bundle=%q compiled=%q", a2.BundleIdentity, a2.Compiled.Hash())
	}
	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// ---- P1-05:防御上下文投影(Compiled.Requires → 运行时上下文 + compiled-aware context_hash)----

// v14PolicyYAML 一份命中 schema_migration 信号的 gate 策略(与 artifacts 包
// 测试夹具同形;engineeringpolicy.Parse + Validate 通过,gate_condition ⊆
// requires)。
const v14PolicyYAML = `id: db_gate
enforcement: gate
rationale: 迁移类任务必须证明全新库升级路径与数据完整性
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
  - data_integrity
gate_condition:
  all_of:
    - fresh_database
    - data_integrity
`

// v14PolicyYAMLV2 同一策略 id 换版后的内容(rationale 不同 → 内容修订不同):
// 权威换版敏感性的驱动输入。
const v14PolicyYAMLV2 = `id: db_gate
enforcement: gate
rationale: 换版后的迁移门禁理由
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
  - data_integrity
gate_condition:
  all_of:
    - fresh_database
    - data_integrity
`

// v14IntentArtifactWithHints 渲染一份带确认风险提示的 FrozenIntent 制品
// (提示走 known 信号词表 → DeriveRisk 产出确定性 detected 信号)。
func v14IntentArtifactWithHints(t *testing.T, taskID string, hints ...string) string {
	t.Helper()
	fi, err := intentevidence.Freeze(intentevidence.Facts{
		TaskID:    taskID,
		TaskType:  intentevidence.TaskTypeBugFix,
		SourceRef: "v14 bundle fixture",
		RiskHints: hints,
	})
	if err != nil {
		t.Fatalf("冻结意图夹具 %s: %v", taskID, err)
	}
	b, err := fi.Render()
	if err != nil {
		t.Fatalf("渲染意图夹具 %s: %v", taskID, err)
	}
	return string(b)
}

// newV14PolicyRepo:TASK-1043 + 确认提示(schema_migration)+ 命中该信号的
// gate 策略 —— CompileAuthority 的 Requires() 非空,防御上下文可端到端观测。
func newV14PolicyRepo(t *testing.T) string {
	t.Helper()
	repo := newV13Repo(t)
	v13WriteRepoFile(t, repo, ".tally/intents/TASK-1043.yaml", v14IntentArtifactWithHints(t, "TASK-1043", "schema_migration"))
	v13WriteRepoFile(t, repo, ".tally/policies/db_gate.yaml", v14PolicyYAML)
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: 命中策略 + 风险提示")
	return repo
}

// newV14HeldPolicyRepo:held-out 契约(TASK-1050)+ 确认提示 + 命中策略 ——
// 防御上下文与 held-out 验证材料同场,暴露回归的驱动夹具。
func newV14HeldPolicyRepo(t *testing.T) string {
	t.Helper()
	repo := newV13HeldRepo(t)
	v13WriteRepoFile(t, repo, ".tally/intents/TASK-1050.yaml", v14IntentArtifactWithHints(t, "TASK-1050", "schema_migration"))
	v13WriteRepoFile(t, repo, ".tally/policies/db_gate.yaml", v14PolicyYAML)
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: held-out + 命中策略")
	return repo
}

// P1-05 AC(实现约束到达 runtime + prompt 快照):命中策略的执行前 requires
// 以确定性文案随每轮迭代指令进入编码上下文;运行事实的 ContextPackHash 与
// 「同一提交重装载重编译 + v14ContextPack 复算」逐字节一致 —— 事实坐标就是
// compiled-aware 哈希,且确实因 requires 而偏离无 requires 的组装。
func TestV14PreventionContextReachesRuntime(t *testing.T) {
	repo := newV14PolicyRepo(t)
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalVerified, runID: 3},
	}}
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	// P1-06 语义叠加(2026-09-12):本夹具命中 db_gate ⇒ 必需档上调 L3,而
	// TASK-1043 只有 visible binding(achieved L1)—— 守卫正确地把该候选
	// 升级为 assurance_insufficient(不再普通 Accepted)。本测试的命题是
	// 防御上下文确实到达 runtime 且 context_hash compiled-aware:执行必须
	// 真实发生(指令/事实断言在下方),终态按守卫语义断言。
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望守卫按 L3 要求升级(执行本身已完整发生)", res.Phase, res.Escalation)
	}

	// prompt 快照:每轮迭代携带同一条确定性 Prevention 行(命中策略的验收告知)。
	const wantLine = "- policy db_gate (gate): 最终会按这些条件验收: data_integrity, fresh_database — 理由: 迁移类任务必须证明全新库升级路径与数据完整性"
	instrs := rt.instructions()
	if len(instrs) == 0 {
		t.Fatal("未观测到任何迭代指令")
	}
	for i, in := range instrs {
		if !strings.Contains(in, "Prevention requirements (compiled contract):") || !strings.Contains(in, wantLine) {
			t.Errorf("迭代 %d 指令缺防御上下文快照行:\n%s", i+1, in)
		}
	}

	// 事实面复算:重装载 + 重编译 + v14ContextPack ⇒ 记录的 ContextPackHash。
	b2, err := artifacts.LoadBundle(repo, "HEAD", "TASK-1043")
	if err != nil {
		t.Fatalf("重装载 bundle: %v", err)
	}
	a2, err := b2.CompileAuthority()
	if err != nil {
		t.Fatalf("重编译权威: %v", err)
	}
	if n := len(a2.Compiled.Requires().Items); n != 1 || a2.Compiled.Requires().Items[0].PolicyID != "db_gate" {
		t.Fatalf("夹具应恰好命中 db_gate, got %+v", a2.Compiled.Requires().Items)
	}
	h2, err := v14ContextPack(repo, b2.SourceCommit, b2.Task.Contract, b2.Verification.Frozen, a2.Compiled.Requires())
	if err != nil {
		t.Fatalf("复算 compiled-aware context_hash: %v", err)
	}
	if len(wf.begins) != 1 {
		t.Fatalf("RunBegin = %d 条, 期望 1", len(wf.begins))
	}
	if wf.begins[0].ContextPackHash != h2 {
		t.Errorf("ContextPackHash = %s, 期望 compiled-aware 复算值 %s", wf.begins[0].ContextPackHash, h2)
	}
	// 哈希确实因 requires 而不同(不是候选面的平凡重复)。
	plain, err := v13ContextPack(repo, b2.SourceCommit, b2.Task.Contract, b2.Verification.Frozen)
	if err != nil {
		t.Fatalf("复算无 requires 哈希: %v", err)
	}
	if plain == h2 {
		t.Error("requires 未纳入 context_hash(compiled-aware 落空)")
	}

	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// P1-05 AC(暴露回归)+ 修复/重放不能悄悄换权威:held-out 契约与命中策略同场
// 时,防御上下文进指令、held-out 验证材料(store/ref、binding id)绝不随行;
// 策略换版(同 id、不同 rationale)后重编译,requires 内容变化 ⇒ compiled-aware
// context_hash 变化,而候选面/合同不变时无 requires 的哈希不变 —— 差异恰好
// 来自编译权威,换权威在事实面上不再无声。同一输入重算同哈希(确定性)。
func TestV14PreventionExcludesHeldOutAndBindsHash(t *testing.T) {
	repo := newV14HeldPolicyRepo(t)
	b, err := artifacts.LoadBundle(repo, "HEAD", "TASK-1050")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	a, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority: %v", err)
	}
	req := a.Compiled.Requires()
	if len(req.Items) != 1 || req.Items[0].PolicyID != "db_gate" {
		t.Fatalf("夹具应恰好命中 db_gate, got %+v", req.Items)
	}
	lines := req.Lines()

	// 暴露回归:指令含策略 requires 行,不含任何 held-out 材料线索。
	instr := v14Instruction(b.Task.Contract, 1, nil, lines)
	if !strings.Contains(instr, "Prevention requirements (compiled contract):") ||
		!strings.Contains(instr, "policy db_gate (gate): 最终会按这些条件验收: data_integrity, fresh_database") {
		t.Errorf("指令缺防御上下文:\n%s", instr)
	}
	for _, leaked := range []string{"HOLD-1", "holdout", "hold-1"} {
		if strings.Contains(instr, leaked) {
			t.Errorf("指令泄漏 held-out 材料 %q:\n%s", leaked, instr)
		}
		if joined := strings.Join(lines, "\n"); strings.Contains(joined, leaked) {
			t.Errorf("requires 渲染行泄漏 held-out 材料 %q:\n%s", leaked, joined)
		}
	}

	h1, err := v14ContextPack(repo, b.SourceCommit, b.Task.Contract, b.Verification.Frozen, req)
	if err != nil {
		t.Fatalf("v14ContextPack(v1): %v", err)
	}
	h1again, err := v14ContextPack(repo, b.SourceCommit, b.Task.Contract, b.Verification.Frozen, req)
	if err != nil || h1again != h1 {
		t.Fatalf("同一输入重算哈希不确定: %s vs %s (err=%v)", h1, h1again, err)
	}

	// 策略换版:同 id、不同 rationale 的新提交。合同/候选面不变。
	v13WriteRepoFile(t, repo, ".tally/policies/db_gate.yaml", v14PolicyYAMLV2)
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: 策略换版 v2")
	b2, err := artifacts.LoadBundle(repo, "HEAD", "TASK-1050")
	if err != nil {
		t.Fatalf("LoadBundle(v2): %v", err)
	}
	a2, err := b2.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(v2): %v", err)
	}
	if a2.Compiled.Hash() == a.Compiled.Hash() {
		t.Fatal("策略换版后编译身份未变(夹具失效)")
	}
	h2, err := v14ContextPack(repo, b2.SourceCommit, b2.Task.Contract, b2.Verification.Frozen, a2.Compiled.Requires())
	if err != nil {
		t.Fatalf("v14ContextPack(v2): %v", err)
	}
	if h2 == h1 {
		t.Error("策略换版后 compiled-aware context_hash 未变(换权威仍可无声)")
	}
	// 对照:同一对提交,无 requires 的组装哈希不变 —— 差异恰好来自 requires。
	plain1, err := v13ContextPack(repo, b.SourceCommit, b.Task.Contract, b.Verification.Frozen)
	if err != nil {
		t.Fatalf("v13ContextPack(v1): %v", err)
	}
	plain2, err := v13ContextPack(repo, b2.SourceCommit, b2.Task.Contract, b2.Verification.Frozen)
	if err != nil {
		t.Fatalf("v13ContextPack(v2): %v", err)
	}
	if plain1 != plain2 {
		t.Errorf("候选面/合同未变时无 requires 哈希漂移: %s vs %s", plain1, plain2)
	}
}

// 零值 requires(零策略仓库):v14 组装与 v13 组装同哈希 —— 槽位向后兼容,
// 未命中策略的执行不因 P1-05 接线改变上下文坐标。
func TestV14ContextPackZeroRequiresMatchesV13(t *testing.T) {
	repo := newV14Repo(t)
	b, err := artifacts.LoadBundle(repo, "HEAD", "TASK-1043")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	a, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority: %v", err)
	}
	if n := len(a.Compiled.Requires().Items); n != 0 {
		t.Fatalf("夹具应零命中策略, got %d 条", n)
	}
	h14, err := v14ContextPack(repo, b.SourceCommit, b.Task.Contract, b.Verification.Frozen, a.Compiled.Requires())
	if err != nil {
		t.Fatalf("v14ContextPack: %v", err)
	}
	h13, err := v13ContextPack(repo, b.SourceCommit, b.Task.Contract, b.Verification.Frozen)
	if err != nil {
		t.Fatalf("v13ContextPack: %v", err)
	}
	if h14 != h13 {
		t.Errorf("零值 requires 改变了上下文坐标: v14=%s v13=%s", h14, h13)
	}
}

// 负面:无编译权威的 runner 拒绝执行(fail-safe,PRD §16 no-bypass 在 runner
// 缝的落点)—— 绝不无声地在未被告知验收条件的情况下开跑。
func TestV14RunnerRequiresCompiledAuthority(t *testing.T) {
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})
	r := &v14CodegenRunner{
		task:    mustV14Task(t),
		runtime: rt,
		sink:    &v13MemRunSink{},
	}
	if _, err := r.RunCodegen(context.Background(), closedloop.CodegenRunRequest{
		RunID: "TASK-1043/attempt/001",
	}); err == nil || !strings.Contains(err.Error(), "编译权威") {
		t.Fatalf("缺编译权威应拒绝执行, got %v", err)
	}
	if instrs := rt.instructions(); len(instrs) != 0 {
		t.Errorf("拒绝执行后不得有任何迭代: %q", instrs)
	}
}

// ---- P1-06:验收守卫(编译权威 → accept 转移,DL-07/A-03)----

// newV14HintRepo:TASK-1043 + 指定确认风险提示(无策略快照)—— 风险面完全
// 由提示决定的仓库夹具。
func newV14HintRepo(t *testing.T, hints ...string) string {
	t.Helper()
	repo := newV13Repo(t)
	v13WriteRepoFile(t, repo, ".tally/intents/TASK-1043.yaml", v14IntentArtifactWithHints(t, "TASK-1043", hints...))
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "v14: 带确认提示的意图制品")
	return repo
}

// AC1(生产路径):命中策略把必需档上调 L3(db_gate 匹配 schema_migration,
// 无 unmatched),验证权威给出 FinalVerified + visible 事实(achieved L1)⇒
// 闭环终态必须是 assurance_insufficient 升级,绝不 Accepted:守卫发生在
// Router 之前 —— 不伪装候选失败、不进修复、不耗预算(恰好一次尝试/验证)。
func TestV14AcceptanceGuardBlocksInsufficientAssurance(t *testing.T) {
	repo := newV14PolicyRepo(t) // schema_migration 提示 + 命中策略 ⇒ required L3
	cfg := v14ConfigFor(t, repo)
	wf := &v13MemRunSink{}
	// FinalVerified + 全绿 visible 覆盖观察(VIS-1 → achieved L1 < required L3)。
	verify := &v14FakeVerify{steps: []v14VerifyStep{{
		status: verification.FinalVerified, runID: 3,
		functional: &verification.FunctionalCoverageReport{
			TaskID: "TASK-1043", ContractRevision: 1, Complete: true,
			Acceptances: []verification.AcceptanceCoverage{
				{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"VIS-1"}},
			},
		},
	}}}
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil {
		t.Fatalf("终态 = %s, 期望 escalated(AC1:required L3 + achieved L1 永不 Accepted)", res.Phase)
	}
	if res.Escalation.Reason != closedloop.EscalationAssuranceInsufficient {
		t.Errorf("升级原因 = %s, 期望 assurance_insufficient", res.Escalation.Reason)
	}
	for _, want := range []string{"AC-01", "required L3", "achieved L1"} {
		if !strings.Contains(res.Escalation.Detail, want) {
			t.Errorf("升级明细缺 %q:%s", want, res.Escalation.Detail)
		}
	}
	if res.Basis != nil {
		t.Error("守卫拒绝的候选不得携带 basis")
	}
	if len(res.Attempts) != 1 || res.Attempts[0].Supersedes != "" || len(wf.changeSets) != 1 || len(verify.calls) != 1 {
		t.Errorf("守卫拒绝不得进入修复:attempts=%d supersede=%q changeSets=%d verify=%d",
			len(res.Attempts), res.Attempts[0].Supersedes, len(wf.changeSets), len(verify.calls))
	}
	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// AC2(生产路径):确认的 auth_symbol_change 提示无任何策略声明 ⇒ unmatched
// 信号。FinalVerified(即使保证档满足)绝不静默放行 —— 升级明细逐字携带
// 信号词面(INV-V14-09),人工/政策裁决后续。
func TestV14AcceptanceGuardBlocksUnmatchedRisk(t *testing.T) {
	repo := newV14HintRepo(t, "auth_symbol_change") // 无策略 ⇒ unmatched
	cfg := v14ConfigFor(t, repo)

	b, err := artifacts.LoadBundle(repo, "HEAD", "TASK-1043")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	a, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority: %v", err)
	}
	got := a.Compiled.Coverage().UnmatchedRiskSignals
	if len(got) != 1 || got[0] != risk.SignalAuthSymbolChange {
		t.Fatalf("夹具失真:UnmatchedRiskSignals = %v", got)
	}

	wf := &v13MemRunSink{}
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalVerified, runID: 4}, // 合成全绿报告(visible → L1)
	}}
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, wfSink: wf, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望 unmatched 风险阻断普通接受(AC2)", res.Phase, res.Escalation)
	}
	if !strings.Contains(res.Escalation.Detail, "auth_symbol_change") {
		t.Errorf("升级明细缺 unmatched 信号词面:%s", res.Escalation.Detail)
	}
	if res.Basis != nil || len(res.Attempts) != 1 || len(verify.calls) != 1 {
		t.Errorf("unmatched 阻断不得产生 basis/修复: basis=%+v attempts=%d verify=%d",
			res.Basis, len(res.Attempts), len(verify.calls))
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// 正面控制(生产路径,零风险零策略):required L1 + visible 事实(L1)⇒
// FinalVerified 正常 Accepted —— 守卫只在缺口存在时阻断。
// (TestV14ClosedLoopFirstPassAccepted / TestV14RunFactsPinCompiledAuthority
// 即该对照,真服务 + 真 worktree;此处不重复。)

// ---- 数据库层:生产事实面(每测试独立 schema;库不可达即 SKIP) ----

// AC1(持久化面):生产组成(仅注入 runtime)把 codegen_run / ChangeSet /
// verification_run / evidence 全部落成 append-only 事实。
func TestV14ClosedLoopPersistsFacts(t *testing.T) {
	runs, st := v13TestStore(t)
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	rt := newV14ScriptedRuntime(v14ScriptStep{content: "package widget\n"})

	res, err := runV14Task(context.Background(), v14Deps{
		cfg:     cfg,
		runtime: rt,
		queries: st, // 生产验证服务 + workflow/lineage/exposure 生产适配全部就位
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%v, 期望 accepted+basis", res.Phase, res.Basis)
	}

	ctx := context.Background()
	timeline, err := runs.CodegenRunsForTask(ctx, "proj-v13", "TASK-1043")
	if err != nil || len(timeline) != 1 {
		t.Fatalf("codegen_run = %+v err=%v, 期望 1 行", timeline, err)
	}
	run := timeline[0]
	if run.Status != runstore.RunCompleted || run.ResultPatchHash == "" || run.WorkspaceID == "" ||
		run.VerificationRevision == "" || run.ContextPackHash == "" {
		t.Fatalf("codegen_run = %+v", run)
	}

	css, err := runs.ChangeSetsForTask(ctx, "proj-v13", "TASK-1043")
	if err != nil || len(css) != 1 || css[0].ID() != res.Basis.ChangeSet {
		t.Fatalf("落库 ChangeSet = %+v err=%v", css, err)
	}
	if css[0].ProducerRun() != runResultRef(run.ID) {
		t.Fatalf("候选 producer run = %s, 期望 codegen_run 引用 %s", css[0].ProducerRun(), runResultRef(run.ID))
	}

	vruns, err := runs.VerificationRunsForTask(ctx, "proj-v13", "TASK-1043")
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("verification_run = %+v err=%v", vruns, err)
	}
	if vruns[0].ChangeSetID != res.Basis.ChangeSet {
		t.Fatalf("verification_run 绑定 = %+v", vruns[0])
	}

	evs, err := runs.EvidenceForRun(ctx, "proj-v13", vruns[0].ID)
	if err != nil || len(evs) != 1 {
		t.Fatalf("evidence = %+v err=%v", evs, err)
	}

	if dirty := v13GitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Errorf("主 worktree 被污染: %q", dirty)
	}
	v14AssertRootEmpty(t, cfg.v13.WorktreeRoot)
}

// 修复谱系持久化:取代事实落库 + Run 链父链接(escalated_from 槽位承载
// 闭环 Run 链的 parent link,ADR-0001 §2 attempt-numbered, parent-linked)。
func TestV14RepairSupersessionPersists(t *testing.T) {
	runs, st := v13TestStore(t)
	repo := newV14Repo(t)
	cfg := v14ConfigFor(t, repo)
	verify := &v14FakeVerify{steps: []v14VerifyStep{
		{status: verification.FinalFailed, runID: 10, bindings: v14FailedVisible()},
		{status: verification.FinalVerified, runID: 11},
	}}
	rt := newV14ScriptedRuntime(
		v14ScriptStep{content: "c1\n"},
		v14ScriptStep{content: "c2\n"},
	)

	res, err := runV14Task(context.Background(), v14Deps{
		cfg: cfg, runtime: rt, queries: st, verify: verify,
	}, "TASK-1043")
	if err != nil {
		t.Fatalf("runV14Task: %v", err)
	}
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil || res.Basis.Attempt != 2 {
		t.Fatalf("终态 = %s basis=%+v, 期望 accepted@attempt2", res.Phase, res.Basis)
	}
	a1, a2 := res.Attempts[0], res.Attempts[1]

	facts, err := runs.ChangeSetSupersessions(ctx0(), a2.ChangeSet)
	if err != nil || len(facts) != 1 {
		t.Fatalf("取代事实 = %+v err=%v, 期望 1 条", facts, err)
	}
	if facts[0].Old != a1.ChangeSet || facts[0].New != a2.ChangeSet || facts[0].Reason != changeset.SupersedeRework {
		t.Errorf("取代事实 = %+v, 期望 a1→a2 rework", facts[0])
	}

	timeline, err := runs.CodegenRunsForTask(ctx0(), "proj-v13", "TASK-1043")
	if err != nil || len(timeline) != 2 {
		t.Fatalf("codegen_run = %d 行 err=%v, 期望 2", len(timeline), err)
	}
	first, second := timeline[0], timeline[1] // 按开始时刻定序
	if second.EscalatedFrom != first.ID {
		t.Errorf("修复 Run 父链接 = %d, 期望首次 Run %d", second.EscalatedFrom, first.ID)
	}
	if first.Status != runstore.RunCompleted || second.Status != runstore.RunCompleted {
		t.Errorf("Run 终态 = %s/%s, 期望 completed/completed", first.Status, second.Status)
	}
}

// runResultRef 把 codegen_run 数字主键投影为持久化 Run 引用(RunStoreSink 合同)。
func runResultRef(id int64) string { return strconv.FormatInt(id, 10) }

func ctx0() context.Context { return context.Background() }
