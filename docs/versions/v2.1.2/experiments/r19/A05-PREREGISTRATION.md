# R19 A05 · 工程抽象专业知识的前瞻性 A/B/C 实验预注册（公开协议）

> **PUBLIC_PROTOCOL_CANDIDATE / A05_INDEPENDENT_REVIEW_PENDING / G-B_NOT_STARTED / ZERO_AGENT_RUNS**  
> [R19 总控 #554](https://github.com/kaicreator-mm/tally/issues/554) · [已审并合并的工程抽象知识](../../software-engineering-knowledge/professional-engineering-abstraction-selection-r19.md) · [已审并合并的八例对照教材](../../software-engineering-knowledge/professional-engineering-abstraction-contrastive-casebook-r19.md) · [中立评价量表](A05-EVALUATION-RUBRIC.md)。
>
> R19 文档与 [R18 冻结研究 #551](https://github.com/kaicreator-mm/tally/issues/551) **相互独立**。本文只公开研究设计和方法，**不得存储**隐藏实际案例、Source/QA/维护题、参考实现、真实知识投放具体字节、组别真值或 Reviewer 答案。G-B 是未来实际冻结、隔离和独立资格审查，不是本次 A05 文档审核。本文不构成新的 ADS 规则、Tally/DH Runtime、SDK 或 Source Claim。

## 1. 真正要回答的研究问题

现有 LLM 可能实现全部当期功能，却在边界/变化轴/语义复用/错误合同上做出不好的工程抽象；相反也可能错误地增加多余接口、Registry 和 Factory。因此本研究只把能被**实际变化、理解或可证伪约束**支持的改进列为质量结果。

- **H0-1**：增加 R19 概念与对照案例，相比共同基线不会减少关键工程抽象缺陷，也不会改善独立维护和阅读表现，或收益被投入成本抵消。
- **H0-2**：在相同概念与案例基础上额外给出显式抽象决策程序，不会带来可观察的进一步改善。
- **H1（均为候选）**：案例引导下的知识包可以提高恰当的共性发现、状态 Owner/信息隐藏和变化隔离；额外决策程序可能进一步改善没有必要额外抽象时的 STOP、错误语义区分和真实维护成本。
- 不预设任何一组胜出；LLM 可以自行做出高质量抽象，知识也可能导致过度设计。

## 2. 三组的唯一预登记处理差异

| Arm | 共同输入 | 额外输入 | 可以回答的比较 |
| --- | --- | --- | --- |
| **A：BASELINE** | 完全相同的已发布 ADS、授权需求/工程质量标准、H0 Source、工具/模型/预算/阶段指令 | 不追加 R19 专业抽象材料 | 与 B 对照的共同基线 |
| **B：CONCEPT_CASE** | 与 A 完全相同 | **精确封存**的 R19 K02/K03/K09 概念性原理片段 + **同一批**正反对照案例 | A→B 仅估计“**概念 + 案例组合知识包**”效果 |
| **C：CONCEPT_CASE_DP** | 与 A 完全相同 | 与 B **字节完全相同**的概念/案例片段，再加有界的显式 K02/K03/K09 决策步骤 | B→C 估计“**附加显式程序**”效果，而非纯知识效果 |

- **A 不能被有意剥夺**合法的 ADS、公开需求、质量场景、运行工具或此前正常工程知识。B/C 不得获得额外业务合同、提示隐藏测试目标、参考解、Reviewer 评分答案、额外人工干预或更多 Effect 权限。
- 三组在各阶段获得**同结构、同标签**的交付信封；A 收到相同阶段 `NO_EXTRA_R19` 声明。不同内容的 Token 量、上下文占用和推理成本属于真实 treatment 成本，**不能冒称完全等长**。所有组的固定上下文窗口和最大预算相同，若 C 因更多字节挤出共同上下文必须标 `CONTEXT_DISPLACEMENT`。
- B **不得接触完整含 DP 的源文章**；C 也不能获得实验 QA。实际 Knowledge Deliveries 必须有 `SOURCE_GIT_BLOB`、精确 UTF-8 SHA256、选定 CASE ID、阶段、实际输入/输出 Token、送达回执和角色授权记录。G-B 负责封存**实际可见字节**，不是仅锁住文章 URL。
- **本三组不能识别“案例单独的因果效应”**。如果以后要单独回答此问题，需另外预注册“概念、但无案例”第四组并重新设计样本和独立审核，不允许看到本轮结果后自行拆分估计。
- 任何组在已知外部连接、浏览权限、之前对话或共享缓存里能访问未被授权的 R19 DP/案例或对方结果时，记录 `TREATMENT_CONTAMINATION` 并停止/标 `NOT_COMPARABLE`，不宣称已成功随机化。

## 3. 任务族与预冻结的完整分母

这轮不重用 R18 F1/F2/F3 项目、QA/维护任务，也不使用 R19 教材的图表/颜色/编码/字符串样本作隐藏任务。公开只能描述**任务族**，具体需求与答案必须在 G-B 独立封存：

| 任务族 | 有权质量风险（非隐藏实例级答案） | 预设真实后续变化能力 |
| --- | --- | --- |
| **T-A：既有域的职责和契约演进** | 既有组件的职责/调用方稳定性、公共接口、状态与故障条件 | 在原始合同不变时引入一项合法新能力，衡量跨 Owner 变化与兼容 |
| **T-B：多处机制的共性判断** | 看似相似的流程/数据/算法、错误边界和复用粒度 | 判断哪些变化应该共享、哪些应保持独立，避免错误泛化与重复维护 |

**固定样本**：2 任务族 × 2 独立重复/族 × 3 组 = **12 个独立首次 L2**，并计划对同样的 **12 份首次 S5** 执行首次实现验收及 **12 条独立陌生维护者增量路径**。不能挑选 L2 “最优秀者”再实施；首次 S5 功能不合格亦纳入完整分母，维护不具资格时标 `INITIAL_S5_INELIGIBLE`，不得静默删除。

四个三组配对区块须使用同一封存 H0/源合同，但不同区块有独立会话、工作区和首次随机性来源；严格隔离日志/缓存/模型上下文。各组顺序要在 G-B 前确定并留下机器可验的随机种子或不可回填分配清单。**只有四个区块，不能保证三组全部排列顺序完全平衡**；应报告残余顺序/会话效应，不得把 A/B/C 差异说成确定的纯因果效应。若模型种子不可控，应显式记录 Provider/版本和不确定性。

每次 Attempt 都是**一次性有限任务**，预冻结最大 Token、Wall Time、工具/人工介入、退出条件；失败/超时/无提交保留。新增资源或修正 Oracle 需要记录 superseded 版本并重审，不能事后补一个有利组的修复机会。

## 4. 阶段性知识送达和不可重写的首次产物

1. **S0 COMMON / PRESTART**：统一 ADS、受权要求、当前 H0 Git SHA、依赖/工具/权限/预算；G-B 已独立验证任务和 hidden QA 是看结果前封存的独立材料。
2. **S1 REQUIREMENT / L2 BEFORE**：相同质量场景给 A/B/C；只在 G-B 预设阶段向 B/C 给相同的概念/案例部分，C 按指定时机增加 DP。不能在输出变差后临时补知识。
3. **S2 FIRST_L2**：**任何实现代码开始前**冻结原始架构/抽象候选、状态 Owner、变化轴、反例/权衡、原始设计 SHA 或不可变内容哈希。设计缺陷即使后来修复也不能回写首次 L2。
4. **S3 IMPLEMENT / FIRST_S5**：同等工具/源合同和预算下实现，保留首个源码提交、测试 discovery/实际执行、错误与超时；固定 `FIRST_S5_SHA`。后期修复与初始实现分开记。
5. **S4 BLIND_EVAL**：真实非 Builder Reviewer 独立评初次 L2；陌生 Reader 阅读匿名首次 S5，对相同实现无关的问题作行为解释/状态定位；Review/Reader 不看处理组别和 Builder Prompt。两位 Reviewer 原始发现先各自封存，随后第三方仲裁。
6. **S5 HELD_OUT_MAINTENANCE**：只有首次 S5 全部封存后，独立维护者从对应原始 S5 SHA 出发，获得 G-B 前封存的**相同合法新增需求**，执行旧+新 Oracle。原 Builder 不得事先看到该需求。
7. **S6 UNBLIND / LEDGER**：两名 Reviewer 原始判断、仲裁和 Reader 答案在不可变证据中封存后，Coordinator 才把 A/B/C 真值和成本台账关联到匿名结果；真实盲性不能成立时据实标 `NOT_BLIND`。
8. **S7 INDEPENDENT_REPLAY / DECISION**：独立复算来源、实际隐藏 QA 校准、结果、成本与反证；不改首次源，不将补测晋升为原始实验成功。

**阶段标号仅属于 R19 研究会话，不改变 ADS 官方 L1/L2 或 #551 R18 的 S2/S5 原实验语义。**

## 5. G-B 的强制材料与实际验收

**仅在本 A05 公共协议独立审过后**，方可生成并审核：
`H0_GIT_SHA_AND_TREE`、`CASE_A_B_AUTHORIZED_REQUIREMENTS`、`QUALITY_ATTR_SCENARIOS`、`ARMS_THREE_EXACT_VISIBLE_UTF8`、`EXAMPLE_IDS_AND_NONISOMORPHIC_SCREEN`、`ADS_PIN`、`ASSIGNMENT_AND_ORDER`、`MUTANTS_AND_POSITIVE_NEGATIVE_REFERENCES`、`FIRST_S5_OLD_QA`、`SEALED_NEW_CHANGE_AND_NEW_QA`、`BLIND_REVIEWER_ONLY_PROJECTIONS`、`FULL_COORDINATOR_LEDGER`、`ROLE_AND_HOST_ACL_DENY_TESTS`、`PROVIDER_PROFILE_BUDGET`、`SHA256_MANIFEST`、`NON_WRITER_GB_VERDICT`。

- 不准把四类角色资料放在同一个可读工作区后称“已隔离”；ACL 需要以**真实各角色身份实际读取/拒绝访问**证明。Reviewer/Reader-only 的投影必须在初始判决前隐藏 `arm`、知识送达 digest、组别指纹、Token/成本关联数据；被代码设计风格无意揭露也应记录有限盲性。
- QA 必须对每类任务有**独立正参考、失败基线、能检测关键差异的故障变异**，并在 Windows/Node22 与正式环境进行真实测试发现/执行、退出码、异步清理检查；Oracle 无法检测关键违约则 `REWORK`。不得把 R18 QA 拿来复用。
- G-B 非 Writer **审核实际私有字节与任务实例是否与 EX-01–EX-08、旧 R18 案例同构**。教材可以有一般机制，但隐藏问题不能是换名字的课堂题，否则 `LEAKED` 且整个相关比较 `NOT_COMPARABLE`。
- **R18 Windows 预检证据只可用于学习怎样检查 ACL，而不能直接证明 R19 角色隔离**。在不同工作区/身份/Provider 与授权上下文下必须独立校验。
- 若 R18 当前仍有尚未关闭的实验且两者无法实现工作区、Reviewer、Provider 会话及知识上下文真正隔离，R19 **停留在协议/封包阶段，不启动 A07**；不催促 R18 改变其冻结 QA/知识方案。

## 6. 结论及研究效力边界

预注册三类 Primary（[量表](A05-EVALUATION-RUBRIC.md) 中细化）：(i) 初次 L2 的事实支持的 `UNDER/WRONG/OVER/JUST_ENOUGH` 与关键错误/质量场景；(ii) 非原作者的真实代码理解正确性/成本；(iii) 后续真实维护时的新旧行为通过率/变化隔离/成本。功能、安全、权限与数据损坏等硬失败必须单列，不允许靠“层次清晰”抵消；静态代码行数/类数/注释数为辅助观察，禁止成为主评分。

两任务族、各组每族仅两次尝试，**只做机制探索**；对差异必须给实际源、Reviewer 分歧、反例、资源和统计局限。任何知识送达 `DELIVERED` 不等于 `APPLIED`、`DECISION_CHANGED` 或 `QUALITY_IMPROVED`。不能据此声称案例知识已提高所有 LLM 的工程抽象能力。

**状态：本文件只是一份公开、待独立审核的候选研究协议。R19 A06/G-B 尚未构建和审过，A07/A08 均未运行。**
