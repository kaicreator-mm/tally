package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Operation string

const (
	OperationRequirementsAnalysis Operation = "requirements_analysis"
	OperationProjectPlanning      Operation = "project_planning"
	OperationArchitectureAnalysis Operation = "architecture_analysis"
	OperationTaskDecomposition    Operation = "task_decomposition"
	OperationTaskDAGProposal      Operation = "task_dag_proposal"
	OperationRiskAnalysis         Operation = "risk_analysis"
	OperationReview               Operation = "review"
	OperationDiagnosis            Operation = "diagnosis"
	OperationRepairAdvice         Operation = "repair_advice"
)

type ProviderKind string

const (
	ProviderDirectLLM ProviderKind = "direct_llm"
	ProviderAIRuntime ProviderKind = "ai_runtime"
)

type Request struct {
	Operation   Operation       `json:"operation"`
	Instruction string          `json:"instruction"`
	Input       json.RawMessage `json:"input"`
	OutputShape json.RawMessage `json:"output_shape,omitempty"`
}

func (r Request) Validate() error {
	if strings.TrimSpace(string(r.Operation)) == "" || strings.TrimSpace(r.Instruction) == "" {
		return errors.New("intelligence: operation and instruction are required")
	}
	if len(r.Input) == 0 || !json.Valid(r.Input) {
		return errors.New("intelligence: input must be valid JSON")
	}
	if len(r.OutputShape) > 0 && !json.Valid(r.OutputShape) {
		return errors.New("intelligence: output shape must be valid JSON")
	}
	return nil
}

type ProviderProvenance struct {
	Kind        ProviderKind `json:"kind"`
	ProviderID  string       `json:"provider_id"`
	OperationID string       `json:"operation_id,omitempty"`
	ObservedAt  time.Time    `json:"observed_at"`
}

type Result struct {
	Proposal       json.RawMessage    `json:"proposal"`
	Assessment     json.RawMessage    `json:"assessment,omitempty"`
	Recommendation json.RawMessage    `json:"recommendation,omitempty"`
	Provenance     ProviderProvenance `json:"provenance"`
}

func (r Result) Validate() error {
	for name, raw := range map[string]json.RawMessage{"proposal": r.Proposal, "assessment": r.Assessment, "recommendation": r.Recommendation} {
		if len(raw) == 0 || !json.Valid(raw) {
			return fmt.Errorf("intelligence: %s must be valid JSON", name)
		}
	}
	if r.Provenance.Kind != ProviderDirectLLM && r.Provenance.Kind != ProviderAIRuntime {
		return fmt.Errorf("intelligence: invalid provider kind %q", r.Provenance.Kind)
	}
	if strings.TrimSpace(r.Provenance.ProviderID) == "" || r.Provenance.ObservedAt.IsZero() {
		return errors.New("intelligence: provider provenance is incomplete")
	}
	return nil
}

type Provider interface {
	Generate(context.Context, Request) (Result, error)
}

// Invoker is the transport-sized seam consumed by ProviderAdapter. Composition
// roots adapt the legacy direct LLM or AI Runtime transport to this contract.
// It deliberately has no project state, scheduling or acceptance methods.
type Invoker interface {
	Invoke(context.Context, Request) (InvocationResult, error)
}

type InvocationResult struct {
	Proposal       json.RawMessage
	Assessment     json.RawMessage
	Recommendation json.RawMessage
	OperationID    string
}

type ProviderAdapter struct {
	kind ProviderKind
	id   string
	now  func() time.Time
	inv  Invoker
}

func NewDirectProvider(id string, inv Invoker) (*ProviderAdapter, error) {
	return newProvider(ProviderDirectLLM, id, inv)
}

func NewAIRuntimeProvider(id string, inv Invoker) (*ProviderAdapter, error) {
	return newProvider(ProviderAIRuntime, id, inv)
}

func newProvider(kind ProviderKind, id string, inv Invoker) (*ProviderAdapter, error) {
	if strings.TrimSpace(id) == "" || inv == nil {
		return nil, errors.New("intelligence: provider id and invoker are required")
	}
	return &ProviderAdapter{kind: kind, id: id, now: time.Now, inv: inv}, nil
}

func (p *ProviderAdapter) Generate(ctx context.Context, req Request) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	raw, err := p.inv.Invoke(ctx, req)
	if err != nil {
		return Result{}, err
	}
	out := Result{
		Proposal:       raw.Proposal,
		Assessment:     raw.Assessment,
		Recommendation: raw.Recommendation,
		Provenance:     ProviderProvenance{Kind: p.kind, ProviderID: p.id, OperationID: raw.OperationID, ObservedAt: p.now().UTC()},
	}
	if err := out.Validate(); err != nil {
		return Result{}, err
	}
	return out, nil
}
