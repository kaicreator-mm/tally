# 软件工程领域智能进阶演算第二卷：W06–W12（架构→交付→运行）

> [专业知识库](README.md) · [第一卷 W01–W05](professional-reasoning-worked-traces.md) · [165 个学科知识点](foundation-concept-coverage-ledger.md) · [模块 R6 深度地图](systematic-foundation-depth-map.md)
>
> **R6 SYNTHETIC_TEACHING_TRACES / NOT_AGENT_EXECUTED / NO_RUNTIME_OR_SOURCE_QUALIFICATION**。七个演算来自新增 K02/K04/K06/K07/K08/K09/K10 专业正文的机制，不是实际 GitHub Task、CI、Host、生产事故或模型得分。每个演算都列来源/假定、机制、选择、可证伪反例、当前无法给出的证明及最小下一步。

## 1. W06 — K02：缓存隔离与分布式付款边界

**输入**：Tenant A/B 同时使用 `taskId=72`；服务端缓存 key 只有 taskId；一个付款 Worker 在远端提交后超时。产品文档声称“数据库事务确保全部跨服务 exactly-once”。

**事实分解**：`tenant/taskId` 冲突与远端 timeout 是教学假设；真实 Effect API 幂等合同、Caller Actor、租户安全边界以及真实 Order 表事务隔离**未知**。数据库事务只明确自己的提交范围。 [K02 架构专业正文](K02/advanced-architecture-invariants-and-failure-tradeoffs.md) 导出“跨租户无泄漏/旧 Attempt 不覆盖/不得重复不可逆效果”的三项候选不变量；[K03 崩溃矩阵](K03/advanced-concurrency-idempotency-proof-sketch.md) 解释远端提交不因本地 rollback 消失。

**方案**：简单服务 + 原子本地 Intent + 远端可查/幂等（若 Provider 真有）；或在没有该合同前隔离 UNKNOWN、走授权对账流程。不因运行低流量自动推荐 Kafka，也不把 `Package→Extension` 声明当 Host/Effect 授权。

**证伪**：两个租户同 key 读回串租户内容；远端一次成功后丢 ACK 导致第二次付款。**错误回答**：“DB transaction PASS + Extension 声明完成，所以缓存隔离、远端 exactly-once 和 Effect 权限都已证明。”**合法输出**：架构候选、相应 Source/Owner 未知与负例计划；不是生产授权。

## 2. W07 — K04：有限 Agent 资源下的关键路径

**输入**：A 合同 2h，B 服务端 4h，C 客户端 3h，D 集成 2h，E 文档 1h，前件 A→B/C→D；仅两名 Builder，共享 schema 需要单一权威。

[K04 的专业推导](K04/advanced-task-graph-critical-path-and-bounded-delivery.md) 给出**不计资源限制的理论关键路径** `2+4+2=8h`，而不是全部任务顺序相加 12h。两名 Builder 在 A 完成后可做 B/C，E 是否并行还取决于它是否依赖已确认版本以及资源分配；若两名 Agent 都改同一个 Schema，就要由一个 Shared Owner 解决冲突或先冻结合同。

**最小反例**：C 编译的是 A@H1，B 实际改成 A@H2，D 用旧 Mock 获得绿色但并未测试实际同版本消费者。**正确回答**：D 当前不能因 B/C 分别 DONE 宣称完成，需核对精确 ABI/SourceSHA 和真正集成测试。**停止策略**：无材料新证据时停止无限细分任务，优先完成有界集成。

## 3. W08 — K06：CI “第一次红第二次绿”的竞争假设

**输入**：同 SHA 的两次 CI，首次 contract-e2e 拒绝字段版本，第二次通过，日志显示首次 schema generator 是 cache hit；有人推断“首次只是网络抖动”。

[K06 诊断专业正文](K06/advanced-incident-causality-and-distinguishing-probes.md) 构造 H1 源码生成错误、H2 缓存遗漏 Schema/工具链输入、H3 消费者 Expected 本身过期，及最小区分探针：核对当前 SourceSHA/Schema/Input digest；在等价工具链下冷构建并比较 Artifact hash；检查当前有权消费者合同；仅补跑受影响的测试。执行时间先后不等于因果关系，`cache hit` 也不是自带罪证。

**反例**：即使第二次绿，也可能是首次测试未发现/并发测试抖动/旧缓存条件改变；就算冷构建修复了症状，若相应配置不同，还不能断言真正修正了缓存键。**结果**：多个可证伪假设和最低探针，不报告实际根因已确认。若没有原始 CI 日志，状态是 `INCIDENT_NOT_OBSERVED`。

## 4. W09 — K07：两个 Agent 的延迟 ACK 和共享语义写集

**输入**：A 基于 H1 修复 `auth.ts`，B 基于 H2 更新 `router.ts`；A 的旧 Attempt 消息迟到且声称 DONE；两个文件名不同。

