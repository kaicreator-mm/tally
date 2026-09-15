package controlplane

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

// GitTaskRequestFactoryConfig binds the application-local scheduler to the
// existing v1.5.0 TaskExecutionRequest contract without inventing a second
// durable execution queue or task contract. BaseRevision may be any Git
// revision expression understood by the existing artifact loader; the factory
// resolves it on the first identity-valid request and pins every later request
// to that exact observed source commit.
type GitTaskRequestFactoryConfig struct {
	Repo              string
	BaseRevision      string
	RuntimeID         string
	WorkspaceRoot     string
	EnvironmentKind   taskexecution.EnvironmentKind
	EnvironmentTarget string
	ModelProfile      string
}

// GitTaskRequestFactory constructs execution requests from the same frozen Git
// authority already consumed by the v1.5 brownfield harness. The control-plane
// checkout used to load the frozen ExecutionBundle remains local authority, but
// the execution workspace locator belongs to the selected Environment. Local
// workspaces are checked on the host; SSH/Docker locators are verified later by
// the selected ExecutionEnvironment and are never os.Stat'ed by Core.
type GitTaskRequestFactory struct {
	cfg        GitTaskRequestFactoryConfig
	loadBundle func(string, string, string) (*artifacts.ExecutionBundle, error)

	pinMu        sync.Mutex
	pinnedCommit string
}

func NewGitTaskRequestFactory(cfg GitTaskRequestFactoryConfig) (*GitTaskRequestFactory, error) {
	switch {
	case strings.TrimSpace(cfg.Repo) == "":
		return nil, errors.New("controlplane: execution content repo is required")
	case strings.TrimSpace(cfg.BaseRevision) == "":
		return nil, errors.New("controlplane: execution base revision is required")
	case strings.TrimSpace(cfg.RuntimeID) == "":
		return nil, errors.New("controlplane: execution runtime id is required")
	case strings.TrimSpace(cfg.WorkspaceRoot) == "":
		return nil, errors.New("controlplane: execution workspace root is required")
	}
	if cfg.EnvironmentKind == "" {
		cfg.EnvironmentKind = taskexecution.EnvironmentLocal
	}
	switch cfg.EnvironmentKind {
	case taskexecution.EnvironmentLocal, taskexecution.EnvironmentSSH, taskexecution.EnvironmentDocker:
	default:
		return nil, fmt.Errorf("controlplane: unsupported execution environment %q", cfg.EnvironmentKind)
	}
	if strings.TrimSpace(cfg.EnvironmentTarget) == "" {
		switch cfg.EnvironmentKind {
		case taskexecution.EnvironmentLocal:
			cfg.EnvironmentTarget = "local"
		case taskexecution.EnvironmentDocker:
			cfg.EnvironmentTarget = "docker"
		default:
			return nil, errors.New("controlplane: non-local execution environment target is required")
		}
	}
	return &GitTaskRequestFactory{cfg: cfg, loadBundle: artifacts.LoadBundle}, nil
}

func (f *GitTaskRequestFactory) BuildTaskExecutionRequest(_ context.Context, snapshot ProjectSnapshot, node engineering.TaskNode) (taskexecution.TaskExecutionRequest, error) {
	if snapshot.ProjectID == "" {
		return taskexecution.TaskExecutionRequest{}, errors.New("controlplane: scheduler snapshot is missing project id")
	}
	contextArtifacts, err := executionContextArtifacts(snapshot)
	if err != nil {
		return taskexecution.TaskExecutionRequest{}, err
	}
	bundle, err := f.loadValidatedPinnedBundle(snapshot.ProjectID, node.ID)
	if err != nil {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: load frozen execution bundle for %s: %w", node.ID, err)
	}

	workspacePath, err := taskWorkspaceLocation(f.cfg.WorkspaceRoot, node.ID, f.cfg.EnvironmentKind)
	if err != nil {
		return taskexecution.TaskExecutionRequest{}, err
	}
	if f.cfg.EnvironmentKind == taskexecution.EnvironmentLocal {
		info, statErr := os.Stat(workspacePath)
		if statErr != nil {
			return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: prepared workspace for %s is unavailable at %s: %w", node.ID, workspacePath, statErr)
		}
		if !info.IsDir() {
			return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: prepared workspace for %s is not a directory: %s", node.ID, workspacePath)
		}
	}

	runtimeCaps := make([]agentruntime.Capability, 0, len(node.RuntimeRequirements))
	for _, requirement := range node.RuntimeRequirements {
		runtimeCaps = append(runtimeCaps, agentruntime.Capability(requirement))
	}
	environmentCaps := requiredEnvironmentCapabilities(f.cfg.RuntimeID, f.cfg.EnvironmentKind, node.EnvironmentRequirements)

	request := taskexecution.TaskExecutionRequest{
		TaskID: node.ID,
		FrozenTaskContract: engineering.ArtifactRef{
			ID:       engineering.ID("task-contract:" + string(node.ID)),
			Kind:     engineering.ArtifactTaskContract,
			URI:      "git:" + bundle.SourceCommit + "#" + bundle.Task.Path,
			Revision: bundle.Task.Contract.RevisionHash(),
			Digest:   bundle.Task.Contract.RevisionHash(),
		},
		EngineeringContext: contextArtifacts,
		Runtime:            taskexecution.RuntimeRef{ID: f.cfg.RuntimeID},
		Environment: taskexecution.EnvironmentRef{
			Kind:   f.cfg.EnvironmentKind,
			Target: f.cfg.EnvironmentTarget,
		},
		Workspace: taskexecution.WorkspaceRef{
			Environment:  f.cfg.EnvironmentKind,
			Target:       f.cfg.EnvironmentTarget,
			Identity:     string(snapshot.ProjectID) + ":" + string(node.ID) + ":" + bundle.SourceCommit,
			Path:         workspacePath,
			BaseRevision: bundle.SourceCommit,
		},
		RequiredCapabilities: taskexecution.CapabilityRequirement{
			Runtime:     agentruntime.CapabilityRequirement{Required: runtimeCaps},
			Environment: environmentCaps,
		},
		Policy:      taskexecution.ExecutionPolicy{ModelProfile: f.cfg.ModelProfile},
		Instruction: frozenTaskInstruction(bundle, node),
	}
	if err := request.Validate(); err != nil {
		return taskexecution.TaskExecutionRequest{}, fmt.Errorf("controlplane: composed execution request for %s is invalid: %w", node.ID, err)
	}
	return request, nil
}

