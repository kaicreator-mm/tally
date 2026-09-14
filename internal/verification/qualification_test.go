package verification

import (
	"strings"
	"testing"
)

// newTestBinding 构造一条 test 类型 binding:baseline 观察为 fail/assertion,
// qualification 声明与观察一致(strong/compact)。用例按需覆盖字段。
func newTestBinding(id string, mutate func(*Binding)) *Binding {
	b := &Binding{
		ID:               id,
		AuthoringMode:    AuthoringPreCommit,
		VisibilityPolicy: VisibilityVisible,
		Evidence:         EvidenceRef{Type: EvidenceTest, Path: "internal/x_test.go", Selector: "TestX"},
		Baseline: &Baseline{
			Observed: &BaselineObserved{Status: "fail", Class: "assertion"},
		},
		Qualification: &Qualification{
			BaselineSensitivity: "strong",
			ChangeOverlap:       "required",
			ReviewMode:          ReviewModeCompact,
		},
	}
	if mutate != nil {
		mutate(b)
	}
	return b
}

func hasProblem(ev Evaluation, code ProblemCode) bool {
	for _, p := range ev.Problems {
		if p.Code == code {
			return true
		}
	}
	return false
}

// AC1:baseline 观察 → assertion/compile_error/panic/error/pass/n-a 的机械分类
// 与 §8.3.1 表格的证明强度一一对应。
func TestClassifyBaselineClasses(t *testing.T) {
	cases := []struct {
		name       string
		evidence   EvidenceType
		status     string
		class      string
		noBaseline bool // test 证据但完全不记录 baseline
		wantClass  BaselineClass
		wantSens   Sensitivity
		wantErrSub string
	}{
		{name: "assertion failure is strong", evidence: EvidenceTest, status: "fail", class: "assertion", wantClass: ClassAssertion, wantSens: SensitivityStrong},
		{name: "compile error is weak", evidence: EvidenceTest, status: "fail", class: "compile_error", wantClass: ClassCompileError, wantSens: SensitivityWeak},
		{name: "panic is weak", evidence: EvidenceTest, status: "fail", class: "panic", wantClass: ClassPanic, wantSens: SensitivityWeak},
		{name: "error is weak", evidence: EvidenceTest, status: "fail", class: "error", wantClass: ClassError, wantSens: SensitivityWeak},
		{name: "baseline pass has no sensitivity", evidence: EvidenceTest, status: "pass", class: "", wantClass: ClassPass, wantSens: SensitivityNone},
		{name: "non-test evidence is n/a", evidence: EvidenceValidator, status: "", class: "", wantClass: ClassNA, wantSens: SensitivityNA},
		{name: "test without baseline execution", evidence: EvidenceTest, noBaseline: false, status: "", class: "", wantErrSub: "baseline"},
		{name: "test with nil baseline", evidence: EvidenceTest, noBaseline: true, wantErrSub: "baseline"},
		{name: "fail without class cannot be classified", evidence: EvidenceTest, status: "fail", class: "", wantErrSub: "class"},
		{name: "pass with failure class is contradictory", evidence: EvidenceTest, status: "pass", class: "assertion", wantErrSub: "矛盾"},
		{name: "unknown status is rejected", evidence: EvidenceTest, status: "skip", class: "", wantErrSub: "status"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newTestBinding("EV-1", func(b *Binding) {
				if tc.evidence != EvidenceTest {
					b.Evidence = EvidenceRef{Type: tc.evidence, Validator: "v1"}
					b.Baseline = nil
					b.Qualification = nil
					return
				}
				switch {
				case tc.noBaseline:
					b.Baseline = nil
				case tc.status == "" && tc.class == "":
					b.Baseline = &Baseline{Observed: &BaselineObserved{Status: "", Class: ""}}
				default:
					b.Baseline = &Baseline{Observed: &BaselineObserved{Status: tc.status, Class: tc.class}}
				}
			})
			class, sens, err := ClassifyBaseline(b)
			if tc.wantErrSub != "" {
				if err == nil {
					t.Fatalf("ClassifyBaseline: expected error containing %q, got nil", tc.wantErrSub)
				}
				if !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("ClassifyBaseline: error %q does not contain %q", err, tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("ClassifyBaseline: unexpected error: %v", err)
			}
			if class != tc.wantClass || sens != tc.wantSens {
				t.Fatalf("ClassifyBaseline = (%s, %s), want (%s, %s)", class, sens, tc.wantClass, tc.wantSens)
			}
		})
	}
}

