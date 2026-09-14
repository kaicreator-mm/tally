package taskcontract

import (
	"fmt"
	"regexp"
	"strings"
)

// TaskContract 静态校验器(PRD §7.4「Tally static validation」一步、§31.1
// Proposal Admission 的确定性子集):在 LLM Proposal 被固化为版本化
// TaskContract 之前,用纯函数规则拒绝结构性错误。全部检查都是确定性的:
// 同一输入永远得到同一结论,不需要也不允许任何 LLM 判断、文件系统访问或
// 网络 —— 合同写得对不对,只由合同文本决定。
//
// 本文件在 types.go Validate 的结构硬约束(必填项、profile、acceptance 与
// profile 的关系、条款唯一性、independence_required 合法性)之上新增:
//
//   - ID 语法:任务 id 与 depends_on 条目必须是 TASK-<数字>;acceptance
//     条款 id 必须是 AC-<数字>(PRD §6.1 目录布局与 §7.3 示例的固定形态);
//   - refs 语法:三类 refs 必须指向 §6.1 布局中对应类型的 .tally/ 目录下
//     的一个 Markdown 文件,允许 #fragment 稳定锚点(§6.1「stable ref」;
//     §7.3 示例 .tally/product/PRD.md#job-reliability);requirement/ADR
//     文件名分别是 REQ-<数字>.md / ADR-<数字>.md / ARCHITECTURE.md;
//   - scope 规则(types.go 把通配语义显式交给了这里的校验器):条目是仓库
//     相对路径,通配只允许整段 * 与末段 **;allowed_paths 与 forbidden_paths
//     若能匹配到同一路径,即「一边允许一边禁止」的自相矛盾合同,必须拒绝
//     —— forbidden_paths 是硬边界(PRD §7.3),允许范围不允许吞掉它;
//   - depends_on 不得自引用:这是单合同内就能确定性判定的最小 DAG 检查
//     (§31.1「DAG no cycle」);跨任务成环需要整个任务集,属于后续
//     admission 步骤,不在这里猜。
//
// Parse 不调用 ValidateStatic —— 校验器属于 admission 步骤,由调用方在
// 解析之后显式执行,保持「解析」与「准入」两个关注点分离。

// ID 与 refs 文件名的确定性文法。唯一依据是 PRD §6.1/§7.3 给出的固定形态
// (TASK-001.yaml、AC-01、REQ-018.md、ADR-012.md、ARCHITECTURE.md)。
var (
	taskIDPattern           = regexp.MustCompile(`^TASK-[0-9]+$`)
	acceptanceIDPattern     = regexp.MustCompile(`^AC-[0-9]+$`)
	requirementFilePattern  = regexp.MustCompile(`^REQ-[0-9]+\.md$`)
	architectureFilePattern = regexp.MustCompile(`^(ADR-[0-9]+\.md|ARCHITECTURE\.md)$`)
)

// 三类 refs 的目录前缀(PRD §6.1 目录布局;布局是扁平的,文件直接位于
// 类型目录之下,不允许再套子目录)。
const (
	productRefPrefix      = ".tally/product/"
	requirementRefPrefix  = ".tally/requirements/"
	architectureRefPrefix = ".tally/architecture/"
)

