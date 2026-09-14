package contextpack

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"tally/internal/contract"
	"tally/internal/taskcontract"
	"tally/internal/verification"

	"gopkg.in/yaml.v3"
)

// canonicalVersion 是上下文哈希规范形式的版本前缀。规范格式将来若变化,递增它,
// 避免不同代格式的哈希空间重叠(与 taskcontract 的做法一致)。
const canonicalVersion = "contextpack/v1"

// Provenance 描述一条历史失败/知识输入与其来源 evidence binding 的关联
// (PRD §12.3:Historical Failures 与 Verified Knowledge 不得成为 held-out
// 泄漏侧信道,过滤按 binding provenance 执行)。
type Provenance struct {
	// BindingID 是该条目来源的 binding id;空表示与验证证据无关的一般上下文。
	// 可见性不由调用方声明:唯一权威是冻结契约里该 binding 的 visibility_policy
	// (实现权威/调用方不能把内容重新标注为可见)。
	BindingID string
}

// FailureInput 是一条候选历史失败条目。
type FailureInput struct {
	Ref        string // 稳定引用(如 failure event id),进入包时原样携带
	Summary    string // 面向实现权威的失败摘要(不携带 held-out verdict 细节)
	Provenance Provenance
}

// KnowledgeInput 是一条候选 Verified Knowledge 条目。
type KnowledgeInput struct {
	Ref     string
	Summary string
	Prov    Provenance
}

// Input 是 Build 的全部输入。文档/依赖任务/可见测试引用由调用方选定(P1 没有
// 相关性选择引擎,直通并记录 included);历史失败与知识输入逐条经 binding
// provenance 过滤;Requires 是编译合同(contract.Bind)产出的执行前 policy
// requires 上下文,零值 = 无命中策略/未提供。
type Input struct {
	Task         taskcontract.TaskContract
	Verification *verification.Frozen

	Budget Budget

	ProductDocs      []string
	RequirementDocs  []string
	ArchitectureDocs []string
	DependencyTasks  []string
	VisibleTests     []string

	// Requires 是冻结合同的执行前 requires 上下文(PRD §8.3 Prevention,
	// P1-07 编译器的产出)。内容只来自命中策略的公开声明,不含 held-out
	// 验证内容(由 contract.Bind 保证);Build 原样渲染为 PolicyRequires 小节
	// 并纳入 context_hash。与合同 refs 同一纪律:必现内容,不参与预算。
	Requires contract.RequiresContext

	Failures  []FailureInput
	Knowledge []KnowledgeInput
}

// Build 组装 P1 最小 TaskContextPack:
//
//  1. 校验输入(契约已冻结、Task 与冻结 VerificationContract 同属一个任务);
//  2. 对冻结契约做 redacted projection(§8.4.3,见 redact.go);
//  3. 按 binding provenance 过滤历史失败/知识输入(§12.3 / INV-VER-05);
//  4. 应用每节条目数预算并规范化排序;
//  5. 对最终(redaction/filter 之后)内容计算 context_hash(§12.3 / §8.4.3)。
//
// Build 是确定性纯函数:相同 Input 产生内容与哈希完全相同的 Pack。
func Build(in Input) (*Pack, error) {
	if in.Verification == nil {
		return nil, ErrNilContract
	}
	if err := in.Task.Validate(); err != nil {
		return nil, fmt.Errorf("contextpack: TaskContract 非法: %w", err)
	}
	if in.Task.ID != in.Verification.TaskID() {
		return nil, fmt.Errorf("contextpack: TaskContract id %s 与冻结 VerificationContract task_id %s 不一致", in.Task.ID, in.Verification.TaskID())
	}
	if err := validateFailures(in.Failures); err != nil {
		return nil, err
	}
	if err := validateKnowledge(in.Knowledge); err != nil {
		return nil, err
	}

	projections, err := RedactVerification(in.Verification)
	if err != nil {
		return nil, err
	}
	verIncluded, verExcluded := verificationSelections(projections)

	// binding 可见性唯一取自冻结契约;调用方无法重新标注。
	visibility := bindingVisibility(in.Verification)
	fKept, fInc, fExc := filterSelections(SectionHistoricalFailure, failureCandidates(in.Failures), visibility)
	kKept, kInc, kExc := filterSelections(SectionKnowledge, knowledgeCandidates(in.Knowledge), visibility)

	docs := docSections(&in)
	applyBudget(in.Budget, &docs, &fKept, &kKept)

	// policy requires 验收告知(PRD §8.3 Prevention):冻结合同派生的必现
	// 小节,不参与预算(实现权威必须在执行前知道验收条件);渲染行由
	// contract.RequiresContext.Lines() 确定,逐策略登记 included 记录。
	reqLines := in.Requires.Lines()
	reqIncluded := requiresSelections(in.Requires)

	included := concatSelections(verIncluded, fInc, kInc, docs.included, reqIncluded)
	excluded := concatSelections(verExcluded, fExc, kExc, docs.excluded)
	sortSelections(included)
	sortSelections(excluded)

	p := &Pack{
		TaskID:             in.Task.ID,
		Task:               in.Task,
		Verification:       projections,
		ProductDocs:        docs.product,
		RequirementDocs:    docs.requirement,
		ArchitectureDocs:   docs.architecture,
		DependencyTasks:    docs.dependencies,
		VisibleTests:       docs.visibleTests,
		HistoricalFailures: fKept,
		Knowledge:          kKept,
		PolicyRequires:     reqLines,
		Budget:             in.Budget,
		Included:           included,
		Excluded:           excluded,
	}
	h, err := contextHash(p)
	if err != nil {
		return nil, err
	}
	p.hash = h
	return p, nil
}

