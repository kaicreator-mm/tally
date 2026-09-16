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
| T151-05 | Local Active TaskDAG scheduler | CODE-COMPLETE | bounded local scheduler; startup RUNNING → INTERRUPTED; pinned-Git TaskExecutionRequest factory; capability propagation; no second durable queue |
| T151-06 | Production composition / migration | CODE-COMPLETE | SQLite-first `cmd/v15server`; optional runtime → fail-closed TaskHarness → RuntimeExecutor → LocalScheduler; profile selector; explicit v1.5.0 import |
| T151-07 | Full Regression / Hidden Validation / Critical Journeys | VALIDATED | Full Regression, Legacy Integration/E2E, all nine Critical Journeys and Hidden Validation PASS on the local Build Host (see closure run and ledger below) |
| T151-08 | Version Closure / Release Qualification | PASS | every mandatory gate has revision-pinned PASS evidence on the frozen candidate; no FAIL / NOT_RUN / ENV-BLOCKED remains |

`CODE-COMPLETE` is an implementation state, not a validation state and not Release PASS.

## Handoff closure run (issue #47)

```text
HANDOFF_REVISION=bc69dbe5e5f7a9beeb3bd83e7dcfce3b530d7568 (main, per #47)
FINAL_REVISION =dff92658e5f556c0af58ee3c2ac31b1b0f7cd0b4 (branch fix/v1.5.1-local-validation-defects)
HOST           =Windows 10.0.26200 x64 (MINGW64_NT-10.0-26200, hostname Nilar)
GO             =go1.27.1 windows/amd64
CARRIER        =Local/Build Host; no GitHub Actions; no CI
```

Validation output root (outside the repository): `C:\Dev\xCode\tally-validation\v1.5.1\<UTCSTAMP>\{validation.log,summary.tsv}`. The FINAL_REVISION release-candidate run is `20260915T223314Z`: revision=dff9265, host=Nilar, Go 1.27.1 windows/amd64 — `persistence` PASS, `control-plane` PASS, `architecture-tests` PASS, `core-server-build` PASS, `race-focused` PASS, `full-regression` PASS, `postgres-conformance` PASS; the script's inline legacy-surface check stayed unconfigured in that run because the legacy Integration/E2E was executed as a separate explicit run with `DATABASE_URL` (PASS, `ok tally/test/e2e 295.479s`).

## Current validation ledger

