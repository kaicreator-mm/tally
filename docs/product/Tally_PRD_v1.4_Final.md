# Tally PRD v1.4 Final

**文档状态：** Final / Implementation Authorized  
**版本：** v1.4 Final  
**日期：** 2026-09-08  
**产品名称：** Tally  
**产品定位：** Contract-driven Closed Engineering Loop for Coding Agents  
**中文定义：** Tally 把 Coding Agent 的任务变成一个闭合的软件工程执行环：先冻结完成条件，再执行、验证和自动修复，只有真正需要人的判断时才把问题交回来。  
**目标用户：** 以 Codex、Claude Code、OpenCode 等 Coding Agent 为主要开发执行者的个人开发者与 AI-heavy 小团队  
**实现方式：** 基于现有 Go + PostgreSQL + Git monorepo 的 brownfield 演进，不重写 v1.3 Trust Core

---

# 0. 文档作用

本文件是 Tally v1.4 的唯一实现授权文档。

它只描述：

- v1.4 当前阶段必须实现的产品闭环；
- 允许新增或修改的核心 Domain Model；
- v1.3 Trust Core 的复用方式；
- v1.4 必须完成的真实价值验证；
- 当前阶段明确禁止提前建设的能力。

长期方向、完整 Software Engineering Harness、Release、Incident、Project Engineering Model、Adaptive、企业集成等内容不属于本文件的实现范围，统一放入：

`Tally_Product_Direction.md`

除非未来新的 Final PRD 显式提升，否则 Direction 文档中的内容：

- 不构成实现授权；
- 不构成数据库 schema 授权；
- 不构成 UI 授权；
- 不得因为“架构已经想清楚”而提前开发。

---

# 1. 版本关系

## 1.1 v1.3 不是废弃版本

v1.3 已经形成以下可信执行基础：

- immutable `TaskContract`；
- immutable `ChangeSet`；
- `VerificationContract`；
- `VerificationRun`；
- `Evidence`；
- visible / held-out evidence；
- exposure / information independence；
- deterministic hunk inventory；
- evidence-driven derived state；
- Agent Runtime abstraction；
- workspace isolation；
- measurement 基础；
- HumanApprovalEvidence；
- append-only / immutable fact discipline。

v1.4 不推翻这些能力。

v1.4 的任务是：

> 把 v1.3 的 Trust Core 从“验证一个结果”的能力，升级成“自动跑完整个 Coding Agent Task，直到 accepted 或必须找人”为止的闭合执行环。

---

# 2. v1.4 核心问题

Bare Coding Agent 的日常循环通常是：

```text
用户给任务
↓
Agent 执行
↓
Agent 说 done
↓
用户检查
↓
发现测试/行为/约束失败
↓
用户把错误复制回 Agent
↓
Agent 修
↓
用户重新检查
↓
重复
```

主要问题不是 Agent 完全不会写代码。

主要问题是：

- 人仍然充当执行循环的中间人；
- 完成条件通常没有在执行前冻结；
- Agent 的 `done` 只是自我宣称；
- 失败信息需要人转译后重新喂给 Agent；
- 修复会产生新的候选结果，但缺少严格的 candidate lineage；
- 检查强度经常与任务风险不匹配；
- 用户为了确保可信，不得不持续盯住执行；
- 长期结果缺少可比较的 Outcome 数据。

v1.4 的目标是去掉这段人工中介。

---

# 3. 产品定义

## 3.1 一句话定义

> **Tally turns a coding-agent task into a closed engineering loop: contract, execute, verify, repair, and escalate only when human judgment is actually needed.**

中文：

> **Tally 把 Coding Agent 的任务变成一个闭合的软件工程执行环：Contract → Execute → Verify → Repair → Accept / Escalate。**

## 3.2 产品不是

Tally v1.4 不是：

- AI Jira；
- 禅道替代；
- Requirement authoring 平台；
- Vibe Kanban；
- Multi-Agent Terminal；
- Chat Hub；
- Prompt 管理工具；
- 完整 Software Engineering System of Record；
- 完整 DevSecOps 平台；
- Release Management 平台；
- Incident Management 平台；
- Adaptive / self-evolving engineering platform。

## 3.3 产品是

Tally v1.4 是：

- Coding Agent Task 的默认执行入口；
- Acceptance Contract 的冻结点；
- Agent execution + verification + repair 的 orchestrator；
- immutable ChangeSet 的候选管理器；
- Evidence 和 assurance basis 的记录者；
- Human escalation 的最小判断入口；
- Outcome 数据的采集点。

