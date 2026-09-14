package fake

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
)

// fixedNow 让事件时间戳完全确定,便于确定性断言。
func fixedNow() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

func instrRequest(text string) agentruntime.ExecuteRequest {
	return agentruntime.ExecuteRequest{Instruction: text}
}

func defaultStart() agentruntime.StartRequest {
	return agentruntime.StartRequest{Workspace: "C:/worktrees/task-1"}
}

func startSession(t *testing.T, rt *Runtime, req agentruntime.StartRequest) agentruntime.Session {
	t.Helper()
	s, err := rt.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("Start: unexpected error: %v", err)
	}
	return s
}

// nextEvent 在超时保护下读取下一个事件;等待失败直接 FailNow。
func nextEvent(t *testing.T, st agentruntime.EventStream) agentruntime.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ev, err := st.Next(ctx)
	if err != nil {
		t.Fatalf("EventStream.Next: unexpected error: %v", err)
	}
	return ev
}

// drainEvents 读取事件直到流关闭(或超时 FailNow),返回已收到的事件。
func drainEvents(t *testing.T, st agentruntime.EventStream) []agentruntime.Event {
	t.Helper()
	var out []agentruntime.Event
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for {
		ev, err := st.Next(ctx)
		if err != nil {
			if !errors.Is(err, agentruntime.ErrEventStreamClosed) {
				t.Fatalf("EventStream.Next: expected ErrEventStreamClosed, got %v", err)
			}
			return out
		}
		out = append(out, ev)
	}
}

// --- AC1: Can start/execute/cancel ---

func TestStartRequiresWorkspace(t *testing.T) {
	rt := New(Config{})
	_, err := rt.Start(context.Background(), agentruntime.StartRequest{})
	if !errors.Is(err, agentruntime.ErrNoWorkspace) {
		t.Fatalf("Start without workspace: want ErrNoWorkspace, got %v", err)
	}
}

func TestStartRefusesUndeclaredCapabilities(t *testing.T) {
	rt := New(Config{})
	requests := []agentruntime.StartRequest{
		// FakeRuntime 故意不声明 model.select/session.resume(PRD §10.4),
		// 带相应需求的 Start 必须在协商处被拒绝而不是被隐式放行(AC4)。
		{Workspace: "C:/w", ModelProfile: "go-standard-cheap"},
		{Workspace: "C:/w", ResumeOf: "session-0001"},
	}
	for _, req := range requests {
		_, err := rt.Start(context.Background(), req)
		var capErr *agentruntime.CapabilityError
		if !errors.As(err, &capErr) {
			t.Fatalf("Start %+v: want *CapabilityError, got %v", req, err)
		}
		if len(capErr.Missing) != 1 {
			t.Fatalf("Start %+v: want exactly 1 missing capability, got %v", req, capErr.Missing)
		}
	}
}

func TestStartRecordsWorkspace(t *testing.T) {
	// PRD §35: receives cwd/workspace。
	rt := New(Config{})
	s := startSession(t, rt, defaultStart())
	fs, ok := s.(*fakeSession)
	if !ok {
		t.Fatalf("Start: want *fakeSession, got %T", s)
	}
	if fs.Workspace() != defaultStart().Workspace {
		t.Fatalf("Workspace: want %q, got %q", defaultStart().Workspace, fs.Workspace())
	}
	if fs.ID() != agentruntime.SessionID("session-0001") {
		t.Fatalf("ID: want session-0001, got %q", fs.ID())
	}
}

func TestExecuteCompletedTurnIsDeterministicClaim(t *testing.T) {
	rt := New(Config{Now: fixedNow})
	s := startSession(t, rt, defaultStart())

	res, err := s.Execute(context.Background(), instrRequest("add a test"))
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}
	if res.Status != agentruntime.RunCompleted {
		t.Fatalf("Status: want %q, got %q", agentruntime.RunCompleted, res.Status)
	}
	if res.RunID != agentruntime.RunID("run-0001") || res.SessionID != agentruntime.SessionID("session-0001") {
		t.Fatalf("IDs not deterministic: run=%q session=%q", res.RunID, res.SessionID)
	}
	if res.Summary != defaultSummaryCompleted {
		t.Fatalf("Summary: want %q, got %q", defaultSummaryCompleted, res.Summary)
	}
	if res.Usage != (agentruntime.Usage{InputTokens: 10, OutputTokens: 5}) {
		t.Fatalf("Usage: want default 10/5, got %+v", res.Usage)
	}

	// Result 返回同一 claim 快照。
	got, err := s.Result(context.Background(), res.RunID)
	if err != nil {
		t.Fatalf("Result: unexpected error: %v", err)
	}
	if got != res {
		t.Fatalf("Result snapshot: want %+v, got %+v", res, got)
	}

	// 对不存在/未知的 run 查询与取消都是 ErrRunNotFound。
	if _, err := s.Result(context.Background(), "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Result(unknown): want ErrRunNotFound, got %v", err)
	}
	if err := s.Cancel(context.Background(), "run-bogus"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("Cancel(unknown): want ErrRunNotFound, got %v", err)
	}
}

