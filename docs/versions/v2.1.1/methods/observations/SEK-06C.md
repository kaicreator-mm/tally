# SEK-06C — Method Observation (Testing, Verification and Validation lane)

**Task:** #324 (SEK-06C lane builder; refs #321, #299)
**Date:** 2026-11-06
**Worktree:** `tally-i324-zcode`, branch `task/tally-v211-sek06c-testing-verification`
**Base HEAD:** `7a70211b4460e4c753c97798670de9f34861d4a4` (after the Owner-authorized 2026-11-06 WIP rebase; original authoring base `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`)
**Status:** EXECUTED / EXPERIMENTAL / HUMAN_REVIEW_PENDING
**Write set (exactly four files):** lane `README.md`, `records.yaml`,
`EXPLANATIONS.md` (under
`docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/`)
and this observation file. No shared taxonomy, registry, model or CI path was
modified.

## Research path

### CS2023 (`se:SRC-CS2023`)

- The official Body of Knowledge PDF
  (`https://csed.acm.org/wp-content/uploads/2024/04/3.1-Body-of-Knowledge-1.pdf`)
  could not be downloaded directly from this network: the TLS connection was
  reset on both direct and proxied routes, and the Internet Archive snapshot
  was rate-limited (HTTP 429). Full text was therefore obtained through a
  text-extraction service on 2026-11-06.
- Consequence: **no binary SHA-256 is recorded for the CS2023 PDF**. Page
  anchors for this lane are limited to (a) registry-verified anchors (SE
  Overview = PDF page indices 174-176; Core Hours table = 176-177; SE-Tools =
  PDF page index 178, one-based 179) and (b) heading/topic-number anchoring
  for the new KU-level locator `se:LOC-CS2023-SE-VALIDATION-KU`, which asserts
  no page number of its own. The printed page token `175` adjacent to the SE
  heading in the text stream is consistent with the registry anchors but was
  not treated as independent confirmation.
- Verbatim material retained: only short topic titles and the two V&V
  questions needed to anchor locators and verify topic identity; all corpus
  statements are concise original paraphrases (extract file:
  `_tmp/tally-sek06-lanes/extract/cs2023-se-verified-anchors.md`).

### NIST SP 800-218 v1.1 (`se:SRC-NIST-SP-800-218-V1-1`)

- The official PDF was downloaded locally on 2026-11-06 from
  `https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-218.pdf`.
- **SHA-256:**
  `617746e553a9e2da49bfbd4eef0dfc3094758a39b869314e4173ac36605cde22`
  (739891 bytes, 36 PDF pages).
- Identity verified against the title page (exact SSDF v1.1 title, authors
  Souppaya/Scarfone/Dodson, February 2022, NIST SP 800-218); pagination
  matches the registry dossier (groups at PDF page index 12 = printed page 4;
  risk guidance at PDF page index 11 = printed page 3).
- Page-level anchors verified in this research (extract files:
  `extract/nist-218-verified-anchors.md`, `extract/nist-pw-rv.txt`,
  `extract/nist-table1.txt`): PW.7 (PDF index 22 / printed 14, tasks
  PW.7.1-PW.7.2), PW.8 (PDF index 23 / printed 15, tasks PW.8.1-PW.8.2 with
  Examples 3/5), RV.1 (24/16, incl. the RV.1.2 "See PW.7 and PW.8"
  cross-reference), RV.2 (25/17), RV.3 (26-27/18-19).
- Locator page convention: "PDF page index N" is zero-based; printed page is
  the number on the page. All NIST lane locators are `LOCATOR_VERIFIED`
  against the local PDF, `accessed_on: 2026-11-06`.

### Interpretive boundaries enforced in records

