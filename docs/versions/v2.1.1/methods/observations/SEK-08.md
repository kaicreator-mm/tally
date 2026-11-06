# SEK-08 — Method Observation (Cross-domain integration, graph materialization & human handbook)

**Task:** #349 (SEK-08 of the Tally v2.1.1 pilot #299; predecessors SEK-06 #321 CLOSED,
SEK-07 #333 independently verified)
**Date:** 2026-11-07
**Worktree / branch:** `tally-i349-zcode` / `task/tally-v211-sek08-graph-materialization`
**Baseline (WIP base):** `7a70211b4460e4c753c97798670de9f34861d4a4` (= `wip/v2.1.1-knowledge`
tip, re-read from the workspace mirror and GitHub API before branch creation; worktree clean at start)
**Status:** EXECUTED / EXPERIMENTAL / HUMAN_REVIEW_PENDING
**HUMAN_REVIEW_BURDEN=NOT_MEASURED** (no human review was performed or timed; all 566
knowledge-class records remain PROPOSED/PENDING_REVIEW; 0 ACCEPTED, 0 ReviewDecisions)

Execution standard: ADS #1359 §21 Delivery-First (one continuous LOCAL_BUILDER work
unit; single writer for all graph/export/handbook outputs; no LOCAL_VALIDATION-only
dispatch; consolidated repair by the same writer; independent final verifier required).

## Claim / CAS readback (pre-write discipline)

- Issue #349 re-read in full (`gh issue view 349`): labels `[]`, state OPEN; bounded
  write set = ten lane bundles verbatim (40 paths) + exactly 7 task-owned outputs.
- Task Pack / Claim / CAS / dispatch authority for this stage: **NOT_PRESENT** — no
  claim-file, generation counter or dispatch record exists for SEK-08 in the tree or
  issue; nothing was fabricated. Single-writer ownership is established by this issue's
  "single-writer" clause and honored: one Builder session produced every write on this
  branch; no parallel writer touched the graph/export/handbook outputs.
- Live input refs re-read before integration from the refreshed workspace mirror
  (`refs/pull/*/head`, fetch exit 0) — all ten heads/trees/merge-bases equal the issue's
  declared values (table below); all ten PRs OPEN/unmerged (GitHub API cross-check by
  the initial independent auditor: `INITIAL_AUDIT=PASS`, P0/P1/P2=0).
- SEK-07 lineage receipts verified present on #333: builder cross-lane receipt
  (comment #6693633331) + different-verifier Exact-SHA receipt (comment #6694373230,
  independently rebuilt TREE `36509c5c…`, `CROSS_LANE_FINAL=PASS`), both routing P3-1
  (boundary_notes truncation) and P3-2 (four locator ID overlaps) to SEK-08.

## Inputs (exact tips, all merge-based on `7a70211b…`)

| Lane | Task/PR | HEAD | TREE |
|---|---|---|---|
| 06A Requirements | #322/#327 | `4b4598d02503e34be5094981d089731e031e7954` | `5ed5e1f8ae0a135265af64a3155c4125d612a98b` |
| 06B Architecture & Design | #323/#328 | `d4d7fc80dd011fc78bdaafb64836c98ea15c4c69` | `c9d62e880bc0527cc3bae0a63561221c7cb2ff85` |
| 06C Testing & Verification | #324/#329 | `b2a643e2907981dcd1d000a82687ffa01276e9c8` | `98a66562bab138ad651a3cf56772902224d2681d` |
| 07A Foundations | #334/#344 | `44406564310e950a3075ab6a323aba7d122b15b8` | `cb5df2378e6619348d1e7d83130f6a4a25ec3b2a` |
| 07B Implementation | #335/#346 | `62a2125f66230f018a05243b51196a04bde8e64e` | `9d278df1f8083cc266813db3f4baaee1c005fa53` |
| 07C Security | #336/#341 | `9079c2f6d761715d0089abf9b42e576b41f62f26` | `f8b568b59a76971b55f6058a59916a8129a4198a` |
| 07D Reliability | #337/#345 | `3021a72ac74b99bca29f11b1c918c200829395e3` | `509516aeec594e056762e53c011b4a754e64bad6` |
| 07E Build & Delivery | #338/#343 | `24aeefe07b32ae2f37a442835a628ed8cd2d7ee4` | `ec28516c9c7b46b4a49ae4a08617b393c0324d1b` |
| 07F Maintenance | #339/#342 | `950025db96d1e566e7138c3df2435a090cd1b3a7` | `279959e79625b2b68e5be403a7f2ac6540eda063` |
| 07G Planning & Risk | #340/#347 | `cfba8a2e1946ac8259991a96d594f2d83a03e6eb` | `87c45e6ae30a94e5d2aeb9e90ac25be05321df49` |

