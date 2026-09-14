package contextpack

import (
	"reflect"
	"strings"
	"testing"

	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
)

// mustRequires 手构一份执行前 requires 上下文(与 contract.Bind 的产出同形;
// Bind → Lines 的端到端确定性由 internal/contract 的测试守护)。
func mustRequires() contract.RequiresContext {
	return contract.RequiresContext{Items: []contract.RequiresItem{
		{
			PolicyID:    "policy-migration",
			Enforcement: engineeringpolicy.EnforcementGate,
			Rationale:   "migration-class changes must prove upgrade path",
			Keys:        []string{"data_integrity", "fresh_database", "n_minus_1_upgrade"},
		},
		{
			PolicyID:    "policy-auth-guidance",
			Enforcement: engineeringpolicy.EnforcementGuidance,
			Keys:        []string{"token_storage_self_review"},
		},
	}}
}

// PRD §8.3 Prevention:执行前 requires 上下文进入包内容与 §12.3 记录面,
// 并纳入 context_hash。
func TestBuildCarriesPolicyRequires(t *testing.T) {
	in := Input{Task: testTask(), Verification: frozenContract(t, nil), Requires: mustRequires()}
	p := buildPack(t, in)

	want := mustRequires().Lines()
	if !reflect.DeepEqual(p.PolicyRequires, want) {
		t.Fatalf("PolicyRequires = %v, want %v", p.PolicyRequires, want)
	}
	joined := strings.Join(p.PolicyRequires, "\n")
	for _, want := range []string{"policy-migration", "fresh_database", "token_storage_self_review"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("PolicyRequires 缺少 %q:\n%s", want, joined)
		}
	}

	// 记录面:每个命中策略一条 included 记录,Ref 是策略 id。
	found := map[string]bool{}
	for _, s := range p.Included {
		if s.Source == SectionPolicyRequires {
			if s.Reason != ReasonPolicyRequires {
				t.Fatalf("policy_requires 记录原因 = %q", s.Reason)
			}
			found[s.Ref] = true
		}
	}
	if !found["policy-migration"] || !found["policy-auth-guidance"] {
		t.Fatalf("Included 缺少 policy_requires 记录:%+v", p.Included)
	}

	// 哈希敏感:含 requires 与不含 requires 的包哈希不同;同一输入哈希相同。
	plain := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil)})
	if p.Hash() == plain.Hash() {
		t.Fatalf("requires 未纳入 context_hash:%s", p.Hash())
	}
	again := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil), Requires: mustRequires()})
	if again.Hash() != p.Hash() {
		t.Fatalf("同一输入两次 Build 哈希不同:%s != %s", again.Hash(), p.Hash())
	}
}

// requires 是冻结合同派生的必现内容(实现权威必须在执行前知道验收条件):
// 与合同 refs 同一纪律,不参与每节条目数预算。
func TestPolicyRequiresNotBudgeted(t *testing.T) {
	in := Input{
		Task:         testTask(),
		Verification: frozenContract(t, nil),
		Budget:       Budget{MaxItemsPerSection: 1},
		ProductDocs:  []string{"doc-a.md", "doc-b.md"},
		Requires:     mustRequires(),
	}
	p := buildPack(t, in)

	if len(p.PolicyRequires) != 2 {
		t.Fatalf("PolicyRequires 被预算裁剪:%v(want 2 条)", p.PolicyRequires)
	}
	if len(p.ProductDocs) != 1 {
		t.Fatalf("ProductDocs 预算未生效:%v(want 1 条)", p.ProductDocs)
	}
	for _, s := range p.Excluded {
		if s.Source == SectionPolicyRequires {
			t.Fatalf("policy_requires 不应出现排除记录:%+v", s)
		}
	}
}

// 零值 Requires:包内容与记录面无 policy_requires 痕迹(向后兼容)。
func TestZeroRequiresLeavesPackUnchanged(t *testing.T) {
	p := buildPack(t, Input{Task: testTask(), Verification: frozenContract(t, nil)})

	if p.PolicyRequires != nil {
		t.Fatalf("零值 Requires 的 PolicyRequires = %v, want nil", p.PolicyRequires)
	}
	for _, s := range p.Included {
		if s.Source == SectionPolicyRequires {
			t.Fatalf("零值 Requires 不应产生 policy_requires 记录:%+v", s)
		}
	}
}

// Select 路径透传 Requires(与 Build 同一权威组装路径)。
func TestSelectPassesThroughRequires(t *testing.T) {
	res, err := Select(SelectInput{
		Task:         testTask(),
		Verification: frozenContract(t, nil),
		Requires:     mustRequires(),
	})
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if !reflect.DeepEqual(res.Pack.PolicyRequires, mustRequires().Lines()) {
		t.Fatalf("Select 的 PolicyRequires = %v", res.Pack.PolicyRequires)
	}
	for _, s := range res.Pack.Included {
		if s.Source == SectionPolicyRequires && s.Ref == "policy-migration" && s.Reason == ReasonPolicyRequires {
			return
		}
	}
	t.Fatalf("Select 的 Included 缺少 policy_requires 记录:%+v", res.Pack.Included)
}
