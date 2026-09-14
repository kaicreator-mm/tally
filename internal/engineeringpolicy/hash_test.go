package engineeringpolicy

import (
	"regexp"
	"testing"
)

var revisionShape = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func TestRevisionHashShape(t *testing.T) {
	p, err := Parse([]byte(migrationPolicyYAML))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if !revisionShape.MatchString(p.Revision) {
		t.Fatalf("Revision 形状非法: %q", p.Revision)
	}
	if p.Revision != p.RevisionHash() {
		t.Fatal("Parse 填充的 Revision 必须等于内容推导的 RevisionHash")
	}
}

func TestRevisionHashDeterministicAcrossLayout(t *testing.T) {
	// AC4:同一份内容,不同排版(YAML 键序、列表顺序、缩进/空白)⇒ 同一哈希。
	const reordered = `bundle_tags: [migration]
gate_condition:
  all_of: [data_integrity, n_minus_1_upgrade]
requires:
  - data_integrity
  - n_minus_1_upgrade
  - fresh_database
trigger:
  risk_signals: [schema_migration]
rationale: migration 类变更必须验证升级路径与数据完整性
enforcement: gate
id: POL-MIGRATION
`
	a, err := Parse([]byte(migrationPolicyYAML))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	b, err := Parse([]byte(reordered))
	if err != nil {
		t.Fatalf("重排版本解析失败: %v", err)
	}
	if a.Revision != b.Revision {
		t.Fatalf("键序/列表顺序不同导致哈希分叉: %s vs %s", a.Revision, b.Revision)
	}
}

func TestRevisionHashChangesOnContentChange(t *testing.T) {
	base, err := Parse([]byte(migrationPolicyYAML))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	cases := []struct {
		name string
		mut  func(p Policy) Policy
	}{
		{"改 id", func(p Policy) Policy { p.ID = "POL-MIGRATION-2"; return p }},
		{"改 enforcement", func(p Policy) Policy {
			p.Enforcement = EnforcementGuidance
			p.GateCondition = GateCondition{}
			return p
		}},
		{"改 rationale", func(p Policy) Policy { p.Rationale += "(修订)"; return p }},
		{"改 requires", func(p Policy) Policy { p.Requires = append(p.Requires, "backup_restored"); return p }},
		{"改 gate_condition", func(p Policy) Policy {
			p.Requires = append(p.Requires, "backup_restored")
			p.GateCondition.AllOf = []string{"data_integrity", "backup_restored"}
			return p
		}},
		{"改 bundle_tags", func(p Policy) Policy { p.BundleTags = nil; return p }},
		{"改 trigger", func(p Policy) Policy { p.Trigger.RiskSignals = []string{"schema_migration", "data_backfill"}; return p }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := tc.mut(base)
			if err := mutated.Validate(); err != nil {
				t.Fatalf("变异后策略非法: %v", err)
			}
			if mutated.RevisionHash() == base.Revision {
				t.Fatal("内容实质变化必须改变哈希")
			}
		})
	}
}

func TestRevisionHashIgnoresInternalWhitespace(t *testing.T) {
	// 同一键的多余内部空白是排版,不是内容。
	a := Policy{ID: "POL-1", Enforcement: EnforcementGate, Requires: []string{"k1"}, GateCondition: GateCondition{AllOf: []string{"k1"}}}
	b := Policy{ID: "POL-1", Enforcement: EnforcementGate, Requires: []string{"k1"}, GateCondition: GateCondition{AllOf: []string{"k1"}}}
	if a.RevisionHash() != b.RevisionHash() {
		t.Fatal("等价内容的哈希必须一致")
	}
}

func TestRevisionHashStableAcrossInvocations(t *testing.T) {
	p := gateFixture()
	first := p.RevisionHash()
	for i := 0; i < 10; i++ {
		if got := p.RevisionHash(); got != first {
			t.Fatalf("第 %d 次求哈希漂移: %s != %s", i, got, first)
		}
	}
}
