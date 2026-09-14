// acceptguard_test.go 是 P1-06 验收守卫的聚焦测试(seam 裁决:强制点在 Engine
// 的 accept 转移,评估语义在 taskstate,组合根在 NewLoop)。
//
// 覆盖任务验收:
//   - Engine:guard 拒绝 ⇒ FinalVerified 候选升级 assurance_insufficient,绝不
//     Accepted;拒绝发生在 Router 之前(不伪装成候选失败、不进修复、不耗预算);
//     修复尝试后的复验同样受守卫裁决;guard 放行 ⇒ 正常 Accepted 带 basis;
//   - 适配:CompiledAcceptanceGuard 构造拒绝 nil;ServiceVerifier 把 FinalOutcome
//     的功能覆盖观察防御性投影进 VerifyOutcome(守卫的事实面);
//   - 组合(生产形态,真实 Compile→Bind 编译权威 + 真实 worktree):
//     Authority 在位 ⇒ 守卫恒接线;required L3 + achieved L1 永不 Accepted(AC1);
//     覆盖观察缺席 fail-safe 阻断;无权威的测试组合保持 v1.4 行为(显式钉住)。
package closedloop

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/assurance"
	"tally/internal/candidateverify"
	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workspace"
)

// ---- 固件:带覆盖观察的 Verifier fake(Engine 直连面) ----

type covVerifyStep struct {
	verdict  Verdict
	runRef   string
	coverage []verification.AcceptanceCoverage
	err      error
}

type covVerifier struct {
	calls []VerifyRequest
	steps []covVerifyStep
}

func (v *covVerifier) VerifyCandidate(_ context.Context, req VerifyRequest) (VerifyOutcome, error) {
	v.calls = append(v.calls, req)
	if len(v.steps) == 0 {
		return VerifyOutcome{}, errors.New("covVerifier: 脚本耗尽")
	}
	s := v.steps[0]
	v.steps = v.steps[1:]
	return VerifyOutcome{Verdict: s.verdict, RunRef: s.runRef, Coverage: s.coverage}, s.err
}

// recordingGuard 是 AcceptGuard 的测试替身:按脚本给出裁决并记录请求。
type recordingGuard struct {
	rulings []AcceptGuardRuling
	got     []AcceptGuardRequest
}

func (g *recordingGuard) AllowAccept(_ context.Context, req AcceptGuardRequest) AcceptGuardRuling {
	g.got = append(g.got, req)
	if len(g.rulings) == 0 {
		return AcceptGuardRuling{Allowed: true}
	}
	r := g.rulings[0]
	g.rulings = g.rulings[1:]
	return r
}

// ---- Engine:守卫拒绝 ⇒ assurance_insufficient,绝不 Accepted ----

func TestEngineAcceptanceGuardDeniesInsufficientAssurance(t *testing.T) {
	guard := &recordingGuard{rulings: []AcceptGuardRuling{
		{Allowed: false, Reason: "acceptance guard: assurance gap [AC-1 required L3 achieved L1]"},
	}}
	cov := []verification.AcceptanceCoverage{{AcceptanceID: "AC-1", Covered: true, SatisfiedBy: []string{"vis-core"}}}
	exec := &fakeExecutor{steps: []execStep{{cand: candidate(t, "c1")}}}
	verify := &covVerifier{steps: []covVerifyStep{{verdict: VerdictVerified, runRef: "9", coverage: cov}}}
	eng, err := NewEngine(Config{Executor: exec, Verifier: verify, AcceptGuard: guard})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	res, err := eng.Run(context.Background(), "TASK-GUARD")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil {
		t.Fatalf("终态 = %s, 期望 escalated(AC1:required L3 + achieved L1 永不 Accepted)", res.Phase)
	}
	if res.Escalation.Reason != EscalationAssuranceInsufficient {
		t.Errorf("升级原因 = %s, 期望 assurance_insufficient(DL-07)", res.Escalation.Reason)
	}
	if res.Basis != nil {
		t.Error("被守卫拒绝的候选不得携带 basis(VERIFIED-like 结果必须带依据,拒绝即无 basis)")
	}
	if len(res.Attempts) != 1 {
		t.Errorf("尝试 = %d, 期望 1(守卫拒绝发生在 Router 之前:不进修复、不伪装候选失败)", len(res.Attempts))
	}
	// 守卫收到的事实面:精确绑定候选与验证运行,覆盖观察原样透传。
	if len(guard.got) != 1 {
		t.Fatalf("守卫咨询 = %d 次, 期望 1", len(guard.got))
	}
	req := guard.got[0]
	if req.TaskID != "TASK-GUARD" || req.Attempt != 1 || req.VerifyRun != "9" {
		t.Errorf("守卫请求身份 = %+v", req)
	}
	if len(req.Coverage) != 1 || req.Coverage[0].AcceptanceID != "AC-1" {
		t.Errorf("守卫请求覆盖观察 = %+v", req.Coverage)
	}
}

