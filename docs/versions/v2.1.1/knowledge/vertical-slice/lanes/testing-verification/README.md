# SEK-06C Lane — Software Testing, Verification and Validation

**Lane ID:** `SEK-06C-TESTING-VERIFICATION-LANE-0.1.0`
**Status:** DRAFT_PENDING_HUMAN_REVIEW
**Task:** #324 (parent), refs #321, #299
**Base HEAD:** `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`
**Created:** 2026-11-06

## Purpose and scope

This lane is the Software Testing, Verification and Validation deep vertical
slice of the SEK-06 lane family. It is a lane collection file
(`records.yaml`), not a graph root; SEK-08 single-writer integration assembles
lanes plus registry-derived Source entities into the full graph.

Two admitted evidence families:

- **CS2023 (`se:SRC-CS2023`) — curriculum authority only.** The
  `SE-Validation: Software Verification and Validation` knowledge unit
  (CS Core topics 1-5, KA Core topics 6-9, Non-core topics 10-14) supplies the
  V&V concept questions, testing objectives, test kinds, DAMP-vs-DRY test
  style, test planning/coverage dimensions, test development techniques and
  tooling, development-cycle V&V (pre/post-commit, defect tracking) and
  domain-specific V&V topics. The `SE-Tools` unit supplies the curriculum
  placement of testing tools.
- **NIST SP 800-218 v1.1 (`se:SRC-NIST-SP-800-218-V1-1`) — secure-development
  cross-cutting source only.** Page-level verified content for PW.7 (code
  review/analysis), PW.8 (executable code testing) and the RV group
  (RV.1/RV.2/RV.3), plus the registry-verified Section 1 risk-based adoption
  guidance that qualifies every NIST-sourced record.

SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`) — identity/registry
reconciliation is complete (SEK-02R #325 merged via PR #326 into this lane's
WIP base), and the 2026-11-06 P2 repair reinspected KA5 Software Testing
sections 1 (1.1, 1.2.7, 1.2.8), 2 (2.1 incl. 2.1.1-2.1.4, 2.2) and 3 (intro,
3.1, 3.2, 3.3 heading) directly from the official PDF (SHA-256
`0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 411
pages), admitting three locators, three evidence records and three claims. All
SWEBOK-derived records stay PENDING_REVIEW with the Guide's
generally-accepted-knowledge caution carried inside each claim; the
coverage-breadth limitation is recorded as `se:GAP-TVV-SWEBOK-KA5-HOLD`.

Out of scope by design: SWEBOK KA5 topics beyond the reinspected sections;
ISO/IEC/IEEE 29119-2 content (ACCESS_RESTRICTED/METADATA_ONLY); any real
engineering case or incident content (no empirical source registered — see
gaps).

## Authority fit and limitations

- CS2023 is a curriculum framework. Its records here claim curriculum content,
  never professional-domain completeness, and never professional consensus.
- NIST SSDF v1.1 is a recommendation framework. Risk, cost, feasibility and
  applicability govern practice selection; it is not a checklist, and
  practice/task/example ordering (including Table 1 order) implies no
  implementation sequence or relative importance. No lane record may be read
  as a universal mandate or as a prescribed order of work.
- NIST notional implementation examples (for example the PW.8.2 regression
  test example) are explicitly neither required nor exhaustive; they ground
  candidate patterns but were **not** promoted into Case records or mandatory
  practice claims.
- CS2023 provenance limit: the official Body of Knowledge PDF could not be
  downloaded as a binary during this lane's research, so **no binary SHA-256
  was obtained for the CS2023 PDF**; the 2026-11-06 text extraction is
  evidence of extracted content, not byte-level PDF evidence. Page anchors
  are therefore restricted to registry-verified anchors, and the new KU-level
  locator (`se:LOC-CS2023-SE-VALIDATION-KU`) is heading-anchored and asserts
  no page number of its own (method and anchors in the SEK-06C observation).
- Per parent issue #324: nothing in this lane equates executing tests with
  establishing correctness, or validation with user acceptance. The two
  conflations are named and rejected in `se:CON-TVV-VERIFICATION-VALIDATION`,
  `se:CLM-TVV-0001` limitations, the coverage-theater anti-pattern causes, and
  `EXPLANATIONS.md`.

## Rights

- CS2023: the registry records permission for educational materials/programs
  and requires specific permission for other uses. Tally's general software-
  engineering knowledge graph is **not presumed** to fall within the educational
  permission scope; no license clearance is inferred. This lane stores only
  metadata, exact locators and concise original paraphrases, but that handling
  does not itself establish permission. Obtain a rights determination/specific
  permission before public redistribution or commercial product embedding
  (ACM/IEEE-CS/AAAI contacts in the source dossier).
