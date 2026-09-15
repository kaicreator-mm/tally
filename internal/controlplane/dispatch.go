package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

var ErrDispatchConflict = errors.New("controlplane: dispatch finalization conflicts with current task state")

// Dispatch claims one READY task under the project mutation lock, executes the
// Engineering Harness/Agent without holding that lock, then commits the result
// against the latest persisted snapshot. Long-running Agent work must never
// serialize otherwise independent project mutations.
func (s *Service) Dispatch(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest) (DispatchResult, error) {
	if s.Harness == nil && s.Executor == nil {
		return DispatchResult{}, errors.New("controlplane: Engineering Harness/AgentExecutor is not configured")
	}

	claimed, err := s.claimDispatch(ctx, projectID, req.TaskID)
	if err != nil {
		return DispatchResult{}, err
	}

	if s.Harness != nil {
		return s.dispatchHarness(ctx, projectID, req, claimed)
	}
	return s.dispatchLegacy(ctx, projectID, req, claimed)
}

func (s *Service) claimDispatch(ctx context.Context, projectID, taskID engineering.ID) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, ErrDAGNotActive
	}
	node, ok := findNode(*snap.ActiveDAG, taskID)
	if !ok {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: unknown task %s", taskID)
	}
	if node.State != engineering.TaskReady {
		return ProjectSnapshot{}, fmt.Errorf("%w: task %s state=%s", ErrTaskNotReady, taskID, node.State)
	}
	if _, ok := snap.Contracts[taskID]; !ok {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: task %s has no CompletionContract", taskID)
	}
	running, err := snap.ActiveDAG.Transition(taskID, engineering.TaskRunning, "")
	if err != nil {
		return ProjectSnapshot{}, err
	}
	snap.ActiveDAG = &running
	if err := s.record(&snap, "task.execution.started", taskID, "engineering_harness"); err != nil {
		return ProjectSnapshot{}, err
	}
	if err := s.save(ctx, &snap); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, nil
}

func (s *Service) dispatchHarness(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest, claimed ProjectSnapshot) (DispatchResult, error) {
	h, runErr := s.Harness.Run(ctx, projectID, req)
	wf := TaskWorkflowResult{Accepted: h.Accepted, Reason: h.Reason, Facts: h.Facts}
	base := DispatchResult{Execution: h.Execution, Workflow: &wf, Snapshot: claimed}

	if runErr != nil {
		if h.Interrupted || isInterruptionError(runErr) {
			return s.persistInterrupted(ctx, projectID, req.TaskID, h.Execution, &wf, runErr)
		}
		snap, commitErr := s.persistBlocked(ctx, projectID, req.TaskID, "engineering harness error: "+classifyError(runErr), "task.harness.error", "engineering_harness")
		if commitErr != nil {
			return base, errors.Join(runErr, commitErr)
		}
		base.Snapshot = snap
		return base, runErr
	}
	if h.Interrupted {
		return s.persistInterrupted(ctx, projectID, req.TaskID, h.Execution, &wf, nil)
	}
	if !h.Accepted {
		reason := strings.TrimSpace(h.Reason)
		if reason == "" {
			reason = "engineering harness did not accept candidate"
		}
		snap, err := s.persistBlocked(ctx, projectID, req.TaskID, reason, "task.harness.blocked", "closedloop")
		if err != nil {
			return base, err
		}
		base.Snapshot = snap
		return base, nil
	}
	return s.finishAccepted(ctx, projectID, req.TaskID, h.Execution, &wf, "closedloop")
}

func (s *Service) dispatchLegacy(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest, claimed ProjectSnapshot) (DispatchResult, error) {
	execution, execErr := s.Executor.Execute(ctx, req)
	base := DispatchResult{Execution: execution, Snapshot: claimed}
	if execErr != nil {
		if isInterruptionError(execErr) || execution.Status == taskexecution.ExecutionInterrupted {
			return s.persistInterrupted(ctx, projectID, req.TaskID, execution, nil, execErr)
		}
		snap, commitErr := s.persistBlocked(ctx, projectID, req.TaskID, "execution error: "+classifyError(execErr), "task.execution.error", "agent_executor")
		if commitErr != nil {
			return base, errors.Join(execErr, commitErr)
		}
		base.Snapshot = snap
		return base, execErr
	}

	switch execution.Status {
	case taskexecution.ExecutionInterrupted:
		return s.persistInterrupted(ctx, projectID, req.TaskID, execution, nil, nil)
	case taskexecution.ExecutionFailed:
		snap, err := s.persistBlocked(ctx, projectID, req.TaskID, "agent runtime reported failure; no verification fact produced", "task.execution.claim_failed", "agent_claim")
		if err != nil {
			return base, err
		}
		base.Snapshot = snap
		return base, nil
	case taskexecution.ExecutionCompleted:
		// Agent completion is only a claim. Legacy compatibility still requires
		// an independent workflow before CompletionContract evaluation.
	default:
		snap, commitErr := s.persistBlocked(ctx, projectID, req.TaskID, "unknown execution status", "task.execution.invalid_status", "agent_executor")
		if commitErr == nil {
			base.Snapshot = snap
		}
		err := fmt.Errorf("controlplane: unknown execution status %q", execution.Status)
		if commitErr != nil {
			err = errors.Join(err, commitErr)
		}
		return base, err
	}

	if s.Workflow == nil {
		snap, err := s.persistBlocked(ctx, projectID, req.TaskID, "verification/review workflow unavailable", "task.verification.unavailable", "verification")
		if err != nil {
			return base, err
		}
		base.Snapshot = snap
		return base, ErrWorkflowUnavailable
	}

	wf, wfErr := s.Workflow.VerifyAndAccept(ctx, req.TaskID, execution)
	base.Workflow = &wf
	if wfErr != nil {
		if isInterruptionError(wfErr) {
			return s.persistInterrupted(ctx, projectID, req.TaskID, execution, &wf, wfErr)
		}
		snap, commitErr := s.persistBlocked(ctx, projectID, req.TaskID, "verification workflow error: "+classifyError(wfErr), "task.verification.error", "verification")
		if commitErr != nil {
			return base, errors.Join(wfErr, commitErr)
		}
		base.Snapshot = snap
		return base, wfErr
	}
	if !wf.Accepted {
		reason := strings.TrimSpace(wf.Reason)
		if reason == "" {
			reason = "verification/review did not accept candidate"
		}
		snap, err := s.persistBlocked(ctx, projectID, req.TaskID, reason, "task.verification.rejected", "verification")
		if err != nil {
			return base, err
		}
		base.Snapshot = snap
		return base, nil
	}
	return s.finishAccepted(ctx, projectID, req.TaskID, execution, &wf, "verification")
}

