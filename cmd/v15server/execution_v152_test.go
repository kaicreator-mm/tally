package main

import (
	"strings"
	"testing"

	"tally/internal/taskexecution"
)

func TestV152RequestModelProfileKeepsDSHSelectionInsideRuntime(t *testing.T) {
	t.Setenv("TALLY_V15_MODEL", "deepseek-chat")
	if got := requestModelProfile("dsh"); got != "" {
		t.Fatalf("DSH request ModelProfile=%q, want empty because DSH uses its patch overlay", got)
	}
	if got := requestModelProfile("codex"); got != "deepseek-chat" {
		t.Fatalf("Codex request ModelProfile=%q", got)
	}
}

func TestV152ExecutionFromEnvKeepsLocalCompatibility(t *testing.T) {
	t.Setenv("TALLY_V15_ENVIRONMENT", "local")
	t.Setenv("TALLY_V15_WORKSPACE_ROOT", t.TempDir())
	composition, err := executionFromEnv("fake", "/content")
	if err != nil {
		t.Fatal(err)
	}
	if composition.Kind != taskexecution.EnvironmentLocal || composition.Target != "local" {
		t.Fatalf("composition=%+v", composition)
	}
	executor, ok := composition.Executor.(*taskexecution.RuntimeExecutor)
	if !ok {
		t.Fatalf("executor=%T", composition.Executor)
	}
	if executor.Environment.Kind() != taskexecution.EnvironmentLocal {
		t.Fatalf("environment=%s", executor.Environment.Kind())
	}
}

func TestV152ExecutionFromEnvComposesDSHOverSSH(t *testing.T) {
	t.Setenv("TALLY_V15_ENVIRONMENT", "ssh")
	t.Setenv("TALLY_V15_WORKSPACE_ROOT", "/srv/tally/workspaces")
	t.Setenv("TALLY_V15_SSH_HOST", "build.example.invalid")
	t.Setenv("TALLY_V15_SSH_USER", "builder")
	t.Setenv("TALLY_V15_SSH_PASSWORD", "test-only")
	t.Setenv("TALLY_V15_ENV_TARGET", "build-host")
	t.Setenv("DEEPSEEK_API_KEY", "test-only")

	composition, err := executionFromEnv("dsh", "/content")
	if err != nil {
		t.Fatal(err)
	}
	if composition.Kind != taskexecution.EnvironmentSSH || composition.Target != "build-host" || composition.WorkspaceRoot != "/srv/tally/workspaces" {
		t.Fatalf("composition=%+v", composition)
	}
	executor := composition.Executor.(*taskexecution.RuntimeExecutor)
	if executor.Environment.Kind() != taskexecution.EnvironmentSSH {
		t.Fatalf("environment=%s", executor.Environment.Kind())
	}
	if _, ok := executor.Runtime.(taskexecution.EnvironmentRuntime); !ok {
		t.Fatalf("DSH runtime does not implement EnvironmentRuntime: %T", executor.Runtime)
	}
}

func TestV152ExecutionFromEnvComposesDSHInDocker(t *testing.T) {
	t.Setenv("TALLY_V15_ENVIRONMENT", "docker")
	t.Setenv("TALLY_V15_DOCKER_IMAGE", "tally/dsh:test")
	t.Setenv("TALLY_V15_DOCKER_WORKSPACE_SOURCE", "tally-workspaces")
	t.Setenv("TALLY_V15_DOCKER_WORKSPACE_TARGET", "/workspace")
	t.Setenv("TALLY_V15_DOCKER_NETWORK", "none")
	t.Setenv("DEEPSEEK_API_KEY", "test-only")

	composition, err := executionFromEnv("dsh", "/content")
	if err != nil {
		t.Fatal(err)
	}
	if composition.Kind != taskexecution.EnvironmentDocker || composition.Target != "docker" || composition.WorkspaceRoot != "/workspace" {
		t.Fatalf("composition=%+v", composition)
	}
	executor := composition.Executor.(*taskexecution.RuntimeExecutor)
	if executor.Environment.Kind() != taskexecution.EnvironmentDocker {
		t.Fatalf("environment=%s", executor.Environment.Kind())
	}
}

func TestV152ExecutionFromEnvRejectsLocalOnlyRuntimeForSSH(t *testing.T) {
	t.Setenv("TALLY_V15_ENVIRONMENT", "ssh")
	t.Setenv("TALLY_V15_WORKSPACE_ROOT", "/srv/tally/workspaces")
	t.Setenv("TALLY_V15_SSH_HOST", "build.example.invalid")
	t.Setenv("TALLY_V15_SSH_USER", "builder")
	t.Setenv("TALLY_V15_SSH_PASSWORD", "test-only")

	_, err := executionFromEnv("codex", "/content")
	if err == nil || !strings.Contains(err.Error(), "does not support non-local") {
		t.Fatalf("err=%v", err)
	}
}

func TestV152ExecutionFromEnvRequiresSSHAuthBoundary(t *testing.T) {
	t.Setenv("TALLY_V15_ENVIRONMENT", "ssh")
	t.Setenv("TALLY_V15_WORKSPACE_ROOT", "/srv/tally/workspaces")
	t.Setenv("TALLY_V15_SSH_HOST", "build.example.invalid")
	t.Setenv("TALLY_V15_SSH_USER", "builder")

	_, err := executionFromEnv("dsh", "/content")
	if err == nil || !strings.Contains(err.Error(), "SSH_KEY_FILE or TALLY_V15_SSH_PASSWORD") {
		t.Fatalf("err=%v", err)
	}
}
