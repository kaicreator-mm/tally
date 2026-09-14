package engineeringpolicy

import (
	"errors"
	"reflect"
	"testing"
)

func TestEvaluateGateSatisfied(t *testing.T) {
	p := gateFixture() // gate 要求 n_minus_1_upgrade + data_integrity
	verdict, err := p.EvaluateGate([]RequirementFulfillment{
		{Key: "n_minus_1_upgrade", Satisfied: true},
		{Key: "data_integrity", Satisfied: true},
		{Key: "fresh_database", Satisfied: true}, // 在 requires[] 但不在 gate:允许,不影响 verdict
	})
	if err != nil {
		t.Fatalf("求值失败: %v", err)
	}
	if !verdict.Satisfied || len(verdict.UnmetKeys) != 0 {
		t.Fatalf("期望 SATISFIED,得到 %+v", verdict)
	}
}

func TestEvaluateGateUnmet(t *testing.T) {
	p := gateFixture()
	verdict, err := p.EvaluateGate([]RequirementFulfillment{
		{Key: "n_minus_1_upgrade", Satisfied: true},
		{Key: "data_integrity", Satisfied: false},
		{Key: "fresh_database", Satisfied: true},
	})
	if err != nil {
		t.Fatalf("求值失败: %v", err)
	}
	if verdict.Satisfied {
		t.Fatal("data_integrity 未满足,门禁不得通过")
	}
	if !reflect.DeepEqual(verdict.UnmetKeys, []string{"data_integrity"}) {
		t.Fatalf("UnmetKeys = %v", verdict.UnmetKeys)
	}
}

func TestEvaluateGateMissingFulfillmentIsUnmet(t *testing.T) {
	// 缺席的 gate 键按未满足处理(fail-safe),而不是报错也不是默认通过。
	p := gateFixture()
	verdict, err := p.EvaluateGate([]RequirementFulfillment{
		{Key: "fresh_database", Satisfied: true},
		{Key: "data_integrity", Satisfied: true},
	})
	if err != nil {
		t.Fatalf("求值失败: %v", err)
	}
	if verdict.Satisfied || !reflect.DeepEqual(verdict.UnmetKeys, []string{"n_minus_1_upgrade"}) {
		t.Fatalf("期望 n_minus_1_upgrade 未满足,得到 %+v", verdict)
	}
}

func TestEvaluateGateUnmetKeysSorted(t *testing.T) {
	// 输出确定性:UnmetKeys 恒升序,与输入顺序无关。
	p := Policy{
		ID:          "POL-MULTI",
		Enforcement: EnforcementGate,
		Requires:    []string{"c", "a", "b"},
		GateCondition: GateCondition{
			AllOf: []string{"c", "a", "b"},
		},
	}
	verdict, err := p.EvaluateGate(nil)
	if err != nil {
		t.Fatalf("求值失败: %v", err)
	}
	if verdict.Satisfied {
		t.Fatal("空输入不得通过门禁")
	}
	if !reflect.DeepEqual(verdict.UnmetKeys, []string{"a", "b", "c"}) {
		t.Fatalf("UnmetKeys 应升序,得到 %v", verdict.UnmetKeys)
	}
}

func TestEvaluateGateDeterministicSameInputSameVerdict(t *testing.T) {
	p := gateFixture()
	inputs := [][]RequirementFulfillment{
		{
			{Key: "data_integrity", Satisfied: true},
			{Key: "n_minus_1_upgrade", Satisfied: true},
		},
		{
			{Key: "n_minus_1_upgrade", Satisfied: true},
			{Key: "data_integrity", Satisfied: true},
			{Key: "fresh_database", Satisfied: false},
		},
	}
	for i, in := range inputs {
		first, err := p.EvaluateGate(in)
		if err != nil {
			t.Fatalf("输入 %d 求值失败: %v", i, err)
		}
		for j := 0; j < 5; j++ {
			again, err := p.EvaluateGate(in)
			if err != nil {
				t.Fatalf("输入 %d 第 %d 次求值失败: %v", i, j, err)
			}
			if !reflect.DeepEqual(first, again) {
				t.Fatalf("同一输入得到不同 verdict: %+v vs %+v", first, again)
			}
		}
	}
}

func TestEvaluateGateRejectsGuidancePolicy(t *testing.T) {
	// AC2:guidance 请求门禁语义被类型化拒绝 —— 阻断权在结构上不可绕过。
	p := Policy{
		ID:          "POL-STYLE",
		Enforcement: EnforcementGuidance,
		Requires:    []string{"conventional_commits"},
	}
	_, err := p.EvaluateGate([]RequirementFulfillment{{Key: "conventional_commits", Satisfied: true}})
	if err == nil {
		t.Fatal("guidance 策略不得参与门禁求值")
	}
	if !errors.Is(err, ErrNotGate) {
		t.Fatalf("期望 ErrNotGate,得到: %v", err)
	}
}

func TestEvaluateGateRejectsInvalidGateInput(t *testing.T) {
	p := gateFixture()
	cases := []struct {
		name string
		in   []RequirementFulfillment
	}{
		{"空白键", []RequirementFulfillment{{Key: "  ", Satisfied: true}}},
		{"重复键", []RequirementFulfillment{{Key: "data_integrity", Satisfied: true}, {Key: "data_integrity", Satisfied: false}}},
		{"未知键", []RequirementFulfillment{{Key: "load_test", Satisfied: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.EvaluateGate(tc.in)
			if err == nil {
				t.Fatal("期望被拒,实际通过")
			}
			if !errors.Is(err, ErrInvalidGateInput) {
				t.Fatalf("期望 ErrInvalidGateInput,得到: %v", err)
			}
		})
	}
}

func TestEvaluateGateRejectsInvalidPolicy(t *testing.T) {
	// 非法策略(此处:gate_condition 引用未声明键)不得产出 verdict。
	p := Policy{ID: "POL-BAD", Enforcement: EnforcementGate, GateCondition: GateCondition{AllOf: []string{"ghost"}}}
	_, err := p.EvaluateGate(nil)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("期望 ErrInvalidPolicy,得到: %v", err)
	}
}
