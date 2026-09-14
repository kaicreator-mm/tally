# Tally PRD v1.4.1
## Code-Aligned Product Closure, Human-Centered UI & Multi-Harness Execution

- **Version**: v1.4.1
- **Status**: Final / L2+L3 Aligned / Implementation Authorized
- **Date**: 2026-09-11
- **Product**: Tally
- **Positioning**: Software Engineering Harness for Coding Agents
- **Upgrade Type**: Brownfield Product Closure / Architecture Hardening
- **Primary Principle**: Extend existing v1.4 primitives; do not create parallel subsystems.
- **UX Principle**: Familiar engineering workflow outside, AI-native structured decision support inside.
- **Execution Principle**: One primary harness per task; Tally owns engineering authority.

---

# 1. Executive Summary

Tally v1.4 已经完成了最困难的一部分：围绕 Coding Agent 建立了真实的工程执行内核，而不是仅提供“任务 + Chat + Agent”。

当前源码已经具备或基本具备：

- Closed Loop：Execute → Verify → Repair → Accept / Escalate；
- disposable worktree；
- immutable ChangeSet 与 supersession / lineage；
- expected-base / Git 写入保护；
- held-out verification exposure protection；
- Codex runtime identity / capability；
- Contract Compiler；
- EngineeringPolicy；
- risk / Assurance 相关能力；
- Outcome domain/store；
- v1.4 Intent / Launch / Clarification UI；
- append-only facts / derived state 等工程基础。

但当前最完整的生产路径仍然接近：

> Frozen Contract → Execute → Verify → Repair → Accept / Escalate

而不是完整的：

> Intent → Clarify → Confirm → Freeze → Compile → Policy / Risk → Execute → Verify → Repair → Accept → Integrate → Outcome

同时，当前 UI 虽然已经开始从自然语言输入转向结构化 Draft / Clarification，但尚未形成完整的人类工程操作心智；人工审核仍更接近“审核单次 Case”，而不是“审核可复用的软件工程规则”。

因此 v1.4.1 的目标不是重新设计 Tally，也不是新增一套 Agent Runtime、Rule Engine 或 Project Manager，而是基于现有代码完成以下五项收口：

1. **Product Closure**：把已有 Contract / Policy / Assurance / Outcome 接入真实生产主链；
2. **Human-Centered UI**：使用工程人员熟悉的 Project / Work / Review / Rule 操作模型；
3. **LLM Dynamic Forms**：让 LLM 负责生成结构化问题、选项与推荐，而不是让 Chat 成为主交互；
4. **Rule-First Review**：将审核重点从重复 Case 提升到 EngineeringPolicy Rule；
5. **Execution Harness Abstraction**：保留现有 Closed Loop，由 Codex / DeepSeek Harness 等仅承担具体 Execute primitive。

v1.4.1 必须是 brownfield evolution，而不是第二次重构。

本 Final PRD 已吸收 v1.4.1 L2/L3 的修正结论。特别是 Runtime integration 正式冻结为：

> **`internal/agentruntime.Runtime` 是唯一 Coding Harness 核心协议；Codex 与 DeepSeek Harness 均通过 out-of-process Runtime Adapter 接入。**


---

# 2. Current Code Baseline

本 PRD 以当前 v1.4 源码为演进基线。

## 2.1 已真实存在并应保留的核心

### Closed Loop

现有执行闭环已经具备：

```text
Execute
  ↓
Verify
  ↓
Repair
  ↓
Verify
  ↓
Accept / Escalate
```

v1.4.1 不重新实现 Closed Loop。

### Git / Workspace / ChangeSet

现有代码已经具备：

- `internal/gitwriter`
- `internal/workspace`
- `internal/changeset`

以及相关：

- disposable worktree；
- expected base；
- commit；
- rollback；
- Git locking；
- immutable ChangeSet；
- lineage / supersession。

v1.4.1 在此基础上扩展 Git Control Plane，不新建另一套 Git ownership。

### Contract / Policy / Assurance

现有：

- `internal/contract/compile.go`
- EngineeringPolicy matching
- risk signals
- Assurance selection

这些能力应进入 production composition root，而不是重新定义同类对象。

### Outcome

已有 Outcome domain/store。

v1.4.1 主要增加 production writer 与真实生命周期接线。

### Intent / Launch UI

已有 v1.4 Launch / Propose / Clarification 方向。

v1.4.1 应在此基础上：

- 接入 production AI executor；
- 增加 Dynamic Form schema；
- 完成 Confirm → Freeze；
- 不重写一套 Chat UI。

### Codex Runtime

