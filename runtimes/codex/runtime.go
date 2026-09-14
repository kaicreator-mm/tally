// Package codex 实现 CodexRuntime —— PRD §10.4 规定的 P1 production runtime
// 适配器,严格按 P1-36 探针记录(docs/tasks/v1.3/runtime/CODEX_RUNTIME_PROBE.md,
// 下称「探针记录」)观测到的真实表面实现,不发明任何 CLI 旗标或协议
// (DECISION_LOCKS #13):
//
//   - 集成表面(探针记录 §8 结论):codex exec(非交互、非实验),一次
//     Execute = 一个 exec 进程;--json 在 stdout 输出 JSONL 事件;-C 钉住
//     workspace(可丢弃 Git worktree,INV-EXEC-01);-s/-a 由配置映射;
//     取消 = 进程终止;无长驻守护进程。
//   - 旗标全部来自探针记录 §2.1 的 --help 逐字输出:--json、-C、-s、-a、-m、
//     --output-schema、-o、--skip-git-repo-check;exec resume/fork 子命令来自
//     §2.1/§8。未在探针记录中出现的旗标一律不使用。
//   - 二进制发现(探针记录 §1):env CODEX_CLI_PATH → ~/.codex/config.toml 的
//     [mcp_servers.*.env] CODEX_CLI_PATH(App 更新时会重写该值)→ PATH 上的
//     codex。绝不硬编码内容哈希路径。
//   - 事件词汇表只建模探针记录 §3 实际观测的 JSONL 形态(thread.started /
//     turn.started / error / item.completed);未记录的形态容忍并忽略,不解释。
//   - 产出的一切(AgentRunResult / Event)都是 Agent Claim(INV-AUTH-01):
//     runtime 自述的完成/失败/用量都不是 Tally 事实,不能替代验证器证据。
//
// 能力声明(探针记录 §8 的诚实清单,PRD §35 capability declaration truthful):
// 声明 session.resume / session.fork / event.streaming / output.structured /
// sandbox.native / model.select / session.persist;不声明 turn.steer /
// turn.interrupt / tool.approval / tools.custom / memory.declared。
// session.isolation:P0-03 起 per-run home 隔离是生产行为(每会话全新
// CODEX_HOME + 仅 auth.json 移交,A0-02/DECISION_LOCKS #22),但声明仍以证据
// 为门:仅当 Config.SessionIsolationVerified 置位 —— 即共享 conformance
// Session Isolation Canary 已对同一真实 codex CLI + 本隔离实现通过(P0-04
// E2E 证据)—— 才声明;零值保持未声明。共享 CODEX_HOME 下 rollout 文件跨
// run 可检索曾是 protocol-observable 的隔离风险(探针记录 §6),由 per-run
// home 移除;声明与否不再取决于实现存在性,而取决于真实 CLI 上的 canary
// 证据。
//
// 已知边界(如实声明,不是缺陷隐藏):
//   - usage 不解析:探针记录未给出 exec 表面的用量事件形态,解析即发明;
//     Usage 恒为零值(「usage recorded when available」的 unavailable 分支)。
//   - exec 表面没有 headless 审批通道(-a 只有 on-request|never),故
//     tool.approval 不声明;-a 固定 never。
package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

// DescriptorName 是 CodexRuntime 的 runtime 标识。只用于展示/遥测/归因;
// 任何控制流都不得按它分支(PRD §10.3)。
const DescriptorName = "codex"

// DescriptorVersion 是本适配器(不是 CLI)的版本。CLI 版本由 Snapshot 经
// `codex --version` 现场发现后记录,绝不硬编码(App 更新会改变它)。
const DescriptorVersion = "1"

