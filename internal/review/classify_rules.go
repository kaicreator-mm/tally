package review

import (
	"fmt"
	"sort"
	"strings"

	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// 本文件实现 PRD §19.2/§19.3 的确定性规则归账(P1-40):对 hunk 清单里的
// 每个改动条目,用封闭的确定性规则集分配恰好一个 primary review bucket,
// 并附带 semantic tags(AC、binding、风险、scope 事实)。规则无法归属的
// 条目保持 Unattributed —— 由 ReviewAgent 残差分类(可选辅助,非权威)或
// Owner 显式标注补齐,在那之前它们阻断 Approve(PRD §19.2)。
//
// 边界(P1):
//   - 只做确定性、可重算的路径规则:protected/scope/required-evidence/
//     风险路径/补充测试/文档。auth/security、public API 这类无法用路径
//     确定判定的风险维度不在规则内 —— 那是 ReviewAgent 的 semantic tag,
//     不是规则的事实(PRD §19.2:bucket 分配细节属 Review Engine
//     Implementation Spec,PRD 只要求 deterministic inventory 与
//     accounting completeness)。
//   - 本文件不调用任何 LLM,不做 Completeness Validator(归 accounting.go),
//     不改写 P1-07 的清单。

// Bucket 是 primary review bucket(PRD §19.2:每个 hunk 恰好一个;
// §19.3 的 Review 分区即 bucket 词表)。
type Bucket string

const (
	// BucketUnattributed 是残差桶:规则与显式归属都没有合理解释该条目。
	// Unattributed 非空时 Owner 不得 Approve(PRD §19.2)。确定性规则
	// 永远不主动分配它 —— 只有「没有任何规则命中」才落入。
	BucketUnattributed Bucket = "unattributed"
	// BucketScopeViolation:改动落在 forbidden_paths 内或 allowed_paths
	// 之外(§19.3 Scope Violations)。forbidden_paths 是硬边界。
	BucketScopeViolation Bucket = "scope_violation"
	// BucketProtectedEvidence:改动触及 protected verification evidence
	// (INV-VER-03:对被授权的实现 Run 不可变;§19.3 Protected Evidence)。
	BucketProtectedEvidence Bucket = "protected_evidence"
	// BucketAcceptanceEvidence:改动触及契约声明的 required evidence 文件
	// (非 protected 的 binding path)。
	BucketAcceptanceEvidence Bucket = "acceptance_evidence"
	// BucketRiskyChange:确定性风险路径规则命中(§19.3 Risky Changes 中
	// 可用路径可靠判定的子集:migration、依赖清单、CI、构建/编排配置)。
	BucketRiskyChange Bucket = "risky_change"
	// BucketSupplementaryTest:Agent 自加的补充测试文件 —— 不是契约的
	// required evidence,单独列出,不与 protected acceptance evidence
	// 混淆(§19.3 Agent-added Tests)。
	BucketSupplementaryTest Bucket = "supplementary_test"
	// BucketDocs:文档改动(§19.3 Documentation Changes)。
	BucketDocs Bucket = "docs"
	// BucketAcceptanceCode 只能由显式归属分配(ReviewAgent 残差分类或
	// Owner 标注):被归因到某条 Acceptance 的实现代码,必须携带 ac: 标签。
	// 「这段代码服务哪个 AC」无法用路径确定性判定,规则永远不分配它;
	// 未归属前该条目保持 Unattributed。
	BucketAcceptanceCode Bucket = "acceptance_code"
)

// Semantic tag 词表(PRD §19.2:一个 hunk 除唯一 primary bucket 外可带
// 多个 semantic tags)。带前缀常量后接具体 id/规则名。
const (
	// TagProtected 标记 protected evidence 命中事实。
	TagProtected = "protected"
	// TagScopeForbidden 标记 forbidden_paths 命中。
	TagScopeForbidden = "scope:forbidden"
	// TagScopeOutsideAllowed 标记 allowed_paths 之外。
	TagScopeOutsideAllowed = "scope:outside_allowed"
	// TagRiskPrefix + 规则名(如 risk:migration)标记确定性风险命中。
	TagRiskPrefix = "risk:"
	// TagACPrefix + Acceptance id(如 ac:AC-01)标记验收关联。
	TagACPrefix = "ac:"
	// TagBindingPrefix + binding id 标记证据 binding 关联。
	TagBindingPrefix = "binding:"
	// TagSupplementaryTest 标记补充测试(非契约 evidence)。
	TagSupplementaryTest = "supplementary-test"
	// TagDocs 标记文档改动。
	TagDocs = "docs"
)

// IsValidBucket 报告 b 是否是合法 bucket 值。归账结果不允许出现任何
// 词表之外的值(AC1:每个 hunk 恰好一个 primary bucket)。
func IsValidBucket(b Bucket) bool {
	switch b {
	case BucketUnattributed, BucketScopeViolation, BucketProtectedEvidence,
		BucketAcceptanceEvidence, BucketRiskyChange, BucketSupplementaryTest,
		BucketDocs, BucketAcceptanceCode:
		return true
	}
	return false
}

// PathRule 是一条命名路径规则。Pattern 语义(与 taskcontract scope 的最小
// 方言一致并做最小扩展):斜杠分段;`*` 在段内匹配该段任意前缀+后缀
// (如 *_test.go);`**` 整段使用、匹配零或多段;不含 ** 结尾的裸条目按
// 目录语义覆盖其子树。全部确定,无 glob 方言猜测。
type PathRule struct {
	// Name 是规则名,进 risk: tag(risk:<Name>)。
	Name string
	// Pattern 是路径模式。
	Pattern string
}

// EvidencePath 是一个契约 evidence 文件路径与其归属(binding/acceptance),
// 供规则把触及该路径的改动映射回验收条款(§19.3 Acceptance Coverage 的
// 「相关代码/required evidence」)。
type EvidencePath struct {
	// Path 是契约里 binding 声明的 evidence 文件路径(具体文件,无通配)。
	Path string
	// BindingID 是声明该 evidence 的 binding id。
	BindingID string
	// AcceptanceID 是该 binding 所属验收条款 id。
	AcceptanceID string
	// ExpectedHash 是 protected binding 声明的 baseline_hash(可空;仅
	// Protected 列表里的条目可能有值,供 §19.3 Protected Evidence 展示)。
	ExpectedHash string
}

// Rules 是一次 review 归账的封闭确定性规则集。Allowed 为空是构造错误:
// review 的对象永远是某份 TaskContract 约束下的 ChangeSet,合同必须声明
// 实现边界(taskcontract.Validate 同规则)。
type Rules struct {
	// Allowed 是 scope.allowed_paths 模式;至少一条。
	Allowed []string
	// Forbidden 是 scope.forbidden_paths 模式(硬边界)。
	Forbidden []string
	// Protected 是 protected evidence 路径(immutable binding)。
	Protected []EvidencePath
	// Evidence 是非 protected 的 required evidence 路径。
	Evidence []EvidencePath
	// Risky 是确定性风险路径规则。
	Risky []PathRule
	// Docs 是文档路径规则。
	Docs []PathRule
	// Test 是补充测试文件名模式规则。
	Test []PathRule
}

// DefaultRules 返回确定性的风险/文档/补充测试默认规则(小而保守的集合,
// 覆盖 PRD §19.3 中可用路径可靠判定的维度)。Allowed/Forbidden 与
// Protected/Evidence 必须来自具体合同/冻结契约 —— 用 RulesFor 构造。
func DefaultRules() Rules {
	return Rules{
		Risky: []PathRule{
			{Name: "migration", Pattern: "migrations/**"},
			{Name: "dependency", Pattern: "go.mod"},
			{Name: "dependency", Pattern: "go.sum"},
			{Name: "dependency", Pattern: "package.json"},
			{Name: "dependency", Pattern: "package-lock.json"},
			{Name: "dependency", Pattern: "yarn.lock"},
			{Name: "dependency", Pattern: "pnpm-lock.yaml"},
			{Name: "dependency", Pattern: "Cargo.toml"},
			{Name: "dependency", Pattern: "Cargo.lock"},
			{Name: "dependency", Pattern: "requirements.txt"},
			{Name: "dependency", Pattern: "pyproject.toml"},
			{Name: "dependency", Pattern: "poetry.lock"},
			{Name: "dependency", Pattern: "pom.xml"},
			{Name: "dependency", Pattern: "build.gradle"},
			{Name: "ci", Pattern: ".github/workflows/**"},
			{Name: "config", Pattern: "Dockerfile"},
			{Name: "config", Pattern: "docker-compose.yml"},
			{Name: "config", Pattern: "docker-compose.yaml"},
		},
		Docs: []PathRule{
			{Name: "docs", Pattern: "docs/**"},
			{Name: "docs", Pattern: "**/*.md"},
			{Name: "docs", Pattern: "**/*.markdown"},
			{Name: "docs", Pattern: "**/*.rst"},
		},
		Test: []PathRule{
			{Name: "test", Pattern: "**/*_test.go"},
			{Name: "test", Pattern: "**/*_test.py"},
			{Name: "test", Pattern: "**/test_*.py"},
			{Name: "test", Pattern: "**/*.test.js"},
			{Name: "test", Pattern: "**/*.test.ts"},
			{Name: "test", Pattern: "**/*.test.tsx"},
			{Name: "test", Pattern: "**/*.spec.js"},
			{Name: "test", Pattern: "**/*.spec.ts"},
		},
	}
}

// RulesFor 从 TaskContract 的 scope 与冻结 VerificationContract 推导完整
// 规则集:scope 进 Allowed/Forbidden;契约里带 repo path 的 binding 进
// Protected(Protection.mode=immutable)或 Evidence;风险/文档/测试取
// 确定性默认。f 为 nil 或推导出的规则集非法时返回 error。
func RulesFor(scope taskcontract.Scope, f *verification.Frozen) (Rules, error) {
	if f == nil {
		return Rules{}, ErrNilContract
	}
	r := DefaultRules()
	r.Allowed = append([]string(nil), scope.AllowedPaths...)
	r.Forbidden = append([]string(nil), scope.ForbiddenPaths...)

	contract := f.Contract()
	for _, acc := range contract.Acceptances {
		for _, b := range acc.EvidenceBindings {
			if strings.TrimSpace(b.Evidence.Path) == "" {
				continue // held_out(store/ref)或具名 validator 没有 repo path 可规则化
			}
			ep := EvidencePath{
				Path:         b.Evidence.Path,
				BindingID:    b.ID,
				AcceptanceID: acc.ID,
			}
			if b.Protection != nil {
				ep.ExpectedHash = b.Protection.BaselineHash
				r.Protected = append(r.Protected, ep)
			} else {
				r.Evidence = append(r.Evidence, ep)
			}
		}
	}
	sortEvidencePaths(r.Protected)
	sortEvidencePaths(r.Evidence)
	if err := r.Validate(); err != nil {
		return Rules{}, err
	}
	return r, nil
}

// sortEvidencePaths 按 (Path, BindingID) 稳定排序,保证规则集确定。
func sortEvidencePaths(ps []EvidencePath) {
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].Path != ps[j].Path {
			return ps[i].Path < ps[j].Path
		}
		return ps[i].BindingID < ps[j].BindingID
	})
}

