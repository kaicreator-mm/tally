# K04.F2.3｜任务完成、证据接受与合法授权边界

> [K04 课程](systematic-syllabus.md) · [任务第一卷](foundations-deep-dive-01.md) · [165 点台账](../foundation-concept-coverage-ledger.md)
>
> **R4 ORIGINAL_KNOWLEDGE_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / NO_TASK_STATUS_CHANGED。** 深化 K04.F2.3 的任务管理学科，绝不以本章创建 GitHub 状态机、Review/Release 角色或替现行 ADS 决议授权。

## 1. 任务完成不是单一 DONE 消息

从软件工程角度至少有四种不同的命题：

1. **工作声称完成**：Worker 认为其范围内操作结束，只是提交报告。
2. **交付对象存在**：当前 Git 代码/文档/测试/制品有可读取的精确版本。
3. **验证事实成立**：对该输入、目标平台和行为 Oracle，必要测试/评审真实执行并有材料性证据。
4. **有权接受**：相应 Product/L2/Task/Review/Release Owner 按已生效的流程接受限定范围。

这些命题可以先后发生，也可以部分缺失。一个 Agent 产生 `done=true` 不推出第 2–4 项；一个 Build PASS 也不代表业务已经被授权发布。

## 2. 最小完工语义：承诺范围×事实×权威

每个 Task 至少要能回答：**要交付什么、由谁判断、基于哪个 Source HEAD、哪些已满足/未满足的必要条件、哪些结果不在范围内**。文档里的建议 Checklist 不能成为新 Native Task 权威，不引入第二套 Task 状态；真实项目按实际 ADS 版本和 GitHub Issue/PR 内容进行裁定。

| 事件 | 合理的知识层解读 | 禁止的晋升 |
| --- | --- | --- |
| Writer 报告完成 | 有一个可审的候选交付 | `Worker ACK = Release` |
| PR 存在，HEAD=H | 可以定位候选源事实 | `PR Open = Product Done` |
| CI 对 H 绿 | 材料性 CI 检查在声明的输入/环境下成功 | `CI Green = 全部消费者验证` |
| 独立 Reviewer 接受差异 D | D 在限定审查 Scope 内可用 | `只读旧版本通过 = 新 H 全部通过` |
| 有权 Owner 接受 | 对特定 Goal/Release 的授权事实 | `Agent 可以代签 Original Claim` |

**适用例外**：资料只变更、无需真实 runtime 时，不能强制重跑大型 Host E2E；需要真实 Provider/Release Required Check 时，不能用等价本地自测冒充其成功。目标是在满足已经生效的必要条件下减少重复工作。

## 3. 并行化与授权是正交维度

Task A 和 B 即使无代码写入冲突，也可能共享有权业务决议；例如 B 的验收 Oracle 依赖仍未明确的权限政策。并行不等于可以跳过这一依赖。相反，如果 A 和 B 只共同使用一份只读原理材料，并不形成阻塞式 `BLOCKED_BY`。

**教学反例**：Agent A 和 B 并行修复不同模块，A 的测试记录绑定 H1，随后 B 更新共享 Schema 到 H2。不得把 A 的全局系统测试自动提升为 H2 的当前性；也不必因 B 的改变与 A 的局部测试无关就重做所有已接受的独立知识审查。按真实影响集选择必要重验。

## 4. 对有权决议和证据时效做精确的条件化判断

GitHub 的受保护分支可以配置 Required Checks、审批人数和旧审批失效规则，但是否适用取决于**实际仓库配置**；不能声称所有仓库默认都启用相同策略。知识层面记录该性质，Task 事实以本项目当前 GitHub/A​​DS 真相为准。历史报告继续可追溯，但不会因为“完成得早”成为最新提交的无限期通行证。

**最小交付建议**：对当前 Work Unit 输出五个字段：Objective（需求）、Exact Input（来源/版本）、Evidence（实测材料）、Decision Owner（实际有效权限）、Residual Scope（未知/排除）。这是便于研究的知识文本，不是新 Wire Schema 或强制 YAML。

## 5. 可证伪任务

给定 Issue T、PR H1、Writer 完成、独立 Review 接受 H1，之后 H2 增加一个受影响的调用路径和未运行测试。要求判断：
- 哪些 H1 历史证据可复用作为事实；
- 哪些 H2 新影响范围需要新的材料验证；
- 哪些未授权操作必须 STOP；
- 为什么不能把整项任务标 PASS，也不能无限重审不受影响内容。

若无法找到最新 Issue/Owner/SourceSHA，正确结果是限定的 UNKNOWN/WAIT，而非新建第二个审查委员会。

**公开来源线索**：[GitHub protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)、[GitHub required status checks](https://docs.github.com/en/pull-requests/reference/status-checks)、[ADS 官方源仓库](https://github.com/kaicreator-mm/ai-development-standard)。这里只解释一般工程原则和当前权威分离，绝不自造 ADS 原文。
