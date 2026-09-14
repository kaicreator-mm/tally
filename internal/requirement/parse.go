package requirement

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"tally/internal/core"
)

// 需求文件是 git 里的 markdown(.tally/requirements/*.md)。格式如下,解析器只认这套:
//
//	# <文档标题>                       (可选)
//
//	## REQ-1 <可选标题>                 条款,编号可独立引用(L1)
//
//	<EARS 语句,一段>                    采用 EARS,不自创语法
//
//	验收: 自动化测试                    检查机制:自动化测试 / 指标 / 人工检查(L2)
//	- 验收条件 1                       一或多条(L2)
//	- 验收条件 2
//
//	## 非目标                           必须非空(L4)
//	- 不做什么
//
//	## 术语表                           必须有(L5)
//	- 术语: 定义
//
// EARS(https://alistairmavin.com/ears/)五种形态,都落在一条语句里:
//
//	常态:     <系统> shall <响应>
//	事件:     WHEN <触发>, <系统> shall <响应>
//	状态:     WHILE <状态>, <系统> shall <响应>
//	可选特性: WHERE <特性>, <系统> shall <响应>
//	异常:     IF <条件>, THEN <系统> shall <响应>
//
// 复合前置可叠加:WHILE <状态>, WHEN <触发>, <系统> shall <响应>。

// 检查机制的合法取值(L2)。
const (
	MechAutoTest = "自动化测试"
	MechMetric   = "指标"
	MechManual   = "人工检查"
)

// Document 是一份解析后的需求文档。
type Document struct {
	Title    string
	Clauses  []Clause
	NonGoals []string
	Glossary []Term
}

// Clause 是一条需求条款。
type Clause struct {
	ID        string   // 如 REQ-1
	Title     string   // 可选标题
	Statement string   // EARS 语句原文
	Mechanism string   // 检查机制
	Criteria  []string // 验收条件
}

// Term 是术语表里的一项。
type Term struct {
	Name       string   // 术语(规范名)
	Definition string   // 定义
	Aliases    []string // 若同一概念给了多个名字,全部列出(L5 用)
}

// EARSKind 是 EARS 子句的类型。
type EARSKind string

const (
	KindWhen  EARSKind = "WHEN"  // 事件触发
	KindIf    EARSKind = "IF"    // 异常触发
	KindWhile EARSKind = "WHILE" // 状态前置
	KindWhere EARSKind = "WHERE" // 可选特性前置
)

// Precondition 是一个前置条件(WHILE 状态 / WHERE 特性)。
type Precondition struct {
	Kind EARSKind
	Text string
}

// Trigger 是触发(至多一个:WHEN 事件 / IF 异常)。
type Trigger struct {
	Kind EARSKind
	Text string
}

// EARSStatement 是一条解析后的 EARS 语句。
type EARSStatement struct {
	Preconditions []Precondition // 零或多
	Trigger       *Trigger       // 零或一
	System        string         // 恰好一个系统名
	Responses     []string       // 一或多
}

var (
	reClauseHeading = regexp.MustCompile(`^##\s+(REQ-[^\s]+)\s*(.*)$`)
	reNonGoals      = regexp.MustCompile(`^##\s*(非目标|Non-?goals?)\s*$`)
	reGlossary      = regexp.MustCompile(`^##\s*(术语表|Glossary)\s*$`)
	reMechanism     = regexp.MustCompile(`^验收\s*[:：]\s*(.+?)\s*$`)
	reBullet        = regexp.MustCompile(`^[-*]\s+(.+?)\s*$`)
	reTerm          = regexp.MustCompile(`^(.+?)\s*[:：]\s*(.*)$`)
	reHeading       = regexp.MustCompile(`^#{1,6}\s`)
	// EARS 关键字 shall,恰好出现一次 = 恰好一个系统-响应对。
	reShall = regexp.MustCompile(`(?i)\bshall\b`)
	// 术语别名分隔:用户/客户、用户、客户。
	reAliasSep = regexp.MustCompile(`[/、,，]`)
)

