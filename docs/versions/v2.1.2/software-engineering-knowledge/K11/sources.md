# K11 · Source and Claim Map / ADS 依据、候选与资格

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #532](https://github.com/kaicreator-mm/tally/issues/532) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 原始跨模块关系与来源

## 10. 来源、适用性与合格资格要求

**ADS 当前标准与实践来源候选**（检查真实现行规范版本及 Exact SHA 后才能用于强约束）：
- [ADS Development Workflow](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/DEVELOPMENT_WORKFLOW.md)
- [ADS Assurance Plan](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/ASSURANCE_PLAN_STANDARD.md)
- [ADS Task Decomposition](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TASK_DECOMPOSITION_STANDARD.md)
- [ADS CI Evidence](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/CI_EVIDENCE_STANDARD.md)
- [ADS #1096 Adaptive Graph](https://github.com/kaicreator-mm/ai-development-standard/issues/1096)
- [ADS #1359 Delivery-First](https://github.com/kaicreator-mm/ai-development-standard/issues/1359)

**Tally 承接**：[Engineering Business/Workflow #276](https://github.com/kaicreator-mm/tally/issues/276)、[ADS→Tally Domain Package #286](https://github.com/kaicreator-mm/tally/issues/286)、[实践反馈候选 #60](https://github.com/kaicreator-mm/tally/issues/60)。[v2.1.1 知识实验 #299](https://github.com/kaicreator-mm/tally/issues/299) 明确 `KNOWLEDGE_ASSET_NOT_QUALIFIED`，原 PENDING Claim 不能自动升级。一次经验只在其已观察环境与适用条件内成立，必须有反例和可信来源才能拓展适用范围。

**候选资产组织**：领域 Concept/Knowledge、Decision Procedure、Agent Skill、正反 Case/Validation、与现有 Workflow 的引用关系。不要为概念的不同展示形式复制规则事实；Tally 不应复制整仓 ADS 文档作为 Agent 的当前上下文。

**文档审查完成标准**：信息具可读性、术语区分、方法可执行且有停止条件、真实消费者可定位、案例可证伪、权威来源明确。**生产知识合格标准**：选定知识的适用来源审查、版权及版本范围、反例/负例、实际工程任务消费和所需独立验证均成立。本文件及其 PR 的通过不等于该模块生产知识合格，也不授予 Task/Host/Release 权威。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 必要保证不得被低成本取代 | `standards/ASSURANCE_PLAN_STANDARD.md` → §11 `Assurance floor versus selected path` | 已发布规范 | 小改动仍保留真实任务保护底线 |
| 小任务拆分与 Fast Path | `standards/TASK_DECOMPOSITION_STANDARD.md` → §12 `Fast Path`、§13 `Decomposition decision prompts`；`standards/DEVELOPMENT_WORKFLOW.md` → §2.2 `Trunk / Fast Path` | 已发布规范 | 不为减少流程而抹除原有依赖与 Gate |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。




## 复用/晋升边界

- ADS v4.19.0 的规范与 Tally v2.1.2 的建议知识不是同一权威；只复用适用的已发布规则。
- 历史 #1100 和未合并 #1384 为候选设计，不自动成为规范。
- Tally #299 原有 806 条实验图谱记录及 Pending Claim 仍 `KNOWLEDGE_ASSET_NOT_QUALIFIED`，不能整库自动导入。
- 用户暂缓的逆向分析和安全供应链专项不得悄悄成为当前知识库实施依赖。
- 真实 Package 消费须由相应版本 Product/L2/Task/Review/Host 权威另行安排；本知识库只提供可追溯方法、示例、反例与预期验证。

## 外部学科与实践资料的具体对照（2026-11-21 新增；未合格候选）

[学科基础与领域方法](disciplinary-foundations.md) · [外部实践与案例](external-practices.md) · [外部总目录](../external-source-catalog.md)。Axx 只代表被发现并链接的来源，不表示已取得版权所有的正文、全文 Clause 资格或生产激活。真正的硬性要求由当前适用 ADS 发布规范和原有工程权限定义。

| 来源 ID | 在 Kxx 中的用途 | 当前知识资格 |
| --- | --- | --- |
| [A01](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A03](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A04](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A10](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A20](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A23](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A33](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A34](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A37](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A39](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A40](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A41](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |

使用前按当前版本、出版方内容、许可证、对应 Task 适用条件、真实失败反例和必要 Reviewer 核实选定小片段，不批量导入原本未合格的 Tally #299 Claims。