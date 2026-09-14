//go:build !windows

package dsh

// POSIX 上的整树终止(P4-03 spike 条件 4 / R-1 的等价物):launcher 自成
// 进程组(Setpgid),取消时对整个进程组发 SIGKILL —— os.Process.Kill 只终止
// launcher 本身,子进程树会存活成孤儿。

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcAttr 让 launcher 自成进程组(组长 = 自身),整树终止以组为单位。
func setProcAttr(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killTree 终止以 p 为根的整个进程组(负 pid = 组)。
func killTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	return syscall.Kill(-p.Pid, syscall.SIGKILL)
}
