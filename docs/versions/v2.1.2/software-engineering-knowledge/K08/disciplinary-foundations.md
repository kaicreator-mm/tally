# K08 · 构建、集成与交付：外部学科知识基础

> [K08 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 学科机制、决策产物与错误泛化

每项知识都对当前真实软件工程任务的某个决策有用，包含受支持来源、适用范围与可证伪反例。来自外部标准/研究/企业实践的**知识方法不是现行 ADS 的新 MUST**；只有实际生效的 ADS 版本和合法项目规则承担强约束。

### F-K08-01 · 软件生命周期过程与具体 CI 工具分离

**学科观点与适用机制：** ISO/IEC/IEEE 12207:2026 定义生命周期过程，但不指定使用哪个开发方法；DORA 的持续交付实践强调小批量、版本控制和反馈。Tally 应在项目已用的 Woodpecker/Provider 工具链上选择最小有效步骤，不为学习新标准而引入新 CI 系统。

**Agent 应产出的结果：** 从真实 manifest/lock/CI 脚本推导当前任务的 Build/Typecheck/受影响测试/集成/打包目标，标注跳过与必须验证理由。

**必须排除的假结论：** 要求所有项目必须用 GitHub Actions，或每个 Workflow 都重新完整 clone。

**外部出处：** [A04](../external-source-catalog.md)、[A33](../external-source-catalog.md)、[A20](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K08-02 · Build/验证/制品/Release 是不同的证据对象

**学科观点与适用机制：** SWEBOK Software Engineering Operations/Configuration 与 Microsoft Playbook 指向多个不同结果：依赖安装、编译/类型、测试发现与行为、用户入口集成、打包、安装和正式 Release。CI 成功并不能自动证明该 Release Artifact 已创建并在目标平台运行；所有收据要绑定源 SHA/配置/Provider/环境。

**Agent 应产出的结果：** 产生分层执行结果表与准确首个失败步骤；对没有运行的真实 Host/安装记 NOT_RUN。

**必须排除的假结论：** 将 npm install SUCCESS 解释为 TypeScript build 或用户应用安装成功。

**外部出处：** [A01](../external-source-catalog.md)、[A20](../external-source-catalog.md)、[A23](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K08-03 · 单次获取源码与测试覆盖之间的成本权衡

**学科观点与适用机制：** DORA 的 Continuous Delivery 和 GitLab 的工程实践提倡可靠、可反馈的交付流水线。减少重复 clone 是潜在优化，但不同 CI Provider 的 Workflow 可能处于隔离容器，必须先核对真实工作区/缓存语义和依赖闭包；不靠 YAML 结构假设所有 clone 是同一网络操作。

**Agent 应产出的结果：** 先测实际 clone 次数/耗时与工作区共享可行性，再调整缓存/ checkout；证明改动后仍验证准确 SourceSHA。

**必须排除的假结论：** 做了 sparse checkout 却丢失测试 fixture/构建脚本，或虚报实际网络节省。

**外部出处：** [A23](../external-source-catalog.md)、[A33](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K08-04 · 持续交付需要用户关键路径和反馈而不是绿色徽章

**学科观点与适用机制：** DORA 把稳定交付与真实业务表现相联系；SWE-bench 的补丁测试验证只适用于它的 Issue→Patch 范围。Tally 真正交付要确认选定应用入口调用目标功能、适用测试层/独立审核、打包产物及首次启动配置。不同环境真实观察不可被 Model 答案代替。

**Agent 应产出的结果：** 将真正的 Critical Journey、Artifacts/Checks/Host/Version/Release 证据组织成可决策但不代签的候选报告。

**必须排除的假结论：** 仅有绿色单元测试和 Merge Commit 却报告安装包可下载。

**外部出处：** [A27](../external-source-catalog.md)、[A33](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K08-05 · 质量权威与标准适用性的最小化

**学科观点与适用机制：** CMMI/企业手册/软件教科书能提供改进流程的学科背景，但只应选对当前团队有价值的能力，不应当借“标准化”增设 11 层 Review。变更若仅为无影响文档，可按现行 ADS 减少重复 CI；真实必须独立 Provider/Host 或受影响公开合同的验证不能省略。

**Agent 应产出的结果：** 记录 Need→Checks 和 skip rationale，由原本已有 Task/Review/Release Owner 判断是否足够。

**必须排除的假结论：** 为了降低 CI 成本直接删除保护高风险接口的合同测试。

**外部出处：** [A20](../external-source-catalog.md)、[A33](../external-source-catalog.md)、[A37](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

## 从学科知识到真实任务的选择方式

1. 确认当前 Product/Task/版本/角色/运行环境，并判断是否真的需要当前模块的学科知识。没有实际问题时不批量加载本模块或组织额外治理。
2. 在上述 F-K08 项中选择能改变该决策的一项，核对外部引用的当前版本与适用边界；知识来自教材/企业研究不意味着本项目必须照搬。
3. 调用 [engineering-methods.md](engineering-methods.md) 的相应 DP，生成明确输入、输出、真实反例和停止条件。
4. 仅对实际受权 Task 执行源码/测试/Host；不能取得的证据记 `UNKNOWN/NOT_RUN`，并保留本知识库尚未 Source/Package Qualified 的现状。

**暂缓领域**：代码库逆向分析和独立软件安全/供应链知识专项仍不建设；原本适用的基础源码读取、权限和保密约束有效。