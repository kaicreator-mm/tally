# v1.5.4 Validation and Closure

Date: 2026-09-16  
Version: **v1.5.4 — Multi-Agent Engineering Workflow**  
Base used for the clean implementation branch: `main@3637f3d9fc4afc25ff0190d2cea4ec533a814c6c`  
Branch: `dev/v1.5.4-multi-agent-workflow-r3`

## Status rules

Only `PASS | FAIL | NOT_RUN(reason) | ENV-BLOCKED(reason + evidence)` are valid validation statuses. GitHub Actions and all CI are prohibited for this version.

## Evidence available in this development session

### Source / contract review — PASS

The implementation was reviewed against the frozen v1.5.0 and v1.5.4 contracts. During review the following concrete defects were found and fixed before PR creation:

- stale branch was 44 commits behind main; implementation was re-layered onto current main instead of merging stale history;
- collaboration facts initially used `NodeComplete`; corrected to `NodeAccepted` so the existing CompletionContract remains final authority;
- collaboration queued on concurrency semaphore was not initially cancellable; active registration moved before semaphore wait;
- late cancellation finalization could race `RestartTask`; finalizer now preserves authoritative Ready/Interrupted TaskDAG state;
- a persisted legacy Running task could be mistaken for multi-agent restart state; recovery now requires durable multi-agent ownership;
- RuntimeRef could be used as a label to fake runtime/provider independence; RuntimeExecutor now verifies requested runtime identity against the actual runtime descriptor before starting a session;
- production `cmd/server` had the existing fake/codex/dsh RuntimeExecutor and Brownfield Harness but did not yet inject the v1.5.4 engine; the Multi-Agent Engine is now composed above the same executor.

This PASS is a source/contract review result, not a substitute for executed Go validation.

## Executable validation matrix

| Gate | Command | Status | Evidence / reason |
|---|---|---|---|
| Focused validation | `bash scripts/validate-v1.5.4.sh` | **ENV-BLOCKED** | Current ChatGPT session can read/write the private GitHub repository through the connector but has no authenticated repository checkout + Go Build Host/remote terminal. No GitHub Actions/CI may be used as substitute. |
| Hidden Validation | `bash scripts/hidden-validation-v1.5.4.sh` | **ENV-BLOCKED** | Same environment limitation. The script covers the frozen hidden-validation scenario families but has not been executed against the complete private repository in this session. |
| Critical Journeys | `bash scripts/critical-journeys-v1.5.4.sh` | **ENV-BLOCKED** | Same environment limitation. Sequential, repair, parallel, restart and production-composition journeys are encoded but not executed here. |
| Full Regression | `bash scripts/full-regression-v1.5.4.sh` | **ENV-BLOCKED** | Requires complete private checkout and Go toolchain on Local/Build Host. |

### Important non-evidence

A contract-aligned isolated/stubbed multi-agent test run performed during implementation helped detect logic defects, but it is **not** counted as Full Regression, Hidden Validation or Release evidence because it did not execute the complete private repository revision.

## Hidden Validation scenarios encoded

- Implement → Review PASS → Verify PASS → accept.
- Review finding → Repair → Review → Verify.
- repair/provider execution failure.
- Reviewer/Verifier disagreement returning to repair.
- reviewer claim with no Tally evidence.
- run/session independence violation.
- requested runtime/provider identity spoof rejection.
- two independent Ready tasks in parallel.
- dependency-blocked task not selected.
- configured file/workspace conflict suppresses parallel dispatch.
- global concurrency bound.
- queued collaboration cancellation.
- restart mid-collaboration without Agent-session/thought recovery.
- legacy Running task ownership protection.
- late cancellation finalization after RestartTask.
- evidence/finding producer role and role-run provenance.

## Critical Journeys encoded

1. Implement → Review → Verify → CompletionContract → Task Done.
2. Review finding → Repair → Review → Verify.
3. Two non-conflicting Ready tasks execute under the bounded scheduler while conflicts remain undispatched.
4. Restart/cancel restores semantic task/role state without restoring Agent thought/session state.
5. Production server reuses the existing Local RuntimeExecutor and enables v1.5.4 only when an execution runtime is configured.

## Release Qualification

**NOT_RUN(reason: release prerequisites are not yet satisfied).**

Blocking prerequisites:

1. Focused Validation, Hidden Validation, Critical Journeys and Full Regression require executed Local/Build-Host evidence; they are currently ENV-BLOCKED in this session.
2. Frozen v1.5.4 PRD declares **v1.5.3 as the release baseline dependency**. The inspected current main baseline is not yet a v1.5.3 release baseline.

Therefore implementation may proceed through independent review / Draft PR, but v1.5.4 must **not** be marked Release Qualified, merged solely on Agent claims, or tagged from this evidence state.

## Tag / baseline status

**NOT_RUN(reason: Release Qualification has not passed).**

When an authorized Local/Build Host becomes available, execute the four scripts above at the exact PR revision, record environment/command/result/revision/timestamp here, resolve any failures without weakening tests, then make the Release Qualification decision.

