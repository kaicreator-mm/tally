# R10 软件工程领域知识选择验证集：六组最小变形对照、十二道可判别题

> [已有 Q01–Q18](domain-intelligence-use-cases.md) · [完整 W01–W22](professional-reasoning-worked-traces-04.md) · [五项原始来源候选](professional-primary-source-claim-dossiers-r10.md) · [评价/评分约定](professional-knowledge-evaluation-protocol-r10.md)
>
> **R10 AUTHOR_FIXTURES_ONLY / 12_SYNTHETIC_CASES / REAL_AGENT_RUNS=0 / QUALIFIED_PRODUCTION_ORACLES=0。** 本集不是从真实软件缺陷或商业数据采样，也不是已执行的 Agent Benchmark。每组**只改变一个对判断有决定意义的事实**，要求未来被测的知识选择机制能随之改变结论；预期输出和反例由 Writer 定义，独立 Reviewer 仅可审核其逻辑/适用性，**不能因此宣布 Agent 通过**。

## 一、定义：单个 Case 的正确结果不等于长篇学科复述

对于每个案例，候选答复至少给出：
- **输入边界**：当事主体/版本/选定动作/观察到的真实材料与缺失材料；
- **最小 Kxx 领域路由**：哪些学科概念是判断必需，哪些材料明确无关；
- **支持/反对依据**：真实用户事实优先、官方规范仅在其范围内支持技术解释；
- **预期决议**：已知条件下何时可得限定结论、何时 ASK/UNKNOWN/NOT_RUN/STOP；
- **独立 Oracle**：什么具体新证据会翻转当前结论，以及目前是否实际可取得；
- **明确禁止的晋升**：文档中没有 Source Claim/Host/CI/Release/PACKAGE 的实际权限。

Case 标识是本篇教学 ID，**不是新 Native Issue、DomainHarness Schema、Agent Input ABI**。所有 SHA、Artifact Digest、Tenant、Runner/Owner 皆为虚构代号，不能指向真实用户资产。

## 二、六个独立维度：对照一览

| 变形组 | A 与 B 唯一变化的关键维度 | 必选知识 Owner | 若未识别差异将犯的错误 |
| --- | --- | --- | --- |
| V01 | 目标测试有没有被真正发现和执行 | K03 K05 K08 | 用 Job GREEN 给未运行的用例签 PASS |
| V02 | 相同规范文本是否由有权 Owner 正式批准且在适用期间生效 | K01 K04 | 将 Draft/旧 FAQ 与正式 Rule 同权 |
| V03 | 远端 Effect 是否存在有权、匹配业务键的已提交收据 | K02 K06 K10 | 把 Timeout/本地 Commit 当成财务真值 |
| V04 | 正式下载制品 Digest 是否与已验验证据指向同一 Artifact | K08 K05 | 旧制品/本机重打包冒充发布验证 |
| V05 | 多项目配置中的凭据是否确按受权 Owner 分离 | K09 K01 K10 | 启动成功即称真实隔离安全 |
| V06 | 性能两轮样本是否来自同负载、同 Journey/可比环境 | K10 K11 | 采样 CPU/平均值硬推用户 P95 因果改善 |

## V01 — 实际执行与未执行（K03/K05/K08）

**共同输入**：受影响合约目标 `contract.spec.mjs`，受检 H2，Node v22.16.0 测试运行器，CI Job exit 0，其余 40 条旧测试 PASS；目标业务 Expected 的权威性尚未获得新证据。

- **EV10-01A**：完整 Discovery Manifest **不含** `contract.spec.mjs` 的 `testId`，执行轨迹也无其 run event。
  - **必要输出**：该目标 `NOT_RUN`，只可描述其余 40 条已运行测试 PASS。核查 Runner globs/过滤器，必要时安排受影响测试真实运行；不能报告新合约 PASS。
  - **可反驳探针**：当前 H2/Runner 的目标 testId discovery/run/actual assert receipt。
