# SEK-06B Lane — Software Architecture and Design (CS2023 SE-Design depth slice)

**Lane:** `SEK-06B-ARCHITECTURE-DESIGN-LANE-0.1.0`  
**Status:** DRAFT_PENDING_HUMAN_REVIEW  
**Task:** #323 (vertical-slice lanes) / parent SEK effort #321  
**Base HEAD:** `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`  
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 single-writer integration)

## Purpose and scope

This lane is the Software Architecture and Design depth slice of the SEK-06 vertical-slice
family. It models the CS2023 Software Engineering knowledge unit `SE-Design: Software
Design` (CS Core topics 1-4, KA Core topics 5-9, Non-core topics 10-13) as curriculum-scoped
knowledge: abstraction levels, system design principles, design paradigms, standard
architectures, component boundaries and dependencies, API design principles, requirement
traceability, secure-design principles and design-tradeoff evaluation.

`records.yaml` is a lane file, not a graph root. SEK-08 integration assembles lanes plus
registry-derived Source entities into the full knowledge graph; the shared records
(`se:DOMAIN-SOFTWARE-ENGINEERING`, `se:VIEW-CS2023-SE`, and the three registry-anchored
locators) are copied verbatim from the shared snippet set so integration can deduplicate
them by canonical ID. The sixth shared record, the CS2023 rights-clearance gap
`se:GAP-CS2023-RIGHTS-CLEARANCE`, was added by the coordinated three-lane correction and
is byte-identical across SEK-06A/06B/06C for the same reason.

## The three-layer distinction (scope boundary of this lane)

This lane deliberately keeps three notions apart:

1. **Architecture description** — the ISO/IEC/IEEE 42010:2022 concept remains **not
   covered**: the standard is admitted only as bibliographic metadata
   (`ACCESS_RESTRICTED` / `METADATA_ONLY` under the ISO AI/ML-use notice); no normative
   content may be cited. Its unavailability is recorded as
   `se:GAP-ARC-ISO42010-ARCH-DESCRIPTION`. Nothing in this lane may be read as a claim
   about architecture description as a standardized artifact. Separately, the 2026-11-06
   P2 repair admitted SWEBOK KA2's own description/views/viewpoints vocabulary
   (`se:CLM-ARC-SWEBOK-DESCRIPTION-DISTINCTION`) from direct reinspection — that is the
   Guide's generally accepted content, not the ISO standard's normative concept.
2. **Architecture design** — covered as a curriculum abstraction level: CS2023
   SE-Design CS Core 1a lists "levels of abstraction (e.g., architectural design and
   detailed design)", and the SE Overview frames Software Design as encompassing "large
   design concerns such as software architecture". SWEBOK KA2/KA3 now also cover it at
   professional-BOK fidelity from the reinspected sections (architectural design as
   decision-making within the architecture process, distinct from high-level design).
3. **Detailed software design** — covered as the other example abstraction level of the
   same curriculum topic, with the Overview's "small-scale design choices like API design"
   framing the detailed end of the span; SWEBOK KA3 §2.2 adds the professional-BOK
   characterization of detailed design from direct reinspection.

The curriculum anchor supports "architectural design and detailed design are distinct
abstraction levels" (`se:CLM-ARC-0001`, CS Core 1a); the reinspected SWEBOK KA3 sections
support the professional-BOK stage split
(`se:CLM-ARC-SWEBOK-DESIGN-STAGE-SPLIT`). The architecture-description layer of the
distinction still has no ISO-normative source in this corpus (metadata-only gap).

## Authority fit and limitations

- **CS2023 (`se:SRC-CS2023`)** is the original content source: CURRICULUM_FRAMEWORK,
  primary for the curriculum knowledge model only. Curriculum statements are teaching
  design facts, not professional consensus and not empirical effectiveness evidence.
