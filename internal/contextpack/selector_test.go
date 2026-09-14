package contextpack

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/taskcontract"
	"tally/internal/verification"

	"gopkg.in/yaml.v3"
)

// selectTask 返回带 refs 与 scope 的测试合同;id 必须与 frozenContract 的
// task_id(TASK-001)一致。
func selectTask() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:               "TASK-001",
		Profile:          taskcontract.ProfileStandard,
		Objective:        "为导出 API 提供 redacted ContextPack 投影",
		ProductRefs:      []string{".tally/product/PRD.md#context-engine"},
		RequirementRefs:  []string{".tally/requirements/REQ-018.md"},
		ArchitectureRefs: []string{".tally/architecture/ADR-012.md"},
		DependsOn:        []string{"TASK-019"},
		Scope: taskcontract.Scope{
			AllowedPaths:   []string{"internal/foo"},
			ForbiddenPaths: []string{"internal/foo/secret"},
		},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: "AC-1", Statement: "导出行为符合规格"},
		},
	}
}

func mustSelect(t *testing.T, in SelectInput) *SelectResult {
	t.Helper()
	r, err := Select(in)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	return r
}

// AC1:TaskContract 请求的 product/requirement/architecture refs 与依赖任务
// 必须入包,且每条 ref 在 Pack 记录面有 included 记录。
func TestSelectIncludesRequestedRefs(t *testing.T) {
	p := mustSelect(t, SelectInput{Task: selectTask(), Verification: frozenContract(t, nil)}).Pack

	if got := p.ProductDocs; len(got) != 1 || got[0] != ".tally/product/PRD.md#context-engine" {
		t.Errorf("ProductDocs = %v", got)
	}
	if got := p.RequirementDocs; len(got) != 1 || got[0] != ".tally/requirements/REQ-018.md" {
		t.Errorf("RequirementDocs = %v", got)
	}
	if got := p.ArchitectureDocs; len(got) != 1 || got[0] != ".tally/architecture/ADR-012.md" {
		t.Errorf("ArchitectureDocs = %v", got)
	}
	if got := p.DependencyTasks; len(got) != 1 || got[0] != "TASK-019" {
		t.Errorf("DependencyTasks = %v", got)
	}

	want := map[string]string{
		".tally/product/PRD.md#context-engine": SectionProductDoc,
		".tally/requirements/REQ-018.md":       SectionRequirementDoc,
		".tally/architecture/ADR-012.md":       SectionArchitectureDoc,
		"TASK-019":                             SectionDependencyTask,
	}
	got := map[string]string{}
	for _, s := range p.Included {
		if s.Reason == ReasonProvided {
			got[s.Ref] = s.Source
		}
	}
	for ref, source := range want {
		if got[ref] != source {
			t.Errorf("ref %q 的 included 记录缺失或小节错误: got %q want %q", ref, got[ref], source)
		}
	}
	if p.Hash() == "" {
		t.Errorf("Pack 未计算 context_hash")
	}
}

// AC2:每个候选都有确定性裁决;选中的测试进入 VisibleTests 并登记 Pack.Included;
// 排除按类别登记(forbidden 携带路径,ref 支撑文件登记在对应文档小节)。
func TestSelectRecordsCandidateDecisions(t *testing.T) {
	r := mustSelect(t, SelectInput{
		Task:         selectTask(),
		Verification: frozenContract(t, nil),
		Candidates: []Candidate{
			{Path: "internal/foo/util.go", Content: "package foo"},
			{Path: "internal/foo/b_test.go", Content: "package foo"},
			{Path: "internal/foo/a_test.go", Content: "package foo"},
			{Path: "internal/foo/secret/hidden_test.go"},
			{Path: "docs/unrelated.md"},
			{Path: ".tally/product/PRD.md"},
		},
	})
	p := r.Pack

	if got := p.VisibleTests; len(got) != 2 || got[0] != "internal/foo/a_test.go" || got[1] != "internal/foo/b_test.go" {
		t.Fatalf("VisibleTests = %v", got)
	}
	included := map[string]string{}
	for _, s := range r.Included {
		included[s.Ref] = s.Reason
	}
	if included["internal/foo/a_test.go"] != ReasonScopeVisibleTest || included["internal/foo/b_test.go"] != ReasonScopeVisibleTest {
		t.Errorf("测试候选 include 原因错误: %v", included)
	}

	excluded := map[string]Selection{}
	for _, s := range r.Excluded {
		excluded[s.Ref] = s
	}
	if s := excluded["internal/foo/util.go"]; s.Reason != ReasonNonTestCode || s.Source != SectionCandidate {
		t.Errorf("util.go 裁决错误: %+v", s)
	}
	if s := excluded["internal/foo/secret/hidden_test.go"]; s.Reason != ReasonForbiddenPath || s.Ref == "" {
		t.Errorf("forbidden 裁决错误(非 held-out 排除应携带路径): %+v", s)
	}
	if s := excluded["docs/unrelated.md"]; s.Reason != ReasonOutOfScope {
		t.Errorf("越界裁决错误: %+v", s)
	}
	if s := excluded[".tally/product/PRD.md"]; s.Reason != ReasonRefBacked || s.Source != SectionProductDoc {
		t.Errorf("ref 支撑文件裁决错误: %+v", s)
	}

	visibleRecords := 0
	for _, s := range p.Included {
		if s.Source == SectionVisibleTest && s.Reason == ReasonProvided {
			visibleRecords++
		}
	}
	if visibleRecords != 2 {
		t.Errorf("Pack.Included 缺少 visible_test 记录: %+v", p.Included)
	}
}

