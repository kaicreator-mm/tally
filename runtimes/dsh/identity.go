package dsh

// identity.go 拥有 IdentityReporter 缝的 DSHRuntime 侧实现:把本 runtime
// 现场「实际」观察到的身份事实映射进 runtime 中立的 agentruntime.ObservedIdentity
//(P0-01;Decision Lock 15:实际身份 != 期望配置,两者必须分开携带、分开落库)。
//
// 观察语义边界(改动前必读):
//
//   - 只报告已建立的观察,绝不用期望配置回填:
//       * RuntimeVersion 来自一次真实的 `dsh --version` 进程观测(由
//         Start/Snapshot 建立、Runtime 级缓存、跨会话共享;观测形态是裸版本
//         行 "0.1.5-rc.1",spike §3 探针 1)。适配器版本 DescriptorVersion
//         与它是两个事实,绝不写入观察字段(AGENTS.md §13)。
//       * RuntimeName 恒为空:`dsh --version` 不自报产品名(与 codex 的
//         "codex-cli 0.153.0" 双字段形态不同),无可观察的名字,不发明。
//       * Model 恒为空:headless 表面不报告实际生效模型(spike §2.5,R-3);
//         Config.Model/Provider 是期望配置,把它当「实际生效」上报即伪造
//         观察(requested ≠ observed 永不相等,spike 条件 5)。
//       * SessionID 恒为空:headless 不给出任何协议层会话句柄;适配器自造的
//         会话编号(AgentRunResult.SessionID)是 Tally 协议层标识,不是 dsh
//         的观察事实,不写入本字段。
//   - Identity() 只读、非阻塞、可并发调用;不在调用时发起新观察(不发进程、
//     不做网络请求),只返回已建立的观察(agentruntime.IdentityReporter 契约)。
//   - 探针失败(版本闸拒绝会话)时本接口不会被触达 —— Start 已 fail-closed;
//     全零值(未观察)仍是合法状态,如实返回,不报错。

import (
	"strings"

	"tally/internal/agentruntime"
)

// cliIdentity 是一次 `dsh --version` 现场观察的解码结果:launcher 对自身
// 版本的自我报告。观测形态(spike §3 探针 1)是裸版本行,无产品名自报 ——
// Name 刻意没有取值路径。零值 = 尚未观察(缓存未建立或探针失败)。
type cliIdentity struct {
	Version string // launcher 自报版本(如 "0.1.5-rc.1")
}

// observed 返回该观察是否携带任何事实。
func (c cliIdentity) observed() bool { return c.Version != "" }

// parseCLIIdentity 从 `dsh --version` 输出解码 CLI 自报版本(观测形态:裸
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
	return agentruntime.ObservedIdentity{
		RuntimeName:    "", // `dsh --version` 不自报产品名:未观察,不发明
		RuntimeVersion: cli.Version,
		Model:          "", // headless 不自报实际模型:未观察,绝不回填期望配置
		SessionID:      "", // headless 无协议层会话句柄:未观察(适配器编号不是观察事实)
	}
}
