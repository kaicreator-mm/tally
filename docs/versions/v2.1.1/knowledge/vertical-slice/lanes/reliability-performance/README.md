# Reliability, Resilience & Performance Lane — SEK-07D

**Lane ID:** `SEK-07D-RELIABILITY-PERFORMANCE-LANE-0.1.0`
**Status:** DRAFT / PENDING HUMAN REVIEW
**Task:** [#337](https://github.com/kaicreator-mm/tally/issues/337) (SEK-07 knowledge-taxonomy lanes)
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4`
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 performs single-writer graph materialization by canonical ID)

## Purpose and scope boundary

This lane is the reliability, resilience and performance depth slice of the
SEK-07 knowledge expansion. It models the CS2023 Software Engineering knowledge
unit **SE-Reliability: Software Reliability** (KA Core topics 1-6, Non-core
topics 7-11) together with directly reinspected SWEBOK Guide v4.0a engineering
sections: KA6 **Software Engineering Operations** sections 1.6 (performance,
reliability and load balancing), 2.3 (availability, continuity and service
levels), 2.4 (capacity management), 2.5 (backup, disaster recovery and
failover) and 4.3 (monitor, measure, track and review), and KA12 **Software
Quality** sections 1.4 (software dependability and integrity levels) and 2.3.1
(software quality measurement).

Scope boundary:

- **Reliability, resilience, availability and performance are kept distinct.**
  Software reliability (failure quantification), managed availability (SLA
  loop), fault tolerance (context-dependent response) and capacity/performance
  (forecast-driven planning) each get separate records; the curriculum's own
  separation of software reliability from system reliability is preserved as a
  boundary claim (`se:CLM-07D-SOFTWARE-VS-SYSTEM-RELIABILITY`).
- **Curriculum framing stays curriculum-scoped.** CS2023 records are teaching
  expectations, not professional consensus; SWEBOK records are scoped to the
  directly reinspected sections and are not automatically accepted.
- **KA12 quality-model content is not cited.** Only KA12's engineering
  sections (failure-risk-reduction techniques; quality measurement) are
  admitted; model-level quality definitions belong to SEK-07A. Security
  operations stay with SEK-07C and CI/CD delivery mechanics with SEK-07E.
- **No invented real cases.** Neither source carries real reliability case
  material, so this lane contains no Case or Counterexample records; the
  missing case evidence is recorded as `se:GAP-07D-NO-REAL-CASE-EVIDENCE`.
- All knowledge records are `PROPOSED` or `PENDING_REVIEW`; no record is
  `ACCEPTED`. Evidence records are `EXTRACTED_PENDING_REVIEW`, `synthetic:
  false`. Knowledge claims are `review_status: PENDING`.

## Authority fit and limits

- CS2023 (`se:SRC-CS2023`) is used strictly within its registry authority:
  curriculum knowledge-unit classification. The SE-Reliability unit's heading
  and topics were verified against a full-text extraction of the official Body
  of Knowledge PDF obtained via a text-extraction service on 2026-11-07
  (direct binary download fails from this network; no PDF SHA-256 is recorded
  for CS2023). The unit-level locator
  `se:LOC-07D-CS2023-SE-RELIABILITY-KU` is **heading-anchored** and asserts no
  page number of its own beyond the registry-verified SE section span.
- SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`) is used within its registry
  authority: a professional body of knowledge. The five KA6 sections and two
  KA12 sections were reopened directly from the official PDF (SHA-256
  `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 411
  pages, hash re-verified against the registry record on 2026-11-07). The
  seven SWEBOK locators record exact 0-based PDF page indices and printed
  chapter-page folios. Registry limitations carry through: the Guide is a
  guide to generally accepted knowledge, not a uniform prescription, and its
  topic citations are not comprehensive. KA6 section 2.4 relays
  ISO/IEC/IEEE 20000-1; the ISO standard itself remains registry
  metadata-only and backs no content claim here.
- One evidence record is a lane-marked `SYNTHESIS`
  (`se:EVD-07D-SYNTHESIS-HOPE-BASED-OPS`). It combines one SWEBOK norm with
  one CS2023 topic; it is explicitly not a sentence from either source, and
  the anti-pattern derived from it repeats that boundary.
- The taxonomy is deliberately multi-view: the shared curriculum view
  `se:VIEW-CS2023-SE` carries the SE-Reliability knowledge-unit node, while
  the lane's own operational view `se:VIEW-07D-RELIABILITY-PERFORMANCE`
  classifies reliability/resilience, performance/capacity and
  observability-feedback concerns. The two views are linked by an explicit
  typed edge (`se:REL-07D-030`), not merged; reliability and performance are
  additionally related by a `CROSS_CUTS` edge (`se:REL-07D-023`) because the
  inspected SWEBOK section plans them together.

## Rights and access restrictions

The CS2023 report's permission notice grants use of the curriculum guidelines
for development of educational materials and programs; other use requires
specific permission (ACM / IEEE-CS / AAAI contacts in
`authority/source-reviews/CS2023.md`). This lane therefore stores only
bibliographic metadata, exact locators and concise original paraphrases of
the verified anchors; short verbatim topic titles appear only where needed
for locator anchoring. SWEBOK content is handled per its registry rights
record: direct content reinspection with concise original paraphrase only; no
PDF mirroring, no substantial reproduction, no inferred redistribution
permission. The unresolved CS2023 boundary is tracked as the first-class
shared CoverageGap `se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH / OPEN, identical
across lanes for SEK-08 deduplication).

## Record counts

