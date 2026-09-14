# Tally Product Direction

**文档状态：** Product Direction / Non-binding  
**日期：** 2026-09-08  
**关联实现版本：** `Tally_PRD_v1.4_Final.md`  
**约束：** 本文件不构成实现授权，不得直接据此新增数据库对象、UI 模块或 Roadmap Task。任何能力必须经过真实产品数据验证后，再由新的 Final PRD 提升。

---

# 1. 长期产品愿景

Tally 的长期方向仍然是：

> **Vibe Coding × Software Engineering**

目标不是恢复传统软件工程的全部流程。

目标是：

> **把传统软件工程中真正能提高正确率、减少返工、控制风险和积累经验的机制，重新编码成 Coding Agent 可以运行其中的 Harness。**

最终用户体验应当是：

```text
Human
负责 Intent / Judgment

↓

Tally
负责 Engineering Harness

↓

Coding Agents
负责高吞吐执行

↓

Tools / Verifiers
提供 Evidence

↓

Tally
负责闭环、风险和长期工程反馈
```

---

# 2. 长期定位

Tally 不应成为：

- AI Jira；
- Vibe Kanban；
- Coding Agent；
- 独立 Verifier；
- 完整 GitLab clone；
- DevSecOps 工具集合；
- 企业流程管理平台。

更合理的长期技术定位是：

> **Software Engineering Harness for Coding Agents**

其中 v1.4 先实现最小闭环：

```text
Contract
→ Execute
→ Verify
→ Repair
→ Accept / Escalate
```

长期再向前后扩展。

---

# 3. 软件工程重新分类

传统软件工程能力可以拆成四类。

## 3.1 Engineering Semantics

必须长期保留：

- Requirement；
- Acceptance；
- Task；
- Defect；
- Dependency；
- Change；
- Verification；
- Release；
- Incident；
- Architecture Decision。

## 3.2 Engineering Control

长期加强：

- Contract；
- Risk；
- Engineering Policy；
- Verification；
- Evidence；
- Quality Gate；
- Traceability；
- Release Gate；
- Security；
- Compatibility。

## 3.3 Human Coordination

长期弱化：

- Assignee；
- Story Point；
- Sprint；
- Workload；
- Daily update；
- manual priority grooming；
- manual status sync。

## 3.4 Management Ceremony

长期尽量删除：

- 人工状态维护；
- 人工工时录入；
- 固定层层审批；
- 人工测试结果同步；
- 人工 PR/Commit 关联；
- 人工 Release 进度同步；
- 日报。

长期设计原则：

> **Preserve engineering semantics. Automate engineering ceremony. Strengthen engineering verification.**

---

# 4. Harness 的三层作用

长期 Tally 需要覆盖：

## 4.1 Prevention

减少 Agent 走错路：

- Contract；
- project-specific context；
- architecture constraints；
- dependency awareness；
- known failure patterns；
- Engineering Guidance；
- examples；
- preferred mechanisms。

## 4.2 Detection

发现错误：

- tests；
- lint；
- build；
- static analysis；
- independent LLM review；
- protected verifier；
- held-out；
- compatibility；
- security；
- performance。

## 4.3 Decision

处理无法自动判定的问题：

- architecture trade-off；
- business decision；
- high-risk approval；
- waiver；
- release decision；
- exception handling。

长期完整 Harness：

```text
Prevent
→ Execute
→ Detect
→ Repair
→ Escalate
→ Learn
```

---

# 5. Guidance 的长期位置

v1.4 不建设 Guidance Plane。

但长期不能假设：

> Agent 越强，所有 Guidance 都会消失。

必须区分：

## 5.1 Generic Guidance

例如：

- 写测试；
- 注意 SQL injection；
- 做 error handling。

价值会随模型能力提高而下降。

不应成为 Tally 核心资产。

## 5.2 Project-specific Engineering Context

例如：

- 为什么项目采用 CQRS；
- 为什么禁止某个依赖；
- 某 API 为什么必须 backward compatible；
- 当前 migration strategy；
- 特定模块的历史坑；
- 项目自己的 reliability standard。

这类信息不会因为模型更强而自动存在于模型参数里。

长期 Tally 可能负责：

- 选择；
- 检索；
- 版本 pin；
- relevance ranking；
- 注入；
- effectiveness observation。

