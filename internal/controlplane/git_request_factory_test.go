package controlplane

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/engineering"
	"tally/internal/taskcontract"
	"tally/internal/taskexecution"
)

func TestGitTaskRequestFactoryBuildsPinnedV150Request(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "TASK-1")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	factory, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo:              "/content",
		BaseRevision:      "frozen-ref",
		RuntimeID:         "fake",
		WorkspaceRoot:     root,
		EnvironmentTarget: "local-test",
		ModelProfile:      "low-cost",
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := taskcontract.TaskContract{
		ID:        "TASK-1",
		Profile:   taskcontract.ProfileLight,
		Title:     "Implement persistence seam",
		Objective: "implement the frozen task without expanding scope",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/**"}},
	}
	factory.loadBundle = func(repo, revision, taskID string) (*artifacts.ExecutionBundle, error) {
		if repo != "/content" || revision != "frozen-ref" || taskID != "TASK-1" {
			t.Fatalf("unexpected loader args repo=%q revision=%q task=%q", repo, revision, taskID)
		}
		return executionBundleForTest("project-1", "0123456789abcdef", contract), nil
	}

	req, err := factory.BuildTaskExecutionRequest(context.Background(), executionSnapshotForTest("project-1"), engineering.TaskNode{
		ID:                       "TASK-1",
		RuntimeRequirements:      []string{"session.isolation"},
		EnvironmentRequirements:  []string{"workspace.write"},
		VerificationRequirements: []string{"AC-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.TaskID != "TASK-1" || req.Runtime.ID != "fake" {
		t.Fatalf("unexpected task/runtime: %#v", req)
	}
	if req.FrozenTaskContract.Kind != engineering.ArtifactTaskContract || req.FrozenTaskContract.Revision != contract.RevisionHash() {
		t.Fatalf("request is not pinned to frozen TaskContract: %#v", req.FrozenTaskContract)
	}
	if len(req.EngineeringContext) != 3 || req.EngineeringContext[0].Kind != engineering.ArtifactRequirements || req.EngineeringContext[1].Kind != engineering.ArtifactProjectPlan || req.EngineeringContext[2].Kind != engineering.ArtifactArchitecture {
		t.Fatalf("engineering context was not propagated: %#v", req.EngineeringContext)
	}
	if req.Workspace.BaseRevision != "0123456789abcdef" || req.Workspace.Path != workspace || req.Workspace.Environment != taskexecution.EnvironmentLocal {
		t.Fatalf("unexpected workspace: %#v", req.Workspace)
	}
	if len(req.RequiredCapabilities.Runtime.Required) != 1 || req.RequiredCapabilities.Runtime.Required[0] != agentruntime.CapSessionIsolation {
		t.Fatalf("runtime capability requirements were not propagated: %#v", req.RequiredCapabilities.Runtime)
	}
	if len(req.RequiredCapabilities.Environment) != 1 || req.RequiredCapabilities.Environment[0] != taskexecution.CapabilityWorkspaceWrite {
		t.Fatalf("environment capability requirements were not propagated: %#v", req.RequiredCapabilities.Environment)
	}
	if !strings.Contains(req.Instruction, contract.Objective) || !strings.Contains(req.Instruction, "0123456789abcdef") {
		t.Fatalf("instruction does not carry frozen authority: %q", req.Instruction)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("composed request violates frozen v1.5.0 contract: %v", err)
	}
}

func TestGitTaskRequestFactoryPinsMovingRevisionAfterFirstValidRequest(t *testing.T) {
	root := t.TempDir()
	for _, id := range []string{"TASK-1", "TASK-2"} {
		if err := os.MkdirAll(filepath.Join(root, id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	factory, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo: "/content", BaseRevision: "main", RuntimeID: "fake", WorkspaceRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	factory.loadBundle = func(_ string, revision string, taskID string) (*artifacts.ExecutionBundle, error) {
		calls++
		switch calls {
		case 1:
			if revision != "main" {
				t.Fatalf("first request must resolve configured revision, got %q", revision)
			}
		case 2:
			if revision != "commit-1" {
				t.Fatalf("later request must use observed pinned commit, got %q", revision)
			}
		}
		contract := taskcontract.TaskContract{ID: taskID, Profile: taskcontract.ProfileLight, Objective: "execute", Scope: taskcontract.Scope{AllowedPaths: []string{"internal/**"}}}
		return executionBundleForTest("project-1", "commit-1", contract), nil
	}
	for _, id := range []engineering.ID{"TASK-1", "TASK-2"} {
		req, err := factory.BuildTaskExecutionRequest(context.Background(), executionSnapshotForTest("project-1"), engineering.TaskNode{ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if req.Workspace.BaseRevision != "commit-1" {
			t.Fatalf("task %s base revision=%q, want pinned commit-1", id, req.Workspace.BaseRevision)
		}
	}
}

func TestGitTaskRequestFactoryRejectsProjectDriftWithoutPoisoningPin(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "TASK-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	factory, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo:          "/content",
		BaseRevision:  "frozen-ref",
		RuntimeID:     "fake",
		WorkspaceRoot: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	factory.loadBundle = func(_, revision, taskID string) (*artifacts.ExecutionBundle, error) {
		calls++
		contract := taskcontract.TaskContract{ID: taskID, Profile: taskcontract.ProfileLight, Objective: "x", Scope: taskcontract.Scope{AllowedPaths: []string{"internal/**"}}}
		if calls == 1 {
			if revision != "frozen-ref" {
				t.Fatalf("unexpected first revision %q", revision)
			}
			return executionBundleForTest("other-project", "bad-commit", contract), nil
		}
		if revision != "frozen-ref" {
			t.Fatalf("rejected bundle poisoned commit pin; second revision=%q", revision)
		}
		return executionBundleForTest("project-1", "good-commit", contract), nil
	}
	_, err = factory.BuildTaskExecutionRequest(context.Background(), executionSnapshotForTest("project-1"), engineering.TaskNode{ID: "TASK-1"})
	if err == nil || !strings.Contains(err.Error(), "project mismatch") {
		t.Fatalf("expected project drift rejection, got %v", err)
	}
	req, err := factory.BuildTaskExecutionRequest(context.Background(), executionSnapshotForTest("project-1"), engineering.TaskNode{ID: "TASK-1"})
	if err != nil {
		t.Fatalf("valid retry after rejected bundle: %v", err)
	}
	if req.Workspace.BaseRevision != "good-commit" {
		t.Fatalf("base revision=%q, want good-commit", req.Workspace.BaseRevision)
	}
}

func TestGitTaskRequestFactoryRejectsMissingEngineeringContext(t *testing.T) {
	factory, err := NewGitTaskRequestFactory(GitTaskRequestFactoryConfig{
		Repo: "/content", BaseRevision: "main", RuntimeID: "fake", WorkspaceRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = factory.BuildTaskExecutionRequest(context.Background(), ProjectSnapshot{ProjectID: "project-1"}, engineering.TaskNode{ID: "TASK-1"})
	if err == nil || !strings.Contains(err.Error(), "requirements, plan and architecture context") {
		t.Fatalf("expected missing engineering context rejection, got %v", err)
	}
}

func TestTaskWorkspacePathRejectsPathTraversal(t *testing.T) {
	for _, id := range []engineering.ID{"../escape", `..\\escape`, "a/b"} {
		if _, err := taskWorkspacePath(t.TempDir(), id); err == nil {
			t.Fatalf("task id %q should be rejected as an unsafe workspace path", id)
		}
	}
}

func executionSnapshotForTest(projectID engineering.ID) ProjectSnapshot {
	return ProjectSnapshot{
		ProjectID: projectID,
		Requirements: &engineering.RequirementsBaseline{Artifact: engineering.ArtifactRef{
			ID: "req-art", Kind: engineering.ArtifactRequirements, URI: "mem:req",
		}},
		Plan: &engineering.ProjectPlan{AcceptedRevision: engineering.PlanningRevision{Artifact: engineering.ArtifactRef{
			ID: "plan-art", Kind: engineering.ArtifactProjectPlan, URI: "mem:plan",
		}}},
		Architecture: &engineering.ArchitectureRevision{Artifact: engineering.ArtifactRef{
			ID: "arch-art", Kind: engineering.ArtifactArchitecture, URI: "mem:arch",
		}},
	}
}

func executionBundleForTest(projectID, commit string, contract taskcontract.TaskContract) *artifacts.ExecutionBundle {
	return &artifacts.ExecutionBundle{
		TaskID:       contract.ID,
		SourceCommit: commit,
		ProjectID:    projectID,
		Task: artifacts.TaskArtifact{
			Path:     ".tally/tasks/" + contract.ID + ".yaml",
			Contract: contract,
		},
	}
}
