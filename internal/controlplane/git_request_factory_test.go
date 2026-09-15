package controlplane

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		return &artifacts.ExecutionBundle{
			TaskID:       taskID,
			SourceCommit: "0123456789abcdef",
			ProjectID:    "project-1",
			Task: artifacts.TaskArtifact{
				Path:     ".tally/tasks/TASK-1.yaml",
				Contract: contract,
			},
		}, nil
	}

	req, err := factory.BuildTaskExecutionRequest(context.Background(), ProjectSnapshot{ProjectID: "project-1"}, engineering.TaskNode{
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
	if req.Workspace.BaseRevision != "0123456789abcdef" || req.Workspace.Path != workspace || req.Workspace.Environment != taskexecution.EnvironmentLocal {
		t.Fatalf("unexpected workspace: %#v", req.Workspace)
	}
	if !strings.Contains(req.Instruction, contract.Objective) || !strings.Contains(req.Instruction, "0123456789abcdef") {
		t.Fatalf("instruction does not carry frozen authority: %q", req.Instruction)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("composed request violates frozen v1.5.0 contract: %v", err)
	}
}

func TestGitTaskRequestFactoryRejectsProjectDriftBeforeDispatch(t *testing.T) {
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
	factory.loadBundle = func(_, _, taskID string) (*artifacts.ExecutionBundle, error) {
		return &artifacts.ExecutionBundle{
			TaskID:       taskID,
			SourceCommit: "0123456789abcdef",
			ProjectID:    "other-project",
			Task: artifacts.TaskArtifact{Path: ".tally/tasks/TASK-1.yaml", Contract: taskcontract.TaskContract{
				ID: "TASK-1", Profile: taskcontract.ProfileLight, Objective: "x", Scope: taskcontract.Scope{AllowedPaths: []string{"internal/**"}},
			}},
		}, nil
	}
	_, err = factory.BuildTaskExecutionRequest(context.Background(), ProjectSnapshot{ProjectID: "project-1"}, engineering.TaskNode{ID: "TASK-1"})
	if err == nil || !strings.Contains(err.Error(), "project mismatch") {
		t.Fatalf("expected project drift rejection, got %v", err)
	}
}
