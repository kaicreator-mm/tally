# Tally
## ViberCoding 工程化软件生命周期平台
### 产品需求与总体架构文档（PRD + Architecture）v1.3 Review Candidate

**文档状态：** Review Candidate / Freeze Pending  
**版本：** v1.3 Review Candidate  
**日期：** 2026-09-03  
**产品名称：** Tally  
**方法论名称：** ViberCoding  
**核心定位：** Evidence-Driven AI Software Lifecycle Platform  
**目标使用规模：** 单人或小团队管理十几个到几十个软件项目  

---

# 0. 文档目的 [ALL]

本文定义 Tally 当前产品方向、产品边界、生命周期模型、LLM/Agent 职责、工程确定性机制、核心工程对象、Agent Runtime 抽象、Verification Independence、ChangeSet/Review/Integration、运行故障闭环、AdaptiveKit 演进策略、MVP 范围及阶段实施路线。

v1.3 在 v1.2 Final 的既有产品模型上做**定点可信性修订**，不新增产品领域对象。核心新增是：

- 将 Verification 从“预先冻结 + protected evidence + Qualification”进一步扩展为 **Authoring / Source / Information / Execution / Applicability** 五个相互独立的可信维度；
- 引入 `visible / held_out` 的 Evidence Visibility capability，同时把“是否必须使用 held-out”留给 risk/profile 驱动的 Verification Policy；
- 明确 held-out 内容泄漏、selection exposure、evidence shopping、fresh binding、verification environment 与 immutable ChangeSet 的边界；
- 将 Independence Requirement 放到 **Acceptance Clause**，Evidence Binding 只描述自身 provenance、visibility、qualification 与 observed facts；
- 区分 Evidence 的历史有效性与对当前 Desired Revision 的适用性；
- 明确 Runtime/Model escalation 终止当前 CodegenRun，并由新 Run 产生新的 immutable candidate ChangeSet；
- 统一指标、Authority ownership、阶段激活与 ADR 边界，避免 PRD 继续膨胀为 Implementation Spec。

本文可独立阅读，不要求读者了解任何前置讨论，也不依赖旧版 PRD。当前状态为 **Review Candidate**；Owner 审核通过后才标记为 `v1.3 Final — Frozen Product Baseline`。在冻结后，算法、存储、注入、判等、Runtime transport、coverage provider 等实现选择进入 ADR / Implementation Spec。

本文的核心判断是：

> **Vibe Coding 的价值是自由、快速、自然语言驱动和高自主性；Tally 的价值不是消灭这种自由，而是用软件工程方法把这种自由工程化、确定化。**

Tally 不要求人重新承担传统软件工程的全部流程劳动，而是尽可能让 LLM/Agent 负责产品文档、工程设计、任务拆分、编码、测试、调试、修复、文档同步和复盘，让人主要承担意图表达、关键决策和高风险审核。

一句话概括：

> **From Idea to Evidence. LLMs do the engineering work; Tally makes it trustworthy.**

# 1. 产品定义 [ALL]

## 1.1 ViberCoding 方法论

ViberCoding 是“工程治理下的 Vibe Coding”。它保留 Vibe Coding 的自然语言交互、快速探索、Agent 自主执行和低流程摩擦，同时要求真正进入项目历史的软件变化必须经过可追踪、可验证的软件工程闭环。

ViberCoding 不是要求 LLM 变得 deterministic，而是：

> **把概率性的生成过程放进确定性的工程边界。**

```text
              Probabilistic Zone

Idea → LLM → Product → Plan → Agent → Code/Debug
                         │
                         ▼
                  Engineering Gates
                         │
                         ▼
              Deterministic Zone

Git Revision / Scope / Verification / Test Result
Build / Deploy / Failure Fingerprint / Evidence State
```

核心原则：

> **Generation can be probabilistic; acceptance cannot be ambiguous.**

## 1.2 Tally 一句话定义

**Tally 是一个将人的想法通过 LLM 转化为产品方案、工程设计和可执行任务，并由 AI Agent 尽可能完成编码、测试、调试、文档维护和修复，再通过独立验证、确定性证据和状态推导把 Vibe Coding 工程化的软件生命周期控制平台。**

## 1.3 Tally 不是什么

Tally 不是：

- 一个只负责聊天的 AI 项目管理界面；
- 一个 Jira/Trello 的 AI 包装层；
- 一个重新实现 Codex、OpenCode、Pi 的 Coding Agent；
- 一个要求人手工填写大量 Requirement、Task、状态和日报的流程系统；
- 一个让 Agent 自己写测试、自己宣布测试通过、自己宣布完成的自动化脚本；
- 一个在早期就依赖大型工作流引擎、微服务或复杂组织权限体系的平台。

## 1.4 Tally 的核心价值

Tally 解决的不是“LLM 能不能写代码”，而是：

1. 人的模糊 Idea 如何低摩擦地变成可信的产品方案；
2. 产品方案如何自动转化为工程设计和可执行任务；
3. Coding Agent 如何在明确边界和可复现上下文中工作；
4. 成功标准如何在实现前确定，并避免 Agent 自证完成；
5. 如何让文档、代码、验证和运行事实持续同步；
6. 如何减少人工 Review 成本，而不是增加项目管理成本；
7. 如何把真正被验证/评审/合并的 ChangeSet 与 Agent Run 分离，避免把执行过程误当交付物；
8. 如何把软件项目中的 service/application/library/worker 等 SoftwareEntity 作为 Context、Build、Deploy、Failure 的共同锚点；
9. 如何把构建、发布、部署、运行故障重新关联回 Change、Task 与产品意图；
10. 如何把单次 FailureEvent 聚合成可管理 Defect，并识别 regression、suspect change 与必要的 Incident；
11. 如何逐步积累开发证据，让未来的 LLM/Agent 行为持续改善。

---

# 2. 产品原则 [ALL]

## 2.1 保留自由

用户可以从自然语言 Idea 开始，不需要先写 PRD、ADR 或 Task。

允许：

- 模糊想法；
- 多轮讨论；
- 快速改变方向；
- LLM 主动补全；
- Agent 自主探索实现；
- 在形成正式工程任务之前进行低成本方案迭代。

## 2.2 提供工程确定性

当某个 Idea 被确认要进入项目时，Tally 自动完成工程化：

- 产品文档化；
- 工程上下文化；
- Task 拆分；
- Scope 明确；
- Verification Contract 固化；
- Workspace 隔离；
- Evidence 收集；
- 风险导向 Review；
- Derived State；
- Failure Trace。

## 2.3 工程化不能重新引入传统流程负担

Tally 的目标不是让一个人像 20 人团队一样操作流程，而是：

> **让机器承担工程纪律，让人承担不可委托的判断。**

所有可自动生成、自动同步、自动推导的内容都不应要求人重复填写。

## 2.4 人负责意图和裁决，机器负责软件工程劳动

人主要负责：

- 表达 Idea；
- 产品取舍；
- 高风险架构决定；
- Requirement/Verification 冲突裁决；
- 高风险 Patch Review；
- Production 高风险确认；
- SpecDefect 最终判断。

LLM/Agent 尽可能负责：

- 产品分析；
- PRD/规格维护；
- 技术方案；
- Task 拆分；
- TaskContract；
- Verification Proposal；
- Context 收集；
- Coding；
- Testing；
- Debug；
- Fix；
- 文档同步；
- Review 预分析；
- Failure Diagnosis；
- Knowledge Proposal。

## 2.5 先 Triage，后正式化

不是每个自然语言输入都应立即成为 ProductChange、Requirement 或 Task。Tally 必须提供低成本 Triage，将输入先归类为：

- `ignore`；
- `duplicate/merge`；
- `attach_to_existing`；
- `fast_path_task`；
- `productize`；
- `escalate`。

Triage 的目标是降低正式工程对象的创建成本，而不是增加一个新的审批层。

## 2.6 Work 建模一次，Views 只是 Projection

Tally 的 Task、Change、Defect、Evidence 等事实对象只建模一次。看板、列表、Cockpit、`Needs My Decision`、`Ready to Merge` 等都只是对同一底层 Engineering Graph 的 Projection。

> **View 不拥有状态，也不能成为新的事实源。**

## 2.7 Differential Governance：默认治理当前变化

Tally 默认优先治理当前 `ChangeSet` 带来的新增或修改风险，而不是要求一个小任务顺便偿还整个遗留代码库的全部历史债务。

```text
Base State
   +
Current ChangeSet
   ↓
Differential Quality / Verification / Review Gates
```

只有当当前变化触发既有问题、扩大风险或项目策略明确要求时，治理范围才扩展到更大的 repository/system。

## 2.8 Desired Intent 与 Observed Evidence 分离

Tally 将版本化工程意图与运行事实长期分离：

```text
Desired Engineering State        Observed Engineering State
-------------------------        --------------------------
Product/Requirement              Code / ChangeSet
TaskContract                     Verification Result
VerificationContract             Build / Deploy Facts
Policy / Delivery Definition     Runtime / Failure Facts
```

Tally 的状态推导、本质检查与部分 reconciliation 都来自两者的比较，而不是让 LLM 直接宣布事实。

---

# 3. Authority Model [ALL]

## 3.1 权限主体

- **Intent Authority — Human Owner：** 决定为什么做、最终业务目标与关键取舍。
- **Proposal Authority — LLM：** 生成产品、设计、任务、验证、修复、文档等候选。
- **Execution Authority — Agent Runtime：** 在受控 Workspace 中执行代码、命令、测试与修改。
- **Verification Specification Authority — Tally + Verification Authority + Pre-Run Approval：** 在实现前定义“如何证明完成”并维护独立性边界。
- **Fact Authority — Evidence Engine：** 记录测试、构建、部署、故障、exposure 等机器事实。
- **Gate Authority — Policy + Human：** 决定高风险动作、Waiver、Merge、Deploy 等是否允许。
- **Evolution Proposal Authority — AdaptiveKit：** 基于 Evidence 提出 Behavior/Policy Review Proposal，但不拥有 Hard Policy Authority。

Authority 是**活动上下文的属性，不是永久绑定某个自然人的身份标签**。同一 Owner 可以在一个时刻承担 Intent/Gate Authority，在另一个时刻亲自贡献 implementation；一旦参与 implementation，其对应活动必须按 Implementation Authority 的信息边界处理。

## 3.2 Architecture Invariants

Architecture Invariant 是**可执行的系统约束**。每条使用稳定 ID，ADR、Task、architecture test 一律引用稳定 ID，而不是依赖显示序号。对应领域未激活时不要求提前实现无主语的测试；领域激活时必须同时落成 validator/test。

1. **`INV-ART-01` · P0** — Git owns versioned intent and engineering artifacts.
2. **`INV-FACT-01` · P1** — Machine facts are append-only whenever practical.
3. **`INV-STATE-01` · P1** — Derived state must be rebuildable.
4. **`INV-AUTH-01` · P1** — Agent Claim is never Project Fact.
5. **`INV-AUTH-02` · P1** — LLM output is Proposal unless independently validated or explicitly approved.
6. **`INV-VER-01` · P1 / post_hoc path P2** — Required verification authority must declare authoring provenance.
7. **`INV-VER-02` · P1** — Implementation authority cannot redefine verification authority.
8. **`INV-VER-03` · P1** — Protected verification evidence is immutable to the authorized implementation run.
9. **`INV-EXEC-01` · P0** — Agent-generated executable changes never run in the Owner primary worktree.
10. **`INV-STATE-02` · P1** — Every factual lifecycle state is evidence-traceable.
11. **`INV-VER-04` · P0/P1** — Spec defects enter review instead of being bypassed.
12. **`INV-GOV-01` · P5** — Adaptive behavior cannot automatically weaken hard policy.
13. **`INV-GOV-02` · P2** — Governance depth is monotonic with deterministic policy.
14. **`INV-CHANGE-01` · P1** — Every changed hunk must be accounted for before approval.
15. **`INV-CHANGE-02` · P1** — ChangeSet is the immutable unit of verification, review and integration.
16. **`INV-VIEW-01` · P2** — Views are projections, never authorities.
17. **`INV-STATE-03` · P1 dev / P4 delivery** — Desired intent and observed evidence remain separately traceable.
18. **`INV-VER-05` · P1** — Held-out evidence is mechanically unreachable within Tally-controlled and Runtime-Protocol-observable Implementation Authority channels.

### 3.2.1 Invariant 18 的可证明边界

`INV-VER-05` 的机械保证范围限定为：

- Tally-controlled：workspace、ContextPack、Holdout Evidence Store、validation output、Agent-visible event stream、knowledge index、historical failure retrieval；
- Runtime-Protocol-observable：session/thread history、tool output、Runtime 声明的 memory/scratchpad/capability。

第三方 provider 内部不可观测的 opaque retention / hidden state 属于 **declared residual risk**，不宣称能被 Tally 数学证明不存在。Runtime 通过 capability declaration + conformance evidence 获得 VerificationPlanner Role 资格。

## 3.3 Product Invariants

Product Invariant 用真实使用指标和 readiness gate 判断，不要求直接写成 architecture test。

1. **Preserve Vibe Coding freedom.** 治理深度必须与实际风险匹配，小改动不能被迫走与高风险 migration 相同的流程。
2. **Governance must pay for itself.** Tally 引入的人工审核、等待延迟和 Control Plane LLM 成本，必须被更低的人工作业量、更低缺陷风险或更高可信工程产出补偿。
3. **Human is a decision authority, not a process operator.** PRD、Task、Verification、文档、状态等可自动生成/推导内容，不应重新转嫁给 Owner 手工维护。
4. **Trust before automation, value before platform expansion.** P1 先证明可信控制；P2 再证明日常价值；多 Runtime、Operate、Adaptive 只能在前一层证据成立后扩展。
5. **Govern current change first.** 默认以 ChangeSet 为主要治理边界；不能因为引入 Tally 就强迫小改动承担与历史遗留代码等量的治理成本。
6. **Triage must reduce formalization cost.** Inbox/Triage 如果让用户比直接创建 Task 更费力，则设计失败。

核心产品 Guardrails：

