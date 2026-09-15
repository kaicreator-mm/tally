package replan

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/engineering"
	"tally/internal/intelligence"
)

// TriggerKind is a closed set of accepted project events that may justify a
// re-plan. A model's periodic speculation is deliberately not a trigger kind.
type TriggerKind string

const (
	TriggerRequirementChanged     TriggerKind = "requirement_changed"
	TriggerArchitectureInvalidated TriggerKind = "architecture_invalidated"
	TriggerRepeatedTaskFailure    TriggerKind = "repeated_task_failure"
	TriggerRepairBudgetExhausted  TriggerKind = "repair_budget_exhausted"
	TriggerBlockedDependency      TriggerKind = "blocked_dependency"
	TriggerEnvironmentImpossible  TriggerKind = "environment_impossible"
	TriggerVerificationEvidence   TriggerKind = "verification_evidence"
	TriggerExplicitHumanRequest   TriggerKind = "explicit_human_request"
)

var (
	ErrInvalidTrigger     = errors.New("replan: invalid trigger")
	ErrInvalidProposal    = errors.New("replan: invalid proposal")
	ErrStaleParent        = errors.New("replan: stale parent revision")
	ErrAcceptedWork       = errors.New("replan: accepted work not preserved or explicitly invalidated")
	ErrCycleBudget        = errors.New("replan: cycle budget exhausted")
	ErrAuthorityRequired  = errors.New("replan: required authority missing")
	ErrSelfAuthorization  = errors.New("replan: proposal provider cannot self-authorize")
	ErrProviderDisagreement = errors.New("replan: intelligence providers disagree")
)

func (k TriggerKind) valid() bool {
	switch k {
	case TriggerRequirementChanged, TriggerArchitectureInvalidated, TriggerRepeatedTaskFailure,
		TriggerRepairBudgetExhausted, TriggerBlockedDependency, TriggerEnvironmentImpossible,
		TriggerVerificationEvidence, TriggerExplicitHumanRequest:
		return true
	default:
		return false
	}
}

type Trigger struct {
	Kind       TriggerKind            `json:"kind"`
	FactIDs    []engineering.ID       `json:"fact_ids,omitempty"`
	Evidence   []engineering.EvidenceRef `json:"evidence,omitempty"`
	HumanActor string                 `json:"human_actor,omitempty"`
	Provenance engineering.Provenance `json:"provenance"`
}

func (t Trigger) Validate() error {
	if !t.Kind.valid() {
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidTrigger, t.Kind)
	}
	for _, id := range t.FactIDs {
		if err := id.Validate("trigger fact"); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidTrigger, err)
		}
	}
	for _, evidence := range t.Evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidTrigger, err)
		}
	}
	if t.Kind == TriggerExplicitHumanRequest {
		if strings.TrimSpace(t.HumanActor) == "" {
			return fmt.Errorf("%w: explicit human request requires actor", ErrInvalidTrigger)
		}
	} else if len(t.FactIDs) == 0 && len(t.Evidence) == 0 {
		return fmt.Errorf("%w: %s requires accepted fact or evidence", ErrInvalidTrigger, t.Kind)
	}
	if strings.TrimSpace(t.Provenance.Source) == "" || t.Provenance.At.IsZero() {
		return fmt.Errorf("%w: trigger provenance is incomplete", ErrInvalidTrigger)
	}
	return nil
}

type DispositionKind string

const (
	DispositionRetained    DispositionKind = "retained"
	DispositionSuperseded  DispositionKind = "superseded"
	DispositionInvalidated DispositionKind = "invalidated"
	DispositionNew         DispositionKind = "new"
)

func (k DispositionKind) valid() bool {
	switch k {
	case DispositionRetained, DispositionSuperseded, DispositionInvalidated, DispositionNew:
		return true
	default:
		return false
	}
}

type TaskDisposition struct {
	TaskID        engineering.ID `json:"task_id"`
	Kind          DispositionKind `json:"kind"`
	ReplacementID engineering.ID `json:"replacement_id,omitempty"`
	Reason        string         `json:"reason,omitempty"`
}

func (d TaskDisposition) Validate() error {
	if err := d.TaskID.Validate("task disposition"); err != nil {
		return err
	}
	if !d.Kind.valid() {
		return fmt.Errorf("replan: invalid disposition %q for %s", d.Kind, d.TaskID)
	}
	if d.ReplacementID != "" {
		if err := d.ReplacementID.Validate("replacement task"); err != nil {
			return err
		}
		if d.ReplacementID == d.TaskID {
			return errors.New("replan: replacement task must differ from replaced task")
		}
	}
	if (d.Kind == DispositionInvalidated || d.Kind == DispositionSuperseded) && strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("replan: %s disposition for %s requires reason", d.Kind, d.TaskID)
	}
	if d.Kind == DispositionNew && d.ReplacementID != "" {
		return errors.New("replan: new task cannot itself declare a replacement")
	}
	return nil
}

