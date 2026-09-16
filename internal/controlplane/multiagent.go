package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"tally/internal/engineering"
	"tally/internal/multiagent"
	"tally/internal/taskexecution"
)

var ErrMultiAgentUnavailable = errors.New("controlplane: multi-agent workflow is not configured")

type MultiAgentBatchResult struct {
	Results  map[engineering.ID]DispatchResult `json:"results"`
	Errors   map[engineering.ID]string         `json:"errors,omitempty"`
	Snapshot ProjectSnapshot                   `json:"snapshot"`
}

func (s *Service) ConfigureMultiAgentTask(ctx context.Context, projectID, taskID engineering.ID, policy multiagent.TaskPolicy) (ProjectSnapshot, error) {
	policy = policy.Normalize()
	if err := policy.Validate(); err != nil {
		return ProjectSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.MultiAgent == nil {
		return ProjectSnapshot{}, ErrMultiAgentUnavailable
	}
	if s.multiActive[taskID] {
		return ProjectSnapshot{}, multiagent.ErrAlreadyActive
	}
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
	// A persisted running task may be owned by an in-flight collaboration from
	// another process. Do not mutate its role policy until it has been
	// interrupted/restarted or otherwise leaves running state.
	if node.State == engineering.TaskRunning {
		return ProjectSnapshot{}, multiagent.ErrAlreadyActive
	}
	ensureMultiAgentMaps(&snap)
	snap.MultiAgentPolicies[taskID] = policy
	if err := s.record(&snap, "multiagent.policy.configured", taskID, "multiagent:policy"); err != nil {
		return ProjectSnapshot{}, err
	}
	return snap, s.save(ctx, &snap)
}

func (s *Service) DispatchMultiAgent(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest) (DispatchResult, error) {
	if s.MultiAgent == nil {
		return DispatchResult{}, ErrMultiAgentUnavailable
	}
	if err := req.Validate(); err != nil {
		return DispatchResult{}, err
	}

	s.mu.Lock()
	_, policy, state, err := s.claimMultiAgentLocked(ctx, projectID, req)
	s.mu.Unlock()
	if err != nil {
		return DispatchResult{}, err
	}
	defer s.releaseMultiAgent(req.TaskID)
	return s.executeClaimedMultiAgent(ctx, projectID, req, policy, state)
}

func (s *Service) DispatchReadyMultiAgent(ctx context.Context, projectID engineering.ID, requests map[engineering.ID]taskexecution.TaskExecutionRequest) (MultiAgentBatchResult, error) {
	if s.MultiAgent == nil {
		return MultiAgentBatchResult{}, ErrMultiAgentUnavailable
	}
	if len(requests) == 0 {
		return MultiAgentBatchResult{}, errors.New("controlplane: multi-agent batch requires at least one request")
	}
	for id, req := range requests {
		if req.TaskID == "" {
			req.TaskID = id
			requests[id] = req
		}
		if req.TaskID != id {
			return MultiAgentBatchResult{}, fmt.Errorf("controlplane: request key %s does not match task %s", id, req.TaskID)
		}
		if err := req.Validate(); err != nil {
			return MultiAgentBatchResult{}, err
		}
	}

	s.mu.Lock()
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		s.mu.Unlock()
		return MultiAgentBatchResult{}, err
	}
	if snap.ActiveDAG == nil {
		s.mu.Unlock()
		return MultiAgentBatchResult{}, ErrDAGNotActive
	}
	ensureMultiAgentMaps(&snap)
	for id := range requests {
		if _, ok := snap.MultiAgentPolicies[id]; !ok {
			snap.MultiAgentPolicies[id] = multiagent.DefaultTaskPolicy()
		}
	}
	selected := multiagent.SelectReady(*snap.ActiveDAG, requests, snap.MultiAgentPolicies, s.MultiAgent.MaxConcurrency())
	if len(selected) == 0 {
		s.mu.Unlock()
		return MultiAgentBatchResult{}, ErrTaskNotReady
	}

	states := make(map[engineering.ID]multiagent.WorkflowState, len(selected))
	policies := make(map[engineering.ID]multiagent.TaskPolicy, len(selected))
	claimed := make([]engineering.ID, 0, len(selected))
	for _, id := range selected {
		if s.multiActive[id] {
			continue
		}
		state := snap.MultiAgentWorkflows[id]
		if state.TaskID == "" {
			state, err = multiagent.NewWorkflowState(id, s.now())
			if err != nil {
				s.mu.Unlock()
				return MultiAgentBatchResult{}, err
			}
		}
		dag, trErr := snap.ActiveDAG.Transition(id, engineering.TaskRunning, "")
		if trErr != nil {
			s.mu.Unlock()
			return MultiAgentBatchResult{}, trErr
		}
		snap.ActiveDAG = &dag
		snap.MultiAgentWorkflows[id] = state
		states[id] = state
		policies[id] = snap.MultiAgentPolicies[id]
		s.multiActive[id] = true
		claimed = append(claimed, id)
		if err := s.record(&snap, "multiagent.task.started", id, "multiagent:scheduler"); err != nil {
			for _, claimedID := range claimed {
				delete(s.multiActive, claimedID)
			}
			s.mu.Unlock()
			return MultiAgentBatchResult{}, err
		}
	}
	if len(claimed) == 0 {
		s.mu.Unlock()
		return MultiAgentBatchResult{}, multiagent.ErrAlreadyActive
	}
	if err := s.save(ctx, &snap); err != nil {
		for _, id := range claimed {
			delete(s.multiActive, id)
		}
		s.mu.Unlock()
		return MultiAgentBatchResult{}, err
	}
	s.mu.Unlock()

	out := MultiAgentBatchResult{Results: map[engineering.ID]DispatchResult{}, Errors: map[engineering.ID]string{}}
	var wg sync.WaitGroup
	var outMu sync.Mutex
	for _, id := range claimed {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer s.releaseMultiAgent(id)
			result, runErr := s.executeClaimedMultiAgent(ctx, projectID, requests[id], policies[id], states[id])
			outMu.Lock()
			out.Results[id] = result
			if runErr != nil {
				out.Errors[id] = runErr.Error()
			}
			outMu.Unlock()
		}()
	}
	wg.Wait()
	final, loadErr := s.Repo.Load(context.WithoutCancel(ctx), projectID)
	if loadErr != nil {
		return out, loadErr
	}
	out.Snapshot = final
	if len(out.Errors) == 0 {
		out.Errors = nil
	}
	return out, nil
}

