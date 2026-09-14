package closedloop

// outcome_test.go 固化 P6-01 的 accepted outcome 缝:
//   - 出口绑定:accept 转移落成之前恰追加一条 accepted 事实,坐标与 Basis
//     同源(task/changeset/verify run/attempt),时刻来自 Engine 时钟;
//   - 失败政策:出口写入失败 → outcome_unavailable 显式升级(无 basis、
//     不进修复、不耗预算),绝不静默接受;
//   - 写入成功后的取消不改写终态(接受已耐久落成);写入前的取消仍是
//     一等取消终态(无 outcome);
//   - StoreAcceptSink 幂等:同一 (task, changeset) 重试/重放不重复落行,
//     读/写失败与非法事实面原样上抛。
//
// 数据库层的生产往返(真 outcome_event 表)由 cmd/worker 的 v14 接线测试
// 覆盖(同仓库测试惯例:DB 适配在消费方组合层验证)。

import (
	"context"
	"errors"
	"testing"

	"tally/internal/outcome"
)

// ---- 固件 ----

// memAcceptSink 记录全部事实;err 非空时写入失败(onCancel 非空:写入前
// 先取消调用方 ctx —— 钉「写入成功后的取消不改写终态」的顺序语义)。
type memAcceptSink struct {
	facts    []AcceptFact
	err      error
	onCancel context.CancelFunc
}

func (m *memAcceptSink) RecordAccepted(ctx context.Context, fact AcceptFact) error {
	if m.onCancel != nil {
		m.onCancel()
	}
	if m.err != nil {
		return m.err
	}
	m.facts = append(m.facts, fact)
	return nil
}

// memOutcomeEvents 是 outcomeEvents 的内存件(StoreAcceptSink 单元测试)。
type memOutcomeEvents struct {
	events []outcome.Event
	record int // Record 调用次数
	forErr error
	recErr error
}

func (m *memOutcomeEvents) Record(ctx context.Context, e outcome.Event) (int64, error) {
	if m.recErr != nil {
		return 0, m.recErr
	}
	m.record++
	m.events = append(m.events, e)
	return int64(len(m.events)), nil
}

func (m *memOutcomeEvents) ForTask(ctx context.Context, taskID string) ([]outcome.Event, error) {
	if m.forErr != nil {
		return nil, m.forErr
	}
	var out []outcome.Event
	for _, e := range m.events {
		if e.TaskID == taskID {
			out = append(out, e)
		}
	}
	return out, nil
}

// ---- Engine 缝 ----

// 出口绑定:accept 转移落成之前恰追加一条 accepted 事实,坐标与 Basis 同源,
// 时刻是 Engine 时钟(不是 sink 补记)。
func TestEngineAcceptSinkRecordsFactAtSeam(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1"}}}
	sink := &memAcceptSink{}
	e := newEngine(t, Config{Executor: exec, Verifier: verify, AcceptSink: sink})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%v, 期望 accepted+basis", res.Phase, res.Basis)
	}
	if len(sink.facts) != 1 {
		t.Fatalf("出口调用 %d 次, 期望恰 1 次(一次接受一条 accepted 事实)", len(sink.facts))
	}
	f := sink.facts[0]
	if f.TaskID != "TASK-1" || f.ChangeSet != cs1.ID() || f.VerifyRun != "vrun-1" ||
		f.Attempt != 1 || !f.RecordedAt.Equal(fixedNow) {
		t.Errorf("AcceptFact = %+v, 期望与 Basis 同源 {TASK-1, %s, vrun-1, 1, %v}",
			f, cs1.ID(), fixedNow)
	}
	if f.ChangeSet != res.Basis.ChangeSet || f.VerifyRun != res.Basis.VerifyRun || f.Attempt != res.Basis.Attempt {
		t.Errorf("AcceptFact 与 Basis 坐标漂移: %+v vs %+v", f, res.Basis)
	}
}

// 修复后接受:outcome 事实绑定最终被接受的候选与最终尝试(中间失败尝试
// 不产生 accepted 事实)。
func TestEngineAcceptSinkBindsFinalCandidateAfterRepair(t *testing.T) {
	cs1 := candidate(t, "c1")
	cs2 := candidate(t, "c2")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}, {cand: cs2}}}
	verify := &fakeVerifier{steps: []verifyStep{
		{verdict: VerdictFailed, runRef: "vrun-a"},
		{verdict: VerdictVerified, runRef: "vrun-b"},
	}}
	router := &fakeRouter{steps: []routeStep{{ruling: repairRuling()}}}
	budget := &fakeBudget{maxRepairNo: 2}
	sink := &memAcceptSink{}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, LineageSink: &memLineage{}, AcceptSink: sink,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted {
		t.Fatalf("终态 = %s, 期望 accepted", res.Phase)
	}
	if len(sink.facts) != 1 || sink.facts[0].ChangeSet != cs2.ID() || sink.facts[0].Attempt != 2 {
		t.Fatalf("facts = %+v, 期望恰一条且绑定修复候选 cs2@attempt2", sink.facts)
	}
}

