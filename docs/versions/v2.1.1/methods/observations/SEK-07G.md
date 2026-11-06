# SEK-07G — Method Observation (Planning, Risk, Quality Management & Technical Decisions lane)

**Task:** #340 (lane SEK-07G of the SEK-07 knowledge-taxonomy lanes; coordinator #333, parent pilot #299)
**Date:** 2026-11-07
**Worktree / branch:** `tally-i340-zcode` / `task/tally-v211-sek07g-planning-risk-quality`
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (= exact WIP base; branch created from it, not stacked)
**Status:** EXECUTED / EXPERIMENTAL / HUMAN_REVIEW_PENDING
**HUMAN_REVIEW_BURDEN=NOT_MEASURED** (no human reviewed the records in this batch; no review time, defect count or burden figure was measured, so none is reported)

## Claim / CAS readback (per Task Pack step 2)

- `gh issue view 340 --repo kaicreator-mm/tally --json labels,assignees,state,comments`
  returned: state `OPEN`, `labels: []`, `assignees: []`, no comments.
- Worktree check for claim/dispatch records found only Tally v2 runtime
  code with "claim"/"cas" in paths (claim-replay fixtures, host-sqlite-cas
  experiments) — no SEK knowledge Claim/Dispatch/CAS authority records.
- **Authoritative readback: Claim/CAS/dispatch state for this task is
  NOT_PRESENT.** No generation or status values were fabricated; no
  Claim or Dispatch records were created (none were authorized); no
  ownership conflict was detected, so single-writer fencing was
  undisturbed.

## Research path — SWEBOK KA9 / KA15 direct reinspection

1. **Binary identity.** The official SWEBOK v4.0a PDF
   (`swebok-v4-bust.pdf`, 411 pages) was used from the SEK-07 shared
   sources directory; its SHA-256
   `0c9db4da0b2425c58cf778b99131eb31bb6c188e98b67163a549a1366d96e511`
   is recorded as re-verified on 2026-11-07 in the shared page map and
   matches the SEK-02R registry dossier. All extraction used the pinned
   venv Python 3.14.7 with pypdf 7.0.0 (`extract_pages.py`);
   full page dumps are retained as research logs in
   `_tmp/tally-sek07-lanes/validation/07G/swebok-ka9-p*.txt` and
   `swebok-ka15-p*.txt` (working artifacts, not committed).
2. **KA9 Software Engineering Management** (chapter start 1-based PDF
   page 202 / 0-based 201, per the shared page map — confirmed by the
   `9-1` folio on that page). Pages 202-220 were extracted and read.
   Lane locators were admitted for: the Introduction (definition,
   measurement-informed management, working definitions, seven-topic
   breakdown; printed 9-1 to 9-6); section 2.3 Effort, Schedule and Cost
   Estimation (printed 9-8 to 9-9); sections 2.5-2.7 Risk / Quality /
   Plan Management (printed 9-9 to 9-10); section 6 Software Engineering
   Measurement 6.1-6.4 (printed 9-14 to 9-16). Every locator string
   carries both 0-based PDF page indices and printed folios, matching the
   SEK-06A convention.
3. **KA15 Software Engineering Economics** (chapter start 1-based 288 /
   0-based 287, confirmed by the `15-1` folio). Pages 288-308 were
   extracted and read. Lane locators: Introduction + breakdown figure
   (printed 15-1 to 15-2); section 2 The Engineering Decision-Making
   Process 2.1-2.7 (printed 15-5 to 15-7); section 8 Estimation 8.1-8.5
   (printed 15-13 to 15-16). Sections 3-7 and 9-10 were read for context
   (multiple-attribute techniques in section 6 inform the pattern's
   trade-offs via the section 2 anchor) but no record cites a section
   outside its own locator's span.
4. **KA14 Professional Practice: deliberately not inspected.** The brief
   lists KA14 as optional and only for decision/professional-judgment
   claims; the needed professional-ethics clause (Code of Ethics 3.09) is
   quoted inside KA15 §8, so the lane cites it only as SWEBOK's
   quotation. No KA14 locator or claim exists.
5. **CS2023.** The Teamwork unit (CS Core topic 5 team processes with
   effort estimation and work schedule; KA Core topic 8 distributed-team
   risks) and the SE-Requirements Non-core topic 9 (effort estimation,
   subtopics 9a-9c) were taken from the SEK-07 heading-anchored research
   extract of 2026-11-07. As in SEK-06, the CS2023 PDF was obtained as
   extracted text via a text-extraction service (direct binary download
   fails from this network), so **no binary SHA-256 of the CS2023 PDF is
   recorded**, and both lane CS2023 locators are heading-anchored with no
   page numbers of their own beyond the registry-verified SE span. The
   Teamwork Core-Hours row (2 + 3 SEP CS Core, 2 KA Core) is cited via
   the shared registry-verified table locator.

## Scope decisions

- **Boundary keeping.** Lifecycle/process model definitions (07A),
  implementation practices (07B), reliability/performance and quality
  attributes as engineering detail (07D), delivery mechanics (07E) and
  technical-debt lifecycle (07F) were not absorbed. The KA9 Dev/Sec/Ops
  passage and SDLC-model comparisons were read but not admitted; the
  lane view records SWEBOK's own "does not recommend a specific SDLC"
  caveat.
