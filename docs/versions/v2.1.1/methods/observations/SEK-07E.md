# SEK-07E — Method Observation (Integration, Build, CI/CD & Delivery lane)

**Task:** #338 (lane SEK-07E of the SEK-07 knowledge-taxonomy expansion; coordinator #333, parent pilot #299)
**Date:** 2026-11-07
**Worktree / branch:** `tally-i338-zcode` / `task/tally-v211-sek07e-build-delivery`
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (= tip of `wip/v2.1.1-knowledge`; merge-base equals base; branch created independently, not stacked)
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING

## Claim / CAS / dispatch readback (brief §9)

Checked before writing any record:

- `gh issue view 338 --repo kaicreator-mm/tally` — state OPEN, title
  "[v2.1.1][07E] Integration, Build, CI/CD & Delivery", **labels: none,
  assignees: none, comments: 0**. No Claim or CAS identifiers, generation
  counters or dispatch records are present on the issue: **Claim/CAS state =
  NOT_PRESENT**.
- Worktree checked for claim/dispatch files at the exact base: none exist
  (the only `claim*` files in the repository root are the pre-existing
  `claim-replay-fixture.mjs` / `claim-replay-runner.mjs` test fixtures from
  the base commit, unrelated to SEK-07E; no SEK-07E claim file was found).
  **Worktree claim files = NOT_PRESENT**.
- No ownership conflict detected; single-writer fencing held (no other agent
  touched the four paths — the lane directory did not exist in the worktree
  before this task).

## Research path — SWEBOK KA6/KA8 and CS2023 SE-Tools verification

1. **SWEBOK v4.0a (source `SE-SRC-SWEBOK-GUIDE-4-0A`)** — reopened directly
   from the official PDF binary at
   `_tmp/tally-sek07-lanes/sources/swebok-v4-bust.pdf`, SHA-256
   `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`
   (411 pages), re-verified against the registry value via the shared
   `KA_PAGE_MAP.md` / `SHA256SUMS.txt` pack before extraction. Chapter starts
   were taken from the mechanical page map (KA6 at 1-based page 151, KA8 at
   185) and then every cited section's actual pages were extracted and read
   with `extract_pages.py` (pypdf 7.0.0, venv Python 3.14.7): KA6 §2.1.1
   (index 156), §2.2 (156-157), §3.2 (159), §3.3 (159-160); KA8 §1.3-1.4
   (186-188), §2.5-2.6 (191-192), §6/§6.1 (196), §6.2 (196-197). Full
   extracts retained as research artifacts at
   `_tmp/tally-sek07-lanes/validation/07E/swebok-ka6-extract.txt` and
   `swebok-ka8-extract.txt` (not committed). Locators record both PDF page
   indices and printed chapter folios; paraphrases are original.
2. **CS2023 (source `SE-SRC-CS2023`)** — the official Body of Knowledge PDF
   could not be downloaded as a binary on this network (the SEK-06 lanes hit
   the same TLS reset; the stub files in the sources dir are 162-byte failed
   downloads), so no binary SHA-256 is recorded for CS2023. The verified
   content base is the full-text extraction captured 2026-11-07 in
   `_tmp/tally-sek07-lanes/sources/cs2023-bok-sek07-extract.md` (SE-Tools unit
   topics and learning outcomes, verbatim) plus the SEK-06 verified anchors
   pack for the Core Hours table row. The lane's SE-Tools KU locator is
   **heading-anchored** (exact unit heading + topic numbering) with the
   registry-verified page anchor (PDF page index 178 / one-based 179) and
   asserts no printed page of its own. No page number was invented.
3. **Rights handling** stayed inside registry boundaries for both sources:
   bibliographic metadata, exact locators, concise original paraphrases; short
   verbatim topic titles only where locator anchoring requires them; no PDF or
   substantial text mirrored into the repository; no redistribution inference.
   The shared `se:GAP-CS2023-RIGHTS-CLEARANCE` (HIGH/OPEN) is carried
   verbatim.