- `human_control_minutes / verified_task`；
- `idea_to_verified_latency`；
- `control_plane_llm_cost_ratio`；
- `post_merge_regression`；
- 生命周期阶段耗时与成本分解。

# 4. 用户与使用规模 [ALL]

## 4.1 目标用户

首期目标不是大型组织，而是：

- 单个技术负责人；
- 小型研发团队；
- 同时维护多个内部或商业软件项目的人；
- 大量使用 Codex、Claude Code、OpenCode、Pi 等 Agent 的开发者；
- 希望提升 Vibe Coding 可靠性但不想引入重型流程的人。

## 4.2 Logical Responsibilities

产品内部可以讨论以下职责，但首版不把它们建模成 RBAC 角色：

- Product responsibility；
- Architecture responsibility；
- Development responsibility；
- Operations responsibility；
- Adaptive responsibility。

这些职责在单人使用时可以全部由同一 Owner 承担。

## 4.3 首版身份模型

只需要：

- `Owner`；
- `System`；
- `Agent`。

生产高风险操作使用 Policy + Owner explicit confirmation，不建立大型 RBAC。

---

# 5. 端到端生命周期 [P1 语义 / P2 Triage+FastPath]

Tally 管理的不是“Task 看板”，而是完整的 **Intent-to-Evidence Pipeline**。

```text
IDEA
  ↓
TRIAGE
  ↓
PRODUCTIZE
  ↓
DESIGN
  ↓
PLAN
  ↓
SPECIFY
  ↓
EXECUTE
  ↓
VERIFY
  ↓
REVIEW
  ↓
INTEGRATE
  ↓
RELEASE
  ↓
OPERATE
  ↓
LEARN
```

## 5.0 Risk-aware Intake 与 Pipeline Fast Path

十三个生命周期阶段是**语义阶段**，不是要求每个 Idea 都串行调用十二个 Agent。Tally 必须先用 Deterministic Risk Rules + 一次轻量 LLM 分析确定最低治理深度。

```text
Idea / Change Request
        ↓
Deterministic Intake Policy
        +
Lightweight LLM Analysis
        ↓
┌─────────────────────────────────────────────┐
│ LIGHT          STANDARD          STRICT     │
│                                             │
│ 直接 Task      PLAN + SPECIFY    全流程/显式审查 │
│ + 预定义验证     + 必要产品上下文     + 高风险 Gate   │
└─────────────────────────────────────────────┘
```

Deterministic policy 决定**最低**：

- risk level；
- Task profile；
- required stages；
- explicit review requirements。

LLM 只能建议提高治理深度，不能降低它。概念上：

```text
effective_governance
=
deterministic_minimum ⊔ llm_suggestion
```

阶段只能由 deterministic policy 判定为：

- `required`；
- `satisfied_by_existing_artifact`；
- `not_applicable`。

例如一个已明确的问题修复可直接满足 PRODUCTIZE 的最低条件，但必须同时满足项目规则，例如：

- 被判定为 `light`；
- 不引入新的 user-visible surface；
- 引用已有 Product/Requirement context；
- 不修改 schema/public API/security-sensitive path。

LLM 可以否决 Fast Path 并升级流程，不能单独批准 Fast Path。

运行中也必须重新评估风险。如果 Light Run 实际触发 migration、auth/security、dependency/public API change、protected path 或超过项目阈值，Tally 必须**停止当前 Run**并升级 profile，不能原地补文书后继续以同一个 Implementation Run 执行。

升级语义分两类：

- **Light → Standard**：当前 candidate patch 被标记为 `quarantined/untrusted proposal`；VerificationPlanner 只接收 Task/Product/Requirement、`base_commit` 代码和触发升级的风险信号，**不得接收 candidate diff 或 candidate code**。新建的 required binding 标记 `authoring_mode: post_hoc`，必须经过 explicit verification review、完成 Qualification 并冻结，然后开启**新的 CodegenRun**。旧 candidate 可以在新 Run 中作为实现提议重新应用，但不携带任何验证权威。
- **Light/Standard → Strict**：丢弃当前 candidate，回到 `base_commit`；重新建立完整 VerificationContract，并以 `authoring_mode: pre_commit` 完成 explicit verification review、Qualification 与 Freeze 后开启 fresh CodegenRun。Strict 路径不得复用已观察过的 candidate 作为被验证实现。

`post_hoc` 是异常升级路径，不得与正常 `pre_commit` evidence 混为同一可信等级；其使用率、审核时间与结果必须单独度量。

## 5.1 IDEA

输入可以是：

- 一个新产品想法；
- 一个功能需求；
- 一个 bug；
- 一个性能问题；
- 一个运行故障；
- 一个现有项目的改进方向；
- 一段自然语言讨论。

该阶段允许完全非结构化。

## 5.2 TRIAGE

所有 Idea、Bug 报告、Failure、改进建议和 Agent escalation 首先进入统一 Triage 语义。Triage 的职责不是“审批需求”，而是决定当前输入是否值得正式化，以及最短的工程路径是什么。

Triage 可以产生：

- `ignore`：无后续动作；
- `duplicate/merge`：合并到现有 Idea/ProductChange/Defect；
- `attach_to_existing`：关联现有 Requirement/Task；
- `fast_path_task`：现有产品语义已经充分，直接生成轻量 Task/Verification；
- `productize`：进入 Product Proposal；
- `escalate`：触发更高治理深度或 Owner Decision。

Triage 使用 Deterministic Policy 决定最低治理要求，LLM 只负责补充语义、发现关联或建议升级，不能降低规则给出的最低要求。

首版 Triage 不需要通用规则编辑器；使用项目 Policy + LLM Proposal + Saved View 即可。

## 5.3 PRODUCTIZE

Product LLM 结合：

- 当前用户输入；
- 已有产品文档；
- 当前代码与功能；
- 历史任务；
- 已知运行事实；

生成 Product Proposal，包括：

- 用户目标；
- 使用场景；
- 范围；
- 功能设计；
- 非功能要求；
- 边界条件；
- 风险；
- 待确认问题；
- 与现有产品的变化。

Product Proposal 必须支持多轮提问和修订，而不是 one-shot PRD generation。

## 5.4 DESIGN

Engineering/Architecture LLM 将已确认 Product Intent 转成：

- 方案设计；
- 关键架构影响；
- 数据/API 变化；
- Migration 方案；
- 兼容性；
- 风险；
- ADR Proposal；
- Verification Strategy。

高风险架构决策必须 Owner Review。

## 5.5 PLAN

Planner LLM 将 Product + Engineering Intent 拆解成 Task Proposal：

- Task；
- Task dependency；
- Scope；
- Acceptance；
- Risk；
- 可能影响模块；
- Verification needs。

Tally deterministic validator 检查：

- 引用存在；
- Task ID 合法；
- DAG 无环；
- 关键产品目标有 Task 覆盖；
- 高风险变化有显式 gate；
- 无明显 orphan task。

## 5.6 SPECIFY

在真正 Coding 前，把 Task Proposal 固化为：

- TaskContract；
- VerificationContract；
- ContextPack；
- AgentProfile；
- ExecutionPolicy。

这一阶段是 Engineering Gate。

## 5.7 EXECUTE

Agent Runtime 在 Disposable Workspace 中完成：

- 代码阅读；
- 实现计划；
- Coding；
- Agent 自增测试；
- Debug；
- Fix；
- 文档更新；
- 局部验证。

## 5.8 VERIFY

Tally 使用实现前被冻结的 VerificationContract 运行确定性验证。

Implementation Agent 的“测试通过”只是一种 Claim；真正的验证结果由 Tally Validator 产生。

## 5.9 REVIEW

Review 重点不是重新读整个 Git Diff，而是按风险、Acceptance 和保护证据组织：

- Acceptance coverage；
- Protected evidence integrity；
- Scope violations；
- High-risk changes；
- Agent-added tests；
- Documentation impact；
- Remaining risks。

LLM 做 Review 预分析，Owner 做高价值判断。

## 5.10 INTEGRATE

完成：

- Commit；
- Merge；
- CI；
- Build；
- Artifact。

## 5.11 RELEASE

完成：

- Deploy；
- Migration；
- Rollback capability；
- environment evidence。

## 5.12 OPERATE

收集运行反馈：

- Build failure；
- Deploy failure；
- Runtime failure；
- Manual incident；
- API/log adapter failure；
- health signals。

## 5.13 LEARN

完成：

- Failure knowledge；
- Development evidence；
- Behavior comparison；
- 人工调整 BehaviorManifest；
- 后续 AdaptiveKit candidate/evaluation（仅 readiness 达标后）。

---

# 6. 产品 Artifact 模型 [P1 core / P2 extended]

## 6.1 Git-first Artifact

项目仓库推荐：

```text
<project>/
├── src/...
├── tests/...
└── .tally/
    ├── project.yaml
    ├── product/
    │   └── PRD.md
    ├── requirements/
    │   └── REQ-001.md
    ├── architecture/
    │   ├── ARCHITECTURE.md
    │   └── ADR-001.md
    ├── tasks/
    │   ├── TASK-001.yaml
    │   └── TASK-002.yaml
    └── policies/
        └── project.yaml
```

首版 Requirement 和 ADR 不建立复杂 Domain；以 Markdown + stable ref + hash/revision 为主。

## 6.2 Artifact 主要由 LLM 维护

上述文档不是要求 Owner 手工维护，而是：

```text
Code/Product Change
      ↓
Impact Detection
      ↓
LLM Documentation Proposal
      ↓
Validation / Risk Review
      ↓
Git Patch
```

目标是 Code-Document Co-evolution。

## 6.3 文档自动同步与 Integration Gate

当任务涉及以下变化时，Tally 可以触发 Documentation Update Proposal：

- API change；
- Schema change；
- Config change；
- Architecture change；
- User-visible product change；
- Deployment procedure change。

**文档语义正确性不属于 `VERIFIED` 的 deterministic correctness gate。** `VERIFIED` 只表示实现满足 VerificationContract。

文档同步属于 `APPROVED/MERGED` 前的 **Integration Readiness**。规则的唯一 Authority 是 **Project Integration Policy**，建议版本化存放于 `.tally/policies/project.yaml`。TaskContract 只声明 `documentation_impact`，不能自行定义或放宽 Integration Gate。

Project Policy 可以使用窄而机械的路径一致性规则，例如：

```yaml
integration_gates:
  documentation_sync:
    triggers:
      - internal/api/**
      - migrations/**
    targets:
      - docs/api/**
```

机械 Gate 只判断“要求的同步产物是否存在有效变化”；文档内容是否真正表达正确，由 ReviewAgent 辅助、Owner 按风险判断。具体 meaningful-diff 判定策略进入 ADR / Implementation Spec，不在 PRD 固化。

## 6.4 Engineering Object Model

Tally 不以“一个万能 Issue”承载所有语义，而将不同软件工程事实拆成少量明确对象。核心对象分层如下：

```text
PRODUCT INTENT
Idea → ProductChange → Requirement → ArchitectureDecision

SOFTWARE CONTEXT
Project → SoftwareEntity

ENGINEERING PLAN
TaskContract + VerificationContract

CHANGE
CodegenRun → ChangeSet (0..1 candidate snapshot per Run) → Commit

VERIFICATION
VerificationRun → VerificationEvidence

INTEGRATION
IntegrationCandidate → ApprovalEvidence → Merge

DELIVERY
BuildArtifact → Release → Deployment → Rollout/Observation

OPERATE
FailureEvent → Defect → Incident(optional)

LEARN
Knowledge + BehaviorRevision
```

### 6.4.1 SoftwareEntity

`Project` 是产品/代码库管理边界；`SoftwareEntity` 是可独立理解、构建、部署或运行的软件单元，例如：

- application；
- service；
- worker；
- library；
- CLI；
- data pipeline。

对于 monorepo，一个 Project 可以包含多个 SoftwareEntity。Task、ContextPack、Build、Deploy、Failure 可以关联一个或多个 entity。

项目可以通过轻量 `.tally/project.yaml` 声明实体，不建设大型 CMDB：

```yaml
project: tally
entities:
  - id: server
    type: service
    path: cmd/server
  - id: worker
    type: worker
    path: cmd/worker
  - id: web
    type: application
    path: web
```

### 6.4.2 ChangeSet

`CodegenRun` 描述 Agent 怎样工作；`ChangeSet` 描述最终要被验证、评审和合并的实际代码变化。

```text
TaskContract
    ↓
├── CodegenRun #1 → ChangeSet-A [superseded]
├── CodegenRun #2 → ChangeSet-B [superseded]
└── CodegenRun #3 → ChangeSet-C
                       ├→ Verification
                       ├→ Quality/Policy Gate
                       ├→ Patch Review
                       └→ Integration
```

Task 是多个 Run / ChangeSet revision 的聚合点；ChangeSet 本身是 immutable candidate snapshot，不跨 Run 累积。

ChangeSet 至少固定：

- `base_commit`；
- `candidate_commit` 或 patch hash；
- files/hunks；
- dependency/schema/API delta（可推导时）；
- affected SoftwareEntity；
- risk signals。

### 6.4.3 VerificationRun

`VerificationContract` 定义“应该验证什么”；`VerificationRun` 表示某个 contract revision 对某个 ChangeSet 的一次实际执行；`VerificationEvidence` 是该次执行产生的事实。

```text
VerificationContract rev 3
          ↓
VerificationRun #203
          ↓
Evidence
├── AC-01 PASS
├── AC-02 PASS
├── Build PASS
└── Architecture PASS
```

### 6.4.4 FailureEvent / Defect / Incident

三者必须分离：

- `FailureEvent`：一次观测到的机器/运行事实；
- `Defect`：一个或多个 FailureEvent 聚合形成的工程问题；
- `Incident`：需要主动响应、恢复和复盘的高影响运行事件。

不是每个 FailureEvent 都产生 Defect，也不是每个 Defect 都升级为 Incident。

## 6.5 Object Activation Matrix

长期领域模型存在于 PRD，不等于当前阶段必须实现。未激活对象/能力不得成为更早阶段的隐式依赖或退出条件。

