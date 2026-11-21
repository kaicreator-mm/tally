# Foundations, Processes & Quality Models Lane — SEK-07A

**Lane ID:** `SEK-07A-FOUNDATIONS-PROCESSES-LANE-0.1.0`
**Status:** DRAFT / PENDING HUMAN REVIEW
**Task:** [#334](https://github.com/kaicreator-mm/tally/issues/334) (SEK-07 vertical-slice lanes, coordinator #333, parent pilot #299)
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (tip of `wip/v2.1.1-knowledge`)
**Integration target:** `wip/v2.1.1-knowledge` (SEK-08 assembles the lanes by canonical ID)

## Purpose and scope boundary

This lane is the foundations / engineering-processes / lifecycle / quality-models
depth slice of the SEK-07 knowledge expansion. It models directly reinspected
sections of SWEBOK Guide v4.0a:

- **KA10 Software Engineering Process** (sections 1-3): process fundamentals,
  life cycle definitions and the four process categories, life cycle models
  and paradigms (waterfall / V / incremental / spiral / Agile; predictive /
  iterative / evolutionary / incremental / continuous), life cycle adaptation
  with documented rationale, and process assessment (PDCA, GQM, CMM/CMMI /
  ISO 33000, retrospectives).
- **KA11 Software Engineering Models and Methods** (introduction, sections 1
  and 3, at model level): models and methods imposing structure, the three
  modeling principles, model properties and model analysis techniques.
- **KA12 Software Quality** (introduction, sections 1.1-1.4, at model level):
  the overloaded quality concept, the quality-versus-quality-requirements
  boundary, cost-of-quality categories and the dependability composition.
- **KA18 Engineering Foundations** (sections 1-3): the iterative engineering
  process, design as a wicked problem, abstraction levels and alternate
  abstractions.

Scope boundary:

- **Three separate taxonomy views, not one tree.** The process/lifecycle lens
  (`se:VIEW-07A-SWEBOK-PROCESS-LIFECYCLE`), the quality lens
  (`se:VIEW-07A-SWEBOK-QUALITY-MODELS`) and the foundations lens
  (`se:VIEW-07A-SWEBOK-ENGINEERING-FOUNDATIONS`) stay distinct and are related
  by an explicit `CROSS_CUTS` edge plus scope-bearing `APPLIES_TO` edges —
  matching the Task Pack requirement not to collapse lifecycle, disciplines
  and quality attributes into one hierarchy.
- **No process model is treated as universally mandatory.** The lane's
  principle `se:PRI-07A-TAILOR-DONT-ADOPT` and its counterpart anti-pattern
  `se:ANT-07A-UNIVERSAL-LIFECYCLE-MANDATE` are grounded in KA10's explicit
  no-ideal-process statement.
- **Out-of-scope neighbors stay out.** Engineering planning/estimation/risk
  governance (07G), implementation/code practices (07B), CI/CD delivery
  mechanics (07E) and reliability/performance engineering detail (07D) are not
  absorbed; KA12 content is admitted at the model level only.
- All knowledge records are `PROPOSED` or `PENDING_REVIEW`; no record is
  `ACCEPTED`. Evidence records are `EXTRACTED_PENDING_REVIEW`,
  `synthetic: false` (one evidence record is lane-marked `SYNTHESIS`).
  Knowledge claims are `review_status: PENDING`.

## Authority fit and limits

- SWEBOK Guide v4.0a (`se:SRC-SWEBOK-GUIDE-4-0A`) is the lane's single
  content source, used within its registry authority: a guide to generally
  accepted knowledge, not a uniform prescription, with non-comprehensive topic
  citations. Every locator was verified on 2026-11-07 against the official PDF
  binary (SHA-256
  `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
  411 pages, hash re-verified against the registry record); locators record
  exact 0-based PDF page indices and printed chapter folios (e.g., 10-1).
- **ISO sources are metadata-only.** ISO/IEC/IEEE 12207:2026, ISO/IEC/IEEE
  15288:2023 and ISO/IEC 25010:2023 are `ACCESS_RESTRICTED` in the registry;
  no normative clause, process list or quality characteristic is ingested,
  summarized or cited. Lifecycle-model content therefore comes from SWEBOK
  alone; the limitation is a first-class gap
  (`se:GAP-07A-ISO-NORMATIVE-LIFECYCLE`).
- **CS2023 is not cited.** The registry-verified CS2023 SE knowledge-unit
  list contains no unit for engineering processes, lifecycle models or
  quality models, so this lane carries no CS2023 content records and no
  shared CS2023 view/locators; the curriculum-side absence is recorded as
  `se:GAP-07A-CS2023-CURRICULUM-SCOPE`.
- One evidence record is a lane-marked `SYNTHESIS`
  (`se:EVD-07A-SYNTHESIS-UNIVERSAL-LIFECYCLE`): it combines three verified
  KA10 statements to motivate the anti-pattern; it is explicitly not a
  SWEBOK sentence, and the anti-pattern derived from it repeats that boundary
  in its scope field.

## Rights and access restrictions

SWEBOK v4.0a is used per the registry rights handling: direct content
reinspection with concise ORIGINAL paraphrase only. The PDF is not committed
or mirrored; no substantial source text is reproduced; no redistribution
permission is inferred. The shared CS2023 rights-clearance gap
`se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH / OPEN) is carried verbatim from the
SEK-07 shared snippet pack and stays OPEN until a documented rights
determination exists, even though this lane cites no CS2023 content — the
gap attaches to the domain-level source boundary shared by all lanes.

## Record counts

| Collection | Count | Records |
|---|---|---|
| domains | 1 | shared `se:DOMAIN-SOFTWARE-ENGINEERING` (verbatim shared snippet) |
| taxonomy_views | 3 | process/lifecycle (PROCESS), quality-model (QUALITY), engineering-foundations (DISCIPLINE) |
| taxonomy_nodes | 6 | KA10 node, lifecycle-models subdomain, assessment practice group, KA12 node, quality-requirements node, KA18 node |
| source_locators | 11 | 5 SWEBOK KA10 + 2 KA11 + 2 KA12 + 2 KA18, all SECTION-scoped with PDF indices + printed folios |
| evidence_records | 12 | 11 `DIRECT_SOURCE_PARAPHRASE` + 1 lane-marked `SYNTHESIS` |
| concepts | 6 | life cycle (TERM), SLCM (TERM), software quality (TERM), quality requirement (QUALITY_ATTRIBUTE), abstraction levels (ABSTRACTION), wicked problem (TERM) |
| knowledge_claims | 11 | 4 DEFINITION, 2 TAXONOMY_STATEMENT, 1 DOMAIN_BOUNDARY, 1 LIMITATION, 3 PRACTICE_CLAIM |
| principles | 2 | tailor-don't-adopt; keep-views-distinct |
| practices | 3 | life cycle tailoring; GQM-style assessment; multi-view modeling |
| patterns | 1 | incremental delivery within an iterative life cycle |
| anti_patterns | 1 | universal life cycle mandate |
| decision_models | 1 | life cycle model selection and tailoring |
| coverage_gaps | 5 | ISO normative content; CS2023 curriculum absence; real-case evidence; foundations breadth; shared CS2023 rights clearance (HIGH/OPEN) |
| relationship_assertions | 25 | 13 SUPPORTED_BY, 4 APPLIES_TO, 2 IMPLEMENTS, 2 DEPENDS_ON, 1 DERIVED_FROM, 1 CROSS_CUTS, 1 RELATED_TO, 1 QUALIFIED_BY |

Shared records (domain + rights-clearance gap) are copied verbatim from the
SEK-07 lane-shared snippet pack so SEK-08 integration can deduplicate them by
canonical ID across the seven SEK-07 lanes. Source entities are **not** stored
in this file; the validation harness derives them from
`authority/SOURCE_REGISTRY.yaml` (`DERIVED_SOURCES=1`,
`se:SRC-SWEBOK-GUIDE-4-0A`).

## Known gaps

1. **ISO normative content (MEDIUM/OPEN).** 12207/15288/25010 are
   metadata-only; SWEBOK's attribution of process categories to
   ISO/IEC/IEEE 12207:2017 is reported as the Guide's citation, never as
   verified normative content (`se:GAP-07A-ISO-NORMATIVE-LIFECYCLE`).
2. **No case evidence (MEDIUM/OPEN).** No case-class source is registered;
   no Case/Counterexample records exist, and the pattern/anti-pattern
   consequences rest on SWEBOK text plus one lane-marked synthesis
   (`se:GAP-07A-NO-CASE-EVIDENCE`).
3. **Curriculum-side absence (LOW/OPEN).** CS2023 has no process/lifecycle/
   quality-model unit, so no curriculum comparison is possible from this lane
   (`se:GAP-07A-CS2023-CURRICULUM-SCOPE`).
4. **Foundations breadth (LOW/OPEN).** Only KA18 sections 1-3 are admitted
   as locators (sections 4-5 were read but not admitted); KA16/KA17 are
   uninspected (`se:GAP-07A-FOUNDATIONS-BREADTH`).
5. **Shared snippet fidelity note.** The shared snippet pack's domain record
   carries a boundary note that ends mid-sentence ("...(SEK-02R"); the lane
   copies it verbatim (parsed-identical to the pack, verified in preflight)
   because rewording shared records is prohibited; SEK-08 deduplication
   expects byte-identical shared records across lanes.

## Reproduction

Run from the worktree root (`VENV` is the SEK venv with Python 3.14.7 and
LinkML 1.12.1; always `export PYTHONDONTWRITEBYTECODE=1`):

```bash
VENV=/c/xDev/kAiCreator/_tmp/i321-repair/venv

# 1. Repository validator self-test (expect SELF_TESTS=66/66)
cd docs/versions/v2.1.1/knowledge/model && $VENV/Scripts/python.exe validate_knowledge_graph.py --self-test | tail -1

# 2. Lane assembly + graph validation + count minimums + no-ACCEPTED check + LinkML instance validation
$VENV/Scripts/python.exe /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py \
  /c/xDev/kAiCreator/tally/tally-i334-zcode 07A \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/foundations-processes/records.yaml \
  /c/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07A
```

Expected key lines: every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS` with `DERIVED_SOURCES=1` (88 lane records, 25
relationship assertions). The exact-tip receipt lives in
`_tmp/tally-sek07-lanes/validation/07A/`.

## Review entry point

Reviewers should start at `authority/SOURCE_REGISTRY.yaml` (SWEBOK entry),
then trace each claim through its `SUPPORTED_BY` edge to the evidence record,
its `se:LOC-07A-*` locator (exact PDF page indices and printed folios) and
the derived source `se:SRC-SWEBOK-GUIDE-4-0A`, checking paraphrase fidelity
against the located sections, the professional-BOK scope qualifiers, the
synthesis disclosure on the anti-pattern chain, and the ISO/CS2023 rights and
access boundaries. Any single record can be rejected or qualified without
editing unrelated records.
