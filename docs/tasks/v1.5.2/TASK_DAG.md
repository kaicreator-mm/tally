# Tally v1.5.2 — Task DAG

This DAG records the implementation and closure structure for v1.5.2. It follows the frozen PRD and Issue #33; it does not introduce CI gates.

Implementation code freeze: `79a31bef1655705aa09eaae30f36ddc3fae2863e`.

```text
T152-01 Contract conformance
├── T152-02 Local hardening
├── T152-03 Workspace abstraction + lifecycle
└── T152-04 Capability profile/model + environment availability preflight

T152-05 SSH adapter ────────────────┐
T152-06 Docker adapter ─────────────┼── T152-07 Runtime/Environment integration
T152-03 Workspace/lifecycle ────────┤
T152-04 Capability/preflight ───────┘

T152-07 Integration
├── T152-07A DSH generic EnvironmentRuntime
├── T152-07B Engineering Harness / Scheduler request composition
├── T152-07C v15server Local/SSH/Docker production composition
└── T152-08 Failure/cancel/cleanup/recovery/security tests

T152-08 Tests/source carriers
└── T152-09 Visible validation + real transport/Agent journey carriers

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
| T152-02 | LocalEnvironment compatibility against shared environment semantics | T152-01 | CODE-COMPLETE | `contracts.go`, `environments.go`, `local_v152_test.go` |
| T152-03 | Environment-native workspace locator, Git inspect/verify/recreate, explicit cleanup, safe absolute remote roots | T152-01 | CODE-COMPLETE | `environments.go`, `workspace_lifecycle.go`, lifecycle tests |
| T152-04 | Non-Local availability preflight plus OS/architecture/toolchain/Git/Docker/network/secret/resource capability profile and typed pre-start rejection | T152-01 | CODE-COMPLETE | `executor.go`, `environments.go`, capability/preflight tests |
| T152-05 | SSH argv/cwd/env/stdout/stderr/cancel/auth boundary/host-key policy + redacted env validation errors | T152-01 | CODE-COMPLETE | `internal/runner/ssh.go`, runner v1.5.2 tests |
| T152-06 | Docker task image/profile/workspace/env/process/cancel cleanup; secret values excluded from Docker CLI argv | T152-01 | CODE-COMPLETE | `internal/runner/docker_task.go`, runner v1.5.2 tests |
| T152-07 | Generic Runner binding for non-Local Runtime; no silent Local fallback | T152-03..06 | CODE-COMPLETE | `internal/taskexecution/executor.go` |
| T152-07A | One DSH Runtime implementation bound to Local/SSH/Docker without Runtime×Environment classes | T152-07 | CODE-COMPLETE | `runtimes/dsh/environment.go` |
| T152-07B | Environment-aware Scheduler request composition; no host `os.Stat` for SSH/Docker; implicit remote execution capabilities | T152-03,04,07A | CODE-COMPLETE | `internal/controlplane/git_request_factory.go` |
| T152-07C | Production `v15server` Local/SSH/Docker environment composition; DSH model stays runtime config | T152-07A,07B | CODE-COMPLETE | `cmd/v15server/execution_v152.go`, `main.go` |
| T152-08 | Fake/contract/negative tests for availability, transport, capability, recovery, cleanup, secret boundary and production composition | T152-07A..C | SOURCE-COMPLETE / EXECUTION ENV-BLOCKED | `*v152_test.go`; execution requires usable Go dependency cache/network |
| T152-09 | Explicit Local/Build Host validation, real SSH/Docker transport probes, opt-in real DSH SSH/Docker journeys | T152-08 | CARRIER-COMPLETE; EXECUTION ENV-BLOCKED | `scripts/validate-v1.5.2.sh`, external probe/journey tests |
| T152-10 | Full regression + Integration/E2E | T152-09 | ENV-BLOCKED | requires complete checkout/dependencies/Build Host |
| T152-11 | Independent Review | frozen candidate | NOT_RUN | independent reviewer evidence required |
| T152-12 | Hidden Validation + Critical Journeys | T152-10,T152-11 | ENV-BLOCKED / NOT_RUN | external held-out pack + real environments |
| T152-13 | Version Closure / Release Qualification | T152-10..12 | NOT_RUN | `VALIDATION_AND_CLOSURE.md` |

## Parallelism

SSH and Docker transport work are independent after T152-01. Workspace and capability work can also progress in parallel. The integration edge remains narrow: both environments expose the same runner/workspace/capability surface, while DSH supplies one generic EnvironmentRuntime implementation. Scheduler and server composition select an Environment but do not acquire environment-specific Runtime classes.

## Completion rule

`CODE-COMPLETE` means the implementation exists at the code-freeze revision. It does not mean test PASS, Release PASS or permission to tag. T152-13 remains blocked until mandatory empirical evidence is acceptable under the frozen release rule.