```text
对象 / 能力                                         激活阶段                    P1
---------------------------------------------------------------------------------------
TaskContract / Acceptance Independence Requirement   P1                         YES
VerificationContract / Run / Evidence                P1                         YES
Evidence Visibility / Holdout Isolation Capability   P1                         YES
CodegenRun / immutable ChangeSet                     P1                         YES
HumanApprovalEvidence / minimal Waiver provenance    P1                         YES
ExecutionPolicy / ContextPack                        P1                         MINIMAL
Minimal Workflow Kernel                              P1 vertical / P2 generalized PARTIAL

Triage / Fast Path / 完整 Task Profiles              P2                         NO
Holdout Failure Recovery Automation                  P2                         NO
ProductChange workflow / Task DAG                    P2                         NO
SoftwareEntity / IntegrationCandidate                P2                         NO
Documentation Integration Gate / Saved View          P2                         NO
LifecycleSpan full analysis                          P2                         NO*

Pi / OpenCode Runtime                                P3                         NO

BuildArtifact / Release / Deployment / Rollout       P4                         NO
FailureEvent / Defect / Incident / DeliveryState     P4                         NO*

BehaviorManifest Observe                             P5                         NO
Adaptive candidate / promotion                       P6 conditional             NO
```

`NO*`：可以提前保留现有原始事件/能力，但不能成为阶段退出条件。P1 Task Pack 默认只消费 `[ALL]/[P0]/[P1]` 语义。

## 6.6 Work 与 Views 分离

Tasks、Changes、Defects 等底层对象只维护一份。Board、列表、`Needs My Decision`、`Ready to Merge`、`Verification Weak`、`Regressed` 等都是 Saved Views。

新增 View 不应新增状态字段，也不应要求同步维护第二份对象。

---

# 7. Task 模型 [P1 standard+isolation / P2 profiles+recovery]

## 7.1 Task 的定位

Task 不是用户 Idea 的直接翻译，而是：

> **Idea 经过 Productize、Design、Plan 后得到的正式工程执行单元。**

Task 必须足够明确，能够被 Agent 执行，也能够被独立验证。

## 7.2 Task Profiles

Task Profile 是治理深度的一部分，由 deterministic intake policy 给出最低等级，LLM 只能升级。

### `light`

适合：

- 小型 UI/文案；
- 已有产品语义下的局部低风险 bug；
- 小型 deterministic change；
- 不涉及 schema/public API/security/dependency/production-critical path 的修改。

特点：

- 可以省略详细 Acceptance clauses；
- 使用预定义 validation profile；
- 默认要求 scope check + focused validation + complete diff accounting + Review；
- 正式 Fast Path 在 P2 激活；
- 一旦运行中风险上升，必须按 §5.0 停止当前 Run 并升级，不得在同一个 Implementation Run 内事后补齐 Verification 后继续执行。

### `standard`

适合普通功能和 bug：

- 明确 Objective；
- Acceptance clauses；
- VerificationContract；
- Scope；
- Documentation impact；
- Final Validation。

`standard` 是否要求 Source / Information Independence 由 Project Verification Policy 与 acceptance risk 决定。P1 reference workflow 使用 `standard`，并至少验证一次真实 held-out isolation vertical scenario。

### `strict`

适合：

- migration；
- security/auth；
- destructive data change；
- architecture invariant；
- production critical logic；
- 其他 deterministic policy 判定的高风险修改。

额外要求：

- protected verification evidence；
- stronger Evidence Qualification；
- architecture/security gates；
- explicit verification review；
- explicit owner patch review；
- 必要时 rollback/recovery test；
- deterministic verification policy 必须让至少一个 Acceptance Clause 的 `independence_required >= source`；
- 对安全、认证、迁移、数据完整性、不可逆操作、negative/“绝不能发生”类 Acceptance，Policy 可要求 `independence_required: information`。

`strict` 的通用 Approval Review **不能**用来自动满足某条 Acceptance 的 Independence Requirement。只有针对具体 Acceptance 的专门 verification evidence，且满足对应 independence 要求，才可参与 independence coverage。

## 7.3 TaskContract 示例

```yaml
id: TASK-023
profile: standard

title: Add stale job lease recovery

objective: >
  Prevent jobs from remaining permanently in running state
  after worker crashes.

product_refs:
  - .tally/product/PRD.md#job-reliability

requirement_refs:
  - .tally/requirements/REQ-018.md

architecture_refs:
  - .tally/architecture/ADR-012.md

depends_on:
  - TASK-019

scope:
  allowed_paths:
    - internal/jobs/**
    - internal/store/**
    - migrations/**

  forbidden_paths:
    - internal/auth/**

acceptance:
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source

  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information

  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none

documentation_impact:
  - architecture
  - schema
```

`independence_required` 属于 Acceptance Clause，而不是单条 Evidence Binding。它是累积等级：

```text
none < source < information
```

- `none`：不要求独立性；
- `source`：至少有一条 applicable evidence 的 `effective_source_independence = true`；
- `information`：同时满足 source independence 与 information independence。

## 7.4 TaskContract 的生成与维护

默认流程：

```text
Product/Engineering Plan
        ↓
Planner LLM
        ↓
TaskContract Proposal
        ↓
Tally static validation
        ↓
Risk classification
        ↓
Low risk: compact review / policy accept
High risk: explicit owner review
        ↓
Versioned TaskContract
```

Owner 不应成为 TaskContract 的主要作者。

TaskContract 只定义 **WHAT / WHY / SCOPE / ACCEPTANCE / DEPENDENCY / RISK / IMPACT**。`max_iterations`、token/wall budget、network、shell、credentials、retry、runtime/model escalation 与 stop conditions **不属于 TaskContract**，统一由 ExecutionPolicy 管理。Task 需要特殊运行条件时，只能产生 ExecutionPolicy Proposal/Override Request，由 Policy 层决定是否接受。

# 8. Verification Contract [P1 core / P2 recovery automation]

## 8.1 目的

VerificationContract 解决 Tally 最核心的可信问题：

> **Implementation Authority 不能在实现之后重新定义“什么叫完成”，也不能通过看见、修改、反复筛选或替换 Evidence 来为自己制造假绿。**

Verification 需要同时回答五类相互独立的问题：

1. **Completeness**：Acceptance 是否有 required evidence / required gates；
2. **Authoring & Source Independence**：验证标准是否独立于当前 implementation decision；
3. **Integrity**：protected/frozen evidence 是否被篡改；
4. **Qualification**：test evidence 是否具有最低机械区分能力与 change relevance；
5. **Information Independence**：高风险 Acceptance 是否存在 Implementation Authority 无法利用其内容或 verdict 迭代的独立 Evidence。

Evidence Qualification 和 Independence 都是工程可信度机制，不宣称机械证明完整业务语义。高风险语义仍需要 Verification Review / Owner Decision。

## 8.2 Verification Authority Independence

对于 `standard` / `strict` Task：

1. required evidence 必须记录 `authoring_mode: pre_commit | post_hoc`；
2. 正常路径使用 `pre_commit`：在其所授权的 Implementation CodegenRun 之前定义、Qualification、review 并冻结；
3. `post_hoc` 只允许用于 Light → Standard 的运行中治理升级，并必须 candidate-blind、explicit review、new CodegenRun；
4. `strict` required evidence 必须使用 `pre_commit`；
5. protected evidence 的引用、protection mode、baseline hash、execution protocol 在冻结后不可由其所授权的 Implementation Run 修改；
6. test-based required evidence 必须执行 baseline qualification；
7. Implementation Agent 新增测试默认属于 supplementary evidence；
8. Agent 若认为 Requirement/Acceptance/Evidence 有问题，必须产生 `SpecDefectProposal`；
9. 修改 VerificationContract 必须形成新 revision，旧事实不得被覆盖；
10. fresh held-out binding **不绕过**本节任何 authoring provenance 规则。

## 8.2.1 Verification Authoring Provenance

- `pre_commit`：目标 Implementation Run 开始前定义，Verification authoring context 不包含该 Run 的 candidate implementation；默认和最高可信路径。
- `post_hoc`：仅运行中治理升级后允许。VerificationPlanner 必须 candidate-blind，Owner explicit review，且 binding 只能授权新的 Run。

`post_hoc` 不等于“事后补文档继续跑”。原 Run 必须停止。

## 8.2.2 Source Independence 是活动/Provenance 独立，而不是身份不同

Evidence Binding 记录 `independence_basis.source`，它只是 Desired provenance basis，例如：

```text
verification_authority
pre_existing
human
external_system
implementation_authority
```

真正的 `effective_source_independence(binding, changeset)` 相对于该 ChangeSet 的实际 Implementation Authority provenance 推导。

原则：

- 同一个自然人历史上写过 test，不意味着今天修改代码时该 pre-existing test 必然 source-dependent；
- 不同的人/Agent 也不天然意味着 source-independent；
- 判断的是 Evidence authoring activity 是否由当前 candidate / implementation decision 影响；
- `authoring_mode: pre_commit`、candidate-blind `post_hoc`、历史 revision、actual implementation contributors 都是 provenance 输入。

> **actor category ≠ authority independence。**

## 8.3 Evidence Qualification

### 8.3.1 Baseline sensitivity

对于 test evidence，冻结前应在 `base_commit` 上实际执行。

| Baseline observed class | 机械证明强度 | 最低审核要求 |
| --- | --- | --- |
| `assertion` | 较强：测试已执行到行为断言且断言不成立 | compact verification review 可用 |
| `compile_error` | 弱：可能只引用了尚不存在的符号 | explicit verification review |
| `panic` / `error` | 弱：失败不一定与 Acceptance 语义有关 | explicit verification review |
| baseline `pass` | 无 red-green sensitivity | explicit verification review + rationale |
| `n/a` | 不适用 test red-green | evidence-specific policy |

### 8.3.2 Candidate green

Implementation 完成后，required test evidence 必须在其 applicable candidate revision 上满足冻结的 execution protocol，且 protected hash 保持一致。

### 8.3.3 Evidence-to-change relevance

对于 `type: test` 的 required evidence，默认需要最低 `change_overlap` 信号：

```text
lines covered by required test
        ∩
changed executable lines
        ≠ ∅
```

这不是 coverage KPI，也不是语义证明。新增文件、generated/glue code、语言级 provider 等细节进入 ADR。

### 8.3.4 Failure fingerprint

Failure fingerprint 只记录实际 baseline/verification execution 的 observed evidence。不得要求 LLM 预先预测 fingerprint。

## 8.4 Evidence Visibility 与 Holdout Capability

### 8.4.1 Visibility 是 Contract Policy，Exposure 是 Fact

Evidence Binding 声明：

```yaml
visibility_policy: visible | held_out
```

`visibility_policy` 冻结后不因运行事实改写。

- `visible`：Implementation Authority 可读，可进入 incremental validation；
- `held_out`：内容与可识别 verdict 不得进入 Tally-controlled / Runtime-Protocol-observable Implementation Authority 通道。

held-out evidence 存放于逻辑 **Holdout Evidence Store**。PRD 只规定唯一必要性质：Implementation workspace 无法访问内容。普通 repository tracked 目录 + `forbidden_paths` 不构成隐藏机制。

### 8.4.2 Role Separation 必须落到 Session Separation

VerificationPlanner 与 CodingAgent/DebugAgent 不得共享 runtime session、thread history、persistent memory 或 shared scratchpad。Runtime Adapter 必须能为不同 Authority 建立互不可检索的会话；不能保证该性质的 Runtime 不得获得 VerificationPlanner Role。

Runtime Conformance Test 至少包含 Session Isolation Canary。

### 8.4.3 ContextPack 使用 redacted projection

Implementation ContextPack 携带的是 VerificationContract 的 **redacted projection**。对 `held_out` binding，只可暴露完成任务所必需的 Acceptance 语义与 evidence type，不暴露 store/ref、selector、测试内容、baseline 细节或 hidden failure 输出。

`context_hash` 在 redaction 后计算。Historical Failures、Verified Knowledge、tool history、retrievable artifacts 同样受 `INV-VER-05` 约束。

## 8.5 Acceptance-level Independence Model

Independence Requirement 属于 **Acceptance Clause**，不是某一条 Evidence Binding。

```text
none < source < information
```

满足关系：

- `none`：不要求独立性；
- `source`：该 Acceptance 的 applicable evidence set 中至少一条 `effective_source_independence = true`；
- `information`：至少一条 applicable evidence 同时满足 effective source independence 与 effective information independence。

Information Independence 永远不能补救 Source Dependence。

### 8.5.1 Planned Independence Coverage

Freeze 时计算：

```text
planned_independence_coverage(acceptance)
=
该 Acceptance 的 planned evidence set 中
至少存在一条其 basis/capability 足以满足 independence_required 的 binding
```

对所有 `independence_required != none` 的 Acceptance 全称检查。设计阶段缺失 → **FREEZE FAIL，无 Waiver**。

### 8.5.2 Effective Independence Coverage

执行后计算：

```text
effective_independence_coverage(acceptance, changeset)
=
该 Acceptance 当前 applicable evidence set 中
至少存在一条 effective independence 满足 independence_required 的 evidence
```

运行时因 exposure/selection contamination 导致 planned ✓ → effective ✗，才可能进入 Independence Waiver。

### 8.5.3 Evidence Strength 与 Independence 正交

`baseline: pass` 的弱测试不会因为 held-out 变强；held-out 提供信息边界，不提供 proof strength。Review/metrics 使用 strength × visibility / independence 的交叉读数，不把 held-out 显示成“更强测试”的等级徽章。

## 8.6 Exposure Model

### 8.6.1 Content Exposure

held-out 内容或诊断进入 Implementation Authority 可见通道时，记录 append-only Exposure Fact。它不回写 VerificationContract。

典型 channel：

```text
debug_disclosure
session_leak
validation_log
knowledge_index
owner_disclosure
manual
```

内容进入持久/全局通道时，对该 binding 的 information independence 永久失效；局部暴露至少污染对应 change lineage。

Owner 是决策角色，不是天然“安全观察者”。Owner 查看 hidden 结果后若向 implementation lineage 贡献 commit、prompt、Task/Verification revision、AgentProfile/ExecutionPolicy/model 选择或其他运行输入，必须按 Implementation Authority exposure 处理。Owner 的主观声明只能作为 risk acceptance annotation，不能恢复 epistemic independence。

