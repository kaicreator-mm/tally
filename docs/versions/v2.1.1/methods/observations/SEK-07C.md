# SEK-07C — Method Observation (Security Engineering lane)

**Task:** #336 (coordinator #333, parent pilot #299)
**Date:** 2026-11-07
**Branch:** `task/tally-v211-sek07c-security-engineering` from exact base
`7a70211b4460e4c753c97798670de9f34861d4a4` (not stacked; merge-base equals the base).
**Status:** EXECUTED / LOCAL_VALIDATION_PASS / HUMAN_REVIEW_PENDING
**HUMAN_REVIEW_BURDEN=NOT_MEASURED** (no human measured review effort for this lane).

## Claim / CAS / dispatch readback (per Task Pack step 2)

- `gh issue view 336` readback: state OPEN, title "[v2.1.1][07C] Security Engineering",
  labels `[]`, assignees `[]`, comments `[]` (0 comments). No claim/generation state
  is recorded on the issue.
- Worktree claim-file search: no `docs/**/claims/` directory exists; the only
  "claim" hit is the unrelated v1.4 research artifact
  `docs/tasks/v1.4/research/A0-01-claim-register.md` (predates SEK-07; not a lane
  Claim/CAS dispatch authority).
- Conclusion: **Claim/Dispatch state = NOT_PRESENT** (authoritative readback values:
  no claim ID, no generation, no status). No Claim or Dispatch records were created
  by this lane; no ownership conflict was detected.

## Source research and locator verification (all locators directly inspected 2026-11-07)

1. **SWEBOK Guide v4.0a KA13 Software Security** — official PDF
   `_tmp/tally-sek07-lanes/sources/swebok-v4-bust.pdf`, SHA-256 re-verified
   `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511` (411 pages;
   matches registry). Chapter starts at 1-based PDF page 264 (0-based index 263);
   full chapter extracted and read (pages 264-273, printed 13-1 to 13-10) with
   pypdf 7.0.0 via `extract_pages.py`. Admitted locators (chapter/section-scoped):
   section 1 with 1.1/1.2 (index 263, printed 13-1), 4.1 (indices 265-266, printed
   13-3 to 13-4), 4.2 + 4.3 (index 266, printed 13-4), 4.4 + 4.5 (indices 266-267,
   printed 13-4 to 13-5), 4.6 + 5.1 (indices 267-268, printed 13-5 to 13-6).
   Printed folios confirmed from the page headers of the extraction.
2. **NIST SP 800-218 v1.1 SSDF** — official PDF
   `_tmp/tally-sek07-lanes/sources/nist-sp-800-218.pdf`, SHA-256 verified
   `617746e553a9e2da49bfbd4eef0dfc3094758a39b869314e4173ac36605cde22` (739,891
   bytes, 36 pages; matches brief). Pages 13-21 extracted and read. Admitted
   locators: practice PO.1 tasks PO.1.1-PO.1.3 (indices 13-14, printed 5-6), PW.1
   tasks PW.1.1-PW.1.3 (index 19, printed 11), PW.2 task PW.2.1 (index 20, printed
   12). Verified against the SEK-06C lane's already-verified NIST locators (PW.7,
   PW.8, RV.1-RV.3, Section 1) — no re-derivation; all four admitted sections are
   different practice rows.
3. **CS2023 SEC knowledge area** — verbatim selections in
   `_tmp/tally-sek07-lanes/sources/cs2023-bok-sek07-extract.md` (extraction method
   and page-anchor limitations stated in that file). Two heading-anchored locators
   admitted: SEC-Engineering (CS Core 1/3/4; KA Core 5/6/9) and SEC-Foundations
   (CS Core 1/5/6). No page numbers asserted (printed pages not independently
   confirmed); rights condition carried via the shared
   `se:GAP-CS2023-RIGHTS-CLEARANCE`.
4. **NIST SP 800-218A** — registry state ABSTRACT_VERIFIED; used for no content
   claim; boundary recorded as `se:GAP-07C-NIST-218A-ABSTRACT-ONLY`.

## Corrections made during execution

- None requiring rework. The first full assemble_and_validate dry run passed
  (LANE_VALIDATION=PASS) with no graph, LinkML or count errors; no validation
  failure occurred at any point of this batch.

## Limitations

- CS2023 PDF binary was not directly downloadable from this network (per the SEK-07
  research extract method note); heading-anchored citation is used and the
  rights-clearance gap stays OPEN (HIGH).
- No Case/Counterexample records: no incident/postmortem/case-study source is
  registered (`se:GAP-07C-NO-CASE-EVIDENCE`); the anti-pattern rests on one
  explicitly marked SYNTHESIS evidence record.
- SWEBOK KA13 sections 1.3, 2.x, 3.2, 5.2 and 6 were read but not admitted
  (`se:GAP-07C-SWEBOK-KA13-UNINSPECTED-SECTIONS`).
- All records PROPOSED/PENDING_REVIEW; no ReviewDecision records; nothing ACCEPTED.
- Validation is Windows-native with the pinned venv (Python 3.14.7, LinkML
  1.12.1, linkml-runtime 1.12.1, PyYAML 6.0.3, jsonschema 4.26.0, pypdf 7.0.0);
  remote CI is not a validation basis for this lane.

## Validation commands and results (final committed tip, 2026-11-07)

Executed with `PY=C:/xDev/kAiCreator/_tmp/i321-repair/venv/Scripts/python.exe`,
`PYTHONDONTWRITEBYTECODE=1`, from worktree `C:/xDev/kAiCreator/tally/tally-i336-zcode`;
full stdout/stderr retained in `_tmp/tally-sek07-lanes/validation/07C/`:

1. Preflight: `git diff --check` exit 0 (no whitespace errors); YAML parse of
   records.yaml OK (85 records, no duplicate IDs, no non-conforming IDs, no
   ACCEPTED records).
2. `$PY docs/versions/v2.1.1/knowledge/model/validate_knowledge_graph.py --self-test`
   → `SELF_TESTS=66/66`, exit 0. Log: `selftest-tip.log`.
3. `$PY .../assemble_and_validate.py <worktree> 07C <records.yaml> <validation/07C>`
   → exit 0; every COUNT_* PASS (locators 10/3, evidence 14/3, claims 9/4,
   concepts 3/2, principles 1/1, practices 3/2, patterns 1/1, anti-patterns 1/1,
   decision models 1/1, gaps 4/1, relationships 30/10);
   GRAPH_VALIDATION=PASS; NO_ACCEPTED_RECORDS=PASS;
   LINKML_INSTANCE_VALIDATION=PASS (LinkML 1.12.1 / runtime 1.12.1);
   LANE_VALIDATION=PASS. Log: `lane-validate-tip.log`.
4. Four-path SHA256 manifest written (`manifest-four-paths.sha256`) with HEAD/TREE
   recorded; worktree clean before and after validation.

## Not done

- No merge, no freeze, no release, no deployment, no production adoption, no
  registry mutation, no schema/catalog/validator change, no graph-root
  materialization (SEK-08 territory), no claim/dispatch creation.
- PR stays OPEN/unmerged until the separately scoped SEK-08 integration boundary
  authorizes it; records await independent Exact-SHA human review.