已有 Codex 执行接线、runtime identity / capability 等基础。

v1.4.1 通过 Adapter 包住现有路径，不重写 Codex worker。

---

# 3. Conflict Analysis & Decisions

新增需求只有在不破坏现有架构的前提下进入 v1.4.1。

## 3.1 UI Navigation vs Existing State Machine

### Potential Conflict

当前系统有：

- Draft
- Frozen
- Compiled
- Executing
- Verifying
- Repairing
- Accepted
- Escalated

这些是机器状态。

如果直接拿它们作为一级导航，会形成“系统内部状态驱动 UI”，不符合 Jira / Linear / GitHub 用户的长期操作习惯。

### Decision

**不修改现有生命周期状态机。**

新增的人类导航：

```text
Projects
Work
Plans
Runs
Reviews
Rules
Repositories
Insights
Settings
```

只是现有领域对象的 projection / view。

因此：

> Domain State ≠ Navigation Structure

两者不冲突。

---

## 3.2 Dynamic Form vs Immutable Contract

### Potential Conflict

LLM 动态产生字段可能导致 Contract schema 不稳定。

### Decision

LLM 不直接创建任意 Contract schema，也不能生成任意 HTML。

新增一个受限的 Presentation Schema：

```text
DynamicForm
├── Field
├── Choice
├── MultiChoice
├── Boolean
├── Number
├── Repository
├── Branch
├── RiskChoice
├── Confirmation
└── OptionalFreeText
```

Dynamic Form 只存在于：

> Intent → Draft → Clarification

阶段。

最终：

```text
Dynamic Form
   ↓
Structured Draft
   ↓
Contract Validator
   ↓
Frozen TaskContract / VerificationContract
```

Contract 仍由现有领域模型定义。

因此动态 UI 不改变 Contract authority。

---

## 3.3 Rule Review vs Existing EngineeringPolicy

### Potential Conflict

新增 Rules 页面容易演变成第二套 Rule Engine，与 `EngineeringPolicy` 重复。

### Decision

**禁止新增平行 Rule Engine。**

UI 中的“Rule”就是 EngineeringPolicy 的人类可审核表现。

新增的是：

- RuleDraft；
- RuleProposal；
- RuleVersion；
- RuleEvidence；
- RuleReplayResult；
- RuleApproval；

而不是新的执行规则语言。

最终 Activated Rule 必须编译/保存为现有 EngineeringPolicy 可消费的结构。

统一模型：

```text
Cases / Outcomes
      ↓
Pattern Detection
      ↓
Rule Proposal
      ↓
Historical Replay
      ↓
Human Review
      ↓
EngineeringPolicy Version
      ↓
Contract Compiler
```

因此 Rule Review 是现有 EngineeringPolicy 的治理层。

---

## 3.4 Execution Harness vs Existing Closed Loop

### Potential Conflict

如果 Codex Runtime Adapter / DeepSeek Harness Runtime Adapter 自己拥有：

- retry；
- verification；
- acceptance；
- repair policy；

就会与现有 Closed Loop 重复。

### Decision

Harness 只替换 Closed Loop 中的 **Execute primitive**。

统一关系：

```text
Tally Closed Loop
    │
    ├── Execute ──> agentruntime.Runtime
    │                  ├── Codex Runtime Adapter
    │                  └── DSH Runtime Adapter
    │
    ├── Verify      ← Tally
    ├── Repair      ← Tally policy
    ├── Accept      ← Tally
    └── Escalate    ← Tally
```

Harness 内部允许进行工具调用和局部 coding loop，但不得拥有最终工程 Acceptance。

---

## 3.5 AI Runtime Modes vs Tally Scope

### Potential Conflict

DIRECT / AUTO / ORCHESTRATED 如果在 Tally 中重新实现，会导致 Tally 与 AI Runtime 职责重叠。

### Decision

DIRECT / AUTO / ORCHESTRATED 是 **AI Runtime 内部 planning policy**。

Tally v1.4.1 不实现这三种 planner。

Tally 只定义和消费统一：

```text
PlanningResult
```

并记录：

```text
PlanningProvenance
```

因此 Tally 只关心：

> 收到什么 ProjectPlan / TaskDAG

而不是：

> AI Runtime 用一个模型还是多个模型生成。

---

## 3.6 Git Control Plane vs Existing gitwriter

### Potential Conflict

重新创建 `internal/git/*` 大型子系统可能破坏当前已验证的 `gitwriter/workspace/changeset`。

### Decision

v1.4.1 采用 incremental extension。

保留：

- gitwriter；
- workspace；
- changeset。

