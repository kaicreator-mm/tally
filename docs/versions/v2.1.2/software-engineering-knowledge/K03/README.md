# K03 软件工程知识库：详细设计与代码实现

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #524](https://github.com/kaicreator-mm/tally/issues/524) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

把已冻结的接口和行为约束变成可维护、可测试、可恢复的当前 Task 实现。

## 使用导航

| 文档 | 具体内容与消费时机 |
| --- | --- |
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