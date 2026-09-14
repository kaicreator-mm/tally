// ports.go 定义外层闭环消费的窄端口(小接口由消费方定义,AGENTS Go 规则)。
//
// 端口面刻意最小:
//
//   - Executor:把 v1.3 codegen 内循环作为不透明服务注入。一次调用 = 一次
//     全新的 CodegenRun(新一次性 worktree,ADR-0001 §2);内层迭代对循环
//     不可见 —— 请求里只有尝试谱系身份,没有任何迭代/重试词汇,本包不
//     import internal/policy(ExecutionPolicy.Retry 结构性不可达,AC3);
//   - Verifier:把独立验证作为服务注入。一次调用 = 对恰好一个候选的一次
//     全新独立验证;
//   - Router:类型化失败路由缝(P2-03 提供分类封闭集与实现);循环只执行
//     裁决,不解释失败内容,入参结构性不含任何 held-out 细节(仅 id 与
//     verdict 词表,ADR-0001 §4 的通道纪律在类型层成立);
//   - RepairBudget:外层修复预算缝(P2-07 提供策略实现);循环永不读内层
//     迭代预算,预算缺席 = 永不修复(fail-safe);
//   - LineageSink:取代事实的 append-only 出口(P2-05/P2-08 提供持久化
//     适配);刻意没有更新/删除方法。
//
// 全部端口方法都显式携带 context.Context;取消沿 ctx 传播,取消的调用
// 产出不构成 verdict、不构成候选事实。
package closedloop

import (
	"context"

	"tally/internal/changeset"
	"tally/internal/verification"
)

// ExecuteRequest 是一次外层尝试交给 Executor 的请求:携带尝试谱系身份,
// 修复尝试(≥2)另携带继续起点 —— 冻结的前候选 RepairBase(仅候选内容,
// 不含任何合同/验收材料)。合同/工作区/策略等执行材料由适配器在构造时
// 绑定(组合根职责,P2-08),不经循环转手 —— 循环对内层执行保持不透明
// (AC3)。
type ExecuteRequest struct {
	// TaskID 是闭环任务 id。
	TaskID string
	// Attempt 是本次尝试序号(1 = 首次执行;≥2 = 修复)。
	Attempt AttemptNo
	// RunID 是本次尝试的 CodegenRun 身份(RunIDFor 推导的幂等键);
	// 适配器必须用它作为该次 CodegenRun 的身份(ADR-0001 §2)。
	RunID RunID
	// ParentRun 是紧邻前一次尝试的 Run 身份(首次尝试为空);Run 链
	// 父链接,适配器据此落 escalated_from 类似的谱系事实。
	ParentRun RunID
	// RepairBase 是修复尝试(Attempt ≥ 2)继续的冻结前候选(ADR-0001 §2:
	// repair base = 最新未被取代候选的 ChangeSet 内容及其工作区基线);
	// 首次尝试恒为 nil。适配器职责(ADR-0001 §2,P2-05):
	//   - 在**全新**一次性 worktree 内以 RepairBase 的 base revision 为基线
	//     检出,并把 RepairBase 的内容**确定性物化**为修复起点(internal/
	//     workspace 的 RunSources.Materialize);前候选与其工作区绝不被改写,
	//     owner 主 worktree 永远不是执行沙箱;
	//   - RepairBase 只携带候选内容,不携带任何验收/证据/保证材料 —— 修复
	//     尝试永远在原始那次冻结契约下执行与复验,契约 pin 逐字不变
	//     (INV-02/03:acceptance / required evidence / assurance 不可由
	//     修复改写)。
	RepairBase *changeset.ChangeSet
	// Feedback 是修复尝试交给实现权威的**唯一失败材料通道**(P2-06,ADR-0001
	// §4 的 allowlist 投影,P2-04 类型);首次尝试恒为 nil。投影类型上不存在
	// 原始 held-out 内容/路径/断言/期望值/摘要的承载字段(INV-V14-07);Engine
	// 只在反馈经 FeedbackSource 安全构造成功后才开始修复 Run,构造失败落
	// feedback_unavailable 升级 —— 修复编码指令里没有第二条失败材料来路。
	Feedback *RepairFeedback
}

