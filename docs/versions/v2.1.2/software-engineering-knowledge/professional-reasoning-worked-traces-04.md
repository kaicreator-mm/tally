# 软件工程领域智能完整推理第四卷：W18–W22

> [第一卷 W01–W05](professional-reasoning-worked-traces.md) · [第二卷 W06–W12](professional-reasoning-worked-traces-02.md) · [第三卷 W13–W17](professional-reasoning-worked-traces-03.md) · [R9 原理到来源候选准入](professional-source-to-claim-readiness-r9.md)
>
> **R9 SYNTHETIC_WORKED_CASES_ONLY / REAL_AGENT_OR_CI_NOT_RUN / NO_EFFECT_OR_RELEASE_AUTHORIZATION**。本卷五例全是原创合成输入，给读者/未来 Agent 使用的“条件—反例—Source/Oracle—STOP”范式，不代表真实 Codex、Host、CI 或故障恢复被测。此前 W01–W17 的内容和接受范围未被修改。

## W18 — 发生 HTTP 504，应该重试付款还是先确认状态？

**事实（合成）**：用户在 Tenant B 发起付款；服务收到 HTTP 504；DB 本地事务保存了 Intent；网关日志不能判断 Provider 是否已 Commit；有人提议无限重试。此时也没有拿到 Provider 正式的业务幂等合同。

**路由**：[K03 错误语义](K03/advanced-error-semantics-observability-and-validation-proof.md) 说明观测的 504 不等于远端未提交；[K06 未知 Effect](K06/advanced-reproduction-unknown-effects-and-experience-qualification.md) 给出远端已提交而 ACK 丢失的反例；[K10 背压](K10/advanced-overload-performance-inference-and-recovery-oracles.md) 指出未经限制的重试可增大过载。

**决策**：保持 `OUTCOME_UNKNOWN`；先用有权的只读 Provider 查询/已有可信回执，或向支付 Owner 请求幂等/状态合同；未获授权时 **STOP 外部再提交**。若 Provider 经核实支持稳定业务键、有特定幂等期限，则才可按其合法合同设计受限恢复。

**否证**：若能取得有权远端账本明确该 key 已提交，就推翻“失败必未扣款”；若只能看到本地 `PENDING`，不能据此推翻远端提交。整个案例未调用真实 Provider。

## W19 — 类型检查绿色，但新的合同测试根本没运行

**事实（合成）**：H1 已通过 `*.test.mjs`；H2 新增 `contract.spec.mjs` 和 Schema 改动；Typecheck PASS，旧 Runner glob 只发现原 40 项，CI Job 显示绿；Reviewer 沿用 H1 成功就准备标记产品验收。

**路由**：[K03 编译/发现/执行](K03/advanced-error-semantics-observability-and-validation-proof.md) 与 [K05 真实性](K05/advanced-verification-validation-and-oracle-qualification.md) 分出 Typecheck、Test Discovery、真实 Execution、合法 Expected；[K09 影响闭包](K09/advanced-configuration-migration-and-evidence-impact-closure.md) 显示 Schema/生成器变化影响实际 Consumer。

**正确状态**：旧被运行 40 项可按其当时条件报告 PASS；H2 新的 `contract.spec.mjs` 是 `NOT_RUN`，不能宣称全部合同测试通过。应核对当前 Exact SHA 的 Test Manifest、Runner 版本/Globs，并仅对受影响合同执行必要测试；不因无关 README 变更就全量重跑 Windows Installer。

**否证**：若新测试真实出现在 Test Discovery 且有该 testId 的有效 run/assert/exit 记录，方能推翻未运行的判断；仅有 Job GREEN 或测试文件存在不足够。

## W20 — Agent 遇到未知正式需求，是否应生成更多 Task？

**事实（合成）**：某个客户想“付款失败时自动重试”。唯一正式政策为旧版 PDF；最新消息是未经批准的草案；Planner 提出 12 个并行 Builder 和三层 Review DAG，在未核实幂等的情况下先改产品代码。

