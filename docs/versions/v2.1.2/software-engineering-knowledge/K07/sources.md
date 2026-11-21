# K07 · Source and Claim Map / ADS 依据、候选与资格

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #528](https://github.com/kaicreator-mm/tally/issues/528) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 原始跨模块关系与来源

## 10. 可追溯 ADS 来源与知识合格条件

**规范及实践来源候选**（正式采用须查验对应有效版本和 Exact SHA）：
- [ADS GitHub Agent Interaction Protocol](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/GITHUB_AGENT_INTERACTION_PROTOCOL.md)
- [ADS Local Agent Handoff](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/LOCAL_AGENT_HANDOFF_PROTOCOL.md)
- [ADS Execution Architecture](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/EXECUTION_ARCHITECTURE_STANDARD.md)
- [ADS #1199 interaction intake](https://github.com/kaicreator-mm/ai-development-standard/issues/1199)
- [ADS #1383 source candidate](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)

**Tally 现有可复用所有者**：[Engineering Business/Workflow #276](https://github.com/kaicreator-mm/tally/issues/276)、[Package 消费与资格候选 #286](https://github.com/kaicreator-mm/tally/issues/286)、[经验→候选 #60](https://github.com/kaicreator-mm/tally/issues/60)。[v2.1.1 知识实验 #299](https://github.com/kaicreator-mm/tally/issues/299) 的实际终局是 `KNOWLEDGE_ASSET_NOT_QUALIFIED`；不能把 Pending 知识自动晋升为 Tally 生产 Rule/Skill/Workflow。

**候选内容类型**：Concept/Knowledge、DecisionProcedure、Skill、Case/Counterexample、Validation、现有 Workflow 的引用。真正生效的硬规则仍由已有 ADS 规范/权限/验证 Owner 负责。必须校验来源是否支持结论、适用条件、失败/反例、版权和版本、独立审查以及至少一个真实受影响工程消费者的行为，不能简单统计知识条目数。

**文档完成的定义**：概念清晰、方法具体、具有正反例/决策停止条件、定位真实 Tally 消费点、来源可追溯；**并不等于**版本 Product/L2 Freeze、Task Claim、Domain Package 生产合格或真实 Agent 开发/交付通过。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 角色与独立审查 | `standards/GITHUB_AGENT_INTERACTION_PROTOCOL.md` → §5 `Actor role and logical operator identity`、§6–7 `Independent Review` | 已发布 GitHub Profile 规范 | 不同会话独立性看实际 Writer/Reviewer 来源，不用账号替代 |
| 有界执行交接 | `standards/LOCAL_AGENT_HANDOFF_PROTOCOL.md` → §2 `Pointer-only principle`、§3 `Machine-verifiable handoff` | 已发布交接规范 | IPC/HTTP 消息只是通道，不自动产生 Claim |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。




## 复用/晋升边界

- ADS v4.19.0 的规范与 Tally v2.1.2 的建议知识不是同一权威；只复用适用的已发布规则。
- 历史 #1100 和未合并 #1384 为候选设计，不自动成为规范。
- Tally #299 原有 806 条实验图谱记录及 Pending Claim 仍 `KNOWLEDGE_ASSET_NOT_QUALIFIED`，不能整库自动导入。
- 用户暂缓的逆向分析和安全供应链专项不得悄悄成为当前知识库实施依赖。
- 真实 Package 消费须由相应版本 Product/L2/Task/Review/Host 权威另行安排；本知识库只提供可追溯方法、示例、反例与预期验证。