// ExecuteOutcome 是一次已完成尝试的领域结果(值,不是 error):适配器只在
// 进程/基础设施意义上失败时才返回 error;「内层收束但没有可固定候选」是
// 正常领域落点(Candidate 为 nil),由循环裁决为升级。
type ExecuteOutcome struct {
	// Candidate 是本次尝试固定的不可变候选快照;nil = 本次尝试没有固定
	// 候选(该次尝试的半途工作由适配器随一次性 worktree 丢弃)。
	Candidate *changeset.ChangeSet
}

// Executor 是注入的 codegen 服务端口(v1.3 内循环保持不透明)。
//
// 契约(适配器责任,ADR-0001 §2):
//   - 每次调用开始一个**全新** CodegenRun,在**全新**一次性 worktree 内
//     执行(owner 主 worktree 永远不是候选执行沙箱);
//   - 每次调用至多固定**一个全新**候选快照;候选身份必须从未出现在本
//     任务的谱系中(Engine 机械强制,重复即 fail-safe 升级);
//   - 内层迭代预算由适配器自持,循环不可见。
type Executor interface {
	ExecuteCandidate(ctx context.Context, req ExecuteRequest) (ExecuteOutcome, error)
}

// Verdict 是独立验证的 verdict 词表(ADR-0001 §3:FAIL 与 ERROR 端到端
// 不同,永不合并)。
type Verdict string

const (
	// VerdictVerified:可信正面 verdict —— 全部必需证据/门禁通过。
	VerdictVerified Verdict = "verified"
	// VerdictFailed:可信负面 verdict —— 验证器给出了可信的失败判定;
	// 唯一允许进入 Router 裁决的 verdict(只有 FAIL 分类 repairable 才
	// 进修复,PRD §17.4)。
	VerdictFailed Verdict = "failed"
	// VerdictInconclusive:验证器无法给出可信 verdict(ERROR 语义:超时/
	// 网络/runner 崩溃/存储不可用)。结构上不可 accept、不可 repair、
	// 不算候选失败、不耗修复预算(ADR-0001 §3)。
	VerdictInconclusive Verdict = "inconclusive"
)

// IsValid 报告 v 是否是词表内的合法 verdict。
func (v Verdict) IsValid() bool {
	switch v {
	case VerdictVerified, VerdictFailed, VerdictInconclusive:
		return true
	default:
		return false
	}
}

// VerifyRequest 是一次独立验证交给 Verifier 的请求:精确绑定被验证候选。
type VerifyRequest struct {
	// TaskID 是闭环任务 id。
	TaskID string
	// Attempt 是产出该候选的尝试序号。
	Attempt AttemptNo
	// ChangeSet 是被验证的不可变候选快照身份(verdict 绑定对象,
	// INV-V14-06:一个 verdict 只绑一个精确 ChangeSet)。
	ChangeSet changeset.ChangeSetID
	// Snapshot 是该候选的冻结内容快照(P2-08 组合):Engine 只透传执行时
	// 固化的同一 *changeset.ChangeSet(类型上不可改写),验证适配器按它
	// 物化一次性验证 workspace 并核对 Snapshot.ID() == ChangeSet —— 身份
	// 与内容不一致时拒绝验证,绝不验证未知内容。
	Snapshot *changeset.ChangeSet
}

// VerifyOutcome 是一次独立验证的领域结果(值,不是 error):验证结论本身
// (failed/inconclusive)不是基础设施失败,必须作为事实返回。
type VerifyOutcome struct {
	// Verdict 是本轮验证结论(词表内三值之一)。
	Verdict Verdict
	// RunRef 是验证服务为本轮验证记录的运行引用(basis 来源)。Engine
	// 不发明运行引用:VerdictVerified 而 RunRef 为空 = 不可信结果,
	// fail-safe 升级,绝不伪造 basis。
	RunRef string
	// Coverage 是验证编排为该候选记录的功能覆盖观察(P1-06 验收守卫的
	// 事实面:每条验收条款的覆盖与满足 binding id;FinalOutcome.Functional.
	// Acceptances 的原样投影)。它只是观察事实 —— 接受裁决归 Engine 的
	// AcceptGuard,验证适配器不裁验收。缺席 = 无观察事实,守卫按 L0
	// fail-safe 处理,绝不把「没有观察」当作「已满足」。
	Coverage []verification.AcceptanceCoverage
}

