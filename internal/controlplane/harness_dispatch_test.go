package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type fakeTaskHarness struct {
	result      HarnessResult
	err         error
	calls       int
	cancelCalls int
}

func (f *fakeTaskHarness) Run(context.Context, engineering.ID, taskexecution.TaskExecutionRequest) (HarnessResult, error) {
	f.calls++
	return f.result, f.err
}
func (f *fakeTaskHarness) Cancel(engineering.ID) error {
	f.cancelCalls++
	return nil
}

func newTaskHarnessService(t *testing.T, h TaskHarness) (*Service, *FileRepository) {
	t.Helper()
	repo, err := NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewServiceWithHarness(repo, h)
	if err != nil {
		t.Fatal(err)
	}
	n := int64(200)
	s.Now = func() time.Time { n++; return time.Unix(n, 0) }
	return s, repo
}

func TestHarnessAcceptedStillRequiresCompletionContract(t *testing.T) {
	h := &fakeTaskHarness{result: HarnessResult{
		Execution: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted},
		Accepted:  true,
		Facts:     engineering.CompletionFacts{State: engineering.NodeAccepted},
	}}
	s, _ := newTaskHarnessService(t, h)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	if h.calls != 1 || res.Completion == nil || res.Completion.Satisfied {
		t.Fatalf("calls=%d completion=%+v", h.calls, res.Completion)
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskBlocked {
		t.Fatalf("state=%s", n.State)
	}
}

func TestHarnessAcceptedAndContractAdvanceProjectFlow(t *testing.T) {
	h := &fakeTaskHarness{result: HarnessResult{
		Execution: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted, CandidateRef: "changeset-1"},
		Accepted: true,
		Facts: engineering.CompletionFacts{
			State: engineering.NodeAccepted,
			Evidence: []engineering.EvidenceRef{{ID: "verify-A", Kind: "verification", URI: "tally://verification-run/1", Status: engineering.EvidencePass}},
		},
	}}
	s, repo := newTaskHarnessService(t, h)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	b, _ := findNode(*res.Snapshot.ActiveDAG, "B")
	if a.State != engineering.TaskDone || b.State != engineering.TaskReady {
		t.Fatalf("A=%s B=%s", a.State, b.State)
	}
	persisted, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	a, _ = findNode(*persisted.ActiveDAG, "A")
	if a.State != engineering.TaskDone {
		t.Fatalf("persisted state=%s", a.State)
	}
}

func TestHarnessEscalationBlocksWithoutGoError(t *testing.T) {
	h := &fakeTaskHarness{result: HarnessResult{
		Execution: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted},
		Reason:    "closed loop escalated: assurance_insufficient",
	}}
	s, _ := newTaskHarnessService(t, h)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskBlocked || n.BlockedReason != h.result.Reason {
		t.Fatalf("node=%+v", n)
	}
}

func TestHarnessInterruptionPersistsAndRestartCancelsRunning(t *testing.T) {
	h := &fakeTaskHarness{result: HarnessResult{Interrupted: true, Execution: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionInterrupted}}}
	s, repo := newTaskHarnessService(t, h)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskInterrupted {
		t.Fatalf("state=%s", n.State)
	}

	// Simulate a process restart that recovered a persisted running task. The
	// Harness receives a best-effort cancel before state is returned to ready.
	snap, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	dag, err := snap.ActiveDAG.Transition("A", engineering.TaskReady, "")
	if err != nil {
		t.Fatal(err)
	}
	dag, err = dag.Transition("A", engineering.TaskRunning, "")
	if err != nil {
		t.Fatal(err)
	}
	snap.ActiveDAG = &dag
	if err := repo.Save(context.Background(), snap); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RestartTask(context.Background(), "project-1", "A"); err != nil {
		t.Fatal(err)
	}
	if h.cancelCalls != 1 {
		t.Fatalf("cancel calls=%d", h.cancelCalls)
	}
}

func TestHarnessOperationalErrorBlocks(t *testing.T) {
	h := &fakeTaskHarness{err: errors.New("fact store unavailable")}
	s, _ := newTaskHarnessService(t, h)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err == nil {
		t.Fatal("expected operational error")
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskBlocked {
		t.Fatalf("state=%s", n.State)
	}
}