- NIST SP 800-218 v1.1: US-government publication not subject to US copyright;
  voluntary use; official citation appreciated
  (Souppaya, Scarfone, Dodson, NIST SP 800-218, February 2022,
  DOI 10.6028/NIST.SP.800-218). Concise paraphrase plus locator retained.

## Record counts

| Collection | Count | Highlights |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` |
| taxonomy_views | 2 | shared `se:VIEW-CS2023-SE`; new `se:VIEW-SSDF-SECURITY` |
| taxonomy_nodes | 3 | `se:NODE-CS2023-SE-VALIDATION`, `se:NODE-CS2023-SE-TOOLS`, `se:NODE-SSDF-RV` |
| source_locators | 12 | 3 shared + 2 CS2023 lane + 4 NIST page-level + 3 SWEBOK KA5 (PDF-page-index + printed-folio verified) |
| evidence_records | 18 | 14 DIRECT_SOURCE_PARAPHRASE + 1 marked SYNTHESIS (`se:EVD-TVV-0014`) + 3 SWEBOK KA5 DIRECT_SOURCE_PARAPHRASE |
| concepts | 2 | V&V boundary concept; test coverage |
| knowledge_claims | 10 | all `review_status: PENDING` (7 curriculum/NIST + 3 SWEBOK-derived) |
| principles | 1 | DAMP over DRY in test code |
| practices | 2 | context-gated executable code testing; defect tracking with reproducibility |
| patterns | 1 | regression safety net (candidate, no real case) |
| anti_patterns | 1 | coverage theater (synthesis-derived) |
| decision_models | 1 | pre-commit vs post-commit test placement |
| coverage_gaps | 4 | ISO 29119-2 metadata-only; SWEBOK KA5 coverage-breadth; no empirical cases; CS2023 rights clearance |
| relationship_assertions | 23 | 16 SUPPORTED_BY, 2 APPLIES_TO, 1 QUALIFIED_BY, 1 DEPENDS_ON, 1 IMPLEMENTS, 1 DERIVED_FROM, 1 CROSS_CUTS |

Lifecycle contract: all knowledge records `PROPOSED`/`PENDING_REVIEW`; all
evidence `EXTRACTED_PENDING_REVIEW`, `synthetic: false`; no `ACCEPTED`
records; locators `ACTIVE`/`LOCATOR_VERIFIED` per the shared-snippet
convention. Shared records are byte-identical copies from
`_tmp/tally-sek06-lanes/extract/shared-snippets.yaml` so SEK-08 can
deduplicate by canonical ID.

## Known gaps

1. `se:GAP-TVV-ISO-29119-2-METADATA-ONLY` — no normative test-process standard
   content is citable; curriculum-to-ISO alignment unverified.
2. `se:GAP-TVV-SWEBOK-KA5-HOLD` — professional-BOK coverage now exists for
   the reinspected KA5 fundamentals/levels/technique sections, but the
   remaining KA5 topics (measures, process, development-process and domain
   testing, emerging technologies, tools) are uninspected and no
   SWEBOK-derived record has passed human review; a content/evidence coverage
   gap, no longer an unmerged dependency.
3. `se:GAP-TVV-NO-EMPIRICAL-CASES` — no Case/Counterexample records; pattern
   and anti-pattern records lack real-world case grounding.
4. `se:GAP-CS2023-RIGHTS-CLEARANCE` — the source's educational-material
   permission is not assumed to cover Tally's general engineering knowledge
   graph; specific permission/rights determination remains open.

Also unpopulated: SSDF PO/PS groups (the security view is PARTIAL by design).

## Reproduction

```bash
VENV=/c/xDev/kAiCreator/_tmp/tally-i305-builder-zcode/venv
cd /c/xDev/kAiCreator/tally/tally-i324-zcode/docs/versions/v2.1.1/knowledge/model \
  && $VENV/Scripts/python.exe validate_knowledge_graph.py --self-test | tail -1
# expected: SELF_TESTS=66/66

$VENV/Scripts/python.exe \
  /c/xDev/kAiCreator/_tmp/tally-sek06-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i324-zcode 06C \
  /c/xDev/kAiCreator/tally/tally-i324-zcode/docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek06-lanes/validation/06C
# expected: COUNT_* PASS, GRAPH_VALIDATION=PASS, NO_ACCEPTED_RECORDS=PASS,
#           LINKML_INSTANCE_VALIDATION=PASS, LANE_VALIDATION=PASS
```

Validation outputs land in `_tmp/tally-sek06-lanes/validation/06C` (outside
the repo write set). Narrative interpretation lives in `EXPLANATIONS.md`;
research method and anchor provenance live in
`docs/versions/v2.1.1/methods/observations/SEK-06C.md`.