func (s *Service) claimMultiAgentLocked(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest) (ProjectSnapshot, multiagent.TaskPolicy, multiagent.WorkflowState, error) {
	if s.multiActive[req.TaskID] {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, multiagent.ErrAlreadyActive
	}
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, err
	}
	if snap.ActiveDAG == nil {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, ErrDAGNotActive
	}
	node, ok := findNode(*snap.ActiveDAG, req.TaskID)
	if !ok {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, fmt.Errorf("controlplane: unknown task %s", req.TaskID)
	}
	ensureMultiAgentMaps(&snap)
	policy, ok := snap.MultiAgentPolicies[req.TaskID]
	if !ok {
		policy = multiagent.DefaultTaskPolicy()
		snap.MultiAgentPolicies[req.TaskID] = policy
	}
	state, hadState := snap.MultiAgentWorkflows[req.TaskID]

	source := "multiagent:dispatch"
	switch node.State {
	case engineering.TaskReady:
		if !hadState {
			state, err = multiagent.NewWorkflowState(req.TaskID, s.now())
			if err != nil {
				return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, err
			}
		}
		dag, trErr := snap.ActiveDAG.Transition(req.TaskID, engineering.TaskRunning, "")
		if trErr != nil {
			return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, trErr
		}
		snap.ActiveDAG = &dag
	case engineering.TaskRunning:
		// A new Service process has an empty multiActive set. Persisted running is
		// recoverable only when this task already owns durable multi-agent role
		// state. Never hijack a running task from the legacy Harness/Executor path.
		if !hadState {
			return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, fmt.Errorf("%w: running task %s has no persisted multi-agent workflow", ErrTaskNotReady, req.TaskID)
		}
		source = "multiagent:restart-recovery"
	default:
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, fmt.Errorf("%w: task %s state=%s", ErrTaskNotReady, req.TaskID, node.State)
	}
	if state.TaskID == "" {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, errors.New("controlplane: persisted multi-agent workflow is empty")
	}
	snap.MultiAgentWorkflows[req.TaskID] = state
	if err := s.record(&snap, "multiagent.task.started", req.TaskID, source); err != nil {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, err
	}
	if err := s.save(ctx, &snap); err != nil {
		return ProjectSnapshot{}, multiagent.TaskPolicy{}, multiagent.WorkflowState{}, err
	}
	s.multiActive[req.TaskID] = true
	return snap, policy, state, nil
}

func (s *Service) executeClaimedMultiAgent(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest, policy multiagent.TaskPolicy, state multiagent.WorkflowState) (DispatchResult, error) {
	result, runErr := s.MultiAgent.Run(ctx, req, policy, state, func(checkpointCtx context.Context, next multiagent.WorkflowState, event multiagent.Event) error {
		return s.persistMultiAgentCheckpoint(context.WithoutCancel(checkpointCtx), projectID, req.TaskID, next, event)
	})
	return s.finalizeMultiAgent(context.WithoutCancel(ctx), projectID, req.TaskID, result, runErr)
}

func (s *Service) persistMultiAgentCheckpoint(ctx context.Context, projectID, taskID engineering.ID, state multiagent.WorkflowState, event multiagent.Event) error {
	if err := state.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return err
	}
	ensureMultiAgentMaps(&snap)
	snap.MultiAgentWorkflows[taskID] = state
	source := "multiagent"
	if event.Role != "" {
		source += ":" + string(event.Role)
	}
	if strings.TrimSpace(event.RunID) != "" {
		source += ":" + event.RunID
	}
	kind := "multiagent." + strings.TrimPrefix(event.Kind, "multiagent.")
	if err := s.record(&snap, kind, taskID, source); err != nil {
		return err
	}
	return s.save(ctx, &snap)
}

