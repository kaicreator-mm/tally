# K07 · Casebook / 贯穿案例及反例库

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #528](https://github.com/kaicreator-mm/tally/issues/528) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 已有贯穿案例与正反推断

## 7. 贯穿示例：正常路线、故意错误路线和适用差异

**情景（教学模拟）：** Tally 使用两个 Agent：Builder 修改一个 API，Reviewer 在独立会话审核；同机第三个 Agent 通过 IPC 提供只读测试建议。

**正确推进：**
1. Planner 从合法 Task 中提取准确 Repo/HEAD、允许写集和用户验收，并确认 API 已有 L2 合同。
2. Builder 在自己受控工作区实施并通过 Git 持久化源码与测试，产出 Exact SHA 及实际受影响测试结果。
3. IPC 子 Agent 只传递测试建议和工件引用，不持有直接发布 Task 或修改共享写集权限。
4. 独立 Reviewer 从 GitHub 读取当前确切候选和真实 Diff，结合需求/测试给出局部结论。
5. 只有合法 Task/Host/CI/Review 路径确定适用结果后，才更新对应工程事实；消息 ACK 不触发此动作。

**有效的验收证据：** 实际源码候选/审查者/最终组合都绑定相应准确 HEAD；并行无共享写冲突且 IPC 消息不授予工程权威。

**错误实现/决策：** Builder 给 Reviewer 发“已通过”聊天消息后 Tally 自动把 Task/Release 标成功；两个 Builder 同时更新 lockfile 并互相覆盖。

**条件变化如何影响选择：** 如果没有适合的独立工作，单 Agent 直接完成是更好的选择；远程 Agent 可用 HTTP/A2A 只替换消息通道而保持同样的 Task 权威限制。

## 既有反模式和失效机制

## 8. 反例库与适用性边界

1. **GitHub-only 假定**：把消息语义与特定平台评论 API 绑定，阻碍本地高效协作
2. **ACK-as-PASS**：通信成功自动变成正式工程资格
3. **Identity Shortcut**：以账号相同/不同代替真实独立审核证据
4. **Shared Writer**：多个 Agent 改同一个生成产物而无租约/集成规则
5. **Late Effect**：新 Task 已换 HEAD，迟到 Worker 仍尝试执行旧版指令
6. **Token Parallelism**：因资源闲置强行拆分本来原子的 Task

## 新增三种反事实场景

### C-K07-01 · 共享 generated manifest

必须唯一 Writer 或有源互斥/集成纪律；不同工作树也会冲突。

**Agent 工作法：** 明确与当前 Task 相关的行为/接口/版本/失败条件，引用 [工程方法矩阵](engineering-methods.md) 选择最小充分 Procedure。若此场景改变了前提、角色或执行环境，就要重新核实有效来源；不将旧正例强行外推。

**预期反例：** 删除必要的来源、引入互相矛盾的状态/权限、或仅给 Mock 结果时，合格 Agent 应指出不能作出的结论并保留 UNKNOWN/NOT_RUN。

### C-K07-02 · 同账号不同 Reviewer 对话

独立性依据具体会话、非 Writer 来源和实际审核行为，不依据账号异同。

**Agent 工作法：** 明确与当前 Task 相关的行为/接口/版本/失败条件，引用 [工程方法矩阵](engineering-methods.md) 选择最小充分 Procedure。若此场景改变了前提、角色或执行环境，就要重新核实有效来源；不将旧正例强行外推。

**预期反例：** 删除必要的来源、引入互相矛盾的状态/权限、或仅给 Mock 结果时，合格 Agent 应指出不能作出的结论并保留 UNKNOWN/NOT_RUN。

### C-K07-03 · 远程 Agent 超时、迟到回包和旧 HEAD（SYNTHETIC_TEACHING_CASE）

**明确输入**：Task #42 的有效 Candidate HEAD 原为 `aaa111`，`Attempt=A17` 的远程 Builder 启动后通信超时；与此同时当前合法 Writer 已将 Task 的 Candidate 更新为 `bbb222`。随后旧 Worker 返回 `ACK/SUCCESS, attempt=A17, source=aaa111`。

**应当执行**：
1. `ACK` 仅表示 Worker 报告已送达；先确认真实代码/测试 Artifact 是否存在并绑定 `aaa111`。
2. 读取 GitHub 当前 Task/PR HEAD `bbb222` 与合法 Attempt/Claim 状态，旧消息属于 `STALE_SOURCE`。
3. 不把旧 Artifact 覆盖新的 Candidate，不将 `A17` 的成功标记为当前 Task/Review/Release PASS。
4. 若需要复用旧结果，必须重新核对当前接口/差异与适用 Source，按现有 Owner/Reviewer 规则评估；未知外部副作用需要独立回读。

**可证伪 Oracle**：系统在此输入下应保持当前 `bbb222` 和既有 Task 状态不变，记录旧结果为 `STALE/UNKNOWN`。若收到 `ACK/SUCCESS` 就更新 Task 为 PASS，则是明确的错误。示例 HEAD/Task/Attempt 为模拟值，非真实项目状态。

## 使用教学案例的资格边界

本文所有新增 C-K07 案例均为 `SYNTHETIC_TEACHING_CASE`。真实生产资格需要来源明确的项目实例及正反/失败/当前版本证据；不能将教学案例当真实事故统计或 Agent 历史成功率。