// 本文件是 P1-07 的合同编译器:把 P1-01 的 ContractSnapshot、P1-05 的策略匹配
// 覆盖与 P1-06 的保证选择聚合为一份「已编译合同」(PRD §7.2 的
// matched_policies[] / policy_match_basis[] / required_evidence[] /
// assurance_requirement 投影),并把策略 requires[](PRD §8.3)同时编译成两个面:
//
//   - Prevention(执行前):RequiresContext —— 告知实现权威「最终会按这些条件
//     验收」的人类可读上下文。内容只来自命中策略本身(id/enforcement/rationale/
//     requires),绝不携带冻结 VerificationContract 的 held_out binding 的任何
//     定位或内容(INV-V14-07;held-out 的 redacted 投影归 contextpack)。
//   - Assurance(执行后):GateKeys() —— 命中 gate 策略的 gate_condition.all_of
//     (engineeringpolicy.Policy.GateKeys),与 Clauses() 的冻结证据绑定一起构成
//     执行后门禁/证据视图。
//
// 不变量:
//
//   - 编译器只消费既有 seam(engineeringpolicy.Match / assurance.Select /
//     Policy.GateKeys),不重新实现匹配或保证选择;不新增任何状态权威 ——
//     Compiled 是 snapshot(钉住的修订)+ 派生事实的纯投影;
//   - 编译不改动 snapshot(P1-01 snapshot 不可变;Compiled 只读引用它);
//   - 冻结先于执行(INV-V14-02):Compiled 是交给执行的冻结输入,访问器全部
//     返回深拷贝,实现权威/Repair 无法经引用改动它;修订 = 对新输入重新 Bind;
//   - fail-safe(PRD §13.4 / INV-V14-09):未覆盖与 unknown 风险信号经
//     Coverage() 原样可见,并把全部条款的必需保证档上调到 L3
//     (assurance.ReasonUnmatchedRiskFailsafe),绝不静默选择最弱保证;
//   - 确定性:同一(snapshot, 策略集, 风险事实)输入 ⇒ 内容与 Hash 完全相同的
//     Compiled,与输入切片顺序无关。
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tally/internal/assurance"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/verification"

	"gopkg.in/yaml.v3"
)

// 编译器哨兵错误:调用方用 errors.Is 判类。策略匹配错误原样透传
// engineeringpolicy 的哨兵(ErrInvalidMatchInput / ErrInvalidPolicy)。
var (
	// ErrNilSnapshot:Bind 收到 nil 的 ContractSnapshot。没有钉住修订的聚合
	// 就没有可编译的合同。
	ErrNilSnapshot = errors.New("contract: ContractSnapshot 为 nil")
	// ErrUnpinnedPolicy:命中策略缺少内容修订(EngineeringPolicy.Revision 为
	// 空)。Compiled 要把每条命中策略钉到具体内容修订;无法钉住的策略拒绝
	// 绑定,而不是带着空 pin 进合同(只有 engineeringpolicy.Parse 会填充)。
	ErrUnpinnedPolicy = errors.New("contract: 策略缺少内容修订 pin")
)

// bandUnmatchedRisk 刻意取 assurance.RiskBand 三个枚举值之外的值:存在未覆盖
// 风险信号时,用「无法归档」的输入进入 assurance.Select,由其 normalizeRisk
// 按 unknown 处理(INV-V14-09 fail-safe:记 ReasonUnmatchedRiskFailsafe,上调
// L3,绝不落到 L0/L1)。选档语义归 P1-06 seam,这里只负责把「有未覆盖信号」
// 这一事实翻译成它的 fail-safe 输入形态。
const bandUnmatchedRisk = assurance.RiskBand("unmatched_risk_signals_present")

// EvidenceRequirement 是编译后合同的一条 required evidence(PRD §7.2):从冻结
// VerificationContract 的 evidence binding 原样编译。这是验证权威面的完整视图
// —— 逐条保留全部冻结绑定,含 held_out binding 的 store/ref;任何丢弃或字段
// 裁剪都会让 required evidence 弱于冻结 VerificationContract(AC2,禁止)。
// 实现权威可见的 redacted 投影是另一条路径(contextpack.RedactVerification)。
type EvidenceRequirement struct {
	AcceptanceID  string                        // 所属验收条款(冻结契约的 acceptance id)
	BindingID     string                        // 冻结 binding id
	AuthoringMode verification.AuthoringMode    // INV-VER-01 authoring provenance
	Visibility    verification.VisibilityPolicy // visible | held_out(INV-VER-05)
	Evidence      verification.EvidenceRef      // 完整定位(visible 为 repo 定位,held_out 为 store/ref)
	Protected     bool                          // 声明了 INV-VER-03 immutable protection
}

