# K02 进阶软件架构：质量场景、状态所有权与失效权衡的完整分析

> [K02 课程](systematic-syllabus.md) · [两卷基础正文](foundations-deep-dive-01.md) / [第二卷](foundations-deep-dive-02.md) · [信任边界聚焦](focus-f4-3-minimal-failure-and-trust-boundary.md)
>
> **R6 ORIGINAL_DISCIPLINARY_REASONING / DOCUMENT_CANDIDATE / NO_ARCHITECTURE_IMPLEMENTED。** 以 K02.F1.2、F2.2、F3.1、F3.2、F4.1、F4.2、F4.3 为核心，说明如何在既定需求和资源约束下比较可证伪的架构。这里的 `Order`、`Tenant`、`Effect` 均为教学例子，不是 Tally 当前生产 Schema 或 DomainHarness 0.11.x ABI 决议。

## 一、架构决策是关于条件和后果的假说

架构不是“选一个微服务/数据库/队列”，而是为有权需求约束、质量属性、故障与变化成本选择可执行的**责任边界和机制**。参考 [ISO/IEC/IEEE 42010 的架构描述来源 A06](../external-source-catalog.md) 与 [SEI ATAM A15](../external-source-catalog.md)，先定义 Stakeholder、Concern、Viewpoint、质量属性场景与备选的权衡，再选择适当视图。

可使用如下质量场景的六要素思路（原创实例）：来源=已认证客户端；刺激=网络连接在扣款响应前断开；环境=远端可能已经提交；影响对象=订单状态/财务 Effect；响应=系统能保留 UNKNOWN 并按权威业务身份查询；度量=不出现超过许可次数的扣款，恢复判定有审计证据。缺少合同化的外部查询接口时，**不能**承诺“所有超时自动一分钟内成功”。

设计方案记录要区分假设、经授权的不变量、被试验反驳的现象和真实运维成本。ADR 只是表达架构决议的文档形态，不是任何候选“设计完就有权实施”的凭据。

## 二、跨层状态边界：局部原子性不等于全局事务

教学系统有 `OrderService`、`PaymentProvider`、`Worker`、`Queue`。将订单数据库事务设为 owner 的正确范围是**本地订单状态**。当 Worker 调用外部付款 API 时，数据库 Commit 不会自动包含外部 Effect。

候选不变量：
- I-A：任何已确认 `PAID` 都有与当前订单版本/租户一致的权威付款记录；
- I-B：不允许因 Worker 重放导致未经业务允许的多次外部扣款；
- I-C：旧任务 Attempt 不可覆盖新的订单状态；
- I-D：无权租户不可查询/修改别的租户订单。

对于 I-A，可以选择先发外部 Effect 再更新本地、先持久化 Intent 再调用远端、或独立协调/补偿流程。但不存在无需外部能力就能同时保证任意崩溃、强一致、低延迟、零冗余调用的“免费”设计。

**崩溃反例**：远端已扣款→ACK 丢失→本地仍为 `PENDING`→系统把超时当“未扣款”→重试再次扣款。即使 Postgres 本地 Serializable 全绿，也不证明远端 Effect 只执行一次。引入业务幂等 key 和可查询结果是一个**受远端合同约束的备选**；若 Provider 不支持，则必须收窄自动恢复的保证。参见 [K03 崩溃点矩阵](../K03/advanced-concurrency-idempotency-proof-sketch.md)。

## 三、边界在哪里？比较三种真实备选

| 方案 | 适用前提 | 可能收益 | 关键失败假说及检验 |
| --- | --- | --- | --- |
| 单进程服务 + 单数据库事务 | 状态 owner 明确、吞吐足够、外部 Effect 可隔离 | 部署/诊断简单、局部状态原子 | 本地吞吐无法满足已核实目标；重启恢复必须有持久 Intent |
| 后台队列 + 幂等 Handler | 入队与恢复合同可核实，生产者/消费者身份稳定 | 突发流量缓冲、非同步业务处理 | ACK 丢失/重复投递、队列积压、Tenant cacheKey 碰撞 |
| 多服务/事件流 + 补偿 | 独立伸缩或组织边界真实存在、跨服务最终一致可接受 | 部署隔离、独立容量与团队所有权 | 消息乱序、业务回滚不等于财务撤销、全链路可观测性不足 |

不能以表中第二、三行“更现代”为由默认推荐复杂系统；也不能把 Saga 当作数据库全局原子性的同义词。对微服务维护成本、可恢复性与组织责任，应由真实使用场景和 Source 数据决定。

## 四、错误和信任边界是两张不同地图

故障边界考虑超时、崩溃、网络分区、资源饱和；信任边界考虑 Actor、Tenant、可使用的 HostPort/Effect 和输入的真伪。二者都影响架构，但**BUILD 期间 Package→声明 Extension import 闭包**不等于运行中取得特定 Host/Effect 权限。即使包只能调用已声明 Extension，其外部副作用仍需经独立的 Host 能力合同授权（对应 [R4 聚焦](focus-f4-3-minimal-failure-and-trust-boundary.md)），不能因“依赖已声明”就允许外部写入。

**跨租户反例**：缓存采用 `taskId` 而不是包含受权租户身份和合法状态版本的安全键，两个租户恰好拥有同一 taskId；缓存命中可能泄露他人的任务内容。即使租户身份记录进日志，也不等于缓存数据隔离真正成立。

## 五、在受限预算下做可证伪架构分析

示例需求：一次导入 20 万条资料，运行中可崩溃恢复，目标完成时间由有权 Owner 尚未决定。常见草率建议是“必须使用 Kafka + Kubernetes”。更合理的分析是：
1. 先列真正受权的不变量：例如导入记录不能丢、重复记录有稳定的业务身份、用户能够看到恢复状态；
2. 把尚未批准的吞吐时间、部署平台、峰值写为 UNKNOWN；
3. 先用可恢复单进程+持久进度与简单后台队列作为**候选**，列出流量/隔离/运维约束触发升级的阈值；
4. 分开测试崩溃点、重复输入、事务失败和使用者可观察进度；
5. 只有测得的需求确实不能满足时才选择更复杂架构；同时保留退化方案和成本证据。

架构 Demo 可以证伪设计假设，例如“只有一个 Worker 仍能满足已批准吞吐”“Crash 后不会重复扣款”。一个 Demo 通过不能证明真实部署全 SLA，也不能把无权限的生产 Effect 作为试验负载。

## 六、知识消费推理的边界

K02 输出候选责任分布、不变量、视图/权衡、替代方案和最小验证计划。K01 拥有业务 Rule 的来源资格，K03 拥有具体编码/原子实现，K05 拥有可证伪验证方案，K10 拥有运行指标与未知 Effect 恢复，K11 拥有成本/收益实验。K02 不重复创建这些模块的语义 Owner。

**题目反例**：当真正的系统只有单用户、完全本地只读操作、没有跨服务 Effect 时，强迫设计多租户授权/分布式 Saga 并不是“专业完整”，而是违反适用条件。

**参考线索**：[A06 ISO/IEC/IEEE 42010](../external-source-catalog.md)、[A07 ISO/IEC 25010](../external-source-catalog.md)、[A15 SEI ATAM](../external-source-catalog.md)、[A16 SEI QAW](../external-source-catalog.md)、[A26 Pact](../external-source-catalog.md)、[A32 AWS Idempotency](../external-source-catalog.md)；只作为公示的学科来源，逐条可用于产品的 Claim 尚未核定。