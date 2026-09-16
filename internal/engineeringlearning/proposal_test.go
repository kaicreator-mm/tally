package engineeringlearning

import (
	"errors"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/intelligence"
	"tally/internal/policygovernance"
)

const testPolicyYAML = `id: POL-V155-LEARNING
enforcement: gate
rationale: learned engineering rule
trigger:
  risk_signals:
    - schema_migration
requires:
  - data_integrity
gate_condition:
  all_of:
    - data_integrity
`

func TestProposalRequiresOutcomeBackedEvidence(t *testing.T) {
	candidate := candidate(t)
	verification := evidence(t, policygovernance.EvidenceVerification, "verify-1")
	_, err := NewProposal("rule-1", candidate, []policygovernance.EvidenceRef{verification}, []string{"out-1"}, "tally/project", 0.8, nil, provenance(), "owner", nil, false, false)
	if !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("expected missing outcome evidence rejection, got %v", err)
	}
}

func TestProposalRejectsInvalidConfidenceAndConflict(t *testing.T) {
	candidate := candidate(t)
	outcome := evidence(t, policygovernance.EvidenceOutcome, "out-1")
	if _, err := NewProposal("rule-1", candidate, []policygovernance.EvidenceRef{outcome}, []string{"out-1"}, "tally/project", 1.1, nil, provenance(), "owner", nil, false, false); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("expected confidence rejection, got %v", err)
	}
	p := mustProposal(t, nil, []Conflict{{RuleID: "POL-OLD", Reason: "opposite requirement"}})
	if err := p.ValidateForPromotion(); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected unresolved conflict, got %v", err)
	}
}

func TestAIProposalCannotSelfPromote(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	provider := &intelligence.ProviderProvenance{Kind: intelligence.ProviderDirectLLM, ProviderID: "direct-ai", OperationID: "op-1", ObservedAt: now}
	p := mustProposal(t, provider, nil)
	requirement := p.AuthorityRequirement()
	self := engineering.AuthorityDecision{RequirementID: requirement.ID, Status: engineering.AuthorityAuthorized, Actor: "direct-ai", At: now}
	if _, err := Promote(p, []engineering.AuthorityDecision{self}, "direct-ai"); !errors.Is(err, ErrSelfPromotion) {
		t.Fatalf("AI self-promotion must be rejected, got %v", err)
	}
}

func TestAuthorizedPromotionPreparesProposedPolicyGovernanceHandoff(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	provider := &intelligence.ProviderProvenance{Kind: intelligence.ProviderAIRuntime, ProviderID: "ai-runtime", OperationID: "op-2", ObservedAt: now}
	p := mustProposal(t, provider, []Conflict{{RuleID: "POL-OLD", Reason: "reviewed", Resolved: true}})
	requirement := p.AuthorityRequirement()
	human := engineering.AuthorityDecision{RequirementID: requirement.ID, Status: engineering.AuthorityAuthorized, Actor: "human-owner", At: now}
	proposed, err := Promote(p, []engineering.AuthorityDecision{human}, "human-owner")
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	// Learning must never fabricate ACTIVE. Only policygovernance.Store.Activate
	// may cross the Git-backed runtime activation boundary.
	if proposed.Status != policygovernance.StatusProposed || proposed.ApprovedBy != "" {
		t.Fatalf("promotion must hand off PROPOSED, got status=%s approved=%q", proposed.Status, proposed.ApprovedBy)
	}
	if proposed.Candidate.Revision() != p.Candidate.Revision() {
		t.Fatal("promotion changed canonical candidate")
	}
	if len(proposed.Evidence) != len(p.Evidence) {
		t.Fatal("promotion lost evidence")
	}
	if proposed.Scope != p.ApplicableScope {
		t.Fatalf("promotion lost scope: got %q want %q", proposed.Scope, p.ApplicableScope)
	}

	event, err := PromotionEvent(proposed, now)
	if err != nil {
		t.Fatalf("PromotionEvent: %v", err)
	}
	if event.Kind != policygovernance.EventSubmit || event.ApprovedBy != "" {
		t.Fatalf("learning handoff must persist submit, not activate: kind=%s approved=%q", event.Kind, event.ApprovedBy)
	}
	if event.ProposalID != proposed.ID || event.CandidateRevision != proposed.Candidate.Revision() || event.Scope != proposed.Scope || len(event.Evidence) != len(proposed.Evidence) {
		t.Fatalf("promotion event lost governance facts: %+v", event)
	}
}

func TestPromotionEventRejectsPrematureActiveValue(t *testing.T) {
	now := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	provider := &intelligence.ProviderProvenance{Kind: intelligence.ProviderAIRuntime, ProviderID: "ai-runtime", OperationID: "op-2", ObservedAt: now}
	p := mustProposal(t, provider, nil)
	requirement := p.AuthorityRequirement()
	human := engineering.AuthorityDecision{RequirementID: requirement.ID, Status: engineering.AuthorityAuthorized, Actor: "human-owner", At: now}
	proposed, err := Promote(p, []engineering.AuthorityDecision{human}, "human-owner")
	if err != nil {
		t.Fatal(err)
	}
	active, err := proposed.Activate("human-owner")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PromotionEvent(active, now); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("premature ACTIVE must not be accepted as persistence handoff, got %v", err)
	}
}

func TestSharedHumanProposalRequiresAuthority(t *testing.T) {
	candidate := candidate(t)
	outcome := evidence(t, policygovernance.EvidenceOutcome, "out-1")
	p, err := NewProposal("rule-human", candidate, []policygovernance.EvidenceRef{outcome}, []string{"out-1"}, "shared/tally", 0.9, nil, provenance(), "human-proposer", nil, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(p, nil, "human-owner"); !errors.Is(err, ErrAuthorityRequired) {
		t.Fatalf("shared promotion must require authority, got %v", err)
	}
}

func candidate(t *testing.T) policygovernance.Candidate {
	t.Helper()
	candidate, err := policygovernance.NewCandidate([]byte(testPolicyYAML))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	return candidate
}

func evidence(t *testing.T, kind policygovernance.EvidenceKind, ref string) policygovernance.EvidenceRef {
	t.Helper()
	e, err := policygovernance.NewEvidenceRef(kind, ref)
	if err != nil {
		t.Fatalf("NewEvidenceRef: %v", err)
	}
	return e
}

func provenance() engineering.Provenance {
	return engineering.Provenance{Actor: "test", Source: "outcome-analysis", At: time.Date(2026, 9, 15, 9, 55, 0, 0, time.UTC)}
}

func mustProposal(t *testing.T, provider *intelligence.ProviderProvenance, conflicts []Conflict) Proposal {
	t.Helper()
	outcome := evidence(t, policygovernance.EvidenceOutcome, "out-1")
	proposedBy := "human"
	if provider != nil {
		proposedBy = provider.ProviderID
	}
	p, err := NewProposal("rule-1", candidate(t), []policygovernance.EvidenceRef{outcome}, []string{"out-1"}, "tally/project", 0.85, conflicts, provenance(), proposedBy, provider, true, true)
	if err != nil {
		t.Fatalf("NewProposal: %v", err)
	}
	return p
}
