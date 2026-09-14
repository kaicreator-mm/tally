package assurance

import (
	"errors"
	"fmt"
)

// 风险上调的落点:Decision Lock #21 把 L3 定为唯一的风险调整上调档。
const upliftLevel = LevelL3

// 默认地板:Decision Lock #21 —— L1 确定性 visible+protected 检查适用于
// 所有验收条款,不是 universal held-out。
const defaultFloor = LevelL1

// RiskBand — 验收条款的风险档(typed 输入)。上游(P1-07 编译器,经
// policy/risk 领域包)把 PRD §13.3 的 risk signals / unmatched signals /
// 任务类型 / 历史需求翻译成这里的档位;本包不做信号采集与打分。
type RiskBand string

const (
	// RiskMinimal — 最小风险:不触发上调,维持默认地板 L1
	// (Decision Lock #21:held-out 是按风险上调,不是 universal 默认)。
	RiskMinimal RiskBand = "minimal"
	// RiskElevated — 高于最小风险:上调到 L3(Lock #21 risk-adjusted uplift)。
	RiskElevated RiskBand = "elevated"
	// RiskHigh — 高风险:同样上调到 L3;是否需要 L4 human authority 由
	// policy(输入 HumanAuthority)显式声明,不从风险档隐式推导。
	RiskHigh RiskBand = "high"
)

// normalizeRisk 把任何非枚举输入(含空串、未知字符串、无法匹配的信号)归类为
// unknown → fail-safe 上调到更强一侧 L3(INV-V14-08:unknown/unmatched risk
// 不得静默选择最弱保证;PRD §13.4 unmatched signal rule)。
func normalizeRisk(r RiskBand) (RiskBand, bool) {
	switch r {
	case RiskMinimal, RiskElevated, RiskHigh:
		return r, false
	default:
		return RiskBand("unknown"), true
	}
}

// Reason — 选择/达成决策链中的类型化理由(AC4:选择器必须给出决策依据)。
// Reasons 列表按固定求值顺序记录所有被触发(-fired)的规则,即便某条规则
// 随后被更高档位覆盖 —— 决策链是完整依据,不只是最终落点的单一原因。
type Reason string

const (
	// ReasonDefaultFloor — Decision Lock #21 的默认地板 L1(所有条款)。
	ReasonDefaultFloor Reason = "default_floor_l1_bakeoff_a0_03"
	// ReasonRiskUplift — 高于最小风险,按 Lock #21 上调 L3(A0-03:L3\L1=7/12)。
	ReasonRiskUplift Reason = "risk_uplift_l3"
	// ReasonUnmatchedRiskFailsafe — 未知/无法匹配的风险信号,fail-safe 上调 L3,
	// 绝不落到 L0/L1(PRD §13.4 / INV-V14-08)。
	ReasonUnmatchedRiskFailsafe Reason = "unmatched_risk_failsafe_l3"
	// ReasonPolicyFloor — policy 声明的额外地板抬高了必需档。
	ReasonPolicyFloor Reason = "policy_floor"
	// ReasonHumanAuthority — policy 要求 human authority(L4)。
	ReasonHumanAuthority Reason = "human_authority_l4"
	// ReasonJudgeNotCounted — judge 事实存在但按 PRD §13.2 的 L2 定义
	// (judge + 确定性检查)不参与达成档位计算。
	ReasonJudgeNotCounted Reason = "judge_not_counted_without_deterministic"
)

// 达成侧(achieved)的类型化事实理由。
const (
	ReasonFactSelfCheck     Reason = "fact_self_check"         // L0:自检记录
	ReasonFactDeterministic Reason = "fact_deterministic"      // L1:确定性检查已执行
	ReasonFactJudge         Reason = "fact_judge_experimental" // L2:judge+确定性(实验)
	ReasonFactHeldOut       Reason = "fact_held_out"           // L3:held-out verifier 已执行
	ReasonFactHumanApproval Reason = "fact_human_approval"     // L4:human authority 记录
	ReasonFactNone          Reason = "fact_no_verification"    // 无任何验证事实
)

// 选择器类型化错误(errors.Is 可判)。
var (
	// ErrEmptyClause — 条款 id 缺失;拒绝匿名条款的保证决策。
	ErrEmptyClause = errors.New("assurance: 条款 id 不能为空")
	// ErrL2NotSelectable — Decision Lock #21:L2 是实验档,不得作为
	// required/policy 地板被选择;live-judge bake-off 重跑前无本地证据。
	ErrL2NotSelectable = errors.New("assurance: L2 为实验档,不能作为必需档被选择(Decision Lock #21)")
	// ErrInvalidLevel — 越界的档位值。
	ErrInvalidLevel = errors.New("assurance: 非法保证档位")
)

