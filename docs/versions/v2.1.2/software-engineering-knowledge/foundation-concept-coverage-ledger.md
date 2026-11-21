# 软件工程系统知识库：165 个知识点的可审计覆盖台账（R5）

> [知识库总索引](README.md) · [第一/第二卷学科正文](systematic-foundation-depth-map.md) · [学科结构](knowledge-coverage-and-growth.md)
>
> **R3_LEDGER_INDEPENDENT_ACCEPT_SCOPED / R4_SIX_FOCUS_DOCUMENT_ACCEPT_SCOPED / R5_ADVANCED_REASONING_DELTA_REVIEW_PENDING / NOT_A_MACHINE_ABI / SOURCE_CLAIM_NOT_QUALIFIED / CONSUMER_NOT_CONNECTED。** 本台账是知识维护和导航的文档事实，不是 Rule/Agent Skill 的生产资格。ID、标题来自真实 11 份 `systematic-syllabus.md`；章节覆盖由写作者明确标注的**主题关联范围**建立，不是自动证明某概念已得到完整教材级论证。

## 阅读本表时的五个重要区分

- **S**：每个知识点都已在原系统课程中解释概念、机制与误用，表示课程稿件存在，**不代表规范 Claim 已验收**。
- **D1/D2**：第一卷/第二卷正文明确讨论或作为主线涉及该知识点；一行带两个链接仍不能宣称该知识有两个独立已证明的 Source Claim。
- **F**：R4 为原先六个仅有课程解释的点新增**逐项聚焦原创正文**，覆盖对象、方法、适用前提、反例、STOP；R4 旧 Exact SHA 的文档限定审核已接受，R5 如有微修仅审实际差异。
- **A**：R5 为 K01/K03/K05/K11 的 25 个**被明确深入讨论的知识点**链接新的进阶原理/专业证伪文章；这些链接和正文尚待 R5 独立审核，不能将主题关联记为该知识点获得正式专业认证。
- **仅 S**：当前 R4 已没有仅 S 的点，但不能以此证明 165 点达到同等深度。159 个只有 D1/D2 的主题关联深度并未因此变成逐项教材级证明。

## 全量课程、两卷正文、R4 六篇聚焦及 R5 进阶原理的对应关系