### 8.6.2 Selection Exposure

即使没人看见 hidden 内容，也不能把同一 held-out verdict 反复当作 candidate selection oracle。

机械判定规则：

> 某 binding 在某 objective/change lineage 上第一次 held-out verdict 发生后，任何**未在该 verdict 之前固定**的 descendant candidate，默认 `selection_exposed = true`。

唯一例外：在首次 verdict **之前**已经存在可验证的 precommit，且后续严格遵守：

- candidate generation schedule；
- candidate count；
- selection criteria（仅 visible criteria）；
- retry conditions。

无法机械证明未受 hidden verdict 影响时，默认取污染侧。

### 8.6.3 Fresh Binding 不是洗白机制

第一次 held-out verdict 一旦产生，该 binding 已完成其在该 implementation decision 中的独立判决使命。后续 repaired candidate 若要重新获得 information-independent final verification，需要 fresh independent binding。

Fresh binding：

- 必须形成新的 VerificationContract revision；
- 完整继承 Authoring Provenance、Source Independence、Qualification、Freeze 规则；
- `strict`：discard/quarantine candidate → clean base → mint pre_commit fresh binding → freeze → NEW Run；
- `standard`：只允许 controlled candidate-blind `post_hoc` + explicit review + NEW Run。

fresh ≠ independent。

## 8.7 Verification Execution Integrity

### 8.7.1 Verdict 绑定 immutable candidate

Information-independent held-out verdict 绑定于：

```text
immutable candidate ChangeSet
+ verification binding revision
+ frozen execution protocol
+ verification environment identity
```

held-out evaluation 后对可执行代码的任何修改都会产生新的 ChangeSet，旧 verdict 不适用于新 ChangeSet。

### 8.7.2 同 candidate 重跑必须预承诺 stopping rule

同一 candidate 的重复执行只有在 repetitions、stopping rule、aggregation 在首次执行前冻结且与 observed outcome 无关时，才可作为同一次 verification protocol 的组成部分。

禁止：

```text
FAIL → rerun → FAIL → rerun → PASS → cherry-pick PASS
```

随机 seed 必须是冻结 protocol 的一部分。改变语义相关 verification environment 不是“同一次 rerun”。

## 8.8 Evidence Validity、Applicability 与 Binding Shopping

### 8.8.1 Failed Evidence 是 append-only Fact

一条合法 frozen evidence 对 immutable ChangeSet 给出的 FAIL 是历史事实。新的 binding 通过不能抹掉旧的合法 FAIL。

同一 Candidate 的 failed binding 只有在 confirmed `holdout_defect`、SpecDefect、invalid-evidence 裁定等带 provenance 的正式流程后，才失去自身 validity。

### 8.8.2 Validity ≠ Applicability

Evidence 可以永久 valid 但不再 applicable。

例如 Requirement/Acceptance 被 Owner 正式修订或 supersede 后，旧 FAIL 仍保留为历史事实，但只 gate 它绑定的 Desired/Verification revision；State Engine 对当前状态只消费 **current desired revision + applicable evidence**。

不得把合法 Intent Revision 误称为 Evidence Invalidation。

### 8.8.3 Functional Gate 与 Independence Coverage 分离

同一 Acceptance 可以有多条 Evidence：

```text
AC-02  independence_required: information

HOLDOUT-1
  functional gate: PASS on repaired ChangeSet
  information independence: compromised

HOLDOUT-2
  functional gate: PASS
  information independence: independent
```

HOLDOUT-1 仍保留 regression value；HOLDOUT-2 可以满足 Acceptance-level Information Independence。Independence 不按 evidence 数量计数，而按 Acceptance coverage 判断。

## 8.9 Holdout Failure Recovery Boundary

P1 只证明：隔离有效、hidden final validation 能执行、FAIL 阻断 VERIFIED、Exposure 会使 effective independence 失效。

P2 才自动化 Recovery Ladder：acceptance-level diagnosis、必要披露、binding retirement/renewal、policy review 等。

Fresh evidence 不是无限 retry 机制。Verification revision / fresh-binding recovery 必须受现有 Execution/Workflow budget 的 stop boundary 约束；预算耗尽进入 `NEEDS_REVIEW`。具体 `max verification revisions / fresh bindings / wall time / control-plane cost` 表达进入 ADR / ExecutionPolicy Spec。

## 8.10 Independence Waiver

Waiver 只能覆盖**运行期 independence 损失**，不能覆盖 Freeze 时已经能发现的 planned coverage 缺失。

Project Verification Policy 决定 Waiver 是否可用，可按 risk category 禁用。允许时必须产生带 provenance 的 HumanApprovalEvidence，至少记录：

```text
compromised acceptances / bindings
exposure refs
coverage gap
reason code
owner risk acceptance
```

验证结论区分：

```text
strict_satisfied
strict_waived
```

`strict_waived` 不得在任何 UI/报表中表述为“Strict requirement 已满足”。

## 8.11 VerificationContract 示例

```yaml
task_id: TASK-023
revision: 6

acceptances:
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_owner_test.go
          selector: TestReclaimStaleLease
        protection:
          mode: immutable
          baseline_hash: sha256:...
        baseline:
          expectation: { status: fail, preferred_class: assertion }
          observed:
            status: fail
            class: assertion
            fingerprint: sha256:...
            commit: abc123
        qualification:
          baseline_sensitivity: strong
          change_overlap: required
          review_mode: compact

  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information
    evidence_bindings:
      - id: EV-AC02-H1
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_safety#TestNeverReclaimLiveLease
        protection:
          mode: immutable
          baseline_hash: sha256:...
        execution_protocol:
          repetitions: 1
          aggregation: all_pass
        baseline:
          expectation: { status: pass }
          observed: { status: pass, commit: abc123 }
        qualification:
          baseline_sensitivity: none
          change_overlap: required
          review_mode: explicit
          rationale: Negative safety invariant; held out for information independence.

  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none
    evidence_bindings:
      - id: EV-AC03-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: retry-budget-v1

implementation_runtime_requirements:
  session_isolation: true
  held_out_unreachable: true
  conformance_class: verification-isolation-v1

required_gates:
  build:
    required: true
    command:
      argv: [go, build, ./...]
      cwd: .
      timeout: 120s
```

实际 Runtime identity/version/conformance evidence 不写入 frozen VerificationContract，而由 CodegenRun / authoring provenance 记录。

## 8.12 Evidence 类型与 Agent 新增测试

Evidence 类型包括 test、validator、build、schema check、static rule、human verification、deployment verification、runtime observation 等。不同类型有自己的 qualification policy。

Implementation Agent 新增测试默认属于 supplementary evidence。提升为未来 required evidence 只能通过新的 Verification revision，对后续 Run 生效。

## 8.13 VerificationRun 与 Triage Authority

每一次针对具体 ChangeSet 的执行形成 VerificationRun，保留历史 Run。

`holdout_triage_outcome` 是诊断分类，不是原始机器事实：

- LLM 只能产生 `proposed_outcome` + provenance；
- `flake/environment` 可在满足 deterministic protocol 时机械确认；
- 其余 outcome 由 Owner/独立 Evidence 确认；
- 产品指标与 AdaptiveKit 只消费 `confirmed_outcome`，未确认条目计入 `unconfirmed`。

Canonical Failure Fingerprint 在 P1 Debug 与 P4 FailureEvent 共用同一语义/实现，算法进入 ADR。

# 9. LLM Agent Roles [P1 Coding+Debug / P2 其余]

## 9.1 Role 与 Runtime 分离

`AgentRole` 是逻辑软件工程职责；`AgentRuntime` 是执行后端。

一个 Runtime 可以承担多个 Role；一个 Role 可以被不同 Runtime/Model Profile 执行。

## 9.2 首批逻辑 Role

- `ProductAgent`：Idea → Product Proposal；
- `ArchitectureAgent`：技术设计与 ADR Proposal；
- `PlannerAgent`：Work Breakdown / Task DAG；
- `VerificationPlanner`：Acceptance → Evidence binding proposal；
- `CodingAgent`：实现代码；
- `DebugAgent`：Failure hypothesis / fix；
- `DocumentationAgent`：文档协同更新；
- `ReviewAgent`：风险和 diff 预审；
- `FailureAgent`：运行故障诊断；
- `KnowledgeAgent`：可验证知识提议。

## 9.3 Role 不等于 Multi-Agent Swarm

首版不要求每个 Role 都是独立 Agent process。

允许：

```text
一个 Codex Runtime
+ 不同 Role Prompt/Behavior/Profile
```

Tally 的价值在于 Contracts、Evidence、Validators 和 State Machine，而不是让大量 Agent 相互聊天。

---

# 10. Agent Runtime Protocol [P1 Fake+Codex / P3 其余]

## 10.1 目标

Tally 不绑定单一 Coding Agent。Codex、OpenCode、Pi 等作为可替换 Runtime。

## 10.2 Minimal Core

统一接口只要求：

```text
Descriptor
Start
Execute
Cancel
Events
Result
```

不把 Codex 高级能力直接写入核心接口。

## 10.3 Capability Negotiation

可选 capability 示例：

```text
session.resume
session.fork
session.isolation
turn.steer
turn.interrupt
event.streaming
output.structured
sandbox.native
tool.approval
tools.custom
model.select
session.persist
memory.declared
```

Tally 使用 Capability，而不是 runtime-name conditionals。

VerificationPlanner 与 Implementation Authority 的 Role separation 必须能落到独立 session/thread/memory boundary。无法提供或无法通过 conformance 的 Runtime 不获得相应 Verification Role 资格。

## 10.4 首版 Runtime

P1：

```text
FakeRuntime   - conformance / abstraction test
CodexRuntime  - production runtime
```

FakeRuntime 应故意缺失部分高级 capability，确保 Core 未泄漏 Codex-specific assumption。

P1 Runtime Conformance 必须包含 Session Isolation Canary：向 Verification session 注入唯一 canary，在新的 Coding session 中尝试通过所有 Protocol-observable retrieval path 获取；成功检索即不满足 Verification Isolation capability。

Canary 是 conformance evidence，不是对 provider 内部不可观测状态的绝对证明。

## 10.5 Runtime 扩展阶段

P3 再正式加入 Pi / OpenCode 等 production adapter。顺序不是产品依赖，只是实施成本控制。

# 11. AgentProfile 与 ExecutionPolicy [P1 minimal]

## 11.1 AgentProfile

Agent Role、Runtime、Model、Behavior 必须分离。

```yaml
id: go-standard-cheap
role: CodingAgent
runtime: { type: codex }
model: { profile: economical }
behavior: { revision: go-backend-r12 }
capabilities:
  required: [event.streaming]
  preferred: [output.structured]
```

未来可以形成：

```text
Task characteristics × Role × Runtime × Model × Behavior
```

但首版不做复杂 Router。

## 11.2 ExecutionPolicy

ExecutionPolicy 是 **Tally Control Plane 约束的唯一 Authority**：

```yaml
execution_policy:
  retry:
    max_iterations: 4
  budgets:
    max_tokens: 180000
    max_wall_time: 30m
  escalation:
    after_failed_iterations: 2
    target_profile: strong-coding
  stop_conditions:
    - protected_evidence_modified
    - forbidden_path_modified
    - governance_profile_upgrade_required
```

ExecutionPolicy 负责 retry、wall/token budget、network/shell/credential、stop conditions、runtime/model escalation、verification recovery stop boundary。

TaskContract / BehaviorManifest / AgentProfile 只能提出需求或偏好，不能重复拥有或放宽同类限制。

### 11.2.1 Escalation 终止当前 CodegenRun

`runtime / model / behavior_revision` 在 CodegenRun 中是单值事实。发生 escalation 时：

```text
current CodegenRun → status: escalated → terminate
                                      ↓
                                 NEW CodegenRun
```

新 Run 记录 `escalated_from`，不得在同一 Run 内替换 runtime/model。CodegenRun **may produce** candidate ChangeSet；没有固定 candidate revision 的 crash/budget-exhausted Run 不产生 ChangeSet。

Behavior/model 归因只使用 homogeneous Run；`mean_iterations` 按 Run，`escalation_rate` 按 Task。

# 12. Context Engine [P1]

## 12.1 ContextPack 是核心基础设施

Agent 质量取决于上下文，而不是单纯 Prompt。Context Engine 同时承担 **relevance selection** 与 **authority-aware projection**：不是“上下文越多越好”，而是“对当前 Authority 足够且不过权”。

## 12.2 ContextPack 内容

```text
TaskContextPack
├── TaskContract
├── VerificationContract (redacted projection)
├── Relevant Product Docs
├── Relevant Requirement Docs
├── Relevant ADR / Architecture
├── Dependency Tasks
├── Repo Map
├── Relevant Symbols / Files
├── Existing Visible Tests
├── Recent Relevant Commits
├── Historical Failures (visibility-filtered)
├── Verified Knowledge (visibility-filtered)
└── Execution Policy
```

## 12.3 上下文裁剪与 Visibility Filter

必须记录：included refs、excluded refs、selection reason、token/size budget、context hash。

对 `visibility_policy: held_out` 的 binding，Implementation ContextPack 只暴露其允许的 redacted projection。`context_hash` 在 redaction/filter 后计算。

Historical Failures、Verified Knowledge、Repo Map、Existing Tests、tool/session history 都不得成为 held-out leakage side channel。任何新的 Context 来源默认受 `INV-VER-05` 约束，而不是“只有列出的几种才需要过滤”。

# 13. Workspace 与执行隔离 [P1]

## 13.1 主原则

Agent 不允许直接修改 Owner 正在工作的 primary worktree。

## 13.2 Disposable Worktree

```text
Repository
   ↓
git worktree add --detach <base_commit>
   ↓
Tally Workspace
   ↓
Agent Runtime
   ↓
Implementation / Debug / Docs / Tests
   ↓
Final Validation
   ↓
Patch Review
   ↓
Commit Proposal / Merge
   ↓
Destroy Workspace
```

