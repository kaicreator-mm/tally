// Package measurement 实现 PRD §25/§30/§38 的 P1 度量事实模型(P1-08):
// 人工控制开销(active/wall 双时长)、LLM/Agent 使用记录(cost_plane 在写入时
// 落成事实字段)与生命周期阶段计时的稳定命名。
//
// 边界(P1):
//   - 本包只定义事实记录的类型、合法枚举与确定性校验;不写库(落库归
//     store/sqlc 任务),也不做查询期重算 —— 尤其不存在任何 role→cost_plane
//     的查询期映射:归属只读每条记录写入时已落成的 cost_plane(PRD §25.3/§30)。
//   - Agent runtime 自报的 token 用量是 claim(internal/agentruntime 的 Usage),
//     本包的 LLMUsage 是 Tally 侧落成的度量事实;两者不混用,claim 不得未经
//     落成直接进事实面。
package measurement

import (
	"fmt"
	"strings"
	"time"
)

// CostPlane 是一次 LLM/Agent 调用的成本归属面(PRD §25.3/§30):
//
//   - control:Intake、产品/架构治理、Planning、验证规划、Review 辅助、
//     orchestration/上下文推理;
//   - execution:Coding、Debug/Fix、实现侧测试生成、文档生成、实现导向推理
//     (文档生成是实际工程产出,不计入 control)。
//
// 它必须在调用发生时由调用方写入,是历史事实字段(PRD §30:否则未来
// role→plane 映射变化会重写历史 control_plane_llm_cost_ratio)。本包只存储
// 与校验该值,不做任何推导;报表按存储值聚合,历史永不重算。
type CostPlane string

const (
	CostPlaneControl   CostPlane = "control"
	CostPlaneExecution CostPlane = "execution"
)

// IsValid 报告该值是否是两个合法归属面之一。空串非法:cost_plane 在 LLM 使用
// 记录上必填,没有默认归属面 —— 猜测会在写入时把历史事实写错。
func (p CostPlane) IsValid() bool {
	switch p {
	case CostPlaneControl, CostPlaneExecution:
		return true
	}
	return false
}

// UsageStatus 是一次 LLM/Agent 调用终态的事实分类(PRD §30 的 status)。
// 失败/取消调用的 token 与成本照样按实际发生值记录,只是终态不同。
type UsageStatus string

const (
	UsageSucceeded UsageStatus = "succeeded"
	UsageFailed    UsageStatus = "failed"
	UsageCancelled UsageStatus = "cancelled"
)

// IsValid 报告该值是否是合法调用终态。空串非法:终态必须显式写入。
func (s UsageStatus) IsValid() bool {
	switch s {
	case UsageSucceeded, UsageFailed, UsageCancelled:
		return true
	}
	return false
}

// LLMUsage 是一次 LLM/Agent 调用的使用/成本事实记录(PRD §30 的 llm_run 度量面、
// §38.4 的 cost 口径)。全部字段在调用发生时写入,之后只读。
type LLMUsage struct {
	// TaskID 归属任务;任务外的控制面调用(如 intake)允许为空。
	TaskID string
	// UseCase 是调用用途(PRD §30 的 use_case/role),如 verify_contract_draft。
	// 它只是描述;成本归属由 CostPlane 承载,与 use_case 命名无关。
	UseCase string
	// CostPlane 是成本归属面,写入时必填(AC1);见 CostPlane。
	CostPlane CostPlane

	// Provider/Runtime/Model 是调用面标识(PRD §30 provider/runtime/model)。
	// 探测面拿不到时留空,不猜。
	Provider string
	Runtime  string
	Model    string

	// Prompt/Behavior revision 与 context hash(PRD §30),原样记录。
	PromptRevision   string
	BehaviorRevision string
	ContextHash      string

	// token 用量与耗时;失败调用也按实际发生值记录。
	InputTokens  int64
	OutputTokens int64
	Latency      time.Duration

	// CostMicros 是本次调用实际发生的成本,单位为记账货币的 1e-6(micro)。
	// 固定整数口径,避免浮点累加误差;所有比率(§25.3/§38.4)一律由本字段推导。
	CostMicros int64

	// UsageReported 显式标记本调用的用量是否被 provider/runtime 上报
	// (P0-05/P0-06,决策锁 16:migrations/0022)。false = 用量未观测(未知):
	// 落库为 usage_reported=false + 三列 NULL,领域侧三值必须为零 —— 「未知」
	// 与「观测到 0」不得互化,不得把未观测合成假 0 上报。true = InputTokens/
	// OutputTokens/CostMicros 是已上报的观测值(观测到 0 是合法事实)。
	UsageReported bool

	// Status 是调用终态;FailureType 仅在失败时给出类型描述(原样记录)。
	Status      UsageStatus
	FailureType string

	// HumanIntervened 记录本次调用是否发生人工介入(PRD §30 human intervention)。
	HumanIntervened bool

	// AuthoringProvenance 在适用时原样记录 authoring/conformance provenance
	// (PRD §30);其语义归 verification/taskcontract,本包不解释。
	AuthoringProvenance string

	// RecordedAt 是事实写入时间;度量事实必须可定位到发生时间。
	RecordedAt time.Time
}

