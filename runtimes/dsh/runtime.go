// Package dsh 实现 DeepSeek Harness(DSH)的 agentruntime.Runtime 适配器
// (任务 P4-04),严格按 P4-03 协议 spike(docs/tasks/v1.4.1/spikes/
// P4-03_dsh_protocol_spike.md,下称「spike」)对钉住版本
// @deepseek-ai/dsh@0.1.5-rc.1 观测到的真实表面实现,不发明任何 CLI 旗标或
// 协议(Runtime lock,AGENTS.md §3):
//
//   - 集成表面(spike §2.1/§6 结论):`dsh --profile headless "<task>"` ——
//     一次 Execute = 一个一次性进程;stdout = 最终回答(纯文本,非 JSON 事件);
//     exit 0 = completed / 非 0 = 未完成;取消 = 进程终止(观测 exit 143);
//     无长驻守护进程。形态: Tally Go → runtimes/dsh → 外部 dsh 进程。
//   - workspace 钉定不经旗标(headless 无 cwd 旗标):workspaceRoot = 进程
//     cwd(spike §2.2 观测),由 spawn cwd = 可丢弃 worktree 实现 —— 与
//     CodexRuntime 的 -C 钉定同一语义(INV-EXEC-01)。
//   - 期望模型/提供者不经旗标(无模型旗标,spike §2.5):只经生成的最小
//     --patch YAML config overlay(patch.go),绝不经 argv。
//   - 认证 = DEEPSEEK_API_KEY 环境变量注入(spike §2.6);凭据绝不落盘。
//   - 产出的一切(AgentRunResult)都是 Agent Claim(INV-AUTH-01):runtime
//     自述的完成不是 Tally 事实,不能替代验证器证据。
//   - 内部 session.v3.jsonl.zstd 是私有格式,绝不作为契约解析(spike R-5)。
//
// 钉版身份闸(spike 条件 1):上游是承诺 breaking changes 的开发者预览版
// (`dsh run` 子命令已被移除是既有破坏实例),适配器的 argv/退出码契约只对
// 钉住版本 PinnedDSHVersion 有效。Start/Snapshot 现场执行 `dsh --version`,
// 观察失败或版本与钉住值不一致即拒绝开启会话(fail-closed)。这与
// runtimes/codex 的「探针失败 = 事实未观察,Start 继续」不同且为有意偏离:
// codex 是有探针记录背书的稳定 GA 表面,DSH 是钉版门控的预览表面。
// 适配器版本(DescriptorVersion)与 CLI 版本是两个事实,后者只来自现场
// 观察(AGENTS.md §13),绝不互相冒充。
//
// 隔离与出口硬化(spike 条件 2/3,DL-15 Tally-owned isolation):
//   - 每会话全新 DSH_HOME(临时目录,Close 时删除):默认 ~/.dsh 跨 run
//     持久化 sessions/storages/profiles,是 protocol-observable 的跨 run
//     状态泄漏面;无 auth 移交(~/.dsh 凭据文件格式未观测,不发明 —— 认证
//     走环境变量这一 spike 观测过的 supported 路径)。
//   - spawn 环境强制 DSH_TELEMETRY_MODE=DISABLED:默认 FEEDBACK_ONLY 把遥测
//     送到 DeepSeek OTLP 端点,候选执行不允许该出口。
//   - spawn 环境强制 DSH_PERMISSION_MODE=配置值(默认 workspace-write):
//     继承环境中的 danger-full-access 升级不得静默生效,只经显式 Config。
//   - 整树终止(spike 条件 4 / R-1):dsh launcher(Node)的子进程树在
//     Cancel/ctx 取消/Close 时被整树终止(Windows: taskkill /T /F;POSIX:
//     进程组 SIGKILL),exec.Cmd.WaitDelay 封顶孤儿进程占用输出管道的
//     Wait 阻塞。teardown 由进程级测试证明(runtime_test.go)。
//
// 能力声明(spike 条件 5 的诚实清单):headless 表面一次一个一次性进程,
// stdout 只有最终回答 —— 基础声明为空集(见 Capabilities);session.isolation
// 仅在 Config.SessionIsolationVerified 持有共享 conformance canary 证据时
// 声明(与 runtimes/codex 同一证据门纪律,声明真实性 PRD §35)。声明本身
// 仍是 claim,由 conformance(PRD §10.4)另行验证。
//
// 已知边界(如实声明,不是缺陷隐藏):
//   - Usage 恒为零值:headless 不自报用量(R-3,「usage recorded when
//     available」的 unavailable 分支)。
//   - 协议 SessionID 是适配器自造的会话编号(如 "dsh-session-000001"),
//     不是 dsh 协议层句柄 —— headless 不给出任何会话句柄,ObservedIdentity.
//     SessionID 恒为空(未观察 ≠ 编造)。
//   - ObservedIdentity.Model 恒为空:headless 不报告实际生效模型;期望配置
//     (Config.Model/Provider)绝不回填观察字段(AGENTS.md §13)。
package dsh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

