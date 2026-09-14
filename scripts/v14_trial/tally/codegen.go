// codegen.go implements the inner codegen execution seam for the Tally-arm
// trial: a faithful mirror of cmd/worker/v14.go's v14CodegenRunner (package
// main there, not importable; P2-10 harness precedent for the in-trial
// replication) driving the REAL CodexRuntime. One outer attempt = one fresh
// CodegenRun in the attempt's one-shot worktree; the inner iteration budget
// comes from ExecutionPolicy.Retry (Decision Lock 7: zero coupling with the
// outer RepairPolicy). The candidate fix point collects the worktree's actual
// changes (git status facts) — the agent's done is a Claim, never a fact.
//
// Trial-side additions (recording only, no behavior change): per-attempt wall
// time, observed runtime identity (IdentityReporter; never backfilled) and
// the agent's self-declared summary are captured for the raw result row.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/core"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/workflow"
)

type attemptObservation struct {
	attempt       int
	status        string
	failureReason string
	iterations    int
	agentClaim    string
	wallSeconds   float64
	observed      agentruntime.ObservedIdentity
	workspaceID   string
}

// trialCodegenRunner implements closedloop.CodegenRunner.
type trialCodegenRunner struct {
	project  core.ProjectID
	task     taskcontract.TaskContract
	policy   policy.ExecutionPolicy
	profile  string
	runtime  agentruntime.Runtime
	model    string
	behavior string
	verifRev string
	packHash string
	sink     workflow.RunSink
	// trial recording surface
	runsRoot string
	taskID   string

	mu   sync.Mutex
	recs map[string]*attemptObservation // workflow.RunID → observation
}

var _ closedloop.CodegenRunner = (*trialCodegenRunner)(nil)

func (r *trialCodegenRunner) record(runID string) *attemptObservation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recs[runID]
}

func (r *trialCodegenRunner) put(runID string, o *attemptObservation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.recs == nil {
		r.recs = map[string]*attemptObservation{}
	}
	r.recs[runID] = o
}

