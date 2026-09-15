package multiagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/engineering"
	"tally/internal/evidence"
	"tally/internal/store"
	"tally/internal/taskexecution"
	"tally/internal/verification"
)

var (
	ErrNoFrozenCandidate          = errors.New("multiagent: role execution produced no immutable candidate")
	ErrReadOnlyRoleMutatedCandidate = errors.New("multiagent: review/verification role mutated the candidate workspace")
)

// VerifiedRoleExecutor decorates the frozen v1.5.0 AgentExecutor contract for
// v1.5.4. Implementer/Repairer runs must freeze a candidate. Reviewer,
// Verifier and Specialist runs are treated as read-only: the candidate is
// frozen before and after the Agent run and any drift is rejected. Their
// factual evidence is produced by the existing candidateverify authority, not
// by AgentRunResult.Summary.
type VerifiedRoleExecutor struct {
	delegate     taskexecution.AgentExecutor
	repo         string
	worktreeRoot string
	holdoutRoot  string
	envName      string
	runtime      agentruntime.RuntimeDescriptor
	queries      *store.Queries
	now          func() time.Time
}

type VerifiedRoleExecutorConfig struct {
	Delegate     taskexecution.AgentExecutor
	Repo         string
	WorktreeRoot string
	HoldoutRoot  string
	EnvName      string
	Runtime      agentruntime.RuntimeDescriptor
	Queries      *store.Queries
	Now          func() time.Time
}

