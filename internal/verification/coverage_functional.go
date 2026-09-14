package verification

import (
	"fmt"
	"strings"
)

// 本文件实现 PRD §8.8.3 的功能门(Functional Gate)覆盖推导(P1-19):对冻结
// VerificationContract 的每一条 Acceptance Clause,推导「当前 revision 是否存在
// 适用且有效的 passing required evidence」。
//
// 规则来源:
//   - §8.1 Completeness:覆盖按 Acceptance Clause 判定(存在性),不按 evidence
//     数量判定 —— 一条 applicable 的有效 PASS 即可覆盖其 acceptance,多条 PASS
//     不能覆盖其他 acceptance(AC1);
//   - §8.8.2 Validity ≠ Applicability:evidence 可以永久 valid 但不再 applicable;
//     State Engine 对当前状态只消费 current desired revision + applicable
//     evidence。本推导只消费 Revision 等于当前契约 revision 的记录(AC2);
//   - §8.8.1 Failed Evidence 是 append-only Fact:对 immutable candidate 的合法
//     FAIL 是历史事实,新的 binding PASS 不能抹掉它;只有 confirmed
//     holdout_defect、SpecDefect、invalid-evidence 裁定等带 provenance 的正式
//     流程才使其失去自身 validity。因此任一条 valid + applicable 的 FAIL 都使
//     该 acceptance 不被覆盖(候选保持非 VERIFIED),直到它被正式 invalidated
//     或不再 applicable(AC3);
//   - §8.8.3 / Decision Lock 10:功能覆盖与 independence 覆盖是两个独立维度。
//     本文件只推导功能覆盖;independence 的 planned coverage(§8.5.1)在
//     freeze 校验中,effective coverage(§8.5.2)归后续任务;
//   - INV-STATE-02:每个事实性 lifecycle state 必须可证据追溯。报告为每条
//     acceptance 记录满足/阻断它的 binding id,使覆盖结论可追溯到具体证据。
//
// 边界(P1):
//   - 输入记录应已是经过 qualification(§8.3,含 §8.3.2 candidate green 判定)
//     的观察事实;本文件不重新推导 qualification,也不解释 held-out/exposure;
//   - 不设置 lifecycle 状态本身;它只产出供 State Engine 消费的覆盖事实。
//     Complete=false 表示当前 revision 存在未被功能覆盖的 acceptance,即候选
//     不能进入 VERIFIED(AC3 的契约级投影)。

// EvidenceStatus — 一条 required evidence 对 candidate 的已观察判定。
type EvidenceStatus string

const (
	EvidenceStatusPass EvidenceStatus = "pass"
	EvidenceStatusFail EvidenceStatus = "fail"
)

// EvidenceValidity — 记录自身的 validity(§8.8.1)。空值视为 valid:记录是
// 观察事实,只有带 provenance 的正式裁定(valid-evidence 裁定、confirmed
// holdout_defect、SpecDefect 等)才使其失效,不存在「默认失效」。
type EvidenceValidity string

const (
	EvidenceValidityValid       EvidenceValidity = "valid"
	EvidenceValidityInvalidated EvidenceValidity = "invalidated"
)

// EvidenceRecord — 一条 required evidence 的已观察判定记录(append-only 事实的
// 只读投影;同一 binding 的多次执行产生多条记录,本推导不丢弃任何一条)。
type EvidenceRecord struct {
	// BindingID — 契约中的 binding id。不在契约中的记录按 §8.12 视为
	// supplementary / 来源不明,不参与功能覆盖,只在报告中列出。
	BindingID string
	// Revision — 该判定针对的 VerificationContract revision(§8.8.2:evidence
	// 只 gate 它所绑定的 revision)。
	Revision int
	// Status — 观察判定:pass | fail。应已按冻结 execution protocol 得出
	// (§8.3.2 / §8.7.2 由 qualification 层保证)。
	Status EvidenceStatus
	// Validity — valid | invalidated;空 = valid。
	Validity EvidenceValidity
}

