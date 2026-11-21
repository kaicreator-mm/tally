# Explanations — SEK-07A Foundations, Processes & Quality Models Lane

Narrative companion to `records.yaml`. Each entry explains **why** the record
was modeled the way it was; the structured YAML remains the only authoritative
store of the record's data. Field values are discussed, not duplicated.

## Shared records (copied verbatim for SEK-08 deduplication)

- `se:DOMAIN-SOFTWARE-ENGINEERING` and `se:GAP-CS2023-RIGHTS-CLEARANCE` —
  byte-level copies of the SEK-07 lane-shared snippet pack, carried so the
  SEK-08 integrator can collapse duplicates by canonical ID across the seven
  lanes. The rights gap is listed in the Coverage gaps section below but is
  shared infrastructure, not 07A authorship. One fidelity note: the pack's
  domain boundary note ends mid-sentence ("...(SEK-02R"); the lane copies it
  unchanged because rewording shared records is prohibited, and the copy was
  verified parsed-identical to the pack in preflight. This lane cites no
  CS2023 content, so the shared CS2023 view and locators are deliberately
  **not** carried.

## Taxonomy views and nodes

- Three views instead of one. The Task Pack explicitly warns against
  collapsing lifecycle, disciplines and quality attributes into one tree, and
  the inspected sources reward the split: KA10 organizes process and life
  cycle, KA12 discusses quality as a concept that recurs across many
  knowledge areas with product and process senses, and KA11 states models and
  methods span phases "regardless of the type of life cycle process model".
  `se:VIEW-07A-SWEBOK-PROCESS-LIFECYCLE` (PROCESS),
  `se:VIEW-07A-SWEBOK-QUALITY-MODELS` (QUALITY) and
  `se:VIEW-07A-SWEBOK-ENGINEERING-FOUNDATIONS` (DISCIPLINE) each carry their
  own purpose, coverage state and limitations.
- `se:NODE-07A-QUALITY-REQUIREMENTS` CROSS_CUTS `se:NODE-07A-KA10-SWE-PROCESS`
  is the lane's main "related but not merged" edge: KA12 ties quality
  planning and assurance to the life cycle, so the quality lens intersects
  the process lens — but the edge is typed CROSS_CUTS, never containment, and
  its qualifiers say so.
- The six nodes were chosen to give scope-bearing `APPLIES_TO` edges real
  endpoints (KA-level nodes, a lifecycle-models subdomain, an assessment
  practice group, a quality-requirements node) rather than pointing
  everything at one root.

## Source locators

- All eleven `se:LOC-07A-*` locators are SECTION-scoped to named SWEBOK
  chapters and subsections, with both 0-based PDF page indices and printed
  chapter folios (the Guide prints pages as `10-1`, `11-2`, etc.). The
  chapter-start page map came from the coordinator's mechanical folio scan,
  but every locator's content was verified by extracting and reading the
  actual pages with pypdf on 2026-11-07 — the scan only decided where to
  start reading. No SWEBOK front-matter table is cited (SEK-03 territory).
- The KA10 locator set splits section 2 into three locators (definition/
  categories; models/paradigms; management/adaptation incl. 2.9) because the
  claims they support are separable: what a life cycle is, which models
  exist, and how a project tailors one. One locator spanning all of section 2
  would have made each evidence record's anchor needlessly broad.

## Evidence records

- `se:EVD-07A-KA10-PROCESS-DEFINITION` anchors the lane's most load-bearing
  single sentence — no ideal process exists — together with the
  process-definition family and the empirical-measurement requirement. Three
  downstream records (a LIMITATION claim, the tailor principle, the decision
  model) rest on it, so it was kept separate from the life cycle material.
- `se:EVD-07A-KA10-LIFECYCLE-STRUCTURE` records the life cycle definition and
  the four process categories. Its limitation field discloses that the
  category enumeration follows the Guide's citation of ISO/IEC/IEEE
  12207:2017 — reported as the Guide's citation only, because the normative
  text is not admitted (that is exactly what
  `se:GAP-07A-ISO-NORMATIVE-LIFECYCLE` exists to say).
