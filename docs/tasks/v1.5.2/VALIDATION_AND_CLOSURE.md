# Tally v1.5.2 Validation & Closure Ledger

Version: **v1.5.2 — Execution Environments**

This ledger separates implementation completeness from empirical Release Qualification. GitHub Actions and CI are prohibited as development, PR, Closure and Release gates. Validation authority is Local / Build Host execution plus independent/held-out evidence.

## Candidate identity

- Base revision: `bc69dbe5e5f7a9beeb3bd83e7dcfce3b530d7568`
- Development branch: `feat/v1.5.2-execution-environments`
- Implementation issue: #33
- Build Host / release-gate issue: #50
- Draft PR: #49
- Implementation code freeze: `9a35c1ddf3b4ff1a313e92c3f0e8100da6d3653d`
- Candidate state: **CODE-COMPLETE / NOT RELEASE QUALIFIED**

The code-freeze revision contains the implementation and source-level test/validation carriers. Documentation-only commits may follow it. Empirical evidence must always name the exact checked-out revision; do not infer validation from this document or from an Agent claim.

## Implementation status

| Scope | State | Evidence |
|---|---|---|
| Frozen v1.5.0 contract compatibility | CODE-COMPLETE | frozen `ExecutionEnvironment` surface retained; v1.5.2 adds optional interfaces rather than redefining the contract |
| Runtime × Environment orthogonality | CODE-COMPLETE | non-Local runtimes bind through generic `runner.Runner`; no CodexSSHRuntime/DSHSSHRuntime/DSHDockerRuntime class explosion |
| Local compatibility | CODE-COMPLETE as source | Local path remains compatibility edge; production Local composition remains available; Local OS/architecture baseline uses `runtime.GOOS/GOARCH` |
| SSH environment | CODE-COMPLETE as source | cwd/env/argv/stdout/stderr, context cancellation, authentication boundary, fail-closed host-key policy, environment-native Git/workspace lifecycle |
| Docker environment | CODE-COMPLETE as source | task image/profile, mount/workdir/env, `--rm`, forced cancellation cleanup; secret values stay out of Docker CLI argv |
| Capability/failure semantics | CODE-COMPLETE as source | OS/normalized architecture/toolchain/Git/Docker/network/secret-availability/resource profile; typed mismatch before Agent start; transport/workspace/cleanup failures remain distinguishable |
| Workspace restart/recovery | CODE-COMPLETE as source | retained workspace inspect; lost workspace recreate from accepted base revision; re-verification mandatory; destructive lifecycle rejects unsafe roots |
| DSH remote/container Runtime | CODE-COMPLETE as source | one DSH Runtime implements generic `StartInEnvironment`; DSH session state is outside the candidate Git workspace; cancellation follows Runner context |
| Scheduler/Harness integration | CODE-COMPLETE as source | `GitTaskRequestFactory` is environment-aware; host `os.Stat` only on Local; non-Local locators are absolute POSIX paths; implicit remote execution capabilities are fail-closed |
| Production server composition | CODE-COMPLETE as source | `v15server` selects Local/SSH/Docker; DSH model/provider remain Runtime config rather than fabricated `model.select` capability |
| Secret boundary | CODE-COMPLETE as source | EnvironmentProfile stores availability only; Docker CLI argv contains env names, not values; malformed SSH/Docker env validation errors do not echo values |
| Cleanup semantics | CODE-COMPLETE as source | remote session cleanup failure is surfaced as typed `FailureCleanup`; Local close behavior remains backward-compatible |
| Source test carriers | SOURCE-COMPLETE | focused/negative/production-composition/real-environment opt-in test sources committed |
| Visible validation carrier | CARRIER-COMPLETE | `scripts/validate-v1.5.2.sh` covers focused packages, production composition, architecture, full regression, real transport and explicit real-DSH gates |
| Hidden validation carrier | CARRIER-COMPLETE | `scripts/run-v1.5.2-hidden.sh` requires an external held-out pack and a clean frozen candidate |

