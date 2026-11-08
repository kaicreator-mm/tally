# SEK-09H — Integrate reviewed real Case records into the graph and deterministic exports

**Task:** #368 · **Parent:** SEK-09 #351 / pilot #299 · **Direct dependency:** #365 / PR #366
(SEK-09F case intake, independently final-verified and merged to WIP) ·
**Status:** BUILT / ALL_GATES_PASS / PENDING_INDEPENDENT_FINAL_REVIEW — nothing here is
ACCEPTED; G2 stays PARTIAL; G4 human review and G5 model scoring remain NOT_MEASURED.

## Exact baseline (rebound per issue requirement)

- WIP baseline HEAD `1ad69d6b6fb945f5924ac7eee6fda798b843782a`, TREE
  `7b460a92991faa8832f536a61823cffcd5101286` (contains PR #366 merge and the SEK-09G
  PROVIDER_UNAVAILABLE evidence; the old `55ab06a…` base was **not** used as diff base).
- `git merge-base HEAD wip/v2.1.1-knowledge` = baseline; worktree clean at start.

## What was merged (canonical inputs only)

1. `authority/SOURCE_REGISTRY.yaml`: +3 first-party publisher sources
   (`SE-SRC-09F-CF-OUTAGE-2025`, `SE-SRC-09F-CF-KV-REDESIGN-2025`,
   `SE-SRC-09F-GH-POSTMORTEM-2018`), `source_class: EMPIRICAL_CASE`, `source_role:
   PRIMARY`, with rights handling, authority fit/limitations, verification
   (`state: CONTENT_PARTIAL_VERIFIED`, `accessed_on: 2026-11-08`) and the
   unpinned-HTML-revision caveat carried verbatim from the SEK-09F review. Registry
   grows 9 → 12 entries; the six unused entries stay unprojected.
2. `knowledge/vertical-slice/lanes/reliability-performance/records.yaml`: +4
   SourceLocator, +5 Evidence, +2 Case records copied verbatim (parsed values
   identical) from the reviewed `case-intake/SEK-09F-INCIDENT-CASES.yaml`, added under
   marked SEK-09F comments. Both cases: `PENDING_REVIEW`, `synthetic: false`,
   `case_kind: POSTMORTEM`; all five evidence: `PENDING_REVIEW`,
   `EXTRACTED_PENDING_REVIEW`, `synthetic: false`. Lane total 94 → 105 records.
3. **Deliberately not done:** no Counterexample (neither case challenges a specific
   qualified claim), no ReviewDecision, no ACCEPTED, no `case_refs` back-patching onto
   patterns, no edits to the 29 CoverageGap records, no edits to schema/catalog/
   validator, other lanes, the legacy fixture, or the case-intake sidecar (read-only).

## Generator

