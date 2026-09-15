package controlplane

import (
	"strings"
	"testing"
	"time"

	"tally/internal/closedloop"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

func TestRepairInstructionAddsOnlyStructuredFeedbackProjection(t *testing.T) {
	base := "implement the task"
	feedback := &closedloop.RepairFeedback{
		TaskID:            "A",
		Attempt:           1,
		RunID:             closedloop.RunIDFor("A", 1),
		ChangeSet:         "cs-1",
		Failure:           closedloop.FailureRepairable,
		DiagnosticSummary: "visible validation failed",
	}
	got, err := repairInstruction(base, feedback)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, base) || !strings.Contains(got, "Tally repair feedback (safe structured projection)") || !strings.Contains(got, "visible validation failed") {
		t.Fatalf("instruction=%q", got)
	}
	if got == base {
		t.Fatal("repair feedback was not attached")
	}
	unchanged, err := repairInstruction(base, nil)
	if err != nil || unchanged != base {
		t.Fatalf("nil feedback changed instruction: %q err=%v", unchanged, err)
	}
}

func TestAcceptedCompletionFactsAreBoundToClosedLoopBasis(t *testing.T) {
	req := taskexecution.TaskExecutionRequest{
		TaskID: "A",
		FrozenTaskContract: engineering.ArtifactRef{
			ID: "task-A", Kind: engineering.ArtifactTaskContract, URI: "git://repo/task-A", Revision: "task-rev",
		},
		EngineeringContext: []engineering.ArtifactRef{{ID: "arch", Kind: engineering.ArtifactArchitecture, URI: "git://repo/arch"}},
	}
	at := time.Unix(100, 0).UTC()
	facts := acceptedCompletionFacts(req, "verification-rev", closedloop.Basis{
		ChangeSet: "changeset-1", VerifyRun: "42", Attempt: 2,
	}, at)
	if facts.State != engineering.NodeAccepted {
		t.Fatalf("state=%s", facts.State)
	}
	if len(facts.Evidence) != 1 || facts.Evidence[0].ID != VerificationEvidenceID("A") || facts.Evidence[0].Status != engineering.EvidencePass || facts.Evidence[0].URI != "tally://verification-run/42" {
		t.Fatalf("evidence=%+v", facts.Evidence)
	}
	if len(facts.Checks) != 1 || facts.Checks[0].ID != ClosedLoopCheckID("A") || facts.Checks[0].Status != engineering.CheckPass || !strings.Contains(facts.Checks[0].Basis, "changeset-1") {
		t.Fatalf("checks=%+v", facts.Checks)
	}
	if len(facts.Records) != 1 || facts.Records[0].ID != AcceptanceRecordID("A") || facts.Records[0].At != at {
		t.Fatalf("records=%+v", facts.Records)
	}
	if len(facts.Artifacts) != 5 {
		t.Fatalf("artifacts=%+v", facts.Artifacts)
	}
}
