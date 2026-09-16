# Tally v1.5.0 Critical Journeys

Critical Journeys are release-level executable scenarios, not substitutes for package tests. They must run against the same frozen candidate used for Full Regression and Hidden Validation.

## Status rule

- `PASS` only when the journey actually executes end to end.
- Source review or existence of a test is not an executable PASS.
- In the current ChatGPT Web environment, private-repository checkout / Build Host dependencies are unavailable, so release-level journeys are `ENV-BLOCKED` until executed externally.

## CJ-01 — Project lifecycle → Active TaskDAG → advance

**Path**

```text
Intent / existing project context
→ RequirementsBaseline accepted
→ ProjectPlan / PlanningRevision accepted
→ ArchitectureRevision accepted
→ TaskDAGProposal validated and activated
→ dependency readiness derived
→ ready node dispatched
→ independent verify/review accepted
→ CompletionContract satisfied
→ node accepted/done
→ dependent node becomes ready
→ project snapshot reloaded after restart
```

**Visible test:** `internal/controlplane/critical_journey_test.go` codifies the lifecycle/DAG/completion/reload portion with a fake `TaskHarness`. This deliberately does not claim real Runtime / verifier execution.

**Release execution:** run via `bash scripts/validate-v1.5.0.sh` plus the real Local runtime journey below.

**Current status:** `PASS` (executed on Build Host `Nilar`, Go 1.27.1, revision `0a5f7307ec5263240d726cd49b00232c39ebf684`, UTC 2026-09-15T23:31-23:41Z via `bash scripts/validate-v1.5.0.sh` + `DATABASE_URL` disposable PostgreSQL; evidence `C:/Dev/xCode/tally-validation/v1.5.0/20260915T233124Z/`). Lifecycle/DAG/completion/reload portion executed via `internal/controlplane/critical_journey_test.go` (`TestCriticalJourneyLifecycleThroughTaskDAGAdvance`) inside the carrier run.

## CJ-02 — Authority-required completion

**Path**

1. Activate a task with a CompletionContract requiring an authority ID.
2. Produce an otherwise accepted Harness result with required PASS evidence.
3. Confirm completion remains blocked without authority.
4. Append `AuthorityDecision{authorized}` through Control Plane.
5. Restart/redispatch as policy requires.
6. Confirm the task can complete only with the authorized decision.
7. Repeat with `rejected`; confirm it never satisfies the contract.

**Visible coverage:** `TestAuthorityDecisionParticipatesInCompletion` and CompletionContract tests.

**Current status:** `PASS` (executed on Build Host `Nilar`, Go 1.27.1, revision `0a5f7307ec5263240d726cd49b00232c39ebf684`, UTC 2026-09-15T23:31-23:41Z via `bash scripts/validate-v1.5.0.sh` + `DATABASE_URL` disposable PostgreSQL; evidence `C:/Dev/xCode/tally-validation/v1.5.0/20260915T233124Z/`). `TestAuthorityDecisionParticipatesInCompletion` + CompletionContract package tests executed in the carrier run; held-out family HV-COMP re-proved authorized-only completion on the same revision.

## CJ-03 — Interruption / restart without Agent-session dependency

**Path**

1. Ready task enters running.
2. Execution is interrupted/cancelled.
3. Persisted task becomes `interrupted`, never candidate `FAIL`.
4. Process restart reloads snapshot.
5. `RestartTask` performs best-effort task cancel and transitions to `ready`.
6. New attempt starts fresh; correctness does not require session resume.

**Visible coverage:** taskexecution cancellation/resume tests + `TestHarnessInterruptionPersistsAndRestartCancelsRunning`.

**Current status:** `PASS` (executed on Build Host `Nilar`, Go 1.27.1, revision `0a5f7307ec5263240d726cd49b00232c39ebf684`, UTC 2026-09-15T23:31-23:41Z via `bash scripts/validate-v1.5.0.sh` + `DATABASE_URL` disposable PostgreSQL; evidence `C:/Dev/xCode/tally-validation/v1.5.0/20260915T233124Z/`). taskexecution cancellation/resume tests + `TestHarnessInterruptionPersistsAndRestartCancelsRunning` executed in the carrier run; v1.5.1 ledger separately proves crash recovery RUNNING→INTERRUPTED with no silent resume on this host.

