# K03 软件工程知识库：详细设计与代码实现

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #524](https://github.com/kaicreator-mm/tally/issues/524) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

把已冻结的接口和行为约束变成可维护、可测试、可恢复的当前 Task 实现。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：算法与实现正确性](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：类型不变量、资源生命周期与背压](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R5 进阶原理与完整演算（旧 SHA 已独立 ACCEPT_SCOPED @ 041949b409fd7772fd50fac6ad3fef8190ae5f9e，文档范围）**：[并发线性化、幂等与崩溃重放的反例](advanced-concurrency-idempotency-proof-sketch.md)。围绕已有学科原理推导因果、方案比较、明确反例、外部来源和不可假定的权限；不是 Source-qualified Rule/Producer/Agent 的当前实现。

**R7 专业基础原理补缺（旧 Exact SHA `a307e0fab92ae3b857b5098d16200b339ffda764` 已获独立 ACCEPT_SCOPED，文档范围）**：[算法前/后条件、类型系统和形式证明局限](advanced-algorithm-contracts-types-and-formal-claims.md)，对本模块既有课程中的 **5 个之前没有进阶 A 的知识点**提出受限规格/推导、专业反例与真正需要的 Oracle；不是被证明的运行时 Claim。详见 [R7 原 87 点差距盘点](../professional-depth-gap-inventory-r7.md)，避免按文件数量推断学科完全合格。

**R9 专业基础及可证伪失效机制（新增待限定独立审核）**：[错误可行动语义、隐私日志与受影响测试真实执行](advanced-error-semantics-observability-and-validation-proof.md)，对原系统课程中 **3 个历史高优先知识点**新增准确 A 专业正文关联；[W18–W22](../professional-reasoning-worked-traces-04.md) 是合成推理，[Source→Claim→Oracle 准入差距](../professional-source-to-claim-readiness-r9.md) 明确保留无真实权限/未运行证据。不是实际 Source Claim、Consumer ABI、Agent/Host/Release 资格。

**R13 R1/R2 实验反馈驱动的限定补充（已通过限定独立文档审核；DOCUMENT_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED）**：[本地文件并发写入、提交收敛、恢复与静止性](advanced-concurrent-local-file-write-convergence-r13.md)。对应 K03.F3.1/F3.2/F4.2，在状态不变量之下明确条件创建、持久化与 ACK 的不同保证、竞争者回读/临时文件资源和跨平台边界；不将 Node `wx` 误称为完整事务或耐掉电保证。Opaque 标识符的合法错误语义仍由 [K01 公开合同](../K01/advanced-opaque-identifier-contract-and-error-precedence-r13.md) 决定。

**R14 实现前的有界投影（新候选，未通过独立文档审核）**：[从 K02 的真实可持久状态序列到 K03 实现与 K05 Oracle 的知识交接](../professional-crash-observability-projection-r14.md)，最小 [L2 知识卡](../professional-l2-crash-observability-card-r14.md)。本模块仍负责具体文件/数据库实现机制，不因 R3 的单个文件协议例子强制推广 `wx` 或跨文件 `rename`。

**R15 R4 限定证据的实现/测试交接（新候选，未独立审核）**：[按风险选择 K02/K03/K05 专业知识](../professional-risk-gated-knowledge-selection-r15.md)及 [K05 测试夹具与 Oracle 有效性](../K05/advanced-test-fixture-and-oracle-validity-r15.md)。R4 T2-A 首次 L2 写序列误分类与首次 S5 实现自修正应**分别记账**；测试对同步/异步资源清理、顺序版本错误和时间窗口的未验证面不等于实现已具合格证据。

**R18 代码可理解性知识（新增候选，G1 已独立接受并合并）**：[陌生 Reader 的行为解释、导航、抽象成本与隐藏副作用](advanced-code-comprehension-and-abstraction-quality-r18.md)；不以行数、注释量、短函数或静态指标单独判代码可读性。真实维护效用继续交由 [K09](../K09/advanced-stranger-maintainer-change-assay-r18.md) 验证，知识阶段选择参阅 [R18](../professional-quality-risk-knowledge-selection-r18.md)。

**R19 工程抽象专题（新知识候选，G-A 已独立接受并合并）**：[何时内联、局部 Helper、领域模块、窄公共接口与可替换策略](advanced-abstraction-level-selection-and-semantic-reuse-r19.md)。以共同不变量、变化轴、错误/资源语义和真实成本判断复用；不得将设计模式词汇或代码行数作为抽象正确性证据。参阅 [R19 知识选择](../professional-engineering-abstraction-selection-r19.md) 与 [正反对照案例库](../professional-engineering-abstraction-contrastive-casebook-r19.md)（G-A2 待审）。

**先了解整个专业学科，再按需选择问题**： [K03 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| `K03.TASK_DESIGN` | 当前 Task 如何转化为可实现的输入/状态/错误？ | 合法 Task/设计/写集 | K03→K05 | 设计建议不构成 Builder 许可 |
| [`K03.ASYNC_IDEMPOTENCY`](topics/ASYNC_IDEMPOTENCY.md) | 请求超时或重复时如何保证副作用语义？ | 提交点/重复请求/状态回读 | K03→K10 | 超时不证明未提交 |
| [`K03.ALGORITHM_CHOICE`](topics/ALGORITHM_CHOICE.md) | 应该选择哪种数据结构与算法？ | 真实数据规模/复杂度/平台 | K03→K11 | 不以大框架解决小问题 |
| `K03.ERROR_LOGGING` | 错误与日志怎样可行动且保护敏感信息？ | 用户角色/错误类别/恢复路径 | K03→K10 | 堆栈不是用户诊断 |
| `K03.IMPLEMENTATION_EVIDENCE` | 代码与受影响测试是否在当前 HEAD 真运行？ | 工具链/测试发现/来源 SHA | K03→K05/K08 | 本地 PASS 不等于 Release |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前已优先深化 2 个容易产生工程误判的专题：[`K03.ASYNC_IDEMPOTENCY`](topics/ASYNC_IDEMPOTENCY.md)、[`K03.ALGORITHM_CHOICE`](topics/ALGORITHM_CHOICE.md)。其余问题仍可通过上表定位既有 [工程方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例库](casebook.md) 和 [来源](sources.md)，不为凑数量强制拆文件。新增专题仍为设计候选，不是实际 Agent 已完成资格证明。

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

1. 输入输出、状态、不变量和非法请求行为是什么？
2. 哪种算法和数据结构足以满足当前规模？
3. 当前修改是否有实际编译/类型/测试证据？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 简单、无公开契约变化的局部修复 | 一个原子 Task，实施与受影响测试同 Writer | Patch+真实收据 |
| 并发请求可能重复造成副作用 | 采用已有幂等键或状态检查并定义 UNKNOWN 回读 | 状态转移+重复调用负例 |
| 输入规模较大、需要流式处理 | 比较内存上界/时间复杂度和实际负载 | 算法理由/性能与行为测试 |
| 现有 L2 合同无法表达部分成功 | 暂停该设计冲突并交 Owner | 合法设计修订请求 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。