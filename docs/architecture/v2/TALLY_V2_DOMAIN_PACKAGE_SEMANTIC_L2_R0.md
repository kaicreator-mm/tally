# Tally v2 — Domain Package First 语义 L2 候选 R0

**状态**：L2_SEMANTIC_DRAFT_R0 / PRODUCT_152_USER_SCOPE_CONFIRMED_R7_PENDING / ADS_418_LOCAL_ADOPTION_PENDING / DH_0102_RELEASE_CONDITIONAL / L2_FREEZE_READY=NO / VERSION_STAGE_EXPERIMENTS_ONLY  
**归属**：[L2 #65](https://github.com/kaicreator-mm/tally/issues/65)、[语义 L2 交付 #78](https://github.com/kaicreator-mm/tally/issues/78)。Product R0 见 [#76](https://github.com/kaicreator-mm/tally/issues/76)、[PR #77](https://github.com/kaicreator-mm/tally/pull/77)。  
**历史功能证据**：v1.5.5 固定 tag commit b5e7b6e38ea38837e3bbcfc46c36586180705de0；[47 项矩阵](../../research/v2/TALLY_V2_FUNCTIONAL_PARITY_R0.md)。  
**平台设计参考**：DomainHarness v0.10.2 L2 Candidate 文档 blob 68e7dbc591f4c0c694818ec574b8585f95239c8b；引用的是**设计方案而非可安装 SDK/物理版本验收**。历史平台 T09 #1330 已完成，v0.10.2 [R01 最终回执](https://github.com/kaicreator-mm/domain-harness/issues/1331#issuecomment-6416079327)已 CLOSED，tag `cee14399d9d0af4e20e34b9111861b16a465b690` 与 main 同树；**RQ=CONDITIONAL**（Hidden 7/7 PASS；Full EXIT1、root NOT_RUN、单次特准），不等于 Tally 当前 Host 的 Release 验收。  
**底线**：不将普通 TS App 写好后包装成 Domain Package。先建立真实的第三方 Business/Capability Domain Package 产物及运行依赖，再编码 Tally 实现。本文件不发布任何新的 DomainHarness Manifest、npm 包名、公共 TypeScript ABI 或 Kernel 代码。

**Product candidate source sync (not Freeze):** Historical parent Product PR #77 at `0d1f1a2e15581e300500bada6da54b2c9bcef75f` has been superseded for v2 design intent by [Product PR #152](https://github.com/kaicreator-mm/tally/pull/152) at `bdf632fa3572074eef0654af99cb52df24eb62e9`: **24 P0 requirements, F01–F47, V01–V82**, including P0-18..24 and V68..82 full Agent-native ALM. User confirmed the 47-feature **19 RETAIN / 23 ADAPT / 5 RETIRE** migration decision on 2026-10-28, but exact-HEAD independent Product **R7 is pending** and no Product Freeze was published. See the updated [R2→R7 provisional source crosswalk](./TALLY_V2_PRODUCT_R2_TO_L2_GAP_CROSSWALK.md). **当前目标为 ADS v4.18.0**，上游 source `2c395c55569fe2edf3f4135f12d3d069e526d0b7` 已 RQ READY；Tally 项目采纳 Draft PR #85 在本 L2 候选时仍缺最新 exact-head verifier 与合格独立 review。旧 #81 v4.17 CONDITIONAL 问题是历史记录，不再是本版本选择障碍。 The scoped P0-15/16 credential confinement, P0-17 Local OS Verifier and L3/release boundaries are mapped in [R2 crosswalk](./TALLY_V2_PRODUCT_R2_TO_L2_GAP_CROSSWALK.md), and incorporated as required semantic negative contracts below. This branch sync is NOT independent Review or physical ABI verification.

## 1. 物理架构硬约束

1. **只有一个 Kernel 主权**。DomainHarness K Primary 是本 Domain App 内 State/CAS/Admission/Journal/Effect/Replay/UNKNOWN 的唯一来源。Tally 不能开发自己的并行 Task Effect 引擎、状态提交仲裁器、独立 replay journal。
2. **Tally 业务逻辑是 Business Domain Package**。Project/Version/Requirement/PRD/L2/DAG/Task/Review/RQ 等工程规则归 Engineering B，不能下沉为所有 Domain App 都被迫使用的 K 或通用 S。
3. **选用已交付的 S**。Rule/Context/Decision/Workflow/Evidence/Shadow 只按实际已发布、BUILD 可选并验证的 Feature 使用；业务策略不写入通用 SDK 判定硬编码。
4. **C 只提供能力**。GitHub、Local Agent、Host/Tool/Model、可选 SSH/Docker/runx 的连接器暴露源或执行能力；工程任务完成/冻结/RQ 的判断由 B 的合同和真实外部权威确定。
5. **UI 是 Domain App 投影**。UI 表单和用户操作必须调用真实的受控 Business Intent/Command，涉及 GitHub/Host 等业务外部效果必须通过受 K 授权的 C。UI 本身不掌握第二份 Tasks 真相。
6. **BUILD 是部署边界**。Manifest、选中 Package、版本/摘要/闭包在 qualified BUILD 时确定；运行中的 occurrence 固定选中 Package pin，不允许在运行时将提议的 Rule/Skill/Candidate 热替换进生产。
7. **Local 是必要能力**。即使完全没有 runx/远程 SSH/Docker，Tally v2 仍要在 Node24 已验证 Host 和 Local Agent 上完成真实单 Task 与版本闭环。
8. **task-meta 只负责旁路观测**。Tally 不实现 task-meta 私有 C adapter；只读取其合法发布到 GitHub 的 ADS 报告。collector、队列、Hook、GitHub telemetry report 延迟不触发 Task 停止或验证阻塞。
9. **GitHub-native 事实与 K State 不能混淆**。Issue/PR/Commit/Review/Check/Tag/ADS published record 的事实来自 GitHub。K 持有 Tally 应用状态、效果决策与回执；SQLite 只能作为经限定的 GitHub 派生缓存或者实际 K Host 持久化接口的一部分。
10. **没有已验证 ABI 时，不写生产实现**。尚未公开、无法 clean install、必须 deep-import 的 SDK 或示例，不是 Tally 新版生产依赖。

## 2. 所有权与事实主权矩阵

| 对象/动作 | 决策 Owner | 外部真相 / 存储 Owner | 执行边界 | 失败/未知处理 |
|---|---|---|---|---|
| Requirements/PRD 修订与审核建议 | Tally Product B | B/K 版本化意图 + 对应 GitHub 合法工件 | App Shell→认证 Host→B/K；GitHub 写入 K→C | 缺 operator/project/version/source link ⇒ DRAFT，不标 FROZEN |
| Operator / UI session / repo resource grant / CSRF / secrets (P0-15) | Host/App Shell 验证 operator、session expiry、Origin/CSRF；Engineering B 鉴权 repo/resource & protected paths；唯一 K Admission，C provider grant 独立验证；Host/C 屏蔽所有日志/回执中的 secret | Host session 与 GitHub 原生 provider scope/当前授权、K Effect、GitHub native readback | **四重许可均必须有效**：Host identity/session/origin、B resource ACL、K permit、C provider permission；Credential redaction before logs | Cross-origin、过期 session、跨仓库越权、protected edit 或日志泄露 token ⇒ DENY/no Effect，不能靠 JSON 或 K-only permit 绕过 |
| Protected-path Repair, independently qualified verifier & reviewer (P0-07) | Assurance B 管理 Repair 预算、保护路径和 Reviewer 资格；K 许可合法 Effect；C Local/GitHub 读取真实 diff、测试与独立审查回执 | Git candidate SHA/diff/protected config, real validator command/exit and source-qualified independent reviewer | Builder 不能更改受保护 ADS/auth/hidden-test 资源而自批准，失败的 verifier 不得被状态转换覆盖 | Protected edit / fake success / self-review / verifier FAIL ⇒ BLOCK |
| L2 设计修订、独立审核要求 | Tally Lifecycle B | B/K + GitHub Issue/PR/review | B 提议，K 准入，C GitHub 执行 | reviewer 缺失/过时 BLOCK |
| Task DAG 提案、依赖/Ready 规则 | Tally Task B | B/K Proposal；native GitHub Issues/Dependencies 以 GitHub 为准 | B 的 DAG 判断→K/C 合法生效 | 环、stale parent、Native Graph 不完整不可 READY |
| Agent Registry/Role/Provider 选择 | Tally Task B + self-declared logical role / current capability and scoped delegation | K 受控配置/会话；外部 Agent 真实性/Host session 可观测；GitHub transport 账号不代表逻辑身份 | B 选择并限制授权委托，S advisory、K/C 真实验证 scope | Stale Claim、委托越界或 Builder 换 reviewer 角色自签 PASS ⇒ BLOCK |
| Task State / business occurrence 状态 | Tally Task B 解释规则 | **唯一 K Primary State/CAS/Journal** | K 原子准入/转移 | 冲突 typed；掉线 UNKNOWN |
| 外部 Tool/Agent/Local/Host 调用 | B 请求意图，S 可给建议 | K Effect + 外部 Provider/Host 最终回执 | K Effect permit → C Local 等 | accepted ≠ terminal；丢 ACK reconcile |
| GitHub Issue/PR/Review/CI/Tag | GitHub 真实权限决定 | **GitHub Native** | K permit → C GitHub；write/readback | 未读回不能称已发布/已合并 |
| Cross-session durable report/decision/review/dispatch (P0-14) | Engineering Lifecycle B 生成绑定 source SHA 的工件与 handoff；K 治理发布，C GitHub 回读 | 实际 GitHub Issue/PR/comment/commit/revision/Source Evidence，**不是当前聊天或 SQLite 草稿** | 所有实质决策、报告、Review、Dispatch 必须持久发表并记录可恢复的 exact ref/version | 只存在会话文本、未成功写回/无法回读 ⇒ NOT_PERSISTED/UNKNOWN，禁止宣称完成 |
| 验证契约、Repair、RQ 决策 | Verification B；S Rule/Evidence advisory | GitHub、Host、独立 Reviewer、实测命令证据 | K 受控 Effect；B 判门禁 | 没跑=NOT_RUN；PR PASS ≠ RQ PASS |
| task-meta 观测与成本 | task-meta 独立采集 | 只有 ADS-合格的 **GitHub 发布摘要**供 Tally 读取 | C GitHub read-only；B 派生经验 | 未观测=UNKNOWN；不影响 Task |
| SQLite Github Projection | B 查询规则/C GitHub Reader | GitHub 原始源仍主权；SQLite 是**派生视图** | Host/缓存层按源版本索引 | 过时/覆盖不全 → STALE / PARTIAL |
| SQLite K 状态持久化 | K Primary | DomainHarness 选中的 Host 持久化 | 真实 K Host API | Tally 无直接绕开 K 的业务 State 写入 |
| Candidate/Policy 提议与激活 | Tally Experience B，S 测评 | GitHub 经验事实 + Qualified BUILD | proposal→独立验证→新 BUILD | Shadow 不发布生产 Effect；旧 pin 不变 |

注意：B 的 Project Revision/Task Contract 是“应用语义”，并不自动在 GitHub 自然存在；用户看到它们时必须有具体的 artifact/authority 标记。GitHub 中 Task/PR/RQ 等事实不能由 K 的“提议状态”替代。

## 3. 逻辑 Engineering B 模块（不是已经决定的四个 npm 包）

| 逻辑模块 | 拥有的业务责任 | v1.5.5 对应功能 | 与 K/S/C 关系 |
|---|---|---|---|
| B-ProductLifecycle | Project Version、Product Evidence、RequirementsRevision、PRD/L2 Review/Freeze、适用 ADS pin | F01–F05、F09、F11、F12、F42 | K 存应用状态；S 提案/规则；C GitHub 工件 |
| B-TaskOrchestration | DAG 与依赖、TaskContract、Agent Profile、ready/blocked、Replan、重试预算、Local 优先 | F06–F07、F13–F20、F27、F30 | B 决策；K effect；C GitHub/Local/可选 runx |
| B-AssuranceRelease | Candidate、ChangeSet、Verification Contract、held-out、Review、Repair、Version Closure/RQ | F08–F10、F21–F29、F47 | S Evidence/Workflow；K permits；C 真验证者/GitHub |
| B-ExperienceDecision | Source-qualified Context、Failure Diagnosis、经验候选、Strategy/Cost 派生视图 | F31–F39、F41 | S Context/Decision 可选；C GitHub report；数据不足弃权 |
| **F40 cross-layer persistence / derived projection (NOT an Engineering B Package)** | **K Primary + Host** own authoritative State/CAS/Journal/Effect durability through selected public Host persistence interface; **B query / C GitHub Reader + Host cache** own source-SHA-qualified GitHub-derived SQLite/Postgres read projection only | **F40** SQLite / Postgres 存储接口 | K/Host writes alone may persist K-owned state; GitHub Issue/PR/Review/Task Report stay native authority; derived cache staleness/partial pagination never upgrades eligibility; **no second Tally K/Journal or authority** |
| App Shell/UI | 阶段向导、表单、Review/Verify/Release 视图 | F14、F42–F46 | UI 仅发送受控意图，不直接构造 Kernel state/effect |

**F40 所有权追踪：**上表 F40 是跨层 Host/Storage+GitHub projection 责任，不是额外的 B npm 包；与 §2 的 `SQLite Github Projection` / `SQLite K 状态持久化` 以及 §7 两类存储一致。原 Go 存储后端能力不会成为 v2 绕 K 更新业务状态的理由。

**L2 待决：**一个 Engineering B Package 还是多个 B Package、是否独立生产 C GitHub / C Local、UI 以何种 public Application Shell composition 加载，完全取决于 DomainHarness 最终可安装的 Producer/Manifest/ABI/BUILD 支持。不以“逻辑模块=物理包”作为默认假设。即便最终只有一个 B 包，仍须明确对外合同和权限最小化。

## 4. Product → L2 → Task → Release 数据对象及身份

仅定义业务**语义不变量**，不是发布的 Public TS interface：

| 语义对象 | 必备身份、时间线 / 来源 | 禁止歧义 |
|---|---|---|
| ProjectVersion | GitHub owner/repo + version id + ADS exact adopted revision + B revision | 不以活动的 main HEAD 假充固定计划的源 |
| RequirementRevision | requirement id, parent version, content/source digest, proposer, review evidence | Draft ≠ approved |
| ArchitectureRevision | L2 revision id, source SHA, selected product baseline, review verdict | 设计提案 ≠物理 Package ABI 已验证 |
| TaskDAGProposal | DAG revision, node id, dependency IDs, expected previous revision, cycle proof | 不将本地 DAG 边缘当 GitHub 真实依赖 |
| TaskContract / Claim | GitHub Issue ref, candidate SHA, exact Task scope/acceptance, actor declared id/role, attempt id | GitHub 账号不等于 Agent 身份；Claim 不自批准 |
| TaskRun / Attempt | run id, attempt id, source and selected BUILD, governed occurrence ref, provider session ref | 子 Agent Run 与父 Task 不共用模糊幂等身份 |
| EffectReceipt | effect id, intent digest, idempotency key, permit, accepted, terminal/UNKNOWN, source revision | accepted≠terminal；UNKNOWN 不直接 Retry 危险 Effect |
| VerificationEvidence | validator identity, source SHA, test scope, exit status, real environment, timestamps, coverage | Agent 文本声称 PASS ≠ 独立验证 |
| ADS ReportObservation | ADS dialect, GitHub comment/id/edit revision, subject/attempt scope, reporter, coverage, metric basis | task-meta 私有 Manifest 不是 Tally 读取前提 |
| ReleaseCandidate / RQ | Candidate exact SHA, full/hidden/CJ/packaging evidence, independent RQ verdict, Tag ref | PR 合并不能自动满足版本 Closure |

最少识别四种未决维度：
- 一项 Task 未完成：受工程 Gate/Claim/Verifier 决定，而**不是**遥测 Coverage；
- 采集不可用：Telemetry UNKNOWN，不推导 Cost=0；
- GitHub 不可回读：Source UNKNOWN，阻止需要其事实授权的 Stage；
- 外部 K Effect 结果丢失：Effect UNKNOWN，按 K 外部回执重新协调，不允许 blind retry。

## 5. 受控调用序列（业务语义，不是已实现代码）

### 5.1 只读同步与 UI 呈现

1. C GitHub 读取 pinned repo/Issue/PR/CI/Review/ADS Report；每条附 source identity、版本、分页覆盖、时间/ETag 等。
2. Source Projection 缓存将数据作为 Derived Data 索引；每次显示标记 COMPLETE/PARTIAL/STALE/UNKNOWN。缓存写入本身不代表 GitHub 事实变更。
3. B 将 GitHub facts 与唯一 K-owned local app state 组合成 Project/Task/Release View。
4. UI 只展示 source-qualified Read Model，若状态需 GitHub readback 后才能确认，UI 不显示“已完成”。

### 5.2 从 UI 提交 Product / L2 / Task DAG

1. UI 发送用户意图和 expected source revision 给 B；
2. B 进行规则、审批条件和依赖检查，产生受控 domain command；
3. K 以选定 BUILD/主体的实际能力执行 Admission/State/Effect，并使用所选 C GitHub 写真实外部工件；
4. C GitHub 记录远程 accepted/终态/可能 UNKNOWN，进行合法 readback；
5. B 依据 K 回执 + GitHub 真实权威更新投影；必要审核缺失保持 REVIEW_PENDING；
6. UI 返回真实下一 Gate 和 missing requirements，不把本地 ACK 当 GitHub success。

### 5.3 真实 Local Task

1. B 按 DAG 当前事实和已冻结 TaskContract、Agent 声明能力、隔离要求选出 Local Agent；
2. K Admission 验证 Task/Claim/currentness/Effect intent；S Context/Decision 是建议而非 Effect 权限；
3. 选定 C Local/Agent 通过实际 Host 启动与监测；accepted 不代表 verified terminal；
4. Hook/collector 可以旁路记录；task-meta 完全掉线依然继续正常 Task 和独立验证；
5. B 收到真实 terminal/evidence，K State 和 GitHub PR/Review 更新分别按权威规则处理；
6. B 根据 Full/Hidden/CI/Review/Release predicates 判断 Task 或版本可否闭合。

### 5.4 被丢弃或重试的 Effect

- 同一 idempotency key 与相同 intent 可由 K 按真实合同 replay/reconcile；
- 相同 key 与不同 input 必须拒绝并留证；
- 失联 ACK → UNKNOWN、查真实外部收据；在“是否发生副作用”仍不明确时不得 blind 重新执行；
- 需人工权威时显式停在 WAIT_AUTHORITY；“Agent 再试一次”不代替权限。

## 6. C 能力边界与安全

### C GitHub（必需）

- Read: repository revision, native Issue/Dependencies/PR/Review/CI/Tag, supported ADS report type/dialect, source-corrected history, pagination/403/429/ETag semantics.
- Write: authorised Issue/PR/Review/Report feedback/tag actions as permitted by actual GitHub grants **and** K-admitted effect, tracked idempotency/expected revision/readback.
- Tally **仅经 GitHub** 读取 task-meta 已发布摘要；无直连 task-meta API/SDK/DB/raw Trace/manifest。
- 禁止：从 comment 文本中的“PASS”自动推断 Host/CI/Reviewer 真结果；旧报表格式静默降级；超出目标 repo 权限。

### C Local / Agent（最低必需）

**Two independent Local rejection identities (Product P0-06, V52/V53/V66):** `LOCAL_AGENT_UNAVAILABLE` means no real installed/usable/authenticated Codex CLI (or only a Fake/echo test provider); the dispatch is not admitted regardless of Builder OS isolation. `BUILDER_CONFINEMENT_UNAVAILABLE` means a *real* Provider is present and authenticated but the least-privileged Builder principal, its credential-store separation, or no-write denial cannot be proved; the dispatch is still not admitted. Both cases keep ordinary authorized Product/Requirements browsing and draft editing usable. Neither status means Hidden-verifier absence: `HIDDEN_VERIFIER_UNAVAILABLE` independently blocks hidden qualification. Typed status must be retained in B projections, K Admission failure receipts, Host/C provider failure mapping and UI; never substitute `NOT_QUALIFIED` for the reason.



- Execution Environment=Local，Workspace 与 Agent Runtime/Provider 分离；**第一阶段**先完成一个真实 Agent/Task/Verify 的物理纵向闭环（可串行）；**完整 v2 Product P0-20/V63/V72** 至少要有 **两个**独立授权、互不冲突写集、各有 native CAS Claim 和 K/Host Permit 的 Builder Task Attempts **真实同时执行**，再由独立 Reviewer Join。第二个执行请求若与已有写集/授权主体冲突必须拒绝或串行（V56/V73）；不能只展示 parallel-ready 集合作为并行 PASS。先前 max N=1 是已被 Product #152 取代的历史第一阶段约束，绝非最终 v2 P0。所有实现及并行 Task DAG 仍须 Product R7、L2 独立冻结和真实 Host 证据。
- **Codex Provider/Auth partitioning:** Operator 在 Host/Provider adapter 配置中合法安装、登录真实 Codex CLI；Host 将**只可用于模型 Provider** 的最小授权会话/认证句柄限定传给低权限 Builder principal，**绝不继承**操作者可写 GitHub 的 gh/GCM/SSH-agent、home 配置、token cache 或提权能力。模型登录成功不证明 GitHub grant，也不成为写权限。若 Codex 授权文件/目录同 GitHub write credentials 无法实证隔离，返回 BUILDER_CONFINEMENT_UNAVAILABLE，Local Task 不可 PASS；禁止把真实秘密写入可发布的日志/证据。
- 关键边界：OS 进程权限、环境变量脱敏、工作目录/命令参数隔离、Git 工作树安全、受控 Tool capability、进程退出/超时/错误码/标准输出留证。
- SSH/Docker/runx **可选**；没有时不可妨碍 Local 工作和 ADS UI。
- “每个 runx 实例的 SSH Server”属于可选远程 Host 提供者，不下沉为 Tally 核心 K 需求。

### C Model/Intelligence（可选）

- LLM/AI Runtime 共享消费者定义的 proposal/advice contract；所有“拒绝/升级/额外证据”是 advisory，只有 B/K/GitHub 真证据改变状态。
- 缺 Model 不应阻止手工 Requirements、Review 和已有资格的非 LLM 基础 UI；需要 Agent 推理的特定任务标明确切 blocked reason。

## 7. SQLite：两类物理数据，两个边界

| 用途 | 允许状态 | 禁止 |
|---|---|---|
| GitHub-derived read cache | version-qualified entity key/source SHA/edit revision/ETag/fetch coverage/watermark/staleness, searchable indexes, bounded refresh | 将缓存当 GitHub 原生 Issue/PR/Review/Tag 主权；不刷新直接解除 Gate |
| K/Host-controlled app state | Host 实现的 K 状态/CAS/Journal/Effect recovery，经实际 K public API | Tally 自行写第二个业务 Journal、直接更新“已完成”工程事实 |
| Mutation reconciliation queue | 仅按 K/Host 已证明的 durable effect/outbox 模型存 idempotency/remote UNKNOWN/reconciliation references | 另建绕过 K 的独立外部 Effect 重试引擎；缓存 ACK 就报 SUCCESS |

L2 物理问题留待 DH：Host SQLite 暴露的范围、K receipt/outbox 是否满足跨 C GitHub 的 reconcile、按 source-key invalidation 的落点、Host API 导入稳定性。此文件不创建 sqlite schema 或技术包名称。

## 8. task-meta 降级一致性

**正常数据流：** Agent/Host → task-meta 独立观测 → ADS GitHub bounded typed Report → Tally C GitHub read → B Experience/Cost/Context projection。

**Down/partial：** task-meta Agent Hook 抛错/Collector down、重试满、队列满、发布超时、GitHub 报告迟到：Task 启动/执行/真实验证的正当流程保持原结果；观测 Coverage=UNAVAILABLE/PARTIAL/UNKNOWN。不能从“没有报表”推断 Task 没跑，也不能从 task-meta 成功发布反推 Task 工程验证已通过。

**权限区别：** GitHub Issue/PR/Review 等原生权威无法获取时，相应工程门禁必须显式 WAIT/UNKNOWN；只把**旁路 Telemetry Publisher 故障**定义为非阻断，不扩大成所有 GitHub 不可用都能 RQ PASS。

## 9. ADS UI command mapping（全部是未来业务意图）

| UI 屏幕 | B Intent | 需要的独立事实 | 需要 K/C 真执行吗？ |
|---|---|---|---|
| Product/PRD | Propose/ReviseRequirement, SubmitProductReview | RequirementsRevision/Review evidence | 发布/状态变更时是 |
| Product Freeze | RequestFreeze/RecordReviewDecision | Reviewer + exact current source | 是 |
| L2 | ProposeArchitecture/RequestL2Review | Product eligible + L2 source review | 是 |
| Task DAG | ProposeDAG/CheckCycles/PublishTaskIssues | Native GitHub Issue graph/readback | 仅纯预测不用，发布要 |
| Execute | SelectAgent/RequestDispatch/Cancel/Reconcile | Claim, target SHA, Agent/Host status | 是 |
| Repair/Review | RequestRepair/RequestIndependentReview | Candidate/Verifier exact proof | 是 |
| Closure/RQ | ProposeRQ/RequestTag/PublishBaseline | Full/Hidden/CJ, GitHub native refs | 是 |
| Learning | ViewCoverage/ProposeExperienceFeedback | GitHub-qualified ADS report | Read-only 可显示；反馈写需 K/C |

所有业务意图仅是概念命名，**不是正式 TS 方法名**。实际 public call/operation schema 与 Module/Package export 要随 DH 公开 ABI 重新对账。

## 10. 关键负例与 L2 评审重点

| L2 case | 触发 | 必须拒绝/保守处置 |
|---|---|---|
| L2-N01 | 假造一个 TallyKernel 或 SQLite taskStateStore 自行提交完成 | BLOCK / no second K |
| L2-N02 | UI 调 GitHub 写入没有 K permit | BLOCK / no external effect |
| L2-N03 | GitHub cache 旧于真实 PR SHA | REFRESH / STALE, not release eligible |
| L2-N04 | 第三方独立 Business Package 必须 deep-import DH src | NOT_QUALIFIED physical ABI |
| L2-N05 | BUILD 编译器进入已安装 Tiny Runtime closure | Reject against actual public Package contract |
| L2-N06 | 任务仅有 Agent 自称 PASS，Verifier FAIL | Task BLOCK |
| L2-N07 | task-meta 采集退出但 Task/Verifier 本身健康 | Task result unchanged, Coverage UNKNOWN |
| L2-N08 | GitHub Issue Native 403，task-meta 正常 | Native authorization WAIT/UNKNOWN |
| L2-N09 | Effect accepted，terminal ACK 丢失 | K reconciliation/UNKNOWN; never blind retry |
| L2-N10 | Provider runx 不存在 | Local Task still eligible |
| L2-N11 | Shadow candidate 试图写生产 GitHub/Kernel State | BLOCK; no prod Effect |
| L2-N12 | 运行中 occurrence 改选 package | BLOCK; new qualified BUILD/occurrence only |
| L2-N13 | 缺 Product/L2 独立审查直接生成 Task DAG | NOT_ADMITTED |
| L2-N14 | PR 局部 CI PASS → Release 自动 Tag | BLOCK; no Release PASS |
| L2-N15 | 私有 task-meta Trace/manifest ref 在 GitHub comment 中 | Tally 不 dereference 私有指针 |
| L2-N16 | 不兼容 ADS Task Report 在 Reader 中混作当前版本 | UNSUPPORTED/INSUFFICIENT_EVIDENCE |
| L2-N17 | Builder 经 gh/GCM/SSH/sudo 获得 GitHub write 权限 direct push | UNGOVERNED_EFFECT/BLOCK, 仅 K→C 发布 |
| L2-N18 | Builder sudo/admin/backup 读取 Hidden fixture | HIDDEN_NOT_QUALIFIED |
| L2-N19 | 真实 Agent 已安装认证，但 Builder credential confinement 无法实证；或独立 verifier OS ACL 缺失 | BUILDER_CONFINEMENT_UNAVAILABLE / HIDDEN_VERIFIER_UNAVAILABLE，按不同提供方保留准确类型，不得隐性 PASS |
| L2-N20 | 已有 GitHub Issue/Ready，但 L3 Tests/Contract/Implementation/Failure/Reference 尚未准入 | L3_NOT_ADMITTED |
| L2-N21 | 把 ADS v4.17 CONDITIONAL RQ 直接当 READY 激活项目 | STANDARD_PIN_PENDING，冻结拒绝 |
| L2-N22 | task-meta hook hang/nonzero 中断 Agent/Verifier | fail-open，Telemetry UNKNOWN，Task 实际结果不变 |
| L2-N23 | 跨源 Origin/CSRF、expired session、跨 repo grant 被 UI 当有权指令执行 | Host/B/K/C 至少一重 DENY，无外部 Effect |
| L2-N24 | Repair 修改 protected ADS/hidden/auth 文件或 self-approve 失败验证 | Protected diff/independent verification BLOCK |
| L2-N25 | 新会话仅依赖聊天记录声称 Product Frozen/Task PASS | NOT_PERSISTED/UNKNOWN，必须 GitHub source-qualified readback |
| L2-N26 | Agent 以别的 logical role 逃避 Claim/独立 Reviewer 权限 | B delegated scope + K/C actual rights fail closed |
| L2-N27 | 两个不同 worktree 的 Task Attempt 同时 EXECUTING | 第 2 个 K Admission NOT_ADMITTED，P0 max=1 |
| L2-N28 | Codex 模型登录令牌目录同时暴露 operator 的 GitHub write token | BUILDER_CONFINEMENT_UNAVAILABLE，Provider/GitHub credential partition |
| L2-N29 | GitHub readback/Agent stderr 回显秘密令牌或受保护凭证 | Host/C redaction，安全错误回执，不得发布明文 |
| L2-N30 | 仅有 Fake/echo、真实 Codex CLI 未安装、未认证或能力不可用 | LOCAL_AGENT_UNAVAILABLE，禁止 CJ03/Task PASS；普通安全读写 UI 仍可用 |

所有案例现为 **SPEC_ONLY / NOT_RUN**；安装测试和真实消息/Effect 调用不得自行判通过。

## 11. 不随意冻结的物理设计清单

| 未决项 | 依赖提供者 | 入场可接受的证据 |
|---|---|---|
| Business Domain Package producer 的 official public import/export | DH Contracts/BUILD | 干净第三方 producer 打包/安装无私有 import |
| B/S/K/C Manifest、package type、ABI、selected closure | DH v0.10.2 | 官方 pack artifacts/digest + independent conformance |
| Runtime Microkernel Facade/Host seam 接线 | DH T04/T03 | Node24 physical E2E, K permit/cancellation/replay |
| C GitHub / Local 的受控 Effect 传输方式 | DH Public Host/SDK + Tally L2 | exact selected K + real C complete/UNKNOWN/denial |
| SQLite cache 与 K Host 状态隔离 | DH Host Node + Tally design | crash recovery + stale-cache denied mutation negatives |
| User-facing App Shell / UI Package loading contract | DH public Application entrypoint | installed consumer App shell with selected B/C and no deep import |
| subscribe/stop/governed event consumer | DH T06 | selected public Facade event coverage without stale exposure |
| 正式 ADS GitHub Task Report dialect | Tally project ADS adoption + ADS release | one immutable qualified adopted revision + actual report/readback |
| Release RQ & Tag | DH R01 + Tally RQ | exact SHA Full/Hidden/CJ/Pack/Independent review |

**不做**：从 DH 私有 compiler 源反推不存在的公共 API。当前已发布 v0.10.2 public SDK 应通过版本锁定的真实 packed/Node24 调用使用；其发布 RQ CONDITIONAL 与 Tally v2 实测/RQ 仍独立。

## 12. 交付与 Gate 状态

- 本文件可由 Web 独立完成、审查并修订；依赖 Product R0 和实际 DH 物理实现的事项**全部明确待定**。
- **语义 L2 R0 完成的标准**：全部权威边界/业务数据/外部失败/正反例/物理开放项均有可追踪记录；不是 physical ABI freeze。
- **正式 L2 Freeze**：先 Product Review/Frozen，再选定合法 ADS pin，对 DomainHarness 真实独立 Package Manifest/ABI/Host/BUILD/Artifact 作独立证据审查，在新 exact-SHA 上闭合未决项。
- **Executable Task DAG**：须在正式 Product/L2 冻结和 Domain Package Ready 后生成，非此文件中的表格即“已准入 DAG”。
- **v2 源码/测试进度**：NOT_STARTED / NOT_RUN；任何历史 Go v1.5.5 RQ 不转化为本版 Tally v2 PASS。

下一步是独立 L2 语义审查，对业务所有权和失败合同给 P0/P1/P2，再等待 DH 真实发布接口填补物理字段，最后才入正式 L2 Freeze。

## 13. Current Product-v2 semantic security and source-adoption binding (candidate only)

- **Builder credential isolation:** Real Codex CLI is launched under a least-privileged Builder OS principal with *no* GitHub write credential/credential helper/GCM/SSH-agent/delegated write capability. Only K-permitted C GitHub has scoped native write authority, expected source SHA and remote readback. Any write by Builder itself ⇒ `UNGOVERNED_EFFECT`, completion BLOCK/UNKNOWN, no false success on locally accepted intent; noninteractive sudo/admin/backup escalation paths must be proven denied. If not demonstrable, `BUILDER_CONFINEMENT_UNAVAILABLE` blocks Local Task closure.
- **Local Hidden without runx/Docker:** A separate verifier OS principal with filesystem ACL denies the Builder access to fixture files, including through privilege escalation, inherited mounts/env/credential stores. Fixtures materialize after candidate freeze, and read-denial witness must be real. Missing verifier ⇒ `HIDDEN_VERIFIER_UNAVAILABLE`; readable fixture ⇒ `HIDDEN_NOT_QUALIFIED`. Runx/container/SSH are optional providers.
- **L3-before-Task:** Native GitHub Task Issue/DAG Ready is not executable without current ADS-version-selected L3 Tests/Contract/Implementation plan/Failure Handling/Reference source-qualified GitHub readback. Post-run Implementation Evidence is separate.
- **Report/release evidence:** telemetry hook nonzero/hang fails open; git-native current source and independent verifier/Full/Hidden/CJ/RQ remain actual gates; source-corrected ADS Report invalidates derived advice only. Candidate→main merge tree drift requires revalidation before tag.
- **Standards barrier:** Product draft may be independently semantically reviewed, but no effective v2 freeze/adoption until canonical project ADS 4.17 selection is *actually allowed by the pinned §2.1C RQ READY criterion* or an authorized prospective clarification, and its source-bound GitHub artifact is read back. Structural verifier PASS is necessary not sufficient.

**Required negative cases L2-N17…N30 are now normative source specifications in §10**, including Host/App Shell session+CSRF, protected-path Repair/role/delegation, durable GitHub facts, strict N=1 and Provider credential partition. All tests SOURCE_SPEC_ONLY / PHYSICAL_TESTS_NOT_RUN.