// ClauseRequirement 是一条验收条款的执行后编译视图:冻结的证据绑定(全量,
// 不丢弃)加该条款的必需保证档(assurance.Selection,P1-06 seam 的输出,
// required 与 achieved 严格区分)。
type ClauseRequirement struct {
	AcceptanceID string
	// Assurance 是该条款的必需保证档及其决策链(default floor / risk uplift /
	// unmatched failsafe)。achieved 归执行后的事实推导(assurance.Achieved),
	// 不属于编译期。
	Assurance assurance.Selection
	// Evidence 按 BindingID 升序。
	Evidence []EvidenceRequirement
}

// GateRequirement 是冻结 VerificationContract 的一条 required gate(如 build):
// required 标志的 gate 才进入 required evidence 视图;Command 为冻结命令规格。
type GateRequirement struct {
	Name    string
	Command *verification.CommandSpec
}

// RequiresItem 是执行前 requires 上下文里的一条(一个命中策略)。Keys 是该
// 策略 requires[] 的升序拷贝 —— 与执行后门禁(GateKeys / gate_condition)
// 同源(PRD §8.3 双重用途),Prevention 面不裁剪任何 requires 键。
type RequiresItem struct {
	PolicyID    string
	Enforcement engineeringpolicy.Enforcement
	Rationale   string
	Keys        []string
}

// RequiresContext 是执行前(PRD §8.3 Prevention)交给实现权威的 requires
// 上下文:每个命中策略一条,按 PolicyID 升序。内容只来自策略本身,不携带
// 冻结验证契约的 held_out 定位或内容(INV-V14-07)。
type RequiresContext struct {
	Items []RequiresItem
}

// Lines 渲染人类可读的通知行(确定性):告知实现权威每条命中策略的验收条件。
// 空上下文返回 nil。
func (rc RequiresContext) Lines() []string {
	if len(rc.Items) == 0 {
		return nil
	}
	out := make([]string, 0, len(rc.Items))
	for _, it := range rc.Items {
		line := fmt.Sprintf("policy %s (%s): 最终会按这些条件验收: %s", it.PolicyID, it.Enforcement, strings.Join(it.Keys, ", "))
		if it.Rationale != "" {
			line += " — 理由: " + it.Rationale
		}
		out = append(out, line)
	}
	return out
}

// clone 深拷贝上下文(访问器防改动纪律)。
func (rc RequiresContext) clone() RequiresContext {
	items := make([]RequiresItem, len(rc.Items))
	for i, it := range rc.Items {
		it.Keys = cloneStrings(it.Keys)
		items[i] = it
	}
	return RequiresContext{Items: items}
}

// Compiled 是 v1.4 语义的已编译合同:一份钉住修订的 ContractSnapshot 加上
// 与之绑定的策略匹配覆盖、必需证据/门禁与每条款必需保证档。它是冻结先于
// 执行(INV-V14-02)的冻结输入:全部访问器返回拷贝或只读视图,没有修改路径;
// 修订 = 对新输入重新 Bind。
type Compiled struct {
	snap     *Snapshot
	coverage engineeringpolicy.PolicyMatchCoverage
	clauses  []ClauseRequirement
	gates    []GateRequirement
	requires RequiresContext
	hash     string
}

// Snapshot 返回被编译的 ContractSnapshot(只读;P1-01 的不可变聚合)。
func (c *Compiled) Snapshot() *Snapshot { return c.snap }

// Pins 返回快照钉住的一对源合同修订 pin(编译器不改写 pin,AC1)。
func (c *Compiled) Pins() Pins { return c.snap.Pins() }

// TaskID 返回聚合双方的共同任务 id。
func (c *Compiled) TaskID() string { return c.snap.TaskID() }

// Hash 返回已编译合同的身份哈希(sha256:<hex>):snapshot 身份 + 策略匹配
// 覆盖 + 条款保证选择 + 冻结门禁 + requires 上下文的确定性纯函数。同一输入
// 重复编译得到同一哈希;任一源合同换版、策略换版或风险事实变化都会改变它。
func (c *Compiled) Hash() string { return c.hash }

// Coverage 返回策略匹配覆盖(P1-05 seam 的原样投影):命中策略及其 match
// basis(即 PRD §7.2 的 matched_policies[] / policy_match_basis[]),以及
// detected / unknown / unmatched 风险信号与 confidence(INV-V14-09 的可见面)。
func (c *Compiled) Coverage() engineeringpolicy.PolicyMatchCoverage { return c.coverage }