The SEK-08 deterministic materializer (build tooling lives outside the repo write set,
per SEK-08 precedent) was adapted for SEK-09H in the workspace temp area
(`C:\xDev\kAiCreator\_tmp\tally-i368-k3\`, not committed): expected-count table
updated (sources 6, locators 90, evidence 142, cases 2) and SEK-09H-specific
in-memory checks added (cases PENDING_REVIEW/non-synthetic/POSTMORTEM/evidenced; 09F
evidence bound to 09F locators; counterexamples absent). Graph root identity
(`se:GRAPH-V2.1.1-SEK08`), `graph_version 0.1.0` and the fixed `BUILD_DATE
2026-11-07` constant are intentionally preserved so repeat builds stay byte-comparable
across SEK-08 → SEK-09H; the only dates in the projection remain fixed constants.

## Results

- **Records:** 794 lane-owned + 6 shared + 6 derived sources = **806** (was 792).
- **Triples:** 9,653 (was 9,461; +192). Canonical digest (sorted N-Triples SHA-256):
  `060fa255a2af69edcee0580d6ef960cdf3ca2bac117d570003c339d2dd94aa79`.
- Exports @ this tree (relative to `docs/versions/v2.1.1/knowledge/`):
  - `generated/graph.ttl` sha256 `3d143897811ccf64dc0d0534f7720b66dcdb64f1b0fa0eb2dd90294cf13362d8`
  - `generated/graph.jsonld` sha256 `0db0e256c4fa2cbac8ec4592df333693b40c9e4fe3a6508792c7bfe82e17ae69`
  - verify with `sha256sum -c generated/manifest.sha256`
- Counts: 106 claims (unchanged, all `review_status: PENDING`), 271 relationship
  assertions (unchanged), 29 coverage gaps (unchanged, all OPEN), 568 knowledge-class
  records (406 PENDING_REVIEW + 162 PROPOSED), 142 evidence (141 PENDING_REVIEW + 1
  PROPOSED), 90 ACTIVE locators, 6 ACTIVE sources, 2 PENDING_REVIEW cases, 0
  counterexamples, 0 review decisions, 0 ACCEPTED.

## Validation (all re-run at the final tree; pinned env: Python 3.14.7,
linkml 1.12.1 / linkml-runtime 1.12.1 / PyYAML 6.0.3 / rdflib 7.6.0 / jsonschema 4.26.0,
isolated venv `C:\xDev\kAiCreator\_tmp\tally-i368-k3\venv`)

- **A. Determinism:** 5 assemblies × 3 lane orders (canonical ×3, reversed, rotate) →
  unique `(ttl_sha256, jsonld_sha256, canonical_digest)` tuple set = **1**;
  regeneration from the committed tree byte-identical (`cmp` PASS for `graph.ttl`,
  `graph.jsonld`, `manifest.sha256`). EXIT=0.
- **B. Export equivalence & identity:** `verify_exports.py` →
  `{"VERIFY":"PASS","ttl_triples":9653,"jsonld_triples":9653,"internal_triples":9653}`
  — TTL ≡ JSON-LD ≡ internal canonical set; NT digest payload parses; all reference
  objects URIRefs; `se:predicate` values URIRefs; typed-edge endpoints URIRefs; IRIs
  under declared prefixes; manifest hashes OK. EXIT=0.
- **C. Repository validator:** `emit_merged.py` → `MERGED_EMIT_OK records=806`;
  `validate_knowledge_graph.py --input merged-graph.yaml` → `GRAPH_VALIDATION=PASS`.
  EXIT=0.
- **D. Validator self-tests (negative fixtures):** `SELF_TESTS=66/66`. EXIT=0.
- **E. Manifest:** `sha256sum -c manifest.sha256` → `graph.ttl: OK`,
  `graph.jsonld: OK`. EXIT=0.
- **F. LinkML 1.12.1:** `JsonSchemaGenerator(mergeimports, top_class=KnowledgeGraph)`
  + Draft7 on the merged instance → `LINKML_ERRORS=0`
  (`linkml=1.12.1 runtime=1.12.1 jsonschema=4.26.0`). EXIT=0.
- **G. Baseline comparison vs `1ad69d6b`:** 8,668 baseline JSON-LD property assertions,
  **0 removed**, 178 added — all on the 14 new record subjects only (no triple touched
  any baseline subject, including the graph root); the 106 baseline claim nodes are
  node-for-node identical; `git diff --name-only` vs baseline = exactly the write set
  (registry, lane records + README, generated graph.ttl/jsonld/manifest/README/
  COVERAGE, HANDBOOK, this file). EXIT=0.
- **H. Targeted negative fixtures (9/9 rejected):** duplicate YAML key (native PyYAML
  duplicate-key rejection via the materializer's UniqueLoader); duplicate
  `(source_ref, locator)`; case flipped to ACCEPTED (repository validator +
  materializer flag); dangling case `evidence_refs`; counterexample with wrong ref
  class (Evidence) + materializer counterexample refusal; duplicate source identifier;
  unknown `evidence_type` enum (LinkML Draft7). Each injected defect was refused.
  EXIT=0.
- **Source-blobs:** the ten lane files (nine untouched byte-identical), schema,
  catalogs, validator, legacy fixture, case-intake sidecar and experiments paths are
  unchanged vs baseline (`git diff --name-only` restricted to those paths = empty).

## Boundary and next steps

Single central Writer; write set limited to the issue's bounded paths. No PR opened,
no WIP/main merge (merge happens only after independent final Exact-SHA review, with
an expected-head guard), no release/tag/DomainForge. G4/G5 NOT_MEASURED; G2 PARTIAL.
The builder has stopped writing; a fresh independent Final Verifier should audit HEAD
of `task/tally-i368-sek09h-graph-k3` against this receipt.
