package engineering

import (
	"errors"
	"testing"
	"time"
)

func testArtifact(id ID, k ArtifactKind) ArtifactRef {
	return ArtifactRef{ID: id, Kind: k, URI: "git://repo/path"}
}
func testProv() Provenance {
	return Provenance{Actor: "tester", Source: "unit", At: time.Unix(1, 0).UTC()}
}
func TestRequirementsBaselineRejectsMissingAcceptanceAndCopiesInput(t *testing.T) {
	if _, err := NewRequirementsBaseline("rb", "r1", []Requirement{{ID: "R1", Statement: "x"}}, testArtifact("a", ArtifactRequirements), testProv()); err == nil {
		t.Fatal("expected missing acceptance criteria")
	}
	reqs := []Requirement{{ID: "R1", Statement: "x", AcceptanceCriteria: []string{"works"}}}
	b, err := NewRequirementsBaseline("rb", "r1", reqs, testArtifact("a", ArtifactRequirements), testProv())
	if err != nil {
		t.Fatal(err)
	}
	reqs[0].AcceptanceCriteria[0] = "mutated"
	if b.Requirements[0].AcceptanceCriteria[0] != "works" {
		t.Fatal("baseline input was not copied")
	}
}
func TestPlanningArchitectureAndAudit(t *testing.T) {
	if _, err := NewPlanningRevision("p1", 1, "", []string{"contract first"}, testArtifact("p", ArtifactProjectPlan), testProv()); err != nil {
		t.Fatal(err)
	}
	if _, err := NewArchitectureRevision("a1", ArchitectureADR, "keep seam", []ArchitectureConstraint{{ID: "c1", Statement: "single runtime"}}, testArtifact("a", ArtifactArchitecture), testProv()); err != nil {
		t.Fatal(err)
	}
	r := ActionRecord{ID: "x", Kind: "freeze", Actor: "owner", At: time.Unix(2, 0).UTC(), Provenance: testProv()}
	l, err := (AuditLog{}).Append(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = l.Append(r); !errors.Is(err, ErrDuplicateActionRecord) {
		t.Fatalf("got %v", err)
	}
}
func TestCompletionRequiresPassAndAuthority(t *testing.T) {
	c := CompletionContract{RequiredState: NodeAccepted, RequiredArtifacts: []ID{"a"}, RequiredEvidence: []ID{"e"}, RequiredChecks: []ID{"c"}, RequiredRecords: []ID{"r"}, RequiredAuthority: []ID{"auth"}, RequiredGitOperations: []ID{"g"}}
	if got := c.Evaluate(CompletionFacts{State: NodeAccepted}); got.Satisfied || len(got.Missing) != 6 {
		t.Fatalf("expected six missing facts: %+v", got)
	}
	f := CompletionFacts{State: NodeAccepted, Artifacts: []ArtifactRef{{ID: "a"}}, Evidence: []EvidenceRef{{ID: "e", Status: EvidencePass}}, Checks: []CheckResult{{ID: "c", Status: CheckPass}}, Records: []ActionRecord{{ID: "r"}}, Authority: []AuthorityDecision{{RequirementID: "auth", Status: AuthorityAuthorized}}, GitOperations: []GitOperationRecord{{ID: "g"}}}
	if got := c.Evaluate(f); !got.Satisfied {
		t.Fatalf("should satisfy: %+v", got)
	}
	f.Evidence[0].Status = EvidenceEnvBlocked
	if c.Evaluate(f).Satisfied {
		t.Fatal("ENV-BLOCKED must not satisfy required evidence")
	}
	f.Evidence[0].Status = EvidencePass
	f.Authority[0].Status = AuthorityRejected
	if c.Evaluate(f).Satisfied {
		t.Fatal("rejected authority must block")
	}
}
