# R9 专业知识进入未来领域智能前的 Source→Claim→Oracle 准入差距

> [来源目录 A01–A72](external-source-catalog.md) · [165 点学科覆盖](foundation-concept-coverage-ledger.md) · [R9 演算 W18–W22](professional-reasoning-worked-traces-04.md) · [Tally 工程领域候选 #286](https://github.com/kaicreator-mm/tally/issues/286)
>
> **R9 CLAIM_READINESS_ANALYSIS_ONLY / INDEPENDENT_ACCEPT_SCOPED_AT_7adbb13bf4cc00c823f625e0f7d903bc880575ca / ALL_ROWS_NOT_QUALIFIED / NO_COMPILER_ABI / NO_SOURCE_DISPATCH。** 这不是新的 Machine Knowledge Schema，也不要求给每个课程 ID 创建 Native Issue。先具体列出 **R9 十三个已选择知识点**若要成为未来实际 Tally Agent 的专业判断依据，还缺什么**有权原始来源、限定适用和独立 Oracle**。不存在以引用数量自动得到合格 Claim 的规则。

## 一、知识正文、正式 Rule 和实际执行收据要逐层分开

```text
Published source (author/version/rights)
    -> proposed claim (own words, applicable scope and uncertainty)
    -> conflicting evidence/invalidating counterexample
    -> qualified authority and approved use
    -> selected DomainHarness consumer contract (version-pinned, not assumed)
    -> actual task-specific independent Oracle, if executed
    -> observed result with ExactSHA/ArtifactDigest
```

其中任何箭头都不能从“Markdown 有链接”或“独立 Reviewer 接受教学内容”自动推出下一层资格。学科正文可以借官方资料阐明专业方法，**正式业务政策仍需该业务合法 Owner**。`NOT_RUN` 表示没有运行选定测试，不是“模型不可靠”或“测试失败”；`UNKNOWN` 表示真实后果/权威来源没有被足够观察，不是对外部 Effect 的确定性失败。

## 二、13 个有明确边界的候选知识点

| 原课程 ID | 学科 Owner 与命题候选（不是生产规则） | 研究出处/适用来源 | 最短负例、实际缺失的有权收据或版本证据 | 当前资格 |
| --- | --- | --- | --- | --- |
| K03.F5.1 | 有错误阶段、后果确定性和安全动作的可行动错误分类 | A52、A70 | HTTP 504 不能证明远端没扣款；缺具体 Provider 的幂等/查询合同和当前响应收据 | REFERENCE_ONLY |
| K03.F5.2 | 诊断日志足够定位但不默认记录 secrets、租户隐私 | A36、A70 | 记录 Bearer Token 只是“方便排查”并不获得许可；缺实际敏感字段策略/Host 日志验证 | REFERENCE_ONLY |
| K03.F5.3 | Type/BUILD、Test Discovery、执行与业务 Oracle 分离 | A57、A58 | `*.spec.mjs` 未被 Runner 发现，Job GREEN 不代表其 PASS；缺目标 Runner/版本运行清单 | NOT_RUN |
| K04.F1.3 | 按风险/可逆性/强制证据选择最小合法工程流程 | A04、A10、A54 | 文档拼写修正被无意义地强制真实付款/Host 全流程；需项目合法 Required Check/权限范围 | REFERENCE_ONLY |
| K04.F4.1 | 不把有权需求缺口改写成已修复的源码 Bug | A05、A13 | 草案与旧 FAQ 冲突但先派 12 个 Builder 修改付款代码；需要权威政策版本/Owner 决议 | ASK_AUTHORITY |
| K06.F2.1 | 最小复现固定 SHA、输入、环境、执行事件 | A57、A59 | 目标测试一轮未发现却被称同测试 Flaky；缺两轮真实 testId/discovery 收据 | NOT_RUN |
| K06.F3.2 | timeout 后外部 Effect 保持 UNKNOWN，直到合法读回 | A32、A52、A71 | 远端已提交、ACK 丢失再重发；缺 Provider 当前真实幂等/查询合同/授权 | UNKNOWN |
| K06.F5.3 | 可复用经验需因果、触发、反例与失效事件 | A54、A71 | 某次网络重试变绿被推广成一切失败都该重试；缺可复核同类任务结果 | REFERENCE_ONLY |
| K09.F3.3 | 配置/数据迁移必须维持用户语义、Tenant/凭据边界 | A35、A60、A61 | 多项目复制同一 Token、单名强行拆分、CANCELED 当 REFUNDED；缺真实数据/产品合同 | NOT_RUN |
| K09.F4.3 | 受影响闭包内复核当前 Consumer，合法复用无关旧证据 | A53、A57、A59 | H2 Schema/Glob 已变仍复用 H1 TEST PASS；缺 H2 选定 Consumer/构建输出 | NOT_RUN |
| K10.F3.3 | 上游限流/背压、隔离、熔断和重试预算需符合可观察服务模型 | A31、A71、A72 | λ>μ 无界队列且重试放大入流；缺真实服务容量、丢弃和 Provider 状态语义 | NOT_MEASURED |
| K10.F4.2 | CPU 样本热点不能代替用户 P95 的因果证明 | A36、A63 | Flamegraph 35% 但用户 P95 多是远端等待；缺同工作负载与同 Artifact 的对照 | NOT_MEASURED |
| K10.F5.2 | 恢复 Oracle 需要真正用户路径/业务 Effect 状态 | A31、A63、A72 | 容器健康检查绿但队列积压/退款 UNKNOWN；缺真实用户 Journey/Provider 回执 | NOT_RUN |

所有条目仅处于**研究/候选层**。表中的 `NOT_RUN`、`REFERENCE_ONLY`、`NOT_MEASURED`、`ASK_AUTHORITY`、`UNKNOWN` 是**专业状态说明**，不是创建新的 ADS 规范状态机或 DomainHarness Wire Enum，也不意味着对应真实系统已经报这些状态。

## 三、哪些知识可先成为有价值的系统知识，而无需等生产资格

可直接作为**文档级学科解释**的内容：观察错误与提交后果的逻辑区别、Typecheck 与测试发现的逻辑区别、给定模型下 `λ>μ` 队列积压、需求来源有效性和版本差异、技术债/Release 证据复用的专业边界。它们既可用于教师式推理，也可作为未来 domain 的**可定位知识候选**。

必须额外求证的内容：任何具体真实 API 的重试幂等保证、某业务规则当前有权版本、真实 Tenant/Token 生命周期、指定 Node/CI/OS 的测试结果、真实缓存缺陷、真实生产 SLO/Release、真实 Agent 质量提升、具体版权或原始条款复制权限。

**优先建议**：当 #286 的真正 Consumer Profile 与 DomainHarness ABI 已有权冻结时，优先选择两个到三个能被真正观测的**只读且低风险**软件工程案例做小型 Source→Claim→Task Oracle 闭环。例如“正确发现被漏选测试”和“比较两个 SHA 的受影响构建证据”比“自动重试真实财务 Effect”更容易独立验证且不会扩大外部权限。这里没有启动这些工作，未对 #286 解冻或实际生成机器 Package。

## 四、为什么“知识库系统化”不能用 A 链接数代替

即使课程从 107 增加到 120 个 A 关联，也不意味着 LLM 在有 Source 版本冲突、运行能力不足或 Provider 不可见时会自动给出正确答案。真正有用的系统化建设需要**学科位置→前提→领域实体/关系→适用机制→反例→权威条件→实际验证**。当前 11 个模块和 48 条历史 Concept 关系继续作为文档逻辑组织基础；R9 没有新建未经校验的原始规范关系或伪造真实执行图。

与 [W18–W22](professional-reasoning-worked-traces-04.md) 的五个合成例子共同使用时，读者可以逐项判断应如何提出最小有权问题、核对运行测试发现与真实 Actor/Host 状态，并在不可证时停在 UNKNOWN。资格边界严格不变。
## 五、R10 选择真正可审原始命题并建立变形题（新材料待审核）

R9 的 13 行 Claim 缺口不因此晋升为合格知识。本轮从已有参考目录挑选**五个**版次和定位可核对的 [C10-01～05 来源命题档案](professional-primary-source-claim-dossiers-r10.md)，将其与 [EV10-01A/B～06A/B 六组十二个变形判断](professional-knowledge-evaluation-fixtures-r10.md) 和 [专业知识选择/评分方案](professional-knowledge-evaluation-protocol-r10.md) 链接。真实 Source/Consumer 仍未授权，因而五项候选资格均为 `REFERENCE_ONLY`；没有实际性能/正确性测量或新的 Tally 生产接口。遵循 [#286 最新 Owner 方向](https://github.com/kaicreator-mm/tally/issues/286) 中 ADS→Tally 的设计意图，但 #286 仍 `DEFERRED/NOT_FROZEN/NO_SOURCE_DISPATCH`。

