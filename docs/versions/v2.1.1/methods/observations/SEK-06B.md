# SEK-06B — Method Observation (Architecture and Design lane)

**Task:** #323 (lane builder, SEK-06 family)  
**Date:** 2026-11-06  
**Baseline HEAD:** `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`  
**Branch:** `task/tally-v211-sek06b-architecture-design` (worktree `tally-i323-zcode`)  
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING

## Research path (same access conditions as SEK-06A)

- The official CS2023 Body of Knowledge PDF
  (`https://csed.acm.org/wp-content/uploads/2024/04/3.1-Body-of-Knowledge-1.pdf`, source
  `SE-SRC-CS2023`) was obtained as full text via a text-extraction service on 2026-11-06.
- Direct binary download from csed.acm.org failed from this network: TLS connection reset
  through both direct and proxied routes; the Internet Archive snapshot was rate-limited
  (HTTP 429). **No binary SHA-256 of the PDF is recorded** for this extraction.
- Heading and topic text is verbatim from the extraction and is the authoritative anchor
  for heading-anchored locators. The printed page token `175` was adjacent to the
  `Software Engineering (SE)` heading in the text stream, consistent with the registry
  anchors.
- Page anchors are limited to registry-verified locators (SE Overview = PDF page indices
  174-176; SE Core Hours table = PDF page indices 176-177). Printed pages for individual
  KU sub-sections were NOT independently confirmed, so the new KU-level locator
  `se:LOC-CS2023-SE-DESIGN-KU` is heading-anchored and explicitly states that no printed
  page is asserted. No page numbers were invented anywhere in the lane.
- ISO/IEC/IEEE 42010:2022 was consulted only at the level of the official bibliographic
  metadata page (registry entry `SE-SRC-ISO-IEC-IEEE-42010-2022`, ACCESS_RESTRICTED /
  METADATA_ONLY under the ISO AI/ML-use notice). No normative content was extracted,
  retained or prompted; its unavailability is recorded as the coverage gap
  `se:GAP-ARC-ISO42010-ARCH-DESCRIPTION`.
