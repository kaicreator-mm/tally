# LinkML Proof-of-fit — Initial Results

## Windows LOCAL_BUILDER readback — 2026-11-04 (supersedes interim statuses below)

**Decision: REPEAT_EXPERIMENT.** LinkML 1.12.1 and linkml-runtime 1.12.1 ran in an isolated Windows CPython 3.12.14 virtual environment. This is evidence for the narrow provenance slice only; it is not a full-model fit or adoption decision. The original input was commit `a5e438d0f0bebc678e581f4518e17e0aeae68163`, tree `07f40bc237ae9f4ea62e49f78a5badc85e711cde`, parent/base `e988321439f65b55936a691c02250b6d62a29409`.

| Experiment | Actual observation |
|---|---|
| Metamodel | Baseline and prefix-fixed schema `linkml validate <schema>` exit 0, `No issues found`. |
| Lint | Exit 1. The fixed schema still reports 72 warnings, including recommended descriptions and naming; the baseline also warns that `schema` maps to HTTPS rather than the canonical HTTP namespace. This is not a lint PASS. |
| A/B/C validation | Extracted A and C validate as KnowledgeCorpus; all four B files validate separately as Source, SourceLocator, Evidence and ClaimBundle. Extracted A equals C as data. B's records assemble to the same record content; B has no corpus wrapper, so the comparison supplies C's wrapper id/title/status rather than claiming those fields exist in B. |
| Negative controls | LinkML rejects an invalid enum, missing statement and missing id. It accepts duplicate identifiers, a dangling locator, a relationship endpoint with the wrong kind and duplicate `(source_ref, locator)`. An independent closure/uniqueness checker rejects those four. It checks only the modeled local typed references: external taxonomy and view identifiers are not resolved. A passing CLI status alone does not establish closure. |
| JSON Schema and context | Baseline context generation encounters a `schema` prefix conflict; baseline JSON Schema generation succeeds when captured from stdout. The tested fix renames the local HTTPS alias to `schemaorg`, preserving the expanded slot IRIs and avoiding the imported HTTP `schema` collision. The plan's `--output` is rejected by this version's JSON Schema CLI; capturing stdout generates valid JSON Schema. Fixed-schema context generation succeeds. Independent `jsonschema` validation of the generated schema agrees on these positive/negative cases, including its failure to catch the four integrity controls; it is not an independent graph model. |
| Schema RDF | `linkml generate rdf` on the original schema fails on that prefix conflict; after the alias fix it produces schema RDF/Turtle. This is a schema representation, not knowledge-instance export. The requested `--diff-stable` option is rejected (exit 2); the installed `linkml generate rdf --help` does not list it. |
| Instance conversion | After the alias fix, original A/C JSON-LD exports succeed but original A/C and all four B TTL exports fail: bare identifiers produce `Unknown CURIE prefix: @base`. No successful original-fixture TTL graph is claimed. |
| Explicit CURIE probe | Derived copies qualify record IDs and typed local references with `se:`; originals remain unchanged. `linkml-convert` produces A/C TTL and JSON-LD, and four separate B TTL files. The merged B graph has 52 triples; A and C have 57 each. Removing only outgoing triples from the explicitly typed ClaimBundle/KnowledgeCorpus wrapper subjects leaves 47 triples in each and all three are RDF-isomorphic. This is a conditional result for transformed IDs, not proof the original fixtures export. Wrapper metadata is excluded. |
| Instance semantics | CURIE TTL contains typed nested records and URI links for source, locator, evidence and subject references. `rdf:object` is a literal such as `"se:SE-EVD-0001"` because `object_ref` is a string; it is not an RDF edge to that resource. For C, generated JSON-LD parses to 50 triples vs Turtle's 57; the seven missing triples are the nested record `rdf:type` statements. The formats are therefore **not** isomorphic. The exact difference is `evidence/windows-312-analysis/ttl-only.nt`. |
| Round-trip | The CURIE-derived C JSON-LD converts back to a YAML object equal to its derived input, even though its RDF graph lacks the seven types above. TTL-to-YAML fails with `SourceLocator.__init__() got an unexpected keyword argument 'source_refs'`; graph round-trip is therefore not established. The raw traceback is retained. |
| Reproducibility | Two successful generations of JSON Schema, context and default schema RDF are byte-identical; the two schema RDF graphs are isomorphic. The CURIE A/C TTL and JSON-LD and each B TTL are byte-identical across two runs; repeated A/C TTL graphs are isomorphic. This is a two-run observation in one environment, not a general determinism guarantee. Two empty files produced by failed original TTL exports are not evidence of stability. `--diff-stable` itself could not be tested because the option is unavailable. |
| Evolution | Adding optional `assessment_status` while deprecating `review_status` accepts the old and dual-write records; dual-write is rejected by the closed original schema. Removing the old slot and requiring the new one rejects the old record, while the explicitly mapped record passes the renamed schema and fails the original. A first probe accidentally made the added slot required, and its old-record failure is preserved separately. Deprecation was modeled; warning behavior was not established. |

