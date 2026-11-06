# SEK-06A — Method Observation (Requirements Engineering lane)

**Task:** #322 (lane SEK-06A of the SEK-06 vertical-slice lanes)
**Date:** 2026-11-06
**Worktree / branch:** `tally-i322-zcode` / `task/tally-v211-sek06a-requirements`
**Baseline:** `7a70211b4460e4c753c97798670de9f34861d4a4` (after the Owner-authorized 2026-11-06 WIP rebase; original authoring base `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`)
**Status:** EXECUTING / EXPERIMENTAL / HUMAN_REVIEW_PENDING

## Research path — CS2023 SE section verification

1. The verified content base for this lane is the CS2023 Body of Knowledge PDF
   (`https://csed.acm.org/wp-content/uploads/2024/04/3.1-Body-of-Knowledge-1.pdf`,
   source `SE-SRC-CS2023`). The official PDF was obtained as **full text via a
   text-extraction service** on 2026-11-06; the SE section text (Preamble,
   Overview items, Core Hours table, and the complete SE-Requirements: Product
   Requirements topic list with learning outcomes) was captured verbatim into
   the lane research extract
   (`_tmp/tally-sek06-lanes/extract/cs2023-se-verified-anchors.md`).
2. **Direct binary download failed** and was not retried into evidence:
   csed.acm.org TLS connections were reset on both the direct route and the
   local proxy route, and the Internet Archive snapshot returned 429
   rate-limiting. Consequence: **no binary SHA-256 of the PDF is recorded**, and
   no record in this lane claims one.
3. Page anchoring therefore follows a strict rule: page-bearing locators use
   only the registry-verified anchors (SE Overview = PDF page indices 174-176;
   Core Hours table = 176-177; the shared locator set). The printed page token
   `175` observed adjacent to the SE heading in the text stream is consistent
   with the registry anchors but was not promoted to a new anchor. The one new
   KU-level locator (`se:LOC-CS2023-SE-REQ-KU`) is **heading-anchored** (exact
   unit heading + KA Core 1-6 / Non-core 7-9 topic numbering, verified against
   the extraction) and explicitly disclaims an unconfirmed printed page inside
   the locator string. No page number was invented anywhere in the lane.
4. Rights handling stayed inside the registry boundary: bibliographic metadata,
   exact locators, concise original paraphrases; short verbatim topic titles
   only where locator anchoring requires them; no substantial report text.

## Scope decisions forced by the source-identity hold

> **Current status (2026-11-06 P2 repair).** The hold described in this
> historical section no longer exists: SEK-02R #325 merged via PR #326 into the
> WIP base this branch was rebased onto, so SWEBOK identity/registry
> reconciliation is complete. The required post-merge reinspection of SWEBOK
> locators was performed in the P2 repair (see the final receipt section);
> KA1 sections 2, 5, 6 and 7 are now cited from direct PDF reinspection. The
> decisions below are preserved as the historical record of how the lane was
> originally authored.

