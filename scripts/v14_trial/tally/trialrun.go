// trialrun.go executes ONE frozen task through the REAL Tally closed loop on
// the REAL codex runtime — the test-equivalent composition of the worker v14
// production path (cmd/worker/v14.go runV14Task), replicated per P2-10/
// P3-03/P3-04 e2e precedent (test/e2e/v14_closedloop_harness_test.go and
// successors; the worker path itself is package main and cannot be imported):
//
//	frozen set load (hash-pinned) → per-task fixture repo (compose.go/repo.go)
//	→ artifacts.Load(HEAD) [task + frozen VerificationContract + policy bundle]
//	→ risk.Derive → contract.Compile + contract.Bind (P1-07 compiler seams)
//	→ workspace.New one-shot worktree root + RunTable
//	→ contextpack.Select (with the compiled requires Prevention context)
//	→ runstore production fact sinks (RunStoreSink / Lineage / Exposure; live
//	  PostgreSQL per the trial environment)
//	→ candidateverify.New (production candidate verification service; binding
//	  commands = the verification probe over the frozen oracle semantics)
//	→ closedloop.NewLoop (P2-08 production composition root) driving the REAL
//	  CodexRuntime (per-session isolated CODEX_HOME + ADR-0003 provider
//	  seeding + auth.json handoff from a minimal owner home)
//	→ external judge: the SAME arm-neutral `bare judge` command arm A uses.
//
// Honest accounting: requested configuration is recorded separately from
// observed identity (Decision Lock 15); the codex exec JSONL surface does not
// self-report the model, so the observed model is recorded as unknown — never
// backfilled from the request (RUN_PROTOCOL §5).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/contextpack"
	"tally/internal/contract"
	"tally/internal/core"
	"tally/internal/engineeringpolicy"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/risk"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/codex"
	"tally/test/eval/v14_product"
)

// trialConfig is the Tally arm's requested configuration surface (recorded per
// row; never conflated with observed identity).
type trialConfig struct {
	Repo           string        // owner repository root (frozen assets)
	RunsRoot       string        // Tally-arm trial records root
	ToolsDir       string        // probe + psqlsh binaries
	CodexBin       string        // codex CLI binary path (explicit, parity with arm A)
	Model          string        // requested model id (recorded as requested only)
	Sandbox        string        // -s mode (workspace-write, parity with arm A)
	AttemptCeiling time.Duration // per-attempt wall-clock ceiling (production policy value)
	TaskCeiling    time.Duration // per-task wall-clock ceiling for the whole loop (documented)
	DSN            string        // PostgreSQL DSN with database name (facts store)
	OwnerCodexHome string        // minimal owner home carrying ONLY auth.json
	AuthPresent    bool          // whether the credential file existed (skip-not-fail)
	EnvName        string        // verification environment stable name
}

// trialResult is one task's recorded run (loop + judge + identity).
type trialResult struct {
	TaskID         string            `json:"task_id"`
	ContractTaskID string            `json:"contract_task_id"`
	StartedAt      string            `json:"started_at"`
	FinishedAt     string            `json:"finished_at"`
	WallSeconds    float64           `json:"wall_seconds"`
	LoopPhase      string            `json:"loop_phase"`
	Outcome        string            `json:"outcome"` // accepted | rejected | escalated | cancelled | not_run
	OutcomeDetail  string            `json:"outcome_detail"`
	Escalation     *escalationRecord `json:"escalation,omitempty"`
	Attempts       []attemptRecord   `json:"attempts"`
	Verifications  []verdictRecord   `json:"verification_lineage"`
	Composition    *taskComposition  `json:"composition"`
	Coverage       coverageRecord    `json:"policy_match_coverage"`
	Identity       identityRecord    `json:"identity"`
	Requested      requestedRecord   `json:"requested"`
	ManifestSHA    string            `json:"manifest_sha256"`
	FrozenHash     string            `json:"frozen_verification_contract_hash"`
	Workspace      string            `json:"candidate_workspace"`
	Judge          json.RawMessage   `json:"judge_result,omitempty"`
	JudgeInvoked   bool              `json:"judge_invoked"`
	JudgeNote      string            `json:"judge_note,omitempty"`
	Notes          []string          `json:"notes"`
}

