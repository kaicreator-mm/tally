package controlplane

import (
	"context"
	"strings"
	"testing"

	"tally/internal/artifacts"
	"tally/internal/engineering"
	"tally/internal/taskcontract"
	"tally/internal/taskexecution"
)

func TestV152GitTaskRequestFactoryBuildsSSHLocatorWithoutHostStat(t *testing.T) {
	factory, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo:              "/content",
		BaseRevision:      "frozen-ref",
		RuntimeID:         "dsh",
		WorkspaceRoot:     "/remote/tally/workspaces",
		EnvironmentKind:   taskexecution.EnvironmentSSH,
		EnvironmentTarget: "build-host",
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := taskcontract.TaskContract{ID: "TASK-SSH", Profile: taskcontract.ProfileLight, Objective: "execute remotely", Scope: taskcontract.Scope{AllowedPaths: []string{"internal/**"}}}
	factory.loadBundle = func(_, _, _ string) (*artifacts.ExecutionBundle, error) {
		return executionBundleForTest("project-1", "remote-commit", contract), nil
	}

	req, err := factory.BuildTaskExecutionRequest(context.Background(), executionSnapshotForTest("project-1"), engineering.TaskNode{ID: "TASK-SSH"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Environment.Kind != taskexecution.EnvironmentSSH || req.Environment.Target != "build-host" {
		t.Fatalf("environment=%+v", req.Environment)
	}
	if req.Workspace.Path != "/remote/tally/workspaces/TASK-SSH" || req.Workspace.Environment != taskexecution.EnvironmentSSH {
		t.Fatalf("workspace=%+v", req.Workspace)
	}
}

func TestV152GitTaskRequestFactoryBuildsDockerLocatorWithoutHostStat(t *testing.T) {
	factory, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo:              "/content",
		BaseRevision:      "frozen-ref",
		RuntimeID:         "dsh",
		WorkspaceRoot:     "/workspace",
		EnvironmentKind:   taskexecution.EnvironmentDocker,
		EnvironmentTarget: "docker",
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := taskcontract.TaskContract{ID: "TASK-DOCKER", Profile: taskcontract.ProfileLight, Objective: "execute in container", Scope: taskcontract.Scope{AllowedPaths: []string{"internal/**"}}}
	factory.loadBundle = func(_, _, _ string) (*artifacts.ExecutionBundle, error) {
		return executionBundleForTest("project-1", "container-commit", contract), nil
	}

	req, err := factory.BuildTaskExecutionRequest(context.Background(), executionSnapshotForTest("project-1"), engineering.TaskNode{ID: "TASK-DOCKER"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Workspace.Path != "/workspace/TASK-DOCKER" || req.Workspace.Environment != taskexecution.EnvironmentDocker {
		t.Fatalf("workspace=%+v", req.Workspace)
	}
}

func TestV152NonLocalWorkspaceRootRejectsRoot(t *testing.T) {
	_, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo:              "/content",
		BaseRevision:      "main",
		RuntimeID:         "dsh",
		WorkspaceRoot:     "/",
		EnvironmentKind:   taskexecution.EnvironmentSSH,
		EnvironmentTarget: "build-host",
	})
	if err != nil {
		// Constructor validates presence only; path safety is per task.
		t.Fatal(err)
	}
	if _, err := taskWorkspaceLocation("/", engineering.ID("TASK-1"), taskexecution.EnvironmentSSH); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("err=%v", err)
	}
}
