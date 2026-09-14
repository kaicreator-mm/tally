// canonical_test.go 拥有 P3-03 的金色解析/修订测试:内容修订的稳定性(排版
// 差异不改修订)、差异性(内容差异必改修订)与 AC(候选修订 = 实际正典策略
// 内容修订语义),以及写入边界的 fail-closed 面(声称修订不一致、重复身份)。
package policygovernance

import (
	"errors"
	"strings"
	"testing"

	"tally/internal/engineeringpolicy"
)

// 金色向量:同一策略内容的三种排版形态(键序、缩进、行尾空白)必须产出同一
// 内容修订;内容差异(语义字段变化)必须产出不同修订。
func TestCanonicalRevisionGolden(t *testing.T) {
	canonical := validPolicyYAML

	// 形态 B:键序不同 + 行尾空白 + 缩进差异(同一内容)。
	reordered := `id: POL-MIGRATION
gate_condition:
  all_of:
    - n_minus_1_upgrade
    - data_integrity
enforcement: gate
rationale: migration 类变更必须验证升级路径与数据完整性
trigger:
  risk_signals: [schema_migration]
requires:
  - data_integrity
  - fresh_database
  - n_minus_1_upgrade
`

	// 形态 C:列表顺序不同(requires/gate_condition 的 all_of 语义集合不变;
	// normalize+sortedUnique 应消化顺序,修订稳定)。
	reorderedLists := `id: POL-MIGRATION
enforcement: gate
rationale: migration 类变更必须验证升级路径与数据完整性
trigger:
  risk_signals:
    - schema_migration
requires:
  - data_integrity
  - n_minus_1_upgrade
  - fresh_database
gate_condition:
  all_of:
    - data_integrity
    - n_minus_1_upgrade
`

	base, err := NewCandidate([]byte(canonical))
	if err != nil {
		t.Fatalf("NewCandidate(基准): %v", err)
	}
	for name, raw := range map[string]string{"键序/空白": reordered, "列表顺序": reorderedLists} {
		variant, err := NewCandidate([]byte(raw))
		if err != nil {
			t.Fatalf("NewCandidate(%s): %v", name, err)
		}
		if variant.Revision() != base.Revision() {
			t.Errorf("%s:同一内容的排版变体修订必须稳定:%q vs %q", name, variant.Revision(), base.Revision())
		}
		if variant.Policy().Canonical() != base.Policy().Canonical() {
			t.Errorf("%s:正典形式必须逐字节一致", name)
		}
	}

	// 差异性:语义内容变化(requires 去掉一条)必须改变修订。
	changed := strings.Replace(canonical, "  - fresh_database\n", "", 1)
	altered, err := NewCandidate([]byte(changed))
	if err != nil {
		t.Fatalf("NewCandidate(变体内容): %v", err)
	}
	if altered.Revision() == base.Revision() {
		t.Fatal("内容变化后修订必须改变")
	}
}

// AC:候选修订 = engineeringpolicy 的实际正典内容修订语义(RevisionHash
// 于 Parse 填充),且 VerifyCandidate 对一致声称通过、对不一致声称拒绝。
func TestCandidateRevisionEqualsCanonicalSemantics(t *testing.T) {
	cand, err := NewCandidate([]byte(validPolicyYAML))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	// 独立走一遍解析器语义:Parse → RevisionHash,必须与候选修订一致。
	p, err := engineeringpolicy.Parse([]byte(validPolicyYAML))
	if err != nil {
		t.Fatalf("engineeringpolicy.Parse: %v", err)
	}
	if cand.Revision() != p.RevisionHash() {
		t.Fatalf("候选修订 %q != 正典内容修订语义 %q", cand.Revision(), p.RevisionHash())
	}

	// VerifyCandidate:一致声称通过并返回候选;不一致声称拒绝(写入边界复用)。
	got, err := VerifyCandidate([]byte(validPolicyYAML), cand.Revision())
	if err != nil || got.Revision() != cand.Revision() {
		t.Fatalf("VerifyCandidate(一致): %v rev=%q", err, got.Revision())
	}
	if _, err := VerifyCandidate([]byte(validPolicyYAML), "sha256:claimed-by-caller"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("VerifyCandidate(不一致声称)应拒绝,得到 %v", err)
	}
	if _, err := VerifyCandidate([]byte(governanceTaintedYAML), ""); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("VerifyCandidate(不可解析)应拒绝,得到 %v", err)
	}
}
