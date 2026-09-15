package replan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/intelligence"
)

type OutcomeRef struct {
	ID          string         `json:"id"`
	TaskID      engineering.ID `json:"task_id,omitempty"`
	ChangesetID string         `json:"changeset_id,omitempty"`
	Kind        string         `json:"kind"`
}

type IntelligenceInput struct {
	Requirements       engineering.RequirementsBaseline    `json:"requirements"`
	Planning           engineering.PlanningRevision        `json:"planning_revision"`
	Architecture       engineering.ArchitectureRevision    `json:"architecture"`
	ActiveDAG          engineering.ActiveTaskDAG           `json:"active_task_dag"`
	Contracts          map[engineering.ID]engineering.CompletionContract `json:"completion_contracts"`
	Evidence           []engineering.EvidenceRef           `json:"evidence,omitempty"`
	Outcomes           []OutcomeRef                        `json:"outcomes,omitempty"`
	RuntimeConstraints []string                            `json:"runtime_constraints,omitempty"`
	GitRequirements    []string                            `json:"git_requirements,omitempty"`
	Triggers           []Trigger                           `json:"triggers"`
}

func (in IntelligenceInput) Validate() error {
	if in.Planning.ID == "" || in.ActiveDAG.RevisionID == "" {
		return fmt.Errorf("%w: accepted planning revision and active DAG are required", ErrInvalidProposal)
	}
	if len(in.Triggers) == 0 {
		return fmt.Errorf("%w: intelligence input requires triggers", ErrInvalidTrigger)
	}
	for _, trigger := range in.Triggers {
		if err := trigger.Validate(); err != nil { return err }
	}
	for _, evidence := range in.Evidence {
		if err := evidence.Validate(); err != nil { return err }
	}
	return nil
}

// GenerateProposal uses the frozen intelligence.Provider seam. The provider is
// allowed to propose engineering content, but parent identities, trigger facts
// and provider provenance are pinned by Tally rather than trusted from model
// output.
func GenerateProposal(ctx context.Context, provider intelligence.Provider, in IntelligenceInput) (Proposal, error) {
	if provider == nil {
		return Proposal{}, fmt.Errorf("%w: intelligence provider is required", ErrInvalidProposal)
	}
	if err := in.Validate(); err != nil { return Proposal{}, err }
	rawInput, err := json.Marshal(in)
	if err != nil { return Proposal{}, fmt.Errorf("replan: marshal intelligence input: %w", err) }
	outputShape := json.RawMessage(`{"type":"ReplanProposal","required":["id","planning_revision","task_dag","completion_contracts","dispositions","impact"]}`)
	result, err := provider.Generate(ctx, intelligence.Request{
		Operation: intelligence.OperationReplanProposal,
		Instruction: "Produce a ReplanProposal only. Preserve compatible accepted work, explicitly disposition every old/new task, and do not claim authority or activation.",
		Input: rawInput,
		OutputShape: outputShape,
	})
	if err != nil { return Proposal{}, err }
	var proposal Proposal
	if err := json.Unmarshal(result.Proposal, &proposal); err != nil {
		return Proposal{}, fmt.Errorf("%w: provider proposal JSON: %v", ErrInvalidProposal, err)
	}
	proposal.ParentPlanRevision = in.Planning.ID
	proposal.ParentDAGRevision = in.ActiveDAG.RevisionID
	proposal.Planning.ParentRevision = in.Planning.ID
	proposal.Triggers = append([]Trigger(nil), in.Triggers...)
	proposal.Provider = result.Provenance
	proposal.Provenance = engineering.Provenance{
		Actor: result.Provenance.ProviderID,
		Source: "intelligence:" + string(result.Provenance.Kind),
		Revision: result.Provenance.OperationID,
		At: result.Provenance.ObservedAt,
	}
	if err := proposal.ValidateBasic(); err != nil { return Proposal{}, err }
	if err := evidenceSubset(proposal.Evidence, in.Evidence); err != nil { return Proposal{}, err }
	return proposal, nil
}

func evidenceSubset(proposed, accepted []engineering.EvidenceRef) error {
	allowed := map[engineering.ID]engineering.EvidenceRef{}
	for _, e := range accepted { allowed[e.ID] = e }
	for _, e := range proposed {
		known, ok := allowed[e.ID]
		if !ok || known.Revision != e.Revision || known.Status != e.Status {
			return fmt.Errorf("%w: proposal references non-accepted evidence %s", ErrInvalidProposal, e.ID)
		}
	}
	return nil
}

type ProviderComparison struct {
	Agree        bool   `json:"agree"`
	FingerprintA string `json:"fingerprint_a"`
	FingerprintB string `json:"fingerprint_b"`
	ProviderA    string `json:"provider_a"`
	ProviderB    string `json:"provider_b"`
}

func CompareProviders(a, b Proposal) (ProviderComparison, error) {
	fa, err := Fingerprint(a)
	if err != nil { return ProviderComparison{}, err }
	fb, err := Fingerprint(b)
	if err != nil { return ProviderComparison{}, err }
	out := ProviderComparison{Agree: fa == fb, FingerprintA: fa, FingerprintB: fb, ProviderA: a.Provider.ProviderID, ProviderB: b.Provider.ProviderID}
	if strings.TrimSpace(out.ProviderA) == "" || strings.TrimSpace(out.ProviderB) == "" {
		return ProviderComparison{}, fmt.Errorf("%w: provider identity required", ErrInvalidProposal)
	}
	if !out.Agree { return out, ErrProviderDisagreement }
	return out, nil
}
