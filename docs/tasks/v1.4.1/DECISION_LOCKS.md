# v1.4.1 Decision Locks

- **Status:** FROZEN (task A0-02, 2026-09-12)
- **Source baseline:** working tree at HEAD `310d961` (clean); L2 archive baseline `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`
- **Authority:** These locks bind every v1.4.1 implementation task. They rank below `docs/product/Tally_PRD_v1.4.1_Final.md` (which declares the L2 amendments incorporated) and approved ADRs, and above every task file, reference pack, and current source idiom.
- **Change control:** A lock may be added, narrowed, or retired only by an approved architecture decision (ADR, or an amendment recorded in `PRD_AMENDMENTS.md`). An implementing agent that cannot satisfy a lock must stop as `BLOCKED`; it must not weaken, route around, or silently reinterpret a lock.
- **Tension rule:** Where raw PRD text and a lock read differently, the lock together with its binding interpretation below is the binding reading (`PRD_AMENDMENTS.md` is incorporated into the Final PRD).

These locks are hard constraints for implementation agents.

| ID | Lock |
|---|---|
| DL-01 | Do not rewrite `internal/closedloop`; preserve its authority over Verify/Repair/Accept/Escalate. |
| DL-02 | Do not introduce a second core runtime protocol; reuse `internal/agentruntime.Runtime`. |
| DL-03 | Execution runtimes do not own Tally Git durability or final acceptance. |
| DL-04 | All authoritative execution artifacts used for one task must resolve from one pinned source commit. |
| DL-05 | `risk.Result` remains derived; do not create a mutable RiskProfile source of truth. |
| DL-06 | Confirmed intake facts required by risk derivation must survive Freeze as immutable evidence. |
| DL-07 | A candidate whose compiled assurance is insufficient must not become Accepted. |
| DL-08 | Do not add governance lifecycle fields to `engineeringpolicy.Policy` YAML/struct. |
| DL-09 | Runtime-active EngineeringPolicy is the exact policy content in the pinned Git snapshot. |
| DL-10 | Rule replay is evidence-bounded and cannot invent counterfactual test/acceptance outcomes. |
| DL-11 | Dynamic forms are bounded presentation schema; no arbitrary HTML/JS and no dynamic domain schema. |
| DL-12 | Keep existing five Outcome event semantics. |
| DL-13 | v1.4.1 Git release scope ends at controlled local integration; remote hosting automation is not required. |
| DL-14 | Codex remains primary; native cross-provider subagent routing is not a release dependency. |
| DL-15 | DeepSeek Harness starts as secondary/canary and runs under Tally-owned isolation. |
| DL-16 | One task has one primary runtime owner. |
| DL-17 | Hidden/held-out verification material must not enter implementation context. |
| DL-18 | Existing Task/Contract/Policy/Outcome domains must be extended, not duplicated. |
| DL-19 | Human-centered navigation is a read/projection concern and must not rewrite the state machine. |
| DL-20 | AI Runtime planning modes are outside Tally implementation scope. |

## Binding interpretations (frozen resolutions)

The two known PRD-text/Decision-Lock tensions are resolved as follows. These resolutions are frozen; downstream tasks inherit them and must not re-litigate them.