// Validate 校验记录的硬约束(PRD §30):
//
//   - use_case 必填;
//   - cost_plane 必须是 control|execution(AC1:LLM 使用上必填,无默认归属面);
//   - status 必须是合法终态;
//   - token/latency/cost 不得为负;RecordedAt 必须显式给出;
//   - UsageReported=false 时 token/cost 三值必须为零(锁 16:未上报的未知
//     用量不是「观测到 0」,携带数值等于把未知伪装成已知,migrations/0022
//     的 CHECK 在数据库层拒绝同一形状)。
//
// 其余字段(provider/model/revision/hash 等)允许为空:探测面不全时,记录已知
// 事实比编造字段值更符合「事实 = 观测」原则。
func (u LLMUsage) Validate() error {
	if strings.TrimSpace(u.UseCase) == "" {
		return fmt.Errorf("measurement: llm usage 缺少 use_case")
	}
	if !u.CostPlane.IsValid() {
		return fmt.Errorf("measurement: llm usage 的 cost_plane %q 非法,必须是 control|execution(写入时必填,PRD §25.3/§30)", string(u.CostPlane))
	}
	if !u.Status.IsValid() {
		return fmt.Errorf("measurement: llm usage 的 status %q 非法,必须是 succeeded|failed|cancelled 之一", string(u.Status))
	}
	if u.InputTokens < 0 {
		return fmt.Errorf("measurement: llm usage 的 input_tokens 为负")
	}
	if u.OutputTokens < 0 {
		return fmt.Errorf("measurement: llm usage 的 output_tokens 为负")
	}
	if u.Latency < 0 {
		return fmt.Errorf("measurement: llm usage 的 latency 为负")
	}
	if u.CostMicros < 0 {
		return fmt.Errorf("measurement: llm usage 的 cost 为负")
	}
	if !u.UsageReported && (u.InputTokens != 0 || u.OutputTokens != 0 || u.CostMicros != 0) {
		return fmt.Errorf("measurement: llm usage 未上报用量时不得携带数值(锁 16:未知不是 0)")
	}
	if u.RecordedAt.IsZero() {
		return fmt.Errorf("measurement: llm usage 缺少 recorded_at(度量事实必须可定位到发生时间)")
	}
	return nil
}

// CostByPlane 按每条记录写入时落成的 cost_plane 聚合成本(micros),
// 返回 (control, execution)。归属只由存储的 cost_plane 决定(AC4):
// use_case/provider/model 等字段无论叫什么都不影响归属,包内不存在任何
// role→plane 查询期映射。cost_plane 非法的记录不计入(写入方应先 Validate;
// 此处保守跳过而不是 panic 或猜测归属)。
func CostByPlane(usages []LLMUsage) (control, execution int64) {
	for _, u := range usages {
		switch u.CostPlane {
		case CostPlaneControl:
			control += u.CostMicros
		case CostPlaneExecution:
			execution += u.CostMicros
		}
	}
	return control, execution
}

