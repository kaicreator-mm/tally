# Requirements Engineering Lane — SEK-06A

**Lane ID:** `SEK-06A-REQUIREMENTS-LANE-0.1.0`
**Status:** DRAFT / PENDING HUMAN REVIEW
**Task:** [#322](https://github.com/kaicreator-mm/tally/issues/322) (SEK-06 vertical-slice lanes)
**Baseline:** `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 assembles the three lanes by canonical ID)

## Purpose and scope boundary

This lane is the Requirements Engineering depth slice of the SEK-06 knowledge
vertical. It models the CS2023 Software Engineering knowledge unit
**SE-Requirements: Product Requirements** (KA Core topics 1-6, Non-core topics
7-9) and its registry-verified surroundings (SE Overview, SE Core Hours table).

Scope boundary:

- **Curriculum authority only.** Every knowledge claim in this lane is scoped to
  the CS2023 curriculum. None is a professional-consensus claim.
- **SWEBOK v4.0a is on source-identity hold** (SEK-02R, #325 unmerged). No SWEBOK
  locator, evidence or claim is cited in this lane. The hold is recorded as a
  CoverageGap (`se:GAP-REQ-PROFESSIONAL-BOK-EVIDENCE`), not silently ignored.
- **No invented real cases.** CS2023 carries no real requirements-engineering
  case material, so this lane contains no Case or Counterexample records; the
  missing case evidence is recorded as `se:GAP-REQ-REAL-CASE-EVIDENCE`.
- All knowledge records are `PROPOSED` or `PENDING_REVIEW`; no record is
  `ACCEPTED`. Evidence records are `EXTRACTED_PENDING_REVIEW`, `synthetic: false`.
  Knowledge claims are `review_status: PENDING`.

## Authority fit and limits

- CS2023 (`se:SRC-CS2023`) is used strictly within its registry authority:
  curriculum knowledge-area / knowledge-unit classification and boundary
  comparison. It is not a professional Software Engineering body of knowledge and
  not evidence that any taught practice is effective in industry.
- The SE Overview and Core Hours page anchors are registry-verified (PDF page
  indices 174-176 and 176-177). The knowledge-unit-level locator
  `se:LOC-CS2023-SE-REQ-KU` is **heading-anchored**: its unit heading and topic
  text were verified against a full-text extraction of the official PDF on
  2026-11-06, but the printed page of the KU subsection was not independently
  confirmed, so the locator asserts no page number of its own.
- One evidence record is a lane-marked `SYNTHESIS`
  (`se:EVD-REQ-SYNTHESIS-QA-DEFERRED`). It combines two verified topic
  statements from the same knowledge unit; it is explicitly not a curriculum
  sentence, and the anti-pattern derived from it repeats that boundary.

## Rights and access restrictions

The CS2023 report's permission notice grants use of the curriculum guidelines
for development of educational materials and programs; other use requires
specific permission (ACM / IEEE-CS / AAAI contacts in
`authority/source-reviews/CS2023.md`). This lane therefore stores only
bibliographic metadata, exact locators and concise original paraphrases of the
verified anchors; short verbatim topic titles appear only where needed for
locator anchoring. No substantial report text is reproduced, and nothing in this
lane assumes commercial redistribution or product-embedding rights. The
unresolved boundary is tracked as a first-class shared CoverageGap record,
`se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH / OPEN, identical across SEK-06A/06B/06C
for SEK-08 deduplication); it stays OPEN until a documented rights determination
or specific permission exists.

## Record counts

