package taskcontract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// minimalExample 是能通过解析的最小合同(standard + 一条未声明独立性要求的条款)。
const minimalExample = `id: TASK-001
profile: standard
objective: make the thing verifiable
scope:
  allowed_paths:
    - internal/**
acceptance:
  - id: AC-01
    statement: it works
`

// prdExample 是 PRD §7.3 的 TaskContract 示例(逐字,仅注掉无关说明)。
const prdExample = `id: TASK-023
profile: standard

title: Add stale job lease recovery

objective: >
  Prevent jobs from remaining permanently in running state
  after worker crashes.

product_refs:
  - .tally/product/PRD.md#job-reliability

requirement_refs:
  - .tally/requirements/REQ-018.md

architecture_refs:
  - .tally/architecture/ADR-012.md

depends_on:
  - TASK-019

scope:
  allowed_paths:
    - internal/jobs/**
    - internal/store/**
    - migrations/**

  forbidden_paths:
    - internal/auth/**

acceptance:
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source

  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information

  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none

documentation_impact:
  - architecture
  - schema
`

func mustParse(t *testing.T, data string) TaskContract {
	t.Helper()
	c, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return c
}

// AC1:objective、refs、dependencies、allowed/forbidden scope、profile、
// acceptance clauses、documentation impact 全部落进类型并被解析。
func TestParsePRDExample(t *testing.T) {
	c := mustParse(t, prdExample)

	if c.ID != "TASK-023" || c.Profile != ProfileStandard || c.Title != "Add stale job lease recovery" {
		t.Fatalf("基础字段解析错误: %+v", c)
	}
	wantObjective := "Prevent jobs from remaining permanently in running state after worker crashes."
	if c.Objective != wantObjective {
		t.Fatalf("objective 折叠错误: got %q want %q", c.Objective, wantObjective)
	}
	if len(c.ProductRefs) != 1 || c.ProductRefs[0] != ".tally/product/PRD.md#job-reliability" {
		t.Fatalf("product_refs: %v", c.ProductRefs)
	}
	if len(c.RequirementRefs) != 1 || c.RequirementRefs[0] != ".tally/requirements/REQ-018.md" {
		t.Fatalf("requirement_refs: %v", c.RequirementRefs)
	}
	if len(c.ArchitectureRefs) != 1 || c.ArchitectureRefs[0] != ".tally/architecture/ADR-012.md" {
		t.Fatalf("architecture_refs: %v", c.ArchitectureRefs)
	}
	if len(c.DependsOn) != 1 || c.DependsOn[0] != "TASK-019" {
		t.Fatalf("depends_on: %v", c.DependsOn)
	}
	wantAllowed := []string{"internal/jobs/**", "internal/store/**", "migrations/**"}
	if len(c.Scope.AllowedPaths) != 3 {
		t.Fatalf("allowed_paths: %v", c.Scope.AllowedPaths)
	}
	for i, p := range wantAllowed {
		if c.Scope.AllowedPaths[i] != p {
			t.Fatalf("allowed_paths[%d] = %q, want %q", i, c.Scope.AllowedPaths[i], p)
		}
	}
	if len(c.Scope.ForbiddenPaths) != 1 || c.Scope.ForbiddenPaths[0] != "internal/auth/**" {
		t.Fatalf("forbidden_paths: %v", c.Scope.ForbiddenPaths)
	}
	if len(c.Acceptance) != 3 {
		t.Fatalf("acceptance 条数: %d", len(c.Acceptance))
	}
	if len(c.DocumentationImpact) != 2 || c.DocumentationImpact[0] != "architecture" || c.DocumentationImpact[1] != "schema" {
		t.Fatalf("documentation_impact: %v", c.DocumentationImpact)
	}
	if !strings.HasPrefix(c.Hash, "sha256:") || len(c.Hash) != len("sha256:")+64 {
		t.Fatalf("Hash 未填充或格式错误: %q", c.Hash)
	}
}

// AC2:independence_required 属于 Acceptance Clause,且 none/source/information
// 三个等级都能声明并被原样读出。
func TestParseIndependenceLevels(t *testing.T) {
	c := mustParse(t, prdExample)
	if got, ok := c.Independence("AC-01"); !ok || got != IndependenceSource {
		t.Fatalf("AC-01 independence = %v,%v", got, ok)
	}
	if got, ok := c.Independence("AC-02"); !ok || got != IndependenceInformation {
		t.Fatalf("AC-02 independence = %v,%v", got, ok)
	}
	if got, ok := c.Independence("AC-03"); !ok || got != IndependenceNone {
		t.Fatalf("AC-03 independence = %v,%v", got, ok)
	}
	if _, ok := c.Independence("AC-404"); ok {
		t.Fatal("不存在的条款 id 不应命中")
	}

	// 未声明 independence_required ⇒ 默认 none(默认只能更弱,不能凭空赋权)。
	minimal := mustParse(t, minimalExample)
	if got := minimal.Acceptance[0].Independence(); got != IndependenceNone {
		t.Fatalf("未声明时默认 = %v, want none", got)
	}

	// 非法等级必须报错,不能静默通过。
	bad := strings.Replace(prdExample, "independence_required: source", "independence_required: absolute", 1)
	if _, err := Parse([]byte(bad)); err == nil || !strings.Contains(err.Error(), "independence_required") {
		t.Fatalf("非法 independence_required 应报错, got %v", err)
	}
}

