# Tally v1.5.0 Engineering Control Plane Foundation

Authority for v1.5.0 execution:
1. `docs/product/v1.5/Tally_PRD_v1.5_FROZEN.md`
2. `docs/product/v1.5/Tally_PRD_v1.5.0_Engineering_Control_Plane_Core.md`
3. `docs/tasks/v1.5.0/DECISION_LOCKS.md`
4. assigned GitHub Issue / Task in `TASK_DAG.md`
5. `docs/tasks/v1.5.0/L3_IMPLEMENTATION_EVIDENCE.md`
6. `PATH_OWNERSHIP.md` / `MODEL_ASSIGNMENT.md`
7. current source/tests as implementation facts

The v1.4.1 task program remains historical/brownfield evidence and does not override frozen v1.5.0 scope. Brownfield invariants that remain valid (single `agentruntime`, ChangeSet/worktree safety, verification independence, append-only evidence, no fabricated PASS) continue to apply.

Validation carrier for this version is explicit Local / Build Host execution. GitHub Actions and all CI systems are prohibited by `.dev-standard/PROJECT_OVERRIDES.md`.

## Closure evidence

- `VALIDATION_MATRIX.md` — exact executed/blocked validation ledger.
- `CRITICAL_JOURNEYS.md` — release-level lifecycle and brownfield journeys.
- `HIDDEN_VALIDATION_PROTOCOL.md` — external held-out pack contract and information-separation rules.
- `RELEASE_QUALIFICATION.md` — 20 release assertions and current qualification state.
- `scripts/validate-v1.5.0.sh` — explicit Local / Build Host visible validation carrier.
- `scripts/run-v1.5-hidden.sh` — explicit external held-out validation carrier.

A focused/PR PASS is not a Release PASS. As long as required executable gates remain `ENV-BLOCKED` or `NOT_RUN`, v1.5.0 remains **NOT RELEASE QUALIFIED**.

Issues: #5 authority/L2, #6 engineering domain/DAG, #7 intelligence/execution contracts, #8 integration, #9 validation/closure.
