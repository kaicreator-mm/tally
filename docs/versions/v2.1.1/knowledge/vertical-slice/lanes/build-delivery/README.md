# Integration, Build, CI/CD & Delivery Lane — SEK-07E

**Lane ID:** `SEK-07E-BUILD-DELIVERY-LANE-0.1.0`
**Status:** DRAFT / PENDING HUMAN REVIEW
**Task:** [#338](https://github.com/kaicreator-mm/tally/issues/338) (SEK-07 knowledge-taxonomy expansion lanes; coordinator #333, parent pilot #299)
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (= tip of `wip/v2.1.1-knowledge`; merge-base equals base)
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 performs single-writer graph materialization/export)

## Purpose and scope boundary

This lane is the SEK-07E depth slice: software integration, version and
configuration interaction at build boundaries, build systems, dependency and
integration verification, continuous integration, delivery pipelines, release
engineering, packaging and deployment-process design as engineering knowledge.

Grounding:

- **SWEBOK Guide v4.0a KA6 Software Engineering Operations** — sections
  2.1.1 (release/version planning tasks), 2.2 (development and operational
  environments), 3.2 (Deployment/Release Engineering) and 3.3 (Rollback and
  Data Migration), directly inspected from the official hash-verified PDF.
- **SWEBOK Guide v4.0a KA8 Software Configuration Management** — sections
  1.3–1.4 (SCM planning, incl. the continuous-integration characterization),
  2.5–2.6 (CI relationships/SBOM and software libraries) and 6 with 6.1/6.2
  (Software Release Management and Delivery: Software Building; Software
  Release Management), directly inspected from the same PDF.
- **CS2023 SE-Tools: Tools and Environments** — CS Core topic 1 (software
  configuration management and version control incl. reproducible
  builds/configuration and branching strategies), KA Core topics 2 and 4
  (release management; software process automation: build systems, CI,
  dependency management) and illustrative learning outcomes 4–6, verified
  against the official Body of Knowledge full-text extraction.

Scope boundary:

- Implementation/code practices (07B), reliability/performance engineering
  detail (07D), maintenance and technical-debt/SCM-theory beyond build and
  integration boundaries (07F) and planning/estimation (07G) stay in their
  sibling lanes; this lane records only integration/build/delivery mechanics.
- **No invented real cases.** Neither SWEBOK nor CS2023 provides real
  build/delivery case material and no case-class source is registered, so this
  lane contains no Case or Counterexample records; the missing evidence
  classes are recorded as `se:GAP-07E-REAL-CASE-EVIDENCE` and
  `se:GAP-07E-NO-EMPIRICS`.
- All knowledge records are `PROPOSED` or `PENDING_REVIEW`; none is
  `ACCEPTED`. Evidence records are `EXTRACTED_PENDING_REVIEW`,
  `synthetic: false` (one lane-marked `SYNTHESIS` evidence). Knowledge claims
  are `review_status: PENDING`.

## Authority fit and limits

- SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`) is used within its registry
  authority: a professional body of knowledge whose cited KA6/KA8 sections
  were reopened directly from the official PDF (SHA-256
  `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`, 411
  pages) on 2026-11-07. All SWEBOK locators record exact PDF page indices and
  printed chapter-page folios. Registry limitations carry through: the Guide
  describes generally accepted knowledge, is not a uniform prescription, and
  its citations are not comprehensive. The PDF/landing-page release-date
  discrepancy recorded by SEK-02R remains OPEN.
- CS2023 (`se:SRC-CS2023`) is used strictly within its registry authority:
  curriculum knowledge-unit classification and topic statements. It is not a
  professional body of knowledge and not evidence that any taught practice is
  effective in industry. The SE-Tools unit has the registry-verified page
  anchor (PDF page index 178 / one-based 179); the unit-level locator
  `se:LOC-07E-CS2023-SE-TOOLS-KU` is **heading-anchored** (exact unit heading
  + topic numbering verified against the full-text extraction on 2026-11-07);
  it asserts no printed page number of its own. The CS2023 PDF was obtained as
  extracted text (direct download failed on this network), so no binary
  SHA-256 of that PDF is recorded.
- Authority levels are kept distinct: curriculum expectations (CS2023),
  professional-BOK characterizations (SWEBOK) and lane-level synthesis are
  labeled as such on every record, and the one cross-source alignment reading
  is quarantined in a SYNTHESIS evidence (`se:EVD-07E-SYNTHESIS-CI-ALIGNMENT`)
  behind a SYNTHESIS-kind claim.
- Terminological variance is preserved, not averaged: SWEBOK KA6 and KA8 use
  "release" with different emphases; both senses are recorded side by side in
  `se:CON-07E-RELEASE` and asserted in
  `se:CLM-07E-SWEBOK-RELEASE-TERM-VARIANCE`.
- ISO sources remain metadata-only and back no content claim in this lane.
  NIST SP 800-218 build/pipeline-security practices were not extracted here
  (no claim is made about them); see Known gaps.

## Rights and access restrictions

The CS2023 report's permission notice grants use of the curriculum guidelines
for development of educational materials and programs; other use requires
specific permission (ACM / IEEE-CS / AAAI contacts in
`authority/source-reviews/CS2023.md`). This lane therefore stores only
bibliographic metadata, exact locators and concise original paraphrases; short
verbatim topic titles appear only where locator anchoring requires them. No
substantial report text is reproduced, no PDF is mirrored into the repository,
and nothing in this lane assumes commercial redistribution or
product-embedding rights. The unresolved boundary is tracked as the shared
CoverageGap `se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH / OPEN, carried verbatim
from the SEK-07 shared snippet pack for SEK-08 deduplication). SWEBOK v4.0a
content is handled per its registry rights record: direct content
reinspection, concise original paraphrase only, no redistribution inference.

