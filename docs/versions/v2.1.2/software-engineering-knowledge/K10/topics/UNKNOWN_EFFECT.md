# K10.UNKNOWN_EFFECT · 超时、取消与可能已发生的远端副作用

> [K10 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

客户端提交订单请求后超时并允许用户再次点击。系统能否直接判定失败并重复提交？

**开始前必须确认：**
1. 业务操作的实际副作用和提交点
2. 服务端查询状态/幂等键的真实合同
3. 超时/取消/重试次数、延迟/窗口的合法要求

**学科知识与因果机制：** Timeout 描述调用方在给定时间内没有拿到确认；并不说明服务器是否已执行、提交是否原子、ACK 是否丢失。Cancel 也可能只停止客户端等待，不回滚已持久化的远端操作。安全恢复首先依赖当前业务操作是否有可重放的幂等键或可验证状态回读；未知情况不能因重试成功就擦掉最初未确认事件。

## 2. 可执行的有界判断过程

1. 区分确定性拒绝、临时网络故障与提交状态 UNKNOWN。
2. 可合法回读时，先根据请求 ID/幂等键读出状态；确认已提交则直接返回已有结果，不再次执行副作用。
3. 只有确认重复调用安全且在有界预算/截止内时才实施重试；对无法确认又不可逆的操作停止自动重试。
4. 分别记录用户可行动消息、技术相关 ID、恢复事件和最终实际验证，历史故障不可被覆盖。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 服务端已提交且能读回 | 展示已完成结果，不重复提交。 |
| 确定性业务拒绝 | 反馈用户修正输入，不做网络重试。 |
| 状态无法确定且无幂等合同 | 保留 UNKNOWN，要求授权排查或人工处理。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 订单 `req-17` 成功写入 DB 但客户端 ACK 丢失；后续按幂等键查得订单 ID 42，重复点击不创建 43。

**失败路线/反模式：** 所有 timeout 都当失败，客户端自动用新的请求 ID 重发，最终同一商品产生两笔订单。

**最低可证伪 Oracle：** 在注入“服务端提交成功但客户端超时”时，系统的可观察业务结果最多一笔订单；不能给未确认的重试路径生成假成功。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K02 规定状态 Owner/接口合同，K03 实现，K06 用真实事故日志诊断；K10 维护一般恢复语义。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [AWS Idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [Google SRE Incident Response](https://sre.google/workbook/incident-response/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ADS Incident Recovery](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/INCIDENT_RECOVERY_FEEDBACK_STANDARD.md)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
