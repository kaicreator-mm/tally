# Tally v2 — Agent-Native Software Engineering Design Hub (R0)

**This folder is the documented **research/design integration entry point**, NOT a second or silently frozen Product/L2 source of truth.** See [Product #63](https://github.com/kaicreator-mm/tally/issues/63), [Product R0 PR #77](https://github.com/kaicreator-mm/tally/pull/77), [L2 PR #82](https://github.com/kaicreator-mm/tally/pull/82), [ADS exact-pin PR #85](https://github.com/kaicreator-mm/tally/pull/85), and [v2 Workfront #86](https://github.com/kaicreator-mm/tally/issues/86). **No Freeze, Task Claim, original Review PASS, or merged-release authorization is conferred by these documents.**

## 1. What Tally is / 产品定位

> **Tally constrains vibe coding with software engineering. Actual development is normally performed by qualified LLM coding/reviewer/verification Agents. Humans intervene only when a goal decision, authority/credential/physical Host, budget/risk boundary or actual release sign-off requires them.**
>
> **Tally 以软件工程约束 Vibe Coding，由各种 LLM Agent 默认执行真实开发、独立审核、修复和验证，只有真正需要时人才介入。** 用户在 Tally 内从软件想法走到真实版本发布和下一轮迭代。工程 Gate 不等于人工 Gate；Agent 自主并不豁免真实 GitHub/ADS/DomainHarness K 权限。

## 2. Read in this order / 推荐阅读顺序

**ALM ownership clarification (2026-10-28):** Tally is primarily the agent-facing ZenTao/Jira-style software-lifecycle **task management, engineering evidence, and learning application**. Qualified external Agents perform actual Product/design/coding/test/review/release-preparation and communicate by the adopted ADS/native GitHub protocol, with or without Tally online. Tally re-reads, manages, coordinates and recommends; it does not implement those Agents or an alternate message broker/runner. See the new **Agent-native ALM Product R0** pair below. This is a proposed Product correction, not a substitute for formal PR #77/PR #82 Review or Freeze.

| Purpose | 中文 | English / Agent |
|---|---|---|
| **Agent-native ALM: requirements → development → QA → release → maintenance + Task learning** | [面向 Agent 的全生命周期软件工程管理](./AGENT_NATIVE_ALM_PRODUCT_R0.zh-CN.md) | [Agent-native ALM Product R0](./AGENT_NATIVE_ALM_PRODUCT_R0.en.md) |
| **Current v2 branch/PR reality and implementable delta — START HERE** | [重构整合与交付路线](./TALLY_V2_REFACTOR_INTEGRATION_R0.zh-CN.md) | [Source crosswalk / implementation handoff](./TALLY_V2_REFACTOR_INTEGRATION_R0.en.md) |
| **LLM Agents default; Human on exceptions** | [Agent 原生软件工程模型](./AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.zh-CN.md) | [Agent-native operating model](./AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.en.md) |
| **Engineering Business B/S/K/C design from real software engineering** | [Domain Package 蓝图](./TALLY_ENGINEERING_DOMAIN_PACKAGE_BLUEPRINT_R0.zh-CN.md) | [Domain Package blueprint](./TALLY_ENGINEERING_DOMAIN_PACKAGE_BLUEPRINT_R0.en.md) |
| **task-meta→ADS→Tally evidence-governed evolution** | [三项目进化协议](./ENGINEERING_EVOLUTION_PROTOCOL_R0.zh-CN.md) | [Engineering evolution protocol](./ENGINEERING_EVOLUTION_PROTOCOL_R0.en.md) |
| **Source-linked retrospective practice patterns** | [12 条工程案例种子](./ENGINEERING_PRACTICE_SEED_R0.json) | Same JSON; *retrospective only*, not honest as-of prediction or normative event |

## 3. Exact owners / 正确负责方

- [#76 Product](https://github.com/kaicreator-mm/tally/issues/76) / PR #77 is the **only formal Product revision/Freeze owner**. Review Product D01–D09 in the integration document; **especially existing V63 `max 1 active Task Attempt` vs safe-parallel Agent-first v2 intent**, plus current legacy parity P1. Update lawfully and independently review **that** actual Product PR HEAD; never treat #147 as its substitute.
- [#78 L2](https://github.com/kaicreator-mm/tally/issues/78) / PR #82 is formal L2 owner; map approved Product into actual public DomainHarness ABI/Host. Logical B bounded contexts are NOT mandatory separate deployable packages.
- [#91 Engineering B](https://github.com/kaicreator-mm/tally/issues/91) designs business contracts; [#64 Agent](https://github.com/kaicreator-mm/tally/issues/64) owns logical/provider/role mapping; [#125 Claim](https://github.com/kaicreator-mm/tally/issues/125) owns real native source/Claim→single K admission; [#92 UI](https://github.com/kaicreator-mm/tally/issues/92) owns complete J01–J08 real UX; [#145 Journey](https://github.com/kaicreator-mm/tally/issues/145) owns legal autonomous next-actions; [#146 Acceptance](https://github.com/kaicreator-mm/tally/issues/146) owns actual user functionality and inherited behavior proof.
- [#139 evidence graph](https://github.com/kaicreator-mm/tally/issues/139), [#140 diagnosis](https://github.com/kaicreator-mm/tally/issues/140), [#141 next version](https://github.com/kaicreator-mm/tally/issues/141), [#142 work-unit](https://github.com/kaicreator-mm/tally/issues/142), [#143 casebook](https://github.com/kaicreator-mm/tally/issues/143), [#144 upstream feedback](https://github.com/kaicreator-mm/tally/issues/144), and [#57/#58/#60 experience](https://github.com/kaicreator-mm/tally/issues/60) retain their existing responsibilities; do not invent more engines or duplicative Issue types.
- [task-meta #27](https://github.com/kaicreator-mm/task-meta/issues/27) is **measurement/coverage publisher**; [ADS #1347](https://github.com/kaicreator-mm/ai-development-standard/issues/1347) is **normative rule evolution**; DomainHarness selected K is **execution-state/effect authority**. Tally does not read private task-meta API/Trace.

## 4. Current state / 设计与交付状态

```ini
DOCUMENTS=AUTHOR_CANDIDATE_R0
SOURCE_DESIGN_READBACK=YES
PRODUCT_PR77_FORMAL_FREEZE=NO
L2_PR82_FORMAL_FREEZE=NO
ADS_PR85_EFFECTIVE_ADOPTION=NO
NEW_AGENT_FIRST_SCOPE_ADMITTED_TASK_DAG=NO
CURRENT_V2_EXISTING_SOURCE=REUSE_WHERE_ACTUALLY_ACCEPTED
TRUE_MULTI_AGENT_END_TO_END=NOT_RUN
REAL_USER_RELEASE_JOURNEY=NOT_RUN
RELEASE_PASS=NO
```

### Review and implementation handoff / 审核与实施建议

1. **Independent Product Reviewer:** decide D01–D09; resolve current PR #77 P1 parity and V63 conflict. Preserve existing 17 P0 / 67 vectors as the source basis, add only genuine new user-outcome tests.
2. **Independent L2 Reviewer:** review current PR #82 plus actual DomainHarness public ABI and appropriate agent role/Host isolation; do not overgeneralize to multiple physical B Packages.
3. **Controller:** reuse version/v2 already merged GitHub/Host/source packages and incoming #129/#131/#133/#135; only source-authorize missing independent implementation concerns with real native dependencies and disjoint write sets.
4. **Agent-native Product E2E:** prove one true Agent Task first, then two safe-parallel legitimate Builder attempts + separate Reviewer/Repair, selected installed consumer behavior, actual authorized Release/next version. Human only on warranted exceptions. **No pretend Async ChatGPT job or automatic self-authorization.**

## 5. Conclusion-to-Issue coverage register / 本轮对话结论覆盖核对（2026-10-28）

**Important:** `ISSUE_IN_V2_PRODUCT_INTAKE` is not the same as `APPROVED_FOR_V2_IMPLEMENTATION`. This register tracks **all outcomes raised in the current DomainHarness-history → software-engineering Domain Package → task-meta/ADS/Tally evolution → Agent-first full-lifecycle discussion** without silently enlarging the already admitted Task DAG. **No GitHub milestone is attached to these sampled Issues**; their `[v2]` titles / parent and explicit issue bodies establish *candidate intent*, not a formal Milestone or Product Freeze.

| Topic / 结论 | Tracked v2 owner(s) | Current disposition / what remains |
|---|---|---|
| Software-engineering-constrained vibe coding; LLM Agents implement/review/verify by default; human exception-only | [Product #63](https://github.com/kaicreator-mm/tally/issues/63), [Agent #64](https://github.com/kaicreator-mm/tally/issues/64), [Engineering B #91](https://github.com/kaicreator-mm/tally/issues/91), [journey #145](https://github.com/kaicreator-mm/tally/issues/145) | **P0 candidate**, in #147 and issue comments; not integrated/frozen in Product PR #77/L2 PR #82 |
| Real end-user full software lifecycle UI, no manual per-step approvals | [W-F UX #92](https://github.com/kaicreator-mm/tally/issues/92), [Product #76](https://github.com/kaicreator-mm/tally/issues/76), [Version Driver #145](https://github.com/kaicreator-mm/tally/issues/145) | **P0 intake**; genuine installed app J01–J08/Agent E2E not verified |
| Max-safe-parallel DAG subrequirements, disjoint Builder Agents, independent review and bounded repair | [DAG/Controller #86](https://github.com/kaicreator-mm/tally/issues/86), [B #91](https://github.com/kaicreator-mm/tally/issues/91), [Agent #64](https://github.com/kaicreator-mm/tally/issues/64) | **P0-vs-P1 Product conflict**: existing V63 allows only one physical executing Task P0. Product #76 must approve final scope change or explicitly label limited MVP |
| Genuine native Task Claim, K-admitted B→C GitHub/Agent effects | [T06 #125](https://github.com/kaicreator-mm/tally/issues/125), [real E2E #62](https://github.com/kaicreator-mm/tally/issues/62) | Real source reader partly merged; Draft #129/#131/#133 and real Claim/Agent consumer qualification pending |
| User-requirement→selected installed behavior/Release and inherited security proof | [Acceptance #146](https://github.com/kaicreator-mm/tally/issues/146), [full UX #92](https://github.com/kaicreator-mm/tally/issues/92) | **P1 intake**, must incorporate minimum v2 release-relevant user acceptance vectors into formal Product; noncritical advanced coverage may follow |
| Current source-qualified WorkUnit/DAG/Claim status vs progress illusion | [#137](https://github.com/kaicreator-mm/tally/issues/137), [Issue≠Task #142](https://github.com/kaicreator-mm/tally/issues/142) | **P1 intake**, reuse T06 read/diagnostic, no second GitHub/K authority |
| True multi-version Product/L2→Task→Review→CI→Release evidence, repair/Agent time and Root Cause | [Evidence #139](https://github.com/kaicreator-mm/tally/issues/139), [Diagnosis #140](https://github.com/kaicreator-mm/tally/issues/140), [Casebook #143](https://github.com/kaicreator-mm/tally/issues/143) | **P1 intake**, 12 cases in #147 are *retrospective design seeds only*, not reconstructed as-of predictive proof |
| History-driven Agent/Task DAG/validation strategy, bounded revisit and model-less fallback | [NextVersion #141](https://github.com/kaicreator-mm/tally/issues/141), [Periodic Learning #57](https://github.com/kaicreator-mm/tally/issues/57), [Task Context #58](https://github.com/kaicreator-mm/tally/issues/58), [Strategy #67](https://github.com/kaicreator-mm/tally/issues/67), [Archive #68](https://github.com/kaicreator-mm/tally/issues/68) | **P1 intake**; first reuse deterministic casebook/context, defer heavyweight scoring/A-B unless approved ([#69](https://github.com/kaicreator-mm/tally/issues/69)) |
| Feedback→rule/procedure/skill/selected Domain Package qualification; downstream SDK defect loop | [Evolution #60](https://github.com/kaicreator-mm/tally/issues/60), [Cross-project #144](https://github.com/kaicreator-mm/tally/issues/144) | **#60 v2 intake**, #144 **P2 intake**; cross-project/auto promotion is not automatically a v2 Release blocker |
| Legacy Go→TS/Node24 old CI profile isolation | [#138](https://github.com/kaicreator-mm/tally/issues/138), [#83](https://github.com/kaicreator-mm/tally/issues/83) | **P1 intake**; qualifies true selected Node24 checks, does not hide mandatory inherited quality baseline |
| task-meta task/agent time/cost/attempt & measurement evolution | [task-meta #27](https://github.com/kaicreator-mm/task-meta/issues/27), [#12](https://github.com/kaicreator-mm/task-meta/issues/12), [#17](https://github.com/kaicreator-mm/task-meta/issues/17) | **Other repository owner**, only approved ADS GitHub published bounded summary enters Tally; not a private connector or Tally source task |
| ADS practice-driven normative evolution / correct Agent Review & escalation | [ADS v4.19 #1347](https://github.com/kaicreator-mm/ai-development-standard/issues/1347), [#1346](https://github.com/kaicreator-mm/ai-development-standard/issues/1346), [Agent normative #1199](https://github.com/kaicreator-mm/ai-development-standard/issues/1199) | **Other repository owner**, explicit version review and adoption required; no automatic upgrade of Tally's effective ADS pin |
| DomainHarness generic Guided Journey & Meta learning SDK when truly reusable | [DomainHarness #1395](https://github.com/kaicreator-mm/domain-harness/issues/1395), [#1398](https://github.com/kaicreator-mm/domain-harness/issues/1398) | **Other repository owner**, do not assume v0.10.3 public released ABI or block v2 on an optional generic package |

### This coverage register does NOT close these concrete gaps

1. **Product** — PR [#77](https://github.com/kaicreator-mm/tally/pull/77) is still Draft/unmerged, unresolved parity P1, V63 single-execution physical cap vs final multi-Agent objective; formally triage and revise core P0 requirements and acceptance under [#76](https://github.com/kaicreator-mm/tally/issues/76).
2. **L2/Version** — PR [#82](https://github.com/kaicreator-mm/tally/pull/82) semantic source reviewed but NOT formally frozen/adopted; ADS [#85](https://github.com/kaicreator-mm/tally/pull/85) project adoption Draft; new #147 source has no Task DAG/Claim/merge authority.
3. **Execution** — full native Claim→physical multi-Agent independent Reviewer/Repair→installed application→authorized RQ/Release **NOT RUN**, and not every P1/P2 intake is automatically a v2 P0 release gate.

**Therefore answer to 'all in v2 refactor?'** Design+Issue inventory **YES, inventoried and owner-linked**; formal frozen v2 Product/L2/native Task DAG/implementation/release **NO**; external task-meta/ADS/DH work remains in **its own repository/version**. The correct next act is an independently reviewed **scope decision and Product/L2 amendment**, not another generic umbrella architecture project or automatic mass promotion of P1/P2 features into v2 P0.
