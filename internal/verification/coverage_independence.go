package verification

import (
	"fmt"
	"strings"
	"time"

	"tally/internal/core"
	"tally/internal/evidence"
)

// 本文件实现 PRD §7.3/§8.5/§8.5.2 的 Acceptance 级 effective independence
// coverage 推导(P1-20):对冻结 VerificationContract 的每一条 Acceptance
// Clause,按其 independence_required(none < source < information,Decision
// Lock 10)推导「当前 applicable evidence set 中是否存在 effective
// independence 满足要求的 evidence」。independence requirement 属于 Acceptance
// Clause,不属于单条 binding(P1-20 source fact:放在 binding 上会造成
// recovery self-lock)。
//
// 规则来源:
//   - §8.5/§7.3:none 不要求任何独立证据(AC1);source 要求至少一条 applicable
//     evidence 的 effective_source_independence = true(AC2);information 要求
//     至少一条 applicable evidence 同时满足 effective source independence 与
//     effective information independence(AC3)。两个维度必须在同一条 evidence
//     上同时成立:Information Independence 永远不能补救 Source Dependence,
//     不允许跨 evidence OR;
//   - §8.5.2:effective coverage 在执行后计算,只消费 applicable evidence set
//     —— 即 §8.8.2 意义下适用于当前契约 revision、且在求值对象 candidate 上
//     存在 valid 且可用判定记录的 evidence。运行期 exposure/selection
//     contamination 使 planned ✓ → effective ✗ 在此被推导暴露;Independence
//     Waiver 是后续 policy 层的独立风险接受标注,本推导没有 waiver/policy
//     输入(compromised 是 observed fact,不能被重新标注为 independent,§8.10);
//   - §8.2.2 / §8.6:per-binding 的 effective source independence(P1-17)与
//     effective information independence(P1-18)直接复用 internal/evidence 的
//     确定性推导。binding 的 declared basis 与 authoring_mode 以冻结契约为权威
//     来源(INV-VER-01/INV-VER-02:求值输入不能注入与冻结声明不同的 provenance),
//     调用方只提供活动接触 attestation 与判定/exposure 投影;
//   - §8.8.3 / Decision Lock 10:功能覆盖(P1-19)与 independence 覆盖是两个
//     独立维度。本推导不消费 PASS/FAIL 方向:被 exposure 危害的旧 binding 在此
//     只是不再满足 independence,它作为功能 regression gate 的事实保持原样
//     (append-only,AC4);fresh binding(§8.6.3:必须形成新 contract revision,
//     因而是不同 BindingRef)提供独立性,fresh ≠ independent —— 它同样要面对
//     上述全部规则;
//   - §8.8.3:independence 不按 evidence 数量计数,而按 Acceptance coverage
//     判断 —— 多条不满足要求的 evidence 永远推不出覆盖,一条满足即可(AC5);
//   - INV-STATE-02:报告为每条 acceptance 记录 required、覆盖结论、满足它的
//     binding id 与逐 binding 的两个维度推导结果,使覆盖结论可追溯到具体证据。
//
// 边界(P1):
//   - planned coverage(§8.5.1)在 freeze 校验中(types.go validatePlannedCoverage);
//     本文件只做执行后的 effective coverage;
//   - 不设置 lifecycle 状态,不解释 qualification(§8.3)与功能 PASS/FAIL;
//     只产出供 State Engine 消费的覆盖事实。

// IndependenceContext — effective coverage 的求值上下文(§8.5.2:coverage 是
// coverage(acceptance, changeset),相对候选的实际实现权威 provenance 推导)。
type IndependenceContext struct {
	// Project — holdout 消费键(evidence.BindingRef)的 project 分量。
	Project core.ProjectID
	// Verification — 被求值冻结契约的内容身份(即 Frozen.Hash()),BindingRef
	// 的 revision 分量;与契约不一致时判定/exposure 永远无法匹配,显式报错。
	Verification evidence.VerificationRevision
	// ChangeSet — 求值对象候选(immutable ChangeSet)。
	ChangeSet evidence.ChangeSetID
	// CandidateFixedAt — 候选快照固化时间;零值 = 无法证明固化,选择暴露取
	// 污染侧(§8.6.2 保守默认)。
	CandidateFixedAt time.Time
	// Lineage — 候选 change lineage:候选自身在前,同实现决策内的祖先随后。
	Lineage []evidence.ChangeSetID
	// Implementation — 候选的实际实现权威 provenance(§8.2.2:effective source
	// independence 相对它推导;为空时推导保守取 source-dependent)。
	Implementation evidence.ImplementationProvenance
	// Precommit — 该候选的机械可验证选择 precommit(§8.6.2);零值 = 无。
	Precommit evidence.SelectionPrecommit
}

