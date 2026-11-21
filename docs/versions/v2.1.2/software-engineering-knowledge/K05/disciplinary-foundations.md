# K05 · 软件测试与质量验证：外部学科知识基础

> [K05 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 从学科依据到可供 Agent 使用的知识

本文件**不是来源链接堆积**。对每个主题分别描述：工程学科中的核心问题/机制、在 Tally 的适用决策、典型不当推断，以及可追溯的 Axx 原始来源。适用任务优先使用已经有权的 ADS v4.19.0 规范；外部知识只提出更准确的概念/可选择的方法，**不自动增加 MUST/Gate**。

### F-K05-01 · 验证与确认：谁的真实需求

**知识解释：** IEEE 1012 规范 Verification and Validation 过程。工程中需要区分产物是否遵从规定设计/合同（verification）与用户需求是否真实达成（validation）。测试用例数量或编译通过只能支持其覆盖范围，不能推导产品验证或正式 Release。

**Agent 在实际工程任务中应产出：** 为每个需求写可观察 Oracle、执行阶段和证据当前性；按适用验证层给出不足之处。

**必须避免的错误泛化：** 用“3 个测试都 PASS”宣称用户业务已经验收。

**参考来源：** [A09](../external-source-catalog.md)、[A01](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K05-02 · 测试设计技术分类

**知识解释：** ISTQB CTFL v4.0.1 涵盖等价类、边界值、决策表、状态转换、经验/探索以及风险驱动测试；ISO/IEC/IEEE 29119-1/2 提供测试概念/流程。Agent 必须选适合当前输入/状态的技术，而非固定测试模板。

**Agent 在实际工程任务中应产出：** 复杂业务规则用决策表，状态流程用转移表，数据范围用边界值；高风险失败副作用加反例。

**必须避免的错误泛化：** 为所有函数固定生成同样数量的 trivial happy-path 测试。

**参考来源：** [A08](../external-source-catalog.md)、[A14](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K05-03 · 质量属性驱动测试取舍

**知识解释：** ISO 25010:2023 的九个质量特征提供质量类别，不代表任何项目都应测试九大项。性能、可靠性、可用性、兼容等需从当前 NFR 选取有来源和可复现的场景；未确认的目标不能凭空生成通过线。

**Agent 在实际工程任务中应产出：** 结合 Quality Scenario→风险/测试层级→实际指标/负例；仅保留项目受影响的最小充分测试。

**必须避免的错误泛化：** 每个小文案修复都跑完整负载和容灾演练以求‘规范完善’。

**参考来源：** [A07](../external-source-catalog.md)、[A09](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K05-04 · Mock、契约测试与真实集成

**知识解释：** Pact 的 Consumer-driven contracts 和 Testcontainers 的实际容器依赖都提供减少错误的技术途径。契约测试证明接口约定在选择的参与方成立，Testcontainers 验证真实依赖行为，但两者仍不等于完整用户 E2E 或真实部署 Host。

**Agent 在实际工程任务中应产出：** 对外 API 合同用消费方协议负例，实际数据库副作用按必要场景使用真实依赖，最后核对真实选定入口。

**必须避免的错误泛化：** Mock PASS 便认定真实持久化/打包/安装全部合格。

**参考来源：** [A25](../external-source-catalog.md)、[A26](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K05-05 · 真实 Agent 修复任务的基准与限制

**知识解释：** SWE-bench 用真实 Issue、代码补丁与测试执行评估修复；SWE-bench PASS 主要证明特定样本的补丁满足某套测试，不证明完整 Product PRD 或独立业务 Oracle。Tally 对代码 Agent 应加上当前用户需求、保留行为、交付/Review 和实际成本评估。

**Agent 在实际工程任务中应产出：** 设计两个测量层：修复补丁测试有效性 + 全任务业务交付/回归/负例；保证基线和知识辅助公平。

**必须避免的错误泛化：** 把 benchmark resolved 数量当完整软件工程 Domain Harness PASS。

**参考来源：** [A27](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

## 从知识到 Tally Workflow 的最小决策

| 任务当前问题 | 合适的调用顺序 | 对结果的最低判断 |
| --- | --- | --- |
| 实际源/需求尚未明确 | 先确认当前 Product/Task/ADS 版本，再选最少的 F-K05 知识 | 保留 UNKNOWN，不根据教材里的示例假设作真实业务决定 |
| 已有正确合同，需要一个工程方法 | 从 [engineering-methods.md](engineering-methods.md) 选一条 DP，并结合上面适用条件 | 输出可被代码/测试/用户行为证伪的方法，而非一本教科书摘要 |
| 真实构建/测试/Host 未运行 | 在 [validation.md](validation.md) 区分设计候选与真实执行证据 | 明确 NOT_RUN/NOT_QUALIFIED，不拿来源权威冒充 Task PASS |

**专门保留的限制**：用户要求暂缓代码库逆向分析和软件安全/供应链知识专门建设；国际标准的覆盖面可能包含这些学科，但本模块只提取当前软件开发的必要内容。