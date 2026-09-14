package runtimeconformance

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
)

// ---- Session Isolation Canary 正向验证(AC1/AC3) ---------------------------

// TestSessionIsolationCanaryPassesConformingRuntime:声明 session.isolation 且
// 协议合规的 runtime 必须通过 canary,且报告携带 declared residual risk 声明。
func TestSessionIsolationCanaryPassesConformingRuntime(t *testing.T) {
	rep := RunSessionIsolationCanary(t, newCFRuntime(cfOptions{caps: []agentruntime.Capability{
		agentruntime.CapEventStreaming,
		agentruntime.CapSessionIsolation,
	}}), SessionIsolationOptions{Timeout: 10 * time.Second})
	if !rep.CapabilityAvailable {
		t.Fatalf("合规 runtime 应通过 Session Isolation Canary: %s", rep.FailureText())
	}
	if !rep.CapabilityDeclared {
		t.Errorf("CapabilityDeclared 应为 true: %+v", rep)
	}
	if !rep.InjectionObserved {
		t.Errorf("回显型 runtime 的注入应经协议可观测: %+v", rep)
	}
	if rep.UnavailableReason != "" {
		t.Errorf("通过时不应有 UnavailableReason: %q", rep.UnavailableReason)
	}
	if len(rep.Failures) != 0 {
		t.Errorf("通过时不应有失败详情: %s", rep.FailureText())
	}
	if rep.ResidualRisk != ResidualRiskNote {
		t.Errorf("报告必须携带 declared residual risk 声明(AC3): %q", rep.ResidualRisk)
	}
}

// TestCanaryTokenUniquePerRun:canary 必须每次运行唯一,否则令牌可能命中
// runtime 训练语料/既有状态而非本次注入。
func TestCanaryTokenUniquePerRun(t *testing.T) {
	a, b := newCanary(), newCanary()
	if a == b {
		t.Fatalf("canary 必须每次运行唯一: %s", a)
	}
	if !strings.HasPrefix(a, "TALLY-CANARY-") || !strings.HasPrefix(b, "TALLY-CANARY-") {
		t.Fatalf("canary 应带可识别前缀: %q %q", a, b)
	}
}

// ---- Session Isolation Canary 反向验证(AC2) -------------------------------
//
// leakRuntime 是故意跨会话泄漏 canary 的最小 runtime:注入会话记录令牌,
// 检索会话按 leakMode 经指定协议通道把它交出去,用于验证 canary 的裁决
// 方向(任何检索成功 = capability unavailable)。

// leakMode 选择泄漏通道;三个通道覆盖协议面上的全部检索面。
type leakMode uint8

const (
	leakSummary leakMode = iota // 经 Execute 返回的 claim summary 泄漏
	leakResult                  // summary 干净,经 Result 快照泄漏
	leakEvents                  // summary 干净,经事件流 agent.message 泄漏
)

// leakRuntime 声明 session.isolation + event.streaming(泄漏的前提是声明了
// 对应能力),会话间共享一个 secret。
type leakRuntime struct {
	mode leakMode
	caps []agentruntime.Capability

	mu     sync.Mutex
	next   int
	secret string
}

func (r *leakRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{
		Name:         "leaky",
		Version:      "test",
		Capabilities: agentruntime.NewCapabilitySet(r.caps...),
	}
}

func (r *leakRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	if err := agentruntime.CheckStart(r.Descriptor(), req); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	return &leakSession{
		rt:   r,
		id:   agentruntime.SessionID(fmt.Sprintf("leak-%d", r.next)),
		subs: map[*leakStream]struct{}{},
	}, nil
}

type leakSession struct {
	rt *leakRuntime
	id agentruntime.SessionID

	mu     sync.Mutex
	seq    int64
	last   agentruntime.AgentRunResult
	subs   map[*leakStream]struct{}
	closed bool
}

