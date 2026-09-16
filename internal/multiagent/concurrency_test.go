package multiagent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type concurrencyExecutor struct {
	mu        sync.Mutex
	active    int
	maxActive int
	entered   chan engineering.ID
	release   chan struct{}
	seq       int
}

func newConcurrencyExecutor(buffer int) *concurrencyExecutor {
	return &concurrencyExecutor{
		entered: make(chan engineering.ID, buffer),
		release: make(chan struct{}, buffer),
	}
}

func (x *concurrencyExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	x.mu.Lock()
	x.active++
	x.seq++
	seq := x.seq
	if x.active > x.maxActive {
		x.maxActive = x.active
	}
	x.mu.Unlock()

	x.entered <- req.TaskID
	select {
	case <-x.release:
	case <-ctx.Done():
		x.mu.Lock()
		x.active--
		x.mu.Unlock()
		return taskexecution.TaskExecutionResult{Runtime: req.Runtime, Environment: req.Environment, Workspace: req.Workspace, Status: taskexecution.ExecutionInterrupted}, nil
	}

	x.mu.Lock()
	x.active--
	x.mu.Unlock()
	return taskexecution.TaskExecutionResult{
		AgentClaim: agentruntime.AgentRunResult{
			SessionID: agentruntime.SessionID(fmt.Sprintf("session-%d", seq)),
			RunID:     agentruntime.RunID(fmt.Sprintf("run-%d", seq)),
			Status:    agentruntime.RunCompleted,
		},
		Runtime: req.Runtime, Environment: req.Environment, Workspace: req.Workspace,
		Status: taskexecution.ExecutionCompleted,
	}, nil
}

func (x *concurrencyExecutor) Cancel(engineering.ID) error { return nil }

func (x *concurrencyExecutor) maximum() int {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.maxActive
}

type stopAfterFirstRoleInterpreter struct{}

func (stopAfterFirstRoleInterpreter) Interpret(context.Context, RoleContext, taskexecution.TaskExecutionResult) (Decision, error) {
	return Decision{Kind: DecisionBlock, Reason: "test stops after one role"}, nil
}

func TestEngineEnforcesGlobalConcurrencyLimit(t *testing.T) {
	exec := newConcurrencyExecutor(3)
	engine, err := NewEngine(exec, stopAfterFirstRoleInterpreter{}, Config{MaxConcurrency: 2, Now: testClock()})
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 3)
	for _, id := range []engineering.ID{"concurrency-A", "concurrency-B", "concurrency-C"} {
		id := id
		go func() {
			_, runErr := engine.Run(context.Background(), testRequest(id), DefaultTaskPolicy(), WorkflowState{}, nil)
			errCh <- runErr
		}()
	}

	<-exec.entered
	<-exec.entered
	select {
	case id := <-exec.entered:
		t.Fatalf("third task %s entered executor before a concurrency slot was released", id)
	case <-time.After(40 * time.Millisecond):
	}

	exec.release <- struct{}{}
	select {
	case <-exec.entered:
	case <-time.After(time.Second):
		t.Fatal("third task did not enter after a concurrency slot was released")
	}
	exec.release <- struct{}{}
	exec.release <- struct{}{}

	for i := 0; i < 3; i++ {
		if runErr := <-errCh; runErr != nil {
			t.Fatalf("run error: %v", runErr)
		}
	}
	if got := exec.maximum(); got != 2 {
		t.Fatalf("max concurrent executions=%d, want 2", got)
	}
}

func TestEngineCanCancelCollaborationQueuedForConcurrencySlot(t *testing.T) {
	exec := newConcurrencyExecutor(2)
	engine, err := NewEngine(exec, stopAfterFirstRoleInterpreter{}, Config{MaxConcurrency: 1, Now: testClock()})
	if err != nil {
		t.Fatal(err)
	}

	firstDone := make(chan error, 1)
	go func() {
		_, runErr := engine.Run(context.Background(), testRequest("queue-first"), DefaultTaskPolicy(), WorkflowState{}, nil)
		firstDone <- runErr
	}()
	if id := <-exec.entered; id != "queue-first" {
		t.Fatalf("first executor entry=%s", id)
	}

	secondDone := make(chan error, 1)
	go func() {
		_, runErr := engine.Run(context.Background(), testRequest("queue-second"), DefaultTaskPolicy(), WorkflowState{}, nil)
		secondDone <- runErr
	}()

	deadline := time.Now().Add(time.Second)
	for !engine.IsActive("queue-second") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !engine.IsActive("queue-second") {
		t.Fatal("queued collaboration was never registered active")
	}
	if err := engine.Cancel("queue-second"); err != nil {
		t.Fatalf("cancel queued collaboration: %v", err)
	}
	select {
	case runErr := <-secondDone:
		if runErr == nil {
			t.Fatal("queued collaboration cancellation returned nil error")
		}
	case <-time.After(time.Second):
		t.Fatal("queued collaboration did not stop after cancellation")
	}
	select {
	case id := <-exec.entered:
		t.Fatalf("cancelled queued collaboration unexpectedly entered executor: %s", id)
	default:
	}

	exec.release <- struct{}{}
	if runErr := <-firstDone; runErr != nil {
		t.Fatalf("first run error: %v", runErr)
	}
}
