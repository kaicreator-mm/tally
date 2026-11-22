# K07.ROLE_SEPARATION · 多 Agent 的 Writer、Reviewer、Executor 真实角色分离

> [K07 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

一个 Builder 完成代码以后，另一个 Agent 的 Review 是否真的独立？两者使用同一 GitHub 账户但来自不同 ChatGPT 对话是否可以？

**开始前必须确认：**
1. 当前 Repo/Issue/PR/Base/Exact HEAD 与候选范围
2. Reviewer 逻辑会话、是否参与 Writer 改动的真实事实
3. 对应 Task 资格/Review 对象及实际测试证据

**学科知识与因果机制：** 审查独立性来自真实工作来源和行为，而非电子邮件/账户是否不同。Writer 生成修改候选，Reviewer 在独立逻辑会话读取其 Base→Exact HEAD Diff、需求/架构合同、实际验证收据，再给出问题和有界结论。Agent Executor/工具权限与通信渠道应区分：Reviewer 可以只读，不能因拥有 GitHub connector 就默认有写入许可。Google 的旧 code review 指南已 Archive，应按成熟历史实践而非现行政策引用。

## 2. 可执行的有界判断过程

1. 识别 Writer 来源与受控写集，选真正未参与该改动的 Reviewer 会话；同一账户可因会话隔离而仍具独立性。
2. Reviewer 直接读取当前 Base/HEAD 真实 Diff、必要源码、原始需求/验收和实际 CI/Host 收据，而非只信 Writer 摘要。
3. 按范围列 P0/P1/P2、证明与缺失证据，输出 ACCEPT_SCOPED/REWORK；若新提交改变审核对象，仅复核受影响范围。
4. 审核传达只作为合法 GitHub Review/Issue 评论和后续 Owner 决策输入，不能自造 Release/Task Claim。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 只有一条 Writer 会话但模型声称切换成 Reviewer | 不能算独立复核。 |
| 相同 GitHub 账户、不同实际对话且 Reviewer 非 Writer | 身份本身不阻碍独立性，仍需核真实审查。 |
| 已审 SHA 与现行 PR HEAD 不同 | 旧结论不能自动覆盖新版本。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** Builder 在 PR HEAD `a111` 修改三个文件；另一对话读取 exact Diff、验证命令和 Source，指出一个负例未运行。Builder 修复为 `b222` 后，旧 Review 保留历史有效范围，新提交的增量必须独立确认。

**失败路线/反模式：** Builder 自己生成名为 `INDEPENDENT_REVIEW_PASS` 的报告却不实际读取 Diff，然后把 Task 自动标为独立审核成功。

**最低可证伪 Oracle：** 在测试中将 Reviewer 输入限制为 Builder 的单段总结、不提供真实 PR Diff，则系统必须表明审核证据不足；同 SHA、不同对话也必须有实际源码检查才能构成真实 Reviewer 结论。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K07 处理角色和交接；K05 负责测试充分性；K08 负责 CI/Host 来源；正式 Review/Task 状态仍走 GitHub 原有权限。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [Google Engineering Practices (archived)](https://github.com/google/eng-practices)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [GitLab Engineering Workflow](https://handbook.gitlab.com/handbook/engineering/workflow/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [OpenHands SDK as implementation reference](https://github.com/OpenHands/software-agent-sdk)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