// IndependenceEvidence — 一条契约 binding 的执行后独立性求值输入。binding 的
// declared source basis 与 authoring_mode 以冻结契约为权威,不在此重复声明
// (INV-VER-01/02);调用方只提供活动接触 attestation 与 append-only 事实投影。
type IndependenceEvidence struct {
	// BindingID — 契约中的 binding id。
	BindingID string
	// Revision — 该求值输入针对的契约 revision(§8.8.2:evidence 只适用于它
	// 绑定的 revision;其他 revision 的输入是历史事实,不参与当前推导)。
	Revision int
	// CandidateBlind — post_hoc authoring 的 candidate-blind attestation
	// (§8.2.1;仅 post_hoc 被消费,pre_commit 的 candidate 缺席由 mode 自证)。
	CandidateBlind bool
	// AuthoredInLineage — 该证据作为当前实现决策 lineage 的一部分被首次创作
	// (§8.2.2 活动接触;与 actor 身份无关)。
	AuthoredInLineage bool
	// ChangedInLineage — 既存证据内容在当前实现决策 lineage 内被修改(§8.2.2)。
	ChangedInLineage bool
	// Verdicts — 该 binding 的判定记录投影(全部 candidate;推导按 BindingRef +
	// 当前 candidate 选择)。visible binding 同样需要:applicable evidence set
	// 的成员资格由当前 candidate 上 valid 且可用的判定记录确立。
	Verdicts []evidence.VerdictRecord
	// Exposures — 全部 exposure facts(§8.6.1);推导按 BindingRef 选择,其他
	// binding 的 exposure 永远不会跨 binding 泄漏。
	Exposures []evidence.Exposure
}

// BindingIndependence — 单条 binding 的独立性推导明细(INV-STATE-02 可追溯)。
type BindingIndependence struct {
	BindingID string
	// Applicable — 该 binding 属于当前 applicable evidence set:在求值对象
	// candidate 上存在 valid 且可用的判定记录(§8.5.2/§8.8.2)。
	Applicable bool
	// SourceIndependent — effective_source_independence(§8.2.2,P1-17 推导)。
	SourceIndependent bool
	// InformationIndependent — effective information independence(§8.5/§8.6,
	// P1-18 推导);visible binding 恒为 false。
	InformationIndependent bool
	// Qualifies — 该 binding 是否满足所属 acceptance 的 independence_required:
	// none 时任何 applicable evidence 均视为满足(无独立性约束);source 时
	// 要求 Applicable && SourceIndependent;information 时要求三者同时成立。
	Qualifies bool
}

// AcceptanceIndependence — 单条 Acceptance Clause 的 independence 覆盖推导结果。
type AcceptanceIndependence struct {
	AcceptanceID string
	// Required — 归一化后的独立性要求(空声明 = none,PRD §7.3)。
	Required IndependenceRequired
	// Covered — none 恒为 true(不要求任何独立证据,AC1);source/information
	// 要求至少一条 qualifying 的 applicable evidence。
	Covered bool
	// SatisfiedBy — 满足独立性要求的 binding id(契约声明顺序);none 恒为空
	// (没有独立性要求,也就没有任何 binding「提供独立性」)。
	SatisfiedBy []string
	// Bindings — 逐 binding 推导明细(契约声明顺序)。
	Bindings []BindingIndependence
}

