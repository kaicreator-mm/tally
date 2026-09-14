package review

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"tally/internal/approval"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/verification"
)

// 本文件实现 PRD §19.2/§19.3 的确定性 review 归账与验收映射(P1-40):
// 以固定 revision 的 ChangeSet 与 P1-07 的 Deterministic Hunk Inventory 为
// 分母,用 classify_rules.go 的规则集给每个 hunk(以及零 hunk 的
// binary/rename/mode-only 文件条目 —— INV-CHANGE-01:任何改动都不允许从
// review 中消失)分配恰好一个 primary bucket,映射 acceptance coverage
// (P1-19)与 independence coverage(P1-20),经 §8.10 的 strict 判定
// (P1-23)后产出 Unattributed 阻断项。
//
// 规则来源:
//   - §19.2 / INV-CHANGE-01:每个 hunk 恰好一个 primary bucket;可带多个
//     semantic tags;Unattributed Changes != ∅ 时 Owner 不得 Approve;
//   - §19.3:Acceptance Coverage(相关代码 + required evidence + 验证
//     结果)、Protected Evidence(LOCKED/expected hash/MODIFIED)、Scope
//     Violations、Risky Changes、Agent-added Tests、Documentation Changes、
//     Unattributed Changes 全部成为报告的确定性分区;
//   - §8.8.3 / Decision Lock 10:功能覆盖与 independence 覆盖是两个独立
//     维度,报告分开呈现,不互相补救;
//   - §8.10 / AC5:strict_waived 与 strict_satisfied 是类型级互斥结论,
//     豁免绝不能显示为「strict 已满足」;豁免 provenance 不完整/policy
//     不允许时按未满足处理并阻断,绝不静默放行。
//
// 边界(P1):
//   - 纯确定性推导:不调用任何 LLM。ReviewAgent 残差分类是可选辅助,只
//     能以显式 Attribution 输入进入,且只能填残差(Unattributed → 带验收
//     归因的 acceptance_code),永远不能改写规则事实 —— 被 relabel 的
//     protected/scope/风险事实会制造「所有改动都已看过」的假覆盖感;
//   - 功能覆盖缺口不是 approve 阻断项:VERIFIED 门归验证/状态域(§8.8.3),
//     报告只如实呈现;review 的 approve 阻断项是 Unattributed、未豁免的
//     independence 缺口(§8.5/§7.3 的验收级 strict 要求)与失效的豁免
//     provenance;
//   - 冻结契约在 Freeze 时已拒绝 planned coverage 缺失(§8.5.1),因此
//     本报告里的 independence 缺口都是运行期损失,才存在 waiver 路径;
//   - 报告是纯值投影(INV-STATE-01):同输入重算必得同报告,可整体重建;
//     不做任何 lifecycle 写入。

// 哨兵错误:调用方用 errors.Is 判因,包装文案只提供字段级上下文。
var (
	// ErrNilChangeSet 缺少 ChangeSet(review 的固定对象)。
	ErrNilChangeSet = errors.New("review: 缺少 ChangeSet")
	// ErrNilContract 缺少冻结 VerificationContract。
	ErrNilContract = errors.New("review: 缺少冻结 VerificationContract")
	// ErrReportMismatch 覆盖报告与冻结契约不属于同一 task/revision/内容,
	// 或缺少某条验收条款的覆盖行 —— 混 revision 的报告不允许进入归账。
	ErrReportMismatch = errors.New("review: 覆盖报告与冻结契约不一致")
	// ErrBadAttribution 显式归属非法:目标不存在、试图改写规则已归账的
	// 条目、引用契约外验收条款、或携带空标签/重复归属。
	ErrBadAttribution = errors.New("review: 无效的显式归属")
)

// Attribution 是一条显式归属(ReviewAgent 残差分类结论或 Owner 标注):
// 把规则无法归账的条目映射到验收条款。它只能填残差,不能改写任何规则
// 已归账的条目(LLM 分类是辅助,不是权威;INV-VER-02 同构约束)。
type Attribution struct {
	// HunkID 是目标 hunk;为空时按 Path 定位零 hunk 的文件条目
	// (binary/纯 rename/mode-only)。
	HunkID HunkID
	// Path 是 HunkID 为空时的目标条目主键路径。
	Path string
	// AcceptanceIDs 是归因到的验收条款(至少一条,必须存在于契约);
	// 归属必须回答「这条改动服务哪个 AC」,否则仍算未归账。
	AcceptanceIDs []string
	// Tags 是附加 semantic tags(可空;如 auth/security 这类非确定性
	// 风险标注,PRD §19.3 Remaining Risks/Risky Changes 的 LLM 维度)。
	Tags []string
}

