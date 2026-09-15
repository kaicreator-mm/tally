# Tally Documentation & Implementation Status

**Current program:** v1.5.0 Engineering Control Plane Foundation  
**Updated:** 2026-09-15  
**Development mode:** Brownfield evolution  
**Release qualification:** **NOT RELEASE QUALIFIED**

## Executive status

```text
v1.4 implementation                         COMPLETE — historical baseline
v1.4.1 closure-repaired baseline            COMPLETE — brownfield base
v1.5.0 authority / L2 / Task DAG            COMPLETE — Issue #5
v1.5.0 engineering domain / Active TaskDAG  COMPLETE — Issue #6
v1.5.0 intelligence / execution contracts   COMPLETE — Issue #7
v1.5.0 brownfield Engineering Harness        COMPLETE — Issue #8
v1.5.0 minimal Control Plane API             IMPLEMENTED
v1.5.0 minimal Control Plane browser UI      IMPLEMENTED on closure candidate
v1.5.0 closure evidence/artifacts            IMPLEMENTED on closure candidate
v1.5.0 full visible regression               ENV-BLOCKED — private checkout / Build Host unavailable here
v1.5.0 real Local Critical Journeys          ENV-BLOCKED — runtime/PostgreSQL/Build Host unavailable here
v1.5.0 Hidden Validation                     ENV-BLOCKED — external held-out pack unavailable here
v1.5.0 Release Qualification                 NOT COMPLETE
```

Issue #9 remains the closure/release-evidence authority. A focused or PR-level PASS does not imply Release PASS.

## Current implementation facts

The current source implements the v1.5.0 Engineering Control Plane Foundation on top of the v1.4.1 brownfield system:

- Requirements Baseline → Project Plan / immutable PlanningRevision → Architecture → validated Active TaskDAG;
- `Active TaskDAG = Project Flow`, with deterministic dependency/readiness and restart persistence;
- typed Artifact/Evidence/Audit/Authority values and deterministic CompletionContract;
- common IntelligenceProvider contract for Direct and AI Runtime adapters;
- thin AgentExecutor and orthogonal Runtime × ExecutionEnvironment contracts;
- Local execution compatibility for v1.5.0;
- project-level Control Plane API and minimal browser console;
- production BrownfieldHarness over the existing `internal/agentruntime`, `internal/closedloop` and `internal/candidateverify` authorities;
- Tally-owned disposable linked worktrees and owner-checkout rejection;
- immutable ChangeSet candidate freeze plus durable CodegenRun/ChangeSet lineage before verification;
- task-level interruption/restart correctness independent of Agent session resume.

Agent or LLM output never becomes Project Fact merely because it claims completion. Task advancement still requires independent accepted facts and CompletionContract satisfaction.

## Runtime / execution authority

```text
Engineering Control Plane
  ↓
TaskHarness / AgentExecutor
  ↓
internal/agentruntime.Runtime  ×  ExecutionEnvironment
  ├─ runtimes/codex               Local implemented in v1.5.0
  └─ runtimes/dsh
  ↓
existing closedloop.WorkspaceExecutor
  ↓
immutable ChangeSet → candidateverify → existing closedloop
  ↓
CompletionFacts → CompletionContract → Active TaskDAG
```

There is no second v1.5 Agent Runtime or second verification/repair/accept loop.

## UI / API

The v1.5 JSON API is mounted under `/api/v15/...`. The closure candidate also exposes `/v15/control-plane` as a thin browser projection over the same API/state. Both are mounted inside the existing authenticated + CSRF-protected web group.

The browser page does not own lifecycle state; it reads the persisted Control Plane snapshot and performs writes through the existing Authority / Restart / Dispatch API operations.

## Validation carrier

v1.5.0 explicitly forbids GitHub Actions and any CI system. Required evidence must be produced by explicit Local / Build Host commands.

Visible carrier:

```bash
bash scripts/validate-v1.5.0.sh
```

Held-out carrier after visible implementation freezes:

```bash
TALLY_V15_HIDDEN_ROOT=/secure/held-out/tally-v1.5.0 \
  bash scripts/run-v1.5-hidden.sh
```

Exact evidence/status is maintained in:

- `docs/tasks/v1.5.0/VALIDATION_MATRIX.md`
- `docs/tasks/v1.5.0/CRITICAL_JOURNEYS.md`
- `docs/tasks/v1.5.0/HIDDEN_VALIDATION_PROTOCOL.md`
- `docs/tasks/v1.5.0/RELEASE_QUALIFICATION.md`

## Release boundary

Current closure implementation may become the continuing **v1.5.0 development baseline** after review/merge. It must not be tagged or described as Release Qualified while required Full Regression / Critical Journeys / Hidden Validation remain `ENV-BLOCKED` or `NOT_RUN`.

The following remain explicitly deferred and must not be inferred complete from v1.5.0:

- v1.5.1 storage redesign (SQLite default / PostgreSQL optional) and DB queue removal;
- v1.5.2 SSH/Docker execution environments;
- v1.5.3 full GitHub Issue/PR/Merge/Tag/Release authority automation;
- v1.5.4 full multi-Agent workflow;
- v1.5.5 automatic re-plan / governed learning.

Historical v1.4/v1.4.1 documents and evaluation records remain retained as brownfield evidence, not as current version authority.
