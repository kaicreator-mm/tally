# Tally Current Codebase Architecture

**Current version program:** v1.5.0 Engineering Control Plane Foundation  
**Brownfield base:** v1.4.1  
**Current qualification:** development baseline / **NOT RELEASE QUALIFIED**

This document describes the current source tree. Historical v1.4/v1.4.1 target documents remain evidence but do not override the frozen v1.5.0 authority under `docs/product/v1.5/` and `docs/tasks/v1.5.0/`.

## 1. Authority model

Tally separates proposal/claim from facts and state:

```text
LLM / IntelligenceProvider     → Proposal / Assessment / Recommendation
Agent Runtime                  → Agent Claim
Validators / verification      → Evidence / machine facts
Human authority                → governed judgment
CompletionContract             → deterministic completion evaluation
Active TaskDAG                 → Project Flow state authority
```

`Active TaskDAG = Project Flow`. No second project-flow state machine is introduced by v1.5.0.

## 2. Project lifecycle domain — `internal/engineering`

The new v1.5 domain freezes common values used by later slices:

- `RequirementsBaseline` / `Requirement`;
- `ProjectPlan` / immutable-style `PlanningRevision`;
- `ArchitectureRevision` / `ArchitectureConstraint`;
- `TaskDAGProposal` / `ActiveTaskDAG` / `TaskNode`;
- deterministic dependency validation/readiness/parallel-ready behavior;
- `ArtifactRef`, `EvidenceRef`, append-only `ActionRecord` / `AuditLog`;
- participation / `AuthorityRequirement` / `AuthorityDecision`;
- deterministic `CompletionContract` / `CompletionFacts` / structured missing requirements.

The domain does not trust LLM/Agent output as completion evidence.

## 3. Intelligence boundary — `internal/intelligence`

`Provider` is the v1.5 intelligence seam:

```text
Provider.Generate(ctx, Request) → Result{
  Proposal,
  Assessment,
  Recommendation,
  ProviderProvenance,
}
```

Direct LLM and AI Runtime use the same contract through small invoker adapters. Provider provenance is informational/traceable; provider output cannot mutate Active TaskDAG or satisfy CompletionContract directly.

Existing `internal/llm` and `internal/airuntime` remain transport/inference boundaries, not project controllers.

## 4. Task execution contracts — `internal/taskexecution`

The v1.5 task-level execution seam is intentionally thin:

```text
Engineering Harness
      ↓
AgentExecutor
      ↓
Agent Runtime × Execution Environment
```

Key values:

- `ExecutionEnvironment` / `EnvironmentKind`;
- `ExecutionTarget` / runtime and environment refs;
- environment-aware `WorkspaceRef`;
- runtime/environment capability requirements;
- `TaskExecutionRequest` carrying Frozen TaskContract + Engineering Context + runtime/environment/workspace/policy/instruction;
- `TaskExecutionResult` carrying Agent Claim, candidate reference and execution outcome.

Only Local execution is implemented for v1.5.0. SSH/Docker environment implementations are v1.5.2 scope.

Cancellation/interruption is distinct from candidate failure. Correctness does not depend on Agent session resume.

## 5. Single Agent Runtime authority — `internal/agentruntime`

The existing protocol remains the one Agent Runtime abstraction. Its minimal boundary is:

- Runtime `Descriptor` / `Start`;
- Session `Execute` / `Cancel` / `Events` / `Result` / `Close`;
- optional features through capability negotiation.

`runtimes/codex` and `runtimes/dsh` remain out-of-process adapters behind this same interface. v1.5.0 does not create a runtime-name-specific project authority.

AgentRunResult is explicitly an Agent Claim.

## 6. Engineering Control Plane — `internal/controlplane`

The project-level Control Plane owns accepted lifecycle stages and the Active TaskDAG:

```text
Requirements accepted
→ Plan accepted
→ Architecture accepted
→ TaskDAG validated + activated
→ ready node
→ dispatch
→ Harness accepted / blocked / interrupted
→ CompletionContract
→ accepted + done
→ dependent readiness refresh
```

### Persistence

`Repository` is a consumer-owned, DB-neutral port. v1.5.0 ships a local file-snapshot compatibility adapter with atomic temp-write/rename behavior sufficient for restart proof. The v1.5.1 storage redesign may replace this adapter without changing the project-domain contract.

This compatibility state does not replace the existing PostgreSQL append-only fact stores used by the brownfield execution/verifier path.

### API/UI

