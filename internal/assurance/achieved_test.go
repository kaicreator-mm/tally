package assurance

import (
	"reflect"
	"testing"
)

func TestAchievedFromFacts(t *testing.T) {
	cases := []struct {
		name      string
		facts     ExecutedFacts
		want      Level
		wantFirst Reason
	}{
		{"human approval", ExecutedFacts{HumanApproval: true}, LevelL4, ReasonFactHumanApproval},
		{"held out", ExecutedFacts{HeldOut: true, Deterministic: true}, LevelL3, ReasonFactHeldOut},
		{"judge + deterministic", ExecutedFacts{Judge: true, Deterministic: true}, LevelL2, ReasonFactJudge},
		{"deterministic only", ExecutedFacts{Deterministic: true}, LevelL1, ReasonFactDeterministic},
		{"self check only", ExecutedFacts{SelfCheck: true}, LevelL0, ReasonFactSelfCheck},
		{"no facts", ExecutedFacts{}, LevelL0, ReasonFactNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			level, basis := Achieved(c.facts)
			if level != c.want {
				t.Fatalf("Achieved = %s, want %s", level, c.want)
			}
			if len(basis.Reasons) == 0 || basis.Reasons[0] != c.wantFirst {
				t.Fatalf("Reasons = %v, 首条 want %s", basis.Reasons, c.wantFirst)
			}
			if !reflect.DeepEqual(basis.Facts, c.facts) {
				t.Fatalf("Facts = %+v, want %+v", basis.Facts, c.facts)
			}
		})
	}
}

func TestAchievedNoFactsFlagged(t *testing.T) {
	level, basis := Achieved(ExecutedFacts{})
	if level != LevelL0 {
		t.Fatalf("无事实档位 = %s, want L0", level)
	}
	if !basis.NoVerificationFacts {
		t.Fatal("无验证事实必须显式标注,不冒充 self-check")
	}
}

// TestAchievedJudgeWithoutDeterministicBase 按 PRD §13.2(L2 = judge + 确定性
// 检查),单独的 judge 事实不抬高档位,但必须留痕。
func TestAchievedJudgeWithoutDeterministicBase(t *testing.T) {
	level, basis := Achieved(ExecutedFacts{Judge: true, SelfCheck: true})
	if level != LevelL0 {
		t.Fatalf("judge 无确定性基础: Achieved = %s, want L0", level)
	}
	want := []Reason{ReasonFactSelfCheck, ReasonJudgeNotCounted}
	if !reflect.DeepEqual(basis.Reasons, want) {
		t.Fatalf("Reasons = %v, want %v", basis.Reasons, want)
	}
}

func TestAchievedJudgeWithHeldOutStaysL3(t *testing.T) {
	level, basis := Achieved(ExecutedFacts{Judge: true, Deterministic: true, HeldOut: true})
	if level != LevelL3 {
		t.Fatalf("judge+确定性+held-out: Achieved = %s, want L3", level)
	}
	want := []Reason{ReasonFactHeldOut}
	if !reflect.DeepEqual(basis.Reasons, want) {
		t.Fatalf("Reasons = %v, want %v", basis.Reasons, want)
	}
}
