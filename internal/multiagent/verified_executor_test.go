package multiagent

// Mandatory v1.5.4 adapter tests (issue #42 Phase 0.3), unit layer:
// candidate freeze/no-fabrication, deletion fail-safe, read-only role drift
// rejection, held-out without isolation fail-safe, and the interpreter's
// claim-is-not-evidence rule. Verification-executing cases (VERIFIED /
// FAILED / NEEDS_REVIEW with fact-store binding) live in
// verified_executor_integration_test.go behind the integration tag.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/changeset"
	"tally/internal/engineering"
	"tally/internal/gitwriter"
	"tally/internal/intake"
	"tally/internal/intakefreeze"
	"tally/internal/store"
	"tally/internal/taskexecution"
	"tally/internal/verification"
)

var vreSig = &gitwriter.Signature{Name: "vre-test", Email: "vre@test.local"}

func vreGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// vreFixture is one disposable content repository with a frozen TASK-0154
// bundle plus a linked disposable worktree at the freeze commit.
type vreFixture struct {
	repo      string
	worktree  string
	freezeSHA string
}

// vreBuild commits the manifest/policy base, then the frozen task trio via
// the production freeze path. The verification plan parameter lets cases
// choose visible, executable or held-out contracts.
func vreBuild(t *testing.T, plan intakefreeze.VerificationPlan) *vreFixture {
	t.Helper()
	repo := t.TempDir()
	vreGit(t, repo, "init", "--quiet")
	vreGit(t, repo, "config", "user.name", "vre-test")
	vreGit(t, repo, "config", "user.email", "vre@test.local")
	if err := os.MkdirAll(filepath.Join(repo, ".tally", "policies"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "project: vre-proj\nentities:\n  - id: server\n    type: service\n    path: cmd/server\n"
	policy := "id: db_gate\nenforcement: gate\nrationale: vre fixture\ntrigger:\n  risk_signals:\n    - schema_migration\n  paths:\n    - \"migrations/**\"\nrequires:\n  - fresh_database\ngate_condition:\n  mode: all\n"
	if err := os.WriteFile(filepath.Join(repo, ".tally", "project.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".tally", "policies", "db_gate.yaml"), []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "internal", "session"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "internal", "session", "session.go"), []byte("package session\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vreGit(t, repo, "add", ".")
	vreGit(t, repo, "commit", "--no-gpg-sign", "--quiet", "-m", "vre base")
	base := vreGit(t, repo, "rev-parse", "HEAD")

	fa, err := intakefreeze.Freeze(intakefreeze.ConfirmedDraft{
		TaskID:    "TASK-0154",
		Profile:   "standard",
		SourceRef: "v1.5.4 adapter test",
		Draft: intake.Draft{
			TaskType:  intake.TaskTypeBugFix,
			Summary:   "session must be revoked on logout",
			Objective: "logout must revoke the session immediately",
			Scope: intake.Scope{
				AllowedPaths:   []string{"internal/session/**"},
				ForbiddenPaths: []string{"internal/legacy/**"},
			},
			Acceptance: []intake.AcceptanceClause{
				{ID: "AC-01", Statement: "requests with a revoked session return 401"},
			},
		},
		RiskHints: []string{"auth_session"},
	}, plan)
	if err != nil {
		t.Fatalf("intakefreeze.Freeze: %v", err)
	}
	if _, err := artifacts.FreezeCommit(context.Background(), repo, artifacts.FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    fa,
		Message:      "freeze: TASK-0154",
		Signature:    vreSig,
	}); err != nil {
		t.Fatalf("FreezeCommit: %v", err)
	}
	freezeSHA := vreGit(t, repo, "rev-parse", "HEAD")

	worktree := filepath.Join(t.TempDir(), "wt")
	vreGit(t, repo, "worktree", "add", "--detach", worktree, freezeSHA)
	t.Cleanup(func() {
		vreGit(t, repo, "worktree", "remove", "--force", worktree)
	})
	return &vreFixture{repo: repo, worktree: worktree, freezeSHA: freezeSHA}
}

func vreVisiblePlan() intakefreeze.VerificationPlan {
	return intakefreeze.VerificationPlan{
		Evidence: []intakefreeze.EvidenceDeclaration{{
			AcceptanceID: "AC-01",
			Kind:         verification.EvidenceTest,
			Path:         "internal/session/logout_test.go",
		}},
	}
}

func vreExecutor(t *testing.T, f *vreFixture, delegate taskexecution.AgentExecutor, caps agentruntime.CapabilitySet) *VerifiedRoleExecutor {
	t.Helper()
	if caps == nil {
		caps = agentruntime.NewCapabilitySet()
	}
	ex, err := NewVerifiedRoleExecutor(VerifiedRoleExecutorConfig{
		Delegate:     delegate,
		Repo:         f.repo,
		WorktreeRoot: t.TempDir(),
		EnvName:      "vre-env",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "vre-fake", Capabilities: caps},
		Queries:      store.New(nil),
		Now:          func() time.Time { return time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return ex
}

func vreRequest(f *vreFixture, role string) taskexecution.TaskExecutionRequest {
	return taskexecution.TaskExecutionRequest{
		TaskID:             "TASK-0154",
		FrozenTaskContract: engineering.ArtifactRef{ID: "TASK-0154:contract", Kind: engineering.ArtifactTaskContract, URI: "git://task"},
		Runtime:            taskexecution.RuntimeRef{ID: "vre-fake"},
		Environment:        taskexecution.EnvironmentRef{Kind: taskexecution.EnvironmentLocal, Target: "local"},
		Workspace: taskexecution.WorkspaceRef{
			Environment: taskexecution.EnvironmentLocal, Target: "local",
			Identity: "ws-TASK-0154", Path: f.worktree, BaseRevision: f.freezeSHA,
		},
		Instruction: "Tally engineering role: " + role + ". Work the frozen contract.",
	}
}

// Case 1: an Implementer that changed the disposable worktree freezes an
// immutable ChangeSet; no evidence is fabricated from the Agent claim.
func TestVerifiedRoleExecutorImplementerFreezesChangeSet(t *testing.T) {
	f := vreBuild(t, vreVisiblePlan())
	if err := os.WriteFile(filepath.Join(f.worktree, "internal", "session", "logout.go"), []byte("package session\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := vreExecutor(t, f, &scriptedExecutor{steps: []scriptedStep{{run: "run-impl-1"}}}, nil)
	res, err := ex.Execute(context.Background(), vreRequest(f, "IMPLEMENTER"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != taskexecution.ExecutionCompleted {
		t.Fatalf("status=%s", res.Status)
	}
	if strings.TrimSpace(res.CandidateRef) == "" {
		t.Fatal("implementer produced no immutable candidate reference")
	}
	if len(res.EvidenceRefs) != 0 {
		t.Fatalf("implementer claim fabricated evidence: %+v", res.EvidenceRefs)
	}
}

// Case 2: an Implementer that produced no candidate change must fail safe —
// no fabricated candidate, no guessed success.
func TestVerifiedRoleExecutorImplementerNoChangeFailsSafe(t *testing.T) {
	f := vreBuild(t, vreVisiblePlan())
	ex := vreExecutor(t, f, &scriptedExecutor{steps: []scriptedStep{{run: "run-impl-empty"}}}, nil)
	res, err := ex.Execute(context.Background(), vreRequest(f, "IMPLEMENTER"))
	if !errors.Is(err, ErrNoFrozenCandidate) {
		t.Fatalf("err=%v, want ErrNoFrozenCandidate", err)
	}
	if strings.TrimSpace(res.CandidateRef) != "" {
		t.Fatalf("fabricated candidate on empty change: %q", res.CandidateRef)
	}
}

// Case 3: a candidate that deletes a tracked file is an explicit typed
// failure, never a guessed success.
func TestVerifiedRoleExecutorCandidateDeletionFails(t *testing.T) {
	f := vreBuild(t, vreVisiblePlan())
	if err := os.Remove(filepath.Join(f.worktree, "internal", "session", "session.go")); err != nil {
		t.Fatal(err)
	}
	ex := vreExecutor(t, f, &scriptedExecutor{steps: []scriptedStep{{run: "run-impl-del"}}}, nil)
	_, err := ex.Execute(context.Background(), vreRequest(f, "IMPLEMENTER"))
	if !errors.Is(err, changeset.ErrCandidateDeletion) {
		t.Fatalf("err=%v, want changeset.ErrCandidateDeletion", err)
	}
}

// mutatingExecutor simulates a misbehaving read-only Agent that writes into
// the candidate workspace during its run.
type mutatingExecutor struct {
	write func() error
}

func (x *mutatingExecutor) Execute(_ context.Context, _ taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	if err := x.write(); err != nil {
		return taskexecution.TaskExecutionResult{}, err
	}
	return taskexecution.TaskExecutionResult{
		Status:     taskexecution.ExecutionCompleted,
		AgentClaim: agentruntime.AgentRunResult{RunID: "run-mutating", SessionID: "s", Status: agentruntime.RunCompleted},
	}, nil
}

func (x *mutatingExecutor) Cancel(engineering.ID) error { return nil }

// vreSeedCandidate leaves the implementer's candidate as uncommitted worktree
// state, the shape a read-only role receives from the Engine.
func vreSeedCandidate(t *testing.T, f *vreFixture) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.worktree, "internal", "session", "logout.go"), []byte("package session\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Case 4: a Reviewer/Verifier/Specialist that mutates the fixed candidate
// changes the candidate identity and is rejected before any verification.
func TestVerifiedRoleExecutorReadOnlyMutationRejected(t *testing.T) {
	for _, role := range []string{"REVIEWER", "VERIFIER"} {
		t.Run(role, func(t *testing.T) {
			f := vreBuild(t, vreVisiblePlan())
			vreSeedCandidate(t, f)
			mut := &mutatingExecutor{write: func() error {
				return os.WriteFile(filepath.Join(f.worktree, "internal", "session", "tampered.go"), []byte("package session\n"), 0o644)
			}}
			ex := vreExecutor(t, f, mut, nil)
			_, err := ex.Execute(context.Background(), vreRequest(f, role))
			if !errors.Is(err, ErrReadOnlyRoleMutatedCandidate) {
				t.Fatalf("err=%v, want ErrReadOnlyRoleMutatedCandidate", err)
			}
		})
	}
}

// Case 7b: held-out verification without a session-isolation runtime fails
// safe — explicit error, no evidence, never PASS.
func TestVerifiedRoleExecutorHeldOutRequiresIsolatedRuntime(t *testing.T) {
	plan := intakefreeze.VerificationPlan{
		Evidence: []intakefreeze.EvidenceDeclaration{{
			AcceptanceID: "AC-01",
			Kind:         verification.EvidenceTest,
			HeldOut:      true,
			Store:        "holdout",
			Ref:          "vre/heldout",
		}},
		RuntimeRequirements: &verification.RuntimeRequirements{SessionIsolation: true, HeldOutUnreachable: true, ConformanceClass: "verification-isolation-v1"},
	}
	f := vreBuild(t, plan)
	vreSeedCandidate(t, f)
	ex := vreExecutor(t, f, &scriptedExecutor{steps: []scriptedStep{{run: "run-ver-heldout"}}}, nil)
	res, err := ex.Execute(context.Background(), vreRequest(f, "VERIFIER"))
	if err == nil || !strings.Contains(err.Error(), "session.isolation") {
		t.Fatalf("err=%v, want held-out isolation refusal", err)
	}
	if len(res.EvidenceRefs) != 0 {
		t.Fatalf("evidence fabricated without isolation: %+v", res.EvidenceRefs)
	}
}

// Case 8 (interpreter boundary): an Agent summary/claim without Tally
// verification facts cannot satisfy reviewer/verifier acceptance; factual
// PASS evidence can; FAIL evidence routes to repair; NOT_RUN/ENV-BLOCKED
// never passes.
func TestEvidenceInterpreterClaimIsNotAcceptance(t *testing.T) {
	in := EvidenceInterpreter{}
	rc := RoleContext{TaskID: "TASK-0154", Role: RoleReviewer, RunID: "run-review-1"}
	completed := func(refs ...engineering.EvidenceRef) taskexecution.TaskExecutionResult {
		return taskexecution.TaskExecutionResult{
			Status:       taskexecution.ExecutionCompleted,
			AgentClaim:   agentruntime.AgentRunResult{RunID: "run-review-1", Status: agentruntime.RunCompleted, Summary: "looks good to me"},
			EvidenceRefs: refs,
		}
	}

	d, err := in.Interpret(context.Background(), rc, completed())
	if err != nil || d.Kind != DecisionBlock {
		t.Fatalf("claim-only decision=%+v err=%v, want block", d, err)
	}
	d, err = in.Interpret(context.Background(), rc, completed(testEvidence("ev-pass", engineering.EvidencePass)))
	if err != nil || d.Kind != DecisionPass {
		t.Fatalf("pass-evidence decision=%+v err=%v, want pass", d, err)
	}
	d, err = in.Interpret(context.Background(), rc, completed(testEvidence("ev-fail", engineering.EvidenceFail)))
	if err != nil || d.Kind != DecisionRepair || len(d.Findings) != 1 {
		t.Fatalf("fail-evidence decision=%+v err=%v, want repair with finding", d, err)
	}
	d, err = in.Interpret(context.Background(), rc, completed(testEvidence("ev-notrun", engineering.EvidenceNotRun)))
	if err != nil || d.Kind != DecisionBlock {
		t.Fatalf("not-run decision=%+v err=%v, want block", d, err)
	}

	// Implementer/Repairer completion remains a claim that only advances to
	// review — never a PASS evidence decision by itself.
	d, err = in.Interpret(context.Background(), RoleContext{TaskID: "TASK-0154", Role: RoleImplementer, RunID: "run-impl-1"}, completed())
	if err != nil || d.Kind != DecisionPass {
		t.Fatalf("implementer decision=%+v err=%v", d, err)
	}
}

// Constructor fail-closed validation.
func TestVerifiedRoleExecutorConfigValidation(t *testing.T) {
	f := vreBuild(t, vreVisiblePlan())
	base := VerifiedRoleExecutorConfig{
		Delegate:     &scriptedExecutor{},
		Repo:         f.repo,
		WorktreeRoot: t.TempDir(),
		EnvName:      "vre-env",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "vre-fake", Capabilities: agentruntime.NewCapabilitySet()},
		Queries:      store.New(nil),
	}
	for name, mutate := range map[string]func(*VerifiedRoleExecutorConfig){
		"nil delegate":    func(c *VerifiedRoleExecutorConfig) { c.Delegate = nil },
		"empty repo":      func(c *VerifiedRoleExecutorConfig) { c.Repo = " " },
		"empty worktree":  func(c *VerifiedRoleExecutorConfig) { c.WorktreeRoot = "" },
		"empty env":       func(c *VerifiedRoleExecutorConfig) { c.EnvName = "" },
		"nil queries":     func(c *VerifiedRoleExecutorConfig) { c.Queries = nil },
		"invalid runtime": func(c *VerifiedRoleExecutorConfig) { c.Runtime = agentruntime.RuntimeDescriptor{} },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base
			mutate(&cfg)
			if _, err := NewVerifiedRoleExecutor(cfg); err == nil {
				t.Fatalf("%s accepted", name)
			}
		})
	}
	if _, err := NewVerifiedRoleExecutor(base); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
}

// Requests without a v1.5.4 role directive are rejected before any Agent run.
func TestVerifiedRoleExecutorRequiresRoleDirective(t *testing.T) {
	f := vreBuild(t, vreVisiblePlan())
	ex := vreExecutor(t, f, &scriptedExecutor{}, nil)
	req := vreRequest(f, "IMPLEMENTER")
	req.Instruction = "implement the frozen task contract"
	if _, err := ex.Execute(context.Background(), req); err == nil {
		t.Fatal("request without role directive accepted")
	}
}
