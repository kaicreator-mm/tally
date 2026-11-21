# K02 · 架构与系统设计：外部学科知识基础

> [K02 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 从学科依据到可供 Agent 使用的知识

本文件**不是来源链接堆积**。对每个主题分别描述：工程学科中的核心问题/机制、在 Tally 的适用决策、典型不当推断，以及可追溯的 Axx 原始来源。适用任务优先使用已经有权的 ADS v4.19.0 规范；外部知识只提出更准确的概念/可选择的方法，**不自动增加 MUST/Gate**。

### F-K02-01 · ISO 42010 的描述与真正架构决定

**知识解释：** Architecture description 用于表达某实体的架构，至少需要能说明相关 stakeholder、concern、viewpoint/view 和一致性等概念。C4 是有用的沟通视图工具，但不是 ISO 42010 的唯一或强制实现；架构文档不能只是一张漂亮的容器图。

**Agent 在实际工程任务中应产出：** Agent 列出用户/维护者/运维关注点→支撑视图与决策；每张图应回答某个架构问题。

**必须避免的错误泛化：** 以 C4 画图完成认定数据所有权、失败语义与接口变更已获批准。

**参考来源：** [A06](../external-source-catalog.md)、[A19](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K02-02 · 质量属性权衡与 SEI ATAM

**知识解释：** CMU SEI ATAM 关注多个质量属性间的冲突，例如吞吐/延迟与可恢复性、模块隔离与集成成本。只有确实影响关键架构选择时才采用有界架构评估；简单改动可复用已有设计，不用开大型研讨会。

**Agent 在实际工程任务中应产出：** 形成质量情景、候选结构、风险/敏感点/Tradeoff 点、验证计划；明确假定与可否证条件。

**必须避免的错误泛化：** 用“高性能、可靠、可扩展”三个形容词证明某架构最优。

**参考来源：** [A07](../external-source-catalog.md)、[A15](../external-source-catalog.md)、[A16](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K02-03 · ADR 是可追溯决策，不是架构审批

**知识解释：** MADR 与 arc42 展示如何记录 Context/Decision/Consequences、约束、运行场景、技术观点等。它们是公开工程模板实践，实际内容需由任务驱动原创形成。已写 ADR 不代表 Product/L2 Authority 冻结或服务部署已经发生。

**Agent 在实际工程任务中应产出：** 用两个现实候选、决定理由、风险、接口/迁移义务、可逆性形成一页 ADR；当前源/Review 仍独立核对。

**必须避免的错误泛化：** 复制模板里所有章节，导致小接口修复产生无用形式要求。

**参考来源：** [A17](../external-source-catalog.md)、[A18](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K02-04 · 模块边界、状态所有权与跨进程事实

**知识解释：** 决定边界时需同时看业务语义、操作一致性、部署环境与协作组织。写入权、提交点、幂等策略和恢复来源构成真正的实现合同；目录结构或服务名称不能自动说明业务一致性。

**Agent 在实际工程任务中应产出：** 明确单一写入 Owner、事务/消息边界、重试和部分失败结果、必要真实集成检验。

**必须避免的错误泛化：** 两个 Agent 因工作树不同就对同一业务表独立实现不兼容写入。

**参考来源：** [A01](../external-source-catalog.md)、[A03](../external-source-catalog.md)、[A06](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K02-05 · 架构模型与代码/运维的双向反馈

**知识解释：** SWEBOK 把设计视为持续可修订的工程活动，SEBoK 补充复杂系统与跨生命期考虑。架构研究需具体说明哪个演示环境验证了什么、哪些质量属性没有测试；事实被证伪时有界修订 L2，不暗改需求。

**Agent 在实际工程任务中应产出：** 输出 Demo 的输入、运行条件、目标假设、失败反例和允许外推范围，连到受影响 Task seam。

**必须避免的错误泛化：** 局部 Demo 达标便宣布生产可用和正式 Release 合格。

**参考来源：** [A01](../external-source-catalog.md)、[A03](../external-source-catalog.md)、[A15](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

## 从知识到 Tally Workflow 的最小决策

| 任务当前问题 | 合适的调用顺序 | 对结果的最低判断 |
| --- | --- | --- |
| 实际源/需求尚未明确 | 先确认当前 Product/Task/ADS 版本，再选最少的 F-K02 知识 | 保留 UNKNOWN，不根据教材里的示例假设作真实业务决定 |
| 已有正确合同，需要一个工程方法 | 从 [engineering-methods.md](engineering-methods.md) 选一条 DP，并结合上面适用条件 | 输出可被代码/测试/用户行为证伪的方法，而非一本教科书摘要 |
| 真实构建/测试/Host 未运行 | 在 [validation.md](validation.md) 区分设计候选与真实执行证据 | 明确 NOT_RUN/NOT_QUALIFIED，不拿来源权威冒充 Task PASS |

**专门保留的限制**：用户要求暂缓代码库逆向分析和软件安全/供应链知识专门建设；国际标准的覆盖面可能包含这些学科，但本模块只提取当前软件开发的必要内容。