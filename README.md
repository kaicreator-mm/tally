# Tally

**Software Engineering Harness for Coding Agents**

Tally turns planning output and software-engineering practice into executable, evidence-gated control over coding agents.

```text
Requirements
  ↓
Project Plan / PlanningRevision
  ↓
Architecture
  ↓
TaskDAGProposal → validated Active TaskDAG
  ↓
Ready Task
  ↓
Engineering Harness
  ↓
AgentExecutor → existing Agent Runtime × Execution Environment
  ↓
immutable ChangeSet
  ↓
Independent Verification / Repair / Review
  ↓
CompletionContract / Human Authority
  ↓
Accept → Active TaskDAG advance
```

> Agent output is a Claim, not a Project Fact. Tally freezes engineering authority, runs executable changes in disposable workspaces, independently verifies immutable candidates, and advances Project Flow only when deterministic completion and authority gates are satisfied.

## Current repository state

**Current development program: v1.5.0 Engineering Control Plane Foundation.**

The source now contains the v1.5.0 lifecycle/domain/execution foundation on top of the v1.4.1 brownfield implementation. It is suitable as the continuing **development baseline**, but it is **NOT RELEASE QUALIFIED** because required closure-candidate Full Regression, real Local Critical Journeys and external Hidden Validation remain `ENV-BLOCKED` in the current ChatGPT Web environment.

No GitHub Actions or CI system is part of the v1.5.0 validation carrier. Validation is explicit Local / Build Host execution only.

Current stack:

- Go 1.23;
- PostgreSQL + pgx + sqlc + goose for existing fact stores;
- v1.5.0 DB-neutral/file-snapshot compatibility repository for Active TaskDAG/restart state (storage redesign deferred to v1.5.1);
- `net/http` + chi;
- Go templates + HTMX plus the thin v1.5 Control Plane console;
- Git-backed engineering artifacts;
- out-of-process Codex / DeepSeek Harness runtime adapters behind one `internal/agentruntime.Runtime` protocol.

## v1.5.0 implemented foundation

- Requirements Baseline, Project Plan / immutable PlanningRevision and ArchitectureRevision;
- validated `TaskDAGProposal → Active TaskDAG` with deterministic dependency/readiness and restart persistence;
- `Active TaskDAG = Project Flow`; no second Project Flow state authority;
- typed Artifact/Evidence/Audit/Authority models;
- deterministic CompletionContract over required state, artifacts, PASS evidence/checks, records, authority and Git operations;
- common IntelligenceProvider contract for Direct and AI Runtime adapters;
- thin AgentExecutor plus Runtime × ExecutionEnvironment contracts;
- Local execution compatibility for v1.5.0; SSH/Docker remain deferred;
- project-level Engineering Control Plane + protected JSON API;
- minimal protected browser console at `/v15/control-plane`;
- production brownfield bridge that reuses existing `internal/agentruntime`, `internal/closedloop` and `internal/candidateverify` rather than creating replacements;
- Tally-owned disposable linked worktrees and owner-checkout rejection;
- immutable/durable CodegenRun → ChangeSet → verification lineage;
- task-level interruption/restart semantics that do not depend on Agent session resume.

The v1.4.1 implementation remains the brownfield base and historical evidence. Its Contract/Risk/Assurance, ChangeSet, verification, closed-loop, runtime and append-only fact invariants continue to be reused where v1.5.0 intentionally builds on them.

## Engineering authority model

```text
LLM output       = Proposal / Assessment / Recommendation
Agent output     = Claim
Validator output = Evidence / Fact
Active TaskDAG   = Project Flow authority
Completion       = deterministic CompletionContract evaluation
Human            = Judgment / authority where policy requires
```

Neither an LLM nor an Agent can directly mark a task complete.

## Runtime integration

Tally embeds one Agent Runtime protocol, not runtime-specific project logic.

