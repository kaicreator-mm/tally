# R15 · K05：测试夹具生命周期、Oracle 有效性与反例区分力

> **DOCUMENT_CANDIDATE / EXPERIMENT_EVIDENCE_BOUNDED / NOT_SOURCE_CLAIM_QUALIFIED / NOT_PRODUCTION_RULE。**
> [K05 知识库](README.md) · [K05.ORACLE](topics/ORACLE.md) · [K05.FALSE_PASS](topics/FALSE_PASS.md) · [R13 不变量驱动区分测试](advanced-invariant-derived-discriminating-oracles-r13.md) · [跨模块知识选择 R15](../professional-risk-gated-knowledge-selection-r15.md)。
>
> 证据边界：R3 [#539](https://github.com/kaicreator-mm/tally/issues/539)（原试验首轮 QA 饱和）；R4 [#545 后置原始二进制复核](https://github.com/kaicreator-mm/tally/issues/545#issuecomment-7084408606)（首 S2 限定证据、原始 QA 无效）；R4 [#547 Windows QA v2 独立于 v2 作者的限定诊断审核](https://github.com/kaicreator-mm/tally/issues/547#issuecomment-7085569993)（P0=0/P1=0/P2=7）。这些是**具体实验的观察/限制**，不是通用 Rule、业务授权或真正独立的全实验多次复现。

## 1. 问题：测试执行成功，不等于被声称的性质已经验证

一个可用 Oracle 至少同时满足三类条件：

1. **业务真值有效**：Expected 来源于批准的需求、不变量或权威外部合同；不从候选实现抄一个 Golden。授权缺口由 K01 标 UNKNOWN。
2. **验证程序可信**：当前 Exact SHA 上真实发现并执行测试，断言正确、资源夹具时序正确，负例能失败，异步任务确实等到完成；Runner EXIT 0 不自动证明目标测试/性质执行了。
3. **观测窗口覆盖声称的性质**：观察的是外部可识别的最终事实或有权时间切点。一次进程退出后的绿色回读不能自动证明“回复成功后、进程退出前”从未发生过数据改变。

这三件事正交。测试夹具缺陷既可能制造**假失败**，过弱的断言也可能制造**假通过**；改变测试夹具不能当作修复受测业务代码。不能只比较 PASS/FAIL 数量推断功能质量。参阅 Node 22 [test runner](https://nodejs.org/download/release/v22.18.0/docs/api/test.html) 与 JavaScript [await 的 Promise 完成语义](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/await)。

## 2. R4 中发生了什么：保留原始与诊断两个独立指标

R4 两个 Node22/TypeScript 合成任务各有三个已经冻结的首次 S5 实现（A=仅 ADS、B=完整知识切片、C=R14 最小 L2 卡），**每任务每组仅一次**。R4 原始冻结测试具有可复现的资源夹具错误：

| 证据层 | T1 三组各自 | T2 三组各自 | 可以陈述什么 |
| --- | --- | --- | --- |
| 首次 S5 **原始冻结 QA** | **8/10**，SR04/SR08 失败 | **6/8**，ER06/ER08 失败 | 四个受影响测试面上的原始共同主要终点 **PARTIALLY_INVALID**，不能标成通过 |
| 后置、版本化 async-safe **诊断 v2** | **10/10** | **8/8** | Windows Node22.23.3 和 Linux Node22.16.0 支持的**限定诊断结果**；不回写原主要终点 |
| H0 未实现负对照 | 0/10 | 0/8 | 能拒绝该 H0，**不能证明**已经覆盖异步成功路径 |
| 对成功摘要定向破坏的变异 | 2/10 | 2/8 | 可检测被注入的**部分错误**，不能证明所有不变量 |
 
旧夹具的结构性失效是：同步的 `try { return fn(root) } finally { remove(root) }` 对返回 *pending Promise* 的回调立即执行清理。测试随后才访问目录，因此出现与候选实现无关的错误。更安全的生命周期模型应在同步返回、同步异常、Promise fulfil/rejection **各自结束之后**清理，保留原异常语义；对异步回调不能只看到其返回了 Promise 就当作测试已经完成。

这项根因已由原始冻结源码与六份候选复验定位。**QA v2 只改变夹具 `withRoots` 清理时序，原测试断言及用例数不变**；旧版源码和原结果仍然不可变。用于后置诊断的修正方案即使得到有限 `ACCEPT_SCOPED_POSTHOC_DIAGNOSTIC`，也不等于 R4 的 S5 预注册评分被追溯升级。

## 3. 已经被变异测试揭示的反例：绿色不等于风险覆盖

#547 Windows 复核中，下列反例必须保留为知识候选中的**失败证据**，不能因为 v2 全绿而略去：

| 风险命题 | 故意错误 / 对照 | 实际观测 | 能否宣称验证 |
| --- | --- | --- | --- |
| “成功回复后没有任何未声明后台状态变化” | 只对最后稳定性检查涉及的标识，在成功回复到进程退出之间清空状态 | v2 T1 **10/10**、T2 **8/8** 仍然通过（各重复两次） | **不能**。现有回读点未固定回复时快照，失去时间区间的可观察性 |
| “旧期望版本必须返回冲突，且不写新状态” | 顺序执行时忽略 T1 `--expect` 前置条件 | T1 **10/10** 仍通过（重复两次） | **不能**。竞态用例在锁路径先失败，未覆盖顺序不匹配 |
| “并发写者必须遵守唯一锁” | 故意移除 T1 Scope Lock | SR04 在 **3 次中只检测到 1 次** | 只有部分、时序依赖的证据；不能表述为确定性的并发保证 |
| “外部 Effect 竞争不得让败者伪造成功” | T2 败者将自己的摘要伪造成完成 | ER06 **3/3 次**发现 | 只支持该明确变异的区分能力 |
| “成功摘要不能凭空替换” | 在隔离实现里把成功响应摘要改为零值 | 两任务均有多个断言失败、两次重复一致 | 支持当前摘要检查，不覆盖时序或所有状态转换 |

**如何设计正确的时间窗测试**：首先在合同中区分业务已经作出回复的 `t_reply`、进程真正退出 `t_exit`、第一次可观察回读 `t_read0` 和稳定性观察 `t_read1`；若第一条独立回读只发生在 `t_exit` 之后，测试最多覆盖从 `t_read0` 开始的可观察区间，不能倒推整个 `[t_reply, t_exit]` 的无变化。要独立验证更强主张，必须通过可观察且授权的机制把成功回复时的权威版本/摘要/状态固定下来，再以明确切点的后续回读直接比对；无法观测则登记 `NOT_TESTED`，不得把 `sleep(500ms)` 当作证明。

**顺序冲突最小区分例**：先成功提交版本 `v+1`，再在**没有并发锁竞争**时显式使用旧 `--expect=v` 提交不同内容；唯一有权 Expected 是合同规定的 `CONFLICT`、已提交数据不变，不能用“竞争者锁失败”替代该断言。真实合同未要求版本约束时此测试不适用。

## 4. 对新测试方案的有限决策程序（不新增 ADS 必经 Gate）

1. **写清风险命题与故障模型**：任务/Owner/公开入口/状态事实/提交点/时间切点/OS/真实或模拟 Fault。无法授权的 Expected 留 `UNKNOWN`。
2. **先确认夹具是有效测量工具**：有确切 SHA、runner discovery、实际退出状态；用独立成功路径、同步/异步成功/失败清理对照证明夹具不会在测试完成前破坏环境。
3. **建立最小可证伪对照**：空实现/H0 必须在应失败处失败，正常参考必须成功；对每个高后果不变量故意移除一个保护机制，观察对应**具体断言**是否失败。变异活下来要判断“测试等价/前置条件未达到/Oracle 太弱/环境非确定性”，不能一概计作真实缺陷或直接消除数据。
4. **测试时间性单独处理**：回复时间、读回时间、进程退出与后台 Effect 属不同事件。若要证明无背景改变，测试必须观察合同真正声明的时间区间和权威状态，而不只观察进程退出后的静止性。
5. **保留证据分层**：`ORIGINAL_FROZEN_RESULT`、`POSTHOC_DIAGNOSTIC`、`NOT_TESTED`、`MUTATION_DETECTED`、`MUTATION_SURVIVED` 分开记；标明具体 Provider/Host/平台和样本数。不得将诊断绿灯替换预注册结果。
6. **按影响范围选择验证**：发现缺口则只补一个能够区分错误的不变量测试；没有需要的授权/平台/原始证据时记录停止条件；不因为文档知识存在而强制每次跑全 CI。

| 最小交付字段 | 必须可复算的内容 |
| --- | --- |
| 需求/Oracle 来源 | 批准合同、版本/Owner、Expected 真值及 `UNKNOWN` |
| 受测对象 | Exact HEAD、入口、真实资源与测试 SHA |
| 测试发现/执行 | 测试数、跳过数、exit、失败 ID、时间窗口 |
| 夹具可靠性 | 同步/异步成功失败与清理顺序的实证 |
| 正向/阴性/变异 | 正常对照、坏实现与“具体哪条断言杀死变异” |
| 未验证范围 | 时间窗口、并发调度、真实 kill、掉电、平台差异、权限/身份 |
| 结论等级 | 原始还是诊断、是否独立、能否复现、下一步最小验证 |

## 5. 跨模块责任与资料来源

- **K01**：有权业务合同，解决 Expected 的合法性；**K02**：状态 Owner、提交点、容许故障面；**K03**：真实代码实现与资源生命周期；**K05**：Oracle/测试夹具/反例区分；**K08**：具体 Runner/Host/构建与可追溯运行证据；**K06**：事故因果诊断。
- 本篇提出的是解释既有测试原理的**专业知识候选**，不增加 Workflow Gate、不自动授权外部 Effect、不改变当前 ADS v4.19.0，也不自动驱动 Tally/DomainHarness。对原先 R4 的 Reviewer 独立性限制（原 QA 作者也参与实验）与同私有 Release 下 Builder/Reviewer 缺乏强制权限隔离，应在未来独立实验中纠正；不得反过来宣称原实验已经隔离充分。
- 公共学科参考：[Node 22 test runner](https://nodejs.org/download/release/v22.18.0/docs/api/test.html)、[MDN Promise 与 `await`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/await)、[MDN Promise finally](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/finally)。这些资料仅支持语言/测试框架原理，**不证明 R4 工程合同正确或生产 Source Claim 已合格**。
