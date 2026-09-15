package multiagent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type Role string

const (
	RoleImplementer Role = "implementer"
	RoleReviewer    Role = "reviewer"
	RoleRepairer    Role = "repairer"
	RoleVerifier    Role = "verifier"
	RoleSpecialist  Role = "specialist"
)

func (r Role) Validate() error {
	switch r {
	case RoleImplementer, RoleReviewer, RoleRepairer, RoleVerifier, RoleSpecialist:
		return nil
	default:
		return fmt.Errorf("multiagent: invalid role %q", r)
	}
}

type Independence string

const (
	IndependenceNone    Independence = "none"
	IndependenceRun     Independence = "run"
	IndependenceRuntime Independence = "runtime"
)

func (i Independence) Validate() error {
	switch i {
	case IndependenceNone, IndependenceRun, IndependenceRuntime:
		return nil
	default:
		return fmt.Errorf("multiagent: invalid independence policy %q", i)
	}
}

type WorkflowStatus string

const (
	WorkflowActive      WorkflowStatus = "active"
	WorkflowVerified    WorkflowStatus = "verified"
	WorkflowAccepted    WorkflowStatus = "accepted"
	WorkflowBlocked     WorkflowStatus = "blocked"
	WorkflowInterrupted WorkflowStatus = "interrupted"
)

type RunStatus string

const (
	RunRunning     RunStatus = "running"
	RunPassed      RunStatus = "passed"
	RunRepair      RunStatus = "repair_requested"
	RunBlocked     RunStatus = "blocked"
	RunInterrupted RunStatus = "interrupted"
	RunFailed      RunStatus = "failed"
)

type RoleAssignment struct {
	RuntimeID    string `json:"runtime_id,omitempty"`
	ModelProfile string `json:"model_profile,omitempty"`
}

type TaskPolicy struct {
	MaxRepairAttempts     int                       `json:"max_repair_attempts"`
	ReviewerIndependence  Independence              `json:"reviewer_independence"`
	VerifierIndependence  Independence              `json:"verifier_independence"`
	ConflictKeys          []string                  `json:"conflict_keys,omitempty"`
	Specialists           []string                  `json:"specialists,omitempty"`
	Assignments           map[Role]RoleAssignment   `json:"assignments,omitempty"`
	SpecialistAssignments map[string]RoleAssignment `json:"specialist_assignments,omitempty"`
}

func DefaultTaskPolicy() TaskPolicy {
	return TaskPolicy{
		MaxRepairAttempts:    2,
		ReviewerIndependence: IndependenceRun,
		VerifierIndependence: IndependenceRun,
	}
}

func (p TaskPolicy) Normalize() TaskPolicy {
	if p.MaxRepairAttempts <= 0 {
		p.MaxRepairAttempts = DefaultTaskPolicy().MaxRepairAttempts
	}
	if p.ReviewerIndependence == "" {
		p.ReviewerIndependence = IndependenceRun
	}
	if p.VerifierIndependence == "" {
		p.VerifierIndependence = IndependenceRun
	}
	p.ConflictKeys = uniqueStrings(p.ConflictKeys)
	p.Specialists = uniqueStrings(p.Specialists)
	if p.Assignments == nil {
		p.Assignments = map[Role]RoleAssignment{}
	}
	if p.SpecialistAssignments == nil {
		p.SpecialistAssignments = map[string]RoleAssignment{}
	}
	return p
}

func (p TaskPolicy) Validate() error {
	p = p.Normalize()
	if p.MaxRepairAttempts < 1 {
		return errors.New("multiagent: max repair attempts must be positive")
	}
	if err := p.ReviewerIndependence.Validate(); err != nil {
		return err
	}
	if err := p.VerifierIndependence.Validate(); err != nil {
		return err
	}
	for role := range p.Assignments {
		if err := role.Validate(); err != nil {
			return err
		}
		if role == RoleSpecialist {
			return errors.New("multiagent: specialist assignment must be keyed by specialist name")
		}
	}
	for _, name := range p.Specialists {
		if strings.TrimSpace(name) == "" {
			return errors.New("multiagent: specialist name is required")
		}
	}
	for name := range p.SpecialistAssignments {
		if strings.TrimSpace(name) == "" {
			return errors.New("multiagent: specialist assignment name is required")
		}
	}
	return nil
}

func (p TaskPolicy) Assignment(role Role, specialist string) RoleAssignment {
	p = p.Normalize()
	if role == RoleSpecialist {
		return p.SpecialistAssignments[specialist]
	}
	return p.Assignments[role]
}

func (p TaskPolicy) IndependenceFor(role Role) Independence {
	p = p.Normalize()
	switch role {
	case RoleReviewer:
		return p.ReviewerIndependence
	case RoleVerifier:
		return p.VerifierIndependence
	default:
		return IndependenceNone
	}
}

type Finding struct {
	ID            engineering.ID            `json:"id"`
	Summary       string                    `json:"summary"`
	EvidenceRefs  []engineering.EvidenceRef `json:"evidence_refs,omitempty"`
	ProducerRole  Role                      `json:"producer_role"`
	ProducerRunID string                    `json:"producer_run_id"`
	Resolved      bool                      `json:"resolved"`
}

