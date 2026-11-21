# K02 软件工程知识库：架构与系统设计

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #523](https://github.com/kaicreator-mm/tally/issues/523) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

以需求驱动、状态/接口合同、替代方案和失效语义作出可解释的架构决策。

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