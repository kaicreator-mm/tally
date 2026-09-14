// Package airuntime 是 Tally 域侧对公共 AI Runtime 的薄适配边界
// (PRD §28 / INV-V14-14 / DECISION_LOCKS #17)。
//
// 边界分工:
//
//	Tally 域只声明 operation / intent / input / budget;
//	公共 AI Runtime 负责 provider 选择、model 路由、tier、重试、成本与 trace。
//
// 因此本包的 Request 里没有、也不允许出现 provider/model/tier/BaseURL/APIKey
// —— 那些是 Runtime 侧的配置,不是域侧的声明(test/arch 用 AST 检查)。
// 旧 internal/llm 自带 BaseURL/APIKey/model 的直连供应商路径是遗留现状,
// 由 P0-10/P0-11 把消费方迁到本边界后在 P0-11 退役;本包不导入 internal/llm。
//
// 本包与 AgentRuntime 是两套系统(PRD §28):coding workspace、shell、
// session、patch 等 Coding Agent 语义留在 internal/agentruntime 与
// runtimes/*,永远不进入本包;反向也不成立 —— AgentRuntime 不经本包做推理。
//
// 本包只提案不判定(INV-1 同款约束):Result.Output 是 Runtime 的原始产物,
// 语义验证永远在调用方;接口方法只有 Execute(test/arch 用反射检查)。
//
// 契约对齐:类型与 JSON 字段名镜像公共 AI Runtime 冻结契约
// (api/openapi.yaml v0.4.0-m1-frozen,ADR-0002):
//
//	operation / intent / input / budget{max_attempts, max_cost_ceiling,
//	deadline_ms, max_tokens} + Idempotency-Key 头(1..200 字符)。
//
// 允许导入:标准库(不含 net/http —— 传输属于公共 AI Runtime 的 SDK,
// 不在本包重建)与 tally/internal/logpipe(脱敏闸)。
package airuntime

import "context"

// Executor 是消费方在 composition root 实现的传输缝:把契约形状的 Request
// 送到公共 AI Runtime,把 Operation 完成事实映射回 Result。
//
// 语义约定(实现方必须遵守,信任不变量):
//
//   - error 表示传输/基础设施层失败(网络、鉴权、序列化、ctx 取消):
//     没有拿到 Operation 结果。此时不得编造 Result。
//   - Result.Status 是 Runtime 自己的终态判定,含 UNCERTAIN(拿不准)。
//     取消、提供方错误在 error 或对应 Status/TerminationReason 里如实呈现,
//     不得折叠成候选失败(FAIL 与 ERROR 端到端可区分)。
//   - 实现方应在执行前调用 Request.Validate()(或用 Validated 包装),
//     让域侧机械闸(意图闸/幂等键/预算/输入/脱敏)不依赖每个实现方自觉。
type Executor interface {
	Execute(ctx context.Context, req Request) (Result, error)
}
