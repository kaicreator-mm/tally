<!-- tally-v212-se-knowledge-version-document-20261121 -->
# Tally v2.1.2 — Software Engineering Knowledge K01–K11
## 版本知识设计与领域 Domain Package 组织文档

| 字段 | 取值 |
| --- | --- |
| Version | **v2.1.2 — design/documentation candidate** |
| Version Product Issue | [Tally #521](https://github.com/kaicreator-mm/tally/issues/521) |
| Parent family index | [Tally v2.1 #269](https://github.com/kaicreator-mm/tally/issues/269) — `INDEX_ONLY` |
| ADS→Tally transfer and consumer candidate | [Tally #286](https://github.com/kaicreator-mm/tally/issues/286) |
| Baseline of this documentation PR | `main@54576543cf877bc535bf33b3f39a178dbe80bf5c` |
| Document state | `SOURCE_CANDIDATE / R1_TO_R10_SCOPED_ACCEPT_ON_PRIOR_EXACT_SHAS / R11_BOUNDED_PILOT_DESIGN_REVIEW_PENDING / R12_ADS_STAGED_COMPLEX_AB_DESIGN_REVIEW_PENDING / NOT_FROZEN` |
| Knowledge qualification | `NOT_QUALIFIED` |
| Package / Host / Agent runtime implementation | `NOT_IN_THIS_DOCUMENT_PR` |

**读者定位：** 本版本现包含 **11 个模块知识库目录（K01/–K11/，每个目录 9 份基础文件，另按实际需要拆分专题 Topic）+ 1 个跨模块总索引 README，另有外部来源与学科覆盖目录**。原 `K01.md`–`K11.md` 只保留初版设计导论与兼容入口；可维护的专业知识位于每个模块目录的领域模型、工程方法/决策程序、Agent Skills、案例库、验证和来源文档。GitHub Issue 管范围、讨论与审核状态。全部仍为知识设计候选，不代表 Tally 生产 Domain Package 已实现或知识合格。

> 重要历史：此前 Tally [#287–#296](https://github.com/kaicreator-mm/tally/issues/287) `[SUPERSEDED] [Harness Knowledge] K00–K09` 曾采用**不同的编号与分类**。这里的 K01–K11 仅在 v2.1.2 的当前知识设计范围内定义，不复活旧任务，不继承其已被废弃的生产者假设或任务状态。

> **独立审核 R1 的关键限定（P1-02）**：下述统一 `DAG-P/DAG-A/DAG-I` 与 `INLINE/BOUNDED_WORKLIST/EXPLICIT_DAG` 是 `TALLY_V212_PROPOSED_KNOWLEDGE`，来源于 [ADS #1100 历史候选](https://github.com/kaicreator-mm/ai-development-standard/issues/1100)及[仍为 Draft 的 ADS #1384](https://github.com/kaicreator-mm/ai-development-standard/pull/1384)，**不是**已发布 ADS v4.19.0 的完整强制三图协议。当前有效路径是 L1 可选 Evidence DAG/依赖映射，L2 按现有 Architecture/Research Demo 做有界研究，正式 implementation Task DAG 及 live dependency 归 GitHub 原权威。下文教学示例仅是未来 Tally Domain Package 设计，不授予新任务状态。

## 本版本学科与外部实践导航（2026-11-21）

- [72 个外部权威/学科/工程实践来源目录](external-source-catalog.md)：出版方/维护状态、适用领域、许可/复制风险和规范与研究的区别。
- [SWEBOK 18 KA × K01–K11 与相关学科映射](disciplinary-coverage.md)：将 18 个学科领域和系统工程、业务分析、质量、实证研究映射到现有 11 个模块；用户明确暂缓的两个专项不建设。
- **11 个模块目录均已新增** `disciplinary-foundations.md`（可应用的学科知识）和 `external-practices.md`（企业/GitHub 开放实践与有界案例）；模块 `sources.md` 标识出处，`README.md` 提供按问题选用入口。
- 当前真实文件形态：**99 个模块基础文件 + 11 份系统课程 + 22 篇两卷学科深化正文 + 6 篇 R4 定点专业深文 + 4 篇 R5 进阶推理正文 + 7 篇 R6 进阶正文 + 3 篇 R7 进阶学理正文 + 5 篇 R8 专业正文 + 5 篇 R9 专业正文 + 26 篇重点 Topic + 11 个原始导论 + 24 份版本级文档（含 W01–W22 演算、R7 缺口分析、R8/R9 来源适用/资格差距、R10 四项知识选择评价资产、R11 一个试验方案与 R12 一个 ADS 分阶段复杂任务方案） = 226 个 Markdown**。额外文档数量不等于新建 99 条已合格的知识记录。

## 体系化软件工程领域知识：从学科到底层服务

> **本版本的目标不是只管理四条高价值候选 Claim。** 完整的 **K01–K11 系统知识库**以 SWEBOK/关联学科的专业概念、工程原理、语义关系和实践方法为底座，通过 **11 份系统学科课程（每模块 5 群×3 点，共 165 个基础知识点）** 提供覆盖，并在两轮中形成 **22 篇专题化基础学科深文 + 6 篇聚焦补缺 + 4 篇 R5 深层原理正文 + 7 篇 R6 深层原理正文 + 3 篇 R7 基础理论深文 + 5 篇 R8 专业决策深文 + 5 篇 R9 专业故障/迁移/可靠性深文**；原 56 条问题索引和 26 篇深入 Topic 是便于 Agent 按需选用的高价值**投影**，并不限制知识广度。

- [系统知识架构](systematic-knowledge-architecture.md)：学科/BOK→模块→知识群→Concept/Claim/Method/Rule/Skill/Case/Oracle/Source/Consumer 的关联，条件/冲突/UNKNOWN 和唯一语义所有者。
- [K01–K11 两卷共 22 篇系统学科正文及 R4 六项补缺](systematic-foundation-depth-map.md)：每模块两篇侧重点不同的原创专业正文（两卷分别已接受文档范围）及 6 篇 R4 聚焦正文（旧 SHA 已独立限定接受）；不是将 165 点均宣称教材级完成或 Source-qualified。
- [六个原仅课程解释点的 R4 定点深文](systematic-foundation-depth-map.md#r4六点聚焦补缺旧-sha-已审)：K02.F4.3、K04.F2.3、K05.F3.3、K05.F4.2、K06.F4.3、K10.F5.1；R4 已独立接受文档候选范围，K02 的 R5 Extension/HostPort 术语修正亦已获独立限定审核；从未授予 Source/Runtime 资格。
- [五条跨模块端到端知识推理演算](professional-reasoning-worked-traces.md)：从事实/Source/有权 Scope→学科知识选择→替代方案→独立 Oracle→反例/UNKNOWN 的完整教学过程（R5 新文档，未实际运行 Agent）。
- [R5 四篇深层专业正文（K01/K03/K05/K11）](systematic-foundation-depth-map.md#r5四篇深层专业推理与五条完整演算旧-sha-已审)：把 25 个原学科知识 ID 进一步联到可反驳的原理和机制，不新增课程点或生产 Schema。
- [R6 七篇专业原理正文（K02/K04/K06/K07/K08/K09/K10）](systematic-foundation-depth-map.md#r6七模块原理深化与-w06w12旧-sha-已审)：保持 11 模块学科 Owner，新增 53 个既有知识点的专业机制链接；11/11 模块现各有一篇以上进阶正文（R6 已在旧 SHA 独立限定接受）。
- [第二卷完整推理 W06–W12](professional-reasoning-worked-traces-02.md)：涵盖跨租户架构、并行关键路径、CI 根因、Agent 延迟消息、构建缓存、版本迁移与 SLO/未知外部 Effect，全部为教学合成且没有真实 Agent/Host 运行。
- [R7 三篇专业基础理论深文](systematic-foundation-depth-map.md#r7对剩余-87-个知识点先做真实缺口盘点再有选择地深化旧-sha-已审)：补需求双向追溯/聚合不变量、算法合同/类型与证明、V&V/Oracle 的因果判别，在 165 个原课程点中增加 15 个精准 A 导航（R7 已在旧 SHA 独立限定接受）。
- [R7 原 87 点逐项盘点](professional-depth-gap-inventory-r7.md)：将旧 R6 基线无 A 的 87 点分成 15 本轮、30 下一优先、42 由真实范围驱动；**无 A 不等于基础知识缺失或 Source Claim 不合格**。
- [R8 五篇高优先级专业深化正文](systematic-foundation-depth-map.md#r8五个实质决策学科领域的高优先级深化旧-sha-已审)：K01 来源与有权 Stakeholder、K02 Extension/Host/State/Consumer 正交边界、K07 Agent 协作与上下文、K08 Provenance/Release/回滚、K11 风险与信息价值，新增 14 个旧课程点 A 链接（R8 已在旧 Exact SHA 独立限定接受）。
- [W13–W17 五条端到端工程判断教学演算](professional-reasoning-worked-traces-03.md)：合计 W01–W17 共 17 条合成演算，不是实际 Agent/Host 操作。
- [R8 原始资料与专业命题适用性对照](professional-source-applicability-r8.md)：不将规范、经验研究、GitHub 设置与生产 Rule/许可证/运行资格混淆；A68–A69 是 R8 新增两条来源发现，截至 R8 共 69 条；R9 再增三条，当前共 72 条参考。
- [R9 五篇故障/流程/迁移/可靠性专业深化](systematic-foundation-depth-map.md#r9工程故障证据迁移与可靠性的跨模块深化旧-sha-已审)：K03 错误语义/日志/测试发现、K04 风险自适应流程、K06 最小复现/未知 Effect、K09 配置/数据迁移、K10 背压/P95/真实恢复，精确新增 13 个原知识 ID A 导航（R9 已在旧 Exact SHA 独立限定接受）。
- [W18–W22 五条完整合成推理演算](professional-reasoning-worked-traces-04.md)：至此 W01–W22 共 22 个教学推理实例，均未真实运行 Agent/Host/CI。
- [R9 13 项 Source→Claim→Oracle 准入差距分析](professional-source-to-claim-readiness-r9.md)：逐项列出原理来源与真正生产消费所缺的版本、合法权威、独立 Oracle，属于分析候选，不是已获 Source Claim 资格；A70–A72 三项新发现参考使总来源达到 72。
- **R10 专业知识真正可判别的研究准备**：[五项原始出处→命题/反例/资格档案](professional-primary-source-claim-dossiers-r10.md)（C10-01～05）、[六组最小变量对照/12 道题](professional-knowledge-evaluation-fixtures-r10.md)（EV10-01A～06B）、[合法基线/评分/成本评价协议](professional-knowledge-evaluation-protocol-r10.md)、[跨 Kxx 选择、来源当前性和有权停止手册](professional-knowledge-selection-playbook-r10.md)。这些是**R10 四份版本级文档，已在旧 Exact SHA 通过独立文档限定审核（P2×1 新增精确修复）**，不是额外四个高价值 Topic、十二次真实 Agent 测试或已经运行的 DomainHarness Package；165 原 ID/120 A/72 外部来源/48 Concept 关系/Q18/W22 均不变。
- **R11 真正受权的最小知识消费试验设计**：[一个真实代码任务、两个隔离 Agent A/B、独立 QA/真实 Runner Oracle](professional-live-knowledge-consumption-pilot-design-r11.md)。先以有权 GitHub Issue 固定 Repo/H0、允许写集、独立负例和真实 Agent Execution；B 组只多选择版次固定的 K03/K05/K08/K09 专业知识。单任务只能验收受限真实闭环，不能证明普遍提效或 Tally 生产知识消费者。R11 当前只是设计候选，后续 E1 真实实验需另有合法 Task/Repo/Host 权限；E2 正式 Tally 消费需 #286 未来编号子版本 Product/L2/DAG 决议。
- **R12 ADS 全流程复杂任务/按阶段知识增强 A/B**：[真实 Node22/TS Monorepo 的多项目可恢复批量导入工程任务与 Stage S0–S6 分阶段 Kxx 注入、受权 GitHub/隐藏 QA/相同 ADS 约束](professional-ads-staged-complex-agent-ab-r12.md)。A/B 两组都有相同 ADS v4.19.0 权威规范，B 只在 L1/L2/L3/验证/维护阶段按阶段收到限量且 pin 住的专业知识；不提前提供全部 Corpus，也不新增生产 Tally/DomainHarness 接入。**R1 本地 Agent 报告尚未在本 GitHub Issue/PR 取得实际收据，R12 没有执行 Agent、QA 或测量**；这一份是待独立审核的设计候选。
- **R13 R1/R2 失败模式补缺（三篇知识候选，待限定独立审查）**：[K01 公开不透明 ID 与错误语义](K01/advanced-opaque-identifier-contract-and-error-precedence-r13.md)、[K03 本地并发文件持久化收敛](K03/advanced-concurrent-local-file-write-convergence-r13.md)、[K05 不变量驱动的高区分力测试与覆盖账本](K05/advanced-invariant-derived-discriminating-oracles-r13.md)。仅形成可审专业知识正文；R2 #535 原始证据尚未独立复算，不能声称经验自动晋升规则。本分支共 **226 Markdown**（在 R12 的 223 上新增 3），原 165 课程 ID、120 A 和 72 来源候选保持不变；R3 的 #536 语料 pin 暂不改变，审核后再决定是否使用新知识版本。
- [165 个学科知识点逐项覆盖台账](foundation-concept-coverage-ledger.md)：独立记录 165/165 课程点、R3 两卷主题关联的 159 点、R4 六篇一对一 F 聚焦正文与 R5 25 点 + R6 53 点 + R7 15 点 + R8 14 点 + R9 13 点 A 进阶原理链接（累计 120 点，R9 旧 Exact SHA 已审，R10 不新增 A）；章节关联不等于 Claim/Rule 资格。
- [K01–K11 完整系统课程索引与知识缺口](knowledge-coverage-and-growth.md)：按模块构建系统知识主干，先覆盖专业基础，按需要深入专题，逐步完成源与消费者资格。
- [Tally Domain Intelligence × DomainHarness 消费边界](domain-harness-consumption.md)：区分 ADS 知识/规范来源、Tally 专业 Domain App/Package 和 DomainHarness 通用执行；准确对应 0.11.0 Foundation、0.11.1 Workflow、0.11.2 领域语义及后续 Agent/Experience 规划，不冒充已实施能力。
- [跨模块知识概念关系网](cross-module-knowledge-relations.md)：48 条实际概念关系（R3 42 条 + R4 6 条已获旧 SHA 限定审核），解释需求、架构、实现、测试、维护、可靠性与决策如何组合推理，但不创建 Native Task Dependency。
- [领域智能的十八个合成查询与可证伪反例](domain-intelligence-use-cases.md)：为未来 DomainHarness/Agent 的 Goal→Concept→Method→Result/UNKNOWN 提供候选评估输入，未运行的实验不得宣称 PASS。
- [SWEBOK 18 KA 与跨学科对照](disciplinary-coverage.md)：覆盖不限于 26 个已深写专题，也包括计算机/数学/系统工程基础。参考 [SFIA 9 × SWEBOK 技能映射](https://sfia-online.org/en/tools-and-resources/bodies-of-knowledge/swebok-software-engineering-body-of-knowledge)，辅助辨别“知道概念”与“能在真实任务中实践”，不强制新认证。

**知识形态的分离**：系统知识 `FOUNDATION_CANDIDATE` 可持续收集、学习和关联；方法/Topic 可在文档中经过限定审核；生产选用必须另外具备确切 Source/Claim/范围/许可/规则来源、合法 Tally/Harness 消费和可观察实际验证。**不能因为四条 Claim 先行试点，就将未入选的学科基础从知识体系删去**；也不能因 165 个知识点编写完成而声称整个学科已经资格通过。

## 以工程问题检索知识（当前首选组织方式）

**分类不变**：K01–K11 是面向 Agent 开发任务的能力模块，不与 SWEBOK 18 个学科一一对应；不要把模块编号当作不可变的 L1→L3 串行执行顺序。知识可从三个视角检索：**专业问题**（K01/K02/K03/K05/K06/K09/K10）、**执行方法**（K04/K07/K08）、**贯穿决策**（K11）。其中 K04/K07/K11 横跨多个阶段，K09/K10 在需求、架构和实施阶段即可按情况启用。

**检索入口**：各模块 `README.md` 的「当前工程问题索引」包含稳定 Topic ID、真实问题、最小输入、跨模块交接、必须拒绝的错误；先按问题找到少量知识，再阅读对应专业方法/Skill/来源/负例。Topic ID **仅用于文档定位**，不创建 Workflow Engine、知识数据库、Task Gate 或额外 Agent 权限。

**职责边界**：K02 跨模块架构/公共合同，K03 Task 内具体实现；K05 测试 Oracle/充分性，K08 Provider/CI/Artifact 的实际执行和证据；K10 通用可靠性/错误/恢复模式，K06 特定事故因果定位，K03 当前合法修复；K09 旧行为兼容/迁移，K01 业务需求；K04 Task 组织、K07 Agent 协作、K11 成本和停止策略为跨阶段方法。

**来源与状态**：问题索引不复制已有 99 个基础专题文件的正文；`K01.md`–`K11.md` 保留为历史导论入口而非第二套权威。引用 [72 项外部来源](external-source-catalog.md) 和 [SWEBOK 对照](disciplinary-coverage.md) 必须按具体 Claim、版本、适用性和反例判断；`SOURCE_CANDIDATE/NOT_QUALIFIED/PROPOSED_NOT_CONNECTED` 不变。

### 首批选定知识的真实来源定位（候选，未取得资格）

[4 条重点 Topic 的 Source→Claim→Counterexample 与最小消费验证准备](selected-source-claims-preflight.md)：精确定位已发布 ADS 的测试发现/实际公开路径、CI 分层和 Agent ACK/旧 HEAD 事实；区别发布标准里的 `SHOULD`、标注 `candidate` 的小节、现行 Validation/Release 权威与合成案例。**只完成来源条款核查，不等于知识/Package/Agent 资格通过**，不提前启动 Deferred 的 #286。

## 1. 产品与领域范围

### 1.1 本版本要解决的问题

让 Tally 在真实软件开发工作中，能够选择**适用、可信、可验证**的软件工程知识，辅助 Agent 完成：

```text
用户需求及业务约束
  → L1 Requirements + optional DAG-P
  → L2 Solution Design + optional DAG-A
  → L3 Detailed Design + Implementation DAG-I
  → actual build / tests / repair / integration / delivery
  → real task outcomes and experience feedback
```

**ADS 的定位是 Agent 开发实验、实践和可复用工程方法来源**；它产出受证据限定的规范、流程、失败/成功案例和研究结果。**Tally 的定位是 Software Engineering Domain App**：其 Engineering Domain Package 组合选定的 Workflow、Knowledge、DecisionProcedure、Skill、Rule、Validation、Case，Engineering Business/Agent 将这些用于实际项目。被开发出的软件在运行时无需依赖 ADS/Tally。**只讨论 ADS 与 Tally**，不增设第三方生产平台或新编译器。

### 1.2 明确暂缓的两个专项

1. **代码库理解和逆向分析**：不是本版本独立知识领域；真实 Task 执行仍必须读取必要源码、调用入口和当前证据，不得伪造代码理解。
2. **软件安全与供应链知识专项**：本版本不编制安全工程大全或供应链风险/扫描 Pipeline；已发布 ADS 的适用权限、敏感数据、依赖质量与安全限制仍有效，不因专项推迟而取消。

体系化建设软件工程专业知识、理论、学科关系与工程技术细目，但不在一次 PR 中穷尽所有教材或复制受版权限制的全文；不开发任意领域 Agent 平台、通用知识编译器、额外调度器、新 Workflow Engine、独立 Task/Claim/Release 权威，也不把 11 个 Issue 都解释成运行时代码任务。

## 2. 十一个知识模块：唯一工作项索引

| Module | Knowledge module | Priority | Main workflow | 知识重点 | 典型决策与 Skill |
| --- | --- | --- | --- | --- | --- |
| [K01 知识库](K01/README.md) · [初版导论](K01.md) · [Issue #522](https://github.com/kaicreator-mm/tally/issues/522) | Requirements Engineering & Business Modeling | P0 | L1 / DAG-P | 需求与业务规则、验收条件 | 需求澄清/冲突判定 |
| [K02 知识库](K02/README.md) · [初版导论](K02.md) · [Issue #523](https://github.com/kaicreator-mm/tally/issues/523) | Architecture & System Design | P0 | L2 / DAG-A | 架构驱动、模块/数据/接口设计 | ADR 与方案权衡 |
| [K03 知识库](K03/README.md) · [初版导论](K03.md) · [Issue #524](https://github.com/kaicreator-mm/tally/issues/524) | Detailed Design & Implementation | P0 | L3 / DAG-I | 算法、状态、错误及可维护代码 | 任务级详细设计/合法 Builder |
| [K04 知识库](K04/README.md) · [初版导论](K04.md) · [Issue #525](https://github.com/kaicreator-mm/tally/issues/525) | Task Planning & Decomposition | P0 | L1/L2/L3 | 按复杂度选择 INLINE/WORKLIST/DAG | 分解真实依赖及写集 |
| [K05 知识库](K05/README.md) · [初版导论](K05.md) · [Issue #526](https://github.com/kaicreator-mm/tally/issues/526) | Software Testing & Quality Verification | P0 | L3/Review | Oracle、契约、边界与真实测试层 | 选测试/验证测试是否实际执行 |
| [K06 知识库](K06/README.md) · [初版导论](K06.md) · [Issue #527](https://github.com/kaicreator-mm/tally/issues/527) | Debugging & Root-cause Repair | P0 | L3/Repair | 错误分类、首个失败点、根因与恢复 | 有界诊断/修复 |
| [K07 知识库](K07/README.md) · [初版导论](K07.md) · [Issue #528](https://github.com/kaicreator-mm/tally/issues/528) | Multi-Agent Engineering Collaboration | P0 | All phases | 角色/会话/任务交接、交互通道与共享写冲突 | 合法 Agent Handoff/独立审核 |
| [K08 知识库](K08/README.md) · [初版导论](K08.md) · [Issue #529](https://github.com/kaicreator-mm/tally/issues/529) | Build, Integration & Delivery | P0 | Integration/Delivery | 项目构建、CI、集成/发布产物与证据 | 最小有效 CI + 正式交付核对 |
| [K09 知识库](K09/README.md) · [初版导论](K09.md) · [Issue #530](https://github.com/kaicreator-mm/tally/issues/530) | Software Evolution & Refactoring | P1 | Maintenance | 兼容性、重构、迁移、历史行为保护 | 变更影响/兼容决策 |
| [K10 知识库](K10/README.md) · [初版导论](K10.md) · [Issue #531](https://github.com/kaicreator-mm/tally/issues/531) | Operability, Reliability & Performance | P1 | Quality/Maintenance | 日志、错误、重试/恢复、性能测量 | 诊断与恢复/性能评估 |
| [K11 知识库](K11/README.md) · [初版导论](K11.md) · [Issue #532](https://github.com/kaicreator-mm/tally/issues/532) | Engineering Economics & Decision-making | P1 | Cross-phase planning | 成本、风险、交付价值与自适应深度 | 工程方案的总成本比较 |

**知识库阅读方式：** 点击 K01–K11 进入对应目录的 `README.md`，按需读取具体主题，不把全部文件装入 Agent 的工作上下文。每个库有领域模型、工程方法和 3 个已有 Decision Procedures、Agent Skill/直接任务提示词、案例与反例、可证伪验证及固定 ADS 来源。原 11 份合计约 8.6 万字符的导论仍保留作历史设计入口；本次知识库保留 **11×9=99 个模块基础文件**，再为 11 个模块补充各自系统课程（合计 **165 个基础知识点**）及两轮 **22 篇系统学科深化正文**、R4 六篇聚焦补缺、R5 四篇、R6 七篇、R7 三篇、R8 五篇与 R9 五篇深层推理正文，并在实际需求驱动下额外深化 **26 个问题级专题**（不是 26 个已合格知识资产）。

**K01–K08**：优先用于首个真实 Agent 开发垂直切片。**K09–K11**：当前版本同样要整理完知识设计与正反例，但其运行时消费按项目需要推进，不作为首条垂直切片的无条件阻塞。模块编号不是执行顺序，亦不建立自然强制的 K01→K02→…→K11 Task Dependency 链。

### 2.1 每模块的知识库文件结构

```text
software-engineering-knowledge/
├── README.md               # v2.1.2 跨模块索引
├── systematic-knowledge-architecture.md # 领域语义与知识关系
├── knowledge-coverage-and-growth.md # 完整覆盖和增长路径
├── foundation-concept-coverage-ledger.md # 165 个课程 ID 对应 S、D1/D2 或 F 正文
├── systematic-foundation-depth-map.md # 22 两卷 + 6 Focus + 4 R5 + 7 R6 + 3 R7 + 5 R8 + 5 R9 进阶正文
├── domain-harness-consumption.md # Tally/Harness 消费与权限边界
├── cross-module-knowledge-relations.md # 48 条课程概念关联（文档候选）
├── domain-intelligence-use-cases.md # Q01–Q18 十八组教学输入与负例
├── professional-reasoning-worked-traces.md # R5 W01–W05 五条跨学科完整演算
├── professional-reasoning-worked-traces-02.md # R6 W06–W12 七条跨学科演算
├── professional-depth-gap-inventory-r7.md # R7 原 87 个无 A 课程点的真实差距/优先级
├── professional-reasoning-worked-traces-03.md # R8 W13–W17 合成推理演算
├── professional-source-applicability-r8.md # R8 来源、候选命题和失效范围
├── professional-reasoning-worked-traces-04.md # R9 W18–W22 合成推理演算
├── professional-source-to-claim-readiness-r9.md # R9 来源→主张→Oracle 资格缺口
├── professional-primary-source-claim-dossiers-r10.md # R10 五条原始出处的命题/反例/资格档案
├── professional-knowledge-evaluation-fixtures-r10.md # R10 六对十二例最小变形问答
├── professional-knowledge-evaluation-protocol-r10.md # R10 同模型基线与独立评价办法
├── professional-knowledge-selection-playbook-r10.md # R10 知识检索/有权来源/停止手册
├── professional-live-knowledge-consumption-pilot-design-r11.md # R11 两隔离Agent/真实任务/QA 受权试验设计
├── professional-ads-staged-complex-agent-ab-r12.md # R12 ADS 全流程复杂任务与 S0–S6 分阶段 Kxx 注入验证设计
├── K01.md ... K11.md        # 初版单篇导论：历史兼容入口
└── K01/ ... K11/
    ├── README.md           # 模块能力边界、问题与选用指南
    ├── systematic-syllabus.md # 五群×三项系统学科知识主干
    ├── foundations-deep-dive-01.md # 第一卷：已按旧 SHA 独立审查的学科正文
    ├── foundations-deep-dive-02.md # 第二卷：R3 已独立接受的文档学科正文
    ├── focus-fx-y-*.md # 仅部分模块：R4 六篇聚焦补缺文档（旧 SHA 已审）
    ├── advanced-*.md # K01–K11：R5 四篇 + R6 七篇 + R7 三篇 + R8 五篇旧 SHA 已审；R9 五篇待审
    ├── disciplinary-foundations.md # 外部学科/标准与适用机制
    ├── external-practices.md # 外部企业/GitHub 案例与可证伪工程做法
    ├── domain-model.md     # 术语/实体/规则/语义关系
    ├── engineering-methods.md # 方法、权衡与有界 Decision Procedures
    ├── agent-skills.md     # Agent 输入输出与直接可用的任务提示词
    ├── casebook.md         # 正反例、场景变体、反模式和经验条件
    ├── validation.md       # 负例、测验、未来真实消费者验证
    ├── sources.md          # ADS 精确来源、Claim 依据及权威边界
    └── topics/             # 按实际使用逐步深化的高价值问题卡片（目前共 26 篇）
```

这不是一个新的执行系统或独立知识数据库；只是**让专业知识可独立维护、按问题选择和增量补充**的文档知识库。K01–K08 优先适用于核心软件开发问题，K09–K11 按真实项目场景有条件选择。每个库不因创建而自动 Source-qualified 或 Connected。

### 2.2 问题级 Topic 的覆盖范围和使用次序

在 11 个模块 README 中共有 **56 条问题级路由**，其中已有 **26 篇经过 Writer 编写、尚待独立知识审查的自包含 Topic**（并非必须为其余 30 条机械补齐文件）。Topic 从真实问题、前置条件、专业原理、备选判断、正负案例、可证伪 Oracle、来源与 STOP 构成。选择时先按当前任务确定路由，再读取最少的 Topic/DP/Skill/Case/Source，而不是载入整个知识库。

| 模块 | 当前已深化的问题专题数 | 重点深入方向 |
| --- | ---: | --- |
| K01 | 2 | 冲突需求、业务验收与可追溯性 |
| K02 | 2 | 质量权衡、状态所有权 |
| K03 | 2 | 异步幂等、算法/数据结构选择 |
| K04 | 2 | 真依赖、并行写集 |
| K05 | 3 | Oracle、False PASS、真实集成证据 |
| K06 | 2 | 首个失败、竞争假设 |
| K07 | 2 | 独立角色、过期消息对账 |
| K08 | 3 | CI 阶段、Clone 复用、实际交付 |
| K09 | 2 | API 演进、数据迁移 |
| K10 | 3 | 可行动错误、UNKNOWN 副作用、性能基线 |
| K11 | 3 | 成本权衡、必须的保证、真实效用对照 |

保留正式 Knowledge/Procedure/Case 的唯一维护权威：Topic 不复制 ADS 规范原文，不新建数据模型或机器运行 Schema。示例和外部参考均为未合格的设计候选；原始 GitHub Task/Claim/Review/Release 权威保持不变。需要真实运行才可报告 `PASS`。

## 3. 统一知识建模：Workflow 与可复用知识分开

### 3.1 软件工程 Domain Package 的逻辑资产

| Asset category | 职责 | 来源与执行边界 |
| --- | --- | --- |
| **Domain Concepts / Glossary / Relations** | Requirement、Rule、Invariant、Design/ADR、Task、Contract、Test/Oracle、Evidence、Case、Decision/Skill 等概念及可追溯关系 | 不发明第二套 GitHub Task/Review 状态；可复用现有 Tally #276 Domain Data |
| **Workflow** | L1 需求、L2 方案设计、L3 实现、Review/Verification、Repair、Integration/Delivery、Maintenance 的有界活动/输入输出 | 定义领域工作推进方法，不是 Agent Executor、独立调度引擎或 Effect Permit |
| **Knowledge / Patterns / Counter-patterns** | 理论/原则/设计取舍、适用边界、正例、反例、失败经验 | 区分规范事实、可复用经验、待证假设，不靠 LLM 自报晋升 |
| **Decision Procedures** | 具体选择方法：需求是否明确、怎样设计、任务如何拆、选择何种测试、如何根因修复 | 结果是受证据约束的建议；需要业务/权限决策时仍交给原有权威 |
| **Skills / Templates / Recipes** | 可直接给 Agent 的任务级操作方法、输入输出结构和停止点 | 不是外部写入授权；适用源与版本必须可恢复 |
| **Rules / Policies / Validation** | 确定性约束、必要证据、准入/独立性及工程质量底线 | 按现行 ADS pinned standard 与 Tally 现有治理，不创建第二个 Gate |
| **Case / Experience / Evidence** | ADS 真实实验及 Tally 开发结果，保留失败、反例、未测量和版本身份 | 不能将一个案例的 PASS 自动变成跨项目 MUST 规则 |

同一 ADS 文档可提供不同类型资产，但**单一语义拥有唯一知识来源与维护权威**，Workflow 只能引用这些资产，不应把同一套要求复制到每个工作流文件中。

### 3.2 每个知识资产至少应能说明什么

本版本不是强制发明机器 Schema。为便于后续 Tally 筛选，知识资产的可读记录至少能恢复：

- **Identity/Provenance**：模块编号、ADS source/ref/exact revision、是否规范已采纳、原始实验/Review/Case 来源及当前性；
- **Question & purpose**：它解决哪个真实工程问题、所处范围、Owner 和允许/禁止的决策；
- **Applicability / boundaries**：场景、前置条件、不适用情况、竞争解释、仍未知的事实；
- **Method/Procedure**：决策步骤、输入/输出、失败处理、人工决策时机、停止规则；
- **Positive & negative**：至少一条来源有支撑的典型正例与一条可识别的失效/反例；
- **Consumer binding**：哪个 Tally Workflow/Business 行为/Agent Skill 能使用，以及使用后能观察到的具体变化；
- **Qualification**：独立/授权来源审查、相关测试/负例、真实任务结果、版本与残余风险。

没有证据的内容只能记为 **EXPERIMENTAL / PROPOSED / UNKNOWN**；被选择和发布的状态不能由字段值或单个 Agent 自报产生。

### 3.3 跨模块能力所有权（P2-02）

| 交叉部分 | 唯一知识/执行职责 | 典型消费关系 |
| --- | --- | --- |
| K10 vs K03 vs K06 | **K10** 提供通用运行质量/可恢复错误模式；**K03** 将已选模式实现在当前 Task 代码/测试；**K06** 根据具体故障证据判断根因及修复 | K03 引用 K10，不复制普遍可靠性 Rule；K06 不重定义原错误合同 |
| K05 vs K08 | **K05** 拥有测试 Oracle、测试充分性/有效性选择；**K08** 拥有 Build/CI/Artifact 的执行路线和真实 Provider/版本收据 | K08 按 K05 已确认测试目标执行并提供来源；K05 不代替发布系统 |
| K01 vs K02 vs K04 | **K01** 拥有需求意义和未决业务问题；**K02** 拥有架构候选/合同；**K04** 只选择任务组织/真实依赖 | P/A 工作图为候选视图；不重复 L1/L2/Task 原始权威 |

以上“拥有”指 **v2.1.2 知识内容归属**，并非创建新的 ADS 规范 Owner 或 Tally 任务状态系统。正式强制规则仍引用适用的已发布 ADS 来源/项目权限。

## 4. Workflow 组合与跨模块依赖

### L1 — Requirements

- 主要使用 K01（需求与业务规则）、K04（工作深度/研究依赖）、K07（角色/协作）；需要质量/成本权衡时引用 K11。
- 目标是可验证的 Product/Requirement/Acceptance 基线与显式 UNKNOWN。L1 的研究 DAG-P 只在真实依赖存在时采用。

### L2 — Solution Design

- 主要使用 K02（架构与权衡）、K01（已确认需求）、K04（设计任务 DAG-A）、K07（多设计者交接）。
- 有旧功能或持久化迁移时引用 K09；若性能、可用性等质量属性是关键驱动，引用 K10。
- Demo 只能证明原 Demo 的假设；设计/业务冻结仍由现有权威负责。

### L3 — Detailed Design, Coding, Repair

- 主要使用 K03（详细设计/实现）、K04（DAG-I 和原子任务）、K05（测试 Oracle）、K06（根因修复）、K07（Agent 协作）、K08（构建集成）。
- 实际 Code/Build/Test/Review 证据绑定 exact source、执行环境、当前 Task/Claim 与现有校验/发布标准。
- 旧代码兼容性/可维护性变更选择 K09，日志/恢复/性能按需要选择 K10。

### Delivery & Improvement

- K08 负责将真实可运行产物、CI/Host/集成/用户关键路径证据组织为交付判断；K06/K09/K10 负责变更或运行反馈；K11 评估交付质量和真实总成本。
- ADS 从经审查的真实成功/失败实例中继续迭代方法，Tally 在未来任务选择版本化知识；不得使运行中的 Task 被无声更新。

**注意：以上为知识调用/内容的语义依赖，不是 GitHub native Task Dependency。** 三个 DAG 视图分别为 P、A、I；实现 Task 的真实依赖和 Claim 仍由现有 native GitHub/ADS 规范掌握。不为跨阶段每项 Knowledge 创建工作图边。

## 5. 每模块的验收焦点与反例索引

| Module | 实际任务中必须能观察到的正确性/反例 | 详细说明 |
| --- | --- | --- |
| K01 | 用明确规则生成可观察验收，而非猜测业务决策 | [模块详细设计 #522](https://github.com/kaicreator-mm/tally/issues/522) |
| K02 | 证据支持设计选型、失败/权衡清晰，不把 Demo 当生产成功 | [模块详细设计 #523](https://github.com/kaicreator-mm/tally/issues/523) |
| K03 | 真实变更契约 + 改动测试/构建而非虚假 PASS | [模块详细设计 #524](https://github.com/kaicreator-mm/tally/issues/524) |
| K04 | 能识别虚假依赖、共享 Writer 冲突和不必要 Issue | [模块详细设计 #525](https://github.com/kaicreator-mm/tally/issues/525) |
| K05 | 正确区分 Mock、本地、CI、Host、Release 证据 | [模块详细设计 #526](https://github.com/kaicreator-mm/tally/issues/526) |
| K06 | 真实可复现、根因修复且有回归证据，能停止重复修复 | [模块详细设计 #527](https://github.com/kaicreator-mm/tally/issues/527) |
| K07 | Git/GitHub 首选但不唯一；ACK ≠ Task/Claim/Review PASS | [模块详细设计 #528](https://github.com/kaicreator-mm/tally/issues/528) |
| K08 | 真实 build/test/集成/安装证据，成本与失败可诊断 | [模块详细设计 #529](https://github.com/kaicreator-mm/tally/issues/529) |
| K09 | 不破坏原消费者，迁移失败/回滚有验证 | [模块详细设计 #530](https://github.com/kaicreator-mm/tally/issues/530) |
| K10 | 真实故障有可操作错误；性能结果有基线 | [模块详细设计 #531](https://github.com/kaicreator-mm/tally/issues/531) |
| K11 | 避免无价值的多 Agent、过量 CI，同时不降低必要保证 | [模块详细设计 #532](https://github.com/kaicreator-mm/tally/issues/532) |

这些案例用于以后可复用知识的最小可证伪验收，**不是声称已经运行**。如果真实源码、Host、权限或人工源审查条件不具备，应输出 `NOT_RUN / UNKNOWN`，而不是在文档中模拟 `PASS`。

## 6. 原始证据与现状（不能假装已合格）

- **ADS 已有强规范资产**：`DEVELOPMENT_WORKFLOW.md`, `ARCHITECTURE_DESIGN_STANDARD.md`, `IMPLEMENTATION_QUALITY_STANDARD.md`, `TASK_DECOMPOSITION_STANDARD.md`, `TASK_DAG_GOVERNANCE_STANDARD.md`, `TESTING_STANDARD.md`, `VALIDATION_STANDARD.md`, `CI_EXECUTION_STANDARD.md`, `GITHUB_AGENT_INTERACTION_PROTOCOL.md`, `INCIDENT_RECOVERY_FEEDBACK_STANDARD.md` 等，在 [ADS repository standards](https://github.com/kaicreator-mm/ai-development-standard/tree/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards)。各模块 Issue 的 §5 分别列出真实规范所有者。
- **ADS 三层图原设计**：[#1100](https://github.com/kaicreator-mm/ai-development-standard/issues/1100)、[#1096](https://github.com/kaicreator-mm/ai-development-standard/issues/1096)、[#1081](https://github.com/kaicreator-mm/ai-development-standard/issues/1081)、[#931](https://github.com/kaicreator-mm/ai-development-standard/issues/931)；新增三级工程扩展 [#1383](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)/[Draft #1384](https://github.com/kaicreator-mm/ai-development-standard/pull/1384) 尚未正式版本采纳。
- **Tally 最小 Business + Workflow**：[Tally #276](https://github.com/kaicreator-mm/tally/issues/276) 的 v2 工作复用了现有 Business/Intent/Workflow Bridge；不等于已包含此 11 模块的独立 Domain Data。
- **Tally v2.1.1 知识实验**：[Tally #299](https://github.com/kaicreator-mm/tally/issues/299) 在 [WIP exact source](https://github.com/kaicreator-mm/tally/tree/706a3ec33c2ba7ba3b60116471139113edb7a6cb/docs/versions/v2.1.1) 完成实验性知识图谱，但结论是 `EXPERIMENT_WORK_COMPLETED_WITH_LIMITATIONS / KNOWLEDGE_ASSET_NOT_QUALIFIED`。图谱 806 个记录、106 原始 PENDING Claims、11 个 PARTIAL views、29 OPEN Coverage Gaps、0 ACCEPTED ReviewDecision、0 真正的 Counterexample；真实人工 G4 审查没有完成。**不能自动晋升这些内容到本版本的生产 Package**。
- **Tally Package 资格候选**：[Tally #286](https://github.com/kaicreator-mm/tally/issues/286) 仍为 deferred planning candidate。任何后续生产知识消费都需要挑选受约束的**少量**合格内容，不能先假设已覆盖 11 个模块。

## 7. 本版本范围与验收分两层，防止伪完成

### A. 本次 **设计/文档交付**（本 PR 处理）

- [x] 11 个不同的知识模块 Issue 已创建并真实读取；各自明确了知识范围、决策程序、Skill 输入输出、正反例、来源、目标 Workflow、知识使用与资格边界。
- [x] 11 个知识库目录分别建成，每库 9 个文件（原 7 文件 + 学科基础 + 外部实践），并保留旧 `Kxx.md` 的导论入口。
- [x] 一个 v2.1.2 版本 Issue 和一个 README，索引 11 个模块目录（99 个基础知识文件 + 11 份系统课程 + 26 个重点问题 Topic）、11 个原始导论、对应 Issue 及 ADS/Tally 来源。
- [ ] DOC-only 源码差异核查、Markdown 结构/链接与历史权威边界核对，必要的不同会话独立文档 Review；不对无关运行时代码执行全量 CI。
- [ ] 通过 GitHub 读取固定候选 HEAD/文档 Blob，记录已完成的是文档，未完成的是合格知识与真实 Runtime。

### B. **以后合格领域能力与真实 Agent 开发**（尚未授权实施）

- 按 K01–K08 挑选当前真实任务必需的小范围，Source→Knowledge Claim→Rule/Decision/Skill→Workflow 消费→真实代码/测试/Host/Review 的精确引用链。
- 知识来源与适用条件合格、正反例可证伪、有独立/相应人类来源审查（适用的权威条件）以及合法访问/版权处理。
- 用真实项目的一个正例 + 至少一个负例/修复路径证明：L1 需求 → L2 设计 → L3 实现/测试/交付闭环；Agent 能区分 UNKNOWN 与已证事实。
- 以可比较的、预先明确标准的真实编码 Task 评估相对于“直接用 Agent”的需求完成、遗漏、回归、无效测试、修复次数、Review 返工、延迟与成本。**不能用只完成知识图谱问答的实验数据代替编码结果**。
- K09–K11 按实际项目要求逐步消费，不成为首个 MVP 的无条件阶段阻塞。当前版本文档建设不产生原本没有的 Release/Host/Task 权限。

## 8. 版本实施边界与推荐顺序

**Documentation planning order, not GitHub native blocked-by:**

1. **本轮**：建立版本 #521 与知识 Issue #522–#532、统一版本 README 和文档审查；只证明目录与具体知识设计完整。
2. **未来实际产品选择**：从 K01/K02/K03/K04/K05/K06 选择能闭环的最小知识/DecisionProcedure/Skill，K07/K08 复用现有工程协作与构建交付能力。
3. **必要的具体能力缺口**：只有当前 Tally Business/Agent/Domain Package 真实消费者缺少时，才提出最小代码改动、合法 Task DAG/环境/Host 验证；不能仅根据本 Issue 的 TODO 派发 11 个编码 Agent。
4. **真实效果验证**：一次完整的面向用户的工程任务，保留有界修复、Review 和必要 Release 证据；对照无辅助 Agent 和知识辅助 Agent 的结果，再决定是否扩大。
5. **扩展 K09–K11**：对应客户或项目实际需要时再择优进行消费资格工作；保留未来与其他 ADS 规范版本并存的升级路径。

## 9. 最小 Agent 知识选用与真实消费者示例（设计草案）

知识文件中的 Procedure/Skill 只有在**当前任务问题确实需要**时才被选择，不将 11 份文档完整灌入每个 Agent 的上下文。Tally 的 Engineering Business/Workflow 保持真实任务和权限权威，知识模块只提供可追溯的工程决策方法。

**最小知识选用次序**（非新调度器、非强制新 Schema）：

1. 从现有 Issue/用户请求获得**当前目标、受权业务约束、真实 Task/版本和下一项需要决定的问题**；不知道的材料事实保持 UNKNOWN。
2. 判断问题属于 L1 需求、L2 设计、L3 具体实现、验证、修复还是交付；阶段可 INLINE/BOUNDED_WORKLIST/EXPLICIT_DAG，但不得为选知识而机械构造 DAG。
3. 按问题只引入对应模块的方法：L1→K01/K04，L2→K02/K04，L3→K03/K05，失败→K06，跨 Agent→K07，构建/交付→K08；实际触发旧功能兼容/运行故障/成本权衡时再选择 K09/K10/K11。
4. 检查选用知识的**来源版本、适用条件、反例和仍未知项**，区分 **ADS 已发布约束**、**知识设计候选**、**Tally #299 的未合格实验观察**。未合格知识只可作为明示的不确定研究候选，不能成为未经授权的 MUST、Task/Review PASS 或 Release 决定。
5. 优先使用对应模块 `agent-skills.md` 的任务级输入、输出与停止条件；`Kxx.md` 附录 A 只是初版教学示例。不得据此创造新强制 Wire Schema。完成实际执行后记录代码/测试/业务结果的准确来源，由现有权限链判定资格。

**贯穿示例：订单取消需求（仅为教学场景，不代表 Host 已运行）**

| 步骤 | 选用内容 | Agent 必须产出或保留 | 必须避免的伪结论 |
| --- | --- | --- | --- |
| 需求 | K01 + 必要时 K04 | 允许取消的状态、未知退款政策、正反验收及真实来源 | “用户说加按钮”就意味着可取消所有订单 |
| 架构 | K02 | 状态写入 Owner、对外 API 错误合同及备选设计取舍 | 做了一个 Demo 就自动 L2 Freeze |
| 实施 | K03 + K04 | 当前 Task 写集内的状态转换、失败/重复调用处理 | 代码提交就等于业务完成 |
| 验证 | K05 | 从规则导出的测试 Oracle、runner 真实发现/执行、必要 API→DB 负例 | Mock 绿灯等于真实用户 E2E |
| 修复（按需） | K06 | 首个失败点、可证伪的根因、最小合法修复与当前回归 | 无限重试或复用旧 CI 的错误诊断 |
| 协作与交付 | K07 + K08 | 精确候选版本、独立 Reviewer 来源、真实集成/打包或明确 NOT_RUN | Agent 交接 ACK 或 PR Merge 自动变 Release |
| 后续演化（按需） | K09/K10/K11 | 旧行为保护、诊断/恢复以及真实成本/效用反馈 | 无真实测量就声称通用质量提升 |

**已核对的 ADS 基线：** 发布提交 [v4.19.0 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`](https://github.com/kaicreator-mm/ai-development-standard/commit/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c)。11 个模块文档内引用的 `standards/` 与 `prompts/` 文件覆盖 **29 个不同源路径，29/29 在此提交真实存在**，且正文链接已由可变 `main` 改为该固定版本。这里只证明文件身份、可访问性及设计范围，**不证明每项知识陈述已完成独立 Source→Claim 审核**。ADS [#1383](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)/[Draft PR #1384](https://github.com/kaicreator-mm/ai-development-standard/pull/1384) 依旧属于未采纳的后继设计，只有未来经合法采纳后才能改变其规范权威。

---

## 10. ADS 与 Tally 来源导航

- [ADS 标准主库](https://github.com/kaicreator-mm/ai-development-standard/tree/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards) 与 [#1383 三级分层](https://github.com/kaicreator-mm/ai-development-standard/issues/1383)。
- [Tally v2.1 家族索引 #269](https://github.com/kaicreator-mm/tally/issues/269)、[v2.1.2 本版本 #521](https://github.com/kaicreator-mm/tally/issues/521)。
- [Tally #286 ADS→Tally Domain Package](https://github.com/kaicreator-mm/tally/issues/286)、[v2.1.1 实验 #299](https://github.com/kaicreator-mm/tally/issues/299)、[v2 Engineering Business/Workflow #276](https://github.com/kaicreator-mm/tally/issues/276)、[经验候选循环 #60](https://github.com/kaicreator-mm/tally/issues/60)。
- `K01/`–`K11/` 分别维护可扩展的知识正文；`K01.md`–`K11.md` 仅作为原始导论与兼容入口；各自模块 Issue 维护问题、修订与审核状态。任何实质变更应保持知识库与 Issue 同步；总 README 只维护跨模块索引和兼容性。

**最终区分：** `11 MODULE ISSUES PRESENT + DOCUMENT CANDIDATE` ≠ `11 KNOWLEDGE MODULES SOURCE-QUALIFIED` ≠ `TALLY DOMAIN PACKAGE QUALIFIED` ≠ `END-TO-END AGENT SOFTWARE DEVELOPMENT PASS`.