新增语义服务层时优先 wrap / compose，而不是搬迁已有代码。

v1.4.1 强制完成：

- L0 Read；
- L1 Workspace；
- L2 Commit；
- L3 Controlled Integration。

L4 Remote 允许做成 adapter / experimental。

L5 Destructive 不自动化，默认 human approval。

---

## 3.7 Rule Evidence vs OutcomeLog

### Potential Conflict

规则审核如果再新建独立 Case History，会重复 OutcomeLog。

### Decision

Rule Proposal 的主要证据来自：

- OutcomeLog；
- Verification result；
- ChangeSet；
- human decisions；
- escalation history。

因此 Outcome 同时是：

1. engineering result；
2. future rule-learning evidence。

---

# 4. Product Definition

Tally v1.4.1 的正式定义：

> **A Software Engineering Harness that turns planning output and accumulated engineering practice into executable constraints, structured human decisions, controlled coding-agent execution and reusable engineering rules.**

---

# 5. Responsibility Boundaries

## 5.1 AI Runtime

AI Runtime 负责：

- Project Planning；
- Task Planning；
- Task DAG；
- dependency；
- parallelism；
- planning risk；
- planning confidence。

支持：

- DIRECT；
- AUTO；
- ORCHESTRATED。

但这些模式属于 AI Runtime。

Tally 只消费统一 PlanningResult。

---

## 5.2 Tally

Tally 负责：

- Intent intake；
- structured clarification；
- Contract Freeze；
- Contract Compile；
- EngineeringPolicy；
- risk / Assurance；
- Git；
- Harness selection；
- Verification；
- Repair governance；
- Acceptance；
- Rule governance；
- Outcome。

---

## 5.3 Execution Harness

Harness 负责：

- read repository；
- reason；
- edit；
- shell / tools；
- local tests；
- coding subagents；
- candidate ChangeSet generation。

v1.4.1：

- Codex Runtime Adapter；
- DeepSeek Harness Runtime Adapter。

---

# 6. UX Architecture

## 6.1 Human Mental Model

Tally UI 应优先沿用软件工程主流产品已经建立的心智：

```text
Project
  ↓
Work
  ↓
Plan
  ↓
Run
  ↓
Review
  ↓
Complete
```

而不是：

```text
Chat
  ↓
Agent Message
  ↓
More Chat
```

一级导航建议：

```text
Home

Projects
Work
Plans
Runs

Reviews
Rules

Repositories
Insights

Settings
```

---

# 7. Work UI

Task / Work 页面建议：

```text
TASK-142  Add GitHub OAuth

Overview
Plan
Changes
Verification
Reviews
Activity
```

对应现有领域对象 projection。

页面顶部展示：

- repository；
- base revision；
- contract state；
- risk；
- assurance；
- harness；
- execution status。

用户无需理解所有内部状态名称才能完成工作。

---

# 8. LLM Dynamic Forms

## 8.1 Interaction Principle

主交互：

> Structured Forms + Generated Choices

Natural Language：

> Intent Input + Escape Hatch

目标比例：

- 主要工程决策使用结构化字段；
- 自然语言用于补充无法结构化的上下文。

---

## 8.2 Dynamic Form Flow

```text
Natural Language Intent
        ↓
LLM Parse
        ↓
Project / Repo Context
        ↓
Missing Decision Detection
        ↓
Dynamic Form Schema
        ↓
Generated Choices
        ↓
User Selection
        ↓
Structured Draft
        ↓
Contract Freeze
```

---

## 8.3 Generated Choice

LLM 可以生成：

- option；
- recommended option；
- recommendation reason；
- confidence；
- conditional follow-up。

例如：

```text
Compatibility

● Preserve current API       Recommended
○ Breaking change allowed
○ Let Tally inspect callers
```

但 recommendation 必须是 advisory。

最终 decision 来自用户或明确 policy。

---

## 8.4 Form Schema Constraint

LLM 只能输出受限 Form DSL。

禁止：

- arbitrary HTML；
- arbitrary JS；
- untyped custom control；
- directly modifying Contract schema。

---

# 9. Review Architecture

## 9.1 Review Should Move from Case-First to Rule-First

目标演进：

```text
Case Review
   ↓
Repeated Pattern
   ↓
Rule Proposal
   ↓
Historical Replay
   ↓
Rule Review
   ↓
EngineeringPolicy
   ↓
Future Cases Automatically Governed
```

人工工作重点从：

> 审核第 101 个相同案例

升级到：

> 审核以后 1000 个案例应该遵守的规则。

---

# 10. Reviews vs Rules

