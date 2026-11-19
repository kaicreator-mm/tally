package grok

// isolation.go 拥有 per-session GROK_HOME 隔离与 spawn 环境白名单:
//
//   - GROK_HOME 不是缓存:grok home(~/.grok,或 $GROK_HOME;postinstall
//     源码注释「$GROK_HOME (else ~/.grok), matching the Rust grok_home()」)
//     下持久化 sessions(docs:~/.grok/sessions)、config.toml、leader socket
//     (--help:默认 ~/.grok/leader.sock)——先前会话的状态是
//     protocol-observable 的检索面,与 P0-03 之前共享 CODEX_HOME 的风险同类。
//     因此每个会话一个全新临时 home,无任何移交;会话 Close 时删除。空临时
//     GROK_HOME 现场观测可用(grok 1.0.87:--version exit 0,目录保持为空,
//     无 ~/.grok 写入)。
//   - 隔离边界 = 会话:Start 各建新 home;同一会话的多个 Execute 共享同一
//     home(会话是 Tally 的会话边界);本适配器不声明 session.resume/fork,
//     跨会话不存在可续用状态,空 home 不构成语义破坏。
//   - 认证 ≠ home 状态:Grok 认证走 XAI_API_KEY 环境变量(keyless stderr
//     原文:「set the XAI_API_KEY environment variable」;随 Config.Env 注入);
//     ~/.grok 下凭据文件格式未观测,不做移交 —— 无凭据的部署形态将以
//     「Not signed in」的 RunFailed claim 显形(fail-safe),不被隐藏。
//   - spawn 环境走白名单制(比 runtimes/dsh 的「继承 + 强制覆盖」更严格,
//     是本适配器的显式要求):只从 owner 环境拷贝 OS 基础键(进程/function
//     所需,见 envAllowlist),其余一切(包括 owner 环境中的其他 provider
//     认证,如 DEEPSEEK_API_KEY / ANTHROPIC_API_KEY)不进入被 spawn 的
//     agent 及其工具子进程。认证只经 Config.Env 注入。边界补充:白名单含
//     HOME/USERPROFILE(子进程里的 git 等工具需要 home 才能取 .gitconfig),
//     grok 自身对 owner ~/.claude 等目录的配置发现是其 CLI 行为(现场观测
//     `grok inspect` 会列出),不是本适配器的注入;真正的执行边界是 Tally
//     的可丢弃 worktree。
//   - 自动更新经 --no-auto-update 强制关闭(process.go,官方文档 scripted
//     environments 的建议旗标,现场观测接受);即便未来上游改变行为,更新
//     也只会写进 per-session 临时 GROK_HOME 并随 Close 消失。

import (
	"fmt"
	"os"
)

const (
	// grokHomeEnvKey 是 grok home 环境变量名(postinstall 源码观测:
	// process.env.GROK_HOME ?? ~/.grok)。
	grokHomeEnvKey = "GROK_HOME"
)

// envAllowlist 是从 owner 环境拷贝进被 spawn 进程的白名单键:OS/进程运行
// 的基础面,不含任何应用层凭据。PATH 让子进程可运行 git/node 等工具
// (coding agent 的功能前提);HOME/USERPROFILE/HOMEDRIVE/HOMEPATH 让
// 子进程的 git 等工具能解析 home;其余是 Windows/POSIX 系统面。两个平台
// 的键放在同一张表,按「存在才拷贝」处理,无需分平台构建。
var envAllowlist = []string{
	// 进程搜索与系统面(双平台)。
	"PATH",
	"OS",
	"LANG",
	"LC_ALL",
	"NUMBER_OF_PROCESSORS",
	"PROCESSOR_ARCHITECTURE",
	"PROCESSOR_IDENTIFIER",
	"PROCESSOR_LEVEL",
	"PROCESSOR_REVISION",
	// Windows 系统面。
	"SYSTEMROOT",
	"SYSTEMDRIVE",
	"WINDIR",
	"COMSPEC",
	"PATHEXT",
	"TEMP",
	"TMP",
	// home 解析(Windows 走 USERPROFILE/HOMEDRIVE/HOMEPATH,POSIX 走 HOME)。
	"HOME",
	"USERPROFILE",
	"HOMEDRIVE",
	"HOMEPATH",
	// POSIX 临时目录。
	"TMPDIR",
}

// newSessionHome 建立一个会话的隔离 GROK_HOME:全新临时目录,无任何移交。
// 创建失败显式报错(不留半初始化状态)。
func newSessionHome() (string, error) {
	home, err := os.MkdirTemp("", "tally-grok-home-")
	if err != nil {
		return "", fmt.Errorf("grok: 建 per-session GROK_HOME 失败: %w", err)
	}
	return home, nil
}

// sessionSpawnEnv 返回被 spawn 进程的环境:白名单拷贝(owner 环境中存在的
// 键)→ Config.Env → 强制 GROK_HOME。构造序保证后面的覆盖前面的:GROK_HOME
// 被无条件钉到 per-session home(Config.Env 中的同名值不生效;owner 环境的
// 同名键根本不进白名单)。
func sessionSpawnEnv(cfgEnv map[string]string, home string) map[string]string {
	env := make(map[string]string, len(envAllowlist)+len(cfgEnv)+1)
	for _, k := range envAllowlist {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}
	for k, v := range cfgEnv {
		env[k] = v
	}
	env[grokHomeEnvKey] = home
	return env
}