// validateRecord 校验单条记录的形态合法性。调用方错误显式报错,不做静默修正。
func validateRecord(r EvidenceRecord) error {
	if strings.TrimSpace(r.BindingID) == "" {
		return fmt.Errorf("verification: evidence record 缺少 binding id")
	}
	switch r.Status {
	case EvidenceStatusPass, EvidenceStatusFail:
	default:
		return fmt.Errorf("verification: evidence record %s status 非法: %q(只支持 pass|fail)", r.BindingID, r.Status)
	}
	switch r.Validity {
	case "", EvidenceValidityValid, EvidenceValidityInvalidated:
	default:
		return fmt.Errorf("verification: evidence record %s validity 非法: %q(只支持 valid|invalidated)", r.BindingID, r.Validity)
	}
	if r.Revision < 1 {
		return fmt.Errorf("verification: evidence record %s revision 必须 >= 1(当前 %d);判定必须绑定具体契约 revision(§8.8.2)", r.BindingID, r.Revision)
	}
	return nil
}

// FunctionalGap — acceptance 未被功能覆盖时的缺口类别(稳定机器词表,调用方
// 按 Gap 分支,不做字符串匹配推断)。
type FunctionalGap string

const (
	GapNone             FunctionalGap = ""
	GapNoEvidence       FunctionalGap = "no_evidence"        // acceptance 没有任何 planned binding(§8.1 Completeness)
	GapNoApplicablePass FunctionalGap = "no_applicable_pass" // 有 binding,但当前 revision 无有效 applicable 的 PASS
	GapBlockingFailure  FunctionalGap = "blocking_failure"   // 存在 valid + applicable 的 FAIL(§8.8.1,AC3)
)

// AcceptanceCoverage — 单条 Acceptance Clause 的功能覆盖推导结果。
type AcceptanceCoverage struct {
	AcceptanceID string
	// Covered — 该 acceptance 当前 revision 是否存在适用且有效的 passing
	// required evidence,且没有被同性质的 FAIL 阻断(§8.8.1/§8.8.2)。
	Covered bool
	// Gap — 未覆盖时的缺口类别;覆盖时为 GapNone。Blocking 与 SatisfiedBy
	// 同时非空时,Blocking 优先(有效 FAIL 不被新 PASS 抹掉,§8.8.1)。
	Gap FunctionalGap
	// SatisfiedBy — 满足覆盖的 binding id(契约声明顺序);空 = 无。
	SatisfiedBy []string
	// Blocking — valid + applicable 的 FAIL binding id(契约声明顺序);
	// 非空时 Covered 恒为 false,直到对应 FAIL 被正式 invalidated 或不再
	// applicable(AC3)。
	Blocking []string
}

// FunctionalCoverageReport — 整份冻结契约的功能覆盖推导结果。
type FunctionalCoverageReport struct {
	TaskID           string
	ContractRevision int
	ContractHash     string
	Acceptances      []AcceptanceCoverage
	// Unmatched — 引用了契约外 binding id 的记录(§8.12 supplementary 或调用方
	// 错误),不参与覆盖;按首次出现顺序去重列出以保持证据可追溯(INV-STATE-02)。
	Unmatched []string
	// Complete — 所有 acceptance 均被功能覆盖。false 表示当前 revision 存在
	// 未覆盖 acceptance,候选不能进入 VERIFIED。
	Complete bool
}

