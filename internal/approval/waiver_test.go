package approval

import (
	"errors"
	"testing"
	"time"

	"tally/internal/core"
	"tally/internal/evidence"
)

// 测试固定量:全部断言基于确定性输入,不依赖墙钟。
var (
	testProject = core.ProjectID("proj-1")
	testTask    = "TASK-023"
	testCS      = "cs-abc123"
	testAt      = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	testZone    = time.FixedZone("UTC+2", 2*60*60)
)

// validWaiverSpec 返回一份完整合法的豁免 provenance 输入(运行期损失)。
func validWaiverSpec() WaiverSpec {
	return WaiverSpec{
		Category: "auth_security",
		Gaps: []CoverageGap{
			{AcceptanceID: "AC-02"},
			{AcceptanceID: "AC-05"},
		},
		CompromisedAcceptances: []string{"AC-02"},
		CompromisedBindings: []evidence.BindingRef{
			{Project: testProject, Verification: "sha256:contract", BindingID: "EV-AC02-01"},
		},
		ExposureRefs: []evidence.ExposureID{"exp-1"},
		ReasonCode:   "holdout_exposure_accepted",
		AcceptedBy:   "owner-7",
		AcceptedAt:   testAt,
	}
}

// mustWaiver 构造失败即 Fatal 的辅助。
func mustWaiver(t *testing.T, spec WaiverSpec) *IndependenceWaiver {
	t.Helper()
	w, err := NewWaiver(spec)
	if err != nil {
		t.Fatalf("NewWaiver: 意外错误: %v", err)
	}
	return w
}

func TestNewWaiverRecordsFullProvenance(t *testing.T) {
	// AC1:Waiver 列出 compromised acceptances/bindings/exposure refs/
	// coverage gap/reason,并记录 owner risk acceptance。
	w := mustWaiver(t, validWaiverSpec())

	if w.Category() != "auth_security" {
		t.Errorf("category = %q, want auth_security", w.Category())
	}
	gotGaps := w.Gaps()
	if len(gotGaps) != 2 || gotGaps[0].AcceptanceID != "AC-02" || gotGaps[1].AcceptanceID != "AC-05" {
		t.Errorf("gaps = %v, want [AC-02 AC-05]", gotGaps)
	}
	if got := w.CompromisedAcceptances(); len(got) != 1 || got[0] != "AC-02" {
		t.Errorf("compromised acceptances = %v, want [AC-02]", got)
	}
	wantBinding := evidence.BindingRef{Project: testProject, Verification: "sha256:contract", BindingID: "EV-AC02-01"}
	if got := w.CompromisedBindings(); len(got) != 1 || got[0] != wantBinding {
		t.Errorf("compromised bindings = %v, want [%v]", got, wantBinding)
	}
	if got := w.ExposureRefs(); len(got) != 1 || got[0] != "exp-1" {
		t.Errorf("exposure refs = %v, want [exp-1]", got)
	}
	if w.ReasonCode() != "holdout_exposure_accepted" {
		t.Errorf("reason code = %q, want holdout_exposure_accepted", w.ReasonCode())
	}
	if w.AcceptedBy() != "owner-7" {
		t.Errorf("accepted by = %q, want owner-7", w.AcceptedBy())
	}
	if !w.AcceptedAt().Equal(testAt) || w.AcceptedAt().Location() != time.UTC {
		t.Errorf("accepted at = %v, want %v (UTC)", w.AcceptedAt(), testAt)
	}
}

func TestNewWaiverNormalizesAcceptedAtToUTC(t *testing.T) {
	spec := validWaiverSpec()
	spec.AcceptedAt = time.Date(2026, 9, 6, 14, 0, 0, 0, testZone) // UTC+2 的 14:00 = UTC 12:00
	w := mustWaiver(t, spec)
	if want := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC); !w.AcceptedAt().Equal(want) || w.AcceptedAt().Location() != time.UTC {
		t.Errorf("accepted at = %v, want %v (UTC)", w.AcceptedAt(), want)
	}
}