| Collection | Count | Records |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` (verbatim shared snippet) |
| taxonomy_views | 2 | shared `se:VIEW-CS2023-SE` + lane `se:VIEW-07D-RELIABILITY-PERFORMANCE` (OPERATIONS) |
| taxonomy_nodes | 4 | CS2023 SE-Reliability unit (KNOWLEDGE_UNIT) + reliability/resilience, performance/capacity (QUALITY_ATTRIBUTE) + observability feedback (CONCERN) |
| source_locators | 9 | 1 shared verbatim (`se:LOC-CS2023-SE-KU-TABLE`) + 1 CS2023 heading-anchored KU locator + 7 SWEBOK (5 KA6 + 2 KA12; PDF-page-index + printed-folio verified) |
| evidence_records | 16 | 8 CS2023 `DIRECT_SOURCE_PARAPHRASE` + 7 SWEBOK `DIRECT_SOURCE_PARAPHRASE` + 1 lane `SYNTHESIS` |
| concepts | 4 | software reliability (QUALITY_ATTRIBUTE), error budget (TERM), fault tolerance (MECHANISM), SLA (TERM) |
| knowledge_claims | 13 | 6 CS2023 (2 DEFINITION, 3 PRACTICE_CLAIM, 1 DOMAIN_BOUNDARY) + 7 SWEBOK (2 DEFINITION, 4 PRACTICE_CLAIM, 1 NORMATIVE_CLAIM) |
| principles | 2 | plan for failure; evidence over hope |
| practices | 4 | defect injection/removal, capacity planning, reliability telemetry, failover rehearsal |
| patterns | 2 | consequence-matched failure response; SLA-anchored availability |
| anti_patterns | 1 | hope-based operations (derived from lane synthesis) |
| decision_models | 2 | error budget vs error-free; capacity investment |
| coverage_gaps | 4 | real-case gap; CS2023 SF/PDC adjacent-topics gap; SWEBOK uninspected-sections gap; shared CS2023 rights-clearance gap (HIGH/OPEN) |
| relationship_assertions | 30 | 13 SUPPORTED_BY claim traces + 3 principle/pattern/DM SUPPORTED_BY + 3 APPLIES_TO + 3 IMPLEMENTS + 1 DEPENDS_ON + 1 DERIVED_FROM + 1 RELATED_TO + 1 CROSS_CUTS + 1 QUALIFIED_BY + 1 cross-view RELATED_TO + 2 claim-node/pattern traces |

Shared records (domain, curriculum view, Core-Hours locator, rights gap) are
copied verbatim from the lane-shared snippet pack so SEK-08 integration can
deduplicate them by canonical ID across the SEK-07 lanes. Source entities are
**not** stored in this file; the validation harness derives them from
`authority/SOURCE_REGISTRY.yaml`.

## Known gaps

1. **Real-case gap.** No case-class source (incident report, postmortem,
   capacity case study) is registered for this scope; pattern and anti-pattern
   records have no real-world illustration, and the anti-pattern's
   counterproductive nature is a lane synthesis inference
   (`se:GAP-07D-NO-REAL-CASE-EVIDENCE`, MEDIUM/OPEN).
2. **CS2023 adjacent-topics gap.** SF-Reliability and PDC-Evaluation topics
   exist in the full CS2023 BOK but are absent from the prepared extraction,
   and the lane could not re-fetch them (direct download fails; archive
   rate-limited), so they are recorded as a locator gap rather than cited
   (`se:GAP-07D-CS2023-ADJACENT-TOPICS`, MEDIUM/OPEN).
3. **SWEBOK coverage-breadth gap.** Only KA6 sections 1.6/2.3/2.4/2.5/4.3 and
   KA12 sections 1.4/2.3.1 were reinspected; other reliability-relevant
   sections (KA6 3.2/3.3, 5.1, 5.2, 6.4; the rest of KA12) remain uninspected
   and no SWEBOK-derived record has passed human review
   (`se:GAP-07D-SWEBOK-UNINSPECTED-SECTIONS`, MEDIUM/OPEN).
4. **Non-core anchoring.** The CS2023 fault-tolerance context contrast and
   measurement-based analysis topics are Non-core (elective); records resting
   on them say so in their scope fields.
5. **KU locator page limitation.** The CS2023 KU locator is heading-anchored;
   its printed page was not independently confirmed (text-extraction
   constraint above). The SWEBOK locators do not share this limitation.

## Reproduction

Run from the worktree root (`VENV` is the pinned SEK venv with Python 3.14.7,
LinkML 1.12.1):

```bash
VENV=/c/xDev/kAiCreator/_tmp/i321-repair/venv
export PYTHONDONTWRITEBYTECODE=1

# 1. Repository validator self-test (expect SELF_TESTS=66/66, exit 0)
$VENV/Scripts/python.exe docs/versions/v2.1.1/knowledge/model/validate_knowledge_graph.py --self-test

# 2. Lane assembly + graph validation + count minimums + no-ACCEPTED check + LinkML instance validation
$VENV/Scripts/python.exe /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i337-zcode 07D \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/reliability-performance/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07D
```

Expected key lines: every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS`, `DERIVED_SOURCES=2` (`se:SRC-CS2023`,
`se:SRC-SWEBOK-GUIDE-4-0A`); 94 lane records, 30 relationship assertions.

## Review entry point

Reviewers should start at `authority/source-reviews/CS2023.md` and
`authority/source-reviews/SWEBOK-GUIDE-v4.0a.md`, then trace each claim
through its `SUPPORTED_BY` edge to the evidence record, its locator and the
derived source, checking statement fidelity against the verified anchors, the
curriculum-scope qualifiers (CS2023 records) or inspected-section qualifiers
(SWEBOK records), and the rights boundary. Any single record can be rejected
or qualified without editing unrelated records.
