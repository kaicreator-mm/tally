# Contributing to Tally

Tally is developed as a brownfield, task-contract-driven Go project.

## Before changing code

Read:

1. `AGENTS.md`
2. `docs/product/Tally_PRD_v1.4_Final.md`
3. `docs/tasks/v1.4/DECISION_LOCKS.md`
4. the assigned task
5. `docs/tasks/v1.4/PATH_OWNERSHIP.md`

## One task, one bounded change

Do not mix unrelated fixes into a task.

If a required change falls outside Allowed Paths, stop and update task ownership through the coordinator rather than editing it silently.

## Validation

Run the smallest focused test first, then the required gate for the task.

Never claim PASS for commands you did not run.

## Generated code

Do not hand-edit sqlc generated files. Change SQL/schema through the task that owns generation and run `sqlc generate`.

## Database

PostgreSQL is authoritative. Integration tests use isolated schemas via the existing test helpers.

## Git/worktree

Candidate execution must never modify the owner primary worktree.

## Documentation

Current documentation should describe current source truth. Historical documents stay historical; do not rewrite archive files to match newer designs.

## Commit style

Prefer one task per commit:

```text
<TASK-ID>: concise change summary
```

Include validation evidence in the task completion report rather than inflating the commit message.
