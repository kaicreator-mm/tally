# SEK-10 — Software Engineering Knowledge Pilot: Final Evidence-Backed Retrospective

**Version:** Tally v2.1.1 (experimental pre-package corpus / knowledge graph)
**Parent:** [#299](https://github.com/kaicreator-mm/tally/issues/299) · **Task:** [#381](https://github.com/kaicreator-mm/tally/issues/381) · **Predecessor:** [#351](https://github.com/kaicreator-mm/tally/issues/351) CLOSED, terminal verdict **BLOCKED_NOT_QUALIFIED**
**Execution:** ADS #1359 Delivery-First · **Baseline:** `wip/v2.1.1-knowledge` HEAD `86b3ca987aeb5e1d2322763a31ce9ea2dfc1c685`, TREE `11a20b4593b25e4644e70bf8048830d809ba7a0d`
**Decision:** **EXPERIMENT_WORK_COMPLETED_WITH_LIMITATIONS / KNOWLEDGE_ASSET_NOT_QUALIFIED / NO_ADOPTION**

This document records what was actually observed. In this pilot, **successful delivery of code, research artifacts or an experimental method does not imply an accepted Software Engineering knowledge base**. Source claims remain review-pending. A negative qualification result is an eligible experimental outcome under the version charter, not evidence that the missing quality gates passed.

## 1. What was built and what was genuinely verified

| Stage | Experiment and evidence | Observed outcome | Explicit boundary |
| --- | --- | --- | --- |
| Scope & taxonomy (SEK-00/01) | Software Engineering domain charter, ten candidate knowledge-area lanes and multiple taxonomy views; #300/#301 | Scope hypotheses and excluded neighboring domains documented | 11 TaxonomyViews still PARTIAL; 29 CoverageGaps OPEN; not independently coverage-complete |
| Authority and sources (SEK-02/03) | Source Registry, source identities/locator records, research and evidence slice; #302/#304, SWEBOK source reconciliation #325/#326 | Traceable source→locator→evidence chain was feasible | CS2023 rights for wider use, SWEBOK edition/date/rights and exact publisher content drift remain unresolved; the early fixture is historical, not authoritative merged corpus |
| Early LinkML PoF (SEK-04) | Pinned LinkML 1.12.1, A/B/C authoring fixtures and executable companion reference validator; #305, `experiments/linkml-pof/RESULTS.md` | 32/32 final self-tests; 12/12 conversions; 25/25 CLI probes; checked 45 ≅ 45 ≅ 45 core Turtle graphs; early decision **REPEAT_EXPERIMENT** | LinkML cannot by itself verify distributed reference closure, compound uniqueness or complete JSON-LD nested rdf:type fidelity; **no production adoption** |
| Full model (SEK-05) | Versioned `knowledge/model/schema/software-engineering.yaml`, catalogs, required identities and validation; #308 | Separated Source, SourceLocator, Evidence, KnowledgeClaim, TaxonomyView, Case, Counterexample, ReviewDecision and first-class RelationshipAssertion; fail-closed checks | Structural model validation does **not** confer human-accepted claims, publisher authority or adequate data coverage |
| Content production (SEK-06/07) | Parallel lanes #321–#347; separate owned YAML/README/source/method files | Ten lane bundles with source-bound claims and explicit gaps; concurrent non-overlapping writes possible | No independent human approval of all knowledge records; taxonomy/coverage debt retained |
| Assembly (SEK-08) | #349/PR #350; one central graph integrator and `knowledge/HANDBOOK.md` | Reproducible JSON-LD/Turtle, typed links and SHA256 manifest; 793 original nodes / 9,461 triples at initial qualification baseline | Historical input includes the excluded SEK-03 legacy fixture and projection-only YAML boundary-note repair; graph equivalence proves identity, not claim truth |
| Real incident ingestion (SEK-09F/H) | #365/PR #366; #368/PR #374; first-party Cloudflare and GitHub incident postmortems | **806 graph records / 9,653 RDF triples; two non-synthetic PENDING_REVIEW Cases**, 3 new Source, 4 SourceLocator, 5 Evidence; deterministic rebuild, LinkML 1.12.1, 66/66 repository self-tests and 9/9 negative fixtures with independent exact-head evidence | **0 Counterexamples**, 0 ReviewDecisions, no new ACCEPTED knowledge, no contradictory claim fabricated; official web revisions are not content-hash-pinned |
| Actual utility trial (SEK-09E/G) | Frozen prereg #361/PR #363; #367/PR #375; `experiments/qualification/runs/SEK-09G/trials/actual-001/` | **14/14 actual local Qwen2.5-3B-Instruct Q4_K_M model requests**, 7 direct (D) + 7 corpus-augmented (K), frozen requests/responses, SHA256 receipts, *separate automated* blind scoring of all 14 outputs | Single model, 7 questions, one sample each; grader is **not a human**; review effort/time not measured; no preregistered binary PASS criterion |
| Final qualification (SEK-09) | #351, specialist #352–#354; final coordinator PR #380 | Terminal **BLOCKED_NOT_QUALIFIED**; original source/graph specialist results preserved; question evidence has a legitimate observed result | G4 human record-level review **NOT_CONDUCTED**; G0/G1/G2 PARTIAL, G3 structurally PASS, G5 OBSERVED_WITH_LIMITATIONS; **corpus acceptance NO** |

No Tally runtime/UI or DomainForge package implementation was part of v2.1.1.

## 2. Observed model-utility comparison — and why it is insufficient for general adoption

**Pinned trial basis:** registration `SEK-09E-UTILITY-PREREGISTRATION.md` blob `5498f830ab7e832e97c9b6a879bf1c1d6d8fddcd`; old graph blob `2d4498dfa5f90bcf57cc81e9c155fb9ede31bf00` (before the two Case additions). **Do not retroactively call this an experiment on the 806-node graph.**

- Loopback-only llama.cpp/Qwen2.5-3B local execution, 14/14 responses, 6,069 observed tokens, no paid API or API key. Actual local run UTC 2026-11-08 08:30:22–08:34:39.
- Actual independent **model** blind grader was distinct from answer generation; the grader only saw frozen blind answers, source context and a blank scoring template. It did **not** receive the D/K mapping. It was **not** a real human domain reviewer.
- Per-question scored averages, on preregistered 0–2 rubrics: **correctness D 0.857 / K 1.857; scope D 1.571 / K 2.000; traceability D 1.000 / K 1.000**. Source-unsupported assertions: **D 9 / K 0** for the 14 responses; average omission rate across the 7 question pairs **D 0.4762 / K 0.1667**.
- **Countervailing evidence:** G5-04 K omitted *more* frozen criteria than D even though K's correctness rating was higher; none of the 7 pairs improved the separate traceability rubric. Averages must not mask regressions or imply unconditional causal benefit.
- Runtime/evidence checks recorded **58/58 final SHA256SUMS**, **47/47 pre-scoring freeze SHA256SUMS**, Node 24.21.0 runner/scorer self-tests, frozen input preflight and 242/242 Builder mechanical checks. PR #375 final HEAD `2ae1d435b967312ee546f2f34d06d5b753703056` was merged to WIP `fdfae02701efc62020384fc469afe071aa7665b8`. Its final Exact-SHA audit was *Owner-directed single Writer*, **not an independent final content review**.
- There was **one** small model, **one** answer per arm per question, and **one** automated grader. Review burden = NOT_MEASURED, no human assessment and no statistical sampling/variance or generalization. The only defensible result: the corpus-augmented arm **scored better on certain recorded metrics in this small, frozen experiment**. It neither establishes cross-model superiority nor verifies the underlying claims.

## 3. Failure/limitation register retained on version exit

| ID | Severity (pilot) | Residual fact | Consequence / future smallest decisive test |
| --- | --- | --- | --- |
| HUMAN-01 | **P1 acceptance blocker** | G4 human reviewer did not navigate and adjudicate 10 fixed evidence chains, no human verdict or observed effort; all 106 Claims still PENDING | Engage one or more genuine qualified people, freeze a bounded 10-chain versioned sample, record identity/conflict/time/source rationale and revision/rejection; no model-generated signatures |
| COVERAGE-01 | P2 | **29 OPEN CoverageGaps**, **11 PARTIAL views**, ten domain lanes are a slice not a proven complete taxonomy | Define reviewed taxonomy-relative minimal coverage profile with numerator/denominator; resolve or explicitly exclude gaps before any completeness claim |
| COUNTER-01 | P2 | Two real incident Cases illustrate reliability failure modes but **no supported contrary claim pair**; 0 Counterexample and 0 CONTRADICTED_BY edges | Introduce a verifiable actual contradicting observation and its correctly scoped target claim, or state NOT_FOUND after bounded search; never invent strawman contradiction |
| SOURCE-01 | P2 / rights-sensitive | CS2023 license and commercial reuse not cleared; SWEBOK date/edition discrepancies and link drift; some exact source bytes unpinned | Recheck original publisher edition/locator/rights with proper clearance; no broad copyrighted-text mirroring; avoid calling a metadata-only ISO citation an inspected standard |
| UTILITY-01 | P2 | G5 sampled 7×2 results only, traceability did not improve, G5-04 omissions regressed, human reviewer effort unmeasured | Precommit pass threshold, second model, repeats, actual human blind audit and measured review effort; reserve disjoint holdout and rights-safe source excerpts |
| GOVERN-01 | P2 governance | #375 and #380 final coordinator/merge verification were transparently Writer-owned, not a distinct independent final Exact-SHA review | Prior to **any future positive promotion**, commission actual independent exact-head review and record precisely what it checked; self-verification is not interchangeable |
| BUILD-01 | P3 method caveat | Generated graph materializer used local workspace tooling during SEK-08; manifest/source inputs checked, but build-script packaging/reusability remains a separate productionization concern | For a future producer product, version and test the full generator CLI from the repo; do not claim generic portability from a one-off workspace helper |

**No P0 issue is asserted from the original sampled qualification reports.** The register above is a **retrospective prioritization** of known limitations, not a new exhaustive security/knowledge-quality audit. The full graph and all claims have not undergone human acceptance.

## 4. Method choices validated or rejected by observation

1. **Separating structured knowledge from evidence and review events was useful.** Separate Source → SourceLocator → Evidence → KnowledgeClaim, typed RelationshipAssertion with provenance and ReviewDecision event format prevents accidental equation of an RDF triple with human-accepted truth. Actual *human* ReviewDecision effectiveness has **not** been demonstrated.
2. **A single taxonomy tree was not enough.** Multiple TaxonomyViews plus explicit CoverageGap preserved ambiguity and made scope debt queryable. No evidence justifies declaring all eleven views complete.
3. **LinkML had a fit for schema-constrained experiment, not an entire knowledge pipeline.** Its limitations required extra validators for referential closure, unique source/locator keys, typed RDF relation identities and deterministic export. Early PoF's `REPEAT_EXPERIMENT` is the recorded narrow decision; expanded model use in Tally does not automatically mean adoption by DomainForge or another runtime.
4. **Deterministic content is not independently credible content.** Export and negative-test success support reproduction of the same corpus; domain source relevance, contradictory claims and rights remain separately gated.
5. **Real examples improved the model test surface.** Postmortem-derived Case records were successfully added with provenance, without inventing a Counterexample when none mapped to a real claim.
6. **Freeze before measuring.** The G5 preregistration, exact source corpus blob and stored raw request/response files reduced post-hoc evaluation leakage. The trial still had one automated grader and no human verification. This is a pilot observation, not evidence of robust scientific generalization.
7. **Delivery-First improvement was causal to completing the task, not to validating knowledge truth.** A prior `PROVIDER_UNAVAILABLE` attempt consumed time without answers; same-Writer provisioning/repair under #367 eventually produced real 14-call evidence and stopped Web/local validation ping-pong. PR/review count should not substitute for milestone acceptance.
8. **A failed qualification should terminate, not spawn indefinite preparedness reports.** #351 documents the negative outcome. Experiment work may finish with `NOT_QUALIFIED`; future G4 remediation belongs to new explicitly scoped work, not imaginary human records inserted into this frozen history.

## 5. Actual cost and effort evidence (no invented estimates)

| Item | Observed / verified | Not inferable |
| --- | --- | --- |
| G5 requests | 14 successful requests in 7 isolated pairs | Reliability across models or prompts |
| Generation model | Qwen2.5-3B-Instruct Q4_K_M, local llama.cpp, non-billable | General production speed/cost |
| Tokens | 6,069 recorded for those 14 requests | Token-based billable API costs (none occurred) |
| Run wall-clock interval | Approx. 4m17s from recorded UTC start/end for actual trial only | Total human engineering/authoring time or full-project task throughput |
| Raw frozen evidence | 47 pre-grade and 58 final files verified by SHA256 receipts | Independent source audit of all 106 claims |
| Human review | **NOT_CONDUCTED / effort NOT_MEASURED** | Human review burden reduction |
| Review/Writer | Independent *automated* grader; Writer-owned final SHA verification on #375/#380 | Separate source-level human acceptance or independent final-verifier substitute |

## 6. Candidate learnings for DomainForge — **NOT ADOPTED**

The version has two families of assets: **instance knowledge** (sources/claims/cases/typed graph/handbook) and **method evidence** (schema/validators, model comparisons, failure modes, authoring decisions). Neither is silently transferred.

| Candidate | Evidence from Tally pilot | Promotion policy |
| --- | --- | --- |
| Versioned domain knowledge manifest with strict Source/Evidence/Claim identities | G3 deterministic dual RDF export; 806-record graph | Candidate for separate DomainForge architecture review **only**, not a new production contract |
| Companion referential/semantic validator next to LinkML | #305/#308 negative fixtures and LinkML conversion limits | Evaluate alternative schema/validation stack and migration in an independent demo before adoption |
| Multiple taxonomy views and first-class CoverageGap | 10 lanes/11 partial views, 29 gaps retained | Test on another domain; do not confuse Software Engineering-specific content with universal ontology |
| Human review workflow based on exact claim/evidence chain | Packet prepared, **human experiment NOT_RUN** | **DO NOT PROMOTE** as a proven human-review process until actual review data exists |
| Frozen paired LLM evaluation with blinded rating | 14 real calls + separate automated scorer, provenance hashes | Demonstration candidate; needs broader models, human grader and precommitted threshold before generic gate |
| Negative-qualification gate and honest release boundary | #351 terminal `BLOCKED_NOT_QUALIFIED` with explicit G0–G5 | Potential ADS/DomainForge lesson; **no automatic standard adoption**, no release authorization |

**Promotion recommendation:** `DO_NOT_PROMOTE_NOW`; later `REVIEW_CANDIDATE` only, with exact Git SHA, source/rights clearance, real human G4 result and cross-domain proof. Tally v2.1.1 remains the evidence owner, DomainForge is observer/reference.

## 7. Final two-axis status

**Production workflow:** research → source registry → atomic evidence claims → ten-lane records → deterministic graph → sample qualifications and genuine 7×2 model trial → **terminal negative knowledge qualification** → method retrospective. **Executed as a bounded experiment**.

**Knowledge asset:** `PRE_PACKAGE` / `PENDING_REVIEW` / **NOT_QUALIFIED**; not a complete accepted Software Engineering domain; no Domain Package, runtime/Harness consumer or DomainForge adoption.

**Task DAG accounting:** SEK-00..08 delivered (9 positive task completions). SEK-09 review was **formally completed with a negative qualification result**, not PASS. SEK-10 produces this final retrospective. Hence finishing all 11 *work-item dispositions* must not be represented as "11/11 acceptance gates PASS"; project qualification success remains **NO**.

**Exact evidence roots:** [#299](https://github.com/kaicreator-mm/tally/issues/299), [#351](https://github.com/kaicreator-mm/tally/issues/351), [#367](https://github.com/kaicreator-mm/tally/issues/367), [PR #374](https://github.com/kaicreator-mm/tally/pull/374), [PR #375](https://github.com/kaicreator-mm/tally/pull/375), [PR #380](https://github.com/kaicreator-mm/tally/pull/380), `knowledge/generated/manifest.sha256`, `experiments/linkml-pof/RESULTS.md`, `experiments/qualification/runs/SEK-09G/trials/actual-001/`.

**No main/release/tag/deploy, DomainForge adoption or accepted knowledge transition is within this retrospective.**
