# Explanations — SEK-07G Planning, Risk, Quality Management & Technical Decisions Lane

Narrative companion to `records.yaml`. Each entry explains **why** the
record was modeled the way it was; the structured YAML remains the only
authoritative store of the record's data. Field values are discussed,
not duplicated.

## Shared records (copied verbatim for SEK-08 deduplication)

- `se:DOMAIN-SOFTWARE-ENGINEERING` and `se:VIEW-CS2023-SE` — shared
  infrastructure from the lane snippet pack, byte-identical across the
  SEK-07 lanes so the integrator can collapse duplicates by canonical ID.
  One honest note: the snippet pack's second domain boundary note is a
  truncated fragment of the SEK-06 wording (it ends mid-citation). It was
  copied **verbatim** per the shared-record rule rather than silently
  repaired, because unilaterally rewording it would break cross-lane
  byte-identity; the artifact is flagged in the observation note and in
  the lane README for the coordinator to fix centrally.
- `se:LOC-CS2023-SE-KU-TABLE` — the registry page-verified Core Hours
  locator, carried because the lane actually uses it: the Teamwork row
  grounds the curriculum-weight evidence.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — the shared rights-clearance gap
  (HIGH/OPEN); its narrative entry lives in the Coverage gaps section.

## Taxonomy

- `se:VIEW-07G-SWEBOK-SEM-PLANNING` — the lane deliberately does **not**
  model SWEBOK's seven-topic breakdown as a universal hierarchy. It takes
  only the four topic groups this lane's scope owns (planning with its
  risk/quality subsections, and measurement), types them as a PROCESS
  view, and states in the view's own limitations that SWEBOK's breakdown
  is activity-based and recommends no SDLC — keeping lifecycle-model
  territory in lane 07A. The CS2023 curriculum lens stays a separate,
  shared view; the two are never merged, and their one real intersection
  (effort estimation taught inside Teamwork vs. estimated inside project
  planning) is recorded as an explicit `CROSS_CUTS` edge instead of a
  forced containment.
- `se:NODE-07G-SEM-PROJECT-PLANNING`, `se:NODE-07G-SEM-RISK-MANAGEMENT`,
  `se:NODE-07G-SEM-QUALITY-MANAGEMENT`,
  `se:NODE-07G-SEM-MEASUREMENT` — practice-group nodes scoped to
  directly inspected sections. The two `PART_OF` edges mirror the
  chapter's own structure (2.5 and 2.6 are subsections of section 2),
  which keeps the containment claims falsifiable against the cited pages
  rather than invented by the lane.
- `se:NODE-07G-CS2023-SE-TEAMWORK` — a knowledge-unit node in the shared
  curriculum view, defined from the unit's topic list rather than from an
  overview positioning sentence (the Teamwork unit has no such sentence;
  the heading-anchored topic list is the honest anchor available).

## Evidence records

- `se:EVD-07G-SWEBOK-SEM-INTRO` — one record spanning the KA9
  introduction because its three payloads (definition,
  measurement-informed management, working definitions) are one
  continuous argument the chapter makes; splitting them would fabricate
  three locators where the text has one. Its limitation explicitly
  declines to adopt the introduction's industry-perception rhetoric
  (late/over budget/poor quality) as an empirical finding.
- `se:EVD-07G-SWEBOK-KA9-ESTIMATION` and
  `se:EVD-07G-SWEBOK-KA15-ESTIMATION` — the two estimation evidences are
  kept separate even though they rhyme: KA9 §2.3 is the *management*
  estimation topic (multiple approaches, reconcile, negotiate) while
  KA15 §8 is the *economics* estimation topic (what an estimate is, the
  five techniques). Collapsing them would erase the very distinction the
  two-chapter reading exists to preserve.
- `se:EVD-07G-SWEBOK-KA9-RISK-QUALITY` — covers 2.5-2.7 in one record.
  The join is disclosed: the sections are contiguous pages, and the lane
  uses the plan-magnitude risk rule from 2.7 as context for the
  risk/quality pair. Its limitations carry the two authority caveats
  that matter most in this lane: PMBOK is characterized, not cited, and
  the quality-attribute enumeration is illustrative.
- `se:EVD-07G-SWEBOK-KA9-MEASUREMENT` — the 15939-following measurement
  process. The record paraphrases only what the Guide says about the
  process; the standard's normative status is repeatedly kept at arm's
  length (limitation + the source gap), because importing a standard by
  summary would be authority inflation.
