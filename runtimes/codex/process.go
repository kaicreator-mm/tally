package codex

// process.go 拥有 codex exec 进程生命周期:结构化 argv 组装、spawn、stdout
// JSONL 解析分类、取消/崩溃处理与二进制发现。全部旗标取值来自 P1-36 探针
// 记录,未记录的旗标/事件形态一律不使用、不解释(DECISION_LOCKS #13)。

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/command"
	"tally/internal/logpipe"
)

// turnMode 描述本轮 exec 形态:零值 = 新 thread;sub="resume"/"fork" 时
// threadID 是目标 codex thread id(探针记录 §2.1:`codex exec resume|fork <id>`)。
type turnMode struct {
	sub      string
	threadID string
}

// run 是会话内一次运行(codex exec 进程)的有界状态。
type run struct {
	id agentruntime.RunID
	// done 在 execTurn 收敛时关闭,释放 ctx watcher。
	done chan struct{}
	// 以下字段只在持有会话锁时读写。status 创建即 RunRunning,终态由
	// Execute 收尾写入。
	status          agentruntime.RunStatus
	result          agentruntime.AgentRunResult
	hasResult       bool
	cancelRequested bool
	startedEmitted  bool
	process         *os.Process
}

// requestCancelRun 标记取消并 kill 底层进程(若已 spawn)。幂等。
func (s *Session) requestCancelRun(r0 *run) {
	s.mu.Lock()
	if r0.status != agentruntime.RunRunning {
		s.mu.Unlock()
		return
	}
	r0.cancelRequested = true
	p := r0.process
	s.mu.Unlock()
	if p != nil {
		_ = p.Kill()
	}
}

// RunProcess 返回 in-flight run 的底层进程(诊断/测试注入面,例如 PRD §35
// crash 场景的外部 kill 注入)。run 不存在或已到终态返回 false。
func (s *Session) RunProcess(id agentruntime.RunID) (*os.Process, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r0, ok := s.runs[id]
	if !ok || r0.status != agentruntime.RunRunning || r0.process == nil {
		return nil, false
	}
	return r0.process, true
}

// execTurn 执行一个完整 turn:组装结构化 argv → spawn → 逐行解析 stdout
// JSONL → wait → 分类终态 → 投递规范化事件 → 返回 claim。
func (s *Session) execTurn(ctx context.Context, r0 *run, instruction string) agentruntime.AgentRunResult {
	s.execMu.Lock()
	defer s.execMu.Unlock()

	mode := s.currentTurnMode()
	defer s.clearForkOnce()

	tmpDir, err := os.MkdirTemp("", "tally-codex-run-")
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("codex exec: 建临时目录失败: %v", err), false)
	}
	defer os.RemoveAll(tmpDir)
	lastMsgPath := filepath.Join(tmpDir, "last-message.txt")
	schemaPath := ""
	if schema := s.outputSchemaValue(); schema != "" {
		schemaPath = filepath.Join(tmpDir, "output-schema.json")
		if werr := os.WriteFile(schemaPath, []byte(schema), 0o644); werr != nil {
			return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("codex exec: 写 output-schema 失败: %v", werr), false)
		}
	}

	spec, err := s.buildExecSpec(mode, instruction, schemaPath, lastMsgPath)
	if err != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("codex exec: 组装 CommandSpec 失败: %v", err), false)
	}

	cmd := exec.Command(spec.Argv[0], spec.Argv[1:]...) //nolint:gosec // spec.Argv 由本包结构化组装,非用户 shell 字符串
	cmd.Dir = spec.Cwd
	if pairs := envPairs(spec.Env); len(pairs) > 0 {
		cmd.Env = append(os.Environ(), pairs...)
	}
	stdout, perr := cmd.StdoutPipe()
	if perr != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("codex exec: 建 stdout 管道失败: %v", perr), false)
	}
	var stderr limitedBuffer
	stderr.limit = 64 * 1024
	cmd.Stderr = &stderr

	if serr := cmd.Start(); serr != nil {
		return s.finishRun(r0, agentruntime.RunFailed, fmt.Sprintf("codex exec: 进程启动失败: %v", serr), false)
	}
	s.mu.Lock()
	r0.process = cmd.Process
	preCancelled := r0.cancelRequested
	s.mu.Unlock()
	if preCancelled {
		// Cancel 抢在 spawn 完成前到达:立即终止刚起来的进程。
		_ = cmd.Process.Kill()
	}

	// ctx watcher:ctx 取消 = 正常取消终态(kill,由 execTurn 自己分类)。
	go func() {
		select {
		case <-ctx.Done():
			s.requestCancelRun(r0)
		case <-r0.done:
		}
	}()

	// stdout 逐行解析。只解释探针记录 §3 观测过的形态;未知 type 容忍并忽略。
	var parseErr error
	var lastErr string
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var raw rawEvent
		if jerr := json.Unmarshal(line, &raw); jerr != nil {
			// JSONL 解析错误是显式错误(探针记录 §8 前提),不是可忽略噪音。
			parseErr = fmt.Errorf("行 %q: %w", truncate(string(line), 200), jerr)
			break
		}
		switch raw.Type {
		case "thread.started":
			if raw.ThreadID != "" {
				s.mu.Lock()
				s.threadID = raw.ThreadID
				s.mu.Unlock()
			}
			s.emitRunStartedOnce(r0.id)
		case "turn.started":
			// run.started 已(或将)按 thread.started 对齐;turn.started 无独立投影。
		case "error":
			if raw.Message != "" {
				lastErr = raw.Message
			}
		case "item.completed":
			if raw.Item != nil && raw.Item.Type == "error" && raw.Item.Message != "" {
				lastErr = raw.Item.Message
			}
		default:
			// 未被探针记录记录的形态:忽略,不发明语义。
		}
	}
	if serr := sc.Err(); serr != nil && parseErr == nil {
		parseErr = fmt.Errorf("读取 stdout 失败: %w", serr)
	}
	waitErr := cmd.Wait()
	close(r0.done)

	status, summary := classifyTurn(r0, parseErr, lastErr, waitErr, stderr.String(), lastMsgPath)
	return s.finishRun(r0, status, summary, status == agentruntime.RunCompleted)
}

