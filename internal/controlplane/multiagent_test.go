package controlplane

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/multiagent"
	"tally/internal/taskexecution"
)

type v154RoleExecutor struct {
	mu               sync.Mutex
	calls            int
	active           int
	maxActive        int
	delay            time.Duration
	reviewerEvidence bool
	rendezvous       *v154Rendezvous
}

// v154Rendezvous releases every waiting party once exactly n parties have
// arrived. It makes the parallel-overlap assertion deterministic: both
// implementer role executions must be in flight at the same time, with no
// sleep window that a loaded host can miss.
type v154Rendezvous struct {
	arrive chan struct{}
	open   chan struct{}
}

func newV154Rendezvous(n int) *v154Rendezvous {
	r := &v154Rendezvous{arrive: make(chan struct{}, n), open: make(chan struct{})}
	go func() {
		for i := 0; i < n; i++ {
			<-r.arrive
		}
		close(r.open)
	}()
	return r
}

func (r *v154Rendezvous) wait(ctx context.Context) bool {
	r.arrive <- struct{}{}
	select {
	case <-r.open:
		return true
	case <-ctx.Done():
		return false
	}
}

func (x *v154RoleExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	x.mu.Lock()
	x.calls++
	call := x.calls
	x.active++
	if x.active > x.maxActive {
		x.maxActive = x.active
	}
	x.mu.Unlock()
	defer func() {
		x.mu.Lock()
		x.active--
		x.mu.Unlock()
	}()
	if x.rendezvous != nil && strings.Contains(req.Instruction, "role: IMPLEMENTER") {
		if !x.rendezvous.wait(ctx) {
			return taskexecution.TaskExecutionResult{Runtime: req.Runtime, Environment: req.Environment, Workspace: req.Workspace, Status: taskexecution.ExecutionInterrupted}, nil
		}
	}
	if x.delay > 0 {
		select {
		case <-time.After(x.delay):
		case <-ctx.Done():
			return taskexecution.TaskExecutionResult{Runtime: req.Runtime, Environment: req.Environment, Workspace: req.Workspace, Status: taskexecution.ExecutionInterrupted}, nil
		}
	}
	result := taskexecution.TaskExecutionResult{
		AgentClaim: agentruntime.AgentRunResult{
			SessionID: agentruntime.SessionID(fmt.Sprintf("%s-session-%d", req.TaskID, call)),
			RunID:     agentruntime.RunID(fmt.Sprintf("%s-run-%d", req.TaskID, call)),
			Status:    agentruntime.RunCompleted,
		},
		Runtime:     req.Runtime,
		Environment: req.Environment,
		Workspace:   req.Workspace,
		Status:      taskexecution.ExecutionCompleted,
	}
	switch {
	case strings.Contains(req.Instruction, "role: REVIEWER"):
		if x.reviewerEvidence {
			result.EvidenceRefs = []engineering.EvidenceRef{{ID: engineering.ID("review-" + string(req.TaskID)), Kind: "review", URI: "memory://review", Status: engineering.EvidencePass}}
		}
	case strings.Contains(req.Instruction, "role: VERIFIER"):
		result.EvidenceRefs = []engineering.EvidenceRef{{ID: engineering.ID("verify-" + string(req.TaskID)), Kind: "verification", URI: "memory://verify", Status: engineering.EvidencePass}}
	}
	return result, nil
}

func (x *v154RoleExecutor) Cancel(engineering.ID) error { return nil }

func v154Clock() func() time.Time {
	var mu sync.Mutex
	n := int64(1000)
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		n++
		return time.Unix(n, 0).UTC()
	}
}

func newV154Service(t *testing.T, exec *v154RoleExecutor, concurrency int) (*Service, *FileRepository) {
	t.Helper()
	repo, err := NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := v154Clock()
	engine, err := multiagent.NewEngine(exec, nil, multiagent.Config{MaxConcurrency: concurrency, Now: clock})
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewServiceWithMultiAgent(repo, engine)
	if err != nil {
		t.Fatal(err)
	}
	s.Now = clock
	return s, repo
}

