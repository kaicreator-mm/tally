# SEK-06A — Method Observation (Requirements Engineering lane)

**Task:** #322 (lane SEK-06A of the SEK-06 vertical-slice lanes)
**Date:** 2026-11-06
**Worktree / branch:** `tally-i322-zcode` / `task/tally-v211-sek06a-requirements`
**Baseline:** `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING

## Research path — CS2023 SE section verification

1. The verified content base for this lane is the CS2023 Body of Knowledge PDF
   (`https://csed.acm.org/wp-content/uploads/2024/04/3.1-Body-of-Knowledge-1.pdf`,
   source `SE-SRC-CS2023`). The official PDF was obtained as **full text via a
   text-extraction service** on 2026-11-06; the SE section text (Preamble,
   Overview items, Core Hours table, and the complete SE-Requirements: Product
   Requirements topic list with learning outcomes) was captured verbatim into
   the lane research extract
   (`_tmp/tally-sek06-lanes/extract/cs2023-se-verified-anchors.md`).
2. **Direct binary download failed** and was not retried into evidence:
   csed.acm.org TLS connections were reset on both the direct route and the
   local proxy route, and the Internet Archive snapshot returned 429
   rate-limiting. Consequence: **no binary SHA-256 of the PDF is recorded**, and
   no record in this lane claims one.
3. Page anchoring therefore follows a strict rule: page-bearing locators use
   only the registry-verified anchors (SE Overview = PDF page indices 174-176;
   Core Hours table = 176-177; the shared locator set). The printed page token
   `175` observed adjacent to the SE heading in the text stream is consistent
   with the registry anchors but was not promoted to a new anchor. The one new
   KU-level locator (`se:LOC-CS2023-SE-REQ-KU`) is **heading-anchored** (exact
   unit heading + KA Core 1-6 / Non-core 7-9 topic numbering, verified against
   the extraction) and explicitly disclaims an unconfirmed printed page inside
   the locator string. No page number was invented anywhere in the lane.
4. Rights handling stayed inside the registry boundary: bibliographic metadata,
   exact locators, concise original paraphrases; short verbatim topic titles
   only where locator anchoring requires them; no substantial report text.

## Scope decisions forced by the source-identity hold

- SWEBOK v4.0a is under source-identity hold (#325 unmerged), so this lane uses
  CS2023 as **curriculum authority only**. Every claim's scope field names the
  CS2023 SE curriculum and its topic number; no record is worded as a
  professional-consensus statement, and the absence of professional-BOK
  evidence is a first-class CoverageGap (`se:GAP-REQ-PROFESSIONAL-BOK-EVIDENCE`).
- CS2023 carries no real requirements-engineering case material and no
  case-class source is registered, so **no Case or Counterexample records were
  created**; the evidence hole is recorded as
  `se:GAP-REQ-REAL-CASE-EVIDENCE` instead of being papered over with invented
  examples.
- The anti-pattern's counterproductive characterization could not be sourced
  from CS2023, so it is derived through a lane-marked SYNTHESIS evidence record
  that combines two verified topic statements, with the inference disclosed at
  the evidence, at the anti-pattern, and on the `DERIVED_FROM` edge.

## Authoring correction process

The lane was drafted against the verbatim anchors first and corrected in
review passes before validation:

1. **Schema-conformance pass.** Two planned fields had no slot in the LinkML
   model and were removed rather than smuggled in: `label` on DecisionModel
   (class has no label slot) and `limitations` on AntiPattern (not a slot of
   that class). The anti-pattern's mandatory "this is lane synthesis, not a
   curriculum statement" disclaimer moved into its scope field, where it is
   still machine-visible.
2. **Scope-qualification pass.** Every claim, concept, principle, practice,
   pattern, anti-pattern and decision model was re-read to ensure the CS2023
   curriculum scope is stated (SWEBOK-hold rule), including the Non-core
   (elective) status of prototyping and product evolution wherever those topics
   carry weight.
3. **Provenance-hygiene pass.** Claim records carry no evidence list; their
   evidence links are canonical `SUPPORTED_BY` edges, matching the SEK-03 slice
   convention. Evidence-bearing edges cite existing evidence records only. The
   five shared records were copied verbatim from the shared snippet pack and
   verified parsed-identical to the pack before validation.
4. **Counterfeit-page pass.** Every locator string was checked against the
   registry anchors / extraction; no page number appears that the registry or
   the extraction does not support.

## Validation — commands and results

Environment: Python 3.14.7, LinkML 1.12.1, linkml-runtime 1.12.1 (SEK venv).
Full logs are saved under `_tmp/tally-sek06-lanes/validation/06A/`
(`self-test.log`, `assemble.log`).

1. Worktree clean at baseline: `git rev-parse HEAD` = `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`.
2. Repository validator self-test:
   `validate_knowledge_graph.py --self-test` → exit 0, final line
   `SELF_TESTS=66/66`.
3. Lane assembly and validation:
   `assemble_and_validate.py <worktree> 06A <records.yaml> <outdir>` → exit 0.
   Key output lines:

```text
LANE_RECORD_COUNTS={"anti_patterns": 1, "concepts": 2, "coverage_gaps": 2, "decision_models": 1, "domains": 1, "evidence_records": 10, "knowledge_claims": 5, "patterns": 1, "practices": 2, "principles": 1, "relationship_assertions": 14, "source_locators": 4, "taxonomy_nodes": 1, "taxonomy_views": 1}
COUNT_source_locators=4 MIN=3 PASS
COUNT_evidence_records=10 MIN=3 PASS
COUNT_knowledge_claims=5 MIN=4 PASS
COUNT_concepts=2 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=2 MIN=2 PASS
COUNT_patterns=1 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=2 MIN=1 PASS
COUNT_relationship_assertions=14 MIN=10 PASS
DERIVED_SOURCES=1 se:SRC-CS2023
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

All record-count minimums PASS; the assembled lane graph passes the repository
graph validator (typed endpoints, evidence/qualifier rules, locator uniqueness,
no ACCEPTED knowledge), the no-ACCEPTED lane gate, and LinkML 1.12.1 instance
validation. RDF/JSON-LD export is not claimed at lane level; it belongs to the
SEK-08 integration experiment.

## Limitations and review burden

- The underlying PDF is verified as extracted text, not as a byte-identical
  binary; no PDF SHA-256 exists for this lane. Heading-anchored locators are
  the compensating control, and a reviewer with PDF access can confirm every
  topic statement against the named unit heading.
- Human comprehension and review time were **NOT_MEASURED**.
- No validation performed here constitutes human review or acceptance; every
  record remains PROPOSED or PENDING_REVIEW pending SEK-08 integration and
  human ReviewDecision.

```ini
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```
