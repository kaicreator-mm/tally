package approval

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"tally/internal/evidence"
)

// Independence Waiver 领域(PRD §8.10,P1-23)。
//
// 规则来源:
//   - §8.10:Waiver 只能覆盖**运行期** independence 损失;Freeze 时(§8.5.1)
//     已经能发现的 planned coverage 缺失不能豁免 —— 那是 FREEZE FAIL,契约
//     根本不会进入执行,不存在「运行期损失」可接受;
//   - §8.10:Project Verification Policy 决定 Waiver 是否可用,可按 risk
//     category 禁用;允许时必须产生带 provenance 的 HumanApprovalEvidence,
//     至少记录 compromised acceptances/bindings、exposure refs、coverage
//     gap、reason code 与 owner risk acceptance;
//   - §8.10:验证结论区分 strict_satisfied / strict_waived;strict_waived
//     不得在任何 UI/报表中表述为「Strict requirement 已满足」;
//   - §8.6.1/§8.8:exposure 是 observed fact,wavier 只是风险接受标注 ——
//     它不改变 internal/evidence 的任何独立性推导,compromised 证据不会被
//     重新标注为 independent;被 exposure 危害的旧 binding 仍按原样作为
//     functional regression gate 存在(§8.8.3)。
//
// 边界(P1):
//   - 本文件只做豁免 provenance 的构造校验与 strict 结论的机械判定;
//     independence coverage 的推导在 internal/verification(P1-20),调用方
//     把「未满足 strict 独立性要求的 acceptance 集合」作为输入传入;
//   - 本包没有 lifecycle 写入:ResolveStrictOutcome 只产出结论事实,状态
//     派生归 State Engine。

// 哨兵错误:调用方用 errors.Is 区分,包装文案只提供字段级上下文。
var (
	// ErrInvalidWaiver 拒绝 provenance 不完整的豁免(缺 category/gap/reason
	// code/owner 接受,或 gap 引用空 acceptance、列表含空项、重复 gap)。
	ErrInvalidWaiver = errors.New("approval: invalid independence waiver")
	// ErrPlannedCoverageNotWaivable 表示存在 design-time planned coverage
	// 缺失(§8.5.1 Freeze 时已可见):任何 waiver、任何 policy 都不能修复
	// 它(AC4)。这不是策略拒绝,而是「不可豁免」。
	ErrPlannedCoverageNotWaivable = errors.New("approval: planned coverage failure is not waivable")
	// ErrWaiverRequired 表示 strict 独立性要求未满足且没有记录任何豁免:
	// 结论不能是 strict_waived,strict 要求照旧不满足。
	ErrWaiverRequired = errors.New("approval: strict independence requirement unmet and no waiver recorded")
	// ErrWaiverNotPermitted 表示 Project Verification Policy 不允许本次豁免
	// (总开关关闭,或该 risk category 被禁用):strict_waived 不可达(AC3)。
	ErrWaiverNotPermitted = errors.New("approval: independence waiver not permitted by project verification policy")
	// ErrWaiverIncomplete 表示豁免未覆盖全部未满足的 acceptance:放行会静默
	// 丢掉未列出的独立性缺口,拒绝。
	ErrWaiverIncomplete = errors.New("approval: waiver does not cover every unmet acceptance")
	// ErrWaiverOverreach 表示豁免声称豁掉了并不存在的缺口(候选已 strict
	// 满足却携带豁免,或豁免列出的 gap 不在未满足集合内)。豁免 provenance
	// 必须如实:不存在豁免零缺口的候选(那应是不带豁免的 strict_satisfied 行)。
	ErrWaiverOverreach = errors.New("approval: waiver claims acceptance that is not unmet")
)

// StrictOutcome 是 §8.10 的 strict 验证结论。两个值在类型级互斥:
// strict_waived 行永远不得被并入 strict_satisfied 呈现(§8.10)。
type StrictOutcome string

const (
	// StrictSatisfied:strict 独立性要求由适用证据集直接满足,无豁免参与。
	StrictSatisfied StrictOutcome = "strict_satisfied"
	// StrictWaived:strict 独立性要求未由证据满足,但存在 policy 允许且
	// 完整覆盖缺口的显式豁免(带 provenance 的风险接受)。
	StrictWaived StrictOutcome = "strict_waived"
)

// RiskCategory 是豁免请求所处的风险类别(§8.10:policy 可按 risk category
// 禁用 Waiver)。集合开放:类别词表是项目 Verification Policy 的编写词汇,
// 本包不定义封闭枚举,未知类别按 policy 声明判定。
type RiskCategory string

