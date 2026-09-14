package dsh

// process.go 拥有 `dsh --profile headless` 进程生命周期:结构化 argv 组装、
// spawn、stdout 最终回答读取、取消/崩溃处理与终态分类。全部旗标与退出码
// 语义来自 P4-03 spike 对 @deepseek-ai/dsh@0.1.5-rc.1 的观测,未观测的
// 旗标/表面一律不使用、不解释(AGENTS.md §3 Runtime lock)。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/command"
	"tally/internal/logpipe"
)

// execTurn 执行一个完整 turn:组装结构化 argv → spawn(cwd 钉 workspace)→
// 读取 stdout 最终回答 → wait → 分类终态 → 返回 claim。
func (s *Session) execTurn(ctx context.Context, r0 *run, instruction string) agentruntime.AgentRunResult {
	s.execMu.Lock()
	defer s.execMu.Unlock()

	// ctx 已取消的 turn 是正常取消终态,不以 RunFailed 伪装(不把取消伪造
	// 成候选失败;trust invariant)。
	if ctx.Err() != nil {
		return s.finishRun(r0, agentruntime.RunCancelled, "dsh headless: run cancelled (context done before start)")
	}

	tmpDir, err := os.MkdirTemp("", "tally-dsh-run-")
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("dsh headless: 建临时目录失败: %v", err))
	}
	defer os.RemoveAll(tmpDir)

	// 期望配置 overlay(patch.go):零值 = 零文件零旗标,逐字节朴素调用。
	patchPath := ""
	if patch := s.runtime.patchOverlay(); patch != "" {
		patchPath = filepath.Join(tmpDir, patchFileName)
		if werr := os.WriteFile(patchPath, []byte(patch), 0o644); werr != nil {
			return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("dsh headless: 写 --patch overlay 失败: %v", werr))
		}
	}

	spec, err := s.buildSpec(instruction, patchPath)
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("dsh headless: 组装 CommandSpec 失败: %v", err))
	}

	// CommandContext 是 cmd.Cancel 的前提(Go os/exec 契约):ctx 取消时 exec
	// 调用 Cancel(整树终止)而不是默认单进程 Kill。
	cmd := exec.CommandContext(ctx, spec.Argv[0], spec.Argv[1:]...) //nolint:gosec // spec.Argv 由本包结构化组装,非用户 shell 字符串
	cmd.Dir = spec.Cwd
	cmd.Env = append(os.Environ(), envPairs(spec.Env)...)
	setProcAttr(cmd)
	// ctx 取消 = 正常取消终态:exec 以本函数替代默认的单进程 Kill,由
	// requestCancelRun 记账并整树终止(条件 4);返回 nil,让 Wait 收敛到
	// 进程真实结局,分类统一走 cancelRequested。
	cmd.Cancel = func() error { s.requestCancelRun(r0); return nil }
	// 兜底硬化:进程已退出但输出管道仍被(孤儿)子进程占用时,WaitDelay
	// 封顶 Wait 阻塞并强制关闭管道 —— 外部单进程 kill(非 Cancel)留下的
	// launcher 孤儿子进程不会把 Execute 拖到无界。stdout/stderr 都走 writer
	// 模式:全部管道等待发生在 cmd.Wait 内部,WaitDelay 才能生效
	//(StdoutPipe + 自行 ReadAll 会绕过它 —— 读端阻塞在 Wait 之外)。
	cmd.WaitDelay = s.runtime.cfg.waitDelay()
	stdout := &bytes.Buffer{}
	var stderr limitedBuffer
	stderr.limit = 64 * 1024
	cmd.Stdout = stdout
	cmd.Stderr = &stderr

	if serr := cmd.Start(); serr != nil {
		if ctx.Err() != nil {
			return s.finishRun(r0, agentruntime.RunCancelled, "dsh headless: run cancelled (context done before start)")
		}
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("dsh headless: 进程启动失败: %v", serr))
	}
	s.mu.Lock()
	r0.process = cmd.Process
	preCancelled := r0.cancelRequested
	s.mu.Unlock()
	if preCancelled {
		// Cancel 抢在 spawn 完成前到达:立即整树终止刚起来的进程。
		_ = killTree(cmd.Process)
	}

	waitErr := cmd.Wait()

	status, summary := classifyRun(r0, waitErr, stderr.String(), stdout.String())
	return s.finishRun(r0, status, summary)
}

