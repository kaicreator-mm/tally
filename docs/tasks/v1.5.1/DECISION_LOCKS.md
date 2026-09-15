# Tally v1.5.1 — Decision Locks

These decisions implement the frozen v1.5/v1.5.1 PRDs. They do not redefine v1.5.0 Foundation contracts.

1. **Active TaskDAG remains Project Flow.** No second workflow state machine or queue-owned task truth.
2. **SQLite is the default persistence profile.** PostgreSQL is optional.
3. **Fresh v1.5 Core startup must not require PostgreSQL.** Optional brownfield features may be unavailable when their legacy adapter is not configured, but Core lifecycle/readiness/dispatch cannot fail solely because PostgreSQL is absent.
4. **Persistence ports are Tally-owned.** Core signatures contain no `pgx`, `pgtype`, JSONB-specific type, SQL row-lock primitive or DB-generated identity semantic.
5. **Application owns identity and time.** Durable domain IDs and semantic timestamps are generated/supplied above adapters.
6. **Immutable records are a Core invariant.** Fact, Evidence, Action/Audit and Outcome are append-only; database constraints are defense in depth.
7. **One conformance contract, multiple adapters.** SQLite and PostgreSQL must exhibit the same Core-visible transaction/identity/immutability behavior.
8. **DB Job Queue is not the v1.5 default execution path.** Legacy queue code may remain only for brownfield compatibility outside v1.5 Core.
9. **No lease correctness in local scheduling.** No heartbeat, stale-worker takeover, `FOR UPDATE SKIP LOCKED`, queue owner or DB time is required to execute an Active TaskDAG locally.
10. **Scheduler is application-local and bounded.** It selects persisted READY nodes and dispatches them through the existing Engineering Harness/AgentExecutor.
11. **Parallelism follows TaskDAG metadata.** `ParallelEligible=false` is never made concurrent merely to fill capacity.
12. **Dispatch does not hold the project mutation lock while the Agent runs.** State claim and result commit are short critical sections; execution is outside the lock.
13. **Restart is state recovery, not session resume.** Persisted RUNNING tasks recover to INTERRUPTED. Readiness is recomputed deterministically; restart correctness does not depend on an Agent session or worker lease.
14. **Run remains provenance/history/state linkage.** It does not become a generic workflow engine.
15. **Existing Agent Runtime boundary is authoritative.** No second runtime lifecycle is introduced.
16. **No scope pull-forward.** v1.5.1 does not implement SSH/Docker, full Git authority automation, Multi-Agent orchestration, Re-plan/Learning, Redis/NATS/Kafka/SQS or a distributed worker fleet.
17. **Validation carrier is Local/Build Host only.** GitHub Actions and all CI systems are prohibited for this version. Unexecuted commands are never PASS.
18. **Frozen PRDs are not edited to fit implementation.** Any implementation conflict blocks or changes the implementation, not the frozen contract.
