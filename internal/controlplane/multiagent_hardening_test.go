package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/multiagent"
	"tally/internal/taskexecution"
)

func TestMultiAgentDoesNotHijackRunningLegacyTask(t *testing.T) {
	exec := &v154RoleExecutor{reviewerEvidence: true}
	s, repo := newV154Service(t, exec, 2)
	bootstrap(t, s)

	snap, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	dag, err := snap.ActiveDAG.Transition("A", engineering.TaskRunning, "")
	if err != nil {
		t.Fatal(err)
	}
	snap.ActiveDAG = &dag
	delete(snap.MultiAgentWorkflows, "A")
	if err := repo.Save(context.Background(), snap); err != nil {
		t.Fatal(err)
	}

	if _, err := s.DispatchMultiAgent(context.Background(), "project-1", request("A")); !errors.Is(err, ErrTaskNotReady) {
		t.Fatalf("dispatch running legacy task err=%v, want ErrTaskNotReady", err)
	}
	if _, err := s.ConfigureMultiAgentTask(context.Background(), "project-1", "A", multiagent.DefaultTaskPolicy()); !errors.Is(err, multiagent.ErrAlreadyActive) {
		t.Fatalf("configure running task err=%v, want ErrAlreadyActive", err)
	}

	persisted, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := persisted.MultiAgentWorkflows["A"]; ok {
		t.Fatal("multi-agent dispatch created role state for a running legacy task")
	}
	node, _ := findNode(*persisted.ActiveDAG, "A")
	if node.State != engineering.TaskRunning {
		t.Fatalf("legacy running task state=%s", node.State)
	}
}

type queuedControlPlaneExecutor struct {
	mu      sync.Mutex
	seq     int
	entered chan engineering.ID
	release chan struct{}
}

func newQueuedControlPlaneExecutor() *queuedControlPlaneExecutor {
	return &queuedControlPlaneExecutor{
		entered: make(chan engineering.ID, 4),
		release: make(chan struct{}, 4),
	}
}

func (x *queuedControlPlaneExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	x.mu.Lock()
	x.seq++
	seq := x.seq
	x.mu.Unlock()
	x.entered <- req.TaskID
	select {
	case <-x.release:
	case <-ctx.Done():
		return taskexecution.TaskExecutionResult{Runtime: req.Runtime, Environment: req.Environment, Workspace: req.Workspace, Status: taskexecution.ExecutionInterrupted}, nil
	}
	return taskexecution.TaskExecutionResult{
		AgentClaim: agentruntime.AgentRunResult{
			SessionID: agentruntime.SessionID(fmt.Sprintf("queued-session-%d", seq)),
			RunID:     agentruntime.RunID(fmt.Sprintf("queued-run-%d", seq)),
			Status:    agentruntime.RunCompleted,
		},
		Runtime: req.Runtime, Environment: req.Environment, Workspace: req.Workspace,
		Status: taskexecution.ExecutionCompleted,
	}, nil
}

func (x *queuedControlPlaneExecutor) Cancel(engineering.ID) error { return nil }

type blockAfterOneRole struct{}

func (blockAfterOneRole) Interpret(context.Context, multiagent.RoleContext, taskexecution.TaskExecutionResult) (multiagent.Decision, error) {
	return multiagent.Decision{Kind: multiagent.DecisionBlock, Reason: "test stops after first role"}, nil
}

func TestRestartQueuedMultiAgentKeepsTaskReadyAndPersistsInterruptedCollaboration(t *testing.T) {
	repo, err := NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exec := newQueuedControlPlaneExecutor()
	clock := v154Clock()
	engine, err := multiagent.NewEngine(exec, blockAfterOneRole{}, multiagent.Config{MaxConcurrency: 1, Now: clock})
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewServiceWithMultiAgent(repo, engine)
	if err != nil {
		t.Fatal(err)
	}
	s.Now = clock
	bootstrapV154Parallel(t, s)

	aReq := request("A")
	aReq.Workspace.Identity, aReq.Workspace.Path = "ws-A", "/tmp/ws-A"
	bReq := request("B")
	bReq.Workspace.Identity, bReq.Workspace.Path = "ws-B", "/tmp/ws-B"

	aDone := make(chan error, 1)
	go func() {
		_, runErr := s.DispatchMultiAgent(context.Background(), "project-1", aReq)
		aDone <- runErr
	}()
	if id := <-exec.entered; id != "A" {
		t.Fatalf("first executing task=%s", id)
	}

	bDone := make(chan error, 1)
	go func() {
		_, runErr := s.DispatchMultiAgent(context.Background(), "project-1", bReq)
		bDone <- runErr
	}()
	deadline := time.Now().Add(time.Second)
	for !engine.IsActive("B") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !engine.IsActive("B") {
		t.Fatal("B never became an active queued collaboration")
	}

	restarted, err := s.RestartTask(context.Background(), "project-1", "B")
	if err != nil {
		t.Fatal(err)
	}
	bNode, _ := findNode(*restarted.ActiveDAG, "B")
	if bNode.State != engineering.TaskReady {
		t.Fatalf("B state after RestartTask=%s, want ready", bNode.State)
	}

	select {
	case runErr := <-bDone:
		if !errors.Is(runErr, context.Canceled) {
			t.Fatalf("queued B dispatch err=%v, want context.Canceled", runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("queued B dispatch did not unwind after RestartTask")
	}

	persisted, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	bNode, _ = findNode(*persisted.ActiveDAG, "B")
	if bNode.State != engineering.TaskReady {
		t.Fatalf("late finalize regressed B state to %s", bNode.State)
	}
	workflow := persisted.MultiAgentWorkflows["B"]
	if workflow.Status != multiagent.WorkflowInterrupted {
		t.Fatalf("B collaboration status=%s, want interrupted", workflow.Status)
	}

	exec.release <- struct{}{}
	select {
	case runErr := <-aDone:
		if runErr != nil {
			t.Fatalf("A dispatch error: %v", runErr)
		}
	case <-time.After(time.Second):
		t.Fatal("A dispatch did not finish after release")
	}
}