package taskcontract

import (
	"fmt"
	"strings"
)

// TaskContract 是 v1.3 的正式工程执行单元(PRD §7):Task 不是 Idea 的直接翻译,
// 而是 Idea 经 Productize、Design、Plan 后得到的、可被 Agent 执行且可被独立验证的
// 合同。它只定义 WHAT / WHY / SCOPE / ACCEPTANCE / DEPENDENCY / RISK / IMPACT。
//
// TaskContract 是 Git 里的版本化 desired 制品(.tally/tasks/TASK-*.yaml,
// PRD §6.1 Git-first Artifact / INV-ART-01);本包只建模内容本身,不碰 git 写入。
//
// 刻意不存在的东西(PRD §7.4):max_iterations、token/wall budget、network、
// shell、credentials、retry、runtime/model escalation、stop conditions 一律
// 不属于 TaskContract,由 ExecutionPolicy 管理。本结构体不声明任何这类字段;
// 解析层(parse.go)会在 YAML 出现这些字段时直接报错,而不是默默吞掉。

// Task profiles(PRD §7.2)。intake policy 给最低等级,LLM 只能升级;
// 本包只认名字,治理深度升级/降级是 policy 层的事。
const (
	ProfileLight    = "light"
	ProfileStandard = "standard"
	ProfileStrict   = "strict"
)

// IsValidProfile 报告 profile 是否是 PRD §7.2 定义的三个名字之一。
func IsValidProfile(p string) bool {
	switch p {
	case ProfileLight, ProfileStandard, ProfileStrict:
		return true
	}
	return false
}

// IndependenceLevel 是 Acceptance Clause 的独立性要求(PRD §7.3 / §8.5):
// 它属于条款,不属于单条 evidence binding;是累积等级 none < source < information。
//
//   - none:不要求独立性;
//   - source:至少一条 applicable evidence 的 effective_source_independence = true;
//   - information:同时满足 source 与 information independence。
type IndependenceLevel string

const (
	IndependenceNone        IndependenceLevel = "none"
	IndependenceSource      IndependenceLevel = "source"
	IndependenceInformation IndependenceLevel = "information"
)

// IsValid 报告该值是否是三个合法等级之一。空串不算合法:用 Independence()
// 取「未声明 → none」的默认,而不是让非法值混进 coverage 计算。
func (l IndependenceLevel) IsValid() bool {
	switch l {
	case IndependenceNone, IndependenceSource, IndependenceInformation:
		return true
	}
	return false
}

// Rank 返回累积等级的序(none=0 < source=1 < information=2),供「某条证据
// 是否满足某条款要求」这类比较用。非法值返回 -1,排序时永远不会被误当合格。
func (l IndependenceLevel) Rank() int {
	switch l {
	case IndependenceNone:
		return 0
	case IndependenceSource:
		return 1
	case IndependenceInformation:
		return 2
	}
	return -1
}

// Satisfies 报告实际达到的等级 got 是否满足条款要求 required。
// 非法值一律不满足(保守默认,不猜)。
func (required IndependenceLevel) Satisfies(got IndependenceLevel) bool {
	if !required.IsValid() || !got.IsValid() {
		return false
	}
	return got.Rank() >= required.Rank()
}

// Scope 是任务的实现边界(PRD §7.3):allowed_paths 之内才可改,
// forbidden_paths 是硬边界。路径用仓库内斜杠风格,通配语义(** 等)
// 由 scope 校验/执行方负责,本包只存储与规范化条目。
type Scope struct {
	AllowedPaths   []string `yaml:"allowed_paths"`
	ForbiddenPaths []string `yaml:"forbidden_paths"`
}

// AcceptanceClause 是一条验收条款。independence_required 是它的属性(AC 语义
// 所在),不是任何单条证据的属性;未声明时按 none 处理(见 Independence)。
type AcceptanceClause struct {
	ID                   string            `yaml:"id"`        // 如 AC-01,合同内唯一
	Statement            string            `yaml:"statement"` // 可判定的验收陈述
	IndependenceRequired IndependenceLevel `yaml:"independence_required"`
}

// Independence 返回生效的独立性要求:未声明(空)时为 none。
// 默认取最弱等级是保守方向 —— 独立性要求只能被显式声明,不能被默认赋予。
func (c AcceptanceClause) Independence() IndependenceLevel {
	if c.IndependenceRequired == "" {
		return IndependenceNone
	}
	return c.IndependenceRequired
}