// requiresSelections 为 policy requires 验收告知小节生成 included 记录(§12.3):
// 每个命中策略一条,Ref 是策略 id(记录面实现权威可读;requires 内容本身经
// Pack.PolicyRequires 暴露)。
func requiresSelections(rc contract.RequiresContext) []Selection {
	out := make([]Selection, 0, len(rc.Items))
	for _, it := range rc.Items {
		out = append(out, Selection{
			Source: SectionPolicyRequires,
			Ref:    it.PolicyID,
			Reason: ReasonPolicyRequires,
		})
	}
	return out
}

// noteCandidate 是过滤前的统一候选条目(失败与知识共用同一过滤路径)。
type noteCandidate struct {
	ref     string
	summary string
	binding string // provenance.binding_id;空 = 一般上下文
}

func failureCandidates(in []FailureInput) []noteCandidate {
	cands := make([]noteCandidate, len(in))
	for i, f := range in {
		cands[i] = noteCandidate{ref: f.Ref, summary: f.Summary, binding: f.Provenance.BindingID}
	}
	return cands
}

func knowledgeCandidates(in []KnowledgeInput) []noteCandidate {
	cands := make([]noteCandidate, len(in))
	for i, k := range in {
		cands[i] = noteCandidate{ref: k.Ref, summary: k.Summary, binding: k.Prov.BindingID}
	}
	return cands
}

// validateFailures 校验历史失败输入:ref 与 summary 必填,否则该条目无法被
// 引用也无内容,属输入错误而非数据。
func validateFailures(in []FailureInput) error {
	for i, f := range in {
		if f.Ref == "" || f.Summary == "" {
			return fmt.Errorf("contextpack: failures[%d] 缺少 ref 或 summary", i)
		}
	}
	return nil
}

// validateKnowledge 校验知识输入:ref 与 summary 必填。
func validateKnowledge(in []KnowledgeInput) error {
	for i, k := range in {
		if k.Ref == "" || k.Summary == "" {
			return fmt.Errorf("contextpack: knowledge[%d] 缺少 ref 或 summary", i)
		}
	}
	return nil
}

// bindingVisibility 汇总冻结契约中全部 binding 的 id → visibility_policy。
func bindingVisibility(f *verification.Frozen) map[string]verification.VisibilityPolicy {
	c := f.Contract()
	vis := make(map[string]verification.VisibilityPolicy)
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			vis[b.ID] = b.VisibilityPolicy
		}
	}
	return vis
}