一个 CodegenRun 内多次 iteration 复用同一个 disposable worktree，不重复创建。

## 13.3 双层 Sandbox

```text
Tally-owned workspace / OS boundary
          +
Agent-native sandbox (when available)
```

Agent 自带 sandbox 仅作为 defense-in-depth，不是 Tally 唯一安全边界。

## 13.4 Dependency Cache Strategy

Disposable Worktree 不能意味着每次重新下载所有依赖。

缓存按 toolchain + dependency lock hash + platform 建立。

示例：

```text
Go
├── GOMODCACHE
└── GOCACHE

Node
├── pnpm store / npm cache
└── dependency cache keyed by lock hash

Python
├── uv / pip cache
└── venv cache keyed by lock hash

Rust
├── cargo registry
└── compiler/build cache
```

项目可声明 workspace bootstrap profile。

---

# 14. Validation Strategy [P1]

## 14.1 Incremental Validation

每轮 Agent 修改后优先运行 affected/focused validation：

- focused **visible** required/protected tests；
- affected package tests；
- typecheck/lint/static rules when configured；
- scope/protected-path checks。

held-out evidence 不作为 Implementation Agent 的迭代反馈。P1 采用 only-final held-out execution；P2 以后若引入 Control-Plane-only early probe，其 disclosure/selection semantics 必须遵守 §8。

## 14.2 Final Validation

最终 `VERIFIED` 之前至少检查：

- all applicable required acceptance functional gates；
- planned/effective independence coverage；
- held-out evidence execution（结果只进入允许的 Control/Review Authority）；
- Evidence Qualification final checks；
- protected evidence hash；
- required build/architecture/security/schema gates；
- ChangeSet identity 与 frozen verification execution context；
- scope/forbidden path；
- applicable unresolved failed evidence = 0。

Documentation semantic sync 不阻断 `VERIFIED`；它属于 Integration Readiness。

## 14.3 Validation Result 是 Fact

Agent 的“我认为通过”只是 Claim。Validator/VerificationRun 记录结构化 command、exit code、stdout/stderr digest、runtime、Task/Verification/ChangeSet revision、validator version、environment identity、qualification/independence results 等才是 Evidence。

held-out 原始失败输出不得进入 Implementation Agent 可见事件流。

# 15. CodegenRun [P1]

每一次正式 Agent 执行必须成为一等对象。

## 15.1 核心字段

```text
id
project_id
task_id
role
runtime
runtime_version
model
behavior_revision
cost_plane

base_commit
task_revision
verification_revision
context_pack_hash

workspace_id
agent_session_id

started_at
finished_at
status
iterations
escalated_from

input_tokens
output_tokens
cost

result_patch_hash
agent_claim
validation_result
failure_reason
runtime_capability_snapshot
conformance_evidence_ref
```

## 15.2 为什么必须记录 revision

必须可以回答同一个 Task 为什么某次成功、某次失败，并能定位：code baseline、TaskContract、VerificationContract、ContextPack、Runtime/Model/Behavior、conformance evidence。

## 15.3 ChangeSet 是 immutable candidate snapshot

CodegenRun 是过程；ChangeSet 是被 Verification / Review / Integration 的固定对象。

一个 Run **may produce** 一个 candidate ChangeSet；没有固定 candidate 的 Run 不产生 ChangeSet。ChangeSet 固定至少：

```text
base revision
candidate revision / patch identity
changed files/hunks
created_at
producer run
```

ChangeSet 不跨 Run 累积，不在验证/Review 中原地变形。后续重做产生新的 ChangeSet。

取代关系用 append-only fact：

```text
ChangeSetSupersededFact
  old
  new
  reason: escalation | rework | spec_defect
```

UI 可派生 `superseded_by`，不得修改 immutable snapshot 本体。

## 15.4 ChangeSet 与 Verification Verdict

held-out verdict 只对其绑定的 immutable ChangeSet + Verification revision + execution protocol + environment identity 生效。任何 executable-code mutation 都产生新 ChangeSet，并需要新的 applicable verification evidence。

## 15.5 IntegrationCandidate

P2 激活。`VERIFIED` ChangeSet 与最新目标分支基线 + required Integration Gates 组成 IntegrationCandidate；并行变化需要 latest-target revalidation。

# 16. Unified Agent Events [P1]

Runtime Adapter 将不同 Agent 的 native event 归一化成：

```text
SESSION_STARTED
TURN_STARTED
MESSAGE
TOOL_REQUESTED
TOOL_STARTED
TOOL_COMPLETED
TOOL_FAILED
FILE_READ
FILE_CHANGED
COMMAND_STARTED
COMMAND_COMPLETED
APPROVAL_REQUESTED
USAGE_UPDATED
TURN_COMPLETED
SESSION_COMPLETED
SESSION_FAILED
SESSION_CANCELLED
```

Event 同时保留：

- normalized type；
- native type；
- raw payload；
- sequence；
- timestamp。

Adapter 不允许伪造底层 Runtime 没有提供的精细事件。

---

# 17. Minimal Workflow Kernel [P0 lease / P1 vertical / P2 generalized]

Tally 自建**领域受限的最小工作流内核**，不引入通用 Workflow Engine。

### P0/P1 vertical subset

- job lease / stale recovery；
- 单 Task 顺序执行；
- CodegenRun 状态迁移；
- validation；
- wait-for-approval；
- cancellation；
- ExecutionPolicy retry/escalation/stop；
- verification freeze / final gate / NEEDS_REVIEW。

P1 必须有最小可替换、可测试的 orchestration abstraction，但不得实现通用 workflow DSL。

### P2 generalized kernel

再加入 Task DAG、ready selector、跨 Task 依赖、multi-task autopilot、更多 retry/recovery orchestration。

不支持 arbitrary business orchestration / generic saga framework。

## 17.1 Job Lease

Job 至少具有：

```text
status
worker_id
locked_at
lease_until
attempts
max_attempts
next_attempt_at
idempotency_key
```

Worker heartbeat；stale lease 安全 reclaim。

# 18. Task State 与 Delivery State [P1 Task / P4 Delivery]

Task execution 与 delivery 是两个不同的状态机。

## 18.1 TaskExecutionState

```text
PLANNED
READY
RUNNING
CANDIDATE
VALIDATING
VERIFIED
APPROVED
MERGED
```

异常态：

```text
BLOCKED
FAILED
NEEDS_REVIEW
CANCELLED
```

状态不得由 Agent 或 UI 直接写入，必须根据 Artifact + Evidence 推导。

`VERIFIED` 至少要求：当前 Desired/Verification revision 的所有 applicable required functional gates 通过；planned/effective independence coverage 满足 Policy 或存在被允许且显式记录的 Waiver；protected/holdout integrity 无 unresolved violation；没有 applicable unresolved failed evidence。

`APPROVED` 虽然来自人的决策，但实现方式仍然是：

```text
Owner decision
   ↓
HumanApprovalEvidence
   ↓
State Engine
   ↓
APPROVED
```

即 **human approval record 是一种 Evidence**，而不是 `UPDATE task SET status='APPROVED'`。

## 18.2 DeliveryState

**P1-P3 中 Delivery subsystem 不作为新架构的激活域，UI 不应显示虚假的 BUILT/DEPLOYED/STABLE 状态。P4 激活 Build/Release/Deployment/Failure/Delivery facts 后，DeliveryState 才进入正式 derived-state 计算。**

```text
NOT_BUILT
BUILT
RELEASED
DEPLOYING
DEPLOYED
ROLLING_OUT
STABLE
DEGRADED
ROLLED_BACK
```

一个 Task `VERIFIED` 不代表 Change 已 `MERGED`，一个 Merge 不代表已经 `RELEASED`，一个 Release 也不代表任何环境已经 `DEPLOYED/STABLE`。Release、Deployment 与 Task completion 必须保持独立语义。

**P1–P3 期间只启用 TaskExecutionState。** DeliveryState 在 P4 `Operate & Failure Loop` 激活后，才由 BuildFact、DeployFact、FailureEvent 与 StabilityPolicy 推导。MVP UI 不显示没有事实来源的 DeliveryState。

## 18.3 Stability Policy

项目/环境可以声明：

```yaml
stability_policy:
  window: 30m
  conditions:
    - no_new_correlated_failure: critical
    - healthcheck: healthy
```

或首版：

```yaml
stability_policy:
  mode: manual
```

STABLE 是 Delivery 状态，不是 Task 状态。

---

# 19. Patch Review [P1]

Patch Review 是 MVP 核心能力，不是附属 Tab。Review 的输入必须是固定 revision 的 `ChangeSet`，而不是正在变化的 Agent workspace。

## 19.1 Review 目标

Owner 的主要瓶颈不是 Agent 会不会写代码，而是：

> **如何快速判断 Agent 是否在正确范围内、按正确目标完成了可靠修改，同时确保没有任何改动在 Review 中“消失”。**

## 19.2 Deterministic Hunk Inventory

ReviewAgent 运行前，Tally 必须先从目标 diff 生成确定性的完整 Hunk Inventory。

```text
Git Diff
  ↓
Deterministic Hunk Inventory
  ↓
Rule-based accounting + LLM residual classification
  ↓
Completeness Validator
```

每个 hunk 必须有**一个 primary review bucket**，可以同时拥有多个 semantic tags（例如同时关联 AC-01、migration、risk）。

任何未能被规则或 ReviewAgent 合理归属的 hunk 进入：

```text
Unattributed Changes
```

`Unattributed Changes != ∅` 时，Owner 不得提交 Approve。这样 ReviewAgent 即使漏判，也不能制造“所有改动都已经看过”的假覆盖感。

哪些 bucket 由 deterministic rule 直接分配、哪些交给 LLM 处理残差，属于 Review Engine ADR/Implementation Spec；PRD 只要求 deterministic inventory 与 accounting completeness。

## 19.3 Acceptance-oriented Review

Review 至少组织以下信息：

### Acceptance Coverage

显示每个 AC：

- 相关代码；
- required evidence；
- Evidence Qualification/验证结果；
- Review Agent explanation。

### Protected Evidence

突出显示：

```text
LOCKED
expected hash
actual hash
UNCHANGED / MODIFIED
```

### Scope Violations

显示 forbidden path、unexpected file、large change、generated artifact anomaly 等。

### Risky Changes

优先突出 migration、auth/security、public API、schema、dependency、config、production behavior。

### Agent-added Tests

单独列出 supplementary tests，不与 protected acceptance evidence 混淆。

### Documentation Changes

显示 Integration Readiness 所需的文档同步和 Review 状态。

### Unattributed Changes

必须显式为 0 才允许 Approve。

### Remaining Risks

Review LLM 给出风险摘要，但风险结论仍是 Proposal。

## 19.4 Review 的核心指标

至少测：

- `diff_review_active_minutes`；
- `diff_review_wall_minutes`；
- `manual_correction_minutes`；
- `unattributed_hunk_count`；
- `unsafe_change_detection_rate`。

人工时间必须由产品自动测量，而不是要求 solo developer 手工记账；具体 activity/idle 规则进入 ADR。

---

# 20. SpecDefect [P0/P1]

当 Agent 发现：

- Requirement 与代码事实冲突；
- Acceptance 本身错误；
- protected test 错误；
- Architecture constraint 不合理；

不得修改对应 hard gate 绕过验证。

流程：

```text
Agent detects conflict
        ↓
SpecDefectProposal
        ↓
NEEDS_REVIEW
        ↓
Owner / Product / Architecture decision
        ├── specification update
        ├── verification update
        └── reject proposal
        ↓
new revisions
        ↓
new CodegenRun
```

---

# 21. Failure Intake 与运行闭环 [P4]

OPERATE 不能从“Failure Evidence 已存在”开始，Tally 必须定义 FailureEvent 的入口。

## 21.1 Failure Sources

首版支持：

- build failure；
- deploy failure；
- runtime/API report；
- manual report；
- log adapter；
- healthcheck adapter。

Tally 不试图成为 TB 级日志平台，只接收或抽取结构化故障事实。

## 21.2 Failure Pipeline

```text
Failure Source
      ↓
Normalize
      ↓
Deterministic Fingerprint
      ↓
FailureEvent
      ↓
Group / Deduplicate
      ↓
Defect
      ↓
Correlate
├── project / SoftwareEntity
├── environment / deployment / release
├── artifact / commit / ChangeSet
├── task / codegen_run
└── first_seen / last_seen / regression
      ↓
Suspect Change + Known Verified Fix Search
      ↓
Diagnosis / Fix Proposal
      ↓
Incident（仅高影响时）
```

## 21.3 Defect、Regression 与 Suspect Change

`FailureEvent` 是事实，`Defect` 是工程管理对象。多个相同或高度相关的 FailureEvent 可以聚合到同一个 Defect。

Defect 至少应能表达：

- fingerprint/grouping key；
- first_seen / last_seen；
- affected SoftwareEntity/environment；
- related release/deployment；
- suspected ChangeSet/commit；
- regression 状态；
- fixed_by Task/ChangeSet；
- resolution。

如果已经 resolved 的 Defect 在后续 release/deployment 再次出现，Tally 应标记为 `REGRESSED`，而不是创建一个完全失去历史关系的新问题。

Suspect Change 是分析/Proposal，不是 Fact；最终归因仍需要 Evidence 或 Review。

## 21.4 Incident

高影响 Defect 可以升级为 Incident。Incident 关注的是“响应、恢复、影响与复盘”，不是替代 Defect。

```text
FailureEvent → Defect → Incident(optional)
                   ↓
                Fix Task
                   ↓
               Verification
```

首期不建设 PagerDuty 式大型 Incident Command 系统；只保留 `severity / started_at / recovered_at / impact / postmortem_ref` 等最小领域语义。

## 21.5 Failure Diagnosis

Failure Agent 只能在 Disposable Worktree 上生成 fix candidate。

固定基线：

```text
exact failure-related commit
```

每个 candidate 在干净或可恢复 baseline 上验证，不能让失败 candidate 污染下一个 candidate。

## 21.6 Structured Repro

不使用 shell string + `strings.Fields` 表示 repro。

采用：

