package closedloop

// engine.go 闭环驱动器测试(全部用 fake 端口,P2-02 AC4)。
//
// 覆盖:
//   - 首过接受(带 basis);修复 = 新 Run(父链接、幂等键)+ 新候选身份 +
//     reason=rework 取代事实 + 复验(ADR-0001 §2);
//   - 预算拒绝/耗尽、非可修复路由、Router 缺席/出错/非法裁决 —— 一律显式
//     升级,绝不静默改道;
//   - Inconclusive(ERROR 语义)不进修复、不耗预算、不咨询 Router;
//   - 取消是一等终态:不伪造 verdict、不采信半途产出、不消耗预算;
//   - 候选身份新鲜性与取代事实的机械强制(身份复用/落库失败 → fail-safe);
//   - 内层重试不透明:Executor 请求只含尝试谱系身份,调用次数 = 尝试数(AC3)。

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
)

var fixedNow = time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

// ---- 固件:fake 端口 ----

// candidate 构造一个内容不可变候选;marker 不同则身份不同,完全相同的输入
// 推导出同一身份(用于身份复用场景)。
func candidate(t *testing.T, marker string) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:      core.ProjectID("proj-1"),
		BaseRevision: "base-abc",
		Files:        []changeset.ChangedFile{{Path: "pkg/widget.go", Content: "marker: " + marker}},
		ProducerRun:  "codegen-run-" + marker,
	})
	if err != nil {
		t.Fatalf("构造候选 ChangeSet: %v", err)
	}
	return cs
}

type execStep struct {
	cand   *changeset.ChangeSet
	err    error
	cancel context.CancelFunc // 非空:调用时先取消 ctx,再返回(半途产出不采信)
}

type fakeExecutor struct {
	calls []ExecuteRequest
	steps []execStep
	// onExecute 非空:每次调用先执行(P2-06 在途暴露记账等旁路观察)。
	onExecute func(ExecuteRequest)
}