- SWEBOK v4.0a is under source-identity hold (#325 unmerged), so this lane uses
  CS2023 as **curriculum authority only**. Every claim's scope field names the
  CS2023 SE curriculum and its topic number; no record is worded as a
  professional-consensus statement, and the absence of professional-BOK
  evidence is a first-class CoverageGap (`se:GAP-REQ-PROFESSIONAL-BOK-EVIDENCE`).
- CS2023 carries no real requirements-engineering case material and no
  case-class source is registered, so **no Case or Counterexample records were
  created**; the evidence hole is recorded as
  `se:GAP-REQ-REAL-CASE-EVIDENCE` instead of being papered over with invented
  examples.
- The anti-pattern's counterproductive characterization could not be sourced
  from CS2023, so it is derived through a lane-marked SYNTHESIS evidence record
  that combines two verified topic statements, with the inference disclosed at
  the evidence, at the anti-pattern, and on the `DERIVED_FROM` edge.

## Authoring correction process

The lane was drafted against the verbatim anchors first and corrected in
review passes before validation:

1. **Schema-conformance pass.** Two planned fields had no slot in the LinkML
   model and were removed rather than smuggled in: `label` on DecisionModel
   (class has no label slot) and `limitations` on AntiPattern (not a slot of
   that class). The anti-pattern's mandatory "this is lane synthesis, not a
   curriculum statement" disclaimer moved into its scope field, where it is
   still machine-visible.
2. **Scope-qualification pass.** Every claim, concept, principle, practice,
   pattern, anti-pattern and decision model was re-read to ensure the CS2023
   curriculum scope is stated (SWEBOK-hold rule), including the Non-core
   (elective) status of prototyping and product evolution wherever those topics
   carry weight.
3. **Provenance-hygiene pass.** Claim records carry no evidence list; their
   evidence links are canonical `SUPPORTED_BY` edges, matching the SEK-03 slice
   convention. Evidence-bearing edges cite existing evidence records only. The
   five snippet-pack shared records were copied verbatim from the shared
   snippet pack and verified parsed-identical to the pack before validation.
4. **Counterfeit-page pass.** Every locator string was checked against the
   registry anchors / extraction; no page number appears that the registry or
   the extraction does not support.
5. **Review-repair pass (exact tip, after the rights-clearance fix).** Four
   in-write-set corrections, no record added or removed: (a) the
   requirement-properties evidence statement now keeps the source's
   "including" hedge instead of a definite "the properties" enumeration;
   (b) README cites the shared rights gap by canonical ID, counts the sixth
   shared record, and points at the exact-tip receipt; (c) EXPLANATIONS moves
   the rights-gap entry into the Coverage gaps section and completes the
   shared-record inventory; (d) this validation section was rewritten to the
   exact-tip receipt (`coverage_gaps=3`). All six shared records re-verified
   parsed-identical to their shared sources (snippet pack; rights gap identical
   across the 06A/06B/06C worktrees).

## Validation — commands and results

Environment: Python 3.14.7, PyYAML 6.0.3, LinkML 1.12.1 / linkml-runtime
1.12.1, `PYTHONDONTWRITEBYTECODE=1`. Receipt history: the initial receipt
(pre rights-clearance fix, `coverage_gaps=2`) is retained under
`_tmp/tally-sek06-lanes/validation/06A/`; the second receipt covered the
rights-clearance fix tip `7aa2777bd27604eb7b18c923347d84eb98ee2c05` and the
then-repaired working tree (logs under
`_tmp/tally-sek06-lanes/validation/06A-bld2/`), both producing the identical
key lines quoted below.

**Historical receipt — exact committed-tip validation at `925ab717`** (the
full validator run and exit codes below bind to that committed tree, not a
working-tree approximation; it was the PR tip until the Builder receipt-rebind
commit `657297c` and is preserved here as receipt history — see the
final-tip receipt appended after this section): HEAD
`925ab717686a2d6e46d8df3018af675879eb56f4`, tree
`339e28827084363d7ab8540b5653347f2b480203`, base
`cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #327
headRefOid equalled this HEAD at that time), clean worktree before (0 dirty)
and after (0 dirty). Receipt dir `_tmp/tally-sek06-lanes/validation/06A-bld3/`
(`identity-tip.txt`, `selftest-tip.log`, `lane-validate-tip.log`,
`manifest-four-paths-tip.sha256`).

1. Repository validator self-test on the committed tree:
   `validate_knowledge_graph.py --self-test` → exit 0, final line
   `SELF_TESTS=66/66`.
2. Lane assembly and validation:
   `assemble_and_validate.py <worktree> 06A <records.yaml> <outdir>` → exit 0.
   Key output lines (exact committed tip; identical to the earlier receipts):
3. Four-path SHA256 manifest at this committed tip (repo-relative; the
   receipt-rebind commit that carries this receipt changes only this
   observation file, so the first three hashes remain valid at the pushed
   HEAD; the post-commit rerun and full-path manifest at the pushed HEAD are
   recorded in the receipt dir as `selftest-post.log`,
   `lane-validate-post.log`, `manifest-four-paths-post.sha256`):

```text
LANE_RECORD_COUNTS={"anti_patterns": 1, "concepts": 2, "coverage_gaps": 3, "decision_models": 1, "domains": 1, "evidence_records": 10, "knowledge_claims": 5, "patterns": 1, "practices": 2, "principles": 1, "relationship_assertions": 14, "source_locators": 4, "taxonomy_nodes": 1, "taxonomy_views": 1}
COUNT_source_locators=4 MIN=3 PASS
COUNT_evidence_records=10 MIN=3 PASS
COUNT_knowledge_claims=5 MIN=4 PASS
COUNT_concepts=2 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=2 MIN=2 PASS
COUNT_patterns=1 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=3 MIN=1 PASS
COUNT_relationship_assertions=14 MIN=10 PASS
DERIVED_SOURCES=1 se:SRC-CS2023
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

Four-path manifest at the committed tip `925ab717` (tree `339e2882`):

```text
10ad4ea3bb8307a2e95ca5c9076d1a9660fefee9e4fe3477087393f963791022  lanes/requirements/README.md
30d2c404f2063f548b88d6bcbcd657e034b9655ce498dcc89cb6daa70ee3314c  lanes/requirements/records.yaml
0ada45a5b3940c2c381dadad26ad5a0e650da1d28e311be59d5847edadf460c1  lanes/requirements/EXPLANATIONS.md
ab5a686cfeb89305a9e62e05783322d22734896589a27144a0e4508989292c33  methods/observations/SEK-06A.md
```

All record-count minimums PASS; the assembled lane graph passes the repository
graph validator (typed endpoints, evidence/qualifier rules, locator uniqueness,
no ACCEPTED knowledge), the no-ACCEPTED lane gate, and LinkML 1.12.1 instance
validation. RDF/JSON-LD export is not claimed at lane level; it belongs to the
SEK-08 integration experiment.

### Historical final-tip receipt — Builder batch `657297c` (2026-11-06 R2 repair append; old WIP base, superseded by the 2026-11-06 WIP rebase onto `7a70211`)

The R2 Exact-SHA review of PR #327 required the observation to name the final
Builder-batch tip; the `925ab717` receipt above (and the earlier `7aa2777b`
receipt) are historical context, preserved unchanged.

- Identity: HEAD `657297c63142661bd2da5ce1763b07dd68700667`, tree
  `b225e57d2c40a58c6f0bdc7d3c9d360540ca6fc6`, base
  `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (merge-base equals base; PR #327
  headRefOid equalled this HEAD at the R2 review).
- Commands and exit codes at this exact tip: `validate_knowledge_graph.py
  --self-test` → `SELF_TESTS=66/66`, exit 0; `assemble_and_validate.py
  <worktree> 06A <records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (counts unchanged:
  47 lane records, 14 relationship assertions, `coverage_gaps=3`, derived
  source `se:SRC-CS2023`). Clean worktree before/after: 0/0 dirty. Logs:
  Builder batch `_tmp/tally-sek06-lanes/validation/06A-bld3/`
  (`identity-post.txt`, `selftest-post.log`, `lane-validate-post.log`,
  `manifest-four-paths-post.sha256`), independently re-run for this repair in
  `_tmp/tally-sek06-lanes/validation/06A-r2r/`.
- Four-path SHA256 manifest at HEAD `657297c` / tree `b225e57`:

```text
10ad4ea3bb8307a2e95ca5c9076d1a9660fefee9e4fe3477087393f963791022  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/README.md
30d2c404f2063f548b88d6bcbcd657e034b9655ce498dcc89cb6daa70ee3314c  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/records.yaml
0ada45a5b3940c2c381dadad26ad5a0e650da1d28e311be59d5847edadf460c1  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/EXPLANATIONS.md
09eb732499983f1a4f87f6483d51025680e6db5c7d68f90153a6f3fbca98f4e8  docs/versions/v2.1.1/methods/observations/SEK-06A.md
```

This append itself is an observation-only R2-repair commit and becomes the
newest tip; its exact HEAD/TREE, the post-commit rerun (same commands, exit
0), the updated four-path manifest and clean-worktree evidence are delivered
in the R2 repair delivery comment on PR #327 / coordinator #321, with logs in
`_tmp/tally-sek06-lanes/validation/06A-r2r/`. The SWEBOK hold (#325/#326
unmerged) is unchanged: no SWEBOK-derived content was added by this receipt.

## Limitations and review burden

- The underlying PDF is verified as extracted text, not as a byte-identical
  binary; no PDF SHA-256 exists for this lane. Heading-anchored locators are
  the compensating control, and a reviewer with PDF access can confirm every
  topic statement against the named unit heading.
- Human comprehension and review time were **NOT_MEASURED**.
- No validation performed here constitutes human review or acceptance; every
  record remains PROPOSED or PENDING_REVIEW pending SEK-08 integration and
  human ReviewDecision.

```ini
PRODUCTION_ADOPTION=NO
FREEZE=NO
MERGE=NO
RELEASE=NO
DEPLOYMENT=NO
```


## Shared CS2023 rights-clearance gap

The CS2023 source registry limits permission to educational materials/programs
and requires specific permission for other uses. This lane is part of Tally's
general software-engineering knowledge graph; it is not presumed to qualify
under that permission. The shared HIGH/OPEN record
`se:GAP-CS2023-RIGHTS-CLEARANCE` is therefore included verbatim across SEK-06A,
SEK-06B and SEK-06C (verified parsed-identical in all three lane worktrees
during the exact-tip repair pass). No rights clearance is claimed. The
four-path correction initially postdated the prior lane-validation receipt;
the exact-tip native validation quoted above (post-fix, `coverage_gaps=3`,
`LANE_VALIDATION=PASS`) has since been run and supersedes it.

## WIP-rebase receipt — Owner-authorized rebase onto `7a70211` (2026-11-06)

Owner authorization recorded on #321 (2026-11-06) merged PR #326 (SEK-02R)
into `wip/v2.1.1-knowledge` — linear rebase merge, no merge commit — moving
the WIP head from `cf2dbc845bfc2dcbc110ed52e240e79953a4cba9` (tree
`da5deb5eac25cd019883772967d288a1f2679a84`) to
`7a70211b4460e4c753c97798670de9f34861d4a4` (tree
`f9b01c74f39993299d9ee179c76cdc08d3baae9d`). This lane branch was then
rebased with `git rebase --onto 7a70211b4460e4c753c97798670de9f34861d4a4
cf2dbc845bfc2dcbc110ed52e240e79953a4cba9`: all five lane commits replayed
with **zero conflicts**, and the three lane content files are byte-identical
to the pre-rebase reviewed tip `221c6cf8727d659e83a14238d834246e7dec354c`
(verified by path-scoped diff against a pre-rebase backup ref). The earlier
receipts above (final-tip `657297c` and its predecessors, all on the old
base) are **historical** and preserved unchanged.

- Identity at the rebased tip: HEAD `944cfee17670ea60e40a3251534196a60e3b3286`,
  tree `881fd8fa168b02a157051094cdfc572922ad0c12`, base / merge-base
  `7a70211b4460e4c753c97798670de9f34861d4a4`; clean worktree before/after:
  0/0 dirty.
- Validation rerun at that exact tip (Windows-native venv Python 3.14.7 /
  PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1,
  `PYTHONDONTWRITEBYTECODE=1`): `validate_knowledge_graph.py --self-test` →
  `SELF_TESTS=66/66`, exit 0; `assemble_and_validate.py <worktree> 06A
  <records.yaml> <outdir>` → exit 0 with every `COUNT_*` PASS,
  `GRAPH_VALIDATION=PASS`, `NO_ACCEPTED_RECORDS=PASS`,
  `LINKML_INSTANCE_VALIDATION=PASS`, `LANE_VALIDATION=PASS` (counts
  unchanged: 47 lane records, 14 relationship assertions, `coverage_gaps=3`,
  derived source `se:SRC-CS2023`). Cross-lane check across the three rebased
  tips (06A `944cfee` / 06B `eff5e86` / 06C `343f30b`):
  `CROSS_LANE_CHECK=PASS` — 6 shared records semantically identical in all
  three lanes, 148 unique lane IDs, merged three-lane graph
  `GRAPH_VALIDATION=PASS` + `NO_ACCEPTED_RECORDS=PASS` + LinkML PASS.
  Logs: `_tmp/tally-sek06-lanes/validation/06A-rb1/`
  (`selftest-tip.log`, `lane-validate-tip.log`, exit receipts) and
  `_tmp/tally-sek06-lanes/validation/cross-lane-rb1-tip.log` (all exit 0).
- Content manifest at the rebased tip (three content files unchanged from
  the R2-reviewed `657297c` / `221c6cf` manifest — the rebase changed only
  base history, not lane content):

```text
10ad4ea3bb8307a2e95ca5c9076d1a9660fefee9e4fe3477087393f963791022  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/README.md
30d2c404f2063f548b88d6bcbcd657e034b9655ce498dcc89cb6daa70ee3314c  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/records.yaml
0ada45a5b3940c2c381dadad26ad5a0e650da1d28e311be59d5847edadf460c1  docs/versions/v2.1.1/knowledge/vertical-slice/lanes/requirements/EXPLANATIONS.md
```

- SWEBOK state after the rebase: the corrected SWEBOK v4.0a source record
  (SEK-02R / PR #326, including the native PDF SHA-256 and the open
  title-page/landing-page date discrepancy) is now part of this branch's
  base, and the source-identity hold is lifted at the registry level.
  **No SWEBOK-derived claim, locator or evidence was added by this rebase**:
  direct SWEBOK reinspection was not performed in this stage, and every
  knowledge record remains PROPOSED / PENDING_REVIEW with the CS2023
  curriculum-only scope and the professional-BOK / real-case gaps intact.
- This append itself is an observation-only rebase-receipt commit and
  becomes the newest tip; its exact HEAD/TREE, the post-commit rerun (same
  commands, exit 0), the updated four-path SHA256 manifest and
  clean-worktree evidence are delivered in the rebase delivery comment on
  PR #327 / coordinator #321, with logs in
  `_tmp/tally-sek06-lanes/validation/06A-rb1-post/`.

## P2 repair receipt — SWEBOK KA1 reinspection + stale-hold correction (2026-11-06)

Repair batch executed under the combined handoff on #321 (R1 review of the
post-WIP-rebase tips, both findings P2), within the original #322 four-path
write set, against reviewed tip `322ed01509bc66f4f335821847ab73ca67c9f33d` /
tree `1b6eee58a805042480b8245194fcf2943e95bc67`, base
`7a70211b4460e4c753c97798670de9f34861d4a4`. Pre-write readback: PR #327 OPEN,
headRefOid equalled the branch ref and the reviewed tip; worktree clean; no
concurrent writer observed; write set unchanged.