func TestExecuteRequiresInstruction(t *testing.T) {
	rt := New(Config{})
	s := startSession(t, rt, defaultStart())
	if _, err := s.Execute(context.Background(), agentruntime.ExecuteRequest{}); !errors.Is(err, agentruntime.ErrNoInstruction) {
		t.Fatalf("Execute without instruction: want ErrNoInstruction, got %v", err)
	}
}

func TestCancelRunningTurn(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnWaitCancel}}})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}

	type execOutcome struct {
		res agentruntime.AgentRunResult
		err error
	}
	outcome := make(chan execOutcome, 1)
	go func() {
		res, err := s.Execute(context.Background(), instrRequest("long work"))
		outcome <- execOutcome{res, err}
	}()

	started := nextEvent(t, st)
	if started.Kind != agentruntime.EventRunStarted || started.RunID != agentruntime.RunID("run-0001") {
		t.Fatalf("first event: want run.started/run-0001, got %s/%s", started.Kind, started.RunID)
	}

	if err := s.Cancel(context.Background(), started.RunID); err != nil {
		t.Fatalf("Cancel: unexpected error: %v", err)
	}
	got := <-outcome
	if got.err != nil {
		t.Fatalf("Execute(cancelled turn): unexpected error: %v", got.err)
	}
	if got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Status: want %q, got %q", agentruntime.RunCancelled, got.res.Status)
	}

	// 终态快照可见;对已终态的 run 再次 Cancel 幂等成功。
	snap, err := s.Result(context.Background(), started.RunID)
	if err != nil || snap.Status != agentruntime.RunCancelled {
		t.Fatalf("Result after cancel: want cancelled snapshot, got %+v err=%v", snap, err)
	}
	if err := s.Cancel(context.Background(), started.RunID); err != nil {
		t.Fatalf("Cancel(terminal run): want idempotent nil, got %v", err)
	}

	// 终态事件关闭事件流(run.started 已被上面的 nextEvent 消费)。
	events := drainEvents(t, st)
	if len(events) != 1 || events[0].Kind != agentruntime.EventRunCancelled {
		t.Fatalf("remaining events: want [run.cancelled], got %+v", events)
	}
}

func TestExecuteCancelledByContext(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnWaitCancel}}})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	type execOutcome struct {
		res agentruntime.AgentRunResult
		err error
	}
	outcome := make(chan execOutcome, 1)
	go func() {
		res, err := s.Execute(ctx, instrRequest("long work"))
		outcome <- execOutcome{res, err}
	}()

	if ev := nextEvent(t, st); ev.Kind != agentruntime.EventRunStarted {
		t.Fatalf("first event: want run.started, got %s", ev.Kind)
	}
	cancel()

	got := <-outcome
	if got.err != nil {
		t.Fatalf("Execute(ctx cancelled): unexpected error: %v", got.err)
	}
	if got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Status: want %q, got %q", agentruntime.RunCancelled, got.res.Status)
	}
}

// --- AC2: Emits normalized events ---