Human review is qualitative, not a timed study: A puts narrative and structured data together but requires an extraction adapter; B localizes separate record edits but requires four-file assembly and closure; C is self-contained but combines unrelated record edits. No review-time comparison is claimed. The remaining choice of ID/IRI policy, typed relationship objects, JSON-LD type preservation, closure enforcement and lint policy needs a follow-up experiment and independent review.

### Reproduction and evidence

`evidence/windows-312/` contains the untouched-baseline run, `windows-312-prefix-fix/` the run after the schema alias fix, `windows-312-curie-probe/` the explicitly transformed instance probe, `windows-312-analysis/` the stdout JSON Schema and graph differences, `windows-312-evolution/` the compatible/rename probe, and `windows-312-roundtrip/` the conversion readback. Each `commands.json` records argv, working directory, exit code and complete stdout/stderr for CLI invocations. JSON Schema convenience files and the merged graph have excess final newlines normalized for Git; the complete generator stdout is retained in `commands.json` and its two raw outputs also match. Environment, package freeze and schema SHA-256 are recorded; `evidence/pip-install.log.gz` contains the full pip installation log. The initial `py -3.12 -m venv ...` returned `No suitable Python runtime found`; the explicit existing Windows CPython executable then created the isolated venv successfully. Installation used `python -m pip install --disable-pip-version-check linkml==1.12.1 --log <pip.log>` and exited 0. No WSL was used.

From the repository root, with a Windows venv containing the pinned package, run the `run_*.py` scripts with a **new, empty output path**. Run `run_windows_builder.py` against the original baseline before applying the alias fix and again after it; then run `run_curie_probe.py`, `run_analysis.py`, `run_evolution.py` and `run_roundtrip.py`. The analysis and round-trip scripts read the named evidence directories shown above. Commands and artifacts record the exact tested inputs, including failed commands, rather than silently replacing them.

**Status:** DESIGN_AND_FIXTURES_READY / EXECUTION_PARTIAL / NO_ADOPTION_DECISION  
**Test date:** 2026-11-04  
**Pinned candidate:** LinkML v1.12.1

## 1. What exists

- A schema draft covering Source, SourceLocator, Evidence, DomainNode, KnowledgeClaim, RelationshipAssertion, ClaimBundle and KnowledgeCorpus.
- Three equivalent authoring-format fixtures using the same SWEBOK claim and provenance chain.
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
| Human review | A/B/C fixtures exist; parsed A and C are equivalent, and the split B files assemble to the same logical corpus; no timed reviewer experiment has been run | Structural equivalence PASS; timed human review NOT_RUN |

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
| A/B/C logical equivalence | WITHDRAWN / RECHECK_REQUIRED | Current file readback found an A/C status-key mismatch (`status` vs `corpus_status`); B's four-file assembly procedure was not committed or reproducibly described |
| Cross-file reference closure | PASS | Source, locator, evidence, claim, domain node and relationship targets resolve to the expected record classes |
| Negative controls | PASS | Independent checker rejected unknown enum, missing required statement, duplicate ID, dangling locator and wrong relationship endpoint kind |
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
