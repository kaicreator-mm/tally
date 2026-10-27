# task-meta × ADS × Tally — Evidence-Governed Software Engineering Evolution Protocol (R0, Agent Edition)

**Date:** 2026-10-28. **Status:** cross-project Product/L2 research proposal; no normative freeze, review PASS, code authorization, Task Claim, approved BUILD, release or automatic adoption. Read Chinese original `ENGINEERING_EVOLUTION_PROTOCOL_R0.zh-CN.md` and the paired Domain Package blueprints.

> **2026-10-28 interpretive precedence:** Actual development and engineering work is **LLM Agent-first**, constrained by software engineering and real ADS/GitHub/K authority, not a human-driven approval chain. Independent review is normally a genuinely independent Agent role/context where allowed; it is not automatically a manual Reviewer. Escalate to humans only on material product ambiguity, nondelegable approval/permissions/physical Host, exceeded authorized risk/budget, or legally/policy-required release responsibility. Improvement measurement should distinguish qualified Agent autonomy, necessary vs unnecessary human intervention, role isolation and actual authorization, with honest coverage. See [Agent-native operating model](./AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.en.md).

## 1. Purpose and invariants

Build a **closed feedback loop from real software delivery** to three separately owned improvement products:

1. **task-meta — measurement/data plane:** credible, correlated, privacy-safe facts and coverage of engineering execution; not an agent scheduler or policy issuer.
2. **ADS — normative engineering process plane:** versioned and independently reviewed Task/Claim, role, evidence, Review/Repair, appropriate verification, change and Release Qualification standards; not an agent executor.
3. **Tally — software-engineering application/decision plane:** the actual project/version Domain App through which users design, build, verify, release and improve software; not the publisher of private task-meta metrics, and not a rival kernel.

**Authoritative boundaries:** Native GitHub Issue/Dependencies/Claim/PR/Review/CI/Release facts; project-adopted immutable ADS pin; one selected DomainHarness K Primary governing application admission/state/effects/journal/replay/UNKNOWN; Tally Business B recommending and coordinating; selected C adapters performing only K-admitted actions. Only ADS-version-qualified, bounded task-meta reports **published to GitHub** are visible to Tally. No direct Tally→task-meta API, DB, raw Trace/manifest dereference or credentials. Missing observation does not block otherwise legitimate Task execution.

### Canonical information and improvement route

```text
User product goal + adopted ADS/GitHub facts
     → Tally B Product/Planning/Journey/Assurance + selected S
     → legally authorized K/C GitHub/Agent/Host operations
     → actual source/Host/Review/CI/Release result
     → task-meta private collection Task→Attempt→Step/Tool/Model/Verifier
     → bounded, source-qualified ADS GitHub Task Report / Correction / coverage
     → Tally selected GitHub C readback → B version graph, diagnosis, learning
     → deduplicated owner-specific ImprovementProposal
       ├── data/measurement gap → task-meta Product/implementation process
       ├── process/normative gap → ADS versioned Rule/Golden process
       ├── app UX/orchestration gap → Tally Engineering B/S/C Product process
       └── demonstrated generic SDK gap → DomainHarness
     → candidate + counterexamples + baseline/holdout + independent review
     → explicitly authorized release, pin, or BUILD selection for a NEW version
     → real downstream consumer retest → outcome feedback
```

Do not create new authoritative cross-repo Task history or change another repository solely because an LLM suggests an improvement.

## 2. Two conceptual layers, four evolution timescales