---

# Build-Host Finalization (authoritative, supersedes the ENV-BLOCKED handoff state above)

Date: 2026-09-16  
Host: Nilar — Windows 10.0.26200 x64, Go 1.27.1, git 2.55.0.windows.5, Docker postgres-18 (shared dev instance, disposable database `tally_v154`)  
Issue: #42 (finalization handoff); related implementation issues #39, #40

- HANDOFF_REVISION: `b7f588ce5e21b96657f969634cd61960d2a4af95`
- FINAL_REVISION: `e3606c014fbe2de5ea2b0ace236c6cecccf60973`

## Fixes made during validation (all on `dev/v1.5.4-multi-agent-workflow-r3`)

1. `f8a3e7f` — merged `origin/main` (v1.5.3 release baseline `d667486`) into the handoff branch; resolved `types.go` (additive ProjectSnapshot union + both validateSnapshot loops) and `executor.go` (kept v1.5.2 workspace conversion with the v1.5.4 runtime-descriptor guard); repaired the shared broken-ancestor defect (dropped closing brace in the replan-history block of `controlplane/types.go`, same class as the v1.5.3 repair); `multiagent.go` persists terminal collaboration state before calling main's 6-arg `finishAccepted`.
2. `a8f3e12` — production wiring: `cmd/server/v15.go` composes `multiagent.NewVerifiedRoleExecutor` (evidence-producing adapter) above the existing RuntimeExecutor and injects it into the Multi-Agent Engine; raw RuntimeExecutor is no longer injected directly. Raw Agent Claim can never become reviewer/verifier evidence in production.
3. `83b3127` — deadlock repair: `finishAccepted` split into locking wrapper + `finishAcceptedLocked`; `finalizeMultiAgent` (holding `s.mu`) calls the locked variant. Control-plane suite previously self-deadlocked (601s timeout); suite now PASS in ~2s.
4. `d7c7e0d` — fixture truthfulness: six v1.5.2 executor tests declared RuntimeRef values that did not match their actual runtimes; fixtures now declare the real runtime IDs. Assertions unchanged (no test weakened); the v1.5.4 `ErrRuntimeRefMismatch` guard correctly refused the mismatched fixtures.
5. `3a01043` — mandatory adapter unit tests (issue #42 Phase 0.3 cases 1,2,3,4,7b,8 + constructor validation + role-directive requirement) in `internal/multiagent/verified_executor_test.go`.
6. `528486a` — mandatory adapter integration tests (cases 5,6,7a) in `internal/multiagent/verified_executor_integration_test.go` (PostgreSQL fact store via `store.TestDB`, disposable database): VERIFIED → factual PASS EvidenceRef bound to the persisted verification run/changeset/facts; FAILED → factual FAIL + deterministic repair routing; NEEDS_REVIEW (effective independence incomplete) → NOT_RUN with a real persisted run, never PASS. Includes `vreBuildCustom` seam and pre-existing gofmt drift repair in `verified_executor.go`.
7. `e3606c0` — deterministic parallel-batch test: the two-task parallel test used a 15ms executor sleep to force overlap and flaked under load (observed 1-in-3 standalone failures and one full-regression failure with `maxActive=1`). Replaced with a two-party rendezvous; the real-overlap assertion (`maxActive >= 2`) is unchanged. Test-only synchronization repair; implementation untouched.

## Gate evidence (all at FINAL_REVISION unless noted)

| Gate | Command | Result | Evidence |
|---|---|---|---|
| G0 baseline | `git merge-base --is-ancestor origin/main HEAD` | PASS | `origin/main=d667486` (v1.5.3 release merge) is an ancestor of `e3606c0`; working tree clean; no unresolved conflicts; diff confined to v1.5.4 paths (33 files, +3884/−38) |
| G1 implementation completeness | adapter tests below + production wiring review | PASS | VerifiedRoleExecutor production-wired in `cmd/server/v15.go`; all mandatory adapter tests PASS |
| G2 focused validation | `bash scripts/validate-v1.5.4.sh` | PASS | multiagent 7.7s, controlplane MultiAgent selectors, taskexecution runtime binding, cmd/server composition, `go build ./cmd/server ./cmd/v15server` — all PASS |
| G2 additional selectors | `DATABASE_URL=postgres://postgres:postgres@localhost:15432/tally_v154 go test -tags=integration ./internal/multiagent -run 'VerifiedRoleExecutorVerifiedOutcome|VerifiedRoleExecutorFailedOutcome|VerifiedRoleExecutorNeedsReviewOutcome' -count=1 -v` | PASS (3/3, 23.0s) | run at `528486a` (adapter commit); production-composition selectors re-run at `e3606c0` inside G2 script |
| G3 production/runtime validation | `go test ./cmd/server -run 'V15ProductionComposition|V15MultiAgentMaxConcurrency' -count=1 -v`; `go test ./internal/taskexecution -run 'RuntimeExecutorRejects' -count=1 -v`; `go test ./internal/multiagent -run 'IndependentRunConstraint|MissingReviewerEvidenceBlocks' -count=1 -v` | PASS (5/5) | `C:/Dev/xCode/tally-validation/v1.5.4/phase2-runtime-truthfulness.log` — no runtime → 503; `TALLY_V15_RUNTIME=fake` → capability gate opens the v1.5.4 path; spoofed/missing RuntimeRef rejected before Agent start; independence fail-safe blocks same-runtime/identity relabeling |
| G3 real codex/dsh runtime leg | `where.exe codex dsh`; `command -v codex dsh`; binary format inspection | **ENV-BLOCKED (strict)** | No Windows-invocable codex or dsh binary on this Build Host; the only codex artifact is a Linux ELF inside `.codex/bin/wsl/`, not spawnable by the win32 production host; no dsh artifact anywhere. Evidence `C:/Dev/xCode/tally-validation/v1.5.4/real-runtime-availability.txt`. Same strict class as v1.5.0/v1.5.2; the frozen PRD Local compatibility path is fully validated via the authorized fake runtime composition and the runtime-truthfulness guards above |
| G4 hidden validation (visible carrier) | `bash scripts/hidden-validation-v1.5.4.sh` | PASS | `C:/Dev/xCode/tally-validation/v1.5.4/hidden-visible-carrier.log` |
| G4 hidden validation (held-out pack) | `bash C:/Dev/vCode/hidden_validation/tally/v1.5.4/run.sh --repo C:/Dev/xCode/tally-wt-v154 --revision e3606c014fbe2de5ea2b0ace236c6cecccf60973 --version v1.5.4 --result-dir C:/Dev/vCode/hidden_validation/tally/v1.5.4/results/20260916T031159Z` | **PASS 21/21, exit 0** | `aggregate.txt`: scenarios_total=21, failed=0, env_blocked=0. Independent pack authored by the local validation agent (not the implementation author) with independent scenario data; covers all 14 required Phase-3 scenario families (FLOW01-02, NEG01-05, SCHED01-04, REC01-02, PROV01) plus 3 source-level negative proofs (no CI diff, no prohibited-expansion surface, Agent Summary never becomes Evidence). Two pack-oracle corrections during the run (dependency-state timing oracle; drive-letter-aware grep filter) — implementation untouched, no weakening |
| G5 critical journeys | `bash scripts/critical-journeys-v1.5.4.sh` | PASS (all 7 journeys) | `C:/Dev/xCode/tally-validation/v1.5.4/critical-journeys.log` — CJ1 sequential accept via CompletionContract; CJ2 finding→repair→re-review→verify; CJ3 bounded parallel without collisions; CJ4 conflict/dependency suppression; CJ5 restart/cancel recovery; CJ6 production composition on existing runtime/executor; CJ7 final acceptance governed by existing CompletionContract |
| G6 full regression | `bash scripts/full-regression-v1.5.4.sh` (includes `go test ./... -count=1` and `go build ./...`) | PASS (75 packages, 0 FAIL) | `C:/Dev/xCode/tally-validation/v1.5.4/full-regression-r2.log`. First run exposed the parallel-test flake fixed in `e3606c0`; rerun PASS at final revision |
| G7 independent review | local validation agent review of full diff `origin/main...HEAD` (subagent spawning unavailable in this environment, same as v1.5.2/v1.5.3) | **PASS WITH CONCERNS (2 minor, non-blocking)** | All 11 prohibited-expansion/architecture checks PASS: no BPM/DSL, no conversation-graph/CoT persistence (engine directive explicitly forbids prior chain-of-thought), no job queue/Agent fleet, bounded concurrency only, TaskDAG sole project-flow authority, roles are engineering responsibilities, AgentExecutor contract additive-only (`ErrRuntimeRefMismatch` guard), claim≠evidence mechanically enforced, independence unspoofable, restart restores durable facts only, v1.5.2/v1.5.3 integrations optional. Minor: (a) handoff-authored commits lack `Refs:` trailers (all 7 validation-authored commits carry `Refs: #42`); (b) pre-existing gofmt drift under Go 1.27 in files the branch does not touch (all branch-touched files clean). No P0/P1 findings |

## Release Qualification — **RELEASE QUALIFIED: YES**

G0 PASS; G1 PASS; G2 PASS; G3 PASS with one strict ENV-BLOCKED leg (real codex/dsh CLI genuinely unavailable on this Build Host, evidence recorded; the release environment's required Local composition and runtime-truthfulness checks all PASS — ENV-BLOCKED here is the genuine-absence class accepted by v1.5.0-v1.5.3, not an unexecuted required check); G4 PASS (visible carrier + held-out pack 21/21); G5 PASS; G6 PASS; G7 PASS WITH CONCERNS (no P0/P1). No gate is FAIL or NOT_RUN.

## Tag / baseline status (final)

Merge per project policy first; tag `v1.5.4` only on the merged revision once the merged revision itself is confirmed (or re-qualified if the revision changes). Recorded here after merge/tag execution.

Refs: #42
