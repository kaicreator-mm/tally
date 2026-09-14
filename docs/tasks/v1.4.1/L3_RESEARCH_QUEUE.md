# L3 Research Status

All 57 tasks now contain an L3 reference pack.

The following tasks are HIGH execution and should not be delegated to a low-cost model:

- **A0-02 — Freeze v1.4.1 architecture decision locks** — Architecture authority and cross-package invariants; a wrong lock creates downstream rework.
- **P0-05 — Wire production Intent AI executor** — Crosses AI Runtime composition and production server wiring; duplicate-provider logic is a silent architecture regression.
- **P0-07 — Verification authoring and freeze admission** — Held-out verification and freeze admission are trust-boundary semantics; mistakes can leak or fabricate evidence.
- **P0-08 — Atomic Confirm-to-Freeze Git transaction** — Atomic multi-artifact Git transaction with stale-base and rollback semantics; partial success is dangerous.
- **P1-02 — Authoritative risk input derivation** — Defines the authoritative source-to-risk derivation chain; silent under-classification is worse than loud failure.
- **P1-03 — Production contract compilation assembler** — Creates the production compilation authority and combines policy/risk/assurance inputs from one source snapshot.
- **P1-06 — Risk-adjusted acceptance guard** — Direct acceptance-safety change: FinalVerified must not bypass required assurance or unmatched risk.
- **P3-04 — Evidence-bounded historical replay** — Historical replay must remain evidence-bounded and avoid unsound counterfactual inference.
- **P3-05 — Policy conflict analysis** — Policy conflict semantics are ambiguous and must stay conservative without inventing a new rule engine.
- **P4-03 — DeepSeek Harness integration spike** — External runtime is developer-preview and rapidly changing; integration protocol must be verified against the exact version.
- **P4-05 — DeepSeek session lifecycle and conformance** — Process lifecycle, cancellation and cleanup across Go↔Node/CLI boundaries are failure-prone and security-sensitive.
- **P5-01 — Semantic local Git integration service** — Defines Git integration authority and semantic API; wrong design can re-expose raw Git or duplicate GitWriter.
- **P5-03 — Safe local merge integration** — Merge/conflict/base-drift correctness is repository-safety critical.
- **P6-01 — Accepted Outcome production writer** — Must attach append-only Outcome to the exact durable acceptance seam with correct retry/idempotency behavior.
- **P6-04 — Migration false-negative reproduction** — Root-cause diagnosis task; requires discriminating verifier, fixture, DB lifecycle, environment and oracle failures.
- **P6-05 — Fix migration verification root cause** — Fix depends on diagnosed root cause and must avoid weakening verifier semantics.
- **P6-08 — Final hidden validation and release gate** — Final adversarial release judgment spans all authority boundaries and must detect subtle bypasses.

LOW tasks should follow the per-task `Low-cost model execution recipe` and use HIGH review where `MODEL_ASSIGNMENT.md` says so.
