package closedloop

// exposure.go 修复安全边界的暴露核算测试(P2-06;ADR-0001 §5;INV-V14-07/08)。
//
// 覆盖:
//   - ExposureLedger:append-only 在账(种子 + 在途记账)、无 sink 拒绝记账、
//     spec 校验 fail-safe、sink 失败不进账、重复身份是错误不是更新;
//   - 谱系单调:锚定祖先的暴露命中所有后代;持久通道封顶 lineage 通道;
//     锚定谱系外的事实不 compromise(显式非 compromise 语义);
//   - engine accept 守卫:compromised 暴露下 verified 在先也不得 accept,
//     落 exposure_compromise 升级、basis 恒空;非 compromise 暴露不影响接受;
//   - FeedbackSource 反馈缝(AC1/AC4):安全投影原样成为修复 Run 的唯一
//     失败材料通道(首次尝试恒 nil);构造失败/身份不一致一律
//     feedback_unavailable 升级,修复 Run 绝不启动、绝不降格转发;
//   - 种子事实不合法拒绝构造引擎(账面绝不带病起步)。

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/evidence"
)

// ---- 固件:暴露账面 ----

var (
	expBindingP206 = evidence.BindingRef{Project: "proj-1", Verification: "verif-r1", BindingID: "EV-H1"}
	expBaseAt      = fixedNow.Add(-2 * time.Hour) // 候选固化时刻(早于一切暴露)
)

// mustExpFact 构造一条合法暴露事实(确定性时刻:在候选固化之后)。
func mustExpFact(t *testing.T, id string, channel evidence.ExposureChannel, anchor changeset.ChangeSetID) evidence.Exposure {
	t.Helper()
	e, err := evidence.NewExposure(evidence.ExposureSpec{
		ID:        evidence.ExposureID(id),
		Binding:   expBindingP206,
		Channel:   channel,
		ChangeSet: evidence.ChangeSetID(anchor),
		ExposedAt: fixedNow.Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("NewExposure(%s): %v", id, err)
	}
	return e
}

// candidateAt 构造固化时刻受控的候选(暴露时间规则的确定性前提)。
func candidateAt(t *testing.T, marker string, at time.Time) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:      "proj-1",
		BaseRevision: "base-abc",
		Files:        []changeset.ChangedFile{{Path: "pkg/widget.go", Content: "marker: " + marker}},
		ProducerRun:  "codegen-run-" + marker,
		CreatedAt:    at,
	})
	if err != nil {
		t.Fatalf("构造候选 ChangeSet: %v", err)
	}
	return cs
}

// memExposureSink 是持久化出口的 fake:原样保存 spec,返回带分配身份的事实;
// err 非空时模拟持久化失败。零 ExposedAt 取当下(与 runstore 合同一致)。
type memExposureSink struct {
	mu    sync.Mutex
	specs []RepairExposureSpec
	seq   int
	err   error
}

func (m *memExposureSink) RecordExposure(_ context.Context, spec RepairExposureSpec) (evidence.Exposure, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return evidence.Exposure{}, m.err
	}
	at := spec.ExposedAt
	if at.IsZero() {
		at = fixedNow
	}
	m.specs = append(m.specs, spec)
	m.seq++
	return evidence.NewExposure(evidence.ExposureSpec{
		ID:         evidence.ExposureID(fmt.Sprintf("exp-%d", m.seq)),
		Binding:    spec.Binding,
		Acceptance: spec.Acceptance,
		ChangeSet:  evidence.ChangeSetID(spec.Anchor),
		Channel:    spec.Channel,
		Reason:     spec.Reason,
		ExposedAt:  at,
	})
}

// fakeFeedbackSource 是反馈缝的 fake:返回预构造的安全投影或错误。
type fakeFeedbackSource struct {
	calls []VerifyFailure
	fb    RepairFeedback
	err   error
}

func (f *fakeFeedbackSource) RepairFeedback(_ context.Context, failure VerifyFailure) (RepairFeedback, error) {
	f.calls = append(f.calls, failure)
	return f.fb, f.err
}

