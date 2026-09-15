package controlplane

import (
	"context"
	"errors"
	"fmt"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

const verificationUnavailableReason = "independent verification/review workflow unavailable"

// ExecutionOnlyHarness is the database-independent v1.5.1 production bridge
// from the Engineering Harness port to the frozen v1.5.0 AgentExecutor. It is
// intentionally fail-closed: Agent completion is only an implementation claim.
// Until an independent verifier/reviewer is composed, a completed claim is
// returned as non-accepted and Service.Dispatch persists BLOCKED rather than
// manufacturing CompletionFacts or promoting the task to DONE.
type ExecutionOnlyHarness struct {
	executor taskexecution.AgentExecutor
}

func NewExecutionOnlyHarness(executor taskexecution.AgentExecutor) (*ExecutionOnlyHarness, error) {
	if executor == nil {
		return nil, errors.New("controlplane: execution-only harness requires AgentExecutor")
	}
	return &ExecutionOnlyHarness{executor: executor}, nil
}

func (h *ExecutionOnlyHarness) Run(ctx context.Context, _ engineering.ID, req taskexecution.TaskExecutionRequest) (HarnessResult, error) {
	execution, err := h.executor.Execute(ctx, req)
	result := HarnessResult{Execution: execution}
	if err != nil {
		if isInterruptionError(err) || execution.Status == taskexecution.ExecutionInterrupted {
			result.Interrupted = true
		}
		return result, err
	}

	switch execution.Status {
	case taskexecution.ExecutionInterrupted:
		result.Interrupted = true
		return result, nil
	case taskexecution.ExecutionFailed:
		result.Reason = "agent runtime reported failure; no verification fact produced"
		return result, nil
	case taskexecution.ExecutionCompleted:
		result.Reason = verificationUnavailableReason
		return result, nil
	default:
		return result, fmt.Errorf("controlplane: unknown execution status %q", execution.Status)
	}
}

func (h *ExecutionOnlyHarness) Cancel(taskID engineering.ID) error {
	return h.executor.Cancel(taskID)
}