| Collection | Count | Records |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` (verbatim shared snippet) |
| taxonomy_views | 1 | shared `se:VIEW-CS2023-SE` (verbatim shared snippet) |
| taxonomy_nodes | 1 | `se:NODE-CS2023-SE-PRODUCT-REQUIREMENTS` |
| source_locators | 4 | 3 shared verbatim + `se:LOC-CS2023-SE-REQ-KU` (heading-anchored) |
| evidence_records | 10 | 9 `DIRECT_SOURCE_PARAPHRASE` + 1 `SYNTHESIS` |
| concepts | 2 | elicitation (TERM), quality attribute (QUALITY_ATTRIBUTE) |
| knowledge_claims | 5 | 2 DEFINITION, 3 PRACTICE_CLAIM |
| principles | 1 | `se:PRI-REQ-EXPECT-CHANGE` |
| practices | 2 | multi-source elicitation, prototype validation |
| patterns | 1 | user-story-structured functional specification |
| anti_patterns | 1 | implicit or deferred quality attributes |
| decision_models | 1 | functional-requirement documentation-method choice |
| coverage_gaps | 3 | professional-BOK gap; real-case gap; shared CS2023 rights-clearance gap (HIGH/OPEN) |
| relationship_assertions | 14 | 7 SUPPORTED_BY, 2 APPLIES_TO, 1 QUALIFIED_BY, 1 DEPENDS_ON, 1 IMPLEMENTS, 1 DERIVED_FROM, 1 RELATED_TO |

Shared records (domain, view, three locators) are copied verbatim from the
lane-shared snippet pack so that SEK-08 integration can deduplicate them by
canonical ID across SEK-06A/06B/06C; the sixth shared record, the
rights-clearance gap `se:GAP-CS2023-RIGHTS-CLEARANCE`, was added by the
coordinated three-lane correction and is byte-identical across the lanes for
the same reason. Source entities are **not** stored in this
file; the validation harness derives them from `authority/SOURCE_REGISTRY.yaml`.

## Known gaps

1. **Professional-BOK gap.** No professional body-of-knowledge evidence for
   requirements engineering exists in this lane while SWEBOK v4.0a is on
   source-identity hold (#325). All claims are curriculum-scoped.
2. **Real-case gap.** No case-class source is registered; pattern and
   anti-pattern records have no real-world illustration, and the anti-pattern's
   counterproductive nature is a lane synthesis inference.
3. **KU locator page limitation.** `se:LOC-CS2023-SE-REQ-KU` is heading-anchored;
   its printed page was not independently confirmed (the underlying PDF was
   obtained as extracted text, not as a byte-identical download, so no PDF
   SHA-256 is recorded either — see the SEK-06A observation note).
4. **Non-core topics.** Prototyping (topic 7) and product evolution (topic 8) are
   elective in CS2023; records resting on them say so in their scope fields.

## Reproduction

Run from the worktree root (`VENV` is the SEK venv with Python 3.14.7 and
LinkML 1.12.1):

```bash
VENV=/c/xDev/kAiCreator/_tmp/tally-i305-builder-zcode/venv

# 1. Repository validator self-test (expect SELF_TESTS=66/66)
cd docs/versions/v2.1.1/knowledge/model && $VENV/Scripts/python.exe validate_knowledge_graph.py --self-test | tail -1

# 2. Lane assembly + graph validation + count minimums + no-ACCEPTED check + LinkML instance validation
$VENV/Scripts/python.exe /c/xDev/kAiCreator/_tmp/tally-sek06-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i322-zcode 06A \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek06-lanes/validation/06A
```

Expected key lines: every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS`. The exact-tip receipt (post rights-clearance fix,
`coverage_gaps=3`) lives in `_tmp/tally-sek06-lanes/validation/06A-bld2/`;
the pre-fix receipt (`coverage_gaps=2`) is retained at
`_tmp/tally-sek06-lanes/validation/06A/` (see
`methods/observations/SEK-06A.md` for the quoted lines).

## Review entry point

Reviewers should start at `authority/source-reviews/CS2023.md`, then trace each
claim through its `SUPPORTED_BY` edge to the evidence record, its locator and
the derived source, checking statement fidelity against the verified anchors,
the curriculum-scope qualifiers, and the rights boundary. Any single record can
be rejected or qualified without editing unrelated records.
