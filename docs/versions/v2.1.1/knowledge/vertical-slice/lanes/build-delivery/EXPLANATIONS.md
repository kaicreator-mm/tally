# Explanations — SEK-07E Integration, Build, CI/CD & Delivery Lane

Narrative companion to `records.yaml`. Each entry explains **why** the record
was modeled the way it was; the structured YAML remains the only authoritative
store of the record's data. Field values are discussed, not duplicated.

## Shared records (copied verbatim for SEK-08 deduplication)

- `se:DOMAIN-SOFTWARE-ENGINEERING`, `se:VIEW-CS2023-SE` and
  `se:LOC-CS2023-SE-KU-TABLE` — parsed-identical copies of the SEK-07 shared
  snippet pack (`_tmp/tally-sek07-lanes/shared-snippets-sek07.yaml`). They are
  shared infrastructure, not build-delivery authorship: the domain keeps the
  professional-BOK exclusion explicit, the view keeps curriculum
  classification separate from professional taxonomies, and the Core Hours
  locator is the one shared CS2023 anchor this lane actually uses (the
  SE-Tools row evidence). The two other shared CS2023 locators
  (`se:LOC-CS2023-SE-OVERVIEW`, `se:LOC-CS2023-BOK-TABLE`) are unused here and
  were therefore not carried, per the lane brief's "only what you use" rule —
  SEK-08 still sees them from the lanes that use them.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — the shared rights condition, carried
  verbatim; its narrative entry lives in the Coverage gaps section below.
- One deliberate non-repair: the shared domain snippet's second boundary note
  ends mid-sentence at "(SEK-02R" in the pack itself. The shared-record rule
  (no rewording) outranks the itch to complete the sentence, because seven
  lanes copying identical bytes is what lets SEK-08 collapse them by canonical
  ID. The truncation is flagged in the lane README for the pack owner.

## Taxonomy

- `se:VIEW-07E-BUILD-DELIVERY-PROCESS` — the lane needs a place to hang the
  two SWEBOK-side nodes without smuggling professional-BOK material into the
  shared curriculum view. Typing it PROCESS (rather than LIFECYCLE) keeps the
  emphasis on the connected build–verify–release–deploy region the inspected
  sections actually describe, and its limitations state outright that it is a
  lane-level organizing lens, not a universal process hierarchy — the direct
  answer to the #299 §2.2 "challenge the taxonomy hypothesis" instruction.
- `se:NODE-07E-SE-TOOLS-KU` — one curriculum node inside the shared CS2023 SE
  view, because CS2023 treats Tools and Environments as one of nine SE units.
  The node's definition confines itself to the unit's build/CI/delivery-relevant
  topics; the unit's IDE/design-tool topics are noted as out of this lane's
  modeled scope by omission (the locator records the full unit for reviewers).
- `se:NODE-07E-SWEBOK-KA6-OPERATIONS` and `se:NODE-07E-SWEBOK-KA8-SCM` —
  evidenced-scope nodes: each definition lists exactly which sections of the
  KA were inspected and states that the KA's remaining sections are outside
  the node's scope. This is the structural enforcement of the SWEBOK breadth
  gap: the node cannot be read as coverage of the whole KA.
- The two `CROSS_CUTS` edges from the curriculum unit to the two SWEBOK nodes
  are the lane's way of saying "same ground, different lenses" without merging
  nodes across views. The catalog offers OVERLAPS for that too, but OVERLAPS
  is semantically symmetric scope-sharing between classification entries;
  what this lane observed is a curriculum unit intersecting professional-BOK
  topic regions, which reads more naturally as crossing a view boundary.

## Evidence records

- The eight SWEBOK evidence records are one-per-inspected-section (or section
  pair) so that every claim traces to a page-anchored passage a reviewer can
  re-open. `se:EVD-07E-SWEBOK-KA6-DEPLOY-RELEASE` is deliberately the longest:
  KA6 3.2 packs the deployment/release definitions, the manual-handoff
  contrast, the strategy families and the canary technique into one printed
  page, and splitting it would have forced three records to share one locator
  without gaining reviewability.
- `se:EVD-07E-CS2023-TOOLS-KU` and `se:EVD-07E-CS2023-TOOLS-OUTCOMES` — the
  unit's topic list and its illustrative learning outcomes are kept separate
  because they are different kinds of curriculum statements (what is taught
  vs. what students should be able to do). The ground-truth outcome is the
  load-bearing one for this lane: it is the curriculum's own epistemic claim
  about CI, and it anchors `se:CLM-07E-CI-GROUND-TRUTH` and the synthesis.
- `se:EVD-07E-CS2023-TOOLS-CORE-HOURS` — the one-line row record earns its
  place the same way SEK-06A's Core-Hours record did: it is the only lane use
  of the shared Core Hours locator, and it documents the joint SDF hour that
  curriculum-comparison work will want later.
- `se:EVD-07E-SYNTHESIS-CI-ALIGNMENT` — the only non-paraphrase evidence. The
  alignment of the CS2023 ground-truth outcome with the SWEBOK pipeline
  description is a lane-level reading over two verified records; quarantining
  it in a labeled SYNTHESIS evidence behind a SYNTHESIS-kind claim keeps the
  cross-source inference inspectable and rejectable without contaminating
  either direct-paraphrase chain.

## Concepts

