package opencode

// process.go 拥有 `opencode run` 进程生命周期:结构化 argv 组装、spawn、
// stdout JSON 事件流读取、取消/崩溃处理与终态分类。全部旗标与退出码语义
// 来自对 opencode-ai@1.19.6 的现场观测,未观测的旗标/表面一律不使用、
// 不解释(AGENTS.md §3 Runtime lock;见包注释)。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/command"
	"tally/internal/logpipe"
)

// execTurn 执行一个完整 turn:组装结构化 argv → spawn(cwd 钉 workspace)→
// 读取 stdout JSON 事件流 → wait → 分类终态 → 返回 claim。
func (s *Session) execTurn(ctx context.Context, r0 *run, instruction string) agentruntime.AgentRunResult {
	s.execMu.Lock()
	defer s.execMu.Unlock()

	// ctx 已取消的 turn 是正常取消终态,不以 RunFailed 伪装(不把取消伪造
	// 成候选失败;trust invariant)。
	if ctx.Err() != nil {
		return s.finishRun(r0, agentruntime.RunCancelled, "opencode run: run cancelled (context done before start)", agentruntime.Usage{})
	}

	spec, err := s.buildSpec(instruction)
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("opencode run: 组装 CommandSpec 失败: %v", err), agentruntime.Usage{})
	}

	// CommandContext 是 cmd.Cancel 的前提(Go os/exec 契约):ctx 取消时 exec
	// 调用 Cancel(整树终止)而不是默认单进程 Kill。
	cmd := exec.CommandContext(ctx, spec.Argv[0], spec.Argv[1:]...) //nolint:gosec // spec.Argv 由本包结构化组装,非用户 shell 字符串
	cmd.Dir = spec.Cwd
	cmd.Env = append(os.Environ(), envPairs(spec.Env)...)
	setProcAttr(cmd)
	// ctx 取消 = 正常取消终态:exec 以本函数替代默认的单进程 Kill,由
	// requestCancelRun 记账并整树终止;返回 nil,让 Wait 收敛到进程真实
	// 结局,分类统一走 cancelRequested。
	cmd.Cancel = func() error { s.requestCancelRun(r0); return nil }
	// 兜底硬化:进程已退出但输出管道仍被(孤儿)子进程占用时,WaitDelay
	// 封顶 Wait 阻塞并强制关闭管道。stdout/stderr 都走 writer 模式:全部
	// 管道等待发生在 cmd.Wait 内部,WaitDelay 才能生效(StdoutPipe + 自行
	// ReadAll 会绕过它 —— 读端阻塞在 Wait 之外)。
	cmd.WaitDelay = s.runtime.cfg.waitDelay()
	stdout := &bytes.Buffer{}
	var stderr limitedBuffer
	stderr.limit = 64 * 1024
	cmd.Stdout = stdout
	cmd.Stderr = &stderr

	if serr := cmd.Start(); serr != nil {
		if ctx.Err() != nil {
			return s.finishRun(r0, agentruntime.RunCancelled, "opencode run: run cancelled (context done before start)", agentruntime.Usage{})
		}
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("opencode run: 进程启动失败: %v", serr), agentruntime.Usage{})
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

	status, summary, usage := classifyRun(r0, waitErr, stderr.String(), stdout.String())
	return s.finishRun(r0, status, summary, usage)
}

// buildSpec 组装本轮的结构化命令描述(argv-only,全部取值来自对 1.19.6 的
// 观测表面):
//
//			[bin] run --format json --pure [--model <provider/model>] [--agent <name>] <instruction>
//
//	  - `--format json`(观测枚举 "default"|"json"):stdout = 逐行 JSON 事件
//	    流,是 Summary/Usage 的结构化来源;解析失败降级原文(output.go)。
//	  - `--pure`(观测旗标,帮助:"run without external plugins"):关闭外部
//	    插件加载面 —— 隔离硬化的一部分(包注释)。
//	  - workspace 不经旗标传递:spawn cwd = workspace(效果级证据:真实 run
//	    在 cwd 内创建文件成功;`--dir` 旗标观测存在但语义冗余,不使用)。
//	  - 期望 Model/Agent 只经 `--model provider/model` / `--agent <name>`
//	    (两者均经真实 run 验证生效);空 = 旗标整个省略,绝不传空值。
//	  - instruction 恒为最后一个位置参数(message positional;单个 argv 元素,
//	    保留原文不切分)。
//
// 环境强制面(实测校准):XDG 四根 + TMP/TEMP 隔离键(isolation.go)之外,
// **PWD/OLDPWD 必须钉到 workspace** —— 真实 CLI 实测(live 探针,1.19.6,
// Windows):opencode 按 **PWD 环境变量**而非进程真实 cwd 解析项目/工作目录
// (进程 cwd 已钉 workspace、PWD 仍是父进程旧值时,agent 的相对路径写盘
// 精确落在 $PWD 而不是 workspace)。PWD 是「当前目录」的标准广告值,把它
// 与 cmd.Dir 对齐是让被钉 cwd 真实生效的必要条件,不是发明新配置面;
// OLDPWD 同理钉掉(实测继承链里它指向另一个 worktree,是跨工作区状态
// 泄漏面)。Config.Env 与隔离键先合并,PWD/OLDPWD 最后压倒(last-wins)。
func (s *Session) buildSpec(instruction string) (command.CommandSpec, error) {
	argv := make([]string, 0, 10)
	argv = append(argv, s.bin, "run", "--format", "json", "--pure")
	if m := s.runtime.cfg.Model; m != "" {
		argv = append(argv, "--model", m)
	}
	if a := s.runtime.cfg.Agent; a != "" {
		argv = append(argv, "--agent", a)
	}
	argv = append(argv, instruction)

	env := make(map[string]string, len(s.runtime.cfg.Env)+8)
	for k, v := range s.runtime.cfg.Env {
		env[k] = v
	}
	for k, v := range s.dirs.env() {
		env[k] = v
	}
	env["PWD"] = s.workspace
	env["OLDPWD"] = s.workspace
	spec := command.CommandSpec{
		Argv: argv,
		Cwd:  s.workspace,
		Env:  env,
	}
	if err := spec.Validate(); err != nil {
		return command.CommandSpec{}, err
	}
	return spec, nil
}