// ParseFile 读取并解析一个需求 markdown 文件。
func ParseFile(path string) (*Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读需求文件 %s: %w", path, err)
	}
	return Parse(string(b)), nil
}

// Parse 解析需求 markdown 文本。解析是宽容的:格式问题不在这里报错,
// 而是留给 Lint 以结构化问题清单报出(解析错误 = 结构缺口的一种)。
func Parse(src string) *Document {
	doc := &Document{}
	section := "" // 当前节:"" / "nongoals" / "glossary" / 条款 id
	var clause *Clause
	var stmtLines []string
	stmtClosed := false

	flushClause := func() {
		if clause != nil {
			clause.Statement = strings.TrimSpace(strings.Join(stmtLines, " "))
			doc.Clauses = append(doc.Clauses, *clause)
			clause = nil
		}
		stmtLines = nil
		stmtClosed = false
	}

	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)

		if m := reClauseHeading.FindStringSubmatch(trimmed); m != nil {
			flushClause()
			section = m[1]
			clause = &Clause{ID: m[1], Title: strings.TrimSpace(m[2])}
			continue
		}
		if reNonGoals.MatchString(trimmed) {
			flushClause()
			section = "nongoals"
			continue
		}
		if reGlossary.MatchString(trimmed) {
			flushClause()
			section = "glossary"
			continue
		}
		// 其它标题:结束当前条款,视为无关节。
		if reHeading.MatchString(trimmed) {
			flushClause()
			if section != "nongoals" && section != "glossary" {
				section = ""
			}
			if strings.HasPrefix(trimmed, "# ") && doc.Title == "" {
				doc.Title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			}
			continue
		}

		switch section {
		case "nongoals":
			if m := reBullet.FindStringSubmatch(trimmed); m != nil {
				doc.NonGoals = append(doc.NonGoals, m[1])
			}
		case "glossary":
			if m := reBullet.FindStringSubmatch(trimmed); m != nil {
				doc.Glossary = append(doc.Glossary, parseTerm(m[1]))
			}
		default:
			if clause == nil {
				// 条款之外的无归属正文(如 H1 之后、第一个条款之前的引言),忽略。
				continue
			}
			switch {
			case reMechanism.MatchString(trimmed):
				clause.Mechanism = strings.TrimSpace(reMechanism.FindStringSubmatch(trimmed)[1])
				stmtClosed = true
			case reBullet.MatchString(trimmed):
				clause.Criteria = append(clause.Criteria, reBullet.FindStringSubmatch(trimmed)[1])
				stmtClosed = true
			case trimmed != "" && !stmtClosed:
				stmtLines = append(stmtLines, trimmed)
			}
		}
	}
	flushClause()
	return doc
}

// parseTerm 解析一个术语条目 "名称: 定义";名称里含别名分隔符时拆出 Aliases。
func parseTerm(s string) Term {
	t := Term{}
	if m := reTerm.FindStringSubmatch(s); m != nil {
		t.Name = strings.TrimSpace(m[1])
		t.Definition = strings.TrimSpace(m[2])
	} else {
		t.Name = strings.TrimSpace(s)
	}
	parts := reAliasSep.Split(t.Name, -1)
	var aliases []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			aliases = append(aliases, p)
		}
	}
	t.Aliases = aliases
	if len(aliases) > 0 {
		t.Name = aliases[0] // 规范名取第一个
	}
	return t
}

