package gitengineering

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/engineering"
)

// Service coordinates authority, providers, and immutable GitActionRecord
// creation. It does not own task workflow state; callers feed the resulting
// records back into the existing EngineeringNode/CompletionContract model.
type Service struct {
	Policy AuthorityPolicy
	Local  LocalProvider
	Remote RemoteProvider
	Now    func() time.Time
}

func NewService(policy AuthorityPolicy, local LocalProvider, remote RemoteProvider) *Service {
	return &Service{Policy: policy, Local: local, Remote: remote, Now: time.Now}
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func validateMeta(meta ActionMeta) error {
	if err := meta.RecordID.Validate("git action record"); err != nil {
		return err
	}
	if err := meta.ArtifactID.Validate("git artifact"); err != nil {
		return err
	}
	if strings.TrimSpace(meta.ReplayKey) == "" {
		return errors.New("gitengineering: replay key is required")
	}
	if err := meta.Trace.Trace.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(meta.Trace.Actor) == "" || strings.TrimSpace(meta.Trace.Provider) == "" || strings.TrimSpace(meta.Trace.Repository) == "" {
		return errors.New("gitengineering: actor, provider and repository are required")
	}
	return nil
}

func (s *Service) baseRecord(meta ActionMeta, action engineering.GitActionKind, kind engineering.GitArtifactKind, externalID, uri, revision, base string, authority engineering.GitAuthorityDecision) engineering.GitActionRecord {
	return engineering.GitActionRecord{
		ID:        meta.RecordID,
		ReplayKey: meta.ReplayKey,
		Action:    action,
		Artifact: engineering.GitArtifactRef{
			ID: meta.ArtifactID, Kind: kind, Provider: meta.Trace.Provider, Repository: meta.Trace.Repository,
			ExternalID: externalID, URI: uri, Revision: revision, BaseRevision: base,
		},
		Trace:      meta.Trace.Trace,
		Actor:      meta.Trace.Actor,
		Authority:  authority,
		At:         s.now(),
		Provenance: meta.Trace.Provenance,
	}
}

func (s *Service) blocked(meta ActionMeta, action engineering.GitActionKind, kind engineering.GitArtifactKind, externalID, uri string, authority engineering.GitAuthorityDecision) (engineering.GitActionRecord, error) {
	r := s.baseRecord(meta, action, kind, externalID, uri, "", authority.TargetBase, authority)
	r.Status = engineering.GitActionBlocked
	if err := r.Validate(); err != nil {
		return r, err
	}
	return r, fmt.Errorf("%w: %s", ErrActionBlocked, authority.Reason)
}

func (s *Service) failed(meta ActionMeta, action engineering.GitActionKind, kind engineering.GitArtifactKind, externalID, uri string, authority engineering.GitAuthorityDecision, cause error) (engineering.GitActionRecord, error) {
	r := s.baseRecord(meta, action, kind, externalID, uri, "", authority.TargetBase, authority)
	r.Status = engineering.GitActionFailed
	r.ErrorCode = classifyError(cause)
	if err := r.Validate(); err != nil {
		return r, fmt.Errorf("%v; record validation: %w", cause, err)
	}
	return r, cause
}

func classifyError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrStaleHead):
		return "stale_head"
	case errors.Is(err, ErrGitCommand):
		return "git_command_failed"
	case errors.Is(err, ErrProviderRequest):
		return "provider_request_failed"
	default:
		return "operation_failed"
	}
}

func pendingURI(repo, path string) string {
	return "scm://" + strings.Trim(repo, "/") + "/" + strings.TrimLeft(path, "/")
}

func (s *Service) authority(action engineering.GitActionKind, meta ActionMeta) engineering.GitAuthorityDecision {
	return s.Policy.Evaluate(action, meta.Guard)
}

func (s *Service) success(meta ActionMeta, action engineering.GitActionKind, kind engineering.GitArtifactKind, out RemoteArtifact, fallbackExternal string, authority engineering.GitAuthorityDecision) (engineering.GitActionRecord, error) {
	external := out.ExternalID
	if external == "" {
		external = fallbackExternal
	}
	uri := out.URI
	if uri == "" {
		uri = pendingURI(meta.Trace.Repository, string(kind)+"/"+external)
	}
	r := s.baseRecord(meta, action, kind, external, uri, out.Revision, out.Base, authority)
	r.Status = engineering.GitActionSucceeded
	r.ResultRevision = out.Revision
	if err := r.Validate(); err != nil {
		return r, err
	}
	return r, nil
}

func (s *Service) CreateIssue(ctx context.Context, meta ActionMeta, in IssueInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionIssueCreated, meta)
	uri := pendingURI(meta.Trace.Repository, "issues/new")
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionIssueCreated, engineering.GitArtifactIssue, "new", uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionIssueCreated, engineering.GitArtifactIssue, "new", uri, a, errors.New("gitengineering: remote provider is required"))
	}
	out, err := s.Remote.CreateIssue(ctx, meta.Trace.Repository, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionIssueCreated, engineering.GitArtifactIssue, "new", uri, a, err)
	}
	return s.success(meta, engineering.GitActionIssueCreated, engineering.GitArtifactIssue, out, "new", a)
}

