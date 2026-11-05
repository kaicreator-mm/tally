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
6. ReviewDecision is a human audit event bound to exact target ID, record version and a recomputed canonical JSON SHA-256. Schema validation cannot accept knowledge.
7. Every RelationshipPredicate enum value has an explicit URI meaning; RDF export must preserve rdf:subject, rdf:predicate and rdf:object as resource nodes.
7. CoverageGap is first-class so partial evidence or unknown coverage is visible rather than silently omitted.
8. Synthetic fixtures are valid for structural testing but cannot be accepted corpus knowledge.

## Full-model execution still required

The candidate schema and graph validator are not yet qualified. Run the exact commands in the local validation handoff:
- LinkML 1.12.1 schema validate/lint.
- Emit the embedded positive graph fixture and validate it as KnowledgeGraph.
- Execute repository validator positive and negative self-tests.
- Generate JSON Schema and JSON-LD context.
- Convert the complete fixture to Turtle and JSON-LD; verify every reified relationship has URIRef subject/predicate/object and the exact source/evidence identity chain. Predicate enum values must serialize as URIRefs, not literals.
- Run `python validate_knowledge_graph.py --format-compare` for A/B/C full-model serialization equivalence, deterministic rebuild, edit locality and two-writer file-conflict simulation; then run compatible/breaking schema evolution.
- Measure edit locality and two-writer merge conflict behavior; record limitations.

No final LinkML adoption outcome is recorded before those results. No production adoption, runtime dependency, model freeze, Domain Package, DomainForge promotion, main merge, release or deployment is implied.
