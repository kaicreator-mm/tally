# SEK-08 Coverage Report — integrated knowledge graph

**Task:** #349 (SEK-08) · **Build date (fixed):** `2026-11-07` ·
**Status:** EXPERIMENTAL / HUMAN_REVIEW_PENDING.

This report distinguishes what the integrated graph covers, what it explicitly does not,
what was excluded, what rights restrict it, and what remains unmeasured human work. It
claims neither 100% completeness nor any measured review efficiency.

## Covered areas

### Lanes (ten reviewed bundles, integrated verbatim)

| Lane | Bundle | Records | Claims | Evidence | Locators | Assertions |
|---|---|---|---|---|---|---|
| SEK-06A | Requirements (CS2023 SE-Requirements + SWEBOK KA1) | 59 | 8 | 13 | 7 | 17 |
| SEK-06B | Architecture & Design | 62 | 9 | 14 | 7 | 16 |
| SEK-06C | Testing & Verification | 81 | 10 | 18 | 12 | 23 |
| SEK-07A | Foundations, Processes, Lifecycle & Quality | 88 | 11 | 12 | 11 | 25 |
| SEK-07B | Implementation & Code (construction) | 79 | 9 | 14 | 9 | 26 |
| SEK-07C | Security Engineering (NIST SSDF) | 85 | 9 | 14 | 10 | 30 |
| SEK-07D | Reliability & Performance | 94 | 13 | 16 | 9 | 30 |
| SEK-07E | Build & Delivery | 99 | 16 | 12 | 10 | 39 |
| SEK-07F | Maintenance & Evolution | 90 | 9 | 13 | 12 | 33 |
| SEK-07G | Planning, Risk & Quality (software engineering management) | 89 | 12 | 11 | 10 | 32 |

(Lane counts include the shared records each lane carries; the merged graph deduplicates
them: 783 lane-owned + 6 shared + 3 derived sources = 792 records.)

### Merged-graph totals

- **Identity:** 789 lane records (783 owned + 6 shared, zero collisions, zero cross-lane
  couplings — re-verified mechanically at this build) + 3 derived `Source` records.
- **By collection:** 1 Domain, 1 Subdomain, 11 taxonomy views, 34 taxonomy nodes,
  3 sources, 86 source locators, 137 evidence records, 106 knowledge claims, 38 concepts,
  14 principles, 28 practices, 12 patterns, 10 anti-patterns, 11 decision models,
  0 cases/counterexamples, 0 review decisions, 29 coverage gaps, 271 relationship
  assertions.
- **Evidence provenance:** all 137 evidence records bind to exactly one locator; all 86
  locator `(source_ref, locator)` keys unique; three registered sources used
  (SWEBOK Guide v4.0a, CS2023, NIST SP 800-218 v1.1). Evidence types: 127
  `DIRECT_SOURCE_PARAPHRASE`, 10 `SYNTHESIS` (lane-level synthesis inferences, all
  grounded in cited source evidence and explicitly labelled as synthesis in their
  statements/scope).
- **Claim support:** 106/106 knowledge claims carry ≥1 `SUPPORTED_BY` edge (157 such
  edges total; every object is an Evidence record).
- **Relationships (271 assertions, 10 catalog predicates):** SUPPORTED_BY 157,
  APPLIES_TO 30, IMPLEMENTS 18, DERIVED_FROM 17, DEPENDS_ON 13, QUALIFIED_BY 11,
  RELATED_TO 11, CROSS_CUTS 6, PART_OF 6, BOUNDED_BY 2 — 0 predicate/endpoint/qualifier/
  duplicate-signature violations against `../model/RELATIONSHIP_CATALOG.yaml` (repository
  validator + projection-level re-check).
- **Taxonomy views (all `coverage_state: PARTIAL`):** CS2023 curriculum SE units; SSDF
  security; SWEBOK process/lifecycle, quality models, engineering foundations;
  construction (CS2023 software development fundamentals); security engineering;
  reliability/performance (operations); build/delivery process; maintenance/evolution
  lifecycle; SEM planning. Node kinds: 11 knowledge units, 7 practice groups,
  6 knowledge areas, 6 concerns, 3 quality attributes, 1 subdomain.

## Explicit gaps

- **29 first-class `CoverageGap` records, all OPEN** — kinds: 24 SOURCE_GAP, 1 each
  REVIEW_GAP / TAXONOMY_GAP / ENTITY_GAP / COVERAGE_GAP / LOCATOR_GAP; severity:
  24 MEDIUM, 2 HIGH, 3 LOW. These include unmodeled SWEBOK knowledge areas not yet
  admitted by any lane, NIST SP 800-218A detailed practices (abstract/metadata verified
  only), CS2023 non-core topics, and rights/authority limitations. Every gap is a graph
  record, not a silent omission.
