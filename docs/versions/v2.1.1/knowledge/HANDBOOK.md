# Software Engineering Knowledge Graph — Human Handbook (v2.1.1 SEK-08, SEK-09H rebuild)

**Task:** #349 · **SEK-09H rebuild:** #368 (SEK-09F real-case integration) ·
**Build date (fixed):** `2026-11-07` (constant reused) ·
**Status:** EXPERIMENTAL / HUMAN_REVIEW_PENDING.

This handbook is the human entry point to the integrated knowledge graph materialized by
SEK-08 from the ten independently reviewed SEK-06/SEK-07 lane bundles and rebuilt by
SEK-09H with the two reviewed SEK-09F real incident cases. It explains what
the graph is, how to navigate it, how every claim traces to a source, what it does not
cover, what rights constrain it, how review states work, and how to rebuild it
byte-for-byte. Machine-facing details live in `generated/README.md`; coverage accounting
lives in `generated/COVERAGE_REPORT.md`.

**What this is not:** not production corpus knowledge, not an accepted ontology, not a
Domain Package, not a DomainForge promotion and not a Tally runtime dependency. Every
knowledge record is awaiting human review.

## 1. Domain boundary

The graph describes one bounded domain: **Software Engineering**
(`se:DOMAIN-SOFTWARE-ENGINEERING`) — "a bounded engineering discipline concerned with
the multi-person, multi-version construction and evolution of software, represented
through curriculum, secure-development and related classification views pending
professional-BOK reconciliation".

- **In scope:** curriculum knowledge-unit classification for software engineering;
  secure-development cross-cutting guidance; the practices, principles, patterns and
  decision models the reviewed sources actually state.
- **Out of scope:** the entirety of computer science; unreviewed professional
  body-of-knowledge claims pending lane-level content admission and review.
