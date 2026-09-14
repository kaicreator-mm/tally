package engineeringpolicy

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"tally/internal/risk"
)

// 策略匹配与 PolicyMatchCoverage(PRD §10,INV-V14-09;归 P1-05):
//
//   - 匹配是纯函数:同一策略集 + 同一任务/变更/风险事实 ⇒ 同一匹配结果,
//     与输入切片顺序无关(AC4)。无时间、无随机、无 I/O、无生命周期写入
//     (AC3:匹配只产出事实,接受/升级决策归下游的合同编译与验证层);
//   - 触发语义刻意最小:trigger 三个轴(risk_signals / paths /
//     requirement_refs)之间是「任一轴命中即命中」;轴内是精确值相等
//     (去首尾空白后比较)。没有 glob、没有正则、没有表达式 —— 新匹配形态
//     是 schema 级变更,不是运行时可扩展点;
//   - 零 Trigger(未声明任何条件)的策略永不命中:PRD §8.1 没有「全匹配」
//     概念,把空条件猜成 match-all 会静默放大策略作用面;
//   - 每个命中都携带机器可读 basis(AC1):哪个轴、哪些值命中;
//   - fail-safe(INV-V14-09):派生风险信号(detected 与 unknown)中没有被
//     任何「已命中策略」的 trigger.risk_signals 声明的,逐条进入
//     UnmatchedRiskSignals,绝不静默当作已覆盖;unknown 信号单独保真露出。
//     匹配层不选 assurance、不升级、不写状态 —— 它只保证「未覆盖的风险
//     不可能被看不见」;
//   - 源事实「一次策略 false-negative 会让空 required 集看似完整」在这里的
//     执行点:策略集内有任何非法策略或重复 id 都整体报错,绝不静默跳过
//     (跳过 = 假阴性)。
//
// 风险信号模型归 P1-04(internal/risk,标准库零依赖);本包只消费其派生
// 结果 risk.Result,不重复派生、不修改该包。

// 匹配输入/策略集的哨兵错误:调用方用 errors.Is 区分。
var (
	// ErrInvalidMatchInput 表示匹配输入本身非法:策略集含重复 id、风险信号
	// id 空白/带空白/重复,或输入列表含空白条目。非法输入整体报错,不做
	// 静默部分匹配(假阴性会让空 required 集看似完整)。
	ErrInvalidMatchInput = errors.New("engineeringpolicy: invalid match input")
)

// MatchConfidence 是 PRD §10 coverage 的 confidence 字段:对「匹配覆盖是否
// 完整」的确定性分类。它是事实陈述,不是 assurance 决策 —— 如何响应
// (自动升级 / Attention / limited verdict)由下游按决策锁 11/21 决定。
type MatchConfidence string

const (
	// ConfidenceNone:没有任何策略命中。此时全部已检测信号(若有)必然
	// 未覆盖。
	ConfidenceNone MatchConfidence = "none"
	// ConfidencePartial:有策略命中,但存在未覆盖的风险信号(含 unknown)。
	// INV-V14-09 的可见面:该值必须被下游显式处置。
	ConfidencePartial MatchConfidence = "partial"
	// ConfidenceFull:有策略命中,且全部已检测与 unknown 信号都被某个命中
	// 策略的 trigger 声明覆盖。
	ConfidenceFull MatchConfidence = "full"
)

// IsValid 报告该值是否为三个保留值之一。空串不合法。
func (c MatchConfidence) IsValid() bool {
	switch c {
	case ConfidenceNone, ConfidencePartial, ConfidenceFull:
		return true
	}
	return false
}

// MatchInput 是一次匹配的任务/变更/风险事实(三个输入轴,与 Trigger 三轴
// 一一对应)。Risk 消费 P1-04 的派生结果;两个列表轴是原始事实,条目去
// 首尾空白后按集合语义参与比较(重复无害,空白非法)。
type MatchInput struct {
	// Risk 是已派生的风险信号集(detected + unknown,INV-V14-09 fail-safe
	// 的输入面)。通常来自 risk.Derive。
	Risk risk.Result
	// ChangedPaths 是变更输入轴:本次候选触碰的仓库路径。
	ChangedPaths []string
	// RequirementRefs 是任务输入轴:任务引用的需求,如
	// .tally/requirements/REQ-018.md。
	RequirementRefs []string
}

// MatchBasis 是一次策略命中的机器可读依据(AC1):逐轴列出实际命中的值,
// 全部升序。三轴全空的结构不可能出现在匹配结果里 —— 那意味着无依据命中。
type MatchBasis struct {
	RiskSignals     []risk.SignalID // 命中的风险信号轴条目
	Paths           []string        // 命中的路径轴条目
	RequirementRefs []string        // 命中的需求引用轴条目
}

// Matched 报告 basis 是否至少记录了一个轴的命中。
func (b MatchBasis) Matched() bool {
	return len(b.RiskSignals) > 0 || len(b.Paths) > 0 || len(b.RequirementRefs) > 0
}

