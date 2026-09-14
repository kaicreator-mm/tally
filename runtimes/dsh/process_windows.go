//go:build windows

package dsh

// Windows 上的整树终止(P4-03 spike 条件 4 / R-1):os.Process.Kill 只终止
// launcher 进程本身,dsh launcher(Node)的子进程树会存活成孤儿 —— 不可接受
// 的候选执行残留。这里用 Windows 自带的 taskkill /T(整树)+ /F(强制)做
// 可靠 teardown:结构化 argv、无 shell 解析(AGENTS.md §11)。
//
// 为什么不是 Job Objects:Job Objects 需要 golang.org/x/sys/windows,而它是
// 依赖白名单(test/arch/deps_test.go INV-6)之外的间接依赖,加白名单须经
// 批准的架构变更;taskkill /T /F 是 Windows 文档化的等价整树终止设施,满足
// spike 条件 4 的「Job Objects or equivalent reliable teardown」。
//
// 兜底:exec.Cmd.WaitDelay(runtime.go)封顶「进程已退出但输出管道仍被孤儿
// 子进程占用」的 Wait 阻塞;外部单进程 kill(诊断注入面)留下的孤儿由它
// 保证 Execute 仍收敛。

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// treeKillTimeout 是单次 taskkill 的上界;超时放弃(调用方按 best-effort
// 处理,真实收敛由进程退出语义保证)。
const treeKillTimeout = 10 * time.Second

// setProcAttr 在 spawn 前设置平台进程属性(Windows:无需额外属性;
// taskkill /T 按父子关系在调用时刻枚举整树,不要求新进程组)。
func setProcAttr(cmd *exec.Cmd) {}

// killTree 终止以 p 为根的整棵进程树。
func killTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), treeKillTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "taskkill", "/PID", strconv.Itoa(p.Pid), "/T", "/F").CombinedOutput()
	if err != nil {
		return fmt.Errorf("dsh: taskkill /T /F pid=%d 失败: %v: %s", p.Pid, err, truncate(string(out), 200))
	}
	return nil
}
