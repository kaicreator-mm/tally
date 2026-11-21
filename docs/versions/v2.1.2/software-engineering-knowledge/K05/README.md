# K05 软件工程知识库：软件测试与质量验证

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #526](https://github.com/kaicreator-mm/tally/issues/526) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

从业务契约推导能暴露实现错误的 Oracle，并证明测试被发现且真实执行。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| `K05.ORACLE` | 业务规则如何成为能发现错误的测试 Oracle？ | Requirement/Invariant/Source | K05→K01 | 没有需求来源不能造 Expected |
| `K05.TEST_SELECTION` | 需要 Unit、Contract、Integration 还是 E2E？ | 受影响行为/风险/环境 | K05→K08 | 不要求每改动全量 CI |
| `K05.FALSE_PASS` | CI 绿但测试没发现或仅 Mock？ | 当前 SHA/runner/测试发现/调用入口 | K05→K08 | 0 tests 不代表功能 PASS |
| `K05.INTEGRATION` | 消费者契约与真实 DB 副作用是否都被证明？ | Consumer/Provider/真实数据依赖 | K05→K02/K08 | Pact/Testcontainers 不等于全产品 E2E |
| `K05.FLAKY_TEST` | 测试失败是时间不确定性还是实现缺陷？ | 时钟/事件/受权时限 | K05→K06 | 无界重试不能修正 Oracle |
| `K05.EVALUATION` | Agent 补丁通过基准是否代表用户产品合格？ | Issue/Patch/Test/用户路径 | K05→K11 | SWE-bench PASS 不等于 Release |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 使用导航

| 文档 | 具体内容与消费时机 |
| --- | --- |
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