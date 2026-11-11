# SEK-09 — Final qualification disposition (pilot evidence, not corpus approval)

**Core Issue:** #351 · **Experiment parent:** #299 · **ADS:** #1359 Delivery-First

**Final experimental qualification decision: `BLOCKED_NOT_QUALIFIED`** for full corpus acceptance. This is a **completed negative/limited qualification assessment**, not a successful knowledge acceptance, not a G4 waiver, and not a claim that the domain graph is complete. The acceptance result is suitable for a **failed/limited-method retrospective** (SEK-10) after formal coordinator closure; future remediation may occur under separately versioned scope.

**Latest binding before this coordinator update:** `wip/v2.1.1-knowledge` HEAD `fdfae02701efc62020384fc469afe071aa7665b8`. Corpus graph input and G5 run hashes are preserved below. Do not reissue reviews as though they inspected later data.

## 2026-11-08 final evidence reassessment

| Gate | Evidence at latest WIP | Final scoped disposition |
| --- | --- | --- |
| G0 taxonomy/coverage | Ten domain lanes and 11 taxonomy views; **11 PARTIAL**, **29 OPEN CoverageGaps** | **PARTIAL**, completeness not demonstrated |
| G1 authority/rights | Original source/locator links plus 3 new first-party incident Sources; SWEBOK publication-date/version inconsistency, CS2023 reuse/rights limits and temporal source drift remain | **PARTIAL / unresolved** |
| G2 substantive knowledge | **806 records**, **2 real PENDING_REVIEW incident Case**, 106 original PENDING KnowledgeClaims preserved; **0 Counterexample** and **0 ReviewDecision** | **PARTIAL**, illustrative cases not proof of contradiction completeness or accepted truth |
| G3 graph export | #368 / PR #374 independently verified **9,653 equivalent RDF triples**, LinkML 1.12.1, 66 self-tests, 9 negative fixtures, deterministic manifest | **PASS structural/reproducibility only** |
| G4 human review | Fixed 10-lane audit packet #360/PR #362 exists; **no recorded real human verdict, measured review effort or acceptance** | **NOT_CONDUCTED / BLOCK for human-readability-in-practice** |
| G5 practical utility | #367 / PR #375 delivered actual 7×2 controlled **local Qwen2.5-3B** trial, 14/14 real responses, independent *automated* blind grader, frozen raw artifacts and 242 Builder checks | **OBSERVED_WITH_LIMITATIONS**, no automatic PASS, insufficient for universal superiority |

### G5 actual experiment (supersedes the original NOT_RUN snapshot)

Experiment baseline is **intentionally pinned** to the earlier corpus blob `2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00` at preregistered commit `9cbf7983c5b47c146bd934f39e10517c3a828b66`; it is **not** a rerun against the new 806-record graph. PR #375 exact reviewed *Builder-owned* head `2ae1d435b967312ee546f2f34d06d5b753703056`, merged WIP `fdfae02701efc62020384fc469afe071aa7665b8`; no independent source/Exact-SHA reviewer was asserted for that final head, by explicit Owner-directed single-Writer exception.

Frozen paired-comparison blob `0b8fe416925847928e54b00c621a48c85be94965`: seven questions (G5-01..07), fourteen real answers, automated blind grader `INDEPENDENT_MODEL_GRADER` (Kimi; **not** a human), raw SHA256 request/response records. Observed **0–2** rubric means: correctness D **0.8571** versus K **1.8571**; scope D **1.5714** versus K **2.0000**; traceability D **1.0000** versus K **1.0000**. Unsupported-assertion *counts in this sample*: D **9**, K **0**. Average per-question omission rates D **0.4762** vs K **0.1667**. For **G5-04**, knowledge assistance worsened omission rate despite higher correctness score; this is retained as a countervailing observation, not hidden.

Neither review-time cost nor real human utility was measured. One small local model, one pass per question, one automated grader, no predeclared pass threshold, no repeated seeds/multi-model robustness and no random population sampling; therefore **no statistically generalizable superiority, clinical-style significance, or full G5 PASS**. Exact source/retrieval and model response records in `experiments/qualification/runs/SEK-09G/trials/actual-001/`.

### Formal disposition and retained blockers

