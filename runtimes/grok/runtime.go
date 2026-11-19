// Package grok 实现 xAI 官方 Grok Build CLI(@xai-official/grok)的
// agentruntime.Runtime 适配器,严格按对本机安装的官方 CLI grok 1.0.87
// (commit 929e6dd187b1,win32-x64,2026-11-19 现场观测)观测到的真实表面实现,
// 不发明任何 CLI 旗标或协议(Runtime lock,AGENTS.md §5)。下文所有「观测」
// 均指对该版本二进制的现场观测;未观测的表面一律不使用、不解释。
//
// 上游身份与安装(本机观测):
//   - 官方包名 @xai-official/grok(npm,bin: grok;maintainer xai-security
//     <security@x.ai>)。npm dist-tag latest 停留在陈旧的 0.1.4(2025-10),
//     实际版本线已演进到 1.0.x(本观测安装 1.0.87,2026-11-14 发布)——
//     dist-tag 不可作版本依据,必须显式钉版。
//   - npm 包本体是引导器:postinstall(node bin/postinstall.js)从平台包
//     @xai-official/grok-win32-x64 解出原生 Rust 二进制(~186MB),安装到
//     $GROK_HOME/bin/grok.exe(Windows 为拷贝;postinstall 源码注释:
//     "$GROK_HOME (else ~/.grok), matching the Rust grok_home()")。适配器
//     契约锚定该原生二进制,不锚定 npm 引导器。
//
// 集成表面(观测自 grok 1.0.87 --help / --version / keyless -p 现场执行):
//   - 一次 Execute = 一个一次性进程:`grok --no-auto-update
//     [-m <model>] [--permission-mode <mode>] --single=<instruction>`。
//     -p/--single:「Single-turn prompt. Prints the response to stdout and
//     exits」(--help 原文)。stdout = 最终回答(--output-format 默认 plain,
//     --help 原文「[default: plain]」);exit 0 = completed;非 0 = 未完成。
//     无长驻守护进程。形态: Tally Go → runtimes/grok → 外部 grok 进程。
//   - instruction 以 --single=<instruction> 附值形态传入(观测:裸
//     `-p "-leading dash task"` 被 clap 误解析为未知旗标 exit 2,而
//     `--single="-leading dash task"` 正确接受为值、推进到认证错误 exit 1;
//     附值形态对任意指令文本安全,指令恒为单个 argv 元素,无 shell 解析)。
//   - --no-auto-update 强制:官方文档 headless-scripting 明示 scripted
//     environments 应传该旗标,现场观测接受(未列于 --help,属隐藏旗标;
//     观测形态 `grok --no-auto-update -p "x"` → 认证错误 exit 1 而非
//     usage 错误 exit 2)。自动更新会把新二进制写进 GROK_HOME/bin —— 候选
//     执行不允许该出口;per-session GROK_HOME 隔离(isolation.go)是其第二道界。
//   - 认证 = XAI_API_KEY 环境变量(docs.x.ai 与 keyless stderr 原文:
//     「set the XAI_API_KEY environment variable」);交互登录
//     (grok login / --device-code)是另一路径,不适配 headless。keyless/
//     坏 key 现场观测:exit 1,stdout 为空,stderr 为「Not signed in. ...」
//     —— 适配器如实归类为 RunFailed claim(stderr 原文进 summary),
//     绝不伪装成功,也不是 Execute error。
//   - 退出码观测:0 = 成功(--version/-p 完成路径);1 = 运行时错误
//     (认证失败);2 = usage 错误(未知旗标)。取消未观测(本机无凭据,
//     无法建立 in-flight run);适配器按进程树终止实现取消,与
//     runtimes/codex、runtimes/dsh 同语义,teardown 由进程级测试证明。
//   - workspace 钉定不经旗标(--cwd <CWD> 在 --help 声明,但其效果未经
//     真实回合观测,不使用):workspaceRoot = 进程 cwd(inspect 现场观测
//     「CWD: <进程 cwd>」),由 spawn cwd = 可丢弃 worktree 实现 —— 与
//     CodexRuntime 的 -C 钉定同一语义(INV-EXEC-01)。
//   - 期望模型经观测旗标 -m/--model <MODEL> 传入(--help 声明);Config.Model
//     为空则不传旗标(零 overlay,逐字节朴素调用)。它是期望配置:与实际
//     生效模型(headless plain 表面不报告)是两个事实,绝不互相回填
//     (AGENTS.md §13)。
//   - 期望审批模式经观测旗标 --permission-mode <MODE> 传入(--help 声明
//     枚举:default, acceptEdits, auto, dontAsk, bypassPermissions, plan)。
//     各取值的运行期语义未经真实回合观测(无凭据),Config 只做枚举透传,
//     空 = 不传旗标(CLI 自身 default)。审批升级只经显式 Config。
//   - 隔离 = per-session 全新 GROK_HOME(临时目录,Close 时删除,
//     isolation.go):sessions(~/.grok/sessions,docs)、config.toml、
//     leader socket(~/.grok/leader.sock,--help 原文)都在 grok home 下。
//     空临时 GROK_HOME 现场观测可用(--version exit 0,目录保持为空,
//     无 ~/.grok 写入)。
//   - spawn 环境走白名单(isolation.go):只拷贝 OS 基础键 + 注入
//     Config.Env —— 被 spawn 的 agent 及其工具子进程绝不能读到 owner 环境
//     里的其他 provider 认证(如 DEEPSEEK_API_KEY),这比 dsh 的「继承 +
//     强制覆盖」更严格,是本适配器的显式要求。
//   - 产出的一切(AgentRunResult)都是 Agent Claim(INV-AUTH-01):runtime
//     自述的完成不是 Tally 事实,不能替代验证器证据。
//
// 钉版身份闸:Grok Build 是快速演进的 preview 级表面(npm ~13 个月发布
// 242 个版本;dist-tag latest 陈旧失准即既有实例),适配器的 argv/退出码
// 契约只对钉住版本 PinnedGrokVersion 有效。Start/Snapshot 现场执行
// `grok --version`(观测形态:stdout 首行 "grok 1.0.87 (929e6dd187b1)",
// exit 0),观察失败或版本与钉住值不一致即拒绝开启会话(fail-closed)。
// 这与 runtimes/codex 的「探针失败 = 事实未观察,Start 继续」不同且为有意
// 偏离:codex 是有探针记录背书的稳定 GA 表面。适配器版本(DescriptorVersion)
// 与 CLI 版本是两个事实,后者只来自现场观察(AGENTS.md §13),绝不互相冒充。
//
// 能力声明(诚实清单,保守映射):headless --single 表面一次一个一次性进程,
// stdout 只有最终回答 —— 基础声明为空集(见 Capabilities);session.isolation
// 仅在 Config.SessionIsolationVerified 持有共享 conformance canary 证据时
// 声明(与 runtimes/codex、runtimes/dsh 同一证据门纪律,声明真实性 PRD §35)。
// 声明本身仍是 claim,由 conformance(PRD §10.4)另行验证。
//
// 已知边界(如实声明,不是缺陷隐藏):
//   - Usage 恒为零值:headless plain 表面不自报用量;--output-format
//     json/streaming-json 或许携带用量,但其 schema 未经真实回合观测,
//     解析即发明 —— 宁缺毋假。
//   - 协议 SessionID 是适配器自造的会话编号(如 "grok-session-000001"),
//     不是 grok 的任何内部句柄 —— plain 一次性表面不给出可观察的会话句柄,
//     ObservedIdentity.SessionID 恒为空(未观察 ≠ 编造)。
//   - ObservedIdentity.Model 恒为空:plain 表面不报告实际生效模型;期望配置
//     (Config.Model)绝不回填观察字段(AGENTS.md §13)。
//   - -s/--session-id、-r/--resume、-c/--continue、--json-schema、
//     --max-turns 等旗标在 --help 声明但效果未经真实回合观测:适配器一律
//     不使用,相应能力(session.resume/fork/persist、output.structured 等)
//     不声明。
package grok

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

