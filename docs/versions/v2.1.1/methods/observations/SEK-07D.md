# SEK-07D — Method Observation

**Task:** #337
**Date:** 2026-11-07
**Baseline (exact WIP base):** `7a70211b4460e4c753c97798670de9f34861d4a4`
**Branch:** `task/tally-v211-sek07d-reliability-performance` (created independently from the exact base; not stacked)
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING
**HUMAN_REVIEW_BURDEN:** NOT_MEASURED (no human reviewer has measured review effort for this lane; no value is invented)

## Inputs and dependency evidence

- Parent pilot: #299. Parallel coordinator: #333. Predecessors: SEK-05 #308
  and SEK-06 #321, both CLOSED; this lane depends on neither's paths.
- Exact WIP base verified in the worktree before any write:
  `git rev-parse HEAD` = `7a70211b4460e4c753c97798670de9f34861d4a4`,
  `git status --short` empty, branch
  `task/tally-v211-sek07d-reliability-performance`.
- Structural template: the reviewed SEK-06A lane records
  (`origin/task/tally-v211-sek06a-requirements`, PR #327) were fetched with
  the exact `git show` command from the operating brief and mirrored in lane
  structure, lifecycle conventions and evidence-record shape.

## Claim / CAS / dispatch readback (actual state, no fabrication)

- `gh issue view 337 --repo kaicreator-mm/tally --json labels,assignees,comments`
  on 2026-11-07 returned: state OPEN, `labels: []`, `assignees: []`,
  `comments: []`. No Claim, CAS or dispatch authority is present on the
  issue: **NOT_PRESENT**.
- Worktree check: no knowledge-lane claim/dispatch files exist in the lane's
  write set or the surrounding knowledge tree on this base (the repo's
  claim/CAS artifacts under `experiments/v2/` are Tally v2 runtime
  experiments, unrelated to knowledge-lane ownership). No Claim or Dispatch
  records were created by this lane; no ACTIVE/generation values are asserted
  anywhere in the four-path write set.
- No ownership conflict was detected; single-writer fencing per the Task Pack
  was preserved (only this lane's four paths changed).

## Source research and locator verification

1. **SWEBOK Guide v4.0a (professional BOK).** Official PDF binary
   `swebok-v4-bust.pdf` (registry-verified SHA-256
   `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
   411 pages; hash re-verified 2026-11-07 per the SEK-07 shared page map).
   Chapter starts located mechanically per the shared KA page map: KA6
   Software Engineering Operations begins at 1-based PDF page 151 (0-based
   150, printed folio 6-1); KA12 Software Quality at 1-based PDF page 246
   (0-based 245, printed 12-1). Text was extracted with pypdf 7.0.0
   (`extract_pages.py`, venv Python 3.14.7) for the full chapter ranges
   (KA6 pages 151-166; KA12 pages 246-263) into working files under
   `_tmp/tally-sek07-lanes/validation/07D/` (not committed), then read. The
   seven admitted locators were verified against the extracted page text and
   printed folios: KA6 sections 1.6 (PDF index 155, 6-6), 2.3 (157, 6-8),
   2.4 (157, 6-8), 2.5 (157-158, 6-8 to 6-9), 4.3 (160, 6-11); KA12 sections
   1.4 with 1.4.1/1.4.2 (249-250, 12-5 to 12-6) and 2.3.1 (252-253, 12-8 to
   12-9). All SWEBOK evidence statements are original paraphrases of read
   text; no substantial source text is reproduced in committed records.
2. **CS2023 SE-Reliability (curriculum).** The prepared verbatim extraction
   (`_tmp/tally-sek07-lanes/sources/cs2023-bok-sek07-extract.md`, extraction
   date 2026-11-07, official Body of Knowledge PDF full text via a
   text-extraction service — direct binary download from csed.acm.org fails
   from this network and the Internet Archive snapshot is rate-limited 429,
   so no CS2023 PDF SHA-256 is recorded) is the anchor. The lane's KU locator
   is heading-anchored (unit heading + topic numbering) inside the
   registry-verified SE section page span and asserts no page number of its
   own. The Core Hours row (Software Reliability: 2 KA Core hours, 0 CS
   Core) is cited through the shared registry-verified KU-table locator.
3. **Boundary discipline.** KA12 was cited only for its engineering sections
   (1.4 failure-risk-reduction techniques and assurance cases; 2.3.1 quality
   measurement); its model-level quality definitions were left to SEK-07A.
   Security-operations and CI/CD mechanics were left to 07C/07E. The
   reliability/performance/resilience/availability vocabulary is kept
   distinct across records rather than merged.

## Corrections made during execution

- None to predecessor artifacts (out of write set). One self-correction
  during authoring: an earlier draft placed the printed folio of KA6 1.6 at
  6-5; re-reading the extracted page confirmed the section heading and body
  sit on PDF index 155 (printed 6-6), and the locator records 6-6.

## Validation (Windows-native, exact tip)

Executed at the committed tip on Windows (Git Bash) with the pinned venv
Python `C:/xDev/kAiCreator/_tmp/i321-repair/venv/Scripts/python.exe`
(Python 3.14.7, LinkML 1.12.1, linkml-runtime 1.12.1, PyYAML 6.0.3,
jsonschema 4.26.0, pypdf 7.0.0), `PYTHONDONTWRITEBYTECODE=1` exported; full
logs in `_tmp/tally-sek07-lanes/validation/07D/`:

1. Preflight (pre-commit): `git diff --check` clean; `yaml.safe_load` parse
   of records.yaml plus structural checks (ID pattern, no ACCEPTED, no
   dangling references, evidence chain, locator uniqueness, edge signatures)
   all clean.
2. Self-test at tip:
   `validate_knowledge_graph.py --self-test` → `SELF_TESTS=66/66`, exit 0
   (`selftest-tip.log`).
3. Lane validation at tip: `assemble_and_validate.py <worktree> 07D
   <records.yaml> <outdir>` → every `COUNT_*` PASS (locators 9/3, evidence
   16/3, claims 13/4, concepts 4/2, principles 2/1, practices 4/2, patterns
   2/1, anti-patterns 1/1, decision models 2/1, gaps 4/1, relationship
   assertions 30/10), `DERIVED_SOURCES=2`, `GRAPH_VALIDATION=PASS`,
   `NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
   `LANE_VALIDATION=PASS`, exit 0 (`lane-validate-tip.log`).
4. Four-path SHA-256 manifest + identity + receipt files written to the
   validation directory with HEAD/TREE recorded; worktree clean before and
   after.

No validation failures occurred at the final tip; no FAIL receipts exist for
this lane. (Had any step failed, the fix-commit-rerun loop would have
produced a new tip and the receipts above name only the final one.)

## Limitations

- CS2023 material rests on a text-extraction service (no byte-identical PDF
  on hand); KU-level locators are heading-anchored. SWEBOK locators rest on
  the hash-verified official PDF and do not share this limitation.
- No case-class source is registered for this scope; pattern/anti-pattern
  records carry no real-world illustration (`se:GAP-07D-NO-REAL-CASE-EVIDENCE`).
- CS2023 SF-Reliability and PDC-Evaluation topics could not be re-verified
  this session and are recorded as a locator gap, not cited
  (`se:GAP-07D-CS2023-ADJACENT-TOPICS`).
- SWEBOK coverage is bounded by the seven inspected sections
  (`se:GAP-07D-SWEBOK-UNINSPECTED-SECTIONS`); nothing beyond them is
  inferred. KA6 2.4 relays ISO/IEC/IEEE 20000-1, whose own text remains
  registry metadata-only.
- All records remain PROPOSED/PENDING_REVIEW pending independent Exact-SHA
  human review; nothing here is accepted knowledge, and no production
  adoption, model freeze, merge, release or deployment is implied or
  authorized by this observation.