// ValidateStatic 在 Validate 的结构硬约束之上执行 §7.4/§31.1 要求的确定性
// 静态校验:ID 语法、refs 语法、scope 矛盾。第一次失败即返回,不做部分修复。
func (c TaskContract) ValidateStatic() error {
	// 结构硬约束(必填项、profile、acceptance 与 profile、条款唯一性、
	// independence_required 合法性)是 §7 的基础:复用 Validate,不重复实现。
	if err := c.Validate(); err != nil {
		return err
	}

	if !taskIDPattern.MatchString(c.ID) {
		return fmt.Errorf("taskcontract: id %q 不符合 TASK-<数字> 形态(PRD §6.1/§7.3)", c.ID)
	}
	for _, dep := range c.DependsOn {
		if !taskIDPattern.MatchString(dep) {
			return fmt.Errorf("taskcontract: depends_on 条目 %q 不是合法任务 id(TASK-<数字>)", dep)
		}
		if dep == c.ID {
			return fmt.Errorf("taskcontract: depends_on 含自身 %q(任务不得依赖自己)", dep)
		}
	}
	if len(c.Acceptance) > 0 {
		for _, a := range c.Acceptance {
			if !acceptanceIDPattern.MatchString(a.ID) {
				return fmt.Errorf("taskcontract: acceptance 条款 id %q 不符合 AC-<数字> 形态(PRD §7.3)", a.ID)
			}
		}
	}

	refs := []struct {
		name     string
		vals     []string
		dir      string
		baseName func(string) bool
	}{
		{"product_refs", c.ProductRefs, productRefPrefix, isProductRefFile},
		{"requirement_refs", c.RequirementRefs, requirementRefPrefix, requirementFilePattern.MatchString},
		{"architecture_refs", c.ArchitectureRefs, architectureRefPrefix, architectureFilePattern.MatchString},
	}
	for _, r := range refs {
		for _, ref := range r.vals {
			if err := checkRef(ref, r.dir, r.baseName); err != nil {
				return fmt.Errorf("taskcontract: %s 条目 %q: %w", r.name, ref, err)
			}
		}
	}

	// scope:先逐条做语法检查,再做 allowed×forbidden 重叠矛盾检查。
	// 同一 allowed 条目对多条 forbidden 都重叠时,报错点名第一对即可 ——
	// 任何一对都足以判死这份合同,顺序无关紧要(确定性)。
	allowed := make([][]string, len(c.Scope.AllowedPaths))
	for i, p := range c.Scope.AllowedPaths {
		segs, err := scopePattern(p)
		if err != nil {
			return fmt.Errorf("taskcontract: scope.allowed_paths 条目 %q: %w", p, err)
		}
		allowed[i] = segs
	}
	for _, p := range c.Scope.ForbiddenPaths {
		fb, err := scopePattern(p)
		if err != nil {
			return fmt.Errorf("taskcontract: scope.forbidden_paths 条目 %q: %w", p, err)
		}
		for i, a := range allowed {
			if scopeOverlap(a, fb) {
				return fmt.Errorf(
					"taskcontract: scope 自相矛盾:allowed_paths %q 与 forbidden_paths %q 覆盖了同一路径"+
						"(forbidden_paths 是硬边界,不能被允许范围吞掉,PRD §7.3)",
					c.Scope.AllowedPaths[i], p)
			}
		}
	}
	return nil
}

// isProductRefFile:product_refs 指向 .tally/product/ 下的任意 Markdown 文档
// (§6.1 布局只示例了 PRD.md,不把文件名钉死成一种)。
func isProductRefFile(base string) bool {
	return !strings.HasPrefix(base, ".") && strings.HasSuffix(base, ".md")
}

// checkRef 校验一条稳定 ref:仓库相对路径 + 可选 #fragment 锚点;路径必须
// 位于 dir 之下且直接是该目录下的一个文件(§6.1 布局扁平,不支持子目录)。
func checkRef(ref, dir string, baseName func(string) bool) error {
	file, fragment, hasFragment := splitRefFragment(ref)
	if hasFragment {
		if fragment == "" {
			return fmt.Errorf("# 后必须有锚点名")
		}
		if strings.ContainsAny(fragment, " \t\r\n#") {
			return fmt.Errorf("fragment %q 含空白或多余的 #", fragment)
		}
	}
	if _, err := checkPathSegments(file, false); err != nil {
		return err
	}
	if !strings.HasPrefix(file, dir) {
		return fmt.Errorf("应位于 %s 之下(PRD §6.1)", dir)
	}
	base := file[len(dir):]
	if strings.Contains(base, "/") {
		return fmt.Errorf("应直接指向 %s 下的文件,不支持子目录", dir)
	}
	if !baseName(base) {
		return fmt.Errorf("文件名 %q 不符合该类 ref 的命名约定(PRD §6.1)", base)
	}
	return nil
}