// ControlPlaneCostRatio 返回 control_plane_llm_cost_ratio =
// control_plane_llm_cost / total_llm_cost(PRD §25.3/§38.4)。
// 总成本为 0(无记录或全部 0 成本)时返回 (0, false):比率未定义,不猜。
// 计算只读存储的 cost_plane,不做任何查询期重映射(AC4)。
func ControlPlaneCostRatio(usages []LLMUsage) (float64, bool) {
	control, execution := CostByPlane(usages)
	total := control + execution
	if total == 0 {
		return 0, false
	}
	return float64(control) / float64(total), true
}

// HumanEffortKind 对应 PRD §25「至少记录」的人工控制开销条目
// (idea_to_verified_latency 属于里程碑跨度,归 timing.go,不在这里)。
type HumanEffortKind string

const (
	EffortContractReview     HumanEffortKind = "contract_review"      // contract_review_minutes
	EffortVerificationReview HumanEffortKind = "verification_review"  // verification_review_minutes
	EffortDiffReview         HumanEffortKind = "diff_review"          // diff_review_active/wall_minutes
	EffortManualCorrection   HumanEffortKind = "manual_correction"    // manual_correction_minutes
	EffortSpecDefectDecision HumanEffortKind = "spec_defect_decision" // spec_defect_decision_minutes
)

// IsValid 报告该值是否是 PRD §25 定义的人工开销条目之一。
func (k HumanEffortKind) IsValid() bool {
	switch k {
	case EffortContractReview, EffortVerificationReview, EffortDiffReview,
		EffortManualCorrection, EffortSpecDefectDecision:
		return true
	}
	return false
}

// HumanEffort 是一次人工控制开销事实记录(PRD §25/§38.2)。
//
// Active 与 Wall 是两个不同的度量,不得合并成一个字段(AC2):
//   - Active 是人实际用于审核/裁决/修正的时间 —— North Star
//     (Verified Engineering Output per Human Review Minute)的分母口径;
//   - Wall 是端到端等待时间(提交后到人介入完成的墙钟跨度)。
//
// 不强制 wall >= active:两个时长可能来自不同测量窗(分段上报各自 active/wall),
// 强加不变式会把合法分段记录判为非法;跨记录的口径一致性归采集方。
type HumanEffort struct {
	// TaskID 归属任务(§38.2 人工开销按任务分维,必填)。
	TaskID string
	// Kind 是 §25 的人工开销条目。
	Kind HumanEffortKind
	// Active 是人工主动投入时长;Wall 是端到端等待时长。允许其一为 0
	// (只测到其中一个口径),但不允许为负。
	Active time.Duration
	Wall   time.Duration
	// RecordedAt 是事实写入时间。
	RecordedAt time.Time
}

// ActiveMinutes 返回 PRD *_minutes 口径(分钟)的人工主动时长。
func (h HumanEffort) ActiveMinutes() float64 { return Minutes(h.Active) }

// WallMinutes 返回 PRD *_minutes 口径(分钟)的墙钟时长。
func (h HumanEffort) WallMinutes() float64 { return Minutes(h.Wall) }

// Validate 校验记录的硬约束:任务归属、条目名、非负时长、写入时间。
func (h HumanEffort) Validate() error {
	if strings.TrimSpace(h.TaskID) == "" {
		return fmt.Errorf("measurement: human effort 缺少 task_id(人工开销按任务归属,PRD §38.2)")
	}
	if !h.Kind.IsValid() {
		return fmt.Errorf("measurement: human effort 的 kind %q 非法,必须是 PRD §25 定义的人工开销条目之一", string(h.Kind))
	}
	if h.Active < 0 {
		return fmt.Errorf("measurement: human effort 的 active 时长为负")
	}
	if h.Wall < 0 {
		return fmt.Errorf("measurement: human effort 的 wall 时长为负")
	}
	if h.RecordedAt.IsZero() {
		return fmt.Errorf("measurement: human effort 缺少 recorded_at(度量事实必须可定位到发生时间)")
	}
	return nil
}
