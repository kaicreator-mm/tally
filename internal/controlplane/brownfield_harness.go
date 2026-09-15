package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/core"
	"tally/internal/engineering"
	"tally/internal/evidence"
	"tally/internal/outcome"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskexecution"
	"tally/internal/workspace"
)

var (
	ErrInvalidBrownfieldHarness = errors.New("controlplane: invalid brownfield Engineering Harness configuration")
	ErrExecutionAuthorityDrift  = errors.New("controlplane: execution request does not match pinned Git authority")
	ErrHeldOutUnavailable       = errors.New("controlplane: held-out verification prerequisites are unavailable")
)

type BrownfieldHarnessConfig struct {
	Repo         string
	WorktreeRoot string
	HoldoutRoot  string
	EnvName      string
	Runtime      agentruntime.RuntimeDescriptor
	Executor     taskexecution.AgentExecutor
	Queries      *store.Queries
	MaxRepair    int
	Now          func() time.Time
}

// BrownfieldHarness is the v1.5 production bridge over the existing v1.4
// authorities. It does not implement a second closed loop: one Run loads and
// compiles the pinned Git execution bundle, delegates attempts to AgentExecutor
// inside closedloop-owned disposable worktrees, freezes immutable ChangeSets,
// and reuses closedloop.NewLoop + candidateverify for verify/repair/accept.
type BrownfieldHarness struct {
	repo         string
	worktreeRoot string
	holdoutRoot  string
	envName      string
	runtime      agentruntime.RuntimeDescriptor
	executor     taskexecution.AgentExecutor
	queries      *store.Queries
	maxRepair    int
	now          func() time.Time
}

