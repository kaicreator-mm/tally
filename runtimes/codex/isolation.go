package codex

// isolation.go 拥有 P0-03 的 per-run session/home 隔离 —— A0-02 选定并经
// DECISION_LOCKS #22 冻结的生产路径:每个会话一个全新 CODEX_HOME,跨 home 只
// 移交 auth.json,绝不复制/symlink 整个 home。
//
// 设计边界(改动前必读):
//
//   - CODEX_HOME 不是缓存(REF-CODEX-INSTR:全局 AGENTS.md 从 CODEX_HOME 加载,
//     是可观测指令状态;REF-CODEX-ROLLOUT:session/rollout 数据根在
//     CODEX_HOME/sessions)。因此隔离 home 全新创建,只移交 auth.json;
//     owner home 的 config.toml/AGENTS.md/sessions 一律不流入,新会话也不
//     继承任何先前会话的 home 状态。
//   - 隔离边界 = 会话:Start/Fork 各建新 home;同一会话的多个 Execute 共享
//     同一 home,会话内 `codex exec resume <thread>` 的线程连续性因此保持
//     (rollout 就在本会话 home 下)。跨会话 ResumeOf/Fork 指向的 thread 的
//     rollout 位于原会话 home;A0-02 只授权 auth.json 移交,不做 rollout
//     搬运 —— 原 home 已释放时,这类续用以 CLI 的 thread-not-found 显式失败
//     (RunFailed claim),不静默换状态。live 语义由 P0-04 真实 E2E 验证。
//   - 认证可用性 ≠ 会话隔离:owner home 有 auth.json(file auth)则逐会话
//     新拷贝一份(0600,内容原样,复制而非链接);没有(如 keyring 模式 ——
//     REF-CODEX-AUTH:其存储键由 codex_home 路径派生,换 home 即改变认证行为)
//     则不伪造认证:该部署形态与隔离不兼容,按 lock 22 如实记录为不支持,
//     认证失败将以 401 claim 显形,不被隐藏。
//   - home 在 Close 时删除(尽力而为:Close 不等待 in-flight 进程收敛,删除
//     失败不作为 Close 错误)。进程终止而未 Close 的会话,其 home 留在系统
//     临时目录由 OS 清理兜底 —— 与既有 per-turn tmpDir 同一策略。

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// codexHomeEnvKey 是 Codex home 环境变量名。
const codexHomeEnvKey = "CODEX_HOME"

// authFileName 是 file auth 在 CODEX_HOME 下的文件名(REF-CODEX-AUTH)。
// 它是唯一的跨 home 移交物:credential material ≠ 会话/指令状态。
const authFileName = "auth.json"

// newSessionHome 建立一个会话的隔离 CODEX_HOME:全新临时目录 + 仅 auth.json
// 移交。创建或移交失败时目录被回收并显式报错(不留半初始化状态)。
func newSessionHome(cfgEnv map[string]string) (string, error) {
	home, err := os.MkdirTemp("", "tally-codex-home-")
	if err != nil {
		return "", fmt.Errorf("codex: 建 per-run CODEX_HOME 失败: %w", err)
	}
	if err := handOffFileAuth(home, ownerCodexHome(cfgEnv)); err != nil {
		_ = os.RemoveAll(home)
		return "", err
	}
	return home, nil
}

// ownerCodexHome 解析 owner home(auth 移交来源),优先级与二进制发现的
// CODEX_HOME 语义一致:Config.Env 的 CODEX_HOME > 进程 env CODEX_HOME >
// ~/.codex。解析不出(无 UserHomeDir)返回空串 —— 视为无 auth 可移交。
func ownerCodexHome(cfgEnv map[string]string) string {
	if p := strings.TrimSpace(cfgEnv[codexHomeEnvKey]); p != "" {
		return p
	}
	if p := strings.TrimSpace(os.Getenv(codexHomeEnvKey)); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".codex")
}

// handOffFileAuth 把 owner home 的 auth.json 复制(非链接)进隔离 home。
// owner home 无 auth.json(os.ErrNotExist)→ 不移交(nil):keyring 部署按
// 文件头说明处理;其它读取/写入错误显式报错(部署明确意图使用 file auth 时,
// 静默以无认证起跑不如即刻显形)。
func handOffFileAuth(home, owner string) error {
	if owner == "" {
		return nil
	}
	src := filepath.Join(owner, authFileName)
	data, err := os.ReadFile(src) //nolint:gosec // 路径由 ownerCodexHome 结构化解析,非用户输入拼接
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("codex: 读取 owner %s 失败: %w", src, err)
	}
	dst := filepath.Join(home, authFileName)
	if werr := os.WriteFile(dst, data, 0o600); werr != nil {
		return fmt.Errorf("codex: 移交 %s 到隔离 home 失败: %w", authFileName, werr)
	}
	return nil
}

// isolatedSpawnEnv 返回被 spawn 进程的环境:Config.Env 之上强制
// CODEX_HOME=home。隔离 home 压倒 Config.Env 与继承环境中的同名变量 ——
// owner home 只作为 auth 移交来源,绝不出现在被 spawn 进程的 CODEX_HOME 里
// (exec.Cmd.Env 对重复键保留最后一个,追加即可覆盖)。
func isolatedSpawnEnv(cfgEnv map[string]string, home string) map[string]string {
	env := make(map[string]string, len(cfgEnv)+1)
	for k, v := range cfgEnv {
		env[k] = v
	}
	env[codexHomeEnvKey] = home
	return env
}