// 失败政策(显式 fail-safe):出口写入失败 → outcome_unavailable 升级,
// 无 basis、无 accepted 终态;不进修复、不耗预算(这不是候选缺陷)。
func TestEngineAcceptSinkFailureEscalates(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1"}}}
	sinkErr := errors.New("outcome store unreachable")
	sink := &memAcceptSink{err: sinkErr}
	router := &fakeRouter{}
	budget := &fakeBudget{maxRepairNo: 2}
	e := newEngine(t, Config{
		Executor: exec, Verifier: verify, Router: router,
		RepairBudget: budget, AcceptSink: sink,
	})

	res, err := e.Run(context.Background(), "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseEscalated || res.Escalation == nil {
		t.Fatalf("终态 = %s escalation=%v, 期望 outcome_unavailable 升级", res.Phase, res.Escalation)
	}
	if res.Escalation.Reason != EscalationOutcomeUnavailable {
		t.Errorf("升级原因 = %s, 期望 %s", res.Escalation.Reason, EscalationOutcomeUnavailable)
	}
	if res.Escalation.Detail == "" {
		t.Error("升级必须携带类型化失败细节(绝不静默丢事件)")
	}
	if res.Basis != nil {
		t.Error("outcome 写失败的接受不得携带 basis(接受未耐久成立)")
	}
	if len(router.calls) != 0 || len(budget.asked) != 0 {
		t.Error("outcome 写失败不是候选失败:不得咨询 Router、不得耗修复预算")
	}
}

// 顺序语义:出口写入成功之后的取消不改写终态 —— 接受(连同 accepted
// 事实)已耐久落成,取消不再伪造取消终态。
func TestEngineCancelAfterAcceptSinkWriteKeepsAccepted(t *testing.T) {
	cs1 := candidate(t, "c1")
	exec := &fakeExecutor{steps: []execStep{{cand: cs1}}}
	verify := &fakeVerifier{steps: []verifyStep{{verdict: VerdictVerified, runRef: "vrun-1"}}}

	ctx, cancel := context.WithCancel(context.Background())
	sink := &memAcceptSink{onCancel: cancel} // 写入成功前取消调用方 ctx
	e := newEngine(t, Config{Executor: exec, Verifier: verify, AcceptSink: sink})

	res, err := e.Run(ctx, "TASK-1")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Phase != PhaseAccepted || len(sink.facts) != 1 {
		t.Fatalf("终态 = %s facts=%d, 期望 accepted 且事实已落成", res.Phase, len(sink.facts))
	}
}

// ---- StoreAcceptSink(幂等/失败面) ----

func okFact() AcceptFact {
	return AcceptFact{
		TaskID:     "TASK-9",
		ChangeSet:  "cs-accepted",
		VerifyRun:  "vrun-1",
		Attempt:    1,
		RecordedAt: fixedNow,
	}
}

// 幂等:同一 (task, changeset) 的重复投递(重试/重放)恰落一条 accepted,
// 不报错、不重复;同任务不同候选的接受是两行事实(事件流如实)。
func TestStoreAcceptSinkIdempotent(t *testing.T) {
	events := &memOutcomeEvents{}
	sink := &StoreAcceptSink{events: events}

	f := okFact()
	for i := 0; i < 3; i++ {
		if err := sink.RecordAccepted(context.Background(), f); err != nil {
			t.Fatalf("第 %d 次投递: %v", i+1, err)
		}
	}
	if events.record != 1 || len(events.events) != 1 {
		t.Fatalf("落行 = %d/%d, 期望重试/重放后恰 1 条", events.record, len(events.events))
	}
	got := events.events[0]
	if got.Kind != outcome.KindAccepted || got.TaskID != f.TaskID ||
		got.ChangesetID != string(f.ChangeSet) || !got.RecordedAt.Equal(fixedNow) {
		t.Errorf("accepted 事件 = %+v, 期望绑定 {%s, %s, %v}", got, f.TaskID, f.ChangeSet, fixedNow)
	}

	// 同任务另一候选的接受:新事实,不是重复。
	f2 := f
	f2.ChangeSet = "cs-later"
	f2.Attempt = 2
	if err := sink.RecordAccepted(context.Background(), f2); err != nil {
		t.Fatalf("第二候选接受: %v", err)
	}
	if events.record != 2 || len(events.events) != 2 {
		t.Fatalf("落行 = %d/%d, 期望 2 条(两次接受是两行事实)", events.record, len(events.events))
	}
}

// 失败面:读流失败、写入失败、非法事实都原样上抛,绝不静默跳过
// (静默跳过 = 丢无法事后补采的事实)。
func TestStoreAcceptSinkFailuresSurface(t *testing.T) {
	readErr := errors.New("read failed")
	sink := &StoreAcceptSink{events: &memOutcomeEvents{forErr: readErr}}
	if err := sink.RecordAccepted(context.Background(), okFact()); err == nil {
		t.Fatal("读流失败应上抛(不得把「读不到」当「没有」)")
	}

	writeErr := errors.New("write failed")
	sink = &StoreAcceptSink{events: &memOutcomeEvents{recErr: writeErr}}
	if err := sink.RecordAccepted(context.Background(), okFact()); err == nil {
		t.Fatal("写入失败应上抛(Engine 落显式升级)")
	}

	sink = &StoreAcceptSink{events: &memOutcomeEvents{}}
	bad := okFact()
	bad.VerifyRun = "" // basis 缺席:不可信结果没有接受
	if err := sink.RecordAccepted(context.Background(), bad); err == nil {
		t.Fatal("非法事实面应被写入前校验拒绝(不触库)")
	}
	if _, err := NewStoreAcceptSink(nil); !errors.Is(err, ErrInvalidAcceptSink) {
		t.Fatalf("nil 事件面应拒绝构造(ErrInvalidAcceptSink), 得到 %v", err)
	}
}