```yaml
command:
  argv:
    - go
    - test
    - ./internal/jobs/...
  cwd: .
  env: {}
  timeout: 120s
  shell: false
```

只有明确 `shell: true` 才允许 shell 语义。

---

# 22. Evidence Graph 与 Desired/Observed Reconciliation [P1 dev / P4 delivery]

Tally 将产品意图、工程计划、实际 Change、验证、交付与运行事实关联为一个可追溯 Engineering Graph：

```text
Idea / ProductChange
       ↓
Requirement / ArchitectureDecision
       ↓
TaskContract + VerificationContract
       ↓
CodegenRun(s)
       ↓
immutable ChangeSet revision(s)
       ↓
VerificationRun / Evidence
       ↓
IntegrationCandidate / Commit
       ↓
BuildArtifact / Release / Deployment
       ↓
FailureEvent → Defect → Incident(optional)
       ↓
Fix Task / Verified Fix / Knowledge
```

Tally 同时保持两类节点可独立定位：

```text
Desired
Product/Requirement/Task/Verification/Policy

Observed
Change/VerificationResult/Build/Deploy/Failure
```

Derived State 本质上是对指定 revision 的 Desired 与 Observed Evidence 的比较。

所有 project-specific node 必须带 `project_id` namespace；SoftwareEntity-specific Fact 还应带 entity reference。Commit Node 至少使用：

```text
(project_id, commit_sha)
```

而不是单纯 `commit_sha`。Trace stale edge 清理必须按完整 tuple 判断，不允许按分列存在性误判。

Engineering Graph 的链接应尽可能自动推导；LLM 可以 Proposal semantic edge，但不能把未验证关系直接写成 Fact。

---

# 23. Git 写入模型 [P0]

## 23.1 Single Git Writer

Tally Server、Worker、Knowledge、Docs Agent 不能同时直接操作同一 Git index。

所有 Git write 通过统一 GitWriter / Job serialization。

至少需要：

- per-repo mutex；
- cross-process lock；
- explicit base revision；
- atomic apply semantics。

避免不同并发操作被意外提交到同一个 commit。

---

# 24. AdaptiveKit - Observe First [P5/P6]

## 24.1 首期定位

AdaptiveKit 首阶段不做自动进化。

定位为：

> **Development Evidence Observatory + Versioned Behavior Management**

收集：

- runtime；
- model；
- behavior revision；
- context strategy；
- iteration；
- validation failures；
- human correction；
- cost；
- post-merge defect；
- success/failure。

Owner 通过查询/对比观察规律，并手工修改 BehaviorManifest。

AdaptiveKit 可以基于 confirmed Evidence 产生 `VerificationPolicyReviewProposal` 等 Review Proposal，但**不得直接生成并应用更弱的 hard policy revision**。降低验证独立性必须由 Owner explicit approval；不得仅以成本为理由。

## 24.2 BehaviorManifest

```yaml
behavior: go_backend_feature
revision: r13

planning:
  strategy: requirement-first-v3

context:
  selector: symbol-impact-v5
  include_recent_failures: true
  include_migration_tests: true

implementation:
  strategy: evidence-first

validation:
  incremental: affected-first
  final: project-required

review:
  strategy: acceptance-first-v2
```

Git 提供：

- diff；
- revision；
- rollback；
- audit。

## 24.3 自动 Evolution 的 Readiness Gate

只有满足以下条件后才进入自动 Candidate/Replay/Promotion：

- 足够的同类 CodegenRun；
- 任务类别可分组；
- baseline/eval dataset 可维护；
- Behavior changes 可归因；
- evaluation cost 可接受；
- 人工观察已经证明存在重复规律。

## 24.4 Replay 的首要用途

未来 Replay 优先作为 **regression evaluation**，不是直接做统计显著性 A/B。

维护一组 Golden Development Cases：

```text
Go backend bug
Migration
API change
UI feature
Refactor
Failure repair
```

新 Behavior 首先必须保证关键历史能力不 regression。

---

# 25. Human Control Overhead 与 Product Value Measurement [P1 measure / P2 gate]

Tally 是否成功不能只看 Agent 成功率。系统必须自动记录 Owner 真实控制成本和端到端等待成本。

至少记录：

```text
contract_review_minutes
verification_review_minutes
diff_review_active_minutes
diff_review_wall_minutes
manual_correction_minutes
spec_defect_decision_minutes
idea_to_verified_latency
```

同时记录生命周期阶段分解：

```text
Intake
Product/Planning
Verification Planning
Queue
Workspace Bootstrap
Agent Execution
Incremental Validation
Final Validation
Human Review
Integration
```

## 25.1 North Star Metric

> **Verified Engineering Output per Human Review Minute**

即：

```text
可靠完成的软件工程产出
÷
人实际用于审核、裁决和修正的时间
```

这是主指标，但必须同时观察 `idea_to_verified_latency`，避免出现“人工只花 3 分钟，但整个小修复等 40 分钟”的产品失败。

## 25.2 Human Authored Artifact Ratio

追踪：

- PRD/Product Spec 由人直接写多少；
- TaskContract 人工重写多少；
- VerificationContract 人工修改多少；
- 文档同步人工写多少。

目标不是强制为 0，而是持续减少“本可以由系统自动完成的人工流程劳动”。

## 25.3 LLM Cost Attribution

LLM 成本分为：

**Control Plane**：

- Intake；
- Product/Architecture governance；
- Planning；
- Verification planning；
- Review assistance；
- orchestration/context reasoning。

**Execution Plane**：

- Coding；
- Debug/Fix；
- implementation test generation；
- Documentation generation；
- implementation-oriented reasoning。

文档生成是实际工程产出，不计入 Control Plane。

每次 LLM/Agent 调用在发生时写入 `cost_plane: control | execution`；报表不得根据未来可能变化的 Role 映射重算历史归属。

关键指标：

```text
control_plane_llm_cost_ratio
=
control_plane_llm_cost / total_llm_cost
```

如果控制 AI 的成本长期超过 AI 实际工程产出的成本，需要优先简化 Pipeline，而不是继续增加 Agent Role。

## 25.4 Readiness Gate 分层

**P1 Trust Readiness** 只回答“是否可信可控”，不因人工时间暂时较高而停止路线。

**P2 Product Value Readiness** 才回答“是否值得日常使用”。P2 使用真实任务进行 Comparative Usage Trial，必须按 `light / standard / strict` profile 分维，不能把三种不同治理形态混成一个总体平均数。

Trial 的**主要产出是生命周期阶段耗时与成本分解**，用于回答慢在 Intake/Product/Planning、Workspace Bootstrap、Agent、Validation、Build 还是 Human Review；单一 Tally vs Bare Agent 比值只是辅助信号。

至少观察：

- `human_minutes / verified_task by profile`；
- `diff_review_active_minutes by profile`；
- `idea_to_verified_latency by profile`；
- `post_merge_regression by profile`；
- `manual_correction_minutes by profile`；
- `control_plane / execution_plane LLM cost`；
- 各生命周期阶段 wall-clock / active-time breakdown。

可采用 matched/alternating 的真实任务比较 Tally 与 Bare Agent，不把少量高度异质任务包装成严格统计实验。

产品参考 Gate：在可比较 profile 内，Tally 的 median human time 不应长期高于 Bare Agent 的约 `1.2×`，同时 regression 不得更差；如人工略高但缺陷明显降低，应由 Owner 基于完整 phase/cost/quality breakdown 决策，而不是由单一阈值自动终止产品。

---

# 26. UI 信息架构 [P1 minimal / P2 complete]

Tally 的导航按 Owner 的工作意图组织，而不是按数据库表组织。目标是让一个人管理十几个项目时优先看到“需要我决定什么”和“什么变化正在发生”。

## 26.1 核心导航

```text
Inbox
Projects
Tasks
Changes
Failures
Views
```

### Inbox

统一承接需要 Triage/Owner Decision 的输入：

- Idea；
- duplicate/merge proposal；
- SpecDefect；
- Agent escalation；
- weak Verification review；
- Incident/critical Defect。

### Projects

展示 Product/Docs、SoftwareEntity、Current Work 与基本 Health，不建设大型 CMDB/Cockpit。

### Tasks

展示 Task、Profile、State、current run、blocked reason、verification summary 与 human attention required。

### Changes

这是 Agent 开发期间最重要的工作区，围绕 ChangeSet 展示：

- CodegenRun timeline；
- VerificationRun；
- Hunk Inventory；
- Acceptance mapping；
- protected evidence；
- risky/unattributed changes；
- Integration readiness。

### Failures

以 Defect 为主要工作对象，向下可以展开 FailureEvent，向上可以看到 release/deployment/suspect Change/fix Task。

### Views

Saved View 是 Projection，不复制对象。首批建议：

- `Needs My Decision`；
- `Agent Running`；
- `Verification Weak`；
- `Ready to Merge`；
- `Risky Changes`；
- `Regressed`；
- `Production Failures`。

## 26.2 P1 MVP UI

P1 为控制范围只实现：

```text
Tasks
Changes / Run Detail
Patch Review
```

Inbox、Failures、Saved Views 在 P2/P4 随对应领域启用，不要求 P1 一次完成完整导航。

## 26.3 Patch Review

Patch Review 必须基于 ChangeSet 和 Deterministic Hunk Inventory。UI 可以用 Acceptance/Risk 分组提高理解效率，但不能隐藏未归属 hunk。

## 26.4 Product/Design UI

Product/Requirement/Architecture 首期继续使用 Git Markdown + Proposal Diff Review；Tally 可以提供 View/Edit/Proposal，但不复制传统大型 Requirement Management UI。

---

# 27. Product Proposal 与文档工作流 [P2]

## 27.1 Idea → Product Proposal

用户输入 Idea 后，ProductAgent：

1. 读取当前项目产品文档；
2. 读取相关代码/功能索引；
3. 判断新建还是修改现有功能；
4. 提问最少必要问题；
5. 生成 Product Change Proposal；
6. 做一致性检查；
7. 展示文档 diff；
8. Owner Accept / Edit / Reject。

## 27.2 Product Docs 不是静态前置条件

如果项目初始文档很差，Tally 可以从代码和现有行为反向建立当前产品事实，再逐步补全 PRD，而不是要求先人工补全文档才能使用。

## 27.3 文档版本关系

每次 Task/CodegenRun 应记录其依赖的：

- product revision；
- requirement revision；
- architecture revision。

避免文档漂移导致历史不可解释。

---

# 28. Planning 与 Task Decomposition [P2]

## 28.1 Planner 输入

Planner 读取：

- Product Proposal；
- Architecture Proposal；
- Repository structure；
- relevant SoftwareEntity / entity dependencies；
- current tasks；
- existing related failures；
- known verified knowledge。

## 28.2 Planner 输出

```text
Task Proposal Set
├── Task objectives
├── dependencies
├── scope
├── risk
├── acceptance
├── verification needs
└── expected documentation impact
```

## 28.3 人工审核策略

不是所有 Task 都逐条全文审核。

Tally 根据风险聚合：

```text
LOW   → bulk/compact review
MED   → normal review
HIGH  → explicit review
```

目标是让 Owner 审“异常和关键决策”，不是维护任务列表。

---

# 29. Debug 与 Fix 生命周期 [P1]

Debug 是正式工程生命周期，不是简单重复 prompt。

```text
Validation Failure
      ↓
Failure Evidence
      ↓
Debug Hypothesis
      ↓
Fix Candidate
      ↓
Focused Validation
      ↓
Regression Validation
      ↓
Success / New Evidence
```

每次 Debug 必须记录 failure fingerprint、hypothesis、attempted fix、result、related files、validation evidence。

Debug 使用与 P4 FailureEvent 相同的 Canonical Failure Fingerprint 语义；算法只实现一次，具体 canonicalization 进入 ADR。

达到 ExecutionPolicy retry/escalation boundary 时：

- 有 target profile/runtime：当前 CodegenRun `status: escalated` 并终止，开启新 Run；
- 无目标或再次失败：`NEEDS_REVIEW`。

不得在同一个 Run 内替换 model/runtime，也不得无限循环。

held-out/fresh-binding recovery 的自动化从 P2 开始，且同样受 Execution/Workflow budget stop boundary。

# 30. LLM 与 Agent 可观测性 [P1]

记录 `llm_run` / `codegen_run` 至少包括：

- use_case / role；
- `cost_plane: control | execution`（**调用发生时写入，不在查询时根据 Role 推导**）；
- provider / runtime / model；
- prompt/behavior revision；
- context hash；
- token；
- latency；
- cost；
- status；
- failure type；
- human intervention；
- authoring/conformance provenance when applicable。

`cost_plane` 是历史事实字段；否则未来 Role → Plane 映射变化会重写历史 `control_plane_llm_cost_ratio`。

Prompt 全文是否持久化由 Privacy Policy 决定；至少保存 revision/hash 和必要 redacted metadata。held-out 内容不得因为 telemetry/log persistence 绕过 Visibility Policy。

# 31. Eval Strategy [P1 admission / P5 protocol]

## 31.1 Schema / Safety Eval

首先验证：

- structured output valid；
- refs valid；
- DAG no cycle；
- no hallucinated task refs；
- no protected evidence mutation proposal；
- no forbidden-path mutation；
- no direct authoritative state claim accepted。

这是 Proposal Admission，不是产品运行指标。

## 31.2 Planning / Codegen Evaluation Protocol

指标定义统一见 §38；本节不重复定义指标。

§31 的职责是 **Admission Methodology 与 Evaluation Protocol**：

- Proposal 准入；
- eval dataset 的构成与时效性；
- baseline 选取与重建；
- execution environment / flake protocol；
- 历史数据污染规避；
- Behavior revision 可比性条件。

P1-P4 只强制 §31.1；后五项主要在 P5 Adaptive Observe 前通过 ADR 落成。

# 32. 安全模型 [P1]

## 32.1 默认原则

- Agent workspace isolated；
- network deny by default for coding runs unless task requires；
- secrets are capability-scoped；
- production credentials never exposed to normal codegen agent；
- shell commands structured where practical；
- destructive commands policy-gated；
- high-risk approval requires Owner。

