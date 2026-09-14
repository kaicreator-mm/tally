package engineeringpolicy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// 解析规则(确定性,不猜):
//
//   - 严格模式(AC1):YAML 里出现 Policy schema 之外的字段直接报错。
//     Policy 是验收/门禁权威,未知字段是拼写错误或私货,静默丢弃会让
//     「声明了什么」与「生效了什么」分叉;
//   - 已知的执行策略字段(决策锁 6 归 ExecutionPolicy:network、shell、
//     credentials、retry、预算、runtime/model、stop conditions)单独点名报错
//     —— 它们出现在 EngineeringPolicy 里是把运行时边界从后门塞进验收策略,
//     必须显式拒绝而不是当未知字段含糊报错;
//   - 只接受单文档 YAML;空文档报错。

// policyFile 是一份 EngineeringPolicy YAML 的解码载体:内嵌策略本身,外加一组
// 「出现即报错」的执行策略字段。类型用 any:无论写成 5、true 还是字符串,都按
// 「字段出现」拒绝,而不是先被类型错误挡住。
type policyFile struct {
	Policy `yaml:",inline"`

	MaxIterations  any `yaml:"max_iterations"`
	TokenBudget    any `yaml:"token_budget"`
	WallBudget     any `yaml:"wall_budget"`
	Timeout        any `yaml:"timeout"`
	Network        any `yaml:"network"`
	Shell          any `yaml:"shell"`
	Credentials    any `yaml:"credentials"`
	Retry          any `yaml:"retry"`
	Runtime        any `yaml:"runtime"`
	Model          any `yaml:"model"`
	StopConditions any `yaml:"stop_conditions"`
}

// forbiddenFields 列出执行策略字段:yaml 名、归属层、取值函数(报错与探测用)。
// 决策锁 6:ExecutionPolicy != EngineeringPolicy,两侧字段不得互相渗透。
var forbiddenFields = []struct {
	yamlField string
	owner     string
	get       func(pf policyFile) any
}{
	{"max_iterations", "ExecutionPolicy", func(pf policyFile) any { return pf.MaxIterations }},
	{"token_budget", "ExecutionPolicy", func(pf policyFile) any { return pf.TokenBudget }},
	{"wall_budget", "ExecutionPolicy", func(pf policyFile) any { return pf.WallBudget }},
	{"timeout", "ExecutionPolicy", func(pf policyFile) any { return pf.Timeout }},
	{"network", "ExecutionPolicy", func(pf policyFile) any { return pf.Network }},
	{"shell", "ExecutionPolicy", func(pf policyFile) any { return pf.Shell }},
	{"credentials", "ExecutionPolicy", func(pf policyFile) any { return pf.Credentials }},
	{"retry", "ExecutionPolicy", func(pf policyFile) any { return pf.Retry }},
	{"runtime", "ExecutionPolicy", func(pf policyFile) any { return pf.Runtime }},
	{"model", "ExecutionPolicy", func(pf policyFile) any { return pf.Model }},
	{"stop_conditions", "ExecutionPolicy", func(pf policyFile) any { return pf.StopConditions }},
}

// Parse 从字节解析并校验一份 EngineeringPolicy,填充内容修订哈希(AC4)。
func Parse(data []byte) (Policy, error) {
	var pf policyFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // 未知字段是错误,不是可忽略数据(AC1)
	if err := dec.Decode(&pf); err != nil {
		if errors.Is(err, io.EOF) {
			return Policy{}, fmt.Errorf("%w: 空 YAML 文档", ErrInvalidPolicy)
		}
		return Policy{}, fmt.Errorf("%w: YAML 解析: %w", ErrInvalidPolicy, err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return Policy{}, fmt.Errorf("%w: 只接受单文档 YAML", ErrInvalidPolicy)
	}
	for _, f := range forbiddenFields {
		if f.get(pf) != nil {
			return Policy{}, fmt.Errorf("%w: 字段 %s 不属于 EngineeringPolicy(由 %s 管理,决策锁 6);运行时预算/网络/shell/凭据不得写进工程策略", ErrInvalidPolicy, f.yamlField, f.owner)
		}
	}
	p := pf.Policy
	p.normalize()
	if err := p.Validate(); err != nil {
		return Policy{}, err
	}
	p.Revision = p.RevisionHash()
	return p, nil
}

// normalize 把解析出的文本规范成「内容即所存」:YAML 的块标量折叠写法会保留
// 尾随换行,列表条目可能带缩进空白 —— 这些是排版,不是内容。去首尾空白后存进
// 类型,下游比对与哈希输入都稳定。
func (p *Policy) normalize() {
	p.ID = strings.TrimSpace(p.ID)
	p.Rationale = strings.TrimSpace(p.Rationale)
	p.Trigger.RiskSignals = trimEntries(p.Trigger.RiskSignals)
	p.Trigger.Paths = trimEntries(p.Trigger.Paths)
	p.Trigger.RequirementRefs = trimEntries(p.Trigger.RequirementRefs)
	p.Requires = trimEntries(p.Requires)
	p.GateCondition.AllOf = trimEntries(p.GateCondition.AllOf)
	p.BundleTags = trimEntries(p.BundleTags)
}

// trimEntries 就地去首尾空白(改原切片)。
func trimEntries(vals []string) []string {
	for i := range vals {
		vals[i] = strings.TrimSpace(vals[i])
	}
	return vals
}

// ParseFile 读并解析一份工程策略文件(仓库内路径由加载方 P1-03 决定)。
func ParseFile(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("engineeringpolicy: 读 %s: %w", path, err)
	}
	p, err := Parse(data)
	if err != nil {
		return Policy{}, fmt.Errorf("engineeringpolicy: 解析 %s: %w", path, err)
	}
	return p, nil
}
