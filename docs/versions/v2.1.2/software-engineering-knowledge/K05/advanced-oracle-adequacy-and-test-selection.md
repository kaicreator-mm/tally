# K05 深层专业知识：独立 Oracle、充分性和反例驱动测试选择

> [K05 系统课程](systematic-syllabus.md) · [第一卷：独立 Oracle](foundations-deep-dive-01.md) · [第二卷：组合/变形/状态验证](foundations-deep-dive-02.md) · [Golden 当前性](focus-f4-2-golden-snapshot-evidence-freshness.md)
>
> **R5 ORIGINAL_PROFESSIONAL_DOCUMENT_CANDIDATE / TESTS_NOT_EXECUTED / SOURCE_CLAIM_NOT_QUALIFIED。** 不增加统一 CI Gate；讨论的是如何在具体风险、知识来源和可观察错误之间选择必要测试，尤其防止 Agent 把“自写测试全绿”当作独立验证。

## 一、Oracle 问题：预期结果从何而来？

测试用例具有 `Input`、`Execution` 和 `Expected/AcceptableOutcome`，其中 Expected 的独立性通常比断言数量更重要。来源可以是已批准的业务规则、标准定义、经审查的历史兼容合同、独立参考实现、数学不变量，或明确采用的变形关系；每一种都有有效范围和盲点。

**独立性不是必须不同语言/不同程序**，而是不能让被测实现自行决定 Expected。若 Agent 先写错误函数 `isAllowed()`，再把该函数的输出存为 Golden 并全部断言相等，测试量可增长但没有形成对错误的真实约束。

区分 `Verification`（相对合同/设计是否正确实现）与 `Validation`（实际用户需要是否得到满足）。业务来源尚未获权时，`Expected` 应带 `ASSUMED/PROPOSED` 而不是“原始 Oracle 已通过”。

## 二、覆盖是覆盖了什么？以权限状态机推理

假设 Actor∈{admin,user}、State∈{OPEN,CLOSED}、Action∈{READ,CANCEL}，一个完整静态组合有 2³=8 种。业务规则：“普通用户不得在 CLOSED 下 CANCEL；管理员可在授权条件下例外。”如果只测以下四例：

| Actor | State | Action | 期待 |
| --- | --- | --- | --- |
| admin | OPEN | READ | allow |
| admin | CLOSED | CANCEL | allow（若正式合同允许） |
| user | OPEN | CANCEL | allow |
| user | CLOSED | READ | allow |

每一对参数取值都可以出现（满足覆盖范围内某些 pairwise 需求），却**未测试** `user,CLOSED,CANCEL` 这一关键三元拒绝。即使生成工具声称 pairwise 达成，也不能覆盖已知三维业务合取条件；最小材料修正是新增准确的拒绝 Oracle，而不是追求一个总覆盖率。

覆盖准则不同：
- 行/分支覆盖说明被访问的代码结构，不说明业务决策正确；
- 条件/决策覆盖（含需要时的 MC/DC 思路）帮助定位布尔条件独立影响，但不替代输入前提和权威预期；
- t-way 组合覆盖强调参数交互强度；
- 状态迁移覆盖需要验证路径、前置状态、重复/反常事件；
- 变异测试问测试是否能识别**特定人为注入的错误**，不是证明所有潜在错误不存在；
- 独立 E2E 只能证明被指定真实用户路径在该 Host 环境中的可观察结果。

**覆盖准则应按风险选用，不必每次一股脑全部套在所有源码上。**

## 三、用小型错误分类选择最有区分力的反例

假设付款 Job 实现存在四个候选缺陷：
- D1：只对 `Status=CLAIMED` 检查版本，忽略权限；
- D2：远端 Effect 已成功，本地超时后无条件重放；
- D3：多租户共享 cache key 引发交叉可见；
- D4：CI glob 忽略新增 `*.spec.mjs`。

