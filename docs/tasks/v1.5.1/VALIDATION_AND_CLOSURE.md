# Tally v1.5.1 Validation & Closure Ledger

Version: **v1.5.1 — Persistence & Execution Simplification**

This file records the development closure state without converting unexecuted checks into PASS. Validation carrier is Local / Build Host only. GitHub Actions and CI are prohibited for this version.

## Development task status

| Task | Scope | Development state | Evidence |
|---|---|---|---|
| T151-01 | DB-neutral persistence ports / Core boundary | CODE-COMPLETE | merged persistence ports and architecture guard |
| T151-02 | SQLite default adapter | CODE-COMPLETE | SQLite adapter, migrations, conformance/contention tests |
| T151-03 | PostgreSQL optional adapter | CODE-COMPLETE | optional adapter plus shared persistence conformance fixture |
| T151-04 | Dispatch concurrency seam | CODE-COMPLETE | claim → unlocked execute → reload/latest commit; duplicate-claim guard |
| T151-05 | Local Active TaskDAG scheduler | CODE-COMPLETE | bounded local scheduler; startup RUNNING → INTERRUPTED; pinned-Git TaskExecutionRequest factory; no second durable queue |
| T151-06 | Production composition / migration | CODE-COMPLETE | SQLite-first `cmd/v15server`; optional runtime → RuntimeExecutor → LocalScheduler; profile selector; explicit v1.5.0 import |
| T151-07 | Full Regression / Hidden Validation / Critical Journeys | ENV-BLOCKED | current ChatGPT Web environment cannot materialize/run the private repository/build host or external held-out pack |
| T151-08 | Version Closure / Release Qualification | NOT_RUN | required empirical gates above are not yet PASS |

`CODE-COMPLETE` is an implementation state, not a validation state and not Release PASS.

## Current validation ledger

| Gate | Status | Basis / blocker |
|---|---|---|
| Static source / contract review | PASS | source/diffs reviewed against frozen PRD/L2/L3; SQLite-first production composition now contains runtime-selectable `RuntimeExecutor + LocalScheduler`; requests are built from the existing pinned Git ExecutionBundle; PostgreSQL-specific behavior remains adapter-local; legacy DB queue/BrownfieldHarness is not constructed by Core |
| Focused Go tests | ENV-BLOCKED | no executable checkout / Go build host available in the current ChatGPT Web tool environment; new request-factory and architecture guards are authored but not claimed PASS |
| SQLite adapter integration | ENV-BLOCKED | same executable-environment blocker |
| PostgreSQL adapter conformance | ENV-BLOCKED | requires executable checkout and explicit `TALLY_TEST_POSTGRES_URL` |
| Dispatch/scheduler race tests | ENV-BLOCKED | requires Local/Build Host and `go test -race` |
| Core server build/startup | ENV-BLOCKED | requires Local/Build Host; must test both default SQLite lifecycle-only startup and runtime-configured scheduler/AgentExecutor startup |
| Runtime-configured TaskDAG execution | ENV-BLOCKED | requires prepared content repo/workspaces and Local runtime; must prove READY task reaches AgentExecutor through scheduler with bounded concurrency |
| Full visible regression | ENV-BLOCKED | requires Local/Build Host |
| Legacy Integration/E2E compatibility | ENV-BLOCKED | requires Local/Build Host and legacy integration prerequisites such as `DATABASE_URL` where applicable |
| Independent Review | NOT_RUN | no independent reviewer execution/evidence has been attached to this candidate |
| Hidden Validation | ENV-BLOCKED | external held-out pack/root is not available in the current environment |
| Critical Journeys | ENV-BLOCKED | executable environment unavailable; journey definitions are below |
| Release Qualification | NOT_RUN | cannot start until all mandatory predecessor gates have acceptable evidence |

## Local / Build Host validation

Run from a clean, frozen checkout of the candidate revision:

```bash
bash scripts/validate-v1.5.1.sh
```

For PostgreSQL parity, provide a disposable test database:

```bash
TALLY_TEST_POSTGRES_URL='postgres://...' bash scripts/validate-v1.5.1.sh
```

For runtime-configured scheduler reachability, prepare one workspace per Task ID under a common root and start the Core with an explicit frozen revision, for example:

