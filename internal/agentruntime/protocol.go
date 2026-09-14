// Package agentruntime 定义 Tally 自有的、运行时中立的编码 Agent 协议
// (PRD §10 Agent Runtime Protocol),供 FakeRuntime(P1-24)与
// CodexRuntime(P1-37)实现,供运行时一致性 harness(P1-25)校验。
//
// 协议核心边界(改动前必读):
//
//   - 最小核心只有六件事:Descriptor / Start / Execute / Cancel / Events /
//     Result(PRD §10.2)。任何具体 runtime(如 Codex)的高级能力一律不进
//     核心接口,只能作为 capability.go 中的可选 capability 协商出现。
//   - 控制流只按能力协商结果(NegotiatedCapabilities)分支,不按 runtime
//     名字分支(PRD §10.3)。Descriptor.Name 只用于标识/遥测/归因。
//   - Runtime 产出的一切(AgentRunResult、Event)都是 Agent Claim,不是
//     Project Fact(INV-AUTH-01):它不能替代 Tally 验证器/证据通道产出的
//     事实,也不能被实现方重定义成验证结论。
//   - 本包只定义协议形状,不执行任何进程、不依赖任何具体 runtime。
package agentruntime

import (
	"context"
	"errors"
)

// SessionID 标识一次 Runtime 会话(session/thread 边界的协议句柄)。
// Role separation(PRD §10.3)要求 Verification 与 Implementation 落在
// 不同 Session 上,由上层用不同 StartRequest 保证;本包只携带标识。
type SessionID string

// RunID 标识会话内的一次运行(一个 turn)。会话内唯一,由适配器生成,
// 经 Execute 返回值与事件流携带,供 Cancel/Result 定位。
type RunID string

// RunStatus 是 Runtime 自述的运行终态。它是 claim 的组成部分:runtime
// 说「完成」不等于验证通过,更不等于事实(INV-AUTH-01)。
type RunStatus string

const (
	// RunRunning 表示运行尚未到终态(Result 此时可返回当前快照)。
	RunRunning RunStatus = "running"
	// RunCompleted 表示 runtime 自述按指令完成。
	RunCompleted RunStatus = "completed"
	// RunFailed 表示 runtime 自述运行失败(含预算耗尽、runtime 内部错误)。
	RunFailed RunStatus = "failed"
	// RunCancelled 表示运行被 Cancel 或被取消的 ctx 终止。
	RunCancelled RunStatus = "cancelled"
)

// 协议级哨兵错误(结构化错误另见 capability.CapabilityError)。
var (
	// ErrNoRuntimeName 表示 RuntimeDescriptor 缺少 Name;无名 runtime 无法归因。
	ErrNoRuntimeName = errors.New("agentruntime: RuntimeDescriptor 缺少 name")
	// ErrNoWorkspace 表示 StartRequest 缺少 workspace。workspace 必须是
	// 可丢弃 Git worktree —— agent 产生的可执行变更绝不在 owner 主
	// worktree 运行(INV-EXEC-01)。
	ErrNoWorkspace = errors.New("agentruntime: StartRequest 缺少 workspace(必须是可丢弃 Git worktree,INV-EXEC-01)")
	// ErrNoInstruction 表示 ExecuteRequest 缺少 instruction;空 turn 无意义。
	ErrNoInstruction = errors.New("agentruntime: ExecuteRequest 缺少 instruction")
	// ErrRunNotFound 表示 Cancel/Result 指定的 run 不存在(或已被遗忘)。
	ErrRunNotFound = errors.New("agentruntime: 指定的 run 不存在")
	// ErrSessionClosed 表示会话已 Close,一切操作都被拒绝。
	ErrSessionClosed = errors.New("agentruntime: 会话已关闭")
)

// RuntimeDescriptor 是 runtime 的自述。Capabilities 是声明而不是承诺:
// 声明的能力是否真实可用由 conformance(PRD §10.4,P1-25 harness)另行
// 验证 —— 声明本身仍是 claim,conformance 结果才是证据。
type RuntimeDescriptor struct {
	// Name 是 runtime 标识(如 "fake"、"codex"),只用于展示/遥测/审计归因。
	// 任何控制流都不得按 Name 分支 —— 按 CapabilitySet 的协商结果分支(PRD §10.3)。
	Name string
	// Version 是适配器/runtime 版本字符串,供归因与排障,可为空。
	Version string
	// Capabilities 是该 runtime 声明的能力集合(见 capability.go)。
	Capabilities CapabilitySet
}

// Validate 校验描述符的最小结构:Name 必须非空。
func (d RuntimeDescriptor) Validate() error {
	if d.Name == "" {
		return ErrNoRuntimeName
	}
	return nil
}

// StartRequest 开启会话的请求。
type StartRequest struct {
	// Workspace 是本次会话的源码执行目录,必须是可丢弃的 Git worktree
	// (INV-EXEC-01:agent 生成的可执行变更不在 owner 主 worktree 运行)。
	Workspace string
	// ResumeOf 非空表示续用既有会话而不是新开;需要 session.resume 能力。
	ResumeOf SessionID
	// ModelProfile 非空表示请求指定模型档位;需要 model.select 能力。
	ModelProfile string
}

