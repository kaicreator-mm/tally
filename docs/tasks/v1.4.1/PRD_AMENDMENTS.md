# v1.4.1 L2 Amendment Record

> Status: **INCORPORATED INTO `docs/product/Tally_PRD_v1.4.1_Final.md`**.  
> This file is retained as architecture-review evidence. The Final PRD is now the implementation authority; this file explains why selected pre-L2 draft ideas were rejected or changed.

This document overrides conflicting implementation details in
`input/Tally_PRD_v1.4.1_Code_Aligned_Revision.md`.

## A-01 Runtime abstraction

**Old direction:** introduce a new generic `ExecutionBackend`.

**L2 decision:** do not add a parallel core abstraction.

Use:

```text
closedloop.CodegenRunner
        ↓
v14CodegenRunner
        ↓
agentruntime.Runtime
        ↓
Codex / DeepSeek Harness adapter
```

`agentruntime.Runtime` remains the single runtime protocol.

## A-02 Frozen intent evidence

Add a minimal immutable source artifact carrying confirmed intake facts that are required later for deterministic risk derivation.

Do not persist mutable `risk.Result`.

## A-03 Acceptance authority

`verification.FinalVerified` alone is not sufficient for v1.4.1 acceptance when the compiled contract requires higher assurance or contains unmatched risk.

A candidate with `VERIFIED_LIMITED` semantics must not become Accepted.

## A-04 Rule governance

`EngineeringPolicy` YAML/struct is the runtime rule schema and must remain narrow.

Proposal state, evidence, replay and human governance are separate facts/projections.

A proposal becomes runtime-active only when canonical policy content is committed into `.tally/policies/*.yaml` and loaded from the pinned source commit.

## A-05 Historical replay

Replay may evaluate:

- rule match;
- required clauses;
- gates;
- assurance/risk deltas;
- conflicts;
- insufficient evidence.

Replay must not claim a historical code outcome it cannot derive from recorded evidence.

## A-06 Outcome semantics

Keep the existing five Outcome kinds:

- accepted
- reverted
- hotfixed
- regression_found
- human_correction

Rejected/escalated/integration failure remain workflow facts unless a future ADR changes the Outcome domain.

## A-07 Git scope

v1.4.1 release requirement:

- L0 read
- L1 workspace
- L2 commit
- L3 controlled local integration

Remote push/PR/MR is optional/NEXT, not a release gate.

## A-08 DeepSeek Harness maturity

DeepSeek Harness is secondary/canary first.

It cannot become the security authority and cannot receive protected branch or durable Git authority.

## A-09 UI

UI is a projection over existing lifecycle/domain state.

Dynamic forms are bounded presentation objects. They cannot define domain contracts or arbitrary executable UI.

## A-10 AI Runtime

DIRECT / AUTO / ORCHESTRATED are AI Runtime planning policies.

Tally consumes a unified planning result/provenance and does not reimplement the planner.