- **EV10-01B**：完整 Discovery Manifest **包含**目标 testId，且 execution event 和对应 Assert/Exit 均为 PASS。
  - **必要输出**：可报告**该测试在此 H2/Runner/Attempt 下已运行 PASS**；但由于题目未授予独立合法 Expected 来源，不能自动声称当前业务政策已被验证或整体 E2E/Release 通过。
  - **可反驳探针**：当前 Expected 与真正业务 Rule/Consumer 的来源和失效反例。
- **变形 Oracle**：相同 Job GREEN，A 的目标结果必须是 `NOT_RUN`，B 的目标执行事实可为 `EXECUTED_PASS_SCOPED`。无权 Oracle 这一共同未知应在两个结果中保留。

## V02 — 草案与正式生效规范（K01/K04）

**共同输入**：Tenant B 于假想的 11 月 5 日请求在 `PACKING` 取消订单；规则文本为“Tenant B 从 11 月 1 日起 PACKING 不可自行取消”，旧客服 FAQ 允许；Actor 和适用租户已明确，未查真实数据库/财务状态。

- **EV10-02A**：该规则只是**尚未签发的 Draft**，缺当前有权 Owner 生效凭据。
  - **必要输出**：不能将 Draft 当现行正式政策；回到上一个**已确认在 Tenant B 当前仍生效**的规则，如它仍有效，可限定应用；若无法确认旧规则是否仍有效，应 `ASK_AUTHORIZED_OWNER`。不得因为草案比 FAQ 新就默认禁止所有租户。
- **EV10-02B**：提供了**有权 Owner 的签发/生效凭据**，文本、Tenant B 及日期均匹配，且不存在材料中更高优先级的覆盖。
  - **必要输出**：可在这套合成材料范围内推导 Tenant B 的 PACKING 用户请求被拒；不得推广到 Tenant A 或将业务拒绝等同外部退款已执行。
- **变形 Oracle**：只改变批准状态，候选不能对 A/B 皆回答“按最新文本拒绝”。A 可 ASK，B 可限域拒绝。两者都无真实政策资格。

## V03 — 丢失 ACK 时的远端可观察性（K02/K06/K10）

**共同输入**：支付 Intent 在本地 PG18 已提交，调用远端时网关发生 Timeout，Provider 实际是否接受不从网关观察获得；没有给出自动重试的权限和业务幂等合同。

- **EV10-03A**：当前没有合法的远端业务键查询回执，只有本地 `PENDING`。
  - **必要输出**：`OUTCOME_UNKNOWN`；无受权/幂等协议不得再次提交付款；请求有权的只读状态确认或人工处理。
- **EV10-03B**：附带**有权 Provider 针对同一业务键、同一 Tenant/Attempt 的已提交账本回执**，确认远端提交完成。
  - **必要输出**：可以在回执范围报告 `REMOTE_COMMIT_CONFIRMED`，不可再盲重试；仍需分开本地状态和用户完整业务交付是否成功。
- **变形 Oracle**：A 未知，B 仅远端提交可确认；两者都不许可新的付款 Effect。R10 没有真实 Provider 回执。

## V04 — Release 候选与实际发布制品（K08/K05）

**共同输入**：代码 H1 编译与已选用例测试都报告合法 PASS，可信构建渠道记录候选 Artifact `D1`；Release 权威审批和目标 OS Installer 完整验收**均未提供**。

- **EV10-04A**：从正式分发位置下载的安装包 SHA256 为 `D1`，与已测试的候选完全一致。
  - **必要输出**：仅可确认**已描述的制品身份一致性**和其已审证据的相应范围；仍是 `RELEASE_NOT_AUTHORIZED`、`HOST_NOT_RUN`，不得把 Digest 相等当成发布审批或 Windows 安装完成。
- **EV10-04B**：实际下载包 Digest 为 `D2`（与 `D1` 不同），且没有合格重打包等价性证明。
  - **必要输出**：已审 D1 的测试/Build 不足以为 D2 提供相同的制品证明；须核实来源和真实材料差异，Release 仍未批准。
