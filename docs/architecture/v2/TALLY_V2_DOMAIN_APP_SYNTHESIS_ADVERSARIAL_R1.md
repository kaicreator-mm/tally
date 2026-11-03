# Tally v2 — Engineering Domain App 综合架构与自我对抗审查 R1

> **性质**：设计综合与 self-adversarial review，`REVIEW_REQUIRED / IMPLEMENTATION_NOT_ADMITTED`；并非独立审查 PASS、正式 L2 Freeze、Task Claim、Host 验收或 Release 资格。仅基于列明的精确 GitHub 来源，不替代真实 Node24/BUILD/Host/CI 验证。**请勿在本文件中声明“已完成”尚未发生的物理验证。**

## 0. 来源、适用边界与现状

- Tally 版本基线：[`version/v2@7ebb1cdd0a79fc82cc938e35f852a80dd8e15332`](https://github.com/kaicreator-mm/tally/tree/7ebb1cdd0a79fc82cc938e35f852a80dd8e15332)。
- 当前单一集成候选：[`PR #258`](https://github.com/kaicreator-mm/tally/pull/258) Draft `review/ui-r1-integrated-candidate-20261101@ad0433f8da6490e46f902cea0d68cf032f3f1b81`。本文件基于此候选建立 docs-only 的**从属审查 PR**，不成为另一个独立进入 `version/v2` 的集成 PR。
- DomainHarness 发布源码：[v0.10.3 Git tag `061376f4a023070f215f9feffad5c24d96d503cb`](https://github.com/kaicreator-mm/domain-harness/tree/v0.10.3)；历史 V10/Business ABI 和选定 Host 的实际安装状态**必须单独证明**。
- DH 设计和经验：[#1465](https://github.com/kaicreator-mm/domain-harness/issues/1465)、Draft [PR #1466](https://github.com/kaicreator-mm/domain-harness/pull/1466)、改进汇总 [#1468](https://github.com/kaicreator-mm/domain-harness/issues/1468)、版本命名检查 [#1467](https://github.com/kaicreator-mm/domain-harness/issues/1467)。经验汇总写入 [DH #1392](https://github.com/kaicreator-mm/domain-harness/issues/1392)。
- Tally 原有架构：[PR #147](https://github.com/kaicreator-mm/tally/pull/147)、[PR #152](https://github.com/kaicreator-mm/tally/pull/152)、[#60](https://github.com/kaicreator-mm/tally/issues/60)。当前协调 [#259](https://github.com/kaicreator-mm/tally/issues/259)，A #260、B #261、C #262、D #263、E #264、Replan #273、Feedback #274、Attempt #275、领域提取 F [#276](https://github.com/kaicreator-mm/tally/issues/276)。
- **不修改** DomainHarness、ADS、task-meta；没有授权对 `main`、`version/v2`、DAG/CAS、生产 Host、发布标签进行任何写入。使用 GitHub 为唯一跨 Agent 交互源。

## 1. 最终架构裁定：三个不同坐标轴

### 1.1 所有权轴（K/S/B/C/Host）

| 所有者 | 唯一责任 | 不应拥有 |
|---|---|---|
| Domain Data（Tally B 拥有） | 工程领域词汇、权威指针、Schema、Rule、Pattern、Recipe、Skill、Workflow、正反例 | 自动运行、授予权限、自动 Promotion |
| B Engineering | 业务不变量、业务 Intent、流程定义/映射、领域判断和投影 | GitHub/ADS 原生事实、K Journal/CAS、任意 Provider |
| S（DH 选定） | 通用有界语义/Context/Decision/Feature 机制 | Tally 专有发布/Task 规则、最终业务批准 |
| K Primary（唯一） | Admission、State、CAS、Journal、Effect Permit、Replay、UNKNOWN | Tally 业务知识硬编码、第二调度器 |
| C Capability | BUILD 选定的窄权限 Operation，外部事实读取或受许可的 Effect | Owner/Release 资格决定、随意调用任意 Tool |
| Trusted Host | Physical Store、凭据、Provider Adapter、环境和生命周期 | 绕过 K 的自主业务状态机或授权 |

DomainHarness 的 `component.ts` 当前正式 Component Family 为 `semantic | tool`；历史 [DH #1361](https://github.com/kaicreator-mm/domain-harness/issues/1361) 提出 Semantic/Hook/Tool 三类统一概念，**不能视为当前已发布的三类同权 ABI**。K Advisory Extension 是 K 受限机制，与业务事件/外部 Webhook 不同；[#1269](https://github.com/kaicreator-mm/domain-harness/pull/1269)、[#1271](https://github.com/kaicreator-mm/domain-harness/pull/1271) 仍未合并，不能假定其中新行为已发布。

### 1.2 调用轴（明确“非 Tool”）

| 类型 | 示例 | 调用契约 |
|---|---|---|
| 模块私有纯函数 | DAG 无环、Task Diff、脱敏、Digest、Evidence 规范化 | Direct（非 Tool） |
| Package 公共 ABI | `getPolicy()`、声明的业务类型 | BUILD/选定入口（非 Agent Tool） |
| B Intent | `proposeReplan`、`approveFeedback` | 受信入口 → K `send` |
| K 状态观察 | WorkUnit/Attempt Snapshot、进度订阅 | 已发布且实测的 K Query/Subscription/Host 接口，不等于 SQLite 扫描 |
| S Feature | 有界语义判断、Context、Workflow 语义 | 显式选中 Feature Export，非默认 Tool |
| C Operation | GitHub Issue read、ADS publish、Agent dispatch、Woodpecker trigger | BUILD 选中后经 K 受控 `resources.call` |
| Host Port | Provider 凭据、HTTP/MCP/CLI、物理持久化 | 仅受信装配/Adapter |
| Agent-facing Tool | “申请执行 Task” | 独立 Tool Adapter → 认证范围内 B Intent → K |

**C Package 不等于外部 Tool**：C 定义受控 Operation；其 Handler 可以用受信 Host Adapter 调用远端 MCP、HTTP API、Agent/CLI/CI，也可以执行受控本地操作。`read/effectful` 与 `local/remote` 是正交属性。当前 C `externalEffect` 为模块级，应拆只读 C 与 effectful C；远程读取不因网络而自动成为外部写入，文件写入不因本地而免于 Permit。不要提供 `github.api(method,url,body)`、`shell.execute(any)`、`sql.query(any)` 等环境全权限 Tool。

### 1.3 资产轴（Domain Knowledge ≠ Semantic）

| 领域资产 | 源格式 / 消费方式 | 运行时限制 |
|---|---|---|
| Glossary、Authority、Reference | 版本化 Markdown/JSON、精确来源 | 不是自动 K State |
| Schema、Rule、Validator | Schema + 纯确定性 B 函数 | 未经 BUILD 绑定不自动执行 |
| Pattern、Recipe、Example/Counterexample | 可审计经验资产、案例和验证样本 | 单次经验不能自动升格为 Rule |
| Skill / Semantic Candidate | bounded schema-bound 指令及已选 S | 不能自行颁发 Permit |
| Workflow Definition | 领域状态、Guard、Transition、Wait、Evidence、Recovery | 不能假设 `domain/workflows/*.yaml` 被 v0.10.3 自动解析 |
| Tool Policy | 权限、Scope、输入、幂等、UNKNOWN、Readback | 不等于已经被 BUILD 选中 |
| Qualification Evidence | 验证报告、真实消费者结果、包 pin | 不等于自动部署 |

## 2. 核心 Tally 业务模型与八个有界 Workflow

保持一个物理 `packages/tally-engineering-business`，内部划分 B1 Product/Requirement、B2 Architecture/Planning、B3 Execution/Journey、B4 Assurance/Release、B5 Improvement/Experience。先不假定当前 Public ABI 支持任意多 B 热插拔。

每个 Workflow 必须分别列出稳定 ID、版本/owner、触发、输入输出 Schema、来源权威、States/Transitions、等待/拒绝、所需 Business Intent 与 C Operation、K-owned State/Effect/Replay、完成证据以及恢复/UNKNOWN。**UI Step 仅是 Workflow Step 的投影，Agent Skill Flow 仅是候选生成过程。**

| Domain Workflow | B context | 要产出的规范证据 | 需要注意的失败路径 |
|---|---|---|---|
| WF01 Project Intake | B1 | Requirement、Acceptance、Product Review/Freeze | 未审批不得伪造 Frozen |
| WF02 Architecture/L2 | B1/B2 | Architecture Decision、Independent L2 Review | 不得从 UI 操作推断 Review PASS |
| WF03 Version Planning / DAG | B2 | Version、Task Pack、native Dependencies、Owner exact blob | 旧 `updatedAt`/非 native edge → STALE/BLOCK |
| WF04 Task Execution / Attempt | B3 | 接受的 Claim、真实 Attempt、Agent/Host receipt | 无责任交接、无 Claim、UNKNOWN 不能标成功 |
| WF05 Independent Review / Repair | B3/B4 | exact-SHA Findings/Repair/Re-review | 自我审查不可替代独立 Reviewer |
| WF06 QA / Release | B4 | Woodpecker、Host/Browser、RQ/Owner | 语法检查必须先于 CI；不假定自动发布 |
| WF07 Maintenance / Replan | B1/B2/B5 | append-only PlanRevision、accepted-work 保留 | Candidate 不等于可执行新 DAG |
| WF08 Experience Qualification | B5 | Evidence→Candidate→Validation→Promotion→BUILD selection | 不允许自动晋升和 active-instance 热切换 |

现有 `src/index.mjs` 是 **1,646 行**，声明 `MODULE_ID`、`PACKAGE_KIND='business'`、`ABI_VERSION='v08-business-abi/1'`、10 个 Intent 和对应 Policy/Effect routing；它主要体现 Business operation/effect 路由，不足以证明上述八个可恢复的 ALM 领域 Workflow 已全部实现。不得简单切分相对 `import` 破坏闭世界 BUILD 闭包。

**同一精确基线的静态结构核对（仅源码盘点，不等于运行时调用图）**：
- `packages/tally-engineering-business/src/` 当前包含 12 个 MJS 文件：`index.mjs` 1,646 行，另有 11 个独立模块共 1,008 行；源码总量约 2,654 行。
- 当前 `index.mjs` 有 **0 条 import**，而 B 的 `package.json` 只将 `.` 映射到 `./src/index.mjs`。因此不能假设其余 11 个模块自动进入 sealed B 的 BUILD closure；必须逐一追踪它们的真实调用方、装配点和是否属于 B/Host/UI 的责任。
- `index.mjs` 除 ABI 元数据外还导出多组 assess/project 函数。它们不是因为被 export 就自动成为 Agent Tool，但必须逐项分类为必要的 Package ABI、仅供同包使用的纯函数，或应由可信 Host/C 消费的适配边界。
- 同一基线的 `packages/tally-engineering-business/README.md` 仍称此包仅拥有一个 Task-readiness advisory Intent，而当前 `index.mjs` 已声明 10 个 Intent / 10 个 Effect。这是已证实的文档漂移，应在重构中修正，但不能据此推断运行时行为已全部接通。

**优化要求**：先做 source-to-runtime call-site / BUILD-closure map，再决定是保留单一入口、将纯函数收回私有、拆分 B 模块，还是由 C/Host 提供 typed adapter。禁止仅按文件行数拆分，也禁止在没有 selected BUILD/clean-consumer 证据时宣称拆分模块已被运行时加载。


### 2.1 B Package source-to-callsite / BUILD-closure map

在同一 `version/v2@7ebb1cdd0a79fc82cc938e35f852a80dd8e15332` 上继续检查实际调用点后，结构问题可以进一步具体化：

1. **V10 Producer 只选择一个 B 源文件**：`experiments/v2/selected-business-build/producer.mjs` 从 `TALLY_BUSINESS_SOURCE` 动态导入文件、读取其文本为 `bBytes`，并将这一个文件作为 Business 模块的 `moduleSource`。它没有导入 11 个 sibling modules。B 的 `index.mjs` 本身也没有 import，因此这些文件不会因为与它同目录就自动进入 sealed B。
2. **10 个声明不等于 10 条已选路由**：B 声明 10 个 Intent / Effect type；Producer 的 route set 是条件化的。`inspectTaskReadiness` 总是选择；其余 `admitClaimedTask`、`runLocalCodexAttempt`、`joinReviewedCandidates`、`publishReviewedCandidate`、`freezeProductIntake`、`admitProjectIntake`、`publishMaintenanceIssue`、`recordTaskLearningOutcome`、`deliverQualifiedRelease` 分别取决于对应 capability/feature flag。不能把 Intent 声明数当作每个 BUILD 的完整路由覆盖。
3. **V10 Consumer 的证明范围**：`experiments/v2/selected-business-build/consumer.mjs` 使用 public Microkernel `load()` 加载 sealed App，并发送 `inspectTaskReadiness`。它证明该条实际选择路径的 K 行为，不能单独证明其余九条可选路由全部端到端通过。
4. **Intake Host Controller 的直接调用**：`web/v2/intake-controller.mjs` 直接导入 `guided-intake.mjs` 的 `nextGuidedStep/guidedCaseRef/GUIDED_TEMPLATE_REF`、`planning.mjs` 的 `planNextVersion`，以及 `index.mjs` 的 `assessProjectIntakeAdmission/projectProjectIntakeAdmission`。它们用于动态步骤、DAG 候选和纯预检/结果投影；真正的准入仍经注入的 `sealedDispatch` 进入 selected K。
5. **Maintenance Host Controller 的直接调用**：`web/v2/maintenance-controller.mjs` 直接导入 `maintenance-intake.mjs` 的 `projectMaintenanceIntake`，以及 `index.mjs` 的 publication / learning assess/project helpers。实际发布和记录操作仍通过 `sealedDispatch`；未发现它直接调用 Provider。
6. **其它 sibling 模块**：在已检查的 `web/v2` Controllers、React UI Feature Models 和 C 源文件中，没有找到 `acceptance.mjs`、`casebook.mjs`、`diagnosis.mjs`、`feedback.mjs`、`journey.mjs`、`lineage.mjs`、`release-readiness.mjs`、`work-units.mjs` 的运行时调用点；它们有各自 package-local tests。这是限定范围扫描，不宣称全仓库已证明 dead code。
7. **React UI 边界**：检查到的 `apps/tally-web/src/features/**` Model 通过 HTTP Host adapter 获取数据，没有直接 import Business source helpers；UI 动态表单 Schema 仍是 UI Step/Field contract，不是 Domain Data Schema。

**关键判断**：当前至少存在两种 Business 源码消费方式：一份 `index.mjs` 被 V10 BUILD sealed 选择；若干纯函数又被 Host Controller 从私有源码路径直接导入。现有代码显示这些 Host 预检/投影并不直接执行外部副作用，副作用仍受 `sealedDispatch` 保护；问题在于代码所有权、源码版本对齐和重复/漂移风险，而不是已经证明的 K Permit 绕过。

因此重构前必须给每个 helper 记录 owner（B/Host/C/UI）、调用点、是否进入 selected `moduleSource`、副作用类别、authority、测试和 Domain Data 映射。不要仅按行数拆分，也不要未经明确契约就让跨层代码 deep-import 私有 Package 内部。新增 Host/UI/CI 写集必须单独修订准入。

### 2.2 现有 B 源文件逐项归属与处置

以下表格以当前源码导出、已检查的调用点和 package-local test 为依据；“未发现调用点”仅表示本轮限定扫描范围，不代表全仓库死代码结论。

| 文件 | 行数 | 当前函数/职责 | 已确认调用点 | 当前测试 | 初步目标归属 |
|---|---:|---|---|---|---|
| `index.mjs` | 1,646 | Business ABI、10 Intent/Effect 声明、Policy handlers、assess/project | V10 Producer 选为唯一 B moduleSource；Intake/Maintenance Host controller 也直接调用其纯 assess/project | `business-policy`、`native-source-diagnostic` | 保留唯一 ABI Facade；逐项区分 K handler 与 Host 预检/投影 helper |
| `acceptance.mjs` | 82 | Acceptance item/coverage/table projection | 本轮 bounded runtime scan 未发现 | `acceptance` | B 纯 helper / Acceptance Domain Data 映射 |
| `casebook.mjs` | 80 | Casebook replay、affected case selection | 本轮 bounded runtime scan 未发现 | `casebook` | Knowledge / positive-negative-counterexample fixtures，不自动成为 Runtime |
| `diagnosis.mjs` | 104 | Workfront diagnosis | 本轮 bounded runtime scan 未发现 | `diagnosis` | B 纯诊断投影，需明确输入证据与 UNKNOWN |
| `feedback.mjs` | 66 | Feedback case projection | 本轮 bounded runtime scan 未发现 | `feedback` | Experience / Feedback projection，ADS authority 不复制 |
| `guided-intake.mjs` | 69 | Guided Journey schema digest、validation、next step | `web/v2/intake-controller.mjs` | `guided-intake` | 需要明确是 Host Journey adapter 还是 B domain step logic；当前依赖 AJV、node:crypto、node:fs，不能直接假定可进入 sealed B |
| `journey.mjs` | 99 | Journey snapshot、next actions | 本轮 bounded runtime scan 未发现 | `journey` | Workflow/UI Journey projection；不拥有 K durable state |
| `lineage.mjs` | 160 | Version timeline reconstruction/format | 本轮 bounded runtime scan 未发现 | `lineage` | Version/Package lineage read model；来源仍为 GitHub/ADS 原生事实 |
| `maintenance-intake.mjs` | 71 | Maintenance candidate projection | `web/v2/maintenance-controller.mjs` | `maintenance-intake` | B 领域投影；Host 可消费的边界需显式化 |
| `planning.mjs` | 93 | `planNextVersion` DAG candidate planning | `web/v2/intake-controller.mjs` | `planning` | B 纯 Replan/DAG 候选；不得写 native Dependency 或创建 Claim |
| `release-readiness.mjs` | 78 | Release readiness projection / required gates | 本轮 bounded runtime scan 未发现 | `release-readiness` | B 资格投影；不能签发独立 RQ/Owner/CI 事实 |
| `work-units.mjs` | 106 | WorkUnit projection | 本轮 bounded runtime scan 未发现 | `work-units` | B 只读 read model；Attempt 权威来源仍需 K/Host |

**当前最重要的边界发现**：

- B `package.json` 是 `private: true`，仅 export 根入口 `.`。Host controller 对 sibling helper 的相对路径 deep import 是 monorepo 内部源码依赖，不是已承诺的 public Package ABI。
- `guided-intake.mjs` 依赖 AJV 和 Node 文件/加密模块；它当前由 Host controller 直接消费，不能在没有显式 bundling/Host boundary 决策时移动进 sealed B。
- `index.mjs` 的 selected moduleSource 与 Host 直接导入的 helper 源码可能随不同构建/部署路径漂移。即使 helper 是纯函数，也需要统一 source pin / digest / consumer contract。
- 因此目标不应预设“所有 12 个文件都要放入一个 B 包”或“拆成多个 B”。先按表中职责把纯 domain computation、Host orchestration、Domain Data、selected B handlers、read model 分开；再决定通过 Producer bundle、受控 Host port 或内部纯函数包复用，且不能扩大 ABI 或外部权限。
## 3. Package 结构、初始化与运行时版本

### 3.1 不同版本坐标：ABI、SDK、应用与运行时

以下版本号属于不同坐标轴，不能相互替换：

| 坐标 | 当前值 | 含义 |
|---|---|---|
| Tally 应用 | `v2` | Engineering Domain App 的产品/版本线 |
| DomainHarness 发布源码 | `v0.10.3` | 不可变 Git tag，`061376f4a023070f215f9feffad5c24d96d503cb` |
| Business Package ABI | `v08-business-abi/1` | V8 Qualified Package 中 Business 角色的 ABI 标识，不是 Tally v0.8，也不是 DH SDK v0.8 |
| BUILD / Runtime Profile | V10 | `V10_TARGET_PROFILE`、`buildV10App()`、`activateV10App()` 所属的构建/激活契约 |
| 实际安装 Host | 必须单独 pin | 由选定 Host 的精确源码、打包产物、依赖闭包和运行回执证明 |

源码证据：
- DH `packages/domain-harness/src/public-v8/contracts.ts` 明确将 `v08-business-abi/1` 列入 `V8_QUALIFIED_ABI`；它表达的是 Business Package ABI 代际。
- Tally `experiments/v2/selected-business-build/producer.mjs` 同时导入 `V10_TARGET_PROFILE` / `buildV10App`，并断言 Tally B 的 `ABI_VERSION === 'v08-business-abi/1'`。因此在该 BUILD 路径中，V10 构建配置与 V8 Business ABI 是并存的，不应机械改名为 `v010-business-abi/1`。
- 这不代表 Tally 已完成 DH v0.10.3 的运行时升级。B README 和 `src/index.mjs` 的历史说明仍 pin v0.10.2；`run-clean.sh` 默认期望 DH SHA `cee14399d9d0af4e20e34b9111861b16a465b690`，仅 `T06M_LOCAL` / `T06N_CONCURRENT` 路径期望 v0.10.3 SHA `061376f4a023070f215f9feffad5c24d96d503cb`；Guided Intake README 仍声明部署 K/BUILD/Host pin v0.10.2。Tally #271 的 Host 预检也没有证明当前已安装 v0.10.3 Runtime。
- DH v0.10.3 是 Git-only 源码 tag；包清单中的历史 `0.10.2-rc.0` 字符串不等于 Git tag，也不能据此推导存在 npm `0.10.3` 包。应使用精确 tag 源码和实际 pack/consumer 证据，不得混装或伪造 semver。

**结论**：保留 `v08-business-abi/1`，除非未来经独立 Product/L2 决策定义了新的 Business ABI。当前真正的 P0 是把 Tally v2 的 DH v0.10.3 BUILD、Runtime、Host 和 clean-consumer 资格链补齐，而不是重命名 ABI 字符串。


建议文件组织为 `domain/{glossary,authority,schemas,rules,knowledge,patterns,recipes,templates,skills,workflows,tool-policies,examples,validation}/` 与 `src/{domain-model,contexts,policy}/`、`test/{contracts,workflows,critical-journeys}/`。**这仅是源码组织约定，不是已验证的 DomainHarness 自动加载目录规范。**

B Package 初始化、业务首次进入流程、Host 初始化是三种不同问题：
- V8 Qualified Package 可选 `init/dispose`；V10 已选 S Feature/K Extension 有受限生命周期。
- `tally-engineering-business` 的当前 B ABI 入口是 `getPolicy()`，没有证据支持为这个 V10 B 随意添加通用 `init()`。
- Schema/知识验证优先在 Producer/BUILD；S/K extension lifecycle 用公开 ABI；Host owns provider/session/connection；首次建立业务 Project/Workflow 必须通过受控 B Intent/K。
- `v0.10.3` Git 标签存在 ≠ 所有包对外 semver 已对齐 ≠ Tally K/Host 已经切换到 v0.10.3。当前 `packages/tally-guided-intake/README.md` 表述“v0.10.3 contract-first、deployed K/BUILD/Host 仍 pin v0.10.2”；`tally-engineering-business` 仍明确声明 v08 ABI。须记录 Runtime exact package exports、BUILD manifest、Host、dependency closure 与 installed clean-consumer 结果后才能宣称运行时升级。

## 4. 两层、多级进化：采用两个职责层、两条正交轴

**层**：
- Task/Business 实例：一个 K Primary，使用已选 B/S/C 完成业务 Workflow，记录可核验的源与结果。
- Meta/Evolution Producer：跨多个已完成 Task 从**获授权、已脱敏**证据提取候选；不能拥有第二个 K、不能自动修改当前实例。

**轴 A：单次 Task 的决策来源优先级**（历史 v0.3 原始顺序，是否在现有 v0.10.3 可运行须额外证明）：
`Compiled Rule → Exact Semantic Reuse → Applicable Promoted Subworkflow → Bounded LLM fallback`。最终结果仍为待 B/K 准入的 Candidate。Semantic Cache ≠ Journal Replay。

**轴 B：资产成熟度和版本**：
`OBSERVED → CANDIDATE → VALIDATED/SHADOW → HUMAN/GOVERNED PROMOTION → QUALIFIED BUILD → NEW ACTIVATION → USAGE FEEDBACK`。Promotion ≠ BUILD selection ≠ Activation ≠ Execution。运行中 occurrence 固定旧 Package pin；缺少真实验证不得升级权威。

## 5. 自我对抗审查（事实/推论/反例、严重级别与处置）

此处 P1/P2 为**设计风险优先级**，不是独立审查 Finding 或已证实生产事故。

| ID | 初步风险 | 对抗质疑 | 源码/任务依据 | 处理与判定 |
|---|---|---|---|---|
| T-AR01 | P1 | “使用 DH v0.10.3”是否把契约消费误写成 Runtime 升级？ | Guided Intake README、B `v08-business-abi/1` | `NOT_PROVEN`：分开 pin Contract/Runtime/BUILD/Host |
| T-AR02 | P1 | “Semantic/Tool/Hook 三类”是否混淆产品提案和已发布 Component API？ | DH `component.ts` 与 #1361，#1269/#1271 OPEN | 只把正式选中的 S/C 视为当前可执行，K Hook 受限 |
| T-AR03 | P1 | “放入 domain/ 即变成 Domain Package”是否错？ | V10 BUILD closed-world，当前 B 只有 `src/index.mjs` export | `BLOCK`：记录 asset→selected module/route→clean consumer 证据 |
| T-AR04 | P1 | “B 自行 init()”是否绕过 B 公开 ABI？ | DH v8 lifecycle 与 V10 selected advisory lifecycle 不同 | 拒绝自定义 B `init()`，优先 Producer/BUILD/Host |
| T-AR05 | P1 | “外部 Tool 直接替代 C”是否绕过 K Permit/业务权限？ | C module-wide `externalEffect`、#274 | 保留 C/Host typed adapter、negative zero-dispatch、native readback |
| T-AR06 | P1 | “业务事件交给 Hook”是否产生隐式副作用/第二调度器？ | DH #1468、K advisory scope；#1269 未合并 | 默认 B Intent/K Workflow；仅通用复现后评估 Domain Event Contract |
| T-AR07 | P1 | “10 个 Policy Transition = 8 个真实 ALM Workflow”是否成立？ | B `index.mjs` 1,646 lines；#276 | `NOT_PROVEN`：逐个 WF state/wait/recovery/Host evidence |
| T-AR08 | P1 | “候选/缓存/Shadow 自动演化”是否与生产资格冲突？ | DH #1398 / TASK_META_EVOLUTION、Tally #60 | 两层非双 K；Candidate 与 Promotion/BUILD/Activation 分离 |
| T-AR09 | P1 | “43 Task 当前已可冻结”是否忽略 #276？ | #260 Task Pack blob `d6c513ca3a2dc227a079223331113f4738023500` 不含 #276 | `STALE`：重新生成 44（若 roster 未变）、原生边 readback、独立 exact-body 审核 |
| T-AR10 | P1 | “Agent 自报/ACK = PASS”是否误判真实执行？ | #261、#274、#275、#264 | 无 selected Host/CI/RQ 原生证据则 `UNKNOWN/NOT_RUN` |
| T-AR11 | P2 | “全部流程重构到 B”是否造成巨型 Workflow/循环依赖？ | #147 五 Bounded Context，#276 | 一个物理 B；八个有界 Workflow；确保 BUILD 闭包 |
| T-AR12 | P2 | “read-only Model = SQL Query”是否误用 K Journal？ | #275 | 公开且实测的 K Query/Host read port；禁止直接 SQLite 私表 |
| T-AR13 | P2 | “API 含版本号 = 代码质量缺陷”是否过度推断？ | DH #1467 | `RESEARCH_ONLY`，保留兼容测试与精确 ABI pin |
| T-AR14 | P1 | “自我审查证明可合并”是否错误？ | 当前文档作者同为方案作者 | 需要独立 exact-SHA、Code/ABI/Host/BUILD Review；本结论不授权 Merge |

### 5.1 反证和证据要求（必须执行，尚未执行则标 NOT_RUN）

1. **正向**：使用准确 public tarball + Node24 clean consumer，BUILD 选中 B/S/C，实际执行至少一个受限 Workflow、一个只读 Operation、一个合法 Effect，返回正确 K/Host/native Receipt。
2. **反向**：未被 BUILD 选中的 Knowledge/Feature/Tool 必须不可调用；未授权 Effect 的 Provider dispatch 次数为 0；非法 Agent Tool Scope/失效 Claim 被拒绝。
3. **Recovery**：重复请求、lost ACK、STALE source、跨 Task 语义缓存失配、Journal replay 与多次 Attempt UNKNOWN 保持真实语义。
4. **UI**：动态表单由 B/Host StepSpec 提供，浏览器不能修改 K/Host 权威状态；运行真实 J01–J08 与 negative。
5. **经验**：不经独立 Evaluation/Owner Promotion/BUILD pin 的 Candidate 不能进入新实例，已运行实例不能热替换。
6. **环境**：Node `>=24`、Windows 无需 WSL，严格执行 `node apps/tally-web/scripts/syntax-preflight.mjs` 在任何 Woodpecker CI 前；所有结果写 GitHub，记录实际 EXIT、Exact SHA/Tree/Blob。

## 6. 进入现有任务流的最小变更（非新增并行执行引擎）

| 任务 | 应纳入的可验证交付 | 无权执行事项 |
|---|---|---|
| #260 A | 重新核对 #276 加入后的 Task roster/边/Claim/写集、独立 witness | 旧 43 Task blob 直接 Freeze |
| #261 B | Host/Agent 操作与 Business Intent、K/Host 权限映射；真正 Attempt 来源 | 浏览器或 Agent 直接拥有底层 C/Host |
| #262 C | Experience Candidate 与资格条件，真实 Next Task usage producer | 读取 task-meta 私有 Trace 或自动 Promotion |
| #273 | Replan Candidate 与独立受审批激活的 PlanRevision Workflow | 修改 Accepted Work 或隐式改 DAG |
| #274 | ADS v5 Feedback B Projection + effect C Publisher + read C，lost-ACK Readback | 单纯 ACK 当发布成功 |
| #275 | K/Host 受控 Attempt Snapshot + ADS 只读来源 | 直接扫描 SQLite/Journal |
| #276 F | Glossary/Authority/Schema/Rule/Workflow 资产提取，现有 B ABI 闭包及真实映射 | 任意增加 B init、非选定模块、第二 Workflow Engine |
| #264 E | F join 后的唯一集成、Node24 语法→测试→Woodpecker、Host/Browser/RQ | 合并 `main`、抢跑合并 PR #258、未授权 Release |

**顺序修订建议（2026-11-03；架构建议，不代替 Owner 对活动 Claim/任务状态的正式操作）**：

1. **暂停 DAG 的最终重建/冻结**：当前 43-Task Task Pack `d6c513ca3a2dc227a079223331113f4738023500` 不包含 #276，已不能作为最终冻结候选。虽然此前曾预计加入 #276 后为 44 Tasks，但在既有重构优化审计完成前，不应假定 44 就是最终数量。暂停 #260 的最终 roster/Task Pack 快照、原生依赖物化和 Freeze writer；不复用旧 Pack 或旧 Owner approval。
2. **暂停新的源码 Claim/Dispatch 与 E 集成准入**：#276 继续保持 PLANNED；#264 保持 BLOCKED/未准入；不为 #273–#276 新开源码执行，直到优化后的职责边界和写集确定。
3. **不撤销已有证据**：保留 A–D 已存在的分支、提交、Claim、PR、LOCAL_VALIDATION 和 CI 证据；不强制取消或重写活动 Claim。已经合法准入、写集不重叠且不依赖本轮架构裁定的验证/收尾工作可以完成，但必须先重读 Exact HEAD、责任人和依赖。
4. **先做只读架构优化审计**：盘点现有 B 模块、10 个 Intent、Workflow 定义、UI/Host/C 调用链和测试；建立领域术语/Authority/Schema/Rule/Knowledge 资产映射；逐个核验八个 ALM Workflow 的状态、等待、失败、恢复、UNKNOWN 和完成证据；对齐 DH v0.10.3 的 V8 Business ABI / V10 BUILD / Host 安装事实；识别重复实现、遗漏、循环依赖、错误抽象和需要调整的独占写集。
5. **审计结论稳定后再恢复 DAG**：先确定哪些旧任务可复用、哪些任务需拆分/合并/重写，再刷新完整 Issue roster、原生 `blocked_by`、Task Pack、allowed_write_set 和依赖读回；完成独立 exact-body 审核后，重新请求绑定最终 blob 的 Owner Freeze approval，再按新 DAG 建立 CAS Claim/JIT。
6. **CI 顺序保持不变**：任何 Woodpecker 前先执行语法预检、`git diff --check` 和 scoped tests；本轮审计本身不触发 CI、不合并 PR #258、不修改 `version/v2` / `main`、不发布。

不得将文本依赖描述伪报为 GitHub 原生 `blocked_by` 事实。

## 7. 给 DomainHarness 的通用反馈及反向依赖

- 复用 [DH #1465/PR #1466](https://github.com/kaicreator-mm/domain-harness/issues/1465) 的 Domain Data 与函数边界。
- 在 [DH #1468](https://github.com/kaicreator-mm/domain-harness/issues/1468) 分类登记 C↔外部 MCP/HTTP/CLI、Business Event、生命周期、Workflow、双层进化；先审已有公开 ABI，再凭 clean consumer 提通用改进。
- [DH #1467](https://github.com/kaicreator-mm/domain-harness/issues/1467) 是命名质量调查，不暗示必须重命名。
- 经验以实际消费者、相同 Exact SHA、预期/实况、最小复现、失败出口和修复证据记录到 [DH #1392](https://github.com/kaicreator-mm/domain-harness/issues/1392)，可复用条目分流 #1465。不能以 Tally 一个项目的业务偏好增加 DH Kernel 硬编码。

### 7.1 经验标识碰撞的实际反例

静态交叉检查 [DH #1465](https://github.com/kaicreator-mm/domain-harness/issues/1465) 评论发现：Tally 经验使用 `DPB-011..DPB-018`，xDownload 另一批经验也曾使用 `DPB-011..DPB-014`。因此先前“稳定经验 ID”的约定没有实现全局唯一。后续 Tally 经验 ID 应采用 `DPB-TALLY-###` 命名空间，以原 GitHub comment URL + 旧 ID 保留历史别名；不可凭无命名空间序号自动关联或晋升 Domain Asset。这是已证实的经验治理问题，优先修复文档/索引，不据此修改 DomainHarness Kernel。

## 8. 本文件的验收与限制

仅证明设计归纳和源码静态交叉核对；`SOURCE_REVIEW=SELF_ADVERSARIAL`、`INDEPENDENT_REVIEW=NOT_RUN`、`NODE24_TEST=NOT_RUN`、`WOODPECKER=NOT_RUN`、`HOST_E2E=NOT_RUN`、`PACKAGE_RUNTIME_MIGRATION=NOT_PROVEN`、`TASK_PACK_FREEZE=NO`、`MERGE_AUTHORIZED=NO`。

后续任何实施必须以当前 GitHub exact HEAD、合法 Task/Claim、独占写集及新 Owner 授权为准；此 R1 不暗示自动实施或自动发布。
