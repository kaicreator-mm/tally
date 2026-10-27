# Tally v2 — Agent-Native Application Lifecycle Management (ALM) Product R0

> **Status:** Product/L2 intake and design proposal only. Not an adopted Product Freeze, an ADS wire protocol, a real native Task Claim, source-write authority, a reviewed executable Task DAG, or Release qualification. Formal owners: [Product #76](https://github.com/kaicreator-mm/tally/issues/76)/[#63](https://github.com/kaicreator-mm/tally/issues/63), [L2 #78](https://github.com/kaicreator-mm/tally/issues/78), [UX #92](https://github.com/kaicreator-mm/tally/issues/92), [Controller #86](https://github.com/kaicreator-mm/tally/issues/86).
>
> **Product definition:** **Tally is an agent-native ZenTao/Jira-like software Application Lifecycle Management system** for multiple external coding, design, review, test, and release-preparation Agents. Tally manages requirements, work units, dependencies, project progress, user acceptance, and evidence-qualified engineering experience. External Agents perform actual software engineering through ADS and authoritative GitHub records. Do not build a coding/reviewer Agent inside Tally Business, a generic BPM executor, a second Kernel, or a proprietary inter-Agent communication protocol.

## 1. Product domain objects — an Issue is not always an executable Task

| Object | User/Agent-facing management responsibility | Authority and source of truth |
|---|---|---|
| Project / Repository | Goals, constraints, team and Agent roster, budget, Host, permissions | Actual GitHub repository/grants |
| Requirement / Epic / Story | Business need, priority, acceptance criteria, changes | Approved Product revisions and ADS-qualified review/freeze/amendment |
| Version / Iteration | Scope baseline, Release criteria, remaining risk and change management | Qualified version source, native branch, RQ and release/tag |
| Architecture / Design | Technical options, interfaces, constraints, scaffolding | L2 source and independent review/freeze |
| WorkUnit | Source Task, Concern, Review, Repair, Campaign, Verification Checkpoint, Planning, etc. | Native Issue, auditable concern/checkpoint identity; do not count Issues as Features |
| Executable Task / Task DAG | Dependencies, conflict keys/write sets, Agent role and capability fit | Native ADS Issue dependencies, accepted Claims; Markdown plans are candidates |
| Attempt / Handoff | One actual Agent attempt, callback, replay/retry, cross-Agent handoff | ADS-native GitHub Result/Report and actual PR/Commit/CI/Host receipts |
| Test Plan / Case / Defect | Behavioral proof, reproductions, findings, repairs and regression evidence | Actual independent Review, CI/Host, native evidence and validation coverage |
| Acceptance / Quality Gate | Proof that a promised user behavior and inherited security invariant are preserved | Actual selected/installed public behavior, required checks and review |
| Release / Maintenance | Qualification, merge, tag, artifact, post-release issues, next version | Authoritative source-qualified GitHub/ADS facts |
| Practice / Learning Candidate | Reusable advice, limitations, counterexamples, validation and feedback | ADS Task Learning/feedback plus independently supportable GitHub evidence |

Business DTOs, UI state and SQLite projections are **derived/reconstructable**. They never become a second GitHub Claim, PR, Review, CI, Product Freeze, or Release source of truth. These names are **not** new normative ADS event types.

## 2. Full lifecycle management, with parallel lanes and lawful backward transitions

| Stage | What Tally manages | Actual producer / work | Required source-qualified gate |
|---|---|---|---|
| **01 Requirements** | Greenfield/import, backlog, Epic/Story, priorities, scope/acceptance and change requests | Product/Research Agents propose, user arbitrates real high-value choices | Reviewed/frozen Product or lawful amendment |
| **02 Architecture** | Alternatives, API/design, architecture risks, scaffolding | Architect/Planner Agents produce proposals and artifacts | Qualified L2 review/freeze |
| **03 Planning** | Version/Sprint, independent concerns, Task DAG, roles, Host/budget/write-set constraints | Planner Agents plan maximum safe independent lanes | Actual ADS/native dependencies and Task contracts; no claim inferred from a plan |
| **04 Development** | Eligibility, dispatch, accepted Claim, attempt, isolated workspace, PR, retries/conflicts | External Builder Agents using their selected authorized Host/CLI/Provider | Native Claim/actual K effect and PR evidence reconciled separately |
| **05 Review & Defects** | Independent findings, defect identity, repair source, re-review/iteration | Qualified independent Reviewer and Repair Agents | Exact SHA, P0/P1 treatment, retest/CI when required |
| **06 Testing & Acceptance** | Test cases/plans, regression, CI/Host, user capability acceptance | Validator Agents, CI/physical Hosts, optional human acceptance | Real selected installed behavior and applicable proof, NOT_RUN/UNKNOWN retained |
| **07 Release** | Candidate, risk/readiness, RQ, authorized merge/tag/artifacts | Release Agents prepare; authorized actors perform governed effects | Exact commit/tree, independent RQ and true merge/tag/artifact readback |
| **08 Maintenance** | Bugs, feature change, escaped regressions, next version, technical debt | Support/Planner/Builder/Review Agents handle new legal work | New authorized Product/L2/Task, preserved immutable prior releases |

The eight stages are **not a one-way wizard**. Independent subrequirements and different versions can progress in parallel if real dependencies permit. Failed Review loops into bounded Repair; post-release defects open new work; Product scope changes require an amendment. Pending upstream Release joins must not create false edges blocking unrelated source work. Tally cannot itself clear an authoritative blocked_by edge or invent a CAS Claim.

## 3. External Agents + ADS + Tally: ownership and data flow

1. **Agents actually work**: Planner, Architect, Builder, Reviewer, Repairer, Validator, and Release-preparation Agents may be different tools, sessions, Hosts and models. Assign by evidenced capability, true independence, role, available Host, security policy, cost and risk; do not bind software-engineering meaning to one model vendor.
2. **ADS is the interop contract**: Work Item/Task Contract, native Claim, Agent↔Agent Handoff, Task Report, independent Review, Task Learning and Release evidence follow the **project's adopted exact ADS revision** and actual GitHub-native carriers. Agents may hand off directly without Tally being online. No Tally-specific alternate Agent message broker or private ADS fork.
3. **Tally maintains tasks**: organize and advise requirements, dependencies, work-unit identity, eligibility, assignment/handoff, evidence, blockers, quality and next actions. If Tally itself initiates effects, actual authorized B→one selected DomainHarness K→C/Host applies. If an external Agent posts ADS/GitHub output directly, Tally re-reads and reconciles the original facts; a local status is not native authority.
4. **task-meta observes but does not authorize**: execution attempts, elapsed/active time, tokens, model/route, tool usage, cost, source coverage are available to Tally only as bounded ADS-published GitHub summaries and optional Learning. No direct task-meta private DB/API/Trace/manifest access; missing observation is UNKNOWN/NOT_COMPARABLE, not zero and not a Task blocker.
5. **Human intervention is exceptional**: material product choice, non-delegable permission/credential, physical Host, budget/risk threshold and genuine required Release signoff. Normal independent Agent Review/Repair/CI should not require a human to copy prompts every step. Show the reason, minimum action and required evidence/readback.
6. **Authority levels must remain separate**: `DRAFT`, `DECLARED`, `OBSERVED`, `SOURCE_QUALIFIED`, `ACCEPTED_CLAIM`, `K_ADMITTED`, and `PHYSICAL_VERIFIED` are illustrative interpretive dimensions, **not new normative ADS statuses**. Missing pages, stale SHA, revoked evidence or untrusted source ⇒ UNKNOWN/STALE, never PASS.

## 4. Agent-native ZenTao/Jira experience

- **Workspace**: Project/Version overview, real next actions, running independent workfronts, blockers, relevant Human Intervention Inbox and evidence/coverage.
- **Requirements**: Backlog, Epic/Story, product revisions, priorities, scope changes, acceptance, traceable design and decision history.
- **Planning**: Version and Sprint, DAG graph, safe independent lanes, roles/Host, dependencies and conflicting write sets.
- **Task board**: WorkUnit versus Issue, Agent and Attempt, Claim, exact SHA, PR/Review/CI, reconcilable states and blocked explanations.
- **Quality**: Test Plan/Case, defects, findings, regression, real installed acceptance, evidence gaps and applicable Release gates.
- **Versions**: Candidate and version contents, RQ, change log, authorization, release artifacts, regressions and next-version intake.
- **Engineering practice**: Learning cases, source-grounded lessons, context packs, qualified procedures/rules/Skills, adoption/rejection and later feedback.
- **Project policy**: Repository, adopted ADS revision, Agent capability/roles, Host/Provider, budget, permissions, audit boundaries.

Build the smallest actionable workspace using existing UI and repository libraries, not a bespoke Jira database, generic enterprise BPM engine, or agent hosting environment.

## 5. Learning that improves verified software engineering outcomes

```text
Requirements/acceptance → Task DAG → Agent Claim + actual Attempt → PR/Review/Tests/Release
                                        │
                     ADS GitHub report/learning + original native evidence
                                        │
               Tally source reconciliation / root causes / counterexamples
                                        │
             Evidence-qualified Practice / Context / Rule / Skill candidate
                                        │
                  Independent qualification / bounded applicability
                                        │
                Next legal Task receives relevant context or advice
                                        │
                     Real follow-up adoption/rejection/outcome
                                        └─────────────────────────────→ re-evaluate
```

A Learning record requires provenance to source Task/Attempt/commit/Review/CI, observed behavior and failure class, applicability, known counterexamples, coverage/confidence, recommendation, actual use/refusal and later **observed** outcome. No access to raw private chain of thought. Do not infer active Agent hours from Issue lead time or claim unmeasured quality/cost gains. Abstain when evidence is insufficient.

Experience may improve requirement decomposition, DAG parallelism, Agent role/model/Host fit, testing strategy, Review/Repair risk, and next-version planning. Normative ADS rules and executable DomainHarness Packages are promoted only by their legitimate owners through their qualified workflow; no automatic self-authorized Package hot-swap.

Reuse [#57 Task learning](https://github.com/kaicreator-mm/tally/issues/57), [#58 Context Pack](https://github.com/kaicreator-mm/tally/issues/58), [#60 Package evolution](https://github.com/kaicreator-mm/tally/issues/60), [#139 Lineage](https://github.com/kaicreator-mm/tally/issues/139), [#140 Diagnosis](https://github.com/kaicreator-mm/tally/issues/140), [#141 Next Version](https://github.com/kaicreator-mm/tally/issues/141), and [#143 Casebook](https://github.com/kaicreator-mm/tally/issues/143). Do not construct a second telemetry collector or unrelated Learning Engine.

## 6. Formal Product amendment needed and real acceptance examples

This document **does not edit frozen/current PR #77 P0/V01–V67**. Product Owner #76 should source-crosswalk and accept/reject only necessary user-outcome deltas:

| Journey | Positive source-backed acceptance | Forbidden false success |
|---|---|---|
| **ALM-01 Demand to work** | User manages real project in Tally; external Agent creates a reviewable Product/L2 and actual lawfully admitted native Task Issues/dependencies | A UI mock or Markdown DAG is an approved Task/Freeze |
| **ALM-02 Real Agent result** | An external Builder accepts genuine authorized Task and submits PR/ADS Result; Tally re-reads authentic Claim, Review and CI and reconstructs after restart | Schema-valid comment/report is accepted CAS Claim or K-admitted effect |
| **ALM-03 Defect lifecycle** | Independent Reviewer identifies real fault; bounded Repair WorkUnit, actual code fix, re-review/regression trace | Closed Issue / merged PR alone proves user acceptance |
| **ALM-04 Safe parallelism** | Two legally independent Agent Builder work units are actually active with isolated write sets; conflicts rejected, Review Join qualified; one Task remains first vertical slice | A static parallel-ready graph proves two executions. Existing Product V63 max1 conflicts with full-v2 target and requires formal change |
| **ALM-05 QA + Release** | Requirements→Test Case→CI/Host→selected installed user acceptance→independent RQ→authorized merge/tag readback | Source merged or Node unit PASS is a Release |
| **ALM-06 Task learning** | Qualified historical result produces context for the next Agent Task; adoption/refusal and true later outcome observable | Model self-evaluation alone promotes a Rule; private task-meta trace read directly |
| **ALM-07 Maintenance** | Defect/feature after release links previous acceptance and becomes next lawful Version; immutable historical evidence remains | Overwriting old Release or silently changing frozen DAG |

Implementation milestones: **M1 genuine requirements/planning sources → M2 real external Agent/Claim/Review chain → M3 QA/Release and independent parallelism → M4 evidence-driven next Task/Version**. Use disjoint source lanes and small integration joins, not a forced fully serial analysis→planning→coding checklist.

**Outstanding gates**: current Product #77 historical parity P1/Formal Freeze; L2 #82; ADS #85 local adoption; real #125 accepted CAS Claim; independent Agent/Review/Quality/Release; upstream v0.10.3 contract-first work only until a real tagged packed ABI can be qualified. PR #151 currently has independent R2 **BLOCK P1×2** despite scoped Node24 10/10 PASS.

**Release-level product test:** A user can manage an entire software lifecycle in Tally, while eligible external Agents produce real requirements/code/review/test/release-prep artifacts under ADS/GitHub authority, and later Tasks demonstrably receive evidence-qualified learning from earlier work.
