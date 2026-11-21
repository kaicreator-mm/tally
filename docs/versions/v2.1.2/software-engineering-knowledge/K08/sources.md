# K08 · Source and Claim Map / ADS 依据、候选与资格

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #529](https://github.com/kaicreator-mm/tally/issues/529) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 原始跨模块关系与来源

## 10. 可追溯 ADS 来源与知识合格条件

**规范及实践来源候选**（正式采用须查验对应有效版本和 Exact SHA）：
- [ADS CI Execution](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/CI_EXECUTION_STANDARD.md)
- [ADS CI Evidence](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/CI_EVIDENCE_STANDARD.md)
- [ADS Build Artifact Governance](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/BUILD_ARTIFACT_GOVERNANCE_STANDARD.md)
- [ADS Release Standard](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/RELEASE_STANDARD.md)
- [ADS Version Integration](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/VERSION_INTEGRATION_WORKFLOW.md)（v3.3 兼容入口；实际集成流程以 `DEVELOPMENT_WORKFLOW.md`、`EXECUTION_ARCHITECTURE_STANDARD.md`、`RELEASE_STANDARD.md` 的适用条款为准）

**Tally 现有可复用所有者**：[Engineering Business/Workflow #276](https://github.com/kaicreator-mm/tally/issues/276)、[Package 消费与资格候选 #286](https://github.com/kaicreator-mm/tally/issues/286)、[经验→候选 #60](https://github.com/kaicreator-mm/tally/issues/60)。[v2.1.1 知识实验 #299](https://github.com/kaicreator-mm/tally/issues/299) 的实际终局是 `KNOWLEDGE_ASSET_NOT_QUALIFIED`；不能把 Pending 知识自动晋升为 Tally 生产 Rule/Skill/Workflow。

**候选内容类型**：Concept/Knowledge、DecisionProcedure、Skill、Case/Counterexample、Validation、现有 Workflow 的引用。真正生效的硬规则仍由已有 ADS 规范/权限/验证 Owner 负责。必须校验来源是否支持结论、适用条件、失败/反例、版权和版本、独立审查以及至少一个真实受影响工程消费者的行为，不能简单统计知识条目数。

**文档完成的定义**：概念清晰、方法具体、具有正反例/决策停止条件、定位真实 Tally 消费点、来源可追溯；**并不等于**版本 Product/L2 Freeze、Task Claim、Domain Package 生产合格或真实 Agent 开发/交付通过。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 真实 clone 与执行阶段 | `standards/CI_EXECUTION_STANDARD.md` → §6 `Clone / Checkout Contract`、§10 `Minimal CI Execution Order` | 已发布 CI 执行权威 | 由当前 Provider 真实工作区语义决定是否可共享 clone |
| 制品与 Release 不互推 | `standards/BUILD_ARTIFACT_GOVERNANCE_STANDARD.md` → §4 `Build output is not promoted artifact`；`standards/RELEASE_STANDARD.md` → §3 `Candidate Prepared vs Candidate Frozen`、§6 `Release verdict` | 已发布制品/Release 权威 | 不能将安装成功/PR Merge 推成 Release；`VERSION_INTEGRATION_WORKFLOW.md` 为兼容入口而非第二集成权威 |

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
| [A04](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A20](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A23](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A27](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A33](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A37](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A41](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A43](../external-source-catalog.md) | 作为本模块学科方法、决策步骤或具体反例的外部来源；详见两份专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |

使用前按当前版本、出版方内容、许可证、对应 Task 适用条件、真实失败反例和必要 Reviewer 核实选定小片段，不批量导入原本未合格的 Tally #299 Claims。