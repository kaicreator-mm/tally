# Tally v1.5 Frozen Architecture PRD

- Version line: v1.5.x
- Status: **FROZEN**
- Date: 2026-09-15
- Product: Software Engineering Harness / Engineering Control Plane for Coding Agents
- Base: Tally v1.4.1 Strong Development Baseline

## 1. Product principle

Tally adopts:

> **Strong Agent + Thin Task Harness + Strong Engineering Harness**

Tally is the Engineering Control Plane. It does not manage an Agent's internal reasoning or detailed implementation loop. It defines engineering correctness, governs project progression, schedules the runtime and execution environment under engineering constraints, verifies resulting changes, and controls whether those changes may progress.

Canonical statement:

> **Before task execution, Tally defines engineering correctness and schedules an appropriate execution environment. After execution, Tally verifies whether correctness was achieved and decides whether the change may enter the next engineering stage.**

## 2. Canonical lifecycle

```text
Project Intent
→ Requirements Analysis
→ Requirements Baseline
→ Project Plan
→ Architecture / Engineering Design
→ TaskDAG Proposal
→ Engineering Analysis / Evaluation / Validation
→ Active TaskDAG = Project Flow
→ Readiness / Dependency / Scheduling
→ TaskContract
→ Runtime + Execution Environment
→ Strong Agent
→ Candidate
→ Verification / Independent Review
→ Repair / Escalation / Acceptance
→ Git Integration
→ Advance / Block / Re-plan TaskDAG
→ Version Closure
→ Hidden Validation / Critical Journeys
→ Release Qualification
→ Tag / Baseline / Outcome / Learning
```

Requirements answer **what and why**. Project Plan describes the intended engineering strategy. Architecture defines important structural decisions and constraints. The Active TaskDAG is the executable Project Flow.

Tally must not maintain a second independent Project Flow state source beside the Active TaskDAG.

## 3. Engineering Harness owns the whole TaskDAG lifecycle

Engineering Harness is not a wrapper around a single Task. It governs Requirements and baseline freeze, Project Plan and Architecture validation, TaskDAG analysis/validation, dependency/readiness/parallelism, risk/assurance, TaskContract, runtime/environment scheduling, verification/review/repair/escalation, human authority, Git authority, TaskDAG advancement/blocking/re-planning, closure, release qualification and engineering learning.

AI output is never directly executable engineering truth. Direct LLM or AI Runtime may produce a proposal; Engineering Harness validates it before it becomes an accepted baseline or active state.

## 4. Intelligence Provider

```text
IntelligenceProvider
├── Direct LLM
└── AI Runtime
```

AI Runtime is an enhanced LLM service that can improve output quality, determinism and assurance through multi-model orchestration, critic/judge/consensus or escalation. It is not a project controller.

> **Any intelligence capability available through AI Runtime must also be available through Direct LLM mode.**

AI Runtime does not own Project state, TaskDAG state, scheduling state, acceptance authority or release authority. Deterministic engineering checks remain deterministic.

## 5. Human authority

Human participation is policy-driven, not hard-coded after every step.

- `AUTO` — Engineering Gate may advance automatically.
- `SUPERVISED` — normally automatic, but review/override is available and exceptions escalate.
- `AUTHORITY_REQUIRED` — cannot cross the gate without authorized human approval.

Risk, evidence quality, architecture impact, scope impact, security, production impact and policy determine the required mode.

## 6. Engineering Node contract

```text
EngineeringNode
├── Input
├── Action
├── Deterministic Logic
├── Intelligence (optional)
├── Human Authority (optional)
├── Artifact
├── Evidence
├── Git Operation (optional/required by policy)
├── Audit / Action Record
├── CompletionContract
└── State Transition
```

A node is complete only when its CompletionContract is satisfied: required state transition, artifacts, evidence, records, Git operations and authority.

> **No engineering state without engineering evidence.**

> **No major engineering decision without a durable artifact or auditable record.**

## 7. Durable outputs

Important nodes produce human-readable and/or machine-readable artifacts as appropriate: Requirements baseline, Project Plan/PlanningRevision, Architecture note/ADR/document, TaskDAG baseline, TaskContract, Verification result, Review result, Acceptance record, Closure/Release artifacts.

