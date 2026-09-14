# v1.5.0 Executable Task DAG

Status: ACTIVE
Baseline after v1.4.1 correctness merge: `71a8f67de512495149e2e1f12e0a08c10e52fcff`

```text
T150-00 authority/L2/locks (#5)
  ├─ T150-10 lifecycle domain (#6)
  │    ├─ T150-11 DAG validation/readiness (#6)
  │    └─ T150-12 completion/authority/audit (#6)
  ├─ T150-20 intelligence contract (#7)
  └─ T150-21 AgentExecutor × Environment contract (#7)
       └─ T150-22 Local runtime compatibility (#7)

T150-10 + T150-11 + T150-12 + T150-20 + T150-22
  └─ T150-30 control-plane orchestration/persistence (#8)
       └─ T150-31 brownfield execution integration/API (#8)
            └─ T150-40 visible regression/minimal UI/API (#9)
                 └─ T150-41 Hidden Validation/Critical Journeys (#9)
                      └─ T150-42 Version Closure/Release Qualification (#9)
```

## Task contracts

| Task | Concern | Depends | Model | Completion evidence |
|---|---|---|---|---|
| T150-00 | authority, L2, locks, no-CI override | baseline | HIGH | docs committed/reviewed |
| T150-10 | requirements/plan/architecture typed domain | T150-00 | HIGH | focused domain tests |
| T150-11 | TaskDAG validation/readiness/transitions | T150-10 | HIGH | cycle/missing dependency/parallel-ready tests |
| T150-12 | EngineeringNode, CompletionContract, authority, refs/audit | T150-10 | HIGH | missing evidence/authority/illegal transition tests |
| T150-20 | IntelligenceProvider + Direct/AI Runtime adapter seam | T150-00 | HIGH | same-contract/provider-error tests |
| T150-21 | execution contracts/workspace/capabilities | T150-00 | HIGH | contract tests |
| T150-22 | LocalEnvironment + AgentExecutor over agentruntime | T150-21 | HIGH | fake runtime/cancel/interruption/resume fallback tests |
| T150-30 | control-plane project lifecycle + restartable repository | T150-11,T150-12,T150-20,T150-22 | HIGH | lifecycle/restart tests |
| T150-31 | brownfield ports + composition/API | T150-30 | HIGH | integration tests / production reachability |
| T150-40 | visible regression + minimal UI/API | T150-31 | HIGH | explicit Local/Build Host evidence |
| T150-41 | held-out Hidden Validation + CJ | T150-40 | HIGH | held-out report |
| T150-42 | closure/release qualification | T150-41 | HIGH | integrated-baseline release report |

## Concurrency
After T150-00, domain, intelligence and execution-contract lanes can proceed concurrently. Integration waits for all shared contracts. Hidden Validation cannot begin before visible candidate freeze.

## Scope guard
No task owns v1.5.1 SQLite/Postgres migration/job queue removal, v1.5.2 SSH/Docker implementation, v1.5.3 full Git authority, v1.5.4 multi-Agent orchestration or v1.5.5 automated re-plan/learning.