| Knowledge ID | 课程原概念 | 稿件级知识覆盖 |
| --- | --- | --- |
| `K01.F1.1` | Need/Goal 与 Solution | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) |
| `K01.F1.2` | Stakeholder 与利益冲突 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [D2](K01/foundations-deep-dive-02.md) |
| `K01.F1.3` | 业务边界与领域语言 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) |
| `K01.F2.1` | 访谈/观察/文档三角验证 | [S](K01/systematic-syllabus.md) · [D2](K01/foundations-deep-dive-02.md) |
| `K01.F2.2` | 事实/假设/未知分类 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [D2](K01/foundations-deep-dive-02.md) · [A](K01/advanced-knowledge-from-evidence-to-contract.md) |
| `K01.F2.3` | 需求非功能影响 | [S](K01/systematic-syllabus.md) · [D2](K01/foundations-deep-dive-02.md) |
| `K01.F3.1` | 实体/值对象/聚合边界 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) |
| `K01.F3.2` | 状态机/业务事件 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [A](K01/advanced-knowledge-from-evidence-to-contract.md) |
| `K01.F3.3` | 业务不变量与可推理约束 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [A](K01/advanced-knowledge-from-evidence-to-contract.md) |
| `K01.F4.1` | 功能合同与边界用例 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [A](K01/advanced-knowledge-from-evidence-to-contract.md) |
| `K01.F4.2` | 需求可追溯链 | [S](K01/systematic-syllabus.md) · [D2](K01/foundations-deep-dive-02.md) |
| `K01.F4.3` | 质量场景/优先级 | [S](K01/systematic-syllabus.md) · [D2](K01/foundations-deep-dive-02.md) |
| `K01.F5.1` | 同条件冲突判定 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [A](K01/advanced-knowledge-from-evidence-to-contract.md) |
| `K01.F5.2` | 验收与反例 | [S](K01/systematic-syllabus.md) · [D1](K01/foundations-deep-dive-01.md) · [A](K01/advanced-knowledge-from-evidence-to-contract.md) |
| `K01.F5.3` | 变更影响和冻结权威 | [S](K01/systematic-syllabus.md) · [D2](K01/foundations-deep-dive-02.md) |
| `K02.F1.1` | 架构与实现的不同抽象层 | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) |
| `K02.F1.2` | Stakeholder / Concern / Viewpoint | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F1.3` | 架构决策记录 ADR | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F2.1` | 模块/组件边界 | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F2.2` | 接口契约与错误语义 | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F2.3` | 依赖方向与扩展点 | [S](K02/systematic-syllabus.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F3.1` | 状态所有权与事务 | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) |
| `K02.F3.2` | 一致性、事件和幂等 | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F3.3` | 持久化和数据模型 | [S](K02/systematic-syllabus.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F4.1` | 质量属性场景 | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) |
| `K02.F4.2` | 架构权衡与可证伪 Demo | [S](K02/systematic-syllabus.md) · [D1](K02/foundations-deep-dive-01.md) |
| `K02.F4.3` | 威胁/失败模式的最小考虑 | [S](K02/systematic-syllabus.md) · [F](K02/focus-f4-3-minimal-failure-and-trust-boundary.md) |
| `K02.F5.1` | 版本/兼容边界 | [S](K02/systematic-syllabus.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F5.2` | 结构化可维护架构说明 | [S](K02/systematic-syllabus.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K02.F5.3` | 架构适用性失效与回评 | [S](K02/systematic-syllabus.md) · [D2](K02/foundations-deep-dive-02.md) |
| `K03.F1.1` | 复杂度/成本模型 | [S](K03/systematic-syllabus.md) · [D1](K03/foundations-deep-dive-01.md) |
| `K03.F1.2` | 数据结构与索引 | [S](K03/systematic-syllabus.md) · [D1](K03/foundations-deep-dive-01.md) |
| `K03.F1.3` | 算法正确性与边界 | [S](K03/systematic-syllabus.md) · [D1](K03/foundations-deep-dive-01.md) |
| `K03.F2.1` | 模块/接口/内聚 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K03.F2.2` | 类型系统与数据验证 | [S](K03/systematic-syllabus.md) · [D1](K03/foundations-deep-dive-01.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K03.F2.3` | 可读性与可维护性 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K03.F3.1` | 状态机/条件更新 | [S](K03/systematic-syllabus.md) · [D1](K03/foundations-deep-dive-01.md) · [D2](K03/foundations-deep-dive-02.md) · [A](K03/advanced-concurrency-idempotency-proof-sketch.md) |
| `K03.F3.2` | 幂等/重试/取消 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) · [A](K03/advanced-concurrency-idempotency-proof-sketch.md) |
| `K03.F3.3` | 异步任务与背压 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) · [A](K03/advanced-concurrency-idempotency-proof-sketch.md) |
| `K03.F4.1` | 数据库和事务实现 | [S](K03/systematic-syllabus.md) · [D1](K03/foundations-deep-dive-01.md) · [D2](K03/foundations-deep-dive-02.md) · [A](K03/advanced-concurrency-idempotency-proof-sketch.md) |
| `K03.F4.2` | 文件/网络/进程资源 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) · [A](K03/advanced-concurrency-idempotency-proof-sketch.md) |
| `K03.F4.3` | 依赖/生成代码边界 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K03.F5.1` | 异常分类和可行动错误 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K03.F5.2` | 日志与敏感性 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K03.F5.3` | 真正的编译与受影响测试 | [S](K03/systematic-syllabus.md) · [D2](K03/foundations-deep-dive-02.md) |
| `K04.F1.1` | 生命周期过程 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F1.2` | 迭代/增量与反馈 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F1.3` | 复杂度与风险驱动流程 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F2.1` | Work Unit/Task Outcome | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) |
| `K04.F2.2` | 问题分解与组合 | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) |
| `K04.F2.3` | 验收与授权 | [S](K04/systematic-syllabus.md) · [F](K04/focus-f2-3-task-acceptance-and-authority.md) |
| `K04.F3.1` | 真实因果依赖 | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) |
| `K04.F3.2` | 共享写集/冲突控制 | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) |
| `K04.F3.3` | 关键路径与资源约束 | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F4.1` | 事实缺口/研究目标 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F4.2` | 变更影响与重规划 | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F4.3` | 计划中的风险预算 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F5.1` | 实际进度的事实来源 | [S](K04/systematic-syllabus.md) · [D1](K04/foundations-deep-dive-01.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F5.2` | 复盘与实验对比 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K04.F5.3` | 停止策略与交付首位 | [S](K04/systematic-syllabus.md) · [D2](K04/foundations-deep-dive-02.md) |
| `K05.F1.1` | 测试与证明的范围 | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F1.2` | 业务来源驱动 Oracle | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F1.3` | 验证与确认 V&V | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) |
| `K05.F2.1` | 等价类与边界值 | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [D2](K05/foundations-deep-dive-02.md) |
| `K05.F2.2` | 决策表与状态转换 | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [D2](K05/foundations-deep-dive-02.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F2.3` | 性质/组合/变异测试 | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [D2](K05/foundations-deep-dive-02.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F3.1` | Unit/Contract/Integration/E2E | [S](K05/systematic-syllabus.md) · [D2](K05/foundations-deep-dive-02.md) |
| `K05.F3.2` | 依赖环境与隔离 | [S](K05/systematic-syllabus.md) · [D2](K05/foundations-deep-dive-02.md) |
| `K05.F3.3` | Build/Package 与平台 | [S](K05/systematic-syllabus.md) · [F](K05/focus-f3-3-build-package-platform-verification.md) |
| `K05.F4.1` | 实际发现/执行 | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F4.2` | Snapshot/Golden 和过期证据 | [S](K05/systematic-syllabus.md) · [F](K05/focus-f4-2-golden-snapshot-evidence-freshness.md) |
| `K05.F4.3` | Flaky 测试与诊断 | [S](K05/systematic-syllabus.md) · [D2](K05/foundations-deep-dive-02.md) |
| `K05.F5.1` | 风险驱动优先级 | [S](K05/systematic-syllabus.md) · [D2](K05/foundations-deep-dive-02.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F5.2` | 覆盖与变异有效性 | [S](K05/systematic-syllabus.md) · [D1](K05/foundations-deep-dive-01.md) · [D2](K05/foundations-deep-dive-02.md) · [A](K05/advanced-oracle-adequacy-and-test-selection.md) |
| `K05.F5.3` | 测试结果与权威 | [S](K05/systematic-syllabus.md) · [D2](K05/foundations-deep-dive-02.md) |
| `K06.F1.1` | 现象与原因分离 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) |
| `K06.F1.2` | 第一失败点与失败传播 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F1.3` | 环境/版本同一性 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F2.1` | 最小复现 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) |
| `K06.F2.2` | 竞争假设区分 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F2.3` | 调试工具与观测 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F3.1` | 确定性 Bug 与基础设施缺陷 | [S](K06/systematic-syllabus.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F3.2` | 超时/未知副作用 | [S](K06/systematic-syllabus.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F3.3` | 跨模块/集成缺陷 | [S](K06/systematic-syllabus.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F4.1` | 最小合法改动 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) |
| `K06.F4.2` | 回归与反事实 | [S](K06/systematic-syllabus.md) · [D1](K06/foundations-deep-dive-01.md) |
| `K06.F4.3` | 补丁当前性与复核 | [S](K06/systematic-syllabus.md) · [F](K06/focus-f4-3-patch-currentness-and-review.md) |
| `K06.F5.1` | 临时缓解与永久修复 | [S](K06/systematic-syllabus.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F5.2` | 事故时序与影响 | [S](K06/systematic-syllabus.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K06.F5.3` | 可复用经验候选 | [S](K06/systematic-syllabus.md) · [D2](K06/foundations-deep-dive-02.md) |
| `K07.F1.1` | 协作目标与成果物 | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F1.2` | 协调成本与瓶颈 | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F1.3` | 人机责任边界 | [S](K07/systematic-syllabus.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F2.1` | Planner/Builder/Reviewer/Validator | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) |
| `K07.F2.2` | Session/Attempt/Source 绑定 | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) |
| `K07.F2.3` | 受控执行和会话边界 | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) |
| `K07.F3.1` | 消息 vs canonical work facts | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) |
| `K07.F3.2` | 超时/重复/迟到处理 | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F3.3` | 上下文选择与精简 | [S](K07/systematic-syllabus.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F4.1` | 独立写集与 Shared Owner | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) |
| `K07.F4.2` | 稳定接口与受控合并 | [S](K07/systematic-syllabus.md) · [D1](K07/foundations-deep-dive-01.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F4.3` | 冲突/失败恢复 | [S](K07/systematic-syllabus.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F5.1` | 可证伪独立代码审查 | [S](K07/systematic-syllabus.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F5.2` | 评估与协作反馈 | [S](K07/systematic-syllabus.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K07.F5.3` | 失误记录与可复用规则候选 | [S](K07/systematic-syllabus.md) · [D2](K07/foundations-deep-dive-02.md) |
| `K08.F1.1` | 源码/配置/制品多重身份 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F1.2` | 变更控制与可复现构建 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F1.3` | 资源/生成代码维护 | [S](K08/systematic-syllabus.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F2.1` | 编译链和依赖图 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F2.2` | 依赖安装 vs Build | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F2.3` | 增量构建与缓存正确性 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F3.1` | Provider 语义和工作区 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) |
| `K08.F3.2` | 最小且真实的测试顺序 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) |
| `K08.F3.3` | 执行阶段与证据分离 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) |
| `K08.F4.1` | 组件/公共入口集成 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F4.2` | Artifact 与平台目标 | [S](K08/systematic-syllabus.md) · [D1](K08/foundations-deep-dive-01.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F4.3` | 首次运行/配置与权限 | [S](K08/systematic-syllabus.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F5.1` | 候选合并与 Release 权威 | [S](K08/systematic-syllabus.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F5.2` | 部署变更与回滚 | [S](K08/systematic-syllabus.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K08.F5.3` | 交付与故障复盘 | [S](K08/systematic-syllabus.md) · [D2](K08/foundations-deep-dive-02.md) |
| `K09.F1.1` | 修复/增强/重构/适配 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F1.2` | 维护优先级与长期成本 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F1.3` | 系统生命期和退役 | [S](K09/systematic-syllabus.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F2.1` | 行为保留的重构 Oracle | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) |
| `K09.F2.2` | 逐步替换与 Strangler 模式 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F2.3` | 可测试性与依赖切割 | [S](K09/systematic-syllabus.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F3.1` | API 兼容版本 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) |
| `K09.F3.2` | 演进式数据库设计 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) |
| `K09.F3.3` | 配置和数据语义迁移 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) |
| `K09.F4.1` | 受支持消费者及契约 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F4.2` | 渐进部署与回滚 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F4.3` | 回归传播和证据复用 | [S](K09/systematic-syllabus.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F5.1` | 技术债成因与利息 | [S](K09/systematic-syllabus.md) · [D1](K09/foundations-deep-dive-01.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F5.2` | 替换与保留的经济评估 | [S](K09/systematic-syllabus.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K09.F5.3` | 受证据驱动的停止条件 | [S](K09/systematic-syllabus.md) · [D2](K09/foundations-deep-dive-02.md) |
| `K10.F1.1` | 质量特征与业务场景 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F1.2` | 故障/缺陷/失效区分 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) |
| `K10.F1.3` | SLO/SLI 与错误预算 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F2.1` | 结构化诊断与关联 ID | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) |
| `K10.F2.2` | Logs/Metrics/Traces | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F2.3` | 监控与告警噪声 | [S](K10/systematic-syllabus.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F3.1` | Timeout/重试/退避 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) |
| `K10.F3.2` | 幂等与恢复一致性 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) |
| `K10.F3.3` | 隔离/熔断/背压 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F4.1` | 用户指标与分层定位 | [S](K10/systematic-syllabus.md) · [D1](K10/foundations-deep-dive-01.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F4.2` | 火焰图/Profiling 和瓶颈 | [S](K10/systematic-syllabus.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F4.3` | 实验对照和误差 | [S](K10/systematic-syllabus.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F5.1` | 首次运行和配置形态 | [S](K10/systematic-syllabus.md) · [F](K10/focus-f5-1-first-run-post-start-configuration.md) |
| `K10.F5.2` | 事故响应与恢复验证 | [S](K10/systematic-syllabus.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K10.F5.3` | 运行质量经验回流 | [S](K10/systematic-syllabus.md) · [D2](K10/foundations-deep-dive-02.md) |
| `K11.F1.1` | 目标、用户价值与效用 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) |
| `K11.F1.2` | 总成本与生命周期 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |
| `K11.F1.3` | 机会成本与可逆性 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) |
| `K11.F2.1` | 估算不是精确承诺 | [S](K11/systematic-syllabus.md) · [D2](K11/foundations-deep-dive-02.md) |
| `K11.F2.2` | 复杂度与依赖成本 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) |
| `K11.F2.3` | 研究价值与停止规则 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |
| `K11.F3.1` | 风险识别与后果强度 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) |
| `K11.F3.2` | 质量保证下限 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |
| `K11.F3.3` | 风险缓解选项 | [S](K11/systematic-syllabus.md) · [D2](K11/foundations-deep-dive-02.md) |
| `K11.F4.1` | 信息需求驱动的度量 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |
| `K11.F4.2` | SPACE / DORA 的解释边界 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |
| `K11.F4.3` | 实验和对照设计 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |
| `K11.F5.1` | 最小必要流程 | [S](K11/systematic-syllabus.md) · [D2](K11/foundations-deep-dive-02.md) |
| `K11.F5.2` | 多 Agent 协作经济学 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) |
| `K11.F5.3` | 知识维护/经验化投资 | [S](K11/systematic-syllabus.md) · [D1](K11/foundations-deep-dive-01.md) · [D2](K11/foundations-deep-dive-02.md) · [A](K11/advanced-empirical-evaluation-and-value-of-knowledge.md) |

