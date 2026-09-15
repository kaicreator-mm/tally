package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/core"
	"tally/internal/engineering"
	"tally/internal/measurement"
	"tally/internal/runstore"
	"tally/internal/taskexecution"
)

type runstoreCodegenRecorder struct {
	inner                *agentExecutorCodegen
	store                *runstore.Store
	project              core.ProjectID
	taskRevision         string
	verificationRevision string
	runtime              agentruntime.RuntimeDescriptor
	baseContextHash      string
	model                string
	now                  func() time.Time
}

var _ closedloop.CodegenRunner = (*runstoreCodegenRecorder)(nil)

func newRunstoreCodegenRecorder(inner *agentExecutorCodegen, store *runstore.Store, project core.ProjectID, req taskexecution.TaskExecutionRequest, taskRevision, verificationRevision string, runtime agentruntime.RuntimeDescriptor, now func() time.Time) (*runstoreCodegenRecorder, error) {
	if inner == nil || store == nil || project == "" || strings.TrimSpace(taskRevision) == "" || strings.TrimSpace(verificationRevision) == "" || strings.TrimSpace(runtime.Name) == "" || strings.TrimSpace(runtime.Version) == "" {
		return nil, errors.New("controlplane: incomplete CodegenRun recording configuration")
	}
	contextHash, err := taskExecutionContextHash(req)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(req.Policy.ModelProfile)
	if model == "" {
		// Expected-coordinate label only; observed model identity remains empty
		// unless the runtime actually reports it.
		model = "runtime-default"
	}
	return &runstoreCodegenRecorder{
		inner: inner, store: store, project: project,
		taskRevision: taskRevision, verificationRevision: verificationRevision,
		runtime: runtime, baseContextHash: contextHash, model: model, now: now,
	}, nil
}

