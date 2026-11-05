# SEK-04 — Method Observation

**Task:** #305  
**Date:** 2026-11-04  
**Status:** EXECUTED / EXPERIMENTAL / NOT_ADOPTED — final checker and LinkML PoF passed at source-bearing HEAD `8257aea33538b96300b91be66e2d22c28c926882`; the current PR tip `e27217ec122a50b31876d71a1cdda780fe681462` was also fully revalidated (32/32, 12/12 conversions, 25/25 probes, seven JSON-LD artifacts). See §18 and Issue #305 comment #6657512368.

> **Current-state note:** The initial environment and static-review findings below are retained as historical method evidence. They are superseded by the Windows-native LinkML execution and final checker validation/review in the later sections. The earlier checker blob `503b370b016616c629f97fb47fc06b0079b574d9` at source-bearing HEAD `2c5c8b3d` is historical. The final checker blob `11c0845d9b1b61ccce1c3ee109b86d0bc6e353b7` passed 32/32 self-tests, 12/12 LinkML conversions, 25/25 CLI probes and seven-artifact JSON-LD edge identity at source-bearing HEAD `8257aea33538b96300b91be66e2d22c28c926882`, then passed a full current-tip revalidation at `e27217ec122a50b31876d71a1cdda780fe681462`; see RESULTS.md §10.16–§10.17 and Issue #305 comments #6656869638 / #6657512368.

## Work performed

- Reviewed current official LinkML schema, inlining, validation, JSON Schema, JSON-LD and RDF documentation.
- Pinned v1.12.1 as the experiment candidate because the official GitHub release page identifies it as the latest release observed on this date.
- Drafted a schema with Source, SourceLocator, Evidence, DomainNode, KnowledgeClaim, RelationshipAssertion, ClaimBundle and KnowledgeCorpus.
- Created equivalent A/B/C fixtures from the real SWEBOK taxonomy claim and its provenance chain.
- Included identifier slots, non-inlined references, a compound source-locator uniqueness constraint, controlled vocabularies and a reified relationship assertion; the first draft's string-valued object endpoint was later found to map to an RDF literal.

## Findings from official documentation

1. LinkML schemas are YAML models whose classes, slots, types and enums define the shape of YAML/JSON data. The schema is not the same thing as the data.
2. Non-inlined references to identified classes serialize by identifier rather than duplicating nested objects. This is promising for source/evidence records shared by multiple claims.
3. JSON Schema generation does not currently implement LinkML `unique_keys` fully. This is a documented generator limitation and must be checked against the intended uniqueness contract.
4. The JSON-LD context generator documents that `@type` directives are not emitted at all levels. The RDF output therefore needs explicit inspection; successful context generation alone is insufficient.
5. RDF generation has an opt-in `--diff-stable` option intended to reduce blank-node label churn. Reproducibility must be tested with and without it.
6. Markdown front matter is not a native LinkML input format in the documented YAML/JSON validation workflow; it needs an extraction step. YAML + Markdown can preserve structured data and narrative separately, while standalone YAML minimizes file count but may make human explanations less comfortable.

## Historical negative and unresolved findings — pre-builder snapshot (superseded by later execution sections)

- `linkml` and `linkml_runtime` are not installed in the current analysis environment.
- A package-install attempt did not produce a usable installation; the cause is not established.
- LinkML metamodel validation, data validation, reference resolution, generator outputs and round-trip tests are NOT_RUN.
- No adoption decision is possible from documentation inspection or static YAML parsing alone.
- Whether LinkML validates dangling non-inlined references across multiple files remains an explicit experiment, not an assumed capability.

## Historical method adjustment — pre-builder snapshot (superseded by later execution sections)

Keep the schema experimental and the CLI status NOT_RUN. Use a bounded local builder only for the exact pinned LinkML commands once the fixtures are ready. Independently validate identifier/reference closure so that a generator's syntactic acceptance is not mistaken for graph integrity.