// provenanceVerdict 按冻结契约的 binding 可见性裁决一条候选条目:
//
//   - 无 provenance(一般上下文)→ 通过;
//   - provenance 指向 visible binding → 通过;
//   - provenance 指向 held_out binding → 剔除(INV-VER-05);
//   - provenance 指向契约中不存在的 binding → 剔除(保守默认:可见性无法证实)。
//
// 返回 (是否通过, 排除原因);通过时原因为空串。
func provenanceVerdict(bindingID string, visibility map[string]verification.VisibilityPolicy) (bool, string) {
	if bindingID == "" {
		return true, ""
	}
	switch visibility[bindingID] {
	case verification.VisibilityVisible:
		return true, ""
	case verification.VisibilityHeldOut:
		return false, ReasonHeldOutProvenance
	default:
		return false, ReasonUnresolvedProvenance
	}
}

// filterSelections 对失败/知识候选逐条执行 binding provenance 过滤并排序,
// 返回 (通过的条目, included 记录, excluded 记录)。因 held-out provenance 被
// 剔除的条目,其记录不携带 ref(记录面本身实现权威可读,不得回泄 held-out 线索)。
func filterSelections(section string, cands []noteCandidate, visibility map[string]verification.VisibilityPolicy) (kept []Note, included, excluded []Selection) {
	for _, c := range cands {
		ok, reason := provenanceVerdict(c.binding, visibility)
		if ok {
			kept = append(kept, Note{Ref: c.ref, Summary: c.summary})
			included = append(included, Selection{Source: section, Ref: c.ref, Reason: ReasonProvided})
			continue
		}
		excluded = append(excluded, Selection{Source: section, Ref: "", Reason: reason})
	}
	sort.Slice(kept, func(a, b int) bool {
		if kept[a].Ref != kept[b].Ref {
			return kept[a].Ref < kept[b].Ref
		}
		return kept[a].Summary < kept[b].Summary
	})
	return kept, included, excluded
}

// docSections 汇集调用方选定的一般上下文小节:排序、去重。预算与 included/
// excluded 记录由 applyBudget 统一生成。
type sectionLists struct {
	product, requirement, architecture, dependencies, visibleTests []string
	included, excluded                                             []Selection
}

func docSections(in *Input) sectionLists {
	var s sectionLists
	s.product = refSection(in.ProductDocs)
	s.requirement = refSection(in.RequirementDocs)
	s.architecture = refSection(in.ArchitectureDocs)
	s.dependencies = refSection(in.DependencyTasks)
	s.visibleTests = refSection(in.VisibleTests)
	return s
}

// refSection 规范化一个字符串引用小节:去重、排序。引用顺序不带语义,
// 统一排序保证同一内容得到同一哈希。
func refSection(refs []string) []string {
	seen := make(map[string]bool, len(refs))
	uniq := make([]string, 0, len(refs))
	for _, r := range refs {
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		uniq = append(uniq, r)
	}
	sort.Strings(uniq)
	return uniq
}

// applyBudget 应用每节条目数预算(§12.3 的 size budget 记录面):每个小节按
// 排序后的确定性顺序保留前 N 条,其余登记 Excluded。同时为全部保留条目生成
// included 记录。N <= 0 表示不限制。
func applyBudget(b Budget, s *sectionLists, failures *[]Note, knowledge *[]Note) {
	keep := func(section string, refs []string, included *[]Selection, excluded *[]Selection) []string {
		n := len(refs)
		if b.MaxItemsPerSection > 0 && n > b.MaxItemsPerSection {
			n = b.MaxItemsPerSection
		}
		for _, r := range refs[:n] {
			*included = append(*included, Selection{Source: section, Ref: r, Reason: ReasonProvided})
		}
		for _, r := range refs[n:] {
			*excluded = append(*excluded, Selection{Source: section, Ref: r, Reason: ReasonBudget})
		}
		return refs[:n]
	}
	keepNote := func(section string, notes []Note, included *[]Selection, excluded *[]Selection) []Note {
		n := len(notes)
		if b.MaxItemsPerSection > 0 && n > b.MaxItemsPerSection {
			n = b.MaxItemsPerSection
		}
		for _, note := range notes[:n] {
			*included = append(*included, Selection{Source: section, Ref: note.Ref, Reason: ReasonProvided})
		}
		for _, note := range notes[n:] {
			*excluded = append(*excluded, Selection{Source: section, Ref: note.Ref, Reason: ReasonBudget})
		}
		return notes[:n]
	}

	s.product = keep(SectionProductDoc, s.product, &s.included, &s.excluded)
	s.requirement = keep(SectionRequirementDoc, s.requirement, &s.included, &s.excluded)
	s.architecture = keep(SectionArchitectureDoc, s.architecture, &s.included, &s.excluded)
	s.dependencies = keep(SectionDependencyTask, s.dependencies, &s.included, &s.excluded)
	s.visibleTests = keep(SectionVisibleTest, s.visibleTests, &s.included, &s.excluded)
	*failures = keepNote(SectionHistoricalFailure, *failures, &s.included, &s.excluded)
	*knowledge = keepNote(SectionKnowledge, *knowledge, &s.included, &s.excluded)
}

