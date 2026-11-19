// Package pi 实现 pi coding agent CLI 的 agentruntime.Runtime 适配器
// (v2.1 provider candidate),严格按本机对钉住版本 pi 0.85.1 的现场观测实现,
// 不发明任何 CLI 旗标或输出契约(Runtime lock,AGENTS.md §5;期望 ≠ 观察,
// AGENTS.md §13 —— 本包文档中的每一条事实都标注观测来源,未观测的不发明):
//
//   - 集成表面:`pi -p ... -- <instruction>`(非交互 print 模式)—— 一次
//     Execute = 一个一次性进程,无长驻守护进程。观测(Windows,0.85.1):
//     exit 0 = 进程自述成功;exit 非 0 = 未完成(如 keyless 时 exit 1 +
//     stderr "No API key found for <provider>.");取消 = 进程终止。
//     形态: Tally Go → runtimes/pi → 外部 pi 进程。
//   - 输出模式(观测 `--mode text|json`):text = stdout 是最终回答纯文本;
//     json = stdout 是 NDJSON 事件流,逐行一个 JSON 对象:
//     首行 {"type":"session","version":3,"id":"<uuid>","cwd":...}(即使
//     --no-session 也有此行,id 是 pi 自报的会话 UUID),其后 agent_start /
//     turn_start / message_start|message_end(user/assistant)/ message_update /
//     turn_end / agent_end / agent_settled;assistant 的 message_end 携带
//     content[].text、provider、model、usage{input,output,cacheRead,...} 与
//     stopReason —— 本适配器的 Summary/Usage/实际模型观察都取自该行。
//     默认 --mode json(可稳定解析、usage 与实际模型可观察);text 模式保留为
//     显式降级选项(usage 恒零值)。mode rpc 未观测,不使用。
//   - workspace 钉定不经旗标(pi 帮助文本无 cwd 旗标):workspace = 进程 cwd
//     (观测:pi 的工具在进程 cwd 内真实写文件,如 pi-smoke.txt 落在 cwd),
//     由 spawn cwd = 可丢弃 worktree 实现 —— 与 runtimes/codex 的 -C 钉定
//     同一语义(INV-EXEC-01)。tools 默认启用(观测:默认面下 pi 能真实创建
//     文件),不做任何工具白名单裁剪 —— 候选执行的写边界是 Tally 的可丢弃
//     worktree,不是 pi 自身声明(pi 不提供可信完整 OS 沙箱;见隔离节)。
//   - 期望模型/提供者经观测旗标传递:`--provider <name>`(帮助文本默认
//     google)、`--model <pattern>`、`--thinking <level>`(帮助文本观测枚举:
//     off|minimal|low|medium|high|xhigh|max)。它们只是期望配置;实际生效
//     provider/model 在 json 模式从 assistant message_end 观察,与期望分开
//     携带(Decision Lock 15),绝不互相回填(AGENTS.md §13)。
//   - 认证(观测):pi 按 provider 读环境变量(帮助文本 Environment Variables
//     节;实测 deepseek provider 的 readiness 随 DEEPSEEK_API_KEY 的存在性
//     翻转:有 = ready(authType api_key),无 = not_ready);`--api-key <key>`
//     旗标存在但本适配器不使用 —— 凭据绝不进 argv(进程列表可观察面)。
//     本适配器绝不读取或移交任何认证目录(含 owner 的 ~/.pi):凭据只经
//     Config.Env 注入对应环境变量。
//   - 产出的一切(AgentRunResult、Event)都是 Agent Claim,不是 Project Fact
//     (INV-AUTH-01):pi 自述的完成不是 Tally 事实,不能替代验证器证据。
//
// 钉版身份闸:pi 迭代活跃(帮助文本展示大量演进中表面),适配器的 argv/输出
// 契约只对钉住版本 PinnedPiVersion 有效。Start/Snapshot 现场执行
// `pi --version`(观测形态:裸版本行 "0.85.1",exit 0),观察失败或版本与
// 钉住值不一致即拒绝开启会话(fail-closed;与 runtimes/dsh 同一纪律,与
// runtimes/codex 的「探针失败 = 事实未观察,Start 继续」不同且为有意偏离)。
// 适配器版本(DescriptorVersion)与 CLI 版本是两个事实,后者只来自现场观察,
// 绝不互相冒充。
//
// 隔离(Tally Host 隔离;pi 自身无可信完整 OS 沙箱声明,绝不依赖):
//   - cwd = StartRequest.Workspace(可丢弃 worktree,INV-EXEC-01);
//   - 每会话全新临时目录组(isolation.go):PI_CODING_AGENT_DIR(pi 帮助文本
//     观测的 config 目录环境变量;实测重定向下 pi 把 auth.json/models-store.json
//     以空对象 "{}" 形态脚手架进重定向目录,owner ~/.pi 零触碰)、
//     PI_CODING_AGENT_SESSION_DIR 与 --session-dir(会话存储;帮助文本注明
//     旗标压倒环境变量)、HOME/USERPROFILE(~ 解析兜底重定向,实测可行);
//     Close 删除。--no-session 保证不落会话文件(session-dir 是防御纵深:
//     若未来版本改语义,文件也落在随会话删除的临时目录内;实测 --no-session
//     下无文件、不设 --no-session 时 .jsonl 会话文件确落 --session-dir)。
//   - 发现面全关(全部为 pi 帮助文本观测旗标,组合实测可行):
//     --no-extensions --no-skills --no-prompt-templates --no-themes
//     --no-context-files(AGENTS.md/CLAUDE.md 不加载),另加 --no-approve
//     (帮助文本:忽略项目本地文件)。owner 与 workspace 侧的 pi 资产都不进
//     候选执行。
//   - spawn 环境强制面(isolation.go):剥离 pi 帮助文本观测的全部 provider
//     凭据环境变量(防 operator 环境的 ANTHROPIC_API_KEY 等静默生效),强制
//     PI_OFFLINE=1 / PI_TELEMETRY=0(帮助文本观测的环境开关;PI_OFFLINE=1
//     实测不阻碍真实模型回合),再层叠 Config.Env(认证透传通道;强制键
//     压倒 Config.Env 同名值)。
//
// 能力声明(诚实清单,保守映射):一次 Execute = 一个一次性同步进程,本适配器
// 基础声明为空集(见 Capabilities)。pi CLI 自身有 resume/fork 等旗标,但
// 本适配器的一次性执行不用它们 —— 声明必须以「本适配器真实兑现」为准。
// 声明由 conformance(PRD §10.4)另行验证;声明本身仍是 claim。
//
// 已知边界(如实声明,不是缺陷隐藏):
//   - instruction 以 "--" 后单位置参数传入(pi 帮助文本观测:-- 结束选项
//     解析)。pi 把开头为 "@" 的位置参数当作文件附件(帮助文本 @files 语法,
//     未逐项观测)—— 以 @ 开头的 Tally instruction 是本适配器的未观测边界,
//     语义由 pi 定义,不在本包内发明规避。
//   - text 模式 Usage 恒为零值(text stdout 无用量自报);json 模式取最后
//     一条 assistant message_end 的 usage.input/output(pi 的 cacheRead 等
//     其余用量维度在协议 Usage 形状之外,不发明映射)。
//   - ObservedIdentity.SessionID 是 pi 自报的会话 UUID(json 流 session 行;
//     --no-session 下为一次性句柄,本适配器不声明 session.resume/persist,
//     不构成可续用承诺)。
package pi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

