# K11 · 工程经济学与决策 — 软件工程领域知识库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #532](https://github.com/kaicreator-mm/tally/issues/532) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

以真实用户交付与质量目标衡量开发活动，选择最小充分研究、任务拆分、Agent 并行和测试验证。

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

1. 额外研究/审查/CI 是否影响关键决策或真实保护底线？
2. 方案的总实施/集成/返工/维护成本是多少，哪些只是估算？
3. Agent 知识辅助是否经真实开发任务公平对照证明有益？

## 快速决策索引

| 触发 | 优先处理 | 期待输出 |
| --- | --- | --- |
| 单一 UI 文案变更，无高风险 | INLINE 一次修复+相关检查 | 最小成本完成 |
| 公共接口具有不可逆迁移风险 | 保留必要设计/负例/恢复/Host 验证 | 风险与保证方案 |
| 多 Agent 并行需共享同一中心文件 | 优先单 Writer/合同后并行只读 | 总集成成本对比 |
| Tally 知识试验只在 7 组问答得分好 | 限定知识问题观察，另设计真实编码任务对照 | 研究级别效用报告 |

**选用规则**：不要求每次调用全部 6 篇文档；依据当前任务所需决策选择最小知识切片。K09–K11 是条件性消费，K07/K08 在首批真实工程协作/交付需要时才选用。Tally 既有 Engineering Business/Host 事实源继续有效；本知识库为 `PROPOSED_NOT_CONNECTED`，不是新的知识服务/数据库或独立执行权限。