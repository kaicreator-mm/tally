package verification

import (
	"fmt"
)

// 本文件实现 PRD §8.3 Evidence Qualification 的机械评估器(P1-16):
//
//   - §8.3.1 Baseline sensitivity:test 证据冻结前在 base_commit 上实际执行,
//     按 observed class 分为 assertion / compile_error / panic / error /
//     baseline pass / n/a,并给出对应的机械证明强度与最低审核要求;
//   - §8.3.2 Candidate green:candidate 在其适用 revision 上满足冻结
//     execution protocol 且 protected hash 一致。它与 baseline class 是两个
//     独立轴(§8.5.3:strength 与 visibility/independence 正交,green 的
//     candidate 不会把弱 baseline 变强);
//   - §8.3.3 Evidence-to-change relevance:最低 change_overlap 信号
//     (test 覆盖行 ∩ 变更可执行行 ≠ ∅)。它只是相关性信号,不是 coverage
//     KPI,更不是 Acceptance 语义的证明;
//   - §8.3.4 Failure fingerprint:只记录实际执行的 observed evidence,本
//     评估器不要求、也不预测 fingerprint。
//
// 边界(P1):本评估器只回答「最低机械区分能力 + 相关性信号 + 契约声明的
// 审核要求是否达标」。它不证明业务语义被满足(§8.1:Qualification 是工程
// 可信度机制,不是语义证明);高风险语义仍需 Verification Review / Owner
// Decision。非 test 证据类型的 qualification policy 由各自类型/ADR 定义
// (§8.12),本包只标记 n/a 并跳过 test 专属检查。

// BaselineClass — §8.3.1 表格的 baseline observed class。值与
// BaselineObserved.Class 的合法词表一致,另加 pass(观察为绿,无 red-green
// 敏感度)与 n/a(不适用 test red-green)。
type BaselineClass string

const (
	ClassAssertion    BaselineClass = "assertion"     // 失败发生在行为断言处:较强
	ClassCompileError BaselineClass = "compile_error" // 可能只引用了尚不存在的符号:弱
	ClassPanic        BaselineClass = "panic"         // 失败不一定与 Acceptance 语义相关:弱
	ClassError        BaselineClass = "error"         // 同上:弱
	ClassPass         BaselineClass = "pass"          // baseline 观察为绿:无 red-green 敏感度
	ClassNA           BaselineClass = "n/a"           // 非 test 证据,不适用 red-green
)

// Sensitivity — §8.3.1 的机械证明强度。值与 Qualification.BaselineSensitivity
// 的合法词表一致(strong | weak | none | n/a)。
type Sensitivity string

const (
	SensitivityStrong Sensitivity = "strong"
	SensitivityWeak   Sensitivity = "weak"
	SensitivityNone   Sensitivity = "none"
	SensitivityNA     Sensitivity = "n/a"
)

// review mode 词表(与 Qualification.ReviewMode 一致)。
const (
	ReviewModeCompact  = "compact"
	ReviewModeExplicit = "explicit"
)

// ProblemCode — 机械评估发现的违规类别。稳定机器码,调用方按 Code 分支,
// 不做错误字符串匹配。
type ProblemCode string

const (
	ProblemBaselineMissing        ProblemCode = "baseline_missing"         // test 证据未记录 baseline 实际执行
	ProblemBaselineContradiction  ProblemCode = "baseline_contradiction"   // observed status/class 自相矛盾或非法
	ProblemBaselineClassMissing   ProblemCode = "baseline_class_missing"   // fail 但未记录 class,无法机械分类
	ProblemSensitivityMismatch    ProblemCode = "sensitivity_mismatch"     // 契约声明的强度与机械推导不一致
	ProblemReviewModeMissing      ProblemCode = "review_mode_missing"      // 有机械最低审核要求但契约未声明 review_mode
	ProblemReviewBelowMinimum     ProblemCode = "review_below_minimum"     // 声明的 review_mode 低于该 class 的最低要求
	ProblemReviewRationaleMissing ProblemCode = "review_rationale_missing" // baseline pass 还须附 rationale
	ProblemOverlapViolated        ProblemCode = "overlap_violated"         // required 信号已观测且交集为空
	ProblemCandidateNotGreen      ProblemCode = "candidate_not_green"      // 已执行的 candidate run 不满足 §8.3.2
)

// Problem — 一条机械违规。Detail 面向人,Code 面向机器。
type Problem struct {
	Code   ProblemCode
	Detail string
}