// DeriveFunctionalCoverage 对冻结契约推导功能覆盖(P1-19):逐 Acceptance
// Clause 判定「当前 revision 是否有适用且有效的 passing required evidence」。
//
// 推导规则:
//   - 只消费 Revision == 当前契约 revision 的记录(§8.8.2:旧 revision 的
//     PASS/FAIL 都是历史事实,只 gate 它绑定的 revision,对当前状态不可见);
//   - validity=invalidated 的记录不参与满足也不参与阻断(§8.8.1:失去自身
//     validity 后不再 gate);
//   - 同一 binding 存在多条有效 applicable 记录时保守合并:任一 FAIL 即阻断,
//     任一 PASS 即可满足;两者并存时 FAIL 优先(append-only 事实 + §8.7.2
//     禁止 cherry-pick:PASS 不抹掉 FAIL);
//   - 契约外 binding id 的记录不参与推导,列入 Unmatched。
//
// f 为 nil 或记录形态非法时返回 error;其余一切结论都在报告中。
func DeriveFunctionalCoverage(f *Frozen, records []EvidenceRecord) (FunctionalCoverageReport, error) {
	if f == nil {
		return FunctionalCoverageReport{}, fmt.Errorf("verification: contract 为 nil,无法推导功能覆盖")
	}

	// binding id → 所属 acceptance 下标。Contract.Validate 保证 binding id
	// 全局唯一,因此无跨 acceptance 归属歧义。
	owner := map[string]int{}
	for i := range f.c.Acceptances {
		acc := &f.c.Acceptances[i]
		for j := range acc.EvidenceBindings {
			owner[acc.EvidenceBindings[j].ID] = i
		}
	}

	rep := FunctionalCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances:      make([]AcceptanceCoverage, len(f.c.Acceptances)),
	}

	passed := map[string]bool{}
	failed := map[string]bool{}
	unmatchedSeen := map[string]bool{}

	for _, r := range records {
		if err := validateRecord(r); err != nil {
			return FunctionalCoverageReport{}, err
		}
		if _, ok := owner[r.BindingID]; !ok {
			if !unmatchedSeen[r.BindingID] {
				unmatchedSeen[r.BindingID] = true
				rep.Unmatched = append(rep.Unmatched, r.BindingID)
			}
			continue
		}
		// §8.8.2:只消费适用于当前 revision 的记录;其他 revision 的记录是
		// 历史事实,对当前状态既不满足也不阻断。
		if r.Revision != f.Revision() {
			continue
		}
		// §8.8.1:被正式 invalidated 的记录失去自身 validity,不再 gate。
		if r.Validity == EvidenceValidityInvalidated {
			continue
		}
		switch r.Status {
		case EvidenceStatusPass:
			passed[r.BindingID] = true
		case EvidenceStatusFail:
			failed[r.BindingID] = true
		}
	}

	complete := true
	for i := range f.c.Acceptances {
		acc := &f.c.Acceptances[i]
		ac := AcceptanceCoverage{AcceptanceID: acc.ID}
		for j := range acc.EvidenceBindings {
			id := acc.EvidenceBindings[j].ID
			if failed[id] {
				ac.Blocking = append(ac.Blocking, id)
			}
			if passed[id] {
				ac.SatisfiedBy = append(ac.SatisfiedBy, id)
			}
		}

		switch {
		case len(ac.Blocking) > 0:
			// AC3:一条 valid + applicable 的 FAIL 使该 acceptance 保持未覆盖,
			// 即使同 acceptance 的其他 binding 已 PASS(§8.8.1)。
			ac.Gap = GapBlockingFailure
		case len(acc.EvidenceBindings) == 0:
			// §8.1 Completeness:没有任何 planned binding 的 acceptance 无被
			// 验证路径(FreezeContract 会拒绝;这里兜底覆盖最小 Freeze 入口)。
			ac.Gap = GapNoEvidence
		case len(ac.SatisfiedBy) == 0:
			// AC1:按 acceptance 判定 —— 该 acceptance 当前没有任何适用且
			// 有效的 PASS(不关心其他 acceptance 有多少 PASS)。
			ac.Gap = GapNoApplicablePass
		default:
			ac.Covered = true
		}

		if !ac.Covered {
			complete = false
		}
		rep.Acceptances[i] = ac
	}
	rep.Complete = complete

	return rep, nil
}
