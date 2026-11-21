# K01–K11 学科知识正文：R2–R8 系统学科与进阶推理

> [版本索引](README.md) · [165 点课程](knowledge-coverage-and-growth.md) · [系统架构](systematic-knowledge-architecture.md)
>
> **状态：R2_R3_R4_R5_R6_R7_INDEPENDENT_ACCEPT_SCOPED_PRIOR_EXACT_SHAS / R8_SELECTED_14_PRIORITY_CONCEPTS_REVIEW_PENDING / SOURCE_CLAIM_NOT_QUALIFIED / CONSUMER_NOT_CONNECTED。** R2 第一卷、R3 第二卷各 11 篇、R4 六篇聚焦正文均已获得旧 Exact SHA 的独立文档限定接受；R5 的 4 篇进阶正文和 5 条完整演算已在旧 Exact SHA 完成文档限定接受，R6 的 7 篇进阶正文与 W06–W12 已在旧 Exact SHA 独立限定接受；R7 的 3 篇基础学理正文与 87 点差距盘点已在旧 Exact SHA `a307e0fab92ae3b857b5098d16200b339ffda764` 获独立限定接受；R8 新增 5 篇专业决策深化正文与五条 W13–W17 演算仍待增量审核。现有 **22 篇两卷系统深化 + 6 篇 R4 聚焦正文 + 4 篇 R5 深层原理正文 + 7 篇 R6 进阶理论正文 + 3 篇 R7 专项基础理论正文 + 5 篇 R8 专业深化正文**，不是已投入生产的 Package 或源级合格 Claim。

## 1. 双重导航

学科学习与持续研究采用 SWEBOK/相关学科 → Kxx → systematic-syllabus → foundations-deep-dive；真实工程任务采用 Goal → Kxx/README 的问题路由 → 必需 Topic/DecisionProcedure/Skill → 学科章节与适用来源。完整学科库与最小 Agent 上下文是不同要求。

## 2. 各模块的两卷系统学科深化正文

| 模块 | R2 第一卷（已审文档） | R3 第二卷（已审文档） | 两卷主题互补性 |
| --- | --- | --- | --- |
| K01 需求工程 | [01](K01/foundations-deep-dive-01.md) · 业务 Rule/状态模型 | [02](K01/foundations-deep-dive-02.md) · 需求样本偏差/质量场景 | 第二卷扩充第一卷尚未展开的学科前提 |
| K02 系统架构 | [01](K02/foundations-deep-dive-01.md) · 质量属性/状态所有权 | [02](K02/foundations-deep-dive-02.md) · 依赖方向/分布式状态 | 第二卷扩充第一卷尚未展开的学科前提 |
| K03 详细设计 | [01](K03/foundations-deep-dive-01.md) · 算法/结构与一致性 | [02](K03/foundations-deep-dive-02.md) · 类型/取消/背压 | 第二卷扩充第一卷尚未展开的学科前提 |
| K04 任务工程 | [01](K04/foundations-deep-dive-01.md) · 因果 DAG/资源竞争 | [02](K04/foundations-deep-dive-02.md) · 估算/反馈/停止规则 | 第二卷扩充第一卷尚未展开的学科前提 |
| K05 软件测试 | [01](K05/foundations-deep-dive-01.md) · Oracle/性质和变异 | [02](K05/foundations-deep-dive-02.md) · 组合覆盖/状态探索 | 第二卷扩充第一卷尚未展开的学科前提 |
| K06 诊断修复 | [01](K06/foundations-deep-dive-01.md) · 第一失败/竞争假设 | [02](K06/foundations-deep-dive-02.md) · 分布式时间线/事故因果 | 第二卷扩充第一卷尚未展开的学科前提 |
| K07 多 Agent 协作 | [01](K07/foundations-deep-dive-01.md) · ACK/Attempt/旧 HEAD | [02](K07/foundations-deep-dive-02.md) · 人因/交接/独立审查 | 第二卷扩充第一卷尚未展开的学科前提 |
| K08 构建交付 | [01](K08/foundations-deep-dive-01.md) · CI 阶段/制品来源 | [02](K08/foundations-deep-dive-02.md) · 可复现构建/缓存因果 | 第二卷扩充第一卷尚未展开的学科前提 |
| K09 软件演进 | [01](K09/foundations-deep-dive-01.md) · 接口兼容/数据迁移 | [02](K09/foundations-deep-dive-02.md) · 技术债/退役决策 | 第二卷扩充第一卷尚未展开的学科前提 |
| K10 可靠性性能 | [01](K10/foundations-deep-dive-01.md) · Timeout/Effect Unknown | [02](K10/foundations-deep-dive-02.md) · 尾延迟/Little 定律/测量 | 第二卷扩充第一卷尚未展开的学科前提 |
| K11 经济和决策 | [01](K11/foundations-deep-dive-01.md) · 成本/对照基线 | [02](K11/foundations-deep-dive-02.md) · 统计推断/投资停止 | 第二卷扩充第一卷尚未展开的学科前提 |

