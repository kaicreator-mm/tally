package engineeringpolicy

import (
	"errors"
	"strings"
	"testing"
)

// migrationPolicyYAML 是 PRD §8.3 的 migration 示例(补全 PRD §8.1 其余字段)。
const migrationPolicyYAML = `id: POL-MIGRATION
enforcement: gate
rationale: migration 类变更必须验证升级路径与数据完整性
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
  - n_minus_1_upgrade
  - data_integrity
gate_condition:
  all_of:
    - n_minus_1_upgrade
    - data_integrity
bundle_tags:
  - migration
`

func TestParseAcceptsValidGatePolicy(t *testing.T) {
	p, err := Parse([]byte(migrationPolicyYAML))
	if err != nil {
		t.Fatalf("合法 gate 策略被拒: %v", err)
	}
	if p.ID != "POL-MIGRATION" {
		t.Fatalf("id = %q", p.ID)
	}
	if p.Enforcement != EnforcementGate {
		t.Fatalf("enforcement = %q", p.Enforcement)
	}
	if len(p.Requires) != 3 {
		t.Fatalf("requires = %v", p.Requires)
	}
	if len(p.GateCondition.AllOf) != 2 {
		t.Fatalf("gate_condition.all_of = %v", p.GateCondition.AllOf)
	}
	if !strings.HasPrefix(p.Revision, "sha256:") {
		t.Fatalf("Revision 未填充内容哈希: %q", p.Revision)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	// AC1:严格模式,schema 之外的字段直接报错,不静默丢弃。
	cases := []struct {
		name string
		yaml string
	}{
		{"拼写错误的 gate 字段", "id: P\nenforcement: gate\ngate_conditions:\n  all_of: [x]\nrequires: [x]\n"},
		{"gate 简写", "id: P\nenforcement: gate\nrequires: [x]\ngate:\n  all_of: [x]\n"},
		{"顶层私货", "id: P\nenforcement: gate\nrequires: [x]\ngate_condition:\n  all_of: [x]\nextra: 1\n"},
		{"嵌套私货", "id: P\nenforcement: gate\nrequires: [x]\ngate_condition:\n  all_of: [x]\n  when: always\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatal("期望被拒,实际通过")
			}
			if !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("期望 ErrInvalidPolicy,得到: %v", err)
			}
		})
	}
}

func TestParseRejectsExecutionPolicyFields(t *testing.T) {
	// 决策锁 6:ExecutionPolicy != EngineeringPolicy。执行策略字段出现在
	// 工程策略里是越权,必须点名拒绝,不得忽略。
	for _, field := range []string{
		"max_iterations", "token_budget", "wall_budget", "timeout", "network",
		"shell", "credentials", "retry", "runtime", "model", "stop_conditions",
	} {
		t.Run(field, func(t *testing.T) {
			yaml := "id: P\nenforcement: gate\nrequires: [x]\ngate_condition:\n  all_of: [x]\n" + field + ": 1\n"
			_, err := Parse([]byte(yaml))
			if err == nil {
				t.Fatalf("字段 %s 应被拒绝", field)
			}
			if !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("期望 ErrInvalidPolicy,得到: %v", err)
			}
			if !strings.Contains(err.Error(), field) || !strings.Contains(err.Error(), "ExecutionPolicy") {
				t.Fatalf("错误应点名字段与归属层,得到: %v", err)
			}
		})
	}
}

func TestParseRejectsMalformedDocuments(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"空文档", ""},
		{"纯注释", "# 只有注释\n"},
		{"多文档", migrationPolicyYAML + "---\nid: POL-OTHER\n"},
		{"YAML 语法错误", "id: [unclosed\n"},
		{"enforcement 缺席", "id: P\nrequires: [x]\n"},
		{"enforcement 未知", "id: P\nenforcement: block\n"},
		{"guidance 携带 gate_condition", "id: P\nenforcement: guidance\ngate_condition:\n  all_of: [x]\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatal("期望被拒,实际通过")
			}
			if !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("期望 ErrInvalidPolicy,得到: %v", err)
			}
		})
	}
}

func TestParseAcceptsGuidancePolicy(t *testing.T) {
	// AC2:guidance 可表示(schema 保留),但没有 gate_condition 可携带。
	const yaml = `id: POL-STYLE
enforcement: guidance
rationale: 提示 agent 遵循仓库提交规范
requires:
  - conventional_commits
`
	p, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("合法 guidance 策略被拒: %v", err)
	}
	if p.Enforcement != EnforcementGuidance {
		t.Fatalf("enforcement = %q", p.Enforcement)
	}
	if p.Revision == "" {
		t.Fatal("guidance 策略同样要有内容修订哈希")
	}
}

func TestParseNormalizesWhitespace(t *testing.T) {
	const yaml = "id: POL-X  \nenforcement: gate\nrationale: \"why  \"\nrequires:\n  - \"  k1  \"\n  - k2\ngate_condition:\n  all_of:\n    - k1\n"
	p, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if p.ID != "POL-X" || p.Rationale != "why" {
		t.Fatalf("标量未去空白: id=%q rationale=%q", p.ID, p.Rationale)
	}
	if p.Requires[0] != "k1" {
		t.Fatalf("列表条目未去空白: %v", p.Requires)
	}
}
