package measurement

// v14_metrics.go(P3-06)是 PRD §31 闭环产品指标的纯读侧投影:从既有事实
// (runstore 的 Run / supersession 事实、approval 的 owner 决定证据)与
// P3-05 的 append-only OutcomeEvent 事件流(internal/outcome)派生
//
//   - Hands-off Completion Rate 输入(§31.1);
//   - Time to Accepted Change(§31.2);
//   - Human Intervention per Accepted Change(§31.3 离散事件);
//   - repair attempts(决策锁 8/23:每个外层 Repair = 新 Run + 新 ChangeSet
//     + rework 取代事实 —— 取代事实因此是 repair attempt 的事实形态);
//   - 14-day Rework(§31.4 fast guardrail)与 slow guardrail 计数(§31.5:
//     reverted / hotfixed / regression_found 事件)。
//
// 投影纪律(与 report.go 同一边界,决策锁 14 / INV-STATE-01):
//   - 派生只读:同一些事实 → 同一些数字,确定、无写路径、不落任何新状态
//     权威 —— 指标不缓存、不回写,重算永远可行。
//   - 本包不 import runstore(runstore 反向依赖本包)与 approval(其依赖
//     internal/evidence):Run / 决定证据 / 取代事实由调用方按本文件的 typed
//     输入投影映射,映射责任在调用方;枚举词表与域包逐字一致。
//   - 诚实缺口(AC1/AC3):输入事实尚不存在于系统任何处的介入种类,以
//     FactSourceCaptured=false + 命名缺口显式呈现「不可用」;缺失绝不是 0,
//     不被折叠进任何已捕获计数,也不伪造样本。
//   - Human 查看 Diff 不是介入也不是失败(§31.1):diff review 计时
//     (EffortDiffReview)与本投影的介入输入完全正交;不带豁免的 approve /
//     reject 决定是评审裁决,不是 §31.3 介入(AC2)。

import (
	"fmt"
	"strings"
	"time"

	"tally/internal/outcome"
)

// V14ReworkWindow 是 §31.4 fast guardrail 的显式时间窗:验收后 14 天内
// 的再修改计为 rework。typed 常量,窗口不从配置或时钟来。
const V14ReworkWindow = 14 * 24 * time.Hour

// V14ReworkAttributionNote 是 §31.4 的归因局限声明(AC4:必须文档化)。
// 渲染方应把它与 rework 读数一并呈现。
const V14ReworkAttributionNote = "14-day rework 把验收后窗口内的任何再修改" +
	"(返工/规格缺陷取代、回归、回退、热修)都计为 rework 信号;outcome 事件" +
	"与取代事实都不携带根因字段,「同一根本原因」无法从现有事实验证 —— 该" +
	"读数是归因上限(只知再修改,不知是否同因),不得当作根因成立率呈现。"

// ---- V14RunStatus:runstore 生命词表的镜像(映射责任在调用方) ----

// V14RunStatus 是 codegen_run 生命周期词表的镜像(与 runstore 的 CHECK 取值
// 逐字一致:running|completed|failed|escalated|cancelled)。本包不解释语义,
// 只做合法性校验;映射责任在调用方。
type V14RunStatus string

const (
	V14RunRunning   V14RunStatus = "running"
	V14RunCompleted V14RunStatus = "completed"
	V14RunFailed    V14RunStatus = "failed"
	V14RunEscalated V14RunStatus = "escalated"
	V14RunCancelled V14RunStatus = "cancelled"
)

// IsValid 报告该值是否在镜像词表内。
func (s V14RunStatus) IsValid() bool {
	switch s {
	case V14RunRunning, V14RunCompleted, V14RunFailed, V14RunEscalated, V14RunCancelled:
		return true
	}
	return false
}

// V14ApprovalDecision 是 owner 决定方向词表的镜像(与 internal/approval 的
// Decision 逐字一致:approve|reject)。映射责任在调用方。
type V14ApprovalDecision string

const (
	V14DecisionApprove V14ApprovalDecision = "approve"
	V14DecisionReject  V14ApprovalDecision = "reject"
)

// IsValid 报告该值是否在镜像词表内。
func (d V14ApprovalDecision) IsValid() bool {
	switch d {
	case V14DecisionApprove, V14DecisionReject:
		return true
	}
	return false
}

// V14SupersedeReason 是取代原因词表的镜像(与 internal/changeset 的
// SupersedeReason 逐字一致:escalation|rework|spec_defect)。映射责任在调用方。
type V14SupersedeReason string

const (
	V14SupersedeEscalation V14SupersedeReason = "escalation"
	V14SupersedeRework     V14SupersedeReason = "rework"
	V14SupersedeSpecDefect V14SupersedeReason = "spec_defect"
)

