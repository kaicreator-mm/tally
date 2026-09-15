package engineeringlearning

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/intelligence"
	"tally/internal/policygovernance"
)

type IntelligenceInput struct {
	SupportingOutcomes []string                       `json:"supporting_outcomes"`
	Evidence           []policygovernance.EvidenceRef `json:"evidence"`
	ApplicableScope    string                         `json:"applicable_scope"`
	Shared             bool                           `json:"shared,omitempty"`
	HighImpact         bool                           `json:"high_impact,omitempty"`
	ExistingRuleIDs    []string                       `json:"existing_rule_ids,omitempty"`
}

type intelligenceWireProposal struct {
	ID            string     `json:"id"`
	CandidateYAML string     `json:"candidate_yaml"`
	Confidence    float64    `json:"confidence"`
	Conflicts     []Conflict `json:"conflicts,omitempty"`
}

func (in IntelligenceInput) Validate() error {
	if len(in.SupportingOutcomes) == 0 || len(in.Evidence) == 0 {
		return fmt.Errorf("%w: outcomes and evidence are required", ErrInvalidProposal)
	}
	if strings.TrimSpace(in.ApplicableScope) == "" {
		return fmt.Errorf("%w: applicable scope is required", ErrInvalidProposal)
	}
	allowedOutcomes := map[string]bool{}
	for _, id := range in.SupportingOutcomes {
		id = strings.TrimSpace(id)
		if id == "" || allowedOutcomes[id] {
			return fmt.Errorf("%w: supporting outcome ids must be unique", ErrInvalidProposal)
		}
		allowedOutcomes[id] = true
	}
	backed := map[string]bool{}
	for _, evidence := range in.Evidence {
		if !evidence.Kind.Valid() || strings.TrimSpace(evidence.Ref) == "" {
			return fmt.Errorf("%w: invalid evidence", ErrInvalidProposal)
		}
		if evidence.Kind == policygovernance.EvidenceOutcome {
			backed[evidence.Ref] = true
		}
	}
	for id := range allowedOutcomes {
		if !backed[id] {
			return fmt.Errorf("%w: supporting outcome %s has no outcome evidence", ErrInvalidProposal, id)
		}
	}
	return nil
}

// GenerateProposal lets Direct LLM and AI Runtime use the same frozen Provider
// contract. Outcome/evidence/scope/impact inputs are pinned by Tally; the model
// may propose only rule content, confidence and conflict analysis.
func GenerateProposal(ctx context.Context, provider intelligence.Provider, in IntelligenceInput) (Proposal, error) {
	if provider == nil {
		return Proposal{}, fmt.Errorf("%w: intelligence provider is required", ErrInvalidProposal)
	}
	if err := in.Validate(); err != nil {
		return Proposal{}, err
	}
	rawInput, err := json.Marshal(in)
	if err != nil {
		return Proposal{}, fmt.Errorf("engineeringlearning: marshal intelligence input: %w", err)
	}
	result, err := provider.Generate(ctx, intelligence.Request{
		Operation: intelligence.OperationEngineeringRuleProposal,
		Instruction: "Propose a Tally Engineering Rule only. Return canonical policy YAML, confidence and conflict analysis. Do not claim authority, promotion or runtime activation.",
		Input: rawInput,
		OutputShape: json.RawMessage([]byte("{\"type\":\"object\",\"title\":\"EngineeringRuleProposal\"}")),
	})
	if err != nil {
		return Proposal{}, err
	}
	if err := result.Validate(); err != nil {
		return Proposal{}, err
	}
	var wire intelligenceWireProposal
	if err := json.Unmarshal(result.Proposal, &wire); err != nil {
		return Proposal{}, fmt.Errorf("%w: provider proposal JSON: %v", ErrInvalidProposal, err)
	}
	candidate, err := policygovernance.NewCandidate([]byte(wire.CandidateYAML))
	if err != nil {
		return Proposal{}, fmt.Errorf("%w: candidate: %v", ErrInvalidProposal, err)
	}
	providerProvenance := result.Provenance
	return NewProposal(
		wire.ID,
		candidate,
		in.Evidence,
		in.SupportingOutcomes,
		in.ApplicableScope,
		wire.Confidence,
		wire.Conflicts,
		engineering.Provenance{
			Actor: result.Provenance.ProviderID,
			Source: "intelligence:" + string(result.Provenance.Kind),
			Revision: result.Provenance.OperationID,
			At: result.Provenance.ObservedAt,
		},
		result.Provenance.ProviderID,
		&providerProvenance,
		in.Shared,
		in.HighImpact,
	)
}