```text
Engineering Harness
  ↓
AgentExecutor
  ↓
internal/agentruntime.Runtime  ×  ExecutionEnvironment
  ├── runtimes/codex              Local in v1.5.0
  └── runtimes/dsh
```

For the production brownfield Local path:

```text
pinned Git execution bundle
→ CompileAuthority
→ disposable worktree
→ AgentExecutor / Agent claim
→ immutable durable ChangeSet
→ candidateverify
→ existing closedloop repair / accept / escalate
→ CompletionFacts
→ CompletionContract
→ Active TaskDAG advance/block
```

## Start here

### Coding agents

1. `AGENTS.md`
2. `docs/product/v1.5/Tally_PRD_v1.5_FROZEN.md`
3. `docs/product/v1.5/Tally_PRD_v1.5.0_Engineering_Control_Plane_Core.md`
4. `docs/tasks/v1.5.0/DECISION_LOCKS.md`
5. assigned GitHub Issue / `docs/tasks/v1.5.0/TASK_DAG.md`
6. `docs/tasks/v1.5.0/L3_IMPLEMENTATION_EVIDENCE.md`

### Current implementation / closure

- `docs/STATUS.md`
- `docs/architecture/CURRENT_CODEBASE.md`
- `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md`
- `docs/releases/v1.5.0_DEVELOPMENT_BASELINE.md`
- `docs/tasks/v1.5.0/VALIDATION_MATRIX.md`
- `docs/tasks/v1.5.0/CRITICAL_JOURNEYS.md`
- `docs/tasks/v1.5.0/HIDDEN_VALIDATION_PROTOCOL.md`
- `docs/tasks/v1.5.0/RELEASE_QUALIFICATION.md`

Historical v1.4/v1.4.1 PRD/tasks/gates remain evidence and are intentionally retained.

## Build and explicit validation

Normal local development commands remain available:

```bash
make migrate
make build
make test
make test-integration
make check
```

The v1.5.0 closure carrier is explicit Local / Build Host execution:

```bash
bash scripts/validate-v1.5.0.sh
```

Held-out validation runs only from an external pack after visible implementation freezes:

```bash
TALLY_V15_HIDDEN_ROOT=/secure/held-out/tally-v1.5.0 \
  bash scripts/run-v1.5-hidden.sh
```

**Never report PASS for a command that did not actually execute successfully.** `PR PASS != Release PASS`; missing required release evidence is `NOT_RUN(...)` or `ENV-BLOCKED(...)`, not inherited PASS.

## Run

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/tally?sslmode=disable'
export TALLY_PASSWORD='change-me'

make migrate
make build

./bin/server
./bin/worker
```

Default endpoints:

- server/UI: `http://localhost:8080`
- v1.5 Engineering Control Plane: `http://localhost:8080/v15/control-plane`
- worker health: `http://localhost:8081/healthz`

The v1.5 execution runtime is opt-in through `TALLY_V15_RUNTIME=fake|codex|dsh`; when unset, lifecycle/state API remains available but dispatch is unavailable rather than silently selecting a fake runtime.

## Product boundary and deferred slices

v1.5.0 establishes the Engineering Control Plane and shared execution contracts. It does **not** claim completion of:

- v1.5.1 SQLite-default / PostgreSQL-optional storage redesign and DB queue removal;
- v1.5.2 SSH/Docker environments;
- v1.5.3 full GitHub Issue/PR/Merge/Tag/Release authority automation;
- v1.5.4 full multi-Agent role workflow;
- v1.5.5 automatic re-plan / governed learning.

AI Runtime may provide project/task planning intelligence, but persistent engineering state and Coding Task completion authority remain in Tally's Engineering Control Plane / Harness.

## Qualification

Current overall v1.5.0 status: **NOT RELEASE QUALIFIED**.

See `docs/tasks/v1.5.0/RELEASE_QUALIFICATION.md` for the 20 qualification assertions and exact blocked gates.