// PolicyMatch 是一条命中:策略本身加上它为何命中的 basis。
type PolicyMatch struct {
	Policy Policy
	Basis  MatchBasis
}

// PolicyMatchCoverage 是 PRD §10 的匹配覆盖投影:命中策略及其依据、纳入
// 判定的风险信号(含 unknown 保真)、未覆盖的风险信号与 confidence。
// 全部列表确定性排序;它是纯派生值,不携带、不指向任何生命周期状态(AC3)。
type PolicyMatchCoverage struct {
	// MatchedPolicies 按 Policy.ID 升序。
	MatchedPolicies []PolicyMatch
	// DetectedRiskSignals 是输入派生集中落在冻结枚举内的信号 id,升序。
	DetectedRiskSignals []risk.SignalID
	// UnknownRiskSignals 是输入派生集中冻结枚举之外的信号 id,原样保真,
	// 升序(INV-V14-09:unknown 永不丢弃、永不降级解读)。
	UnknownRiskSignals []risk.SignalID
	// UnmatchedRiskSignals 是 Detected+Unknown 中没有被任何命中策略的
	// trigger.risk_signals 声明的信号 id,升序。非空即 INV-V14-09 触发面。
	UnmatchedRiskSignals []risk.SignalID
	// Confidence 见 MatchConfidence。
	Confidence MatchConfidence
}

// HasUnmatchedRisk 报告是否存在未覆盖的风险信号。true 时下游必须显式处置
// (自动升级 / Attention / limited verdict,PRD §10),不得映射到最低
// assurance。
func (c PolicyMatchCoverage) HasUnmatchedRisk() bool {
	return len(c.UnmatchedRiskSignals) > 0
}

// Match 以确定性方式把策略集匹配到任务/变更/风险输入上。步骤:
//
//  1. 校验策略集:每个策略必须通过 Validate(非法策略整体报错而非静默
//     跳过 —— 静默跳过是策略 false-negative 的直接来源),id 不得重复;
//  2. 校验输入:风险信号 id 非空、不带首尾空白、detected/unknown 合并后
//     不重复;两个列表轴不含空白条目;
//  3. 逐策略求触发:三个轴任一命中即命中,basis 记录全部命中轴的命中值;
//  4. 命中按 Policy.ID 升序输出;覆盖与未覆盖信号升序输出;
//  5. 由覆盖情况推导 confidence。
//
// 输入切片内容绝不被修改。
func Match(policies []Policy, in MatchInput) (PolicyMatchCoverage, error) {
	// 步骤 1:策略集校验(fail-safe,不静默跳过)。
	seen := make(map[string]struct{}, len(policies))
	for i := range policies {
		p := policies[i]
		if err := p.Validate(); err != nil {
			return PolicyMatchCoverage{}, fmt.Errorf("engineeringpolicy: 策略[%d] id=%q: %w", i, p.ID, err)
		}
		if _, dup := seen[p.ID]; dup {
			return PolicyMatchCoverage{}, fmt.Errorf("%w: 策略 id %q 重复", ErrInvalidMatchInput, p.ID)
		}
		seen[p.ID] = struct{}{}
	}

	// 步骤 2:输入校验与规范化(拷贝,不改调用方切片)。
	paths, err := normalizeInputList(in.ChangedPaths, "ChangedPaths")
	if err != nil {
		return PolicyMatchCoverage{}, err
	}
	reqs, err := normalizeInputList(in.RequirementRefs, "RequirementRefs")
	if err != nil {
		return PolicyMatchCoverage{}, err
	}
	detected, unknown, err := normalizeSignals(in.Risk)
	if err != nil {
		return PolicyMatchCoverage{}, err
	}

	// 步骤 3:逐策略求触发。输入集合上做成员判定,策略触发条目去首尾空白
	// 后精确比较(与 Parse 的 trim 语义一致)。
	var matches []PolicyMatch
	for i := range policies {
		p := policies[i]
		basis := matchBasis(p.Trigger, detected, unknown, paths, reqs)
		if !basis.Matched() {
			continue
		}
		matches = append(matches, PolicyMatch{Policy: p, Basis: basis})
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].Policy.ID < matches[j].Policy.ID })

	// 步骤 4:覆盖投影。信号被覆盖 ⇔ 某个命中策略的 trigger.risk_signals
	// 声明了它(detected 与 unknown 同一规则;unknown 只有被项目策略显式
	// 声明才算覆盖,否则保持未覆盖可见)。
	covered := make(map[risk.SignalID]struct{})
	for _, m := range matches {
		for _, s := range m.Policy.Trigger.RiskSignals {
			covered[risk.SignalID(strings.TrimSpace(s))] = struct{}{}
		}
	}
	var unmatched []risk.SignalID
	for _, id := range append(append([]risk.SignalID{}, detected...), unknown...) {
		if _, ok := covered[id]; !ok {
			unmatched = append(unmatched, id)
		}
	}

	// 步骤 5:confidence 推导(纯事实分类,不做 assurance 决策)。
	var conf MatchConfidence
	switch {
	case len(matches) == 0:
		conf = ConfidenceNone
	case len(unmatched) == 0:
		conf = ConfidenceFull
	default:
		conf = ConfidencePartial
	}

	return PolicyMatchCoverage{
		MatchedPolicies:      matches,
		DetectedRiskSignals:  detected,
		UnknownRiskSignals:   unknown,
		UnmatchedRiskSignals: unmatched,
		Confidence:           conf,
	}, nil
}

