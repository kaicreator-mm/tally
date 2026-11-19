package pi

// process.go 拥有 `pi -p` 一次性进程生命周期:结构化 argv 组装、spawn、
// stdout 输出读取与解析、取消/崩溃处理与终态分类。全部旗标与退出码语义
// 来自本机对 pi 0.85.1 的现场观测(见包注释),未观测的旗标/表面一律不
// 使用、不解释(Runtime lock;期望 ≠ 观察,AGENTS.md §13)。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/command"
	"tally/internal/logpipe"
)

// execTurn 执行一个完整 turn:组装结构化 argv → spawn(cwd 钉 workspace)→
// 读取 stdout 输出 → wait → 分类终态 → 返回 claim。
func (s *Session) execTurn(ctx context.Context, r0 *run, instruction string) agentruntime.AgentRunResult {
	s.execMu.Lock()
	defer s.execMu.Unlock()

	// ctx 已取消的 turn 是正常取消终态,不以 RunFailed 伪装(不把取消伪造
	// 成候选失败;trust invariant)。
	if ctx.Err() != nil {
		return s.finishRun(r0, agentruntime.RunCancelled, "pi print: run cancelled (context done before start)", runObservation{}, agentruntime.Usage{})
	}

	spec, err := s.buildSpec(instruction)
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("pi print: 组装 CommandSpec 失败: %v", err), runObservation{}, agentruntime.Usage{})
	}

	// CommandContext 是 cmd.Cancel 的前提(Go os/exec 契约):ctx 取消时 exec
	// 调用 Cancel(整树终止)而不是默认单进程 Kill。
	cmd := exec.CommandContext(ctx, spec.Argv[0], spec.Argv[1:]...) //nolint:gosec // spec.Argv 由本包结构化组装,非用户 shell 字符串
	cmd.Dir = spec.Cwd
	cmd.Env = spawnEnvList(s.runtime.cfg.Env, s.dirs)
	setProcAttr(cmd)
	// ctx 取消 = 正常取消终态:exec 以本函数替代默认的单进程 Kill,由
	// requestCancelRun 记账并整树终止;返回 nil,让 Wait 收敛到进程真实结局,
	// 分类统一走 cancelRequested。
	cmd.Cancel = func() error { s.requestCancelRun(r0); return nil }
	// 兜底硬化:进程已退出但输出管道仍被(孤儿子)进程占用时,WaitDelay
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
			return s.finishRun(r0, agentruntime.RunCancelled, "pi print: run cancelled (context done before start)", runObservation{}, agentruntime.Usage{})
		}
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("pi print: 进程启动失败: %v", serr), runObservation{}, agentruntime.Usage{})
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

	status, summary, obs, usage := classifyRun(r0, waitErr, s.runtime.outputMode(), stderr.String(), stdout.String())
	s.recordObserved(obs)
	return s.finishRun(r0, status, summary, obs, usage)
}

// buildSpec 组装本轮的结构化命令描述(argv-only,全部取值来自 0.85.1 观测):
//
//			[bin] -p --mode <mode> --no-session --session-dir <dir> --no-extensions
//			      --no-skills --no-prompt-templates --no-themes --no-context-files
//			      --no-approve [--provider <p>] [--model <m>] [--thinking <t>]
//			      -- <instruction>
//
//	  - workspace 不经旗标传递(pi 帮助文本无 cwd 旗标):workspace = 进程 cwd,
//	    由 CommandSpec.Cwd / cmd.Dir 钉住(观测:pi 的文件工具在 cwd 内真实
//	    落盘)。
//	  - --no-session 与 --session-dir 并用:前者保证不落会话文件(观测),后者
//	    是防御纵深(若语义变化,文件落在随会话删除的临时目录内;观测:不设
//	    --no-session 时 .jsonl 会话文件确落 --session-dir)。
//	  - 发现面全关与 --no-approve:见包注释隔离节(全部为帮助文本观测旗标,
//	    组合实测可行)。
//	  - Provider/Model/ThinkingLevel 只在显式配置时传旗标(期望配置,绝不冒充
//	    观察)。
//	  - instruction 恒为 "--" 后的单个位置参数(-- 结束选项解析,pi 帮助文本
//	    观测的传法)。
//
// 环境经 spawnEnvList 强制 per-session 目录重定向与凭据剥离面(isolation.go)。
func (s *Session) buildSpec(instruction string) (command.CommandSpec, error) {
	mode := s.runtime.outputMode()
	argv := make([]string, 0, 20)
	argv = append(argv, s.bin, "-p", "--mode", mode,
		"--no-session", "--session-dir", s.dirs.sessions,
		"--no-extensions", "--no-skills", "--no-prompt-templates",
		"--no-themes", "--no-context-files", "--no-approve")
	if p := s.runtime.cfg.Provider; p != "" {
		argv = append(argv, "--provider", p)
	}
	if m := s.runtime.cfg.Model; m != "" {
		argv = append(argv, "--model", m)
	}
	if t := s.runtime.cfg.ThinkingLevel; t != "" {
		argv = append(argv, "--thinking", t)
	}
	argv = append(argv, "--", instruction)
	spec := command.CommandSpec{
		Argv: argv,
		Cwd:  s.workspace,
	}
	if err := spec.Validate(); err != nil {
		return command.CommandSpec{}, err
	}
	return spec, nil
}

