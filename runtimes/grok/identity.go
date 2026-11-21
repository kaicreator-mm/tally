package grok

// identity.go 拥有 IdentityReporter 缝的 GrokRuntime 侧实现:把本 runtime
// 现场「实际」观察到的身份事实映射进 runtime 中立的 agentruntime.ObservedIdentity
// (P0-01;Decision Lock 15:实际身份 != 期望配置,两者必须分开携带、分开落库)。
//
// 观察语义边界(改动前必读):
//
//   - 只报告已建立的观察,绝不用期望配置回填:
//       * RuntimeVersion 来自一次真实的 `grok --version` 进程观测(由
//         Start/Snapshot 建立、Runtime 级缓存、跨会话共享;观测形态是
//         "grok 1.0.87 (929e6dd187b1)",grok 1.0.87 现场观测)。适配器版本
//         DescriptorVersion 与它是两个事实,绝不写入观察字段(AGENTS.md §13)。
//       * RuntimeName 来自同一观测的首个字段(观测形态 "grok ..." 自报产品名,
//         与 dsh 的裸版本行不同、与 codex 的 "codex-cli 0.153.0" 双字段形态
//         同类)。名字只在 --version 输出可解析出该字段时携带。
//       * Model 恒为空:--single plain 表面不报告实际生效模型;Config.Model
//         是期望配置,把它当「实际生效」上报即伪造观察(requested ≠ observed
//         永不相等,AGENTS.md §13)。
//       * SessionID 恒为空:--single 一次性表面不给出任何协议层会话句柄;
//         适配器自造的会话编号(AgentRunResult.SessionID)是 Tally 协议层
//         标识,不是 grok 的观察事实,不写入本字段。
//   - Identity() 只读、非阻塞、可并发调用;不在调用时发起新观察(不发进程、
//     不做网络请求),只返回已建立的观察(agentruntime.IdentityReporter 契约)。
//   - 探针失败(版本闸拒绝会话)时本接口不会被触达 —— Start 已 fail-closed;
//     全零值(未观察)仍是合法状态,如实返回,不报错。

import (
	"strings"

	"tally/internal/agentruntime"
)

// cliIdentity 是一次 `grok --version` 现场观察的解码结果:CLI 对自身版本的
// 自我报告。观测形态(grok 1.0.87)是 "grok 1.0.87 (929e6dd187b1)":
// 产品名、版本、括号内 commit。解码只取按空白切分后可辨认的字段,不发明
// 更多格式。零值 = 尚未观察(缓存未建立或探针失败)。
type cliIdentity struct {
	// Name 是自报产品名(观测值 "grok");输出形态不含名字时为空。
	Name string
	// Version 是自报版本(观测值 "1.0.87")。
	Version string
	// Commit 是自报 commit(观测值 "929e6dd187b1",括号剥除);形态不含时为空。
	Commit string
}

// observed 返回该观察是否携带任何事实(版本是最低可辨认事实)。
func (c cliIdentity) observed() bool { return c.Version != "" }

// parseCLIIdentity 从 `grok --version` 输出解码 CLI 自报身份(观测形态:
// "grok 1.0.87 (929e6dd187b1)";取首个非空行,按空白切分:name=字段1,
// version=字段2,commit=字段3 去括号)。字段缺失时只填实际可辨认的部分,
// 不猜测。空输出/无可辨认版本返回零值。
func parseCLIIdentity(out string) cliIdentity {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// 观测形态的版本字段以数字开头("1.0.87"):以此拒绝把纯文本行
		// (如 TUI 标语 "Grok Build TUI")误读为身份。
		if len(fields) < 2 || !versionLike(fields[1]) {
			return cliIdentity{}
		}
		c := cliIdentity{Name: fields[0], Version: fields[1]}
		if len(fields) > 2 {
			c.Commit = strings.Trim(fields[2], "()")
		}
		return c
	}
	return cliIdentity{}
}

// versionLike 报告 s 是否形如观测到的版本字段:非空且以数字开头。
func versionLike(s string) bool {
	if s == "" {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

// Identity 实现 agentruntime.IdentityReporter(P0-01 可选缝)。返回该会话
// 当前已观察到的实际身份快照;语义见文件头。
func (s *Session) Identity() agentruntime.ObservedIdentity {
	r := s.runtime
	r.mu.Lock()
	cli := r.cli
	r.mu.Unlock()
	return agentruntime.ObservedIdentity{
		RuntimeName:    cli.Name,    // `grok --version` 自报产品名(观测形态首字段)
		RuntimeVersion: cli.Version, // 现场观察,非硬编码
		Model:          "",          // plain 表面不自报实际模型:未观察,绝不回填期望配置
		SessionID:      "",          // --single 无协议层会话句柄:未观察(适配器编号不是观察事实)
	}
}
