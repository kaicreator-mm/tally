// conflict_test.go 拥有 P3-05 的表驱动冲突测试:AC 的矛盾/重叠夹具产出显式
// conflict/warning;确定性(同输入同输出序列,顺序无关);保守方向(零值触发
// 面宁多报给人看)。
package policygovernance

import (
	"strings"
	"testing"

	"tally/internal/engineeringpolicy"
)

// p(const) 便捷构造:直接构造 Policy(域内分析消费解析后的值;解析语义已由
// engineeringpolicy 与 P3-01/P3-03 测试覆盖)。
func p(id, enforcement string, signals, requires, allOf []string) engineeringpolicy.Policy {
	pol := engineeringpolicy.Policy{
		ID:          id,
		Enforcement: engineeringpolicy.Enforcement(enforcement),
		Trigger:     engineeringpolicy.Trigger{RiskSignals: signals},
		Requires:    requires,
	}
	if enforcement == "gate" {
		pol.GateCondition = engineeringpolicy.GateCondition{AllOf: allOf}
	}
	return pol
}

func find(cs []Conflict, kind ConflictKind) []Conflict {
	var out []Conflict
	for _, c := range cs {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

// AC:已知矛盾/重叠夹具产出显式 conflict/warning(info 的叠加面同样可见)。
func TestAnalyzeConflictsTable(t *testing.T) {
	migrationSignals := []string{"schema_migration"}

	cases := []struct {
		name      string
		policies  []engineeringpolicy.Policy
		wantKinds map[ConflictKind]int
	}{
		{
			name:      "干净集合:零发现",
			policies:  []engineeringpolicy.Policy{p("A", "gate", migrationSignals, []string{"k1"}, []string{"k1"})},
			wantKinds: map[ConflictKind]int{},
		},
		{
			name: "矛盾夹具:重复 id(内容不同)",
			policies: []engineeringpolicy.Policy{
				p("A", "gate", migrationSignals, []string{"k1"}, []string{"k1"}),
				p("A", "gate", migrationSignals, []string{"k2"}, []string{"k2"}),
			},
			wantKinds: map[ConflictKind]int{ConflictDuplicateID: 1},
		},
		{
			name: "重叠夹具:触发轴重叠的双 gate → 叠加 info",
			policies: []engineeringpolicy.Policy{
				p("A", "gate", migrationSignals, []string{"k1"}, []string{"k1"}),
				p("B", "gate", migrationSignals, []string{"k2"}, []string{"k2"}),
			},
			wantKinds: map[ConflictKind]int{ConflictStackedGates: 1},
		},
		{
			name: "意图矛盾:同触发面同门禁键,强度不同 → mismatch",
			policies: []engineeringpolicy.Policy{
				p("A", "gate", migrationSignals, []string{"k1"}, []string{"k1"}),
				p("B", "guidance", migrationSignals, []string{"k1"}, nil),
			},
			wantKinds: map[ConflictKind]int{ConflictEnforcementMismatch: 1},
		},
		{
			name: "完全遮蔽:全同语义不同 id → shadowed",
			policies: []engineeringpolicy.Policy{
				p("A", "gate", migrationSignals, []string{"k1"}, []string{"k1"}),
				p("B", "gate", migrationSignals, []string{"k1"}, []string{"k1"}),
			},
			wantKinds: map[ConflictKind]int{ConflictShadowedDuplicate: 1},
		},
		{
			name: "触发轴不相交:零发现",
			policies: []engineeringpolicy.Policy{
				p("A", "gate", []string{"schema_migration"}, []string{"k1"}, []string{"k1"}),
				p("B", "gate", []string{"perf_regression"}, []string{"k2"}, []string{"k2"}),
			},
			wantKinds: map[ConflictKind]int{},
		},
		{
			name: "复合:身份冲突 + 叠加 + mismatch 同时可见",
			policies: []engineeringpolicy.Policy{
				p("A", "gate", migrationSignals, []string{"k1"}, []string{"k1"}),
				p("A", "guidance", migrationSignals, []string{"k1"}, nil), // 同 id 变体:身份冲突上报,组内去重不级联
				p("B", "gate", migrationSignals, []string{"k2"}, []string{"k2"}),
				p("C", "guidance", migrationSignals, []string{"k1"}, nil),
			},
			wantKinds: map[ConflictKind]int{
				ConflictDuplicateID:         1,
				ConflictStackedGates:        1,
				ConflictEnforcementMismatch: 1,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AnalyzeConflicts(tc.policies)
			counts := map[ConflictKind]int{}
			for _, c := range got {
				counts[c.Kind]++
				if c.Detail == "" {
					t.Errorf("发现 %s 的 Detail 不得为空(人可读说明是交付面)", c.Kind)
				}
				if len(c.PolicyIDs) == 0 {
					t.Errorf("发现 %s 必须携带涉及策略 ID", c.Kind)
				}
			}
			if len(counts) != len(tc.wantKinds) {
				t.Fatalf("发现种类 = %v, want %v", counts, tc.wantKinds)
			}
			for kind, n := range tc.wantKinds {
				if counts[kind] != n {
					t.Errorf("kind %s 次数 = %d, want %d", kind, counts[kind], n)
				}
			}
		})
	}
}

// 确定性 AC:同一集合、不同输入顺序 → 完全相同的发现序列。
func TestAnalyzeConflictsDeterministic(t *testing.T) {
	set := []engineeringpolicy.Policy{
		p("A", "gate", []string{"schema_migration"}, []string{"k1"}, []string{"k1"}),
		p("B", "gate", []string{"schema_migration"}, []string{"k2"}, []string{"k2"}),
		p("C", "gate", []string{"schema_migration"}, []string{"k1"}, []string{"k1"}),
		p("A", "gate", []string{"schema_migration"}, []string{"k1"}, []string{"k1"}),
	}
	first := AnalyzeConflicts(set)
	for i := 0; i < 3; i++ {
		// 旋转输入顺序,输出必须逐字节一致(排序在输出侧,不在输入侧)。
		rotated := append(append([]engineeringpolicy.Policy(nil), set[1:]...), set[0])
		got := AnalyzeConflicts(rotated)
		if len(got) != len(first) {
			t.Fatalf("第 %d 轮发现数漂移: %d vs %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j].Kind != first[j].Kind || strings.Join(got[j].PolicyIDs, ",") != strings.Join(first[j].PolicyIDs, ",") {
				t.Fatalf("第 %d 轮发现序列漂移: %+v vs %+v", i, got[j], first[j])
			}
		}
	}
}

// 保守方向:零值触发面(无条件触发 = 匹配一切)与任何策略都按重叠上报。
func TestAnalyzeConflictsZeroTriggerConservative(t *testing.T) {
	wildcard := p("WILD", "gate", nil, []string{"k0"}, []string{"k0"})
	specific := p("SPEC", "gate", []string{"schema_migration"}, []string{"k1"}, []string{"k1"})
	got := AnalyzeConflicts([]engineeringpolicy.Policy{wildcard, specific})
	if n := len(find(got, ConflictStackedGates)); n != 1 {
		t.Fatalf("零值触发面应保守上报叠加,得到 %d 条 stacked_gates", n)
	}
}
