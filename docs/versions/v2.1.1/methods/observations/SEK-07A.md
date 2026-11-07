# SEK-07A — Method Observation (Foundations, Processes, Lifecycle & Quality Models lane)

**Task:** #334 (lane SEK-07A of the SEK-07 lanes; coordinator #333; parent pilot #299)
**Date:** 2026-11-07
**Worktree / branch:** `tally-i334-zcode` / `task/tally-v211-sek07a-foundations-lifecycle`
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (verified: `git rev-parse HEAD` matched; clean worktree at start)
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING
**HUMAN_REVIEW_BURDEN=NOT_MEASURED** (no human review was timed or performed in this lane; all records remain PROPOSED / PENDING_REVIEW)

## Claim / CAS readback (Task Pack step 2)

- `gh issue view 334 --repo kaicreator-mm/tally --json labels,assignees,state,title,comments`:
  labels `[]`, assignees `[]`, state OPEN, comments `0`.
- Worktree claim-file sweep: the only "claim"-named files in the tree are
  Tally v2 runtime code (`experiments/v2/...`, `claim-replay-*.mjs`,
  workflow definitions) — unrelated to knowledge-lane CAS dispatch; no
  knowledge Claim/Dispatch/CAS authority file exists for this lane.
- **Result: Claim/CAS/dispatch authority = NOT_PRESENT.** No generation or
  status values were fabricated; no duplicate Claim/Dispatch records were
  created; no ownership conflict detected.

## Research path — SWEBOK v4.0a reinspection

1. PDF identity: `swebok-v4-bust.pdf`, native SHA-256
   `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`,
   6,015,672 bytes, 411 pages — hash matches the SEK-02R registry record and
   the shared `sources/SHA256SUMS.txt`. This is the same verified binary the
   registry and the SEK-06 lanes cite.
2. Chapter anchors came from the coordinator's `KA_PAGE_MAP.md` (mechanical
   folio scan: KA10 starts at 1-based PDF page 221 / index 220; KA11 at 233 /
   232; KA12 at 246 / 245; KA18 at 367 / 366). The map is a discovery aid
   only; every locator below was verified by extracting and reading the
   actual pages with `extract_pages.py` (pypdf 7.0.0, the pinned venv
   Python) on 2026-11-07:
   - KA10 pages 221-232 (printed 10-1 to 10-12) — full chapter read,
     including the topic matrix and reference list (used to confirm section
     numbering and that the chapter ends at 10-12).
   - KA11 pages 233-241 (printed 11-1 to 11-9) — introduction, sections 1-4
     (through the Agile methods opening); sections 1 and 3 admitted as
     locators.
   - KA12 pages 246-251 (printed 12-1 to 12-6) — introduction and sections
     1-2 (through the SQM/QMS opening); introduction + sections 1.1-1.4
     admitted as locators.
   - KA18 pages 367-372 (printed 18-1 to 18-6) — introduction and sections
     1-5 (through statistical analysis); only sections 1-3 admitted as
     locators (see the foundations-breadth gap).
3. Extraction artifacts (page text) live only under
   `_tmp/tally-sek07-lanes/ka*.txt`; nothing PDF-derived was committed to the
   repository. All corpus text is concise original paraphrase.
4. Printed-folio convention: the Guide prints chapter-relative folios
   (`10-1`, `11-2`, ...). The lane's locators record both 0-based PDF page
   indices (the convention of existing SEK-06 locators) and printed folios.

## Scope decisions

- **Three taxonomy views, not one tree.** The Task Pack requires capturing
  how lifecycle/process/quality views relate without collapsing them; the
  lane models PROCESS (`se:VIEW-07A-SWEBOK-PROCESS-LIFECYCLE`), QUALITY
  (`se:VIEW-07A-SWEBOK-QUALITY-MODELS`) and DISCIPLINE
  (`se:VIEW-07A-SWEBOK-ENGINEERING-FOUNDATIONS`) views joined by one typed
  `CROSS_CUTS` edge and scope-bearing `APPLIES_TO` edges — never containment.
- **ISO stays metadata-only.** 12207/15288/25010 are ACCESS_RESTRICTED in the
  registry; no normative text was opened, summarized or cited. SWEBOK's
  attribution of the process categories to ISO/IEC/IEEE 12207:2017 is
  recorded as the Guide's citation with an explicit limitation
  (`se:GAP-07A-ISO-NORMATIVE-LIFECYCLE`).
- **CS2023 skipped entirely.** The registry-verified CS2023 SE unit list
  (nine units) has no process/lifecycle/quality-models unit; per the brief,
  the lane records the curriculum-side absence as a CoverageGap
  (`se:GAP-07A-CS2023-CURRICULUM-SCOPE`) and carries **no** CS2023 shared
  view/locators (only the mandatory shared domain and rights gap, both
  verbatim).
- **No invented cases.** No case-class source is registered for this scope;
  the lane therefore contains no Case/Counterexample records
  (`se:GAP-07A-NO-CASE-EVIDENCE`), and the anti-pattern's counterproductive
  characterization runs through a lane-marked SYNTHESIS evidence record
  disclosed at the evidence, at the anti-pattern's scope field, and on the
  `DERIVED_FROM` edge.
- **Boundary respect.** KA12 is admitted at model level only (reliability/
  performance engineering stays 07D); planning/estimation/risk governance
  (07G), implementation practices (07B) and CI/CD mechanics (07E) are not
  absorbed; KA10 DevOps text is not turned into delivery-mechanics records.

