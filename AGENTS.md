# AGENTS.md — Tally v1.4.1 Engineering Instructions

Read this file before changing code.

These instructions are canonical for Codex and other coding agents. Claude Code must also read `CLAUDE.md`.

## 1. Current mission

Tally v1.4 is complete. v1.4.1 is a **brownfield product-closure and architecture-hardening increment** over the real v1.4 source.

Expected audited source baseline:

`979a55ad3ba5e514bb80f0e012aa36e0e6577eba`

Do not rewrite the repository.

The v1.4.1 product loop is:

```text
AI Runtime PlanningResult
→ Intent / bounded Dynamic Form
→ Frozen Contract
→ Compile / EngineeringPolicy / Risk / Assurance
→ isolated Coding Harness Runtime
→ immutable ChangeSet
→ Verify / Repair / Escalate
→ Accept
→ controlled local Git integration
→ Outcome
→ Rule Proposal / Replay / EngineeringPolicy evolution
```

## 2. Authority order

When documents, source or comments disagree, use this order:

1. `docs/product/Tally_PRD_v1.4.1_Final.md`
2. approved `docs/architecture/adr/**`
3. `docs/tasks/v1.4.1/DECISION_LOCKS.md`
4. the assigned `docs/tasks/v1.4.1/tasks/<phase>/<TASK>.md`
5. the Task's appended L3 Reference Pack
6. `docs/tasks/v1.4.1/TASK_DAG.md`, `PATH_OWNERSHIP.md`, `MODEL_ASSIGNMENT.md`
7. current source and executable tests as implementation facts
8. `docs/architecture/CURRENT_CODEBASE.md`
9. active `docs/guides/**`
10. L2/L3 architecture/reference evidence
11. v1.4 documents and older task programs — historical evidence only

A Reference Pack never overrides a local Task contract or Decision Lock.

## 3. Current source reality and v1.4.1 locks

The current source already contains the v1.4 Closed Loop, Contract/Policy/Risk/Assurance modules, Git/workspace/ChangeSet authorities and the Codex Runtime Adapter.

Do not recreate an existing package because a product concept has a newer name.

### Runtime lock

`internal/agentruntime.Runtime/Session` is the **only** core Coding Harness protocol.

Current Codex mode:

```text
Tally Go
→ runtimes/codex
→ external Codex CLI process
```

DeepSeek Harness v1.4.1 mode must be:

```text
Tally Go
→ runtimes/dsh
→ external DeepSeek Harness process
```

If a DSH SDK is required, use an out-of-process bridge:

```text
Tally Go
→ runtimes/dsh
→ subprocess bridge
→ DSH SDK
```

Never:
- create a peer `ExecutionBackend` lifecycle protocol;
- embed the Harness agent loop into Tally core;
- grant a Runtime durable Git/integration authority;
- treat Runtime output as Verification or Acceptance.

### Rule lock

EngineeringPolicy remains the runtime rule schema.
Governance proposal/replay/approval is a layer around it.
Runtime-active policy is the exact policy content in the pinned Git source snapshot.

### Model-routing lock

Follow `docs/tasks/v1.4.1/MODEL_ASSIGNMENT.md`:
- LOW execution → low-cost capable model;
- HIGH execution → high-capability model;
- HIGH review → independent strong semantic review before merge.

### Hidden-validation lock

The real Hidden Validation Pack is external and withheld during implementation.
Do not search for, request, mount, inspect or summarize it before visible tasks/gates are complete and candidate HEAD is frozen.


## 4. Brownfield rule

Use:

```text
retain
→ repair
→ add bounded seam
→ prove production wiring
→ remove superseded path only when owned
```

Never do a directory-only rewrite or a speculative architecture cleanup.

## 5. Retained architecture locks

Unless an assigned task explicitly changes a lock:

- Go stays the control-plane language.
- PostgreSQL + pgx + sqlc + goose remain persistence.
- Git stays the versioned intent/content/artifact system.
- no ORM.
- no Kafka.
- no Temporal.
- no Kubernetes requirement.
- no new frontend framework.
- no generic plugin framework.
- no generic workflow DSL.
- no full Jira/Release/Incident platform.
- one v1.4.1 task = one bounded change.

### Trust invariants

- Agent `done` is a Claim, never a Delivery Fact.
- Acceptance is frozen before execution.
- ChangeSet is immutable after freeze.
- repair creates a new CodegenRun and a new ChangeSet.
- supersession is append-only.
- satisfied and waived remain distinct.
- unknown/unmatched risk fails safe; it does not silently choose the weakest assurance.
- cancellation/provider/verifier errors must not be fabricated into candidate failures.
- every ACCEPTABLE / VERIFIED-like result must carry a basis.
- the owner primary worktree is never the candidate execution sandbox.

## 6. AgentRuntime versus AI Runtime

These are different systems.

### Tally AgentRuntime owns

- coding workspace
- Coding Agent process/session
- files/patch
- Agent events
- runtime capabilities
- execution cancellation
- coding-agent-specific runtime identity

### shared AI Runtime owns generic inference

- provider/model selection
- common retry
- generic fallback
- inference cost
- trace
- rollout / baseline / canary

Do not move CodexRuntime/FakeRuntime into generic AI Runtime.

