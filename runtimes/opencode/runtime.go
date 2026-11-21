// Package opencode 实现 OpenCode CLI(opencode-ai npm 包)的
// agentruntime.Runtime 适配器(V2_1_PROVIDER_CANDIDATE),严格按对钉住版本
// opencode-ai@1.19.6 的现场观测实现,不发明任何 CLI 旗标或输出契约
// (Runtime lock,AGENTS.md §3;「观测自 1.19.6,未观测的不发明」):
//
//   - 集成表面:`opencode run [flags] <message>` —— 一次 Execute = 一个一次性
//     进程;默认(default 格式)stdout = 最终回答纯文本;`--format json`
//     时 stdout = 逐行 JSON 事件流(见 output.go 的观测结构)。exit 0 =
//     completed / 非 0 = 未完成(观测:未知 model → exit 1 + stderr 结构化
//     错误 JSON、stdout 空)。形态: Tally Go → runtimes/opencode → 外部
//     opencode 进程。
//
// standalone 结论(任务前置问题):1.19.6 的 run 表面**没有 `--standalone`
// 旗标**(run --help 全文无此项;传入未知旗标时 yargs 打印 run 帮助拒绝执行)。
// 观测到的等价隔离事实:
//
//   - `opencode run`(不带 --attach)每次启动**进程内自带的临时本地
//     server**(`--port` 帮助原文:"port for the local server (defaults to
//     random port if no value provided)"),不连接任何共享后台服务;
//     运行期进程树内只观察到单个 opencode.exe(内部 server 同进程),
//     运行结束后无任何 opencode 进程残留(tasklist 取证)。
//   - `--attach <url>`(连接已运行 server)是观测存在的共享面,本适配器
//     **绝不传入**;每个 Execute 都是全新独立进程 + 全新会话。
//   - 会话隔离:不带 -c/--session 时每次 run 创建**新 session**(隔离数据
//     目录下 `opencode session list` 观测到与 run 次数一致的独立 ses_* 句柄)。
//
// 隔离与出口硬化(DL-15 Tally-owned isolation):
//   - 每会话全新临时目录树经 **XDG_CONFIG_HOME / XDG_DATA_HOME /
//     XDG_CACHE_HOME / XDG_STATE_HOME** 四个环境变量重定向 opencode 的
//     config/data/cache/state(opencode debug paths 观测:四个路径全部跟随
//     对应 XDG 根、追加 "opencode" 子目录),owner 的 ~/.config/opencode、
//     ~/.local/share/opencode(auth.json/log/sessions)等零读取零写入
//     (实测:owner log 目录在隔离 run 前后 mtime 无变化)。Close 删除。
//     另注入通用 TMP/TEMP 指向会话临时目录(平台级 temp 重定向,非 opencode
//     私有契约)。**PWD/OLDPWD 强制钉到 workspace**(live 实测,1.19.6,
//     Windows):opencode 按 PWD 环境变量而非进程真实 cwd 解析项目/工作
//     目录 —— 进程 cwd 已钉 workspace 而 PWD 仍是父进程旧值时,agent 的
//     相对路径写盘落在 $PWD 而不是 workspace(钉版身份闸之外的第二条
//     fail-closed 实测校正)。
//   - spawn 一律带观测旗标 `--pure`(帮助:"run without external plugins"),
//     关闭外部插件加载面 —— 与遥测/权限类硬化同理,候选执行不允许
//     workspace 级插件注入;per-session 空 config 已隔离 owner 配置,
//     --pure 收窄的是 workspace 级 plugin 面。
//   - 期望配置只经观测旗标 `--model provider/model` 与 `--agent <name>`
//     (两者均经真实 run 验证生效),绝不经 argv 之外的面;观测存在的
//     `OPENCODE_CONFIG` 环境变量(config 注入面)刻意不使用 —— 期望配置
//     管道单一化,不引入第二配置面。
//   - 认证:opencode 经环境变量识别 provider API key(providers list 观测:
//     "Environment: DeepSeek DEEPSEEK_API_KEY");Config.Env 注入
//     DEEPSEEK_API_KEY 即用,凭据绝不落盘。**无凭据不必然失败**:观测到
//     无任何 key 时 CLI 自动落到免凭据默认模型(big-pickle)并成功完成
//     —— 适配器不假设 MISSING_CREDENTIAL 语义,凭据缺失的后果由真实退出
//     码/输出显形。
//
// 钉版身份闸:上游迭代快(run 表面在版本间增删旗标,如 --standalone 在
// 部分历史版本存在),适配器的 argv/输出契约只对钉住版本
// PinnedOpenCodeVersion 有效。Start/Snapshot 现场执行 `opencode --version`
// (观测形态:裸版本行,exit 0),观察失败或版本与钉住值不一致即拒绝
// 开启会话(fail-closed)。适配器版本(DescriptorVersion)与 CLI 版本是
// 两个事实,后者只来自现场观察(AGENTS.md §13),绝不互相冒充。
//
// 整树终止:取消/ctx 超时/Close 时整树终止底层进程(Windows: taskkill
// /T /F;POSIX: 进程组 SIGKILL),exec.Cmd.WaitDelay 封顶孤儿进程占用
// 输出管道的 Wait 阻塞。观测的 1.19.6 进程形态是单体可执行(Bun 编译,
// 运行期单进程),整树终止对单体进程无害、对其派生子进程(未观测,但
// --port 的本地 server 形态保留可能性)是正确防护。
//
// 能力声明:诚实空集起步(见 Capabilities)。stdout 的 JSON 事件流是
// 一次性进程的输出,不是可独立订阅的会话事件流 → 不声明
// event.streaming;每 run 新会话、无 resume/fork argv → 不声明
// session.resume/fork;session.isolation 仅在 Config.SessionIsolationVerified
// 持有共享 conformance canary 证据时声明(与 runtimes/codex、runtimes/dsh
// 同一证据门纪律);期望 model/agent 经旗标传入但实际生效值不可逐 run
// 观察 → 不声明 model.select(声明即要求可验证;requested ≠ observed
// 永不成立,AGENTS.md §13)。声明由 conformance(PRD §10.4)另行验证。
//
// 已知边界(如实声明,不是缺陷隐藏):
//   - Usage:--format json 的 step_finish 事件携带 tokens 观测结构时按
//     run 内逐步求和填充(claim);无 JSON 事件/解析失败时恒为零值
//     (宁缺毋假,INV-AUTH-01)。
//   - 协议 SessionID 是适配器自造的会话编号("opencode-session-NNNNNN")。
//     JSON 事件流中可观察到真实的 ses_* 会话句柄,本适配器按任务边界
//     刻意不将其上报进 ObservedIdentity(保守映射:句柄属每 run 事实,
//     非会话建立时事实);ObservedIdentity 只携带现场 `--version` 的
//     版本观测。
//   - ObservedIdentity.Model 恒为空:期望配置(Config.Model/Agent)与
//     实际生效模型是两个事实,绝不回填观察字段(AGENTS.md §13)。
package opencode

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
)