// DescriptorName 是 PiRuntime 的 runtime 标识。只用于展示/遥测/归因;
// 任何控制流都不得按它分支(PRD §10.3)。
const DescriptorName = "pi"

// DescriptorVersion 是本适配器(不是 CLI)的版本。CLI 版本由 Snapshot/Start
// 经 `pi --version` 现场发现后记录,绝不硬编码冒充观察(AGENTS.md §13)。
const DescriptorVersion = "1"

// PinnedPiVersion 是本适配器契约锚定的上游精确版本(本机 pi 0.85.1 完成全部
// Phase A 表面观测)。Config.PinnedVersion 为空时即用它。
const PinnedPiVersion = "0.85.1"

// DefaultOutputMode 是未配置时的输出模式:json(观测:NDJSON 事件流可稳定
// 解析,usage 与实际生效 provider/model 可观察)。合法值见 outputModes。
const DefaultOutputMode = "json"

// outputModes 是 --mode 的全部已知取值(pi 帮助文本观测:text(默认)/json/rpc;
// rpc 未观测,不进适配器合法集)。
var outputModes = map[string]bool{
	"json": true,
	"text": true,
}

// thinkingLevels 是 --thinking 的全部已知取值(pi 帮助文本观测枚举;未逐值
// live 观测,枚举本身来自 CLI 自述)。
var thinkingLevels = map[string]bool{
	"off":     true,
	"minimal": true,
	"low":     true,
	"medium":  true,
	"high":    true,
	"xhigh":   true,
	"max":     true,
}

