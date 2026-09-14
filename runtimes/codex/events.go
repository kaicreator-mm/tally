package codex

// events.go 拥有事件面:codex exec --json 的 JSONL 原生形态(只建模 P1-36
// 探针记录 §3 观测过的形态)、规范化事件投递与会话事件流。

import (
	"context"

	"tally/internal/agentruntime"
)

// rawEvent 是 codex exec --json JSONL 的原生事件形态。字段只来自 P1-36
// 探针记录 §3 的实测输出:
//
//	{"type":"thread.started","thread_id":"<uuid>"}
//	{"type":"turn.started"}
//	{"type":"error","message":"..."}
//	{"type":"item.completed","item":{"id":"item_0","type":"error","message":"..."}}
//
// 未知 type / 未知 item.type 容忍并忽略(不解释即不发明,DECISION_LOCKS #13)。
type rawEvent struct {
	Type     string   `json:"type"`
	ThreadID string   `json:"thread_id"` // thread.started
	Message  string   `json:"message"`   // error
	Item     *rawItem `json:"item"`      // item.completed
}

// rawItem 是 item.completed 事件里的 item 对象(实测形态仅 error 项)。
type rawItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message string `json:"message"` // item.type == "error"
}

// emitRunStartedOnce 投递该 run 的 run.started(每个 run 恰好一次)。本适配器
// 的协议 SessionID = codex thread id,而 thread id 在 thread.started 到达后才
// 已知:为保住「事件归属与 claim 一致」,run.started 延迟到 thread.started
// (或 run 收尾时兜底)再投递,等待它的订阅者只是短暂阻塞。
func (s *Session) emitRunStartedOnce(id agentruntime.RunID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r0 := s.runs[id]
	if r0 == nil || r0.startedEmitted {
		return
	}
	r0.startedEmitted = true
	s.emitLocked(agentruntime.EventRunStarted, id, "", nil)
}

// emitEvent 追加一条规范化事件并广播。只允许 execTurn 所在 goroutine 调用,
// 保证事件顺序与进程输出顺序一致。
func (s *Session) emitEvent(kind agentruntime.EventKind, id agentruntime.RunID, text string, native *agentruntime.NativeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitLocked(kind, id, text, native)
}

// emitLocked 追加事件并广播;调用方必须持有 s.mu。
func (s *Session) emitLocked(kind agentruntime.EventKind, id agentruntime.RunID, text string, native *agentruntime.NativeEvent) {
	s.seq++
	s.events = append(s.events, agentruntime.Event{
		Kind:      kind,
		SessionID: agentruntime.SessionID(s.threadID),
		RunID:     id,
		Seq:       s.seq,
		Time:      s.runtime.now(),
		Text:      text,
		Native:    native,
	})
	s.broadcastLocked()
}

// broadcastLocked 唤醒所有事件流等待者:换入新广播通道后关闭旧通道。这是
// 本会话内唯一的通道关闭点(所有状态变化都在持有 s.mu 时经过它),因此每代
// 通道恰好被关闭一次。
func (s *Session) broadcastLocked() {
	old := s.notify
	s.notify = make(chan struct{})
	close(old)
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

// Events 返回从订阅时刻开始的会话事件流(尽力而为的旁路观察;权威结果以
// Execute 返回值/Result 为准)。未声明 event.streaming 时被 CheckEvents 拒绝
// (本适配器已声明);会话关闭后返回 ErrSessionClosed。
func (s *Session) Events(ctx context.Context) (agentruntime.EventStream, error) {
	if err := agentruntime.CheckEvents(s.runtime.Descriptor()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, agentruntime.ErrSessionClosed
	}
	// 从订阅时刻开始交付(不重放历史):先订阅后 Execute 仍能收到
	// run.started 与 RunID。
	return &sessionStream{session: s, cursor: len(s.events)}, nil
}

// sessionStream 是 agentruntime.EventStream 的 CodexRuntime 实现:从订阅时刻
// 起按 Seq 升序交付会话事件,投递终态事件后、被放弃后或会话关闭并排干缓冲后
// 返回 ErrEventStreamClosed。每个流有独立游标;单消费者语义。
type sessionStream struct {
	session *Session
	// cursor 是下一个待交付事件在会话事件日志中的下标(持有会话锁时读写)。
	cursor int
	// abandoned 表示订阅方已调用 Close 放弃订阅。
	abandoned bool
	// lastTerminal 表示上一个交付的事件是终态事件。
	lastTerminal bool
}

// Next 按序交付事件。等待新事件时不持有会话锁;会话每次追加事件都会广播。
func (st *sessionStream) Next(ctx context.Context) (agentruntime.Event, error) {
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
func (st *sessionStream) Close() error {
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
