// repair_policy.go 实现 P2-07:外层修复的**有界 RepairPolicy**(PRD §17.2
// 自动 Repair 上限、§17.3 硬约束、§18.2 超限升级;ADR-0001 §1/§6;
// DECISION_LOCKS 7/23)。它同时是 P2-02 骨架在 ports.go 预留的 RepairBudget
// 端口的策略实现:Engine 在裁决为修复后、开始新 Run 前咨询本策略。
//
// 纪律:
//
//   - 内外预算互不相通(锁 7 / ADR-0001 §1):本类型与 internal/policy 的
//     ExecutionPolicy.Retry.MaxIterations 没有任何类型、字段或导入层面的
//     关系 —— 本文件不 import internal/policy,不出现内层迭代词汇(测试守卫
//     钉住);MaxRepairAttempts 只治理「验证驱动的返工」,内层迭代耗尽不消耗
//     本预算,本预算也不延长内层迭代;
//   - 默认 2 是**策略默认值**,不是架构常量(锁 23 / ADR-0001 §6):可按策略
//     修订配置,但必须落在硬界 [MinRepairAttempts, HardMaxRepairAttempts] 内
//     —— 架构不变量是「外层循环有限」,具体数值归策略(AC1/AC2);
//   - 零/负/超界一律拒绝(AC3):零与负数**不表示无界**(与「0 = 不限次」的
//     常见惯例刻意相反),超过硬上限的近无界配置同样拒绝;非法策略的落点是
//     fail-safe「永不修复」,绝不静默放宽;
//   - 预算是**位置式**的,不是累计计数器:CanRepair 是尝试序号的纯函数,
//     对 (task, attempt_no) 幂等(ADR-0001 §2 的尝试幂等键)—— 崩溃后的修复
//     尝试按键恢复,不存在重复扣减;取消与 Inconclusive 的尝试不推进序号,
//     结构上就不消耗预算(Engine 落点已强制,P2-02;本策略不重复记账);
//   - 耗尽即显式升级(AC4):预算拒绝下一次修复时,Engine 落
//     EscalationRepairExhausted 终态(result.go 的类型化原因词表),绝不循环、
//     绝不静默改道、绝不自动接受。
package closedloop

import (
	"errors"
	"fmt"
)

// ErrInvalidRepairPolicy:RepairPolicy 值非法(零/负数/超过硬上限)。调用方用
// errors.Is 判因,包装文案只提供字段级上下文。
var ErrInvalidRepairPolicy = errors.New("closedloop: invalid repair policy")

// 修复预算的硬界(策略常量;AC1「独立可配,但必须落在硬界内」)。
const (
	// MinRepairAttempts 是 MaxRepairAttempts 的硬下限:至少 1 次 —— 预算必须
	// 为正,零/负数不表示无界(AC3)。
	MinRepairAttempts = 1
	// DefaultMaxRepairAttempts 是 PRD §17.2 的 MVP 默认:2 次自动修复
	// (初始尝试之外的第 2、3 次尝试,超过后 Escalate)。锁 23:这是策略
	// 默认值,不是架构常量(AC2)。
	DefaultMaxRepairAttempts = 2
	// HardMaxRepairAttempts 是硬上限:任何配置不得超过 —— 「外层循环有限」
	// (ADR-0001 §6)由类型校验保证,不靠调用方自觉(AC3)。上限的具体数值
	// 是策略常量,调高它属于策略修订,必须保留有限性。
	HardMaxRepairAttempts = 10
)

// RepairPolicy 是外层修复的有界策略值(AC1)。MaxRepairAttempts 是**修复尝试**
// 的次数上限(verification-driven rework,锁 7 词表):初始尝试(AttemptNo=1)
// 不归它治理(Engine 无条件启动首次尝试),第 2..MaxRepairAttempts+1 次尝试
// 是修复,用尽后升级。
//
// 它与内层 ExecutionPolicy.Retry.MaxIterations 零耦合(锁 7/23):没有共享
// 类型、共享字段或共享导入;本类型只依赖本包的 AttemptNo 词表。按值使用;
// 构造后必须通过 Validate 才能视为有效,非法值的落点是拒绝一切修复(fail-safe)。
type RepairPolicy struct {
	// MaxRepairAttempts 是允许的修复尝试次数。必须落在
	// [MinRepairAttempts, HardMaxRepairAttempts] 内(AC1/AC3)。
	MaxRepairAttempts int
}

// RepairPolicy 实现 P2-02 预留的 RepairBudget 端口(编译期钉住;组合根把
// 本值作为 Config.RepairBudget 注入,P2-08 接线)。
var _ RepairBudget = RepairPolicy{}

// DefaultRepairPolicy 返回 PRD §17.2 的默认修复策略:2 次自动修复(AC2)。
func DefaultRepairPolicy() RepairPolicy {
	return RepairPolicy{MaxRepairAttempts: DefaultMaxRepairAttempts}
}

// NewRepairPolicy 构造并校验一个 RepairPolicy;越界值返回包装
// ErrInvalidRepairPolicy 的错误与零值策略(AC1/AC3)。
func NewRepairPolicy(maxRepairAttempts int) (RepairPolicy, error) {
	p := RepairPolicy{MaxRepairAttempts: maxRepairAttempts}
	if err := p.Validate(); err != nil {
		return RepairPolicy{}, err
	}
	return p, nil
}

// Validate 校验修复预算落在硬界内(AC3):零/负数拒绝 —— 它们不表示无界,
// 预算必须有限且为正;超过 HardMaxRepairAttempts 拒绝 —— 无界/近无界的修复
// 循环在类型上不可表达(ADR-0001 §6:有限是不变量)。
func (p RepairPolicy) Validate() error {
	if p.MaxRepairAttempts < MinRepairAttempts {
		return fmt.Errorf("%w: max_repair_attempts=%d, 必须 >=%d(零/负数不表示无界,修复预算必须有限且为正)",
			ErrInvalidRepairPolicy, p.MaxRepairAttempts, MinRepairAttempts)
	}
	if p.MaxRepairAttempts > HardMaxRepairAttempts {
		return fmt.Errorf("%w: max_repair_attempts=%d 超过硬上限 %d(外层修复必须有限)",
			ErrInvalidRepairPolicy, p.MaxRepairAttempts, HardMaxRepairAttempts)
	}
	return nil
}

// CanRepair 实现 RepairBudget 端口(P2-02 缝,本任务侧实现)。Engine 在可信
// FAIL 被裁决为修复后、开始新 Run 前以「下一次尝试序号」no 咨询一次;按端口
// 契约 no 恒 ≥2。
//
// 语义:第 no 次尝试是第 no-1 次修复;允许当且仅当 no ∈ [2, MaxRepairAttempts+1]
// —— 默认 2 允许尝试 2、3,拒绝尝试 4(耗尽点恰好落在第 2 次修复失败之后,
// off-by-one 由该闭区间表达)。
//
// fail-safe:策略非法(如零值策略)或 no 不是 ≥2 的修复序号 → false(永不
// 修复),绝不猜测。纯函数:无状态、无时钟,对同一 no 恒定 —— 取消与
// Inconclusive 的尝试不推进序号,结构上不消耗预算(与 Engine 落点强制对齐,
// 不重复记账;耗尽落点 EscalationRepairExhausted 归 Engine,AC4)。
func (p RepairPolicy) CanRepair(no AttemptNo) bool {
	if err := p.Validate(); err != nil {
		return false
	}
	if !no.IsValid() || !no.IsRepair() {
		return false
	}
	return int(no) <= p.MaxRepairAttempts+1
}
