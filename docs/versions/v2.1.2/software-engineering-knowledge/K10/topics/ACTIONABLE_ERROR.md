# K10.ACTIONABLE_ERROR · 分层错误消息、日志与敏感信息保护

> [K10 知识库](../README.md) · [工程方法](../engineering-methods.md) · [Agent Skills](../agent-skills.md) · [案例](../casebook.md) · [原始依据](../sources.md) · [外部资料目录](../../external-source-catalog.md)。
>
> `KNOWLEDGE_TOPIC_CANDIDATE / NOT_SOURCE_CLAIM_QUALIFIED / PROPOSED_NOT_CONNECTED`。本文件是原创工程问题知识切片，不是 Tally 已生效的机器 Schema，也不是当前 Task/Review/Host/Release 执行许可。案例仅为合成教学示例。

## 1. 工程问题和适用前提

用户操作失败时只看到 `500 Internal Error`，日志输出完整环境变量。怎样做出既有诊断价值又安全可行动的错误合同？

**开始前必须确认：**
1. 应用形态：交互 UI、CLI、HTTP API 或后台服务
2. 现有业务错误类别、用户角色与允许的可行动恢复方式
3. 实际日志/异常来源、相关 ID/数据保密限制

**学科知识与因果机制：** 错误应针对消费对象分层：用户需要知道操作是否完成、能否修复和下一步；操作人员需要安全可追踪的阶段/相关 ID；开发者可在授权环境定位技术错误。OpenTelemetry 的日志/指标/Trace 是不同可观测信号，并不意味着小型 CLI 必须引入集中式 tracing。日志重复和敏感信息泄漏都可能降低可靠性。

## 2. 可执行的有界判断过程

1. 把输入/业务拒绝、配置缺失、暂时外部故障、内部缺陷和结果未知分类；明确用户能不能重试。
2. 为每类定义稳定错误码或类型、安全可行动的解释、当前操作影响和相关 ID，技术堆栈保留在适当访问控制的日志中。
3. 审查敏感数据输出：令牌、凭据、个人信息、完整 request body 或环境变量不得因为调试就暴露。
4. 在实际入口测试配置缺失、无权访问、服务不可达及内部错误，验证使用者能采取下一步且错误不会被误吞掉。

## 3. 场景分支、替代方案和不适用范围

| 条件 | 推荐的判断及范围 |
| --- | --- |
| 交互 App 第一次未配置 | 可打开设置入口，阻止无授权业务请求。 |
| Headless CLI 参数缺失 | 清晰 stderr/exit/帮助，不要求弹出图形设置。 |
| 错误可能已提交且状态未知 | 标 UNKNOWN，建议安全状态回读而非“重试即可”。 |

## 4. 真实可证伪的教学情景

**受限教学输入及正确路线：** 桌面程序首次打开无仓库配置，显示需要设置仓库 URL/凭据的引导；关联错误码 `CONFIG_REQUIRED`，业务按钮暂禁用，内部日志不输出 token。

**失败路线/反模式：** CLI 调用失败就把 `process.env` 全部打印到公开 Issue，或者返回同一句 `Unknown Error` 没有追踪 ID、用户无从恢复。

**最低可证伪 Oracle：** 在包含 `TEST_SECRET=abc` 的合成输入下，用户消息和普通日志都不应出现 secret 值；缺少配置时应有可执行的下一步而非崩溃。

## 5. 最小交接与明确的 STOP

**Agent 输出契约**：当前已观察事实/证据身份、选择的知识与前提、备选方案取舍、正反 Oracle、未得来源或环境、最小下一步。凡 Source Claim 未获得正式资格、Host/CI 未运行、Product/Task 未授权的内容，必须与可证实事实区分。

**跨模块交接**：K03 当前代码中实现错误合同，K06 查根因，K08 证明真正首次运行的应用入口。

**停止规则**：需要越过合法写集、缺少有权业务决定、结果可能有不可逆副作用、外部状态不可确认或没有真实执行来源时，仅提出最小受控检查并保持 UNKNOWN/NOT_RUN，不通过重复调用或虚构状态推进。

## 6. 外部来源与可采纳边界

- [Google SRE Incident Management](https://sre.google/resources/practices-and-processes/incident-management-guide/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [OpenTelemetry Signals](https://opentelemetry.io/docs/concepts/signals/)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。
- [ADS Implementation Quality](https://github.com/kaicreator-mm/ai-development-standard/blob/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c/standards/IMPLEMENTATION_QUALITY_STANDARD.md)：学科概念/实践方法的参考出处；不是本案例实际执行的证据，引用应核对版本与使用许可。

**已发布 ADS 约束**：`kaicreator-mm/ai-development-standard@933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c`。遇到差异应按当前有权规范与 Product/L2/Task/Host/Release 的真实作用域处理；未合并的三阶段 DAG 候选不自动变为正式工作图。
