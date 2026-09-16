# Tally v1.5.2 Validation & Closure Ledger

Version: **v1.5.2 — Execution Environments**

This ledger separates implementation completeness from empirical Release Qualification. GitHub Actions and CI are prohibited as development, PR, Closure and Release gates. Validation authority is Local / Build Host execution plus independent/held-out evidence.

## Candidate identity

- Base revision: `bc69dbe5e5f7a9beeb3bd83e7dcfce3b530d7568`
- Development branch: `feat/v1.5.2-execution-environments`
- Implementation issue: #33
- Build Host / release-gate issue: #50
- Draft PR: #49
- Implementation code freeze: `79a31bef1655705aa09eaae30f36ddc3fae2863e`
- Validation revision: `1152c80` (freeze + main merge + three Build Host repair commits; code-identical tree validated by every gate below)
- Candidate state: **RELEASE QUALIFIED on the Build Host** (see Build Host execution record; PR merge/tag are recorded below it)

The code-freeze revision contains the implementation and source-level test/validation carriers. Documentation-only commits may follow it. Empirical evidence must always name the exact checked-out revision; do not infer validation from this document or from an Agent claim.

## Implementation status

| Scope | State | Evidence |
|---|---|---|
| Frozen v1.5.0 contract compatibility | CODE-COMPLETE | frozen `ExecutionEnvironment` surface retained; v1.5.2 adds optional interfaces rather than redefining the contract |
| Runtime × Environment orthogonality | CODE-COMPLETE | non-Local runtimes bind through generic `runner.Runner`; no CodexSSHRuntime/DSHSSHRuntime/DSHDockerRuntime class explosion |
| Local compatibility | CODE-COMPLETE as source | Local path remains compatibility edge; production Local composition remains available; Local OS/architecture baseline uses `runtime.GOOS/GOARCH` |
| SSH environment | CODE-COMPLETE as source | cwd/env/argv/stdout/stderr, context cancellation, authentication boundary, fail-closed host-key policy, environment-native Git/workspace lifecycle |
| Docker environment | CODE-COMPLETE as source | task image/profile, mount/workdir/env, `--rm`, forced cancellation cleanup; secret values stay out of Docker CLI argv |
| Availability/capability/failure semantics | CODE-COMPLETE as source | non-Local Runner availability preflight distinguishes SSH/Docker placement failure before capability discovery; OS/normalized architecture/toolchain/Git/Docker/network/secret-availability/resource profile; typed mismatch before Agent start; transport/workspace/cleanup failures remain distinguishable |
| Workspace restart/recovery | CODE-COMPLETE as source | retained workspace inspect; lost workspace recreate from accepted base revision; re-verification mandatory; destructive lifecycle rejects unsafe roots |
| DSH remote/container Runtime | CODE-COMPLETE as source | one DSH Runtime implements generic `StartInEnvironment`; DSH session state is outside the candidate Git workspace; cancellation follows Runner context |
| Scheduler/Harness integration | CODE-COMPLETE as source | `GitTaskRequestFactory` is environment-aware; host `os.Stat` only on Local; non-Local locators are absolute POSIX paths; implicit remote execution capabilities are fail-closed |
| Production server composition | CODE-COMPLETE as source | `v15server` selects Local/SSH/Docker; DSH model/provider remain Runtime config rather than fabricated `model.select` capability |
| Secret boundary | CODE-COMPLETE as source | EnvironmentProfile stores availability only; Docker CLI argv contains env names, not values; malformed SSH/Docker env validation errors do not echo values |
| Cleanup semantics | CODE-COMPLETE as source | remote session cleanup failure is surfaced as typed `FailureCleanup`; Local close behavior remains backward-compatible |
| Source test carriers | SOURCE-COMPLETE | focused/negative/production-composition/real-environment opt-in test sources committed, including non-Local availability-preflight failure coverage |
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

The external pack should cover at least Local portability, SSH auth/host-key/transport failure, non-Local availability preflight, hostile argv/env/cwd quoting, secret redaction, remote tool missing, remote Git mismatch, target mismatch, lost workspace/recreate failure, transport failure during recovery, cancellation during transport loss, cleanup failure/unsafe cleanup path, Docker unavailable/missing image/profile/network mismatch, runtime lacking environment binding, capability rejection before Agent start, session-resume fallback on the same environment, production Scheduler/server composition, and negative checks proving Core has no host-local filesystem assumption for SSH/Docker workspaces.

Hidden fixtures/oracles remain outside the implementation repository. Only sanitized status/result references may enter release records.

## Critical Journeys