func (s *leakSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	if err := req.Validate(); err != nil {
		return agentruntime.AgentRunResult{}, err
	}
	res := agentruntime.AgentRunResult{
		RunID:     "leak-run",
		SessionID: s.id,
		Status:    agentruntime.RunCompleted,
		Summary:   "leak: nothing to recall",
	}
	snap := res
	text := "ack"
	switch {
	case strings.Contains(req.Instruction, canaryInjectPrefix):
		// 注入回合:记录令牌(注入协议把令牌放在指令末尾)。
		s.rt.mu.Lock()
		s.rt.secret = leakLastField(req.Instruction)
		s.rt.mu.Unlock()
	case strings.Contains(req.Instruction, canaryRetrievalPrefix):
		// 检索回合:按模式把令牌交到指定协议通道。
		s.rt.mu.Lock()
		secret := s.rt.secret
		s.rt.mu.Unlock()
		if secret != "" {
			recalled := "leak: recalled " + secret
			switch s.rt.mode {
			case leakSummary:
				res.Summary = recalled
				snap = res
			case leakResult:
				snap.Summary = recalled
			case leakEvents:
				text = recalled
			}
		}
	}
	s.mu.Lock()
	s.last = snap
	s.mu.Unlock()
	s.emit(res.RunID, agentruntime.EventRunStarted, "")
	s.emit(res.RunID, agentruntime.EventAgentMessage, text)
	s.emit(res.RunID, agentruntime.EventRunCompleted, "")
	return res, nil
}

func (s *leakSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last.RunID == "" || s.last.RunID != id {
		return agentruntime.AgentRunResult{}, agentruntime.ErrRunNotFound
	}
	return s.last, nil
}

func (s *leakSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	if err := agentruntime.CheckEvents(s.rt.Descriptor()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, agentruntime.ErrSessionClosed
	}
	st := &leakStream{sess: s, ch: make(chan agentruntime.Event, 64)}
	s.subs[st] = struct{}{}
	return st, nil
}

func (s *leakSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return agentruntime.ErrRunNotFound
}

func (s *leakSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return agentruntime.ErrSessionClosed
	}
	s.closed = true
	for st := range s.subs {
		delete(s.subs, st)
		close(st.ch)
	}
	return nil
}

// emit 广播一条事件;缓冲(64)远大于单回合事件数(3),不会丢弃。
func (s *leakSession) emit(runID agentruntime.RunID, kind agentruntime.EventKind, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	ev := agentruntime.Event{
		Kind:      kind,
		SessionID: s.id,
		RunID:     runID,
		Seq:       s.seq,
		Time:      time.Now(),
		Text:      text,
	}
	for st := range s.subs {
		select {
		case st.ch <- ev:
		default:
		}
	}
}

// leakStream 是 leakSession 的单消费者事件流;Close 关闭底层通道。
type leakStream struct {
	sess *leakSession
	ch   chan agentruntime.Event
}

func (st *leakStream) Next(ctx context.Context) (agentruntime.Event, error) {
	select {
	case <-ctx.Done():
		return agentruntime.Event{}, ctx.Err()
	case ev, ok := <-st.ch:
		if !ok {
			return agentruntime.Event{}, agentruntime.ErrEventStreamClosed
		}
		return ev, nil
	}
}

func (st *leakStream) Close() error {
	st.sess.mu.Lock()
	defer st.sess.mu.Unlock()
	if _, ok := st.sess.subs[st]; !ok {
		return nil
	}
	delete(st.sess.subs, st)
	close(st.ch)
	return nil
}

// leakLastField 返回文本最后一个空白分隔字段(注入协议把令牌放在末尾)。
func leakLastField(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return f[len(f)-1]
}