// DescriptorName 是 DSHRuntime 的 runtime 标识。只用于展示/遥测/归因;
// 任何控制流都不得按它分支(PRD §10.3)。
const DescriptorName = "dsh"

// DescriptorVersion 是本适配器(不是 CLI)的版本。CLI 版本由 Snapshot/Start
// 经 `dsh --version` 现场发现后记录,绝不硬编码冒充观察(AGENTS.md §13)。
const DescriptorVersion = "1"

// PinnedDSHVersion 是本适配器契约锚定的上游精确版本(P4-03 spike 条件 1;
// spike 对该版本完成全部协议观测)。Config.PinnedVersion 为空时即用它。
const PinnedDSHVersion = "0.1.5-rc.1"

// DefaultPermissionMode 是未配置时的 DSH_PERMISSION_MODE 取值:允许写
// workspace(可丢弃 worktree)但不升级 danger-full-access —— 保守默认
// (spike §2.2 观测的默认档)。合法值见 permissionModes(spike 观测枚举)。
const DefaultPermissionMode = "workspace-write"

// permissionModes 是 DSH_PERMISSION_MODE 的全部已知取值(spike §2.2 观测:
// 默认 'workspace-write' 与升级档 'danger-full-access';其余取值未观测,
// 不发明)。
var permissionModes = map[string]bool{
	"workspace-write":    true,
	"danger-full-access": true,
}

// Capabilities 返回 DSH headless 的基础能力声明 —— 空集(spike 条件 5 的
// 诚实清单,保守映射)。headless 是一次性进程,stdout 只有最终回答:
//
//   - 无事件词汇表 → 不声明 event.streaming(R-3);
//   - 无 resume/fork argv → 不声明 session.resume / session.fork;
//   - 无结构化输出约定 → 不声明 output.structured;
//   - 无 CLI 模型旗标且实际模型不可观察 → 不声明 model.select(期望模型只经
//     生成的 --patch overlay;requested ≠ observed 永不成立,AGENTS.md §13);
//   - 取消 = 进程终止 = 核心 Cancel 语义 → 不声明 turn.interrupt / turn.steer;
//   - 无 headless 审批通道 → 不声明 tool.approval;无工具注入面 → 不声明
//     tools.custom;
//   - 上游 SAFETY.md 明示 sandbox/approval 不保证隔离、DSH 非唯一安全控制
//     → 不声明 sandbox.native(保守映射;Tally 侧可丢弃 worktree 才是边界);
//   - 内部 session 持久化不可经 headless 恢复/查询 → 不声明 session.persist;
//   - 无跨会话记忆声明 → 不声明 memory.declared。
//
// 声明由 conformance(PRD §10.4/§35)另行验证;声明本身仍是 claim。实例生效
// 声明见 capabilities(session.isolation 证据门)。
func Capabilities() agentruntime.CapabilitySet {
	return agentruntime.NewCapabilitySet()
}

