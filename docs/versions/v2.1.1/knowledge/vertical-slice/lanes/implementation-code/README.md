# Implementation, Code Engineering & Maintainability Lane — SEK-07B

**Lane ID:** `SEK-07B-IMPLEMENTATION-CODE-LANE-0.1.0`
**Status:** DRAFT / PENDING HUMAN REVIEW
**Task:** [#335](https://github.com/kaicreator-mm/tally/issues/335) (SEK-07 knowledge-taxonomy expansion, coordinator #333, pilot #299)
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4`
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 assembles the lanes by canonical ID)

## Purpose and scope boundary

This lane is the software construction / implementation depth slice of the SEK-07
knowledge expansion. It models code-level construction knowledge from two
authorities:

- **SWEBOK Guide v4.0a KA4 Software Construction** — Introduction and section 1
  (fundamentals), subsections 3.3 (Coding), 3.6 (Construction Quality), 3.7
  (Integration), 4.1 (API Design and Use), 4.4 (Assertions / Design by Contract /
  Defensive Programming) and 4.5 (Error/Exception Handling), directly reinspected
  from the hash-verified official PDF on 2026-11-07.
- **CS2023 curriculum** — the knowledge units SE-Construction: Software
  Construction (CS Core topics 1-2, KA Core topics 3-7), SDF-Practices: Software
  Development Practices (CS Core topics 1-4), the SE-Tools analysis-tooling topic
  (KA Core topic 3 with outcome 4), and the SE-Design code-smell topic (CS Core
  topic 4, from the SEK-06 verified anchors pack).

Scope boundary:

- **In scope:** coding practices and code structure, construction quality
  techniques, code-level integration strategy, API design and use at code level,
  assertions/contracts/defensive programming, error and exception handling,
  static/dynamic analysis at code level, code-level quality and maintainability
  indications.
- **Out of scope (owned by sibling lanes):** secure-coding content detail (07C,
  incl. SWEBOK KA13/SEC-Coding material — this lane records only construction-side
  topic placements and cross-references), reliability/performance engineering
  depth (07D), CI/CD delivery mechanics (07E; SWEBOK 3.7's CI characterization is
  cited only as a construction-side integration option), refactoring /
  technical-debt lifecycle depth (07F), and general architecture design (SEK-06B).
- **No invented real cases.** Neither SWEBOK KA4 nor the cited CS2023 units
  contains real construction case material usable as Case or Counterexample
  evidence, so this lane contains none; the missing case evidence class is
  recorded as `se:GAP-07B-REAL-CASE-EVIDENCE`.
- All knowledge records are `PROPOSED` or `PENDING_REVIEW`; no record is
  `ACCEPTED`. Evidence records are `EXTRACTED_PENDING_REVIEW`, `synthetic: false`
  (one explicitly marked `SYNTHESIS` evidence stays `PROPOSED`). Knowledge claims
  are `review_status: PENDING`.

## Authority fit and limits

- SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`) is used within its registry
  authority: a professional body of knowledge whose KA4 sections were reopened
  directly from the official PDF (SHA-256
  `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 411 pages,
  hash re-verified before extraction on 2026-11-07). The five SWEBOK locators
  record exact PDF page indices and printed chapter-page folios (4-1 to 4-11).
  Registry limitations carry through: the Guide describes generally accepted
  knowledge, not a uniform prescription, and its topic citations are not
  comprehensive. Hedges in the source are preserved verbatim-adjacent in records
  ("is thought to improve" for design by contract; "thought to offer many
  advantages" for incremental integration).
- CS2023 (`se:SRC-CS2023`) is used strictly within its registry authority:
  curriculum knowledge-unit classification and topic placement. It is not
  evidence that any taught practice is effective in industry. The SE-Tools unit
  page anchor is registry-verified (PDF page index 178); the SE-Construction,
  SDF-Practices and SE-Design code-smell locators are **heading-anchored**
  (unit heading + topic numbering verified against the full-text extraction;
  printed pages not independently confirmed, so those locators assert no page
  numbers of their own).
- One evidence record is lane-marked `SYNTHESIS`
  (`se:EVD-07B-SYNTHESIS-BIG-BANG`, `PROPOSED`): it inverts SWEBOK 3.7's stated
  advantages of incremental integration onto its description of phased ("big
  bang") integration to characterize the anti-pattern. It is explicitly not a
  SWEBOK sentence, and the anti-pattern derived from it repeats that boundary.

## Taxonomy view

The lane keeps multiple lenses separate rather than forcing one hierarchy:

- the shared CS2023 SE curriculum view (`se:VIEW-CS2023-SE`) with the
  SE-Construction knowledge-unit node;
- a narrow lane-local CS2023 SDF curriculum view (`se:VIEW-07B-CS2023-SDF`)
  covering only the SDF-Practices unit (CS2023 places it in the SDF area, not SE);
- a professional-lens subdomain node (`se:SUB-07B-SOFTWARE-CONSTRUCTION`, child of
  the shared domain) bounded by the reinspected SWEBOK KA4 Introduction.

The curriculum node and the professional subdomain are linked by one qualified
`RELATED_TO` edge (`se:REL-07B-023`) — a cross-lens mapping, not an equivalence.

## Rights and access restrictions

The CS2023 permission notice grants use for development of educational materials
and programs; other use requires specific permission (ACM / IEEE-CS / AAAI
contacts in `authority/source-reviews/CS2023.md`). This lane therefore stores only
bibliographic metadata, exact locators and concise original paraphrases; short
verbatim topic titles appear only where locator anchoring requires them. No
substantial report text is reproduced, and nothing here assumes commercial
redistribution or product-embedding rights. The unresolved boundary is tracked as
the shared CoverageGap `se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH / OPEN, verbatim
shared snippet). SWEBOK v4.0a is cited under its registered rights handling
(concise original paraphrase; no PDF mirroring or substantial text reproduction).