// 审批语义的文档化事实:现场复核(P1-37)实测 codex-cli 0.153.0 的
// `codex exec` 拒绝 -a/--ask-for-approval(与探针记录 §2.1 的抄录不符,以
// 实测表面为准,DECISION_LOCKS #13)。exec 是非交互一次性进程,无审批通道:
// 本适配器不传任何审批旗标,也因此不声明 tool.approval(PRD §35 truthful)。
// defaultSandbox 是未配置时的 -s 取值:允许写 workspace(可丢弃 worktree)
// 但不给 danger-full-access —— 保守默认(AGENTS.md §8)。
const defaultSandbox = "workspace-write"

// sandboxModes 是 -s/--sandbox 的全部合法取值(探针记录 §2.1 枚举逐字)。
var sandboxModes = map[string]bool{
	"read-only":          true,
	"workspace-write":    true,
	"danger-full-access": true,
}

// Capabilities 返回 CodexRuntime 的基础能力清单(探针记录 §8 诚实清单,不含
// session.isolation)。声明由 conformance(PRD §35)另行验证;声明本身仍是
// claim。实例生效声明见 capabilities。
func Capabilities() agentruntime.CapabilitySet {
	return agentruntime.NewCapabilitySet(
		agentruntime.CapSessionResume,
		agentruntime.CapSessionFork,
		agentruntime.CapEventStreaming,
		agentruntime.CapOutputStructured,
		agentruntime.CapSandboxNative,
		agentruntime.CapModelSelect,
		agentruntime.CapSessionPersist,
	)
}

// capabilities 返回本 Runtime 实例生效的能力声明:基础清单 + session.isolation
// (当且仅当 SessionIsolationVerified 置位 —— 持有「共享 Session Isolation
// Canary 已对同一真实 CLI + 本隔离实现通过」的证据,DECISION_LOCKS #22)。
// 零值保持未声明:实现存在性不等于真实表面证据(P0-03 AC3)。
func (r *Runtime) capabilities() agentruntime.CapabilitySet {
	if !r.cfg.SessionIsolationVerified {
		return Capabilities()
	}
	return agentruntime.NewCapabilitySet(append(Capabilities().List(), agentruntime.CapSessionIsolation)...)
}

// Config 是 CodexRuntime 的构造配置。New 之后必须保持只读。
type Config struct {
	// BinPath 显式指定 codex CLI 路径;空则按 ResolveBinaryPath 的观测顺序
	// 发现(env CODEX_CLI_PATH → config.toml → PATH)。测试注入替身二进制用。
	BinPath string
	// Sandbox 是 -s 的取值;空则 defaultSandbox。只接受 sandboxModes 中的
	// 探针记录枚举值,其它值在 Start 处拒绝。
	Sandbox string
	// Model 是未指定 ModelProfile 时的默认 -m 取值;空则不传 -m。
	Model string
	// Env 追加到被 spawn 进程的环境变量(不是完整环境快照)。语义对齐
	// command.CommandSpec.Env;其中 CODEX_HOME 被解释为 owner home(auth.json
	// 移交来源),实际 spawn 一律使用 per-session 隔离 home(isolatedSpawnEnv
	// 强制覆盖,继承环境中的同名变量同样被压倒)。
	Env map[string]string
	// SessionIsolationVerified 是 session.isolation 的声明门(DECISION_LOCKS
	// #22):置 true 断言「共享 conformance Session Isolation Canary 已对与
	// 生产同一解析出的真实 codex CLI + 本 per-run home 隔离实现运行并通过」
	// (证据由 P0-04 真实 E2E 产出)。只有持有该证据才可置位;零值保持
	// session.isolation 未声明。per-run home 隔离本身不受此开关影响 —— 它是
	// 无条件生产行为,开关只控制声明真伪。
	SessionIsolationVerified bool
	// Provider 是可选的自定义 provider 端点声明(ADR-0003,DECISION_LOCKS
	// #24):置位时每个会话的 per-run home 会在构造时被播种一份 Tally 生成的
	// 最小 config.toml(model_provider 端点 + 可选 model)。这是期望配置
	// 管道,不是状态移交;凭据材料仍然只经 #22 的 auth.json 移交或
	// Config.Env 透传,播种绝不写机密。零值(nil)= 零播种,行为与 P0-03
	// 完全一致。字段集合封闭:指令/会话状态结构性不可播种。
	Provider *ProviderEndpoint
	// Now 返回事件时间戳;nil 则用 time.Now。测试可注入固定时钟。
	Now func() time.Time
}

