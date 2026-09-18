package multiagent

import (
	"context"
	"testing"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type routingSpyExecutor struct {
	calls   int
	cancels int
}

func (s *routingSpyExecutor) Execute(_ context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	s.calls++
	return taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted, Workspace: req.Workspace}, nil
}

func (s *routingSpyExecutor) Cancel(engineering.ID) error {
	s.cancels++
	return nil
}

func TestRoleExecutorRouterUsesVerificationOnlyForReviewerAndVerifier(t *testing.T) {
	ordinary := &routingSpyExecutor{}
	verified := &routingSpyExecutor{}
	router, err := NewRoleExecutorRouter(ordinary, verified)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		instruction string
		verified    bool
	}{
		{"Tally engineering role: IMPLEMENTER.", false},
		{"Tally engineering role: REPAIRER.", false},
		{"Tally engineering role: SPECIALIST (security).", false},
		{"Tally engineering role: REVIEWER.", true},
		{"Tally engineering role: VERIFIER.", true},
		{"ordinary execution without a role directive", false},
	}

	ordinaryWant := 0
	verifiedWant := 0
	for _, tc := range cases {
		if _, err := router.Execute(context.Background(), taskexecution.TaskExecutionRequest{Instruction: tc.instruction}); err != nil {
			t.Fatalf("execute %q: %v", tc.instruction, err)
		}
		if tc.verified {
			verifiedWant++
		} else {
			ordinaryWant++
		}
		if ordinary.calls != ordinaryWant || verified.calls != verifiedWant {
			t.Fatalf("after %q ordinary=%d/%d verified=%d/%d", tc.instruction, ordinary.calls, ordinaryWant, verified.calls, verifiedWant)
		}
	}
}

func TestRoleExecutorRouterCancelUsesProductionExecutorOnce(t *testing.T) {
	ordinary := &routingSpyExecutor{}
	verified := &routingSpyExecutor{}
	router, err := NewRoleExecutorRouter(ordinary, verified)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.Cancel("task-1"); err != nil {
		t.Fatal(err)
	}
	if ordinary.cancels != 1 || verified.cancels != 0 {
		t.Fatalf("cancel routing ordinary=%d verified=%d", ordinary.cancels, verified.cancels)
	}
}