但不一定负责原始 authoring。

---

# 6. Requirement 的长期位置

Tally 不应第一天要求用户迁移 GitHub Issues / Jira。

长期采用：

> **External-source-friendly Requirement Semantics**

可能来源：

```text
GitHub Issue
GitLab Issue
Jira
Linear
自然语言
外部 PRD
```

Tally 自己真正需要 authoritative 的是：

```text
Intent
Acceptance
Scope
Constraints
Revision
SourceRef
```

也就是：

> **可执行 Acceptance Contract。**

随着使用增加，Tally 可能自然形成更多 Requirement authority，但不把 SoR migration 作为采用前提。

---

# 7. Software Engineering UI 长期方向

长期 UI 更接近：

> AI-native Engineering Hub

而不是 Agent Terminal。

但必须避免完整 Jira 化。

可能长期存在：

```text
Attention
Work
Changes
Engineering
Insights
```

而不是 10+ 一级菜单。

## 7.1 Attention

人的主要入口。

只展示真正需要 Judgment 的事项。

## 7.2 Work

提供：

- List；
- Board；
- Dependency；
- Requirement projection；
- Defect projection。

Kanban 是 View，不是状态 authority。

## 7.3 Changes

整合：

- ChangeSet；
- Verification；
- Evidence；
- Review。

不拆成三个一级模块。

## 7.4 Engineering

可能包括：

- Policies；
- Architecture docs；
- ADR；
- engineering context；
- risk / assurance configuration。

## 7.5 Insights

在有足够真实数据以后才建设。

---

# 8. Human UI 的长期原则

## 8.1 Structured record, natural language input

录入：

> Natural language first。

记录：

> Structured first。

修正：

> Diff + language。

判断：

> Structured options。

## 8.2 Attention ≠ Notification

Attention 只允许来自：

- ambiguity；
- policy conflict；
- unmatched risk；
- failed repair；
- human authority；
- waiver；
- release blocker；
- incident decision。

不能退化成消息中心。

## 8.3 Review 的目标

长期 Human Review 不应默认等于：

> 阅读所有 Diff。

Tally 应先展示：

- change intent；
- critical hunks；
- risk；
- policy；
- evidence；
- uncertainty；
- residual risk。

人仍可查看完整 Diff。

目标是：

> 优化 Human Judgment，不是消除 Human Visibility。

---

# 9. Engineering Policy 长期方向

Policy 是长期最重要的新 Engineering abstraction 之一。

可能包含：

```text
Policy
├── rationale
├── enforcement
├── trigger
├── requires
├── verification
├── gate
├── scope
└── revision
```

## 9.1 Guidance

帮助 Agent。

不直接阻断。

## 9.2 Gate

影响 Acceptance。

必须：

- 可解释；
- 有 Evidence 支持；
- 有 revision；
- 可观察 false positive / waiver；
- 不能随意由 LLM 静默修改。

## 9.3 Policy 不能变成 YAML bureaucracy

如果长期出现：

- waiver 率过高；
- false positive 高；
- Policy 编写维护成本高；
- 用户持续绕过；

应缩减 Policy，而不是要求用户适应系统。

---

# 10. Risk-adjusted Assurance 长期方向

长期不追求：

> 所有任务最高强度验证。

而使用：

```text
risk
+
verification availability
+
cost
+
historical effectiveness
```

动态决定 Assurance。

未来更合理的模型可能从 v1.4 的单轴 L0–L4，扩展成：

```text
AssuranceStrength
├── self
├── deterministic
├── independent
├── protected
└── held_out

HumanAuthority
├── not_required
├── recommended
└── required
```

原因：

> Human judgment 与 deterministic assurance 不是同一强度维度。

---

# 11. Evidence Provider 生态

Tally 不应该自己重新实现：

- Sonar；
- SAST；
- Playwright；
- Pact；
- CI；
- test framework；
- observability；
- deployment platform。

长期统一抽象：

```text
EvidenceProvider
```

候选来源：

- `go test`；
- pytest；
- Jest；
- GitHub Actions；
- GitLab CI；
- SonarQube；
- Semgrep；
- Snyk；
- Playwright；
- Pact；
- benchmark；
- deployment health；
- observability。

Tally 的价值是：

- candidate binding；
- provenance；
- applicability；
- assurance；
- evidence composition；

