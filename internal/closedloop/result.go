// result.go 实现 P2-02 的尝试谱系与终态结果载体(PRD §17-18;ADR-0001 §2:
// 尝试身份对 (task, attempt_no) 幂等;谱系 append-only)。
//
// Result / Attempt / Basis 只由 Engine 构造,调用方只读:终态结果不存在
// 任何改写路径;升级与取消都携带完整的已完成尝试谱系,半途取消的尝试
// 不入谱系(取消不产生 verdict,ADR-0001 §3)。
package closedloop

import (
	"fmt"
	"strings"

	"tally/internal/changeset"
)

// AttemptNo 是外层尝试序号(1 起):1 = 首次执行,≥2 = 第 (n-1) 次修复。
// 它与内层迭代计数完全无关(ADR-0001 §1:两个预算互不相通)。
type AttemptNo int

// IsValid 报告 n 是否是合法序号(≥1)。
func (n AttemptNo) IsValid() bool { return n >= 1 }

// IsRepair 报告 n 是否是修复尝试(≥2;ADR-0001 §2:修复按序号计)。
func (n AttemptNo) IsRepair() bool { return n >= 2 }

// RunID 标识一次尝试的 CodegenRun。由 (task, AttemptNo) 确定性推导
// (RunIDFor):同任务同序号恒得同 id —— 这就是 ADR-0001 §2 的尝试幂等键,
// 崩溃后的修复尝试按键恢复,而不是复用死尝试的 ChangeSet。
type RunID string

// RunIDFor 从 (task, attempt) 确定性推导该次尝试的 Run 身份。
// 纯函数:无时钟、无随机;不同 (task, attempt) 推导出不同 id。
func RunIDFor(task string, no AttemptNo) RunID {
	return RunID(fmt.Sprintf("%s/attempt/%03d", task, int(no)))
}

// Attempt 是一次已完成尝试的谱系记录(append-only;构造后不可变):
// 它属于且只属于一个 (task, AttemptNo),携带该次尝试的 Run 身份、父链
// 与(固定了候选时)候选身份。
type Attempt struct {
	// No 是尝试序号(1 起)。
	No AttemptNo
	// RunID 是本次尝试的 CodegenRun 身份(RunIDFor 推导,幂等键)。
	RunID RunID
	// ParentRun 是紧邻前一次尝试的 Run 身份;首次尝试为空。
	// Run 链因此是父链接的(ADR-0001 §2:attempt-numbered, parent-linked)。
	ParentRun RunID
	// ChangeSet 是本次尝试固定的不可变候选快照身份;没有固定候选时为空。
	ChangeSet changeset.ChangeSetID
	// Supersedes 是本次修复候选所取代的前一候选身份(reason=rework 的
	// 取代事实同步经 LineageSink 追加);非修复尝试恒为空。
	Supersedes changeset.ChangeSetID
}

// Clone 返回谱系切片的深拷贝(Attempt 全为值字段,拷贝切片即可)。
func CloneAttempts(in []Attempt) []Attempt {
	if in == nil {
		return nil
	}
	out := make([]Attempt, len(in))
	copy(out, in)
	return out
}

// Basis 是 Accepted 终态的交付依据(INV:每个 VERIFIED-like 结果必须携带
// basis)。三个字段共同把「可接受」绑定到:哪一个候选、由哪一次独立验证
// 运行确认、属于哪一次尝试 —— 缺一不可,Engine 在构造处机械强制。
type Basis struct {
	// ChangeSet 是被接受的不可变候选快照身份(verdict 绑定对象)。
	ChangeSet changeset.ChangeSetID
	// VerifyRun 是确认该候选的独立验证运行引用(由验证服务如实报告;
	// Engine 不发明、不补齐)。
	VerifyRun string
	// Attempt 是产出该候选的尝试序号。
	Attempt AttemptNo
}

// EscalationReason 是升级原因的类型化词表。Engine 内部 fail-safe 落点使用
// 下方常量;Router 裁决的升级原因按原样透传(非空即可,P2-03 的分类词表
// 归 P2-03 所有,本骨架不解释)。
type EscalationReason string

