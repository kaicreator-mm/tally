package agentruntime

import (
	"fmt"
	"sort"
	"strings"
)

// Capability 是协议可选能力的稳定标识。字符串值本身是契约(PRD §10.3
// 原文),改字符串 = 破坏协商兼容;常量名仅供 Go 侧引用。
type Capability string

// PRD §10.3 列出的全部可选能力。核心接口(§10.2)不含其中任何一项;
// 它们只能经协商后作为附加行为出现。
const (
	// CapSessionResume 续用既有会话(StartRequest.ResumeOf 需要)。
	CapSessionResume Capability = "session.resume"
	// CapSessionFork 从既有会话派生新会话。
	CapSessionFork Capability = "session.fork"
	// CapSessionIsolation 提供独立 session/thread/memory 边界。这是
	// VerificationPlanner Role 资格的必要条件(非充分):还要通过
	// conformance 的 Session Isolation Canary(PRD §10.4)。
	CapSessionIsolation Capability = "session.isolation"
	// CapTurnSteer 在 turn 进行中追加引导输入。
	CapTurnSteer Capability = "turn.steer"
	// CapTurnInterrupt 在 turn 进行中打断(超出核心 Cancel 语义的中断)。
	CapTurnInterrupt Capability = "turn.interrupt"
	// CapEventStreaming 提供会话事件流(Session.Events 需要)。
	CapEventStreaming Capability = "event.streaming"
	// CapOutputStructured 支持结构化输出约定。
	CapOutputStructured Capability = "output.structured"
	// CapSandboxNative 由 runtime 提供原生沙箱。
	CapSandboxNative Capability = "sandbox.native"
	// CapToolApproval 工具执行前经审批事件征得同意。
	CapToolApproval Capability = "tool.approval"
	// CapToolsCustom 支持注入自定义工具。
	CapToolsCustom Capability = "tools.custom"
	// CapModelSelect 支持按请求选择模型档位(StartRequest.ModelProfile 需要)。
	CapModelSelect Capability = "model.select"
	// CapSessionPersist 会话可持久化并在之后恢复查询。
	CapSessionPersist Capability = "session.persist"
	// CapMemoryDeclared runtime 声明了跨会话记忆(INV-VER-05 的
	// Runtime-Protocol-observable 通道之一;holdout 审计必须覆盖它)。
	CapMemoryDeclared Capability = "memory.declared"
)

// CapabilitySet 是已声明能力的集合。构造后当只读值用。零值/nil 合法,
// 表示什么能力都没声明 —— 能力只能由 runtime 声明,不能被调用方想象。
type CapabilitySet map[Capability]struct{}

// NewCapabilitySet 由能力列表构造集合,重复项自动去重。
func NewCapabilitySet(caps ...Capability) CapabilitySet {
	s := make(CapabilitySet, len(caps))
	for _, c := range caps {
		s[c] = struct{}{}
	}
	return s
}

// Has 报告是否声明了某项能力。
func (s CapabilitySet) Has(c Capability) bool {
	_, ok := s[c]
	return ok
}

// List 返回声明的能力(无序)。
func (s CapabilitySet) List() []Capability {
	out := make([]Capability, 0, len(s))
	for c := range s {
		out = append(out, c)
	}
	return out
}

// Sorted 返回字典序能力列表,供展示/日志/测试断言保持确定性。
func (s CapabilitySet) Sorted() []Capability {
	return SortedCapabilities(s.List())
}

// SortedCapabilities 去重并按字典序排序;nil/空输入返回空切片。
func SortedCapabilities(caps []Capability) []Capability {
	seen := make(CapabilitySet, len(caps))
	out := make([]Capability, 0, len(caps))
	for _, c := range caps {
		if !seen.Has(c) {
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// CapabilityRequirement 是 Tally 侧对 runtime 的能力需求:Required 全部
// 满足才通过协商;Preferred 只做偏好记录,不否决(PRD §10.3)。
type CapabilityRequirement struct {
	Required  []Capability
	Preferred []Capability
}

// NegotiatedCapabilities 是协商结果 —— 纯数据。调用方的控制流只读这份
// 数据,不读 runtime 名字(PRD §10.3;AC2 的落点)。
type NegotiatedCapabilities struct {
	// Granted 是必需能力去重排序后的结果(Negotiate 成功即全部已声明)。
	Granted []Capability
	// PreferredGranted 是可选能力中已声明的部分(字典序)。
	PreferredGranted []Capability
	// PreferredAbsent 是可选能力中未声明的部分(字典序)。调用方据此走
	// 降级路径:如无 event.streaming 就不订阅事件流,改用 Result 兜底。
	PreferredAbsent []Capability
}

// Negotiate 把需求与声明做一次纯函数协商。必需能力有缺失时返回
// *CapabilityError(用 errors.As 取结构化数据),不返回部分结果。
func Negotiate(declared CapabilitySet, req CapabilityRequirement) (NegotiatedCapabilities, error) {
	var missing []Capability
	for _, c := range SortedCapabilities(req.Required) {
		if !declared.Has(c) {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return NegotiatedCapabilities{}, &CapabilityError{Missing: missing}
	}
	out := NegotiatedCapabilities{
		Granted:          SortedCapabilities(req.Required),
		PreferredGranted: []Capability{},
		PreferredAbsent:  []Capability{},
	}
	for _, c := range SortedCapabilities(req.Preferred) {
		if declared.Has(c) {
			out.PreferredGranted = append(out.PreferredGranted, c)
		} else {
			out.PreferredAbsent = append(out.PreferredAbsent, c)
		}
	}
	return out, nil
}

// CapabilityError 表示必需能力未被声明。Missing 按字典序;调用方用
// errors.As 取数据做决策,不要解析错误文本。
type CapabilityError struct {
	Missing []Capability
}

// Error 实现 error。
func (e *CapabilityError) Error() string {
	names := make([]string, len(e.Missing))
	for i, c := range e.Missing {
		names[i] = string(c)
	}
	return fmt.Sprintf("agentruntime: runtime 未声明必需能力: %s", strings.Join(names, ", "))
}