func (f *fakeExecutor) ExecuteCandidate(ctx context.Context, req ExecuteRequest) (ExecuteOutcome, error) {
	f.calls = append(f.calls, req)
	if f.onExecute != nil {
		f.onExecute(req)
	}
	if len(f.steps) == 0 {
		return ExecuteOutcome{}, errors.New("fakeExecutor: 脚本耗尽")
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	if s.cancel != nil {
		s.cancel()
	}
	return ExecuteOutcome{Candidate: s.cand}, s.err
}

type verifyStep struct {
	verdict Verdict
	runRef  string
	err     error
	cancel  context.CancelFunc // 非空:调用时先取消 ctx,再返回(取消不构成 verdict)
}

type fakeVerifier struct {
	calls []VerifyRequest
	steps []verifyStep
}

func (f *fakeVerifier) VerifyCandidate(ctx context.Context, req VerifyRequest) (VerifyOutcome, error) {
	f.calls = append(f.calls, req)
	if len(f.steps) == 0 {
		return VerifyOutcome{}, errors.New("fakeVerifier: 脚本耗尽")
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	if s.cancel != nil {
		s.cancel()
	}
	return VerifyOutcome{Verdict: s.verdict, RunRef: s.runRef}, s.err
}

type routeStep struct {
	ruling Ruling
	err    error
}

type fakeRouter struct {
	calls []VerifyFailure
	steps []routeStep
}

func (f *fakeRouter) RouteFailure(ctx context.Context, failure VerifyFailure) (Ruling, error) {
	f.calls = append(f.calls, failure)
	if len(f.steps) == 0 {
		return Ruling{}, errors.New("fakeRouter: 脚本耗尽")
	}
	s := f.steps[0]
	f.steps = f.steps[1:]
	return s.ruling, s.err
}

func repairRuling() Ruling { return Ruling{Route: RouteRepair} }

func escalateRuling(reason EscalationReason) Ruling {
	return Ruling{Route: RouteEscalate, Reason: reason}
}

// fakeBudget 允许到 maxRepairNo(含)为止的修复序号;记录全部询问。
type fakeBudget struct {
	maxRepairNo AttemptNo
	asked       []AttemptNo
}

func (f *fakeBudget) CanRepair(no AttemptNo) bool {
	f.asked = append(f.asked, no)
	return no <= f.maxRepairNo
}

type memLineage struct {
	facts []changeset.SupersededFact
	err   error
}

func (m *memLineage) RecordSuperseded(ctx context.Context, fact changeset.SupersededFact) error {
	if m.err != nil {
		return m.err
	}
	m.facts = append(m.facts, fact)
	return nil
}

// newEngine 组装一个带固定时钟的 Engine。
func newEngine(t *testing.T, cfg Config) *Engine {
	t.Helper()
	cfg.Now = func() time.Time { return fixedNow }
	e, err := NewEngine(cfg)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

// ---- 场景 ----

// 首过接受:无 Repair,verifier 报告的运行引用原样成为 basis。
func TestEngineFirstPassAccepted(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1"}}}
	// Router/Budget 刻意缺席:Accepted 路径不得依赖它们(缺席只有在需要
	// 裁决失败时才构成 fail-safe)。
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted {
		t.Fatalf("Phase = %s, 期望 accepted", res.Phase)
	}
	if !res.Phase.Terminal() {
		t.Fatal("Accepted 应为终态")
	}
	if res.Basis == nil {
		t.Fatal("Accepted 结果必须携带 basis(INV:每个可接受结果都有依据)")
	}
	if res.Basis.ChangeSet != cs1.ID() || res.Basis.VerifyRun != "vrun-1" || res.Basis.Attempt != 1 {
		t.Errorf("Basis = %+v, 期望 {cs1, vrun-1, 1}", res.Basis)
	}
	if res.Escalation != nil {
		t.Errorf("Accepted 不应携带 Escalation: %+v", res.Escalation)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].ChangeSet != cs1.ID() || res.Attempts[0].Supersedes != "" {
		t.Errorf("Attempts = %+v, 期望恰一次尝试且无取代", res.Attempts)
	}
	if len(exec.calls) != 1 || len(verify.calls) != 1 {
		t.Errorf("端口调用次数: exec=%d verify=%d, 期望 1/1", len(exec.calls), len(verify.calls))
	}
}

// 修复后接受:新 Run 父链接、新候选身份、reason=rework 取代事实、复验。
func TestEngineRepairThenAccepted(t *testing.T) {
	cs1 := candidate(t, "c1")
	cs2 := candidate(t, "c2")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictVerified, runRef: "vrun-b"},
	}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	lineage := &memLineage{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("期望 Accepted 带 basis,得到 %s / %+v", res.Phase, res.Basis)
	}
	if res.Basis.ChangeSet != cs2.ID() || res.Basis.Attempt != 2 || res.Basis.VerifyRun != "vrun-b" {
		t.Errorf("Basis = %+v, 期望绑定第二个候选与第二次尝试", res.Basis)
	}

	// 尝试谱系:两 attempts,修复尝试父链接并取代前一候选。
	if len(res.Attempts) != 2 {
		t.Fatalf("Attempts = %+v, 期望 2 次", res.Attempts)
	}
	a1, a2 := res.Attempts[0], res.Attempts[1]
	if a1.No != 1 || a1.RunID != RunIDFor("TASK-1", 1) || a1.ParentRun != "" || a1.ChangeSet != cs1.ID() {
		t.Errorf("attempt1 = %+v", a1)
	}
	if a2.No != 2 || a2.RunID != RunIDFor("TASK-1", 2) || a2.ParentRun != RunIDFor("TASK-1", 1) ||
		a2.ChangeSet != cs2.ID() || a2.Supersedes != cs1.ID() {
		t.Errorf("attempt2 = %+v, 期望父链接 attempt1 并取代 cs1", a2)
	}

	// 取代事实:恰一条,reason=rework,append-only。
	if len(lineage.facts) != 1 {
		t.Fatalf("取代事实 = %d 条, 期望 1", len(lineage.facts))
	}
	fact := lineage.facts[0]
	if fact.Old != cs1.ID() || fact.New != cs2.ID() || fact.Reason != changeset.SupersedeRework {
		t.Errorf("取代事实 = %+v, 期望 cs1→cs2 (rework)", fact)
	}
	if !fact.At.Equal(fixedNow) {
		t.Errorf("取代事实时刻 = %v, 期望注入时钟 %v", fact.At, fixedNow)
	}

	// 端口调用:exec/verify 各 2 次(内层迭代不透明:调用数 = 尝试数,AC3);
	// Router 恰好被咨询 1 次,入参只含 id 与 verdict 谱系。
	if len(exec.calls) != 2 || len(verify.calls) != 2 {
		t.Errorf("端口调用次数: exec=%d verify=%d, 期望 2/2", len(exec.calls), len(verify.calls))
	}
	if len(router.calls) != 1 {
		t.Fatalf("Router 调用 = %d 次, 期望 1", len(router.calls))
	}
	f := router.calls[0]
	if f.TaskID != "TASK-1" || f.Attempt != 1 || f.ChangeSet != cs1.ID() || f.RunRef != "vrun-a" {
		t.Errorf("Router 入参 = %+v", f)
	}
	// 修复 Run 请求只携带尝试谱系身份(AC3:内层重试词汇结构性缺席),
	// 并携带冻结的前候选作为继续基(P2-05,ADR-0001 §2);首次尝试没有
	// 继续基。
	req2 := exec.calls[1]
	if req2.Attempt != 2 || req2.RunID != RunIDFor("TASK-1", 2) || req2.ParentRun != RunIDFor("TASK-1", 1) {
		t.Errorf("修复 ExecuteRequest = %+v", req2)
	}
	if exec.calls[0].RepairBase != nil {
		t.Errorf("首次尝试不应携带 RepairBase:%v", exec.calls[0].RepairBase)
	}
	if req2.RepairBase == nil || req2.RepairBase.ID() != cs1.ID() || req2.RepairBase.BaseRevision() != cs1.BaseRevision() {
		t.Errorf("修复请求应携带冻结前候选 %s 作为 RepairBase,得到 %v", cs1.ID(), req2.RepairBase)
	} else if len(req2.RepairBase.Files()) != len(cs1.Files()) || req2.RepairBase.Files()[0].Content != cs1.Files()[0].Content {
		t.Errorf("RepairBase 内容应与失败候选逐字一致(不可改写):%v", req2.RepairBase.Files())
	}
}

// 预算拒绝下一次修复:显式 repair_exhausted 升级(不静默、不再试)。
func TestEngineRepairBudgetExhausted(t *testing.T) {
	cs1 := candidate(t, "c1")
	cs2 := candidate(t, "c2")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictFailed, runRef: "vrun-b"},
	}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}, {ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2} // 只允许第 2 次尝试,拒绝第 3 次
	lineage := &memLineage{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil {
		t.Fatalf("期望 Escalated,得到 %s", res.Phase)
	}
	if res.Escalation.Reason != EscalationRepairExhausted {
		t.Errorf("升级原因 = %s, 期望 repair_exhausted", res.Escalation.Reason)
	}
	if len(res.Attempts) != 2 || len(exec.calls) != 2 || len(verify.calls) != 2 {
		t.Errorf("尝试/调用次数: attempts=%d exec=%d verify=%d, 期望 2/2/2",
			len(res.Attempts), len(exec.calls), len(verify.calls))
	}
	if len(budget.asked) != 2 || budget.asked[0] != 2 || budget.asked[1] != 3 {
		t.Errorf("预算询问 = %v, 期望 [2 3]", budget.asked)
	}
	if len(lineage.facts) != 1 {
		t.Errorf("取代事实 = %d 条, 期望 1(第二次修复未发生)", len(lineage.facts))
	}
}

