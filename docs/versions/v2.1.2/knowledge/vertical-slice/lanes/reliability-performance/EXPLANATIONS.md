# Explanations — SEK-07D Reliability, Resilience & Performance Lane

Narrative companion to `records.yaml`. Each entry explains **why** the record
was modeled the way it was; the structured YAML remains the only authoritative
store of the record's data. Field values are discussed, not duplicated.

## Shared records (copied verbatim for SEK-08 deduplication)

- `se:DOMAIN-SOFTWARE-ENGINEERING`, `se:VIEW-CS2023-SE`,
  `se:LOC-CS2023-SE-KU-TABLE` and `se:GAP-CS2023-RIGHTS-CLEARANCE` — copied
  unchanged from the SEK-07 lane-shared snippet pack (itself derived from the
  reviewed SEK-06A final tip) so the SEK-08 integrator can collapse
  duplicates by canonical ID instead of reconciling near-duplicates. The
  curriculum view's own limitation — professional-BOK alignment lives outside
  it — is what licenses this lane keeping a second, operational view instead
  of stretching the curriculum lens to hold operations content.

## Taxonomy

- `se:VIEW-07D-RELIABILITY-PERFORMANCE` — the lane's own lens. Reliability
  content arrives from two very different angles in the anchors: a curriculum
  unit about *concepts* and an operations KA about *managed obligations*.
  Forcing either into the other's shape would erase that difference, so the
  view classifies operational concerns and leaves curriculum classification
  to the shared view. It is typed OPERATIONS rather than QUALITY because the
  inspected SWEBOK material treats these as things a running service must
  manage (measure, investigate, rehearse), not as model characteristics —
  which also keeps SEK-07A's quality-model territory untouched.
- `se:NODE-07D-RELIABILITY-RESILIENCE` and
  `se:NODE-07D-PERFORMANCE-CAPACITY` — two sibling concern clusters that the
  lane deliberately does **not** merge, because the task pack requires
  reliability and performance to be distinguished rather than conflated. What
  keeps them honest is `se:REL-07D-023` (CROSS_CUTS): SWEBOK KA6 1.6 plans
  performance, reliability and load balancing as one early-planning concern,
  so the honest structure is two intersecting clusters, not a clean partition
  and not one blob.
- `se:NODE-07D-OBSERVABILITY-FEEDBACK` — typed CONCERN, not
  QUALITY_ATTRIBUTE, on purpose. In both anchors measurement is a *mechanism*
  (telemetry, dashboards, KPIs feeding decisions), not a property the
  software exhibits. Making it a concern node lets the telemetry practice
  apply to it (`se:REL-07D-025`) without pretending "observability" is a
  quality attribute either source defines.
- `se:NODE-07D-CS2023-SE-RELIABILITY` — the curriculum unit node inside the
  shared CS2023 view. It exists so scope-bearing edges have a curriculum
  endpoint without the operational view absorbing curriculum identity.
  `se:REL-07D-030` (RELATED_TO) is the explicit cross-view bridge the pilot's
  multi-view rule requires; it asserts topical overlap only.

## Source locators

- `se:LOC-07D-CS2023-SE-RELIABILITY-KU` — heading-anchored for the same
  reason as SEK-06A's KU locator: the registry has page anchors only down to
  the SE section span, and the CS2023 PDF was read as extracted text, so a
  page number for the KU subsection would be invention. The locator string
  says both things.
- The seven SWEBOK locators (`se:LOC-07D-SWEBOK-KA6-*`,
  `se:LOC-07D-SWEBOK-KA12-*`) — chapter/section-scoped with 0-based PDF page
  indices and printed folios, verified against the hash-checked official PDF
  on 2026-11-07. KA6 2.3 and 2.4 share a printed page (6-8); their locator
  strings differ by section number, and locator uniqueness is per
  (source, locator string), so the shared page is not a problem. Front-matter
  tables (SWEBOK I.1/I.2, CS2023 Knowledge-Areas table) are deliberately not
  cited — that is SEK-03-era territory and would collide cross-lane.

## Evidence records

- `se:EVD-07D-CS2023-SE-RELIABILITY-KU` — the topic-inventory evidence that
  grounds the curriculum node. Split from the per-topic evidence because node
  identity (what the unit covers, core vs elective) is a different question
  from any single topic's content.
- `se:EVD-07D-CS2023-RELIABILITY-CONCEPT` — topic 1 alone. The dual
  quantification (probability of failure; mean time between failures) plus
  the fault-causes-failure framing is the curriculum's conceptual kernel; it
  later meets SWEBOK's failure-rate measurement in the concept record, where
  the two provenances stay visible side by side.
- `se:EVD-07D-CS2023-DEFECT-INJECTION-REMOVAL` — topics 3 and 5 together.
  Like SEK-06A's topic-with-subtopics grouping, the injection lever and the
  removal lever are meaningless apart — the whole point is the cycle — so one
  evidence record carries both and says so in its context.
