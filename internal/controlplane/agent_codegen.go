package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/core"
	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type agentExecutorCodegen struct {
	executor taskexecution.AgentExecutor
	template taskexecution.TaskExecutionRequest
	project  core.ProjectID
	runtime  string
	now      func() time.Time

	mu   sync.Mutex
	last taskexecution.TaskExecutionResult
}

var _ closedloop.CodegenRunner = (*agentExecutorCodegen)(nil)

func (r *agentExecutorCodegen) RunCodegen(ctx context.Context, call closedloop.CodegenRunRequest) (*changeset.ChangeSet, error) {
	req := r.template
	req.TaskID = engineering.ID(call.TaskID)
	req.Runtime = taskexecution.RuntimeRef{ID: r.runtime}
	target := strings.TrimSpace(req.Environment.Target)
	if target == "" {
		target = "local"
	}
	req.Environment = taskexecution.EnvironmentRef{Kind: taskexecution.EnvironmentLocal, Target: target}
	req.Workspace = taskexecution.WorkspaceRef{
		Environment:  taskexecution.EnvironmentLocal,
		Target:       target,
		Identity:     string(call.RunID),
		Path:         call.WorkspacePath,
		BaseRevision: call.BaseRevision,
	}
	// Each closed-loop attempt is a fresh task-level execution. Correctness is
	// independent of Agent session resume; the prior candidate is already
	// materialized in the new disposable worktree by WorkspaceExecutor.
	req.Policy.ResumeSession = ""
	req.Policy.FallbackToFresh = true
	instruction, err := repairInstruction(req.Instruction, call.Feedback)
	if err != nil {
		return nil, err
	}
	req.Instruction = instruction

	execution, err := r.executor.Execute(ctx, req)
	r.setLast(execution)
	if err != nil {
		return nil, err
	}
	switch execution.Status {
	case taskexecution.ExecutionInterrupted:
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("controlplane: AgentExecutor reported interruption without caller cancellation")
	case taskexecution.ExecutionFailed:
		// A failed Agent claim creates no immutable candidate. The brownfield
		// closed loop deterministically escalates no_candidate; no verdict is
		// fabricated and repair budget is not consumed.
		return nil, nil
	case taskexecution.ExecutionCompleted:
	default:
		return nil, fmt.Errorf("controlplane: unknown task execution status %q", execution.Status)
	}

	candidate, ok, err := changeset.FreezeWorktree(ctx, changeset.WorktreeFreezeSpec{
		Project:       r.project,
		BaseRevision:  call.BaseRevision,
		ProducerRun:   string(call.RunID), // recorder rebinds this to the durable DB run id
		WorkspacePath: call.WorkspacePath,
		CreatedAt:     r.now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	execution.CandidateRef = string(candidate.ID())
	r.setLast(execution)
	return candidate, nil
}

func (r *agentExecutorCodegen) LastExecution() taskexecution.TaskExecutionResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.last
}

func (r *agentExecutorCodegen) setLast(result taskexecution.TaskExecutionResult) {
	r.mu.Lock()
	r.last = result
	r.mu.Unlock()
}

func repairInstruction(base string, feedback *closedloop.RepairFeedback) (string, error) {
	if feedback == nil {
		return base, nil
	}
	raw, err := json.Marshal(feedback)
	if err != nil {
		return "", fmt.Errorf("controlplane: encode safe repair feedback: %w", err)
	}
	return base + "\n\nTally repair feedback (safe structured projection):\n" + string(raw), nil
}