// TaskContract 是一份解析后的任务合同。字段与 PRD §7.3 示例一一对应;
// Hash 是由内容推导的稳定修订哈希(见 hash.go),解析时填充,不来自 YAML。
type TaskContract struct {
	ID        string `yaml:"id"`        // 如 TASK-023,在项目任务集内唯一
	Profile   string `yaml:"profile"`   // light / standard / strict(PRD §7.2)
	Title     string `yaml:"title"`     // 人类可读标题,可空
	Objective string `yaml:"objective"` // 要达成什么(WHY/WHAT);任何 profile 都必填

	ProductRefs      []string `yaml:"product_refs"`      // 如 .tally/product/PRD.md#...
	RequirementRefs  []string `yaml:"requirement_refs"`  // 如 .tally/requirements/REQ-018.md
	ArchitectureRefs []string `yaml:"architecture_refs"` // 如 .tally/architecture/ADR-012.md

	DependsOn []string `yaml:"depends_on"` // 前置任务 id,如 TASK-019

	Scope Scope `yaml:"scope"`

	Acceptance []AcceptanceClause `yaml:"acceptance"` // light 可省略;standard/strict 必须非空

	// documentation_impact 只声明影响面(PRD §6.3);能否过 Integration Gate
	// 由 Project Integration Policy 决定,TaskContract 不能自行放宽它。
	DocumentationImpact []string `yaml:"documentation_impact"`

	// 非 YAML 字段:内容修订哈希(Parse 时填充)。
	Hash string `yaml:"-"`
}

// Independence 报告该合同里 id 对应条款的生效独立性要求;未找到该 id 时返回
// (none, false)。覆盖计算按条款 id 查询,不允许按下标猜。
func (c TaskContract) Independence(clauseID string) (IndependenceLevel, bool) {
	for _, a := range c.Acceptance {
		if a.ID == clauseID {
			return a.Independence(), true
		}
	}
	return IndependenceNone, false
}

// Validate 校验合同的硬约束。Parse 时调用;调用方覆盖字段后再调一次。
// 规则全部来自 PRD §7,不确定的不猜:
//
//   - id / profile / objective 必填;profile 必须是三个名字之一;
//   - scope.allowed_paths 必须非空(合同必须声明实现边界),条目不得为空白;
//   - acceptance:standard/strict 必须至少一条;light 可省略(PRD §7.2);
//   - 每条 acceptance 的 id/statement 必填、id 唯一;independence_required
//     未声明按 none,声明了就必须是合法等级;
//   - 其余列表字段允许为空,但出现空白条目即报错(空条目是写错了,不是数据)。
func (c TaskContract) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("taskcontract: 缺少 id")
	}
	if strings.ContainsAny(c.ID, " \t\r\n") {
		return fmt.Errorf("taskcontract: id %q 含空白字符", c.ID)
	}
	if !IsValidProfile(c.Profile) {
		return fmt.Errorf("taskcontract: profile %q 非法,必须是 light/standard/strict 之一", c.Profile)
	}
	if strings.TrimSpace(c.Objective) == "" {
		return fmt.Errorf("taskcontract: 缺少 objective(Task 必须足够明确才能被执行与验证,PRD §7.1)")
	}
	if len(c.Scope.AllowedPaths) == 0 {
		return fmt.Errorf("taskcontract: scope.allowed_paths 不能为空(合同必须声明允许修改的边界)")
	}
	if c.Profile != ProfileLight && len(c.Acceptance) == 0 {
		return fmt.Errorf("taskcontract: profile %s 要求至少一条 acceptance 条款(light 才可省略,PRD §7.2)", c.Profile)
	}

	seen := map[string]bool{}
	for _, a := range c.Acceptance {
		if strings.TrimSpace(a.ID) == "" {
			return fmt.Errorf("taskcontract: acceptance 条款缺少 id")
		}
		if strings.ContainsAny(a.ID, " \t\r\n") {
			return fmt.Errorf("taskcontract: acceptance id %q 含空白字符", a.ID)
		}
		if seen[a.ID] {
			return fmt.Errorf("taskcontract: acceptance id %q 重复", a.ID)
		}
		seen[a.ID] = true
		if strings.TrimSpace(a.Statement) == "" {
			return fmt.Errorf("taskcontract: acceptance %s 缺少 statement", a.ID)
		}
		if a.IndependenceRequired != "" && !a.IndependenceRequired.IsValid() {
			return fmt.Errorf("taskcontract: acceptance %s 的 independence_required %q 非法,必须是 none/source/information 之一", a.ID, a.IndependenceRequired)
		}
	}

	for _, list := range []struct {
		name string
		vals []string
	}{
		{"product_refs", c.ProductRefs},
		{"requirement_refs", c.RequirementRefs},
		{"architecture_refs", c.ArchitectureRefs},
		{"depends_on", c.DependsOn},
		{"scope.allowed_paths", c.Scope.AllowedPaths},
		{"scope.forbidden_paths", c.Scope.ForbiddenPaths},
		{"documentation_impact", c.DocumentationImpact},
	} {
		for _, v := range list.vals {
			if strings.TrimSpace(v) == "" {
				return fmt.Errorf("taskcontract: %s 含空白条目", list.name)
			}
		}
	}
	return nil
}
