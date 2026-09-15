package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/engineering"
	"tally/internal/replan"
	"tally/internal/taskexecution"
)

const SnapshotSchemaVersion = 1

var (
	ErrProjectNotFound     = errors.New("controlplane: project not found")
	ErrStageOrder          = errors.New("controlplane: lifecycle stage prerequisite missing")
	ErrImmutableStage      = errors.New("controlplane: accepted lifecycle stage is immutable in v1.5.0")
	ErrDAGNotActive        = errors.New("controlplane: active task DAG is not available")
	ErrTaskNotReady        = errors.New("controlplane: task is not ready")
	ErrWorkflowUnavailable = errors.New("controlplane: verification/review workflow unavailable")
)

type AuthorityRecord struct {
	NodeID   engineering.ID                `json:"node_id"`
	Decision engineering.AuthorityDecision `json:"decision"`
}

type ProjectSnapshot struct {
	SchemaVersion      int                                               `json:"schema_version"`
	ProjectID          engineering.ID                                    `json:"project_id"`
	Requirements       *engineering.RequirementsBaseline                 `json:"requirements,omitempty"`
	Plan               *engineering.ProjectPlan                          `json:"plan,omitempty"`
	Architecture       *engineering.ArchitectureRevision                 `json:"architecture,omitempty"`
	ActiveDAG          *engineering.ActiveTaskDAG                        `json:"active_task_dag,omitempty"`
	Contracts          map[engineering.ID]engineering.CompletionContract `json:"completion_contracts,omitempty"`
	AuthorityDecisions []AuthorityRecord                                 `json:"authority_decisions,omitempty"`
	ReplanHistory      []replan.ActivationRecord                         `json:"replan_history,omitempty"`
	Audit              engineering.AuditLog                              `json:"audit"`
	UpdatedAt          time.Time                                         `json:"updated_at"`
}

type Repository interface {
	Load(context.Context, engineering.ID) (ProjectSnapshot, error)
	Save(context.Context, ProjectSnapshot) error
}

type TaskWorkflowResult struct {
	Accepted bool                        `json:"accepted"`
	Reason   string                      `json:"reason,omitempty"`
	Facts    engineering.CompletionFacts `json:"facts"`
}

type TaskWorkflow interface {
	VerifyAndAccept(context.Context, engineering.ID, taskexecution.TaskExecutionResult) (TaskWorkflowResult, error)
}

// HarnessResult is the project-level result of one task run through the
// Engineering Harness. The Harness, not an Agent claim, owns the transition
// from implementation to verification/repair/acceptance. Interrupted is a
// first-class execution outcome; a non-accepted non-interrupted result is a
// governed block/escalation and is not a Go error.
type HarnessResult struct {
	Execution   taskexecution.TaskExecutionResult `json:"execution"`
	Accepted    bool                              `json:"accepted"`
	Interrupted bool                              `json:"interrupted,omitempty"`
	Reason      string                            `json:"reason,omitempty"`
	Facts       engineering.CompletionFacts       `json:"facts"`
}

// TaskHarness is the production v1.5 task-level control-loop port. Its
// implementation may compose the brownfield closed loop, but the Control
// Plane only consumes the bounded outcome and CompletionFacts.
type TaskHarness interface {
	Run(context.Context, engineering.ID, taskexecution.TaskExecutionRequest) (HarnessResult, error)
	Cancel(engineering.ID) error
}

type DispatchResult struct {
	Execution  taskexecution.TaskExecutionResult `json:"execution"`
	Workflow   *TaskWorkflowResult               `json:"workflow,omitempty"`
	Completion *engineering.CompletionResult     `json:"completion,omitempty"`
	Snapshot   ProjectSnapshot                   `json:"snapshot"`
}

