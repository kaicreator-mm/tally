# SEK-06B Explanations — Architecture and Design Lane

Narrative companions for the lane's canonical records in
[`records.yaml`](records.yaml). These explanations are not a second data store: identity,
typed fields and provenance live only in the structured records, and nothing here overrides
them. Every record below remains PENDING_REVIEW until a human ReviewDecision accepts an
exact revision.

## Shared records (used, not owned, by this lane)

- `se:DOMAIN-SOFTWARE-ENGINEERING` and `se:VIEW-CS2023-SE` are copied verbatim from the
  SEK-06 shared snippet set so that SEK-08 integration can deduplicate them across lanes.
  The domain record's boundary notes already encode the two constraints this lane
  depends on: no single source is a complete ontology, and SWEBOK is not citable until
  #325 merges.
- The three shared locators anchor the registry-verified page positions. This lane uses
  `se:LOC-CS2023-SE-OVERVIEW` for the Software Design unit framing and
  `se:LOC-CS2023-SE-KU-TABLE` for the core-hours fact; `se:LOC-CS2023-BOK-TABLE` is
  carried for integration completeness and is not cited by lane evidence.

## Taxonomy node — `se:NODE-CS2023-SE-SOFTWARE-DESIGN`

The node exists so that APPLIES_TO edges from lane claims and practices land on a
curriculum entity rather than on the whole domain. Its definition leans on the SE
Overview's contrast — Product Requirements for user-facing functionality, Software Design
for engineer-facing internal design — because that contrast is what keeps this lane from
swallowing requirements topics. The node deliberately claims nothing about professional
software design writ large.

## Evidence family — `se:EVD-ARC-0001` … `se:EVD-ARC-0011`

The evidence set walks the SE-Design knowledge unit top to bottom. Three records carry the
weight of the lane's central distinction: `se:EVD-ARC-0001` (architectural vs detailed
design as abstraction levels, CS Core 1a), `se:EVD-ARC-0009` (the Overview's
architecture-to-API span) and `se:EVD-ARC-0008` (programming in the large vs small, and
code smells distinct from correctness). Others ground the principle topics
(`se:EVD-ARC-0002`), the paradigm and standard-architecture enumerations
(`se:EVD-ARC-0003`, `se:EVD-ARC-0004`), the API design principles block
(`se:EVD-ARC-0005`), requirement traceability (`se:EVD-ARC-0006`), and the Non-core
secure-design and tradeoff topics (`se:EVD-ARC-0007`). `se:EVD-ARC-0011` records the
core-hours allocation as a curriculum-weight fact.

Every statement is a paraphrase of heading-anchored topic text or of registry-verified
table content. Where the curriculum only names a topic (coupling and cohesion, the secure
design principles), the evidence says so rather than filling the silence.

`se:EVD-ARC-0010` is the single SYNTHESIS record. It inverts the curriculum's stated
benefits of API consistency into the formulation of an anti-pattern. The record says
out loud what that means: CS2023 names no anti-pattern, no case evidence exists, and the
anti-pattern's observable attributes are interpretations. A reviewer who finds that
inference too aggressive can reject the anti-pattern without touching the paraphrase
evidence.

## Claims — `se:CLM-ARC-0001` … `se:CLM-ARC-0006`

The claims are deliberately small. `se:CLM-ARC-0001` asserts only what CS Core 1a
supports: two abstraction levels presented as distinct. It explicitly disclaims the
normative architecture-description layer, which has no admissible source.
`se:CLM-ARC-0002` and `se:CLM-ARC-0003` are enumerations, with the source's own
"e.g." hedge retained as a limitation on `se:CLM-ARC-0003`. `se:CLM-ARC-0004` and
`se:CLM-ARC-0005` are practice claims about what the curriculum teaches — not claims that
the practices are industry-consensus or effective. `se:CLM-ARC-0006` is a curriculum
design fact (core hours) that exists to support curriculum analysis, never importance
rankings.

## Concepts — `se:CON-ARC-SEPARATION-OF-CONCERNS`, `se:CON-ARC-COUPLING-COHESION`, `se:CON-ARC-DESIGN-ABSTRACTION-LEVELS`

