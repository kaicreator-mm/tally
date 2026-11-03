# Tally v2 — 面向 Agent 的全生命周期软件工程管理系统（ALM）Product R0

> **性质**：用户产品意图的增量设计/PRD Intake，**不是**已冻结 Product、L2、ADS wire schema、原生 Task Claim、开发授权或发布资格。正式决议由 Tally Product [#76](https://github.com/kaicreator-mm/tally/issues/76)/[#63](https://github.com/kaicreator-mm/tally/issues/63)，L2 [#78](https://github.com/kaicreator-mm/tally/issues/78)，UI [#92](https://github.com/kaicreator-mm/tally/issues/92)，Controller [#86](https://github.com/kaicreator-mm/tally/issues/86) 按 ADS v4.18 处理。
>
> **核心定义**：**Tally 是面向多个外部 AI Agent 的禅道/Jira 式软件应用生命周期管理（ALM）产品**。人和 Agent 均在同一工程工作区理解/管理项目，Agent 按 ADS 和 GitHub 权威事实自主完成开发、测试、审核与发布准备。Tally 管理需求、任务、版本、证据和经验；不在 B 中实现 Coding/Review Agent、通用工作流执行器、第二 Kernel 或替代 ADS 的通信协议。

## 1. 产品核心对象：不是每个 Issue 都是执行 Task

| 概念 | 作用与最小可见内容 | GitHub/ADS 关系 |
|---|---|---|
| Project / Repository | 软件目标、范围、成员/Agent、策略、Host/权限/预算 | GitHub 仓库与当前真实授权 |
| Product Requirement / User Story | 用户要解决的问题、优先级、可检验的能力与验收条件 | Product/PRD 的合法修订、Review、Freeze/Amendment |
| Version / Iteration | 版本目标、范围基线、Release Criteria、遗留风险 | source-qualified version/branch、RQ 与 tag |
| Architecture / Design | 方案、接口、约束、风险决策、Scaffold | L2 源版本/独立 Review/Freeze |
| WorkUnit | Concern、Source Task、Review、Repair、Test Checkpoint、Campaign 等；不是 Issue 数 | 原生 Issue identity + 有出处的 WorkUnit identity；缺证据 UNKNOWN |
| Task / Dependency DAG | 可执行子需求、前置、写集、Agent 角色与 Host 条件 | ADS-native Issue/dependency/accepted Claim；Markdown DAG 仅候选 |
| Attempt / Handoff | 外部 Agent 一次真实执行、交接、重试或后续修复 | ADS Result/Report、原始 GitHub PR/Commit/CI、Host receipt；task-meta 可选受限观测 |
| Test Case / Test Plan / Defect | 要证明的用户行为、发现的缺陷、重现、修复与复测 | 测试证据、Finding、Issue、Review、CI/Host，严禁由 Issue CLOSED 推断已通过 |
| Acceptance / Quality Gate | 来源一致的用户能力与保留安全不变量是否真实验证 | selected installed behavior、required verification、independent Review |
| Release / Change / Maintenance | 候选、RQ、合并、标签、交付、后继缺陷和新版本 | GitHub 原生事实与 ADS 版本发布治理 |
| Practice / Learning Candidate | 从多次 Task 结果归纳的可用经验、适用/失效条件、验证记录 | ADS-published Task Learning/feedback + GitHub evidence，受治理的升级候选 |

**原则**：Tally 可以给以上对象建立便于用户使用的派生管理视图、表单和 SQLite 索引，但原生 GitHub/ADS 的 Claim、PR、Review、CI、Release 权威不能被 Tally 缓存/界面替代；内部业务 DTO 不是新的 ADS 标准消息。

## 2. 软件工程闭环：八个管理 Workflows，允许并行与回流

| 阶段 | Tally 管理内容 | 实际产出者/行为 | 过关条件及下一步 |
|---|---|---|---|
| **01 需求发现** | 新项目/旧仓库导入，需求池、Epic/Story、优先级、变更申请 | Product/Research Agent 提议、用户处理重大争议 | 可追踪需求、明确验收与真正批准的 Product 基线 |
| **02 分析设计** | 架构、接口、技术约束、风险/依赖、Scaffold | Architect/Planner Agent 形成候选和源材料 | 对应 L2 Review/Freeze；无需所有需求一律串行 |
| **03 迭代规划** | Version/Sprint、能力分解、独立 WorkUnit、Task DAG、写集与角色、预算/Host | Planner Agent 给出多个安全并行 Lane 及检验点 | 经合法 ADS/native planning 和依赖确认；候选 Task 不等于 accepted Claim |
| **04 开发协作** | Task 就绪、分派/领取、Claim、attempt、工作区、PR、冲突、暂停/恢复 | 外部 Builder Agents 根据 ADS 工作，可经独立 Host/runx/Codex/Claude 等实际能力 | 真实源、PR 与 accepted Claim/K 效果证据分别读回；Tally 无需代写代码 |
| **05 Review/缺陷修复** | 独立 Review、P0/P1/P2 Finding、Defect/Repair WorkUnit、复审链 | 合格 Reviewer、Builder/Repair Agent；人只处理真正必需决策 | Exact-SHA independent Review、风险分类、修复/重新验证，而非“模型说完成” |
| **06 测试/用户验收** | 测试计划、Cases、CI/Host、回归、验收矩阵、必须保留的能力 | Validator/测试 Agent、真实 CI/物理 Host/用户验收见证 | 区分 UNIT_PASS、selected installed behavior、UNKNOWN/NOT_RUN、required gate |
| **07 发布/交付** | Release 候选、完整性、版本变更、RQ、审批、合并/Tag、发行产物 | Release Agent 准备；依 ADS 权限由合格主体执行最终操作 | 来源 SHA/tree、RQ、merge/tag、交付 readback 独立证明，不能仅看 PR merged |
| **08 维护与下版** | 线上问题、Bug/Feature、版本继承、后继回归、技术债、经验复用 | Support/Planner/Agent 继续创建/实施合法工作 | 旧版事实不改写，新版重新形成合法 Product/L2/Task/验收 |

**不是线性八步向导**：各 Stage 可以拥有平行 Lane；Review BLOCK 返回修复；线上缺陷返回需求/迭代；新需求走合法 amendment；新版本 source workfront 不必因前版不相关的 Release join 而假串行。没有合法 Claim/原生权限时，Tally 只能解释等待与下一项可行操作。

## 3. Agent 协作与 Tally 的职责分界

1. **外部 Agent 真正执行任务**：Planner/Architect/Builder/Reviewer/Repairer/Validator/Release-prep 可以是不同 Agent/会话/模型/Host；不以某一供应商为硬编码业务对象。Agent 选择依据真实能力、授权、环境、风险、成本、独立性。
2. **ADS 是互操作协议**：Task Contract、native Claim、Agent↔Agent Handoff、Task Result/Report/Learning、Review/Release 采用当前有效 ADS 的字段和 GitHub 原生承载；不编造 `tally-agent-message` 或让所有交接依赖 Tally 在线。
3. **Tally 是管理与编排视图**：登记/建议任务、组织依赖、匹配 Agent、检查领取/完成证据、识别阻塞和冲突、解释下一合法动作；被授权的 Tally 自身 GitHub 操作才走选定 B→K→C/Host。外部 Agent 按 ADS 直接协作时，Tally重新读取并对账事实。
4. **task-meta 只观测**：执行时间、attempt、Token、Model、Cost/coverage 等经 ADS 定义的 GitHub report/learning 安全摘要进入 Tally；禁止 Tally 查询 task-meta 私有 Trace/DB/API，观测缺失不阻断 Task，只标 UNKNOWN/NOT_COMPARABLE。
5. **Human 只处理真实例外**：重大需求取舍、不可委派授权/凭据、实际设备/Host 配置、预算/风险越界以及要求人的正式 Release 签署；正常 Agent→Agent Review/Repair 不应每步要求手动复制 Prompt。Tally 必须给出原因、最低步骤及证据回读。
6. **信任等级不可混淆**：`DRAFT`、`DECLARED`、`OBSERVED`、`SOURCE_QUALIFIED`、`ACCEPTED_CLAIM`、`K_ADMITTED`、`PHYSICAL_VERIFIED` 是示例性业务解释维度，不是新 ADS 原生状态。真实缺字段/分页/受限权限/被撤销时 `UNKNOWN/STALE`，不得折叠成 PASS。

## 4. Tally 实际操作界面（禅道/Jira 能力，但 Agent 原生）

- **工作台**：我的项目/版本、当前下一步、活动 Workfront、阻塞、真正需要人工处理的 Inbox、成本覆盖与风险。
- **需求中心**：需求池、Epic/Story、优先级、PRD/L2 版本、变更记录、原始需求与验收关系。
- **规划中心**：版本范围、Task DAG 与并行 Lane、Agent/Host 匹配、依赖和写集、哪些工作可立即启动。
- **任务中心**：Task/WorkUnit、Claim/Agent/Attempt、源 SHA、PR/Review/CI、实际状态、修复记录和法律准入解释。
- **质量中心**：测试计划/用例、Defect、Review Finding、回归、selected-path 实际用户能力、Release Gate。
- **版本中心**：Release 候选、RQ、变更内容、发布与后继缺陷；真实未验证/未发布必须明确。
- **工程经验中心**：Task Learning、复盘、被拒建议、Casebook、适用规则/Skill/上下文包、后续实际效果。
- **配置/治理**：仓库接入、ADS 版本、Agent 能力/角色、Host/Provider、权限、审计和预算边界。

UI 可以先做一个工作区和必要的页面组合，不应因“禅道/Jira”而重写成熟 issue 管理/表单库、通用 BPM 引擎或跨组织 SaaS 管理底座。

## 5. 经验固化必须以可核验软件工程效果为目标

```text
真实需求/验收 → Task DAG → Agent Claim/Attempt → PR/Review/Tests/Release
                                    │
                ADS GitHub Task Report / Learning + 原生来源
                                    │
           Tally 事实对账 + 根因/反例/适用范围 + 复盘
                                    │
         Evidence-qualified Practice / Context / Rule / Skill 候选
                                    │
           独立验证、授权采纳、受范围限制的下一 Task 使用
                                    │
             后续真实结果/成本/回归 → 反馈更新或撤回
```

最小能力不是“AI 自我感觉优化”：每条学习记录至少连接 `source task/attempt/commit/Review/CI`、观察与失败类别、适用条件、反例、信心/覆盖、建议动作、是否使用、使用后经证实的结果。如果无足够可比证据，必须 `ABSTAIN/UNKNOWN`，不能假称成本降低或 Agent 更优秀。

经验可指导 **需求拆分、测试边界、DAG 并行、Agent 能力选择、Review/Repair 风险、Host 适用和下一版本规划**。ADS 规则/DomainHarness Package 的规范级更新仍由各自项目和独立审核/发布负责；Tally 不自动改写规范或热替换已选 Package。

Owner 复用：[Task learning #57](https://github.com/kaicreator-mm/tally/issues/57) / [Context Pack #58](https://github.com/kaicreator-mm/tally/issues/58) / [Experience→Package #60](https://github.com/kaicreator-mm/tally/issues/60) / [Lineage #139](https://github.com/kaicreator-mm/tally/issues/139) / [Diagnosis #140](https://github.com/kaicreator-mm/tally/issues/140) / [Next-version #141](https://github.com/kaicreator-mm/tally/issues/141) / [Casebook #143](https://github.com/kaicreator-mm/tally/issues/143)。

## 6. 对既有 v2 Product 的必要修订及真实验收

**正式 Product #77 的 P0/V01–V67 尚未由本文件更新**。由 Product #76 Owner 审查新增或替换的最小用户结果：

| CJ / 验收 | 真正必须观察到的用户价值 | 禁止假阳性 |
|---|---|---|
| **ALM-01 需求到任务** | 用户在 Tally 建项目→Agent 按 ADS 产生真正可审查 Product/L2/合法 Task Issue 与原生依赖，用户无需手动写 JSON/Issue | UI mock / Markdown Task DAG = 已冻结 |
| **ALM-02 Agent 结果回流** | 至少一个真实外部 Builder 领取合法 Task、提交实际 GitHub PR/ADS 结果；Tally 读回工作进度、Claim、Review、CI 并可重启重建 | 声明 `DISPATCH_CLAIMED` 评论 = CAS accepted，Schema-valid Report = K-admitted |
| **ALM-03 独立审核与缺陷** | 独立 Reviewer Agent 发现真实缺陷，Defect/Repair WorkUnit 能追踪源码修复、复审与所需回归验证 | Issue CLOSED 或 PR merged = defect resolved/user usable |
| **ALM-04 并行 Agent** | 两个无冲突合法独立 WorkUnit 可由两个 Builder 并行，并展示冲突拒绝和正常 Review Join；首先完成单 Task 垂直切片 | 并行 READY 静态图 = 两项真实执行；现有 V63 max1 与最终目标冲突，需合法 Product 修订 |
| **ALM-05 质量与发布** | Requirements→Test Case→CI/Host→真正 installed User Acceptance→独立 RQ→授权 Merge/Tag 读回 | Node 单元 PASS、Source merged、Mock UI = 可发布 |
| **ALM-06 后续任务学到经验** | Tally 从已验证旧 Task 产生经证据约束的经验，下一个 Task Agent 获得 Context/建议，保留该建议的实际采纳/拒绝及结果 | 靠历史 Issue 时间伪造 active time、用模型自述推广 Rule、读取 task-meta 私有 Trace |
| **ALM-07 版本维护** | 发布后缺陷/变更可关联旧需求/验收，发起下一合法版本；旧 Release/Review 不被覆盖 | 新版风险覆盖历史 PASS，自动修改冻结 Product/DAG |

建议按 **M1 需求/规划真实来源 → M2 Task/Agent/Review 链路 → M3 QA/Release + 并行 → M4 经验与下一版本** 做有实际用户可用性的集成里程碑；纯设计、CI profile、源投影和 scoped tests 可以按独立写集平行开发，**不新增串行化的“所有分析完成才开始写代码”门禁**。

**仍待独立 Gate**：#77 historical parity P1/正式 Product Freeze；#82 L2；#85 ADS adoption；真实 #125 accepted Claim；完整 Agent+Review+Quality+Release；DH v0.10.3 已冻结契约仅能先行开发，待真正 tag/packed ABI 再一次对齐。PR #151 目前独立 R2 `BLOCK P1=2`，不因局部 10/10 Test PASS 进入 version 分支。

**一句话验收**：用户在 Tally 内完成需求、开发、缺陷、测试和发布的管理及必要决策；合格外部 Agent 通过 ADS 完成实际工作并公布事实；Tally 在每次 Task 后成为更好的工程任务管理与决策系统。