// 预算端口缺席:修复结构性不可用(fail-safe),且不烧第二次 Run。
func TestEngineRepairWithoutBudgetFailsSafe(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	e := newEngine(t, Config{Executor: exec, Verifier: verify, Router: router})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationRepairUnavailable {
		t.Errorf("期望 repair_unavailable 升级,得到 %+v", res.Escalation)
	}
	if len(exec.calls) != 1 {
		t.Errorf("exec 调用 = %d, 期望 1(修复不可用时不得开新 Run)", len(exec.calls))
	}
}

// 取代事实出口缺席:没有取代事实的修复非法(ADR-0001 §2)。
func TestEngineRepairWithoutLineageSinkFailsSafe(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router, RepairBudget: budget,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationRepairUnavailable {
		t.Errorf("期望 repair_unavailable 升级,得到 %+v", res.Escalation)
	}
	if len(exec.calls) != 1 {
		t.Errorf("exec 调用 = %d, 期望 1", len(exec.calls))
	}
}

// 非可修复路由:升级原因由 Router 类型化给出,循环原样透传。
func TestEngineNonRepairableEscalates(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
	router := &fakeRouter{steps: []routeStep{{ruling: escalateRuling("contract_defect")}}}
	lineage := &memLineage{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationReason("contract_defect") {
		t.Errorf("期望透传 contract_defect 升级,得到 %+v", res.Escalation)
	}
	if len(lineage.facts) != 0 {
		t.Errorf("非修复路径不应产生取代事实,得到 %d 条", len(lineage.facts))
	}
}

// Router 缺席 / 出错 / 非法裁决:一律 fail-safe 升级,绝不默认修复。
func TestEngineRouterFailsSafe(t *testing.T) {
	run := func(t *testing.T, router Router) Result {
		cs1 := candidate(t, "c1")
		exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
		verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
		e := newEngine(t, Config{Executor: exec, Verifier: verify, Router: router})
		res, err := e.Run(context.Background(), "TASK-1")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		return res
	}

	cases := []struct {
		name   string
		router Router
	}{
		{"nil_router", nil},
		{"router_error", &fakeRouter{steps: []routeStep{{err: errors.New("boom")}}}},
		{"unknown_route", &fakeRouter{steps: []routeStep{{ruling: Ruling{Route: Route(9)}}}}},
		{"escalate_without_reason", &fakeRouter{steps: []routeStep{{ruling: Ruling{Route: RouteEscalate}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := run(t, tc.router)
			if res.Escalation == nil || res.Escalation.Reason != EscalationRouterUnavailable {
				t.Errorf("期望 router_unavailable 升级,得到 %+v", res.Escalation)
			}
		})
	}
}

// Inconclusive(ERROR 语义):不咨询 Router、不耗预算、不进修复,
// 升级为 verification_inconclusive。
func TestEngineInconclusiveNeverRepairs(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictInconclusive, runRef: "vrun-x"}}}
	router := &fakeRouter{} // 有端口但不应被咨询
	budget := &fakeBudget{maxRepairNo: 2}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: &memLineage{},
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationInconclusive {
		t.Errorf("期望 verification_inconclusive 升级,得到 %+v", res.Escalation)
	}
	if len(router.calls) != 0 {
		t.Errorf("Inconclusive 不应咨询 Router,得到 %d 次调用", len(router.calls))
	}
	if len(budget.asked) != 0 {
		t.Errorf("Inconclusive 不应消耗修复预算,预算被询问 %v", budget.asked)
	}
	if len(res.Attempts) != 1 {
		t.Errorf("尝试谱系 = %d, 期望 1(候选完成但无 verdict)", len(res.Attempts))
	}
}

