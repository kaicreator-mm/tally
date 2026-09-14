package taskcontract

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
//   - 严格模式:YAML 里出现合同 schema 之外的字段直接报错。TaskContract 是
//     LLM Proposal 经 Tally 静态校验后的制品(PRD §7.4),未知字段是拼写错误
//     或私货,静默丢弃会让「声明了什么」与「生效了什么」分叉;
//   - 已知的执行策略字段(PRD §7.4:max_iterations、token/wall budget、
//     network、shell、credentials、retry、runtime/model、stop conditions)
//     单独点名报错 —— 它们属于 ExecutionPolicy,出现在 TaskContract 里是
//     越权,必须显式拒绝而不是忽略(否则执行预算会从合同后门溜回来);
//   - 只接受单文档 YAML;空文档报错。

// contractFile 是一份 TaskContract YAML 文件的解码载体:内嵌合同本身,
// 外加一组「出现即报错」的执行策略字段。类型用 any:无论写成 5、true 还是
// 字符串,都按「字段出现」拒绝,而不是先被类型错误挡住。
type contractFile struct {
	TaskContract `yaml:",inline"`

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
var forbiddenFields = []struct {
	yamlField string
	owner     string
	get       func(cf contractFile) any
}{
	{"max_iterations", "ExecutionPolicy", func(cf contractFile) any { return cf.MaxIterations }},
	{"token_budget", "ExecutionPolicy", func(cf contractFile) any { return cf.TokenBudget }},
	{"wall_budget", "ExecutionPolicy", func(cf contractFile) any { return cf.WallBudget }},
	{"timeout", "ExecutionPolicy", func(cf contractFile) any { return cf.Timeout }},
	{"network", "ExecutionPolicy", func(cf contractFile) any { return cf.Network }},
	{"shell", "ExecutionPolicy", func(cf contractFile) any { return cf.Shell }},
	{"credentials", "ExecutionPolicy", func(cf contractFile) any { return cf.Credentials }},
	{"retry", "ExecutionPolicy", func(cf contractFile) any { return cf.Retry }},
	{"runtime", "ExecutionPolicy", func(cf contractFile) any { return cf.Runtime }},
	{"model", "ExecutionPolicy", func(cf contractFile) any { return cf.Model }},
	{"stop_conditions", "ExecutionPolicy", func(cf contractFile) any { return cf.StopConditions }},
}

// Parse 从字节解析并校验一份 TaskContract,填充内容修订哈希。
func Parse(data []byte) (TaskContract, error) {
	var cf contractFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // 未知字段是错误,不是可忽略数据
	if err := dec.Decode(&cf); err != nil {
		if errors.Is(err, io.EOF) {
			return TaskContract{}, fmt.Errorf("taskcontract: 空 YAML 文档")
		}
		return TaskContract{}, fmt.Errorf("taskcontract: YAML 解析: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return TaskContract{}, fmt.Errorf("taskcontract: 只接受单文档 YAML")
	}
	for _, f := range forbiddenFields {
		if f.get(cf) != nil {
			return TaskContract{}, fmt.Errorf("taskcontract: 字段 %s 不属于 TaskContract(由 %s 管理,PRD §7.4);执行预算/网络/shell/凭证不得写进任务合同", f.yamlField, f.owner)
		}
	}
	c := cf.TaskContract
	c.normalize()
	if err := c.Validate(); err != nil {
		return TaskContract{}, err
	}
	c.Hash = c.RevisionHash()
	return c, nil
}

// normalize 把解析出的文本规范成「内容即所存」:YAML 的块标量折叠写法
// (objective: >)会保留一个尾随换行,路径条目可能带缩进空白 —— 这些是排版,
// 不是内容。去首尾空白后存进类型,下游比对与哈希输入都稳定。
func (c *TaskContract) normalize() {
	c.ID = strings.TrimSpace(c.ID)
	c.Profile = strings.TrimSpace(c.Profile)
	c.Title = strings.TrimSpace(c.Title)
	c.Objective = strings.TrimSpace(c.Objective)
	c.ProductRefs = trimEntries(c.ProductRefs)
	c.RequirementRefs = trimEntries(c.RequirementRefs)
	c.ArchitectureRefs = trimEntries(c.ArchitectureRefs)
	c.DependsOn = trimEntries(c.DependsOn)
	c.Scope.AllowedPaths = trimEntries(c.Scope.AllowedPaths)
	c.Scope.ForbiddenPaths = trimEntries(c.Scope.ForbiddenPaths)
	c.DocumentationImpact = trimEntries(c.DocumentationImpact)
	for i := range c.Acceptance {
		c.Acceptance[i].ID = strings.TrimSpace(c.Acceptance[i].ID)
		c.Acceptance[i].Statement = strings.TrimSpace(c.Acceptance[i].Statement)
		c.Acceptance[i].IndependenceRequired = IndependenceLevel(strings.TrimSpace(string(c.Acceptance[i].IndependenceRequired)))
	}
}

// trimEntries 就地去首尾空白(改原切片)。
func trimEntries(vals []string) []string {
	for i := range vals {
		vals[i] = strings.TrimSpace(vals[i])
	}
	return vals
}

// ParseFile 读并解析一份任务合同文件(仓库内路径 .tally/tasks/TASK-*.yaml)。
func ParseFile(path string) (TaskContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TaskContract{}, fmt.Errorf("taskcontract: 读 %s: %w", path, err)
	}
	c, err := Parse(data)
	if err != nil {
		return TaskContract{}, fmt.Errorf("taskcontract: 解析 %s: %w", path, err)
	}
	return c, nil
}
