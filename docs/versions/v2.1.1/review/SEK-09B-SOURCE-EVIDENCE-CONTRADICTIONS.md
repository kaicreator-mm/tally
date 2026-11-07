# SEK-09B — Independent Source / Evidence / Contradiction Review

**Task:** #353 · **Coordinator:** #351 · **Parent:** #299  
**Baseline:** `wip/v2.1.1-knowledge` @ `902126199c3ec18665873f5b6a6afe59854369ad`  
**TREE:** `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9`  
**Branch:** `task/tally-v211-sek09b-source-contradictions`  
**Sole write path:** this report

## Result

**SEK09B=PASS_WITH_LIMITATIONS** for the sampled source/evidence chains and explicit rights/provenance boundaries.  
**P0=0 / P1=0 / P2=2 / P3=0.**

This is a bounded sample, not a claim that every one of 106 claims has received semantic human review.

## Sampling method

Selected one representative KnowledgeClaim from each of the ten SEK-06/SEK-07 lanes, then followed the graph's `SUPPORTED_BY` edge to Evidence, SourceLocator and Source. Selection was purposive auditor judgment, not random sampling or a deterministic representative-ranking algorithm; the exact 40 IDs below are the frozen, reproducible audit sample. Reproduction means resolving these enumerated chains rather than attempting to regenerate an unspecified representative choice. Each selected claim exists in the exact WIP JSON-LD graph; each selected evidence node is a direct-source paraphrase; each selected locator and source resolves.

| Lane | Claim | Evidence | Locator | Source |
|---|---|---|---|---|
| 06A | `se:CLM-REQ-FUNCTIONAL-DOC-METHODS` | `se:EVD-REQ-FUNCTIONAL-REQ-METHODS` | `se:LOC-CS2023-SE-REQ-KU` | `se:SRC-CS2023` |
| 06B | `se:CLM-ARC-0001` | `se:EVD-ARC-0001` | `se:LOC-CS2023-SE-DESIGN-KU` | `se:SRC-CS2023` |
| 06C | `se:CLM-TVV-0001` | `se:EVD-TVV-0003` | `se:LOC-CS2023-SE-VALIDATION-KU` | `se:SRC-CS2023` |
| 07A | `se:CLM-07A-NO-UNIVERSAL-PROCESS` | `se:EVD-07A-KA10-PROCESS-DEFINITION` | `se:LOC-07A-SWEBOK-KA10-FUNDAMENTALS` | `se:SRC-SWEBOK-GUIDE-4-0A` |
| 07B | `se:CLM-07B-SWEBOK-CONSTRUCTION-DEFINITION` | `se:EVD-07B-SWEBOK-CONSTRUCTION-DEFINITION` | `se:LOC-07B-SWEBOK-KA4-INTRO-FUNDAMENTALS` | `se:SRC-SWEBOK-GUIDE-4-0A` |
| 07C | `se:CLM-07C-SWEBOK-SECURITY-BY-DESIGN` | `se:EVD-07C-SWEBOK-SECURITY-BY-DESIGN` | `se:LOC-07C-SWEBOK-KA13-FUNDAMENTALS` | `se:SRC-SWEBOK-GUIDE-4-0A` |
| 07D | `se:CLM-07D-RELIABILITY-PROBABILITY-DEFINITION` | `se:EVD-07D-CS2023-RELIABILITY-CONCEPT` | `se:LOC-07D-CS2023-SE-RELIABILITY-KU` | `se:SRC-CS2023` |
| 07E | `se:CLM-07E-CI-PRELIMINARY-VALIDATION` | `se:EVD-07E-CS2023-TOOLS-KU` | `se:LOC-07E-CS2023-SE-TOOLS-KU` | `se:SRC-CS2023` |
| 07F | `se:CLM-07F-SWEBOK-MAINT-DEFINITION` | `se:EVD-07F-SWEBOK-MAINT-DEFINITION` | `se:LOC-07F-SWEBOK-KA7-MAINT-DEF` | `se:SRC-SWEBOK-GUIDE-4-0A` |
| 07G | `se:CLM-07G-SWEBOK-SEM-DEFINITION` | `se:EVD-07G-SWEBOK-SEM-INTRO` | `se:LOC-07G-SWEBOK-KA9-INTRO` | `se:SRC-SWEBOK-GUIDE-4-0A` |