func TestNewWaiverRejectsIncompleteProvenance(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*WaiverSpec)
	}{
		{"missing category", func(s *WaiverSpec) { s.Category = "" }},
		{"no gaps", func(s *WaiverSpec) { s.Gaps = nil }},
		{"empty gap acceptance id", func(s *WaiverSpec) { s.Gaps = []CoverageGap{{AcceptanceID: " "}} }},
		{"duplicate gap", func(s *WaiverSpec) { s.Gaps = []CoverageGap{{AcceptanceID: "AC-02"}, {AcceptanceID: "AC-02"}} }},
		{"empty compromised acceptance entry", func(s *WaiverSpec) { s.CompromisedAcceptances = []string{"AC-02", ""} }},
		{"incomplete binding ref", func(s *WaiverSpec) {
			s.CompromisedBindings = []evidence.BindingRef{{Project: testProject, BindingID: "EV-1"}}
		}},
		{"empty exposure ref", func(s *WaiverSpec) { s.ExposureRefs = []evidence.ExposureID{"exp-1", " "} }},
		{"missing reason code", func(s *WaiverSpec) { s.ReasonCode = "  " }},
		{"missing owner acceptance", func(s *WaiverSpec) { s.AcceptedBy = "" }},
		{"missing accepted at", func(s *WaiverSpec) { s.AcceptedAt = time.Time{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := validWaiverSpec()
			tc.mut(&spec)
			if _, err := NewWaiver(spec); !errors.Is(err, ErrInvalidWaiver) {
				t.Fatalf("NewWaiver err = %v, want ErrInvalidWaiver", err)
			}
		})
	}
}

// AC4(第一道防线):planned coverage 缺失在构造豁免时即被拒绝 ——
// design-time(§8.5.1 Freeze 时已可见)缺失不是运行期损失,不可豁免。
func TestNewWaiverRejectsPlannedCoverageGap(t *testing.T) {
	spec := validWaiverSpec()
	spec.Gaps = []CoverageGap{{AcceptanceID: "AC-07", Planned: true}}
	_, err := NewWaiver(spec)
	if !errors.Is(err, ErrPlannedCoverageNotWaivable) {
		t.Fatalf("NewWaiver err = %v, want ErrPlannedCoverageNotWaivable", err)
	}
}

func TestNewWaiverIsIsolatedFromSpecAndAccessors(t *testing.T) {
	spec := validWaiverSpec()
	w := mustWaiver(t, spec)

	// 构造后改动 spec 的切片不影响已构造豁免。
	spec.Gaps[0].AcceptanceID = "AC-XX"
	spec.CompromisedAcceptances[0] = "AC-XX"
	spec.ExposureRefs[0] = "exp-XX"
	if got := w.Gaps()[0].AcceptanceID; got != "AC-02" {
		t.Errorf("gap acceptance = %q, want AC-02(spec 泄漏)", got)
	}
	if got := w.CompromisedAcceptances()[0]; got != "AC-02" {
		t.Errorf("compromised acceptance = %q, want AC-02(spec 泄漏)", got)
	}
	if got := w.ExposureRefs()[0]; got != "exp-1" {
		t.Errorf("exposure ref = %q, want exp-1(spec 泄漏)", got)
	}

	// 访问器返回副本:改动返回值不影响豁免。
	w.Gaps()[0].AcceptanceID = "AC-YY"
	w.CompromisedAcceptances()[0] = "AC-YY"
	w.ExposureRefs()[0] = "exp-YY"
	if got := w.Gaps()[0].AcceptanceID; got != "AC-02" {
		t.Errorf("gap acceptance = %q, want AC-02(访问器未拷贝)", got)
	}
	if got := w.CompromisedAcceptances()[0]; got != "AC-02" {
		t.Errorf("compromised acceptance = %q, want AC-02(访问器未拷贝)", got)
	}
	if got := w.ExposureRefs()[0]; got != "exp-1" {
		t.Errorf("exposure ref = %q, want exp-1(访问器未拷贝)", got)
	}
}