---

# 4. 核心设计原则

## 4.1 Agent done ≠ Delivery Fact

Agent 的：

```text
done
finished
tests pass
implemented
```

全部只属于 Claim。

只有 Evidence Engine / Derived State 才能产生正式判定。

## 4.2 Contract before execution

Agent 开始执行前，必须存在被冻结的完成条件。

至少包括：

- intent；
- acceptance clauses；
- scope；
- non-scope；
- constraints；
- matched policies；
- required evidence；
- assurance requirement。

## 4.3 Probabilistic generation, deterministic acceptance

Agent 可以：

- 探索；
- 猜测；
- 失败；
- 修复；
- 多次生成候选。

Acceptance 不允许模糊。

Acceptance 必须基于：

- frozen Contract；
- immutable ChangeSet；
- explicit policy requirement；
- traceable Evidence；
- derived verdict。

## 4.4 State is derived, not manually declared

用户或 Agent 不得直接把 Task 拖成：

```text
VERIFIED
ACCEPTED
```

状态来自事实。

## 4.5 Human attention is scarce

v1.4 不追求：

- 显示更多 Agent Event；
- 增加更多配置页；
- 让用户审批更多步骤。

系统应自动完成：

```text
Execute
→ Verify
→ Repair
→ Verify
```

只有无法安全自动推进时才找人。

## 4.6 Never ask humans for derivable data

系统能够从：

- Git；
- Runtime；
- Test；
- Build；
- Evidence Provider；
- ChangeSet；
- Outcome；

获得的信息，不要求人重复填写。

## 4.7 Low friction is a hard requirement

低风险 Task：

- 人工必填字段 = 0；
- natural language intent 可以直接开始；
- P50 `intent → first agent run` 目标 ≤ 90 秒；
- Tally 相对 bare agent 的启动额外开销 P50 ≤ 60 秒。

## 4.8 Verification strength is risk-adjusted

不是所有 Task 都使用 held-out。

Verification 采用分层 assurance。

---

# 5. v1.4 最小产品闭环

```text
1. Intent
   ↓
2. Contract
   ↓
3. Execute
   ↓
4. Verify
   ↓
5. Repair
   ↓
6. Accept / Escalate
```

这是 v1.4 唯一必须完整跑通的主链路。

---

# 6. Step 1 — Intent

## 6.1 输入方式

默认入口：

> 一句自然语言。

例如：

> “修复 logout 后 session 仍然有效的问题。”

可以附带外部来源：

```text
source_ref = GitHub Issue #142
source_ref = Jira ABC-123
source_ref = GitLab Issue #88
```

v1.4 不要求把外部 Issue 搬进 Tally。

## 6.2 Intent 自动结构化

Tally 可以调用 LLM 将 Intent 生成：

- intent summary；
- initial acceptance；
- scope；
- non-scope；
- constraints；
- task type；
- candidate risk signals。

用户只确认结构化 Diff。

低风险任务默认不要求手工逐字段填写。

---

# 7. Step 2 — Contract

## 7.1 Contract 是什么

Contract 合并 v1.3 中：

- TaskContract；
- VerificationContract；

形成一次执行统一冻结的 Contract。

## 7.2 Contract 字段

```text
Contract
├── work_item_revision_pin
├── frozen_intent
├── frozen_acceptance[]
├── scope
├── non_scope
├── constraints
├── matched_policies[]
├── policy_match_basis[]
├── required_evidence[]
├── assurance_requirement
├── completion_semantics
├── amendments[]
└── frozen_at
```

## 7.3 Contract 冻结

开始 Agent execution 后：

- Agent 不能修改 frozen acceptance；
- Agent 不能降低 required evidence；
- Agent 不能静默改变 assurance level；
- Agent 不能修改 Policy 后让自己通过。

## 7.4 Contract Amendment

如果执行后发现：

- 新风险；
- 未预期 affected path；
- policy late activation；
- requirement defect；

只能新增 append-only amendment。

不能覆盖原 Contract 历史。

---

# 8. Policy

## 8.1 v1.4 唯一新增的主要工程规则对象

```text
Policy
├── id
├── revision
├── enforcement
├── rationale
├── trigger
├── requires[]
├── gate_condition
└── bundle_tags[]
```

## 8.2 enforcement

预留：

```text
gate
guidance
```

v1.4 只正式实现：

```text
gate
```

但 schema 保留 `guidance`，避免未来迁移破坏。