// Validate 校验规则集自身。Allowed 必须非空(合同必须声明实现边界);
// 所有模式语法合法;evidence/protected 路径必须是具体文件(无通配);
// 同一 (Path, BindingID) 条目不得重复。
func (r Rules) Validate() error {
	if len(r.Allowed) == 0 {
		return fmt.Errorf("review: 规则缺少 allowed_paths(review 对象必须是合同边界内的 ChangeSet)")
	}
	for _, list := range []struct {
		name string
		ps   []string
	}{
		{"allowed_paths", r.Allowed},
		{"forbidden_paths", r.Forbidden},
	} {
		for _, p := range list.ps {
			if _, err := splitPatternSegments(p); err != nil {
				return fmt.Errorf("review: %s 条目 %q 非法: %v", list.name, p, err)
			}
		}
	}
	seen := map[string]bool{}
	for _, list := range []struct {
		name string
		ps   []EvidencePath
	}{
		{"protected", r.Protected},
		{"evidence", r.Evidence},
	} {
		for _, p := range list.ps {
			if _, err := splitPatternSegments(p.Path); err != nil {
				return fmt.Errorf("review: %s evidence %q 非法: %v", list.name, p.Path, err)
			}
			if strings.ContainsAny(p.Path, "*") {
				return fmt.Errorf("review: %s evidence 路径 %q 必须是具体文件,不允许通配", list.name, p.Path)
			}
			if strings.TrimSpace(p.BindingID) == "" {
				return fmt.Errorf("review: %s evidence %q 缺少 binding id", list.name, p.Path)
			}
			key := p.Path + "\x00" + p.BindingID
			if seen[key] {
				return fmt.Errorf("review: evidence 条目重复: %s (%s)", p.Path, p.BindingID)
			}
			seen[key] = true
		}
	}
	for _, list := range []struct {
		name  string
		rules []PathRule
	}{
		{"risky", r.Risky},
		{"docs", r.Docs},
		{"test", r.Test},
	} {
		for _, rule := range list.rules {
			if strings.TrimSpace(rule.Name) == "" {
				return fmt.Errorf("review: %s 规则缺少名称", list.name)
			}
			if _, err := splitPatternSegments(rule.Pattern); err != nil {
				return fmt.Errorf("review: %s 规则 %s 的模式 %q 非法: %v", list.name, rule.Name, rule.Pattern, err)
			}
		}
	}
	return nil
}

