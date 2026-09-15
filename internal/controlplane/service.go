package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type Service struct {
	Repo     Repository
	Executor taskexecution.AgentExecutor
	Workflow TaskWorkflow
	Harness  TaskHarness
	Actor    string
	Now      func() time.Time

	mu sync.Mutex // serializes project mutations in this process; repository remains truth.
}

func NewService(repo Repository, exec taskexecution.AgentExecutor, workflow TaskWorkflow) (*Service, error) {
	if repo == nil {
		return nil, errors.New("controlplane: repository is required")
	}
	return &Service{Repo: repo, Executor: exec, Workflow: workflow, Actor: "tally-controlplane", Now: time.Now}, nil
}

// NewServiceWithHarness builds the production v1.5 composition in which the
// Engineering Harness owns implementation → verification/repair → acceptance.
// The legacy Executor+TaskWorkflow seam remains for bounded tests/compatibility,
// but production must not execute both paths for one dispatch.
func NewServiceWithHarness(repo Repository, harness TaskHarness) (*Service, error) {
	if harness == nil {
		return nil, errors.New("controlplane: Engineering Harness is required")
	}
	s, err := NewService(repo, nil, nil)
	if err != nil {
		return nil, err
	}
	s.Harness = harness
	return s, nil
}

func (s *Service) Snapshot(ctx context.Context, projectID engineering.ID) (ProjectSnapshot, error) {
	return s.Repo.Load(ctx, projectID)
}

