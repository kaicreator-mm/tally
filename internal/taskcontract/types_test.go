package taskcontract

import (
	"reflect"
	"strings"
	"testing"
)

// AC4 的结构守卫:TaskContract 上不允许出现任何执行预算/网络/shell/凭证类
// 字段(PRD §7.4:这些由 ExecutionPolicy 管理)。解析层的拒绝测试在
// parse_test.go;这里守的是类型本身 —— 字段集合必须是白名单里的领域字段。
func TestTaskContractFieldWhitelist(t *testing.T) {
	allowed := map[string]bool{
		"ID": true, "Profile": true, "Title": true, "Objective": true,
		"ProductRefs": true, "RequirementRefs": true, "ArchitectureRefs": true,
		"DependsOn": true, "Scope": true, "Acceptance": true,
		"DocumentationImpact": true, "Hash": true,
	}
	tp := reflect.TypeOf(TaskContract{})
	for i := 0; i < tp.NumField(); i++ {
		name := tp.Field(i).Name
		if !allowed[name] {
			t.Errorf("TaskContract 出现白名单外的字段 %s;执行预算/网络/凭证类字段不得进入 TaskContract(PRD §7.4)", name)
		}
	}

	// 双保险:即使白名单漏更新,已知的执行字段名也按名字直接拦。
	forbidden := []string{
		"MaxIterations", "TokenBudget", "WallBudget", "Timeout", "Network",
		"Shell", "Credentials", "Retry", "Runtime", "Model", "StopConditions",
		"Budget", "MaxAttempts",
	}
	for i := 0; i < tp.NumField(); i++ {
		name := tp.Field(i).Name
		for _, f := range forbidden {
			if strings.EqualFold(name, f) {
				t.Errorf("TaskContract 字段 %s 属于 ExecutionPolicy 语义,不得存在", name)
			}
		}
	}
}

// AC2:independence_required 属于 Acceptance Clause,是累积等级
// none < source < information(PRD §7.3 / §8.5)。
func TestIndependenceLevelOrdering(t *testing.T) {
	if !(IndependenceNone.Rank() < IndependenceSource.Rank() &&
		IndependenceSource.Rank() < IndependenceInformation.Rank()) {
		t.Fatal("等级序错误: 应为 none < source < information")
	}
	for _, l := range []IndependenceLevel{IndependenceNone, IndependenceSource, IndependenceInformation} {
		if !l.IsValid() {
			t.Fatalf("%v 应合法", l)
		}
	}
	if IndependenceLevel("strong").IsValid() || IndependenceLevel("").IsValid() {
		t.Fatal("非法/空等级不应 IsValid")
	}
	if IndependenceLevel("bogus").Rank() != -1 {
		t.Fatal("非法等级 Rank 应为 -1,永不参与合格比较")
	}
}

func TestIndependenceSatisfies(t *testing.T) {
	// 要求 none:任何合法达到等级都满足。
	for _, got := range []IndependenceLevel{IndependenceNone, IndependenceSource, IndependenceInformation} {
		if !IndependenceNone.Satisfies(got) {
			t.Fatalf("none 要求应被 %v 满足", got)
		}
	}
	// 累积语义:要求 source,information 也满足;要求 information,source 不满足。
	if !IndependenceSource.Satisfies(IndependenceInformation) {
		t.Fatal("source 要求应被 information 满足(累积等级)")
	}
	if IndependenceInformation.Satisfies(IndependenceSource) {
		t.Fatal("information 要求不应被 source 满足")
	}
	if IndependenceInformation.Satisfies(IndependenceNone) {
		t.Fatal("information 要求不应被 none 满足")
	}
	// 非法值永不满足、也不构成要求。
	if IndependenceSource.Satisfies(IndependenceLevel("bogus")) ||
		IndependenceLevel("bogus").Satisfies(IndependenceSource) {
		t.Fatal("非法值不应参与满足判断")
	}
}

// 未声明的独立性要求默认 none(Independence() 的默认规则):
// 独立性只能显式声明获得,不能凭空默认赋予。
func TestAcceptanceIndependenceDefault(t *testing.T) {
	undclared := AcceptanceClause{ID: "AC-01", Statement: "x"}
	if got := undclared.Independence(); got != IndependenceNone {
		t.Fatalf("未声明默认 = %v, want none", got)
	}
	declared := AcceptanceClause{ID: "AC-02", Statement: "x", IndependenceRequired: IndependenceInformation}
	if got := declared.Independence(); got != IndependenceInformation {
		t.Fatalf("显式声明被改写: %v", got)
	}
}

// IsValidProfile 只认 PRD §7.2 的三个名字。
func TestIsValidProfile(t *testing.T) {
	for _, p := range []string{ProfileLight, ProfileStandard, ProfileStrict} {
		if !IsValidProfile(p) {
			t.Fatalf("%v 应合法", p)
		}
	}
	for _, p := range []string{"", "standard+", "STANDARD", "fast"} {
		if IsValidProfile(p) {
			t.Fatalf("%q 不应合法", p)
		}
	}
}

// TaskContract.Independence 按条款 id 查询生效等级;找不到时显式 false,
// 不允许调用方把「没这条」当成「none 要求」。
func TestContractIndependenceLookup(t *testing.T) {
	c := TaskContract{Acceptance: []AcceptanceClause{
		{ID: "AC-01", IndependenceRequired: IndependenceSource},
	}}
	if got, ok := c.Independence("AC-01"); !ok || got != IndependenceSource {
		t.Fatalf("AC-01 = %v,%v", got, ok)
	}
	if _, ok := c.Independence("nope"); ok {
		t.Fatal("查不到的条款应返回 false")
	}
}

// 规范化辅助:collapseSpace 折叠任意空白,sortedUnique 排序去重且不改原切片。
func TestCanonicalHelpers(t *testing.T) {
	if got := collapseSpace("  a\t\n b   c \n"); got != "a b c" {
		t.Fatalf("collapseSpace = %q", got)
	}
	orig := []string{"b", "a", "b", ""}
	got := sortedUnique(orig, []string{"a", "c"})
	want := "a b c"
	if strings.Join(got, " ") != want {
		t.Fatalf("sortedUnique = %v, want [%s]", got, want)
	}
	if len(orig) != 4 || orig[0] != "b" {
		t.Fatalf("sortedUnique 修改了原切片: %v", orig)
	}
}