// 守卫放行 ⇒ 正常 Accepted,守卫收到精确的 (候选, 验证运行) 绑定。
func TestEngineAcceptanceGuardAllowsSufficientCandidate(t *testing.T) {
	guard := &recordingGuard{}
	cand := candidate(t, "ok")
	exec := &fakeExecutor{steps: []execStep{{cand: cand}}}
	verify := &covVerifier{steps: []covVerifyStep{{verdict: VerdictVerified, runRef: "41"}}}
	eng, err := NewEngine(Config{Executor: exec, Verifier: verify, AcceptGuard: guard})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	res, err := eng.Run(context.Background(), "TASK-GUARD")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%v, 期望 accepted + basis", res.Phase, res.Basis)
	}
	if res.Basis.ChangeSet != cand.ID() || res.Basis.VerifyRun != "41" {
		t.Errorf("Basis = %+v, 期望绑定候选与验证运行 41", res.Basis)
	}
	if len(guard.got) != 1 || guard.got[0].ChangeSet != cand.ID() {
		t.Errorf("守卫请求 = %+v, 期望绑定被接受候选", guard.got)
	}
}

// 修复后的复验同样受守卫裁决:FAIL → repair → verified(带缺口)⇒ 升级,
// 修复谱系完整保留(取代事实已落),绝不带缺口接受。
func TestEngineAcceptanceGuardFiresOnRepairedCandidate(t *testing.T) {
	guard := &recordingGuard{rulings: []AcceptGuardRuling{
		{Allowed: false, Reason: "acceptance guard: unmatched risk signals [auth_symbol_change]"},
	}}
	lineage := &memLineage{}
	policy, err := NewRepairPolicy(DefaultMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy: %v", err)
	}
	eng, err := NewEngine(Config{
		Executor: &fakeExecutor{steps: []execStep{
			{cand: candidate(t, "c1")},
			{cand: candidate(t, "c2")},
		}},
		Verifier: &covVerifier{steps: []covVerifyStep{
			{verdict: VerdictFailed, runRef: "a"},
			{verdict: VerdictVerified, runRef: "b"},
		}},
		Router:       &fakeRouter{steps: []routeStep{{ruling: RulingForClass(FailureRepairable)}}},
		RepairBudget: policy,
		LineageSink:  lineage,
		AcceptGuard:  guard,
		Now:          func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	res, err := eng.Run(context.Background(), "TASK-GUARD")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %+v, 期望修复后的候选也被守卫阻断", res)
	}
	if len(res.Attempts) != 2 || len(lineage.facts) != 1 {
		t.Errorf("修复谱系应完整保留:attempts=%d facts=%d", len(res.Attempts), len(lineage.facts))
	}
	if len(guard.got) != 1 || guard.got[0].Attempt != 2 {
		t.Errorf("守卫应只裁决 accept 转移(修复后的候选): %+v", guard.got)
	}
}

// ---- 适配:构造守卫 + ServiceVerifier 覆盖投影 ----

func TestNewCompiledAcceptanceGuardRejectsNilInputs(t *testing.T) {
	frozen := lpContract(t)
	if _, err := NewCompiledAcceptanceGuard(nil, frozen); !errors.Is(err, ErrInvalidAcceptanceGuard) {
		t.Errorf("nil 编译权威应拒绝构造,得到 %v", err)
	}
	compiled := lpGuardAuthority(t, frozen, "")
	if _, err := NewCompiledAcceptanceGuard(compiled, nil); !errors.Is(err, ErrInvalidAcceptanceGuard) {
		t.Errorf("nil 冻结契约应拒绝构造,得到 %v", err)
	}
	if _, err := NewCompiledAcceptanceGuard(compiled, frozen); err != nil {
		t.Fatalf("合法输入应可构造: %v", err)
	}
}

// covService 是 CandidateVerification 替身:返回携带功能覆盖报告的 FinalOutcome。
type covService struct {
	out *verification.FinalOutcome
}

func (s *covService) Verify(_ context.Context, req candidateverify.Request) (*verification.FinalOutcome, error) {
	out := *s.out
	out.TaskID = req.Contract.TaskID()
	out.ChangeSetID = req.ChangeSet.ID()
	return &out, nil
}

// 守卫的事实面来自 FinalOutcome.Functional 的防御性投影:调用方原地改写已交付
// 的结论不得影响 VerifyOutcome.Coverage(事实只增不改的同纪律)。
func TestServiceVerifierProjectsCoverageForGuard(t *testing.T) {
	svc := &covService{out: &verification.FinalOutcome{
		RunID:  5,
		Status: verification.FinalVerified,
		Functional: verification.FunctionalCoverageReport{
			Complete: true,
			Acceptances: []verification.AcceptanceCoverage{
				{AcceptanceID: "AC-1", Covered: true, SatisfiedBy: []string{"vis-core"}},
			},
		},
	}}
	v, err := NewServiceVerifier(ServiceVerifierConfig{Service: svc, Contract: lpContract(t), Runtime: loopRuntime})
	if err != nil {
		t.Fatalf("NewServiceVerifier: %v", err)
	}
	cs := candidate(t, "c1")
	vo, err := v.VerifyCandidate(context.Background(), VerifyRequest{
		TaskID: loopTask, Attempt: 1, ChangeSet: cs.ID(), Snapshot: cs,
	})
	if err != nil {
		t.Fatalf("VerifyCandidate: %v", err)
	}
	if vo.Verdict != VerdictVerified || vo.RunRef != "5" {
		t.Fatalf("verdict/runRef = %s/%s", vo.Verdict, vo.RunRef)
	}
	if len(vo.Coverage) != 1 || vo.Coverage[0].AcceptanceID != "AC-1" || !vo.Coverage[0].Covered {
		t.Fatalf("Coverage = %+v, 期望投影 AC-1 覆盖观察", vo.Coverage)
	}
	svc.out.Functional.Acceptances[0].AcceptanceID = "TAMPERED"
	if vo.Coverage[0].AcceptanceID != "AC-1" {
		t.Error("Coverage 未与调用方输入脱钩(防御性拷贝缺失)")
	}
}

// ---- 组合(生产形态):NewLoop 在编译权威在位时恒接线守卫 ----

// lpGuardTaskYAML 与 lpContract 同 task id 的任务合同(Compile 要求 id 一致)。
const lpGuardTaskYAML = `
id: TASK-P2-08
profile: standard
title: acceptance guard fixture
objective: prove the compiled acceptance authority gates the accept transition
requirement_refs:
  - .tally/requirements/REQ-007.md
scope:
  allowed_paths:
    - pkg/**
acceptance:
  - id: AC-1
    statement: the widget greets deterministically
`

// lpGuardGatePolicyYAML 命中 schema_migration 的 gate 策略(内容修订由 Parse
// 填充)—— 风险上调必需档到 L3 的驱动输入。
const lpGuardGatePolicyYAML = `
id: policy-guard-migration
enforcement: gate
rationale: migration-class changes must prove upgrade path on a fresh database
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`

// lpGuardAuthority 走真实 Compile→Bind 编译权威(绝不手构 contract.Compiled);
// signals 为空 = 零风险(全条款 L1 地板)。
func lpGuardAuthority(t *testing.T, frozen *verification.Frozen, policyYAML string, signals ...risk.SignalID) *contract.Compiled {
	t.Helper()
	tc, err := taskcontract.Parse([]byte(lpGuardTaskYAML))
	if err != nil {
		t.Fatalf("taskcontract.Parse: %v", err)
	}
	snap, err := contract.Compile(tc, frozen)
	if err != nil {
		t.Fatalf("contract.Compile: %v", err)
	}
	var policies []engineeringpolicy.Policy
	if policyYAML != "" {
		p, perr := engineeringpolicy.Parse([]byte(policyYAML))
		if perr != nil {
			t.Fatalf("engineeringpolicy.Parse: %v", perr)
		}
		policies = append(policies, p)
	}
	sigs := make([]risk.Signal, 0, len(signals))
	for _, id := range signals {
		sigs = append(sigs, risk.Signal{ID: id, Basis: []string{"task:TASK-P2-08"}})
	}
	r, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	c, err := contract.Bind(snap, policies, r)
	if err != nil {
		t.Fatalf("contract.Bind: %v", err)
	}
	return c
}

// lpGuardLoop 组装带编译权威的生产形态闭环(真实 git 仓库 + 一次性 worktree)。
func lpGuardLoop(t *testing.T, codegen *loopRunner, verify *loopVerification, authority *contract.Compiled) *lpFixture {
	t.Helper()
	repo, base := lpRepo(t)
	root := filepath.Join(t.TempDir(), "wt-guard")
	mgr, err := workspace.New(repo, root)
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	frozen := lpContract(t)
	verify.contract = frozen
	policy, err := NewRepairPolicy(DefaultMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy: %v", err)
	}
	engine, err := NewLoop(LoopConfig{
		Codegen:      codegen,
		Verify:       verify,
		Contract:     frozen,
		Authority:    authority,
		Runtime:      loopRuntime,
		Worktrees:    workspace.NewRunTable(mgr),
		BaseRevision: func(context.Context, string) (string, error) { return base, nil },
		Repair:       policy,
		Lineage:      &memLineage{},
		Now:          func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewLoop: %v", err)
	}
	return &lpFixture{engine: engine, codegen: codegen, verify: verify, base: base, root: root}
}

// AC1(经生产组合):required L3(命中策略上调)+ achieved L1(visible 事实)
// ⇒ FinalVerified 升级 assurance_insufficient,绝不 Accepted,不进修复。
func TestLoopGuardBlocksRequiredL3WithAchievedL1(t *testing.T) {
	frozen := lpContract(t) // 先建 frozen 供编译权威使用
	authority := lpGuardAuthority(t, frozen, lpGuardGatePolicyYAML, risk.SignalSchemaMigration)
	if cl := authority.Clauses(); len(cl) != 1 || cl[0].Assurance.Required != assurance.LevelL3 {
		t.Fatalf("夹具失真:required = %+v, 期望 L3", cl)
	}
	verify := &loopVerification{steps: []lpVerifyStep{{
		status: verification.FinalVerified, runID: 7,
		functional: &verification.FunctionalCoverageReport{
			Complete: true,
			Acceptances: []verification.AcceptanceCoverage{
				{AcceptanceID: loopAcceptance, Covered: true, SatisfiedBy: []string{loopBinding}}, // visible → L1
			},
		},
	}}}
	fx := lpGuardLoop(t, &loopRunner{steps: []lpRunStep{{content: "candidate one"}}}, verify, authority)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望 assurance_insufficient(AC1)", res.Phase, res.Escalation)
	}
	for _, want := range []string{loopAcceptance, "required L3", "achieved L1"} {
		if !strings.Contains(res.Escalation.Detail, want) {
			t.Errorf("升级明细缺 %q:%s", want, res.Escalation.Detail)
		}
	}
	if res.Basis != nil {
		t.Error("守卫拒绝的候选不得携带 basis")
	}
	// 不进修复、不伪装候选失败:恰好一次执行 + 一次验证。
	if len(fx.codegen.calls) != 1 || len(fx.verify.calls) != 1 {
		t.Errorf("端口调用 = exec %d / verify %d, 期望 1/1", len(fx.codegen.calls), len(fx.verify.calls))
	}
	lpAssertRootEmpty(t, fx.root)
}

// fail-safe(经生产组合):FinalVerified 而功能覆盖观察缺席 = 无观察事实,
// 守卫按 L0 阻断 —— 绝不把「没有观察」当作「已满足」。
func TestLoopGuardBlocksMissingCoverageObservation(t *testing.T) {
	frozen := lpContract(t)
	authority := lpGuardAuthority(t, frozen, "") // 零风险:L1 地板
	verify := &loopVerification{steps: []lpVerifyStep{{
		status: verification.FinalVerified, runID: 8, // functional 缺席
	}}}
	fx := lpGuardLoop(t, &loopRunner{steps: []lpRunStep{{content: "candidate one"}}}, verify, authority)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望缺席观察 fail-safe 阻断", res.Phase, res.Escalation)
	}
	if !strings.Contains(res.Escalation.Detail, loopAcceptance) ||
		!strings.Contains(res.Escalation.Detail, "no verification facts") {
		t.Errorf("升级明细缺无事实标注:%s", res.Escalation.Detail)
	}
	lpAssertRootEmpty(t, fx.root)
}