- `se:EVD-07A-KA10-LIFECYCLE-MODELS` is the model-vs-life-cycle distinction
  plus the paradigm definitions and the named Agile misconceptions. The
  misconceptions matter downstream: they become causes in the anti-pattern.
- `se:EVD-07A-KA10-STAGES-ADAPTATION` deliberately covers sections 2.6-2.9 in
  one record because the six non-sequential stages, the adaptation activity
  and the estimation/measurement lessons form one argument chain: stages are
  a framework, adaptation fits the framework to the project, and measurement
  keeps the fit honest.
- `se:EVD-07A-KA10-ASSESSMENT` covers section 3's four approach families
  under their shared empirical-evidence framing — that framing (not any one
  method) is what the claim and practice reuse.
- `se:EVD-07A-KA11-MODELING-PRINCIPLES` and `se:EVD-07A-KA11-MODEL-ANALYSIS`
  are the lane's KA11 slice, kept at model level: how models impose
  structure and how they are analyzed. The KA's method families (heuristic,
  formal, prototyping, Agile) are outside this lane's needs and are not
  summarized.
- `se:EVD-07A-KA12-QUALITY-OVERLOAD` is the widest record in the lane because
  KA12's introduction argues as one unit: the overloaded term, the surveyed
  definition family, the quality-versus-quality-requirements boundary,
  process-quality characteristics and the life-cycle-linked quality planning.
  Splitting it would have produced four near-duplicate statements about the
  same two pages.
- `se:EVD-07A-KA12-DEPENDABILITY` covers the dependability composition and
  integrity levels only; reliability engineering detail beyond the
  model-level composition is lane 07D's scope and is excluded here.
- `se:EVD-07A-KA18-ENG-PROCESS` and `se:EVD-07A-KA18-DESIGN-ABSTRACTION`
  carry the foundations slice: the iterative six-step engineering process
  (including "monitor the performance of the selected solution — estimates
  can be wrong", which quietly rhymes with KA10's measurement emphasis), and
  the design-as-wicked-problem plus alternate-abstractions material that
  grounds the multi-view modeling practice.
- `se:EVD-07A-SYNTHESIS-UNIVERSAL-LIFECYCLE` is the only non-paraphrase
  evidence. It exists so the anti-pattern's core move (reading the
  no-ideal-process statement against the model-family survey and the
  measurement lessons) is inspectable and rejectable as a unit; its context
  line states in plain words that no such sentence appears in SWEBOK.

## Concepts

- `se:CON-07A-SOFTWARE-LIFE-CYCLE` and `se:CON-07A-SLCM` are kept as two
  TERMs because the lane's central terminological finding is precisely their
  separation (a life cycle is created *in conformance with* a model).
  Collapsing them into one concept would erase the distinction the decision
  model depends on.
- `se:CON-07A-SOFTWARE-QUALITY` is a TERM, not a single definition, because
  the inspected source's finding is that the term is overloaded; the concept
  records the multi-sense character. `se:CON-07A-QUALITY-REQUIREMENT` uses
  the QUALITY_ATTRIBUTE concept kind to make the "-ilities" boundary visible
  next to it — the QUALIFIED_BY edge on the quality concept keeps the
  definition-family caveat machine-visible.
- `se:CON-07A-ABSTRACTION-LEVELS` (ABSTRACTION) and
  `se:CON-07A-WICKED-PROBLEM` (TERM) are the foundations pair: one grounds
  multi-view modeling, the other explains why lifecycle/model choice resists
  one-shot optimization — you must solve once to define, solve again to
  deliver.

## Claims

Eleven claims, each scoped to the exact inspected sections (chapter, section
numbers, printed folios) and each carrying a SUPPORTED_BY edge. Kinds were
chosen to match epistemic shape: LIMITATION for the no-ideal-process
statement (it bounds what any source may prescribe), DEFINITION for
terminology findings (model vs life cycle; non-sequential stages; quality
overload; models-impose-structure), TAXONOMY_STATEMENT for the four-category
and dependability compositions, DOMAIN_BOUNDARY for the
quality-versus-quality-requirements split, and PRACTICE_CLAIM for
adaptation/assessment/foundations-process characterizations. None is worded
as a free-standing professional imperative; they are claims about what the
Guide's inspected sections present.

## Principles, practices, pattern, anti-pattern, decision model

