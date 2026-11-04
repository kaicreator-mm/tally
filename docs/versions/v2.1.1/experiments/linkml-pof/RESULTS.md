# LinkML Proof-of-fit — Initial Results

**Status:** CLI_EXECUTED_2026-11-05 / RESULTS_RECORDED / NO_ADOPTION_DECISION  
**Test date:** 2026-11-04 (design), 2026-11-05 (LOCAL_BUILDER execution, section 10)  
**Pinned candidate:** LinkML v1.12.1

## 1. What exists

- A schema draft covering Source, SourceLocator, Evidence, DomainNode, KnowledgeClaim, RelationshipAssertion, ClaimBundle and KnowledgeCorpus.
- Three authoring-format fixtures encode the same SWEBOK claim/provenance chain; LinkML-generated RDF graph equivalence PASS (L12), pending independent R2 review.
- A compound uniqueness constraint on source + locator.
- Non-inlined references for source, locator, claim and evidence relationships.
- Controlled vocabularies for source class, locator type, verification state, claim type/status and relationship predicates.

## 2. Static observations

| Area | Current observation | Evidence / status |
|---|---|---|
| Schema and fixture YAML | PyYAML 6.0.3 and LinkML v1.12.1 parsed/validated the schema and all A/B/C fixtures after three execution-discovered repairs | YAML syntax PASS; LinkML metamodel PASS (L01); positive data validation PASS (L02) |
| Identifiers | All A/B/C record IDs and typed class references use canonical se: CURIEs; generated Turtle expands them to the declared base | LinkML validate/convert PASS; duplicate and closure checks require the companion validator |
| Cross-file references | LinkML validate/JSON Schema do not resolve non-inlined cross-file references; the committed validate_relationship_integrity.py companion enforces typed closure | LinkML native resolution gap CONFIRMED; prior checker revision executed on repair HEAD 59862b9 — A/B/C PASS (§10.8); current exact-class hardening awaits rerun (§10.9) |
| Compound uniqueness | SourceLocator has a compound key on source_ref + locator; JSON Schema omits it | Repository companion checker enforces the key; generator limitation CONFIRMED |
| JSON-LD | Context omits nested @type; instance JSON-LD has 48 triples vs 55 in TTL, missing seven rdf:type triples | Generated and measured (L09); limitation CONFIRMED |
| RDF | LinkML v1.12.1 RDF CLI has no --diff-stable; default schema/instance outputs were byte-identical across reruns | Generated and measured (L10/L13) |
| Front Matter | Requires and successfully used an extraction adapter before LinkML validation | A fixture conversion PASS (L02/L11) |
| Human review | A/B/C LinkML-generated graphs are semantically equivalent after excluding only container-root wrapper triples; timed reviewer study not run | Graph equivalence PASS (L12); timed human review NOT_RUN |

## 3. Verified tool/documentation constraints

- LinkML v1.12.1 is the latest official release observed on 2026-11-04.
- The JSON Schema generator documentation states that `unique_keys` are not yet supported in JSON Schema; identifier/key dictionary forms provide only a limited uniqueness mechanism.
- The JSON-LD context documentation states that `@type` directives are not emitted at all levels.
- The LinkML v1.12.1 RDF CLI did not expose `--diff-stable`; default outputs were byte-stable in this experiment.

These are documented limitations, not failures observed from running this particular schema.

## 4. Historical pre-builder status — superseded by §10

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

This was the pre-builder status on 2026-11-04. The pinned CLI was subsequently installed and executed on Windows-native Python 3.14.7; see §10.

## 5. Interim position

No production adoption decision is justified. The narrow LinkML PoF has now been executed, including generated RDF and A/B/C graph comparison. The full knowledge entity model, executable integrity companion and canonical authoring decision remain subject to independent review and SEK-05 #308.


## 6. Static validation actually performed — 2026-11-04

> **Historical snapshot — superseded by §10.** This section and the pre-builder `NOT_RUN` / `NOT_PROVEN` statements in §§7–9 describe the state before the Windows-native LinkML v1.12.1 execution and repository-checker LOCAL_VALIDATION. They are retained as chronology, not current status. For current execution status, use §§10.3–10.8: LinkML metamodel and instance conversion PASS; A/B/C graph equivalence PASS; generated `rdf:object` URIRef evidence PASS; repository checker self-test and A/B/C YAML+RDF integrity PASS. The experiment remains **NOT_ADOPTED**; independent Exact-SHA R2 review is the remaining gate.

