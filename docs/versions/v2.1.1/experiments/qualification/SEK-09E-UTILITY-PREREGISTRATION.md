# SEK-09E — Pre-Registered Corpus Utility Experiment

**Work item:** #361 · **Core gate:** #351 SEK-09 / #299 G5 · **Execution:** ADS #1359 Delivery-First  
**Pinned corpus at registration:** `wip/v2.1.1-knowledge` HEAD `9cbf7983c5b47c146bd934f39e10517c3a828b66`, TREE `fe8f5e106f349bb0c814696fcd231732f7568250`; underlying corpus TREE `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9`.  
**Status: PREREGISTERED_DESIGN / EXPERIMENT=NOT_RUN / HUMAN_OR_INDEPENDENT_SCORING=NOT_RUN / G5=NOT_MEASURED.**

## 1. Hypothesis, leakage controls and scope

Compare a **direct-model answer** (no knowledge corpus) with a **corpus-grounded answer** (explicit retrieved claim/evidence/locator/source path) on seven *new* source-scoped questions. Measure correctness, relevance, source traceability, unsupported content, omission and actual review effort. The three already inspected SEK-09C qualitative probes about universal process choice (07A), CI preliminary validation (07E) and security by design (07C) are **excluded** from benchmark scoring to limit post-hoc leakage.

Neither arm may be described as superior until actual contemporaneous controlled runs, independent blinded evaluation and the calculations below exist. The claims below are `PENDING` experimental knowledge, not universally validated prescriptions. The study measures utility on these questions only, not Software Engineering completeness.

## 2. Frozen question bank and expected criteria — BEFORE answer collection

Each expected criterion is a source-scoped paraphrase of the **pinned** lane's claim; it is not a ground-truth guarantee that the original publication is correct. A reviewer must check evidence/source fit separately. The questions deliberately ask about corpus/source scope, not general authoritative advice.

| Q ID | Fresh question | Target claim → Evidence → Locator → Source | Required answer criteria | Unsupported / boundary traps |
| --- | --- | --- | --- | --- |
| G5-01 | In CS2023's requirements curriculum, which functional-requirement description approaches are named, and is one ranked best? | `CLM-REQ-FUNCTIONAL-DOC-METHODS` → `EVD-REQ-FUNCTIONAL-REQ-METHODS` → `LOC-CS2023-SE-REQ-KU` → `SRC-CS2023` | Mentions use cases and user stories; describes method's design/implementation support plus strengths/weaknesses; notes no superiority ranking | Claims required universal method or empirically proven superiority |
| G5-02 | Does CS2023 distinguish architectural from detailed design, and specify exact criteria for their boundary? | `CLM-ARC-0001` → `EVD-ARC-0001` → `LOC-CS2023-SE-DESIGN-KU` → `SRC-CS2023` | Two levels of design abstraction are distinguished; detailed boundary criteria not supplied by the cited curriculum item | Implies ISO 42010 was directly inspected or gives a universal boundary |
| G5-03 | How does the CS2023 SE-Validation unit distinguish verification from validation? | `CLM-TVV-0001` → `EVD-TVV-0003` → `LOC-CS2023-SE-VALIDATION-KU` → `SRC-CS2023` | Verification asks whether building right; validation asks whether the right thing is built; explicitly curriculum-scoped | Asserts professional-standard authority from an uninspected ISO source |
| G5-04 | Does SWEBOK v4.0a's construction definition include verification, tests and debugging, and establish fixed process boundaries? | `CLM-07B-SWEBOK-CONSTRUCTION-DEFINITION` → `EVD-07B-SWEBOK-CONSTRUCTION-DEFINITION` → `LOC-07B-SWEBOK-KA4-INTRO-FUNDAMENTALS` → `SRC-SWEBOK-GUIDE-4-0A` | Construction includes coding, verification, unit/integration testing and debugging; design also occurs there; boundaries vary with lifecycle | Enforces a single SDLC decomposition |
| G5-05 | Does CS2023's Software Reliability unit specify a measurement procedure for its reliability quantities? | `CLM-07D-RELIABILITY-PROBABILITY-DEFINITION` → `EVD-07D-CS2023-RELIABILITY-CONCEPT` → `LOC-07D-CS2023-SE-RELIABILITY-KU` → `SRC-CS2023` | Mentions probability-of-failure / mean-time-between-failures characterizations; faults cause failures; no measurement procedure in this extracted topic | Invents a reliability formula or calls the curriculum a normative standard |
| G5-06 | Is maintenance in SWEBOK v4.0a limited to post-delivery, and was the cited ISO 14764 text inspected here? | `CLM-07F-SWEBOK-MAINT-DEFINITION` → `EVD-07F-SWEBOK-MAINT-DEFINITION` → `LOC-07F-SWEBOK-KA7-MAINT-DEF` → `SRC-SWEBOK-GUIDE-4-0A` | Activities occur pre-/post-delivery; cost-effective support for operational software; ISO 14764 objective is reported second-hand by SWEBOK | Treats project as having inspected ISO 14764 normative text |
| G5-07 | What activities fall under software engineering management according to SWEBOK v4.0a, and does that establish universal consensus? | `CLM-07G-SWEBOK-SEM-DEFINITION` → `EVD-07G-SWEBOK-SEM-INTRO` → `LOC-07G-SWEBOK-KA9-INTRO` → `SRC-SWEBOK-GUIDE-4-0A` | Planning, estimating, measuring, controlling, coordinating, leading and risk management, stakeholder outcome; explicitly SWEBOK definition | Claims industry-wide consensus/regulation not established by corpus |

