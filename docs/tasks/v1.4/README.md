# Tally v1.4 Implementation Task Pack

This pack was generated from:

- `Tally_PRD_v1.4_Final.md`
- source archive `tally.zip` at HEAD `18193d5ff9ff9b6699a4d9b16b6429e2e084094e`
- `L2_Architecture_Evidence_Research_Prompt.md`

It is optimized for low-cost coding agents and parallel execution **after architecture evidence review**.

## Architecture status

`READY_AFTER_FIXES`

Start with A0. Do not skip the Architecture Freeze gate.

## Execution order

```text
A0  Architecture evidence / spikes / freeze
P0  v1.3.1 Trust Closeout
P1  Contract + EngineeringPolicy + Assurance foundation
P2  Closed Execute -> Verify -> Repair loop
P3  Bug Fix + Migration MVP harness and minimal UI
P4  Real product-value validation and closeout
```

## Core rule

Do not rewrite v1.3 Trust Core. v1.4 is primarily an **outer closed engineering loop** built around existing immutable ChangeSet and verification/evidence semantics.

## Files

- `ARCHITECTURE_EVIDENCE.md` — L2 claim/evidence/freeze result.
- `DECISION_LOCKS.md` — architecture decisions implementation agents cannot silently revise.
- `TASK_DAG.md` — dependency graph by wave.
- `TASK_INDEX.md` — all 55 tasks.
- `TASK_MANIFEST.json` — machine-readable scheduler manifest.
- `PARALLEL_EXECUTION_PLAN.md` — wave scheduling.
- `PATH_OWNERSHIP.md` — conflict/hotspot rules.
- `SOURCE_BASELINE.md` — exact source identity and validation limitations.
- `DEFERRED_NEXT.md` — explicitly non-authorized future scope.
- `LOW_COST_AGENT_PROMPT.md` — per-task worker prompt.
- `GATE_REVIEW_PROMPT.md` — gate review prompt.

## L3 usage

L3 Implementation Evidence Research is intentionally **after** this Task DAG. Run L3 only for tasks whose manifest field is:

- `yes`: research before implementation;
- `optional`: use when local code/contract leaves material uncertainty;
- `no`: local contracts/tests should be sufficient.

L3 Reference Packs must not modify frozen Contract/Architecture authority.
