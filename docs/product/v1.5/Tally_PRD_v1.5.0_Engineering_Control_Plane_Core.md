# Tally PRD v1.5.0 — Engineering Control Plane Core

- Status: **FROZEN EXECUTABLE SCOPE**
- Base: v1.4.1
- Purpose: establish the correct Engineering Control Plane domain skeleton without simultaneously replacing every infrastructure adapter.

## Goal

Make Tally own the software-engineering lifecycle from Requirements through an active, validated TaskDAG, with explicit completion evidence and authority.

## In scope

1. Requirements Analysis and Requirements Baseline.
2. Project Plan and PlanningRevision.
3. Architecture / Engineering Design artifact and constraints.
4. `TaskDAGProposal` → Engineering Harness analyze/evaluate/validate → `Active TaskDAG`.
5. `Active TaskDAG = Project Flow`; no second Project Flow state source.
6. Persist TaskNode, dependency, readiness, block reason, parallelism eligibility and task lifecycle state.
7. EngineeringNode common model.
8. CompletionContract with required artifacts, evidence, records and authority.
9. ArtifactRef, EvidenceRef and append-only Action/Audit record concepts.
10. Participation modes: `AUTO`, `SUPERVISED`, `AUTHORITY_REQUIRED`.
11. Intelligence Provider boundary with Direct LLM and AI Runtime adapters implementing the same intelligence contracts.
12. Engineering Harness control loop for validate → ready → execute existing task path → verify/review → accept/repair/escalate → advance/block.
13. Preserve the current Agent runtime boundary; do not create a second Runtime abstraction.
14. Keep current persistence/execution infrastructure only as a transitional implementation detail where required for this release.

## Required artifacts

At minimum: Requirements baseline, Project Plan/PlanningRevision, Architecture note/ADR/document, TaskDAG baseline, TaskContract, Verification result, Review result and Acceptance record. Human-readable and machine-readable forms may coexist.

## Intelligence rule

Every intelligence-assisted function in this release uses the same `IntelligenceProvider` contract with Direct LLM and AI Runtime implementations. AI Runtime may improve quality but may not own persistent engineering state.

## Human authority

Node policy determines AUTO, SUPERVISED or AUTHORITY_REQUIRED. Scope-changing requirement decisions and high-impact architecture decisions must support an authority gate.

## Out of scope

SQLite default/PostgreSQL adapter migration; default DB Job Queue removal; SSH/Docker execution; full GitHub Issue/PR/Merge automation; full multi-Agent role workflow; automated re-plan and rule learning.

## Acceptance criteria

- Project progresses Intent → Requirements Baseline → Project Plan → Architecture → validated Active TaskDAG.
- Invalid/cyclic/incomplete DAG proposals are rejected before activation.
- Dependency/readiness state persists across Tally restart.
- CompletionContract blocks Task completion when required artifact/evidence/record is missing.
- Direct LLM and AI Runtime satisfy the same intelligence contract for implemented intelligence nodes.
- Human authority can block a transition that would otherwise pass.
- Active TaskDAG is the single Project Flow state authority.
- Existing Agent execution remains usable without a new Agent runtime stack.

## Hidden Validation

Ambiguous requirements, missing AC, architecture constraint violation, DAG cycle, missing dependency, blocked→ready, parallel-ready tasks, missing artifacts/evidence, authority-required transition, provider substitution and restart state recovery.

## Release qualification

The lifecycle runs end-to-end using transitional current infrastructure, all required artifacts/evidence are traceable, and no deferred v1.5.1+ capability is falsely claimed complete.