不是复制工具本身。

---

# 12. Agent Runtime 生态

长期支持多个 Coding Agent：

- Codex；
- Claude Code；
- OpenCode；
- Pi；
- future agents。

Tally 不把：

> 支持最多 Agent

作为核心卖点。

Runtime neutrality 只是必要基础。

长期真正有价值的是：

> 同一 Contract / Policy / Evidence semantics 可以跨不同 Agent 运行。

---

# 13. Closed Loop 的长期扩展

v1.4：

```text
Intent
→ Contract
→ Execute
→ Verify
→ Repair
→ Accept / Escalate
```

未来可以扩展成：

```text
Requirement
→ Plan
→ Contract
→ Execute
→ Verify
→ Repair
→ Review
→ Release
→ Operate
→ Outcome
→ Learn
```

这是一条长期语义链。

不是要求一次性开发全部 UI。

---

# 14. Release 长期方向

未来 Release 可以成为正式对象。

可能绑定：

- Requirements；
- accepted ChangeSets；
- migrations；
- security；
- compatibility；
- known defects；
- release notes；
- deployment evidence。

ReleaseGate 应由 Evidence 派生。

不允许简单手工拖成：

```text
READY
```

---

# 15. Defect / Incident 长期方向

长期闭环：

```text
Production Failure
↓
Defect / Incident
↓
Root Cause
↓
WorkItem
↓
Contract
↓
ChangeSet
↓
Regression Verification
↓
Release
↓
Outcome
```

Incident 是高严重度协调对象。

不与普通 FailureEvent 混为一体。

---

# 16. Outcome 长期方向

Outcome 可能包括：

- merged；
- released；
- reverted；
- hotfixed；
- regression；
- incident；
- human correction；
- stable window passed。

Outcome 不允许重写历史 Evidence。

Outcome 主要用于回答：

> 过去哪些 Harness 真的有用？

---

# 17. Project Engineering Model

只有当真实数据达到足够数量时，才考虑正式产品化。

它不是首版 Domain Object。

长期可能表达：

- 哪类 Change 最危险；
- 哪些模块 regression 多；
- 哪类 Verification 真正有效；
- 哪些 Policy noise 高；
- 哪些 Agent 在某类任务表现好；
- 哪些 repair pattern 反复出现；
- Human Attention 主要花在哪里。

第一阶段用 SQL / instrumentation 即可。

不要提前造“AI 自我学习模型”。

---

# 18. Adaptive Harness 长期方向

Adaptive 不是：

> 自动改 Prompt。

长期可能是：

```text
Outcome
+
Evidence
+
Failure
+
Repair
+
Policy effectiveness
        ↓
Proposal
        ↓
Replay
        ↓
Compare
        ↓
Human / policy approval
        ↓
Promotion
```

可演进内容：

- Policy candidate；
- Assurance default；
- risk rule；
- Agent choice；
- Context strategy；
- repair strategy。

不能直接自动降低 hard gate。

---

# 19. 软件工程最佳实践的长期建模

不要建立一个“最佳实践 Wiki”。

长期应分：

```text
Principle
↓
Policy
↓
Mechanism
↓
Provider
```

例如：

```text
Principle
Backward compatibility matters

↓

Policy
Public API changes require compatibility classification

↓

Mechanism
Consumer contract verification

↓

Provider
Pact
```

Principle 可以只是 rationale。

不一定独立成 Domain Object。

---

# 20. 传统开发模式的 Agent 化

可能长期支持：

- Regression-first；
- TDD；
- BDD；
- Migration-safe；
- Compatibility-first；
- Security-sensitive；
- Performance-sensitive；
- UI visual validation。

这些不应该首先做成：

> workflow builder。

应该作为：

> Harness behavior / policy bundle。

---

# 21. 不同任务使用不同 Harness

长期目标：

## Documentation

轻 Harness。

## Bug Fix

Regression Harness。

## Migration

Migration Harness。

## Authentication

Security Harness。

## Public API

Compatibility Harness。

## UI

UI interaction / visual Harness。

## Performance

Benchmark Harness。

原则：

> 不让所有 Task 走同样沉重的流程。

---

# 22. Metrics 长期方向

传统弱化：

- Story Point；
- Sprint Velocity；
- manual effort；
- completed percentage。

长期重点：

## Throughput

