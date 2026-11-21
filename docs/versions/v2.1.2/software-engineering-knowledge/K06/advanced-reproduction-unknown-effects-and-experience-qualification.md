# K06 专业深化：最小复现、未知外部 Effect 与经验的可反驳资格

> [系统课程](systematic-syllabus.md) · [R6 故障根因与区分性探针](advanced-incident-causality-and-distinguishing-probes.md) · [K05 Oracle](../K05/advanced-verification-validation-and-oracle-qualification.md)
>
> **R9 DOCUMENT_CANDIDATE / REAL_INCIDENT_NOT_DIAGNOSED / NO_PRODUCTION_EFFECT**。精确深化 **K06.F2.1 最小复现**、**K06.F3.2 超时/未知副作用**和 **K06.F5.3 可复用经验候选**。所用请求、事务、CI 日志均为原创建模，不声称读取真实生产日志或运行外部付款 API。

## 一、最小复现要保留“触发故障的最小充分条件”

一次失败的详细日志通常包含数千条无关信息。最小复现的目标并非越少行越好，而是保留**触发被观察现象所必需的输入、运行环境、状态与执行顺序**，并能重新获得同一类可辨别的失败。

示例：同一个 Source SHA 的 `contract.spec.mjs` 在 A 轮 FAIL、B 轮套件 GREEN。首先辨别**实际执行/发现**与**Oracle 预期有效性**两个不同维度。以下是需要区分、但并非全部互斥的假设与审查事项：
- H1：相同 testId 两轮都执行，因共享 DB、种子、时间竞争导致结果振荡；
- H2：B 轮 Test Discovery 未包含该 testId，因此 B 轮该测试应为 `NOT_RUN`；
- H3（Oracle 有效性问题，可与 H1 同时成立）：两轮测试都执行且都引用旧 Expected，但当前有权合同版本已经变化；旧 Expected 可以令测试结果**稳定地不具备正确性资格**，它本身不能解释同测试、同条件下的随机红绿。若结果确实振荡，还需独立查输入、时序或环境的变化。

**最短区分步骤**：固定 exact SHA/Runner、打印两个 Attempt 的 testId+run events，再比较隔离资源、输入/Oracle Edition。只有证实两轮实际执行同一测试才可将结果振荡归入 H1 的调查范围；H2 的目标 NOT_RUN 与该目标两轮均运行互斥，但 H1 的实际执行不稳定与 H3 的 Oracle 无效可以共存。单独的稳定旧 Oracle 并不导致同条件随机 Flaky。

### 可撤销测试与现实 Effect 分开

为快速复现问题，可以在本地隔离数据库构造可销毁样本，但不可把这个方式直接套到真实支付、正式 Issue 发布、真实数据迁移等不可逆 Effect。再现“可能已扣款但 ACK 丢失”应使用 Mock/纯状态模型，真实系统若没有权限和风险控制就停止；不能为“验证现象”主动进行第二次财务操作。

## 二、请求超时与远端提交是两个不同事件

合成轨迹：
```text
T0 client sends payment(order=O, key=K)
T1 provider accepts and commits the payment
T2 network drops ACK
T3 client times out and records LOCAL_UNKNOWN
T4 recovery worker restarts
```

客户端从 `T3` 观察到 Timeout 不能反证 `T1`；在允许该 Provider 将相同 key 当同一请求的真实合同**未取得时**，从 T4 直接重发付款可能造成第二次扣款。另一方面有些请求在 T1 前就被网络拒绝；仅靠“超时”也不能证明远端成功。

从技术上可区分三个状态：
- **CONFIRMED_SUCCESS**：受权 Provider 回执或查询/账本读回证明确已提交；
- **CONFIRMED_REJECTION**：受权合同和回执足以说明该尝试未提交（不能凭任意 HTTP 4xx 泛化）；
- **OUTCOME_UNKNOWN**：没有足够证据区分已提交/未提交，需要受权查询、幂等键或人工处理。