func NewVerifiedRoleExecutor(cfg VerifiedRoleExecutorConfig) (*VerifiedRoleExecutor, error) {
	switch {
	case cfg.Delegate == nil:
		return nil, errors.New("multiagent: verified role executor requires AgentExecutor")
	case strings.TrimSpace(cfg.Repo) == "":
		return nil, errors.New("multiagent: verified role executor requires content repo")
	case strings.TrimSpace(cfg.WorktreeRoot) == "":
		return nil, errors.New("multiagent: verified role executor requires verification worktree root")
	case strings.TrimSpace(cfg.EnvName) == "":
		return nil, errors.New("multiagent: verified role executor requires environment name")
	case cfg.Queries == nil:
		return nil, errors.New("multiagent: verified role executor requires fact-store queries")
	}
	if err := cfg.Runtime.Validate(); err != nil {
		return nil, err
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &VerifiedRoleExecutor{
		delegate: cfg.Delegate, repo: cfg.Repo, worktreeRoot: cfg.WorktreeRoot,
		holdoutRoot: cfg.HoldoutRoot, envName: cfg.EnvName, runtime: cfg.Runtime,
		queries: cfg.Queries, now: cfg.Now,
	}, nil
}

func (e *VerifiedRoleExecutor) Cancel(id engineering.ID) error { return e.delegate.Cancel(id) }

func (e *VerifiedRoleExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	role, ok := roleFromInstruction(req.Instruction)
	if !ok {
		return taskexecution.TaskExecutionResult{}, errors.New("multiagent: verified role executor received request without a v1.5.4 role directive")
	}

	bundle, err := artifacts.LoadBundle(e.repo, req.Workspace.BaseRevision, string(req.TaskID))
	if err != nil {
		return taskexecution.TaskExecutionResult{}, fmt.Errorf("multiagent: load frozen task bundle: %w", err)
	}

	var before *changeset.ChangeSet
	if readOnlyRole(role) {
		before, err = e.freeze(ctx, req, core.ProjectID(bundle.ProjectID), "readonly:"+string(req.TaskID)+":"+string(role))
		if err != nil {
			return taskexecution.TaskExecutionResult{}, err
		}
	}

	result, execErr := e.delegate.Execute(ctx, req)
	if execErr != nil || result.Status != taskexecution.ExecutionCompleted {
		return result, execErr
	}

	if role == RoleImplementer || role == RoleRepairer {
		producer := strings.TrimSpace(string(result.AgentClaim.RunID))
		if producer == "" {
			producer = "role:" + string(req.TaskID) + ":" + string(role)
		}
		candidate, freezeErr := e.freeze(ctx, req, core.ProjectID(bundle.ProjectID), producer)
		if freezeErr != nil {
			return result, freezeErr
		}
		result.CandidateRef = string(candidate.ID())
		return result, nil
	}

	after, err := e.freeze(ctx, req, core.ProjectID(bundle.ProjectID), "readonly:"+string(req.TaskID)+":"+string(role))
	if err != nil {
		return result, err
	}
	if before.ID() != after.ID() {
		return result, fmt.Errorf("%w: role=%s before=%s after=%s", ErrReadOnlyRoleMutatedCandidate, role, before.ID(), after.ID())
	}
	result.CandidateRef = string(before.ID())

	if candidateverify.ContractHasHeldOut(bundle.Verification.Frozen) && !e.runtime.Capabilities.Has(agentruntime.CapSessionIsolation) {
		return result, fmt.Errorf("multiagent: held-out verification requires runtime session.isolation")
	}
	verify := candidateverify.New(candidateverify.Config{
		Repo: e.repo, WorktreeRoot: e.worktreeRoot, HoldoutRoot: e.holdoutRoot,
		EnvName: e.envName,
		Recorder: evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-controlplane/v1.5.4"},
	}, candidateverify.Deps{Queries: e.queries})
	outcome, err := verify.Verify(ctx, candidateverify.Request{
		Contract: bundle.Verification.Frozen,
		ChangeSet: before,
		Runtime: e.runtime.Name,
	})
	if err != nil {
		return result, fmt.Errorf("multiagent: role verification authority failed: %w", err)
	}
	result.EvidenceRefs = append(result.EvidenceRefs, finalOutcomeEvidence(req.TaskID, role, outcome))
	return result, nil
}

func (e *VerifiedRoleExecutor) freeze(ctx context.Context, req taskexecution.TaskExecutionRequest, project core.ProjectID, producer string) (*changeset.ChangeSet, error) {
	candidate, ok, err := changeset.FreezeWorktree(ctx, changeset.WorktreeFreezeSpec{
		Project: project, BaseRevision: req.Workspace.BaseRevision, ProducerRun: producer,
		WorkspacePath: req.Workspace.Path, CreatedAt: e.now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("multiagent: freeze candidate: %w", err)
	}
	if !ok || candidate == nil {
		return nil, ErrNoFrozenCandidate
	}
	return candidate, nil
}

func finalOutcomeEvidence(taskID engineering.ID, role Role, outcome *verification.FinalOutcome) engineering.EvidenceRef {
	status := engineering.EvidenceNotRun
	if outcome != nil {
		switch outcome.Status {
		case verification.FinalVerified:
			status = engineering.EvidencePass
		case verification.FinalFailed:
			status = engineering.EvidenceFail
		case verification.FinalNeedsReview:
			status = engineering.EvidenceNotRun
		}
	}
	runID := int64(0)
	changeSetID := ""
	if outcome != nil {
		runID = outcome.RunID
		changeSetID = string(outcome.ChangeSetID)
	}
	id := engineering.ID(fmt.Sprintf("%s:%s:verification:%d", taskID, role, runID))
	return engineering.EvidenceRef{
		ID: id,
		Kind: "verification.final",
		URI: fmt.Sprintf("tally://verification/run/%d", runID),
		Revision: changeSetID,
		Status: status,
	}
}

func readOnlyRole(role Role) bool {
	return role == RoleReviewer || role == RoleVerifier || role == RoleSpecialist
}

func roleFromInstruction(instruction string) (Role, bool) {
	markers := []struct {
		text string
		role Role
	}{
		{"Tally engineering role: IMPLEMENTER.", RoleImplementer},
		{"Tally engineering role: REVIEWER.", RoleReviewer},
		{"Tally engineering role: REPAIRER.", RoleRepairer},
		{"Tally engineering role: VERIFIER.", RoleVerifier},
		{"Tally engineering role: SPECIALIST (", RoleSpecialist},
	}
	for _, marker := range markers {
		if strings.Contains(instruction, marker.text) {
			return marker.role, true
		}
	}
	return "", false
}
