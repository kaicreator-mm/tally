# A0-05 — Architecture readiness gate: Execution Report

- **TASK:** A0-05
- **STATUS:** DONE
- **Gate verdict:** **READY_TO_IMPLEMENT**
- **Gate-time HEAD:** `6946bb5` (clean tree; all A0 evidence applied). The A0-05 commit itself adds docs only; broad implementation starts at/after it.
- **Audited source baseline (unchanged):** `979a55ad3ba5e514bb80f0e012aa36e0e6577eba` (L2 archive; v1.4 55/55 DONE, Final Gate `PASS_WITH_KNOWN_LIMITATIONS`)
- **Integration-proof source baseline:** `3c1e983` — verified at gate time that only docs, `scripts/preflight/**` and `test/arch/v141_decision_lock_guards_test.go` changed since; no `internal/`/`cmd/`/migrations delta (see §4 #1).
- **Date:** 2026-09-12
- **Execution/Review model:** LOW / HIGH (per task pack: authority-boundary release gate → HIGH review required)
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)

## 1. Goal and deliverable

Issue the pre-implementation verdict and freeze the brownfield baseline.

Delivered: this report — the A0 evidence review (A0-01..A0-04), fresh gate-time
execution of the arch guard suite and the deterministic preflight, the
`READY_TO_IMPLEMENT` verdict with its evidence basis, the recorded baseline
commits, and the register of unresolved L3 items (all deliberately open, owned,
and non-blocking for P0 start).

## 2. Gate evidence review (A0-01..A0-04)

Dependency check: per `TASK_DAG.md`, **A0-05 ← A0-02, A0-03, A0-04**; all three
carry `## Execution status → Status: DONE` addenda and DONE reports. A0-01 is
DONE via its report (see §6 item 2 for a docs-consistency note on its task file).