// 类型化错误(errors.Is 判别;调用方不解析错误文本)。
var (
	// ErrBinaryMissing 表示解析出的 pi 可执行文件不存在/不可用。
	ErrBinaryMissing = errors.New("pi: pi 可执行文件缺失")
	// ErrVersionPinMismatch 表示现场观察的 CLI 版本与钉住版本不一致
	// (Start fail-closed 的依据)。
	ErrVersionPinMismatch = errors.New("pi: 观察到的 CLI 版本与钉住版本不一致")
)

// Capabilities 返回 pi 一次性执行适配器的基础能力声明 —— 空集(诚实清单,
// 保守映射)。一次 Execute = 一个同步一次性进程,stdout 输出在本适配器内部
// 消费:
//
//   - 不声明 event.streaming:json NDJSON 流是 Execute 的内部读取面,不是
//     可订阅的会话事件流(Session.Events 恒 ErrStreamingNotDeclared);
//   - pi CLI 有 resume/fork/session-dir 旗标,但本适配器一次性执行不用,
//     不声明 session.resume / session.fork / session.persist;
//   - 无 turn 内引导/打断通道 → 不声明 turn.steer / turn.interrupt
//     (取消 = 进程终止 = 核心 Cancel 语义);
//   - 无结构化输出约定 → 不声明 output.structured(--mode json 是适配器的
//     解析面,不是 pi 的输出承诺契约);
//   - pi 无可信完整 OS 沙箱 → 不声明 sandbox.native(Tally 可丢弃 worktree
//     才是边界);
//   - 无审批事件通道 → 不声明 tool.approval;无工具注入面 → 不声明
//     tools.custom;
//   - Config.Model/Provider 只是期望旗标,actual model 是观察值而不是可承诺
//     的选择协商 → 不声明 model.select;
//   - 无跨会话记忆声明 → 不声明 memory.declared;session.isolation 以证据
//     门控制(见 capabilities),零值不声明。
//
// 声明由 conformance(PRD §10.4/§35)另行验证;声明本身仍是 claim。
func Capabilities() agentruntime.CapabilitySet {
	return agentruntime.NewCapabilitySet()
}

// capabilities 返回本 Runtime 实例生效的能力声明:空基础集 + session.isolation
// (当且仅当 SessionIsolationVerified 置位 —— 持有「共享 conformance Session
// Isolation Canary 已对同一真实 pi + 本 per-session 隔离实现运行并通过」的
// 证据;与 runtimes/codex、runtimes/dsh 同一证据门纪律)。
func (r *Runtime) capabilities() agentruntime.CapabilitySet {
	if !r.cfg.SessionIsolationVerified {
		return Capabilities()
	}
	return agentruntime.NewCapabilitySet(append(Capabilities().List(), agentruntime.CapSessionIsolation)...)
}

// Config 是 PiRuntime 的构造配置。New 之后必须保持只读。
type Config struct {
	// BinPath 显式指定 pi 可执行文件路径;必填(空值 Validate 拒绝)。钉住
	// 版本面不做隐式发现:路径由部署方显式给出(与 worker canary 登记的
	// 显式二进制闸同一方向)。
	BinPath string
	// PinnedVersion 是钉住的上游精确版本;空则 PinnedPiVersion。Start/Snapshot
	// 现场观察 `--version` 与它不一致即 fail-closed。刻意不提供「关掉钉版」
	// 的取值:没有版本锚的 CLI 契约不成立。
	PinnedVersion string
	// Provider 是期望提供者(经观测旗标 --provider 传递;如 "deepseek")。
	// 空 = 不传旗标(pi 自行按帮助文本默认 google)。它只是期望配置:与实际
	// 生效 provider(json 模式可观察)是两个事实,绝不互相回填。
	Provider string
	// Model 是期望模型(经观测旗标 --model 传递;如 "deepseek-v4-flash")。
	// 空 = 不传旗标。与实际生效模型(json 模式可观察)是两个事实。
	Model string
	// ThinkingLevel 是期望思考档位(经观测旗标 --thinking 传递;合法值见
	// thinkingLevels)。空 = 不传旗标。
	ThinkingLevel string
	// OutputMode 是输出模式("json" | "text");空则 DefaultOutputMode。
	// json:Summary/Usage/ObservedIdentity 取自 NDJSON 事件流;text:Summary =
	// stdout 原文、Usage 恒零值。
	OutputMode string
	// Env 追加到被 spawn 进程的环境变量(不是完整环境快照)。典型用途:对应
	// provider 的 API key 环境变量(认证透传,如 DEEPSEEK_API_KEY)。凭据
	// 只经这里进环境,绝不进 argv、绝不落盘。PI_CODING_AGENT_DIR /
	// PI_CODING_AGENT_SESSION_DIR / HOME / USERPROFILE / PI_OFFLINE /
	// PI_TELEMETRY 六个键被隔离/硬化逻辑强制覆盖(isolation.go),Config.Env
	// 中的同名值不生效;pi 帮助文本观测的 provider 凭据环境变量被强制剥离
	//(防 operator 环境静默生效),Config.Env 是唯一显式例外通道。
	Env map[string]string
	// SessionIsolationVerified 是 session.isolation 的声明门(与 runtimes/codex
	// 同一证据门纪律):置 true 断言「共享 conformance Session Isolation Canary
	// 已对与生产同一解析出的真实 pi + 本 per-session 隔离实现运行并通过」。
	// per-session 隔离本身是无条件生产行为,开关只控制声明真伪。
	SessionIsolationVerified bool
	// WaitDelay 封顶「进程已退出但输出管道仍被(孤儿子)进程占用」时 Wait 的
	// 阻塞上限(exec.Cmd.WaitDelay;整树终止的兜底硬化)。零值取 defaultWaitDelay。
	WaitDelay time.Duration
	// Now 返回事件时间戳;nil 则用 time.Now。测试可注入固定时钟。
	Now func() time.Time
}

