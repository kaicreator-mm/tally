// Package engineeringpolicy 定义 v1.4 EngineeringPolicy(PRD §8):验收/门禁策略,
// 与 internal/policy 的 ExecutionPolicy(运行时资源/网络/shell/retry 边界)是两个
// 不同的对象 —— 决策锁 6:ExecutionPolicy != EngineeringPolicy。本包不引用、
// 不复用、不扩展 ExecutionPolicy 的任何类型(包括 Retry.MaxIterations;外层
// Repair 预算按决策锁 7/23 归 P2-07,与本包无关)。
//
// Policy 按 PRD §8.1 建模八个字段:id、revision、enforcement、rationale、
// trigger、requires[]、gate_condition、bundle_tags[]。刻意保持小:
// 没有策略 DSL、没有规则市场、没有可视化编排 —— gate_condition 只有
// 「这些 requires 键必须全部满足」一种形态。
//
// requires[] 的双重用途(PRD §8.3):
//   - Prevention:执行前作为 Context 告知 Agent「最终会按这些条件验收」;
//   - Assurance:执行后编译 Required Evidence / gate。
//
// 同一份列表同时服务两者;gate_condition 只能引用 requires[] 里声明过的键
// (门禁不得要求 Agent 事先未被告知的条件),由 Validate 强制。
//
// 边界(本包类型形状即约束):
//   - enforcement 只有 gate | guidance 两个保留值(PRD §8.2);v1.4 只正式实现
//     gate。guidance 可表示,但在类型上拿不到阻断权:guidance 携带
//     gate_condition 直接判非法,门禁求值入口对非 gate 策略返回 ErrNotGate(AC2)。
//   - 本包只建模策略值、解析与确定性门禁求值;不执行任何进程、不发起任何
//     网络请求、不嵌入任何 provider/tool 执行(AC3,test/arch 的边界测试守护)。
//   - Revision 是由内容推导的确定性修订哈希(AC4,见 hash.go),不来自 YAML。
package engineeringpolicy

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// 哨兵错误:调用方用 errors.Is 区分,包装文案只提供字段级上下文。
var (
	// ErrInvalidPolicy 表示策略值本身非法(缺 id、未知 enforcement、guidance
	// 携带 gate_condition、gate_condition 引用未声明的 requires 键等)。
	ErrInvalidPolicy = errors.New("engineeringpolicy: invalid policy")
	// ErrNotGate 表示对非 gate(如 guidance)策略请求门禁语义。guidance 在
	// v1.4 不能获得阻断权 —— 这是 AC2 的执行点,不是提醒。
	ErrNotGate = errors.New("engineeringpolicy: policy enforcement is not gate")
	// ErrInvalidGateInput 表示门禁求值输入非法:空白键、重复键、或引用了
	// 策略 requires[] 之外的未知键(未知即失败,不静默忽略)。
	ErrInvalidGateInput = errors.New("engineeringpolicy: invalid gate input")
)

// Enforcement 是策略的执行方式(PRD §8.2)。两个值都是 schema 保留值;
// v1.4 只正式实现 gate。空值不合法:缺席的 enforcement 不得被猜成任何一种。
type Enforcement string

const (
	// EnforcementGate 阻断式:gate_condition 未满足时验收门禁不通过。
	EnforcementGate Enforcement = "gate"
	// EnforcementGuidance 指导式:仅进入上下文/提示,v1.4 无阻断权。
	// schema 保留此值,避免未来迁移破坏(PRD §8.2)。
	EnforcementGuidance Enforcement = "guidance"
)

// IsValid 报告该值是否是 PRD §8.2 定义的两个保留值之一。空串不合法。
func (e Enforcement) IsValid() bool {
	switch e {
	case EnforcementGate, EnforcementGuidance:
		return true
	}
	return false
}

// Trigger 是策略的匹配条件(PRD §8.1),按 task/change/risk 三类输入分轴;
// 匹配语义与覆盖推导(PolicyMatchCoverage)归 P1-05,本包只存储与校验。
// 全部字段可选;空 Trigger 表示该策略不声明任何匹配条件。
type Trigger struct {
	RiskSignals     []string `yaml:"risk_signals"`     // 风险信号名,如 schema_migration(信号模型归 P1-04)
	Paths           []string `yaml:"paths"`            // 仓库内路径(变更输入轴)
	RequirementRefs []string `yaml:"requirement_refs"` // 需求引用,如 .tally/requirements/REQ-018.md(任务输入轴)
}

// IsZero 报告 Trigger 是否未声明任何匹配条件。
func (t Trigger) IsZero() bool {
	return len(t.RiskSignals) == 0 && len(t.Paths) == 0 && len(t.RequirementRefs) == 0
}