type RoleRun struct {
	ID         string                            `json:"id"`
	Sequence   int                               `json:"sequence"`
	Role       Role                              `json:"role"`
	Specialist string                            `json:"specialist,omitempty"`
	Status     RunStatus                         `json:"status"`
	StartedAt  time.Time                         `json:"started_at"`
	FinishedAt time.Time                         `json:"finished_at,omitempty"`
	Execution  taskexecution.TaskExecutionResult `json:"execution"`
	Reason     string                            `json:"reason,omitempty"`
}

func (r RoleRun) RuntimeID() string                 { return strings.TrimSpace(r.Execution.Runtime.ID) }
func (r RoleRun) SessionID() agentruntime.SessionID { return r.Execution.AgentClaim.SessionID }
func (r RoleRun) AgentRunID() agentruntime.RunID    { return r.Execution.AgentClaim.RunID }

type WorkflowState struct {
	TaskID          engineering.ID `json:"task_id"`
	Status          WorkflowStatus `json:"status"`
	NextRole        Role           `json:"next_role"`
	SpecialistIndex int            `json:"specialist_index,omitempty"`
	RepairAttempts  int            `json:"repair_attempts"`
	Runs            []RoleRun      `json:"runs,omitempty"`
	Findings        []Finding      `json:"findings,omitempty"`
	Reason          string         `json:"reason,omitempty"`
	Revision        uint64         `json:"revision"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func NewWorkflowState(taskID engineering.ID, now time.Time) (WorkflowState, error) {
	if err := taskID.Validate("multi-agent task"); err != nil {
		return WorkflowState{}, err
	}
	if now.IsZero() {
		return WorkflowState{}, errors.New("multiagent: workflow time is required")
	}
	return WorkflowState{TaskID: taskID, Status: WorkflowActive, NextRole: RoleImplementer, Revision: 1, UpdatedAt: now.UTC()}, nil
}

func (s WorkflowState) Validate() error {
	if err := s.TaskID.Validate("multi-agent task"); err != nil {
		return err
	}
	switch s.Status {
	case WorkflowActive, WorkflowVerified, WorkflowAccepted, WorkflowBlocked, WorkflowInterrupted:
	default:
		return fmt.Errorf("multiagent: invalid workflow status %q", s.Status)
	}
	if s.Status == WorkflowActive || s.Status == WorkflowInterrupted || s.Status == WorkflowBlocked {
		if err := s.NextRole.Validate(); err != nil {
			return err
		}
	}
	if s.Revision == 0 || s.UpdatedAt.IsZero() {
		return errors.New("multiagent: workflow revision and updated_at are required")
	}
	for i, run := range s.Runs {
		if strings.TrimSpace(run.ID) == "" || run.Sequence != i+1 {
			return fmt.Errorf("multiagent: invalid role run sequence at index %d", i)
		}
		if err := run.Role.Validate(); err != nil {
			return err
		}
		if run.Role == RoleSpecialist && strings.TrimSpace(run.Specialist) == "" {
			return errors.New("multiagent: specialist run is missing specialist name")
		}
		if run.StartedAt.IsZero() {
			return errors.New("multiagent: role run start time is required")
		}
	}
	for _, finding := range s.Findings {
		if err := finding.ID.Validate("multi-agent finding"); err != nil {
			return err
		}
		if strings.TrimSpace(finding.Summary) == "" || strings.TrimSpace(finding.ProducerRunID) == "" {
			return errors.New("multiagent: finding summary and producer run are required")
		}
		if err := finding.ProducerRole.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s WorkflowState) CandidateRun() (RoleRun, bool) {
	for i := len(s.Runs) - 1; i >= 0; i-- {
		run := s.Runs[i]
		if (run.Role == RoleImplementer || run.Role == RoleRepairer) && run.Status == RunPassed {
			return run, true
		}
	}
	return RoleRun{}, false
}

func (s WorkflowState) LatestRun(role Role) (RoleRun, bool) {
	for i := len(s.Runs) - 1; i >= 0; i-- {
		if s.Runs[i].Role == role {
			return s.Runs[i], true
		}
	}
	return RoleRun{}, false
}

func (s WorkflowState) Facts() engineering.CompletionFacts {
	facts := engineering.CompletionFacts{State: engineering.NodeComplete}
	seenA := map[engineering.ID]bool{}
	seenE := map[engineering.ID]bool{}
	for _, run := range s.Runs {
		if run.Status != RunPassed {
			continue
		}
		for _, a := range run.Execution.ArtifactRefs {
			if !seenA[a.ID] {
				seenA[a.ID] = true
				facts.Artifacts = append(facts.Artifacts, a)
			}
		}
		for _, e := range run.Execution.EvidenceRefs {
			if !seenE[e.ID] {
				seenE[e.ID] = true
				facts.Evidence = append(facts.Evidence, e)
			}
		}
	}
	return facts
}

func (s WorkflowState) CandidateExecution() taskexecution.TaskExecutionResult {
	if run, ok := s.CandidateRun(); ok {
		return run.Execution
	}
	return taskexecution.TaskExecutionResult{}
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		x := strings.TrimSpace(raw)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
