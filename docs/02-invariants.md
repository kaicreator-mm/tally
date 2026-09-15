# Compatibility Pointer — Invariants

Current v1.5.0 hard locks are in [`tasks/v1.5.0/DECISION_LOCKS.md`](tasks/v1.5.0/DECISION_LOCKS.md), the frozen v1.5 PRDs and executable architecture/domain tests.

Brownfield v1.4/v1.4.1 invariants remain binding where v1.5.0 explicitly retains those authorities, especially:

- one `internal/agentruntime` protocol;
- Agent output is Claim, not Project Fact;
- disposable-worktree execution and immutable ChangeSet identity;
- independent verification / held-out information separation;
- append-only machine facts and provenance;
- no fabricated PASS.

Current implementation mapping is documented in [`architecture/CURRENT_CODEBASE.md`](architecture/CURRENT_CODEBASE.md).