## R4 六项补缺与仍需深度建设的区别

R3 独立审核确认六项仅有 S；R4 已新增六篇专业聚焦正文，并在本台账以 F 逐项链接（**本轮新增，仍待独立审核**）：

| 原课程点 | R4 专业补缺中心 | 本轮资格 |
| --- | --- | --- |
| [K02.F4.3](K02/focus-f4-3-minimal-failure-and-trust-boundary.md) | 信任/失败边界、租户隔离反例 | Document Candidate |
| [K04.F2.3](K04/focus-f2-3-task-acceptance-and-authority.md) | Task/证据/有权接受的区分 | Document Candidate |
| [K05.F3.3](K05/focus-f3-3-build-package-platform-verification.md) | Build→Package→Install→Launch Oracle | Document Candidate |
| [K05.F4.2](K05/focus-f4-2-golden-snapshot-evidence-freshness.md) | Golden 权威、过期与正反例 | Document Candidate |
| [K06.F4.3](K06/focus-f4-3-patch-currentness-and-review.md) | Patch HEAD、诊断、受影响复核 | Document Candidate |
| [K10.F5.1](K10/focus-f5-1-first-run-post-start-configuration.md) | 首次运行、运行后配置、Secret 隔离 | Document Candidate |

新增 F 是**六个原基础缺口的正文补齐**，不是把全部 165 点各自写成了独立专著。下一阶段重点按真实学科结构和教学反例的不足继续完善其他 D1/D2 主题（以及逐项来源资格）；不为凑数字复制 165 篇重复文档。