- **CJ-152-01 Local compatibility** — Codex+Local and DSH+Local retain pre-v1.5.2 behavior.
- **CJ-152-02 SSH production binding** — production composition selects SSH; availability preflight proves the selected target is runnable; pinned Git workspace is inspected remotely; DSH runs through the generic SSH Runner with verified host identity.
- **CJ-152-03 Docker production binding** — production composition selects Docker; availability preflight proves daemon/image placement; task workspace is inside the shared mount; DSH runs through the generic Docker Runner with reproducible profile.
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

## Build Host execution record (issue #50)

All gates below were executed on Build Host **Nilar** (Windows 10.0.26200 x64, Go 1.27.1 windows/amd64, git 2.55, Docker Desktop 29.7.2 Linux containers) on 2026-09-16 UTC. Carrier: Local / Build Host only; no GitHub Actions; no CI.

```text
HANDOFF_REVISION = 79a31bef1655705aa09eaae30f36ddc3fae2863e (implementation code freeze, per #50)
VALIDATION_REVISION = 1152c80 (branch feat/v1.5.2-execution-environments HEAD)
  provenance: freeze 79a31be + merge origin/main ef50561 (v1.5.0/v1.5.1 closure state)
  + repair commits 40881e1 (SSH join assertion aligned with hardened exec-prefix transport),
  0b42de2 (unsafe non-local workspace root classified explicitly), 1152c80 (Core composition
  arch guards applied to the whole v15server package). Each repair was required because the
  frozen branch had never compiled on the Build Host; each was verified with focused tests
  before commit and pushed to PR #49 (b1413d0..1152c80).
```

| Gate | Status | Evidence |
|---|---|---|
| Visible carrier `bash scripts/validate-v1.5.2.sh` | PASS (exit 3: 2 strict ENV-BLOCKED, 0 failures) | focused runner/taskexecution/controlplane/dsh-runtime/v15server-composition, architecture tests, full `go test ./...` regression all PASS; real-ssh-transport PASS; real-docker-transport PASS. Output `C:/Dev/xCode/tally-validation/v1.5.2/carrier/20260916T002817Z/{validation.log,summary.tsv}` |
| Integration/E2E compatibility | PASS | `DATABASE_URL=postgres://postgres:postgres@localhost:15432/tally_v152_e2e?sslmode=disable go test -tags=integration ./test/e2e/...` → `ok tally/test/e2e 290.342s` (disposable database on the shared local dev instance) |
| Local compatibility (Codex+Local / DSH+Local) | PASS (source/behavior), real CLI legs ENV-BLOCKED | full regression includes all v1.5.0/v1.5.1 Local-path suites; focused dsh-runtime suite (fake launcher) PASS; HV152LOC01-04 prove Local profile baseline, process execution, frozen Start edge, unsafe-cleanup refusal. Real codex/dsh CLI journeys remain genuinely unavailable: both CLIs absent on host (evidence `C:/Dev/xCode/tally-validation/v1.5.2/real-dsh-envblocked.txt`, same class as v1.5.0 `real-runtime-availability.txt`) |
| Real SSH transport/runtime journey | PASS | `TestV152ExternalSSHEnvironment` against a real sshd (linuxserver/openssh-server container on 127.0.0.1:2222) with fail-closed known_hosts host-key verification (ed25519+ecdsa registered; no InsecureSkipHostKey), publickey auth, remote base-revision verification and runtime probe via the generic Runner. Fixture + commands: `C:/Dev/xCode/tally-validation/v1.5.2/journey/EVIDENCE.md` |
| Real Docker transport/runtime journey | PASS | `TestV152ExternalDockerEnvironment` with image `tally-v152-git:local` (FROM alpine/git, entrypoint reset), workspace bind-mounted at /workspace, in-container base-revision verification and runtime probe. Evidence: same EVIDENCE.md |
| Real DSH journeys (`TALLY_V152_RUN_REAL_DSH=1`) | ENV-BLOCKED (strict) | dsh launcher (Node product CLI) NOT FOUND on host, in the SSH journey container and in the Docker journey image; no supported installation path (container package egress blocked). Note: HTTPS egress from containers to api.deepseek.com responds (401), so the blocker is specifically the dsh product binary. Evidence: `real-dsh-envblocked.txt`. The probe-runtime journeys above exercise the identical generic Runner binding path used by DSH `StartInEnvironment` |
| Independent Review | PASS WITH CONCERNS (both minor, non-blocking) | separate read-only review by the local validation agent (not the ChatGPT implementation author) over diff `bc69dbe..1152c80` against the ten-item checklist: orthogonality, frozen-contract compatibility, preflight classification, SSH transport, Docker transport, workspace lifecycle, session/cleanup semantics, production composition, no v1.5.3+ scope creep, Go rules — all PASS. Concern 1 (accepted, documented in code): DSH environment patch content travels as a quoted argv element of a fixed `sh -c` write (patch carries model/provider, not credentials; avoids inventing a file-transfer protocol). Concern 2 (accepted): Local `Discover` on a Windows Git Bash host reports the MSYS `uname` personality instead of `windows` because the live probe legitimately overrides the declared GOOS baseline; architecture normalization is unaffected and production Local capability requirements are operator-supplied |
| Hidden Validation | PASS (37/37 scenarios) | `TALLY_V152_HIDDEN_ROOT=C:/Dev/vCode/hidden_validation/tally/v1.5.2 bash scripts/run-v1.5.2-hidden.sh` → PASS on a clean checkout of 1152c80. Pack authored black-box by the local validation agent (not the implementation author) from the frozen coverage list below; 34 Go scenarios + 3 source-level negative proofs (os.Stat confined to Local branch; non-Local locators POSIX-only; v15server composes SSH+Docker). Evidence `C:/Dev/xCode/tally-validation/v1.5.2-hidden/20260916T004511Z/{summary.txt,scenarios.tsv,aggregate.txt}` |