## Ten-lane integration (pre-generation gate)

Ten sequential `--no-ff` merges (canonical order 06A→07G) onto the WIP base, all clean
(disjoint four-path write sets). Result verified **before** any generated output was
added:

- `git rev-parse HEAD^{tree}` = `36509c5c51c5286e0e709d6fd912b0ac0ceb18da` —
  exact match with the SEK-07 cross-lane receipt's ten-lane tree.
- `git diff --name-only 7a70211b… HEAD` = 40 paths (10 lanes × 4), `git status
  --porcelain` empty.

## What the Builder produced (single continuous work unit)

1. **Deterministic materializer** (build tooling lives outside the repo write set, in
   the workspace temp area; only the 7 bounded outputs are committed): unique-key YAML
   load of the ten lane `records.yaml`, record-id union with parsed-identity assertion
   for shared records, derivation of the three referenced Source records from
   `SOURCE_REGISTRY.yaml`, projection-only normalization, canonical triple assembly,
   and byte-deterministic Turtle/JSON-LD/manifest emission.
2. **Merged graph** (792 records = 783 lane-owned + 6 shared + 3 derived sources);
   counts re-verified against the SEK-07 independent receipt (domains 1, subdomains 1,
   taxonomy_views 11, taxonomy_nodes 34, sources 3, source_locators 86, evidence 137,
   claims 106, concepts 38, principles 14, practices 28, patterns 12, anti_patterns 10,
   decision_models 11, coverage_gaps 29, relationship_assertions 271; lifecycle 404
   PENDING_REVIEW + 162 PROPOSED knowledge-class, 136+1 evidence, 86 ACTIVE locators,
   3 ACTIVE sources; all 106 claims review_status PENDING; 0 ReviewDecisions).
3. **Legacy SEK-03 compatibility analysis** (see decision 1 below) and **locator
   crosswalk** (decision 2), computed mechanically and rendered into
   `generated/README.md` + `generated/COVERAGE_REPORT.md`.
4. **Exports**: `generated/graph.ttl` (Turtle), `generated/graph.jsonld` (JSON-LD),
   `generated/manifest.sha256`; plus the three prose outputs
   (`generated/README.md`, `generated/COVERAGE_REPORT.md`, `knowledge/HANDBOOK.md`)
   and this observation.

## Integration decisions (issue-required)

1. **Legacy SEK-03 `KNOWLEDGE_RECORDS.yaml`: EXCLUDED, retained as historical
   fixture.** Field-level mapping analysis (17 locators, 12 evidence, 12 claims, 12
   assertions) shows the mapping into model 0.3.0 is not lossless: model-required
   per-record identity/lifecycle fields (`record_version`, `lifecycle_status`,
   `created_on`, locator `accessed_on`, assertion `context`/`rationale`) are absent and
   would have to be synthesized; claim-level `source_refs`/`taxonomy_refs` and
   slice-level `registry_ref`/`taxonomy_ref` have no model representation. Documented
   renames (`locator_id`→`id`, `verified_content`→`verified_content_summary`,
   `paraphrase`→`statement`, `claim_type`→`claim_kind`, `scope`→`scope_description`,
   `subject_id`/`object_id`→`subject_ref`/`object_ref`) do not rescue losslessness.
   Counts, reasons and the exclusion decision are published in
   `generated/COVERAGE_REPORT.md` §"Excluded legacy material"; the fixture file is
   untouched (byte-identical to base).