// WaiverPolicy 是 Project Verification Policy 中决定 Waiver 可用性的输入
// (AC3):豁免是否可用完全由 policy 决定,本包不存在「默认允许」。
// 零值 WaiverPolicy 禁止一切豁免 —— 保守默认:没有显式声明的放行不存在。
type WaiverPolicy struct {
	// Allowed 是豁免总开关;false(含零值)时任何类别都不可豁免。
	Allowed bool
	// DisabledCategories 是总开关开启时仍被禁用的风险类别(§8.10:可按
	// risk category 禁用);重复项无害。
	DisabledCategories []RiskCategory
}

// Allows 报告 policy 是否允许对类别 c 的豁免:总开关开启且 c 未被按类别禁用。
func (p WaiverPolicy) Allows(c RiskCategory) bool {
	return p.Allowed && !slices.Contains(p.DisabledCategories, c)
}

// CoverageGap 是一条 strict 独立性要求未由适用证据集满足的 Acceptance Clause
// (P1-20 DeriveIndependenceCoverage 的 Covered=false 条目,或后续状态引擎
// 组合出的同义输入)。豁免按 acceptance 粒度声明覆盖范围。
type CoverageGap struct {
	// AcceptanceID 是未满足 strict 独立性要求的验收条款 id(必填)。
	AcceptanceID string
	// Planned 标记该缺口是 design-time planned coverage 缺失(§8.5.1:
	// Freeze 时已可见,如 required=information 却没有 held_out binding)。
	// planned=true 的缺口不可被任何豁免覆盖(AC4)。
	Planned bool
}

// WaiverSpec 是 NewWaiver 的构造输入;构造成功后与产物彻底脱钩。
type WaiverSpec struct {
	// Category 是豁免请求的风险类别(policy 按它判定可用性,必填)。
	Category RiskCategory
	// Gaps 是本次豁免覆盖的独立性缺口(§8.10 coverage gap;至少一条)。
	// 运行期损失(Planned=false)才可列入;planned 缺失直接拒绝。
	Gaps []CoverageGap
	// CompromisedAcceptances 是证据被 exposure 危害的 acceptance id 原样
	// 列表(§8.10 compromised acceptances;可选,可空)。它是「为什么存在
	// 缺口」的 provenance:与 Gaps 允许不同 —— required=none 的 acceptance
	// 被危害不构成独立性缺口,但仍如实记录。
	CompromisedAcceptances []string
	// CompromisedBindings 是被危害 evidence binding 的完整引用(§8.10
	// compromised bindings;可选)。复用 internal/evidence.BindingRef:
	// project + 冻结契约 revision 身份 + binding id,可精确回查 holdout
	// 消费与 exposure 记账。
	CompromisedBindings []evidence.BindingRef
	// ExposureRefs 是支撑本次风险接受的 exposure 事实 id(§8.10 exposure
	// refs;可选,可空 —— 缺口也可能来自没有可适用独立证据而非 exposure)。
	ExposureRefs []evidence.ExposureID
	// ReasonCode 是豁免原因代码(§8.10 必填;对应 human_approval 的
	// waiver_reason_code,CHECK 兜底非空)。
	ReasonCode string
	// AcceptedBy 是接受风险的 Owner 身份(§8.10 owner risk acceptance;
	// Owner 是决策角色,不是天然安全观察者 —— 这里只记录「谁接受了风险」,
	// 不改变任何独立性推导)。
	AcceptedBy string
	// AcceptedAt 是 Owner 风险接受时刻(必填;构造时归一化为 UTC)。
	AcceptedAt time.Time
}

// IndependenceWaiver 是一条 §8.10 independence 豁免 provenance:Owner 对
// 运行期 independence 损失的显式风险接受。字段全部私有、只读访问;它作为
// HumanApprovalEvidence 的组成部分随批准行 append-only 落库,永不改写。
type IndependenceWaiver struct {
	category               RiskCategory
	gaps                   []CoverageGap
	compromisedAcceptances []string
	compromisedBindings    []evidence.BindingRef
	exposureRefs           []evidence.ExposureID
	reasonCode             string
	acceptedBy             string
	acceptedAt             time.Time
}

