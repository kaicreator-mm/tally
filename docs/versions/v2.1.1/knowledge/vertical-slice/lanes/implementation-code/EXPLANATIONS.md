# Explanations — SEK-07B Implementation, Code Engineering & Maintainability Lane

Narrative companion to `records.yaml`. Each entry explains **why** the record was
modeled the way it was; the structured YAML remains the only authoritative store
of the record's data. Field values are discussed, not duplicated.

## Shared records (copied verbatim for SEK-08 deduplication)

- `se:DOMAIN-SOFTWARE-ENGINEERING`, `se:VIEW-CS2023-SE`,
  `se:LOC-CS2023-SE-KU-TABLE` — copied byte-identically from the SEK-07
  lane-shared snippet pack so the SEK-08 integrator can collapse duplicates by
  canonical ID across the seven SEK-07 lanes. Copying discipline note: the
  snippet pack's domain record carries a second boundary note that ends
  mid-sentence ("…complete (SEK-02R"); the copy-verbatim rule forbids rewording,
  so the lane copied it as-is and reports the truncation to the coordinator in
  `methods/observations/SEK-07B.md` instead of silently repairing it. Only the
  three snippets this lane actually uses were carried; the SE Overview and
  front-matter BOK-table locators are not cited here and were therefore not
  copied.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — the fourth shared record, copied verbatim.
  Being a CoverageGap, its narrative entry lives in the Coverage gaps section
  below; it is listed here so the shared-record inventory stays complete.

## Taxonomy

- `se:SUB-07B-SOFTWARE-CONSTRUCTION` — the lane needs a professional-lens anchor
  for applicability edges that are *not* curriculum-scoped. Modeled as a
  Subdomain (not a TaxonomyNode) so it can never be mistaken for a CS2023
  classification, and bounded by a `BOUNDED_BY` edge to the single evidence
  record (the KA4 Introduction) that characterizes it. Its boundary note keeps
  SWEBOK's own caveat that design/construction/testing boundaries are life-cycle
  dependent — the subdomain is an anchor, not a process decomposition.
- `se:VIEW-07B-CS2023-SDF` + `se:NODE-07B-CS2023-SDF-PRACTICES` — CS2023 places
  SDF-Practices in the Software Development Fundamentals area, so anchoring it
  inside the shared SE view would have misfiled the unit. A narrow lane-local
  view scoped to exactly that one unit (coverage_state PARTIAL, limitation
  saying so) was the honest alternative to either misfiling or dropping the
  unit, whose basic-practice topics (readability-focused style, natural-language
  module specification) are squarely in this lane's scope.
- `se:NODE-07B-CS2023-SE-CONSTRUCTION` — the curriculum-side hub. Its evidence
  is deliberately the *shared* Core Hours table locator: anchoring node identity
  on the registry-verified hour row (rather than the heading-anchored KU
  locator) gives the node the strongest available page anchor, while the
  heading-anchored locator carries the topic-level content separately.

## Evidence records

- `se:EVD-07B-SWEBOK-CONSTRUCTION-DEFINITION` — the KA definition including its
  "detailed creation **and maintenance**" wording and the life-cycle-dependent
  boundary statement. Pulled out separately from the fundamentals because it is
  the only inspected text that licenses the subdomain node's scope and the
  cross-lens `RELATED_TO` edge's qualifier.
- `se:EVD-07B-SWEBOK-MINIMIZE-COMPLEXITY` — the densest fundamentals record:
  working-memory rationale, everywhere-and-testing scope, cyclomatic complexity
  with its test-case corollary, and the simple-over-clever rule. One record
  because these sentences form a single argument in section 1.1; it feeds the
  principle, a concept and two edges.
- `se:EVD-07B-SWEBOK-CONSTRUCT-FOR-VERIFICATION` — section 1.3's four technique
  families only. Kept lean because the practice record operationalizes exactly
  these four families; anything beyond them in the practice is flagged there as
  lane-added procedure.
- `se:EVD-07B-SWEBOK-CODING-CONSIDERATIONS` — section 3.3's consideration list.
  Recorded as an evidence record without a dependent claim: it exists so the
  locator's verified-content claim is inspectable and so future lanes (or review
  repairs) can hang claims off an already-verified anchor.
- `se:EVD-07B-SWEBOK-QUALITY-TECHNIQUES` — the primary-technique enumeration.
  The code-adjacent-focus sentence is preserved because it is what keeps this
  lane's quality records from bleeding into process-level quality management
  (07G territory).
- `se:EVD-07B-SWEBOK-INTEGRATION` — the phased/incremental/CI contrast in one
  record, hedge included. The limitation notes that CI delivery mechanics
  belong to 07E: this lane cites CI only as a construction-side integration
  option.
