package taskexecution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/runner"
)

type v152ExternalProbeRuntime struct{}

func (*v152ExternalProbeRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{Name: "v1.5.2-external-probe", Capabilities: agentruntime.NewCapabilitySet()}
}

func (*v152ExternalProbeRuntime) Start(context.Context, agentruntime.StartRequest) (agentruntime.Session, error) {
	return nil, errors.New("v1.5.2 external probe must not use local Start")
}

func (*v152ExternalProbeRuntime) StartInEnvironment(ctx context.Context, start agentruntime.StartRequest, exec runner.Runner) (agentruntime.Session, error) {
	res, err := exec.Run(ctx, runner.Command{Argv: []string{"git", "-C", start.Workspace, "rev-parse", "HEAD"}})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("environment runtime probe exited %d: %s", res.ExitCode, res.Stderr)
	}
	return &v152ExternalProbeSession{}, nil
}

type v152ExternalProbeSession struct{}

func (*v152ExternalProbeSession) Execute(context.Context, agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{RunID: "v152-external", SessionID: "v152-external", Status: agentruntime.RunCompleted}, nil
}
func (*v152ExternalProbeSession) Cancel(context.Context, agentruntime.RunID) error { return nil }
func (*v152ExternalProbeSession) Events(context.Context) (agentruntime.EventStream, error) {
	return nil, errors.New("external probe event stream is unused")
}
func (*v152ExternalProbeSession) Result(context.Context, agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{}, nil
}
func (*v152ExternalProbeSession) Close(context.Context) error { return nil }

func v152ExternalRequest(kind EnvironmentKind, target, workspace, revision string) TaskExecutionRequest {
	return TaskExecutionRequest{
		TaskID:             "T-v152-external",
		FrozenTaskContract: engineering.ArtifactRef{ID: "v152-external-contract", Kind: engineering.ArtifactTaskContract, URI: "git://v152-external"},
		Runtime:            RuntimeRef{ID: "v1.5.2-external-probe"},
		Environment:        EnvironmentRef{Kind: kind, Target: target},
		Workspace: WorkspaceRef{
			Environment:  kind,
			Target:       target,
			Identity:     "v1.5.2-external-workspace",
			Path:         workspace,
			BaseRevision: revision,
		},
		RequiredCapabilities: CapabilityRequirement{Environment: []EnvironmentCapability{CapabilityProcess, CapabilityWorkspaceRead}},
		Instruction:          "probe selected execution environment",
	}
}

func TestV152ExternalSSHEnvironment(t *testing.T) {
	host := os.Getenv("TALLY_V152_SSH_HOST")
	user := os.Getenv("TALLY_V152_SSH_USER")
	workspace := os.Getenv("TALLY_V152_SSH_WORKSPACE")
	revision := os.Getenv("TALLY_V152_SSH_BASE_REVISION")
	if host == "" || user == "" || workspace == "" || revision == "" {
		t.Skip("TALLY_V152_SSH_HOST/USER/WORKSPACE/BASE_REVISION not configured")
	}
	cfg := runner.SSHConfig{Host: host, User: user, KeyFile: os.Getenv("TALLY_V152_SSH_KEY_FILE"), Password: os.Getenv("TALLY_V152_SSH_PASSWORD")}
	if cfg.KeyFile == "" && cfg.Password == "" {
		t.Skip("TALLY_V152_SSH_KEY_FILE or TALLY_V152_SSH_PASSWORD not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	env := NewSSHEnvironment(host, runner.NewSSHRunner(cfg), EnvironmentProfile{}, nil)
	executor, err := NewRuntimeExecutor(&v152ExternalProbeRuntime{}, env)
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(ctx, v152ExternalRequest(EnvironmentSSH, host, workspace, revision))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ExecutionCompleted {
		t.Fatalf("status=%s", result.Status)
	}
}

func TestV152ExternalDockerEnvironment(t *testing.T) {
	image := os.Getenv("TALLY_V152_DOCKER_IMAGE")
	source := os.Getenv("TALLY_V152_DOCKER_WORKSPACE_SOURCE")
	revision := os.Getenv("TALLY_V152_DOCKER_BASE_REVISION")
	if image == "" || source == "" || revision == "" {
		t.Skip("TALLY_V152_DOCKER_IMAGE/WORKSPACE_SOURCE/BASE_REVISION not configured")
	}
	target := os.Getenv("TALLY_V152_DOCKER_WORKSPACE_TARGET")
	if target == "" {
		target = "/workspace"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	docker := &runner.DockerTaskRunner{Profile: runner.DockerTaskProfile{Image: image, Workdir: target, WorkspaceSource: source, WorkspaceTarget: target}}
	env := NewDockerEnvironment("docker", docker, EnvironmentProfile{}, nil)
	executor, err := NewRuntimeExecutor(&v152ExternalProbeRuntime{}, env)
	if err != nil {
		t.Fatal(err)
	}
	result, err := executor.Execute(ctx, v152ExternalRequest(EnvironmentDocker, "docker", target, revision))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != ExecutionCompleted {
		t.Fatalf("status=%s", result.Status)
	}
}