- `se:CON-07E-CONTINUOUS-INTEGRATION` is dual-anchored (SWEBOK KA8 6.1
  mechanism + CS2023 topic 4b characterization) because the two sources
  describe the same mechanism at different authority levels; the concept's
  scope field says exactly that rather than pretending one canonical
  definition.
- `se:CON-07E-RELEASE` records two SWEBOK senses side by side instead of
  averaging them. KA6's sense (making features available to customers, a
  business decision) and KA8's sense (distributing artifacts outside the
  development activity, including internal releases) are each correct in their
  chapter; merging them would erase exactly the kind of boundary information
  this knowledge graph exists to preserve. The variance claim
  (`se:CLM-07E-SWEBOK-RELEASE-TERM-VARIANCE`) asserts the observation, and
  `se:CON-07E-DEPLOYMENT` keeps KA6's deployment term distinct from both.
- `se:CON-07E-REPRODUCIBLE-BUILD` and `se:CON-07E-SBOM` are the two
  build-output integrity concepts the inspected sections actually define. The
  `RELATED_TO` edge between them records a lane-level structural pairing (what
  was built vs. re-creating what was built) whose linkage is not stated by
  either section — the edge's qualifiers say so.

## Principle, practices, pattern, anti-pattern, decision model

- `se:PRI-07E-BUILD-FROM-CONTROLLED-SOURCE` — the lane's one principle fuses
  the two inspected halves of the same idea (KA6's single-source environments;
  KA8's correct-versions-plus-configuration-data builds with reproducibility).
  The `QUALIFIED_BY` edge preserves the asymmetry inside its own evidence:
  KA6 states environment coherence as a must but the single-repository
  automation as a DevOps recommendation, so the principle's environment half
  is intentionally softer than its build half.
- The three practices are the actionable projections of the inspected
  sections: `se:PRA-07E-CI-PIPELINE` (KA8 6.1 pipeline + CS2023 depth bound),
  `se:PRA-07E-REHEARSED-ROLLBACK` (KA6 3.3 + 2.1.1's plan-the-rollback task),
  `se:PRA-07E-ENV-AUTOMATION` (KA6 2.2). Each `IMPLEMENTS` edge names which
  claim or principle element the practice operationalizes, and each practice's
  scope field marks its BOD-fidelity boundary so the added procedural detail
  (record-keeping steps, risk phrasing) cannot be mistaken for source text.
- `se:PAT-07E-PIPELINE-GATED-DELIVERY` — neither source names this pattern;
  the scope field says the packaging is lane-level structure over the KA8
  6.1/6.2 pipeline descriptions plus the CS2023 build/CI topics. The pattern
  exists because the two SWEBOK subsections describe the same pipeline from
  its build end and its release end, and the pattern is where the two ends
  meet. It `DEPENDS_ON` the CI concept to make that presupposition explicit.
- `se:ANT-07E-MANUAL-HANDOFF-DEPLOYMENT` — unusually for an anti-pattern
  record, the counterproductive characterization is directly sourced: KA6 3.2
  itself calls the manual dev-to-ops hand-off inefficient in time and quality.
  The `DERIVED_FROM` edge still discloses that the symptoms/causes/remediation
  packaging is lane-level arrangement of that one inspected contrast.
- `se:DEC-07E-RELEASE-STRATEGY` — structured entirely from KA6's strategy
  families (3.2), rehearsed-rollback requirement (3.3) and release-planning
  tasks (2.1.1). Its constraints deliberately encode the section's separation
  of concerns: rehearse the rollback *before* deployment, but release (making
  features available) is a *business* decision *after* it — the decision model
  exists to keep engineers from collapsing those two decisions.

## Coverage gaps

- `se:GAP-07E-SWEBOK-BREADTH` — the honest frontier of the SWEBOK side: which
  KA6/KA8 sections were inspected is a positive list; everything else "may
  not be inferred" is the operative rule. The two evidenced-scope taxonomy
  nodes are its structural counterpart.
- `se:GAP-07E-REAL-CASE-EVIDENCE` — no case-class source exists in the
  permitted registry, so the lane contains zero Case/Counterexample records by
  construction, and the gap names the evidence class a future task should
  register (CI/CD adoption case studies, release-engineering postmortems).
- `se:GAP-07E-NO-EMPIRICS` — recorded at LOW severity because nothing in the
  lane claims empirical effectiveness; the gap exists so nobody later cites
  these records *as if* they carried measured outcomes.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — the shared HIGH/OPEN condition carried
  verbatim; it is the reason every CS2023-derived record in this lane stays
  paraphrase-only and none of them may be treated as cleared for
  redistribution or product embedding.

## Relationship assertions

The 39 edges break into four families: canonical `SUPPORTED_BY` traces (every
claim, plus the principle, pattern, practices and decision model trace to
their primary evidence); applicability edges (`APPLIES_TO` into the taxonomy
nodes, always with authority-scope qualifiers); structural edges
(`DEPENDS_ON`, `IMPLEMENTS`, `RELATED_TO`, `CROSS_CUTS`) that carry evidence
per the catalog's requirement; and provenance edges (`DERIVED_FROM`) for the
two records whose packaging is lane-level rather than source-level (the
anti-pattern and the synthesis claim). Multi-evidence records
(`se:CON-07E-RELEASE`, `se:PRI-07E-BUILD-FROM-CONTROLLED-SOURCE`,
`se:PRA-07E-CI-PIPELINE`) take one `SUPPORTED_BY` edge per evidence rather
than a merged list, so a reviewer can reject one leg without touching the
other.