// DescriptorName 是 OpenCodeRuntime 的 runtime 标识。只用于展示/遥测/归因;
// 任何控制流都不得按它分支(PRD §10.3)。
const DescriptorName = "opencode"

// DescriptorVersion 是本适配器(不是 CLI)的版本。CLI 版本由 Snapshot/Start
// 经 `opencode --version` 现场发现后记录,绝不硬编码冒充观察(AGENTS.md §13)。
const DescriptorVersion = "1"

// PinnedOpenCodeVersion 是本适配器契约锚定的上游精确版本(对它完成全部
// run 表面观测)。Config.PinnedVersion 为空时即用它。
const PinnedOpenCodeVersion = "1.19.6"

// Capabilities 返回 OpenCode run 表面的基础能力声明 —— 空集(诚实清单,
// 保守映射;逐项理由见包注释)。声明由 conformance(PRD §10.4/§35)另行
// 验证;实例生效声明见 capabilities(session.isolation 证据门)。
func Capabilities() agentruntime.CapabilitySet {
	return agentruntime.NewCapabilitySet()
}

// capabilities 返回本 Runtime 实例生效的能力声明:空基础集 + session.isolation
// (当且仅当 SessionIsolationVerified 置位 —— 持有「共享 conformance Session
// Isolation Canary 已对同一真实 opencode + 本 per-session 隔离实现运行并通过」
// 的证据)。零值保持未声明:实现存在性不等于真实表面证据。
func (r *Runtime) capabilities() agentruntime.CapabilitySet {
	if !r.cfg.SessionIsolationVerified {
		return Capabilities()
	}
	return agentruntime.NewCapabilitySet(append(Capabilities().List(), agentruntime.CapSessionIsolation)...)
}

