package assurance

import (
	"errors"
	"reflect"
	"testing"
)

// TestSelectDefaultFloorIsL1ForMinimalRisk 锁定 Decision Lock #21 的另一面:
// held-out 不是 universal 默认 —— 最小风险条款必需档就是 L1,不上调。
func TestSelectDefaultFloorIsL1ForMinimalRisk(t *testing.T) {
	sel, err := Select(ClauseInput{ClauseID: "AC-01", Risk: RiskMinimal})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.Required != LevelL1 {
		t.Fatalf("最小风险条款 Required = %s, want L1", sel.Required)
	}
	wantReasons := []Reason{ReasonDefaultFloor}
	if !reflect.DeepEqual(sel.Basis.Reasons, wantReasons) {
		t.Fatalf("Reasons = %v, want %v", sel.Basis.Reasons, wantReasons)
	}
	if sel.Basis.RiskUnknown {
		t.Fatal("minimal 不应标记 unknown")
	}
}

func TestSelectRiskUpliftAboveMinimal(t *testing.T) {
	for _, risk := range []RiskBand{RiskElevated, RiskHigh} {
		sel, err := Select(ClauseInput{ClauseID: "AC-02", Risk: risk})
		if err != nil {
			t.Fatalf("Select(%s): %v", risk, err)
		}
		if sel.Required != LevelL3 {
			t.Fatalf("风险 %s: Required = %s, want L3(Lock #21 上调)", risk, sel.Required)
		}
		if len(sel.Basis.Reasons) != 2 || sel.Basis.Reasons[1] != ReasonRiskUplift {
			t.Fatalf("风险 %s: Reasons = %v, 缺少 %s", risk, sel.Basis.Reasons, ReasonRiskUplift)
		}
		if sel.Basis.RiskUnknown {
			t.Fatalf("风险 %s 不应标记 unknown", risk)
		}
	}
}

// TestSelectUnmatchedRiskFailsSafeToL3 是 AC3 的直接落点:未知/无法匹配的
// 风险输入(含空零值)fail-safe 到更强一侧 L3,绝不静默选择 L0/L1。
func TestSelectUnmatchedRiskFailsSafeToL3(t *testing.T) {
	for _, risk := range []RiskBand{"", RiskBand("critical"), RiskBand("unknown"), RiskBand("Minimal")} {
		sel, err := Select(ClauseInput{ClauseID: "AC-03", Risk: risk})
		if err != nil {
			t.Fatalf("Select(%q): %v", risk, err)
		}
		if sel.Required != LevelL3 {
			t.Fatalf("未匹配风险 %q: Required = %s, want L3(fail-safe)", risk, sel.Required)
		}
		if !sel.Basis.RiskUnknown || sel.Basis.Risk != "unknown" {
			t.Fatalf("未匹配风险 %q: Basis.Risk = %q, RiskUnknown = %v", risk, sel.Basis.Risk, sel.Basis.RiskUnknown)
		}
		if len(sel.Basis.Reasons) != 2 || sel.Basis.Reasons[1] != ReasonUnmatchedRiskFailsafe {
			t.Fatalf("未匹配风险 %q: Reasons = %v", risk, sel.Basis.Reasons)
		}
	}
}

func TestSelectPolicyFloorRaises(t *testing.T) {
	// policy L3 + minimal risk → L3(policy 理由出现)。
	sel, err := Select(ClauseInput{ClauseID: "AC-04", Risk: RiskMinimal, PolicyFloor: LevelL3})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.Required != LevelL3 {
		t.Fatalf("policy floor L3: Required = %s, want L3", sel.Required)
	}
	want := []Reason{ReasonDefaultFloor, ReasonPolicyFloor}
	if !reflect.DeepEqual(sel.Basis.Reasons, want) {
		t.Fatalf("Reasons = %v, want %v", sel.Basis.Reasons, want)
	}

	// policy L4 + minimal risk → L4。
	sel, err = Select(ClauseInput{ClauseID: "AC-04", Risk: RiskMinimal, PolicyFloor: LevelL4})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.Required != LevelL4 {
		t.Fatalf("policy floor L4: Required = %s, want L4", sel.Required)
	}

	// policy L1 == 默认地板:不追加多余理由。
	sel, err = Select(ClauseInput{ClauseID: "AC-04", Risk: RiskMinimal, PolicyFloor: LevelL1})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.Required != LevelL1 || len(sel.Basis.Reasons) != 1 {
		t.Fatalf("policy floor L1: Required = %s, Reasons = %v", sel.Required, sel.Basis.Reasons)
	}
}

// TestSelectPolicyFloorL2Rejected 锁定 Lock #21:L2 实验档不能被选择为
// 必需档(默认选择器永不输出 L2)。
func TestSelectPolicyFloorL2Rejected(t *testing.T) {
	_, err := Select(ClauseInput{ClauseID: "AC-05", Risk: RiskMinimal, PolicyFloor: LevelL2})
	if !errors.Is(err, ErrL2NotSelectable) {
		t.Fatalf("policy floor L2 应返回 ErrL2NotSelectable,得到 %v", err)
	}
}

func TestSelectPolicyFloorInvalidLevel(t *testing.T) {
	_, err := Select(ClauseInput{ClauseID: "AC-05", Risk: RiskMinimal, PolicyFloor: Level(9)})
	if !errors.Is(err, ErrInvalidLevel) {
		t.Fatalf("policy floor 9 应返回 ErrInvalidLevel,得到 %v", err)
	}
}

