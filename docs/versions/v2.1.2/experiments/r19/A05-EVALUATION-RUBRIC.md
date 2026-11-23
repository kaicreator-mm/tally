# R19 A05 · 工程抽象质量的中立可复核评价量表（公开设计候选）

> **PUBLIC_RUBRIC_CANDIDATE / A05_INDEPENDENT_REVIEW_PENDING / NOT_A_HIDDEN_ANSWER_KEY**。与 [R19 前瞻性预注册](A05-PREREGISTRATION.md)、[R19 #554](https://github.com/kaicreator-mm/tally/issues/554) 配套。复用 [R18 已接受的通用质量评价原则](../../software-engineering-knowledge/professional-r18-quality-evaluator-rubric.md)，**不修改原 R18 文档、任务或评价结果**。这是一套中立的方法，不允许 Reviewer 依据 A/B/C 组别、教材提及的模式名称、文件数、行数、文档长度判断胜负。

## 1. 评价对象和证据资格

**Primary 1 · FIRST_L2_ABSTRACTION_JUDGMENT**：首次冻结的领域/模块/机制抽象决策在当前 Source/质量场景下是否有效，必须可由真正约束和反例证伪。后续代码修复应另行记 `POST_L2_IMPROVEMENT`，不能擦掉最初设计缺陷。

**Primary 2 · FIRST_S5_STRANGER_COMPREHENSION**：一个没有参与最初开发的 Reader，能否在相同可用工具和预算内正确预测被测代码的公共行为、定位状态/错误 Owner 与变化位置；计数准确率、实际导航/时间、提示、不能回答的来源不足。不能用 Reviewer 说“很易懂”代替真正的答题结果。

**Primary 3 · HELD_OUT_MAINTENANCE**：独立维护者从真实首次 S5 SHA 出发，按 G-B 前封存的合法新需求实现变化；真正运行旧/新正负 Oracle，记录兼容回归、必要/非必要职责模块触及、独立阅读、返工和全成本。功能不合格的首次 S5 仍保留在分母中。

**硬功能门槛**：授权行为、旧合同、隔离、安全、权限、错误语义、资源清理、数据完整性分别报告；任何重要违约都不能被好看的架构/较少 Token 补偿。QA 必须有真正的测试发现/执行和结果凭证，不能把不存在的测试记为 PASS。

每个判定使用同一证据链：

```text
SOURCE/QUALITY_SCENARIO_ID -> FROZEN_CANDIDATE_SHA/LOCATION
-> INVARIANT_OR_FAILURE_MECHANISM -> FALSIFIER_OR_ACTUAL_TEST
-> CLASSIFICATION / SEVERITY / LIMITATION
```

未知 Source、故障是否可达或缺少质量阈值时使用 `UNKNOWN`；未执行验证使用 `NOT_TESTED`。没有有效可证伪机制的“架构很乱”只作为 Reviewer 意见而非主要缺陷。

## 2. 四类抽象判断，不设“抽象越多越好”的赢家

| 分类 | 必须有的证据 | 同样应考虑的反证/合法替代 |
| --- | --- | --- |
| `UNDER_ABSTRACTION` | 已证明相同不变量/变化方向的机制错误地重复，或真实 Owner 边界泄漏使授权局部变化必须重复/跨职责维护，且有具体变化/故障路径 | 代码外形相同但业务规则不同；拆分反而造成事务/清理错误时不能凭重复就判差 |
| `WRONG_ABSTRACTION` | 统一合同错误地混同授权/错误/事务/状态 Owner/资源生命周期，实际或可验证的合法刺激推翻共同接口 | 纯机制可共享、各自有权边界正确维护差异时，并非所有统一接口都错 |
| `OVER_ABSTRACTION` | 不必要的层级、Factory、Registry/通用接口造成**可观察**调试、导航、失败/维护负担，而目前无合法扩展/替换收益 | 多实现、真实兼容/隔离要求足以证明额外接口必要；不能机械数层 |
| `JUST_ENOUGH_ABSTRACTION` | 现行授权不变量清楚、变化被恰当隐藏，真实 Owner/错误合同保持一致、代价合理 | 可能**完全没有新增接口**；当前没有反证并不证明所有未来变更都安全 |
| `UNKNOWN/NOT_TESTED` | 源合同/变更实例/执行证据不足，不能判断抽象效果 | 不能用漂亮 ADR 或模型自述补足证据 |

**严重程度单列：** `MATERIAL` 需真实当前合同/质量场景违约、明确输入/状态/代码路径及故障刺激或强可验证机制；`NONMATERIAL` 是有依据的成本/演进风险，不能假装已导致当前失败；`STYLE_ONLY` 与模式名/形式差异不进入主缺陷计数。同一个缺陷不能在 `UNDER` 和 `WRONG` 重复计为两个独立 MATERIAL；必须提供 Root Cause ID 与消重规则。

## 3. 四道等价测量，避免通过模板刷质量分

**D1 · State/Responsibility Owner**：指出唯一权威写入、只读投影、合法消费者、预期故障/恢复边界。隐式共享可变写者、错误源丢失可形成具体风险；合法单模块事务结构不得因“未拆 Package”被扣分。

**D2 · Real Semantic Reuse**：判断复用前置/后置条件、授权、错误、时间、生命周期和真实变化方向是否相同。仅形似并不等价；抽出来的接口若允许不同语义漂移，需定位真实消费者影响。

**D3 · Minimal Variation Boundary**：比较至少两种可行设计（可以“保持当前直接实现”与“局部 Helper”），说明一个真正变化源会影响哪一份职责与接口。强制完整 ADR/完整 Design Pattern 表不能作为正向评价；无必要替代时保留简单方案可得正当结论。

**D4 · Change Locality Under Held-Out Change**：预先由独立 Oracle Owner 根据受权需求定义应变化的**职责集合**，不按文件名强制规定。实际对比 `REQUIRED_OWNER_TOUCH / INCIDENTAL_CROSS_OWNER_TOUCH / CONTRACT_COMPATIBILITY`；如果一文件内有多个职责，不能把 touched files=1 自动打高分。旧新测试都要实际运行，非通行观察必须 `NOT_TESTED`。

## 4. Reader 和 Maintainer 的实际题目资格

- **Reader** 在看任一组实现之前，由非 Writer 按真实公共行为准备同构但不依赖任何候选私有变量名的理解题：给定输入的输出/错误/状态、从公共入口定位权威写者、一个小合法变化会改哪些职责、哪些重要结论不能从当前资料推断。题目和行为 Oracle 必须在 G-B 之前冻结。
- **成功表现**：`CORRECT/INCORRECT/UNKNOWN` 各计一次；真实耗时/代码位置访问/跳转、问题提示次数、已获得事实的准确性分开记录。某 Reader 猜对输出但无法证明来源，不能等同确定理解。
- **Maintainer** 是另一个未写初始代码的实体，不能由同一 Builder 在另一个窗口扮演；预登记维护者模型和权限、可能的顺序携带效应，保留真正的角色证据。初始 S5 不合格的维护尝试不能通过选择性排除制造“高维护成功率”。

## 5. 盲评、校准与冲突处理

- 每个 FIRST_L2 至少**两名非 Builder** 独立 Reviewer 先提交带版本 SHA 的事实与负发现，再由第三名按相同冻结证据标准仲裁。两份首判与分歧不得被后写成同一个结果；不能由于知道某组“知识更多”而加分。
- REVIEWER_VIEW 只包含匿名 Candidate ID、共同授权 Source/质量场景、冻结设计/实现与允许的 QA 观察；**不得含** `arm`、`assignment_code`、`knowledge_excerpt_sha`、`delivery_schedule`、`builder_prompt`、`cost_token_total` 或人类提示组别。审查者如果看过完整源码中的明显组别标识，明确 `BLINDNESS_LIMITED_BY_CODE` 或 `NOT_BLIND`。
- Coordinator 专用 FULL_LEDGER 包含真实 A/B/C、知识字节/时间/成本，可在首判和仲裁不可变封存后关联。审查者身份/文件访问必须在真实 Host 以不同凭据测试**允许与拒绝**，不接受仅把不同文件命名为 reviewer-only 的形式隔离。
- **Oracle 校准在正式试验之前完成**：至少对各任务族提供可被完整旧新 QA 接受的独立正参考、一份有意失败的负参考、两个与抽象判断有关系但不等同设计模式的坏行为/兼容变异（如真实生命周期/授权或契约差异）。允许更多 Mutants，但必须冻结总分母与幸存变异及未测面。若 QA 对关键破坏不敏感，标 `ORACLE_LIMITATION`、修复后重新审，不可事后拿修复过的 QA 改写首次 PASS。
- 若 Reviewer 分歧来自新增事实，先遮组别按同一方法复核全部相关候选；保留每次原始判决和新版本，不允许按获胜愿望选择对照。

## 6. 记录全部样本与成本，避免假因果

每个 12 初始 L2/S5 样本及 12 预选维护路径必须保留：

```text
CASE_FAMILY / REPLICATE / MASKED_CANDIDATE_ID / H0_SHA
SOURCE_AND_QUALITY_SCENARIO_SHA / ADS_VERSION_SHA / MODEL_PROVIDER_PROFILE
KNOWLEDGE_PACKAGE_DIGEST [COORDINATOR ONLY]
FIRST_L2_SHA / FIRST_L2_QUALITY_FINDINGS / REVIEWER_A / REVIEWER_B / ADJUDICATION
FIRST_S5_SHA / TEST_DISCOVERY_IDS / ACTUAL_EXIT_CODE / HARD_FAILURES
READER_ID / READER_ANSWER_DIGEST / COMPREHENSION_CORRECTNESS / READ_TIME
PRESEALED_CHANGE_DIGEST / MAINTAINER_ID / MAINTENANCE_SHA
OLD_ORACLE / NEW_ORACLE / MUTANT_PROBES / NOT_TESTED
OWNER_TOUCH_SET / INCIDENTAL_TOUCH_SET / INTERFACE_CHANGE
INPUT_OUTPUT_TOKENS / REWORK / WALL_TIME / HUMAN_TIME / TOOLS_COST [COORDINATOR ONLY]
ATTEMPT_STATUS / TIMEOUT / MISSING_RESULTS / UNBLIND_TIME / ACL_RESULT
```

缺失或未提交仍在分母内，按原因如实记 `NO_SUBMISSION`、`INELIGIBLE`、`NOT_TESTED` 或 `NOT_COMPARABLE`，不可只分析成功实现。成本按单位分别汇总，Token 不和人工小时直接相加；执行/生成/校准/独立测量/维护成本都公开（可保护身份与保密材料）。

**输出不可跨维度抵消**：分别报告首次抽象缺陷、Reader 正确性、陌生维护成功、硬失败、总成本、知识送达/实际使用证据。不能得出“代码越多模块越好”“C 最复杂所以胜”“有文档因此质量提高”。每族每组 n=2，只作探索性观察，提供反例/共存解释和非因果局限。

## 7. 文档审核和后续材料边界

A05 Reviewer 只审核此公开评分口径与 [预注册协议](A05-PREREGISTRATION.md)，不得为某组调整未来任务和得分规则；**不批准隐藏题目或执行 Agent**。下一步必须独立完成 G-B 真实授权源、精确字节输入、封存样例 IDs、真实 Role ACL、QA Mutation 敏感性、维护题、随机/顺序/预算和双盲/非盲诚实标记。

本研究**与 R18 独立**：原先 Windows G2 预检报告或后续 R18 接受记录不能替代本 R19 角色资格审查，也不能向 R19 实验者提供 R18 hidden answers。若无法保证隔离，R19 维持 `STOP_BEFORE_AGENT_RUN`。

**本文无任何实验测量值，无知识增益结论，无生产资格。**
