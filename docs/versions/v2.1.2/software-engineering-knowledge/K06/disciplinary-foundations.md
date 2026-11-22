# K06 · 调试与根因修复：外部学科知识基础

> [K06 知识库](README.md) · [v2.1.2 #521](https://github.com/kaicreator-mm/tally/issues/521) · [外部来源总目录](../external-source-catalog.md)。**EXTERNAL_DISCIPLINARY_SOURCE_CANDIDATE / NOT_SOURCE_QUALIFIED / NOT_CONNECTED**；独立原创方法总结，非 ISO/IEEE/书籍原文转录，不授予 Tally Package/Host/Task 资格。

## 从学科依据到可供 Agent 使用的知识

本文件**不是来源链接堆积**。对每个主题分别描述：工程学科中的核心问题/机制、在 Tally 的适用决策、典型不当推断，以及可追溯的 Axx 原始来源。适用任务优先使用已经有权的 ADS v4.19.0 规范；外部知识只提出更准确的概念/可选择的方法，**不自动增加 MUST/Gate**。

### F-K06-01 · 调试是因果分析而非异常文本匹配

**知识解释：** SWEBOK Construction/Testing 与 ISTQB Defect Management 区分故障表现、可能缺陷与可验证修复。先读真实首个失败阶段、源/测试版本与运行环境，再提出能被证伪的根因解释，避免将历史某次安装失败套用到当前 TypeScript 编译错误。

**Agent 在实际工程任务中应产出：** 输出 Observed Failure→Competing Causes→Minimal Probe→Fix→Regression；区别 SOURCE_BACKED 与 UNKNOWN。

**必须避免的错误泛化：** 看到 pipeline exit 2 就无条件重建 lockfile 或增加 retry。

**参考来源：** [A01](../external-source-catalog.md)、[A14](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K06-02 · 事故记录和无责复盘

**知识解释：** Google SRE postmortem 实践关注时间线、影响、缓解、根因分析和系统性改进；不能把事故总结当作个人错误归责，也不能通过恢复后涂改历史收据抹掉曾发生的故障。

**Agent 在实际工程任务中应产出：** 记录 earliest failure、Timeline、Impact、Mitigation、Recovery、Verification、Follow-up，来源与当前性可核对。

**必须避免的错误泛化：** 服务恢复后把过去失败日志改成 SUCCESS，或把未运行的回归测试写成 PASS。

**参考来源：** [A31](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K06-03 · 重试与幂等的边界

**知识解释：** AWS Builders' Library 的幂等 API 案例说明：网络超时后副作用可能已提交。诊断应优先区分永久/临时故障、幂等键与业务状态回读，再决定是否重试。盲目重试确定性代码缺陷只会放大 CI 消耗和副作用。

**Agent 在实际工程任务中应产出：** 给出必要状态查询、是否可安全重复、次数/截止与 UNKNOWN 分类；实际工程中只使用已授权的恢复/Host 工具。

**必须避免的错误泛化：** 所有 5xx/timeout 统一无限重试，或 HTTP 超时即宣布业务操作失败。

**参考来源：** [A32](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K06-04 · 基准可复现补丁与真实回归

**知识解释：** SWE-bench/SWE-agent 提供问题描述→源码补丁→测试复现的可研究链路，但其中的代理实现方式或基准结果并非所有项目的最佳修复方法。Tally 应强调‘一轮有界定位和修复’、尽量复用旧已接受证据、真实编译/影响测试与独立 Review。

**Agent 在实际工程任务中应产出：** 限定源码基线/补丁/实际 tests/环境，输出新旧行为差异和未覆盖内容；超过写集向 Owner 升级。

**必须避免的错误泛化：** 将某 benchmark 测试绿灯冒充真实 UI/Host/发布路径证明。

**参考来源：** [A27](../external-source-catalog.md)、[A28](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

### F-K06-05 · 失败类型升级到正确责任层

**知识解释：** 真正的原因可能属于产品误解、架构一致性、实现/类型、测试 Oracle 或 Provider/Runner。不同层级的 Owner 负责不同变更；K06 的诊断建议不允许私自修改冻结 Product、通用 K10 运行可靠性模式或其他 Writer 文件。

**Agent 在实际工程任务中应产出：** 输出 root cause 所属层级、最小修复与受影响验证，必要时交 K01/K02/K03/K05/K08 原 Owner。

**必须避免的错误泛化：** 源码编译错误修复任务变成全新架构重构项目。

**参考来源：** [A01](../external-source-catalog.md)、[A31](../external-source-catalog.md)；出版方原文链接和版本见总目录，知识合格性需另行确认。

## 从知识到 Tally Workflow 的最小决策

| 任务当前问题 | 合适的调用顺序 | 对结果的最低判断 |
| --- | --- | --- |
| 实际源/需求尚未明确 | 先确认当前 Product/Task/ADS 版本，再选最少的 F-K06 知识 | 保留 UNKNOWN，不根据教材里的示例假设作真实业务决定 |
| 已有正确合同，需要一个工程方法 | 从 [engineering-methods.md](engineering-methods.md) 选一条 DP，并结合上面适用条件 | 输出可被代码/测试/用户行为证伪的方法，而非一本教科书摘要 |
| 真实构建/测试/Host 未运行 | 在 [validation.md](validation.md) 区分设计候选与真实执行证据 | 明确 NOT_RUN/NOT_QUALIFIED，不拿来源权威冒充 Task PASS |

**专门保留的限制**：用户要求暂缓代码库逆向分析和软件安全/供应链知识专门建设；国际标准的覆盖面可能包含这些学科，但本模块只提取当前软件开发的必要内容。