// NewWaiver 构造一条豁免 provenance。category/gaps/reason_code/accepted_by/
// accepted_at 任一缺失、gap 引用空 acceptance、gap 重复、任何 provenance 列表
// 含空项、或试图把 planned coverage 缺失列入豁免时返回 error 且不产生豁免;
// 成功时 AcceptedAt 归一化为 UTC,后续对 spec 的改动不影响已构造豁免。
func NewWaiver(spec WaiverSpec) (*IndependenceWaiver, error) {
	w := IndependenceWaiver{
		category:               spec.Category,
		gaps:                   cloneGaps(spec.Gaps),
		compromisedAcceptances: slices.Clone(spec.CompromisedAcceptances),
		compromisedBindings:    slices.Clone(spec.CompromisedBindings),
		exposureRefs:           slices.Clone(spec.ExposureRefs),
		reasonCode:             spec.ReasonCode,
		acceptedBy:             spec.AcceptedBy,
		acceptedAt:             spec.AcceptedAt.UTC(),
	}
	if err := w.validate(); err != nil {
		return nil, err
	}
	return &w, nil
}

// validate 校验豁免 provenance 的全部确定性约束;构造与外部输入防御共用。
// Planned coverage 缺失在此直接拒绝(AC4 第一道防线):运行期损失才有豁免。
func (w *IndependenceWaiver) validate() error {
	if w == nil {
		return fmt.Errorf("%w: 豁免为 nil", ErrInvalidWaiver)
	}
	if strings.TrimSpace(string(w.category)) == "" {
		return fmt.Errorf("%w: 缺少 risk category(policy 按类别判定可用性)", ErrInvalidWaiver)
	}
	if len(w.gaps) == 0 {
		return fmt.Errorf("%w: 缺少 coverage gap(豁免必须声明它覆盖的独立性缺口)", ErrInvalidWaiver)
	}
	seen := make(map[string]bool, len(w.gaps))
	for i, g := range w.gaps {
		if strings.TrimSpace(g.AcceptanceID) == "" {
			return fmt.Errorf("%w: gaps[%d] 引用空 acceptance id", ErrInvalidWaiver, i)
		}
		if g.Planned {
			return fmt.Errorf("%w: acceptance %s 是 design-time planned coverage 缺失(§8.5.1 Freeze 时已可见),不能列入豁免", ErrPlannedCoverageNotWaivable, g.AcceptanceID)
		}
		if seen[g.AcceptanceID] {
			return fmt.Errorf("%w: coverage gap 重复: %s", ErrInvalidWaiver, g.AcceptanceID)
		}
		seen[g.AcceptanceID] = true
	}
	for i, a := range w.compromisedAcceptances {
		if strings.TrimSpace(a) == "" {
			return fmt.Errorf("%w: compromised_acceptances[%d] 为空", ErrInvalidWaiver, i)
		}
	}
	for i, b := range w.compromisedBindings {
		if b.Project == "" || b.Verification == "" || b.BindingID == "" {
			return fmt.Errorf("%w: compromised_bindings[%d] 引用不完整(project/verification/binding id 必填)", ErrInvalidWaiver, i)
		}
	}
	for i, e := range w.exposureRefs {
		if strings.TrimSpace(string(e)) == "" {
			return fmt.Errorf("%w: exposure_refs[%d] 为空", ErrInvalidWaiver, i)
		}
	}
	if strings.TrimSpace(w.reasonCode) == "" {
		return fmt.Errorf("%w: 缺少 reason code(§8.10:豁免 provenance 至少含 reason code)", ErrInvalidWaiver)
	}
	if strings.TrimSpace(w.acceptedBy) == "" {
		return fmt.Errorf("%w: 缺少 owner risk acceptance(§8.10:必须记录接受风险的 Owner)", ErrInvalidWaiver)
	}
	if w.acceptedAt.IsZero() {
		return fmt.Errorf("%w: 缺少 accepted_at(风险接受必须可定位到时刻)", ErrInvalidWaiver)
	}
	return nil
}

// Category 返回豁免的风险类别。
func (w *IndependenceWaiver) Category() RiskCategory { return w.category }

// Gaps 返回豁免覆盖的独立性缺口(副本,契约声明顺序)。
func (w *IndependenceWaiver) Gaps() []CoverageGap { return cloneGaps(w.gaps) }

// CompromisedAcceptances 返回被危害 acceptance id 列表(副本)。
func (w *IndependenceWaiver) CompromisedAcceptances() []string {
	return slices.Clone(w.compromisedAcceptances)
}

// CompromisedBindings 返回被危害 binding 引用列表(副本)。
func (w *IndependenceWaiver) CompromisedBindings() []evidence.BindingRef {
	return slices.Clone(w.compromisedBindings)
}

// ExposureRefs 返回引用的 exposure 事实 id 列表(副本)。
func (w *IndependenceWaiver) ExposureRefs() []evidence.ExposureID {
	return slices.Clone(w.exposureRefs)
}

// ReasonCode 返回豁免原因代码。
func (w *IndependenceWaiver) ReasonCode() string { return w.reasonCode }