// AC1:每个 class 的最低审核要求与 §8.3.1 表格第三列一致。
func TestMinimumReviewByClass(t *testing.T) {
	cases := []struct {
		class             BaselineClass
		wantMode          string
		wantRationale     bool
		wantPolicyDefined bool
	}{
		{class: ClassAssertion, wantMode: ReviewModeCompact},
		{class: ClassCompileError, wantMode: ReviewModeExplicit},
		{class: ClassPanic, wantMode: ReviewModeExplicit},
		{class: ClassError, wantMode: ReviewModeExplicit},
		{class: ClassPass, wantMode: ReviewModeExplicit, wantRationale: true},
		{class: ClassNA, wantPolicyDefined: true},
	}
	for _, tc := range cases {
		got := MinimumReview(tc.class)
		if got.Mode != tc.wantMode || got.RationaleRequired != tc.wantRationale || got.PolicyDefined != tc.wantPolicyDefined {
			t.Fatalf("MinimumReview(%s) = %+v, want mode=%q rationale=%v policyDefined=%v",
				tc.class, got, tc.wantMode, tc.wantRationale, tc.wantPolicyDefined)
		}
	}
}

// AC2:candidate green 需要同时满足 pass + 冻结 protocol + protected hash 一致。
func TestCandidateGreenRequiresProtocolAndHash(t *testing.T) {
	cases := []struct {
		name  string
		cand  CandidateObservation
		green bool
	}{
		{name: "all conditions", cand: CandidateObservation{Status: "pass", ProtocolHeld: true, HashMatch: true}, green: true},
		{name: "failed candidate", cand: CandidateObservation{Status: "fail", ProtocolHeld: true, HashMatch: true}, green: false},
		{name: "protocol violated", cand: CandidateObservation{Status: "pass", ProtocolHeld: false, HashMatch: true}, green: false},
		{name: "hash mismatch", cand: CandidateObservation{Status: "pass", ProtocolHeld: true, HashMatch: false}, green: false},
		{name: "not executed", cand: CandidateObservation{}, green: false},
	}
	for _, tc := range cases {
		if got := tc.cand.Green(); got != tc.green {
			t.Fatalf("%s: Green() = %v, want %v", tc.name, got, tc.green)
		}
	}
}

// AC2:candidate green 与 baseline class 相互独立——green 不能把弱 baseline
// 变强,baseline class 也不因 candidate 结果改变。
func TestCandidateGreenSeparateFromBaselineClass(t *testing.T) {
	t.Run("green candidate does not strengthen pass baseline", func(t *testing.T) {
		b := newTestBinding("EV-PASS", func(b *Binding) {
			b.Baseline.Observed = &BaselineObserved{Status: "pass"}
			b.Qualification = &Qualification{
				BaselineSensitivity: "none",
				ChangeOverlap:       "required",
				ReviewMode:          ReviewModeExplicit,
				Rationale:           "negative invariant; baseline cannot be red",
			}
		})
		ev, err := Evaluate(b, CandidateObservation{Status: "pass", ProtocolHeld: true, HashMatch: true}, ChangeOverlapSignal{Known: true, Present: true})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if !ev.CandidateGreen {
			t.Fatalf("CandidateGreen = false, want true")
		}
		if ev.Class != ClassPass || ev.Sensitivity != SensitivityNone {
			t.Fatalf("class/sensitivity = (%s, %s), want (pass, none): green must not strengthen baseline", ev.Class, ev.Sensitivity)
		}
		if !ev.MinimumReview.RationaleRequired || ev.MinimumReview.Mode != ReviewModeExplicit {
			t.Fatalf("minimum review = %+v, want explicit+rationale", ev.MinimumReview)
		}
		if !ev.Qualified() {
			t.Fatalf("unexpected problems: %+v", ev.Problems)
		}
	})

	t.Run("failed candidate does not change assertion baseline", func(t *testing.T) {
		b := newTestBinding("EV-A", nil)
		ev, err := Evaluate(b, CandidateObservation{Status: "fail", ProtocolHeld: true, HashMatch: true}, ChangeOverlapSignal{Known: true, Present: true})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if ev.Class != ClassAssertion || ev.Sensitivity != SensitivityStrong {
			t.Fatalf("class/sensitivity = (%s, %s), want (assertion, strong)", ev.Class, ev.Sensitivity)
		}
		if ev.CandidateGreen {
			t.Fatalf("CandidateGreen = true, want false")
		}
		if !hasProblem(ev, ProblemCandidateNotGreen) {
			t.Fatalf("expected candidate_not_green problem, got %+v", ev.Problems)
		}
		if ev.Qualified() {
			t.Fatalf("executed non-green candidate must not qualify")
		}
	})

	t.Run("candidate not yet executed is reported but not a violation", func(t *testing.T) {
		b := newTestBinding("EV-A2", nil)
		ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if ev.CandidateGreen {
			t.Fatalf("CandidateGreen = true, want false for zero observation")
		}
		if hasProblem(ev, ProblemCandidateNotGreen) {
			t.Fatalf("not-yet-executed candidate must not be candidate_not_green: %+v", ev.Problems)
		}
		if !ev.Qualified() {
			t.Fatalf("unexpected problems: %+v", ev.Problems)
		}
	})
}

