// failure.go 实现 P2-03:验证/基础设施结果的失败分类(PRD §17.4)与
// Router 端口(ports.go 的类型化失败路由缝)的默认实现。
//
// 纪律:
//
//   - 封闭词表:失败类是编译期常量的封闭集(PRD §17.4 的七类 + unknown),
//     没有自由字符串;每个类恰好映射一个确定性闭环动作(AC1);
//   - fail-safe:unknown、无法解析、非法值的落点一律升级,绝不修复、绝不
//     静默(INV-V14-09:unknown/unmatched 绝不静默选最弱路径)(AC2);
//   - FAIL ≠ ERROR(ADR-0001 §3):只有可信 FAIL 才可能是候选失败
//     (repairable);inconclusive / environment / provider 不是候选失败,
//     永不进修复、不耗修复预算、不被伪装成候选失败(INV-09/10);
//   - 不重释验收语义(AC3):分类消费已成的类型化事实(verdict 词、编排
//     结论、类型化 error),只命名失败类;纯函数的输入里根本不存在验收
//     材料,不存在接受/拒绝候选或改写冻结验收的路径;
//   - 无字符串嗅探:错误分类只经 errors.Is 判因(internal/verification 的
//     类型化哨兵 + context 超时),绝不做 stderr/message 匹配。
package closedloop

import (
	"context"
	"errors"

	"tally/internal/verification"
)

// FailureClass 是失败分类的封闭词表(PRD §17.4)。字符串值即 PRD 词面:
// 经 RouteEscalate 原样透传后成为 Escalation.Reason,稳定且人类可读
// (词表解释权归本文件;Engine 不解释升级原因,只执行裁决)。
type FailureClass string

const (
	// FailureRepairable:可信的候选缺陷(验证器给出了确定性失败判定)。
	// 默认唯一进入自动修复的类(PRD §17.4):每次修复 = 新 Run + 新一次性
	// worktree + 新不可变 ChangeSet + reason=rework 取代事实(ADR-0001 §2);
	// 修复预算与「反复修复失败即升级」由 Engine 与 P2-07 治理(§17.2/§18.2)。
	FailureRepairable FailureClass = "repairable"
	// FailureContractDefect:冻结契约自身的缺陷(如契约未声明 INV-VER-05
	// 所需 runtime 能力)。候选修复不可能修复契约,升级给人。
	FailureContractDefect FailureClass = "contract_defect"
	// FailurePolicyConflict:冻结执行/治理纪律的冲突(§8.7.2 cherry-pick
	// 纪律:超停止规则重跑、同序号覆盖、seed 换用;决策锁 12:一次性
	// held-out 证据再消耗)。纪律冲突不重试,升级给人/政策裁决。
	FailurePolicyConflict FailureClass = "policy_conflict"
	// FailureEnvironment:验证/执行环境失败(墙钟预算耗尽、workspace 缺失、
	// 环境身份漂移、执行边界产出畸形、超时)。不是候选失败:走基础设施
	// 重试路线(上限与节奏归 P2-08 组合层接线;一次性资源绝不重耗,决策
	// 锁 12),反复失败升级(§18.2)。
	FailureEnvironment FailureClass = "environment_failure"
	// FailureProvider:代码生成提供方(coding agent runtime)失败。不是
	// 候选失败:走 provider 重试路线(新 CodegenRun,归 P2-08 接线),
	// 反复失败升级(§18.2)。
	FailureProvider FailureClass = "provider_failure"
	// FailureInconclusive:验证器无法给出可信 verdict(ERROR 语义)。结构上
	// 不可 accept、不可 repair、不算候选失败、不耗预算(ADR-0001 §3);
	// 词面与 Engine 内部 EscalationInconclusive 一致(测试守卫)。
	FailureInconclusive FailureClass = "verification_inconclusive"
	// FailureRequiresHuman:需要人的判断(held-out/独立性覆盖缺口需政策或
	// waiver 裁决、保护证据链完整性被破坏等 —— §18.2)。
	FailureRequiresHuman FailureClass = "requires_human_judgment"
	// FailureUnknown:无法归入任何已知名类(未匹配的 error、接线/misuse
	// 哨兵、非失败结论被误送分类)。fail-safe 词面:唯一动作是升级
	// (AC2;INV-V14-09)。
	FailureUnknown FailureClass = "unknown"
)

