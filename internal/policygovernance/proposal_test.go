// proposal_test.go 拥有 P3-01 提案域的验证测试:构造校验、不可变性、封闭
// 生命周期转移与 AC 的正负两面(提案引用可正典解析候选与证据;治理词汇绝不
// 进入运行时策略 YAML —— DL-08/BI-01)。
package policygovernance

import (
	"errors"
	"strings"
	"testing"
)

const validPolicyYAML = `id: POL-MIGRATION
enforcement: gate
rationale: migration 类变更必须验证升级路径与数据完整性
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
  - n_minus_1_upgrade
  - data_integrity
gate_condition:
  all_of:
    - n_minus_1_upgrade
    - data_integrity
`

const governanceTaintedYAML = `id: POL-MIGRATION
enforcement: gate
requires: [fresh_database]
gate_condition:
  all_of: [fresh_database]
governance_status: PROPOSED
approved_by: someone
`

func mustCandidate(t *testing.T, raw string) Candidate {
	t.Helper()
	c, err := NewCandidate([]byte(raw))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	return c
}

func mustRef(t *testing.T, kind EvidenceKind, ref string) EvidenceRef {
	t.Helper()
	e, err := NewEvidenceRef(kind, ref)
	if err != nil {
		t.Fatalf("NewEvidenceRef(%s,%s): %v", kind, ref, err)
	}
	return e
}

// AC 正面:已提交提案引用可正典解析候选(内容修订可追溯)与证据(≥1)。
func TestProposalReferencesCanonicalCandidateAndEvidence(t *testing.T) {
	cand := mustCandidate(t, validPolicyYAML)
	if cand.Revision() == "" {
		t.Fatal("候选应携带内容修订(Parse 填充 RevisionHash)")
	}
	if cand.Policy().ID != "POL-MIGRATION" {
		t.Fatalf("候选正典解析结果: id = %q", cand.Policy().ID)
	}
	p, err := NewProposal("prop-001", cand, []EvidenceRef{
		mustRef(t, EvidenceOutcome, "out-001"),
		mustRef(t, EvidenceVerification, "sha256:abc"),
	}, "owner-a")
	if err != nil {
		t.Fatalf("NewProposal: %v", err)
	}
	if p.Status != StatusProposed {
		t.Fatalf("Status = %s, want PROPOSED", p.Status)
	}
	if len(p.Evidence) != 2 || p.Evidence[0].String() != "outcome:out-001" {
		t.Fatalf("Evidence = %v", p.Evidence)
	}
}

// 不可变性(要求 1):原始字节副本隔离;候选内容出生即固定,零 setter。
func TestCandidateImmutable(t *testing.T) {
	raw := []byte(validPolicyYAML)
	cand, err := NewCandidate(raw)
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	// 外部切片变更不影响候选。
	raw[0] = 'X'
	if got := string(cand.Raw()[0]); got == "X" {
		t.Fatal("候选原始字节被外部切片变更穿透")
	}
	// Raw() 返回副本:改返回值不穿透候选。
	out := cand.Raw()
	out[1] = 'Y'
	if cand.Raw()[1] == 'Y' {
		t.Fatal("Raw() 副本未隔离")
	}
	// 内容修订在两次构造间稳定(同内容 → 同修订;内容派生)。
	again := mustCandidate(t, validPolicyYAML)
	if again.Revision() != cand.Revision() {
		t.Fatalf("同内容候选修订不稳定: %q vs %q", again.Revision(), cand.Revision())
	}
}

// 治理词汇绝不进入运行时策略 YAML(要求 2/DL-08):携带治理字段的候选在
// 正典解析处出生即拒绝(engineeringpolicy.Parse KnownFields 严格模式)。
func TestCandidateRejectsGovernanceFields(t *testing.T) {
	_, err := NewCandidate([]byte(governanceTaintedYAML))
	if !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("治理字段污染的候选应被拒绝,得到: %v", err)
	}
	// Parse 的错误链必须保留(候选不可解析的根因可追溯)。
	if !strings.Contains(err.Error(), "governance_status") && !strings.Contains(err.Error(), "未知") {
		t.Fatalf("错误应指向未知字段根因: %v", err)
	}
}