1. **No accepted Software Engineering corpus:** 106 original Claims remain pending; two incident Cases are review-pending. Do not convert them to ACCEPTED or mint a human ReviewDecision.
2. **G4 was not conducted.** G5 automated grader, code review and graph schema validations cannot substitute for actual human source/record assessment. Evidence is explicitly missing and the project has not demonstrated full G4.
3. **G0–G2 are incomplete by design and evidence.** Preserve 29 gaps, 11 partial views, rights/source issues and absent counterexamples. No promise that an encyclopedic corpus was produced.
4. **Terminal negative/limited experiment outcome:** because #299 explicitly permits an unsuccessful experiment as a legitimate result, record `SEK-09_REVIEW_EXECUTED=YES`, `KNOWLEDGE_ASSET_QUALIFIED=NO`, `TERMINAL_VERDICT=BLOCKED_NOT_QUALIFIED`. This is not a waiver or a PASS; it ends the technical qualification review with an honest fail and enables SEK-10 to document why the pilot did not reach acceptance **once #351 is formally closed**.
5. A future human-review or source-rights/counterexample remediation may be separately scoped; no automatic carry-over acceptance to DomainForge and no main merge, freeze, tag, release or deployment.

**Reviewer boundary:** three #352/#353/#354 specialist reports had independent reviews on their original, earlier graph; #368 had independent exact-head verification of graph integration. #367 real model blind scoring was distinct from answer generation, but PR #375 final Exact-SHA verification was **single Writer**, not independent. This coordinator makes a separate *synthesis* of recorded evidence and does not claim a fresh human review.

---

## Historical initial SEK-09 qualification snapshot (earlier WIP; superseded for current gate status only)

# SEK-09 — Independent Qualification, Contradiction, Coverage & Completeness Gate

**Outcome Task:** #351 (parent #299) · **Evidence lanes:** #352/#353/#354 · **Execution principle:** ADS #1359 Delivery-First

**Coordinator decision: REQUIRES_FURTHER_EVIDENCE; full G4/G5 qualification BLOCKED.**

**Core Task state: OPEN — no completion/acceptance claim.**

## Exact integration baseline and report provenance

- Pre-coordinator integrated WIP branch: `wip/v2.1.1-knowledge`.
- WIP HEAD after report integrations: `9cbf7983c5b47c146bd934f39e10517c3a828b66`; TREE: `fe8f5e106f349bb0c814696fcd231732f7568250`.
- Original corpus/graph baseline: HEAD `902126199c3ec18665873f5b6a6afe59854369ad`; TREE `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9`.
- Reports are newly added, disjoint, report-only files. They do not modify the underlying graph/model/handbook/lane records.
- #355 (SEK-09A): reviewed/final HEAD `0d936d89a9139461059ad5fa0d65599b45e7909d`; WIP merge `76fc192aa36ff6c712f15dd38abc2047d52eef63`; report blob `da85dcb135a3fcb0918b80102681f158656d0227`.
- #356 (SEK-09B): reviewed/final HEAD `ae1f2f298b8f45ad7842b19933928e77132a5812`; WIP merge `c9c9754d0b03bcec5ff04ebdf0c5117b63074caa`; report blob `5abda3a4e5c07cb65ada4e1229970aaaf7c18dfd`.
- #357 (SEK-09C): reviewed/final HEAD `cb24278b20f86e7e244ab742dbe9bc881ed79ac5`; WIP merge `9cbf7983c5b47c146bd934f39e10517c3a828b66`; report blob `1104e221f3241f95ac7903143f5da4a6b04a2679`.
- Underlying graph: Turtle blob `1c0aee238901b1d02ac00ab19e0235a01ba0dfd8`; JSON-LD blob `2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00`; Relationship Catalog blob `a0f088d11878b27b600e604f2cc8d792a85cca9d`.
- Source receipts: #352 comment 6711340680; #353 comment 6711340988; #354 comment 6711341264. Each states fresh post-repair Exact-SHA verification and a sole-report-path write set.
- GitHub merged the three exact-head PRs in sequence into WIP. WIP TREE includes report additions, so same-tree test reuse below is **only** for the unchanged underlying corpus/graph bytes, not for new coordinator or report bytes.

## Aggregated lane decisions

| Lane | Report deliverable | Substantive result | Remaining findings |
| --- | --- | --- | --- |
| SEK-09A #352 | PASS | **Graph/schema/export PASS** in audited scope | P0=0 P1=0 P2=0 P3=0 |
| SEK-09B #353 | PASS | **PASS_WITH_LIMITATIONS** (sampled evidence/provenance) | P0=0 P1=0 P2=2 P3=0 |
| SEK-09C #354 | PASS | **BLOCK** for full G4/G5 | P0=0 P1=0 P2=2 P3=0 |