| Task | Evidence | Review result |
|---|---|---|
| A0-01 authority map | Report `reports/A0-01_source_to_prd_authority_map.md`: PRD §1–§40 every section has exactly one owner seam or is narrative/deferred; source facts F1–F17 verified in-tree; amendments A-01…A-10 mapped; 14 duplicate-architecture risks (R-*) registered; DL-01…DL-20 cross-referenced. | Consistent with DECISION_LOCKS and PRD_AMENDMENTS; no contradiction found. Spot re-checks at gate time: `runtimes/` = codex+fake only (F11 holds); `contract.Bind(` in `cmd/`+`internal/` hits only `internal/taskstate/basis_test.go` (test caller — F4 "no production call site" holds). |
| A0-02 decision locks frozen | `DECISION_LOCKS.md` FROZEN header (baseline `310d961`), BI-01/BI-02 binding interpretations, 5 SPIKE_REQUIRED open seams with owning tasks, machine enforcement index. | Frozen content intact (not touched by A0-03/04/05); enforcement index names A0-04 as guard-gap owner — A0-04 delivered. |
| A0-03 preflight | `scripts/preflight` staged tool; classification deterministic (dependency/network failures never reported as code regression); BASELINE_OK connected + offline + DB-gated integration at HEAD `3c1e983`. | Supersedes the `BASELINE_NOT_EXECUTED_NETWORK_BLOCKED` note in SOURCE_BASELINE.md §Validation preflight (see §6 item 1). Source identity since the integration proof verified (§4 #1). |
| A0-04 guard tests | `test/arch/v141_decision_lock_guards_test.go`: 8 guards for the v1.4.1-specific readings (DL-02 protocol uniqueness; BI-01 governance-field exclusion on schema+Parse+no DB activation flag; BI-02 outcome kind-set closure), negative-proven by probes + permanent negative tests; review-enforced remainder documented. | Guards enforce exactly the frozen readings; no existing test weakened or deleted. |

Cross-consistency: 0/57 task files are missing a `DECISION_LOCKS` reference
(gate-time re-count, `grep -L`); STATUS.md 4/57 tally matches the four DONE
reports; DAG edges match the task files' `Depends on` fields.

## 3. Gate-time validation executed (exact commands and results)

Windows / Git Bash, HEAD `6946bb5` (clean), Go `go1.27.1 windows/amd64`.

| # | Command | Result |
|---|---|---|
| 1 | `go test -count=1 ./test/arch/...` | PASS — `ok tally/test/arch 1.055s` (fresh, uncached; includes the four v1.4-era guard files and the 8 v1.4.1 A0-04 guards) |
| 2 | `go run ./scripts/preflight` | PASS — exit 0, `BASELINE_OK`, `is_code_regression=false`; gating checks all PASS: toolchain, modules (1.58s), build (4.07s), vet (1.83s), unit_arch (`ok tally/test/arch 1.043s`), unit_internal (`ok tally/internal/workspace`, 1m0.7s); `integration`: SKIP — NOT_RUN(reason: `-integration` not requested; DB-gated baseline is opt-in) |
| 3 | `git diff --stat 3c1e983..HEAD` | PASS — only docs, `scripts/preflight/**`, `test/arch/v141_decision_lock_guards_test.go`; no production source changed since the A0-03 integration-proof baseline |
| 4 | `ls runtimes/` | PASS — `codex`, `fake` only; no `dsh` (F11 / DL-15 expectation: DSH is new work, not existing fact) |
| 5 | `grep -rn "contract.Bind(" cmd/ internal/` | PASS — single hit is `internal/taskstate/basis_test.go:134` (test); F4 holds |
| 6 | `grep -L "DECISION_LOCKS" docs/tasks/v1.4.1/tasks/*/*.md` | PASS — empty (57/57 reference the locks) |
| 7 | `git diff --check` (before commit) | PASS — clean |

**Integration re-run at gate time: NOT_RUN(reason: opt-in DB-gated; A0-03 proved
`BASELINE_OK` including `go test -tags=integration ./...` at `3c1e983`, and
command #3 shows the production source is byte-identical since — the delta is
docs + new preflight tool + new arch guards only).** This is evidence-based
carry-forward, not a fresh claim.

## 4. Baseline vs introduced failures

- **Baseline failures observed at gate time:** none. Arch suite and preflight
  (build/vet/unit) are green at `6946bb5`; A0-03 recorded the DB-gated
  integration suite green at the identical source.
- **Introduced failures by A0-05:** none. This task changed documentation only.

## 5. Gate verdict and its basis

**READY_TO_IMPLEMENT.**

Basis (each element independently checkable):

1. Authority chain complete: Final PRD (amendments incorporated) → frozen
   DECISION_LOCKS (DL-01…DL-20 + BI-01/BI-02 + tension rule + change control) →
   source-to-PRD owner map covering every PRD section → per-task contracts.
2. Executable baseline green at gate time (arch guards + full unit baseline via
   preflight, exit 0) and the DB-gated integration baseline proven green at the
   identical source by A0-03.
3. Duplicate-architecture defenses in place before any implementation: 8 new
   v1.4.1 arch guards (A0-04) on top of the v1.4 guard suite, negative-proven;
   14 R-* risk registers with named existing single authorities (A0-01).
4. Dependency state deterministic: preflight distinguishes
   DEPENDENCY_BLOCKED / TOOLCHAIN_BLOCKED / SOURCE_FAILURE; module cache proven
   complete (offline run, A0-03 run 6).
5. All DAG prerequisites of A0-05 are DONE; no task file contradicts a frozen
   lock (tension readings BI-01/BI-02 are frozen into the locks document every
   task must read first).

No blocker exists. Exact blocker IDs: none.

## 6. Unresolved L3 items (recorded; none blocks P0 start)

1. **SOURCE_BASELINE.md staleness (docs-only).** Its "Validation preflight"
   section still records `BASELINE_NOT_EXECUTED_NETWORK_BLOCKED`; A0-03
   superseded this with `BASELINE_OK` (unit + integration, offline-proven).
   `SOURCE_BASELINE.md` is outside this task's Allowed Paths
   (`docs/tasks/v1.4.1/reports/**`), so the stale text is recorded here instead
   of edited. Any task reading the baseline must read A0-03's report as the
   current fact; the executable authority is `scripts/preflight`, which any
   agent can re-run.
2. **A0-01 task-file addendum missing (docs-only).** The A0-01 task file has no
   `## Execution status` appendix (unlike A0-02/03/04); its DONE state is
   recorded in its report and in STATUS.md's 4/57 (now 5/57) tally. Outside
   Allowed Paths; recorded here. The instruction "each task file has an
   Execution status appendix" should be satisfied for A0-01 by an authorized
   docs pass.
3. **Five SPIKE_REQUIRED open seams** (deliberately unfrozen per
   DECISION_LOCKS.md — invention is forbidden, each is fixed only by its owning
   task): FrozenIntent exact path/schema → P0-01; acceptance-guard placement →
   P1-06; DSH headless transport → P4-03 (spike gate first); rule replay
   storage/query → P3-04; L3 local Git integration service API → P5-01. Any task
   needing one of these before its owner lands stops `BLOCKED`.
4. **Review-enforced remainder (from A0-04 §2 and the enforcement index):**
   wiring-level DL-02 facts (one runtime session per attempt; adapters route
   through `agentruntime.Runtime`), the BI-01 semantic face (runtime consumes
   the exact pinned-commit policy blob), and BI-02 full generality (a parallel
   event store avoiding the `event_kind` column name) are not machine-checkable
   until their owning tasks land; locks DL-06/DL-10/DL-11/DL-13/DL-19 remain
   review-enforced.
5. **Known environment sensitivities (carried from A0-03/A0-04, non-blocking):**
   preflight classifier vocabulary is Go-version sensitive (unmatched wording
   degrades fail-safe to `DEPENDENCY_BLOCKED`, never a false regression and
   never a false PASS); real-codex/DeepSeek e2e tests skip without
   `TALLY_E2E_CODEX_AUTH_FILE` (live-LLM paths owned by P4); pre-existing gofmt
   drift in several `test/arch` files and `scripts/v14_trial/tally/rows.go`
   (untouched, `go test`/`go vet` unaffected).

## 7. L3 completion evidence (per pack)

- **Exact sources/tests read:** all four A0 reports; `DECISION_LOCKS.md`,
  `PRD_AMENDMENTS.md`, `SOURCE_BASELINE.md`, `STATUS.md`, `TASK_DAG.md`,
  `MODEL_ASSIGNMENT.md` (A0-05 row), the five A0 task files (addenda), spot
  re-checks of `runtimes/`, `internal/taskstate/basis_test.go`; executed
  `test/arch` suite and `scripts/preflight`.
- **External reference actually used:** none (pack: "No external implementation
  is required").
- **Narrowest valid seam:** the deliverable is the verdict report itself under
  the task's sole Allowed Path `docs/tasks/v1.4.1/reports/**`; no code, no API,
  no test, no authority document was touched.
- **Negative tests added:** none — this gate introduces no behavior ("new
  behavior needs focused tests" is N/A); the guards it relies on carry their own
  negative-proven tests (A0-04).
- **Known uncertainty / version sensitivity:** item 6.5 above; the verdict's
  executable claims are reproducible via the commands in §3.
- **Decision Locks not bypassed:** confirmed — see §8.

## 8. Decision Locks

**DL_TOUCHED: no.** `DECISION_LOCKS.md` was not edited; no lock weakened,
routed around, or silently reinterpreted. DL-17 respected: nothing under
`docs/tasks/v1.4.1/hidden/` was read, listed, or summarized.

## 9. Remaining risks

- The verdict is a point-in-time fact bound to gate-time HEAD `6946bb5` + this
  docs-only delta; any future source change re-enters the loop guarded by
  preflight + arch guards, so drift is detectable rather than silent.
- Items §6.1/§6.2 are documentation inconsistencies outside this task's allowed
  paths; if left unaddressed they waste a future agent's read, but cannot cause
  an incorrect implementation decision because the executable checks and the
  frozen locks take precedence (authority order, AGENTS §2).
- Integration baseline is carried forward by source-identity argument
  (§3 command #3) rather than re-executed at gate time; any doubt is resolved by
  re-running `go run ./scripts/preflight -integration` with a `DATABASE_URL`.