## Reviews

用于：

- exceptions；
- high-risk decisions；
- unresolved conflict；
- one-off approval；
- cases not covered by policy。

## Rules

用于：

- proposed EngineeringPolicy；
- active policy；
- conflicts；
- superseded rules；
- rule history；
- evidence；
- historical simulation。

Case Review 不消失。

它变成：

> Exception + Evidence + Rule Discovery Source

---

# 11. Rule Proposal

Rule Proposal 至少显示：

```text
Rule Name

Why proposed
Supporting cases
Observed decisions

Condition

Action / Constraints

Scope

Historical replay

Potential behavior changes

Exceptions

[Reject]
[Edit]
[Approve & Activate]
```

---

# 12. Historical Replay / Dry Run

Rule 在 Activate 前必须执行历史 replay。

至少展示：

```text
Matched cases
Same behavior
More strict
Less strict
Needs exception
Potential regression
```

规则默认不得在未完成 replay 的情况下自动全局激活。

对于没有足够历史数据的规则：

- 标记 LOW_EVIDENCE；
- 允许 limited scope / trial activation。

---

# 13. Exception → Rule

当用户人工处理 Case 时提供：

```text
[Apply Once]

[Apply and Propose Rule]
```

第二个动作：

1. 保存本次 human decision；
2. 形成 Outcome / review evidence；
3. 创建 RuleProposal；
4. 不直接激活 EngineeringPolicy。

---

# 14. Production Closure

主链必须演进为：

```text
Intent
  ↓
Dynamic Clarification
  ↓
Confirmed Draft
  ↓
Freeze
  ↓
Compile
  ↓
EngineeringPolicy
  ↓
Risk / Assurance
  ↓
Workspace
  ↓
Harness
  ↓
ChangeSet
  ↓
Verification
  ↓
Repair / Escalation
  ↓
Acceptance
  ↓
Integration
  ↓
Outcome
```

---

# 15. Intent / Confirm / Freeze

现有 v1.4 Launch flow 继续使用。

v1.4.1 增加：

- production proposal executor；
- DynamicForm schema；
- schema validation；
- Confirm → Freeze；
- Contract identity；
- immutable revision。

Invariant：

> Confirmed UI state 不等于 executable Task。

只有 Frozen Contract 才可执行。

---

# 16. Contract Compiler Authority

所有 Production Task：

```text
Frozen Contract
      ↓
Compiler
      ↓
Policy Match
      ↓
Risk
      ↓
Assurance
      ↓
CompiledContract
      ↓
Execution
```

Worker 不允许绕过 Compiler。

---

# 17. EngineeringPolicy Evolution

v1.4.1 不创建第二种 Rule type。

EngineeringPolicy 增加治理元信息：

```text
policy_id
version
status

DRAFT
PROPOSED
ACTIVE
SUPERSEDED
REJECTED
```

以及：

```text
evidence_refs
created_from_cases
approved_by
approved_at
supersedes
scope
```

Runtime 仍只消费 ACTIVE policy。

---

# 18. Git Control Plane

## 18.1 Principle

> Agent produces code changes.  
> Tally owns durable repository state.

---

## 18.2 Permission Levels

### L0 Read

- status
- diff
- log
- show

### L1 Workspace

- create disposable worktree
- cleanup
- recovery

### L2 Commit

- expected-base
- local commit
- rollback
- immutable ChangeSet

### L3 Integration

- controlled branch
- merge
- rebase where policy allows
- conflict workflow

### L4 Remote

- fetch
- push
- future PR/MR

v1.4.1 可先作为 capability-gated adapter。

### L5 Destructive

- force push
- history rewrite
- protected branch deletion

默认 human approval。

---

# 19. Execution Harness Abstraction

v1.4.1 **不新增第二套 `ExecutionBackend` 核心接口**。

当前源码已经存在并经过 v1.4 验证的 Runtime Protocol：

```text
internal/agentruntime.Runtime
internal/agentruntime.Session
```

其核心生命周期为：

```text
Runtime.Descriptor()
Runtime.Start()
Session.Execute()
Session.Cancel()
Session.Events()
Session.Result()
Session.Close()
```

并通过 `CapabilitySet` 进行能力协商。

v1.4.1 的正式原则是：

> **Tally embeds the Harness protocol, not the Harness implementation.**

也就是：

> **Tally 内嵌 Harness 控制协议与 Adapter，不把 Codex 或 DeepSeek Harness 本体嵌入 Go 进程。**

生产调用关系保持：