// RuntimeSnapshot 是 runtime 身份/版本/能力的一次可记录快照(P1-37 AC3):
// 供上层把「跑的是哪个 codex、什么版本、声明了什么能力」落成审计记录。
type RuntimeSnapshot struct {
	// Name 是 runtime 标识(DescriptorName)。
	Name string
	// AdapterVersion 是本适配器版本(DescriptorVersion)。
	AdapterVersion string
	// CLIVersion 是现场发现的 codex CLI 版本(如 "0.153.0"),非硬编码。
	CLIVersion string
	// BinaryPath 是现场解析出的 CLI 可执行文件路径。
	BinaryPath string
	// Capabilities 是声明能力(字典序)。声明由 conformance 另行验证。
	Capabilities []agentruntime.Capability
}

// Runtime 是 CodexRuntime 的 agentruntime.Runtime 实现。
type Runtime struct {
	cfg Config

	mu      sync.Mutex
	seq     int // 已开启会话计数(会话编号)
	binPath string
	cli     cliIdentity // `codex --version` 现场观察缓存(每 Runtime 至多探一次)
}

// New 构造 CodexRuntime。零值 Config 合法(BinPath 走发现流程)。
func New(cfg Config) *Runtime {
	return &Runtime{cfg: cfg}
}

// Descriptor 返回自述。Version 是适配器版本;CLI 版本属现场事实,由
// Snapshot 发现并记录,不进 Descriptor(避免 Descriptor 触发进程)。
// Capabilities 是本实例生效声明(capabilities):session.isolation 仅在
// SessionIsolationVerified 置位时出现。
func (r *Runtime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{
		Name:         DescriptorName,
		Version:      DescriptorVersion,
		Capabilities: r.capabilities(),
	}
}

// Snapshot 确保二进制与版本已发现,返回可记录的 RuntimeSnapshot(AC3)。
func (r *Runtime) Snapshot(ctx context.Context) (RuntimeSnapshot, error) {
	bin, err := r.ensureBinary()
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	ver, err := r.ensureCLIIdentity(ctx, bin)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	return RuntimeSnapshot{
		Name:           DescriptorName,
		AdapterVersion: DescriptorVersion,
		CLIVersion:     ver.Version,
		BinaryPath:     bin,
		Capabilities:   r.capabilities().Sorted(),
	}, nil
}

// Start 开启会话。统一走 agentruntime.CheckStart:结构校验 + 能力协商
// (ResumeOf ⇒ session.resume、ModelProfile ⇒ model.select,均已声明)。
// Start 不 spawn exec 进程:exec 表面一次 Execute = 一个进程;Start 只解析
// 二进制路径(缺失即刻报错,不拖到 Execute)并建立会话状态。唯一额外的进程
// 动作是有界的身份探针(P0-02):至多一次 `codex --version`(结果缓存于
// Runtime,跨会话共享),使会话经 IdentityReporter 自报实际 CLI 身份;
// 探针失败 = 该事实未观察,不是 Start 失败(见 observeCLIIdentity)。
func (r *Runtime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	if err := agentruntime.CheckStart(r.Descriptor(), req); err != nil {
		return nil, err
	}
	bin, err := r.ensureBinary()
	if err != nil {
		return nil, err
	}
	r.observeCLIIdentity(ctx, bin)
	sandbox, err := r.sandboxMode()
	if err != nil {
		return nil, err
	}
	model := req.ModelProfile
	if model == "" {
		model = r.cfg.Model
	}
	r.mu.Lock()
	r.seq++
	n := r.seq
	r.mu.Unlock()

	// per-run home 隔离(P0-03,A0-02/DECISION_LOCKS #22):每个新会话一个
	// 全新 CODEX_HOME,仅移交 auth.json;Config.Provider 置位时再播种生成的
	// 最小 config.toml(ADR-0003/#24)。创建或播种失败 = Start 显式失败
	//(不留无边界的会话)。
	home, err := newSeededSessionHome(r.cfg)
	if err != nil {
		return nil, err
	}

	// ResumeOf 直接是 codex thread id(本适配器的 SessionID = codex thread id,
	// 见 execTurn):续用会话的首个 Execute 走 `codex exec resume <id>`。
	return &Session{
		runtime:   r,
		bin:       bin,
		sandbox:   sandbox,
		model:     model,
		index:     n,
		workspace: req.Workspace,
		home:      home,
		threadID:  string(req.ResumeOf),
		runs:      make(map[agentruntime.RunID]*run),
		notify:    make(chan struct{}),
	}, nil
}