- `se:EVD-07B-SWEBOK-API-DESIGN` — signature-plus-semantics definition and the
  six design goals. Scoped as *code-level* API design; architectural service
  boundaries stay with SEK-06B.
- `se:EVD-07B-SWEBOK-DBC-ERROR` — merges 4.4 and 4.5 into one record because the
  assertion/contract/defensive triple and the exception-policy principles form
  one argument about executable interface expectations. The Guide's own hedge
  on design by contract ("is thought to improve") is preserved verbatim-adjacent
  and propagates to the concept and pattern.
- `se:EVD-07B-CS2023-CONSTRUCTION-KU` — the whole-unit paraphrase with the TDD
  psychological rationale kept, because that rationale (not TDD mechanics) is
  what this lane's CS2023 test-first claim rests on. The limitation fences
  secure-coding content off to 07C.
- `se:EVD-07B-CS2023-CORE-HOURS` — uses the *shared* table locator (see
  taxonomy above) and records the SDF joint delivery of both CS Core and KA
  Core hours, the curriculum fact that motivates this lane reading SDF-Practices
  at all.
- `se:EVD-07B-CS2023-SDF-PRACTICES` — basic-practice topics at curriculum
  fidelity; deliberately scoped as early-exposure curriculum, not professional
  practice.
- `se:EVD-07B-CS2023-SE-TOOLS-ANALYSIS` — only the analysis-tooling topic and
  its integration outcome; the rest of the SE-Tools unit (build systems, IDEs,
  DevOps tooling) belongs to 07E's scope.
- `se:EVD-07B-CS2023-CODE-SMELLS` — one topic line from the SEK-06 verified
  anchors pack. Its locator string says so explicitly (extraction 2026-11-06,
  re-read 2026-11-07) so provenance of the anchor is visible in-band.
- `se:EVD-07B-SYNTHESIS-BIG-BANG` — the only non-paraphrase evidence,
  `PROPOSED` per the synthesis rule. It exists so the anti-pattern's core move
  (inverting SWEBOK's advantage list onto the phased-integration description)
  is inspectable and rejectable as a unit, instead of being smuggled into the
  anti-pattern as if the Guide had said it.

## Concepts

- `se:CON-07B-CYCLOMATIC-COMPLEXITY` — a METRIC concept because the source
  supplies a measure with an attribution (McCabe 1976) and a corollary (at
  least that many test cases), which is exactly metric-level fidelity.
- `se:CON-07B-DESIGN-BY-CONTRACT` — a MECHANISM concept; the hedged quality
  claim stays in the definition text so no consumer can quote the concept
  without the hedge.
- `se:CON-07B-DEFENSIVE-PROGRAMMING` — defined at SWEBOK fidelity with the
  CS2023 topic placement recorded as corroboration, not as a second definition
  (one canonical definition per concept).
- `se:CON-07B-CODE-SMELL` — a TERM concept at curriculum-category fidelity: the
  source names the category and its distinctness from correctness but does not
  define or enumerate smells; the concept resists importing a smell catalogue
  from general knowledge, which would exceed the anchor.

## Claims

Five SWEBOK claims carry scope fields naming the exact reinspected KA4
subsections and repeat the Guide's authority limits; four CS2023 claims name the
knowledge unit and topic numbers. Nothing curriculum-derived is promoted to
professional consensus, and nothing guide-derived is promoted to an empirical
finding.

- `se:CLM-07B-SWEBOK-CONSTRUCTION-DEFINITION` (definition) — includes the
  life-cycle-dependent boundary sentence, which is the claim's most load-bearing
  caveat for later lane coordination.
- `se:CLM-07B-SWEBOK-MINIMIZE-COMPLEXITY` (practice claim) — the
  working-memory-grounded goal statement, kept as a claim about what the Guide
  presents rather than a free-standing imperative.
- `se:CLM-07B-SWEBOK-QUALITY-TECHNIQUES` (practice claim) — the enumeration
  with its selection dependency; deliberately not a ranking.
- `se:CLM-07B-SWEBOK-INTEGRATION-APPROACHES` (practice claim) — carries the
  "thought to offer" hedge inside the claim text so it cannot be quoted without
  it.
- `se:CLM-07B-SWEBOK-API-DESIGN-GOALS` (normative claim) — the only
  NORMATIVE_CLAIM in the lane, because section 4.1 states design goals in a
  should-form; the scope field keeps it at guide-guidance authority, below
  standard requirements.
- `se:CLM-07B-CS2023-CODING-STYLE-BEST-PRACTICES`, `se:CLM-07B-CS2023-TEST-FIRST-PSYCHOLOGICAL`,
  `se:CLM-07B-CS2023-TOOLS-ANALYSIS-INTEGRATION` (practice claims) — curriculum
  teaching expectations, including the explicitly pedagogical TDD rationale.
- `se:CLM-07B-CS2023-CODE-SMELLS-BOUNDARY` (taxonomy statement) — records
  *where* CS2023 classifies code-quality indications, not what any smell is.

## Principle, practices, patterns, anti-pattern, decision model

- `se:PRI-07B-MINIMIZE-COMPLEXITY` — the lane's only principle, restating
  section 1.1's argument chain (working memory → minimize complexity → simple
  readable code over clever code). Its exceptions list is empty *because the
  Guide states none in the inspected text*; the limitations field says so
  explicitly, so no reviewer mistakes an empty list for an exhaustive one.