// AC3:修订哈希由内容推导 —— 键序、列表顺序、排版折叠不影响哈希;
// 内容实质变化必须改变哈希。
func TestRevisionHashStable(t *testing.T) {
	a := mustParse(t, prdExample)

	// 同一内容,换 YAML 键序、换列表顺序、换折叠写法。
	reordered := `documentation_impact: [schema, architecture]
acceptance:
  - {id: AC-03, statement: retries must respect max_attempts, independence_required: none}
  - {id: AC-01, statement: stale leases can be reclaimed, independence_required: source}
  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information
scope:
  forbidden_paths: [internal/auth/**]
  allowed_paths: [migrations/**, internal/jobs/**, internal/store/**]
depends_on: [TASK-019]
architecture_refs: [.tally/architecture/ADR-012.md]
requirement_refs: [.tally/requirements/REQ-018.md]
product_refs: [.tally/product/PRD.md#job-reliability]
objective: "Prevent jobs from remaining permanently in running state after worker crashes."
title: Add stale job lease recovery
profile: standard
id: TASK-023
`
	b := mustParse(t, reordered)
	if a.Hash != b.Hash {
		t.Fatalf("同内容不同排版哈希不同:\n%s\n%s", a.Hash, b.Hash)
	}

	// 内容实质变化(改一条 statement)⇒ 哈希必须变。
	changed := strings.Replace(prdExample,
		"statement: retries must respect max_attempts",
		"statement: retries must respect max attempts limit", 1)
	c := mustParse(t, changed)
	if a.Hash == c.Hash {
		t.Fatal("内容变化后哈希未变")
	}

	// 哈希只由内容决定:Parse 得到的 Hash 与直接按类型算的 RevisionHash 一致。
	if a.Hash != a.RevisionHash() {
		t.Fatalf("Hash %q != RevisionHash %q", a.Hash, a.RevisionHash())
	}
}