// AcceptedBy 返回接受风险的 Owner 身份。
func (w *IndependenceWaiver) AcceptedBy() string { return w.acceptedBy }

// AcceptedAt 返回 Owner 风险接受时刻(UTC)。
func (w *IndependenceWaiver) AcceptedAt() time.Time { return w.acceptedAt }

// ResolveStrictOutcome 按 §8.10 机械判定 strict 结论。
//
// 输入:
//   - policy:Project Verification Policy 的豁免可用性输入(AC3);
//   - unmet:strict 独立性要求未由适用证据集满足的 acceptance 集合(每条
//     带 Planned 标记;调用方来自 internal/verification 的 coverage 推导);
//   - waiver:Owner 已记录的豁免 provenance(nil = 无)。
//
// 判定顺序(确定性,显式报错而不是静默修正):
//
//  1. unmet 为空且无豁免 → StrictSatisfied;unmet 为空却带豁免 →
//     ErrWaiverOverreach(豁免零缺口的候选会污染 strict 统计);
//  2. unmet 含 planned 缺失 → ErrPlannedCoverageNotWaivable —— 在 policy
//     与豁免检查之前:任何 waiver、任何 policy 都不能修复 design-time
//     planned coverage(AC4);
//  3. 无豁免 → ErrWaiverRequired;policy 不允许该类别 → ErrWaiverNotPermitted;
//  4. 豁免覆盖集合与 unmet 集合精确一致才产出 StrictWaived;少列
//     → ErrWaiverIncomplete,多列 → ErrWaiverOverreach(豁免必须如实覆盖
//     当前真实缺口;缺口变化后追加新的决定证据行,而不是复用旧豁免)。
//
// unmet 含空 acceptance id 或重复 id 是调用方错误,返回 error。
func ResolveStrictOutcome(policy WaiverPolicy, unmet []CoverageGap, waiver *IndependenceWaiver) (StrictOutcome, error) {
	unmetIDs := make([]string, 0, len(unmet))
	seen := make(map[string]bool, len(unmet))
	for i, g := range unmet {
		id := g.AcceptanceID
		if strings.TrimSpace(id) == "" {
			return "", fmt.Errorf("approval: unmet[%d] 引用空 acceptance id", i)
		}
		if seen[id] {
			return "", fmt.Errorf("approval: unmet 重复 acceptance id: %s(coverage 推导每条 acceptance 至多一项)", id)
		}
		seen[id] = true
		unmetIDs = append(unmetIDs, id)
	}

	if len(unmetIDs) == 0 {
		if waiver != nil {
			return "", fmt.Errorf("%w: 候选 strict 独立性已满足,不得携带豁免 provenance", ErrWaiverOverreach)
		}
		return StrictSatisfied, nil
	}

	// AC4:planned coverage 缺失永远不可豁免 —— 先于 policy/豁免检查。
	for _, g := range unmet {
		if g.Planned {
			return "", fmt.Errorf("%w: acceptance %s 是 design-time planned coverage 缺失(§8.5.1),任何 waiver 都不能修复它", ErrPlannedCoverageNotWaivable, g.AcceptanceID)
		}
	}

	if waiver == nil {
		return "", fmt.Errorf("%w: %d 条 acceptance 的 strict 独立性要求未满足", ErrWaiverRequired, len(unmetIDs))
	}
	if err := waiver.validate(); err != nil {
		return "", err
	}
	if !policy.Allows(waiver.Category()) {
		return "", fmt.Errorf("%w: risk category %q 未被 Project Verification Policy 允许", ErrWaiverNotPermitted, string(waiver.Category()))
	}

	covered := make(map[string]bool, len(waiver.gaps))
	for _, g := range waiver.gaps {
		covered[g.AcceptanceID] = true
	}
	var missing []string
	for _, id := range unmetIDs {
		if !covered[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("%w: 豁免未覆盖 %d 条未满足 acceptance(如 %s)", ErrWaiverIncomplete, len(missing), missing[0])
	}
	for _, g := range waiver.gaps {
		if !seen[g.AcceptanceID] {
			return "", fmt.Errorf("%w: 豁免声称的 acceptance %s 并未未满足", ErrWaiverOverreach, g.AcceptanceID)
		}
	}
	return StrictWaived, nil
}

// cloneGaps 返回 gaps 的深拷贝(nil 保持 nil),构造与访问共用。
func cloneGaps(gaps []CoverageGap) []CoverageGap {
	if gaps == nil {
		return nil
	}
	out := make([]CoverageGap, len(gaps))
	copy(out, gaps)
	return out
}
