package assurance

import (
	"errors"
	"testing"
)

func mustSelect(t *testing.T, in ClauseInput) Selection {
	t.Helper()
	sel, err := Select(in)
	if err != nil {
		t.Fatalf("Select(%+v): %v", in, err)
	}
	return sel
}

// TestRequiredAndAchievedAreDistinctTypes 是 AC1 的类型层证据:required 与
// achieved 是两个独立字段/独立推导路径的值,互不改写。
func TestRequiredAndAchievedAreDistinctTypes(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-01", Risk: RiskElevated})
	level, basis := Achieved(ExecutedFacts{Deterministic: true})
	req, err := sel.Evaluate(level, basis, nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if req.Selection.Required != LevelL3 {
		t.Fatalf("Required = %s, want L3", req.Selection.Required)
	}
	if req.AchievedLevel != LevelL1 {
		t.Fatalf("AchievedLevel = %s, want L1", req.AchievedLevel)
	}
	if req.Selection.Required == req.AchievedLevel {
		t.Fatal("required 与 achieved 不应相同:二者是不同的值")
	}
}

func TestEvaluateSatisfied(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-02", Risk: RiskMinimal})
	level, basis := Achieved(ExecutedFacts{Deterministic: true})
	req, err := sel.Evaluate(level, basis, nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if req.Status != StatusSatisfied || !req.Satisfied() || req.Waived() {
		t.Fatalf("Status = %s, Satisfied = %v, Waived = %v", req.Status, req.Satisfied(), req.Waived())
	}
	if req.Waiver != nil {
		t.Fatal("satisfied 条款的 Waiver 必须为 nil")
	}
}

func TestEvaluateUnmet(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-03", Risk: RiskHigh})
	level, basis := Achieved(ExecutedFacts{Deterministic: true})
	req, err := sel.Evaluate(level, basis, nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if req.Status != StatusUnmet || req.Satisfied() || req.Waived() {
		t.Fatalf("Status = %s, want unmet", req.Status)
	}
}

// TestEvaluateWaivedDistinctFromSatisfied 是 AC2 的核心:waived 与 satisfied
// 是两个不同状态;waiver 不改写 required/achieved 档位事实(PRD §13.5)。
func TestEvaluateWaivedDistinctFromSatisfied(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-04", Risk: RiskHigh})
	level, basis := Achieved(ExecutedFacts{Deterministic: true})
	req, err := sel.Evaluate(level, basis, &Waiver{ID: "WAIVER-9", Reason: "policy exception, owner approved"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if req.Status != StatusWaived {
		t.Fatalf("Status = %s, want waived", req.Status)
	}
	if !req.Waived() || req.Satisfied() {
		t.Fatalf("waived 与 satisfied 必须互斥: Waived=%v Satisfied=%v", req.Waived(), req.Satisfied())
	}
	if req.Status == StatusSatisfied {
		t.Fatal("waived 不得合并进 satisfied(PRD §13.5)")
	}
	if req.Waiver == nil || req.Waiver.ID != "WAIVER-9" || req.Waiver.Reason == "" {
		t.Fatalf("Waiver = %+v", req.Waiver)
	}
	if req.Selection.Required != LevelL3 || req.AchievedLevel != LevelL1 {
		t.Fatalf("waiver 不得改写档位事实: required=%s achieved=%s", req.Selection.Required, req.AchievedLevel)
	}
}

func TestEvaluateWaiverIdentityRequired(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-05", Risk: RiskHigh})
	level, basis := Achieved(ExecutedFacts{Deterministic: true})
	for _, w := range []Waiver{{}, {ID: "", Reason: "r"}, {ID: "W", Reason: ""}} {
		waiver := w
		if _, err := sel.Evaluate(level, basis, &waiver); !errors.Is(err, ErrWaiverIdentity) {
			t.Fatalf("waiver %+v 应返回 ErrWaiverIdentity,得到 %v", waiver, err)
		}
	}
}

func TestEvaluateWaiverOnSatisfiedRejected(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-06", Risk: RiskMinimal})
	level, basis := Achieved(ExecutedFacts{Deterministic: true})
	_, err := sel.Evaluate(level, basis, &Waiver{ID: "W-1", Reason: "redundant"})
	if !errors.Is(err, ErrWaiverNotNeeded) {
		t.Fatalf("satisfied 条款上的 waiver 应返回 ErrWaiverNotNeeded,得到 %v", err)
	}
}

func TestEvaluateInvalidAchievedLevel(t *testing.T) {
	sel := mustSelect(t, ClauseInput{ClauseID: "AC-07", Risk: RiskMinimal})
	_, err := sel.Evaluate(Level(7), AchievedBasis{}, nil)
	if !errors.Is(err, ErrInvalidLevel) {
		t.Fatalf("非法 achieved 档位应返回 ErrInvalidLevel,得到 %v", err)
	}
}

// TestEndToEndBakeoffDecisionMapping 用 A0-03/Lock #21 的语义做一次端到端
// 演练:最小风险 + L1 达成 → satisfied;高于最小风险 + 仅 L1 达成 → unmet;
// 显式 waiver → waived(档位事实保留)。
func TestEndToEndBakeoffDecisionMapping(t *testing.T) {
	detLevel, detBasis := Achieved(ExecutedFacts{Deterministic: true})

	minimal := mustSelect(t, ClauseInput{ClauseID: "AC-min", Risk: RiskMinimal})
	req, err := minimal.Evaluate(detLevel, detBasis, nil)
	if err != nil || req.Status != StatusSatisfied {
		t.Fatalf("最小风险条款: status=%v err=%v, want satisfied", req.Status, err)
	}

	risky := mustSelect(t, ClauseInput{ClauseID: "AC-risky", Risk: RiskElevated})
	req, err = risky.Evaluate(detLevel, detBasis, nil)
	if err != nil || req.Status != StatusUnmet {
		t.Fatalf("高于最小风险且仅 L1 达成: status=%v err=%v, want unmet", req.Status, err)
	}

	req, err = risky.Evaluate(detLevel, detBasis, &Waiver{ID: "W-2", Reason: "documented exception"})
	if err != nil || req.Status != StatusWaived || !req.Waived() {
		t.Fatalf("显式 waiver: status=%v err=%v, want waived", req.Status, err)
	}

	// held-out 执行后同一风险条款严格满足(上调档被真实达成,而非豁免)。
	hoLevel, hoBasis := Achieved(ExecutedFacts{Deterministic: true, HeldOut: true})
	req, err = risky.Evaluate(hoLevel, hoBasis, nil)
	if err != nil || req.Status != StatusSatisfied {
		t.Fatalf("L3 达成后: status=%v err=%v, want satisfied", req.Status, err)
	}
}
