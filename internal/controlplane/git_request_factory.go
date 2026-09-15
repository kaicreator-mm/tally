package controlplane

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tally/internal/artifacts"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

// GitTaskRequestFactoryConfig binds the application-local scheduler to the
// existing v1.5.0 TaskExecutionRequest contract without inventing a second
// durable execution queue or task contract. BaseRevision is resolved through
// artifacts.LoadBundle and every produced request is pinned to the observed Git
// commit returned by that loader.
type GitTaskRequestFactoryConfig struct {
	Repo              string
	BaseRevision      string
	RuntimeID         string
	WorkspaceRoot     string
	EnvironmentTarget string
	ModelProfile      string
}

// GitTaskRequestFactory constructs execution requests from the same frozen Git
// authority already consumed by the v1.5 brownfield harness. Workspace
// preparation itself is intentionally outside v1.5.1 (Git authority/workspace
// automation belongs to later slices); each task therefore expects a prepared
// local workspace at <WorkspaceRoot>/<task-id>.
type GitTaskRequestFactory struct {
	cfg        GitTaskRequestFactoryConfig
	loadBundle func(string, string, string) (*artifacts.ExecutionBundle, error)
}

func NewGitTaskRequestFactory(cfg GitTaskRequestFactoryConfig) (*GitTaskRequestFactory, error) {
	switch {
	case strings.TrimSpace(cfg.Repo) == "":
		return nil, errors.New("controlplane: execution content repo is required")
	case strings.TrimSpace(cfg.BaseRevision) == "":
		return nil, errors.New("controlplane: pinned execution base revision is required")
	case strings.TrimSpace(cfg.RuntimeID) == "":
		return nil, errors.New("controlplane: execution runtime id is required")
	case strings.TrimSpace(cfg.WorkspaceRoot) == "":
		return nil, errors.New("controlplane: execution workspace root is required")
	}
	if strings.TrimSpace(cfg.EnvironmentTarget) == "" {
		cfg.EnvironmentTarget = "local"
	}
	return &GitTaskRequestFactory{cfg: cfg, loadBundle: artifacts.LoadBundle}, nil
}

func (f *GitTaskRequestFactory) BuildTaskExecutionRequest(_ context.Context, snapshot ProjectSnapshot, node engineering.TaskNode) (taskexecution.TaskExecutionRequest, error) {
	if snapshot.ProjectID == "" {
		return taskexecution.TaskExecutionRequest{}, errors.New("controlplane: scheduler snapshot is missing project id")
	}
	bundle, err := f.loadBundle(f.cfg.Repo, f.cfg.BaseRevision, string(node.ID))
	if err != nil {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: load frozen execution bundle for %s: %w", node.ID, err)
	}
	if bundle.ProjectID == "" {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: frozen execution bundle for %s has no project identity", node.ID)
	}
	if bundle.ProjectID != string(snapshot.ProjectID) {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: frozen execution project mismatch: snapshot=%s bundle=%s", snapshot.ProjectID, bundle.ProjectID)
	}
	if bundle.Task.Contract.ID != string(node.ID) {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: frozen task identity mismatch: node=%s contract=%s", node.ID, bundle.Task.Contract.ID)
	}

	workspacePath := filepath.Join(f.cfg.WorkspaceRoot, filepath.FromSlash(string(node.ID)))
	info, err := os.Stat(workspacePath)
	if err != nil {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: prepared workspace for %s is unavailable at %s: %w", node.ID, workspacePath, err)
	}
	if !info.IsDir() {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: prepared workspace for %s is not a directory: %s", node.ID, workspacePath)
	}

	request := taskexecution.TaskExecutionRequest{
		TaskID: node.ID,
		FrozenTaskContract: engineering.ArtifactRef{
			ID:       engineering.ID("task-contract:" + string(node.ID)),
			Kind:     engineering.ArtifactTaskContract,
			URI:      "git:" + bundle.SourceCommit + "#" + bundle.Task.Path,
			Revision: bundle.Task.Contract.RevisionHash(),
			Digest:   bundle.Task.Contract.RevisionHash(),
		},
		Runtime: taskexecution.RuntimeRef{ID: f.cfg.RuntimeID},
		Environment: taskexecution.EnvironmentRef{
			Kind:   taskexecution.EnvironmentLocal,
			Target: f.cfg.EnvironmentTarget,
		},
		Workspace: taskexecution.WorkspaceRef{
			Environment:  taskexecution.EnvironmentLocal,
			Target:       f.cfg.EnvironmentTarget,
			Identity:     string(snapshot.ProjectID) + ":" + string(node.ID) + ":" + bundle.SourceCommit,
			Path:         workspacePath,
			BaseRevision: bundle.SourceCommit,
		},
		Policy:      taskexecution.ExecutionPolicy{ModelProfile: f.cfg.ModelProfile},
		Instruction: frozenTaskInstruction(bundle, node),
	}
	if err := request.Validate(); err != nil {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: composed execution request for %s is invalid: %w", node.ID, err)
	}
	return request, nil
}

func frozenTaskInstruction(bundle *artifacts.ExecutionBundle, node engineering.TaskNode) string {
	contract := bundle.Task.Contract
	var b strings.Builder
	fmt.Fprintf(&b, "Execute frozen Tally TaskContract %s at source commit %s.\n", contract.ID, bundle.SourceCommit)
	if strings.TrimSpace(contract.Title) != "" {
		fmt.Fprintf(&b, "Title: %s\n", contract.Title)
	}
	fmt.Fprintf(&b, "Objective: %s\n", contract.Objective)
	fmt.Fprintf(&b, "Profile: %s\n", contract.Profile)
	if len(contract.Scope.AllowedPaths) > 0 {
		fmt.Fprintf(&b, "Allowed paths: %s\n", strings.Join(contract.Scope.AllowedPaths, ", "))
	}
	if len(contract.Scope.ForbiddenPaths) > 0 {
		fmt.Fprintf(&b, "Forbidden paths: %s\n", strings.Join(contract.Scope.ForbiddenPaths, ", "))
	}
	if len(contract.Acceptance) > 0 {
		b.WriteString("Acceptance criteria:\n")
		for _, clause := range contract.Acceptance {
			fmt.Fprintf(&b, "- %s: %s\n", clause.ID, clause.Statement)
		}
	}
	if len(node.RuntimeRequirements) > 0 {
		fmt.Fprintf(&b, "Runtime requirements: %s\n", strings.Join(node.RuntimeRequirements, ", "))
	}
	if len(node.EnvironmentRequirements) > 0 {
		fmt.Fprintf(&b, "Environment requirements: %s\n", strings.Join(node.EnvironmentRequirements, ", "))
	}
	if len(node.VerificationRequirements) > 0 {
		fmt.Fprintf(&b, "Verification requirements: %s\n", strings.Join(node.VerificationRequirements, ", "))
	}
	b.WriteString("Do not change the frozen task scope or acceptance contract. Return an implementation candidate for independent verification.\n")
	return b.String()
}
