package agentruntime

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// EventKind 是规范化事件的类型。规范化面向 Tally 关心的最小词汇表;
// runtime 私有的其它事件形态经 Event.Native 原样保留,不强行规范化。
type EventKind string

const (
	// 生命周期事件:run.started 开场,其余三个是终态(投递后事件流关闭)。
	EventRunStarted   EventKind = "run.started"
	EventRunCompleted EventKind = "run.completed"
	EventRunFailed    EventKind = "run.failed"
	EventRunCancelled EventKind = "run.cancelled"

	// 语义流事件:agent 的可见输出、推理痕迹、工具活动与用量自报。
	EventAgentMessage   EventKind = "agent.message"
	EventAgentReasoning EventKind = "agent.reasoning"
	EventToolCall       EventKind = "tool.call"
	EventToolOutput     EventKind = "tool.output"
	EventUsageReported  EventKind = "usage.reported"

	// 审批事件:仅在声明 tool.approval 能力的 runtime 上出现。
	EventApprovalRequested EventKind = "approval.requested"
	EventApprovalResolved  EventKind = "approval.resolved"
)

// Terminal 报告该事件是否为 run 终态事件;事件流在投递终态事件后关闭。
func (k EventKind) Terminal() bool {
	switch k {
	case EventRunCompleted, EventRunFailed, EventRunCancelled:
		return true
	default:
		return false
	}
}

// NativeEvent 是规范化事件可选拴带的 runtime 原生形态(AC4):规范化
// 必然丢信息,原生载荷按字节保留,供审计/排障/未来规范化升级。
type NativeEvent struct {
	// Type 是 runtime 原生事件类型名(如适配协议里的事件名),原样保留。
	Type string `json:"type"`
	// Payload 是原生事件的原始编码(json.RawMessage:已是合法 JSON 时
	// 零拷贝;非 JSON 来源由适配器封装成 JSON)。字节保真,本包不解释。
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Event 是规范化后的协议事件。全部字段来自 runtime,按 INV-AUTH-01 都是
// 观察到的 claim 侧信息,不是 Tally 事实;Tally 对它只做记录与投影。
type Event struct {
	// Kind 是规范化事件类型。
	Kind EventKind `json:"kind"`
	// SessionID/RunID 归属。
	SessionID SessionID `json:"session_id"`
	RunID     RunID     `json:"run_id"`
	// Seq 在会话内从 1 开始单调递增,供排序/去重/断点续传。
	Seq int64 `json:"seq"`
	// Time 是适配器观察到事件的时刻。
	Time time.Time `json:"time"`
	// Text 是该事件的规范化文本载荷(消息正文/工具输出摘要),可空。
	Text string `json:"text,omitempty"`
	// Native 是可选的原生事件保留(AC4);nil 表示没有对应原生形态。
	Native *NativeEvent `json:"native,omitempty"`
}

// HasNative 报告该事件是否携带原生形态。
func (e Event) HasNative() bool { return e.Native != nil }

// EventStream 按序交付会话事件。实现必须保证:
//
//   - Next 按 Seq 升序交付,不重不漏(会话未关闭的部分);
//   - 投递完终态事件之后再 Next 返回 ErrEventStreamClosed;
//   - Close 之后再 Next 返回 ErrEventStreamClosed;
//   - ctx 先于事件就绪时返回 ctx.Err()。
type EventStream interface {
	Next(ctx context.Context) (Event, error)
	// Close 主动放弃订阅,之后 Next 返回 ErrEventStreamClosed。幂等。
	Close() error
}

var (
	// ErrEventStreamClosed 表示事件流已终结(终态事件已投递、订阅被放弃
	// 或会话已关闭)。
	ErrEventStreamClosed = errors.New("agentruntime: 事件流已关闭")
	// ErrStreamingNotDeclared 表示 runtime 未声明 event.streaming 能力,
	// 不提供事件流。调用方应按 NegotiatedCapabilities.PreferredAbsent
	// 提前走降级路径,而不是到这里才碰壁。
	ErrStreamingNotDeclared = errors.New("agentruntime: runtime 未声明 event.streaming,不提供事件流")
)
