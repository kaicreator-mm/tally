# K06.CAUSE_PROBE · 竞争根因假设、可区分实验与有界修复

> [K06 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

用户报告 UI 请求偶发失败，日志只有 timeout。到底是 Provider 网络、应用取消、业务重复提交，还是测试环境问题？

**适用前置输入：**
1. 用户可观察症状、确切日志、Host/时间窗
2. 业务请求是否可能有副作用、是否有状态回读
3. 至少两种可检验假设与差异预测

**学科原理与为什么这样决策：** 根因定位不是尽快选择一个故事，而是提出可以产生不同观察预测的竞争模型，再选择最低成本的区分实验。若两个假设都符合所有已知观察，就不能把其中一个叫“已证实根因”。有界实验需保留原始证据、环境条件、业务副作用风险及停止规则；事故复盘不能用单次缓解替代真正修复。

## 2. 实际决策程序（按现有权限执行）

1. 将 Observed（发生了什么）和 Hypothesis（为什么发生）分栏，禁止在观察栏加入未经测量的因果解释。
2. 为 H1/H2 逐条写预测：如果是网络问题，服务端状态是否更新？如果是客户端取消，Host 日志是否有已提交事件？
3. 挑不产生新的不可逆副作用的探针：读状态、比对日志相关 ID、有限模拟/复现；必要时隔离环境。
4. 根据反证排除/保留假设，找最小合法补丁和一个应因原错误失败的回归；没有能进一步区分的证据时 STOP 并写 UNKNOWN。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 两个假设均有相同输出 | 目前证据不足，不强选。 |
| 只读状态可合法确认提交 | 先回读，避免重复操作。 |
| 需要越权 Host/数据访问 | 交原 Owner，不能模拟读取成功。 |

## 4. 可复用的教学案例与负例

**合成正例：** 客户端超时，但服务端操作日志有相同 Correlation ID 的成功持久提交；这与“请求完全未到达”假设矛盾，却不自动排除客户端界面更新问题。

**失败/反模式：** 看见 timeout 就宣称数据库死锁、直接增加超时 10 倍和 20 次重试，之后再也不记录原始失败。

**可证伪的结果 Oracle：** 给 H1/H2 能不同预测的至少一条观察，并在没有真实证据时保持两者未决；若一个假设已被事实否证却仍被选定，应判为错误诊断。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K10 提供可恢复故障语义，K08 提供 Host/CI 证据，K03 仅在有权范围修复实现。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [Google SRE Postmortem](https://sre.google/workbook/postmortem-culture/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [CMU SEI Early Architecture Analysis](https://www.sei.cmu.edu/library/a-playbook-for-early-architecture-analysis/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ADS Maintenance/EOL/Hotfix](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/MAINTENANCE_EOL_HOTFIX_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
