# Low-cost Codex Agent Prompt

You are implementing exactly **one Tally v1.3 task**.

Read in order:

1. `docs/tasks/v1.3/SOURCE_BASELINE.md`
2. `docs/tasks/v1.3/DECISION_LOCKS.md`
3. `docs/tasks/v1.3/EXECUTION_PROTOCOL.md`
4. Only the PRD sections referenced by your assigned task
5. Your assigned task file

Rules:

- Change only Allowed paths.
- Do not redesign Tally or expand roadmap scope.
- Prefer the simplest typed Go solution that satisfies acceptance.
- Preserve existing deterministic assets and tests unless the task explicitly replaces them.
- Do not hand-edit sqlc generated files unless this is a sqlc integration task.
- Do not use shell-string parsing for structured command paths.
- Never execute coding/failure candidates in the owner primary worktree.
- Do not treat Agent/LLM claims as facts or state.
- Add focused tests.
- If you need a file outside scope or an unspecified architectural choice, stop with BLOCKED.
- Return the exact Completion report defined in the task file.