// TestSessionIsolationCanaryCatchesLeaksAcrossProtocolChannels:经任一
// protocol-observable 通道检索成功都必须把 conformance capability 判为
// unavailable(AC1/AC2)。
func TestSessionIsolationCanaryCatchesLeaksAcrossProtocolChannels(t *testing.T) {
	tests := []struct {
		name        string
		mode        leakMode
		wantChannel string // 失败详情应指明的泄漏通道
	}{
		{"claim summary", leakSummary, "claim summary"},
		{"Result snapshot", leakResult, "Result 快照"},
		{"event stream", leakEvents, "事件流"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			rt := &leakRuntime{mode: tt.mode, caps: []agentruntime.Capability{
				agentruntime.CapEventStreaming,
				agentruntime.CapSessionIsolation,
			}}
			rep := CheckSessionIsolation(context.Background(), rt, SessionIsolationOptions{Timeout: 10 * time.Second})
			if rep.CapabilityAvailable {
				t.Fatalf("泄漏 runtime 必须被判 capability unavailable(AC2): %+v", rep)
			}
			if rep.UnavailableReason != "" {
				t.Errorf("探针已运行,不可用应由 Failures 承载而非 UnavailableReason: %q", rep.UnavailableReason)
			}
			if len(rep.Failures) == 0 {
				t.Fatal("泄漏必须记录失败详情")
			}
			if txt := rep.FailureText(); !strings.Contains(txt, tt.wantChannel) {
				t.Errorf("失败详情应指明泄漏通道 %q: %s", tt.wantChannel, txt)
			}
			if txt := rep.FailureText(); !strings.Contains(txt, "("+roleCoding+" 会话)") {
				t.Errorf("失败详情应归因到 coding 检索会话: %s", txt)
			}
			if rep.ResidualRisk != ResidualRiskNote {
				t.Errorf("失败报告同样必须携带 residual risk 声明(AC3): %q", rep.ResidualRisk)
			}
		})
	}
}

// TestSessionIsolationCanaryUndeclaredCapabilityUnavailable:未声明
// session.isolation 的 runtime 不可 qualified(必要条件不成立),canary
// 探针不得运行(能力专项场景按声明门控)。
func TestSessionIsolationCanaryUndeclaredCapabilityUnavailable(t *testing.T) {
	rt := newCFRuntime(cfOptions{})
	rep := CheckSessionIsolation(context.Background(), rt, SessionIsolationOptions{Timeout: 10 * time.Second})
	if rep.CapabilityAvailable || rep.CapabilityDeclared {
		t.Fatalf("未声明 session.isolation 时 capability 必须 unavailable: %+v", rep)
	}
	if rep.UnavailableReason == "" {
		t.Fatal("应说明 capability unavailable 的原因")
	}
	if len(rep.Failures) != 0 {
		t.Errorf("探针未运行,不应产生检索失败: %s", rep.FailureText())
	}
	if rt.next != 0 {
		t.Errorf("未声明能力时不得开启任何会话: started %d", rt.next)
	}
	if rep.ResidualRisk != ResidualRiskNote {
		t.Errorf("报告必须携带 declared residual risk 声明(AC3): %q", rep.ResidualRisk)
	}
}

// TestSessionIsolationCanaryDeclaresResidualRisk(AC3):任何裁决 —— 包括
// nil runtime 的退化路径 —— 都原样携带 declared residual risk 声明。
func TestSessionIsolationCanaryDeclaresResidualRisk(t *testing.T) {
	rep := CheckSessionIsolation(context.Background(), nil, SessionIsolationOptions{})
	if rep.CapabilityAvailable {
		t.Fatal("nil runtime 不可 qualified")
	}
	if rep.UnavailableReason == "" {
		t.Fatal("nil runtime 应记录不可用原因")
	}
	if rep.ResidualRisk != ResidualRiskNote {
		t.Fatalf("报告必须携带 declared residual risk 声明(AC3): %q", rep.ResidualRisk)
	}
	if !strings.Contains(rep.ResidualRisk, "declared residual risk") {
		t.Errorf("residual risk 声明必须点名 declared residual risk: %q", rep.ResidualRisk)
	}
}