// ClassifyFile 对清单里的一个文件条目做确定性归账:返回唯一 primary
// bucket 与 semantic tags。同文件的全部 hunk 继承该结果 —— bucket 是
// 路径事实,hunk 级再细分没有确定依据。优先级(确定,先命中先得):
// protected > scope violation > required evidence > risky > 补充测试 >
// docs > unattributed。scope 事实永远计算并进 tags:protected/evidence
// 文件若在边界之外,主 bucket 保持更强的事实,scope 事实不被吞掉。
func (r Rules) ClassifyFile(e FileEntry) (Bucket, []string) {
	paths := entryPaths(e)

	var protectedHits, evidenceHits []EvidencePath
	for _, p := range r.Protected {
		if matchAnyPath(p.Path, paths) {
			protectedHits = append(protectedHits, p)
		}
	}
	for _, p := range r.Evidence {
		if matchAnyPath(p.Path, paths) {
			evidenceHits = append(evidenceHits, p)
		}
	}
	inAllowed := false
	forbidden := false
	for _, path := range paths {
		for _, pat := range r.Allowed {
			if matchPattern(pat, path) {
				inAllowed = true
			}
		}
		for _, pat := range r.Forbidden {
			if matchPattern(pat, path) {
				forbidden = true
			}
		}
	}
	outsideAllowed := !inAllowed
	var riskyHits []string
	for _, rule := range r.Risky {
		if matchAnyPattern(rule.Pattern, paths) {
			riskyHits = append(riskyHits, rule.Name)
		}
	}
	testHit := matchAnyPatternRule(r.Test, paths)
	docsHit := matchAnyPatternRule(r.Docs, paths)

	var tags []string
	add := func(ts ...string) { tags = append(tags, ts...) }
	for _, p := range protectedHits {
		add(TagProtected, TagBindingPrefix+p.BindingID)
		if p.AcceptanceID != "" {
			add(TagACPrefix + p.AcceptanceID)
		}
	}
	for _, p := range evidenceHits {
		add(TagBindingPrefix + p.BindingID)
		if p.AcceptanceID != "" {
			add(TagACPrefix + p.AcceptanceID)
		}
	}
	if forbidden {
		add(TagScopeForbidden)
	}
	if outsideAllowed {
		add(TagScopeOutsideAllowed)
	}
	for _, name := range riskyHits {
		add(TagRiskPrefix + name)
	}
	if testHit {
		add(TagSupplementaryTest)
	}
	if docsHit {
		add(TagDocs)
	}

	var bucket Bucket
	switch {
	case len(protectedHits) > 0:
		bucket = BucketProtectedEvidence
	case forbidden || outsideAllowed:
		bucket = BucketScopeViolation
	case len(evidenceHits) > 0:
		bucket = BucketAcceptanceEvidence
	case len(riskyHits) > 0:
		bucket = BucketRiskyChange
	case testHit:
		bucket = BucketSupplementaryTest
	case docsHit:
		bucket = BucketDocs
	default:
		bucket = BucketUnattributed
	}
	return bucket, dedupeStrings(tags)
}

