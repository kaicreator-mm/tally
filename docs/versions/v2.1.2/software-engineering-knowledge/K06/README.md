# K06 软件工程知识库：调试与根因修复

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #527](https://github.com/kaicreator-mm/tally/issues/527) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

用最早可观察失败事实、竞争解释和最小复现定位因果根因，避免无界修复循环。

## 当前工程问题索引（优先入口）

先根据当前 Task 的**真实问题**命中下表最小 Topic，再阅读与之相关的 [专业方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例](casebook.md)、[来源](sources.md)；不要把九个专题文件全部塞进一次 Agent 上下文。**Topic ID 是文档检索标识，不是新增的 Tally Wire Schema/Task Status。** 跨模块列仅表示引用知识及交接位置，不改变原有工程权威。

| Topic ID | 当前问题/触发 | 最小必要输入 | 知识交接 | 必须拒绝的错误 |
| --- | --- | --- | --- | --- |
| [`K06.FIRST_FAILURE`](topics/FIRST_FAILURE.md) | 当前执行究竟哪个步骤最早失败？ | 真实日志/阶段/源码 SHA | K06→K08 | 退出码不是根因 |
| [`K06.CAUSE_PROBE`](topics/CAUSE_PROBE.md) | 如何从多个根因假设找到最小探针？ | 可区分解释/复现/环境 | K06→K03 | 先实验再修复 |
| `K06.RETRY_OR_RECOVER` | 网络失败是否可以安全重试？ | 外部副作用/幂等/读回 | K06→K10 | 不可盲重放操作 |
| `K06.REPAIR_BOUNDS` | 多轮重复失败何时停止/升级？ | 已有尝试/新增证据/Owner | K06→K04 | 不得无限 Agent 循环 |
| `K06.INCIDENT_LEARNING` | 真实事故如何转化为有效经验候选？ | 故障/恢复/验证/来源 | K06→K11 | 恢复不倒改旧失败 |

**选用顺序：** 真实任务/版本/角色 → Topic → 相关 DP/Skill/正反例 → 当前来源/适用条件 → 原有 Validation/Task/Host 资格。文档只提供分析方法和停止条件。模块内容唯一维护在现有专题文件，Topic 行不复制其全文。

## 已深化的问题级专题

当前已优先深化 2 个容易产生工程误判的专题：[`K06.FIRST_FAILURE`](topics/FIRST_FAILURE.md)、[`K06.CAUSE_PROBE`](topics/CAUSE_PROBE.md)。其余问题仍可通过上表定位既有 [工程方法](engineering-methods.md)、[Agent Skills](agent-skills.md)、[案例库](casebook.md) 和 [来源](sources.md)，不为凑数量强制拆文件。新增专题仍为设计候选，不是实际 Agent 已完成资格证明。

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

1. 哪条命令/调用最早失败？
2. 至少两种根因解释如何用最便宜实验区分？
3. 修复后哪个正负例证明根因已消除？

## 如何选择而非加载整个库

在已有 Tally Engineering Business/Task/Host 中，先用当前具体问题和相关源事实定位相应文档；如果现有答案足够，直接结束，不强制读取其他五份文件。示例/Decision Procedure 属 `PROPOSED_KNOWLEDGE`，不是新 schema 或 Gate；真实 Task/Claim/Review/Release 仍由当前权威处理。**特别注意：完整 DAG-P/A/I 与三阶段统一形态为候选，不是已发布 ADS v4.19 必经流程。**

## 快速能力索引

| 用户问题 | 优先知识 | 交付判断 |
| --- | --- | --- |
| 当前 install 成功，首次 build 出 TypeScript 错 | 定位编译器具体源码/合同问题 | root cause hypothesis + scope |
| 网络超时但业务是否提交未知 | 先读回状态和 Provider 证据 | UNKNOWN/恢复路径 |
| 测试用例根本没被发现 | 修复测试选择/布局，非业务逻辑 | test discovery 修复 |
| 多轮同类失败没有新证据 | 停止振荡，报告阻塞与最小新探针 | 有界升级记录 |

## 状态与版本

`DOCUMENTED_KNOWLEDGE_LIBRARY / NOT_QUALIFIED / PROPOSED_NOT_CONNECTED`。这是文档中可消费知识的逻辑库，并非独立数据库、运行时 SDK 或新执行引擎。