// AC3:held-out 内部不得经选择路径进入实现侧包 —— 整个 SelectResult(Pack +
// 双台账)序列化后不含任何 held-out 标记内容;held-out 派生的排除记录不携带路径。
func TestSelectFiltersHeldOutInternals(t *testing.T) {
	r := mustSelect(t, SelectInput{
		Task:         selectTask(),
		Verification: frozenContract(t, nil),
		Candidates: []Candidate{
			{Path: "internal/foo/a_test.go"},
			{Path: "internal/foo/b_test.go"},
		},
	})
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	assertAbsent(t, string(data), secretStore, secretRef, secretFingerprint, secretBindingID)

	// 纵深防御:候选路径与 held_out binding 声称的 repo path 重合时(合法冻结
	// 契约中不可能出现,§8.4.1 冻结校验保证),分类必须排除,且记录不携带路径。
	ix := &scopeIndex{
		refFiles:    map[string]string{},
		heldOut:     map[string]bool{"internal/foo/leak_test.go": true},
		visibleTest: map[string]bool{},
	}
	section, include, reason := ix.classify("internal/foo/leak_test.go")
	if include {
		t.Fatalf("held-out 路径候选被选中")
	}
	rec := candidateRecord(section, "internal/foo/leak_test.go", reason)
	if rec.Ref != "" || rec.Reason != ReasonHeldOutProvenance {
		t.Errorf("held-out 排除记录回泄路径: %+v", rec)
	}
}

// scopeIndex 推导:visible test binding 的 repo path 登记为权威指定可见测试
// (scope 之外也选中);held_out binding(store/ref 引用)不产生任何路径登记。
func TestSelectScopeIndexFromBindings(t *testing.T) {
	f := frozenContract(t, func(c *verification.Contract) {
		c.Acceptances[0].EvidenceBindings = append(c.Acceptances[0].EvidenceBindings, verification.Binding{
			ID:               "b-scopeless",
			AuthoringMode:    verification.AuthoringPreCommit,
			VisibilityPolicy: verification.VisibilityVisible,
			Evidence: verification.EvidenceRef{
				Type:     verification.EvidenceTest,
				Path:     "third_party/e2e_test.go",
				Selector: "TestE2E",
			},
		})
	})
	task := selectTask()
	ix := buildScopeIndex(&task, f)
	if !ix.visibleTest["third_party/e2e_test.go"] {
		t.Errorf("visible test binding path 未登记: %v", ix.visibleTest)
	}
	if len(ix.heldOut) != 0 {
		t.Errorf("held_out binding 不应产生路径登记: %v", ix.heldOut)
	}

	r := mustSelect(t, SelectInput{
		Task:         task,
		Verification: f,
		Candidates:   []Candidate{{Path: "third_party/e2e_test.go"}},
	})
	if got := r.Pack.VisibleTests; len(got) != 1 || got[0] != "third_party/e2e_test.go" {
		t.Fatalf("VisibleTests = %v", got)
	}
	included := map[string]string{}
	for _, s := range r.Included {
		included[s.Ref] = s.Reason
	}
	if included["third_party/e2e_test.go"] != ReasonVisibleBinding {
		t.Errorf("binding 指定测试的 include 原因错误: %v", included)
	}
}