// buildExecSpec 组装本轮的结构化命令描述(argv-only,旗标全部经真实表面
// 验证):
//
//	[bin] exec --json --skip-git-repo-check -C <ws> -s <mode> [-m <model>]
//	     [--output-schema <file>] -o <file> [resume|fork <id>] <instruction>
//
// 旗标必须在 exec 子命令之前:codex-cli 0.153.0 实测(2026-09-09 现场探针,
// P0_CODEX_TRUST §6.2)resume/fork 子命令的 OPTIONS 不含 -C/-s,旧顺序
// `exec resume <id> --json ... -C ...` 被 clap 拒绝(exit 2,"error:
// unexpected argument '-C' found");用法面 `codex exec [OPTIONS] <COMMAND>
// [ARGS]` 只在 exec 层接受全部旗标,同日探针实测旗标前置的 resume/fork
// argv 均通过 clap 进入运行时层(报 "no rollout found",exit 1 —— 非 clap
// 的 exit 2)。真实 resume 路径由 P4-03 在带凭据表面上再验证。
//
// instruction 恒作为最后一个位置参数传入(exec 在无 prompt 且 stdin 接通时会
// 读 stdin,探针记录 §3;传参避免该行为)。已知局限(与 "-" 前缀同类):
// instruction 恰为 "resume"/"fork" 时会被 CLI 当子命令解析;Tally 上层组装的
// instruction 不会是这些裸词。
//
// 环境经 isolatedSpawnEnv 强制 CODEX_HOME=<本会话 per-run home>(P0-03 隔离;
// Config.Env 的 CODEX_HOME 只作 auth 移交来源,见 isolation.go)。
//
// 关于 -a/--ask-for-approval:P1-36 探针记录 §2.1 抄录了该旗标,但 P1-37 现场
// 复核(同一安装,codex-cli 0.153.0)实测 `codex exec` 拒绝它(clap:
// "unexpected argument '-a' found",长短形态皆然)。按 DECISION_LOCKS #13 以
// 实测表面为准:不传审批旗标,exec 的 headless 默认行为即无审批通道 —— 与
// 「不声明 tool.approval」一致。已知局限:instruction 以 "-" 开头时可能被 CLI
// 当旗标解析;探针记录未记录 "--" 分隔符,不发明;Tally 上层组装的
// instruction 不以 "-" 开头。
func (s *Session) buildExecSpec(mode turnMode, instruction, schemaPath, lastMsgPath string) (command.CommandSpec, error) {
	argv := make([]string, 0, 18)
	argv = append(argv, s.bin, "exec")
	argv = append(argv,
		"--json",
		"--skip-git-repo-check",
		"-C", s.workspace,
		"-s", s.sandbox,
	)
	if s.model != "" {
		argv = append(argv, "-m", s.model)
	}
	if schemaPath != "" {
		argv = append(argv, "--output-schema", schemaPath)
	}
	argv = append(argv, "-o", lastMsgPath)
	if mode.sub != "" {
		argv = append(argv, mode.sub, mode.threadID)
	}
	argv = append(argv, instruction)

	spec := command.CommandSpec{
		Argv: argv,
		Cwd:  s.workspace,
		Env:  isolatedSpawnEnv(s.runtime.cfg.Env, s.home),
	}
	if err := spec.Validate(); err != nil {
		return command.CommandSpec{}, err
	}
	return spec, nil
}

