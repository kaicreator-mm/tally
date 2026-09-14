# Tally PRD v1.5.4 — Multi-Agent Engineering Workflow

- Status: **FROZEN EXECUTABLE SCOPE**
- Implementation dependency: **v1.5.0 TaskDAG + AgentExecutor + EngineeringNode contracts only**
- Release baseline dependency: v1.5.3
- Parallel lane: may be implemented concurrently with v1.5.1, v1.5.2, v1.5.3 and v1.5.5; SSH/Docker and full Git authority are not prerequisites for its core implementation.

## Goal

Support multiple strong Agents cooperating through engineering roles, TaskDAG dependencies and evidence without turning Tally into a generic Agent workflow engine.

## In scope

1. Engineering roles: implementer, reviewer, repairer, verifier and optional specialists.
2. Role requirements attach to Task/Engineering Gate, not Agent thought processes.
3. Sequential flow: Implement → Review → Repair → Verify → Accept.
4. Parallel collaboration where DAG dependencies/conflict analysis permit it.
5. Independent review/verification may require a different Agent/run/provider from implementer.
6. Bounded concurrency governed by readiness, execution capability availability and policy.
7. Role-specific TaskContract/Execution Context views.
8. Evidence/findings move between roles through durable artifacts/records.
9. Failures return to Engineering Harness for repair, reassign, block or authority escalation.
10. All role execution uses the shared v1.5.0 AgentExecutor/request/result contract.
11. Restart remains Task-level; no conversation/thought recovery.

## Parallel-development rule

Core multi-Agent behavior must work using Local execution and Tally-native action records. v1.5.2 environment adapters and v1.5.3 Git authority integrate later through frozen contracts and are not blockers for implementation.

## Non-goals

Generic BPM/workflow DSL, arbitrary Agent conversation graph, persistent chain-of-thought, distributed worker fleet or Job Queue as collaboration model.

## Acceptance criteria

- Implementer→reviewer→repairer→verifier lifecycle works.
- Independent reviewer/verifier constraints are enforceable.
- Two ready non-conflicting Tasks can run in parallel under bounded concurrency.
- Dependency-blocked Tasks cannot run early.
- Findings/evidence identify producing role/run.
- Acceptance occurs only after role/gate requirements are met.
- Restart restores semantic TaskDAG/role state without Agent reasoning.

## Hidden Validation

Reviewer no issue, repair request, repair failure, verifier disagreement, role provider unavailable, parallel file conflict, dependency race, restart mid-collaboration and concurrency enforcement.

## Release qualification

Sequential and parallel multi-Agent critical journeys PASS; independent-review policy PASS; no generic workflow/queue dependency introduced.