```text
closedloop.CodegenRunner
        ↓
cmd/worker.v14CodegenRunner
        ↓
internal/agentruntime.Runtime
        ↓
Runtime Adapter
        ↓
external Harness process
```

允许新增的是新的 Runtime Adapter，例如：

```text
runtimes/
├── codex/
└── dsh/
```

禁止新增一个与 `agentruntime.Runtime` 平行的通用 Runtime/Backend 生命周期协议。

# 20. Execution Request

不新增第二套 `ExecutionRequest` Domain Model。

Runtime Adapter 继续消费现有协议：

```go
agentruntime.StartRequest{
    Workspace:    disposableWorktree,
    ResumeOf:     ...,
    ModelProfile: ...,
}

agentruntime.ExecuteRequest{
    Instruction: ...,
}
```

其中：

- `Workspace` 必须继续由 Tally 创建和持有；
- `ModelProfile` 只在 Runtime 声明 `model.select` capability 时传入；
- provider-specific / harness-specific configuration 属于 Adapter / composition configuration；
- 不把 DeepSeek、Codex 等供应商字段加入 TaskContract；
- CompiledContract / EngineeringPolicy 先由 Tally 转换成受控 implementation instruction/context；
- hidden / held-out verification material不得进入 ExecuteRequest。

如果 DeepSeek Harness 最终需要 Node/TypeScript SDK，允许 Adapter 使用一个小型 bridge process：

```text
Tally Go
   ↓
runtimes/dsh
   ↓ subprocess / stdio protocol
DSH bridge process
   ↓
DeepSeek Harness SDK
```

从 Tally 视角仍然必须是 **out-of-process Runtime**。

# 21. Execution Result

继续使用现有 `agentruntime.AgentRunResult` 和规范化 Event stream 作为 Runtime Claim。

Runtime 自述只能包含：

- RunID / SessionID；
- runtime status；
- summary；
- usage；
- normalized events；
- observed runtime identity（若 Adapter 能真实观察）。

候选代码变化仍然由 `v14CodegenRunner` / workflow candidate freezer 从 disposable worktree 冻结成 Tally `ChangeSet`。

因此：

```text
Runtime result
   ≠ ChangeSet authority
   ≠ Verification result
   ≠ Acceptance
```

核心 invariant：

> **Harness says "done" ≠ Tally says "accepted".**

# 22. Codex Runtime Adapter

Codex 继续作为 v1.4.1 Primary Coding Runtime。

现有：

```text
runtimes/codex
```

已经以 **外部 Codex CLI 进程**工作的方式实现 `agentruntime.Runtime/Session`。

v1.4.1 的工作不是重写 Codex integration，而是：

> **Preserve, normalize composition, and prove no regression.**

推荐关系：

```text
Tally Go
   ↓
runtimes/codex Runtime Adapter
   ↓ subprocess
codex exec ...
```

Codex 优先适合：

- repo-wide change；
- architecture-sensitive change；
- complex refactor；
- migration；
- integration-sensitive task；
- Git-sensitive task。

v1.4.1 不把 Codex native cross-provider subagent tree 作为 release dependency。

# 23. DeepSeek Harness Runtime Adapter

DeepSeek Harness 是 v1.4.1 的第二 Runtime Adapter，必须与 Codex 使用同一 Tally 集成模式：

```text
Tally Go
   ↓
internal/agentruntime.Runtime
   ↓
runtimes/dsh Runtime Adapter
   ↓ subprocess / headless process
DeepSeek Harness
```

**禁止**把 DeepSeek Harness 的 Agent Loop 直接重新实现到 Tally 内部。

**禁止**让 DSH 自己成为 Tally 的 Git / Verification / Acceptance authority。

如果 DSH 当前最稳定的集成面是 SDK，也必须通过 out-of-process bridge：

```text
Tally Go
   ↓
DSH Runtime Adapter
   ↓ stdio / JSON
Bridge process
   ↓
DeepSeek Harness SDK
```

首版优先适合：

- multi-provider；
- bounded parallelism；
- low-cost workers；
- tests；
- docs；
- review；
- static analysis；
- cross-model validation；
- independent patches。

由于 DeepSeek Harness 当前属于 developer-preview / fast-moving dependency，正式实现前必须先完成 exact-version Spike：

1. pin exact DSH commit/package version；
2. 确认 headless process / bridge protocol；
3. 确认 cwd/workspace；
4. 确认 provider/model selection；
5. 确认 cancel/kill；
6. 确认 child-process cleanup；
7. 返回 `GO / GO_WITH_LIMITS / NO_GO`。

初始安全边界：