- `se:EVD-07D-CS2023-ERROR-BUDGET` — topic 6. Deliberately thin: the
  curriculum *poses* a comparison and asks for domain mapping; it does not
  supply one. The evidence records the posing, not an imagined resolution.
- `se:EVD-07D-CS2023-FAULT-TOLERANCE-CONTEXT` — Non-core topic 8 (with 8a)
  plus topic 10. The flight-critical vs data-processing contrast is the
  single most load-bearing sentence in the lane for the failure-response
  pattern, and its elective status is recorded so downstream readers know how
  hard the curriculum itself pushes it.
- `se:EVD-07D-CS2023-MEASUREMENT-TELEMETRY` — Non-core topic 11; the
  curriculum-side half of the observability story, paired with the SWEBOK
  half in the practice and principle records.
- `se:EVD-07D-CS2023-SYSTEM-RELIABILITY` — topics 2 and 4. These are titles
  only in the extract, and the evidence resists importing system-reliability
  analysis methods from general knowledge; it licenses the *distinction*,
  which is exactly what the boundary claim asserts.
- `se:EVD-07D-CS2023-RELIABILITY-CORE-HOURS` — uses the shared Core-Hours
  locator, which would otherwise sit in the lane unused while still being
  required verbatim; it also documents that the unit has no separate CS Core
  allocation, a fact curriculum-comparison work will want.
- `se:EVD-07D-SWEBOK-PERF-RELIABILITY-EARLY` — KA6 1.6. A short section, but
  it carries two distinct facts the lane needs: performance and reliability
  are planned *together and early* (which justifies the CROSS_CUTS edge), and
  infrastructure services can be tested during development.
- `se:EVD-07D-SWEBOK-AVAILABILITY-SLA` — KA6 2.3. The availability
  manage-measure-investigate-report loop is what turns "availability" from a
  vocabulary word into a managed obligation; it also anchors the SLA concept
  and the SLA-anchored pattern.
- `se:EVD-07D-SWEBOK-CAPACITY-MGMT` — KA6 2.4. The forecast-translate-analyze
  -plan loop with costed options grounds both the capacity practice and the
  capacity-investment decision model. Its limitation notes the ISO relay:
  SWEBOK describes the need; the ISO text itself stays metadata-only.
- `se:EVD-07D-SWEBOK-FAILOVER-INEVITABLE` — KA6 2.5. The failure-is-inevitable
  stance is the professional-BOK spine of the plan-for-failure principle; the
  planned-during-development requirement is what connects it to the
  curriculum's context-dependent response.
- `se:EVD-07D-SWEBOK-MONITOR-EVIDENCE` — KA6 4.3. The "hope should not be a
  strategy" norm plus the real-time KPI inventory; the primary anchor for the
  observability concern, the evidence-over-hope principle and the telemetry
  practice.
- `se:EVD-07D-SWEBOK-RISK-REDUCTION-TRIO` — KA12 1.4. Admitted for the three
  failure-risk-reduction technique families and the assurance-case structure
  only; the dependability characteristic enumeration and integrity-level
  schemes border SEK-07A's model level and are explicitly not claimed.
- `se:EVD-07D-SWEBOK-RELIABILITY-MEASURES` — KA12 2.3.1. Names the
  measurements (error density, defect density, failure rate/MTTF) and the
  failure-data-to-reliability-model chain that connects to the curriculum's
  MTBF framing.
- `se:EVD-07D-SYNTHESIS-HOPE-BASED-OPS` — the lane's one clearly marked
  SYNTHESIS record. The anti-pattern needs a *characterization of
  counterproductivity* that neither source states; combining SWEBOK's norm
  with CS2023's measurement topic supplies an inference trail while flagging
  that no source sentence says it. The DERIVED_FROM edge from the anti-pattern
  points here so the derivation is auditable.

## Concepts

- `se:CON-07D-SOFTWARE-RELIABILITY` — a QUALITY_ATTRIBUTE concept built from
  two anchors (curriculum topic 1; SWEBOK measurement names). Its scope note
  refuses to be a "complete professional definition": both anchors are
  quantifications, and that is all the concept claims.
- `se:CON-07D-ERROR-BUDGET` — kept minimal because the curriculum names the
  approach without defining budget mechanics; inventing SLO-math here would
  exceed the anchor. The QUALIFIED_BY edge records that its applicability is
  domain-relative.
- `se:CON-07D-FAULT-TOLERANCE` — MECHANISM kind because both anchors describe
  *how* systems tolerate faults (context-chosen responses; planned
  failure-handling logic), not what the term means in the abstract.
- `se:CON-07D-SERVICE-LEVEL-AGREEMENT` — the availability records need a
  named instrument to refer to; SWEBOK 2.3 supplies exactly the
  operations-services sense, and the scope note fences off other
  communities' SLA variants.

