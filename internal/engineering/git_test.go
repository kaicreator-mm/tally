package engineering

import (
	"errors"
	"testing"
	"time"
)

func validGitRecord() GitActionRecord {
	now := time.Unix(10, 0).UTC()
	return GitActionRecord{
		ID:        "ga-1",
		ReplayKey: "task-1:pr_merged:42:head",
		Action:    GitActionPRMerged,
		Artifact:  GitArtifactRef{ID: "pr-42", Kind: GitArtifactPullRequest, Provider: "github", Repository: "o/r", ExternalID: "42", URI: "https://example/pr/42"},
		Trace:     GitTraceLinks{ProjectID: "p", PlanningRevisionID: "plan", TaskDAGID: "dag", TaskNodeID: "task-1", TaskContractID: "contract", RunID: "run", ChangeSetID: "cs", VerificationID: "verify", ReviewID: "review", AcceptanceID: "accept"},
		Actor:     "agent",
		Authority: GitAuthorityDecision{Allowed: true, TargetBase: "main", PolicyRevision: "policy-1"},
		At:        now, Status: GitActionSucceeded, ResultRevision: "abc123",
		Provenance: Provenance{Actor: "agent", Source: "unit", Revision: "abc123", At: now},
	}
}

func TestGitActionRecordValidationAndCompletionProjection(t *testing.T) {
	r := validGitRecord()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	op, ok := r.CompletionOperation()
	if !ok || op.ID != r.ID || op.Operation != string(GitActionPRMerged) || op.Revision != "abc123" {
		t.Fatalf("bad projection: %+v %v", op, ok)
	}
	facts := AddGitActionToCompletion(CompletionFacts{State: NodeAcceptedAwaitingMerge}, r)
	facts = AddGitActionToCompletion(facts, r)
	if len(facts.GitOperations) != 1 {
		t.Fatalf("expected deduplicated op, got %+v", facts.GitOperations)
	}
}

func TestGitActionRejectsSucceededWhenAuthorityDenied(t *testing.T) {
	r := validGitRecord()
	r.Authority = GitAuthorityDecision{Allowed: false, Reason: "verification missing"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error")
	}
	r.Status = GitActionBlocked
	r.ResultRevision = ""
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.CompletionOperation(); ok {
		t.Fatal("blocked action must not satisfy completion")
	}
}

func TestGitActionLogRejectsDuplicateAndReplay(t *testing.T) {
	r := validGitRecord()
	log, err := (GitActionLog{}).Append(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = log.Append(r); !errors.Is(err, ErrDuplicateGitAction) {
		t.Fatalf("got %v", err)
	}
	r.ID = "ga-2"
	if _, err = log.Append(r); !errors.Is(err, ErrReplayedGitAction) {
		t.Fatalf("got %v", err)
	}
}

func TestGitActionRequiresResultRevisionForMaterialRevisionActions(t *testing.T) {
	for _, action := range []GitActionKind{GitActionCommitCreated, GitActionPushCompleted, GitActionPRMerged, GitActionTagCreated, GitActionReleasePublished} {
		r := validGitRecord()
		r.Action = action
		r.ResultRevision = ""
		if err := r.Validate(); err == nil {
			t.Fatalf("%s should require result revision", action)
		}
	}
}

func TestResultRevisionUsesLatestSuccessfulReleaseLikeAction(t *testing.T) {
	r1 := validGitRecord()
	r1.Action = GitActionPRMerged
	r1.ResultRevision = "merge"
	r1.ID = "m"
	r1.ReplayKey = "m"
	r2 := validGitRecord()
	r2.Action = GitActionTagCreated
	r2.ResultRevision = "merge"
	r2.ID = "t"
	r2.ReplayKey = "t"
	r3 := validGitRecord()
	r3.Action = GitActionReleasePublished
	r3.ResultRevision = "merge"
	r3.ID = "r"
	r3.ReplayKey = "r"
	log := GitActionLog{Records: []GitActionRecord{r1, r2, r3}}
	if got := log.ResultRevision("task-1"); got != "merge" {
		t.Fatalf("got %q", got)
	}
}

func TestDeriveGitLifecycleStateRequiresMergeBeforeDone(t *testing.T) {
	f := GitLifecycleFacts{Implemented: true, Verified: true, Reviewed: true, Accepted: true, MergeRequired: true}
	if got := DeriveGitLifecycleState(f); got != NodeAcceptedAwaitingMerge {
		t.Fatalf("got %s", got)
	}
	f.Merged = true
	if got := DeriveGitLifecycleState(f); got != NodeDone {
		t.Fatalf("got %s", got)
	}
	f.MergeRequired = false
	f.Merged = false
	if got := DeriveGitLifecycleState(f); got != NodeDone {
		t.Fatalf("no-merge task got %s", got)
	}
}
