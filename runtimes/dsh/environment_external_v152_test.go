package dsh

import (
	"context"
	"os"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/runner"
	"tally/internal/taskexecution"
)

func v152RealDSHRequest(kind taskexecution.EnvironmentKind, target, workspace, revision string, docker bool) taskexecution.TaskExecutionRequest {
	required := []taskexecution.EnvironmentCapability{
		taskexecution.CapabilityProcess,
		taskexecution.CapabilityCancel,
		taskexecution.CapabilityWorkspaceRead,
		taskexecution.CapabilityWorkspaceWrite,
		"git",
		"tool:dsh",
		"network",
		"secret:DEEPSEEK_API_KEY",
	}
	if docker {
		required = append(required, "docker")
	}
	instruction := os.Getenv("TALLY_V152_DSH_SMOKE_PROMPT")
	if instruction == "" {
		instruction = "Return exactly TALLY_V152_REMOTE_OK. Do not modify files."
	}
	return taskexecution.TaskExecutionRequest{
		TaskID:             "T-v152-real-dsh",
		FrozenTaskContract: engineering.ArtifactRef{ID: "contract-v152-real-dsh", Kind: engineering.ArtifactTaskContract, URI: "git://v152-real-dsh"},
		Runtime:            taskexecution.RuntimeRef{ID: "dsh"},
		Environment:        taskexecution.EnvironmentRef{Kind: kind, Target: target},
		Workspace: taskexecution.WorkspaceRef{
			Environment:  kind,
			Target:       target,
			Identity:     "v152-real-dsh-workspace",
			Path:         workspace,
			BaseRevision: revision,
		},
		RequiredCapabilities: taskexecution.CapabilityRequirement{Environment: required},
		Instruction:          instruction,
	}
}

func realDSHRuntime(t *testing.T) *Runtime {
	t.Helper()
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Skip("DEEPSEEK_API_KEY is not configured")
	}
	return New(Config{
		BinPath:  os.Getenv("TALLY_V152_DSH_BIN"),
		Model:    os.Getenv("TALLY_V152_DSH_MODEL"),
		Provider: os.Getenv("TALLY_V152_DSH_PROVIDER"),
		Env:      map[string]string{"DEEPSEEK_API_KEY": key},
	})
}

func TestV152RealDSHOverSSH(t *testing.T) {
	if os.Getenv("TALLY_V152_RUN_REAL_DSH") != "1" {
		t.Skip("set TALLY_V152_RUN_REAL_DSH=1 to execute the real DSH critical journey")
	}
	host := os.Getenv("TALLY_V152_SSH_HOST")
	user := os.Getenv("TALLY_V152_SSH_USER")
	workspace := os.Getenv("TALLY_V152_SSH_WORKSPACE")
	revision := os.Getenv("TALLY_V152_SSH_BASE_REVISION")
	if host == "" || user == "" || workspace == "" || revision == "" {
		t.Skip("SSH target/user/workspace/base revision is not configured")
	}
	cfg := runner.SSHConfig{
		Host:           host,
		User:           user,
		Password:       os.Getenv("TALLY_V152_SSH_PASSWORD"),
		KeyFile:        os.Getenv("TALLY_V152_SSH_KEY_FILE"),
		KnownHostsFile: os.Getenv("TALLY_V152_SSH_KNOWN_HOSTS"),
	}
	if cfg.Password == "" && cfg.KeyFile == "" {
		t.Skip("SSH authentication is not configured")
	}
	profile := taskexecution.EnvironmentProfile{
		Network: true,
		Secrets: map[string]bool{"DEEPSEEK_API_KEY": true},
	}
	env := taskexecution.NewSSHEnvironment(host, runner.NewSSHRunner(cfg), profile, nil)
	executor, err := taskexecution.NewRuntimeExecutor(realDSHRuntime(t), env)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := executor.Execute(ctx, v152RealDSHRequest(taskexecution.EnvironmentSSH, host, workspace, revision, false))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != taskexecution.ExecutionCompleted {
		t.Fatalf("status=%s claim=%+v", result.Status, result.AgentClaim)
	}
}

func TestV152RealDSHInDocker(t *testing.T) {
	if os.Getenv("TALLY_V152_RUN_REAL_DSH") != "1" {
		t.Skip("set TALLY_V152_RUN_REAL_DSH=1 to execute the real DSH critical journey")
	}
	image := os.Getenv("TALLY_V152_DOCKER_IMAGE")
	source := os.Getenv("TALLY_V152_DOCKER_WORKSPACE_SOURCE")
	mountTarget := os.Getenv("TALLY_V152_DOCKER_WORKSPACE_TARGET")
	workspace := os.Getenv("TALLY_V152_DOCKER_WORKSPACE")
	revision := os.Getenv("TALLY_V152_DOCKER_BASE_REVISION")
	if mountTarget == "" {
		mountTarget = "/workspace"
	}
	if image == "" || source == "" || workspace == "" || revision == "" {
		t.Skip("Docker image/workspace source/task workspace/base revision is not configured")
	}
	network := os.Getenv("TALLY_V152_DOCKER_NETWORK")
	if network == "none" {
		t.Skip("real DSH Docker journey requires network access")
	}
	docker := &runner.DockerTaskRunner{Profile: runner.DockerTaskProfile{
		Image:           image,
		Workdir:         mountTarget,
		WorkspaceSource: source,
		WorkspaceTarget: mountTarget,
		Network:         network,
	}}
	profile := taskexecution.EnvironmentProfile{
		Network: true,
		Secrets: map[string]bool{"DEEPSEEK_API_KEY": true},
	}
	env := taskexecution.NewDockerEnvironment("docker", docker, profile, nil)
	executor, err := taskexecution.NewRuntimeExecutor(realDSHRuntime(t), env)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result, err := executor.Execute(ctx, v152RealDSHRequest(taskexecution.EnvironmentDocker, "docker", workspace, revision, true))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != taskexecution.ExecutionCompleted {
		t.Fatalf("status=%s claim=%+v", result.Status, result.AgentClaim)
	}
}