func NewBrownfieldHarness(cfg BrownfieldHarnessConfig) (*BrownfieldHarness, error) {
	switch {
	case strings.TrimSpace(cfg.Repo) == "":
		return nil, fmt.Errorf("%w: content repo is required", ErrInvalidBrownfieldHarness)
	case strings.TrimSpace(cfg.WorktreeRoot) == "":
		return nil, fmt.Errorf("%w: disposable worktree root is required", ErrInvalidBrownfieldHarness)
	case strings.TrimSpace(cfg.Runtime.Name) == "":
		return nil, fmt.Errorf("%w: runtime descriptor is required", ErrInvalidBrownfieldHarness)
	case cfg.Executor == nil:
		return nil, fmt.Errorf("%w: AgentExecutor is required", ErrInvalidBrownfieldHarness)
	case cfg.Queries == nil:
		return nil, fmt.Errorf("%w: fact store queries are required", ErrInvalidBrownfieldHarness)
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &BrownfieldHarness{
		repo:         cfg.Repo,
		worktreeRoot: cfg.WorktreeRoot,
		holdoutRoot:  cfg.HoldoutRoot,
		envName:      cfg.EnvName,
		runtime:      cfg.Runtime,
		executor:     cfg.Executor,
		queries:      cfg.Queries,
		maxRepair:    cfg.MaxRepair,
		now:          now,
	}, nil
}

func (h *BrownfieldHarness) Cancel(id engineering.ID) error {
	return h.executor.Cancel(id)
}

func (h *BrownfieldHarness) Run(ctx context.Context, projectID engineering.ID, req taskexecution.TaskExecutionRequest) (HarnessResult, error) {
	if err := req.Validate(); err != nil {
		return HarnessResult{}, err
	}
	if req.Environment.Kind != taskexecution.EnvironmentLocal || req.Workspace.Environment != taskexecution.EnvironmentLocal {
		return HarnessResult{}, fmt.Errorf("%w: v1.5.0 production bridge is Local-only", ErrInvalidBrownfieldHarness)
	}
	if req.FrozenTaskContract.Kind != engineering.ArtifactTaskContract {
		return HarnessResult{}, fmt.Errorf("%w: frozen task artifact kind=%q", ErrExecutionAuthorityDrift, req.FrozenTaskContract.Kind)
	}
	for _, a := range req.EngineeringContext {
		if err := a.Validate(); err != nil {
			return HarnessResult{}, fmt.Errorf("controlplane: invalid engineering context artifact: %w", err)
		}
	}

	// Load one pinned execution bundle before opening any runtime session or
	// disposable workspace. A moving branch expression is resolved once by the
	// loader; all authority below uses bundle.SourceCommit.
	bundle, err := artifacts.LoadBundle(h.repo, req.Workspace.BaseRevision, string(req.TaskID))
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: load execution bundle: %w", err)
	}
	if bundle.ProjectID != string(projectID) {
		return HarnessResult{}, fmt.Errorf("%w: project request=%s bundle=%s", ErrExecutionAuthorityDrift, projectID, bundle.ProjectID)
	}
	if rev := strings.TrimSpace(req.FrozenTaskContract.Revision); rev != "" && rev != bundle.Task.Contract.RevisionHash() {
		return HarnessResult{}, fmt.Errorf("%w: task revision request=%s bundle=%s", ErrExecutionAuthorityDrift, rev, bundle.Task.Contract.RevisionHash())
	}
	authority, err := bundle.CompileAuthority()
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: compile execution authority: %w", err)
	}

	frozen := bundle.Verification.Frozen
	if candidateverify.ContractHasHeldOut(frozen) {
		if strings.TrimSpace(h.holdoutRoot) == "" {
			return HarnessResult{}, fmt.Errorf("%w: TALLY_V15_HOLDOUT_ROOT is required", ErrHeldOutUnavailable)
		}
		if !h.runtime.Capabilities.Has(agentruntime.CapSessionIsolation) {
			return HarnessResult{}, fmt.Errorf("%w: runtime %s lacks session.isolation", ErrHeldOutUnavailable, h.runtime.Name)
		}
	}

	manager, err := workspace.New(h.repo, h.worktreeRoot)
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: create disposable workspace manager: %w", err)
	}
	worktrees := workspace.NewRunTable(manager)

	verify := candidateverify.New(candidateverify.Config{
		Repo:         h.repo,
		WorktreeRoot: h.worktreeRoot,
		HoldoutRoot:  h.holdoutRoot,
		EnvName:      h.envName,
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-controlplane/v1.5"},
	}, candidateverify.Deps{Queries: h.queries})

	runs := runstore.New(h.queries)
	lineage, err := closedloop.NewStoreLineageSink(runs)
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: wire supersession lineage: %w", err)
	}
	exposure, err := closedloop.NewStoreExposureSink(runs, frozen.Revision())
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: wire exposure facts: %w", err)
	}
	accept, err := closedloop.NewStoreAcceptSink(outcome.New(h.queries))
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: wire accepted outcome: %w", err)
	}
	seeds, err := runs.ExposuresForTask(ctx, core.ProjectID(bundle.ProjectID), string(req.TaskID))
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: load exposure facts: %w", err)
	}

	repair := closedloop.DefaultRepairPolicy()
	if h.maxRepair != 0 {
		repair, err = closedloop.NewRepairPolicy(h.maxRepair)
		if err != nil {
			return HarnessResult{}, err
		}
	}

	codegen := &agentExecutorCodegen{
		executor: h.executor,
		template: req,
		project:  core.ProjectID(bundle.ProjectID),
		runtime:  h.runtime.Name,
		now:      h.now,
	}
	loop, err := closedloop.NewLoop(closedloop.LoopConfig{
		Codegen:  codegen,
		Verify:   verify,
		Contract: frozen,
		Authority: authority.Compiled,
		Runtime:   h.runtime.Name,
		Worktrees: worktrees,
		BaseRevision: func(context.Context, string) (string, error) {
			return bundle.SourceCommit, nil
		},
		Repair:     repair,
		Lineage:    lineage,
		Exposure:   exposure,
		Exposures:  seeds,
		AcceptSink: accept,
		Now:        h.now,
	})
	if err != nil {
		return HarnessResult{}, fmt.Errorf("controlplane: compose brownfield closed loop: %w", err)
	}

	result, err := loop.Run(ctx, string(req.TaskID))
	last := codegen.LastExecution()
	if err != nil {
		return HarnessResult{Execution: last}, err
	}
	switch result.Phase {
	case closedloop.PhaseAccepted:
		if result.Basis == nil {
			return HarnessResult{Execution: last}, errors.New("controlplane: accepted closed loop missing basis")
		}
		return HarnessResult{
			Execution: last,
			Accepted:  true,
			Facts:     acceptedCompletionFacts(req, frozen.Hash(), *result.Basis, h.now().UTC()),
		}, nil
	case closedloop.PhaseCancelled:
		return HarnessResult{Execution: last, Interrupted: true, Reason: "closed loop cancelled"}, nil
	case closedloop.PhaseEscalated:
		reason := "closed loop escalated"
		if result.Escalation != nil {
			reason = fmt.Sprintf("closed loop escalated: %s", result.Escalation.Reason)
			if strings.TrimSpace(result.Escalation.Detail) != "" {
				reason += " (" + result.Escalation.Detail + ")"
			}
		}
		return HarnessResult{Execution: last, Reason: reason}, nil
	default:
		return HarnessResult{Execution: last}, fmt.Errorf("controlplane: closed loop returned non-terminal phase %q", result.Phase)
	}
}

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
		ProducerRun:   string(call.RunID),
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

