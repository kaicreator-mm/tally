# K09 · 软件演进与重构 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #530](https://github.com/kaicreator-mm/tally/issues/530) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

在维护已支持行为的前提下做小步重构、兼容接口升级和数据迁移，避免用新测试掩盖旧消费者回归。

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

1. 当前变化属于 Bug/Refactor/Feature/Migration/Deprecation 哪一种？
2. 哪些旧用户、客户端或数据格式仍受支持？
3. 数据迁移中断/升级回滚后系统保持什么不变量？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 仅内部函数拆分，公开行为保持 | 先建立旧行为 Oracle 后做小步重构 | 前后兼容证据 |
| API 需要分页但旧客户端只认数组 | 用版本/可选字段或新端点保护旧合同 | 旧新 consumer 两臂测试 |
| 迁移涉及不可逆数据删除 | 准备备份、读回、补偿/受权决策 | 风险和恢复计划 |
| 旧代码只是风格不漂亮且无实际成本 | 允许延期，聚焦当前交付 | 债务候选及观察点 |

**选用规则**：不要求每次调用全部 6 篇文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。