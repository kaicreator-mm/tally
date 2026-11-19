package pi

// identity.go 拥有 IdentityReporter 缝的 PiRuntime 侧实现:把本 runtime 现场
// 「实际」观察到的身份事实映射进 runtime 中立的 agentruntime.ObservedIdentity
//(P0-01;Decision Lock 15:实际身份 != 期望配置,两者必须分开携带、分开
// 落库)。
//
// 观察语义边界(改动前必读):
//
//   - 只报告已建立的观察,绝不用期望配置回填:
//       * RuntimeVersion 来自一次真实的 `pi --version` 进程观测(由
//         Start/Snapshot 建立、Runtime 级缓存、跨会话共享;观测形态是裸版本
//         行 "0.85.1")。适配器版本 DescriptorVersion 与它是两个事实,绝不
//         写入观察字段(AGENTS.md §13)。
//       * RuntimeName 恒为空:`pi --version` 只输出裸版本号,不自报产品名
//        (实测输出恰为 "0.85.1" 单行),无可观察的名字,不发明。
//       * Model/Provider 是 json 模式下最近一次可解析回合中 assistant
//         message_end 自报的实际生效值(实测 "model":"deepseek-v4-flash"、
//         "provider":"deepseek")。Config.Model/Config.Provider 是期望旗标,
//         把它们当「实际生效」上报即伪造观察 —— 只有流里真实出现过才入本
//         字段;text 模式与失败/取消回合无观察,保持零值。
//       * SessionID 是 pi 在 json 流 session 行自报的会话 UUID(实测即使
//         --no-session 也输出该行;一次性句柄,本适配器不声明 resume/
//         persist,不构成可续用承诺)。适配器自造的会话编号
//        (AgentRunResult.SessionID)是 Tally 协议层标识,不是 pi 的观察
//         事实,不写入本字段。
//   - Identity() 只读、非阻塞、可并发调用;不在调用时发起新观察(不发进程、
//     不做网络请求),只返回已建立的观察(agentruntime.IdentityReporter 契约)。
//   - 探针失败(版本闸拒绝会话)时本接口不会被触达 —— Start 已 fail-closed;
//     全零值(未观察)仍是合法状态,如实返回,不报错。

import (
	"strings"

	"tally/internal/agentruntime"
)

// cliIdentity 是一次 `pi --version` 现场观察的解码结果:CLI 对自身版本的
// 自我报告。观测形态是裸版本行(无产品名自报)—— Name 刻意没有取值路径。
// 零值 = 尚未观察(缓存未建立或探针失败)。
type cliIdentity struct {
	Version string // CLI 自报版本(如 "0.85.1")
}

// observed 返回该观察是否携带任何事实。
func (c cliIdentity) observed() bool { return c.Version != "" }

// parseCLIIdentity 从 `pi --version` 输出解码 CLI 自报版本(观测形态:裸
// 版本行,取首个非空行原文,不发明格式)。空输出返回零值。
func parseCLIIdentity(out string) cliIdentity {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return cliIdentity{Version: line}
		}
	}
	return cliIdentity{}
}

// Identity 实现 agentruntime.IdentityReporter(P0-01 可选缝)。返回该会话
// 当前已观察到的实际身份快照;语义见文件头。
func (s *Session) Identity() agentruntime.ObservedIdentity {
	r := s.runtime
	r.mu.Lock()
	cli := r.cli
	r.mu.Unlock()
	s.mu.Lock()
	handle, model := s.observedSessionHandle, s.observedModel
	s.mu.Unlock()
	return agentruntime.ObservedIdentity{
		RuntimeName:    "", // `pi --version` 只报裸版本号:产品名未观察,不发明
		RuntimeVersion: cli.Version,
		Model:          model, // json 流 assistant message_end 自报的实际生效值;未观察为空,绝不回填期望旗标
		SessionID:      agentruntime.SessionID(handle),
	}
}
