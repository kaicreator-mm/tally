package failure

import (
	"context"
	"strings"
	"time"

	"tally/internal/airuntime"
	"tally/internal/command"
	"tally/internal/core"
)

// Runner 是 failure 依赖的执行抽象:跑复现命令 / 构建目标。
//
// 在使用方定义(见 docs/04-code-standards.md「接口」):真实 docker/ssh 执行器是
// runner 包的事(T01),这里只留 port;单测注入确定性 fake,离线可跑。
type Runner interface {
	Run(ctx context.Context, spec RunSpec) (RunResult, error)
}

// RunSpec 是一次执行。Fix 非空 = 应用该修复后再跑(G2/G3);为空 = 基线复现(G1)。
// Command 是结构化命令(PRD §21.6:不用 shell string + strings.Fields 表示):
// 跑什么(Argv)、在哪跑(Cwd)、带什么环境(Env)全在 spec 里,执行器不拆不改。
// Target 非空 = 跑指定构建目标(G3 检查其余目标):此时 Command 已由闸门换成该
// 目标自己的构建命令,Target 只是目标名标签,不携带也不表示任何可执行命令。
type RunSpec struct {
	Command command.CommandSpec
	Fix     []core.FileChange
	Target  string
}

// RunResult 是一次执行结果。Green=true 为通过;Output 必须已脱敏(真实执行器的责任)。
type RunResult struct {
	Green  bool
	Output string
}

// KnowledgeBase 是 failure 对知识库的期望(使用方定义,业务包互不导入):
// 精确命中直接给修复(零 LLM),模糊命中只给线索,写入必须带验证证据。
// 实现是 internal/knowledge.Store,由上层(web/jobs)注入 —— 两边签名同形,
// 在各自包内用镜像接口守住形状。
type KnowledgeBase interface {
	LookupExact(ctx context.Context, fp core.Fingerprint, current core.VersionCombo) (core.FixHit, bool, error)
	LookupFuzzy(ctx context.Context, fp core.Fingerprint, current core.VersionCombo) ([]core.Lead, error)
	Store(ctx context.Context, fp core.Fingerprint, fix core.Fix, proof core.VerificationProof) (string, error)
}

// SpecDefectSink 接收「需求本身有问题」的出口,回写 requirement 并触发失效传播
// (PRD 第四部分规则 ④、第六部分)。实现由上层把 requirement 的能力注入;
// failure 只定义接口,测试用 fake。
type SpecDefectSink interface {
	// ReportSpecDefect 校验条款存在后记录。条款不存在 → 返回错误 ——
	// 这是确定性闸门:LLM 只能提案,编不出不存在的条款(铁律 1)。
	ReportSpecDefect(ctx context.Context, projectID, clauseID, ambiguity string) error
}

// OtherTarget 是一个其余构建目标(G3:修复不得影响)的结构化描述:
// Target 是目标名标签;Command 是该目标的构建命令(PRD §21.6 结构化,非 shell 字符串)。
// JSON 字段名与 failure_event.other_target_commands 的持久化合同一致。
type OtherTarget struct {
	Target  string              `json:"target"`
	Command command.CommandSpec `json:"command"`
}

// Event 是一次失败诊断的输入:过了日志流水线的 failure_event 内容,
// 加上验证所需的结构化复现上下文(复现命令与其余构建目标,由队列 payload 给)。
type Event struct {
	ProjectID        string
	Source           string // build / deploy / environment
	BuildRunID       int64  // 三个来源外键恰好一个非零
	DeployRunID      int64
	EnvironmentRunID int64
	RawLogRef        string
	RawLogBytes      int64
	Excerpt          string // 抽取后的关键片段(已脱敏)
	Normalized       string
	PipelineVersion  string
	Exact            core.Fingerprint
	Fuzzy            core.Fingerprint
	Combo            core.VersionCombo
	OccurredAt       time.Time

	// Repro 是复现命令(重跑那次构建/部署),结构化 CommandSpec(PRD §21.6),
	// 工作目录在 Cwd 里。Argv 为空 = 没有可复现命令(Handle 按 D3 跳过 G1/G2)。
	Repro        command.CommandSpec
	OtherTargets []OtherTarget // 其余构建目标(G3:不受影响),各带自己的构建命令
}

// LegacyReproCommand 从 0016 前存量 failure_event 行的文本两列(workdir/repro)
// 重建结构化复现命令 —— 读取侧的兼容装载器,也是本包唯一允许 strings.Fields 的地方:
// 旧 worker 正是 strings.Fields(repro) + Dir(workdir) 执行的,重建按同一旧执行语义
// (合同见 internal/store/queries/failure_event.sql 头注)。
// repro 为空(部署/环境失败本无可复现命令)→ 返回 Argv 为空的零值 spec,由调用方按
// 「缺复现上下文」处理(D3),不得假装验证过。
func LegacyReproCommand(workdir, repro string) command.CommandSpec {
	return command.CommandSpec{Argv: strings.Fields(repro), Cwd: workdir}
}

// Conclusion 是一轮诊断的结论。
type Conclusion struct {
	Kind         OutcomeKind // 出口;"" = 本轮无出口(候选没过验证,等下一轮)
	FixRef       string      // fixed 时修复在 git 里的引用(提交哈希)
	Fix          *core.Fix   // 修复内容(精确命中 / 本轮验证通过时给出)
	Archived     bool        // 三轮预算用尽:暂无自动修复(PRD 第四部分规则 ⑤)
	Paused       bool        // 修复改到应用源码:暂停,等人确认
	SkippedRepro bool        // 缺复现上下文(Repro 无 Argv):跳过 G1/G2,本轮不提不验(D3)
}

// Deps 汇集 Handle 需要的全部依赖。全部注入,业务包互不导入、协作由上层编排:
// 推理只提案(经 airuntime 边界,P0-11);Runner 是执行 port;KnowledgeBase 是
// knowledge.Store;StatStore 包装 fingerprint_stat;SpecSink 注入 requirement 的回写能力。
type Deps struct {
	Executor airuntime.Executor // 通用推理只提案(INV-1);判定在四道闸
	Runner   Runner
	KB       KnowledgeBase
	Stats    StatStore
	SpecSink SpecDefectSink   // 可不注入:未注入时 spec_defect 出口不可达
	Now      func() time.Time // 可不注入:缺省 time.Now
}