func TestEventsDeliverNormalizedSequence(t *testing.T) {
	rt := New(Config{
		Now:   fixedNow,
		Turns: []TurnSpec{{Message: "did the thing", Usage: &agentruntime.Usage{InputTokens: 7, OutputTokens: 3}}},
	})
	s := startSession(t, rt, defaultStart())

	// 先订阅后 Execute:run.started 携带 Execute 内部分配的 RunID。
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	if _, err := s.Execute(context.Background(), instrRequest("do it")); err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	events := drainEvents(t, st)
	wantKinds := []agentruntime.EventKind{
		agentruntime.EventRunStarted,
		agentruntime.EventAgentMessage,
		agentruntime.EventUsageReported,
		agentruntime.EventRunCompleted,
	}
	if len(events) != len(wantKinds) {
		t.Fatalf("event count: want %d, got %d: %+v", len(wantKinds), len(events), events)
	}
	var prevSeq int64
	for i, ev := range events {
		if ev.Kind != wantKinds[i] {
			t.Fatalf("event[%d]: want %s, got %s", i, wantKinds[i], ev.Kind)
		}
		if ev.RunID != agentruntime.RunID("run-0001") || ev.SessionID != agentruntime.SessionID("session-0001") {
			t.Fatalf("event[%d] attribution: run=%q session=%q", i, ev.RunID, ev.SessionID)
		}
		if ev.Seq != prevSeq+1 {
			t.Fatalf("event[%d]: Seq not contiguous: prev=%d got=%d", i, prevSeq, ev.Seq)
		}
		prevSeq = ev.Seq
		if !ev.Time.Equal(fixedNow()) {
			t.Fatalf("event[%d]: Time not deterministic: %v", i, ev.Time)
		}
	}
	if events[1].Text != "did the thing" {
		t.Fatalf("agent.message text: want scripted override, got %q", events[1].Text)
	}
	if events[2].Text != "input_tokens=7 output_tokens=3" {
		t.Fatalf("usage.reported text: got %q", events[2].Text)
	}
}

func TestEventsEchoInstructionByDefault(t *testing.T) {
	// agent.message 默认原样回显 instruction:证明指令确实到达 runtime。
	rt := New(Config{})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	if _, err := s.Execute(context.Background(), instrRequest("write the fizzbuzz")); err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}
	events := drainEvents(t, st)
	if len(events) != 4 {
		t.Fatalf("event count: want 4, got %d: %+v", len(events), events)
	}
	if events[1].Kind != agentruntime.EventAgentMessage || events[1].Text != "write the fizzbuzz" {
		t.Fatalf("agent.message: want instruction echo, got %s/%q", events[1].Kind, events[1].Text)
	}
}

func TestEventsHaveNoNativePayload(t *testing.T) {
	// AC5:FakeRuntime 没有任何原生 session/事件形态,规范化事件不携带
	// Native 载荷。
	rt := New(Config{})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	if _, err := s.Execute(context.Background(), instrRequest("x")); err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}
	for _, ev := range drainEvents(t, st) {
		if ev.HasNative() {
			t.Fatalf("event %s: want no native payload, got %+v", ev.Kind, ev.Native)
		}
	}
}

func TestEventsCancelWaitingNextOnContextDone(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnWaitCancel}}})
	s := startSession(t, rt, defaultStart())

	// 用流 A 同步等到 run.started,再订阅流 B:此后直到取消都不会有事件,
	// B 的 Next 只能被 ctx 释放。
	stA, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	type execOutcome struct {
		res agentruntime.AgentRunResult
		err error
	}
	outcome := make(chan execOutcome, 1)
	go func() {
		res, err := s.Execute(context.Background(), instrRequest("long work"))
		outcome <- execOutcome{res, err}
	}()
	if ev := nextEvent(t, stA); ev.Kind != agentruntime.EventRunStarted {
		t.Fatalf("first event: want run.started, got %s", ev.Kind)
	}

	stB, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := stB.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Next(cancelled ctx): want context.Canceled, got %v", err)
	}

	if err := s.Cancel(context.Background(), "run-0001"); err != nil {
		t.Fatalf("Cancel: unexpected error: %v", err)
	}
	if got := <-outcome; got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Status: want cancelled, got %q", got.res.Status)
	}
}

func TestEventStreamCloseAbandonsSubscription(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnWaitCancel}}})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	type execOutcome struct {
		res agentruntime.AgentRunResult
		err error
	}
	outcome := make(chan execOutcome, 1)
	go func() {
		res, err := s.Execute(context.Background(), instrRequest("long work"))
		outcome <- execOutcome{res, err}
	}()
	if ev := nextEvent(t, st); ev.Kind != agentruntime.EventRunStarted {
		t.Fatalf("first event: want run.started, got %s", ev.Kind)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("stream Close: unexpected error: %v", err)
	}
	// 放弃订阅后 Next 恒返回 ErrEventStreamClosed;Close 幂等。
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := st.Next(ctx); !errors.Is(err, agentruntime.ErrEventStreamClosed) {
		t.Fatalf("Next after Close: want ErrEventStreamClosed, got %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("stream Close (idempotent): unexpected error: %v", err)
	}
	// 收尾:终止被阻塞的 run,确认 Execute 正常返回。
	if err := s.Cancel(context.Background(), "run-0001"); err != nil {
		t.Fatalf("Cancel: unexpected error: %v", err)
	}
	if got := <-outcome; got.err != nil || got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("Execute outcome: want cancelled, got %+v err=%v", got.res, got.err)
	}
}