## 8.3 requires[] 的双重用途

同一份 `requires[]` 同时用于：

### Assurance

编译 Required Evidence。

### Prevention

在 Agent 执行前作为 Context 告知：

> “最终会按这些条件验收。”

例如 migration：

```text
requires:
- fresh_database
- n_minus_1_upgrade
- data_integrity
```

Agent 在开始前就知道这些要求。

v1.4 不建设独立 Guidance Authoring Plane。

---

# 9. Project Engineering Context

v1.4 不拥有通用 Guidance。

但可以读取并透传已有项目上下文，例如：

- `CLAUDE.md`；
- `AGENTS.md`；
- architecture docs；
- relevant ADR；
- repository conventions；
- existing tests；
- project-specific instructions。

这些内容进入 ContextPack。

原则：

> **Tally consumes project engineering context before it tries to author project engineering context.**

---

# 10. Policy Match Coverage

Policy 匹配不能只返回：

```text
matched = [...]
```

必须同时派生：

```text
PolicyMatchCoverage
├── matched_policies
├── match_basis
├── detected_risk_signals
├── unmatched_risk_signals
└── confidence
```

如果存在：

```text
unmatched_risk_signals != []
```

不能静默使用最低 Assurance。

必须：

- 自动升级；
- 或产生 Attention；
- 或形成显式 limited verdict。

---

# 11. Step 3 — Execute

## 11.1 AgentRuntime

继续沿用 v1.3 Agent Runtime abstraction。

负责：

- workspace；
- session；
- command；
- tool；
- patch；
- events；
- isolation。

## 11.2 Workspace isolation

真实 Agent 执行必须运行在：

- disposable worktree；
- isolated workspace；
- 明确 session boundary。

Owner primary worktree 不作为 Agent sandbox。

## 11.3 Actual runtime facts

每次执行必须记录实际：

- Runtime；
- Runtime version；
- CLI version；
- Model / profile；
- Session ID；
- capabilities；
- start/end；
- exit reason。

Desired model 和 actual model 不得混淆。

---

# 12. ChangeSet

## 12.1 ChangeSet 是真正候选物

Run 不是交付物。

真正被验证的是 immutable ChangeSet。

```text
Run #1
→ ChangeSet #1
```

Repair 后：

```text
Run #2
→ ChangeSet #2
→ supersedes #1
```

## 12.2 ChangeSet 必须绑定

- base revision；
- result identity；
- Contract revision；
- actual Runtime；
- actual Model；
- hunk inventory；
- repair attempt index；
- supersession lineage。

## 12.3 Deterministic hunk accounting

AI Summary 不能隐藏任何代码修改。

每个 ChangeSet 必须有完整确定性的 diff / hunk accounting。

---

# 13. Step 4 — Verify

## 13.1 Verification 不等于单纯跑测试

v1.4 MVP 首先关注：

- functional；
- regression；
- migration correctness；
- required policy evidence；
- protected verifier mutation；
- independence strength。

## 13.2 Assurance Levels

v1.4 采用：

```text
L0 — self-check
L1 — deterministic tools
L2 — independent LLM judge + deterministic mutation checks
L3 — protected / held-out verifier
L4 — human authority required
```

说明：

- L4 不是数学意义上“比 L3 更强”；
- v1.4 先用单轴 level 简化实现；
- 未来可以将 `assurance strength` 与 `human authority` 拆分。

## 13.3 Level selection

Assurance level 由：

- task type；
- policy；
- risk signals；
- unmatched signals；
- historical requirement；

派生。

不得只按成本选择最低档。

## 13.4 unmatched signal rule

如果：

```text
unmatched_risk_signals != []
```

禁止静默使用 L0/L1。

## 13.5 satisfied 与 waived 永不合并

例如：

```text
strict_satisfied
strict_waived
```

必须保持不同语义。

---

# 14. VerifiedBasis

VERIFIED 不再只是 bool。

必须至少包含：

```text
VerifiedBasis
├── verdict
├── assurance_level_required
├── assurance_level_achieved
├── evidence_coverage
├── policy_coverage
├── matched_policies
├── unmatched_risk_signals
├── satisfied_requirements
├── waived_requirements
└── residual_risk
```

示例：

```text
VERIFIED
assurance = L2
coverage = 0.93
unmatched_risk_signals = []
waived = []
```

或者：

```text
VERIFIED_LIMITED
assurance = L1
coverage = 0.74
unmatched_risk_signals = [auth_symbol_change]
```

