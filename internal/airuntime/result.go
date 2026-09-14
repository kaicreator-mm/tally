package airuntime

import "encoding/json"

// Status 镜像公共 AI Runtime 契约的 OperationStatus(终态)。
// UNCERTAIN 是一等公民:Runtime 拿不准时如实说拿不准,
// 调用方不得把它折叠成失败或成功(信任不变量:FAIL 与 ERROR/拿不准可区分)。
type Status string

const (
	StatusSucceeded Status = "SUCCEEDED"
	StatusUncertain Status = "UNCERTAIN"
	StatusFailed    Status = "FAILED"
	StatusCancelled Status = "CANCELLED"
)

// TerminationReason 镜像契约 TerminationReason:为什么停,独立于 Status。
type TerminationReason string

const (
	TerminationAttemptsExhausted TerminationReason = "ATTEMPTS_EXHAUSTED"
	TerminationBudgetExhausted   TerminationReason = "BUDGET_EXHAUSTED"
	TerminationClientCancelled   TerminationReason = "CLIENT_CANCELLED"
	TerminationCompleted         TerminationReason = "COMPLETED"
	TerminationDeadlineExceeded  TerminationReason = "DEADLINE_EXCEEDED"
	TerminationPolicyRejected    TerminationReason = "POLICY_REJECTED"
	TerminationProviderFailure   TerminationReason = "PROVIDER_FAILURE"
)

// StrategyAssignment 镜像契约 StrategyAssignment:本次操作实际落在哪个
// rollout 策略。这是 Runtime 观测到的事实,不是域侧的期望配置
// (DECISION_LOCKS #15:actual != desired)。
type StrategyAssignment string

const (
	StrategyBaseline      StrategyAssignment = "BASELINE"
	StrategyCanary        StrategyAssignment = "CANARY"
	StrategyShadowPrimary StrategyAssignment = "SHADOW_PRIMARY"
)

// Result 承载公共 AI Runtime 的 Operation 完成事实(契约 Operation 的域侧映射)。
// 本包对 Result 不做任何判定:Output 是 Runtime 的原始产物,
// 语义验证(反序列化、业务校验)永远在调用方 —— LLM/AI 产出永远是 Proposal。
type Result struct {
	// OperationID 是 Runtime 分配的操作 ID,供后续 trace/outcome 查询。
	OperationID string `json:"operation_id"`
	// Status 是 Runtime 的终态判定。
	Status Status `json:"status"`
	// TerminationReason 解释为什么停,独立于 Status。
	TerminationReason TerminationReason `json:"termination_reason"`
	// Output 是原始产物(JSON);语义验证在调用方,可能为空。
	Output json.RawMessage `json:"output,omitempty"`
	// ProductionCostUSD 是 Runtime 计量的生产成本(美元展示值)。
	ProductionCostUSD float64 `json:"production_cost"`
	// Strategy 是本次实际命中的 rollout 策略(Runtime 观测)。
	Strategy StrategyAssignment `json:"strategy_assignment"`
}
