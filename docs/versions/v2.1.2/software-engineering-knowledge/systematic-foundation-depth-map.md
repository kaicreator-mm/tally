# K01–K11 学科知识正文：R2/R3 两卷深度覆盖与未解决基础缺口

> [版本索引](README.md) · [165 点课程](knowledge-coverage-and-growth.md) · [系统架构](systematic-knowledge-architecture.md)
>
> **状态：R2_FIRST_VOLUME_INDEPENDENT_ACCEPT_SCOPED / R3_SECOND_VOLUME_REVIEW_PENDING / SOURCE_CLAIM_NOT_QUALIFIED / CONSUMER_NOT_CONNECTED。** 11 篇 R2 正文已在独立审查范围接受，R3 新增第二卷 11 篇仍待增量审核；**两卷共 22 篇原创深度章节**，并非 22 个生产 Package 或 165 条合格知识 Claim。

## 1. 双重导航

学科学习与持续研究采用 SWEBOK/相关学科 → Kxx → systematic-syllabus → foundations-deep-dive；真实工程任务采用 Goal → Kxx/README 的问题路由 → 必需 Topic/DecisionProcedure/Skill → 学科章节与适用来源。完整学科库与最小 Agent 上下文是不同要求。

## 2. 各模块的两卷系统学科深化正文

| 模块 | R2 第一卷（已审文档） | R3 第二卷（本轮候选） | 两卷主题互补性 |
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

**165 个唯一课程 ID 依然是系统纲要，而非 165 篇专业教材。** R3 另提供 [165 点逐项覆盖台账](foundation-concept-coverage-ledger.md)：每项说明只有课程解释 S、还是被第一卷 D1/第二卷 D2 明确关联。当前 **159/165 有至少一个深入章节主题关联，6/165 仍仅有课程解释**；“章节关联”并不意味着该点获得完整的源级资格或学科证明。原 26 个问题 Topic 继续作为工程任务入口。

## 3. 学科深化要达到的专业标准

后续每个新增学科章节必须有明确知识 ID/语义 Owner、可理解的原理、输入条件和模型（必要时含公式/状态）、多种可行方案及权衡、具体反例/可观察反驳、出处版本与权利说明、跨模块交接、当前 UNKNOWN 与 STOP。不能用填充式模板代替专业讲解，也不能只更新高价值问题候选而放弃基础理论。

层次状态应区分：课程基础（SYLLABUS_EXPLAINED_CANDIDATE）、深入原创正文（CHAPTER_DEEPENED_CANDIDATE）、逐条来源资格（SOURCE_CLAIM_CHECKED）、选定合法消费者真实验证（CONSUMER_VERIFIED）。本轮只实现前两类中的文档候选；这些是知识维护标签，不是新 ADS Gate 或 DomainHarness ABI。

## 4. 下一步与独立审核边界

按覆盖台账中六个仅有 S 的知识点和其余仍不充分的基础子主题继续遍历学科群，不以 26 个热点 Topic 为知识边界。保留唯一 Kxx Owner 与必要跨模块引用，选中生产用途后才对具体条款和真实 Agent/Host 场景逐项核验。R2 的独立 ACCEPT_SCOPED 对未变化的第一卷和课程可复用；R3 新第二卷、六组新增教学查询、来源增量与导航应做限定增量审核，不重复审核未变的教材。Tally #286 仍 DEFERRED；不声明 Harness 已连接、编译器已存在或 CI 实测。