// safeFeedbackFor 构造与 (task, attempt, changeset) 严格绑定的最小合法反馈。
func safeFeedbackFor(t *testing.T, taskID string, no AttemptNo, cs changeset.ChangeSetID) RepairFeedback {
	t.Helper()
	fb, err := NewFeedbackBuilder(taskID, no, cs, "vrun-fail").
		ForAcceptance("AC-1", "the widget counts occurrences").
		WithFailure(FailureRepairable).
		WithDiagnosticSummary("a visible assertion failed while counting").
		Build()
	if err != nil {
		t.Fatalf("构造安全反馈: %v", err)
	}
	return fb
}

// ---- ExposureLedger:append-only 在账 ----

func TestExposureLedgerSeedsAndAppends(t *testing.T) {
	sink := &memExposureSink{}
	seed := mustExpFact(t, "seed-1", evidence.ChannelDebugDisclosure, "cs-old")
	ledger, err := NewExposureLedger(sink, []evidence.Exposure{seed})
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	if got := ledger.Facts(); len(got) != 1 || got[0].ID() != seed.ID() {
		t.Fatalf("种子事实 = %+v, 期望恰含 %s", got, seed.ID())
	}

	exp, err := ledger.RecordExposure(context.Background(), RepairExposureSpec{
		TaskID:    "TASK-1",
		Binding:   expBindingP206,
		Anchor:    "cs-1",
		Channel:   evidence.ChannelSessionLeak,
		Reason:    "diagnostic surfaced a hidden assertion name",
		ExposedAt: fixedNow,
	})
	if err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}
	if exp.ID() == "" {
		t.Fatal("落库事实必须取得持久化出口分配的身份")
	}
	if sink.specs[0].TaskID != "TASK-1" || sink.specs[0].Anchor != "cs-1" {
		t.Fatalf("出口收到的 spec = %+v", sink.specs[0])
	}
	facts := ledger.Facts()
	if len(facts) != 2 || facts[1].ID() != exp.ID() {
		t.Fatalf("在账事实 = %d 条, 期望种子 + 新事实", len(facts))
	}
}

func TestExposureLedgerFailsSafeWithoutSinkOrSpec(t *testing.T) {
	ledger, err := NewExposureLedger(nil, nil)
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	// 无持久化出口:记账被拒绝,绝不退化为只进内存账(未记账的暴露不得
	// 静默存在)。
	if _, err := ledger.RecordExposure(context.Background(), RepairExposureSpec{
		TaskID:  "TASK-1",
		Binding: expBindingP206,
		Channel: evidence.ChannelDebugDisclosure,
	}); !errors.Is(err, ErrExposureSinkUnavailable) {
		t.Fatalf("期望 ErrExposureSinkUnavailable,得到 %v", err)
	}
	if len(ledger.Facts()) != 0 {
		t.Fatal("被拒绝的记账不得进账")
	}

	// spec 不合法:类型化报错,事实绝不带病落库。
	sink := &memExposureSink{}
	ledger2, err := NewExposureLedger(sink, nil)
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	cases := []RepairExposureSpec{
		{Binding: expBindingP206, Channel: evidence.ChannelDebugDisclosure}, // 缺 task
		{TaskID: "T", Channel: evidence.ChannelDebugDisclosure},             // 缺 binding
		{TaskID: "T", Binding: expBindingP206},                              // 缺 channel
		{TaskID: "T", Binding: expBindingP206, Channel: evidence.ChannelDebugDisclosure, Reason: string(make([]rune, MaxExposureReasonLen+1))}, // provenance 超界
	}
	for i, spec := range cases {
		if _, err := ledger2.RecordExposure(context.Background(), spec); !errors.Is(err, ErrInvalidExposureSpec) {
			t.Fatalf("case %d: 期望 ErrInvalidExposureSpec,得到 %v", i, err)
		}
	}
	if len(sink.specs) != 0 {
		t.Fatal("不合法 spec 不得到达持久化出口")
	}
}

func TestExposureLedgerSinkFailureDoesNotTrack(t *testing.T) {
	sink := &memExposureSink{err: errors.New("store down")}
	ledger, err := NewExposureLedger(sink, nil)
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	if _, err := ledger.RecordExposure(context.Background(), RepairExposureSpec{
		TaskID: "TASK-1", Binding: expBindingP206, Channel: evidence.ChannelDebugDisclosure,
	}); err == nil {
		t.Fatal("持久化失败必须向上返回")
	}
	if len(ledger.Facts()) != 0 {
		t.Fatal("持久化失败的事实不进账(调用方负责 fail-safe,绝不静默)")
	}
}

