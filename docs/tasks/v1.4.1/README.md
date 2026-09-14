# Tally v1.4.1 — L2 Task Pack

This package is designed to be copied into a Tally working tree as the implementation plan for v1.4.1.

## Status

- Source baseline: `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`
- L2: DONE
- L3: DONE
- Verdict: `SUPPORTED_WITH_CHANGES`
- Tasks: 57
- Implementation: NOT STARTED
- L3 reference packs added to all 57 tasks.
- Model allocation: 40 LOW execution / 17 HIGH execution.

## Read order for an implementation agent

1. `SOURCE_BASELINE.md`
2. `L2_ARCHITECTURE_EVIDENCE.md`
3. `PRD_AMENDMENTS.md`
4. `DECISION_LOCKS.md`
5. `TASK_DAG.md`
6. `PATH_OWNERSHIP.md`
7. `PARALLEL_EXECUTION_PLAN.md`
8. `L3_RESEARCH_QUEUE.md`
9. selected `tasks/<phase>/...md`

## Authority

If the original PRD conflicts with this pack:

1. source invariants;
2. `DECISION_LOCKS.md`;
3. `PRD_AMENDMENTS.md`;
4. this Task Pack;
5. input PRD

in that order.

## Suggested execution policy

- Run A0 first.
- Do not start broad implementation until A0-05 is green.
- P0 and P1 are the critical authority chain.
- Use low-cost models for bounded tasks.
- Use strong models only at marked high-risk seams/gates.
- Run L3 research before every task marked `L3=YES`.
- Merge wave-by-wave, not all branches at the end.

## Key L2 changes from PRD

- reuse `agentruntime.Runtime`, do not add another core ExecutionBackend;
- add immutable FrozenIntent evidence for authoritative risk inputs;
- add risk-adjusted acceptance guard;
- keep governance outside EngineeringPolicy runtime schema;
- preserve five Outcome kinds;
- DeepSeek Harness is canary/secondary first;
- v1.4.1 Git gate ends at local controlled integration;
- AI Runtime planning modes stay outside Tally implementation.

## Local validation note

The analysis environment could not fetch uncached Go dependencies; see `SOURCE_BASELINE.md`.
Do not interpret that as a source test failure.

## L3 additions

- `L3_IMPLEMENTATION_EVIDENCE.md`
- `L3_REFERENCE_CATALOG.md`
- `MODEL_ASSIGNMENT.md`
- L3 reference pack appended to every individual task file.

## Final runtime integration lock

- `internal/agentruntime.Runtime/Session` is the only core Coding Harness protocol.
- Codex remains an out-of-process Runtime Adapter.
- DeepSeek Harness must also run out-of-process behind `runtimes/dsh` (direct process or subprocess bridge).
- Do not introduce a peer `ExecutionBackend` lifecycle protocol.
