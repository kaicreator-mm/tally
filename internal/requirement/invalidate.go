package requirement

import (
	"tally/internal/core"
)

// 条款改了,下游要知道(PRD 第六部分规则 ⑥,INV-4)。
//
// 做法就是"没有额外机制":挂钩记的是内容哈希,不是逻辑编号。条款内容一变,
// 哈希变,旧挂钩自然对不上当前版本 → 下游显示"依据已变更"。失效是哈希比对
// 的自然结果,不需要一张失效表、不需要传播逻辑。

// RefStatus 是一条下游引用相对条款当前版本的状态。
type RefStatus int

const (
	// RefValid:引用的哈希就是当前版,依据有效。
	RefValid RefStatus = iota
	// RefChanged:引用的是旧哈希,"依据已变更"。
	RefChanged
)

// String 返回状态的展示文案。
func (s RefStatus) String() string {
	if s == RefChanged {
		return "依据已变更"
	}
	return "依据有效"
}

// CheckReference 比较下游引用的哈希与条款当前哈希。
// 对不上 = 依据已变更。就这么简单 —— 这就是 INV-4 要的全部机制。
func CheckReference(referenced, current core.ClauseHash) RefStatus {
	if referenced == current {
		return RefValid
	}
	return RefChanged
}

// CurrentHashes 返回文档里每条条款当前的内容哈希(clause_id → hash)。
func CurrentHashes(doc *Document) map[string]core.ClauseHash {
	m := make(map[string]core.ClauseHash, len(doc.Clauses))
	for _, c := range doc.Clauses {
		m[c.ID] = ContentHash(c)
	}
	return m
}

// Reference 是一条下游记录对条款的引用:它挂的是某个历史版本的哈希。
type Reference struct {
	ClauseID string
	Hash     core.ClauseHash
}

// ReferenceReport 是一条引用对照当前版本的结果。
type ReferenceReport struct {
	Reference Reference
	Status    RefStatus
}

// ClassifyReferences 把一批下游引用对照文档当前版本,逐条标出是否"依据已变更"。
// 引用指向的条款已不在文档里(被删)时,同样视为依据已变更。
func ClassifyReferences(refs []Reference, doc *Document) []ReferenceReport {
	current := CurrentHashes(doc)
	reports := make([]ReferenceReport, 0, len(refs))
	for _, r := range refs {
		status := RefChanged
		if cur, ok := current[r.ClauseID]; ok {
			status = CheckReference(r.Hash, cur)
		}
		reports = append(reports, ReferenceReport{Reference: r, Status: status})
	}
	return reports
}
