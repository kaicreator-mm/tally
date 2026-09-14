package engineeringpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Policy 的修订哈希(AC4):由**内容**推导,与 YAML 键序、列表顺序、折叠写法、
// 多余空白无关 —— 同一份策略内容无论怎么排版,哈希不变;任何实质内容变化都
// 改变哈希。因此 Revision 可以作为策略 revision 的稳定标识被下游(匹配 P1-05、
// 合同编译 P1-07、Git 制品加载 P1-03)引用,满足「Git owns versioned intent」
// 下内容寻址版本化制品的需要。

// canonicalVersion 是规范形式的版本前缀。规范格式将来若变化,递增它,
// 避免不同代格式的哈希空间重叠。
const canonicalVersion = "engineeringpolicy/v1"

// Canonical 返回策略的规范文本:所有集合类列表排序去重、文本折叠为单行空白。
// 这是 RevisionHash 的唯一输入。
func (p Policy) Canonical() string {
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
	emit("id", p.ID)
	emit("enforcement", string(p.Enforcement))
	emit("rationale", p.Rationale)
	emitLines(&b, "trigger.risk_signal", sortedUnique(p.Trigger.RiskSignals))
	emitLines(&b, "trigger.path", sortedUnique(p.Trigger.Paths))
	emitLines(&b, "trigger.requirement_ref", sortedUnique(p.Trigger.RequirementRefs))
	emitLines(&b, "requires", sortedUnique(p.Requires))
	emitLines(&b, "gate.all_of", sortedUnique(p.GateCondition.AllOf))
	emitLines(&b, "bundle_tag", sortedUnique(p.BundleTags))
	return b.String()
}

// RevisionHash 返回 "sha256:<hex>" 形式的内容修订哈希(AC4)。规范文本未变 ⇒
// 哈希不变;内容实质变化 ⇒ 哈希变化。哈希不代表「策略正确」,只代表
// 「这是哪一版内容」。
func (p Policy) RevisionHash() string {
	sum := sha256.Sum256([]byte(p.Canonical()))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// GateKeys 返回 gate_condition.all_of 的升序拷贝(门禁要求的全部键)。
// 供下游证据编译(P1-07)按需取用,不改原切片。
func (p Policy) GateKeys() []string {
	return sortedUnique(p.GateCondition.AllOf)
}

// emitLines 按 label 逐行写列表条目;空列表什么都不写(缺席与空等价)。
func emitLines(b *strings.Builder, label string, vals []string) {
	for _, v := range vals {
		fmt.Fprintf(b, "%s:%s\n", label, collapseSpace(v))
	}
}

// sortedUnique 返回排序去重后的拷贝:条目先经 collapseSpace 规范化,空白条目
// 丢弃。这样「内部多几个空格的同一个键」也是同一条内容。不改原切片。
func sortedUnique(vals []string) []string {
	out := make([]string, 0, len(vals))
	seen := map[string]bool{}
	for _, v := range vals {
		c := collapseSpace(v)
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// collapseSpace 把字符串内所有空白折叠为单个空格并去首尾。
// 目的:同一句话的多种等价排版(YAML 折叠、换行、多空格)产出同一规范文本。
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
