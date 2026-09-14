package agentruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- 测试专用最小适配器 -------------------------------------------------
//
// fakeRuntime 只为验证协议形状本身(AC1)存在,不承担 P1-24 参考适配器的
// 职责。它按声明能力被约束:未声明 event.streaming 时 Events 拒绝服务,
// 以此验证「能力是协商出来的数据,不是接口存在性」(AC2)。

type fakeRuntime struct {
	desc     RuntimeDescriptor
	mu       sync.Mutex
	nextSess int
	sessions map[SessionID]*fakeSession
}

func newFakeRuntime(caps ...Capability) *fakeRuntime {
	return &fakeRuntime{
		desc:     RuntimeDescriptor{Name: "fake", Capabilities: NewCapabilitySet(caps...)},
		sessions: map[SessionID]*fakeSession{},
	}
}

func (r *fakeRuntime) Descriptor() RuntimeDescriptor { return r.desc }

func (r *fakeRuntime) Start(ctx context.Context, req StartRequest) (Session, error) {
	if err := CheckStart(r.desc, req); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextSess++
	s := &fakeSession{
		rt:   r.desc,
		id:   SessionID(fmt.Sprintf("sess-%d", r.nextSess)),
		runs: map[RunID]*fakeRun{},
		subs: map[*fakeStream]struct{}{},
	}
	r.sessions[s.id] = s
	return s, nil
}

type fakeRun struct {
	id      RunID
	release chan struct{} // instruction == "block" 时等它放行
	cancel  chan struct{} // Cancel 唤醒 Execute
	mu      sync.Mutex
	result  AgentRunResult
}

type fakeSession struct {
	rt RuntimeDescriptor
	id SessionID

	mu      sync.Mutex
	closed  bool
	nextRun int
	seq     int64
	runs    map[RunID]*fakeRun
	subs    map[*fakeStream]struct{}
}

func (s *fakeSession) Execute(ctx context.Context, req ExecuteRequest) (AgentRunResult, error) {
	if err := req.Validate(); err != nil {
		return AgentRunResult{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return AgentRunResult{}, ErrSessionClosed
	}
	s.nextRun++
	run := &fakeRun{
		id:      RunID(fmt.Sprintf("run-%d", s.nextRun)),
		release: make(chan struct{}),
		cancel:  make(chan struct{}),
	}
	s.runs[run.id] = run
	s.mu.Unlock()

	s.emit(run, EventRunStarted, "", nil)

	var status RunStatus
	var summary string
	switch req.Instruction {
	case "block":
		select {
		case <-ctx.Done():
			status, summary = RunCancelled, "被取消的 ctx 终止"
		case <-run.cancel:
			status, summary = RunCancelled, "被 Cancel 终止"
		case <-run.release:
			status, summary = RunCompleted, "block 放行完成"
		}
	default:
		status, summary = RunCompleted, req.Instruction
	}
	return s.finish(run, status, summary), nil
}

func (s *fakeSession) Cancel(ctx context.Context, id RunID) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrSessionClosed
	}
	run, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return ErrRunNotFound
	}
	select {
	case <-run.cancel: // 已取消,幂等
	default:
		close(run.cancel)
	}
	return nil
}

func (s *fakeSession) Events(ctx context.Context) (EventStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrSessionClosed
	}
	if err := CheckEvents(s.rt); err != nil {
		return nil, err
	}
	st := &fakeStream{sess: s, ch: make(chan Event, 128)}
	s.subs[st] = struct{}{}
	return st, nil
}

func (s *fakeSession) Result(ctx context.Context, id RunID) (AgentRunResult, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return AgentRunResult{}, ErrSessionClosed
	}
	run, ok := s.runs[id]
	s.mu.Unlock()
	if !ok {
		return AgentRunResult{}, ErrRunNotFound
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	return run.result, nil
}

func (s *fakeSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrSessionClosed
	}
	s.closed = true
	for st := range s.subs {
		close(st.ch)
	}
	s.subs = map[*fakeStream]struct{}{}
	return nil
}

