# Tally PRD v1.5.0 — Engineering Control Plane Foundation

- Status: **FROZEN EXECUTABLE SCOPE**
- Base: v1.4.1
- Purpose: establish the shared domain and execution contracts that allow all later v1.5 capability slices to be developed in parallel.

## Goal

Make Tally own the software-engineering lifecycle from Requirements through an active, validated TaskDAG, while freezing the common execution abstraction used by persistence, remote environments, Git authority, multi-Agent workflow and re-planning.

## In scope — Engineering Control Plane

1. Requirements Analysis and Requirements Baseline.
2. Project Plan and immutable PlanningRevision.
3. Architecture / Engineering Design artifact and constraints.
4. `TaskDAGProposal` → Engineering Harness analyze/evaluate/validate → `Active TaskDAG`.
5. `Active TaskDAG = Project Flow`; no second Project Flow state source.
6. Persist TaskNode, dependency, readiness, block reason, parallelism eligibility and task lifecycle state.
7. `EngineeringNode` common model.
8. `CompletionContract` with required artifacts, evidence, records, Git requirements and authority requirements.
9. ArtifactRef, EvidenceRef and append-only Action/Audit record concepts.
10. Participation modes: `AUTO`, `SUPERVISED`, `AUTHORITY_REQUIRED`.
11. `IntelligenceProvider` boundary with Direct LLM and AI Runtime implementing the same intelligence contracts.
12. Engineering Harness control loop for validate → ready → dispatch → verify/review → accept/repair/escalate → advance/block.

## In scope — shared Agent execution contracts

v1.5.0 also freezes the minimum execution abstraction needed by all parallel follow-up versions:

- `AgentExecutor` — thin task-level execution port.
- existing Agent Runtime boundary — preserve `internal/agentruntime`; do not create a second runtime stack.
- `ExecutionEnvironment` — where/how a process and workspace execute.
- `ExecutionTarget` — selected concrete target/profile.
- `WorkspaceRef` — environment-aware workspace identity, not assumed to be a local path.
- `CapabilityRequirement` / runtime and environment capability descriptors.
- `TaskExecutionRequest` — Frozen TaskContract + Engineering Context + Runtime + Environment + Workspace + execution policy.
- `TaskExecutionResult` — Agent claim/candidate + artifacts/evidence + runtime/environment metadata + execution status.
- cancel/interruption/task-level restart semantics.
- Local compatibility adapter sufficient to keep current Codex/DSH execution usable.

Canonical boundary:

```text
Engineering Harness
        ↓
   AgentExecutor
        ↓
Agent Runtime  ×  Execution Environment
(Codex/DSH)      (Local now; SSH/Docker later)
```

Runtime defines how Tally interacts with an Agent. Environment defines where and how the process/workspace executes. Task correctness must not depend on Agent session resume.

## Required artifacts

At minimum: Requirements baseline, Project Plan/PlanningRevision, Architecture note/ADR/document, TaskDAG baseline, TaskContract, Verification result, Review result and Acceptance record. Human-readable and machine-readable forms may coexist.

## Intelligence rule

Every intelligence-assisted function uses the same `IntelligenceProvider` contract with Direct LLM and AI Runtime implementations. AI Runtime may improve quality/assurance but may not own persistent engineering state.

## Human authority

Node policy determines AUTO, SUPERVISED or AUTHORITY_REQUIRED. Scope-changing requirement decisions and high-impact architecture decisions must support an authority gate.

## Out of scope

- SQLite default/PostgreSQL adapter migration and DB Job Queue removal (v1.5.1).
- SSH/Docker environment implementations and remote workspace mechanics (v1.5.2).
- full GitHub Issue/PR/Merge/Tag/Release authority automation (v1.5.3).
- full multi-Agent role workflow (v1.5.4).
- automated re-plan and governed engineering learning (v1.5.5).

## Acceptance criteria

- Project progresses Intent → Requirements Baseline → Project Plan → Architecture → validated Active TaskDAG.
- Invalid/cyclic/incomplete DAG proposals are rejected before activation.
- Dependency/readiness state persists across Tally restart.
- CompletionContract blocks completion when required artifact/evidence/record/authority is missing.
- Direct LLM and AI Runtime satisfy the same intelligence contract for implemented intelligence nodes.
- Human authority can block a transition that would otherwise pass.
- Active TaskDAG is the single Project Flow state authority.
- `AgentExecutor`, Runtime boundary, `ExecutionEnvironment`, `ExecutionTarget`, `WorkspaceRef`, capability and request/result contracts are frozen and exercised through Local execution.
- Existing Codex/DSH execution remains usable without a second runtime abstraction.
- Later versions can compile/design against these contracts without waiting on each other.

## Hidden Validation

Ambiguous requirements, missing AC, architecture constraint violation, DAG cycle, missing dependency, blocked→ready, parallel-ready tasks, missing artifacts/evidence, authority-required transition, provider substitution, restart state recovery, local execution via AgentExecutor, cancel/interruption, workspace reference preservation and session-resume-unavailable fallback.

## Release qualification

The lifecycle runs end-to-end using current/local execution; shared v1.5 contracts are stable enough for parallel v1.5.1–v1.5.5 development; all required artifacts/evidence are traceable; no deferred capability is falsely claimed complete.