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
