# K02 · Sources / 原始标准与候选知识的来源

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #523](https://github.com/kaicreator-mm/tally/issues/523) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 原始文档的跨模块关系与来源

## 10. 与其他模块的关系

从 K01 获得明确的需求驱动；K04 为设计研究选择 DAG-A；K03 执行接口/状态细节，K05 建立 contract/状态验证，K09 负责兼容/迁移，K10 在运行属性为主驱动时扩充质量分析。

## 11. ADS 证据来源、Tally 资产映射与资格边界

**ADS 规范来源候选**（采用前核对实际生效版本、语义拥有者和 Exact SHA）：
- [ADS Architecture Design](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/ARCHITECTURE_DESIGN_STANDARD.md)
- [ADS Architecture Research Demo](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/ARCHITECTURE_RESEARCH_DEMO_STANDARD.md)
- [ADS L2 Evidence](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/prompts/L2_ARCHITECTURE_EVIDENCE.md)
- [ADS #931](https://github.com/kaicreator-mm/ai-development-standard/issues/931)
- [ADS #1383 source candidate](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)

**模块资产候选**：Knowledge 概念/模式/反模式、以上 Decision Procedures、Agent Skills、适用案例/反例、与 L2 Solution Design / DAG-A 的消费关系。除原规范中真实的 MUST 规则外，本文的建议默认是**候选专业方法**；禁止从 Issue 标题、成功经验或单次 LLM 回答直接推导强制权威。

**内容审查**：检查来源具体支持该观点、适用条件及相反案例、案例是否真的属于该类、权利和版本边界，必要时独立复核。**效用审查**：在真实项目任务上对照不使用此知识的 Agent，比较有效需求/设计/代码/测试、返工、质量与耗时；只把可观察差异记为结果，不伪造提升幅度。Tally [#299](https://github.com/kaicreator-mm/tally/issues/299) 的原知识实验是 `BLOCKED_NOT_QUALIFIED`，不得直接把它的 PENDING Claim 升格为 ACCEPTED。Tally [#286](https://github.com/kaicreator-mm/tally/issues/286) 仍负责以后实际 Package 消费与资格闭环。

**本文完成条件**：术语可辨别、方法可复述、正反例可证伪、与 Workflow 消费边界明确、与 ADS 规范没有冲突，并完成文档来源核查。**这不等于** Agent Runtime 实现、全部知识合格或软件 Release。


## 附录 A · Agent 直接调用的工作示例（非正式 Wire Schema）

**教学输入：** 用户需要导入较大文件，并能观察进度及重试；系统当前仅同步处理请求。
- 使用知识：ArchitectureDriver、StateOwnership、InterfaceContract、ADR，执行 DP-K02-01。
- 比较：同步、进程内队列、持久化队列；显式讨论重启恢复、重复提交、部署复杂度和成本。
- 期望产物：驱动→候选→选定理由→失败合同→Owner/接口→验证点的 ADR 及未证 UNKNOWN。
- 停止与交接：无法证明持久化/副作用语义时只声明 Demo 范围，不能宣布 L2 Freeze；选定接口交 K03/K05。

**来源基线：** 本文 ADS `standards/` / `prompts/` 引用已固定到正式发布的 `v4.19.0@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`，并核查路径确实存在。固定来源只证明**引用身份与可读性**，不等于逐个知识 Claim 已经获得独立资格审核。ADS [#1383](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)/[Draft #1384](https://github.com/kaicreator-mm/ai-development-standard/pull/1384) 是后继设计候选，不能倒推成为 v4.19.0 已生效规则。此示例用于人工/Agent 推理和 Tally 消费方案，不是要求 Tally 新增这些字段或消息格式。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 结构/所有权/失败合同 | `standards/ARCHITECTURE_DESIGN_STANDARD.md` → §3 `Required design facts`、§6 `Ownership and contracts` | 已发布规范 | 仅真实结构性变更适用 |
| 可证伪架构实验 | `standards/ARCHITECTURE_RESEARCH_DEMO_STANDARD.md` → §4 `Every Demo starts with a falsifiable hypothesis`、§9 `Evidence strength` | 已发布有界研究 | Demo 证据不得自动越级为 Production/Release |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。

## 具体工程主张与 ADS 条款

## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 结构/所有权/失败合同 | `standards/ARCHITECTURE_DESIGN_STANDARD.md` → §3 `Required design facts`、§6 `Ownership and contracts` | 已发布规范 | 仅真实结构性变更适用 |
| 可证伪架构实验 | `standards/ARCHITECTURE_RESEARCH_DEMO_STANDARD.md` → §4 `Every Demo starts with a falsifiable hypothesis`、§9 `Evidence strength` | 已发布有界研究 | Demo 证据不得自动越级为 Production/Release |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。


## 所有权、适用性与扩展时注意事项

1. ADS v4.19.0 的现行规范权威仍保留在其已发布章节；Tally KB 是领域方法与任务选用入口，不是复制后的第二份 MUST 规范。
2. ADS #1100 历史方案及 ADS #1383/#1384 当前候选需继续标 `HISTORICAL/UNADOPTED`，其统一三阶段 DAG 与全阶段 Shape 不能冒充发布规范。
3. Tally #299 的实验性图谱和 Pending Claims 仍未取得真实 Knowledge Qualification；不把它们批量转换为已合格 KB。
4. 来源真正变化时记录新源 SHA/claim 范围、受影响方法及负例；旧 Task 仍用其已明确生效版本，不静默迁移。
5. 用户已暂缓代码库逆向分析、安全与供应链专项，此处只复用实际任务原本适用的必要底线。