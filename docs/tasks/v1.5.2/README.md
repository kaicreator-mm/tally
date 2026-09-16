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

- `internal/runner/ssh.go`: SSH cwd/env/argv preservation, stdout/stderr, cancellation, authentication boundary and fail-closed host-key policy; malformed environment errors never include values.
- `internal/runner/docker_task.go`: reproducible task-level Docker image/profile, workspace mount, env/workdir and cancellation cleanup. Docker receives environment values through the docker client process environment and only environment names appear in `docker run --env` argv, so credentials are not copied into the Docker CLI command line.
- `internal/taskexecution/contracts.go`: environment-local resolved workspace locator with LocalPath compatibility edge; additive environment-profile validator; typed capability failure.
- `internal/taskexecution/environments.go`: Local/SSH/Docker process/workspace adapters; OS/architecture/toolchain/Git/Docker/network/secret/resource discovery/validation; Git inspect/recreate; explicit workspace cleanup.
- `internal/taskexecution/workspace_lifecycle.go`: safe environment-native workspace recreation from accepted Git revision, destructive-path guards and correct transport-vs-workspace failure classification.
- `internal/taskexecution/executor.go`: capability gate before workspace/Agent start; remote runtime binding through generic `runner.Runner`; non-Local execution fails closed if runtime binding is unsupported; non-Local session cleanup failures are surfaced as typed cleanup failures.
- `internal/controlplane/git_request_factory.go`: Scheduler request composition is environment-aware; Core host filesystem checks apply only to Local. Non-Local workspaces require absolute non-root POSIX locators. Non-Local DSH requests automatically require process/cancel/workspace/Git/DSH/network/credential capabilities before Agent start.
- `runtimes/dsh/environment.go`: one DSH Runtime implementation executes through a generic Environment runner for SSH or Docker; there are no DSHSSHRuntime/DSHDockerRuntime classes. Environment session state lives outside the candidate Git workspace.
- `cmd/v15server/execution_v152.go`: production Local/SSH/Docker composition and environment profile configuration.
- `cmd/v15server/main.go`: production Scheduler/Harness wiring uses the selected environment rather than forcing Local.
- `internal/*/*v152_test.go`, `runtimes/dsh/environment*_v152_test.go`, `cmd/v15server/execution_v152_test.go`: fake/contract/failure-path/production-composition coverage.
- `internal/taskexecution/environments_external_v152_test.go`: opt-in real SSH/Docker transport/runtime-binding probes.
- `runtimes/dsh/environment_external_v152_test.go`: explicit opt-in real DSH-over-SSH and DSH-in-Docker Critical Journey carriers.
- `scripts/validate-v1.5.2.sh`: explicit Local/Build Host visible validation carrier.
- `scripts/run-v1.5.2-hidden.sh`: external held-out validation carrier.

## Production configuration

Common execution selection:

```text
TALLY_V15_RUNTIME=fake|codex|dsh
TALLY_V15_ENVIRONMENT=local|ssh|docker
TALLY_V15_CONTENT_REPO=<Git source used for frozen task authority/recreation>
TALLY_V15_BASE_REVISION=<accepted base revision>
TALLY_V15_WORKSPACE_ROOT=<environment-local workspace root>
TALLY_V15_ENV_TARGET=<optional logical target id>
```

Local keeps the pre-v1.5.2 Runtime path. Codex and DSH Local behavior remains separate from environment placement.

SSH additionally requires:

```text
TALLY_V15_SSH_HOST
TALLY_V15_SSH_USER
TALLY_V15_SSH_KEY_FILE or TALLY_V15_SSH_PASSWORD
TALLY_V15_SSH_KNOWN_HOSTS     # optional when the service account already has ~/.ssh/known_hosts
TALLY_V15_ENV_NETWORK=1       # required for remote DSH provider access
```

Host-key verification is fail-closed. The production composition does not expose an insecure-skip environment switch.

Docker additionally requires:

```text
TALLY_V15_DOCKER_IMAGE
TALLY_V15_DOCKER_WORKSPACE_SOURCE
TALLY_V15_DOCKER_WORKSPACE_TARGET   # default /workspace; environment mount root
TALLY_V15_DOCKER_NETWORK            # `none` intentionally fails remote DSH network capability
TALLY_V15_DOCKER_USER               # optional
TALLY_V15_DOCKER_READ_ONLY          # optional rootfs hardening
TALLY_V15_DOCKER_BIN                # optional docker binary override
```

DSH production execution also consumes `DEEPSEEK_API_KEY`; its availability is represented as capability metadata without storing the secret value in `EnvironmentProfile`. `TALLY_V15_MODEL` / `TALLY_V15_DSH_PROVIDER` remain DSH runtime configuration and are not misreported as `agentruntime model.select`.

Optional declared environment capabilities/resources:

```text
TALLY_V15_ENV_SECRETS=name1,name2
TALLY_V15_ENV_RESOURCES=cpu=4,memory=8Gi
TALLY_V15_ENV_RECREATE_REPO=<environment-visible Git URL/path; defaults to TALLY_V15_CONTENT_REPO>
```

## Real-environment validation profile

`bash scripts/validate-v1.5.2.sh` runs focused packages, production composition, architecture checks and full regression with real DSH provider calls forcibly disabled. Real transport and Agent journeys are separate named gates.

SSH transport / DSH journey variables:

```text
TALLY_V152_SSH_HOST
TALLY_V152_SSH_USER
TALLY_V152_SSH_WORKSPACE
TALLY_V152_SSH_BASE_REVISION
TALLY_V152_SSH_KEY_FILE or TALLY_V152_SSH_PASSWORD
TALLY_V152_SSH_KNOWN_HOSTS           # required when standard ~/.ssh/known_hosts is unavailable
```

Docker transport / DSH journey variables:

```text
TALLY_V152_DOCKER_IMAGE
TALLY_V152_DOCKER_WORKSPACE_SOURCE
TALLY_V152_DOCKER_WORKSPACE_TARGET   # mount root, normally /workspace
TALLY_V152_DOCKER_WORKSPACE          # actual task Git workspace inside the mount, e.g. /workspace/TASK-1
TALLY_V152_DOCKER_BASE_REVISION
TALLY_V152_DOCKER_NETWORK            # must permit provider access for the real DSH journey
```

Real DSH journeys additionally require:

```text
TALLY_V152_RUN_REAL_DSH=1
DEEPSEEK_API_KEY
TALLY_V152_DSH_BIN                    # optional; default dsh in selected environment
TALLY_V152_DSH_MODEL                  # optional
TALLY_V152_DSH_PROVIDER               # optional
TALLY_V152_DSH_SMOKE_PROMPT           # optional; default is a no-file-change smoke prompt
```

Without explicit `TALLY_V152_RUN_REAL_DSH=1`, the script reports the real-Agent gates as `ENV-BLOCKED`; it never silently invokes an external model during focused/full regression.

## Validation rule

No GitHub Actions or CI gate is used for this version. `CODE-COMPLETE` means the candidate implementation exists; it does not mean tests, Hidden Validation, Critical Journeys or Release Qualification have passed.

See `VALIDATION_AND_CLOSURE.md` for the current evidence state.
