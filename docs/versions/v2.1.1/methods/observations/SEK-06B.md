# SEK-06B — Method Observation (Architecture and Design lane)

**Task:** #323 (lane builder, SEK-06 family)  
**Date:** 2026-11-06  
**Baseline HEAD:** `7a70211b4460e4c753c97798670de9f34861d4a4` (after the Owner-authorized 2026-11-06 WIP rebase; original authoring base `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`)  
**Branch:** `task/tally-v211-sek06b-architecture-design` (worktree `tally-i323-zcode`)  
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING

## Research path (same access conditions as SEK-06A)

- The official CS2023 Body of Knowledge PDF
  (`https://csed.acm.org/wp-content/uploads/2024/04/3.1-Body-of-Knowledge-1.pdf`, source
  `SE-SRC-CS2023`) was obtained as full text via a text-extraction service on 2026-11-06.
- Direct binary download from csed.acm.org failed from this network: TLS connection reset
  through both direct and proxied routes; the Internet Archive snapshot was rate-limited
  (HTTP 429). **No binary SHA-256 of the PDF is recorded** for this extraction.
- Heading and topic text is verbatim from the extraction and is the authoritative anchor
  for heading-anchored locators. The printed page token `175` was adjacent to the
  `Software Engineering (SE)` heading in the text stream, consistent with the registry
  anchors.
- Page anchors are limited to registry-verified locators (SE Overview = PDF page indices
  174-176; SE Core Hours table = PDF page indices 176-177). Printed pages for individual
  KU sub-sections were NOT independently confirmed, so the new KU-level locator
  `se:LOC-CS2023-SE-DESIGN-KU` is heading-anchored and explicitly states that no printed
  page is asserted. No page numbers were invented anywhere in the lane.
- ISO/IEC/IEEE 42010:2022 was consulted only at the level of the official bibliographic
  metadata page (registry entry `SE-SRC-ISO-IEC-IEEE-42010-2022`, ACCESS_RESTRICTED /
  METADATA_ONLY under the ISO AI/ML-use notice). No normative content was extracted,
  retained or prompted; its unavailability is recorded as the coverage gap
  `se:GAP-ARC-ISO42010-ARCH-DESCRIPTION`.
