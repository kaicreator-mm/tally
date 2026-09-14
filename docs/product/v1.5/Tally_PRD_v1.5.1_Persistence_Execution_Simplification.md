# Tally PRD v1.5.1 — Persistence & Execution Simplification

- Status: **FROZEN EXECUTABLE SCOPE**
- Depends on: v1.5.0

## Goal

Make Engineering Control Plane semantics database-independent and remove the PostgreSQL Job Queue from the default correctness/execution path.

## In scope

1. Persistence ports for Project, Requirements, Planning, TaskGraph, TaskContract, Run, Fact, Evidence, Action/Audit, Outcome and transactions.
2. Remove `pgx` / `pgtype` from Core/domain-facing contracts.
3. Application-owned identity and Clock where correctness requires them.
4. Immutable fact/evidence contracts enforced in Core; DB triggers are optional defense-in-depth.
5. SQLite becomes default clean-install storage.
6. PostgreSQL remains an optional conforming adapter.
7. Storage conformance suite runs against both adapters.
8. Replace default DB Job Queue/worker claim path with application-local TaskDAG readiness scheduling + Thin Task Harness/existing executor path.
9. Remove default correctness dependency on leases, heartbeat, stale ownership and `FOR UPDATE SKIP LOCKED`.
10. Run remains provenance/history/evidence/state linkage, not a generic workflow engine.

## Out of scope

Distributed worker fleet, Redis/NATS/Kafka/SQS, SSH/Docker environments, Git authority automation and multi-Agent roles.

## Acceptance criteria

- Fresh Tally install runs with SQLite without PostgreSQL.
- PostgreSQL passes the same persistence conformance contract.
- Core exposes no PostgreSQL driver types.
- Correctness does not depend on PostgreSQL JSONB, triggers, DB `now()`, identity sequence, SKIP LOCKED, lease or heartbeat semantics.
- Ready Tasks execute through local scheduling with bounded concurrency.
- Restart recomputes ready/interrupted work from semantic persisted state rather than queue leases.
- v1.5.0 lifecycle behavior remains unchanged.

## Hidden Validation

SQLite clean install, rollback, contention, immutable mutation attempt, duplicate identity, migration, PostgreSQL parity, interrupted RUNNING state, readiness recomputation and absence of DB queue dependencies.

## Release qualification

SQLite clean install PASS; PostgreSQL compatibility/conformance PASS; TaskDAG progression PASS; no default DB Job Queue/Lease/Heartbeat/SKIP LOCKED correctness dependency.
