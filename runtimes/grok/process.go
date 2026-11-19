package grok

// process.go 拥有 `grok --single` 进程生命周期:结构化 argv 组装、spawn、
// stdout 最终回答读取、取消/崩溃处理与终态分类。全部旗标与退出码语义来自
// 对官方 grok 1.0.87(commit 929e6dd187b1)的现场观测,未观测的旗标/表面
// 一律不使用、不解释(AGENTS.md §5 Runtime lock;观测事实见包注释)。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/command"
	"tally/internal/logpipe"
)

// noAutoUpdateFlag 是强制附加的自动更新关闭旗标(官方文档 headless-scripting
// 明示 scripted environments 应传;grok 1.0.87 现场观测接受,--help 未列出
// 属隐藏旗标)。自动更新会把新二进制写进 GROK_HOME/bin,候选执行不允许该出口。
const noAutoUpdateFlag = "--no-auto-update"

// singleFlag 是 headless 单回合旗标的附值前缀(观测:`-p/--single
// <PROMPT>`「Single-turn prompt. Prints the response to stdout and exits」;
// 附值形态 --single=<text> 对以 "-" 开头的指令文本安全 —— 裸 `-p "-..."`
// 被 clap 误解析为未知旗标 exit 2,附值形态现场观测正确接受为值)。
const singleFlag = "--single="

// execTurn 执行一个完整 turn:组装结构化 argv → spawn(cwd 钉 workspace)→
// 读取 stdout 最终回答 → wait → 分类终态 → 返回 claim。
func (s *Session) execTurn(ctx context.Context, r0 *run, instruction string) agentruntime.AgentRunResult {
	s.execMu.Lock()
	defer s.execMu.Unlock()

	// ctx 已取消的 turn 是正常取消终态,不以 RunFailed 伪装(不把取消伪造
	// 成候选失败;trust invariant)。
	if ctx.Err() != nil {
		return s.finishRun(r0, agentruntime.RunCancelled, "grok headless: run cancelled (context done before start)")
	}

	spec, err := s.buildSpec(instruction)
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("grok headless: 组装 CommandSpec 失败: %v", err))
	}

	// CommandContext 是 cmd.Cancel 的前提(Go os/exec 契约):ctx 取消时 exec
	// 调用 Cancel(整树终止)而不是默认单进程 Kill。
	cmd := exec.CommandContext(ctx, spec.Argv[0], spec.Argv[1:]...) //nolint:gosec // spec.Argv 由本包结构化组装,非用户 shell 字符串
	cmd.Dir = spec.Cwd
	// spawn 环境是白名单全量(不是「继承 + 追加」):exec.Cmd.Env 一旦非 nil
	// 就是完整环境列表,这里只给 sessionSpawnEnv 白名单构造的键 —— owner
	// 环境中的其他 provider 认证(DEEPSEEK_API_KEY 等)结构性不进入子进程
	// (isolation.go;比 runtimes/dsh 的继承语义更严格,是本适配器的显式要求)。
	cmd.Env = envPairs(spec.Env)
	setProcAttr(cmd)
	// ctx 取消 = 正常取消终态:exec 以本函数替代默认的单进程 Kill,由
	// requestCancelRun 记账并整树终止;返回 nil,让 Wait 收敛到进程真实结局,
	// 分类统一走 cancelRequested。
	cmd.Cancel = func() error { s.requestCancelRun(r0); return nil }
	// 兜底硬化:进程已退出但输出管道仍被(孤儿)子进程占用时,WaitDelay
	// 封顶 Wait 阻塞并强制关闭管道 —— 外部单进程 kill(非 Cancel)留下的
	// grok 孤儿子进程不会把 Execute 拖到无界。stdout/stderr 都走 writer
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
			return s.finishRun(r0, agentruntime.RunCancelled, "grok headless: run cancelled (context done before start)")
		}
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("grok headless: 进程启动失败: %v", serr))
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