// Engine 内部确定性 fail-safe 升级原因(全部落在 Engine 的自有裁决点,
// 不经过 Router)。
const (
	// EscalationNoCandidate:尝试终结但未固定任何候选 —— 无可验证物,
	// 外层循环不能对空候选验证或修复(PRD §17.1 链条的前提断裂)。
	EscalationNoCandidate EscalationReason = "no_candidate"
	// EscalationRouterUnavailable:Router 缺席、出错或给出非法裁决 ——
	// 无法裁决的失败绝不默认修复,一律 fail-safe 升级(INV-V14-09)。
	EscalationRouterUnavailable EscalationReason = "router_unavailable"
	// EscalationRepairUnavailable:Router 裁决修复,但取代事实出口或修复
	// 预算端口缺席 —— 修复在结构上不可进行(没有取代事实的修复非法,
	// ADR-0001 §2),fail-safe 升级。
	EscalationRepairUnavailable EscalationReason = "repair_unavailable"
	// EscalationRepairExhausted:修复预算拒绝下一次修复尝试(PRD §17.2:
	// 超限即 Escalate;预算语义归 RepairBudget 实现,P2-07)。
	EscalationRepairExhausted EscalationReason = "repair_exhausted"
	// EscalationInconclusive:验证器无法给出可信 verdict(ERROR 语义,
	// ADR-0001 §3):永远不算候选失败、不进修复、不耗预算;infra 复验
	// 路径归 P2-03/P2-08,骨架落点为 fail-safe 升级。
	EscalationInconclusive EscalationReason = "verification_inconclusive"
	// EscalationIdentityReuse:某次尝试返回的候选身份已在谱系中出现 ——
	// ADR-0001 §2(新候选身份必须全新)被违背,绝不接受也绝不修复,
	// fail-safe 升级。
	EscalationIdentityReuse EscalationReason = "candidate_identity_reuse"
	// EscalationSupersessionFailed:取代事实无法追加 —— 谱系断裂的修复
	// 不成立,fail-safe 升级(取代关系 append-only,无回滚补写)。
	EscalationSupersessionFailed EscalationReason = "supersession_unavailable"
	// EscalationInfrastructure:端口返回基础设施错误(取消除外)。端口
	// 错误绝不伪装成候选失败,也绝不静默重试(AGENTS 信任不变量);
	// infra/provider 路由由 P2-03/P2-08 细化。
	EscalationInfrastructure EscalationReason = "infrastructure_failure"
	// EscalationInvalidOutcome:端口返回了循环无法信任的结果值(未知
	// verdict、verified 却无验证运行引用等)—— 不经 Router 解释,
	// fail-safe 升级。
	EscalationInvalidOutcome EscalationReason = "invalid_outcome"
	// EscalationAssuranceInsufficient(P1-06,DL-07/A-03):编译权威的必需
	// 保证档未达成、存在未匹配风险信号或被豁免条款 —— FinalVerified 只是
	// 验证事实,不足以授权普通 Accept。fail-safe 升级交人/政策裁决:绝不
	// 静默保持 VERIFIED、绝不自动豁免、也绝不伪装成候选失败进入修复。
	EscalationAssuranceInsufficient EscalationReason = "assurance_insufficient"
	// EscalationOutcomeUnavailable(P6-01):accepted outcome 事实出口写入
	// 失败 —— 一次没有耐久 accepted outcome 的接受等于丢掉了事后无法补采的
	// §23 事实。fail-safe 升级:绝不静默接受、绝不静默丢事件、也绝不伪装成
	// 候选缺陷进入修复(接受裁决本身已通过,这不是候选失败)。
	EscalationOutcomeUnavailable EscalationReason = "outcome_unavailable"
)

// IsValid 报告原因非空(类型化词表的最低门槛;语义归产出方)。
func (r EscalationReason) IsValid() bool { return strings.TrimSpace(string(r)) != "" }

// Escalation 是 Escalated 终态的类型化升级事实(PRD §18.3:升级必须说明
// 发生了什么;Detail 是人类可读摘要,结构性不含任何 held-out 内容 ——
// 循环内本来就不存在失败细节通道)。
type Escalation struct {
	// Reason 是类型化升级原因(非空)。
	Reason EscalationReason
	// Detail 是人类可读补充(可空;仅描述性,不承载验证原始输出)。
	Detail string
}

// Result 是一次外层闭环 Run 的终态结果:恰好一个终态相位 + 完整已完成
// 尝试谱系 + 终态对应的依据/升级载荷。由 Engine 在收束处一次性构造。
type Result struct {
	// TaskID 是本次闭环的任务 id。
	TaskID string
	// Phase 是终态(Accepted / Escalated / Cancelled;调用方可用
	// Phase.Terminal() 复核)。
	Phase Phase
	// Attempts 是已完成的尝试谱系(执行顺序;append-only 的投影)。
	// 取消时半途尝试不入谱系。
	Attempts []Attempt
	// Basis 仅 Phase == PhaseAccepted 时非 nil(Accepted 必带依据)。
	Basis *Basis
	// Escalation 仅 Phase == PhaseEscalated 时非 nil。
	Escalation *Escalation
}
