# K10 专业深化：过载控制、性能归因与恢复完成的独立 Oracle

> [K10 系统课程](systematic-syllabus.md) · [R6 SLI/SLO、未知 Effect 与测量](advanced-reliability-slo-effect-recovery-and-measurement.md) · [K06 故障因果](../K06/advanced-reproduction-unknown-effects-and-experience-qualification.md)
>
> **R9 DOCUMENT_CANDIDATE / NO_LOAD_TEST_PERFORMED / NO_PRODUCTION_INCIDENT_CLAIM**。精确深化旧课程 **K10.F3.3 隔离、熔断和背压**、**K10.F4.2 Profiling 与瓶颈归因**、**K10.F5.2 事故响应与恢复验证**。下列吞吐、请求和响应时间是原创合成模型；不表明实际 Woodpecker、Tally GUI 或 DomainHarness Server 有这些负载。

## 一、超载是服务能力和到达率的关系，不等于“网络暂时慢”

假设一个有界 Worker 平稳每秒完成 `μ=100` 个任务，持续进入 `λ=120` 个任务/秒。在未限流、无丢弃、服务能力不变的模型下，待处理积压大致以 `λ−μ=20` 个任务/秒增长，60 秒理论净增加约 **1200** 个任务。这是简单的流量平衡模型，不是已经测得实际服务器吞吐；现实中批处理、重试、优先级、工作类型与队列容量会改变实际曲线。

当消费者超时又重试时，若原请求仍在处理，额外重试增加的到达率可能使问题更严重。**重试只是对特定错误、权威合同和时间预算下的一个选项**，不能默认为跨服务金融 Effect 的“自动恢复”。

## 二、四种过载控制的专业含义不同

| 机制 | 何时可能有效 | 失效/代价与必须考虑的内容 |
| --- | --- | --- |
| **隔离（Bulkhead）** | 某 Provider 或 Tenant 请求耗尽共享连接时限制其传播 | 隔离池总容量成本、工作分配偏差和合法优先级；无法自动证明数据库/网络完全隔离 |
| **背压（Backpressure）** | Producer 可受控减速，或有权拒绝超量任务 | 上游是否尊重信号、队列最大容量和丢弃/恢复语义；仅限流不保证旧任务都已安全提交 |
| **熔断（Circuit breaker）** | 下游持续失败/过载时限制额外请求，加快恢复机会 | 开/半开/闭合及状态重置依合同；不能把拒绝新请求当作撤销已提交的远端 Effect |
| **重试预算、退避/抖动** | 短暂可安全重试错误且 Provider 真正幂等 | 多层叠加重试放大调用；超时可能仍已提交，重试不等于幂等；需要有界尝试和观察 |

真实可靠性设计应该区分 **控制请求量** 与 **业务状态正确性**。例如下游付款服务因网络超时被熔断，业务订单仍可能处于 `EFFECT_UNKNOWN`；不能因熔断成功把它变成 REJECTED。

## 三、火焰图并不能直接解释端到端 P95

采样 Flamegraph 显示某个 JS 函数占采样 CPU 的 35%，这说明在该采样窗口/条件下它可能是 CPU 热路径。它不证明“把这段函数优化 2×，用户 P95 一定改善 17.5%”。

原因包括：
- **采样代表性**：只采了 CPU Busy 的时间，忽略网络/磁盘等待或某类低频慢请求；
- **系统瓶颈归属**：若用户 P95 主要是 Provider 远端排队，前端局部 CPU 优化可能几乎不影响尾延迟；
- **完成路径与用户端点**：测量的是函数平均 CPU，而用户感知是多个串并行阶段组合的尾部；
- **质量约束**：通过少做校验缩短响应时间，可能损害有权业务合同而并非有效优化。

合成比较：目标 User Journey 在同负载下 P95=900ms，Profile 显示某函数 CPU 平均耗时约 40ms。如果这 40ms 全部可删除且位于关键路径上，理论上才有机会减少对应路径的一部分延迟；但 P95 的变化取决于其在慢尾分布中的贡献，不能直接做“900−40=860ms”作为实测或必然值。

**可证伪测量**：指定相同业务输入分布、采样窗口、并发负载、实际 Provider、被测 SourceSHA/ArtifactDigest、度量口径与异常剔除规则，观察同一用户 Journey P50/P95、错误率及资源占用。如果 Sample、负载或部署版本不同，就应说明混淆变量而不是宣称因果提速。

## 四、恢复不等于进程已启动

事故发生后，最容易误报的结论是“容器重启成功，故障已恢复”。实际上需要按事故作用域选择合法的**用户可观察恢复 Oracle**：
1. 服务或 GUI 是否可以启动并在未配置时提供合法安全入口；
2. 已授权用户目标是否在选定 Scope 上重新成功，恢复期间是否产生了重复请求/丢任务；
3. 之前的 `UNKNOWN` 外部 Effect 是否通过受权 Provider 查询被确认或送到有权人工处理；
4. 数据/配置/Artifact 版本与恢复动作是否兼容；
5. 当前告警回落是否有意义，是否因为禁用观测器/漏选测试而“变绿”。

如果只检查 HTTP health endpoint 200、后台队列处理仍停滞，不能宣布用户任务已恢复。反过来离线桌面工具没有生产 SLO/值班队列时，不该机械引入双窗口 SLO Pager；按用户实际交付目标验证首次启动和关键路径即可。

## 五、事故临时缓解与永久修复的两种证据链

例如某队列因为持续 λ>μ 而积压：
- 暂时拒绝新请求后，队列长度下降，可以报告特定窗口的**缓解观察**；
- 若根因是重试风暴、无限 Producer 或 Tenant 无隔离，只有明确修复其机制、在受限负载下重新验证故障条件，才能报告**已修复的范围**；
- 如果付款任务有未证实结果，队列清空也不能当作业务交易全成功；应保留外部 Effect 的独立恢复收据。

以 [A71 AWS retry/backoff](../external-source-catalog.md) 和 [A72 Google SRE overload/cascading](../external-source-catalog.md) 为专业原始参考，研究的是有界缓解与失败传播机制，绝不自动证明 Tally 需要同样的业务参数或已经实际完成容量测试。

## 六、合成跨模块演习

给定 Worker 理论处理 100/s、外部入流 120/s、某个 Provider 延迟 ACK 和一个 CPU 热点的 Flamegraph：

- K10 先推断有条件的每秒净入队 +20，不将这个值当作真实运行观测；
- K06 将 Provider Timeout 维持 `OUTCOME_UNKNOWN`，不支持无授权付款重试；
- K03/K05 区分程序错误、测试未运行与独立用户路径 Oracle；
- K08 保留受检 SourceSHA/ArtifactDigest，避免拿旧 Release 数据解释新性能；
- K11 在效果数据未实测时仅给出负载控制备选及可证伪实验。

**STOP**：无法确认 Provider 幂等、真实状态或用户/资金风险未受控时，不自动重放任何外部 Effect。可以继续设计容量模型、模拟重试/过载和受限实验，但不可宣称真实恢复完成。

**参考**：[A31 Google SRE](../external-source-catalog.md)、[A36 OpenTelemetry Signals](../external-source-catalog.md)、[A63 Google SRE SLO](../external-source-catalog.md)、[A71 AWS Retry](../external-source-catalog.md)、[A72 Google Cascading Failures](../external-source-catalog.md)。全部是技术方法参考，非实测运行收据。