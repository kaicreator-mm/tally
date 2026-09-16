# Tally v1.5.1 Production Profile

Version: **v1.5.1 — Persistence & Execution Simplification**

## Core boundary

`cmd/v15server` is the v1.5.1 Core composition. It intentionally does **not** construct the legacy PostgreSQL job queue and does not require `DATABASE_URL` to start.

The legacy `cmd/server` remains a brownfield v1.4-compatible composition and may still require PostgreSQL for legacy web/session/job/fact-store capabilities. Its requirements are not the correctness boundary of v1.5.1 Core.

The standalone `cmd/v15server` always exposes the Engineering Control Plane persistence/lifecycle surface. With no `TALLY_V15_RUNTIME`, execution is explicitly disabled and no FakeRuntime is selected silently. When a supported runtime is configured, the same SQLite-first Core composes the canonical production topology:

```text
Active TaskDAG
  → LocalScheduler
  → Service.Dispatch
  → TaskHarness (fail-closed execution bridge)
  → v1.5.0 AgentExecutor / RuntimeExecutor
  → existing Agent Runtime
```

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

Execution is opt-in. Configure a runtime and execution authority explicitly:

```text
TALLY_V15_RUNTIME=codex                 # fake | codex | dsh
TALLY_V15_CONTENT_REPO=/srv/project     # Git content repository
TALLY_V15_BASE_REVISION=<revision>      # resolved once, then pinned to observed commit
TALLY_V15_WORKSPACE_ROOT=/srv/tally-workspaces
TALLY_V15_SCHEDULER_MAX_CONCURRENT=2    # optional; default 2
TALLY_V15_ENV_TARGET=local              # optional; default local
```

For each scheduled task, v1.5.1 expects an already prepared local workspace at:

```text
<TALLY_V15_WORKSPACE_ROOT>/<TASK-ID>
```

Workspace creation/Git authority automation is deliberately not invented here; those mechanics are outside v1.5.1. The request factory loads the existing same-source `ExecutionBundle`, verifies project/task identity, and on the first successful request pins the configured revision expression to the exact observed source commit. All later requests from that scheduler instance load from the same commit even if the original branch/tag moves.

The factory emits the already-frozen v1.5.0 `TaskExecutionRequest`: TaskContract revision/digest, runtime/environment requirements, local workspace reference, execution policy and implementation instruction are carried through existing contracts. Runtime/environment capability requirements from the Active TaskDAG are propagated into `RequiredCapabilities`; unsupported capability combinations therefore fail through the existing v1.5.0 negotiation path instead of being reduced to prompt text.

Task IDs containing path separators/traversal forms are rejected before they are used as prepared-workspace paths.

Runtime-specific optional settings remain the existing Local compatibility settings, including `TALLY_V15_CODEX_BIN`, `TALLY_V15_DSH_BIN`, `TALLY_V15_MODEL`, `TALLY_V15_DSH_PROVIDER`, and `TALLY_V15_SESSION_ISOLATION_VERIFIED`.

The scheduler control surface is:

```text
POST /api/v15/projects/{project}/scheduler/recover
POST /api/v15/projects/{project}/scheduler/run-once
POST /api/v15/projects/{project}/scheduler/run
```

`recover` converts persisted `RUNNING` work to `INTERRUPTED`. `run-once` executes one deterministic bounded READY batch. `run` performs startup recovery and then drains READY batches until the project is idle or a batch reports an execution/build failure.

## Verification degradation boundary

The default SQLite-first Core does **not** pull the PostgreSQL-backed legacy `BrownfieldHarness` into its composition. This is intentional: v1.5.1 removes PostgreSQL as a default correctness/execution dependency rather than hiding it behind startup configuration.

Instead, runtime-configured Core uses `ExecutionOnlyHarness`, a fail-closed `TaskHarness` implementation around the frozen v1.5.0 `AgentExecutor`. It preserves the canonical Harness → AgentExecutor boundary but never manufactures independent verification facts. A completed Agent claim is therefore returned as non-accepted; `Service.Dispatch` persists the task as `BLOCKED` with reason `independent verification/review workflow unavailable`. Failed execution also blocks, and interruption remains `INTERRUPTED`.

The legacy brownfield composition may still provide the full existing closed loop when its legacy PostgreSQL fact-store dependencies are explicitly available. That compatibility path is not required for v1.5.1 Core scheduling/Agent execution correctness.

`GET /healthz` reports the boundary explicitly. Typical values are:

```text
execution=disabled
verification=not_configured
```

or, with the runtime/scheduler profile configured:

```text
execution=local_scheduler+harness+agent_executor
verification=fail_closed_unavailable
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