// AC3:change_overlap 是相关性信号:required 的 test 证据默认需要它;
// 已观测且为空才违规;未观测只是数据不足;信号存在也不能替代审核要求。
func TestChangeOverlapIsSignalNotProof(t *testing.T) {
	t.Run("satisfied truth table", func(t *testing.T) {
		cases := []struct {
			sig  ChangeOverlapSignal
			want bool
		}{
			{ChangeOverlapSignal{}, true},                           // 未要求
			{ChangeOverlapSignal{Known: true, Present: true}, true}, // 要求但 Required 未置(由 Evaluate 强制)
			{ChangeOverlapSignal{Required: true, Known: true, Present: true}, true},
			{ChangeOverlapSignal{Required: true, Known: true, Present: false}, false},
			{ChangeOverlapSignal{Required: true, Known: false}, false}, // 未观测 ≠ 满足
		}
		for i, tc := range cases {
			if got := tc.sig.Satisfied(); got != tc.want {
				t.Fatalf("case %d: Satisfied() = %v, want %v (sig=%+v)", i, got, tc.want, tc.sig)
			}
		}
	})

	t.Run("test evidence defaults to required even when undeclared", func(t *testing.T) {
		b := newTestBinding("EV-D", func(b *Binding) {
			b.Qualification = &Qualification{BaselineSensitivity: "strong", ReviewMode: ReviewModeCompact}
		})
		ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: false})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if !ev.Overlap.Required {
			t.Fatalf("test evidence must default change_overlap=required (§8.3.3)")
		}
		if !hasProblem(ev, ProblemOverlapViolated) {
			t.Fatalf("observed empty overlap must be overlap_violated: %+v", ev.Problems)
		}
	})

	t.Run("unknown overlap is data, not violation", func(t *testing.T) {
		b := newTestBinding("EV-U", nil)
		ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if ev.OverlapSatisfied {
			t.Fatalf("unknown overlap must not count as satisfied")
		}
		if hasProblem(ev, ProblemOverlapViolated) {
			t.Fatalf("unobserved overlap must not be a violation: %+v", ev.Problems)
		}
		if !ev.Qualified() {
			t.Fatalf("unexpected problems: %+v", ev.Problems)
		}
	})

	t.Run("present overlap cannot replace review requirements", func(t *testing.T) {
		b := newTestBinding("EV-R", func(b *Binding) {
			b.Baseline.Observed = &BaselineObserved{Status: "pass"}
			b.Qualification = &Qualification{BaselineSensitivity: "none", ChangeOverlap: "required", ReviewMode: ReviewModeExplicit}
		})
		ev, err := Evaluate(b, CandidateObservation{Status: "pass", ProtocolHeld: true, HashMatch: true}, ChangeOverlapSignal{Known: true, Present: true})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if !ev.OverlapSatisfied {
			t.Fatalf("overlap should be satisfied in this scenario")
		}
		if ev.Qualified() {
			t.Fatalf("overlap signal must not prove semantics: missing rationale must still fail")
		}
		if !hasProblem(ev, ProblemReviewRationaleMissing) {
			t.Fatalf("expected review_rationale_missing, got %+v", ev.Problems)
		}
	})
}

