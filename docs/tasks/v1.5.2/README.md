# Tally v1.5.2 — Execution Environments

Authority: `docs/product/v1.5/Tally_PRD_v1.5.2_Execution_Environments.md` and Issue #33.

v1.5.2 evolves the frozen v1.5.0 Runtime × ExecutionEnvironment seam without creating Runtime×Environment combination classes. Local remains compatible; SSH and Docker become first-class execution environments with environment-native workspace/process semantics.

## Implementation map

- `internal/runner/ssh.go`: SSH cwd/env/argv preservation, cancellation and host-key callback plumbing.
- `internal/runner/docker_task.go`: reproducible task-level Docker execution profile and cleanup.
- `internal/taskexecution/contracts.go`: environment-local resolved workspace locator with LocalPath compatibility edge; typed capability failure.
- `internal/taskexecution/environments.go`: Local/SSH/Docker process/workspace/capability adapters, discovery, Git inspection and recreation seam.
- `internal/taskexecution/executor.go`: remote runtime binding through generic `runner.Runner`; non-Local execution fails closed if runtime binding is unsupported.
- `internal/*/*v152_test.go`: fake/contract/failure-path coverage.
- `internal/taskexecution/environments_external_v152_test.go`: opt-in real SSH/Docker runtime-binding probes.
- `scripts/validate-v1.5.2.sh`: explicit Local/Build Host visible validation carrier.
- `scripts/run-v1.5.2-hidden.sh`: external held-out validation carrier.

No GitHub Actions or CI gate is used for this version.

See `VALIDATION_AND_CLOSURE.md` for evidence status and Critical Journeys.
