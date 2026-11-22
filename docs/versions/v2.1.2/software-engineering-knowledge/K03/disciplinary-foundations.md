# K03 · 详细设计与代码实现：外部学科知识基础

> [K03 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 从学科依据到可供 Agent 使用的知识

本文件**不是来源链接堆积**。对每个主题分别描述：工程学科中的核心问题/机制、在 Tally 的适用决策、典型不当推断，以及可追溯的 Axx 原始来源。适用任务优先使用已经有权的 ADS v4.19.0 规范；外部知识只提出更准确的概念/可选择的方法，**不自动增加 MUST/Gate**。

### F-K03-01 · 构造与设计的差别

**知识解释：** SWEBOK 将 Architecture/Design 和 Construction 区分为不同知识领域；具体实现仍要遵从输入输出、状态/错误、资源生命周期、运行环境与可维护性约束。编译器没有报告语法错误并不意味着行为正确。

**Agent 在实际工程任务中应产出：** 输入当前 Task、L2 契约与实际 Write Set，输出有完整正反路径的实现方案和最小受影响验证。

**必须避免的错误泛化：** 只对成功分支加功能，遗漏重复调用、错误、取消与资源回收。

**参考来源：** [A01](../external-source-catalog.md)、[A20](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K03-02 · 语言特定习惯与项目实际工具链

**知识解释：** Google Style Guides 展示语言特定命名/结构/注释手法；不同语言对异常、内存、并发、依赖管理、模块编译的最佳实践不一样。可维护的通用原则不能机械化为所有项目必须使用同一个 Linter 或 Node 版本。

**Agent 在实际工程任务中应产出：** 先识别真实 package/lock/目标 runtime，再选语言适用的格式/类型/构建和影响测试；避免新增无必要工具。

**必须避免的错误泛化：** 仅因 Agent 主机装 Node 24，就要求 Node 22 项目提高依赖底线。

**参考来源：** [A22](../external-source-catalog.md)、[A20](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K03-03 · 长时维护与抽象的总成本

**知识解释：** Software Engineering at Google 可作为软件长期演化的研究参考，强调代码在时间和协作者之间持续改变；当前源受 CC BY-NC-ND 许可，不能搬运内容。代码模块职责、命名/注释、测试边界、复用抽象应对真实需求有益而非追逐模式。

**Agent 在实际工程任务中应产出：** 根据当前变更复杂度选择提炼函数或保留直接实现，说明测试/演进影响及公开接口变动。

**必须避免的错误泛化：** 仅为消除少量重复创建通用 Provider Framework 或全仓重构。

**参考来源：** [A41](../external-source-catalog.md)、[A21](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K03-04 · 异步副作用、幂等与一致性

**知识解释：** AWS Builders' Library 对幂等操作的研究表明：请求超时/重复并不自然保证业务效果‘正好一次’，需要明确幂等标识、状态读取、提交点和响应语义。需要在真实操作的业务合同下设计，不是对每个纯函数添加全局分布式锁。

**Agent 在实际工程任务中应产出：** 对重复、超时、取消、部分成功确定预期效果与 UNKNOWN 状态，并建立失败注入或对应合同测试。

**必须避免的错误泛化：** 遇到所有异常都无限重试，导致重复下单、重复删除等副作用。

**参考来源：** [A32](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K03-05 · 实际代码审核与结果验证

**知识解释：** Google Engineering Practices（归档历史）和 Microsoft Playbook 提供变更可审查性、测试、Review 反馈与小批量交付的参考。明确区别 Builder 实现、测试真实执行、独立审查与正式交付。

**Agent 在实际工程任务中应产出：** 提交完整源/测试候选、真实命令/退出码/测试发现及 Exact HEAD，非法写集或没有 Host 时保留不足。

**必须避免的错误泛化：** PR 合并成功便向用户声称已实现正式安装/Release。

**参考来源：** [A20](../external-source-catalog.md)、[A21](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

## 从知识到 Tally Workflow 的最小决策

| 任务当前问题 | 合适的调用顺序 | 对结果的最低判断 |
| --- | --- | --- |
| 实际源/需求尚未明确 | 先确认当前 Product/Task/ADS 版本，再选最少的 F-K03 知识 | 保留 UNKNOWN，不根据教材里的示例假设作真实业务决定 |
| 已有正确合同，需要一个工程方法 | 从 [engineering-methods.md](engineering-methods.md) 选一条 DP，并结合上面适用条件 | 输出可被代码/测试/用户行为证伪的方法，而非一本教科书摘要 |
| 真实构建/测试/Host 未运行 | 在 [validation.md](validation.md) 区分设计候选与真实执行证据 | 明确 NOT_RUN/NOT_QUALIFIED，不拿来源权威冒充 Task PASS |

**专门保留的限制**：用户要求暂缓代码库逆向分析和软件安全/供应链知识专门建设；国际标准的覆盖面可能包含这些学科，但本模块只提取当前软件开发的必要内容。