# Tally v1.5.2 — Task DAG

This DAG records the implemented/development closure structure for v1.5.2. It follows the frozen PRD and Issue #33; it does not introduce CI gates.

```text
T152-01 Contract conformance
├── T152-02 Local hardening
├── T152-03 Workspace abstraction cleanup
└── T152-04 Capability profile/model

T152-05 SSH adapter ─────────┐
T152-06 Docker adapter ──────┼── T152-07 Runtime/Environment integration
T152-03 Workspace abstraction┤
T152-04 Capability model ────┘

T152-07 Integration
├── T152-08 Failure/cancel/cleanup/recovery tests
└── T152-09 Real-environment probes + visible validation carrier

T152-09 Validation carrier
├── T152-10 Full regression / Integration-E2E
├── T152-11 Independent Review
├── T152-12 Hidden Validation / Critical Journeys
└── T152-13 Version Closure / Release Qualification
```

## Task ledger

| Task | Scope | Dependencies | State | Primary evidence |
|---|---|---|---|---|
| T152-01 | Preserve frozen v1.5.0 execution contracts; additive v1.5.2 seams only | v1.5.1 baseline | CODE-COMPLETE | `internal/taskexecution/contracts.go` |
| T152-02 | LocalEnvironment compatibility against shared environment semantics | T152-01 | CODE-COMPLETE | `contracts.go`, `environments.go` |
| T152-03 | Environment-native workspace locator, Git inspect/verify/recreate, explicit cleanup | T152-01 | CODE-COMPLETE | `environments.go`, v1.5.2 tests |
| T152-04 | OS/architecture/toolchain/Git/Docker/network/secret/resource capability profile and typed pre-start rejection | T152-01 | CODE-COMPLETE | `environments.go`, `executor.go` |
| T152-05 | SSH argv/cwd/env/stdout/stderr/cancel/auth boundary/host-key policy | T152-01 | CODE-COMPLETE | `internal/runner/ssh.go` |
| T152-06 | Docker task image/profile/workspace/env/process/cancel cleanup | T152-01 | CODE-COMPLETE | `internal/runner/docker_task.go` |
| T152-07 | Generic Runner binding for non-Local Runtime; no silent Local fallback | T152-03..06 | CODE-COMPLETE | `internal/taskexecution/executor.go` |
| T152-08 | Fake/contract/negative tests for transport, capability, recovery, cleanup and fallback | T152-07 | CODE-COMPLETE / EXECUTION ENV-BLOCKED | `*v152_test.go`; package execution requires usable Go dependency cache/network |
| T152-09 | Explicit Local/Build Host validation and real SSH/Docker probes | T152-08 | CARRIER-COMPLETE; EXECUTION ENV-BLOCKED | `scripts/validate-v1.5.2.sh`, external probe tests |
| T152-10 | Full regression + Integration/E2E | T152-09 | ENV-BLOCKED | requires complete checkout/dependencies/Build Host |
| T152-11 | Independent Review | frozen candidate | NOT_RUN | independent reviewer evidence required |
| T152-12 | Hidden Validation + Critical Journeys | T152-10,T152-11 | ENV-BLOCKED / NOT_RUN | external held-out pack + real environments |
| T152-13 | Version Closure / Release Qualification | T152-10..12 | NOT_RUN | `VALIDATION_AND_CLOSURE.md` |

## Parallelism

SSH and Docker adapter work are independent after T152-01. Workspace and capability work can also progress in parallel. The integration edge is intentionally narrow: both environments expose a common runner/workspace/capability surface before reaching Runtime.

## Completion rule

`CODE-COMPLETE` means implementation exists in the candidate. It does not mean test PASS, Release PASS or permission to tag. T152-13 remains blocked until mandatory empirical evidence is acceptable under the frozen release rule.
