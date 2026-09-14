package codex

// identity.go 拥有 P0-01 IdentityReporter 缝的 CodexRuntime 侧实现(P0-02):
// 把本 runtime 现场「实际」观察到的 CLI/会话/模型事实映射进 runtime 中立的
// agentruntime.ObservedIdentity,供 workflow 在开跑时捕获并随 CodegenRun 落库
//(PRD §11.3 actual runtime facts;Decision Lock 15)。
//
// 观察语义边界(改动前必读):
//
//   - 只报告已建立的观察,绝不用期望配置回填:
//       * RuntimeName / RuntimeVersion 来自一次真实的 `codex --version` 进程
//         观测(由 Start 建立、Runtime 级缓存、跨会话共享),是 CLI 二进制
//         的自我报告(实测形态 "codex-cli 0.153.0")。适配器版本
//         DescriptorVersion("1")与它是两个事实,绝不写入观察字段
//         —— 把适配器版本冒充 CLI 版本即伪造观察(P0-02 AC1)。
//       * SessionID 是协议层实际给出的 codex thread id(thread.started
//         事件观察;resume 会话则是已建立的续用 thread 句柄)。新会话首个
//         Execute 之前 thread id 尚未观察到,如实为空。
//       * Model 恒为空:探针记录 §3 的 exec JSONL 词汇表(thread.started /
//         turn.started / error / item.completed)没有模型自报字段,解析即
//         发明。未观察 ≠ 请求值 —— Start 请求的 -m 档位绝不能被当成「实际
//         生效模型」上报(P0-02 AC3;宁缺毋假,INV-AUTH-01 同源纪律)。
//   - Identity() 只读、非阻塞、可并发调用;不在调用时发起新观察(不发进程、
//     不做网络请求),只返回已建立的观察(agentruntime.IdentityReporter 契约)。
//   - 全零值(未观察)是合法状态:观察尚未建立或探针失败时如实返回,不报错。

import (
	"context"

	"tally/internal/agentruntime"
)

// cliSelfName 是 `codex --version` 实测输出中 CLI 自报的产品名
// (探针记录 §1:"codex-cli 0.153.0")。仅该形态下 Name 才被观察到。
const cliSelfName = "codex-cli"

// cliIdentity 是一次 `codex --version` 现场观察的解码结果:CLI 二进制对自身
// 标识与版本的自我报告。零值 = 尚未观察(缓存未建立或探针失败)。
type cliIdentity struct {
	Name    string // CLI 自报标识(如 "codex-cli");不可解析时为空
	Version string // CLI 自报版本(如 "0.153.0");不可解析时为整行原文
}

// observed 返回该观察是否携带任何事实。
func (c cliIdentity) observed() bool { return c.Name != "" || c.Version != "" }

// Identity 实现 agentruntime.IdentityReporter(P0-01 可选缝;P0-02 落地)。
// 返回该会话当前已观察到的实际身份快照;语义见文件头。
func (s *Session) Identity() agentruntime.ObservedIdentity {
	s.mu.Lock()
	thread := s.threadID
	s.mu.Unlock()
	cli := s.runtime.observedCLI()
	return agentruntime.ObservedIdentity{
		RuntimeName:    cli.Name,
		RuntimeVersion: cli.Version,
		Model:          "", // exec 表面不自报模型:未观察,不回填请求档位
		SessionID:      agentruntime.SessionID(thread),
	}
}

// observedCLI 读取 Runtime 已建立的 CLI 身份观察(只读缓存,不发探针)。
func (r *Runtime) observedCLI() cliIdentity {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cli
}

// observeCLIIdentity 尽力建立 CLI 身份观察(至多一次真实探针;结果缓存于
// Runtime)。观察失败 = 该事实未观察,不是 Start 失败:runtime 进程问题按本
// 适配器惯例在 Execute 以 claim 显形,会话开启不做进程健康门(Start 对
// 二进制缺失仍然即刻显式报错,那是不一样的失败:没有可执行文件)。
func (r *Runtime) observeCLIIdentity(ctx context.Context, bin string) {
	_, _ = r.ensureCLIIdentity(ctx, bin)
}
