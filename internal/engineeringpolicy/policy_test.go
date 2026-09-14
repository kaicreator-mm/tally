package engineeringpolicy

import (
	"errors"
	"testing"
)

// gateFixture 返回一份最小的合法 gate 策略,测试在此基础上覆盖字段。
func gateFixture() Policy {
	return Policy{
		ID:          "POL-MIGRATION",
		Enforcement: EnforcementGate,
		Rationale:   "migration 类变更必须验证数据完整性",
		Trigger:     Trigger{RiskSignals: []string{"schema_migration"}},
		Requires:    []string{"fresh_database", "n_minus_1_upgrade", "data_integrity"},
		GateCondition: GateCondition{
			AllOf: []string{"n_minus_1_upgrade", "data_integrity"},
		},
		BundleTags: []string{"migration"},
	}
}

func TestValidateAcceptsMinimalGatePolicy(t *testing.T) {
	p := Policy{ID: "POL-1", Enforcement: EnforcementGate, GateCondition: GateCondition{AllOf: []string{"k"}}, Requires: []string{"k"}}
	if err := p.Validate(); err != nil {
		t.Fatalf("最小合法 gate 策略被拒: %v", err)
	}
}

func TestValidateRejectsInvalidPolicy(t *testing.T) {
	cases := []struct {
		name string
		mut  func(p Policy) Policy
	}{
		{"缺少 id", func(p Policy) Policy { p.ID = ""; return p }},
		{"id 全空白", func(p Policy) Policy { p.ID = "   "; return p }},
		{"id 含空白", func(p Policy) Policy { p.ID = "POL 1"; return p }},
		{"enforcement 缺席", func(p Policy) Policy { p.Enforcement = ""; return p }},
		{"enforcement 未知", func(p Policy) Policy { p.Enforcement = "block"; return p }},
		{"guidance 携带 gate_condition", func(p Policy) Policy {
			p.Enforcement = EnforcementGuidance
			return p // fixture 自带 gate_condition
		}},
		{"gate 缺 gate_condition", func(p Policy) Policy { p.GateCondition = GateCondition{}; return p }},
		{"gate_condition 引用未声明键", func(p Policy) Policy {
			p.GateCondition.AllOf = []string{"data_integrity", "load_test"}
			return p
		}},
		{"requires 空白条目", func(p Policy) Policy { p.Requires = []string{"fresh_database", "  "}; return p }},
		{"requires 重复条目", func(p Policy) Policy {
			p.Requires = []string{"fresh_database", "fresh_database", "data_integrity"}
			return p
		}},
		{"gate_condition 重复条目", func(p Policy) Policy { p.GateCondition.AllOf = []string{"data_integrity", "data_integrity"}; return p }},
		{"bundle_tags 空白条目", func(p Policy) Policy { p.BundleTags = []string{""}; return p }},
		{"bundle_tags 重复", func(p Policy) Policy { p.BundleTags = []string{"migration", "migration"}; return p }},
		{"trigger.risk_signals 空白条目", func(p Policy) Policy { p.Trigger.RiskSignals = []string{" "}; return p }},
		{"trigger.paths 重复", func(p Policy) Policy { p.Trigger.Paths = []string{"migrations/", "migrations/"}; return p }},
		{"trigger.requirement_refs 空白条目", func(p Policy) Policy { p.Trigger.RequirementRefs = []string{""}; return p }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := gateFixtureMut(tc.mut).Validate()
			if err == nil {
				t.Fatal("期望被拒,实际通过")
			}
			if !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("期望 ErrInvalidPolicy,得到: %v", err)
			}
		})
	}
}

func gateFixtureMut(mut func(Policy) Policy) Policy {
	return mut(gateFixture())
}

func TestValidateAcceptsGuidanceWithoutGateCondition(t *testing.T) {
	// AC2:guidance 可表示 —— 合法的 guidance 策略(无 gate_condition)必须通过。
	p := Policy{
		ID:          "POL-STYLE",
		Enforcement: EnforcementGuidance,
		Rationale:   "提示 agent 遵循仓库提交规范",
		Requires:    []string{"conventional_commits"},
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("合法 guidance 策略被拒: %v", err)
	}
}

func TestEnforcementIsValid(t *testing.T) {
	for _, e := range []Enforcement{EnforcementGate, EnforcementGuidance} {
		if !e.IsValid() {
			t.Fatalf("%q 应为合法 enforcement", string(e))
		}
	}
	for _, e := range []Enforcement{"", "blocking", "GATE"} {
		if e.IsValid() {
			t.Fatalf("%q 不应为合法 enforcement", string(e))
		}
	}
}

func TestTriggerIsZero(t *testing.T) {
	if !(Trigger{}).IsZero() {
		t.Fatal("空 Trigger 应为 zero")
	}
	if (Trigger{RiskSignals: []string{"schema_migration"}}).IsZero() {
		t.Fatal("带 risk_signals 的 Trigger 不应为 zero")
	}
}

func TestGateKeysSortedCopy(t *testing.T) {
	p := Policy{GateCondition: GateCondition{AllOf: []string{"b", "a"}}}
	got := p.GateKeys()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("GateKeys 应升序返回,得到 %v", got)
	}
	if p.GateCondition.AllOf[0] != "b" {
		t.Fatal("GateKeys 不得改原切片")
	}
}
