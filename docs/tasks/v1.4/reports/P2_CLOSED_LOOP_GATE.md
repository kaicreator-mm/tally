# P2_CLOSED_LOOP_GATE — Closed Loop Gate (P2-11)

**Date:** 2026-09-09 (Beijing), executed by coordinator
**Task:** `docs/tasks/v1.4/P2/P2-11.md`
**Verdict:** **PASS — the core product loop (Execute→Verify→Repair→Accept/Escalate) is correct enough to authorize MVP harness/UI work. P3 unblocked.**

## Validation (fresh, executed this gate)

```text
go test -count=1 ./internal/closedloop/...                       PASS
go test -count=1 -tags=integration ./test/e2e/...                PASS (exit 0; v13 regression suites + v14 codex trust [real codex env] + v14 closed-loop matrix; Postgres 16 via disposable Docker, per-test schema + goose)
go test -count=1 ./...                                           PASS (exit 0; 50 packages ok)
```

## Acceptance criteria

- **AC1 (closed-loop E2E pass): PASS** — P2-10 `8e045f4` matrix (8 cases: first-pass accept / fail→repair→accept with frozen-base materialization + rework supersession / repair exhaustion→escalate / provider failure→infrastructure escalation / environment ERROR→bounded re-verify→accept with ZERO supersessions [FAIL≠ERROR end-to-end] / held-out failure→requires_human_judgment with leak-scan clean instruction / repair-time exposure→exposure_compromise despite re-verified PASS / owner worktree clean in every case). All against the production composition (NewLoop + candidateverify + runstore sinks) on real PostgreSQL.
- **AC2 (inner vs outer retry distinguishable): PASS** — inner budget = ExecutionPolicy.Retry.MaxIterations on the codegen runner (v1.3 semantics, P2-09 `ee2478b`); outer budget = positional RepairPolicy (default 2, hard bounds [1,10], P2-07 `d964dcf`). Identities distinct: outer attempts mint new RunIDs (RunIDFor(task,attempt), parent-linked via EscalatedFrom), new ChangeSet ids, and append-only reason=rework supersession facts; the matrix asserts 1+2 attempts = 3 runs / 2 facts / ancestors chain [B,A] — never conflatable with inner iterations.
- **AC3 (no held-out leakage false-positive assurance): PASS** — P2-06 `1009b50` exposure e2e (21/21) + matrix held-out case: implementation-visible instruction scanned clean (no store root/ref/digests/binding id); deliberate repair-time disclosure converts re-verified PASS into exposure_compromise escalation with nil basis — lineage-monotonic exposure accounting cannot preserve full assurance after crossing.
- **AC4 (no v1.3 regression): PASS** — v13 e2e suites (standard+holdout, holdout-fail, exposure, changeset-supersession) green in this gate's e2e run; worker v13 tests green with zero v13.go changes (P2-09 touched main.go +8 lines registration only); full unit suite exit 0.

## Incidents during the gate

- First full-unit attempt showed 3 failures — all in `internal/web/handlers/v14_launch*_test.go`, the concurrently in-flight P3-08 agent's untracked files (not P2 scope). Final clean rerun after P3-08 landed: exit 0, 50 ok. Recorded for honesty; no P2 package was ever red.
- P3-08 (parallel wave) returned BLOCKED on the shared-hotspot route registration; coordinator applied the declared one-line `RegisterV14Routes` wiring in server.go — the exact minimal edit the agent named, no more.

## Decision

**Gate PASS. P2 complete (11/11). P3 MVP harness + surface wave (P3-01/02/05 now unblocked; P3-09 in flight; P3-03/04/06 after P3-01/02/05) may proceed.**