## Historical static validation and graph projection — 2026-11-04 (superseded by LinkML execution below)

- PyYAML 6.0.3 parsed the schema and all A/B/C fixture YAML documents; a duplicate-key loader found no duplicate keys.
- An independent static checker verified declared class slots, ranges, prefixes, required fields and enum values.
- The earlier A/B/C logical-equivalence PASS was withdrawn after exact readback found a fixture key mismatch and no reproducible B assembly procedure. A/C key was corrected; A/B/C equivalence remains NOT_PROVEN pending rerun.
- Negative controls were rejected by the independent checker: unknown enum, missing required statement, duplicate ID, dangling source-locator reference and wrong relationship endpoint kind.
- A manually constructed RDFLib 7.5.0 graph produced 45 triples; Turtle and JSON-LD serialization round-tripped to an isomorphic graph with 45 triples.

These results establish that the sample's YAML structure, local reference closure and a manually designed RDF projection are coherent. They do **not** establish LinkML metamodel validity, LinkML reference validation, or correctness of LinkML-generated JSON Schema/JSON-LD/RDF. Those remain NOT_RUN until LinkML v1.12.1 CLI is executed.


## Historical Delivery-First review correction — 2026-11-04 (superseded by LinkML execution below)

A current repository readback found that the front-matter fixture used `status`, whereas the schema and standalone fixture use `corpus_status`. The front-matter key has been corrected. The earlier A/B/C equivalence PASS is withdrawn pending re-execution.

The previous experiment plan also conflated schema RDF generation and corpus instance export. Official LinkML documentation distinguishes:
- `linkml generate rdf`: RDF representation of the LinkML schema.
- `linkml-convert -s schema data -t ttl/json-ld`: conversion of instance data into RDF/JSON-LD.

The revised POF plan separates schema-level RDF generation from instance-level graph export and requires the four split B records to be converted and merged before graph comparison. No LinkML CLI or instance-export PASS is claimed yet.


## Historical container toolchain attempt — 2026-11-04 (superseded by Windows-native execution below)

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


## Independent R2 P1-01 — typed RDF relationship target

The independent Exact-SHA R2 review found that `RelationshipAssertion.object_ref` had `range: string` with `slot_uri: rdf:object`. LinkML maps a Type value to an RDF Literal; the separate `object_kind` enum did not turn it into a resource reference.

A repair candidate was applied:
- Introduced abstract `GraphNode`.
- Made `Evidence` and `DomainNode` concrete subclasses of `GraphNode`.
- Changed `object_ref` to `range: GraphNode`, `inlined: false`, `slot_uri: rdf:object`.
- Removed redundant `object_kind`; predicate-specific endpoint constraints are now in `RELATIONSHIP_RULES.yaml`.
- Changed fixture identifiers and typed references to `se:` CURIEs with `uriorcurie`.

Windows-native LinkML/linkml-runtime v1.12.1 execution subsequently verified the abstract-class range, non-inlined reference serialization, CURIE expansion and URIRef resource-node behavior in generated Turtle. A/B/C graph equivalence, exact YAML-to-RDF edge identity, JSON-LD edge identity and fail-closed endpoint validation are recorded in RESULTS.md §10.14. The checker source passed 31/31 self-tests and earlier independent review at source-bearing HEAD `2c5c8b3d`; the final follow-up review found a missing DomainNode source/locator coherence invariant in that checker blob, so the earlier clean disposition is superseded and the repaired checker requires 32/32 revalidation. No production adoption is implied.

## R2 P2 remediation — status reconciliation

- A/B/C equivalence: NOT_PROVEN; removed contradictory earlier PASS wording.
- CS2023 rights: exact Copyright and Reprint Permissions notice added to registry and dossier.
- Native dependency edges: exact live graph materialized and read back; OUTPUT_LEDGER status corrected.
- CI: previous exact HEAD trailer-check PASS; quality remains FAIL for repository formatting/Postgres integration checks. No unrelated Go changes were made.

