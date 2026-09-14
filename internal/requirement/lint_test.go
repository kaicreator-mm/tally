package requirement

import (
	"strings"
	"testing"
)

// cleanDoc 返回一份过五闸的干净文档,各负例在它上面做最小改动。
func cleanDoc() *Document {
	return &Document{
		Title: "草稿管理",
		Clauses: []Clause{
			{ID: "REQ-1", Title: "保存", Statement: "WHEN 用户点击保存, 系统 shall 将草稿写入 git",
				Mechanism: MechAutoTest, Criteria: []string{"保存后仓库出现新提交"}},
		},
		NonGoals: []string{"不做云端同步"},
		Glossary: []Term{{Name: "草稿", Definition: "未签署的需求文本", Aliases: []string{"草稿"}}},
	}
}

// findIssue 找 level 级且消息含 substr 的问题。
func findIssue(issues []Issue, level, substr string) bool {
	for _, i := range issues {
		if i.Level == level && (substr == "" || strings.Contains(i.Message, substr)) {
			return true
		}
	}
	return false
}

// 干净文档应零问题。
func TestLintClean(t *testing.T) {
	if issues := Lint(cleanDoc()); len(issues) != 0 {
		t.Errorf("干净文档不应有问题, got %v", issues)
	}
}

// L1 条款化:正/负。
func TestLintL1(t *testing.T) {
	// 正:干净文档无 L1
	if findIssue(Lint(cleanDoc()), "L1", "") {
		t.Error("干净文档不应有 L1 问题")
	}
	// 负:编号重复
	d := cleanDoc()
	d.Clauses = append(d.Clauses, Clause{ID: "REQ-1", Statement: "系统 shall B", Mechanism: MechMetric, Criteria: []string{"y"}})
	if !findIssue(Lint(d), "L1", "重复") {
		t.Error("重复编号应报 L1")
	}
	// 负:不是单一 EARS(两个 shall)
	d2 := cleanDoc()
	d2.Clauses[0].Statement = "系统 shall A; 系统 shall B"
	if !findIssue(Lint(d2), "L1", "EARS") {
		t.Error("非单一 EARS 语句应报 L1")
	}
}

// L2 验收条件 + 检查机制:正/负。
func TestLintL2(t *testing.T) {
	if findIssue(Lint(cleanDoc()), "L2", "") {
		t.Error("干净文档不应有 L2 问题")
	}
	// 负:缺验收条件
	d := cleanDoc()
	d.Clauses[0].Criteria = nil
	if !findIssue(Lint(d), "L2", "验收条件") {
		t.Error("缺验收条件应报 L2")
	}
	// 负:未指明检查机制
	d2 := cleanDoc()
	d2.Clauses[0].Mechanism = ""
	if !findIssue(Lint(d2), "L2", "检查机制") {
		t.Error("缺检查机制应报 L2")
	}
	// 负:检查机制非法
	d3 := cleanDoc()
	d3.Clauses[0].Mechanism = "感觉"
	if !findIssue(Lint(d3), "L2", "不合法") {
		t.Error("非法检查机制应报 L2")
	}
}

// L3 零未决:正/负。
func TestLintL3(t *testing.T) {
	if findIssue(Lint(cleanDoc()), "L3", "") {
		t.Error("干净文档不应有 L3 问题")
	}
	// 负:TBD
	d := cleanDoc()
	d.Clauses[0].Statement = "系统 shall 支持 TBD 的协议"
	if !findIssue(Lint(d), "L3", "TBD") {
		t.Error("TBD 应报 L3")
	}
	// 负:待定
	d2 := cleanDoc()
	d2.Clauses[0].Criteria = []string{"超时时间待定"}
	if !findIssue(Lint(d2), "L3", "待定") {
		t.Error("待定应报 L3")
	}
	// 负:未选的二选一
	d3 := cleanDoc()
	d3.Clauses[0].Statement = "系统 shall 决定用 Postgres 还是 MySQL?"
	if !findIssue(Lint(d3), "L3", "二选一") {
		t.Error("未选的二选一应报 L3")
	}
}

// L4 非目标非空:正/负。
func TestLintL4(t *testing.T) {
	if findIssue(Lint(cleanDoc()), "L4", "") {
		t.Error("干净文档不应有 L4 问题")
	}
	d := cleanDoc()
	d.NonGoals = nil
	if !findIssue(Lint(d), "L4", "非目标") {
		t.Error("缺非目标应报 L4")
	}
}

// L5 术语唯一 + 术语表:正/负。
func TestLintL5(t *testing.T) {
	if findIssue(Lint(cleanDoc()), "L5", "") {
		t.Error("干净文档不应有 L5 问题")
	}
	// 负:缺术语表
	d := cleanDoc()
	d.Glossary = nil
	if !findIssue(Lint(d), "L5", "术语表") {
		t.Error("缺术语表应报 L5")
	}
	// 负:一词多义
	d2 := cleanDoc()
	d2.Glossary = append(d2.Glossary, Term{Name: "用户", Definition: "使用系统的人", Aliases: []string{"用户", "客户"}})
	if !findIssue(Lint(d2), "L5", "一词多义") {
		t.Error("一词多义应报 L5")
	}
}

// 输出是结构化清单(带条款定位),不是评分;且能端到端从 markdown 解析后检查。
func TestLintStructuredOutputFromMarkdown(t *testing.T) {
	src := `# x

## REQ-3

系统 shall 做点什么

## 非目标

- 不做 A

## 术语表

- 术语: 定义
`
	doc := Parse(src)
	issues := Lint(doc)
	if !findIssue(issues, "L2", "验收条件") {
		t.Errorf("REQ-3 缺验收条件应被结构化报出, got %v", issues)
	}
	// 问题应带上条款编号
	var found bool
	for _, i := range issues {
		if i.Clause == "REQ-3" {
			found = true
		}
	}
	if !found {
		t.Errorf("问题应定位到 REQ-3, got %v", issues)
	}
}