type Impact struct {
	RequirementsChanged      bool `json:"requirements_changed,omitempty"`
	ScopeChanged             bool `json:"scope_changed,omitempty"`
	ArchitectureReplaced     bool `json:"architecture_replaced,omitempty"`
	AcceptedWorkInvalidated  bool `json:"accepted_work_invalidated,omitempty"`
	HighCost                 bool `json:"high_cost,omitempty"`
	HighRisk                 bool `json:"high_risk,omitempty"`
}

func (i Impact) Material() bool {
	return i.RequirementsChanged || i.ScopeChanged || i.ArchitectureReplaced || i.AcceptedWorkInvalidated || i.HighCost || i.HighRisk
}

type Proposal struct {
	ID                 engineering.ID                                      `json:"id"`
	ParentPlanRevision engineering.ID                                      `json:"parent_plan_revision"`
	ParentDAGRevision  engineering.ID                                      `json:"parent_dag_revision"`
	Planning           engineering.PlanningRevision                        `json:"planning_revision"`
	DAG                engineering.TaskDAGProposal                          `json:"task_dag"`
	Contracts          map[engineering.ID]engineering.CompletionContract   `json:"completion_contracts"`
	Dispositions       []TaskDisposition                                   `json:"dispositions"`
	Impact             Impact                                              `json:"impact"`
	Triggers           []Trigger                                           `json:"triggers"`
	Evidence           []engineering.EvidenceRef                           `json:"evidence,omitempty"`
	Provider           intelligence.ProviderProvenance                     `json:"provider"`
	Provenance         engineering.Provenance                              `json:"provenance"`
}

func (p Proposal) ValidateBasic() error {
	if err := p.ID.Validate("replan proposal"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProposal, err)
	}
	if err := p.ParentPlanRevision.Validate("parent planning revision"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProposal, err)
	}
	if err := p.ParentDAGRevision.Validate("parent DAG revision"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProposal, err)
	}
	if p.Planning.ParentRevision != p.ParentPlanRevision {
		return fmt.Errorf("%w: planning parent does not match proposal parent", ErrInvalidProposal)
	}
	if _, err := engineering.NewPlanningRevision(p.Planning.ID, p.Planning.Number, p.Planning.ParentRevision, p.Planning.Strategy, p.Planning.Artifact, p.Planning.Provenance); err != nil {
		return fmt.Errorf("%w: planning revision: %v", ErrInvalidProposal, err)
	}
	if p.DAG.RevisionID == "" || len(p.DAG.Nodes) == 0 {
		return fmt.Errorf("%w: proposed DAG is required", ErrInvalidProposal)
	}
	if len(p.Triggers) == 0 {
		return fmt.Errorf("%w: at least one fact-backed trigger is required", ErrInvalidProposal)
	}
	for _, trigger := range p.Triggers {
		if err := trigger.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidProposal, err)
		}
	}
	for _, evidence := range p.Evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("%w: evidence: %v", ErrInvalidProposal, err)
		}
	}
	seen := map[engineering.ID]bool{}
	for _, disposition := range p.Dispositions {
		if err := disposition.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidProposal, err)
		}
		if seen[disposition.TaskID] {
			return fmt.Errorf("%w: duplicate disposition for %s", ErrInvalidProposal, disposition.TaskID)
		}
		seen[disposition.TaskID] = true
	}
	if len(p.Dispositions) == 0 {
		return fmt.Errorf("%w: task dispositions are required", ErrInvalidProposal)
	}
	if p.Provider.Kind != intelligence.ProviderDirectLLM && p.Provider.Kind != intelligence.ProviderAIRuntime {
		return fmt.Errorf("%w: invalid provider kind %q", ErrInvalidProposal, p.Provider.Kind)
	}
	if strings.TrimSpace(p.Provider.ProviderID) == "" || p.Provider.ObservedAt.IsZero() {
		return fmt.Errorf("%w: provider provenance is incomplete", ErrInvalidProposal)
	}
	if strings.TrimSpace(p.Provenance.Source) == "" || p.Provenance.At.IsZero() {
		return fmt.Errorf("%w: proposal provenance is incomplete", ErrInvalidProposal)
	}
	return nil
}

type CurrentState struct {
	Plan      engineering.ProjectPlan
	DAG       engineering.ActiveTaskDAG
	Contracts map[engineering.ID]engineering.CompletionContract
}

type CyclePolicy struct {
	MaxReplans         uint64
	MaxSameFingerprint uint64
}

func DefaultCyclePolicy() CyclePolicy {
	return CyclePolicy{MaxReplans: 8, MaxSameFingerprint: 2}
}

type CycleLedger struct {
	Activations  uint64
	Fingerprints []string
}

type Decision struct {
	proposalID        engineering.ID
	parentPlan        engineering.ID
	parentDAG         engineering.ID
	planning          engineering.PlanningRevision
	dag               engineering.ActiveTaskDAG
	contracts         map[engineering.ID]engineering.CompletionContract
	dispositions      []TaskDisposition
	impact            Impact
	fingerprint       string
	requiredAuthority []engineering.AuthorityRequirement
	provider          intelligence.ProviderProvenance
	validated         bool
}