- **Every taxonomy view is PARTIAL** — curriculum classification is not professional
  completeness. SWEBOK Guide coverage is limited to the chapters/sections the lanes
  directly reinspected from the hash-verified official PDF (spanning the requirements,
  design, construction, testing-hold, maintenance, configuration-management, quality,
  engineering-process, foundations and management knowledge areas); each lane's
  CoverageGap records enumerate what was intentionally left uninspected (e.g. KA13's
  uninspected sections, KA12 quality sections, KA18 foundations breadth, KA4 breadth).
  No view claims domain completeness.
- **No cases/counterexamples** — the corpus contains no real engineering cases yet
  (0 Case, 0 Counterexample records); practice/pattern records therefore rest on source
  evidence only.
- **Six registry sources unused** (metadata-only): five ISO/IEC/IEEE standards
  (12207:2026, 15288:2023, 25010:2023, 42010:2022, 29119-2:2021) and NIST SP
  800-218A — deliberately not projected; ISO remains bibliographic-record-only per the
  registry's AI-use restrictions.

## Excluded legacy material

The SEK-03-era legacy pilot fixture `../vertical-slice/KNOWLEDGE_RECORDS.yaml`
(`SE-VERTICAL-SLICE-0.1.0`, 2026-11-04: 17 locators, 12 evidence records, 12 claims,
12 relationship assertions) was inspected for a lossless mapping into model 0.3.0 and
**excluded from the projection; retained untouched as a historical fixture**. The
mapping is demonstrably not lossless:

1. **Model-required identity/lifecycle fields are absent from every legacy record**
   (`record_version`, `lifecycle_status`, `created_on`; additionally `accessed_on` on
   locators, `statement`/`synthetic` shape on evidence, `claim_kind`/`scope_description`
   on claims after documented renames). A mapping would have to synthesize values, which
   is invention, not lossless conversion.
2. **Relationship assertions lack the catalog-required non-empty `context` and
   `rationale`** (legacy shape is `relationship_id`/`subject_id`/`predicate`/`object_id`/
   `status` only); recovery would require authoring new content.
3. **The four overlapping locator identities would create duplicate/ambiguous graph
   identities**: `SE-LOC-CS2023-BOK-TABLE`, `SE-LOC-CS2023-SE-OVERVIEW`,
   `SE-LOC-CS2023-SE-TOOLS` and `SE-LOC-NIST-218-RISK` carry the same verified
   (source, locator, type, verification-state) as the canonical `se:LOC-*` records
   (see the crosswalk in `README.md`), so merging both would violate the model's
   compound uniqueness on `(source_ref, locator)`.
4. **Fields with no model representation**: claim-level `source_refs` (model claims cite
   Evidence via SUPPORTED_BY, never Sources directly) and `taxonomy_refs` /
   slice-level `registry_ref` (SE-TAXONOMY-0.1.0 and SE-AUTHORITY-REGISTRY-0.1.0 are
   not graph records and would dangle).

Mappable-with-rename fields (`locator_id`→`id`, `verified_content`→
`verified_content_summary`, `paraphrase`→`statement`, `claim_type`→`claim_kind`,
`scope`→`scope_description`, `subject_id`/`object_id`→`subject_ref`/`object_ref`) do
not rescue the mapping while (1)–(4) hold. Nothing was silently omitted or coerced: the
fixture stays in the tree byte-identical and is documented here and in `README.md`.

## Rights restrictions (per `SOURCE_REGISTRY.yaml`; carried verbatim into the projection)

- **SWEBOK Guide v4.0a (IEEE CS)** — most restrictive: "Do not commit, mirror, publish
  or redistribute the PDF. Store bibliographic metadata, exact locators and concise
  original paraphrases only. Do not infer broad commercial or public-redistribution
  permission; seek IEEE permission where the proposed use exceeds the stated
  personal/educational conditions."
- **CS2023 (ACM/IEEE-CS/AAAI)** — educational-use posture stays explicit: permission is
  granted to use the guidelines for development of educational materials and programs;
  other use requires specific permission. Handling: "Store bibliographic metadata, exact
  locators and concise original paraphrases only; do not reproduce substantial report
  text or redistribute beyond the stated permission without obtaining specific
  permission."
- **NIST SP 800-218 v1.1** — "Prefer original paraphrase, citation and locator; do not
  copy substantial text."
- **ISO/IEC/IEEE standards** — not used by the corpus; bibliographic record only, no
  normative text, clause extracts or generated summaries; any deeper review requires a
  permitted human-access procedure.
- All SWEBOK paraphrases in the corpus are concise, source-grounded paraphrases; no
  record is ACCEPTED, so no rights-heavy reuse is authorized by this build.

## Unmeasured human-review burden

`HUMAN_REVIEW_BURDEN=NOT_MEASURED`. No human review was performed or timed in SEK-08.
566 knowledge-class records (404 PENDING_REVIEW + 162 PROPOSED), 137 evidence records
(136 PENDING_REVIEW + 1 PROPOSED, all `EXTRACTED_PENDING_REVIEW`) and 106 claims
(review_status PENDING) await human verification of source, locator, interpretation,
applicability and rights. This report makes no claim about review efficiency or the
rate of eventual acceptance; mechanical duplicate/label scans are not a substitute for
semantic review.
