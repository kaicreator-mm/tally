# v1.5.4 L2 Architecture Evidence

## Scope verdict

**PASS for implementation compatibility.** v1.5.4 can be implemented as a bounded collaboration layer above v1.5.0 without replacing TaskDAG, AgentExecutor, TaskExecutionRequest/Result, internal/agentruntime or CompletionContract.

## Existing authorities reused

1. `engineering.ActiveTaskDAG` remains the single Project Flow state authority. Multi-agent state records role collaboration only; it does not define a second project/task lifecycle.
2. `taskexecution.AgentExecutor` remains the only task-level execution port. Roles create role-specific views of the same frozen TaskExecutionRequest contract.
3. `agentruntime.AgentRunResult` remains an Agent Claim. Reviewer/verifier PASS is derived from Tally EvidenceRefs, not from an Agent summary or completion claim.
4. `engineering.CompletionContract` remains the final task acceptance authority. A verified collaboration produces `NodeAccepted` facts; the existing Control Plane performs Accepted → Done only after CompletionContract passes.
5. Existing Local RuntimeExecutor and the current fake/codex/dsh runtime adapters are reused by the production `cmd/server` composition root.

## Added bounded components

- `internal/multiagent`: role model, deterministic role state machine, independence checks, evidence/finding provenance and bounded concurrency.
- scheduler: reads TaskDAG readiness and conflict metadata; it does not persist a separate queue.
- Control Plane optional state: `MultiAgentPolicies` and `MultiAgentWorkflows` inside ProjectSnapshot.
- optional Control Plane HTTP endpoints for role policy, single collaboration dispatch and ready-task batch dispatch.
- production composition: the Multi-Agent Engine receives the same Local RuntimeExecutor already used by the Brownfield Engineering Harness.

## Architecture risks found during review and resolutions

### R1 — stale implementation branch

The first implementation branch became 44 commits behind `main`. It was not merged. The implementation was re-layered onto the then-current main baseline in `dev/v1.5.4-multi-agent-workflow-r3`, preventing unrelated stale code from entering the version.

### R2 — CompletionContract state mismatch

An early implementation produced `NodeComplete` facts. Existing v1.5.0 CompletionContracts expect `NodeAccepted`. Fixed: collaboration verification now produces `NodeAccepted`; existing `finishAccepted` owns TaskDAG Accepted → Done.

### R3 — queued cancellation / restart race

A collaboration originally registered active only after acquiring its concurrency slot. Fixed: it registers before waiting, so Restart/Cancel can stop queued work. Late finalization preserves a Task already restored to Ready by `RestartTask` and records the collaboration as interrupted.

### R4 — legacy TaskRunning hijack

A new process could otherwise mistake an unrelated persisted Running task for a recoverable multi-agent collaboration. Fixed: Running recovery is allowed only when durable `MultiAgentWorkflow` state already exists for that task.

### R5 — runtime independence label spoofing

Role policy can request runtime/provider independence. A RuntimeRef label alone is not evidence that a different runtime executed. Fixed at the existing RuntimeExecutor boundary: request `Runtime.ID` must equal the executor's actual `RuntimeDescriptor.Name` before any Agent session starts. A policy asking for a different runtime on a single-runtime deployment therefore fails closed instead of pretending independence.

## Non-goals preserved

No generic BPM/workflow DSL, Agent conversation graph, chain-of-thought persistence, distributed worker/Job Queue, remote Agent fleet or unlimited Agent spawning was introduced.

## Parallel-version compatibility

Core behavior uses Local execution and Tally-native persisted state. v1.5.2 SSH/Docker and v1.5.3 Git authority are not implementation prerequisites. The frozen PRD still declares v1.5.3 as the **release baseline dependency**, so v1.5.4 may complete implementation/PR work before v1.5.3 but cannot be marked Release Qualified/tagged ahead of that baseline.