func requiredEnvironmentCapabilities(runtimeID string, kind taskexecution.EnvironmentKind, explicit []string) []taskexecution.EnvironmentCapability {
	out := make([]taskexecution.EnvironmentCapability, 0, len(explicit)+9)
	seen := map[taskexecution.EnvironmentCapability]bool{}
	add := func(capability taskexecution.EnvironmentCapability) {
		if capability == "" || seen[capability] {
			return
		}
		seen[capability] = true
		out = append(out, capability)
	}
	for _, requirement := range explicit {
		add(taskexecution.EnvironmentCapability(requirement))
	}
	if kind == taskexecution.EnvironmentLocal {
		return out
	}
	add(taskexecution.CapabilityProcess)
	add(taskexecution.CapabilityCancel)
	add(taskexecution.CapabilityWorkspaceRead)
	add(taskexecution.CapabilityWorkspaceWrite)
	add(taskexecution.EnvironmentCapability("git"))
	if runtimeID != "" {
		add(taskexecution.EnvironmentCapability("tool:" + runtimeID))
	}
	if kind == taskexecution.EnvironmentDocker {
		add(taskexecution.EnvironmentCapability("docker"))
	}
	if runtimeID == "dsh" {
		add(taskexecution.EnvironmentCapability("network"))
		add(taskexecution.EnvironmentCapability("secret:DEEPSEEK_API_KEY"))
	}
	return out
}

func executionContextArtifacts(snapshot ProjectSnapshot) ([]engineering.ArtifactRef, error) {
	if snapshot.Requirements == nil || snapshot.Plan == nil || snapshot.Architecture == nil {
		return nil, errors.New("controlplane: active execution requires requirements, plan and architecture context")
	}
	refs := []engineering.ArtifactRef{
		snapshot.Requirements.Artifact,
		snapshot.Plan.AcceptedRevision.Artifact,
		snapshot.Architecture.Artifact,
	}
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("controlplane: invalid engineering context artifact: %w", err)
		}
	}
	return refs, nil
}

// loadValidatedPinnedBundle resolves the configured revision expression once.
// A bundle can establish the factory-wide source commit only after its project
// and task identities match the persisted scheduling authority. This prevents
// rejected/mismatched input from poisoning the later execution pin.
func (f *GitTaskRequestFactory) loadValidatedPinnedBundle(projectID, taskID engineering.ID) (*artifacts.ExecutionBundle, error) {
	f.pinMu.Lock()
	defer f.pinMu.Unlock()

	revision := f.cfg.BaseRevision
	if f.pinnedCommit != "" {
		revision = f.pinnedCommit
	}
	bundle, err := f.loadBundle(f.cfg.Repo, revision, string(taskID))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(bundle.SourceCommit) == "" {
		return nil, errors.New("controlplane: execution bundle did not report an observed source commit")
	}
	if bundle.ProjectID == "" {
		return nil, fmt.Errorf("controlplane: frozen execution bundle for %s has no project identity", taskID)
	}
	if bundle.ProjectID != string(projectID) {
		return nil, fmt.Errorf("controlplane: frozen execution project mismatch: snapshot=%s bundle=%s", projectID, bundle.ProjectID)
	}
	if bundle.Task.Contract.ID != string(taskID) {
		return nil, fmt.Errorf("controlplane: frozen task identity mismatch: node=%s contract=%s", taskID, bundle.Task.Contract.ID)
	}
	if f.pinnedCommit == "" {
		f.pinnedCommit = bundle.SourceCommit
	} else if bundle.SourceCommit != f.pinnedCommit {
		return nil, fmt.Errorf("controlplane: execution authority drift: pinned=%s observed=%s", f.pinnedCommit, bundle.SourceCommit)
	}
	return bundle, nil
}

func taskWorkspaceLocation(root string, taskID engineering.ID, kind taskexecution.EnvironmentKind) (string, error) {
	id := string(taskID)
	if err := taskID.Validate("workspace task"); err != nil {
		return "", err
	}
	if id == "." || id == ".." || strings.ContainsAny(id, `/\\`) || filepath.Base(id) != id {
		return "", fmt.Errorf("controlplane: task id %q is unsafe for prepared workspace path", id)
	}
	if kind == taskexecution.EnvironmentLocal {
		return filepath.Join(root, id), nil
	}
	// v1.5.2 SSH/Docker profiles target POSIX execution environments. The Core
	// must not use host filepath semantics for an environment-native locator.
	cleanRoot := path.Clean(strings.TrimSpace(root))
	if !path.IsAbs(cleanRoot) || cleanRoot == "/" {
		return "", fmt.Errorf("controlplane: non-local workspace root %q must be an absolute non-root POSIX path", root)
	}
	return path.Join(cleanRoot, id), nil
}

// taskWorkspacePath is retained for v1.5.1 tests/callers and is explicitly the
// Local compatibility edge.
func taskWorkspacePath(root string, taskID engineering.ID) (string, error) {
	return taskWorkspaceLocation(root, taskID, taskexecution.EnvironmentLocal)
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