// Fork 从 parent 派生新会话(PRD §10.3 session.fork;协议核心无 fork 操作面,
// 这是适配器特有入口,对应探针记录 §2.1 的 `codex exec fork <SESSION_ID>`)。
// parent 尚无 thread(从未 Execute)时无可派生,子会话按新会话开始 —— fork
// 语义要求落在新 thread id 上,协议检查「子会话 ≠ 父会话」仍然成立。
func (r *Runtime) Fork(ctx context.Context, parent agentruntime.Session) (agentruntime.Session, error) {
	p, ok := parent.(*Session)
	if !ok {
		return nil, fmt.Errorf("codex: Fork 需要 codex 会话,得到 %T", parent)
	}
	p.mu.Lock()
	closed, workspace, model, threadID := p.closed, p.workspace, p.model, p.threadID
	p.mu.Unlock()
	if closed {
		return nil, agentruntime.ErrSessionClosed
	}
	r.mu.Lock()
	r.seq++
	n := r.seq
	r.mu.Unlock()
	// 派生会话同样获得自己的全新 per-run home(A0-02 边界 = 会话):fork 的
	// thread 连续性边界见 isolation.go 文件头;Config.Provider 置位时同样
	// 播种(ADR-0003:会话构造 = Start/Fork)。
	home, err := newSeededSessionHome(r.cfg)
	if err != nil {
		return nil, err
	}
	return &Session{
		runtime:   r,
		bin:       p.bin,
		sandbox:   p.sandbox,
		model:     model,
		index:     n,
		workspace: workspace,
		home:      home,
		forkFrom:  threadID,
		runs:      make(map[agentruntime.RunID]*run),
		notify:    make(chan struct{}),
	}, nil
}

// ensureBinary 解析并缓存 codex CLI 路径;缺失即刻报错(显式错误,不静默)。
func (r *Runtime) ensureBinary() (string, error) {
	r.mu.Lock()
	if r.binPath != "" {
		bin := r.binPath
		r.mu.Unlock()
		return bin, nil
	}
	r.mu.Unlock()
	if r.cfg.BinPath != "" {
		if err := requireFile(r.cfg.BinPath, "Config.BinPath"); err != nil {
			return "", err
		}
		r.mu.Lock()
		r.binPath = r.cfg.BinPath
		r.mu.Unlock()
		return r.cfg.BinPath, nil
	}
	bin, err := ResolveBinaryPath()
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	r.binPath = bin
	r.mu.Unlock()
	return bin, nil
}

