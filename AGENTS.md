# Tally AGENTS — Version-aware authority routing

**Prospective v2 version-branch routing only.** Determine the applicable standards from the accepted GitHub Issue/Claim, exact version branch and effective immutable ADS pin. Existing v1.5.x Tasks/Claims and old main remain under their original ADS 1.2.1 and v1.5.0 rules below; do not retroactively change their source, no-CI carrier, Review, or evidence.

**For v2 source work**, consult the current, independently accepted exact SHA of Product PRD (docs/product/v2/TALLY_V2_PRD_DOMAIN_PACKAGE_FIRST_R0.md), semantic L2 (docs/architecture/v2/TALLY_V2_DOMAIN_PACKAGE_SEMANTIC_L2_R0.md), parallel Task DAG (docs/implementation/v2/TALLY_V2_ADS_V418_PARALLEL_TASK_DAG_R0.md), and the specific Issue #87–#92/L3. These artifacts may live in separate pending PR branches and **must not be assumed to exist in the current checkout** until merged on the qualified version branch. Use docs/research/v2/GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md and docs/research/v2/OUTPUT_LEDGER.md for source-qualified GitHub delivery. ADS v4.18.0 is the chosen upstream READY source; root .dev-standard/VERSION and one A1_MANUAL_PROTOCOL declaration are still **proposed project adoption until independently qualified #85 and version-only integration**.

**For v2 implementation**, use selected DomainHarness Business/Standard/one Kernel/Capability/Host packages with public ABI. Do not build an independent Tally K, direct private DomainHarness imports or a standalone TS service retrofitted as a Domain Package. GitHub remains Issue/PR/Review/ADS Task Report source authority, SQLite P0 only persists K/Host State/CAS/Journal/Effect through its public Host and a separate GitHub-derived cache; PostgreSQL optional. Real Local Codex CLI, separately isolated Builder GitHub write secrets, verifier OS denied fixture access, and P0 max one executing Task Attempt need actual Node24/OS physical evidence. The DomainHarness v0.10.2 R01 independent Release RQ remains separate.

**For v2 CI**, the prospective source declares ADS v4.18 custom profile: exact pin/current HEAD, trailer, independent source Review and full-root ADS verifier apply to docs/adoption scope; Node24, sealed Host/SQLite/public ABI and OS privilege negatives become required when those implementation paths ship; final Full/Hidden/Critical/Packaging/RQ remain required at release. Independent Tally #83 owner concluded the inherited v1.5 Go source-quality/Postgres failures are actual FAIL but informational for the examined v2 docs-only changes. Do not treat them as PASS/waived or extend this scoped decision to v2 runtime changes.

**Before executing any v2 Task**, require native GitHub Issue Dependencies/current Claim and version-bound L3, real Product/ADS/L2/DH selected Host admission and K source-currentness. Planning Issue or Draft pin alone is NOT_ADMITTED. Keep v2 changes on a version branch until its Release Qualification authorizes main integration. Status must distinguish PASS, FAIL, NOT_RUN, ENV_BLOCKED, UNKNOWN with exact evidence.

---

## Historical Tally v1.5.0 instructions (unchanged applicability)
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


## 15. Conversation output persistence — user direction (2026-10-24)

For **Tally v2 planning, audit, design, review, and implementation work**, read and follow `docs/research/v2/GITHUB_OUTPUT_PERSISTENCE_PROTOCOL.md` and update `docs/research/v2/OUTPUT_LEDGER.md` before reporting a substantive deliverable complete. GitHub is the durable record; conversation-only outputs are not considered delivered. Use version branches/PRs for documents and code, Issue/PR comments for narrow evidence or handoffs, exact SHA for source reviews and explicit NOT_PERSISTED on failed writes. Record source-qualified decisions and preserve the latest user's stated product constraints. This section does not retroactively change v1.5.x frozen product or validation rules.

