# K04 软件工程知识库：任务规划与分解

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #525](https://github.com/kaicreator-mm/tally/issues/525) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

选择最小可独立验收的工作单位与真实依赖，使并行收益大于协调成本。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：任务依赖图与关键路径](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：估算不确定性与迭代重规划](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R4 逐点学科缺口补齐（独立 ACCEPT_SCOPED @ 42e7ba8bd883ec79736e25ff7c75bdbda1b738b3，旧 SHA 限定文档资格）**：[`K04.F2.3` Task 完成证据/授权](focus-f2-3-task-acceptance-and-authority.md)。这些文件是对课程的定点专业深化，包含机制/条件/反例/STOP；并非获权 Source Claim、CI/Host/Package 或产品发布资格。

**R6 进阶原理与专业反例（旧 Exact SHA `83ebee092db412b5e38db0376562ee51b0d4f032` 已获独立 ACCEPT_SCOPED，文档范围）**：[真实因果 DAG、共享写集和关键路径的专业演算](advanced-task-graph-critical-path-and-bounded-delivery.md)。对本模块的选定课程 ID 补充可验证的机制、适用前提、替代选项和否证轨迹；相应 [W06–W12 跨模块完整演算](../professional-reasoning-worked-traces-02.md) 均为教学而非实际运行。无 Source Claim、DomainHarness ABI 或生产 Agent/Host 权限资格。

**先了解整个专业学科，再按需选择问题**： [K04 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| `K04.ATOMIC_TASK` | 工作是否可以由一个任务完整交付？ | 用户结果/代码及测试边界 | K04→K03 | 禁止按文件机械拆 Issue |
| [`K04.REAL_DEPENDENCY`](topics/REAL_DEPENDENCY.md) | BLOCKED_BY 是否是不可替代的真实前置？ | 前置产物/反事实分析 | K04→K08 | CONSUMES_EVIDENCE 不是 BLOCKED_BY |
| [`K04.PARALLEL_WRITES`](topics/PARALLEL_WRITES.md) | 多个 Agent 会不会冲突修改共享资源？ | 写集/稳定接口/Owner | K04→K07 | 多工作树不等于独立写集 |
| `K04.ADAPTIVE_SHAPE` | 简单 Task 是否需要显式 DAG？ | 复杂度/后果/真正的依赖 | K04→K11 | 三 DAG 统一规范未采纳 |
| `K04.PLAN_REPAIR` | 范围变动后如何最小重规划？ | 变更的当前事实/影响节点 | K04→K09 | 不全量重开旧任务 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前已优先深化 2 个容易产生工程误判的专题：[`K04.REAL_DEPENDENCY`](topics/REAL_DEPENDENCY.md)、[`K04.PARALLEL_WRITES`](topics/PARALLEL_WRITES.md)。其余问题仍可通过上表定位既有 [工程方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例库](casebook.md) 和 [来源](sources.md)，不为凑数量强制拆文件。新增专题仍为设计候选，不是实际 Agent 已完成资格证明。

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

1. 哪个成果可以独立验证且带完整失败处理与测试？
2. 哪条阻塞边具有真实不可替代前置条件？
3. 是不是正在为了更多 Agent 或 Issue 过度拆分？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 单一模块小修复、可当天完整验收 | INLINE 单 Task + 真实必要测试 | 原子任务包 |
| 多模块但接口尚未确定 | 先完成共享契约，再拆合法并行写集 | 合同 Task 与后续依赖 |
| 多项相互依赖的 L1/L2 研究 | 按已发布 ADS 原 owner 做有界映射/研究 | 研究问题和停止条件 |
| 上游仅一个接口发生变化 | 传播到受影响任务和测试 | 最小重规划 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。