## Claims

- The thirteen claims split cleanly by provenance: six CS2023
  (DEFINITION/PRACTICE_CLAIM/DOMAIN_BOUNDARY) and seven SWEBOK
  (DEFINITION/PRACTICE_CLAIM/NORMATIVE_CLAIM). Each carries scope language
  naming its source and section so no curriculum statement can be read as
  professional consensus or vice versa.
- `se:CLM-07D-SWEBOK-EVIDENCE-NOT-HOPE` is the only NORMATIVE_CLAIM: SWEBOK
  4.3 states a *should*, and the claim kind makes that visible instead of
  laundering a norm into a descriptive statement.
- `se:CLM-07D-SOFTWARE-VS-SYSTEM-RELIABILITY` is the boundary-keeper: its
  whole content is that the curriculum separates these scopes, which is the
  lane-level answer to "distinguish rather than conflate".

## Principles, practices, patterns, anti-pattern, decision models

- `se:PRI-07D-PLAN-FOR-FAILURE` — SWEBOK's inevitability stance is the
  rationale's spine; the CS2023 context contrast enters the *statement*
  ("matched to the failure's consequences") but the primary SUPPORTED_BY edge
  points at the SWEBOK evidence, keeping the strongest anchor load-bearing.
- `se:PRI-07D-EVIDENCE-OVER-HOPE` — crosses the two sources deliberately
  (SWEBOK norm + CS2023 elective topic); its limitation records that the
  curriculum side is Non-core so the principle is not overstated.
- The four practices each operationalize exactly one or two verified
  statements and say in their scope fields where procedural detail is the
  lane's own addition (e.g. the capacity practice's step ordering). The
  telemetry practice carries *two* evidence refs because it genuinely
  implements two anchors — the SWEBOK norm and the CS2023 instrument list.
- `se:PAT-07D-CONTEXT-DEPENDENT-FAULT-RESPONSE` — the pattern the two
  consequence examples were made for; its trade-off field keeps the honest
  cost (classification effort) visible.
- `se:PAT-07D-SLA-ANCHORED-AVAILABILITY` — a faithful restructuring of the
  SWEBOK 2.3 loop into problem/context/solution shape; nothing in the
  solution exceeds the section's elements.
- `se:ANT-07D-HOPE-BASED-OPERATIONS` — the lane's only anti-pattern, derived
  (via the synthesis evidence) from a norm's negation. Its scope field names
  the synthesis basis, mirroring SEK-06A's treatment of its QA-deferral
  anti-pattern.
- `se:DEC-07D-ERROR-BUDGET-OR-ERROR-FREE` — the curriculum poses this
  decision as an exercise; the decision model makes the exercise's structure
  (options, criteria, constraints) explicit without answering it. The
  consequence-context criterion imports the fault-tolerance contrast so the
  decision stays anchored in admitted evidence.
- `se:DEC-07D-CAPACITY-INVESTMENT` — SWEBOK 2.4's costed-options requirement
  is already a decision procedure in embryo; the model surfaces it.

## Coverage gaps

- `se:GAP-07D-NO-REAL-CASE-EVIDENCE` — the honest answer to "no invented
  cases": the evidence class is missing, so it is named.
- `se:GAP-07D-CS2023-ADJACENT-TOPICS` — records *why* SF-Reliability and
  PDC-Evaluation are absent (network-verified re-fetch impossible this
  session) instead of silently ignoring them; the resolution note names the
  re-verification path.
- `se:GAP-07D-SWEBOK-UNINSPECTED-SECTIONS` — bounds the professional-BOK
  claim surface: what this lane cites is what this lane read.
- `se:GAP-CS2023-RIGHTS-CLEARANCE` — the shared HIGH/OPEN rights gap; see the
  README's rights section. It is carried verbatim so every lane reports the
  same condition.

## Relationship assertions

- Thirteen SUPPORTED_BY edges give every claim its canonical evidence trace;
  three more do the same for the principle pair and the SLA pattern, and two
  for the decision models.
- The structural edges exercise the catalog deliberately: IMPLEMENTS
  (practice→claim), DEPENDS_ON (rehearsal practice needs the fault-tolerance
  concept), DERIVED_FROM (anti-pattern→synthesis), QUALIFIED_BY
  (error-budget applicability), CROSS_CUTS (reliability/performance
  intersection), APPLIES_TO (claims/practices to nodes), and one cross-view
  RELATED_TO. Every evidence-required predicate carries evidence_refs
  pointing at lane Evidence records; every qualifier-required predicate
  carries qualifiers.
- `se:REL-07D-030` deserves a final note: linking the curriculum node to the
  operational node is the concrete instantiation of the pilot's requirement
  that lifecycle/quality/discipline views not be forced into one hierarchy —
  the edge records where the views intersect without merging them.