## 32.2 Hard Policy 与 Adaptive Behavior 分离

Hard Policy：

- security gates；
- protected evidence；
- production permission；
- secret access；
- owner-required approvals。

Adaptive Behavior：

- prompt；
- context strategy；
- planning strategy；
- model routing；
- iteration budget；
- debug strategy；
- review summarization。

AdaptiveKit 不得修改 Hard Policy。

---

# 33. 数据与存储原则 [P1]

## 33.1 Git

适合存：

- PRD；
- Requirements；
- Architecture/ADR；
- TaskContract；
- VerificationContract；
- BehaviorManifest；
- project policy；
- accepted docs patches。

## 33.2 PostgreSQL

适合存：

- CodegenRun；
- AgentEvent；
- validation facts；
- build/deploy facts；
- FailureEvent；
- job queue；
- derived indexes；
- trace edges；
- usage/cost；
- human review timing。

## 33.3 Derived Index

Git artifact 建立 DB index：

```text
project_index
task_index
verification_index
behavior_index
```

所有 ingest 必须验证 project identity，避免 ghost project facts。

---

# 34. 推荐 Monorepo 结构 [P1]

Tally 自身作为多模块项目时优先使用 Git monorepo：

```text
tally/
├── apps/
│   ├── server/
│   ├── worker/
│   └── cli/
│
├── modules/
│   ├── project/
│   ├── artifacts/
│   ├── task/
│   ├── verification/
│   ├── context/
│   ├── evidence/
│   ├── trace/
│   ├── failure/
│   ├── workflow/
│   ├── review/
│   ├── runtime-protocol/
│   └── adaptivekit/
│
├── runtimes/
│   ├── fake/
│   ├── codex/
│   ├── pi/
│   └── opencode/
│
├── behaviors/
├── policies/
├── evals/
├── migrations/
├── tests/
│   ├── architecture/
│   ├── integration/
│   ├── e2e/
│   └── runtime-conformance/
└── docs/
```

首期只要求 `fake/` 和 `codex/` 实现完成。

---

# 35. Runtime Conformance Tests [P1 / P3 extend]

任何 Runtime Adapter 必须通过通用测试：

- can start；
- receives cwd/workspace；
- can execute task；
- can emit events；
- can cancel；
- returns result；
- process crash handled；
- primary worktree isolation preserved；
- usage recorded when available；
- capability declaration truthful。

Capability-specific：

```text
if resume → resume works
if fork → fork isolation works
if structured output → schema enforced
if native approval → deny actually blocks action
if verification isolation → Session Isolation Canary passes
```

### Session Isolation Canary

向 VerificationPlanner session 注入唯一 canary，然后以 Coding/Debug Role 在同一项目/Task 上尝试从 Runtime-Protocol-observable session/thread/memory/tool-history path 检索。任何成功检索都判定该 Runtime 不具备对应 Verification Isolation conformance。

Canary 只证明 Protocol-observable 范围的隔离，不证明 provider 内部不可观测状态不存在。

# 36. MVP：Vertical Trust Loop [P1]

MVP 不追求完整 SDLC，也不在 P1 证明经济收益。P1 只证明：**Tally 能可靠约束一个真实 Coding Agent，独立定义并验证成功标准，形成 immutable ChangeSet，完整 Review 所有代码变化，并把控制成本测出来。**

P1 reference workflow 使用 `standard` Task。Light Fast Path / 完整 profiles / holdout recovery automation 在 P2 才正式激活；但 **Evidence Visibility / Session Isolation / hidden final validation 是 P1 Trust Capability**。

## 36.1 MVP 三条主线

### A. Trustworthy Execution

必须包含：

1. 一个现有 Project 注册；
2. Markdown Product/Requirement/ADR refs；
3. TaskContract；
4. VerificationContract；
5. Evidence Qualification；
6. Authoring Provenance + planned independence coverage；
7. pre-run freeze；
8. ContextPack redacted projection；
9. Disposable Worktree；
10. Dependency Cache；
11. FakeRuntime；
12. CodexRuntime；
13. Session Isolation Canary；
14. CodegenRun；
15. Agent events；
16. immutable ChangeSet；
17. VerificationRun + Evidence；
18. visible Incremental Validation；
19. held-out final execution；
20. Final deterministic validation；
21. protected evidence / exposure / applicability checks。

### B. Trustworthy Review

- deterministic Hunk Inventory；
- complete hunk accounting；
- Acceptance functional + independence coverage；
- protected evidence；
- scope/risk；
- supplementary tests；
- `Unattributed Changes` blocking Approve；
- HumanApprovalEvidence / Waiver provenance when policy permits。

### C. Measurement

自动记录 human control/review time、wall-clock stage latency、Agent/model token/cost、control/execution plane、final validation outcome、verification strength/visibility/independence distributions。

## 36.2 MVP 明确不做

- 完整 Requirement Domain；
- ADR Management Domain；
- Product Cockpit；
- production Pi/OpenCode adapter；
- 多 Task Autopilot；
- 自动 Runtime Router；
- holdout failure Recovery Ladder 自动化；
- early holdout probe；
- Adaptive Evolution；
- DeliveryState / production lifecycle UI；
- 大型 RBAC；
- 通用 Workflow DSL。

## 36.3 P1 Trust 验收

主路径：

```text
Task/Change Intent
  ↓
TaskContract + Acceptance Independence Requirements
  ↓
Qualified + Frozen VerificationContract
  ↓
Redacted Context + Disposable Workspace
  ↓
CodegenRun
  ↓
Immutable ChangeSet
  ↓
VerificationRun / visible + held-out Evidence
  ↓
Functional + Independence + Integrity Gates
  ↓
Complete Patch Accounting
  ↓
HumanApprovalEvidence
  ↓
Verified / Failed / Needs Review
```

Trust Gate：

- primary worktree contamination = 0；
- protected evidence silent modification = 0；
- scope violation escaping validator = 0；
- known false VERIFIED = 0；
- CodegenRun / Task / Verification / ChangeSet traceability = 100%；
- Hunk accounting completeness = 100%；
- Session Isolation Canary 在 FakeRuntime 与 CodexRuntime 通过；
- 至少一个真实 `standard + held_out` Task 端到端执行；
- 至少一次构造 holdout failure 阻断 VERIFIED；
- 至少一次 exposure fact 让 effective information independence 变为 compromised 并在 Review 可见；
- first-pass validation baseline 可产出；
- qualification strength、authoring mode、visibility、source/information independence 分布可产出；
- human/control/cost/latency 自动记录。

P1 不预设 held-out/strong-evidence 比率阈值，也不要求比 Bare Agent 更省时间。

# 37. 分阶段路线 [ALL]

## P0 - Current Tally Correctness & Safety

修复现有 correctness / execution safety：trace tuple、project namespace、LLM role separation、failure worktree、structured repro、lease recovery、Git serialization、spec_defect closure、false-green、deploy inconsistency。

## P1 - Vertical Trust MVP

目标：`standard` Task 单任务闭环可信、可追踪、可完整 Review、可测量。

包括 TaskContract、Acceptance-level Independence Requirement、VerificationContract + Qualification、held-out isolation capability、Context redaction、workspace/cache、Fake/Codex Runtime、Session Canary、CodegenRun、immutable ChangeSet、VerificationRun、visible/held-out validation、Exposure facts、deterministic Hunk Inventory、Patch Review、human/wall/cost measurement。

退出标准采用 §36.3；不要求已经比裸 Agent 更省时间。

## P2 - Engineering Automation & Product Value

目标：把 P1 中由 Owner 承担的控制工作尽可能交给 LLM，并验证 Tally 是否值得日常使用。

包括：Triage、Fast Path、完整 profile 旅程、Product/Planning automation、VerificationPlanner、Task DAG、generalized Workflow Kernel、retry/escalation automation、holdout failure Recovery Ladder、documentation Integration Gate、Saved Views、LifecycleSpan/value-stream、Comparative Usage Trial。

## P3 - Runtime Portability

Capability registry、Pi/OpenCode adapter、conformance hardening、AgentProfile/runtime selection。退出标准：至少两个 production Runtime 保持同一 Task/Verification semantics。

## P4 - Operate & Failure Loop

Release/Deployment、Failure Intake、Canonical Fingerprint、FailureEvent → Defect、regression/suspect change、Incident minimum model、DeliveryState、rollout/stability observation。

## P5 - Adaptive Observe

BehaviorManifest、CodegenRun comparison、context/model/runtime/behavior dimensions、Golden Cases、regression replay foundation。AdaptiveKit 可以提出 Policy Review Proposal，但不能自动降低 hard verification policy。

## P6 - Adaptive Evolve（Conditional）

仅在 readiness 达标后启用 candidate/replay/promotion/rollback 等；是否开发由 P5 数据决定。

# 38. 关键产品指标 [P1 record / P2 evaluate]

## 38.1 North Star

**Verified Engineering Output per Human Review Minute**  
Guardrail：**Idea-to-Verified Latency**。

## 38.2 Human Effort

按 `task_profile` 分维：task count、human minutes/verified task、contract/verification/diff review active/wall、manual correction、spec defect decision、escalation rate。

P1 主要产生 standard 数据；P2 激活 light/strict 真实分布。

## 38.3 Lifecycle Span / Value Stream

至少记录 IDEA_RECEIVED、TRIAGE_COMPLETED、PLAN_COMPLETED、AGENT_STARTED/COMPLETED、VERIFICATION_STARTED/COMPLETED、REVIEW_STARTED/COMPLETED、MERGED、RELEASED、DEPLOYED、STABLE，并计算各阶段 latency。

## 38.4 LLM / Agent Cost

- cost per verified task；
- control_plane_llm_cost；
- execution_plane_llm_cost；
- control_plane_llm_cost_ratio；
- tokens by role/runtime/model；
- escalation rate；
- mean iterations；
- runs per verified task。

`cost_plane` 在调用发生时落成事实字段。

## 38.5 Engineering Trust & Verification Independence

至少记录：

- first-pass final validation：`visible_only / with_holdout`；
- qualification strength distribution：`assertion_strong / compile_error_weak / panic_error_weak / baseline_pass / n_a`；
- authoring mode：`pre_commit / post_hoc`；
- visibility：`visible / held_out`；
- strength × visibility 交叉分布；
- acceptance-level planned/effective independence coverage；
- source vs information independence gaps；
- evidence exposure events by channel/scope；
- `verification_generalization_gap = P(held-out fail | all visible required gates passed and held-out executed)`；
- confirmed `holdout_triage_outcome`：implementation_gap / holdout_defect / acceptance_ambiguity / flake / environment / unconfirmed；
- `holdout_bindings_minted_per_task`（仅作为 Verification Policy / Acceptance Design 复核信号，不单独承载因果解释）；
- independence waiver rate / strict_satisfied vs strict_waived；
- protected evidence violation；
- unattributed hunk rate；
- runtime/worktree contamination incidents。

`verification_generalization_gap` 不是 Agent overfitting rate。Policy 调整必须联合 confirmed triage、regression、human correction、exposure 与成本解释。

## 38.6 Software Quality

post-merge regression、post-deploy correlated failure（P4）、rollback（P4）、spec defect、docs drift/integration exception。

## 38.7 Product Automation

LLM artifact acceptance、auto-derived state coverage、manual contract authorship、direct manual coding、fast-path usage/escalation、Policy Review Proposal acceptance。

# 39. 产品成功标准 [ALL]

Tally 成功首先意味着：

- Idea 可以低摩擦进入 Triage，而不是先被迫填完整工程表单；
- ChangeSet 可以被完整验证、Review 和追溯；
- 当前变化受到足够治理，但历史遗留债务不会无关地阻断小任务；
- 软件实体、交付与运行 Failure 可以重新关联回 Intent/Task/Change。

Tally 成功不是因为：

- Agent 跑得多；
- token 用得多；
- 页面多；
- Runtime 支持多；
- Task 自动生成多。

Tally 成功应表现为：

1. 用户从 Idea 到可执行工程任务的人工劳动明显减少；
2. Coding/Debug/Docs 大部分可由 Agent 完成；
3. Agent 无法通过修改验收定义来自证完成；
4. Review 关注关键风险而不是机械读 diff；
5. Task 状态不依赖人工维护；
6. 失败能关联到 Task/Run/Commit/Deploy；
7. 文档随代码变化自动演进；
8. P2 之后与裸 Agent 相比，Owner 实际投入时间/等待时间/质量形成可解释的正向价值；
9. 多 Runtime 可以替换而不破坏工程确定性；
10. 长期行为改进建立在真实证据，而不是 Prompt 直觉上。

---

# 40. 非功能要求 [P1 baseline]

## 40.1 可靠性

- worker crash 不永久卡 job；
- derived state 可重建；
- Git write 不并发污染；
- CodegenRun 可审计；
- validation evidence immutable/referenceable。

## 40.2 性能

- Context selection 避免全仓库无差别塞入；
- shared dependency cache；
- incremental validation；
- final validation 只在 candidate ready 后执行；
- run event streaming 不阻塞 worker。

## 40.3 可维护性

- Go monorepo 保持核心；
- Runtime-specific code 隔离；
- 不为抽象而拆微服务；
- 新 Domain 逐步替换旧逻辑；
- architecture/conformance tests 防止边界退化。

## 40.4 可审计性

关键动作必须具备：

```text
who/what
project
task
revision
runtime/model
base commit
result
validation
approval
created_at
```

---

# 41. 产品边界与未来扩展 [ALL]

未来可评估：

- Claude Agent SDK / Gemini CLI / Aider / OpenHands Adapter；
- 多模型 routing；
- 更完整 Product/Requirement domain；
- richer architecture model；
- IDE integration；
- GitHub/GitLab PR integration；
- remote worker pools；
- pgvector for large knowledge corpus；
- advanced AdaptiveKit；
- team collaboration / approvals。

但任何扩展必须满足：

> **不会让 Tally 从“减少软件工程人工劳动”退化为“维护平台本身成为新的流程劳动”。**

---

# 42. 核心产品关系 [ALL]

