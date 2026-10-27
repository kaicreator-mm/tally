# Tally v2：真实软件工程实践驱动的 Domain Package 设计基线（R0 / 中文）

> **性质：PRODUCT/L2 研究候选，非已冻结契约、非 SDK 已实现证明、非 Task Claim 或 Release 授权。** 2026-10-28。目标是以 Tally 作为普通用户完成软件项目从目标到多版本发布的统一 Domain App。所有设计须经当前有效 ADS Product/L2、确切 DomainHarness 已发布 ABI、独立 Review 后才进入合法开发。
>
> 关联：[Product #63](https://github.com/kaicreator-mm/tally/issues/63)、[B Contracts #91](https://github.com/kaicreator-mm/tally/issues/91)、[完整用户旅程 #92](https://github.com/kaicreator-mm/tally/issues/92)、[Version Driver #145](https://github.com/kaicreator-mm/tally/issues/145)、[验收账本 #146](https://github.com/kaicreator-mm/tally/issues/146)、[Casebook #143](https://github.com/kaicreator-mm/tally/issues/143)。
> 英文 Agent 设计副本见 `TALLY_ENGINEERING_DOMAIN_PACKAGE_BLUEPRINT_R0.en.md`；三系统进化与变更归属见 `ENGINEERING_EVOLUTION_PROTOCOL_R0.zh-CN.md`。

> **2026-10-28 P0 产品定位更新（优先于本文任何可能被解释为人驱动步骤的表述）：Tally 是由 LLM Agent 默认执行软件开发、以软件工程规范约束 Vibe Coding、仅必要时人类介入的 Domain App。** 人不是每阶段批准者；Planner/Architect/Builder/独立 Reviewer/Verifier/Integrator/Release Coordinator 原则上都由有能力的 Agent 承担，人工仅在目标/范围重大分歧、有效授权/物理 Host/凭据缺口、越界风险预算或确实要求人的发布责任时介入。工程 Gate 不等于 Human Gate；Agent 自动执行仍需要 ADS/GitHub 原生 Claim 和单 DomainHarness K 的真实准入。详见 [Agent 原生执行模型](./AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.zh-CN.md) 与 [English Agent Edition](./AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.en.md)。

## 1. 根本产品问题

普通用户在 Tally 输入产品目标/导入仓库，经渐进式交互确认需求，Tally 按 ADS 组织 Product→L2→Task DAG→真实 Claim/Agent/Host 执行→独立审核→诊断/修复→真实用户功能/集成验证→合法 RQ/发布→下一版。**成功定义为真实软件产品交付及下一版继续，而非 Issue 数、PR 数、Agent 自述 PASS 或 UI 流程展示完毕。**

GitHub 原生 Issue/Dependencies/PR/Review/CI/Release 是工程事实；ADS 规定规范和版本；DomainHarness **唯一 K Primary** 掌握应用状态、Admission、CAS、Journal、Effect、Replay/UNKNOWN；Tally Engineering B 是软件工程领域决策和协调者，选中的 S 提供可复用语义，C 在 K 允许后连 GitHub/Agent/Host/CI。**task-meta 私有 Trace 不由 Tally 查询**：只有通过 ADS 版本合格的 GitHub 公布摘要可读。无遥测不阻碍 Task 正常执行，指标 UNKNOWN ≠ 0。

## 2. 从真实案例抽取的软件工程知识：证据→原则→业务能力

| DomainHarness 事实（来源） | 可复用的软件工程原则 | 对 Tally 的可检验行为 |
|---|---|---|
| [v0.1 #39](https://github.com/kaicreator-mm/domain-harness/issues/39) 可见 117/117 PASS 仍待 Hidden/发布 | **Definition of Done 分层**、Configuration/Release Management | 分列源码、独立 Review、Host/Hidden、RQ/Tag；自动提示合法剩余工作，禁止提前显示发布 |
| [v0.2 #99](https://github.com/kaicreator-mm/domain-harness/issues/99) Android/Hermes 必须真实重启 | Verification vs Validation、适当验证环境 | Host 能力匹配；提供局部人工设备操作和精确证据回传，不用 Node 模拟替代 |
| [v0.3 #177](https://github.com/kaicreator-mm/domain-harness/issues/177)、[#180](https://github.com/kaicreator-mm/domain-harness/issues/180) 13 个下游 App 合约揭露 Tool/provisioning 缺口 | Requirements Engineering、Consumer Contract Testing | Product 可用能力及 public consumer 旅程验收；跨消费者判断共性缺口，不先建巨大抽象 |
| [v0.4 #366](https://github.com/kaicreator-mm/domain-harness/issues/366) 已合并的基线测试类型错误影响下一 Task | Integration Management、Dependency Hygiene | 区分需修复的基线门禁与被阻任务，限定修复写集，修完后重新计算可执行 Lane |
| [v0.6 #472](https://github.com/kaicreator-mm/domain-harness/issues/472) 从过宽的通用引擎转向已有能力的高 ROI Fast Path | Lean/Value Engineering、Reuse-before-build | 每个方案需列已有能力、用户价值、额外成本/回归面、DEFER/REUSE/BUILD 备选 |
| [v0.7 #530](https://github.com/kaicreator-mm/domain-harness/issues/530) 把前一版最终 Closure 错当下一版所有 Source 前置 | CPM/PERT 与真正的 Task dependency、并行架构 | Task DAG 按子需求和独立 Lane 拆分；前序兼容验证落在 Integration Join，不制造假串行 |
| [v0.7 #586](https://github.com/kaicreator-mm/domain-harness/issues/586) 同一 Verification Campaign 多 Checkpoint | Risk-based V&V、合适工作单元粒度 | Campaign 可追踪多证据/检查点；不一条评论一 Task；不重复整版 Full |
| [v0.8 #1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) 重叠 Claim/PR，组件已合并但 Host→Kernel 主流程不可执行 | DDD Identity/Ownership、Integration-first | IssueContainer≠WorkUnit；写集冲突提示；selected public end-to-end 必须实测 |
| [v0.8.1 #1100](https://github.com/kaicreator-mm/domain-harness/issues/1100) stale CAS/Effect 回归 | Change Impact Analysis、Safety Invariants、Regression | 前版 must-retain 行为进入新版本验收，旧 PASS 不被覆盖，新版回归单列 |
| [v0.9 #1119](https://github.com/kaicreator-mm/domain-harness/issues/1119) 外部 clean-install 双 B 消费 | Release Engineering、Interface Contract | 分清仓库内测试与独立安装使用、已选中真实接口；实际能力账本 |
| [v0.10 #1264](https://github.com/kaicreator-mm/domain-harness/issues/1264) 扩展 import/init 但 K 未实际调用 | Requirements Traceability、Test Oracle | 每项必需能力必须有真实运行调用链证据；导出/加载不是行为通过 |
| [v0.10.2 #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385) Windows 15 次复测 4 次失败，初看为偶发 | Defect Triage、Statistical Evidence、Flaky Test Management | 标出复现次数/环境/置信范围；判断根因/产品影响，既不掩盖 P1 也不无界反复 Full |
| [v0.10.3 #1393](https://github.com/kaicreator-mm/domain-harness/issues/1393) Tally 消费发现外部 Provider 完整性缺口 | Supplier/Software Supply Chain、Real Consumer Feedback | 正确路由至 SDK Owner，复用上游修复，必须真实下游回测才能证明闭环 |
| [v0.10.3 #1411](https://github.com/kaicreator-mm/domain-harness/issues/1411) 源 DAG 已规划但原生 GitHub dependency API 未绑定 | Evidence-based Control、Tool Capability Awareness | 候选 DAG≠准入 DAG；告知用户最小本地绑定动作，GitHub 原生读回后才真正派发 |

参照 SWEBOK 的 Requirements/Design/Construction/Testing/Maintenance/Configuration/Management/Quality 知识域，ISO/IEC/IEEE 29148 需求可追踪思路、ISO/IEC 25010 质量特性、ISO/IEC/IEEE 12207 生命周期过程，以及 Lean/PDCA/根因分析和 DORA/SPACE 多维绩效观。**这些是设计知识来源，不宣称正式标准符合性或伪造外部基准数值**。不要把 CMMI/标准条款照搬成每个小 Task 的繁复审批。

## 3. DDD 业务边界：建议的 Engineering B Package 职责

> **逻辑职责包，不等于 v0.10.2/v0.10.3 ABI 已证明可任意组合多个物理 B。** 第一阶段可为一个物理 Engineering Business Package 内多个明确 bounded context；能合法通过公共 BUILD 组合且有两个真实消费者需求后再拆独立部署包。避免“为模块化而模块化”。

| B 领域关注点 / 最小聚合 | 拥有的业务决策（建议接口） | 不拥有的事实/权限 | 首要验收 |
|---|---|---|---|
| **B1 Project/Product**：Project, Version, Requirement, Decision | `proposeProduct`、`evaluateScope`、`draftAcceptance`、`amendmentProposal` | ADS Product Freeze/原生 Issue/用户批准 | 自然语言目标转可审议 PRD，未知不假冻结 |
| **B2 Architecture/Planning**：ArchitectureDecision, WorkUnit, PlanRevision | `proposeArchitecture`、`decomposeIntoLanes`、`evaluateDAG`、`planAgentHostReview` | native blocked_by、真实 Claim、BUILD | 合法并行，发现写集重叠、假串行，提供替代方案 |
| **B3 Execution/Journey**：JourneySnapshot, TaskAttempt, Handoff | `nextActions`、`prepareHandoff`、`proposeDispatch`、`reconcileAfterRestart` | DomainHarness K Effect/CAS/Journal、外部 Job Commit | 能走到真实 Agent/Reviewer/Host；人工只处理必须介入处 |
| **B4 Assurance/Release**：AcceptanceItem, Finding, VerificationPlan, ReleaseCandidate | `explainAcceptanceCoverage`、`proposeRepair`、`selectProportionateChecks`、`proposeRelease` | Review verdict、CI/Host 真结果、Release authority | Import/init≠调用；P1、Missing Host、stale SHA 不误判通过 |
| **B5 Improvement/Experience**：PracticeCase, Candidate, Qualification | `diagnoseBottleneck`、`proposeNextVersion`、`proposePracticeCandidate`、`evaluateReuse` | task-meta 私有日志、ADS 规则权威、BUILD 激活 | 有可检验适用性/反例/版本/使用反馈，不用 LLM 自评级当证明 |

面向用户 #92 J01–J08 的**一个项目工作区**只展示这些 B 领域的一组协调视图；#145 负责持续合法 next action；#146 负责用户可用能力账本；#139 历史事实索引、#140 根因诊断、#141 下一版建议、#142 Issue/Concern 身份、#143 Casebook、#144 下游反馈均复用，不再造第二套产品状态事实。

## 4. K/S/B/C 分层与 Domain Package 候选

- **K Primary（DomainHarness 已选定的可信 Kernel）**：唯一 Instance/State、Effect Admission/CAS、Journal/Replay/UNKNOWN，K Permit 是唯一 Tally Host Effect 前提；Tally 不制作第二调度 Kernel 或 Task CAS。
- **Standard S（优先复用）**：Context、Memory、Rule/Decision/Workflow、候选 Shadow/Experience、按状态生成下一 StepSpec 的 Guided Journey。动态问题集可以预生成、版本固定、执行时按事实组合，仅未知/新场景使用有界 LLM；遵循 [DH #1395](https://github.com/kaicreator-mm/domain-harness/issues/1395)、[#1398](https://github.com/kaicreator-mm/domain-harness/issues/1398)。**是否当前公开可安装应按 exact SDK ABI 验证**，不是声称 v0.10.3 已全部发布。
- **Business B（Tally 自己）**：工程业务决策、用户目标、任务计划、版本协调、评估、实践晋升候选。B 可提出作用于 GitHub/Agent/Host 的 Intent，**不可直接作为批准通过的效果**。
- **Capability C**：选中的 GitHub ADS Reader/Writer、Local Agent/CLI、Model、Host、CI、Release/optional runx；适配器各持自己的受控资源与秘密。C 真实 Effect 通过 K 的合法准入；外部 ACK ≠ terminal。
- **UX Shell**：Web 引导式界面、项目工作区、任务进度、证据导航、人工待办。它不是 Domain Package 权威层，不持有另一套 Task 状态机。
- **数据**：SQLite 可做 source-qualified 可重建索引和合法 K Host 存储；GitHub 仍为原生 Task/PR/Review/CI 真相；task-meta 观测与成本在 GitHub 只读合格报告，没有就 UNKNOWN。

推荐业务流：`用户意图 → B1 Requirement/Acceptance → B2 L2/DAG candidate → ADS 审核/原生准入 → B3 B→K→C Agent/GitHub Effect → B4 QA/Release → B5 Evidence-driven feedback → B1 next version`。技术上的 K/S/B/C 拓扑不是这个有向阶段列表的串行锁；允许独立工作 Lane 提前开发，只有真正的 Join/Gate 才等待。

## 5. 契约草案（语义示例，**不是已发布 DTO 或 SDK 方法**）

```ts
type EvidenceStatus = "OBSERVED" | "DECLARED" | "INFERRED" |
  "UNKNOWN" | "STALE" | "NOT_RUN";

type SourceRef = {
  repository: string; url: string; sourceSha?: string;
  adsPin?: string; observedAt: string; status: EvidenceStatus;
};

type NextActionProposal = {
  versionRef: SourceRef; actionId: string;
  concern: "PRODUCT"|"ARCHITECTURE"|"TASK"|"REVIEW"|
           "HOST_VALIDATION"|"REPAIR"|"RELEASE";
  reason: string; evidence: SourceRef[];
  readiness: "AUTO_ELIGIBLE"|"NEEDS_USER_DECISION"|
             "NEEDS_HUMAN_HOST"|"WAIT_EXTERNAL"|"BLOCKED"|"UNKNOWN";
  suggestedActorRole: string; requiredCapability: string[];
  blockedBy: SourceRef[]; authorityRequired: string;
  // This DTO grants NO GitHub Claim, K permit or merge permission.
};

type AcceptanceCoverage = {
  requirement: SourceRef; expectedBehavior: string;
  inheritedInvariantRefs: SourceRef[];
  implementationRefs: SourceRef[];
  selectedInstalledConsumerProof: SourceRef[];
  requiredVerificationRefs: SourceRef[];
  evidenceCompleteness: "COMPLETE"|"PARTIAL"|"UNKNOWN";
  // Distinct from approved ADS Release Qualification.
};

type PracticeCandidate = {
  fingerprint: string; applicableWhen: string[];
  evidenceRefs: SourceRef[]; counterexamples: SourceRef[];
  proposal: "REUSE_RULE"|"PROCEDURE"|"SCOPED_REPAIR"|
            "TEST_STRATEGY"|"AGENT_ROUTING"|"DAG_PATTERN";
  status: "DRAFT"|"EVALUATED"|"INDEPENDENTLY_REVIEWED"|
          "APPROVED_FOR_NEXT_BUILD"|"REJECTED";
};
```

任何 `status` 只是 Tally 对来源的**解释性投影**，不得替代原生 ADS status 或 DomainHarness K 真实状态；用户点击“执行”必须重新验证当前 source/权限/Claim/Effect admission。

## 6. 最小工程交付切片及验收矩阵

1. **Vertical A — 单项目工作区+真实来源**：J01–J03 用户输入→PRD/L2/DAG 草案、出处、待决议；即使没有 LLM/Agent/任务指标仍可工作。
2. **Vertical B — 一个真实 Task**：从真正冻结的 ADS Task 和合格 GitHub native dependency/CAS Claim → B→K→C Local Agent/Host → PR/Review/CI；外部超时/丢 ACK 不自称完成。这是 [#62](https://github.com/kaicreator-mm/tally/issues/62)。
3. **Vertical C — 独立 Lane 与真实用户验收**：两个不冲突 concern 同时进行；其中一个 Review BLOCK→根因修复→独立复审；真实 installed/selected consumer evidence，J06 能说明功能确实可用，而非 export/pass count。
4. **Vertical D — 发布和下一版**：合格 RQ、合法 Version/Main merge/Tag，旧版本证据保持，下一版按已验证原能力、历史经验和新目标规划。用户始终通过 Tally 工作区操作必要步骤；外部 OAuth/Host/人类审批精准告知且有读回。
5. **反例 Golden（至少）**：DH #530 假串行、#586 Campaign 非 Task、#1013 冲突/断开的 Host→K、#1100 安全回归、#1264 imported≠invoked、#99 真实重启、#1385 重复观测和 #1411 source DAG≠native admission。采用 [#143](https://github.com/kaicreator-mm/tally/issues/143) 同一 Casebook，不复制历史事实库；未经历史时点可见性重建的样例不能称“提前预测成功”。

## 7. 如何防止 Domain Package 过度设计

按 `用户结果 → Domain 意义 → 已有 B/S/K/C 能力 → 真缺口 → 最小公共扩展 → 实际 clean consumer` 决策：
- 同一业务领域的两个子步骤，不因为流程图有两格就拆成两个 Package/Issue。
- 无真实共性需求时，保持一个 B Package 内的纯函数或规则；多下游独立复用且公共 ABI 证据充足时才考虑抽为选中的 S Feature 或 C。
- 对安全关键不变量保留严格 Gate，对边际低风险条件用有界修复/已知限制/延后处理，不把小概率非致命输入拖成版本永不发布；实际 P0/P1 的授权、来源与不幂等 Effect 仍不得放宽。
- 优先成熟软件工程/Node24 生态库，不能在 Tally 重写 Git、SQLite、Agent Provider、Task telemetry、DomainHarness K 或 CI。
- 软件工程知识作为**版本化、可解释、可拒绝、可测试**的实践资产，非 LLM 隐含 Prompt、非模型权重更新。

**下一阶段**：本文件经 Product/L2 独立复审裁定与已有 PRD/接口交叉对账后，由 #91 按合法 Task DAG 拆成最小写集；应避免现有 `version/v2` 在途 Builder、exact-SHA Review、冻结合同被此研究文档隐式修改。
