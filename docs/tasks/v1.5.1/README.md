# Tally v1.5.1 Execution Package

Version: **v1.5.1 — Persistence & Execution Simplification**

Authority:
1. `docs/product/v1.5/Tally_PRD_v1.5_FROZEN.md`
2. `docs/product/v1.5/Tally_PRD_v1.5.0_Engineering_Control_Plane_Core.md` for frozen shared contracts
3. `docs/product/v1.5/Tally_PRD_v1.5.1_Persistence_Execution_Simplification.md`
4. `DECISION_LOCKS.md`
5. assigned GitHub Issue / `TASK_DAG.md`
6. `L3_IMPLEMENTATION_EVIDENCE.md`
7. current source/tests as brownfield facts

Artifacts:
- `L2_ARCHITECTURE_EVIDENCE.md`
- `DECISION_LOCKS.md`
- `TASK_DAG.md`
- `L3_IMPLEMENTATION_EVIDENCE.md`
- `PRODUCTION_PROFILE.md` — SQLite-first / PostgreSQL-optional production composition, migration and scheduler boundary
- `VALIDATION_AND_CLOSURE.md` — implementation state, validation ledger, Critical Journeys, Hidden Validation contract and Release Qualification rule

Local / Build Host carriers:
- `scripts/validate-v1.5.1.sh`
- `scripts/run-v1.5.1-hidden.sh`

Issues:
- #18 persistence contracts / architecture evidence
- #19 SQLite default adapter
- #20 PostgreSQL optional adapter / conformance
- #21 local TaskDAG scheduler / default queue removal
- #22 production wiring / validation / release qualification

Current implementation state: T151-01 through T151-06 are code-complete. T151-07/T151-08 remain evidence-gated because executable Local/Build Host, independent review and external held-out validation evidence are not available in the current ChatGPT Web environment.

Validation carrier for v1.5.1 is explicit Local / Build Host. GitHub Actions and all CI systems are prohibited. PR merge is not Release Qualification; no tag/release is allowed while mandatory gates remain `ENV-BLOCKED` or `NOT_RUN`.
