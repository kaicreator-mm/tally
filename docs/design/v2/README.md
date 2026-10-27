# Tally v2 — Agent-Native Software Engineering Design Hub (R0)

**This folder is the documented **research/design integration entry point**, NOT a second or silently frozen Product/L2 source of truth.** See [Product #63](https://github.com/kaicreator-mm/tally/issues/63), [Product R0 PR #77](https://github.com/kaicreator-mm/tally/pull/77), [L2 PR #82](https://github.com/kaicreator-mm/tally/pull/82), [ADS exact-pin PR #85](https://github.com/kaicreator-mm/tally/pull/85), and [v2 Workfront #86](https://github.com/kaicreator-mm/tally/issues/86). **No Freeze, Task Claim, original Review PASS, or merged-release authorization is conferred by these documents.**

## 1. What Tally is / 产品定位

> **Tally constrains vibe coding with software engineering. Actual development is normally performed by qualified LLM coding/reviewer/verification Agents. Humans intervene only when a goal decision, authority/credential/physical Host, budget/risk boundary or actual release sign-off requires them.**
>
> **Tally 以软件工程约束 Vibe Coding，由各种 LLM Agent 默认执行真实开发、独立审核、修复和验证，只有真正需要时人才介入。** 用户在 Tally 内从软件想法走到真实版本发布和下一轮迭代。工程 Gate 不等于人工 Gate；Agent 自主并不豁免真实 GitHub/ADS/DomainHarness K 权限。

## 2. Read in this order / 推荐阅读顺序

| Purpose | 中文 | English / Agent |
|---|---|---|
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