// splitRefFragment 在第一个 # 处把 ref 拆成路径与 fragment。
func splitRefFragment(ref string) (file, fragment string, hasFragment bool) {
	if i := strings.IndexByte(ref, '#'); i >= 0 {
		return ref[:i], ref[i+1:], true
	}
	return ref, "", false
}

// checkPathSegments 校验一条仓库相对路径;wildcard=true(scope 条目)时把
// 路径拆成段返回,供重叠比较用:
//   - 通用:斜杠分隔;拒绝反斜杠、盘符(:)、绝对路径、`.`/`..` 段、空段
//     与段内空白 —— ref/scope 都必须留在仓库内;
//   - wildcard=false(refs):段内不允许任何通配符,refs 指向真实文件;
//   - wildcard=true(scope):* 只能作为完整一段(匹配任意一段),** 只能
//     作为最后一段(匹配零或多段),其余通配字符(?[]{})拒绝 —— 通配
//     语义是本包显式定义的最小集,不猜 glob 方言。
func checkPathSegments(p string, wildcard bool) ([]string, error) {
	if p == "" {
		return nil, fmt.Errorf("路径为空")
	}
	if strings.ContainsAny(p, " \t\r\n") {
		return nil, fmt.Errorf("路径含空白字符")
	}
	if strings.Contains(p, `\`) {
		return nil, fmt.Errorf("路径必须用斜杠分隔,不支持反斜杠")
	}
	if strings.ContainsAny(p, "?[]{}:") {
		return nil, fmt.Errorf("路径含不支持的字符 ?[]{}:")
	}
	if strings.HasPrefix(p, "/") {
		return nil, fmt.Errorf("必须是仓库相对路径,不能以 / 开头")
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		switch {
		case s == "":
			return nil, fmt.Errorf("路径含空段(连续或首尾斜杠)")
		case s == "." || s == "..":
			return nil, fmt.Errorf("路径段 %q 不允许(必须留在仓库内)", s)
		case strings.Contains(s, "*"):
			if !wildcard {
				return nil, fmt.Errorf("refs 指向真实文件,不允许通配符")
			}
			if s != "*" && s != "**" {
				return nil, fmt.Errorf("通配符只能整段使用,段 %q 非法", s)
			}
			if s == "**" && i != len(segs)-1 {
				return nil, fmt.Errorf("** 只能作为最后一段:%q", p)
			}
		}
	}
	return segs, nil
}

// scopePattern 把一条 scope 条目变成用于重叠比较的段模式:先做语法检查,
// 再把裸条目按目录语义补上末段 **(`internal` 覆盖其整个子树,等价于
// `internal/**`)。静态校验分不清条目是文件还是目录,矛盾判定取保守方向:
// 只要可能冲突就判矛盾 —— forbidden_paths 是硬边界,宁可误拒也不放过
// 「一边允许一边禁止」的合同。
func scopePattern(p string) ([]string, error) {
	segs, err := checkPathSegments(p, true)
	if err != nil {
		return nil, err
	}
	if segs[len(segs)-1] != "**" {
		segs = append(segs, "**")
	}
	return segs, nil
}

// scopeOverlap 判断两个 scope 条目的段模式能否匹配到同一路径。
// * 匹配任意一段;末段 ** 匹配零或多段;其余段按字面相等比较。
// 纯段比较,不访问文件系统 —— 矛盾与否由合同文本决定。
func scopeOverlap(a, b []string) bool {
	for {
		switch {
		case len(a) == 0 && len(b) == 0:
			return true
		case len(a) > 0 && a[0] == "**":
			if scopeOverlap(a[1:], b) { // ** 匹配零段
				return true
			}
			if len(b) == 0 {
				return false
			}
			b = b[1:] // ** 吃掉 b 的一段后继续
		case len(b) > 0 && b[0] == "**":
			if scopeOverlap(a, b[1:]) {
				return true
			}
			if len(a) == 0 {
				return false
			}
			a = a[1:]
		case len(a) == 0 || len(b) == 0:
			return false
		case a[0] == "*" || b[0] == "*" || a[0] == b[0]:
			a, b = a[1:], b[1:]
		default:
			return false
		}
	}
}