- `se:EVD-07G-SWEBOK-KA15-VALUE` — grounds the governance-of-trade-offs
  scope: technical decisions carry business implications, value includes
  unquantifiables. Modeled as its own evidence so the value-based claim
  does not lean on the decision-process sections it does not cite.
- `se:EVD-07G-SWEBOK-KA15-DECISION-PROCESS` — the six-step process with
  the risk/uncertainty technique split. The record preserves the
  section's own caveat that the process is fluid in practice, so the
  pattern built on it cannot be read as a rigid waterfall of decisions.
- `se:EVD-07G-CS2023-TEAMWORK-PROCESSES` — deliberately includes the
  unit's other topics in the statement (communication, cooperative
  programming, diversity) while the lane admits only the
  planning/estimation/risk-bearing topics into claims; the uncertainty
  field records that boundary judgment so a reviewer can tighten or
  reject it.
- `se:EVD-07G-CS2023-TEAMWORK-HOURS` — one curriculum-bookkeeping row,
  earned for the same reason as its SEK-06 counterpart: it exercises the
  shared Core-Hours locator the lane must carry anyway and documents
  Teamwork's hour weight for future curriculum-comparison work.
- `se:EVD-07G-CS2023-REQ-EFFORT` — the elective estimation topic,
  including the "students are likely to be way off" caveat, which the
  lane treats as a curriculum-scope limitation statement, never as
  evidence about professional estimation accuracy.
- `se:EVD-07G-SYNTHESIS-POINT-ESTIMATE` — the lane's only non-paraphrase
  evidence. It exists so the anti-pattern's core move (reading KA15 §8's
  inherent-uncertainty + ethics-clause against KA9 §2.3's
  error-proneness) is inspectable and rejectable as a unit. Its anchor
  locator is the KA15 leg (the anti-pattern's center of gravity); the
  method field names the KA9 leg explicitly.

## Concepts

- `se:CON-07G-EFFORT-ESTIMATE` — a TERM, because KA15 §8 supplies a
  crisp working definition, not a mechanism; the concept keeps the
  decisions-not-certainty framing inside the definition so downstream
  records cannot quote "estimate" as if it meant "measurement of the
  future".
- `se:CON-07G-RISK-VS-UNCERTAINTY` — a BOUNDARY_CONCEPT, the cleanest
  use of that kind in the inspected text: one sentence that separates
  two things projects constantly conflate.
- `se:CON-07G-RISK-REGISTER` — kept narrow (a document/tool with a
  compliance use); the lane does not invent a register schema the source
  never gives.
- `se:CON-07G-INFORMATION-NEED` — scoped to SWEBOK's characterization of
  the 15939 concept, with the standard's own definition explicitly out
  of reach (source gap).

## Claims

Every claim's scope field names the exact chapter/section (SWEBOK) or
unit/topic (CS2023), and every SWEBOK claim repeats the
"generally-accepted-knowledge, not-prescription" limitation. Two
claim-kind choices worth explaining:

- The measurement-informed and value-based claims are `NORMATIVE_CLAIM`,
  not practice claims: both are the sources' own *positions* ("must",
  "should be treated as"), and typing them as practice claims would
  launder an opinion into a description of practice.
- The two estimation-difficulty/uncertainty claims are `LIMITATION`:
  each records what estimation *cannot do* (be certain) rather than what
  a team should do — the claim kind keeps the epistemic status visible.
- The distributed-team risk claim stays CS2023-scoped; the lane does not
  promote team-distribution risks into general engineering-risk claims,
  which would cross into generic project-management territory the scope
  forbids.

## Principles, practices, pattern, anti-pattern, decision model

- `se:PRI-07G-MEASUREMENT-INFORMED-MANAGEMENT` — the lane's SWEBOK-anchored
  principle. Its limitations state plainly that the Guide asserts rather
  than evidences it; the `IMPLEMENTS` edge from the needs-driven
  measurement practice shows what the principle looks like when
  operationalized, instead of leaving it as rhetoric.
- `se:PRI-07G-CLOSE-ESTIMATION-LOOP` — the lane's one deliberately
  cross-source principle: KA15 §2.7 says compare estimates to actuals;
  CS2023 teaches exactly that comparison to students. Both evidence
  records are cited, and a `QUALIFIED_BY` edge records that the CS2023
  leg is elective and pedagogical — support with an honest ceiling.
