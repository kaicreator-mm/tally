package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeInvoker struct {
	out InvocationResult
	err error
}

func (f fakeInvoker) Invoke(context.Context, Request) (InvocationResult, error) { return f.out, f.err }

func conformance(t *testing.T, makeProvider func(Invoker) (*ProviderAdapter, error), kind ProviderKind) {
	t.Helper()
	p, err := makeProvider(fakeInvoker{out: InvocationResult{
		Proposal:       json.RawMessage(`{"tasks":["a"]}`),
		Assessment:     json.RawMessage(`{"risk":"low"}`),
		Recommendation: json.RawMessage(`{"action":"accept_proposal_for_validation"}`),
		OperationID:    "op-1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	p.now = func() time.Time { return time.Unix(1, 0) }
	got, err := p.Generate(context.Background(), Request{Operation: OperationTaskDAGProposal, Instruction: "propose", Input: json.RawMessage(`{"intent":"x"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Provenance.Kind != kind || got.Provenance.ProviderID != "provider" || got.Provenance.OperationID != "op-1" {
		t.Fatalf("bad provenance: %+v", got.Provenance)
	}
	if string(got.Proposal) != `{"tasks":["a"]}` {
		t.Fatalf("proposal=%s", got.Proposal)
	}
}

func TestDirectAndAIRuntimeShareContract(t *testing.T) {
	conformance(t, func(i Invoker) (*ProviderAdapter, error) { return NewDirectProvider("provider", i) }, ProviderDirectLLM)
	conformance(t, func(i Invoker) (*ProviderAdapter, error) { return NewAIRuntimeProvider("provider", i) }, ProviderAIRuntime)
}

func TestProviderErrorDoesNotFabricateResult(t *testing.T) {
	boom := errors.New("provider unavailable")
	p, _ := NewAIRuntimeProvider("provider", fakeInvoker{err: boom})
	got, err := p.Generate(context.Background(), Request{Operation: OperationReview, Instruction: "review", Input: json.RawMessage(`{}`)})
	if !errors.Is(err, boom) {
		t.Fatalf("err=%v", err)
	}
	if len(got.Proposal) != 0 || got.Provenance.ProviderID != "" {
		t.Fatalf("fabricated result: %+v", got)
	}
}

func TestProviderRejectsMalformedOutput(t *testing.T) {
	p, _ := NewDirectProvider("provider", fakeInvoker{out: InvocationResult{Proposal: json.RawMessage(`not-json`)}})
	if _, err := p.Generate(context.Background(), Request{Operation: OperationDiagnosis, Instruction: "diagnose", Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("expected invalid proposal rejection")
	}
}