// RequiredCapabilities 由请求内容推导出的必需能力(纯数据,供 Negotiate
// 使用):基础 Start 无额外要求;ResumeOf ⇒ session.resume;
// ModelProfile ⇒ model.select。返回值去重并按字典序排序。
func (r StartRequest) RequiredCapabilities() []Capability {
	var caps []Capability
	if r.ResumeOf != "" {
		caps = append(caps, CapSessionResume)
	}
	if r.ModelProfile != "" {
		caps = append(caps, CapModelSelect)
	}
	return SortedCapabilities(caps)
}

// Validate 校验请求的最小结构。
func (r StartRequest) Validate() error {
	if r.Workspace == "" {
		return ErrNoWorkspace
	}
	return nil
}

// ExecuteRequest 是一个 turn 的输入。
type ExecuteRequest struct {
	// Instruction 是本轮给 coding agent 的自然语言指令,由 Tally 上层组装。
	// 对 agent 而言它只是输入;其产出按 INV-AUTH-02 仍只是提案/claim。
	Instruction string
}

// Validate 校验请求的最小结构。
func (r ExecuteRequest) Validate() error {
	if r.Instruction == "" {
		return ErrNoInstruction
	}
	return nil
}

// Usage 是 runtime 自报的一次运行的 token 用量。数值由 runtime 提供,
// 属于 claim;预算执行由 Tally 侧 ExecutionPolicy 按自己的记账裁决
// (PRD §11.2),不直接采信本值。
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// AgentRunResult 是 Runtime 对一次运行结果的自述 —— 明确是 Agent Claim,
// 不是 Project Fact(INV-AUTH-01;也是本类型刻意不叫 TaskResult 的原因):
//
//   - Status/Summary/Usage 都是 runtime 的说法;「完成」不等于验证通过;
//   - 它不得被持久化为事实表记录、不得当验证证据、不得单独据此推进
//     生命周期状态;事实只能来自 Tally 验证器/证据通道(PRD §8/§14);
//   - 生命周期状态是派生的,只能由证据推导(INV-STATE-01)。
type AgentRunResult struct {
	// RunID/SessionID 定位这次运行。
	RunID     RunID     `json:"run_id"`
	SessionID SessionID `json:"session_id"`
	// Status 是 runtime 自述的终态。
	Status RunStatus `json:"status"`
	// Summary 是 runtime 对「这轮做了什么/为什么」的自述文本,原样保留,
	// 不做任何事实化解释。
	Summary string `json:"summary"`
	// Usage 是 runtime 自报的 token 用量。
	Usage Usage `json:"usage"`
}

// Runtime 是适配器要实现的最小核心(PRD §10.2)。六件事的落点:
// Descriptor/Start 在 Runtime 上;Execute/Cancel/Events/Result 在其开启的
// Session 上。
type Runtime interface {
	// Descriptor 返回自述(标识 + 声明能力)。声明由 conformance 另行验证。
	Descriptor() RuntimeDescriptor
	// Start 开启一个会话。实现应先过 CheckStart:校验请求结构,并按请求
	// 推导的必需能力与声明能力协商,缺任何一项即拒绝(AC2)。
	Start(ctx context.Context, req StartRequest) (Session, error)
}

// Session 是一次已开启的协议会话。
type Session interface {
	// Execute 同步执行一个 turn,返回 runtime 的 AgentRunResult(claim)。
	// 本轮被取消(ctx 取消或 Cancel)是正常终态:返回 RunCancelled 的结果
	// 而非 error;只有请求本身非法(Validate 不过)才返回 error。
	Execute(ctx context.Context, req ExecuteRequest) (AgentRunResult, error)
	// Cancel 请求终止指定 run。run 不存在返回 ErrRunNotFound;对已到终态
	// 的 run 幂等成功。
	Cancel(ctx context.Context, id RunID) error
	// Events 订阅本会话的规范化事件流(会话级:先订阅后 Execute 也能收到
	// run.started 与 RunID)。未声明 event.streaming 的 runtime 返回
	// ErrStreamingNotDeclared;会话关闭后返回 ErrSessionClosed。
	// 事件是尽力而为的旁路观察,不是权威结果;权威结果永远以 Execute 的
	// 返回值/Result 为准(它们同样只是 claim)。
	Events(ctx context.Context) (EventStream, error)
	// Result 返回指定 run 最近一次已知的 AgentRunResult(claim 快照)。
	// run 不存在返回 ErrRunNotFound。供 Cancel 后查询、或恢复会话后补查。
	Result(ctx context.Context, id RunID) (AgentRunResult, error)
	// Close 结束会话并释放适配器持有的资源(进程/连接等)。关闭后一切
	// 操作返回 ErrSessionClosed。Close 属生命周期收尾,不是可选 capability。
	Close(ctx context.Context) error
}

// CheckStart 是 Start 实现的统一前置闸:校验请求结构,并按请求推导出的
// 必需能力与声明能力做协商,缺任何一项返回 *CapabilityError。适配器不得
// 跳过它自行放行,否则「按能力协商」就退化成了随意放行(AC2)。
func CheckStart(d RuntimeDescriptor, req StartRequest) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if err := req.Validate(); err != nil {
		return err
	}
	_, err := Negotiate(d.Capabilities, CapabilityRequirement{Required: req.RequiredCapabilities()})
	return err
}

// CheckEvents 是 Events 实现的统一前置闸:未声明 event.streaming 的
// runtime 直接拒绝(能力是协商出来的数据,不是接口存在性)。
func CheckEvents(d RuntimeDescriptor) error {
	if !d.Capabilities.Has(CapEventStreaming) {
		return ErrStreamingNotDeclared
	}
	return nil
}