// AccountingInput 是 BuildAccounting 的全部输入。Functional/Independence
// 必须是 Frozen 同一 revision 的覆盖推导结果(P1-19/P1-20 的输出)。
type AccountingInput struct {
	// ChangeSet 是被 review 的不可变候选快照(§19:review 输入必须是
	// 固定 revision 的 ChangeSet,不是正在变化的 workspace)。
	ChangeSet *changeset.ChangeSet
	// Inventory 是 P1-07 的确定性 hunk 清单(记账分母)。
	Inventory Inventory
	// Rules 是确定性归账规则集(用 RulesFor 从 scope+契约构造)。
	Rules Rules
	// Frozen 是冻结 VerificationContract(验收条款与 binding 的权威)。
	Frozen *verification.Frozen
	// Functional 是 P1-19 功能覆盖报告。
	Functional verification.FunctionalCoverageReport
	// Independence 是 P1-20 effective independence 覆盖报告。
	Independence verification.IndependenceCoverageReport
	// WaiverPolicy 是 Project Verification Policy 的豁免可用性输入;
	// 零值 = 不允许任何豁免(保守默认)。
	WaiverPolicy approval.WaiverPolicy
	// Waiver 是 Owner 已记录的 independence 豁免 provenance(nil = 无)。
	Waiver *approval.IndependenceWaiver
	// Attributions 是显式归属输入(可空)。
	Attributions []Attribution
}

// AccountingEntry 是一个记账条目:清单里的一个文本 hunk,或一个零 hunk
// 的文件条目(binary/纯 rename/mode-only/空文件新增 —— 不允许消失)。
// HunkID 为空表示后者。
type AccountingEntry struct {
	// HunkID 是稳定 hunk id;零 hunk 文件条目为空。
	HunkID HunkID
	// Path 是条目主键路径。
	Path string
	// Kind/Content 是所属文件条目的改动/内容种类(展示上下文)。
	Kind    FileChangeKind
	Content ContentKind
	// Bucket 是唯一 primary review bucket(AC1)。
	Bucket Bucket
	// Tags 是 semantic tags(确定性去重、顺序稳定)。
	Tags []string
	// ByRule 报告 bucket 是否由确定性规则分配;false = 显式归属。
	ByRule bool
}

// StrictStatus 是一条验收条款的 strict 独立性呈现状态(§8.10)。四个值
// 互斥:waived 永远不会以 satisfied 呈现(AC5)。
type StrictStatus string

const (
	// StrictNotRequired:条款未声明独立性要求(none)。
	StrictNotRequired StrictStatus = "not_required"
	// StrictSatisfied:effective independence 覆盖直接满足,无豁免参与。
	StrictSatisfied StrictStatus = "strict_satisfied"
	// StrictWaived:未被证据满足,但由 policy 允许且完整覆盖缺口的显式
	// 豁免接受风险。显示上永远是 strict_waived,不是 strict_satisfied。
	StrictWaived StrictStatus = "strict_waived"
	// StrictUnmet:未被满足且没有有效豁免 —— 显式缺口,阻断 approve。
	StrictUnmet StrictStatus = "strict_unmet"
)

// AcceptanceReview 是一条验收条款的 review 视图(§19.3 Acceptance
// Coverage):相关代码 + 功能覆盖 + independence 覆盖 + strict 结论。
type AcceptanceReview struct {
	AcceptanceID string
	// Statement 是验收陈述(冻结契约原文)。
	Statement string
	// RelatedHunks 是带 ac:<ID> 标签的 hunk id(契约声明顺序下按清单
	// 顺序);零 hunk 文件条目不进入此列表(仍在 Entries 中带标签)。
	RelatedHunks []HunkID
	// FunctionalCovered/FunctionalGap/FunctionalSatisfiedBy/FunctionalBlocking
	// 是 P1-19 的功能覆盖事实(展示;VERIFIED 门不在这里)。
	FunctionalCovered     bool
	FunctionalGap         verification.FunctionalGap
	FunctionalSatisfiedBy []string
	FunctionalBlocking    []string
	// IndependenceRequired/Covered/SatisfiedBy 是 P1-20 的 effective
	// independence 覆盖事实(与功能覆盖分开呈现,§8.8.3)。
	IndependenceRequired    verification.IndependenceRequired
	IndependenceCovered     bool
	IndependenceSatisfiedBy []string
	// Strict 是本条款的 strict 呈现状态(AC5:互斥,不混淆)。
	Strict StrictStatus
}

