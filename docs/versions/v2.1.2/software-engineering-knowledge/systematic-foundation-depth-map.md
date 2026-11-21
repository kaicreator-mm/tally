# K01–K11 学科知识正文：第一轮深度覆盖与下一阶段缺口

> [版本索引](README.md) · [165 点课程](knowledge-coverage-and-growth.md) · [系统架构](systematic-knowledge-architecture.md)
>
> **状态：DOCUMENT_BODY_CANDIDATE / NEW_DELTA_REVIEW_PENDING / SOURCE_CLAIM_NOT_QUALIFIED / CONSUMER_NOT_CONNECTED。** 11 篇新文档是对既有课程的深写，不是 11 个生产 Package 或 165 条完成资格的知识 Claim。

## 1. 双重导航

学科学习与持续研究采用 SWEBOK/相关学科 → Kxx → systematic-syllabus → foundations-deep-dive；真实工程任务采用 Goal → Kxx/README 的问题路由 → 必需 Topic/DecisionProcedure/Skill → 学科章节与适用来源。完整学科库与最小 Agent 上下文是不同要求。

## 2. 本轮 11 篇原创学科深化正文

| 模块 | 深化章节 | 正文专业机制 | 尚待深化的基础学科空白 |
| --- | --- | --- | --- |
| K01 需求与业务行为 | [系统深化 01](K01/foundations-deep-dive-01.md) | 状态机、政策谓词、聚合/验收 | 需求采样偏差与可测质量需求 |
| K02 架构与一致性 | [系统深化 01](K02/foundations-deep-dive-01.md) | 质量场景、状态所有权、恢复性研究 | 依赖倒置与演进式架构 |
| K03 算法与构造 | [系统深化 01](K03/foundations-deep-dive-01.md) | 数据结构成本、相等性、条件写 | 类型系统、内存与背压 |
| K04 任务分解 | [系统深化 01](K04/foundations-deep-dive-01.md) | 因果 DAG、写集、关键路径 | 估算不确定性与过程反馈 |
| K05 测试与验证 | [系统深化 01](K05/foundations-deep-dive-01.md) | Oracle、性质/状态/变异测试 | 组合覆盖与统计模型 |
| K06 调试与因果 | [系统深化 01](K06/foundations-deep-dive-01.md) | 竞争假设、复现、探针/回归 | 复杂分布式故障时序 |
| K07 Agent 协作 | [系统深化 01](K07/foundations-deep-dive-01.md) | ACK 与 Task 来源、乱序、写集 | 认知偏差与协作反馈 |
| K08 构建与交付 | [系统深化 01](K08/foundations-deep-dive-01.md) | 可复现输入、缓存、CI 阶段 | 制品矩阵与部署演进 |
| K09 软件演进 | [系统深化 01](K09/foundations-deep-dive-01.md) | Expand–Migrate–Contract、旧数据 | 技术债与退役 |
| K10 可靠性与性能 | [系统深化 01](K10/foundations-deep-dive-01.md) | Timeout、Effect Unknown、背压 | 容量与性能统计 |
| K11 经济与决策 | [系统深化 01](K11/foundations-deep-dive-01.md) | 同任务对照、成本与独立 Oracle | 统计置信与样本偏差 |

本阶段 **165 个唯一课程 ID 依然是全范围系统纲要**；11 篇新章节覆盖了部分知识群的专业机制，不代表每个知识点拥有独立教材深度。原有 26 个问题 Topic 保持有效；不因为扩写课程而复制 Topic 内容或新建工作流。

## 3. 学科深化要达到的专业标准

后续每个新增学科章节必须有明确知识 ID/语义 Owner、可理解的原理、输入条件和模型（必要时含公式/状态）、多种可行方案及权衡、具体反例/可观察反驳、出处版本与权利说明、跨模块交接、当前 UNKNOWN 与 STOP。不能用填充式模板代替专业讲解，也不能只更新高价值问题候选而放弃基础理论。

层次状态应区分：课程基础（SYLLABUS_EXPLAINED_CANDIDATE）、深入原创正文（CHAPTER_DEEPENED_CANDIDATE）、逐条来源资格（SOURCE_CLAIM_CHECKED）、选定合法消费者真实验证（CONSUMER_VERIFIED）。本轮只实现前两类中的文档候选；这些是知识维护标签，不是新 ADS Gate 或 DomainHarness ABI。

## 4. 下一步与独立审核边界

按上表的专业基础缺口继续遍历学科群，不以 26 个热点 Topic 为知识边界。保留唯一 Kxx Owner 与必要跨模块引用，选中生产用途后才对具体条款和真实 Agent/Host 场景逐项核验。R1 的旧 HEAD 文档 ACCEPT_SCOPED 可复用；R2 新章、P2 关系精度修正与导航应做限定增量审核，不重复审核未变的教材。Tally #286 仍 DEFERRED；不声明 Harness 已连接、编译器已存在或 CI 实测。
