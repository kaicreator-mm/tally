# Explanations — SEK-06A Requirements Engineering Lane

Narrative companion to `records.yaml`. Each entry explains **why** the record
was modeled the way it was; the structured YAML remains the only authoritative
store of the record's data. Field values are discussed, not duplicated.

## Shared records (copied verbatim for SEK-08 deduplication)

- `se:DOMAIN-SOFTWARE-ENGINEERING`, `se:VIEW-CS2023-SE`,
  `se:LOC-CS2023-SE-OVERVIEW`, `se:LOC-CS2023-SE-KU-TABLE`,
  `se:LOC-CS2023-BOK-TABLE` — these five records are byte-identical copies of
  the lane-shared snippet pack used by SEK-06A/06B/06C. They are shared
  infrastructure, not requirements-lane authorship: the domain keeps the
  professional-BOK exclusion explicit, the view keeps curriculum classification
  separate from professional taxonomies, and the three locators carry the only
  page anchors the registry has verified for the CS2023 SE section. Copying them
  unchanged lets the SEK-08 integrator collapse duplicates by canonical ID
  instead of reconciling near-duplicates by hand.

## Taxonomy

- `se:NODE-CS2023-SE-PRODUCT-REQUIREMENTS` — the lane needs exactly one
  curriculum node to anchor scope-bearing edges (the two `APPLIES_TO` assertions
  and one CoverageGap). The node is typed as a knowledge unit inside the CS2023
  SE view rather than as a discipline node, because CS2023 treats Product
  Requirements as one of nine SE units, not as a top-level area. Its definition
  deliberately cites the SE Overview's positioning sentence (via the overview
  evidence) rather than the topic list, keeping node identity about *what the
  unit is* and leaving *what it teaches* to the evidence records.
- `se:LOC-CS2023-SE-REQ-KU` — the knowledge-unit-level locator. The registry has
  page anchors only down to the SE section and Core Hours table; the KU
  subsection's printed page was never independently confirmed (the PDF was read
  as extracted text). So this locator anchors on the exact unit heading plus the
  topic numbering and *says so in the locator string itself*, including the
  registry page span it falls inside. Inventing a page number here would have
  been the one unforced verification error this lane could commit.

## Evidence records

- `se:EVD-REQ-SE-OVERVIEW-PRODUCT-REQUIREMENTS` — grounds the taxonomy node.
  Paraphrases the Overview's item 3, which is the curriculum's own
  one-sentence placement of the unit. Recorded separately from the topic-level
  evidence because it lives at a different locator (registry page anchors) and
  serves a different purpose (node identity, not topic content).
- `se:EVD-REQ-SE-PREAMBLE-MULTIVERSION` — the preamble's "multi-version
  programs" framing is the curriculum's reason requirements *change* at all.
  Pulled out as its own evidence because the change principle needs a grounding
  that the topic list alone cannot supply. Its limitation notes that a preamble
  motivates; it does not prescribe process.
- `se:EVD-REQ-CORE-HOURS` — one line of curriculum bookkeeping, but it earns its
  record: it uses the *shared* Core Hours locator, which would otherwise sit
  unused in the lane while still being required verbatim for deduplication. It
  also documents that Product Requirements' CS Core exposure is delivered
  jointly with SEP, a fact future curriculum-comparison work will want.
- `se:EVD-REQ-FUNCTIONAL-REQ-METHODS` — topic 1 with subtopics 1a-c in one
  evidence record. Kept together because the subtopics are meaningless without
  the topic sentence, and split would force three near-duplicate statements.
  This one record feeds a claim, a pattern, a decision model and one
  `RELATED_TO` edge — the densest hub in the lane.
- `se:EVD-REQ-REQUIREMENT-PROPERTIES` — topic 2 is a bare enumeration in the
  source; the evidence says so and resists the temptation to import definitions
  of the four properties from general knowledge, which would exceed the anchor.
- `se:EVD-REQ-ELICITATION-SOURCES-METHODS` — topic 3's two example lists (who
  you ask; how you ask). The illustrative-not-exhaustive caveat is recorded as a
  limitation because two downstream records (claim and practice) both lean on
  those lists.
- `se:EVD-REQ-QUALITY-ATTRIBUTES` — topic 4's terminology equivalence is the
  most reusable single fact in the lane: it licenses a Concept and a DEFINITION
  claim, and its cross-reference to SEP-Sustainability is kept to show the
  curriculum itself treats quality attributes as shared, not SE-proprietary.
- `se:EVD-REQ-RISK-ETHICS` — topic 5 exists in the lane mainly to keep the
  coverage of KA Core topics 1-6 honest; the ethical-considerations clause is
  preserved verbatim-adjacent because it is the part most likely to be dropped
  by a careless paraphrase.
- `se:EVD-REQ-PROTOTYPE-AND-CHANGE` — topics 7 and 8 share one record because
  the lane reads them as a pair: the prototype is the curriculum's example
  instrument for discovering requirements, and product evolution is the
  situation in which that discovery repeats. Both being Non-core is stated in
  the limitations and propagates into the qualifiers of two relationship edges.
- `se:EVD-REQ-SYNTHESIS-QA-DEFERRED` — the only non-paraphrase evidence. It
  exists so that the anti-pattern's core move (reading topic 4's core-ness
  against topic 1's design-support demand) is inspectable and rejectable as a
  *unit*, instead of being smuggled into an anti-pattern record as if the
  curriculum had said it. Its context line says in plain words that no such
  sentence appears in CS2023.

## Concepts

- `se:CON-REQ-REQUIREMENTS-ELICITATION` — modeled as a TERM (not a mechanism)
  because the curriculum supplies a characterization-by-examples, not a working
  definition of a mechanism; a TERM record can honestly carry that fidelity.