这不是对当前 Tally 付款实现的断言，只是对分布式系统不可观察事实的专业说明。局部 DB 的 `SERIALIZABLE` 事务不扩展到第三方金融系统；Event Loop 中 Promise 被 reject 也不是远端操作回滚证据。

## 三、诊断工具要选择能反驳竞争假说的最小探针

| 竞争假设 | 最小事实探针 | 有力的反驳 | 误用边界 |
| --- | --- | --- | --- |
| 本地 Cache Key 漏包含 Tenant | 冷缓存、双租户同 ID、原始 Cache Key 计算 | 两个身份的隔离键/读取完全正确且现象仍存在 | 单次 Cache Hit 不证明泄漏原因 |
| Test Runner 误选用例 | Exact SHA 的 Manifest 与实际 run events | 目标 testId 两轮均确实执行 | Job GREEN 不能冒充目标 PASS |
| Provider Effect 成功但 ACK 丢失 | 有权 Provider 只读查询/可核实回执 | 查询证明未提交且合同保证完整性 | 无权限时不得猜测远端真实支付 |
| 数据 Race 触发用户状态错乱 | 有界时序扰动/版本条件写测试 | 全部可疑交错被独立 Oracle 拒绝 | 延时或额外日志本身可能改变时序 |

为了得到真正有信息量的结论，不要求对所有模块重新复现全部现象；若最短区分性探针仍留下多个假说，应报告 `ROOT_CAUSE_UNRESOLVED`，而不是按第一条错误日志自信地选择一个根因。

## 四、临时缓解、永久修复、复盘和经验沉淀各有独立判据

一个修复方案限制并发请求，可能让告警消失，但尚未证明 Cache Key 正确，也未修复真实 Provider 业务幂等。应分别记录：
1. **Mitigation**：在何种条件下减轻了用户症状；
2. **Root-cause hypothesis**：哪些竞争解释仍然没有排除；
3. **Correction**：哪个当前 Exact SHA 变更实际修正了因果机制；
4. **Oracle evidence**：有哪些独立负例被真正运行；
5. **Experience candidate**：未来什么相同前提下可能复用这次经验；
6. **Invalidation trigger**：什么新事实会推翻经验适用性。

经验规则不应被压缩成“CI 红再跑一次就行”“网络超时直接重试”“所有 PR 必须重复三次 CI”。有用的候选是条件化的，例如：“同一 SHA/Runner 的已执行测试因共享测试 DB 引发冲突，在可销毁隔离环境中清除该状态后，此项特定污染假说得到了验证；非同类远端 Effect 不适用。”

## 五、经验化不能跳过来源和真实性

DomainHarness 的 Experience/Evolution 相关版本将来可能消费系统性执行信息；但本文没有确认其最终 ABI/策略。本章只区分：
- **Raw Observation**：原始失败/成功收据和环境；
- **Interpretation**：分析提出的根因；
- **Candidate Procedure**：在指定条件下可能减少重复错误的做法；
- **Qualified Procedure**：有权来源/验证/适用范围已经单独通过的生产用知识（**本章没有新增**）。

若把未验证经验直接变为强制 Gate，可能对所有文档修订追加无意义 Windows Installer E2E；若把真实金融超时经验误泛化成“重试可修”，风险更高。

## 六、专业练习和 STOP

给出某条 Woodpecker CI 因网络重试出现 GREEN，同时另一个执行任务中的远端 Effect ACK 丢失。要求分别写出能确认 Test Discovery 的证据、能确认远端提交状态的**受权证据**、哪一步可以安全模拟、哪一步必须 ASK/STOP。完整答案不能以“二者最终都绿了”合并事实。

**来源线索**：[A50 TLA+ 状态机制](../external-source-catalog.md)、[A51 PostgreSQL Isolation](../external-source-catalog.md)、[A52 RFC 9110](../external-source-catalog.md)、[A59 Git Diff/Merge Base](../external-source-catalog.md)、[A71 AWS timeout/retry](../external-source-catalog.md)。全部是原理/方法来源，不是当前生产故障已复现或证明。