```text
ViberCoding
= Methodology

Tally
= Engineering Control Platform

LLM Roles
= Product / Design / Planning / Verification / Review intelligence

Codex / OpenCode / Pi
= Agent Execution Runtimes

Evidence Engine
= Engineering Fact Authority

AdaptiveKit
= Observe / Behavior Evolution Component
```

最终闭环：

```text
Human Idea
    ↓
LLM Productize
    ↓
LLM Design
    ↓
LLM Plan
    ↓
TaskContract + VerificationContract
    ↓
Agent Runtime
    ↓
Coding / Testing / Debug / Docs
    ↓
Deterministic Validation
    ↓
Risk-oriented Patch Review
    ↓
Merge / Build / Deploy
    ↓
Failure Intake / Runtime Evidence
    ↓
Knowledge / Adaptive Observe
    ↓
Better future Product / Planning / Agent Behavior
```

---

# 43. 最终产品原则摘要 [ALL]

Tally 的所有产品和架构决策最终都应回答以下问题：

1. 是否保留了 Vibe Coding 的自由与速度，并让治理深度与风险成比例？
2. 是否把正式工程结果变得可验证、可追溯、可恢复？
3. 是否让 LLM/Agent 承担了更多软件工程劳动，而不是把流程转嫁给人？
4. required success criteria 是否在实现前预提交、具备最低 Evidence Qualification，并且需要独立性的 Acceptance 是否获得了真实 Source/Information Independence？
5. Agent 是否仍然无法通过修改 protected evidence 或规范来自证完成？
6. Deterministic policy 规定的 risk/profile/stage 是否只允许单调升级，不能被 LLM 降级？
7. 每一个代码 hunk 是否在 Approve 前被完整 accounting，是否不存在隐藏的 Unattributed Change？
8. Owner 是否可以用更少或更有价值的时间完成更可靠的软件工程工作？
9. `idea_to_verified_latency` 与 Control Plane LLM 成本是否仍然适合真实日常开发？
10. Runtime 是否可以被替换，而工程确定性仍然保持？
11. 运行故障是否能回到代码、Task 和开发过程？
12. 行为改进是否基于证据，而不是 Prompt 直觉？

如果一个功能不能改善上述至少一项，就不应因为“平台看起来更完整”而优先开发。

---

# 44. MVP 最简用户旅程 [P1]

用户：

> “worker 崩溃后 job 会一直 running，把这个问题完整修掉。”

在 P1 Vertical Trust MVP 中，已有 Change Intent/Task context 进入：

```text
1. TaskContract + Acceptance Independence Requirement Proposal / Review
2. VerificationContract Proposal
3. Baseline Evidence Qualification + planned independence coverage
4. Explicit/Compact Verification Review as required
5. Freeze Verification revision / execution protocol
6. Create disposable worktree + reuse dependency cache
7. Build redacted ContextPack
8. Run Codex in isolated Implementation session
9. Incremental **visible** test/debug loop
10. Fix immutable candidate ChangeSet
11. Final deterministic validation + held-out execution
12. Functional / Independence / Protected Evidence checks
13. Deterministic Hunk Inventory
14. Acceptance/risk-oriented Patch Review
15. Unattributed Changes must be zero
16. Owner decision → HumanApprovalEvidence / Waiver when policy permits
17. Derived TaskExecutionState update
```

P2 再在这条可信链之前自动加入 Risk-aware Intake、Fast Path、Product/Planning/Verification Planning，让用户逐渐回到最自然的入口：

```text
表达 Idea / Change Request
→ 看少量关键 Proposal
→ 审关键 Evidence/Diff/Risk
→ 决策
```

而不是：

```text
手写 PRD
→ 手写 Task
→ 手写测试计划
→ 手工更新状态
→ 自己 Debug
→ 自己补文档
```

这就是 Tally 要实现的 ViberCoding 工程化。

---

# 45. v1.3 Review Candidate 冻结规则与后续文档边界 [ALL]

v1.3 只完成 v1.2 的冻结后可信性补强，不新增长期领域对象。核心变化：

1. Verification Independence 拆成 Authoring、Source、Information、Execution Integrity 与 Applicability；
2. `independence_required` 提升到 Acceptance Clause，Evidence 只记录自身 provenance/visibility/qualification/facts；
3. held-out capability 与 profile/policy 使用要求分离；
4. Contract Policy 与 Exposure Fact 分离；
5. content exposure 与 selection exposure 都会消耗 information independence；
6. verdict 绑定 immutable ChangeSet + binding revision + frozen protocol + environment identity；
7. valid failed evidence 不可被 evidence shopping 抹除，但 gating applicability 随 Desired Revision 变化；
8. fresh binding 不绕过 authoring provenance，且 recovery 有预算边界；
9. escalation/new Run/immutable ChangeSet、cost_plane、stable invariant ID、阶段激活与 Eval/Metric ownership 统一收口。

## 45.1 Freeze 条件

Owner 审核本版后，如不存在以下 Blocker，即可标记：

> **`v1.3 Final — Frozen Product Baseline`**

仅检查：

- Authority 是否仍有双主人；
- Contract/Fact 是否混写；
- P1/P2/P4 activation 是否冲突；
- Architecture Invariant 是否互相矛盾或无法测试；
- Schema/lifecycle 是否存在直接冲突。

不再以“还能增加什么功能/竞品思想”为冻结条件。

## 45.2 后续进入 ADR / Implementation Spec

明确下沉：

- Holdout Evidence Store 物理存储与语言级注入；
- Context redaction / exposure detection 具体实现；
- Session Isolation 各 Runtime 的实现、canary 细节与 provider/account 隔离策略；
- Source Independence / authority lineage 判定算法；
- selection exposure 的 objective/change lineage key；
- precommit candidate-generation/retry record 格式与符合性 validator；
- candidate identity / executable-change 判定；
- verification environment identity 字段；
- flake repetitions/stopping/aggregation 规则；
- fresh-binding / verification revision budget；
- invalid-evidence / holdout-defect confirmation workflow；
- language-specific coverage/change-overlap provider；
- Hunk identity/invalidation；
- Review active/idle timing；
- Documentation meaningful diff；
- Canonical Failure Fingerprint 算法；
- Runtime transport、cache key、locking、conformance class；
- SoftwareEntity Catalog、Defect grouping、rollout/merge queue 等后期实现。

后续文档链：

```text
PRD v1.3 Final (Frozen, after Owner approval)
  ↓
ADR Pack
  ↓
P0/P1 Implementation Specs
  ↓
P1 Vertical Trust Task Pack
  ↓
真实 Task Evidence
  ↓
P2 Product Value Decision
```

从此以后，qualification strength、generalization gap、independence coverage、human review minutes、lifecycle latency、control/execution cost、false-verified/regression 等真实数据拥有比继续开放式文本评审更高的产品决策权。

# 附录 A：术语

**Idea**  
用户最初的自然语言意图、问题或想法。

**Product Proposal**  
LLM 根据 Idea 和现有产品上下文提出的产品变化候选。

**TaskContract**  
定义 Agent 要做什么、允许改什么、有哪些约束的版本化工程合同。

**VerificationContract**  
定义 Task 如何被独立证明完成的版本化合同。

**Verification Authoring Mode**  
required evidence 的定义来源时机：`pre_commit` 表示在其授权的 Implementation Run 前、且不含 candidate context；`post_hoc` 仅用于受控治理升级，必须 candidate-blind、显式审核并授权新的 Run。

**Protected Evidence**  
Implementation Run 不允许修改的 required verification evidence。

**Evidence Qualification**  
required evidence 在获得 acceptance authority 前必须满足的最低机械资格；对 test evidence 包括 baseline sensitivity、candidate green 和 change relevance 等，具体规则由 VerificationContract/ADR 定义。

**ExecutionPolicy**  
Tally Control Plane 对 retry、budget、escalation、stop condition、network/shell 等执行边界的版本化策略，与可进化的 BehaviorManifest 分离。

**Governance Monotonicity**  
Deterministic policy 给出的最低 risk/profile/stage 只能保持或升级，不能被 LLM/Agent 降低。

**Unattributed Changes**  
存在于 deterministic Hunk Inventory 中但尚未被 Review accounting 的改动；非空时阻断 Approve。

**Agent Claim**  
Agent 对自己工作的描述，不是事实。

**Evidence**  
由 Tally/外部工具实际执行或记录得到的可追溯工程事实。

**CodegenRun**  
一次受控 Agent 实现/调试执行。

**ContextPack**  
为某个 Role/Task 构造的版本化上下文集合。

**AgentRole**  
Product、Planning、Coding、Debug、Docs、Review 等逻辑职责。

**AgentRuntime**  
Codex、OpenCode、Pi 等实际执行后端。

**AgentProfile**  
Role + Runtime + Model + Behavior + Budget + Capability requirements 的组合。

**BehaviorManifest**  
可版本化的 LLM/Agent 行为策略配置。

**Adaptive Observe**  
基于开发证据进行比较和人工行为调优，但不自动 promotion。

**SpecDefectProposal**  
Agent 认为 Requirement/Acceptance/Verification 本身存在问题时生成的正式审查请求。

**Minimal Workflow Kernel**  
Tally 为 Task/Run/Validation/Retry/Approval 等自身领域实现的受限工作流能力。

**ViberCoding**  
工程治理下的 Vibe Coding：保留 AI 驱动开发自由，同时把产品意图、工程执行、验证证据和生命周期状态显式化、可审计化。

---

**Triage**  
将 Idea/Failure/Change Request 在正式化之前进行归并、关联、快速通道或升级判定的低成本入口。

**SoftwareEntity**  
Project 中可独立理解、构建、部署或运行的软件单元，如 service、worker、application、library。

**ChangeSet**  
固定 base/candidate revision 的实际代码变化，是 Verification、Review 与 Integration 的正式单位。

**VerificationRun**  
针对特定 VerificationContract revision 与 ChangeSet 的一次实际验证执行。

**IntegrationCandidate**  
已完成必要 Verification、准备基于最新目标分支进入 Integration Gates 的 ChangeSet。

**Release**  
被正式标识为可交付的软件版本/构建集合；不等同于 Deployment。

**Deployment**  
某个 Release/Artifact 在具体 Environment 中的实际部署事实。

**FailureEvent**  
一次实际观测到的失败事实。

**Defect**  
一个或多个相关 FailureEvent 聚合形成的工程问题。

**Incident**  
需要主动响应、恢复和复盘的高影响运行事件。

**Saved View**  
对底层 Engineering Graph 的查询/展示 Projection，不拥有独立事实状态。

**LifecycleSpan**  
两个生命周期事件之间的可测 wall-clock 时间，用于 Value Stream 与瓶颈分析。



**Verification Independence**  
Acceptance 的 required evidence 相对于当前 Implementation Authority 的可信独立性，包括 authoring/source/information 等维度；不是一个单一布尔标签。

**Evidence Visibility**  
Evidence 相对 Implementation Authority 的可见性策略：`visible` 可用于迭代；`held_out` 必须在受控/可观测信息通道中不可达。

**Source Independence**  
Evidence authoring activity 与当前 implementation decision/authority lineage 的 provenance 独立性，不等于作者 user_id 不同。

**Information Independence**  
Implementation Authority 未获得 held-out evidence 内容、可识别 verdict，也未通过 repeated selection 使用该 verdict 优化 candidate 的状态。

**Selection Exposure**  
held-out verdict 被用于 retry/reject/select/generate descendant candidate 所形成的低带宽反馈污染。默认按 verdict/candidate 时序与 precommit 机械判定。

**Evidence Applicability**  
某条历史 Evidence 是否对当前 Desired/Verification revision 仍参与 Gate。它与 Evidence 历史 validity 分离。

**Independence Waiver**  
Project Verification Policy 允许时，Owner 对运行期 independence coverage 损失做显式风险接受的 HumanApprovalEvidence。Waiver 不改变 independence 事实，也不能弥补 Freeze 阶段 planned coverage 缺失。

# 附录 B：核心设计口号

> **From Idea to Evidence.**

> **LLMs do the engineering work; Tally makes it trustworthy.**

> **Keep the freedom of Vibe Coding. Add the discipline of software engineering.**

> **Generation can be probabilistic; acceptance cannot be ambiguous.**

> **Human is the decision authority, not the process operator.**

---

# 附录 C：成熟软件工程产品思想映射

本附录只记录设计来源，避免未来把 Tally 误解成某个单一工具的克隆。

- **禅道**：吸收 Product/Project/Requirement/Task/Test/Bug 的生命周期对象闭环；不复制大量人工流程字段与组织角色操作。
- **Jira**：吸收 Work hierarchy、semantic links、workflow/gates、saved views；不复制通用 workflow builder、大型 RBAC，也不把人工拖状态视为事实。
- **Linear**：吸收 Triage、低摩擦输入、关系建议、Views projection；不把所有工程语义都压缩成同一种轻量 Issue。
- **GitHub/GitLab**：吸收 Change/Review、required checks、integration compatibility 与 value-stream spans；不复制完整 SCM/CI 平台。
- **SonarQube**：吸收 Clean-as-you-change / differential governance 与 quality gate 思想；不让全库历史债务阻断每个小改动。
- **Backstage**：吸收 SoftwareEntity catalog、Docs-as-Code、engineering profiles；不建设大型开发者门户或 CMDB。
- **TestRail**：吸收 Verification Spec / Run / Result 分离与 traceability；不复制传统人工测试管理全套流程。
- **Sentry**：吸收 Event grouping、Defect、regression、suspect change；不替代日志/APM 基础设施。
- **PagerDuty**：吸收 Incident 与 Defect 分离、恢复后 Postmortem/Learn；不建设大型 on-call/incident command 平台。
- **Argo CD**：吸收 Desired vs Observed 与 drift/reconciliation 思想；不复制 Kubernetes/GitOps 控制器。
- **LaunchDarkly**：吸收 Release progression、guarded rollout、rollback gate；不自建 feature flag 平台。

Tally 的目标不是这些产品的功能集合，而是把成熟软件工程思想重新组合为 LLM/Agent 时代的低摩擦控制平面：

> **传统工程软件让人维护过程；Tally 让 LLM 维护过程，让 Agent 执行劳动，让 Evidence 决定事实。**