// ProtectedTouch 是被候选触及的一个 protected evidence 文件(§19.3
// Protected Evidence 的确定性部分:文件出现在 ChangeSet diff 中即
// MODIFIED;expected hash 来自契约声明,比对归验证域)。
type ProtectedTouch struct {
	Path         string
	BindingID    string
	AcceptanceID string
	// ExpectedHash 是契约声明的 baseline_hash(可空)。
	ExpectedHash string
}

// BlockerKind 是 approve 阻断项的种类。
type BlockerKind string

const (
	// BlockerUnattributed:存在未归账条目(INV-CHANGE-01/§19.2)。
	BlockerUnattributed BlockerKind = "unattributed"
	// BlockerIndependenceGap:验收条款的 strict 独立性要求未被满足且没有
	// 有效豁免(§8.5/§7.3;planned 缺失在 Freeze 已被拒绝)。
	BlockerIndependenceGap BlockerKind = "independence_gap"
	// BlockerStrictResolution:豁免 provenance 失效(policy 不允许、覆盖
	// 不完整、超范围、或对已满足的候选携带豁免)—— 按 §8.10 绝不能把
	// 这种候选呈现为 satisfied/waived。
	BlockerStrictResolution BlockerKind = "strict_resolution"
)

// Blocker 是一个 approve 阻断项。字段按 Kind 有值:Unattributed →
// HunkID/Path;IndependenceGap → AcceptanceID;StrictResolution → Err。
type Blocker struct {
	Kind         BlockerKind
	HunkID       HunkID
	Path         string
	AcceptanceID string
	// Err 携带 strict 判定的原始错误(approval 包哨兵,errors.Is 判因)。
	Err error
}

// AccountingReport 是确定性 review 归账报告:纯值投影,同输入重算必等。
type AccountingReport struct {
	// ChangeSetID/Project/BaseRevision/CandidateRevision 是被 review 候选
	// 的身份(§19:review 输入是固定 revision 的 ChangeSet)。
	ChangeSetID       changeset.ChangeSetID
	Project           core.ProjectID
	BaseRevision      string
	CandidateRevision string

	// Entries 是全部记账条目(清单顺序:文件按路径、hunk 按 diff 顺序,
	// 零 hunk 文件条目在其 hunk 之前)。长度 = 清单 hunk 总数 + 零 hunk
	// 文件条目数,与清单一一对应(AC1)。
	Entries []AccountingEntry

	// Acceptances 是逐验收条款的 review 视图(冻结契约声明顺序)。
	Acceptances []AcceptanceReview

	// ProtectedTouched 是被触及的 protected evidence(按 Path+BindingID
	// 排序;§19.3 Protected Evidence)。
	ProtectedTouched []ProtectedTouch

	// Blockers 是全部 approve 阻断项。空 = 允许 Approve。
	Blockers []Blocker

	// StrictOutcome 是候选级 §8.10 strict 结论;判定失败时为空值,
	// 失败原因在 StrictResolution(绝不把失败静默折算成 satisfied)。
	StrictOutcome approval.StrictOutcome
	// StrictResolution 携带 ResolveStrictOutcome 的错误(nil = 判定成功)。
	StrictResolution error
}

// UnattributedCount 是未归账条目数(PRD §19.4 unattributed_hunk_count 的
// 确定性来源)。
func (r AccountingReport) UnattributedCount() int {
	n := 0
	for _, e := range r.Entries {
		if e.Bucket == BucketUnattributed {
			n++
		}
	}
	return n
}

// UnattributedEntries 返回未归账条目(保持 Entries 顺序)。
func (r AccountingReport) UnattributedEntries() []AccountingEntry {
	var out []AccountingEntry
	for _, e := range r.Entries {
		if e.Bucket == BucketUnattributed {
			out = append(out, e)
		}
	}
	return out
}