// Config 是 OpenCodeRuntime 的构造配置。New 之后必须保持只读。
type Config struct {
	// BinPath 显式指定 opencode 可执行文件路径;必填,空值在 Validate/Start
	// 处显式拒绝(canary 候选不做隐式发现,对齐 cmd/worker 对 dsh 的显式
	// 路径纪律)。测试注入替身二进制用。
	BinPath string
	// PinnedVersion 是钉住的上游精确版本;空则 PinnedOpenCodeVersion。
	// Start/Snapshot 现场观察 `--version` 与它不一致即 fail-closed。刻意
	// 不提供「关掉钉版」的取值:本适配器契约没有版本锚即不成立。
	PinnedVersion string
	// Model 是期望模型,格式 "provider/model"(run --model 旗标的观测格式,
	// 如 "deepseek/deepseek-v4-pro")。空 = 不传 --model(CLI 落默认模型)。
	// 它只是期望配置:与实际生效模型是两个事实,绝不互相回填(AGENTS.md §13)。
	Model string
	// Agent 是期望 agent 名(run --agent 旗标;如 "build")。空 = 不传
	// --agent(CLI 落默认 agent,观测默认为 "build")。同为期望配置。
	Agent string
	// Env 追加到被 spawn 进程的环境变量(不是完整环境快照)。典型用途:
	// DEEPSEEK_API_KEY(认证,经环境变量识别,providers list 观测)、测试
	// 注入。隔离键(XDG_* / TMP / TEMP)被隔离逻辑强制覆盖(isolation.go),
	// Config.Env 中的同名值不生效。
	Env map[string]string
	// SessionIsolationVerified 是 session.isolation 的声明门(与
	// runtimes/codex、runtimes/dsh 同一证据门纪律):置 true 断言「共享
	// conformance Session Isolation Canary 已对与生产同一解析出的真实
	// opencode + 本 per-session 隔离实现运行并通过」。per-session 隔离本身
	// 是无条件生产行为,开关只控制声明真伪。
	SessionIsolationVerified bool
	// WaitDelay 封顶「进程已退出但输出管道仍被(孤儿子)进程占用」时 Wait
	// 的阻塞上限(exec.Cmd.WaitDelay;整树终止的兜底硬化)。零值取
	// defaultWaitDelay。
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

// Validate 预检配置面(Invalid configuration fails before task execution
// 的最早闸):BinPath 必填;期望 Model/Agent 的旗标取值可安全裸传。不发起
// 进程、不做任何观察(钉版身份闸仍在 Start/Snapshot)。
func (c Config) Validate() error {
	if c.BinPath == "" {
		return fmt.Errorf("opencode: Config.BinPath 必填(显式路径,canary 候选不做隐式发现)")
	}
	if err := validateModelFlag(c.Model); err != nil {
		return err
	}
	return validateAgentFlag(c.Agent)
}

// validateModelFlag 校验期望 Model 的 `--model` 旗标取值:观测格式
// "provider/model"(恰好一个 '/',两侧均非空)。取值字符集限定为可安全
// 裸传 argv 的受限集合(不做引号转义器,直接拒绝可疑取值)。
func validateModelFlag(model string) error {
	if model == "" {
		return nil
	}
	provider, spec, ok := strings.Cut(model, "/")
	if !ok || provider == "" || spec == "" || strings.Contains(spec, "/") {
		return fmt.Errorf("opencode: Config.Model %q 不符合观测格式 provider/model(恰好一个 '/',两侧均非空;--format json 同版本的 --model 帮助)", model)
	}
	for _, part := range []string{provider, spec} {
		if !validFlagToken(part) {
			return fmt.Errorf("opencode: Config.Model %q 含旗标取值不安全字符(允许:字母数字与 . _ -;拒绝可疑取值而不做转义)", model)
		}
	}
	return nil
}

// validateAgentFlag 校验期望 Agent 的 `--agent` 旗标取值:单 token,不含
// '/' 与空白,字符集同 validateModelFlag。
func validateAgentFlag(agent string) error {
	if agent == "" {
		return nil
	}
	if !validFlagToken(agent) {
		return fmt.Errorf("opencode: Config.Agent %q 含旗标取值不安全字符(允许:字母数字与 . _ -;拒绝可疑取值而不做转义)", agent)
	}
	return nil
}

// validFlagToken 报告 s 是否可安全裸传 argv 值位:非空,且仅含字母数字与
// . _ -(观测形态 "deepseek/deepseek-v4-pro"、"build" 的字符集,'/' 在
// Model 中由 validateModelFlag 单独定位)。
func validFlagToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// RuntimeSnapshot 是 runtime 身份/版本/能力的一次可记录快照(对齐
// runtimes/codex、runtimes/dsh 的 AC3 缝):供上层把「跑的是哪个 opencode、
// 什么版本、声明了什么能力」落成审计记录。
type RuntimeSnapshot struct {
	// Name 是 runtime 标识(DescriptorName)。
	Name string
	// AdapterVersion 是本适配器版本(DescriptorVersion)。
	AdapterVersion string
	// CLIVersion 是现场观察的 opencode 版本(经 `--version`;已通过钉版闸,
	// 等于生效钉住值),非硬编码。
	CLIVersion string
	// BinaryPath 是 Config.BinPath 解析出的可执行文件路径。
	BinaryPath string
	// Capabilities 是声明能力(字典序)。声明由 conformance 另行验证。
	Capabilities []agentruntime.Capability
}

// Runtime 是 OpenCodeRuntime 的 agentruntime.Runtime 实现。
type Runtime struct {
	cfg Config

	mu      sync.Mutex
	seq     int // 已开启会话计数(会话编号)
	binPath string
	cli     cliIdentity // `opencode --version` 现场观察缓存(每 Runtime 至多探一次)
}

// New 构造 OpenCodeRuntime。BinPath 必填(Validate/Start 强制),其余零值
// 合法(版本走 PinnedOpenCodeVersion 钉版闸)。
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
// 失败(BinPath 不可用、--version 不可观察、版本不匹配钉住值)显式报错
// —— 钉版候选表面的身份不可观察即不可用(fail-closed,见包注释)。
func (r *Runtime) Snapshot(ctx context.Context) (RuntimeSnapshot, error) {
	if err := r.cfg.Validate(); err != nil {
		return RuntimeSnapshot{}, err
	}
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
// (未声明即拒绝是能力协商的契约方向)。
//
// Start 不 spawn run 进程:一次 Execute = 一个进程;Start 只做四件有界的事
// —— 校验配置(BinPath/model/agent 旗标取值)、解析二进制(缺失即刻报错,
// 不拖到 Execute)、执行钉版身份闸(--version 现场观察 + 与钉住值比对,
// 失败即拒绝;结果缓存于 Runtime 跨会话共享)。随后建立会话状态与全新
// per-session 隔离目录树(isolation.go)。
func (r *Runtime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	if err := agentruntime.CheckStart(r.Descriptor(), req); err != nil {
		return nil, err
	}
	if err := r.cfg.Validate(); err != nil {
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

	// per-session 隔离目录树(XDG 重定向,包注释):每个新会话一个全新临时
	// 目录;Close 时删除。owner 的 config/data(含 auth.json)零读取零写入。
	// 创建失败 = Start 显式失败(不留无边界的会话)。
	dirs, err := newSessionDirs()
	if err != nil {
		return nil, err
	}
	return &Session{
		runtime:   r,
		bin:       bin,
		index:     n,
		session:   agentruntime.SessionID(fmt.Sprintf("opencode-session-%06d", n)),
		workspace: req.Workspace,
		dirs:      dirs,
		runs:      make(map[agentruntime.RunID]*run),
	}, nil
}

// ensureBinary 解析并缓存 opencode 可执行路径;BinPath 必填(canary 候选
// 不做隐式发现),指向缺失路径即刻显式报错。
func (r *Runtime) ensureBinary() (string, error) {
	r.mu.Lock()
	if r.binPath != "" {
		bin := r.binPath
		r.mu.Unlock()
		return bin, nil
	}
	r.mu.Unlock()
	if err := requireFile(r.cfg.BinPath, "Config.BinPath"); err != nil {
		return "", err
	}
	r.mu.Lock()
	r.binPath = r.cfg.BinPath
	r.mu.Unlock()
	return r.cfg.BinPath, nil
}

// envList 把 Config.Env 转成 KEY=VALUE 切片(键序确定;身份探针用)。
func (r *Runtime) envList() []string { return envPairs(r.cfg.Env) }

// now 返回事件时间戳来源。(本适配器未声明 event.streaming,时间戳暂无
// 投放面;保留与 dsh 同形的注入缝供未来事件面使用。)
func (r *Runtime) now() time.Time {
	if r.cfg.Now != nil {
		return r.cfg.Now()
	}
	return time.Now()
}

// Session 是 OpenCodeRuntime 的 agentruntime.Session 实现:一个 Tally 侧
// 有界状态对象。协议 SessionID 是适配器自造的会话编号(包注释:真实
// ses_* 句柄属每 run 观察面,刻意不上报);同一会话的 Execute 串行化
// (execMu):run 是一次性进程,同一会话上的并发 turn 无观测依据,不做
// 并发假设。
type Session struct {
	runtime *Runtime
	bin     string
	index   int
	// session 是适配器自造的协议会话编号(opencode-session-NNNNNN)。它标识
	// Tally 协议层的会话边界,不是 opencode 的内部 ses_* 句柄;两者不可混淆。
	session agentruntime.SessionID
	// workspace 是 Start 收到的源码执行目录:spawn cwd 钉定它(run 无需
	// --dir 旗标;效果级证据:真实 run 在 cwd 内创建文件成功,Phase A 观测)。
	workspace string
	// dirs 是本会话专属的 per-session 隔离目录树(isolation.go):Start 时
	// 全新创建,Close 时删除;同一会话的全部 Execute 共享,跨会话永不共享。
	dirs sessionDirs

	// execMu 串行化同会话的 run 进程(一个会话同时最多一个进程)。
	execMu sync.Mutex

	mu     sync.Mutex
	runSeq int
	runs   map[agentruntime.RunID]*run
	closed bool
}

// run 是会话内一次运行(一个 run 进程)的有界状态。
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

// Execute 同步执行一个 turn:spawn 一个 `opencode run` 进程,读取 stdout
// JSON 事件流(或降级原文),返回 runtime 的 AgentRunResult(claim,
// INV-AUTH-01)。取消(ctx 取消或 Cancel)是正常终态:返回 RunCancelled 的
// 结果而非 error;进程崩溃/非零退出/契约不兑现(成功退出但无任何回答)都
// 归为 RunFailed claim,绝不把取消或环境错误伪装成候选失败。
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

// Cancel 请求终止指定 run:整树终止底层进程。run 不存在返回 ErrRunNotFound;
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

// Events 返回 ErrStreamingNotDeclared:本适配器未声明 event.streaming
// (诚实能力集,空基础集)。--format json 的事件流是一次性进程 stdout 的
// 后验输出,不是可独立订阅的会话事件词汇表;把它声明成流式能力即发明
// (CheckEvents 是协议统一前置闸)。会话关闭同样以该未声明拒绝为先
// (对齐 runtimes/fake、runtimes/dsh 顺序)。
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
// 拒绝后续操作,最后删除本会话的 per-session 隔离目录树(尽力而为:Close
// 不等待 in-flight 进程收敛,删除失败不作为 Close 错误;进程终止而未 Close
// 的会话,目录留在系统临时目录由 OS 清理兜底 —— 与 runtimes/codex、
// runtimes/dsh 同一策略)。幂等。
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
	dirs := s.dirs
	s.mu.Unlock()
	for _, r0 := range running {
		s.requestCancelRun(r0)
	}
	dirs.remove()
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

// RunProcess 返回 in-flight run 的底层 opencode 进程(诊断/测试注入面,
// 例如 PRD §35 crash 场景的外部 kill 注入;返回的是 opencode 进程本身,不是
// 其子进程树)。run 不存在或已到终态返回 false。
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
// 只报告已观察到底层进程的 run(注入者需要可 kill 的进程句柄);无则返回
// false。
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

// requireFile 校验路径存在且是普通文件,否则给出指明来源的显式错误。
func requireFile(path, source string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("opencode: %s 解析到的路径不可用 %q: %w", source, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("opencode: %s 解析到的是目录而非可执行文件 %q", source, path)
	}
	return nil
}

// ensureCLIIdentity 现场执行 `<bin> --version`(观测形态:裸版本行
// "1.19.6",exit 0)并缓存解码结果。探针没跑起来、非零退出或输出不可解析
// 都显式报错且不缓存。与 runtimes/codex 的「尽力而为探针」不同,本适配器
// 在 Start/Snapshot 把它作为钉版闸的硬前提(fail-closed,见包注释)。
func (r *Runtime) ensureCLIIdentity(ctx context.Context, bin string) (cliIdentity, error) {
	r.mu.Lock()
	if r.cli.observed() {
		c := r.cli
		r.mu.Unlock()
		return c, nil
	}
	r.mu.Unlock()
	res, err := probeVersion(ctx, bin, r.envList())
	if err != nil {
		return cliIdentity{}, err
	}
	c := parseCLIIdentity(res)
	if !c.observed() {
		return cliIdentity{}, fmt.Errorf("opencode: --version 输出不可解析: %q", truncate(res, 200))
	}
	r.mu.Lock()
	r.cli = c
	r.mu.Unlock()
	return c, nil
}

// checkVersionPin 把现场观察的 CLI 身份与钉住版本比对:不一致即拒绝 ——
// 适配器的 argv/输出契约只对钉住版本有效(上游迭代快,run 表面旗标随版本
// 增删;--standalone 的存在性即随版本漂移的实例)。
func checkVersionPin(pinned string, cli cliIdentity) error {
	if pinned == "" {
		pinned = PinnedOpenCodeVersion
	}
	if cli.Version != pinned {
		return fmt.Errorf("opencode: 观察到的 CLI 版本 %q 与钉住版本 %q 不一致(适配器契约按精确版本钉住):拒绝开启会话", cli.Version, pinned)
	}
	return nil
}
