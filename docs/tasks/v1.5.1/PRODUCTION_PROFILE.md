# Tally v1.5.1 Production Profile

Version: **v1.5.1 — Persistence & Execution Simplification**

## Core boundary

`cmd/v15server` is the v1.5.1 Core composition. It intentionally does **not** construct the legacy PostgreSQL job queue and does not require `DATABASE_URL` to start.

The legacy `cmd/server` remains a brownfield v1.4-compatible composition and may still require PostgreSQL for legacy web/session/job/fact-store capabilities. Its requirements are not the correctness boundary of v1.5.1 Core.

The standalone `cmd/v15server` exposes the Engineering Control Plane persistence/lifecycle surface. It intentionally starts with execution `not_configured`; production embedding may provide the frozen v1.5.0 `TaskHarness` / `AgentExecutor` composition. Dispatch without one must be treated as unavailable rather than as successful execution.

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

## v1.5.0 state migration

The v1.5.0 compatibility `FileRepository` can be imported explicitly on startup:

```text
TALLY_V15_IMPORT_V150_STATE_ROOT=/path/to/v1.5.0/state
```

Migration behavior is deliberately conservative:

- every source `.json` snapshot is decoded and validated before target mutation starts;
- existing target projects are skipped, never overwritten;
- source files are never deleted or rewritten;
- rerunning the import is idempotent;
- invalid/corrupt source input fails closed.

After evidence confirms the imported target is healthy, operators may archive the old source separately. The application does not delete it automatically.

## Scheduling profile

v1.5.1 schedules directly from the persisted Active TaskDAG. `READY → RUNNING` is claimed under the Control Plane mutation lock; Agent/Harness execution occurs outside that lock; finalization reloads the latest snapshot before applying `INTERRUPTED`, `BLOCKED`, or accepted/done state.

The Local Scheduler owns no durable queue, lease, heartbeat, stale-worker takeover or `SKIP LOCKED` correctness primitive. Parallelism is bounded in-process using `ParallelEligible` plus configured maximum concurrency. A persisted `RUNNING` task found after process restart is recovered to `INTERRUPTED`; it is never silently marked `READY` or resumed.

## Network default

`cmd/v15server` binds to `127.0.0.1:8081` by default via `TALLY_V15_ADDR`. This keeps the bare Core API loopback-only until an authenticated outer composition or reverse proxy is intentionally configured.
