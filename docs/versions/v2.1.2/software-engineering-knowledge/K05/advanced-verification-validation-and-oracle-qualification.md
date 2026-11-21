# K05 专业基础深化：V&V、测试边界、Oracle 资格与真实消费者

> [K05 课程](systematic-syllabus.md) · [R5 独立 Oracle 与测试充分性](advanced-oracle-adequacy-and-test-selection.md) · [R4 Build/平台 Focus](focus-f3-3-build-package-platform-verification.md) · [R4 Golden Focus](focus-f4-2-golden-snapshot-evidence-freshness.md)
>
> **R7 PROFESSIONAL_VERIFICATION_THEORY_CANDIDATE / TESTS_NOT_RUN / NO_RELEASE_AUTHORITY。** 对应 K05.F1.3、F2.1、F3.1、F3.2、F4.3、F5.3 六个原课程概念。以正反 Oracle、消费者契约和真实环境的证据区分为重点，不能把“形式验证”“安全审查”“测试执行”混成一种证书。

## 一、Verification 与 Validation 分别回答两个不同问题

- **Verification**：按某个已选合同或规格，构建的对象是否满足它；例如二分查找在已排序输入条件下是否输出正确索引。
- **Validation**：合同和实际软件是否满足合法的 Stakeholder/用户需要；例如用户想在付款超时后**确认支付状态**，实现者却正确实现了“重复发起付款”。

即使形式证明写成 `implementation satisfies spec`，若 spec 本身来源错误、条件缺失或假定了远端 API 无重放，也不能推导业务需求合法满足。程序构造、规格可信性、测试证据与有权 Release 是相关但不同的评价对象。

NASA SWE-052 [A65](../external-source-catalog.md) 提供一个受限的高保证要求追溯参考；IEEE 1012 与 [ISO 29119 A08/A09](../external-source-catalog.md) 帮助理解 V&V、测试设计/文档的专业分工。不得把某标准的 NASA/航空场景强制条件搬到所有 Tally PR。

## 二、用同一个错误展示各层测试能证明什么

教学需求（未实际批准）：用户在 `PAID` 或 `PACKING` 状态可以申请取消；发货一旦提交 `SHIPPED`，普通用户不可再取消。

| 验证层 | 构造输入 | 能够限定证明的内容 | 不能由此证明的内容 |
| --- | --- | --- | --- |
| 静态/类型检查 | 错误状态枚举、缺方法 | 类型/构建合同的一部分 | 业务资格和远端付款真相 |
| Unit | 订单取消 Decision Function 的正反 State | 代码在被选 Test Oracle 条件下作出预期决定 | DB 原子提交/跨进程交错 |
| 组件/Contract | API 请求/响应、Consumer 版本 | 已选 Provider/Consumer 的接口形态与错误语义 | 全系统部署和未选平台 |
| Integration | 实际 DB 同时发货/取消、版本竞争 | 选定 DB/事务组合的真实提交行为 | 真实外部支付 API 资金 Effect |
| E2E / Host | 用户操作/应用首次配置/选定平台 | 该用户 Journey 在该具体 Host/输入下的行为 | 所有平台、所有业务政策、所有未来代码 |

一个 Mock 可能验证“遇到外部错误时本地能保持 UNKNOWN”，这是有价值的局部合同，但不能因此声称已与真实付款 Provider 签署 exactly-once 保证。

## 三、等价类/边界值并不等于输入各类已被独立授权

考虑退款权限 `isAllowed(actor, status, amount, policyVersion)`。等价类可能按 actor 权限、status 有效区间、amount 上下限、policyVersion 有效期拆分。**关键边界**不只是 `amount = 0, 1, max`，还要含目标货币精度/NaN/溢出、不可信 JSON、历史已支付但外部退款未知的状态。

一个高价值负例是“UI 显示可以退款，但 API 的 Actor 并无权执行”：前端快照与按钮禁用测试不能代替服务端权限。另一个是“新正式政策把 `PACKING` 移出可取消状态，但测试 Expected 仍来自旧正式版”，此时旧版 GREEN 是历史事实而非当前验收。

[NIST 组合测试 A67](../external-source-catalog.md) 给出 t-way 的研究方法。参数组合 2-way 覆盖所有二元值对，仍不能保证三维谓词（例如普通用户 + 已关闭订单 + 取消请求）被实际测试；既有高阶业务不变量必须有专门的负例。也不应只为提高数字指标制造同质化测试。

## 四、Flaky 测试、测试未发现与过期 Oracle 是三种不同的因果类型

**A. 同一个测试实际运行且结果振荡（真正的执行不稳定）**：前提是确认同一 `testId` 在两次执行中**都被发现且实际运行**，相同 `SourceSHA`、Runner/测试版本，但出现一次 `FAIL`、一次 `PASS`。竞争假设包括共享数据库未隔离、事件时序/随机性、进程或系统资源竞争、外部依赖延迟/限流、未被固定的时钟或输入。这说明还缺少受控变量，不等于“必然只是测试代码坏”或“系统一定无缺陷”。

