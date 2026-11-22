# K05 · Sources / 原始标准与候选知识的来源

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #526](https://github.com/kaicreator-mm/tally/issues/526) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 原始文档的跨模块关系与来源

## 10. ADS 来源、知识分类与完成边界

**ADS 来源候选（正式使用前以有效版本和原文件当前性为准）：**
- [ADS Testing Standard](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TESTING_STANDARD.md)
- [ADS Test Data & Scenario](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TEST_DATA_AND_SCENARIO_STANDARD.md)
- [ADS Validation](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/VALIDATION_STANDARD.md)
- [ADS CI Evidence](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/CI_EVIDENCE_STANDARD.md)

**Tally 现有能力：** [#276 工程 Business/Workflow](https://github.com/kaicreator-mm/tally/issues/276)；[#286 ADS→Tally 内容映射/消费](https://github.com/kaicreator-mm/tally/issues/286)；[#299 v2.1.1 未合格知识实验](https://github.com/kaicreator-mm/tally/issues/299)。仅选取有来源、适用条件和真实消费者的内容；旧试验 `BLOCKED_NOT_QUALIFIED`，未受审 Claim 禁止提升成生产知识。知识可属于 Concepts/Knowledge、Procedure/Skill、Workflow reference、Validation/Case；同一规则只保留一位语义 Owner，不能在多个模块制造互相冲突的 MUST。

**文档验收：** 术语可理解、具体方法可供 Agent 操作、有正反案例、来源和 Tally 消费点清晰、适用性及不可推断的状态明确。**生产资格**则需要选中知识的独立/适用来源审核、实际代码或业务行为、所需负例、Task/Review/Host 等原始证据和当前 Package 版本。文档合并不产生 Product/L2/DAG Freeze、Task Claim、代码执行授权、Package Qualification 或 Release。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 测试分层与真实执行 | `standards/TESTING_STANDARD.md` → §2 `测试层级`、§6.1 `Executed-test discovery`、§6.2 `Selected public path` | 已发布测试规范（子条款可能为 v4.19 候选文本；依实际生效政策） | Mock/Unit 不代表真实 User Critical Journey |
| 验证 Gate 和身份 | `standards/VALIDATION_STANDARD.md` → §1 `Purpose and states`、§3 `Validation Tuple`、§6 `Exact-SHA evidence and drift` | 已发布 Validation 权威 | 不得创造 `MOCK_ONLY` 作为 Gate State |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。




## 所有权、适用性与扩展时注意事项

1. ADS v4.19.0 的现行规范权威仍保留在其已发布章节；Tally KB 是领域方法与任务选用入口，不是复制后的第二份 MUST 规范。
2. ADS #1100 历史方案及 ADS #1383/#1384 当前候选需继续标 `HISTORICAL/UNADOPTED`，其统一三阶段 DAG 与全阶段 Shape 不能冒充发布规范。
3. Tally #299 的实验性图谱和 Pending Claims 仍未取得真实 Knowledge Qualification；不把它们批量转换为已合格 KB。
4. 来源真正变化时记录新源 SHA/claim 范围、受影响方法及负例；旧 Task 仍用其已明确生效版本，不静默迁移。
5. 用户已暂缓代码库逆向分析、安全与供应链专项，此处只复用实际任务原本适用的必要底线。

## 外部学科与实践资料的具体对照（2026-11-21 新增；未合格候选）

[学科基础与领域方法](disciplinary-foundations.md) · [公开工程实践与案例](external-practices.md) · [43 项外部来源总目录](../external-source-catalog.md)。下列引用的 Axx 是**来源目录索引，不代表已经完成完整标准条款/许可审查**。真正硬性约束以适用的 ADS 已发布规则为准。

| 外部源 ID | 对本模块的具体作用 | Source→Claim 当前状态 |
| --- | --- | --- |
| [A01](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A07](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A08](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A09](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A11](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A14](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A25](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A26](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A27](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |

未来选用前：核对所引用出版方的有效原文/版本、是否规范或研究、领域适用范围、数据/文本版权及实际用户 Task 的受控证据；不复制 ISO/IEEE/专业教材受限全文，不把案例推广为通用 MUST。