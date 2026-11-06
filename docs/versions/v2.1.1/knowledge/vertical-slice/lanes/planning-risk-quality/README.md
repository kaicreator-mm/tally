# Engineering Planning, Risk, Quality Management & Technical Decisions Lane — SEK-07G

**Lane ID:** `SEK-07G-PLANNING-RISK-QUALITY-LANE-0.1.0`
**Status:** DRAFT / PENDING HUMAN REVIEW
**Task:** [#340](https://github.com/kaicreator-mm/tally/issues/340) (SEK-07 knowledge-taxonomy lanes, coordinator #333, parent pilot #299)
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (tip of `wip/v2.1.1-knowledge`)
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 assembles the lanes by canonical ID)

## Purpose and scope boundary

This lane is the planning / risk / quality-management / measurement /
technical-decision depth slice of SEK-07. It is grounded in direct
reinspection of two SWEBOK Guide v4.0a chapters from the official
hash-verified PDF — Chapter 9 *Software Engineering Management*
(Introduction, section 2 Software Project Planning incl. 2.3 estimation,
2.5 risk, 2.6 quality, 2.7 plan management, and section 6 Software
Engineering Measurement) and Chapter 15 *Software Engineering Economics*
(Introduction, section 2 The Engineering Decision-Making Process,
section 8 Estimation) — plus the CS2023 **SE-Teamwork** knowledge unit
(team processes incl. effort estimation; distributed-team risks) and the
CS2023 **SE-Requirements** Non-core effort-estimation topic.

Scope boundary:

- **Not generic business/project-management knowledge.** Every record is
  scoped to the software-engineering treatment in the inspected sources;
  where those sources lean on external standards (PMBOK Guide, SWX,
  ISO/IEC/IEEE 15939, ISO 12207/15288), the lane cites only SWEBOK's
  secondary characterization and records a CoverageGap
  (`se:GAP-07G-NO-PM-STANDARD-SOURCE`) because no such standard is
  registered.
- **Quality management in, quality models and attributes out.** The
  planning-side quality-management direction of KA9 §2.6 is in scope;
  quality MODELS stay in lane 07A territory and quality ATTRIBUTES as
  engineering detail stay in 07D. SWEBOK KA12 (Software Quality) itself
  was not in this lane's reading plan; that limit is a first-class gap
  (`se:GAP-07G-KA12-QUALITY-UNINSPECTED`).
- **Lifecycle/process models (07A), implementation practices (07B),
  reliability/performance (07D), delivery mechanics (07E), technical-debt
  lifecycle (07F) are not absorbed.** The lane view records that SWEBOK's
  activity-based breakdown explicitly does not recommend an SDLC.
- **No invented numbers.** No numeric productivity or estimation-accuracy
  claims are made; technique characterizations (e.g., "parametric
  typically most accurate") are reported as SWEBOK's overview statements,
  not as empirical rankings.
- **No real cases.** No case-class source is registered, so the lane
  contains no Case or Counterexample records; the missing evidence class
  is recorded as `se:GAP-07G-NO-REAL-CASE-EVIDENCE`.
- All knowledge records are `PROPOSED` or `PENDING_REVIEW`; none is
  `ACCEPTED`. Evidence records are `EXTRACTED_PENDING_REVIEW`,
  `synthetic: false` (one `SYNTHESIS` evidence record is lane-marked).
  Knowledge claims are `review_status: PENDING`.

## Authority fit and limits

- SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`) is used within its
  registry authority: a professional body of knowledge whose cited
  sections were reopened directly from the official PDF (SHA-256
  `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
  411 pages, hash re-verified 2026-11-07). All seven SWEBOK locators
  record exact PDF page indices (0-based) and printed chapter folios.
  Registry limitations carry through: the Guide presents generally
  accepted knowledge, not a uniform prescription, and its references are
  not comprehensive. SWEBOK's characterizations of PMBOK/SWX/ISO
  standards are secondary statements and are labeled as such in the
  affected records.