后者不得显示成普通 VERIFIED。

---

# 15. Evidence

继续沿用 v1.3 Evidence 模型。

Evidence 至少保留：

- provider identity；
- provider config revision；
- candidate binding；
- result；
- applicability；
- visibility；
- provenance；
- exposure lineage；
- assurance level achieved。

Agent Claim 不是 Evidence。

---

# 16. Assurance Bake-off

v1.4 开发初期必须完成一次真实 bake-off。

目标：

> 确定 L1 / L2 / L3 在真实任务中的缺陷捕获增量和成本。

至少比较：

### L1

- deterministic test；
- build；
- lint；
- protected-file mutation detection。

### L2

L1 +

- independent LLM judge；
- trajectory；
- diff；
- Contract。

### L3

L2 +

- protected / held-out verifier；
- isolated execution。

关键问题：

> L2 能捕获 L3 所捕获问题的多少？

结果用于设置默认 Assurance。

不是用于决定 Tally 是否“只做 held-out”。

---

# 17. Step 5 — Repair

## 17.1 Repair 是 v1.4 的核心产品能力

如果 Verify 失败：

```text
FAIL
↓
classify
↓
repairable
↓
failure evidence → Agent
↓
new ChangeSet
↓
Verify again
```

人不应默认充当错误搬运工。

## 17.2 MVP 自动 Repair 上限

默认：

```text
max_auto_repair_attempts = 2
```

超过后 Escalate。

## 17.3 Repair 硬约束

Repair：

- 不允许修改 frozen acceptance；
- 不允许降低 policy；
- 不允许降低 required assurance；
- 不允许静默修改 protected verifier；
- 每次必须产生新 immutable ChangeSet；
- 每次必须有 supersession lineage。

## 17.4 Failure classification

失败至少分：

```text
repairable
contract_defect
policy_conflict
environment_failure
provider_failure
verification_inconclusive
requires_human_judgment
```

只有 `repairable` 默认进入自动 Repair。

---

# 18. Step 6 — Accept / Escalate

## 18.1 自动接受候选

当：

- required evidence complete；
- assurance achieved；
- unmatched risk = none；
- blocking policy = satisfied；
- no required human authority；

可以进入：

```text
ACCEPTABLE
```

是否真正 Merge 可以继续由外部 Git provider 或用户决定。

## 18.2 Escalate

只有以下情况默认找人：

- Contract ambiguity；
- policy conflict；
- unmatched risk；
- repeated repair failure；
- assurance 无法达到；
- human authority required；
- waiver request；
- environment / provider 反复失败。

## 18.3 Escalation UI

每次 Escalation 必须告诉用户：

```text
发生了什么
为什么系统不能继续
有哪些可选方案
推荐哪一个
选择后会改变什么
```

不把完整 Agent transcript 当作默认决策 UI。

---

# 19. Attention

v1.4 不把 AttentionItem 作为主要持久业务对象。

Attention 是 Derived Projection。

合法来源只包括：

- needs clarification；
- unmatched risk；
- policy conflict；
- assurance missing；
- repair exhausted；
- human gate；
- waiver；
- infrastructure/provider blocking failure。

持久化只需要：

- dismissal；
- snooze；
- human decision evidence。

---

# 20. Minimal Persistent Domain Model

v1.4 持久化核心对象限制为：

```text
1. WorkItem
2. Contract
3. Policy
4. ChangeSet
5. Evidence
```

另加：

```text
OutcomeLog
```

作为 append-only operational log。

---

# 21. WorkItem

```text
WorkItem
├── id
├── parent_id
├── type
├── source_ref
├── intent
└── revision
```

type 可以包括：

```text
feature
defect
chore
engineering
```

type：

- 可以影响 Policy；
- 可以影响 UI projection；

不得创建每类 WorkItem 独立状态机。

---

# 22. Derived Model

以下不作为主要业务对象持久化：

```text
RiskSignals
PolicyMatchCoverage
VerifiedBasis
AttentionQueue
WorkState
ReworkRate14d
```

必须可以由底层事实稳定重算。

---

# 23. OutcomeLog

从 v1.4 开始立即采集：

```text
work_item_id
changeset_id
accepted
reverted
hotfixed
regression_found
human_correction
timestamp
```

v1.4 不建设完整 Outcome UI。

目的：

> 这些数据事后无法完整补采。

---

# 24. MVP Change Types

v1.4 Final 只要求两类 Harness。

## 24.1 Bug Fix

目标闭环：

