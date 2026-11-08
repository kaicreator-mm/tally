# SEK-09 — Final pilot-method observations and failed qualification outcome

**Core:** #351 · **Evidence source:** WIP `fdfae02701efc62020384fc469afe071aa7665b8`, #367/PR #375, #368/PR #374 and the original independent #352–354 specialist reports.

## Observed results, not plans

- **Deliverable reality:** 10 domain lanes generated a 806-record semantic graph with 9,653 RDF triples, 106 unresolved review-pending claims and 2 real source-backed incident Cases. The deterministic exporter, LinkML 1.12.1 and negative validators were verified; structural correctness did not settle domain truth.
- **Control experiment actually executed:** 14 localhost Qwen2.5-3B calls completed, two matched arms for 7 preregistered questions. Independent automated blind scoring recorded mean correctness **0.857 vs 1.857**, scope **1.571 vs 2.000**, traceability **1 vs 1**, unsupported claims **9 vs 0** for direct vs corpus-grounded. G5-04's omission worsened with grounding despite greater correctness; no reviewer-effort observation. No superiority claim/generalization.
- **Real process friction:** the first local execution machine lacked an inference server and recorded 0 attempted calls. A small free llama.cpp + local Qwen model ultimately produced actual trials; one same Writer owned Node24 runner repair, native execution and frozen evidence. It avoided ping-pong across Web/local.
- **Review distinction:** blind model scoring was carried out by a separate automated grader; it does not count as human G4. Owner-directed single-Writer final Exact-SHA for PR #375 (242 checks and manifest verification) is not falsely described as independent final review.
- **Experimental failure is information:** human record-level navigation/revision never occurred; 29 OPEN CoverageGaps, 11 PARTIAL TaxonomyViews, rights uncertainty, 0 Counterexample and 0 ReviewDecision remain. These defeat full qualification even though the graph and 7x2 benchmark are executable.

## Concrete method decision for SEK-10 retrospective

1. Preserve **three different thresholds**: build/schema **PASS**, observed localized model utility **evidence-with-limitations**, and knowledge-corpus acceptance **BLOCKED_NOT_QUALIFIED**. Do not convert one to another through terminology.
2. Next pilot needs a *small real-human-review budget* at scope freeze, not another document-only reviewer-readiness task; sample 10 records and record actual findings/effort, with fail-closed UNKNOWN on inaccessible sources.
3. Test both positive and negative examples with explicit counterexample-to-claim links; do not mislabel independent postmortems as contradictions to a claim that was never made.
4. Separate source identity, edition and rights eligibility from schema/provenance structural compliance; publishing a graph does not resolve publication-date conflicts or copyright reuse.
5. For utility comparisons, preserve raw frozen prompts/responses, blind key, scorer identity and raw hashes, predefine success thresholds and use independent/multiple graders or human sample audit where possible. A single 7×2 run cannot justify generalized superiority.
6. **Version-level closure criterion:** because #299 accepts unsuccessful experiments, finish this pilot with an explicit negative qualification rather than fake G4 PASS or accumulate endless readiness PRs. SEK-10 may document the tested failed method after #351 is formally closed; no immediate DomainForge or Domain Package adoption.

**Captured actual timing/cost:** 14 real generation calls, 6,069 recorded tokens, no paid API. Human review time, manual authoring effort, throughput/cost across all lanes remain **NOT_MEASURED**. Unknown measurements cannot be inferred from commit counts or task durations.

**Method decision:** `EXPERIMENT_COMPLETED_WITH_LIMITATIONS`; `CORPUS_ACCEPTANCE=NO`; `G4=NOT_CONDUCTED`; `G5=OBSERVED_NO_GENERAL_PASS`; recommendation `DO_NOT_PROMOTE` until new explicit human/source reviews.

---

## Historical early qualification-method observations (earlier WIP; superseded for current evidence only)

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