**Aggregated known substantive findings:** P0=0, P1=0, P2=4, P3=0 across these specialist reports; counts are not a new exhaustive corpus audit. No cross-lane conflict was identified in their reported counts/identities. Deliverable PASS is not corpus acceptance.

## G0–G5 assessment against #299

| Gate | Coordinator disposition | Grounded reason |
| --- | --- | --- |
| G0 scope / taxonomy | **PARTIAL** | Ten topic lanes cover the ten hypothesized areas at slice level; eleven TaxonomyViews are PARTIAL, 29 CoverageGaps OPEN. Completeness is taxonomy-relative, not encyclopedic. |
| G1 source identity / authority / rights | **PARTIAL** | 3 projected sources; source/locator chains exist, but SWEBOK v4.0a release-date authority unresolved, no live publisher revalidation in SEK-09B, CS2023 commercial reuse not cleared. |
| G2 knowledge | **PARTIAL** | 106 claims have 157 SUPPORTED_BY edges; sampled chains pass, but no Case/Counterexample records and contradiction coverage not exhaustive (0 CONTRADICTED_BY edges). 566 knowledge-class records remain pending/proposed. |
| G3 graph / deterministic export | **PASS for checked structural scope** | 793 unique graph nodes, 271 assertions, 1,515 checked references/targets with zero dangling refs, 78/78 qualifiers; Turtle and JSON-LD both produce 9,461 equivalent triples; validator self-tests independently rerun 66/66. Regeneration, full repository validator and LinkML 1.12.1 execution are reused SEK-08 R2 evidence, not newly executed in this coordinator. |
| G4 human review | **NOT_MEASURED / BLOCK for full gate** | Human navigation is structurally possible, but zero ReviewDecision nodes, all 106 claims remain PENDING and human review burden/time not measured. Independent technical report review is not record-level human knowledge acceptance. |
| G5 practical utility | **PARTIAL / NOT_MEASURED / BLOCK for full gate** | Three qualitative traceable question probes; expected criteria not pre-registered, no controlled direct-model benchmark, independent answer scoring, omission/hallucination metric or measured effort. |

## Open substantive findings / required evidence

1. **P2-09B-01:** resolve SWEBOK official PDF vs landing-page date discrepancy with independently checked authoritative source; leave date UNKNOWN until proven.
2. **P2-09B-02:** conduct a broader source/claim contradiction review; zero CONTRADICTED_BY edges does not prove zero contradictions.
3. **P2-09C-01:** add source-grounded genuine Case and Counterexample records only through a separately authorized knowledge write set with provenance and rights handling; do not manufacture a green fixture.
4. **P2-09C-02:** obtain bounded record-level human review and a controlled, pre-registered utility comparison (actual questions, criteria, traceability, direct-model baseline, independent scoring, review effort, omission/error measures). Until performed, G4/G5 stay NOT_MEASURED.
5. Retain the 29 OPEN CoverageGaps, eleven PARTIAL views, rights limits, four-legacy-locator crosswalk and legacy SEK-03 fixture exclusion. Do not silently promote any records or collapse unresolved source authority.

## Disposition / critical path

SEK-09's **review reports and integrated gap register are delivered**, but the outcome gate is **REQUIRES_FURTHER_EVIDENCE** and cannot be marked PASS/closed. The first critical blocker for a full version exit is genuine G4/G5 evaluation; correcting labels or running unchanged-tree PASS suites will not resolve it. Plan any evidence-production work against an explicit legal write set before modifying the corpus. #351 stays open; SEK-10 is dependent and must not be represented as started or closed.

**Assurance provenance:** the coordinator cross-read GitHub PR metadata, exact merged WIP ref/tree and report files. No new local Node/Python/LinkML/Woodpecker run was executed by this coordinator; independent specialist evidence remains attributed to the original reports and is not relabeled as fresh execution. A fresh independent final Exact-SHA verification of the coordinator's two-path output is still required before its acceptance.

**Non-goals:** no corpus/graph/schema/source/handbook edits; no ACCEPTED transitions, ReviewDecision, automated DomainForge promotion, main/release merge, tag or deployment.
