# SEK-05 — Method Observation

**Task:** #308  
**Date:** 2026-11-05  
**Baseline:** PR #303 HEAD e146b1894e6bdf5f41faf18816627e117d44db07 / Tree 1a68d62e4f6e7583e66aeecfd45f7147eee87b06  
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING

## Inputs and dependency evidence

- SEK-00 #300: CLOSED / ACCEPTED; Pilot Charter and scope boundary.
- SEK-01 #301: CLOSED / ACCEPTED; taxonomy remains PROPOSED / PARTIAL, not frozen.
- SEK-02 #302: CLOSED / ACCEPTED; source authority, verification and rights framework.
- SEK-03 #304: CLOSED / ACCEPTED as an evidence slice; 17 locators, 12 evidence records, 12 claims and 12 support relationships. All claim/evidence records remain pending human review.
- SEK-04 #305: CLOSED / ACCEPTED as early PoF; LinkML 1.12.1 narrow-slice result is REPEAT_EXPERIMENT. Full model must be retested.
- Integrated WIP PR #303 exact input: e146b1894e6bdf5f41faf18816627e117d44db07 / Tree 1a68d62e4f6e7583e66aeecfd45f7147eee87b06.
- Native dependency readback: #308 is blocked by #305; #305 is closed. No task is allowed to edit predecessor paths.

## Model decisions embodied in the candidate

1. Separate the domain boundary (Domain/Subdomain), classification lenses (TaxonomyView/TaxonomyNode), authority/provenance (Source/SourceLocator/Evidence), knowledge content, review events and typed graph edges.
2. Do not force all Software Engineering knowledge into one tree. A TaxonomyNode belongs to one view; cross-view relations are explicit typed edges.
3. A knowledge claim is not a concept, principle, practice, pattern or case. They have distinct semantics and required fields.
4. Source identity is not evidence. SourceLocator is an exact inspected location; Evidence is an extracted observation; downstream knowledge references Evidence.
5. RelationshipAssertion is a first-class record with its own ID, revision, lifecycle, context, rationale, evidence and predicate-specific endpoint constraints.
6. ReviewDecision is a human audit event bound to exact target ID, record version and a recomputed canonical JSON SHA-256. YAML date values are normalized before hashing; schema validation cannot accept knowledge.
7. Every RelationshipPredicate enum value has an explicit URI meaning; RDF export must preserve rdf:subject, rdf:predicate and rdf:object as resource nodes.
8. CoverageGap is first-class so partial evidence or unknown coverage is visible rather than silently omitted.
9. Synthetic fixtures are valid for structural testing but cannot be accepted corpus knowledge.

## Full-model execution still required

The candidate schema and graph validator are not yet qualified. Run the exact commands in the local validation handoff:
- LinkML 1.12.1 schema validate/lint.
- Emit the embedded positive graph fixture and validate it as KnowledgeGraph.
- Execute repository validator positive and negative self-tests.
- Generate JSON Schema and JSON-LD context.
- Convert the complete fixture to Turtle and JSON-LD; verify every reified relationship has URIRef subject/predicate/object and the exact source/evidence identity chain. Predicate enum values must serialize as URIRefs, not literals.
- Run `python validate_knowledge_graph.py --format-compare` for A/B/C full-model serialization equivalence, deterministic rebuild, edit locality and two-writer file-conflict simulation; then run compatible/breaking schema evolution.
- Verify accepted Evidence has its own exact ACCEPT ReviewDecision and verified SourceLocator; verify conflicting review outcomes and synthetic cases fail closed.
- Verify ReviewDecision reviewer identity, policy and rationale are non-empty. External identity/event authenticity remains an integrating-host responsibility, not a graph-validator claim.
- Measure edit locality and two-writer merge conflict behavior; record limitations.

No final LinkML adoption outcome is recorded before those results. No production adoption, runtime dependency, model freeze, Domain Package, DomainForge promotion, main merge, release or deployment is implied.


## Independent Exact-SHA source review R2 — 2026-11-06

**Reviewed input:** PR #318 HEAD `23407a082875249de60d699f1d522675fa67376b` / Tree `9dbf57cf07a496eb2bdaa94f011580494e3f9ac5` / Base `e146b1894e6bdf5f41faf18816627e117d44db07`.

### Findings repaired within the original seven-path write set

1. **P1 — inactive ReviewDecision could still authorize or conflict with current knowledge.** The validator previously counted RETRACTED/SUPERSEDED/non-current review events as current decisions. It now treats only ACTIVE ReviewDecision records as effective; non-ACTIVE events remain historical and are checked for valid target-version/digest syntax without binding the current record. Added negative/positive coverage for revoked acceptance, inactive rejection and stale historical digest.
2. **P2 — Format B explanation integrity was not actually checked.** The parser previously ignored the Markdown explanation collection, allowing missing, orphaned or wrong-ID explanations to pass. It now enforces the one-to-one record/explanation mapping and canonical-ID heading; added three negative tests.
3. **P2 — Entity Catalog and LinkML schema disagreed on Case fields.** The schema requires `problem` for Case and inherited Counterexample, but the catalog omitted it. The catalog required-field lists now match the schema.

### Review status

```ini
SOURCE_REVIEW=REPAIRED_IN_SCOPE
P0=0
P1=0
P2=0
SELF_TEST_EXPECTED=35
WINDOWS_NATIVE_VALIDATION=PENDING
LINKML_FULL_MODEL_VALIDATION=PENDING
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```

The source repairs are not a substitute for native Windows execution at the resulting exact HEAD/Tree. No LinkML adoption decision or production promotion is made here.


## Post-R2 source hardening — 2026-11-06