**路由**：[K04 风险和信息缺口](K04/advanced-risk-adaptive-workflow-and-information-gaps.md) 将目标、事实、假设、权限和必要下一个动作分离；[K01 有权 Source](K01/advanced-stakeholder-authority-and-triangulated-evidence.md) 明确最后发消息的 Actor 不一定有权改政策；[K11 工程经济](K11/advanced-decision-risk-value-of-information-and-process-cost.md) 说明过量编排的成本。

**正确行动**：先问有权业务 Owner “重试是再次扣款还是查询旧结果？已批准合同适用于哪些租户与版本？”；允许无外部 Effect 的只读调查和模拟状态图；冻结危险自动重试的合并/运行，不必停止一切文档/测试方案工作。

**否证**：若真有当前受权规则、Provider 幂等和安全回执，可以继续界定实际受影响 Task DAG；仍不能因为有了规则就让 Builder 自己签发独立 Review。**STOP** 缺失的业务授权。

## W21 — 升级配置并回滚后用户的数据为什么没有恢复？

**事实（合成）**：交互式应用从单项目环境变量升级到多项目启动后配置。迁移把项目 Token 复制到所有项目实例；数据模型将 `CANCELED` 当 `REFUNDED`；上线出错后镜像恢复 H1，所有人都说“代码已经回滚”。

**路由**：[K09 语义迁移](K09/advanced-configuration-migration-and-evidence-impact-closure.md) 区分 Boot/Config/Authorized、Tenant Credential Owner 和不可逆数据变换；[K08 Release/回滚](K08/advanced-release-provenance-permission-and-rollback.md) 区分旧镜像和数据状态恢复；[K01](K01/advanced-requirements-traceability-and-change-propagation.md) 说明业务取消与财务退款不是同一个状态。

**最低验收 Oracle**：能在目标 OS 首次启动进入配置入口，未完成权限/项目选择前不发远端写操作；两个项目的 Token/仓库/版本引用不串扰；必要数据迁移保留精确来源和可恢复性；已提交外部 Effect 不靠回滚镜像“撤销”。未执行真实 Windows/Linux 安装与迁移演算，全部只能作为未来测试设计。

**否证**：若实际迁移具有有权并已验证的无损双向映射，并独立验证回滚后的消费者读取/Effect 状态，才能逐项补充相应范围的 PASS。HTTP health 200 不足以证明这些能力。

## W22 — 排队与 Flamegraph 数据如何导致错误事故总结？

**事实（合成）**：某服务处理上限假定为 100 请求/秒，持续进入 120 请求/秒，且客户端在超时后重试；CPU Flamegraph 35% 显示某 JSON 处理函数为热点。事故处理员把某容器重新启动后宣称“性能问题已完全恢复”。

**计算**：在无丢弃、服务速度固定的简化模型中，队列约每秒增长 20 条，60 秒约增加 1200 条。35% CPU 热点是采样 CPU 占比，不能推导端到端 P95 改进 35%。[K10](K10/advanced-overload-performance-inference-and-recovery-oracles.md) 需要用户 Journey/队列/负载/失败模式实际观测，[K06](K06/advanced-reproduction-unknown-effects-and-experience-qualification.md) 需要排除根因假说。

**方案**：以有权限的条件化背压/节流/隔离和有界重试控制拥塞，保留已提交但 ACK 丢失的 Effect UNKNOWN；选择具有代表性的负载/目标 Artifact 与真实用户 P95/错误率/数据状态 Oracle。CPU 采样、服务启动和绿色健康检查都不足以确认完整恢复。

**否证**：相同受控负载下用户 Journey/队列/容量及负面业务指标均有可核查的改善，才能报告选定范围的改善；本例没有真实压测。不能让本教程强制全部 GUI 应用安装 SLO Pager。

## 文档资格和下一步

五条跨模块演算只演示如何从 K01–K11 中选真正相关的学科内容，遵守专业来源的适用/时效与业务权限边界，尤其在 `NOT_RUN`、`OUTCOME_UNKNOWN` 与 `NEEDS_AUTHORITY` 上正确停止。增加 W 不新增真实 Native Task DAG、Agent Skill、Provider 接口、DomainHarness Consumer ABI 或合格 Source Claim；这些只能由未来独立拥有者按真实选定版本和实际验证取得。