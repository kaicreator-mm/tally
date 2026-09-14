# Tally v1.4 — Decision Locks

These locks are implementation authority for the v1.4 task pack. A task may not silently revise them. If a task requires a change, return `BLOCKED` and name the exact lock.

1. **Brownfield, not greenfield.** v1.3 Trust Core remains the base.
2. **Modular monolith.** No new microservices, Kafka, Temporal, generic workflow engine, or distributed queue in v1.4.
3. **PostgreSQL job lease queue remains.** Improve only where a demonstrated bug requires it.
4. **Git monorepo remains.** Disposable worktrees remain the writable Coding Agent boundary.
5. **Contract is a semantic aggregate first.** Do not physically merge `.tally/tasks` and `.tally/verifications` in v1.4 MVP. Use a pinned aggregate facade/snapshot.
6. **ExecutionPolicy != EngineeringPolicy.** Existing runtime resource/network/shell/retry policy remains separate from acceptance/gate policy.
7. **Outer Repair != inner retry.** `ExecutionPolicy.Retry.MaxIterations` controls codegen/runtime iteration. `MaxRepairAttempts` controls verification-driven rework.
8. **Each outer Repair = new Run + new ChangeSet.** Never mutate a verified/failed candidate in place.
9. **Verification remains external authority.** Agent `done` and agent-added tests remain Claims/supplementary evidence unless promoted by policy.
10. **Held-out secrecy is monotonic.** Raw hidden assets/results cannot be passed to implementation while claiming strict information independence. Exposure must be recorded and assurance can only stay or decrease.
11. **Risk-adjusted assurance.** Do not hardcode held-out as universal default. Exact mappings follow the A0 bake-off.
12. **VerifiedBasis is derived.** Extend existing `taskstate` read model; no new mutable lifecycle-state table.
13. **WorkItem rewrite is deferred.** Existing TaskContract identity is enough for NOW. `source_ref`/thin WorkItem belongs to NEXT unless required by an approved gate.
14. **Outcome is append-only event capture.** Do not create mutable outcome booleans as authority.
15. **Actual != desired runtime identity.** Persist observed runtime/model/session/version facts separately from requested profile.
16. **Unknown usage is still a call.** Persist the call even when tokens/cost are unavailable; never synthesize false zero as “reported usage”.
17. **AI Runtime boundary.** Generic inference/provider routing moves to common AI Runtime. Coding Agent runtime/workspace/session/patch semantics remain in Tally.
18. **No independent Guidance authoring plane in v1.4.** Pass through existing project guidance and inject frozen `Policy.requires`/Contract constraints.
19. **Minimal UI.** Launch/closed-loop detail/change detail/attention only. No full Requirement/Release/Knowledge/Quality/Jira IA in v1.4.
20. **No validation bluffing.** Every completion report must use `PASS | FAIL | NOT_RUN(reason)` for each required command.

## A0 Architecture Freeze resolutions (2026-09-08, gate A0-06)

Appended by the A0-06 gate; these convert spike outcomes into frozen decisions. They refine, never contradict, locks 1-20.

21. **Assurance default (resolves lock 11 via A0-03).** Default floor is L1 deterministic visible+protected checks for all acceptance clauses; L3 held-out is the risk-adjusted uplift for any clause above minimal risk. L2 independent-judge remains an experimental tier only — not a default — until a live-judge bake-off reruns (owned by P1-06 with real AI Runtime). Basis: frozen corpus v2 `e83b8bd3…`, L3\L1 = 7/12 unique catches; L2 NOT_RUN(no judge credentials).
22. **Codex session isolation (resolves A11 via A0-02).** Production path is per-run `CODEX_HOME` + handoff of only `auth.json`; never copy/symlink the whole home. `session.isolation` stays undeclared until the conformance canary passes against the real Codex CLI (owned by P0-03/P0-04). Keyring-auth deployments must document file-auth handoff or mark isolation unsupported.
23. **Repair semantics (resolves locks 7/8 via A0-04).** ADR-0001 (`docs/architecture/adr/ADR-0001-v14-closed-loop-repair.md`) is accepted: inner retry and outer Repair are separate budgets; every Repair is a new Run + new disposable worktree + new ChangeSet + rework supersession fact; RepairFeedback is allowlisted (never raw held-out content, assertions, expected values, fixture internals, or oracle code); exposure is append-only, lineage-monotonic, and can only preserve or reduce assurance; `MaxRepairAttempts=2` is a policy default, not an architecture constant.
24. **Codex custom-provider endpoint seeding (refines #22, ADR-0003).** For custom-provider deployments, the per-run home may be seeded with a Tally-generated minimal `config.toml` derived from a typed, explicit `Config.Provider` (id/name/base_url/wire_api/requires_openai_auth/env_key) — desired configuration, not state handoff. No instruction/session-bearing keys are seedable; credential material still crosses homes only via the #22 auth.json handoff or `Config.Env` passthrough; owner-home config/AGENTS/sessions still never flow into run homes; `session.isolation` declaration remains canary-evidence-gated. ADR-0003 (`docs/architecture/adr/ADR-0003-codex-provider-endpoint-seeding.md`).
