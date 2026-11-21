package opencode

// identity.go 拥有钉版身份闸的版本探针与 ObservedIdentity 缝:把本 runtime
// 现场「实际」观察到的身份事实映射进 runtime 中立的
// agentruntime.ObservedIdentity(P0-01;Decision Lock 15:实际身份 !=
// 期望配置,两者必须分开携带、分开落库)。
//
// 观察语义边界(改动前必读):
//
//   - 只报告已建立的观察,绝不用期望配置回填:
//       * RuntimeVersion 来自一次真实的 `opencode --version` 进程观测(由
//         Start/Snapshot 建立、Runtime 级缓存、跨会话共享;观测形态是裸版本
//         行 "1.19.6",exit 0)。适配器版本 DescriptorVersion 与它是两个
//         事实,绝不写入观察字段(AGENTS.md §13)。
//       * RuntimeName 恒为空:`opencode --version` 只输出裸版本号,不自报
//         产品名,无可观察的名字,不发明。
//       * Model 恒为空:run 表面不逐 run 报告实际生效模型;Config.Model/
//         Agent 是期望配置,把它当「实际生效」上报即伪造观察(requested ≠
//         observed 永不相等,AGENTS.md §13)。
//       * SessionID 恒为空:`--format json` 事件流可观察到每 run 的真实
//         ses_* 句柄,但它是「一次运行」的观察事实而非「会话建立」时刻的
//         事实,且解析属尽力而为 —— 按任务边界刻意不上报(保守映射);
//         适配器自造的会话编号(AgentRunResult.SessionID)是 Tally 协议层
//         标识,不是 opencode 的观察事实,不写入本字段。
//   - Identity() 只读、非阻塞、可并发调用;不在调用时发起新观察(不发进程、
//     不做网络请求),只返回已建立的观察(agentruntime.IdentityReporter 契约)。
//   - 探针失败(版本闸拒绝会话)时本接口不会被触达 —— Start 已 fail-closed;
//     全零值(未观察)仍是合法状态,如实返回,不报错。

import (
	"context"
	"fmt"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

// cliIdentity 是一次 `opencode --version` 现场观察的解码结果:CLI 对自身
// 版本的自我报告。观测形态是裸版本行,无产品名自报 —— Name 刻意没有取值
// 路径。零值 = 尚未观察(缓存未建立或探针失败)。
type cliIdentity struct {
	Version string // CLI 自报版本(如 "1.19.6")
}

// observed 返回该观察是否携带任何事实。
func (c cliIdentity) observed() bool { return c.Version != "" }

// parseCLIIdentity 从 `opencode --version` 输出解码 CLI 自报版本(观测形态:
// 裸版本行,取首个非空行原文,不发明格式)。空输出返回零值。
func parseCLIIdentity(out string) cliIdentity {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return cliIdentity{Version: line}
		}
	}
	return cliIdentity{}
}

// probeVersion 执行一次 `<bin> --version` 探针:非零退出或进程启动失败都
// 显式报错(钉版闸的硬前提,fail-closed)。env 是 Config.Env 的 KEY=VALUE
// 列表(与身份探针同面;探针不触碰任何配置/数据目录,隔离键不参与)。
func probeVersion(ctx context.Context, bin string, env []string) (string, error) {
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{Argv: []string{bin, "--version"}, Env: env})
	if err != nil {
		return "", fmt.Errorf("opencode: 读取 --version 失败: %w", err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("opencode: --version 退出码 %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	return res.Stdout, nil
}

// Identity 实现 agentruntime.IdentityReporter(P0-01 可选缝)。返回该会话
// 当前已观察到的实际身份快照;语义见文件头。
func (s *Session) Identity() agentruntime.ObservedIdentity {
	r := s.runtime
	r.mu.Lock()
	cli := r.cli
	r.mu.Unlock()
	return agentruntime.ObservedIdentity{
		RuntimeName:    "", // `opencode --version` 不自报产品名:未观察,不发明
		RuntimeVersion: cli.Version,
		Model:          "", // run 表面不逐 run 报告实际模型:未观察,绝不回填期望配置
		SessionID:      "", // 每 run 的 ses_* 句柄刻意不上报(保守映射,见文件头)
	}
}
