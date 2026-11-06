# SEK-06C — Method Observation (Testing, Verification and Validation lane)

**Task:** #324 (SEK-06C lane builder; refs #321, #299)
**Date:** 2026-11-06
**Worktree:** `tally-i324-zcode`, branch `task/tally-v211-sek06c-testing-verification`
**Base HEAD:** `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`
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

## Final-tip receipt — Builder batch `a1ef094` (2026-11-06 R2 repair append)

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
