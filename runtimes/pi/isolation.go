package pi

// isolation.go 拥有 per-session 隔离目录组与 spawn 环境强制面 —— Tally Host
// 隔离的生产路径(pi 不提供可信完整 OS 沙箱,绝不依赖 pi 自身沙箱声明):
//
//   - cwd = 可丢弃 worktree(StartRequest.Workspace,INV-EXEC-01),由
//     process.go 的 spawn cwd 保证;
//   - 每会话一组全新临时目录(观测:pi 启动时会向 config 目录脚手架写入
//     auth.json / models-store.json,实测内容为空对象 "{}";重定向后 owner
//     的 ~/.pi 零触碰 —— 实测重定向下真实 ~/.pi 的文件 mtime 不变):
//       * agentConfig → PI_CODING_AGENT_DIR(pi 帮助文本观测的 config 目录
//         环境变量,默认 ~/.pi/agent);
//       * sessions   → PI_CODING_AGENT_SESSION_DIR 与 --session-dir(pi 帮助
//         文本注明旗标压倒环境变量;--no-session 下实测无文件落盘,目录是
//         防御纵深);
//       * home       → HOME / USERPROFILE(~ 解析兜底重定向,实测可行);
//     Close 时整组删除;进程终止而未 Close 的会话由 OS 临时目录清理兜底。
//   - spawn 环境强制面:「干净 env」按凭据/配置面解释 —— 继承 os.Environ()
//     但剥离 pi 帮助文本观测的全部 provider 凭据环境变量与 pi 配置环境开关
//    (防 operator 环境的 ANTHROPIC_API_KEY / OPENAI_API_KEY / DEEPSEEK_API_KEY
//     等静默生效、防隐式读写 owner 之外的认证/配置目录),强制六个隔离键,
//     再层叠 Config.Env(唯一的显式凭据注入通道)。刻意不做「全空 env」:
//     未观测过全空 env 下 pi 工具子进程(bash/powershell)与 Windows TLS
//     (SystemRoot)的存活面,保留 OS 功能面不是认证通道。
//   - 强制键压倒 Config.Env 同名值(大小写不敏感匹配,Windows 环境变量名
//     不区分大小写):PI_CODING_AGENT_DIR / PI_CODING_AGENT_SESSION_DIR /
//     HOME / USERPROFILE / PI_OFFLINE / PI_TELEMETRY。
//   - PI_OFFLINE=1(帮助文本:禁用启动期网络操作;实测不阻碍真实模型回合)
//     与 PI_TELEMETRY=0(帮助文本:关闭安装遥测)是出口硬化,候选执行不允许
//     遥测/目录刷新类额外出口。

import (
	"fmt"
	"os"
	"strings"
)

// piCredentialEnvVars 是 pi 帮助文本(0.85.1)观测的全部 provider 凭据环境
// 变量:spawn 一律剥离,operator 环境的凭据不得静默流入候选执行;显式注入
// 只经 Config.Env(见 spawnEnvList)。
var piCredentialEnvVars = []string{
	"ANTHROPIC_AUTH_TOKEN",
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_OAUTH_TOKEN",
	"ANT_LING_API_KEY",
	"OPENAI_API_KEY",
	"AZURE_OPENAI_API_KEY",
	"AZURE_OPENAI_BASE_URL",
	"AZURE_OPENAI_RESOURCE_NAME",
	"AZURE_OPENAI_API_VERSION",
	"AZURE_OPENAI_DEPLOYMENT_NAME_MAP",
	"DEEPSEEK_API_KEY",
	"NVIDIA_API_KEY",
	"GEMINI_API_KEY",
	"GROQ_API_KEY",
	"CEREBRAS_API_KEY",
	"XAI_API_KEY",
	"FIREWORKS_API_KEY",
	"TOGETHER_API_KEY",
	"BASETEN_API_KEY",
	"OPENROUTER_API_KEY",
	"AI_GATEWAY_API_KEY",
	"ZAI_API_KEY",
	"ZAI_CODING_CN_API_KEY",
	"MISTRAL_API_KEY",
	"MINIMAX_API_KEY",
	"MOONSHOT_API_KEY",
	"OPENCODE_API_KEY",
	"KIMI_API_KEY",
	"CLOUDFLARE_API_KEY",
	"CLOUDFLARE_ACCOUNT_ID",
	"CLOUDFLARE_GATEWAY_ID",
	"QWEN_TOKEN_PLAN_API_KEY",
	"QWEN_TOKEN_PLAN_CN_API_KEY",
	"XIAOMI_API_KEY",
	"XIAOMI_TOKEN_PLAN_CN_API_KEY",
	"XIAOMI_TOKEN_PLAN_AMS_API_KEY",
	"XIAOMI_TOKEN_PLAN_SGP_API_KEY",
	"AWS_PROFILE",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_BEARER_TOKEN_BEDROCK",
	"AWS_REGION",
}

// piConfigEnvVars 是 pi 帮助文本观测的配置/行为环境开关(凭据之外):spawn
// 一律剥离后按隔离语义强制(offline/telemetry 硬化;config/session 目录
// 重定向;package 目录防意外发现面;share URL 防环境漂移)。
var piConfigEnvVars = []string{
	"PI_CODING_AGENT_DIR",
	"PI_CODING_AGENT_SESSION_DIR",
	"PI_PACKAGE_DIR",
	"PI_OFFLINE",
	"PI_TELEMETRY",
	"PI_SHARE_VIEWER_URL",
}