func (r *runstoreCodegenRecorder) RunCodegen(ctx context.Context, call closedloop.CodegenRunRequest) (*changeset.ChangeSet, error) {
	capabilities, err := runtimeCapabilitySnapshot(r.runtime)
	if err != nil {
		return nil, err
	}
	contextHash, err := attemptContextHash(r.baseContextHash, call)
	if err != nil {
		return nil, err
	}
	runID, err := r.store.StartCodegenRun(ctx, runstore.StartRunSpec{
		Project:                   r.project,
		Task:                      call.TaskID,
		Role:                      "coding",
		Runtime:                   r.runtime.Name,
		RuntimeVersion:            r.runtime.Version,
		Model:                     r.model,
		BehaviorRevision:          "v1.5-controlplane",
		CostPlane:                 measurement.CostPlaneExecution,
		BaseCommit:                call.BaseRevision,
		TaskRevision:              r.taskRevision,
		VerificationRevision:      r.verificationRevision,
		ContextPackHash:           contextHash,
		WorkspaceID:               string(call.RunID),
		RuntimeCapabilitySnapshot: capabilities,
		StartedAt:                 r.now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("controlplane: record CodegenRun start: %w", err)
	}

	candidate, runErr := r.inner.RunCodegen(ctx, call)
	execution := r.inner.LastExecution()
	if runErr != nil {
		status := runstore.RunFailed
		if ctx.Err() != nil || execution.Status == taskexecution.ExecutionInterrupted {
			status = runstore.RunCancelled
		}
		finishErr := r.finish(ctx, runID, status, execution, nil, runErr.Error())
		return nil, errors.Join(runErr, finishErr)
	}

	if candidate == nil {
		status := runstore.RunCompleted
		reason := ""
		switch execution.Status {
		case taskexecution.ExecutionFailed:
			status = runstore.RunFailed
			reason = "agent runtime reported failure without immutable candidate"
		case taskexecution.ExecutionInterrupted:
			status = runstore.RunCancelled
			reason = "agent execution interrupted"
		}
		if err := r.finish(ctx, runID, status, execution, nil, reason); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// The inner runner freezes content immediately after Agent execution. Rebind
	// that exact immutable content to the durable CodegenRun primary key before
	// validation so runstore/verification foreign-key lineage is complete.
	durable, err := changeset.New(changeset.Spec{
		Project:           candidate.Project(),
		BaseRevision:      candidate.BaseRevision(),
		CandidateRevision: candidate.CandidateRevision(),
		Files:             candidate.Files(),
		Entities:          candidate.Entities(),
		ProducerRun:       strconv.FormatInt(runID, 10),
		CreatedAt:         candidate.CreatedAt(),
	})
	if err != nil {
		finishErr := r.finish(ctx, runID, runstore.RunFailed, execution, nil, "failed to bind candidate to durable CodegenRun")
		return nil, errors.Join(err, finishErr)
	}
	if err := r.store.SaveChangeSet(ctx, durable); err != nil {
		finishErr := r.finish(ctx, runID, runstore.RunFailed, execution, nil, "failed to persist immutable ChangeSet")
		return nil, errors.Join(err, finishErr)
	}

	execution.CandidateRef = string(durable.ID())
	r.inner.setLast(execution)
	if err := r.finish(ctx, runID, runstore.RunCompleted, execution, durable, ""); err != nil {
		return nil, err
	}
	return durable, nil
}

func (r *runstoreCodegenRecorder) finish(ctx context.Context, runID int64, status runstore.RunStatus, execution taskexecution.TaskExecutionResult, candidate *changeset.ChangeSet, failure string) error {
	claim := ""
	if raw, err := json.Marshal(execution.AgentClaim); err == nil {
		claim = string(raw)
	} else {
		claim = execution.AgentClaim.Summary
	}
	patch := ""
	if candidate != nil {
		patch = candidate.PatchIdentity()
	}
	ok, err := r.store.FinishCodegenRun(context.WithoutCancel(ctx), runstore.FinishRunSpec{
		ID: runID, Status: status, FinishedAt: r.now().UTC(), Iterations: 1,
		ResultPatchHash: patch, AgentClaim: claim, FailureReason: failure,
	})
	if err != nil {
		return fmt.Errorf("controlplane: record CodegenRun terminal state: %w", err)
	}
	if !ok {
		return fmt.Errorf("controlplane: CodegenRun %d terminal transition did not apply", runID)
	}
	return nil
}

func runtimeCapabilitySnapshot(runtime agentruntime.RuntimeDescriptor) (json.RawMessage, error) {
	raw, err := json.Marshal(map[string]any{
		"adapter_version":       runtime.Version,
		"declared_capabilities": runtime.Capabilities.Sorted(),
	})
	if err != nil {
		return nil, fmt.Errorf("controlplane: encode runtime capability snapshot: %w", err)
	}
	return raw, nil
}

func taskExecutionContextHash(req taskexecution.TaskExecutionRequest) (string, error) {
	projection := struct {
		Task    engineering.ArtifactRef   `json:"task"`
		Context []engineering.ArtifactRef `json:"context,omitempty"`
		Prompt  string                    `json:"prompt"`
	}{Task: req.FrozenTaskContract, Context: req.EngineeringContext, Prompt: req.Instruction}
	return hashJSON(projection)
}

func attemptContextHash(base string, call closedloop.CodegenRunRequest) (string, error) {
	repairBase := ""
	if call.RepairBase != nil {
		repairBase = string(call.RepairBase.ID())
	}
	projection := struct {
		Base       string                     `json:"base"`
		ParentRun  closedloop.RunID           `json:"parent_run,omitempty"`
		RepairBase string                     `json:"repair_base,omitempty"`
		Feedback   *closedloop.RepairFeedback `json:"feedback,omitempty"`
	}{Base: base, ParentRun: call.ParentRun, RepairBase: repairBase, Feedback: call.Feedback}
	return hashJSON(projection)
}

func hashJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("controlplane: encode execution context identity: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
