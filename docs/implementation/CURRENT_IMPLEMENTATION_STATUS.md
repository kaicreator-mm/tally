# Tally Current Implementation Status

**Current program:** v1.5.0 Engineering Control Plane Foundation  
**Base:** v1.4.1 brownfield implementation  
**Status:** implementation foundation complete through Issue #8; Issue #9 closure candidate in progress  
**Release:** **NOT RELEASE QUALIFIED**

## Summary

The repository has moved beyond the v1.4.1 development baseline. v1.5.0 now establishes the project-level Engineering Control Plane, Active TaskDAG authority and shared intelligence/execution seams while deliberately reusing the existing Agent Runtime, ChangeSet, verification and closed-loop authorities.

Completed implementation slices:

- #5 — v1.5 authority, L2 architecture and executable task program;
- #6 — engineering lifecycle domain, TaskDAG, CompletionContract, authority/audit model;
- #7 — common IntelligenceProvider plus AgentExecutor × ExecutionEnvironment contracts;
- #8 — restartable Control Plane, Local production composition and brownfield closed-loop execution bridge.

Issue #9 owns minimal UI/API completion, validation, Critical Journeys, Hidden Validation and Release Qualification. The closure branch contains those artifacts, but required release-level executable evidence remains unavailable in the current ChatGPT Web environment.

## Implemented — engineering lifecycle

`internal/engineering` contains:

- RequirementsBaseline with required statement + acceptance criteria;
- ProjectPlan / PlanningRevision;
- ArchitectureRevision / ArchitectureConstraint;
- TaskDAG proposal validation and ActiveTaskDAG;
- deterministic dependency/readiness and legal transitions;
- ArtifactRef / EvidenceRef;
- append-only ActionRecord / AuditLog;
- human participation/authority values;
- CompletionContract / CompletionFacts / structured missing requirements.

`Active TaskDAG = Project Flow`. v1.5.0 does not persist a second project-flow authority.

## Implemented — intelligence/execution contracts

`internal/intelligence`:

- shared Provider contract;
- Direct and AI Runtime adapters via a common invoker shape;
- Result = Proposal + Assessment + Recommendation + provider provenance;
- malformed/missing structured result is rejected;
- provider failure does not fabricate a result or own project state.

`internal/taskexecution`:

- AgentExecutor;
- RuntimeRef / EnvironmentRef;
- ExecutionEnvironment and LocalEnvironment;
- environment-aware WorkspaceRef;
- runtime/environment capability requirements;
- TaskExecutionRequest / TaskExecutionResult;
- explicit cancellation/interruption;
- bounded resume/fallback behavior.

Only Local environment execution is implemented in v1.5.0.

## Implemented — Control Plane

`internal/controlplane` owns project lifecycle and task dispatch over the persisted Active TaskDAG.

Implemented behavior:

- ordered acceptance Requirements → Plan → Architecture;
- validated TaskDAG activation only after lifecycle prerequisites;
- restartable compatibility snapshot repository;
- readiness refresh and task transitions;
- append-only project audit;
- authority decisions scoped to a task;
- task restart independent of Agent-session resume;
- Dispatch through a production TaskHarness;
- CompletionContract still gates advancement after the Harness reports accepted.

JSON API operations exist under `/api/v15/...`.

The closure candidate adds `/v15/control-plane`, a thin browser console for lifecycle/DAG/audit observation plus Authority/Restart/Dispatch actions. The page owns no state and is mounted behind the existing authentication + CSRF middleware.

## Implemented — brownfield production bridge

`BrownfieldHarness` is the v1.5.0 production task-level composition. It does not create a second workflow/runtime stack.

For each Local task run it:

1. loads one exact/pinned Git execution bundle;
2. checks project/task authority and compiles existing contract authority;
3. checks held-out prerequisites before runtime execution;
4. creates the existing workspace manager / closed-loop run table;
5. uses existing `closedloop.NewLoop`;
6. delegates coding attempts to the new AgentExecutor but forces the workspace to the disposable worktree created by the existing WorkspaceExecutor;
7. freezes actual modified/untracked content into an immutable ChangeSet candidate;
8. records a durable CodegenRun and rebinds exact candidate content to that database run identity;
9. persists the authoritative ChangeSet before independent candidate verification;
10. reuses existing candidateverify and closedloop verification/repair/acceptance;
11. projects only an actual closed-loop Accepted Basis to CompletionFacts;
12. lets CompletionContract and project authority decide whether Active TaskDAG advances.

Operational/runtime/provider failure remains distinct from candidate verification failure.

## Implemented safety properties

- Agent `completed` is not Evidence and is insufficient to advance by itself.
- executable candidate work happens in Tally-owned linked worktrees;
- owner primary checkout is rejected by FreezeWorktree;
- no-change completion produces no candidate;
- unsupported deletion representation fails closed;
- repair attempt correctness does not depend on session resume;
- safe RepairFeedback remains bounded/allowlisted;
- held-out execution requires external held-out material plus evidence-gated session isolation;
- supersession/exposure/outcome facts reuse append-only brownfield stores;
- Codex/DSH remain behind the one existing Agent Runtime protocol.

## Validation evidence

Actually executed focused evidence is recorded in `docs/tasks/v1.5.0/VALIDATION_MATRIX.md`:

- engineering domain / TaskDAG / Completion focused test, vet and race PASS from PR #11;
- IntelligenceProvider / taskexecution focused test, vet and race PASS from PR #12;
- immutable real-Git worktree freeze test, vet and race PASS from PR #16 isolated harness;
- PR #16 manual contract/interface review PASS.

The closure branch also codifies a visible lifecycle Critical Journey and UI/API security tests. They have **not** been executed against the closure candidate in the current environment because the private repository is not available as an authenticated local checkout.

Current required release-level gates:

```text
closure-candidate focused/full tests    ENV-BLOCKED
full visible regression                 ENV-BLOCKED
integration/E2E                         ENV-BLOCKED
real Local Brownfield Critical Journey  ENV-BLOCKED
external Hidden Validation              ENV-BLOCKED
Release Qualification                   NOT COMPLETE
```

These are not inherited PASS.

## Explicit local validation carriers

```bash
bash scripts/validate-v1.5.0.sh

TALLY_V15_HIDDEN_ROOT=/secure/held-out/tally-v1.5.0 \
  bash scripts/run-v1.5-hidden.sh
```

No GitHub Actions or CI system is authorized for this version.

## Deferred / not implemented in v1.5.0

- SQLite-default/PostgreSQL-optional persistence redesign and DB Job Queue removal — v1.5.1;
- SSH/Docker ExecutionEnvironment implementations — v1.5.2;
- full GitHub Issue/PR/Merge/Tag/Release authority automation — v1.5.3;
- full multi-Agent role workflow — v1.5.4;
- automatic re-plan / governed engineering learning — v1.5.5.

## Current conclusion

The source is converging on a coherent **v1.5.0 development baseline**. It must remain **NOT RELEASE QUALIFIED** until one frozen candidate passes the required visible regression, Critical Journeys and external Hidden Validation described in `docs/tasks/v1.5.0/RELEASE_QUALIFICATION.md`.
