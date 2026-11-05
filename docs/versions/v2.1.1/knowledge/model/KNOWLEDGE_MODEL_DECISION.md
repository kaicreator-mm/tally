# SEK-05 — Knowledge Model and Authoring Decision

**Task:** [SEK-05 #308](https://github.com/kaicreator-mm/tally/issues/308)  
**Parent:** [v2.1.1 Product/Knowledge Pilot #299](https://github.com/kaicreator-mm/tally/issues/299)  
**Baseline:** PR #303 HEAD e146b1894e6bdf5f41faf18816627e117d44db07 / Tree 1a68d62e4f6e7583e66aeecfd45f7147eee87b06  
**Status:** EXPERIMENTAL_CANDIDATE / INDEPENDENT_REVIEW_PENDING / NO_PRODUCTION_ADOPTION

**Model version:** 0.3.0

## Decision summary

This file defines the full-model candidate and the decision gates. It does not claim the final LinkML outcome before the full model has been executed and independently reviewed.

- **Canonical authoring candidate:** Format B — structured YAML records paired with Markdown explanations.
- **Schema candidate:** LinkML 1.12.1 with amendments: a repository-contained deterministic graph-integrity validator, explicit relationship catalog, and explicit RDF/JSON-LD identity checks.
- **Proposed decision:** REPEAT_EXPERIMENT. Structural and machine-validation evidence is available, but human review effort has not been measured and the resulting exact revision still requires independent review. Format B is the selected experimental authoring direction for that follow-up, not a production adoption.
- **Production boundary:** no runtime dependency, production ontology, Tally UI integration, Domain Package, DomainForge promotion, official version freeze, release or deployment is authorized by this experiment.

## Why Format B is the leading candidate

Format B stores one canonical structured record per entity (or a bounded record family) in YAML and links it to a human-readable Markdown explanation when narrative is material. It separates machine-validated facts from rationale and review context without forcing every record into a single large YAML document or a Markdown front-matter parser.

The choice is provisional until the full-model local comparison:
- A (Markdown + YAML front matter) has good narrative locality but couples parser behavior to front-matter conventions and makes cross-record references less visible in a single-file review.
- B (structured YAML + Markdown) gives explicit record identity, small diffs, clear file ownership and direct schema validation; narrative can change without rewriting structured identity/provenance.
- C (standalone YAML) is deterministic and machine-friendly, but large collections create broad diffs and are less approachable for domain reviewers.

The A/B/C comparison must measure real record edits, multi-entity changes, merge conflicts, review time and deterministic exports, not just compare syntax aesthetics.

## Model design

The model separates:
1. **Domain boundary:** Domain and Subdomain.
2. **Classification:** TaxonomyView and TaxonomyNode. Every node belongs to one view; cross-view relationships are explicit and typed. This prevents discipline, curriculum, lifecycle, quality, architecture and security views from being collapsed into one tree.
3. **Authority/provenance:** Source → SourceLocator → Evidence. Downstream knowledge records reference Evidence; they do not duplicate source metadata or page citations.
4. **Knowledge content:** Concept, KnowledgeClaim, Principle, Practice, Pattern, AntiPattern, DecisionModel, Case and Counterexample.
5. **Governance:** ReviewDecision and CoverageGap.
6. **Graph semantics:** RelationshipAssertion is a first-class, directed, versioned entity governed by RELATIONSHIP_CATALOG.yaml.

All records have a stable canonical se: CURIE, record_version, lifecycle_status and created_on. Aliases are non-canonical lookup hints, never graph endpoints. Record revisions keep identity; changed conceptual identity creates a new ID. Accepted knowledge requires evidence closure: each Evidence record must itself be ACCEPTED, have extraction_status=REVIEWED, point to a verified SourceLocator and carry an exact ACCEPT ReviewDecision digest. The accepted knowledge record also has its own ReviewDecision. Conflicting ACCEPT versus REJECT/REQUEST_CHANGES decisions on the same exact revision fail closed. Synthetic fixtures can validate structure but can never be accepted knowledge.

## Required validation profile

| Gate | Required evidence |
|---|---|
| Schema/metamodel | LinkML 1.12.1 schema validate/lint and generated JSON Schema/context |
| Entity coverage | Every required entity family instantiated by positive fixture and represented in catalog |
| Positive/negative fixtures | Required fields, enums, aliases, IDs, class references, synthetic restrictions and lifecycle gates |
| Graph integrity | Duplicate IDs/aliases, dangling references, wrong endpoint class, unsupported predicate, missing evidence/qualifier and cycles fail closed |
| A/B/C equivalence | Same logical records and relationship assertions; only format-specific container wrappers may differ |
| RDF/JSON-LD | Record identity and typed subject/predicate/object resource nodes preserved; source/evidence chain preserved; no false claim of full graph equivalence when converter omits nested types |
| Evolution | Optional compatible field addition and breaking rename/removal with old/new fixture behavior documented |
| Human review | Edit locality, review diff noise, conflict surface and approximate author/reviewer effort recorded |
| Rights | No ISO normative content or substantial copyrighted source text ingested without permission |

## Known risks and mitigations

- LinkML reference closure is not a substitute for a graph validator. Mitigation: validate all IDs and typed endpoints in validate_knowledge_graph.py.
- YAML date scalars may load as native date objects; canonical review digests normalize dates to ISO-8601 before JSON hashing so the same record hashes consistently across YAML and JSON representations.
- The graph validator checks that reviewer identity, policy and rationale are non-empty and binds each decision to an exact target digest. It cannot independently authenticate that identity or verify the external review event; an integrating host must verify the reviewer against its authoritative review system before production adoption.
- LinkML enum values without a `meaning` serialize as RDF literals. Mitigation: every RelationshipPredicate permissible value maps to a stable `se:` URI so `rdf:predicate` is a URIRef; verify this in both Turtle and JSON-LD.
- JSON Schema generation may not preserve LinkML unique-key/reference semantics. Mitigation: executable graph validator is authoritative for graph-wide invariants.
- LinkML 1.12.1 JSON-LD conversion previously omitted some nested rdf:type triples (SEK-04 L09). This must be retested with the full model; do not declare RDF/JSON-LD parity from Turtle alone.
- A full entity catalog can encourage premature ontology freezing. Mitigation: keep the schema experimental, lifecycle statuses explicit and CoverageGap first-class.
- Evidence extraction is not acceptance. Mitigation: human ReviewDecision bound to target version and SHA-256; machine validation cannot accept claims.

## Final decision rubric

- **ADOPT:** LinkML and the selected authoring format satisfy all material gates without a companion semantic layer beyond routine data checks.
- **ADOPT_WITH_AMENDMENTS:** LinkML is fit when paired with the bounded graph validator, controlled relationship catalog and explicit export profile; the amendments are documented and reproducible.
- **REPEAT_EXPERIMENT:** a material uncertainty remains in schema coverage, RDF/JSON-LD identity, migration, authoring burden or human review cost.
- **REJECT:** LinkML causes material semantic loss or disproportionate burden and a tested proportionate alternative performs better.

A final outcome may only be recorded after exact local execution and independent Exact-SHA review. No outcome implies production adoption or DomainForge promotion.


## R5 schema/catalog required-field conformance — 2026-11-06

A fresh source comparison of ENTITY_CATALOG.yaml against the LinkML class/slot declarations found four required-field gaps: Pattern.context, Case.context (inherited by Counterexample), RelationshipAssertion.context, and AntiPattern.symptoms/causes/consequences. The catalog was already explicit about these fields, but the LinkML schema did not enforce all of them.

The schema candidate is now 0.3.0 and uses class-level `slot_usage` to enforce those requirements without making the shared context/consequences slots globally required for unrelated classes. This is a breaking constraint refinement to the experimental schema; the full-model validation must include negative instances omitting each required field. The candidate remains experimental and unadopted.

## SEK-05 lint and RDF export profile

LinkML 1.12.1's default recommended linter expects lowercase permissible values. This model deliberately uses uppercase controlled values already referenced by the entity and relationship catalogs, fixtures, and RDF predicate meanings; changing their spelling would change the data contract. The schema now includes descriptions for the previously undocumented slots and enums. Use the following explicit profile, which retains the 1.12.1 recommended rules and selects uppercase permissible values; it does not disable metamodel errors or suppress warnings. Save it as a temporary `sek05-lint.yaml`, then run `linkml-lint -c <temporary-path>/sek05-lint.yaml schema/software-engineering.yaml` from the model directory.

```yaml
rules:
  recommended:
    level: warning
  no_xsd_int_type:
    level: error
  no_invalid_slot_usage:
    level: error
  no_undeclared_slots:
    level: error
  no_undeclared_ranges:
    level: error
  one_identifier_per_class:
    level: error
  one_key_per_class:
    level: error
  root_type_checks:
    level: error
  standard_naming:
    level: warning
    permissible_values_upper_case: true
  canonical_prefixes:
    level: warning
```

The default, unconfigured lint still reports uppercase enum values as naming warnings; report those separately rather than treating default lint as clean. `CoverageGap.label` is an optional slot, matching existing fixtures without imposing a new required field on older records.

Direct LinkML 1.12.1 JSON-LD output must not be represented as graph-equivalent to its Turtle output: the full-model experiment found nested `rdf:type` triples missing in the direct JSON-LD. For graph-equivalent JSON-LD, convert the validated instance to Turtle with LinkML, then serialize that RDF graph to JSON-LD with RDFLib. For example, with both paths outside the repository, run `python -c "from rdflib import Graph; Graph().parse('fixture.ttl', format='turtle').serialize(destination='fixture-rdf.jsonld', format='json-ld')"`. Compare the parsed RDF graphs and separately check relationship URIRefs and source-to-evidence closure. The directly generated JSON-LD remains useful as a diagnostic artifact; it is not the canonical RDF-equivalent export.

### Full-model local observations

- The graph validator passed 52/52 self-tests; the positive fixture passed LinkML schema and instance validation. The seven class-level required-field omissions were rejected, while Evidence.context and Pattern.consequences remained optional.
- Formats A, B and C round-tripped to the same canonical graph digest. Their file counts were 53, 105 and 1 respectively; independent-record edit conflicts were 0, 0 and 1. Data-versus-narrative edits to one entity conflicted in A but not B; C has no separate narrative channel. These are deterministic layout measurements, not a human comprehension or time study.
- Turtle contained 666 triples and 16 typed RDF statements. The direct LinkML JSON-LD preserved the subject/predicate/object URIRefs and source-to-evidence identity closure, but contained 614 triples and no nested statement types. RDFLib JSON-LD serialized from Turtle preserved all 666 triples and was graph-isomorphic to Turtle.
- A 0.2.0 fixture without the new required fields was accepted by 0.2.0 and rejected by 0.3.0 until explicitly migrated using the original fixture values. A temporary optional-field addition accepted old records, while old closed schemas rejected the new field; a temporary rename required explicit migration in both directions.
- Reviewer comprehension and elapsed review time were not measured. Independent review must bind the final commit and assess the profile, the JSON-LD limitation, and the proposed `REPEAT_EXPERIMENT` outcome before downstream use.


## Independent Exact-SHA R7 review — 2026-11-06

Reviewed exact candidate HEAD `64b65e34f06b7f2b77fbc29210499a6c29fd1aef` / Tree `0b8f0cb4d2fa9a9c17252a2ba4d548b5a51a4c44` / Base `e146b1894e6bdf5f41faf18816627e117d44db07` after the Windows-native full-model receipt. The receipt's machine results are retained; they are not treated as independent source approval.

R7 found and repaired within the original seven-path write set:
- PART_OF now requires both TaxonomyNode endpoints to share the same TaxonomyView; cross-view mapping must use an explicit mapping predicate.
- Accepted evidence provenance now rejects REJECTED/WITHDRAWN/RETRACTED Source and SourceLocator records and requires Source verification at least IDENTITY_VERIFIED.
- Optional empty claim_refs/case_refs arrays are accepted, while malformed and dangling members remain rejected.
- VALIDATED_BY referencing ReviewDecision now binds the exact active ReviewDecision record ID, target ID, version and digest. Any active exact outcome records a review event; it does not imply acceptance. Inactive review records cannot be borrowed through another active event for the same target.
- Format B explanations are optional as the authoring contract states; present explanations remain unique and ID-bound. Format B rejects unexpected graph-root fields and non-text explanations. Format A/C duplicate YAML keys now fail closed.
- Added regression coverage for all of the above. Expected validator self-tests: **63**.

The full-model machine receipt at `64b65e34f06b7f2b77fbc29210499a6c29fd1aef` remains historical. The R7 repair candidate requires a fresh Windows-native exact-HEAD rerun of all validator, LinkML, RDF/JSON-LD and schema-evolution checks. HUMAN_REVIEW_BURDEN remains NOT_MEASURED; the candidate remains experimental and the proposed decision remains REPEAT_EXPERIMENT. No model freeze, production adoption, merge, release, deployment, Domain Package or DomainForge promotion.