// finish 落终态:发语义事件(带用量),再发终态事件,记录 claim 结果。
func (s *fakeSession) finish(run *fakeRun, status RunStatus, summary string) AgentRunResult {
	var native *NativeEvent
	if status == RunCompleted {
		native = &NativeEvent{Type: "fake.message", Payload: json.RawMessage(`{"text":` + quote(summary) + `}`)}
	}
	s.emit(run, EventAgentMessage, summary, native)
	s.emit(run, EventUsageReported, "", &NativeEvent{
		Type:    "fake.usage",
		Payload: json.RawMessage(`{"input_tokens":11,"output_tokens":7}`),
	})
	terminal := EventRunCompleted
	switch status {
	case RunFailed:
		terminal = EventRunFailed
	case RunCancelled:
		terminal = EventRunCancelled
	}
	s.emit(run, terminal, "", nil)

	res := AgentRunResult{
		RunID:     run.id,
		SessionID: s.id,
		Status:    status,
		Summary:   summary,
		Usage:     Usage{InputTokens: 11, OutputTokens: 7},
	}
	run.mu.Lock()
	run.result = res
	run.mu.Unlock()
	return res
}

// emit 广播一条事件。非阻塞投递:事件是旁路观察,缓冲满即丢,
// 权威结果以 Execute 返回值/Result 为准。
func (s *fakeSession) emit(run *fakeRun, kind EventKind, text string, native *NativeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	ev := Event{
		Kind:      kind,
		SessionID: s.id,
		RunID:     run.id,
		Seq:       s.seq,
		Time:      time.Now(),
		Text:      text,
		Native:    native,
	}
	for st := range s.subs {
		select {
		case st.ch <- ev:
		default:
		}
	}
}

type fakeStream struct {
	sess *fakeSession
	ch   chan Event

	mu          sync.Mutex
	closed      bool
	sawTerminal bool
}

func (st *fakeStream) Next(ctx context.Context) (Event, error) {
	st.mu.Lock()
	closed, done := st.closed, st.sawTerminal
	st.mu.Unlock()
	if closed || done {
		return Event{}, ErrEventStreamClosed
	}
	select {
	case <-ctx.Done():
		return Event{}, ctx.Err()
	case ev, ok := <-st.ch:
		if !ok {
			return Event{}, ErrEventStreamClosed
		}
		if ev.Kind.Terminal() {
			st.mu.Lock()
			st.sawTerminal = true
			st.mu.Unlock()
		}
		return ev, nil
	}
}

func (st *fakeStream) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.closed {
		return nil
	}
	st.closed = true
	st.sess.mu.Lock()
	delete(st.sess.subs, st)
	st.sess.mu.Unlock()
	return nil
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ---- AC1:核心 descriptor/start/execute/cancel(events/result)----------