4. **ISO sources untouched**: ISO/IEC/IEEE entries remain metadata-only and
   back no content claim. NIST SP 800-218/218A are registered but were not
   extracted for this lane; no record claims anything about them (README
   Known-gaps item 4).

## Scope decisions

- Build/CI/delivery mechanics only. Configuration-management theory beyond
  build/integration boundaries, maintenance/evolution, implementation
  practices, reliability/SRE operations and planning/estimation were left to
  their sibling lanes even where adjacent SWEBOK sections were one page away
  (e.g. KA6 §4 change management and KA8 §3 change control were read during
  extraction but deliberately not admitted).
- The curriculum and professional-BOK lenses are kept as separate taxonomy
  views (`se:VIEW-CS2023-SE` shared; `se:VIEW-07E-BUILD-DELIVERY-PROCESS`
  lane-local), related only through explicit `CROSS_CUTS` edges — a direct
  response to the #299 §2.2 instruction to challenge any single-hierarchy
  taxonomy hypothesis rather than force lifecycle, disciplines and quality
  attributes into one tree.
- Terminological variance preserved: SWEBOK KA6 and KA8 use "release" with
  different emphases; both senses are recorded side by side
  (`se:CON-07E-RELEASE`, asserted by `se:CLM-07E-SWEBOK-RELEASE-TERM-VARIANCE`)
  instead of averaged into one definition.
