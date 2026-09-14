package engineeringpolicy

import (
	"fmt"
	"slices"
)

// 门禁语义(PRD §8.1 gate_condition / 任务目标「implement gate semantics only」):
//
//   - 输入是证据编译方给出的一组 RequirementFulfillment(requires 键 → 是否
//     被证据满足);本包不拥有证据本身,也不执行任何验证 —— 那是
//     internal/verification / 证据编译(P1-07)的事;
//   - 求值是纯函数:同一策略 + 同一输入 ⇒ 同一 verdict(确定性,无时间、
//     无随机、无外部调用);
//   - 只对 enforcement=gate 的策略可用;guidance 策略请求门禁语义返回
//     ErrNotGate(AC2:guidance 在 v1.4 拿不到阻断权,结构上不可绕过);
//   - fail-safe:输入里的未知键(不在 requires[])报错而不是忽略 —— 静默忽略
//     会让调用方把「没对上」当成「已覆盖」。

// RequirementFulfillment 是一条 requires 键的满足事实,由证据编译方给出。
// Satisfied=false 表示该键当前没有被任何可接受证据满足。
type RequirementFulfillment struct {
	Key       string
	Satisfied bool
}

// GateVerdict 是门禁求值的确定性结论。Satisfied=true 当且仅当
// gate_condition.all_of 的每个键都有一条 Satisfied=true 的 fulfillment;
// 否则 UnmetKeys 按升序列出未满足的键。
type GateVerdict struct {
	Satisfied bool
	UnmetKeys []string
}

// EvaluateGate 按策略的 gate_condition 求值门禁。步骤:
//
//  1. 策略必须自身合法(Validate),否则 ErrInvalidPolicy;
//  2. 策略必须是 enforcement=gate,否则 ErrNotGate(guidance 无阻断权,AC2);
//  3. 输入必须合法:键非空、不重复、且都在 requires[] 内,否则
//     ErrInvalidGateInput(fail-safe,未知键不静默忽略);
//  4. 对 gate_condition.all_of 逐键求值,UnmetKeys 升序输出。
func (p Policy) EvaluateGate(fulfillments []RequirementFulfillment) (GateVerdict, error) {
	if err := p.Validate(); err != nil {
		return GateVerdict{}, err
	}
	if p.Enforcement != EnforcementGate {
		return GateVerdict{}, fmt.Errorf("%w: id=%s enforcement=%q(guidance 在 v1.4 不能获得阻断权)", ErrNotGate, p.ID, string(p.Enforcement))
	}

	byKey := make(map[string]bool, len(fulfillments))
	for i, f := range fulfillments {
		if f.Key == "" {
			return GateVerdict{}, fmt.Errorf("%w: fulfillments[%d] 键为空", ErrInvalidGateInput, i)
		}
		if _, dup := byKey[f.Key]; dup {
			return GateVerdict{}, fmt.Errorf("%w: fulfillments 键 %q 重复", ErrInvalidGateInput, f.Key)
		}
		if !slices.Contains(p.Requires, f.Key) {
			return GateVerdict{}, fmt.Errorf("%w: fulfillments 键 %q 不在 requires[](未知键不静默忽略)", ErrInvalidGateInput, f.Key)
		}
		byKey[f.Key] = f.Satisfied
	}

	var unmet []string
	for _, key := range p.GateCondition.AllOf {
		if !byKey[key] {
			unmet = append(unmet, key)
		}
	}
	slices.Sort(unmet)
	return GateVerdict{Satisfied: len(unmet) == 0, UnmetKeys: unmet}, nil
}
