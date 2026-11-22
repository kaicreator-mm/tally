# K02.STATE_OWNER · 跨模块状态所有权、事务边界和重复投递语义

> [K02 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

API、Worker、数据库和客户端共同参与 Job 完成状态，谁允许写 `COMPLETED`？客户端超时或 Worker 重试如何防重复副作用？

**适用前置输入：**
1. 状态机、事件和业务不变量
2. 组件部署/持久化边界和真正存在的接口
3. 是否允许重复请求、幂等键与外部副作用

**学科原理与为什么这样决策：** 数据/业务状态所有权不是某文件在哪个仓库的问题。需要明确最终可信的提交点、哪些组件能改变状态、读者通过何处得到当前结果、重复事件怎样被拒绝或合并；分布式系统没有简单可靠的“请求超时=写入失败”。可以采用单一状态写入者、事务原子更新和幂等操作合同，但不在架构文档里强加所有项目的数据库实现。

## 2. 实际决策程序（按现有权限执行）

1. 按一次业务调用画状态变化时间线：接收、校验、执行业务、持久提交、发布事件、客户端确认。
2. 指定一个当前最终写入 Owner 和各阶段可观察/读取的事实；若是多 Writer，明确一致性机制与冲突解决责任。
3. 针对同一业务请求重复、乱序事件、提交成功 ACK 丢失和进程重启定义结果/读回合同。
4. 交 K03 当前 Task 的接口/错误实现，交 K05 设计真实持久化与迟到消息负例；模型 Demo 不代替 Host 测试。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 本地数据库同事务 | 优先复用当前事务/唯一键和幂等资源。 |
| 外部服务不可原子提交 | 承认部分成功/补偿/UNKNOWN，先核合法状态回读。 |
| 没有可信状态/回读 | 不宣称 exactly-once，也不能强行重试不可逆操作。 |

## 4. 可复用的教学案例与负例

**合成正例：** Worker 收到相同 `job-42` 两次，业务结果只能写入一次；首次执行已完成但 ACK 丢失，后续读状态得 `COMPLETED`，不再次扣减库存。

**失败/反模式：** Worker 与 API 都把 `UNKNOWN` 当待执行并再次出库，导致实际业务操作重复。

**可证伪的结果 Oracle：** 在重复投递和 ACK 丢失两个反例下观察同一 Job 的副作用计数与状态，超时导致的 UNKNOWN 不得被系统偷偷标为失败并重复执行业务。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K03 负责具体代码，K10 负责通用重试与恢复模式，K06 只针对实际事故诊断。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [ISO 42010 Architecture description](https://www.iso.org/standard/74393.html)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [AWS Builders' Library Idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [SWEBOK Software Architecture](https://www.computer.org/education/bodies-of-knowledge/software-engineering/topics)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