func (s *Service) finalizeMultiAgent(ctx context.Context, projectID, taskID engineering.ID, result multiagent.Result, runErr error) (DispatchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, err := s.Repo.Load(ctx, projectID)
	if err != nil {
		return DispatchResult{}, err
	}
	if snap.ActiveDAG == nil {
		return DispatchResult{}, ErrDAGNotActive
	}
	ensureMultiAgentMaps(&snap)
	if result.State.TaskID != "" {
		// Cancellation can happen while the Engine is still queued on its global
		// concurrency semaphore, before any role checkpoint exists. Persist an
		// interrupted collaboration state so TaskDAG and role state agree.
		if errors.Is(runErr, context.Canceled) && result.State.Status == multiagent.WorkflowActive {
			result.State.Status = multiagent.WorkflowInterrupted
			result.State.Reason = "collaboration cancelled before the next role could run"
			result.State.Revision++
			result.State.UpdatedAt = s.now()
		}
		snap.MultiAgentWorkflows[taskID] = result.State
	}
	if _, ok := snap.Contracts[taskID]; !ok {
		return DispatchResult{}, fmt.Errorf("controlplane: task %s has no CompletionContract", taskID)
	}
	wf := TaskWorkflowResult{Accepted: result.Verified, Reason: result.Reason, Facts: result.Facts}
	base := DispatchResult{Execution: result.CandidateExecution, Workflow: &wf, Collaboration: &result.State, Snapshot: snap}

	if result.State.Status == multiagent.WorkflowInterrupted || errors.Is(runErr, context.Canceled) {
		node, found := findNode(*snap.ActiveDAG, taskID)
		if !found {
			return DispatchResult{}, fmt.Errorf("controlplane: unknown task %s", taskID)
		}
		switch node.State {
		case engineering.TaskRunning:
			next, trErr := snap.ActiveDAG.Transition(taskID, engineering.TaskInterrupted, "")
			if trErr != nil {
				return DispatchResult{}, trErr
			}
			snap.ActiveDAG = &next
		case engineering.TaskInterrupted, engineering.TaskReady:
			// RestartTask may already have cancelled the Engine and completed
			// Running -> Interrupted -> Ready while this run was unwinding.
			// Preserve that authoritative TaskDAG state rather than applying a
			// stale second transition.
		default:
			return DispatchResult{}, fmt.Errorf("controlplane: interrupted collaboration cannot finalize task %s from state %s", taskID, node.State)
		}
		_ = s.record(&snap, "multiagent.task.interrupted", taskID, "multiagent")
		if err := s.save(ctx, &snap); err != nil {
			return DispatchResult{}, err
		}
		base.Collaboration = &result.State
		base.Snapshot = snap
		return base, runErr
	}
	if !result.Verified {
		reason := strings.TrimSpace(result.Reason)
		if reason == "" && runErr != nil {
			reason = runErr.Error()
		}
		if reason == "" {
			reason = "multi-agent workflow did not verify candidate"
		}
		next, trErr := snap.ActiveDAG.Transition(taskID, engineering.TaskBlocked, reason)
		if trErr != nil {
			return DispatchResult{}, trErr
		}
		snap.ActiveDAG = &next
		_ = s.record(&snap, "multiagent.task.blocked", taskID, "multiagent")
		if err := s.save(ctx, &snap); err != nil {
			return DispatchResult{}, err
		}
		base.Snapshot = snap
		return base, runErr
	}

	// finishAccepted reloads the authoritative snapshot, so the terminal
	// collaboration state must be durable before the shared completion path runs.
	if err := s.save(ctx, &snap); err != nil {
		return DispatchResult{}, err
	}
	accepted, finishErr := s.finishAcceptedLocked(ctx, projectID, taskID, result.CandidateExecution, &wf, "multiagent")
	accepted.Collaboration = &result.State
	if finishErr != nil {
		return accepted, finishErr
	}
	if accepted.Completion != nil && accepted.Completion.Satisfied {
		state := accepted.Snapshot.MultiAgentWorkflows[taskID]
		state.Status = multiagent.WorkflowAccepted
		state.Reason = ""
		state.Revision++
		state.UpdatedAt = s.now()
		accepted.Snapshot.MultiAgentWorkflows[taskID] = state
		if err := s.record(&accepted.Snapshot, "multiagent.workflow.accepted", taskID, "completion_contract"); err != nil {
			return DispatchResult{}, err
		}
		if err := s.save(ctx, &accepted.Snapshot); err != nil {
			return DispatchResult{}, err
		}
		accepted.Collaboration = &state
	}
	return accepted, runErr
}

func (s *Service) releaseMultiAgent(taskID engineering.ID) {
	s.mu.Lock()
	delete(s.multiActive, taskID)
	s.mu.Unlock()
}

func ensureMultiAgentMaps(snap *ProjectSnapshot) {
	if snap.MultiAgentPolicies == nil {
		snap.MultiAgentPolicies = map[engineering.ID]multiagent.TaskPolicy{}
	}
	if snap.MultiAgentWorkflows == nil {
		snap.MultiAgentWorkflows = map[engineering.ID]multiagent.WorkflowState{}
	}
}
