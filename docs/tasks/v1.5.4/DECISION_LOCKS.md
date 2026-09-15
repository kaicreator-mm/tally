# v1.5.4 Decision Locks

These decisions are frozen for the v1.5.4 implementation slice.

1. **Role, not persona.** Implementer/Reviewer/Repairer/Verifier/Specialist are engineering responsibilities attached to Task/Gate execution, never persisted personalities or conversation participants.
2. **One Project Flow.** Active TaskDAG remains authoritative for task readiness/lifecycle. Multi-agent workflow state is subordinate collaboration state only.
3. **One execution contract.** Every role executes through the frozen v1.5.0 `AgentExecutor` / `TaskExecutionRequest` / `TaskExecutionResult` boundary and existing `internal/agentruntime`.
4. **Claim is not fact.** Agent completion cannot satisfy review/verification. Reviewer/verifier PASS requires Tally evidence.
5. **Deterministic role order.** Implement → optional Specialist(s) → Review → Repair when required → Review → Verify → CompletionContract acceptance. Repair is bounded.
6. **Independence is typed.** Default reviewer/verifier independence is distinct run/session. Runtime/provider independence is stricter and fails closed unless the actual executor identity differs.
7. **Runtime identity is truthful.** `RuntimeExecutor` rejects a `RuntimeRef.ID` that differs from the actual Runtime descriptor before starting a session.
8. **Restart is semantic.** Persist TaskDAG, policy, role state, findings, evidence and references. Never persist or reconstruct chain-of-thought. An in-flight role is recovered as interrupted and restarted fresh.
9. **Parallelism is bounded.** Only Ready + ParallelEligible tasks may enter a batch. Dependency state, conflict keys/workspace collision and global concurrency limit all apply.
10. **No Job Queue.** The batch scheduler is in-process bounded orchestration over the persisted TaskDAG; no distributed queue/worker fleet is introduced.
11. **CompletionContract stays final.** Multi-agent verification yields acceptance facts; only the existing CompletionContract/Control Plane can mark the task Done.
12. **No CI.** GitHub Actions and all CI systems are excluded for this version. Validation runs only on Local/Build Host.
13. **Release dependency.** v1.5.3 is the frozen release baseline dependency. Its absence blocks Release Qualification/tag, not v1.5.4 core implementation or review.