// RunCodegen implements closedloop.CodegenRunner (v14CodegenRunner contract).
func (r *trialCodegenRunner) RunCodegen(ctx context.Context, req closedloop.CodegenRunRequest) (*changeset.ChangeSet, error) {
	runID := workflow.RunID(req.RunID)
	started := time.Now()
	factCtx := context.WithoutCancel(ctx)

	desc := r.runtime.Descriptor()
	startReq := agentruntime.StartRequest{Workspace: req.WorkspacePath}
	if desc.Capabilities.Has(agentruntime.CapModelSelect) {
		startReq.ModelProfile = r.model
	}
	session, err := r.runtime.Start(ctx, startReq)
	if err != nil {
		return nil, fmt.Errorf("trial: %s 开启 runtime 会话: %w", req.RunID, err)
	}
	// Observed identity is an optional observation; never backfilled from the
	// request (Decision Lock 15).
	var observed agentruntime.ObservedIdentity
	if rep, ok := session.(agentruntime.IdentityReporter); ok {
		observed = rep.Identity()
	}

	capSnap, err := capabilitySnapshot(desc)
	if err != nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("trial: %s 观测能力快照: %w", req.RunID, err)
	}
	runRef, err := r.sink.BeginRun(factCtx, workflow.RunBegin{
		ID:                   runID,
		EscalatedFrom:        workflow.RunID(req.ParentRun),
		Project:              r.project,
		TaskID:               r.task.ID,
		TaskRevision:         r.task.Hash,
		Role:                 "coding",
		Profile:              r.profile,
		RuntimeName:          desc.Name,
		RuntimeVersion:       desc.Version,
		Model:                r.model,
		BehaviorRevision:     r.behavior,
		ObservedIdentity:     observed,
		BaseRevision:         req.BaseRevision,
		WorkspaceID:          req.WorkspacePath,
		CapabilitySnapshot:   capSnap,
		StartedAt:            time.Now(),
		VerificationRevision: r.verifRev,
		ContextPackHash:      r.packHash,
	})
	if err != nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("trial: %s 记录开跑: %w", req.RunID, err)
	}
	if strings.TrimSpace(runRef) == "" {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("trial: %s 的 sink 未返回持久化 Run 引用", req.RunID)
	}

	runCtx, cancelWall := context.WithTimeout(ctx, r.policy.Budgets.MaxWallTime)
	defer cancelWall()

	var (
		iterations int
		inTokens   int64
		outTokens  int64
		changeSet  *changeset.ChangeSet
		status     runstore.RunStatus
		why        string
		retErr     error
		lastClaim  agentruntime.AgentRunResult
	)

	for iterations < r.policy.Retry.MaxIterations {
		iterations++
		res, execErr := session.Execute(runCtx, agentruntime.ExecuteRequest{
			Instruction: trialInstruction(r.task, iterations, req.Feedback),
		})
		lastClaim = res
		inTokens += res.Usage.InputTokens
		outTokens += res.Usage.OutputTokens
		if pErr := r.sink.RecordProgress(factCtx, runID, workflow.RunProgress{
			Iterations:  int32(iterations),
			InputTokens: &inTokens, OutputTokens: &outTokens,
		}); pErr != nil {
			status, why, retErr = runstore.RunFailed, "record_progress_failed", pErr
			break
		}
		if ctx.Err() != nil {
			status, why, retErr = runstore.RunCancelled, "cancelled", ctx.Err()
			break
		}
		if runCtx.Err() != nil {
			status, why = runstore.RunFailed, "wall_time_budget_exhausted"
			break
		}
		if inTokens+outTokens > r.policy.Budgets.MaxTokens {
			status, why = runstore.RunFailed, "token_budget_exhausted"
			break
		}
		if execErr != nil {
			status, why, retErr = runstore.RunFailed, "runtime_error", execErr
			break
		}
		if res.Status != agentruntime.RunCompleted {
			continue
		}
		snap, ok, fixErr := fixCandidateFromWorktree(ctx, req.WorkspacePath)
		if fixErr != nil {
			status, why, retErr = runstore.RunFailed, "fix_candidate_failed", fixErr
			break
		}
		if !ok {
			continue
		}
		cs, csErr := changeset.New(changeset.Spec{
			Project:           r.project,
			BaseRevision:      req.BaseRevision,
			CandidateRevision: snap.revision,
			Files:             snap.files,
			ProducerRun:       runRef,
			CreatedAt:         time.Now(),
		})
		if csErr != nil {
			status, why, retErr = runstore.RunFailed, "changeset_failed", csErr
			break
		}
		if rErr := r.sink.RecordChangeSet(factCtx, runID, cs); rErr != nil {
			status, why, retErr = runstore.RunFailed, "record_changeset_failed", rErr
			break
		}
		changeSet = cs
		status = runstore.RunCompleted
		break
	}
	if status == "" {
		status, why = runstore.RunFailed, "iterations_exhausted"
	}

	if finErr := r.sink.FinishRun(factCtx, workflow.RunFinish{
		ID:              runID,
		Status:          status,
		Iterations:      int32(iterations),
		InputTokens:     &inTokens,
		OutputTokens:    &outTokens,
		ResultPatchHash: changeSetID(changeSet),
		AgentClaim:      lastClaim.Summary,
		FailureReason:   why,
		FinishedAt:      time.Now(),
	}); finErr != nil && retErr == nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("trial: %s 记录终态: %w", req.RunID, finErr)
	}
	_ = session.Close(context.WithoutCancel(ctx))

	wall := time.Since(started).Seconds()
	r.put(string(runID), &attemptObservation{
		status:        string(status),
		failureReason: why,
		iterations:    iterations,
		agentClaim:    lastClaim.Summary,
		wallSeconds:   wall,
		observed:      observed,
		workspaceID:   req.WorkspacePath,
	})
	if r.runsRoot != "" {
		note := fmt.Sprintf("run %s: status=%s iterations=%d wall=%.1fs claim=%q",
			string(runID), string(status), iterations, wall, truncateUTF8(lastClaim.Summary, 200))
		appendEvent(r.runsRoot, r.taskID, 0, evAgentAttempt, note)
	}
	if retErr != nil {
		return nil, retErr
	}
	return changeSet, nil
}

// candidateSnapshot is the worktree-collected candidate (minimal projection of
// the workflow kernel CandidateSnapshot; clFixCandidateFromWorktree contract).
type candidateSnapshot struct {
	revision string
	files    []changeset.ChangedFile
}

