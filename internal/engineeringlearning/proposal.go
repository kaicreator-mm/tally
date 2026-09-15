package engineeringlearning

import (
	"errors"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/intelligence"
	"tally/internal/policygovernance"
)

var (
	ErrInvalidProposal   = errors.New("engineeringlearning: invalid rule proposal")
	ErrConflict          = errors.New("engineeringlearning: unresolved rule conflict")
	ErrAuthorityRequired = errors.New("engineeringlearning: promotion authority required")
	ErrSelfPromotion     = errors.New("engineeringlearning: AI/provider cannot self-promote")
)

type Conflict struct {
	RuleID   string `json:"rule_id"`
	Reason   string `json:"reason"`
	Resolved bool   `json:"resolved"`
}

type Proposal struct {
	ID                 string                              `json:"id"`
	Candidate          policygovernance.Candidate          `json:"-"`
	Evidence           []policygovernance.EvidenceRef      `json:"evidence"`
	SupportingOutcomes []string                            `json:"supporting_outcomes"`
	ApplicableScope    string                              `json:"applicable_scope"`
	Confidence         float64                             `json:"confidence"`
	Conflicts          []Conflict                          `json:"conflicts,omitempty"`
	Provenance         engineering.Provenance              `json:"provenance"`
	ProposedBy         string                              `json:"proposed_by"`
	Provider           *intelligence.ProviderProvenance    `json:"provider,omitempty"`
	Shared             bool                                `json:"shared,omitempty"`
	HighImpact         bool                                `json:"high_impact,omitempty"`
}

func NewProposal(
	id string,
	candidate policygovernance.Candidate,
	evidence []policygovernance.EvidenceRef,
	outcomes []string,
	scope string,
	confidence float64,
	conflicts []Conflict,
	provenance engineering.Provenance,
	proposedBy string,
	provider *intelligence.ProviderProvenance,
	shared bool,
	highImpact bool,
) (Proposal, error) {
	p := Proposal{
		ID: id, Candidate: candidate,
		Evidence: append([]policygovernance.EvidenceRef(nil), evidence...),
		SupportingOutcomes: append([]string(nil), outcomes...),
		ApplicableScope: scope, Confidence: confidence,
		Conflicts: append([]Conflict(nil), conflicts...), Provenance: provenance,
		ProposedBy: proposedBy, Shared: shared, HighImpact: highImpact,
	}
	if provider != nil {
		cp := *provider
		p.Provider = &cp
	}
	if err := p.Validate(); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

func (p Proposal) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.ContainsAny(p.ID, " \t\r\n") {
		return fmt.Errorf("%w: id must be a non-blank token", ErrInvalidProposal)
	}
	if p.Candidate.Revision() == "" {
		return fmt.Errorf("%w: canonical policy candidate is required", ErrInvalidProposal)
	}
	if strings.TrimSpace(p.ApplicableScope) == "" {
		return fmt.Errorf("%w: applicable scope is required", ErrInvalidProposal)
	}
	if p.Confidence <= 0 || p.Confidence > 1 {
		return fmt.Errorf("%w: confidence must be > 0 and <= 1", ErrInvalidProposal)
	}
	if strings.TrimSpace(p.ProposedBy) == "" {
		return fmt.Errorf("%w: proposed_by is required", ErrInvalidProposal)
	}
	if strings.TrimSpace(p.Provenance.Source) == "" || p.Provenance.At.IsZero() {
		return fmt.Errorf("%w: provenance source/time are required", ErrInvalidProposal)
	}
	if len(p.SupportingOutcomes) == 0 || len(p.Evidence) == 0 {
		return fmt.Errorf("%w: supporting outcomes and evidence are required", ErrInvalidProposal)
	}
	outcomeEvidence := map[string]bool{}
	for _, evidence := range p.Evidence {
		if !evidence.Kind.Valid() || strings.TrimSpace(evidence.Ref) == "" {
			return fmt.Errorf("%w: invalid evidence reference", ErrInvalidProposal)
		}
		if evidence.Kind == policygovernance.EvidenceOutcome {
			outcomeEvidence[evidence.Ref] = true
		}
	}
	seenOutcomes := map[string]bool{}
	for _, outcomeID := range p.SupportingOutcomes {
		outcomeID = strings.TrimSpace(outcomeID)
		if outcomeID == "" || seenOutcomes[outcomeID] {
			return fmt.Errorf("%w: supporting outcomes must be unique non-blank ids", ErrInvalidProposal)
		}
		seenOutcomes[outcomeID] = true
		if !outcomeEvidence[outcomeID] {
			return fmt.Errorf("%w: supporting outcome %s lacks outcome evidence reference", ErrInvalidProposal, outcomeID)
		}
	}
	for _, conflict := range p.Conflicts {
		if strings.TrimSpace(conflict.RuleID) == "" || strings.TrimSpace(conflict.Reason) == "" {
			return fmt.Errorf("%w: conflicts require rule_id and reason", ErrInvalidProposal)
		}
	}
	if p.Provider != nil {
		if p.Provider.Kind != intelligence.ProviderDirectLLM && p.Provider.Kind != intelligence.ProviderAIRuntime {
			return fmt.Errorf("%w: invalid provider kind %q", ErrInvalidProposal, p.Provider.Kind)
		}
		if strings.TrimSpace(p.Provider.ProviderID) == "" || p.Provider.ObservedAt.IsZero() {
			return fmt.Errorf("%w: provider provenance is incomplete", ErrInvalidProposal)
		}
	}
	return nil
}

