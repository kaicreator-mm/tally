package taskcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// TaskContract 的修订哈希(PRD §6.1:以 stable ref + hash/revision 为主):
// 由**内容**推导,与 YAML 的键序、列表顺序、折叠写法无关 —— 同一份合同内容
// 无论怎么排版、条目怎么换位,哈希不变;内容任何实质变化都改变哈希。
// 因此它可以作为 TaskContract revision 的稳定标识被下游(execution、
// verification、trace)引用,满足 INV-ART-01「Git owns versioned intent」下
// 「内容寻址的版本化制品」需要。

// canonicalVersion 是规范形式的版本前缀。规范格式将来若变化,递增它,
// 避免不同代格式的哈希空间重叠。
const canonicalVersion = "taskcontract/v1"

// Canonical 返回合同的规范文本:所有集合类列表排序去重、acceptance 按 id
// 排序、文本折叠为单行空白。这是 RevisionHash 的唯一输入。
func (c TaskContract) Canonical() string {
	var b strings.Builder
	b.WriteString(canonicalVersion)
	b.WriteByte('\n')
	emit := func(label, text string) {
		if t := collapseSpace(text); t != "" {
			b.WriteString(label)
			b.WriteByte(':')
			b.WriteString(t)
			b.WriteByte('\n')
		}
	}
	emit("id", c.ID)
	emit("profile", c.Profile)
	emit("title", c.Title)
	emit("objective", c.Objective)
	emitLines(&b, "ref", sortedUnique(c.ProductRefs, c.RequirementRefs, c.ArchitectureRefs))
	emitLines(&b, "depends_on", sortedUnique(c.DependsOn))
	emitLines(&b, "scope.allowed", sortedUnique(c.Scope.AllowedPaths))
	emitLines(&b, "scope.forbidden", sortedUnique(c.Scope.ForbiddenPaths))

	clauses := make([]AcceptanceClause, len(c.Acceptance))
	copy(clauses, c.Acceptance)
	sort.Slice(clauses, func(i, j int) bool { return clauses[i].ID < clauses[j].ID })
	for _, a := range clauses {
		fmt.Fprintf(&b, "acceptance:%s|%s|%s\n", collapseSpace(a.ID), a.Independence(), collapseSpace(a.Statement))
	}

	emitLines(&b, "documentation_impact", sortedUnique(c.DocumentationImpact))
	return b.String()
}

// RevisionHash 返回 "sha256:<hex>" 形式的内容修订哈希。规范文本未变 ⇒ 哈希不变;
// 内容实质变化 ⇒ 哈希变化。哈希不代表「合同正确」,只代表「这是哪一版内容」。
func (c TaskContract) RevisionHash() string {
	sum := sha256.Sum256([]byte(c.Canonical()))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// emitLines 按 label 逐行写列表条目;空列表什么都不写(缺席与空等价)。
func emitLines(b *strings.Builder, label string, vals []string) {
	for _, v := range vals {
		fmt.Fprintf(b, "%s:%s\n", label, collapseSpace(v))
	}
}

// sortedUnique 把多个列表合成一个排序去重后的拷贝:条目先经 collapseSpace
// 规范化,空白条目丢弃。这样「内部多几个空格的同一个路径」也是同一条内容。
// 不改原切片。
func sortedUnique(lists ...[]string) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range lists {
		for _, v := range l {
			c := collapseSpace(v)
			if c == "" || seen[c] {
				continue
			}
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

// collapseSpace 把字符串内所有空白折叠为单个空格并去首尾。
// 目的:同一句话的多种等价排版(YAML 折叠、换行、多空格)产出同一规范文本。
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
