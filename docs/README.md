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

**Historical v1.5.0** source rules used ADS 1.2.1 and explicit Local/Build Host verification, with CI prohibited for that historical line only. **Prospective v2** adopts native TypeScript DomainHarness Packages and uses an exact ADS source pin via a separate version-stage PR; the candidate is **ADS 4.18.0@2c395c55569fe2edf3f4135f12d3d069e526d0b7**, independently RQ READY as a *source standard*, but **not effective in Tally** until project-local compatibility, currentness, verification and approved merge. Source/Host physical tests remain separately required; do not inherit v1.5 no-CI policy for v2.

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