func (s *Service) AcceptRequirements(ctx context.Context, projectID engineering.ID, b engineering.RequirementsBaseline) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	frozen, err := engineering.NewRequirementsBaseline(b.ID, b.Revision, b.Requirements, b.Artifact, b.Provenance)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	snap, err := s.loadOrNew(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.Requirements != nil {
		return ProjectSnapshot{}, ErrImmutableStage
	}
	snap.Requirements = &frozen
	if err := s.record(&snap, "requirements.accepted", "", "lifecycle"); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) AcceptPlan(ctx context.Context, projectID engineering.ID, p engineering.ProjectPlan) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := p.ID.Validate("project plan"); err != nil {
		return ProjectSnapshot{}, err
	}
	r := p.AcceptedRevision
	frozenRevision, err := engineering.NewPlanningRevision(r.ID, r.Number, r.ParentRevision, r.Strategy, r.Artifact, r.Provenance)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	frozen := engineering.ProjectPlan{ID: p.ID, AcceptedRevision: frozenRevision}
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.Requirements == nil {
		return ProjectSnapshot{}, fmt.Errorf("%w: requirements", ErrStageOrder)
	}
	if snap.Plan != nil {
		return ProjectSnapshot{}, ErrImmutableStage
	}
	snap.Plan = &frozen
	if err := s.record(&snap, "plan.accepted", "", "lifecycle"); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) AcceptArchitecture(ctx context.Context, projectID engineering.ID, a engineering.ArchitectureRevision) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	frozen, err := engineering.NewArchitectureRevision(a.ID, a.Kind, a.Summary, a.Constraints, a.Artifact, a.Provenance)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.Requirements == nil || snap.Plan == nil {
		return ProjectSnapshot{}, fmt.Errorf("%w: plan", ErrStageOrder)
	}
	if snap.Architecture != nil {
		return ProjectSnapshot{}, ErrImmutableStage
	}
	snap.Architecture = &frozen
	if err := s.record(&snap, "architecture.accepted", "", "lifecycle"); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) ActivateDAG(ctx context.Context, projectID engineering.ID, p engineering.TaskDAGProposal, contracts map[engineering.ID]engineering.CompletionContract) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.Requirements == nil || snap.Plan == nil || snap.Architecture == nil {
		return ProjectSnapshot{}, fmt.Errorf("%w: architecture", ErrStageOrder)
	}
	if snap.ActiveDAG != nil {
		return ProjectSnapshot{}, ErrImmutableStage
	}
	dag, err := engineering.ActivateTaskDAG(p)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if len(contracts) != len(dag.Nodes) {
		return ProjectSnapshot{}, errors.New("controlplane: every DAG task requires exactly one CompletionContract")
	}
	cp := make(map[engineering.ID]engineering.CompletionContract, len(contracts))
	for _, n := range dag.Nodes {
		c, ok := contracts[n.ID]
		if !ok {
			return ProjectSnapshot{}, fmt.Errorf("controlplane: task %s is missing CompletionContract", n.ID)
		}
		cp[n.ID] = c
	}
	snap.ActiveDAG = &dag
	snap.Contracts = cp
	if err := s.record(&snap, "task_dag.activated", "", "lifecycle"); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) RecordAuthority(ctx context.Context, projectID, nodeID engineering.ID, d engineering.AuthorityDecision) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := d.Validate(); err != nil {
		return ProjectSnapshot{}, err
	}
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, ErrDAGNotActive
	}
	if _, ok := findNode(*snap.ActiveDAG, nodeID); !ok {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: unknown task %s", nodeID)
	}
	snap.AuthorityDecisions = append(snap.AuthorityDecisions, AuthorityRecord{NodeID: nodeID, Decision: d})
	if err := s.record(&snap, "authority."+string(d.Status), nodeID, "authority:"+string(d.RequirementID)); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) RestartTask(ctx context.Context, projectID, taskID engineering.ID) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, ErrDAGNotActive
	}
	node, ok := findNode(*snap.ActiveDAG, taskID)
	if !ok {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: unknown task %s", taskID)
	}
	dag := *snap.ActiveDAG
	if node.State == engineering.TaskRunning {
		// Best-effort cancel of an in-process run. Restart correctness never
		// depends on a resumable Agent session: persisted running is first
		// recovered to interrupted and then explicitly returned to ready.
		if s.Harness != nil {
			_ = s.Harness.Cancel(taskID)
		} else if s.Executor != nil {
			_ = s.Executor.Cancel(taskID)
		}
		dag, err = dag.Transition(taskID, engineering.TaskInterrupted, "")
		if err != nil {
			return ProjectSnapshot{}, err
		}
		if err := s.record(&snap, "task.recovered_interrupted", taskID, "restart"); err != nil {
			return ProjectSnapshot{}, err
		}
	}
	dag, err = dag.Transition(taskID, engineering.TaskReady, "")
	if err != nil {
		return ProjectSnapshot{}, err
	}
	snap.ActiveDAG = &dag
	if err := s.record(&snap, "task.restart.ready", taskID, "restart"); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) loadOrNew(ctx context.Context, id engineering.ID) (ProjectSnapshot, error) {
	if err := id.Validate("project"); err != nil {
		return ProjectSnapshot{}, err
	}
	snap, err := s.Repo.Load(ctx, id)
	if err == nil {
		return snap, nil
	}
	if !errors.Is(err, ErrProjectNotFound) {
		return ProjectSnapshot{}, err
	}
	return ProjectSnapshot{SchemaVersion: SnapshotSchemaVersion, ProjectID: id, Contracts: map[engineering.ID]engineering.CompletionContract{}, UpdatedAt: s.now()}, nil
}

func (s *Service) save(ctx context.Context, snap *ProjectSnapshot) error {
	snap.SchemaVersion = SnapshotSchemaVersion
	snap.UpdatedAt = s.now()
	return s.Repo.Save(ctx, *snap)
}

func (s *Service) record(snap *ProjectSnapshot, kind string, nodeID engineering.ID, source string) error {
	at := s.now()
	actor := strings.TrimSpace(s.Actor)
	if actor == "" {
		actor = "tally-controlplane"
	}
	seq := len(snap.Audit.Records) + 1
	id := engineering.ID(fmt.Sprintf("%s:%06d", snap.ProjectID, seq))
	rec := engineering.ActionRecord{ID: id, NodeID: nodeID, Kind: kind, Actor: actor, At: at, Provenance: engineering.Provenance{Actor: actor, Source: source, At: at}}
	next, err := snap.Audit.Append(rec)
	if err != nil {
		return err
	}
	snap.Audit = next
	return nil
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func findNode(d engineering.ActiveTaskDAG, id engineering.ID) (engineering.TaskNode, bool) {
	for _, n := range d.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return engineering.TaskNode{}, false
}

func classifyError(err error) string {
	if err == nil {
		return "unknown"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	return "operational_failure"
}