func TestRuntimeLifecycleDescriptorStartExecuteResult(t *testing.T) {
	rt := newFakeRuntime(CapEventStreaming)
	if err := rt.desc.Validate(); err != nil {
		t.Fatalf("描述符应合法: %v", err)
	}
	sess, err := rt.Start(context.Background(), StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	res, err := sess.Execute(context.Background(), ExecuteRequest{Instruction: "修复指纹检索"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Status != RunCompleted || res.RunID == "" || res.SessionID == "" {
		t.Fatalf("Execute 返回的 claim 不完整: %+v", res)
	}
	got, err := sess.Result(context.Background(), res.RunID)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if got != res {
		t.Fatalf("Result 应返回同一 claim 快照: got %+v want %+v", got, res)
	}
	if err := sess.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestExecuteRequestValidation(t *testing.T) {
	rt := newFakeRuntime()
	sess, err := rt.Start(context.Background(), StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := sess.Execute(context.Background(), ExecuteRequest{}); !errors.Is(err, ErrNoInstruction) {
		t.Fatalf("空 instruction 应返回 ErrNoInstruction,得到 %v", err)
	}
}

func TestCancelRunningAndTerminalRun(t *testing.T) {
	ctx := context.Background()
	rt := newFakeRuntime(CapEventStreaming)
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	stream, err := sess.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	defer stream.Close()

	resCh := make(chan AgentRunResult, 1)
	go func() {
		res, err := sess.Execute(ctx, ExecuteRequest{Instruction: "block"})
		if err != nil {
			t.Errorf("Execute: %v", err)
		}
		resCh <- res
	}()

	ev, err := stream.Next(ctx)
	if err != nil || ev.Kind != EventRunStarted {
		t.Fatalf("应先收到 run.started: ev=%+v err=%v", ev, err)
	}
	if err := sess.Cancel(ctx, RunID("nope")); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("未知 run 应返回 ErrRunNotFound,得到 %v", err)
	}
	if err := sess.Cancel(ctx, ev.RunID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := sess.Cancel(ctx, ev.RunID); err != nil {
		t.Fatalf("Cancel 对已取消 run 应幂等: %v", err)
	}
	res := <-resCh
	if res.Status != RunCancelled {
		t.Fatalf("被 Cancel 的 run 应返回 RunCancelled claim,得到 %+v", res)
	}
	snap, err := sess.Result(ctx, res.RunID)
	if err != nil || snap.Status != RunCancelled {
		t.Fatalf("Cancel 后 Result 应给出 RunCancelled: %+v err=%v", snap, err)
	}
	assertStreamTerminates(ctx, t, stream, res.RunID, EventRunCancelled)
}

func TestResultUnknownRun(t *testing.T) {
	rt := newFakeRuntime()
	sess, err := rt.Start(context.Background(), StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := sess.Result(context.Background(), RunID("missing")); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("未知 run 应返回 ErrRunNotFound,得到 %v", err)
	}
}

func TestSessionCloseRejectsFurtherWork(t *testing.T) {
	ctx := context.Background()
	rt := newFakeRuntime(CapEventStreaming)
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := sess.Close(ctx); err != nil {
		t.Fatalf("首次 Close 应成功: %v", err)
	}
	if err := sess.Close(ctx); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("重复 Close 应返回 ErrSessionClosed,得到 %v", err)
	}
	if _, err := sess.Execute(ctx, ExecuteRequest{Instruction: "x"}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("关闭后 Execute 应返回 ErrSessionClosed,得到 %v", err)
	}
	if _, err := sess.Events(ctx); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("关闭后 Events 应返回 ErrSessionClosed,得到 %v", err)
	}
	if _, err := sess.Result(ctx, RunID("run-1")); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("关闭后 Result 应返回 ErrSessionClosed,得到 %v", err)
	}
}

// ---- AC2:能力是协商出来的数据,不是 agent 名字开关 ----------------------

func TestCapabilityConstantsMatchPRD(t *testing.T) {
	// 字符串值是契约(PRD §10.3 原文);意外改名会静默破坏协商。
	want := map[Capability]string{
		CapSessionResume:    "session.resume",
		CapSessionFork:      "session.fork",
		CapSessionIsolation: "session.isolation",
		CapTurnSteer:        "turn.steer",
		CapTurnInterrupt:    "turn.interrupt",
		CapEventStreaming:   "event.streaming",
		CapOutputStructured: "output.structured",
		CapSandboxNative:    "sandbox.native",
		CapToolApproval:     "tool.approval",
		CapToolsCustom:      "tools.custom",
		CapModelSelect:      "model.select",
		CapSessionPersist:   "session.persist",
		CapMemoryDeclared:   "memory.declared",
	}
	for cap, s := range want {
		if string(cap) != s {
			t.Errorf("capability %q 应为 %q(PRD §10.3)", string(cap), s)
		}
	}
}

func TestNegotiateGrantsAndPartitionsPreferred(t *testing.T) {
	declared := NewCapabilitySet(CapEventStreaming, CapOutputStructured)
	got, err := Negotiate(declared, CapabilityRequirement{
		Required:  []Capability{CapEventStreaming},
		Preferred: []Capability{CapOutputStructured, CapSessionIsolation},
	})
	if err != nil {
		t.Fatalf("Negotiate: %v", err)
	}
	if want := []Capability{CapEventStreaming}; !reflect.DeepEqual(got.Granted, want) {
		t.Fatalf("Granted = %v, want %v", got.Granted, want)
	}
	if want := []Capability{CapOutputStructured}; !reflect.DeepEqual(got.PreferredGranted, want) {
		t.Fatalf("PreferredGranted = %v, want %v", got.PreferredGranted, want)
	}
	if want := []Capability{CapSessionIsolation}; !reflect.DeepEqual(got.PreferredAbsent, want) {
		t.Fatalf("PreferredAbsent = %v, want %v", got.PreferredAbsent, want)
	}
}

func TestNegotiateMissingRequiredIsStructuredError(t *testing.T) {
	got, err := Negotiate(
		NewCapabilitySet(CapEventStreaming),
		CapabilityRequirement{Required: []Capability{CapSessionIsolation, CapEventStreaming, CapModelSelect}},
	)
	var capErr *CapabilityError
	if !errors.As(err, &capErr) {
		t.Fatalf("缺必需能力应返回 *CapabilityError,得到 %v", err)
	}
	if want := []Capability{CapModelSelect, CapSessionIsolation}; !reflect.DeepEqual(capErr.Missing, want) {
		t.Fatalf("Missing = %v, want %v(字典序)", capErr.Missing, want)
	}
	if len(got.Granted)+len(got.PreferredGranted)+len(got.PreferredAbsent) != 0 {
		t.Fatalf("协商失败不得返回部分结果: %+v", got)
	}
	if !bytes.Contains([]byte(capErr.Error()), []byte("session.isolation")) {
		t.Fatalf("错误文本应列出缺失能力,得到 %q", capErr.Error())
	}
}

func TestNegotiationDependsOnCapabilitiesNotRuntimeName(t *testing.T) {
	// 同能力不同名 → 协商结果完全一致;同名不同能力 → 结果不同。
	// 这是 AC2 的回归钉:控制流只能来自 CapabilitySet。
	req := CapabilityRequirement{
		Required:  []Capability{CapEventStreaming},
		Preferred: []Capability{CapOutputStructured},
	}
	byName := map[string]RuntimeDescriptor{
		"fake":  {Name: "fake", Capabilities: NewCapabilitySet(CapEventStreaming)},
		"codex": {Name: "codex", Capabilities: NewCapabilitySet(CapEventStreaming)},
	}
	var results []NegotiatedCapabilities
	for _, name := range []string{"fake", "codex"} {
		got, err := Negotiate(byName[name].Capabilities, req)
		if err != nil {
			t.Fatalf("Negotiate(%s): %v", name, err)
		}
		results = append(results, got)
	}
	if !reflect.DeepEqual(results[0], results[1]) {
		t.Fatalf("同能力不同名应得到相同协商结果: %+v vs %+v", results[0], results[1])
	}
	bareFake := RuntimeDescriptor{Name: "fake", Capabilities: NewCapabilitySet()}
	if _, err := Negotiate(bareFake.Capabilities, req); err == nil {
		t.Fatal("同名的 fake 无 event.streaming 时协商必须失败")
	}
}

func TestStartRequestRequiredCapabilities(t *testing.T) {
	cases := []struct {
		name string
		req  StartRequest
		want []Capability
	}{
		{"基础请求无额外要求", StartRequest{Workspace: "w"}, []Capability{}},
		{"恢复会话需要 session.resume", StartRequest{Workspace: "w", ResumeOf: "s1"}, []Capability{CapSessionResume}},
		{"模型档位需要 model.select", StartRequest{Workspace: "w", ModelProfile: "economical"}, []Capability{CapModelSelect}},
		{"两者按字典序", StartRequest{Workspace: "w", ResumeOf: "s1", ModelProfile: "economical"},
			[]Capability{CapModelSelect, CapSessionResume}},
	}
	for _, tc := range cases {
		if got := tc.req.RequiredCapabilities(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: RequiredCapabilities = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCheckStartGatesRequestAndCapabilities(t *testing.T) {
	desc := RuntimeDescriptor{Name: "fake", Capabilities: NewCapabilitySet(CapSessionResume)}
	if err := CheckStart(desc, StartRequest{}); !errors.Is(err, ErrNoWorkspace) {
		t.Fatalf("缺 workspace 应返回 ErrNoWorkspace,得到 %v", err)
	}
	if err := CheckStart(desc, StartRequest{Workspace: "w", ResumeOf: "s1"}); err != nil {
		t.Fatalf("声明 session.resume 时应放行: %v", err)
	}
	err := CheckStart(desc, StartRequest{Workspace: "w", ModelProfile: "economical"})
	var capErr *CapabilityError
	if !errors.As(err, &capErr) {
		t.Fatalf("未声明 model.select 应被 CheckStart 拒绝,得到 %v", err)
	}
	if err := CheckStart(RuntimeDescriptor{}, StartRequest{Workspace: "w"}); !errors.Is(err, ErrNoRuntimeName) {
		t.Fatalf("无名描述符应返回 ErrNoRuntimeName,得到 %v", err)
	}
}

func TestEventsRequireDeclaredStreamingCapability(t *testing.T) {
	ctx := context.Background()
	// FakeRuntime 应故意缺失部分能力(PRD §10.4):这里缺 event.streaming。
	rt := newFakeRuntime(CapSessionPersist)
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := sess.Events(ctx); !errors.Is(err, ErrStreamingNotDeclared) {
		t.Fatalf("未声明 event.streaming 应返回 ErrStreamingNotDeclared,得到 %v", err)
	}
}

// ---- AC3:AgentRunResult 明确是 claim,不是 TaskResult/Fact -------------

func TestAgentRunResultIsClaimNotFact(t *testing.T) {
	// 类型名是 AC3 的机制之一:它必须叫 AgentRunResult(claim),
	// 不得演化成 TaskResult/Fact 之类的事实化命名(INV-AUTH-01)。
	if got := reflect.TypeOf(AgentRunResult{}).Name(); got != "AgentRunResult" {
		t.Fatalf("结果类型必须保持命名 AgentRunResult,得到 %q", got)
	}
	forbidden := map[string]bool{"TaskResult": true, "Fact": true, "TaskFact": true, "AgentFact": true}
	found := map[string]bool{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("读包目录: %v", err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatalf("解析 %s: %v", e.Name(), err)
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				found[ts.Name.Name] = true
				if forbidden[ts.Name.Name] {
					t.Errorf("包内不得定义事实化类型 %s(INV-AUTH-01)", ts.Name.Name)
				}
			}
		}
	}
	if !found["AgentRunResult"] {
		t.Fatal("包内应定义 AgentRunResult 类型")
	}
}

// ---- AC4:规范化事件保留可选原生形态 ------------------------------------

func TestNormalizedEventsPreserveOptionalNative(t *testing.T) {
	ctx := context.Background()
	rt := newFakeRuntime(CapEventStreaming)
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	stream, err := sess.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	defer stream.Close()

	res, err := sess.Execute(ctx, ExecuteRequest{Instruction: "原生事件保真"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var events []Event
	for {
		ev, err := stream.Next(ctx)
		if err != nil {
			break
		}
		events = append(events, ev)
	}
	wantKinds := []EventKind{EventRunStarted, EventAgentMessage, EventUsageReported, EventRunCompleted}
	if len(events) != len(wantKinds) {
		t.Fatalf("应收到 %d 条事件,得到 %d: %+v", len(wantKinds), len(events), events)
	}
	for i, ev := range events {
		if ev.Kind != wantKinds[i] {
			t.Fatalf("第 %d 条事件应为 %s,得到 %s", i, wantKinds[i], ev.Kind)
		}
		if ev.RunID != res.RunID || ev.SessionID != res.SessionID {
			t.Fatalf("事件归属错误: %+v", ev)
		}
		if ev.Seq != int64(i+1) {
			t.Fatalf("Seq 应会话内单调: got %d want %d", ev.Seq, i+1)
		}
	}

	msg := events[1]
	if !msg.HasNative() || msg.Native.Type != "fake.message" {
		t.Fatalf("message 事件应保留原生类型: %+v", msg.Native)
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Native.Payload, &payload); err != nil {
		t.Fatalf("原生载荷应是合法 JSON: %v", err)
	}
	if payload.Text != res.Summary {
		t.Fatalf("原生载荷应字节保真: got %q want %q", payload.Text, res.Summary)
	}

	usage := events[2]
	if !usage.HasNative() || usage.Native.Type != "fake.usage" {
		t.Fatalf("usage 事件应保留原生类型: %+v", usage.Native)
	}

	// JSON 往返后原生形态不丢(AC4 的持久化前提)。
	wire, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal Event: %v", err)
	}
	var back Event
	if err := json.Unmarshal(wire, &back); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}
	if !back.HasNative() || back.Native.Type != msg.Native.Type || !bytes.Equal(back.Native.Payload, msg.Native.Payload) {
		t.Fatalf("JSON 往返应保留原生形态: got %+v want %+v", back.Native, msg.Native)
	}
}

func TestEventStreamTerminatesAfterTerminalEvent(t *testing.T) {
	ctx := context.Background()
	rt := newFakeRuntime(CapEventStreaming)
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	stream, err := sess.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if _, err := sess.Execute(ctx, ExecuteRequest{Instruction: "正常完成"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	assertStreamTerminates(ctx, t, stream, RunID("run-1"), EventRunCompleted)
	if err := stream.Close(); err != nil {
		t.Fatalf("Close 应幂等成功: %v", err)
	}
	if _, err := stream.Next(ctx); !errors.Is(err, ErrEventStreamClosed) {
		t.Fatalf("Close 后 Next 应返回 ErrEventStreamClosed,得到 %v", err)
	}
}

func TestStreamNextRespectsContextCancellation(t *testing.T) {
	ctx := context.Background()
	rt := newFakeRuntime(CapEventStreaming)
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	stream, err := sess.Events(ctx)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	defer stream.Close()

	waitCtx, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if _, err := stream.Next(waitCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx 取消后 Next 应返回 ctx.Err(),得到 %v", err)
	}
}

// ---- P0-01:实际身份观察(可选,能力安全)--------------------------------

// reportingSession 包装一个 Session 并自报固定身份,验证 IdentityReporter
// 的实现侧形状。生产适配器按各自实际观察实现,不受此测试约束。
type reportingSession struct {
	Session
	id ObservedIdentity
}

func (s *reportingSession) Identity() ObservedIdentity { return s.id }

func TestIdentityReporterIsOptionalCapabilitySafe(t *testing.T) {
	ctx := context.Background()
	rt := newFakeRuntime()
	sess, err := rt.Start(ctx, StartRequest{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	// 缺席是合法状态:不实现 IdentityReporter 的会话(如 FakeRuntime)
	// 断言失败,调用方走零值降级,不报错(AC3)。
	if rep, ok := sess.(IdentityReporter); ok {
		t.Fatalf("未实现 IdentityReporter 的会话不得断言成功: %T", rep)
	}
	var zero ObservedIdentity
	if zero.Observed() {
		t.Fatal("零值 ObservedIdentity 必须报告未观察")
	}

	// 部分观察:只报告已实际观察到的字段,其余保持零值(不补齐)。
	part := &reportingSession{Session: sess, id: ObservedIdentity{RuntimeVersion: "0.153.0"}}
	if got := part.Identity(); got != (ObservedIdentity{RuntimeVersion: "0.153.0"}) {
		t.Fatalf("部分观察身份 = %+v", got)
	}
	if !part.Identity().Observed() {
		t.Fatal("含观察事实的身份必须报告已观察")
	}

	// 完整观察:runtime/模型/会话/版本四项各自独立携带。
	full := &reportingSession{Session: sess, id: ObservedIdentity{
		RuntimeName:    "observed-cli",
		RuntimeVersion: "9.9.9",
		Model:          "actual-model",
		SessionID:      "sess-1",
	}}
	if got := full.Identity(); !got.Observed() || got.Model != "actual-model" || got.SessionID != "sess-1" {
		t.Fatalf("完整观察身份 = %+v", got)
	}
}

// ---- 断言辅助 -----------------------------------------------------------

// assertStreamTerminates 断言事件流按序走到终态事件,随后关闭。
func assertStreamTerminates(ctx context.Context, t *testing.T, stream EventStream, runID RunID, terminal EventKind) {
	t.Helper()
	for {
		ev, err := stream.Next(ctx)
		if err != nil {
			t.Fatalf("终态事件 %s 之前不应出错: %v", terminal, err)
		}
		if ev.RunID != runID {
			t.Fatalf("事件归属错误: %+v", ev)
		}
		if ev.Kind == terminal {
			break
		}
		if ev.Kind.Terminal() {
			t.Fatalf("应先收到 %s 而不是 %s", terminal, ev.Kind)
		}
	}
	if _, err := stream.Next(ctx); !errors.Is(err, ErrEventStreamClosed) {
		t.Fatalf("终态后 Next 应返回 ErrEventStreamClosed,得到 %v", err)
	}
}
