// acceptguard.go 实现 P1-06 的验收语义纯评估器(PATH_OWNERSHIP:taskstate
// 的验收语义 helper 单一属主):裁决一个 verified-like 候选能否成为**普通**
// Accepted 终态 —— PRD §13.5 / A-03 / DL-07。
//
// 语义与 VerifiedBasis 投影(basis.go)同源:存在未消化缺口(保证缺口、
// 未匹配风险信号、被豁免条款)时,结论词是 VERIFIED_LIMITED,语义上不是
// 普通 VERIFIED —— 不得作为普通 Accepted 放行(fail-safe 到非 Accept 路径,
// 由闭环 Engine 的 accept 守卫执行,本包只做纯裁决):
//
//   - required 取自已编译合同的条款视图(contract.Compiled.Clauses,P1-03/07
//     对 assurance.Select 的原样聚合,含 unmatched fail-safe 上调),本包绝不
//     重算选择;
//   - achieved 经 assurance.Achieved 从功能覆盖观察投影(只有验证事实,Agent
//     claim 永不参与;缺席观察 = 无事实 = L0 + NoVerificationFacts,不冒充
//     self-check);
//   - satisfied 与 waived 永不合并(PRD §13.5 / INV-07):waived 条款即使
//     achieved 已达 required 也进入 waived 面,不得表述为严格满足,且 waived
//     本身就是 VERIFIED_LIMITED 触发面(不得作为普通 Accepted 静默放行);
//   - unknown/unmatched 风险信号原样保真(INV-V14-09):信号存在即不得普通
//     接受,词面逐字进入裁决明细。
//
// 本文件不执行验证、不落任何状态;运行时强制点在 internal/closedloop 的
// AcceptGuard(engine accept 转移),读模型消费归 P1-07。
package taskstate

import (
	"fmt"
	"strings"

	"tally/internal/assurance"
	"tally/internal/contract"
	"tally/internal/risk"
	"tally/internal/verification"
)

// AcceptanceGuardInput 是 EvaluateAcceptanceGuard 的全部输入(全部 typed)。
// Compiled 是必填权威;Coverage/Visibility 是同一候选的验证观察面。
type AcceptanceGuardInput struct {
	// Compiled 是被验收候选对应的已编译合同(必需档与未匹配风险的唯一权威
	// 视图)。nil 拒绝评估:没有权威就没有可裁决的验收面(fail-safe)。
	Compiled *contract.Compiled
	// Coverage 是验证编排为该候选记录的功能覆盖观察
	// (verification.FinalOutcome.Functional.Acceptances 的原样投影)。
	// 条款缺席 = 无观察事实,按 L0 fail-safe 处理,绝不当作已满足。
	Coverage []verification.AcceptanceCoverage
	// Visibility 是 binding id → 可见性策略(冻结验证契约;与 Compiled 同一
	// 快照)。未知 binding 不产生任何档位事实(不冒充可信观察)。
	Visibility map[string]verification.VisibilityPolicy
	// Waived 是经有效 Independence Waiver 放行的 acceptance id(升序)。
	// v1.4.1 闭环内没有豁免通道(运行时传空);读模型侧可传入。waived 与
	// satisfied 永不合并,且 waived 本身使结论只能是 VERIFIED_LIMITED。
	Waived []string
}

// AcceptanceGuardDecision 是验收守卫的纯裁决结果(值,不是 error)。
type AcceptanceGuardDecision struct {
	// Verdict 是验证结论词表:VERIFIED(无缺口)或 VERIFIED_LIMITED(存在
	// 任何未消化缺口)。NOT_VERIFIED 不在此出现 —— 守卫只在 verified-like
	// 候选的 accept 转移上求值。
	Verdict Verdict
	// Acceptable 报告是否允许普通 Accept(Accepted 终态)。恒等价于
	// Verdict == VerdictVerified:VERIFIED_LIMITED 语义的候选不得作为普通
	// Accepted 放行(A-03 / DL-07)。
	Acceptable bool
	// UnmetClauses 是 achieved 档位低于 required 档位的未豁免条款 id(升序)。
	UnmetClauses []string
	// WaivedRequirements 是被有效 Waiver 放行的条款 id(升序;与
	// UnmetClauses/Satisfied 互斥,INV-07)。
	WaivedRequirements []string
	// UnmatchedRiskSignals 是未被任何命中策略声明的风险信号(词表原样保真,
	// INV-V14-09)。
	UnmatchedRiskSignals []risk.SignalID
	// Detail 是人类可读的裁决明细(拒绝时非空,确定性词面;输入里不存在
	// held-out 内容,明细不构成泄漏面)。可接受时为空。
	Detail string
}