// matchBasis 计算单个策略触发在各轴上的命中值。detected/unknown 已升序且
// 不重复;返回 basis 内各轴升序且去重(触发条目按去空白后比较,而 Validate
// 只拒绝原样重复,手构策略可能给出 trim 后相等的条目,此处按同一条处理)。
func matchBasis(t Trigger, detected, unknown []risk.SignalID, paths, reqs []string) MatchBasis {
	var basis MatchBasis
	for _, s := range t.RiskSignals {
		id := risk.SignalID(strings.TrimSpace(s))
		if (containsSignal(detected, id) || containsSignal(unknown, id)) && !containsSignal(basis.RiskSignals, id) {
			basis.RiskSignals = append(basis.RiskSignals, id)
		}
	}
	for _, p := range t.Paths {
		if v := strings.TrimSpace(p); containsString(paths, v) && !containsString(basis.Paths, v) {
			basis.Paths = append(basis.Paths, v)
		}
	}
	for _, r := range t.RequirementRefs {
		if v := strings.TrimSpace(r); containsString(reqs, v) && !containsString(basis.RequirementRefs, v) {
			basis.RequirementRefs = append(basis.RequirementRefs, v)
		}
	}
	sort.Slice(basis.RiskSignals, func(i, j int) bool { return basis.RiskSignals[i] < basis.RiskSignals[j] })
	sort.Strings(basis.Paths)
	sort.Strings(basis.RequirementRefs)
	return basis
}

// normalizeInputList 去首尾空白并按集合语义去重(升序拷贝);空白条目是
// 非法数据,整体报错(fail-safe,不静默忽略)。nil 输入保持 nil 语义:返回
// 空切片。调用方切片不被修改。
func normalizeInputList(vals []string, field string) ([]string, error) {
	if len(vals) == 0 {
		return nil, nil
	}
	trimmed := make([]string, 0, len(vals))
	for i, v := range vals {
		t := strings.TrimSpace(v)
		if t == "" {
			return nil, fmt.Errorf("%w: %s[%d] 为空白条目", ErrInvalidMatchInput, field, i)
		}
		trimmed = append(trimmed, t)
	}
	sort.Strings(trimmed)
	out := trimmed[:0]
	for i, t := range trimmed {
		if i == 0 || t != trimmed[i-1] {
			out = append(out, t)
		}
	}
	return out, nil
}

// normalizeSignals 校验并拷贝派生风险信号:detected/unknown 的 id 必须非空、
// 不带首尾空白,合并后不得重复(重复说明派生结果不是 risk.Derive 的产物,
// 是构造 bug,不是数据)。返回升序拷贝;空集保持 nil。
func normalizeSignals(r risk.Result) (detected, unknown []risk.SignalID, err error) {
	detected, err = signalIDs(r.Detected, "Risk.Detected")
	if err != nil {
		return nil, nil, err
	}
	unknown, err = signalIDs(r.Unknown, "Risk.Unknown")
	if err != nil {
		return nil, nil, err
	}
	seen := make(map[risk.SignalID]struct{}, len(detected)+len(unknown))
	for _, list := range [][]risk.SignalID{detected, unknown} {
		for _, id := range list {
			if _, dup := seen[id]; dup {
				return nil, nil, fmt.Errorf("%w: 风险信号 %q 重复", ErrInvalidMatchInput, string(id))
			}
			seen[id] = struct{}{}
		}
	}
	return detected, unknown, nil
}

// signalIDs 从信号投影提取升序 id 拷贝,拒绝空白/带空白 id。
func signalIDs(sigs []risk.Signal, field string) ([]risk.SignalID, error) {
	if len(sigs) == 0 {
		return nil, nil
	}
	ids := make([]risk.SignalID, 0, len(sigs))
	for i, s := range sigs {
		id := string(s.ID)
		switch {
		case strings.TrimSpace(id) == "":
			return nil, fmt.Errorf("%w: %s[%d] 信号 id 为空白", ErrInvalidMatchInput, field, i)
		case strings.TrimSpace(id) != id:
			return nil, fmt.Errorf("%w: %s[%d] 信号 id %q 带首尾空白", ErrInvalidMatchInput, field, i, id)
		}
		ids = append(ids, s.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func containsSignal(ids []risk.SignalID, id risk.SignalID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func containsString(vals []string, v string) bool {
	for _, x := range vals {
		if x == v {
			return true
		}
	}
	return false
}