// 强制写入的隔离键值(值来自 per-session 目录组)。
const (
	envHome           = "HOME"
	envUserProfile    = "USERPROFILE"
	envPiAgentDir     = "PI_CODING_AGENT_DIR"
	envPiSessionDir   = "PI_CODING_AGENT_SESSION_DIR"
	envPiOffline      = "PI_OFFLINE"
	envPiTelemetry    = "PI_TELEMETRY"
	offlineForced     = "1"
	telemetryDisabled = "0"
)

// sessionDirs 是一个会话的隔离目录组:root 之下三个子目录,Close 时整组删除。
type sessionDirs struct {
	// root 是本会话的临时根目录(其他三个都在其下)。
	root string
	// agentConfig 是重定向的 pi config 目录(PI_CODING_AGENT_DIR)。
	agentConfig string
	// sessions 是重定向的会话存储目录(--session-dir 与
	// PI_CODING_AGENT_SESSION_DIR)。
	sessions string
	// home 是重定向的 ~ 目录(HOME / USERPROFILE)。
	home string
}

// newSessionDirs 建立一个会话的隔离目录组:全新临时根 + 三个子目录。创建
// 失败显式报错(不留半初始化状态;已创建部分一并回收)。
func newSessionDirs() (*sessionDirs, error) {
	root, err := os.MkdirTemp("", "tally-pi-session-")
	if err != nil {
		return nil, fmt.Errorf("pi: 建 per-session 隔离目录失败: %w", err)
	}
	d := &sessionDirs{
		root:        root,
		agentConfig: root + string(os.PathSeparator) + "agent-config",
		sessions:    root + string(os.PathSeparator) + "sessions",
		home:        root + string(os.PathSeparator) + "home",
	}
	for _, dir := range []string{d.agentConfig, d.sessions, d.home} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			_ = os.RemoveAll(root)
			return nil, fmt.Errorf("pi: 建 per-session 隔离子目录失败: %w", err)
		}
	}
	return d, nil
}

// Close 删除整组隔离目录(尽力而为:删除失败不阻塞会话收尾,残留由 OS
// 临时目录清理兜底)。
func (d *sessionDirs) Close() error {
	if d == nil || d.root == "" {
		return nil
	}
	return os.RemoveAll(d.root)
}

// spawnEnv 返回被 spawn 进程的环境(键 → 值;process.go 负责 token 化与
// 顺序)。语义:
//  1. 继承环境剥离 pi 凭据/配置面(大小写不敏感);
//  2. 强制六个隔离键(压倒一切同名值,大小写不敏感);
//  3. Config.Env 层叠(强制键以外的显式注入,如 provider API key)。
func spawnEnv(cfgEnv map[string]string, dirs *sessionDirs) map[string]string {
	forced := map[string]string{
		envHome:         dirs.home,
		envUserProfile:  dirs.home,
		envPiAgentDir:   dirs.agentConfig,
		envPiSessionDir: dirs.sessions,
		envPiOffline:    offlineForced,
		envPiTelemetry:  telemetryDisabled,
	}
	env := make(map[string]string, len(cfgEnv)+len(forced))
	// 继承环境(剥离面)。
	for _, kv := range os.Environ() {
		name := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			name = kv[:i]
		}
		if envNameStripped(name) || envNameForced(name) {
			continue
		}
		// 继承环境本就无重复键;Windows 大小写不敏感场景由 envNameStripped /
		// envNameForced 的折叠匹配兜住,同名(异大小写)残留不可能出现。
		env[name] = kv[len(name)+1:]
	}
	// 强制隔离键(压倒继承与 Config.Env)。
	for k, v := range forced {
		env[envCanonicalKey(env, k)] = v
	}
	// Config.Env 显式注入(强制键除外)。
	keys := make([]string, 0, len(cfgEnv))
	for k := range cfgEnv {
		if envNameForced(k) {
			continue
		}
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		env[envCanonicalKey(env, k)] = cfgEnv[k]
	}
	return env
}

// envNameStripped 报告环境变量名是否在剥离面(凭据 + pi 配置开关;Windows
// 环境变量名大小写不敏感,统一折叠匹配)。
func envNameStripped(name string) bool {
	for _, s := range piCredentialEnvVars {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	for _, s := range piConfigEnvVars {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	return false
}

// envNameForced 报告环境变量名是否是被强制的隔离键(压倒 Config.Env)。
func envNameForced(name string) bool {
	for _, f := range []string{envHome, envUserProfile, envPiAgentDir, envPiSessionDir, envPiOffline, envPiTelemetry} {
		if strings.EqualFold(name, f) {
			return true
		}
	}
	return false
}

// envCanonicalKey 清除 env 中与 name 大小写不敏感同名的键并返回规范键名
// (Windows 环境变量名大小写不敏感:同一变量的异大小写形态不得共存)。
func envCanonicalKey(env map[string]string, name string) string {
	for k := range env {
		if strings.EqualFold(k, name) {
			delete(env, k)
			return name
		}
	}
	return name
}

// spawnEnvList 把 spawnEnv 的结果转成 KEY=VALUE 切片(键序确定),供
// exec.Cmd.Env 使用。
func spawnEnvList(cfgEnv map[string]string, dirs *sessionDirs) []string {
	return envPairs(spawnEnv(cfgEnv, dirs))
}

// envPairs 把 map 转成键序确定的 KEY=VALUE 切片(CommandSpec.Env 语义)。
func envPairs(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sortStrings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+env[k])
	}
	return out
}

// sortStrings 就地排序(小工具,避免为此引入 slices 依赖差异)。
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}