**165 个唯一课程 ID 仍是系统纲要，不是 165 篇专业教材。** [逐项覆盖台账](foundation-concept-coverage-ledger.md) 保留 R3 基线：159/165 与两卷学科正文有主题关联，原 6/165 仅 S。R4 为这六项分别新增 F 聚焦正文，形成 **165/165 至少有 D1/D2 或 F 深化正文导航**，而 R4 的六篇 Focus 已于旧 Exact SHA `42e7ba8bd883ec79736e25ff7c75bdbda1b738b3` 独立 `ACCEPT_SCOPED`（K02 的 R5 小范围术语修正亦已独审确认）；“有正文关联”并不等于 165 条 Source-qualified Claims 或 165 项教材级完整性。26 个问题 Topic 仍是任务入口。

## R4：六点聚焦补缺（旧 SHA 已审）

| 原基础缺口 | 新增专业正文 | 学科机制 |
| --- | --- | --- |
| K02.F4.3 | [最低信任/失效边界](K02/focus-f4-3-minimal-failure-and-trust-boundary.md) | 数据流、租户/权限隔离、架构最小威胁意识 |
| K04.F2.3 | [任务接受与合法授权](K04/focus-f2-3-task-acceptance-and-authority.md) | Writer DONE、Evidence 与合法 Owner 的分离 |
| K05.F3.3 | [Build/Package/平台验证](K05/focus-f3-3-build-package-platform-verification.md) | 构建、打包、安装、首次运行的不同 Oracle |
| K05.F4.2 | [Golden 权威与证据时效](K05/focus-f4-2-golden-snapshot-evidence-freshness.md) | Expected 来源、旧 SHA、快照错误更新 |
| K06.F4.3 | [补丁精确当前性与复核](K06/focus-f4-3-patch-currentness-and-review.md) | Git 影响闭包、受影响回归、增量证据 |
| K10.F5.1 | [首次运行与配置策略](K10/focus-f5-1-first-run-post-start-configuration.md) | GUI/CLI/Headless 的启动与权限态 |

R4 独立审核已对这一文档范围接受；K02.focus 的 Extension Import 与 HostPort/Effect 语义分离已在 R5 独立审核确认修正；此处不授予运行时 Effect 权限。本轮只关闭“没有任何深入正文重点覆盖”的六个导航缺口，不改变其他 159 个知识点仍需持续专业深化、来源核验和实际 Domain Intelligence 消费能力验证的事实。

## R5：四篇深层专业推理与五条完整演算（旧 SHA 已审）

与 R2/R3 的学科广度文章、R4 的单点补缺不同，R5 聚焦**已有专业概念的推理深度和错误诊断能力**，不追求一次填充 165 份同样模板：

| 领域 Owner | R5 原创深化正文 | 显式映射课程 ID | 关键专业演算 |
| --- | --- | --- | --- |
| K01 需求事实与规范合同 | [证据→合同→状态验收](K01/advanced-knowledge-from-evidence-to-contract.md) | 6 点 | 同条件政策冲突、并发发货/取消反例 |
| K03 实现与并发语义 | [线性化/幂等/崩溃矩阵](K03/advanced-concurrency-idempotency-proof-sketch.md) | 5 点 | CAS、远端未知副作用、队列重放 |
| K05 Oracle 与测试充分性 | [独立 Expected/组合覆盖/区分性测试](K05/advanced-oracle-adequacy-and-test-selection.md) | 7 点 | Pairwise 无法覆盖业务三元权限；Mock/发现/验证分离 |
| K11 真实价值与实证判断 | [受控对照/混杂/总成本](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) | 7 点 | 8/10 vs 6/10 的观察差异不等于因果收益 |

共 25 个原已存在的学科 ID 在 [165 点台账](foundation-concept-coverage-ledger.md) 中被额外标记 A；其余 140 个知识点没有被自动升级为“已完全深入”或“已得到证实”。同样重要的是 [W01–W05 五条跨模块完整推理演算](professional-reasoning-worked-traces.md)：统一演示怎样处理事实/来源/条件、知识选择、备选机制、最小反例、UNKNOWN/STOP 与真实 Owner。五个 W 是**教学场景**，不与旧 Q01–Q18 合并计数，也不构成实际 Agent Bench/E2E。

**深度判断不应简化为链接数**：一个概念即使有 S+D1+D2+A，仍需对实际命题、适用范围、解释的反例和来源资格独立判断。R5/R6 文件仍是原创专业推导与教学设计，绝不自动创建 Tally 当前 Source Claim 或 DomainHarness ABI。