- SWEBOK v4.0a remains on source-identity hold (#325 unmerged); no SWEBOK KA2/KA3 content
  is cited by this lane (`se:GAP-ARC-SWEBOK-DESIGN-CONSENSUS`).

## Locator verification

| Locator | Type | Anchor | State |
|---|---|---|---|
| `se:LOC-CS2023-SE-OVERVIEW` | SECTION (shared, verbatim) | PDF page indices 174-176, SE Overview | LOCATOR_VERIFIED (registry) |
| `se:LOC-CS2023-SE-KU-TABLE` | TABLE (shared, verbatim) | PDF page indices 176-177, Core Hours table | LOCATOR_VERIFIED (registry) |
| `se:LOC-CS2023-BOK-TABLE` | TABLE (shared, verbatim) | PDF page index 0, front-matter KA table | LOCATOR_VERIFIED (registry) |
| `se:LOC-CS2023-SE-DESIGN-KU` | SECTION (new this lane) | heading-anchored 'SE-Design: Software Design' unit, CS Core 1-4 / KA Core 5-9 / Non-core 10-13; printed page not asserted | LOCATOR_VERIFIED (full-text extraction 2026-11-06) |

All evidence `source_locator_ref` values resolve to the above; all `source_ref` values are
`se:SRC-CS2023`. The five shared records (domain, view, three registry locators) were
copied verbatim from the SEK-06 shared snippet set and verified record-equivalent to the
snippet file after writing.

## Lane content summary

- 11 evidence records: 10 DIRECT_SOURCE_PARAPHRASE + 1 declared SYNTHESIS
  (`se:EVD-ARC-0010`, feeding the anti-pattern); all `EXTRACTED_PENDING_REVIEW`,
  `synthetic: false`.
- 6 knowledge claims (all `review_status: PENDING`), including the abstraction-levels
  definition claim that keeps architecture design and detailed design distinct without
  touching the unavailable architecture-description layer.
- 3 concepts, 1 principle, 2 practices, 1 pattern (layered architecture, context
  provided), 1 anti-pattern (inconsistent API surface, DERIVED_FROM the synthesis
  evidence), 1 decision model (design tradeoffs), 4 coverage gaps (three lane gaps plus
  the shared CS2023 rights-clearance gap), 13 typed relationship
  assertions (6 SUPPORTED_BY, 2 APPLIES_TO, 1 QUALIFIED_BY, 1 DEPENDS_ON, 1 IMPLEMENTS,
  1 DERIVED_FROM, 1 RELATED_TO).
- No record is ACCEPTED; no Case/Counterexample records were created (no real-case
  evidence source admitted; recorded as `se:GAP-ARC-DESIGN-REAL-CASES`).

## Validation results (2026-11-06, Windows native, venv Python 3.14.7 / LinkML 1.12.1)

Repository validator self-test:

```
SELF_TESTS=66/66
```

Lane assembly and validation, exact tip `eb32a2e` revalidated 2026-11-06
(`_tmp/tally-sek06-lanes/validation/06B-bld2/lane-validation.log`; script
`_tmp/tally-sek06-lanes/scripts/assemble_and_validate.py`; the pre-fix initial runs
remain archived under `_tmp/tally-sek06-lanes/validation/06B/`):

```
LANE_RECORD_COUNTS={"anti_patterns": 1, "concepts": 3, "coverage_gaps": 4, "decision_models": 1, "domains": 1, "evidence_records": 11, "knowledge_claims": 6, "patterns": 1, "practices": 2, "principles": 1, "relationship_assertions": 13, "source_locators": 4, "taxonomy_nodes": 1, "taxonomy_views": 1}
COUNT_source_locators=4 MIN=3 PASS
COUNT_evidence_records=11 MIN=3 PASS
COUNT_knowledge_claims=6 MIN=4 PASS
COUNT_concepts=3 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=2 MIN=2 PASS
COUNT_patterns=1 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=4 MIN=1 PASS
COUNT_relationship_assertions=13 MIN=10 PASS
DERIVED_SOURCES=1 se:SRC-CS2023
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

One iteration was needed: an unquoted limitation string containing `": "` in
`se:EVD-ARC-0010` parsed as a YAML mapping and failed LinkML string typing (run1). Fixed
by quoting the string; run2 is fully green. Shared files (validation script, model,
registry) were not modified.

## Review burden and state

```ini
REVIEW_BURDEN=NOT_MEASURED
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```

All records remain PENDING_REVIEW / EXTRACTED_PENDING_REVIEW pending human review; the
synthesis evidence and the two interpretive edges (DEPENDS_ON, RELATED_TO) are flagged as
the highest-priority review targets because their inferential content exceeds the
paraphrase base by design.


## Shared CS2023 rights-clearance gap

The CS2023 source registry limits permission to educational materials/programs
and requires specific permission for other uses. This lane is part of Tally's
general software-engineering knowledge graph; it is not presumed to qualify
under that permission. The shared HIGH/OPEN record
`se:GAP-CS2023-RIGHTS-CLEARANCE` is included verbatim across SEK-06A, SEK-06B
and SEK-06C. No rights clearance is claimed. The four-path correction initially
postdated the lane-validation receipt above; the receipt was re-run at the
exact tip `eb32a2e` on 2026-11-06
(`_tmp/tally-sek06-lanes/validation/06B-bld2/lane-validation.log`) and is
fully green, including `COUNT_coverage_gaps=4`.

## Independent review repair (rev1, 2026-11-06)

Exact-SHA review at `7c57d808` (identity, mechanism, semantics) found one
quotation-accuracy defect: the README three-layer item 2 presented "as large
as software architecture" in quotation marks as SE Overview wording, but the
verbatim source phrase is "large design concerns such as software
architecture". Fixed in the README by quoting the verbatim phrase; no other
file changed (`records.yaml` byte-identical to the `eb32a2e` receipt tip, all
six shared records verified record-equal to the shared snippet set). Lane
validation rerun after the fix is fully green
(`_tmp/tally-sek06-lanes/validation/06B-rev1/lane-validation-postfix.log`),
matching the receipt above line-for-line. No shared-record or
schema/catalog/validator change was needed.
