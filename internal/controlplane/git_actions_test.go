package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type gitActionMemoryRepo struct {
	snapshot ProjectSnapshot
}

func (r *gitActionMemoryRepo) Load(_ context.Context, id engineering.ID) (ProjectSnapshot, error) {
	if r.snapshot.ProjectID != id {
		return ProjectSnapshot{}, ErrProjectNotFound
	}
	return r.snapshot, nil
}

func (r *gitActionMemoryRepo) Save(_ context.Context, s ProjectSnapshot) error {
	r.snapshot = s
	return nil
}

func runningGitGatedSnapshot() ProjectSnapshot {
	return ProjectSnapshot{
		SchemaVersion: SnapshotSchemaVersion,
		ProjectID:     "project",
		ActiveDAG: &engineering.ActiveTaskDAG{RevisionID: "dag", Nodes: []engineering.TaskNode{{
			ID: "task", State: engineering.TaskRunning, Acceptance: engineering.AcceptancePending,
		}}},
		Contracts: map[engineering.ID]engineering.CompletionContract{
			"task": {RequiredState: engineering.NodeComplete, RequiredGitOperations: []engineering.ID{"merge-op"}},
		},
		UpdatedAt: time.Unix(1, 0).UTC(),
	}
}

func successfulMergeRecord() engineering.GitActionRecord {
	now := time.Unix(3, 0).UTC()
	return engineering.GitActionRecord{
		ID:        "merge-op",
		ReplayKey: "task:merge:42:head",
		Action:    engineering.GitActionPRMerged,
		Artifact: engineering.GitArtifactRef{
			ID: "pr-42", Kind: engineering.GitArtifactPullRequest, Provider: "github", Repository: "o/r", ExternalID: "42", URI: "https://example.test/pull/42",
		},
		Trace: engineering.GitTraceLinks{
			ProjectID: "project", PlanningRevisionID: "plan", TaskDAGID: "dag", TaskNodeID: "task", TaskContractID: "contract", RunID: "run", ChangeSetID: "change", VerificationID: "verify", ReviewID: "review", AcceptanceID: "accept",
		},
		Actor:          "tally-git-authority",
		Authority:      engineering.GitAuthorityDecision{Allowed: true, TargetBase: "main", PolicyRevision: "policy"},
		At:             now,
		Status:         engineering.GitActionSucceeded,
		ResultRevision: "merge-sha",
		Provenance:     engineering.Provenance{Actor: "tally-git-authority", Source: "unit", Revision: "merge-sha", At: now},
	}
}

func TestAcceptedTaskWaitsForRequiredMergeThenBecomesDone(t *testing.T) {
	repo := &gitActionMemoryRepo{snapshot: runningGitGatedSnapshot()}
	s, err := NewService(repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return time.Unix(2, 0).UTC() }
	wf := &TaskWorkflowResult{Accepted: true, Facts: engineering.CompletionFacts{State: engineering.NodeComplete}}

	first, err := s.finishAccepted(context.Background(), "project", "task", taskexecution.TaskExecutionResult{}, wf, "verification")
	if err != nil {
		t.Fatal(err)
	}
	if first.Completion == nil || first.Completion.Satisfied {
		t.Fatalf("expected merge-gated incomplete contract, got %+v", first.Completion)
	}
	if got, _ := findNode(*first.Snapshot.ActiveDAG, "task"); got.State != engineering.TaskAccepted {
		t.Fatalf("accepted task should wait for merge, got %s", got.State)
	}
	if len(first.Completion.Missing) != 1 || first.Completion.Missing[0].Kind != engineering.MissingGitOperation {
		t.Fatalf("unexpected missing requirements: %+v", first.Completion.Missing)
	}

	second, err := s.RecordGitAction(context.Background(), "project", "task", successfulMergeRecord())
	if err != nil {
		t.Fatal(err)
	}
	if second.Completion == nil || !second.Completion.Satisfied {
		t.Fatalf("merge should satisfy completion: %+v", second.Completion)
	}
	if got, _ := findNode(*second.Snapshot.ActiveDAG, "task"); got.State != engineering.TaskDone {
		t.Fatalf("task should be done after merge, got %s", got.State)
	}
	if got := second.Snapshot.GitActions.ResultRevision("task"); got != "merge-sha" {
		t.Fatalf("downstream result revision = %q", got)
	}
	facts := second.Snapshot.CompletionFacts["task"]
	if len(facts.GitOperations) != 1 || facts.GitOperations[0].ID != "merge-op" || facts.GitOperations[0].Revision != "merge-sha" {
		t.Fatalf("completion provenance missing merge: %+v", facts.GitOperations)
	}
}

func TestBlockedGitActionIsAuditedButCannotCompleteTask(t *testing.T) {
	snap := runningGitGatedSnapshot()
	snap.ActiveDAG.Nodes[0].State = engineering.TaskAccepted
	snap.ActiveDAG.Nodes[0].Acceptance = engineering.AcceptanceAccepted
	snap.CompletionFacts = map[engineering.ID]engineering.CompletionFacts{"task": {State: engineering.NodeComplete}}
	repo := &gitActionMemoryRepo{snapshot: snap}
	s, err := NewService(repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return time.Unix(4, 0).UTC() }
	r := successfulMergeRecord()
	r.ID = "blocked-merge"
	r.ReplayKey = "blocked"
	r.Status = engineering.GitActionBlocked
	r.ResultRevision = ""
	r.Authority = engineering.GitAuthorityDecision{Allowed: false, TargetBase: "main", PolicyRevision: "policy", Reason: "verification missing"}

	result, err := s.RecordGitAction(context.Background(), "project", "task", r)
	if err != nil {
		t.Fatal(err)
	}
	if result.Completion != nil {
		t.Fatalf("blocked action must not trigger completion: %+v", result.Completion)
	}
	if got, _ := findNode(*result.Snapshot.ActiveDAG, "task"); got.State != engineering.TaskAccepted {
		t.Fatalf("blocked action advanced task to %s", got.State)
	}
	if len(result.Snapshot.GitActions.Records) != 1 || len(result.Snapshot.CompletionFacts["task"].GitOperations) != 0 {
		t.Fatalf("blocked action provenance projection is unsafe: %+v", result.Snapshot)
	}
}

func TestRecordGitActionRejectsReplayAndTraceMismatch(t *testing.T) {
	snap := runningGitGatedSnapshot()
	snap.ActiveDAG.Nodes[0].State = engineering.TaskAccepted
	snap.ActiveDAG.Nodes[0].Acceptance = engineering.AcceptanceAccepted
	snap.CompletionFacts = map[engineering.ID]engineering.CompletionFacts{"task": {State: engineering.NodeComplete}}
	repo := &gitActionMemoryRepo{snapshot: snap}
	s, err := NewService(repo, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	r := successfulMergeRecord()
	if _, err := s.RecordGitAction(context.Background(), "other-project", "task", r); err == nil {
		t.Fatal("project trace mismatch accepted")
	}
	if _, err := s.RecordGitAction(context.Background(), "project", "other-task", r); err == nil {
		t.Fatal("task trace mismatch accepted")
	}
	if _, err := s.RecordGitAction(context.Background(), "project", "task", r); err != nil {
		t.Fatal(err)
	}
	second := r
	second.ID = "merge-op-2"
	if _, err := s.RecordGitAction(context.Background(), "project", "task", second); !errors.Is(err, engineering.ErrReplayedGitAction) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}
