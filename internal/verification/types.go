// Package verification 实现 PRD §8 的 VerificationContract 领域类型(P1-02):
// 不可变(frozen)的验证契约,含 evidence binding、authoring provenance(INV-VER-01)、
// visibility policy 与 held-out 能力要求(INV-VER-05)、required gates 和
// 实现侧 runtime 能力要求。
//
// 边界(P1):
//   - 本包只负责「契约长什么样、如何解析、如何校验、如何冻结与哈希」;
//     不执行任何验证,也不持有 evidence 存储或 runtime(后续任务)。
//   - 冻结后的契约只读;修改验证契约 = 作者侧形成新 Contract/新 revision 再 Freeze
//     (PRD §8.2 第 9 条:修改必须形成新 revision,旧事实不得被覆盖)。
//     这是 INV-VER-02(实现权威不能重定义验证权威)在类型层面的落点。
package verification

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AuthoringMode — 验证权威的 authoring provenance(INV-VER-01:required verification
// authority 必须声明 authoring provenance)。PRD §8.2.1。
type AuthoringMode string

const (
	AuthoringPreCommit AuthoringMode = "pre_commit" // 默认与最高可信路径:目标 Run 开始前定义并冻结
	AuthoringPostHoc   AuthoringMode = "post_hoc"   // 仅运行中治理升级;candidate-blind 流程约束归后续任务
)

// VisibilityPolicy — binding 的可见性契约策略;冻结后不因运行事实改写(PRD §8.4.1)。
type VisibilityPolicy string

const (
	VisibilityVisible VisibilityPolicy = "visible"  // Implementation Authority 可读
	VisibilityHeldOut VisibilityPolicy = "held_out" // 内容与可识别 verdict 不得进入实现侧通道
)

// IndependenceRequired — Acceptance 级独立性要求;属于 Acceptance Clause 而非单条
// binding(PRD §8.5,Decision Lock 10:none < source < information)。
type IndependenceRequired string

const (
	IndependenceNone        IndependenceRequired = "none"
	IndependenceSource      IndependenceRequired = "source"
	IndependenceInformation IndependenceRequired = "information"
)

// SourceBasis — binding 的 desired provenance basis(PRD §8.2.2)。
// 注意:这只是声明;真正的 effective_source_independence(binding, changeset)
// 相对该 ChangeSet 的实际 Implementation Authority provenance 推导,不在本类型内。
type SourceBasis string

const (
	SourceVerificationAuthority   SourceBasis = "verification_authority"
	SourcePreExisting             SourceBasis = "pre_existing"
	SourceHuman                   SourceBasis = "human"
	SourceExternalSystem          SourceBasis = "external_system"
	SourceImplementationAuthority SourceBasis = "implementation_authority"
)

// InformationBasis — information independence 维度的声明;与 visibility_policy 一致
// (PRD §8.11 示例:visible↔visible、held_out↔held_out)。
type InformationBasis string

const (
	InfoVisible InformationBasis = "visible"
	InfoHeldOut InformationBasis = "held_out"
)

// EvidenceType — PRD §8.12 列举的证据类型;不同类型的 qualification policy 归 ADR/后续任务。
type EvidenceType string

const (
	EvidenceTest                   EvidenceType = "test"
	EvidenceValidator              EvidenceType = "validator"
	EvidenceBuild                  EvidenceType = "build"
	EvidenceSchemaCheck            EvidenceType = "schema_check"
	EvidenceStaticRule             EvidenceType = "static_rule"
	EvidenceHumanVerification      EvidenceType = "human_verification"
	EvidenceDeploymentVerification EvidenceType = "deployment_verification"
	EvidenceRuntimeObservation     EvidenceType = "runtime_observation"
)

// ProtectionMode — protected evidence 的保护模式。P1 唯一合法值是 immutable
// (INV-VER-03:protected verification evidence 对其所授权的实现 Run 不可变)。
type ProtectionMode string

const ProtectionImmutable ProtectionMode = "immutable"

// Aggregation — 同 candidate 多次重复执行的聚合方式(PRD §8.7.2 禁止 cherry-pick)。
// P1 只支持 all_pass。
type Aggregation string

const AggregationAllPass Aggregation = "all_pass"

// IndependenceBasis — PRD §8.2.2 的 independence_basis 声明。
type IndependenceBasis struct {
	Source      SourceBasis      `yaml:"source,omitempty"`
	Information InformationBasis `yaml:"information,omitempty"`
}

// EvidenceRef — 定位一条 evidence:repo 内 test(path+selector)、holdout store 引用
// (store+ref)或具名 validator。held_out binding 只允许 store/ref(PRD §8.4.1:
// 普通 tracked 目录 + forbidden_paths 不构成隐藏机制)。
type EvidenceRef struct {
	Type      EvidenceType `yaml:"type"`
	Path      string       `yaml:"path,omitempty"`
	Selector  string       `yaml:"selector,omitempty"`
	Store     string       `yaml:"store,omitempty"`
	Ref       string       `yaml:"ref,omitempty"`
	Validator string       `yaml:"validator,omitempty"`
}

// Protection — protected evidence 保护声明;冻结后不可被其所授权的 Implementation Run
// 修改(PRD §8.2 第 5 条 / INV-VER-03)。
type Protection struct {
	Mode         ProtectionMode `yaml:"mode"`
	BaselineHash string         `yaml:"baseline_hash,omitempty"` // 形如 sha256:<64 位十六进制>
}

// CommandSpec — 结构化命令(argv/cwd/env/timeout)。argv 语义,无 shell 字符串
// (AGENTS.md 固定栈:controlled commands 用结构化 argv/cwd/env/timeout)。
type CommandSpec struct {
	Argv    []string `yaml:"argv"`
	Dir     string   `yaml:"cwd,omitempty"`
	Env     []string `yaml:"env,omitempty"` // 追加环境变量,KEY=VALUE
	Timeout Duration `yaml:"timeout,omitempty"`
	// InfraExitCodes 声明该命令「文档化的 wiring/infra 错误」退出码集合
	// (P6-05:信任不变量「verifier 错误不得伪造为候选失败」的通道)。命令以
	// 命中的退出码终止 = 内容无关的基础设施失败:编排器记为无 verdict 的
	// 执行错误(needs_review),不落任何 evidence 事实。未命中的退出码语义
	// 不变:0 = pass,其余(含 1)= 可信 FAIL。只允许声明 >= 2 的退出码:
	// 0(pass)与 1(canonical trusted FAIL)不可被重定义为 infra,否则会在
	// 结构上削弱判定语义(见 Validate)。
	InfraExitCodes []int `yaml:"infra_exit_codes,omitempty"`
}

// Validate 校验命令规格:argv 非空、timeout 非负、infra 退出码合法。
func (s *CommandSpec) Validate() error {
	if s == nil {
		return nil
	}
	if len(s.Argv) == 0 || strings.TrimSpace(s.Argv[0]) == "" {
		return fmt.Errorf("verification: 命令 argv 不能为空(结构化 argv,拒绝 shell 字符串)")
	}
	if s.Timeout < 0 {
		return fmt.Errorf("verification: 命令 timeout 不能为负(%v)", time.Duration(s.Timeout))
	}
	seen := map[int]bool{}
	for _, code := range s.InfraExitCodes {
		if code < 2 {
			return fmt.Errorf("verification: infra_exit_codes 只能声明 >= 2 的退出码(0=pass、1=可信 FAIL 不可重定义为 infra,削弱判定语义),当前 %d", code)
		}
		if seen[code] {
			return fmt.Errorf("verification: infra_exit_codes 重复声明退出码 %d", code)
		}
		seen[code] = true
	}
	return nil
}

// isInfraExitCode 报告 code 是否命中冻结协议声明的 infra 退出码集合。
func (s *CommandSpec) isInfraExitCode(code int) bool {
	if s == nil {
		return false
	}
	for _, c := range s.InfraExitCodes {
		if c == code {
			return true
		}
	}
	return false
}

// Duration — 支持 "120s" 形式的 YAML 时长。
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return fmt.Errorf("verification: 时长必须是字符串(如 120s): %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("verification: 非法时长 %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

// ExecutionProtocol — 与 binding 一起冻结的执行协议(PRD §8.7):重复次数、聚合方式、
// 随机 seed 与(可选)结构化命令。同 candidate 重跑只有在 repetitions/stopping rule/
// aggregation 在首次执行前冻结时才属于同一 protocol(§8.7.2)。
type ExecutionProtocol struct {
	Repetitions int          `yaml:"repetitions,omitempty"`
	Aggregation Aggregation  `yaml:"aggregation,omitempty"`
	Seed        string       `yaml:"seed,omitempty"` // 随机 seed 是冻结 protocol 的一部分(§8.7.2)
	Command     *CommandSpec `yaml:"command,omitempty"`
}

// BaselineExpectation — 冻结时对 baseline 执行的期望(PRD §8.3.1)。
type BaselineExpectation struct {
	Status         string `yaml:"status"` // pass | fail
	PreferredClass string `yaml:"preferred_class,omitempty"`
}

// BaselineObserved — 实际 baseline 执行的观察事实(§8.3.4:只记录 observed evidence,
// 不得要求 LLM 预先预测 fingerprint)。
type BaselineObserved struct {
	Status      string `yaml:"status,omitempty"` // pass | fail
	Class       string `yaml:"class,omitempty"`  // assertion | compile_error | panic | error
	Fingerprint string `yaml:"fingerprint,omitempty"`
	Commit      string `yaml:"commit,omitempty"`
}

// Baseline — baseline qualification 的期望与观察。
type Baseline struct {
	Expectation *BaselineExpectation `yaml:"expectation,omitempty"`
	Observed    *BaselineObserved    `yaml:"observed,omitempty"`
}

// Qualification — evidence qualification 声明(PRD §8.3)。
type Qualification struct {
	BaselineSensitivity string `yaml:"baseline_sensitivity,omitempty"` // strong | weak | none | n/a
	ChangeOverlap       string `yaml:"change_overlap,omitempty"`       // required
	ReviewMode          string `yaml:"review_mode,omitempty"`          // compact | explicit
	Rationale           string `yaml:"rationale,omitempty"`
}

// Binding — 一条 required evidence 的完整声明:authoring provenance(INV-VER-01)、
// visibility policy(INV-VER-05)、independence basis、evidence 定位、protection
// (INV-VER-03)、冻结执行协议、baseline 与 qualification。
type Binding struct {
	ID                string             `yaml:"id"`
	AuthoringMode     AuthoringMode      `yaml:"authoring_mode"`
	VisibilityPolicy  VisibilityPolicy   `yaml:"visibility_policy"`
	Basis             *IndependenceBasis `yaml:"independence_basis,omitempty"`
	Evidence          EvidenceRef        `yaml:"evidence"`
	Protection        *Protection        `yaml:"protection,omitempty"`
	ExecutionProtocol *ExecutionProtocol `yaml:"execution_protocol,omitempty"`
	Baseline          *Baseline          `yaml:"baseline,omitempty"`
	Qualification     *Qualification     `yaml:"qualification,omitempty"`
}

// Acceptance — 一条验收条款:statement + Acceptance 级 independence requirement
// (Decision Lock 10)+ planned evidence bindings。
type Acceptance struct {
	ID                   string               `yaml:"id"`
	Statement            string               `yaml:"statement"`
	IndependenceRequired IndependenceRequired `yaml:"independence_required,omitempty"` // 空缺省 = none
	EvidenceBindings     []Binding            `yaml:"evidence_bindings"`
}

// RuntimeRequirements — 实现侧 runtime 必须声明的能力要求(PRD §8.4.2 / INV-VER-05)。
//
// 只声明能力与 conformance class,不声明具体 runtime 身份/版本:实际 runtime identity
// 与 conformance evidence 由 CodegenRun / authoring provenance 记录,不写入 frozen
// VerificationContract(PRD §8.11 末注)。运行时若不能保证这些能力,就不得获得
// VerificationPlanner Role 资格;资格判定归 runtime adapter/conformance 任务。
type RuntimeRequirements struct {
	SessionIsolation   bool   `yaml:"session_isolation"`           // VerificationPlanner 与 CodingAgent 会话互不可检索
	HeldOutUnreachable bool   `yaml:"held_out_unreachable"`        // held-out 内容对实现 workspace 机械不可达
	ConformanceClass   string `yaml:"conformance_class,omitempty"` // 如 verification-isolation-v1
}

// Gate — required gate(如 build):required 标志 + 结构化命令。
type Gate struct {
	Required bool         `yaml:"required"`
	Command  *CommandSpec `yaml:"command,omitempty"`
}

// Contract — VerificationContract 的作者侧(未冻结)表示。字段与 PRD §8.11 示例一一对应。
type Contract struct {
	TaskID              string              `yaml:"task_id"`
	Revision            int                 `yaml:"revision"`
	Acceptances         []Acceptance        `yaml:"acceptances"`
	RuntimeRequirements RuntimeRequirements `yaml:"implementation_runtime_requirements"`
	RequiredGates       map[string]Gate     `yaml:"required_gates,omitempty"`
}

// Frozen — 冻结后的 VerificationContract:校验通过后对内容做深拷贝并计算规范化哈希。
// 只读访问器;没有任何修改路径。INV-VER-02/03 的类型层落点。
type Frozen struct {
	c    Contract // 深拷贝,调用方对原 Contract 的后续改动不影响
	hash string
}

// Freeze 校验并冻结契约,返回只读视图。失败返回 error 且不产生 Frozen。
func (c *Contract) Freeze() (*Frozen, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	fc := c.clone()
	h, err := Hash(&fc)
	if err != nil {
		return nil, fmt.Errorf("verification: 计算契约哈希: %w", err)
	}
	return &Frozen{c: fc, hash: h}, nil
}

// TaskID 返回契约所属任务 id。
func (f *Frozen) TaskID() string { return f.c.TaskID }

// Revision 返回契约 revision。
func (f *Frozen) Revision() int { return f.c.Revision }

// Hash 返回冻结内容哈希(sha256:<hex>),重复调用稳定。
func (f *Frozen) Hash() string { return f.hash }

// RuntimeRequirements 返回实现侧 runtime 能力要求(纯值,无引用共享)。
func (f *Frozen) RuntimeRequirements() RuntimeRequirements { return f.c.RuntimeRequirements }

// Contract 返回冻结内容的深拷贝;对其修改不影响 Frozen。
func (f *Frozen) Contract() Contract { return f.c.clone() }

var sha256HashRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// Validate 校验契约的全部确定性约束。Parse 与 Freeze 都调用它。
func (c *Contract) Validate() error {
	if strings.TrimSpace(c.TaskID) == "" {
		return fmt.Errorf("verification: task_id 不能为空")
	}
	if c.Revision < 1 {
		return fmt.Errorf("verification: revision 必须 >= 1(当前 %d)", c.Revision)
	}
	if len(c.Acceptances) == 0 {
		return fmt.Errorf("verification: acceptances 不能为空")
	}

	accIDs := map[string]bool{}
	bindingIDs := map[string]bool{}
	heldOut := false

	for i := range c.Acceptances {
		acc := &c.Acceptances[i]
		if strings.TrimSpace(acc.ID) == "" {
			return fmt.Errorf("verification: acceptances[%d].id 不能为空", i)
		}
		if accIDs[acc.ID] {
			return fmt.Errorf("verification: acceptance id 重复: %s", acc.ID)
		}
		accIDs[acc.ID] = true
		if strings.TrimSpace(acc.Statement) == "" {
			return fmt.Errorf("verification: acceptance %s statement 不能为空", acc.ID)
		}
		switch acc.IndependenceRequired {
		case "", IndependenceNone, IndependenceSource, IndependenceInformation:
		default:
			return fmt.Errorf("verification: acceptance %s independence_required 非法: %q(none|source|information)", acc.ID, acc.IndependenceRequired)
		}

		for j := range acc.EvidenceBindings {
			b := &acc.EvidenceBindings[j]
			if err := validateBinding(b, bindingIDs); err != nil {
				return fmt.Errorf("verification: acceptance %s binding[%d]: %w", acc.ID, j, err)
			}
			if b.VisibilityPolicy == VisibilityHeldOut {
				heldOut = true
			}
		}

		// Planned independence coverage(PRD §8.5.1):freeze 时全称检查,
		// independence_required != none 而无 basis 足以满足的 binding → FREEZE FAIL,无 Waiver。
		if err := validatePlannedCoverage(acc); err != nil {
			return err
		}
	}

	// INV-VER-05(PRD §8.4.2/8.4.1):契约含 held_out binding 时,runtime 必须声明
	// 会话隔离与 held-out 不可达能力,并给出 conformance class。
	if heldOut {
		r := c.RuntimeRequirements
		if !r.SessionIsolation || !r.HeldOutUnreachable || strings.TrimSpace(r.ConformanceClass) == "" {
			return fmt.Errorf("verification: 契约含 held_out binding,implementation_runtime_requirements 必须声明 session_isolation、held_out_unreachable 与 conformance_class(INV-VER-05)")
		}
	}

	for name, gate := range c.RequiredGates {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("verification: required_gates 出现空名称")
		}
		if gate.Required {
			if gate.Command == nil {
				return fmt.Errorf("verification: required gate %s 标记 required 但缺少 command", name)
			}
			if err := gate.Command.Validate(); err != nil {
				return fmt.Errorf("verification: gate %s: %w", name, err)
			}
		} else if gate.Command != nil {
			if err := gate.Command.Validate(); err != nil {
				return fmt.Errorf("verification: gate %s: %w", name, err)
			}
		}
	}
	return nil
}

func validateBinding(b *Binding, bindingIDs map[string]bool) error {
	if strings.TrimSpace(b.ID) == "" {
		return fmt.Errorf("id 不能为空")
	}
	if bindingIDs[b.ID] {
		return fmt.Errorf("binding id 重复: %s", b.ID)
	}
	bindingIDs[b.ID] = true

	// INV-VER-01:required verification authority 必须声明 authoring provenance。
	switch b.AuthoringMode {
	case AuthoringPreCommit, AuthoringPostHoc:
	default:
		return fmt.Errorf("binding %s authoring_mode 必须声明为 pre_commit|post_hoc(INV-VER-01),当前 %q", b.ID, b.AuthoringMode)
	}

	switch b.VisibilityPolicy {
	case VisibilityVisible, VisibilityHeldOut:
	default:
		return fmt.Errorf("binding %s visibility_policy 必须是 visible|held_out,当前 %q", b.ID, b.VisibilityPolicy)
	}

	if b.Basis != nil {
		switch b.Basis.Source {
		case "", SourceVerificationAuthority, SourcePreExisting, SourceHuman, SourceExternalSystem, SourceImplementationAuthority:
		default:
			return fmt.Errorf("binding %s independence_basis.source 非法: %q", b.ID, b.Basis.Source)
		}
		switch b.Basis.Information {
		case "", InfoVisible, InfoHeldOut:
		default:
			return fmt.Errorf("binding %s independence_basis.information 非法: %q", b.ID, b.Basis.Information)
		}
		if b.Basis.Information != "" && InformationBasis(b.VisibilityPolicy) != b.Basis.Information {
			return fmt.Errorf("binding %s independence_basis.information(%s)与 visibility_policy(%s)矛盾", b.ID, b.Basis.Information, b.VisibilityPolicy)
		}
	}

	switch b.Evidence.Type {
	case EvidenceTest, EvidenceValidator, EvidenceBuild, EvidenceSchemaCheck, EvidenceStaticRule,
		EvidenceHumanVerification, EvidenceDeploymentVerification, EvidenceRuntimeObservation:
	default:
		return fmt.Errorf("binding %s evidence.type 非法: %q", b.ID, b.Evidence.Type)
	}

	// PRD §8.4.1:held_out 内容只能放在 holdout store,不能以 repo path 形式暴露;
	// 反向:visible binding 不得引用 holdout store。
	if b.VisibilityPolicy == VisibilityHeldOut {
		if b.Evidence.Path != "" {
			return fmt.Errorf("binding %s 为 held_out 但 evidence.path 指向 repo 路径;held_out 只能经 store/ref 引用(§8.4.1)", b.ID)
		}
		if strings.TrimSpace(b.Evidence.Store) == "" || strings.TrimSpace(b.Evidence.Ref) == "" {
			return fmt.Errorf("binding %s 为 held_out 但缺少 evidence.store/ref", b.ID)
		}
	} else if strings.EqualFold(b.Evidence.Store, "holdout") {
		return fmt.Errorf("binding %s 为 visible 但引用 holdout store;holdout store 对实现侧不可达", b.ID)
	}
	if b.Evidence.Type == EvidenceValidator && strings.TrimSpace(b.Evidence.Validator) == "" {
		return fmt.Errorf("binding %s 为 validator 但缺少 evidence.validator", b.ID)
	}

	if b.Protection != nil {
		// INV-VER-03:protected evidence 对实现 Run 不可变。
		if b.Protection.Mode != ProtectionImmutable {
			return fmt.Errorf("binding %s protection.mode 必须是 immutable(INV-VER-03),当前 %q", b.ID, b.Protection.Mode)
		}
		if b.Protection.BaselineHash != "" && !sha256HashRe.MatchString(b.Protection.BaselineHash) {
			return fmt.Errorf("binding %s protection.baseline_hash 必须形如 sha256:<64 位十六进制>,当前 %q", b.ID, b.Protection.BaselineHash)
		}
	}

	if b.ExecutionProtocol != nil {
		p := b.ExecutionProtocol
		if p.Repetitions < 1 {
			return fmt.Errorf("binding %s execution_protocol.repetitions 必须 >= 1", b.ID)
		}
		if p.Aggregation != AggregationAllPass {
			return fmt.Errorf("binding %s execution_protocol.aggregation 只支持 all_pass(P1,§8.7.2),当前 %q", b.ID, p.Aggregation)
		}
		if err := p.Command.Validate(); err != nil {
			return fmt.Errorf("binding %s execution_protocol: %w", b.ID, err)
		}
	}

	if b.Baseline != nil {
		if e := b.Baseline.Expectation; e != nil {
			if e.Status != "pass" && e.Status != "fail" {
				return fmt.Errorf("binding %s baseline.expectation.status 必须是 pass|fail,当前 %q", b.ID, e.Status)
			}
		}
		if o := b.Baseline.Observed; o != nil {
			if o.Status != "" && o.Status != "pass" && o.Status != "fail" {
				return fmt.Errorf("binding %s baseline.observed.status 必须是 pass|fail,当前 %q", b.ID, o.Status)
			}
			switch o.Class {
			case "", "assertion", "compile_error", "panic", "error":
			default:
				return fmt.Errorf("binding %s baseline.observed.class 非法: %q", b.ID, o.Class)
			}
		}
	}

	if q := b.Qualification; q != nil {
		switch q.BaselineSensitivity {
		case "", "strong", "weak", "none", "n/a":
		default:
			return fmt.Errorf("binding %s qualification.baseline_sensitivity 非法: %q", b.ID, q.BaselineSensitivity)
		}
		switch q.ChangeOverlap {
		case "", "required":
		default:
			return fmt.Errorf("binding %s qualification.change_overlap 只支持 required,当前 %q", b.ID, q.ChangeOverlap)
		}
		switch q.ReviewMode {
		case "", "compact", "explicit":
		default:
			return fmt.Errorf("binding %s qualification.review_mode 必须是 compact|explicit,当前 %q", b.ID, q.ReviewMode)
		}
	}
	return nil
}

// validatePlannedCoverage 实现 PRD §8.5.1 的 freeze 时机械检查:
//   - source:至少一条 binding 声明了可能 source-independent 的 basis(非 implementation_authority);
//   - information:至少一条 binding 同时满足可能 source-independent 的 basis 与 held_out 可见性。
//
// effective independence 相对具体 ChangeSet 的运行期推导不在本包(§8.5.2,后续任务)。
func validatePlannedCoverage(acc *Acceptance) error {
	switch acc.IndependenceRequired {
	case IndependenceNone, "":
		return nil
	}
	for i := range acc.EvidenceBindings {
		b := &acc.EvidenceBindings[i]
		basisOK := b.Basis != nil && b.Basis.Source != "" && b.Basis.Source != SourceImplementationAuthority
		if !basisOK {
			continue
		}
		if acc.IndependenceRequired == IndependenceSource {
			return nil
		}
		if b.VisibilityPolicy == VisibilityHeldOut {
			return nil // information = source 独立 + held_out 能力同时满足(§8.5)
		}
	}
	if acc.IndependenceRequired == IndependenceSource {
		return fmt.Errorf("verification: acceptance %s independence_required=source 但没有声明可用 source basis 的 binding(planned coverage 缺失,§8.5.1 FREEZE FAIL)", acc.ID)
	}
	return fmt.Errorf("verification: acceptance %s independence_required=information 但没有 held_out 且声明可用 source basis 的 binding(planned coverage 缺失,§8.5.1 FREEZE FAIL)", acc.ID)
}

// clone 返回 c 的深拷贝;Freeze 用它保证冻结内容与作者侧对象彻底脱钩。
func (c *Contract) clone() Contract {
	cp := *c
	cp.Acceptances = make([]Acceptance, len(c.Acceptances))
	for i, acc := range c.Acceptances {
		orig := acc.EvidenceBindings
		acc.EvidenceBindings = make([]Binding, len(orig))
		for j, b := range orig {
			acc.EvidenceBindings[j] = cloneBinding(b)
		}
		cp.Acceptances[i] = acc
	}
	if c.RequiredGates != nil {
		cp.RequiredGates = make(map[string]Gate, len(c.RequiredGates))
		for name, gate := range c.RequiredGates {
			gate.Command = cloneCommandSpec(gate.Command)
			cp.RequiredGates[name] = gate
		}
	}
	return cp
}

func cloneBinding(b Binding) Binding {
	if b.Basis != nil {
		v := *b.Basis
		b.Basis = &v
	}
	if b.Protection != nil {
		v := *b.Protection
		b.Protection = &v
	}
	if b.ExecutionProtocol != nil {
		v := *b.ExecutionProtocol
		v.Command = cloneCommandSpec(v.Command)
		b.ExecutionProtocol = &v
	}
	if b.Baseline != nil {
		v := *b.Baseline
		if v.Expectation != nil {
			e := *v.Expectation
			v.Expectation = &e
		}
		if v.Observed != nil {
			o := *v.Observed
			v.Observed = &o
		}
		b.Baseline = &v
	}
	if b.Qualification != nil {
		v := *b.Qualification
		b.Qualification = &v
	}
	return b
}

func cloneCommandSpec(s *CommandSpec) *CommandSpec {
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
	if s.InfraExitCodes != nil {
		cp.InfraExitCodes = append([]int(nil), s.InfraExitCodes...)
	}
	return &cp
}
