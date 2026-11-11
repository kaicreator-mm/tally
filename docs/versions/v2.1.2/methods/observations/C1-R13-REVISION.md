# C1-R13 owner-decision source revision — build and gate receipts (v2.1.2, single Writer)

**Task:** [Tally #398](https://github.com/kaicreator-mm/tally/issues/398) (C1 gate:
OD05 + GH-2018 SourceLocator producer decisions) · **Executed per:** #398 R13
Writer approval scope (comment `6808543942`) · **Writer:** exactly one (this task,
branch `task/tally-v212-c1-owner-revision`, worktree base WIP `706a3ec3`)
**Status:** BUILT / ALL_GATES_PASS / PENDING_INDEPENDENT_REVIEW — no ACCEPTED
record, no ReviewDecision, no Oracle attestation, no qualification promotion.

## 1. Decisions executed

| Decision | Disposition | Target | Change |
|---|---|---|---|
| OD05 | **NARROW_CLAIM** | `se:DM-REQ-FUNCTIONAL-DOC-METHOD` (requirements lane) | universal uniformity constraint removed → context-appropriate selection + rationale (CS2023 topics 1a–1c); option "applied uniformly" narrowed; no-superiority tradeoff note added; record_version 0.1.0 → 0.2.0; stays PROPOSED |
| GH-2018 | **ACCEPT_PATCH** | `se:CASE-09F-GH-DB-FAILOVER` (reliability lane) | + `se:LOC-09F-GH-RECOVERY-BACKLOG` (16:24 → 23:03 UTC, 2018-10-22); + `se:EVD-09F-GH-RECOVERY-OUTCOME` (PENDING, dropped-payload caveat); +1 case evidence-ref append; original 16:24 locator and both original evidence records unchanged |

Frozen items verified untouched in this branch: `docs/versions/v2.1.1/**`
(byte-identical to WIP baseline `706a3ec3`; gate STRUCT-1 checks the two lane files
against `git show HEAD:…`), pinned snapshot Blob `0f6007315fd7352fe6fd218effcfecf672cb4961`,
historical #299/#351, frozen R8.

## 2. First-party source readbacks (independent verification at execution time)

- **CS2023** (`https://csed.acm.org/wp-content/uploads/2025/11/CS2023-Report.htm`,
  read 2026-11-12): SE knowledge area → `SE-Requirements: Product Requirements`
  KA Core topic 1 = "Describe functional requirements using, for example, use cases
  or user stories." with subtopics **1a** "Using at least one method of documenting
  and structuring functional requirements.", **1b** "Understanding how the method
  supports design and implementation.", **1c** "Strengths and weaknesses of using a
  specific approach." **No uniform-across-requirement-set requirement exists.**
  Confirms the v2.1.1 universal constraint was unsupported by the pinned source →
  NARROW_CLAIM source-proof.
- **GitHub postmortem** (`https://github.blog/news-insights/company-news/oct21-post-incident-analysis/`,
  read 2026-11-12): timeline entry **2018 October 22 16:24 UTC** — replicas in sync,
  failback to original topology, "we kept the service status red while we began
  processing the backlog"; **16:45 UTC** — "over five million hook events and 80
  thousand Pages builds queued", "we processed ~200,000 webhook payloads that had
  outlived an internal TTL and were dropped"; **23:03 UTC** — "All pending webhooks
  and Pages builds had been processed and the integrity and proper operation of all
  systems had been confirmed. The site status was updated to green." Publisher
  metadata shows `article:modified_time: 2021-12-20` (HTML mutable). Raw HTML bytes
  and SHA-256 remain **unfrozen** (UNKNOWN retained in record uncertainty fields).
  16:24 ≠ backlog completion; 23:03 ≠ lossless delivery.

## 3. Build environment (pinned, identical to SEK-09H)