## Record counts

| Collection | Count | Records |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` (verbatim shared snippet) |
| taxonomy_views | 2 | shared `se:VIEW-CS2023-SE` (verbatim) + `se:VIEW-07E-BUILD-DELIVERY-PROCESS` (PROCESS) |
| taxonomy_nodes | 3 | `se:NODE-07E-SE-TOOLS-KU` (curriculum unit) + SWEBOK KA6 / KA8 evidenced-scope nodes |
| source_locators | 10 | 1 shared verbatim (`se:LOC-CS2023-SE-KU-TABLE`) + `se:LOC-07E-CS2023-SE-TOOLS-KU` (heading-anchored) + 4 SWEBOK KA6 + 4 SWEBOK KA8 (PDF-index + printed-folio verified) |
| evidence_records | 12 | 3 CS2023 `DIRECT_SOURCE_PARAPHRASE` + 8 SWEBOK KA6/KA8 `DIRECT_SOURCE_PARAPHRASE` + 1 lane `SYNTHESIS` |
| concepts | 5 | continuous integration (MECHANISM), deployment (TERM), release two-senses (TERM), reproducible build (TERM), SBOM (TERM) |
| knowledge_claims | 16 | 3 DEFINITION, 1 TAXONOMY_STATEMENT, 9 PRACTICE_CLAIM, 1 NORMATIVE_CLAIM, 1 LIMITATION, 1 SYNTHESIS (CS2023 ×3; SWEBOK ×12; lane synthesis ×1) |
| principles | 1 | `se:PRI-07E-BUILD-FROM-CONTROLLED-SOURCE` |
| practices | 3 | CI pipeline, rehearsed rollback, environment automation |
| patterns | 1 | `se:PAT-07E-PIPELINE-GATED-DELIVERY` |
| anti_patterns | 1 | `se:ANT-07E-MANUAL-HANDOFF-DEPLOYMENT` (directly sourced contrast) |
| decision_models | 1 | `se:DEC-07E-RELEASE-STRATEGY` |
| coverage_gaps | 4 | SWEBOK breadth; real-case evidence; no empirics; shared CS2023 rights-clearance (HIGH/OPEN) |
| relationship_assertions | 39 | 25 SUPPORTED_BY, 4 APPLIES_TO, 2 DEPENDS_ON, 2 IMPLEMENTS, 1 QUALIFIED_BY, 1 RELATED_TO, 2 CROSS_CUTS, 2 DERIVED_FROM |

Shared records (domain, view, Core-Hours locator, rights gap) are copied
verbatim (parsed-identical) from the SEK-07 lane-shared snippet pack so SEK-08
can deduplicate them by canonical ID across the seven SEK-07 lanes. Per the
brief, only the shared CS2023 locators this lane actually uses are carried:
`se:LOC-CS2023-SE-KU-TABLE` backs the SE-Tools Core-Hours row evidence;
`se:LOC-CS2023-SE-OVERVIEW` and `se:LOC-CS2023-BOK-TABLE` are unused here and
therefore not carried. Source entities are **not** stored in this file; the
validation harness derives them from `authority/SOURCE_REGISTRY.yaml`.

## Taxonomy view

Two lenses are kept separate on purpose:

- `se:VIEW-CS2023-SE` (shared, CURRICULUM) — the curriculum classification;
  this lane adds one node, `se:NODE-07E-SE-TOOLS-KU`, into it.
- `se:VIEW-07E-BUILD-DELIVERY-PROCESS` (lane, PROCESS) — a lane-level lens
  over the inspected SWEBOK sections with two evidenced-scope KNOWLEDGE_AREA
  nodes (KA6 delivery-side sections; KA8 build/release sections).

The curriculum unit and the two SWEBOK nodes are related by explicit
`CROSS_CUTS` edges, not merged; the #299 §2.2 taxonomy is treated as a
hypothesis to challenge, with the lifecycle/process split recorded as views
rather than forced into one hierarchy.

## Known gaps

1. **SWEBOK breadth gap** (`se:GAP-07E-SWEBOK-BREADTH`, MEDIUM/OPEN): only
   KA6 2.1.1/2.2/3.2/3.3 and KA8 1.3–1.4/2.5–2.6/6/6.1/6.2 were inspected; the
   KAs' remaining sections may not be inferred and no SWEBOK-derived record
   has passed human review.
2. **Real-case gap** (`se:GAP-07E-REAL-CASE-EVIDENCE`, MEDIUM/OPEN): no
   case-class source is registered; pattern and anti-pattern records have no
   real-world illustration. The manual-handoff anti-pattern's inefficiency
   characterization is a professional-BOK statement, not a measured outcome.
3. **No empirics** (`se:GAP-07E-NO-EMPIRICS`, LOW/OPEN): no empirical
   evidence class backs CI/CD or release-practice effectiveness claims; this
   lane's value statements rest on SWEBOK characterization and CS2023
   curriculum expectations only.
4. **Secure-build guidance not extracted**: NIST SP 800-218/218A
   build/pipeline security practices are registered sources but were not
   extracted into this lane; no record claims anything about them. Admitting
   them belongs to a future lane or task with its own locator verification.
5. **CS2023 KU locator page limitation**:
   `se:LOC-07E-CS2023-SE-TOOLS-KU` is heading-anchored with the registry
   page anchor; the printed page of the KU subsection was not independently
   confirmed (PDF read as extracted text; no binary SHA-256 recorded). The
   SWEBOK locators do not share this limitation (hash-checked official PDF).
6. **Shared-record wording**: the shared domain snippet's second
   boundary note is truncated mid-sentence at "(SEK-02R" in the shared
   snippet pack itself; it was copied verbatim (parsed-identical) per the
   shared-record rule rather than locally repaired, to keep all SEK-07 lane
   copies identical for SEK-08 deduplication. Flagged for the snippet-pack
   owner, not fixed in this lane.

## Reproduction

Run from the worktree root (`VENV` is the SEK venv: Python 3.14.7, PyYAML
6.0.3, LinkML 1.12.1 / linkml-runtime 1.12.1, jsonschema 4.26.0, pypdf 7.0.0;
`export PYTHONDONTWRITEBYTECODE=1`):

```bash
VENV=/c/xDev/kAiCreator/_tmp/i321-repair/venv
export PYTHONDONTWRITEBYTECODE=1