// ApproveBlocked 报告当前是否必须阻断 Approve(§19.2:Unattributed 非空
// 不得 Approve;未豁免的 independence 缺口与失效豁免同样阻断)。
func (r AccountingReport) ApproveBlocked() bool {
	return len(r.Blockers) > 0
}

// BuildAccounting 对固定 ChangeSet 的清单做确定性归账、验收映射与 approve
// 阻断推导。任何输入形态非法(缺对象、报告跨 revision、归属指向不明)
// 都返回 error,绝不产出静默残缺的报告。
func BuildAccounting(input AccountingInput) (AccountingReport, error) {
	if input.ChangeSet == nil {
		return AccountingReport{}, ErrNilChangeSet
	}
	if input.Frozen == nil {
		return AccountingReport{}, ErrNilContract
	}
	if err := input.Rules.Validate(); err != nil {
		return AccountingReport{}, err
	}
	if err := validateReportsMatch(input.Frozen, input.Functional, input.Independence); err != nil {
		return AccountingReport{}, err
	}

	rep := AccountingReport{
		ChangeSetID:       input.ChangeSet.ID(),
		Project:           input.ChangeSet.Project(),
		BaseRevision:      input.ChangeSet.BaseRevision(),
		CandidateRevision: input.ChangeSet.CandidateRevision(),
	}

	// 1) 记账条目:每个 hunk 恰好一条;零 hunk 文件条目也各得一条
	// (INV-CHANGE-01:binary/rename/mode-only 不允许从 review 消失)。
	for _, f := range input.Inventory.Files {
		if len(f.Hunks) == 0 {
			bucket, tags := input.Rules.ClassifyFile(f)
			rep.Entries = append(rep.Entries, AccountingEntry{
				Path: f.Path, Kind: f.Kind, Content: f.Content,
				Bucket: bucket, Tags: tags, ByRule: true,
			})
			continue
		}
		bucket, tags := input.Rules.ClassifyFile(f)
		for _, h := range f.Hunks {
			rep.Entries = append(rep.Entries, AccountingEntry{
				HunkID: h.ID, Path: f.Path, Kind: f.Kind, Content: f.Content,
				Bucket: bucket, Tags: dedupeStrings(append([]string(nil), tags...)),
				ByRule: true,
			})
		}
	}

	// 2) 显式归属:只能填残差,不能改写规则事实。
	accIDs := map[string]bool{}
	for _, acc := range input.Frozen.Contract().Acceptances {
		accIDs[acc.ID] = true
	}
	if err := applyAttributions(rep.Entries, input.Attributions, accIDs); err != nil {
		return AccountingReport{}, err
	}

	// 3) Protected Evidence:被触及的 protected 文件(§19.3)。
	rep.ProtectedTouched = protectedTouches(input.Inventory, input.Rules.Protected)

	// 4) Acceptance Coverage + Independence Coverage + strict 结论。
	functional := indexFunctional(input.Functional)
	independence := indexIndependence(input.Independence)

	var unmetRows []verification.AcceptanceIndependence

	for _, acc := range input.Frozen.Contract().Acceptances {
		ar := AcceptanceReview{
			AcceptanceID: acc.ID,
			Statement:    acc.Statement,
		}
		fc, ok := functional[acc.ID]
		if !ok {
			return AccountingReport{}, fmt.Errorf("%w: 功能覆盖报告缺少条款 %s", ErrReportMismatch, acc.ID)
		}
		ar.FunctionalCovered = fc.Covered
		ar.FunctionalGap = fc.Gap
		ar.FunctionalSatisfiedBy = fc.SatisfiedBy
		ar.FunctionalBlocking = fc.Blocking
		ic, ok := independence[acc.ID]
		if !ok {
			return AccountingReport{}, fmt.Errorf("%w: independence 覆盖报告缺少条款 %s", ErrReportMismatch, acc.ID)
		}
		ar.IndependenceRequired = ic.Required
		ar.IndependenceCovered = ic.Covered
		ar.IndependenceSatisfiedBy = ic.SatisfiedBy

		switch {
		case ic.Required == verification.IndependenceNone:
			ar.Strict = StrictNotRequired
		case ic.Covered:
			ar.Strict = StrictSatisfied
		default:
			ar.Strict = StrictUnmet // 先按未满足;waived 在 strict 判定后回填
			unmetRows = append(unmetRows, ic)
		}
		rep.Acceptances = append(rep.Acceptances, ar)
	}

	// 5) §8.10 strict 判定(P1-23):缺口全部来自运行期损失(Freeze 已拒
	// 绝 planned 缺失),因此 CoverageGap.Planned 恒为 false。
	gaps := make([]approval.CoverageGap, 0, len(unmetRows))
	for _, u := range unmetRows {
		gaps = append(gaps, approval.CoverageGap{AcceptanceID: u.AcceptanceID})
	}
	outcome, serr := approval.ResolveStrictOutcome(input.WaiverPolicy, gaps, input.Waiver)
	rep.StrictOutcome = outcome
	rep.StrictResolution = serr

	waived := serr == nil && outcome == approval.StrictWaived
	for i := range rep.Acceptances {
		ar := &rep.Acceptances[i]
		// Related code(§19.3):带 ac:<ID> 标签的 hunk(规则 evidence 命中
		// 与显式归属都计入);保持 Entries 顺序,内容确定性。
		for _, e := range rep.Entries {
			if e.HunkID != "" && hasTag(e.Tags, TagACPrefix+ar.AcceptanceID) {
				ar.RelatedHunks = append(ar.RelatedHunks, e.HunkID)
			}
		}
		if ar.Strict != StrictUnmet {
			continue
		}
		if waived {
			// AC5:结论只能是 strict_waived,绝不显示为 strict_satisfied。
			ar.Strict = StrictWaived
		} else {
			rep.Blockers = append(rep.Blockers, Blocker{
				Kind:         BlockerIndependenceGap,
				AcceptanceID: ar.AcceptanceID,
			})
		}
	}
	// 豁免 provenance 失效(policy 不允许/覆盖不完整/超范围/空缺口带豁免):
	// 判定本身不可信,作为独立阻断项如实呈现。ErrWaiverRequired 只是
	// 「没有豁免」,缺口阻断项已经表达,不重复报。
	if serr != nil && !errors.Is(serr, approval.ErrWaiverRequired) {
		rep.Blockers = append(rep.Blockers, Blocker{Kind: BlockerStrictResolution, Err: serr})
	}

	// 6) Unattributed 阻断(§19.2 / AC3)。
	for _, e := range rep.Entries {
		if e.Bucket == BucketUnattributed {
			rep.Blockers = append(rep.Blockers, Blocker{
				Kind:   BlockerUnattributed,
				HunkID: e.HunkID,
				Path:   e.Path,
			})
		}
	}

	return rep, nil
}

