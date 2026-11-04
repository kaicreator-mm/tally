# SEK-04 — LinkML Proof-of-fit Plan

**Status:** DESIGN_READY / TOOL_EXECUTION_NOT_RUN / NOT_ADOPTED  
**Task:** [#305](https://github.com/kaicreator-mm/tally/issues/305)  
**Input sample:** [SEK-03 #304](https://github.com/kaicreator-mm/tally/issues/304), especially KNOWLEDGE_RECORDS.yaml  
**Experiment date:** 2026-11-04  
**Candidate tool pin:** LinkML v1.12.1 (official latest release observed on this date; experiment pin only)

## 1. Question

Can LinkML represent the corpus's stable identifiers, source/evidence provenance, cross-file references, typed relationships and graph exports while preserving human-readable authoring and Git review quality better than the three requested authoring formats?

This is a proof-of-fit, not an adoption plan. The result may be ADOPT, ADOPT_WITH_AMENDMENTS, REPEAT_EXPERIMENT or REJECT.

## 2. Real sample

Use the SEK-03 vertical slice:
- SWEBOK Guide v4.0a source identity and Table I.1 locator;
- one SourceLocator record;
- one Evidence record;
- claim SE-CLM-0001;
- a typed SUPPORTS relationship;
- one human explanation and review-state field.

The larger SEK-03 data set remains the source of truth for the experiment. The three fixtures below are alternative representations of the same logical sample, not three canonical copies.

## 3. Formats under comparison

### A — Markdown + YAML Front Matter

- One Markdown file per claim.
- Structured metadata in front matter.
- Human explanation, assumptions and limitations in the Markdown body.
- Source and evidence records remain linked by stable IDs.
- Main risk: front-matter parsing is outside LinkML's native YAML data loader; an extraction adapter may be needed.

### B — YAML data + Markdown explanation

- One YAML record per typed entity (Source, SourceLocator, Evidence, Claim).
- A linked Markdown file carries rationale, contextual explanation and reviewer notes.
- Main risk: more files and cross-file reference resolution; Git Diff is granular but reviewers navigate multiple files.

### C — Standalone YAML

- A single YAML corpus document contains structured source, locator, evidence, claim and relationship records.
- Explanation and limitations are structured fields.
- Main risk: large diffs, verbose nesting and reduced narrative flexibility.

## 4. Candidate LinkML model

The candidate schema models:
- Source
- SourceLocator
- Evidence
- KnowledgeClaim
- RelationshipAssertion
- DomainNode
- ReviewDecision
- KnowledgeCorpus (tree root)

Stable IDs use LinkML identifier slots. Cross-object references use non-inlined class references. RelationshipAssertion uses subject/object IDs plus explicit entity-kind enums because the corpus needs polymorphic relationship endpoints; a separate graph-integrity check must resolve those IDs.

A SourceLocator unique key over source_ref + locator_value is included as a deliberate generator test. It must not be assumed that every target generator enforces this constraint.

## 5. Test matrix

| Dimension | Valid case | Invalid/adversarial case | What to inspect |
|---|---|---|---|
| Schema validity | Candidate schema conforms to LinkML metamodel | Unknown slot, invalid range, invalid URI | Metamodel validator diagnostics |
| Stable IDs | Unique Source/Locator/Evidence/Claim IDs | Duplicate identifier across records | LinkML runtime and generated validators |
| Local keys | Identifier vs key control | Duplicate key in one collection | Difference between global identifier and container-local key |
| Cross-file references | Evidence → Source/Locator; Claim → Evidence | Missing source, locator or evidence target | Whether loader/validator resolves external references |
| Cardinality | One/many evidence references | Missing required reference; wrong scalar/list shape | Validation errors and generated JSON Schema |
| Controlled values | Valid enums | Unknown claim/evidence/status enum | Runtime and JSON Schema parity |
| Unique keys | Unique source+locator pair | Duplicate source+locator pair | LinkML runtime vs JSON Schema generator limitation |
| Typed relations | CLAIM SUPPORTED_BY EVIDENCE | Unknown entity kind or dangling endpoint | Schema-level vs graph-level integrity |
| JSON Schema | Generate and validate JSON instances | Test unsupported constraints | Lost/changed semantics |
| JSON-LD | Generate context and JSON-LD | Missing type annotations at nested levels | @id, @type, context and round-trip |
| RDF | Generate Turtle/N-Triples | Missing class type, wrong IRI, blank-node churn | rdf:type, URI identity, deterministic output |
| Evolution | Add optional slot and new enum value | Rename/remove required slot without migration | Backward compatibility and migration effort |
| Git review | Small targeted edits in each format | Reorder/rename one field and one relationship | Diff noise and reviewer comprehension |

## 6. Known documentation-level limitations to test

1. LinkML identifier slots support object references by identifier when a referenced class is non-inlined.
2. A class without an identifier cannot be referenced with inlined=false.
3. JSON Schema generation does not support every LinkML unique_keys constraint.
4. JSON-LD context generation documents that @type information is not emitted at all levels.
5. LinkML RDF tooling uses RDFLib-supported serializations; inspect whether class_uri and slot_uri preserve useful RDF types and predicates.
6. Front Matter parsing is a format adapter concern, not a LinkML schema feature.
7. A polymorphic relationship endpoint represented as string ID + entity kind needs an additional graph-integrity validator.

These are documented hypotheses. Only executed tests can turn them into experiment results.

## 7. Planned tool commands

Run with the pinned LinkML version and preserve exact command output/version:

~~~bash
linkml --version
linkml validate schema/software-engineering.yaml
linkml lint schema/software-engineering.yaml
linkml validate -s schema/software-engineering.yaml --target-class KnowledgeCorpus fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml
linkml validate -s schema/software-engineering.yaml --target-class Source fixtures/yaml-plus-markdown/SOURCE.yaml
linkml validate -s schema/software-engineering.yaml --target-class SourceLocator fixtures/yaml-plus-markdown/SOURCE_LOCATOR.yaml
linkml validate -s schema/software-engineering.yaml --target-class Evidence fixtures/yaml-plus-markdown/EVIDENCE.yaml
linkml validate -s schema/software-engineering.yaml --target-class KnowledgeClaim fixtures/yaml-plus-markdown/SE-CLM-0001.yaml
gen-jsonld-context schema/software-engineering.yaml --output generated/schema.context.jsonld --emit-frame
~~~

Use the installed v1.12.1 generator help for JSON Schema and RDF options; do not assume flags across versions. At minimum, generate JSON Schema, JSON-LD context/frame and RDF/Turtle from the same schema and record the exact command lines.

## 8. Acceptance and decision

No adoption decision is possible until:
- the schema metamodel and lint checks run;
- valid and invalid fixtures are tested;
- reference closure and duplicate IDs are tested;
- JSON Schema, JSON-LD and RDF outputs are inspected;
- at least one schema evolution scenario is executed;
- A/B/C authoring and Git Diff are compared by a human reviewer.

If the local toolchain remains unavailable, the outcome is REPEAT_EXPERIMENT / NOT_RUN rather than an inferred PASS.

## 9. Current environment limitation

The current analysis sandbox does not have LinkML installed. A package-install attempt did not produce a usable installation. No CLI or generator execution is claimed, and no specific DNS/network cause is asserted. The schema and fixtures are design candidates only.