func TestExposureLedgerRejectsDuplicateIdentity(t *testing.T) {
	seed := mustExpFact(t, "exp-1", evidence.ChannelDebugDisclosure, "cs-1")
	// 种子内重复身份:拒绝构造(账面绝不带病起步)。
	if _, err := NewExposureLedger(nil, []evidence.Exposure{seed, seed}); !errors.Is(err, ErrInvalidExposureLedger) {
		t.Fatalf("期望 ErrInvalidExposureLedger,得到 %v", err)
	}
	// 在途记账撞上既有身份:append-only 语义由 evidence.ExposureLog 保证
	// (重复身份是错误,不是更新)。
	ledger, err := NewExposureLedger(nil, []evidence.Exposure{seed})
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	if err := ledger.log.Append(seed); err == nil {
		t.Fatal("重复身份必须是错误,不是更新")
	}
}

// ---- 谱系单调求值 ----

func TestExposureLedgerCompromisesLineageMonotonic(t *testing.T) {
	// 锚定祖先 cs-1 的 lineage 局部暴露:命中后代 cs-2,且持久事实封顶更重。
	local := mustExpFact(t, "exp-local", evidence.ChannelDebugDisclosure, "cs-1")
	persist := mustExpFact(t, "exp-per", evidence.ChannelValidationLog, "cs-elsewhere") // 锚定谱系外但持久通道
	forward, err := NewExposureLedger(nil, []evidence.Exposure{local, persist})
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	fact, scope, hit := forward.Compromises([]evidence.ChangeSetID{"cs-2", "cs-1"}, expBaseAt)
	if !hit {
		t.Fatal("谱系外锚定的持久通道事实必须 binding-wide 命中")
	}
	if scope != evidence.ChannelScopePersistent || fact.ID() != persist.ID() {
		t.Fatalf("scope/fact = %s/%s, 期望 persistent/%s", scope, fact.ID(), persist.ID())
	}

	// 只有谱系内锚定的局部事实:lineage 封顶。
	only, err := NewExposureLedger(nil, []evidence.Exposure{local})
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	_, scope, hit = only.Compromises([]evidence.ChangeSetID{"cs-2", "cs-1"}, expBaseAt)
	if !hit || scope != evidence.ChannelScopeLineage {
		t.Fatalf("scope/hit = %s/%v, 期望 lineage/true", scope, hit)
	}

	// 锚定谱系外、候选可证明固化于暴露之前的局部事实:显式非 compromise。
	other := mustExpFact(t, "exp-other", evidence.ChannelDebugDisclosure, "cs-x")
	clean, err := NewExposureLedger(nil, []evidence.Exposure{other})
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	if _, _, hit := clean.Compromises([]evidence.ChangeSetID{"cs-2"}, expBaseAt); hit {
		t.Fatal("谱系外锚定的局部事实不得 compromise 该候选(非 compromise 语义)")
	}

	// 空账恒无 compromise。
	empty, err := NewExposureLedger(nil, nil)
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	if _, _, hit := empty.Compromises([]evidence.ChangeSetID{"cs-2"}, expBaseAt); hit {
		t.Fatal("空账不得产生 compromise")
	}
}

func TestExposureLineageOrder(t *testing.T) {
	a1 := Attempt{No: 1, ChangeSet: "cs-1"}
	a2 := Attempt{No: 2, ChangeSet: "cs-2", Supersedes: "cs-1"}
	a3 := Attempt{No: 3, ChangeSet: "cs-3", Supersedes: "cs-2"}
	got := exposureLineage([]Attempt{a1, a2, a3}, "cs-3")
	want := []evidence.ChangeSetID{"cs-3", "cs-2", "cs-1"}
	if len(got) != len(want) {
		t.Fatalf("lineage = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("lineage = %v, want %v(当前候选在前,祖先由近到远)", got, want)
		}
	}
	// 空候选的尝试不参与。
	got = exposureLineage([]Attempt{a1, {No: 2}}, "cs-1")
	if len(got) != 1 || got[0] != "cs-1" {
		t.Fatalf("lineage = %v, want [cs-1]", got)
	}
}

