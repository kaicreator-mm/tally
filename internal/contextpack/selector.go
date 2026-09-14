package contextpack

import (
	"fmt"
	"sort"
	"strings"

	"tally/internal/contract"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// 本文件是 P1-42 的 v1.3 ContextPack 选择路径(PRD §12 Context Engine;§36 P1
// 主线 A.8「ContextPack redacted projection」)。P1-21 的 Build 是确定性纯函数,
// 文档/测试引用由调用方给定;本文件补上「选择」这一步:
//
//   - 请求性引用:TaskContract 的 product_refs / requirement_refs /
//     architecture_refs 与 depends_on 直接入包(AC1:合同指定的 authority 文档
//     与依赖任务必须出现,不参与相关性筛选,也不参与 token 预算);
//   - 相关性选择:P1 没有语义相关性引擎,采用确定性机械规则 —— 候选文件面中
//     scope.allowed_paths 之下的 _test.go(以及 visible test binding 指定的路径)
//     进入 Existing Visible Tests;其余候选一律排除并逐条登记原因(AC2);
//   - 防泄漏:候选路径若与 held_out binding 声称的 repo path 重合(合法冻结契约
//     中不可能出现,§8.4.1 冻结校验保证 held_out 只经 store/ref 引用),纵深防御
//     性排除,且记录不携带路径(AC3);held-out 内容不得经任何来源进入实现侧包;
//   - 规模预算:MaxTotalTokens 以固定规则估算候选内容规模,按路径序确定性排除
//     超限候选(AC4)。
//
// 全程不读文件系统、不碰 Git、不执行命令;Candidate.Content 由调用方提供,仅用于
// 预算计量,绝不进入包内容。最终组装仍委托 Build —— redacted VerificationContract
// 投影(§8.4.3)、历史失败/知识的 provenance 过滤(§12.3 / INV-VER-05)、§12.3
// 记录面与 context_hash 的唯一权威实现都在 Build。

// SectionCandidate 是选择路径专属登记小节:候选文件的排除记录(非测试代码、
// 越界、命中 forbidden 等)统一登记在这里,不把未知类别的候选硬塞进文档/测试
// 小节。被选中的测试候选登记为 SectionVisibleTest,与 Pack 小节一致。
const SectionCandidate = "candidate"

// 选择路径的确定性原因文案(与 types.go 的固定文案同一纪律:同一输入产生同一
// 记录与同一哈希)。
const (
	// ReasonScopeVisibleTest — scope 内的 Go 测试文件被选为 Existing Visible Tests。
	ReasonScopeVisibleTest = "included: scope 内可见测试(_test.go)"
	// ReasonForbiddenPath — 候选路径命中 scope.forbidden_paths(forbidden 是硬边界)。
	ReasonForbiddenPath = "excluded: 候选路径命中 scope.forbidden_paths(硬边界)"
	// ReasonRefBacked — 文档候选已由 TaskContract ref 代表(ref 本身计入文档小节),
	// 不重复选取,也不在包之外另开内容通道(context_hash 必须覆盖实现权威可见的一切)。
	ReasonRefBacked = "excluded: 候选已由 TaskContract ref 代表(ref 计入文档小节),不重复选取"
	// ReasonNonTestCode — scope 内非测试代码文件:P1 包形状没有代码文件小节,
	// Repo Map / 符号级选择属后续任务;不选入,但诚实登记排除原因。
	ReasonNonTestCode = "excluded: scope 内非测试文件;P1 选择路径只携带文档 ref 与可见测试,Repo Map/符号选择属后续任务"
	// ReasonOutOfScope — 候选与任何相关性规则都不匹配。
	ReasonOutOfScope = "excluded: 候选路径不在 scope.allowed_paths 相关面内"
	// ReasonTokenBudget — 候选内容估算超过 MaxTotalTokens 剩余额度。
	ReasonTokenBudget = "excluded: 超过 token/size 预算"
)

// Candidate 是候选文件面中的一条:仓库相对路径 + 可选内容。Content 只用于 token
// 预算计量,不进入包内容;为空时按 0 计(此时预算对它没有约束力,调用方应填充
// 内容让预算有意义)。
type Candidate struct {
	Path    string
	Content string
}

// SelectInput 是 Select 的全部输入。
type SelectInput struct {
	Task         taskcontract.TaskContract
	Verification *verification.Frozen
	Candidates   []Candidate

	// Budget 原样传给 Build(每节条目数上限,§12.3 记录面,记录进 Pack)。
	Budget Budget
	// MaxTotalTokens 是候选内容的总 token 预算(§12.3 token/size budget;P1 用
	// EstimateTokens 固定规则做确定性近似)。0 表示不限制。只约束被选中的候选;
	// 合同指定的 refs 不参与预算(AC1:必须出现)。
	MaxTotalTokens int

	// Requires 是编译合同(contract.Bind)产出的执行前 policy requires 上下文
	// (PRD §8.3 Prevention)。原样传给 Build:必现内容,不参与候选预算,
	// 纳入 context_hash。零值 = 无命中策略/未提供。
	Requires contract.RequiresContext
}

// SelectResult 是选择路径的产出:组装完成的 Pack,加上候选面的逐条裁决台账。
//
// 记录面分两层,职责不同(§12.3):
//   - Pack.Included / Pack.Excluded 是权威记录面,参与 context_hash;
//   - Included / Excluded 台账是候选池的裁决审计(每个候选为什么进/不进),
//     不参与哈希;台账中选中的测试候选与 Pack 的 visible_test 记录一一对应。
type SelectResult struct {
	Pack     *Pack
	Included []Selection
	Excluded []Selection
}

// Select 组装最小真实 ContextPack(P1-42):
//
//  1. TaskContract refs → 文档/依赖任务小节(AC1,不做相关性筛选与预算);
//  2. 候选文件面逐条确定性裁决(AC2/AC3,规则见 scopeIndex.classify);
//  3. 被选中的测试候选做 token 预算(AC4,路径序贪心,超出剩余额度即排除);
//  4. 委托 Build 完成 redacted VerificationContract 投影、provenance 过滤与
//     context_hash。
//
// Select 是确定性纯函数:相同输入产生内容与哈希完全相同的结果;候选输入顺序
// 不影响结果。nil 契约、空/重复候选路径、非法合同、任务不一致都确定性报错。
func Select(in SelectInput) (*SelectResult, error) {
	if in.Verification == nil {
		return nil, ErrNilContract
	}
	seen := make(map[string]bool, len(in.Candidates))
	for i, c := range in.Candidates {
		if strings.TrimSpace(c.Path) == "" {
			return nil, fmt.Errorf("contextpack: candidates[%d] 缺少 path", i)
		}
		if seen[c.Path] {
			return nil, fmt.Errorf("contextpack: candidates 路径重复: %s", c.Path)
		}
		seen[c.Path] = true
	}

	ix := buildScopeIndex(&in.Task, in.Verification)

	// 确定性处理顺序:按路径排序;调用方给出的顺序不影响结果与哈希。
	sorted := make([]Candidate, len(in.Candidates))
	copy(sorted, in.Candidates)
	sort.Slice(sorted, func(a, b int) bool { return sorted[a].Path < sorted[b].Path })

	res := &SelectResult{}
	var tests []string
	used := 0
	for _, c := range sorted {
		section, include, reason := ix.classify(c.Path)
		if !include {
			res.Excluded = append(res.Excluded, candidateRecord(section, c.Path, reason))
			continue
		}
		if in.MaxTotalTokens > 0 {
			need := EstimateTokens(c.Content)
			if used+need > in.MaxTotalTokens {
				res.Excluded = append(res.Excluded, Selection{Source: section, Ref: c.Path, Reason: ReasonTokenBudget})
				continue
			}
			used += need
		}
		tests = append(tests, c.Path)
		res.Included = append(res.Included, Selection{Source: section, Ref: c.Path, Reason: reason})
	}

	pack, err := Build(Input{
		Task:             in.Task,
		Verification:     in.Verification,
		Budget:           in.Budget,
		ProductDocs:      in.Task.ProductRefs,
		RequirementDocs:  in.Task.RequirementRefs,
		ArchitectureDocs: in.Task.ArchitectureRefs,
		DependencyTasks:  in.Task.DependsOn,
		VisibleTests:     tests,
		Requires:         in.Requires,
	})
	if err != nil {
		return nil, err
	}
	res.Pack = pack
	sortSelections(res.Included)
	sortSelections(res.Excluded)
	return res, nil
}

// candidateRecord 构造候选裁决记录;held-out 派生的排除不携带路径 —— 记录面
// 本身也是实现权威可读的,不得借记录回泄 held-out 内容的存在性线索(与 Build
// 对失败/知识排除记录的纪律一致)。
func candidateRecord(section, path, reason string) Selection {
	if reason == ReasonHeldOutProvenance {
		return Selection{Source: section, Reason: reason}
	}
	return Selection{Source: section, Ref: path, Reason: reason}
}

// scopeIndex 预处理 Task scope、合同 refs 与冻结契约 binding,把逐候选裁决收敛
// 为纯查表 + 确定性规则。
type scopeIndex struct {
	refFiles    map[string]string // ref 的文件部分 → 所属文档小节
	allowed     []string          // scope.allowed_paths(相关性 = 其下)
	forbidden   []string          // scope.forbidden_paths(硬边界)
	heldOut     map[string]bool   // held_out binding 声称的 repo path(纵深防御)
	visibleTest map[string]bool   // visible test binding 指定的 repo path(权威指定可见测试)
}

// buildScopeIndex 从冻结契约推导 held-out / visible-test 路径集合。合法冻结契约
// 中 held_out binding 只经 store/ref 引用、没有 repo path(§8.4.1 冻结校验);
// 这里仍登记其声称的 path 作为选择路径自己的纵深防御 —— 选择层不信任「上游一定
// 已经校验过」。
func buildScopeIndex(task *taskcontract.TaskContract, f *verification.Frozen) *scopeIndex {
	ix := &scopeIndex{
		refFiles:    make(map[string]string, len(task.ProductRefs)+len(task.RequirementRefs)+len(task.ArchitectureRefs)),
		allowed:     task.Scope.AllowedPaths,
		forbidden:   task.Scope.ForbiddenPaths,
		heldOut:     map[string]bool{},
		visibleTest: map[string]bool{},
	}
	for _, ref := range task.ProductRefs {
		ix.refFiles[refFilePart(ref)] = SectionProductDoc
	}
	for _, ref := range task.RequirementRefs {
		ix.refFiles[refFilePart(ref)] = SectionRequirementDoc
	}
	for _, ref := range task.ArchitectureRefs {
		ix.refFiles[refFilePart(ref)] = SectionArchitectureDoc
	}
	c := f.Contract() // 深拷贝,读侧与冻结对象脱钩
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			switch b.VisibilityPolicy {
			case verification.VisibilityHeldOut:
				if b.Evidence.Path != "" {
					ix.heldOut[b.Evidence.Path] = true
				}
			case verification.VisibilityVisible:
				if b.Evidence.Type == verification.EvidenceTest && b.Evidence.Path != "" {
					ix.visibleTest[b.Evidence.Path] = true
				}
			}
		}
	}
	return ix
}