- disposable workspace；
- no direct protected push；
- no owner-worktree mutation；
- candidate output only；
- Tally-owned ChangeSet；
- Tally-owned Verification；
- Tally-owned Acceptance。

# 24. Harness Routing

Harness routing 基于现有 Runtime Descriptor / CapabilitySet，不基于 Runtime 名称在核心控制流中分支。

目标：

```text
CompiledContract
      ↓
ExecutionRequirement
      ↓
CapabilityRequirement
      ↓
agentruntime.Negotiate(...)
      ↓
Composition selects configured Runtime Adapter
```

可能涉及：

- `model.select`
- `session.resume`
- `event.streaming`
- 以及经 ADR 明确批准的少量通用 capability。

以下能力如果只是 DSH 内部能力：

```text
multi_provider
parallel_subagents
provider/model route
```

优先留在 DSH Adapter 的配置 / observed metadata 中，不应未经证明就污染 `agentruntime` 核心 capability vocabulary。

控制流原则：

> Runtime `Name` 只用于 identity / audit；通用行为按 capability 协商。

# 25. One Primary Harness Per Task

一个 Task execution 默认只有一个 Tally primary Runtime：

```text
Task
  ↓
Tally Closed Loop
  ↓
ONE primary agentruntime.Runtime
```

允许 Harness 自己在其内部使用 subagents，但必须满足：

- provenance 可观察；
- Tally 仍只有一个 primary Runtime Session owner；
- cancellation 可以传播并最终收敛；
- Harness 内部 child agents 无 Durable Git authority；
- child model/provider 不改变 Tally Acceptance authority。

不把下列递归调用作为 v1.4.1 release requirement：

```text
Tally
 → DSH
   → Codex
     → Codex subagent
```

未来如需支持，必须有单独 ADR 和 provenance/cancellation/budget contract。

# 26. Verification Hardening

Migration false-negative 是 v1.4.1 需要解决的现有问题。

必须分类：

- contract；
- fixture；
- DB lifecycle；
- environment；
- oracle；
- timeout；
- setup / teardown；
- state leakage。

Harness-local tests 只能作为 Evidence。

Authoritative Verification 属于 Tally。

---

# 27. Outcome Production Closure

生产主链必须写入：

- accepted；
- rejected；
- escalated；
- reverted；
- regression_detected；
- hotfix_required；
- human_corrected；
- integration_failed。

Outcome 同时为未来 Rule Proposal 提供 evidence。

---

# 28. Data / Domain Changes

优先增加最少对象。

## Existing objects reused

- TaskContract
- VerificationContract
- CompiledContract
- ChangeSet
- EngineeringPolicy
- Outcome
- Workspace
- Execution state

## Minimal new objects / projections

### DynamicFormDefinition

Presentation-only。

### PlanningProvenance

记录 AI Runtime plan origin。

### Runtime Identity / CapabilitySet

执行 Runtime Adapter 所需的观察/能力投影；优先复用现有 `RuntimeDescriptor` / `ObservedIdentity`，避免新增重复 identity model。

### RuleProposal

EngineeringPolicy draft governance object。

### RuleReplayResult

Policy historical simulation projection。

### ReviewDecision

Human decision evidence。

不得新增：

- 第二套 Task；
- 第二套 Contract；
- 第二套 Policy Engine；
- 第二套 Closed Loop。

---

# 29. UI Information Architecture

```text
Home

Projects
  └── Project

Work
  └── Task
       ├── Overview
       ├── Plan
       ├── Changes
       ├── Verification
       ├── Reviews
       └── Activity

Plans

Runs

Reviews
  ├── Exceptions
  ├── High Risk
  └── Pending Decisions

Rules
  ├── Proposed
  ├── Active
  ├── Conflicts
  ├── Superseded
  └── History

Repositories

Insights

Settings
```

系统状态仍使用当前 state machine。

---

# 30. UI Design Rules

1. 用户看到 Work，而不是内部 State Machine。
2. Chat 不成为一级入口。
3. LLM 用于理解、生成选项、推荐和补全。
4. 高价值决策使用选择项。
5. 每个 AI recommendation 都允许查看 reason / evidence。
6. 高风险 Rule activation 需要历史 replay。
7. Exception 应能够沉淀 Rule Proposal。
8. 页面不暴露无关 Harness 内部实现。
9. 用户能明确看到“为什么需要我处理”。
10. 默认关注 Exception，而不是要求人工查看所有成功 Case。

---

# 31. Failure Taxonomy

保留并强化：

### PlanningFailure

来自 AI Runtime 输出问题。

### ContractFailure

Freeze / Compile / Policy conflict。

