# Tally v1.5.1 — L3 Implementation Evidence

Each task follows `Tests → Contract → Implementation → Failure Handling → Reference`.

## T151-01 Persistence contracts

### Tests
- compile-time adapter/interface assertions;
- architecture scan rejects `pgx`/`pgtype` imports from v1.5 Core persistence boundary;
- duplicate identity and append-only mutation expectations are specified in conformance tests.

### Contract
Expose small consumer-owned ports for Project, Requirements, Planning, TaskGraph, TaskContract, Run, Fact, Evidence, Action/Audit, Outcome and transaction boundary. IDs and semantic timestamps are application-owned.

### Implementation
Prefer existing `engineering` values where they already own semantics. Persistence DTOs are typed and use `json.RawMessage` only for deliberately opaque serialized payloads; do not introduce `map[string]any` Core state. Keep `controlplane.Repository` as the v1.5.0 aggregate compatibility seam while adapters converge on the new ports.

### Failure Handling
Return typed/wrapped errors that work with `errors.Is/As`; never string-match database error messages in Core. Duplicate IDs and immutable replacement are deterministic contract failures.

### Reference
Frozen v1.5 §13; v1.5.1 Persistence Ports/Core Correctness; current `internal/controlplane.Repository`; current `internal/store/db.go` as negative PG-specific evidence.

---

## T151-02 SQLite adapter

### Tests
- clean temporary database open/migrate;
- CRUD/round-trip for each port;
- transaction commit and rollback;
- restart/reopen preserves state;
- duplicate application ID rejected;
- append-only records cannot be updated/replaced;
- concurrent readers + bounded writers/contention;
- Active TaskDAG and INTERRUPTED state survive reopen.

### Contract
SQLite is the default storage profile and must require no PostgreSQL service. Core-visible semantics must match PostgreSQL.

### Implementation
Use `database/sql` with a pure-Go SQLite driver. Enable foreign keys; use bounded busy timeout. WAL is permitted as an adapter optimization. Schema stores canonical JSON/text/blobs without relying on SQLite JSON query semantics for correctness. Use application-supplied UTC timestamps and IDs.

### Failure Handling
Busy/locked errors are retried only within a small bounded adapter policy or surfaced; never spin forever. Schema migration failure aborts startup with explicit evidence. Corruption is surfaced, not silently regenerated.

### Reference
v1.5.1 default storage and required SQLite validation cases.

---

## T151-03 PostgreSQL adapter

### Tests
Run the same conformance suite used for SQLite, plus migration parity and transaction behavior.

### Contract
PostgreSQL is optional. pgx/pgtype, DB time, JSONB optimizations, sequences, triggers and lock primitives never appear in Core contracts.

### Implementation
Reuse existing PostgreSQL facilities where possible. New adapter-owned code may use pgx/sqlc. Do not rewrite unrelated v1.4 persistence solely for cosmetic layering.

### Failure Handling
No PostgreSQL connection must be attempted in the default SQLite v1.5 Core profile. Optional PostgreSQL profile fails explicitly when configured but unavailable.

### Reference
Current `internal/store`; v1.5.1 PostgreSQL conformance requirement.

---

## T151-04 Dispatch concurrency seam

### Tests
- two independent READY + parallel-eligible tasks can overlap execution;
- state claim for each task is durable before Agent start;
- finalization reloads latest snapshot so concurrent sibling completion is not lost;
- duplicate dispatch of the same task cannot claim a non-READY task;
- cancellation persists INTERRUPTED;
- CompletionContract behavior remains unchanged.

### Contract
Task execution must not hold the project mutation lock. Only persisted state mutations are serialized.

### Implementation
Refactor `Service.Dispatch` into claim/execute/commit phases. Claim transitions READY→RUNNING and persists. Release lock. Execute Harness/Agent. Reacquire lock and reload current repository state before applying the task outcome. Factor finalization helpers so they operate on the freshly loaded snapshot.

### Failure Handling
If finalization finds a state other than RUNNING/expected interruption state, return a conflict and do not overwrite newer state. If execution errors, persist BLOCKED/INTERRUPTED using the same commit discipline.

### Reference
Current `internal/controlplane/dispatch.go`; v1.5.0 task state semantics.

---

## T151-05 Local scheduler

### Tests
- READY task dispatch;
- blocked dependency is not dispatched;
- parallel batch obeys configured maximum;
- non-parallel task is not overlapped with other scheduler-owned work;
- restart converts persisted RUNNING→INTERRUPTED;
- readiness recomputation unlocks downstream nodes after DONE;
- no lease/heartbeat/stale takeover calls/types exist.

### Contract
Scheduler is an application-local coordinator over Active TaskDAG. It is not a generic workflow engine and owns no second durable project state.

### Implementation
Load persisted snapshot, recover interrupted startup state, call `RefreshReadiness`, persist recovery, then schedule READY nodes. Construct `TaskExecutionRequest` through existing task-contract/execution composition. Dispatch only through `controlplane.Service` / Engineering Harness. Use bounded goroutines/errgroup with explicit cancellation ownership.

### Failure Handling
A task failure affects that task state and scheduler iteration; do not falsely mark siblings done. Context cancellation stops new dispatch and lets active executions persist interruption through existing semantics. Panics are not a recoverable control-flow mechanism.

### Reference
`engineering.ActiveTaskDAG.ReadyNodes/ParallelReady`; v1.5.1 Job Queue replacement path.

---

## T151-06 Production wiring

### Tests
- default configuration selects SQLite and starts v1.5 Core without PostgreSQL;
- explicit PostgreSQL profile selects PG adapter;
- invalid profile is rejected;
- legacy PG-dependent surfaces degrade explicitly rather than forcing PG into v1.5 Core;
- architecture test confirms default v1.5 composition does not construct `jobs.Queue`.

### Contract
`SQLite = default`, `PostgreSQL = optional`. DB Job Queue is not the default execution path.

### Implementation
Separate v1.5 Core composition from legacy PostgreSQL web dependencies. Make persistence profile and SQLite path explicit configuration. Keep brownfield routes only when their dependencies are configured; do not silently fake unavailable data.

### Failure Handling
Configuration errors are fatal and explicit. Optional legacy capability absence is visible/disabled rather than silently substituting another semantic path.

### Reference
Current `cmd/server/main.go`, `cmd/server/v15.go`, v1.5.1 Release Qualification.

---

## T151-07/08 Validation and Closure

### Tests
Focused → Local/Build Host → Integration/E2E → Independent Review → Full Regression → Hidden Validation → Critical Journeys → Version Closure → Release Qualification.

### Contract
No GitHub Actions/CI. No unexecuted PASS. Hidden cases stay held out until visible implementation/tests are complete and candidate revision is frozen.

### Implementation
Record a validation ledger containing command, environment, result, evidence, revision and timestamp. Release artifacts summarize exact acceptance evidence and remaining risk.

### Failure Handling
Environment-only blockers use `ENV-BLOCKED(reason + evidence)`. Product/code failures are `FAIL`, not environment blockers. Release is not qualified while a required code failure exists.

### Reference
Project override and v1.5.1 full-flow execution prompt.