func concatSelections(lists ...[]Selection) []Selection {
	var out []Selection
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

func sortSelections(s []Selection) {
	sort.Slice(s, func(a, b int) bool {
		if s[a].Source != s[b].Source {
			return s[a].Source < s[b].Source
		}
		if s[a].Ref != s[b].Ref {
			return s[a].Ref < s[b].Ref
		}
		return s[a].Reason < s[b].Reason
	})
}

// canonicalPack 是哈希专用的规范投影:固定字段顺序 + 全列表规范化(非 nil),
// 序列化为 YAML 后与前缀行一起取 sha256。哈希只覆盖实现权威实际可见的最终
// 内容 —— 此时 redaction/filter 已全部完成(§8.4.3 / §12.3),被裁剪掉的
// held-out 内容既不在规范形式里,也不影响哈希值。
type canonicalPack struct {
	TaskID             string                    `yaml:"task_id"`
	Task               taskcontract.TaskContract `yaml:"task"`
	Verification       []AcceptanceProjection    `yaml:"verification"`
	ProductDocs        []string                  `yaml:"product_docs"`
	RequirementDocs    []string                  `yaml:"requirement_docs"`
	ArchitectureDocs   []string                  `yaml:"architecture_docs"`
	DependencyTasks    []string                  `yaml:"dependency_tasks"`
	VisibleTests       []string                  `yaml:"visible_tests"`
	HistoricalFailures []Note                    `yaml:"historical_failures"`
	Knowledge          []Note                    `yaml:"knowledge"`
	PolicyRequires     []string                  `yaml:"policy_requires"`
	Budget             Budget                    `yaml:"budget"`
	Included           []Selection               `yaml:"included"`
	Excluded           []Selection               `yaml:"excluded"`
}

// contextHash 计算并返回 "sha256:<hex>" 形式的上下文哈希。
func contextHash(p *Pack) (string, error) {
	cc := canonicalPack{
		TaskID:             p.TaskID,
		Task:               p.Task,
		Verification:       p.Verification,
		ProductDocs:        orEmpty(p.ProductDocs),
		RequirementDocs:    orEmpty(p.RequirementDocs),
		ArchitectureDocs:   orEmpty(p.ArchitectureDocs),
		DependencyTasks:    orEmpty(p.DependencyTasks),
		VisibleTests:       orEmpty(p.VisibleTests),
		HistoricalFailures: orEmptyNotes(p.HistoricalFailures),
		Knowledge:          orEmptyNotes(p.Knowledge),
		PolicyRequires:     orEmpty(p.PolicyRequires),
		Budget:             p.Budget,
		Included:           orEmptySelections(p.Included),
		Excluded:           orEmptySelections(p.Excluded),
	}
	data, err := yaml.Marshal(&cc)
	if err != nil {
		return "", fmt.Errorf("contextpack: 规范化序列化: %w", err)
	}
	sum := sha256.Sum256(append([]byte(canonicalVersion+"\n"), data...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func orEmptyNotes(s []Note) []Note {
	if s == nil {
		return []Note{}
	}
	return s
}

func orEmptySelections(s []Selection) []Selection {
	if s == nil {
		return []Selection{}
	}
	return s
}
