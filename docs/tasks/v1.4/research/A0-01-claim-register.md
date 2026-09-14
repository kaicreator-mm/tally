# A0-01 — v1.4 Architecture Claim Register (finalized, source-backed)

**Date:** 2026-09-08
**Baseline HEAD:** `18193d5ff9ff9b6699a4d9b16b6429e2e084094e`
**Input authority:** `docs/tasks/v1.4/ARCHITECTURE_EVIDENCE.md` §3 (L2 claim register), re-verified against the live working tree.

This register finalizes the L2 claim matrix as the frozen input to the v1.4 task DAG. Each claim carries its L2 decision plus an A0-01 source-verification note. No production source was changed by this task.

## Source spot-checks performed

| Fact | Source evidence | Result |
|---|---|---|
| Codex does not declare `session.isolation` | `runtimes/codex/runtime.go:25` comment; conformance canary gated off in `runtimes/codex/runtime_test.go:1146-1148` | CONFIRMED |
| Workflow omits configured `ModelProfile` at runtime start | `internal/workflow/v13.go:421` — `StartRequest{Workspace: ...}` only | CONFIRMED |
| Immutable ChangeSet / evidence / measurement migrations exist | `migrations/0019_codegen_changeset.sql`, `0020_verification_evidence.sql`, `0021_approval_measurement.sql` | CONFIRMED |
| Usage-availability aggregate semantics exist in measurement | `internal/measurement/recorder.go`, `recorder_test.go` (`UsageAvailable` flips under missing usage) | CONFIRMED (bug per A22) |
| Historical P1 Trust Gate report exists | `docs/tasks/v1.3/reports/P1_TRUST_GATE.md` | CONFIRMED (historical only) |
| PRD hash matches L2 record | SHA-256 of `docs/product/Tally_PRD_v1.4_Final.md` = `35474f42…b38907` | CONFIRMED |

## Finalized claim register

Decisions are inherited unchanged from L2 (`ARCHITECTURE_EVIDENCE.md` §3); this register records their freeze status for v1.4 implementation.

| Claim | L2 decision | Freeze status |
|---|---|---|
| A01 Go modular monolith + Git monorepo + PostgreSQL | KEEP | FROZEN |
| A02 PostgreSQL lease queue sufficient (no Temporal/Kafka) | KEEP | FROZEN |
| A03 Disposable Git worktree = execution boundary | KEEP | FROZEN |
| A04 Immutable ChangeSet + append-only facts + derived state | KEEP | FROZEN |
| A05 Physically merge TaskContract + VerificationContract | CHANGE (do not merge) | FROZEN — Decision Lock 5 |
| A06 `ContractSnapshot` aggregate facade over both contracts | KEEP | FROZEN — implemented by P1-01 |
| A07 EngineeringPolicy separate from `ExecutionPolicy` | KEEP | FROZEN — Decision Lock 6 |
| A08 `Policy.requires[]` pre-execution + compiled evidence | KEEP WITH CONSTRAINTS | FROZEN — no hidden-verifier leakage (INV-V14-07) |
| A09 L0/L1/L2/L3 default mapping known | SPIKE REQUIRED | OPEN — resolved by A0-03 bake-off |
| A10 Held-out as universal default | REJECT | FROZEN — Decision Lock 11 |
| A11 Codex `session.isolation` today | REJECT / FIX | OPEN — resolved by A0-02 spike, then P0-03 |
| A12 Auto Repair inside inner retry loop | REJECT | FROZEN — Decision Lock 7 |
| A13 Outer closed-loop orchestrator needed | KEEP | FROZEN — implemented by P2-* |
| A14 Outer Repair bounded, default 2 | CHANGE (bound required, count is policy) | FROZEN pending A0-04 ADR wording |
| A15 Structured verification feedback drives Repair | KEEP | FROZEN pending A0-04 ADR schema |
| A16 Raw held-out diagnostics to Repair Agent | REJECT | FROZEN — INV-V14-07/08 |
| A17 `VerifiedBasis` as new mutable lifecycle object | CHANGE (extend `taskstate` read model) | FROZEN — Decision Lock 12, P1-08 |
| A18 Full WorkItem/Requirement rewrite | DEFER | FROZEN — Decision Lock 13 |
| A19 Outcome as one mutable row | CHANGE (append-only events) | FROZEN — Decision Lock 14, P3-05 |
| A20 Outcome capture starts now | KEEP | FROZEN |
| A21 Descriptor-only runtime identity | CHANGE (persist actual snapshot) | FROZEN — Decision Lock 15, P0-01/P0-02 |
| A22 Unknown usage may be dropped | CHANGE (persist call, explicit availability) | FROZEN — Decision Lock 16, P0-05/P0-06 |
| A23 Generic LLM routing in Tally | CHANGE (migrate to AI Runtime boundary) | FROZEN — Decision Lock 17, P0-09..P0-11 |
| A24 Move CodexRuntime into common AI Runtime | REJECT | FROZEN |
| A25 Full Guidance Plane in v1.4 | DEFER | FROZEN — Decision Lock 18 |
| A26 Full Jira/GitLab-style IA | REJECT/DEFER | FROZEN — Decision Lock 19 |
| A27 Current source viable brownfield base | KEEP | FROZEN |
| A28 Repair lineage includes exposure across ancestors | KEEP | FROZEN — P0-07, P2-06 |

## Open items at A0 entry

Only three claims remain open, each owned by an A0 spike/ADR:

1. **A09** — assurance default mapping → A0-03.
2. **A11** — Codex session isolation → A0-02 (then P0-03 implementation).
3. **A14/A15 wording** — repair bound + feedback schema → A0-04 ADR.

All other claims are frozen and must not be reopened by implementation tasks; conflicts return `BLOCKED` per `DECISION_LOCKS.md`.

## Validation honesty note

This task executed `git diff --check` (exit 2, pre-existing trailing whitespace in refreshed docs — not introduced by A0-01) and `git diff --ignore-cr-at-eol --quiet` (exit 1, real doc-refresh content changes present; line-ending-only claim no longer holds for the working tree). No Go validation was run for this task (not required; environment cannot fetch modules).
