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

## 四、Flaky：非确定性现象与确定性的 Oracle 缺陷如何区分

一个测试在同 SHA、同 Runner 下第一次红、第二次绿：
- 可能因共享临时数据库状态没有隔离；
- 可能因事件时序/资源竞争触发真实漏洞；
- 可能因远端服务限流/网络重试；
- 可能因当前 Test Discovery 每次不同，失败用例某轮根本没运行；
- 也可能使用了过期 Snapshot/自我认可的 Expected。

处理应该记录 Runner 版本、实际发现集合、随机种子/数据集、时钟/Attempt、Source SHA 和相关资源。**简单 rerun 直到绿并删除首轮失败证据**无法区分这几种机制；但也不应因为一次未知的运行器抖动强制重跑所有无关真实主机测试。

**区分性实验**：同样 SourceSHA 与确定测试发现集合，隔离资源并复现数据竞争；如果隔离后全部通过，还需判断是测试污染还是系统真实共享状态冲突；若 SourceSHA 变了，不能视为同一次可比重复。

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
