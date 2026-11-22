# K03 专业深化：异常语义、可行动日志与测试证据的不可混淆性

> [K03 系统课程](systematic-syllabus.md) · [R7 算法/类型/形式合同](advanced-algorithm-contracts-types-and-formal-claims.md) · [K05 Test Oracle](../K05/advanced-verification-validation-and-oracle-qualification.md)
>
> **R9 DOCUMENT_CANDIDATE / NO_SOURCE_CLAIM_QUALIFIED / NO_REAL_CI_OR_HOST_RUN**。精确深化 **K03.F5.1 异常分类和可行动错误**、**K03.F5.2 日志与敏感性**、**K03.F5.3 真正编译与受影响测试**，只给出可证伪的工程实现机制，不为每个小任务增加新 Gate。

## 一、错误信息的目标是让操作者采取安全行动

“请求失败”无法告诉用户重试是否安全、权限是否欠缺、请求是否到达远端或哪个真实阶段发生错误。一个可行动的错误至少分开：
- **Location**：LOCAL_INPUT / AUTHORITY / BUILD / TEST_DISCOVERY / EXECUTION / EXTERNAL_EFFECT；
- **Observation**：真正观察到的 `HTTP 409`、`ETIMEDOUT`、实际 `testId` `FAIL`、`NOT_RUN`；
- **Known postcondition**：有权依据下确定执行未开始、已成功、已明确拒绝，还是 `UNKNOWN`；
- **Safe action**：可自行纠正输入、需补权限、需要核实下游、允许受限重试或立即停止；
- **Correlation/evidence**：当前 SourceSHA、Attempt、受权的追踪标识与非敏感详情。

这些维度相互正交。例如 HTTP 504 是客户端所见的网关状态，**不能证明下游没有持久提交**。若已知业务合同规定可用稳定业务幂等键读回当前状态，应先合法查询；合同未知时，`EFFECT_UNKNOWN` 比“自动重试直到成功”更正确。

## 二、状态分类不是把所有错误变成同一个 enum

候选程序接口：
```text
ErrorKind = INPUT_INVALID | PERMISSION_DENIED | BUILD_INPUT_INVALID
          | TEST_NOT_DISCOVERED | TEST_EXECUTED_FAIL | REMOTE_REJECTED
          | REMOTE_OUTCOME_UNKNOWN | INTERNAL_UNEXPECTED

Evidence = { sourceSha, attemptId, stage, observedCode?, correlationId? }
Recovery = { action: "FIX_INPUT" | "ASK_OWNER" | "VERIFY_REMOTE" | "REVIEW_SOURCE" | "STOP",
             requiresAuthority: boolean }
```

以上是**教学概念结构**，不是 Tally/DomainHarness 的新机器 Schema。若 HTTP API 真正采用 [IETF RFC 9457 A70](../external-source-catalog.md)，可以用结构化 Problem Details 表述错误（例如 `type`、`status`、`title`、`detail`、`instance`）。但 Problem Details 的 HTTP 层含义既不指定业务重试权，也不自动查询财务状态；`detail` 面向用户可见时不应包含服务器密钥、AccessToken、租户隐私或原始 SQL。

### 区分性负例

- 服务端把缺少仓库写入权限返回 `500` 并建议“反复重试”，将实际 **ASK_AUTHORITY** 误标成暂时服务故障；
- 付款请求已被第三方提交，但网关超时，本地 `Error` 字段被赋值 `NOT_PAID`，诱发重复付款；
- 缺少测试文件时 CI 显示 Exit 0，日志仍打印“all affected tests PASS”；实际应报告目标 `NOT_RUN`。

## 三、日志是失效证据，不是敏感原始材料仓库

同一个错误需要足够多的诊断上下文，又必须保留合法的保密边界。建议最小记录：
```text
timeUtc, severity, component, sourceSha, stage, attempt,
eventCode, causalCorrelationId, sanitizedMessage, affectedScope
```
`causalCorrelationId` 只可作为调试关联键，不能等同正式权限凭据或完整可复现证据。对用户身份、邮箱、API key、支付卡号、命令行 secrets、完整远端响应做受限处理；**敏感数据是否可以落盘**应以项目实际权限/保留政策为准，不能凭“日志用于调试”获得默认授权。

最短反例：为了复现 `401`，程序把 `Authorization: Bearer ...` 写进 Woodpecker 日志并发给 Reviewer。这确实增加故障信息，但扩大了泄露面；应输出非秘密的错误类型、受限 Owner/Host 作用域、请求阶段、可复现条件与足以定位的安全 ID。

“打更多日志”也可能改变时序并掩盖 Race；如真实系统以时间排序推断先后，需要单独考虑跨进程时钟偏移、Trace/Span 因果和采样缺失，而非仅按毫秒排序报告确定根因。

## 四、Syntax/Type/Build/Discovery/Execution/Oracle 五层不能互相证明

一次真实提交前可以低成本按受影响范围递进：
1. **语法与类型检查**：能解析/类型符合目标工具链；不代表生成内容新鲜或业务正确；
2. **BUILD**：在指定输入闭包/编译器/Node 版本下真实产出目标结构；不代表目标 Consumer 可运行；
3. **Test Discovery**：确认至少真实找到了所有被需求选中的测试 ID，测试目录/扩展名/过滤器准确；
4. **Execution**：对已发现用例有对应 Attempt/Exit/Assert/结果，不能把漏选记 PASS；
5. **Oracle/Consumer**：Expected 来自有权来源/合同，在选定环境/版本受检；Mock 不等于真实 Provider/Installer。

**反证演算**：`tsc --noEmit` 成功，`node --test test/**/*.test.mjs` 成功，但新增测试是 `contract.spec.mjs` 且没有被发现。可以报告“已执行的 40 项测试 PASS”，不能报告“合同新测试已通过”。如果节点上 Node22 runner 版本不兼容某 glob，需以实际版本文档和 Test Discovery 列表核查，不强制升级到 Node24 来掩盖误配置。

## 五、最小输入变化与证据复用的工程意义

旧 H1 已通过的测试结果可在受控材料等价范围内复用；H2 改了同一业务规则的路由/Schema/测试选择而声称“没有修改测试文件所以不用重测”不成立。另一方面 H2 只修复 README 标点且没有更改被测输入闭包，也不应强制重复完全等价的真实目标平台安装。

**可证伪的最小问题**：固定 H1/H2、工具链、Test Discovery、Oracle Source，列出受影响的代码/生成器/Consumer 依赖、单次实际验证的证据范围以及何时只可写 NOT_RUN。不能仅以“CI 绿”或文件名同名取代 Source/Artifact Identity。

## 六、学科交接与实际限制

K01 负责有权业务/权限规则，K02 负责架构/Host 状态边界，K05 负责独立 Oracle，K06 负责最小故障复现和根因验证，K08 负责 Build/制品身份，K10 负责运行时过载，K11 负责信息增益/成本。本章只教**如何不丢失错误语义以及如何生成可复核证据**，不定义新产品 ErrorKind 或生产日志权限，也未执行真实 CI。

**原始专业来源线索**：[A52 IETF RFC 9110](../external-source-catalog.md)、[A57 Node v22 Runner](../external-source-catalog.md)、[A58 Jest Snapshot](../external-source-catalog.md)、[A70 RFC 9457](../external-source-catalog.md)。引用不能取代正式 Source Claim 或运行时验证。