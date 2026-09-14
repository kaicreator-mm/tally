// acceptguard_test.go 是 P1-06 验收语义评估器的聚焦对抗测试。
//
// 覆盖任务验收:
//   - AC1:required L3 + achieved L1 永不返回可接受(VERIFIED_LIMITED,非 Accept);
//   - AC2:unmatched 风险信号即使保证档满足也绝不静默放行(词面逐字保真);
//   - VERIFIED_LIMITED 语义:waived 与 unmet/satisfied 互斥(INV-07),waived
//     本身使结论只能是 LIMITED;
//   - fail-safe:覆盖观察缺席/未知可见性 = 无事实 = L0,绝不当作已满足;
//     nil 编译权威拒绝评估。
//
// 夹具复用 basis_test.go 的同包构造(basisFrozen/basisCompiled/basisFuncReport):
// 编译权威走真实 Compile→Bind 路径,绝不手构 contract.Compiled。
package taskstate

import (
	"strings"
	"testing"

	"tally/internal/assurance"
	"tally/internal/risk"
	"tally/internal/verification"
)

// guardVisibility 从冻结验证契约提取 binding 可见性(评估器的 Visibility 输入,
// 与生产 CompiledAcceptanceGuard 同一构造规则)。
func guardVisibility(f *verification.Frozen) map[string]verification.VisibilityPolicy {
	c := f.Contract()
	vis := map[string]verification.VisibilityPolicy{}
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			vis[b.ID] = b.VisibilityPolicy
		}
	}
	return vis
}

// mustEvaluate 跑评估器;出错即测试失败。
func mustEvaluate(t *testing.T, in AcceptanceGuardInput) AcceptanceGuardDecision {
	t.Helper()
	d, err := EvaluateAcceptanceGuard(in)
	if err != nil {
		t.Fatalf("EvaluateAcceptanceGuard: %v", err)
	}
	return d
}

// AC 正面控制:零策略零风险(全条款 L1 地板)+ 全绿覆盖观察 ⇒ VERIFIED,
// 可普通接受,无任何缺口面。
func TestAcceptanceGuardPlainVerdictVerified(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	d := mustEvaluate(t, AcceptanceGuardInput{
		Compiled:   c,
		Coverage:   basisFuncReport(f).Acceptances,
		Visibility: guardVisibility(f),
	})
	if !d.Acceptable || d.Verdict != VerdictVerified {
		t.Fatalf("decision = %+v, 期望 VERIFIED 且可接受(L1 地板 + 全绿事实)", d)
	}
	if len(d.UnmetClauses) != 0 || len(d.WaivedRequirements) != 0 || len(d.UnmatchedRiskSignals) != 0 {
		t.Fatalf("全绿候选不得携带缺口面:%+v", d)
	}
	if d.Detail != "" {
		t.Errorf("可接受裁决不应携带明细:%q", d.Detail)
	}
}

// AC1:required L3(命中策略风险上调)+ achieved L1(visible 确定性事实)
// 永不返回可接受 —— VERIFIED_LIMITED,缺口条款与档位词面进入明细。
func TestAcceptanceGuardRequiredL3AchievedL1NeverAcceptable(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, true, risk.SignalSchemaMigration) // 命中策略:无 unmatched,但上调 L3
	for _, cl := range c.Clauses() {
		if cl.Assurance.Required != assurance.LevelL3 {
			t.Fatalf("夹具失真:条款 %s required=%s, 期望 L3", cl.AcceptanceID, cl.Assurance.Required)
		}
	}
	d := mustEvaluate(t, AcceptanceGuardInput{
		Compiled: c,
		Coverage: []verification.AcceptanceCoverage{
			{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"EV-1"}}, // visible → L1
			{AcceptanceID: "AC-02", Covered: true, SatisfiedBy: []string{"EV-2"}}, // held_out → L3
		},
		Visibility: guardVisibility(f),
	})
	if d.Acceptable || d.Verdict != VerdictVerifiedLimited {
		t.Fatalf("decision = %+v, 期望 VERIFIED_LIMITED 且不可接受(AC1)", d)
	}
	if len(d.UnmetClauses) != 1 || d.UnmetClauses[0] != "AC-01" {
		t.Fatalf("UnmetClauses = %v, 期望 [AC-01](缺口条款可见)", d.UnmetClauses)
	}
	for _, want := range []string{"AC-01", "required L3", "achieved L1"} {
		if !strings.Contains(d.Detail, want) {
			t.Errorf("裁决明细缺 %q:%s", want, d.Detail)
		}
	}
	if len(d.UnmatchedRiskSignals) != 0 {
		t.Errorf("命中策略场景不应有 unmatched 信号:%+v", d.UnmatchedRiskSignals)
	}
}