type escalationRecord struct {
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

type attemptRecord struct {
	Attempt          int                           `json:"attempt"`
	RunID            string                        `json:"run_id"`
	ParentRun        string                        `json:"parent_run,omitempty"`
	Supersedes       string                        `json:"supersedes,omitempty"`
	ChangeSet        string                        `json:"changeset,omitempty"`
	HasCandidate     bool                          `json:"has_candidate"`
	Status           string                        `json:"status"`
	FailureReason    string                        `json:"failure_reason,omitempty"`
	Iterations       int                           `json:"iterations"`
	AgentClaim       string                        `json:"agent_claim,omitempty"`
	WallSeconds      float64                       `json:"wall_seconds"`
	ObservedIdentity agentruntime.ObservedIdentity `json:"observed_identity"`
	WorkspaceID      string                        `json:"workspace_id"`
}

type verdictRecord struct {
	VerifyRun  string   `json:"verify_run"`
	ChangeSet  string   `json:"changeset"`
	Status     string   `json:"status"`
	BindingIDs []string `json:"binding_ids,omitempty"`
	GateNames  []string `json:"gate_names,omitempty"`
}

type coverageRecord struct {
	MatchedPolicies   []string `json:"matched_policies"`
	GateKeys          []string `json:"gate_keys"`
	Confidence        string   `json:"confidence"`
	DetectedRisk      []string `json:"detected_risk_signals"`
	UnmatchedRisk     []string `json:"unmatched_risk_signals"`
	RequiresPolicyIDs []string `json:"requires_policy_ids"`
}

type identityRecord struct {
	HostOS         string `json:"host_os"`
	HostArch       string `json:"host_arch"`
	GoVersion      string `json:"go_version"`
	TrialDate      string `json:"trial_date"`
	DBVersion      string `json:"db_version,omitempty"`
	DBVersionKnown bool   `json:"db_version_recorded"`
	RuntimeName    string `json:"runtime_name"`
	AdapterVersion string `json:"adapter_version"`
	CLIVersion     string `json:"cli_version"`
	BinaryPath     string `json:"binary_path"`
	ModelObserved  bool   `json:"model_observed"`
	ModelNote      string `json:"model_note"`
}

type requestedRecord struct {
	CodexBin       string         `json:"codex_bin"`
	Model          string         `json:"model"`
	Sandbox        string         `json:"sandbox"`
	AttemptSeconds int            `json:"attempt_ceiling_seconds"`
	TaskSeconds    int            `json:"task_ceiling_seconds"`
	Provider       providerRecord `json:"provider"`
	AuthHandoff    string         `json:"auth_handoff"`
}

type providerRecord struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	BaseURL            string `json:"base_url"`
	WireAPI            string `json:"wire_api"`
	RequiresOpenAIAuth bool   `json:"requires_openai_auth"`
}

// outcomeRecorder wraps the production candidate verification service to
// record the verification lineage and each verdict's exact ChangeSet (needed
// to materialize the accepted candidate for the external judge). It changes no
// verification behavior.
type outcomeRecorder struct {
	inner closedloop.CandidateVerification

	mu       sync.Mutex
	outcomes []*verification.FinalOutcome
	changes  map[string]*changeset.ChangeSet // FinalOutcome.RunID → verified ChangeSet
}

func (r *outcomeRecorder) Verify(ctx context.Context, req candidateverify.Request) (*verification.FinalOutcome, error) {
	out, err := r.inner.Verify(ctx, req)
	if err == nil && out != nil {
		r.mu.Lock()
		r.outcomes = append(r.outcomes, out)
		if r.changes == nil {
			r.changes = map[string]*changeset.ChangeSet{}
		}
		r.changes[fmt.Sprintf("%d", out.RunID)] = req.ChangeSet
		r.mu.Unlock()
	}
	return out, err
}

func (r *outcomeRecorder) snapshot() ([]*verification.FinalOutcome, map[string]*changeset.ChangeSet) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]*verification.FinalOutcome(nil), r.outcomes...)
	m := map[string]*changeset.ChangeSet{}
	for k, v := range r.changes {
		m[k] = v
	}
	return out, m
}

