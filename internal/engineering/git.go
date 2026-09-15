package engineering

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// GitArtifactKind describes SCM engineering artifacts without binding the
// domain model to one provider such as GitHub.
type GitArtifactKind string

const (
	GitArtifactIssue       GitArtifactKind = "issue"
	GitArtifactBranch      GitArtifactKind = "branch"
	GitArtifactCommit      GitArtifactKind = "commit"
	GitArtifactPullRequest GitArtifactKind = "pull_request"
	GitArtifactTag         GitArtifactKind = "tag"
	GitArtifactRelease     GitArtifactKind = "release"
)

func (k GitArtifactKind) valid() bool {
	switch k {
	case GitArtifactIssue, GitArtifactBranch, GitArtifactCommit, GitArtifactPullRequest, GitArtifactTag, GitArtifactRelease:
		return true
	default:
		return false
	}
}

// GitActionKind is the material engineering action vocabulary frozen by the
// v1.5.3 PRD. Adapters may expose provider-specific capabilities internally,
// but persisted engineering records use these provider-neutral semantics.
type GitActionKind string

const (
	GitActionIssueCreated       GitActionKind = "issue_created"
	GitActionIssueUpdated       GitActionKind = "issue_updated"
	GitActionIssueClosed        GitActionKind = "issue_closed"
	GitActionBranchCreated      GitActionKind = "branch_created"
	GitActionCommitCreated      GitActionKind = "commit_created"
	GitActionPushCompleted      GitActionKind = "push_completed"
	GitActionPROpened           GitActionKind = "pr_opened"
	GitActionPRUpdated          GitActionKind = "pr_updated"
	GitActionPRReviewed         GitActionKind = "pr_reviewed"
	GitActionPRApproved         GitActionKind = "pr_approved"
	GitActionPRChangesRequested GitActionKind = "pr_changes_requested"
	GitActionPRMerged           GitActionKind = "pr_merged"
	GitActionTagCreated         GitActionKind = "tag_created"
	GitActionReleasePublished   GitActionKind = "release_published"
)

func (k GitActionKind) valid() bool {
	switch k {
	case GitActionIssueCreated, GitActionIssueUpdated, GitActionIssueClosed,
		GitActionBranchCreated, GitActionCommitCreated, GitActionPushCompleted,
		GitActionPROpened, GitActionPRUpdated, GitActionPRReviewed,
		GitActionPRApproved, GitActionPRChangesRequested, GitActionPRMerged,
		GitActionTagCreated, GitActionReleasePublished:
		return true
	default:
		return false
	}
}

// GitActionStatus records the engineering result of an attempted Git action.
// Command/provider success alone does not imply authority PASS; blocked actions
// are first-class audit facts and are never projected into CompletionFacts.
type GitActionStatus string

const (
	GitActionSucceeded GitActionStatus = "succeeded"
	GitActionFailed    GitActionStatus = "failed"
	GitActionBlocked   GitActionStatus = "blocked"
)

// GitArtifactRef is a provider-neutral reference to a material SCM artifact.
type GitArtifactRef struct {
	ID           ID              `json:"id"`
	Kind         GitArtifactKind `json:"kind"`
	Provider     string          `json:"provider"`
	Repository   string          `json:"repository"`
	ExternalID   string          `json:"external_id,omitempty"`
	URI          string          `json:"uri,omitempty"`
	Revision     string          `json:"revision,omitempty"`
	BaseRevision string          `json:"base_revision,omitempty"`
}

func (r GitArtifactRef) Validate() error {
	if err := r.ID.Validate("git artifact"); err != nil {
		return err
	}
	if !r.Kind.valid() {
		return fmt.Errorf("engineering: invalid git artifact kind %q", r.Kind)
	}
	if strings.TrimSpace(r.Provider) == "" || strings.TrimSpace(r.Repository) == "" {
		return errors.New("engineering: git artifact provider and repository are required")
	}
	if strings.TrimSpace(r.ExternalID) == "" && strings.TrimSpace(r.URI) == "" && strings.TrimSpace(r.Revision) == "" {
		return errors.New("engineering: git artifact requires external id, URI, or revision")
	}
	return nil
}