// ---- engine accept 守卫(AC3:strict satisfied 不能在 compromised 暴露下存活)----

// runRepairWithExposures 组装「首跑 FAIL → 修复 → verified」的闭环(cs1 失败、
// cs2 修复后通过),暴露账面由 seed 决定;返回终态与端口 fakes。
func runRepairWithExposures(t *testing.T, cs1, cs2 *changeset.ChangeSet, seed []evidence.Exposure, inRunExposure bool) Result {
	t.Helper()
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictVerified, runRef: "vrun-b"},
	}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	lineage := &memLineage{}
	sink := &memExposureSink{}
	e, err := NewEngine(Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
		ExposureSink: sink,
		Exposures:    seed,
		Now:          func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if inRunExposure {
		// 修复尝试在途时,执行适配器蓄意披露了一份 held-out 诊断:经共享
		// 账面记账(锚定祖先 cs1,lineage 局部通道)。
		prev := e.ExposureLedger()
		if prev == nil {
			t.Fatal("Engine 必须持有暴露在账集合")
		}
		exec.onExecute = func(req ExecuteRequest) {
			if !req.Attempt.IsRepair() {
				return
			}
			if _, err := prev.RecordExposure(context.Background(), RepairExposureSpec{
				TaskID:  "TASK-1",
				Binding: expBindingP206,
				Anchor:  cs1.ID(),
				Channel: evidence.ChannelDebugDisclosure,
				Reason:  "repair diagnostic surfaced held-out assertion internals",
			}); err != nil {
				t.Errorf("在途记账: %v", err)
			}
		}
	}
	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res
}

func repairCandidates(t *testing.T) (*changeset.ChangeSet, *changeset.ChangeSet) {
	t.Helper()
	return candidateAt(t, "c1", expBaseAt), candidateAt(t, "c2", expBaseAt.Add(time.Minute))
}

func TestEngineSeedExposureBlocksAcceptAndEscalates(t *testing.T) {
	cs1, cs2 := repairCandidates(t)
	// 种子:锚定祖先 cs1 的局部暴露 —— 修复候选 cs2 的谱系含 cs1,
	// 谱系单调:验证通过也不能解除封顶。
	seed := []evidence.Exposure{mustExpFact(t, "exp-anc", evidence.ChannelDebugDisclosure, cs1.ID())}
	res := runRepairWithExposures(t, cs1, cs2, seed, false)

	if res.Phase != PhaseEscalated {
		t.Fatalf("Phase = %s, 期望 escalated", res.Phase)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationExposureCompromise {
		t.Fatalf("期望 exposure_compromise 升级,得到 %+v", res.Escalation)
	}
	if res.Basis != nil {
		t.Fatal("compromised 暴露下 verified 不得产生 basis(strict 主张绝不带病存活)")
	}
	if len(res.Attempts) != 2 {
		t.Fatalf("尝试谱系 = %d 条, 期望 2(候选事实保留)", len(res.Attempts))
	}
}

func TestEngineInRunExposureBlocksAccept(t *testing.T) {
	// Run 在途蓄意披露:经共享账面记账,同样封顶 accept。
	cs1, cs2 := repairCandidates(t)
	res := runRepairWithExposures(t, cs1, cs2, nil, true)
	if res.Escalation == nil || res.Escalation.Reason != EscalationExposureCompromise {
		t.Fatalf("期望 exposure_compromise 升级,得到 %+v", res.Escalation)
	}
	if res.Basis != nil {
		t.Fatal("在途暴露封顶 accept:basis 必须为空")
	}
}

func TestEngineNonCompromisingExposureStillAccepts(t *testing.T) {
	// 锚定谱系外的局部事实 + 候选可证明固化于暴露之前:显式非 compromise,
	// 修复后正常接受(暴露核算只降不升,但也绝不冤枉干净谱系)。
	cs1, cs2 := repairCandidates(t)
	seed := []evidence.Exposure{mustExpFact(t, "exp-out", evidence.ChannelDebugDisclosure, "cs-elsewhere")}
	res := runRepairWithExposures(t, cs1, cs2, seed, false)
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("非 compromise 暴露不得阻断接受: %s / %+v", res.Phase, res.Basis)
	}
	if res.Escalation != nil {
		t.Fatalf("接受终态不得携带升级载荷: %+v", res.Escalation)
	}
}