// IsValid 报告该值是否在镜像词表内。
func (r V14SupersedeReason) IsValid() bool {
	switch r {
	case V14SupersedeEscalation, V14SupersedeRework, V14SupersedeSpecDefect:
		return true
	}
	return false
}

// ---- §31.3 介入种类(封闭词表) ----

// V14InterventionKind 是 PRD §31.3 的离散人工介入种类(封闭词表,AC3:种类
// 互相可区分,绝不合并成一个「介入数」)。字符串值即机器词面,只增不改。
type V14InterventionKind string

const (
	V14InterventionClarification        V14InterventionKind = "clarification"
	V14InterventionManualRepair         V14InterventionKind = "manual_repair"
	V14InterventionManualRerun          V14InterventionKind = "manual_rerun"
	V14InterventionWaiver               V14InterventionKind = "waiver"
	V14InterventionTradeOffDecision     V14InterventionKind = "trade_off_decision"
	V14InterventionForcedRetry          V14InterventionKind = "forced_retry"
	V14InterventionDirectCodeCorrection V14InterventionKind = "direct_code_correction"
)

// v14InterventionKinds 是 §31.3 全词表的固定顺序(报告/测试共用)。
var v14InterventionKinds = []V14InterventionKind{
	V14InterventionClarification,
	V14InterventionManualRepair,
	V14InterventionManualRerun,
	V14InterventionWaiver,
	V14InterventionTradeOffDecision,
	V14InterventionForcedRetry,
	V14InterventionDirectCodeCorrection,
}

// IsValid 报告该值是否是 §31.3 定义的介入种类之一。
func (k V14InterventionKind) IsValid() bool {
	for _, known := range v14InterventionKinds {
		if k == known {
			return true
		}
	}
	return false
}

// v14BreaksHandsOff 报告该介入种类是否属于 §31.1 列举的四类「手工作业失败
// 模式」(手工修代码 / 复制失败 / 重新发 prompt / 重启 verification),即
// 是否打破 hands-off:
//   - manual_repair / manual_rerun / forced_retry / direct_code_correction
//     打破(修代码、重启验证、重发指令);
//   - waiver / trade_off_decision 不打破:§31.1 明说 READY_FOR_HUMAN_DECISION
//     是合法终点,人的裁决是决策不是手工作业;
//   - clarification 不打破:它是意图/信息的补全,不是 §31.1 列举的四类之一
//     (但仍是 §31.3 的人工介入,照常计数)。
//   - Human 查看 Diff(以及随后不带豁免的 approve/reject)从一开始就不是
//     介入种类,永不进入本投影(AC2)。
func v14BreaksHandsOff(k V14InterventionKind) bool {
	switch k {
	case V14InterventionManualRepair, V14InterventionManualRerun,
		V14InterventionForcedRetry, V14InterventionDirectCodeCorrection:
		return true
	case V14InterventionClarification, V14InterventionWaiver, V14InterventionTradeOffDecision:
		return false
	default:
		return false
	}
}

// ---- 命名缺口(结构性:输入事实尚不存在于系统任何处) ----

// V14DataGapKind 是结构性缺口的封闭词表:每个值对应一个「指标输入事实尚未
// 被任何采集面捕获」的命名事实。缺口是显式不可用,不是 0(AC1/AC3)。
type V14DataGapKind string

const (
	// GapRunInitiatorNotCaptured:run 无发起方/触发源事实,人工发起与闭环
	// 自动发起的 run 不可区分(manual_repair / manual_rerun / forced_retry
	// 三类介入因此不可用)。
	GapRunInitiatorNotCaptured V14DataGapKind = "run_initiator_not_captured"
	// GapClarificationNotCaptured:intake 澄清请求/回答只有提案流的内存形态,
	// 没有持久事实面。
	GapClarificationNotCaptured V14DataGapKind = "clarification_not_captured"
	// GapTradeOffDecisionNotCaptured:审批证据只承载 approve|reject 与 waiver,
	// 权衡裁决没有专属事实类型。
	GapTradeOffDecisionNotCaptured V14DataGapKind = "trade_off_decision_not_captured"
	// GapProductionIncidentNotCaptured:§31.5 的 production incident 关联没有
	// 事实类型;最近似的已捕获形态是 regression_found 事件。
	GapProductionIncidentNotCaptured V14DataGapKind = "production_incident_not_captured"
)

// V14DataGap 是一条结构性缺口的显式声明。
type V14DataGap struct {
	Kind V14DataGapKind
	// Detail 是缺口的事实性说明(什么没采、影响哪些读数)。
	Detail string
	// Affects 是受影响的 §31.3 介入种类(slow guardrail 缺口为空)。
	Affects []V14InterventionKind
}