// AC4:弱(compile_error/panic/error)与 pass 路径按契约要求 explicit review
// (pass 还须附 rationale);assertion 允许 compact;声明缺失或低于最低要求即违规。
func TestWeakAndPassRequireExplicitReviewPerContract(t *testing.T) {
	weakClasses := []BaselineObserved{{Status: "fail", Class: "compile_error"}, {Status: "fail", Class: "panic"}, {Status: "fail", Class: "error"}}
	for _, obs := range weakClasses {
		t.Run("weak "+obs.Class, func(t *testing.T) {
			mk := func(review string) *Binding {
				return newTestBinding("EV-W", func(b *Binding) {
					b.Baseline.Observed = &BaselineObserved{Status: obs.Status, Class: obs.Class}
					b.Qualification = &Qualification{BaselineSensitivity: "weak", ChangeOverlap: "required", ReviewMode: review}
				})
			}
			if ev, err := Evaluate(mk(ReviewModeCompact), CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true}); err != nil {
				t.Fatalf("Evaluate: %v", err)
			} else if !hasProblem(ev, ProblemReviewBelowMinimum) || ev.ReviewSatisfied {
				t.Fatalf("compact below explicit minimum: %+v", ev.Problems)
			}
			if ev, err := Evaluate(mk(""), CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true}); err != nil {
				t.Fatalf("Evaluate: %v", err)
			} else if !hasProblem(ev, ProblemReviewModeMissing) {
				t.Fatalf("missing review_mode must be flagged: %+v", ev.Problems)
			}
			ev, err := Evaluate(mk(ReviewModeExplicit), CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true})
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if !ev.ReviewSatisfied || !ev.Qualified() {
				t.Fatalf("explicit review should satisfy weak class: %+v", ev.Problems)
			}
		})
	}

	t.Run("baseline pass", func(t *testing.T) {
		mk := func(review, rationale string) *Binding {
			return newTestBinding("EV-P", func(b *Binding) {
				b.Baseline.Observed = &BaselineObserved{Status: "pass"}
				b.Qualification = &Qualification{BaselineSensitivity: "none", ChangeOverlap: "required", ReviewMode: review, Rationale: rationale}
			})
		}
		if ev, err := Evaluate(mk(ReviewModeExplicit, ""), CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true}); err != nil {
			t.Fatalf("Evaluate: %v", err)
		} else if !hasProblem(ev, ProblemReviewRationaleMissing) || ev.Qualified() {
			t.Fatalf("pass without rationale must fail: %+v", ev.Problems)
		}
		if ev, err := Evaluate(mk(ReviewModeCompact, "why"), CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true}); err != nil {
			t.Fatalf("Evaluate: %v", err)
		} else if !hasProblem(ev, ProblemReviewBelowMinimum) || ev.Qualified() {
			t.Fatalf("pass with compact review must fail: %+v", ev.Problems)
		}
		ev, err := Evaluate(mk(ReviewModeExplicit, "why"), CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if !ev.Qualified() {
			t.Fatalf("explicit + rationale should qualify pass baseline: %+v", ev.Problems)
		}
	})

	t.Run("assertion allows compact and explicit", func(t *testing.T) {
		for _, review := range []string{ReviewModeCompact, ReviewModeExplicit} {
			b := newTestBinding("EV-A", func(b *Binding) {
				b.Qualification.ReviewMode = review
			})
			ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true})
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if !ev.Qualified() {
				t.Fatalf("assertion with review_mode=%q should qualify: %+v", review, ev.Problems)
			}
		}
	})
}