// Clauses 返回每条验收条款的必需证据与必需保证档(深拷贝;按 acceptance id
// 升序)。required_evidence[] 逐条保留冻结 VerificationContract 的全部绑定,
// 编译不得使其弱于冻结契约(AC2)。
func (c *Compiled) Clauses() []ClauseRequirement {
	out := make([]ClauseRequirement, len(c.clauses))
	for i, cl := range c.clauses {
		cl.Assurance.Basis.Reasons = cloneReasons(cl.Assurance.Basis.Reasons)
		cl.Evidence = cloneEvidence(cl.Evidence)
		out[i] = cl
	}
	return out
}

// FrozenGates 返回冻结 VerificationContract 的 required gates(深拷贝,按
// 名称升序)。
func (c *Compiled) FrozenGates() []GateRequirement {
	out := make([]GateRequirement, len(c.gates))
	for i, g := range c.gates {
		out[i] = GateRequirement{Name: g.Name, Command: cloneCommandSpec(g.Command)}
	}
	return out
}

// Requires 返回执行前 requires 上下文(深拷贝;PRD §8.3 Prevention 面)。
func (c *Compiled) Requires() RequiresContext { return c.requires.clone() }

// GateKeys 返回执行后门禁视图:全部命中 gate 策略的 gate_condition.all_of
// 键的并集,升序去重。键经 engineeringpolicy.Policy.GateKeys(P1-02 seam)取
// 出;guidance 策略没有阻断权,不进入门禁视图(其 requires 只出现在执行前
// RequiresContext)。这就是 requires[] 的 Assurance 面:与 Prevention 面同源。
func (c *Compiled) GateKeys() []string {
	var out []string
	for _, m := range c.coverage.MatchedPolicies {
		if m.Policy.Enforcement != engineeringpolicy.EnforcementGate {
			continue
		}
		out = append(out, m.Policy.GateKeys()...)
	}
	return sortDedupe(out)
}

// Bind 把一份 ContractSnapshot 与工程策略集、任务级派生风险事实编译成一份
// 已编译合同。步骤:
//
//  1. 校验:.snapshot 非 nil;每条策略的内容修订 pin 已填充(否则拒绝绑定);
//  2. 策略匹配(P1-05 seam):匹配输入的任务轴(requirement_refs)直接取自
//     snapshot 钉住的 TaskContract —— 与 pin 一致,不由调用方断言;变更路径轴
//     在执行前不存在(INV-V14-02),保持为空。命中路径触发的策略属执行期覆盖
//     (P1-05 在候选变更上的复用),不在本编译面;
//  3. 风险档翻译:存在未覆盖信号 → fail-safe 档( assurance.Select 记
//     ReasonUnmatchedRiskFailsafe);否则任一信号存在 → elevated(Lock #21
//     上调);空集 → minimal(维持 L1 默认地板);
//  4. 逐验收条款保证选择(P1-06 seam)+ 冻结证据绑定全量编译(AC2);
//  5. 编译执行前 requires 上下文(PRD §8.3 Prevention)与身份哈希。
func Bind(snap *Snapshot, policies []engineeringpolicy.Policy, riskFacts risk.Result) (*Compiled, error) {
	if snap == nil {
		return nil, ErrNilSnapshot
	}
	for i := range policies {
		if policies[i].Revision == "" {
			return nil, fmt.Errorf("%w: policy %q 为空(只有 engineeringpolicy.Parse 会填充内容修订)", ErrUnpinnedPolicy, policies[i].ID)
		}
	}

	in := engineeringpolicy.MatchInput{
		Risk:            riskFacts,
		RequirementRefs: snap.Task().RequirementRefs,
	}
	cov, err := engineeringpolicy.Match(policies, in)
	if err != nil {
		return nil, fmt.Errorf("contract: 策略匹配: %w", err)
	}
	// Compiled 是冻结输入:匹配结果深拷贝后再入合同,不与调用方策略切片共享
	// 底层数组(Match 本身只做值拷贝,触发列表仍指向调用方数组)。
	cov = cloneCoverage(cov)

	band := riskBand(cov, riskFacts)

	// 冻结验证契约是必需证据的唯一权威:深拷贝后逐条款编译,编译路径不可能
	// 改写它(Frozen 无修改路径)。
	fc := snap.Verification().Contract()
	clauses := make([]ClauseRequirement, 0, len(fc.Acceptances))
	for _, acc := range fc.Acceptances {
		sel, err := assurance.Select(assurance.ClauseInput{ClauseID: acc.ID, Risk: band})
		if err != nil {
			return nil, fmt.Errorf("contract: acceptance %s 保证选择: %w", acc.ID, err)
		}
		clauses = append(clauses, ClauseRequirement{
			AcceptanceID: acc.ID,
			Assurance:    sel,
			Evidence:     evidenceRequirements(acc),
		})
	}
	sort.Slice(clauses, func(i, j int) bool { return clauses[i].AcceptanceID < clauses[j].AcceptanceID })

	c := &Compiled{
		snap:     snap,
		coverage: cov,
		clauses:  clauses,
		gates:    frozenGates(fc.RequiredGates),
		requires: requiresContext(cov),
	}
	h, err := compiledHash(c)
	if err != nil {
		return nil, err
	}
	c.hash = h
	return c, nil
}

