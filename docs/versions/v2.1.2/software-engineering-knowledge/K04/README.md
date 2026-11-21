# K04 软件工程知识库：任务规划与分解

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #525](https://github.com/kaicreator-mm/tally/issues/525) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

选择最小可独立验收的工作单位与真实依赖，使并行收益大于协调成本。

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