- **BI-01 — binds PRD §17 via DL-08 + DL-09 + A-04.** Governance lifecycle metadata — status (`DRAFT`/`PROPOSED`/`ACTIVE`/`SUPERSEDED`/`REJECTED`), `evidence_refs`, `approved_by`, `supersedes`, `scope` — must NOT be added to the `engineeringpolicy.Policy` YAML/struct, and no database flag may declare a policy runtime-active. "Runtime consumes ACTIVE" means: the exact policy blob committed at `.tally/policies/*.yaml` in the pinned Git snapshot (loaded via `artifacts.LoadPolicies`). Governance metadata lives in separate governance facts owned by the P3 lane; activation is a `gitwriter` commit of canonical policy YAML. Evidence: `internal/engineeringpolicy/policy.go` schema is narrow with content-derived `Revision`; `artifacts.LoadPolicies` reads the pinned commit tree.
- **BI-02 — binds PRD §27 via DL-12 + A-06.** The Outcome domain stays exactly the five existing kinds (`accepted`, `reverted`, `hotfixed`, `regression_found`, `human_correction`). PRD §27 lifecycle labels map onto them: `accepted` → `KindAccepted`; `reverted` → `KindReverted`; `human_corrected` → `KindHumanCorrection`; `hotfix_required` → `KindHotfixed` (hotfix event); `regression_detected` → `KindRegressionFound`. `rejected` / `escalated` / `integration_failed` remain workflow/attention/integration facts (runstore, attention projections) and must not be added to OutcomeLog or to a parallel case-history store. Widening the Outcome domain requires a future ADR. Evidence: `internal/outcome/types.go` closed enum; migration `0023_v14_outcome_event.sql` CHECK + append-only trigger; `test/arch/v14_outcome_event_test.go`.

## Open seams — SPIKE_REQUIRED (deliberately not frozen)

Per L2 evidence §7, the following seams are intentionally unfrozen. No task may invent an API for them; each is fixed only by its owning task, with tests. If a task needs one of these seams before its owner has landed, stop as `BLOCKED` instead of designing a parallel one.

| Seam | Owning task | Status |
|---|---|---|
| FrozenIntent exact file path / schema | P0-01 | SPIKE_REQUIRED (open) |
| Acceptance-guard exact placement (Closed Loop Engine vs ServiceVerifier) | P1-06 | FROZEN by owning task P1-06 (2026-09-12, with tests): enforcement in the Closed Loop Engine accept transition (`Config.AcceptGuard` → escalate `assurance_insufficient`); the single semantic evaluator is `taskstate.EvaluateAcceptanceGuard` over `contract.Compiled`; `NewLoop` wires `CompiledAcceptanceGuard` from `LoopConfig.Authority` and the worker always supplies it; ServiceVerifier stays a verification-fact passthrough — ruling and evidence in `reports/P1-06_risk_adjusted_acceptance_guard.md` |
| DeepSeek Harness headless transport shape | P4-03 (spike gate first) | SPIKE_REQUIRED (open) |
| Rule replay storage / query implementation | P3-04 | SPIKE_REQUIRED (open) |
| L3 local Git integration service API | P5-01 | SPIKE_REQUIRED (open) |

## Machine enforcement index

Locks with executable guards at the frozen baseline (`go test ./test/arch/...` passes at HEAD `310d961`). Locks without a machine guard are review-enforced; A0-04 owns extending the guard suite for the v1.4.1-specific readings (e.g. governance-field exclusion on `engineeringpolicy.Policy`, Outcome kind-set stability).

| Guard | Locks it supports |
|---|---|
| `test/arch/v14_runtime_boundary_test.go` (AC2a: agentruntime/airuntime mutual isolation; AC2b: trust-core runtime neutrality) | DL-01, DL-02, DL-18, DL-20 |
| `test/arch/v14_runtime_boundary_test.go` (AC3a/AC3b: EngineeringPolicy import boundary; no ExecutionPolicy vocabulary; parser rejects ExecutionPolicy fields) | DL-08, DL-09 |
| `test/arch/v14_contract_authority_test.go` (AC4a–e: read-only aggregates, content-derived pins, compiler does not rewrite authority) | DL-04, DL-18 |
| `test/arch/v14_assurance_failsafe_test.go` (AC5a–c: unmatched/unknown risk fails safe to L3; VERIFIED_LIMITED surfaced) | DL-05, DL-07 |
| `test/arch/v14_outcome_event_test.go` (AC1–AC5) + migration `0023` CHECK / append-only trigger | DL-12 |
| `internal/gitwriter` expected-base + cross-process lock discipline | DL-03 |
| `internal/closedloop/ports.go` single `Executor` port; one runtime session per attempt in `cmd/worker/v14.go` | DL-01, DL-16 |
| `internal/verification` holdout path-leak tests; `HIDDEN_VALIDATION_HANDOFF.md` process | DL-17 |
