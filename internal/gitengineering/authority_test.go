package gitengineering

import (
	"testing"
	"time"

	"tally/internal/engineering"
)

func authorized(req engineering.ID) *engineering.AuthorityDecision {
	d := engineering.AuthorityDecision{
		RequirementID: req,
		Status:        engineering.AuthorityAuthorized,
		Actor:         "owner",
		At:            time.Unix(1, 0).UTC(),
		Provenance:    engineering.Provenance{Actor: "owner", Source: "unit", At: time.Unix(1, 0).UTC()},
	}
	return &d
}

func validMergeFacts() GuardFacts {
	return GuardFacts{
		TargetBase:             "main",
		ExpectedBaseRevision:   "base",
		ObservedBaseRevision:   "base",
		ExpectedHeadRevision:   "head",
		ObservedHeadRevision:   "head",
		PullRequestExists:      true,
		VerificationPassed:     true,
		ReviewPassed:           true,
		AcceptancePassed:       true,
		AuthorityRequirementID: "merge-auth",
		HumanAuthority:         authorized("merge-auth"),
	}
}

func TestAuthorityBlocksWrongBaseAndStaleHead(t *testing.T) {
	p := DefaultAuthorityPolicy()
	f := validMergeFacts()
	f.TargetBase = "develop"
	if d := p.Evaluate(engineering.GitActionPRMerged, f); d.Allowed || d.Reason == "" {
		t.Fatalf("wrong base allowed: %+v", d)
	}
	f = validMergeFacts()
	f.ObservedHeadRevision = "new-head"
	if d := p.Evaluate(engineering.GitActionPRMerged, f); d.Allowed || d.Reason != "stale head revision" {
		t.Fatalf("stale head: %+v", d)
	}
}

func TestAuthorityBlocksMissingPRAndMergeBeforeVerification(t *testing.T) {
	p := DefaultAuthorityPolicy()
	f := validMergeFacts()
	f.PullRequestExists = false
	if d := p.Evaluate(engineering.GitActionPRMerged, f); d.Allowed {
		t.Fatal("missing PR allowed")
	}
	f = validMergeFacts()
	f.VerificationPassed = false
	if d := p.Evaluate(engineering.GitActionPRMerged, f); d.Allowed || d.Reason != "verification has not passed" {
		t.Fatalf("verification gate: %+v", d)
	}
}

func TestAuthorityBlocksUnauthorizedMerge(t *testing.T) {
	p := DefaultAuthorityPolicy()
	f := validMergeFacts()
	f.HumanAuthority = nil
	d := p.Evaluate(engineering.GitActionPRMerged, f)
	if d.Allowed || !d.HumanAuthorityRequired {
		t.Fatalf("unauthorized merge: %+v", d)
	}
}

func TestRequestChangesRequiresRepairAndRereview(t *testing.T) {
	p := DefaultAuthorityPolicy()
	f := validMergeFacts()
	f.ChangesRequested = true
	if d := p.Evaluate(engineering.GitActionPRMerged, f); d.Allowed {
		t.Fatal("merge allowed with outstanding changes request")
	}
	f.ChangesRequested = false
	f.VerificationPassed = true
	f.ReviewPassed = true
	if d := p.Evaluate(engineering.GitActionPRMerged, f); !d.Allowed {
		t.Fatalf("re-reviewed merge blocked: %+v", d)
	}
}

func TestTagAndReleaseMustMatchAcceptedBaseline(t *testing.T) {
	p := DefaultAuthorityPolicy()
	for _, action := range []engineering.GitActionKind{engineering.GitActionTagCreated, engineering.GitActionReleasePublished} {
		f := GuardFacts{TargetBase: "main", AcceptedBaselineRevision: "accepted", TargetRevision: "other", AuthorityRequirementID: "rel-auth", HumanAuthority: authorized("rel-auth"), TagExists: true}
		if d := p.Evaluate(action, f); d.Allowed {
			t.Fatalf("%s allowed wrong baseline", action)
		}
		f.TargetRevision = "accepted"
		if action == engineering.GitActionReleasePublished {
			f.TagExists = false
			if d := p.Evaluate(action, f); d.Allowed {
				t.Fatal("release allowed without tag")
			}
			f.TagExists = true
		}
		if d := p.Evaluate(action, f); !d.Allowed {
			t.Fatalf("%s valid baseline blocked: %+v", action, d)
		}
	}
}

func TestOpenPRRequiresPush(t *testing.T) {
	p := DefaultAuthorityPolicy()
	f := GuardFacts{TargetBase: "main"}
	if d := p.Evaluate(engineering.GitActionPROpened, f); d.Allowed {
		t.Fatal("PR opened without push")
	}
	f.PushCompleted = true
	if d := p.Evaluate(engineering.GitActionPROpened, f); !d.Allowed {
		t.Fatalf("PR after push blocked: %+v", d)
	}
}
