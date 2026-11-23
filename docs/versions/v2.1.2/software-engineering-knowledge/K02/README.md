# K02 软件工程知识库：架构与系统设计

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #523](https://github.com/kaicreator-mm/tally/issues/523) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

以需求驱动、状态/接口合同、替代方案和失效语义作出可解释的架构决策。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：架构质量场景与状态一致性](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：依赖方向、架构视图与分布式状态](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R4 逐点学科缺口补齐（独立 ACCEPT_SCOPED @ 42e7ba8bd883ec79736e25ff7c75bdbda1b738b3，旧 SHA 限定文档资格；R5 的 K02 Extension import/HostPort 权限边界已独立限定复核通过）**：[`K02.F4.3` 架构信任边界/失败模式](focus-f4-3-minimal-failure-and-trust-boundary.md)。这些文件是对课程的定点专业深化，包含机制/条件/反例/STOP；并非获权 Source Claim、CI/Host/Package 或产品发布资格。

**R6 进阶原理与专业反例（旧 Exact SHA `83ebee092db412b5e38db0376562ee51b0d4f032` 已获独立 ACCEPT_SCOPED，文档范围）**：[质量场景、分布式状态与故障/信任边界推理](advanced-architecture-invariants-and-failure-tradeoffs.md)。对本模块的选定课程 ID 补充可验证的机制、适用前提、替代选项和否证轨迹；相应 [W06–W12 跨模块完整演算](../professional-reasoning-worked-traces-02.md) 均为教学而非实际运行。无 Source Claim、DomainHarness ABI 或生产 Agent/Host 权限资格。

**R8 深化工程判断专业基础（旧 Exact SHA `258f5cd6ff8d34a62b99d1f81f5e5a8c0167062d` 已独立 ACCEPT_SCOPED，文档范围）**：[Extension/Host/状态 Owner/版本合同的正交边界](advanced-extension-authority-state-and-consumer-boundaries.md)，精确连接本模块 **4 个原有高优先级课程 ID**，明确 Source/范围、机制假设、负例与 STOP；[W13–W17](../professional-reasoning-worked-traces-03.md) 展示综合选择，[来源适用矩阵](../professional-source-applicability-r8.md) 仍是引用候选，不构成真实 Runtime/Agent/Host/Release 资格。

**R14 任务级最小知识投影（新候选，未通过独立文档审核）**：[K02/K03/K05 提交状态与故障可观察性跨模块投影](../professional-crash-observability-projection-r14.md)和[仅在真实持久变更风险命中时启用的 L2 知识卡](../professional-l2-crash-observability-card-r14.md)。**K02 独占架构选择/状态 Owner**，K03/K05 按实现/验证交接，不把本卡当强制架构或新 Rule。

**R15 R4 限定证据的架构阶段知识选择（新候选，未独立审核）**：[按真实持久化/并发/外部 Effect 风险选用 K02→K03→K05](../professional-risk-gated-knowledge-selection-r15.md)；#545 后置非盲源码核对发现 T2-A 首次 L2 的 `CORRUPT`/ `CONFLICT` 误分类确有合同依据，但 n=1/组且原 QA 部分无效，**不能推断知识输入的因果优势**。K02 仍只负责架构、状态 Owner/提交点，不新增默认状态机。

**R18 架构质量知识（新增候选，G1 已独立接受并合并）**：[可证伪质量属性场景、依赖和 Owner 结构缺陷、反过度设计评价](advanced-scenario-driven-architecture-quality-r18.md)；从真实需求/环境/响应衡量结构质量，而非从类、层或 ADR 篇数打分。结合 [跨模块 R18 最小质量知识选择](../professional-quality-risk-knowledge-selection-r18.md)，不改变 K02 的架构 Owner 或已有 Source 资格。

**R19 工程抽象专题（新知识候选，G-A 已独立接受并合并）**：[从不变量、变化轴与信息隐藏选择模块职责边界](advanced-engineering-abstraction-boundaries-and-variation-r19.md)。不是接口/模块越多越好；以实际状态 Owner、错误语义和变化隔离反证抽象不足、错误抽象及过度抽象；知识选择参阅 [R19 最小专业入口](../professional-engineering-abstraction-selection-r19.md)。参见 [R19 工程抽象正反对照案例库](../professional-engineering-abstraction-contrastive-casebook-r19.md)（新增候选，G-A2 待审），不修改 R18 冻结实验。

**先了解整个专业学科，再按需选择问题**： [K02 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| [`K02.QUALITY_TRADEOFF`](topics/QUALITY_TRADEOFF.md) | 两个架构方案如何权衡可靠性与成本？ | 质量场景/目标/替代方案 | K02→K10/K11 | Demo≠生产合格 |
| [`K02.STATE_OWNER`](topics/STATE_OWNER.md) | 跨组件业务状态由谁唯一写入？ | 事务/共享状态/提交点 | K02→K03 | 目录结构不等于数据 Owner |
| `K02.INTERFACE_CONTRACT` | 多模块应先稳定哪些接口与错误语义？ | 消费者/Schema/错误合同 | K02→K03/K05 | 不能用图代替合同 |
| `K02.ARCHITECTURE_DEMO` | 何时值得做可证伪 Demo？ | 真实未知/可区分实验 | K02→K05 | 不要对低风险改动强制 ATAM |
| `K02.ARCHITECTURE_EVOLUTION` | 架构变化如何保护旧接口？ | 现有消费者与版本窗口 | K02→K09 | 不能把破坏性变更叫重构 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前已优先深化 2 个容易产生工程误判的专题：[`K02.QUALITY_TRADEOFF`](topics/QUALITY_TRADEOFF.md)、[`K02.STATE_OWNER`](topics/STATE_OWNER.md)。其余问题仍可通过上表定位既有 [工程方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例库](casebook.md) 和 [来源](sources.md)，不为凑数量强制拆文件。新增专题仍为设计候选，不是实际 Agent 已完成资格证明。

## 使用导航

| 文档 | 具体内容与消费时机 |
| --- | --- |
| [systematic-syllabus.md](systematic-syllabus.md) | 本模块系统专业学科：五个知识群、15 个概念/机制/反例 |
| [disciplinary-foundations.md](disciplinary-foundations.md) | SWEBOK/ISO/专业学科方法与具体 Agent 决策、适用反例 |
| [external-practices.md](external-practices.md) | 工程实践项目、操作步骤及可证伪任务样例 |
| [domain-model.md](domain-model.md) | 领域概念、不变量、语义关系与错误解释；第一次判断问题时读 |
| [engineering-methods.md](engineering-methods.md) | 工程方法与决策矩阵；需要比较方案/选择手段时读 |
| [agent-skills.md](agent-skills.md) | Agent Skills、输入输出合同与直接可用提示词；真实 Task 时读 |
| [casebook.md](casebook.md) | 贯穿案例、变体、反模式；遇到相似问题或错误时读 |
| [validation.md](validation.md) | 反例、测验与未来合格证据；需要证明适用性时读 |
| [sources.md](sources.md) | 知识→ADS 具体条款、规范/候选边界和接入条件；决定知识授权时读 |

## 能力检查问题

1. 真实架构驱动是否支持新增模块或服务？
2. 接口、数据与状态的唯一写入责任如何定义？
3. 需要何种最小可证伪 Demo 才能关闭关键未知？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 本次改动可复用现有模块与合同 | 记录 INLINE ADR 或设计备注，复用已有接口 | 边界+契约+最小验证 |
| 状态所有权存在冲突 | 停止并定义单一写入 Owner 与一致性范围 | 状态责任/失败表 |
| 两方案真正影响部署或可靠性 | 同维度比较并设计最小 Demo | ADR、风险、证伪标准 |
| 涉及版本升级数据迁移 | 引用 K09 建兼容/迁移路线 | 合同版本和恢复要求 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。