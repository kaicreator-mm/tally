# K10 进阶可运行性：SLI/SLO、错误预算、可行动告警与 UNKNOWN Effect

> [K10 课程](systematic-syllabus.md) · [可靠性第一卷](foundations-deep-dive-01.md) · [Little 定律/尾延迟第二卷](foundations-deep-dive-02.md) · [首次启动配置 Focus](focus-f5-1-first-run-post-start-configuration.md)
>
> **R6 OPERABILITY_THEORY_CANDIDATE / NO_PRODUCTION_TELEMETRY_READ / NO_ACTIVE_MONITOR_DEPLOYED。** 映射 K10.F1.1、F1.3、F2.1、F2.2、F2.3、F3.1、F3.2、F4.1、F4.3。监控阈值必须有项目需求/用户体验证据；专业知识并不意味着为所有桌面或 CLI 项目新增 SRE 基础设施。

## 一、先从用户真实体验定义 SLI

将系统 “服务可用率” 写成一个没有分母的 99.9% 是不够的。教学 SaaS API 的一个候选 SLI：
`GoodRequests/EligibleRequests`，其中 Good 的定义可结合成功状态、目标延迟和用户可观察业务正确性；Eligible 必须明确排除或纳入客户端错误、重试、内部探针、计划维护与批处理调用。不同服务请求不同的“Good”，不能机械将所有 HTTP 2xx 视为业务成功；例如返回 200 但支付重复扣款不满足真正的业务正确性。

**计算例子（合成）**：30 天期间有 100,000 次符合明确 SLI 范围的请求，SLO 目标 99.9%，最多允许 100 次未达标；如果 120 次未达标，则按这一指标窗口超出 20 次。这个比例不证明软件有 bug，更不自动授予事故优先级；真实告警需考虑用户、严重性、错误预算策略与测量覆盖。SLO 是有权产品/团队采纳的目标，不是本章自动签发的生产 SLA。

## 二、单窗口静态阈值 vs 多窗口告警

固定阈值 “5 分钟 HTTP 5xx 超过 2% 就 Pager” 容易在低流量时被几个样本触发，也可能错过高流量下逐渐消耗预算的风险。可考虑按预先批准的 SLI/SLO 计算**错误预算消耗速率（burn rate）**，结合短窗和长窗，降低瞬时噪声同时捕捉持续高损失。例如 [Google SRE Workbook SLO 告警来源 A63](../external-source-catalog.md) 讨论了多窗口/多燃烧率方法。

但不能直接把公共书籍中的某一阈值数字硬塞入 Tally：服务请求类型、窗口、页级响应责任、Pager 预算和真实样本量不同，告警价值不同。桌面应用可能更适合按用户启动失败/配置异常与主动反馈上报，而不是 24×7 Pager。

## 三、结构化日志与追踪的专业最低要求

一个可行动失败事件应尽量包含 `eventType, severity, task/attempt, sourceSHA, selectedComponent, operation, causalCorrelationId, errorCode, recoverability`，但不能泄露 Token、密码、用户个人内容或原始敏感业务载荷。**结构化诊断**强调错误上下文足够定位、采取行动，并能跨进程关联，而不要求所有项目同时安装 Logs+Metrics+Tracing 的全套后台。

对异步多 Agent 任务，需要区分多种时间：事件产生时间、入队时间、处理时间、实际外部 Effect 确认时间。只按单机器 wall clock 排序不能证明跨进程因果；应结合 Attempt、parent/child Trace 和权威状态转换。K06 负责诊断原因，K10 负责提供可复查的运行测量事实。

## 四、未知外部效果与熔断/背压

当远端请求在 HTTP 超时之前可能完成时，必须先区分 `CALL_NOT_SENT / EFFECT_UNKNOWN / EFFECT_CONFIRMED / EFFECT_REJECTED`。这些名称只是**教学分类**，不是 Runtime 新 Enum。对不可逆业务 Effect，UNKNOWN 不能自动退化成“失败可重试”；应依真实 Provider 幂等、查询和补偿合同选择行为（[A32 AWS Idempotent APIs](../external-source-catalog.md)）。

超时、重试和退避会改变负载。如果持续平均输入率 `λ` ≥ 可持续服务率 `μ`，无界队列模型可能持续堆积；重试使实际输入率进一步增加。熔断可以帮助保护已不健康依赖，但未必适用于每个短调用，也可能因测量误差拒绝本可恢复的流量。队列上限、退避、系统恢复时间和任务丢失语义需作为一组权衡，不是单独加 `retry=3` 便证明高可用。

## 五、性能诊断的可证伪流程

**性能候选**：“将 95 分位响应时间降低 30%”。第一步确认目标是谁的 P95（请求、用户 Journey、后台任务）以及测量窗口与请求分布。然后采用相同工作负载/硬件/代码 SHAs 的对照，分开分析采样误差、缓存热态、外部服务时间、资源饱和度及尾延迟。

例如 CPU Flamegraph 显示占 35% 的序列化栈，仅说明在该采样/负载下分配了对应 CPU 时间，不能自动断言“改为更快序列化后端到端 P95 必然下降 35%”。若关键路径主要等待远端网络，CPU 微优化未必影响用户等待。优化后需复查 correctness 与总体费用，并留下一项会使推荐失效的反例（如序列化快但数据损坏）。

**工程经济**：日志体积、Tracing 采样、告警误报、错误处理分支及可靠性目标都增加成本。根据真实故障后果选择最少必要仪表盘/日志，不为演示观测架构而破坏用户的数据与隐私。

## 六、反例与适用边界

任务给出 “CI 全绿，但 Windows 用户首次运行出现 UNCONFIGURED，需要在应用内选择多个 GitHub 项目；运维要求 99.999% uptime”。应指出：
- GUI 可启动并进入 UNCONFIGURED 不代表具备远端任务执行权限；
- 在用户主动配置/授权后才进入目标能力 READY，且错误应提供可行动原因；
- 没有适当的适用对象/SLI 分母，99.999% 不能直接作为桌面 UI 的可执行目标；
- 如果真实业务需要稳定后台服务，则由有权 Owner 决定适用 SLI，并按目标平台的真实测量验证。

**STOP**：无法证实真实远端 Effect、SLO 范围或敏感凭据边界时暂停危险推断，不阻塞独立离线诊断。绝不把本章描述称为已运行 SRE Agent、实际告警策略或生产 E2E。

**来源线索**：[A07 ISO 25010](../external-source-catalog.md)、[A10 ISO 15939](../external-source-catalog.md)、[A31 Google SRE](../external-source-catalog.md)、[A32 AWS Idempotent APIs](../external-source-catalog.md)、[A36 OpenTelemetry](../external-source-catalog.md)、[A63 Google SRE SLO alerting](../external-source-catalog.md)。