// buildSpec 组装本轮的结构化命令描述(argv-only,全部取值来自 grok 1.0.87
// 观测的 CLI 表面):
//
//			[bin] --no-auto-update [-m <model>] [--permission-mode <mode>] --single=<instruction>
//
//	  - workspace 不经旗标传递(--cwd 效果未观测,不使用):workspaceRoot =
//	    进程 cwd(inspect 现场观测「CWD: <进程 cwd>」),由 CommandSpec.Cwd /
//	    cmd.Dir 钉住 —— 等价于 runtimes/codex 的 -C 钉定语义。
//	  - 期望模型/审批模式只经观测旗标透传(Config.Model / Config.PermissionMode,
//	    枚举校验在 Config.Validate);均为空则不传对应旗标(零 overlay,逐字节
//	    朴素调用)。
//	  - instruction 以 --single=<instruction> 附值形态作为最后一个 argv 元素
//	    (观测:对以 "-" 开头的文本安全,见 singleFlag)。
//
// 环境经 sessionSpawnEnv 强制 per-session GROK_HOME 与白名单拷贝
// (isolation.go):owner 环境中的应用层凭据不进入子进程。
func (s *Session) buildSpec(instruction string) (command.CommandSpec, error) {
	argv := make([]string, 0, 7)
	argv = append(argv, s.bin, noAutoUpdateFlag)
	if model := s.runtime.cfg.Model; strings.TrimSpace(model) != "" {
		argv = append(argv, "-m", model)
	}
	if pm := s.runtime.cfg.PermissionMode; pm != "" {
		argv = append(argv, "--permission-mode", pm)
	}
	argv = append(argv, singleFlag+instruction)
	spec := command.CommandSpec{
		Argv: argv,
		Cwd:  s.workspace,
		Env:  sessionSpawnEnv(s.runtime.cfg.Env, s.home),
	}
	if err := spec.Validate(); err != nil {
		return command.CommandSpec{}, err
	}
	return spec, nil
}

// classifyRun 把进程结局分类为协议终态与 claim summary。观测契约(grok
// 1.0.87 现场观测):-p/--single「Prints the response to stdout and exits」
// (--help 原文),--output-format 默认 plain;exit 0 = completed 且 stdout =
// 最终回答;keyless/坏 key = exit 1 + stdout 空 + stderr「Not signed in. ...」;
// 未知旗标 = exit 2。取消未观测(本机无凭据),按进程树终止实现。优先级:
// 取消(用户意图)> 进程异常退出(含 ErrWaitDelay 管道兜底与 stderr 原文)>
// 成功退出 + stdout 最终回答 > 成功退出但无回答(契约未兑现,FAIL-SAFE 判败,
// 不猜测成功)。
func classifyRun(r0 *run, waitErr error, stderrTail, stdout string) (agentruntime.RunStatus, string) {
	if r0.cancelRequested {
		return agentruntime.RunCancelled, "grok headless: run cancelled"
	}
	if waitErr != nil {
		if errors.Is(waitErr, exec.ErrWaitDelay) {
			// 进程已退出但输出管道未收敛:grok 的子进程仍存活(外部单进程
			// kill 留下孤儿,或整树终止尚未收敛)。如实上报,不猜测结果。
			msg := "grok headless: 进程退出后输出管道未收敛(WaitDelay 兜底;存在 grok 孤儿子进程)"
			if tail := strings.TrimSpace(stderrTail); tail != "" {
				msg += "; stderr: " + truncate(tail, 500)
			}
			return agentruntime.RunFailed, msg
		}
		// 非零退出:进程自证未完成。stderr 的错误原文(如「Not signed in. ...」
		// 认证失败)比退出码更具体,作为 summary 携带。
		msg := "grok headless: 进程异常退出: " + waitErr.Error()
		if tail := strings.TrimSpace(stderrTail); tail != "" {
			msg += "; stderr: " + truncate(tail, 500)
		}
		return agentruntime.RunFailed, msg
	}
	answer := strings.TrimSpace(stdout)
	if answer == "" {
		return agentruntime.RunFailed, "grok headless: 成功退出但 stdout 无最终回答(契约:exit 0 = completed 且 stdout = 最终回答,--help:Prints the response to stdout and exits)"
	}
	return agentruntime.RunCompleted, answer
}

// finishRun 组装 AgentRunResult(claim)。Usage 恒为零值:plain 表面不自报
// 用量;json/streaming-json 的 schema 未经真实回合观测,解析即发明
// (「usage recorded when available」的 unavailable 分支)。非完成态 summary
// 经 logpipe.RedactFine 做机密脱敏(失败诊断可能携带外部 CLI 的 stderr,
// 进入 AgentRunResult/log/UI 前必须脱敏)。
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