func TestEngineFirstAttemptVerifiedUnderCompromisingExposure(t *testing.T) {
	// 首跑候选自身被 compromise:首个 verified 也不得 accept。
	cs1 := candidateAt(t, "c1", expBaseAt)
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1"}}}
	seed := []evidence.Exposure{mustExpFact(t, "exp-1", evidence.ChannelDebugDisclosure, cs1.ID())}
	e, err := NewEngine(Config{
		Executor: exec, Verifier: verify,
		Exposures: seed,
		Now:       func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Escalation == nil || res.Escalation.Reason != EscalationExposureCompromise || res.Basis != nil {
		t.Fatalf("期望 exposure_compromise 升级且无 basis,得到 %+v / %v", res.Escalation, res.Basis)
	}
	// 升级说明只含身份/通道/分类,绝不携带 held-out 内容(事实本身无内容)。
	if got := res.Escalation.Detail; got != "" && len(got) > 256 {
		t.Fatalf("升级 detail 规模越界: %q", got)
	}
}

// ---- FeedbackSource 反馈缝(AC1/AC4)----

// runRepairWithFeedback 组装「首跑 FAIL → 裁决修复 → 修复候选 verified」的
// 闭环,反馈缝由 src 决定。
func runRepairWithFeedback(t *testing.T, src FeedbackSource) (Result, *fakeExecutor, *fakeFeedbackSource) {
	t.Helper()
	cs1 := candidateAt(t, "c1", expBaseAt)
	cs2 := candidateAt(t, "c2", expBaseAt.Add(time.Minute))
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictVerified, runRef: "vrun-b"},
	}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	lineage := &memLineage{}
	var srcImpl *fakeFeedbackSource
	if s, ok := src.(*fakeFeedbackSource); ok {
		srcImpl = s
	}
	e, err := NewEngine(Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: lineage,
		FeedbackSource: src,
		Now:            func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res, exec, srcImpl
}

func TestEngineAttachesSafeFeedbackToRepairRun(t *testing.T) {
	cs1 := candidateAt(t, "c1", expBaseAt)
	fb := safeFeedbackFor(t, "TASK-1", 1, cs1.ID())
	src := &fakeFeedbackSource{fb: fb}
	res, exec, srcImpl := runRepairWithFeedback(t, src)

	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("带安全反馈的修复应正常接受: %s / %+v", res.Phase, res.Basis)
	}
	if len(srcImpl.calls) != 1 {
		t.Fatalf("反馈缝咨询 = %d 次, 期望 1", len(srcImpl.calls))
	}
	f := srcImpl.calls[0]
	if f.TaskID != "TASK-1" || f.Attempt != 1 || f.ChangeSet != cs1.ID() || f.RunRef != "vrun-a" {
		t.Fatalf("反馈缝入参 = %+v", f)
	}
	// 首次尝试不带反馈;修复尝试的编码上下文恰携带安全投影(唯一失败材料通道)。
	if exec.calls[0].Feedback != nil {
		t.Fatalf("首次尝试不应携带反馈: %+v", exec.calls[0].Feedback)
	}
	got := exec.calls[1].Feedback
	if got == nil {
		t.Fatal("修复 Run 必须携带安全反馈投影")
	}
	if got.TaskID != fb.TaskID || got.ChangeSet != fb.ChangeSet || got.DiagnosticSummary != fb.DiagnosticSummary {
		t.Fatalf("修复反馈被改写: %+v, 期望 %+v", got, fb)
	}
}

func TestEngineInsufficientFeedbackEscalatesWithoutRepair(t *testing.T) {
	cs1 := candidateAt(t, "c1", expBaseAt)
	// 构造路径的真实失败形态:builder 拒绝非 repairable 材料。
	_, buildErr := NewFeedbackBuilder("TASK-1", 1, cs1.ID(), "vrun-a").
		ForAcceptance("AC-1", "statement").
		WithFailure(FailureRequiresHuman).
		WithDiagnosticSummary("x").
		Build()
	src := &fakeFeedbackSource{err: buildErr}
	res, exec, _ := runRepairWithFeedback(t, src)

	if res.Escalation == nil || res.Escalation.Reason != EscalationFeedbackUnavailable {
		t.Fatalf("期望 feedback_unavailable 升级,得到 %+v", res.Escalation)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("exec 调用 = %d, 期望 1(没有安全反馈就不开始修复 Run)", len(exec.calls))
	}
	if res.Basis != nil {
		t.Fatal("升级终态不得携带 basis")
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("尝试谱系 = %d, 期望 1", len(res.Attempts))
	}
}