- CS2023 (`se:SRC-CS2023`) is used strictly within its registry
  authority: curriculum knowledge-unit classification. The two CS2023
  locators are **heading-anchored** (exact unit heading + topic
  numbering, verified against the full-text extraction of 2026-11-07);
  their printed pages were not independently confirmed, so those
  locators assert no page numbers of their own beyond the
  registry-verified SE section span. The shared Core-Hours locator
  (`se:LOC-CS2023-SE-KU-TABLE`) is registry page-verified and grounds the
  Teamwork hours evidence.
- SWEBOK KA14 (Professional Practice) was optional in this lane's source
  plan and was **not** inspected; the Software Engineering Code of Ethics
  clause 3.09 used by the estimation records is cited only as quoted
  inside KA15 §8, never as an independent inspection of the Code.
- The taxonomy hypothesis is preserved, not flattened: the lane keeps the
  CS2023 curriculum lens (`se:VIEW-CS2023-SE`) and a SWEBOK process lens
  (`se:VIEW-07G-SWEBOK-SEM-PLANNING`) separate and records their
  estimation-topic intersection as an explicit `CROSS_CUTS` edge rather
  than merging hierarchies.

## Rights and access restrictions

The CS2023 permission notice grants use for development of educational
materials and programs; other use requires specific permission (ACM /
IEEE-CS / AAAI contacts in `authority/source-reviews/CS2023.md`). This
lane therefore stores only bibliographic metadata, exact locators and
concise original paraphrases of the verified anchors; short verbatim
topic titles appear only where needed for locator anchoring. SWEBOK
content is used per the registry rights handling (direct content
reinspection, concise original paraphrase only). No PDF or full-text
extract is committed, no substantial source text is reproduced, and
nothing here assumes commercial redistribution or product-embedding
rights. The unresolved CS2023 boundary is tracked as the shared
CoverageGap `se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH / OPEN), carried
verbatim for SEK-08 deduplication.

## Record counts

| Collection | Count | Records |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` (verbatim shared snippet) |
| taxonomy_views | 2 | shared `se:VIEW-CS2023-SE` (verbatim) + `se:VIEW-07G-SWEBOK-SEM-PLANNING` (PROCESS lens) |
| taxonomy_nodes | 5 | 4 SWEBOK KA9 practice groups (planning, risk, quality, measurement) + `se:NODE-07G-CS2023-SE-TEAMWORK` |
| source_locators | 10 | 7 SWEBOK (4 KA9 + 3 KA15; PDF-index + printed-folio verified) + 2 CS2023 heading-anchored + shared `se:LOC-CS2023-SE-KU-TABLE` |
| evidence_records | 11 | 10 `DIRECT_SOURCE_PARAPHRASE` (7 SWEBOK, 3 CS2023) + 1 lane-marked `SYNTHESIS` |
| concepts | 4 | estimate (TERM), risk-vs-uncertainty (BOUNDARY_CONCEPT), risk register (TERM), information need (TERM) |
| knowledge_claims | 12 | SWEBOK: 1 DEFINITION, 2 NORMATIVE_CLAIM, 4 PRACTICE_CLAIM (estimation, risk, quality, measurement) + 1 PRACTICE_CLAIM (decision process) + 1 LIMITATION; CS2023: 2 PRACTICE_CLAIM + 1 LIMITATION |
| principles | 2 | pair management with measurement; close the loop on estimates (cross-source: SWEBOK KA15 §2.7 + CS2023 Non-core topic 9b) |
| practices | 3 | risk-register maintenance; multi-technique estimation; needs-driven measurement |
| patterns | 1 | common-basis multi-attribute trade-off evaluation |
| anti_patterns | 1 | single-point estimate treated as a commitment |
| decision_models | 1 | estimation-technique selection |
| cases / counterexamples | 0 | no case-class source registered (see gap) |
| review_decisions | 0 | human review pending; none permitted from a builder |
| coverage_gaps | 4 | PM/measurement-standard source gap; KA12 not inspected; no real-case evidence; shared CS2023 rights clearance (HIGH/OPEN) |
| relationship_assertions | 32 | 21 SUPPORTED_BY (12 claims, 3 principle edges, 1 pattern, 1 decision model, 4 concepts), 2 APPLIES_TO, 3 IMPLEMENTS, 2 PART_OF, 1 each QUALIFIED_BY / DEPENDS_ON / DERIVED_FROM / CROSS_CUTS |

