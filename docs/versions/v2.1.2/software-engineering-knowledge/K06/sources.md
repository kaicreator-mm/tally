# K06 · Sources / 原始标准与候选知识的来源

> **v2.1.2 知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #527](https://github.com/kaicreator-mm/tally/issues/527) · [总索引](../README.md)。知识来源锁定 ADS v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；文档设计不代表已获知识资格或实际 Tally Runtime 消费。

## 原始文档的跨模块关系与来源

## 10. ADS 来源、知识分类与完成边界

**ADS 来源候选（正式使用前以有效版本和原文件当前性为准）：**
- [ADS Incident & Recovery Feedback](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/INCIDENT_RECOVERY_FEEDBACK_STANDARD.md)
- [ADS Maintenance Hotfix](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/MAINTENANCE_EOL_HOTFIX_STANDARD.md)
- [ADS Implementation Quality](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/IMPLEMENTATION_QUALITY_STANDARD.md)
- [ADS #1359 Delivery-First](https://github.com/kaicreator-mm/ai-development-standard/issues/1359)

**Tally 现有能力：** [#276 工程 Business/Workflow](https://github.com/kaicreator-mm/tally/issues/276)；[#286 ADS→Tally 内容映射/消费](https://github.com/kaicreator-mm/tally/issues/286)；[#299 v2.1.1 未合格知识实验](https://github.com/kaicreator-mm/tally/issues/299)。仅选取有来源、适用条件和真实消费者的内容；旧试验 `BLOCKED_NOT_QUALIFIED`，未受审 Claim 禁止提升成生产知识。知识可属于 Concepts/Knowledge、Procedure/Skill、Workflow reference、Validation/Case；同一规则只保留一位语义 Owner，不能在多个模块制造互相冲突的 MUST。

**文档验收：** 术语可理解、具体方法可供 Agent 操作、有正反案例、来源和 Tally 消费点清晰、适用性及不可推断的状态明确。**生产资格**则需要选中知识的独立/适用来源审核、实际代码或业务行为、所需负例、Task/Review/Host 等原始证据和当前 Package 版本。文档合并不产生 Product/L2/DAG Freeze、Task Claim、代码执行授权、Package Qualification 或 Release。


## 附录 A · Agent 直接调用的工作示例（非正式 Wire Schema）

**教学输入：** Pipeline 安装依赖成功，`tsc` 因某类继承语义报错。
- 使用知识：FirstFailingStep、CompetingHypothesis、Reproducer，执行 DP-K06-01/02。
- 期望产物：第一失败命令与具体诊断位置、两个可区分的根因假设、最小受控源码更正与回归检查。
- 反证：既然依赖安装成功，不应仅凭另一次历史安装失败就重建锁文件或修改代理配置。
- 停止与交接：修复超出写集或缺少真正 Runner/Host 条件，保留 `BLOCKED/NOT_RUN`；不派发无界循环。

**来源基线：** 本文 ADS `standards/` / `prompts/` 引用已固定到正式发布的 `v4.19.0@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`，并核查路径确实存在。固定来源只证明**引用身份与可读性**，不等于逐个知识 Claim 已经获得独立资格审核。ADS [#1383](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)/[Draft #1384](https://github.com/kaicreator-mm/ai-development-standard/pull/1384) 是后继设计候选，不能倒推成为 v4.19.0 已生效规则。此示例用于人工/Agent 推理和 Tally 消费方案，不是要求 Tally 新增这些字段或消息格式。


## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 事故历史与恢复 | `standards/INCIDENT_RECOVERY_FEEDBACK_STANDARD.md` → §2 `Append-oriented incident history`、§5 `Verification after recovery` | 已发布规范 | 故障缓解不能修改旧证据或直接当验证 |
| 维修/热修证据非传递 | `standards/MAINTENANCE_EOL_HOTFIX_STANDARD.md` → §5 `Evidence non-transfer`、§6 `Maintenance Fast Path` | 已发布规范 | 局部修复通过不等于整版本 Release |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。



## 具体工程主张与 ADS 条款

## 附录 B · 独立审核 R1 最小 Claim→ADS 条款映射（文档级，不是知识资格）

| 高影响工程主张 | 发布 ADS 的实际来源/节 | 身份 | 适用限制 |
| --- | --- | --- | --- |
| 事故历史与恢复 | `standards/INCIDENT_RECOVERY_FEEDBACK_STANDARD.md` → §2 `Append-oriented incident history`、§5 `Verification after recovery` | 已发布规范 | 故障缓解不能修改旧证据或直接当验证 |
| 维修/热修证据非传递 | `standards/MAINTENANCE_EOL_HOTFIX_STANDARD.md` → §5 `Evidence non-transfer`、§6 `Maintenance Fast Path` | 已发布规范 | 局部修复通过不等于整版本 Release |

**历史/候选与正式规范区分：** 本知识库提出的额外建议（含三阶段图统一命名）仍为 `TALLY_V212_PROPOSED_KNOWLEDGE`。上表只是精确源定位，不代替独立的 Source→Claim 审核，也不会赋予合法实施/Release/Host 资格。


## 所有权、适用性与扩展时注意事项

1. ADS v4.19.0 的现行规范权威仍保留在其已发布章节；Tally KB 是领域方法与任务选用入口，不是复制后的第二份 MUST 规范。
2. ADS #1100 历史方案及 ADS #1383/#1384 当前候选需继续标 `HISTORICAL/UNADOPTED`，其统一三阶段 DAG 与全阶段 Shape 不能冒充发布规范。
3. Tally #299 的实验性图谱和 Pending Claims 仍未取得真实 Knowledge Qualification；不把它们批量转换为已合格 KB。
4. 来源真正变化时记录新源 SHA/claim 范围、受影响方法及负例；旧 Task 仍用其已明确生效版本，不静默迁移。
5. 用户已暂缓代码库逆向分析、安全与供应链专项，此处只复用实际任务原本适用的必要底线。