## R6：七模块原理深化与 W06–W12（旧 SHA 已审）

R5 的 K01/K03/K05/K11 通过后，R6 将**剩下七个模块**也各自建立一个原创、条件明确、有计算/故障反例和 STOP 的进阶专业章节，完成 11/11 模块的至少一篇进阶学理正文。原有 165 个概念仍保留，不新造平行知识分类。

| 模块 | 进阶正文 | 选中现有课程 ID 数 | 深入推演与反例 |
| --- | --- | --- | --- |
| K02 | [质量场景、状态所有权、失效与信任权衡](K02/advanced-architecture-invariants-and-failure-tradeoffs.md) | 7 | 本地事务无法保证外部 Effect；Tenant cacheKey 冲突 |
| K04 | [Task 偏序、写集冲突与关键路径](K04/advanced-task-graph-critical-path-and-bounded-delivery.md) | 7 | 教学 DAG 理想 8h 下界与资源/合同冲突 |
| K06 | [根因竞争假设与区分性探针](K06/advanced-incident-causality-and-distinguishing-probes.md) | 7 | CI cache hit 不是缓存缺陷根因的直接证明 |
| K07 | [Attempt/Source SHA/共享写集与独立复核](K07/advanced-agent-concurrency-provenance-and-review-isolation.md) | 7 | H1 迟到 ACK 不能覆盖 H2 当前授权 |
| K08 | [Hermetic Build、单次可信源码获取与平台验收](K08/advanced-hermetic-ci-artifact-identity-and-target-host.md) | 8 | 理论 19 × 300MiB = 5,700MiB ≈ 5.57GiB；需真实 Provider 测量 |
| K09 | [消费者版本合同、数据迁移与不兼容重写](K09/advanced-contract-evolution-migration-and-retirement.md) | 8 | Expand/Contract 条件失效、有损业务状态转移 |
| K10 | [SLI/SLO、告警燃烧率、未知 Effect 与性能](K10/advanced-reliability-slo-effect-recovery-and-measurement.md) | 9 | 99.9%/100,000 次示例的错误预算，以及重试/熔断风险 |

七篇进阶正文关联 **53 个原知识 ID**，与 R5 25 个 A 合计 **78/165 个 A 进阶导航点**。不是 78 条独立 Source Claim、78 项完全掌握的教材章节或真实运行 Agent。配套 [W06–W12 七条端到端教学推理](professional-reasoning-worked-traces-02.md) 和此前 W01–W05（共 12 条 W）用于检验综合选择知识的能力，仍全部 **SYNTHETIC_NOT_RUN**。

R6 仅增 [A62 Bazel Hermeticity / A63 Google SRE SLO / A64 Danilo Sato Parallel Change（Martin Fowler 网站）](external-source-catalog.md) 三项公开正式/作者原始来源线索，R6 当时为 **64 条参考**，R7 另增三项正式学科参考线索，当前合计 67 条。来源可定位/适用不等于版权复制批准、原始 Claim 获权或 Harness Compiler ABI 决议。

## R7：对剩余 87 个知识点先做真实缺口盘点，再有选择地深化（旧 SHA 已审）

在 R6 旧 SHA 已审知识的基线上，原有 **78/165** 课程点包含额外 A 进阶原理链接，剩余 **87 点没有 A**。本轮并未把 87 点当作无知识，而是对每个原课程 ID 审核其现有 S/D1/D2/F 导航与工程深度要求，在 [87 项真实差距盘点](professional-depth-gap-inventory-r7.md) 分为 **15 项本轮深入、30 项下一优先、42 项随具体任务深化**，不制造同质化文档。

| 学科 | R7 新增专业学理正文 | 本轮新增精确 A 课程点 | 深化的真实理论问题 |
| --- | --- | --- | --- |
| K01 | [需求双向追溯、聚合与变更传播](K01/advanced-requirements-traceability-and-change-propagation.md) | 4（F1.1、F3.1、F4.2、F5.3） | 目标≠方案、Goal→Rule→Code→Oracle、业务不变量与语义影响 |
| K03 | [算法合同、循环不变量、类型与形式证明边界](K03/advanced-algorithm-contracts-types-and-formal-claims.md) | 5（F1.1、F1.2、F1.3、F2.2、F4.3） | 有条件的前/后置与终止、排序前提、静态类型不等于外部 Effect |
| K05 | [Verification/Validation、测试层与 Oracle 资格](K05/advanced-verification-validation-and-oracle-qualification.md) | 6（F1.3、F2.1、F3.1、F3.2、F4.3、F5.3） | 正确实现错误规格、等价类与高阶反例、Mock/真实环境与 Flaky |

