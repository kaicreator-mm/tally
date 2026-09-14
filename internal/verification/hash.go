package verification

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// Hash 返回契约的规范化内容哈希,形如 sha256:<64 位十六进制>。
//
// 规范化规则(P1-02 AC4:revision/hash 稳定且冻结后不可变):
//   - acceptances、acceptance 内的 bindings 按 ID 排序;
//   - required_gates(map,遍历无序)转为按名称排序的列表;
//   - 其余按结构体字段顺序 YAML 序列化后取 sha256。
//
// 因此哈希与输入键序/列表顺序无关,同一内容重复计算结果相同;内容任何变化
// (包括 revision)都会改变哈希。
func Hash(c *Contract) (string, error) {
	data, err := CanonicalYAML(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// CanonicalYAML 返回契约的规范化(排序后)YAML 序列化。导出供测试与调试。
func CanonicalYAML(c *Contract) ([]byte, error) {
	cc := canonicalContract{
		TaskID:              c.TaskID,
		Revision:            c.Revision,
		RuntimeRequirements: c.RuntimeRequirements,
	}
	cc.Acceptances = make([]Acceptance, len(c.Acceptances))
	for i, acc := range c.Acceptances {
		acc.EvidenceBindings = make([]Binding, len(acc.EvidenceBindings))
		copy(acc.EvidenceBindings, c.Acceptances[i].EvidenceBindings)
		sort.Slice(acc.EvidenceBindings, func(a, b int) bool {
			return acc.EvidenceBindings[a].ID < acc.EvidenceBindings[b].ID
		})
		cc.Acceptances[i] = acc
	}
	sort.Slice(cc.Acceptances, func(a, b int) bool {
		return cc.Acceptances[a].ID < cc.Acceptances[b].ID
	})

	if len(c.RequiredGates) > 0 {
		names := make([]string, 0, len(c.RequiredGates))
		for name := range c.RequiredGates {
			names = append(names, name)
		}
		sort.Strings(names)
		cc.RequiredGates = make([]canonicalGate, 0, len(names))
		for _, name := range names {
			cc.RequiredGates = append(cc.RequiredGates, canonicalGate{Name: name, Gate: c.RequiredGates[name]})
		}
	}

	data, err := yaml.Marshal(&cc)
	if err != nil {
		return nil, fmt.Errorf("verification: 规范化序列化: %w", err)
	}
	return data, nil
}

// canonicalContract — 哈希专用的有序投影:gates 由 map 变为排序列表。
type canonicalContract struct {
	TaskID              string              `yaml:"task_id"`
	Revision            int                 `yaml:"revision"`
	Acceptances         []Acceptance        `yaml:"acceptances"`
	RuntimeRequirements RuntimeRequirements `yaml:"implementation_runtime_requirements"`
	RequiredGates       []canonicalGate     `yaml:"required_gates,omitempty"`
}

type canonicalGate struct {
	Name string `yaml:"name"`
	Gate Gate   `yaml:"gate"`
}