// DescriptorName 是 GrokRuntime 的 runtime 标识。只用于展示/遥测/归因;
// 任何控制流都不得按它分支(PRD §10.3)。
const DescriptorName = "grok"

// DescriptorVersion 是本适配器(不是 CLI)的版本。CLI 版本由 Snapshot/Start
// 经 `grok --version` 现场发现后记录,绝不硬编码冒充观察(AGENTS.md §13)。
const DescriptorVersion = "1"

// PinnedGrokVersion 是本适配器契约锚定的上游精确版本(本机安装并完成全部
// 现场观测的版本,npm 包 @xai-official/grok@1.0.87 的原生二进制,commit
// 929e6dd187b1)。Config.PinnedVersion 为空时即用它。
const PinnedGrokVersion = "1.0.87"

// permissionModes 是 --permission-mode 的全部取值(grok 1.0.87 --help 逐字
// 枚举:「[possible values: default, acceptEdits, auto, dontAsk,
// bypassPermissions, plan]」)。各取值的运行期语义未经真实回合观测(本机无
// 凭据);适配器只做枚举透传,不解释、不猜测。
var permissionModes = map[string]bool{
	"default":           true,
	"acceptEdits":       true,
	"auto":              true,
	"dontAsk":           true,
	"bypassPermissions": true,
	"plan":              true,
}

