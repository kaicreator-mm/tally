# R10 软件工程领域知识选择手册：来源优先、最小充分、资格分层和正确停止

> [系统知识架构](systematic-knowledge-architecture.md) · [五项原始来源→命题档案](professional-primary-source-claim-dossiers-r10.md) · [六组最小变形评价题](professional-knowledge-evaluation-fixtures-r10.md) · [独立评价协议](professional-knowledge-evaluation-protocol-r10.md)
>
> **R10 DOCUMENT_DESIGN_ONLY / NOT_A_WIRE_SCHEMA / NO_CONSUMER_RUNTIME / SOURCE_DISPATCH_DISABLED。** 这是未来软件工程 Domain Agent 可能采用的**专业推理选择方法**，不是新的 DomainHarness Kernel、Domain Package、Agent Skill 自动调度、Tally Provider/API 或已冻结 Consumer ABI。当前 Tally [#286](https://github.com/kaicreator-mm/tally/issues/286) 最新有效 Owner 方向优先：ADS 是版本化软件开发实践/经验来源，Tally 是选定软件工程领域知识和工作流的应用使用方；**不得将历史“必须由第三方 Producer 生成才能继续”当作当前强制前置条件**。但 #286 仍为 `DEFERRED/NOT_FROZEN/NO_SOURCE_DISPATCH`，任何真实 Package/Consumer/Agent 接入都需要另外的版本与 Product/L2 授权。

## 一、为什么“全文检索最相似段落”不等于领域智能

一个 Agent 回答“付款超时后能否重试”时，召回许多包含“timeout”的通用教程，仍可能漏掉这次真实 Actor、Provider、业务键及远端已提交的可能性。**专业选择的对象不是文件数量，而是决策所需且适用的条件性命题**。

最短的知识消费链是：
```text
Goal + authorized actual source facts + missing facts
  -> identify necessary Kxx knowledge owners, not a single nearest keyword
  -> retrieve a small current, version-matched source/claim/counterexample set
  -> distinguish accepted engineering facts from unqualified academic methods
  -> reason about competing outcomes and the smallest falsifying probe
  -> bounded conclusion / ASK / UNKNOWN / STOP
  -> (only with separate authority) actual Task/Host/Review/Release
```

前五步可以成为**文档可核对的专业推理方案**；最后一步没有因本文获得执行权限。

## 二、六个有序判断：先确保有权事实，再选择深文

| 阶段 | 问题 | 最小需要 | 常见错误 |
| --- | --- | --- | --- |
| 1. 识别 Goal 和真实 Context | 谁在什么仓库、版本、Tenant、业务状态、目标 Host 上要求什么？ | Actual Goal/Actor/SourceSHA/Time/Environment；缺少时 `UNKNOWN` | 把默认示例仓库或旧 Issue 当当前 Task |
| 2. 分开规范/观察/假设 | 哪条 Rule 有权批准且已生效？实际执行观察到了什么？ | Norm Owner、版次、适用范围/授权；Observed Receipt | 三个博客重复旧 FAQ 的说法盖过现行业务政策 |
| 3. 选择知识 Owner | 当前缺口属于需求、架构、实现、测试、流程、Agent、交付、迁移还是可靠性？ | K01–K11 唯一主责和必要邻接关系 | “有 timeout”便只选 K10，不查 K02/K06 远端状态 |
| 4. 取最小充分原理与反例 | 哪个机制能区分两个看似相同的决策？ | 受审专业章节、实际有效 Source 片段、至少一个会翻转答案的反例 | 把全库 221 篇全部放入上下文，却遗漏真正有权的 Provider 合同 |
| 5. 分离资格和 Oracle | Reference 是研究线索还是有权 Claim？某个测试/Host 是否真运行？ | Exact SHA、文档/条款状态、Test Discovery/真实收据 | 只见 Markdown A 链接就说业务规则已获权，或 Job GREEN 就说所有测试 PASS |
| 6. 选择有界行动 | 现在能回答、能合法只读查询、还是必须 ASK/STOP？ | 能解释下个可证伪探针以及不可越过的权限 | 在无幂等证据情况下通过自动重试“验证”远端真实付款 |

表中阶段是**知识选择思路**，不是另一套强制 ADS 流程。小型只读修正可能在一次当前 GitHub 源码核查后直接解决，不应机械创建六个 Issue。

## 三、知识 Owner 不是按文件名硬匹配

**问题类型→主责→常见必要次责（按实际条件选择）**：

| 当前材料问题 | 主知识 Owner | 只有材料相关时才追加 | 反例触发 |
| --- | --- | --- | --- |
| Stakeholder/Rule/时间/租户冲突 | K01 | K04 权限接受、K05 验收 Oracle | 旧 FAQ 与已签发政策作用范围相反 |
| Extension 能调用什么、远端状态属谁 | K02 | K03 错误/幂等、K06 复现、K10 恢复 | A→B→C 误用传递依赖；本地 Commit 被当远端结果 |
| Test Job GREEN但目标合同缺席 | K05 | K03 Test Discovery、K08 Build/目标入口 | testId 不在当前实际 Run Manifest |
| Agent H1 成功消息追不上 H2 | K07 | K06 Exact SHA 复核、K04 接受、K08 Artifact | 旧 Review/ACK 试图晋升 H2 |
| 代码/生成器/配置/数据升级 | K09 | K01 业务语义、K08 制品、K05 回归 | `CANCELED` 错映射成 `REFUNDED` |
| 队列、重试、性能和用户恢复 | K10 | K06 根因、K02 效果边界、K11 测量 | Flamegraph CPU 35% 直接宣称 P95 降 35% |
| 方案成本、额外 Review/CI 是否必要 | K11 | K04 风险、K05 Oracle、K08 真实发布约束 | 只统计完成任务并忽略失败/人工成本 |

Kxx 只表示学科 Owner；具体运行授权和 GitHub 当前事实仍由有权 Source/Task/Host 所有，不得反向让一个学科章节改写真实权限。

## 四、三条最短可证伪领域选择演练

### S10-01：同一 Job GREEN，目标测试能否签 PASS？

**输入**：H2、Node22.16 Runner、Job Exit 0、40 个旧测试 PASS、目标 `contract.spec.mjs`。
**必要知识**：[K03 错误和 Test Discovery](K03/advanced-error-semantics-observability-and-validation-proof.md) + [K05 V&V/Oracle](K05/advanced-verification-validation-and-oracle-qualification.md)；官方 [C10-02 Runner 来源档案](professional-primary-source-claim-dossiers-r10.md) 是技术解释，不能替代真实 Run Manifest。
**短答演练**：若未提供目标 testId discovery/run，不能判目标 PASS；若明确未发现则 `NOT_RUN`；若明确运行且断言通过，只能在真实 Assert/Runner/Attempt 范围报告 `EXECUTED_PASS`，另问 Expected 是否有权。最小探针是读取当前 Runner 的目标 testId 列表，不应因为文档题创建真实 CI。
**变形依据**：[EV10-01A/B](professional-knowledge-evaluation-fixtures-r10.md)。

### S10-02：后端付款超时，数据库隔离是否保证“不重复扣款”？

**输入**：PG18 本地 `Intent` 已提交，Provider HTTP 504，远端 ACK 不存在。
**必要知识**：[K02 外部状态 Owner](K02/advanced-extension-authority-state-and-consumer-boundaries.md) + [K06 未知 Effect](K06/advanced-reproduction-unknown-effects-and-experience-qualification.md)；[C10-03 PostgreSQL 隔离档案](professional-primary-source-claim-dossiers-r10.md) 明确其事务边界，[C10-01 Problem Details](professional-primary-source-claim-dossiers-r10.md) 不授予支付重试权。
**短答演练**：未见受权 Provider 回执时 `OUTCOME_UNKNOWN`、不执行第二次付款；有可核实同业务键已提交账本时，只能报告远端已提交，不能凭此宣称整个用户业务已交付/可再次提交。未知就提出最小的授权只读状态读回。
**变形依据**：[EV10-03A/B](professional-knowledge-evaluation-fixtures-r10.md)。

### S10-03：H1 制品 D1 测试绿，能否给 D2 发布？

**输入**：编译与受影响测试指向 H1/D1，正式下载 Digest D2，Release Owner 和真实目标 OS Installer 结果均缺。
**必要知识**：[K08 Release/Provenance](K08/advanced-release-provenance-permission-and-rollback.md) + [K05 Artifact/OS Oracle](K05/focus-f3-3-build-package-platform-verification.md)；[C10-04 SLSA](professional-primary-source-claim-dossiers-r10.md) 只为来源/验证概念提供公开技术参考。
**短答演练**：不同 Digest 无等价证明时不能继承 D1 的完整测试范围；即使 D1==D2，也只支持制品身份，缺失有权 Release/目标 OS 不得补成 PASS。最小探针是合法下载校验 Digest、Builder/测试材料和真正被要求的 Host 验收。
**变形依据**：[EV10-04A/B](professional-knowledge-evaluation-fixtures-r10.md)。

## 五、最小充分不是永远只读固定三篇

对于只修复已确认的无语义文档链接，专业知识可能只需要 [K06 精确 Diff](K06/focus-f4-3-patch-currentness-and-review.md) 与相关目录链接；附加整个 K01–K11 反而浪费上下文。对于当前 Scope 的不可逆财务 Effect，缺少 Provider 的有权外部合同是**决议缺口**，再读 100 篇知识不能改变已知不足。

因此“是否可以停止检索”的条件是：
1. 已识别真正的当前 Goal/有权业务事实与所需范围；
2. 候选结论的关键机制有足够的技术/学科来源和至少一个真正可推翻它的反例；
3. 未解决的材料缺口已经明确列为 ASK/UNKNOWN/STOP；
4. 继续检索同一版本、同一出处的重复解释不会改变下一合法动作；
5. 若需要真正操作，已经明确额外的 Provider/Task/Host/Release 权限条件，而不在本知识库里擅自放行。

若系统不能判断某个 Source 的版次/身份/是否当前，就保留不确定性，绝不能靠“相似度最高”强行选择业务规范。

## 六、评估与经验回流不是已实现的自动循环

本手册提供 S10-01～S10-03 三条**原创纸面演练**；它们不是新 W 编号、也不是实际 Agent benchmark。未来按照 [R10 评价协议](professional-knowledge-evaluation-protocol-r10.md) 可比较 Baseline、Topic-only、Full-corpus 与 Scoped Selection，逐案记录真实 Correct/Incorrect/Over-abstain/Hard Fail，并在权限允许时形成有证据的**经验候选**。反馈仍区分客户端任务上下文选取错误与服务端知识库适用性错误；正式 DomainHarness Experience/Evolution 版本、周期、Rule Promotion 和执行权须另行裁决。本轮不部署后台任务、定时优化或强制 CI Gate。

**R10 里程碑的真实可交付对象是**：5项来源命题档案、6对12例变形题、评估协议和本选择手册，形成一套可审核的**系统性专业推理练习**；绝不是可下载的机器 Package 或已经观察到的 Agent 提效。
