# Canonical Authoring Format — SEK-05 Candidate

**Status:** FORMAT_B_CANDIDATE / FULL_MODEL_HUMAN_DIFF_VALIDATION_PENDING  
**Task:** SEK-05 #308  
**Baseline:** PR #303 HEAD e146b1894e6bdf5f41faf18816627e117d44db07

## Candidate: structured YAML records paired with Markdown explanations

Use one canonical structured YAML record per entity (or a small, cohesive record family). Use Markdown only for narrative explanation that cannot be represented as structured fields: rationale, interpretation, review notes, decision history and human-readable context.

The YAML record is the source of truth for identity, typed fields, lifecycle, provenance references and relationships. Markdown must link to record IDs and must not duplicate editable copies of structured facts.

### Proposed logical layout

~~~text
knowledge/
  domain/
    DOMAIN-SOFTWARE-ENGINEERING.yaml
    subdomains/
  taxonomy/
    views/
    nodes/
  authority/
    sources/
    locators/
  evidence/
  concepts/
  claims/
  principles/
  practices/
  patterns/
  anti-patterns/
  decision-models/
  cases/
  counterexamples/
  reviews/
  coverage-gaps/
  relationships/
  explanations/
    <record-id>.md
  generated/
    graph.ttl
    graph.jsonld
    manifest.sha256
~~~

This is a proposed future layout, not a write-set expansion or a directory migration in SEK-05. The current v2.1.1 vertical slice remains intact and is treated as an input fixture.

## Format comparison

| Criterion | A: Markdown + front matter | B: YAML + Markdown | C: standalone YAML |
|---|---|---|---|
| Machine validation | Requires front-matter extraction adapter | Direct LinkML/validator input | Direct LinkML/validator input |
| Human narrative | Strong local context | Strong, linked explanation | Weak unless comments/parallel docs added |
| Structured diff | Good for one record; mixed prose/metadata | Small and focused | Can become broad for bundled data |
| Cross-record references | Harder to inspect across embedded front matter | Explicit IDs and files | Explicit IDs, but large file churn possible |
| Merge conflicts | Same file mixes prose and data | Usually record-local | More likely when one large corpus file is shared |
| Duplicate-fact risk | Higher if prose repeats front matter | Controlled by YAML-authoritative rule | Low if no parallel prose; weaker explanatory context |
| Review audience | Writers comfortable with Markdown | Domain reviewers and tooling | Primarily machine-oriented contributors |
| Export | Requires extraction adapter then schema conversion | Direct conversion | Direct conversion |

SEK-04 tested a narrow A/B/C slice. It proved serialization and conversion behavior for that slice only. The full-model comparison must re-run the same semantic fixture through all three forms; the table above is a hypothesis to be measured, not an observed benchmark.

## Authoring rules

1. Every YAML record has canonical id, record_version, lifecycle_status and created_on.
2. Use canonical se: CURIE references. Never copy the referenced entity's title, source metadata or definition into a second record.
3. Store a source fact once: Source → SourceLocator → Evidence. Knowledge records reference Evidence.
4. A record may have one linked Markdown explanation. The explanation links by canonical ID and is not another data store.
5. Multi-view classification is represented by separate TaxonomyNode records, one per view. Cross-view relationships are typed RelationshipAssertion records.
6. Relationship assertions are first-class YAML records with their own identity, version, evidence and review state.
7. Synthetic fixtures carry synthetic=true or EvidenceType=SYNTHETIC_FIXTURE and cannot be accepted knowledge.
8. No record becomes ACCEPTED because LinkML, the graph validator or CI passed. Acceptance requires a human ReviewDecision bound to an exact record revision and digest.
9. Conflicts and counterexamples are retained as separate records; never overwrite or average them away.
10. Generated TTL/JSON-LD and indexes are derived artifacts. Do not hand-edit them or use them as the canonical source.

## Review and change workflow

- One-record changes should normally touch one YAML record and, if needed, one linked explanation.
- Multi-record changes include explicit RelationshipAssertion updates and preserve a small, reviewable diff.
- Schema changes increment the schema version and include compatible and breaking migration fixtures.
- Reviewers inspect both the structured record and its explanation, then create a ReviewDecision only after source, locator, applicability, rights and contradictions are checked.
- A reviewer must be able to trace every accepted claim to evidence, locator and source without searching prose.

## Reproducible format comparison

The repository validator provides `python validate_knowledge_graph.py --format-compare`. It serializes the same logical graph as:
- A: one Markdown document with YAML front matter;
- B: one YAML file per entity plus linked Markdown explanations and a graph root;
- C: one standalone YAML graph document.

The command round-trips all three forms and compares canonical graph SHA-256 digests. It also reports one-record and two-record edit locality, independent-entity two-writer conflicts, same-entity structured-data-versus-narrative conflicts, file counts and deterministic rebuild. A and B both use one file per entity; A combines front matter and narrative in one Markdown file, while B separates authoritative YAML from Markdown explanation. C is one standalone YAML graph document. These are deterministic structural measurements for the declared layouts, not a substitute for a human reviewer measuring comprehension or actual team merge behavior.

## Remaining measurement

The local full-model validation must record:
- A/B/C conversion and graph isomorphism results from `--format-compare`;
- edit diff sizes for one-record and multi-record changes;
- merge-conflict simulation for two independent record edits;
- reviewer comprehension/time estimate, with method and limitations;
- deterministic output hashes;
- any LinkML JSON-LD type/identity loss and the companion validation needed.