func (p Proposal) RequiresAuthority() bool {
	// Shared/high-impact rules always require human authority. AI-generated local
	// rules also require independent authority because AI may never self-promote.
	return p.Shared || p.HighImpact || p.Provider != nil
}

func (p Proposal) AuthorityRequirement() engineering.AuthorityRequirement {
	return engineering.AuthorityRequirement{
		ID: engineering.ID("engineering-rule-authority-" + p.ID),
		Mode: engineering.ModeAuthorityRequired,
		Scope: "engineering_learning.promotion",
		Reason: "governed Engineering Rule promotion",
	}
}

func (p Proposal) ValidateForPromotion() error {
	if err := p.Validate(); err != nil {
		return err
	}
	for _, conflict := range p.Conflicts {
		if !conflict.Resolved {
			return fmt.Errorf("%w: %s", ErrConflict, conflict.RuleID)
		}
	}
	return nil
}

// Promote crosses the Engineering Learning governance boundary into the
// existing policygovernance lifecycle. It does not modify runtime policy
// authority; the returned ACTIVE governance proposal still requires the
// existing Git-backed activation path to become a pinned runtime policy.
func Promote(p Proposal, decisions []engineering.AuthorityDecision, actor string) (policygovernance.Proposal, error) {
	if err := p.ValidateForPromotion(); err != nil {
		return policygovernance.Proposal{}, err
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return policygovernance.Proposal{}, fmt.Errorf("%w: promotion actor is required", ErrInvalidProposal)
	}
	if p.Provider != nil && strings.EqualFold(actor, strings.TrimSpace(p.Provider.ProviderID)) {
		return policygovernance.Proposal{}, ErrSelfPromotion
	}
	if p.RequiresAuthority() {
		requirement := p.AuthorityRequirement()
		authorized := false
		for _, decision := range decisions {
			if err := decision.Validate(); err != nil {
				return policygovernance.Proposal{}, fmt.Errorf("%w: invalid authority decision: %v", ErrAuthorityRequired, err)
			}
			if decision.RequirementID != requirement.ID || decision.Status != engineering.AuthorityAuthorized {
				continue
			}
			if p.Provider != nil && strings.EqualFold(strings.TrimSpace(decision.Actor), strings.TrimSpace(p.Provider.ProviderID)) {
				return policygovernance.Proposal{}, ErrSelfPromotion
			}
			if !strings.EqualFold(strings.TrimSpace(decision.Actor), actor) {
				continue
			}
			authorized = true
		}
		if !authorized {
			return policygovernance.Proposal{}, fmt.Errorf("%w: %s", ErrAuthorityRequired, requirement.ID)
		}
	}
	governance, err := policygovernance.NewProposal(p.ID, p.Candidate, p.Evidence, p.ProposedBy)
	if err != nil {
		return policygovernance.Proposal{}, fmt.Errorf("%w: policy governance proposal: %v", ErrInvalidProposal, err)
	}
	active, err := governance.Activate(actor)
	if err != nil {
		return policygovernance.Proposal{}, err
	}
	return active, nil
}