# 1. Repository validator self-test (expect SELF_TESTS=66/66)
$VENV/Scripts/python.exe docs/versions/v2.1.1/knowledge/model/validate_knowledge_graph.py --self-test | tail -1

# 2. Lane assembly + counts + graph validation + no-ACCEPTED + LinkML instance validation
$VENV/Scripts/python.exe /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i338-zcode 07E \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/build-delivery/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07E
```

Expected key lines: every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS`, with `DERIVED_SOURCES=2` (`se:SRC-CS2023`,
`se:SRC-SWEBOK-GUIDE-4-0A`). The exact-tip receipt (selftest-tip.log,
lane-validate-tip.log, manifest-four-paths.sha256, identity.txt, receipt.ini)
lives in `_tmp/tally-sek07-lanes/validation/07E/`; the SWEBOK page extracts
used for locator verification (swebok-ka6-extract.txt, swebok-ka8-extract.txt)
are retained alongside as research artifacts.

## Review entry point

Reviewers should start at `authority/source-reviews/CS2023.md` and
`authority/source-reviews/SWEBOK-GUIDE-v4.0a.md`, then trace each claim
through its `SUPPORTED_BY` edge to the evidence record, its locator and the
derived source — checking paraphrase fidelity against the verified anchors,
the curriculum-scope qualifiers (CS2023 records) or inspected-section
qualifiers (SWEBOK records), and the rights boundary. Any single record can be
rejected or qualified without editing unrelated records. All records await
independent Exact-SHA review; none is ACCEPTED.
