# LinkML Proof-of-fit — Experiment Plan

**Task:** [SEK-04 #305](https://github.com/kaicreator-mm/tally/issues/305)  
**Parent:** [SEK-03 #304](https://github.com/kaicreator-mm/tally/issues/304)  
**Status:** EXECUTED_2026-11-05 / HISTORICAL_CHECKER_31_CASE_VALIDATION_PASS_ON_2c5c8b3d / FINAL_R2_SOURCE_LOCATOR_COHERENCE_REPAIR_PENDING_VALIDATION — see RESULTS.md §10.15 and POF_PLAN.md §16  
**Candidate version:** LinkML v1.12.1 (pin for this experiment only)

## 1. Decision question

Can LinkML model the source → locator → evidence → claim → domain-node → relationship chain with stronger validation and reliable JSON-LD/RDF projection, without imposing disproportionate authoring, review and migration costs?

This experiment does not presume adoption. The final decision remains one of ADOPT, ADOPT_WITH_AMENDMENTS, REPEAT_EXPERIMENT or REJECT.

## 2. Real-data fixture

The fixture is derived from the first real source-grounded record:
- Source: `SE-SRC-SWEBOK-GUIDE-4-0A`
- Locator: `SE-LOC-SWEBOK-KA-TABLE`
- Evidence: `SE-EVD-0001`
- Claim: `SE-CLM-0001`
- Domain node: `SE-KA-13`
- Relationships: `SUPPORTED_BY` and `INCLUDES`

All records remain pending human review. The LinkML fixture is a serialization experiment, not an accepted ontology.

## 3. Authoring formats

| Format | Fixture | Structural intent |
|---|---|---|
| A. Markdown + YAML Front Matter | `fixtures/frontmatter/SE-CLM-0001.md` | One reviewable Markdown artifact with all structured records in front matter |
| B. YAML data + Markdown explanation | `fixtures/yaml-plus-markdown/` | Separate Source, SourceLocator, Evidence and ClaimBundle YAML records, plus a human explanation |
| C. Standalone YAML | `fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml` | One structured corpus container with the same logical records |

The three fixtures must preserve identical IDs, statements, provenance and relationship assertions. Format A requires a front-matter extraction step before LinkML validation; LinkML's documented input model is YAML/JSON and does not itself imply native Markdown parsing.

## 4. Schema modeling choices under test

- Every record identifier uses the `uriorcurie` type with the declared `se:` prefix; non-inlined references use the same canonical CURIE form so RDF nodes expand to absolute resource IRIs.
- Source, locator, evidence, claim and domain-node references use non-inlined LinkML class ranges, serializing as identifier references rather than nested copies.
- `Evidence` and `DomainNode` inherit an abstract `GraphNode` superclass. `RelationshipAssertion.object_ref` has `range: GraphNode` and `inlined: false`, so it is a class reference and maps to an RDF resource node rather than a string literal. The redundant `object_kind` field is removed. Predicate-specific endpoint classes are declared in `RELATIONSHIP_RULES.yaml` and enforced by an independent graph-integrity check.
- `SourceLocator` declares a compound uniqueness constraint on (source_ref, locator).
- `KnowledgeCorpus` is the tree root for the standalone YAML format.
- A separate `ClaimBundle` root supports the split YAML + Markdown format.

These are experimental choices, not frozen schema commitments.

## 5. Test matrix

| ID | Test | Expected observation |
|---|---|---|
| L01 | Validate schema against LinkML metamodel | Required fields, valid slot ranges, prefixes and enum references are accepted |
| L02 | Validate all three positive fixtures | Equivalent logical records pass |
| L03 | Invalid enum value | Validator rejects an unknown claim/predicate status |
| L04 | Missing required statement or identifier | Validator rejects the record |
| L05 | Duplicate identifiers | Determine which LinkML validator/generator paths enforce uniqueness |
| L06 | Dangling source/locator/evidence/domain-node references | Determine whether non-inlined references are resolved by LinkML or require graph-integrity validation |
| L06a | Relationship object reference RDF node kind | Assert `rdf:object` is an RDF URIRef/resource node and never an xsd:string Literal for Evidence/DomainNode targets |
| L06b | Predicate-specific endpoint integrity | `SUPPORTED_BY` resolves to Evidence; `INCLUDES` resolves to DomainNode; wrong-class and dangling targets fail closed |
| L07 | Compound SourceLocator uniqueness | Compare LinkML validation with generated JSON Schema |
| L08 | Generate JSON Schema | Inspect identifier references, nullability, enum metadata and uniqueness behavior |
| L09 | Generate JSON-LD context | Inspect @id and @type mapping at top-level and nested records |
| L10 | Generate RDF/OWL representation of the LinkML schema | Inspect class/property URIs and schema semantics; this is schema output, not the instance knowledge graph |
| L11 | Convert each instance fixture to RDF/TTL and JSON-LD with `linkml-convert` | Inspect actual domain nodes, claim/evidence/source provenance and typed relationship edges; assert each `rdf:object` is a resource node, not a literal |
| L12 | Merge the four split B-format RDF outputs and compare graph semantics with A/C | Compare graph isomorphism after excluding only format-specific container-wrapper triples; preserve all domain-record and provenance triples |
| L13 | Repeat schema generation and instance conversion | Compare byte-level diff and RDF graph isomorphism; do not assume `--diff-stable` applies to `linkml-convert` |
| L14 | Schema evolution | Add a new optional slot, deprecate one slot, and validate old/new fixtures |
| L15 | Human diff review | Compare edit locality, noise, narrative context and review time for A/B/C |

## 6. Commands to run in a LinkML-enabled environment

> **CLI correction — 2026-11-05 (post-execution).** The originally planned command forms below were corrected after execution on LinkML v1.12.1: `linkml generate json-schema` has **no `--output/-o` option** (writes to stdout; redirect instead); `linkml generate rdf` has **no `--diff-stable` option** in 1.12.1 (`No such option '--diff-stable'`, exit 2) — default output was measured byte-stable anyway (L13); the schema `schema:` prefix must be `http://schema.org/` to match `linkml:types` or every generator/converter fails with `Prefix: schema mismatch`. Executed, verified forms:

~~~bash
python -m pip show linkml linkml-runtime
linkml --version
linkml validate experiments/linkml-pof/schema/software-engineering.yaml
linkml lint experiments/linkml-pof/schema/software-engineering.yaml
linkml validate -s experiments/linkml-pof/schema/software-engineering.yaml --target-class KnowledgeCorpus experiments/linkml-pof/fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml
linkml validate -s experiments/linkml-pof/schema/software-engineering.yaml --target-class ClaimBundle experiments/linkml-pof/fixtures/yaml-plus-markdown/SE-CLM-0001.yaml
linkml validate -s experiments/linkml-pof/schema/software-engineering.yaml --target-class Source experiments/linkml-pof/fixtures/yaml-plus-markdown/SOURCE.yaml
linkml validate -s experiments/linkml-pof/schema/software-engineering.yaml --target-class SourceLocator experiments/linkml-pof/fixtures/yaml-plus-markdown/SOURCE_LOCATOR.yaml
linkml validate -s experiments/linkml-pof/schema/software-engineering.yaml --target-class Evidence experiments/linkml-pof/fixtures/yaml-plus-markdown/EVIDENCE.yaml
linkml generate json-schema experiments/linkml-pof/schema/software-engineering.yaml > se-knowledge.schema.json
linkml generate jsonld-context experiments/linkml-pof/schema/software-engineering.yaml -o se-knowledge.context.jsonld
# Schema-level RDF/OWL output only (not the instance knowledge graph).
linkml generate rdf --format ttl experiments/linkml-pof/schema/software-engineering.yaml -o se-knowledge-schema.ttl

# Instance-level RDF/JSON-LD conversion for the standalone corpus.
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C KnowledgeCorpus experiments/linkml-pof/fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml -t ttl -o se-corpus-c.ttl
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C KnowledgeCorpus experiments/linkml-pof/fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml -t json-ld -o se-corpus-c.jsonld

# Extract the YAML front matter to a temporary file first, then run the same
# KnowledgeCorpus conversion against se-frontmatter-a.yaml.
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C KnowledgeCorpus se-frontmatter-a.yaml -t ttl -o se-corpus-a.ttl
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C KnowledgeCorpus se-frontmatter-a.yaml -t json-ld -o se-corpus-a.jsonld

# Split B-format records are converted independently, then merged as RDF graphs.
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C Source experiments/linkml-pof/fixtures/yaml-plus-markdown/SOURCE.yaml -t ttl -o se-b-source.ttl
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C SourceLocator experiments/linkml-pof/fixtures/yaml-plus-markdown/SOURCE_LOCATOR.yaml -t ttl -o se-b-locator.ttl
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C Evidence experiments/linkml-pof/fixtures/yaml-plus-markdown/EVIDENCE.yaml -t ttl -o se-b-evidence.ttl
linkml-convert -s experiments/linkml-pof/schema/software-engineering.yaml -C ClaimBundle experiments/linkml-pof/fixtures/yaml-plus-markdown/SE-CLM-0001.yaml -t ttl -o se-b-claim-bundle.ttl
~~~

The front-matter fixture must first be extracted to a temporary YAML file; that adapter is deliberately not assumed to be native LinkML functionality. The `linkml generate rdf` command emits an RDF representation of the **schema**. Actual instance-data graph export must use `linkml-convert`; these outputs are separate test products. For format B, merge the four separately converted RDF graphs with RDFLib before comparing against A/C. Compare graph semantics after excluding only the format-specific root wrapper, not domain/evidence/provenance triples.

## 7. Known limitations to test

Official LinkML documentation currently records:
- JSON Schema generation does not support LinkML `unique_keys` in full; identifier/key slots have a narrower uniqueness behavior when represented as dictionary keys.
- JSON-LD context generation does not emit `@type` directives at all levels.
- Schema RDF generation exposes a `--diff-stable` option to reduce blank-node label churn; this is an opt-in behavior. Do not assume the same option is available for `linkml-convert` instance dumps.
- Non-inlined class references serialize as identifiers. Whether all dangling cross-file references are rejected must be measured; do not assume reference resolution from the schema alone.

Official references:
- Release v1.12.1: https://github.com/linkml/linkml/releases/tag/v1.12.1
- Schema model: https://linkml.io/linkml/schemas/
- Identifiers and inlining: https://linkml.io/linkml/schemas/inlining.html
- Validation CLI: https://linkml.io/linkml/cli/validate.html
- JSON Schema generator: https://linkml.io/linkml/generators/json-schema.html
- JSON-LD context generator: https://linkml.io/linkml/generators/jsonld-context.html
- RDF: https://linkml.io/linkml/data/rdf.html
- RDF generator options: https://linkml.io/linkml/cli/generate.html

## 8. Historical pre-builder environment limitation

This section records the original analysis-container limitation before the Windows-native LOCAL_BUILDER run. It is superseded by the actual execution evidence in RESULTS.md §10. The initial container did not have linkml or linkml_runtime; its package-install attempt did not produce an installed package. This was an environment-specific failure, not a schema result. The later Windows-native environment installed and executed the pinned toolchain.
- At that historical point, only YAML syntax/static checks were available.
- LinkML execution was subsequently completed on Windows-native Python 3.14.7 with LinkML/linkml-runtime 1.12.1.
- No production adoption or full-model compatibility claim follows from the narrow PoF.

## 9. Exit criteria

The L01–L15 CLI and authoring-format experiment has been executed and recorded in RESULTS.md §10. The committed repository integrity checker passed Windows-native exact-source-HEAD validation at `2c5c8b3dfe877e4e9dc335d3c6d50b4440359ac3` (31/31 self-tests, A/B/C YAML+Turtle, JSON-LD edge identity and 24/24 CLI probes); the earlier independent source review found no remaining P0/P1/P2 defect at that SHA; a subsequent final R2 pass found a missing DomainNode source/locator coherence invariant in the same checker blob, superseding that clean disposition (see §16). The validation and review evidence are recorded in RESULTS.md §10.14 and Issue #305 comments #6643352340 / the current review terminal. The two commits immediately following the source-bearing HEAD were documentation-only reconciliation within the same four-path write set. The later final R2 source-coherence repair in §16 changes the checker; its resulting exact HEAD must receive fresh 32-case local validation before integration. SEK-04 remains OPEN pending that rebind, integration into #310 → #303, and the evidence-backed recommendation. The final canonical schema/authoring adoption decision remains in SEK-05 #308; SEK-04 does not adopt LinkML for production.


## 10. R2 P1-01 schema correction — typed resource target

The former `object_ref: string` mapped to an RDF Literal despite `slot_uri: rdf:object`. The revised candidate uses:

- Abstract `GraphNode` with the inherited identifier slot.
- `Evidence` and `DomainNode` as concrete subclasses of `GraphNode`.
- `RelationshipAssertion.object_ref` with `range: GraphNode` and `inlined: false`.
- Canonical `se:` CURIE identifiers (`uriorcurie`) so referenced nodes expand to absolute IRIs.
- No separate `object_kind` field; predicate-specific object classes are a single source of truth in `RELATIONSHIP_RULES.yaml`.

LOCAL_BUILDER execution with LinkML v1.12.1 verified the abstract-class range, non-inlined reference serialization, CURIE expansion and RDF object node kind in generated Turtle. The repository companion checker added by the R2 repair independently enforces predicate-specific endpoint classes, unique IDs and dangling-reference rejection.


## 11. R2 executable integrity checker — 2026-11-05

The RELATIONSHIP_RULES.yaml file is a declarative rule source, not an executable validator by itself. The companion checker is committed at:

docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py

Run its deterministic self-tests:

~~~bash
python docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py --self-test
~~~

Run the checker for all three authoring formats and their LinkML-generated Turtle:

~~~bash
python docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py --frontmatter docs/versions/v2.1.1/experiments/linkml-pof/fixtures/frontmatter/SE-CLM-0001.md --ttl se-corpus-a.ttl
python docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py --corpus docs/versions/v2.1.1/experiments/linkml-pof/fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml --ttl se-corpus-c.ttl
python docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py --bundle docs/versions/v2.1.1/experiments/linkml-pof/fixtures/yaml-plus-markdown/SE-CLM-0001.yaml --source docs/versions/v2.1.1/experiments/linkml-pof/fixtures/yaml-plus-markdown/SOURCE.yaml --locator docs/versions/v2.1.1/experiments/linkml-pof/fixtures/yaml-plus-markdown/SOURCE_LOCATOR.yaml --evidence docs/versions/v2.1.1/experiments/linkml-pof/fixtures/yaml-plus-markdown/EVIDENCE.yaml --ttl se-b-merged.ttl
~~~

Before the B check, merge the four independently generated B Turtle outputs into se-b-merged.ttl with RDFLib. The checker enforces canonical se: CURIE IDs and typed references, uniqueness, cross-record closure, predicate-specific subject/object classes, required evidence, and RDF resource-node semantics. It rejects Literal-valued rdf:object and wrong-class/dangling endpoints.

taxonomy_ref, taxonomy_refs, and view_id remain opaque string identifiers in this narrow PoF; they are not modeled as typed graph resources here. The complete taxonomy entity model is deferred to SEK-05 #308.


## 12. Exact RDF type-node validation — 2026-11-05

The strict endpoint class checker also rejects any non-URIRef value used as an endpoint's `rdf:type` object. This closes a malformed-RDF path where a valid `se:` class assertion could otherwise coexist with a Literal class assertion and be silently ignored. The checker self-test was expanded to 23 cases and the exact Windows-native run passed on checker blob `8561971c0404c3031e8fc2b0e9f9fd32796218e5` (see RESULTS.md §10.11). A subsequent independent review identified additional strictness gaps; the follow-up checker now targets 30/30 and requires a new exact-HEAD local run.

The LinkML v1.12.1 JSON-LD output has a known seven-triple nested-`rdf:type` omission (L09). Therefore, JSON-LD edge-node verification is a separate check: parse A/C and all four B JSON-LD outputs as RDF, require exactly one URIRef subject/predicate/object for every reified statement, reject Literal `rdf:object`, and compare edge identities to YAML. Do not claim JSON-LD/TTL graph equivalence or predicate-specific class validation from missing JSON-LD type triples. The separate JSON-LD edge identity check subsequently passed for A/C and B's four outputs (RESULTS.md §10.11).


## 13. Independent R2 follow-up — complete RDF type and ruleset contract

The exact Windows-native 23/23 run on checker blob `8561971c0404c3031e8fc2b0e9f9fd32796218e5` passed, including A/B/C Turtle, JSON-LD edge identity, and the recorded negative probes. A subsequent source audit found three gaps in that checker revision:

1. Endpoint class validation filtered out non-`se:` URIRef classes, allowing an expected class plus an unrelated foreign-namespace class.
2. Reified `rdf:Statement` nodes were selected by the expected type but their full type set was not checked; malformed Literal or incompatible additional type assertions could pass.
3. The ruleset validator compared `fail_closed` as a set (allowing duplicate list entries) and relied on Python equality (`True == 1`) for RDF expectation values.

The follow-up repair is committed inside the same #305 write set:
- Code repair commit: `148b656c39a44ff60c196438202521413b79c87c`.
- Checker blob: `4d26401bbab6ce3e43778f7096f265ff5424eb22`.
- New self-test target: **30/30 PASS**.
- Changes: exact complete endpoint rdf:type set (only the declared class and allowed GraphNode ancestor); global URIRef-only rdf:type object check; exact rdf:Statement type set; unique fail_closed list; strict per-field RDF expectation types; seven additional negative tests.
- At this historical checkpoint, exact Windows-native revalidation of the 30-case checker was **PENDING**. It was superseded by the 31-case checker and its exact-source-HEAD validation recorded in §15 / RESULTS.md §10.14.
- No schema, fixture, LinkML output, production runtime, CI workflow, or out-of-write-set path changed.


## 14. RDF reification completeness — 2026-11-05

A further source audit found that the checker only enumerated nodes typed `rdf:Statement`; an additional untyped node carrying `rdf:subject`, `rdf:predicate` or `rdf:object` could be ignored. The checker now enumerates all reification edge-property subjects and rejects any that is not also a typed `rdf:Statement`. A negative self-test covers an untyped extra edge.

- Repair commit: `5c0966de7f42b8dbe71c87994732177d4cd43846`.
- Checker blob: `503b370b016616c629f97fb47fc06b0079b574d9`.
- New self-test target: **31/31 PASS**.
- At this historical checkpoint, exact Windows-native revalidation was pending. The 31-case checker subsequently passed at source-bearing HEAD `2c5c8b3dfe877e4e9dc335d3c6d50b4440359ac3`; see §15 / RESULTS.md §10.14. The documentation-only reconciliation commits after that source-bearing HEAD require a final exact-HEAD identity rebind before integration.


## 15. Exact-source-HEAD validation and independent review — 2026-11-05

The final checker implementation was locally validated on Windows-native Python 3.14.7 (no WSL) at:

- Source-bearing HEAD: `2c5c8b3dfe877e4e9dc335d3c6d50b4440359ac3`
- Tree: `174c3512337b60c1083f8d75882c88cbcfe8659b`
- Checker blob: `503b370b016616c629f97fb47fc06b0079b574d9`
- Base / PR #310 HEAD: `41722b8b024760fd61d56d312b2916a3bd6c66d2`

Results: py_compile EXIT=0; checker self-test 31/31 PASS; A/B/C LinkML conversions and YAML+Turtle integrity PASS; 24/24 CLI negative/positive probes PASS; seven JSON-LD artifact edge-identity checks PASS. L09 nested `rdf:type` omissions remain explicit; no JSON-LD/TTL full-graph equivalence is claimed. Full command evidence, environment versions, artifacts and SHA256 manifest are in Issue #305 comment #6643352340 and its linked evidence bundle.

The earlier independent source audit at that source-bearing HEAD reported no remaining P0/P1/P2 defect; the final follow-up audit identified a missing DomainNode source/locator coherence invariant in the same checker blob, superseding that disposition (see §16). The documentation reconciliation described at this historical checkpoint did not change the checker blob. The later final R2 source-coherence repair in §16 does change the checker; its 32-case validation must pass on the new exact HEAD/Tree before integration. No merge, LinkML adoption, release or deployment is authorized by this record.


## 16. Final R2 follow-up — DomainNode source/locator coherence — 2026-11-05

The final independent R2 review found one remaining cross-record integrity gap in checker blob 503b370b016616c629f97fb47fc06b0079b574d9: DomainNode.source_ref and DomainNode.source_locator_ref were each resolved independently, but the referenced SourceLocator.source_ref was not required to equal the DomainNode.source_ref. A malformed DomainNode could therefore name Source A while citing a locator owned by Source B, with both references individually valid.

The checker is repaired within the existing #305 write set. It now resolves the DomainNode source and locator records, validates the locator's own source_ref, and rejects a mismatch. A dedicated self-test constructs two valid Source records and two valid SourceLocator records, then binds the DomainNode to mismatched source/locator values. The test must fail with the exact source-coherence diagnostic. Self-test target is now **32/32**.

**Current status: REVALIDATION_PENDING.** Run py_compile, the 32/32 self-test, all existing A/B/C YAML+Turtle checks, 24 existing CLI probes plus the new DomainNode source/locator mismatch negative probe, and the independent seven-artifact JSON-LD edge check on the exact new HEAD/Tree. The earlier 31/31 evidence applies to checker blob 503b370b... only and does not validate the repaired checker. No schema, fixture, runtime, CI or out-of-write-set path changes.
