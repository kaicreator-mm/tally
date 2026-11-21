# K07 · 多 Agent 工程协作 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #528](https://github.com/kaicreator-mm/tally/issues/528) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

让不同 Agent 在真实角色、消息、执行、候选、审查和权限边界下协作，不让通信 ACK 冒充正式 Task/Claim/Release。

## 目录和何时使用

| 知识资源 | 为什么要读 |
| --- | --- |
| [disciplinary-foundations.md](disciplinary-foundations.md) | 学科理论、可应用的工程方法与误推断 |
| [external-practices.md](external-practices.md) | 企业/GitHub 真实工程参考与可证伪案例 |
| [domain-model.md](domain-model.md) | 区分工程概念、事实来源及容易混淆的语义 |
| [engineering-methods.md](engineering-methods.md) | 根据具体条件选择方法、方案与有界 Decision Procedure |
| [agent-skills.md](agent-skills.md) | 供 Tally 选择的 Agent Skill、单次可复制提示词和拒绝条件 |
| [casebook.md](casebook.md) | 正反案例、变体、反模式和实际工程决策案例 |
| [validation.md](validation.md) | 负例、知识适用性训练、未来真实消费者效用验收 |
| [sources.md](sources.md) | ADS 具体条款、有效版本、候选/规范和权限边界 |

## 三个优先问题

1. 单 Agent 完成是否已经比并行更划算？
2. 消息与持久证据的身份和 ACK 如何对应？
3. Writer/Reviewer 是否真的在独立工作上下文中处理同一 HEAD？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 只有一个局部 Task 和一个共享代码区域 | 单 Builder 处理，独立审查适用时再单独执行 | 最小角色计划 |
| 多个独立只读研究问题 | 可并行分析，经 Owner 汇总证据 | 分工与同源事实 |
| IPC/HTTP 已返回 ACK | 只记录消息交付并读回正式 Task/Host | Ack+Source/Attempt 对账 |
| 候选 Head 已变化 | 迟到 Worker 结果标 STALE 并重核来源 | 阻止旧 Artifact 覆盖新结果 |

**选用规则**：不要求每次调用全部 6 篇文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。