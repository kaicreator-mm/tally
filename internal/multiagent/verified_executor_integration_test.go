//go:build integration

package multiagent

// Mandatory v1.5.4 adapter tests (issue #42 Phase 0.3), integration layer:
// the verifier path runs the real candidateverify authority against a real
// fact store (store.TestDB / PostgreSQL). The emitted EvidenceRef must be
// bound to the persisted verification run and its evidence facts — never to
// the Agent claim. Covered outcomes: VERIFIED -> factual PASS (case 5),
// FAILED -> factual FAIL + deterministic repair routing (case 6),
// NEEDS_REVIEW -> NOT_RUN, never PASS (case 7a).

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/store"
	"tally/internal/taskexecution"
	"tally/internal/verification"
)

// vreExecutableContract freezes a single visible, executable binding for
// TASK-0154/AC-01 so the verifier path runs a deterministic protocol command.
// A non-none independence requirement carries a usable declared source basis
// (planned coverage, §8.5.1) while effective independence still derives at
// run time from the actual implementation provenance.
func vreExecutableContract(t *testing.T, indep verification.IndependenceRequired, argv []string) *verification.Frozen {
	t.Helper()
	acc := verification.Acceptance{
		ID:                   "AC-01",
		Statement:            "requests with a revoked session return 401",
		IndependenceRequired: indep,
		EvidenceBindings: []verification.Binding{{
			ID:               "VIS-1",
			AuthoringMode:    verification.AuthoringPreCommit,
			VisibilityPolicy: verification.VisibilityVisible,
			Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Path: "internal/session/logout_test.go"},
			ExecutionProtocol: &verification.ExecutionProtocol{
				Repetitions: 1,
				Aggregation: verification.AggregationAllPass,
				Command:     &verification.CommandSpec{Argv: argv},
			},
		}},
	}
	if indep != "" && indep != verification.IndependenceNone {
		acc.EvidenceBindings[0].Basis = &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority}
	}
	c := verification.Contract{TaskID: "TASK-0154", Revision: 1, Acceptances: []verification.Acceptance{acc}}
	f, err := verification.FreezeContract(&c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("freeze executable contract: %v", err)
	}
	return f
}

