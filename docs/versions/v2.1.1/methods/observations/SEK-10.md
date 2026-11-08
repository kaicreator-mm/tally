# SEK-10 — Method Observation and Final Pilot Handoff

**Task:** #381 (SEK-10), parent #299, prerequisite #351 formally CLOSED with negative qualification.
**Authoritative starting commit:** `wip/v2.1.1-knowledge@86b3ca987aeb5e1d2322763a31ce9ea2dfc1c685`; tree `11a20b4593b25e4644e70bf8048830d809ba7a0d`.
**Primary detailed report:** `docs/versions/v2.1.1/retrospective/SEK-10-EXPERIMENT-RETROSPECTIVE.md`.

## 1. Actual final outcomes

- LinkML v1.12.1 early A/B/C PoF ran with 32-case final companion checker, 12 conversions and 25 CLI probes; **early decision REPEAT_EXPERIMENT**, no production/DomainForge adoption. A source/locator integrity checker was required beyond LinkML's native cross-file capabilities.
- The expanded experimental model and controlled lane authoring produced a **806-record, 9,653-triple** RDF/JSON-LD graph and human-readable handbook. Source/evidence link validation and deterministic export passed; real Case intake added 2 `PENDING_REVIEW` incident records, not synthetic Counterexamples.
- Human qualification **G4 NOT_CONDUCTED**; 106 Claims remain pending, 0 Counterexample, 0 ReviewDecision, 11 partial taxonomy views and 29 open gaps. Rights/source authority constraints remain. The `SEK-09` coordinator formally concluded **BLOCKED_NOT_QUALIFIED**, *not* ACCEPTED or G4 waived.
- Controlled G5 actual local sample: 7 D + 7 K Qwen2.5-3B-Instruct calls, independent automated blind grading, correctness means 0.857 vs 1.857, scope 1.571 vs 2, traceability 1 vs 1, unsupported claims 9 vs 0. One K answer worsened frozen-criterion omission (G5-04); human review effort unmeasured; no superiority/pass inference. Model provider failure was resolved locally with a same-Writer inference setup.
- PR #375 and #380 used transparent **Writer-owned exact SHA checking**, *not distinct independent final reviews*. Original specialist graph/source checks are not relabeled or silently inherited as reviews of new findings.

## 2. Recoverable process improvements

- **Stop report inflation:** count a core DAG node only on its stated outcome, not PRs or supplemental readiness subtasks. If G4 requires a human, appoint one early or explicitly accept a negative terminal result rather than treating a blank packet as progress.
- **Smallest executable demo:** freeze 7 questions, exact source identifiers, blind rating rules, same-model request isolation, actual returned content and hashes. A local-free model removes the billable API dependency but not sampling/scorer bias.
- **One Writer per write set:** retain Writer responsibility for validation failures and repair. Disjoint ten-lane authoring enables concurrency; the generated graph has one integration Writer. The distinction limits Web/Local round trips.
- **Structural vs semantic quality gates:** a LinkML/JSON-LD/TTL/negative-test PASS is not independent claim correctness, coverage completeness, licensing clearance or human reviewer acceptability.
- **Keep negative evidence:** no counterexample link should be manufactured merely to satisfy an acceptance checkbox. Record source/date disputes and G5 regressions rather than averaging them away.
- **Preserve exact historical records:** immutable prereg corpus, old source graph hash, final new graph hashes and G5 answer artifacts remain separate; later content was not leaked into old G5 trials.

## 3. Decisions and follow-up constraints

| Axis | Final disposition | Future requirement |
| --- | --- | --- |
| Task workflow | **SEK-10 retrospective delivered** (upon WIP merge and Issue closure) | No gratuitous additional validation-only task |
| Experiment | **Executed with failures / lessons captured** | New pilot may use smaller human-first scope |
| Knowledge | **NOT_QUALIFIED / PENDING_REVIEW** | Source rights and real G4 adjudication, taxonomy coverage, counterexample evidence |
| LinkML/schema | **Experimental, not adopted for production or DomainForge** | Independent alternative comparison and full-model migration qualification |
| DomainForge | **REFERENCE_ONLY / NOT_PROMOTED** | Explicit separate architecture and evidence-based approval |
| Release | **WIP-only, no main merge/tag/release** | Never conflate retrospective completion with a qualified knowledge release |

**Unmeasured:** human review effort, full authoring/rework costs, multi-model score robustness, statistical superiority and corpus-wide claim accuracy. **Measured:** 14 paired experiment calls / 6,069 tokens / exact raw response hashes; LinkML and graph verification counts as credited in their scoped receipts.

Do not fabricate human G4, an `ACCEPTED` or `ReviewDecision` transition, source reuse rights, a fully complete ontology, new DomainForge ownership or any external runtime effect. The next version must choose whether to spend real human-review capacity or explicitly accept another negative result.

**Delivery plan:** create/update only this observation and the named retrospective file; inspect final HEAD/TREE + `trailer-check`, record single-Writer vs independent status honestly, then WIP-only integration and #381 closure. Parent #299 can report **all 11 tasks dispositioned** but **knowledge acceptance failed**.
