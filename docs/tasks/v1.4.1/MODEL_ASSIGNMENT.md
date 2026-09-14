# v1.4.1 Model Assignment

- Total: **57**
- LOW execution: **40**
- HIGH execution: **17**

## Classification rule

**LOW** = bounded implementation with an existing source seam, deterministic tests, and visible failure. L3 references are intended to make these tasks safe for lower-cost models.

**HIGH** = semantic authority/safety boundary, cross-module invariant, unstable external protocol, Git merge correctness, root-cause diagnosis, or final adversarial release judgment.

A LOW execution task may still require HIGH review. This is deliberate: it keeps implementation cost low while concentrating expensive reasoning on merge/authority checkpoints.

| ID | Phase | Task | Execute | Review | Why |
|---|---|---|---|---|---|
| A0-01 | A0 | Current source-to-PRD authority map | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| A0-02 | A0 | Freeze v1.4.1 architecture decision locks | **HIGH** | HIGH | Architecture authority and cross-package invariants; a wrong lock creates downstream rework. |
| A0-03 | A0 | Validation and dependency preflight | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| A0-04 | A0 | Architecture guard tests | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| A0-05 | A0 | Architecture readiness gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P0-01 | P0 | Define immutable FrozenIntent evidence | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P0-02 | P0 | FrozenIntent Git artifact loader | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P0-03 | P0 | Bounded dynamic clarification schema | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P0-04 | P0 | Render dynamic forms in v1.4 Launch UI | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P0-05 | P0 | Wire production Intent AI executor | **HIGH** | HIGH | Crosses AI Runtime composition and production server wiring; duplicate-provider logic is a silent architecture regression. |
| P0-06 | P0 | Draft-to-frozen domain mapper | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P0-07 | P0 | Verification authoring and freeze admission | **HIGH** | HIGH | Held-out verification and freeze admission are trust-boundary semantics; mistakes can leak or fabricate evidence. |
| P0-08 | P0 | Atomic Confirm-to-Freeze Git transaction | **HIGH** | HIGH | Atomic multi-artifact Git transaction with stale-base and rollback semantics; partial success is dangerous. |
| P0-09 | P0 | Intent-to-Freeze E2E gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P1-01 | P1 | Same-source execution bundle loader | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P1-02 | P1 | Authoritative risk input derivation | **HIGH** | HIGH | Defines the authoritative source-to-risk derivation chain; silent under-classification is worse than loud failure. |
| P1-03 | P1 | Production contract compilation assembler | **HIGH** | HIGH | Creates the production compilation authority and combines policy/risk/assurance inputs from one source snapshot. |
| P1-04 | P1 | Wire worker to compiled authority | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P1-05 | P1 | Compiled prevention context projection | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P1-06 | P1 | Risk-adjusted acceptance guard | **HIGH** | HIGH | Direct acceptance-safety change: FinalVerified must not bypass required assurance or unmatched risk. |
| P1-07 | P1 | Compiled basis in task/read projections | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P1-08 | P1 | Compiled-authority no-bypass gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P2-01 | P2 | Human-centered navigation projection | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P2-02 | P2 | Work list and task detail projection | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P2-03 | P2 | Runs projection | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P2-04 | P2 | Reviews projection with human-needed reason | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P2-05 | P2 | Human-centered UX gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-01 | P3 | Policy governance proposal domain | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-02 | P3 | Append-only policy governance store | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-03 | P3 | Policy candidate canonicalization and validation | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-04 | P3 | Evidence-bounded historical replay | **HIGH** | HIGH | Historical replay must remain evidence-bounded and avoid unsound counterfactual inference. |
| P3-05 | P3 | Policy conflict analysis | **HIGH** | HIGH | Policy conflict semantics are ambiguous and must stay conservative without inventing a new rule engine. |
| P3-06 | P3 | Rules UI projections | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-07 | P3 | Approve and activate policy via GitWriter | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-08 | P3 | Apply Once vs Propose Rule review action | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P3-09 | P3 | Rule-governance gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P4-01 | P4 | Runtime composition registry/factory | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P4-02 | P4 | Codex runtime regression through factory | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P4-03 | P4 | DeepSeek Harness integration spike | **HIGH** | HIGH | External runtime is developer-preview and rapidly changing; integration protocol must be verified against the exact version. |
| P4-04 | P4 | DeepSeek Harness agentruntime adapter skeleton | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P4-05 | P4 | DeepSeek session lifecycle and conformance | **HIGH** | HIGH | Process lifecycle, cancellation and cleanup across Go↔Node/CLI boundaries are failure-prone and security-sensitive. |
| P4-06 | P4 | DeepSeek configuration and canary policy | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P4-07 | P4 | DeepSeek limited-readiness gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P5-01 | P5 | Semantic local Git integration service | **HIGH** | HIGH | Defines Git integration authority and semantic API; wrong design can re-expose raw Git or duplicate GitWriter. |
| P5-02 | P5 | Controlled integration branch creation | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P5-03 | P5 | Safe local merge integration | **HIGH** | HIGH | Merge/conflict/base-drift correctness is repository-safety critical. |
| P5-04 | P5 | Protected-branch and Git permission policy L0-L3 | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P5-05 | P5 | Integration audit/read projection | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P5-06 | P5 | Local Git control-plane gate | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P6-01 | P6 | Accepted Outcome production writer | **HIGH** | HIGH | Must attach append-only Outcome to the exact durable acceptance seam with correct retry/idempotency behavior. |
| P6-02 | P6 | Remaining existing Outcome event capture | **LOW** | HIGH | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P6-03 | P6 | Outcome evidence projection for Rules | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P6-04 | P6 | Migration false-negative reproduction | **HIGH** | HIGH | Root-cause diagnosis task; requires discriminating verifier, fixture, DB lifecycle, environment and oracle failures. |
| P6-05 | P6 | Fix migration verification root cause | **HIGH** | HIGH | Fix depends on diagnosed root cause and must avoid weakening verifier semantics. |
| P6-06 | P6 | Safe project export | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P6-07 | P6 | Current documentation synchronization | **LOW** | LOW | The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred. |
| P6-08 | P6 | Final hidden validation and release gate | **HIGH** | HIGH | Final adversarial release judgment spans all authority boundaries and must detect subtle bypasses. |