func TestWaiverPolicyAvailabilityIsPolicyInput(t *testing.T) {
	// AC3:零值 policy 禁止一切豁免(保守默认:没有显式声明的放行不存在)。
	var zero WaiverPolicy
	if zero.Allows("auth_security") {
		t.Error("零值 WaiverPolicy 放行了豁免,保守默认必须全禁")
	}
	disabledCategory := WaiverPolicy{Allowed: true, DisabledCategories: []RiskCategory{"schema"}}
	if disabledCategory.Allows("schema") {
		t.Error("policy 按 risk category 禁用后仍放行该类别(§8.10)")
	}
	if !disabledCategory.Allows("auth_security") {
		t.Error("policy 总开关开启且类别未禁用时未放行")
	}
	denied := WaiverPolicy{Allowed: false}
	if denied.Allows("auth_security") {
		t.Error("总开关关闭时仍放行豁免")
	}
}

// runtimeGaps 运行期独立性缺口;plannedGap design-time 缺失。
var (
	runtimeGaps = []CoverageGap{{AcceptanceID: "AC-02"}, {AcceptanceID: "AC-05"}}
	plannedGap  = []CoverageGap{{AcceptanceID: "AC-07", Planned: true}}
	permissive  = WaiverPolicy{Allowed: true}
)

func TestResolveStrictOutcomeSatisfiedWithoutWaiver(t *testing.T) {
	// AC2:strict_satisfied 与 strict_waived 是两个互斥结论值。
	outcome, err := ResolveStrictOutcome(permissive, nil, nil)
	if err != nil {
		t.Fatalf("ResolveStrictOutcome: 意外错误: %v", err)
	}
	if outcome != StrictSatisfied {
		t.Errorf("outcome = %q, want %q", outcome, StrictSatisfied)
	}
	if StrictSatisfied == StrictWaived || StrictSatisfied == "strict_waived" || StrictWaived != "strict_waived" || StrictSatisfied != "strict_satisfied" {
		t.Errorf("strict 结论值不满足 §8.10 令牌:satisfied=%q waived=%q", StrictSatisfied, StrictWaived)
	}
}

func TestResolveStrictOutcomeWaivedWithCoveringWaiver(t *testing.T) {
	// AC2/AC3:policy 允许 + 豁免精确覆盖全部运行期缺口 → strict_waived,
	// 与 strict_satisfied 截然不同;strict_waived 不是「已满足」。
	w := mustWaiver(t, validWaiverSpec())
	outcome, err := ResolveStrictOutcome(permissive, runtimeGaps, w)
	if err != nil {
		t.Fatalf("ResolveStrictOutcome: 意外错误: %v", err)
	}
	if outcome != StrictWaived {
		t.Errorf("outcome = %q, want %q", outcome, StrictWaived)
	}
}

func TestResolveStrictOutcomeRejectsUnmetWithoutOrWithDeniedWaiver(t *testing.T) {
	w := mustWaiver(t, validWaiverSpec())

	// 未满足且无豁免:strict 要求照旧不满足。
	if _, err := ResolveStrictOutcome(permissive, runtimeGaps, nil); !errors.Is(err, ErrWaiverRequired) {
		t.Fatalf("err = %v, want ErrWaiverRequired", err)
	}
	// 零值 policy:豁免不可用(AC3 保守默认)。
	if _, err := ResolveStrictOutcome(WaiverPolicy{}, runtimeGaps, w); !errors.Is(err, ErrWaiverNotPermitted) {
		t.Fatalf("err = %v, want ErrWaiverNotPermitted", err)
	}
	// 总开关开启但该 risk category 被禁用(§8.10)。
	denied := WaiverPolicy{Allowed: true, DisabledCategories: []RiskCategory{"auth_security"}}
	if _, err := ResolveStrictOutcome(denied, runtimeGaps, w); !errors.Is(err, ErrWaiverNotPermitted) {
		t.Fatalf("err = %v, want ErrWaiverNotPermitted", err)
	}
}

func TestResolveStrictOutcomeRequiresExactCoverage(t *testing.T) {
	w := mustWaiver(t, validWaiverSpec()) // 覆盖 AC-02、AC-05

	// 少列:豁免只覆盖部分缺口,放行会静默丢掉其余缺口。
	subset := []CoverageGap{{AcceptanceID: "AC-02"}, {AcceptanceID: "AC-05"}, {AcceptanceID: "AC-09"}}
	if _, err := ResolveStrictOutcome(permissive, subset, w); !errors.Is(err, ErrWaiverIncomplete) {
		t.Fatalf("err = %v, want ErrWaiverIncomplete", err)
	}
	// 多列:豁免声称豁掉并不存在的缺口。
	extra := []CoverageGap{{AcceptanceID: "AC-02"}}
	if _, err := ResolveStrictOutcome(permissive, extra, w); !errors.Is(err, ErrWaiverOverreach) {
		t.Fatalf("err = %v, want ErrWaiverOverreach", err)
	}
}

