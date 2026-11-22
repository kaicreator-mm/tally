# K02 进阶架构：扩展依赖、运行授权、数据 Owner 与消费者版本的正交边界

> [系统课程](systematic-syllabus.md) · [R6 架构不变量与失效权衡](advanced-architecture-invariants-and-failure-tradeoffs.md) · [K09 版本演进](../K09/advanced-contract-evolution-migration-and-retirement.md)
>
> **R8 DOCUMENT_CANDIDATE / NOT_A_DOMAINHARNESS_ABI / NO_RUNTIME_EFFECT。** 精确深化 **K02.F1.3 ADR**、**K02.F2.3 依赖与扩展点**、**K02.F3.3 持久化和数据模型**、**K02.F5.1 版本与兼容边界**四个原课程点。引用相关 DomainHarness 0.11.x 概念只作为有权产品分层边界的研究例子；本章不定义新的 Package Element 字段或运行时加载合同。

## 一、四张边界图，不能合并成“模块能调用就是有权”

| 边界 | 真正回答的问题 | 正例限定 | 混淆后的错误 |
| --- | --- | --- | --- |
| 依赖/扩展声明 | 源码或确定性 BUILD 可以引用什么接口？ | Package 显式声明直接 Extension；只允许合法直接依赖 | 有 A→B→C 就允许 A 越过 B 直接调用 C |
| 业务身份/能力授权 | 谁能对哪个对象/租户执行哪种 Effect？ | Actor、Tenant、Operation、Host scope、有效时间均被合法核实 | 将声明 Extension 当作拿到某个外部 HostPort 的全部权限 |
| 状态/数据所有权 | 哪个提交边界决定某个事实的版本？ | 本地 DB row version 在该事务里 CAS | 本地 COMMIT 被误当成远端付款已扣/已退 |
| 消费者版本合同 | 在什么时候由哪个 Consumer 读取或解释接口？ | 明确允许重叠版本或明确禁止旧版兼容 | 在完整重写系列擅加旧 Shim，或在确有旧客户端时强删旧字段 |

这四种边界之间需要**清楚的对应关系**，却不存在一个可以同时证明依赖、授权、外部状态与兼容性的“总许可证”。一个 Extension 已声明不能证明它内部新访问的网络 Host 就自动获得业务权限。

## 二、反例：两跳依赖不是直接依赖

教学拓扑：`A -> B -> C`，A 显式声明 B，B 显式声明 C。若确定性 BUILD 合同规定 Package 只能使用**自己声明的直接 Extension**，A 调 C 即使 TypeScript 通过、构建工具能解析并且 C 是传递依赖，也不能据此认为架构合同满足。A 必须改为明确声明 C（若合法）或者通过 B 公开的受约束接口完成需求。

反过来，“直接声明 C”也仅涉及依赖闭包，不是 Host 的外部 Effect 授权。若 C 请求创建真实仓库 Issue，还必须有受权的 Host Effect/Actor、目标仓库、操作范围、执行阶段与可观察回执。不得在 BUILD 静态规则不存在或版本未知时假定其已被某一真实 Kernel 自动执行。

**反证步骤**：一个构建样例让 A 使用未声明 C：静态规范若成立应拒绝；再在声明完整的前提下模拟未获 Host 授权的业务操作：应由真实合法权限边界拒绝。只看到第一种 PASS，不能推导第二种 PASS。两种实验本文均 `NOT_RUN`。

## 三、状态所有权、数据库约束和外部事件的具体分裂

合成付款状态：`OrderV7.status=PENDING`、外部 Provider 可能已扣款。
- 事务 T1 提交本地 `Intent(sendPayment,orderId,key)`；
- Worker 调用远端 `pay(key)`；
- 远端已经提交；
- ACK 丢失；
- 本地仍 `PENDING`。

