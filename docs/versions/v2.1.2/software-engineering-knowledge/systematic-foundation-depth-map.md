# K01–K11 学科知识正文：R2/R3 两卷与 R4 六点补缺

> [版本索引](README.md) · [165 点课程](knowledge-coverage-and-growth.md) · [系统架构](systematic-knowledge-architecture.md)
>
> **状态：R2_FIRST_VOLUME_ACCEPT_SCOPED / R3_SECOND_VOLUME_ACCEPT_SCOPED / R4_SIX_POINT_FOCUS_REVIEW_PENDING / SOURCE_CLAIM_NOT_QUALIFIED / CONSUMER_NOT_CONNECTED。** R2 第一卷、R3 第二卷各 11 篇已获独立文档限定接受；R4 新增 6 篇聚焦补缺文章待审核。共 **22 篇两卷系统深化 + 6 篇聚焦正文**，不是 28 个生产 Package 或已合格的 165 条 Claim。

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

**165 个唯一课程 ID 仍是系统纲要，不是 165 篇专业教材。** [逐项覆盖台账](foundation-concept-coverage-ledger.md) 保留 R3 基线：159/165 与两卷学科正文有主题关联，原 6/165 仅 S。R4 为这六项分别新增 F 聚焦正文，形成 **165/165 至少有 D1/D2 或 F 深化正文导航**，但 R4 六项未独立审核；“有正文关联”并不等于 165 条 Source-qualified Claims 或 165 项教材级完整性。26 个问题 Topic 仍是任务入口。

## R4：六点聚焦补缺（待独立审核）

| 原基础缺口 | 新增专业正文 | 学科机制 |
| --- | --- | --- |
| K02.F4.3 | [最低信任/失效边界](K02/focus-f4-3-minimal-failure-and-trust-boundary.md) | 数据流、租户/权限隔离、架构最小威胁意识 |
| K04.F2.3 | [任务接受与合法授权](K04/focus-f2-3-task-acceptance-and-authority.md) | Writer DONE、Evidence 与合法 Owner 的分离 |
| K05.F3.3 | [Build/Package/平台验证](K05/focus-f3-3-build-package-platform-verification.md) | 构建、打包、安装、首次运行的不同 Oracle |
| K05.F4.2 | [Golden 权威与证据时效](K05/focus-f4-2-golden-snapshot-evidence-freshness.md) | Expected 来源、旧 SHA、快照错误更新 |
| K06.F4.3 | [补丁精确当前性与复核](K06/focus-f4-3-patch-currentness-and-review.md) | Git 影响闭包、受影响回归、增量证据 |
| K10.F5.1 | [首次运行与配置策略](K10/focus-f5-1-first-run-post-start-configuration.md) | GUI/CLI/Headless 的启动与权限态 |

本轮只关闭“没有任何深入正文重点覆盖”的六个导航缺口，不改变其他 159 个知识点仍需持续专业深化、来源核验和实际 Domain Intelligence 消费能力验证的事实。

## 3. 学科深化要达到的专业标准

后续每个新增学科章节必须有明确知识 ID/语义 Owner、可理解的原理、输入条件和模型（必要时含公式/状态）、多种可行方案及权衡、具体反例/可观察反驳、出处版本与权利说明、跨模块交接、当前 UNKNOWN 与 STOP。不能用填充式模板代替专业讲解，也不能只更新高价值问题候选而放弃基础理论。

层次状态应区分：课程基础（SYLLABUS_EXPLAINED_CANDIDATE）、深入原创正文（CHAPTER_DEEPENED_CANDIDATE）、逐条来源资格（SOURCE_CLAIM_CHECKED）、选定合法消费者真实验证（CONSUMER_VERIFIED）。本轮只实现前两类中的文档候选；这些是知识维护标签，不是新 ADS Gate 或 DomainHarness ABI。

## 4. 下一步与独立审核边界

R4 六点聚焦后，按覆盖台账与原 18 KA 结构识别其他仍不充分的基础子主题继续深化，不以数量凑文件，不以 26 个热点 Topic 为知识边界。保留唯一 Kxx Owner 与必要跨模块引用，选中生产用途后才对具体条款和真实 Agent/Host 场景逐项核验。R3 独立 ACCEPT_SCOPED 对未变的两卷和基础课程可复用；R4 六点聚焦正文、61 来源中新增 A55–A61、18 案例中新增 Q13–Q18 与关系增量仅做限定审核，不重复审核未变的教材。Tally #286 仍 DEFERRED；不声明 Harness 已连接、编译器已存在或 CI 实测。