// classifyTurn 把进程结局分类为协议终态与 claim summary。优先级:
// 取消(用户意图)> JSONL 解析失败 > 退出码 > 完成产物(-o)> error 事件;
// 「进程异常退出/产物缺失」时以流内 error 原文作最具体的失败原因。
//
// 完成判定依据(P0_CODEX_TRUST §6.1 + 2026-09-09 对 codex-cli 0.153.0 的现场
// 探针):未知模型名(custom provider 模型)在每个 turn 起始发非致命
// item.type=="error" 元数据警告("Model metadata ... not found. Defaulting to
// fallback metadata ..."),随后 turn 照常进行 —— 因此 error 事件/item 不是权威
// 失败信号,「成功退出 + 产出 -o last-message 产物」才是权威完成证据;只有
// 在回合没有完成证据(非零退出或无产物)时,error 原文才如实失败:
//   - 401 实测(本机 0.153.0):exit 1 + turn.failed + 终态 error 事件 +
//     无产物 → 退出码分支,summary 携带 401 原文;
//   - 401 实测(P1-36 探针记录 §3,旧观察):exit 0 + error item + 无产物
//     → 产物缺失分支,summary 携带 401 原文。
//
// 换言之,事实完成的 turn 不再被流内 error 文本(元数据警告、瞬时
// Reconnecting 事件)误判为失败;而任何真实失败形态(无最终消息产出)仍
// FAIL-SAFE 判败。
func classifyTurn(r0 *run, parseErr error, lastErr string, waitErr error, stderrTail, lastMsgPath string) (agentruntime.RunStatus, string) {
	if r0.cancelRequested {
		return agentruntime.RunCancelled, "codex exec: run cancelled"
	}
	if parseErr != nil {
		return agentruntime.RunFailed, "codex exec: stdout JSONL 解析失败: " + parseErr.Error()
	}
	if waitErr != nil {
		// 非零退出:进程自证失败。流内 error 原文(如 401、schema 违约
		// item)比退出码更具体,优先作为 summary。
		if lastErr != "" {
			return agentruntime.RunFailed, "codex exec: " + lastErr
		}
		msg := "codex exec: 进程异常退出: " + waitErr.Error()
		if tail := strings.TrimSpace(stderrTail); tail != "" {
			msg += "; stderr: " + truncate(tail, 500)
		}
		return agentruntime.RunFailed, msg
	}
	data, rerr := os.ReadFile(lastMsgPath)
	if rerr != nil {
		// exit 0 但没有 -o 产物:turn 未产出最终消息,不是成功。若流内有
		// error 事件(401 旧实测形态:exit 0 + error item),原文是最具体
		// 的失败原因;否则显式报产物缺失,不猜测成功。
		if lastErr != "" {
			return agentruntime.RunFailed, "codex exec: " + lastErr
		}
		return agentruntime.RunFailed, "codex exec: 成功退出但未产出 last-message(-o)文件: " + rerr.Error()
	}
	// exit 0 + last-message 产物 = 权威完成证据:流内曾出现的 error 事件
	// /item(未知模型名的元数据警告、瞬时 Reconnecting 等)是非致命噪音,
	// 不以失败上报。
	return agentruntime.RunCompleted, strings.TrimSpace(string(data))
}

// finishRun 投递该 run 的收尾事件并组装 AgentRunResult(claim)。
// emitMessage 仅对成功且 summary 非空成立:失败文本属于 run.failed 的
// summary,不伪装成 agent.message。
func (s *Session) finishRun(r0 *run, status agentruntime.RunStatus, summary string, emitMessage bool) agentruntime.AgentRunResult {
	if status != agentruntime.RunCompleted {
		summary = logpipe.RedactFine(summary)
	}
	s.emitRunStartedOnce(r0.id)
	if emitMessage && summary != "" {
		s.emitEvent(agentruntime.EventAgentMessage, r0.id, summary, nil)
	}
	s.emitEvent(terminalKind(status), r0.id, "", nil)
	return agentruntime.AgentRunResult{
		RunID:     r0.id,
		SessionID: s.currentThreadID(),
		Status:    status,
		Summary:   summary,
	}
}

// limitedBuffer 是有上界的 stderr 收集器(防失控输出撑爆内存)。
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

