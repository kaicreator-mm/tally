# Tally v1.5.0 Decision Locks

Status: ACTIVE for v1.5.0
Authority: frozen v1.5 PRD + executable v1.5.0 PRD
Issue: #5

## DL-150-01 Product boundary
Tally is the Engineering Control Plane: Strong Agent + Thin Task Harness + Strong Engineering Harness. Agent thought/conversation/checkpoint state is not Tally correctness state.

## DL-150-02 Project flow authority
`Active TaskDAG` is the sole Project Flow state authority. No peer ProjectFlow state machine or mutable completion-percentage source may be introduced.

## DL-150-03 Brownfield ownership
Retain `internal/workflow` as bounded/single-task legacy execution behavior and reuse `internal/closedloop` for verification/repair/acceptance. Project-level lifecycle orchestration is added as a bounded v1.5 control-plane seam; no second Closed Loop is created.

## DL-150-04 Runtime authority
`internal/agentruntime.Runtime/Session` remains the only Agent Runtime protocol. `AgentExecutor` is a consumer-side task port layered above it, not a peer runtime lifecycle.

## DL-150-05 Runtime × environment
Agent Runtime and Execution Environment are orthogonal. v1.5.0 implements Local compatibility only. SSH/Docker behavior is contract-only and deferred to v1.5.2.

## DL-150-06 Workspace identity
Core contracts use environment-aware `WorkspaceRef`; a local path is only a Local environment locator. Core logic must not require every workspace to be locally readable.

## DL-150-07 Intelligence
Direct LLM and AI Runtime satisfy the same `IntelligenceProvider` contract. Providers return Proposal/Assessment/Recommendation; they never own persistent engineering state or deterministic gate decisions.

## DL-150-08 Deterministic gates
Dependency satisfaction, cycle detection, readiness, required artifact/evidence/record/authority checks, capability matching, legal state transition and revision/hash checks are deterministic code, not LLM judgment.

## DL-150-09 Completion
A node becomes complete only when its `CompletionContract` is satisfied. Function return, Agent `done`, or review prose alone cannot complete a node.

## DL-150-10 Authority
Participation modes are `AUTO`, `SUPERVISED`, `AUTHORITY_REQUIRED`. Required authority is a transition gate, not a hard-coded workflow step.

## DL-150-11 Evidence/audit
No engineering state without engineering evidence. Material decisions produce a durable artifact or append-only auditable record. Runtime claims are never silently promoted to evidence.

## DL-150-12 Persistence slice
v1.5.0 defines a small project-state persistence port and a compatibility implementation sufficient for restart recovery. It must not perform the v1.5.1 SQLite/PostgreSQL redesign or DB-job-queue removal.

## DL-150-13 Deferred capabilities
Do not implement full SQLite migration/job queue removal (v1.5.1), SSH/Docker (v1.5.2), full Git authority automation (v1.5.3), full multi-Agent orchestration (v1.5.4), or automatic re-plan/learning (v1.5.5).

## DL-150-14 Validation carrier
No GitHub Actions and no CI system for v1.5.0. Explicit Local/Build Host execution carries validation evidence. Missing environment capability is `ENV-BLOCKED`, never fabricated PASS.

## DL-150-15 Parallel follow-up lanes
v1.5.1-v1.5.5 may proceed in parallel only against the shared contracts frozen by v1.5.0. Follow-up lanes may extend adapters/behavior but cannot redefine the contracts unilaterally.
