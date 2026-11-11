# SEK-08/SEK-09H generated graph — deterministic RDF projection

**Task:** #349 (SEK-08 cross-domain integration; coordinator lineage: SEK-06 #321 closed, SEK-07 #333 independently verified); **SEK-09H rebuild:** #368 (integration of the reviewed SEK-09F incident cases, #365 / PR #366)
**Build date (fixed reproducibility constant):** `2026-11-07` (unchanged from SEK-08; the SEK-09H rebuild reuses the constant so repeat builds stay byte-comparable)
**Status:** EXPERIMENTAL / HUMAN_REVIEW_PENDING — nothing here is ACCEPTED corpus knowledge.

This directory holds the machine-generated, byte-deterministic RDF projection of the
integrated Software Engineering knowledge graph. Canonical inputs are the ten reviewed
SEK-06/SEK-07 lane bundles (40 source paths; the reliability-performance lane also
carries the SEK-09F incident-case records added by SEK-09H), the pinned knowledge model
0.3.0 (`../model/`), the relationship catalog, and
`../../authority/SOURCE_REGISTRY.yaml` (extended by SEK-09H with the three SEK-09F
publisher sources). This is an experimental corpus integration, not a production
ontology adoption, not a Domain Package and not DomainForge promotion.

## Files

| File | Content |
|---|---|
| `graph.ttl` | Turtle serialization of the canonical projection (9,653 triples) |
| `graph.jsonld` | JSON-LD serialization of the same triple set |
| `manifest.sha256` | SHA-256 of the two graph files + canonical graph digest; verify with `sha256sum -c manifest.sha256` |
| `README.md` | This file: projection semantics, derivations and reconciliation decisions |
| `COVERAGE_REPORT.md` | Covered areas, explicit gaps, excluded legacy material, rights, unmeasured review burden |

Canonical graph digest (SHA-256 over the sorted N-Triples form of the projection):
`2625a6b336a8a0b392e7eccd13f157e3fb9030e9262836a12ec8afc85eb2d2e8`.

## SEK-09H change summary (vs the SEK-08 projection)