// ParseEARS 解析一条 EARS 语句,并做 ruleset 校验:
// 零或多前置(WHILE/WHERE)、零或一触发(WHEN/IF)、恰好一个系统名、一或多响应。
func ParseEARS(stmt string) (EARSStatement, error) {
	var out EARSStatement
	loc := reShall.FindAllStringIndex(stmt, -1)
	switch {
	case len(loc) == 0:
		return out, fmt.Errorf("缺少关键字 shall:不是合法的 EARS 语句")
	case len(loc) > 1:
		return out, fmt.Errorf("出现 %d 个 shall:一条条款只能有一个系统-响应对(恰好一个系统名)", len(loc))
	}

	head := stmt[:loc[0][0]]
	tail := stmt[loc[0][1]:]

	// 响应:以 ; / ; 分隔,一或多。
	for _, r := range strings.FieldsFunc(tail, func(r rune) bool { return r == ';' || r == '；' }) {
		if r = strings.TrimSpace(r); r != "" {
			out.Responses = append(out.Responses, strings.TrimRight(r, "。."))
		}
	}
	if len(out.Responses) == 0 {
		return out, fmt.Errorf("缺少响应:shall 之后必须有一或多条响应")
	}

	// 头部按 , / , 切成子句,按关键字归类。
	var systemParts []string
	sawThen := false
	for _, part := range strings.FieldsFunc(head, func(r rune) bool { return r == ',' || r == '，' }) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kind, rest := splitKeyword(part)
		switch kind {
		case KindWhen, KindIf:
			if out.Trigger != nil {
				return out, fmt.Errorf("多个触发(%s 与 %s):至多一个触发", out.Trigger.Kind, kind)
			}
			out.Trigger = &Trigger{Kind: kind, Text: rest}
		case KindWhile, KindWhere:
			out.Preconditions = append(out.Preconditions, Precondition{Kind: kind, Text: rest})
		case "THEN":
			sawThen = true
			if rest != "" {
				systemParts = append(systemParts, rest)
			}
		default:
			systemParts = append(systemParts, part)
		}
	}
	if sawThen && out.Trigger != nil && out.Trigger.Kind != KindIf {
		return out, fmt.Errorf("THEN 只能跟在 IF 之后")
	}

	out.System = cleanSystemName(strings.Join(systemParts, " "))
	if out.System == "" {
		return out, fmt.Errorf("缺少系统名:shall 之前必须指明恰好一个系统")
	}
	return out, nil
}

// splitKeyword 识别子句开头的 EARS 关键字,返回 (关键字, 剩余文本)。
// 非关键字子句返回 ("", 原文)。
func splitKeyword(part string) (EARSKind, string) {
	up := strings.ToUpper(part)
	for _, kw := range []EARSKind{KindWhen, KindWhile, KindWhere, KindIf} {
		if strings.HasPrefix(up, string(kw)+" ") {
			return kw, strings.TrimSpace(part[len(kw):])
		}
	}
	if strings.HasPrefix(up, "THEN ") {
		return "THEN", strings.TrimSpace(part[len("THEN"):])
	}
	if up == "THEN" {
		return "THEN", ""
	}
	return "", part
}

// cleanSystemName 去掉英文 EARS 里系统名前的冠词 "the"/"The"。
func cleanSystemName(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"the ", "The ", "THE "} {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimSpace(s[len(prefix):])
		}
	}
	return s
}

// ContentHash 计算一条条款内容的内容哈希(INV-4)。
//
// 挂钩记的是这个哈希,不是逻辑编号:条款内容一变,哈希变,旧挂钩自动对不上 →
// 下游显示"依据已变更",失效传播不需要额外机制。
//
// 规范化(折叠空白、去首尾空白)让纯排版差异不改变哈希,但真实内容变化一定改变。
// 权威内容在 git;此哈希用于追溯边指向具体某一版。
func ContentHash(c Clause) core.ClauseHash {
	var b strings.Builder
	b.WriteString(normalize(c.ID))
	b.WriteByte('\n')
	b.WriteString(normalize(c.Statement))
	b.WriteByte('\n')
	b.WriteString(normalize(c.Mechanism))
	b.WriteByte('\n')
	for _, cr := range c.Criteria {
		b.WriteString(normalize(cr))
		b.WriteByte('\n')
	}
	return hashString(b.String())
}

// normalize 折叠连续空白为单个空格并去首尾空白,使排版差异不影响内容哈希。
func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