- `se:PRA-07B-CONSTRUCT-FOR-VERIFICATION` — operationalizes 1.3's four
  technique families; the preconditions/steps beyond the four families are the
  practice's own addition and the `APPLIES_TO` edge qualifier discloses that
  delta.
- `se:PRA-07B-INTERFACE-DOCUMENTATION` — turns the CS2023 documentation topic
  into a procedure, keeping the curriculum's own emphases (contracts optional
  "potentially including"; non-obvious-only comments; refactor-first) rather
  than upgrading them to a documentation standard.
- `se:PRA-07B-STATIC-ANALYSIS-INTEGRATION` — bridges the CS2023 tool-integration
  expectation with SWEBOK 3.6's static-analysis placement; the `IMPLEMENTS` edge
  binds it to the curriculum claim while the evidence list carries both
  sources.
- `se:PAT-07B-INCREMENTAL-INTEGRATION` — a pattern whose consequences *are*
  SWEBOK's advantage list, hedged; the separate `QUALIFIED_BY` edge
  (`se:REL-07B-016`) exists so the hedge is a first-class part of the support
  structure, not a footnote.
- `se:PAT-07B-DESIGN-BY-CONTRACT` — routine-level pattern with the
  compiled-in/compiled-out assertion lifecycle from 4.4; the production-runs-
  unguarded tradeoff is drawn directly from that lifecycle.
- `se:ANT-07B-BIG-BANG-INTEGRATION` — the lane's anti-pattern. Its class has no
  limitations slot, so the mandatory disclaimer — the characterization is lane
  synthesis inverting SWEBOK's comparison, not a Guide sentence — lives in the
  scope field, and the `DERIVED_FROM` edge points at the synthesis evidence so
  the inference is a rejectable graph edge.
- `se:DM-07B-INTEGRATION-STRATEGY` — options drawn from both sources (SWEBOK's
  phased/incremental/CI plus CS2023 outcome 7's top-down/bottom-up/sandwich),
  with two `SUPPORTED_BY` edges, one per source, so each anchor is separately
  auditable. Like SEK-06's decision model, it omits a label because the class
  has none in the schema.

## Coverage gaps

- `se:GAP-07B-SWEBOK-KA4-BREADTH` — scoped to the subdomain node: the lane
  admits records only from seven inspected KA4 locations; everything else in
  the chapter was read for navigation but is not admitted, and nothing may be
  inferred from it.
- `se:GAP-07B-REAL-CASE-EVIDENCE` — scoped to the curriculum node: no
  case-class source exists, which is why the lane has an evidence-shaped hole
  where war stories would go, rather than invented ones.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — HIGH/OPEN shared gap; scoped to the domain
  because the permission question attaches to the CS2023 source as a whole.

## Relationship assertions

Twenty-six edges, chosen to exercise the predicate catalog meaningfully: fifteen
`SUPPORTED_BY` (nine claims, the principle, the incremental-integration pattern,
the decision model twice — one edge per source anchor — and both taxonomy
nodes), four `APPLIES_TO` (principle and construct-for-verification practice to
the subdomain; interface-documentation practice to the curriculum node;
code-smell concept to the subdomain), one `IMPLEMENTS` (analysis-integration
practice operationalizes the curriculum claim, with the fidelity-delta
qualifier), one `DEPENDS_ON` (design-by-contract pattern needs its concept),
one `DERIVED_FROM` (anti-pattern ← synthesis evidence), two `RELATED_TO`
(curriculum node ↔ professional subdomain as a cross-lens mapping;
design-by-contract pattern ↔ interface-documentation practice as a
curriculum/BOK bridge), one `BOUNDED_BY` (subdomain ← the KA4 Introduction
evidence that characterizes it), one `QUALIFIED_BY` (incremental-integration
pattern ← its own evidence, carrying the "thought to offer" hedge).
Evidence-bearing edges cite the same evidence records their endpoints use — no
evidence is invented for an edge alone.
