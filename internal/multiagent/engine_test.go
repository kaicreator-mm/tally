package multiagent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type scriptedStep struct {
	status   taskexecution.ExecutionStatus
	evidence []engineering.EvidenceRef
	err      error
	session  agentruntime.SessionID
	run      agentruntime.RunID
}

type scriptedExecutor struct {
	mu    sync.Mutex
	steps []scriptedStep
	calls []taskexecution.TaskExecutionRequest
}

func (x *scriptedExecutor) Execute(_ context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.calls = append(x.calls, req)
	if len(x.steps) == 0 {
		return taskexecution.TaskExecutionResult{}, errors.New("test: no scripted step")
	}
	step := x.steps[0]
	x.steps = x.steps[1:]
	if step.status == "" {
		step.status = taskexecution.ExecutionCompleted
	}
	result := taskexecution.TaskExecutionResult{
		AgentClaim:   agentruntime.AgentRunResult{SessionID: step.session, RunID: step.run, Status: agentruntime.RunCompleted},
		Runtime:      req.Runtime,
		Environment:  req.Environment,
		Workspace:    req.Workspace,
		Status:       step.status,
		EvidenceRefs: append([]engineering.EvidenceRef(nil), step.evidence...),
	}
	return result, step.err
}

func (x *scriptedExecutor) Cancel(engineering.ID) error { return nil }

func testEvidence(id string, status engineering.EvidenceStatus) engineering.EvidenceRef {
	return engineering.EvidenceRef{ID: engineering.ID(id), Kind: "test", URI: "memory://" + id, Status: status}
}

func testRequest(id engineering.ID) taskexecution.TaskExecutionRequest {
	return taskexecution.TaskExecutionRequest{
		TaskID:             id,
		FrozenTaskContract: engineering.ArtifactRef{ID: engineering.ID(string(id) + ":contract"), Kind: engineering.ArtifactTaskContract, URI: "memory://contract"},
		Runtime:            taskexecution.RuntimeRef{ID: "runtime-a"},
		Environment:        taskexecution.EnvironmentRef{Kind: taskexecution.EnvironmentLocal, Target: "local"},
		Workspace:          taskexecution.WorkspaceRef{Environment: taskexecution.EnvironmentLocal, Target: "local", Identity: "ws-" + string(id), Path: "/tmp/" + string(id), BaseRevision: "base"},
		Instruction:        "implement the frozen task contract",
	}
}

func testClock() func() time.Time {
	current := time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC)
	return func() time.Time {
		current = current.Add(time.Second)
		return current
	}
}

func TestEngineSequentialHappyPath(t *testing.T) {
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "s1", run: "r1"},
		{session: "s2", run: "r2", evidence: []engineering.EvidenceRef{testEvidence("review-pass", engineering.EvidencePass)}},
		{session: "s3", run: "r3", evidence: []engineering.EvidenceRef{testEvidence("verify-pass", engineering.EvidencePass)}},
	}}
	engine, err := NewEngine(exec, nil, Config{MaxConcurrency: 2, Now: testClock()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Run(context.Background(), testRequest("task-1"), DefaultTaskPolicy(), WorkflowState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified || result.State.Status != WorkflowVerified {
		t.Fatalf("expected verified, got %+v", result.State)
	}
	roles := []Role{RoleImplementer, RoleReviewer, RoleVerifier}
	if len(result.State.Runs) != len(roles) {
		t.Fatalf("runs=%d", len(result.State.Runs))
	}
	for i, role := range roles {
		if result.State.Runs[i].Role != role {
			t.Fatalf("run %d role=%s", i, result.State.Runs[i].Role)
		}
	}
	if len(result.Facts.Evidence) != 2 {
		t.Fatalf("evidence=%d", len(result.Facts.Evidence))
	}
}

func TestEngineReviewerRepairLoop(t *testing.T) {
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "s1", run: "r1"},
		{session: "s2", run: "r2", evidence: []engineering.EvidenceRef{testEvidence("review-fail", engineering.EvidenceFail)}},
		{session: "s3", run: "r3"},
		{session: "s4", run: "r4", evidence: []engineering.EvidenceRef{testEvidence("review-pass", engineering.EvidencePass)}},
		{session: "s5", run: "r5", evidence: []engineering.EvidenceRef{testEvidence("verify-pass", engineering.EvidencePass)}},
	}}
	engine, _ := NewEngine(exec, nil, Config{Now: testClock()})
	result, err := engine.Run(context.Background(), testRequest("task-repair"), DefaultTaskPolicy(), WorkflowState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Role{RoleImplementer, RoleReviewer, RoleRepairer, RoleReviewer, RoleVerifier}
	if len(result.State.Runs) != len(want) {
		t.Fatalf("runs=%d", len(result.State.Runs))
	}
	for i := range want {
		if result.State.Runs[i].Role != want[i] {
			t.Fatalf("run %d role=%s want=%s", i, result.State.Runs[i].Role, want[i])
		}
	}
	if result.State.RepairAttempts != 1 || !result.State.Findings[0].Resolved {
		t.Fatalf("repair state=%+v findings=%+v", result.State, result.State.Findings)
	}
}