- **Quality management without KA12.** Quality management is in this
  lane's scope, but the authorized SWEBOK reading plan (KA9 + KA15,
  KA14 optional) excluded Chapter 12 Software Quality — the very KA that
  KA9 §2.6 defers to. The lane admits only the planning-side §2.6
  direction and records the limit as `se:GAP-07G-KA12-QUALITY-UNINSPECTED`
  rather than inferring KA12 content.
- **Standards kept at arm's length.** SWEBOK characterizes PMBOK, SWX,
  ISO/IEC/IEEE 15939, ISO 12207 and ISO 15288; none is registered (ISO
  entries are metadata-only). All affected records label the SWEBOK
  statement as secondary characterization; `se:GAP-07G-NO-PM-STANDARD-SOURCE`
  records the missing source class, and the README proposes concrete
  sources for a separate single-writer registry task.
- **No numeric invention.** Technique characterizations ("parametric
  typically most accurate") are reported as the Guide's overview
  statements; no estimation-accuracy or productivity numbers were created.
- **No real cases.** No case-class source exists in the inspected
  material; no Case/Counterexample records were created, and the
  anti-pattern's counterproductive characterization is a lane-marked
  SYNTHESIS evidence record combining KA15 §8 and KA9 §2.3, disclosed at
  the evidence, at the anti-pattern, and on the DERIVED_FROM edge.
- **Shared snippet artifact.** The shared-snippet pack's second domain
  boundary note is a truncated fragment of the SEK-06 wording (ends
  mid-citation). It was copied **verbatim** per the copy-EXACTLY rule;
  unilaterally rewording it would break cross-lane byte-identity needed
  for SEK-08 deduplication. Flagged here and in the lane README for
  central repair by the coordinator.
- **Taxonomy plurality.** Two views are kept separate (shared CS2023
  curriculum view; lane SWEBOK process view); the estimation-topic
  intersection is one qualified, evidenced CROSS_CUTS edge, not a merged
  hierarchy.

## Authoring corrections

- Drafted against the SEK-06A structural template; schema-conformance
  rules from SEK-06A were applied preemptively (no `label` on
  DecisionModel, no `limitations` on AntiPattern — the synthesis
  disclaimer lives in the anti-pattern's scope field). No
  post-drafting schema corrections were needed.
- Claim records carry no evidence list; their evidence links are the
  canonical SUPPORTED_BY edges (SEK-03 slice convention), and concepts
  carry evidence_refs directly, mirroring SEK-06A.

## Validation (Windows-native, pinned venv: Python 3.14.7, LinkML 1.12.1, linkml-runtime 1.12.1, PyYAML 6.0.3, jsonschema 4.26.0, pypdf 7.0.0; PYTHONDONTWRITEBYTECODE=1)

Pre-commit preflight (working tree, 2026-11-07):

1. `git diff --check` — clean (rc=0); worktree contains only the four
   lane paths as untracked files at this stage.
2. YAML parse of records.yaml with the venv python — OK; 14 collections
   parsed (`preflight-yaml-parse.log`).
3. `validate_knowledge_graph.py --self-test` — `SELF_TESTS=66/66`, exit 0
   (`preflight-selftest.log`).
4. `assemble_and_validate.py <worktree> 07G <records.yaml> <outdir>`
   pre-commit dry run — exit 0: all `COUNT_*` PASS (locators 10,
   evidence 11, claims 12, concepts 4, principles 2, practices 3,
   patterns 1, anti-patterns 1, decision models 1, gaps 4, relationship
   assertions 32), `DERIVED_SOURCES=2`, `GRAPH_VALIDATION=PASS`,
   `NO_ACCEPTED_RECORDS=PASS`, `LINKML_INSTANCE_VALIDATION=PASS`,
   `LANE_VALIDATION=PASS` (`preflight-lane-dryrun.log`).

Post-commit validation at the exact receipt tip: re-run of steps 3-4 with
logs written to `_tmp/tally-sek07-lanes/validation/07G/selftest-tip.log`
and `lane-validate-tip.log`, plus the four-path SHA256 manifest
(`manifest-four-paths.sha256`), `identity.txt` and `receipt.ini` binding
HEAD/TREE/BASE. **Validation failures: none encountered** — the
pre-commit dry run and the tip run both passed on the first attempt; no
repair commits were required after the initial commit.

## Limitations

- All records are PROPOSED/PENDING_REVIEW; none is ACCEPTED; no
  ReviewDecision records exist. Independent Exact-SHA review must verify
  paraphrase fidelity, scope qualifiers, applicability and rights before
  any acceptance.
- SWEBOK-derived records cover only the inspected sections; KA9 sections
  1, 3-5, 7 and KA15 sections 1, 3-7, 9-10 are not cited by any record.
- CS2023-derived records carry the inherited text-extraction constraint
  (no PDF hash; heading-anchored locators) and the shared rights gap
  (HIGH/OPEN).
- The technique-accuracy ordering, the PMBOK/SWX/15939 characterizations
  and the quoted ethics clause are SWEBOK's statements, not independent
  findings of this lane.
- HUMAN_REVIEW_BURDEN=NOT_MEASURED.