R7 新增 **15 个原课程点的 A 链接**，在旧 78 点上达到 **93/165**，另 **72 个点尚无 A**。这属于专业正文导航与课程内容的增量，**不是 93 条已通过产品证明的 Source Claim、不是 93 个机器形式化定理、也不是实际运行的 15 次 Agent 实验**。新来源 A65–A67（NASA 追溯、Dafny 规格/验证、NIST 组合测试）用于原理定位，未授权复制受限制的规范全文，未启动新的 Tally/Harness Runtime ABI。

## R8：五个实质决策学科领域的高优先级深化（待独立审核）

R8 沿用 R7 已受审的 [87 点差距分流](professional-depth-gap-inventory-r7.md)，从当时 30 个 `NEXT_HIGH_VALUE` 里只选择真正跨层错误机制尚需补足的 **14 个原 ID**。具体学科 Owner、课程关联和论文是：

| 原模块 | R8 原创新专业正文 | 本轮新增 A | 新的可证伪机制与应用边界 |
| --- | --- | --- | --- |
| K01 | [Stakeholder 冲突/证据三角核验](K01/advanced-stakeholder-authority-and-triangulated-evidence.md) | F1.2、F2.1（2） | 规范授权与观察证据非同类；旧版/不同 Tenant 不能投票决定真实政策 |
| K02 | [依赖/Host/数据/Consumer 四种边界](K02/advanced-extension-authority-state-and-consumer-boundaries.md) | F1.3、F2.3、F3.3、F5.1（4） | A→B→C 不代替 A 直接声明；BUILD 合法不等于有权 Host；本地 Commit 非远端 Effect |
| K07 | [角色/Context/协作成果](K07/advanced-agent-role-context-and-outcome-evaluation.md) | F2.1、F3.3、F5.2（3） | Writer 自审与独立审查不同；最小充分知识和 Token 经济受实际 Oracle 约束 |
| K08 | [Release Provenance/权限/回滚](K08/advanced-release-provenance-permission-and-rollback.md) | F5.1、F5.2（2） | H1/D1 验证不可冒充 D2 Release；代码回滚不一定恢复数据/真实 Effect |
| K11 | [估算/风险/信息价值/流程成本](K11/advanced-decision-risk-value-of-information-and-process-cost.md) | F2.1、F3.1、F5.1（3） | 12h 工作量与 8h 理论关键路径不同；高后果低频风险、信息调查的可证伪条件 |

**原 93 A + R8 新增 14 A = 107/165，58 点无 A**，与 R7 历史分流的 `15 + 30 + 42 = 87` 不冲突：R8 从历史 30 中选了 14，剩余 16 加 42 等于 58。本轮同时新增 [W13–W17 五条跨学科真实条件推理教学案例](professional-reasoning-worked-traces-03.md) 与 [原始来源→命题→反例适用矩阵](professional-source-applicability-r8.md)；W01–W17 均为**合成、未运行**，不代表模型成绩。A68 SLSA v1.2 Provenance/A69 DORA 官方度量更新仅是两项原始参考发现，当前 69 条来源尚未取得原始 Claim 或复制全文许可。

## 3. 学科深化要达到的专业标准

后续每个新增学科章节必须有明确知识 ID/语义 Owner、可理解的原理、输入条件和模型（必要时含公式/状态）、多种可行方案及权衡、具体反例/可观察反驳、出处版本与权利说明、跨模块交接、当前 UNKNOWN 与 STOP。不能用填充式模板代替专业讲解，也不能只更新高价值问题候选而放弃基础理论。

层次状态应区分：课程基础（SYLLABUS_EXPLAINED_CANDIDATE）、深入原创正文（CHAPTER_DEEPENED_CANDIDATE）、逐条来源资格（SOURCE_CLAIM_CHECKED）、选定合法消费者真实验证（CONSUMER_VERIFIED）。本轮只实现前两类中的文档候选；这些是知识维护标签，不是新 ADS Gate 或 DomainHarness ABI。

## 4. 下一步与独立审核边界

R4 六点聚焦后，按覆盖台账与原 18 KA 结构识别其他仍不充分的基础子主题继续深化，不以数量凑文件，不以 26 个热点 Topic 为知识边界。保留唯一 Kxx Owner 与必要跨模块引用，选中生产用途后才对具体条款和真实 Agent/Host 场景逐项核验。R1–R7 的独立 ACCEPT_SCOPED 仅在旧 SHA 未改内容和限定范围内可复用；R8 只复审五篇新专业深化正文、W13–W17、来源适用矩阵、14 处新 A 链接、A68–A69 发现及相关导航增量，不重复审核以前未变化的专业正文。Tally #286 仍 DEFERRED；不声明 Harness 已连接、编译器已存在或 CI 实测。
