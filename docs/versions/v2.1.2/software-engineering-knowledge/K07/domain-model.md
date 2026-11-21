# K07 · Domain Model / 领域概念及事实边界

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #528](https://github.com/kaicreator-mm/tally/issues/528) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 2. 概念、语义及常见混淆

| 概念 | 工程语义 | 容易出现的错误 |
| --- | --- | --- |
| Logical Agent Identity | 同一执行单元/Session/Attempt 的逻辑身份和角色标识 | 用 GitHub 登录名等同模型身份或审核独立性 |
| Role / Responsibility | Planner、Builder、Reviewer、Verifier、Repairer 的实际工作职责 | 同一 Writer 给自己签独立审核 |
| AgentExecutor | 真正调用 Agent 和运行其工具/环境的现有能力 | 将消息发送成功当作 Agent 已运行 |
| AgentInteractionPort | 消息/请求/观察/确认/工件引用的抽象边界 | 将它与 AgentExecutor 和权威状态存储做成同一接口 |
| Git / GitHub | Git 版本化源码/工件；GitHub 当前正式 Task/PR/Issue/依赖事实 | Git 是唯一可行的 Agent 即时消息传输 |
| Message / ACK | 传输中的指令、通知、确认与相关 ID | ACK 自动表示代码已通过验收 |
| Artifact Ref | 源代码、测试、日志和审查产物的可校验位置与版本 | 引用可变 latest 作为 Exact SHA |
| Write Set / Fencing | 并发 Worker 能合法改的文件/共享产物及写锁/租约边界 | 不重叠文件便默认没有逻辑合同冲突 |
| Review Independence | 来源/时间/写者与审查证据可区别的技术审核过程 | 同账号不同会话就自动认定所有条件都满足 |
| Late/Retry Message | 重复、迟到、丢失或乱序传输需要对账/幂等判断 | 队列至少一次投递被当作 exactly-once 的工程事实 |

此处是领域知识概念，而非要求另外建立一套消息数据库、Task 状态机或通用 Schema；必要时可映射到现有 Tally Engineering Business/Domain Data 结构。

## 必须区分的语义关系

### 关系 1

Task/Role/Session/Attempt/Correlation/ArtifactRef 各自独立；GitHub 账户不等于逻辑身份，IPC 通道不等于执行者。

### 关系 2

AgentInteractionPort（发送/观察/确认/对账）与 AgentExecutor（执行工具）分离；GitHub Native Task/Claim 事实源不随消息通道改变。

### 关系 3

并行可用性既要求写集隔离，也要求接口、生成物、版本当前性和合并验证；ACK/Timeout 必须按 Attempt/Exact SHA 对账。

## 来源事实、方法建议与业务资格的区别

| 来源性质 | Agent 可以采用的方式 | 禁止推断 |
| --- | --- | --- |
| 现行 ADS v4.19 标准 | 在其适用范围内遵守发布的正式规则 | 本 KB 所有补充解释都拥有同等 MUST 权威 |
| Tally v2.1.2 设计方法 | 根据真实 Task 选择并与来源/案例验证 | 直接创建 Native Task/Claim/Review/Release |
| ADS #1100 / #1383 / Draft #1384 | 明确历史/候选身份，提供方法探索 | 当作已经发布的统一 DAG-P/A/I 强制流程 |
| 合成教学案例 | 说明在这些输入假设下应产生什么可观察结果 | 声称已有真实 Host/CI PASS |
| UNKNOWN | 明确缺失证据与下一项最小检查 | 通过猜测强行升格为已采纳 Rule |

## 领域实践中的三项核查

- 单 Agent 完成是否已经比并行更划算？
- 消息与持久证据的身份和 ACK 如何对应？
- Writer/Reviewer 是否真的在独立工作上下文中处理同一 HEAD？

本项目暂缓独立建设代码库逆向分析、软件安全和供应链专项；原有适用的受控权限、敏感数据与必要验证仍须执行。