// runObservation 是一次 run 的可观察身份事实(json 模式从 NDJSON 事件流
// 观察;text 模式全零值 —— 未观察 ≠ 编造,AGENTS.md §13)。
type runObservation struct {
	// sessionHandle 是 pi 自报的会话 UUID(json 流 session 行;--no-session
	// 下为一次性句柄)。
	sessionHandle string
	// model/provider 是最后一条 assistant message_end 自报的实际生效值。
	model    string
	provider string
	// text 是该 assistant 消息的规范化文本。
	text string
	// hasAssistant 报告是否观察到带文本的 assistant message_end(成功分类
	// 的前提)。
	hasAssistant bool
}

// recordObserved 把本次 run 的观察快照记入会话(IdentityReporter 读的最新
// 观察;未观察到任何事实时不覆盖既有观察)。
func (s *Session) recordObserved(obs runObservation) {
	if !obs.hasAssistant && obs.sessionHandle == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if obs.sessionHandle != "" {
		s.observedSessionHandle = obs.sessionHandle
	}
	if obs.hasAssistant {
		s.observedModel = obs.model
		s.observedProvider = obs.provider
	}
}

// piStreamEvent 是 pi json 模式 NDJSON 事件行的解码投影(0.85.1 观测形态;
// 只声明本适配器消费的字段,其余字段原样留在流里不解释)。
type piStreamEvent struct {
	Type    string     `json:"type"`
	ID      string     `json:"id"`      // type=session 行:pi 自报会话 UUID
	Cwd     string     `json:"cwd"`     // type=session 行:进程 cwd(诊断)
	Message *piMessage `json:"message"` // message_start/message_end/turn_end 行
}

// piMessage 是事件行 message 字段的解码投影(观测:assistant message_end 携带
// role/content/provider/model/usage/stopReason)。
type piMessage struct {
	Role     string      `json:"role"`
	Content  []piContent `json:"content"`
	Provider string      `json:"provider"`
	Model    string      `json:"model"`
	Usage    *piUsage    `json:"usage"`
}

// piContent 是 message.content[] 的解码投影(观测形态:{"type":"text",
// "text":"..."};其他 content 类型存在与否不解释)。
type piContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// piUsage 是 message.usage 的解码投影(观测字段 input/output;cacheRead 等
// 其余维度在协议 Usage 形状之外,不发明映射)。
type piUsage struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
}

// parseJSONStream 解析 pi json 模式的 NDJSON stdout(0.85.1 观测):逐行解码,
// 捕获 session 行的会话 UUID 与最后一条带文本 assistant message_end 的文本/
// 用量/provider/model。行间其他事件类型与行内其余字段不解释;截断的尾行
// (取消/崩溃中途)解码失败被跳过 —— 流的完整性由 hasAssistant 判定,不靠
// 「行都合法」猜测成功。
func parseJSONStream(stdout string) (runObservation, agentruntime.Usage) {
	var obs runObservation
	var usage agentruntime.Usage
	dec := json.NewDecoder(strings.NewReader(stdout))
	for {
		var ev piStreamEvent
		if err := dec.Decode(&ev); err != nil {
			break // 流结束或尾行截断:按已观察内容判定,不猜测成功
		}
		switch ev.Type {
		case "session":
			obs.sessionHandle = ev.ID
		case "message_end":
			if ev.Message == nil || ev.Message.Role != "assistant" {
				continue
			}
			text := assistantText(ev.Message.Content)
			if text == "" {
				continue
			}
			obs.hasAssistant = true
			obs.model = ev.Message.Model
			obs.provider = ev.Message.Provider
			obs.text = text
			if ev.Message.Usage != nil {
				usage = agentruntime.Usage{InputTokens: ev.Message.Usage.Input, OutputTokens: ev.Message.Usage.Output}
			}
		}
	}
	return obs, usage
}

