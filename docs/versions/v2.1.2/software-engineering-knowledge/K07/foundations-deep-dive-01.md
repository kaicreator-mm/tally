# K07 · 学科深化 01：多 Agent 的消息语义、来源身份与权限边界

> [系统课程](systematic-syllabus.md) · [角色分离](topics/ROLE_SEPARATION.md) · [旧结果](topics/STALE_RESULT.md) · [协作方法](engineering-methods.md)
>
> **ORIGINAL_EXPLANATORY_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_AGENT_RUN**。深化 K07.F1.1/F1.2、K07.F2.1/F2.2/F2.3、K07.F3.1/F3.2 和 K07.F4.1/F4.2。Agent 通信协议知识不是 Tally 已实现的 SDK 或权限系统。

## 1. 消息、执行请求、任务事实是不同对象

消息通道可以是 GitHub、进程 IPC、HTTP、队列或 A2A，但传输成功只表示消息被接收/确认，不代表代码已经被合并、测试已运行或 Release 已获权。把 `ACK`、`WORKER_RESULT`、`CANONICAL_TASK_STATE`、`HOST_OBSERVATION` 分层，避免“下游答复 SUCCESS”覆盖任务事实。

每次受控交接至少标识 `TaskRef`、`Attempt/Session`、`SourceSHA`、`Claim/WriteSet`（若真实存在）、`ResultRef`、`ObservedAt` 与合法的结果阅读者。字段只是概念上的信息要求，不宣称存在已发布 Tally wire schema；真实传输格式依现有接口选择。

## 2. 迟到、重复、乱序和重放是正常分布式问题

假设 Worker A 在 `HEAD=a111` 上开始，而当前合并候选为 `HEAD=b222`。A 返回 `ACK=SUCCESS` 与旧 Build 日志：
- 此消息可以被保存为该 Attempt 的历史事实。
- 不能把旧源结果当成 b222 的测试/源码审核/Release 准备结论。
- 如果 A 试图在共享写集写入，先检查当前写权限和期望版本；拒绝旧写入或重新建立受权 Task/Claim，不允许盲目覆盖。
- 如果返回信息涉及可能已发生的远端副作用，由 K10/K06 区分 `UNKNOWN_EFFECT`，不可凭重新投递“补成功”。

去重键必须覆盖业务/任务真正的稳定身份与具体操作，不能仅凭相同 GitHub 登录名去重 Agent；独立 Reviewer 可以使用同一账户下的其他独立对话，只要真实角色隔离/非 Writer 关系成立。

## 3. 并行安全依赖写集与合同，不依赖 Agent 数

两个 Agent 分别改变 `frontend/` 和 `backend/`，若共享 API Schema 或根锁文件，就仍有逻辑合同/资源冲突。启动并行前确认：可独立验收成果、稳定合同、受权写集、合并 Owner、真实关键路径收益。K04 负责因果依赖与计划，K07 负责角色/交接/共享写集协作协议，K11 评估额外 Reviewer、合并与上下文成本。

| 交接情形 | 正确解释 | 错误晋升 |
| --- | --- | --- |
| 传输层确认收件 | 消息已送达 | Task 已 PASS |
| 旧 SHA Worker 返回绿日志 | 历史 Attempt 证据 | 当前 HEAD 验证已 PASS |
| Reviewer 给出有界意见 | 该范围审查事实 | 任意 Release 自动批准 |
| Builder 报告完成但未读回 | 待验证 Writer 声明 | 合法 Host 已执行 |
| 超时后副作用未知 | 查询可信状态/保留未知 | 直接重复不可逆调用 |

## 4. 任务使用的最小上下文和拒绝规则

Agent 不应一次装载全量 BOK；按 Goal→问题路由→必要知识→当前 SourceSHA/受权范围选择。影响当前结论的证据才需要装入；已确定的 Task 事实可引用精确版本，不重复全文复制。缺少当前任务权威、关联 HEAD、写入授权或独立 Reviewer 事实时拒绝相应晋升，允许保持候选指导。

## 5. 反事实教学练习

让一个 Worker 先后发送“准备”“完成”“重试完成”，第三条在另一 HEAD 合并后抵达。答案应能分别说明消息接收事实、代码状态和合法 Task/Review 状态；如果只靠消息时间排序就宣布第三条覆盖第二条，是错误做法。教学没有真正运行 Agent，不可写成协作机制实际 E2E PASS。

**来源线索**：[A20 Microsoft Playbook](../external-source-catalog.md)、[A23 GitLab](../external-source-catalog.md)、[A28 SWE-agent](../external-source-catalog.md)、[A29 OpenHands SDK](../external-source-catalog.md)、[A34 DORA AI 报告](../external-source-catalog.md)。