// validateReportsMatch 校验两份覆盖报告确实属于冻结契约的同一 revision:
// task、revision 与内容哈希逐一相等(§8.8.2:跨 revision 的报告是历史
// 事实,混入当前归账会制造假覆盖)。
func validateReportsMatch(f *verification.Frozen, functional verification.FunctionalCoverageReport, independence verification.IndependenceCoverageReport) error {
	for _, c := range []struct {
		name string
		task string
		rev  int
		hash string
	}{
		{"功能覆盖报告", functional.TaskID, functional.ContractRevision, functional.ContractHash},
		{"independence 覆盖报告", independence.TaskID, independence.ContractRevision, independence.ContractHash},
	} {
		if c.task != f.TaskID() || c.rev != f.Revision() || c.hash != f.Hash() {
			return fmt.Errorf("%w: %s(task=%s rev=%d hash=%s)与冻结契约(task=%s rev=%d hash=%s)不一致",
				ErrReportMismatch, c.name, c.task, c.rev, c.hash, f.TaskID(), f.Revision(), f.Hash())
		}
	}
	return nil
}

// indexFunctional / indexIndependence 把覆盖报告转成按条款 id 查询的映射。
func indexFunctional(r verification.FunctionalCoverageReport) map[string]verification.AcceptanceCoverage {
	m := make(map[string]verification.AcceptanceCoverage, len(r.Acceptances))
	for _, a := range r.Acceptances {
		m[a.AcceptanceID] = a
	}
	return m
}

func indexIndependence(r verification.IndependenceCoverageReport) map[string]verification.AcceptanceIndependence {
	m := make(map[string]verification.AcceptanceIndependence, len(r.Acceptances))
	for _, a := range r.Acceptances {
		m[a.AcceptanceID] = a
	}
	return m
}