// v14InterventionGap 返回某介入种类的命名缺口;第二返回值 false 表示该种类
// 的事实面已在账(waiver / direct_code_correction)。
func v14InterventionGap(k V14InterventionKind) (V14DataGapKind, bool) {
	switch k {
	case V14InterventionManualRepair, V14InterventionManualRerun, V14InterventionForcedRetry:
		return GapRunInitiatorNotCaptured, true
	case V14InterventionClarification:
		return GapClarificationNotCaptured, true
	case V14InterventionTradeOffDecision:
		return GapTradeOffDecisionNotCaptured, true
	default:
		return "", false
	}
}

// v14StructuralGaps 返回当前全部结构性缺口(固定集合、固定顺序;每次返回
// 新切片,调用方改动不影响包内状态)。
func v14StructuralGaps() []V14DataGap {
	return []V14DataGap{
		{
			Kind: GapRunInitiatorNotCaptured,
			Detail: "run 发起方/触发源事实未捕获:codegen_run 没有发起方维度," +
				"人工发起与闭环自动发起的 run 尚不可区分;manual_repair / " +
				"manual_rerun / forced_retry 的介入计数显式不可用(补齐前保持" +
				"缺失,不补 0)。",
			Affects: []V14InterventionKind{
				V14InterventionManualRepair, V14InterventionManualRerun, V14InterventionForcedRetry,
			},
		},
		{
			Kind: GapClarificationNotCaptured,
			Detail: "clarification 事实未捕获:intake 澄清请求/回答目前只存在于" +
				"提案流的内存形态,没有持久事实面;介入计数显式不可用。",
			Affects: []V14InterventionKind{V14InterventionClarification},
		},
		{
			Kind: GapTradeOffDecisionNotCaptured,
			Detail: "trade-off decision 事实未捕获:审批证据只承载 approve|reject" +
				" 与 waiver,权衡裁决没有专属事实类型;介入计数显式不可用。",
			Affects: []V14InterventionKind{V14InterventionTradeOffDecision},
		},
		{
			Kind: GapProductionIncidentNotCaptured,
			Detail: "production incident 关联事实未捕获:§31.5 的 escaped defect /" +
				" production incident 没有专属事实类型;当前最近似的已捕获形态是" +
				" regression_found 事件。",
			Affects: nil,
		},
	}
}

// ---- 输入:已落成事实的任务级投影(映射责任在调用方) ----

// V14RunFact 是一个实现 Run 的最小事实投影(身份 + 开始时刻)。§31 指标只用
// 它判定「任务进过闭环」;不携带状态/用时,避免本投影出现投机消费面。
type V14RunFact struct {
	ID        int64
	StartedAt time.Time
}

// V14ApprovalFact 是一条 owner 决定证据的投影(来自 approval 落库行)。
// WaivedIndependence=true 的行是 §8.10 waiver 介入的事实来源;不带豁免的
// approve/reject 是评审裁决,不是介入(AC2)。
type V14ApprovalFact struct {
	TaskID string
	// ChangesetID 是被审候选;空 = 非候选裁决(坐标缺席,不编造)。
	ChangesetID        string
	Decision           V14ApprovalDecision
	WaivedIndependence bool
	DecidedAt          time.Time
}

// V14SupersessionFact 是一条取代事实的投影(来自 runstore 的 append-only
// supersession 面)。Old/New 是候选快照身份。
type V14SupersessionFact struct {
	Old    string
	New    string
	Reason V14SupersedeReason
	At     time.Time
}

// V14TaskLoopFacts 是一个任务已落成闭环事实的投影(BuildV14LoopMetrics 的
// 输入)。全部字段都是事实读侧投影;本包不做采集,也不回写任何任务状态。
type V14TaskLoopFacts struct {
	// TaskID 归属任务(必填,输入内唯一)。
	TaskID string
	// IntentAt 是 intent 起点(任务计划事实时刻,§31.2 的起点端)。
	// 零值 = 未观测:该任务退出 Time to Accepted 样本,计入 MissingIntent,
	// 绝不用 run 开始时刻冒充 intent。
	IntentAt time.Time
	// Runs 是实现 Run 时间线(可为空;空且无 outcome = 任务未进过闭环)。
	Runs []V14RunFact
	// Approvals 是 owner 决定证据(waiver 介入的来源;可为空)。
	Approvals []V14ApprovalFact
	// Outcomes 是该任务的 §23 outcome 事件(outcome.Store.ForTask /
	// ForChangeset 读出;可为空)。TaskID 为空的事件(仅 changeset 绑定、
	// 经调用方映射进本任务)合法;TaskID 非空必须与本任务一致。
	Outcomes []outcome.Event
	// Supersessions 是该任务候选参与的取代事实(可为空)。
	Supersessions []V14SupersessionFact
}