**Within Task, fast decision fallback:** retain DomainHarness v0.3 historical preference from [#1398](https://github.com/kaicreator-mm/domain-harness/issues/1398): deterministic compiled Rule → exact semantic reuse → applicable qualified promoted subworkflow → bounded Harness/LLM fallback. This is a *conceptual tested design*; each real selected v0.10.x S/Host capability needs independently verified public ABI, not historical-name inference. Rule/procedure output is advisory; K/ADS source facts are authoritative.

**Task/Workfront adaptation:** after a verified attempt/CI/review/failure, Tally can recommend continue, bounded retry, repair, revert, alternate Agent/Host, human escalation or abstention. Evidence-unknown -> MORE_EVIDENCE/UNKNOWN. A material change to accepted scope, GitHub claim, writers or DAG requires existing lawful ADS amendment/handoff, not direct autonomous mutability. Owners: Tally #57/#58/#67/#145; task-meta observes only.

**Within-version replan:** handle real integration root defects, a reproducible flaky Host failure, duplicated Claim/PR, unnecessary Full loops, architecture mismatch vs local defect. Preserve prior accepted work and independent Review. Scope tests to affected evidence and risk; do not weaken P0 safety, provenance, CAS or necessary consumer validation for apparent speed.

**Across versions/projects:** collect PracticeCases, propose transferable Rule/StepTemplate/Procedure/Agent-Routing/DAG/Test patterns, run casebook regression, actual baseline and shadow/independent verification, then release:
- new collector/report dialect through **task-meta** qualified publisher/version;
- new normative rule via **ADS** approved version and explicit project adoption pin;
- new selected Tally B/S/C assets via **Product/L2/independent review/new closed-world BUILD/deploy**, preserving old occurrence pins.
A Candidate never self-promotes or hot-swaps a running production package.

## 3. Project-specific continuous improvement contracts

### task-meta — provenance and measurement quality

Collect source-correlated Task, Attempt, Step, Tool/Model/Verifier/Host, outcome, retry, duration/time-window and cost/token/usage facts **only when actually observed**. Distinguish measured, estimated, not observed, unsupported; identify observer coverage and publish correction to edited/duplicated reports. Never infer active Agent working hours from GitHub Issue `created_at→closed_at`, or claim provider bills where unavailable. Privacy constraints: no raw chain of thought, credentials, private prompt or private Trace in GitHub reports. Observer degradation never fails the Task. Qualify publisher→GitHub→Tally reader E2E, including source exact SHA/version/adoption and corrected/retracted records.

Owner references: [task-meta #12](https://github.com/kaicreator-mm/task-meta/issues/12), [#17](https://github.com/kaicreator-mm/task-meta/issues/17), [#27](https://github.com/kaicreator-mm/task-meta/issues/27), [#28](https://github.com/kaicreator-mm/task-meta/issues/28), [#29](https://github.com/kaicreator-mm/task-meta/issues/29). New measurement fields require real consumer need, provenance and bounded privacy/compatibility evidence, not a new telemetry warehouse.

### ADS — normalized process rules and policy validation

Input: authentic source-bound failure/correct practice, counterexamples, current ADS dialect, actual Risk/Safety/Quality scope, associated published metrics with coverage, competing goal and affected projects. Output: proposed smallest normative delta, explicit unchanged guarantees, positive/negative executable Golden, source/time-qualified comparison, exact independent review/RQ, and project-by-project immutable adoption. Examples:
- [DH #530](https://github.com/kaicreator-mm/domain-harness/issues/530): avoid false serialization; do not remove a genuine native dependency without authority.
- [DH #586](https://github.com/kaicreator-mm/domain-harness/issues/586): one legitimate campaign containing many checkpoints is a positive control against needless issue explosion.
- [DH #1013](https://github.com/kaicreator-mm/domain-harness/issues/1013): overlapping writers/claims and integration omission; enforce valid native scope and actual consumer checks.
- [DH #472](https://github.com/kaicreator-mm/domain-harness/issues/472): avoid speculative all-purpose engines when existing Fast Path solves the real user problem.
- [DH #1100](https://github.com/kaicreator-mm/domain-harness/issues/1100): P0 CAS safety invariants must not be downgraded to optimize speed.
- [DH #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385): repeat failure triage from actual evidence, not blind CI retry.

Owners: [ADS v4.19 #1347](https://github.com/kaicreator-mm/ai-development-standard/issues/1347), [#1345](https://github.com/kaicreator-mm/ai-development-standard/issues/1345), [#1346](https://github.com/kaicreator-mm/ai-development-standard/issues/1346). Rule changes should be proportionate: keep required security/independent Review/true Host/Release but avoid overhead for trivial noncritical changes. No new mandatory gate purely to collect more process evidence.

### Tally — engineering outcomes and Package learning

Input: user Goal/Requirement/Acceptance, adopted ADS version, current GitHub native source/Claim/Host/Review/CI/release, published task-meta summary, Agent/Host availability and evidence-qualified historical cases. B generates next lawful actions, valid parallel lanes and owner decisions, bounded repair decisions, live acceptance coverage, next-version plan, and approved candidate practice suggestions.

Concrete implementation owners: [Tally #91](https://github.com/kaicreator-mm/tally/issues/91) B domain contracts; [#92](https://github.com/kaicreator-mm/tally/issues/92) complete end-user UI; [#145](https://github.com/kaicreator-mm/tally/issues/145) Version Journey Driver; [#146](https://github.com/kaicreator-mm/tally/issues/146) Acceptance Coverage; [#139](https://github.com/kaicreator-mm/tally/issues/139) source graph; [#140](https://github.com/kaicreator-mm/tally/issues/140) process bottleneck/root; [#141](https://github.com/kaicreator-mm/tally/issues/141) next-version planning; [#143](https://github.com/kaicreator-mm/tally/issues/143) historical casebook; [#60](https://github.com/kaicreator-mm/tally/issues/60) qualified candidate→Package evolution.

Promotion: recorded PracticeCase → typed Candidate (applicability + contraindications + positive/negative evidence) → source-verified evaluation → counterexample regression/holdout/Shadow → independent reviewer, product authority and version approval → actual selected new sealed BUILD → real consumer use → adoption outcome/rollback. Start with a B rule/helper; split off an optional S feature only after **two actual downstream consumers** show a common gap, and physically verified public ABI supports it. No model self-rating, automatic normative process edits or hidden production package updates.

## 4. Prospective minimal exchange records (NOT released DTOs)

```ts
type EvidenceRef = {
  repository: string;
  githubUrl: string; exactSourceSha?: string;
  adsPin?: string; versionRef?: string;
  observedAt: string;
  source: "GITHUB_NATIVE"|"ADS_PUBLISHED_REPORT"|"DERIVED";
  coverage: "COMPLETE"|"PARTIAL"|"UNKNOWN"|"NOT_SUPPORTED";
};
type Finding = {
  kind: "FALSE_DEPENDENCY"|"WRITESET_CONFLICT"|"REVIEW_LOOP"|
        "ARCHITECTURE_ROOT"|"MISSING_INSTALLED_PROOF"|"HOST_GAP"|
        "FLAKY_TEST"|"UPSTREAM_SDK_GAP"|"TELEMETRY_GAP"|"UNKNOWN";
  evidence: EvidenceRef[];
  counterEvidence: EvidenceRef[];
  ownerCandidate: "TASK_META"|"ADS"|"TALLY"|"DOMAIN_HARNESS"|"APP";
  groundedness: "OBSERVED"|"HYPOTHESIS"|"UNKNOWN";
};
type ImprovementProposal = {
  evidence: EvidenceRef[]; expectedUserOutcome: string;
  applicableWhen: string[]; forbiddenWhen: string[];
  targetProject: string; targetVersion?: string;
  mechanism: "DATA_COLLECTION"|"NORMATIVE_RULE"|"B_PROCEDURE"|
             "S_FEATURE"|"C_ADAPTER"|"DOCS";
  exactTestCaseIds: string[];
  status: "INTAKE"|"TRIAGED"|"EVALUATED"|"INDEPENDENTLY_REVIEWED"|
          "APPROVED_FOR_NEXT_RELEASE"|"REJECTED"|"DEFERRED";
};
```

Use existing published ADS Github dialect FIRST; fields here must not be claimed as new `ads/github/v3` events, K permit, GitHub native state or finalized v0.10.x Package manifest. Keep expensive analysis as bounded, rebuildable Tally read projections with source hyperlinks.

## 5. Qualification of improvement, not narrative self-reward

Evaluate delivery/user outcome, real installed selected-path behavior, true release/next version, first-pass reviewed correctness, escaped/returned defects, blocked-by/currentness correctness, review/repair iterations, write-set conflict, external human wait and repeated CI. A task-meta measured active duration is distinct from calendar issue lead time; missing coverage means no quantitative productivity claim. Separate independent work lanes from critical joins and potentially dangerous write conflicts. Preserve safety/CI/review test coverage and task risk strata. Compare similar Task types, Agent/Host/model/tool versions and samples; abstain when unable to reconstruct as-of inputs. Use [Tally #143](https://github.com/kaicreator-mm/tally/issues/143) `RETROSPECTIVE_DIAGNOSIS` vs `HISTORICALLY_BLIND`; unreconstructable history is NOT_RECONSTRUCTABLE, not predictive success. Metrics are **feedback**, never sole gates or target incentives.

**Eight gates in order:** Source intake/coverage → ownership triage → candidate+counterexamples → proportional independent test+holdout → appropriate project Product/Review/RQ approval → exact pinned release/BUILD adoption → physical downstream retest → observed outcome/correction. Reasonable DEFER/REJECT is a correct terminal, not a failure to improve.

## 6. Six scenario Golden checks

| Source | Correct Owner/Proposal | Forbidden false action |
|---|---|---|
| [DH #530](https://github.com/kaicreator-mm/domain-harness/issues/530) | Tally DAG advisor + ADS dependency guidance | Silently delete native blocked_by |
| [DH #586](https://github.com/kaicreator-mm/domain-harness/issues/586) | B WorkUnit/Campaign positive pattern | Spawn one Task for every checkpoint |
| [DH #1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) | B write-set conflict + consumer acceptance | All PRs merged ⇒ Release PASS |
| [DH #1100](https://github.com/kaicreator-mm/domain-harness/issues/1100) | ADS mandatory CAS invariant | Weaken P0 safety for throughput |
| [DH #1264](https://github.com/kaicreator-mm/domain-harness/issues/1264) | B Acceptance Item selected-path test | Import/initialization ⇒ actual feature |
| [DH #1393](https://github.com/kaicreator-mm/domain-harness/issues/1393) | Upstream general SDK fix + downstream retest | Build separate Tally npm/K security subsystem |

## 7. First practical vertical slice

Reuse #143's 6–8 **source pinned** GitHub practice fixtures. Emit deterministic `Finding → CandidateOwner → bounded Proposal + contrary evidence` without requiring an LLM. In the same Tally Engineering B design, feed #145 next action, #146 acceptance and #140 diagnosis. Publish qualified, redacted task-meta metrics only if observed. Evaluate #1347 ADS Golden and existing task-meta schema issues as separate owners. The product-level test is a **real user completing two versions in Tally**, including actual Agent/Host, reviewer repair, installed behavior verification and release; only then compare costs/latency if telemetry coverage is sufficient.

**Status:** `RESEARCH_R0`; `NORMATIVE_ADS_CHANGE=NONE`; `TALLY_BUILD_PROMOTION=NONE`; `IMPLEMENTATION_NOT_ADMITTED`.
