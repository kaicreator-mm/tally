# Tally v2 Engineering Domain Package Blueprint from Real DomainHarness Iterations — R0 (Agent Edition)

**Status:** PRODUCT/L2 RESEARCH CANDIDATE, NOT FROZEN. Not a DomainHarness public ABI, Task Claim, implementation authorization, review verdict or Release Qualification. Date: 2026-10-28.

**Authoritative product intent:** A non-expert user completes a real software project from natural-language goal, through Product/L2/parallel Task DAG, Agent implementation, independent review/repair, selected real validation, authorized release, and a subsequent version **inside Tally**. Refer to [#63](https://github.com/kaicreator-mm/tally/issues/63), [#91](https://github.com/kaicreator-mm/tally/issues/91), [#92](https://github.com/kaicreator-mm/tally/issues/92), [#145](https://github.com/kaicreator-mm/tally/issues/145), [#146](https://github.com/kaicreator-mm/tally/issues/146), and [#143](https://github.com/kaicreator-mm/tally/issues/143). Chinese detailed counterpart: `TALLY_ENGINEERING_DOMAIN_PACKAGE_BLUEPRINT_R0.zh-CN.md`. Governance: `ENGINEERING_EVOLUTION_PROTOCOL_R0.zh-CN.md`.

## 1. Hard authority boundaries

- **GitHub** owns native Issue, dependency, accepted Claim, PR, commit, reviewer/CI and Release facts. **ADS** governs their version-pinned meanings, admissibility, Agent/role/provenance, reviews, amendments and qualification.
- **DomainHarness single selected K Primary** owns runtime state, CAS, admission, effects, journal, replay and UNKNOWN. A Tally-derived readiness decision, Issue comment, Task Report or model output is never K permission.
- **Tally Engineering B** owns software-engineering business decisions, project/version coordination, planning, human-in-the-loop UX, Agent routing recommendations, acceptance coverage and improvement candidates.
- **Selected S** supplies general-purpose Rule/Decision/Context/Workflow/optional Guided Journey, evidence reuse and Shadow where actually exposed through approved public ABI; **selected C** provides GitHub, Agent, Model, CI/Host and other authorized I/O.
- **task-meta** privately captures Task/Attempt/Tool/Model/Verifier/time/cost facts; the **only Tally interchange** is safely published, ADS-version-qualified GitHub Task Report / Learning evidence. Never add a direct Tally→task-meta API/DB/private Trace/manifest dependency; missing telemetry = UNKNOWN, not execution failure or zero.
- SQLite is a rebuildable read projection or legitimate K/Host store under its existing contract, not a competing native Github Task or DomainHarness K source of truth.

## 2. Software engineering Body of Knowledge → concrete practice patterns

Standards/knowledge provide reasoning tools, **not a claim of formal compliance**:
- ISO/IEC/IEEE 29148 requirements traceability → Product acceptance/intent and changed-requirement ancestry; verify actual user outcomes.
- ISO/IEC/IEEE 12207 life-cycle processes and SWEBOK → distinct requirements, design, construction, V&V, maintenance, configuration and release concerns, but no mandatory ceremony for trivial tasks.
- ISO/IEC 25010 product-quality characteristics and risk-based V&V → assess security, reliability, usability, maintainability and portability when **applicable to this product**, not blanket every-task full test matrices.
- Domain-Driven Design aggregates/bounded contexts → Issue is a container, not necessarily a single WorkUnit/Claim; business ownership separate from infrastructure.
- Lean/value engineering & reuse-before-build → compare delivered user value, incumbent capability, implementation/maintenance cost and acceptance risk before introducing an abstraction.
- Critical-path / PERT-DAG / architecture decisions → stages and independent subrequirement lanes coexist; one late compatibility join must not serially block all early source work.
- Root cause analysis and change impact → distinguish architectural mismatch, integration seam, brittle test baseline, true P0 safety regression, flaky infrastructure and a nonblocking edge condition.
- DORA/SPACE-style multi-dimensional observation → do not optimize on one scalar throughput measure or issue closure count; observed active Agent time differs from calendar lead, Human/CI wait and missing telemetry.

### Grounded cases and their expected business behavior

| DomainHarness source | Failure / correct pattern | Required Tally response |
|---|---|---|
| [v0.1 #39](https://github.com/kaicreator-mm/domain-harness/issues/39) | 117/117 visible tests != Hidden/release | Show Release-specific evidence and lawful next actions |
| [v0.2 #99](https://github.com/kaicreator-mm/domain-harness/issues/99) | Real Android/Hermes process-restart gate | Exact Local Host handoff; proxy != physical PASS |
| [v0.3 #177](https://github.com/kaicreator-mm/domain-harness/issues/177), [#180](https://github.com/kaicreator-mm/domain-harness/issues/180) | 13 consumer App contracts expose local binding/provisioning gaps | Consumer acceptance and owner attribution, not speculative generic engine |
| [v0.4 #366](https://github.com/kaicreator-mm/domain-harness/issues/366) | Typecheck baseline blocks later PR | Minimal scoped repair, then dependent lane re-evaluation |
| [v0.6 #472](https://github.com/kaicreator-mm/domain-harness/issues/472) | Re-scope from generic goal engine to proven Fast Path | Reuse/defer/build alternatives with value/ROI hypotheses |
| [v0.7 #530](https://github.com/kaicreator-mm/domain-harness/issues/530) | Unnecessary previous-version final-source dependency | Keep independent lanes running; compatibility join late |
| [v0.7 #586](https://github.com/kaicreator-mm/domain-harness/issues/586) | Verification Campaign with multiple Checkpoints | Reuse one campaign; checkpoint terminal != new Feature |
| [v0.8 #1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) | Overlapping PR/Claim and disconnected Host→K composition | Conflict/readback; selected-path integration required |
| [v0.8.1 #1100](https://github.com/kaicreator-mm/domain-harness/issues/1100) | CAS/effect regression in successor | Carry forward required safety invariants; preserve old verdict |
| [v0.9 #1119](https://github.com/kaicreator-mm/domain-harness/issues/1119) | External clean-installed two-B consumer | Public packaged install+selected call evidence, not internal exports |
| [v0.10 #1264](https://github.com/kaicreator-mm/domain-harness/issues/1264) | Imported/initialized hooks never invoked by K | Check executed behavior coverage, not imports/PR merged |
| [v0.10.2 #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385) | 4/15 repeat failures of Windows WAL setup | Correctly distinguish reproducible failure from isolated PASS; bounded diagnosis |
| [v0.10.3 #1393](https://github.com/kaicreator-mm/domain-harness/issues/1393) | Real Tally consumer reveals external npm Provider integrity gap | Route bounded upstream fix, then actual downstream retest |
| [v0.10.3 #1411](https://github.com/kaicreator-mm/domain-harness/issues/1411) | Source DAG does not equal native dependency admission | Explain missing Host/GitHub mutation/readback and do not claim CLAIM_READY |

All are **retrospective evidence candidates**. Do not claim accurate pre-finding prediction without time-valid as-of input reconstruction. Reuse casebook #143.

## 3. Proposed business bounded contexts (B Domain Package responsibilities)

These are **logical** packages/concerns. The physically packaged/selected B composition must match the actual verified DomainHarness v0.10.2/v0.10.3 public BUILD/Host ABI. Start with coherent concerns within one selected B if that is the minimal qualified design. Do not require one deployable Package per table row.

| Logical B context | Business operations (PROPOSED, not public SDK functions) | Owned decision / nonowned authority |
|---|---|---|
| Product/Project | `proposeProduct`, `proposeAcceptance`, `evaluateAmendment` | Requirements/alternatives; ADS Review/Freeze is external |
| Architecture/Planning | `proposeArchitecture`, `decomposeIntoLanes`, `evaluateDAG`, `proposeAgentHost` | WorkUnit/readiness advice; native DAG and accepted Claim external |
| Version/Execution Journey | `nextActions`, `prepareHandoff`, `proposeDispatch`, `reconcile` | Human inbox, orchestration candidate; K owns Effect, GitHub native execution facts |
| Assurance/Release | `explainCoverage`, `proposeRepair`, `selectChecks`, `proposeRelease` | Test strategy and coverage; actual CI/Host/Review/RQ external |
| Experience/Improvement | `diagnoseBottleneck`, `proposeNextVersion`, `proposePracticeCandidate`, `evaluateApplicability` | Evidence-backed candidates; ADS rules and promotion/BUILD authority external |

**Application shell** (#92): one project workspace, progressive adaptive steps, user decisions, current effective Version/WorkUnit, explanations and human actions. UX has no independent Task status engine. **Version Driver** (#145) returns a *set* of parallel lawful next-action suggestions with `AUTO_ELIGIBLE / NEEDS_USER_DECISION / NEEDS_HUMAN_HOST / WAIT_EXTERNAL / BLOCKED / UNKNOWN`, source+as-of, rights and required capabilities. **Acceptance ledger** (#146) follows each Product user outcome and must-retain invariant through implemented/reviewed/**actually executed installed behavior**/release evidence; UNKNOWN and REGRESSED stay visible.

## 4. Proposed semantic contract sketches (not frozen DTOs)

```ts
type SourceQualified = {
  githubUrl: string; repo: string; sha?: string;
  adsPin?: string; observedAt: string;
  evidence: "OBSERVED"|"DECLARED"|"INFERRED"|"UNKNOWN"|"STALE"|"NOT_RUN";
};
type NextAction = {
  subject: SourceQualified; requiredRole: string;
  requiredCapabilities: string[]; prerequisites: SourceQualified[];
  authorizationRequired: string;
  readiness: "AUTO_ELIGIBLE"|"NEEDS_USER_DECISION"|"NEEDS_HUMAN_HOST"|
             "WAIT_EXTERNAL"|"BLOCKED"|"UNKNOWN";
  evidenceRefs: SourceQualified[]; rationale: string;
  // Never a native Claim, approval or Effect permit.
};
type AcceptanceItem = {
  productRequirement: SourceQualified;
  expectedUserBehavior: string;
  previousVersionMustRetain: SourceQualified[];
  actualSelectedInstalledConsumerProof: SourceQualified[];
  evidenceCompleteness: "COMPLETE"|"PARTIAL"|"UNKNOWN";
};
type PracticeCandidate = {
  scope: string; applicability: string[]; evidence: SourceQualified[];
  counterexamples: SourceQualified[]; risk: string;
  status: "DRAFT"|"EVALUATED"|"REVIEWED"|"APPROVED_FOR_NEW_BUILD"|"REJECTED";
};
```

Read-side DTOs must not masquerade as normative GitHub schemas, compile-time DomainHarness package manifests or authorized runtime Intent contracts. Product/L2 must map each field to publicly qualified sources and output validation.

## 5. Recommended physical integration path / risk controls

`User intent → Tally Product B → Tally Plan B → ADS-qualified native GitHub facts → selected K-admitted C Agent/Host/GitHub Effect → Tally Assurance B → ADS Release → Tally Experience B → Next Version`.

This is a **logical product journey**, not an all-serial Task DAG. Architect independent file/ownership lanes and meaningful source/verification joins. Reuse actual selected S Rule→Exact semantic reuse→applicable promoted subflow→bounded LLM fallback as historically specified in [DomainHarness #1398](https://github.com/kaicreator-mm/domain-harness/issues/1398); real v0.10.3 support needs exact ABI qualification. Optional [Guided Journey #1395](https://github.com/kaicreator-mm/domain-harness/issues/1395) should use versioned, preauthored step templates first; unknown paths may invoke bounded LLM suggestions, never granting Tool authority.

Avoid generic BPM DSL, raw task-meta connector, duplicate Github reader, K replacement, mandatory runx/SSH, an unrelated graph store, endless per-leaf Full/Hidden, auto-approval or automatic production Package hot-swap.

## 6. Product acceptance in real environments

- **Slice A**: Real Project/Requirement/Plan workspace, proper source/currentness and decisions, no raw JSON; supports no-model/offline read/draft.
- **Slice B**: One genuine frozen Task with accepted GitHub CAS Claim → selected B/K/C → physical Local Agent/Host → PR/Review/CI; lost ACK and missing task-meta negative.
- **Slice C**: Two independent work lanes, one independent Reviewer finding and repair/re-review, selected external-installed consumer functionality with actual evidence, no fabricated Release.
- **Slice D**: Qualified RQ, actual authorized merge/Tag/delivered artifact, subsequent version, inherited invariant check and evidence-based next plan.
- **Historical Golden**: #530, #586, #1013, #1100, #1264, #99, #1385, #1411;  retrospective ≠ as-of prediction. Shared under #143, not another event database.

**Gate:** Reconcile with currently effective Product [PR #77](https://github.com/kaicreator-mm/tally/pull/77), L2 [#65](https://github.com/kaicreator-mm/tally/issues/65)/[#82](https://github.com/kaicreator-mm/tally/pull/82), current source branch and ADS pin. No code/authority/merge claim follows from this draft. Review via #91 and independent Product/L2 before Task DAG admission.
