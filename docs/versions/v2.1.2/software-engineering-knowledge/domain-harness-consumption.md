# Tally 软件工程 Domain Intelligence × DomainHarness：消费与服务边界
> [知识体系](systematic-knowledge-architecture.md) · [系统课程](knowledge-coverage-and-growth.md) · [来源预核查](selected-source-claims-preflight.md) · [Tally 消费规划 #286](https://github.com/kaicreator-mm/tally/issues/286)
>
> **设计提案：ARCHITECTURE_CANDIDATE / NOT_IMPLEMENTED / NOT_L1_L2_FROZEN**。这里不是提交新的 SDK/Package JSON Schema，不为 Tally 或 DomainHarness 加入已授权 Task。0.11.x 是 DomainHarness **完全重写并在单列车最终统一发布**，不能将旧 v0.10.3 / Draft 接口等同新版可用 API。引用范围按实际 GitHub Issue 最新 owner 记录。

## 1. 三层拥有者：知识领域、专业应用、通用 Harness

| 层 | 负责的实体和行为 | 不拥有 |
| --- | --- | --- |
| **ADS（来源）** | 已发布软件开发工程规则、可追溯的实践实验、原始成功/失败/反例、可信来源和版本变更 | Tally 私有 Agent Runtime 或所有任务决策 |
| **Tally 软件工程领域 App/选定 Package** | K01–K11 的概念与业务语义、适用性、知识索引、专业 Rule/Method/Skill/Case 的选择、领域 Workflow 对接、工程需求与反馈回流 | DomainHarness Kernel、额外通用知识编译平台、GitHub/Host 的第二权威 |
| **DomainHarness 通用执行基础** | Element/Function/Package 构建和选择、KDP/Workflow/Function 执行、未来的 Domain Intelligence/受限推理、真实 Evidence 与按授权反馈 | 软件工程专业判断的硬编码、Tally 自定义业务规则或 ADS 规范内容 |

Tally #286 最新 Owner 顶部已明确：ADS 实践→Tally Engineering Domain Package 不以第三方知识生产项目为先决条件；历史 #286 下方“必须由 DomainForge 提供全部知识”观点已被其最新决议替代。**但 #286 仍为 `DEFERRED/NOT_FROZEN/NO_SOURCE_DISPATCH`**；本库文档只构成该未来 Product/L2 的输入，不自动允许现在实施 Runtime。

## 2. 与 DomainHarness 0.11.x 当前阶段的精确对接

| 阶段 / 权威 | 本知识库希望被使用的内容 | 当前应承认的事实 |
| --- | --- | --- |
| [0.11.0 Foundation #1564](https://github.com/kaicreator-mm/domain-harness/issues/1564) | 仅把未来选定知识和 Function 看作需要 Element/Package/BUILD 的可承载内容 | ElementRef / FunctionRef / BUILD/Lock/KDP 基础仍按 #1624 修正；不得发明额外 `EntryRef` 兼容层 |
| [0.11.1 Task Execution #1605](https://github.com/kaicreator-mm/domain-harness/issues/1605) | Workflow 选用已批准专业方法/知识输出，Task Step/Outcome 不混成知识事实 | L1/L2 候选，常规业务路径 Package.Init→Workflow→Function，尚非已发版 API |
| [0.11.2 Domain Intelligence #1612](https://github.com/kaicreator-mm/domain-harness/issues/1612) | Domain Concept/Relation/Fact、Knowledge/Rule/Recognizer/Validator/Decision 与有界 Reader/Query；前提、UNKNOWN 和来源解释 | 该功能尚未实施；只将专业内容映射为**可能的 selected Element Content/Function**，不预设具体机器 Schema |
| [0.11.3 Semantic Interaction #1613](https://github.com/kaicreator-mm/domain-harness/issues/1613) | 用 K01 领域对象/术语理解用户 Goal 与缺失业务输入，ASK 而非猜政策 | 未来选择性交互，不把原始对话当 Business Rule |
| [0.11.4 Authorized Action #1614](https://github.com/kaicreator-mm/domain-harness/issues/1614) | 领域方法给出需要哪些能力；真正 Effect/Host/权限仍由原 owner 约束 | “推荐调用 Tool”不是运行或授权证明 |
| [0.11.5 Bounded Reasoning #1615](https://github.com/kaicreator-mm/domain-harness/issues/1615) | 确定性可解则无需 LLM；知识不足或歧义时，只在获权范围内使用 LLM 输出校验过的候选 | 当前文档不是已部署的 LLM Provider/Tool Loop |
| [0.11.7–0.11.9 #1617/#1618/#1619](https://github.com/kaicreator-mm/domain-harness/issues/1617) | 当前知识版本、被选 Method/Rule、实际任务 Outcome 和错误/成本；Client 精确经验复用、Server 经授权分析候选 | 这些执行/反馈功能尚未实施，不能把计划写成真实进化闭环 |

**边界策略：** 先规范领域语义和知识选择，再等相应消费者 L1/L2 冻结后接入已有 Element/Function/Workflow。不要因为想让本知识库“可执行”就在 Tally 里克隆 DomainHarness 的 KDP、Task Loop、规则执行器、知识数据库或 Agent Runtime。知识内容可以文件优先，必要的索引/机器格式由未来真实消费者需求决定。

## 3. 领域消费的逻辑过程（概念行为，不是当前运行接口）

```text
Tally Engineering Business / 当前 GitHub Task
  → 确认业务 Goal、Owner、事实/版本、当前支持的 Task 权限
  → Tally Knowledge Selector（未来，依据专业域 + Topic + 限定前提）
     → 优先完整学科概念/原理/关系
     → 适用的确定性 Rule / DecisionProcedure / Validator
     → 正反例、来源、版本、许可和冲突检查
     → 当前证据足够？ YES：解释/给出受限步骤
                       NO：ASK / UNKNOWN / 有权范围内的 LLM 候选
  → DomainHarness selected Workflow/Element.Function（未来，不新增执行引擎）
  → 真正 Host/CI/GitHub 观察与现有 Validator/Review/Release Owner
  → 事实 Evidence + Scope/Pins + 假设/失败
  → Tally 知识维护候选和限定反馈，不自动晋升 Rule
```

**消费者选用不需要一次载入整个学科**：完整 BOK 是可浏览/可推理的知识**库存**，选定 Knowledge/Rule/Procedure 仅构成当前任务有界**上下文**。没有真实确定性 Rule 或 Source 缺失时，不得用高相似度的其它版本经验替代受权行为规则。

## 4. Tally 层的知识服务需求（产品目标而非已实现 API）

| 能力问题 | 最小领域能力 | 可观察正确结果 | 不允许的结果 |
| --- | --- | --- | --- |
| 目标是什么专业问题？ | `classifyGoal` 概念候选：Kxx/Subject/Topic 多标签及理由 | 订单取消优先 K01.Rule/K05.Oracle，非 K08 CI | 因关键词“发布”直接跳到 Release |
| 哪些知识有当前适用性？ | `selectKnowledge`：输入事实、版本、角色、平台、有效期、版权、来源/资格 | 只返回当前事实可适用的知识及缺口 | 未资格 Claim 变成 Rule |
| 当前约束如何推断？ | `evaluateCondition` / `explainDecision`：稳定、可证伪、确定性优先 | 同一已知事实得同一合法结论及引用来源 | LLM 生成违反业务规则的“必然 PASS” |
| 哪里存在知识冲突？ | `resolveConflict` 候选：条件同一性、来源级别、时间和 UNKNOWN | 有相同条件相斥时 ASK/UNKNOWN | 选最新网页来覆盖 Product Rule |
| 哪个方法能指导实施？ | `chooseProcedure`：DP/Skill 输入/STOP、最小跨模块闭包 | 只给任务需要的 K02+K05，保留授权 | 为每项小任务自动生成 11 个子 Task |
| 效果真实有改善吗？ | `evaluateKnowledgeUse`：测试/用户结果、版本、错用/成本 | 提供对照、准确 failure + unknown | 自评好就自动上架成正式 Rule |

以上函数名仅是用于讨论的**服务能力名称示例**，不宣称是当前 DomainHarness/Tally public SDK，也不是要求在 #521 直接新增接口。

## 5. 知识、规则、流程、工具的严格边界（防止系统演化时混淆）

- **Concept/Knowledge**：例如“重试时可能有重复副作用”解释真实工程原理，是任何开发者都可阅读的候选专业知识。
- **Rule**：例如“已发货订单不得通过消费者端取消”，必须来自当前 Product 合法业务政策并有真实适用条件，才能成为选定领域规则；不能把教学案例的假定政策强行写入 Tally Rule。
- **DecisionProcedure/Skill**：解释如何读取现有 Rule、做验收/诊断/取舍，输出建议/候选或合法受权操作；并不代表已获得某个 Agent 可写权限。
- **Workflow**：组织 L1/L2/L3/Review/Repair/CI 等实际 Task 与相应读回。允许多个知识主题支撑一个 Task，也允许一个专题复用多个 Workflow；本章不是固定三阶段 DAG-P/A/I 正式发布依据。
- **Function/Tool/Effect**：由实际选定 Element、合法 Host/Provider 执行并留下真实结果；模型/Agent 的答案不是副作用收据。
- **Feedback/Experience**：保留成功、失败、负例和限制，提供可审查候选；只有原本有权的选择/验证/版本机制才能改变未来被选用的 Package。

## 6. 端到端分层例子（均未执行）

**问题 A：要求对坏订单状态拒绝取消**

- K01 解析角色、状态和 Rule 来源；无权来源时 HOLD。
- K05 读取有效 Rule 生成正例/非法状态负例；必要时 K02/K03 定义状态/实现。
- Harness 将来只在明确选定的 Workflow/Function 运行该测试；未获得真实 Host/Test 证据时 `NOT_RUN`。
- 不是因为 README 写了正反例就认定订单系统实际拒绝非法取消。

**问题 B：CI 绿灯却目标测试没有执行**

- K05.FALSE_PASS 用真正 Test Runner 收据判断目标是否发现；K08.PIPELINE_STAGES 解释阶段事实。
- 如果缺目标测试，仅已有测试范围有自己的 PASS；材料目标仍 `NOT_RUN`，不能合成 E2E PASS。
- 真实符合既有 ADS 验证权限的执行者才能补跑或声明对应 Gate，知识消费者仅辅助识别。

**问题 C：旧 Worker 超时后迟到 ACK**

- K07.STALE_RESULT 分离 transport 和 canonical GitHub 状态，K10.UNKNOWN_EFFECT 指出可能存在已发生外部效果。
- 对比来源 SHA/Attempt 和最新 Task readback；不知道 Effect 时先询问合法 readback，不擅自重复。
- 未来 Harness 只能作为已授权执行通道，不发明另一套 Task/Review 事实数据库。

## 7. 真正接入 DomainHarness 前还需要哪些具体事实

现阶段不把 v0.11.2 未冻结概念建成生产结构。#286 后续合法 Product/L2 应明确：选定 Domain/Package/ElementRef 与 FunctionRef 身份、知识 Content/Rule/Method 类型与合法 Source/Version 绑定、BUILD/Lock 的选取粒度、Reader/Validator 实际接口、权限/数据隔离、UNKNOWN/冲突协议、确定性 vs LLM 选择条件、允许的反馈路径和真实用户任务验收。**先用 Tally #276 既有 Engineering Business/Domain Data 的真实现有接口核对，才决定最小适配**；不建立双份 KnowledgeGraph/Rule Engine，也不提前假设选择泛化 SQLite/Vector DB。