func TestEngineFeedbackIdentityMismatchFailsSafe(t *testing.T) {
	// 反馈身份与本次失败候选不一致:绝不转发(跨候选/跨任务的材料不可信)。
	cs1 := candidateAt(t, "c1", expBaseAt)
	foreign := safeFeedbackFor(t, "TASK-OTHER", 1, "cs-foreign")
	src := &fakeFeedbackSource{fb: foreign}
	res, exec, _ := runRepairWithFeedback(t, src)
	if res.Escalation == nil || res.Escalation.Reason != EscalationFeedbackUnavailable {
		t.Fatalf("期望 feedback_unavailable 升级,得到 %+v", res.Escalation)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("exec 调用 = %d, 期望 1(身份不匹配的反馈绝不进入修复)", len(exec.calls))
	}
	_ = cs1
}

func TestEngineFeedbackWrongFailureClassFailsSafe(t *testing.T) {
	// 反馈缺失败类(零值):结构上不是合法修复材料,fail-safe。
	src := &fakeFeedbackSource{fb: RepairFeedback{TaskID: "TASK-1"}}
	res, exec, _ := runRepairWithFeedback(t, src)
	if res.Escalation == nil || res.Escalation.Reason != EscalationFeedbackUnavailable {
		t.Fatalf("期望 feedback_unavailable 升级,得到 %+v", res.Escalation)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("exec 调用 = %d, 期望 1", len(exec.calls))
	}
}

func TestEngineWithoutFeedbackSourceRepairsUnchanged(t *testing.T) {
	// 反馈缝缺席:修复照旧进行(结构上没有失败材料进入通道,不存在泄漏面);
	// 这是 P2-02..07 落地语义的保持,组合根(P2-08)决定是否接线反馈缝。
	cs1 := candidateAt(t, "c1", expBaseAt)
	cs2 := candidateAt(t, "c2", expBaseAt.Add(time.Minute))
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictVerified, runRef: "vrun-b"},
	}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	e, err := NewEngine(Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: &fakeBudget{maxRepairNo: 2}, LineageSink: &memLineage{},
		Now: func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || len(exec.calls) != 2 {
		t.Fatalf("反馈缝缺席不得改变修复行为: %s / %d", res.Phase, len(exec.calls))
	}
	if exec.calls[1].Feedback != nil {
		t.Fatal("无反馈缝时修复 Run 不得携带反馈")
	}
}

// ---- 构造纪律 ----

func TestNewEngineRejectsInvalidExposureSeed(t *testing.T) {
	exec := &fakeExecutor{steps: []execStep{{cand: candidateAt(t, "c1", expBaseAt)}}}
	verify := &fakeVerifier{}
	bad := evidence.Exposure{} // 零值事实:身份/binding/通道皆缺
	if _, err := NewEngine(Config{Executor: exec, Verifier: verify, Exposures: []evidence.Exposure{bad}}); err == nil {
		t.Fatal("带不合法种子事实的引擎必须拒绝构造")
	}
}

func TestExposureLedgerConcurrentRecording(t *testing.T) {
	// 多个适配器并发记账:append-only 与身份唯一在锁内成立。
	sink := &memExposureSink{}
	ledger, err := NewExposureLedger(sink, nil)
	if err != nil {
		t.Fatalf("NewExposureLedger: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if _, err := ledger.RecordExposure(context.Background(), RepairExposureSpec{
				TaskID:    "TASK-1",
				Binding:   expBindingP206,
				Channel:   evidence.ChannelDebugDisclosure,
				Reason:    fmt.Sprintf("disclosure %d", n),
				ExposedAt: fixedNow,
			}); err != nil {
				t.Errorf("并发记账 %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()
	if got := ledger.Facts(); len(got) != 8 {
		t.Fatalf("在账事实 = %d 条, 期望 8", len(got))
	}
}
