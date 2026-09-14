# CLAUDE.md — Tally v1.4.1

Claude Code must read `AGENTS.md` first. `AGENTS.md` is the canonical cross-agent ruleset.

## Current baseline

Read:

1. `docs/product/Tally_PRD_v1.4.1_Final.md`
2. `docs/architecture/CURRENT_CODEBASE.md`
3. `docs/architecture/V1.4.1_TARGET.md`
4. `docs/tasks/v1.4.1/DECISION_LOCKS.md`
5. assigned Task
6. the appended L3 Reference Pack

`Tally_Product_Direction.md` is non-binding.

Runtime reminder:
`agentruntime.Runtime` is the single core Harness protocol. Codex and DeepSeek Harness are out-of-process Runtime Adapters.

Hidden Validation is external and unavailable during implementation. Do not try to obtain it early.


## Working mode

Tally v1.4.1 is brownfield over a completed v1.4 baseline.

Do not start by proposing a rewrite, a new service split, Temporal, Kafka, Kubernetes, a new ORM, a new frontend framework, or a generic multi-agent framework.

Before editing:

- inspect the real implementation;
- identify which existing seam is being extended;
- identify the real production composition root;
- identify the validation path that proves wiring.

For complex tasks, prefer a short implementation plan that names files and invariants, then code.

## Context discipline

Do not read the whole repository by default.

Use:

```text
Task
→ referenced architecture
→ relevant local package/tests
→ L3 Reference Pack
```

When an L3 pack says `DO NOT READ WHOLE REPOSITORY`, obey it.

External references are examples/evidence, never authority.

## High-risk areas

Use extra scrutiny for:

- Confirm→Freeze atomicity;
- Compiled risk/assurance acceptance authority;
- DSH out-of-process lifecycle/cancellation;
- EngineeringPolicy governance split-brain;
- local Git integration conflicts;

- Codex `CODEX_HOME` / session isolation;
- runtime/model identity;
- worktree lifecycle;
- ChangeSet immutability/supersession;
- protected/held-out evidence;
- evidence exposure lineage;
- failure classification;
- nested retry/repair loops;
- cancellation;
- transaction boundaries;
- append-only facts;
- state derivation.

## Claude-specific anti-patterns

Do not:

- create a generic framework because several structs look similar;
- refactor adjacent code “for cleanliness” outside Allowed Paths;
- collapse desired and observed state;
- use prose-only tests for deterministic contracts;
- treat a reviewer LLM as deterministic evidence;
- feed protected verifier internals into the implementation context;
- invent missing architecture rather than returning `BLOCKED`.

## Source-of-truth reminder

Current source can prove that an implementation exists or is wired.

The PRD/Decision Locks decide what it **should mean**.

Historical docs can explain why something exists but cannot override current authority.

## Required finish check

Before returning DONE:

1. review changed paths against Allowed Paths;
2. verify no protected test/contract was weakened;
3. verify production wiring when required;
4. verify any new loop is bounded;
5. verify any new status is derived from facts, not mutable UI state;
6. run task validation;
7. report unavailable tooling as `NOT_RUN(reason)`;
8. keep the repository LF policy (`.gitattributes`) intact; never mix content changes into EOL normalization.