- One lane-marked SYNTHESIS evidence (`se:EVD-07E-SYNTHESIS-CI-ALIGNMENT`)
  quarantines the only cross-source inference (CS2023 CI-ground-truth outcome
  aligns with SWEBOK KA8's pipeline description); it is clearly labeled as
  not a sentence in either source and stays PENDING_REVIEW.
- No Case/Counterexample records: neither permitted source carries real
  build/delivery case material and no case-class source is registered. The
  missing evidence classes are recorded as `se:GAP-07E-REAL-CASE-EVIDENCE`
  and `se:GAP-07E-NO-EMPIRICS` rather than papered over with invented cases.

## Authoring corrections during drafting

1. **Count-fidelity pass.** The README's record-count table was corrected once
   before commit (knowledge_claims kind histogram said "4 DEFINITION"; the
   actual records carry 3 DEFINITION + 1 SYNTHESIS). Counts in README,
   EXPLANATIONS and the preflight log were then cross-checked against the
   parsed YAML (all 14 collections match).
2. **Schema-conformance pass.** No fields outside the LinkML slots were used
   (DecisionModel has no `label` slot — the SEK-06A lesson was applied at
   authoring time, not discovered at validation time). Predicates with
   `evidence_required` and/or `qualifiers_required` were checked against
   RELATIONSHIP_CATALOG.yaml in a pre-commit preflight script
   (`preflight.py`, log `preflight.log` in the validation dir) before the
   first harness run; the first real harness run then passed clean.
3. **Shared-record identity pass.** The four carried shared records (domain,
   view, Core-Hours locator, rights gap) were verified parsed-identical to
   `_tmp/tally-sek07-lanes/shared-snippets-sek07.yaml` by the same preflight
   script. Note: the shared pack's domain record contains a boundary note
   truncated mid-sentence at "(SEK-02R"; per the verbatim-copy rule it was
   copied as-is (not repaired locally) and flagged in the README for the pack
   owner. Only the shared locators actually used were carried
   (`se:LOC-CS2023-SE-KU-TABLE`); the two unused ones were omitted per the
   brief's "only what you use" rule.
4. **Preflight script bug (process correction, not data).** The first
   preflight run reported 95 false "lane token missing" failures caused by a
   string-splitting bug in the check itself (`rest[0]` already contains the
   `se:` prefix); the data was correct, the check was fixed, and the rerun
   passed 0 fails. Recorded here so the noisy first log is not mistaken for a
   data defect.

## Limitations

- SWEBOK coverage is breadth-limited to the inspected sections
  (`se:GAP-07E-SWEBOK-BREADTH`, MEDIUM/OPEN); uninspected KA6/KA8 sections may
  not be inferred.
- No empirical evidence backs practice effectiveness in this lane
  (`se:GAP-07E-NO-EMPIRICS`, LOW/OPEN); the manual-handoff anti-pattern's
  inefficiency characterization is SWEBOK's qualitative statement.
- The CS2023 SE-Tools KU locator is heading-anchored; its printed page was not
  independently confirmed (text-extraction constraint, same as SEK-06).
- All records are PROPOSED/PENDING_REVIEW; nothing is ACCEPTED, and no
  ReviewDecision records exist in this lane.

## Validation — exact committed-tip receipt

Environment: Windows-native, venv Python 3.14.7, PyYAML 6.0.3, LinkML 1.12.1 /
linkml-runtime 1.12.1, jsonschema 4.26.0, pypdf 7.0.0,
`PYTHONDONTWRITEBYTECODE=1`. Validation ran at the final committed tip of
`task/tally-v211-sek07e-build-delivery` (single content commit; the exact
HEAD/TREE, base equality and clean-worktree states are bound in
`_tmp/tally-sek07-lanes/validation/07E/identity.txt` and `receipt.ini`; full
stdout/stderr in `selftest-tip.log`, `lane-validate-tip.log`; four-path
SHA256 manifest in `manifest-four-paths.sha256`).

Commands (run from the worktree root, exit codes in the logs and receipt):

```bash
PY=C:/xDev/kAiCreator/_tmp/i321-repair/venv/Scripts/python.exe

# 1. Validator self-test
$PY docs/versions/v2.1.1/knowledge/model/validate_knowledge_graph.py --self-test
#    -> SELF_TESTS=66/66, exit 0 (selftest-tip.log)

# 2. Lane assembly + counts + graph + no-ACCEPTED + LinkML instance validation
$PY C:/xDev/kAiCreator/_tmp/tally-sek07-lanes/scripts/assemble_and_validate.py \
  C:/xDev/kAiCreator/tally/tally-i338-zcode 07E \
  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/build-delivery/records.yaml \
  C:/xDev/kAiCreator/_tmp/tally-sek07-lanes/validation/07E
#    -> every COUNT_* PASS (locators 10, evidence 12, claims 16, concepts 5,
#       principles 1, practices 3, patterns 1, anti_patterns 1,
#       decision_models 1, coverage_gaps 4, relationship_assertions 39),
#       GRAPH_VALIDATION=PASS, NO_ACCEPTED_RECORDS=PASS,
#       LINKML_INSTANCE_VALIDATION=PASS, LANE_VALIDATION=PASS, exit 0
#       (lane-validate-tip.log)
```

A pre-commit smoke run of both commands on the identical working-tree content
produced the identical PASS lines (retained under
`_tmp/tally-sek07-lanes/validation/07E-smoke/`); the binding receipt is the
committed-tip run above.

## Human review burden

HUMAN_REVIEW_BURDEN=NOT_MEASURED. No human review time was measured for this
lane; the 95 records await independent Exact-SHA review, and no acceptance is
implied by the machine validation above.

## Not done (out of scope by design)

- No merge, freeze, tag, release, deployment or production adoption; the PR
  stays OPEN until the separately scoped SEK-08 integration boundary
  authorizes integration.
- No central taxonomy, Source Registry, schema/catalog/validator, shared graph
  root, other lanes' files, Tally v2 code, DomainForge or Domain Package
  changes; the diff touches exactly the four permitted paths.
- No Case/Counterexample records, no ReviewDecision records, no ACCEPTED
  records, no registry mutation (the NIST build-security extraction gap is
  proposed in the README, deferred to a future task).