// ensureCLIIdentity 现场执行 `<bin> --version`(探针记录 §1 观测形态:
// "codex-cli 0.153.0",exit 0)并缓存解码结果(CLI 自报标识 + 版本)。
// 探针没跑起来、非零退出或输出不可解析都显式报错且不缓存(版本未发现,
// 不是空版本成功);CLIVersion 只来自该实测,绝不取 DescriptorVersion
// (P0-02:适配器版本 ≠ CLI 版本)。
func (r *Runtime) ensureCLIIdentity(ctx context.Context, bin string) (cliIdentity, error) {
	r.mu.Lock()
	if r.cli.observed() {
		c := r.cli
		r.mu.Unlock()
		return c, nil
	}
	r.mu.Unlock()
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{Argv: []string{bin, "--version"}, Env: r.envList()})
	if err != nil {
		return cliIdentity{}, fmt.Errorf("codex: 读取 --version 失败: %w", err)
	}
	if res.ExitCode != 0 {
		return cliIdentity{}, fmt.Errorf("codex: --version 退出码 %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	c := parseCLIIdentity(res.Stdout)
	if c.Version == "" {
		return cliIdentity{}, fmt.Errorf("codex: --version 输出不可解析: %q", truncate(strings.TrimSpace(res.Stdout), 200))
	}
	r.mu.Lock()
	r.cli = c
	r.mu.Unlock()
	return c, nil
}

// sandboxMode 校验并返回 -s 取值:只接受探针记录 §2.1 的枚举。
func (r *Runtime) sandboxMode() (string, error) {
	mode := r.cfg.Sandbox
	if mode == "" {
		return defaultSandbox, nil
	}
	if !sandboxModes[mode] {
		return "", fmt.Errorf("codex: 非法 Sandbox %q(合法值: read-only | workspace-write | danger-full-access,探针记录 §2.1)", mode)
	}
	return mode, nil
}

// envList 把 Config.Env 转成 KEY=VALUE 切片(键序确定)。
func (r *Runtime) envList() []string { return envPairs(r.cfg.Env) }

// now 返回事件时间戳来源。
func (r *Runtime) now() time.Time {
	if r.cfg.Now != nil {
		return r.cfg.Now()
	}
	return time.Now()
}

// Session 是 CodexRuntime 的 agentruntime.Session 实现:一个 Tally 侧有界
// 状态对象。协议 SessionID = codex thread id(由 thread.started 事件给出,
// 探针记录 §3);thread id 已知前事件与 claim 的 SessionID 为空 —— 归因
// 恒一致。同一会话的 Execute 串行化(execMu):codex exec 是一次性进程,
// 同一 thread 上的并发 turn 未被探针记录覆盖,不做并发假设。
type Session struct {
	runtime   *Runtime
	bin       string
	sandbox   string
	model     string
	index     int
	workspace string
	// home 是本会话专属的 per-run CODEX_HOME(isolation.go):Start/Fork 时
	// 全新创建,Close 时删除;同一会话的全部 Execute 共享(会话内 thread
	// 连续性),不同会话永不共享(隔离边界)。
	home string
	// threadID 是当前 codex thread id(= 协议 SessionID)。
	threadID string
	// forkFrom 非空表示首个 Execute 走 `codex exec fork <forkFrom>`,用后即清。
	forkFrom string

	// execMu 串行化同会话的 codex turn(一个 thread 同时最多一个进程)。
	execMu sync.Mutex

	mu           sync.Mutex
	runSeq       int
	runs         map[agentruntime.RunID]*run
	events       []agentruntime.Event
	seq          int64 // 会话内事件序号,从 1 起单调递增
	notify       chan struct{}
	closed       bool
	outputSchema string
}

// Workspace 返回 Start 时收到的 workspace(供测试/conformance 断言
// 「receives cwd/workspace」,PRD §35;不属于核心协议)。
func (s *Session) Workspace() string { return s.workspace }

// SetOutputSchema 设置后续 Execute 的最终消息 JSON Schema(适配器特有入口;
// 需要声明 output.structured,已声明)。协议核心 ExecuteRequest 不携带 schema,
// 这里按探针记录 §2.1 的 --output-schema 落地。非合法 JSON 即拒绝。
func (s *Session) SetOutputSchema(schemaJSON string) error {
	trimmed := strings.TrimSpace(schemaJSON)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return fmt.Errorf("codex: output schema 必须是合法的非空 JSON")
	}
	s.mu.Lock()
	s.outputSchema = trimmed
	s.mu.Unlock()
	return nil
}