- SWEBOK v4.0a remains on source-identity hold (#325 unmerged); no SWEBOK KA2/KA3 content
  is cited by this lane (`se:GAP-ARC-SWEBOK-DESIGN-CONSENSUS`).

  *(Historical authoring-time statement. Current status after the 2026-11-06 P2 repair:
  #325 merged via PR #326 into the WIP base; identity/registry reconciliation complete;
  KA2/KA3 sections reinspected directly and cited at content level — see the P2 repair
  receipt at the end of this observation.)*

## Locator verification

| Locator | Type | Anchor | State |
|---|---|---|---|
| `se:LOC-CS2023-SE-OVERVIEW` | SECTION (shared, verbatim) | PDF page indices 174-176, SE Overview | LOCATOR_VERIFIED (registry) |
| `se:LOC-CS2023-SE-KU-TABLE` | TABLE (shared, verbatim) | PDF page indices 176-177, Core Hours table | LOCATOR_VERIFIED (registry) |
| `se:LOC-CS2023-BOK-TABLE` | TABLE (shared, verbatim) | PDF page index 0, front-matter KA table | LOCATOR_VERIFIED (registry) |
| `se:LOC-CS2023-SE-DESIGN-KU` | SECTION (new this lane) | heading-anchored 'SE-Design: Software Design' unit, CS Core 1-4 / KA Core 5-9 / Non-core 10-13; printed page not asserted | LOCATOR_VERIFIED (full-text extraction 2026-11-06) |

All evidence `source_locator_ref` values resolve to the above; all `source_ref` values are
`se:SRC-CS2023`. The five shared records (domain, view, three registry locators) were
copied verbatim from the SEK-06 shared snippet set and verified record-equivalent to the
snippet file after writing.

## Lane content summary

- 11 evidence records: 10 DIRECT_SOURCE_PARAPHRASE + 1 declared SYNTHESIS
  (`se:EVD-ARC-0010`, feeding the anti-pattern); all `EXTRACTED_PENDING_REVIEW`,
  `synthetic: false`.
- 6 knowledge claims (all `review_status: PENDING`), including the abstraction-levels
  definition claim that keeps architecture design and detailed design distinct without
  touching the unavailable architecture-description layer.
- 3 concepts, 1 principle, 2 practices, 1 pattern (layered architecture, context
  provided), 1 anti-pattern (inconsistent API surface, DERIVED_FROM the synthesis
  evidence), 1 decision model (design tradeoffs), 4 coverage gaps (three lane gaps plus
  the shared CS2023 rights-clearance gap), 13 typed relationship
  assertions (6 SUPPORTED_BY, 2 APPLIES_TO, 1 QUALIFIED_BY, 1 DEPENDS_ON, 1 IMPLEMENTS,
  1 DERIVED_FROM, 1 RELATED_TO).
- No record is ACCEPTED; no Case/Counterexample records were created (no real-case
  evidence source admitted; recorded as `se:GAP-ARC-DESIGN-REAL-CASES`).

## Validation results (2026-11-06, Windows native, venv Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1)

Repository validator self-test (`validate_knowledge_graph.py --self-test`,
`PYTHONDONTWRITEBYTECODE=1`): exit 0, final line `SELF_TESTS=66/66`.

**Historical receipt — exact committed-tip validation at `d46cfc4`** (the
full validator run and exit codes bind to that committed tree, not a
working-tree approximation; it was the PR tip until the Builder receipt-rebind
commit `9342aec` and is preserved here as receipt history — see the final-tip
receipt appended at the end of this file): HEAD
`d46cfc4af8381fabd13ec1dfd6f10149d2ed1294`, tree
`932ad13f9aed93322c471fd97da1501e4e401662`, base
`cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #328
headRefOid equalled this HEAD at that time; clean worktree before 0 dirty /
after 0 dirty).
Commands on the committed tree: `validate_knowledge_graph.py --self-test`
→ exit 0 (`SELF_TESTS=66/66`); `assemble_and_validate.py <worktree> 06B
<records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
`GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
`LINKML_INSTANCE_VALIDATION=PASS` (LinkML 1.12.1 / linkml-runtime 1.12.1),
`LANE_VALIDATION=PASS`; key lines identical to the quoted block below.
Receipt dir `_tmp/tally-sek06-lanes/validation/06B-bld4/` (`identity-tip.txt`,
`selftest-tip.log`, `lane-validate-tip.log`,
`manifest-four-paths-tip.sha256`).

Four-path SHA256 manifest at this committed tip (repo-relative; this
receipt-rebind commit changes only this observation file, so the first three
hashes remain valid at the pushed HEAD; the post-commit rerun and full-path
manifest at the pushed HEAD are recorded in the receipt dir as
`selftest-post.log`, `lane-validate-post.log`,
`manifest-four-paths-post.sha256`):

```text
3ec0a34cb3187b1f6f9cd8bcb4f98d57b09eda1ebedc326d91511b470cacede8  lanes/architecture-design/README.md
298751c49f2ab5008585e72ae7cf15025aebd83d52ba42264ca20a310fac55fa  lanes/architecture-design/records.yaml
c299cea74c2f764ed43df51a8de4d1336c7151ebeb715c65b38e5e071cb88482  lanes/architecture-design/EXPLANATIONS.md
057b225936ec4d44f4dfa23229bdc0df326f0df318ecf056392208e44ad7fc72  methods/observations/SEK-06B.md
```

Earlier receipts retained as history only (all superseded by the exact-tip
receipt above): `eb32a2e` (`06B-bld2`), the builder pass at `7c57d80`
(`06B-bld3`), the post-quotation-fix rerun (`06B-rev1/lane-validation-postfix.log`),
and the pre-fix initial runs (`06B/`). The quoted assembly block is identical
at every green receipt tip:

```
LANE_RECORD_COUNTS={"anti_patterns": 1, "concepts": 3, "coverage_gaps": 4, "decision_models": 1, "domains": 1, "evidence_records": 11, "knowledge_claims": 6, "patterns": 1, "practices": 2, "principles": 1, "relationship_assertions": 13, "source_locators": 4, "taxonomy_nodes": 1, "taxonomy_views": 1}
COUNT_source_locators=4 MIN=3 PASS
COUNT_evidence_records=11 MIN=3 PASS
COUNT_knowledge_claims=6 MIN=4 PASS
COUNT_concepts=3 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=2 MIN=2 PASS
COUNT_patterns=1 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=4 MIN=1 PASS
COUNT_relationship_assertions=13 MIN=10 PASS
DERIVED_SOURCES=1 se:SRC-CS2023
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

One iteration was needed: an unquoted limitation string containing `": "` in
`se:EVD-ARC-0010` parsed as a YAML mapping and failed LinkML string typing (run1). Fixed
by quoting the string; run2 is fully green. Shared files (validation script, model,
registry) were not modified.

## Review burden and state

```ini
REVIEW_BURDEN=NOT_MEASURED
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```

All records remain PENDING_REVIEW / EXTRACTED_PENDING_REVIEW pending human review; the
synthesis evidence and the two interpretive edges (DEPENDS_ON, RELATED_TO) are flagged as
the highest-priority review targets because their inferential content exceeds the
paraphrase base by design.


## Shared CS2023 rights-clearance gap

The CS2023 source registry limits permission to educational materials/programs
and requires specific permission for other uses. This lane is part of Tally's
general software-engineering knowledge graph; it is not presumed to qualify
under that permission. The shared HIGH/OPEN record
`se:GAP-CS2023-RIGHTS-CLEARANCE` is included verbatim across SEK-06A, SEK-06B
and SEK-06C. No rights clearance is claimed. The four-path correction initially
postdated the lane-validation receipt above; the receipt was re-run at the
exact tip `eb32a2e` on 2026-11-06
(`_tmp/tally-sek06-lanes/validation/06B-bld2/lane-validation.log`) and is
fully green, including `COUNT_coverage_gaps=4`.

## Independent review repair (rev1, 2026-11-06)

Exact-SHA review at `7c57d808` (identity, mechanism, semantics) found one
quotation-accuracy defect: the README three-layer item 2 presented "as large
as software architecture" in quotation marks as SE Overview wording, but the
verbatim source phrase is "large design concerns such as software
architecture". Fixed in the README by quoting the verbatim phrase; no other
file changed (`records.yaml` byte-identical to the `eb32a2e` receipt tip, all
six shared records verified record-equal to the shared snippet set). Lane
validation rerun after the fix is fully green
(`_tmp/tally-sek06-lanes/validation/06B-rev1/lane-validation-postfix.log`),
matching the receipt above line-for-line. No shared-record or
schema/catalog/validator change was needed.

## Historical final-tip receipt — Builder batch `9342aec` (2026-11-06 R2 repair append; old WIP base, superseded by the 2026-11-06 WIP rebase onto `7a70211`)

The R2 Exact-SHA review of PR #328 required the observation to name the final
Builder-batch tip; the `d46cfc4` receipt above (and the earlier `7c57d80` /
`eb32a2e` receipts) are historical context, preserved unchanged.

- Identity: HEAD `9342aecb6f70c65248e7d0c1e279fcd1acf45abb`, tree
  `04e0b76212b4f1a79189b39d2ca45834f82bc741`, base
  `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #328
  headRefOid equalled this HEAD at the R2 review).
- Commands and exit codes at this exact tip: `validate_knowledge_graph.py
  --self-test` → `SELF_TESTS=66/66`, exit 0; `assemble_and_validate.py
  <worktree> 06B <records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (counts unchanged:
  50 lane records, 13 relationship assertions, `coverage_gaps=4`, derived
  source `se:SRC-CS2023`). Clean worktree before/after: 0/0 dirty. Logs:
  Builder batch `_tmp/tally-sek06-lanes/validation/06B-bld4/`
  (`identity-post.txt`, `selftest-post.log`, `lane-validate-post.log`,
  `manifest-four-paths-post.sha256`), independently re-run for this repair in
  `_tmp/tally-sek06-lanes/validation/06B-r2r/`.
- Four-path SHA256 manifest at HEAD `9342aec` / tree `04e0b76`:

```text
3ec0a34cb3187b1f6f9cd8bcb4f98d57b09eda1ebedc326d91511b470cacede8  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/README.md
298751c49f2ab5008585e72ae7cf15025aebd83d52ba42264ca20a310fac55fa  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/records.yaml
c299cea74c2f764ed43df51a8de4d1336c7151ebeb715c65b38e5e071cb88482  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/EXPLANATIONS.md
c1de34d3b2369fe088f3aae6f5cf8f8c45371e1c1c256c48978068638d9e9ea2  docs/versions/v2.1.1/methods/observations/SEK-06B.md
```

This append itself is an observation-only R2-repair commit and becomes the
newest tip; its exact HEAD/TREE, the post-commit rerun (same commands, exit
0), the updated four-path manifest and clean-worktree evidence are delivered
in the R2 repair delivery comment on PR #328 / coordinator #321, with logs in
`_tmp/tally-sek06-lanes/validation/06B-r2r/`. The ISO/IEC/IEEE 42010
metadata-only gap (`se:GAP-ARC-ISO42010-ARCH-DESCRIPTION`) and the no-case
gap (`se:GAP-ARC-DESIGN-REAL-CASES`) are unchanged by this receipt, as is the
SWEBOK hold (#325/#326 unmerged).

## WIP-rebase receipt — Owner-authorized rebase onto `7a70211` (2026-11-06)

Owner authorization recorded on #321 (2026-11-06) merged PR #326 (SEK-02R)
into `wip/v2.1.1-knowledge` — linear rebase merge, no merge commit — moving
the WIP head from `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (tree
`da5deb5eac25cd019883772967d288a1f2679a84`) to
`7a70211b4460e4c753c97798670de9f34861d4a4` (tree
`f9b01c74f39993299d9ee179c76cdc08d3baae9d`). This lane branch was then
rebased with `git rebase --onto 7a70211b4460e4c753c97798670de9f34861d4a4
cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`: all seven lane commits replayed
with **zero conflicts**, and the three lane content files are byte-identical
to the pre-rebase reviewed tip `7676fae68ae38c3ff58f9722dcf984366f7b7e58`
(verified by path-scoped diff against a pre-rebase backup ref). The earlier
receipts above (final-tip `9342aec` and its predecessors, all on the old
base) are **historical** and preserved unchanged.

- Identity at the rebased tip: HEAD `eff5e86c8c9266d12837b11ba0c3f5437b3f8ee3`,
  tree `1402d8569aaa0c51c41b386242e08f5768e171ec`, base / merge-base
  `7a70211b4460e4c753c97798670de9f34861d4a4`; clean worktree before/after:
  0/0 dirty.
- Validation rerun at that exact tip (Windows-native venv Python 3.14.7 /
  PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1,
  `PYTHONDONTWRITEBYTECODE=1`): `validate_knowledge_graph.py --self-test` →
  `SELF_TESTS=66/66`, exit 0; `assemble_and_validate.py <worktree> 06B
  <records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (counts
  unchanged: 50 lane records, 13 relationship assertions, `coverage_gaps=4`,
  derived source `se:SRC-CS2023`). Cross-lane check across the three rebased
  tips (06A `944cfee` / 06B `eff5e86` / 06C `343f30b`):
  `CROSS_LANE_CHECK=PASS` — 6 shared records semantically identical in all
  three lanes, 148 unique lane IDs, merged three-lane graph
  `GRAPH_VALIDATION=PASS` + `NO_ACCEPTED_RECORDS=PASS` + LinkML PASS.
  Logs: `_tmp/tally-sek06-lanes/validation/06B-rb1/`
  (`selftest-tip.log`, `lane-validate-tip.log`, exit receipts) and
  `_tmp/tally-sek06-lanes/validation/cross-lane-rb1-tip.log` (all exit 0).
- Content manifest at the rebased tip (three content files unchanged from
  the R2-reviewed `9342aec` / `7676fae` manifest — the rebase changed only
  base history, not lane content):

```text
3ec0a34cb3187b1f6f9cd8bcb4f98d57b09eda1ebedc326d91511b470cacede8  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/README.md
298751c49f2ab5008585e72ae7cf15025aebd83d52ba42264ca20a310fac55fa  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/records.yaml
c299cea74c2f764ed43df51a8de4d1336c7151ebeb715c65b38e5e071cb88482  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/architecture-design/EXPLANATIONS.md
```

- SWEBOK state after the rebase: the corrected SWEBOK v4.0a source record
  (SEK-02R / PR #326, including the native PDF SHA-256 and the open
  title-page/landing-page date discrepancy) is now part of this branch's
  base, and the source-identity hold is lifted at the registry level.
  **No SWEBOK-derived claim, locator or evidence was added by this rebase**:
  direct SWEBOK reinspection was not performed in this stage, and every
  knowledge record remains PROPOSED / PENDING_REVIEW with the CS2023
  curriculum-only scope, the ISO 42010 metadata-only gap and the no-case gap
  intact.
- This append itself is an observation-only rebase-receipt commit and
  becomes the newest tip; its exact HEAD/TREE, the post-commit rerun (same
  commands, exit 0), the updated four-path SHA256 manifest and
  clean-worktree evidence are delivered in the rebase delivery comment on
  PR #328 / coordinator #321, with logs in
  `_tmp/tally-sek06-lanes/validation/06B-rb1-post/`.

## P2 repair receipt — SWEBOK KA2/KA3 reinspection + stale-hold correction (2026-11-06)

Repair batch executed under the combined handoff on #321 (R1 review of the
post-WIP-rebase tips, both findings P2), within the original #323 four-path
write set, against reviewed tip `6b963b7345aa974d0c91af9c23ee63813beb46b6` /
tree `e7add997ca5ff8e80f1e854d3a971cca4dc96db8`, base
`7a70211b4460e4c753c97798670de9f34861d4a4`. Pre-write readback: PR #328 OPEN,
headRefOid equalled the branch ref and the reviewed tip; worktree clean; no
concurrent writer observed; write set unchanged.

### P2-01 — stale source-hold statements corrected

README.md, records.yaml and EXPLANATIONS.md no longer state that #325 is
unmerged or that SWEBOK is on source-identity hold. Current-state wording now
says: identity/registry reconciliation complete (#326 merged into this lane's
WIP base); content-level reinspection performed by this repair;
`se:GAP-ARC-SWEBOK-DESIGN-CONSENSUS` preserved as a stable ID but redescribed
as a content/evidence coverage gap (KA2 §4 and KA3 §3-6 uninspected; no human
review of SWEBOK-derived records), still MEDIUM/OPEN. The shared Domain and
View records' hold language was reworded **identically in all three lane
files** (06A/06B/06C) so the six shared records remain byte-identical for
SEK-08. Historical statements in this observation are preserved and labeled.
The ISO/IEC/IEEE 42010 metadata-only boundary is untouched.

### P2-02 — SWEBOK KA2/KA3 locator re-verification performed

Official source reopened locally: the retained official PDF copy
(`_tmp/i325-localval-r3/pdf/swebok-v4.pdf`) re-verified as SHA-256
`0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
6,015,672 bytes, 411 pages, `%PDF-1.7`, matching the SEK-02R registry record
(canonical source `SE-SRC-SWEBOK-GUIDE-4-0A`). Chapter/section headings and
printed folios were verified by direct text extraction at the exact page
indices. Admitted locators (PDF page indices 0-based; printed pages use the
Guide's chapter-page folios):

| Locator ID | Section(s) verified | PDF page indices | Printed pages |
|---|---|---|---|
| `se:LOC-SWEBOK-KA2-ARCH-DESCRIPTION` | Ch.2 §2 Software Architecture Description (§2.1 Views and Viewpoints, §2.2 Patterns, Styles and Reference Architectures incl. its reference-architecture passage completing on printed 2-7; §2.3 excluded) | 69–72 | 2-4 to 2-7 |
| `se:LOC-SWEBOK-KA2-ARCH-DECISIONS-PROCESS` | Ch.2 §2.4 Architecture as Significant Decisions and §3 Software Architecture Process (§3.1 incl. 3.1.1, §3.2 Architectural Design) | 72–74 | 2-7 to 2-9 |
| `se:LOC-SWEBOK-KA3-DESIGN-PROCESSES` | Ch.3 §1 Software Design Fundamentals (§1.1, §1.3, §1.4) and §2 Software Design Processes (§2.1 High-Level, §2.2 Detailed) | 83–87 | 3-2 to 3-6 |

Chapter 2 opens at PDF page index 66 (printed 2-1) and Chapter 3 at PDF page
index 82 (printed 3-1). Added records: the three locators above, three
`DIRECT_SOURCE_PARAPHRASE` evidence records
(`se:EVD-ARC-SWEBOK-DESCRIPTION-VIEWS`, `se:EVD-ARC-SWEBOK-DECISIONS-DEBT`,
`se:EVD-ARC-SWEBOK-DESIGN-STAGES`), three claims
(`se:CLM-ARC-SWEBOK-DESCRIPTION-DISTINCTION`,
`se:CLM-ARC-SWEBOK-DECISIONS-RATIONALE`,
`se:CLM-ARC-SWEBOK-DESIGN-STAGE-SPLIT`) and three canonical `SUPPORTED_BY`
edges (`se:REL-ARC-0014/0015/0016`). No page number, section or claim was
written that the extracted PDF text does not support; the Guide's
generally-accepted-knowledge-not-prescription limitation is carried inside each
claim. Rights boundaries kept: PDF not committed/mirrored/redistributed,
original paraphrases only, release-date discrepancy remains OPEN, all records
PROPOSED/PENDING_REVIEW, zero ACCEPTED, ISO 42010 metadata-only preserved.

### Validation (Windows-native, Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1, PYTHONDONTWRITEBYTECODE=1)

Run on the exact repaired working tree (= the committed tree of this repair
commit); rerun after the commit and the final four-path SHA256 manifest at the
pushed HEAD are delivered in the P2 repair delivery comment on PR #328 /
coordinator #321, logs in `_tmp/tally-sek06-lanes/validation/06B-p2/`.

1. `validate_knowledge_graph.py --self-test` → `SELF_TESTS=66/66`, exit 0.
2. `assemble_and_validate.py <worktree> 06B <records.yaml> <outdir>` → exit 0:
   every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
   `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (62 lane records:
   `source_locators=7`, `evidence_records=14`, `knowledge_claims=9`,
   `relationship_assertions=16`, `coverage_gaps=4`; derived sources
   `se:SRC-CS2023` + `se:SRC-SWEBOK-GUIDE-4-0A`). 50 → 62 lane records (three
   content files plus this observation are the only paths changed).

Cross-lane check at the three repaired tips and clean-worktree evidence are
recorded in the delivery comment. Human review burden remains NOT_MEASURED; no
record is ACCEPTED.

## R2 consolidated delivery receipt — reference-architecture fidelity + evidence closure (2026-11-06)

R2 review findings (#328, P2-01 source fidelity; P2-02 evidence closure). This
commit repairs both within the original #323 four-path write set.

### P2-01 repair — §2.2 reference-architecture locator and wording

Official PDF re-opened at the relevant pages (retained hash-verified copy,
SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`):
the §2.2 reference-architecture passage begins on PDF page index 71 (printed
2-6) — "A reference architecture (RA) is an architecture constraining or
guiding other architectures…" — and completes on PDF page index 72 (printed
2-7) with the commonalities/standardization clause and the domain list;
section 2.3 begins later on the same printed page 2-7. Corrections applied:

- `se:LOC-SWEBOK-KA2-ARCH-DESCRIPTION`: locator extended to **PDF page indices
  69-72 / printed 2-4 to 2-7**, explicitly scoping the §2.2 continuation on
  printed 2-7 and excluding §2.3 content on that page; summary rewritten to
  the source's own definitions (styles/patterns as documented solution forms
  with pattern catalogs; reference architectures as architectures constraining
  or guiding other architectures, providing a common basis and capturing
  commonalities to promote development, integration, interoperability and
  standardization).
- `se:EVD-ARC-SWEBOK-DESCRIPTION-VIEWS`: the mischaracterized "codify proven
  structures for a domain" clause replaced by the source definition above
  (including the product-line/families/domains common-basis wording); context
  now states the passage spans the 2-6/2-7 page break; result line updated.
- `se:CLM-ARC-SWEBOK-DESCRIPTION-DISTINCTION`: "recurring organization
  vocabularies" replaced — styles/patterns as documented solution forms,
  reference architectures as architectures that constrain or guide other
  architectures and provide a common basis; scope pages updated to 2-4 to 2-7.
- `se:REL-ARC-0014` rationale: locator range updated to PDF indices 69-72 /
  printed 2-4 to 2-7.
- The P2-receipt locator table above was corrected in place (row previously
  read 69–71 / 2-4 to 2-6); this note records that correction explicitly.
- README/EXPLANATIONS carried none of the mischaracterized wording or the old
  page range (checked); no change needed there.

### Claim/CAS identity, generation/status and exclusive-Writer readback (pre-write)

- Active claim chain for this repair batch: Owner-authorized combined
  Builder/Repairer handoffs on coordinator #321 — P2 stage comment 6684006230
  and R2 checkpoint comment 6686070478 — plus the R2 Repairer dispatch to the
  local executor on 2026-11-06. Status: ACTIVE for this batch; authorizes
  exactly the in-write-set R2 repairs and receipt closure. No new Task, Pack
  or Claim created for R2; generation unchanged: Task Pack issue #323 and its
  four-path write set remain the sole authority, re-read before writing.
- Exclusive Writer readback (2026-11-06, immediately before writing): PR #328
  OPEN; live PR headRefOid `0bd24b94ddf410c2d6519320f82d206a1227d549` ==
  branch ref `task/tally-v211-sek06b-architecture-design` == R2-reviewed HEAD;
  local worktree at that HEAD with 0 dirty paths; no other writer observed on
  the branch since the P2 push. Push after commit uses `--force-with-lease`
  with expected remote head `0bd24b94…`, fencing any concurrent writer.
- Identity bound by this receipt: R2-reviewed HEAD
  `0bd24b94ddf410c2d6519320f82d206a1227d549` / tree
  `f5ff424b996a037de2aad1c7e008a43182109bbf` / base (merge-base)
  `7a70211b4460e4c753c97798670de9f34861d4a4`.

### Validation — actual commands and exit codes (repaired tree = this commit's tree; Windows-native venv Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1, PYTHONDONTWRITEBYTECODE=1)

Receipt dir `_tmp/tally-sek06-lanes/validation/06B-r2/` (`identity-pre.txt`,
`selftest-pre.log`, `lane-validate-pre.log`); `git diff --check` exit 0.

1. `…/knowledge/model/validate_knowledge_graph.py --self-test` → exit 0,
   final line `SELF_TESTS=66/66`.
2. `…/tally-sek06-lanes/scripts/assemble_and_validate.py <worktree> 06B
   <records.yaml> <outdir>` → exit 0: `LANE_RECORD_COUNTS` 62 records
   unchanged (source_locators 7, evidence_records 14, knowledge_claims 9,
   relationship_assertions 16, coverage_gaps 4), every `COUNT_*` PASS,
   `DERIVED_SOURCES=2 se:SRC-CS2023,se:SRC-SWEBOK-GUIDE-4-0A`,
   `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
   `LINKML_INSTANCE_VALIDATION=PASS` (1.12.1/1.12.1), `LANE_VALIDATION=PASS`.

Clean worktree before this receipt: 0 dirty. The identical command set is
re-run at the committed final HEAD (logs `selftest-post.log`,
`lane-validate-post.log`, exit codes recorded in the delivery comment), and the
consolidated R2 delivery comment on coordinator #321 binds to the final
HEAD/TREE: the four-path SHA256 manifest at the final HEAD, the post-commit
exit codes, clean-worktree-after evidence, and `CROSS_LANE_CHECK` at the three
R2-final tips. No LOCAL_VALIDATION-only task exists; no record is ACCEPTED;
ISO/IEC/IEEE 42010 remains metadata-only; HUMAN_REVIEW_BURDEN remains
NOT_MEASURED.

## R3 ownership/dispatch readback receipt — Claim/CAS NOT_PRESENT (2026-11-06)

R3 review 5801169940 (P2-01) and handoff #321:6687306225 require an
independently readable Claim/CAS identity, numeric `claim_generation`, status,
owner/lease readback and Execution Pack / Dispatch ID — explicitly not
inferable from Owner comments, Task Pack issue numbers, PR authorship or
`--force-with-lease`. This receipt records the concrete authoritative readback
performed for this batch (initial HEAD `fc19da5ed5baa346457ea0cd0057d7db661e403c`
/ tree `057f329507f198619df2d31dce34b320e1148eb7` / base
`7a70211b4460e4c753c97798670de9f34861d4a4`; PR #328 OPEN, headRefOid == branch
ref == local HEAD, worktree clean 0/0 immediately before this write). It
supersedes the R2 receipt's Owner-comment "claim chain" framing, which remains
above as history.

### Authoritative store query (reproducible; log `r3-claim-readback/store-sweep.log`)

The workspace's tally Claim/CAS authority is the host single-writer claim
controller family: `openSqliteCasStore` SQLite registries (records
`tally:v2:claim-admission:<sha>` with `protectedClaimKey`
`kaicreator-mm/tally#<issue>:<role>:__default__`, numeric `generation`, phase,
`dispatchId`, `writerId`) plus its GitHub-native publication channel
(`<!-- ads:github -->` claim-event comments per
`packages/tally-github-native-c/src/task-claim-github-publisher.mjs`).
Queried on 2026-11-06:

1. Every accessible registry enumerated and each claim record's
   protectedClaimKey read: `_tmp/tally-alm-claims/registry-{197,198,199,200,
   202,203,207,208,209,210,211,212,219}.sqlite`,
   `_tmp/i319-claim/run-20261106T043955/registry-319.sqlite`,
   `_tmp/i319-terminalization/*.bak`,
   `tally-host-state/release-{claim-222,operations}.sqlite` — 15 claim
   admissions total, all bound to tally issues #197–#212, #219, #319 and the
   #222 release chain. **None is bound to #321, #322, #323 or #324.**
2. Byte-level scan of all those registries for `#32x` / `321` / `sek06` key
   material: `NO_MATCH_IN_ANY_REGISTRY`.
3. GitHub-native claim-event channel: all comments on #321/#322/#323/#324
   paginated and scanned for the `<!-- ads:github -->` marker: **0 matches**.
4. Workspace sweep for claim/ledger stores modified since 2026-11-05: no store
   covering the v2.1.1 knowledge workstream exists (SEK-02R #325 session
   artifacts likewise contain none).

### Concrete readback values (explicit; no substitution)

- Claim ID / CAS key: **NOT_PRESENT** — no `tally:v2:claim-admission:*` record
  bound to `kaicreator-mm/tally#323` (or the SEK-06 lanes) exists in any
  accessible registry. The stores are readable, so this is an authoritative
  NOT_PRESENT, not an inaccessible/UNKNOWN store.
- `claim_generation`: **NOT_PRESENT** (no protected claim key exists; numeric
  generation undefined — deliberately not recorded as "unchanged").
- Claim status / Task-Pack claim binding / owner/lease/fencing tokens:
  **NOT_PRESENT** (none exist; no secret tokens to withhold).
- Execution Pack / Dispatch / attempt ID with target SHA and state:
  **NOT_PRESENT** — no dispatch record exists in any store; the R1/R2/R3
  coordination comments on #321 are Owner handoffs, not claim events, and per
  the R3 finding are not claimed here as claim evidence.
- Exclusive-Writer mechanism actually in force: single-writer discipline with
  git `--force-with-lease` fencing (expected old head re-read immediately
  before push) and live `headRefOid` == branch-ref == local-HEAD readback per
  write. This is fencing evidence only; it is not represented as Claim/CAS
  identity.

### Authoritative ownership mechanism and ADS basis for this receipt

Write authorization for this lane is the Owner-frozen Task Pack #323 (original
four-path write set) exercised through the coordinator #321 combined
Builder/Repairer handoffs, with single-writer fencing as above; the SEK-06
workstream has never instantiated Claim/CAS objects. Per the R3 handoff rule 4,
this NOT_PRESENT receipt is appended because the existing ADS protocol
explicitly permits it: ADS #1359 v4.19 §10.8 final paragraph — Claim/event
fields "暂不可用时，可在结构化回执中明确表达，不应阻塞真实交付" (when the
fields are unavailable, state so explicitly in a structured receipt; do not
block real delivery) — and Task Pack #323 §"Independent review and repair"
authorizes reviewer-repairs within the four paths including updating this
observation. ADS acceptance criterion 7 (Owner ordinary instructions must not
bypass Claim/CAS constraints) is respected: no Claim/CAS object exists to
bypass, and **no new Task, Pack, Claim, Dispatch or repair generation was
created** for this batch. Knowledge records, lifecycle/review states and all
non-observation paths are untouched; this is an observation-only append.

### Validation at the resulting exact HEAD (Windows-native venv Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1, PYTHONDONTWRITEBYTECODE=1)

Run after this receipt commit (records.yaml unchanged by this batch; per
handoff rule 7 the applicable checks are rerun and rebound at the new SHA):
`git diff --check` exit 0; `validate_knowledge_graph.py --self-test` →
`SELF_TESTS=66/66` exit 0; `assemble_and_validate.py <worktree> 06B …` →
exit 0 with every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS` (62 records; DERIVED_SOURCES=2); clean worktree 0/0;
four-path SHA256 manifest at the new HEAD and one cross-lane check against all
three R3-final tips are bound in the R3 delivery comment on coordinator #321
(receipt dir `_tmp/tally-sek06-lanes/validation/06B-r3/`). Builder/Repairer
writing stops after this single combined batch; final Exact-SHA verification
belongs to an independent Verifier.