```bash
TALLY_V15_RUNTIME=fake \
TALLY_V15_CONTENT_REPO=/path/to/content-repo \
TALLY_V15_BASE_REVISION=<frozen-revision> \
TALLY_V15_WORKSPACE_ROOT=/path/to/prepared-workspaces \
go run ./cmd/v15server
```

Then invoke `POST /api/v15/projects/{project}/scheduler/run-once` (or `/scheduler/run`) against a project whose Active TaskDAG has READY work. In the SQLite-first Core profile, a completed Agent claim without the optional independent verification workflow must become explicit `BLOCKED` / `ErrWorkflowUnavailable`; it must never be promoted to accepted/done.

For the external held-out pack:

```bash
TALLY_V151_HIDDEN_ROOT=/outside/repo/v1.5.1-hidden \
  bash scripts/run-v1.5.1-hidden.sh
```

A missing environment prerequisite must remain `ENV-BLOCKED`; it is never equivalent to PASS.

## Critical Journeys

CJ-151-01 — **Fresh default startup without PostgreSQL**: start `cmd/v15server` with no persistence/runtime environment variables; verify SQLite file creation, health reports `persistence=sqlite` and `execution=disabled`, and no PostgreSQL connection is attempted.

CJ-151-02 — **Persist/restart Active TaskDAG**: create a lifecycle snapshot and Active TaskDAG in SQLite, stop the process, restart it, and verify the same durable project/DAG state is loaded with readiness preserved/recomputed correctly.

CJ-151-03 — **v1.5.0 state migration**: import a valid FileRepository state root into an empty SQLite target; verify project identity/state continuity; rerun and verify existing target state is skipped rather than overwritten; inject a corrupt source and verify import fails before writes.

CJ-151-04 — **Production LocalScheduler → AgentExecutor reachability**: configure `TALLY_V15_RUNTIME`, content repo, frozen base revision and prepared task workspaces; health must report `execution=local_scheduler+agent_executor`; a READY task must be claimed RUNNING and actually reach the v1.5.0 AgentExecutor path. Without independent verification composition, successful Agent completion must be persisted as BLOCKED with explicit workflow-unavailable evidence rather than accepted/done.

CJ-151-05 — **Independent parallel siblings**: two `ParallelEligible` READY siblings must both reach Agent execution concurrently under configured capacity; completion/failure of one must not overwrite the other's latest persisted state.

CJ-151-06 — **Cancellation and restart recovery**: cancellation/deadline during execution persists `INTERRUPTED`; process-start recovery converts previously persisted `RUNNING` to `INTERRUPTED`; no implicit resume occurs; explicit restart policy is required to return work to READY.

CJ-151-07 — **Optional PostgreSQL parity**: explicitly select PostgreSQL, run the shared persistence conformance fixture, and verify externally observable persistence behavior matches SQLite for the neutral contract.

CJ-151-08 — **No default DB queue dependency**: start the v1.5 Core path without `DATABASE_URL`; verify no `jobs.Queue`, lease/heartbeat, `SKIP LOCKED`, stale-worker takeover mechanism, or PostgreSQL-backed BrownfieldHarness is required for TaskDAG scheduling/AgentExecutor reachability.

CJ-151-09 — **Pinned request authority**: move the named Git ref after server startup/request preparation and verify each execution request remains pinned to the exact commit observed by `ExecutionBundle`; inject a project/task identity mismatch and verify dispatch is rejected before Agent execution.

## Hidden Validation contract

The held-out pack must live outside the implementation repository and should cover at least: normal persistence lifecycle, invalid/corrupt persistence data, immutable append-only records, transaction rollback, SQLite contention/restart, concurrent sibling completion, stale/duplicate dispatch, pinned request authority/project drift, cancellation, startup recovery, optional PostgreSQL parity, migration idempotence/corruption handling, and negative checks proving queue-specific database semantics are not Core invariants.

Only sanitized status/evidence may be copied into release records. Hidden fixtures/oracles remain outside the repository.

## Release qualification rule

Do **not** tag or publish v1.5.1 while any mandatory gate is `FAIL`, `NOT_RUN`, or `ENV-BLOCKED`. After Local/Build Host validation, independent review, Hidden Validation and Critical Journeys all have acceptable evidence, update this ledger with revision-pinned result locations and then perform Release Qualification. PR merge alone is explicitly insufficient.