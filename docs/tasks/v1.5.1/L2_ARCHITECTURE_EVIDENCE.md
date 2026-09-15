# Tally v1.5.1 — L2 Architecture Evidence

Status: **EXECUTABLE**  
Base revision: `d0c0f72390fb04d5891cfc82937f977729488f25`  
Authority: frozen v1.5 architecture + v1.5.0 contracts + v1.5.1 frozen PRD.

## 1. Compatibility verdict

**GO.** v1.5.1 does not conflict with v1.5.0. It consumes the v1.5.0 Foundation contracts and owns only the persistence/execution-infrastructure slice explicitly deferred by v1.5.0.

Frozen and unchanged:
- `ActiveTaskDAG` remains the single Project Flow authority.
- `EngineeringNode` / `CompletionContract` semantics remain unchanged.
- `internal/agentruntime.Runtime/Session` remains the only Agent Runtime protocol.
- `AgentExecutor`, `TaskExecutionRequest`, `TaskExecutionResult`, `ExecutionEnvironment`, `WorkspaceRef` and interruption/restart semantics are consumed, not redefined.
- Engineering Harness remains the authority for verification/repair/acceptance.

## 2. Brownfield evidence

### 2.1 Existing bounded seams are sufficient

`internal/controlplane.Repository` already separates the v1.5 Control Plane from its compatibility file repository. `Service` depends on that interface, not PostgreSQL.

`engineering.ActiveTaskDAG` already owns deterministic readiness via `RefreshReadiness`, `ReadyNodes` and `ParallelReady`.

`controlplane.Dispatch` already validates READY state, persists RUNNING before execution, delegates through Engineering Harness/AgentExecutor, persists interruption, and evaluates CompletionContract before DONE.

Therefore v1.5.1 should evolve these seams instead of introducing another Project Flow, runtime lifecycle or workflow engine.

### 2.2 PostgreSQL is still a brownfield default outside the new seam

Current `cmd/server/main.go` defaults `DATABASE_URL` to PostgreSQL and exits if `store.Open` fails. `web.Deps`, sessions, legacy execution sources, knowledge/indexing and old diagnose paths are wired directly to `*store.Queries`.

`internal/store` is sqlc/pgx-specific. `internal/jobs` directly imports `pgx/pgtype` and defines correctness around `FOR UPDATE SKIP LOCKED`, lease ownership, heartbeat and stale takeover.

These are migration targets, not reasons to change v1.5.0 domain contracts.

### 2.3 Concurrency defect to remove for local scheduling

`controlplane.Service.Dispatch` currently holds the project mutation mutex across the entire Harness/Agent execution. This serializes same-process task execution even when multiple DAG nodes are parallel-eligible.

v1.5.1 must narrow locking to state-claim and result-commit critical sections. The Agent/Harness run occurs outside the mutation lock. This is an implementation correction; TaskDAG and AgentExecutor contracts stay unchanged.

## 3. Persistence boundary

Introduce database-independent ports under `internal/persistence`:

- `ProjectStore`
- `RequirementStore`
- `PlanningStore`
- `TaskGraphStore`
- `TaskContractStore`
- `RunStore`
- `FactStore`
- `EvidenceStore`
- `ActionAuditStore`
- `OutcomeStore`
- `TransactionBoundary`

Ports use Tally-owned identifiers/domain values or typed persistence records. They must not expose `pgx`, `pgtype`, JSONB-specific operators, DB-generated identifiers or DB clock values.

The existing aggregate `controlplane.Repository` remains a v1.5.0 compatibility seam. v1.5.1 adapters may satisfy both the aggregate repository and the finer persistence ports while brownfield callers are migrated incrementally.

## 4. Storage profiles

### 4.1 SQLite default

Default profile is SQLite, in-process and file-backed. A fresh v1.5 Control Plane must start without PostgreSQL.

Rules:
- application generates durable IDs;
- application supplies Clock values in UTC;
- JSON is canonical serialized content, not a query-semantic requirement;
- transactions are explicit through `TransactionBoundary`;
- SQLite busy handling is bounded;
- foreign keys are enabled;
- WAL may be used as an adapter optimization, never as Core correctness;
- immutable facts/evidence/audit/outcomes are append-only at Core level and reinforced by adapter constraints.