Do not add new provider SDKs or provider-specific routing inside Tally. Existing `internal/llm` is legacy current source until its authorized migration task replaces it.

## 7. Task execution protocol

For each task:

1. read the assigned task file;
2. read only the referenced PRD/Decision Lock sections;
3. inspect the current local source before editing;
4. if the task has an L3 Reference Pack, read it **after** local authority;
5. stay inside Allowed Paths;
6. implement the smallest complete change;
7. wire it into the real production composition root when the task requires production behavior;
8. run focused tests;
9. run the task-required validation;
10. inspect `git diff --check` and changed paths;
11. return the structured completion report.

If architecture is genuinely missing, return `BLOCKED`; do not silently invent a platform-level design.

## 8. L3 Reference Pack rule

A pack under:

```text
the appended L3 pack inside `docs/tasks/v1.4.1/tasks/<phase>/<TASK>.md`
```

is curated implementation evidence.

Read in this order:

```text
local TaskContract
→ Decision Locks
→ current local source
→ reference tests
→ reference contracts
→ reference core implementation
```

Do not read an entire external repository unless the pack explicitly says to.

Do not copy external code without checking license/attribution at implementation time.

## 9. Completion definition

A task is not DONE merely because code compiles or a unit test passes.

When applicable, prove:

```text
IMPLEMENTED
→ WIRED
→ TESTED
→ PRODUCTION_REACHABLE
→ E2E_PROVEN
```

For a pure library/domain task, `PRODUCTION_REACHABLE` can be `N/A(reason)` until the owned integration task, but the task must say so explicitly.

Never mark an unexecuted validation command PASS.

## 10. Shared hotspots

Do not modify unless the assigned task owns them:

- `cmd/worker/main.go`
- `cmd/worker/v13.go`
- `internal/web/handlers/server.go`
- sqlc generated files under `internal/store/`
- migrations
- root guidance files
- `docs/product/Tally_PRD_v1.4.1_Final.md`
- task/decision lock files outside the assigned task
- protected verifier / held-out fixtures not owned by the task

Follow `docs/tasks/v1.4.1/PATH_OWNERSHIP.md`.

## 11. Go rules

Prefer:

- typed concrete domain values;
- small interfaces defined by the consumer;
- explicit `context.Context`;
- deterministic state transitions;
- `errors.Is` / `errors.As`;
- structured argv/cwd/env/timeout;
- bounded retries;
- append-only evidence;
- simple orchestration over framework abstractions.

Avoid:

- `map[string]any` for core state;
- string-matched domain errors;
- ownerless goroutines;
- panic for recoverable errors;
- `sh -c` / implicit shell parsing;
- runtime-name branches in generic workflow code;
- new abstractions with only one speculative consumer.

## 12. Repair-loop rules

Inner execution retry and outer repair are distinct.

```text
inner:
same CodegenRun / Agent execution policy

outer:
immutable candidate
→ independent verification
→ typed repairable failure
→ new CodegenRun
→ new ChangeSet
→ verification
```

Do not:

- reuse `ExecutionPolicy.Retry.MaxIterations` as repair budget;
- mutate a verified candidate in place;
- lower acceptance/assurance during repair;
- feed raw protected held-out details to implementation if that breaks independence;
- create an unbounded repair loop.

## 13. Runtime provenance rules

Desired configuration and observed execution identity are different.

Never write:

```text
requested model == actual model
```

unless the runtime actually reports that observation.

Codex adapter version and Codex CLI version are distinct.

Do not store auth secrets or raw credential material as runtime provenance.

## 14. Testing

Useful sequence:

```bash
go test ./test/arch/...
go test ./internal/<changed-package>/...
go test ./runtimes/<runtime>/...
go test ./test/runtimeconformance/...
go test -tags=integration ./test/e2e/...
```

`make check` requires integration dependencies.

Network/tool unavailable:

```text
NOT_RUN(reason)
```

not PASS.

## 15. Documentation updates

When implementation changes a current fact:

- update `docs/architecture/CURRENT_CODEBASE.md` if architecture/source reality changed;
- update `docs/implementation/CURRENT_IMPLEMENTATION_STATUS.md` for phase status;
- update current guide only when user/operator behavior changed;
- update the assigned v1.4.1 Task status/report;
- do **not** edit archived v1.3 documents to pretend they were always different.

## 16. Never do these to finish a task

- weaken/delete protected tests;
- mark unrun validation PASS;
- widen scope silently;
- modify the PRD to fit the code;
- implement long-term Product Direction early;
- add a framework to avoid a small explicit state machine;
- run generated candidate code in the owner primary worktree;
- turn Agent/runtime claims into Evidence;
- hide gaps behind a boolean VERIFIED.
- rewrite tracked files' line endings wholesale, or mix content changes into a line-ending normalization pass; `.gitattributes` (`* text=auto eol=lf`) is the only line-ending authority.

## 17. Completion report

Return:

```text
TASK: <id>
STATUS: DONE | BLOCKED | FAILED

CHANGED:
- ...

WIRED:
- production entry/path or N/A(reason)

VALIDATION:
- <command>: PASS | FAIL | NOT_RUN(reason)

ACCEPTANCE:
- AC-..: PASS | FAIL

PRODUCTION_REACHABILITY:
- PASS | FAIL | N/A(reason)

RISKS:
- ...

NOTES:
- ...
```
