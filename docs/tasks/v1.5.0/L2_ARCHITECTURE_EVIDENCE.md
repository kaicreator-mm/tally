# v1.5.0 L2 Architecture Evidence

Issue: #5
Baseline: `71a8f67de512495149e2e1f12e0a08c10e52fcff`
Standard: ai-development-standard 1.2.1 @ `e836f7a4e1f1c7fcf4530a811af235c8c8dba8a1`

## 1. Brownfield findings

The audited source already has strong single-task execution seams:
- `internal/agentruntime` is a runtime-neutral Runtime/Session protocol with capability negotiation. `StartRequest` currently carries a local workspace string; v1.5.0 must not replace the protocol, but the new task-level contract must stop propagating that local-path assumption into Project Core.
- `internal/workflow/v13.go` is a large bounded single-task workflow; it is retained and is not promoted into a Project-level workflow DSL.
- `internal/closedloop` already owns Execute → Verify → Repair → re-Verify → Accept/Escalate and has explicit acceptance guards. It is reused behind ports rather than copied.
- `internal/artifacts`, `contract`, `engineeringpolicy`, `risk`, `assurance`, `verification`, `changeset`, workspace/git and runtime adapters are retained authorities.
- `internal/environment` already describes Tally-managed application/deployment environments. It is not reused as the v1.5 Agent process ExecutionEnvironment domain because that would conflate deployment lifecycle with coding-agent execution context.
- web handlers already expose projects, requirements, tasks, execution state and v1.4 launch/task surfaces. v1.5 adds a bounded API/read-model surface rather than a frontend rewrite.

## 2. Package decisions

### `internal/engineering`
Owns project-level typed engineering domain values only:
RequirementsBaseline, PlanningRevision/ProjectPlan, ArchitectureRevision/constraints, ArtifactRef/EvidenceRef/ActionRecord, AuthorityRequirement/Decision, EngineeringNode/CompletionContract and TaskDAG.

This is intentionally not a generic workflow framework. Its state transitions are explicit software-engineering states.

### `internal/intelligence`
Owns the consumer-side `Provider` contract and typed request/result. Direct and AI Runtime are adapters through small injected invoker interfaces; provider identity is provenance only. No provider can decide deterministic gates.

### `internal/taskexecution`
Owns the thin task execution port above `internal/agentruntime`: `AgentExecutor`, `ExecutionEnvironment`, `ExecutionTarget`, `WorkspaceRef`, capability requirement, TaskExecutionRequest/Result and interruption semantics. Local adapter maps Local WorkspaceRef to the existing `agentruntime.StartRequest.Workspace`; this is the compatibility edge where the legacy path string remains allowed.

### `internal/controlplane`
Owns Project-level Engineering Harness orchestration and a small `Repository` port for accepted project state. It coordinates engineering domain state and task dispatch but delegates candidate execution to taskexecution and verification/repair/acceptance to existing brownfield services through narrow ports.

## 3. Requirements / Planning / Architecture placement

They live in `internal/engineering`, not in `internal/workflow`, `internal/llm` or provider-specific packages. Revisions are immutable values with identity/source/provenance. AI may propose content, but validation/freeze creates accepted domain state.

## 4. Active TaskDAG persistence

`controlplane.Repository` stores/reloads an accepted ProjectSnapshot containing revision identities, Active TaskDAG and append-only action records. v1.5.0 supplies a deterministic filesystem compatibility store under a caller-provided state path for Local use and tests. The port is deliberately DB-neutral so v1.5.1 can implement SQLite/PostgreSQL adapters without changing domain contracts. This is not the target v1.5.1 persistence redesign.

## 5. Task state and existing workflow/closedloop

TaskDAG node state is Project Flow truth. Existing workflow/closedloop state is execution/verification evidence for one Task and is projected into a task transition by the Engineering Harness. It never becomes a second Project Flow source.

## 6. `internal/workflow`

Retain as existing single-task behavior. Do not add Project DAG ownership there. New v1.5 orchestration calls bounded task execution services through ports.

## 7. `internal/closedloop`