// 验证结果值不可信的 fail-safe:verified 无运行引用 / 未知 verdict。
func TestEngineInvalidVerifyOutcomeFailsSafe(t *testing.T) {
	cases := []struct {
		name string
		step verifyStep
		want EscalationReason
	}{
		{"verified_without_run_ref", verifyStep{verdict: VerdictVerified, runRef: "   "}, EscalationInvalidOutcome},
		{"unknown_verdict", verifyStep{verdict: Verdict("sort-of"), runRef: "vrun-1"}, EscalationInvalidOutcome},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cs1 := candidate(t, "c1")
			exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
			verify := &fakeVerifier{steps: []verifyStep{tc.step}}
			e := newEngine(t, Config{Executor: exec, Verifier: verify})
			res, err := e.Run(context.Background(), "TASK-1")
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if res.Escalation == nil || res.Escalation.Reason != tc.want {
				t.Errorf("期望 %s 升级,得到 %+v", tc.want, res.Escalation)
			}
			if res.Basis != nil {
				t.Error("不可信验证结果不得产生 basis")
			}
		})
	}
}

// 端口基础设施错误:fail-safe 升级,绝不伪装成候选失败。
func TestExecutorInfrastructureErrorEscalates(t *testing.T) {
	exec := &fakeExecutor{steps: []execStep{{err: errors.New("worktree open failed")}}}
	verify := &fakeVerifier{}
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationInfrastructure {
		t.Errorf("期望 infrastructure_failure 升级,得到 %+v", res.Escalation)
	}
	if len(verify.calls) != 0 {
		t.Error("无可验证物时不得调用验证端口")
	}
}

