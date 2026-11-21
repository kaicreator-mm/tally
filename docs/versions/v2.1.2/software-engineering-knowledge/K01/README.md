# K01 软件工程知识库：需求工程与业务建模

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #522](https://github.com/kaicreator-mm/tally/issues/522) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

把用户目标转换为受权、可检查的业务需求，保持事实、假设与未知分离。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：业务行为与受权验收](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：需求采样偏差与质量目标](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R5 进阶原理与完整演算（旧 SHA 已独立 ACCEPT_SCOPED @ 041949b409fd7772fd50fac6ad3fef8190ae5f9e，文档范围）**：[需求证据→合法合同→并发验收的完整推理](advanced-knowledge-from-evidence-to-contract.md)。围绕已有学科原理推导因果、方案比较、明确反例、外部来源和不可假定的权限；不是 Source-qualified Rule/Producer/Agent 的当前实现。

**R7 专业基础原理补缺（旧 Exact SHA `a307e0fab92ae3b857b5098d16200b339ffda764` 已获独立 ACCEPT_SCOPED，文档范围）**：[需求双向追溯、领域不变量和变化影响](advanced-requirements-traceability-and-change-propagation.md)，对本模块既有课程中的 **4 个之前没有进阶 A 的知识点**提出受限规格/推导、专业反例与真正需要的 Oracle；不是被证明的运行时 Claim。详见 [R7 原 87 点差距盘点](../professional-depth-gap-inventory-r7.md)，避免按文件数量推断学科完全合格。

**R8 深化工程判断专业基础（新章待限定独立审核）**：[Stakeholder 权威冲突与证据三角核验](advanced-stakeholder-authority-and-triangulated-evidence.md)，精确连接本模块 **2 个原有高优先级课程 ID**，明确 Source/范围、机制假设、负例与 STOP；[W13–W17](../professional-reasoning-worked-traces-03.md) 展示综合选择，[来源适用矩阵](../professional-source-applicability-r8.md) 仍是引用候选，不构成真实 Runtime/Agent/Host/Release 资格。

**先了解整个专业学科，再按需选择问题**： [K01 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| `K01.GOAL` | 用户的功能请求是否真的表达了业务需要？ | 需求原意/使用者/价值 | K01→K02 | 不要把方案代替需求 |
| [`K01.RULE_CONFLICT`](topics/RULE_CONFLICT.md) | 两个需求来源对同一状态的处理冲突？ | 来源版本/适用角色/有效规则 | K01→K05 | 无有权来源则 UNKNOWN |
| [`K01.ACCEPTANCE`](topics/ACCEPTANCE.md) | 怎样把规则变成可观察的正反验收？ | 角色×状态×动作矩阵 | K01→K05 | 未获批准的期望不能充当 Oracle |
| `K01.QUALITY_REQUIREMENT` | 性能/可靠性要求是否有真实可验证场景？ | 质量驱动/使用环境/观察指标 | K01→K02/K10 | 不得伪造 SLO |
| `K01.CHANGE_IMPACT` | 需求修订影响哪些合同与旧消费者？ | 变更范围/不变约束 | K01→K09 | 不自动扩大所有模块工作 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前已优先深化 2 个容易产生工程误判的专题：[`K01.RULE_CONFLICT`](topics/RULE_CONFLICT.md)、[`K01.ACCEPTANCE`](topics/ACCEPTANCE.md)。其余问题仍可通过上表定位既有 [工程方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例库](casebook.md) 和 [来源](sources.md)，不为凑数量强制拆文件。新增专题仍为设计候选，不是实际 Agent 已完成资格证明。

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

1. 谁提出需求、由谁实际使用、真实成功条件是什么？
2. 哪些业务规则与不变量必须经 Product Owner 确认？
3. 什么时候能把研究交给 L2，而不是持续调研？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 原始需求已有明确角色、状态和验收 | INLINE：复述范围并构造可观察验收 | Requirement+Acceptance |
| 两个权威来源对同一业务规则冲突 | 定位适用版本/角色，列选项交有权决策者 | CONFLICT/UNKNOWN+影响范围 |
| 需要多个依赖性需求访谈/实验 | 先做当前 ADS 允许的按需 Evidence DAG/轻量映射；统一 DAG-P 为候选 | 研究问题和真实前置 |
| 需求变更仅影响一个业务状态 | 只传播关联 Rule→Design→Test | 受影响列表和未变更证明 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。