- `se:PRI-07A-TAILOR-DONT-ADOPT` assembles the no-ideal-process statement
  with the adaptation activity; its exception field honestly notes that
  normative contexts may bound tailoring — recorded as an exception because
  the lane cannot admit the normative text itself.
- `se:PRI-07A-KEEP-VIEWS-DISTINCT` is the lane's structuring principle and is
  openly marked as lane-level synthesis in its scope field; it governs the
  three-view design rather than claiming source authority.
- `se:PRA-07A-LIFECYCLE-ADAPTATION` adds one step beyond the source (the
  measurement-support step, drawn from KA10 2.9) — the IMPLEMENTS edge
  qualifier discloses that delta. `se:PRA-07A-GQM-ASSESSMENT` stays strictly
  inside section 3's goals-change-evaluate loop.
  `se:PRA-07A-MULTI-VIEW-MODELING` is the cross-KA practice (KA11 principles
  + KA18 alternate abstractions); its risks field carries both sources'
  cautions (single-model illusion; synchronization difficulty).
- `se:PAT-07A-INCREMENTAL-DELIVERY` is deliberately framed as one paradigm
  among several, not the winning one — its scope field says so, and the
  RELATED_TO edge to the SLCM concept keeps the model-choice question open
  (that question belongs to the decision model).
- `se:ANT-07A-UNIVERSAL-LIFECYCLE-MANDATE` has no limitations slot in the
  schema, so its mandatory disclaimer — the counterproductive
  characterization is lane synthesis — lives in the scope field, and the
  DERIVED_FROM edge points at the synthesis evidence so the inference chain
  is a first-class, rejectable graph edge.
- `se:DM-07A-LIFECYCLE-MODEL-SELECTION` turns the inspected material into a
  decision aid: options are the Guide's model families, criteria are the
  paradigm characterizations (requirements stability, risk-drivenness,
  increment deliverability) plus context factors, constraints encode the
  no-ideal-process and documented-rationale requirements. The class has no
  label slot in the schema; the record intentionally omits one.

## Coverage gaps

- `se:GAP-07A-ISO-NORMATIVE-LIFECYCLE` — scoped to the process/lifecycle
  view: the normative standards underpinning lifecycle terminology are
  ACCESS_RESTRICTED, so nothing here claims ISO content.
- `se:GAP-07A-CS2023-CURRICULUM-SCOPE` — scoped the same way: the
  registry-verified CS2023 unit list has no process/lifecycle/quality-model
  unit, so the lane skips CS2023 entirely rather than stretching a unit
  beyond its verified scope.
- `se:GAP-07A-NO-CASE-EVIDENCE` — scoped to the KA10 node: no case-class
  source exists, hence no Case/Counterexample records; this gap is why the
  lane has an evidence-shaped hole instead of invented war stories.
- `se:GAP-07A-FOUNDATIONS-BREADTH` — scoped to the foundations view: KA18
  sections 4-5 were read but not admitted as locators, and KA16/KA17 were not
  inspected at all.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — the shared HIGH/OPEN rights gap, carried
  verbatim although this lane cites no CS2023 content, because the permission
  question attaches to the domain-level source boundary shared by all lanes.

## Relationship assertions

Twenty-five edges: thirteen SUPPORTED_BY (one per knowledge claim — the
canonical trace the vertical slice requires — plus the two principles), four
APPLIES_TO (claims and the tailoring practice bound to knowledge-area,
subdomain, practice-group and quality nodes, each with evidence and
qualifiers), two IMPLEMENTS (tailoring practice → tailor principle;
multi-view practice → keep-views-distinct principle, both with fidelity-delta
qualifiers), two DEPENDS_ON (decision model → SLCM concept; multi-view
practice → abstraction-levels concept), one DERIVED_FROM (anti-pattern ←
synthesis evidence), one CROSS_CUTS (quality requirements → KA10 process
node, the explicit not-containment relation between the quality and process
lenses), one RELATED_TO (incremental-delivery pattern ↔ SLCM concept as the
boundary fence that keeps model choice open), one QUALIFIED_BY (software
quality concept ← the overload evidence). Evidence-bearing edges cite
existing evidence records only; no evidence is invented for an edge alone.