Environment: Python 3.13; PyYAML 6.0.3; RDFLib 7.5.0.

| Check | Result | Boundary |
|---|---|---|
| YAML parse | PASS | Schema, Source, SourceLocator, Evidence, ClaimBundle, standalone corpus and extracted front matter parsed |
| Duplicate YAML keys | PASS | Custom PyYAML loader found no duplicate mapping keys in those seven YAML documents |
| Schema structural consistency | PASS | All class slots are declared; all ranges resolve to a class/enum/built-in type; all CURIE prefixes are declared; all non-inlined target classes declare the identifier slot |
| Required slots and enum values | PASS | All positive fixture objects satisfy the independent static checks |
| A/B/C logical equivalence | NOT_PROVEN | A and C now use the same `corpus_status` key; B's split files still require reproducible assembly, LinkML conversion and graph merge/isomorphism |
| Cross-file reference closure | LinkML-native resolution NOT_SUPPORTED for these non-inlined references | The builder's external checker passed six negative controls; a repository-contained companion checker is added by the R2 repair and must be rerun on the exact candidate SHA |
| Negative controls | LinkML structural negatives PASS; graph-integrity negatives require the companion checker | Builder evidence reports six fail-closed graph negatives; repository checker self-test and fixture run are the remaining reproducibility gate |
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


## 9. Historical Exact-SHA R2 findings and remediation candidate — 2026-11-05 (superseded by §10.7–§10.8)

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


## 10. LOCAL_BUILDER execution — 2026-11-05

The pinned LinkML v1.12.1 CLI was executed on Windows-native Python in an isolated venv (no WSL). This section supersedes the NOT_RUN items in section 4. Raw command logs, environment capture and all generated artifacts are preserved in the builder evidence bundle referenced by the PR; per-command exit codes are reproduced below.

### 10.1 Environment

- OS: Windows (win32 10.0.26300), Git Bash runner; **no WSL used**.
- Python 3.14.7 (Windows native, `python -m venv`), pip 26.2.1, index pypi.org (direct, no proxy).
- `linkml 1.12.1`, `linkml-runtime 1.12.1` (resolved together), rdflib 7.6.0, pydantic 2.14.2, pyyaml 6.0.3, click 8.5.0, jsonschema 4.26.0.
- `linkml --version` → 1.12.1; `linkml-convert --version` → 1.12.1.

### 10.2 Defects found in the committed input (fixed inside the #305 write set)