// 声明的强度必须与机械推导一致:不允许用声明把弱观察包装成强;
// 非 test 证据声明非 n/a 强度同样视为包装。
func TestDeclaredSensitivityMustMatchMechanicalDerivation(t *testing.T) {
	t.Run("declared strong over pass baseline is a mismatch", func(t *testing.T) {
		b := newTestBinding("EV-M", func(b *Binding) {
			b.Baseline.Observed = &BaselineObserved{Status: "pass"}
			b.Qualification = &Qualification{BaselineSensitivity: "strong", ReviewMode: ReviewModeExplicit, Rationale: "why"}
		})
		ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if !hasProblem(ev, ProblemSensitivityMismatch) || ev.Qualified() {
			t.Fatalf("declared strong over pass must be flagged: %+v", ev.Problems)
		}
		if ev.Sensitivity != SensitivityNone {
			t.Fatalf("mechanical sensitivity must stay none, got %s", ev.Sensitivity)
		}
	})

	t.Run("declared none over pass baseline matches", func(t *testing.T) {
		b := newTestBinding("EV-M2", func(b *Binding) {
			b.Baseline.Observed = &BaselineObserved{Status: "pass"}
			b.Qualification = &Qualification{BaselineSensitivity: "none", ReviewMode: ReviewModeExplicit, Rationale: "why"}
		})
		ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{Known: true, Present: true})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if hasProblem(ev, ProblemSensitivityMismatch) || !ev.Qualified() {
			t.Fatalf("unexpected problems: %+v", ev.Problems)
		}
	})

	t.Run("non-test evidence with non-n/a sensitivity is a mismatch", func(t *testing.T) {
		b := &Binding{
			ID:               "EV-V",
			AuthoringMode:    AuthoringPreCommit,
			VisibilityPolicy: VisibilityVisible,
			Evidence:         EvidenceRef{Type: EvidenceValidator, Validator: "v1"},
			Qualification:    &Qualification{BaselineSensitivity: "strong"},
		}
		ev, err := Evaluate(b, CandidateObservation{}, ChangeOverlapSignal{})
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if ev.Class != ClassNA || ev.Sensitivity != SensitivityNA {
			t.Fatalf("non-test evidence must classify n/a, got (%s, %s)", ev.Class, ev.Sensitivity)
		}
		if !hasProblem(ev, ProblemSensitivityMismatch) || ev.Qualified() {
			t.Fatalf("validator declared strong must be flagged: %+v", ev.Problems)
		}
	})
}

// 端到端:用 PRD §8.11 示例契约(与 types_test.go 同一 fixture)验证
// assertion/strong/compact 与 pass/none/explicit+rationale 两条路径都可判定。
func TestEvaluateContractBindings(t *testing.T) {
	c := mustParse(t, fullContractYAML)
	frozen, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	fc := frozen.Contract()

	b1 := fc.Acceptances[0].EvidenceBindings[0] // EV-AC01-01: fail/assertion, strong/compact/required
	ev, err := Evaluate(&b1, CandidateObservation{Status: "pass", ProtocolHeld: true, HashMatch: true}, ChangeOverlapSignal{Known: true, Present: true})
	if err != nil {
		t.Fatalf("Evaluate AC-01 binding: %v", err)
	}
	if ev.BindingID != "EV-AC01-01" || ev.EvidenceType != EvidenceTest {
		t.Fatalf("evaluation header = %s/%s", ev.BindingID, ev.EvidenceType)
	}
	if ev.Class != ClassAssertion || ev.Sensitivity != SensitivityStrong || !ev.Qualified() {
		t.Fatalf("AC-01 evaluation = %+v", ev)
	}

	b2 := fc.Acceptances[1].EvidenceBindings[0] // EV-AC02-H1: pass baseline, none/explicit+rationale
	ev2, err := Evaluate(&b2, CandidateObservation{Status: "pass", ProtocolHeld: true, HashMatch: true}, ChangeOverlapSignal{Known: true, Present: true})
	if err != nil {
		t.Fatalf("Evaluate AC-02 binding: %v", err)
	}
	if ev2.Class != ClassPass || ev2.Sensitivity != SensitivityNone || ev2.MinimumReview.Mode != ReviewModeExplicit || !ev2.MinimumReview.RationaleRequired {
		t.Fatalf("AC-02 evaluation header = %+v", ev2)
	}
	if !ev2.Qualified() {
		t.Fatalf("AC-02 unexpected problems: %+v", ev2.Problems)
	}

	b3 := fc.Acceptances[2].EvidenceBindings[0] // EV-AC03-01: validator → n/a
	ev3, err := Evaluate(&b3, CandidateObservation{}, ChangeOverlapSignal{})
	if err != nil {
		t.Fatalf("Evaluate AC-03 binding: %v", err)
	}
	if ev3.Class != ClassNA || !ev3.Qualified() {
		t.Fatalf("AC-03 evaluation = %+v", ev3)
	}
}

func TestEvaluateNilBinding(t *testing.T) {
	if _, err := Evaluate(nil, CandidateObservation{}, ChangeOverlapSignal{}); err == nil {
		t.Fatalf("Evaluate(nil): expected error, got nil")
	}
}
