// Package fake 提供 FakeRuntime —— PRD §10.4 规定的 P1 首版参考实现:
//
//	FakeRuntime   - conformance / abstraction test
//	CodexRuntime  - production runtime
//
// 设计目标(任务 P1-24)是「故意简单、确定性的 reference/conformance 目标,
// 不携带任何 Codex-specific 假设」:
//
//   - 没有进程、没有网络、没有文件系统访问、没有原生 session/thread 机制
//     (AC5):一个 Session 只是本进程内一个独立的有界状态对象,事件与结果
//     全部由显式脚本(Config.Turns)确定,不解析任何指令文本魔法。
//   - 能力显式声明(Capabilities,AC4),并且故意缺少全部高级 capability
//     (PRD §10.4):核心接口之外的任何行为都必须先通过协商,缺声明即拒绝,
//     以此证明 Core 没有泄漏按 runtime 名字/隐式能力分支的假设。
//   - 产出的一切(AgentRunResult/Event)都只是 Agent Claim
//     (INV-AUTH-01),本包不做任何事实化解释。
//   - 支持确定性注入失败/崩溃/用量(AC3),供 conformance(PRD §35)与
//     上层降级路径测试使用,无需真实 LLM。
package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tally/internal/agentruntime"
)

// DescriptorName 是 FakeRuntime 的 runtime 标识。它只用于展示/遥测/归因;
// 任何控制流都不得按它分支(PRD §10.3)。
const DescriptorName = "fake"

// DescriptorVersion 是 FakeRuntime 适配器版本。
const DescriptorVersion = "1"

// 默认确定性用量(AC3):脚本未指定 Usage 时每个完成的 turn 自报这份固定值,
// 让「usage recorded」路径无需配置即可被 conformance 覆盖。
var defaultUsage = agentruntime.Usage{InputTokens: 10, OutputTokens: 5}

// 各终态的确定性默认 Summary(runtime 自述文本,原样保留,不做事实化解释)。
const (
	defaultSummaryCompleted = "fake: turn completed"
	defaultSummaryFailed    = "fake: turn failed"
	defaultSummaryCrash     = "fake: simulated runtime crash"
	defaultSummaryCancelled = "fake: run cancelled"
)

// Capabilities 返回 FakeRuntime 显式声明的能力集合。
//
// 只声明两条,且都真实可实现:
//
//   - event.streaming:Session.Events 提供真实的会话事件流;
//   - session.isolation:每个 Session 是互相独立的有界状态对象,不存在
//     任何跨 session 的 Protocol-observable 检索路径,Session Isolation
//     Canary(PRD §10.4/§35)在协议可观察范围内成立。
//
// 其余全部高级 capability(session.resume/fork、turn.steer/interrupt、
// output.structured、sandbox.native、tool.approval、tools.custom、
// model.select、session.persist、memory.declared)故意不声明(PRD §10.4:
// FakeRuntime 应故意缺失部分高级 capability,确保 Core 未泄漏
// Codex-specific assumption):带相应需求的 Start 必须在协商处被拒绝,
// 而不是被隐式放行。
func Capabilities() agentruntime.CapabilitySet {
	return agentruntime.NewCapabilitySet(
		agentruntime.CapEventStreaming,
		agentruntime.CapSessionIsolation,
	)
}

// TurnBehavior 选择一个被脚本化 turn 的确定性结局(AC1/AC3)。
type TurnBehavior uint8

const (
	// TurnComplete(零值)是默认结局:正常完成。
	TurnComplete TurnBehavior = iota
	// TurnFail 让 runtime 自述本轮失败(RunFailed),Session 仍可用。
	TurnFail
	// TurnCrash 模拟 runtime 进程崩溃:本轮以 RunFailed 终止,且 Session
	// 随之关闭(后续一切操作返回 ErrSessionClosed),用于验证崩溃处理
	// (PRD §35:process crash handled)。
	TurnCrash
	// TurnWaitCancel 让本轮阻塞,直到 Cancel/ctx 取消/会话关闭,以
	// RunCancelled 终止;用于确定性测试 Cancel 路径。
	TurnWaitCancel
)

// TurnSpec 描述脚本中一个 turn 的确定性结局。零值 = 正常完成 + 默认
// Summary + 默认用量。
type TurnSpec struct {
	// Behavior 指定结局类型,默认 TurnComplete。
	Behavior TurnBehavior
	// Summary 覆盖 AgentRunResult.Summary 的确定性默认值;空则用默认。
	Summary string
	// Message 覆盖 agent.message 事件文本;空则原样回显本轮 instruction
	// (证明 instruction 确实到达 runtime)。
	Message string
	// Usage 覆盖自报用量;nil 则用确定性默认值(defaultUsage)。
	Usage *agentruntime.Usage
}