Retain as Task Engineering Workflow for candidate verification, bounded repair/escalation and acceptance. v1.5 does not duplicate its ChangeSet or verification semantics. The control plane consumes a typed outcome/evidence projection from this seam.

## 8. Project-level DAG orchestration

Lives in `internal/controlplane`, using deterministic functions from `internal/engineering`. Scheduler logic selects only ready nodes and respects bounded concurrency/capability policy; it does not implement a generic distributed job scheduler.

## 9. AgentExecutor reuse

`taskexecution.RuntimeExecutor` wraps exactly one `agentruntime.Runtime`; it starts a Session, executes one bounded task instruction, captures AgentRunResult as an Agent claim, maps cancellation/interruption, closes the session and returns. It cannot verify or accept the result.

## 10. ExecutionEnvironment seam

`taskexecution.ExecutionEnvironment` prepares/resolves process/workspace context independently from Agent Runtime. `LocalEnvironment` is the only real v1.5.0 implementation. SSH/Docker kinds are representable as typed identifiers but have no implementation and must return unsupported when used.

## 11. WorkspaceRef

WorkspaceRef contains Environment, Target, Identity/Path and BaseRevision. Only `LocalEnvironment.Resolve` converts it to a local path. Engineering/controlplane packages never open files through WorkspaceRef directly.

## 12. IntelligenceProvider migration

`internal/intelligence.Provider` is consumer-owned. Direct and AIRuntime adapters accept narrow invoker interfaces so existing `internal/llm` and `internal/airuntime` can be connected without provider SDK leakage. A provider returns Proposal/Assessment/Recommendation plus provenance. Domain names such as AIRuntimeTask/Plan/Workflow are prohibited.

## 13. CompletionContract alignment

Existing contract/policy/verification/evidence subsystems continue to create task-specific acceptance and verification facts. v1.5 CompletionContract is the Project/node gate: it checks the presence/status of required ArtifactRef, EvidenceRef, checks, records, authority decisions and Git-operation evidence. It does not replace `contract.Compiled` or risk-adjusted acceptance.

## 14. Human Authority

Authority types live in `internal/engineering`. `CompletionContract.Evaluate` and legal transition functions enforce pending/rejected decisions. UI/API can submit AuthorityDecision records through controlplane; no fixed mandatory human step is inserted.

## 15. Existing packages retained

Retain without parallel replacements: `agentruntime`, `closedloop`, `workflow`, `artifacts`, `changeset`, `contract`, `engineeringpolicy`, `risk`, `assurance`, `verification`, `workspace`, Git writer/integration, outcomes and current runtime adapters.

## 16. Deferred migrations

v1.5.0 does not perform SQLite/Postgres redesign, DB queue removal, SSH/Docker execution, full GitHub authority automation, multi-Agent role orchestration or automatic re-planning/learning. Only stable seams/types needed by these lanes are frozen.

## 17. Backward compatibility

Existing Codex/DSH paths remain valid because `agentruntime` is unchanged. The Local taskexecution compatibility adapter converts WorkspaceRef at the edge. Existing v1.4 web/task paths remain mounted. v1.5 APIs are additive.

## 18. Parallel-lane guarantee

v1.5.1 depends on `controlplane.Repository`; v1.5.2 on taskexecution environment/workspace contracts; v1.5.3 on EngineeringNode/CompletionContract/GitRequirement; v1.5.4 on Active TaskDAG + AgentExecutor; v1.5.5 on immutable PlanningRevision + DAG revision/evidence/authority. None requires redefining another lane's contract.

## 19. Failure model

Deterministic invalid state returns typed errors. Provider/runtime/environment failures are operational facts and cannot be converted to candidate-verification failures. Unsupported remote environments and unavailable resume are explicit. Append-only records are never rewritten.

## 20. Validation architecture

Visible tests cover domain contracts, DAG validation/readiness, completion/authority, provider conformance, Local AgentExecutor/cancel/interruption and snapshot restart. Hidden validation executes only after visible candidate freeze. For this project, every command is run explicitly Local/Build Host; GitHub Actions/CI is not a validation carrier.