// ReviewRequirement — §8.3.1 表格「最低审核要求」列的机械投影。
type ReviewRequirement struct {
	// Mode 是最低 review_mode:声明为 explicit 时满足任何 compact 以下的要求;
	// 空字符串表示该 class 没有机械可判定的 mode(配合 PolicyDefined)。
	Mode string
	// RationaleRequired 表示除 review 外还必须附非空 rationale(baseline pass 行)。
	RationaleRequired bool
	// PolicyDefined 表示最低要求由 evidence-specific policy 定义(§8.3.1 n/a 行),
	// 本包不做机械判定。
	PolicyDefined bool
}

// MinimumReview 返回一个 baseline class 的最低审核要求(§8.3.1):
//
//	assertion           → compact 可用
//	compile_error/panic/error → explicit verification review
//	baseline pass       → explicit verification review + rationale
//	n/a(及其他未知值)  → evidence-specific policy,机械评估不判定
func MinimumReview(class BaselineClass) ReviewRequirement {
	switch class {
	case ClassAssertion:
		return ReviewRequirement{Mode: ReviewModeCompact}
	case ClassCompileError, ClassPanic, ClassError:
		return ReviewRequirement{Mode: ReviewModeExplicit}
	case ClassPass:
		return ReviewRequirement{Mode: ReviewModeExplicit, RationaleRequired: true}
	default:
		return ReviewRequirement{PolicyDefined: true}
	}
}

// ChangeOverlapSignal — §8.3.3 的最低 change relevance 信号:
//
//	lines covered by required test ∩ changed executable lines ≠ ∅
//
// 它只是一个相关性信号:不是 coverage KPI,也不是「Acceptance 语义被满足」
// 的普遍证明。新增文件、generated/glue code、语言级 provider 等细节归 ADR。
type ChangeOverlapSignal struct {
	// Required:§8.3.3 对 type: test 的 required evidence 默认要求该信号;
	// Evaluate 对 test 证据强制置真,契约也可显式声明 change_overlap: required。
	Required bool
	// Known:是否已在具体 ChangeSet 上观测过(冻结时 candidate 尚不存在,可未知)。
	Known bool
	// Present:观测到的交集非空。
	Present bool
}

// Satisfied 报告信号是否已被观测满足。未开启 Required 时恒满足;
// 尚未观测(Known=false)不视为满足,但也不在此处判违规——是否违规由
// Evaluate 在「已观测且为空」时才报告,避免把冻结时的未知当成失败。
func (s ChangeOverlapSignal) Satisfied() bool {
	return !s.Required || (s.Known && s.Present)
}

// CandidateObservation — §8.3.2 candidate green 的运行期观察输入。它是运行
// 事实,不属于 frozen contract;与 baseline class 相互独立(§8.5.3)。
type CandidateObservation struct {
	// Status:适用 candidate revision 上的观察结果。"pass" | "fail";空 = 尚未执行。
	Status string
	// ProtocolHeld:是否按冻结的 execution protocol 执行(§8.7.2)。
	ProtocolHeld bool
	// HashMatch:protected hash 与冻结一致;binding 无 protected 声明时由调用方置 true。
	HashMatch bool
}

// Green 报告 candidate 是否满足 §8.3.2:在适用 revision 上 pass,且按冻结
// protocol 执行,且 protected hash 一致。尚未执行(Status 为空)不算 green。
func (o CandidateObservation) Green() bool {
	return o.Status == "pass" && o.ProtocolHeld && o.HashMatch
}

// Evaluation — 单条 binding 的机械 qualification 评估结果。它携带三个相互
// 独立的轴:class/sensitivity(来自 baseline 观察)、candidate green(来自
// candidate 观察)、change_overlap 信号(来自 ChangeSet 观察),并按 §8.3.1
// 报告最低审核要求与契约声明是否达标。Qualified 只表示「没有机械违规」,
// 不表示业务语义被证明。
type Evaluation struct {
	BindingID      string
	EvidenceType   EvidenceType
	Class          BaselineClass // 分类失败时为空
	Sensitivity    Sensitivity   // 分类失败时为空
	MinimumReview  ReviewRequirement
	DeclaredReview string

	ReviewSatisfied  bool
	CandidateGreen   bool
	Overlap          ChangeOverlapSignal
	OverlapSatisfied bool

	Problems []Problem
}

// Qualified 报告评估是否未发现机械违规。它是最低机械门槛:不含语义判断,
// 也不含「尚未观测」的项(那些在观测后由调用方重新评估)。
func (e Evaluation) Qualified() bool { return len(e.Problems) == 0 }