func TestEventsRejectedAfterSessionClose(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnWaitCancel}}})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	type execOutcome struct {
		res agentruntime.AgentRunResult
		err error
	}
	outcome := make(chan execOutcome, 1)
	go func() {
		res, err := s.Execute(context.Background(), instrRequest("long work"))
		outcome <- execOutcome{res, err}
	}()
	if ev := nextEvent(t, st); ev.Kind != agentruntime.EventRunStarted {
		t.Fatalf("first event: want run.started, got %s", ev.Kind)
	}

	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close: unexpected error: %v", err)
	}
	// 关闭后一切操作返回 ErrSessionClosed。
	if _, err := s.Events(context.Background()); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Events after Close: want ErrSessionClosed, got %v", err)
	}
	if _, err := s.Execute(context.Background(), instrRequest("x")); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Execute after Close: want ErrSessionClosed, got %v", err)
	}
	if _, err := s.Result(context.Background(), "run-0001"); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Result after Close: want ErrSessionClosed, got %v", err)
	}
	if err := s.Cancel(context.Background(), "run-0001"); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Cancel after Close: want ErrSessionClosed, got %v", err)
	}
	// Close 幂等。
	if err := s.Close(context.Background()); err != nil {
		t.Fatalf("Close (idempotent): unexpected error: %v", err)
	}
	// 阻塞中的 run 以取消收尾;run.started 已被消费,排干后只剩终态事件。
	got := <-outcome
	if got.err != nil || got.res.Status != agentruntime.RunCancelled {
		t.Fatalf("in-flight run after Close: want cancelled result, got %+v err=%v", got.res, got.err)
	}
	rest := drainEvents(t, st)
	if len(rest) != 1 || rest[0].Kind != agentruntime.EventRunCancelled {
		t.Fatalf("remaining events after Close: want [run.cancelled], got %+v", rest)
	}
}

// --- AC3: Can simulate crash/failure/usage ---

func TestTurnFailKeepsSessionUsable(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{
		{Behavior: TurnFail, Summary: "could not compile", Usage: &agentruntime.Usage{InputTokens: 1, OutputTokens: 1}},
	}})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}

	res, err := s.Execute(context.Background(), instrRequest("break it"))
	if err != nil {
		t.Fatalf("Execute(fail turn): unexpected error: %v", err)
	}
	if res.Status != agentruntime.RunFailed || res.Summary != "could not compile" {
		t.Fatalf("fail turn claim: want failed/\"could not compile\", got %q/%q", res.Status, res.Summary)
	}

	// 失败不是崩溃:会话仍可用,下一个(默认)turn 正常完成。
	// 先订阅第二个流再执行(事件流从订阅时刻开始交付)。
	st2, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events (second run): unexpected error: %v", err)
	}
	res2, err := s.Execute(context.Background(), instrRequest("try again"))
	if err != nil {
		t.Fatalf("Execute(after fail): unexpected error: %v", err)
	}
	if res2.Status != agentruntime.RunCompleted {
		t.Fatalf("second turn: want completed, got %q", res2.Status)
	}

	// 流在首个终态事件(run.failed)后关闭;观察后续 run 需重新订阅。
	events := drainEvents(t, st)
	wantKinds := []agentruntime.EventKind{
		agentruntime.EventRunStarted,
		agentruntime.EventAgentMessage,
		agentruntime.EventUsageReported,
		agentruntime.EventRunFailed,
	}
	if len(events) != len(wantKinds) {
		t.Fatalf("event count: want %d, got %d: %+v", len(wantKinds), len(events), events)
	}
	for i, ev := range events {
		if ev.Kind != wantKinds[i] {
			t.Fatalf("event[%d]: want %s, got %s", i, wantKinds[i], ev.Kind)
		}
	}

	events2 := drainEvents(t, st2)
	wantKinds2 := []agentruntime.EventKind{
		agentruntime.EventRunStarted,
		agentruntime.EventAgentMessage,
		agentruntime.EventUsageReported,
		agentruntime.EventRunCompleted,
	}
	if len(events2) != len(wantKinds2) {
		t.Fatalf("event count (second run): want %d, got %d: %+v", len(wantKinds2), len(events2), events2)
	}
	for i, ev := range events2 {
		if ev.Kind != wantKinds2[i] {
			t.Fatalf("event[%d] (second run): want %s, got %s", i, wantKinds2[i], ev.Kind)
		}
	}
}