// ---- 输出:§31 指标投影 ----

// V14LoopMetrics 是 §31 闭环产品指标的一次确定性投影。全部数值从输入事实
// 推导;样本不足的读数 Sufficient=false,消费方必须按「样本不足」呈现,不得
// 当作 0、当作通过,也不得据此做阈值结论(与 P1 报告同一 AC4 纪律)。
type V14LoopMetrics struct {
	// GeneratedAt 是投影生成时刻(显式传入)。
	GeneratedAt time.Time
	// Tasks 是进入投影的任务数;TasksInScope 是其中有闭环事实(≥1 Run 或
	// ≥1 outcome 事件)的任务数 —— §31.1 分母的口径(没进过闭环的任务不构成
	// hands-off 样本)。
	Tasks        int
	TasksInScope int

	HandsOff       V14HandsOffMetric
	TimeToAccepted V14TimeToAcceptedMetric
	Interventions  V14InterventionMetric
	Repairs        V14RepairMetric
	Rework         V14ReworkMetric
	Guardrails     V14SlowGuardrailMetric
	// Gaps 是结构性命名缺口(固定集合):对应输入事实在任何采集面都不存在,
	// 相关读数显式不可用而不是 0(AC1/AC3)。
	Gaps []V14DataGap
}

// V14HandsOffMetric 是 §31.1 Hands-off Completion Rate 的输入读数。
type V14HandsOffMetric struct {
	// AcceptedTasks 是到达验收终点(≥1 accepted 事件)的任务数。
	AcceptedTasks int
	// HandsOffAccepted 是其中未发生任何 §31.1 打破类介入的任务数
	// (waiver / trade-off / clarification 不打破,见 v14BreaksHandsOff)。
	HandsOffAccepted int
	// NotAccepted 是进过闭环但未到验收终点的任务数(同样不算 hands-off 完成)。
	NotAccepted int
	// Rate = HandsOffAccepted / TasksInScope。TasksInScope=0 时未定义:
	// RateValid=false,不猜、不补 0。
	Rate      float64
	RateValid bool
	// Samples 标注分母(TasksInScope)的样本充分性。
	Samples Samples
}

// V14TimeToAcceptedMetric 是 §31.2 Time to Accepted Change 的读数
// (IntentAt → 该任务首个 accepted 事件时刻,单位分钟)。
type V14TimeToAcceptedMetric struct {
	// Minutes 是两端点齐备任务的样本分布。
	Minutes ScalarDistribution
	// MissingIntent 是有 accepted 事实但缺 intent 起点的任务数:显式缺失,
	// 不进样本,也不被补成 0 时长(AC1)。
	MissingIntent int
}

// V14InterventionTally 是一个 §31.3 介入种类的计数。
type V14InterventionTally struct {
	// Count 是已观测的该类介入事件数。FactSourceCaptured=false 时 Count 恒为
	// 0,语义是「未捕获」而绝不是「观测到 0 次」—— 不得把缺失读成零(AC1)。
	Count int
	// FactSourceCaptured 报告该种类的输入事实是否已在系统某处捕获。
	FactSourceCaptured bool
	// Gap 是未捕获时指向的命名缺口(捕获面就绪后为空)。
	Gap V14DataGapKind
}

// V14InterventionMetric 是 §31.3 Human Intervention per Accepted Change 的
// 读数。全词表七类都出现且互相可区分(AC3);有事实源的按事实计数,无事实源
// 的以命名缺口显式不可用。
type V14InterventionMetric struct {
	// AcceptedChanges 是分母:accepted outcome 事件数(每次验收 = 一个
	// accepted change;返工后再验收产生新的 accepted change)。
	AcceptedChanges int
	// CapturedTotal 是已捕获介入事件合计(仅 FactSourceCaptured=true 的种类)。
	CapturedTotal int
	// ByKind 是全词表计数(含未捕获种类)。
	ByKind map[V14InterventionKind]V14InterventionTally
	// PerAcceptedChange = CapturedTotal / AcceptedChanges。分母为 0 时未定义
	// (PerAcceptedChangeValid=false),且它只覆盖已捕获种类 —— 命名缺口使
	// 该比率系统性偏低,渲染方必须连同缺口一并呈现。
	PerAcceptedChange      float64
	PerAcceptedChangeValid bool
	// Samples 标注分母(AcceptedChanges)的样本充分性。
	Samples Samples
}

