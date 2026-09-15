package controlplane

import (
	"testing"

	"tally/internal/changeset"
	"tally/internal/closedloop"
)

func TestTaskExecutionContextHashIsDeterministic(t *testing.T) {
	req := request("A")
	first, err := taskExecutionContextHash(req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := taskExecutionContextHash(req)
	if err != nil {
		t.Fatal(err)
	}
	if first == "" || first != second {
		t.Fatalf("context hashes differ: %q %q", first, second)
	}
	req.Instruction = "different implementation intent"
	changed, err := taskExecutionContextHash(req)
	if err != nil {
		t.Fatal(err)
	}
	if changed == first {
		t.Fatal("prompt change did not change context hash")
	}
}

func TestAttemptContextHashSeparatesRepairLineage(t *testing.T) {
	base := "sha256:base"
	initial := closedloop.CodegenRunRequest{TaskID: "A", Attempt: 1, RunID: closedloop.RunIDFor("A", 1)}
	first, err := attemptContextHash(base, initial)
	if err != nil {
		t.Fatal(err)
	}

	prior, err := changeset.New(changeset.Spec{
		Project: "project-1", BaseRevision: "base", ProducerRun: "1",
		Files: []changeset.ChangedFile{{Path: "a.go", Content: "package a\n"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	feedback := &closedloop.RepairFeedback{
		TaskID: "A", Attempt: 1, RunID: closedloop.RunIDFor("A", 1),
		ChangeSet: prior.ID(), Failure: closedloop.FailureRepairable,
		DiagnosticSummary: "visible verification failed",
	}
	repair := closedloop.CodegenRunRequest{
		TaskID: "A", Attempt: 2, RunID: closedloop.RunIDFor("A", 2), ParentRun: closedloop.RunIDFor("A", 1),
		RepairBase: prior, Feedback: feedback,
	}
	repairHash, err := attemptContextHash(base, repair)
	if err != nil {
		t.Fatal(err)
	}
	if repairHash == first {
		t.Fatal("repair attempt reused initial context identity")
	}
	again, err := attemptContextHash(base, repair)
	if err != nil {
		t.Fatal(err)
	}
	if again != repairHash {
		t.Fatalf("repair context not deterministic: %q %q", repairHash, again)
	}
}