// entryPaths 返回条目参与归账的具体路径:主键 Path,加上 rename/copy 的
// OldPath(从 protected/受限路径改名出去仍然触及了那个文件)。
func entryPaths(e FileEntry) []string {
	paths := []string{e.Path}
	if e.OldPath != "" && e.OldPath != e.Path {
		paths = append(paths, e.OldPath)
	}
	return paths
}

// matchAnyPath 报告具体路径 p 是否命中 paths 中任一路径(精确相等)。
func matchAnyPath(p string, paths []string) bool {
	for _, path := range paths {
		if path == p {
			return true
		}
	}
	return false
}

// matchAnyPattern 报告 paths 中任一路径是否命中模式。
func matchAnyPattern(pattern string, paths []string) bool {
	for _, path := range paths {
		if matchPattern(pattern, path) {
			return true
		}
	}
	return false
}

// matchAnyPatternRule 是带命名的 matchAnyPattern(命中即真)。
func matchAnyPatternRule(rules []PathRule, paths []string) bool {
	for _, rule := range rules {
		if matchAnyPattern(rule.Pattern, paths) {
			return true
		}
	}
	return false
}

// matchPattern 判断具体路径是否命中模式。裸条目按目录语义补末段 **
// (internal 覆盖 internal/**,与 taskcontract scopePattern 一致)。
func matchPattern(pattern, path string) bool {
	pat, err := splitPatternSegments(pattern)
	if err != nil {
		return false // Validate 已拦截;防御性 false,不猜
	}
	if pat[len(pat)-1] != "**" {
		pat = append(pat, "**")
	}
	return matchSegments(pat, strings.Split(path, "/"))
}