// Config 是 FakeRuntime 的确定性脚本。Config 与其中切片在 New 之后必须
// 保持只读;每个 Session 从脚本头开始独立消费,互不影响。
type Config struct {
	// Turns 按顺序给出每个 Execute turn 的结局;脚本耗尽后用 DefaultTurn。
	Turns []TurnSpec
	// DefaultTurn 是脚本耗尽后所有 turn 的结局(零值 = 正常完成)。
	DefaultTurn TurnSpec
	// Now 返回事件时间戳;nil 则用 time.Now。测试可注入固定时钟获得完全
	// 确定性输出。
	Now func() time.Time
}

// Runtime 是 FakeRuntime 的 agentruntime.Runtime 实现。
type Runtime struct {
	cfg Config

	mu       sync.Mutex
	sessions int
}

// New 构造 FakeRuntime。零值 Config 合法且可用。
func New(cfg Config) *Runtime {
	return &Runtime{cfg: cfg}
}

// Descriptor 返回自述。声明能力见 Capabilities。
func (r *Runtime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{
		Name:         DescriptorName,
		Version:      DescriptorVersion,
		Capabilities: Capabilities(),
	}
}

// Start 开启会话。统一走 agentruntime.CheckStart:先做结构校验,再按请求
// 推导的必需能力与声明能力协商 —— 例如带 ModelProfile 的请求会因未声明
// model.select 被 *CapabilityError 拒绝,而不是被隐式放行(AC4)。
func (r *Runtime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	if err := agentruntime.CheckStart(r.Descriptor(), req); err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.sessions++
	id := agentruntime.SessionID(fmt.Sprintf("session-%04d", r.sessions))
	cfg := r.cfg
	r.mu.Unlock()

	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	s := &fakeSession{
		id:        id,
		workspace: req.Workspace,
		cfg:       cfg,
		now:       now,
		caps:      Capabilities(),
		runs:      make(map[agentruntime.RunID]*fakeRun),
		notify:    make(chan struct{}),
		closeCh:   make(chan struct{}),
	}
	return s, nil
}

// fakeRun 是会话内一次运行的有界状态。status/result 只在持有会话锁时读写;
// done 由 Cancel 关闭,用于唤醒阻塞在 TurnWaitCancel 上的 Execute。
type fakeRun struct {
	id        agentruntime.RunID
	status    agentruntime.RunStatus
	result    agentruntime.AgentRunResult
	hasResult bool
	done      chan struct{}
}

// fakeSession 是 agentruntime.Session 的 FakeRuntime 实现:一个互斥锁保护
// 的有界状态对象,没有任何跨 session 共享或持久化机制(AC5)。
type fakeSession struct {
	id        agentruntime.SessionID
	workspace string
	cfg       Config
	now       func() time.Time
	caps      agentruntime.CapabilitySet

	mu      sync.Mutex
	turnIdx int
	runs    map[agentruntime.RunID]*fakeRun
	runSeq  int
	seq     int64
	events  []agentruntime.Event
	// notify 是广播通道:每次追加事件后关闭旧通道并换新,唤醒事件流等待者。
	notify chan struct{}
	closed bool
	// closeCh 在会话关闭时关闭一次,唤醒阻塞中的 Execute(TurnWaitCancel)。
	closeCh chan struct{}
}

// ID 返回会话标识(供测试/conformance 断言;不属于核心协议)。
func (s *fakeSession) ID() agentruntime.SessionID { return s.id }

// Workspace 返回 Start 时收到的 workspace(供测试/conformance 断言
// 「receives cwd/workspace」,PRD §35;不属于核心协议)。
func (s *fakeSession) Workspace() string { return s.workspace }