// applyAttributions 把显式归属应用到记账条目。
// 只允许改写 Unattributed 条目;目标必须存在;验收 id 必须在契约内;
// 标签不得为空白;同一目标不得重复归属。确定性事实(protected/scope/
// evidence/risky/test/docs)不可被任何归属 relabel。
func applyAttributions(entries []AccountingEntry, attrs []Attribution, accIDs map[string]bool) error {
	byHunk := make(map[HunkID]int, len(entries))
	byContentlessPath := make(map[string]int)
	for i, e := range entries {
		if e.HunkID == "" {
			byContentlessPath[e.Path] = i
			continue
		}
		if _, dup := byHunk[e.HunkID]; dup {
			return fmt.Errorf("%w: hunk %s 在清单中重复", ErrBadAttribution, e.HunkID)
		}
		byHunk[e.HunkID] = i
	}

	seenTarget := make(map[string]bool, len(attrs))
	for i, a := range attrs {
		idx := -1
		if a.HunkID != "" {
			j, ok := byHunk[a.HunkID]
			if !ok {
				return fmt.Errorf("%w: attrs[%d] 引用清单中不存在的 hunk %s", ErrBadAttribution, i, a.HunkID)
			}
			idx = j
		} else {
			if strings.TrimSpace(a.Path) == "" {
				return fmt.Errorf("%w: attrs[%d] 缺少目标(hunk id 与 path 均为空)", ErrBadAttribution, i)
			}
			j, ok := byContentlessPath[a.Path]
			if !ok {
				return fmt.Errorf("%w: attrs[%d] 引用清单中不存在的零 hunk 条目 %q", ErrBadAttribution, i, a.Path)
			}
			idx = j
		}
		key := string(a.HunkID) + "\x00" + a.Path
		if seenTarget[key] {
			return fmt.Errorf("%w: attrs[%d] 重复归属同一目标", ErrBadAttribution, i)
		}
		seenTarget[key] = true

		e := &entries[idx]
		if e.Bucket != BucketUnattributed {
			return fmt.Errorf("%w: 目标 %s%s 已由确定性规则归账为 %q,不能被改写",
				ErrBadAttribution, a.HunkID, a.Path, e.Bucket)
		}
		if len(a.AcceptanceIDs) == 0 {
			return fmt.Errorf("%w: attrs[%d] 缺少验收归因(归属必须回答服务哪个 AC)", ErrBadAttribution, i)
		}
		var tags []string
		seenAC := map[string]bool{}
		for _, id := range a.AcceptanceIDs {
			if !accIDs[id] {
				return fmt.Errorf("%w: attrs[%d] 引用契约外验收条款 %q", ErrBadAttribution, i, id)
			}
			if seenAC[id] {
				return fmt.Errorf("%w: attrs[%d] 重复引用验收条款 %q", ErrBadAttribution, i, id)
			}
			seenAC[id] = true
			tags = append(tags, TagACPrefix+id)
		}
		for _, tag := range a.Tags {
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("%w: attrs[%d] 携带空标签", ErrBadAttribution, i)
			}
			tags = append(tags, tag)
		}
		e.Bucket = BucketAcceptanceCode
		e.Tags = dedupeStrings(append(append([]string(nil), e.Tags...), tags...))
		e.ByRule = false
	}
	return nil
}

// protectedTouches 收集被候选触及的 protected evidence 文件:任一清单
// 条目的参与路径(新路径 + rename 来源)等于 protected 声明路径即为触及
// —— 出现在 ChangeSet diff 中就是 MODIFIED(§19.3)。
func protectedTouches(inv Inventory, protected []EvidencePath) []ProtectedTouch {
	touched := map[string]bool{}
	for _, f := range inv.Files {
		for _, p := range entryPaths(f) {
			touched[p] = true
		}
	}
	var out []ProtectedTouch
	for _, p := range protected {
		if touched[p.Path] {
			out = append(out, ProtectedTouch{
				Path:         p.Path,
				BindingID:    p.BindingID,
				AcceptanceID: p.AcceptanceID,
				ExpectedHash: p.ExpectedHash,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].BindingID < out[j].BindingID
	})
	return out
}
