# K10 · 可运维性、可靠性与性能 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #531](https://github.com/kaicreator-mm/tally/issues/531) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

为不同应用形态设计可行动日志/错误、可靠失败与恢复，并将性能判断绑定真实基线。

## 目录和何时使用

| 知识资源 | 为什么要读 |
| --- | --- |
| [domain-model.md](domain-model.md) | 区分工程概念、事实来源及容易混淆的语义 |
| [engineering-methods.md](engineering-methods.md) | 根据具体条件选择方法、方案与有界 Decision Procedure |
| [agent-skills.md](agent-skills.md) | 供 Tally 选择的 Agent Skill、单次可复制提示词和拒绝条件 |
| [casebook.md](casebook.md) | 正反案例、变体、反模式和实际工程决策案例 |
| [validation.md](validation.md) | 负例、知识适用性训练、未来真实消费者效用验收 |
| [sources.md](sources.md) | ADS 具体条款、有效版本、候选/规范和权限边界 |

## 三个优先问题

1. 用户和维护者能否判断失败发生了什么、下一步怎么做？
2. 超时与重试是否可能产生重复业务副作用？
3. 有无同条件性能基线支持实际优化结论？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 交互桌面工具尚未设置仓库 | 允许启动配置 UI，禁止未授权业务操作 | 首次启动正负场景 |
| CLI/后台服务配置错误 | 按无头形态输出明确的配置诊断 | 退出码/日志或管理 API |
| 超时后状态未知 | 先查询结果并选择有界安全重试 | UNKNOWN/幂等/恢复合同 |
| 没有固定负载基线 | 先定义测量方法；否则 NOT_MEASURED | 性能测试计划 |

**选用规则**：不要求每次调用全部 6 篇文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。