// outputSchemaValue 读当前 schema 快照。
func (s *Session) outputSchemaValue() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outputSchema
}

// Execute 同步执行一个 turn:spawn 一个 `codex exec` 进程,解析 --json JSONL,
// 返回 runtime 的 AgentRunResult(claim,INV-AUTH-01)。取消(ctx 取消或
// Cancel)是正常终态:返回 RunCancelled 的结果而非 error;进程崩溃/非零退出/
// error 事件/JSONL 解析失败都归为 RunFailed claim。
func (s *Session) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	if err := req.Validate(); err != nil {
		return agentruntime.AgentRunResult{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	s.runSeq++
	id := agentruntime.RunID(fmt.Sprintf("run-%06d", s.runSeq))
	r0 := &run{id: id, status: agentruntime.RunRunning, done: make(chan struct{})}
	s.runs[id] = r0
	s.mu.Unlock()

	res := s.execTurn(ctx, r0, req.Instruction)

	s.mu.Lock()
	r0.status = res.Status
	r0.result = res
	r0.hasResult = true
	s.mu.Unlock()
	return res, nil
}

// Cancel 请求终止指定 run:kill 底层进程。run 不存在返回 ErrRunNotFound;
// 已到终态幂等成功;关闭后的会话返回 ErrSessionClosed。
func (s *Session) Cancel(ctx context.Context, id agentruntime.RunID) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.ErrSessionClosed
	}
	r0, ok := s.runs[id]
	if !ok {
		s.mu.Unlock()
		return agentruntime.ErrRunNotFound
	}
	if r0.status != agentruntime.RunRunning {
		s.mu.Unlock()
		return nil
	}
	r0.cancelRequested = true
	p := r0.process
	s.mu.Unlock()
	if p != nil {
		_ = p.Kill()
	}
	return nil
}

// Result 返回指定 run 最近一次已知的 AgentRunResult(claim 快照)。
func (s *Session) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	r0, ok := s.runs[id]
	if !ok {
		return agentruntime.AgentRunResult{}, agentruntime.ErrRunNotFound
	}
	if !r0.hasResult {
		return agentruntime.AgentRunResult{
			RunID:     id,
			SessionID: agentruntime.SessionID(s.threadID),
			Status:    agentruntime.RunRunning,
		}, nil
	}
	return r0.result, nil
}

// Close 结束会话:kill 全部 in-flight run(它们以 RunCancelled 收尾)并拒绝
// 后续操作,最后删除本会话的 per-run home(含 auth.json 移交副本与会话
// rollout;尽力而为 —— Close 不等待 in-flight 进程收敛,删除失败不作为
// Close 错误,见 isolation.go 文件头)。幂等。
func (s *Session) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	var running []*run
	for _, r0 := range s.runs {
		if r0.status == agentruntime.RunRunning {
			running = append(running, r0)
		}
	}
	s.broadcastLocked()
	s.mu.Unlock()
	for _, r0 := range running {
		s.requestCancelRun(r0)
	}
	if s.home != "" {
		_ = os.RemoveAll(s.home)
	}
	return nil
}

// currentThreadID 读当前 thread id(= 协议 SessionID)。
func (s *Session) currentThreadID() agentruntime.SessionID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return agentruntime.SessionID(s.threadID)
}

// currentTurnMode 计算本轮的 exec 形态:fork(仅首回合)> resume > 新会话。
func (s *Session) currentTurnMode() turnMode {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.forkFrom != "" {
		return turnMode{sub: "fork", threadID: s.forkFrom}
	}
	if s.threadID != "" {
		return turnMode{sub: "resume", threadID: s.threadID}
	}
	return turnMode{}
}

// clearForkOnce 在首个 turn 结束后清掉 forkFrom(fork 只作用于首回合)。
func (s *Session) clearForkOnce() {
	s.mu.Lock()
	s.forkFrom = ""
	s.mu.Unlock()
}
