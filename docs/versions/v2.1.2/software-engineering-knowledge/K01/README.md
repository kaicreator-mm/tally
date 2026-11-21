# K01 软件工程知识库：需求工程与业务建模

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #522](https://github.com/kaicreator-mm/tally/issues/522) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

把用户目标转换为受权、可检查的业务需求，保持事实、假设与未知分离。

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