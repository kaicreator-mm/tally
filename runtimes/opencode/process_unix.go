//go:build !windows

package opencode

// POSIX 上的整树终止(Windows taskkill /T /F 的等价物):opencode 进程自成
// 进程组(Setpgid),取消时对整个进程组发 SIGKILL —— os.Process.Kill 只终止
// 进程本身,子进程树会存活成孤儿。(与 runtimes/dsh 同一结构。)

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// setProcAttr 让 opencode 进程自成进程组(组长 = 自身),整树终止以组为单位。
func setProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killTree 终止 p 所在的整个进程组。正常生产调用传入 opencode 进程(组长),
// 但崩溃清理/诊断可能只剩 surviving child;此时不能假设 p.Pid == PGID,
// 必须先查询实际 PGID,再向该组发 SIGKILL。这样 opencode 已死但同组子进程
// 仍存活时也能可靠收敛,不留下孤儿。
func killTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	pgid, err := syscall.Getpgid(p.Pid)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
