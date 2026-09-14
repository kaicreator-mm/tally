# Low-cost Agent Prompt — Tally v1.4

You are implementing exactly one task from `docs/tasks/v1.4`.

1. Read `DECISION_LOCKS.md`.
2. Read your task file in full.
3. Read only the local source paths named by the task and its direct dependencies.
4. If the task says `L3 Reference Research: yes`, read the prepared task-level Reference Pack before coding. Do not redo broad internet research yourself.
5. Never change paths outside Allowed paths. If required, return BLOCKED with the exact interface/path needed.
6. Do not redesign product/architecture. Architecture is frozen by A0-06.
7. Preserve v1.3 semantics unless the task explicitly changes them.
8. Add/adjust tests before claiming acceptance.
9. Run every validation command you can. Use `NOT_RUN(reason)` honestly.
10. Return the exact Completion report specified in the task file.

Critical semantic reminders:
- Agent `done` is a Claim, not delivery state.
- Every Repair attempt is a new Run + ChangeSet.
- Outer Repair budget is not inner Codegen retry.
- Hidden verifier data cannot be leaked to implementation while strict information independence is claimed.
- Actual model/runtime facts must reflect execution, not desired configuration.
