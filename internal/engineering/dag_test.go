package engineering

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDAGRejectsMissingDependencyAndCycle(t *testing.T) {
	if _, err := ActivateTaskDAG(TaskDAGProposal{RevisionID: "d", Nodes: []TaskNode{{ID: "a", Dependencies: []ID{"missing"}}}}); err == nil {
		t.Fatal("expected missing dependency")
	}
	_, err := ActivateTaskDAG(TaskDAGProposal{RevisionID: "d", Nodes: []TaskNode{{ID: "a", Dependencies: []ID{"b"}}, {ID: "b", Dependencies: []ID{"a"}}}})
	if !errors.Is(err, ErrDAGCycle) {
		t.Fatalf("expected cycle, got %v", err)
	}
}
func TestDAGBlockedReadyParallelAndTransitions(t *testing.T) {
	d, err := ActivateTaskDAG(TaskDAGProposal{RevisionID: "d", Nodes: []TaskNode{{ID: "a", ParallelEligible: true}, {ID: "b", Dependencies: []ID{"a"}, ParallelEligible: true}, {ID: "c", ParallelEligible: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.ParallelReady()) != 2 {
		t.Fatalf("initial parallel ready=%v", d.ParallelReady())
	}
	d, err = d.Transition("a", TaskRunning, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err = d.Transition("a", TaskAccepted, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err = d.Transition("a", TaskDone, "")
	if err != nil {
		t.Fatal(err)
	}
	ready := d.ReadyNodes()
	if len(ready) != 2 || ready[0].ID != "b" || ready[1].ID != "c" {
		t.Fatalf("ready=%v", ready)
	}
	if _, err = d.Transition("b", TaskDone, ""); !errors.Is(err, ErrIllegalTaskTransition) {
		t.Fatalf("expected illegal transition, got %v", err)
	}
}
func TestDAGRestartRoundTripPreservesReadiness(t *testing.T) {
	d, err := ActivateTaskDAG(TaskDAGProposal{RevisionID: "d", Nodes: []TaskNode{{ID: "a"}, {ID: "b", Dependencies: []ID{"a"}}}})
	if err != nil {
		t.Fatal(err)
	}
	d, _ = d.Transition("a", TaskRunning, "")
	d, _ = d.Transition("a", TaskAccepted, "")
	d, _ = d.Transition("a", TaskDone, "")
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var restored ActiveTaskDAG
	if err = json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	restored = restored.RefreshReadiness()
	if len(restored.ReadyNodes()) != 1 || restored.ReadyNodes()[0].ID != "b" {
		t.Fatalf("restored ready=%v", restored.ReadyNodes())
	}
}

func TestDAGOperationalBlockedAndInterruptedStayExplicit(t *testing.T) {
	d, err := ActivateTaskDAG(TaskDAGProposal{RevisionID: "d", Nodes: []TaskNode{{ID: "a"}}})
	if err != nil {
		t.Fatal(err)
	}
	d, err = d.Transition("a", TaskRunning, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err = d.Transition("a", TaskBlocked, "runtime unavailable")
	if err != nil {
		t.Fatal(err)
	}
	if d.Nodes[0].State != TaskBlocked || d.RefreshReadiness().Nodes[0].State != TaskBlocked {
		t.Fatalf("operational blocked was lost: %+v", d.Nodes[0])
	}
	d, err = d.Transition("a", TaskReady, "")
	if err != nil {
		t.Fatalf("explicit retry should be allowed: %v", err)
	}
	d, err = d.Transition("a", TaskRunning, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err = d.Transition("a", TaskInterrupted, "")
	if err != nil {
		t.Fatal(err)
	}
	if d.Nodes[0].State != TaskInterrupted || d.RefreshReadiness().Nodes[0].State != TaskInterrupted {
		t.Fatalf("interrupted state was lost: %+v", d.Nodes[0])
	}
	d, err = d.Transition("a", TaskReady, "")
	if err != nil {
		t.Fatalf("explicit restart should be allowed: %v", err)
	}
	if d.Nodes[0].State != TaskReady {
		t.Fatalf("state=%s", d.Nodes[0].State)
	}
}

func TestDependencyBlockedStillAutoRefreshes(t *testing.T) {
	d, err := ActivateTaskDAG(TaskDAGProposal{RevisionID: "d", Nodes: []TaskNode{{ID: "a"}, {ID: "b", Dependencies: []ID{"a"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Nodes[1].State != TaskBlocked {
		t.Fatalf("state=%s", d.Nodes[1].State)
	}
	d, _ = d.Transition("a", TaskRunning, "")
	d, _ = d.Transition("a", TaskAccepted, "")
	d, _ = d.Transition("a", TaskDone, "")
	if d.Nodes[1].State != TaskReady {
		t.Fatalf("state=%s reason=%s", d.Nodes[1].State, d.Nodes[1].BlockedReason)
	}
}