// 尝试终结但没有候选:入谱系(无候选),升级 no_candidate。
func TestEngineNoCandidateEscalates(t *testing.T) {
	exec := &fakeExecutor{steps: []execStep{{cand: nil}}}
	verify := &fakeVerifier{}
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationNoCandidate {
		t.Errorf("期望 no_candidate 升级,得到 %+v", res.Escalation)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].ChangeSet != "" {
		t.Errorf("尝试谱系 = %+v, 期望一次无候选尝试", res.Attempts)
	}
	if len(verify.calls) != 0 {
		t.Error("无候选不得验证")
	}
}

// 候选身份复用:ADR-0001 §2 违背被机械捕获,取代事实不落库。
func TestEngineCandidateIdentityReuseFailsSafe(t *testing.T) {
	cs1 := candidate(t, "same") // 与第二次返回完全相同的内容 → 同一身份
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	lineage := &memLineage{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationIdentityReuse {
		t.Errorf("期望 candidate_identity_reuse 升级,得到 %+v", res.Escalation)
	}
	if len(lineage.facts) != 0 {
		t.Errorf("身份复用的候选不得落取代事实,得到 %d 条", len(lineage.facts))
	}
	if len(res.Attempts) != 1 {
		t.Errorf("违规尝试不得入谱系,得到 %d 条", len(res.Attempts))
	}
}

// 取代事实落库失败:谱系断裂的修复不成立,fail-safe 升级。
func TestEngineSupersessionFailureFailsSafe(t *testing.T) {
	cs1 := candidate(t, "c1")
	cs2 := candidate(t, "c2")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictFailed, runRef: "vrun-a"}}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	lineage := &memLineage{err: errors.New("sink down")}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationSupersessionFailed {
		t.Errorf("期望 supersession_unavailable 升级,得到 %+v", res.Escalation)
	}
	if len(verify.calls) != 1 {
		t.Errorf("取代事实未落库的候选不得进入验证,verify 调用 = %d", len(verify.calls))
	}
}

// 取消(执行中):ctx 取消优先于端口返回值 —— 合法外形的候选也不采信,
// 半途尝试不入谱系,验证端口不被调用。
func TestEngineCancellationDuringExecute(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1, cancel: cancel}}}
	verify := &fakeVerifier{}
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	res, err := e.Run(ctx, "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseCancelled {
		t.Fatalf("Phase = %s, 期望 cancelled", res.Phase)
	}
	if res.Basis != nil || res.Escalation != nil {
		t.Errorf("取消不得产生 basis/升级载荷:basis=%v escalation=%v", res.Basis, res.Escalation)
	}
	if len(res.Attempts) != 0 {
		t.Errorf("半途尝试不得入谱系,得到 %d 条", len(res.Attempts))
	}
	if len(verify.calls) != 0 {
		t.Error("取消后不得进入验证")
	}
}

// 取消(验证中):已完成的尝试保留为事实,但验证产出不构成 verdict。
func TestEngineCancellationDuringVerify(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1", cancel: cancel}}}
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	res, err := e.Run(ctx, "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseCancelled {
		t.Fatalf("Phase = %s, 期望 cancelled", res.Phase)
	}
	if res.Basis != nil {
		t.Errorf("取消的验证不得伪造 basis:%+v", res.Basis)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].ChangeSet != cs1.ID() {
		t.Errorf("已完成尝试应保留为事实,得到 %+v", res.Attempts)
	}
}

