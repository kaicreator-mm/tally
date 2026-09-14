# CLAUDE.md — Tally v1.5.0

Read `AGENTS.md` first; it is the cross-agent authority summary.

Current authority:
1. frozen v1.5 architecture PRD;
2. frozen executable v1.5.0 PRD;
3. `docs/tasks/v1.5.0/DECISION_LOCKS.md`;
4. assigned Issue/Task;
5. L3 evidence and path ownership;
6. current source/tests.

Working mode is brownfield. Retain existing `internal/agentruntime`, `internal/closedloop`, `internal/workflow`, ChangeSet/workspace/Git safety and verification authorities. Do not create a second Runtime, Closed Loop, Project Flow state source, generic workflow DSL, provider-specific project model or speculative framework.

v1.5.0 freezes project lifecycle/TaskDAG, CompletionContract/Authority, IntelligenceProvider and AgentExecutor × ExecutionEnvironment contracts. Local execution is implemented; SQLite/Postgres redesign, SSH/Docker, full Git authority, full Multi-Agent and automatic Re-plan are later slices.

Validation override: **no GitHub Actions and no CI system** for v1.5.0. Execute required checks explicitly Local / Build Host and record exact commands and `PASS/FAIL/NOT_RUN/ENV-BLOCKED`. Never infer PASS from a merge or tool claim.

Hidden Validation remains withheld until visible candidate freeze.