- **Boundary notes** (from the Domain record): no single source is treated as a complete
  ontology of the discipline; SWEBOK v4.0a identity/registry reconciliation is complete
  (#325 / PR #326) and SWEBOK content is cited only where a lane directly reinspected
  the located sections. One boundary note historically truncated in the lane YAML
  (an unquoted `#` parsed as a comment) is repaired **in the derived projection only** —
  the lane sources keep the defect visible; see `generated/README.md` for the exact
  raw line, rule and before/after values.
- One subdomain is currently carved out: `se:SUB-07B-SOFTWARE-CONSTRUCTION`.

## 2. How the graph is organized

Records live in typed collections (counts at this build):

| Collection | Count | What it holds |
|---|---|---|
| Domain / Subdomain | 1 / 1 | The bounded discipline and its one specialization |
| TaxonomyView / TaxonomyNode | 11 / 34 | Classification lenses and their nodes |
| Source / SourceLocator | 6 / 90 | Bibliographic authorities and exact inspected locations |
| Evidence | 142 | Source-grounded observations (132 paraphrase, 10 labelled synthesis) |
| KnowledgeClaim | 106 | Atomic, scoped, reviewable statements |
| Concept / Principle / Practice | 38 / 14 / 28 | Terms, guiding propositions, actionable activities |
| Pattern / AntiPattern / DecisionModel | 12 / 10 / 11 | Recurring structures, counterproductive approaches, decision aids |
| Case | 2 | Real incident postmortems (both PENDING_REVIEW; SEK-09H) |
| CoverageGap | 29 | First-class missing/weak/out-of-scope items |
| RelationshipAssertion | 271 | Directed typed edges |
| Counterexample / ReviewDecision | 0 / 0 | Intentionally empty — see §6 |

**Navigation.** Record ids are self-describing CURIEs (`se:` prefix):
`se:LOC-*` locators, `se:EVD-*` evidence, `se:CLM-*` claims, `se:PRA-*` practices,
`se:PAT-*`/`se:ANT-*` patterns/anti-patterns, `se:PRI-*` principles, `se:CON-*`
concepts, `se:DM-*` decision models, `se:GAP-*` coverage gaps, `se:CASE-*` cases,
`se:REL-*` or
`se:REL-…` assertions, `se:NODE-*`/`se:VIEW-*` taxonomy. Ids embed the owning lane
(`se:PRA-07C-*` → SEK-07C security lane; `se:EVD-09F-*`/`se:CASE-09F-*` → the SEK-09F
reviewed intake merged into the reliability-performance lane by SEK-09H). The RDF projection (`generated/graph.ttl`,
`generated/graph.jsonld`) exposes every record as a resource with `se:<slot>`
predicates; reference slots are IRIs, so any RDF tool can walk the graph.

## 3. Taxonomy views

Eleven views, all `coverage_state: PARTIAL`, deliberately not merged into one tree:

- **CS2023 curriculum view** (`se:VIEW-CS2023-SE`) — the nine SE knowledge units of the
  CS2023 Body of Knowledge (curriculum classification, not professional completeness).
- **SSDF security view** (`se:VIEW-SSDF-SECURITY`) — NIST SP 800-218 v1.1 practice groups.
- **SWEBOK-grounded views** (07A): engineering foundations (DISCIPLINE), process and
  lifecycle (PROCESS), quality models (QUALITY).
- **Lane views**: 07B construction (CURRICULUM, CS2023 software development
  fundamentals), 07C security engineering (SECURITY), 07D reliability/performance
  (OPERATIONS), 07E build & delivery process (PROCESS), 07F maintenance/evolution
  (LIFECYCLE), 07G software engineering management planning (PROCESS).

Cross-view semantics are typed relationship assertions (e.g. `CROSS_CUTS`), never a
forced universal hierarchy; the same topic may legitimately appear in several views.

## 4. Traceability: source → locator → evidence → claim

Every knowledge assertion traces through one chain, stored once (no duplicated source
facts downstream):

1. **Source** (`se:SRC-*`, derived from `authority/SOURCE_REGISTRY.yaml`): bibliographic
   identity, authority fit/limitations, rights handling, verification state. Six
   sources are used: SWEBOK Guide v4.0a, CS2023, NIST SP 800-218 v1.1, plus the three
   SEK-09H-registered first-party publisher incident sources (Cloudflare 2025-06-12
   outage, Cloudflare 2025-08-08 KV redesign, GitHub 2018-10-21 postmortem — the last
   three carry the source-revision-unpinned caveat: HTML revisions are mutable and no
   response SHA-256 is pinned).
2. **SourceLocator** (`se:LOC-*`): one exact inspected location (PDF page index/printed
   page/section/table, or named web-page section) within exactly one source;
   `(source_ref, locator)` is unique across the graph (90 locators).
3. **Evidence** (`se:EVD-*`): a concise source-grounded paraphrase (or an explicitly
   labelled synthesis inference) bound to exactly one locator; all 142 are
   `EXTRACTED_PENDING_REVIEW`.
4. **KnowledgeClaim / practice / principle / …**: knowledge records cite evidence via
   `SUPPORTED_BY` edges; all 106/106 claims have ≥1 supporting evidence record.
   Real incident **Cases** (`se:CASE-*`, SEK-09H) cite evidence via `evidence_refs`;
   they are PENDING_REVIEW and are not yet bound to any practice/pattern `case_refs`.

`se:LOC-*` locator texts state the exact position (e.g. "Body of Knowledge PDF page
indices 174-176; Software Engineering (SE), Overview"), so a reviewer can re-open the
registered source and check the paraphrase. The six registry sources not used by any
lane (five ISO/IEC/IEEE standards, NIST SP 800-218A) are intentionally absent from the
projection — see §6.

## 5. Relationships

271 typed, directed, reviewable edges across ten catalog predicates
(`../model/RELATIONSHIP_CATALOG.yaml` controls endpoint classes, direction and
qualifiers; no inverse edges are inferred):

| Predicate | Count | Meaning |
|---|---|---|
| SUPPORTED_BY | 157 | subject is supported by evidence (object is always an Evidence record) |
| APPLIES_TO | 30 | subject applies within a domain/subdomain/taxonomy-node scope |
| IMPLEMENTS | 18 | subject implements an object (e.g. practice implements SSDF task) |
| DERIVED_FROM | 17 | subject derived from evidence/claim/concept |
| DEPENDS_ON | 13 | subject depends on object |
| QUALIFIED_BY | 11 | object qualifies the scope/applicability of subject (qualifiers required) |
| RELATED_TO | 11 | contextual relation |
| CROSS_CUTS | 6 | subject cuts across an object scope |
| PART_OF | 6 | structural membership |
| BOUNDED_BY | 2 | subject bounded by an object boundary |

In the RDF projection each assertion exists both as a resource (with context,
rationale, qualifiers, lifecycle) and as the direct triple
`<subject> <predicate-URI> <object>`.

## 6. Applicability, limitations and coverage gaps

- **Nothing is accepted.** All 568 knowledge-class records are PROPOSED (162) or
  PENDING_REVIEW (406); all 106 claims are `review_status: PENDING`; there are zero
  ReviewDecisions and zero ACCEPTED records. Treat every statement as "claimed, pending
  human verification", never as settled knowledge.
- **Applicability is explicit per record** (`applicability`, `scope_description`,
  `limitations`, `assumptions`, `exceptions` where applicable); several records carry
  QUALIFIED_BY edges. Do not strip qualifications when quoting.
- **29 CoverageGap records (all OPEN)** enumerate what is missing: uninspected source
  sections, absent case/incident material, ISO standards that are metadata-only, NIST
  800-218A abstract-only admission, CS2023 adjacent topics, rights clearance for any
  use beyond the CS2023 educational permission. See `generated/COVERAGE_REPORT.md`.
- **No counterexamples yet; cases are unreviewed**: two real incident cases exist
  (`se:CASE-09F-CF-KV-DEPENDENCY`, `se:CASE-09F-GH-DB-FAILOVER`, both PENDING_REVIEW),
  integrated by SEK-09H from the reviewed SEK-09F intake; practices/patterns still rest
  on source evidence only and carry no `case_refs`. No record is a Counterexample: no
  contrary observation targets a specific qualified claim, so none was invented.
- **Known historical defect, preserved deliberately**: the shared Domain
  `boundary_notes[1]` truncation in the ten lane files (repaired in the projection
  only), and the four legacy locator-ID overlaps documented in the deterministic
  crosswalk (`generated/README.md`).

## 7. Rights

Per-source rights handling is carried verbatim from `authority/SOURCE_REGISTRY.yaml`
into the projection (`se:rights_handling` on each `se:SRC-*`):

- **SWEBOK Guide v4.0a** — do not commit/mirror/publish/redistribute the PDF;
  bibliographic metadata, exact locators and concise paraphrases only; no broad
  commercial or public-redistribution permission may be inferred (IEEE permission
  required beyond personal/educational conditions).
- **CS2023** — permission covers educational materials and programs; other use requires
  specific permission; store metadata/locators/concise paraphrases only; the
  educational-permission scope and the "other use requires specific permission" boundary
  stay explicit (tracked as the HIGH-severity `se:GAP-CS2023-RIGHTS-CLEARANCE` gap).
- **NIST SP 800-218 v1.1** — prefer original paraphrase, citation and locator; do not
  copy substantial text.
- **Cloudflare / GitHub first-party incident posts (SEK-09H)** — bibliographic
  references and short original paraphrases only; publisher retains copyright; do not
  copy substantial incident text or redistribute page content; the linked HTML revisions
  are mutable and unpinned.
- **ISO/IEC/IEEE standards (unused)** — bibliographic record only; AI-tool use
  restrictions; no normative text, clause extracts or generated summaries.

## 8. Lifecycle and review states

Records carry `lifecycle_status` (locator/source records are ACTIVE infrastructure;
knowledge records are PROPOSED or PENDING_REVIEW) and, for claims, `review_status:
PENDING`. Acceptance is exclusively a human act: a `ReviewDecision` (ACCEPT / REJECT /
REQUEST_CHANGES / DEFER) bound to the exact record revision and SHA-256 content digest
created by a qualified human reviewer after checking source, locator, interpretation,
applicability and rights. This build creates no ReviewDecisions and asserts no
acceptance. SEK-08's and SEK-09H's own human-review burden is unmeasured (`NOT_MEASURED`).

## 9. Reproducible build

The projection rebuilds byte-identically. Requirements: the ten lane bundles verbatim
(SEK-09H additionally requires its registry + reliability-lane canonical merge) on the
integration tree (base WIP `1ad69d6b…` at the SEK-09H rebuild), the pinned
model/catalog/registry, and the pinned environment (Windows-native Python 3.14.7,
linkml 1.12.1 / linkml-runtime 1.12.1, PyYAML 6.0.3, rdflib 7.6.0, jsonschema 4.26.0).

Steps: (1) assemble the merged graph from `vertical-slice/lanes/*/records.yaml`
(shared records must be parsed-identical); (2) derive the six Source records from the
registry; (3) apply the projection-only boundary-notes normalization; (4) serialize the
sorted canonical triple set to `generated/graph.ttl` + `generated/graph.jsonld` and
write `generated/manifest.sha256` (fixed build date; no timestamps leak into bytes).
At this tree: five assemblies in three lane orders produced identical bytes and
identical canonical digest `060fa255a2af69edcee0580d6ef960cdf3ca2bac117d570003c339d2dd94aa79`;
Turtle and JSON-LD parse back triple-for-triple equivalent (9,653 triples each); the
repository validator passes the merged graph; LinkML 1.12.1 JSON-Schema validation
reports 0 errors; validator self-tests pass 66/66. Exact commands and exit codes:
`../methods/observations/SEK-08.md` and `../methods/observations/SEK-09H.md`.

## 10. Where things live

```
docs/versions/v2.1.1/
├── authority/SOURCE_REGISTRY.yaml      # source rights/verification (SEK-09H extended: +3 sources)
├── knowledge/
│   ├── HANDBOOK.md                     # this handbook (SEK-08 output, SEK-09H updated)
│   ├── generated/                      # graph.ttl, graph.jsonld,
│   │                                   #   manifest.sha256, README.md, COVERAGE_REPORT.md
│   ├── case-intake/                    # SEK-09F intake sidecar (source of the merged cases; not canonical input)
│   ├── model/                          # pinned model 0.3.0, catalogs, validator
│   └── vertical-slice/
│       ├── KNOWLEDGE_RECORDS.yaml      # legacy SEK-03/SEK-05 fixture (excluded, untouched)
│       └── lanes/<ten lanes>/          # reviewed inputs; reliability-performance also carries the SEK-09F cases
└── methods/observations/SEK-08.md      # SEK-08 method observation + builder receipt
└── methods/observations/SEK-09H.md     # SEK-09H method observation + builder receipt
```
