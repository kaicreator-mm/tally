package requirement

import (
	"fmt"
	"regexp"
	"strings"
)

// Lint 跑五闸,返回结构化问题清单:第几条、缺什么。**不是评分**(PRD 第六部分规则 ③)。
//
// 五闸全部机械判断,不评价"写得够不够好"—— 那是让 LLM 当裁判,它当不了。
//
//	L1 条款化      每条可独立编号引用(编号唯一、是单一 EARS 语句)
//	L2 验收条件    每条都有验收条件,且指明检查机制
//	L3 零未决      无 TBD、待定、未选的二选一
//	L4 非目标非空  必须显式写"不做什么"
//	L5 术语唯一    同一概念一个词,附术语表
func Lint(doc *Document) []Issue {
	var issues []Issue
	issues = append(issues, lintClauses(doc)...)
	issues = append(issues, lintNonGoals(doc)...)
	issues = append(issues, lintGlossary(doc)...)
	return issues
}

// Issue 是一个结构化问题。
type Issue struct {
	Level   string // L1..L5
	Clause  string // 条款编号;文档级问题为空
	Message string // 缺什么
}

func (i Issue) String() string {
	if i.Clause == "" {
		return fmt.Sprintf("[%s] %s", i.Level, i.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", i.Level, i.Clause, i.Message)
}

// HasStructuralGap 报告是否还有结构缺口。结构缺口是硬闸:不清零不许签。
// Lint 报出的每一条都是结构缺口(linter 管的是"该有的结构有没有")。
func HasStructuralGap(issues []Issue) bool { return len(issues) > 0 }

// lintClauses 跑 L1(条款化)、L2(验收条件)、L3(零未决)。
func lintClauses(doc *Document) []Issue {
	var issues []Issue
	if len(doc.Clauses) == 0 {
		issues = append(issues, Issue{Level: "L1", Message: "没有任何条款"})
		return issues
	}

	seen := map[string]bool{}
	for _, c := range doc.Clauses {
		// L1:编号唯一
		if seen[c.ID] {
			issues = append(issues, Issue{Level: "L1", Clause: c.ID, Message: "条款编号重复,无法独立引用"})
		}
		seen[c.ID] = true

		// L1:必须能解析为单一 EARS 语句(ruleset 校验在此发生)
		if _, err := ParseEARS(c.Statement); err != nil {
			issues = append(issues, Issue{Level: "L1", Clause: c.ID, Message: "不是单一合法的 EARS 语句: " + err.Error()})
		}

		// L2:验收条件 + 检查机制
		if len(c.Criteria) == 0 {
			issues = append(issues, Issue{Level: "L2", Clause: c.ID, Message: "缺少验收条件"})
		}
		switch {
		case c.Mechanism == "":
			issues = append(issues, Issue{Level: "L2", Clause: c.ID, Message: "未指明检查机制"})
		case !validMechanism(c.Mechanism):
			issues = append(issues, Issue{Level: "L2", Clause: c.ID,
				Message: fmt.Sprintf("检查机制 %q 不合法,应为 %s/%s/%s", c.Mechanism, MechAutoTest, MechMetric, MechManual)})
		}

		// L3:零未决
		issues = append(issues, lintOpenEnded(c)...)
	}
	return issues
}

func validMechanism(m string) bool {
	return m == MechAutoTest || m == MechMetric || m == MechManual
}

// 未决标记:L3 要清掉的。
var openMarkers = []struct {
	re   *regexp.Regexp
	name string
}{
	{regexp.MustCompile(`(?i)\bTBD\b`), "TBD"},
	{regexp.MustCompile(`(?i)\bTODO\b`), "TODO"},
	{regexp.MustCompile(`待定`), "待定"},
	{regexp.MustCompile(`未定`), "未定"},
	{regexp.MustCompile(`二选一`), "二选一"},
}

// reEitherOr 抓"还没选"的二选一疑问:……还是……?。
var reEitherOr = regexp.MustCompile(`还是[^。;；!！]*[??]`)

// lintOpenEnded 扫一条条款的全部文本(标题+语句+验收条件)找未决标记。
func lintOpenEnded(c Clause) []Issue {
	text := strings.Join(append([]string{c.Title, c.Statement}, c.Criteria...), " ")
	var issues []Issue
	reported := map[string]bool{}
	for _, m := range openMarkers {
		if loc := m.re.FindStringIndex(text); loc != nil && !reported[m.name] {
			reported[m.name] = true
			issues = append(issues, Issue{Level: "L3", Clause: c.ID,
				Message: fmt.Sprintf("含未决标记 %q(%q)", m.name, text[loc[0]:loc[1]])})
		}
	}
	if loc := reEitherOr.FindStringIndex(text); loc != nil {
		issues = append(issues, Issue{Level: "L3", Clause: c.ID,
			Message: fmt.Sprintf("有未选的二选一(%q)", text[loc[0]:loc[1]])})
	}
	return issues
}

// lintNonGoals 跑 L4:非目标必须显式写,且非空。
func lintNonGoals(doc *Document) []Issue {
	if len(doc.NonGoals) == 0 {
		return []Issue{{Level: "L4", Message: "缺少非目标:必须显式写“不做什么”"}}
	}
	return nil
}

// lintGlossary 跑 L5:必须有术语表;同一概念不得给多个名字(一词多义);术语不重复定义。
func lintGlossary(doc *Document) []Issue {
	var issues []Issue
	if len(doc.Glossary) == 0 {
		issues = append(issues, Issue{Level: "L5", Message: "缺少术语表"})
		return issues
	}
	seen := map[string]bool{}
	for _, t := range doc.Glossary {
		if len(t.Aliases) > 1 {
			issues = append(issues, Issue{Level: "L5",
				Message: fmt.Sprintf("术语 %q 一词多义:同一概念给了多个名字 %s", t.Name, strings.Join(t.Aliases, "/"))})
		}
		if seen[t.Name] {
			issues = append(issues, Issue{Level: "L5", Message: fmt.Sprintf("术语 %q 重复定义", t.Name)})
		}
		seen[t.Name] = true
	}
	return issues
}
