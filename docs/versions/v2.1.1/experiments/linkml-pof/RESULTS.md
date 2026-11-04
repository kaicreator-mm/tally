# LinkML Proof-of-fit — Initial Results

**Status:** DESIGN_AND_FIXTURES_READY / EXECUTION_PARTIAL / NO_ADOPTION_DECISION  
**Test date:** 2026-11-04  
**Pinned candidate:** LinkML v1.12.1

## 1. What exists

- A schema draft covering Source, SourceLocator, Evidence, DomainNode, KnowledgeClaim, RelationshipAssertion, ClaimBundle and KnowledgeCorpus.
- Three authoring-format fixtures intended to encode the same SWEBOK claim and provenance chain; equivalence remains NOT_PROVEN.
- A compound uniqueness constraint on source + locator.
- Non-inlined references for source, locator, claim and evidence relationships.
- Controlled vocabularies for source class, locator type, verification state, claim type/status and relationship predicates.

## 2. Static observations

| Area | Current observation | Evidence / status |
|---|---|---|
| Schema and fixture YAML | PyYAML 6.0.3 parsed the LinkML schema, all split YAML fixtures, standalone corpus and extracted front matter; duplicate-key detection passed | PASS for YAML syntax only; LinkML metamodel validation NOT_RUN |
| Identifiers | One identifier slot is defined for each record class | Schema draft; LinkML CLI NOT_RUN |
| Cross-file references | Independent closure checker resolved Source → Locator → Evidence → Claim → DomainNode and both relationship endpoint kinds across the split and standalone representations | PASS for independent static checker; LinkML's own reference resolution NOT_RUN |
| Compound uniqueness | SourceLocator has a compound key on source_ref + locator | Schema draft; JSON Schema generator limitation documented |
| JSON-LD | Nested @type behavior must be tested; official docs disclose a current limitation | Generator NOT_RUN |
| RDF | Diff-stable output is available as an option, not a default assumption | Generator NOT_RUN |
| Front Matter | Requires an extraction adapter before LinkML validation | Design observation; adapter not implemented |
| Human review | A/B/C fixtures are intended to encode the same logical corpus; A/C parse equivalence is structurally inspectable, but B assembly and RDF graph comparison have not been executed | A/B/C equivalence NOT_PROVEN; timed human review NOT_RUN |

## 3. Verified tool/documentation constraints

- LinkML v1.12.1 is the latest official release observed on 2026-11-04.
- The JSON Schema generator documentation states that `unique_keys` are not yet supported in JSON Schema; identifier/key dictionary forms provide only a limited uniqueness mechanism.
- The JSON-LD context documentation states that `@type` directives are not emitted at all levels.
- The RDF generator exposes `--diff-stable` to reduce blank-node label churn.

These are documented limitations, not failures observed from running this particular schema.

## 4. Not run

- LinkML schema metamodel validation.
- LinkML positive and negative data validation.
- Duplicate-ID and dangling-reference execution tests.
- JSON Schema and JSON-LD context generation.
- LinkML schema RDF/OWL generation.
- Instance-level `linkml-convert` export to RDF/Turtle and JSON-LD.
- Merge/isomorphism comparison of split B-format RDF outputs against A/C.
- RDF round-trip and deterministic-generation comparison.
- Schema evolution migration.
- Human review time/diff comparison.

The environment does not currently have `linkml` or `linkml_runtime` installed. The reason is not established. This is NOT_RUN, not PASS or a confirmed network failure.

## 5. Interim position

No adoption decision is justified. The schema and fixtures are useful as a test harness, but the core claims about validation, reference closure and graph export remain unproven until the pinned CLI is executed and the output inspected.


## 6. Static validation actually performed — 2026-11-04

Environment: Python 3.13; PyYAML 6.0.3; RDFLib 7.5.0.

| Check | Result | Boundary |
|---|---|---|
| YAML parse | PASS | Schema, Source, SourceLocator, Evidence, ClaimBundle, standalone corpus and extracted front matter parsed |
| Duplicate YAML keys | PASS | Custom PyYAML loader found no duplicate mapping keys in those seven YAML documents |
| Schema structural consistency | PASS | All class slots are declared; all ranges resolve to a class/enum/built-in type; all CURIE prefixes are declared; all non-inlined target classes declare the identifier slot |
| Required slots and enum values | PASS | All positive fixture objects satisfy the independent static checks |
| A/B/C logical equivalence | NOT_PROVEN | A and C now use the same `corpus_status` key; B's split files still require reproducible assembly, LinkML conversion and graph merge/isomorphism |
| Cross-file reference closure | HISTORICAL PASS / RECHECK_REQUIRED | The previous static checker passed the earlier string/object-kind draft; the new GraphNode/CURIE model and predicate-specific rules require rerun |
| Negative controls | HISTORICAL PASS / RECHECK_REQUIRED | The previous checker rejected unknown enum, missing statement, duplicate ID, dangling locator and wrong endpoint kind; rerun against the new GraphNode/CURIE contract |
| Manual RDF projection | PASS — limited | RDFLib generated a 45-triple manually constructed reference graph; Turtle and JSON-LD serialization round-tripped isomorphically. This does not validate LinkML instance conversion |

**Important:** these are independent static checks and a manually authored RDF reference projection. They are not LinkML metamodel validation, LinkML data validation, or LinkML-generated JSON Schema/JSON-LD/RDF output. They cannot be used to claim LinkML compatibility or adoption.

## 7. Updated interim position

The data model is now sufficiently concrete to run the pinned LinkML CLI experiment. The remaining critical gap is tool execution, not absence of real sample data. A bounded local builder is required for LinkML v1.12.1 schema validation, fixture validation, generator output and negative tests. No production dependency or adoption decision is implied.