All machine IDs in this table use `se:` namespace; abbreviated local IDs above are expanded to `se:<local-id>`. Target linkages derive from the frozen audit sample in #353, not new external authority verification.

## 3. Reproducible trial design and isolation

1. **Freeze first:** preserve this PR's exact registration commit/tree and the corpus digests; do not edit questions/criteria after first model output. Any amendment is a new version with separately reported trials.
2. **Choose and record runtime:** provider, model identifier/revision, temperature, seed when supported, system prompt, user question, tool/retrieval settings, UTC timestamp, trial number, token/time cost if actually observed, request/response raw digests, status (`SUCCESS`, `FAILED`, `NOT_RUN`, `UNKNOWN`). If no immutable model revision available, mark `MODEL_VERSION_UNKNOWN`.
3. **Direct arm (D):** isolated stateless session with the frozen question only; no access to corpus, source IDs, retrieval, evidence chain or expected criteria. No accidental shared context or prior answer. Record any external tool use; avoid it by default.
4. **Corpus arm (K):** same model/settings where feasible and an isolated stateless session; retrieval restricted to pinned corpus files and allowed source/locator metadata. Record exactly which node IDs and evidence excerpts were presented, retrieval ordering and transcript digest. No source PDF mirroring.
5. **Answer freezing:** record both raw answers before evaluator receives either; anonymize and randomize D/K order using reproducible seeded allocation. Judge cannot view arm labels or expected advantage while rating.
6. **Independent evaluator:** separate evaluator from answer generator; review source/evidence/rights fit, answer factual support and omissions against frozen criteria. Actual judge identity and any conflict must be logged. If independent judge is unavailable, record `NOT_RUN`, not a synthetic grade.
7. **Optional repeats:** predeclare repeat count before running; start with one paired response per question per arm (7 × 2 = 14 planned responses). Do not report precision, confidence intervals, significance or superiority from planning alone.

## 4. Frozen scoring rules (per question, per blinded arm)

- **Required-criteria recall:** for every criterion above, binary `MET`/`OMITTED`/`UNVERIFIABLE`. Denominator equals explicitly listed criteria, and `UNVERIFIABLE` is reported separately, not silently scored as correct.
- **Correctness:** 0 = materially contradicted; 1 = partial/uncertain; 2 = fully source-consistent under the declared scope.
- **Applicability/scope:** 0 = overgeneralizes; 1 = partly qualified; 2 = correctly source-qualified.
- **Provenance/traceability:** 0 = none/false; 1 = source mentioned but no resolvable locator; 2 = explicit valid claim→evidence→locator→source chain. For D arm, no hidden corpus IDs are retroactively supplied by evaluator.
- **Unsupported assertions:** count of material claims not supportable by inspected source/allowed corpus; distinguish direct false claims from merely missing evidence. For each, record quoted answer span (short) and independent source check.
- **Omission rate:** `OMITTED / (MET + OMITTED)` for determinable required criteria; never divide by zero and publish `UNDEFINED` if none determinable.
- **Observed effort:** wall clock/evaluator annotation time only when recorded; otherwise `NOT_MEASURED`.
- No precommitted pass threshold is assumed: this is an exploratory paired comparison. Define and document a decision rule *before running* if G5 requires a binary pass; otherwise produce empirical descriptive statistics with uncertainty and **no automatic PASS**.

## 5. Trial capture slots — all blank by design

| Q IDs | Direct arm | Corpus arm | Independent scorer | Observed outputs/scores | Status |
| --- | --- | --- | --- | --- | --- |
| G5-01..07 | NOT_RUN | NOT_RUN | NOT_ASSIGNED | NONE | NOT_MEASURED |

Create actual immutable run records and scoring artifacts in a later authorized evidence write set, including retrieval and human or independent-grader receipts. **No run or evaluation has occurred in producing this preregistration.** G4 human participation is handled separately under #360. Full G5 remains `BLOCK` pending controlled outputs, independent scoring and comparison. No DomainForge/Domain Package promotion, source-content edits, RecordDecision/ACCEPTED transitions, or release/deployment.