// Capabilities 返回 Grok headless 的基础能力声明 —— 空集(诚实清单,保守
// 映射)。--single 是一次性进程,stdout 只有最终回答(plain 默认):
//
//   - 无事件词汇表 → 不声明 event.streaming;
//   - resume 旗标存在但效果未观测 → 不声明 session.resume / session.fork;
//   - 无结构化输出约定(plain 默认)→ 不声明 output.structured;
//   - -m 是期望配置,实际生效模型不可观察 → 不声明 model.select(requested
//     ≠ observed 永不成立,AGENTS.md §13);
//   - 取消 = 进程终止 = 核心 Cancel 语义 → 不声明 turn.interrupt / turn.steer;
//   - 无 headless 审批通道 → 不声明 tool.approval;无工具注入面 → 不声明
//     tools.custom;
//   - --sandbox profile 存在但效果未观测,Tally 侧可丢弃 worktree 才是边界
//     → 不声明 sandbox.native;
//   - 会话持久化未经观测 → 不声明 session.persist;
//   - 无跨会话记忆声明 → 不声明 memory.declared。
//
// 声明由 conformance(PRD §10.4/§35)另行验证;声明本身仍是 claim。实例生效
// 声明见 capabilities(session.isolation 证据门)。
func Capabilities() agentruntime.CapabilitySet {
	return agentruntime.NewCapabilitySet()
}

// capabilities 返回本 Runtime 实例生效的能力声明:空基础集 + session.isolation
// (当且仅当 SessionIsolationVerified 置位 —— 持有「共享 conformance Session
// Isolation Canary 已对同一真实 grok + 本 per-session GROK_HOME 隔离实现运行
// 并通过」的证据)。零值保持未声明:实现存在性不等于真实表面证据。
func (r *Runtime) capabilities() agentruntime.CapabilitySet {
	if !r.cfg.SessionIsolationVerified {
		return Capabilities()
	}
	return agentruntime.NewCapabilitySet(append(Capabilities().List(), agentruntime.CapSessionIsolation)...)
}

// Config 是 GrokRuntime 的构造配置。New 之后必须保持只读。
type Config struct {
	// BinPath 显式指定 grok 原生二进制路径。显式必填(空值 Validate 拒绝,
	// 不做隐式发现):npm dist-tag latest 已观测失准(停在陈旧 0.1.4),
	// PATH 上的 "grok" 可能是任意来源,候选执行不接受隐式二进制。测试注入
	// 替身二进制也走本字段。
	BinPath string
	// PinnedVersion 是钉住的上游精确版本;空则 PinnedGrokVersion。Start/Snapshot
	// 现场观察 `--version` 与它不一致即 fail-closed。刻意不提供「关掉钉版」
	// 的取值:快速演进表面的契约没有版本锚即不成立。
	PinnedVersion string
	// Model 是期望模型(观测旗标 -m/--model <MODEL>;如 "grok-4.7")。
	// 空 = 不传旗标。它只是期望配置:与实际生效模型(plain 表面不可观察)
	// 是两个事实,绝不互相回填(AGENTS.md §13)。
	Model string
	// PermissionMode 是 --permission-mode 的取值(观测枚举见 permissionModes);
	// 空 = 不传旗标(CLI 自身 default)。只接受观测枚举值,其它值在 Validate/
	// Start 处拒绝。审批升级只经显式 Config,不经继承环境。
	PermissionMode string
	// Env 追加到被 spawn 进程的环境变量(不是完整环境快照)。语义对齐
	// command.CommandSpec.Env;典型用途:XAI_API_KEY(认证,观测的 supported
	// 路径)、测试注入。凭据值绝不落盘、绝不进日志/错误串。GROK_HOME 一个键
	// 被隔离逻辑强制覆盖(isolation.go),Config.Env 中的同名值不生效。spawn
	// 环境是白名单制:owner 环境中未被白名单拷贝的键(含其他 provider 认证)
	// 不会到达子进程。
	Env map[string]string
	// SessionIsolationVerified 是 session.isolation 的声明门(与 runtimes/codex
	// 同一证据门纪律):置 true 断言「共享 conformance Session Isolation Canary
	// 已对与生产同一解析出的真实 grok + 本 per-session GROK_HOME 隔离实现运行
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

// Validate 预检配置面(Invalid configuration fails before task execution 的
// 最早闸):BinPath 显式必填、期望模型非空白、权限枚举取值合法。不发起进程、
// 不做任何观察(钉版身份闸仍在 Start)。
func (c Config) Validate() error {
	if strings.TrimSpace(c.BinPath) == "" {
		return fmt.Errorf("grok: Config.BinPath 不能为空(显式必填:npm dist-tag latest 已观测失准,PATH 上的 grok 来源不可控,不做隐式发现)")
	}
	if c.Model != "" && strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("grok: Config.Model 不能是纯空白")
	}
	if c.PermissionMode != "" && !permissionModes[c.PermissionMode] {
		return fmt.Errorf("grok: 非法 PermissionMode %q(grok 1.0.87 --help 观测枚举: default | acceptEdits | auto | dontAsk | bypassPermissions | plan;各值运行期语义未观测,适配器只透传)", c.PermissionMode)
	}
	return nil
}