// V14RepairMetric 是 repair attempts 读数(决策锁 8/23:每个外层 Repair =
// 新 Run + 新 ChangeSet + rework 取代事实;取代事实因此是 repair attempt 的
// 事实形态)。注意:未产出候选的失败 repair run 没有取代事实,本读数对这类
// attempt 计不到 —— 这是事实面的下界,不是估计。
type V14RepairMetric struct {
	// Total 是 reason=rework 的取代事实数(repair attempts)。
	Total int
	// TasksWithRepairs 是发生 ≥1 次 repair 的任务数。
	TasksWithRepairs int
	// PerTask 是每个 in-scope 任务 repair attempts 的样本分布:0 是观测到
	// 「零次修复」的合法事实,不是缺失(与缺失的区分靠样本语义,不靠省略)。
	PerTask ScalarDistribution
	// ByReason 是取代事实按原因的全词表计数(escalation / rework /
	// spec_defect 互相可区分;escalation 是升级重做,不计入 repair attempts)。
	ByReason map[V14SupersedeReason]int
}

// V14ReworkSignal 是 §31.4 再修改信号的封闭词表(同一 accepted change 的
// 不同再修改形态互相可区分)。
type V14ReworkSignal string

const (
	V14ReworkSupersededRework     V14ReworkSignal = "superseded_rework"
	V14ReworkSupersededSpecDefect V14ReworkSignal = "superseded_spec_defect"
	V14ReworkRegressionFound      V14ReworkSignal = "regression_found"
	V14ReworkReverted             V14ReworkSignal = "reverted"
	V14ReworkHotfixed             V14ReworkSignal = "hotfixed"
)

// v14ReworkSignals 是信号全词表固定顺序。
var v14ReworkSignals = []V14ReworkSignal{
	V14ReworkSupersededRework,
	V14ReworkSupersededSpecDefect,
	V14ReworkRegressionFound,
	V14ReworkReverted,
	V14ReworkHotfixed,
}

// v14ReworkOutcomeKind 把 outcome 事件类别映射为 rework 信号;非再修改类别
// (accepted / human_correction)返回 false。human_correction 不是对已验收
// change 的再修改事实,它归 §31.3 介入,不归 §31.4 rework。
func v14ReworkOutcomeKind(k outcome.Kind) (V14ReworkSignal, bool) {
	switch k {
	case outcome.KindRegressionFound:
		return V14ReworkRegressionFound, true
	case outcome.KindReverted:
		return V14ReworkReverted, true
	case outcome.KindHotfixed:
		return V14ReworkHotfixed, true
	default:
		return "", false
	}
}

// V14ReworkMetric 是 §31.4 14-day Rework Rate 的读数。归因局限见
// V14ReworkAttributionNote(AC4)。
type V14ReworkMetric struct {
	// Window 恒为 V14ReworkWindow(typed,显式)。
	Window time.Duration
	// AcceptedChanges 是分母:accepted 事件数(含缺 changeset 绑定的)。
	AcceptedChanges int
	// UnboundAccepted 是缺 changeset 绑定的 accepted 事件数:它们只能接收
	// 任务级归因的信号(change 级精度不可得),显式计数(AC1)。
	UnboundAccepted int
	// ReworkedChanges 是窗口内出现 ≥1 再修改信号的 accepted change 数。
	ReworkedChanges int
	// Rate = ReworkedChanges / AcceptedChanges。分母为 0 时未定义。
	Rate      float64
	RateValid bool
	// Samples 标注分母的样本充分性。
	Samples Samples
	// SignalCounts 是各再修改信号的全词表计数(同一 change 的多个信号各自
	// 计数;重复事实行各计一次)。
	SignalCounts map[V14ReworkSignal]int
	// UnattributedSignals 是任务绑定的再修改事件找到任何更早 accepted
	// change 时落空(事件早于首次验收)的次数:显式缺失,不静默丢弃。
	UnattributedSignals int
}

// V14SlowGuardrailMetric 是 §31.5 slow guardrail 的已捕获计数:rollback /
// hotfix / regression 直接对应 §23 事件类别。escaped defect / production
// incident 没有专属事实类型(GapProductionIncidentNotCaptured)。
type V14SlowGuardrailMetric struct {
	// ByKind 是三个 guardrail 事件类别的全量计数(未观测类别显式 0)。
	ByKind map[outcome.Kind]V14GuardrailTally
}

// V14GuardrailTally 是一个 guardrail 类别的计数与样本充分性。
type V14GuardrailTally struct {
	Count   int
	Samples Samples
}