即使数据库采用 Serializable，也只能说明在数据库事务边界内满足相应隔离；不能把远端付款并入该事务。重放任务时，必须依 Provider **真实的幂等/查询合同**决定查询或安全重试是否可行；未知则记录 `EFFECT_UNKNOWN`，不靠字段名猜测。

**数据 Owner 冲突的反例**：把 `Order.status` 同时由服务 A 与服务 B 当作独立权威写入，一个根据已发货消息更新 SHIPPED，一个用旧读写回 CANCELED。解决思路要考虑版本条件更新、消息语义与状态机有权规则，而不只是新增一个 Kafka Topic。跨数据库事务/补偿的可行性同样依赖不同系统的可查询/可撤销能力。

## 四、ADR 是包含失败条件的可回评论证，不是永久批准

一份小 ADR 应至少写明 `Decision/SourceOwner, Context/Status, Alternatives, ChosenMechanism, Assumptions, FalsifyingScenario, Consequences, RevisitTrigger`。示例：

- 假定每次批量导入少于 10,000 行、读写完全本地、无跨租户隔离要求，先采用单 DB + 后台批处理；
- 若实测每次 300,000 行、当前用户 Journey 目标已被有权制定并持续违反，触发重新评估队列与批量写入机制；
- 若新的 Tenant/外部 Provider 真正进入范围，重新评估 Host 权限与分布式 Effect，而不是继续引用旧 ADR “当时通过”；
- 若只是另一位工程师偏好微服务，没有新的约束/证据，旧 ADR 不必自动作废。

上面数字是**合成阈值**，非真实需求或性能测量。架构图、ADR 与 Source 是否正确、代码是否真的实现、真实性能如何、Reviewer 是否批准，是分开记录的事实。

## 五、版本兼容是一项有 Owner 的决议

场景甲：v1/v2 Consumer 同时活跃且合约要求不停机迁移，则 Expand–Migrate–Contract 可能合理；需要证明旧 Reader、新 Reader、回填和数据有损规则。
场景乙：有权版本策略明确一个系列**完全重写且不兼容旧版**，必须在归属版本直接解决不一致，不能“先引入旧 Shim、在下一个子版本修”；旧 Consumer 若依赖别的版本，应由真正的迁移/退役工作承担，不得偷偷混入新版本 ABI。

**错误方案**：“我们对所有版本都要求兼容，毕竟这是最佳实践。”该说法未指定受支持消费者集合与实际重叠期，也可能直接违背有权产品需求。

## 六、最小选择矩阵与跨学科交接

| 需求/证据情境 | 首先求证 | 不可擅自做的事 |
| --- | --- | --- |
| Package 未声明直接依赖 | BUILD 目标合同和来源版本 | 借传递依赖绕过声明 |
| 扩展已声明，但外部付款要提交 | Host/Actor/Tenant/Effect 权限合同 | 将构建合法当作业务授权 |
| 本地事务成功、远端 ACK 丢失 | Provider 幂等/查询边界 | 把远端 Effect 记录成失败 |
| 旧 Consumer 是否仍在用 | 受权版本策略与部署事实 | 一律增加或移除兼容层 |
| ADR 性能前提失效 | 实测负载、目标与适用范围 | 把图纸自动提升为当前实现证明 |

[K01](../K01/advanced-stakeholder-authority-and-triangulated-evidence.md) 负责真正有权规则；K03 编码/错误边界；K05 独立 Oracle；K08 Build/Release 证据；K09 Consumer 迁移；K10 可运行性。本章只给**架构决策的学科解释**。

专业来源：[A06 ISO/IEC/IEEE 42010:2022](../external-source-catalog.md)、[A15 SEI ATAM](../external-source-catalog.md)、[A26 Pact Consumer/Provider 合同](../external-source-catalog.md)、[A32 AWS Idempotent APIs](../external-source-catalog.md)、[A51 PostgreSQL Isolation](../external-source-catalog.md)、[A64 Parallel Change](../external-source-catalog.md)。以上不能自动为真实 Tally 构建/运行/Release 给出审核资格。