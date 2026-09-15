package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

func (s *Service) Dispatch(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest) (DispatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Harness == nil && s.Executor == nil {
		return DispatchResult{}, errors.New("controlplane: Engineering Harness/AgentExecutor is not configured")
	}
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return DispatchResult{}, err
	}
	if snap.ActiveDAG == nil {
		return DispatchResult{}, ErrDAGNotActive
	}
	node, ok := findNode(*snap.ActiveDAG, req.TaskID)
	if !ok {
		return DispatchResult{}, fmt.Errorf("controlplane: unknown task %s", req.TaskID)
	}
	if node.State != engineering.TaskReady {
		return DispatchResult{}, fmt.Errorf("%w: task %s state=%s", ErrTaskNotReady, req.TaskID, node.State)
	}
	contract, ok := snap.Contracts[req.TaskID]
	if !ok {
		return DispatchResult{}, fmt.Errorf("controlplane: task %s has no CompletionContract", req.TaskID)
	}
	running, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskRunning, "")
	if err != nil {
		return DispatchResult{}, err
	}
	snap.ActiveDAG = &running
	if err := s.record(&snap, "task.execution.started", req.TaskID, "engineering_harness"); err != nil {
		return DispatchResult{}, err
	}
	if err := s.save(ctx, &snap); err != nil {
		return DispatchResult{}, err
	}

	if s.Harness != nil {
		return s.dispatchHarness(ctx, projectID, req, contract, snap)
	}
	return s.dispatchLegacy(ctx, req, contract, snap)
}

func (s *Service) dispatchHarness(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest, contract engineering.CompletionContract, snap ProjectSnapshot) (DispatchResult, error) {
	h, runErr := s.Harness.Run(ctx, projectID, req)
	wf := TaskWorkflowResult{Accepted: h.Accepted, Reason: h.Reason, Facts: h.Facts}
	base := DispatchResult{Execution: h.Execution, Workflow: &wf, Snapshot: snap}

	if runErr != nil {
		if h.Interrupted || errors.Is(runErr, context.Canceled) {
			return s.persistInterrupted(ctx, req.TaskID, h.Execution, &wf, snap, runErr)
		}
		next, trErr := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, "engineering harness error: "+classifyError(runErr))
		if trErr == nil {
			snap.ActiveDAG = &next
			_ = s.record(&snap, "task.harness.error", req.TaskID, "engineering_harness")
			_ = s.save(context.WithoutCancel(ctx), &snap)
			base.Snapshot = snap
		}
		return base, runErr
	}
	if h.Interrupted {
		return s.persistInterrupted(ctx, req.TaskID, h.Execution, &wf, snap, nil)
	}
	if !h.Accepted {
		reason := strings.TrimSpace(h.Reason)
		if reason == "" {
			reason = "engineering harness did not accept candidate"
		}
		next, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, reason)
		if err != nil {
			return DispatchResult{}, err
		}
		snap.ActiveDAG = &next
		if err := s.record(&snap, "task.harness.blocked", req.TaskID, "closedloop"); err != nil {
			return DispatchResult{}, err
		}
		if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
			return DispatchResult{}, err
		}
		base.Snapshot = snap
		return base, nil
	}
	return s.finishAccepted(ctx, req.TaskID, h.Execution, &wf, contract, snap, "closedloop")
}