## LOCAL_BUILDER execution — 2026-11-05

Executed the full POF matrix with the pinned LinkML v1.12.1 on Windows-native Python 3.14.7 (isolated venv, no WSL; PyPI direct). Full results: `experiments/linkml-pof/RESULTS.md` §10. Task learning recorded here:

- **Validation green ≠ conversion green.** `linkml validate` accepted records with bare (non-CURIE) identifiers, while `linkml-convert -t ttl` crashed on the same data (`Unknown CURIE prefix: @base`). Any future harness must run the full validate→convert chain, not stop at validation.
- **One prefix character killed every generator.** Declaring `schema: https://schema.org/` (vs `http://`) conflicts with `linkml:types` and hard-fails `linkml-convert` / jsonld-context with `Prefix: schema mismatch`. The linter warned; only execution made it blocking. Canonical-prefix lint findings deserve P1 attention when generators are in scope.
- **YAML plain scalars containing `prefix:` colons break parsing** (`se:` inside an unquoted description). Quoting rule for any slot description that mentions a CURIE prefix.
- **Reference/duplicate integrity is not LinkML's job** (measured, not assumed): validate and JSON Schema paths accept duplicates/dangling refs; only the convert/dump path rejects duplicate ids (`duplicate key`); nothing resolves cross-record references. `RELATIONSHIP_RULES.yaml` + a small checker (~150 lines) closed all six negative classes fail-closed.
- **Documented limitations all reproduced**: JSON-Schema `unique_keys` absent and `object_ref` degrades to plain string; JSON-LD context omits nested `@type` (instance json-ld measured at 48 vs 55 triples — exactly the 7 `rdf:type` triples missing); `--diff-stable` absent from the 1.12.1 RDF generator CLI while default output was already byte-identical across reruns.
- **Schema-RDF is a metamodel projection, not OWL** (828 triples, `linkml:ClassDefinition` nodes, 0 `owl:Class`); element subject IRIs replace the schema id's last path segment — do not consume schema RDF as the domain ontology.
- **Schema evolution**: adding an optional slot is backward-compatible; unknown slots are rejected closed-world (`Additional properties are not allowed`); LinkML `deprecated` takes a **string** note, not a boolean; using a deprecated slot emits no warning.
- **Effort**: ~3 authoring fixes (all in write set), 0 LinkML code changes; the whole matrix runs in minutes once fixtures are canonical. Reviewer effort concentrated in fixture ID normalization and generator-limitation bookkeeping, not in tool operation.


## R2 review remediation — executable integrity layer (2026-11-05)

The LOCAL_BUILDER run on LinkML/linkml-runtime 1.12.1 established:
- all modeled A/B/C record IDs and typed class references were normalized to canonical se: CURIEs;
- LinkML-generated Turtle uses URIRef resources for rdf:object, rdf:subject and rdf:predicate, with zero Literal relationship endpoints;
- A/B/C merged domain graphs are isomorphic at 45 triples after excluding only container-root wrapper triples;
- LinkML itself accepts dangling non-inlined references and does not enforce predicate-specific endpoint classes or compound-key uniqueness through the tested validate/JSON-Schema paths.