// assistantText 是 assistant 消息的规范化文本(多个 text content 块按出现序
// 以换行拼接;观测形态是单 text 块,多块拼接是适配器侧呈现选择)。
func assistantText(content []piContent) string {
	var texts []string
	for _, c := range content {
		if c.Type == "text" && c.Text != "" {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "\n")
}

// classifyRun 把进程结局分类为协议终态、claim summary、观察快照与用量。
// 观测契约(0.85.1):exit 0 = 进程自述成功(json:流内应有带文本的
// assistant message_end;text:stdout 是最终回答);exit 非 0 = 未完成
// (keyless 实测 exit 1 + stderr "No API key found for <provider>.";错误
// 模型实测 exit 1 + provider 400 文本);杀进程取消 = 部分流中途截断。
// 优先级:取消(用户意图)> 进程异常退出(含 ErrWaitDelay 管道兜底与
// stderr 原文)> 成功退出 + 可解析回答 > 成功退出但无回答(契约未兑现,
// FAIL-SAFE 判败,不猜测成功)。
func classifyRun(r0 *run, waitErr error, mode, stderrTail, stdout string) (agentruntime.RunStatus, string, runObservation, agentruntime.Usage) {
	if r0.cancelRequested {
		return agentruntime.RunCancelled, "pi print: run cancelled", runObservation{}, agentruntime.Usage{}
	}
	if waitErr != nil {
		if errors.Is(waitErr, exec.ErrWaitDelay) {
			// 进程已退出但输出管道未收敛:子进程仍存活(外部单进程 kill 留下
			// 孤儿,或整树终止尚未收敛)。如实上报,不猜测结果。
			msg := "pi print: 进程退出后输出管道未收敛(WaitDelay 兜底;存在孤儿子进程)"
			if tail := strings.TrimSpace(stderrTail); tail != "" {
				msg += "; stderr: " + truncate(tail, 500)
			}
			return agentruntime.RunFailed, msg, runObservation{}, agentruntime.Usage{}
		}
		// 非零退出:进程自证未完成。stderr 的错误原文(keyless 的
		// "No API key found..."、provider 400)比退出码更具体,作为 summary。
		msg := "pi print: 进程异常退出: " + waitErr.Error()
		if tail := strings.TrimSpace(stderrTail); tail != "" {
			msg += "; stderr: " + truncate(tail, 500)
		}
		return agentruntime.RunFailed, msg, runObservation{}, agentruntime.Usage{}
	}
	if mode == "text" {
		answer := strings.TrimSpace(stdout)
		if answer == "" {
			return agentruntime.RunFailed, "pi print: 成功退出但 stdout 无最终回答(观测契约:exit 0 = 成功且 stdout 为最终回答)", runObservation{}, agentruntime.Usage{}
		}
		return agentruntime.RunCompleted, answer, runObservation{}, agentruntime.Usage{}
	}
	obs, usage := parseJSONStream(stdout)
	if !obs.hasAssistant {
		return agentruntime.RunFailed, "pi print: 成功退出但 json 流无带文本的 assistant message_end(观测契约:成功回合以 assistant message_end 收束)", obs, agentruntime.Usage{}
	}
	return agentruntime.RunCompleted, obs.text, obs, usage
}

// finishRun 组装 AgentRunResult(claim)。非完成态 summary 先脱敏(logpipe
// 同一安全边界:外部进程的失败输出可能夹带密钥形态文本);Usage 由分类器
// 提供(json 模式取带文本 assistant message_end 的 usage;text 模式与全部
// 失败/取消态为零值 —— 未观察,宁缺毋假,INV-AUTH-01)。
func (s *Session) finishRun(r0 *run, status agentruntime.RunStatus, summary string, _ runObservation, usage agentruntime.Usage) agentruntime.AgentRunResult {
	if status != agentruntime.RunCompleted {
		summary = logpipe.RedactFine(summary)
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

// truncate 按 rune 截断并加省略标记。
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}
