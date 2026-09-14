// runtimefactory.go(P4-01)是 worker 的 runtime 组合注册表:把 v13/v14 组合
// 根里按 runtime 名字选择适配器的硬编码 switch,收拢为一个 name → builder 的
// 小注册表。
//
// 边界(AGENTS.md §3 Runtime lock / DL-02 / PRD §10.3):
//   - 注册表只做「名字 → agentruntime.Runtime 适配器」的组合选择;它不是第二
//     执行协议,也不定义任何新执行核心接口 —— 产出值就是既有 agentruntime.Runtime。
//   - 它是 worker 内唯一按 runtime 名字分支的点,且只存在于组合边界:
//     workflow / closedloop / candidateverify 内核只接收 agentruntime.Runtime
//     值,控制流只按能力协商结果分支(PRD §10.3),绝不按名字分支。
//   - 能力逻辑保持 runtime 中立(如 enforceHeldOutPrerequisites 按
//     CapabilitySet 裁决):注册表不承载、不改写任何能力声明。
//   - 未知 runtime 在任务执行前被拒绝(fail-closed):不开一次性 worktree、
//     不开 runtime 会话、不落任何 run 事实。
//
// 扩展缝(DeepSeek Harness 等,Runtime lock:仍走 runtimes/<rt> 外部进程形态):
// 新适配器在 newRuntimeRegistry 登记一个 builder 即被 v13/v14 两条路径同时
// 选用,组合流程零改动;本文件随之仍是唯一的新增名字分支点。
package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"tally/internal/agentruntime"
	"tally/runtimes/codex"
	"tally/runtimes/dsh"
	"tally/runtimes/fake"
)

// runtimeBuilder 按 worker 共享配置面构造一个 runtime 适配器。builder 只在
// 组合边界被调用:输入是配置(v13Config 的 runtime 选择字段),输出是协议值
// (agentruntime.Runtime)—— 接触不到任何内核类型,也不得携带 Git/验证/验收
// 权威(DL-03)。
type runtimeBuilder func(cfg v13Config) (agentruntime.Runtime, error)

// runtimeRegistry 是 runtime 适配器的组合注册表(名字 → 构造器)。
type runtimeRegistry struct {
	builders map[string]runtimeBuilder
	// defaultName 是配置缺省(Runtime 名为空)时的选择(FakeRuntime 缺省,
	// CodexRuntime 显式配置)。
	defaultName string
}

// workerRuntimes 是生产注册表:v13/v14 组合共用同一实例;登记只发生在构造时。
var workerRuntimes = newRuntimeRegistry()

// newRuntimeRegistry 构造注册表并登记全部生产适配器:fake(缺省)与 codex。
// 登记面即支持集合:选择失败的错误信息按登记键序推导,后续适配器(如 DSH)
// 登记后自动进入提示,无需另改错误文案。
func newRuntimeRegistry() *runtimeRegistry {
	r := &runtimeRegistry{
		builders:    make(map[string]runtimeBuilder, 2),
		defaultName: fake.DescriptorName,
	}
	r.register(fake.DescriptorName, func(v13Config) (agentruntime.Runtime, error) {
		// FakeRuntime 是确定性 reference 实现,零配置(缺省 runtime)。
		return fake.New(fake.Config{}), nil
	})
	r.register(codex.DescriptorName, func(cfg v13Config) (agentruntime.Runtime, error) {
		// CodexRuntime:外部 codex CLI 进程;BinPath 显式指定,空则由适配器
		// 按既有探针顺序发现(env CODEX_CLI_PATH → config.toml → PATH)。
		return codex.New(codex.Config{BinPath: cfg.CodexBin}), nil
	})
	// P4-06 canary:DSH developer-preview 适配器缺省禁用 —— runtime=dsh 且
	// canary 开关(TALLY_DSH_CANARY=1)且显式二进制路径三者同时出现才可构造;
	// 任一缺席在任务执行前类型化失败(fail-closed:选择失败绝不开 runtime
	// 会话、绝不建 worktree、绝不落 run 事实)。canary 关闭时 Codex 缺省面
	// 零影响(AC)。
	r.register(dsh.DescriptorName, func(cfg v13Config) (agentruntime.Runtime, error) {
		if !cfg.DSHCanary {
			return nil, fmt.Errorf("dsh 为 developer-preview canary 适配器,默认禁用:需 TALLY_DSH_CANARY=1 显式启用(canary/bounded-task 策略见 docs/guides/dsh-canary.md)")
		}
		if strings.TrimSpace(cfg.DSHBin) == "" {
			return nil, fmt.Errorf("dsh canary 需要显式二进制路径 TALLY_DSH_BIN(钉死 developer-preview 版本面,不做隐式发现)")
		}
		dcfg := dsh.Config{
			BinPath:       cfg.DSHBin,
			Model:         cfg.DSHModel,
			Provider:      cfg.DSHProvider,
			PinnedVersion: dsh.PinnedDSHVersion,
		}
		// 最早闸:权限枚举/overlay token 等配置面在此拒绝(P4-06 AC)。
		if err := dcfg.Validate(); err != nil {
			return nil, fmt.Errorf("dsh 配置无效: %w", err)
		}
		rt := dsh.New(dcfg)
		// 配置有效性闸(Invalid configuration fails before task execution):
		// Start 的钉版身份探针 + overlay token 校验在任何会话/工作树之前
		// 执行;此处预检二进制可解析,把路径错误也提前到执行前。
		if strings.TrimSpace(cfg.DSHBin) == "" {
			if _, err := dsh.ResolveBinaryPath(); err != nil {
				return nil, fmt.Errorf("dsh 二进制不可解析: %w", err)
			}
		}
		return rt, nil
	})
	return r
}

// register 登记一个适配器构造器(组合根构造与测试注入用;生产登记全部集中
// 在 newRuntimeRegistry)。同名重复登记以最后登记为准 —— 登记点唯一,不构成
// 运行时分支。
func (r *runtimeRegistry) register(name string, build runtimeBuilder) {
	r.builders[name] = build
}

// build 按配置的 Runtime 名字构造适配器:空名字取缺省(fake);未登记的名字
// 返回错误,由调用方在任务执行前拒绝 —— 选择失败绝不开 runtime 会话、绝不建
// 一次性 worktree、绝不落 run 事实(fail-closed)。
func (r *runtimeRegistry) build(cfg v13Config) (agentruntime.Runtime, error) {
	name := cfg.Runtime
	if name == "" {
		name = r.defaultName
	}
	b, ok := r.builders[name]
	if !ok {
		return nil, fmt.Errorf("未知 runtime %q(支持 %s)", name,
			strings.Join(slices.Sorted(maps.Keys(r.builders)), "|"))
	}
	return b(cfg)
}