func bootstrapV154Parallel(t *testing.T, s *Service) {
	t.Helper()
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
	_, err := s.ActivateDAG(ctx, "project-1", engineering.TaskDAGProposal{RevisionID: "dag-v154", Nodes: []engineering.TaskNode{
		{ID: "A", ParallelEligible: true},
		{ID: "B", ParallelEligible: true},
	}}, map[engineering.ID]engineering.CompletionContract{
		"A": {RequiredState: engineering.NodeAccepted, RequiredEvidence: []engineering.ID{"verify-A"}},
		"B": {RequiredState: engineering.NodeAccepted, RequiredEvidence: []engineering.ID{"verify-B"}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultiAgentDispatchUsesExistingCompletionContractForAcceptance(t *testing.T) {
	exec := &v154RoleExecutor{reviewerEvidence: true}
	s, repo := newV154Service(t, exec, 2)
	bootstrap(t, s)

	res, err := s.DispatchMultiAgent(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Collaboration == nil || res.Collaboration.Status != multiagent.WorkflowAccepted {
		t.Fatalf("collaboration=%+v", res.Collaboration)
	}
	if res.Completion == nil || !res.Completion.Satisfied {
		t.Fatalf("completion=%+v", res.Completion)
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
	if persisted.MultiAgentWorkflows["A"].Status != multiagent.WorkflowAccepted {
		t.Fatalf("persisted workflow=%+v", persisted.MultiAgentWorkflows["A"])
	}
}

func TestMultiAgentReviewerClaimWithoutEvidenceCannotAdvance(t *testing.T) {
	exec := &v154RoleExecutor{reviewerEvidence: false}
	s, _ := newV154Service(t, exec, 2)
	bootstrap(t, s)

	res, err := s.DispatchMultiAgent(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Collaboration == nil || res.Collaboration.Status != multiagent.WorkflowBlocked {
		t.Fatalf("collaboration=%+v", res.Collaboration)
	}
	node, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if node.State != engineering.TaskBlocked {
		t.Fatalf("task state=%s", node.State)
	}
}

func TestMultiAgentBatchRunsTwoReadyNonConflictingTasksInParallel(t *testing.T) {
	exec := &v154RoleExecutor{reviewerEvidence: true, rendezvous: newV154Rendezvous(2)}
	s, _ := newV154Service(t, exec, 2)
	bootstrapV154Parallel(t, s)

	a := request("A")
	a.Workspace.Identity, a.Workspace.Path = "ws-A", "/tmp/ws-A"
	b := request("B")
	b.Workspace.Identity, b.Workspace.Path = "ws-B", "/tmp/ws-B"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := s.DispatchReadyMultiAgent(ctx, "project-1", map[engineering.ID]taskexecution.TaskExecutionRequest{"A": a, "B": b})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 2 || len(res.Errors) != 0 {
		t.Fatalf("results=%d errors=%v", len(res.Results), res.Errors)
	}
	for _, id := range []engineering.ID{"A", "B"} {
		node, _ := findNode(*res.Snapshot.ActiveDAG, id)
		if node.State != engineering.TaskDone {
			t.Fatalf("task %s state=%s", id, node.State)
		}
	}
	exec.mu.Lock()
	maxActive := exec.maxActive
	exec.mu.Unlock()
	if maxActive < 2 {
		t.Fatalf("expected actual parallel execution, maxActive=%d", maxActive)
	}
}

func TestMultiAgentBatchSuppressesConfiguredConflict(t *testing.T) {
	exec := &v154RoleExecutor{reviewerEvidence: true}
	s, _ := newV154Service(t, exec, 2)
	bootstrapV154Parallel(t, s)
	policy := multiagent.DefaultTaskPolicy()
	policy.ConflictKeys = []string{"file:shared.go"}
	if _, err := s.ConfigureMultiAgentTask(context.Background(), "project-1", "A", policy); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfigureMultiAgentTask(context.Background(), "project-1", "B", policy); err != nil {
		t.Fatal(err)
	}
	a := request("A")
	a.Workspace.Identity, a.Workspace.Path = "ws-A", "/tmp/ws-A"
	b := request("B")
	b.Workspace.Identity, b.Workspace.Path = "ws-B", "/tmp/ws-B"
	res, err := s.DispatchReadyMultiAgent(context.Background(), "project-1", map[engineering.ID]taskexecution.TaskExecutionRequest{"A": a, "B": b})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("results=%d", len(res.Results))
	}
	bNode, _ := findNode(*res.Snapshot.ActiveDAG, "B")
	if bNode.State != engineering.TaskReady {
		t.Fatalf("conflicting task B advanced early: %s", bNode.State)
	}
}
