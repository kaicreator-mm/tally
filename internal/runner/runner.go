package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// Command 是一次要执行的命令。
//
// Argv 是程序名 + 参数。本地执行(LocalRunner)时 Argv[0] 是可执行文件;
// DockerRunner 会在前面补上 docker 二进制名;SSHRunner 把 Argv 拼成一条远端 shell 命令。
type Command struct {
	Argv []string
	Dir  string   // 工作目录(本地执行用;SSH 忽略)
	Env  []string // 追加的环境变量(本地执行用;SSH 忽略)
}

// Result 是一次执行的原始结果。Runner 不解释它(见 doc.go):
// 非零退出码放在 ExitCode 里,**不**作 error 返回 —— 判断成功失败是调用方的事。
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Runner 执行一条命令并返回原始结果。
//
// error 只表示「没跑起来」:找不到程序、ctx 超时/取消、SSH 连不上。
// 命令跑完了但退出码非零,不是 error,是 Result.ExitCode。
type Runner interface {
	Run(ctx context.Context, cmd Command) (Result, error)
}

// LocalRunner 用 os/exec 在本机执行命令。是 DockerRunner 的默认底座,也可单独用。
type LocalRunner struct{}

// Run 实现 Runner。
func (LocalRunner) Run(ctx context.Context, cmd Command) (Result, error) {
	if len(cmd.Argv) == 0 {
		return Result{}, errors.New("runner: 空命令")
	}
	c := exec.CommandContext(ctx, cmd.Argv[0], cmd.Argv[1:]...)
	c.Dir = cmd.Dir
	if len(cmd.Env) > 0 {
		c.Env = append(c.Environ(), cmd.Env...)
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	start := time.Now()
	runErr := c.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String(), Duration: time.Since(start)}
	if runErr != nil {
		// ctx 到期/取消导致进程被杀:算「没跑完」,上抛而不是落进 ExitCode。
		if ctx.Err() != nil {
			return res, fmt.Errorf("执行 %q: %w", cmd.Argv[0], ctx.Err())
		}
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			res.ExitCode = ee.ExitCode() // 跑完了,退出码非零不是 error
			return res, nil
		}
		return res, fmt.Errorf("执行 %q: %w", cmd.Argv[0], runErr) // 程序没起来
	}
	return res, nil
}