// BuildV14LoopMetrics 从已落成事实投影组装 §31 闭环产品指标。
//
// 输入校验是硬边界:任务 id 必填且唯一;Run/决定/取代事实自洽;outcome 事件
// 经 Event.Validate,归属非空时必须与本任务一致;验收时刻不得早于 intent
// 起点。映射错误(而非数据缺失)一律报错 —— 指标丢行或静默归类比失败更危险。
// 数据缺失不是错误:按命名缺口 / MissingIntent / UnboundAccepted 等显式呈现
// (AC1)。
func BuildV14LoopMetrics(tasks []V14TaskLoopFacts, generatedAt time.Time) (V14LoopMetrics, error) {
	if generatedAt.IsZero() {
		return V14LoopMetrics{}, fmt.Errorf("measurement: v14 闭环指标缺少生成时刻")
	}
	m := V14LoopMetrics{
		GeneratedAt: generatedAt.UTC(),
		Tasks:       len(tasks),
		Interventions: V14InterventionMetric{
			ByKind: make(map[V14InterventionKind]V14InterventionTally, len(v14InterventionKinds)),
		},
		Repairs:    V14RepairMetric{ByReason: make(map[V14SupersedeReason]int, 3)},
		Rework:     V14ReworkMetric{Window: V14ReworkWindow, SignalCounts: make(map[V14ReworkSignal]int, len(v14ReworkSignals))},
		Guardrails: V14SlowGuardrailMetric{ByKind: make(map[outcome.Kind]V14GuardrailTally, 3)},
		Gaps:       v14StructuralGaps(),
	}
	// 词表全量初始化:未观测取值显式 0 计数,不靠省略表达缺失(AC3/AC4)。
	for _, k := range v14InterventionKinds {
		t := V14InterventionTally{}
		if gap, uncap := v14InterventionGap(k); uncap {
			t.Gap = gap
		} else {
			t.FactSourceCaptured = true
		}
		m.Interventions.ByKind[k] = t
	}
	for _, r := range []V14SupersedeReason{V14SupersedeEscalation, V14SupersedeRework, V14SupersedeSpecDefect} {
		m.Repairs.ByReason[r] = 0
	}
	for _, s := range v14ReworkSignals {
		m.Rework.SignalCounts[s] = 0
	}
	for _, k := range []outcome.Kind{outcome.KindReverted, outcome.KindHotfixed, outcome.KindRegressionFound} {
		m.Guardrails.ByKind[k] = V14GuardrailTally{}
	}

	seen := make(map[string]bool, len(tasks))
	var ttaMinutes []float64
	var perTaskRepairs []float64

	for i := range tasks {
		t := &tasks[i]
		if strings.TrimSpace(t.TaskID) == "" {
			return V14LoopMetrics{}, fmt.Errorf("measurement: v14 闭环指标输入 tasks[%d] 缺少 task_id", i)
		}
		if seen[t.TaskID] {
			return V14LoopMetrics{}, fmt.Errorf("measurement: v14 闭环指标输入任务重复: %s", t.TaskID)
		}
		seen[t.TaskID] = true

		for j, r := range t.Runs {
			switch {
			case r.ID <= 0:
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 runs[%d] 缺少正数 run id", t.TaskID, j)
			case r.StartedAt.IsZero():
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 runs[%d] 缺少开始时刻", t.TaskID, j)
			}
		}
		for j, a := range t.Approvals {
			switch {
			case a.TaskID != t.TaskID:
				return V14LoopMetrics{}, fmt.Errorf("measurement: 决定证据归属任务 %s,却出现在任务 %s 的投影里(approvals[%d])", a.TaskID, t.TaskID, j)
			case !a.Decision.IsValid():
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 approvals[%d] 决定 %q 非法,必须是 approve|reject", t.TaskID, j, string(a.Decision))
			case a.DecidedAt.IsZero():
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 approvals[%d] 缺少决定时刻", t.TaskID, j)
			}
		}
		for j, e := range t.Outcomes {
			if err := e.Validate(); err != nil {
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 outcomes[%d]: %w", t.TaskID, j, err)
			}
			if e.TaskID != "" && e.TaskID != t.TaskID {
				return V14LoopMetrics{}, fmt.Errorf("measurement: outcome 事件归属任务 %s,却出现在任务 %s 的投影里(outcomes[%d])", e.TaskID, t.TaskID, j)
			}
		}
		for j, f := range t.Supersessions {
			switch {
			case f.Old == "" || f.New == "":
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 supersessions[%d] 缺少 old/new 候选身份", t.TaskID, j)
			case f.Old == f.New:
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 supersessions[%d] old 与 new 相同", t.TaskID, j)
			case !f.Reason.IsValid():
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 supersessions[%d] 原因 %q 非法,必须是 escalation|rework|spec_defect", t.TaskID, j, string(f.Reason))
			case f.At.IsZero():
				return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 supersessions[%d] 缺少取代时刻", t.TaskID, j)
			}
		}

		// 未进过闭环的任务(无 Run 且无 outcome 事件)不构成 §31.1 样本。
		if len(t.Runs) == 0 && len(t.Outcomes) == 0 {
			continue
		}
		m.TasksInScope++

		// ---- §31.3 介入(已捕获源:waiver ← 决定证据;direct code
		// correction ← human_correction 事件)与 §31.1 打破判定 ----
		violated := false
		for _, a := range t.Approvals {
			if !a.WaivedIndependence {
				continue
			}
			tally := m.Interventions.ByKind[V14InterventionWaiver]
			tally.Count++
			m.Interventions.ByKind[V14InterventionWaiver] = tally
		}
		for _, e := range t.Outcomes {
			if e.Kind != outcome.KindHumanCorrection {
				continue
			}
			tally := m.Interventions.ByKind[V14InterventionDirectCodeCorrection]
			tally.Count++
			m.Interventions.ByKind[V14InterventionDirectCodeCorrection] = tally
		}
		for _, k := range v14InterventionKinds {
			if v14BreaksHandsOff(k) && m.Interventions.ByKind[k].Count > 0 {
				violated = true
			}
		}
		// 注:violated 只可能由 direct_code_correction 触发 —— 其余打破类
		// 种类的计数恒为 0(命名缺口),任务级打破判定随捕获面补齐自动生效。

		// ---- accepted 事件(验收终点事实) ----
		var accepted []outcome.Event
		for _, e := range t.Outcomes {
			if e.Kind == outcome.KindAccepted {
				accepted = append(accepted, e)
			}
		}

		// ---- §31.1 hands-off 输入 ----
		if len(accepted) > 0 {
			m.HandsOff.AcceptedTasks++
			if !violated {
				m.HandsOff.HandsOffAccepted++
			}
		}

		// ---- §31.2 time to accepted ----
		m.Interventions.AcceptedChanges += len(accepted)
		m.Rework.AcceptedChanges += len(accepted)
		if len(accepted) > 0 {
			first := accepted[0].RecordedAt
			unbound := 0
			for _, e := range accepted {
				if e.ChangesetID == "" {
					unbound++
				}
				if e.RecordedAt.Before(first) {
					first = e.RecordedAt
				}
			}
			m.Rework.UnboundAccepted += unbound
			if t.IntentAt.IsZero() {
				m.TimeToAccepted.MissingIntent++
			} else {
				if first.Before(t.IntentAt) {
					return V14LoopMetrics{}, fmt.Errorf("measurement: 任务 %s 的 accepted 时刻早于 intent 起点(事实映射矛盾)", t.TaskID)
				}
				ttaMinutes = append(ttaMinutes, Minutes(first.Sub(t.IntentAt)))
			}
		}

		// ---- repair attempts(决策锁 8/23 的事实形态) ----
		taskRepairs := 0
		for _, f := range t.Supersessions {
			m.Repairs.ByReason[f.Reason]++
			if f.Reason == V14SupersedeRework {
				taskRepairs++
			}
		}
		m.Repairs.Total += taskRepairs
		if taskRepairs > 0 {
			m.Repairs.TasksWithRepairs++
		}
		perTaskRepairs = append(perTaskRepairs, float64(taskRepairs))

		// ---- §31.4 rework 与 §31.5 guardrails ----
		v14ApplyRework(&m.Rework, accepted, t.Outcomes, t.Supersessions)
		for _, e := range t.Outcomes {
			g, ok := m.Guardrails.ByKind[e.Kind]
			if !ok {
				continue // accepted / human_correction 归主指标,不归 guardrails
			}
			g.Count++
			m.Guardrails.ByKind[e.Kind] = g
		}
	}

	// ---- 汇总读数与充分性标注(AC4) ----
	m.HandsOff.NotAccepted = m.TasksInScope - m.HandsOff.AcceptedTasks
	if m.TasksInScope > 0 {
		m.HandsOff.Rate = float64(m.HandsOff.HandsOffAccepted) / float64(m.TasksInScope)
		m.HandsOff.RateValid = true
	}
	m.HandsOff.Samples = sampleSufficiency(m.TasksInScope)

	m.TimeToAccepted.Minutes = newScalarDistribution(ttaMinutes)

	// CapturedTotal 只累计事实面已在账的种类;命名缺口种类保持缺失,
	// 不被并进任何合计(AC1)。
	for _, k := range v14InterventionKinds {
		if t := m.Interventions.ByKind[k]; t.FactSourceCaptured {
			m.Interventions.CapturedTotal += t.Count
		}
	}
	if m.Interventions.AcceptedChanges > 0 {
		m.Interventions.PerAcceptedChange =
			float64(m.Interventions.CapturedTotal) / float64(m.Interventions.AcceptedChanges)
		m.Interventions.PerAcceptedChangeValid = true
	}
	m.Interventions.Samples = sampleSufficiency(m.Interventions.AcceptedChanges)

	m.Repairs.PerTask = newScalarDistribution(perTaskRepairs)

	if m.Rework.AcceptedChanges > 0 {
		m.Rework.Rate = float64(m.Rework.ReworkedChanges) / float64(m.Rework.AcceptedChanges)
		m.Rework.RateValid = true
	}
	m.Rework.Samples = sampleSufficiency(m.Rework.AcceptedChanges)

	for k, g := range m.Guardrails.ByKind {
		g.Samples = sampleSufficiency(g.Count)
		m.Guardrails.ByKind[k] = g
	}
	return m, nil
}