R2 P1-03 therefore requires an executable companion, not only RELATIONSHIP_RULES.yaml. A bounded validate_relationship_integrity.py is added inside the #305 experiment write set. It checks canonical CURIE identity, duplicate IDs, SourceLocator compound keys, reference closure, predicate-specific endpoint classes, required evidence and generated Turtle URIRef semantics. Its exact-SHA local verification gates were executed on Windows-native Python 3.14.7 against repair PR #311 HEAD `59862b92ab79e4e382682343c0e3506c9174a9a6`: py_compile clean, self-test 16/16 PASS, A/B/C Turtle regenerated with linkml-convert 1.12.1 (byte-identical to the PR #310 evidence), B's four graphs merged (50 triples), and all three YAML+RDF checker runs PASS with statement/subject/predicate/object/evidence identities bound exactly to the YAML CURIEs (24/24 independent readback identical; a tampered object endpoint was rejected fail-closed). Full per-command exit codes: RESULTS.md §10.8. This first checker checkpoint is historical and is superseded by the later 31/31 validation at source-bearing HEAD 2c5c8b3d. The final R2 follow-up then repaired DomainNode source/locator coherence; the new 32-case checker requires exact-HEAD revalidation as recorded in the final section.

This remains experiment-only; no production runtime, DomainHarness capability or canonical #308 adoption decision is implied.


## Historical independent Exact-SHA R2 follow-up — strict RDF class identity (2026-11-05; superseded by final validation)

A fresh code review found that the first repository checker revision could accept an RDF resource with the expected se: class plus an additional incompatible se: class, because it checked expected-type presence rather than the complete modeled class set. The checker was hardened in repair PR #311: exact class identity is now enforced for relationship subjects, objects and evidence nodes (with only declared GraphNode ancestry allowed for Evidence/DomainNode); the rules document shape and executable contract are validated; duplicate/unknown/malformed rule documents fail closed. Self-test coverage is expanded from 16 to 22 cases, including wrong subject class and incompatible extra class assertions. The new checker blob is 8de88001484d0c8b1ba54678da0a1f06a34252b8 at HEAD 9ef26c68d20515ac4f967e6cece3bfee6a405c9e. The previous 16/16 result applies to the previous checker blob only. Exact-HEAD local revalidation of the new 22-case checker and A/B/C generated Turtle remains pending; P1-03 is not closed until that run succeeds.


## Historical JSON-LD edge-node validation boundary — 2026-11-05 (superseded by final edge-identity validation)

The LinkML v1.12.1 builder generated JSON-LD for A/C, but the existing L06a URIRef-versus-Literal assertion is explicitly recorded for generated Turtle (A/C/B-merged). L09 reports that JSON-LD omits seven nested rdf:type triples (48 vs 55 TTL triples); this is a known context-generation limitation, not evidence that rdf:object is a Literal. The current exact-HEAD local revalidation must parse A/C and all four B split JSON-LD outputs as RDF and independently assert URIRef rdf:subject/rdf:predicate/rdf:object, zero Literal rdf:object values, and exact YAML CURIE edge identity. Do not claim full JSON-LD/TTL graph equivalence or predicate-specific class validation from JSON-LD where the nested type triples are absent. Until that assertion is recorded, P1-01 is TURTLE_PASS / JSONLD_NOT_VERIFIED.


## Historical R2 hardening — malformed RDF type objects (2026-11-05; superseded by final checker validation)

Independent review found that the endpoint class helper ignored non-URIRef objects of `rdf:type`. A malformed Literal-valued class assertion could coexist with the required `se:` class and pass. The helper now fails closed on any non-URIRef `rdf:type` object, with a deterministic negative self-test. Checker blob `8561971c0404c3031e8fc2b0e9f9fd32796218e5`; self-test target **23/23**. Exact Windows-native local revalidation is pending.

JSON-LD requires a separate edge-node check: LinkML's generated JSON-LD is known to omit seven nested `rdf:type` triples, so the current task must assert URIRef subject/predicate/object and exact YAML edge identity for A/C and B's four JSON-LD outputs, without claiming JSON-LD/TTL graph equivalence or predicate class validation.


## Historical exact local validation and independent R2 follow-up — 2026-11-05 (superseded by final 31-case validation)