// runTask executes the full Tally-arm trial for one frozen task.
func runTask(set *frozenSet, task *v14product.ManifestTask, cfg trialConfig) (*trialResult, error) {
	started := time.Now()
	res := &trialResult{
		TaskID:         task.TaskID,
		ContractTaskID: contractTaskIDFor(set, task.TaskID),
		ManifestSHA:    set.sha,
		Identity:       captureIdentity(),
		Requested: requestedRecord{
			CodexBin:       cfg.CodexBin,
			Model:          cfg.Model,
			Sandbox:        cfg.Sandbox,
			AttemptSeconds: int(cfg.AttemptCeiling.Seconds()),
			TaskSeconds:    int(cfg.TaskCeiling.Seconds()),
			Provider:       trialProviderRecord(),
			AuthHandoff:    "auth.json copied from the minimal owner home (contents never recorded); present=" + fmt.Sprintf("%v", cfg.AuthPresent),
		},
	}
	defer func() {
		res.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		res.WallSeconds = time.Since(started).Seconds()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.TaskCeiling)
	defer cancel()

	taskDir := filepath.Join(cfg.RunsRoot, task.TaskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return res, err
	}

	// ---- 0. tools (probe) ---------------------------------------------------
	probePath, err := ensureProbe(cfg.ToolsDir)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "probe build failed: "+err.Error()
		return res, err
	}

	// ---- 1. composition inputs (frozen bytes only) --------------------------
	acc, err := loadAcceptanceDoc(set.root, task.TaskID)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "frozen acceptance unreadable: "+err.Error()
		return res, err
	}
	intentRel := "test/eval/v14_product/testdata/tasks/" + task.TaskID + "/intent.md"
	intent, err := os.ReadFile(filepath.Join(set.root, filepath.FromSlash(intentRel)))
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "frozen intent unreadable: "+err.Error()
		return res, err
	}
	dsnBase := maintenanceURI(cfg.DSN)
	taskYAML, verifYAML, reqs, rec, err := composeInputs(set, task, acc, intent, probePath, set.root, dsnBase)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "contract composition failed: "+err.Error()
		return res, err
	}
	res.Composition = rec

	// ---- 2. per-task fixture repository -------------------------------------
	repoDir := filepath.Join(taskDir, "repo")
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		res.Outcome, res.OutcomeDetail = "not_run", "fixture repository already exists; clean the runs root before re-running"
		return res, fmt.Errorf("repo %s already exists", repoDir)
	}
	baseRev, headRev, err := buildTrialRepo(set, task, acc, rec, taskYAML, verifYAML, reqs, repoDir)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "fixture repo build failed: "+err.Error()
		return res, err
	}
	_ = headRev // HEAD carries the verification authority; the loop works from baseRev

	// ---- 3. real runtime identity (observed, before anything runs) ----------
	rt := codex.New(codex.Config{
		BinPath: cfg.CodexBin,
		Sandbox: cfg.Sandbox,
		Model:   cfg.Model,
		Env:     map[string]string{"CODEX_HOME": cfg.OwnerCodexHome},
		Provider: &codex.ProviderEndpoint{
			ID:                 "custom",
			Name:               "deepseek",
			BaseURL:            "https://api.deepseek.com",
			WireAPI:            "responses",
			RequiresOpenAIAuth: true,
		},
	})
	snap, err := rt.Snapshot(ctx)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "codex runtime identity unavailable: "+err.Error()
		return res, err
	}
	res.Identity.RuntimeName = snap.Name
	res.Identity.AdapterVersion = snap.AdapterVersion
	res.Identity.CLIVersion = snap.CLIVersion
	res.Identity.BinaryPath = snap.BinaryPath
	res.Identity.ModelNote = "the codex exec JSONL surface does not self-report the model; the requested " +
		"-m value is recorded separately under requested.model and never backfilled (Decision Lock 15, RUN_PROTOCOL §5)"

	// ---- 4. facts store (live PostgreSQL) -----------------------------------
	st, err := store.Open(ctx, cfg.DSN)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "facts store unavailable: "+err.Error()
		return res, err
	}
	defer st.Close()
	queries := st.Queries()

	// ---- 5. Git HEAD snapshot load (production loader) ----------------------
	asnap, err := artifacts.Load(repoDir, "HEAD")
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "artifacts load failed: "+err.Error()
		return res, err
	}
	var tc taskcontract.TaskContract
	found := false
	for _, a := range asnap.Tasks {
		if a.Contract.ID == rec.ContractTaskID {
			tc, found = a.Contract, true
		}
	}
	if !found {
		res.Outcome, res.OutcomeDetail = "not_run", fmt.Sprintf("HEAD snapshot has no task %s; rejected: %s", rec.ContractTaskID, rejectionsSummary(asnap.Rejected))
		return res, fmt.Errorf("task %s not in snapshot", rec.ContractTaskID)
	}
	if tc.Profile != taskcontract.ProfileStandard {
		res.Outcome, res.OutcomeDetail = "not_run", "contract profile is not standard"
		return res, fmt.Errorf("profile %q", tc.Profile)
	}
	var frozen *verification.Frozen
	for _, v := range asnap.Verifications {
		if v.Frozen.TaskID() == rec.ContractTaskID {
			frozen = v.Frozen
		}
	}
	if frozen == nil {
		res.Outcome, res.OutcomeDetail = "not_run", "HEAD snapshot has no frozen VerificationContract for "+rec.ContractTaskID
		return res, fmt.Errorf("no frozen verification contract")
	}
	res.FrozenHash = frozen.Hash()

	// ---- 6. policy bundle + risk facts + contract compiler seams ------------
	psnap, err := artifacts.LoadPolicies(repoDir, "HEAD")
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "policy load failed: "+err.Error()
		return res, err
	}
	if len(psnap.Rejected) != 0 || len(psnap.Policies) != 1 {
		res.Outcome, res.OutcomeDetail = "not_run", fmt.Sprintf("policy load = %d policies, %d rejections, want 1/0", len(psnap.Policies), len(psnap.Rejected))
		return res, fmt.Errorf("policy load mismatch")
	}
	bundle := psnap.Policies[0].Policy
	if bundle.ID != rec.BundleID {
		res.Outcome, res.OutcomeDetail = "not_run", fmt.Sprintf("loaded bundle %q, want %q", bundle.ID, rec.BundleID)
		return res, fmt.Errorf("bundle mismatch")
	}
	var signals []risk.Signal
	for i, id := range rec.RiskSignals {
		basis := rec.RiskBasis
		if len(basis) == 0 {
			basis = []string{"trial-composition:" + task.TaskID}
		}
		_ = i
		signals = append(signals, risk.Signal{ID: risk.SignalID(id), Basis: basis})
	}
	riskRes, err := risk.Derive(signals)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "risk derivation failed: "+err.Error()
		return res, err
	}
	snapAgg, err := contract.Compile(tc, frozen)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "contract compile failed: "+err.Error()
		return res, err
	}
	compiled, err := contract.Bind(snapAgg, []engineeringpolicy.Policy{bundle}, riskRes)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "contract bind failed: "+err.Error()
		return res, err
	}
	cov := compiled.Coverage()
	for _, m := range cov.MatchedPolicies {
		res.Coverage.MatchedPolicies = append(res.Coverage.MatchedPolicies, m.Policy.ID)
		res.Coverage.RequiresPolicyIDs = append(res.Coverage.RequiresPolicyIDs, m.Policy.ID)
	}
	res.Coverage.GateKeys = compiled.GateKeys()
	res.Coverage.Confidence = string(cov.Confidence)
	for _, s := range cov.DetectedRiskSignals {
		res.Coverage.DetectedRisk = append(res.Coverage.DetectedRisk, string(s))
	}
	for _, s := range cov.UnmatchedRiskSignals {
		res.Coverage.UnmatchedRisk = append(res.Coverage.UnmatchedRisk, string(s))
	}

	// ---- 7. one-shot workspaces ---------------------------------------------
	worktreeRoot := filepath.Join(taskDir, "worktrees")
	mgr, err := workspace.New(repoDir, worktreeRoot)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "workspace manager refused composition: "+err.Error()
		return res, err
	}
	worktrees := workspace.NewRunTable(mgr)
	verifyRoot := filepath.Join(taskDir, "verify-worktrees")

	// ---- 8. ContextPack coordinate (worker v13/v14 rule) --------------------
	packHash, err := contextPackHash(repoDir, baseRev, tc, frozen, compiled)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "contextpack failed: "+err.Error()
		return res, err
	}

	// ---- 9. production fact sinks + candidate verification service ----------
	runs := runstore.New(queries)
	wfSink := workflow.NewRunStoreSink(runs)
	recorder := &outcomeRecorder{}
	verifyImpl := candidateverify.New(candidateverify.Config{
		Repo:         repoDir,
		WorktreeRoot: verifyRoot,
		EnvName:      cfg.EnvName,
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-v14-trial/p4-03"},
	}, candidateverify.Deps{Queries: queries})
	recorder.inner = verifyImpl
	var verify closedloop.CandidateVerification = recorder
	lineage, err := closedloop.NewStoreLineageSink(runs)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "lineage sink: "+err.Error()
		return res, err
	}
	exposure, err := closedloop.NewStoreExposureSink(runs, frozen.Revision())
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "exposure sink: "+err.Error()
		return res, err
	}
	seeds, err := runs.ExposuresForTask(ctx, core.ProjectID(asnap.ProjectID), rec.ContractTaskID)
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "exposure seeds: "+err.Error()
		return res, err
	}

	// ---- 10. inner codegen execution seam (worker v14CodegenRunner mirror) --
	runner0 := &trialCodegenRunner{
		project:  core.ProjectID(asnap.ProjectID),
		task:     tc,
		policy:   policy.DefaultExecutionPolicy(),
		profile:  "default-coding",
		runtime:  rt,
		model:    cfg.Model,
		behavior: "v1",
		verifRev: frozen.Hash(),
		packHash: packHash,
		sink:     wfSink,
		runsRoot: cfg.RunsRoot,
		taskID:   task.TaskID,
	}

	// ---- 11. closed-loop composition root -----------------------------------
	repair := closedloop.DefaultRepairPolicy()
	loop, err := closedloop.NewLoop(closedloop.LoopConfig{
		Codegen:   runner0,
		Verify:    verify,
		Contract:  frozen,
		Runtime:   rt.Descriptor().Name,
		Worktrees: worktrees,
		BaseRevision: func(context.Context, string) (string, error) {
			// First-attempt baseline = the agent base commit (contract-sourced
			// Git revision; the verification authority rides in HEAD above it).
			return baseRev, nil
		},
		Repair:    repair,
		Lineage:   lineage,
		Exposure:  exposure,
		Exposures: seeds,
	})
	if err != nil {
		res.Outcome, res.OutcomeDetail = "not_run", "loop composition refused: "+err.Error()
		return res, err
	}

	// ---- 12. run the real closed loop ---------------------------------------
	res.LoopPhase = "running"
	loopRes, runErr := loop.Run(ctx, rec.ContractTaskID)
	appendEvent(cfg.RunsRoot, task.TaskID, 0, evAgentAttempt,
		fmt.Sprintf("closed loop finished: phase=%s attempts=%d err=%v", loopRes.Phase, len(loopRes.Attempts), runErr))

	switch {
	case runErr != nil:
		res.LoopPhase = string(loopRes.Phase)
		if loopRes.Phase == "" {
			res.LoopPhase = "error"
		}
		res.Outcome, res.OutcomeDetail = "not_run", "loop infrastructure error: "+errDetail(runErr)
		res.Notes = append(res.Notes, "the loop returned an error; any completed attempts are retained below (never silently dropped)")
	case loopRes.Phase == closedloop.PhaseAccepted:
		res.LoopPhase = "accepted"
		if loopRes.Basis == nil {
			res.Outcome, res.OutcomeDetail = "not_run", "accepted without basis (invariant violation)"
			return res, fmt.Errorf("accepted without basis")
		}
		res.Outcome, res.OutcomeDetail = "accepted", fmt.Sprintf("accepted at attempt %d (verify run %s)", loopRes.Basis.Attempt, loopRes.Basis.VerifyRun)
	case loopRes.Phase == closedloop.PhaseEscalated:
		res.LoopPhase = "escalated"
		res.Outcome = "escalated"
		if loopRes.Escalation != nil {
			res.Escalation = &escalationRecord{Reason: string(loopRes.Escalation.Reason), Detail: loopRes.Escalation.Detail}
			res.OutcomeDetail = "escalated: " + string(loopRes.Escalation.Reason) + " — " + loopRes.Escalation.Detail
			appendEvent(cfg.RunsRoot, task.TaskID, int(attemptCount(loopRes)), evEscalation,
				"reason="+string(loopRes.Escalation.Reason)+" detail="+loopRes.Escalation.Detail)
		} else {
			res.OutcomeDetail = "escalated (no typed reason)"
		}
	default:
		res.LoopPhase = string(loopRes.Phase)
		res.Outcome, res.OutcomeDetail = "cancelled", "loop phase "+string(loopRes.Phase)
	}

	// ---- 13. attempts + verification lineage (facts, never dropped) ---------
	for _, a := range loopRes.Attempts {
		rec2 := attemptRecord{
			Attempt:      int(a.No),
			RunID:        string(a.RunID),
			ParentRun:    string(a.ParentRun),
			Supersedes:   string(a.Supersedes),
			ChangeSet:    string(a.ChangeSet),
			HasCandidate: a.ChangeSet != "",
		}
		if r0 := runner0.record(string(a.RunID)); r0 != nil {
			rec2.Status = r0.status
			rec2.FailureReason = r0.failureReason
			rec2.Iterations = r0.iterations
			rec2.AgentClaim = truncateUTF8(r0.agentClaim, 800)
			rec2.WallSeconds = r0.wallSeconds
			rec2.ObservedIdentity = r0.observed
			rec2.WorkspaceID = r0.workspaceID
		}
		res.Attempts = append(res.Attempts, rec2)
	}
	outcomes, changes := recorder.snapshot()
	for _, o := range outcomes {
		vr := verdictRecord{
			VerifyRun: fmt.Sprintf("%d", o.RunID),
			ChangeSet: string(o.ChangeSetID),
			Status:    string(o.Status),
		}
		for _, b := range o.Bindings {
			vr.BindingIDs = append(vr.BindingIDs, b.BindingID)
		}
		for _, g := range o.Gates {
			vr.GateNames = append(vr.GateNames, g.Name)
		}
		res.Verifications = append(res.Verifications, vr)
	}

	// ---- 14. materialize the delivered candidate for the external judge -----
	lastChangeSet := lastCandidateChangeSet(loopRes, changes)
	candWS := filepath.Join(taskDir, "candidate-ws")
	if err := materializeCandidateWorkspace(set, task, candWS, lastChangeSet); err != nil {
		res.OutcomeDetail += "; candidate materialization failed: " + err.Error()
		res.Notes = append(res.Notes, "judge NOT_RUN: candidate workspace could not be materialized: "+err.Error())
		res.JudgeNote = "NOT_RUN(candidate materialization failed: " + err.Error() + ")"
		return res, nil
	}
	res.Workspace = candWS

	// ---- 15. the SAME arm-neutral judge command arm A uses ------------------
	judgeRes, judgeNote, jerr := delegateJudge(set, cfg, task, candWS, filepath.Join(taskDir, "judge-result.json"))
	res.JudgeNote = judgeNote
	if jerr == nil {
		res.Judge = judgeRes
		res.JudgeInvoked = true
		accepted, complete := judgeVerdict(judgeRes)
		if res.Outcome == "accepted" {
			if accepted {
				// accepted stays accepted
			} else if complete {
				res.Outcome = "rejected"
				res.OutcomeDetail += "; external judge NOT_ACCEPTED (loop/judge disagreement, recorded honestly)"
			} else {
				res.Outcome = "not_run"
				res.OutcomeDetail += "; external judge incomplete (NOT_RUN keys retained)"
			}
		} else if res.Outcome == "escalated" || res.Outcome == "cancelled" {
			// The loop's terminal state stands; the judge verdict on the last
			// candidate state is recorded for transparency.
			res.Notes = append(res.Notes, "external judge ran on the last candidate state for transparency; the loop verdict governs the row")
		}
	} else {
		res.Notes = append(res.Notes, "judge NOT_RUN: "+judgeNote)
	}

	return res, nil
}

