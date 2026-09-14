// 本文件实现 P1-08:PRD §14 的 VerifiedBasis 投影(Decision Lock 12:
// VerifiedBasis is derived —— 扩展现有 taskstate 只读模型,不新增任何可变
// 生命周期状态表、状态机或 setter)。
//
// 投影纪律:
//
//   - 纯派生:与 Derive 同一事实集 ⇒ 完全相同的 VerifiedBasis(INV-STATE-01);
//     全部列表确定性排序,无 map[string]any,无随机、无时钟;
//   - 只消费既有 seam,不重新实现策略匹配或保证选择:required 档取自
//     contract.Compiled 的条款视图(P1-07 编译器对 assurance.Select 的原样
//     聚合),achieved 档经 assurance.Achieved 从 taskstate 已观察的验证事实
//     推导(P1-06 seam);策略覆盖/matched policies/unmatched 风险信号取自
//     Compiled.Coverage()(P1-05 seam 原样投影);
//   - satisfied 与 waived 永不合并(PRD §13.5 / INV-07):一个条款只出现在
//     两个列表之一;waiver 语义与 v1.3 Independence Waiver(§8.10)同源,
//     不引入第二种 waiver 权威;
//   - VERIFIED 不能隐藏缺口(AC2 / INV-11):存在未匹配风险信号、被豁免条款
//     或保证缺口(achieved < required 的未豁免条款)时,verdict 必须是
//     VERIFIED_LIMITED,不得显示成普通 VERIFIED(PRD §14)。
package taskstate

import (
	"tally/internal/assurance"
	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/verification"
)

// Verdict 是 PRD §14 的验证结论词表。VERIFIED_LIMITED 语义上不是普通
// VERIFIED:调用方(UI/报表)必须将其区分为 limited verdict 展示。
type Verdict string

const (
	// VerdictNone:未接入 v1.4 投影输入(Snapshot.Compiled == nil 时
	// Result.VerifiedBasis 为 nil,而不是携带 VerdictNone 的空投影)。
	VerdictNone Verdict = ""
	// VerdictVerified:验证门全部满足,且无未匹配风险、无被豁免条款、
	// 无保证缺口。
	VerdictVerified Verdict = "VERIFIED"
	// VerdictVerifiedLimited:验证门满足,但存在未匹配风险信号、被豁免条款
	// 或达成档位低于必需档位的未豁免条款(PRD §14 第二例;PRD §10 显式
	// limited verdict)。不得显示成普通 VERIFIED。
	VerdictVerifiedLimited Verdict = "VERIFIED_LIMITED"
	// VerdictNotVerified:主状态未达 VERIFIED(Gaps 说明原因)。
	VerdictNotVerified Verdict = "NOT_VERIFIED"
)

// MatchedPolicy 是 matched_policies[] 的一条投影:命中策略 id 及其内容修订
// pin(身份可追溯,INV-STATE-02)。按 ID 升序(P1-05 seam 的输出顺序)。
type MatchedPolicy struct {
	ID          string
	Revision    string
	Enforcement engineeringpolicy.Enforcement
}

// EvidenceCoverage 是 PRD §14 的 evidence_coverage:已编译合同要求面的
// acceptance 级功能覆盖计数(数据来自适用的 FunctionalCoverageReport;
// 报告缺失或绑定其它 revision 时 §8.8.2 对当前状态不可见,此时
// FunctionalReportMatched=false 且全部条款计入 Uncovered)。
type EvidenceCoverage struct {
	// FunctionalReportMatched 报告功能覆盖报告是否绑定当前冻结契约。
	FunctionalReportMatched bool
	// Covered 是功能覆盖成立的条款数。
	Covered int
	// Total 是已编译合同的条款总数(分母,required 视图)。
	Total int
	// Uncovered 是未覆盖的 acceptance id,升序。
	Uncovered []string
}

// Ratio 返回覆盖比例(Covered/Total);Total==0 时返回 (0, false)。
func (c EvidenceCoverage) Ratio() (float64, bool) {
	if c.Total == 0 {
		return 0, false
	}
	return float64(c.Covered) / float64(c.Total), true
}

// PolicyCoverage 是 PRD §14 的 policy_coverage(P1-05 匹配覆盖投影的计数与
// 信号面;PRD §10:confidence 与 unknown 信号必须原样可见)。
type PolicyCoverage struct {
	// MatchedCount 是命中策略数(明细见 VerifiedBasis.MatchedPolicies)。
	MatchedCount int
	// Confidence 是匹配覆盖置信度(none/partial/full)。
	Confidence engineeringpolicy.MatchConfidence
	// DetectedRiskSignals 是已检测的冻结枚举内风险信号,升序。
	DetectedRiskSignals []risk.SignalID
	// UnknownRiskSignals 是冻结枚举之外的信号(原样保真,升序;
	// INV-V14-09:unknown 永不丢弃)。
	UnknownRiskSignals []risk.SignalID
}

