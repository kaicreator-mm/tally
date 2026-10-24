# Tally Documentation

Current program: **v1.5.0 Engineering Control Plane Foundation**  
Current qualification: **NOT RELEASE QUALIFIED**

## Authority

1. `product/v1.5/Tally_PRD_v1.5_FROZEN.md`
2. `product/v1.5/Tally_PRD_v1.5.0_Engineering_Control_Plane_Core.md`
3. `tasks/v1.5.0/DECISION_LOCKS.md`
4. assigned GitHub Issue / `tasks/v1.5.0/TASK_DAG.md`
5. `tasks/v1.5.0/L3_IMPLEMENTATION_EVIDENCE.md`
6. current source/tests as implementation facts
7. `architecture/CURRENT_CODEBASE.md`
8. `implementation/CURRENT_IMPLEMENTATION_STATUS.md`

**Historical v1.5.0** used the original ADS 1.2.1 YAML pin at `main@fab321845a0eb630d50eb5791f3de54b6870f5fb` and explicit Local/Build Host-only verification, with CI prohibited **only for that release line**. **Prospective Tally v2 version branch** instead stages the canonical ADS 4.17.0 `repository/version/revision` source at [PR #81](https://github.com/kaicreator-mm/tally/pull/81), together with its v4 adoption level in `.dev-standard/PROJECT_OVERRIDES.md`. That draft is **not yet effective/qualified for project use**: ADS v4.17 independent source RQ is CONDITIONAL, while `standards/PROJECT_ADOPTION.md` §2.1C(2) says READY for manual adoption. [Issue #80](https://github.com/kaicreator-mm/tally/issues/80) owns the disposition. No Product/L2 freeze or real runtime test is asserted. The historical v1.5.0 rules above remain its immutable release evidence.

## Current

- Product: `product/v1.5/Tally_PRD_v1.5_FROZEN.md`
- v1.5.0 executable scope: `product/v1.5/Tally_PRD_v1.5.0_Engineering_Control_Plane_Core.md`
- Source reality: `architecture/CURRENT_CODEBASE.md`
- Implementation status: `implementation/CURRENT_IMPLEMENTATION_STATUS.md`
- Task program: `tasks/v1.5.0/README.md`
- Validation ledger: `tasks/v1.5.0/VALIDATION_MATRIX.md`
- Critical Journeys: `tasks/v1.5.0/CRITICAL_JOURNEYS.md`
- Hidden Validation protocol: `tasks/v1.5.0/HIDDEN_VALIDATION_PROTOCOL.md`
- Release Qualification: `tasks/v1.5.0/RELEASE_QUALIFICATION.md`
- Development baseline note: `releases/v1.5.0_DEVELOPMENT_BASELINE.md`

## Historical but retained

- v1.4.1 PRD/task program/closure evidence: the brownfield base reused by v1.5.0.
- v1.4 PRD/task program/evals: completed baseline evidence.
- v1.3 and older material: historical evidence / archive.

Historical documents do not override current v1.5.0 authority.