// attemptCount is a small helper for escalation journaling.
func attemptCount(r closedloop.Result) closedloop.AttemptNo {
	if n := len(r.Attempts); n > 0 {
		return r.Attempts[n-1].No
	}
	return 1
}

// lastCandidateChangeSet picks the candidate the external judge sees: the
// accepted candidate for accepted runs, else the LAST attempt's candidate
// (the loop's final delivered state; for no-candidate runs, nil = the base
// workspace, which is exactly what a do-nothing delivery looks like).
func lastCandidateChangeSet(loopRes closedloop.Result, changes map[string]*changeset.ChangeSet) *changeset.ChangeSet {
	if loopRes.Phase == closedloop.PhaseAccepted && loopRes.Basis != nil {
		if cs := changes[loopRes.Basis.VerifyRun]; cs != nil {
			return cs
		}
	}
	for i := len(loopRes.Attempts) - 1; i >= 0; i-- {
		id := loopRes.Attempts[i].ChangeSet
		if id == "" {
			continue
		}
		for _, cs := range changes {
			if string(cs.ID()) == string(id) {
				return cs
			}
		}
	}
	return nil
}

// materializeCandidateWorkspace provisions the agent-facing frozen layout and
// applies the candidate's changed files (full contents, core.FileChange
// convention). nil ChangeSet = the unchanged base workspace.
func materializeCandidateWorkspace(set *frozenSet, task *v14product.ManifestTask, ws string, cs *changeset.ChangeSet) error {
	if err := provisionTask(set, task, ws); err != nil {
		return err
	}
	if cs == nil {
		return nil
	}
	for _, f := range cs.Files() {
		rel := filepath.FromSlash(filepath.Clean(f.Path))
		if filepath.IsAbs(rel) || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("candidate path %q escapes the workspace", f.Path)
		}
		p := filepath.Join(ws, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(f.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// delegateJudge runs the arm-neutral judge EXACTLY as arm A runs it
// (RUN_PROTOCOL §3): `go run ./scripts/v14_trial/bare judge ...` with the
// shared --psql surface for live-DB evidence.
func delegateJudge(set *frozenSet, cfg trialConfig, task *v14product.ManifestTask, ws, outPath string) (json.RawMessage, string, error) {
	psqlshPath := filepath.Join(cfg.ToolsDir, "psqlsh"+exeSuffix())
	args := []string{
		"run", "./scripts/v14_trial/bare", "judge",
		"--repo", set.root,
		"--runs-root", cfg.RunsRoot,
		"--task", task.TaskID,
		"--workspace", ws,
		"--out", outPath,
		"--psql-bin", psqlshPath,
	}
	if task.Harness == "database_migration" {
		args = append(args, "--pg", maintenanceURI(cfg.DSN))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...) //nolint:gosec // structured argv, explicit flags
	cmd.Dir = set.root
	out, err := cmd.CombinedOutput()
	_ = os.WriteFile(filepath.Join(filepath.Dir(outPath), "judge-invocation.txt"),
		[]byte("argv: go "+strings.Join(args, " ")+"\n\n"+string(out)), 0o644)
	if err != nil {
		return nil, "judge command failed: " + truncateUTF8(strings.TrimSpace(string(out)), 500), err
	}
	b, rerr := os.ReadFile(outPath)
	if rerr != nil {
		return nil, "judge result unreadable: " + rerr.Error(), rerr
	}
	return json.RawMessage(b), "judge executed (identical command as the bare arm)", nil
}

// judgeVerdict extracts (accepted, complete) from the judge result JSON
// (shape owned by scripts/v14_trial/bare/judge.go).
func judgeVerdict(raw json.RawMessage) (accepted, complete bool) {
	var doc struct {
		Accepted bool `json:"accepted"`
		Complete bool `json:"complete"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return false, false
	}
	return doc.Accepted, doc.Complete
}

func maintenanceURI(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	// The arm-neutral judge appends "/<db>" to this URI, so the query string
	// must be dropped too (a URI like "...?sslmode=disable/db" does not parse).
	u.Path = ""
	u.RawQuery = ""
	return u.String()
}

func rejectionsSummary(rej []artifacts.Rejection) string {
	if len(rej) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(rej))
	for _, r := range rej {
		parts = append(parts, r.Path+": "+r.Reason)
	}
	return strings.Join(parts, " | ")
}

func errDetail(err error) string {
	s := err.Error()
	return truncateUTF8(s, 800)
}

func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// contextPackHash mirrors the worker v13/v14 ContextPack rule: candidates from
// the base revision tree (implementation files, .tally excluded), selected by
// contextpack.Select together with the compiled requires Prevention context.
func contextPackHash(repoDir, baseRev string, tc taskcontract.TaskContract, frozen *verification.Frozen, compiled *contract.Compiled) (string, error) {
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{"git", "-C", repoDir, "ls-tree", "-r", "--name-only", "-z", baseRev},
		Env:  gitEnv(),
	})
	if err != nil || res.ExitCode != 0 {
		return "", fmt.Errorf("git ls-tree %s: err=%v res=%v", baseRev, err, res)
	}
	seen := map[string]bool{}
	var candidates []contextpack.Candidate
	for _, p := range strings.Split(res.Stdout, "\x00") {
		if p == "" || seen[p] || strings.HasPrefix(p, ".tally/") {
			continue
		}
		seen[p] = true
		candidates = append(candidates, contextpack.Candidate{Path: p})
	}
	out, err := contextpack.Select(contextpack.SelectInput{
		Task:         tc,
		Verification: frozen,
		Candidates:   candidates,
		Requires:     compiled.Requires(),
	})
	if err != nil {
		return "", err
	}
	return out.Pack.Hash(), nil
}

func exeSuffix() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}

// ensureProbe builds the verification probe into the tools dir (go build is
// cached; the probe source is trial tooling under scripts/v14_trial/tally).
func ensureProbe(toolsDir string) (string, error) {
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(toolsDir, "v14tallyprobe"+exeSuffix())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, "./scripts/v14_trial/tally/probe") //nolint:gosec // structured argv
	out2, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build probe: %v: %s", err, truncateUTF8(string(out2), 500))
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// rejudgeTask re-runs the arm-neutral judge on an already-materialized
// candidate workspace and updates the task's run record. It exists for
// completing the judge step when the original judge execution was incomplete
// for an infra reason (e.g. the 2026-09-10 maintenance-URI query-string bug
// that made every database_migration gate NOT_RUN). It never touches the
// loop facts, the candidate workspace, or the frozen bytes: judging is
// external, post-hoc and arm-neutral. The superseded incomplete judge result
// is preserved as judge-result-incomplete-<reason>.json.
func rejudgeTask(set *frozenSet, cfg trialConfig, task *v14product.ManifestTask, reason string) error {
	taskDir := filepath.Join(cfg.RunsRoot, task.TaskID)
	recPath := filepath.Join(taskDir, "run-record.json")
	b, err := os.ReadFile(recPath)
	if err != nil {
		return fmt.Errorf("run record unreadable: %w", err)
	}
	var res trialResult
	if err := json.Unmarshal(b, &res); err != nil {
		return fmt.Errorf("run record corrupt: %w", err)
	}
	ws := filepath.Join(taskDir, "candidate-ws")
	if _, err := os.Stat(ws); err != nil {
		return fmt.Errorf("candidate workspace missing: %w", err)
	}
	if res.Judge != nil {
		superseded := filepath.Join(taskDir, "judge-result-incomplete-"+reason+".json")
		if err := os.WriteFile(superseded, res.Judge, 0o644); err != nil {
			return err
		}
		res.Notes = append(res.Notes, "superseded incomplete judge result preserved: "+filepath.Base(superseded))
	}
	judgeRes, judgeNote, jerr := delegateJudge(set, cfg, task, ws, filepath.Join(taskDir, "judge-result.json"))
	if jerr != nil {
		res.JudgeNote = judgeNote
	} else {
		res.Judge = judgeRes
		res.JudgeInvoked = true
		res.JudgeNote = judgeNote
		accepted, complete := judgeVerdict(judgeRes)
		switch {
		case res.Outcome == "accepted" || res.Outcome == "rejected":
			if res.Outcome == "accepted" && !accepted && complete {
				res.Outcome = "rejected"
				res.OutcomeDetail += "; external judge NOT_ACCEPTED (loop/judge disagreement, recorded honestly)"
			}
		case res.Outcome == "escalated" || res.Outcome == "cancelled":
			res.Notes = append(res.Notes, "external judge re-run on the preserved last candidate state for transparency; the loop verdict governs the row")
		}
	}
	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(recPath, append(out, []byte("\n")...), 0o644)
}
