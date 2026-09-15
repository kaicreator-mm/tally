package controlplane

import (
	"context"
	"errors"
	"testing"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type executionOnlyExecutorStub struct {
	result      taskexecution.TaskExecutionResult
	err         error
	cancelID    engineering.ID
	cancelError error
}

func (s *executionOnlyExecutorStub) Execute(context.Context, taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	return s.result, s.err
}

func (s *executionOnlyExecutorStub) Cancel(id engineering.ID) error {
	s.cancelID = id
	return s.cancelError
}

func TestExecutionOnlyHarnessNeverAcceptsAgentCompletion(t *testing.T) {
	exec := &executionOnlyExecutorStub{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}
	h, err := NewExecutionOnlyHarness(exec)
	if err != nil {
		t.Fatal(err)
	}
	result, err := h.Run(context.Background(), "project-1", request("TASK-1"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted {
		t.Fatal("Agent completion must not be promoted to accepted without independent verification")
	}
	if result.Interrupted {
		t.Fatal("completed execution must not be reported interrupted")
	}
	if result.Reason != verificationUnavailableReason {
		t.Fatalf("reason=%q, want explicit verification-unavailable block", result.Reason)
	}
}

func TestExecutionOnlyHarnessPreservesFailureAndInterruption(t *testing.T) {
	t.Run("failed", func(t *testing.T) {
		h, err := NewExecutionOnlyHarness(&executionOnlyExecutorStub{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionFailed}})
		if err != nil {
			t.Fatal(err)
		}
		result, err := h.Run(context.Background(), "project-1", request("TASK-1"))
		if err != nil {
			t.Fatal(err)
		}
		if result.Accepted || result.Interrupted || result.Reason == "" {
			t.Fatalf("unexpected failed result: %#v", result)
		}
	})

	t.Run("interrupted", func(t *testing.T) {
		h, err := NewExecutionOnlyHarness(&executionOnlyExecutorStub{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionInterrupted}})
		if err != nil {
			t.Fatal(err)
		}
		result, err := h.Run(context.Background(), "project-1", request("TASK-1"))
		if err != nil {
			t.Fatal(err)
		}
		if !result.Interrupted || result.Accepted {
			t.Fatalf("unexpected interrupted result: %#v", result)
		}
	})

	t.Run("cancel error", func(t *testing.T) {
		want := errors.New("cancel failed")
		exec := &executionOnlyExecutorStub{cancelError: want}
		h, err := NewExecutionOnlyHarness(exec)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.Cancel("TASK-1"); !errors.Is(err, want) {
			t.Fatalf("cancel error=%v, want %v", err, want)
		}
		if exec.cancelID != "TASK-1" {
			t.Fatalf("cancel task=%s, want TASK-1", exec.cancelID)
		}
	})
}
