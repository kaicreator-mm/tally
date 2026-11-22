# K05 软件工程知识库：软件测试与质量验证

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #526](https://github.com/kaicreator-mm/tally/issues/526) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

从业务契约推导能暴露实现错误的 Oracle，并证明测试被发现且真实执行。

**系统正文第一卷（R2 文档范围独立 ACCEPT_SCOPED @ ede9d646cb591b8df5179dc6bc11b60668fe968c）**：[学科深化 01：测试 Oracle 与性质验证](foundations-deep-dive-01.md)。此章解释术语背后的机制、适用条件、工程取舍与可证伪反例；仍为文档知识候选，不是已合格 Source Claim 或真实 Harness/Agent 功能。

**系统正文第二卷（R3 文档范围独立 ACCEPT_SCOPED @ a39a64a82beae103318da3e45bd09b5efed15ad1）**：[学科深化 02：组合覆盖、变形与状态空间](foundations-deep-dive-02.md)。本卷补充基础理论、适用条件、备选技术与可证伪反例；不宣称 165 点均已教材级展开，也不自动取得生产 Source/Consumer 资格。

**R4 逐点学科缺口补齐（独立 ACCEPT_SCOPED @ 42e7ba8bd883ec79736e25ff7c75bdbda1b738b3，旧 SHA 限定文档资格）**：[`K05.F3.3` Build/Package/平台的验证层](focus-f3-3-build-package-platform-verification.md)；[`K05.F4.2` Golden/Snapshot 权威与时效](focus-f4-2-golden-snapshot-evidence-freshness.md)。这些文件是对课程的定点专业深化，包含机制/条件/反例/STOP；并非获权 Source Claim、CI/Host/Package 或产品发布资格。

**R5 进阶原理与完整演算（旧 SHA 已独立 ACCEPT_SCOPED @ 041949b409fd7772fd50fac6ad3fef8190ae5f9e，文档范围）**：[独立 Oracle、测试充分性与最小区分性反例](advanced-oracle-adequacy-and-test-selection.md)。围绕已有学科原理推导因果、方案比较、明确反例、外部来源和不可假定的权限；不是 Source-qualified Rule/Producer/Agent 的当前实现。

**R7 专业基础原理补缺（旧 Exact SHA `a307e0fab92ae3b857b5098d16200b339ffda764` 已获独立 ACCEPT_SCOPED，文档范围）**：[V&V/Oracle 来源、测试层次和 Flaky 区分](advanced-verification-validation-and-oracle-qualification.md)，对本模块既有课程中的 **6 个之前没有进阶 A 的知识点**提出受限规格/推导、专业反例与真正需要的 Oracle；不是被证明的运行时 Claim。详见 [R7 原 87 点差距盘点](../professional-depth-gap-inventory-r7.md)，避免按文件数量推断学科完全合格。

**R13 R1/R2 实验反馈驱动的限定补充（已通过限定独立文档审核；DOCUMENT_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED）**：[从工程不变量推导竞争/旧行为/损坏负例以及 FR×Oracle 直接断言账本](advanced-invariant-derived-discriminating-oracles-r13.md)。对应 K05.F2.3/F5.1/F4.1，避免把更多测试/绿色统计等同于更高区分力；明确 `NOT_INDEPENDENTLY_VERIFIED` 和可复算 Test Discovery。仍未取得 R2 原始 QA 独立复核资格。

**R14 R3 复核反馈的 Oracle 交接（新候选，未通过独立文档审核）**：[持久状态可观察性投影与提交切点反例](../professional-crash-observability-projection-r14.md)及[阶段限定 L2 卡](../professional-l2-crash-observability-card-r14.md)。仅在实际选定故障模型下证明执行过的 Test/Oracle；R3 #539 的六项预登记未测面不得随新文档晋升为 PASS。

**先了解整个专业学科，再按需选择问题**： [K05 系统课程：5 个知识群 / 15 个专业知识点](systematic-syllabus.md) 用于完整概念、原理和常见反例的持续积累；下表是每次任务的有界检索入口，并非该模块的全部知识。课程内容尚未得到 Source Claim/Domain Package 生产资格。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| [`K05.ORACLE`](topics/ORACLE.md) | 业务规则如何成为能发现错误的测试 Oracle？ | Requirement/Invariant/Source | K05→K01 | 没有需求来源不能造 Expected |
| `K05.TEST_SELECTION` | 需要 Unit、Contract、Integration 还是 E2E？ | 受影响行为/风险/环境 | K05→K08 | 不要求每改动全量 CI |
| [`K05.FALSE_PASS`](topics/FALSE_PASS.md) | CI 绿但测试没发现或仅 Mock？ | 当前 SHA/runner/测试发现/调用入口 | K05→K08 | 0 tests 不代表功能 PASS |
| [`K05.INTEGRATION`](topics/INTEGRATION.md) | 消费者契约与真实 DB 副作用是否都被证明？ | Consumer/Provider/真实数据依赖 | K05→K02/K08 | Pact/Testcontainers 不等于全产品 E2E |
| `K05.FLAKY_TEST` | 测试失败是时间不确定性还是实现缺陷？ | 时钟/事件/受权时限 | K05→K06 | 无界重试不能修正 Oracle |
| `K05.EVALUATION` | Agent 补丁通过基准是否代表用户产品合格？ | Issue/Patch/Test/用户路径 | K05→K11 | SWE-bench PASS 不等于 Release |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前已优先深化 3 个容易产生工程误判的专题：[`K05.ORACLE`](topics/ORACLE.md)、[`K05.FALSE_PASS`](topics/FALSE_PASS.md)、[`K05.INTEGRATION`](topics/INTEGRATION.md)。其余问题仍可通过上表定位既有 [工程方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例库](casebook.md) 和 [来源](sources.md)，不为凑数量强制拆文件。新增专题仍为设计候选，不是实际 Agent 已完成资格证明。

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

1. 哪个来源决定 expected 行为？
2. 哪些负例和真实消费者能证明目标不变量？
3. CI GREEN 是否实际上覆盖当前受影响代码？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 没有可信业务规则但 Agent 想写期望 | 先回 K01 确认 Oracle 的真实来源 | 规则 UNKNOWN |
| 新接口只在 Mock 上验证 | 增加真正公开入口的契约/必要真实集成路径 | 受约束执行计划 |
| 0 个测试被发现但 runner 退出 0 | 标为发现缺陷，修正有效测试选择 | test discovery 收据 |
| 仅 docs 改动没有运行时影响 | 复用合法当前证据，跳过无关重测 | 跳过原因与范围 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。