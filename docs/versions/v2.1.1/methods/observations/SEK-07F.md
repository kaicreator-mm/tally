# SEK-07F — Maintenance, Evolution, Refactoring & Technical Debt Lane

**Task:** #339 (SEK-07 lane 07F; coordinator #333; parent pilot #299)
**Date:** 2026-11-07
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (exact `wip/v2.1.1-knowledge` tip at lane start)
**Status:** LANE_AUTHORED / NATIVE_VALIDATION_PASSED_AT_TIP / RECORDS_AWAIT_INDEPENDENT_EXACT_SHA_REVIEW
**HUMAN_REVIEW_BURDEN=NOT_MEASURED** (no human review time was measured for this lane)

## Claim/CAS readback (before writing)

- `gh issue view 339 --repo kaicreator-mm/tally --json labels,assignees,comments` returned
  `labels=[]`, `assignees=[]`, `comments=[]` on 2026-11-07 — no Claim/CAS/dispatch
  authority is attached to the issue. State: **NOT_PRESENT**.
- Worktree search for claim/dispatch/CAS files under `docs/` found none relevant to this
  lane (only unrelated v1.4-era artifacts). State: **NOT_PRESENT**.
- No Claim or Dispatch records were created; no ownership conflict detected.

## Source research and locator verification

All locators were directly inspected on 2026-11-07 before any record was written.

- **SWEBOK v4.0a** (registry `SE-SRC-SWEBOK-GUIDE-4-0A`): the pinned official PDF
  (`swebok-v4-bust.pdf`, SHA-256 `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
  6,015,672 bytes, 411 pages) was hash-re-verified locally against the registry value, then
  extracted with the shared pypdf 7.0.0 scripts and read at the page ranges recorded in the
  KA page map (KA7 chapter starts at 1-based PDF page 167 / index 166; KA8 at 185 / 184).
  Sections read in full: KA7 chapter Introduction, 1.1, 1.5, 1.6, 2.1.1-2.1.4, 2.2.1-2.2.5,
  2.3.1-2.3.2, 2.4, 3.1-3.2.4, 4.1-4.5, 5, topic matrix and references; KA8 sections
  3, 3.1, 3.1.1-3.1.3, 3.2, 3.3. Records were admitted **only** from the sections the lane
  scope permits (KA7 Intro/1.1, 1.5, 1.6, 2.1.3, 2.1.4, 2.2.1, 2.3.1, 4.2, 4.3; KA8 3-3.2);
  KA7 4.4 (CI/CD) was read but deliberately excluded per the 07E lane boundary. Printed
  folios (7-1..7-14, 8-9..8-10) were confirmed on the extracted pages themselves.
- **CS2023** (registry `SE-SRC-CS2023`): the SEK-07 shared research extract
  (full text via extraction service, 2026-11-07; binary download unavailable from this
  network, so no binary SHA-256 is recorded) was read for the SE knowledge unit
  "SE-Refactoring: Refactoring and Code Evolution" (KA Core topics 1-4, Non-core topic 5,
  illustrative learning outcomes) and the SE Core Hours table row. The KU locator is
  heading-anchored per the extract's rule (no invented page numbers); the shared Core-Hours
  locator (`se:LOC-CS2023-SE-KU-TABLE`) is registry-verified and was re-read in the extract.
  Shared records `se:VIEW-CS2023-SE`, `se:LOC-CS2023-SE-KU-TABLE`,
  `se:GAP-CS2023-RIGHTS-CLEARANCE` and `se:DOMAIN-SOFTWARE-ENGINEERING` were carried into
  the lane; `se:LOC-CS2023-SE-OVERVIEW` and `se:LOC-CS2023-BOK-TABLE` were not used and
  were not carried.
- **ISO/IEC/IEEE 14764 and 12207**: metadata-only; every ISO attribution in lane records
  is explicitly "as reported by SWEBOK", never ingested ISO text.

## Corrections made during the batch

1. `se:REL-07F-011` initially referenced a non-existent subject id
   (`se:PRI-07F-EVOLUTION-COMPLEXITY` instead of `se:PRI-07F-MANAGE-EVOLUTION-COMPLEXITY`);
   caught by the lane graph validator dry run (DANGLING_REFERENCE) and fixed before commit.
2. The shared-snippets file `_tmp/tally-sek07-lanes/shared-snippets-sek07.yaml` carries the
   shared Domain record's second `boundary_notes` entry line-wrap-truncated mid-sentence
   (it ends "(SEK-02R"; verified by strict YAML parse of that file). The lane restores the
   complete reviewed sentence from the SEK-06A final tip
   (`origin/task/tally-v211-sek06a-requirements`), which the snippets were programmatically
   derived from. All other shared field values are copied exactly; the truncation is
   recorded here so SEK-08 deduplication does not propagate the corrupted variant.

## Limitations and honest boundaries

- The Lehman evolution laws are recorded as SWEBOK's secondhand report of research; no
  primary Lehman source is registered (see `se:GAP-07F-NO-CASE-EVIDENCE`).
- No case-class source exists for this scope, so the lane contains no Case or
  Counterexample records; pattern/anti-pattern consequences rest on BOM/curriculum text
  plus one clearly marked synthesis evidence record (`se:EVD-07F-SYNTHESIS-DEBT-PRIORITIZATION`).
- KA7/KA8 coverage is limited to the sections listed in `se:GAP-07F-KA7-CONTENT-BREADTH`;
  no content beyond inspected sections was inferred.
- CS2023 rights clearance remains HIGH/OPEN (`se:GAP-CS2023-RIGHTS-CLEARANCE`); no
  general reuse permission is inferred.

## Validation

All commands run Windows-native in the worktree with the pinned venv interpreter
`C:/xDev/kAiCreator/_tmp/i321-repair/venv/Scripts/python.exe` (Python 3.14.7,
LinkML 1.12.1, linkml-runtime 1.12.1, PyYAML 6.0.3, jsonschema 4.26.0, pypdf 7.0.0),
`PYTHONDONTWRITEBYTECODE=1`.

- Preflight (pre-commit): `git diff --check` clean; strict YAML parse of `records.yaml`
  PASS (90 records, unique ids, all ids match `^se:[A-Z0-9][A-Z0-9._-]*$`).
- Working-tree dry run of the lane assembler (pre-commit, after the REL-07F-011 fix):
  all `COUNT_*` PASS, `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS`, exit 0.
- Final tip receipts (self-test 66/66, assembler, four-path SHA-256 manifest,
  identity/receipt files) are produced at the committed tip and preserved in
  `_tmp/tally-sek07-lanes/validation/07F/` (outside the write set), per the brief.
- No validation failure remained unfixed; no PASS was reported from an earlier tree.
- Independent human review of any record: PENDING. Nothing in this lane is ACCEPTED.