// EvaluateAcceptanceGuard 对一个 verified-like 候选做验收语义纯评估:同输入
// 恒同输出,无时钟、无随机、无外部状态。缺口判定与 VerifiedBasis 投影
// (basis.go)的 VERIFIED_LIMITED 触发面完全一致 —— 全仓只有这一处验收语义
// (Forbidden boundary:不复制 assurance 数学,required/achieved 计算原样
// 委托 internal/assurance 的 Select 产物与 Achieved)。
func EvaluateAcceptanceGuard(in AcceptanceGuardInput) (AcceptanceGuardDecision, error) {
	if in.Compiled == nil {
		return AcceptanceGuardDecision{}, fmt.Errorf("taskstate: 验收守卫缺少已编译合同,拒绝评估(fail-safe)")
	}
	clauses := in.Compiled.Clauses()
	if len(clauses) == 0 {
		// 冻结校验保证 ≥1 条验收条款;空权威按缺口处理(不得静默放行)。
		return AcceptanceGuardDecision{
			Verdict:    VerdictVerifiedLimited,
			Acceptable: false,
			Detail:     "acceptance guard: compiled contract has no acceptance clauses",
		}, nil
	}

	// 覆盖观察按 acceptance id 索引(报告条目与编译条款同源:编译器按冻结
	// 契约的 acceptance 集逐一编译;额外条目不参与裁决)。
	cov := make(map[string]verification.AcceptanceCoverage, len(in.Coverage))
	for _, ac := range in.Coverage {
		cov[ac.AcceptanceID] = ac
	}
	waived := make(map[string]bool, len(in.Waived))
	for _, id := range in.Waived {
		waived[id] = true
	}

	d := AcceptanceGuardDecision{
		Verdict:              VerdictVerified,
		UnmatchedRiskSignals: cloneRiskSignals(in.Compiled.Coverage().UnmatchedRiskSignals),
	}
	var parts []string
	var waivedIDs []string

	for _, cl := range clauses {
		id := cl.AcceptanceID
		switch {
		case waived[id]:
			// waived 与 satisfied 永不合并(INV-07):即使 achieved 已达
			// required 也不得表述为严格满足;豁免本身使结论只能是 LIMITED。
			waivedIDs = append(waivedIDs, id)
		default:
			facts := clauseFacts(cov[id], in.Visibility)
			level, basis := assurance.Achieved(facts)
			if !level.Satisfies(cl.Assurance.Required) {
				d.UnmetClauses = append(d.UnmetClauses, id)
				note := ""
				if basis.NoVerificationFacts {
					note = " (no verification facts)"
				}
				parts = append(parts, fmt.Sprintf("assurance gap [%s required %s achieved %s%s]",
					id, cl.Assurance.Required, level, note))
			}
		}
	}
	if len(waivedIDs) > 0 {
		d.WaivedRequirements = waivedIDs
		parts = append(parts, "waived requirements ["+strings.Join(waivedIDs, " ")+"]")
	}
	if len(d.UnmatchedRiskSignals) > 0 {
		sigs := make([]string, len(d.UnmatchedRiskSignals))
		for i, s := range d.UnmatchedRiskSignals {
			sigs[i] = string(s)
		}
		parts = append(parts, "unmatched risk signals ["+strings.Join(sigs, " ")+"]")
	}

	d.Acceptable = len(d.UnmetClauses) == 0 && len(d.WaivedRequirements) == 0 && len(d.UnmatchedRiskSignals) == 0
	if !d.Acceptable {
		d.Verdict = VerdictVerifiedLimited
		d.Detail = "acceptance guard: " + strings.Join(parts, "; ")
	}
	return d, nil
}

// clauseFacts 把一条功能覆盖观察 + binding 可见性投影为已执行验证事实
// (与 basis.go 的达成事实映射同一规则:visible binding 满足 → 确定性检查
// 已执行(L1 特征事实);held_out binding 满足 → protected/held-out verifier
// 已执行(L3 特征事实)。可见性未知/缺席的 binding 不产生任何事实 —— 不冒充
// 可信观察,缺席观察自然落入 L0 fail-safe。
func clauseFacts(ac verification.AcceptanceCoverage, visibility map[string]verification.VisibilityPolicy) assurance.ExecutedFacts {
	var facts assurance.ExecutedFacts
	if !ac.Covered {
		return facts
	}
	for _, bid := range ac.SatisfiedBy {
		switch visibility[bid] {
		case verification.VisibilityHeldOut:
			facts.HeldOut = true
		case verification.VisibilityVisible:
			facts.Deterministic = true
		}
	}
	return facts
}