// 构造负路径:不可解析候选、空证据、非法身份/证据词表。
func TestProposalConstructionNegatives(t *testing.T) {
	cand := mustCandidate(t, validPolicyYAML)

	if _, err := NewCandidate([]byte("id: P\nenforcement: not-a-mode\n")); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("不可解析候选应拒绝: %v", err)
	}
	if _, err := NewCandidate(nil); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("空候选应拒绝: %v", err)
	}
	if _, err := NewProposal("prop-002", cand, nil, "owner-a"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("零证据提案应拒绝(AC): %v", err)
	}
	if _, err := NewProposal("", cand, []EvidenceRef{mustRef(t, EvidenceOutcome, "o")}, "owner-a"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("空 ID 应拒绝: %v", err)
	}
	if _, err := NewProposal("prop 003", cand, []EvidenceRef{mustRef(t, EvidenceOutcome, "o")}, "owner-a"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("含空白 ID 应拒绝: %v", err)
	}
	if _, err := NewProposal("prop-004", cand, []EvidenceRef{mustRef(t, EvidenceOutcome, "o")}, ""); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("空提案人应拒绝: %v", err)
	}
	if _, err := NewEvidenceRef("deploy", "x"); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("词表外证据类型应拒绝: %v", err)
	}
	if _, err := NewEvidenceRef(EvidenceTask, " "); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("空白引用应拒绝: %v", err)
	}
}

// 生命周期封闭转移:全合法路径 + 全非法路径(表驱动,要求 2)。
func TestLifecycleTransitions(t *testing.T) {
	cand := mustCandidate(t, validPolicyYAML)
	ev := []EvidenceRef{mustRef(t, EvidenceOutcome, "out-001")}

	draft, err := NewDraft("prop-100", cand, "owner-a")
	if err != nil {
		t.Fatalf("NewDraft: %v", err)
	}
	// DRAFT → PROPOSED(提交补证据)。
	proposed, err := draft.Submit(ev)
	if err != nil || proposed.Status != StatusProposed {
		t.Fatalf("Submit: %v status=%s", err, proposed.Status)
	}
	// PROPOSED → ACTIVE(记录批准人)。
	active, err := proposed.Activate("owner-b")
	if err != nil || active.Status != StatusActive || active.ApprovedBy != "owner-b" {
		t.Fatalf("Activate: %v status=%s approvedBy=%q", err, active.Status, active.ApprovedBy)
	}
	// ACTIVE → SUPERSEDED(终态)。
	superseded, err := active.Supersede()
	if err != nil || superseded.Status != StatusSuperseded {
		t.Fatalf("Supersede: %v status=%s", err, superseded.Status)
	}

	// PROPOSED → REJECTED(终态)。
	rejected, err := proposed.Reject()
	if err != nil || rejected.Status != StatusRejected {
		t.Fatalf("Reject: %v status=%s", err, rejected.Status)
	}

	// 非法转移表:每个非法起点/动作组合都必须 ErrInvalidTransition 且原值不变。
	banned := []struct {
		name string
		p    Proposal
		act  func(Proposal) (Proposal, error)
	}{
		{"DRAFT 直接激活", draft, func(p Proposal) (Proposal, error) { return p.Activate("x") }},
		{"DRAFT 否决", draft, func(p Proposal) (Proposal, error) { return p.Reject() }},
		{"DRAFT 取代", draft, func(p Proposal) (Proposal, error) { return p.Supersede() }},
		{"PROPOSED 取代", proposed, func(p Proposal) (Proposal, error) { return p.Supersede() }},
		{"ACTIVE 再激活", active, func(p Proposal) (Proposal, error) { return p.Activate("y") }},
		{"ACTIVE 否决", active, func(p Proposal) (Proposal, error) { return p.Reject() }},
		{"REJECTED 再激活", rejected, func(p Proposal) (Proposal, error) { return rejected.Activate("z") }},
		{"SUPERSEDED 再取代", superseded, func(p Proposal) (Proposal, error) { return superseded.Supersede() }},
		{"SUPERSEDED 否决", superseded, func(p Proposal) (Proposal, error) { return superseded.Reject() }},
	}
	for _, tc := range banned {
		next, err := tc.act(tc.p)
		if !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("%s:期望 ErrInvalidTransition,得到 %v", tc.name, err)
		}
		if next.Status != tc.p.Status {
			t.Errorf("%s:非法转移不得改变原提案状态", tc.name)
		}
	}

	// Activate 批准人非空是硬性要求。
	if _, err := proposed.Activate(""); !errors.Is(err, ErrInvalidProposal) {
		t.Errorf("空批准人应拒绝,得到 %v", err)
	}
	// Submit 零证据拒绝(AC)。
	if _, err := draft.Submit(nil); !errors.Is(err, ErrInvalidProposal) {
		t.Errorf("Submit 零证据应拒绝,得到 %v", err)
	}
}