// classifyRun 把进程结局分类为协议终态、claim summary 与 usage 提取。
// 观测契约:exit 0 = turn completed 且 stdout 携带最终回答(--format json
// 下是事件流,其中最后一个 text 事件 = 最终回答);exit 非 0 = 未完成
// (stderr 携带结构化错误 JSON,如未知 model;观测 exit 1);整树终止由
// cancelRequested 记账(外部 kill 的子进程退出码不构成分类依据)。优先级:
// 取消(用户意图)> 进程异常退出(含 ErrWaitDelay 管道兜底与 stderr 原文)
// > 成功退出 + 可提取回答 > 成功退出但无任何回答(契约未兑现,FAIL-SAFE
// 判败,不猜测成功)。
//
// usage 只在成功分类且事件流携带 step_finish tokens 时非零;失败/取消/降级
// 路径恒为零值(宁缺毋假,INV-AUTH-01)。
func classifyRun(r0 *run, waitErr error, stderrTail, stdout string) (agentruntime.RunStatus, string, agentruntime.Usage) {
	if r0.cancelRequested {
		return agentruntime.RunCancelled, "opencode run: run cancelled", agentruntime.Usage{}
	}
	if waitErr != nil {
		if errors.Is(waitErr, exec.ErrWaitDelay) {
			// 进程已退出但输出管道未收敛:(潜在)子进程仍存活。如实上报,
			// 不猜测结果。
			msg := "opencode run: 进程退出后输出管道未收敛(WaitDelay 兜底;存在存活子进程)"
			if tail := strings.TrimSpace(stderrTail); tail != "" {
				msg += "; stderr: " + truncate(tail, 500)
			}
			return agentruntime.RunFailed, msg, agentruntime.Usage{}
		}
		// 非零退出:进程自证未完成。stderr 的结构化错误原文(观测形态是
		// JSON 错误对象)比退出码更具体,作为 summary 携带。
		msg := fmt.Sprintf("opencode run: 进程异常退出: %v", waitErr)
		if tail := strings.TrimSpace(stderrTail); tail != "" {
			msg += "; stderr: " + truncate(tail, 500)
		}
		return agentruntime.RunFailed, msg, agentruntime.Usage{}
	}

	parsed, _ := parseRunOutput(stdout)
	// 成功退出:优先取事件流提取的回答;事件流无可提取回答但 stdout 有
	// 内容 → 降级为原文(解析失败绝不 panic,也不丢回答);两者皆空 =
	// 契约未兑现,判败。
	if answer := strings.TrimSpace(parsed.Answer); answer != "" {
		return agentruntime.RunCompleted, answer, parsed.Usage
	}
	if raw := strings.TrimSpace(stdout); raw != "" {
		return agentruntime.RunCompleted, raw, agentruntime.Usage{}
	}
	return agentruntime.RunFailed, "opencode run: 成功退出但 stdout 无最终回答(契约:exit 0 = completed 且 stdout 携带回答,--format json 观测)", agentruntime.Usage{}
}

// finishRun 组装 AgentRunResult(claim)。usage 由 classifyRun 提取(仅
// 成功分类非零);失败/取消的 summary 经 logpipe.RedactFine 脱敏(stderr
// 原文可能携带 provider 侧敏感串)。
func (s *Session) finishRun(r0 *run, status agentruntime.RunStatus, summary string, usage agentruntime.Usage) agentruntime.AgentRunResult {
	if status != agentruntime.RunCompleted {
		summary = logpipe.RedactFine(summary)
		usage = agentruntime.Usage{}
	}
	return agentruntime.AgentRunResult{
		RunID:     r0.id,
		SessionID: s.sessionID(),
		Status:    status,
		Summary:   summary,
		Usage:     usage,
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
