# K10 · Domain Model / 领域概念及事实边界

> **Tally v2.1.2 软件工程知识库候选** · [版本 #521](https://github.com/kaicreator-mm/tally/issues/521) · [模块 Issue #531](https://github.com/kaicreator-mm/tally/issues/531) · [总索引](../README.md) · ADS 正式基线 `933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。当前只完成文档化的知识设计，尚无正式 Source→Claim 资格或真实 Package 执行。

## 2. 术语表、关系与边界误区

| 概念 | 可使用的工程定义 | 必须避免的推断 |
| --- | --- | --- |
| **Actionable Error** | 用户或操作人员能据此判断发生什么、影响是什么、怎么处理的错误 | 只返回未知异常堆栈或一串无解释 exit code |
| **Structured Log** | 适量、结构化、与来源/操作相关的运行记录 | 多打印调试输出即表示可观测 |
| **Correlation ID** | 将一次业务操作在多个模块/调用之间关联的标识 | 把用户秘密或敏感请求全文当相关 ID |
| **Transient vs Permanent** | 可因条件变化而恢复的临时问题，与确定性业务/代码错误 | 将所有异常一律无限重试 |
| **Retry/Idempotency** | 有条件重复请求时保证目标副作用语义不会错乱的机制 | 重复发送就认为 exactly-once 已经实现 |
| **Timeout/Cancellation** | 等待与取消的边界，包括超时之后业务结果仍未知的可能 | 超时=操作没有发生或一定失败 |
| **Reliability Scenario** | 故障模式、恢复方式、用户可见结果与保证条件 | 平均情况成功便宣称所有故障可恢复 |
| **SLO/Performance Baseline** | 明确真实负载、目标和测量环境的延迟/吞吐/资源评估 | 没有测量就说优化了 3 倍 |
| **Startup Configuration** | 首次启动/初始化与配置生效/失败的用户体验和后台例外 | 所有 CLI/容器都要同桌面 App 的交互设置 |
| **Incident Evidence** | 故障、缓解、恢复、验证和后续修复的可关联历史 | 恢复后删除/重写之前失败证据 |

这些概念可以作为 Domain Concepts/Knowledge 与 Tally Workflow 的语义输入，但**不是**新增通用数据中心或自建 Task/Claim/Release 权限状态机的理由。

## 必须区分的语义关系

### 关系 1

ErrorCategory/ActionableMessage/CorrelationID 与后台 Trace/IncidentEvidence 分层，不将原始敏感上下文输出给用户。

### 关系 2

Committed/Failed/Unknown 是业务副作用的不同观察状态；网络 TIMEOUT 不自动表示未提交。

### 关系 3

SLO/NFR 需要用户场景与权威来源，Profile/Benchmark 必须冻结负载、硬件、缓存、版本等可比条件。

## 来源事实、方法建议与业务资格的区别

| 来源性质 | Agent 可以采用的方式 | 禁止推断 |
| --- | --- | --- |
| 现行 ADS v4.19 标准 | 在其适用范围内遵守发布的正式规则 | 本 KB 所有补充解释都拥有同等 MUST 权威 |
| Tally v2.1.2 设计方法 | 根据真实 Task 选择并与来源/案例验证 | 直接创建 Native Task/Claim/Review/Release |
| ADS #1100 / #1383 / Draft #1384 | 明确历史/候选身份，提供方法探索 | 当作已经发布的统一 DAG-P/A/I 强制流程 |
| 合成教学案例 | 说明在这些输入假设下应产生什么可观察结果 | 声称已有真实 Host/CI PASS |
| UNKNOWN | 明确缺失证据与下一项最小检查 | 通过猜测强行升格为已采纳 Rule |

## 领域实践中的三项核查

- 用户和维护者能否判断失败发生了什么、下一步怎么做？
- 超时与重试是否可能产生重复业务副作用？
- 有无同条件性能基线支持实际优化结论？

本项目暂缓独立建设代码库逆向分析、软件安全和供应链专项；原有适用的受控权限、敏感数据与必要验证仍须执行。