Shared records (domain, view, rights gap, Core-Hours locator) are copied
verbatim from the lane-shared snippet pack
(`_tmp/tally-sek07-lanes/shared-snippets-sek07.yaml`) so SEK-08 can
deduplicate by canonical ID; the snippet pack's second domain boundary
note is a truncated fragment of the SEK-06 wording and was copied
verbatim per the shared-record rule (flagged in the observation note,
not reworded). Source entities are **not** stored in this file; the
validation harness derives them from `authority/SOURCE_REGISTRY.yaml`.

## Known gaps

1. **No project-management or measurement standard source** (MEDIUM/OPEN,
   `se:GAP-07G-NO-PM-STANDARD-SOURCE`): PMBOK Guide, SWX and
   ISO/IEC/IEEE 15939 (and ISO 12207/15288) are unregistered; affected
   records cite SWEBOK's characterization only. Proposed for a separate
   single-writer source task: *PMBOK Guide 6th ed. (PMI 2017)*,
   *Software Extension to the PMBOK Guide (PMI 2013)*,
   *ISO/IEC/IEEE 15939:2017*.
2. **SWEBOK KA12 not inspected** (MEDIUM/OPEN,
   `se:GAP-07G-KA12-QUALITY-UNINSPECTED`): quality-management coverage is
   limited to KA9 §2.6's planning-side direction.
3. **No real case evidence** (MEDIUM/OPEN,
   `se:GAP-07G-NO-REAL-CASE-EVIDENCE`): pattern/anti-pattern records lack
   real-world illustration; the anti-pattern's counterproductive nature
   is a lane synthesis inference.
4. **CS2023 rights clearance** (HIGH/OPEN, shared
   `se:GAP-CS2023-RIGHTS-CLEARANCE`): educational-scope permission is not
   presumed to cover this knowledge graph.
5. **Heading-anchored CS2023 locators**: the two lane CS2023 locators
   assert no page numbers of their own (extraction was text-based; no
   binary SHA-256 of the CS2023 PDF exists — inherited SEK-06 constraint).

## Reproduction

Run from the worktree root (`VENV` is the SEK venv with Python 3.14.7,
LinkML 1.12.1):

```bash
VENV=/c/xDev/kAiCreator/_tmp/i321-repair/venv

# 1. Repository validator self-test (expect SELF_TESTS=66/66)
cd docs/versions/v2.1.1/knowledge/model && $VENV/Scripts/python.exe validate_knowledge_graph.py --self-test | tail -1

# 2. Lane assembly + graph validation + count minimums + no-ACCEPTED check + LinkML instance validation
$VENV/Scripts/python.exe /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i340-zcode 07G \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/planning-risk-quality/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07G
```

Expected key lines: every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS`, with `DERIVED_SOURCES=2` (`se:SRC-CS2023`,
`se:SRC-SWEBOK-GUIDE-4-0A`). The exact-tip receipts (identity, manifest,
logs) live in `_tmp/tally-sek07-lanes/validation/07G/` and are quoted in
`methods/observations/SEK-07G.md`.

## Review entry point

Reviewers should start at `authority/source-reviews/SWEBOK-GUIDE-v4.0a.md`
and `authority/source-reviews/CS2023.md`, then trace each claim through
its `SUPPORTED_BY` edge to the evidence record, its locator and the
derived source — checking paraphrase fidelity against the verified page
anchors (SWEBOK) or unit headings (CS2023), the scope qualifiers
(SWEBOK "generally accepted knowledge, not a uniform prescription";
CS2023 curriculum scope; CS2023-derived Non-core status where relevant),
the secondary-characterization labels on standards-touching records, and
the rights boundary. Any single record can be rejected or qualified
without editing unrelated records.
