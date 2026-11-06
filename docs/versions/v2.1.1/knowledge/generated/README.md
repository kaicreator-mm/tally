# SEK-08 generated graph — deterministic RDF projection

**Task:** #349 (SEK-08 cross-domain integration; coordinator lineage: SEK-06 #321 closed, SEK-07 #333 independently verified)
**Build date (fixed reproducibility constant):** `2026-11-07`
**Status:** EXPERIMENTAL / HUMAN_REVIEW_PENDING — nothing here is ACCEPTED corpus knowledge.

This directory holds the machine-generated, byte-deterministic RDF projection of the
integrated Software Engineering knowledge graph. Canonical inputs are the ten reviewed
SEK-06/SEK-07 lane bundles (40 source paths, verbatim, never edited in this task), the
pinned knowledge model 0.3.0 (`../model/`), the relationship catalog, and
`../../authority/SOURCE_REGISTRY.yaml`. This is an experimental corpus integration, not a
production ontology adoption, not a Domain Package and not DomainForge promotion.

## Files

| File | Content |
|---|---|
| `graph.ttl` | Turtle serialization of the canonical projection (9,461 triples) |
| `graph.jsonld` | JSON-LD serialization of the same triple set |
| `manifest.sha256` | SHA-256 of the two graph files + canonical graph digest; verify with `sha256sum -c manifest.sha256` |
| `README.md` | This file: projection semantics, derivations and reconciliation decisions |
| `COVERAGE_REPORT.md` | Covered areas, explicit gaps, excluded legacy material, rights, unmeasured review burden |

Canonical graph digest (SHA-256 over the sorted N-Triples form of the projection):
`ec18a1aac76d91230337ab68d39173ec4ddf9195ab5424b10af32836693fc3cd`.

## Projection semantics

- **Node per record.** Every knowledge record becomes an RDF resource. `@id` is the
  expanded canonical CURIE (`se:` = `https://github.com/kaicreator-mm/tally#`), `@type`
  is the model class IRI (`se:Domain`, `se:Evidence`, …).
- **Slot predicates.** Record fields serialize under `se:<slot_name>` predicates.
  Reference-valued slots (`*_ref`, `*_refs`) serialize as URIRefs (never literals);
  `se:synthetic` as `xsd:boolean`; all other fields as plain string literals.
- **Typed relationship edges.** In addition to the `se:RelationshipAssertion` resource
  nodes (which carry context, rationale, qualifiers and lifecycle), each assertion also
  materializes the direct triple `<subject_ref> <predicate-URI> <object_ref>` using the
  catalog's `meaning` IRI (e.g. `se:SUPPORTED_BY`). A literal predicate value never
  appears as an edge; per the relationship catalog, `se:predicate` values are URIRefs.
- **Graph root.** One `se:KnowledgeGraph` resource (`se:GRAPH-V2.1.1-SEK08`) binds
  `se:graph_version` (`0.1.0`), `se:generated_on` (the fixed build date above) and
  `se:domain_ref`.
- **Identity.** 783 lane-owned records + 6 shared records (parsed-identical in every
  carrying lane) + 3 derived Source records = 792 record resources; zero collisions,
  zero dangling references (checked mechanically over the exported triples).

## Derived Source records

Lane files intentionally do not store `Source` records. The projection derives the three
referenced sources from `SOURCE_REGISTRY.yaml` (`se:SRC-SWEBOK-GUIDE-4-0A`,
`se:SRC-CS2023`, `se:SRC-NIST-SP-800-218-V1-1`; the six unused registry entries,
including all ISO records, are **not** projected — ISO remains metadata-only):

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

Four locator identities overlap between the legacy SEK-05-era fixture
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
bundles (integration tree = WIP base `7a70211b…` + ten lane tips; final tree also
carries this directory):

1. Assemble the merged graph from the ten `lanes/*/records.yaml` (unique-key YAML
   load; shared records must be parsed-identical) and derive the three Source records.
2. Apply the projection-only boundary-notes normalization.
3. Serialize the canonical triple set (sorted subjects/predicates/objects; the only
   date is the fixed build constant) to `graph.ttl` + `graph.jsonld`;
   write `manifest.sha256`.

Determinism evidence at this tree: five independent assemblies (canonical lane order ×2,
reversed order, rotated order, plus a fresh repeat) produced byte-identical
`graph.ttl`/`graph.jsonld` and identical canonical digest. Turtle and JSON-LD exports
were parsed back with rdflib 7.6.0 and verified triple-for-triple equivalent to the
internal canonical set (9,461 = 9,461 = 9,461), with every reference slot resolving to a
URIRef, no dangling record references, and relationship endpoints/predicates/qualifiers
conforming to `../model/RELATIONSHIP_CATALOG.yaml`.

Full commands, exit codes and the builder receipt are recorded in
`../../methods/observations/SEK-08.md`.
