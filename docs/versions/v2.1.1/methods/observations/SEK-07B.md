# SEK-07B — Method Observation (Implementation, Code Engineering & Maintainability lane)

**Task:** #335 (lane SEK-07B of the SEK-07 knowledge-taxonomy expansion; coordinator #333, pilot #299)
**Date:** 2026-11-07
**Worktree / branch:** `tally-i335-zcode` / `task/tally-v211-sek07b-implementation-code`
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (exact WIP base; branch created from it, not stacked)
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING
**HUMAN_REVIEW_BURDEN=NOT_MEASURED**

## Claim / CAS / dispatch readback (per Task Pack §Execution step 2)

- `gh issue view 335 --repo kaicreator-mm/tally --json labels,assignees,comments`
  returned `{"assignees":[],"comments":[],"labels":[]}` — **no Claim, CAS or
  dispatch authority is recorded on the issue: NOT_PRESENT**. No generation or
  status values exist to read back, so none are reported.
- Worktree check: no SEK claim/dispatch files exist for this task. The files
  `claim-replay-fixture.mjs` / `claim-replay-runner.mjs` at the repo root are
  pre-existing Tally v2 product code (tracked since base, CI-era commits), not
  knowledge-claim authority records.
- Ownership fencing: `git ls-tree` over all seven pushed SEK-07 task branches
  shows no branch writes `lanes/implementation-code`; the pushed
  `origin/task/tally-v211-sek07b-implementation-code` tip equals the base commit
  `7a70211b...` (coordinator-created placeholder, zero commits). No ownership
  conflict detected; no duplicate Claim/Dispatch created.

## Research path — SWEBOK KA4 direct reinspection

