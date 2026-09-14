# AGENTS.md — Tally v1.5.0 Engineering Instructions

Read this file before changing code.

## 1. Current mission
Tally v1.5.0 is the **Engineering Control Plane Foundation** over the v1.4.1 brownfield baseline.

Product principle:

```text
Strong Agent + Thin Task Harness + Strong Engineering Harness
```

Tally defines engineering correctness and Project TaskDAG state. It does not own Agent thought state, conversation recovery, fine-grained checkpoints or a generic workflow engine.

## 2. Authority order
For v1.5.0 use:
1. `docs/product/v1.5/Tally_PRD_v1.5_FROZEN.md`
2. `docs/product/v1.5/Tally_PRD_v1.5.0_Engineering_Control_Plane_Core.md`
3. `docs/tasks/v1.5.0/DECISION_LOCKS.md`
4. assigned v1.5.0 GitHub Issue / `docs/tasks/v1.5.0/TASK_DAG.md`
5. `docs/tasks/v1.5.0/L3_IMPLEMENTATION_EVIDENCE.md`
6. `docs/tasks/v1.5.0/PATH_OWNERSHIP.md` and `MODEL_ASSIGNMENT.md`
7. current source/tests as implementation facts
8. current architecture/implementation status docs
9. v1.4.1 program as historical/brownfield evidence only

Frozen PRDs are not edited to fit implementation.

## 3. Development standard
Read `.dev-standard/VERSION` and `.dev-standard/PROJECT_OVERRIDES.md`.

For v1.5.0 **do not use GitHub Actions and do not use any CI system**. This changes only the validation carrier. Required tests, Review, Full Regression, Hidden Validation, Critical Journeys and Release Qualification remain required and are executed explicitly Local / Build Host.

Never mark an unexecuted command PASS. Use `PASS`, `FAIL`, `NOT_RUN(reason)` or `ENV-BLOCKED(reason/evidence)`.

## 4. Brownfield locks
Use:

```text
retain → repair → add bounded seam → prove real wiring → remove superseded path only when owned
```

Retain and reuse existing authorities: `internal/agentruntime`, `internal/closedloop`, `internal/workflow`, ChangeSet, workspace/Git safety, contract/policy/risk/assurance/verification/evidence, current Codex/DSH adapters and existing product routes.

Do not recreate a package because v1.5 uses a newer product term.

## 5. Runtime lock
`internal/agentruntime.Runtime/Session` is the **only Agent Runtime protocol**.

Codex and DSH remain out-of-process runtime adapters. A v1.5 `AgentExecutor` is a thin consumer-side task port above this protocol, not a second runtime lifecycle.

Runtime output is an Agent Claim, never Verification or Acceptance.

## 6. Project-flow lock
`Active TaskDAG = Project Flow`.

No second mutable ProjectFlow, completion-percentage authority or generic BPM/workflow DSL may be introduced. Dependency, readiness, cycles, legal transitions, required evidence/artifacts/authority and capability matching are deterministic code.

## 7. Intelligence lock
Direct LLM and AI Runtime implement the same consumer-owned IntelligenceProvider contract. AI Runtime is an enhanced inference provider, not Project Controller, Scheduler, Task state store or DAG owner. Provider output is Proposal/Assessment/Recommendation only.

## 8. Runtime × Environment lock
Runtime and Execution Environment are orthogonal. v1.5.0 implements Local compatibility. SSH/Docker implementations are deferred to v1.5.2.

Core uses environment-aware WorkspaceRef; only the Local compatibility edge converts it to the legacy local workspace path required by `agentruntime.StartRequest`.

## 9. Completion and authority
A node is not complete because a function returned nil or an Agent said done. `CompletionContract` requires the configured state, artifacts, evidence, checks, records, authority and Git requirements.

Participation modes: `AUTO`, `SUPERVISED`, `AUTHORITY_REQUIRED`. Required human authority blocks the transition.

## 10. Scope boundary
Do not implement early:
- v1.5.1 SQLite default/PostgreSQL adapter redesign or DB Job Queue removal;
- v1.5.2 SSH/Docker remote execution;
- v1.5.3 full Git Issue/PR/Merge/Tag/Release authority automation;
- v1.5.4 full Multi-Agent role orchestration;
- v1.5.5 automatic Re-plan/Learning.

Freeze only the shared contracts those lanes need.

## 11. Go/code rules
Prefer typed concrete domain values, small consumer-owned interfaces, explicit context, deterministic transitions, `errors.Is/As`, structured argv/cwd/env, bounded concurrency/retries, append-only records and simple orchestration.

Avoid `map[string]any` core state, string-matched domain errors, ownerless goroutines, recoverable panics, shell-string execution, runtime-name switches in generic orchestration, speculative frameworks and PostgreSQL-specific correctness leakage into new v1.5 Foundation domain.

## 12. Task protocol
For each task: read its authority, inspect current source, stay inside owned paths, implement the smallest complete change, wire production behavior when required, run focused explicit tests, review the diff, record validation evidence, open a PR, review it, then merge safe changes to main. One concern = one PR.

Commit trailers:

```text
Refs: <Requirement/Issue>
```

## 13. Hidden Validation
Held-out cases are not read or used to tune implementation before visible implementation/tests are complete and candidate HEAD is frozen. Hidden Validation runs afterward as an independent release gate.

## 14. Completion report
Report changed paths, production wiring, exact validation commands/results, acceptance criteria, risks and any `ENV-BLOCKED` evidence. `PR PASS != Release PASS`.