// capabilities 返回本 Runtime 实例生效的能力声明:空基础集 + session.isolation
// (当且仅当 SessionIsolationVerified 置位 —— 持有「共享 conformance Session
// Isolation Canary 已对同一真实 dsh + 本 per-session home 隔离实现运行并通过」
// 的证据)。零值保持未声明:实现存在性不等于真实表面证据。
func (r *Runtime) capabilities() agentruntime.CapabilitySet {
	if !r.cfg.SessionIsolationVerified {
		return Capabilities()
	}
	return agentruntime.NewCapabilitySet(append(Capabilities().List(), agentruntime.CapSessionIsolation)...)
}

// Config 是 DSHRuntime 的构造配置。New 之后必须保持只读。
type Config struct {
	// BinPath 显式指定 dsh launcher 路径;空则按 ResolveBinaryPath 的观测顺序
	// 发现(env DSH_BIN_PATH → PATH 上的 dsh)。测试注入替身二进制用。
	BinPath string
	// PinnedVersion 是钉住的上游精确版本;空则 PinnedDSHVersion。Start/Snapshot
	// 现场观察 `--version` 与它不一致即 fail-closed(spike 条件 1)。刻意不提供
	// 「关掉钉版」的取值:开发者预览表面的契约没有版本锚即不成立。
	PinnedVersion string
	// Model 是期望模型(写入生成的 --patch overlay;如 "deepseek-chat")。
	// 空 = 零 overlay。它只是期望配置:与实际生效模型(headless 不可观察)
	// 是两个事实,绝不互相回填(AGENTS.md §13)。
	Model string
	// Provider 是期望提供者(写入生成的 --patch overlay;如 "deepseek-official")。
	// 空 = overlay 中省略 provider 行。
	Provider string
	// PermissionMode 是 DSH_PERMISSION_MODE 的取值;空则 DefaultPermissionMode。
	// 只接受 permissionModes 中的观测枚举值,其它值在 Start 处拒绝。spawn 一律
	// 强制该值,压倒继承环境中的同名变量(升级必须显式经 Config)。
	PermissionMode string
	// Env 追加到被 spawn 进程的环境变量(不是完整环境快照)。语义对齐
	// command.CommandSpec.Env;典型用途:DEEPSEEK_API_KEY(认证,spike §2.6)、
	// 测试注入。DSH_HOME / DSH_TELEMETRY_MODE / DSH_PERMISSION_MODE 三个键被
	// 隔离/硬化逻辑强制覆盖(isolation.go),Config.Env 中的同名值不生效。
	Env map[string]string
	// SessionIsolationVerified 是 session.isolation 的声明门(与 runtimes/codex
	// 同一证据门纪律):置 true 断言「共享 conformance Session Isolation Canary
	// 已对与生产同一解析出的真实 dsh + 本 per-session DSH_HOME 隔离实现运行
	// 并通过」。per-session home 隔离本身是无条件生产行为,开关只控制声明真伪。
	SessionIsolationVerified bool
	// WaitDelay 封顶「进程已退出但输出管道仍被(孤儿子)进程占用」时 Wait 的
	// 阻塞上限(exec.Cmd.WaitDelay;整树终止的兜底硬化)。零值取 defaultWaitDelay。
	WaitDelay time.Duration
	// Now 返回事件时间戳;nil 则用 time.Now。测试可注入固定时钟。
	Now func() time.Time
}

// defaultWaitDelay 是 WaitDelay 的默认兜底值:足以让整树终止收敛,又不让
// 孤儿管道把 Execute 拖到无界。
const defaultWaitDelay = 5 * time.Second

// waitDelay 返回生效的 WaitDelay。
func (c Config) waitDelay() time.Duration {
	if c.WaitDelay > 0 {
		return c.WaitDelay
	}
	return defaultWaitDelay
}