// ResidualRisk 是 PRD §14 的 residual_risk:satisfied/waived 之外的残余风险
// 面。未匹配风险信号有独立字段(UnmatchedRiskSignals),不在此重复。
type ResidualRisk struct {
	// UnmetClauses 是既未严格满足、也无有效 Waiver 的条款(达成档位低于
	// 必需档位),升序。
	UnmetClauses []string
	// UnresolvedViolations 是 unresolved protected/holdout integrity 违规 id,
	// 升序(不可被 Waiver 覆盖,§8.10)。
	UnresolvedViolations []string
	// BlockingFailures 是当前候选 applicable unresolved failed evidence 的
	// fact id,升序(§8.8.1:新 PASS 不抹掉合法 FAIL)。
	BlockingFailures []string
}

// VerifiedBasis 是 PRD §14 的 v1.4 验证结论投影。它是纯派生只读模型
// (Decision Lock 12):由 Derive 从同一 Snapshot 确定性产出,不是新的状态
// 权威;没有 setter,唯一产出路径是 Derive。
type VerifiedBasis struct {
	// Verdict 是验证结论(VERIFIED / VERIFIED_LIMITED / NOT_VERIFIED)。
	Verdict Verdict
	// AssuranceRequired 是全合同必需保证档(条款必需档的最大值;weakest
	// link 语义由 achieved 侧表达)。
	AssuranceRequired assurance.Level
	// AssuranceAchieved 是达成保证档(条款达成档的最小值:聚合结论按最弱
	// 条款诚实上报,不取最大值冒充)。
	AssuranceAchieved assurance.Level
	// EvidenceCoverage 是 required evidence 覆盖(PRD §14 evidence_coverage)。
	EvidenceCoverage EvidenceCoverage
	// PolicyCoverage 是策略匹配覆盖(PRD §14 policy_coverage)。
	PolicyCoverage PolicyCoverage
	// MatchedPolicies 是命中策略明细(matched_policies[]),按 ID 升序。
	MatchedPolicies []MatchedPolicy
	// UnmatchedRiskSignals 是未被任何命中策略声明的风险信号
	// (unmatched_risk_signals,升序;PRD §13.4 / INV-V14-09 触发面)。
	UnmatchedRiskSignals []risk.SignalID
	// SatisfiedRequirements 是严格满足的 acceptance id(satisfied_requirements,
	// 升序;与 WaivedRequirements 互斥,PRD §13.5 / INV-07)。
	SatisfiedRequirements []string
	// WaivedRequirements 是经有效 Independence Waiver 放行的 acceptance id
	// (waived_requirements,升序;与 SatisfiedRequirements 互斥)。
	WaivedRequirements []string
	// ResidualRisk 是 satisfied/waived 之外的残余风险(residual_risk)。
	ResidualRisk ResidualRisk
	// CompiledHash 是投影所依据的已编译合同身份哈希(sha256:<hex>);同一
	// 哈希 ⇒ 同一 required/coverage 视图,结论可单独复算(INV-STATE-02)。
	CompiledHash string
}

// basisInput 是 projectVerifiedBasis 的全部输入(全部是 Derive 已推导的中间
// 值;显式传参保持投影函数可独立单测的纯函数性质)。
type basisInput struct {
	compiled             *contract.Compiled
	contract             *verification.Frozen // 已由 Derive 校验与 compiled 同一内容修订
	functional           *verification.FunctionalCoverageReport
	functionalMatched    bool
	state                State
	waivedAcceptances    []string // 有效 Waiver 放行的 acceptance id(升序)
	unresolvedViolations []string // 升序
	blockingFailures     []string // 升序
}