The Windows-native local run passed the prior checker blob `8561971c0404c3031e8fc2b0e9f9fd32796218e5` at PR #311 HEAD `bc9d3f55309651eed0d5dd0f5ecd0c004c8f8035`: py_compile EXIT=0, 23/23 self-test, A/B/C Turtle, five negative probes and separate JSON-LD edge identity checks all passed. L09's nested JSON-LD rdf:type omissions remain explicitly documented; no full JSON-LD/TTL equivalence is claimed. Raw evidence and SHA256 manifest are linked in Issue #305.

Independent R2 then found three checker gaps: foreign-namespace rdf:type assertions were filtered out on endpoints; statement nodes were not checked for complete rdf:type identity; and ruleset contract comparison accepted duplicate fail_closed entries and integer 1 for boolean true.

Repair completed in the same #305 write set:
- Repair commit `148b656c39a44ff60c196438202521413b79c87c`.
- Checker blob `4d26401bbab6ce3e43778f7096f265ff5424eb22`.
- Self-test target **30/30**. New checks cover foreign endpoint classes, statement-node type identity, unique fail_closed lists and strict RDF expectation types.
- Exact Windows-native rerun of this new checker is pending; prior 23/23 evidence does not transfer to this code.
- No schema, fixtures, production runtime or CI workflow changed.


## Historical RDF reification completeness repair — 2026-11-05 (superseded by final validation below)

A further independent source pass found that an extra node carrying `rdf:subject`, `rdf:predicate` or `rdf:object` without `rdf:type rdf:Statement` could be ignored. The checker now enumerates all reification edge nodes and rejects untyped extras.

- Repair commit: `5c0966de7f42b8dbe71c87994732177d4cd43846`.
- Checker blob: `503b370b016616c629f97fb47fc06b0079b574d9`.
- Self-test target: **31/31**.
- At this repair checkpoint, exact Windows-native rerun was pending; the 31-case checker subsequently passed at source-bearing HEAD `2c5c8b3dfe877e4e9dc335d3c6d50b4440359ac3`, Tree `174c3512337b60c1083f8d75882c88cbcfe8659b`. See the final validation record below.


## Final exact-source-HEAD LOCAL_VALIDATION and independent review — 2026-11-05

The final checker blob `503b370b016616c629f97fb47fc06b0079b574d9` was validated on Windows-native Python 3.14.7 (no WSL), pinned LinkML/linkml-runtime 1.12.1, RDFLib 7.6.0, PyYAML 6.0.3, pydantic 2.14.2, click 8.5.0 and jsonschema 4.26.0.

Exact source-bearing binding:
- HEAD `2c5c8b3dfe877e4e9dc335d3c6d50b4440359ac3`
- Tree `174c3512337b60c1083f8d75882c88cbcfe8659b`
- Base / PR #310 HEAD `41722b8b024760fd61d56d312b2916a3bd6c66d2`

Results: py_compile EXIT=0; self-test 31/31 PASS; 12/12 LinkML conversions EXIT=0; A/B/C YAML+Turtle checker PASS with exact statement/subject/predicate/object/evidence CURIE identity; 24/24 CLI negative/positive probes PASS; independent JSON-LD edge identity PASS across all seven generated artifacts. L09's missing nested rdf:type triples remain explicit and no JSON-LD/TTL full-graph equivalence is claimed.

Raw stdout/stderr, per-command exit codes, generated artifacts and SHA256 manifest are preserved in the evidence bundle linked from Issue #305 LOCAL_VALIDATION comment #6643352340. The earlier independent source review of this same source-bearing HEAD reported P0=0 / P1=0 / P2=0; the final follow-up finding in §183 supersedes that clean disposition.

The earlier documentation reconciliation was docs-only and remained inside the original four-path #305 write set. The subsequent final R2 repair also changes the checker within that same write set; exact-HEAD 32-case validation is now required before integration. No LinkML adoption, merge to main, release or deployment is implied.


## Final independent R2 follow-up — DomainNode source/locator coherence — 2026-11-05

