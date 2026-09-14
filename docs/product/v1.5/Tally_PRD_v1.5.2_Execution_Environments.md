# Tally PRD v1.5.2 — Execution Environments

- Status: **FROZEN EXECUTABLE SCOPE**
- Depends on: v1.5.1

## Goal

Decouple Agent Runtime from where the Agent executes, and make Local, SSH and Docker first-class execution environments.

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

1. Small `ExecutionEnvironment` abstraction.
2. Preserve `internal/agentruntime` as the runtime-neutral Agent protocol.
3. LocalEnvironment.
4. SSHEnvironment for remote build/dev hosts.
5. DockerEnvironment for isolated/reproducible execution.
6. Environment capability model: OS, tools, Docker, network, secrets/policy and optional resources.
7. Engineering Harness selects/validates runtime + environment before execution.
8. Replace local-path assumptions with environment-aware `WorkspaceRef`.
9. Environment-aware workspace operations required by Thin Task Harness.
10. Task-level restart for local, remote or container-backed workspaces.
11. Runtime session resume remains optional.

## Execution context

Git/base revision + Frozen TaskContract + Engineering Context + Required Evidence + Runtime + Environment/target + WorkspaceRef. Git may transport code/versioned artifacts; live Task control state remains in Tally.

## Out of scope

Remote Agent fleet scheduling, Kubernetes orchestration, cross-host distributed queue, automatic cloud provisioning, GitHub lifecycle authority and multi-Agent role workflow.

## Acceptance criteria

- Codex and DSH retain Local behavior.
- At least one runtime executes through SSH on a configured remote target.
- At least one runtime executes in Docker.
- Core does not assume a workspace path is locally readable.
- Capability mismatch blocks dispatch before Agent start.
- Cancellation/cleanup works per environment.
- Interrupted Task can restart with retained WorkspaceRef or recreate from accepted base revision.

## Hidden Validation

Local, SSH unreachable/auth failure, remote tool missing, Docker mismatch, cancellation, workspace retained/lost, remote Git inspection, capability rejection and resume-not-supported fallback.

## Release qualification

Local PASS; SSH PASS or qualified ENV-BLOCKED evidence; Docker PASS or qualified ENV-BLOCKED evidence; no runtime×environment class explosion; no correctness dependence on session resume.