// defaultWaitDelay 是 WaitDelay 的默认兜底值。
const defaultWaitDelay = 5 * time.Second

// waitDelay 返回生效的 WaitDelay。
func (c Config) waitDelay() time.Duration {
	if c.WaitDelay > 0 {
		return c.WaitDelay
	}
	return defaultWaitDelay
}

// RuntimeSnapshot 是 runtime 身份/版本/能力的一次可记录快照(对齐
// runtimes/codex 的 AC3 缝):供上层把「跑的是哪个 pi、什么版本、声明了什么
// 能力」落成审计记录。
type RuntimeSnapshot struct {
	// Name 是 runtime 标识(DescriptorName)。
	Name string
	// AdapterVersion 是本适配器版本(DescriptorVersion)。
	AdapterVersion string
	// CLIVersion 是现场观察的 pi 版本(经 `--version`;已通过钉版闸,等于
	// 生效钉住值),非硬编码。
	CLIVersion string
	// BinaryPath 是 Config 解析出的 pi 可执行文件路径。
	BinaryPath string
	// Capabilities 是声明能力(字典序)。声明由 conformance 另行验证。
	Capabilities []agentruntime.Capability
}

// Runtime 是 PiRuntime 的 agentruntime.Runtime 实现。
type Runtime struct {
	cfg Config

	mu      sync.Mutex
	seq     int // 已开启会话计数(会话编号)
	binPath string
	cli     cliIdentity // `pi --version` 现场观察缓存(每 Runtime 至多探一次)
}

// New 构造 PiRuntime。调用方必须先过 Config.Validate(BinPath 必填等);
// 零值 Config 在 Start 处被 Validate 拒绝。
func New(cfg Config) *Runtime {
	return &Runtime{cfg: cfg}
}

// Descriptor 返回自述。Version 是适配器版本;CLI 版本属现场事实,由
// Snapshot/Start 观察并记录,不进 Descriptor(避免 Descriptor 触发进程)。
func (r *Runtime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{
		Name:         DescriptorName,
		Version:      DescriptorVersion,
		Capabilities: r.capabilities(),
	}
}

// Snapshot 确保二进制与钉版身份已观察,返回可记录的 RuntimeSnapshot。任一步
// 失败(BinPath 缺失、--version 不可观察、版本不匹配钉住值)显式报错 ——
// 钉版表面的身份不可观察即不可用(fail-closed,见包注释)。
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
// 基础声明为空集,带 ResumeOf(需要 session.resume)或 ModelProfile(需要
// model.select)的请求在协商处被 *CapabilityError 拒绝,不被隐式放行。
//
// Start 不 spawn pi 进程:一次 Execute = 一个进程;Start 只做四件有界的事 ——
// 校验配置(Validate)、解析二进制路径(缺失即刻报错)、执行钉版身份闸
// (--version 现场观察 + 钉住值比对,失败即拒绝;结果缓存于 Runtime 跨会话
// 共享)、建立会话状态与全新 per-session 隔离目录组(isolation.go)。
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

	// per-session 隔离目录组(isolation.go):每个新会话一组全新临时目录;
	// Close 时删除。无认证移交(pi 凭据只经环境变量,见包注释);创建失败 =
	// Start 显式失败(不留无边界的会话)。
	dirs, err := newSessionDirs()
	if err != nil {
		return nil, err
	}
	return &Session{
		runtime:   r,
		bin:       bin,
		index:     n,
		session:   agentruntime.SessionID(fmt.Sprintf("pi-session-%06d", n)),
		workspace: req.Workspace,
		dirs:      dirs,
		runs:      make(map[agentruntime.RunID]*run),
	}, nil
}

