# Tally PRD v1.5.2 — Execution Environments

- Status: **FROZEN EXECUTABLE SCOPE**
- Implementation dependency: **v1.5.0 execution contracts only**
- Release baseline dependency: v1.5.1
- Parallel lane: may be implemented concurrently with v1.5.1, v1.5.3, v1.5.4 and v1.5.5.

## Goal

Implement first-class SSH and Docker execution using the Runtime/Environment/Workspace contracts already frozen in v1.5.0.

## Architecture rule

```text
Agent Runtime              Execution Environment
-------------              ---------------------
Codex                      Local
DSH                        SSH
Future Agent               Docker
```

Runtime defines Agent protocol/interaction. Environment defines workspace/process/file execution.

## In scope

1. Keep v1.5.0 `ExecutionEnvironment`, `ExecutionTarget`, `WorkspaceRef` and capability contracts unchanged unless an explicit cross-version contract change is approved.
2. Harden LocalEnvironment against the shared contract.
3. SSHEnvironment for remote build/dev hosts.
4. DockerEnvironment for isolated/reproducible execution.
5. Environment capability discovery/profile: OS, tools, Docker, network, secrets/policy and optional resources.
6. Engineering Harness selects/validates runtime + environment before execution.
7. Environment-aware workspace operations required by Thin Task Harness.
8. Remote/container cancellation and cleanup.
9. Task-level restart for local, remote or container-backed workspaces.
10. Runtime session resume remains optional.

## Execution context

Git/base revision + Frozen TaskContract + Engineering Context + Required Evidence + Runtime + Environment/target + WorkspaceRef. Git may transport code/versioned artifacts; live Task control state remains in Tally.

## Out of scope

Remote Agent fleet scheduling, Kubernetes orchestration, cross-host distributed queue, automatic cloud provisioning, GitHub lifecycle authority and multi-Agent role orchestration.

## Acceptance criteria

- Codex and DSH retain Local behavior.
- At least one runtime executes through SSH on a configured remote target.
- At least one runtime executes in Docker.
- Core does not assume a workspace path is locally readable.
- Capability mismatch blocks dispatch before Agent start.
- Cancellation/cleanup works per environment.
- Interrupted Task can restart with retained WorkspaceRef or recreate from accepted base revision.
- No runtime×environment class explosion is introduced.

## Hidden Validation

Local, SSH unreachable/auth failure, remote tool missing, Docker mismatch, cancellation, workspace retained/lost, remote Git inspection, capability rejection and resume-not-supported fallback.

## Release qualification

Local PASS; SSH PASS or qualified ENV-BLOCKED evidence; Docker PASS or qualified ENV-BLOCKED evidence; v1.5.0 execution contracts remain compatible; no correctness dependence on session resume.