// riskBand 把 P1-05 覆盖投影 + P1-04 风险集确定性地翻译成 P1-06 的风险档
// (PRD §13.3 的 P1 最小映射:风险信号 + unmatched 信号;任务类型/历史需求的
// 信号采集属后续任务,经同一入口接入)。unknown 但已被命中策略声明的信号不算
// 未覆盖:项目策略显式声明过它的处理方式,按常规风险上调(AboveMinimal → L3)。
func riskBand(cov engineeringpolicy.PolicyMatchCoverage, r risk.Result) assurance.RiskBand {
	switch {
	case cov.HasUnmatchedRisk():
		return bandUnmatchedRisk
	case r.AboveMinimal():
		return assurance.RiskElevated
	default:
		return assurance.RiskMinimal
	}
}

// evidenceRequirements 编译一条验收条款的全部冻结绑定(AC2:全量保留,含
// held_out 的 store/ref —— 这是验证权威面)。输出按 BindingID 升序。
func evidenceRequirements(acc verification.Acceptance) []EvidenceRequirement {
	out := make([]EvidenceRequirement, 0, len(acc.EvidenceBindings))
	for _, b := range acc.EvidenceBindings {
		out = append(out, EvidenceRequirement{
			AcceptanceID:  acc.ID,
			BindingID:     b.ID,
			AuthoringMode: b.AuthoringMode,
			Visibility:    b.VisibilityPolicy,
			Evidence:      b.Evidence, // 值类型(纯字符串字段),按值复制即与冻结内容脱钩
			Protected:     b.Protection != nil,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].BindingID < out[j].BindingID })
	return out
}