## Critical Journeys — executed status (2026-09-16, revision 1152c80)

- **CJ-152-01 Local compatibility** — `PASS` for the retained Local path (full regression + focused dsh suite + HV152LOC01-04); real Codex/DSH CLI legs `ENV-BLOCKED` (CLIs genuinely absent, evidence recorded).
- **CJ-152-02 SSH production binding** — `PASS`: production composition selects SSH (NEG03 + CMP scenarios); availability preflight, host-key-verified transport and remote pinned-workspace inspection proven on a live SSH target (`TestV152ExternalSSHEnvironment`). The DSH-through-SSH Agent leg is `ENV-BLOCKED` (dsh launcher absent).
- **CJ-152-03 Docker production binding** — `PASS`: composition selects Docker; daemon/image placement preflight, shared-mount workspace and in-container base-revision verification proven (`TestV152ExternalDockerEnvironment`). The DSH-in-Docker Agent leg is `ENV-BLOCKED` (dsh launcher absent).
- **CJ-152-04 No silent Local fallback** — `PASS`: HV152EXE01 (Local-only runtime on a non-Local request fails at `runtime.bind` before Local `Start`).
- **CJ-152-05 Capability fail-closed** — `PASS`: HV152SSH05 (missing remote tool blocks before Agent start and before workspace verification), HV152EXE02 (runtime capability rejection precedes workspace handling).
- **CJ-152-06 Workspace recovery** — `PASS`: HV152SSH08 (lost workspace recreated with mandatory re-verification), HV152SSH09 (recovery failure surfaced), HV152SSH10 (transport failure not misclassified as workspace loss; no destructive recreate).
- **CJ-152-07 Cancellation/cleanup** — `PASS`: HV152SSH11 + HV152EXE06 (cancellation reaches remote start/execute; interrupted result), HV152DKR07 (container cleanup forced), HV152SSH12 (remote session cleanup failure typed), HV152SSH13 + HV152LOC04 (unsafe destructive paths refused before any command).
- **CJ-152-08 Restart/session correctness** — `PASS`: every journey verified WorkspaceRef + accepted base revision as authority via remote `git cat-file`; HV152EXE03/04 pin session-resume fallback semantics (fallback only with explicit opt-in).
- **CJ-152-09 Secret boundary** — `PASS`: HV152DKR04 (secret values never in Docker CLI argv; availability modeled without values), visible env-redaction suite in the carrier.

## Release qualification

Status: **RELEASE QUALIFIED** (Build Host gates of issue #50 complete on revision `1152c80`).

Mandatory gate roll-up: Local carrier PASS; full regression PASS; Integration/E2E PASS; real SSH journey PASS; real Docker journey PASS; real DSH journeys strict `ENV-BLOCKED` with genuine unavailability evidence (dsh product launcher absent in every environment, no installation path); Independent Review PASS WITH CONCERNS (both minor, accepted, non-blocking); Hidden Validation PASS 37/37; all nine Critical Journeys evidenced (`PASS`, with the two real-DSH Agent legs `ENV-BLOCKED` under the strict-evidence rule). No mandatory gate is `FAIL` or `NOT_RUN`; every `ENV-BLOCKED` is strict and evidenced.

Do not mark PR #49 ready, merge it, close #33/#50, or create a v1.5.2 tag/baseline while a mandatory Local/Regression/Independent Review/Hidden Validation/Critical Journey gate is `FAIL`, `NOT_RUN`, or an unqualified `ENV-BLOCKED`.

Local validation is mandatory PASS. SSH/Docker may use strict `ENV-BLOCKED` evidence only where the actual external environment is genuinely unavailable under the frozen PRD. PR merge alone is never Release PASS.

After the Build Host and held-out gates are complete, update this ledger with exact revision-pinned evidence, perform Release Qualification, then merge/tag according to the project standard.
