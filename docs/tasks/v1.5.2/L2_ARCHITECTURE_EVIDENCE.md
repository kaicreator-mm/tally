# Tally v1.5.2 — L2 Architecture Evidence

Authority: frozen `docs/product/v1.5/Tally_PRD_v1.5.2_Execution_Environments.md` and Issue #33.

## Architecture conclusion

v1.5.2 must evolve the frozen v1.5.0 `Runtime × ExecutionEnvironment` seam additively. Runtime continues to define Agent protocol/interaction. Environment owns where workspace/process execution happens and how it is inspected, cancelled, cleaned and recovered. No Runtime×Environment combination type is required.

The implementation therefore keeps the frozen `ExecutionEnvironment` interface unchanged and adds optional extension interfaces consumed only where v1.5.2 behavior is required.

## Brownfield evidence

### Local-path assumptions

The v1.5.0 path carried `ResolvedWorkspace.LocalPath` into `agentruntime.StartRequest.Workspace`. That representation is valid only at the Local compatibility edge. SSH/Docker need a locator meaningful inside the selected environment, not a host-readable path.

Decision: `ResolvedWorkspace.Location` is the environment-native runtime locator. `LocalPath` remains only for v1.5.0 Local compatibility. Non-Local environments return no LocalPath.

### Process model

Existing transports are retained:

- Local: `runner.LocalRunner`
- SSH: `runner.SSHRunner`
- Docker: existing Docker CLI transport plus task-level `runner.DockerTaskRunner`

The common `runner.Runner` boundary carries argv/cwd/env/result/cancellation semantics. Non-Local runtimes opt into the additive `EnvironmentRuntime` interface and receive only this generic runner. This prevents `CodexSSHRuntime`, `DSHDockerRuntime`, and equivalent class multiplication.

### SSH

SSH process semantics require argv shell quoting, cwd preservation, env validation/injection, stdout/stderr capture and context cancellation. Authentication remains transport-local. Password/private-key material is never part of capability/provenance structures. Host-key verification is fail-closed by default: use an explicit callback or known_hosts policy; insecure host-key bypass is an explicit transport opt-in only.

### Docker

Task execution requires image/profile, workspace mount, workdir, env, network/user/read-only/extra arguments, deterministic command construction, `--rm`, and best-effort forced cleanup on cancellation. Runtime code sees only a generic runner.

### Workspace / Git

Workspace identity remains `WorkspaceRef` plus accepted base revision. Resolution verifies the pinned revision with Git inside the selected environment. Inspection also executes Git inside the selected environment. Missing workspace may invoke one configured recreation seam and must then re-verify the same accepted base revision.

Cleanup is environment-owned and rejects unsafe root-like paths before invoking destructive operations.

### Capability model

Static execution capabilities remain compatible with v1.5.0. The v1.5.2 additive profile represents:

- OS;
- normalized architecture (`amd64`, `arm64`, or observed value);
- toolchain availability/version;
- Git;
- Docker;
- network availability;
- secret availability by logical name only;
- optional named resources.

A request may require `os:<value>`, `arch:<value>`, `tool:<name>`, `git`, `docker`, `network`, `secret:<name>`, or `resource:<name>[=<value>]`. Profile mismatch is returned as typed `EnvironmentCapabilityError` before workspace resolution and before Agent start.

### Cancellation and cleanup

Cancellation is context-driven. Local uses `exec.CommandContext`; SSH signals/closes the remote session; Docker cancellation performs best-effort `docker rm -f` after the run transport reports cancellation. Workspace cleanup is a separate environment operation and must not be inferred from process cleanup.

### Restart/recovery

Correctness authority is WorkspaceRef + accepted base revision + bounded TaskContract, not Agent session state:

1. retained workspace → inspect/re-verify and continue/rerun;
2. unavailable workspace → recreate from accepted base revision, re-verify, rerun bounded contract;
3. session resume is optional and must never cause fallback to a different environment.

## Dependency and boundary impact

The v1.5.2 implementation adds no new Go module dependency. `internal/taskexecution` reuses `internal/runner` and existing `agentruntime` contracts. Scheduler/control-plane lifecycle state is not moved into SSH or Docker.

## Failure semantics

Environment failures are classifiable as environment unavailable, workspace unavailable, process failure or cleanup failure. Unsupported non-Local runtime binding fails closed. Target mismatch fails before execution. Capability mismatch fails before Agent start. A remote request must never silently call Local `Runtime.Start`.

## Architecture risks addressed

- host-local path leakage: removed from non-Local resolution;
- runtime/environment class explosion: prevented by generic runner binding;
- shell argument/env corruption: explicit quoting and env-name validation;
- host-key trust-on-first-use ambiguity: fail-closed default;
- secret leakage through capability profile: store availability, not values;
- stale/lost workspace: accepted-revision verification and bounded recreation seam;
- dangerous cleanup: root-like paths rejected;
- architecture alias mismatch: normalized architecture comparison.

## Deferred / out of scope

No remote fleet manager, Kubernetes, cloud provisioning, distributed queue, GitHub lifecycle authority, Multi-Agent orchestration or automatic re-plan is introduced by v1.5.2.
