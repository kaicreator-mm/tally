# K01 · Sources / 原始标准与候选知识的来源

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #522](https://github.com/kaicreator-mm/tally/issues/522) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 原始文档的跨模块关系与来源

## 10. 与其他模块的关系

K02 使用被确认的需求与质量目标；K04 决定是否生成 DAG-P；K05 将验收转化为 Oracle；K07 处理多角色澄清/Review；K11 评估进一步研究的成本与停止点。

## 11. ADS 证据来源、Tally 资产映射与资格边界

**ADS 规范来源候选**（采用前核对实际生效版本、语义拥有者和 Exact SHA）：
- [ADS Development Workflow](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/DEVELOPMENT_WORKFLOW.md)
- [ADS Intent & Assumption](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/INTENT_ASSUMPTION_GOVERNANCE_STANDARD.md)
- [ADS L1 Product Evidence](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/prompts/L1_PRODUCT_EVIDENCE.md)
- [ADS #1081 L1 Evidence DAG](https://github.com/kaicreator-mm/ai-development-standard/issues/1081)
- [ADS #1383 source candidate](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)

**模块资产候选**：Knowledge 概念/模式/反模式、以上 Decision Procedures、Agent Skills、适用案例/反例、与 L1 Requirements / DAG-P 的消费关系。除原规范中真实的 MUST 规则外，本文的建议默认是**候选专业方法**；禁止从 Issue 标题、成功经验或单次 LLM 回答直接推导强制权威。

**内容审查**：检查来源具体支持该观点、适用条件及相反案例、案例是否真的属于该类、权利和版本边界，必要时独立复核。**效用审查**：在真实项目任务上对照不使用此知识的 Agent，比较有效需求/设计/代码/测试、返工、质量与耗时；只把可观察差异记为结果，不伪造提升幅度。Tally [#299](https://github.com/kaicreator-mm/tally/issues/299) 的原知识实验是 `BLOCKED_NOT_QUALIFIED`，不得直接把它的 PENDING Claim 升格为 ACCEPTED。Tally [#286](https://github.com/kaicreator-mm/tally/issues/286) 仍负责以后实际 Package 消费与资格闭环。

**本文完成条件**：术语可辨别、方法可复述、正反例可证伪、与 Workflow 消费边界明确、与 ADS 规范没有冲突，并完成文档来源核查。**这不等于** Agent Runtime 实现、全部知识合格或软件 Release。


## 附录 A · Agent 直接调用的工作示例（非正式 Wire Schema）

**教学输入：** 业务请求“增加订单取消”，当前 Product 尚未确认“已发货后是否可取消”。
- 使用知识：Requirement / BusinessRule / Acceptance / Assumption，执行 DP-K01-01；只检索当前有效 Product 与用户原始请求。
- 事实与解释：`USER_INTENT=希望取消订单`；`ASSUMPTION=仅未发货可取消`；`UNKNOWN=退款策略、重复取消语义`。
- 期望产物：带来源的需求表、业务状态矩阵、至少一个允许和一个拒绝验收示例、需 Product Owner 决策的问题。
- 停止与交接：不能把 ASSUMPTION 写成正式 MUST；待决事项局部 HOLD，已经确认的需求可按现有 Product 权威交 L2。

**来源基线：** 本文 ADS `standards/` / `prompts/` 引用已固定到正式发布的 `v4.19.0@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`，并核查路径确实存在。固定来源只证明**引用身份与可读性**，不等于逐个知识 Claim 已经获得独立资格审核。ADS [#1383](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)/[Draft #1384](https://github.com/kaicreator-mm/ai-development-standard/pull/1384) 是后继设计候选，不能倒推成为 v4.19.0 已生效规则。此示例用于人工/Agent 推理和 Tally 消费方案，不是要求 Tally 新增这些字段或消息格式。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 需求范围是否已批准 | `standards/DEVELOPMENT_WORKFLOW.md` → `Stage 1 — Product / Scope` 与 `Stage 1 Checkpoint` | 已发布流程约束 | L1 来源未获 Product 权威确认，不可写成 Frozen Requirement |
| 多问题研究映射 | `prompts/L1_PRODUCT_EVIDENCE.md` → `多项依赖研究时的 L1 收敛（有界、按需）` | 已发布可选 evidence mapping，**不是**统一 DAG-P 契约 | 真实前置研究才适用，简单需求可 INLINE |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。

## 具体工程主张与 ADS 条款

## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 需求范围是否已批准 | `standards/DEVELOPMENT_WORKFLOW.md` → `Stage 1 — Product / Scope` 与 `Stage 1 Checkpoint` | 已发布流程约束 | L1 来源未获 Product 权威确认，不可写成 Frozen Requirement |
| 多问题研究映射 | `prompts/L1_PRODUCT_EVIDENCE.md` → `多项依赖研究时的 L1 收敛（有界、按需）` | 已发布可选 evidence mapping，**不是**统一 DAG-P 契约 | 真实前置研究才适用，简单需求可 INLINE |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。


## 所有权、适用性与扩展时注意事项

1. ADS v4.19.0 的现行规范权威仍保留在其已发布章节；Tally KB 是领域方法与任务选用入口，不是复制后的第二份 MUST 规范。
2. ADS #1100 历史方案及 ADS #1383/#1384 当前候选需继续标 `HISTORICAL/UNADOPTED`，其统一三阶段 DAG 与全阶段 Shape 不能冒充发布规范。
3. Tally #299 的实验性图谱和 Pending Claims 仍未取得真实 Knowledge Qualification；不把它们批量转换为已合格 KB。
4. 来源真正变化时记录新源 SHA/claim 范围、受影响方法及负例；旧 Task 仍用其已明确生效版本，不静默迁移。
5. 用户已暂缓代码库逆向分析、安全与供应链专项，此处只复用实际任务原本适用的必要底线。