// allFailureClasses 是词表的穷举(测试用它守住封闭集的完整性)。
var allFailureClasses = [...]FailureClass{
	FailureRepairable, FailureContractDefect, FailurePolicyConflict,
	FailureEnvironment, FailureProvider, FailureInconclusive,
	FailureRequiresHuman, FailureUnknown,
}

// IsValid 报告 c 是否在封闭词表内。
func (c FailureClass) IsValid() bool {
	for _, known := range allFailureClasses {
		if c == known {
			return true
		}
	}
	return false
}

// ClassAction 是分类驱动的确定性闭环动作词表(AC1:repair / escalate /
// retry-provider / stop)。分类只裁决不执行:动作由闭环(Engine 与 P2-08
// 的组合层接线)落地。
type ClassAction uint8

const (
	// ActionRepair:进入外层修复(仅 repairable;预算治理归 Engine/P2-07)。
	ActionRepair ClassAction = iota
	// ActionRetryProvider:提供方/基础设施重试路线(environment / provider)。
	// 不耗外层修复预算(决策锁 7/23:inner retry ≠ outer Repair 的另一面
	// —— infra 重试同样不是外层修复);上限与节奏归 P2-08。
	ActionRetryProvider
	// ActionEscalate:带类型化原因升级给人(contract / policy / human /
	// unknown;原因词面 = 失败类词面)。
	ActionEscalate
	// ActionStop:无 verdict 终止本次闭环 Run(inconclusive:不修复、不重试、
	// 不算候选失败;人与策略裁决后续)。
	ActionStop
)

// String 返回动作的规范名(诊断/测试用途;与 AC1 的词面一致)。
func (a ClassAction) String() string {
	switch a {
	case ActionRepair:
		return "repair"
	case ActionRetryProvider:
		return "retry_provider"
	case ActionEscalate:
		return "escalate"
	case ActionStop:
		return "stop"
	default:
		return "ClassAction(unknown)"
	}
}

// Action 返回该类的确定性闭环动作(AC1:每类恰好一个)。非法值按 fail-safe
// 落升级(AC2):词表之外不存在第四种命运,绝不修复。
func (c FailureClass) Action() ClassAction {
	switch c {
	case FailureRepairable:
		return ActionRepair
	case FailureEnvironment, FailureProvider:
		return ActionRetryProvider
	case FailureInconclusive:
		return ActionStop
	case FailureContractDefect, FailurePolicyConflict, FailureRequiresHuman, FailureUnknown:
		return ActionEscalate
	default:
		return ActionEscalate
	}
}

// ClassifyError 把一次基础设施/验证执行失败的类型化 error 分类为失败类。
//
// 只经 errors.Is 判因(internal/verification 的类型化哨兵 + context 超时),
// 绝不做消息/stderr 字符串匹配。哨兵 → 类的映射原则:
//
//   - 契约自身缺陷(契约漏声明)→ contract_defect;
//   - §8.7.2 冻结协议纪律与决策锁 12 的违反 → policy_conflict;
//   - 环境/执行边界资源失败与超时 → environment_failure;
//   - 有协议纪律但无可用 verdict → verification_inconclusive;
//   - 保护证据链完整性被破坏 → requires_human_judgment(unmatched risk,
//     §18.2;一次性通道,重试无意义也不被允许);
//   - 其余一切(接线/misuse 哨兵、取消、未知 error)→ unknown fail-safe。
//
// context.Canceled 不是失败:Engine 把取消当一等终态并在端口调用前后先于
// error 检查 ctx,分类原则上见不到它;即便见到也归 unknown(升级),绝不
// 伪装成候选失败。nil 同理(nil 不是失败;调用方应当在确有失败时调用)。
func ClassifyError(err error) FailureClass {
	switch {
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, verification.ErrBudgetExhausted),
		errors.Is(err, verification.ErrNoWorkspace),
		errors.Is(err, verification.ErrEnvironmentMismatch),
		errors.Is(err, verification.ErrInvalidRunOutput):
		return FailureEnvironment
	case errors.Is(err, verification.ErrRuntimeRequirementsUnmet):
		return FailureContractDefect
	case errors.Is(err, verification.ErrRunBeyondStoppingRule),
		errors.Is(err, verification.ErrRunIndexAlreadyRecorded),
		errors.Is(err, verification.ErrSeedMismatch),
		errors.Is(err, verification.ErrBindingAlreadyExecuted):
		return FailurePolicyConflict
	case errors.Is(err, verification.ErrStoppingRuleIncomplete):
		return FailureInconclusive
	case errors.Is(err, verification.ErrHoldoutIntegrity):
		return FailureRequiresHuman
	default:
		return FailureUnknown
	}
}

