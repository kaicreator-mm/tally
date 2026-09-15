# Tally v1.5.2 Validation & Closure Ledger

Version: **v1.5.2 — Execution Environments**

This ledger separates implementation completeness from empirical Release Qualification. GitHub Actions and CI are prohibited as gates; validation is Local / Build Host plus an external held-out pack.

## Development status

| Scope | State | Evidence |
|---|---|---|
| Runtime × Environment orthogonality | CODE-COMPLETE | non-Local runtimes bind through generic `runner.Runner`; no CodexSSHRuntime/DSHDockerRuntime classes |
| Local compatibility | CODE-COMPLETE | frozen `ExecutionEnvironment` contract retained; LocalPath remains a compatibility edge only |
| SSH environment | CODE-COMPLETE | cwd/env/argv-safe remote command, cancellation, remote Git workspace verification/recreation seam, discovery |
| Docker environment | CODE-COMPLETE | task image/profile, workspace mount/workdir/env, `--rm`, cancellation cleanup, Git workspace verification |
| capability/failure semantics | CODE-COMPLETE | pre-start environment/runtime capability negotiation and typed environment failures |
| fake/contract/failure tests | CODE-COMPLETE | runner and taskexecution v1.5.2 tests committed |
| real SSH execution | ENV-BLOCKED | current ChatGPT execution environment has no configured reachable SSH target/auth/workspace |
| real Docker execution | ENV-BLOCKED | current ChatGPT execution environment has no Docker CLI/daemon |
| full repository regression | ENV-BLOCKED | current tool cannot materialize the private repository as a complete checkout; GitHub connector supports source read/write but not private archive checkout |
| Independent Review | NOT_RUN | requires independent reviewer evidence on frozen candidate |
| Hidden Validation | ENV-BLOCKED | external v1.5.2 held-out pack not available in current environment |
| Release Qualification | NOT_RUN | mandatory empirical predecessors are not all PASS |

`CODE-COMPLETE` is not Release PASS.

## Validation carriers

Visible validation from a clean candidate checkout:

```bash
bash scripts/validate-v1.5.2.sh
```

The script runs focused runner/taskexecution tests, architecture tests and `go test ./...`. Real SSH and Docker probes become PASS only when their explicit environment profiles are configured; missing prerequisites are emitted as `ENV-BLOCKED`, never PASS.

Real SSH variables:

```text
TALLY_V152_SSH_HOST
TALLY_V152_SSH_USER
TALLY_V152_SSH_WORKSPACE
TALLY_V152_SSH_BASE_REVISION
TALLY_V152_SSH_KEY_FILE or TALLY_V152_SSH_PASSWORD
```

Real Docker variables:

```text
TALLY_V152_DOCKER_IMAGE
TALLY_V152_DOCKER_WORKSPACE_SOURCE
TALLY_V152_DOCKER_WORKSPACE_TARGET   # optional; default /workspace
TALLY_V152_DOCKER_BASE_REVISION
```

The selected Docker image must contain `git` because the environment contract verifies the pinned revision from inside the container.

Held-out validation:

```bash
TALLY_V152_HIDDEN_ROOT=/outside/repo/tally-v1.5.2-hidden \
  bash scripts/run-v1.5.2-hidden.sh
```

## Critical Journeys

CJ-152-01 — **Local compatibility**: Local request resolves an existing local workspace and reaches the pre-v1.5.2 `Runtime.Start` path without requiring environment-specific runtime types.

CJ-152-02 — **SSH runtime binding**: configured SSH target verifies the pinned Git revision remotely, then a runtime implementing the environment extension executes through the selected SSH `runner.Runner`; cwd/env/argv containing whitespace/quotes remain intact.

CJ-152-03 — **Docker runtime binding**: configured Docker profile mounts the selected workspace, applies workdir/env/profile, verifies the pinned revision inside the container and executes through the selected Docker `runner.Runner`.

CJ-152-04 — **No silent Local fallback**: an SSH/Docker request paired with a Local-only runtime fails before agent start; local `Runtime.Start` is never invoked.

CJ-152-05 — **Capability mismatch fail-closed**: missing required environment or runtime capability is rejected before runtime/session start and remains classifiable.

CJ-152-06 — **Lost workspace recovery**: failed pinned-revision verification invokes the configured recreation seam once and re-verifies; unrecoverable loss returns typed `workspace_unavailable`.

CJ-152-07 — **Cancellation and cleanup**: context cancellation reaches the SSH session; Docker cancellation triggers best-effort forced container cleanup; execution is not reported completed.

CJ-152-08 — **Restart/session correctness**: workspace identity and pinned base revision remain execution authority; optional session resume may optimize continuity but failure/absence of resume cannot redefine correctness and fallback stays bound to the same environment.

CJ-152-09 — **Capability discovery**: environment discovery reports OS/tool availability plus declared Docker/network/secrets/resource profile without moving execution state into the discovery result.

## Hidden Validation contract

The external pack should cover at least: SSH authentication/transport failure, hostile argv/env/cwd quoting, remote Git mismatch, target mismatch, lost workspace/recreation failure, cancellation during transport loss, Docker missing image/profile mismatch, failed container cleanup, runtime lacking environment binding, capability mismatch before start, session-resume fallback on the same environment, and negative checks proving Core has no host-local filesystem assumption for SSH/Docker workspaces.

Fixtures/oracles remain outside the repository. Only sanitized status/result references may enter release records.

## Release qualification rule

Do not tag or publish v1.5.2 while any mandatory gate is `FAIL`, `NOT_RUN`, or `ENV-BLOCKED`. PR merge is not Release PASS. After Local/Build Host visible validation, real-environment Critical Journeys, independent review and Hidden Validation have acceptable evidence, update this ledger with the frozen revision/result locations and perform Release Qualification.
