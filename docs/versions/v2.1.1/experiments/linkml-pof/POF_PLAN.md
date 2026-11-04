# LinkML Proof-of-fit — Experiment Plan

**Task:** [SEK-04 #305](https://github.com/kaicreator-mm/tally/issues/305)  
**Parent:** [SEK-03 #304](https://github.com/kaicreator-mm/tally/issues/304)  
**Status:** EXECUTED_2026-11-05 (LOCAL_BUILDER, LinkML v1.12.1, Windows-native Python) — see RESULTS.md §10  
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

## 8. Current execution limitation

The current analysis environment does not have `linkml` or `linkml_runtime` installed. A package-install attempt did not produce an installed package. The cause has not been established. Therefore:
- YAML syntax can be checked independently with PyYAML.
- LinkML metamodel validation, generated outputs and actual LinkML data validation remain NOT_RUN.
- No PASS, ADOPT or compatibility claim may be inferred from the static schema draft.
- A bounded LOCAL_BUILDER run is needed for the exact commands above when an environment with LinkML v1.12.1 is available.

## 9. Exit criteria

The experiment remains OPEN until positive and negative tests, generated artifacts, schema evolution, reference integrity and human-diff comparison have actual recorded results. A final adoption decision is prohibited while CLI validation or the required output tests remain unexecuted.


## 10. R2 P1-01 schema correction — typed resource target

The former `object_ref: string` mapped to an RDF Literal despite `slot_uri: rdf:object`. The revised candidate uses:

- Abstract `GraphNode` with the inherited identifier slot.
- `Evidence` and `DomainNode` as concrete subclasses of `GraphNode`.
- `RelationshipAssertion.object_ref` with `range: GraphNode` and `inlined: false`.
- Canonical `se:` CURIE identifiers (`uriorcurie`) so referenced nodes expand to absolute IRIs.
- No separate `object_kind` field; predicate-specific object classes are a single source of truth in `RELATIONSHIP_RULES.yaml`.

This is a schema repair candidate, not a claim that LinkML v1.12.1 has accepted or generated the intended graph. The CLI must verify the abstract-class range, reference serialization, CURIE expansion and RDF object node kind. The independent graph check must verify predicate-specific target classes and dangling references.
