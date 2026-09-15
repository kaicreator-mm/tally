package controlplane

import (
	"context"
	"testing"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

// TestCriticalJourneyLifecycleThroughTaskDAGAdvance is the visible v1.5.0
// control-plane journey. It intentionally uses a fake TaskHarness: this test
// proves lifecycle/DAG/completion orchestration, while brownfield Agent Runtime
// + candidateverify execution remains covered by its own integration evidence.
func TestCriticalJourneyLifecycleThroughTaskDAGAdvance(t *testing.T) {
	h := &fakeTaskHarness{}
	s, repo := newTaskHarnessService(t, h)
	bootstrap(t, s) // Requirements -> Plan -> Architecture -> Active TaskDAG.

	ctx := context.Background()
	snap, err := s.Snapshot(ctx, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := findNode(*snap.ActiveDAG, "A")
	b, _ := findNode(*snap.ActiveDAG, "B")
	if a.State != engineering.TaskReady || b.State != engineering.TaskBlocked {
		t.Fatalf("initial flow A=%s B=%s", a.State, b.State)
	}

	// Existing brownfield closed loop is represented here by the narrow Harness
	// result: an Agent claim alone is not enough; Accepted + PASS evidence is.
	h.result = HarnessResult{
		Execution: taskexecution.TaskExecutionResult{
			Status:       taskexecution.ExecutionCompleted,
			CandidateRef: "changeset-A",
		},
		Accepted: true,
		Facts: engineering.CompletionFacts{
			State: engineering.NodeAccepted,
			Evidence: []engineering.EvidenceRef{{
				ID: "verify-A", Kind: "closedloop.final_verification",
				URI: "tally://verification-run/A", Status: engineering.EvidencePass,
			}},
		},
	}
	result, err := s.Dispatch(ctx, "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ = findNode(*result.Snapshot.ActiveDAG, "A")
	b, _ = findNode(*result.Snapshot.ActiveDAG, "B")
	if a.State != engineering.TaskDone || b.State != engineering.TaskReady {
		t.Fatalf("after A A=%s B=%s", a.State, b.State)
	}
	if result.Completion == nil || !result.Completion.Satisfied {
		t.Fatalf("A completion=%+v", result.Completion)
	}

	// B has no additional evidence requirement in bootstrap; accepted Harness
	// facts satisfy its CompletionContract and finish the Active TaskDAG.
	h.result = HarnessResult{
		Execution: taskexecution.TaskExecutionResult{
			Status:       taskexecution.ExecutionCompleted,
			CandidateRef: "changeset-B",
		},
		Accepted: true,
		Facts: engineering.CompletionFacts{State: engineering.NodeAccepted},
	}
	result, err = s.Dispatch(ctx, "project-1", request("B"))
	if err != nil {
		t.Fatal(err)
	}
	b, _ = findNode(*result.Snapshot.ActiveDAG, "B")
	if b.State != engineering.TaskDone {
		t.Fatalf("B state=%s", b.State)
	}

	// Process restart proof: rebuild the Service from the same repository and
	// recover Project Flow from persisted Active TaskDAG, not in-memory state.
	restarted, err := NewServiceWithHarness(repo, h)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := restarted.Snapshot(ctx, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	a, _ = findNode(*persisted.ActiveDAG, "A")
	b, _ = findNode(*persisted.ActiveDAG, "B")
	if a.State != engineering.TaskDone || b.State != engineering.TaskDone {
		t.Fatalf("persisted flow A=%s B=%s", a.State, b.State)
	}
	if len(persisted.Audit.Records) < 8 {
		t.Fatalf("expected lifecycle/execution audit trail, got %d records", len(persisted.Audit.Records))
	}
}