// vreStoreExecutor wires the adapter against a real fact store so the
// candidateverify authority persists run/evidence rows.
func vreStoreExecutor(t *testing.T, f *vreFixture, delegate taskexecution.AgentExecutor, q *store.Queries) *VerifiedRoleExecutor {
	t.Helper()
	ex, err := NewVerifiedRoleExecutor(VerifiedRoleExecutorConfig{
		Delegate:     delegate,
		Repo:         f.repo,
		WorktreeRoot: t.TempDir(),
		EnvName:      "vre-env",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "vre-fake", Capabilities: agentruntime.NewCapabilitySet()},
		Queries:      q,
		Now:          func() time.Time { return time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return ex
}

// vreRunEvidence executes one VERIFIER role run against the given contract
// and returns the single factual evidence ref plus the parsed run id.
func vreRunEvidence(t *testing.T, q *store.Queries, contract *verification.Frozen, runName string) (taskexecution.TaskExecutionResult, engineering.EvidenceRef, int64) {
	t.Helper()
	f := vreBuildCustom(t, vreVisiblePlan(), contract)
	vreSeedCandidate(t, f)
	ex := vreStoreExecutor(t, f, &scriptedExecutor{steps: []scriptedStep{{run: agentruntime.RunID(runName)}}}, q)
	res, err := ex.Execute(context.Background(), vreRequest(f, "VERIFIER"))
	if err != nil {
		t.Fatalf("verifier execute: %v", err)
	}
	if res.Status != taskexecution.ExecutionCompleted {
		t.Fatalf("status=%s, want completed", res.Status)
	}
	if strings.TrimSpace(res.CandidateRef) == "" {
		t.Fatal("verifier run lost the frozen candidate reference")
	}
	if len(res.EvidenceRefs) != 1 {
		t.Fatalf("want exactly one factual evidence ref, got %+v", res.EvidenceRefs)
	}
	ev := res.EvidenceRefs[0]
	var runID int64
	if _, err := fmt.Sscanf(ev.URI, "tally://verification/run/%d", &runID); err != nil || runID <= 0 {
		t.Fatalf("evidence URI %q is not bound to a persisted verification run", ev.URI)
	}
	if ev.Revision != res.CandidateRef {
		t.Fatalf("evidence revision %q not bound to frozen candidate %q", ev.Revision, res.CandidateRef)
	}
	wantID := engineering.ID(fmt.Sprintf("TASK-0154:%s:verification:%d", RoleVerifier, runID))
	if ev.ID != wantID {
		t.Fatalf("evidence id %q, want %q", ev.ID, wantID)
	}
	return res, ev, runID
}

// Case 5: candidateverify VERIFIED -> factual PASS EvidenceRef bound to the
// persisted verification run and its evidence facts; the verifier decision
// passes on fact, not on the Agent claim.
func TestVerifiedRoleExecutorVerifiedOutcomeProducesFactualPassEvidence(t *testing.T) {
	st := store.TestDB(t)
	contract := vreExecutableContract(t, verification.IndependenceNone, []string{"go", "version"})
	res, ev, runID := vreRunEvidence(t, st.Queries(), contract, "run-ver-pass")

	if ev.Status != engineering.EvidencePass {
		t.Fatalf("evidence status=%s, want pass", ev.Status)
	}
	run, err := st.Queries().GetVerificationRun(context.Background(), store.GetVerificationRunParams{ProjectID: "vre-proj", ID: runID})
	if err != nil {
		t.Fatalf("verification run %d not persisted: %v", runID, err)
	}
	if string(run.Status) != string(verification.RunStatusPass) {
		t.Fatalf("persisted run status=%s, want pass", run.Status)
	}
	if run.ChangesetID != res.CandidateRef {
		t.Fatalf("run changeset %q != evidence candidate %q", run.ChangesetID, res.CandidateRef)
	}
	facts, err := st.Queries().ListVerificationEvidenceForRun(context.Background(), store.ListVerificationEvidenceForRunParams{RunID: runID, ProjectID: "vre-proj"})
	if err != nil {
		t.Fatalf("list run facts: %v", err)
	}
	if len(facts) != 1 || facts[0].BindingID != "VIS-1" || facts[0].Verdict != "pass" || facts[0].TaskID != "TASK-0154" {
		t.Fatalf("persisted facts=%+v, want single pass verdict for VIS-1/TASK-0154", facts)
	}

	d, err := EvidenceInterpreter{}.Interpret(context.Background(), RoleContext{TaskID: "TASK-0154", Role: RoleVerifier, RunID: "run-ver-pass"}, res)
	if err != nil || d.Kind != DecisionPass {
		t.Fatalf("decision=%+v err=%v, want pass on factual evidence", d, err)
	}
}

// Case 6: verification FAILED -> factual FAIL evidence and the deterministic
// repair path (one finding carrying the failing evidence).
func TestVerifiedRoleExecutorFailedOutcomeProducesFailEvidenceAndRepair(t *testing.T) {
	st := store.TestDB(t)
	contract := vreExecutableContract(t, verification.IndependenceNone, []string{"git", "rev-parse", "--verify", "refs/heads/vre-missing-ref"})
	res, ev, runID := vreRunEvidence(t, st.Queries(), contract, "run-ver-fail")

	if ev.Status != engineering.EvidenceFail {
		t.Fatalf("evidence status=%s, want fail", ev.Status)
	}
	run, err := st.Queries().GetVerificationRun(context.Background(), store.GetVerificationRunParams{ProjectID: "vre-proj", ID: runID})
	if err != nil {
		t.Fatalf("verification run %d not persisted: %v", runID, err)
	}
	if string(run.Status) != string(verification.RunStatusFail) {
		t.Fatalf("persisted run status=%s, want fail", run.Status)
	}
	facts, err := st.Queries().ListVerificationEvidenceForRun(context.Background(), store.ListVerificationEvidenceForRunParams{RunID: runID, ProjectID: "vre-proj"})
	if err != nil {
		t.Fatalf("list run facts: %v", err)
	}
	if len(facts) != 1 || facts[0].BindingID != "VIS-1" || facts[0].Verdict != "fail" {
		t.Fatalf("persisted facts=%+v, want single fail verdict for VIS-1", facts)
	}

	d, err := EvidenceInterpreter{}.Interpret(context.Background(), RoleContext{TaskID: "TASK-0154", Role: RoleVerifier, RunID: "run-ver-fail"}, res)
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != DecisionRepair || len(d.Findings) != 1 {
		t.Fatalf("decision=%+v, want repair with exactly one finding", d)
	}
	if len(d.Findings[0].EvidenceRefs) != 1 || d.Findings[0].EvidenceRefs[0].ID != ev.ID {
		t.Fatalf("finding evidence=%+v, want the failing evidence %s", d.Findings[0].EvidenceRefs, ev.ID)
	}
}

// Case 7a: verification NEEDS_REVIEW (effective independence coverage
// incomplete against an unprovenanced implementation) -> NOT_RUN semantics
// with a real persisted run; the evidence can never pass.
func TestVerifiedRoleExecutorNeedsReviewOutcomeNeverPasses(t *testing.T) {
	st := store.TestDB(t)
	contract := vreExecutableContract(t, verification.IndependenceSource, []string{"go", "version"})
	res, ev, runID := vreRunEvidence(t, st.Queries(), contract, "run-ver-review")

	if ev.Status != engineering.EvidenceNotRun {
		t.Fatalf("evidence status=%s, want not_run (needs_review must never pass)", ev.Status)
	}
	run, err := st.Queries().GetVerificationRun(context.Background(), store.GetVerificationRunParams{ProjectID: "vre-proj", ID: runID})
	if err != nil {
		t.Fatalf("verification run %d not persisted: %v", runID, err)
	}
	if string(run.Status) == string(verification.RunStatusPass) {
		t.Fatalf("needs_review run persisted as pass: %+v", run)
	}

	d, err := EvidenceInterpreter{}.Interpret(context.Background(), RoleContext{TaskID: "TASK-0154", Role: RoleVerifier, RunID: "run-ver-review"}, res)
	if err != nil || d.Kind != DecisionBlock {
		t.Fatalf("decision=%+v err=%v, want block on needs_review evidence", d, err)
	}
}
