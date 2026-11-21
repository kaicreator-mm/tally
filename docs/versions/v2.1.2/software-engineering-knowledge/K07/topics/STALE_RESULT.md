# K07.STALE_RESULT · 迟到消息、过期 Head 和未知外部效果的对账

> [K07 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

一个远程 Agent 在旧 HEAD 上已经开始处理，通讯超时后当前 PR 已更新，旧 Worker 又返回 `ACK/SUCCESS`；怎样处理旧结果和未确认副作用？

**开始前必须确认：**
1. 消息 Role/Task/Attempt/SourceSHA/ArtifactRef/Correlation
2. 当前 GitHub Task/PR HEAD、合法 Writer 与 Review 收据
3. 外部副作用能否读回、是否具备幂等合同

**学科知识与因果机制：** Task 身份、Attempt/Correlation ID、Source SHA、Artifact 和真实 GitHub 状态是相互关联却不相同的事实。消息 ACK 仅说明某次消息报告送达或处理，并不等于源码已经合法集成、测试已经通过或用户动作未重复。超时也不等于 Worker 没有执行。受影响的制品应绑定其准确源，迟到旧消息必须先审查而不能覆盖新候选。

## 2. 可执行的有界判断过程

1. 读当前权威 Task/PR HEAD，对比迟到消息中的 SourceSHA/Attempt，区分 STALE、CURRENT、UNKNOWN。
2. 把 ACK 或工具返回作为传输事件存档；如有 Artifact，检查是否实际存在以及其源码和测试是否对应旧 HEAD。
3. 对于已过期成果，只能按合法复用审查流程选择局部迁移/重新验证，不自动覆盖当前文件和资格。
4. 若远程任务可能产生不可逆副作用，先按授权接口读回状态，安全幂等合同明确时才考虑有界重试；缺失则报告 UNKNOWN。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| SourceSHA 不匹配 | 旧消息 STALE，不得使当前 Task PASS。 |
| SourceSHA 匹配但只有 ACK | 仍需真实 Artifact 和原有验证。 |
| 远端副作用未知且无状态读回 | 停止危险重试，并要求原 Owner 处理。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** `Attempt=A17` 基于 `a111` 作出修改，当前 Head 为 `b222`；`A17 ACK` 后返回了旧补丁。系统保留旧 Artifact 以供分析，不将其合并或标当前 Review ACCEPT。

**失败路线/反模式：** 收到 `SUCCESS` 字符串便忽略 source SHA，将旧补丁覆盖当前 `b222`，同时重发一次可能已提交的外部变更。

**最低可证伪 Oracle：** 模拟旧 Attempt 后到的消息时，当前 Head 和 Task 状态必须保持不变，返回 `STALE_SOURCE` 解释；若自动更新则失败。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K06 负责真实故障归因，K10 负责重试/恢复语义，K08 负责确切 CI/Artifact 证据。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [OpenHands SDK events/workspaces](https://github.com/OpenHands/software-agent-sdk)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [GitLab Engineering Workflow](https://handbook.gitlab.com/handbook/engineering/workflow/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [AWS Idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