- Time to Accepted；
- Hands-off Completion；
- repair attempts。

## Human

- intervention；
- judgment；
- clarification；
- review burden。

## Quality

- rework；
- escaped defect；
- regression；
- rollback。

## Assurance

- level achieved；
- policy coverage；
- unmatched risk；
- waiver；
- verifier effectiveness。

## Economics

- model cost；
- verification cost；
- accepted change cost。

---

# 23. 长期真正可能形成的资产

不要依赖：

> 竞争对手不会做。

长期可积累资产可能来自：

## 23.1 Project-specific failure patterns

真实发生过的：

- regression；
- rollback；
- repair；
- verification miss。

## 23.2 Evidence / Change / Outcome history

项目真实工程历史。

## 23.3 Harness effectiveness history

知道：

> 哪些工程规则真的有价值。

## 23.4 Cross-agent actual performance

如果样本量足够：

- task type × agent；
- task type × model；
- cost；
- success；
- rework；
- human intervention。

## 23.5 Behavioral trust

如果 Tally 长期可靠，用户逐渐允许更多 Task：

> hands-off。

这是 workflow stickiness，而不是封闭数据锁定。

---

# 24. 长期竞争策略

## 不与 Jira/禅道正面竞争

不追求：

- 全量项目管理；
- HR；
- 工时；
- Scrum 管理；
- 企业审批。

## 不与 Kiro 正面竞争

不追求：

- 最强 Spec Editor；
- IDE-first coding experience。

## 不与 Factory 正面竞争

不从第一天做：

- Enterprise full SDLC；
- 巨型 orchestration platform。

## 不与 CodeRabbit/Qodo/Sonar 正面竞争

它们优先作为 Provider。

Tally 聚焦：

> Closed Loop + Contract + Evidence + Repair + Risk-adjusted Assurance。

---

# 25. 产品扩展顺序

长期优先顺序必须由数据决定。

原则上：

```text
Closed Loop
↓
Assurance
↓
Repair
↓
Attention
↓
Provider ecosystem
↓
Broader Harness
↓
Release / Operate
↓
Instrumentation
↓
Adaptive
```

不允许反过来。

---

# 26. Persona 扩展

## 当前

个人 Vibe Coder。

## 下一步

AI-heavy 小团队。

## 更晚

已有成熟 GitHub/GitLab/Jira/CI 的组织。

进入企业阶段前必须先证明：

- 单人高频使用；
- Human intervention 显著下降；
- closed loop 明显优于 bare agent；
- Policy 不产生官僚成本。

---

# 27. Tally 长期不应该拥有的东西

明确长期避免：

- HR；
- Attendance；
- Payroll；
- 工时核算；
- 完整 Scrum 平台；
- SAFe；
- 组织 OKR；
- 企业 Wiki clone；
- 完整 CI；
- 完整 Static Analysis；
- Coding Agent 自身；
- 通用 LLM Provider Runtime。

---

# 28. Product Direction Invariants

1. 最终目标是减少 Human orchestration，不是消灭 Human judgment。
2. Software Engineering semantics 可以保留，ceremony 尽量自动化。
3. Agent 不拥有 Acceptance Authority。
4. Evidence 必须绑定具体 candidate。
5. High-risk Harness 只能显式降级，不能静默降级。
6. Guidance 不能变成第二套 Wiki。
7. Policy 不能变成新的 Jira 必填字段。
8. 更强模型不会让 project-specific context 消失。
9. 更强 Agent 会提高 verification / governance 的相对重要性。
10. Tally 不依赖任何单一 Agent 的能力作为长期护城河。
11. 用户频率优先于理论架构完整性。
12. 没有真实 Outcome 数据，不谈 self-evolution。
13. 新产品模块只有在现有闭环真实遇到瓶颈时才能升级为 implementation scope。

---

# 29. 长期产品定义

> **Tally 是 Vibe Coding 时代的软件工程 Harness。**
>
> 它不试图重新创造整个 Jira、GitLab、QA、Sonar 或 Coding Agent，而是把这些体系背后真正有价值的软件工程约束、验证、风险控制和经验沉淀，压缩成一个 Agent 可以连续运行的闭环。
>
> **Human 负责 Intent 和 Judgment；Agent 负责 Execution；Tally 负责 Contract、闭环、Assurance 和长期工程反馈。**