1. The official SWEBOK Guide v4.0a PDF was located at
   `_tmp/i325-localval/pdf/swebok-v4.pdf` and its SHA-256 recomputed on
   2026-11-07: `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
   matching the registry-verified record (411 pages).
2. Chapter boundaries were located by phrase search with the SEK-07 pypdf
   scripts (venv Python 3.14.7): "4 SOFTWARE CONSTRUCTION" hits 1-based pages 9,
   99, 402 and "5 SOFTWARE TESTING" hits 10, 117, 402 — so KA4 spans 1-based PDF
   pages 99-116 (0-based indices 98-115), consistent with the brief's stated
   chapter start (1-based page 99). The full chapter text was extracted to
   `_tmp/tally-sek07-lanes/validation/07B/swebok-ka4-extract.txt` (working
   artifact, not committed) and read in full.
3. Five locators were admitted, each scoped to chapter + section/subsection
   numbers with both PDF page indices (0-based) and printed chapter-page folios:
   Introduction + section 1 (indices 98-101, 4-1 to 4-4); 3.3 + 3.6 (104-106,
   4-7 to 4-9); 3.7 (106-107, 4-9 to 4-10); 4.1 + 4.4 + 4.5 (107-108, 4-10 to
   4-11). Sections read for navigation but deliberately not admitted as records
   are enumerated in `se:GAP-07B-SWEBOK-KA4-BREADTH`.
4. Source hedges were preserved in-band: design by contract "is thought to
   improve" (4.4); incremental integration "is thought to offer many advantages"
   (3.7). No hedged statement was promoted to an unhedged one.

## Research path — CS2023 heading-anchored units

1. The verified content base is the 2026-11-07 full-text extraction of the
   official CS2023 Body of Knowledge PDF captured in
   `_tmp/tally-sek07-lanes/sources/cs2023-bok-sek07-extract.md` (SE-Construction
   and SDF-Practices units, SE-Tools analysis topic), plus the SEK-06 verified
   anchors pack (`_tmp/tally-sek06-lanes/extract/cs2023-se-verified-anchors.md`,
   extraction 2026-11-06) for the SE-Design code-smell topic and the SE
   Core-Hours rows.
2. Direct binary download of the CS2023 PDF was not re-attempted in this lane;
   the same network constraint as SEK-06 stands, so **no binary SHA-256 of the
   CS2023 PDF is recorded and no record claims one**.
3. Page-anchoring rule applied: page-bearing locators use only the
   registry-verified anchors (shared Core Hours table = PDF page indices
   176-177; SE-Tools unit = PDF page index 178). The three new CS2023 locators
   (SE-Construction KU, SDF-Practices KU, SE-Design code-smell topic) are
   **heading-anchored** — exact unit heading + topic numbering verified against
   the extraction — and explicitly disclaim unconfirmed printed pages inside
   their locator strings. No page number was invented anywhere in the lane.
4. Rights handling stayed inside the registry boundary: bibliographic metadata,
   exact locators, concise original paraphrases; short verbatim topic titles
   only where locator anchoring requires them; no substantial report text.

## Corrections and authoring discipline

1. **Shared-snippet anomaly (reported, not repaired).** The SEK-07 shared
   snippet pack's domain record `se:DOMAIN-SOFTWARE-ENGINEERING` carries a
   second boundary note that ends mid-sentence ("…complete (SEK-02R"), unlike
   the SEK-06 lane version which completes the thought. The copy-verbatim rule
   forbids rewording shared records, so the lane copied it byte-identically and
   flagged the truncation here, in the lane README, and in the builder report.
   Resolution belongs to the coordinator (regenerate the snippet pack) or
   SEK-08, not to this lane's write set.
2. **Schema-conformance pass.** All records were checked against the LinkML
   schema before validation: DecisionModel carries no `label` (omitted);
   AntiPattern carries no `limitations` (the synthesis disclaimer lives in its
   scope field); Practice/Pattern/AntiPattern required lists populated.
3. **Provenance-hygiene pass.** Claim records carry no evidence list; their
   evidence links are canonical `SUPPORTED_BY` edges. Evidence-bearing edges
   cite existing evidence records only. The four shared records were copied
   verbatim from `shared-snippets-sek07.yaml` and re-verified
   parsed-identical to the pack before validation.
4. **Boundary-fence pass.** Secure-coding detail fenced to 07C (records carry
   the fence in limitations/scope); CI delivery mechanics fenced to 07E;
   reliability/performance engineering fenced to 07D; refactoring/technical-debt
   depth fenced to 07F; general architecture fenced to SEK-06B. No
   cross-lane relationship edge to another lane's records was created (none
   exist in this lane's graph); the fences are carried as qualifiers, scope
   fields and this note.
5. **Synthesis marking.** The single lane inference (big-bang anti-pattern
   characterization) is an explicitly marked `SYNTHESIS` evidence record at
   `PROPOSED` (not PENDING_REVIEW), with the inference disclosed at the
   evidence, at the anti-pattern, and on the `DERIVED_FROM` edge.

## Limitations

- CS2023 locators beyond the two registry-verified anchors are heading-anchored
  (no binary PDF hash obtainable this session); SWEBOK locators are verified
  against the hash-checked official PDF binary.
- No Case/Counterexample records exist; the missing evidence class is recorded
  as `se:GAP-07B-REAL-CASE-EVIDENCE` rather than papered over.
- No empirical evidence is cited anywhere in the lane; SWEBOK hedges stay
  hedged.

## Validation — commands and results (final exact-tip receipt)

See `C:/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07B/` for full logs.
Environment: pinned venv Python 3.14.7 (LinkML 1.12.1, linkml-runtime 1.12.1,
PyYAML 6.0.3, jsonschema 4.26.0, pypdf 7.0.0), `PYTHONDONTWRITEBYTECODE=1`,
Windows-native.

<!-- VALIDATION_RECEIPT -->

1. Preflight (working tree, before commit): `git diff --check` exit 0 (no
   whitespace errors); `records.yaml` parsed with the venv Python
   (`yaml.safe_load`) — parse OK.
2. Repository validator self-test at the final committed tip:
   `validate_knowledge_graph.py --self-test` → exit 0, final line
   `SELF_TESTS=66/66` (log: `selftest-tip.log`).
3. Lane assembly and validation at the same committed tip:
   `assemble_and_validate.py <worktree> 07B <records.yaml>
   C:/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07B` → exit 0
   (log: `lane-validate-tip.log`), key lines:

```text
LANE_RECORD_COUNTS={"anti_patterns": 1, "concepts": 4, "coverage_gaps": 3, "decision_models": 1, "domains": 1, "evidence_records": 14, "knowledge_claims": 9, "patterns": 2, "practices": 3, "principles": 1, "relationship_assertions": 26, "source_locators": 9, "subdomains": 1, "taxonomy_nodes": 2, "taxonomy_views": 2}
COUNT_source_locators=9 MIN=3 PASS
COUNT_evidence_records=14 MIN=3 PASS
COUNT_knowledge_claims=9 MIN=4 PASS
COUNT_concepts=4 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=3 MIN=2 PASS
COUNT_patterns=2 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=3 MIN=1 PASS
COUNT_relationship_assertions=26 MIN=10 PASS
DERIVED_SOURCES=2 se:SRC-CS2023,se:SRC-SWEBOK-GUIDE-4-0A
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

4. Four-path SHA256 manifest and identity binding at the same tip:
   `manifest-four-paths.sha256`, `identity.txt`, `receipt.ini` in the receipt
   dir; HEAD/TREE/BASE and the clean-worktree before/after states are recorded
   there. The validation runs bind to the committed tree, not a working-tree
   approximation.
5. Validation failures during authoring: none — the first dry run of both
   commands on the completed four-file write set was fully green; no repair
   iterations were needed (the two README count-row corrections — evidence and
   locator counts — were made during preflight, before any validation run).