## HIGH execution tasks

- **A0-02 — Freeze v1.4.1 architecture decision locks**: Architecture authority and cross-package invariants; a wrong lock creates downstream rework.
- **P0-05 — Wire production Intent AI executor**: Crosses AI Runtime composition and production server wiring; duplicate-provider logic is a silent architecture regression.
- **P0-07 — Verification authoring and freeze admission**: Held-out verification and freeze admission are trust-boundary semantics; mistakes can leak or fabricate evidence.
- **P0-08 — Atomic Confirm-to-Freeze Git transaction**: Atomic multi-artifact Git transaction with stale-base and rollback semantics; partial success is dangerous.
- **P1-02 — Authoritative risk input derivation**: Defines the authoritative source-to-risk derivation chain; silent under-classification is worse than loud failure.
- **P1-03 — Production contract compilation assembler**: Creates the production compilation authority and combines policy/risk/assurance inputs from one source snapshot.
- **P1-06 — Risk-adjusted acceptance guard**: Direct acceptance-safety change: FinalVerified must not bypass required assurance or unmatched risk.
- **P3-04 — Evidence-bounded historical replay**: Historical replay must remain evidence-bounded and avoid unsound counterfactual inference.
- **P3-05 — Policy conflict analysis**: Policy conflict semantics are ambiguous and must stay conservative without inventing a new rule engine.
- **P4-03 — DeepSeek Harness integration spike**: External runtime is developer-preview and rapidly changing; integration protocol must be verified against the exact version.
- **P4-05 — DeepSeek session lifecycle and conformance**: Process lifecycle, cancellation and cleanup across Go↔Node/CLI boundaries are failure-prone and security-sensitive.
- **P5-01 — Semantic local Git integration service**: Defines Git integration authority and semantic API; wrong design can re-expose raw Git or duplicate GitWriter.
- **P5-03 — Safe local merge integration**: Merge/conflict/base-drift correctness is repository-safety critical.
- **P6-01 — Accepted Outcome production writer**: Must attach append-only Outcome to the exact durable acceptance seam with correct retry/idempotency behavior.
- **P6-04 — Migration false-negative reproduction**: Root-cause diagnosis task; requires discriminating verifier, fixture, DB lifecycle, environment and oracle failures.
- **P6-05 — Fix migration verification root cause**: Fix depends on diagnosed root cause and must avoid weakening verifier semantics.
- **P6-08 — Final hidden validation and release gate**: Final adversarial release judgment spans all authority boundaries and must detect subtle bypasses.

## Low-cost production policy

For LOW tasks:
1. feed only the task file + Decision Locks + directly referenced source/tests;
2. do not ask the model to re-plan the architecture;
3. one task = one branch/worktree/ChangeSet;
4. stop on source mismatch rather than improvising;
5. run focused tests before broad tests;
6. tasks marked HIGH review are merged only after a strong-model semantic review.