// frozenGates 编译冻结契约的 required gates(required=false 的 gate 不是必需
// 证据,不进入视图)。map 迭代无序,按名称升序输出。
func frozenGates(gates map[string]verification.Gate) []GateRequirement {
	out := make([]GateRequirement, 0, len(gates))
	for name, g := range gates {
		if !g.Required {
			continue
		}
		out = append(out, GateRequirement{Name: name, Command: cloneCommandSpec(g.Command)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// requiresContext 把命中策略编译成执行前 requires 上下文(每策略一条,按
// PolicyID 升序;Keys 升序去重)。只取策略自身的公开内容,不含任何冻结验证
// 契约的定位信息(INV-V14-07)。
func requiresContext(cov engineeringpolicy.PolicyMatchCoverage) RequiresContext {
	items := make([]RequiresItem, 0, len(cov.MatchedPolicies))
	for _, m := range cov.MatchedPolicies {
		items = append(items, RequiresItem{
			PolicyID:    m.Policy.ID,
			Enforcement: m.Policy.Enforcement,
			Rationale:   m.Policy.Rationale,
			Keys:        sortDedupe(m.Policy.Requires),
		})
	}
	return RequiresContext{Items: items}
}

// compiledCanonicalVersion 是已编译合同规范文本的版本前缀,与 contract/v1、
// contextpack/v1 同一纪律:规范格式变化时递增,避免哈希空间重叠。
const compiledCanonicalVersion = "contract/compiled/v1"

// canonicalCompiled 是哈希专用的规范投影:已编译合同对外暴露的全部派生面。
// snapshot 身份覆盖两份源合同内容;coverage/clauses/gates/requires 覆盖策略、
// 风险与保证绑定的全部实质内容。列表均已确定性排序。
type canonicalCompiled struct {
	Version      string                                `yaml:"version"`
	SnapshotHash string                                `yaml:"snapshot_hash"`
	Coverage     engineeringpolicy.PolicyMatchCoverage `yaml:"policy_match_coverage"`
	Clauses      []ClauseRequirement                   `yaml:"clause_requirements"`
	Gates        []GateRequirement                     `yaml:"frozen_gates"`
	Requires     RequiresContext                       `yaml:"requires"`
}

// compiledHash 计算已编译合同身份哈希,形如 "sha256:<hex>"。
func compiledHash(c *Compiled) (string, error) {
	cc := canonicalCompiled{
		Version:      compiledCanonicalVersion,
		SnapshotHash: c.snap.Hash(),
		Coverage:     c.coverage,
		Clauses:      c.clauses,
		Gates:        c.gates,
		Requires:     c.requires,
	}
	data, err := yaml.Marshal(&cc)
	if err != nil {
		return "", fmt.Errorf("contract: 已编译合同规范化序列化: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// cloneCoverage 深拷贝 P1-05 匹配结果:Compiled 是冻结输入,策略与依据里的
// 切片不得与调用方输入共享底层数组(engineeringpolicy.Match 只做值拷贝,
// 触发/requires 列表仍指向调用方数组)。
func cloneCoverage(cov engineeringpolicy.PolicyMatchCoverage) engineeringpolicy.PolicyMatchCoverage {
	out := cov
	out.MatchedPolicies = make([]engineeringpolicy.PolicyMatch, len(cov.MatchedPolicies))
	for i, m := range cov.MatchedPolicies {
		m.Policy = clonePolicy(m.Policy)
		m.Basis.RiskSignals = cloneSignalIDs(m.Basis.RiskSignals)
		m.Basis.Paths = cloneStrings(m.Basis.Paths)
		m.Basis.RequirementRefs = cloneStrings(m.Basis.RequirementRefs)
		out.MatchedPolicies[i] = m
	}
	out.DetectedRiskSignals = cloneSignalIDs(cov.DetectedRiskSignals)
	out.UnknownRiskSignals = cloneSignalIDs(cov.UnknownRiskSignals)
	out.UnmatchedRiskSignals = cloneSignalIDs(cov.UnmatchedRiskSignals)
	return out
}

// clonePolicy 深拷贝策略的全部切片字段。
func clonePolicy(p engineeringpolicy.Policy) engineeringpolicy.Policy {
	p.Trigger.RiskSignals = cloneStrings(p.Trigger.RiskSignals)
	p.Trigger.Paths = cloneStrings(p.Trigger.Paths)
	p.Trigger.RequirementRefs = cloneStrings(p.Trigger.RequirementRefs)
	p.Requires = cloneStrings(p.Requires)
	p.GateCondition.AllOf = cloneStrings(p.GateCondition.AllOf)
	p.BundleTags = cloneStrings(p.BundleTags)
	return p
}

// cloneSignalIDs 深拷贝风险信号 id 切片;nil 保持 nil。
func cloneSignalIDs(in []risk.SignalID) []risk.SignalID {
	if in == nil {
		return nil
	}
	out := make([]risk.SignalID, len(in))
	copy(out, in)
	return out
}

// cloneEvidence 深拷贝证据编译条目切片。
func cloneEvidence(in []EvidenceRequirement) []EvidenceRequirement {
	out := make([]EvidenceRequirement, len(in))
	copy(out, in)
	return out
}

// cloneReasons 深拷贝保证决策理由切片。
func cloneReasons(in []assurance.Reason) []assurance.Reason {
	out := make([]assurance.Reason, len(in))
	copy(out, in)
	return out
}

// cloneCommandSpec 深拷贝冻结命令规格(argv/env 切片与指针全部脱钩),
// 编译产物不与调用方或冻结视图共享可变引用。
func cloneCommandSpec(s *verification.CommandSpec) *verification.CommandSpec {
	if s == nil {
		return nil
	}
	cp := *s
	if s.Argv != nil {
		cp.Argv = append([]string(nil), s.Argv...)
	}
	if s.Env != nil {
		cp.Env = append([]string(nil), s.Env...)
	}
	return &cp
}

// sortDedupe 返回升序去重拷贝;条目先去首尾空白,空白条目丢弃。nil 输入返回
// nil。用于 requires 键与门禁键的规范化(Validate 已拒绝原样重复,这里防御
// 「trim 后相等」的手构输入)。
func sortDedupe(vals []string) []string {
	if len(vals) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(vals))
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			trimmed = append(trimmed, t)
		}
	}
	sort.Strings(trimmed)
	out := trimmed[:0]
	for i, t := range trimmed {
		if i == 0 || t != trimmed[i-1] {
			out = append(out, t)
		}
	}
	return out
}