**B. 测试并未运行而流水线显示 GREEN（发现/选择的假阳性）**：例如 H1 的 `contract.spec.mjs` 被 Runner 发现并失败，H2 的执行因为 Test Glob 配错，实际**没有发现/运行这条测试**，其他测试全部 PASS，Job exit 0。此时正确表述为**测试套件总体 GREEN，但目标测试 `NOT_RUN`**；绝不能将未运行的测试写成“同一测试第二次 PASS”，即使 Commit SHA/Runner 版本相同。两次 Test Discovery 数量、选定用例 ID、实际 run events 可明确区分 A 与 B。

**C. Oracle 稳定但错误（确定性预期缺陷）**：对相同源、相同真实输入/环境和同一错误的 Snapshot/Expected，测试通常稳定地给出错误判断（始终伪 PASS 或伪 FAIL），**单独的过期 Golden 不会使同一个测试随机红绿**。只有当 Oracle 文件、输入数据、选择条件或环境在两次运行中改变时，才能与结果振荡相关；那应报告具体变化，不能把“Golden 旧了”当作无条件 Flaky 原因。独立业务 Source/Expected 核验和 Golden 变更记录可区分 C 与 A。

| 假说 | 必备可观察证据 | 最小反驳/区分探针 | 可报告结果 |
| --- | --- | --- | --- |
| A：同一实际测试执行不稳定 | 两次均有相同 testId 的 run + 相反结果 | 固定输入/时钟/种子、隔离共享资源、比较实际执行轨迹 | `EXECUTED_FAIL` / `EXECUTED_PASS` 均留证，根因未证实时 `UNRESOLVED` |
| B：漏选目标测试 | GREEN 仅来自其他测试，目标 testId 没有 run | 比较两次 Test Discovery manifest、globs、testId 与完整执行清单 | 目标测试 `NOT_RUN`，不能冒充目标 PASS |
| C：稳定 Oracle 错误 | 测试执行与输入一致但 Expected 有权依据不成立 | 对照当前有权 Rule/Consumer expected，并查 Oracle 版本差异 | `ORACLE_INVALID`（正确性未证实），不能把稳定误判解释成随机波动 |

**诊断顺序**：先确定是否真正两次都执行了**同一个测试**，再查实际输入/执行时序与隔离，最后核验 Expected 的业务来源。记录 `SourceSHA, Runner, Selection/Discovery, testId, run result, Oracle edition, seed, clock, Attempt, environment`，保留第一次红的证据，不靠 rerun-until-green 掩盖。隔离后全绿也只支持该隔离条件下的结果，仍需区分测试状态污染与系统真实共享状态缺陷；没有任何实际测试运行则全部属于教学区分而非真实已观察根因。

## 五、Evidence Currentness 与资格边界

一个测试报告的专业有效性必须至少明确：
`Source/Artifact Identity, Selection/Discovery, Execution environment, Oracle source/edition, Actual result, Affected scope, Permission/Owner`。

针对 GitHub 已通过的旧 H1 审核，H2 若只修改真正无关的文件且不改变所测输入/输出/环境，旧内容证据可在限定范围内复用；如果 H2 修改了被测 Router、Schema、Provider ABI、测试发现规则，就应重新验证受影响材料。重复进行完全等价的二次 CI 未必有专业信息增益；但真正必要的 GitHub Required Check、指定 Provider 或真实平台不能以节约成本为由忽略。

文档中的 165 个课程点不构成 165 个通过的源/消费者测试。一次文字审查 ACCEPT_SCOPED 只意味着该变更对应的专业文档已接受，不意味着软件或 L2/Release 有资格。

## 六、实际学习判断与不可证实结论

给一个案例：已有 Unit 40/40 PASS、Integration 使用 SQLite Mock、Node test Runner 忽略 `*.spec.mjs`、Windows 安装包未创建，Agent 宣称“全部测试通过，可以发布”。

正确答复应该按**实际被发现及运行的集合**拆开：Unit 某范围通过；有关联的 SQLite 模拟可说明什么；材料性合同测试 NOT_RUN；正式 Windows Package NOT_BUILT；目标 Host/Installer NOT_RUN；有权 Release 仍由正式流程决定。对真正不可替代的目标 OS/Provider 还需要真实验收，但不能借此给所有简单文档提交制造新强制多层 Gate。

**STOP**：对真实生产付款或不可逆 Effect 缺少权威 Oracle/Host 证据时暂停相应 PASS 声称或危险操作；独立于此的静态分析、课程研究和有限验证可继续。

**来源线索**：[A08 ISO 29119](../external-source-catalog.md)、[A09 IEEE 1012 V&V](../external-source-catalog.md)、[A14 ISTQB](../external-source-catalog.md)、[A57 Node v22 Test Runner](../external-source-catalog.md)、[A67 NIST 组合测试](../external-source-catalog.md)。全部是学科参考，不表示真实 CI 已执行本章例子。
