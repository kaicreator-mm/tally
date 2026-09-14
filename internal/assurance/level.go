// Package assurance 实现 PRD §13(Assurance Levels / Level selection /
// unmatched signal rule / satisfied 与 waived 永不合并)与 PRD §16(bake-off
// 结果 → 默认映射)的保证领域模型(P1-06)。
//
// 默认映射来自 A0-03 bake-off 决策(Decision Lock #21,A0-06 冻结):
//
//   - 默认地板:L1(确定性 visible+protected 检查)适用于所有验收条款;
//   - 风险上调:L3(protected/held-out verifier)是任何高于最小风险条款的
//     风险调整上调(A0-03 冻结语料 v2 sha256 e83b8bd3…:L3\L1 = 7/12 unique
//     catches;L1 alone 只捕获 5/12);
//   - L2(独立 LLM judge)保持实验档,不是默认可选档:bake-off 中 L2 judge
//     NOT_RUN(no judge credentials),L2 捕获只是确定性下界,无本地证据支持
//     其作为默认映射。默认选择器永不输出 L2(见 select.go 的类型化错误)。
//
// 选择器对 typed 输入(ClauseInput)做纯确定性计算;任务类型/历史需求等
// PRD §13.3 信号由上游(P1-07 编译器,经 policy/risk 领域包)翻译为
// RiskBand 或 policy 地板后进入本包,本包不做信号采集。
//
// 边界(P1):
//   - required 与 achieved 是两个不同的类型化值(AC1);satisfied 与 waived
//     是两个不同的结论状态,永不合并(PRD §13.5 / AC2);
//   - 本包只做纯领域计算:不执行任何验证、不改 internal/verification 的执行
//     语义、不持有存储;生产接线归 P1-07(编译器)与 P1-08(VerifiedBasis 投影)。
package assurance

import (
	"fmt"
)

// Level — PRD §13.2 的单轴保证档位。v1.4 用单轴简化实现;L4(human
// authority)在数学意义上并不"强于"L3,只是 v1.4 把 human authority 折叠进
// 同一根轴(PRD §13.2 说明),因此比较语义按轴序定义,调用方不应把
// "L4 >= L3" 解读为 L4 覆盖 L3 的捕获能力。
type Level int

const (
	// LevelL0 — self-check(实现侧自检;Agent done 属于 Claim,不是 Delivery Fact)。
	LevelL0 Level = iota
	// LevelL1 — 确定性工具:visible+protected 确定性检查(build/test/lint/
	// protected-file 变更检测;Decision Lock #21 的默认地板)。
	LevelL1
	// LevelL2 — L1 + 独立 LLM judge。实验档(Decision Lock #21):在真实
	// live-judge bake-off 重跑之前,不得作为默认/必需档被选择。
	LevelL2
	// LevelL3 — protected/held-out verifier;Decision Lock #21 的风险调整上调档。
	LevelL3
	// LevelL4 — human authority required(PRD §13.2)。
	LevelL4
)

// String 返回档位的规范名("L0"…"L4")。
func (l Level) String() string {
	switch l {
	case LevelL0:
		return "L0"
	case LevelL1:
		return "L1"
	case LevelL2:
		return "L2"
	case LevelL3:
		return "L3"
	case LevelL4:
		return "L4"
	default:
		return fmt.Sprintf("Level(%d)", int(l))
	}
}

// IsValid 报告 l 是否是 L0..L4 中的合法档位。
func (l Level) IsValid() bool {
	return l >= LevelL0 && l <= LevelL4
}

// ParseLevel 解析规范名("L0"…"L4");其余输入返回错误。
func ParseLevel(s string) (Level, error) {
	switch s {
	case "L0":
		return LevelL0, nil
	case "L1":
		return LevelL1, nil
	case "L2":
		return LevelL2, nil
	case "L3":
		return LevelL3, nil
	case "L4":
		return LevelL4, nil
	default:
		return LevelL0, fmt.Errorf("assurance: 非法保证档位 %q(合法值 L0..L4)", s)
	}
}

// Satisfies 按单轴语义报告 achieved 档位是否满足 required 档位。
// 单轴折叠意味着 L4 满足一切 <= L4 的要求(PRD §13.2 简化说明)。
func (l Level) Satisfies(required Level) bool {
	return l >= required
}