// fixCandidateFromWorktree collects the one-shot worktree's actual changes
// (git status -z machine format, no rename detection, all untracked files);
// changed files enter the snapshot with their full post-change contents; no
// changes = no fixable candidate.
func fixCandidateFromWorktree(ctx context.Context, ws string) (candidateSnapshot, bool, error) {
	local := runner.LocalRunner{}
	res, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", ws, "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all"},
		Env:  gitEnv(),
	})
	if err != nil {
		return candidateSnapshot{}, false, fmt.Errorf("git status %s: %w", ws, err)
	}
	if res.ExitCode != 0 {
		return candidateSnapshot{}, false, fmt.Errorf("git status %s: %s", ws, strings.TrimSpace(res.Stderr))
	}
	var files []changeset.ChangedFile
	for _, rec := range strings.Split(res.Stdout, "\x00") {
		if rec == "" {
			continue
		}
		if len(rec) < 4 || rec[2] != ' ' {
			return candidateSnapshot{}, false, fmt.Errorf("trial: 无法解析 git status 记录 %q", rec)
		}
		code, path := rec[:2], rec[3:]
		if strings.Contains(code, "D") {
			return candidateSnapshot{}, false, fmt.Errorf("trial: 候选包含删除改动 %s,拒绝", path)
		}
		content, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(filepath.Clean(path))))
		if err != nil {
			return candidateSnapshot{}, false, fmt.Errorf("trial: 读取候选文件 %s: %w", path, err)
		}
		files = append(files, changeset.ChangedFile{Path: path, Content: string(content)})
	}
	if len(files) == 0 {
		return candidateSnapshot{}, false, nil
	}
	rev, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", ws, "rev-parse", "HEAD"},
		Env:  gitEnv(),
	})
	if err != nil || rev.ExitCode != 0 {
		return candidateSnapshot{}, false, fmt.Errorf("trial: 解析 worktree HEAD: %s", strings.TrimSpace(rev.Stderr))
	}
	return candidateSnapshot{revision: strings.TrimSpace(rev.Stdout), files: files}, true, nil
}

func capabilitySnapshot(d agentruntime.RuntimeDescriptor) (json.RawMessage, error) {
	caps := d.Capabilities.Sorted()
	names := make([]string, 0, len(caps))
	for _, c := range caps {
		names = append(names, string(c))
	}
	data, err := json.Marshal(struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}{Name: d.Name, Version: d.Version, Capabilities: names})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func changeSetID(cs *changeset.ChangeSet) string {
	if cs == nil {
		return ""
	}
	return string(cs.ID())
}

// trialInstruction mirrors cmd/worker v14Instruction exactly: contract
// objective + acceptance clauses; repair attempts attach only the P2-04 safe
// projection (the sole failure-material channel); iteration > 1 without
// feedback notes the retry semantics. The instruction is input only — the
// output remains a proposal, judged by the frozen verification authority.
func trialInstruction(task taskcontract.TaskContract, iteration int, fb *closedloop.RepairFeedback) string {
	var b strings.Builder
	b.WriteString("Task " + task.ID)
	if task.Title != "" {
		b.WriteString(": " + task.Title)
	}
	b.WriteString("\n\nObjective:\n" + task.Objective + "\n")
	if len(task.Acceptance) > 0 {
		b.WriteString("\nAcceptance:\n")
		for _, a := range task.Acceptance {
			b.WriteString("- " + a.ID + ": " + a.Statement + "\n")
		}
	}
	if fb != nil {
		b.WriteString("\nRepair context (verification feedback):\n")
		b.WriteString("- failed attempt: " + fmt.Sprintf("%d", int(fb.Attempt)) +
			" (run " + string(fb.RunID) + ")\n")
		if fb.Subject.AcceptanceID != "" {
			b.WriteString("- failed acceptance " + fb.Subject.AcceptanceID + ": " + fb.Subject.Statement + "\n")
		}
		if fb.Subject.Gate != "" {
			b.WriteString("- failed required gate: " + fb.Subject.Gate + "\n")
		}
		b.WriteString("- failure class: " + string(fb.Failure) + "\n")
		b.WriteString("- diagnostic summary: " + fb.DiagnosticSummary + "\n")
		for _, ev := range fb.Evidence {
			b.WriteString("- visible evidence " + ev.BindingID)
			if ev.Path != "" {
				b.WriteString(" at " + ev.Path)
			}
			if ev.Selector != "" {
				b.WriteString(" selector " + ev.Selector)
			}
			b.WriteString(" status " + string(ev.Status) + "\n")
		}
		b.WriteString("\nThis is repair attempt " + fmt.Sprintf("%d", iteration) +
			"; the previous candidate is already materialized in this workspace. " +
			"Re-work the change so the failed acceptance passes under the same frozen contract.\n")
	} else if iteration > 1 {
		b.WriteString("\nThis is retry iteration " + fmt.Sprintf("%d", iteration) +
			"; the previous iteration did not produce an acceptable candidate. Re-work the change in the same workspace.\n")
	}
	return b.String()
}
