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

## 3. Package 结构、初始化与运行时版本

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

**顺序**：A–D 已准入范围并行；F 的纯文档/资产设计可审阅，但修改与现有 B 写集冲突的源码必须等待依赖及合法 CAS/JIT；E 只有通过真实依赖读回才可集成。不得将文本依赖描述伪报为 GitHub 原生 `blocked_by` 事实。

## 7. 给 DomainHarness 的通用反馈及反向依赖

- 复用 [DH #1465/PR #1466](https://github.com/kaicreator-mm/domain-harness/issues/1465) 的 Domain Data 与函数边界。
- 在 [DH #1468](https://github.com/kaicreator-mm/domain-harness/issues/1468) 分类登记 C↔外部 MCP/HTTP/CLI、Business Event、生命周期、Workflow、双层进化；先审已有公开 ABI，再凭 clean consumer 提通用改进。
- [DH #1467](https://github.com/kaicreator-mm/domain-harness/issues/1467) 是命名质量调查，不暗示必须重命名。
- 经验以实际消费者、相同 Exact SHA、预期/实况、最小复现、失败出口和修复证据记录到 [DH #1392](https://github.com/kaicreator-mm/domain-harness/issues/1392)，可复用条目分流 #1465。不能以 Tally 一个项目的业务偏好增加 DH Kernel 硬编码。

## 8. 本文件的验收与限制

仅证明设计归纳和源码静态交叉核对；`SOURCE_REVIEW=SELF_ADVERSARIAL`、`INDEPENDENT_REVIEW=NOT_RUN`、`NODE24_TEST=NOT_RUN`、`WOODPECKER=NOT_RUN`、`HOST_E2E=NOT_RUN`、`PACKAGE_RUNTIME_MIGRATION=NOT_PROVEN`、`TASK_PACK_FREEZE=NO`、`MERGE_AUTHORIZED=NO`。

后续任何实施必须以当前 GitHub exact HEAD、合法 Task/Claim、独占写集及新 Owner 授权为准；此 R1 不暗示自动实施或自动发布。