- **变形 Oracle**：A=Artifact identity matches, B=identity mismatch；两者都不得输出已 Release。

## V05 — 项目隔离与凭据归属（K09/K01/K10）

**共同输入**：交互式多项目客户端可无配置启动并进入安全配置界面，两个项目均已由用户选择，模拟执行不触及外部 Host；相关用户权限流程仍需真实 Provider 验收。

- **EV10-05A**：配置模型经**静态材料**证明 Project A 和 B 的 Credential Reference/Repository/Version 作用域分别绑定，互不复用。
  - **必要输出**：可报告“静态配置层**没有出现所述共用引用**”；不得声称真实 OS KeyStore/Provider 操作已隔离，也不授予业务 Effect 资格。
- **EV10-05B**：配置模型将 Project B 指向 Project A 的 Token Reference，且无合法共享授权。
  - **必要输出**：识别跨项目凭据归属风险，阻断未经授权的操作，要求正确的用户项目身份/Token 范围；不能因为 UI 可打开或 Mock PASS 就当隔离完成。
- **变形 Oracle**：A 静态检查通过限域，B 有明确不安全引用。两者都没有真实 Windows/Host 收据。

## V06 — 性能可比性与真实用户延迟（K10/K11）

**共同输入**：某优化比较两组样本，声称平均响应由 100ms 降为 80ms；没有给出有权用户 SLO，也无证据证明请求业务正确性未变。两组数据均是**人工虚构**的数值。

- **EV10-06A**：两组记录均来自相同输入分布、相同用户 Journey/环境/Artifact 约束，统计摘要包含明确样本量和误差范围；仍未提供独立 Oracle 证明业务结果正确。
  - **必要输出**：可以讨论在这些**假定可比条件下**，样本平均差异为 -20ms；不能宣称生产体验提升 20%、P95 必然降低或正式验收 PASS，需补正确性和真实测量证据。
- **EV10-06B**：第二组换了简单的任务类型/更低并发，缺请求分布与业务 Journey 的可比性。
  - **必要输出**：不能将平均 20ms 差异归因到代码优化；需按任务分布/负载分层做受控比较，也不能从单个 CPU Flamegraph 推出 P95。
- **变形 Oracle**：A 允许限定的**假设下描述性差异**，B 不允许“优化导致差异”的归因；两者均 `NOT_REAL_MEASUREMENT`。

## 三、独立复核必须覆盖的六条硬负例

1. **测试漏发现**永远不能在同一 Test ID 被记作 PASS。
2. **Draft 权威缺失**不能靠更新时间/重复 URL 越过合法 Owner。
3. **未知 Effect**不能靠本地数据库 Commit、HTTP 响应或 LLM 推断变成确定失败，也不能因此自动重放。
4. **匹配 Artifact Digest**仍不得补出缺失的 Reviewer/Release/Target OS 收据。
5. **静态配置未见串扰**不等于真实 Token/Provider/Host 完成授权。
6. **理论/合成性能数值**不能伪称真实生产 SLO、Agent 经济收益或因果测试结果。

未来真实任务要记录 **Dataset Exact SHA、Goal、变形组、Source/Consumer/Provider 版本、Agent baseline/assisted 策略、全部尝试与失败、合法 Oracle、Reviewer 独立性和成本分母**。现在没有任何一次真实 Agent 被测，本集不能发布 Accuracy/PassRate。

## 四、与旧 Q/W 的关系

Q01–Q18 提供跨学科问题覆盖，W01–W22 提供完整演绎教学；本 R10 的 **EV10-01A～06B 是六组十二例最小事实变形**，用来评估“改变单个关键事实时结论是否变化”，并非新增 12 条 W，也非新增 12 个真实 Agent 运行。与已有 Q/W 的重复主题是特意用于反事实对照，区别在**严格控制单一变化维度**。当前总 W 仍 22、Q 仍 18，累计课程 A 仍 120。
