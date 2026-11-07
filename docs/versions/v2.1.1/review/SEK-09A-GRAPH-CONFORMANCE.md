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

The exact WIP graph files and pinned catalog were fetched at the baseline above and inspected.

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

All 271 Turtle predicate slots are prefixed IRI terms, not quoted literals. All 271 JSON-LD predicate slots use `@id` objects. The ten distinct values used by the graph are members of the pinned Relationship Catalog's `meaning` IRIs. All 1,515 `*_ref`, `*_refs` and typed relationship targets inspected resolve to one of the 793 graph node IDs; no malformed target representation or dangling target was found.

The 793 graph nodes are consistent with 792 record resources plus one graph root. The catalog declares additional predicates that are not used by this corpus; unused catalog entries are not treated as graph defects.

## Exact-tree validation evidence

The WIP tree is byte-identical to the SEK-08 R2 tree. Under ADS #1359 §21, unchanged-tree PASS evidence is reused rather than needlessly rerun under a different merge-commit SHA.

R2 Builder receipt #6698638901 binds these checks to the exact tree above:

- Deterministic materialization: five assemblies across three lane orders produced one unique output/digest tuple; regeneration from the committed tree was byte-identical.
- RDF/JSON-LD/internal canonical equivalence: 9,461 = 9,461 = 9,461 triples; N-Triples digest payload parsed back as valid N-Triples.
- Repository graph validator: `GRAPH_VALIDATION=PASS`.
- Validator self-tests: `SELF_TESTS=66/66`.
- Manifest: `sha256sum -c manifest.sha256` reported both graph files OK.
- LinkML/linkml-runtime 1.12.1 JSON-Schema validation: `LINKML_ERRORS=0`.

The manifest entries read from WIP match the R2 receipt. These local commands were not rerun by this read-only audit because the underlying tree is unchanged. Their provenance is explicitly the R2 Builder receipt, not a new local execution by this lane.

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
