package verification

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Parse 从 YAML 字节解析 VerificationContract(PRD §8.11 的字段形态)并做完整校验。
//
// 解析是严格的:未知字段直接报错,防止把契约写歪后静默丢失约束(例如把
// authoring_mode 拼错导致 INV-VER-01 声明丢失)。返回的是作者侧(未冻结)契约;
// 调用方在 review 通过后调用 Contract.Freeze 产生不可变视图与内容哈希。
func Parse(data []byte) (*Contract, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var c Contract
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("verification: YAML 解析: %w", err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}
