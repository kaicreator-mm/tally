package controlplane

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type parallelHarnessV151 struct {
	started chan engineering.ID
	release chan struct{}
	active  atomic.Int32
	max     atomic.Int32
}

func (h *parallelHarnessV151) Run(_ context.Context, _ engineering.ID, req taskexecution.TaskExecutionRequest) (HarnessResult, error) {
	active := h.active.Add(1)
	defer h.active.Add(-1)
	for {
		old := h.max.Load()
		if active <= old || h.max.CompareAndSwap(old, active) {
			break
		}
	}
	h.started <- req.TaskID
	<-h.release
	return HarnessResult{
		Execution: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted},
		Accepted:  true,
		Facts:     engineering.CompletionFacts{State: engineering.NodeAccepted},
	}, nil
}

func (h *parallelHarnessV151) Cancel(engineering.ID) error { return nil }

func newParallelProjectV151(t *testing.T, nodes []engineering.TaskNode, h TaskHarness) (*Service, *FileRepository) {
	t.Helper()
	s, repo := newTaskHarnessService(t, h)
	ctx := context.Background()
	if _, err := s.AcceptRequirements(ctx, "project-1", baseline(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptPlan(ctx, "project-1", plan(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptArchitecture(ctx, "project-1", arch(t)); err != nil {
		t.Fatal(err)
	}
	contracts := make(map[engineering.ID]engineering.CompletionContract, len(nodes))
	for _, node := range nodes {
		contracts[node.ID] = engineering.CompletionContract{RequiredState: engineering.NodeAccepted}
	}
	if _, err := s.ActivateDAG(ctx, "project-1", engineering.TaskDAGProposal{RevisionID: "dag-parallel", Nodes: nodes}, contracts); err != nil {
		t.Fatal(err)
	}
	return s, repo
}

func TestDispatchDoesNotHoldProjectMutationLockDuringHarnessRun(t *testing.T) {
	h := &parallelHarnessV151{started: make(chan engineering.ID, 2), release: make(chan struct{})}
	s, repo := newParallelProjectV151(t, []engineering.TaskNode{
		{ID: "A", ParallelEligible: true},
		{ID: "B", ParallelEligible: true},
	}, h)

	type dispatchOutcome struct {
		result DispatchResult
		err    error
	}
	outcomes := make(chan dispatchOutcome, 2)
	for _, id := range []engineering.ID{"A", "B"} {
		id := id
		go func() {
			res, err := s.Dispatch(context.Background(), "project-1", request(id))
			outcomes <- dispatchOutcome{result: res, err: err}
		}()
	}

	for i := 0; i < 2; i++ {
		select {
		case <-h.started:
		case <-time.After(2 * time.Second):
			close(h.release)
			t.Fatal("independent task executions were serialized by the control-plane mutation lock")
		}
	}
	if h.max.Load() < 2 {
		close(h.release)
		t.Fatalf("max concurrent harness runs=%d, want >=2", h.max.Load())
	}
	close(h.release)

	for i := 0; i < 2; i++ {
		out := <-outcomes
		if out.err != nil {
			t.Fatalf("Dispatch: %v", out.err)
		}
	}
	persisted, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []engineering.ID{"A", "B"} {
		node, _ := findNode(*persisted.ActiveDAG, id)
		if node.State != engineering.TaskDone {
			t.Fatalf("task %s state=%s, want done", id, node.State)
		}
	}
}

func TestDuplicateDispatchCannotClaimRunningTask(t *testing.T) {
	h := &parallelHarnessV151{started: make(chan engineering.ID, 1), release: make(chan struct{})}
	s, _ := newParallelProjectV151(t, []engineering.TaskNode{{ID: "A", ParallelEligible: true}}, h)
	firstDone := make(chan error, 1)
	go func() {
		_, err := s.Dispatch(context.Background(), "project-1", request("A"))
		firstDone <- err
	}()

	select {
	case <-h.started:
	case <-time.After(2 * time.Second):
		t.Fatal("first dispatch did not start")
	}
	_, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if !errors.Is(err, ErrTaskNotReady) {
		close(h.release)
		t.Fatalf("duplicate dispatch error=%v, want ErrTaskNotReady", err)
	}
	close(h.release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first dispatch: %v", err)
	}
}

func TestLocalSchedulerBoundsParallelBatchAndLeavesRemainingReady(t *testing.T) {
	h := &parallelHarnessV151{started: make(chan engineering.ID, 3), release: make(chan struct{})}
	s, repo := newParallelProjectV151(t, []engineering.TaskNode{
		{ID: "A", ParallelEligible: true},
		{ID: "B", ParallelEligible: true},
		{ID: "C", ParallelEligible: true},
	}, h)
	factory := TaskRequestFactoryFunc(func(_ context.Context, _ ProjectSnapshot, node engineering.TaskNode) (taskexecution.TaskExecutionRequest, error) {
		return request(node.ID), nil
	})
	scheduler, err := NewLocalScheduler(s, factory, 2)
	if err != nil {
		t.Fatal(err)
	}

	type runOutcome struct {
		results []DispatchResult
		err     error
	}
	done := make(chan runOutcome, 1)
	go func() {
		results, err := scheduler.RunOnce(context.Background(), "project-1")
		done <- runOutcome{results: results, err: err}
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-h.started:
		case <-time.After(2 * time.Second):
			close(h.release)
			t.Fatal("bounded scheduler batch did not start two parallel tasks")
		}
	}
	select {
	case id := <-h.started:
		close(h.release)
		t.Fatalf("scheduler exceeded MaxConcurrent; unexpected third task %s", id)
	default:
	}
	close(h.release)
	out := <-done
	if out.err != nil {
		t.Fatal(out.err)
	}
	if len(out.results) != 2 {
		t.Fatalf("results=%d, want 2", len(out.results))
	}
	persisted, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	ready := persisted.ActiveDAG.ReadyNodes()
	if len(ready) != 1 || ready[0].ID != "C" {
		t.Fatalf("ready=%+v, want only C", ready)
	}
}

func TestLocalSchedulerNonParallelNodeIsExclusive(t *testing.T) {
	batch := selectSchedulerBatch([]engineering.TaskNode{
		{ID: "A", ParallelEligible: true},
		{ID: "B", ParallelEligible: false},
		{ID: "C", ParallelEligible: true},
	}, 3)
	if len(batch) != 1 || batch[0].ID != "B" {
		t.Fatalf("batch=%+v, want exclusive B", batch)
	}
}

func TestLocalSchedulerStartupRecoveryPersistsInterrupted(t *testing.T) {
	h := &fakeTaskHarness{}
	s, repo := newParallelProjectV151(t, []engineering.TaskNode{{ID: "A", ParallelEligible: true}}, h)
	ctx := context.Background()
	snap, err := repo.Load(ctx, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	running, err := snap.ActiveDAG.Transition("A", engineering.TaskRunning, "")
	if err != nil {
		t.Fatal(err)
	}
	snap.ActiveDAG = &running
	if err := repo.Save(ctx, snap); err != nil {
		t.Fatal(err)
	}

	factory := TaskRequestFactoryFunc(func(_ context.Context, _ ProjectSnapshot, node engineering.TaskNode) (taskexecution.TaskExecutionRequest, error) {
		return request(node.ID), nil
	})
	scheduler, err := NewLocalScheduler(s, factory, 1)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := scheduler.RecoverStartup(ctx, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	node, _ := findNode(*recovered.ActiveDAG, "A")
	if node.State != engineering.TaskInterrupted {
		t.Fatalf("state=%s, want interrupted", node.State)
	}
	if len(recovered.ActiveDAG.ReadyNodes()) != 0 {
		t.Fatal("startup recovery must not silently restart interrupted work")
	}
}

func TestLocalSchedulerSerializesRunOnceCalls(t *testing.T) {
	// The runMu is the in-process ownership seam that prevents two scheduler
	// iterations from violating non-parallel exclusion. This test exercises it
	// indirectly with a blocking request factory.
	h := &fakeTaskHarness{result: HarnessResult{Accepted: true, Facts: engineering.CompletionFacts{State: engineering.NodeAccepted}}}
	s, _ := newParallelProjectV151(t, []engineering.TaskNode{{ID: "A"}}, h)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var once sync.Once
	factory := TaskRequestFactoryFunc(func(_ context.Context, _ ProjectSnapshot, node engineering.TaskNode) (taskexecution.TaskExecutionRequest, error) {
		once.Do(func() { entered <- struct{}{} })
		<-release
		return request(node.ID), nil
	})
	scheduler, err := NewLocalScheduler(s, factory, 1)
	if err != nil {
		t.Fatal(err)
	}
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { _, err := scheduler.RunOnce(context.Background(), "project-1"); first <- err }()
	<-entered
	go func() { _, err := scheduler.RunOnce(context.Background(), "project-1"); second <- err }()
	select {
	case err := <-second:
		close(release)
		t.Fatalf("second RunOnce escaped scheduler ownership early: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}
}
