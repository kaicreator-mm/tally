# Tally v1.5.2 — L3 Implementation Evidence

This file provides implementation guidance/evidence for the v1.5.2 Task DAG while preserving the frozen PRD and v1.5.0 execution contracts.

## T152-01 / T152-02 — Contract conformance and Local hardening

**Tests**
- existing Local execution behavior must remain compatible;
- environment capability rejection must stay classifiable;
- Local request must continue to use traditional `Runtime.Start`.

**Contract**
- do not add methods to frozen `ExecutionEnvironment`;
- additive optional interfaces may expose v1.5.2 capability/process/workspace lifecycle behavior.

**Implementation**
- retain `LocalPath` only as Local compatibility edge;
- use environment-native `Location` for runtime start location;
- extend Local with the same optional discovery/inspection/cleanup surface where safe.

**Failure handling**
- invalid/mismatched WorkspaceRef rejects before start;
- unsafe cleanup path rejects without executing deletion.

## T152-03 — Workspace abstraction

**Tests**
- SSH/Docker resolved workspace contains environment location and no required host-local path;
- pinned base revision is inspected with Git inside the environment;
- lost workspace recreation is followed by mandatory re-verification;
- remote cleanup goes through the environment runner;
- root-like cleanup paths fail closed.

**Contract**
- WorkspaceRef identity/target/base revision remain authority;
- session identity is never substituted for workspace correctness.

**Implementation**
- `environmentAdapter.Resolve` → validate target → verify pinned Git commit → optional recreate → re-verify;
- `InspectWorkspace` reads remote/container Git revision and dirty state;
- `CleanupWorkspace` is explicit and typed.

**Failure handling**
- transport unavailable → `environment_unavailable`;
- revision/workspace missing → `workspace_unavailable`;
- cleanup failure → `cleanup_failed`.

## T152-04 — Capability model

**Tests**
- capability mismatch blocks before Runtime/Agent start;
- OS and architecture comparison is stable across common aliases;
- requested toolchains may be probed dynamically;
- Git/Docker/network/secret/resource requirements are independently rejectable.

**Contract**
- requirement tokens are carried in the existing environment requirement list;
- secret values never appear in the capability profile.

**Implementation**
- discover OS with `uname -s`, architecture with `uname -m`;
- normalize x86_64/x64/amd64 to amd64 and aarch64/arm64 to arm64;
- probe Git/Docker and requested `tool:<name>` with `--version`;
- retain declared network, secret availability and resource profile;
- call `ValidateCapabilities` before workspace resolution/Agent start.

**Failure handling**
- missing requirement → typed `EnvironmentCapabilityError` with missing tokens;
- inability to discover because transport is unavailable → typed environment error.

## T152-05 — SSH environment

**Tests**
- hostile whitespace/quote argv/cwd/env round-trips through command construction;
- invalid env names reject before transport;
- cancellation interrupts remote session;
- missing host-key policy fails closed.

**Implementation**
- quote every argv/dir/env entry for POSIX shell transport;
- validate env variable names;
- capture stdout/stderr and preserve exit-code-vs-transport-error semantics;
- on context cancellation, signal/close session;
- resolve host key through explicit callback or known_hosts; insecure skip is never implicit;
- keep authentication material within transport configuration.

**Failure handling**
- auth/dial/handshake/session errors are transport errors;
- command non-zero exit remains `Result.ExitCode`.

## T152-06 — Docker environment

**Tests**
- deterministic Docker argv includes image/profile/workspace/workdir/env;
- cancellation causes forced container cleanup attempt;
- incomplete workspace source/target profile rejects.

**Implementation**
- task-specific `docker run --rm --name ...`;
- mount source to environment target;
- sorted env arguments for deterministic execution;
- profile supports network/user/read-only/extra args;
- generic Runner remains the Runtime-facing boundary.

**Failure handling**
- missing image/profile → pre-run error;
- cancellation → run error plus best-effort `docker rm -f`.

## T152-07 — Runtime/Environment integration

**Tests**
- non-Local runtime receives the exact selected environment runner;
- Local-only Runtime paired with SSH/Docker must not invoke `Runtime.Start`;
- resume fallback, when allowed, remains on the same selected environment.

**Implementation**
- additive `EnvironmentRuntime.StartInEnvironment(..., runner.Runner)`;
- Local uses frozen Runtime.Start;
- non-Local requires both ProcessExecutionEnvironment and EnvironmentRuntime.

**Failure handling**
- missing non-Local binding → typed process/environment failure before Agent execution;
- no silent downgrade to Local.

## T152-08..T152-13 — Validation and closure

Focused source tests must cover normal and negative paths. `scripts/validate-v1.5.2.sh` is the explicit Local/Build Host carrier and must not be wired to CI. Real SSH/Docker probes are opt-in and can be ENV-BLOCKED only when prerequisites are genuinely absent. Hidden Validation stays outside the implementation repository. Independent Review and Full Regression remain separate gates.

A release tag may be created only after the Closure ledger records acceptable mandatory evidence; source implementation or PR merge by itself is insufficient.