[K07 进阶协作](K07/advanced-agent-concurrency-provenance-and-review-isolation.md) 区分 Task/Attempt/SourceSHA/ArtifactDigest/Reviewer 与有权正式决议；[K04](K04/advanced-task-graph-critical-path-and-bounded-delivery.md) 识别 Router 可绕过 Auth 的**读写语义冲突**。A 的 H1 审核对历史范围仍有效，但无法自动覆盖 H2 引入的认证路径。应在 H2 的受影响用户路径上读取真实 Diff、运行必要 Oracle，并由与 Writer 不同的合法审核会话限定复核。

**反例**：把最后到达的 A@H1 DONE 当当前 H2 的审核成功。**合理输出**：H1 HISTORICAL_ACCEPTED_SCOPED / H2 AFFECTED_SCOPE_PENDING，不能凭 Agent 数量或相同 GitHub 账号推断审查独立性无效；真实独立性取决于不同写作/审查会话和有权约束。

## 5. W10 — K08：19 组 Workflow 的网络成本与缓存错误

**输入**：一次 Pipeline 有 19 个 Workflow，每个都含 Clone；假设每次实际下载 300MiB；H2 更换 Schema 但缓存键遗漏生成依赖；CI test exit 0，却没有目标平台安装验收。

[K08 构建研究](K08/advanced-hermetic-ci-artifact-identity-and-target-host.md) 计算**理论**源传输 `19 × 300MiB = 5700MiB ≈ 5.57GiB`，但 Provider 的底层 Git object cache 可复用，所以光看 YAML 的 19 个 Clone 步骤无法确证实际传输。候选优化是一个以精确 SHA 密封的 Source Artifact 分发到隔离 Workflow，前提是每个 Workspace 能验证制品完整性，并有真实 Provider 支持。**不是**让各 Workflow 使用不受控可变共享目录。

**双重反例**：Schema 改变不引起缓存重编译；Runner 忽略 `*.spec.mjs` 却报告“测试全通过”。**不能推断**：单次 clone、所有测试实际执行、Package 可安装、Windows Host PASS。需要真实 Pipeline 日志/编译输入/当前 SHA 安装试验才可晋升相应事实。

## 6. W11 — K09：哪些版本需要兼容、哪些需要完整重写

**输入**：生产中存在旧 API Consumer 和新客户端两种版本，同时数据库 `full_name` 要拆列；另一条重构产品线的 Owner 已明确“不需要兼容旧版本，但必须在归属子版本内部自洽”。

[K09 演进正文](K09/advanced-contract-evolution-migration-and-retirement.md) 分开问题：对于确实同时存在的旧/新 Consumer，应选定受权的过渡期策略，可用 Expand–Migrate–Contract 并审查双写、非无损映射与真实退役时点；对于明确完整重写且不支持旧接口的分支，应在当前被指定的子版本修正矛盾，而非强加旧 API Shim。二者**不是一个统一“永远兼容”的决策**。

**强反例**：将 `full_name=Prince` 无损自动拆成合法的 `first_name,last_name`，或者把历史 `canceled` 直接映射为“外部退款已完成”。这些需要业务/数据来源和 UNKNOWN。**真实验证**：受支持消费者合同、历史数据采样与独立业务 Oracle，不能凭 Migration 脚本存在宣称回滚安全。

## 7. W12 — K10：桌面启动、错误预算和不可逆 Effect

**输入**：Windows 桌面程序首次启动没有预配置项目，UI 可进入设置页面；有人将其视为“必须是服务不可用事故”。另一个远端付款调用超时，团队以 HTTP 504 为由无限重试。

[K10 可运行性](K10/advanced-reliability-slo-effect-recovery-and-measurement.md) 先识别两个不同用户目标：桌面首次启动允许 `UNCONFIGURED`，但需要清晰提示/安全秘密存储/不发未授权网络动作；远端付款必须把 `EFFECT_UNKNOWN` 与 `CONFIRMED_REJECTED` 分开。是否应定义可用率 SLI、SLO 或 Pager 取决于有权用户 Journey、真实分母与故障职责。

**教学数例**：30 天有 100,000 个经定义 Eligible 的 API 请求，99.9% SLO 允许 100 个不符合；若观测 120 个，应记录超出预算，而不是因 CI GREEN 自动宣告 99.9% 满足。也不能将“桌面尚未设置项目”自动算作远端后台服务宕机。

**反例**：HTTP timeout 后把未知付款当失败重新扣款，或泄漏 Secret 到普通日志。**合法结果**：按远端幂等/查询合同读回，未知时暂停危险重复效果；对桌面启动单独验证 GUI/权限/配置 Oracle。无真实日志时不是生产 SLI 收据。

## 8. 跨演算的专业知识可用性原则

这七条的真正考核对象是**能不能给出受权前提、选择适用原理、排除竞争错误方案并指出哪个实验可推翻推荐**。一次教学推导不等于 Agent 已实际完成同等推理，也不等于案例中涉及的远端 Host/Effect 已经取得资格。

现有 Q01–Q18 是短案例，W01–W05 是前五条完整教学演算，W06–W12 是新七条；**12 个 W 全是教学合成例子，没有一次是真实受控 Agent 试验**。新的引用不更改 48 条原 Concept 关系或 165 个原始学科 ID。实际使用前仍需检查 A01–A64 出版方范围、许可、具体业务 Claim 与当前合法消费者。