The sampled claims preserve their authority scope: the 06A/06B/06C and 07D/07E samples are explicitly curriculum-scoped CS2023 statements; the 07A/07B/07C/07F/07G samples are explicitly SWEBOK Guide claims, not universal prescriptions or regulatory definitions. The selected evidence is linked to exact source locators. CS2023 locators are heading/topic anchored to official PDF page indices; the SourceLocator records explicitly state that printed pages were not independently confirmed (printed-page correspondence: **UNKNOWN**), while the registry records the page-index/printed-page offset ambiguity. SWEBOK locators include PDF page indices and printed page ranges.

## Source and provenance observations

- The graph projects three used sources: SWEBOK Guide v4.0a, CS2023 and NIST SP 800-218 v1.1.
- Coverage report counts 137 Evidence records: 127 `DIRECT_SOURCE_PARAPHRASE` and 10 explicitly labelled `SYNTHESIS`.
- All 137 evidence records have `extraction_status: EXTRACTED_PENDING_REVIEW`; lifecycle state is 136 `PENDING_REVIEW` plus 1 `PROPOSED` (`se:EVD-07B-SYNTHESIS-BIG-BANG`). Every evidence record binds to one locator.
- All 106/106 KnowledgeClaims have at least one `SUPPORTED_BY` edge (157 such edges total); support edges point to Evidence records.
- The 10 sampled claim/evidence chains resolved through locator and source records. No dangling source/evidence chain was found in the sample.
- The 10 sampled claims state applicability and limitations; examples include CS2023's curriculum-only scope, SWEBOK's non-prescriptive BOK scope, and the explicit note that the security-by-design statement is not an empirical effectiveness finding.
- Claim atomicity was assessed for all ten samples: each states one principal proposition, with conditions, lifecycle stages or examples qualifying that proposition rather than introducing an independent unsupported claim. The 07E synthesis remains explicitly labelled as synthesis rather than direct-source evidence.
- The CI alignment synthesis in 07E is explicitly identified as lane-level synthesis, distinguishes two source roles, and records that functional alignment does not establish equal authority or endorsement.

## Rights and authority boundary

1. **CS2023:** the registry states permission for educational materials/programs; other use requires specific permission. The graph's CS2023-derived claims are curriculum-scoped and the project remains pre-package/human-review-pending. This is not permission for unrestricted commercial reuse.
2. **SWEBOK:** no PDF commit/mirroring/redistribution; concise original paraphrases and exact locators only; no broad commercial/public redistribution assumption.
3. **NIST SP 800-218 v1.1:** original paraphrase, citation and locator preferred; no substantial copying.
4. **ISO/IEC/IEEE standards:** metadata-only in this corpus; no normative text, clause extracts or generated summaries. A SWEBOK claim that reports an ISO 14764 objective explicitly says the ISO text itself was not inspected.

The five ISO/IEC/IEEE sources and NIST SP 800-218A remain unused metadata-only registry entries, not silently ingested evidence.

## Findings

### P2-01 — SWEBOK v4.0a release-date discrepancy remains unresolved

The verified official PDF title page states Released August 2026, while the official IEEE landing page separately records a 25 September 2025 update identifying v4.0a. The Source Registry preserves both observations and marks the discrepancy open pending clarification; the authoritative release date is **UNKNOWN**. The exact PDF identity and ISBN are recorded; no DOI is invented. This is transparent and does not invalidate the sampled locators, but the bibliographic date should not be normalized to one value without authority resolution.

### P2-02 — Contradiction coverage is not exhaustive and the graph currently has no CONTRADICTED_BY assertions

The 10-lane sample found no direct contradiction within the selected claim/evidence chains. However, the integrated graph uses ten predicate types and contains no `CONTRADICTED_BY` assertions. Absence of such edges is not proof that the corpus has no contradictions. The SWEBOK date discrepancy is documented in the Source Registry rather than represented as a graph-level disputed Source/Claim record. A broader contradiction pass is still needed before any claim of contradiction completeness.

## Disposition

- **Current experimental corpus:** sampled provenance and rights boundaries are coherent; PASS_WITH_LIMITATIONS.
- **Production/commercial reuse of CS2023-derived material:** BLOCK until specific permission is established for the intended use.
- **Human acceptance:** not assessed here; no ReviewDecision or ACCEPTED record was created.
- **Uncertainty:** this lane reviewed one representative claim per subdomain; it did not exhaustively re-open every source PDF locator or semantically review all 106 claims. Live external revalidation of the SWEBOK PDF, IEEE landing page and other publisher pages was **NOT_RUN**; their current external state is **UNKNOWN** beyond the recorded registry evidence.
