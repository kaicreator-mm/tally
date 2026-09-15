package gitengineering

import (
	"context"
	"errors"
	"testing"
	"time"

	"tally/internal/engineering"
)

type fakeRemote struct {
	mergeCalls int
	mergeOut   RemoteArtifact
	mergeErr   error
	reviewOut  RemoteArtifact
}

func (f *fakeRemote) CreateIssue(context.Context, string, IssueInput) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "1", URI: "u"}, nil
}
func (f *fakeRemote) UpdateIssue(context.Context, string, string, IssueInput) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "1", URI: "u"}, nil
}
func (f *fakeRemote) CloseIssue(context.Context, string, string) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "1", URI: "u"}, nil
}
func (f *fakeRemote) OpenPullRequest(context.Context, string, PullRequestInput) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "2", URI: "pr", Revision: "head", Base: "main"}, nil
}
func (f *fakeRemote) UpdatePullRequest(context.Context, string, string, PullRequestInput) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "2", URI: "pr", Revision: "head", Base: "main"}, nil
}
func (f *fakeRemote) ReviewPullRequest(context.Context, string, string, ReviewInput) (RemoteArtifact, error) {
	if f.reviewOut.ExternalID == "" {
		return RemoteArtifact{ExternalID: "7", URI: "review", Revision: "head"}, nil
	}
	return f.reviewOut, nil
}
func (f *fakeRemote) MergePullRequest(context.Context, string, string, MergeInput) (RemoteArtifact, error) {
	f.mergeCalls++
	if f.mergeErr != nil {
		return RemoteArtifact{}, f.mergeErr
	}
	if f.mergeOut.Revision == "" {
		return RemoteArtifact{ExternalID: "2", Revision: "merge"}, nil
	}
	return f.mergeOut, nil
}
func (f *fakeRemote) CreateTag(context.Context, string, TagInput) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "refs/tags/v", URI: "tag", Revision: "merge"}, nil
}
func (f *fakeRemote) PublishRelease(context.Context, string, ReleaseInput) (RemoteArtifact, error) {
	return RemoteArtifact{ExternalID: "9", URI: "release", Revision: "merge"}, nil
}

func metaFor(action string) ActionMeta {
	now := time.Unix(10, 0).UTC()
	return ActionMeta{
		RecordID: engineering.ID("record-" + action), ArtifactID: engineering.ID("artifact-" + action), ReplayKey: "task:" + action,
		Trace: TraceContext{
			Trace: engineering.GitTraceLinks{ProjectID: "p", PlanningRevisionID: "plan", TaskDAGID: "dag", TaskNodeID: "task", TaskContractID: "contract", RunID: "run", ChangeSetID: "cs", VerificationID: "verify", ReviewID: "review", AcceptanceID: "accept"},
			Actor: "agent", Provider: "github", Repository: "o/r", Provenance: engineering.Provenance{Actor: "agent", Source: "unit", At: now},
		},
	}
}

func TestServiceBlocksMergeBeforeProviderCall(t *testing.T) {
	remote := &fakeRemote{}
	s := NewService(DefaultAuthorityPolicy(), nil, remote)
	s.Now = func() time.Time { return time.Unix(20, 0) }
	m := metaFor("merge")
	m.Guard = validMergeFacts()
	m.Guard.VerificationPassed = false
	r, err := s.MergePullRequest(context.Background(), m, "2", MergeInput{ExpectedHeadRevision: "head"})
	if !errors.Is(err, ErrActionBlocked) || r.Status != engineering.GitActionBlocked {
		t.Fatalf("record=%+v err=%v", r, err)
	}
	if remote.mergeCalls != 0 {
		t.Fatal("provider called for blocked merge")
	}
	if _, ok := r.CompletionOperation(); ok {
		t.Fatal("blocked merge satisfied completion")
	}
}

func TestServiceMergeProducesResultRevisionAndCompletionFact(t *testing.T) {
	remote := &fakeRemote{mergeOut: RemoteArtifact{ExternalID: "2", URI: "pr", Revision: "merge-sha"}}
	s := NewService(DefaultAuthorityPolicy(), nil, remote)
	s.Now = func() time.Time { return time.Unix(20, 0) }
	m := metaFor("merge")
	m.Guard = validMergeFacts()
	r, err := s.MergePullRequest(context.Background(), m, "2", MergeInput{ExpectedHeadRevision: "head"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != engineering.GitActionSucceeded || r.ResultRevision != "merge-sha" {
		t.Fatalf("record=%+v", r)
	}
	facts := engineering.AddGitActionToCompletion(engineering.CompletionFacts{State: engineering.NodeAcceptedAwaitingMerge}, r)
	if len(facts.GitOperations) != 1 || facts.GitOperations[0].Revision != "merge-sha" {
		t.Fatalf("facts=%+v", facts)
	}
	log, err := (engineering.GitActionLog{}).Append(r)
	if err != nil {
		t.Fatal(err)
	}
	if got := log.ResultRevision("task"); got != "merge-sha" {
		t.Fatalf("got=%q", got)
	}
}

func TestServiceRequestChangesMapsDistinctAction(t *testing.T) {
	remote := &fakeRemote{}
	s := NewService(DefaultAuthorityPolicy(), nil, remote)
	m := metaFor("review")
	m.Guard = GuardFacts{TargetBase: "main", PullRequestExists: true}
	r, err := s.ReviewPullRequest(context.Background(), m, "2", ReviewInput{State: ReviewRequestChanges, Body: "fix"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Action != engineering.GitActionPRChangesRequested {
		t.Fatalf("action=%s", r.Action)
	}
}

func TestServiceProviderFailureIsNotPass(t *testing.T) {
	remote := &fakeRemote{mergeErr: ErrProviderRequest}
	s := NewService(DefaultAuthorityPolicy(), nil, remote)
	m := metaFor("merge-fail")
	m.Guard = validMergeFacts()
	r, err := s.MergePullRequest(context.Background(), m, "2", MergeInput{ExpectedHeadRevision: "head"})
	if err == nil || r.Status != engineering.GitActionFailed || r.ErrorCode != "provider_request_failed" {
		t.Fatalf("record=%+v err=%v", r, err)
	}
	if _, ok := r.CompletionOperation(); ok {
		t.Fatal("failed merge satisfied completion")
	}
}