- **SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`)** — identity/registry reconciliation
  is complete (SEK-02R #325 merged via PR #326 into this lane's WIP base) and the
  2026-11-06 P2 repair reinspected KA2 sections 2 (2.1, 2.2), 2.4, 3 (3.1, 3.1.1, 3.2)
  and KA3 sections 1 (1.1, 1.3, 1.4), 2 (2.1, 2.2) directly from the official PDF
  (SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 411
  pages), admitting three locators, three evidence records and three claims. All
  SWEBOK-derived records stay PENDING_REVIEW; the Guide's
  generally-accepted-knowledge-not-prescription limitation carries into every claim, and
  the remaining coverage-breadth limitation is recorded as
  `se:GAP-ARC-SWEBOK-DESIGN-CONSENSUS`. Curriculum statements are not upgraded into
  professional-consensus claims by this admission.
- **ISO/IEC/IEEE 42010:2022** is metadata-only (see above); it is never cited as content
  evidence.
- Page anchors are limited to registry-verified locators (SE Overview = PDF page indices
  174-176; Core Hours table = PDF page indices 176-177). The knowledge-unit-level locator
  `se:LOC-CS2023-SE-DESIGN-KU` is heading-anchored with an explicit no-printed-page
  caveat; no page numbers were invented. The SWEBOK locators record exact PDF page
  indices and printed chapter-page folios verified against the hash-checked official
  PDF binary.

## Rights and access restrictions

- CS2023 permission scope is educational-material and program development; Tally's
  general software-engineering knowledge graph is **not presumed** to qualify for that
  permission. No rights clearance is inferred. This lane stores bibliographic metadata,
  exact locators and concise original paraphrases only, but that handling does not itself
  establish permission. Obtain a documented rights determination/specific permission
  before public redistribution or commercial product embedding. Verbatim reuse is limited
  to short topic titles needed for locator anchoring.
- The ISO AI/ML-use notice bars prompting ISO content; ISO records remain bibliographic
  only, and no ISO text was extracted or retained.

## Record counts

| Collection | Count | Notes |
|---|---|---|
| domains | 1 | shared, verbatim |
| taxonomy_views | 1 | shared, verbatim |
| taxonomy_nodes | 1 | `se:NODE-CS2023-SE-SOFTWARE-DESIGN` (KNOWLEDGE_UNIT) |
| source_locators | 7 | 3 shared verbatim + 1 heading-anchored KU locator + 3 SWEBOK KA2/KA3 (PDF-page-index + printed-folio verified) |
| evidence_records | 14 | 10 DIRECT_SOURCE_PARAPHRASE + 1 SYNTHESIS (declared) + 3 SWEBOK KA2/KA3 DIRECT_SOURCE_PARAPHRASE |
| knowledge_claims | 9 | review_status PENDING for all (6 CS2023 + 3 SWEBOK-derived) |
| concepts | 3 | separation of concerns; coupling and cohesion; design abstraction levels |
| principles | 1 | separation of concerns as curriculum-taught guidance |
| practices | 2 | component boundary/dependency identification; API contract documentation |
| patterns | 1 | layered architecture (context required and provided) |
| anti_patterns | 1 | inconsistent API surface (derived from SYNTHESIS evidence) |
| decision_models | 1 | design-tradeoff evaluation |
| coverage_gaps | 4 | ISO 42010 metadata-only; SWEBOK coverage-breadth; real-case evidence; shared CS2023 rights-clearance gap (HIGH/OPEN) |
| relationship_assertions | 16 | 9 SUPPORTED_BY, 2 APPLIES_TO, 1 QUALIFIED_BY, 1 DEPENDS_ON, 1 IMPLEMENTS, 1 DERIVED_FROM, 1 RELATED_TO |

No record is ACCEPTED. All knowledge records are PENDING_REVIEW (shared domain/view are
PROPOSED, locators ACTIVE per the shared snippet contract); all Evidence is
`EXTRACTED_PENDING_REVIEW`, `synthetic: false`; all KnowledgeClaim records are
`review_status: PENDING`. No Case or Counterexample records exist in this lane because no
real-case evidence source is admitted.

## Coverage gaps (first-class results)

1. `se:GAP-ARC-ISO42010-ARCH-DESCRIPTION` (SOURCE_GAP, HIGH, OPEN) — architecture
   description concept unavailable under the ISO metadata-only restriction.
2. `se:GAP-ARC-SWEBOK-DESIGN-CONSENSUS` (SOURCE_GAP, MEDIUM, OPEN) — professional-BOK
   coverage now exists for the reinspected KA2/KA3 sections, but KA2 §4 (architecture
   evaluation) and KA3 §3-6 remain uninspected and no SWEBOK-derived record has passed
   human review; a content/evidence coverage gap, no longer an unmerged dependency.
3. `se:GAP-ARC-DESIGN-REAL-CASES` (SOURCE_GAP, MEDIUM, OPEN) — no real architecture or
   design case evidence source admitted; Pattern/AntiPattern records are curriculum- and
   synthesis-grounded, never case-grounded.
4. `se:GAP-CS2023-RIGHTS-CLEARANCE` (REVIEW_GAP, HIGH, OPEN) — CS2023 educational
   permission is not presumed to cover Tally's general engineering knowledge graph.

## Reproduction

From the worktree root (Python environment with linkml 1.12.1 and jsonschema installed):

```bash
cd docs/versions/v2.1.1/knowledge/model
python validate_knowledge_graph.py --self-test        # expect SELF_TESTS=66/66
python <extract-dir>/assemble_and_validate.py <worktree> 06B \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/records.yaml \
  <validation-outdir>
# expect: all COUNT_* PASS, GRAPH_VALIDATION=PASS, NO_ACCEPTED_RECORDS=PASS,
#         LINKML_INSTANCE_VALIDATION=PASS, LANE_VALIDATION=PASS
```

Validation outputs for this lane are archived outside the repository under
`_tmp/tally-sek06-lanes/validation/06B/` (initial runs) and
`_tmp/tally-sek06-lanes/validation/06B-bld2/` (exact-tip revalidation after the
rights-gap fix); key lines are quoted in
`docs/versions/v2.1.1/methods/observations/SEK-06B.md`.