// GitAuthorityDecision is the authority result attached to a material Git
// action. RequirementID points at the existing v1.5.0 authority contract when
// human authority is required.
type GitAuthorityDecision struct {
	Allowed                bool   `json:"allowed"`
	TargetBase             string `json:"target_base,omitempty"`
	HumanAuthorityRequired bool   `json:"human_authority_required"`
	RequirementID          ID     `json:"requirement_id,omitempty"`
	PolicyRevision         string `json:"policy_revision,omitempty"`
	Reason                 string `json:"reason,omitempty"`
}

func (d GitAuthorityDecision) Validate() error {
	if d.HumanAuthorityRequired && d.Allowed {
		if err := d.RequirementID.Validate("git authority requirement"); err != nil {
			return err
		}
	}
	if !d.Allowed && strings.TrimSpace(d.Reason) == "" {
		return errors.New("engineering: blocked git authority decision requires reason")
	}
	return nil
}

// GitTraceLinks ties Git lifecycle activity back to the existing Engineering
// Harness objects. Optional links stay empty only when that object is not yet
// created/applicable for the action (for example a PR does not exist while an
// issue is being created).
type GitTraceLinks struct {
	ProjectID          ID `json:"project_id"`
	PlanningRevisionID ID `json:"planning_revision_id,omitempty"`
	TaskDAGID          ID `json:"task_dag_id,omitempty"`
	TaskNodeID         ID `json:"task_node_id"`
	TaskContractID     ID `json:"task_contract_id"`
	RunID              ID `json:"run_id,omitempty"`
	ChangeSetID        ID `json:"change_set_id,omitempty"`
	VerificationID     ID `json:"verification_id,omitempty"`
	ReviewID           ID `json:"review_id,omitempty"`
	AcceptanceID       ID `json:"acceptance_id,omitempty"`
	ReleaseID          ID `json:"release_id,omitempty"`
}

func (l GitTraceLinks) Validate() error {
	if err := l.ProjectID.Validate("project"); err != nil {
		return err
	}
	if err := l.TaskNodeID.Validate("task node"); err != nil {
		return err
	}
	if err := l.TaskContractID.Validate("task contract"); err != nil {
		return err
	}
	return nil
}

// GitActionRecord is the immutable audit/provenance carrier for a material Git
// action. ReplayKey identifies one logical command and prevents duplicate or
// replayed successful actions from silently becoming additional evidence.
type GitActionRecord struct {
	ID             ID                   `json:"id"`
	ReplayKey      string               `json:"replay_key"`
	Action         GitActionKind        `json:"action"`
	Artifact       GitArtifactRef       `json:"artifact"`
	Trace          GitTraceLinks        `json:"trace"`
	Actor          string               `json:"actor"`
	Authority      GitAuthorityDecision `json:"authority"`
	At             time.Time            `json:"at"`
	Status         GitActionStatus      `json:"status"`
	ResultRevision string               `json:"result_revision,omitempty"`
	ErrorCode      string               `json:"error_code,omitempty"`
	Provenance     Provenance           `json:"provenance"`
}

func (r GitActionRecord) Validate() error {
	if err := r.ID.Validate("git action record"); err != nil {
		return err
	}
	if strings.TrimSpace(r.ReplayKey) == "" {
		return errors.New("engineering: git action replay key is required")
	}
	if !r.Action.valid() {
		return fmt.Errorf("engineering: invalid git action %q", r.Action)
	}
	if err := r.Artifact.Validate(); err != nil {
		return err
	}
	if err := r.Trace.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Actor) == "" || r.At.IsZero() {
		return errors.New("engineering: git action actor and time are required")
	}
	if err := r.Authority.Validate(); err != nil {
		return err
	}
	switch r.Status {
	case GitActionSucceeded:
		if !r.Authority.Allowed {
			return errors.New("engineering: blocked authority cannot produce succeeded git action")
		}
	case GitActionFailed:
		if strings.TrimSpace(r.ErrorCode) == "" {
			return errors.New("engineering: failed git action requires error code")
		}
	case GitActionBlocked:
		if r.Authority.Allowed {
			return errors.New("engineering: blocked git action requires denied authority")
		}
	default:
		return fmt.Errorf("engineering: invalid git action status %q", r.Status)
	}
	if (r.Action == GitActionCommitCreated || r.Action == GitActionPushCompleted || r.Action == GitActionPRMerged || r.Action == GitActionTagCreated || r.Action == GitActionReleasePublished) && r.Status == GitActionSucceeded && strings.TrimSpace(r.ResultRevision) == "" {
		return fmt.Errorf("engineering: %s requires result revision", r.Action)
	}
	return nil
}