| 证伪测试 | 可检出的失败模式 | 不能说明什么 |
| --- | --- | --- |
| 非授权 Actor + 合法状态 | D1 | 无法保证远端幂等 |
| Remote Commit 后丢 ACK，再投递相同 key | D2，需真实远端语义或准确模拟合同 | Mock 不等于真实 Provider |
| 两个租户相同 taskId 的交错 | D3 | 不代表所有缓存层都隔离 |
| Runner 列举实际被发现 suite 与 SourceSHA | D4 | 测试被发现不意味着结果正确 |

“独立反例”应与被测试的实现错误假说对应。对开发者未知的错误，也可使用状态探索、随机生成与变形关系寻找新反例，但必须保留生成器限制、随机种子、执行环境和实际发现数量，不能从没有找到错误推出不可能存在错误。

## 四、怎样判断某个测试的边际信息价值

设当前有两个竞争假设：H1 业务权限错误，H2 外部请求重复执行。单纯增加十个正常页面读取用例，可能对两者都没有区分力。一个 `user,CLOSED,CANCEL` 的负例可能立刻证伪 H1；一个“Remote Commit + ACK Loss”错误注入更适合检验 H2。**测试价值来自区分假设和实际风险，不是样例数本身。**

风险优先级可以参考失效后果、发生条件、历史错误、变更覆盖与可观察性，但没有正式统计时这些只是定性依据，不应把主观分数写成已计算的事故概率。

对于材料性测试已在当前 Exact SHA/环境等价运行成功的工作，不需要机械要求另一个毫无新增风险的同环境 CI 重跑；但若有 GitHub Required Check 或目标 OS/Provider 不可替代，仍须遵循真实要求。测试策略不能偷换为“尽量少测”。

## 五、测试真实性、当前性与发布真相

同一事实链应拆开记录：
`Requirement/Oracle -> Test source/fixture -> Discovery -> Execution -> Assertion result -> Environment/SourceSHA -> Affected scope`。在这个链上，任何一步都可能缺失：
- Test 被设计但未编写：**PROPOSED**；
- Test 文件存在却 glob 未发现：**NOT_RUN**；
- Test 执行了但只用 Mock：**MOCK_SCOPE_PASS**（不表明真实 Host）；
- 当前 SHA 的受影响实际测试通过：可以报告指定范围 **PASS**；
- 旧 SHA 的审核仍然可读，但是否满足新提交的审查/保护分支条件需按真实配置。

Snapshot 有更新不等于业务授权；Benchmark 耗时降低不等于正确性未退化；Source 目录的 61 条链接不等于 61 个正式可信 Oracle。用这些区分保持文档的专业知识与发布资格各司其职。

## 六、案例练习与错误答案

给学生两份实现（A 快但可能重复结算；B 慢但支持幂等 ID）、一份只记录普通成功的测试、一个已获准的业务“不重复付款”规则，以及一个只读 Mock。要求选**必要且足以区分当前假说的最少反例**、明确 Mock 的剩余风险、配置真实 Host 验收所需前提，最后把缺运行的材料标 NOT_RUN。

错误答案包括：“Pairwise 100% 所以合同充分”、“Snapshot 都绿所以新业务规则无冲突”、“Mock E2E 成功所以安装包能用”、“Agent 自写测试通过即 Reviewer 可省略独立必要审查”。

**停止条件**：业务 Oracle 未获权、真实必要 Provider/测试没运行或 SourceSHA 不匹配时，仅停止对应 PASS 声称，不自动阻塞无关材料的验证和工程推进。

**来源线索**：[A08 软件测试标准目录](../external-source-catalog.md)、[A14 ISTQB](../external-source-catalog.md)、[A50 TLA+](../external-source-catalog.md)、[A57 Node v22 test runner](../external-source-catalog.md)、[A58 Jest Snapshot](../external-source-catalog.md)。在这里它们是进一步核验的来源线索，而非直接引用官方受版权保护的完整条款。