// v14ApplyRework 把一个任务的再修改信号归因到 accepted change 并累入 rework
// 读数(AC4;归因局限见 V14ReworkAttributionNote):
//
//   - change 级:取代事实(old=该候选)与 changeset 绑定的再修改 outcome
//     事件,归因到该候选的首次验收事件;
//   - 任务级:仅有任务绑定的再修改 outcome 事件,归因到任务内最近一次先于
//     事件的验收(§31.4 的「WorkItem / Change」两级口径);没有任何先于事件
//     的验收时计入 UnattributedSignals,不静默丢弃;
//   - 窗口:验收时刻 < 信号时刻 ≤ 验收时刻 + V14ReworkWindow。验收前的取代
//     是闭环内的正常 repair,归 Repairs,不归 rework。
func v14ApplyRework(r *V14ReworkMetric, accepted, outcomes []outcome.Event, sups []V14SupersessionFact) {
	signaled := make(map[int]bool)
	mark := func(idx int, s V14ReworkSignal) {
		signaled[idx] = true
		r.SignalCounts[s]++
	}

	// 取代事实:被取代候选若被验收过,验收后的取代是再修改。
	for _, f := range sups {
		var signal V14ReworkSignal
		switch f.Reason {
		case V14SupersedeRework:
			signal = V14ReworkSupersededRework
		case V14SupersedeSpecDefect:
			signal = V14ReworkSupersededSpecDefect
		default: // escalation 是升级重做,不是缺陷返工
			continue
		}
		idx, ok := v14FirstAcceptedIndex(accepted, f.Old)
		if !ok || !v14InReworkWindow(accepted[idx].RecordedAt, f.At) {
			continue
		}
		mark(idx, signal)
	}
	// outcome 事件:changeset 绑定走 change 级,仅任务绑定走任务级。
	for _, e := range outcomes {
		signal, ok := v14ReworkOutcomeKind(e.Kind)
		if !ok {
			continue
		}
		if e.ChangesetID != "" {
			idx, bound := v14FirstAcceptedIndex(accepted, e.ChangesetID)
			if bound && v14InReworkWindow(accepted[idx].RecordedAt, e.RecordedAt) {
				mark(idx, signal)
			}
			continue
		}
		idx, prior := v14LatestAcceptedBefore(accepted, e.RecordedAt)
		if !prior {
			r.UnattributedSignals++
			continue
		}
		if v14InReworkWindow(accepted[idx].RecordedAt, e.RecordedAt) {
			mark(idx, signal)
		}
	}
	r.ReworkedChanges += len(signaled)
}

// v14InReworkWindow 报告信号时刻是否落在验收后的 rework 窗口内。
func v14InReworkWindow(acceptedAt, at time.Time) bool {
	return at.After(acceptedAt) && !at.After(acceptedAt.Add(V14ReworkWindow))
}

// v14FirstAcceptedIndex 返回候选 cs 的首次验收事件下标(同刻取最早;无则
// false)。
func v14FirstAcceptedIndex(accepted []outcome.Event, cs string) (int, bool) {
	best := -1
	for i, e := range accepted {
		if e.ChangesetID != cs {
			continue
		}
		if best == -1 || e.RecordedAt.Before(accepted[best].RecordedAt) {
			best = i
		}
	}
	return best, best >= 0
}

// v14LatestAcceptedBefore 返回严格先于 at 的最近验收事件下标(同刻按切片
// 位置取后者 = 更晚追加;无则 false)。
func v14LatestAcceptedBefore(accepted []outcome.Event, at time.Time) (int, bool) {
	best := -1
	for i, e := range accepted {
		if !e.RecordedAt.Before(at) {
			continue
		}
		if best == -1 || !e.RecordedAt.Before(accepted[best].RecordedAt) {
			best = i
		}
	}
	return best, best >= 0
}