## R5：25 个已存课程点获得更详细的推理原理链接（候选）

这不是再造新课程 ID，也不是将现有 159 个 D1/D2 一概升级。R5 新增四篇深层技术研究文章，分别映射 K01 6 点、K03 5 点、K05 7 点、K11 7 点，共 **25/165** 个课程概念的额外 A 导航；它们在上一轮已经有 S 与 D1/D2 主题关联，现在获得更可复查的演算、反例与失败序列。其余 140 点仍维持原有深度现状，不应自动标为高级教材已经完成。

另有 [五条跨模块完整推理演算](professional-reasoning-worked-traces.md)，用于检验从事实、授权范围、知识选择、机制、Oracle、反例直到 UNKNOWN 的推理链，**不是五次真实 Agent 模型实验**。原 Q01–Q18 和 48 条概念关系未因此增加或修改。

## 学科完成度只能按正确维度报告

- **S：165/165** 个唯一 ID 的课程概要存在。**D1/D2 文档主题触及：159/165；其余 6/165 有 R4 一点一篇 F 深文；S+D1/D2/F 的导航关联覆盖 165/165。** 这只是有正文引用的覆盖率，不是完整学科掌握或 Source 资格认证。
- **11/11 模块**均有两卷原创深化正文；每卷是该模块不同学科侧面的探讨而非两个重复模板。
- **外部来源目录：61 条发现/适用参考**，不是 61 条合格 Claim；现有 26 Topic 为问题级投影而非全部学科范围。
- **当前合格 Source/DomainHarness 消费**仍无本轮新增实证；任何 Agent/Host/CI、Product/L2/Release 资格需另行真实验证。

**R5 额外进阶 A 引用：25/165**，并非 25 条已合格 Source Claim。**各模块的主题关联或 R4 聚焦正文覆盖**：K01–K11 均 15/15（其中 K02 1 点、K04 1 点、K05 2 点、K06 1 点、K10 1 点新增 F）；保留 R3 原始 159 点 D1/D2 限定事实，勿将新增 F 计为原两卷本身已涉及。

## 维护规则

新增或改写一个知识点时，首先明确其 Kxx Owner 与原 syllabus ID，再选择是更新原定义、补 D1/D2 的适用/反例，还是新建必要的主题章节；不得把跨模块 Concept 引用、Source URL 或合成 Case 当作运行观察。每个章节后续变更应只复核受影响的 ID/链接/正反例和来源，不重复审查已保持 Exact-SHA 相同的其他专业知识。