// classify 裁决单个候选,返回 (登记小节, 是否选中, 确定性原因)。规则与优先序:
//
//  1. held-out 派生(纵深防御;合法冻结契约到不了这一分支)→ 排除;
//  2. 已由 TaskContract ref 代表 → 排除(ref 计入文档小节,不重复选取);
//  3. visible test binding 指定路径 → 选中(可读性由验证权威的 visibility policy
//     决定;scope 管的是可改边界,不是可读边界);
//  4. forbidden → 排除(硬边界);
//  5. scope 内 _test.go → 选中;
//  6. scope 内其余文件 → 排除(P1 不携带代码文件,见 ReasonNonTestCode);
//  7. 其余 → 排除(越界)。
func (ix *scopeIndex) classify(path string) (section string, include bool, reason string) {
	if ix.heldOut[path] {
		return SectionVerification, false, ReasonHeldOutProvenance
	}
	if sec, ok := ix.refFiles[path]; ok {
		return sec, false, ReasonRefBacked
	}
	if ix.visibleTest[path] {
		return SectionVisibleTest, true, ReasonVisibleBinding
	}
	if matchesScopeRoot(ix.forbidden, path) {
		return SectionCandidate, false, ReasonForbiddenPath
	}
	if isTestPath(path) && matchesScopeRoot(ix.allowed, path) {
		return SectionVisibleTest, true, ReasonScopeVisibleTest
	}
	if matchesScopeRoot(ix.allowed, path) {
		return SectionCandidate, false, ReasonNonTestCode
	}
	return SectionCandidate, false, ReasonOutOfScope
}

// refFilePart 取 ref 的文件部分(第一个 # 之前的路径),用于候选支撑文件匹配。
func refFilePart(ref string) string {
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		return ref[:i]
	}
	return ref
}

// matchesScopeRoot 报告 path 是否等于任一根或位于任一根之下(目录式前缀匹配)。
// 通配模式不做展开:P1 语义下通配条目只可能按字面精确命中,命不中的候选走排除
// 路径 —— 宁可少选,不猜语义(通配展开归 scope 校验/执行方,见 taskcontract.Scope
// 注释)。
func matchesScopeRoot(roots []string, path string) bool {
	for _, root := range roots {
		root = strings.TrimSuffix(root, "/")
		if root == "" {
			continue
		}
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

// isTestPath:P1 栈为 Go,候选测试识别用 _test.go 后缀;其他语言的测试识别归
// 后续任务。binding 指定的可见测试不受此限制(见 classify 第 3 条)。
func isTestPath(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}

// EstimateTokens 以「每 4 字节 ≈ 1 token」的固定规则估算内容规模(§12.3 的
// token/size budget 在 P1 的确定性近似;不引入分词器依赖)。空内容为 0。
func EstimateTokens(s string) int {
	return (len(s) + 3) / 4
}