// Verifier 是注入的独立验证服务端口(v1.4 验证权威,外部于实现权威)。
//
// 契约:每次调用对恰好一个候选做一次全新独立验证(冻结验收在执行前已
// 固定,修复尝试用与原始验证相同的冻结契约复验,绝不放宽 —— ADR-0001
// 后果节);验证事实的落库由适配器负责(append-only)。
type Verifier interface {
	VerifyCandidate(ctx context.Context, req VerifyRequest) (VerifyOutcome, error)
}

// VerifyFailure 是交给 Router 的类型化失败面:只有 id 与 verdict 词表,
// 结构性不含任何 held-out 内容/路径/断言/原始输出(ADR-0001 §4 的通道
// 纪律;安全反馈投影归 P2-04)。
type VerifyFailure struct {
	// TaskID 是闭环任务 id。
	TaskID string
	// Attempt 是产出失败候选的尝试序号。
	Attempt AttemptNo
	// ChangeSet 是未通过验证的候选快照身份。
	ChangeSet changeset.ChangeSetID
	// RunRef 是给出 FAIL 判定的验证运行引用(失败谱系可追溯)。
	RunRef string
}

// Route 是循环可执行的裁决动作词表(消费方词汇;分类封闭集与判定逻辑
// 归 P2-03,本骨架只定义「循环会做什么」)。
type Route uint8

const (
	// RouteRepair:进入外层修复 —— 新 Run + 新一次性 worktree + 新不可变
	// ChangeSet + reason=rework 取代事实 + 全新独立验证;消耗修复预算
	// (预算拒绝则显式升级,ADR-0001 §1/§6)。
	RouteRepair Route = iota
	// RouteEscalate:终止循环,带类型化原因升级给人。
	RouteEscalate
)

// String 返回路由的规范名(诊断/测试用途)。
func (r Route) String() string {
	switch r {
	case RouteRepair:
		return "repair"
	case RouteEscalate:
		return "escalate"
	default:
		return "Route(unknown)"
	}
}

// Ruling 是 Router 对一次可信 FAIL 的裁决。Route == RouteEscalate 时
// Reason 必须非空(原样透传为升级原因);非法裁决按无法裁决处理,
// Engine fail-safe 升级,绝不猜测。
type Ruling struct {
	// Route 是裁决动作。
	Route Route
	// Reason 是 Route == RouteEscalate 时的升级原因(非空;词表归
	// Router 实现方,循环不解释)。
	Reason EscalationReason
}

// Router 是类型化失败路由缝(实现归 P2-03)。循环只在候选收到可信 FAIL
// 时咨询 Router;Router 缺席/出错/非法裁决的落点一律是 fail-safe 升级
// (INV-V14-09:unknown/unmatched 绝不静默选最弱路径)。
type Router interface {
	RouteFailure(ctx context.Context, failure VerifyFailure) (Ruling, error)
}

// RepairBudget 是外层修复预算缝(策略实现归 P2-07;默认 2 次,ADR-0001
// §6:预算默认是策略值,不是架构常量)。Engine 只在裁决为修复后、开始
// 新 Run 前咨询一次;no 一定是 ≥2 的下一次修复序号。
//
// 与内层迭代预算互不相通:本接口的实现不得读取 ExecutionPolicy.Retry,
// 循环侧也永不读(结构性保证:本包不 import internal/policy)。
type RepairBudget interface {
	// CanRepair 报告第 no 次修复尝试(恒 ≥2)是否可以开始。
	CanRepair(no AttemptNo) bool
}

// LineageSink 是取代事实的 append-only 出口(持久化适配归 P2-05/P2-08)。
// 刻意只有追加方法:取代关系只增不改(ADR-0001 §2;INV-V14-06/16),
// 不存在任何改写或回滚路径。
type LineageSink interface {
	// RecordSuperseded 追加一条取代事实(修复候选取代前一候选,
	// reason=rework)。
	RecordSuperseded(ctx context.Context, fact changeset.SupersededFact) error
}
