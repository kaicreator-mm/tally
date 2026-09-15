package replan

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"tally/internal/intelligence"
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
		Proposal:       f.raw,
		Assessment:     json.RawMessage(`{}`),
		Recommendation: json.RawMessage(`{}`),
		Provenance: intelligence.ProviderProvenance{
			Kind: f.kind, ProviderID: f.id, OperationID: "op-1", ObservedAt: time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
		},
	}, nil
}

func TestGenerateProposalDirectAndAIRuntimeUseSameContract(t *testing.T) {
	current, template := fixture(t)
	input := IntelligenceInput{Planning: current.Plan.AcceptedRevision, ActiveDAG: current.DAG, Contracts: current.Contracts, Triggers: template.Triggers}
	wire := template
	wire.ParentPlanRevision = ""
	wire.ParentDAGRevision = ""
	wire.Triggers = nil
	wire.Provider = intelligence.ProviderProvenance{}
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	direct := &fakeProvider{kind: intelligence.ProviderDirectLLM, id: "direct", raw: raw}
	runtime := &fakeProvider{kind: intelligence.ProviderAIRuntime, id: "runtime", raw: raw}

	a, err := GenerateProposal(context.Background(), direct, input)
	if err != nil {
		t.Fatalf("direct: %v", err)
	}
	b, err := GenerateProposal(context.Background(), runtime, input)
	if err != nil {
		t.Fatalf("runtime: %v", err)
	}
	if direct.last.Operation != intelligence.OperationReplanProposal || runtime.last.Operation != intelligence.OperationReplanProposal {
		t.Fatal("providers did not receive the same replan operation")
	}
	comparison, err := CompareProviders(a, b)
	if err != nil {
		t.Fatalf("equivalent providers disagree: %v", err)
	}
	if !comparison.Agree {
		t.Fatal("expected provider agreement")
	}
}

func TestCompareProvidersEscalatesDisagreement(t *testing.T) {
	_, a := fixture(t)
	b := a
	b.Provider.Kind = intelligence.ProviderAIRuntime
	b.Provider.ProviderID = "runtime"
	b.Planning.Strategy = []string{"materially different"}
	comparison, err := CompareProviders(a, b)
	if !errors.Is(err, ErrProviderDisagreement) {
		t.Fatalf("expected disagreement, got %v", err)
	}
	if comparison.Agree {
		t.Fatal("disagreement cannot be reported as agreement")
	}
}
