# R12 ADS 全流程复杂工程任务：分阶段知识注入的真实 Agent A/B 实验

> [R11 单任务真实消费设计](professional-live-knowledge-consumption-pilot-design-r11.md) · [R10 独立评估协议](professional-knowledge-evaluation-protocol-r10.md) · [R10 知识选择手册](professional-knowledge-selection-playbook-r10.md) · [165 课程点](foundation-concept-coverage-ledger.md) · [ADS v4.19.0 固定基线](https://github.com/kaicreator-mm/ai-development-standard/tree/933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c)

> **R12 EXPERIMENT_DESIGN_ONLY / R1_LOCAL_RESULTS_NOT_IN_GITHUB_RECORD / NO_REAL_R2_AGENT_RUN / NO_RUNTIME_OR_PACKAGE_CONSUMER / PRODUCT_L2_NOT_FROZEN。** R12 测量 *K01–K11 专业知识对遵循相同 ADS 规范的本地 Agent 实际工程行为的额外影响*；不使用 Tally 应用或 DomainHarness Runtime，也不冒充正式生产消费者。R1 本地实验标签 TALLY_V212_LOCAL_AGENT_KNOWLEDGE_AB_R1 已由用户报告完成，但当前可读取的 GitHub #521/#533 未包含它的 TEST_REPO、H0、A/B、Oracle 或成本收据；因此本次**不推定 R1 PASS、不推定知识已提效**。本地 Runner 开始 R2 前应读取 R1 原始报告，仅用于校准实验预算和泄漏风险，不以其单次结果选择性重写 R2 的评估 Oracle。

## 一、唯一检验的因果变量

**A = 当前有效 ADS v4.19.0 规范 + 真实任务/源码/原始有权 Source + 普通 Agent。**

**B = 完全相同的 ADS 规范、任务、源码、Actor/工具/模型和实验预算 + 在明确阶段入口逐批提供 pin 住的 K01–K11 专业知识。**

A 不是“裸 Prompt 且不给标准”的人为弱基线。B 不可以额外得到隐形验收答案、更多工具权限、旧 A 组输出或独享 ADS 规范；B 的差异**仅为选定专业知识**。B 可随着工作进入下一阶段保留已经提供的前阶段知识和自己产生的有权产物，但不能预先读取未来阶段的 Kxx 包。每组各自独立会话/工作区，不跨组复制经验、缓存/生成物或人工修复。

**已生效 ADS 源**：v4.19.0 commit \`933c4fd3e0e1b9f1a73ed6a9b552077baaf5376c\` 的 \`standards/DEVELOPMENT_WORKFLOW.md\`、\`standards/VALIDATION_STANDARD.md\`、\`standards/CI_EXECUTION_STANDARD.md\`，以及 \`prompts/L1_PRODUCT_EVIDENCE.md\`、\`L2_ARCHITECTURE_EVIDENCE.md\`、\`L3_IMPLEMENTATION_EVIDENCE.md\`。以目标测试项目真实 \`PROJECT_OVERRIDES\`/必需 Gate 为准。ADS #1383/PR #1384 三阶段 DAG-P/A/I 的新增定义为**尚未正式发布的设计候选**；仅当两组预先一致采用同一实验配置时，才可用于研究，不得冒充 v4.19.0 已生效额外权限或 Gate。

## 二、一个有材料复杂度、但不涉及真实财务/生产 Host 的任务

**Task ID: SE-R12-RESUMABLE-MULTIPROJECT-IMPORT。**
在一个有权测试仓库的 **Node >=22.12 / TypeScript Git monorepo** 中，从相同冻结基线 H0 完成一个可恢复、分项目隔离的本地批量导入服务。测试仓库应包含最小 starter skeleton 和真实脚本，而不是两组各自从空仓库造完全不同的基础设施。建议单一 monorepo 下的 \`packages/contracts\`（领域合同）、\`packages/import-core\`（状态/并发/业务服务）、\`packages/storage\`（本地持久化适配）、\`apps/cli\`（命令入口）；**不使用 Tally/DomainHarness，且不新增通用平台框架。** 具体 H0/项目 Toolchain 由本地准备者在受权实际仓库中冻结；没有仓库和 H0 时本篇不声称存在可执行源码。

### 对两组公开且等同的需求（不是隐藏答案）

| ID | 必须实现的可观察行为 | 真实验收面 |
| --- | --- | --- |
| FR-01 | \`projectId\` 维度隔离导入数据/状态/查询；不同项目相同 \`requestKey\` 不能互相污染 | 双项目同名请求键、跨项目读取拒绝 |
| FR-02 | 对同一项目内 \`requestKey + 内容摘要\` 提交重试返回同一 Job 身份；同 key 不同内容明确冲突 | 并发双提交、同键异内容 |
| FR-03 | 导入流程具有明确 PENDING/RUNNING/COMPLETED/FAILED/CANCELED 转移；非法状态跃迁拒绝 | 终态再次执行、取消后状态与部分输出 |
| FR-04 | 批量数据具有明确持久化检查点；异常终止后重启可以有限恢复，不重复产生对外可观察的业务记录 | 在写入前/写入后/ACK 前注入 crash，恢复后计数与身份 |
| FR-05 | CLI 支持 submit/status/resume/cancel、明确错误码、可行动诊断且不泄露凭据/输入隐私 | 非法项目、坏输入、不存在 Job、恢复失败 |
| FR-06 | 并发处理受控，不因双 Worker 或重复调用突破隔离、幂等和状态不变量 | 至少两 Worker、批次交错、同项目/跨项目 |
| FR-07 | 旧格式检查点遇到不能无损映射的状态要安全停止或要求有权转换，不静默标 COMPLETED | 兼容/未知版本/损坏输入负例 |
| FR-08 | 有真实选定测试 Discovery/Run/Exit 和受影响代码/消费者证明，不能只以原有 Runner GREEN 宣称全部验证完成 | 实际目标 testId、负例、Source SHA 和执行日志 |

**NFR**：本地文件/Mock 外部边界；目标环境 Windows 原生 Node22（可加 Linux，但不强制不必要矩阵）；状态持久化与并发模型可解释、少依赖、重复实验可清理；不承诺任何未经测量的吞吐/P99；不要求 GUI 首次配置合同套到 CLI；日志不含 secrets。不得引入真实支付/网络 Provider/客户数据、任意跨仓操作、生产 Release。持久化具体采用已验证的本地技术，L2 必须解释选型、崩溃一致性和原子性边界，不强制 SQLite 或引入不支持本机 Node22 的依赖。

**任务复杂而有界**：需要跨合同、核心、存储、CLI 和集成测试四条主链，真正处理三种风险：身份隔离、同键并发、崩溃后的事实恢复；不要人为追加微服务、Kubernetes、通用任务总线或真实云部署。QA 在交付任务前编写并冻结**没有公开给 A/B 的材料性负例**，但隐藏 Oracle 只能验证上述公开合同，不能偷偷提出额外功能。

## 三、知识何时开放：只在相应阶段进入 B 组

| ADS 工作阶段，两组都必须执行 | 两组共同得到的事实/流程 | **仅 B 在这个阶段入口新增的专业知识** | 阶段可审交付和判别点 |
| --- | --- | --- | --- |
| **S0 Intake & baseline** | 实际 Repo/Issue/H0/ADS PIN/Runner、受权代码写集、需求原文和冻结 QA 标准 | **无 Kxx 正文**；只知道存在可选专业库 | Source/Role/Gate 基线清单；不能先泄露后续知识 |
| **S1 L1 Requirements / Product** | 相同需求、Owner 的预冻结澄清回答、需求模板/受权口径 | **K01** 权威/需求追溯 + **K11** 风险/信息价值（按当前问题选择最多约2–3节） | 用例/Actor/状态/数据合同/开放未知、FR/NFR/非法行为/验收映射；原问题缺口识别 |
| **S2 L2 Architecture / Design** | 相同上游已授权需求、源码合同、Architecture ADR 要求 | **K02** 状态 Owner/依赖/错误边界、**K09** 演化兼容、**K10** 故障/恢复（按当前风险选） | 备选与 ADR、原子写/检查点/并发策略、数据流、故障与迁移回滚、Interface、不变量、反例 |
| **S3 L3 Task DAG / Detailed Design** | 相同 ADS Task/Review/Issue 权威、Scope/现有实现约束 | **K04** 真实依赖与风险驱动拆分、**K07** 角色和来源当前性（仅有关时） | 有材料依赖的 DAG-I、Concern 写集/责任、源头 Trace、Review Policy \`required/recommended/not-required\`；禁止伪 DAG |
| **S4 L3 Implementation / Local Validation** | 相同代码、测试工具、Commit 检查规范、合法依赖 | **K03** 错误/幂等、**K05** Test Discovery/Oracle；发生真实失败时才可按预登记条件进一步给 **K06** 根因与区分性探针 | 真实 H1A/H1B 代码、实际 syntax/type/test/命令、错误日志、最小负例；不因学理文本就报 PASS |
| **S5 Review / QA / Integration** | 相同隐藏 QA 审查入口、真实必要验证、权限与 Review Policy | **K05** V&V、**K08** Build/制品/Source 证据、必要时 **K10** 恢复验证 | Frozen candidate、独立执行 Oracle（QA 不属于 B 的模型 Context）、限定审查与差异；只有允许的修复窗口才向 Agent 返回材料失败摘要 |
| **S6 Closure / Maintenance Feedback** | 相同 ADS closure 规范、真实源码/测试/成本收据 | **K06/K09/K11** 仅用于**事后解释/经验候选**，此时 Agent 不再改已冻结 candidate | 每个风险/测试/成本/失效解释和候选改善；不自动写规范、强行 Release 或把 Agent 自评当 Owner |

**分批机制**：知识版本 \`CORPUS_SHA\` 固定；每个 S1–S6 的 B 专业包在启动 A/B 前由独立的 Knowledge Selector 预提交 \`STAGE / Knowledge ID / exact path / blob SHA / chunk / selection rationale / applicability / counterexample\`，且只有当前 stage gate 才投递。若在后续阶段触发已预定义的额外 K06 故障帮助，需记录触发的真实日志事实/授权、同时保留该检索的调用与成本。不要在看过 A/B 的评估结果后“临时精选”让 B 得到答案。单阶段优先 2–3 项真正有帮助的原理片段，避免全文灌入导致“只是 Context 更长”的混杂。**所有已发布的 ADS 必要规范和用户权威资料均对两组开放**。

**重要公平性**：A、B 的规范 Gate/Review/QA、Owner 的澄清答复、合规停机条件相同；B 可以在进入下一阶段后保留前阶段知识，因此这是“逐步新增”，并非删掉旧知识。两组在同一阶段都有相同的进度/工具预算与修复机会，但 B 自身使用知识的 Token/检索/工具成本计入总成本，不能只算它的修复时间。

## 四、按 ADS 走真实流程，而不是模拟打卡

\`\`\`text
Intake + real GitHub Issue/H0 + project profile
    ↓
S1 L1 Requirement Evidence → PRD/scope + approved clarification → Product owner freeze
    ↓
S2 L2 Design Evidence → invariant/ADR/spike only if material UNKNOWN → L2 freeze
    ↓
S3 L3 Task DAG-I/Concern write sets + Review Policy/actual dependencies
    ↓
S4 independent concern implementations + syntax/type/affected tests
    ↓
S5 integrate candidate → actual selected test discovery/execution
   → required validation + independently selected review + hidden QA
    ↓
S6 evidence-limited closure + observed comparison + improvement candidates
\`\`\`

只有真实 Issue/项目合同授权的 Actor 才可以产生正式 Freeze、Claim、Review/Release。实验者不能用一个文本 \`PRODUCT_FROZEN=TRUE\` 冒充真正 Owner 事件：在纯本地测试环境如果未真实承载相应 ADS Authority，明确写 **ADS-CONFORMANT-STAGE_SIMULATION / NO_FORMAL_OWNER_FREEZE**；不能声称正式 ADS 生命周期已经通过。**发布 Release 不属于本轮实验验收**；最多报告候选准备和“如果发布还缺哪些真实 Gate”。项目 CI 若已有必需 Runner/Required Check，则真实执行且两组一致；若本地等价校验在当前执行环境充分，依 v4.19.0 \`VALIDATION_STANDARD\` 的 local-equivalent-first 原则不重复 CI；缺的不可替代 Host/CI Gate 如实记 NOT_RUN/BLOCKED，不假 PASS。

若本任务已经满足显式 DAG-I 的实质条件，可以为 Contract→Persistence→Core→CLI/Integration 建立受权 DAG；L1/L2 是否创建独立 DAG-P/DAG-A 要根据真实不确定性与并行材料性判断，不以 Draft #1384 强制让两组生成相同数量的 Agent/Issue。

## 五、真实独立 Oracle：在两个 Agent 开始之前冻结

由独立 QA 准备至少下列类别的**同一份隐藏验收套件**，保密到每组 candidate 首次提交；允许的公开合同由上面 FR 定义，隐藏测试不得创造新需求：

- O1 正确合法请求、稳定输出与身份：相同 \`projectId/requestKey/digest\` 幂等，同键异摘要冲突；
- O2 两个项目同 key 互不串读、跨项目查找/取消被拒；
- O3 同一项目并发双提交/双 Worker 交错，结果不多次生成；
- O4 在若干持久化边界注入崩溃后重启，不能丢失有权已提交结果或重复输出，\`UNKNOWN\` 按合同保留；
- O5 取消/完成/失败的非法状态转移和恢复权限；
- O6 旧格式/损坏检查点安全停止，不将损坏静默洗成成功；
- O7 CLI 错误诊断完整、保密，不输出 secrets/原数据；必要测试真实选中、执行和断言；
- O8 对每个 candidate H1 的 Source SHA、实际 Test Discovery/Runner/Assertions、受影响 Consumer、依赖/Artifact 证据执行限定核查。

**Hidden Oracle 不应在 L1/L2 的中途泄漏**；公开的验收规则可以给两组，但 QA 私有测试实现/结果只在公平的验证检查点由独立 QA 消费。若失败允许一个实验预算内的等量有界修复窗口，收据保留原 FAIL、修复变更与新 Evidence；不得无限迭代到两组都 PASS 后宣布平局。实验的固定停止预算不是“ADS 对所有项目设统一重试次数”的规范。

## 六、正确衡量四类结果，不只比较 PASS

| 比较层 | 可核对指标 | 避免的假收益 |
| --- | --- | --- |
| L1 / Requirement | 缺失/矛盾条件检出、需求→验收 Trace、Owner 澄清质量、阶段返工 | 把 B 多写的文字当需求正确性 |
| L2 / Architecture | 明确状态 Owner、持久化/原子性、恢复/权限不变量、可证伪选择与材料风险 | 把复杂微服务选型当架构能力 |
| L3 / Implementation & QA | FR 真正通过数、目标真实运行 Test IDs、隐藏负例、严重缺陷、风险误放行、独立 Review | Agent 自述 PASS、作业 EXIT0、旧 SHA Review 冒充当前合格 |
| 完整过程与成本 | 所有 Attempts、发现/修复失败、模型/工具调用、Token、墙钟、人工澄清/Review/Repair、CI/Host、有效 L1→L3 Trace | 仅统计成功任务、免费算 B 的知识检索成本 |

同时保留负收益：知识引用错误、引用旧版标准、过度 STOP、答案污染、阶段提前给知识、知识片段不适用、强制无谓流程、错误归因。对未运行/没有资质的真实检验标 NOT_RUN/NOT_MEASURED。一个复杂任务即使 B 更好也只能得到**个案机制证据**；跨领域收益至少继续选其他独立题材和多次/配对任务而非复用同一公开题本。

## 七、可执行交接前的必备核对与明确停止点

运行前需固定：\`EXPERIMENT_REPO\`、\`EXPERIMENT_ISSUE\`、\`BASE_SHA H0\`、真实环境 Node/Package lock、公开受权 FR/NFR、两组独立初始 worktree 和 Agent 版本、ADS 4.19.0 具体标准版本、Corpus Commit/Blob 清单、两组相同的 Owner Q&A、独立 QA 预冻结的 tests/expected + 权限/写集、保密防泄漏规则及计费/消耗采集。历史 R1 本地报告如存在，应先读回原始收据而不能从用户的“完成”虚推实验 PASS。

**无上述材料时 R12 仍为设计，不交付真实 Agent 触发词来猜 Repo/H0/Issue。** 实验一旦启动，严格有限：一个任务、A/B 各一条隔离执行、按 S0–S6 明确结束，固定修复窗口后最终审查，存档证据即停。独立 Reviewer 的知识库文档范围接受不属于真实实验结果；实际 GitHub Task/Release 权限仍依已生效 ADS。

**R12 当前版本物**：任务需求与复杂度约束、ADS Stage Gate、分阶段 Kxx 注入矩阵、隐藏 Oracle 与 A/B 评分和停止条件。当前 **REAL_R2_AGENT_RUNS=0**；R10 已通过的 \`165 syllabus / 120 A /72 reference leads /48 conceptual edges /18 Q /22 W\` 维持不变，本文只增加一个候选版本级设计文档。