`CODE-COMPLETE` and `SOURCE-COMPLETE` are implementation states only. They are not PASS evidence.

## Current execution evidence

Evidence captured in the ChatGPT implementation environment during 2026-09-15/16:

| Check | Environment | Command / observation | Result | Evidence |
|---|---|---|---|---|
| Go toolchain availability | ChatGPT container | `go version` | PASS (environment observation only) | `go version go1.23.2 linux/amd64` |
| Docker prerequisite | ChatGPT container | `docker version` | `ENV-BLOCKED` | shell reports Docker CLI unavailable |
| Go dependency retrieval required for focused tests | ChatGPT container / isolated package checkout | `GOSUMDB=off go mod download golang.org/x/crypto@v0.28.0` | `ENV-BLOCKED` | DNS/module retrieval to `proxy.golang.org` failed at resolver/network boundary |
| Complete executable private checkout | ChatGPT tool environment | GitHub connector can read/write private source but no complete executable checkout/archive is exposed to the shell | `ENV-BLOCKED` | repository-wide commands cannot be executed against the exact private candidate here |
| Real SSH target | ChatGPT container | no configured v1.5.2 host/user/auth/workspace/base revision | `ENV-BLOCKED` | external target configuration absent |
| Real Docker target | ChatGPT container | no Docker CLI/daemon/profile | `ENV-BLOCKED` | actual container execution unavailable |
| Real DSH SSH/Docker journeys | ChatGPT container | require explicit real target plus provider credential | `ENV-BLOCKED` | no target/profile; real Agent tests are explicit opt-in and were not invoked |
| Hidden pack | ChatGPT container | `TALLY_V152_HIDDEN_ROOT` not supplied | `ENV-BLOCKED` | held-out fixtures/oracles unavailable |
| Independent Review | external reviewer | no independent review submitted on frozen candidate | `NOT_RUN` | self-review is not independent evidence |

The dependency/network blocker is not evidence that the packages compile or fail to compile. No focused/full-regression PASS is claimed for the frozen candidate from this environment.

## Validation carrier

From a clean checkout of the frozen candidate:

```bash
bash scripts/validate-v1.5.2.sh
```

The script runs, in order:

```text
focused runner
focused taskexecution
focused controlplane
focused DSH runtime
focused v15server production composition
architecture tests
full go test ./... regression
real SSH transport probe or strict ENV-BLOCKED
real Docker transport probe or strict ENV-BLOCKED
explicit real DSH-over-SSH journey or strict ENV-BLOCKED
explicit real DSH-in-Docker journey or strict ENV-BLOCKED
```

Focused/full regression forcibly sets `TALLY_V152_RUN_REAL_DSH=0`; a real provider is called only in the named Agent Critical Journey gates when the operator explicitly sets `TALLY_V152_RUN_REAL_DSH=1`.

### Real SSH profile

```text
TALLY_V152_SSH_HOST
TALLY_V152_SSH_USER
TALLY_V152_SSH_WORKSPACE
TALLY_V152_SSH_BASE_REVISION
TALLY_V152_SSH_KEY_FILE or TALLY_V152_SSH_PASSWORD
TALLY_V152_SSH_KNOWN_HOSTS           # when standard ~/.ssh/known_hosts is unavailable
```

### Real Docker profile

```text
TALLY_V152_DOCKER_IMAGE
TALLY_V152_DOCKER_WORKSPACE_SOURCE
TALLY_V152_DOCKER_WORKSPACE_TARGET   # mount root, normally /workspace
TALLY_V152_DOCKER_WORKSPACE          # actual task Git workspace inside the mount
TALLY_V152_DOCKER_BASE_REVISION
TALLY_V152_DOCKER_NETWORK
```

### Real DSH Agent journey opt-in

