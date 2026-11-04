# SEK-04 — Method Observation

## Windows builder update — 2026-11-04

The pinned LinkML 1.12.1 CLI was executed in an isolated Windows Python 3.12.14 environment; the earlier NOT_RUN statements below are historical. Full commands, outputs, fixtures and graph comparisons are in `../../experiments/linkml-pof/RESULTS.md` and its `evidence/` directory. The schema metamodel and positive fixtures passed, while lint returned 1 and negative integrity cases exposed missing closure/uniqueness enforcement. The initial HTTPS `schema` prefix conflicted with imported LinkML types; an alias-only correction unblocked schema generators. Original bare IDs still blocked instance TTL conversion.

On explicitly CURIE-qualified copies, separately converted and merged B records match A/C after removing only format-wrapper subjects. LinkML's JSON-LD output nevertheless loses seven nested `rdf:type` triples relative to its TTL output, and `rdf:object` remains a literal. The installed RDF generator rejects `--diff-stable`; successful default output was stable across the two measured runs. Optional-slot evolution and an explicit breaking rename/migration were exercised. Result: **REPEAT_EXPERIMENT**, pending IRI policy, semantic graph fidelity, reference validation, lint policy and independent review; this is not an adoption or full-model result.

**Task:** #305  
**Date:** 2026-11-04  
**Status:** OBSERVED / DRAFT

## Work performed

- Reviewed current official LinkML schema, inlining, validation, JSON Schema, JSON-LD and RDF documentation.
- Pinned v1.12.1 as the experiment candidate because the official GitHub release page identifies it as the latest release observed on this date.
- Drafted a schema with Source, SourceLocator, Evidence, DomainNode, KnowledgeClaim, RelationshipAssertion, ClaimBundle and KnowledgeCorpus.
- Created equivalent A/B/C fixtures from the real SWEBOK taxonomy claim and its provenance chain.
- Included identifier slots, non-inlined references, a compound source-locator uniqueness constraint, controlled vocabularies and a typed relationship assertion with endpoint-kind metadata.

## Findings from official documentation

1. LinkML schemas are YAML models whose classes, slots, types and enums define the shape of YAML/JSON data. The schema is not the same thing as the data.
2. Non-inlined references to identified classes serialize by identifier rather than duplicating nested objects. This is promising for source/evidence records shared by multiple claims.
3. JSON Schema generation does not currently implement LinkML `unique_keys` fully. This is a documented generator limitation and must be checked against the intended uniqueness contract.
4. The JSON-LD context generator documents that `@type` directives are not emitted at all levels. The RDF output therefore needs explicit inspection; successful context generation alone is insufficient.
5. RDF generation has an opt-in `--diff-stable` option intended to reduce blank-node label churn. Reproducibility must be tested with and without it.
6. Markdown front matter is not a native LinkML input format in the documented YAML/JSON validation workflow; it needs an extraction step. YAML + Markdown can preserve structured data and narrative separately, while standalone YAML minimizes file count but may make human explanations less comfortable.

## Negative and unresolved findings

- `linkml` and `linkml_runtime` are not installed in the current analysis environment.
- A package-install attempt did not produce a usable installation; the cause is not established.
- LinkML metamodel validation, data validation, reference resolution, generator outputs and round-trip tests are NOT_RUN.
- No adoption decision is possible from documentation inspection or static YAML parsing alone.
- Whether LinkML validates dangling non-inlined references across multiple files remains an explicit experiment, not an assumed capability.

## Method adjustment

Keep the schema experimental and the CLI status NOT_RUN. Use a bounded local builder only for the exact pinned LinkML commands once the fixtures are ready. Independently validate identifier/reference closure so that a generator's syntactic acceptance is not mistaken for graph integrity.


## Static validation and graph projection — 2026-11-04

- PyYAML 6.0.3 parsed the schema and all A/B/C fixture YAML documents; a duplicate-key loader found no duplicate keys.
- An independent static checker verified declared class slots, ranges, prefixes, required fields and enum values.
- A/B/C logical equivalence passed: extracted front matter equals standalone YAML; the four split YAML files assemble to the same logical corpus.
- Negative controls were rejected by the independent checker: unknown enum, missing required statement, duplicate ID, dangling source-locator reference and wrong relationship endpoint kind.
- A manually constructed RDFLib 7.5.0 graph produced 45 triples; Turtle and JSON-LD serialization round-tripped to an isomorphic graph with 45 triples.

These results establish that the sample's YAML structure, local reference closure and a manually designed RDF projection are coherent. They do **not** establish LinkML metamodel validity, LinkML reference validation, or correctness of LinkML-generated JSON Schema/JSON-LD/RDF. Those remain NOT_RUN until LinkML v1.12.1 CLI is executed.


## Delivery-First review correction — 2026-11-04

A current repository readback found that the front-matter fixture used `status`, whereas the schema and standalone fixture use `corpus_status`. The front-matter key has been corrected. The earlier A/B/C equivalence PASS is withdrawn pending re-execution.

The previous experiment plan also conflated schema RDF generation and corpus instance export. Official LinkML documentation distinguishes:
- `linkml generate rdf`: RDF representation of the LinkML schema.
- `linkml-convert -s schema data -t ttl/json-ld`: conversion of instance data into RDF/JSON-LD.

The revised POF plan separates schema-level RDF generation from instance-level graph export and requires the four split B records to be converted and merged before graph comparison. No LinkML CLI or instance-export PASS is claimed yet.


## Container toolchain attempt — 2026-11-04

- Container Python: 3.13.5.
- Available: PyYAML 6.0.3 and RDFLib 7.5.0.
- Missing: `linkml` and `linkml_runtime`.
- Attempted `python -m pip install --disable-pip-version-check 'linkml==1.12.1'`.
- pip failed after five retries with `Temporary failure in name resolution` (`Errno -3`) resolving the package index.
- This records the failure mode of this container only; it must not be generalized to the user's local environment.
- No package installation or source change succeeded. LinkML CLI and `linkml-convert` remain NOT_RUN.

Local execution is justified only for the missing pinned toolchain. It must preserve exact command output and distinguish environment failure from schema/test failure.


## Model coverage limitation

The current LinkML fixture is a narrow provenance-chain proof-of-fit, not the full v2.1.1 knowledge model. It does not model distinct Concept, Practice, Case/Counterexample, Pattern/AntiPattern, DecisionModel, ReviewDecision or CoverageGap classes. This is an explicit deferral to SEK-05 #308, not evidence that those entity types are unnecessary. No full-model fit or adoption conclusion is claimed.