var (
	ErrDuplicateGitAction = errors.New("engineering: duplicate git action record")
	ErrReplayedGitAction  = errors.New("engineering: replayed git action")
)

// GitActionLog is append-only. IDs prevent accidental duplicate persistence;
// ReplayKey prevents a caller from repeating one logical action under a new ID.
type GitActionLog struct {
	Records []GitActionRecord `json:"records"`
}

func (l GitActionLog) Append(r GitActionRecord) (GitActionLog, error) {
	if err := r.Validate(); err != nil {
		return l, err
	}
	for _, x := range l.Records {
		if x.ID == r.ID {
			return l, ErrDuplicateGitAction
		}
		if x.ReplayKey == r.ReplayKey {
			return l, ErrReplayedGitAction
		}
	}
	out := GitActionLog{Records: append([]GitActionRecord(nil), l.Records...)}
	out.Records = append(out.Records, r)
	return out, nil
}

// CompletionOperation projects only an authorized, successful Git action into
// the existing CompletionContract fact model.
func (r GitActionRecord) CompletionOperation() (GitOperationRecord, bool) {
	if r.Status != GitActionSucceeded || !r.Authority.Allowed {
		return GitOperationRecord{}, false
	}
	return GitOperationRecord{ID: r.ID, Operation: string(r.Action), Revision: r.ResultRevision}, true
}

// AddGitActionToCompletion appends a successful action to existing completion
// facts while preserving the current workflow/state authority.
func AddGitActionToCompletion(f CompletionFacts, r GitActionRecord) CompletionFacts {
	op, ok := r.CompletionOperation()
	if !ok {
		return f
	}
	for _, existing := range f.GitOperations {
		if existing.ID == op.ID {
			return f
		}
	}
	f.GitOperations = append(f.GitOperations, op)
	return f
}

// ResultRevision returns the immutable downstream revision created by the last
// successful merge/tag/release action for the task.
func (l GitActionLog) ResultRevision(taskNodeID ID) string {
	for i := len(l.Records) - 1; i >= 0; i-- {
		r := l.Records[i]
		if r.Trace.TaskNodeID != taskNodeID || r.Status != GitActionSucceeded {
			continue
		}
		switch r.Action {
		case GitActionPRMerged, GitActionTagCreated, GitActionReleasePublished:
			if r.ResultRevision != "" {
				return r.ResultRevision
			}
		}
	}
	return ""
}

// v1.5.3 lifecycle semantics. Existing states remain valid; these additive
// states let callers distinguish acceptance from merge-dependent DONE.
const (
	NodeImplemented           NodeState = "implemented"
	NodeVerified              NodeState = "verified"
	NodeReviewed              NodeState = "reviewed"
	NodeAcceptedAwaitingMerge NodeState = "accepted_awaiting_merge"
	NodeDone                  NodeState = "done"
)

// GitLifecycleFacts projects existing implementation/verification/review/
// acceptance/merge facts into the additive v1.5.3 task lifecycle semantics.
// It is a view over existing authorities, not a second workflow authority.
type GitLifecycleFacts struct {
	Implemented   bool `json:"implemented"`
	Verified      bool `json:"verified"`
	Reviewed      bool `json:"reviewed"`
	Accepted      bool `json:"accepted"`
	MergeRequired bool `json:"merge_required"`
	Merged        bool `json:"merged"`
}

func DeriveGitLifecycleState(f GitLifecycleFacts) NodeState {
	if !f.Implemented {
		return NodePending
	}
	if !f.Verified {
		return NodeImplemented
	}
	if !f.Reviewed {
		return NodeVerified
	}
	if !f.Accepted {
		return NodeReviewed
	}
	if f.MergeRequired && !f.Merged {
		return NodeAcceptedAwaitingMerge
	}
	return NodeDone
}