// AC4:token 预算确定性生效 —— 固定估算规则 + 路径序贪心;同输入重复选择与
// 乱序输入产生完全相同的结果(Pack 与台账)与哈希。
func TestSelectTokenBudgetDeterministic(t *testing.T) {
	task := selectTask()
	frozen := frozenContract(t, nil)
	cands := []Candidate{
		{Path: "internal/foo/b_test.go", Content: strings.Repeat("b", 8000)}, // 2000 tokens
		{Path: "internal/foo/a_test.go", Content: strings.Repeat("a", 400)},  // 100 tokens
		{Path: "internal/foo/c_test.go", Content: strings.Repeat("c", 40)},   // 10 tokens
		{Path: "internal/foo/util.go"},
	}
	r := mustSelect(t, SelectInput{
		Task:           task,
		Verification:   frozen,
		Candidates:     cands,
		MaxTotalTokens: 120,
	})

	if got := r.Pack.VisibleTests; len(got) != 2 || got[0] != "internal/foo/a_test.go" || got[1] != "internal/foo/c_test.go" {
		t.Fatalf("token 预算截断错误: %v", got)
	}
	var budgetExcluded []string
	for _, s := range r.Excluded {
		if s.Reason == ReasonTokenBudget {
			budgetExcluded = append(budgetExcluded, s.Ref)
		}
	}
	if len(budgetExcluded) != 1 || budgetExcluded[0] != "internal/foo/b_test.go" {
		t.Errorf("token 预算排除记录错误: %v", budgetExcluded)
	}

	again := mustSelect(t, SelectInput{
		Task:           task,
		Verification:   frozen,
		Candidates:     cands,
		MaxTotalTokens: 120,
	})
	if !reflect.DeepEqual(*r, *again) {
		t.Errorf("同输入两次 Select 结果不一致")
	}

	shuffled := []Candidate{cands[2], cands[1], cands[3], cands[0]}
	reshuffled := mustSelect(t, SelectInput{
		Task:           task,
		Verification:   frozen,
		Candidates:     shuffled,
		MaxTotalTokens: 120,
	})
	if !reflect.DeepEqual(*r, *reshuffled) {
		t.Errorf("输入顺序影响了选择结果")
	}
}

// Build 的每节条目预算与 token 预算叠加生效,且 Pack 记录 Budget 并登记排除。
func TestSelectBudgetPassthrough(t *testing.T) {
	r := mustSelect(t, SelectInput{
		Task:         selectTask(),
		Verification: frozenContract(t, nil),
		Budget:       Budget{MaxItemsPerSection: 1},
		Candidates: []Candidate{
			{Path: "internal/foo/a_test.go"},
			{Path: "internal/foo/b_test.go"},
		},
	})
	if len(r.Pack.VisibleTests) != 1 || r.Pack.VisibleTests[0] != "internal/foo/a_test.go" {
		t.Fatalf("每节条目预算未生效: %v", r.Pack.VisibleTests)
	}
	if r.Pack.Budget.MaxItemsPerSection != 1 {
		t.Errorf("Budget 未记录进 Pack: %+v", r.Pack.Budget)
	}
	found := false
	for _, s := range r.Pack.Excluded {
		if s.Reason == ReasonBudget && s.Ref == "internal/foo/b_test.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("Pack.Excluded 缺少条目预算排除记录: %+v", r.Pack.Excluded)
	}
}

// Select 输入校验:nil 契约、空/重复候选路径、非法合同、任务 id 不一致都报错。
func TestSelectInputValidation(t *testing.T) {
	if _, err := Select(SelectInput{Task: selectTask()}); !errors.Is(err, ErrNilContract) {
		t.Errorf("nil 契约应返回 ErrNilContract, got %v", err)
	}
	if _, err := Select(SelectInput{Task: selectTask(), Verification: frozenContract(t, nil),
		Candidates: []Candidate{{Path: "  "}}}); err == nil {
		t.Errorf("空白路径候选应报错")
	}
	if _, err := Select(SelectInput{Task: selectTask(), Verification: frozenContract(t, nil),
		Candidates: []Candidate{{Path: "a"}, {Path: "a"}}}); err == nil {
		t.Errorf("重复路径候选应报错")
	}
	mismatched := selectTask()
	mismatched.ID = "TASK-002"
	if _, err := Select(SelectInput{Task: mismatched, Verification: frozenContract(t, nil)}); err == nil {
		t.Errorf("任务 id 不一致应报错")
	}
	invalid := selectTask()
	invalid.Objective = ""
	if _, err := Select(SelectInput{Task: invalid, Verification: frozenContract(t, nil)}); err == nil {
		t.Errorf("非法 TaskContract 应报错")
	}
}

// EstimateTokens 是固定确定性规则:每 4 字节 ≈ 1 token,向上取整,空为 0。
func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"a", 1},
		{"abcd", 1},
		{"abcde", 2},
		{strings.Repeat("x", 400), 100},
	}
	for _, c := range cases {
		if got := EstimateTokens(c.in); got != c.want {
			t.Errorf("EstimateTokens(%d bytes) = %d, want %d", len(c.in), got, c.want)
		}
	}
}
