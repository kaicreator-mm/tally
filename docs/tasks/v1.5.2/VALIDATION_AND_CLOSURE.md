# Tally v1.5.2 Validation & Closure Ledger

Version: **v1.5.2 — Execution Environments**

This ledger separates implementation completeness from empirical Release Qualification. GitHub Actions and CI are prohibited as gates; validation is Local / Build Host plus an external held-out pack.

## Candidate status

- Base revision: `bc69dbe5e5f7a9beeb3bd83e7dcfce3b530d7568`
- Development branch: `feat/v1.5.2-execution-environments`
- Issue: #33
- Candidate state: **CODE-COMPLETE / NOT RELEASE QUALIFIED**

The candidate revision must be pinned again when the branch is frozen for PR review; subsequent evidence must name that exact revision.

## Development status

| Scope | State | Evidence |
|---|---|---|
| Runtime × Environment orthogonality | CODE-COMPLETE | non-Local runtimes bind through generic `runner.Runner`; no CodexSSHRuntime/DSHSSHRuntime/Docker combination classes |
| Local compatibility | CODE-COMPLETE | frozen `ExecutionEnvironment` contract retained; LocalPath is a compatibility edge; Local capability baseline uses Go `GOOS/GOARCH` so Windows/Linux Local do not depend on `uname` |
| SSH environment | CODE-COMPLETE | cwd/env/argv-safe remote command, stdout/stderr, cancellation, auth boundary, fail-closed host-key policy, remote Git workspace verify/inspect/recreate/cleanup |
| Docker environment | CODE-COMPLETE | task image/profile, workspace mount/workdir/env, `--rm`, cancellation cleanup, Git workspace verify/inspect and explicit workspace cleanup |
| capability/failure semantics | CODE-COMPLETE | OS/normalized architecture/toolchain/Git/Docker/network/secret-availability/resource profile; typed pre-start mismatch; dynamic requested-tool probe |
| workspace restart/recovery | CODE-COMPLETE | retained workspace inspection; lost workspace recreation seam followed by mandatory accepted-revision re-verification |
| fake/contract/failure tests | CODE-COMPLETE as source | runner/taskexecution v1.5.2 tests committed, including no-Local-fallback, capability rejection, host-key policy, cleanup guards and Local OS/arch baseline |
| visible focused test execution | ENV-BLOCKED | current ChatGPT container cannot obtain required Go module because outbound DNS/module retrieval is unavailable; no PASS claimed |
| real SSH execution | ENV-BLOCKED | current ChatGPT execution environment has no configured reachable SSH target/auth/workspace |
| real Docker execution | ENV-BLOCKED | current ChatGPT execution environment has no Docker CLI/daemon |
| full repository regression | ENV-BLOCKED | current tool cannot materialize the private repository as a complete checkout and current container cannot retrieve missing Go modules |
| Independent Review | NOT_RUN | requires reviewer evidence on the frozen candidate |
| Hidden Validation | ENV-BLOCKED | external v1.5.2 held-out pack is not available in the current environment |
| Critical Journeys | PARTIAL / ENV-BLOCKED | source/fake journey carriers exist; Local/real SSH/real Docker empirical journeys require executable Build Host evidence |
| Release Qualification | NOT_RUN | mandatory empirical predecessors are not all acceptable |

`CODE-COMPLETE` is an implementation state, not Release PASS.

## Current-session validation evidence

Evidence captured in the implementation session at **2026-09-15T21:11:47Z**:

| Check | Environment | Command / observation | Result | Evidence |
|---|---|---|---|---|
| Go toolchain | ChatGPT container | `go version` | PASS (environment observation only) | `go version go1.23.2 linux/amd64` |
| Docker prerequisite | ChatGPT container | `docker version` | ENV-BLOCKED | shell reports `docker: command not found` |
| Go dependency retrieval needed for focused runner test | ChatGPT container / isolated package checkout | `GOSUMDB=off go mod download golang.org/x/crypto@v0.28.0` | ENV-BLOCKED | DNS request for `https://proxy.golang.org/...` failed: resolver connection refused |
| Full private checkout | ChatGPT tool environment | GitHub private source is readable/writable through connector, but no complete executable checkout/archive is exposed | ENV-BLOCKED | cannot run repository-wide commands against an exact private checkout here |
| Real SSH prerequisite | ChatGPT container | no v1.5.2 SSH target/auth/workspace configured | ENV-BLOCKED | external target configuration absent |
| Hidden pack | ChatGPT container | `TALLY_V152_HIDDEN_ROOT` not provided | ENV-BLOCKED | held-out pack unavailable |

The module-fetch failure is an execution-environment blocker, not evidence that the package compiles or fails to compile. Focused tests remain unclaimed until run from a clean checkout with dependencies available.

