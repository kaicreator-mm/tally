# v1.5.4 Task DAG

```text
T154-01 Role contracts / durable state
    ├── T154-02 Independence + evidence provenance
    └── T154-03 Sequential role engine / repair / restart
              │
              ├──────────────┐
              ▼              ▼
T154-04 Ready/conflict   T154-05 Control Plane persistence/API
scheduler + bounds             │
              └──────┬─────────┘
                     ▼
          T154-06 Production composition
                     │
          same existing Local RuntimeExecutor
          (fake/codex/dsh), no second runtime
                     │
                     ▼
          T154-07 Hidden Validation / CJ
                     │
                     ▼
          T154-08 Full Regression / Closure
```

## Task mapping

| Task | Deliverable | Dependency | GitHub |
|---|---|---|---|
| T154-01 | Role/Policy/WorkflowState/Finding/RoleRun models | v1.5.0 contracts | #39 |
| T154-02 | run/runtime independence + truthful runtime identity + producer provenance | T154-01 | #39 |
| T154-03 | Implement/Review/Repair/Verify state machine, bounded repair, restart | T154-01..02 | #39 |
| T154-04 | readiness/conflict/workspace/bounded concurrency scheduler | v1.5.0 TaskDAG + T154-01 | #40 |
| T154-05 | ProjectSnapshot persistence, single/batch dispatch, HTTP endpoints, CompletionContract integration | T154-03..04 | #40 |
| T154-06 | `cmd/server` production composition using existing RuntimeExecutor | T154-05 | #40 |
| T154-07 | focused tests, Hidden Validation and Critical Journeys | T154-03..06 | #42 |
| T154-08 | Full Regression, independent review, release decision/tag status | T154-07 + v1.5.3 release baseline | #42 |

## Parallel execution

T154-01/02/03 are contract/core work. Scheduler work T154-04 can progress in parallel with parts of T154-03 because it consumes only frozen DAG/request/policy contracts. T154-05 joins them. T154-06 is a narrow integration concern. Validation T154-07 follows implementation; T154-08 cannot become Release PASS until all required executed evidence exists and the v1.5.3 release baseline dependency is satisfied.

No task in this DAG requires GitHub Actions, CI, distributed Job Queue or v1.5.2 SSH/Docker to implement the v1.5.4 core.