func VerificationArtifactID(task engineering.ID) engineering.ID { return engineering.ID(string(task) + ":verification-artifact") }
func ReviewArtifactID(task engineering.ID) engineering.ID       { return engineering.ID(string(task) + ":review-artifact") }
func AcceptanceArtifactID(task engineering.ID) engineering.ID   { return engineering.ID(string(task) + ":acceptance-artifact") }
func VerificationEvidenceID(task engineering.ID) engineering.ID { return engineering.ID(string(task) + ":verification-evidence") }
func ClosedLoopCheckID(task engineering.ID) engineering.ID      { return engineering.ID(string(task) + ":closedloop-check") }
func AcceptanceRecordID(task engineering.ID) engineering.ID     { return engineering.ID(string(task) + ":acceptance-record") }

func acceptedCompletionFacts(req taskexecution.TaskExecutionRequest, verificationRevision string, basis closedloop.Basis, at time.Time) engineering.CompletionFacts {
	task := req.TaskID
	artifactsOut := make([]engineering.ArtifactRef, 0, len(req.EngineeringContext)+4)
	artifactsOut = append(artifactsOut, req.FrozenTaskContract)
	artifactsOut = append(artifactsOut, req.EngineeringContext...)
	verifyURI := "tally://verification-run/" + basis.VerifyRun
	candidateURI := "tally://changeset/" + string(basis.ChangeSet)
	artifactsOut = append(artifactsOut,
		engineering.ArtifactRef{ID: VerificationArtifactID(task), Kind: engineering.ArtifactVerification, URI: verifyURI, Revision: verificationRevision},
		engineering.ArtifactRef{ID: ReviewArtifactID(task), Kind: engineering.ArtifactReview, URI: candidateURI + "/review", Revision: verificationRevision},
		engineering.ArtifactRef{ID: AcceptanceArtifactID(task), Kind: engineering.ArtifactAcceptance, URI: candidateURI + "/accepted", Revision: verificationRevision},
	)
	return engineering.CompletionFacts{
		State:     engineering.NodeAccepted,
		Artifacts: artifactsOut,
		Evidence: []engineering.EvidenceRef{{
			ID: VerificationEvidenceID(task), Kind: "closedloop.final_verification", URI: verifyURI,
			Revision: verificationRevision, Status: engineering.EvidencePass,
		}},
		Checks: []engineering.CheckResult{{
			ID: ClosedLoopCheckID(task), Status: engineering.CheckPass,
			Basis: fmt.Sprintf("closedloop accepted immutable changeset %s via verification %s attempt %d", basis.ChangeSet, basis.VerifyRun, basis.Attempt),
		}},
		Records: []engineering.ActionRecord{{
			ID: AcceptanceRecordID(task), NodeID: task, Kind: "closedloop.accepted", Actor: "tally-closedloop", At: at,
			Provenance: engineering.Provenance{Actor: "tally-closedloop", Source: verifyURI, Revision: verificationRevision, At: at},
		}},
	}
}
