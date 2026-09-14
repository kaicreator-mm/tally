# P0/P1 Decision Locks

These choices are intentionally fixed so low-cost agents do not redesign the system independently.

1. **Language/runtime:** Go remains the authoritative control-plane language. Go-first, not Go-only; Runtime adapters may use another language only when an explicit task permits it. P1 Codex adapter in this pack is Go unless the probe proves only a sidecar surface is viable; such a change must be reported BLOCKED for integration review rather than silently added.
2. **Database:** PostgreSQL + sqlc + goose. No ORM. Redis is not a source of truth and is not required for P0/P1.
3. **Repository strategy:** Brownfield Core Replacement. No greenfield rewrite; no whole-repo package relocation for aesthetics.
4. **New P1 bounded packages:** use the task-owned package paths in the task files (`internal/taskcontract`, `internal/verification`, `internal/evidence`, etc.). Do not rename them during leaf work.
5. **Git:** all production Git writes eventually route through `internal/gitwriter`; project content remains Git-authored.
6. **Commands:** structured `CommandSpec`; shell semantics explicit and denied by default.
7. **Workspace:** source execution happens in disposable Git worktrees; dependency caches may be shared outside the source worktree.
8. **ChangeSet:** immutable snapshot. A CodegenRun may produce zero or one candidate ChangeSet. Supersession is an append-only fact.
9. **Desired vs Observed:** TaskContract/VerificationContract are Git/versioned desired artifacts; CodegenRun/ChangeSet/VerificationRun/Evidence/Approval/usage are observed DB records/facts as specified.
10. **Independence requirement:** belongs to Acceptance Clause (`none < source < information`), not individual evidence binding. Functional and independence coverage are separate.
11. **Holdout P1 storage:** minimal Tally-controlled filesystem root outside project/worktree behind an interface. This is an implementation baseline, not a permanent product commitment.
12. **P1 held-out scope:** isolation + final execution + exposure accounting only. Automated recovery ladder/early probes are P2.
13. **Runtime:** FakeRuntime first. CodexRuntime must use a probed real local surface; never invent CLI flags/protocol.
14. **Workflow:** P1 single-Task vertical subset only. No DAG scheduler, workflow DSL, multi-task autopilot or generalized orchestration.
15. **UI:** P1 only Tasks/Changes/Review minimal surfaces. No P2 Inbox/Saved Views or P4 Failure redesign.
16. **Metrics:** P1 records baseline/distributions; it does not set product-value thresholds or claim superiority over bare Codex.