```text
TALLY_V152_RUN_REAL_DSH=1
DEEPSEEK_API_KEY
TALLY_V152_DSH_BIN                    # optional
TALLY_V152_DSH_MODEL                  # optional
TALLY_V152_DSH_PROVIDER               # optional
TALLY_V152_DSH_SMOKE_PROMPT           # optional
```

## Hidden Validation

Run only against a clean frozen candidate, with the held-out pack outside this repository:

```bash
TALLY_V152_HIDDEN_ROOT=/outside/repo/tally-v1.5.2-hidden \
  bash scripts/run-v1.5.2-hidden.sh
```

The external pack should cover at least Local portability, SSH auth/host-key/transport failure, hostile argv/env/cwd quoting, secret redaction, remote tool missing, remote Git mismatch, target mismatch, lost workspace/recreate failure, transport failure during recovery, cancellation during transport loss, cleanup failure/unsafe cleanup path, Docker unavailable/missing image/profile/network mismatch, runtime lacking environment binding, capability rejection before Agent start, session-resume fallback on the same environment, production Scheduler/server composition, and negative checks proving Core has no host-local filesystem assumption for SSH/Docker workspaces.

Hidden fixtures/oracles remain outside the implementation repository. Only sanitized status/result references may enter release records.

## Critical Journeys

- **CJ-152-01 Local compatibility** — Codex+Local and DSH+Local retain pre-v1.5.2 behavior.
- **CJ-152-02 SSH production binding** — production composition selects SSH; pinned Git workspace is inspected remotely; DSH runs through the generic SSH Runner with verified host identity.
- **CJ-152-03 Docker production binding** — production composition selects Docker; task workspace is inside the shared mount; DSH runs through the generic Docker Runner with reproducible profile.
- **CJ-152-04 No silent Local fallback** — a non-Local request with a Local-only Runtime fails before Local `Start` is called.
- **CJ-152-05 Capability fail-closed** — OS/arch/tool/Git/Docker/network/secret/resource mismatch blocks before Agent start.
- **CJ-152-06 Workspace recovery** — retained workspace continues; missing/mismatched workspace recreates from accepted base revision; transport failure is not misclassified as workspace loss.
- **CJ-152-07 Cancellation/cleanup** — cancellation reaches SSH/Docker process placement; Docker container cleanup runs; remote Runtime session cleanup failures surface as typed failures; unsafe destructive paths are rejected.
- **CJ-152-08 Restart/session correctness** — WorkspaceRef + accepted base revision remain correctness authority; session resume is optional optimization only.
- **CJ-152-09 Secret boundary** — secret availability is modeled without recording values; Docker command argv and validation errors do not expose secret values.

## Build Host handoff

Issue #50 owns the empirical work that cannot be executed in the current ChatGPT environment. It must record command, environment, result, evidence, exact revision and timestamp for every gate. Do not replace evidence with an Agent claim.

Required sequence:

```text
clean frozen checkout
→ scripts/validate-v1.5.2.sh
→ Local Codex + DSH journeys
→ Integration/E2E + full regression
→ real SSH/Docker evidence
→ Independent Review
→ external Hidden Validation
→ Critical Journeys
→ update this ledger
→ Release Qualification
```

## Release qualification

Current status: **NOT RELEASE QUALIFIED**.

Do not mark PR #49 ready, merge it, close #33/#50, or create a v1.5.2 tag/baseline while a mandatory Local/Regression/Independent Review/Hidden Validation/Critical Journey gate is `FAIL`, `NOT_RUN`, or an unqualified `ENV-BLOCKED`.

Local validation is mandatory PASS. SSH/Docker may use strict `ENV-BLOCKED` evidence only where the actual external environment is genuinely unavailable under the frozen PRD. PR merge alone is never Release PASS.

After the Build Host and held-out gates are complete, update this ledger with exact revision-pinned evidence, perform Release Qualification, then merge/tag according to the project standard.