```text
Intent
→ reproduce
→ acceptance
→ Agent
→ regression evidence
→ Repair if needed
→ verdict
```

重点验证：

- false complete；
- regression；
- auto repair；
- human intervention。

## 24.2 Database Migration

目标闭环：

```text
Intent
→ migration requirements
→ Agent
→ fresh DB
→ N-1 upgrade
→ data integrity
→ repair
→ verdict
```

重点验证：

- Contract-as-prevention；
- strict assurance；
- held-out / protected verifier 的增量价值。

---

# 25. v1.4 不实现的 Harness

当前明确不实现：

- Public API full Harness；
- Authentication full Harness；
- Performance full Harness；
- UI visual full Harness；
- Release Harness；
- Incident Harness。

未来只有在 MVP 价值成立后提升。

---

# 26. UI Scope

v1.4 UI 不建设完整 Jira 式 IA。

必须实现的核心面：

## 26.1 Launch

```text
Intent
source_ref optional
generated Contract diff
Run
```

## 26.2 Task / Closed Loop Detail

一个页面显示：

```text
Intent
Contract
Current Run
Current ChangeSet
Verification
Repair attempts
Verdict
Human action if required
```

## 26.3 Change Detail

回答：

```text
改了什么？
为什么可信？
哪里不确定？
需要我做什么？
```

## 26.4 Attention

只显示真正需要人的事项。

v1.4 不要求：

- Requirements 一级模块；
- Release 一级模块；
- Quality dashboard；
- Knowledge module；
- Roadmap；
- Sprint；
- Metrics 大盘。

---

# 27. Structured UI 与自然语言

规则：

## 输入

Natural language first。

## 记录

Structured first。

## 修正

Structured diff 或 natural language 都可以。

## 判断

Structured options first。

失败模式不是“使用表单”。

失败模式是：

> 让用户自己填表单。

---

# 28. Runtime Boundary

Tally AgentRuntime 与公共 AI Runtime 继续严格分离。

## AgentRuntime

负责：

- coding workspace；
- shell；
- session；
- patch；
- files；
- coding agent execution。

## AI Runtime

负责通用 LLM：

- provider；
- model routing；
- common retry；
- cost；
- trace；
- rollout。

Tally Domain 只声明 Operation / intent / structured input。

---

# 29. v1.3.1 Trust Closeout

在正式认为 v1.4 Core 可用前，必须完成：

1. CodexRuntime Session Isolation；
2. Fake + Codex 同一 isolation canary；
3. real Codex standard + held-out E2E；
4. actual ModelProfile 真正传给 Runtime；
5. actual runtime/model/CLI/session/conformance trace；
6. Measurement `UsageAvailable` 修复；
7. token unknown 时 runtime call 仍持久化；
8. Exposure ancestor lineage 可见；
9. Baseline readout operationalization；
10. 通用 LLM client 迁入 AI Runtime；
11. 文档与代码状态同步。

---

# 30. Product Value Experiment

v1.4 的价值不能只靠作者主观判断。

必须预注册实验：

- task set；
- metrics；
- thresholds；
- evaluation method；
- exclusions。

保存进 Git 并打 tag。

## 30.1 对照

```text
Bare Coding Agent
vs
Tally Closed Loop + Same Agent
```

## 30.2 样本

首轮至少覆盖：

- Bug Fix；
- Database Migration。

数量根据已有真实 Task 能力推进，但最终目标应达到足以判断趋势的一组固定任务。

不得中途因为结果不好替换 Task。

---

# 31. 核心产品指标

不采用单一“Trust Score”。

## 31.1 Primary — Hands-off Completion Rate

定义：

> 从 Intent 到 `ACCEPTABLE / READY_FOR_HUMAN_DECISION` 的过程中，不需要人手工修代码、复制失败、重新发 prompt、重新启动 verification 的 Task 比例。

Human 查看 Diff 不算失败。

真正希望减少的是：

> Human orchestration labor。

## 31.2 Primary — Time to Accepted Change

```text
Intent
→
ACCEPTABLE / accepted change
```

## 31.3 Primary — Human Intervention per Accepted Change

记录离散事件：

- clarification；
- manual repair；
- manual rerun；
- waiver；
- trade-off decision；
- forced retry；
- direct code correction。

## 31.4 Fast Guardrail — 14-day Rework Rate

同一 WorkItem / Change 在 14 天内是否因同一根本原因重新修改。

## 31.5 Slow Guardrail

