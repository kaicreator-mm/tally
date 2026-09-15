# Tally v1.5.1 — Task DAG

Base: v1.5.0 Foundation merged on `main`.

```text
T151-01 Persistence ports / Core cleanup (#18)
├── T151-02 SQLite default adapter (#19)
├── T151-03 PostgreSQL optional adapter/conformance (#20)
└── T151-04 Dispatch concurrency seam (#21)

T151-02 + T151-04
└── T151-05 Local TaskDAG scheduler / v1.5 default queue removal (#21)

T151-02 + T151-03 + T151-05
└── T151-06 Production composition/config/startup (#22)
    └── T151-07 Full regression + Hidden Validation + Critical Journeys (#22)
        └── T151-08 Version Closure / Release Qualification (#22)
```

## T151-01 — Persistence contracts

Owner: #18  
Model: high for interface/transaction decisions; low-cost model may implement mechanical DTO/compile-safe cleanup after L3.

Deliver:
- database-independent named persistence ports;
- Clock/identity policy;
- transaction and append-only contract;
- architecture tests that prevent pgx/pgtype leakage into the new Core boundary.

## T151-02 — SQLite adapter

Owner: #19  
Parallel after T151-01.

Deliver:
- pure-Go SQLite driver integration;
- default schema/migrations;
- implementation of persistence contract / aggregate Control Plane repository;
- bounded busy/contention configuration;
- restart and append-only behavior;
- reusable conformance fixture.

## T151-03 — PostgreSQL adapter

Owner: #20  
Parallel with T151-02 after T151-01.

Deliver:
- PostgreSQL implementation behind adapter-only package;
- same conformance suite as SQLite;
- migration/transaction parity evidence;
- no Core signature leakage.

## T151-04 — Dispatch concurrency seam

Owner: #21  
Parallel with storage adapters after T151-01.

Deliver:
- claim READY→RUNNING under mutation lock;
- Harness/Agent execution outside mutation lock;
- final result commit under mutation lock against latest persisted snapshot;
- tests proving independent parallel-eligible nodes are not serialized by Service lock.

## T151-05 — Local scheduler

Owner: #21  
Depends on T151-02 and T151-04.

Deliver:
- persisted DAG recovery/readiness refresh;
- bounded local scheduling;
- non-parallel exclusion;
- restart RUNNING→INTERRUPTED recovery;
- dispatch via existing Service/Engineering Harness;
- no Job Queue/lease/heartbeat/SKIP LOCKED dependency.

## T151-06 — Production wiring

Owner: #22.

Deliver:
- SQLite default profile;
- PostgreSQL optional profile;
- legacy/brownfield PG dependencies explicitly optional and separated from v1.5 Core startup;
- config/docs/AGENTS/project override updates;
- no GitHub Actions/CI changes.

## T151-07 — Validation

Owner: #22.

Run explicitly on Local/Build Host:
- focused tests;
- SQLite conformance;
- PostgreSQL conformance where environment is available;
- integration/E2E;
- independent review;
- full regression;
- held-out Hidden Validation after candidate freeze;
- Critical Journeys.

Each entry records command, environment, result, evidence, revision and timestamp. Status is only `PASS`, `FAIL`, `NOT_RUN(reason)` or `ENV-BLOCKED(reason + evidence)`.

## T151-08 — Closure

Owner: #22.

Release only when the v1.5.1 acceptance proof is complete. PR merge alone is not Release Qualification.