func validateSnapshot(s ProjectSnapshot) error {
	if s.SchemaVersion != SnapshotSchemaVersion {
		return fmt.Errorf("controlplane: unsupported snapshot schema %d", s.SchemaVersion)
	}
	if err := s.ProjectID.Validate("project"); err != nil {
		return err
	}
	if s.Requirements != nil {
		if _, err := engineering.NewRequirementsBaseline(s.Requirements.ID, s.Requirements.Revision, s.Requirements.Requirements, s.Requirements.Artifact, s.Requirements.Provenance); err != nil {
			return fmt.Errorf("controlplane: invalid requirements snapshot: %w", err)
		}
	}
	if s.Plan != nil {
		if s.Requirements == nil {
			return fmt.Errorf("%w: plan requires requirements", ErrStageOrder)
		}
		if err := s.Plan.ID.Validate("project plan"); err != nil {
			return err
		}
		r := s.Plan.AcceptedRevision
		if _, err := engineering.NewPlanningRevision(r.ID, r.Number, r.ParentRevision, r.Strategy, r.Artifact, r.Provenance); err != nil {
			return fmt.Errorf("controlplane: invalid planning snapshot: %w", err)
		}
	}
	if s.Architecture != nil {
		if s.Plan == nil {
			return fmt.Errorf("%w: architecture requires plan", ErrStageOrder)
		}
		a := s.Architecture
		if _, err := engineering.NewArchitectureRevision(a.ID, a.Kind, a.Summary, a.Constraints, a.Artifact, a.Provenance); err != nil {
			return fmt.Errorf("controlplane: invalid architecture snapshot: %w", err)
		}
	}
	if s.ActiveDAG != nil {
		if s.Architecture == nil {
			return fmt.Errorf("%w: active DAG requires architecture", ErrStageOrder)
		}
		if _, err := engineering.ActivateTaskDAG(engineering.TaskDAGProposal{RevisionID: s.ActiveDAG.RevisionID, Nodes: s.ActiveDAG.Nodes}); err != nil {
			return fmt.Errorf("controlplane: invalid active DAG structure: %w", err)
		}
		for _, n := range s.ActiveDAG.Nodes {
			switch n.State {
			case engineering.TaskPending, engineering.TaskReady, engineering.TaskRunning, engineering.TaskBlocked, engineering.TaskInterrupted, engineering.TaskAccepted, engineering.TaskDone:
			default:
				return fmt.Errorf("controlplane: task %s has invalid persisted state %q", n.ID, n.State)
			}
			if n.State == engineering.TaskBlocked && strings.TrimSpace(n.BlockedReason) == "" {
				return fmt.Errorf("controlplane: blocked task %s is missing a reason", n.ID)
			}
			if _, ok := s.Contracts[n.ID]; !ok {
				return fmt.Errorf("controlplane: task %s is missing CompletionContract", n.ID)
			}
		}
	}
	if err := validateReplanHistory(s); err != nil {
		return err
	}
	for _, d := range s.AuthorityDecisions {
		if err := d.NodeID.Validate("authority node"); err != nil {
			return err
		}
		if err := d.Decision.Validate(); err != nil {
			return fmt.Errorf("controlplane: invalid authority decision: %w", err)
		}
		// Before v1.5.5 every authority record had to reference the current DAG.
		// Re-plan can explicitly supersede/invalidate a task, so durable authority
		// history is valid if the task exists in either current or retained DAG history.
		if !snapshotHasTask(s, d.NodeID) {
			return fmt.Errorf("controlplane: authority decision references unknown task %s", d.NodeID)
		}
	}
	for _, r := range s.Audit.Records {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("controlplane: invalid audit record: %w", err)
		}
	}
	if s.UpdatedAt.IsZero() {
		return errors.New("controlplane: updated_at is required")
	}
	return nil
}

func validateReplanHistory(s ProjectSnapshot) error {
	for i, record := range s.ReplanHistory {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("controlplane: invalid replan history[%d]: %w", i, err)
		}
		if i > 0 {
			previous := s.ReplanHistory[i-1]
			if record.PreviousPlanning.ID != previous.ActivatedPlanning.ID || record.PreviousDAG.RevisionID != previous.ActivatedDAG.RevisionID {
				return fmt.Errorf("controlplane: replan history[%d] does not continue previous activation", i)
			}
		}
	}
	if len(s.ReplanHistory) == 0 {
		return nil
	}
	last := s.ReplanHistory[len(s.ReplanHistory)-1]
	if s.Plan == nil || s.ActiveDAG == nil {
		return errors.New("controlplane: replan history requires current plan and DAG")
	}
	if s.Plan.AcceptedRevision.ID != last.ActivatedPlanning.ID || s.ActiveDAG.RevisionID != last.ActivatedDAG.RevisionID {
		return errors.New("controlplane: current plan/DAG do not match latest replan activation")
	}
	return nil
}

func snapshotHasTask(s ProjectSnapshot, id engineering.ID) bool {
	if s.ActiveDAG != nil {
		if _, ok := findNode(*s.ActiveDAG, id); ok {
			return true
		}
	}
	for _, record := range s.ReplanHistory {
		if _, ok := findNode(record.PreviousDAG, id); ok {
			return true
		}
		if _, ok := findNode(record.ActivatedDAG, id); ok {
			return true
		}
	}
	return false
}
