# K04 · 任务规划与分解：外部学科知识基础

> [K04 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 从学科依据到可供 Agent 使用的知识

本文件**不是来源链接堆积**。对每个主题分别描述：工程学科中的核心问题/机制、在 Tally 的适用决策、典型不当推断，以及可追溯的 Axx 原始来源。适用任务优先使用已经有权的 ADS v4.19.0 规范；外部知识只提出更准确的概念/可选择的方法，**不自动增加 MUST/Gate**。

### F-K04-01 · 生命周期过程不是固定模板

**知识解释：** ISO/IEC/IEEE 12207:2026 的官方范围说明明确不要求单一生命周期模型或开发方法。SWEBOK Process 也提供多种活动的适用方法。Tally 应根据实际交付、依赖和证据选择有界工作形式，而非把标准解读为所有项目都必须生成完整 L1/L2/L3 三张 DAG。

**Agent 在实际工程任务中应产出：** 给出当前 Work Outcome、阶段与最小必要任务/协调，说明为何某项工作可直接 INLINE。

**必须避免的错误泛化：** 把 2026 生命周期标准当成强制三-DAG 审批系统。

**参考来源：** [A01](../external-source-catalog.md)、[A04](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K04-02 · 任务原子性与工作切片

**知识解释：** GitLab 模块化代码原则提示横向按技术层拆分可能造成不必要 handoff。Task 的合理边界是可验证用户结果、受控写集和稳定集成 seam，而非“数据库一张 PR、逻辑一张、测试一张”。

**Agent 在实际工程任务中应产出：** 按 User-visible Outcome 做适当的纵向切片，在确有多个独立成果时才分 Task；代码和对应测试优先同一 Worker 验收。

**必须避免的错误泛化：** 把每个实现动作都变为 Task/Issue 导致反复合并和多 CI。

**参考来源：** [A23](../external-source-catalog.md)、[A24](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K04-03 · Scrum、CMMI、Essence 不能成为新权威

**知识解释：** Scrum Guide 2020 是可选团队反馈/增量框架；CMMI 是过程改进参考；OMG Essence 提供方法术语，2026 2.0 beta 仍处候选状态。它们不能替代 GitHub Native Task Dependencies、实际 Product/Review/Host 决策权威。

**Agent 在实际工程任务中应产出：** 仅在现有团队/项目已经应用相应方法或任务确实受益时复用其小范围知识。

**必须避免的错误泛化：** 为模拟某个方法的完整度新增 Sprint/Alpha/Gate 的第二套 Tally Task 事实源。

**参考来源：** [A37](../external-source-catalog.md)、[A38](../external-source-catalog.md)、[A39](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K04-04 · 真正依赖与非阻塞证据

**知识解释：** A BLOCKED_BY B 必须对应后项确实需要且目前不可替代的前置产物；CONSUMES_EVIDENCE 只是参考来源。研究任务、架构设计与正式代码实施任务可以互相关联，但不能因为标注同一图就让研究节点自动获得 Native Task 权限。

**Agent 在实际工程任务中应产出：** 给每条边写因果反事实：移除前任务能否合法做后任务？可以则删硬阻塞；不存在真实前置时允许并行。

**必须避免的错误泛化：** 把所有相关 Issue 都变成串行依赖，使版本长期无法收尾。

**参考来源：** [A01](../external-source-catalog.md)、[A04](../external-source-catalog.md)、[A24](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K04-05 · 并行收益和真实融合成本

**知识解释：** 外部工程手册强调小批量、真实可交付成果、及时反馈和减少闲置任务。Agent 数越多越不一定快：共享代码写冲突、重复环境初始化、合并和独立 Reviewer 成本可能抵消节省时间。

**Agent 在实际工程任务中应产出：** 比较单 Builder、并行只读与 disjoint Write Sets 方案的总耗时和真实风险，选择最小充分。

**必须避免的错误泛化：** 为了利用订阅 Token 强行八个 Builder 并行写一份锁文件。

**参考来源：** [A20](../external-source-catalog.md)、[A23](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

## 从知识到 Tally Workflow 的最小决策

| 任务当前问题 | 合适的调用顺序 | 对结果的最低判断 |
| --- | --- | --- |
| 实际源/需求尚未明确 | 先确认当前 Product/Task/ADS 版本，再选最少的 F-K04 知识 | 保留 UNKNOWN，不根据教材里的示例假设作真实业务决定 |
| 已有正确合同，需要一个工程方法 | 从 [engineering-methods.md](engineering-methods.md) 选一条 DP，并结合上面适用条件 | 输出可被代码/测试/用户行为证伪的方法，而非一本教科书摘要 |
| 真实构建/测试/Host 未运行 | 在 [validation.md](validation.md) 区分设计候选与真实执行证据 | 明确 NOT_RUN/NOT_QUALIFIED，不拿来源权威冒充 Task PASS |

**专门保留的限制**：用户要求暂缓代码库逆向分析和软件安全/供应链知识专门建设；国际标准的覆盖面可能包含这些学科，但本模块只提取当前软件开发的必要内容。