- SWEBOK v4.0a hold (#325 not merged): no SWEBOK KA5 citation anywhere; the
  hold is recorded as `se:GAP-TVV-SWEBOK-KA5-HOLD`.

  *(Historical authoring-time statement. Current status after the 2026-11-06
  P2 repair: #325 merged via PR #326 into the WIP base; identity/registry
  reconciliation complete; KA5 sections 1/2/3 reinspected directly and cited
  at content level — see the P2 repair receipt at the end of this
  observation.)*
- NIST content is used only as a secure-development cross-cutting source:
  risk/cost/feasibility/applicability govern practice selection, SSDF is not a
  checklist, Table 1 order implies no implementation sequence, and notional
  examples are neither required nor exhaustive (qualifiers on
  `se:REL-TVV-0016`, limitations on `se:CLM-TVV-0004`, and the
  QUALIFIED_BY/limitation pattern throughout).
- No real engineering cases were invented: both sources lack real case
  evidence, so no Case/Counterexample records exist; the regression pattern
  stays a candidate and the coverage-theater anti-pattern is derived from
  explicitly marked synthesis evidence (`se:EVD-TVV-0014`,
  `se:REL-TVV-0019`).
- CS2023's registered permission scope is educational materials/programs;
  Tally's general engineering knowledge graph is not presumed to qualify.
  This was corrected as the first-class rights gap
  `se:GAP-CS2023-RIGHTS-CLEARANCE`. No license clearance is claimed.
- Per issue #324: test execution is never equated with correctness and
  validation is never equated with acceptance; the distinction is carried in
  `se:CON-TVV-VERIFICATION-VALIDATION`, `se:CLM-TVV-0001` limitations, the
  anti-pattern causes, and `EXPLANATIONS.md`.

## Locator verification method

1. Registry-first: only registry-verified locators may carry CS2023 page
   anchors; KU-level locators are heading-anchored (exact unit heading + topic
   numbering verified against the 2026-11-06 full-text extraction).
2. NIST locators verified page-by-page from the SHA-256-recorded local PDF
   (both zero-based PDF index and printed page recorded; practice/task/example
   text matched verbatim before paraphrasing).
3. Every evidence `statement` is a concise original paraphrase of verbatim
   extracted content or of a registry-verified fact; nothing else was admitted.

## Validation results (2026-11-06, Windows native, exact tip `293efbc` — historical receipt)

Environment: Python 3.14.7, PyYAML 6.0.3; LinkML 1.12.1 / linkml-runtime
1.12.1 (SEK venv, `PYTHONDONTWRITEBYTECODE=1`). Validator compile check:
`py_compile validate_knowledge_graph.py` — exit 0.

Validator self-test:
`SELF_TESTS=66/66` (exit 0).

Lane assembly + validation
(`assemble_and_validate.py <worktree> 06C <records.yaml> <outdir>`), rerun at
the then-exact committed final tip after the rights-gap correction and the
documentation-only follow-up (historical receipt — superseded as tip by the
Builder-batch commit `a1ef094`, whose final-tip receipt is appended at the end
of this file) — worktree HEAD
`293efbc46299b0827e426e5708c7d2b9d8c38893`, tree
`e99afa5b47eebac710e76a6a3ea6ddfbce6b0e56`, base
`cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #329
headRefOid equals this HEAD; clean worktree before 0 dirty / after 0 dirty;
exit 0). The earlier rerun at
`2b8464a348afd982d5c4a0a4793cb7b390cce906` / tree
`21f0484e6a4fe1385b2270ed9356760bc9f0110c` (`records.yaml` byte-identical to
the final tip) is retained as historical context:

```text
LANE_RECORD_COUNTS={"anti_patterns":1,"concepts":2,"coverage_gaps":4,"decision_models":1,"domains":1,"evidence_records":15,"knowledge_claims":7,"patterns":1,"practices":2,"principles":1,"relationship_assertions":20,"source_locators":9,"taxonomy_nodes":3,"taxonomy_views":2}
COUNT_source_locators=9 MIN=3 PASS
COUNT_evidence_records=15 MIN=3 PASS
COUNT_knowledge_claims=7 MIN=4 PASS
COUNT_concepts=2 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=2 MIN=2 PASS
COUNT_patterns=1 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=4 MIN=1 PASS
COUNT_relationship_assertions=20 MIN=10 PASS
DERIVED_SOURCES=2 se:SRC-CS2023,se:SRC-NIST-SP-800-218-V1-1
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

Exact-tip receipt directories (outside the repo write set): the
final-tip receipt at `293efbc` is `_tmp/tally-sek06-lanes/validation/06C-bld3/`
(`identity-tip.txt`, `selftest-tip.log`, `lane-validate-tip.log`,
`manifest-four-paths-tip.sha256`; post-commit rerun and manifest at the
pushed HEAD: `selftest-post.log`, `lane-validate-post.log`,
`manifest-four-paths-post.sha256`). It supersedes the receipts at
`06C-bld2` (the rerun at the `2b8464a`-lineage tip after the
documentation-only follow-up commit) and at `06C` (pre-correction,
`coverage_gaps=3` before `se:GAP-CS2023-RIGHTS-CLEARANCE` was added); those
directories are retained as history only.

Four-path SHA256 manifest at the committed tip `293efbc` (tree `e99afa5`;
repo-relative; the receipt-rebind commit that carries this receipt changes
only this observation file, so the first three hashes remain valid at the
pushed HEAD):

```text
4e7c733e934d4e937213b25b55ee71874a4a446f2eaeaaa293bbab25e466b447  lanes/testing-verification/README.md
2d66085c5357b783a0dfa9c8a4fc3bab8173e00c3d954ece2c3c1ca26d24a468  lanes/testing-verification/records.yaml
fdb49b03410ea38bf0cce7288969d44b4c833fc837cf2c0056994e5b38a87445  lanes/testing-verification/EXPLANATIONS.md
c54985a0562f9720ad31818116ae2931d469b3ab989fb56132652f9e760aaa01  methods/observations/SEK-06C.md
```

Machine validation is not human review: every record remains
PROPOSED/PENDING_REVIEW with evidence EXTRACTED_PENDING_REVIEW.

## Deviations and notes

- `se:LOC-NIST-218-RISK` (registry-verified Section 1 anchor) was added
  beyond the minimum locator list because the required QUALIFIED_BY edge needs
  risk-applicability evidence; its locator string matches the registry
  dossier anchor exactly.
- The `se:LOC-CS2023-SE-VALIDATION-KU` locator is heading-anchored only (no
  page assertion) — enforced because the CS2023 PDF lacks a recorded binary
  hash this round.
- Shared snippets (`se:DOMAIN-SOFTWARE-ENGINEERING`, `se:VIEW-CS2023-SE`, the
  three shared locators) are byte-identical copies from
  `_tmp/tally-sek06-lanes/extract/shared-snippets.yaml` for SEK-08
  deduplication.

## Qualification state

```ini
SEK_06C_LANE=EXECUTED
GRAPH_VALIDATION=PASS
LINKML_INSTANCE_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
HUMAN_REVIEW_BURDEN=NOT_MEASURED
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```


## Rights-clearance correction

The initial README wording could be read as assuming that the CS2023 educational
permission covers this project's use. That inference is not supported by the
registry's source-specific rights note. The revised lane treats the permission
scope as unresolved for a general engineering knowledge graph and records a
HIGH/OPEN REVIEW_GAP. The earlier lane-validation receipt predates this
four-path correction and must not be used as validation evidence for the new
HEAD; exact-tip native validation was first performed at HEAD
`2b8464a348afd982d5c4a0a4793cb7b390cce906` (historical receipt
`_tmp/tally-sek06-lanes/validation/06C-bld2`) and is now bound to the final
committed tip `293efbc46299b0827e426e5708c7d2b9d8c38893` / tree
`e99afa5b47eebac710e76a6a3ea6ddfbce6b0e56` (receipt
`_tmp/tally-sek06-lanes/validation/06C-bld3`), passing in both runs
(`LANE_VALIDATION=PASS`, `coverage_gaps=4`).

## Historical final-tip receipt — Builder batch `a1ef094` (2026-11-06 R2 repair append; old WIP base, superseded by the 2026-11-06 WIP rebase onto `7a70211`)

The R2 Exact-SHA review of PR #329 required the observation to name the final
Builder-batch tip; the `293efbc` receipt above (and the earlier `2b8464a`
receipt) are historical context, preserved unchanged.

- Identity: HEAD `a1ef094c00c45a32f3dfa11663ad251d2bd23ed3`, tree
  `fb925ea6878b6531da7ee8febff845c356387344`, base
  `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #329
  headRefOid equalled this HEAD at the R2 review).
- Commands and exit codes at this exact tip: `validate_knowledge_graph.py
  --self-test` → `SELF_TESTS=66/66`, exit 0; `assemble_and_validate.py
  <worktree> 06C <records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (counts unchanged:
  69 lane records, 20 relationship assertions, `coverage_gaps=4`, 2 derived
  sources `se:SRC-CS2023` + `se:SRC-NIST-SP-800-218-V1-1`). Clean worktree
  before/after: 0/0 dirty. Logs: Builder batch
  `_tmp/tally-sek06-lanes/validation/06C-bld3/` (`identity-post.txt`,
  `selftest-post.log`, `lane-validate-post.log`,
  `manifest-four-paths-post.sha256`), independently re-run for this repair in
  `_tmp/tally-sek06-lanes/validation/06C-r2r/`.
- Four-path SHA256 manifest at HEAD `a1ef094` / tree `fb925ea`:

```text
4e7c733e934d4e937213b25b55ee71874a4a446f2eaeaaa293bbab25e466b447  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/README.md
2d66085c5357b783a0dfa9c8a4fc3bab8173e00c3d954ece2c3c1ca26d24a468  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/records.yaml
fdb49b03410ea38bf0cce7288969d44b4c833fc837cf2c0056994e5b38a87445  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/EXPLANATIONS.md
44141dfa4a5616b2fbc3bb4c8fa8446ca02b7fe80933937e853283e6efd507ff  docs/versions/v2.1.1/methods/observations/SEK-06C.md
```

This append itself is an observation-only R2-repair commit and becomes the
newest tip; its exact HEAD/TREE, the post-commit rerun (same commands, exit
0), the updated four-path manifest and clean-worktree evidence are delivered
in the R2 repair delivery comment on PR #329 / coordinator #321, with logs in
`_tmp/tally-sek06-lanes/validation/06C-r2r/`. Preserved unchanged by this
receipt: the CS2023 PDF binary SHA-256 remains **NOT_OBTAINED**
(text-extraction provenance, heading-anchored compensating control), the NIST
PDF SHA-256
`617746e553a9e2da49bfbd4eef0dfc3094758a39b869314e4173ac36605cde22`
(739,891 bytes) and page-level anchors, the shared HIGH/OPEN CS2023
rights-clearance gap, and the SWEBOK hold (#325/#326 unmerged).

## WIP-rebase receipt — Owner-authorized rebase onto `7a70211` (2026-11-06)

Owner authorization recorded on #321 (2026-11-06) merged PR #326 (SEK-02R)
into `wip/v2.1.1-knowledge` — linear rebase merge, no merge commit — moving
the WIP head from `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (tree
`da5deb5eac25cd019883772967d288a1f2679a84`) to
`7a70211b4460e4c753c97798670de9f34861d4a4` (tree
`f9b01c74f39993299d9ee179c76cdc08d3baae9d`). This lane branch was then
rebased with `git rebase --onto 7a70211b4460e4c753c97798670de9f34861d4a4
cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`: all six lane commits replayed
with **zero conflicts**, and the three lane content files are byte-identical
to the pre-rebase reviewed tip `8fa8cd2385690bbbf96cc34cacb5e35c8749d29c`
(verified by path-scoped diff against a pre-rebase backup ref). The earlier
receipts above (final-tip `a1ef094` and its predecessors, all on the old
base) are **historical** and preserved unchanged.

- Identity at the rebased tip: HEAD `343f30b91bc38e6ccdaca42103c5b201353faa3e`,
  tree `7e01679b45acbf0ffad0abe6b0555d7e9d3bbd00`, base / merge-base
  `7a70211b4460e4c753c97798670de9f34861d4a4`; clean worktree before/after:
  0/0 dirty.
- Validation rerun at that exact tip (Windows-native venv Python 3.14.7 /
  PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1,
  `PYTHONDONTWRITEBYTECODE=1`): `validate_knowledge_graph.py --self-test` →
  `SELF_TESTS=66/66`, exit 0; `assemble_and_validate.py <worktree> 06C
  <records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (counts
  unchanged: 69 lane records, 20 relationship assertions, `coverage_gaps=4`,
  2 derived sources `se:SRC-CS2023` + `se:SRC-NIST-SP-800-218-V1-1`).
  Cross-lane check across the three rebased tips (06A `944cfee` /
  06B `eff5e86` / 06C `343f30b`): `CROSS_LANE_CHECK=PASS` — 6 shared
  records semantically identical in all three lanes, 148 unique lane IDs,
  merged three-lane graph `GRAPH_VALIDATION=PASS` +
  `NO_ACCEPTED_RECORDS=PASS` + LinkML PASS. Logs:
  `_tmp/tally-sek06-lanes/validation/06C-rb1/`
  (`selftest-tip.log`, `lane-validate-tip.log`, exit receipts) and
  `_tmp/tally-sek06-lanes/validation/cross-lane-rb1-tip.log` (all exit 0).
- Content manifest at the rebased tip (three content files unchanged from
  the R2-reviewed `a1ef094` / `8fa8cd2` manifest — the rebase changed only
  base history, not lane content):

```text
4e7c733e934d4e937213b25b55ee71874a4a446f2eaeaaa293bbab25e466b447  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/README.md
2d66085c5357b783a0dfa9c8a4fc3bab8173e00c3d954ece2c3c1ca26d24a468  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/records.yaml
fdb49b03410ea38bf0cce7288969d44b4c833fc837cf2c0056994e5b38a87445  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/testing-verification/EXPLANATIONS.md
```

- SWEBOK state after the rebase: the corrected SWEBOK v4.0a source record
  (SEK-02R / PR #326, including the native PDF SHA-256 and the open
  title-page/landing-page date discrepancy) is now part of this branch's
  base, and the source-identity hold is lifted at the registry level.
  **No SWEBOK-derived claim, locator or evidence was added by this rebase**:
  direct SWEBOK reinspection was not performed in this stage, and every
  knowledge record remains PROPOSED / PENDING_REVIEW. Preserved unchanged:
  the CS2023 PDF binary SHA-256 **NOT_OBTAINED** limit, the NIST PDF
  SHA-256 `617746e5...` and page-level anchors, and the shared HIGH/OPEN
  CS2023 rights-clearance gap.
- This append itself is an observation-only rebase-receipt commit and
  becomes the newest tip; its exact HEAD/TREE, the post-commit rerun (same
  commands, exit 0), the updated four-path SHA256 manifest and
  clean-worktree evidence are delivered in the rebase delivery comment on
  PR #329 / coordinator #321, with logs in
  `_tmp/tally-sek06-lanes/validation/06C-rb1-post/`.

## P2 repair receipt — SWEBOK KA5 reinspection + stale-hold correction (2026-11-06)

Repair batch executed under the combined handoff on #321 (R1 review of the
post-WIP-rebase tips, both findings P2), within the original #324 four-path
write set, against reviewed tip `858d3d7604e598b41178119859f7dd24356faa10` /
tree `70e3336124a350b7facb6c49b10e52d8fb1b802c`, base
`7a70211b4460e4c753c97798670de9f34861d4a4`. Pre-write readback: PR #329 OPEN,
headRefOid equalled the branch ref and the reviewed tip; worktree clean; no
concurrent writer observed; write set unchanged.

### P2-01 — stale source-hold statements corrected

README.md, records.yaml and EXPLANATIONS.md no longer state that #325 is not
merged or that SWEBOK is on hold. Current-state wording now says:
identity/registry reconciliation complete (#326 merged into this lane's WIP
base); content-level reinspection performed by this repair;
`se:GAP-TVV-SWEBOK-KA5-HOLD` preserved as a stable ID but redescribed as a
content/evidence coverage gap (KA5 topics beyond sections 1/2/3 uninspected;
no human review of SWEBOK-derived records), still MEDIUM/OPEN. The shared
Domain and View records' hold language was reworded **identically in all
three lane files** (06A/06B/06C) so the six shared records remain
byte-identical for SEK-08. Historical statements in this observation are
preserved and labeled. The ISO/IEC/IEEE 29119-2 metadata-only boundary and the
CS2023 PDF SHA-256 NOT_OBTAINED disclosure are untouched.

### P2-02 — SWEBOK KA5 locator re-verification performed

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
| `se:LOC-SWEBOK-KA5-FUNDAMENTALS` | Ch.5 §1 Software Testing Fundamentals (§1.1 Faults vs. Failures, §1.2.7 The Oracle Problem, §1.2.8 Theoretical and Practical Limitations) | 118–120 | 5-3 to 5-5 |
| `se:LOC-SWEBOK-KA5-LEVELS-OBJECTIVES` | Ch.5 §2 Test Levels (§2.1 The Target of the Test incl. 2.1.1–2.1.4; §2.2 Objectives of Testing) | 121–122 | 5-6 to 5-7 |
| `se:LOC-SWEBOK-KA5-TECHNIQUES` | Ch.5 §3 Test Techniques (intro; §3.1 Specification-Based; §3.2 Structure-Based; §3.3 heading) | 125–128 | 5-10 to 5-13 |

Chapter 5 opens at PDF page index 116 (printed 5-1). Added records: the three
locators above, three `DIRECT_SOURCE_PARAPHRASE` evidence records
(`se:EVD-TVV-SWEBOK-FAULTS-ORACLES-LIMITS`,
`se:EVD-TVV-SWEBOK-LEVELS-OBJECTIVES`,
`se:EVD-TVV-SWEBOK-TECHNIQUE-CLASSES`), three claims
(`se:CLM-TVV-SWEBOK-FAULT-FAILURE-LIMIT`,
`se:CLM-TVV-SWEBOK-LEVELS-NO-PROCESS`,
`se:CLM-TVV-SWEBOK-TECHNIQUE-CLASSES`) and three canonical `SUPPORTED_BY`
edges (`se:REL-TVV-0021/0022/0023`). No page number, section or claim was
written that the extracted PDF text does not support; the Guide's
generally-accepted-knowledge-not-prescription limitation is carried inside
each claim, and the Dijkstra-attributed presence-not-absence point is
paraphrased, not quoted. Rights boundaries kept: PDF not
committed/mirrored/redistributed, original paraphrases only, release-date
discrepancy remains OPEN, all records PROPOSED/PENDING_REVIEW, zero ACCEPTED,
ISO 29119-2 metadata-only preserved.

### Validation (Windows-native, Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1, PYTHONDONTWRITEBYTECODE=1)

Run on the exact repaired working tree (= the committed tree of this repair
commit); rerun after the commit and the final four-path SHA256 manifest at the
pushed HEAD are delivered in the P2 repair delivery comment on PR #329 /
coordinator #321, logs in `_tmp/tally-sek06-lanes/validation/06C-p2/`.

1. `validate_knowledge_graph.py --self-test` → `SELF_TESTS=66/66`, exit 0.
2. `assemble_and_validate.py <worktree> 06C <records.yaml> <outdir>` → exit 0:
   every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
   `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (81 lane records:
   `source_locators=12`, `evidence_records=18`, `knowledge_claims=10`,
   `relationship_assertions=23`, `coverage_gaps=4`; derived sources
   `se:SRC-CS2023` + `se:SRC-NIST-SP-800-218-V1-1` +
   `se:SRC-SWEBOK-GUIDE-4-0A`). 69 → 81 lane records (three content files plus
   this observation are the only paths changed).

Cross-lane check at the three repaired tips and clean-worktree evidence are
recorded in the delivery comment. Human review burden remains NOT_MEASURED; no
record is ACCEPTED.

## R2 consolidated delivery receipt — evidence closure (2026-11-06)

R2 review finding (#329, P2-01): the P2 receipt promised a delivery comment that
was not posted, and the receipt did not identify the active Claim/CAS identity,
generation/status or exclusive-Writer readback. This receipt closes that gap
for SEK-06C; per the R2 handoff, no source-content change was indicated for
this lane, so this commit changes **only this observation file** (write set
intact).

### Claim/CAS identity, generation/status and exclusive-Writer readback (pre-write)

- Active claim chain for this repair batch: Owner-authorized combined
  Builder/Repairer handoffs on coordinator #321 — P2 stage comment 6684006230
  and R2 checkpoint comment 6686070478 — plus the R2 Repairer dispatch to the
  local executor on 2026-11-06. Status: ACTIVE for this batch; it authorizes
  exactly the in-write-set R2 repairs and receipt closure. No new Task, Pack or
  Claim was created for R2 (forbidden by the handoff); generation is unchanged:
  the original Task Pack issue #324 and its four-path write set remain the sole
  authority, re-read before writing.
- Exclusive Writer readback (2026-11-06, immediately before writing): PR #329
  OPEN; live PR headRefOid `1cbe13783cbd548fef59ef1d65e6dd494b91d45a` == branch
  ref `task/tally-v211-sek06c-testing-verification` == R2-reviewed HEAD; local
  worktree at that HEAD with 0 dirty paths; no other writer observed on the
  branch since the P2 push (remote head unchanged at readback). Push after
  commit uses `--force-with-lease` with expected remote head `1cbe1378…`,
  fencing any concurrent writer.
- Identity bound by this receipt: R2-reviewed HEAD
  `1cbe13783cbd548fef59ef1d65e6dd494b91d45a` / tree
  `58c983e088021684a820b95db4ea9251ceed4005` / base (merge-base)
  `7a70211b4460e4c753c97798670de9f34861d4a4`.

### Validation — actual commands and exit codes (repaired tree = this commit's tree; Windows-native venv Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1, PYTHONDONTWRITEBYTECODE=1)

Receipt dir `_tmp/tally-sek06-lanes/validation/06C-r2/` (`identity-pre.txt`,
`selftest-pre.log`, `lane-validate-pre.log`); `git diff --check` exit 0.

1. `…/knowledge/model/validate_knowledge_graph.py --self-test` → exit 0,
   final line `SELF_TESTS=66/66`.
2. `…/tally-sek06-lanes/scripts/assemble_and_validate.py <worktree> 06C
   <records.yaml> <outdir>` → exit 0: `LANE_RECORD_COUNTS` 81 records
   (source_locators 12, evidence_records 18, knowledge_claims 10,
   relationship_assertions 23, coverage_gaps 4), every `COUNT_*` PASS,
   `DERIVED_SOURCES=3 se:SRC-CS2023,se:SRC-NIST-SP-800-218-V1-1,se:SRC-SWEBOK-GUIDE-4-0A`,
   `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
   `LINKML_INSTANCE_VALIDATION=PASS` (1.12.1/1.12.1), `LANE_VALIDATION=PASS`.

Clean worktree before this receipt: 0 dirty. The identical command set is
re-run at the committed final HEAD (logs `selftest-post.log`,
`lane-validate-post.log`, exit codes recorded in the delivery comment), and the
consolidated R2 delivery comment on coordinator #321 binds to the final
HEAD/TREE: the four-path SHA256 manifest at the final HEAD, the post-commit
exit codes, clean-worktree-after evidence, and `CROSS_LANE_CHECK` at the three
R2-final tips. No LOCAL_VALIDATION-only task exists; no record is ACCEPTED;
ISO/IEC/IEEE 29119-2 remains metadata-only and the CS2023 PDF hash
NOT_OBTAINED disclosure is retained; HUMAN_REVIEW_BURDEN remains NOT_MEASURED.


## R3 ownership/dispatch readback receipt — Claim/CAS NOT_PRESENT (2026-11-06)

R3 review 5801171081 (P2-01) and handoff #321:6687306225 require an
independently readable Claim/CAS identity, numeric `claim_generation`, status,
owner/lease readback and Execution Pack / Dispatch ID — explicitly not
inferable from Owner comments, Task Pack issue numbers, PR authorship or
`--force-with-lease`. This receipt records the concrete authoritative readback
performed for this batch (initial HEAD `7564e94569a6d6b5a2449f0fb5ee4d31ea95ce25`
/ tree `fc50580bb4d3268d91e496031b28a4f87fecabe7` / base
`7a70211b4460e4c753c97798670de9f34861d4a4`; PR #329 OPEN, headRefOid == branch
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
  bound to `kaicreator-mm/tally#324` (or the SEK-06 lanes) exists in any
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

Write authorization for this lane is the Owner-frozen Task Pack #324 (original
four-path write set) exercised through the coordinator #321 combined
Builder/Repairer handoffs, with single-writer fencing as above; the SEK-06
workstream has never instantiated Claim/CAS objects. Per the R3 handoff rule 4,
this NOT_PRESENT receipt is appended because the existing ADS protocol
explicitly permits it: ADS #1359 v4.19 §10.8 final paragraph — Claim/event
fields "暂不可用时，可在结构化回执中明确表达，不应阻塞真实交付" (when the
fields are unavailable, state so explicitly in a structured receipt; do not
block real delivery) — and Task Pack #324 §"Independent review and repair"
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
`SELF_TESTS=66/66` exit 0; `assemble_and_validate.py <worktree> 06C …` →
exit 0 with every `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`,
`NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
`LANE_VALIDATION=PASS` (81 records; DERIVED_SOURCES=3); clean worktree 0/0;
four-path SHA256 manifest at the new HEAD and one cross-lane check against all
three R3-final tips are bound in the R3 delivery comment on coordinator #321
(receipt dir `_tmp/tally-sek06-lanes/validation/06C-r3/`). Builder/Repairer
writing stops after this single combined batch; final Exact-SHA verification
belongs to an independent Verifier.
