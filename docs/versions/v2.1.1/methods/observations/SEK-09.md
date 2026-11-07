# SEK-09 — Delivery-First qualification-method observation

**Task:** #351 · **Source of truth:** three exact-SHA final specialist review reports integrated under `wip/v2.1.1-knowledge` · **Decision:** REQUIRES_FURTHER_EVIDENCE (not Task PASS).

## Verified execution and method observations

- The three independent qualification lanes (#352–#354) inspected one immutable underlying graph baseline and wrote **disjoint one-file reports**. Each underwent independent review, a bounded in-place report-only repair, and a distinct final exact-SHA verification before integration.
- Exact reviewed heads: #355 `0d936d89a9139461059ad5fa0d65599b45e7909d`; #356 `ae1f2f298b8f45ad7842b19933928e77132a5812`; #357 `cb24278b20f86e7e244ab742dbe9bc881ed79ac5`.
- Actual WIP integration commits in order: `76fc192aa36ff6c712f15dd38abc2047d52eef63`, `c9c9754d0b03bcec5ff04ebdf0c5117b63074caa`, `9cbf7983c5b47c146bd934f39e10517c3a828b66`. Post-integration TREE: `fe8f5e106f349bb0c814696fcd231732f7568250`.
- Exact graph blobs were unchanged by report-only integration; this supports *scoped* reuse of earlier corpus PASS evidence. It does not grant a new PASS to report or coordinator bytes.
- Structural graph validity (793 nodes, 271 assertions, 9,461 equivalent triples, 66/66 self-tests) is independently distinguishable from source authority, contradiction completeness, record-level human acceptance and practical utility.
- Sampled provenance can be coherent while most records remain review-pending. A representative ten-claim audit is not exhaustive validation of all 106 claims.
- Eleven PARTIAL taxonomy views, 29 OPEN gaps, no Case/Counterexample or ReviewDecision and absent G4/G5 measurements are explicit failure-to-qualify evidence, **not** reasons to fabricate an ACCEPTED state.
- Source provenance/rights matter: CS2023 current educational use permission is not a blanket commercial permission; the SWEBOK publication-date conflict is UNKNOWN rather than arbitrarily resolved.
- Observed task-flow cost, throughput, handoff elapsed times and human review time: **UNKNOWN/NOT_MEASURED**; PR counts are not outcome progress.

## Repeatable lessons / next lowest-cost action

1. Keep separate reports for independent technical structure, source/evidence truth, and human/utility qualification, then integrate one coordinator decision; avoid concurrent writes to a shared knowledge model.
2. Repair report defects within the report's legal writer/write set before requesting fresh final verification; do not repeat unchanged-tree expensive tests to create activity.
3. Preserve an explicit **REQUIRES_FURTHER_EVIDENCE** outcome when evidence gaps are real, rather than calling a technically valid graph a validated knowledge asset.
4. Advance the bottleneck with a pre-registered limited human-review + answer-quality evaluation and true source-grounded examples, under a distinct lawful input/write set. Do not initiate SEK-10 before #351 qualification closure.
5. This is an experimental Tally corpus and method record, not an adopted generic standard; no automatic DomainForge synchronization or Domain Package production.

**Coordinator record scope:** `review/SEK-09-QUALIFICATION.md` and this file only. No new real-host/CI execution in this coordinator; final independent Exact-SHA review remains a gate.