### P2-01 — stale source-hold statements corrected

README.md, records.yaml and EXPLANATIONS.md no longer state that #325 is
unmerged or that SWEBOK is under source-identity hold. Current-state wording
now says: identity/registry reconciliation complete (#326 merged into this
lane's WIP base); content-level reinspection performed by this repair;
`se:GAP-REQ-PROFESSIONAL-BOK-EVIDENCE` preserved as a stable ID but redescribed
as a content/evidence coverage gap (uninspected KA1 sections 1/3/4/8; no human
review of SWEBOK-derived records), severity HIGH → MEDIUM. The shared Domain
and View records' hold language was reworded **identically in all three lane
files** (06A/06B/06C) so the six shared records remain byte-identical for
SEK-08. Historical statements in this observation are preserved and labeled.

### P2-02 — SWEBOK KA1 locator re-verification performed

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
| `se:LOC-SWEBOK-KA1-ELICITATION` | Ch.1 §2 Requirements Elicitation (§2.1 Sources, §2.2 Common Elicitation Techniques) | 47–48 | 1-6 to 1-7 |
| `se:LOC-SWEBOK-KA1-VALIDATION` | Ch.1 §5 Requirements Validation (§5.1 Reviews, §5.2 Simulation and Execution, §5.3 Prototyping) | 56–57 | 1-15 to 1-16 |
| `se:LOC-SWEBOK-KA1-MANAGEMENT` | Ch.1 §6 Requirements Management Activities (§6.1–6.3) and §7 Practical Considerations (§7.1–7.4) | 57–60 | 1-16 to 1-19 |

Chapter 1 opens at PDF page index 42 (printed 1-1). Added records: the three
locators above, three `DIRECT_SOURCE_PARAPHRASE` evidence records
(`se:EVD-REQ-SWEBOK-ELICITATION`, `se:EVD-REQ-SWEBOK-VALIDATION-METHODS`,
`se:EVD-REQ-SWEBOK-CHANGE-SCOPE`), three claims
(`se:CLM-REQ-SWEBOK-ELICITATION-ACTIVE`,
`se:CLM-REQ-SWEBOK-VALIDATION-TRIO`,
`se:CLM-REQ-SWEBOK-MANAGEMENT-AGREEMENT`) and three canonical `SUPPORTED_BY`
edges (`se:REL-REQ-015/016/017`). No page number, section or claim was written
that the extracted PDF text does not support; the Guide's
generally-accepted-knowledge-not-prescription limitation is carried inside each
claim. Rights boundaries kept: PDF not committed/mirrored/redistributed,
original paraphrases only, release-date discrepancy remains OPEN, all records
PROPOSED/PENDING_REVIEW, zero ACCEPTED.

### Validation (Windows-native, Python 3.14.7 / PyYAML 6.0.3 / LinkML 1.12.1 / linkml-runtime 1.12.1, PYTHONDONTWRITEBYTECODE=1)

Run on the exact repaired working tree (= the committed tree of this repair
commit); rerun after the commit and the final four-path SHA256 manifest at the
pushed HEAD are delivered in the P2 repair delivery comment on PR #327 /
coordinator #321, logs in `_tmp/tally-sek06-lanes/validation/06A-p2/`.

1. `validate_knowledge_graph.py --self-test` → `SELF_TESTS=66/66`, exit 0.
2. `assemble_and_validate.py <worktree> 06A <records.yaml> <outdir>` → exit 0:

```text
LANE_RECORD_COUNTS={"anti_patterns": 1, "concepts": 2, "coverage_gaps": 3, "decision_models": 1, "domains": 1, "evidence_records": 13, "knowledge_claims": 8, "patterns": 1, "practices": 2, "principles": 1, "relationship_assertions": 17, "source_locators": 7, "taxonomy_nodes": 1, "taxonomy_views": 1}
COUNT_source_locators=7 MIN=3 PASS
COUNT_evidence_records=13 MIN=3 PASS
COUNT_knowledge_claims=8 MIN=4 PASS
COUNT_concepts=2 MIN=2 PASS
COUNT_principles=1 MIN=1 PASS
COUNT_practices=2 MIN=2 PASS
COUNT_patterns=1 MIN=1 PASS
COUNT_anti_patterns=1 MIN=1 PASS
COUNT_decision_models=1 MIN=1 PASS
COUNT_coverage_gaps=3 MIN=1 PASS
COUNT_relationship_assertions=17 MIN=10 PASS
DERIVED_SOURCES=2 se:SRC-CS2023,se:SRC-SWEBOK-GUIDE-4-0A
GRAPH_VALIDATION=PASS
NO_ACCEPTED_RECORDS=PASS
LINKML_VERSION=1.12.1
LINKML_RUNTIME_VERSION=1.12.1
LINKML_INSTANCE_VALIDATION=PASS
RDF_JSONLD_EXPORT=NOT_CLAIMED_AT_LANE_LEVEL
LANE_VALIDATION=PASS
```

47 → 59 lane records (three content files plus this observation are the only
paths changed). Cross-lane check at the three repaired tips and clean-worktree
evidence are recorded in the delivery comment. Human review burden remains
NOT_MEASURED; no record is ACCEPTED.