Python **3.14.7** / PyYAML **6.0.3** / rdflib **7.6.0** / linkml **1.12.1** /
linkml-runtime **1.12.1** / jsonschema **4.26.0** (isolated venv, build tooling
outside the repo write set per SEK-08/SEK-09H precedent).
Materializer: SEK-09H `materialize.py` adapted for v2.1.2 paths, `EXPECTED_COUNTS`
`source_locators 90→91`, `evidence_records 142→143`, and the SEK-09F evidence
provenance check 5→6; `BUILD_DATE 2026-11-07`, graph root and `graph_version 0.1.0`
intentionally preserved.

## 4. Gate results (all re-run at the final tree)

| Gate | Command (venv python, worktree `tally-i398-zcode`) | Result | Exit |
|---|---|---|---|
| Materialize+write | `materialize.py --worktree <wt> --out <generated> --write` | 9,680 triples; digest below | 0 |
| Merged instance | `emit_merged.py --worktree <wt>` | 808 records | 0 |
| Repository validator | inside `gate_c1_revision.py` STRUCT-3 (`vkg.validate_graph`) | 0 errors | 0 |
| LinkML | `gate_linkml.py` (schema `software-engineering.yaml`) | LINKML_ERRORS=0 | 0 |
| Export equivalence | `verify_exports.py --worktree <wt> --dir <generated>` | PASS 9,680 = 9,680 = 9,680 | 0 |
| Determinism | 5 assemblies × {canonical, reversed, rotate} | unique (digest, ttl_sha256, jsonld_sha256) = 1; regeneration byte-identical (`cmp` ×3) | 0 |
| Manifest | `sha256sum -c manifest.sha256` | graph.ttl OK; graph.jsonld OK | 0 |
| Negative fixtures | `gate_negative.py` (duplicate YAML key, duplicate locator, ACCEPTED case, dangling case ref, wrong-class counterexample, registry collision, unknown enum) | 9/9 rejected | 0 |
| C1 revision P/N/U | `gate_c1_revision.py` (OD05 9 checks, GH 11 checks, structural 3 checks) | **23/23 PASS** | 0 |

## 5. Exact outputs at the final tree (file SHA-256)

- `knowledge/generated/graph.ttl` sha256 `07a7b37d08429085fc4c94fe29ce3a3be94b328076a25d85c01afb7d5852a894`
- `knowledge/generated/graph.jsonld` sha256 `31d723b68dc43c230ae10bd5885155c5fcd53661f0334a83c6535f4b83220709`
- Canonical graph digest (sorted N-Triples SHA-256) `e56bed65b7f90d4bff1e6efdadd55abad0de30837e12e3e30be4c54e5687eac6`
- Counts: 808 records (796 lane-owned + 6 shared + 6 derived sources) → 809
  `@graph` objects; 9,680 triples (+27 vs SEK-09H); 91 locators; 143 evidence
  (142 PENDING_REVIEW + 1 PROPOSED); 106 claims (all PENDING, unchanged); 2 cases
  (PENDING_REVIEW); 11 decision models (1 at record_version 0.2.0); 0
  counterexamples; 0 review decisions.
- Git Exact HEAD/TREE/blobs: recorded in the follow-up source receipt on #398
  after commit (this file is part of the committed tree, so the commit SHA is
  only knowable after commit).

## 6. Remaining UNKNOWN (honest, not erased)

- Publisher raw-HTML exact bytes/SHA-256 not obtained; HTML revision mutable
  (observed `modified_time 2021-12-20`).
- Source-reported operational claims not independently audited; the later handling
  of the ~200,000 dropped payloads beyond the source account is unspecified.
- OD05: no source-established optimal method; no universal project-wide procedure;
  the narrowed DecisionModel is curriculum-scope only.
- All records remain PENDING/PROPOSED; independent Element/Global/Root semantic
  Oracle NOT_ATTESTED; SAP 0/6; Root UNKNOWN; DKE R8 0/53; DIS CORE-002/005/016
  PARTIAL. The historical R28 809-node DKE PROPOSAL_ONLY draft was **not** adopted;
  this revision is an independent, owner-authorized construction with Tally-native
  record ids.