// ensureBinary 校验并缓存 BinPath(显式必填,无发现流程):路径必须指向
// 存在的普通文件,否则返回包裹 ErrBinaryMissing 的显式错误。
func (r *Runtime) ensureBinary() (string, error) {
	r.mu.Lock()
	if r.binPath != "" {
		bin := r.binPath
		r.mu.Unlock()
		return bin, nil
	}
	r.mu.Unlock()
	bin := strings.TrimSpace(r.cfg.BinPath)
	if bin == "" {
		return "", fmt.Errorf("%w: Config.BinPath 为空(pi 钉版面要求显式路径,不做隐式发现)", ErrBinaryMissing)
	}
	if err := requireFile(bin); err != nil {
		return "", err
	}
	r.mu.Lock()
	r.binPath = bin
	r.mu.Unlock()
	return bin, nil
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

// ensureCLIIdentity 现场执行 `<bin> --version`(观测形态:裸版本行
// "0.85.1",exit 0)并缓存解码结果。探针没跑起来、非零退出或输出不可解析
// 都显式报错且不缓存。本适配器在 Start/Snapshot 把它作为钉版闸的硬前提
// (fail-closed,见包注释;与 runtimes/codex 的尽力而为探针是有意偏离)。
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
		return cliIdentity{}, fmt.Errorf("pi: 读取 --version 失败: %w", err)
	}
	if res.ExitCode != 0 {
		return cliIdentity{}, fmt.Errorf("pi: --version 退出码 %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	c := parseCLIIdentity(res.Stdout)
	if !c.observed() {
		return cliIdentity{}, fmt.Errorf("pi: --version 输出不可解析: %q", truncate(strings.TrimSpace(res.Stdout), 200))
	}
	r.mu.Lock()
	r.cli = c
	r.mu.Unlock()
	return c, nil
}

// checkVersionPin 把现场观察的 CLI 身份与钉住版本比对:不一致即拒绝 ——
// 适配器的 argv/输出契约只对钉住版本有效,pi 迭代活跃,版本漂移必须显形。
func checkVersionPin(pinned string, cli cliIdentity) error {
	if pinned == "" {
		pinned = PinnedPiVersion
	}
	if cli.Version != pinned {
		return fmt.Errorf("%w: 观察 %q vs 钉住 %q:拒绝开启会话", ErrVersionPinMismatch, cli.Version, pinned)
	}
	return nil
}

// outputMode 返回生效的输出模式(Validate 已保证合法)。
func (r *Runtime) outputMode() string {
	if r.cfg.OutputMode == "" {
		return DefaultOutputMode
	}
	return r.cfg.OutputMode
}

// Validate 预检配置面(Invalid configuration fails before task execution 的
// 最早闸):BinPath 必填、OutputMode/ThinkingLevel 只接受观测枚举、
// Provider/Model 必须是可安全裸传 argv 的 token。不发起进程、不做任何观察
// (钉版身份闸仍在 Start)。
func (c Config) Validate() error {
	if strings.TrimSpace(c.BinPath) == "" {
		return fmt.Errorf("pi: Config.BinPath 为空(pi 钉版面要求显式路径,不做隐式发现)")
	}
	mode := c.OutputMode
	if mode != "" && !outputModes[mode] {
		return fmt.Errorf("pi: 非法 OutputMode %q(已知值: json | text,观测自 0.85.1 --mode 帮助;rpc 未观测不收)", mode)
	}
	if c.ThinkingLevel != "" && !thinkingLevels[c.ThinkingLevel] {
		return fmt.Errorf("pi: 非法 ThinkingLevel %q(pi 0.85.1 --thinking 观测枚举: off|minimal|low|medium|high|xhigh|max)", c.ThinkingLevel)
	}
	for _, f := range []struct {
		name  string
		value string
	}{
		{"Provider", c.Provider},
		{"Model", c.Model},
	} {
		if f.value != "" && !validConfigToken(f.value) {
			return fmt.Errorf("pi: Config.%s %q 含 argv 不安全字符(允许:字母数字与 . _ / - : +;拒绝可疑取值而不做转义)", f.name, f.value)
		}
	}
	return nil
}

// validConfigToken 报告 s 是否可安全裸写入 argv 值位:非空,且仅含字母数字与
// . _ / - :(观测形态 "deepseek"、"deepseek-v4-flash"、provider/id 形态帮助
// 文本、thinking 后缀 ":high" 帮助文本的字符集;空格/引号/控制字符一律拒绝,
// 不做转义器)。
func validConfigToken(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-', r == '/', r == ':', r == '+':
		default:
			return false
		}
	}
	return true
}