Baseline for this rebuild: WIP `1ad69d6b6fb945f5924ac7eee6fda798b843782a` (TREE
`7b460a92991faa8832f536a61823cffcd5101286`), which already contains the SEK-09F
case-intake sidecar (merged via PR #366). SEK-09H merged the reviewed intake into the
canonical inputs and rebuilt the projection:

- `../../authority/SOURCE_REGISTRY.yaml`: +3 first-party publisher sources
  (`SE-SRC-09F-CF-OUTAGE-2025`, `SE-SRC-09F-CF-KV-REDESIGN-2025`,
  `SE-SRC-09F-GH-POSTMORTEM-2018`). The registry preserves every limitation carried by
  the reviewed intake (first-party self-report, no independently audited causal
  generalization, unpinned mutable HTML revision) and adds one explicit conservative
  limitation on the GitHub source — "Historical and specific to GitHub's reported
  MySQL topology of 2018" — which restates the `historical/MySQL-topology-specific`
  scope already recorded in the #365 SOURCE-REVIEW. It is an addition, not a
  verbatim copy of the intake's authority fields.
- `../vertical-slice/lanes/reliability-performance/records.yaml`: +4 SourceLocator,
  +5 Evidence, +2 Case records whose parsed values are field-for-field identical to
  the independently reviewed SEK-09F intake (11 records; mechanical field-level
  comparison re-run after repair, see `../../methods/observations/SEK-09H.md`).
  Both cases stay `PENDING_REVIEW`, `synthetic: false`,
  `case_kind: POSTMORTEM`; all five evidence records stay `PENDING_REVIEW`,
  `EXTRACTED_PENDING_REVIEW`, `synthetic: false`.
- No Counterexample, ReviewDecision or ACCEPTED record was created; the 106 knowledge
  claims are byte-identical to the baseline; the 29 CoverageGap records are untouched.
- Triple-level proof vs baseline: 0 baseline triples removed; all 178 added JSON-LD
  property assertions (plus 14 new `rdf:type` triples = 192 new triples) sit only on
  the 14 new record subjects.

## Projection semantics

- **Node per record.** Every knowledge record becomes an RDF resource. `@id` is the
  expanded canonical CURIE (`se:` = `https://github.com/kaicreator-mm/tally#`), `@type`
  is the model class IRI (`se:Domain`, `se:Evidence`, …).
- **Slot predicates.** Record fields serialize under `se:<slot_name>` predicates.
  Reference-valued slots (`*_ref`, `*_refs`) and `se:predicate` serialize as
  URIRefs (never literals); `se:synthetic` as `xsd:boolean`; remaining fields
  as plain string literals.
- **Typed relationship edges.** In addition to the `se:RelationshipAssertion` resource
  nodes (which carry context, rationale, qualifiers and lifecycle), each assertion also
  materializes the direct triple `<subject_ref> <predicate-URI> <object_ref>` using the
  catalog's `meaning` IRI (e.g. `se:SUPPORTED_BY`). A literal predicate value never
  appears as an edge; per the relationship catalog, `se:predicate` values are URIRefs.
- **Graph root.** One `se:KnowledgeGraph` resource (`se:GRAPH-V2.1.1-SEK08`) binds
  `se:graph_version` (`0.1.0`), `se:generated_on` (the fixed build date above) and
  `se:domain_ref`.
- **Identity.** 794 lane-owned records + 6 shared records (parsed-identical in every
  carrying lane) + 6 derived Source records = 806 record resources; zero collisions,
  zero dangling references (checked mechanically over the exported triples).

## Derived Source records

Lane files intentionally do not store `Source` records. The projection derives the six
referenced sources from `SOURCE_REGISTRY.yaml` (`se:SRC-SWEBOK-GUIDE-4-0A`,
`se:SRC-CS2023`, `se:SRC-NIST-SP-800-218-V1-1`, plus the three SEK-09H-registered
first-party incident sources `se:SRC-09F-CF-OUTAGE-2025`,
`se:SRC-09F-CF-KV-REDESIGN-2025`, `se:SRC-09F-GH-POSTMORTEM-2018`; the six unused
registry entries, including all ISO records, are **not** projected — ISO remains
metadata-only):

- `id` = `se:` + registry `source_id` minus the `SE-` legacy prefix;
- `record_version` `0.1.0`, `lifecycle_status` `ACTIVE`,
  `created_on` = registry `verification.accessed_on` (fallback `as_of`);
- `source_roles` = `source_role` + `additional_roles`; `authors` = registry `authors`
  (for SWEBOK the registry lists an `editor`, which is carried into `authors` verbatim —
  no name is invented); `rights_handling` = registry `rights.handling` verbatim;
  `verification_state` = registry `verification.state`.

## Projection-only normalization: shared Domain `boundary_notes[1]`

Historical defect (independently confirmed twice in #333 receipts and preserved
verbatim in all ten lane files): the shared `se:DOMAIN-SOFTWARE-ENGINEERING`
`boundary_notes[1]` was authored in the SEK-06 lanes as an unquoted YAML plain scalar
containing `" #"`, so YAML parses a comment and truncates the value; the SEK-07 shared
snippet pack then propagated the already-truncated literal (their raw files carry the
short form; the three SEK-06 raw files — lanes/requirements line 42,
architecture-design line 45, testing-verification line 55 — carry the complete intended
sentence). Because the raw reviewed input unambiguously preserves the full text, the
projection applies one deterministic normalization — the derived graph only; no lane
file is rewritten and the source defect stays visible in-repo:

- Raw source: `docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/records.yaml` line 42
  (lane 06A, first canonical carrier).
- Rule: the intended value is the verbatim raw sequence-entry text after the `- ` marker
  (the `" #"` pair is a YAML comment cut at parse time, not content).
- Before (parsed, as in all ten lane files):
  `SWEBOK v4.0a identity and registry reconciliation is complete (SEK-02R`
- After (projection): `SWEBOK v4.0a identity and registry reconciliation is complete
  (SEK-02R #325 merged via PR #326); SWEBOK content is cited only where a lane has
  directly reinspected the located sections, and such records stay
  PROPOSED/PENDING_REVIEW until human review.`

## Locator ID-space crosswalk (deterministic)

Four locator identities overlap between the legacy SEK-03-era fixture
(`../vertical-slice/KNOWLEDGE_RECORDS.yaml`, `SE-LOC-*` id space) and lane records
(`se:LOC-*`). Mechanical comparison over (mapped source, locator text, locator type,
verification state) confirms equivalence in all four cases; the canonical graph identity
is the lane record, and no equivalence beyond these four verified fields is asserted:

| Legacy id (SEK-03 fixture, unmodified) | Canonical graph id | Verified match |
|---|---|---|
| `SE-LOC-CS2023-BOK-TABLE` | `se:LOC-CS2023-BOK-TABLE` | same source (`se:SRC-CS2023`), same locator text, TABLE, LOCATOR_VERIFIED |
| `SE-LOC-CS2023-SE-OVERVIEW` | `se:LOC-CS2023-SE-OVERVIEW` | same source, same locator text, SECTION, LOCATOR_VERIFIED |
| `SE-LOC-CS2023-SE-TOOLS` | `se:LOC-CS2023-SE-TOOLS` | same source, same locator text, SECTION, LOCATOR_VERIFIED |
| `SE-LOC-NIST-218-RISK` | `se:LOC-NIST-218-RISK` | same source (`se:SRC-NIST-SP-800-218-V1-1`), same locator text, SECTION, LOCATOR_VERIFIED |

## Legacy SEK-03 fixture — excluded from the projection

`../vertical-slice/KNOWLEDGE_RECORDS.yaml` (17 locators, 12 evidence records, 12 claims,
12 relationship assertions) was explicitly inspected for a lossless mapping into model
0.3.0 and **retained as a historical fixture instead**; the mapping is not lossless
(see `COVERAGE_REPORT.md` "Excluded legacy material" for the field-level analysis).
Nothing in that file was modified, silently omitted or silently coerced.

## Reproducible build

Windows-native, pinned: Python 3.14.7, `linkml==1.12.1`, `linkml-runtime==1.12.1`,
PyYAML 6.0.3, rdflib 7.6.0, jsonschema 4.26.0. From a tree that contains the ten lane
bundles plus the SEK-09H canonical merge (base = WIP `1ad69d6b…`):

1. Assemble the merged graph from the ten `lanes/*/records.yaml` (unique-key YAML
   load; shared records must be parsed-identical) and derive the six Source records.
2. Apply the projection-only boundary-notes normalization.
3. Serialize the canonical triple set (sorted subjects/predicates/objects; the only
   date is the fixed build constant) to `graph.ttl` + `graph.jsonld`;
   write `manifest.sha256`.

Determinism evidence at this tree: five independent assemblies (canonical lane order ×2,
reversed order, rotated order, plus a fresh repeat) produced byte-identical
`graph.ttl`/`graph.jsonld` and identical canonical digest. Turtle and JSON-LD exports
were parsed back with rdflib 7.6.0 and verified triple-for-triple equivalent to the
internal canonical set (9,653 = 9,653 = 9,653), with every reference slot resolving to a
URIRef, no dangling record references, and relationship endpoints/predicates/qualifiers
conforming to `../model/RELATIONSHIP_CATALOG.yaml`.

Full commands, exit codes and the builder receipt are recorded in
`../../methods/observations/SEK-08.md` (SEK-08) and
`../../methods/observations/SEK-09H.md` (SEK-09H rebuild).