// AC4(第二道防线):即使 policy 允许、豁免在构造层之外出现,Resolve 也在
// policy/豁免检查之前拒绝 planned coverage 缺失 —— 任何 waiver 都不能修复它。
func TestResolveStrictOutcomeNeverWaivesPlannedCoverage(t *testing.T) {
	w := mustWaiver(t, validWaiverSpec()) // 覆盖 AC-02、AC-05,不含 AC-07

	// 只有 planned 缺失。
	if _, err := ResolveStrictOutcome(permissive, plannedGap, nil); !errors.Is(err, ErrPlannedCoverageNotWaivable) {
		t.Fatalf("err = %v, want ErrPlannedCoverageNotWaivable", err)
	}
	// planned + 运行期混合:仍拒绝,不得部分豁免后假装满足。
	mixed := append(append([]CoverageGap{}, runtimeGaps...), plannedGap...)
	if _, err := ResolveStrictOutcome(permissive, mixed, w); !errors.Is(err, ErrPlannedCoverageNotWaivable) {
		t.Fatalf("err = %v, want ErrPlannedCoverageNotWaivable", err)
	}
}

func TestResolveStrictOutcomeRejectsOverreachAndBadInputs(t *testing.T) {
	w := mustWaiver(t, validWaiverSpec())

	// strict 已满足却带豁免:不得用豁免行冒充/污染 strict_satisfied 统计。
	if _, err := ResolveStrictOutcome(permissive, nil, w); !errors.Is(err, ErrWaiverOverreach) {
		t.Fatalf("err = %v, want ErrWaiverOverreach", err)
	}
	// 调用方错误:unmet 含空 id。
	if _, err := ResolveStrictOutcome(permissive, []CoverageGap{{AcceptanceID: " "}}, nil); err == nil {
		t.Fatal("unmet 含空 acceptance id 却通过了判定")
	}
	// 调用方错误:unmet 重复 id(coverage 推导每条 acceptance 至多一项)。
	dup := []CoverageGap{{AcceptanceID: "AC-02"}, {AcceptanceID: "AC-02"}}
	if _, err := ResolveStrictOutcome(permissive, dup, nil); err == nil {
		t.Fatal("unmet 重复 acceptance id 却通过了判定")
	}
	// 零值豁免(provenance 全空)不可充当有效豁免。
	if _, err := ResolveStrictOutcome(permissive, runtimeGaps, &IndependenceWaiver{}); !errors.Is(err, ErrInvalidWaiver) {
		t.Fatalf("err = %v, want ErrInvalidWaiver", err)
	}
}

// validApprovalSpec 返回一份完整合法的批准决定输入。
func validApprovalSpec() ApprovalSpec {
	return ApprovalSpec{
		Project:   testProject,
		TaskID:    testTask,
		Decision:  DecisionApprove,
		Reason:    "review complete; residual risk accepted",
		DecidedBy: "owner-7",
		DecidedAt: testAt,
	}
}

func TestNewApprovalRecordsDecisionEvidence(t *testing.T) {
	// §18.1:决定只以证据存在,不携带任务状态。
	a, err := NewApproval(validApprovalSpec())
	if err != nil {
		t.Fatalf("NewApproval: 意外错误: %v", err)
	}
	if a.Project() != testProject || a.TaskID() != testTask {
		t.Errorf("project/task = %s/%s, want %s/%s", a.Project(), a.TaskID(), testProject, testTask)
	}
	if cs, ok := a.ChangeSetID(); ok || cs != "" {
		t.Errorf("ChangeSetID = (%q, %v), want empty/unset", cs, ok)
	}
	if a.Decision() != DecisionApprove || a.DecidedBy() != "owner-7" {
		t.Errorf("decision/by = %s/%s, want approve/owner-7", a.Decision(), a.DecidedBy())
	}
	if !a.DecidedAt().Equal(testAt) || a.DecidedAt().Location() != time.UTC {
		t.Errorf("decided at = %v, want %v (UTC)", a.DecidedAt(), testAt)
	}
	if a.WaivedIndependence() || a.Waiver() != nil {
		t.Error("无豁免批准被标记为 waived_independence")
	}
}