Separation of concerns is stored as an abstraction the curriculum teaches and expects
students to apply to flawed designs. Coupling and cohesion is stored as a PROPERTY whose
curriculum anchor is name-level; the definition flags that the property gloss is a
conventional reading rather than extracted text, so a reviewer can hold the two layers
apart. Design abstraction levels is a BOUNDARY_CONCEPT: its job is to keep
architecture-level and detailed-design statements at different abstraction levels and to
point at the ISO 42010 gap for everything the corpus cannot say.

## Principle — `se:PRI-ARC-SEPARATION-OF-CONCERNS`

The principle record turns the curriculum's topic-plus-learning-outcome pair into a
guiding proposition with a stated rationale. Its exceptions and limitations do the heavy
lifting: the curriculum itself separates tradeoff evaluation from any single principle
(Non-core 13), the anchor is name-level, and professional-consensus status is unavailable
under the SWEBOK hold. It is guidance with a curriculum passport, not a prescription.

## Practices — `se:PRA-ARC-BOUNDARY-IDENTIFICATION`, `se:PRA-ARC-API-CONTRACT-DOCUMENTATION`

Both practices are operationalizations of named curriculum competencies: component
boundary and dependency identification (CS Core 2c plus learning outcome 4) and API
contract documentation (KA Core 5 plus learning outcome 5). The preconditions, steps,
outcomes and risks are lane-authored scaffolding around those anchors; the risks are
phrased as negations of the curriculum's stated expectations (e.g., silently absorbing
contract-violating input contradicts topic 5e.ii) so that they stay traceable.

## Pattern — `se:PAT-ARC-LAYERED-ARCHITECTURE`

The layered architecture pattern splices two adjacent anchors — layered design as a
paradigm (CS Core 2a.i) and n-layer as a standard architecture (CS Core 2b). Its
consequences and tradeoffs fields are intentionally sparse and say why: the curriculum
enumerates without elaborating, and the lane declines to import textbook consequences the
source does not state. What the record adds is structure (problem/context/forces/
solution), clearly labeled as assembled from curriculum topic names.

## Anti-pattern — `se:ANT-ARC-INCONSISTENT-API-SURFACE`

The anti-pattern is the downstream face of the synthesis evidence: symptoms and causes
describe what an inconsistent API surface looks like, the consequence is exactly the
forfeiture of the curriculum's stated consistency benefits, and remediation points back
to topic 5a. The DERIVED_FROM edge to `se:EVD-ARC-0010` makes the inferential lineage
auditable; a reviewer can sever the whole construct in one decision.

## Decision model — `se:DM-ARC-DESIGN-TRADEOFF-EVALUATION`

The decision model structures judgment the curriculum assigns to Non-core topic 13: when
efficiency opposes reliability or security opposes usability, name the goals, compare
options against the named criteria, record the chosen tradeoff — and, borrowing KA Core
topic 9, record which requirements the chosen design still satisfies. The three options
are lane scaffolding, and the constraints field says so; the example pairs are capped to
the curriculum's own examples.

## Coverage gaps — `se:GAP-ARC-ISO42010-ARCH-DESCRIPTION`, `se:GAP-ARC-SWEBOK-DESIGN-CONSENSUS`, `se:GAP-ARC-DESIGN-REAL-CASES`

The three gaps are the lane's honest edges. The first keeps the architecture-description
layer explicitly unsourced rather than quietly borrowing the ISO standard's title. The
second prevents any curriculum statement here from being read as professional consensus
while SWEBOK is held. The third records why no Case or Counterexample records exist: with
no admitted real-case source, inventing one would violate the corpus's provenance rules,
so Pattern and AntiPattern records stand on curriculum and declared-synthesis evidence
alone.

## Relationship assertions — `se:REL-ARC-0001` … `se:REL-ARC-0013`

The edge set follows the required mix. Six SUPPORTED_BY edges bind claims to evidence.
Two APPLIES_TO edges scope a claim and a practice to the SE-Design node with explicit
qualifiers. The QUALIFIED_BY edge lets the tradeoff evidence temper the
separation-of-concerns principle. DEPENDS_ON and IMPLEMENTS express the structural
reading that layered design and boundary identification lean on the
separation-of-concerns concept and principle respectively — both marked as lane-level
interpretation in their rationales. DERIVED_FROM ties the anti-pattern to its synthesis
evidence, and RELATED_TO links the API-contract and traceability claims with the
interpretive character of that link stated in its qualifier.