func (s *Service) dispatchLegacy(ctx context.Context, req taskexecution.TaskExecutionRequest, contract engineering.CompletionContract, snap ProjectSnapshot) (DispatchResult, error) {
	execution, execErr := s.Executor.Execute(ctx, req)
	if execErr != nil {
		blocked, trErr := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, "execution error: "+classifyError(execErr))
		if trErr == nil {
			snap.ActiveDAG = &blocked
			_ = s.record(&snap, "task.execution.error", req.TaskID, "agent_executor")
			_ = s.save(context.WithoutCancel(ctx), &snap)
		}
		return DispatchResult{Execution: execution, Snapshot: snap}, execErr
	}

	switch execution.Status {
	case taskexecution.ExecutionInterrupted:
		return s.persistInterrupted(ctx, req.TaskID, execution, nil, snap, nil)
	case taskexecution.ExecutionFailed:
		next, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, "agent runtime reported failure; no verification fact produced")
		if err != nil {
			return DispatchResult{}, err
		}
		snap.ActiveDAG = &next
		if err := s.record(&snap, "task.execution.claim_failed", req.TaskID, "agent_claim"); err != nil {
			return DispatchResult{}, err
		}
		if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
			return DispatchResult{}, err
		}
		return DispatchResult{Execution: execution, Snapshot: snap}, nil
	case taskexecution.ExecutionCompleted:
		// Agent completion is only a claim. Legacy compatibility still requires
		// an independent workflow before CompletionContract evaluation.
	default:
		next, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, "unknown execution status")
		if err == nil {
			snap.ActiveDAG = &next
			_ = s.record(&snap, "task.execution.invalid_status", req.TaskID, "agent_executor")
			_ = s.save(context.WithoutCancel(ctx), &snap)
		}
		return DispatchResult{Execution: execution, Snapshot: snap}, fmt.Errorf("controlplane: unknown execution status %q", execution.Status)
	}

	if s.Workflow == nil {
		next, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, "verification/review workflow unavailable")
		if err != nil {
			return DispatchResult{}, err
		}
		snap.ActiveDAG = &next
		if err := s.record(&snap, "task.verification.unavailable", req.TaskID, "verification"); err != nil {
			return DispatchResult{}, err
		}
		if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
			return DispatchResult{}, err
		}
		return DispatchResult{Execution: execution, Snapshot: snap}, ErrWorkflowUnavailable
	}

	wf, wfErr := s.Workflow.VerifyAndAccept(ctx, req.TaskID, execution)
	if wfErr != nil {
		next, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, "verification workflow error: "+classifyError(wfErr))
		if err == nil {
			snap.ActiveDAG = &next
			_ = s.record(&snap, "task.verification.error", req.TaskID, "verification")
			_ = s.save(context.WithoutCancel(ctx), &snap)
		}
		return DispatchResult{Execution: execution, Workflow: &wf, Snapshot: snap}, wfErr
	}
	if !wf.Accepted {
		reason := strings.TrimSpace(wf.Reason)
		if reason == "" {
			reason = "verification/review did not accept candidate"
		}
		next, err := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskBlocked, reason)
		if err != nil {
			return DispatchResult{}, err
		}
		snap.ActiveDAG = &next
		if err := s.record(&snap, "task.verification.rejected", req.TaskID, "verification"); err != nil {
			return DispatchResult{}, err
		}
		if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
			return DispatchResult{}, err
		}
		return DispatchResult{Execution: execution, Workflow: &wf, Snapshot: snap}, nil
	}
	return s.finishAccepted(ctx, req.TaskID, execution, &wf, contract, snap, "verification")
}

func (s *Service) persistInterrupted(ctx context.Context, taskID engineering.ID, execution taskexecution.TaskExecutionResult, wf *TaskWorkflowResult, snap ProjectSnapshot, retErr error) (DispatchResult, error) {
	next, err := snap.ActiveDAG.Transition(taskID, engineering.TaskInterrupted, "")
	if err != nil {
		return DispatchResult{}, err
	}
	snap.ActiveDAG = &next
	if err := s.record(&snap, "task.execution.interrupted", taskID, "engineering_harness"); err != nil {
		return DispatchResult{}, err
	}
	if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Execution: execution, Workflow: wf, Snapshot: snap}, retErr
}

func (s *Service) finishAccepted(ctx context.Context, taskID engineering.ID, execution taskexecution.TaskExecutionResult, wf *TaskWorkflowResult, contract engineering.CompletionContract, snap ProjectSnapshot, source string) (DispatchResult, error) {
	// Human authority is project fact owned by the control plane. Only decisions
	// scoped to this node join independently produced Harness/workflow facts.
	authority := append([]engineering.AuthorityDecision(nil), wf.Facts.Authority...)
	for _, record := range snap.AuthorityDecisions {
		if record.NodeID == taskID {
			authority = append(authority, record.Decision)
		}
	}
	wf.Facts.Authority = authority
	completion := contract.Evaluate(wf.Facts)
	if !completion.Satisfied {
		next, err := snap.ActiveDAG.Transition(taskID, engineering.TaskBlocked, "CompletionContract not satisfied")
		if err != nil {
			return DispatchResult{}, err
		}
		snap.ActiveDAG = &next
		if err := s.record(&snap, "task.completion.blocked", taskID, "completion_contract"); err != nil {
			return DispatchResult{}, err
		}
		if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
			return DispatchResult{}, err
		}
		return DispatchResult{Execution: execution, Workflow: wf, Completion: &completion, Snapshot: snap}, nil
	}

	accepted, err := snap.ActiveDAG.Transition(taskID, engineering.TaskAccepted, "")
	if err != nil {
		return DispatchResult{}, err
	}
	done, err := accepted.Transition(taskID, engineering.TaskDone, "")
	if err != nil {
		return DispatchResult{}, err
	}
	snap.ActiveDAG = &done
	if err := s.record(&snap, "task.accepted", taskID, source); err != nil {
		return DispatchResult{}, err
	}
	if err := s.record(&snap, "task.completed", taskID, "completion_contract"); err != nil {
		return DispatchResult{}, err
	}
	if err := s.save(context.WithoutCancel(ctx), &snap); err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Execution: execution, Workflow: wf, Completion: &completion, Snapshot: snap}, nil
}