- escaped defect；
- rollback；
- hotfix；
- regression；
- production incident related to accepted change。

---

# 32. Success Criteria

v1.4 成功至少意味着：

1. Tally 路径能够自动完成 Execute → Verify → Repair；
2. 用户不再需要把失败手工复制回 Agent；
3. Contract 在执行前真正影响 Agent 行为；
4. VERIFIED / ACCEPTABLE 有清晰 basis；
5. Assurance strength 与 risk 匹配；
6. Tally 相对 bare agent 没有不可接受的启动摩擦；
7. Human intervention 明显下降；
8. 14-day rework 没有因为自动化而恶化；
9. OutcomeLog 开始形成真实历史。

---

# 33. Kill / Pivot Signals

## K1 — 没有减少人工中介

如果：

```text
Hands-off Completion Rate
```

长期很低，且主要失败来自：

- Tally 自己的流程；
- Contract 生成；
- Verification noise；

则 Pivot。

## K2 — Time to Accepted Change 变慢

如果长期：

```text
Tally > bare agent + human loop
```

且差异无法由质量改善解释，则 Pivot。

## K3 — Human Intervention 不下降

如果使用数周后：

```text
intervention / accepted change
```

没有下降，则核心价值不成立。

## K4 — Rework 增加

如果自动化后：

```text
14-day rework
```

明显增加，则不能继续追求 hands-off。

## K5 — 用户绕过

如果真实 Coding Task 大量直接绕过 Tally，说明摩擦过高。

## K6 — Policy noise

如果：

- policy block；
- waiver；
- false positive；

成为日常噪音，则缩减 Policy。

## K7 — Assurance cost 失衡

如果 L3 成本远高于实际增量价值，则降为 strict-only。

---

# 34. Roadmap

## NOW

1. v1.3.1 Trust Closeout；
2. Assurance bake-off；
3. Value experiment pre-registration；
4. OutcomeLog；
5. Contract merge / freeze；
6. Policy.requires → Agent Context；
7. Auto Repair；
8. `VERDICT(level,basis,coverage,unmatched)`；
9. Bug Fix Harness；
10. Migration Harness；
11. 最小 Launch / Closed Loop UI。

## NEXT

只有 NOW 验证成立后：

1. project guidance passthrough；
2. Attention projection；
3. assurance artifact export；
4. second AgentRuntime；
5. broader provider integration；
6. fixed-task blind comparison；
7. 14-day rework reporting。

## DEFER

- Public API Harness；
- Auth/Security Harness；
- Release / ReleaseGate；
- Incident；
- Requirements authoring；
- Quality full UI；
- Project Engineering Model；
- Adaptive Harness；
- enterprise Jira/GitLab bidirectional sync。

## REMOVE FROM v1.4

- System of Record 作为定位；
- Sprint；
- Story Point；
- Roadmap object；
- Knowledge module；
- full Guidance Plane；
- HR/workload/time tracking；
- full DevSecOps replacement。

---

# 35. Product Invariants

1. Agent done 永远不是 Delivery Fact。
2. Acceptance semantics 必须在执行前冻结。
3. Agent 无权静默修改 Acceptance。
4. Agent 无权静默降低 Assurance。
5. ChangeSet 进入 Verification 后 immutable。
6. Repair 每次产生新 ChangeSet。
7. 每个 Verdict 必须绑定具体 ChangeSet。
8. 每个 VERIFIED / ACCEPTABLE 必须携带 basis。
9. `satisfied` 与 `waived` 永不合并。
10. unmatched risk 不得静默走最低 assurance。
11. Human 不负责搬运机器可自动处理的失败。
12. 自动 Repair 有硬上限。
13. Project guidance 可以透传，但 v1.4 不建设独立 Guidance 平台。
14. Runtime/model 是 execution facts，不改变 Contract authority。
15. View 不拥有 State。
16. Outcome history append-only。
17. Tally 必须优先减少 Human orchestration，而不是最大化 Agent activity。

---

# 36. Final Definition

> **Tally 是 Coding Agent 的闭合软件工程执行环。**
>
> 用户给出 Intent，Tally 冻结 Acceptance Contract，并把可验证要求提前提供给 Agent；Agent 在隔离环境中执行并产生 immutable ChangeSet；Tally 根据风险选择 Assurance，收集 Evidence，并在失败时自动带着结构化失败依据重新 Repair。只有系统无法安全继续时才找人做判断。
>
> **验证不是产品终点。验证是让 Agent Task 可以真正“脱手运行”的引擎。**
