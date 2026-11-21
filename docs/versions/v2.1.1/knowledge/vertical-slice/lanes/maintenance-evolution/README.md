# SEK-07F Lane — Maintenance, Evolution, Refactoring & Technical Debt

Vertical-slice knowledge lane for kaicreator-mm/tally issue #339 (SEK-07 lane 07F,
coordinator #333, parent pilot #299). This directory is independently reviewable; it is
not the integrated graph root. SEK-08 performs single-writer graph materialization/export.

- Lane record file: `records.yaml` (model/schema 0.3.0, LinkML 1.12.1)
- Narrative companion: `EXPLANATIONS.md` (links canonical IDs; no structured-fact copies)
- Builder observation: `docs/versions/v2.1.1/methods/observations/SEK-07F.md`
- Branch: `task/tally-v211-sek07f-maintenance-evolution` (base `7a70211b4460e4c753c97798670de9f34861d4a4`,
  the exact `wip/v2.1.1-knowledge` tip at lane start; not stacked on any other lane)

## Scope

Software maintenance types and processes, software evolution, change impact, refactoring,
modernization, technical-debt identification/management and long-term system evolution —
with evidence-backed principles kept separate from context-dependent heuristics.

Explicit boundaries (kept, not absorbed):

- Construction-time code practices stay in 07B (Software Construction).
- CI/CD delivery pipelines and mechanics stay in 07E; SWEBOK KA7 section 4.4
  (Continuous Integration, Delivery, Testing and Deployment) was inspected only as
  context and is deliberately not a record source here.
- Lifecycle/process models stay in 07A; planning/estimation governance stays in 07G
  (maintenance cost estimation detail beyond the inspected debt guidance is not recorded).
- KA8 SCM content is cited only where it supports maintenance/change control
  (sections 3, 3.1, 3.1.1, 3.2); the rest of SCM belongs to other lanes.

## Authority fit and sources

All locators were directly inspected on 2026-11-07 (see the observation file for method):

| Source | Registry ID | Verification state | Use in this lane |
|---|---|---|---|
| SWEBOK Guide v4.0a, KA7 Software Maintenance | `se:SRC-SWEBOK-GUIDE-4-0A` | IDENTITY_VERIFIED source; LOCATOR_VERIFIED lane locators | Sections Intro/1.1, 1.5, 1.6, 2.1.3, 2.1.4, 2.2.1, 2.3.1, 4.2, 4.3 — definitions, evolution laws, categories, impact analysis, maintainability/debt, org priorities, debt measurement, reengineering/refactoring, reverse engineering |
| SWEBOK Guide v4.0a, KA8 Software Configuration Management | `se:SRC-SWEBOK-GUIDE-4-0A` | same | Sections 3-3.2 change control only (SCR process, CCB, implementation/audits) |
| CS2023, SE knowledge unit "SE-Refactoring: Refactoring and Code Evolution" | `se:SRC-CS2023` | PARTIAL_CONTENT_VERIFIED source; heading-anchored lane locator + shared Core-Hours locator | KA Core topics 1-4, Non-core topic 5, learning outcomes; Core-Hours row (2 KA Core, 0 CS Core) |

Authority separation is preserved: curriculum teaching expectations (CS2023) are never
recorded as professional-consensus claims, SWEBOK content is recorded as BOK-described
generally accepted knowledge, the Lehman evolution laws are recorded as SWEBOK's secondhand
report of empirical research, and ISO/IEC/IEEE 14764 and 12207 are metadata-only — every
ISO attribution in this lane is "as reported by SWEBOK", never ingested ISO text.

## Taxonomy view

Two lenses are kept separate and linked by typed edges (no merged hierarchy):

- `se:VIEW-07F-MAINTENANCE-EVOLUTION` (LIFECYCLE, lane-local): nodes
  `se:NODE-07F-SOFTWARE-MAINTENANCE`, `se:NODE-07F-CHANGE-IMPACT-AND-CONTROL`,
  `se:NODE-07F-REFACTORING-AND-TECH-DEBT`.
- `se:VIEW-CS2023-SE` (CURRICULUM, shared): node `se:NODE-07F-CS2023-SE-REFACTORING`
  for the SE-Refactoring unit.
- Cross-view link: `se:REL-07F-030` (RELATED_TO with boundary-keeping qualifiers).

## Record counts (in `records.yaml`)

| Collection | Count |
|---|---|
| domains (shared) | 1 |
| taxonomy_views (1 shared + 1 lane) | 2 |
| taxonomy_nodes | 4 |
| source_locators (1 shared + 11 lane) | 12 |
| evidence_records (12 direct paraphrase + 1 marked synthesis) | 13 |
| concepts | 5 |
| knowledge_claims | 9 |
| principles | 2 |
| practices | 3 |
| patterns | 1 |
| anti_patterns | 1 |
| decision_models | 1 |
| coverage_gaps (1 shared + 2 lane) | 3 |
| relationship_assertions | 33 |

All knowledge records are PROPOSED or PENDING_REVIEW; locators are ACTIVE; nothing is
ACCEPTED and no ReviewDecision records exist. Source entities are not stored in the lane
file; the validation harness derives them from `SOURCE_REGISTRY.yaml`.

## Gaps and limitations

- `se:GAP-07F-KA7-CONTENT-BREADTH` (SOURCE_GAP, MEDIUM, OPEN): only the listed KA7/KA8
  sections were inspected; the rest (maintenance processes detail, measurement, staffing,
  organizational aspects, program comprehension, CI/CD, tools) is not covered.
- `se:GAP-07F-NO-CASE-EVIDENCE` (SOURCE_GAP, MEDIUM, OPEN): no case-class source is
  registered; the lane has no Case/Counterexample records, so pattern/anti-pattern
  consequences rest on BOK text plus one marked synthesis.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` (REVIEW_GAP, HIGH, OPEN, shared): CS2023 permission
  scope covers educational materials/programs; no broader reuse clearance is asserted.

## Edition, access and rights limits

- SWEBOK v4.0a: official PDF (SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
  411 pages) re-verified 2026-11-07. Concise original paraphrase only; no PDF or
  substantial text committed; no redistribution permission inferred.
- CS2023: full text obtained via extraction service on 2026-11-07 (binary download
  unavailable from this network, so no binary SHA-256 is recorded); heading-anchored
  locators only, verbatim use limited to short topic titles needed for anchoring.
  Rights gap above stays open.
- ISO/IEC/IEEE 14764 and 12207: metadata-only; never summarized or quoted.

## Reproduction steps

1. Worktree at the lane tip (branch `task/tally-v211-sek07f-maintenance-evolution`).
2. Preflight: `git diff --check`; YAML parse of `records.yaml` (Python 3.14 / PyYAML).
3. Self-test: `python docs/versions/v2.1.1/knowledge/model/validate_knowledge_graph.py --self-test`
   (must end `SELF_TESTS=66/66`).
4. Lane validation:
   `python <_tmp>/tally-sek07-lanes/scripts/assemble_and_validate.py <worktree> 07F docs/versions/v2.1.1/knowledge/vertical-slice/lanes/maintenance-evolution/records.yaml <outdir>`
   — every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
   `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS`.
5. Locator re-verification (optional): the pinned SWEBOK PDF at the verified page
   indices named in each `se:LOC-07F-*` locator; the CS2023 unit heading at the
   registry-recorded SE section span.
