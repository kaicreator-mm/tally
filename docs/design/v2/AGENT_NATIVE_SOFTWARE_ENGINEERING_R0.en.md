# Tally v2 — Agent-Native Software Engineering Domain App (R0, Agent Edition)

> **2026-10-28 controlling product clarification: Tally constrains vibe coding with software-engineering practice; actual software engineering work is performed by LLM-based Agents by default, with humans involved ONLY when needed.** This is Product/L2 research and acceptance input, not a frozen ADS normative rule, accepted Claim, proven DomainHarness ABI, or authorized implementation. [Product #63](https://github.com/kaicreator-mm/tally/issues/63), [Agent #64](https://github.com/kaicreator-mm/tally/issues/64), [B #91](https://github.com/kaicreator-mm/tally/issues/91), [UX #92](https://github.com/kaicreator-mm/tally/issues/92), [Version Driver #145](https://github.com/kaicreator-mm/tally/issues/145). Chinese detailed counterpart: `AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.zh-CN.md`.

## 1. Fundamental product inversion

**Agent-first execution + engineering constraints + exception-based human intervention.** The default operator is a qualified LLM Agent, not a human Project Controller. Users define software goals, material product decisions, budget/risk/permission boundaries, and required external responsibilities. Agents do the normal planning, architecture, decomposition, implementation, test generation, debugging, independent review, bounded repair, integration validation, Release preparation, and next-version planning. Tally continuously coordinates lawfully eligible work; users do **not** have to tell it "continue" after each stage, copy prompts into each CLI, create native Issues/Claims by hand, or relay Review findings.

**Software engineering is the control and validation system**, not a mandatory human-gated waterfall. Use DDD, Requirements Traceability, Architecture Decision, Task DAG, Configuration Management, Verification & Validation, Quality/Release Management, Lean/Risk and evidence-qualified software process improvement as guardrails around autonomous Agent activity. Apply proportional process: a trivial safe change should not spawn a 5-agent committee or per-step approvals; risky multi-version shared interfaces require current-source independent Review, selected real consumer proof and qualified release gates.

The normal flow is `User Goal → Agent Product/Planning → applicable ADS review/freeze → legally admitted Task DAG/native Claim → Agent Build/Tests → Independent Agent Review/Repair → Integration/Host/User Acceptance → qualified release action → next version`, with independent lanes wherever write-set/source constraints permit. Neither the product flow nor this document itself grants Claim, merge, review, K Effect or Release permission.

## 2. Agent roles are logical capability assignments, not one Agent process per stage

| Role | Preferred executor and outputs | Non-negotiable boundary |
|---|---|---|
| Product Analyst/Planner | LLM Agent: requirements, intended user outcome, alternatives, PRD/evidence candidate | Material ambiguous product choices / applicable approvals must be resolved lawfully |
| Architect/DAG Planner | LLM Agent: L2 decision options, scaffold, subrequirements, genuine dependencies, disjoint write sets, test & Host requirements | Draft DAG never equals GitHub native admitted dependency/Claim |
| Builder | Actual supported Codex, Claude Code, ZCode or other Agent/CLI with permitted source/Host path; changes, commits, PR, local tests | Real accepted Claim, exact source/currentness and write set; cannot self-certify independent Review |
| Reviewer | Separately qualified Agent context with exact PR HEAD and required source evidence; P0/P1/P2 and failure repro | Independence is context, evidence, writer separation and ADS-specific requirements, not "a human by default" or mere name change |
| Verifier | Agent+real CI/Host/installed consumer; true positive and negative executable oracles | Model prose PASS, import/init and mock test cannot substitute selected physical execution |
| Integrator/Release Coordinator | Agent for integration, RQ proposal, evidence reconciliation and release preparation | Actual merge/tag/deploy only with required external GitHub/K and human authority where applicable |

A low-risk Task can reuse one appropriate Agent for compatible Planner/Builder/test work. Spin up distinct Reviewer/Validator contexts only when independence or genuine verification requires them. Different logical Agents can share a transport GitHub login; that does NOT turn them into the same actor or prove reviewer independence. Do not automatically create roles with zero value.

Configure capabilities at runtime using [#64](https://github.com/kaicreator-mm/tally/issues/64) registry: logical agent id, provider/model and actual supported adapter revision, execution Host, active session, capability profile, role independence, task risk/complexity, context needs, concurrency/cost limits, and actual observed success/coverage. Do not assume arbitrary Agents support resumable sessions, git-native Claim writes, Windows/Android devices, long-running operations, model hooks or all CLI features. Data-poor ranking must return UNKNOWN / NO_ELIGIBLE_AGENT instead of invented performance.

## 3. Delegated autonomy is strictly narrower than actual authority

This proposed **product preference** is not a substitute for ADS or DomainHarness K approval:

- **A — Read/propose freely within access:** inspect repo, analyze, draft PRD/DAG, propose tests, report next actions. No source mutation from this tier.
- **B — Execute within legitimately delegated envelope:** source and GitHub actions within accepted Claim, exact SHA/currentness, bounded branch/write set, eligible role, budget/time/Tool/Host grants; DomainHarness B proposes and selected K admits each effect, C performs it, original native GitHub rights still apply. CI/Reviewer may be orchestrated automatically under approved gates.
- **C — Material authority boundary:** changed frozen Product/L2/scope, exceptions to required gates, unbounded cost/security risk, credentials/privileged machine access, irreversible external action, production deployment or a release that applicable policy delegates to a real human. Request actual authorized approval and verify readback; no agent can self-approve.

**Only four types of human intervention** should appear: (1) unresolved material goal/scope/risk/value choices; (2) inaccessible physical Host, local machine, accounts/credentials or legally external action; (3) out-of-scope rights, budget, high-risk irreversible effect or source/Claim conflict that bounded Agent procedures cannot resolve; (4) actual governing policy requiring human acceptance/sign-off. Valid errors/failed CI/ordinary source review are generally **Agent repair work**, not reasons to ask the user to relay a prompt. If not human-required, do not put it in a user approval inbox.

The user-facing `HumanInterventionRequest` must justify **why an Agent/CI cannot do it**, supply concise options and real provenance/risk/expected host evidence, and explain which workfront can resume on qualified approval. Neither a "user clicked Approve" mock nor a K Tool ACK alone is terminal acceptance. Never synthesize approval from user silence.

## 4. Key Engineering Business B capabilities (proposed semantic DTOs)

- `AgentWorkPlan`: Task exact source, accepted native Claim, output and L3 acceptance, allowed write set, risk, required Review/CI/real Host, estimated budget, Agent/model/host capability fit, independent reviewer needs.
- `RoleAndCapabilityMatch`: supported provider/model/adapter/CLI, host, currentness, context requirements, real capability evidence, budget/concurrency/cost; `DECLARED` distinct from `OBSERVED`.
- `DelegatedAuthorityEnvelope`: product allowance with source/rights/time/budget bounds; not a native GitHub Claim, issued K permit or independent approval.
- `TaskOrchestrationDecision`: `START_ELIGIBLE/WAIT_GATE/WAIT_AGENT/HANDOFF/REVIEW/REPAIR/RECONCILE/HUMAN_REQUIRED/ABSTAIN`, all with source, owner and reason.
- `HumanInterventionRequest`: decision rationale, alternatives, external action/required evidence, exact restart and readback, only if truly needed.
- `ReviewerSeparationProof`: reviewer context/session and model/provider details, independence from Builder write/repair role, exact HEAD and actual review provenance.

**No second scheduler/Kernel/Claim store.** Engineering B via [#91](https://github.com/kaicreator-mm/tally/issues/91) + [#145](https://github.com/kaicreator-mm/tally/issues/145) decides lawful next actions, selected S provides Rule/Context/Decision/optional Guided Journey, DomainHarness **single selected K** owns state/admission/CAS/journal/effect/recovery/UNKNOWN, C connects real local Agent/CLI, GitHub, CI, Host/Model/optional runx. UX [#92](https://github.com/kaicreator-mm/tally/issues/92) shows current Agent work and exceptions rather than forcing a big static wizard. task-meta is private observer with bounded ADS-published GitHub report; outage cannot block engineering.

## 5. Actual DomainHarness evidence → Agent-native Golden

| Source | Expected business response | Forbidden false behavior |
|---|---|---|
| [DH #530](https://github.com/kaicreator-mm/domain-harness/issues/530) false serialization | Planner proposes disjoint parallel work, leave true dependency/compatibility join | Silently remove native blocked_by |
| [DH #586](https://github.com/kaicreator-mm/domain-harness/issues/586) verification Campaign | Verifier reuses checkpoints and effective evidence | One Feature Issue/mandatory human per checkpoint |
| [DH #1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) overlapping Claim/PR | Agent Coordinator diagnoses writer collision, stops duplicate source writes and provides lawful handoff | Two Builder Agents concurrently modify same protected write set |
| [DH #1119](https://github.com/kaicreator-mm/domain-harness/issues/1119) clean consumer | Verifier Agent requests true external installed SDK proof | Builder internal smoke = real user acceptance |
| [DH #1264](https://github.com/kaicreator-mm/domain-harness/issues/1264) hooks imported not invoked | Independent Agent Reviewer requires actual selected-path call | Trust compile/import claims |
| [DH #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385) reproducible Host defect | Diagnoser uses real 4/15 replays and bounded failure/root classification | Single success erases repeated failure evidence |
| [DH #1411](https://github.com/kaicreator-mm/domain-harness/issues/1411) unavailable GitHub dependency mutation | Human/local Host handoff only for missing physical tool action; Agent resumes after native readback | Claim Task DAG admitted based only on prose planning graph |

**Lifecycle acceptance:** User specifies a project goal and bounded delegation once. With properly authorized existing gates, agents propose Product/L2/DAG, do two parallel real Tasks, one independent Agent Review finds a defect, lawful scoped Builder repair and independent re-review follow, CI/real Host/installed consumer succeeds, final Release approval when required and actual authorized merge/tag occur, and next version begins. The user does not click/relay every stage. Negative scenarios: fake independent reviewer context; stale HEAD; split Agent identities under one GitHub account; parallel writer collision; unsupported native dependency API; unreconciled external ACK; missing Host; risk/budget scope escalation; remote credential unavailable; task-meta outage should not block Task. A false Human Required prompt for ordinary eligible CI/Agent work is itself a failing UX acceptance probe.

Measure delivered user software outcomes, source-current task correctness, actual review/CI/Host and qualified release, genuine autonomous continuity, count/causes of **necessary vs unnecessary human interventions**, actual measured Agent/model/tool cost/time and coverage, duplicate source work, repair rework. Fewer humans is NOT a substitute for correctness and security; no agent-hour inference from Issue created/closed timestamps.

## 6. Three-project learning consequences

**task-meta** should observe real Agent identity+Provider/Model/Role/Attempt/Tool/Host/verifier, verified handoffs and only measurable Human waiting/intervention causes, bounded source-qualified publication and measurement coverage; no scheduler or approval semantics. **ADS** should specify normative Agent role, independence, native Claim and evidence semantics and the precise cases that genuinely require a qualified human, avoiding artificial human gates. **Tally** should learn which qualified Agent/Model/Host arrangement and amount of parallelism produce verified user software results, and recommend valid task routing, reusable Context/Repair/Validation procedures or exception handoffs; promotion requires existing casebook/holdout/independent review/new sealed BUILD.

**Status:** `DESIGN_R0`, `HUMAN_EXCEPTION_BASED=PROPOSED`, `AGENT_NATIVE_BUSINESS_CONTRACT=PROPOSED`, `NO_NEW_KERNEL`, `NO_AUTOMATIC_APPROVAL`, `SOURCE_IMPLEMENTATION_NOT_AUTHORIZED`.