2. **Four locator ID-space overlaps: deterministic crosswalk, one canonical identity.**
   `SE-LOC-CS2023-BOK-TABLE`→`se:LOC-CS2023-BOK-TABLE`,
   `SE-LOC-CS2023-SE-OVERVIEW`→`se:LOC-CS2023-SE-OVERVIEW`,
   `SE-LOC-CS2023-SE-TOOLS`→`se:LOC-CS2023-SE-TOOLS`,
   `SE-LOC-NIST-218-RISK`→`se:LOC-NIST-218-RISK`. Each was mechanically compared on
   (mapped source_ref, locator text, locator_type, verification_state) — all four
   identical; the canonical graph identity is the lane record. No equivalence beyond
   those verified fields is asserted (no owl:sameAs, no merged records). Table in
   `generated/README.md`.
3. **`boundary_notes[1]` historical truncation: repaired in the derived projection
   only.** Raw source: `docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/records.yaml`
   **line 42** (lane 06A, the first canonical carrier). Carrier state verified across
   all ten lanes: the three SEK-06 lane files (06A line 42, 06B line 45, 06C line 55)
   preserve the complete intended sentence in raw bytes as an unquoted plain scalar
   whose `" #"` parses as a comment; the seven SEK-07 lane files carry the
   already-truncated literal (SEK-07's shared snippet pack propagated the parsed
   value — four carriers quoted, three unquoted). Parsed values are identically
   truncated in 10/10 lanes. The raw reviewed input therefore unambiguously preserves
   the complete intended text. Normalization rule: the intended value is the verbatim
   raw sequence-entry text after the `- ` marker in the first canonical (SEK-06)
   carrier. Before (parsed, kept verbatim in all lane files):
   `SWEBOK v4.0a identity and registry reconciliation is complete (SEK-02R`. After
   (projection only): `SWEBOK v4.0a identity and registry reconciliation is complete
   (SEK-02R #325 merged via PR #326); SWEBOK content is cited only where a lane has
   directly reinspected the located sections, and such records stay
   PROPOSED/PENDING_REVIEW until human review.` No lane file was rewritten; the source
   defect remains visible in-repo and is documented here, in `generated/README.md` and
   in `knowledge/HANDBOOK.md` §1.
4. **Rights and authority boundaries preserved.** CS2023's educational-permission
   scope and other-use-requires-permission boundary remain explicit (including the
   HIGH-severity rights-clearance gap record); SWEBOK handling (no PDF redistribution,
   concise paraphrase only) and NIST paraphrase preference are carried verbatim from
   the registry into `se:rights_handling`; ISO stays metadata-only and is not
   projected; no new Source Registry entries or rights assumptions were made.
5. **No automatic acceptance.** Every knowledge/evidence/relationship record keeps its
   reviewed PROPOSED/PENDING_REVIEW state; zero ReviewDecisions were created; the
   merged instance re-validated by the repository validator (which enforces
   no-ACCEPTED invariants) and by the projection-level checks.

## Validation — commands and exit codes (final tree, pinned env)

Environment: Windows-native Python 3.14.7; `linkml==1.12.1`, `linkml-runtime==1.12.1`,
`PyYAML==6.0.3`, `rdflib==7.6.0`, `jsonschema==4.26.0` (the SEK-07 verification venv,
pins re-checked via `importlib.metadata`). Build scripts under the workspace temp area
(`_tmp/sek08-build/`), not in the repository write set.

