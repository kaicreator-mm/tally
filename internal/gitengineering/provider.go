package gitengineering

import (
	"context"

	"tally/internal/engineering"
)

// LocalProvider contains repository operations that execute against a checked
// out Git workspace. The domain service depends on this seam rather than git CLI
// details.
type LocalProvider interface {
	ResolveRevision(ctx context.Context, repoDir, revision string) (string, error)
	CreateBranch(ctx context.Context, repoDir, branch, baseRevision string) (revision string, err error)
	Commit(ctx context.Context, repoDir, message string, paths []string) (revision string, err error)
	Push(ctx context.Context, repoDir, remote, branch, expectedHeadRevision string) (revision string, err error)
}

// RemoteProvider models SCM collaboration/release operations without exposing
// GitHub-specific request/response types to the Engineering Harness domain.
type RemoteProvider interface {
	CreateIssue(ctx context.Context, repo string, in IssueInput) (RemoteArtifact, error)
	UpdateIssue(ctx context.Context, repo, issueID string, in IssueInput) (RemoteArtifact, error)
	CloseIssue(ctx context.Context, repo, issueID string) (RemoteArtifact, error)
	OpenPullRequest(ctx context.Context, repo string, in PullRequestInput) (RemoteArtifact, error)
	UpdatePullRequest(ctx context.Context, repo, pullRequestID string, in PullRequestInput) (RemoteArtifact, error)
	ReviewPullRequest(ctx context.Context, repo, pullRequestID string, in ReviewInput) (RemoteArtifact, error)
	MergePullRequest(ctx context.Context, repo, pullRequestID string, in MergeInput) (RemoteArtifact, error)
	CreateTag(ctx context.Context, repo string, in TagInput) (RemoteArtifact, error)
	PublishRelease(ctx context.Context, repo string, in ReleaseInput) (RemoteArtifact, error)
}

type IssueInput struct {
	Title string
	Body  string
}

type PullRequestInput struct {
	Title        string
	Body         string
	Head         string
	Base         string
	HeadRevision string
}

type ReviewState string

const (
	ReviewComment        ReviewState = "comment"
	ReviewApprove        ReviewState = "approve"
	ReviewRequestChanges ReviewState = "request_changes"
)

type ReviewInput struct {
	State        ReviewState
	Body         string
	HeadRevision string
}

type MergeInput struct {
	ExpectedHeadRevision string
	Method               string
}

type TagInput struct {
	Name     string
	Revision string
}

type ReleaseInput struct {
	TagName  string
	Name     string
	Body     string
	Revision string
}

// RemoteArtifact is the normalized result returned by an SCM provider.
type RemoteArtifact struct {
	ExternalID string
	URI        string
	Revision   string
	Base       string
}

// TraceContext links one action to the existing v1.5.0 engineering objects.
type TraceContext struct {
	Trace      engineering.GitTraceLinks
	Actor      string
	Provider   string
	Repository string
	Provenance engineering.Provenance
}

// GuardFacts are observations used by Git Authority. They are facts, not a
// parallel workflow model; verification/review/acceptance remain owned by the
// existing Engineering Harness.
type GuardFacts struct {
	TargetBase               string
	ExpectedBaseRevision     string
	ObservedBaseRevision     string
	ExpectedHeadRevision     string
	ObservedHeadRevision     string
	PullRequestExists        bool
	PushCompleted            bool
	VerificationPassed       bool
	ReviewPassed             bool
	AcceptancePassed         bool
	ChangesRequested         bool
	AcceptedBaselineRevision string
	TargetRevision           string
	TagExists                bool
	HumanAuthority           *engineering.AuthorityDecision
	AuthorityRequirementID   engineering.ID
}

// ActionMeta supplies immutable record identity and audit links for one action.
type ActionMeta struct {
	RecordID   engineering.ID
	ArtifactID engineering.ID
	ReplayKey  string
	Trace      TraceContext
	Guard      GuardFacts
}