// Execute 同步执行一个 turn。事件全部由本 goroutine 按固定顺序投递:
// run.started → [agent.message → usage.reported] → 终态事件,Seq 会话内
// 单调递增(AC2)。取消(ctx 取消或 Cancel)是正常终态:返回 RunCancelled
// 的结果而非 error;只有请求非法或会话已关闭才返回 error。
func (s *fakeSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	if err := req.Validate(); err != nil {
		return agentruntime.AgentRunResult{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	spec := s.cfg.DefaultTurn
	if s.turnIdx < len(s.cfg.Turns) {
		spec = s.cfg.Turns[s.turnIdx]
	}
	s.turnIdx++
	s.runSeq++
	id := agentruntime.RunID(fmt.Sprintf("run-%04d", s.runSeq))
	run := &fakeRun{id: id, status: agentruntime.RunRunning, done: make(chan struct{})}
	s.runs[id] = run
	s.emitLocked(agentruntime.EventRunStarted, id, "")
	s.mu.Unlock()

	res := s.runTurn(ctx, run, spec, req.Instruction)

	s.mu.Lock()
	run.status = res.Status
	run.result = res
	run.hasResult = true
	s.mu.Unlock()
	return res, nil
}

// runTurn 按 spec 执行 turn 主体并投递后续事件,返回最终 claim。
func (s *fakeSession) runTurn(ctx context.Context, run *fakeRun, spec TurnSpec, instruction string) agentruntime.AgentRunResult {
	if spec.Behavior == TurnWaitCancel {
		// 阻塞直到被取消/会话关闭;等待期间不投递任何事件,保证
		// 「订阅后先收到 run.started」的确定性窗口可测。
		select {
		case <-run.done:
		case <-ctx.Done():
		case <-s.closeCh:
		}
		return s.finishRun(run, agentruntime.RunCancelled, defaultSummaryCancelled, nil)
	}

	status := agentruntime.RunCompleted
	summary := defaultSummaryCompleted
	usage := defaultUsage
	if spec.Usage != nil {
		usage = *spec.Usage
	}
	switch spec.Behavior {
	case TurnFail:
		status = agentruntime.RunFailed
		summary = defaultSummaryFailed
	case TurnCrash:
		status = agentruntime.RunFailed
		summary = defaultSummaryCrash
	}
	if spec.Summary != "" {
		summary = spec.Summary
	}

	id := run.id
	s.mu.Lock()
	s.emitLocked(agentruntime.EventAgentMessage, id, messageText(spec, instruction))
	s.emitLocked(agentruntime.EventUsageReported, id, usageText(usage))
	s.mu.Unlock()

	res := s.finishRun(run, status, summary, &usage)
	if spec.Behavior == TurnCrash {
		// 崩溃 = 进程死亡:关闭会话,阻塞中的其它 run 一并被终止。
		s.Close(ctx)
	}
	return res
}

// finishRun 投递终态事件并组装 AgentRunResult(claim)。usage 为 nil 表示
// 中途被取消、未产出用量。
func (s *fakeSession) finishRun(run *fakeRun, status agentruntime.RunStatus, summary string, usage *agentruntime.Usage) agentruntime.AgentRunResult {
	id := run.id
	s.mu.Lock()
	s.emitLocked(terminalKind(status), id, "")
	s.mu.Unlock()

	res := agentruntime.AgentRunResult{
		RunID:     id,
		SessionID: s.id,
		Status:    status,
		Summary:   summary,
	}
	if usage != nil {
		res.Usage = *usage
	}
	return res
}

// Cancel 请求终止指定 run:run 不存在返回 ErrRunNotFound;已到终态幂等
// 成功;运行中则发出信号,由 Execute goroutine 负责投递 run.cancelled 终态
// 事件并返回 RunCancelled 结果(保证事件顺序只由单个 goroutine 写入)。
func (s *fakeSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return agentruntime.ErrSessionClosed
	}
	run, ok := s.runs[id]
	if !ok {
		return agentruntime.ErrRunNotFound
	}
	if run.status != agentruntime.RunRunning {
		return nil
	}
	close(run.done)
	return nil
}

// Events 返回从订阅时刻开始的会话事件流(尽力而为的旁路观察;权威结果
// 以 Execute 返回值/Result 为准)。投递终态事件后流关闭;会话关闭则先
// 排干已缓冲事件再关闭。
func (s *fakeSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	if err := agentruntime.CheckEvents(s.descriptor()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, agentruntime.ErrSessionClosed
	}
	// 从订阅时刻开始交付(不重放历史):先订阅后 Execute 仍能收到
	// run.started 与 RunID;已投递过终态的旧 run 不阻塞后续观察。
	return &eventStream{session: s, cursor: len(s.events)}, nil
}

// Result 返回指定 run 最近一次已知的 AgentRunResult(claim 快照)。
func (s *fakeSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	run, ok := s.runs[id]
	if !ok {
		return agentruntime.AgentRunResult{}, agentruntime.ErrRunNotFound
	}
	if !run.hasResult {
		return agentruntime.AgentRunResult{
			RunID:     id,
			SessionID: s.id,
			Status:    agentruntime.RunRunning,
		}, nil
	}
	return run.result, nil
}

// Close 结束会话。幂等;关闭后一切操作返回 ErrSessionClosed,阻塞中的
// Execute 以 RunCancelled 收尾,事件流等待者排干缓冲后被释放。
func (s *fakeSession) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.closeCh)
	s.broadcastLocked()
	s.mu.Unlock()
	return nil
}

