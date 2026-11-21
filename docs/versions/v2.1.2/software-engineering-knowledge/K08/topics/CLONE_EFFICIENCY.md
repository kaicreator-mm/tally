# K08.CLONE_EFFICIENCY · 针对多 Workflow 的真实 Clone 成本与安全复用

> [K08 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

Woodpecker 单次 Pipeline 下 19 个 Workflow 均声明 `clone-exact-sha`，是否可以只 Clone 一次且不丢失必要源码？

**开始前必须确认：**
1. 成功/失败流水线真实 clone 步骤日志和墙钟耗时
2. Woodpecker/Runner 每个 Workflow 的文件系统、容器/缓存隔离
3. 源码依赖闭包（源码、脚本、测试 fixture、生成物）及 Current HEAD

**学科知识与因果机制：** 减少重复源码下载的收益由 Provider 工作区隔离、实际 Git 网络次数、仓库体积、深度/过滤器、缓存共享范围、Runner 存储与构建依赖决定。YAML 文件出现 19 个 clone 节点不能直接得出实际 Git 请求次数与节省百分比；也不能假设并行 Workflow 可共享一个工作区。安全复用必须仍绑定 exact SHA 和当前所有必要源码/测试 fixture。

## 2. 可执行的有界判断过程

1. 按真实 Workflow/step 统计各步骤实际执行的 git fetch/clone/reset，记录耗时、网络请求与缓存命中。
2. 判断 Provider 是否允许一次 fetch 产生可供其他步骤安全复用的不可变工作区或合法缓存，明确不同容器读写冲突。
3. 尝试最小优化：当前精确提交 checkout、合理浅克隆/过滤、可验证的同 SHA 工作区复用；不得删必要测试/构建文件。
4. 比较改前后同目标和可比环境的真实 clone 次数/时长、Build/Tests 结果；如果没有测量只报告设计候选。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 多个 Workflow 真正隔离 | 不声称能直接复用彼此的工作树。 |
| 同一 Workflow 多阶段共享工作区 | 可考虑一次 checkout 并复用。 |
| Sparse checkout 排除了所需 fixture | 优化无效，必须恢复正确依赖闭包。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** Node monorepo 的一个 Workflow 在 checkout 后依次 Build/Tests，共享同一目录；可以减少同 Workflow 内重复 checkout，但另 18 个隔离 Workflow 必须按 Provider 实际共享能力审查。

**失败路线/反模式：** 仅把 19 个 clone step 移到全局 YAML 却没有对应共享 volume/缓存配置，导致部分 Workflow 找不到源码或使用了旧 checkout。

**最低可证伪 Oracle：** 改动后每个工作单元必须证明 SOURCE_SHA 与当前 PR 一致、正确加载受影响 package/test fixture，且真实网络耗时有对照；任何缺件不能用“节省 80%”假数掩盖。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K04 选择构建任务依赖，K11 比较复杂度收益，K05 确认测试覆盖。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [GitLab Engineering Workflow](https://handbook.gitlab.com/handbook/engineering/workflow/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [DORA Research](https://dora.dev/research/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ADS CI Execution Standard](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/CI_EXECUTION_STANDARD.md)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
