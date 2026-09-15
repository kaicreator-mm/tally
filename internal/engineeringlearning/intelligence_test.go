package engineeringlearning

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"tally/internal/intelligence"
	"tally/internal/policygovernance"
)

type fakeProvider struct {
	kind intelligence.ProviderKind
	id   string
	raw  json.RawMessage
	last intelligence.Request
}

func (f *fakeProvider) Generate(_ context.Context, req intelligence.Request) (intelligence.Result, error) {
	if err := req.Validate(); err != nil {
		return intelligence.Result{}, err
	}
	f.last = req
	return intelligence.Result{
		Proposal: f.raw,
		Assessment: json.RawMessage(`{}`),
		Recommendation: json.RawMessage(`{}`),
		Provenance: intelligence.ProviderProvenance{
			Kind: f.kind,
			ProviderID: f.id,
			OperationID: "op-rule",
			ObservedAt: time.Date(2026, 9, 15, 10, 20, 0, 0, time.UTC),
		},
	}, nil
}

func TestGenerateProposalDirectAndAIRuntimeUseSameContract(t *testing.T) {
	wire, err := json.Marshal(intelligenceWireProposal{
		ID: "rule-ai",
		CandidateYAML: testPolicyYAML,
		Confidence: 0.82,
		Conflicts: []Conflict{{RuleID: "POL-OLD", Reason: "reviewed", Resolved: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := evidence(t, policygovernance.EvidenceOutcome, "out-1")
	input := IntelligenceInput{
		SupportingOutcomes: []string{"out-1"},
		Evidence: []policygovernance.EvidenceRef{outcome},
		ApplicableScope: "tally/project",
		Shared: true,
		HighImpact: true,
	}
	direct := &fakeProvider{kind: intelligence.ProviderDirectLLM, id: "direct", raw: wire}
	runtime := &fakeProvider{kind: intelligence.ProviderAIRuntime, id: "runtime", raw: wire}

	a, err := GenerateProposal(context.Background(), direct, input)
	if err != nil {
		t.Fatalf("direct: %v", err)
	}
	b, err := GenerateProposal(context.Background(), runtime, input)
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	if direct.last.Operation != intelligence.OperationEngineeringRuleProposal || runtime.last.Operation != intelligence.OperationEngineeringRuleProposal {
		t.Fatal("providers did not receive the same Engineering Rule operation")
	}
	if a.Candidate.Revision() != b.Candidate.Revision() || a.Confidence != b.Confidence {
		t.Fatal("provider substitution changed typed Engineering Rule proposal semantics")
	}
	if a.Provider == nil || b.Provider == nil || a.Provider.Kind == b.Provider.Kind {
		t.Fatal("provider provenance was not preserved independently")
	}
	if len(a.SupportingOutcomes) != 1 || a.SupportingOutcomes[0] != "out-1" || a.ApplicableScope != input.ApplicableScope {
		t.Fatal("accepted outcome/scope inputs were not pinned into proposal")
	}
	if !a.RequiresAuthority() || !b.RequiresAuthority() {
		t.Fatal("AI rule proposals must require independent authority")
	}
}