func TestTurnCrashClosesSession(t *testing.T) {
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnCrash}}})
	s := startSession(t, rt, defaultStart())

	// 崩溃的 turn 自述失败而非挂起(PRD §35: process crash handled)。
	res, err := s.Execute(context.Background(), instrRequest("explode"))
	if err != nil {
		t.Fatalf("Execute(crash turn): unexpected error: %v", err)
	}
	if res.Status != agentruntime.RunFailed || res.Summary != defaultSummaryCrash {
		t.Fatalf("crash turn claim: want failed/%q, got %q/%q", defaultSummaryCrash, res.Status, res.Summary)
	}

	// 崩溃 = 进程死亡:会话随之关闭,后续一切操作被拒绝。
	if _, err := s.Execute(context.Background(), instrRequest("again")); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Execute after crash: want ErrSessionClosed, got %v", err)
	}
	if _, err := s.Result(context.Background(), res.RunID); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Result after crash: want ErrSessionClosed, got %v", err)
	}
	if err := s.Cancel(context.Background(), res.RunID); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Cancel after crash: want ErrSessionClosed, got %v", err)
	}
	if _, err := s.Events(context.Background()); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("Events after crash: want ErrSessionClosed, got %v", err)
	}
}

func TestScriptTurnsConsumedInOrderThenFallback(t *testing.T) {
	rt := New(Config{
		Turns:       []TurnSpec{{Behavior: TurnFail}, {Behavior: TurnComplete, Summary: "second"}},
		DefaultTurn: TurnSpec{Behavior: TurnCrash},
	})
	s := startSession(t, rt, defaultStart())

	first, err := s.Execute(context.Background(), instrRequest("1"))
	if err != nil || first.Status != agentruntime.RunFailed {
		t.Fatalf("turn 1: want failed, got %q err=%v", first.Status, err)
	}
	second, err := s.Execute(context.Background(), instrRequest("2"))
	if err != nil || second.Status != agentruntime.RunCompleted || second.Summary != "second" {
		t.Fatalf("turn 2: want completed/\"second\", got %q/%q err=%v", second.Status, second.Summary, err)
	}
	// 脚本耗尽后回落到 DefaultTurn(此处模拟崩溃并关闭会话)。
	third, err := s.Execute(context.Background(), instrRequest("3"))
	if err != nil || third.Status != agentruntime.RunFailed || third.Summary != defaultSummaryCrash {
		t.Fatalf("turn 3 (fallback): want failed crash, got %q/%q err=%v", third.Status, third.Summary, err)
	}
	if _, err := s.Execute(context.Background(), instrRequest("4")); !errors.Is(err, agentruntime.ErrSessionClosed) {
		t.Fatalf("turn 4 after crash: want ErrSessionClosed, got %v", err)
	}
}

// --- AC4: Capabilities are explicit ---

func TestDescriptorDeclaresExplicitCapabilities(t *testing.T) {
	d := New(Config{}).Descriptor()
	if err := d.Validate(); err != nil {
		t.Fatalf("Descriptor.Validate: %v", err)
	}
	if d.Name != DescriptorName {
		t.Fatalf("Name: want %q, got %q", DescriptorName, d.Name)
	}
	got := d.Capabilities.Sorted()
	want := []agentruntime.Capability{
		agentruntime.CapEventStreaming,
		agentruntime.CapSessionIsolation,
	}
	if len(got) != len(want) {
		t.Fatalf("declared capabilities: want exactly %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("declared capabilities: want %v, got %v", want, got)
		}
	}
	// 高级 capability 必须全部缺席(声明是显式清单,不是开关集合)。
	for _, absent := range []agentruntime.Capability{
		agentruntime.CapSessionResume,
		agentruntime.CapSessionFork,
		agentruntime.CapSessionPersist,
		agentruntime.CapTurnSteer,
		agentruntime.CapTurnInterrupt,
		agentruntime.CapOutputStructured,
		agentruntime.CapSandboxNative,
		agentruntime.CapToolApproval,
		agentruntime.CapToolsCustom,
		agentruntime.CapModelSelect,
		agentruntime.CapMemoryDeclared,
	} {
		if d.Capabilities.Has(absent) {
			t.Fatalf("capability %s must stay undeclared by FakeRuntime", absent)
		}
	}
}