func TestNewApprovalApproveWithWaiverCarriesWaivedIndependence(t *testing.T) {
	spec := validApprovalSpec()
	spec.ChangeSetID = testCS
	w := mustWaiver(t, validWaiverSpec())
	spec.Waiver = w

	a, err := NewApproval(spec)
	if err != nil {
		t.Fatalf("NewApproval: 意外错误: %v", err)
	}
	if cs, ok := a.ChangeSetID(); !ok || cs != testCS {
		t.Errorf("ChangeSetID = (%q, %v), want (%q, true)", cs, ok, testCS)
	}
	if !a.WaivedIndependence() {
		t.Error("携带豁免的批准行 waived_independence = false")
	}
	if a.Waiver() != w || a.Waiver().ReasonCode() != "holdout_exposure_accepted" {
		t.Error("Waiver() 未返回所附豁免 provenance")
	}
}

func TestNewApprovalNormalizesDecidedAtToUTC(t *testing.T) {
	spec := validApprovalSpec()
	spec.DecidedAt = time.Date(2026, 9, 6, 14, 0, 0, 0, testZone)
	a, err := NewApproval(spec)
	if err != nil {
		t.Fatalf("NewApproval: 意外错误: %v", err)
	}
	if want := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC); !a.DecidedAt().Equal(want) || a.DecidedAt().Location() != time.UTC {
		t.Errorf("decided at = %v, want %v (UTC)", a.DecidedAt(), want)
	}
}

func TestNewApprovalRejectsInvalidEvidence(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ApprovalSpec)
	}{
		{"missing project", func(s *ApprovalSpec) { s.Project = "" }},
		{"missing task", func(s *ApprovalSpec) { s.TaskID = " " }},
		{"empty decision", func(s *ApprovalSpec) { s.Decision = "" }},
		{"unknown decision", func(s *ApprovalSpec) { s.Decision = Decision("maybe") }},
		{"missing reason", func(s *ApprovalSpec) { s.Reason = "  " }},
		{"missing decided by", func(s *ApprovalSpec) { s.DecidedBy = "" }},
		{"missing decided at", func(s *ApprovalSpec) { s.DecidedAt = time.Time{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := validApprovalSpec()
			tc.mut(&spec)
			if _, err := NewApproval(spec); !errors.Is(err, ErrInvalidApproval) {
				t.Fatalf("NewApproval err = %v, want ErrInvalidApproval", err)
			}
		})
	}
}

func TestNewApprovalRejectIsAllowedWithoutWaiver(t *testing.T) {
	// §19:未批准/否决一律显式记 reject,原因必填;不带豁免。
	spec := validApprovalSpec()
	spec.Decision = DecisionReject
	spec.Reason = "unattributed changes remain"

	a, err := NewApproval(spec)
	if err != nil {
		t.Fatalf("NewApproval: 意外错误: %v", err)
	}
	if a.Decision() != DecisionReject || a.WaivedIndependence() {
		t.Errorf("decision/waived = %s/%v, want reject/false", a.Decision(), a.WaivedIndependence())
	}
}

// reject 与「接受风险放行」矛盾:豁免只随 approve 记录。
func TestNewApprovalRejectRejectsWaiver(t *testing.T) {
	spec := validApprovalSpec()
	spec.Decision = DecisionReject
	spec.Waiver = mustWaiver(t, validWaiverSpec())
	if _, err := NewApproval(spec); !errors.Is(err, ErrInvalidApproval) {
		t.Fatalf("NewApproval err = %v, want ErrInvalidApproval", err)
	}
}

// 零值豁免指针(provenance 全空)不可通过构造防御混入决定证据。
func TestNewApprovalRejectsZeroValueWaiver(t *testing.T) {
	spec := validApprovalSpec()
	spec.Waiver = &IndependenceWaiver{}
	if _, err := NewApproval(spec); !errors.Is(err, ErrInvalidWaiver) {
		t.Fatalf("NewApproval err = %v, want ErrInvalidWaiver", err)
	}
}
