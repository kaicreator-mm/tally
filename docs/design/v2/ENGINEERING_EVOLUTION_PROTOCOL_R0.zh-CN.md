# task-meta × ADS × Tally：从软件开发实践进化的职责与协议（R0 / 中文）

> 2026-10-28 · **跨项目 Product/L2 研究提案；无规范 Freeze、无自动规则发布、无代码/Task Claim 授权。** 配套：[Tally Domain Package 基线](./TALLY_ENGINEERING_DOMAIN_PACKAGE_BLUEPRINT_R0.zh-CN.md)、[Agent 英文版](./ENGINEERING_EVOLUTION_PROTOCOL_R0.en.md)。源项目所有者仍由各自项目的正式治理过程批准变更。
>
> 现有 Owner：task-meta [#12](https://github.com/kaicreator-mm/task-meta/issues/12)、[#17](https://github.com/kaicreator-mm/task-meta/issues/17)、[#27](https://github.com/kaicreator-mm/task-meta/issues/27)、[#28](https://github.com/kaicreator-mm/task-meta/issues/28)、[#29](https://github.com/kaicreator-mm/task-meta/issues/29)；ADS [#1345](https://github.com/kaicreator-mm/ai-development-standard/issues/1345)、[#1347](https://github.com/kaicreator-mm/ai-development-standard/issues/1347)、[#1346](https://github.com/kaicreator-mm/ai-development-standard/issues/1346)；Tally [#57](https://github.com/kaicreator-mm/tally/issues/57)、[#58](https://github.com/kaicreator-mm/tally/issues/58)、[#60](https://github.com/kaicreator-mm/tally/issues/60)、[#139](https://github.com/kaicreator-mm/tally/issues/139)、[#140](https://github.com/kaicreator-mm/tally/issues/140)、[#141](https://github.com/kaicreator-mm/tally/issues/141)、[#143](https://github.com/kaicreator-mm/tally/issues/143)、[#145](https://github.com/kaicreator-mm/tally/issues/145)、[#146](https://github.com/kaicreator-mm/tally/issues/146)。DomainHarness 跨任务元进化依据 [#1398](https://github.com/kaicreator-mm/domain-harness/issues/1398)。

## 1. 进化目标与循环

目标不是“训练模型后自动编写新规范”，而是从**可观察的真实软件结果**中不断改进三类不同资产：
1. **事实质量**（task-meta）：更准确、可归因、可比较、隐私安全的执行观测；
2. **工程规则**（ADS）：更少无效步骤、但保留真实安全性和源权威的过程/验收规则；
3. **执行知识与产品能力**（Tally）：能在合适场景复用的工程计划、Domain Package 决策/表单/Agent/验证/修复 Procedure。

```text
用户目标/现有真实项目
     │
     ▼
Tally B/S: Product→L2→Task DAG→Agent/Host/Review/Release 候选
     │                     │
     │ 现行 ADS exact-pin  │ 合法 Claim/角色/证据门禁
     ▼                     ▼
GitHub native Issue/PR/CI/Review ←→ DomainHarness K-admitted C Effects
     │                                 │
     │ 真实结果/源事件                 │ Task→Attempt→Step→Tool/Verifier
     │                                 ▼
     └─────────────────────────── task-meta 采集（私有）
                                      │ 安全且版本合格的公开投影
                                      ▼
                           ADS 结构化 GitHub Task Report
                                      │
                      Tally selected GitHub C 只读
                                      ▼
                   Tally evidence graph / diagnose / use feedback
                      ├── telemetry gap → task-meta Issue
                      ├── normative/process flaw → ADS Issue
                      ├── app behavior/UX → Tally Issue
                      └── generic reusable SDK defect → DomainHarness Issue
                                      │
                          source-qualified 候选+反例
                                      ▼
                           回放/Shadow/独立审核
                                      │
                 现有授权与版本化发布，或明确 DEFER/REJECT
                                      ▼
                新版 ADS pin / task-meta publisher / Tally BUILD
                                      │
                         新任务/真实消费者回测+再评价
```

**GitHub 是跨项目交换可公开审查事实的入口**，不是新引入一个中心数据库。Tally 不直接拉取 task-meta 私有运行细节或凭 manifest 访问原始 Trace；向 GitHub 写反馈也必须通过已批准的 GitHub C + DomainHarness K Effect。某环节没有数据时应继续合法 Task，但该项分析/成本/策略建议输出 UNKNOWN。

## 2. 两个层次、四条不同时间尺度的进化

**A. 单次 Task 运行内的智能决策（快）**：按 [DomainHarness v0.3 历史 #1398](https://github.com/kaicreator-mm/domain-harness/issues/1398) 已冻结的默认顺序，优先 `编译确定规则 → Exact semantic reuse → 已合格晋升的适用 Subworkflow → 有界 Harness/LLM fallback`；只有当前已知语义/公共 ABI 确认能实现时才使用。保持 K 效果边界；模型输出是建议，不会因为“更聪明”自动改规则。未知来源或过时经验 -> 不应用/请求必要信息。

**B. Task/Workfront 反馈循环（分钟～天）**：实际 Agent/工具/Review/CI 后进行纠错、重试、根因分类、Task-level Context Pack 与 Agent/验证策略调整；动作若不改变接受的 Product/DAG/Claim 可采用原有授权的继续/恢复语义；涉及写集/Owner/冻结范围则提出 **ADS amendment/Claim handoff**，不能自动抢 Claim。Tally #57/#58/#67/#145 负责；task-meta 只记录事实。

**C. 版本内工程改进（天～周）**：从已发生的失败中决定是否合理拆分任务、复用成熟库、停止无效 Full Loop、改变测试范围、补充关键真实消费者验收；由 Tally B 产生可审议选择，依据 ADS 当前版本进行受控 replan，保留此前已批准的证据/身份，避免把每次 Review finding 都升级为 Product 重构。

**D. 跨版本/跨项目改进（周～月）**：从多个真实实践生成 Pattern/Candidate，研究适用条件/反例，经过独立检验后：
- 属于 **task-meta**：采集/校正/公开投影协议的新版本；
- 属于 **ADS**：经独立审核并发布的新规范版本，项目显式采纳 exact pin；
- 属于 **Tally B/S/C**：经源/消费者验证、独立 Reviewer/适用 Approval、**新的闭世界 BUILD/部署**，仅新选定 occurrence 使用；运行中旧实例保持其 pinned Package identity。
不同项目并不需要同步升级；每次版本变更写明向后兼容、消费者迁移、缺失/过时行为。它不是 K Runtime 的自动热更新。

## 3. 三个项目分别怎样从实践学习

### task-meta：事实生产者 / Measurement System

**采集对象**：有合法相关来源的 `Task→Attempt→Step/Operation→Tool/Model/AgentRole/Verifier/Host`，采用稳定相关键、原始观测时间与时区、tool/result/failure fingerprint、模型/token/计费来源、执行尝试、排队/人类等待/CI 耗时（仅能够来源证明的才统计）；`recorded/estimated/not_observed/not_supported` 覆盖度不可混用。纠正被编辑的 Task Report、重复发布、缺失版本/source SHA、跨 Task 归因歧义。

**源→报告质量**：`Task/Attempt/operation` 关联需与合法 ADS/GitHub native source identity 对齐，输出有界结构化的**GitHub 可见** Task 结果/成本/质量/适用性/coverage 和真实 correction notice；原始 CoT、Prompt、凭据和个人私有 Trace 不公开。task-meta 采集故障不能影响 Tally 发起、Review 或完成工程 Task。

**自身进化路径**：当发现指标缺口/偏差，先造采集负例与黄金测试，修 Collector/normalized schema/publisher，保留旧报表版本身份和更正记录，再做真实 Tool provider+GitHub publisher→reader E2E；比较新旧覆盖度/重复归因/异常识别，拒绝仅因统计可用而断言性能已提升。拥有 **事实语义与采集覆盖**，不拥有 Agent 调度/工程规则权威/Rule 晋升。

### ADS：规范与工程治理 / Process System

**输入**：Tally 提供的具有 SourceRef 的流程失败候选 + 原 GitHub/PR/CI/Review 事实 + task-meta 合格摘要；不接受“某次 Agent 感觉应该如此”直接升级规范。

**规范评估案例**：DH [#530](https://github.com/kaicreator-mm/domain-harness/issues/530) 说明错误串行；[#586](https://github.com/kaicreator-mm/domain-harness/issues/586) 是减少不必要拆 Issue 的正控；[#1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) 说明 Claim/Concern 冲突；[#472](https://github.com/kaicreator-mm/domain-harness/issues/472) 说明过度抽象；[#1385](https://github.com/kaicreator-mm/domain-harness/issues/1385) 说明重试和失效分类；[#1100](https://github.com/kaicreator-mm/domain-harness/issues/1100) 说明绝对不可为了速度忽略核心 CAS/Effect 不变量。

**自身进化路径**：`真实 Failure Pattern → 反例/适用性/危险界限 → 最小规则改动提案 → 正/反 Golden + 项目影响矩阵 → 独立 Review → Version RQ/Release → 每项目显式采纳 pin → 新项目实战效果回收`。对用户价值低的小边际场景允许 DEFER / documented limitation；对 P0、实证严重 P1 和原生 Claim/K 授权绝不可用“降门禁提速”处理。把过度 Full/Hidden、错误 Task DAG、重复 Review/人工交接转成**比例化**流程，不造新管理层审批。

### Tally：工程业务与 Domain Package / Decision & Delivery System

**输入**：当前用户目标/产品/架构与 Acceptance、当前 ADS pin、native GitHub WorkUnit/Claim/PR/Review/CI/Release、经过批准公开的 task-meta 摘要、可用 Agent/Host、历史 Casebook 和用户反馈。

**执行内**：`B Product→Planning→Journey→Assurance` 负责动态 next actions、多 Agent 写集/角色编排、在前方识别集成/真正 Host 验证、根因优先于反复补丁、合格且有限的经验复用。S 负责确定规则/上下文/有界 LLM，K 负责真实 effects，C 负责连接 GitHub/Agent/Host。完整软件开发界面由 #92；Business Driver #145；用户可用能力账本 #146。

**自身进化路径**：`Task/版本运行结果 → 归因到能力缺口/过程成本 → PracticeCandidate(适用条件/不适用/反例/源链接) → 与无策略/既有策略 baseline 对比 → 独立 Golden/holdout/真实消费者验证 → Product/L2 审查/授权 → New BUILD/Activation → next version real-use follow-up`。重合的个别经验可以先写为 B 规则/静态 Scenario 或 Context Pack，而不是马上创建独立 S Package；只有两个真实消费者、公共接口足够稳定时再向 DomainHarness 提独立可复用 Feature。Tally 不能从自己生成的“PASS”自动决定 Promotion。

## 4. 可交换的有界证据契约建议（**尚未冻结的逻辑 DTO**）

```ts
type Provenance = {
  sourceRepo: string; githubUrl: string; exactSourceSha?: string;
  adsRevision?: string; versionRef?: string; observedAt: string;
  sourceKind: "NATIVE_GITHUB"|"ADS_PUBLISHED_REPORT"|"READ_ONLY_DERIVATION";
  coverage: "COMPLETE"|"PARTIAL"|"UNKNOWN"|"NOT_SUPPORTED";
};

type PracticeFinding = {
  kind: "FALSE_SERIALIZATION"|"CLAIM_OVERLAP"|"REVIEW_LOOP"|
        "ARCHITECTURE_ROOT"|"MISSING_CONSUMER_PROOF"|
        "TEST_FLAKINESS"|"HOST_HANDOFF_GAP"|"SDK_DEFECT"|
        "TELEMETRY_GAP"|"UNKNOWN";
  evidence: Provenance[];
  observedImpact?: { metric: string; unit: string; value?: number;
                     measurement: "MEASURED"|"ESTIMATED"|"UNKNOWN" };
  suspectedOwner: "TALLY"|"ADS"|"TASK_META"|"DOMAIN_HARNESS"|"APP";
  contraryEvidence: Provenance[]; confidence: "PROVEN"|"CANDIDATE"|"UNKNOWN";
};

type ImprovementProposal = {
  findingRefs: Provenance[]; targetProject: string; targetVersion?: string;
  candidateType: "DATA_CONTRACT"|"NORMATIVE_RULE"|
                 "B_RULE_OR_PROCEDURE"|"S_FEATURE"|"C_ADAPTER"|"DOC_ONLY";
  applicableWhen: string[]; notApplicableWhen: string[];
  requiredGateOwner: string; forbiddenAutomaticEffects: string[];
  expectedUserOutcome: string; testCases: string[];
  state: "DRAFT"|"TRIAGED"|"EVALUATED"|"INDEPENDENTLY_REVIEWED"|
         "APPROVED_FOR_NEW_VERSION"|"REJECTED"|"DEFERRED";
};
```

传输不是强制所有包采用此 schema：以**所采纳 ADS 报告方言现有字段优先**，不足时提出相应仓库 Issue。Github SourceRef 是可回溯索引，不能把本提案包装为现有 `ads/github/v3` 原生 Event 或授权声明。GitHub 报告应尽量有界，复杂因果分析保留为 Tally 的派生 read model 和可审计引用。

## 5. 度量与真实改进证明（不能从 Issue 时间计算 Agent 工时）

| 指标 | 正确口径/价值 | 不能做的推断 |
|---|---|---|
| 完整软件版本交付 | 用户可用功能的实际验收覆盖、Release RQ/Tag、下一版开始 | PR merged/Issue CLOSED ≠ 软件完成 |
| Lead time | 明确 `start/end` 口径的 **日历经过时间** | Issue created→closed ≠ Agent 工作小时 |
| Agent/Tool 成本 | 只汇总 task-meta 报告中来源合格的 measured 计费/Token/active，展示 coverage | 缺采集填 0、估算等于账单、跨模型无条件比较 |
| Waiting & overhead | 有证据的 Human/CI/Review 等待，重复 Handoff 和无效重试 | “Review 多”必然浪费；P0 安全 Review 要保留 |
| 并行效率 | 有源 DAG 真依赖、可独立写集、时间重叠、实际 Join/冲突 | 减少依赖边数就一定更快；强制并行写同一个 manifest |
| Repair effectiveness | 同类根因复发、修复后用户功能验证通过、因临时补丁新增缺陷 | 修复 Issue 关闭等于根因已解 |
| Strategy contribution | 真实对照组、时间界限、适用性/反例/风险、用户接受反馈 | 事后案例回放自动等于当年正确预测 |

比较时按 Agent 角色、任务复杂度/风险、语言/Host、模型/工具版本、验收覆盖、项目类型分层；量小只给示例不虚构百分比。使用 [#143](https://github.com/kaicreator-mm/tally/issues/143) 的 `RETROSPECTIVE_DIAGNOSIS` 与 `HISTORICALLY_BLIND` 两种互斥模式。看不到当时输入变化历史 → `NOT_RECONSTRUCTABLE` 而不是声称可提前发现。

## 6. 发布、回读与防止错误进化的 Gate

1. **Intake**：真实 Issue/PR/CI/Host/ADS 报告来源和证据覆盖明确。重复 Finding 合并为候选，而非自动创建多个跨仓库 Task。
2. **Triage**：判断产品/SDK/规范/观测具体 Owner；区分新公共能力、文档误用、局部业务 bug、真实 P0/P1 安全性；无足够事实为 UNKNOWN/REQUEST_EVIDENCE。
3. **Candidate**：规定 what/why、反例、适用域、实现范围、成本/安全边界、negative controls；模型可以生成文本但不能做 approval。
4. **Evaluation**：廉价 source/static/isolated tests 在先；对风险与真实用户路径选择集成/Host/Hidden；独立 Reviewer 校验测试 oracle；用 Casebook Golden/holdout 排除过拟合/事后信息泄漏。
5. **Approval**：ADS 由规范 Owner/独立 Review+版本 RQ 决定；task-meta 由观测 schema/publisher Owner 决定；Tally 由 Product/L2 和实际选中 Package BUILD/RQ 决定。任何文档研究或 GitHub comment 都不会自动构成通过。
6. **Adoption**：新 ADS pin 显式采用；task-meta 报告版本/读方言明确；Tally 新 sealed Package BUILD、已选中 Agent/Host/C 适配、运行实例固定身份，不对旧实例热替换。
7. **Real consumer validation**：改动后验证真实 Project/Issue→Agent→Review→User Acceptance→Release 旅程，对 SDK 反馈还要 clean consumer 回测；失败保留原 Candidate 和阻塞/回滚路径。
8. **Outcome**：将真正观测到的后续成效及缺失信息重新发布/索引，下次改进可以接受、拒绝或废止此前策略；必要时按现有新版本流程修正规范/Package。

## 7. 用真实实践检验这份协议的 6 个 Golden

| 案例 | 正确提案归属 | 禁止的错误行动 |
|---|---|---|
| DH [#530](https://github.com/kaicreator-mm/domain-harness/issues/530) 不必要跨版串行 | Tally Task DAG proposal；ADS 并行原则适用性 | 自动删 GitHub native blocked_by |
| DH [#586](https://github.com/kaicreator-mm/domain-harness/issues/586) 合法单 Campaign | Tally WorkUnit/Campaign 语义；ADS 正控 | 为每个 checkpoint 开新 Feature Task |
| DH [#1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) 重叠 PR + 真实集成缺失 | Tally Owner/workset/conflict 和 Assurance | PR 全部 merged 即 RQ PASS |
| DH [#1100](https://github.com/kaicreator-mm/domain-harness/issues/1100) stale CAS P0 | ADS 必需不变量、Tally must-retain 验收 | 用速度/成本理由削减安全 Gate |
| DH [#1264](https://github.com/kaicreator-mm/domain-harness/issues/1264) loaded≠invoked | Tally Acceptance Coverage/集成测试模式 | Import PASS 即用户功能通过 |
| DH [#1393](https://github.com/kaicreator-mm/domain-harness/issues/1393) 下游 SDK Provider integrity | DomainHarness 通用 SDK Owner；Tally/ADS 消费验证 | Tally 自己另造整个 npm/K 安全体系 |

## 8. 首期可执行切片（减少新系统）

- 用现有 Tally #143 Casebook 的 6–8 个 source pinned 案例，创建**只读** `PracticeFinding → SuggestedOwner → Candidate` 结果；不要求模型参与也可重放。
- 连上 #145 Version Journey Driver 的 next-action 候选、#146 Acceptance Coverage 和 #140 实际流程诊断。构成一个 Tally B 横向切片；无需额外中央进化引擎。
- task-meta 先复用现行 ADS GitHub published report/coverage，只针对可实证缺失向 task-meta #12/#17/#27 反馈；ADS 先复用 v4.19 #1347 Intake 和 #1346 Golden，不人为增加第二规则发布体系。
- 合法发布后，用两次真实 Tally 软件版本交付衡量用户在 Tally 内完成开发的能力；只有可对比证据充分才计算 Agent 工时/质量/成本变化，持续监测回归。

**状态：** `DOCUMENT_R0=PROPOSED` · `ADS_NORM_CHANGE=NONE` · `TASK_META_PRIVATE_DATA_ACCESS=NONE` · `TALLY_PACKAGE_PROMOTION=NONE` · `PRODUCT_FREEZE=NO` · `IMPLEMENTATION_CLAIM=NO`。
