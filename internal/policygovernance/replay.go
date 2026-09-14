// replay.go 拥有证据边界的历史重放(P3-04):把候选策略放回「当时的录得
// 事实」上,回答三件事 —— 当时会不会命中、会新增哪些要求/门禁键、这些键的
// 结局能否从录得事实确定。绝不发明反事实执行结局(DL-10)。
//
// 证据边界(改动前必读):
//   - 触发输入轴(风险信号/变更路径/需求引用)必须被显式声明为「已录得」
//     (TriggerRecorded)。未录得时匹配判定 = INSUFFICIENT_EVIDENCE ——
//     「没录」不等于「没命中」,也不等于「没命中到可以忽略」;绝不猜测。
//   - 门禁键结局只来自录得的验证事实(RecordedGateResults:验证绑定结果按
//     gate 键的投影,来源 = verification 事实面):passed → would_have_passed,
//     failed → would_have_failed,缺席 → insufficient_evidence(一等输出,
//     不是错误、不是静默省略)。historical 代码「会不会过」只有录得事实能
//     确定时才判定;
//   - 匹配判定复用 engineeringpolicy.Match(要求 1:不建第二 DSL);冲突
//     复用 AnalyzeConflicts(P3-05);两者都是纯函数,同一证据快照重放恒等
//     (AC:确定性);
//   - unknown/unmatched 风险信号原样保真输出(INV-V14-09 同纪律);
//   - 政策不携带 assurance 等级(DL-08 模式面),assurance 差量在治理词汇内
//     表现为「要求键集差量」—— 新增门禁键即新增必须被验证满足的键;本重放
//     不猜测其通过性,只报告差量与录得结局。
package policygovernance

import (
	"errors"
	"fmt"
	"sort"

	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
)

// ErrInvalidReplayInput:重放输入不合法(候选无效 / 历史集合重复 id 等;
// engineeringpolicy.Match 的类型化错误原样透传)。
var ErrInvalidReplayInput = errors.New("policygovernance: 非法重放输入")

// ReplayMatchDecision 是匹配判定的封闭三值:matched / unmatched 之外,
// insufficient_evidence 是一等输出(DL-10:证据不足不是错误,是结论)。
type ReplayMatchDecision string

const (
	ReplayMatched      ReplayMatchDecision = "matched"
	ReplayUnmatched    ReplayMatchDecision = "unmatched"
	ReplayInsufficient ReplayMatchDecision = "insufficient_evidence"
)

// ReplayGateStatus 是单个门禁键结局的封闭三值:只有录得事实能确定时才给
// would_have_passed / would_have_failed;否则 insufficient_evidence。
type ReplayGateStatus string

const (
	GateWouldPass            ReplayGateStatus = "would_have_passed"
	GateWouldFail            ReplayGateStatus = "would_have_failed"
	GateInsufficientEvidence ReplayGateStatus = "insufficient_evidence"
)

// ReplayGateOutcome 是一个门禁键的重放结局。
type ReplayGateOutcome struct {
	GateKey string
	Status  ReplayGateStatus
}

// ReplayInput 是一次重放的证据快照:候选 + 当时的策略集 + 录得的触发输入与
// 门禁结局。全部值语义,确定性消费。
type ReplayInput struct {
	// Candidate 是被重放的候选策略。
	Candidate engineeringpolicy.Policy
	// Historical 是当时生效的策略集(来自 pinned 快照;P3-06 Active 面同源)。
	Historical []engineeringpolicy.Policy
	// TriggerRecorded 声明触发输入轴是否被录得。false → 匹配判定
	// INSUFFICIENT_EVIDENCE(不猜 matched/unmatched)。
	TriggerRecorded bool
	// Match 是录得的触发输入(TriggerRecorded=true 时消费;Risk 来自
	// risk.Derive 的录得信号,ChangedPaths/RequirementRefs 来自任务事实面)。
	Match engineeringpolicy.MatchInput
	// RecordedGateResults 是录得的门禁键结局(键 → "passed"|"failed";
	// 来源 = 验证绑定结果按 gate 键投影)。缺席键 = 证据不足,绝不猜。
	RecordedGateResults map[string]string
}

// ReplayResult 是重放产出:匹配判定 + 差量 + 逐键结局 + 冲突 + 保真信号。
// 全部列表确定性排序;同一证据快照重放恒等(AC)。
type ReplayResult struct {
	PolicyID string
	Decision ReplayMatchDecision

	// 命中依据(仅 matched;来自 engineeringpolicy.Match 的 basis)。
	MatchedRiskSignals     []string
	MatchedPaths           []string
	MatchedRequirementRefs []string

	// 差量:候选相对「当时命中的历史策略集」新增/不变的要求键与门禁键。
	NewlyRequiredKeys []string
	NewlyGatedKeys    []string
	UnchangedKeys     []string

	// 逐键结局(差量键;insufficient_evidence 一等呈现)。
	GateOutcomes []ReplayGateOutcome

	// 涉及候选的冲突发现(P3-05)。
	Conflicts []Conflict

	// 风险信号保真(INV-V14-09)。
	UnknownRiskSignals   []string
	UnmatchedRiskSignals []string
	// Notes 是确定性的人类可读说明(证据边界声明等)。
	Notes []string
}