// descriptor 返回会话持有的运行时自述副本(供 CheckEvents 使用)。
func (s *fakeSession) descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{
		Name:         DescriptorName,
		Version:      DescriptorVersion,
		Capabilities: s.caps,
	}
}

// eventStream 是 agentruntime.EventStream 的 FakeRuntime 实现:从订阅时刻
// 起按 Seq 升序交付会话事件,投递终态事件后、被放弃后或会话关闭并排干
// 缓冲后返回 ErrEventStreamClosed。每个流有独立游标;单消费者语义。
type eventStream struct {
	session *fakeSession
	// cursor 是下一个待交付事件在会话事件日志中的下标(持有会话锁时读写)。
	cursor int
	// abandoned 表示订阅方已调用 Close 放弃订阅。
	abandoned bool
	// lastTerminal 表示上一个交付的事件是终态事件。
	lastTerminal bool
}

// Next 按序交付事件。等待新事件时不持有会话锁;会话每次追加事件都会广播。
func (st *eventStream) Next(ctx context.Context) (agentruntime.Event, error) {
	for {
		st.session.mu.Lock()
		if st.abandoned || st.lastTerminal {
			st.session.mu.Unlock()
			return agentruntime.Event{}, agentruntime.ErrEventStreamClosed
		}
		if st.cursor < len(st.session.events) {
			ev := st.session.events[st.cursor]
			st.cursor++
			st.lastTerminal = ev.Kind.Terminal()
			st.session.mu.Unlock()
			return ev, nil
		}
		if st.session.closed {
			st.session.mu.Unlock()
			return agentruntime.Event{}, agentruntime.ErrEventStreamClosed
		}
		notify := st.session.notify
		st.session.mu.Unlock()

		select {
		case <-notify:
			// 有新事件(或会话关闭),回到循环在锁下重新检查。
		case <-ctx.Done():
			return agentruntime.Event{}, ctx.Err()
		}
	}
}

// Close 放弃订阅,幂等。之后 Next 恒返回 ErrEventStreamClosed。
func (st *eventStream) Close() error {
	st.session.mu.Lock()
	if st.abandoned {
		st.session.mu.Unlock()
		return nil
	}
	st.abandoned = true
	// 广播一次,唤醒可能正阻塞在别的 goroutine 的 Next。
	st.session.broadcastLocked()
	st.session.mu.Unlock()
	return nil
}

// broadcastLocked 唤醒所有事件流等待者:先换入新的广播通道再关闭旧通道。
// 这是本会话内唯一的通道关闭点(所有状态变化都在持有 s.mu 时经过它),
// 因此每代通道恰好被关闭一次 —— 包括会话关闭后仍有 in-flight run 投递
// 终态事件的情形。
func (s *fakeSession) broadcastLocked() {
	old := s.notify
	s.notify = make(chan struct{})
	close(old)
}

// emitLocked 追加一个规范化事件并广播。调用方必须持有 s.mu。
func (s *fakeSession) emitLocked(kind agentruntime.EventKind, id agentruntime.RunID, text string) {
	s.seq++
	s.events = append(s.events, agentruntime.Event{
		Kind:      kind,
		SessionID: s.id,
		RunID:     id,
		Seq:       s.seq,
		Time:      s.now(),
		Text:      text,
		// Native 恒为 nil:FakeRuntime 没有任何原生事件形态可保留(AC5)。
	})
	s.broadcastLocked()
}

// messageText 计算 agent.message 事件文本:脚本覆盖或原样回显 instruction。
func messageText(spec TurnSpec, instruction string) string {
	if spec.Message != "" {
		return spec.Message
	}
	return instruction
}

// usageText 是 usage.reported 事件的确定性规范化文本载荷。
func usageText(u agentruntime.Usage) string {
	return fmt.Sprintf("input_tokens=%d output_tokens=%d", u.InputTokens, u.OutputTokens)
}

// terminalKind 把 runtime 自述终态映射到规范化终态事件。
func terminalKind(status agentruntime.RunStatus) agentruntime.EventKind {
	switch status {
	case agentruntime.RunFailed:
		return agentruntime.EventRunFailed
	case agentruntime.RunCancelled:
		return agentruntime.EventRunCancelled
	default:
		return agentruntime.EventRunCompleted
	}
}