func (d Decision) Validated() bool { return d.validated }
func (d Decision) ProposalID() engineering.ID { return d.proposalID }
func (d Decision) ParentPlanRevision() engineering.ID { return d.parentPlan }
func (d Decision) ParentDAGRevision() engineering.ID { return d.parentDAG }
func (d Decision) Planning() engineering.PlanningRevision { return d.planning }
func (d Decision) DAG() engineering.ActiveTaskDAG { return cloneDAG(d.dag) }
func (d Decision) Contracts() map[engineering.ID]engineering.CompletionContract { return cloneContracts(d.contracts) }
func (d Decision) Dispositions() []TaskDisposition { return append([]TaskDisposition(nil), d.dispositions...) }
func (d Decision) Impact() Impact { return d.impact }
func (d Decision) Fingerprint() string { return d.fingerprint }
func (d Decision) RequiredAuthority() []engineering.AuthorityRequirement { return append([]engineering.AuthorityRequirement(nil), d.requiredAuthority...) }
func (d Decision) Provider() intelligence.ProviderProvenance { return d.provider }

type ActivationRecord struct {
	ProposalID        engineering.ID                                    `json:"proposal_id"`
	Fingerprint       string                                            `json:"fingerprint"`
	PreviousPlanning  engineering.PlanningRevision                      `json:"previous_planning"`
	PreviousDAG       engineering.ActiveTaskDAG                         `json:"previous_dag"`
	PreviousContracts map[engineering.ID]engineering.CompletionContract `json:"previous_contracts"`
	ActivatedPlanning engineering.PlanningRevision                      `json:"activated_planning"`
	ActivatedDAG      engineering.ActiveTaskDAG                         `json:"activated_dag"`
	ActivatedContracts map[engineering.ID]engineering.CompletionContract `json:"activated_contracts"`
	Dispositions      []TaskDisposition                                 `json:"dispositions"`
	Impact            Impact                                            `json:"impact"`
	Authority         []engineering.AuthorityDecision                   `json:"authority,omitempty"`
	Actor             string                                            `json:"actor"`
	ActivatedAt       time.Time                                         `json:"activated_at"`
}

func (r ActivationRecord) Validate() error {
	if err := r.ProposalID.Validate("replan proposal"); err != nil {
		return err
	}
	if strings.TrimSpace(r.Fingerprint) == "" || strings.TrimSpace(r.Actor) == "" || r.ActivatedAt.IsZero() {
		return errors.New("replan: activation record fingerprint, actor and time are required")
	}
	if r.ActivatedPlanning.ParentRevision != r.PreviousPlanning.ID || r.ActivatedPlanning.Number != r.PreviousPlanning.Number+1 {
		return errors.New("replan: activation history has invalid planning ancestry")
	}
	if _, err := engineering.ActivateTaskDAG(engineering.TaskDAGProposal{RevisionID: r.PreviousDAG.RevisionID, Nodes: r.PreviousDAG.Nodes}); err != nil {
		return fmt.Errorf("replan: invalid previous DAG in history: %w", err)
	}
	if _, err := engineering.ActivateTaskDAG(engineering.TaskDAGProposal{RevisionID: r.ActivatedDAG.RevisionID, Nodes: r.ActivatedDAG.Nodes}); err != nil {
		return fmt.Errorf("replan: invalid activated DAG in history: %w", err)
	}
	return nil
}

func cloneContracts(in map[engineering.ID]engineering.CompletionContract) map[engineering.ID]engineering.CompletionContract {
	out := make(map[engineering.ID]engineering.CompletionContract, len(in))
	for id, c := range in {
		c.RequiredArtifacts = append([]engineering.ID(nil), c.RequiredArtifacts...)
		c.RequiredEvidence = append([]engineering.ID(nil), c.RequiredEvidence...)
		c.RequiredChecks = append([]engineering.ID(nil), c.RequiredChecks...)
		c.RequiredRecords = append([]engineering.ID(nil), c.RequiredRecords...)
		c.RequiredAuthority = append([]engineering.ID(nil), c.RequiredAuthority...)
		c.RequiredGitOperations = append([]engineering.ID(nil), c.RequiredGitOperations...)
		out[id] = c
	}
	return out
}

func cloneDAG(in engineering.ActiveTaskDAG) engineering.ActiveTaskDAG {
	out := engineering.ActiveTaskDAG{RevisionID: in.RevisionID, Nodes: make([]engineering.TaskNode, len(in.Nodes))}
	for i, n := range in.Nodes {
		n.Dependencies = append([]engineering.ID(nil), n.Dependencies...)
		n.RuntimeRequirements = append([]string(nil), n.RuntimeRequirements...)
		n.EnvironmentRequirements = append([]string(nil), n.EnvironmentRequirements...)
		n.VerificationRequirements = append([]string(nil), n.VerificationRequirements...)
		out.Nodes[i] = n
	}
	return out
}