func TestSelectHumanAuthority(t *testing.T) {
	sel, err := Select(ClauseInput{ClauseID: "AC-06", Risk: RiskMinimal, HumanAuthority: true})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.Required != LevelL4 {
		t.Fatalf("human authority: Required = %s, want L4", sel.Required)
	}
	want := []Reason{ReasonDefaultFloor, ReasonHumanAuthority}
	if !reflect.DeepEqual(sel.Basis.Reasons, want) {
		t.Fatalf("Reasons = %v, want %v", sel.Basis.Reasons, want)
	}
}

func TestSelectCombinedDecisionChain(t *testing.T) {
	sel, err := Select(ClauseInput{ClauseID: "AC-07", Risk: RiskHigh, PolicyFloor: LevelL3, HumanAuthority: true})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if sel.Required != LevelL4 {
		t.Fatalf("组合输入: Required = %s, want L4", sel.Required)
	}
	// 决策链按求值顺序记录所有已触发规则;policy L3 未抬高(已被风险上调覆盖)故不出现。
	want := []Reason{ReasonDefaultFloor, ReasonRiskUplift, ReasonHumanAuthority}
	if !reflect.DeepEqual(sel.Basis.Reasons, want) {
		t.Fatalf("Reasons = %v, want %v", sel.Basis.Reasons, want)
	}
	if sel.Basis.DefaultFloor != LevelL1 || sel.Basis.Risk != RiskHigh || !sel.Basis.HumanAuthority {
		t.Fatalf("Basis = %+v", sel.Basis)
	}
}

func TestSelectEmptyClauseID(t *testing.T) {
	_, err := Select(ClauseInput{Risk: RiskMinimal})
	if !errors.Is(err, ErrEmptyClause) {
		t.Fatalf("空条款 id 应返回 ErrEmptyClause,得到 %v", err)
	}
}

// TestSelectDeterministic 同输入两次求值结果完全一致(AC:选择必须确定性)。
func TestSelectDeterministic(t *testing.T) {
	in := ClauseInput{ClauseID: "AC-08", Risk: RiskElevated, PolicyFloor: LevelL3}
	a, err := Select(in)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	b, err := Select(in)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("选择不确定: %+v vs %+v", a, b)
	}
}

// TestSelectNeverBelowDefaultFloorAndNeverL2 穷举输入网格,锁定两条不变量:
//   - required 永不低于默认地板 L1(Lock #21:所有条款);
//   - required 永不等于 L2(Lock #21:L2 实验档,不是默认可选档)。
func TestSelectNeverBelowDefaultFloorAndNeverL2(t *testing.T) {
	risks := []RiskBand{"", RiskMinimal, RiskElevated, RiskHigh, RiskBand("bogus")}
	policies := []Level{LevelL0, LevelL1, LevelL2, LevelL3, LevelL4}
	humans := []bool{false, true}

	cases := 0
	for _, r := range risks {
		for _, p := range policies {
			for _, h := range humans {
				if p == LevelL2 {
					if _, err := Select(ClauseInput{ClauseID: "AC-X", Risk: r, PolicyFloor: p, HumanAuthority: h}); !errors.Is(err, ErrL2NotSelectable) {
						t.Fatalf("policy L2 (risk=%s human=%v): %v, want ErrL2NotSelectable", r, h, err)
					}
					continue
				}
				sel, err := Select(ClauseInput{ClauseID: "AC-X", Risk: r, PolicyFloor: p, HumanAuthority: h})
				if err != nil {
					t.Fatalf("Select(risk=%s policy=%s human=%v): %v", r, p, h, err)
				}
				if sel.Required == LevelL0 || sel.Required == LevelL2 {
					t.Fatalf("不变量破坏: risk=%s policy=%s human=%v → Required=%s", r, p, h, sel.Required)
				}
				cases++
			}
		}
	}
	if cases == 0 {
		t.Fatal("穷举网格为空")
	}
}

// TestSelectBasisEmittedForEachDecision 是 AC4 的抽查:每个决策都有依据,
// 且首条理由恒为默认地板。
func TestSelectBasisEmittedForEachDecision(t *testing.T) {
	inputs := []ClauseInput{
		{ClauseID: "AC-A", Risk: RiskMinimal},
		{ClauseID: "AC-B", Risk: RiskElevated},
		{ClauseID: "AC-C", Risk: "???"},
		{ClauseID: "AC-D", Risk: RiskMinimal, HumanAuthority: true},
		{ClauseID: "AC-E", Risk: RiskMinimal, PolicyFloor: LevelL4},
	}
	for _, in := range inputs {
		sel, err := Select(in)
		if err != nil {
			t.Fatalf("Select(%+v): %v", in, err)
		}
		if len(sel.Basis.Reasons) == 0 || sel.Basis.Reasons[0] != ReasonDefaultFloor {
			t.Fatalf("%s: 决策链缺少默认地板依据: %v", in.ClauseID, sel.Basis.Reasons)
		}
		if sel.Basis.DefaultFloor != LevelL1 {
			t.Fatalf("%s: DefaultFloor = %s, want L1", in.ClauseID, sel.Basis.DefaultFloor)
		}
	}
}
