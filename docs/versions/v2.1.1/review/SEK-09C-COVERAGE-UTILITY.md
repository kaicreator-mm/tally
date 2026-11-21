# SEK-09C — Taxonomy Coverage / Reviewability / Practical Utility Evaluation

**Task:** #354 · **Coordinator:** #351 · **Parent:** #299  
**Baseline:** `wip/v2.1.1-knowledge` @ `902126199c3ec18665873f5b6a6afe59854369ad`  
**TREE:** `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9`  
**Branch:** `task/tally-v211-sek09c-coverage-utility`  
**Sole write path:** this report

## Result

**SEK09C=BLOCK** for full G4/G5 qualification; taxonomy-relative coverage and review navigation are documented, but human review and practical utility are not yet measured to the standard required by #299.  
**P0=0 / P1=0 / P2=2 / P3=0.**

## Taxonomy-relative coverage

The corpus contains ten lane bundles across the v2.1.1 taxonomy and eleven distinct TaxonomyViews. All eleven views explicitly have `coverage_state: PARTIAL`; none claims professional-domain or encyclopedic completeness.

| Lane | Records | Claims | Evidence | Locators | Assertions |
|---|---:|---:|---:|---:|---:|
| 06A Requirements | 59 | 8 | 13 | 7 | 17 |
| 06B Architecture & Design | 62 | 9 | 14 | 7 | 16 |
| 06C Testing & Verification | 81 | 10 | 18 | 12 | 23 |
| 07A Foundations / Process / Lifecycle | 88 | 11 | 12 | 11 | 25 |
| 07B Implementation / Construction | 79 | 9 | 14 | 9 | 26 |
| 07C Security Engineering | 85 | 9 | 14 | 10 | 30 |
| 07D Reliability / Performance | 94 | 13 | 16 | 9 | 30 |
| 07E Build / Delivery | 99 | 16 | 12 | 10 | 39 |
| 07F Maintenance / Evolution | 90 | 9 | 13 | 12 | 33 |
| 07G Planning / Risk / Quality | 89 | 12 | 11 | 10 | 32 |

Lane totals include shared records; the merged graph deduplicates to 783 lane-owned + 6 shared + 3 derived Source records = 792 records.

### Cross-cutting relations, balance and source diversity

The ten lane bundles address all ten #299 taxonomy-hypothesis areas at least partially; this establishes lane allocation breadth, not complete coverage within any area. Eleven TaxonomyViews provide multiple source- and topic-specific lenses across those ten lane perspectives.

The graph contains 6 `CROSS_CUTS`, 11 `QUALIFIED_BY` and 11 `RELATED_TO` RelationshipAssertions. These make intersections and qualifications explicit without treating them as containment or inverse-edge inference. Their limited counts are not evidence that all cross-cutting relationships have been discovered; the 29 OPEN gaps and PARTIAL views remain authoritative limitations.

Coverage depth is uneven: 07E has the largest lane totals (99 records, 39 assertions), while 06A has the smallest (59 records, 17 assertions). These relative counts identify where review and expansion may be needed, but do not prove absolute overrepresentation in 07E or sufficiency in any other lane. No lane is represented as complete.

Source diversity is narrow. The three projected sources comprise a curriculum source (CS2023), a professional-BOK guide (SWEBOK) and an advisory framework (NIST SP 800-218); there is no projected empirical-study source. `se:GAP-07E-NO-EMPIRICS` explicitly records the missing empirical evidence class, while `se:GAP-07G-NO-PM-STANDARD-SOURCE` records reliance on secondary characterization where primary standards are unavailable. Source count therefore must not be read as evidence-class or authority diversity.

### Explicit gap inventory

The SEK-08 coverage report records 29 first-class OPEN CoverageGap records:
- 24 SOURCE_GAP
- 1 REVIEW_GAP
- 1 TAXONOMY_GAP
- 1 ENTITY_GAP
- 1 COVERAGE_GAP
- 1 LOCATOR_GAP

Severity: 24 MEDIUM, 2 HIGH, 3 LOW.

The coverage report also identifies:
- 0 Case and 0 Counterexample records.
- Five ISO/IEC/IEEE sources and NIST SP 800-218A are metadata-only and not projected.
- Uninspected SWEBOK knowledge-area sections and non-core CS2023 topics remain explicit gaps.
- The SEK-03 legacy fixture is excluded because mapping is not lossless; it remains untouched and the field-level reasons are documented.
- Four legacy/current locator overlaps have an explicit deterministic crosswalk.
- The historical `boundary_notes[1]` truncation is repaired only in the derived projection; source lane files remain unchanged and the normalization is documented.

These are declared limitations, not hidden omissions. They mean the graph is complete only relative to the current reviewed slice and gap register, not a complete Software Engineering body of knowledge.

## Reviewability observations