// ReplayCandidate 对单个候选执行证据边界重放。历史集合为空 = 当时没有生效
// 策略(合法输入:全部差量键即候选自身要求)。
func ReplayCandidate(in ReplayInput) (ReplayResult, error) {
	if err := in.Candidate.Validate(); err != nil {
		return ReplayResult{}, fmt.Errorf("%w: 候选策略: %w", ErrInvalidReplayInput, err)
	}

	res := ReplayResult{PolicyID: in.Candidate.ID}

	// ---- 1. 匹配判定(证据边界)。 ----
	if !in.TriggerRecorded {
		res.Decision = ReplayInsufficient
		res.NotesAppend("触发输入轴未录得:无法判定当时是否命中(不猜测)。")
		return finishReplay(res, in), nil
	}
	cov, err := engineeringpolicy.Match([]engineeringpolicy.Policy{in.Candidate}, in.Match)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("%w: 候选匹配: %w", ErrInvalidReplayInput, err)
	}
	res.UnknownRiskSignals = riskIDs(cov.UnknownRiskSignals)
	res.UnmatchedRiskSignals = riskIDs(cov.UnmatchedRiskSignals)
	if len(cov.MatchedPolicies) == 0 {
		res.Decision = ReplayUnmatched
		return finishReplay(res, in), nil
	}
	res.Decision = ReplayMatched
	for _, m := range cov.MatchedPolicies {
		res.MatchedRiskSignals = append(res.MatchedRiskSignals, riskIDs(m.Basis.RiskSignals)...)
		for _, p := range m.Basis.Paths {
			res.MatchedPaths = append(res.MatchedPaths, p)
		}
		res.MatchedRequirementRefs = append(res.MatchedRequirementRefs, m.Basis.RequirementRefs...)
	}
	sort.Strings(res.MatchedRiskSignals)
	sort.Strings(res.MatchedPaths)
	sort.Strings(res.MatchedRequirementRefs)

	// ---- 2. 差量:相对「同样命中该变更的历史策略集」。 ----
	var histMatchedRequires, histMatchedGated []string
	if len(in.Historical) > 0 {
		hcov, err := engineeringpolicy.Match(in.Historical, in.Match)
		if err != nil {
			return ReplayResult{}, fmt.Errorf("%w: 历史集合匹配: %w", ErrInvalidReplayInput, err)
		}
		for _, m := range hcov.MatchedPolicies {
			histMatchedRequires = append(histMatchedRequires, m.Policy.Requires...)
			histMatchedGated = append(histMatchedGated, m.Policy.GateCondition.AllOf...)
		}
	}
	histReqSet := setOf(histMatchedRequires)
	histGateSet := setOf(histMatchedGated)

	newly := map[string]bool{}
	for _, k := range in.Candidate.Requires {
		if !histReqSet[k] {
			newly[k] = true
		}
	}
	for _, k := range in.Candidate.GateCondition.AllOf {
		if !histGateSet[k] {
			newly[k] = true
		}
	}
	res.NewlyRequiredKeys = sortedKeys(newly)
	var unchanged []string
	for _, k := range in.Candidate.Requires {
		if histReqSet[k] {
			unchanged = append(unchanged, k)
		}
	}
	res.UnchangedKeys = sortedCopy(unchanged)
	res.NewlyGatedKeys = sortedCopy(subtract(in.Candidate.GateCondition.AllOf, histGateSet))

	// ---- 3. 逐键结局:只认录得事实;缺席 = 证据不足(一等)。 ----
	for _, k := range res.NewlyRequiredKeys {
		switch in.RecordedGateResults[k] {
		case "passed":
			res.GateOutcomes = append(res.GateOutcomes, ReplayGateOutcome{GateKey: k, Status: GateWouldPass})
		case "failed":
			res.GateOutcomes = append(res.GateOutcomes, ReplayGateOutcome{GateKey: k, Status: GateWouldFail})
		default:
			res.GateOutcomes = append(res.GateOutcomes, ReplayGateOutcome{GateKey: k, Status: GateInsufficientEvidence})
		}
	}
	sort.SliceStable(res.GateOutcomes, func(i, j int) bool { return res.GateOutcomes[i].GateKey < res.GateOutcomes[j].GateKey })

	return finishReplay(res, in), nil
}

// finishReplay 统一收尾:冲突发现(P3-05,仅保留涉及候选者)+ 全列表确定性
// 排序。历史集合同序拼接候选后分析 —— 输入顺序无关由 AnalyzeConflicts 保证。
func finishReplay(res ReplayResult, in ReplayInput) ReplayResult {
	if len(in.Historical) > 0 {
		set := append(append([]engineeringpolicy.Policy(nil), in.Historical...), in.Candidate)
		for _, c := range AnalyzeConflicts(set) {
			for _, id := range c.PolicyIDs {
				if id == in.Candidate.ID {
					res.Conflicts = append(res.Conflicts, c)
					break
				}
			}
		}
	}
	return res
}

// NotesAppend 是 Notes 的确定性追加(测试与投影可读)。保留显式方法以避免
// 调用方随意拼装非确定内容。
func (r *ReplayResult) NotesAppend(note string) {
	r.Notes = append(r.Notes, note)
}

func riskIDs(ids []risk.SignalID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	sort.Strings(out)
	return out
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func subtract(xs []string, notIn map[string]bool) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if !notIn[x] {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
