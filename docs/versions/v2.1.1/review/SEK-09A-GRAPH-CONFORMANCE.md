# SEK-09A — Independent Graph / Export / Schema Conformance Audit

**Task:** #352 · **Coordinator:** #351 · **Parent:** #299  
**Baseline:** `wip/v2.1.1-knowledge` @ `902126199c3ec18665873f5b6a6afe59854369ad`  
**TREE:** `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9`  
**Branch:** `task/tally-v211-sek09a-graph-conformance`  
**Sole write path:** this report  
**Audit mode:** independent GitHub read-only structural inspection; exact-tree Builder gates reused where unchanged.

## Result

**SEK09A=PASS** — structural and identity invariants checked in this audit are consistent.  
**P0=0 / P1=0 / P2=0 / P3=0.**

## Exact identity

- WIP ref readback: HEAD `902126199c3ec18665873f5b6a6afe59854369ad`.
- WIP merge commit tree: `a7b972fea96b9f12bbb8066fe6c88c13beb5f8d9`, identical to reviewed SEK-08 R2 tree.
- SEK-08 reviewed source commit: `74370f10d7cc294dec0275d835082feb28803e79`; merge commit `902126199c3ec18665873f5b6a6afe59854369ad`.
- R2 exact-SHA review already established R1→R2 as one README-only path and BASE→R2 as the original 47-path write set. WIP merge preserves the exact R2 tree.
- The 47 paths are the 40 immutable lane input files plus seven SEK-08 task-owned outputs. No schema, catalog, registry, validator or lane source was changed by R2.

## Independent structural checks

The exact WIP graph files and pinned catalog were fetched at the baseline above and inspected. The combined independent review used the following read-only methods, all with exit code 0 unless explicitly marked NOT_RUN below:

- `git rev-parse HEAD HEAD^{tree} HEAD^ HEAD^^{tree}` and `git diff --name-status 902126199c3ec18665873f5b6a6afe59854369ad dbfc72cd717a3ee2c8af79e7ee9b4aba951f4c83` for exact identity and write-set checks.
- Python 3.14.7 with rdflib 7.6.0 and JSON/YAML parsers for node, reference, endpoint, predicate, qualifier, duplicate-signature, namespace and serialization-equivalence checks.
- `python validate_knowledge_graph.py --catalog ENTITY_CATALOG.yaml --rules RELATIONSHIP_CATALOG.yaml --self-test` from `knowledge/model/`: `SELF_TESTS=66/66`.
- `sha256sum -c manifest.sha256` from `knowledge/generated/`: both graph files OK.

| Check | Observed |
|---|---:|
| JSON-LD graph nodes | 793 |
| Unique JSON-LD `@id` values | 793 |
| `se:predicate` slots in Turtle | 271 |
| `se:predicate` slots in JSON-LD | 271 |
| Distinct predicate values used | 10 |
| Predicate values outside catalog `meaning` IRIs | 0 |
| Reference / typed-edge targets checked | 1,515 |
| Malformed reference values | 0 |
| Dangling reference targets | 0 |
| Assertion endpoint/class-conformance violations | 0 |
| Required-qualifier assertions carrying qualifiers | 78/78 |
| Missing or unexpected required qualifiers | 0 |
| Duplicate subject/predicate/object signatures | 0 |
| Declared namespaces | rdf, xsd, se; consistent across serializations |

All 271 Turtle predicate slots are prefixed IRI terms, not quoted literals. All 271 JSON-LD predicate slots use `@id` objects. The ten distinct values used by the graph are members of the pinned Relationship Catalog's `meaning` IRIs. All 1,515 `*_ref`, `*_refs` and typed relationship targets inspected resolve to one of the 793 graph node IDs; no malformed target representation or dangling target was found. Catalog subject/object classes and required qualifiers were checked for all 271 assertions.

The 793 graph nodes are consistent with 792 record resources plus one graph root. The catalog declares additional predicates that are not used by this corpus; unused catalog entries are not treated as graph defects.

## Exact-tree validation evidence

The WIP tree is byte-identical to the SEK-08 R2 tree. Under ADS #1359 §21, unchanged-tree PASS evidence is reused only for checks not independently rerun; the review executions above are distinguished below.

R2 Builder receipt #6698638901 binds these checks to the exact tree above:

- Deterministic materialization: five assemblies across three lane orders produced one unique output/digest tuple; regeneration from the committed tree was byte-identical. **Reused R2 evidence; NOT_RUN in this lane.**
- RDF/JSON-LD/internal canonical equivalence: **independently rechecked** with rdflib; both serializations produced 9,461 triples with identical triple sets, and the sorted N-Triples SHA-256 digest prefix `d924e78f371dfe8c` was reproduced.
- Repository graph validator: `GRAPH_VALIDATION=PASS`. **Reused R2 evidence; NOT_RUN in this lane.**
- Validator self-tests: **independently rerun**, `SELF_TESTS=66/66`, exit 0.
- Manifest: **independently rerun**, `sha256sum -c manifest.sha256` reported both graph files OK, exit 0.
- LinkML/linkml-runtime 1.12.1 JSON-Schema validation: `LINKML_ERRORS=0`. **Reused R2 evidence; NOT_RUN in this lane.**

The manifest entries read from WIP match the R2 receipt. Deterministic regeneration, the full repository validator and LinkML execution retain their original R2 Builder provenance; no NOT_RUN item is represented as a new execution by this lane.

## Limitations

- This is structural and serialization qualification, not semantic truth review of every claim; that is SEK-09B.
- A valid graph does not establish human acceptance, source authority, complete taxonomy coverage or practical usefulness.
- All corpus records remain human-review-pending. No ReviewDecision or ACCEPTED record is created by this audit.
- This report does not authorize schema freeze, production ontology adoption, DomainForge synchronization or Domain Package promotion.

## Evidence references

- SEK-08 R2 Builder receipt: #349 comment #6698638901.
- SEK-08 R2 independent final review: PR #350 review #5808758478.
- WIP merge: PR #350 / merge commit `902126199c3ec18665873f5b6a6afe59854369ad`.
- Pinned Relationship Catalog blob at this tree: `a0f088d11878b27b600e604f2cc8d792a85cca9d`.
- Inspected `graph.ttl` blob: `1c0aee238901b1d02ac00ab19e0235a01ba0dfd8`.
- Inspected `graph.jsonld` blob: `2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00`.
