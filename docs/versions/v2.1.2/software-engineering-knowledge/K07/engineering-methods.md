# K07 · Engineering Methods / 工程方法与决策程序

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #528](https://github.com/kaicreator-mm/tally/issues/528) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 既有核心方法

## 3. 知识正文：分析方法与工程权衡

### 3.1 应不应该采用多个 Agent

从真实交付结果拆分出相互独立的分析/实现部分，再评估额外 Agent 的协同成本。单一 Task 内只读分析可多 Agent 并行；共享核心模块写入通常由一个 Writer 负责。多模型提供交叉意见不必等同多份代码分支；将多 Agent 使用建立在需求、质量或实际时间收益上，而非 Token 剩余量。

### 3.2 角色与独立性需要证据

Planner 划范围、Builder 实施、Reviewer 查设计/代码/测试、Verifier 在最新候选上核验、Repairer 在合法写集修复。身份需要区分账号、会话、Attempt、工作区和实际是否写过目标源码。审核独立性是**来源绑定的过程事实**；同账号不同对话可以独立，但不能仅凭账号相同/不同决定。Reviewer 不应冒充 Product/Release Owner。

### 3.3 交接合同要能重建工作现场

一次有用交接包含 Repo/Task、精确 source HEAD、已接受上游 Product/L2/Task Pack、允许/禁止范围、实际依赖、运行环境/工具、需验证的行为、消息相关 ID 和失败停止条件。没有这些，Agent 只能靠猜测重复调查，且旧源码的“PASS”容易漂移。信息优先引用准确 Git 源/证据，避免把整个日志/知识库塞给 Worker。

### 3.4 交互端口和传输适配器

AgentInteractionPort 概念可支持 send/request、receive/observe、ack/reconcile、ArtifactRef；GitHub event 和 Git PR 是默认耐久绑定，同主机 IPC 可快速协调子 Agent，HTTP/RPC 可用于远程服务，queue 可用于异步通知，A2A 可作为有实际 Provider 时的协议适配。定义抽象不等于一次安装全部协议；MCP 工具访问与 Agent-to-Agent 交互职责不同。

### 3.5 Task 事实源与传输成功分离

GitHub native Issue Dependencies、Claim/FSW、Review 与 Release 的既有权限不会因 IPC/HTTP/A2A 消息到达而改变。Host 执行、Artifact 写入、消息 ACK、构建测试、正式审查是不同事实。只有选定的合法权限路径才可以申请工作状态变更；不能新增 Agent 私有状态库覆盖 GitHub 中已存在的 canonical Task。

### 3.6 安全并行与冲突处理

写集不重叠是必要条件之一，不足以保证合同稳定：两个 Agent 即使改不同模块，也可能假设互斥的 Schema/状态语义。先冻结真实必要接口，再让模块独立写作；任何共享生成物、lockfile、根 index 或迁移需单 Writer/显式串行。完成时以同一真实集成基础合并，验证当前 HEAD 的完整功能。

### 3.7 失败、迟到与重复消息

消息可能重复发送、超时未确认、Worker 实际完成但 ACK 丢失、迟到响应已过期。用 Task/Attempt/correlation/idempotency 与准确 Source Ref 对账；不知道执行是否发生时保留 UNKNOWN 并读回事实，禁止无条件重做不可重复副作用。离线子 Agent 的报告不能在未接纳的情况下写入正式 Release 状态。

## 方法论进一步细化

多 Agent 先比较是否真正有相互独立的工作成果。Planner 可以并行让多个只读 Agent 分析，Builder 需要明确写集和公共接口的唯一 Writer。消息统一携带任务标识、逻辑会话/Attempt、来源 SHA、允许角色、相关 ID、Artifact 引用与失败/截止条件；任何 ACK 仅表示通信完成，不能改变正式 GitHub Claim/Review/Release。独立 Reviewer 必须真实读取被审 Diff 及测试结果，不能以同一 Builder 的自报替代。

## 操作决策表

| 任务/环境条件 | 选择或调查路径 | 应交付证据 | 需要排除的错误 |
| --- | --- | --- | --- |
| 只有一个局部 Task 和一个共享代码区域 | 单 Builder 处理，独立审查适用时再单独执行 | 最小角色计划 | 不要因有多个 Provider 就并发写入 |
| 多个独立只读研究问题 | 可并行分析，经 Owner 汇总证据 | 分工与同源事实 | 阅读建议不自动写 Task 状态 |
| IPC/HTTP 已返回 ACK | 只记录消息交付并读回正式 Task/Host | Ack+Source/Attempt 对账 | ACK != PASS |
| 候选 Head 已变化 | 迟到 Worker 结果标 STALE 并重核来源 | 阻止旧 Artifact 覆盖新结果 | 不能按旧 SHA 发布 Review |

## 原始 Decision Procedures：输入、步骤、输出、停止

## 4. Decision Procedures（Agent 可执行的决策方法）

### DP-K07-01：协作形态和 Agent 数量选择

**输入：** 目标任务、独立交付/读写范围、能力/环境约束、集成成本

**步骤：**
1. 划分真实可独立验收的结果，不按 Agent 名称拆任务。
2. 将只读、合同、核心写入、集成/审核角色分开，识别共享写集。
3. 比较单 Agent、并行子 Agent、远程专用 Agent 的真实协同成本。
4. 只对有确切可行性和收益的部分派多 Agent。
5. 定义候选合法执行环境和等待/停止条件。

**输出：** 最小角色/并行计划、写集和任务交接

**停机与边界：** 没有合法 Task/权限或无法验证效果时停在规划而非派发。

### DP-K07-02：Agent 交互选择与合同

**输入：** 消息用途、双方位置、Git/source 状态和权限

**步骤：**
1. 决定消息是需求澄清、只读分析、实现工作、结果通知还是权威请求。
2. 选择 GitHub/PR 持久交接、同机 IPC 或真实提供的其他适配器。
3. 传送 Task/Attempt/correlation、身份、精确 source/artifact ref、期待结果和期限。
4. ACK 后读取合法事实权威并对账；只在真实允许时进行状态改变。

**输出：** 通道选择、消息语义、对账和幂等约定

**停机与边界：** Agent 的 HTTP 成功响应不能直接授予 Task PASS。

### DP-K07-03：独立审核与并发集成

**输入：** 候选 Git SHA、真实 Diff、Builder/Reviewer 会话与目标接口

**步骤：**
1. 检查实际写者、审查者/会话的来源绑定，避免原 Writer 自称独审。
2. 独立读取精确候选与用户/设计/测试要求；记录可执行 P0/P1/P2。
3. 若需修复，由合法当前 Writer 处理并生成新 SHA；对新 SHA 做必要增量审核。
4. 集成前检查共享代码和接口冲突，整合后验证真实选定路径。

**输出：** 独立审查候选/真实集成收据/遗留阻塞

**停机与边界：** 缺少独立 Reviewer 的证据时不得自造 ACCEPT。

这些步骤为可引用的决策方法，不是第二个调度器、权限授权或 Review/Release PASS。

## 方法适用性限制

这些 Decision Procedures 是可解释的候选工程知识，不自动创建新增 Runtime Workflow；遇到无权 Product/L2/Task/Host/Release 决策时应输出上游 Owner、缺失原始证据与最小下一步。已有真实证据可复用，不因模型上下文长或文档多而重复重审。