Use a pure-Go SQLite driver so default installation does not require CGO/toolchain-specific native SQLite linkage.

### 4.2 PostgreSQL optional

PostgreSQL remains an optional adapter and may retain pgx/sqlc/JSONB/index/trigger optimizations inside adapter-owned paths. It must conform to the same observable persistence contract as SQLite.

Existing v1.4 PostgreSQL-backed features may remain brownfield compatibility paths when outside v1.5.1 ownership, but they cannot be required for v1.5 Core lifecycle correctness.

## 5. Transaction semantics

A transaction boundary groups a logical engineering state mutation and its durable audit/evidence linkage. Core decides what must be atomic; adapter decides SQL mechanics.

No correctness rule may depend on:
- DB `now()`;
- serial/identity-generated domain IDs;
- trigger-only invariants;
- JSONB behavior;
- PostgreSQL lock primitives.

Duplicate application identities fail deterministically. Append-only records reject mutation/replacement.

## 6. Migration strategy

Use the brownfield sequence:

`retain → introduce ports → add SQLite → isolate PostgreSQL → migrate v1.5 production wiring → remove default queue dependency`.

The compatibility file repository is retained until SQLite persistence has visible conformance/restart coverage; it is not the v1.5.1 default target.

Legacy DB queue code is retained only for legacy features that still call it. No destructive deletion is required to prove that v1.5 scheduling no longer depends on it.

## 7. Application-local scheduler

Canonical default execution path:

`persisted Active TaskDAG → restart recovery/readiness refresh → bounded local scheduler → Service.Dispatch → Engineering Harness → AgentExecutor`.

Scheduler rules:
- only READY nodes may be claimed;
- non-parallel nodes execute alone relative to the same scheduling batch;
- parallel-eligible nodes may run up to configured bound;
- no lease, heartbeat, stale-owner takeover or DB row claim exists;
- restart converts persisted RUNNING to INTERRUPTED before readiness scheduling;
- dependency readiness is recomputed from persisted DAG state;
- interruption does not imply completion or automatic acceptance.

## 8. Dispatch concurrency design

Split dispatch into three phases:
1. **Claim under lock**: reload latest snapshot, validate READY, transition to RUNNING, append audit, save.
2. **Execute without project mutation lock**: run Engineering Harness/AgentExecutor.
3. **Commit under lock**: reload latest snapshot, confirm task is still RUNNING for this execution, merge outcome, evaluate CompletionContract, persist transition/audit.

This permits bounded parallel tasks without moving correctness into a database queue.

## 9. Composition and compatibility

v1.5 persistence profile selection is explicit:
- default: `sqlite`;
- optional: `postgres`;
- compatibility file store may exist only as migration/test support, not the default release profile.

Brownfield v1.4 web surfaces that still require PostgreSQL are not allowed to force PostgreSQL into the v1.5 Core composition. Composition must separate v1.5 Core dependencies from optional legacy dependencies.

## 10. Conformance and release evidence

One conformance suite must be reusable for SQLite and PostgreSQL and cover CRUD/round-trip, transaction rollback, duplicate identity, append-only mutation rejection, restart, DAG state persistence and interrupted state.

Scheduler coverage must include ready/blocked, bounded concurrency, non-parallel exclusion, restart recovery, interruption and explicit absence of lease/heartbeat correctness.

Release Qualification requires explicit evidence that:
- SQLite default starts without PostgreSQL;
- PostgreSQL remains optional and conformant;
- no Core package imports pgx/pgtype;
- default v1.5 execution does not depend on `internal/jobs`, `SKIP LOCKED`, leases or heartbeats;
- v1.5.0 lifecycle and completion authority remain intact.

## 11. Validation carrier

For v1.5.1, GitHub Actions and all CI systems are prohibited. Validation is executed explicitly on Local / Build Host and recorded with command, environment, result, evidence, revision and timestamp. Unexecuted validation is `NOT_RUN(...)` or `ENV-BLOCKED(...)`, never PASS.
