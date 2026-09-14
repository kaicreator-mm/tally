// Package command 定义 Tally 的结构化命令值类型 CommandSpec。
//
// CommandSpec 只描述「要执行什么」(argv/cwd/env/timeout/shell),不执行任何进程:
// repro(PRD §21.6)、required gate(PRD §8.11)与依赖退役命令等路径统一用它
// 取代 shell 字符串 + strings.Fields 的表示。执行归 runner/执行器所有。
package command

import (
	"errors"
	"time"
)

// ErrEmptyArgv 表示 shell=false 时 argv 为空。结构化命令不允许隐式空命令。
var ErrEmptyArgv = errors.New("command: argv must not be empty when shell is false")

// CommandSpec 是一条结构化可执行命令的期望描述,按值使用。
//
// 字段语义:
//   - Argv: 程序名 + 参数,与 runner.Command.Argv 对齐;禁止用 shell 字符串再切分。
//   - Cwd: 工作目录;"" 表示未指定,由执行环境决定。
//   - Env: 追加到执行环境的环境变量(不是完整环境快照);nil 与空 map 等价。
//   - Timeout: 建议超时;0 表示未设置,实际预算由 ExecutionPolicy 决定。
//   - Shell: 只有显式置 true 才允许 shell 语义;零值为 false(PRD §32 默认拒绝)。
type CommandSpec struct {
	Argv    []string
	Cwd     string
	Env     map[string]string
	Timeout time.Duration
	Shell   bool
}

// Validate 校验 CommandSpec 的最小结构约束。
//
// 当前唯一硬约束:shell=false 时 argv 不能为空——空 argv 只可能是漏填,
// 不允许静默落到「什么也不执行」。
func (s CommandSpec) Validate() error {
	if !s.Shell && len(s.Argv) == 0 {
		return ErrEmptyArgv
	}
	return nil
}
