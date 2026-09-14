// conflict.go 拥有候选策略集的冲突分析(P3-05):在既有 trigger/requires/
// gate_condition/enforcement 封闭词汇内,检测需要人工关注的交互 —— 绝不发明
// 通用布尔规则引擎,绝不自动消解(要求 1/2:保守告警优先于不可靠的自动裁决)。
//
// 词汇边界(改动前必读):
//   - EngineeringPolicy 的门禁语义是合取:一份变更命中多份 gate 策略时,
//     所有 all_of 键都必须满足 —— 叠加本身不是错误,但意图需要人确认;
//   - 词汇中没有否定(none_of/except),真正的逻辑矛盾不可表达;本分析把
//     「需要人裁决的不一致」如实分三级上报:conflict(结构性冲突,如身份
//     重复)/ warning(意图不一致或完全遮蔽,如同语义不同 enforcement)/
//     info(叠加信息);
//   - 输出确定性:同一输入集恒产出同一发现序列(按 kind 次序 + 策略 ID 排序);
//     集合级唯一性检查不因输入顺序不同而漂移;
//   - 触发轴重叠判定 = 任一轴(RiskSignals/Paths/RequirementRefs)有交集,
//     或双方 Trigger 均为零值(无条件触发 = 匹配一切,重叠按定义成立)。
package policygovernance

import (
	"fmt"
	"sort"
	"strings"

	"tally/internal/engineeringpolicy"
)

// ConflictSeverity 是发现的严重度(封闭词表):conflict 需要人裁决后才能
// 继续治理链;warning 需要人知晓;info 是确定性事实。
type ConflictSeverity string

const (
	SeverityConflict ConflictSeverity = "conflict"
	SeverityWarning  ConflictSeverity = "warning"
	SeverityInfo     ConflictSeverity = "info"
)

// ConflictKind 是发现类型的封闭词表(词汇内可确定性判定的全部交互形态)。
type ConflictKind string

const (
	// ConflictDuplicateID:同一策略集内出现相同 id —— 身份冲突(内容可能
	// 不同,身份必须唯一;P1-03 集合加载方同判,这里在治理面先行可见)。
	ConflictDuplicateID ConflictKind = "duplicate_id"
	// ConflictShadowedDuplicate:不同 id 但 trigger/requires/gate_condition/
	// enforcement 全同 —— 完全遮蔽的重复(意图不明:第二条是笔误还是备份)。
	ConflictShadowedDuplicate ConflictKind = "shadowed_duplicate"
	// ConflictEnforcementMismatch:触发与门禁键全同但 enforcement 不同
	// (gate vs guidance)—— 同一语义两处声明强度不一致,意图需要人确认。
	ConflictEnforcementMismatch ConflictKind = "enforcement_mismatch"
	// ConflictStackedGates:触发轴重叠的多份 gate 策略 —— 合取叠加是合法
	// 语义,但要求人知晓叠加后的全部键(info;不自动裁决谁更优)。
	ConflictStackedGates ConflictKind = "stacked_gates"
)

// Conflict 是一条需要人工关注的发现。只读;Detail 是确定性说明。
type Conflict struct {
	Severity  ConflictSeverity
	Kind      ConflictKind
	PolicyIDs []string
	// GateKeys 是涉及方共享/叠加的 requires 键(仅门禁键类发现携带)。
	GateKeys []string
	Detail   string
}

// rank 用于确定性输出排序(kind 词表次序,再按严重度、ID 列表)。
var conflictKindRank = map[ConflictKind]int{
	ConflictDuplicateID:         0,
	ConflictShadowedDuplicate:   1,
	ConflictEnforcementMismatch: 2,
	ConflictStackedGates:        3,
}

