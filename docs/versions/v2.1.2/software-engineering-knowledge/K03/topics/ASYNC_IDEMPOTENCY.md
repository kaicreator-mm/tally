# K03.ASYNC_IDEMPOTENCY · 异步调用、幂等和资源生命周期的具体代码合同

> [K03 知识库](../README.md) · [专业方法](../engineering-methods.md) · [Agent Skill](../agent-skills.md) · [反例与案例](../casebook.md) · [ADS 来源与外部学科目录](../sources.md) · [外部资料总索引](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / ORIGINAL_EXPLANATION / NOT_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。Topic 是问题驱动的知识切片，不改变 Workflow、Task、Reviewer、Host 和 Release 权限；下面案例为合成教学条件，不是实际 Host PASS。

## 1. 问题与应用触发

同一批量业务操作网络可能重试、用户可能重复点击，代码实现如何处理重复键、资源释放、取消和超时？

**适用前置输入：**
1. 已批准业务合同与幂等适用范围
2. 调用方标识/请求 ID 与持久化能力
3. 提交点、允许取消阶段、错误/超时环境

**学科原理与为什么这样决策：** “幂等”是一次业务操作在重复调用下所规定的业务效果性质，不是对任意函数套一个缓存。幂等键需绑定请求身份与业务参数；参数相同重试可返回既有结果，参数不同却使用同一键应拒绝冲突。取消不能自动回滚已经对外提交的副作用；资源生命周期必须覆盖成功、错误、取消、异常和最终清理。

## 2. 实际决策程序（按现有权限执行）

1. 明确业务副作用、何时持久提交和幂等键的作用域/存活期。
2. 正常首次执行按已授权事务写结果；重复相同键/参数读回已提交结果；同键不同参数拒绝冲突。
3. 处理迟到结果、取消与超时；无法读回时标 UNKNOWN，不能安全重复的操作不做自动重试。
4. 确认连接/文件句柄、临时文件、数据库会话等在每个错误/取消分支被适当清理，并针对最材料路径写负例。

## 3. 按条件选择，而不是背诵一种答案

| 场景条件 | 当前应作的判断 |
| --- | --- |
| 纯查询/计算且无副作用 | 无需复杂幂等存储。 |
| 持久写入有唯一约束与稳定请求 ID | 可考虑复用现有幂等资源。 |
| 跨服务不可逆操作且无回读 | STOP，交有权 Owner 设计或明确 UNKNOWN。 |

## 4. 可复用的教学案例与负例

**合成正例：** 同一 `tenantA:request17` 创建一笔订单；第二次相同参数回已有 ID，不新增订单；第三次相同 key 不同商品返回冲突。断线后客户端查询请求 ID 状态。

**失败/反模式：** 按 `Date.now()` 为每次重试生成新幂等 key，网络超时时就重复创建新订单。

**可证伪的结果 Oracle：** 并发 2 次相同键调用的最终持久副作用次数为 1，按真实已批准合同决定返回；差异参数复用键必须拒绝而不能误返回旧资源。

## 5. 最小输出、拒绝和跨模块交接

**Agent 最小输出**：当前事实/适用条件、被选知识及原因、候选/未选方案的权衡、具体正反 Oracle、缺失证据和 STOP、原始来源身份。不能取得真正 Product/Task/CI/Host 证据时要标 `UNKNOWN/NOT_RUN`，不能将本 Topic 引用的外部资料当作已发布 Tally 规则。

**交接关系：** K02 确认状态/接口所有权，K05 编写真正持久化负例，K10 负责跨应用通用重试。

**停止条件**：业务政策无权确定、错误可能影响不可逆外部副作用、需要超出合法写集的实施、无法取得必要证据时，输出受影响范围及最小合法下一步，而不是调用无界 Agent 循环。

## 6. 来源、版本与知识资格

- [AWS Builders' Library Idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [Microsoft Engineering Playbook](https://github.com/microsoft/code-with-engineering-playbook)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。
- [ADS Implementation Quality v4.19.0](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/IMPLEMENTATION_QUALITY_STANDARD.md)：**参考概念/方法，不是本案例的实际执行收据**；按出版方版本与版权范围使用。

**ADS 约束基线**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`；若与现行项目已生效规范冲突，应以具有权威的原规则为准并说明冲突。历史 ADS #1100、Draft #1384 的统一三阶段 DAG 观点不作为现行强制规则。