// Write 实现 io.Writer。
func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.buf.Len(); room > 0 {
		if len(p) > room {
			b.buf.Write(p[:room])
			b.truncated = true
		} else {
			b.buf.Write(p)
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

// String 返回已收集内容(截断时带标记)。
func (b *limitedBuffer) String() string {
	out := b.buf.String()
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

// ---- 二进制发现(P1-36 探针记录 §1)-----------------------------------------

// envCodexCLIPath 是桌面 App 发布 CLI 路径的环境变量名(探针记录 §1)。
const envCodexCLIPath = "CODEX_CLI_PATH"

// codexCLIPathKey 是 config.toml [mcp_servers.*.env] 节内发布 CLI 路径的键。
const codexCLIPathKey = "CODEX_CLI_PATH"

// ResolveBinaryPath 按 P1-36 探针记录 §1 的观测顺序解析 codex CLI 路径,
// 绝不硬编码内容哈希路径(App 更新会重写它):
//
//  1. env CODEX_CLI_PATH(桌面 App 发布的当前路径);
//  2. CODEX_HOME(默认 ~/.codex)下 config.toml 的 [mcp_servers.*.env]
//     CODEX_CLI_PATH;
//  3. PATH 上的 codex。
//
// 前两级解析出的路径必须真实存在:App 更新后旧哈希目录会消失,显式报错优于
// 静默回落到错误的二进制。
func ResolveBinaryPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(envCodexCLIPath)); p != "" {
		if err := requireFile(p, "env "+envCodexCLIPath); err != nil {
			return "", err
		}
		return p, nil
	}
	if p, ok := codexPathFromConfigToml(configTomlPath()); ok {
		if err := requireFile(p, envCodexCLIPath+" in "+configTomlPath()); err != nil {
			return "", err
		}
		return p, nil
	}
	if p, err := exec.LookPath("codex"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("codex: 未找到 codex CLI(已尝试 env %s、%s 的 %s、PATH 上的 codex;见 P1-36 探针记录 §1)",
		envCodexCLIPath, configTomlPath(), codexCLIPathKey)
}

// requireFile 校验路径存在且是普通文件,否则给出指明来源的显式错误。
func requireFile(path, source string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("codex: %s 解析到的路径不可用 %q: %w", source, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("codex: %s 解析到的是目录而非可执行文件 %q", source, path)
	}
	return nil
}

// configTomlPath 返回 CODEX_HOME(默认 ~/.codex)下的 config.toml 路径。
func configTomlPath() string {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return filepath.Join(home, "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".codex", "config.toml")
	}
	return filepath.Join(home, ".codex", "config.toml")
}

// codexPathFromConfigToml 在 config.toml 中按探针记录 §1 的观测位置查找 CLI
// 路径:[mcp_servers.<name>.env] 节内的 CODEX_CLI_PATH 键。这是有界扫描,
// 不是通用 TOML 解析器:只跟踪 [section] 头,只在 mcp_servers.*.env 节内匹配
// 该键;值支持 TOML 基本字符串("...")与字面量字符串('...',Windows 反斜杠
// 路径的观测形态)。找不到返回 (\"\", false)。
func codexPathFromConfigToml(path string) (string, bool) {
	f, err := os.Open(path) //nolint:gosec // 路径由 configTomlPath/调用方给定
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()

	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if !strings.HasPrefix(section, "mcp_servers.") || !strings.HasSuffix(section, ".env") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		if strings.TrimSpace(line[:eq]) != codexCLIPathKey {
			continue
		}
		if v, ok := tomlStringValue(strings.TrimSpace(line[eq+1:])); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// tomlStringValue 提取 TOML 标量字符串值:字面量('...',无转义)或基本
// ("...",有转义)。允许值后跟注释/空白。非字符串返回 false。
func tomlStringValue(v string) (string, bool) {
	if v == "" {
		return "", false
	}
	switch v[0] {
	case '\'':
		if end := strings.IndexByte(v[1:], '\''); end >= 0 {
			return v[1 : 1+end], true
		}
	case '"':
		if end := indexUnescapedDoubleQuote(v[1:]); end >= 0 {
			if u, uerr := strconv.Unquote(v[:1+end+2]); uerr == nil {
				return u, true
			}
		}
	}
	return "", false
}

// indexUnescapedDoubleQuote 返回 s 中第一个未转义双引号的下标。
func indexUnescapedDoubleQuote(s string) int {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++ // 跳过转义字符
		case '"':
			return i
		}
	}
	return -1
}

// parseCLIIdentity 从 `codex --version` 输出解码 CLI 自报身份(观测形态:
// "codex-cli 0.153.0")。仅该实测形态下自报标识可解析;版本不可解析时取
// 整行原文(不发明格式),标识为空 —— 两个事实同源同刻,谁没观察到谁为空。
func parseCLIIdentity(out string) cliIdentity {
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) >= 2 && fields[0] == cliSelfName {
		return cliIdentity{Name: fields[0], Version: fields[1]}
	}
	return cliIdentity{Version: strings.TrimSpace(out)}
}
