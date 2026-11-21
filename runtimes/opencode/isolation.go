package opencode

// isolation.go 拥有 per-session 隔离目录树与 spawn 环境硬化(DL-15:
// opencode 在 Tally-owned isolation 下运行):
//
//   - opencode 的全局状态面(config/data/cache/state)默认落在用户级
//     XDG 目录(owner 实测:~/.config/opencode、~/.local/share/opencode,
//     后者含 auth.json 凭据、log、session 存储与 snapshot)。跨 run 复用
//     它们 = 先前会话的会话状态/凭据/日志成为 protocol-observable 的检索
//     面(与 P0-03 之前共享 CODEX_HOME、P4-03 之前共享 ~/.dsh 同类风险)。
//     `opencode debug paths` 观测:四个路径全部跟随 XDG_CONFIG_HOME /
//     XDG_DATA_HOME / XDG_CACHE_HOME / XDG_STATE_HOME(追加 "opencode"
//     子目录),因此每个会话一个全新临时目录树,经这四个变量重定向;无任何
//     移交;会话 Close 时删除。owner 目录零读取零写入(实测 owner log 目录
//     mtime 在隔离 run 前后不变)。
//   - 隔离边界 = 会话:Start 各建新目录树;同一会话的多个 Execute 共享同一
//     棵(会话是 Tally 的会话边界);本适配器不声明 session.resume/fork,
//     跨会话不存在可续用状态,空目录树不构成语义破坏。
//   - 认证 ≠ 目录状态:opencode 经环境变量识别 provider API key
//     (providers list 观测 "Environment: DeepSeek DEEPSEEK_API_KEY"),
//     随 Config.Env 注入;隔离目录树内的 auth.json 恒为空(无凭据落盘)。
//     无凭据不必然失败(观测到 CLI 落免凭据默认模型成功)—— 适配器不做
//     凭据存在性假设,后果由真实退出码/输出显形。
//   - TMP/TEMP 同步指向会话临时目录:平台级 temp 重定向(opencode 的
//     `tmp` 路径跟随 OS temp,debug paths 观测),让子进程的临时文件活动
//     也收敛进会话目录树。这是通用进程环境语义,不是 opencode 私有契约。
//   - exec.Cmd.Env 由 os.Environ() 与追加对组成,对重复键保留最后一个
//     (exec 侧去重,last-wins),因此这里的赋值压倒继承环境与 Config.Env
//     中的同名变量(owner 部署面的 XDG_* / TMP / TEMP 不得静默生效)。

import (
	"fmt"
	"os"
	"path/filepath"
)

// sessionDirs 是一个会话的隔离目录树:根目录下五个子目录,分别作为 XDG
// 四根与 OS temp 的重定向目标(opencode 在每个 XDG 根下自建 "opencode"
// 子目录,观测见包注释)。
type sessionDirs struct {
	root string
}

// xdgKeys 是重定向到会话目录树的环境变量 → 子目录名映射(键序确定)。
var xdgKeys = []struct {
	env string
	sub string
}{
	{"XDG_CONFIG_HOME", "config"},
	{"XDG_DATA_HOME", "data"},
	{"XDG_CACHE_HOME", "cache"},
	{"XDG_STATE_HOME", "state"},
}

// newSessionDirs 建立一个会话的隔离目录树:全新临时根 + 五个子目录,无任何
// 移交。创建失败显式报错(不留半初始化状态;Start 直接失败)。
func newSessionDirs() (sessionDirs, error) {
	root, err := os.MkdirTemp("", "tally-opencode-session-")
	if err != nil {
		return sessionDirs{}, fmt.Errorf("opencode: 建 per-session 隔离目录树失败: %w", err)
	}
	d := sessionDirs{root: root}
	mk := func(sub string) error {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o700); err != nil {
			return fmt.Errorf("opencode: 建隔离子目录 %s 失败: %w", sub, err)
		}
		return nil
	}
	for _, k := range xdgKeys {
		if err := mk(k.sub); err != nil {
			d.remove()
			return sessionDirs{}, err
		}
	}
	if err := mk("tmp"); err != nil {
		d.remove()
		return sessionDirs{}, err
	}
	return d, nil
}

// env 返回会话的隔离环境覆盖:六个键全部压倒继承环境与 Config.Env 中的
// 同名值(exec last-wins 语义)。XDG 四根指向各自子目录;TMP/TEMP 共享
// tmp 子目录。
func (d sessionDirs) env() map[string]string {
	out := make(map[string]string, len(xdgKeys)+2)
	for _, k := range xdgKeys {
		out[k.env] = filepath.Join(d.root, k.sub)
	}
	tmp := filepath.Join(d.root, "tmp")
	out["TMP"] = tmp
	out["TEMP"] = tmp
	return out
}

// remove 删除整棵会话目录树(尽力而为:Close 收尾,失败由 OS 临时目录
// 清理兜底,不作为 Close 错误)。空值安全(未初始化的零值无害)。
func (d sessionDirs) remove() {
	if d.root == "" {
		return
	}
	_ = os.RemoveAll(d.root)
}