// ClassifyFinalStatus 把最终验证编排的类型化结论(verification.FinalStatus,
// PRD §36.3)分类为失败类。结论是既成事实,分类只命名、不重算(AC3):
//
//   - FinalFailed:确定性候选失败(可见 binding / required gate 的可信
//     FAIL)→ repairable(PRD §17.4 中唯一默认进入自动修复的类;修复仍受
//     预算治理,反复失败升级 §18.2);
//   - FinalNeedsReview:非 VERIFIED 但需 Control/Review 侧人工/政策裁决
//     (执行不完整、held-out 阻断、独立性覆盖缺口)→ requires_human_judgment
//     (§18.2;绝不伪装成候选失败);
//   - FinalVerified:不是失败,分类不适用 → unknown(其动作仍是升级:分类
//     永不把任何输入变成「接受候选」—— AC3 的另一面);
//   - 非法值:unknown fail-safe。
func ClassifyFinalStatus(s verification.FinalStatus) FailureClass {
	switch s {
	case verification.FinalFailed:
		return FailureRepairable
	case verification.FinalNeedsReview:
		return FailureRequiresHuman
	case verification.FinalVerified:
		return FailureUnknown
	default:
		return FailureUnknown
	}
}

// RulingForClass 把失败类投影为 Router 端口可执行的裁决(Ruling):
//
//   - 仅 repairable → RouteRepair(结构上不可能从任何其他类产出修复);
//   - 其余一切 → RouteEscalate,原因 = 类词面原样透传(Engine 不解释,
//     只执行)。
//
// environment / provider / inconclusive 在结构上不该以可信 FAIL 身份到达
// Router(ADR-0001 §3:只有可信 FAIL 进路由);若组合层解析器仍给出这些类,
// 投影为带类型化原因的升级 —— Engine 骨架的 Router 词表只有 repair/escalate,
// infra/provider 重试路线由 P2-08 在组合层接线(doc.go 归属表)。非法值按
// unknown 投影(fail-safe 升级,AC2)。
func RulingForClass(c FailureClass) Ruling {
	if c == FailureRepairable {
		return Ruling{Route: RouteRepair}
	}
	if !c.IsValid() {
		c = FailureUnknown
	}
	return Ruling{Route: RouteEscalate, Reason: EscalationReason(c)}
}

// ClassResolver 由组合层注入:从其已持有的类型化记录(冻结契约的
// FinalOutcome、端口 error、执行谱系等)解析一次可信 FAIL 的失败类。
// 返回值不在封闭词表内时,Router 按 unknown 处理(fail-safe 升级)。
// 修复永远是「有类型化 repairable 判定才发生」:解析器缺席或不认识失败,
// 落点都是升级,绝不默认修复。
type ClassResolver func(VerifyFailure) FailureClass

// NewFailureRouter 构造 Router 端口的默认实现:解析失败类 → 确定性裁决。
// resolve 可为 nil(没有解析来源时一切失败按 unknown 升级 —— 与 Engine 的
// 「Router 缺席 = router_unavailable 升级」同向,但保留类型化原因)。
func NewFailureRouter(resolve ClassResolver) Router {
	return failureRouter{resolve: resolve}
}

// failureRouter 是 Router 的纯分类实现:同步、无 I/O、确定性。
var _ Router = failureRouter{}

type failureRouter struct {
	resolve ClassResolver
}

// RouteFailure 实现 Router 端口。分类是纯函数,没有可取消的 I/O:ctx 仅
// 承载端口签名,取消纪律由 Engine 在调用前后机械执行(ports.go 契约)。
// 永不返回 error:无法裁决不是基础设施错误,而是 unknown 失败类(→ 升级,
// 原因类型化,比 router_unavailable 更可诊断)。
func (r failureRouter) RouteFailure(_ context.Context, failure VerifyFailure) (Ruling, error) {
	cls := FailureUnknown
	if r.resolve != nil {
		cls = r.resolve(failure)
	}
	return RulingForClass(cls), nil
}