1. **Schema was not parseable YAML at HEAD 88f0050.** The `id` slot description contained an unquoted `se:` (colon-space) in a plain scalar → `mapping values are not allowed here` (line 139, col 86). PyYAML 6.0.3 confirms the identical error independently. Fix: quoted the scalar. The earlier "PyYAML parsed the schema" static claim held for an older revision, not for this HEAD.
2. **`schema:` prefix mismatch hard-failed every generator/converter.** The schema declared `schema: https://schema.org/` while `linkml:types` maps `schema` → `http://schema.org/`; `linkml generate jsonld-context` and every `linkml-convert` invocation died with `ValueError: Prefix: schema mismatch between tally_software_engineering_knowledge_pof and types`. Fix: aligned the prefix to `http://schema.org/` (also clears the linter's canonical_prefixes warning).
3. **Nested record IDs without the `se:` prefix crashed the RDF dumper.** `linkml-convert -t ttl` for KnowledgeCorpus/ClaimBundle roots failed with `ValueError: Unknown CURIE prefix: @base` on the first bare id. The bare-id form passed `linkml validate` and survives in compacted JSON-LD via `@base`, but the rdflib dumper requires well-formed CURIEs — i.e. validation green does not imply conversion green. Fix: normalized all nested record ids in A/B/C fixtures to canonical `se:` CURIEs (matching the schema's `id_prefixes` and every reference form).

All three are fixture/schema authoring defects, not LinkML bugs; they were fixed inside the exact write set and every affected test was rerun from scratch.

### 10.3 Test matrix results (after fixes; per-command exit codes in the log bundle)

| ID | Test | Result |
|---|---|---|
| L01 | `linkml validate <schema>` (metamodel) | **PASS** — No issues found (EXIT=0) |
| L01 | `linkml lint <schema>` | 0 errors / 69 warnings (EXIT=1): 42× missing slot `description`, 26× UPPER_CASE enum naming, 1× canonical prefix (fixed by 10.2-2). Advisory only. |
| L02 | Positive A (front matter extracted by adapter), C standalone, B ×4 split + ClaimBundle | **PASS ×6** — No issues found (EXIT=0) |
| L03 | Invalid claim status / unsupported predicate (`APPROVED`, `CONTRADICTS`) | **Rejected** (EXIT=1, exact enum list in diagnostic) |
| L04 | Missing required `statement` | **Rejected** (EXIT=1, `'statement' is a required property`) |
| L05 | Duplicate source id | `linkml validate`: **accepted** (EXIT=0); generated JSON Schema: accepted; `linkml-convert`: **rejected** — `ValueError: duplicate key` (EXIT=1). Enforcement exists only in the convert/dump path. |
| L06 | Dangling refs (`object_ref→SE-EVD-9999`, `source_ref→SE-SRC-MISSING-0000`) | Accepted by validate, JSON Schema and convert paths (EXIT=0). **No LinkML path resolves non-inlined references**; fail-closed only via the independent rules checker (10.4). |
| L06a | `rdf:object` node kind | **PASS** — in A, C and B-merged: every reified `rdf:Statement` carries `rdf:object`/`rdf:subject`/`rdf:predicate` as **URIRef resource nodes, zero Literals**; `se:` IRIs expand to `https://github.com/kaicreator-mm/tally#…`. R2 P1-01 verified at the generated-graph level. |
| L06b | `RELATIONSHIP_RULES.yaml` endpoints | RDF-level: SUPPORTED_BY→Evidence, INCLUDES→DomainNode, subjects KnowledgeClaim, evidence refs resolve — PASS on A/B/C. Data-level fail-closed checker: **6/6 negatives rejected** (duplicate id, dangling object, dangling source, wrong object class `SUPPORTED_BY→DomainNode`, compound-key duplicate, unknown predicate), each with a one-line cause. |
| L07 | Compound (source_ref, locator) uniqueness | Not enforced by validate, JSON Schema or convert (documented generator gap **confirmed**); enforced by the independent checker. |
| L08 | JSON Schema generation | Generated (19 `$defs`). `Source.id` and `RelationshipAssertion.object_ref` degrade to `{"type": "string"}` — no pattern, no reference semantics; no `uniqueKeys` emitted; enums and required lists work (N1/N3 rejected through jsonschema too). |
| L09 | JSON-LD context generation | Generated. Class entries are `{"@id": …}` only — **no nested `@type` emission** (documented, now observed). Concretely: `-t json-ld` instance export parses to 48 triples vs 55 in TTL; the 7 missing are exactly each record's `rdf:type`. |
| L10 | Schema-level RDF (`generate rdf`) | 828 triples. **Metamodel projection, not OWL** (0 `owl:Class`): 9 `linkml:ClassDefinition`, 41 `linkml:SlotDefinition`, 19 `linkml:TypeDefinition`; `Evidence`/`DomainNode is_a GraphNode` preserved; `linkml:class_uri` annotations carry the intended `…tally#Class` URIs (RelationshipAssertion → `rdf:Statement`). Quirk: element subject IRIs use the schema id with the last path segment replaced (`https://github.com/kaicreator-mm/<Element>`). |
| L11 | `linkml-convert` instance export | A and C → TTL + JSON-LD; B's four records → TTL each; all EXIT=0 after the id normalization (before it, every root-container TTL conversion crashed — see 10.2-3). |
| L12 | B four-file RDF merge; A/B/C graph semantics | **PASS** — A ≅ C full graphs (55 ≡ 55 triples); after excluding only container-root wrapper triples: A ≅ B-merged ≅ C at **45 ≡ 45 ≡ 45** triples. All domain, provenance and relationship triples identical across the three authoring formats. |
| L13 | Repeat generation / determinism | **PASS** — all 10 artifacts (json-schema, context, schema TTL, 6 conversions) byte-identical across reruns. Note: `--diff-stable` **does not exist** in the 1.12.1 RDF generator CLI (`No such option`, EXIT=2); default output is already canonicalized/byte-stable for this schema. |
| L14 | Schema evolution | Evolved copy: add optional `reviewer_note`, deprecate `applicability`. Old data → old/new schema PASS; new data → old schema **rejected** (`Additional properties are not allowed`); new data → new schema PASS; evolved schema passes metamodel. Learnings: LinkML `deprecated` is a **string note, not a boolean** (`True is not of type 'string'` from the metamodel); using a deprecated slot emits **no warning**; JSON Schema evolution surface is a clean 16-line additive diff. |
| L15 | Human diff comparison | Same semantic edit (one added limitation) applied to A/B/C: identical 2-line diffs in all three formats — edit locality equivalent for record-level edits. Format differences are structural (see 10.5), not diff-size. |

### 10.4 Independent validation paths compared (required experiment #10)

1. `linkml validate` — enum/required/type structure only; blind to references, duplicates (list form), uniqueness and endpoint semantics.
2. Generated JSON Schema + `jsonschema 4.26.0` — same coverage class as (1); additionally loses reference semantics (`object_ref` as plain string).
3. Builder evidence-bundle scripts (`rules_check.py` / `rdf_check.py`) — 6/6 graph-negative cases rejected on the executed fixtures.
4. Repository companion `validate_relationship_integrity.py` — executable implementation of the committed `RELATIONSHIP_RULES.yaml`; checks canonical CURIE identity, duplicate IDs, compound keys, cross-record closure, predicate-specific endpoints and RDF URIRef semantics. Its exact-HEAD self-test and A/B/C rerun are recorded separately from the builder's original logs.

**Conclusion:** LinkML alone does not provide graph integrity for this model. The machine-readable rules plus a small repository-contained checker are a necessary companion.

### 10.5 Format comparison (A/B/C) — operational observations

- **A (front matter)** needs an extraction adapter (14-line regex script) before any LinkML operation; narrative lives next to the data; the YAML block is a full C-style corpus inline.
- **B (YAML + Markdown)** validates per record and converts per record; the cross-record reference closure exists only at merge time; assembly procedure is now demonstrated reproducibly (four `linkml-convert` runs + RDF merge).
- **C (standalone YAML)** is the only format whose file alone is both valid and convertible as one unit.
- All three produce the same 45-triple domain graph (L12), so the choice is an authoring-workflow decision, not a semantic one.

### 10.6 Interim verdict for SEK-05

LinkML v1.12.1 handled this narrow provenance chain after three fixture/schema authoring fixes: metamodel-valid, structurally validating, deterministic, byte-stable outputs, and — critically for R2 P1-01 — rdf:object serializes as URIRef resources with canonical se: CURIEs expanded to the declared base. The A/B/C domain graphs are isomorphic. LinkML does not resolve non-inlined cross-record references, JSON Schema omits reference/unique-key semantics, and JSON-LD omits nested @type; these require a companion integrity layer and format-aware handling. This is a narrow PoF result, not adoption of LinkML as the canonical full knowledge model.


## 10.7 R2 P1-02 / P1-03 repair disposition — 2026-11-05

- P1-01 — RDF resource target: LinkML v1.12.1 generated A/C/B Turtle. All reified rdf:object, rdf:subject and rdf:predicate values were URIRefs; zero Literals were observed. se: CURIEs expanded to the declared namespace. Builder evidence: PASS.
- P1-02 — canonical resource identity: PR #310 normalizes every modeled record ID and typed class reference in A/B/C to the same se: CURIE form. A/B/C fixture readback shows no bare modeled record IDs or bare typed class references. The checker resolves each reference to exactly one record of the expected class. Taxonomy/view tokens remain opaque strings because taxonomy resources are outside this narrow PoF and belong to #308.
- P1-03 — executable endpoint enforcement: RELATIONSHIP_RULES.yaml alone is declarative. The prior repository checker revision executed on repair HEAD 59862b9 with 16/16 self-test and A/B/C YAML+RDF PASS (§10.8). A fresh source review then found that additional incompatible RDF class assertions could pass; the checker is hardened in §10.9, and its new 23-case self-test plus A/B/C rerun are REVALIDATION_PENDING.
- A/B/C graph equivalence: PASS, 45 ≡ 45 ≡ 45 triples after excluding only the format-specific root wrapper.
- No adoption: LinkML remains a candidate for SEK-05's full-model decision; this experiment does not freeze the entity catalog or authoring format.

## 10.8 Repository checker LOCAL_VALIDATION — prior checker revision 974c30d — repair HEAD 59862b9 — 2026-11-05 (superseded for current checker by §10.9)

Executed on Windows-native Python 3.14.7 (no WSL) against the exact repair PR #311 HEAD `59862b92ab79e4e382682343c0e3506c9174a9a6`, tree `9ed814c77ae588416b21953858852ee543e802b7`, using prior checker blob `974c30d23374c95433edb74e19c14b824866683c`, reusing the pinned LOCAL_BUILDER venv (linkml/linkml-runtime 1.12.1, rdflib 7.6.0, PyYAML 6.0.3, pydantic 2.14.2, click 8.5.0, jsonschema 4.26.0). Raw command logs, per-command exit codes and all generated artifacts are preserved in the local validation bundle `_tmp/tally-i305-r2-val-zcode/` (logs `01`–`06`, artifacts under `out/`); the exit codes are reproduced below.

| Step | Command (working dir: repo root of the repair worktree) | Exit |
|---|---|---|
| Compile | `python -m py_compile docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py` | 0 |
| Self-test | `python docs/versions/v2.1.1/experiments/linkml-pof/validate_relationship_integrity.py --self-test` | 0 — `SELF-TEST SUMMARY: 16/16 PASS` |
| Regenerate C | `linkml-convert -s schema/software-engineering.yaml -C KnowledgeCorpus fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml -t ttl -o se-corpus-c.ttl` | 0 |
| Extract A front matter | format-A adapter (regex front-matter extraction, `extract_frontmatter.py`) → se-frontmatter-a.yaml | 0 |
| Regenerate A | `linkml-convert -s schema/software-engineering.yaml -C KnowledgeCorpus se-frontmatter-a.yaml -t ttl -o se-corpus-a.ttl` | 0 |
| Regenerate B (×4) | `linkml-convert -C Source\|SourceLocator\|Evidence\|ClaimBundle` on the four split fixtures → se-b-{source,locator,evidence,claim-bundle}.ttl | 0 |
| Merge B | RDFLib graph merge of the four B Turtle files → se-b-merged.ttl (6+5+6+33 = 50 triples, 2 rdf:Statement nodes) | 0 |
| Check A | checker `--frontmatter fixtures/frontmatter/SE-CLM-0001.md --ttl se-corpus-a.ttl` | 0 — YAML PASS + RDF PASS (2 statements) |
| Check B | checker `--bundle ... --source ... --locator ... --evidence ... --ttl se-b-merged.ttl` | 0 — YAML PASS + RDF PASS (2 statements) |
| Check C | checker `--corpus fixtures/standalone-yaml/KNOWLEDGE_CORPUS.yaml --ttl se-corpus-c.ttl` | 0 — YAML PASS + RDF PASS (2 statements) |

Record counts agreed across A/B/C: 1 Source, 1 SourceLocator, 1 Evidence, 1 DomainNode, 1 KnowledgeClaim, 2 RelationshipAssertion (A/C additionally carry the KnowledgeCorpus root, B the ClaimBundle root).

RDF↔YAML identity binding was verified twice. First, the checker's built-in binding: for each graph the set of rdf:Statement identifiers matched the YAML RelationshipAssertion IDs exactly, and each statement's rdf:subject/rdf:predicate/rdf:object/prov:wasDerivedFrom equaled the YAML CURIE identities expanded to `https://github.com/kaicreator-mm/tally#`. Second, an independent readback script compared every statement field of all three graphs against the YAML records: 24/24 comparisons IDENTICAL (3 graphs × 2 statements × subject/predicate/object/evidence), zero mismatches; all endpoint and evidence nodes are URIRefs in the se: namespace. A live fail-closed probe redirected statement `se:SE-REL-POF-0001`'s rdf:object from `se:SE-EVD-0001` to `se:SE-KA-13` in a copy of se-corpus-a.ttl; the checker rejected it with exit 1 (`generated RDF edge does not preserve the exact YAML subject/predicate/object CURIE identity`), confirming the binding is active on generated artifacts, not vacuous.

Determinism cross-check: all six regenerated Turtle outputs are byte-identical to the PR #310 builder evidence bundle artifacts, reproducing the L13 byte-stability result on a second run of the pinned venv.

No defects were found in the repair HEAD; no code changes were required by this validation. Independent Exact-SHA R2 review of PR #311 remains the open gate before SEK-05 #308.


## 10.9 Independent R2 hardening — exact RDF class identity and ruleset validation — 2026-11-05

A fresh source-level review of the committed checker found a fail-closed gap not covered by the earlier 16/16 tests: the RDF check required the expected subject class to be present, and checked only the Evidence/DomainNode sibling set for objects. A resource typed as the expected class and an additional incompatible se: class (for example Evidence + KnowledgeClaim) could therefore pass. This contradicted the PR's stated exact-endpoint-class guarantee.

The current checker revision (validate_relationship_integrity.py blob 8de88001484d0c8b1ba54678da0a1f06a34252b8; first introduced at 9ef26c68d20515ac4f967e6cece3bfee6a405c9e; current repair PR #311 tip 280ca4a4826d99b1ee7c45f99929cb41b781d290, tree 855707aa25bc71a31556e5a48e19dda3c3f6e9e1) now:

- validates the complete rules document shape, exact required predicate set, subject/object class mapping, object_slot, boolean evidence requirement, complete fail-closed list and RDF expectations; duplicate, unknown and malformed rules are rejected;
- requires the exact expected se: class set for relationship subjects, objects and evidence targets, permitting only the declared abstract GraphNode ancestor for Evidence/DomainNode;
- adds negative tests for wrong subject class, incompatible additional subject/object classes, duplicate/unknown/malformed rules; the self-test now requires 23/23 checks and returns non-zero if the count is incomplete.

**Current status: REVALIDATION_PENDING.** The previous 16/16 run in §10.8 applies only to checker blob 974c30d...; it is not evidence for the current checker blob 8de8800.... Before closing P1-03, rerun py_compile, --self-test (23/23), A/B/C YAML+generated-Turtle checks and the relevant negative probes on the exact current repair HEAD. Also generate JSON-LD for A, C and B's four split fixtures; parse each as RDF and assert every reified rdf:subject/rdf:predicate/rdf:object is a URIRef, rdf:object is never a Literal, and each edge identity matches the YAML CURIEs. L09's seven missing nested rdf:type triples remain a known JSON-LD context limitation: do not claim JSON-LD/TTL graph equivalence or predicate class validation from JSON-LD when those type triples are absent. Until the explicit JSON-LD edge check passes, report P1-01 as TURTLE_PASS / JSONLD_NOT_VERIFIED. No LinkML schema, fixture, runtime, CI workflow or out-of-write-set path was changed by this hardening.


## 10.10 Additional fail-closed RDF type-node hardening — 2026-11-05

A further independent source review found that the exact-class helper filtered non-URIRef values from `rdf:type` rather than rejecting them. An RDF endpoint could therefore carry the required `se:` class plus a malformed Literal class assertion without being rejected. The checker now rejects every non-URIRef `rdf:type` object before comparing the allowed `se:` class set.

- Updated checker blob: `8561971c0404c3031e8fc2b0e9f9fd32796218e5`.
- Parent repair HEAD: `1eafccc7e45b579dc4bd587426844c7d86e82d89`, Tree `0029d570e92b2eba746d3abb8383ca219a9ab47c`.
- Self-test now requires **23/23 PASS**, including malformed Literal-valued `rdf:type`.
- Current status remains **REVALIDATION_PENDING**. No result from the previous 16/16 checker or a source-only inspection substitutes for the exact 23/23 Windows-native run.
- JSON-LD edge-node verification is required separately because L09 records seven missing nested `rdf:type` triples. Verify A/C and all four B JSON-LD outputs as RDF: exactly one URIRef subject/predicate/object per statement, no Literal object, and exact YAML CURIE identity. Do not claim JSON-LD/TTL graph equivalence or class validation where the JSON-LD type triples are absent.

No schema, fixture, production runtime, CI workflow or out-of-write-set path changed. This remains an experiment-only repair.