**Code snapshot before this documentation update:** HEAD `4a0bdb30777a20f2ecab5f7274c6c26006074105` / Tree `eefaf3ab2aedf95bf3148f842a6f63101c82f210`.

The R2 source review was anchored to its stated earlier exact HEAD. Subsequent in-scope repairs preserve the seven-path Task Pack boundary:

1. Format A snapshots are now represented as the actual per-entity Markdown file map, allowing the A/B/C round-trip and edit-locality comparison to execute.
2. The graph validator rejects malformed root-domain, evidence, source-locator and relationship endpoint types with explicit validation errors rather than unhandled hash/type errors.
3. Added negative fixtures for malformed references. Together with the R2 review-lifecycle and Format B explanation-link fixtures, the current expected repository self-test count is **38**.
4. The missing `Refs: #308` trailer in one intermediate repair commit was corrected by rebuilding that commit and its descendants while preserving their exact trees. The latest trailer-check is SUCCESS.

**Current qualification state:**

```ini
SOURCE_REVIEW_R2=HISTORICAL_EXACT_SHA_PASS_AFTER_IN_SCOPE_REPAIRS
SELF_TEST_EXPECTED=38
WINDOWS_NATIVE_VALIDATION=PENDING
LINKML_FULL_MODEL_VALIDATION=PENDING
RDF_TURTLE_JSONLD_VALIDATION=PENDING
SCHEMA_EVOLUTION_VALIDATION=PENDING
HUMAN_REVIEW_BURDEN=NOT_MEASURED
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```

The exact local validation handoff must bind the final branch HEAD/Tree after this documentation update. No final LinkML adoption decision is recorded until that validation and a fresh independent Exact-SHA review complete.


## Independent Exact-SHA source review R3 — 2026-11-06

**Reviewed input:** PR #318 HEAD `fb7020430b46a73a1d7656932c51fbf6086c1ed8` / Tree `5cb2aeb12fa6c9178f4149bf91878d229fca8181` / Base `e146b1894e6bdf5f41faf18816627e117d44db07`.

### Additional P2 repaired within the original write set

Malformed lifecycle and record-version values could reach set/hash membership before their type was rejected, causing a validator exception instead of a fail-closed graph error. ReviewDecision identity fields and exact review-object bindings now require strings before comparison or set membership. Added negative coverage for list-valued lifecycle, record version and reviewer identity.

```ini
SOURCE_REVIEW=REPAIRED_IN_SCOPE
P0=0
P1=0
P2=0
SELF_TEST_EXPECTED=41
WINDOWS_NATIVE_VALIDATION=PENDING
LINKML_FULL_MODEL_VALIDATION=PENDING
```


## Independent Exact-SHA source review R4 — 2026-11-06

**Reviewed input:** PR #318 HEAD `f2c91bac48acbcd215e2215f06e606e641681fac` / Tree `f217df290c4905e3443c445b9ecaf1fada5be692` / Base `e146b1894e6bdf5f41faf18816627e117d44db07`.

### Additional P2 findings repaired within the original seven-path write set

1. **Malformed relationship lifecycle could escape validation.** A list-valued lifecycle on a PART_OF/SUPERSEDES edge reached set membership and raised TypeError. The cycle detector now only evaluates string lifecycle values; the common-field validator reports the invalid type.
2. **Accepted Pattern case-version binding could raise TypeError.** A malformed case record_version was inserted into a set-membership tuple without type validation. The validator now checks semantic-version syntax before binding the case review.
3. **Historical review digest accepted numeric coercion.** A 64-digit integer could stringify to a SHA-256-shaped value. Review target versions and digests now require actual strings before syntax validation.
4. **Format B parser had malformed-input escape paths.** It now rejects unhashable/non-string collection names, missing graph-root fields, noncanonical IDs, mismatched record paths, duplicate YAML keys, orphan records and unexpected paths with explicit ValidationError codes.

Added eleven fail-closed regression cases across graph lifecycle/review binding and Format B parsing.

```ini
SOURCE_REVIEW_R4=REPAIRED_IN_SCOPE
P0=0
P1=0
P2=0
SELF_TEST_EXPECTED=52
WINDOWS_NATIVE_VALIDATION=PENDING
LINKML_FULL_MODEL_VALIDATION=PENDING
RDF_TURTLE_JSONLD_VALIDATION=PENDING
SCHEMA_EVOLUTION_VALIDATION=PENDING
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```


## Independent Exact-SHA source review R5 — 2026-11-06

**Reviewed input:** PR #318 HEAD `c33eba0b7f262e2560dea9ac6b7ce8e163be3cb5` / Tree `3140774ca233731f42540060059127fccba3d253` / Base `e146b1894e6bdf5f41faf18816627e117d44db07`.

### P2 finding repaired within the original seven-path write set

The Entity Catalog's class-level `required_fields` were stricter than the LinkML schema for Pattern.context, Case.context (inherited by Counterexample), RelationshipAssertion.context, and AntiPattern.symptoms/causes/consequences. Added class-level `slot_usage` requirements, preserving the optional nature of shared slots for unrelated entity classes. Bumped the experimental schema/model version from 0.2.0 to 0.3.0 because the added required constraints are breaking for prior candidate instances.

The positive fixture already supplies these fields. Native LinkML validation must additionally prove each omission is rejected and that unrelated classes can still omit optional context/consequences.

```ini
SOURCE_REVIEW_R5=REPAIRED_IN_SCOPE
P0=0
P1=0
P2=0
SCHEMA_VERSION=0.3.0
SELF_TEST_EXPECTED=52
WINDOWS_NATIVE_VALIDATION=PENDING
LINKML_REQUIRED_FIELD_NEGATIVES=PENDING
RDF_TURTLE_JSONLD_VALIDATION=PENDING
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```