// RuntimeSnapshot 是 runtime 身份/版本/能力的一次可记录快照(对齐
// runtimes/codex 的 AC3 缝):供上层把「跑的是哪个 dsh、什么版本、声明了
// 什么能力」落成审计记录。
type RuntimeSnapshot struct {
	// Name 是 runtime 标识(DescriptorName)。
	Name string
	// AdapterVersion 是本适配器版本(DescriptorVersion)。
	AdapterVersion string
	// CLIVersion 是现场观察的 dsh launcher 版本(经 `--version`;已通过钉版
	// 闸,等于生效钉住值),非硬编码。
	CLIVersion string
	// BinaryPath 是现场解析出的 launcher 可执行文件路径。
	BinaryPath string
	// Capabilities 是声明能力(字典序)。声明由 conformance 另行验证。
	Capabilities []agentruntime.Capability
}

// Runtime 是 DSHRuntime 的 agentruntime.Runtime 实现。
type Runtime struct {
	cfg Config

	mu      sync.Mutex
	seq     int // 已开启会话计数(会话编号)
	binPath string
	cli     cliIdentity // `dsh --version` 现场观察缓存(每 Runtime 至多探一次)
}

// New 构造 DSHRuntime。零值 Config 合法(BinPath 走发现流程,版本走
// PinnedDSHVersion 钉版闸)。
func New(cfg Config) *Runtime {
	return &Runtime{cfg: cfg}
}

// Descriptor 返回自述。Version 是适配器版本;CLI 版本属现场事实,由
// Snapshot/Start 观察并记录,不进 Descriptor(避免 Descriptor 触发进程)。
// Capabilities 是本实例生效声明(capabilities)。
func (r *Runtime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{
		Name:         DescriptorName,
		Version:      DescriptorVersion,
		Capabilities: r.capabilities(),
	}
}

// Snapshot 确保二进制与钉版身份已观察,返回可记录的 RuntimeSnapshot。任一步
// 失败(二进制不可发现、--version 不可观察、版本不匹配钉住值)显式报错
// —— 钉版预览表面的身份不可观察即不可用(fail-closed,见包注释)。
func (r *Runtime) Snapshot(ctx context.Context) (RuntimeSnapshot, error) {
	bin, err := r.ensureBinary()
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	cli, err := r.ensureCLIIdentity(ctx, bin)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	if err := checkVersionPin(r.cfg.PinnedVersion, cli); err != nil {
		return RuntimeSnapshot{}, err
	}
	return RuntimeSnapshot{
		Name:           DescriptorName,
		AdapterVersion: DescriptorVersion,
		CLIVersion:     cli.Version,
		BinaryPath:     bin,
		Capabilities:   r.capabilities().Sorted(),
	}, nil
}

// Start 开启会话。统一走 agentruntime.CheckStart:结构校验 + 能力协商 ——
// 基础声明为空集,因此带 ResumeOf(需要 session.resume)或 ModelProfile
// (需要 model.select)的请求在协商处被 *CapabilityError 拒绝,不被隐式放行
// (spike 条件 5;未声明即拒绝是能力协商的契约方向)。
//
// Start 不 spawn headless 进程:一次 Execute = 一个进程;Start 只做三件有界
// 的事 —— 校验配置(permissionMode / patch overlay token)、解析二进制路径
// (缺失即刻报错,不拖到 Execute)、执行钉版身份闸(--version 现场观察 +
// 与钉住值比对,失败即拒绝;结果缓存于 Runtime 跨会话共享)。随后建立会话
// 状态与全新 per-session DSH_HOME(isolation.go)。
func (r *Runtime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	if err := agentruntime.CheckStart(r.Descriptor(), req); err != nil {
		return nil, err
	}
	permission, err := r.permissionMode()
	if err != nil {
		return nil, err
	}
	if _, err := renderPatchOverlay(r.cfg.Provider, r.cfg.Model); err != nil {
		return nil, err
	}
	bin, err := r.ensureBinary()
	if err != nil {
		return nil, err
	}
	cli, err := r.ensureCLIIdentity(ctx, bin)
	if err != nil {
		return nil, err
	}
	if err := checkVersionPin(r.cfg.PinnedVersion, cli); err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.seq++
	n := r.seq
	r.mu.Unlock()

	// per-session DSH_HOME 隔离(spike 条件 2):每个新会话一个全新临时 home;
	// Close 时删除。无 auth 移交:DSH 认证走 DEEPSEEK_API_KEY 环境变量
	//(spike §2.6 观测的 supported 路径;~/.dsh 下凭据文件格式未观测,不发明
	// 移交面 —— 该部署形态的缺失将以 MISSING_CREDENTIAL 的 RunFailed claim
	// 显形,不被隐藏)。创建失败 = Start 显式失败(不留无边界的会话)。
	home, err := newSessionHome()
	if err != nil {
		return nil, err
	}
	return &Session{
		runtime:    r,
		bin:        bin,
		permission: permission,
		index:      n,
		session:    agentruntime.SessionID(fmt.Sprintf("dsh-session-%06d", n)),
		workspace:  req.Workspace,
		home:       home,
		runs:       make(map[agentruntime.RunID]*run),
	}, nil
}