// AC2:unmatched 风险信号即使每条条款的达成档都已满足 required,也绝不静默
// 放行 —— 信号词面逐字进入裁决(INV-V14-09),结论只能是 VERIFIED_LIMITED。
func TestAcceptanceGuardUnmatchedRiskDeniedEvenWhenAssuranceSatisfied(t *testing.T) {
	f := basisFrozen(t, 1)
	const unheard = risk.SignalAuthSymbolChange
	c := basisCompiled(t, f, true, risk.SignalSchemaMigration, unheard)
	if got := c.Coverage().UnmatchedRiskSignals; len(got) != 1 || got[0] != unheard {
		t.Fatalf("夹具失真:UnmatchedRiskSignals = %v", got)
	}
	// 两条款都达成 L3(AC-01 由 held_out binding 共同满足的合成全绿报告)。
	d := mustEvaluate(t, AcceptanceGuardInput{
		Compiled: c,
		Coverage: []verification.AcceptanceCoverage{
			{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"EV-1", "EV-2"}},
			{AcceptanceID: "AC-02", Covered: true, SatisfiedBy: []string{"EV-2"}},
		},
		Visibility: guardVisibility(f),
	})
	if d.Acceptable || d.Verdict != VerdictVerifiedLimited {
		t.Fatalf("decision = %+v, 期望 unmatched 信号阻断普通接受(AC2)", d)
	}
	if len(d.UnmetClauses) != 0 {
		t.Errorf("保证档已满足,不应有 unmet 条款:%v", d.UnmetClauses)
	}
	if len(d.UnmatchedRiskSignals) != 1 || d.UnmatchedRiskSignals[0] != unheard {
		t.Fatalf("UnmatchedRiskSignals = %v, 期望 [%s](词面保真)", d.UnmatchedRiskSignals, unheard)
	}
	if !strings.Contains(d.Detail, "unmatched risk signals [auth_symbol_change]") {
		t.Errorf("裁决明细缺信号词面:%s", d.Detail)
	}
}

// VERIFIED_LIMITED 语义(INV-07):waived 条款与 satisfied/unmet 互斥;waived
// 本身阻断普通接受 —— 豁免是 LIMITED verdict 的触发面,不是静默放行通道。
func TestAcceptanceGuardWaivedIsLimitedDistinctFromUnmet(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	d := mustEvaluate(t, AcceptanceGuardInput{
		Compiled:   c,
		Coverage:   basisFuncReport(f).Acceptances,
		Visibility: guardVisibility(f),
		Waived:     []string{"AC-01"},
	})
	if d.Acceptable || d.Verdict != VerdictVerifiedLimited {
		t.Fatalf("decision = %+v, 期望 waived 阻断普通接受", d)
	}
	if len(d.WaivedRequirements) != 1 || d.WaivedRequirements[0] != "AC-01" {
		t.Fatalf("WaivedRequirements = %v, 期望 [AC-01]", d.WaivedRequirements)
	}
	if len(d.UnmetClauses) != 0 {
		t.Errorf("waived 条款不得混入 unmet(互斥,INV-07):%v", d.UnmetClauses)
	}
	if !strings.Contains(d.Detail, "waived requirements [AC-01]") {
		t.Errorf("裁决明细缺豁免词面:%s", d.Detail)
	}
}

// fail-safe:条款的覆盖观察缺席 = 无验证事实 = L0,绝不当作已满足
// (不冒充 self-check,明细显式标注)。
func TestAcceptanceGuardMissingCoverageFailsSafe(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	d := mustEvaluate(t, AcceptanceGuardInput{
		Compiled: c,
		Coverage: []verification.AcceptanceCoverage{
			{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"EV-1"}},
			// AC-02 缺席观察。
		},
		Visibility: guardVisibility(f),
	})
	if d.Acceptable || d.Verdict != VerdictVerifiedLimited {
		t.Fatalf("decision = %+v, 期望缺席观察 fail-safe 阻断", d)
	}
	if len(d.UnmetClauses) != 1 || d.UnmetClauses[0] != "AC-02" {
		t.Fatalf("UnmetClauses = %v, 期望 [AC-02]", d.UnmetClauses)
	}
	if !strings.Contains(d.Detail, "AC-02") || !strings.Contains(d.Detail, "no verification facts") {
		t.Errorf("裁决明细缺无事实标注:%s", d.Detail)
	}
}

// fail-safe:未知可见性的 binding 不产生任何档位事实(不冒充可信观察)。
func TestAcceptanceGuardUnknownVisibilityYieldsNoFacts(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	d := mustEvaluate(t, AcceptanceGuardInput{
		Compiled: c,
		Coverage: []verification.AcceptanceCoverage{
			{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"EV-GHOST"}},
			{AcceptanceID: "AC-02", Covered: true, SatisfiedBy: []string{"EV-2"}},
		},
		Visibility: guardVisibility(f),
	})
	if d.Acceptable {
		t.Fatalf("未知可见性的事实不得满足条款:%+v", d)
	}
	if len(d.UnmetClauses) != 1 || d.UnmetClauses[0] != "AC-01" {
		t.Fatalf("UnmetClauses = %v, 期望 [AC-01]", d.UnmetClauses)
	}
}

// nil 编译权威拒绝评估:没有权威就没有可裁决的验收面(生产守卫按拒绝处理,
// 绝不默认放行)。
func TestAcceptanceGuardNilCompiledRejected(t *testing.T) {
	if _, err := EvaluateAcceptanceGuard(AcceptanceGuardInput{}); err == nil {
		t.Fatal("nil 编译权威应拒绝评估")
	}
}