// RuntimeSnapshot 是 runtime 身份/版本/能力的一次可记录快照(对齐
// runtimes/codex 的 AC3 缝):供上层把「跑的是哪个 grok、什么版本、声明了
// 什么能力」落成审计记录。
type RuntimeSnapshot struct {
	// Name 是 runtime 标识(DescriptorName)。
	Name string
	// AdapterVersion 是本适配器版本(DescriptorVersion)。
	AdapterVersion string
	// CLIVersion 是现场观察的 grok 版本(经 `--version`;已通过钉版闸,等于
	// 生效钉住值),非硬编码。
	CLIVersion string
	// BinaryPath 是现场解析出的 grok 二进制路径。
	BinaryPath string
	// Capabilities 是声明能力(字典序)。声明由 conformance 另行验证。
	Capabilities []agentruntime.Capability
}

// Runtime 是 GrokRuntime 的 agentruntime.Runtime 实现。
type Runtime struct {
	cfg Config

	mu      sync.Mutex
	seq     int // 已开启会话计数(会话编号)
	binPath string
	cli     cliIdentity // `grok --version` 现场观察缓存(每 Runtime 至多探一次)
}

// New 构造 GrokRuntime。BinPath 显式必填(Config.Validate 拒绝空值)。
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
// 失败(路径不可用、--version 不可观察、版本不匹配钉住值)显式报错
// —— 钉版表面的身份不可观察即不可用(fail-closed,见包注释)。
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
// (未声明即拒绝是能力协商的契约方向)。
//
// Start 不 spawn headless 进程:一次 Execute = 一个进程;Start 只做四件有界
// 的事 —— 校验配置(BinPath/Model/PermissionMode)、执行钉版身份闸
// (--version 现场观察 + 与钉住值比对,失败即拒绝;结果缓存于 Runtime 跨
// 会话共享)、解析二进制路径(缺失即刻报错,不拖到 Execute)。随后建立会话
// 状态与全新 per-session GROK_HOME(isolation.go)。
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

	// per-session GROK_HOME 隔离:每个新会话一个全新临时 home;Close 时删除。
	// 无 auth 移交(~/.grok 下凭据文件格式未观测,不发明 —— 认证走
	// XAI_API_KEY 环境变量这一观测过的 supported 路径);无凭据的部署形态
	// 将以「Not signed in」的 RunFailed claim 显形,不被隐藏。创建失败 =
	// Start 显式失败(不留无边界的会话)。
	home, err := newSessionHome()
	if err != nil {
		return nil, err
	}
	return &Session{
		runtime:   r,
		bin:       bin,
		index:     n,
		session:   agentruntime.SessionID(fmt.Sprintf("grok-session-%06d", n)),
		workspace: req.Workspace,
		home:      home,
		runs:      make(map[agentruntime.RunID]*run),
	}, nil
}

// ensureBinary 校验并缓存 Config.BinPath 指向的二进制;缺失即刻报错
// (显式错误,不静默)。BinPath 显式必填,无发现流程。
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

// now 返回事件时间戳来源。
func (r *Runtime) now() time.Time {
	if r.cfg.Now != nil {
		return r.cfg.Now()
	}
	return time.Now()
}

// requireFile 校验路径存在且是普通文件,否则给出指明来源的显式错误。
func requireFile(path, source string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("grok: %s 解析到的路径不可用 %q: %w", source, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("grok: %s 解析到的是目录而非可执行文件 %q", source, path)
	}
	return nil
}