// GateCondition 是门禁条件(PRD §8.1):一份必须全部满足的 requires 键集合。
// 刻意不是布尔表达式树 —— 「全部满足」一种确定性语义,不做 DSL。
type GateCondition struct {
	AllOf []string `yaml:"all_of"` // 须全部满足的 requires 键;gate 策略必填非空
}

// Policy 是一份解析后的 v1.4 工程策略(PRD §8.1)。字段与 PRD 一一对应;
// Revision 是内容推导的修订哈希(hash.go),Parse 时填充,不来自 YAML。
type Policy struct {
	ID          string      `yaml:"id"`          // 项目策略集内唯一;集合层唯一性归加载方(P1-03)
	Enforcement Enforcement `yaml:"enforcement"` // gate | guidance(PRD §8.2);必填
	Rationale   string      `yaml:"rationale"`   // 这条策略为什么存在(人类可读,可空)
	Trigger     Trigger     `yaml:"trigger"`

	// Requires 双重用途(PRD §8.3,见包注释)。键是稳定机器名,如 fresh_database。
	Requires []string `yaml:"requires"`

	GateCondition GateCondition `yaml:"gate_condition"`

	BundleTags []string `yaml:"bundle_tags"` // 策略打包标签,供下游按束引用/匹配

	// 非 YAML 字段:内容修订哈希(Parse 时填充,AC4)。
	Revision string `yaml:"-"`
}

// Validate 校验策略自身的硬约束。Parse 时调用;调用方覆盖字段后再调一次。
// 规则全部来自 PRD §8 与决策锁 6,不确定的不猜:
//
//   - id 必填且不含空白字符;
//   - enforcement 必填,必须是 gate | guidance 之一(缺席不猜);
//   - guidance 不得携带 gate_condition(AC2:guidance 在 v1.4 拿不到阻断权);
//   - gate 策略必须声明非空 gate_condition(没有条件的门禁永远空过,
//     等于静默放行);
//   - gate_condition.all_of 的每个键必须出现在 requires[](双重用途:门禁
//     不得要求 Agent 事先未被告知的条件,PRD §8.3);
//   - requires / gate_condition.all_of / bundle_tags / trigger 各列表:
//     条目不得为空白,不得重复(重复是写错,不是数据,严格拒绝)。
func (p Policy) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("%w: 缺少 id", ErrInvalidPolicy)
	}
	if strings.ContainsAny(p.ID, " \t\r\n") {
		return fmt.Errorf("%w: id %q 含空白字符", ErrInvalidPolicy, p.ID)
	}
	if !p.Enforcement.IsValid() {
		return fmt.Errorf("%w: enforcement %q 非法,必须是 gate | guidance 之一(PRD §8.2)", ErrInvalidPolicy, string(p.Enforcement))
	}
	if err := validateStringList("trigger.risk_signals", p.Trigger.RiskSignals); err != nil {
		return err
	}
	if err := validateStringList("trigger.paths", p.Trigger.Paths); err != nil {
		return err
	}
	if err := validateStringList("trigger.requirement_refs", p.Trigger.RequirementRefs); err != nil {
		return err
	}
	if err := validateStringList("requires", p.Requires); err != nil {
		return err
	}
	if err := validateStringList("gate_condition.all_of", p.GateCondition.AllOf); err != nil {
		return err
	}
	if err := validateStringList("bundle_tags", p.BundleTags); err != nil {
		return err
	}

	switch p.Enforcement {
	case EnforcementGuidance:
		if len(p.GateCondition.AllOf) > 0 {
			return fmt.Errorf("%w: enforcement=guidance 不得携带 gate_condition(guidance 在 v1.4 不能获得阻断权)", ErrInvalidPolicy)
		}
	case EnforcementGate:
		if len(p.GateCondition.AllOf) == 0 {
			return fmt.Errorf("%w: enforcement=gate 要求非空 gate_condition.all_of(无条件门禁等于静默放行)", ErrInvalidPolicy)
		}
		for _, key := range p.GateCondition.AllOf {
			if !slices.Contains(p.Requires, key) {
				return fmt.Errorf("%w: gate_condition 引用键 %q 不在 requires[](门禁不得要求未事先告知的条件,PRD §8.3)", ErrInvalidPolicy, key)
			}
		}
	}
	return nil
}

// validateStringList 拒绝空白条目与重复条目。
func validateStringList(field string, vals []string) error {
	for i, v := range vals {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%w: %s[%d] 为空白条目", ErrInvalidPolicy, field, i)
		}
	}
	if containsDuplicate(vals) {
		return fmt.Errorf("%w: %s 存在重复条目", ErrInvalidPolicy, field)
	}
	return nil
}

func containsDuplicate(xs []string) bool {
	seen := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			return true
		}
		seen[x] = struct{}{}
	}
	return false
}