## Corrections and authoring passes

1. **Schema-conformance pass.** Records use only slots the LinkML model
   defines for each class (no `label` on DecisionModel; no `limitations` on
   AntiPattern — the synthesis disclaimer lives in its scope field, following
   the SEK-06 convention).
2. **Claim-provenance pass.** Claim records carry no `evidence_refs`; every
   claim's evidence link is a canonical `SUPPORTED_BY` edge (verified in
   preflight: 11 claims, none missing the edge). Evidence-bearing edges cite
   existing evidence records only.
3. **Counterfeit-page pass.** Every locator's PDF-index/folio span was
   re-derived from the extracted page headers before writing (e.g.,
   `LOC-07A-SWEBOK-KA10-MANAGEMENT-ADAPTATION` spans 2.6-2.9 = indices
   226-228 = printed 10-7 to 10-9, confirmed by the `10-7`/`10-8`/`10-9`
   running headers). No page anchor appears that the extraction does not
   support.
4. **Shared-record fidelity check.** The two shared records (domain,
  CS2023 rights gap) were verified parsed-identical to the SEK-07 snippet
  pack in preflight (`SHARED_RECORDS_EQUALITY=PASS`). Noted without editing:
  the pack's domain boundary note ends mid-sentence ("...(SEK-02R") — an
  artifact of the pack's generation; rewording shared records is prohibited,
  so the lane carries it verbatim and flags it in the lane README for the
  SEK-08 integrator.
5. **Extraction-tool caveat.** pypdf 7.0.0 emitted fontTools warnings for
  some embedded fonts (CFF Type1 encoding); text extraction was unaffected
  for the pages cited (verified by reading the extracted text against the
  section structure). No content decision rests on a garbled fragment.

## Validation — commands and results

Environment: Windows-native Git Bash; pinned interpreter
`C:/xDev/kAiCreator/_tmp/i321-repair/venv/Scripts/python.exe` (Python 3.14.7,
PyYAML 6.0.3, LinkML 1.12.1 / linkml-runtime 1.12.1, jsonschema 4.26.0,
pypdf 7.0.0); `PYTHONDONTWRITEBYTECODE=1` exported for every run.

Pre-commit working-tree runs (all green before the commit):

1. `git diff --check` → exit 0 (no whitespace errors).
2. YAML parse + integrity script over `records.yaml`: parse OK;
   `SHARED_RECORDS_EQUALITY=PASS`; 88 records with unique canonical IDs
   (`UNIQUE_IDS=PASS`); every claim has a SUPPORTED_BY edge.
3. `validate_knowledge_graph.py --self-test` → `SELF_TESTS=66/66`, exit 0.
4. `assemble_and_validate.py <worktree> 07A <records.yaml> <validation/smoke>`
   → every `COUNT_*` PASS (locators 11/3, evidence 12/3, claims 11/4,
   concepts 6/2, principles 2/1, practices 3/2, patterns 1/1, anti-patterns
   1/1, decision models 1/1, gaps 5/1, relationships 25/10),
   `DERIVED_SOURCES=1 se:SRC-SWEBOK-GUIDE-4-0A`, `GRAPH_VALIDATION=PASS`,
   `NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
   `LANE_VALIDATION=PASS`, exit 0.

Validation failures encountered: **none** — no step required a repair
re-run at any point (both the working-tree run and the exact committed-tip
run below were green on their first execution).

Exact committed-tip receipt (single commit, receipt dir
`_tmp/tally-sek07-lanes/validation/07A/`):

- `selftest-tip.log` — self-test at the committed tip: `SELF_TESTS=66/66`,
  exit 0.
- `lane-validate-tip.log` — lane assembly + graph validation + count
  minimums + no-ACCEPTED check + LinkML instance validation at the
  committed tip: all keys PASS, `LANE_VALIDATION=PASS`, exit 0.
- `manifest-four-paths.sha256` — SHA-256 over the four changed repo paths,
  with HEAD/TREE recorded; worktree clean before and after.
- `identity.txt` / `receipt.ini` — HEAD, TREE, BASE
  (`7a70211b4460e4c753c97798670de9f34861d4a4`; merge-base equals base),
  `SOURCE_REVIEW=REINSPECTED_2026-11-07`, `SELF_TEST=66/66`,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS`, record counts,
  clean worktree before/after, `WINDOWS_NATIVE_VALIDATION=DONE`,
  `PRODUCTION_ADOPTION=NO`, `FREEZE=NO`, `MERGE=NO`, `RELEASE=NO`,
  `DEPLOYMENT=NO`.

## Limitations

- Every content record derives from ONE source (SWEBOK v4.0a); nothing in
  the lane has passed human review; the Guide's own authority limits
  (generally accepted knowledge, not prescription; non-comprehensive
  citations) carry into the records' scope and limitation fields.
- No empirical, case or counterexample evidence exists in the lane; pattern
  and anti-pattern consequences are untested at the corpus level.
- The SWEBOK PDF/landing-page release-date discrepancy recorded by SEK-02R
  remains OPEN in the registry and is inherited untouched by this lane.
- KA18 sections 4-5 were read but not admitted as locators; KA16/KA17 were
  not opened (foundations-breadth gap, LOW/OPEN).
- This observation was authored in the same batch as the lane records; its
  validation section quotes receipt key lines, with the authoritative
  artifacts in the receipt directory rather than embedded here.
