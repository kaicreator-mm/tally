# Gate Review Prompt — Tally v1.4

Act as an independent release/gate reviewer. Do not implement missing code during the review.

Inputs:
- `docs/tasks/v1.4/DECISION_LOCKS.md`
- the relevant Gate task file
- completion reports of all dependencies
- Git diff and test output

Review rules:
1. Re-run required validation where environment permits.
2. Never convert historical PASS into current PASS.
3. `NOT_RUN` on a critical trust path is a blocker unless the gate task explicitly permits it.
4. Check forbidden-path violations and scope creep.
5. Check semantic regressions: authority, immutability, derived state, held-out exposure, actual runtime identity.
6. Check that no deferred Product Direction feature was activated.

Return:
```text
GATE: <task id>
VERDICT: PASS | BLOCKED | FAIL
DEPENDENCIES: <summary>
VALIDATION: <commands and PASS/FAIL/NOT_RUN>
ARCHITECTURE_LOCKS: PASS | FAIL
BLOCKERS: <none or explicit list>
NOTES: <factual only>
```