## Validation carriers

Visible validation from a clean frozen candidate checkout:

```bash
bash scripts/validate-v1.5.2.sh
```

The script runs focused runner/taskexecution tests, architecture tests and `go test ./...`. Real SSH and Docker probes become PASS only when their explicit environment profiles are configured; missing prerequisites are emitted as `ENV-BLOCKED`, never PASS.

Real SSH prerequisites:

```text
TALLY_V152_SSH_HOST
TALLY_V152_SSH_USER
TALLY_V152_SSH_WORKSPACE
TALLY_V152_SSH_BASE_REVISION
TALLY_V152_SSH_KEY_FILE or TALLY_V152_SSH_PASSWORD
```

SSH host identity must also be verifiable by an explicit callback/configuration or the executing account's standard `~/.ssh/known_hosts`; the implementation does not silently disable host-key checking.

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

## Required Build Host evidence

From a clean checkout pinned to the frozen PR revision:

1. run `bash scripts/validate-v1.5.2.sh` and retain the generated summary/log;
2. run Local Codex and DSH compatibility journeys required by the frozen PRD;
3. configure at least one real SSH target and execute the SSH probe, or attach strict external-environment blocker evidence;
4. configure Docker and execute the Docker probe, or attach strict external-environment blocker evidence;
5. run relevant Integration/E2E and full `go test ./...` regression;
6. attach independent review evidence;
7. run the external held-out pack;
8. execute/record the Critical Journeys below;
9. only then perform Release Qualification.

## Critical Journeys

CJ-152-01 — **Local compatibility**: Local request resolves an existing local workspace and reaches the pre-v1.5.2 `Runtime.Start` path without requiring environment-specific runtime types. Capability discovery must retain a valid OS/architecture baseline on both Unix-like and Windows Local hosts.

CJ-152-02 — **SSH runtime binding**: configured SSH target verifies the pinned Git revision remotely, then a runtime implementing the environment extension executes through the selected SSH `runner.Runner`; cwd/env/argv containing whitespace/quotes remain intact and host identity is verified.

CJ-152-03 — **Docker runtime binding**: configured Docker profile mounts the selected workspace, applies workdir/env/profile, verifies the pinned revision inside the container and executes through the selected Docker `runner.Runner`.

CJ-152-04 — **No silent Local fallback**: an SSH/Docker request paired with a Local-only runtime fails before Agent start; local `Runtime.Start` is never invoked.

CJ-152-05 — **Capability mismatch fail-closed**: missing required OS/architecture/toolchain/Git/Docker/network/secret/resource or runtime capability is rejected before runtime/session start and remains classifiable.

CJ-152-06 — **Lost workspace recovery**: failed pinned-revision verification invokes the configured recreation seam once and re-verifies; unrecoverable loss returns typed `workspace_unavailable`.

CJ-152-07 — **Cancellation and cleanup**: context cancellation reaches the SSH session; Docker cancellation triggers best-effort forced container cleanup; explicit workspace cleanup executes in the selected environment and rejects unsafe root-like paths.

CJ-152-08 — **Restart/session correctness**: workspace identity and pinned base revision remain execution authority; optional session resume may optimize continuity but failure/absence of resume cannot redefine correctness and fallback stays bound to the same environment.

CJ-152-09 — **Capability discovery**: environment discovery reports OS/architecture/tool availability plus Git/Docker/network/secret-availability/resource profile without moving execution state into the discovery result or recording secret values.

## Hidden Validation contract

The external pack should cover at least: Local capability portability, SSH authentication/host-key/transport failure, hostile argv/env/cwd quoting, remote tool missing, remote Git mismatch, target mismatch, lost workspace/recreation failure, cancellation during transport loss, cleanup failure/unsafe cleanup path, Docker unavailable/missing image/profile mismatch, runtime lacking environment binding, capability mismatch before start, session-resume fallback on the same environment, and negative checks proving Core has no host-local filesystem assumption for SSH/Docker workspaces.

Fixtures/oracles remain outside the repository. Only sanitized status/result references may enter release records.

## Release qualification rule

Do not tag or publish v1.5.2 while any mandatory gate is `FAIL`, `NOT_RUN`, or unqualified `ENV-BLOCKED`. Local validation is mandatory PASS. SSH/Docker may use strict `ENV-BLOCKED` evidence only when the actual external environment is genuinely unavailable. PR merge alone is not Release PASS.

After Local/Build Host validation, real-environment evidence, independent review, Full Regression, Hidden Validation and Critical Journeys have acceptable results, update this ledger with revision-pinned evidence and perform Release Qualification. Until then: **NOT RELEASE QUALIFIED; NO v1.5.2 TAG/BASELINE**.