// ClassifyBaseline 按 §8.3.1 对 binding 的 baseline 观察做机械分类,返回
// observed class 与机械证明强度。
//
// 规则:
//   - 非 test 证据 → (n/a, n/a):test red-green 不适用;
//   - test 证据必须在 base_commit 上实际执行并记录 observed(§8.2 第 6 条、
//     §8.3.1),否则分类失败;
//   - observed pass(且未记录失败 class)→ pass / none;
//   - observed fail 按 class 映射:assertion→strong,compile_error / panic /
//     error→weak;fail 但未记录 class → 分类失败(无法机械分类)。
func ClassifyBaseline(b *Binding) (BaselineClass, Sensitivity, error) {
	class, sens, p := classifyBaseline(b)
	if p != nil {
		return "", "", fmt.Errorf("verification: %s", p.Detail)
	}
	return class, sens, nil
}

func classifyBaseline(b *Binding) (BaselineClass, Sensitivity, *Problem) {
	if b.Evidence.Type != EvidenceTest {
		return ClassNA, SensitivityNA, nil
	}
	if b.Baseline == nil || b.Baseline.Observed == nil || b.Baseline.Observed.Status == "" {
		return "", "", &Problem{
			Code:   ProblemBaselineMissing,
			Detail: fmt.Sprintf("binding %s 为 test 证据但未记录 baseline 实际执行(§8.2 第 6 条、§8.3.1:冻结前应在 base_commit 上执行)", b.ID),
		}
	}
	o := b.Baseline.Observed
	switch o.Status {
	case "pass":
		if o.Class != "" {
			return "", "", &Problem{
				Code:   ProblemBaselineContradiction,
				Detail: fmt.Sprintf("binding %s baseline observed 矛盾:status=pass 但记录了 class=%q", b.ID, o.Class),
			}
		}
		return ClassPass, SensitivityNone, nil
	case "fail":
		switch BaselineClass(o.Class) {
		case ClassAssertion:
			return ClassAssertion, SensitivityStrong, nil
		case ClassCompileError:
			return ClassCompileError, SensitivityWeak, nil
		case ClassPanic, ClassError:
			return BaselineClass(o.Class), SensitivityWeak, nil
		case "":
			return "", "", &Problem{
				Code:   ProblemBaselineClassMissing,
				Detail: fmt.Sprintf("binding %s baseline observed status=fail 但未记录 class,无法机械分类(§8.3.1)", b.ID),
			}
		default:
			return "", "", &Problem{
				Code:   ProblemBaselineContradiction,
				Detail: fmt.Sprintf("binding %s baseline observed class 非法: %q", b.ID, o.Class),
			}
		}
	default:
		return "", "", &Problem{
			Code:   ProblemBaselineContradiction,
			Detail: fmt.Sprintf("binding %s baseline observed status 非法: %q(只支持 pass|fail)", b.ID, o.Status),
		}
	}
}

// reviewRank 把 review_mode 映射为可比较强度:compact(1) < explicit(2);
// 未声明/未知为 0。
func reviewRank(mode string) int {
	switch mode {
	case ReviewModeCompact:
		return 1
	case ReviewModeExplicit:
		return 2
	default:
		return 0
	}
}