// ensureBinary 解析并缓存 dsh launcher 路径;缺失即刻报错(显式错误,不静默)。
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

// permissionMode 校验并返回 DSH_PERMISSION_MODE 取值:只接受 spike §2.2 的
// 观测枚举。
func (r *Runtime) permissionMode() (string, error) {
	mode := r.cfg.PermissionMode
	if mode == "" {
		return DefaultPermissionMode, nil
	}
	if !permissionModes[mode] {
		return "", fmt.Errorf("dsh: 非法 PermissionMode %q(已知值: workspace-write | danger-full-access,spike §2.2;其余取值未观测不发明)", mode)
	}
	return mode, nil
}

// envList 把 Config.Env 转成 KEY=VALUE 切片(键序确定;身份探针用)。
func (r *Runtime) envList() []string { return envPairs(r.cfg.Env) }

// now 返回事件时间戳来源。
func (r *Runtime) now() time.Time {
	if r.cfg.Now != nil {
		return r.cfg.Now()
	}
	return time.Now()
}

// patchOverlay 返回生成的期望配置 overlay 内容;空串 = 零 overlay(调用面
// 逐字节等于朴素 `--profile headless <task>`)。渲染是纯函数;构造期已在
// Start 校验过 token,这里的错误是防御分支。
func (r *Runtime) patchOverlay() string {
	content, err := renderPatchOverlay(r.cfg.Provider, r.cfg.Model)
	if err != nil {
		return ""
	}
	return content
}

// Session 是 DSHRuntime 的 agentruntime.Session 实现:一个 Tally 侧有界状态
// 对象。协议 SessionID 是适配器自造的会话编号(headless 不给出任何协议层
// 会话句柄,见包注释);同一会话的 Execute 串行化(execMu):headless 是
// 一次性进程,同一会话上的并发 turn 无观测依据,不做并发假设。
type Session struct {
	runtime    *Runtime
	bin        string
	permission string
	index      int
	// session 是适配器自造的协议会话编号(dsh-session-NNNNNN)。它标识 Tally
	// 协议层的会话边界,不是 dsh 的任何内部句柄;两者不可混淆。
	session agentruntime.SessionID
	// workspace 是 Start 收到的源码执行目录:spawn cwd 钉定它(headless 无
	// cwd 旗标,workspaceRoot = 进程 cwd,spike §2.2)。
	workspace string
	// home 是本会话专属的 per-session DSH_HOME(isolation.go):Start 时全新
	// 创建,Close 时删除;同一会话的全部 Execute 共享,跨会话永不共享。
	home string

	// execMu 串行化同会话的 headless 进程(一个会话同时最多一个进程)。
	execMu sync.Mutex

	mu     sync.Mutex
	runSeq int
	runs   map[agentruntime.RunID]*run
	closed bool
}