// 启动前已取消:无端口调用、空谱系取消终态。
func TestEngineCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	exec := &fakeExecutor{}
	verify := &fakeVerifier{}
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	res, err := e.Run(ctx, "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseCancelled || len(res.Attempts) != 0 {
		t.Errorf("期望空谱系取消终态,得到 %s / %d attempts", res.Phase, len(res.Attempts))
	}
	if len(exec.calls) != 0 || len(verify.calls) != 0 {
		t.Error("取消后不得调用任何端口")
	}
}

// misuse:空任务 id、二次执行、缺必填端口。
func TestEngineMisuse(t *testing.T) {
	exec := &fakeExecutor{steps: []execStep{{cand: candidate(t, "c1")}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1"}}}
	e := newEngine(t, Config{Executor: exec, Verifier: verify})

	if _, err := e.Run(context.Background(), "   "); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("空任务 id 应返回 ErrInvalidRequest,得到 %v", err)
	}
	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil || res.Phase != PhaseAccepted {
		t.Fatalf("合法 Run 应成功,得到 %v / %s", err, res.Phase)
	}
	if _, err := e.Run(context.Background(), "TASK-1"); !errors.Is(err, ErrAlreadyRun) {
		t.Errorf("二次执行应返回 ErrAlreadyRun,得到 %v", err)
	}

	if _, err := NewEngine(Config{Executor: exec}); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("缺 Verifier 应返回 ErrInvalidConfig,得到 %v", err)
	}
	if _, err := NewEngine(Config{Verifier: verify}); !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("缺 Executor 应返回 ErrInvalidConfig,得到 %v", err)
	}
}

// 多轮修复预算上限内的循环有限性:预算给多少就修复多少次,循环必然终止。
func TestEngineLoopIsBoundedByBudget(t *testing.T) {
	const maxRepairs = 4
	exec := &fakeExecutor{}
	verify := &fakeVerifier{}
	router := &fakeRouter{}
	for i := 0; i < maxRepairs+1; i++ {
		exec.steps = append(exec.steps, execStep{cand: candidate(t, "c"+strconv.Itoa(i))})
		verify.steps = append(verify.steps, verifyStep{verdict: VerdictFailed, runRef: "vrun-" + strconv.Itoa(i)})
		router.steps = append(router.steps, routeStep{ruling: repairRuling()})
	}
	budget := &fakeBudget{maxRepairNo: AttemptNo(maxRepairs + 1)} // 允许尝试 2..5
	lineage := &memLineage{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationRepairExhausted {
		t.Fatalf("期望预算耗尽升级,得到 %+v", res.Escalation)
	}
	if len(res.Attempts) != maxRepairs+1 {
		t.Errorf("尝试数 = %d, 期望 %d(1 次首跑 + %d 次修复)", len(res.Attempts), maxRepairs+1, maxRepairs)
	}
	// 谱系链:每次修复取代前一候选,取代事实恰好 = 修复次数。
	if len(lineage.facts) != maxRepairs {
		t.Fatalf("取代事实 = %d 条, 期望 %d", len(lineage.facts), maxRepairs)
	}
	for i, fact := range lineage.facts {
		wantOld := res.Attempts[i].ChangeSet
		wantNew := res.Attempts[i+1].ChangeSet
		if fact.Old != wantOld || fact.New != wantNew || fact.Reason != changeset.SupersedeRework {
			t.Errorf("取代事实[%d] = %+v, 期望 %s→%s (rework)", i, fact, wantOld, wantNew)
		}
	}
	// 继续链(P2-05,ADR-0001 §2):每次修复从紧邻的前候选继续,首次
	// 尝试没有继续基。
	if exec.calls[0].RepairBase != nil {
		t.Errorf("首次尝试不应携带 RepairBase:%v", exec.calls[0].RepairBase)
	}
	for i := 1; i < len(exec.calls); i++ {
		rb := exec.calls[i].RepairBase
		if rb == nil || rb.ID() != res.Attempts[i-1].ChangeSet {
			t.Errorf("修复 %d 应从前候选 %s 继续,RepairBase = %v", i, res.Attempts[i-1].ChangeSet, rb)
		}
	}
}
