# Tally v1.5.2 — Execution Environments

Authority: frozen `docs/product/v1.5/Tally_PRD_v1.5.2_Execution_Environments.md` and Issue #33.

v1.5.2 evolves the frozen v1.5.0 Runtime × ExecutionEnvironment seam without creating Runtime×Environment combination classes. Local remains compatible; SSH and Docker become first-class execution environments with environment-native workspace/process/capability/lifecycle semantics.

## Engineering evidence

- `L2_ARCHITECTURE_EVIDENCE.md` — brownfield architecture analysis and selected seams.
- `DECISION_LOCKS.md` — frozen implementation decisions and scope guards.
- `TASK_DAG.md` — task dependencies, implementation state and validation gates.
- `L3_IMPLEMENTATION_EVIDENCE.md` — Tests → Contract → Implementation → Failure Handling guidance/evidence.
- `VALIDATION_AND_CLOSURE.md` — empirical evidence ledger, Critical Journeys and release rule.

## Implementation map

- `internal/runner/ssh.go`: SSH cwd/env/argv preservation, stdout/stderr, cancellation, authentication boundary and fail-closed host-key policy.
- `internal/runner/docker_task.go`: reproducible task-level Docker image/profile, workspace mount, env/workdir and cancellation cleanup.
- `internal/taskexecution/contracts.go`: environment-local resolved workspace locator with LocalPath compatibility edge; additive environment-profile validator; typed capability failure.
- `internal/taskexecution/environments.go`: Local/SSH/Docker process/workspace adapters; OS/architecture/toolchain/Git/Docker/network/secret/resource discovery/validation; Git inspect/recreate; explicit workspace cleanup.
- `internal/taskexecution/executor.go`: capability gate before workspace/Agent start; remote runtime binding through generic `runner.Runner`; non-Local execution fails closed if runtime binding is unsupported.
- `internal/*/*v152_test.go`: fake/contract/failure-path/cross-platform Local coverage.
- `internal/taskexecution/environments_external_v152_test.go`: opt-in real SSH/Docker runtime-binding probes.
- `scripts/validate-v1.5.2.sh`: explicit Local/Build Host visible validation carrier.
- `scripts/run-v1.5.2-hidden.sh`: external held-out validation carrier.

## Validation rule

No GitHub Actions or CI gate is used for this version. `CODE-COMPLETE` means the candidate implementation exists; it does not mean tests, Hidden Validation, Critical Journeys or Release Qualification have passed.

See `VALIDATION_AND_CLOSURE.md` for the current evidence state.