| Gate | Status | Basis / blocker |
|---|---|---|
| Static source / contract review | PASS | source/diffs reviewed against frozen PRD/L2/L3; SQLite-first production topology is `LocalScheduler → Service.Dispatch → TaskHarness → AgentExecutor`; requests use existing pinned Git ExecutionBundle authority and propagate runtime/environment capabilities; PostgreSQL-specific behavior remains adapter-local; legacy DB queue/BrownfieldHarness is not constructed by Core |
| Focused Go tests | PASS | `go test ./internal/persistence/... ./internal/controlplane/... ./test/arch/...` PASS on FINAL_REVISION (validation.log `persistence`, `control-plane`, `architecture-tests`) |
| SQLite adapter integration | PASS | focused persistence suite incl. migrations, contention, restart continuity |
| PostgreSQL adapter conformance | PASS | `TALLY_TEST_POSTGRES_URL=postgres://postgres:postgres@localhost:15432/tally_v151_validation?sslmode=disable` (disposable database on the shared local dev instance); shared conformance fixture PASS after the JSONB→JSON payload-fidelity fix |
| Dispatch/scheduler race tests | PASS | `go test -race ./internal/persistence/... ./internal/controlplane/...` |
| Core server build/startup | PASS | `go build -o bin/v15server ./cmd/v15server`; both lifecycle-only (CJ-151-01) and runtime-configured (CJ-151-04) startups exercised live |
| Runtime-configured TaskDAG execution | PASS | CJ-151-04/05/06 journeys executed against `127.0.0.1:8081` with prepared content repo/workspaces; see Critical Journeys below |
| Full visible regression | PASS | `go test ./...` (whole module) on FINAL_REVISION |
| Legacy Integration/E2E compatibility | PASS | `DATABASE_URL=postgres://postgres:postgres@localhost:15432/tally_legacy_integration?sslmode=disable go test -tags=integration ./test/e2e/...` → `ok tally/test/e2e 295.479s` (disposable database; per-test schema isolation) |
| Independent Review | PASS WITH CONCERNS (both minor, non-blocking) | separate read-only reviewer pass over the frozen candidate; all eight checklist items PASS (pgx confinement, SQLite default, optional Postgres, no DB queue, no second flow/contract, canonical Harness→AgentExecutor boundary, fail-closed preserved, no v1.5.2+ scope). Concern 1 (stale "JSONB" wording in adapter doc) fixed in dff9265^; concern 2 (pre-fix dev databases keep their JSONB columns because `CREATE TABLE IF NOT EXISTS` does not migrate) recorded here as accepted: inserts with `::json` casts remain assignment-compatible and release schema is created fresh as JSON |
| Hidden Validation | PASS | external held-out pack `C:/Dev/vCode/hidden_validation/tally/v1.5.1` (authored 2026-09-16 by the local Build-Host validation agent — a non-implementation-author — black-box from the frozen PRD and the #47 coverage list; fixtures/oracles remain outside the repository) executed via `TALLY_V151_HIDDEN_ROOT=... TALLY_TEST_POSTGRES_URL=<disposable tally_v151_hidden> bash scripts/run-v1.5.1-hidden.sh` on clean detached checkout `0a5f7307ec5263240d726cd49b00232c39ebf684` (code-identical to FINAL_REVISION `dff9265` merged via PR #51; only docs commits separate them) at UTC 2026-09-15T23:53:02Z: **all scenarios PASS** — SQLite shared conformance suite, restart continuity, corrupt-payload-at-rest load failure, append-only immutability, transaction rollback, dual-connection interleaved writes, durable claim before execution, duplicate/stale dispatch refusal, claim-without-verifier fail-closed-to-blocked, crash recovery RUNNING-to-INTERRUPTED, bounded parallel batch, unsafe task-identity rejection, capability-mismatch fail-closed before agent start, FileRepository import idempotence + corrupt-source fail-before-write, PostgreSQL shared conformance parity, and the negative proof that `cmd/v15server` has no `tally/internal/jobs` dependency; sanitized evidence `C:/Dev/xCode/tally-validation/v1.5.1-hidden/20260915T235302Z/` |
| Critical Journeys | PASS | all nine journeys below have local evidence (live journeys + shared-conformance/architecture-guard tests); none ENV-BLOCKED |
| Release Qualification | PASS | every mandatory gate (focused tests, build, race, full regression, SQLite journeys, production LocalScheduler - TaskHarness - AgentExecutor reachability, cancellation/restart semantics, PostgreSQL conformance, legacy Integration/E2E, Independent Review, all nine Critical Journeys, Hidden Validation) has PASS evidence on the frozen candidate lineage FINAL_REVISION `dff9265` / merged `0a5f730`; no gate remains FAIL / NOT_RUN / ENV-BLOCKED |

## Local / Build Host validation

Run from a clean, frozen checkout of the candidate revision:

```bash
bash scripts/validate-v1.5.1.sh
```

For PostgreSQL parity, provide a disposable test database:

```bash
TALLY_TEST_POSTGRES_URL='postgres://...' bash scripts/validate-v1.5.1.sh
```

For runtime-configured scheduler reachability, prepare one workspace per Task ID under a common root and start the Core with an explicit revision, for example:

```bash
TALLY_V15_RUNTIME=fake \
TALLY_V15_CONTENT_REPO=/path/to/content-repo \
TALLY_V15_BASE_REVISION=<revision> \
TALLY_V15_WORKSPACE_ROOT=/path/to/prepared-workspaces \
go run ./cmd/v15server
```

Then invoke `POST /api/v15/projects/{project}/scheduler/run-once` (or `/scheduler/run`) against a project whose Active TaskDAG has READY work. The first successful ExecutionBundle resolution pins the scheduler instance to the exact observed commit. In the SQLite-first Core profile, a completed Agent claim without an independent verifier must become explicit `BLOCKED` with reason `independent verification/review workflow unavailable`; it must never be promoted to accepted/done.

For the external held-out pack:

```bash
TALLY_V151_HIDDEN_ROOT=/outside/repo/v1.5.1-hidden \
  bash scripts/run-v1.5.1-hidden.sh
```

A missing environment prerequisite must remain `ENV-BLOCKED`; it is never equivalent to PASS.

## Critical Journeys

CJ-151-01 — **Fresh default startup without PostgreSQL**: PASS — started `cmd/v15server` with no persistence/runtime environment variables (cwd outside the repo); health reported `persistence=sqlite`, `execution=disabled`, `verification=not_configured`; default SQLite files created at `./tally-state/v1.5/tally.db`; no PostgreSQL connection attempted (no DSN configured, no pgx activity possible in the Core composition — see Independent Review item 1).

CJ-151-02 — **Persist/restart Active TaskDAG**: PASS — journey project seeded over HTTP, tasks executed to persisted terminal states; server process killed; restart on the same SQLite file reloaded identical project/DAG state with readiness preserved (`running` and `ready` states observed post-restart before recovery), then explicit recovery applied (CJ-151-06). Additionally covered by `TestLifecyclePersistsAndReloadsReadiness` and SQLite restart-continuity tests.

CJ-151-03 — **v1.5.0 state migration**: PASS — a valid v1.5.0 FileRepository state root (materialized by the repository's own control-plane code) was imported into an empty SQLite target: `imported=1 skipped_existing=0`; project identity and lifecycle continuity verified via `GET /control-plane` (requirements revision, DAG revision, READY node preserved). Rerun against the same target: `imported=0 skipped_existing=1` (idempotent, existing target skipped, never overwritten). Corrupt input (truncated source JSON) on a fresh target: startup failed with `v1.5.0 snapshot import failed ... unexpected end of JSON input`; the target DB contained no imported project (subsequent `GET` → 404), proving the import fails before target writes.

CJ-151-04 — **Production LocalScheduler → TaskHarness → AgentExecutor reachability**: PASS — with `TALLY_V15_RUNTIME=fake`, content repo, base revision and prepared task workspaces, health reported `execution=local_scheduler+harness+agent_executor` and `verification=fail_closed_unavailable`; `POST /scheduler/recover` → 200; READY task was durably claimed `running` (verified via control-plane snapshot mid-run with a blocking runtime) and actually reached the v1.5.0 AgentExecutor path through `ExecutionOnlyHarness`; the completed Agent claim was persisted as `blocked` with reason `independent verification/review workflow unavailable`, never accepted/done. Failure path also exercised: killing the agent process persisted `blocked` with `agent runtime reported failure; no verification fact produced`.

CJ-151-05 — **Independent parallel siblings**: PASS — with `TALLY_V15_SCHEDULER_MAX_CONCURRENT=2`, one `run-once` dispatched two `ParallelEligible` READY siblings in a single batch (batch size 2 in the response); both executed against the same pinned commit and each kept its own persisted terminal state (no overwrite). Actual concurrent harness execution, batch bounding and mutation-lock release during execution are pinned by `TestDispatchDoesNotHoldProjectMutationLockDuringHarnessRun`, `TestLocalSchedulerBoundsParallelBatchAndLeavesRemainingReady`, and live: a concurrent authority mutation for the second task returned 200 in ~7 ms while the first task's Agent run was still executing.

CJ-151-06 — **Cancellation and restart recovery**: PASS — with a deliberately blocking codex runtime, the crash-killed server left the task durably persisted `running`; restart + `POST /scheduler/recover` converted it to `interrupted`; no implicit resume occurred (readiness did not return the task to `ready`); `POST /tasks/{task}/restart` (explicit restart policy) returned it to `ready`, after which it executed again. Startup recovery without silent resume is additionally pinned by `TestLocalSchedulerStartupRecoveryPersistsInterrupted`.

CJ-151-07 — **Optional PostgreSQL parity**: PASS — shared persistence conformance fixture run against a disposable PostgreSQL database with the explicitly selected adapter (`TALLY_TEST_POSTGRES_URL`); externally observable persistence behavior (round-trip fidelity, mutable update, append-only immutability) matches the SQLite TEXT behavior after the JSONB→JSON fix.

CJ-151-08 — **No default DB queue dependency**: PASS — every live journey above ran `cmd/v15server` with no `DATABASE_URL`; no `jobs.Queue`, lease/heartbeat, `SKIP LOCKED`, stale-worker takeover or PostgreSQL-backed BrownfieldHarness exists in the Core composition (architecture guards `TestV151CoreComposition*` plus Independent Review items 1/4).

CJ-151-09 — **Pinned request authority and capability enforcement**: PASS — configured movable Git ref `main`; after the first executed request pinned the observed commit, `main` was moved (task artifacts removed at the new head) and the second `run-once` still loaded the first observed commit and executed successfully (proving later requests use the pin, and that a rejected/moved authority cannot silently redirect execution). Live negative paths: project identity mismatch (content repo manifest declaring another project) → `422 frozen execution project mismatch: snapshot=journey-project bundle=other-project` before any execution; unsafe workspace Task ID (`../escape` DAG node) → request build rejected (`unsafe for prepared workspace path`) before workspace use, no workspace directory created, task never claimed; unsupported runtime capability (`model.select`) → `422 agentruntime: runtime 未声明必需能力` through the existing negotiation boundary; unsupported environment capability (`cluster.gpu`) → `422 taskexecution: missing environment capabilities: [cluster.gpu]`; Requirements / accepted Plan / Architecture context present in `TaskExecutionRequest.EngineeringContext` (pinned by `TestGitTaskRequestFactoryBuildsPinnedV150Request`, enforced live by `TestGitTaskRequestFactoryRejectsMissingEngineeringContext` behavior).

## Defects found and repaired during this closure (one concern per commit, branch `fix/v1.5.1-local-validation-defects`)

| Commit | Defect | Classification |
|---|---|---|
| cab2665 | go.mod missing modernc.org/sqlite indirect requirements — every build-host command failed with `updates to go.mod needed` (web-side handoff could not run Go tooling) | v1.5.1-owned |
| 4c68623 | `runstore_codegen_test.go` referenced non-existent `CodegenRunRequest.Attempt` field — package did not compile | v1.5.1-owned |
| 3d15dbb | `ExecutionOnlyHarness` relied on implicit zero `Accepted`; architecture guard requires the fail-closed decision to be explicit. Same commit: PostgreSQL adapter stored document payloads as `JSONB`, which normalizes JSON text and broke the shared round-trip-fidelity conformance; switched to text-preserving `JSON` | v1.5.1-owned |
| bc4b0c3 | web handlers test composition never mounted the v1.5 Control Plane, so the `/v15/control-plane` nav entry 404ed under `TestNavLinksResolveToRealRoutes`; test server now mirrors production `cmd/server` wiring | v1.5-series test composition |
| 6ea6baa | validation script built `v15server(.exe)` into the repo root, dirtying the worktree mid-run so export candidate-identity tests refused HEAD export; binary now goes to ignored `bin/` | v1.5.1-owned (validation carrier) |
| dff9265 | independent-review follow-up: stale "JSONB" wording in postgres adapter doc comment | documentation |

## Hidden Validation contract

The held-out pack must live outside the implementation repository and should cover at least: normal persistence lifecycle, invalid/corrupt persistence data, immutable append-only records, transaction rollback, SQLite contention/restart, concurrent sibling completion, stale/duplicate dispatch, pinned request authority/project drift, capability mismatch, unsafe workspace identity, cancellation, startup recovery, optional PostgreSQL parity, migration idempotence/corruption handling, and negative checks proving queue-specific database semantics are not Core invariants.

Only sanitized status/evidence may be copied into release records. Hidden fixtures/oracles remain outside the repository.

## Release qualification rule

Do **not** tag or publish v1.5.1 while any mandatory gate is `FAIL`, `NOT_RUN`, or `ENV-BLOCKED`. After Local/Build Host validation, independent review, Hidden Validation and Critical Journeys all have acceptable evidence, update this ledger with revision-pinned result locations and then perform Release Qualification. PR merge alone is explicitly insufficient.

Current state: every mandatory gate — including Hidden Validation — has PASS evidence on the frozen candidate lineage (FINAL_REVISION `dff9265`, merged `0a5f730`, hidden-validation run on clean detached `0a5f730` at UTC 2026-09-15T23:53:02Z). v1.5.1 Release Qualification: **PASS**. The held-out pack lives outside the repository at `C:/Dev/vCode/hidden_validation/tally/v1.5.1/` (manifest.json + run.sh + gotests/); its provenance is recorded in the Hidden Validation ledger row above, and a future independently-authored pack can re-run the same wrapper as an additional gate. Tag/baseline creation remains a release-policy action separate from this ledger..