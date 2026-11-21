# K11 · 工程经济学与决策：外部学科知识基础

> [K11 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；本文件为独立原创工程知识与实际方法的总结，非受版权保护的国际标准/教材原文复刻，不提升现有 Task/Release 权威。

## 学科机制、决策产物与错误泛化

每项知识都对当前真实软件工程任务的某个决策有用，包含受支持来源、适用范围与可证伪反例。来自外部标准/研究/企业实践的**知识方法不是现行 ADS 的新 MUST**；只有实际生效的 ADS 版本和合法项目规则承担强约束。

### F-K11-01 · 工程经济学不仅看 Token 或单次 CI 成本

**学科观点与适用机制：** SWEBOK Software Engineering Economics 将成本、风险、不确定性和生命周期选择视为核心知识。真正的工程选择需要比较当前编码、研究、Review、CI/Host、集成、维护和错误修复的总影响，并与用户交付价值相连。

**Agent 应产出的结果：** 对同一目标的最小两个可行方案列观察成本、估计成本、业务/质量后果和不可逆风险。

**必须排除的假结论：** 仅凭最小 Agent token 或最多完成 Issue 判断生产率。

**外部出处：** [A01](../external-source-catalog.md)、[A10](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K11-02 · 真实度量从信息需求开始

**学科观点与适用机制：** ISO/IEC/IEEE 15939:2017 讨论如何按信息需求设计测量过程；SPACE 论文强调开发者生产率不能由一个指标决定，包含满意度/福祉、绩效、活动、协作和效率/流。Tally 用这些概念确定真实应观察什么，不能为每项任务添加复杂绩效系统。

**Agent 应产出的结果：** 选择当前任务所需的两到三个有真实来源/可验证的 outcome/cost/quality 指标，说明抽样偏差。

**必须排除的假结论：** 把 PR 频率、代码行或 Token 总量当作唯一交付质量。

**外部出处：** [A10](../external-source-catalog.md)、[A40](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K11-03 · DORA 是实证关系，不是通用因果公式

**学科观点与适用机制：** DORA 持续交付研究关注交付效率、稳定性、组织条件与技术能力。2025 AI 报告强调 AI 的效用受团队条件和工程系统影响；研究发现的相关性不能推定 Tally 加入某个知识文件必然使每个开发任务效率翻倍。

**Agent 应产出的结果：** 用真实对照任务和可比较环境计算本地观察，保留负收益/小样本与无法外推情况。

**必须排除的假结论：** 把 DORA 的跨组织调查直接转换为当前 PR 的量化提效承诺。

**外部出处：** [A33](../external-source-catalog.md)、[A34](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K11-04 · 方法适应性与风险保证下限

**学科观点与适用机制：** ISO/IEC/IEEE 12207:2026 不要求固定开发模型；CMMI/Essence/企业 Playbook 可以提供可选流程改进语言，但不能变成强制额外 Gate。对小而清楚的任务，INLINE 与受影响测试通常经济；对不可逆/高后果任务，即使改动三行也不能放弃真实适用保证。

**Agent 应产出的结果：** 将复杂度、风险、可逆性、权限和新证据潜在收益并列，明确何时停止研究/重复 CI。

**必须排除的假结论：** 为了省一次 CI 就把必须在真实 Host 跑的迁移验收删掉。

**外部出处：** [A04](../external-source-catalog.md)、[A37](../external-source-catalog.md)、[A39](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K11-05 · 多 Agent 调度成本与工作图选择

**学科观点与适用机制：** GitLab/企业实践指出碎片化/水平任务可引入额外 handoff 和合并风险。并行收益需超过重复上下文、等待、共享写冲突、Reviewer/集成和失败修复成本；有剩余订阅 Token 不是自动并行的理由。

**Agent 应产出的结果：** 比较单 Writer/多只读 Agent/多个 disjoint module Builder 的实际交付耗时与质量/返工，优先完整闭环。

**必须排除的假结论：** 把 8 个 Writer 分别安排在同一个锁文件/共享生成物并声称速度必然提升。

**外部出处：** [A20](../external-source-catalog.md)、[A23](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

### F-K11-06 · Agent 效用评估要与任务正确性耦合

**学科观点与适用机制：** SWE-bench 类型的 Issue→Patch→Test 评价只能覆盖特定修复目标。Tally 的 Domain Harness 目标还要求 L1 需求/L2 设计、合法 L3 实现、真实用户路径和独立复核。实验应冻结样本、权限、预算、知识版本和评审条件，报告实现质量、测试有效性和成本。

**Agent 应产出的结果：** 实际设计 baseline Agent 与 selected-knowledge Agent 两种相同任务条件的对照，保留失败/未执行、平均与个体差异。

**必须排除的假结论：** 用 7 个模型问答的分数或自评“更好”宣称可高质量自主完成完整软件研发。

**外部出处：** [A27](../external-source-catalog.md)、[A33](../external-source-catalog.md)、[A34](../external-source-catalog.md)、[A40](../external-source-catalog.md)；请以来源总目录的版本和出版方链接核查全文/许可证。

## 从学科知识到真实任务的选择方式

1. 确认当前 Product/Task/版本/角色/运行环境，并判断是否真的需要当前模块的学科知识。没有实际问题时不批量加载本模块或组织额外治理。
2. 在上述 F-K11 项中选择能改变该决策的一项，核对外部引用的当前版本与适用边界；知识来自教材/企业研究不意味着本项目必须照搬。
3. 调用 [engineering-methods.md](engineering-methods.md) 的相应 DP，生成明确输入、输出、真实反例和停止条件。
4. 仅对实际受权 Task 执行源码/测试/Host；不能取得的证据记 `UNKNOWN/NOT_RUN`，并保留本知识库尚未 Source/Package Qualified 的现状。

**暂缓领域**：代码库逆向分析和独立软件安全/供应链知识专项仍不建设；原本适用的基础源码读取、权限和保密约束有效。