// run 是会话内一次运行(一个 headless 进程)的有界状态。
type run struct {
	id agentruntime.RunID
	// 以下字段只在持有会话锁时读写。status 创建即 RunRunning,终态由
	// Execute 收尾写入。
	status          agentruntime.RunStatus
	result          agentruntime.AgentRunResult
	hasResult       bool
	cancelRequested bool
	process         *os.Process
}

// Workspace 返回 Start 时收到的 workspace(供测试/conformance 断言
// 「receives cwd/workspace」,PRD §35;不属于核心协议)。
func (s *Session) Workspace() string { return s.workspace }

// Execute 同步执行一个 turn:spawn 一个 `dsh --profile headless` 进程,读取
// stdout 最终回答,返回 runtime 的 AgentRunResult(claim,INV-AUTH-01)。
// 取消(ctx 取消或 Cancel)是正常终态:返回 RunCancelled 的结果而非 error;
// 进程崩溃/非零退出/契约不兑现(成功退出但无回答)都归为 RunFailed claim,
// 绝不把取消或环境错误伪装成候选失败。
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
	r0 := &run{id: id, status: agentruntime.RunRunning}
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

// Cancel 请求终止指定 run:整树终止底层进程(条件 4)。run 不存在返回
// ErrRunNotFound;已到终态幂等成功;关闭后的会话返回 ErrSessionClosed。
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
		_ = killTree(p)
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
			SessionID: s.session,
			Status:    agentruntime.RunRunning,
		}, nil
	}
	return r0.result, nil
}

// Events 返回 ErrStreamingNotDeclared:DSH headless 未声明 event.streaming
// (诚实能力集,空基础集)。stdout 只有最终回答、stderr 的 reasoning 流不是
// 可规范化的会话事件词汇表,解释即发明(spike §2.3/R-3;CheckEvents 是协议
// 统一前置闸)。会话关闭同样以该未声明拒绝为先(对齐 runtimes/fake 顺序)。
func (s *Session) Events(ctx context.Context) (agentruntime.EventStream, error) {
	if err := agentruntime.CheckEvents(s.runtime.Descriptor()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, agentruntime.ErrSessionClosed
	}
	return nil, agentruntime.ErrStreamingNotDeclared
}

// Close 结束会话:整树终止全部 in-flight run(它们以 RunCancelled 收尾)并
// 拒绝后续操作,最后删除本会话的 per-session DSH_HOME(尽力而为:Close 不
// 等待 in-flight 进程收敛,删除失败不作为 Close 错误;进程终止而未 Close 的
// 会话,home 留在系统临时目录由 OS 清理兜底 —— 与 runtimes/codex 同一策略)。
// 幂等。
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
	s.mu.Unlock()
	for _, r0 := range running {
		s.requestCancelRun(r0)
	}
	if s.home != "" {
		_ = os.RemoveAll(s.home)
	}
	return nil
}

// sessionID 读协议会话编号。
func (s *Session) sessionID() agentruntime.SessionID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.session
}

// requestCancelRun 标记取消并整树终止底层进程(若已 spawn)。幂等。
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
		_ = killTree(p)
	}
}

// RunProcess 返回 in-flight run 的底层 launcher 进程(诊断/测试注入面,
// 例如 PRD §35 crash 场景的外部 kill 注入;返回的是 launcher 本身,不是其
// 子进程树)。run 不存在或已到终态返回 false。
func (s *Session) RunProcess(id agentruntime.RunID) (*os.Process, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r0, ok := s.runs[id]
	if !ok || r0.status != agentruntime.RunRunning || r0.process == nil {
		return nil, false
	}
	return r0.process, true
}