func (s *Service) UpdateIssue(ctx context.Context, meta ActionMeta, issueID string, in IssueInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionIssueUpdated, meta)
	uri := pendingURI(meta.Trace.Repository, "issues/"+issueID)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionIssueUpdated, engineering.GitArtifactIssue, issueID, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionIssueUpdated, engineering.GitArtifactIssue, issueID, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	out, err := s.Remote.UpdateIssue(ctx, meta.Trace.Repository, issueID, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionIssueUpdated, engineering.GitArtifactIssue, issueID, uri, a, err)
	}
	return s.success(meta, engineering.GitActionIssueUpdated, engineering.GitArtifactIssue, out, issueID, a)
}

func (s *Service) CloseIssue(ctx context.Context, meta ActionMeta, issueID string) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionIssueClosed, meta)
	uri := pendingURI(meta.Trace.Repository, "issues/"+issueID)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionIssueClosed, engineering.GitArtifactIssue, issueID, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionIssueClosed, engineering.GitArtifactIssue, issueID, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	out, err := s.Remote.CloseIssue(ctx, meta.Trace.Repository, issueID)
	if err != nil {
		return s.failed(meta, engineering.GitActionIssueClosed, engineering.GitArtifactIssue, issueID, uri, a, err)
	}
	return s.success(meta, engineering.GitActionIssueClosed, engineering.GitArtifactIssue, out, issueID, a)
}

func (s *Service) CreateBranch(ctx context.Context, meta ActionMeta, repoDir, branch, baseRevision string) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionBranchCreated, meta)
	uri := pendingURI(meta.Trace.Repository, "branches/"+branch)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionBranchCreated, engineering.GitArtifactBranch, branch, uri, a)
	}
	if s.Local == nil {
		return s.failed(meta, engineering.GitActionBranchCreated, engineering.GitArtifactBranch, branch, uri, a, errors.New("gitengineering: local provider is required"))
	}
	rev, err := s.Local.CreateBranch(ctx, repoDir, branch, baseRevision)
	if err != nil {
		return s.failed(meta, engineering.GitActionBranchCreated, engineering.GitArtifactBranch, branch, uri, a, err)
	}
	return s.success(meta, engineering.GitActionBranchCreated, engineering.GitArtifactBranch, RemoteArtifact{ExternalID: branch, URI: uri, Revision: rev, Base: baseRevision}, branch, a)
}

func (s *Service) Commit(ctx context.Context, meta ActionMeta, repoDir, message string, paths []string) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionCommitCreated, meta)
	uri := pendingURI(meta.Trace.Repository, "commits/pending")
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionCommitCreated, engineering.GitArtifactCommit, "pending", uri, a)
	}
	if s.Local == nil {
		return s.failed(meta, engineering.GitActionCommitCreated, engineering.GitArtifactCommit, "pending", uri, a, errors.New("gitengineering: local provider is required"))
	}
	rev, err := s.Local.Commit(ctx, repoDir, message, paths)
	if err != nil {
		return s.failed(meta, engineering.GitActionCommitCreated, engineering.GitArtifactCommit, "pending", uri, a, err)
	}
	return s.success(meta, engineering.GitActionCommitCreated, engineering.GitArtifactCommit, RemoteArtifact{ExternalID: rev, URI: pendingURI(meta.Trace.Repository, "commits/"+rev), Revision: rev}, rev, a)
}

func (s *Service) Push(ctx context.Context, meta ActionMeta, repoDir, remote, branch, expectedHead string) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionPushCompleted, meta)
	uri := pendingURI(meta.Trace.Repository, "branches/"+branch)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionPushCompleted, engineering.GitArtifactBranch, branch, uri, a)
	}
	if s.Local == nil {
		return s.failed(meta, engineering.GitActionPushCompleted, engineering.GitArtifactBranch, branch, uri, a, errors.New("gitengineering: local provider is required"))
	}
	rev, err := s.Local.Push(ctx, repoDir, remote, branch, expectedHead)
	if err != nil {
		return s.failed(meta, engineering.GitActionPushCompleted, engineering.GitArtifactBranch, branch, uri, a, err)
	}
	return s.success(meta, engineering.GitActionPushCompleted, engineering.GitArtifactBranch, RemoteArtifact{ExternalID: branch, URI: uri, Revision: rev}, branch, a)
}

func (s *Service) OpenPullRequest(ctx context.Context, meta ActionMeta, in PullRequestInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionPROpened, meta)
	uri := pendingURI(meta.Trace.Repository, "pulls/new")
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionPROpened, engineering.GitArtifactPullRequest, "new", uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionPROpened, engineering.GitArtifactPullRequest, "new", uri, a, errors.New("gitengineering: remote provider is required"))
	}
	in.Base = a.TargetBase
	out, err := s.Remote.OpenPullRequest(ctx, meta.Trace.Repository, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionPROpened, engineering.GitArtifactPullRequest, "new", uri, a, err)
	}
	return s.success(meta, engineering.GitActionPROpened, engineering.GitArtifactPullRequest, out, "new", a)
}