func (s *Service) persistInterrupted(ctx context.Context, projectID, taskID engineering.ID, execution taskexecution.TaskExecutionResult, wf *TaskWorkflowResult, retErr error) (DispatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	persistCtx := context.WithoutCancel(ctx)
	snap, err := s.loadRunningForCommit(persistCtx, projectID, taskID)
	if err != nil {
		if retErr != nil {
			return DispatchResult{Execution: execution, Workflow: wf}, errors.Join(retErr, err)
		}
		return DispatchResult{Execution: execution, Workflow: wf}, err
	}
	next, err := snap.ActiveDAG.Transition(taskID, engineering.TaskInterrupted, "")
	if err != nil {
		return DispatchResult{}, err
	}
	snap.ActiveDAG = &next
	if err := s.record(&snap, "task.execution.interrupted", taskID, "engineering_harness"); err != nil {
		return DispatchResult{}, err
	}
	if err := s.save(persistCtx, &snap); err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Execution: execution, Workflow: wf, Snapshot: snap}, retErr
}

func (s *Service) persistBlocked(ctx context.Context, projectID, taskID engineering.ID, reason, kind, source string) (ProjectSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	persistCtx := context.WithoutCancel(ctx)
	snap, err := s.loadRunningForCommit(persistCtx, projectID, taskID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	next, err := snap.ActiveDAG.Transition(taskID, engineering.TaskBlocked, reason)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	snap.ActiveDAG = &next
	if err := s.record(&snap, kind, taskID, source); err != nil {
		return ProjectSnapshot{}, err
	}
	if err := s.save(persistCtx, &snap); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, nil
}

func (s *Service) finishAccepted(ctx context.Context, projectID, taskID engineering.ID, execution taskexecution.TaskExecutionResult, wf *TaskWorkflowResult, source string) (DispatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	persistCtx := context.WithoutCancel(ctx)
	snap, err := s.loadRunningForCommit(persistCtx, projectID, taskID)
	if err != nil {
		return DispatchResult{Execution: execution, Workflow: wf}, err
	}
	contract, ok := snap.Contracts[taskID]
	if !ok {
		return DispatchResult{Execution: execution, Workflow: wf, Snapshot: snap}, fmt.Errorf("%w: task %s CompletionContract changed or disappeared", ErrDispatchConflict, taskID)
	}

	// Human authority is project fact owned by the control plane. Decisions are
	// read from the latest snapshot so authority recorded while the Agent runs is
	// neither lost nor overwritten by stale dispatch state.
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
		if err := s.save(persistCtx, &snap); err != nil {
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
	if err := s.save(persistCtx, &snap); err != nil {
		return DispatchResult{}, err
	}
	return DispatchResult{Execution: execution, Workflow: wf, Completion: &completion, Snapshot: snap}, nil
}

func (s *Service) loadRunningForCommit(ctx context.Context, projectID, taskID engineering.ID) (ProjectSnapshot, error) {
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, fmt.Errorf("%w: active DAG disappeared while task %s was executing", ErrDispatchConflict, taskID)
	}
	node, ok := findNode(*snap.ActiveDAG, taskID)
	if !ok {
		return ProjectSnapshot{}, fmt.Errorf("%w: task %s no longer exists in current DAG", ErrDispatchConflict, taskID)
	}
	if node.State != engineering.TaskRunning {
		return ProjectSnapshot{}, fmt.Errorf("%w: task %s state=%s, expected running", ErrDispatchConflict, taskID, node.State)
	}
	return snap, nil
}

func isInterruptionError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