### HarnessFailure

runtime / provider / tool loop / capability。

### ExecutionFailure

candidate 无法完成。

### VerificationFailure

工程验证未通过。

### InfrastructureFailure

Git / FS / sandbox / process。

### IntegrationFailure

base drift / conflict / branch protection。

### PolicyFailure

rule conflict / invalid active policy / replay failure。

---

# 32. Safe Export

增加正式 project export。

必须排除：

- `.git/`
- `.worktrees/`
- `.e2e/`
- credentials
- runtime secrets
- `.codegraph/`
- binaries
- caches
- local execution artifacts

不能依赖 `.gitignore`。

---

# 33. Non-Goals for v1.4.1

为了保证可从现有代码演进，以下不作为本版本强制目标：

- 全量替代 Jira / Linear；
- 任意 LLM-generated UI；
- 新 Rule DSL；
- 自动规则自我修改；
- 全自动 push / PR / merge；
- autonomous destructive Git；
- complex nested harness trees；
- 在 Tally 内实现 AI Runtime planner；
- 在 Tally 内重做 multi-agent runtime；
- 大规模重新设计现有数据库；
- 大规模迁移现有 package structure。

---

# 34. Implementation Phases

## Phase 0 — Baseline Closure

- 固定当前 source baseline；
- 修复 docs drift；
- 建立 code-to-PRD mapping；
- 保留 v1.4 regression suite。

## Phase 1 — Intent / Dynamic Form / Freeze

基于现有 Launch UI：

- production AI executor；
- DynamicFormDefinition；
- generated choices；
- Confirm → Freeze；
- Contract validation。

## Phase 2 — Compiler Production Wiring

- CompiledContract 成为 Worker 输入；
- EngineeringPolicy；
- Risk；
- Assurance；
- no-bypass tests。

## Phase 3 — Human-Centered UI Projection

不改状态机：

- Work；
- Runs；
- Reviews；
- Rules；
- Task detail projections。

## Phase 4 — Rule Governance

基于 EngineeringPolicy：

- RuleProposal；
- Policy version lifecycle；
- Evidence；
- Historical Replay；
- Apply Once / Propose Rule。

## Phase 5 — Harness Adapter

先保持当前 `runtimes/codex` 作为 `agentruntime.Runtime`，只重构 composition root 为 runtime factory/registry；

验证无 regression 后：

- 新增 `runtimes/dsh`，实现同一 `agentruntime.Runtime/Session` 协议；
- 完成 exact-version DSH Spike；
- 使用现有 CapabilitySet/Negotiate，只有经过 ADR 的通用能力才进入核心协议。

## Phase 6 — Git Integration

在现有 gitwriter/workspace/changeset 之上：

- L3 integration；
- protected branch policy；
- optional L4 adapter。

## Phase 7 — Verification / Outcome Closure

- migration false-negative；
- Outcome production writers；
- Rule evidence connection。

## Phase 8 — Hidden Validation / Product Validation

- security；
- policy bypass；
- harness bypass；
- rule replay；
- UI decision flow；
- backend comparison。

---

# 35. Release Gates

## Gate A — No Architecture Duplication

不得存在第二套：

- Contract；
- Policy Engine；
- Closed Loop；
- Git writer；
- Task lifecycle。

## Gate B — Product Closure

Intent 可到 Accepted / Escalated / Outcome。

## Gate C — Compiler Authority

100% supported production executions use CompiledContract。

## Gate D — Human-Centered UX

至少完成：

- Work view；
- Dynamic Clarification；
- Review view；
- Rules view。

## Gate E — Rule-First Review

Rule Proposal 能：

- collect evidence；
- replay history；
- human approve；
- activate EngineeringPolicy version。

## Gate F — Harness Compatibility

现有 Codex path 无 regression；

DeepSeek backend 至少完成 bounded real-repository scenario。

## Gate G — Git Authority

持久 Git 变更不能绕过 Tally policy。

## Gate H — Outcome

Accepted / reverted / human-corrected 均进入 Outcome。

## Gate I — Verification

Migration disagreement 已解释、修复或被明确隔离。

## Gate J — Documentation / Export

Current docs 与源码一致，安全 export 可用。

---

# 36. Acceptance Scenarios

## Scenario 1 — Simple Bugfix

```text
Intent
→ Dynamic Form
→ Freeze
→ Compile
→ Codex
→ Verify
→ Accept
→ Commit
→ Outcome
```

## Scenario 2 — Low-cost Parallel Task

```text
Task
→ Capability Match
→ DeepSeek Harness
→ Candidate ChangeSet
→ Tally Verify
→ Accept
```