// requireFile 校验路径存在且是普通文件,否则返回包裹 ErrBinaryMissing 的
// 显式错误。
func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: Config.BinPath %q: %v", ErrBinaryMissing, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: Config.BinPath %q 是目录而非可执行文件", ErrBinaryMissing, path)
	}
	return nil
}

// Session 是 PiRuntime 的 agentruntime.Session 实现:一个 Tally 侧有界状态
// 对象。协议 SessionID 是适配器自造的会话编号(pi 在 json 流里自报一次性
// 会话 UUID,见 identity.go 的观察映射);同一会话的 Execute 串行化(execMu):
// 一次 Execute = 一个一次性进程,同一会话上的并发 turn 无观测依据,不做
// 并发假设。
type Session struct {
	runtime *Runtime
	bin     string
	index   int
	// session 是适配器自造的协议会话编号(pi-session-NNNNNN)。它标识 Tally
	// 协议层的会话边界,不是 pi 的内部会话句柄;pi 自报的 UUID 只作为观察值
	// 进 ObservedIdentity(observedSessionHandle),两者不混淆。
	session agentruntime.SessionID
	// workspace 是 Start 收到的源码执行目录:spawn cwd 钉定它(pi 无 cwd
	// 旗标,workspace = 进程 cwd,观测)。
	workspace string
	// dirs 是本会话专属的隔离目录组(isolation.go):Start 时全新创建,
	// Close 时删除;同一会话的全部 Execute 共享,跨会话永不共享。
	dirs *sessionDirs

	// execMu 串行化同会话的 pi 进程(一个会话同时最多一个进程)。
	execMu sync.Mutex

	mu     sync.Mutex
	runSeq int
	runs   map[agentruntime.RunID]*run
	closed bool
	// 下列是最近一次可解析 run 的观察快照(IdentityReporter 只读面)。
	observedSessionHandle string
	observedModel         string
	observedProvider      string
}

// run 是会话内一次运行(一个 pi 进程)的有界状态。
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

// Execute 同步执行一个 turn:spawn 一个 `pi -p` 进程,读取/解析 stdout 输出,
// 返回 runtime 的 AgentRunResult(claim,INV-AUTH-01)。取消(ctx 取消或
// Cancel)是正常终态:返回 RunCancelled 的结果而非 error;进程崩溃/非零退出/
// 契约不兑现(成功退出但无可解析回答)都归为 RunFailed claim,绝不把取消
// 或环境错误伪装成候选失败。
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

// Cancel 请求终止指定 run:整树终止底层进程(观测:Windows taskkill /T /F
// 干净终止 pi 进程树,无孤儿残留)。run 不存在返回 ErrRunNotFound;已到
// 终态幂等成功;关闭后的会话返回 ErrSessionClosed。
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
// (诚实能力集:pi 的 json NDJSON 流是 Execute 的内部读取面,不是可订阅的
// 会话事件流;CheckEvents 是协议统一前置闸)。会话关闭同样以该未声明拒绝
// 为先(对齐 runtimes/fake 顺序)。
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
// 拒绝后续操作,最后删除本会话的隔离目录组(尽力而为:Close 不等待
// in-flight 进程收敛,删除失败不作为 Close 错误;进程终止而未 Close 的
// 会话,目录留在系统临时目录由 OS 清理兜底 —— 与 runtimes/codex、
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
	if dirs != nil {
		_ = dirs.Close()
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

// RunProcess 返回 in-flight run 的底层 pi 进程(诊断/测试注入面,例如 PRD
// §35 crash 场景的外部 kill 注入;返回的是 pi 进程本身,不是其子进程树)。
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