// projectVerifiedBasis 从已编译合同与 taskstate 已观察的验证事实确定性投影
// VerifiedBasis。不修改任何输入;不重新实现匹配/选择(seam 原样消费)。
func projectVerifiedBasis(in basisInput) *VerifiedBasis {
	cov := in.compiled.Coverage()
	clauses := in.compiled.Clauses()

	// binding id → visibility(冻结契约是 visibility 的唯一权威;identity 一致
	// 性已由 Derive 的 ErrCompiledMismatch 校验保证)。
	visibility := map[string]verification.VisibilityPolicy{}
	if in.contract != nil {
		fc := in.contract.Contract()
		for i := range fc.Acceptances {
			for _, b := range fc.Acceptances[i].EvidenceBindings {
				visibility[b.ID] = b.VisibilityPolicy
			}
		}
	}

	// 适用功能覆盖报告的 acceptance 视图(§8.8.2:绑定其它 revision 的报告
	// 对当前状态不可见)。
	functional := map[string]verification.AcceptanceCoverage{}
	if in.functionalMatched && in.functional != nil {
		for _, ac := range in.functional.Acceptances {
			functional[ac.AcceptanceID] = ac
		}
	}

	waivedSet := make(map[string]bool, len(in.waivedAcceptances))
	for _, id := range in.waivedAcceptances {
		waivedSet[id] = true
	}

	out := &VerifiedBasis{
		CompiledHash: in.compiled.Hash(),
		EvidenceCoverage: EvidenceCoverage{
			FunctionalReportMatched: in.functionalMatched && in.functional != nil,
			Total:                   len(clauses),
		},
	}
	// 条款必需档的最大值与达成档的最小值(weakest link)。
	requiredMax := assurance.LevelL0
	achievedMin := assurance.LevelL0

	for i, cl := range clauses {
		// 达成事实映射(只有 taskstate 已观察的验证事实,Agent claim 永不参与):
		//   - 功能覆盖由 visible binding 满足 → deterministic 检查已执行
		//     (Lock #21 的 L1 特征事实,protected 与否不改变档位);
		//   - 由 held_out binding 满足 → protected/held-out verifier 已执行
		//     (L3 特征事实)。
		var facts assurance.ExecutedFacts
		ac, acOK := functional[cl.AcceptanceID]
		if acOK && ac.Covered {
			out.EvidenceCoverage.Covered++
			for _, bid := range ac.SatisfiedBy {
				switch visibility[bid] {
				case verification.VisibilityHeldOut:
					facts.HeldOut = true
				case verification.VisibilityVisible:
					facts.Deterministic = true
				}
			}
		} else {
			out.EvidenceCoverage.Uncovered = append(out.EvidenceCoverage.Uncovered, cl.AcceptanceID)
		}
		// achieved 经 P1-06 seam 推导(无事实 → L0 + NoVerificationFacts,
		// 不冒充 self-check)。
		level, _ := assurance.Achieved(facts)

		// satisfied 与 waived 互斥(INV-07):先判 waived(§8.10 的 waiver
		// 路径,即使 achieved 已达 required 也不得表述为 Strict 已满足,
		// 与 VerificationClass 的 strict_waived 语义一致)。
		switch {
		case waivedSet[cl.AcceptanceID]:
			out.WaivedRequirements = append(out.WaivedRequirements, cl.AcceptanceID)
		case level.Satisfies(cl.Assurance.Required):
			out.SatisfiedRequirements = append(out.SatisfiedRequirements, cl.AcceptanceID)
		default:
			out.ResidualRisk.UnmetClauses = append(out.ResidualRisk.UnmetClauses, cl.AcceptanceID)
		}

		if i == 0 || cl.Assurance.Required > requiredMax {
			requiredMax = cl.Assurance.Required
		}
		if i == 0 || level < achievedMin {
			achievedMin = level
		}
	}
	out.AssuranceRequired = requiredMax
	out.AssuranceAchieved = achievedMin

	// 策略覆盖与 matched policies(P1-05 seam 原样投影;已按 id 升序,这里
	// 只做拷贝)。
	out.PolicyCoverage = PolicyCoverage{
		MatchedCount:        len(cov.MatchedPolicies),
		Confidence:          cov.Confidence,
		DetectedRiskSignals: cloneRiskSignals(cov.DetectedRiskSignals),
		UnknownRiskSignals:  cloneRiskSignals(cov.UnknownRiskSignals),
	}
	for _, m := range cov.MatchedPolicies {
		out.MatchedPolicies = append(out.MatchedPolicies, MatchedPolicy{
			ID:          m.Policy.ID,
			Revision:    m.Policy.Revision,
			Enforcement: m.Policy.Enforcement,
		})
	}
	out.UnmatchedRiskSignals = cloneRiskSignals(cov.UnmatchedRiskSignals)

	// 残余风险(输入列表已排序,拷贝即可)。
	out.ResidualRisk.UnresolvedViolations = cloneStrList(in.unresolvedViolations)
	out.ResidualRisk.BlockingFailures = cloneStrList(in.blockingFailures)

	// verdict(AC2 / INV-11):verified-like 状态下存在任何未消化缺口
	// (unmatched 风险、waived 条款、保证缺口)必须是 VERIFIED_LIMITED;
	// 主状态未达 VERIFIED 一律 NOT_VERIFIED。
	if in.state == StateVerified || in.state == StateApproved || in.state == StateMerged {
		if len(out.UnmatchedRiskSignals) > 0 || len(out.WaivedRequirements) > 0 ||
			len(out.ResidualRisk.UnmetClauses) > 0 {
			out.Verdict = VerdictVerifiedLimited
		} else {
			out.Verdict = VerdictVerified
		}
	} else {
		out.Verdict = VerdictNotVerified
	}
	return out
}

// cloneRiskSignals 返回升序信号切片的拷贝;nil 保持 nil。
func cloneRiskSignals(in []risk.SignalID) []risk.SignalID {
	if in == nil {
		return nil
	}
	out := make([]risk.SignalID, len(in))
	copy(out, in)
	return out
}

// cloneStrList 返回字符串切片的拷贝;nil 保持 nil。
func cloneStrList(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}
