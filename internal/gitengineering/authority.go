package gitengineering

import (
	"errors"
	"fmt"
	"strings"

	"tally/internal/engineering"
)

var ErrActionBlocked = errors.New("gitengineering: action blocked by authority")

// AuthorityPolicy is the v1.5.3 policy surface for material Git actions. It
// decides whether an action is permitted, the canonical base, and whether a
// human authority decision is required.
type AuthorityPolicy struct {
	Revision             string
	DefaultBase          string
	AllowedActions       map[engineering.GitActionKind]bool
	HumanRequiredActions map[engineering.GitActionKind]bool
}

func DefaultAuthorityPolicy() AuthorityPolicy {
	allowed := map[engineering.GitActionKind]bool{}
	for _, action := range []engineering.GitActionKind{
		engineering.GitActionIssueCreated, engineering.GitActionIssueUpdated, engineering.GitActionIssueClosed,
		engineering.GitActionBranchCreated, engineering.GitActionCommitCreated, engineering.GitActionPushCompleted,
		engineering.GitActionPROpened, engineering.GitActionPRUpdated, engineering.GitActionPRReviewed,
		engineering.GitActionPRApproved, engineering.GitActionPRChangesRequested, engineering.GitActionPRMerged,
		engineering.GitActionTagCreated, engineering.GitActionReleasePublished,
	} {
		allowed[action] = true
	}
	return AuthorityPolicy{
		Revision:       "v1.5.3-default",
		DefaultBase:    "main",
		AllowedActions: allowed,
		HumanRequiredActions: map[engineering.GitActionKind]bool{
			engineering.GitActionPRMerged:         true,
			engineering.GitActionTagCreated:       true,
			engineering.GitActionReleasePublished: true,
		},
	}
}

func (p AuthorityPolicy) Evaluate(action engineering.GitActionKind, facts GuardFacts) engineering.GitAuthorityDecision {
	targetBase := strings.TrimSpace(facts.TargetBase)
	if targetBase == "" {
		targetBase = strings.TrimSpace(p.DefaultBase)
	}
	deny := func(reason string, human bool) engineering.GitAuthorityDecision {
		return engineering.GitAuthorityDecision{
			Allowed:                false,
			TargetBase:             targetBase,
			HumanAuthorityRequired: human,
			RequirementID:          facts.AuthorityRequirementID,
			PolicyRevision:         p.Revision,
			Reason:                 reason,
		}
	}

	if p.AllowedActions != nil {
		allowed, known := p.AllowedActions[action]
		if known && !allowed {
			return deny("policy forbids action", p.HumanRequiredActions[action])
		}
		if !known {
			return deny("action is not admitted by policy", p.HumanRequiredActions[action])
		}
	}

	if strings.TrimSpace(p.DefaultBase) != "" && targetBase != p.DefaultBase {
		return deny(fmt.Sprintf("wrong base: requires %s, got %s", p.DefaultBase, targetBase), p.HumanRequiredActions[action])
	}
	if facts.ExpectedBaseRevision != "" && facts.ObservedBaseRevision != "" && facts.ExpectedBaseRevision != facts.ObservedBaseRevision {
		return deny("stale base revision", p.HumanRequiredActions[action])
	}
	if facts.ExpectedHeadRevision != "" && facts.ObservedHeadRevision != "" && facts.ExpectedHeadRevision != facts.ObservedHeadRevision {
		return deny("stale head revision", p.HumanRequiredActions[action])
	}

	switch action {
	case engineering.GitActionPROpened:
		if !facts.PushCompleted {
			return deny("pull request requires completed push", p.HumanRequiredActions[action])
		}
	case engineering.GitActionPRUpdated, engineering.GitActionPRReviewed, engineering.GitActionPRApproved, engineering.GitActionPRChangesRequested:
		if !facts.PullRequestExists {
			return deny("pull request does not exist", p.HumanRequiredActions[action])
		}
	case engineering.GitActionPRMerged:
		if !facts.PullRequestExists {
			return deny("pull request does not exist", p.HumanRequiredActions[action])
		}
		if !facts.VerificationPassed {
			return deny("verification has not passed", p.HumanRequiredActions[action])
		}
		if !facts.ReviewPassed {
			return deny("independent review has not passed", p.HumanRequiredActions[action])
		}
		if facts.ChangesRequested {
			return deny("changes requested must be repaired and re-reviewed", p.HumanRequiredActions[action])
		}
		if !facts.AcceptancePassed {
			return deny("acceptance has not passed", p.HumanRequiredActions[action])
		}
	case engineering.GitActionTagCreated:
		if facts.AcceptedBaselineRevision == "" || facts.TargetRevision == "" || facts.AcceptedBaselineRevision != facts.TargetRevision {
			return deny("tag target does not match accepted baseline", p.HumanRequiredActions[action])
		}
	case engineering.GitActionReleasePublished:
		if facts.AcceptedBaselineRevision == "" || facts.TargetRevision == "" || facts.AcceptedBaselineRevision != facts.TargetRevision {
			return deny("release target does not match accepted baseline", p.HumanRequiredActions[action])
		}
		if !facts.TagExists {
			return deny("release requires accepted baseline tag", p.HumanRequiredActions[action])
		}
	}

	humanRequired := p.HumanRequiredActions[action]
	if humanRequired {
		if facts.AuthorityRequirementID == "" {
			return deny("human authority requirement is missing", true)
		}
		if facts.HumanAuthority == nil {
			return deny("human authority has not been supplied", true)
		}
		decision := *facts.HumanAuthority
		if decision.RequirementID != facts.AuthorityRequirementID {
			return deny("human authority decision does not match requirement", true)
		}
		if err := decision.Validate(); err != nil {
			return deny("invalid human authority decision: "+err.Error(), true)
		}
		if decision.Status != engineering.AuthorityAuthorized {
			return deny("human authority is not authorized", true)
		}
	}

	return engineering.GitAuthorityDecision{
		Allowed:                true,
		TargetBase:             targetBase,
		HumanAuthorityRequired: humanRequired,
		RequirementID:          facts.AuthorityRequirementID,
		PolicyRevision:         p.Revision,
	}
}
