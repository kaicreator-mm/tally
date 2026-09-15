package taskexecution

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
)

type EnvironmentKind string

const (
	EnvironmentLocal  EnvironmentKind = "local"
	EnvironmentSSH    EnvironmentKind = "ssh"
	EnvironmentDocker EnvironmentKind = "docker"
)

type EnvironmentCapability string

const (
	CapabilityWorkspaceRead  EnvironmentCapability = "workspace.read"
	CapabilityWorkspaceWrite EnvironmentCapability = "workspace.write"
	CapabilityProcess        EnvironmentCapability = "process"
	CapabilityCancel         EnvironmentCapability = "cancel"
)

type ExecutionTarget struct {
	ID   string          `json:"id"`
	Kind EnvironmentKind `json:"kind"`
}

type WorkspaceRef struct {
	Environment  EnvironmentKind `json:"environment"`
	Target       string          `json:"target"`
	Identity     string          `json:"identity"`
	Path         string          `json:"path,omitempty"`
	BaseRevision string          `json:"base_revision"`
}

func (w WorkspaceRef) Validate() error {
	if w.Environment == "" || strings.TrimSpace(w.Target) == "" || strings.TrimSpace(w.Identity) == "" || strings.TrimSpace(w.BaseRevision) == "" {
		return errors.New("taskexecution: environment, target, workspace identity and base revision are required")
	}
	return nil
}

// ResolvedWorkspace preserves LocalPath for v1.5.0 compatibility while
// Location is the path/locator meaningful inside the selected environment.
type ResolvedWorkspace struct {
	LocalPath string
	Location  string
}

func (w ResolvedWorkspace) RuntimeLocation(kind EnvironmentKind) (string, error) {
	if strings.TrimSpace(w.Location) != "" {
		return w.Location, nil
	}
	if kind == EnvironmentLocal && strings.TrimSpace(w.LocalPath) != "" {
		return w.LocalPath, nil
	}
	return "", errors.New("taskexecution: resolved workspace has no environment-local location")
}

type ExecutionEnvironment interface {
	Kind() EnvironmentKind
	Capabilities() []EnvironmentCapability
	Resolve(context.Context, WorkspaceRef) (ResolvedWorkspace, error)
}

var ErrUnsupportedEnvironment = errors.New("taskexecution: unsupported execution environment")

type LocalEnvironment struct{}

func (LocalEnvironment) Kind() EnvironmentKind { return EnvironmentLocal }
func (LocalEnvironment) Capabilities() []EnvironmentCapability {
	return []EnvironmentCapability{CapabilityCancel, CapabilityProcess, CapabilityWorkspaceRead, CapabilityWorkspaceWrite}
}
func (LocalEnvironment) Resolve(_ context.Context, w WorkspaceRef) (ResolvedWorkspace, error) {
	if err := w.Validate(); err != nil {
		return ResolvedWorkspace{}, err
	}
	if w.Environment != EnvironmentLocal {
		return ResolvedWorkspace{}, ErrUnsupportedEnvironment
	}
	if strings.TrimSpace(w.Path) == "" {
		return ResolvedWorkspace{}, errors.New("taskexecution: local workspace path is required at Local compatibility edge")
	}
	return ResolvedWorkspace{LocalPath: w.Path, Location: w.Path}, nil
}

type CapabilityRequirement struct {
	Runtime     agentruntime.CapabilityRequirement `json:"-"`
	Environment []EnvironmentCapability            `json:"environment,omitempty"`
}

type RuntimeRef struct{ ID string `json:"id"` }
type EnvironmentRef struct {
	Kind   EnvironmentKind `json:"kind"`
	Target string          `json:"target"`
}

type ExecutionPolicy struct {
	ModelProfile    string                 `json:"model_profile,omitempty"`
	ResumeSession   agentruntime.SessionID `json:"resume_session,omitempty"`
	FallbackToFresh bool                   `json:"fallback_to_fresh"`
}

type TaskExecutionRequest struct {
	TaskID               engineering.ID            `json:"task_id"`
	FrozenTaskContract   engineering.ArtifactRef   `json:"frozen_task_contract"`
	EngineeringContext   []engineering.ArtifactRef `json:"engineering_context,omitempty"`
	Runtime              RuntimeRef                `json:"runtime"`
	Environment          EnvironmentRef            `json:"environment"`
	Workspace            WorkspaceRef              `json:"workspace"`
	RequiredCapabilities CapabilityRequirement     `json:"required_capabilities"`
	Policy               ExecutionPolicy           `json:"policy"`
	Instruction          string                    `json:"instruction"`
}

func (r TaskExecutionRequest) Validate() error {
	if err := r.TaskID.Validate("task execution"); err != nil {
		return err
	}
	if err := r.FrozenTaskContract.Validate(); err != nil {
		return err
	}
	if err := r.Workspace.Validate(); err != nil {
		return err
	}
	if r.Environment.Kind != r.Workspace.Environment {
		return errors.New("taskexecution: environment reference does not match workspace")
	}
	if strings.TrimSpace(r.Instruction) == "" {
		return errors.New("taskexecution: instruction is required")
	}
	return nil
}

type ExecutionStatus string

const (
	ExecutionCompleted   ExecutionStatus = "completed"
	ExecutionFailed      ExecutionStatus = "failed"
	ExecutionInterrupted ExecutionStatus = "interrupted"
)

type TaskExecutionResult struct {
	AgentClaim   agentruntime.AgentRunResult `json:"agent_claim"`
	CandidateRef string                      `json:"candidate_ref,omitempty"`
	ArtifactRefs []engineering.ArtifactRef   `json:"artifact_refs,omitempty"`
	EvidenceRefs []engineering.EvidenceRef   `json:"evidence_refs,omitempty"`
	Workspace    WorkspaceRef                `json:"workspace"`
	Runtime      RuntimeRef                  `json:"runtime"`
	Environment  EnvironmentRef              `json:"environment"`
	Status       ExecutionStatus             `json:"status"`
}

type AgentExecutor interface {
	Execute(context.Context, TaskExecutionRequest) (TaskExecutionResult, error)
	Cancel(engineering.ID) error
}

// EnvironmentCapabilityError makes pre-dispatch rejection classifiable.
type EnvironmentCapabilityError struct{ Missing []EnvironmentCapability }

func (e *EnvironmentCapabilityError) Error() string {
	return fmt.Sprintf("taskexecution: missing environment capabilities: %v", e.Missing)
}

func checkEnvironmentCapabilities(env ExecutionEnvironment, required []EnvironmentCapability) error {
	have := map[EnvironmentCapability]bool{}
	for _, c := range env.Capabilities() {
		have[c] = true
	}
	var missing []EnvironmentCapability
	for _, c := range required {
		if !have[c] {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return &EnvironmentCapabilityError{Missing: missing}
	}
	return nil
}