// matchSegments 段级匹配:`**` 匹配零或多段,其余段经 matchSegment。
func matchSegments(pat, segs []string) bool {
	for len(pat) > 0 {
		switch pat[0] {
		case "**":
			for i := 0; i <= len(segs); i++ {
				if matchSegments(pat[1:], segs[i:]) {
					return true
				}
			}
			return false
		default:
			if len(segs) == 0 || !matchSegment(pat[0], segs[0]) {
				return false
			}
			pat, segs = pat[1:], segs[1:]
		}
	}
	return len(segs) == 0
}

// matchSegment 单段匹配:无 * 精确相等;恰有一个 * 时按前缀+后缀匹配
// (* 可匹配段内任意 —— 含空 —— 中缀,如 *_test.go 匹配 x_test.go)。
func matchSegment(pattern, seg string) bool {
	i := strings.IndexByte(pattern, '*')
	if i < 0 {
		return pattern == seg
	}
	prefix, suffix := pattern[:i], pattern[i+1:]
	return len(seg) >= len(prefix)+len(suffix) &&
		strings.HasPrefix(seg, prefix) && strings.HasSuffix(seg, suffix)
}

// splitPatternSegments 校验并切分路径模式:斜杠分段;拒绝空白、反斜杠、
// 绝对路径、`.`/`..`/空段、?[]{}: 等字符;* 与 ** 只能整段,单段至多一个
// *;** 允许任意位置(匹配零或多段)。与 taskcontract 的 scope 方言一致
// (scope 条目 ** 仅末段),并做最小扩展以支撑 *_test.go 这类后缀规则。
func splitPatternSegments(p string) ([]string, error) {
	if p == "" {
		return nil, fmt.Errorf("路径模式为空")
	}
	if strings.ContainsAny(p, " \t\r\n") {
		return nil, fmt.Errorf("路径模式含空白字符")
	}
	if strings.Contains(p, `\`) {
		return nil, fmt.Errorf("路径模式必须用斜杠分隔,不支持反斜杠")
	}
	if strings.ContainsAny(p, "?[]{}:") {
		return nil, fmt.Errorf("路径模式含不支持的字符 ?[]{}:")
	}
	if strings.HasPrefix(p, "/") {
		return nil, fmt.Errorf("必须是仓库相对路径,不能以 / 开头")
	}
	segs := strings.Split(p, "/")
	for _, s := range segs {
		switch {
		case s == "":
			return nil, fmt.Errorf("路径模式含空段(连续或首尾斜杠)")
		case s == "." || s == "..":
			return nil, fmt.Errorf("路径段 %q 不允许(必须留在仓库内)", s)
		case s != "*" && s != "**" && strings.Count(s, "*") > 1:
			return nil, fmt.Errorf("路径段 %q 至多含一个 *", s)
		}
	}
	return segs, nil
}

// dedupeStrings 按首次出现顺序去重,保持 tag 顺序确定。
func dedupeStrings(xs []string) []string {
	seen := make(map[string]bool, len(xs))
	out := xs[:0]
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// hasTag 报告 tag 列表是否包含 want。
func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