func TestEngineRepairFailureBlocksWithoutConsumingInfraFailureAsQualityAttempt(t *testing.T) {
	providerErr := errors.New("provider unavailable")
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "s1", run: "r1"},
		{session: "s2", run: "r2", evidence: []engineering.EvidenceRef{testEvidence("review-fail", engineering.EvidenceFail)}},
		{err: providerErr, session: "s3", run: "r3"},
	}}
	engine, _ := NewEngine(exec, nil, Config{Now: testClock()})
	result, err := engine.Run(context.Background(), testRequest("task-provider"), DefaultTaskPolicy(), WorkflowState{}, nil)
	if !errors.Is(err, providerErr) {
		t.Fatalf("err=%v", err)
	}
	if result.State.Status != WorkflowBlocked {
		t.Fatalf("status=%s", result.State.Status)
	}
	if result.State.RepairAttempts != 0 {
		t.Fatalf("infra failure consumed repair budget: %d", result.State.RepairAttempts)
	}
}

func TestEngineVerifierDisagreementReturnsToRepair(t *testing.T) {
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "s1", run: "r1"},
		{session: "s2", run: "r2", evidence: []engineering.EvidenceRef{testEvidence("review-pass-a", engineering.EvidencePass)}},
		{session: "s3", run: "r3", evidence: []engineering.EvidenceRef{testEvidence("verify-fail", engineering.EvidenceFail)}},
		{session: "s4", run: "r4"},
		{session: "s5", run: "r5", evidence: []engineering.EvidenceRef{testEvidence("review-pass-b", engineering.EvidencePass)}},
		{session: "s6", run: "r6", evidence: []engineering.EvidenceRef{testEvidence("verify-pass", engineering.EvidencePass)}},
	}}
	engine, _ := NewEngine(exec, nil, Config{Now: testClock()})
	result, err := engine.Run(context.Background(), testRequest("task-verify-repair"), DefaultTaskPolicy(), WorkflowState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified || result.State.RepairAttempts != 1 {
		t.Fatalf("state=%+v", result.State)
	}
	if result.State.Runs[3].Role != RoleRepairer {
		t.Fatalf("expected repair after verifier, got %s", result.State.Runs[3].Role)
	}
}

func TestEngineIndependentRunConstraint(t *testing.T) {
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "same", run: "same-run"},
		{session: "same", run: "review-run", evidence: []engineering.EvidenceRef{testEvidence("review-pass", engineering.EvidencePass)}},
	}}
	engine, _ := NewEngine(exec, nil, Config{Now: testClock()})
	result, err := engine.Run(context.Background(), testRequest("task-independent"), DefaultTaskPolicy(), WorkflowState{}, nil)
	if !errors.Is(err, ErrIndependenceViolation) {
		t.Fatalf("err=%v", err)
	}
	if result.State.Status != WorkflowBlocked {
		t.Fatalf("status=%s", result.State.Status)
	}
}

func TestEngineMissingReviewerEvidenceBlocks(t *testing.T) {
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "s1", run: "r1"},
		{session: "s2", run: "r2"},
	}}
	engine, _ := NewEngine(exec, nil, Config{Now: testClock()})
	result, err := engine.Run(context.Background(), testRequest("task-no-evidence"), DefaultTaskPolicy(), WorkflowState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Status != WorkflowBlocked || !strings.Contains(result.State.Reason, "no Tally evidence") {
		t.Fatalf("state=%+v", result.State)
	}
}

func TestEngineRecoversInFlightReviewerWithoutSessionResume(t *testing.T) {
	now := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	state := WorkflowState{
		TaskID: "task-restart", Status: WorkflowActive, NextRole: RoleReviewer, Revision: 3, UpdatedAt: now,
		Runs: []RoleRun{
			{ID: "task-restart:implementer:001", Sequence: 1, Role: RoleImplementer, Status: RunPassed, StartedAt: now, FinishedAt: now.Add(time.Second), Execution: taskexecution.TaskExecutionResult{AgentClaim: agentruntime.AgentRunResult{SessionID: "s1", RunID: "r1"}, Runtime: taskexecution.RuntimeRef{ID: "runtime-a"}, Status: taskexecution.ExecutionCompleted}},
			{ID: "task-restart:reviewer:002", Sequence: 2, Role: RoleReviewer, Status: RunRunning, StartedAt: now.Add(2 * time.Second)},
		},
	}
	exec := &scriptedExecutor{steps: []scriptedStep{
		{session: "s2", run: "r2", evidence: []engineering.EvidenceRef{testEvidence("review-pass", engineering.EvidencePass)}},
		{session: "s3", run: "r3", evidence: []engineering.EvidenceRef{testEvidence("verify-pass", engineering.EvidencePass)}},
	}}
	engine, _ := NewEngine(exec, nil, Config{Now: testClock()})
	var events []Event
	result, err := engine.Run(context.Background(), testRequest("task-restart"), DefaultTaskPolicy(), state, func(_ context.Context, _ WorkflowState, event Event) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Verified {
		t.Fatalf("state=%+v", result.State)
	}
	if result.State.Runs[1].Status != RunInterrupted || result.State.Runs[2].Role != RoleReviewer {
		t.Fatalf("runs=%+v", result.State.Runs)
	}
	if len(exec.calls) == 0 || exec.calls[0].Policy.ResumeSession != "" {
		t.Fatalf("review resumed old Agent session")
	}
	if len(events) == 0 || events[0].Kind != "role.recovered_interrupted" {
		t.Fatalf("events=%+v", events)
	}
}