// AnalyzeConflicts 对候选策略集做确定性冲突分析。输入顺序无关:输出按
// kind 词表次序 + 涉及策略 ID 升序稳定排序。零发现 = 零长度切片(不是 nil
// 语义差异,调用方按 len 判读)。
func AnalyzeConflicts(policies []engineeringpolicy.Policy) []Conflict {
	var out []Conflict

	// 按 id 分组(身份冲突 + 同 id 内容不一致检测的基础)。
	byID := map[string][]engineeringpolicy.Policy{}
	for _, p := range policies {
		byID[p.ID] = append(byID[p.ID], p)
	}

	// 1. duplicate_id(conflict):同 id 多份 —— 携带内容修订让分歧可追溯。
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		group := byID[id]
		if len(group) < 2 {
			continue
		}
		revs := make([]string, 0, len(group))
		for _, p := range group {
			revs = append(revs, p.Revision)
		}
		sort.Strings(revs)
		out = append(out, Conflict{
			Severity:  SeverityConflict,
			Kind:      ConflictDuplicateID,
			PolicyIDs: []string{id},
			Detail: fmt.Sprintf("策略 id %q 在集合中出现 %d 次(内容修订:%s);身份必须唯一,先解决身份冲突再进治理链。",
				id, len(group), strings.Join(revs, ", ")),
		})
	}

	// 2. 两两比较(同 id 组内跳过 —— 已上报身份冲突;跨 id 才有意义;同 id
	//    多份时取修订序第一份进入比较,避免身份冲突的级联噪声)。
	uniq := make([]engineeringpolicy.Policy, 0, len(ids))
	for _, id := range ids {
		uniq = append(uniq, byID[id][0])
	}
	for i := 0; i < len(uniq); i++ {
		for j := i + 1; j < len(uniq); j++ {
			a, b := uniq[i], uniq[j]
			aa, bb := a, b
			if aa.ID > bb.ID {
				aa, bb = bb, aa
			}
			ids2 := []string{aa.ID, bb.ID}

			switch {
			case sameSemantics(a, b):
				// 3. shadowed_duplicate(warning):全同语义、不同身份。
				out = append(out, Conflict{
					Severity:  SeverityWarning,
					Kind:      ConflictShadowedDuplicate,
					PolicyIDs: ids2,
					GateKeys:  sortedCopy(a.Requires),
					Detail: fmt.Sprintf("策略 %q 与 %q 的 trigger/requires/gate_condition/enforcement 完全相同:后者完全遮蔽,意图需要人确认(备份/笔误/有意冗余)。",
						aa.ID, bb.ID),
				})
			case triggersOverlap(a.Trigger, b.Trigger) && sameSet(a.Requires, b.Requires) && a.Enforcement != b.Enforcement:
				// 4. enforcement_mismatch(warning):同一触发面 + 同一要求键集,
				//    声明强度不同 —— gate 拿阻断权,guidance 只提示;意图需人确认。
				//    (键比较只用 requires:guidance 按词汇表不携带 gate_condition,
				//    用 AllOf 比较会让 gate/guidance 对永不命中。)
				out = append(out, Conflict{
					Severity:  SeverityWarning,
					Kind:      ConflictEnforcementMismatch,
					PolicyIDs: ids2,
					GateKeys:  sortedCopy(a.Requires),
					Detail: fmt.Sprintf("策略 %q(%s)与 %q(%s)触发面重叠且门禁键全同,但声明强度不同;运行时合取下 gate 才有阻断权,强度差异是否有意需要人确认。",
						aa.ID, aa.Enforcement, bb.ID, bb.Enforcement),
				})
			default:
				// 5. stacked_gates(info):触发轴重叠的双 gate —— 合取叠加合法,
				//    人需知晓叠加后的全部键(并集,去重升序)。
				if triggersOverlap(a.Trigger, b.Trigger) &&
					a.Enforcement == engineeringpolicy.EnforcementGate && b.Enforcement == engineeringpolicy.EnforcementGate {
					keys := unionSorted(a.Requires, b.Requires)
					out = append(out, Conflict{
						Severity:  SeverityInfo,
						Kind:      ConflictStackedGates,
						PolicyIDs: ids2,
						GateKeys:  keys,
						Detail: fmt.Sprintf("策略 %q 与 %q 触发轴重叠且均为 gate:合取叠加后共同要求键 [%s];这是合法语义,请人确认叠加意图。",
							aa.ID, bb.ID, strings.Join(keys, ", ")),
					})
				}
			}
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := conflictKindRank[out[i].Kind], conflictKindRank[out[j].Kind]
		if ri != rj {
			return ri < rj
		}
		return strings.Join(out[i].PolicyIDs, "\x00") < strings.Join(out[j].PolicyIDs, "\x00")
	})
	return out
}

// sameSemantics 报告两份策略的治理语义是否完全相同(触发轴、要求键、门禁
// 条件、强度;不含 Rationale/BundleTags/Revision —— 那些不是门禁语义)。
func sameSemantics(a, b engineeringpolicy.Policy) bool {
	if a.Enforcement != b.Enforcement {
		return false
	}
	if !sameSet(a.Trigger.RiskSignals, b.Trigger.RiskSignals) ||
		!sameSet(a.Trigger.Paths, b.Trigger.Paths) ||
		!sameSet(a.Trigger.RequirementRefs, b.Trigger.RequirementRefs) {
		return false
	}
	return sameSet(a.Requires, b.Requires) && sameSet(a.GateCondition.AllOf, b.GateCondition.AllOf)
}

// triggersOverlap 报告两个触发轴是否可能命中同一变更:任一轴有交集,或双方
// 均为无条件触发(零值 Trigger 匹配一切)。
func triggersOverlap(a, b engineeringpolicy.Trigger) bool {
	if a.IsZero() && b.IsZero() {
		return true
	}
	if axisOverlaps(a.RiskSignals, b.RiskSignals) || axisOverlaps(a.Paths, b.Paths) || axisOverlaps(a.RequirementRefs, b.RequirementRefs) {
		return true
	}
	// 零值轴匹配一切:一方在该轴无条件即视为重叠候选(保守方向:宁多报给人看)。
	if a.IsZero() || b.IsZero() {
		return true
	}
	return false
}

func axisOverlaps(x, y []string) bool {
	if len(x) == 0 || len(y) == 0 {
		return false
	}
	set := map[string]bool{}
	for _, v := range x {
		set[v] = true
	}
	for _, v := range y {
		if set[v] {
			return true
		}
	}
	return false
}

func sameSet(x, y []string) bool {
	return strings.Join(sortedCopy(x), "\x00") == strings.Join(sortedCopy(y), "\x00")
}

func sortedCopy(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	return out
}

func unionSorted(a, b []string) []string {
	set := map[string]bool{}
	for _, v := range append(append([]string(nil), a...), b...) {
		set[v] = true
	}
	return sortedCopy(keysOf(set))
}

func keysOf(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