// Evaluate 对一条 binding 做完整的机械 qualification 评估(P1-16)。
//
// 输入:
//   - b:契约(或作者侧)binding,提供 evidence 类型、baseline 观察、
//     qualification 声明;
//   - cand:candidate green 观察(§8.3.2);尚未执行时传零值;
//   - overlap:change_overlap 信号观测(§8.3.3);尚未观测时 Known=false。
//
// 语义:
//   - test 证据的 change_overlap 默认 required(§8.3.3「默认需要」),与契约
//     是否显式声明无关;
//   - candidate green 与 baseline class 相互独立:green 不会提升 sensitivity,
//     baseline 弭也不会否定 green;已执行但不 green 才记违规(§8.3.2);
//   - 契约声明的 baseline_sensitivity 与机械推导不一致时记违规:评估器不允许
//     用声明把弱观察包装成强(「不假装证明业务语义」);
//   - 弱(assertion 以外)/pass 路径按契约要求 explicit review(pass 还须附
//     rationale);assertion 允许 compact;
//   - 非 test 证据 → n/a:跳过 test 专属检查,其 qualification policy 归
//     证据类型自身/ADR(§8.12);声明了非 n/a 强度则视为包装,记违规。
//
// b 为 nil 是调用错误,返回 error;其余一切结论都在 Evaluation.Problems 里。
func Evaluate(b *Binding, cand CandidateObservation, overlap ChangeOverlapSignal) (Evaluation, error) {
	if b == nil {
		return Evaluation{}, fmt.Errorf("verification: binding 为 nil,无法评估 qualification")
	}

	ev := Evaluation{
		BindingID:      b.ID,
		EvidenceType:   b.Evidence.Type,
		CandidateGreen: cand.Green(),
	}
	isTest := b.Evidence.Type == EvidenceTest

	// §8.3.3:test 证据的最低 change_overlap 信号默认 required。
	ev.Overlap = overlap
	ev.Overlap.Required = overlap.Required || isTest
	ev.OverlapSatisfied = ev.Overlap.Satisfied()
	if ev.Overlap.Required && ev.Overlap.Known && !ev.Overlap.Present {
		ev.Problems = append(ev.Problems, Problem{
			Code:   ProblemOverlapViolated,
			Detail: fmt.Sprintf("binding %s 的 change_overlap 信号已观测且为空:required test 覆盖行与变更可执行行无交集(§8.3.3)", b.ID),
		})
	}

	// §8.3.2:candidate green 与 baseline 独立;已执行但不 green 才违规。
	if isTest && cand.Status != "" && !cand.Green() {
		ev.Problems = append(ev.Problems, Problem{
			Code:   ProblemCandidateNotGreen,
			Detail: fmt.Sprintf("binding %s 的 candidate run 已执行但不满足 green(§8.3.2:需 pass + 冻结 protocol + protected hash 一致)", b.ID),
		})
	}

	class, sens, p := classifyBaseline(b)
	if p != nil {
		ev.Problems = append(ev.Problems, *p)
		return ev, nil // 分类失败时无法做 class 相关检查
	}
	ev.Class = class
	ev.Sensitivity = sens
	ev.MinimumReview = MinimumReview(class)

	q := b.Qualification
	if q != nil {
		ev.DeclaredReview = q.ReviewMode
	}

	// 声明的强度不得偏离机械推导(不允许把弱观察声明成强,反之亦然)。
	var declared Sensitivity
	if q != nil {
		declared = Sensitivity(q.BaselineSensitivity)
	}
	if declared != "" && declared != sens {
		ev.Problems = append(ev.Problems, Problem{
			Code:   ProblemSensitivityMismatch,
			Detail: fmt.Sprintf("binding %s 声明 baseline_sensitivity=%q 与机械推导 %q(%s)不一致(§8.3.1)", b.ID, declared, sens, class),
		})
	}

	// 最低审核要求(§8.3.1 第三列)。
	switch {
	case ev.MinimumReview.PolicyDefined:
		// n/a:审核要求由 evidence-specific policy 定义,本包不做机械判定。
		ev.ReviewSatisfied = true
	case ev.DeclaredReview == "":
		ev.Problems = append(ev.Problems, Problem{
			Code:   ProblemReviewModeMissing,
			Detail: fmt.Sprintf("binding %s 的 class %s 有最低审核要求 %q 但契约未声明 qualification.review_mode(§8.3.1)", b.ID, class, ev.MinimumReview.Mode),
		})
	default:
		if reviewRank(ev.DeclaredReview) < reviewRank(ev.MinimumReview.Mode) {
			ev.Problems = append(ev.Problems, Problem{
				Code:   ProblemReviewBelowMinimum,
				Detail: fmt.Sprintf("binding %s 声明 review_mode=%q 低于 class %s 的最低要求 %q(§8.3.1)", b.ID, ev.DeclaredReview, class, ev.MinimumReview.Mode),
			})
		}
		if ev.MinimumReview.RationaleRequired && (q == nil || q.Rationale == "") {
			ev.Problems = append(ev.Problems, Problem{
				Code:   ProblemReviewRationaleMissing,
				Detail: fmt.Sprintf("binding %s baseline 为 pass(无 red-green 敏感度),explicit review 还须附非空 rationale(§8.3.1)", b.ID),
			})
		}
		ev.ReviewSatisfied = true
		for i := range ev.Problems {
			switch ev.Problems[i].Code {
			case ProblemReviewBelowMinimum, ProblemReviewRationaleMissing, ProblemReviewModeMissing:
				ev.ReviewSatisfied = false
			}
		}
	}

	return ev, nil
}