func TestDeclaredEventStreamingIsTruthful(t *testing.T) {
	// capability declaration truthful(PRD §35):声明了 event.streaming,
	// Events 就必须真的可用。
	rt := New(Config{})
	s := startSession(t, rt, defaultStart())
	st, err := s.Events(context.Background())
	if err != nil {
		t.Fatalf("Events: unexpected error: %v", err)
	}
	_ = st
}

// --- AC5: No native session tricks are assumed ---

func TestSessionIsolationNoCrossSessionRetrieval(t *testing.T) {
	// Session Isolation Canary 的协议可观察面:经协议注入 session A 的
	// 唯一 canary,在 session B 上不存在任何 Protocol-observable 检索路径。
	const canary = "canary-7f3a1e"
	rt := New(Config{})
	sA := startSession(t, rt, defaultStart())
	sB := startSession(t, rt, defaultStart())

	// canary 经 A 的 instruction 进入 A 的协议可观察状态(agent.message
	// 默认回显 instruction),保证检查不是空洞的。
	stA, err := sA.Events(context.Background())
	if err != nil {
		t.Fatalf("Events A: unexpected error: %v", err)
	}
	resA, err := sA.Execute(context.Background(), instrRequest("do secret "+canary+" work"))
	if err != nil {
		t.Fatalf("Execute A: unexpected error: %v", err)
	}
	var sawCanaryInA bool
	for _, ev := range drainEvents(t, stA) {
		if strings.Contains(ev.Text, canary) {
			sawCanaryInA = true
		}
	}
	if !sawCanaryInA {
		t.Fatalf("canary never entered session A observable state; check is vacuous")
	}

	// B 上经协议观察不到 canary:自己的 turn 产出与事件流都不含它。
	stB, err := sB.Events(context.Background())
	if err != nil {
		t.Fatalf("Events B: unexpected error: %v", err)
	}
	resB, err := sB.Execute(context.Background(), instrRequest("b"))
	if err != nil {
		t.Fatalf("Execute B: unexpected error: %v", err)
	}
	if strings.Contains(resB.Summary, canary) {
		t.Fatalf("canary leaked across sessions via result: %q", resB.Summary)
	}
	for _, ev := range drainEvents(t, stB) {
		if strings.Contains(ev.Text, canary) {
			t.Fatalf("canary leaked across sessions via event %s: %q", ev.Kind, ev.Text)
		}
	}

	// Run 身份以 Session 为界(协议:RunID 会话内唯一):B 检索同名
	// RunID 只能看到自己的 claim,看不到 A 的任何状态。
	snapB, err := sB.Result(context.Background(), resB.RunID)
	if err != nil {
		t.Fatalf("Result B: unexpected error: %v", err)
	}
	if snapB.SessionID != agentruntime.SessionID("session-0002") || snapB != resB {
		t.Fatalf("Result B: want B's own claim %+v, got %+v", resB, snapB)
	}
	if snapB == resA {
		t.Fatalf("Result B returned session A's claim: %+v", snapB)
	}
	// B 上不存在 A 独有的 run(无共享 run 表/无原生恢复通道)。
	if _, err := sB.Result(context.Background(), "run-9999"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("cross-session Result: want ErrRunNotFound, got %v", err)
	}
	if err := sB.Cancel(context.Background(), "run-9999"); !errors.Is(err, agentruntime.ErrRunNotFound) {
		t.Fatalf("cross-session Cancel: want ErrRunNotFound, got %v", err)
	}
}

func TestConcurrentSessionsExecuteIndependently(t *testing.T) {
	// 并发会话各自消费脚本;主要供 -race 验证内部状态无数据竞争。
	rt := New(Config{Turns: []TurnSpec{{Behavior: TurnComplete}}})
	sessions := make([]agentruntime.Session, 4)
	for i := range sessions {
		sessions[i] = startSession(t, rt, defaultStart())
	}
	done := make(chan error, len(sessions))
	for _, s := range sessions {
		go func(s agentruntime.Session) {
			_, err := s.Execute(context.Background(), instrRequest("parallel"))
			done <- err
		}(s)
	}
	for range sessions {
		if err := <-done; err != nil {
			t.Fatalf("Execute: unexpected error: %v", err)
		}
	}
}
