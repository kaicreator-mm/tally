# K07 · 多 Agent 工程协作：外部学科知识基础

> [K07 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 学科机制、决策产物与错误泛化

每项知识都对当前真实软件工程任务的某个决策有用，包含受支持来源、适用范围与可证伪反例。来自外部标准/研究/企业实践的**知识方法不是现行 ADS 的新 MUST**；只有实际生效的 ADS 版本和合法项目规则承担强约束。

### F-K07-01 · 协作是软件工程专业实践，不是消息 API 的集合

**学科观点与适用机制：** SWEBOK 的 Professional Practice/Management 领域与 GitLab/Goggle/Microsoft 的审核实践重视反馈、职责、版本化交接及可检查的工作成果。Agent 组织可借鉴这些实践，但不能将某个企业的团队审批方式直接变成当前项目的 Task/Release 强制流程。

**Agent 应产出的结果：** 先识别 Planner/Builder/Reviewer 的合法职责、实际成果与接口；用真实代码、测试和独立会话证据证明交付。

**必须排除的假结论：** 以聊天消息“已完成”直接更新 Task PASS，或用账号相同/不同代替审核来源。

**外部出处：** [A01](../external-source-catalog.md)、[A20](../external-source-catalog.md)、[A21](../external-source-catalog.md)、[A23](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K07-02 · Agent Executor、通信端口与事实权威分离

**学科观点与适用机制：** OpenHands Software Agent SDK 展示事件、工具、工作区和 Agent Server 等实际设计选择，支持把工具执行和消息/状态事件解耦。但这是该项目的实现参考，不是 Tally 已存在的 API。Git/GitHub 提供版本与耐久成果事实，同机 IPC/HTTP/队列只是交互通道，必须允许拒绝、重试和对账。

**Agent 应产出的结果：** 为一次 Agent 交互标出 Task/Attempt/Role/SourceSHA/ArtifactRef、观察结果和正式任务读回，明确消息 ACK 与执行成功的区别。

**必须排除的假结论：** 把非 GitHub 传输禁用、把 ACK 当授权或自动创建第二套 Tally Agent 执行状态系统。

**外部出处：** [A23](../external-source-catalog.md)、[A29](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K07-03 · 多 Agent 并行取决于工作图和集成成本

**学科观点与适用机制：** GitLab 模块化代码原则强调把相关工作放在能清晰交付的边界里，避免按横向技术层造成不必要交接。多 Agent 可以并行只读探索、测试分析或互不冲突的模块实施；共享 Schema/lockfile/生成物必须有真实唯一 Writer 或明确串行方案。

**Agent 应产出的结果：** 为每个 Agent 给一个独立可验收 Outcome、允许写集和稳定接口，估算交接/合并/Review 成本，证明并行有实际收益。

**必须排除的假结论：** 仅因模型订阅剩余 Token 多就强制十个 Builder 同时编辑中心依赖。

**外部出处：** [A24](../external-source-catalog.md)、[A20](../external-source-catalog.md)、[A34](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K07-04 · 独立审查是来源行为而不是 Reviewer 字符串

**学科观点与适用机制：** Google 归档的工程实践和 GitLab Code Review 描述了实际查看变更、讨论与达成可审查结论的流程。Tally 的独立审查应绑定 Reviewer 的实际逻辑会话、是否参与写入、被审源码 SHA 和审核覆盖内容；同账号不同对话可具备独立性，但不能凭声明代替实际读取差异。

**Agent 应产出的结果：** 审查产物含 exact Base/HEAD、关联需求和实际测试的证据、问题级别与处理结论；修复后新 SHA 增量复核。

**必须排除的假结论：** 将 Builder 自报测试 PASS 当成不同角色审核，或复用旧 HEAD 的 ACCEPT 到新候选。

**外部出处：** [A21](../external-source-catalog.md)、[A23](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K07-05 · Agent 基准成绩与生产协作正确性是两个目标

**学科观点与适用机制：** SWE-agent、OpenHands SDK/Benchmarks 让我们研究 coding Agent 的工具路径、工作区/事件或真实 Issue 修复评价；其成功率和集成能力只能说明相应任务/环境。Tally 还必须评估完整任务交接、冲突/迟到消息、实际用户验收与 Owner Authority。DORA 2025 关于 AI 的实证观察也不能变成任何单次多 Agent 部署必然成功的保证。

**Agent 应产出的结果：** 用同任务基线比较单 Agent 与并行方案的真实交付时间、冲突/返工和负面结果；通信成功不替代 Task 合法完结。

**必须排除的假结论：** 把 benchmark 分数和 GitHub Stars 直接当作企业级 Agent 协作可靠性证明。

**外部出处：** [A28](../external-source-catalog.md)、[A29](../external-source-catalog.md)、[A30](../external-source-catalog.md)、[A34](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

## 从学科知识到真实任务的选择方式

1. 确认当前 Product/Task/版本/角色/运行环境，并判断是否真的需要当前模块的学科知识。没有实际问题时不批量加载本模块或组织额外治理。
2. 在上述 F-K07 项中选择能改变该决策的一项，核对外部引用的当前版本与适用边界；知识来自教材/企业研究不意味着本项目必须照搬。
3. 调用 [engineering-methods.md](engineering-methods.md) 的相应 DP，生成明确输入、输出、真实反例和停止条件。
4. 仅对实际受权 Task 执行源码/测试/Host；不能取得的证据记 `UNKNOWN/NOT_RUN`，并保留本知识库尚未 Source/Package Qualified 的现状。

**暂缓领域**：代码库逆向分析和独立软件安全/供应链知识专项仍不建设；原本适用的基础源码读取、权限和保密约束有效。