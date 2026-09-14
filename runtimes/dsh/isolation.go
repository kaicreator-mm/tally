package dsh

// isolation.go 拥有 per-session DSH_HOME 隔离与 spawn 环境硬化 —— P4-03
// spike 条件 2/3 的生产路径(DL-15:DSH 在 Tally-owned isolation 下运行):
//
//   - DSH_HOME 不是缓存:默认 ~/.dsh 下 sessions/storages/profiles 跨 run
//     持久化(spike §2.6 观测),先前会话的内部状态是 protocol-observable
//     的检索面 —— 与 P0-03 之前共享 CODEX_HOME 的风险同类。因此每个会话一个
//     全新临时 home,无任何移交;会话 Close 时删除。
//   - 隔离边界 = 会话:Start 各建新 home;同一会话的多个 Execute 共享同一
//     home(会话是 Tally 的会话边界);本适配器不声明 session.resume/fork,
//     跨会话不存在可续用状态,空 home 不构成语义破坏。
//   - 认证 ≠ home 状态:DSH 认证走 DEEPSEEK_API_KEY 环境变量(spike §2.6
//     观测的 supported 路径,随继承环境或 Config.Env 注入);~/.dsh 下的
//     凭据文件格式未观测,不做移交 —— 该部署形态将以 MISSING_CREDENTIAL
//     的 RunFailed claim 显形(fail-safe),不被隐藏。
//   - DSH_TELEMETRY_MODE=DISABLED 强制:默认 FEEDBACK_ONLY 把遥测送到
//     DeepSeek OTLP 端点(spike §2.2 dump),候选执行不允许该出口。
//   - DSH_PERMISSION_MODE 强制为 Config 值(默认 workspace-write):继承
//     环境中的 danger-full-access 升级不得静默生效,升级只经显式 Config。
//     权限边界本身不构成安全控制(上游 SAFETY.md):真正的隔离边界是 Tally
//     的可丢弃 worktree,DSH 永远只在其内执行。

import (
	"fmt"
	"os"
)

const (
	// dshHomeEnvKey 是 DSH home 环境变量名(spike §2.6:默认 ~/.dsh)。
	dshHomeEnvKey = "DSH_HOME"
	// dshTelemetryModeEnvKey 是遥测模式环境变量名(spike §2.2 dump 观测)。
	dshTelemetryModeEnvKey = "DSH_TELEMETRY_MODE"
	// dshPermissionModeEnvKey 是权限模式环境变量名(spike §2.2 dump 观测)。
	dshPermissionModeEnvKey = "DSH_PERMISSION_MODE"
)

// telemetryDisabled 是强制写入的遥测模式值:上游观测枚举中的完全关闭档
// (默认 FEEDBACK_ONLY 会向 harness-telemetry.deepseeksvc.com 送 OTLP)。
const telemetryDisabled = "DISABLED"

// newSessionHome 建立一个会话的隔离 DSH_HOME:全新临时目录,无任何移交。
// 创建失败显式报错(不留半初始化状态)。
func newSessionHome() (string, error) {
	home, err := os.MkdirTemp("", "tally-dsh-home-")
	if err != nil {
		return "", fmt.Errorf("dsh: 建 per-session DSH_HOME 失败: %w", err)
	}
	return home, nil
}

// sessionSpawnEnv 返回被 spawn 进程的环境:Config.Env 之上强制三个键。
// exec.Cmd.Env 由 os.Environ() 与追加对组成,对重复键保留最后一个
// (exec 侧去重,last-wins),因此这里的赋值压倒继承环境与 Config.Env 中
// 的同名变量。三个键的语义见文件头。
func sessionSpawnEnv(cfgEnv map[string]string, home, permission string) map[string]string {
	env := make(map[string]string, len(cfgEnv)+3)
	for k, v := range cfgEnv {
		env[k] = v
	}
	env[dshHomeEnvKey] = home
	env[dshTelemetryModeEnvKey] = telemetryDisabled
	env[dshPermissionModeEnvKey] = permission
	return env
}