## Scenario 3 — High-risk Migration

```text
Task
→ HIGH Assurance
→ Harness
→ Migration Verification
→ Repair / Escalate
→ Human Exception if needed
```

## Scenario 4 — Repeated Human Decision

```text
Exception Case
→ Human Decision
→ Apply and Propose Rule
→ Historical Replay
→ Rule Review
→ EngineeringPolicy ACTIVE
→ Future Cases auto-governed
```

## Scenario 5 — Rule Conflict

```text
New Rule Proposal
→ Replay
→ Conflict with Active Policy
→ Rules / Conflicts
→ Human resolve
→ Version supersession
```

## Scenario 6 — Base Drift

```text
Execution complete
→ Owner HEAD changed
→ integration rejected
→ explicit rebase / rerun decision
```

---

# 37. Metrics

## Engineering

- hands-off acceptance；
- repair rate；
- escalation；
- regression；
- verification false-negative；
- base-drift rejection。

## UX

- clarification turns avoided；
- % decisions via structured form；
- natural-language fallback rate；
- time to freeze contract；
- human decision count per task。

## Rule Governance

- repeat cases before rule；
- rule proposal acceptance；
- cases automatically governed；
- exception rate after rule activation；
- replay conflict rate；
- policy regression rate。

## Harness

- acceptance by backend；
- cost；
- latency；
- repairs；
- human intervention；
- downstream Outcome quality。

---

# 38. Definition of Done

v1.4.1 完成时必须证明：

1. 新设计没有引入第二套 Contract / Policy / Closed Loop；
2. 当前 Codex production path 经 Adapter 后没有核心行为 regression；
3. Intent UI 能通过受限 Dynamic Form 完成结构化 Clarification；
4. Confirm 真正产生 Frozen Contract；
5. Worker 强制消费 CompiledContract；
6. EngineeringPolicy 是 Rules UI 的唯一 runtime rule authority；
7. Case Review 可以产生 RuleProposal；
8. RuleProposal 可以 historical replay 后成为 ACTIVE EngineeringPolicy；
9. Codex Runtime Adapter 与 DeepSeek Harness Runtime Adapter 都实现现有 `agentruntime.Runtime/Session`，且均为 out-of-process；
10. Harness 无法 self-accept；
11. Git durable state 由 Tally 控制；
12. Outcome 已接生产链；
13. 用户主要操作模型符合 Project / Work / Review / Rule；
14. Current documentation 与实现一致；
15. Safe Export 不包含本地敏感状态。

---

# 39. Final Architecture

```text
                       Product Goal
                            │
                            ▼
                     ┌─────────────┐
                     │ AI Runtime  │
                     │             │
                     │ DIRECT      │
                     │ AUTO        │
                     │ ORCHESTRATED│
                     └──────┬──────┘
                            │
                     PlanningResult
                            │
                            ▼
┌──────────────────────────────────────────────────┐
│                     Tally                        │
│                                                  │
│ Intent → Dynamic Form → Frozen Contract          │
│                     ↓                            │
│              Contract Compiler                   │
│                     ↓                            │
│          EngineeringPolicy / Risk                │
│                     ↓                            │
│                Assurance                         │
│                     ↓                            │
│          Git-controlled Workspace                │
│                     ↓                            │
│             Harness Capability Match             │
│                 /             \                  │
│                /               \                 │
│          Codex Runtime Adapter     DSH Runtime Adapter  │
│                \               /                 │
│                 Candidate ChangeSet              │
│                         ↓                        │
│                    Verification                  │
│                         ↓                        │
│                 Repair / Escalation              │
│                         ↓                        │
│                     Acceptance                   │
│                         ↓                        │
│                  Git Integration                 │
│                         ↓                        │
│                      Outcome                     │
│                         ↓                        │
│        Evidence → Rule Proposal → Replay         │
│                         ↓                        │
│                 EngineeringPolicy                │
└──────────────────────────────────────────────────┘
```

---

# 40. Product Statement

v1.4.1 完成后，Tally 不只是：

> 管理 Coding Agent 的任务。

而是：

> **用人类熟悉的软件工程工作方式收集意图和决策，用 LLM 动态表单降低操作成本，用 EngineeringPolicy 将重复人工判断沉淀成规则，并用 Git、Verification、Acceptance 和多种 Coding Harness 将这些规则强制执行。**

其核心闭环变成：

> **Plan → Contract → Execute → Verify → Decide → Learn Rule → Enforce**

这也是 v1.4.1 相对于 v1.4 最重要的产品升级。