// buildSpec 组装本轮的结构化命令描述(argv-only,全部取值来自 spike 观测的
// launcher 表面):
//
//		[bin] --profile headless [--patch <file>] <instruction>
//
//	  - workspace 不经旗标传递:headless 无 cwd 旗标,workspaceRoot = 进程 cwd
//	    (spike §2.2),由 CommandSpec.Cwd / cmd.Dir 钉住 —— 等价于
//	    runtimes/codex 的 -C 钉定语义。
//	  - 模型/提供者不经 argv(无模型旗标,spike §2.5):期望配置只经生成的
//	    最小 --patch YAML overlay(patch.go)。
//	  - instruction 恒为最后一个位置参数(headless 表面的 [task...])。
//
// 环境经 sessionSpawnEnv 强制 per-session DSH_HOME、DSH_TELEMETRY_MODE=
// DISABLED 与保守 DSH_PERMISSION_MODE(isolation.go)。
func (s *Session) buildSpec(instruction, patchPath string) (command.CommandSpec, error) {
	argv := make([]string, 0, 6)
	argv = append(argv, s.bin, "--profile", "headless")
	if patchPath != "" {
		argv = append(argv, patchFlag, patchPath)
	}
	argv = append(argv, instruction)
	spec := command.CommandSpec{
		Argv: argv,
		Cwd:  s.workspace,
		Env:  sessionSpawnEnv(s.runtime.cfg.Env, s.home, s.permission),
	}
	if err := spec.Validate(); err != nil {
		return command.CommandSpec{}, err
	}
	return spec, nil
}

// classifyRun 把进程结局分类为协议终态与 claim summary。观测契约(spike
// §2.3):exit 0 = turn completed 且 stdout = 最终回答;exit 非 0 = 未完成,
// stderr 携带结构化错误原文(如 keyless 的 MISSING_CREDENTIAL);SIGTERM
// 杀进程 = exit 143。优先级:取消(用户意图)> 进程异常退出(含 ErrWaitDelay
// 管道兜底与 stderr 原文)> 成功退出 + stdout 最终回答 > 成功退出但无回答
// (契约未兑现,FAIL-SAFE 判败,不猜测成功)。
func classifyRun(r0 *run, waitErr error, stderrTail, stdout string) (agentruntime.RunStatus, string) {
	if r0.cancelRequested {
		return agentruntime.RunCancelled, "dsh headless: run cancelled"
	}
	if waitErr != nil {
		if errors.Is(waitErr, exec.ErrWaitDelay) {
			// 进程已退出但输出管道未收敛:launcher 的子进程仍存活(外部单进程
			// kill 留下孤儿,或整树终止尚未收敛)。如实上报,不猜测结果。
			msg := "dsh headless: 进程退出后输出管道未收敛(WaitDelay 兜底;存在 launcher 孤儿子进程)"
			if tail := strings.TrimSpace(stderrTail); tail != "" {
				msg += "; stderr: " + truncate(tail, 500)
			}
			return agentruntime.RunFailed, msg
		}
		// 非零退出:进程自证未完成。stderr 的结构化错误原文(如
		// MISSING_CREDENTIAL)比退出码更具体,作为 summary 携带。
		msg := "dsh headless: 进程异常退出: " + waitErr.Error()
		if tail := strings.TrimSpace(stderrTail); tail != "" {
			msg += "; stderr: " + truncate(tail, 500)
		}
		return agentruntime.RunFailed, msg
	}
	answer := strings.TrimSpace(stdout)
	if answer == "" {
		return agentruntime.RunFailed, "dsh headless: 成功退出但 stdout 无最终回答(契约:exit 0 = completed 且 stdout = 最终回答,spike §2.3)"
	}
	return agentruntime.RunCompleted, answer
}

// finishRun 组装 AgentRunResult(claim)。Usage 恒为零值:headless 不自报
// 用量(R-3,「usage recorded when available」的 unavailable 分支)。
func (s *Session) finishRun(r0 *run, status agentruntime.RunStatus, summary string) agentruntime.AgentRunResult {
	if status != agentruntime.RunCompleted {
		summary = logpipe.RedactFine(summary)
	}
	return agentruntime.AgentRunResult{
		RunID:     r0.id,
		SessionID: s.sessionID(),
		Status:    status,
		Summary:   summary,
		// Usage:未观察,宁缺毋假(INV-AUTH-01)。
	}
}

// limitedBuffer 是有上界的 stderr 收集器(防失控输出撑爆内存)。
type limitedBuffer struct {
	buf       []byte
	limit     int
	truncated bool
}

// Write 实现 io.Writer。
func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - len(b.buf); room > 0 {
		if len(p) > room {
			b.buf = append(b.buf, p[:room]...)
			b.truncated = true
		} else {
			b.buf = append(b.buf, p...)
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

// String 返回已收集内容(截断时带标记)。
func (b *limitedBuffer) String() string {
	out := string(b.buf)
	if b.truncated {
		out += "\n...(truncated)"
	}
	return out
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

// truncate 按 rune 截断并加省略标记。
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
