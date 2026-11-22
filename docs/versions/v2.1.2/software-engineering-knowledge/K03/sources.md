# K03 · Sources / 原始标准与候选知识的来源

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #524](https://github.com/kaicreator-mm/tally/issues/524) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 原始文档的跨模块关系与来源

## 10. 与其他模块的关系

从 K01/K02 获取有效需求与接口边界；K04 确认实现 Task 拆分/共享写集；K05 设计和执行测试，K06 处理根因级修复，K07 负责 Agent 交付与独立 Review，K08 负责真实打包与集成。

## 11. ADS 证据来源、Tally 资产映射与资格边界

**ADS 规范来源候选**（采用前核对实际生效版本、语义拥有者和 Exact SHA）：
- [ADS Implementation Quality](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/IMPLEMENTATION_QUALITY_STANDARD.md)
- [ADS L3 Evidence](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/prompts/L3_IMPLEMENTATION_EVIDENCE.md)
- [ADS Dependency & Toolchain](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/DEPENDENCY_TOOLCHAIN_GOVERNANCE_STANDARD.md)
- [ADS Task Decomposition](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/TASK_DECOMPOSITION_STANDARD.md)
- [ADS #1383 source candidate](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)

**模块资产候选**：Knowledge 概念/模式/反模式、以上 Decision Procedures、Agent Skills、适用案例/反例、与 L3 Implementation Task DAG-I 的消费关系。除原规范中真实的 MUST 规则外，本文的建议默认是**候选专业方法**；禁止从 Issue 标题、成功经验或单次 LLM 回答直接推导强制权威。

**内容审查**：检查来源具体支持该观点、适用条件及相反案例、案例是否真的属于该类、权利和版本边界，必要时独立复核。**效用审查**：在真实项目任务上对照不使用此知识的 Agent，比较有效需求/设计/代码/测试、返工、质量与耗时；只把可观察差异记为结果，不伪造提升幅度。Tally [#299](https://github.com/kaicreator-mm/tally/issues/299) 的原知识实验是 `BLOCKED_NOT_QUALIFIED`，不得直接把它的 PENDING Claim 升格为 ACCEPTED。Tally [#286](https://github.com/kaicreator-mm/tally/issues/286) 仍负责以后实际 Package 消费与资格闭环。

**本文完成条件**：术语可辨别、方法可复述、正反例可证伪、与 Workflow 消费边界明确、与 ADS 规范没有冲突，并完成文档来源核查。**这不等于** Agent Runtime 实现、全部知识合格或软件 Release。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 实现契约与错误 | `standards/IMPLEMENTATION_QUALITY_STANDARD.md` → §5 `Public contracts and errors`、§12 `Runtime diagnostic logs and actionable failures` | 已发布规范 | 针对当前可执行路径，诊断/保密不另设 Gate |
| Task 级证据参考 | `prompts/L3_IMPLEMENTATION_EVIDENCE.md` → `证据优先级`、`输出 Reference Pack` | 已发布执行面投影 | 只读相关切片，不创建 L1/L2 权威 |

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
| [A20](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A21](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A22](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A32](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A41](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |
| [A43](../external-source-catalog.md) | 对应本模块一个学科概念、实践方法或可证伪案例；具体适用条件请看两个新增专题文档 | `DISCOVERED_NOT_CLAIM_QUALIFIED` |

未来选用前：核对所引用出版方的有效原文/版本、是否规范或研究、领域适用范围、数据/文本版权及实际用户 Task 的受控证据；不复制 ISO/IEEE/专业教材受限全文，不把案例推广为通用 MUST。