## 8. Delivery-First correction from exact fixture readback — 2026-11-04

A direct read of the committed fixture files found two issues that invalidate the earlier A/B/C equivalence PASS statement:

1. Front Matter A used `status: DRAFT_PENDING_HUMAN_REVIEW`, while Standalone C used `corpus_status: DRAFT_PENDING_HUMAN_REVIEW`. A has now been corrected to `corpus_status`; the equivalence check must be rerun before PASS is restored.
2. B is represented by four separate YAML records plus a ClaimBundle. The earlier result asserted that these were assembled into the same logical corpus, but no committed/reproducible assembly procedure was documented. The equivalence result is withdrawn until the local experiment performs and records the split-record graph merge.

A second important distinction was made explicit: `linkml generate rdf` generates an RDF representation of the **schema**. It does not export the actual corpus instance graph. Actual instance-data conversion must use `linkml-convert` (e.g. schema + data + output format TTL/JSON-LD). The revised experiment now tests schema-level RDF separately from instance-level RDF/JSON-LD export.

This correction is based on the official LinkML RDF and conversion documentation:
- https://linkml.io/linkml/data/rdf.html
- https://linkml.io/linkml/data/conversion.html
- https://linkml.io/linkml/intro/tutorial01.html

No LinkML CLI result is inferred from this correction. CLI execution remains NOT_RUN.


## Additional environment attempt — 2026-11-04

A second toolchain attempt was made in the available container (Python 3.13.5). The environment already has PyYAML 6.0.3 and RDFLib 7.5.0, but not LinkML or linkml-runtime.

Command attempted:

```bash
python -m pip install --disable-pip-version-check 'linkml==1.12.1'
```

Result: FAILED / NOT_INSTALLED. pip retried five times and reported `Temporary failure in name resolution` (`Errno -3`) while resolving the package index. This is an observed failure in this container only; it does not establish the cause in any other environment. No package or repository state was changed by the attempt.

The pinned LinkML CLI and `linkml-convert` tests therefore remain NOT_RUN. A local builder with working package access or an approved offline wheel/dependency cache is still required.


## Model coverage boundary — 2026-11-04

This LinkML proof-of-fit is intentionally a **narrow source/provenance slice**. It models Source, SourceLocator, Evidence, DomainNode, KnowledgeClaim, RelationshipAssertion, ClaimBundle and KnowledgeCorpus. It does not yet demonstrate fit for distinct Concept, Principle, Practice, Pattern/AntiPattern, DecisionModel, Case/Counterexample, ReviewDecision or CoverageGap classes.

Do not describe this result as a full Software Engineering knowledge model. The broader entity/relationship catalog and selected canonical schema are assigned to SEK-05 #308, which remains preplanned and blocked on this experiment plus independent review of PR #303. Synthetic structural fixtures may be used there, but must not be admitted as domain knowledge.


## 9. Independent Exact-SHA R2 findings and remediation candidate — 2026-11-05

The independent R2 review bound to the previous exact HEAD and returned **BLOCK / P0=0 / P1=1 / P2=5**.

### P1-01 — RDF relationship object must be a resource node

The previous `RelationshipAssertion.object_ref` used `range: string` with `slot_uri: rdf:object`. Under LinkML's RDF mapping, a Type value maps to an RDF Literal, not a resource node. The `object_kind` enum did not change that mapping or resolve the target class.

The schema/fixture repair candidate now:
- introduces abstract `GraphNode`;
- makes `Evidence` and `DomainNode` concrete subclasses;
- changes `object_ref` to a non-inlined `GraphNode` class reference;
- removes the redundant `object_kind` field;
- changes fixture IDs and typed references to `se:` CURIEs using `uriorcurie`;
- adds `RELATIONSHIP_RULES.yaml` to enforce `SUPPORTED_BY → Evidence` and `INCLUDES → DomainNode`.

This only addresses the static schema defect. LinkML v1.12.1 metamodel/data validation, RDF/JSON-LD generation and URIRef-vs-Literal inspection remain NOT_RUN. Do not close P1-01 until independent review of the revised exact SHA and actual generated graph.

### P2-01 — LinkML CLI / instance export

Still NOT_RUN. The local environment must run the pinned LinkML v1.12.1 metamodel, positive/negative data, schema generator and `linkml-convert` tests. Manual RDFLib output is not LinkML evidence.

### P2-02 — A/B/C equivalence

The old PASS wording is removed. Current state is NOT_PROVEN. A and C use the same corrected `corpus_status` key; B still needs a reproducible split-record assembly and graph merge/isomorphism execution.

### P2-03 — CI

Exact previous HEAD checks remain: `trailer-check` PASS; `quality` FAIL (Formatting and Postgres integration). The current PR changes documentation/schema/fixtures only; no unrelated Go source is changed to silence repository-wide failures.

### P2-04 — Native dependency persistence

The live GitHub Dependency API readback now confirms the exact graph:
- #301 blocked by #300
- #302 blocked by #300
- #304 blocked by #301 and #302
- #305 blocked by #301, #302 and #304
- #308 blocked by #305

The OUTPUT_LEDGER and PR/workstream narratives are being reconciled. This does not freeze the DAG or close any task.

### P2-05 — CS2023 rights

The official report's exact Copyright and Reprint Permissions notice is recorded in the source registry and dossier. It grants use for educational-material/program development; other use requires specific permission. Contact details are recorded as metadata. This is a rights-handling boundary, not legal advice.

### Current decision

`LINKML_POF=NOT_PROVEN`; `A_B_C_EQUIVALENCE=NOT_PROVEN`; `SCHEMA_REPAIR=IMPLEMENTED_CANDIDATE / CLI_NOT_RUN`; `ADOPTION=NONE`.
