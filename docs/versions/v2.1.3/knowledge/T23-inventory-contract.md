# T23-R1 · 细粒度知识分类与盘点合同（候选）

[#584](https://github.com/kaicreator-mm/tally/issues/584) / [#563](https://github.com/kaicreator-mm/tally/issues/563#issuecomment-7163669468) 独立规划审核 0/0/4。输入冻结 `383358955d117d155e2b3b2fc4cd7886e77f71b9`，T22 已合入。R1 是有界的实质盘点候选，不冻结最终分类、不授予后续写作范围或生产资格。P2-PLAN-01–04 待独立核查。复用原 [系统架构](../../v2.1.2/software-engineering-knowledge/systematic-knowledge-architecture.md)、[深度地图](../../v2.1.2/software-engineering-knowledge/systematic-foundation-depth-map.md)、[165 点台账](../../v2.1.2/software-engineering-knowledge/foundation-concept-coverage-ledger.md)、[R7 历史差距分流](../../v2.1.2/software-engineering-knowledge/professional-depth-gap-inventory-r7.md) 与 [T22 当前入口](casebook/waves/T22-current-knowledge-entry.md)，不重写旧台账。

## 1. 组织语义

K01–K11 是维护与专业责任入口，165 历史 ID 稳定；不是未来概念数上限或所有基础理论必归一棵树。一个确定含义只维护一个权威正文。Concept（定义）、Mechanism（因果/计算机制）、Model（对象及假设）、Method（实施/分析方法）、Decision（条件选择）分别识别；前置、约束、组成、同义、替代、应用、实现、反例是不同关系。源、Claim、Case、Oracle、Observation 作为关联资产，并非学科层级；保留原资格不自行升级。项目 Profile/Rule/Skill/Workflow 是应用/授权映射，不是普遍学理。功能族、产品类型、语言平台、质量目标、工程阶段、读者角色和问题都是检索视图，不重复定义。不同语言确有不同语义才建立具体版本化主题；框架命令只是引用。无需一细节一文件、所有主题同样深度、强制同一模板或第二套运行时/schema。

范围复用 [SWEBOK 18 KA 历史对照](../../v2.1.2/software-engineering-knowledge/disciplinary-coverage.md)；具体原始依据是 #563 7163669468 的实际读源记录（IEEE SWEBOK v4.0a PDF 导论/部分目录、CS2023 KA/HCI/SE/DM/PDC/SEP、IREB 课程登记、ISO 25010:2023 公开摘要、WCAG 2.2）。本轮未重新读全外部标准，不声称官方认证或全标准细目覆盖。18 KA 保留原映射，模型方法明确 K01/K02/K03/K05，数学基础补明确 K03（算法/类型/逻辑）及 K02 应用、K05/K11，专业实践明确 K07 通用人类实践与 Agent profile 的区别。计算/数学/工程为通用支撑，最低深度按决策需要；HCI/SEP 基础为核心横切，AI/图形/专门平台为专项。安全（独立专项）、供应链安全、代码库逆向依原授权暂缓，公开登记为未覆盖范围，不推导已安全；既有授权、租户、保密及完整性底线不失效。

## 2. 正交盘点字段与使用边界（P2-PLAN-01）

| 字段 | 值与判定 |
|---|---|
| Scope | CORE / SUPPORT / SPECIALIST / DEFERRED / EXCLUDED：适用范围，不是成熟度。跨界主题拆开不同含义。 |
| Inspection | READ（所列正文实际读过）/ PARTIAL（只读代表性正文或片段，尚未穷尽相关主题）/ UNREAD（仅候选）。READ 仅对该行明确的阅读范围成立。 |
| Content | SUFFICIENT_FOR_TARGET（已读材料能回答所列有限问题，给出理由）/ NEEDS_DEPTH（已读内容不能回答目标中的明确部分，先继续搜已有深文）/ MISSING_AFTER_SEARCH（记录实际搜索范围和证据后才能使用）/ UNRESOLVED（未形成材料判断）。未读不是缺失。 |
| Qualification | 沿用原 Source/Claim/Case/Oracle/Observation 资格；本轮 D=读到既有知识文档解释或教学例，非新增 Claim；N=只有候选导航、未核具体来源。不能以 D 代替来源核查、代码运行或生产资格。 |
| Identity & owner | 原 ID/现有 Subject、实际正文/冻结链接、提议的细节主题和维护 owner；本轮候选编号只作盘点行引用，不是新 Concept ID。 |
| Target & update reason | 要区分的工程问题、所读正文能解决什么、仍要查什么、为何值得扩展、下一次更新条件。 |

满足目标深度的最小条件是：真实决策问题有定义/机制、可区分的选择条件、至少一个适用边界或反例；若目标是具体实现，另需所选语言/运行环境合同、失败路径与可证伪观察；若是理论基础不强制代码或运行。质量、证据与内容深度独立。不存在统一字数、固定 A 覆盖目标、所有候选都要写一篇。现有 165 S/159 D1/D2+6 F/120 A 与 R7 的 15/30/42 历史分流原样保留，45 无 A 不等于缺失。现有 T22 142 案例、143 源、164 关系（129 M/35 C）、27/6/132 标引不改。R7 未选主题不是永久排除，真实细节风险可触发。无新的来源/案例/Concept 计数、运行或实验。

## 3. 阅读范围、后续步骤

本轮读取原两卷 22 篇的章节目录及代表性正文段落（详见盘点表实际定位；并非逐句审核全章）；R7、深度地图、覆盖台账、学科映射、系统架构、R10 手册作为导航；额外读 K03 算法进阶与并发进阶、K05 V&V、K07 角色/上下文、K08 release、K09 迁移、K10 首次配置 Focus、K04 Task 图的代表性段落；T18 真实案例正文。#584 前次 8 syllabus/19 章 pilot 是实际历史读源。本轮对候选细节做有限有证据盘点，未穷尽 reviewer 所述 244 个历史文件，不做 MISSING_AFTER_SEARCH 全库裁定。先确定已有解释能否满足特定目标，继续读取对应深文/Topic/Case、原始来源；已充分则引用，不充分则补新材料，未读则明确未知。下一步由实际盘点风险和语义归属选择更细子批次，独立审核再冻结；不启动 T24–T36 实施。具体候选与归属、导航见同目录的 T23 文档。