// 正面控制(经生产组合):必需档 L1 + 全绿覆盖观察(L1 事实)⇒ Accepted 带
// basis —— 守卫只在缺口存在时阻断,不改变正常路径。
func TestLoopGuardAllowsSufficientCandidate(t *testing.T) {
	frozen := lpContract(t)
	authority := lpGuardAuthority(t, frozen, "")
	verify := &loopVerification{steps: []lpVerifyStep{{
		status: verification.FinalVerified, runID: 41,
		functional: &verification.FunctionalCoverageReport{
			Complete: true,
			Acceptances: []verification.AcceptanceCoverage{
				{AcceptanceID: loopAcceptance, Covered: true, SatisfiedBy: []string{loopBinding}},
			},
		},
	}}}
	fx := lpGuardLoop(t, &loopRunner{steps: []lpRunStep{{content: "candidate one"}}}, verify, authority)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil || res.Basis.VerifyRun != "41" {
		t.Fatalf("终态 = %s basis=%+v, 期望 accepted + basis", res.Phase, res.Basis)
	}
	lpAssertRootEmpty(t, fx.root)
}

// 显式钉住组合语义:Authority 缺席(无编译权威的测试/试验组合)= 守卫未绑定,
// 保持 v1.4 行为;生产组合根(cmd/worker)恒传编译权威,其 no-bypass 由
// P1-08 的门与 worker 接线测试钉住。
func TestLoopWithoutAuthorityKeepsLegacyAccept(t *testing.T) {
	verify := &loopVerification{steps: []lpVerifyStep{{
		status: verification.FinalVerified, runID: 3, // functional 缺席
	}}}
	fx, _ := lpDefault(t, &loopRunner{steps: []lpRunStep{{content: "candidate one"}}}, verify)

	res, err := fx.engine.Run(context.Background(), loopTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted {
		t.Fatalf("终态 = %s, 期望无权威组合保持 v1.4 行为", res.Phase)
	}
}