// InFlightRunID 返回当前唯一 in-flight run 的 ID(诊断/测试注入面;本适配器
// 未声明 event.streaming,订阅不到 run.started,crash 注入经此定位 run)。
// 只报告已观察到底层进程的 run(注入者需要可 kill 的进程句柄);无则返回 false。
func (s *Session) InFlightRunID() (agentruntime.RunID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, r0 := range s.runs {
		if r0.status == agentruntime.RunRunning && r0.process != nil {
			return id, true
		}
	}
	return "", false
}

// ---- 二进制发现 ----------------------------------------------------------------

// envDSHBinPath 是显式指定 dsh launcher 路径的环境变量名。
const envDSHBinPath = "DSH_BIN_PATH"

// ResolveBinaryPath 解析 dsh launcher 路径:env DSH_BIN_PATH → PATH 上的 dsh。
// 前一级解析出的路径必须真实存在(显式报错优于静默回落到错误的二进制)。
// 发现面刻意最小:spike 未观测到任何 config 文件发现机制,不发明。
func ResolveBinaryPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(envDSHBinPath)); p != "" {
		if err := requireFile(p, "env "+envDSHBinPath); err != nil {
			return "", err
		}
		return p, nil
	}
	if p, err := exec.LookPath("dsh"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("dsh: 未找到 dsh launcher(已尝试 env %s、PATH 上的 dsh;spike §2.1)", envDSHBinPath)
}

// requireFile 校验路径存在且是普通文件,否则给出指明来源的显式错误。
func requireFile(path, source string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("dsh: %s 解析到的路径不可用 %q: %w", source, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("dsh: %s 解析到的是目录而非可执行文件 %q", source, path)
	}
	return nil
}

// ensureCLIIdentity 现场执行 `<bin> --version`(spike §3 探针 1 观测形态:
// 裸版本行 "0.1.5-rc.1",exit 0)并缓存解码结果。探针没跑起来、非零退出或
// 输出不可解析都显式报错且不缓存。与 runtimes/codex 的「尽力而为探针」不同,
// 本适配器在 Start/Snapshot 把它作为钉版闸的硬前提(见包注释的有意偏离)。
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
		return cliIdentity{}, fmt.Errorf("dsh: 读取 --version 失败: %w", err)
	}
	if res.ExitCode != 0 {
		return cliIdentity{}, fmt.Errorf("dsh: --version 退出码 %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	c := parseCLIIdentity(res.Stdout)
	if !c.observed() {
		return cliIdentity{}, fmt.Errorf("dsh: --version 输出不可解析: %q", truncate(strings.TrimSpace(res.Stdout), 200))
	}
	r.mu.Lock()
	r.cli = c
	r.mu.Unlock()
	return c, nil
}

// checkVersionPin 把现场观察的 CLI 身份与钉住版本比对(spike 条件 1):
// 不一致即拒绝 —— 适配器的 argv/退出码契约只对钉住版本有效,开发者预览上游
// 承诺 breaking changes(spike §5;`dsh run` 已被移除是既有破坏实例)。
func checkVersionPin(pinned string, cli cliIdentity) error {
	if pinned == "" {
		pinned = PinnedDSHVersion
	}
	if cli.Version != pinned {
		return fmt.Errorf("dsh: 观察到的 launcher 版本 %q 与钉住版本 %q 不一致(开发者预览表面按精确版本钉住,spike 条件 1):拒绝开启会话", cli.Version, pinned)
	}
	return nil
}

// Validate 预检配置面(P4-06:Invalid configuration fails before task
// execution 的最早闸):权限枚举取值合法、期望配置 overlay 可渲染。不发起
// 进程、不做任何观察(钉版身份闸仍在 Start)。
func (c Config) Validate() error {
	if c.PermissionMode != "" && !permissionModes[c.PermissionMode] {
		return fmt.Errorf("dsh: 非法 PermissionMode %q(已知值: workspace-write | danger-full-access,spike §2.2;其余取值未观测不发明)", c.PermissionMode)
	}
	if _, err := renderPatchOverlay(c.Provider, c.Model); err != nil {
		return err
	}
	return nil
}