- `se:PRA-07G-MULTI-TECHNIQUE-ESTIMATION` — the one practice composed
  from two sections (KA9 §2.3 + KA15 §8.5). The composition is disclosed
  in the practice's scope and repeated as an `IMPLEMENTS` edge
  qualifier, so no one can attribute the joined procedure to either
  section alone.
- `se:PRA-07G-NEEDS-DRIVEN-MEASUREMENT` — operationalizes the planning
  half of the KA9 §6 process; its risks field carries the failure modes
  the section implies (needs drift, collection disruption) rather than
  inventing quantitative thresholds.
- `se:PRA-07G-RISK-REGISTER-MAINTENANCE` — the periodic-revisit rule is
  the practice's load-bearing element (the source's distinctive claim vs.
  generic risk-list advice), and the practice says so in its own risk
  list.
- `se:PAT-07G-MULTI-ATTRIBUTE-TRADEOFF` — a pattern, not a prescription:
  its trade-offs carry the compensatory/non-compensatory tension, and
  its context cites the decision examples the KA15 introduction itself
  lists (buy-or-build, risk-based testing sufficiency) rather than
  inventing scenarios.
- `se:ANT-07G-POINT-ESTIMATE-COMMITMENT` — the anti-pattern's class has
  no limitations slot, so the mandatory disclaimer (counterproductive
  characterization is lane synthesis, not a SWEBOK sentence) lives in
  the scope field, and the `DERIVED_FROM` edge points at the synthesis
  evidence so the inference chain is a first-class, rejectable edge.
- `se:DM-07G-ESTIMATION-TECHNIQUE-SELECTION` — options/criteria/trade-offs
  structured from KA15 §8.1-8.5 plus KA9 §2.3's reconciliation advice;
  the constraint that every estimate carry an uncertainty assessment is
  attributed to the ethics clause *as quoted by SWEBOK*, never as an
  independent reading of the Code (KA14 was not inspected).

## Coverage gaps

- `se:GAP-07G-NO-PM-STANDARD-SOURCE` — the lane's most consequential
  structural limit: SWEBOK's management chapters are standards-heavy,
  and none of the standards is registered. The gap records the exact
  affected claims and proposes the concrete sources (PMBOK 6th ed., SWX,
  ISO/IEC/IEEE 15939) for a separate single-writer registry task rather
  than importing them ad hoc.
- `se:GAP-07G-KA12-QUALITY-UNINSPECTED` — turns the lane's own boundary
  decision (quality management in scope, but KA12 not in the reading
  plan) into an auditable gap instead of silent under-coverage.
- `se:GAP-07G-NO-REAL-CASE-EVIDENCE` — why the lane has an evidence
  hole instead of invented war stories; the anti-pattern's consequences
  are explicitly inference, not observation.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — shared, verbatim, HIGH/OPEN; the
  rights question attaches to the CS2023 source as a whole, hence the
  domain-level scope.

## Relationship assertions

Thirty-two edges, chosen to make the predicate catalog do real work:
twenty-one `SUPPORTED_BY` keep the canonical claim/principle/pattern/
model/concept-to-evidence pattern (the cross-source principle gets two,
one per evidence); the `QUALIFIED_BY` edge exists so the elective status
of the CS2023 estimation topic is part of the principle's support
structure, not a footnote; two `APPLIES_TO` bind practices to the nodes
they operationalize; three `IMPLEMENTS` edges record
practice-implements-claim/principle with fidelity-delta qualifiers; the
`DEPENDS_ON` edge ties the decision model to the estimate concept it
presupposes; the `DERIVED_FROM` edge makes the anti-pattern's synthesis
provenance auditable; two `PART_OF` edges mirror the chapter's section
structure inside one view; and the single `CROSS_CUTS` edge is the
lane's structural answer to "do not force lifecycle, disciplines and
quality attributes into one hierarchy" — the curriculum and
professional-BOK taxonomies stay separate, with their intersection
typed, qualified and evidenced. Evidence-bearing edges cite evidence
records their endpoints already use; no evidence is invented for an edge
alone. No `CONTRADICTED_BY` edge exists because no conflict between the
inspected sources was observed — the SWEBOK/CS2023 estimation statements
are consistent in kind (both say estimation is hard and comparison is
the remedy), and preserving a conflict that was not found would be its
own kind of fabrication.