```text
Git               = durable versioned engineering baseline
Tally Persistence = live engineering state + audit
Evidence Store    = logs/reports/test artifacts/large evidence
```

## 8. Git is part of the engineering lifecycle

Material Git operations are formal engineering actions: Issue, Branch, Commit, Push, Pull Request, Review/Approve/Request Changes, Merge, Tag and Release.

> **Every material Git operation must be attributable to an engineering node, contract, evidence set and authority decision.**

Git operations may be required by a CompletionContract. A Task whose policy requires merge is not `DONE` merely because its code passed tests.

## 9. Thin Task Harness

Thin Task Harness receives a frozen TaskContract, engineering context, selected runtime, selected execution environment and workspace reference. It starts/cancels/waits for the Agent, captures events and the Agent claim/candidate, supports bounded concurrency and task-level restart, then returns control to Engineering Harness.

It does not own Project flow, verification authority, acceptance authority, generic workflow, Agent thoughts, conversation recovery or a generic distributed scheduler.

## 10. Runtime and execution environment are orthogonal

```text
Agent Runtime              Execution Environment
-------------              ---------------------
Codex                      Local
DSH                        SSH
Future Agent               Docker
                           Future Sandbox / Remote Host
```

Runtime defines how to interact with the Agent. Environment defines where and how the process/workspace executes. Workspace references must become environment-aware; Tally Core must not assume every workspace is a locally readable path.

## 11. Task-level restartability

Correctness derives from Frozen TaskContract + Git/workspace state + Engineering Context + Verification Evidence. Runtime session resume is optional and never required for correctness. Preserve the disposable worktree when possible; otherwise recreate it from the accepted base revision and rerun the bounded TaskContract.

## 12. Multi-Agent

Multi-Agent collaboration is an Engineering Workflow / TaskDAG concern, not a database queue concern. Tally controls roles, gates, dependencies, readiness and evidence. Agents control how they perform assigned work.

## 13. Persistence and execution infrastructure

Core semantics must not depend on PostgreSQL-specific behavior such as `pgx/pgtype`, `FOR UPDATE SKIP LOCKED`, leases, heartbeats, JSONB semantics, triggers as the sole invariant, DB-generated identity or DB time.

Target profile: SQLite default; PostgreSQL optional adapter. A database Job Queue is not Tally Core. A future distributed queue may exist only as an optional Distributed Execution Adapter.

## 14. Version slicing

1. **v1.5.0 Engineering Control Plane Core** — lifecycle/domain model, requirements→plan→architecture→validated TaskDAG, EngineeringNode, CompletionContract, artifacts/evidence/audit, intelligence-provider boundary, authority modes.
2. **v1.5.1 Persistence & Execution Simplification** — persistence ports, SQLite default, PostgreSQL adapter, remove PostgreSQL-specific core semantics and default DB Job Queue execution.
3. **v1.5.2 Execution Environments** — Runtime/Environment decoupling, Local/SSH/Docker, WorkspaceRef and capability-aware scheduling.
4. **v1.5.3 Git Engineering Authority** — formal Issue/Branch/Commit/Push/PR/Review/Merge/Tag/Release linkage and authority.
5. **v1.5.4 Multi-Agent Engineering Workflow** — engineering roles, independent review/verification, parallel and sequential collaboration.
6. **v1.5.5 Re-plan & Engineering Learning** — controlled re-plan revisions, impact analysis, outcome learning and governed rule promotion.

## 15. Non-goals for v1.5

Generic BPM/workflow DSL, Agent thought-state persistence, Agent checkpoint framework, conversation recovery, generic Job System as product core, Kubernetes-style Agent fleet management, HA/multi-region control plane, Redis/Kafka/NATS/SQS as baseline requirements, a second Agent Runtime abstraction, or a second Closed Loop/Rule Engine.

## 16. Release discipline

Each version follows:

```text
L2 Architecture Evidence
→ Task DAG
→ L3 Implementation Evidence
→ Issues / PR
→ CI
→ Version Closure
→ Hidden Validation / Critical Journeys / Packaging
→ Release Qualification
→ Tag / Baseline
```

**PR PASS ≠ Release PASS.**
