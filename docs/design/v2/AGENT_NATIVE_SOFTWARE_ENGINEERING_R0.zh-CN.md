# Tally v2 — Agent 原生软件工程 Domain App（R0 / 中文）

> **2026-10-28 用户明确的首要产品定位：Tally 用软件工程约束 Vibe Coding，实际开发工作默认由多种 LLM Agent 执行，人仅在真正必要时介入。** 本文为 Product/L2 研究设计，不是已通过的 ADS 规范、物理 DomainHarness ABI、有效 Task Claim 或实现授权。并行参考 [Tally Product #63](https://github.com/kaicreator-mm/tally/issues/63)、[Agent #64](https://github.com/kaicreator-mm/tally/issues/64)、[Business #91](https://github.com/kaicreator-mm/tally/issues/91)、[UX #92](https://github.com/kaicreator-mm/tally/issues/92)、[Version Driver #145](https://github.com/kaicreator-mm/tally/issues/145)。英文版：`AGENT_NATIVE_SOFTWARE_ENGINEERING_R0.en.md`。

## 1. 产品成功定义：Agent 默认推进，用户管理目标而非手续

使用软件工程的需求工程、架构设计、配置/变更管理、测试与验证、质量保证、持续集成和发布管理来**提高 LLM Agent 生成的软件的真实质量、开发效率、可维护性和可交付性**。不是把瀑布式人力审批流程照搬成 AI Coding 表单。Tally 作为软件工程 Domain App 将这些活动组织为可中断、可并行、可追溯、风险比例化的**Agent 执行闭环**。

**默认操作者 = 合格 LLM Agent。默认用户角色 = 产品目标拥有者/被授权决策者，而不是每个 Issue 的手工 Controller。** 用户提出软件目标与约束后，Tally 能在当前已合法接受的授权边界内，自动组织 Agent 做适合的后续工程工作直到真正需要人：`Goal → Agent Discover/Plan → Independent Agent Review (when ADS required) → legally accepted Task DAG/Claim → Agent Build/Test → Independent Agent Review/Repair → Integration/Acceptance → qualified Release proposal → next version`。在每个 required approval/actual effect gate 前重新读取事实、检查当前权限；没有必要权限时保持候选/等待，不“自动批准”。

### 1.1 五条优先规则

1. **Agent-first，而不是 Human-first**：Planner、Architect、Builder、Reviewer、Verifier、Integrator、Release Coordinator 都默认可由不同配置的 LLM Agent/自动化 Tool 承担；可用性/独立性/任务风险不足才安排人工。人类可以随时接管，但不被要求做无价值的重复批准。
2. **Engineering-first，而不是 Prompt-first**：Agent 的输出必须和可检验的 Task 契约、接受标准、source/Claim/write set、当前配置、测试、Review、真实用户行为相绑定；生成文字不等于证明正确。低风险微小变更采用最小充分流程，复杂或高风险版本采用必要的独立检查。
3. **Capability-first，而不是固定 Agent 职位编队**：优先用一个有资格的 Agent 完成同一低风险 Concern 的连续工作；涉及独立审核/验证或者确有并行价值时才切成多个 Agent 上下文/角色。逻辑 Agent 身份和具体 GitHub 登录分离。同一模型也可作为独立 Reviewer 的物理 Provider，前提是 ADS 要求的上下文独立、实际 Review 隔离成立；仅换角色标签、自审自批不算独立。
4. **Autonomy within delegated authority**：日常 Plan/Task/修复/Review/CI 可以按现行 ADS 和真实权限自动推进；修改已冻结 scope、擅自重占 Claim、放松 Gate、执行未授权副作用、推送生产、无界消耗成本都不得因为 LLM 自信而自动实施。明确授权才允许批准边界内的操作；批准是可验证的真实来源事件而非文本提示。
5. **Exception-driven human participation**：只在无法由规范与来源确定的目标/风险取舍、适用政策要求人工批准、GitHub/Host/本机/凭据授权、外部实体动作、预算上限、无法收敛的冲突、需要用户体验验收或正式发布权责时请求人。能通过 Agent/CI/已授权 Host 执行的普通工作不必请人手工转发 Prompt/Issue 评论。

## 2. Agent 业务角色、能力证据与角色分离

| 角色 / 可按需合并 | Agent 主要工作 | 角色独立和结束证据 |
|---|---|---|
| Product Analyst / Planner | 仓库扫描、需求澄清候选、PRD/功能验收草案、历史经验检索、非目标、风险/成本 | 用户目标明确且来源充分则自动出候选；真正的 Product/范围确认采用 ADS 当前可接受的批准路径 |
| Architect / Decomposer | 架构和 Scaffold 选型、子需求/真实 blocked_by/并行 Lane、写集、测试方案、Agent/Host 能力匹配 | Task DAG 原生依赖和 accepted source 与独立必要 Review，不因模型图示而入场 |
| Builder | 在 accepted Task+Claim+范围内以 Codex/Claude Code/ZCode 等有实际适配的 Agent/CLI 修改源码、生成 PR、局部验证 | 提交 commit/PR、exact SHA、检测输出、任务运行记录；不可自我宣称独立 PASS |
| Reviewer（独立） | 对指定 exact HEAD 做实现/架构/安全/回归审核，给 P0/P1/P2 与可定位 Finding | 与 Builder 上下文分离，ADS/项目适用独立性合格；Review 自述不得替代 CI/真实 Host |
| Verifier / Test Agent | 生成和执行适合的测试、CI/真实设备/installed consumer 检验、negative oracle | 验证实际执行来源/版本/环境/结果，低覆盖和未运行显式 UNKNOWN/NOT_RUN |
| Integrator / Release Agent | 对 PR/写集/集成契约/变更影响做 Join，准备 Release Candidate/RQ/Tag/发布回读 | 只有现行授权允许的 merge/tag/deploy 才执行；模型输出 `READY` 不是授权 |

**执行实体** `AgentInstance` 关联 provider/model/version/session/host/profile/capabilities/actual task role/Claim/Attempt；能力既有自声明，也须记录来源可证的真实成功/失败与覆盖。L2 以现有 [#64](https://github.com/kaicreator-mm/tally/issues/64) registry 复用，不能创建另一个统一 Agent SDK 或假定某 Provider 已经支持所有工具、长任务或自恢复。模型路由可按可用性、任务类型、上下文要求、历史适用性、预算、Reviewer 独立要求选择，**不得仅按历史 Issue 数/LLM 自评分数排名**。

### 2.1 Planner→Builder→Reviewer→Repair 的自主循环

```text
User goal & bounded delegation
    ↓
Tally Engineering B nextActions + qualified GitHub source
    ↓
Agent Planner/Architect: Requirement, L2, DAG candidates
    ↓ mandatory normative gate only
ADS-qualified review/freeze/native dependency+Claim admission
    ↓
Parallel eligible Builder Agents ←── pinned Task Context, write set, CI/Host profile
    ↓ actual GitHub PR, selected tests, source SHA
Independent Reviewer Agent + qualified Validator/CI
    ├── qualified PASS → legal integration/version gates
    ├── P0/P1 Finding → scoped root-cause repair by lawful Builder
    │                        └─ successor exact SHA → independent re-review
    ├── source/Claim/tool/Host UNKNOWN → reconcile or request evidence
    └── material product/policy/budget conflict → only then Human Decision Inbox
    ↓
Acceptance: installed selected user journey, required CI/Host/Full/Critical
    ↓
Authorized Release RQ, merge/tag/deploy as applicable
    ↓
New version, evidence and qualified experience reuse
```

一条工作可以由一个 Agent 连续完成可兼容的分析、开发和本地测试，但**不能在需要独立审查的环节自行变成独立审核者**。Review 多轮有上限和根因分析，不能无限建新 Issue、跑完整测试或把每个 Finding 都触发重新 Product Freeze。合法 Task 恢复或更换 Agent 必须维护真实 GitHub accepted Claim 的交接/失效/补偿语义，未终结 Effect 保持 UNKNOWN 而非再次盲目执行。

## 3. 三种 Agent 自主级别和四个明确人工介入条件

**自主级别是 Tally 的产品/会话业务预期，不是替代 ADS/GitHub/K 授权的凭证：**

| 工作级别 | 建议默认体验 | 门禁/人类边界 |
|---|---|---|
| A — 可安全自主工作 | Agent 读取资料、生成选项、草案、运行允许的只读检查；在合法已授予 Task/Claim 里开发/测试/Review/修复 | 无需用户反复发“继续”；没有源码写入资格时仍只能做只读候选 |
| B — 有条件自主效果 | 已限定仓库、分支、写集、成本/工具预算、适用 CI、授权的 PR/Task action | 只有通过现行源/Claim/K/C 的实际准入才能做 Effect；自动执行范围可审计、可撤销未来权限 |
| C — 明确提升权限/重大决定 | 范围新增或退役、破坏性重构、不可逆外部动作、风险豁免、生产部署、要求 Owner 批准的 Release | 应呈现简短选项+影响+推荐+证据，由真正合格决策者批准，或按已存在的明确授权机制进行；未授权 BLOCK |

**人工介入准确触发为：**（i）产品目标、冲突需求、重大技术/风险取舍无已授权规则可裁定；（ii）权限/凭据、Local Android/Windows/真实物理环境或真实外部业务动作无法由当前 Agent/CI 执行；（iii）真正超出批准预算、时间、工具、写集、Claim、版本范围/安全策略；（iv）项目现行规范要求有主体资格的人对特定发布或风险承担责任。其他如 CI 红、P1 可复现、Reviewer 提出局部修复，应由 Agent **先提出和执行合法有界修复路径**，而不是立刻打断用户。

用户 Inbox 每项必须说明：`为什么一定是人？为什么 Agent/CI 不能？现在可选的最少决策、实际风险/成本、来源链接、授权后能继续哪些 Lane`。能够自主解决就不产生 Human Required 项。用户批准并不等于绕过源/Claim/K 的真实技术校验。

## 4. 工程知识的 Agent 化：Domain Package B/S/K/C

**Tally B（Business）** 负责 `Goal→Requirements→Architecture/DAG→agent-aware eligible Task→Repair/Assurance→Release→Improvement` 的场景和决策，关键新增业务：
- `AgentWorkPlan`：定义 `taskContract/exactSource/writeSet/requiredEvidence/assignedRole/capabilityFit/modelAndHost/budget/requiredIndependence/exitCondition`；
- `RoleAndCapabilityMatch`：区分 `DECLARED` vs `OBSERVED` 的能力证据，缺验证时 `NO_ELIGIBLE_AGENT`，不强制云 Provider 处理需要本地 Host 的任务；
- `DelegatedAuthorityEnvelope`：作用域、预算、准入来源和到期/撤销参考，用于向**现有** K/C 准入提供上下文，**自身绝不是 K Permit**；
- `TaskOrchestrationDecision`：`START_ELIGIBLE/WAIT_GATE/WAIT_AGENT/HANDOFF/REVIEW/REPAIR/RECONCILE/HUMAN_REQUIRED/ABSTAIN`，带 source/currentness、Actor、理由和下一步；
- `HumanInterventionRequest`：仅必要时生成，含差异、选择、最小本地动作、回读条件和恢复线索；
- `ReviewerSeparationProof`：逻辑身份、session/上下文来源、Builder 排除、exact HEAD/只读边界、是否符合当前适用 ADS 独立性。
以上为 **proposal DTO**，需和现有 #64/#91/#125/#145 适配，不声明其已在 SDK 中实现。

**S（优先复用）**：Context Pack、决策规则、适用已验证 Procedure、自动化测试/审查策略建议、在下一步是否需 LLM 的选择、引导式 StepSpec；已解决的问题尽量走规则/经验，LLM 只处理需要真实推理和新代码的情况。模型选择/提示词不是权威。**K** 唯一 CAS/Journal/Effect/Replay/UNKNOWN；**C** 接真实 Codex/Claude Code/ZCode、GitHub、CI、Host/Model/optional runx，各自可选且有真实能力证据。**UX** 重点展示“正在做什么/为何被阻/现在是否真的需要我”，不是要求用户完成冗长表单。

## 5. DomainHarness 实战案例作为 Agent 原生产品反例

| 实例 | Agent-native 教训 | Tally 正确响应 |
|---|---|---|
| [DH v0.7 #530](https://github.com/kaicreator-mm/domain-harness/issues/530) 错误跨版串行 | Planner Agent 的 Task DAG 不能把最终兼容 Join 拖成所有 Builder 前置 | 在规则内建议合法并行；绝不私删已采纳 native blocked_by |
| [v0.7 #586](https://github.com/kaicreator-mm/domain-harness/issues/586) 多 checkpoint 验证 Campaign | Validator Agent 可复用 Campaign 和证据，不必每次新派一人/新 Issue | 一个 Campaign，多有界 checkpoint terminal，缺项仅补有关检查 |
| [v0.8 #1013](https://github.com/kaicreator-mm/domain-harness/issues/1013) 三个重叠实现 PR | Agent 配置不能替代写集/Claim single writer 和 Integration Owner | 提醒/止损重叠 Builder，真实协调合法交接而非双重写 |
| [v0.9 #1119](https://github.com/kaicreator-mm/domain-harness/issues/1119) 外部 clean consumer | Validator Agent 必须检验用户实际使用，不只是 Builder 本地运行 | 自动安排独立安装使用测试，未执行保持 NOT_RUN |
| [v0.10 #1264](https://github.com/kaicreator-mm/domain-harness/issues/1264) 已 load 未 invoke | Reviewer Agent 要识别契约和真实 selected path 的差别 | 已加载、已实现、真实调用和产品验收分开 |
| [v0.10.2 #1385](https://github.com/kaicreator-mm/domain-harness/issues/1385) 4/15 Host 失败 | Diagnoser Agent 应做有界统计/根因，不能单次 CI PASS 就放行 | 使用真实多次证据和 CI 适用性，避免无界 Full 循环 |
| [v0.10.3 #1411](https://github.com/kaicreator-mm/domain-harness/issues/1411) 原生 GitHub dependency API 缺失 | Tool capability/permission 缺口而不是人为审批目标 | 只对该物理操作产生 Human/Local Handoff，读回后 Agent 自动继续 |

## 6. 可衡量的成功和 Golden

**完整真实 Agent-first 验收：** 用户在 Tally 中描述“开发某个管理软件”后无需每个阶段手动给 Agent 下指令；Agent 自主分析、产生需要用户决定的最少问题、生成 Product/L2/DAG；条件允许时独立 Agent 按真实 GitHub Claim 并行完成两个 Concern，Agent Reviewer BLOCK 一次并由 Builder 自动进行合法有界修复，真实 CI/Host/installed consumer 验证，发布权责获有效授权后自动完成合法 merge/tag/回读，下一版自然延续。必要人工接入应仅来自上述四类，并按理由统计，不将人工 Review 设为唯一正例。

**必须验证的负例**：Builder 伪装 Reviewer；两个 Agent 同一 GitHub login 但声称是同一 logical actor；同一写集抢 Claim；Agent 未证明工具能写 native blocked_by 却宣称已派发；Agent output says PASS but real Host 未跑；Review stale exact SHA；模型自动批准新 Product scope/高风险豁免；Host ACK 丢失后重复外部副作用；task-meta down 却拦截正常执行；人尚未授予凭据却让 Agent 自动获取；预算上限耗尽但仍调用付费模型。以上不得标“Agent Autonomous PASS”。

**评价指标**：真实用户目标达成、真实 Host/CI/Release 合格、人工介入次数和必要性、人类等待/交接开销、有效 Agent 自主连续工作占比、错误 Claim/重复 PR、返工根因、Agent/模型成本/耗时和计量覆盖。**减少人工次数本身不是安全/质量指标**，不能通过省略必要 Gate 来优化。Active Agent 时间只能来自实际有覆盖的 task-meta 观测；Issue 创建关闭日期不等于执行时间。

## 7. 三项目对 Agent-native 的持续进化

- **task-meta**：关注 `AgentIdentity/Provider/Model/Role/Attempt/Step/Tool/CI/Host` 真实观测、可靠归因、coverage、human intervention reason、handoff wait、模型成本/工具成功率（按可发布公开摘要与隐私约束）；不负责 Agent 的选择或调度。
- **ADS**：定义可验证的逻辑 Agent 身份、真实 Claim、独立 Review、正常无人介入的适用准入及“何时确实必须人工”的标准，不把 Human Approval 编成每 Task 永久仪式；规范更新仍须独立受控。
- **Tally**：在当前 ADS/pin/K 授权下学习何种 Agent/Model/Role/Host 对哪类 Concern 和验证要求最可靠、怎样安排最少必要角色/上下文切换、怎样自动恢复/修复，以及何时 abstain/handoff；经验晋升经过 Casebook Golden、holdout、独立评价、批准和新的封闭 BUILD。

**验收边界**：此文只改变产品定位与研究设计，不扩张在途 `version/v2` Task DAG 或自动冻结 #77/#82；后续独立 Product/L2 Reviewer 须验证 User Outcome 是否真 Agent-first、是否减少了“人必须点下一步”、是否存在伪独立 Review/隐藏人工要求、是否重建了 Kernel/Agent SDK。