- `se:CON-REQ-QUALITY-ATTRIBUTE` — modeled with the QUALITY_ATTRIBUTE concept
  kind, and its definition states the CS2023 terminology equation explicitly,
  because the whole lane's usefulness for later cross-source work hinges on
  that alias being visible rather than buried in a claim.

## Claims

All five claims carry a scope field naming the CS2023 SE curriculum and the
specific topic number, per the SWEBOK-hold rule: nothing here is promoted to
professional consensus.

- `se:CLM-REQ-FUNCTIONAL-DOC-METHODS` (practice claim) — what the curriculum
  *asks students to do* with functional requirements, including the
  understand-support-and-weaknesses demand that later becomes decision-model
  criteria.
- `se:CLM-REQ-REQUIREMENT-PROPERTIES` (definition) — the four-property
  enumeration, scoped as curriculum terminology.
- `se:CLM-REQ-ELICITATION-MULTI-SOURCE` (practice claim) — the multi-source,
  multi-method reading of topic 3.
- `se:CLM-REQ-QA-TERMINOLOGY` (definition) — the non-functional-equals-quality-
  attribute equivalence, scoped so other frameworks' differing usage is not
  contradicted.
- `se:CLM-REQ-PROTOTYPE-DUAL-ROLE` (practice claim) — the dual instrumental
  role of prototypes, with the Non-core status inside the claim itself so the
  claim cannot be quoted without its elective context.

## Principle, practices, pattern, anti-pattern, decision model

- `se:PRI-REQ-EXPECT-CHANGE` — the lane's only principle, assembled from the
  preamble (why change is expected) plus topic 8 (what to do about it). Its
  limitations field carries the two honest caveats: evolution is an elective
  topic, and no empirical evidence in this lane tests the principle beyond
  curriculum scope. The `QUALIFIED_BY` edge from the prototype/change evidence
  exists precisely so a reviewer sees the elective-status qualification as part
  of the principle's support structure, not an afterthought.
- `se:PRA-REQ-MULTI-SOURCE-ELICITATION` — turns topic 3 into a runnable
  procedure. The step the curriculum does *not* contain — recording source and
  method provenance per requirement — is the practice's own addition; the
  `IMPLEMENTS` edge qualifier discloses exactly that delta so the curriculum
  claim and the practice cannot be conflated.
- `se:PRA-REQ-PROTOTYPE-VALIDATION` — operationalizes topic 7 with mock-up/MVP
  fidelity choice and stakeholder walkthrough, and states prototype-specific
  risks (throwaway misread as product direction) that the curriculum implies
  but does not enumerate.
- `se:PAT-REQ-USER-STORY-SPECIFICATION` — a pattern, not a prescription: the
  curriculum names use cases and user stories as interchangeable example
  methods, so the pattern's solution keeps "user story or comparable structured
  unit" and its trade-offs carry the lightweight-versus-detailed tension. The
  `RELATED_TO` edge to the quality-attribute concept is a boundary fence: the
  pattern structures functional requirements only, and the edge's qualifiers
  say a specification built this way still owes explicit quality attributes.
- `se:ANT-REQ-IMPLICIT-QUALITY-ATTRIBUTES` — the lane's anti-pattern. Its
  schema class has no limitations slot, so the mandatory disclaimer — the
  counterproductive characterization is lane synthesis, not a curriculum
  statement — lives in the scope field, and the `DERIVED_FROM` edge points at
  the synthesis evidence so the inference chain is a first-class, rejectable
  graph edge.
- `se:DM-REQ-FUNCTIONAL-DOC-METHOD` — a decision aid structured directly from
  topic 1: options are the curriculum's example methods, criteria embed the
  1b/1c axes (design/implementation support; strengths and weaknesses) plus
  team familiarity. Note the class has no label slot in the schema; the record
  intentionally omits one rather than fighting the model.

## Coverage gaps

- `se:GAP-REQ-PROFESSIONAL-BOK-EVIDENCE` — scoped to the taxonomy node and
  marked HIGH/OPEN: while SWEBOK is on hold, this lane's records must not be
  read as professional requirements-engineering knowledge. The resolution note
  names the unblocking event (#325) so the gap is actionable, not decorative.
- `se:GAP-REQ-REAL-CASE-EVIDENCE` — scoped to the view and marked MEDIUM/OPEN:
  no case-class source is registered, so no Case/Counterexample records exist
  and the pattern/anti-pattern records rest on curriculum text plus one marked
  synthesis. This gap is why the lane has an evidence-shaped hole rather than
  invented war stories.

## Relationship assertions

Fourteen edges, chosen to exercise the predicate catalog meaningfully rather
than to pad counts: seven `SUPPORTED_BY` (five claims plus the principle and
the decision model, keeping the canonical claim-to-evidence pattern of the
vertical slice — which is also why claim records themselves do not repeat an
evidence list); two `APPLIES_TO` (claim and practice each bound to the
curriculum node, both carrying evidence and qualifiers as the catalog
requires); one `QUALIFIED_BY` (principle ← prototype/change evidence, carrying
the elective-status qualifiers); one `DEPENDS_ON` (prototype-validation
practice needs the elicitation concept, since the curriculum's prototype serves
elicitation too); one `IMPLEMENTS` (elicitation practice operationalizes the
elicitation claim, with the fidelity-delta qualifier); one `DERIVED_FROM`
(anti-pattern ← synthesis evidence, making the inference auditable); one
`RELATED_TO` (pattern ↔ quality-attribute concept as the boundary fence
described above). Evidence-bearing edges cite the same evidence records their
endpoints use — no evidence is invented for an edge alone.