func (s *Service) UpdatePullRequest(ctx context.Context, meta ActionMeta, pullRequestID string, in PullRequestInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionPRUpdated, meta)
	uri := pendingURI(meta.Trace.Repository, "pulls/"+pullRequestID)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionPRUpdated, engineering.GitArtifactPullRequest, pullRequestID, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionPRUpdated, engineering.GitArtifactPullRequest, pullRequestID, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	if in.Base != "" {
		in.Base = a.TargetBase
	}
	out, err := s.Remote.UpdatePullRequest(ctx, meta.Trace.Repository, pullRequestID, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionPRUpdated, engineering.GitArtifactPullRequest, pullRequestID, uri, a, err)
	}
	return s.success(meta, engineering.GitActionPRUpdated, engineering.GitArtifactPullRequest, out, pullRequestID, a)
}

func reviewAction(state ReviewState) (engineering.GitActionKind, error) {
	switch state {
	case ReviewComment:
		return engineering.GitActionPRReviewed, nil
	case ReviewApprove:
		return engineering.GitActionPRApproved, nil
	case ReviewRequestChanges:
		return engineering.GitActionPRChangesRequested, nil
	default:
		return "", fmt.Errorf("gitengineering: invalid review state %q", state)
	}
}

func (s *Service) ReviewPullRequest(ctx context.Context, meta ActionMeta, pullRequestID string, in ReviewInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	action, err := reviewAction(in.State)
	if err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(action, meta)
	uri := pendingURI(meta.Trace.Repository, "pulls/"+pullRequestID+"/reviews")
	if !a.Allowed {
		return s.blocked(meta, action, engineering.GitArtifactPullRequest, pullRequestID, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, action, engineering.GitArtifactPullRequest, pullRequestID, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	out, err := s.Remote.ReviewPullRequest(ctx, meta.Trace.Repository, pullRequestID, in)
	if err != nil {
		return s.failed(meta, action, engineering.GitArtifactPullRequest, pullRequestID, uri, a, err)
	}
	// Review actions keep the Pull Request as the material artifact. Provider review
	// IDs remain available through the review URI/Trace.ReviewID; they must not
	// replace the PR identity in GitArtifactRef. Reviews also do not create a new
	// result revision required by CompletionContract.
	out.ExternalID = pullRequestID
	out.Revision = ""
	return s.success(meta, action, engineering.GitArtifactPullRequest, out, pullRequestID, a)
}

func (s *Service) MergePullRequest(ctx context.Context, meta ActionMeta, pullRequestID string, in MergeInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionPRMerged, meta)
	uri := pendingURI(meta.Trace.Repository, "pulls/"+pullRequestID)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionPRMerged, engineering.GitArtifactPullRequest, pullRequestID, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionPRMerged, engineering.GitArtifactPullRequest, pullRequestID, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	if in.ExpectedHeadRevision == "" {
		in.ExpectedHeadRevision = meta.Guard.ExpectedHeadRevision
	}
	out, err := s.Remote.MergePullRequest(ctx, meta.Trace.Repository, pullRequestID, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionPRMerged, engineering.GitArtifactPullRequest, pullRequestID, uri, a, err)
	}
	return s.success(meta, engineering.GitActionPRMerged, engineering.GitArtifactPullRequest, out, pullRequestID, a)
}

func (s *Service) CreateTag(ctx context.Context, meta ActionMeta, in TagInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionTagCreated, meta)
	uri := pendingURI(meta.Trace.Repository, "tags/"+in.Name)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionTagCreated, engineering.GitArtifactTag, in.Name, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionTagCreated, engineering.GitArtifactTag, in.Name, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	out, err := s.Remote.CreateTag(ctx, meta.Trace.Repository, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionTagCreated, engineering.GitArtifactTag, in.Name, uri, a, err)
	}
	return s.success(meta, engineering.GitActionTagCreated, engineering.GitArtifactTag, out, in.Name, a)
}

func (s *Service) PublishRelease(ctx context.Context, meta ActionMeta, in ReleaseInput) (engineering.GitActionRecord, error) {
	if err := validateMeta(meta); err != nil {
		return engineering.GitActionRecord{}, err
	}
	a := s.authority(engineering.GitActionReleasePublished, meta)
	uri := pendingURI(meta.Trace.Repository, "releases/"+in.TagName)
	if !a.Allowed {
		return s.blocked(meta, engineering.GitActionReleasePublished, engineering.GitArtifactRelease, in.TagName, uri, a)
	}
	if s.Remote == nil {
		return s.failed(meta, engineering.GitActionReleasePublished, engineering.GitArtifactRelease, in.TagName, uri, a, errors.New("gitengineering: remote provider is required"))
	}
	out, err := s.Remote.PublishRelease(ctx, meta.Trace.Repository, in)
	if err != nil {
		return s.failed(meta, engineering.GitActionReleasePublished, engineering.GitArtifactRelease, in.TagName, uri, a, err)
	}
	return s.success(meta, engineering.GitActionReleasePublished, engineering.GitArtifactRelease, out, in.TagName, a)
}