## Record counts

| Collection | Count | Records |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` (verbatim shared snippet) |
| subdomains | 1 | `se:SUB-07B-SOFTWARE-CONSTRUCTION` |
| taxonomy_views | 2 | shared `se:VIEW-CS2023-SE` + lane-local `se:VIEW-07B-CS2023-SDF` |
| taxonomy_nodes | 2 | `se:NODE-07B-CS2023-SE-CONSTRUCTION`, `se:NODE-07B-CS2023-SDF-PRACTICES` |
| source_locators | 9 | 1 shared verbatim + 4 SWEBOK KA4 (PDF-page-index + printed-folio verified) + 4 CS2023 heading-anchored |
| evidence_records | 14 | 13 `DIRECT_SOURCE_PARAPHRASE` (6 SWEBOK KA4 + 7 CS2023, one of which uses the shared Core Hours table locator) + 1 lane-marked `SYNTHESIS` (PROPOSED) |
| concepts | 4 | cyclomatic complexity (METRIC), design by contract (MECHANISM), defensive programming (MECHANISM), code smell (TERM) |
| knowledge_claims | 9 | 5 SWEBOK KA4 (1 DEFINITION, 3 PRACTICE_CLAIM, 1 NORMATIVE_CLAIM) + 4 CS2023 (3 PRACTICE_CLAIM, 1 TAXONOMY_STATEMENT) |
| principles | 1 | `se:PRI-07B-MINIMIZE-COMPLEXITY` |
| practices | 3 | construct for verification; contract-style interface documentation; analysis-tooling integration |
| patterns | 2 | incremental integration; design by contract at routine level |
| anti_patterns | 1 | big bang (phased) integration as default strategy |
| decision_models | 1 | integration-strategy choice |
| cases / counterexamples | 0 | none (no permitted source supports a real case; see gaps) |
| coverage_gaps | 3 | SWEBOK KA4 breadth; real-case evidence; shared CS2023 rights clearance (HIGH/OPEN) |
| relationship_assertions | 26 | 15 SUPPORTED_BY, 4 APPLIES_TO, 1 IMPLEMENTS, 1 DEPENDS_ON, 1 DERIVED_FROM, 2 RELATED_TO, 1 BOUNDED_BY, 1 QUALIFIED_BY |

Shared records (domain, SE view, Core Hours table locator, rights gap) are copied
verbatim from the SEK-07 lane-shared snippet pack
(`_tmp/tally-sek07-lanes/shared-snippets-sek07.yaml`) so SEK-08 can deduplicate
by canonical ID. Note recorded for reviewers: the shared snippet's domain record
carries a boundary note that ends mid-sentence ("…complete (SEK-02R"); it was
copied byte-identically per the copy-verbatim rule, and the truncation is
reported to the coordinator rather than silently reworded (see
`methods/observations/SEK-07B.md`). Source entities are **not** stored in this
file; the validation harness derives them from `authority/SOURCE_REGISTRY.yaml`.

## Known gaps

1. **SWEBOK KA4 breadth gap.** Only the Introduction, section 1, and subsections
   3.3, 3.6, 3.7, 4.1, 4.4 and 4.5 were admitted as records; the rest of KA4
   (managing construction, construction design/languages, construction testing,
   reuse, cross-platform, most construction technologies, tools) is uncovered
   (`se:GAP-07B-SWEBOK-KA4-BREADTH`, MEDIUM/OPEN).
2. **Real-case gap.** No case-class source is registered; pattern and
   anti-pattern records have no real-world illustration, and the anti-pattern's
   counterproductive nature is a lane-marked synthesis inference
   (`se:GAP-07B-REAL-CASE-EVIDENCE`, MEDIUM/OPEN).
3. **Heading-anchored CS2023 locators.** The SE-Construction, SDF-Practices and
   SE-Design code-smell locators assert no page numbers (PDF obtained as
   extracted text; no binary SHA-256 recorded for CS2023 — see the observation
   note). The SWEBOK locators do not share this limitation: they were verified
   against the hash-checked official PDF binary.
4. **Rights clearance.** CS2023 reuse beyond educational materials/programs is
   unresolved (`se:GAP-CS2023-RIGHTS-CLEARANCE`, HIGH/OPEN, shared).
5. **Hedged source claims.** SWEBOK's own hedges (design-by-contract quality
   effect; incremental-integration advantages) are preserved in claims, pattern
   consequences and a dedicated `QUALIFIED_BY` edge; they must not be promoted to
   unhedged statements during review.

## Reproduction

Run from the worktree root (`VENV` is the SEK venv with Python 3.14.7 and
LinkML 1.12.1):

```bash
VENV=/c/xDev/kAiCreator/_tmp/i321-repair/venv

# 1. Repository validator self-test (expect SELF_TESTS=66/66)
cd docs/versions/v2.1.1/knowledge/model && $VENV/Scripts/python.exe validate_knowledge_graph.py --self-test | tail -1

# 2. Lane assembly + graph validation + count minimums + no-ACCEPTED check + LinkML instance validation
$VENV/Scripts/python.exe /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i335-zcode 07B \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/implementation-code/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07B
```

Expected key lines: every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS`, `DERIVED_SOURCES=2` (`se:SRC-CS2023`,
`se:SRC-SWEBOK-GUIDE-4-0A`). The exact-tip receipt lives in
`_tmp/tally-sek07-lanes/validation/07B/`.

## Review entry point

Reviewers should start at `authority/source-reviews/CS2023.md` and
`authority/source-reviews/SWEBOK-GUIDE-v4.0a.md`, then trace each claim through
its `SUPPORTED_BY` edge to the evidence record, its locator and the derived
source, checking statement fidelity against the verified anchors, the
professional-BOK qualifiers (SWEBOK records) or curriculum-scope qualifiers
(CS2023 records), the preserved hedges, and the rights boundary. Any single
record can be rejected or qualified without editing unrelated records.