// AC4:执行预算/网络等字段不属于 TaskContract —— 出现任何一个都必须被点名拒绝。
func TestParseRejectsExecutionFields(t *testing.T) {
	for _, f := range []string{
		"max_iterations", "token_budget", "wall_budget", "timeout", "network",
		"shell", "credentials", "retry", "runtime", "model", "stop_conditions",
	} {
		yamlDoc := prdExample + fmt.Sprintf("%s: 5\n", f)
		_, err := Parse([]byte(yamlDoc))
		if err == nil {
			t.Errorf("字段 %s 被静默接受;应报「不属于 TaskContract」", f)
			continue
		}
		if !strings.Contains(err.Error(), f) || !strings.Contains(err.Error(), "ExecutionPolicy") {
			t.Errorf("字段 %s 的报错未点名字段与归属: %v", f, err)
		}
	}

	// 非 scalar 取值(bool/list)同样按「字段出现」拒绝,而不是类型错误先炸。
	if _, err := Parse([]byte(prdExample + "network: true\n")); err == nil {
		t.Error("network: true 被静默接受")
	}
	if _, err := Parse([]byte(prdExample + "retry: {max: 3}\n")); err == nil {
		t.Error("retry: {max: 3} 被静默接受")
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	if _, err := Parse([]byte(prdExample + "bogus_field: 1\n")); err == nil {
		t.Fatal("未知字段应报错(严格模式)")
	} else if !strings.Contains(err.Error(), "bogus_field") {
		t.Fatalf("报错应包含未知字段名: %v", err)
	}
}

func TestParseRejectsStructurallyBadYAML(t *testing.T) {
	if _, err := Parse(nil); err == nil || !strings.Contains(err.Error(), "空 YAML 文档") {
		t.Fatalf("空文档应报错: %v", err)
	}
	if _, err := Parse([]byte("")); err == nil || !strings.Contains(err.Error(), "空 YAML 文档") {
		t.Fatalf("空串应报错: %v", err)
	}
	two := prdExample + "---\n" + prdExample
	if _, err := Parse([]byte(two)); err == nil || !strings.Contains(err.Error(), "单文档") {
		t.Fatalf("多文档应报错: %v", err)
	}
	if _, err := Parse([]byte("id: [不是, 标量]\n")); err == nil {
		t.Fatal("类型错误应报错")
	}
}

// Validate 的硬约束(PRD §7.1/§7.2):必填项、profile 合法性、acceptance
// 与 profile 的关系、条款 id 唯一性。
func TestValidate(t *testing.T) {
	valid := mustParse(t, prdExample)
	if err := valid.Validate(); err != nil {
		t.Fatalf("PRD 示例应通过 Validate: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*TaskContract)
		wantErr string
	}{
		{"缺 id", func(c *TaskContract) { c.ID = "" }, "id"},
		{"id 含空白", func(c *TaskContract) { c.ID = "TASK 023" }, "空白"},
		{"profile 非法", func(c *TaskContract) { c.Profile = "extreme" }, "profile"},
		{"缺 objective", func(c *TaskContract) { c.Objective = "  " }, "objective"},
		{"空 allowed_paths", func(c *TaskContract) { c.Scope.AllowedPaths = nil }, "allowed_paths"},
		{"standard 无 acceptance", func(c *TaskContract) { c.Acceptance = nil }, "acceptance"},
		{"strict 无 acceptance", func(c *TaskContract) { c.Profile = ProfileStrict; c.Acceptance = nil }, "acceptance"},
		{"条款 id 重复", func(c *TaskContract) { c.Acceptance[2].ID = "AC-01" }, "重复"},
		{"条款缺 statement", func(c *TaskContract) { c.Acceptance[0].Statement = "" }, "statement"},
		{"列表空白条目", func(c *TaskContract) { c.DependsOn = []string{" "}; c.Acceptance = nil; c.Profile = ProfileLight }, "空白条目"},
	}
	for _, tt := range tests {
		c := mustParse(t, prdExample) // 每例全新解析,避免用例间共享底层切片互污染
		tt.mutate(&c)
		err := c.Validate()
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Errorf("%s: got %v, want 含 %q", tt.name, err, tt.wantErr)
		}
	}

	// light 可省略 acceptance(PRD §7.2)。
	light := mustParse(t, minimalExample)
	light.Profile = ProfileLight
	light.Acceptance = nil
	if err := light.Validate(); err != nil {
		t.Fatalf("light 无 acceptance 应通过: %v", err)
	}
	// 校验后改动需重新 Validate —— 与 deploy.Definition 同一使用方式。
	light.Profile = "bogus"
	if err := light.Validate(); err == nil {
		t.Fatal("改动后 Validate 应重新生效")
	}
}

// 排版不是内容:块标量尾随换行、条目首尾空白在解析时规范化掉。
func TestParseNormalizesWhitespace(t *testing.T) {
	padded := `id: "TASK-023 "
profile: standard
objective: "Prevent jobs from remaining permanently in running state after worker crashes.   "
scope:
  allowed_paths:
    - " internal/jobs/** "
acceptance:
  - id: " AC-01 "
    statement: "stale leases can be reclaimed"
    independence_required: source
`
	c := mustParse(t, padded)
	if c.ID != "TASK-023" || c.Objective != "Prevent jobs from remaining permanently in running state after worker crashes." {
		t.Fatalf("字段未去首尾空白: id=%q objective=%q", c.ID, c.Objective)
	}
	if c.Scope.AllowedPaths[0] != "internal/jobs/**" || c.Acceptance[0].ID != "AC-01" {
		t.Fatalf("条目未去首尾空白: %v %q", c.Scope.AllowedPaths, c.Acceptance[0].ID)
	}
	// 规范化后与无排版噪声的同一内容同哈希。
	a := mustParse(t, padded)
	b := mustParse(t, strings.NewReplacer(
		`id: "TASK-023 "`, `id: TASK-023`,
		`objective: "Prevent jobs from remaining permanently in running state after worker crashes.   "`,
		`objective: Prevent jobs from remaining permanently in running state after worker crashes.`,
		`- " internal/jobs/** "`, `- internal/jobs/**`,
		`id: " AC-01 "`, `id: AC-01`,
	).Replace(padded))
	if a.Hash != b.Hash {
		t.Fatalf("仅排版差异的合同哈希不同: %s vs %s", a.Hash, b.Hash)
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "TASK-023.yaml")
	if err := os.WriteFile(p, []byte(prdExample), 0o644); err != nil {
		t.Fatalf("写临时文件: %v", err)
	}
	c, err := ParseFile(p)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if c.ID != "TASK-023" || c.Hash == "" {
		t.Fatalf("ParseFile 结果异常: %+v", c)
	}
	if _, err := ParseFile(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("读不到的文件应报错")
	}
}