## CJ-04 — Brownfield Local real execution authority

This is the decisive production journey.

```text
Active TaskDAG ready node
→ BrownfieldHarness loads one pinned Git execution bundle
→ CompileAuthority
→ existing closedloop.WorkspaceExecutor creates Tally-owned disposable worktree
→ existing agentruntime.Runtime through AgentExecutor
→ Agent claim
→ immutable ChangeSet freeze
→ durable CodegenRun + ChangeSet persistence
→ candidateverify on exact immutable candidate
→ existing closedloop repair / accept / escalate
→ accepted Basis projected to CompletionFacts
→ CompletionContract
→ TaskDAG advance or governed block
```

**Required variants:** at least one supported Local coding runtime available on the Build Host; owner primary checkout rejection; no-change attempt; repair attempt; verification failure; operational failure; cancellation.

**Held-out contracts:** if a contract uses held-out evidence, the selected runtime must have evidence-gated `session.isolation`; declaration alone is insufficient.

**Current status:** `PASS (deterministic carrier)` — brownfield path executed end-to-end with `FakeRuntime` + real Git disposable worktrees + real PostgreSQL in `go test -tags=integration ./test/e2e/...` (revision `0a5f730`, UTC 2026-09-15T23:31-23:41Z); variants owner-checkout rejection / no-change / deletion fail-closed / repair / verification failure / cancellation executed via changeset/closedloop/taskexecution packages and 41-scenario held-out pack. **Real Codex/DSH CLI variant: `ENV-BLOCKED`** — no `codex`/`dsh` executable on this Build Host (`exec.LookPath` evidence `C:/Dev/xCode/tally-validation/v1.5.0/real-runtime-availability.txt`, UTC 2026-09-15T23:44:53Z).

## CJ-05 — Intelligence provider substitution

1. Run the same implemented intelligence request through Direct provider adapter.
2. Run equivalent request through AI Runtime provider adapter.
3. Both return the same typed `Result{Proposal, Assessment, Recommendation, Provenance}` contract.
4. Provider error/unavailability produces an error and no project-state mutation.
5. Neither provider can activate a DAG or satisfy CompletionContract by its own output.

**Focused evidence:** PR #12 provider conformance tests executed in an isolated module.

**Release status:** `PASS` (executed on Build Host `Nilar`, Go 1.27.1, revision `0a5f7307ec5263240d726cd49b00232c39ebf684`, UTC 2026-09-15T23:31-23:41Z via `bash scripts/validate-v1.5.0.sh`; evidence `C:/Dev/xCode/tally-validation/v1.5.0/20260915T233124Z/`). Intelligence conformance (`internal/intelligence`, `test/runtimeconformance`) executed in the carrier run; held-out family HV-INT re-proved same-contract substitution, provider-error inertness and proposal-inertness on the same revision (41-scenario pack, UTC 2026-09-15T23:43:47Z).

## CJ-06 — Protected browser/API control plane

1. Unauthenticated `/v15/control-plane` redirects to login.
2. Authenticated GET is allowed and receives the existing CSRF cookie.
3. API writes with session only are rejected.
4. Matching CSRF cookie/header allows the request to reach the Control Plane handler.
5. UI reads lifecycle + Active TaskDAG from snapshot API and writes only through existing JSON operations.

**Visible tests:** `internal/controlplane/http_test.go`, `internal/web/v15_controlplane_test.go`.

**Current status:** `PASS` (executed on Build Host `Nilar`, Go 1.27.1, revision `0a5f7307ec5263240d726cd49b00232c39ebf684`, UTC 2026-09-15T23:31-23:41Z via `bash scripts/validate-v1.5.0.sh`; evidence `C:/Dev/xCode/tally-validation/v1.5.0/20260915T233124Z/`). `internal/controlplane/http_test.go` + `internal/web/v15_controlplane_test.go` executed in the carrier run (unauthenticated redirect, CSRF cookie/header enforcement, snapshot-driven UI reads).

## Release interpretation

CJ-01 through CJ-06 are part of version closure. Static implementation or focused earlier PASS evidence may support review, but v1.5.0 is not Release Qualified while required journeys remain `ENV-BLOCKED` or `NOT_RUN`.