```
# A. determinism — 5 assemblies, 3 lane orders (canonical ×3, reversed, rotate)
python materialize.py --worktree <wt> --out scratch/<o> --order <o>   # ×5, each EXIT=0
# unique (ttl_sha256, jsonld_sha256, canonical_digest) tuples across orders: 1
ttl_sha256   = cdbee8d52137a43e86bd5b94ba4afd23014c9e9469b8bbdaa88a420d10c8758c
jsonld_sha256= f4eba494db3c8eb4e3c6b248b842856af013d6a45999a4145fbde783987356bc

# B. export equivalence + identity checks (rdflib parse-back)
python verify_exports.py --worktree <wt> --dir <wt>/…/knowledge/generated
# → {"VERIFY":"PASS","ttl_triples":9461,"jsonld_triples":9461,"internal_triples":9461,
#    "canonical_digest":"ec18a1aac76d91230337ab68d39173ec4ddf9195ab5424b10af32836693fc3cd"}
# checks: TTL ≡ JSON-LD ≡ internal canonical set (triple-for-triple); every reference
# slot object is a URIRef; se:predicate values are URIRefs; typed-edge endpoints are
# URIRefs; every IRI under se:/rdf:/xsd namespaces; no dangling record references;
# relationship endpoints/predicates/qualifier requirements satisfied vs catalog.
EXIT=0

# C. repository validator on the merged graph (raw parsed values + derived sources)
python emit_merged.py --worktree <wt> --out-yaml merged.yaml --out-json merged.json
# → MERGED_EMIT_OK records=792
python <wt>/…/knowledge/model/validate_knowledge_graph.py --input merged.yaml
# → GRAPH_VALIDATION=PASS
EXIT=0

# D. validator self-tests (final tree model dir)
python …/knowledge/model/validate_knowledge_graph.py --self-test
# → SELF_TESTS=66/66
EXIT=0

# E. manifest check (in generated dir)
sha256sum -c manifest.sha256   # → graph.ttl: OK / graph.jsonld: OK
EXIT=0

# F. LinkML 1.12.1 JSON-Schema instance validation (merged instance, top_class=KnowledgeGraph,
#    JsonSchemaGenerator mergeimports=True, jsonschema Draft7)
# → linkml=1.12.1 linkml_runtime=1.12.1 jsonschema=4.26.0 / LINKML_ERRORS=0
EXIT=0
```

Canonical graph digest (SHA-256 over the sorted N-Triples form of the projection):
`ec18a1aac76d91230337ab68d39173ec4ddf9195ab5424b10af32836693fc3cd` — stable across all
five assemblies and both serialization formats. (The SEK-07 receipt's digest
`368a212a…` was computed by the SEK-07 builder's own canonicalization over the merged
YAML records; SEK-08 defines its digest over the RDF projection and additionally
re-verifies every SEK-07 merged-graph count instead of reusing the number.)

## Corrections and deviations

- Two serialization defects were found and fixed during the build (before any commit):
  a Turtle subject/`a` concatenation and a JSON-LD top-level `@id`+`@graph` form that
  rdflib parses as an empty named graph. Both were caught by the parse-back
  equivalence gate; after the fix, determinism was re-proven end-to-end (five
  assemblies). No repository content was affected by these iterations.
- `REPAIR_BATCH` (in-write-set repairs found by review): **none required so far** —
  all gates green at first full run after the tooling fixes above; awaiting the
  independent final verifier's Exact-SHA readback.

## Limitations

- Experimental projection for a pilot corpus; not a production ontology, not a Tally
  runtime dependency, no DomainForge/Domain Package promotion, no main merge, no
  release/deploy (all out of scope for #349).
- All knowledge content is human-review-pending; this build asserts no acceptance and
  measures no review effort (`HUMAN_REVIEW_BURDEN=NOT_MEASURED`).
- The projection derives three Source records from the registry (documented mapping,
  including carrying the SWEBOK registry `editor` into `authors` verbatim); the six
  unused registry sources (incl. all ISO records) are intentionally not projected.
- The boundary_notes normalization is deliberately confined to the derived projection;
  the ten lane source files keep the historical defect verbatim (single-writer repair
  of reviewed lane files is out of scope for SEK-08).
- Final HEAD/TREE/base bindings, complete changed-path list and the full receipt are
  published on Issue #349 by the Builder after commit; a different independent
  Verifier must audit that exact SHA before any PASS-to-next step (integration PR to
  WIP; never merge main).