The human handbook provides a clear navigation model:
1. Domain / taxonomy view
2. Canonical graph record ID
3. Claim / Practice / Principle / RelationshipAssertion
4. Evidence
5. SourceLocator
6. Source and rights handling

The JSON-LD and Turtle expose record references as IRIs. The handbook explains eleven taxonomy views across ten lane perspectives, directed typed relationships, source/evidence lifecycle and limitations. This is structurally reviewable.

However:
- 566 knowledge-class records remain 404 PENDING_REVIEW + 162 PROPOSED.
- 137 Evidence records remain EXTRACTED_PENDING_REVIEW (136 PENDING_REVIEW + 1 PROPOSED).
- All 106 claims remain review_status PENDING.
- There are 0 ReviewDecision nodes.
- `HUMAN_REVIEW_BURDEN=NOT_MEASURED`; no human review was performed or timed in SEK-08.

Therefore G4 is **REVIEWABILITY=PARTIAL / HUMAN_REVIEW=NOT_MEASURED**, not PASS. Machine validation is not a human review substitute.

## Representative practical-utility probes

These are bounded qualitative probes against the exact corpus graph, not a controlled benchmark or a claim of model superiority. For all three cases, expected-answer criteria were **not pre-registered**; expected content and omissions were assessed qualitatively after inspection, and no omission rate was measured.

| Question | Corpus-grounded answer path | Utility / limitation |
|---|---|---|
| Should one lifecycle process be mandated for every software project? | `se:CLM-07A-NO-UNIVERSAL-PROCESS` → `se:EVD-07A-KA10-PROCESS-DEFINITION` → `se:LOC-07A-SWEBOK-KA10-FUNDAMENTALS` | Provides a source-bound answer: select and adapt processes to project/organizational context and use empirical measurement. The claim is SWEBOK's professional-BOK position, not an empirical outcome study. |
| What does CI validate? | `se:CLM-07E-CI-PRELIMINARY-VALIDATION` → `se:EVD-07E-CS2023-TOOLS-KU` → `se:LOC-07E-CS2023-SE-TOOLS-KU` | Keeps the answer bounded to preliminary validation that current head/trunk builds and passes basic tests. The source does not define “basic tests” or prescribe a pipeline. |
| How should security be considered through the lifecycle? | `se:CLM-07C-SWEBOK-SECURITY-BY-DESIGN` → `se:EVD-07C-SWEBOK-SECURITY-BY-DESIGN` → `se:LOC-07C-SWEBOK-KA13-FUNDAMENTALS` | Provides a traceable lifecycle-spanning security position (requirements, design, construction, testing, maintenance and third-party components). It is a BOK characterization, not empirical proof of effectiveness. |

The corpus-assisted path improves traceability and makes scope/limitations visible compared with an ungrounded direct answer. This is an architectural utility observation only: no blinded answer scoring, independent domain-expert rating, omission/hallucination rate or review-time measurement was conducted. G5 is therefore **PARTIAL / NOT_MEASURED**, not PASS.

## Findings

### P2-01 — No Case or Counterexample records

The graph has zero Case and zero Counterexample records. The task packs correctly forbid inventing synthetic cases as accepted knowledge, and the coverage report records the absence. Nevertheless, #299's quality/exit profile calls for source and case quality and representative positive/negative/boundary examples. The corpus currently has no real engineering case evidence to test that part of the model.

### P2-02 — G4/G5 are not measured

No human ReviewDecision exists, human review effort is unmeasured, and the practical-utility probes above are qualitative only. These are transparent limits but prevent a full G4/G5 PASS.

## Gate disposition

| Gate | Status |
|---|---|
| G0 scope/taxonomy | PARTIAL — explicit boundary, 11 partial views, 29 OPEN gaps |
| G1 sources | PARTIAL — 3 used sources, authority/rights limitations explicit |
| G2 knowledge | PARTIAL — claims/evidence traceability exists; no Case/Counterexample records |
| G3 graph | PASS in specialist lane #352 / PR #355; this remains a separate open integration dependency and its report is not embedded at this HEAD |
| G4 human review | NOT_MEASURED |
| G5 practical utility | PARTIAL / NOT_MEASURED |

**Overall SEK09C=BLOCK for full qualification**, while retaining the corpus as a valid experimental candidate with explicit gaps. Do not change lifecycle statuses or promote records to ACCEPTED to make the gate green.

## Recommended follow-up

- Obtain human domain review for a bounded, representative record sample and record actual reviewer identity/policy/rationale against exact record digests; do not claim all 566 records reviewed.
- Add only source-grounded real cases/counterexamples where permitted sources exist; otherwise keep explicit CoverageGap records.
- Run a controlled practical-utility evaluation with predefined questions, expected-answer criteria, blind or independent scoring, direct-model baseline, corpus-grounded retrieval path, omission/hallucination/traceability scoring and measured human review effort.
- Revisit the 29 gap register after those results. No automatic DomainForge/Domain Package promotion follows from this report.
