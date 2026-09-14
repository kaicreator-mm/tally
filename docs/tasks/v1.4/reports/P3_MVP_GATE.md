# P3_MVP_GATE — MVP Product Gate (P3-10)

**Date:** 2026-09-10 (Beijing), executed by coordinator
**Task:** `docs/tasks/v1.4/P3/P3-10.md`
**Verdict:** **PASS — the v1.4 MVP surface (two harnesses + minimal UI + metrics) is complete and fact-grounded. P4 product validation may proceed.**

## Validation (fresh, executed this gate)

```text
go test -count=1 ./...                      PASS (exit 0; 51 packages ok)
go test -count=1 -tags=integration ./...    PASS (exit 0; 55 packages ok; Postgres 16 via disposable Docker with migrations 0001→0023 applied through the production entry; real codex env available via TALLY_E2E_CODEX_AUTH_FILE)
```

## Acceptance criteria

- **AC1 (Bug and migration E2E pass): PASS** — P3-03 `b113f2f` (bug-fix loop: repro-fails-before → wrong-fix FAIL → repair via new Run/supersession/safe feedback → accepted@2 with basis; escalated leg with typed repair_exhausted intervention) and P3-04 `6aaeccf` (migration loop: fresh/N-1/data-integrity real SQL legs, deterministic 23502 defect, repair+re-verify, VERIFIED_LIMITED gap visible). Both green in this gate's integration run alongside the full P2-10 matrix and all v13 suites.
- **AC2 (Launch-to-verdict without manual failure-copy/retry): PASS** — P3-07 intent intake (`43961bd`) → P3-08 Launch/confirm/run surface (`93db843`) → worker v14 job (P2-09 `ee2478b`) → closed loop (P2-08 `6b4329a`) → verified basis → P3-08/P3-09 detail surfaces. Human-facing failure-copy/retry loop is executed by the engine (classify → repair → re-verify), evidenced by P2-10/P3-03/P3-04 matrices; the human sees escalations/attention (P3-09 `68e1e84`), not retry middleware.
- **AC3 (Metrics generated from facts): PASS** — P3-05 `efff885` (append-only outcome events, trigger-enforced, sqlc-generated, arch-pinned) + P3-06 `4673a11` (deterministic projections: hands-off completion, time-to-accepted, interventions per accepted, 14-day rework; missing data surfaces as named gaps, never zeros; diff-review excluded from inputs by construction).
- **AC4 (No deferred module accidentally activated): PASS** — DEFERRED_NEXT.md scope (Release/Incident/Adaptive Harness/full Guidance Plane/Enterprise sync/full Quality Center/metrics platform) untouched: no new packages outside the task-allowed seams this phase; UI additions confined to the four v1.4 surfaces (Launch, task detail, change detail, attention); policy surface remains two bundles + typed schema (no DSL/framework).

## Incidents

- None this gate. (Historical note carried from P2-11: transient sibling in-flight test failures during parallel waves — all resolved before their landing commits; final suites clean.)

## Decision

**Gate PASS. P3 complete (10/10). P4 (product value experiment) unblocked: P4-01 next (deps P3-10 + A0-05 both satisfied).**
