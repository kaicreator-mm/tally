# Tally v1.5.1 Production Profile

Version: **v1.5.1 — Persistence & Execution Simplification**

## Core boundary

`cmd/v15server` is the v1.5.1 Core composition. It intentionally does **not** construct the legacy PostgreSQL job queue and does not require `DATABASE_URL` to start.

The legacy `cmd/server` remains a brownfield v1.4-compatible composition and may still require PostgreSQL for legacy web/session/job/fact-store capabilities. Its requirements are not the correctness boundary of v1.5.1 Core.

The standalone `cmd/v15server` always exposes the Engineering Control Plane persistence/lifecycle surface. With no `TALLY_V15_RUNTIME`, execution is explicitly disabled and no FakeRuntime is selected silently. When a supported runtime is configured, the same SQLite-first Core composes the frozen v1.5.0 `AgentExecutor` through `RuntimeExecutor`, a pinned-Git `TaskRequestFactory`, and the v1.5.1 application-local `LocalScheduler`.

This production path contains no durable DB job queue. `Active TaskDAG` remains the scheduling state authority.

## Persistence profiles

Default profile:

```text
TALLY_V15_PERSISTENCE=sqlite
TALLY_V15_SQLITE_PATH=./tally-state/v1.5/tally.db
```

`TALLY_V15_PERSISTENCE` may be omitted; SQLite is selected by default. The SQLite adapter owns SQLite SQL, PRAGMAs, migrations and writer serialization. Core consumes only persistence ports / `controlplane.Repository`.

Optional PostgreSQL profile:

```text
TALLY_V15_PERSISTENCE=postgres
TALLY_V15_POSTGRES_URL=postgres://...
```

PostgreSQL is opened only when explicitly selected. A missing PostgreSQL DSN is a startup error. PostgreSQL-specific SQL/JSONB/TIMESTAMPTZ behavior is confined to the adapter and is not a Core scheduling invariant.

## Local execution profile

Execution is opt-in. Configure a frozen runtime and execution authority explicitly:

```text
TALLY_V15_RUNTIME=codex                 # fake | codex | dsh
TALLY_V15_CONTENT_REPO=/srv/project     # Git content repository
TALLY_V15_BASE_REVISION=<revision>      # resolved once per request to an observed commit
TALLY_V15_WORKSPACE_ROOT=/srv/tally-workspaces
TALLY_V15_SCHEDULER_MAX_CONCURRENT=2    # optional; default 2
TALLY_V15_ENV_TARGET=local              # optional; default local
```

For each scheduled task, v1.5.1 expects an already prepared local workspace at:

```text
<TALLY_V15_WORKSPACE_ROOT>/<TASK-ID>
```

Workspace creation/Git authority automation is deliberately not invented here; those mechanics are outside v1.5.1. The request factory loads the existing same-source `ExecutionBundle` from `TALLY_V15_CONTENT_REPO` and `TALLY_V15_BASE_REVISION`, checks project/task identity, pins the request to the observed source commit and existing TaskContract revision, then emits the already-frozen v1.5.0 `TaskExecutionRequest` contract.

Runtime-specific optional settings remain the existing Local compatibility settings, including `TALLY_V15_CODEX_BIN`, `TALLY_V15_DSH_BIN`, `TALLY_V15_MODEL`, `TALLY_V15_DSH_PROVIDER`, and `TALLY_V15_SESSION_ISOLATION_VERIFIED`.

The scheduler control surface is:

```text
POST /api/v15/projects/{project}/scheduler/recover
POST /api/v15/projects/{project}/scheduler/run-once
POST /api/v15/projects/{project}/scheduler/run
```

`recover` converts persisted `RUNNING` work to `INTERRUPTED`. `run-once` executes one deterministic bounded READY batch. `run` drains READY batches until the project is idle or execution reports an error.

## Verification degradation boundary

The default SQLite-first Core does **not** pull the PostgreSQL-backed legacy `BrownfieldHarness` into its composition. This is intentional: v1.5.1 removes PostgreSQL as a default correctness/execution dependency rather than hiding it behind startup configuration.

When the Local `AgentExecutor` completes but no independent verification/review workflow is composed, Agent completion remains only an implementation claim. `Service.Dispatch` persists the task as `BLOCKED` and returns `ErrWorkflowUnavailable`; it never promotes the task to accepted/done. The legacy brownfield composition may still provide the full existing closed loop when its legacy PostgreSQL fact-store dependencies are explicitly available.

`GET /healthz` reports the boundary explicitly. Typical values are:

```text
execution=disabled
verification=not_configured
```

or, with the runtime/scheduler profile configured:

```text
execution=local_scheduler+agent_executor
verification=legacy_brownfield_not_composed
```

## v1.5.0 state migration

The v1.5.0 compatibility `FileRepository` can be imported explicitly on startup:

```text
TALLY_V15_IMPORT_V150_STATE_ROOT=/path/to/v1.5.0/state
```

Migration behavior is deliberately conservative:

- every source `.json` snapshot is decoded and validated before target mutation starts;
- only the latest valid immutable generation for each source project is selected;
- existing target projects are skipped, never overwritten;
- source files are never deleted or rewritten;
- rerunning the import is idempotent;
- invalid/corrupt source input fails closed before target writes.

After evidence confirms the imported target is healthy, operators may archive the old source separately. The application does not delete it automatically.

## Scheduling profile

v1.5.1 schedules directly from the persisted Active TaskDAG. `READY → RUNNING` is claimed under the Control Plane mutation lock; Agent/Harness execution occurs outside that lock; finalization reloads the latest snapshot before applying `INTERRUPTED`, `BLOCKED`, or accepted/done state.

The Local Scheduler owns no durable queue, lease, heartbeat, stale-worker takeover or `SKIP LOCKED` correctness primitive. Parallelism is bounded in-process using `ParallelEligible` plus configured maximum concurrency. A persisted `RUNNING` task found after process restart is recovered to `INTERRUPTED`; it is never silently marked `READY` or resumed.

## Network default

`cmd/v15server` binds to `127.0.0.1:8081` by default via `TALLY_V15_ADDR`. This keeps the bare Core API loopback-only until an authenticated outer composition or reverse proxy is intentionally configured.