The checker previously required DomainNode.source_ref to resolve to a Source and DomainNode.source_locator_ref to resolve to a SourceLocator, but did not verify that the SourceLocator's own source_ref equals the DomainNode's source_ref. This permits a provenance mismatch even when both references are individually valid.

The checker now resolves the locator record and rejects a mismatch between the two canonical source CURIEs. A dedicated negative self-test covers this case; the required self-test total is 32/32. This is a bounded #305 integrity repair, not a schema or production-model change. The exact-HEAD Windows-native validation subsequently passed at HEAD `8257aea33538b96300b91be66e2d22c28c926882` / Tree `102d6a6ee41b08065ecca8cd0750540a3784962c` / checker blob `11c0845d9b1b61ccce1c3ee109b86d0bc6e353b7`; see the final validation record below. The prior 31/31 evidence applies only to its previous checker blob.


## Final LOCAL_VALIDATION — source/locator coherence repair — 2026-11-05

**Result: PASS** — Issue #305 receipt [#6656869638](https://github.com/kaicreator-mm/tally/issues/305#issuecomment-6656869638).

- Exact HEAD: `8257aea33538b96300b91be66e2d22c28c926882`
- Exact Tree: `102d6a6ee41b08065ecca8cd0750540a3784962c`
- Checker blob: `11c0845d9b1b61ccce1c3ee109b86d0bc6e353b7`
- Base / PR #310 HEAD: `41722b8b024760fd61d56d312b2916a3bd6c66d2`
- Environment: Windows-native Python 3.14.7 (no WSL), LinkML/linkml-runtime 1.12.1, RDFLib 7.6.0, PyYAML 6.0.3, pydantic 2.14.2, click 8.5.0, jsonschema 4.26.0, linkml-convert 1.12.1.
- `py_compile` EXIT=0; checker self-test **32/32 PASS**.
- LinkML conversions **12/12 PASS**; A/B/C YAML + generated Turtle integrity PASS; exactly two relationship statements per graph and exact YAML/RDF statement, subject, predicate, object and evidence CURIE identity.
- CLI probes **25/25 PASS**: 24 negative cases exit 1 with expected diagnostics, including the new cross-source DomainNode locator mismatch; one positive GraphNode-ancestor control exits 0.
- Independent JSON-LD edge identity **PASS across seven artifacts**: URIRef subject/predicate/object, zero Literal rdf:object, exact YAML edge/evidence identity. Known L09 nested rdf:type omissions remain explicit; no full JSON-LD/TTL graph equivalence is claimed.
- Raw logs, generated artifacts, environment versions and SHA256 manifest are preserved in the [57-file evidence bundle](https://gist.github.com/kaicreator-mm/45dd6427fbc2146fb35daf7ac6051ccf).

This receipt supersedes the earlier 31/31 result for the prior checker blob and the pending status recorded when the source/locator repair was first committed. The validation was read-only in a detached worktree and made no repository changes. The later documentation-only reconciliation commit was also fully revalidated at current tip `e27217ec122a50b31876d71a1cdda780fe681462`; see §18 and Issue #305 comment #6657512368. LinkML remains EXPERIMENTAL / NOT_ADOPTED.


## Current-tip final validation rebind — 2026-11-05

The current PR #311 tip was fully revalidated after the final documentation reconciliation.

- HEAD: `e27217ec122a50b31876d71a1cdda780fe681462`
- Tree: `eec4341820c557abf5918e1fc45db6acb3c06436`
- Checker blob: `11c0845d9b1b61ccce1c3ee109b86d0bc6e353b7`
- Result: py_compile PASS; self-test 32/32; LinkML conversions 12/12; A/B/C YAML+Turtle PASS; CLI 25/25; seven JSON-LD artifacts PASS.
- Current-tip receipt: Issue #305 comment [#6657512368](https://github.com/kaicreator-mm/tally/issues/305#issuecomment-6657512368).

This supersedes the pending-rebind wording in earlier chronology. LinkML remains experimental and not adopted.