// IndependenceCoverageReport — 整份冻结契约的 effective independence coverage。
type IndependenceCoverageReport struct {
	TaskID           string
	ContractRevision int
	ContractHash     string
	Acceptances      []AcceptanceIndependence
	// Unmatched — 引用了契约外 binding id 的求值输入(supplementary 或调用方
	// 错误),不参与推导;按首次出现顺序去重列出(INV-STATE-02)。
	Unmatched []string
	// Complete — 所有 acceptance 的 independence_required 均被满足。false 表示
	// 存在 planned ✓ → effective ✗ 的 acceptance;Waiver 判定归 policy 层,
	// 不在本推导。
	Complete bool
}

// DeriveIndependenceCoverage 对冻结契约推导 Acceptance 级 effective independence
// coverage(P1-20)。逐 Acceptance Clause 判定其 independence_required 是否由
// 当前 applicable evidence set 满足;per-binding 的两个独立性维度直接复用
// internal/evidence 的确定性推导,本函数只做 Acceptance 级组合。
//
// f、ctx 为 nil/不完整,或求值输入形态非法时返回 error;其余一切结论都在报告中
// (包括不覆盖),不做静默修正。
func DeriveIndependenceCoverage(f *Frozen, ctx IndependenceContext, records []IndependenceEvidence) (IndependenceCoverageReport, error) {
	if f == nil {
		return IndependenceCoverageReport{}, fmt.Errorf("verification: contract 为 nil,无法推导 independence coverage")
	}
	if ctx.Project == "" {
		return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值上下文缺少 project(holdout 消费键分量)")
	}
	if ctx.Verification == "" {
		return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值上下文缺少 verification revision 身份")
	}
	// BindingRef 的 revision 分量就是冻结契约的内容哈希(internal/evidence
	// BindingRef 契约);与被求值契约不一致时判定/exposure 永远无法命中,显式
	// 报错而不是静默推出全不覆盖。
	if ctx.Verification != evidence.VerificationRevision(f.Hash()) {
		return IndependenceCoverageReport{}, fmt.Errorf("verification: 求值上下文 verification 身份 %s 与被求值契约哈希 %s 不一致", ctx.Verification, f.Hash())
	}
	if ctx.ChangeSet == "" {
		return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值上下文缺少候选 ChangeSet(§8.5.2:coverage 是 coverage(acceptance, changeset))")
	}

	rep := IndependenceCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances:      make([]AcceptanceIndependence, len(f.c.Acceptances)),
		Complete:         true,
	}

	// 契约 binding id 集合。Contract.Validate 保证 binding id 全局唯一且跨
	// acceptance 不重复,因此按 id 分派无归属歧义。
	bindingIDs := map[string]bool{}
	for i := range f.c.Acceptances {
		acc := &f.c.Acceptances[i]
		for j := range acc.EvidenceBindings {
			bindingIDs[acc.EvidenceBindings[j].ID] = true
		}
	}

	// 分派求值输入:契约外 id 列入 Unmatched;历史 revision 的输入按 §8.8.2
	// 跳过;同 binding 的重复输入是调用方错误(authoring provenance 每 binding
	// 唯一),显式报错而不是静默合并。
	inputs := map[string]*IndependenceEvidence{}
	unmatchedSeen := map[string]bool{}
	for k := range records {
		r := &records[k]
		if strings.TrimSpace(r.BindingID) == "" {
			return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值输入缺少 binding id")
		}
		if r.Revision < 1 {
			return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值输入 %s revision 必须 >= 1(当前 %d);求值必须绑定具体契约 revision(§8.8.2)", r.BindingID, r.Revision)
		}
		if !bindingIDs[r.BindingID] {
			if !unmatchedSeen[r.BindingID] {
				unmatchedSeen[r.BindingID] = true
				rep.Unmatched = append(rep.Unmatched, r.BindingID)
			}
			continue
		}
		if r.Revision != f.Revision() {
			continue
		}
		if _, dup := inputs[r.BindingID]; dup {
			return IndependenceCoverageReport{}, fmt.Errorf("verification: binding %s 出现多条当前 revision 的求值输入;authoring provenance 每 binding 唯一,请合并 Verdicts/Exposures 投影", r.BindingID)
		}
		inputs[r.BindingID] = r
	}

	for i := range f.c.Acceptances {
		acc := &f.c.Acceptances[i]
		required := acc.IndependenceRequired
		if required == "" {
			required = IndependenceNone
		}
		ac := AcceptanceIndependence{
			AcceptanceID: acc.ID,
			Required:     required,
			Bindings:     make([]BindingIndependence, len(acc.EvidenceBindings)),
		}

		for j := range acc.EvidenceBindings {
			b := &acc.EvidenceBindings[j]
			bi := BindingIndependence{BindingID: b.ID}
			if rec, ok := inputs[b.ID]; ok {
				ref := evidence.BindingRef{Project: ctx.Project, Verification: ctx.Verification, BindingID: b.ID}
				bi.Applicable = hasUsableVerdict(rec.Verdicts, ref, ctx.ChangeSet)
				authorship := evidence.EvidenceAuthorship{
					Basis:             declaredSourceBasis(b),
					AuthoringMode:     evidence.AuthoringMode(b.AuthoringMode),
					CandidateBlind:    rec.CandidateBlind,
					AuthoredInLineage: rec.AuthoredInLineage,
					ChangedInLineage:  rec.ChangedInLineage,
				}
				bi.SourceIndependent = evidence.EffectiveSourceIndependence(authorship, ctx.Implementation)
				bi.InformationIndependent = evidence.EffectiveInformationIndependence(evidence.InformationIndependenceInput{
					Binding:          ref,
					Visibility:       evidence.VisibilityPolicy(b.VisibilityPolicy),
					Candidate:        ctx.ChangeSet,
					CandidateFixedAt: ctx.CandidateFixedAt,
					Lineage:          ctx.Lineage,
					Exposures:        rec.Exposures,
					Verdicts:         rec.Verdicts,
					Precommit:        ctx.Precommit,
				})
			}
			// 无求值输入的 binding:not applicable,两个维度保守 false ——
			// 缺省的 provenance 不构成独立证据(§8.5.2)。
			switch required {
			case IndependenceNone:
				bi.Qualifies = bi.Applicable
			case IndependenceSource:
				bi.Qualifies = bi.Applicable && bi.SourceIndependent
			default: // IndependenceInformation
				// §8.5:同一 evidence 必须同时满足 source + information;
				// information independence 不能补救 source dependence。
				bi.Qualifies = bi.Applicable && bi.SourceIndependent && bi.InformationIndependent
			}
			// none 没有独立性要求:SatisfiedBy 保持空,不制造「该 binding 提供
			// 了独立性」的假象;覆盖结论不受影响(AC1)。
			if required != IndependenceNone && bi.Qualifies {
				ac.SatisfiedBy = append(ac.SatisfiedBy, b.ID)
			}
			ac.Bindings[j] = bi
		}

		if required == IndependenceNone {
			// AC1:none 不要求任何独立证据,恒覆盖(功能覆盖是否成立是另一维度)。
			ac.Covered = true
		} else {
			ac.Covered = len(ac.SatisfiedBy) > 0
		}
		if !ac.Covered {
			rep.Complete = false
		}
		rep.Acceptances[i] = ac
	}

	return rep, nil
}

// hasUsableVerdict 报告 binding ref 在 candidate 上是否存在 valid 且可用的判定
// 记录 —— applicable evidence set 的成员资格(§8.5.2)。invalidated 的记录失去
// 自身 validity(§8.8.1),error 无可用判定,均不计入;判定方向(pass/fail)不
// 在此判断 —— 功能门与 independence 覆盖是独立维度(§8.8.3)。
func hasUsableVerdict(verdicts []evidence.VerdictRecord, ref evidence.BindingRef, candidate evidence.ChangeSetID) bool {
	for _, v := range verdicts {
		if v.Binding == ref && v.ChangeSet == candidate && v.Valid && v.Verdict != evidence.VerdictError {
			return true
		}
	}
	return false
}

// declaredSourceBasis 返回冻结 binding 声明的 desired provenance basis(§8.2.2);
// 未声明时返回空 —— evidence 推导对未知 basis 保守取 source-dependent。
func declaredSourceBasis(b *Binding) evidence.SourceBasis {
	if b.Basis == nil {
		return evidence.SourceBasis("")
	}
	return evidence.SourceBasis(b.Basis.Source)
}