`internal/controlplane/http.go` exposes additive `/api/v15/...` JSON operations for snapshot, lifecycle stages, TaskDAG activation, authority, restart and dispatch.

The closure candidate adds `/v15/control-plane`, a thin browser projection. It owns no state; it reads snapshot data and writes through the same JSON operations. `internal/web` mounts both API and console inside the existing authentication + CSRF security boundary.

## 7. Brownfield Engineering Harness

Production Local dispatch uses `BrownfieldHarness`. It composes existing authorities rather than reimplementing them:

```text
Control Plane / Ready Task
  ↓
BrownfieldHarness
  ↓
artifacts.LoadBundle(exact revision)
  ↓
CompileAuthority
  ↓
existing workspace.Manager / closedloop.WorkspaceExecutor
  ↓
Tally-owned disposable linked Git worktree
  ↓
AgentExecutor → existing agentruntime.Runtime
  ↓
Agent Claim
  ↓
FreezeWorktree → immutable candidate
  ↓
runstore CodegenRun + durable ChangeSet
  ↓
existing candidateverify.Service
  ↓
existing closedloop.NewLoop
  ├─ verify
  ├─ safe RepairFeedback
  ├─ bounded repair
  ├─ supersession / exposure lineage
  └─ accept / escalate
  ↓
Accepted Basis → CompletionFacts
  ↓
CompletionContract → Active TaskDAG
```

### Candidate/workspace safety

- executable changes are produced only in Tally-owned disposable linked worktrees;
- the owner/main checkout is rejected by the candidate-freeze boundary;
- modified/untracked full post-change content becomes the immutable candidate;
- no-change Agent completion produces no candidate;
- deletion is currently not representable by the full-content ChangeSet model and therefore fails closed;
- candidate identity includes its durable producer Run, so rebinding exact content from closed-loop attempt identity to the database CodegenRun produces the authoritative ChangeSet identity before verification.

### Held-out separation

When a frozen verification contract requires held-out execution, production requires:

- configured external held-out root;
- runtime capability `session.isolation` only after independent conformance evidence has gated that declaration.

Repair agents receive only the existing allowlisted safe `RepairFeedback` projection. Hidden fixture/oracle content is not part of the implementation authority.

## 8. Existing brownfield authorities retained

v1.5.0 intentionally reuses important v1.4/v1.4.1 packages:

- `internal/contract` — compiled contract authority;
- `internal/changeset` — immutable candidate identity/model;
- `internal/workspace` — disposable Git worktrees;
- `internal/candidateverify` / `internal/verification` — independent candidate verification;
- `internal/closedloop` — verification-driven repair / accept / escalation;
- `internal/runstore`, `internal/outcome`, `internal/evidence` — append-only machine facts/lineage;
- `internal/workflow` — bounded existing single-task behavior where still used; not promoted to a second project workflow.

## 9. Storage truth in v1.5.0

There are two intentionally different persistence concerns during this foundation version:

1. **v1.5 project-control compatibility snapshot** — DB-neutral Repository / file adapter for Requirements/Plan/Architecture/Active TaskDAG/Authority/Audit restartability.
2. **existing brownfield machine-fact stores** — PostgreSQL-backed CodegenRun, ChangeSet, verification/evidence/exposure/supersession/outcome facts.

Unification/redesign is explicitly v1.5.1 scope. v1.5.0 must not silently migrate schemas or queues.

## 10. Validation and release boundary

No GitHub Actions or CI system is a v1.5.0 validation authority. Explicit Local / Build Host carriers are:

```bash
bash scripts/validate-v1.5.0.sh
TALLY_V15_HIDDEN_ROOT=/secure/held-out/tally-v1.5.0 bash scripts/run-v1.5-hidden.sh
```

See `docs/tasks/v1.5.0/VALIDATION_MATRIX.md`, `CRITICAL_JOURNEYS.md`, `HIDDEN_VALIDATION_PROTOCOL.md` and `RELEASE_QUALIFICATION.md`.

Required closure-candidate Full Regression, real Local brownfield Critical Journey and Hidden Validation are currently `ENV-BLOCKED` in the ChatGPT Web environment; therefore the current source is **NOT RELEASE QUALIFIED**.

## 11. Deferred slices

The current architecture freezes shared seams but does not implement later capabilities:

- v1.5.1 storage redesign / DB queue removal;
- v1.5.2 SSH/Docker environments;
- v1.5.3 full GitHub engineering authority automation;
- v1.5.4 full multi-Agent role workflow;
- v1.5.5 automatic governed re-plan / learning.