// ensureCLIIdentity 现场执行 `<bin> --version`(观测形态:stdout 首行
// "grok 1.0.87 (929e6dd187b1)",exit 0)并缓存解码结果。探针没跑起来、
// 非零退出或输出不可解析都显式报错且不缓存。与 runtimes/codex 的「尽力而为
// 探针」不同,本适配器在 Start/Snapshot 把它作为钉版闸的硬前提(见包注释的
// 有意偏离)。
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
		return cliIdentity{}, fmt.Errorf("grok: 读取 --version 失败: %w", err)
	}
	if res.ExitCode != 0 {
		return cliIdentity{}, fmt.Errorf("grok: --version 退出码 %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	c := parseCLIIdentity(res.Stdout)
	if !c.observed() {
		return cliIdentity{}, fmt.Errorf("grok: --version 输出不可解析: %q", truncate(strings.TrimSpace(res.Stdout), 200))
	}
	r.mu.Lock()
	r.cli = c
	r.mu.Unlock()
	return c, nil
}

// checkVersionPin 把现场观察的 CLI 身份与钉住版本比对:不一致即拒绝 ——
// 适配器的 argv/退出码契约只对钉住版本有效,快速演进上游以既有实例证明
// 会破坏表面(npm dist-tag latest 失准、242 版本/13 个月)。
func checkVersionPin(pinned string, cli cliIdentity) error {
	if pinned == "" {
		pinned = PinnedGrokVersion
	}
	if cli.Version != pinned {
		return fmt.Errorf("grok: 观察到的 CLI 版本 %q 与钉住版本 %q 不一致(快速演进表面按精确版本钉住):拒绝开启会话", cli.Version, pinned)
	}
	return nil
}

// Session 是 GrokRuntime 的 agentruntime.Session 实现:一个 Tally 侧有界状态
// 对象。协议 SessionID 是适配器自造的会话编号(--single 表面不给出任何协议层
// 会话句柄,见包注释);同一会话的 Execute 串行化(execMu):--single 是
// 一次性进程,同一会话上的并发 turn 无观测依据,不做并发假设。
type Session struct {
	runtime *Runtime
	bin     string
	index   int
	// session 是适配器自造的协议会话编号(grok-session-NNNNNN)。它标识 Tally
	// 协议层的会话边界,不是 grok 的任何内部句柄;两者不可混淆。
	session agentruntime.SessionID
	// workspace 是 Start 收到的源码执行目录:spawn cwd 钉定它(--cwd 旗标
	// 效果未观测不用;inspect 观测 workspaceRoot = 进程 cwd,包注释)。
	workspace string
	// home 是本会话专属的 per-session GROK_HOME(isolation.go):Start 时全新
	// 创建,Close 时删除;同一会话的全部 Execute 共享,跨会话永不共享。
	home string

	// execMu 串行化同会话的 --single 进程(一个会话同时最多一个进程)。
	execMu sync.Mutex

	mu     sync.Mutex
	runSeq int
	runs   map[agentruntime.RunID]*run
	closed bool
}

// run 是会话内一次运行(一个 --single 进程)的有界状态。
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

// Execute 同步执行一个 turn:spawn 一个 `grok --single` 进程,读取 stdout
// 最终回答,返回 runtime 的 AgentRunResult(claim,INV-AUTH-01)。取消
// (ctx 取消或 Cancel)是正常终态:返回 RunCancelled 的结果而非 error;
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

// Events 返回 ErrStreamingNotDeclared:Grok headless 未声明 event.streaming
// (诚实能力集,空基础集)。stdout 只有最终回答、--output-format
// streaming-json 未经真实回合观测,解释即发明(CheckEvents 是协议统一前置闸)。
// 会话关闭同样以该未声明拒绝为先(对齐 runtimes/fake 顺序)。
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
// 拒绝后续操作,最后删除本会话的 per-session GROK_HOME(尽力而为:Close 不
// 等待 in-flight 进程收敛,删除失败不作为 Close 错误;进程终止而未 Close 的
// 会话,home 留在系统临时目录由 OS 清理兜底 —— 与 runtimes/codex、
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

// RunProcess 返回 in-flight run 的底层 grok 进程(诊断/测试注入面,例如
// PRD §35 crash 场景的外部 kill 注入;返回的是 grok 本身,不是其子进程树)。
// run 不存在或已到终态返回 false。
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
