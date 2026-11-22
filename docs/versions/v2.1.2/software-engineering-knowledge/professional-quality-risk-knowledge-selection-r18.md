# R18 · 用当前工程风险选择最小质量知识：架构、可读性和后续维护

> `DOCUMENT_CANDIDATE / QUALITY_KNOWLEDGE_SELECTOR_NOT_RUNTIME_RULE` · [系统知识库](README.md) · [R15 风险选择既有方法](professional-risk-gated-knowledge-selection-r15.md) · [R10 评价协议](professional-knowledge-evaluation-protocol-r10.md) · [#551 Task DAG](https://github.com/kaicreator-mm/tally/issues/551)。
>
> **职责**：为已有 K01–K11 专业知识增加面向质量判断的**选择视图**，不是新 Owner、Schema、强制路由器或 Tally/DomainHarness 已启用的 Consumer。没有现行 Task/Source/质量目标，不主动注入下面五篇。

## 1. 不是“所有任务都读五篇”，而是按风险和决策时机选用

| 真实触发事实 | 最小主要知识 | 对决策的预期帮助 | 不适用与 STOP |
| --- | --- | --- | --- |
| 实际质量属性驱动、复杂状态 Owner、模块边界/依赖/架构备选 | **L2：**[K02 场景驱动架构质量](K02/advanced-scenario-driven-architecture-quality-r18.md) | 把环境/响应/可观察措施映射到职责 Owner 与最小可证伪场景；拒绝过度设计 | 无获权质量场景不编造数字阈值；简单文档变更不需要 ATAM |
| 类型/方法/包组织复杂，读者难从公开入口推理真实行为或失败流 | **L3：**[K03 代码理解与抽象成本](K03/advanced-code-comprehension-and-abstraction-quality-r18.md) | 识别隐藏副作用、间接性、误导性命名、抽象收益/代价，能说明具体路径 | 不强制全仓重构；LOC/注释数/短函数非唯一起分数 |
| 修改旧实现或真实后续维护目标，关心兼容/变化传播 | **维护：**[K09 陌生维护者修改方法](K09/advanced-stranger-maintainer-change-assay-r18.md) | 保留原行为、控制传播、测试新旧合同，比较真实接手成本 | 无后续需求/独立旧合同时不能声称“维护更便宜” |
| 需要独立比较首次 L2、S5 可读性和后续维护表现 | **QA：**[K05 质量审核和评分校准](K05/advanced-quality-reviewer-calibration-r18.md) | 把重大设计缺陷与风格分歧分开，要求独立盲评、正负/变异对照及原始判断 | 实验 Reviewer 不应提前看到隐藏答案；普通小修不启用全面双 Reviewer |
| 方案初期更贵却可能后续易维护，需要判断经济性 | **决策：**[K11 首次到后续维护总成本](K11/advanced-total-quality-cost-and-maintenance-economics-r18.md) | 分开首次/测试/人工/返工/维护成本，成本与质量同单位公平比较 | 质量合同不等价、不存在精确成本来源、样本很少则不能宣称 ROI |

**允许不注入**：任务只是文档拼写、指定的单处类型修改、已证实的简单局部 bug 时，优先复用现有合同与改动验证。缺少权限或业务事实时先走 K01 有权要求与 `ASK/UNKNOWN/STOP`，**不能通过多读专业知识推导出授权**。

## 2. 质量判断的最短证据链

`CURRENT_TASK + AUTHORIZED_QUALITY_REQUIREMENT + CURRENT_SOURCE_SHA` → `RISK_TRIGGER` → `APPLICABLE_Kxx`（可为 NONE）→ `CURRENT_DECISION`（首次 L2 / 实施 / 首次 S5 / 维护）→ `ALTERNATIVES_AND_FALSIFIER` → `INDEPENDENT_OBSERVATION` → `LIMITED_VERDICT/NOT_TESTED`。

这个链条不要求每个决策新增一个独立 Issue 或流程。对重大架构决定应有 ADR/失败条件；代码路径应有真实执行来源；维护主张必须有后续变更实证；质量 Reviewer 必须写清哪些缺陷基于真正有权要求、哪些只是个人风格。知识 **DELIVERED/READ ≠ CHANGED_DECISION ≠ IMPROVED_QUALITY**。

## 3. 同一质量任务的多阶段交接

1. **L1**：原有 K01/项目 Owner 固定需求、受影响用户、业务不变量与必要质量属性，不能让某组看到更详细的“质量目标”而另一组被隐瞒。
2. **L2**：只有触发实际架构风险时才选 K02，给候选优缺点和可以推翻方案的场景。**第一次 L2 要单独封存**，不得用 S4 修复掩盖原架构失误。
3. **L3 / 实现**：选 K03 的方法与反例，避免把局部代码行数或抽象层数当作优化目标；同样遵守真正项目代码/工具/直接依赖规范。
4. **QA / Review**：K05 独立核验权威行为、Reader 理解和设计的具体可证伪失误。运行功能测试和审查架构是不同结果；实际独立性无法证明则明确 `NOT_BLIND`。
5. **后续维护**：只有预先授权真实变更时使用 K09，陌生维护者按冻结原代码、旧新合同进行改动。K11 合并有真实单位的时间/Token/人工成本，同时保留严重错误与未测项。

**避免泄漏**：以上只是通用专业领域知识，不包含 #551 将来的三个特定实验题目、隐藏 QA、架构评分锚点、后续维护变更或可直接照抄的解答。正式实验开始前，必须分别冻结知识版本/选取表与真正独立的实验数据，Evaluator 不可引用知识文章的自述作为候选 PASS。

## 4. 资格与证伪边界

- 本篇复用 [R15](professional-risk-gated-knowledge-selection-r15.md) 的既有源/风险/Owner 路由；只是把原来偏并发/持久化的选择例扩展到架构、代码理解和维护成本。既有 11 模块与 165 课程点不变。
- 评价方法可参考 [ISO/IEC 25010:2023 官方九大产品质量特性](https://www.iso.org/standard/78176.html)、[CMU SEI ATAM Collection](https://www.sei.cmu.edu/library/architecture-tradeoff-analysis-method-collection/)、[Google 工程审查方法](https://google.github.io/eng-practices/review/reviewer/looking-for.html)。引用这些来源仅支持专业背景，不产生生产级 Source Claim，也没有复制标准正文。
- R1–R4 的单样本、不同任务、后置评价和 R4 QA 原始失效都必须保留；不以论文/教学方法的存在宣称质量提升。真正是否改善，只有按 #551 G1/G2 之后的独立实证才能回答。
- 当前 `NO_RUNTIME_CONSUMER / NO_ADS_GATE / NO_AGENT_STUDY / NO_RELEASE`。缺少真实质量目标或当前版本则 `NOT_APPLICABLE`，而不是自动展开五篇文档。