// ClauseInput — 单个验收条款的选择输入(全部 typed,无 map[string]any)。
type ClauseInput struct {
	// ClauseID — 验收条款 id(必填)。
	ClauseID string
	// Risk — 风险档;零值/未知值 fail-safe 上调 L3(见 normalizeRisk)。
	Risk RiskBand
	// PolicyFloor — policy 声明的额外地板;LevelL0 表示未声明。L2 非法
	// (ErrL2NotSelectable)。policy 只能抬高,不能把必需档压到默认地板之下。
	PolicyFloor Level
	// HumanAuthority — policy 明确要求 human authority(L4)。
	HumanAuthority bool
}

// Basis — 选择决策的完整依据(AC4)。字段全部是 typed 值。
type Basis struct {
	// DefaultFloor — 本次决策采用的默认地板(恒为 L1,Lock #21)。
	DefaultFloor Level
	// PolicyFloor — 输入的 policy 地板(LevelL0 = 未声明)。
	PolicyFloor Level
	// Risk — 归一化后的风险档;unknown 表示输入无法匹配已知档位。
	Risk RiskBand
	// RiskUnknown — true 表示触发 unmatched/unknown fail-safe。
	RiskUnknown bool
	// HumanAuthority — policy 是否要求 human authority。
	HumanAuthority bool
	// Reasons — 按固定求值顺序排列的已触发规则(决策链,允许高于最终
	// 落点的中间规则留存;首条恒为 ReasonDefaultFloor)。
	Reasons []Reason
}

// Selection — 一个验收条款的必需保证档位及其依据。required 与 achieved
// 严格区分(AC1):Selection 只表达 required;achieved 由 ExecutedFacts 推导。
type Selection struct {
	ClauseID string
	Required Level
	Basis    Basis
}

// Select 对单个验收条款做确定性档位选择(同输入恒同输出,无随机、无时钟、
// 无外部状态)。
//
// 决策规则(按求值顺序;最终 Required 取各规则候选的最大值):
//
//  1. 默认地板 L1(ReasonDefaultFloor)——所有条款,Lock #21;
//  2. 风险高于 minimal → L3(ReasonRiskUplift);
//     未知/无法匹配风险 → L3(ReasonUnmatchedRiskFailsafe),绝不 L0/L1;
//  3. PolicyFloor > 当前候选 → 抬到 PolicyFloor(ReasonPolicyFloor);
//     PolicyFloor == L2 → ErrL2NotSelectable(实验档,Lock #21);
//  4. HumanAuthority → L4(ReasonHumanAuthority)。
//
// 输出永不低于默认地板 L1,也永不等于 L2(两条不变量由测试穷举保证)。
func Select(in ClauseInput) (Selection, error) {
	if in.ClauseID == "" {
		return Selection{}, ErrEmptyClause
	}
	if in.PolicyFloor != LevelL0 && !in.PolicyFloor.IsValid() {
		return Selection{}, fmt.Errorf("assurance: 条款 %s policy floor %s: %w", in.ClauseID, in.PolicyFloor, ErrInvalidLevel)
	}
	if in.PolicyFloor == LevelL2 {
		return Selection{}, fmt.Errorf("assurance: 条款 %s policy floor L2: %w", in.ClauseID, ErrL2NotSelectable)
	}

	risk, unknown := normalizeRisk(in.Risk)

	basis := Basis{
		DefaultFloor:   defaultFloor,
		PolicyFloor:    in.PolicyFloor,
		Risk:           risk,
		RiskUnknown:    unknown,
		HumanAuthority: in.HumanAuthority,
		Reasons:        []Reason{ReasonDefaultFloor},
	}
	required := defaultFloor

	switch risk {
	case RiskMinimal:
		// 不上调:Lock #21 —— held-out 是按风险上调,不是 universal 默认。
	case RiskElevated, RiskHigh:
		basis.Reasons = append(basis.Reasons, ReasonRiskUplift)
		required = maxLevel(required, upliftLevel)
	default: // unknown(含空串等一切无法匹配的输入)
		basis.Reasons = append(basis.Reasons, ReasonUnmatchedRiskFailsafe)
		required = maxLevel(required, upliftLevel)
	}

	if in.PolicyFloor > required {
		basis.Reasons = append(basis.Reasons, ReasonPolicyFloor)
		required = in.PolicyFloor
	}

	if in.HumanAuthority {
		basis.Reasons = append(basis.Reasons, ReasonHumanAuthority)
		required = maxLevel(required, LevelL4)
	}

	return Selection{ClauseID: in.ClauseID, Required: required, Basis: basis}, nil
}

// maxLevel 返回两档位中较高者。
func maxLevel(a, b Level) Level {
	if a > b {
		return a
	}
	return b
}
