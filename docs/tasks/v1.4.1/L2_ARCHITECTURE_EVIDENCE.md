# Tally v1.4.1 — L2 Architecture Evidence Research

**Status:** COMPLETE  
**Architecture Verdict:** `SUPPORTED_WITH_CHANGES / READY_AFTER_FIXES`  
**Source baseline:** `979a55ad3ba5e514bb80f0e012aa36e0e6577eba`  
**PRD input:** `Tally_PRD_v1.4.1_Code_Aligned_Revision.md`  
**Research date:** 2026-09-11

---

## 1. L2 Objective

本轮 L2 不是证明 PRD 原方案正确，而是验证：

1. v1.4.1 是否能从当前 v1.4 brownfield 代码自然演进；
2. 哪些能力已经有稳定 seam，应该复用而不是重建；
3. 哪些设计在接入生产主链时会形成双重权威或语义冲突；
4. 哪些新能力有成熟工程模式和外部证据支持；
5. 哪些需求应进入 v1.4.1，哪些应降级为 Spike / Canary / NEXT；
6. 最终如何拆成可并行、可验证、低成本模型友好的 Task DAG。

L2 原则：

> Constraints > Fashion  
> Evidence > Popularity  
> Failure Modes > Happy Path  
> Operational Simplicity > Premature Distribution  
> Observed Scale > Imagined Scale  
> Standards > Blog Opinion  
> Reproducible Validation > Anecdote

---

# 2. Executive Verdict

v1.4.1 **可以从当前代码演进，不需要推倒重构**。

但是，原 PRD 中有若干设计如果直接实现，会造成新的重复架构。L2 后的正式结论是：

| Area | Verdict | L2 Decision |
|---|---|---|
| Closed Loop | KEEP / FREEZE | 保留现有 Execute→Verify→Repair→Accept/Escalate |
| Disposable worktree / ChangeSet | KEEP / FREEZE | Tally 继续持有 workspace / Git / candidate authority |
| Confirm→Freeze | MUST FIX FIRST | 当前生产链未接通，P0 解决 |
| Risk input authority | CHANGE / BLOCKER | 增最小 immutable FrozenIntent companion artifact |
| Contract Compiler | KEEP / MUST WIRE | Production 必须调用 Bind，但不能只“调用一下” |
| Assurance acceptance | CHANGE / BLOCKER | 必须防止 `VERIFIED_LIMITED` 被 Closed Loop 当 Accepted |
| Dynamic Form | KEEP WITH CHANGES | 基于现有 Intake Clarification 扩展 bounded presentation schema |
| Rule-first Review | KEEP WITH CHANGES | Governance layer over EngineeringPolicy，不建第二套 Rule Engine |
| Historical Replay | KEEP WITH LIMITS | 只做 evidence-bounded replay，不伪造反事实 acceptance |
| Harness abstraction | CHANGE PRD | 复用 `agentruntime.Runtime`，不新增第四层 ExecutionBackend |
| Codex | KEEP PRIMARY | 不把 native cross-provider subagent 当 release dependency |
| DeepSeek Harness | SPIKE → CANARY | 作为第二 Runtime adapter，先 conformance / safety spike |
| Git Control Plane | KEEP INCREMENTAL | v1.4.1 做 L0–L3；Remote/PR 不作为 release blocker |
| Outcome | CHANGE PRD | 保持现有 5 个 Outcome kinds，不混入 workflow failure |
| Human-centered UI | KEEP AS PROJECTION | 不改 state machine，不要求 SPA 重写 |
| AI Runtime DIRECT/AUTO/ORCHESTRATED | OUTSIDE TALLY | Tally 只消费 unified planning result / provenance |
| Safe Export / docs drift | DO NOW | 属于工程基线卫生和安全门 |

---

# 3. Source-Level Evidence

## 3.1 Closed Loop 已经是稳定核心

当前源码中：

- `internal/closedloop/ports.go` 定义窄端口；
- `internal/closedloop/executor.go` 的 `WorkspaceExecutor` 持有 disposable worktree；
- `internal/closedloop/wiring.go` 组合 Executor / Verifier / Failure Router / Repair / Exposure；
- `cmd/worker/v14.go` 通过 `v14CodegenRunner` 把 runtime 执行结果冻结为 ChangeSet。

**结论：**
不要为多 Harness 重建 Closed Loop。新的 Runtime 只能替换 Execute primitive。

---

## 3.2 当前已经有 Runtime abstraction

`internal/agentruntime/protocol.go` 已经定义：

- `Runtime`
- `Session`
- `Execute`
- `Cancel`
- `Events`
- `Result`
- `Close`

`runtimes/codex` 已经实现该协议。

**结论：**
PRD 中新增的泛化 `ExecutionBackend` 会与现有 `agentruntime.Runtime` 重叠，必须删除。  
DeepSeek Harness 应实现 `agentruntime.Runtime` adapter。

---

## 3.3 Confirm→Freeze 尚未进入生产

`internal/web/handlers/v14_launch.go` 的 `V14ConfirmDraft` 当前只验证并重新渲染 confirmed draft；源码注释明确说明 Draft→TaskContract / VerificationContract Freeze 尚未接线。

同时 `SetV14ProposeExecutor` 有注入 seam，但生产 composition root 尚未完成真实接线。

**结论：**
这是 v1.4.1 第一优先级，而不是 UI 美化问题。

---

## 3.4 Contract Bind 尚无生产调用

`internal/contract/compile.go` 中已经存在 `Bind(...)`，但生产 `cmd/worker` 路径没有调用点。

`taskstate` 已能消费 `Compiled` 生成 VerifiedBasis，但当前生产读链缺少 Compiled authority。

**结论：**
“Compiler 已实现”不等于“Compiler 已成为执行权威”。

---

## 3.5 风险事实存在 authority gap

当前 Intake Draft 中可出现候选 RiskSignals，但它们明确只是 proposal，并不会自然成为 immutable execution fact。

同时：

- `risk.Result` 是 derived-only；
- `TaskContract` 不保存 task_type / risk hints；
- `contract.Bind` 需要 `risk.Result`；
- `engineeringpolicy.MatchInput` 的 risk axis 需要真实 basis refs；
- pre-execution 时 actual changed paths 尚不存在。

**结论：**
如果直接把 Bind 接上而传空 risk，系统会“看起来启用了 risk/assurance”，实际上静默漏掉高风险任务。

L2 建议新增最小 immutable companion artifact：

`FrozenIntent` / `IntentEvidence`

建议字段仅包含：

- task_id
- task_type
- source_ref
- confirmed risk hints / declared facts

它是 risk derivation 的底层事实，不是另一个 RiskProfile 状态对象。

---

## 3.6 当前存在 Assurance Acceptance Gap

`verification.FinalVerified` 会被 `closedloop.ServiceVerifier` 映射为 `VerdictVerified`。

但是 `taskstate.VerifiedBasis` 已经支持：

- required assurance
- achieved assurance
- unmatched risk
- `VERIFIED_LIMITED`

因此存在一种危险路径：

> 基础 VerificationContract 全部门通过 → `FinalVerified`  
> 但 CompiledContract 要求更高 Assurance / 存在 unmatched risk → `VERIFIED_LIMITED`  
> Closed Loop 如果只看 FinalVerified，仍可能 Accept。

**结论：**
v1.4.1 必须增加 Compiled / Assurance Acceptance Guard。  
这是 release blocker，不是 UI/read-model 优化。

---

## 3.7 EngineeringPolicy 已经是 runtime rule authority

`internal/engineeringpolicy/policy.go` 的 schema 已经很窄，并且 revision 来自内容。

`internal/artifacts/v14_policy_loader.go` 从 **指定 Git commit tree** 加载 `.tally/policies/*.yaml`，而不是 working tree。

**结论：**

- Rule-first Review 是合理方向；
- 但不能把 `DRAFT/PROPOSED/ACTIVE/...` 等治理字段塞进 `engineeringpolicy.Policy` YAML；
- Runtime Active 的唯一事实应继续是：**当前 source commit 里真实存在、可解析的 policy blob**；
- Proposal / approval / replay 是独立治理事实；
- Approve & Activate 最终动作是通过 GitWriter 提交 canonical policy YAML。

这样避免：

> DB 说 ACTIVE，但 Git source snapshot 是另一份 policy

的 split brain。

---

## 3.8 OutcomeLog 已有稳定语义

`internal/outcome/types.go` 当前明确只有五个事件：

- accepted
- reverted
- hotfixed
- regression_found
- human_correction

**结论：**
不要为了 v1.4.1 把 rejected / escalated / integration_failed 强塞进 OutcomeLog。

这些属于：

- workflow fact
- attention fact
- integration fact

而不是长期产品 Outcome。

---

# 4. External Architecture Evidence

## 4.1 Git worktree supports current isolation design

Git 官方 `git-worktree` 提供多个 linked worktree，以及 detached worktree 语义。

这直接支持 Tally 当前：

> owner repo + disposable detached worktree

的模型。

Reference:
- https://git-scm.com/docs/git-worktree

**Decision:** KEEP / FREEZE workspace ownership.

---

## 4.2 Bounded declarative dynamic forms are preferable to arbitrary generated UI

JSON Schema Draft 2020-12 提供成熟的声明式 schema / validation 思路。

Tally 不需要引入完整 JSON Schema renderer，但它证明：

> dynamic structure should still be constrained by a stable declarative vocabulary.

Reference:
- https://json-schema.org/draft/2020-12

**Decision:** Dynamic Form uses a bounded Tally-owned DTO/DSL. No arbitrary HTML/JS.

---

## 4.3 Rule Evaluate before Enforcement is established engineering practice

GitHub Repository Rulesets supports:

- named rulesets;
- layered rules;
- active/evaluate modes;
- Rule Insights;
- seeing which actions would pass/fail if a ruleset were active.

Reference:
- https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository

GitLab supports approval rules / policies and protected-branch-oriented governance.

References:
- https://docs.gitlab.com/user/project/merge_requests/approvals/rules/
- https://docs.gitlab.com/user/application_security/policies/merge_request_approval_policies/

**Decision:** Tally RuleProposal + Historical Replay + explicit activation is strongly supported.

---

## 4.4 Codex is strong as primary runtime, but cross-provider subagents are not a safe release dependency

Current Codex `spawn_agent` tool exposes child model/reasoning controls in its public model-facing spec, but provider routing is not an equally stable first-class surface.

Current upstream issue evidence:

- explicit subagent `model_provider` override can be ignored while model override works;
- Multi-Agent V2 can emit OpenAI-specific `agent_message` payloads that external Responses providers cannot consume.

References:
- https://github.com/openai/codex/blob/main/codex-rs/core/src/tools/handlers/multi_agents_spec.rs
- https://github.com/openai/codex/issues/40858
- https://github.com/openai/codex/issues/33551

**Decision:** Codex stays primary; do not make native cross-provider children a v1.4.1 gate.

---

## 4.5 DeepSeek Harness has the right multi-provider seam, but is not production-ready security authority

DeepSeek Harness currently has:

- named subagent provider registry;
- Codex / Claude Code / ACP / DSH SDK sibling providers;
- provider/model/reasoning selection in tool-subagent;
- explicit model selection policy.

References:
- https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/subagent.md
- https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/subagent/tool-subagent/src/list-models.ts
- https://github.com/deepseek-ai/deepseek-harness/blob/master/.agents/notes/implemented/feature/2026-08-04-claude-code-and-codex-subagent-backends.md

But its own SAFETY document explicitly labels it experimental developer preview and says it is not audited or production-ready, and should not be the sole security control.

Reference:
- https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md

**Decision:** Add as secondary runtime through Spike → Canary → Limited Ready, while Tally owns workspace/Git/verification/acceptance.

---

# 5. Most Likely Failure Modes

## F1 — Source Commit Split Brain

**Trigger:** Task/verification loaded from commit A; policies or intent/risk inputs loaded from B.  
**Weakness:** Compiled decision cannot be reproduced.  
**Blast radius:** Wrong assurance / wrong acceptance.  
**Prevention:** Load all authoritative artifacts from one pinned source commit; compiled hash records inputs.

---

## F2 — Assurance Bypass

**Trigger:** VerificationContract passes but CompiledContract requires higher assurance.  
**Weakness:** `FinalVerified` maps directly to Closed Loop Accepted.  
**Blast radius:** High-risk candidate accepted under insufficient evidence.  
**Prevention:** Compiled acceptance guard; `VERIFIED_LIMITED` cannot become Accepted.

---

## F3 — Policy Governance Split Brain

**Trigger:** DB says proposal is ACTIVE while Git policy content differs.  
**Weakness:** Two policy authorities.  
**Blast radius:** UI shows one rule; runtime executes another.  
**Prevention:** Git snapshot is runtime authority. Governance approval only authorizes a GitWriter activation commit.

---

## F4 — Dynamic Form Becomes Dynamic Contract

**Trigger:** LLM invents arbitrary fields/UI/schema.  
**Weakness:** Presentation controls domain semantics.  
**Blast radius:** contract drift, prompt injection, impossible validation.  
**Prevention:** bounded DTO/DSL; strict server validation; explicit mapper to frozen domain artifacts.

---

## F5 — Runtime Abstraction Duplication

**Trigger:** New ExecutionBackend + existing agentruntime + CodegenRunner coexist as peer authorities.  
**Weakness:** lifecycle/cancel/result ownership becomes ambiguous.  
**Blast radius:** leaked processes, inconsistent failure semantics.  
**Prevention:** reuse `agentruntime.Runtime`; keep CodegenRunner as closedloop bridge.

---

## F6 — DSH Treated as Production Security Boundary

**Trigger:** DSH gets owner repo/credentials/direct Git authority.  
**Weakness:** upstream explicitly says developer preview / not security-audited.  
**Blast radius:** host/file/credential damage.  
**Prevention:** disposable workspace, least privilege, no protected Git authority, canary rollout.

---

## F7 — Codex Cross-Provider Assumption

**Trigger:** Tally relies on OpenAI parent + arbitrary provider child.  
**Weakness:** current provider override/wire compatibility issues.  
**Blast radius:** wrong provider, task never reaches child, nondeterministic execution.  
**Prevention:** not a v1.4.1 dependency; use DSH for multi-provider needs.

---

## F8 — Historical Replay Overclaims Counterfactual Outcome

**Trigger:** Replay says “this old task would have been accepted/rejected” without sufficient historic evidence.  
**Weakness:** policy simulation exceeds available facts.  
**Blast radius:** human activates unsafe rule.  
**Prevention:** replay only deterministic match/effect deltas; emit INSUFFICIENT_EVIDENCE where outcome cannot be inferred.

---

## F9 — Git Scope Explosion

**Trigger:** v1.4.1 expands into fetch/push/PR/MR/release automation.  
**Weakness:** project closure becomes hosting integration project.  
**Blast radius:** schedule slip and new credential/security surface.  
**Prevention:** v1.4.1 release gate ends at controlled local integration L3. Remote is optional/NEXT.

---

## F10 — Outcome Semantic Pollution

**Trigger:** workflow failures added to OutcomeLog.  
**Weakness:** “long-term result” and “run state” become same thing.  
**Blast radius:** measurement and rule-learning evidence become misleading.  
**Prevention:** retain current five kinds.

---

## F11 — Parallel Agent File Collision

**Trigger:** low-cost tasks concurrently edit worker/launch/policy hotspots.  
**Weakness:** task DAG ignores path ownership.  
**Blast radius:** merge conflicts and accidental semantic overwrite.  
**Prevention:** explicit path ownership and sequential waves for hotspots.

---

## F12 — Local Dependency Fetch Failure Misclassified as Code Failure

**Trigger:** offline environment runs uncached `go test`.  
**Weakness:** dependency fetch error interpreted as regression.  
**Prevention:** preflight dependency availability; CI/connected environment is authoritative gate.

---

# 6. PRD Amendments Produced by L2

The following amendments are authoritative for v1.4.1 implementation.

1. **DELETE** the proposed new generic `ExecutionBackend` core abstraction.
   - Reuse `agentruntime.Runtime`.
   - DeepSeek Harness implements a runtime adapter.

2. **ADD** `FrozenIntent` / `IntentEvidence` as a minimal immutable source artifact.
   - It preserves confirmed intake facts required for risk derivation.
   - It is not a mutable RiskProfile.

3. **ADD** a Compiled/Assurance Acceptance Guard.
   - Production may not accept when required assurance is not achieved or risk coverage is incomplete.

4. **CHANGE** Rule lifecycle implementation.
   - Governance lifecycle is not stored inside runtime `EngineeringPolicy`.
   - ACTIVE runtime policy means the exact policy blob in the pinned Git snapshot.

5. **LIMIT** Historical Replay.
   - It evaluates match/effect using historical facts.
   - It does not fabricate counterfactual code/test outcomes.

6. **CHANGE** Outcome scope.
   - Keep exactly the existing event semantic set unless a separate future ADR explicitly changes it.

7. **LIMIT** Git Control Plane release scope.
   - v1.4.1 requires local controlled integration (L0–L3).
   - Remote push/PR/MR is not a release blocker.

8. **CHANGE** DeepSeek Harness maturity.
   - Secondary/canary runtime, not equal production trust on day one.

9. **KEEP** UI as projection.
   - Do not rewrite lifecycle/state machine.
   - Dynamic form stays presentation-only.

10. **KEEP OUTSIDE TALLY** AI Runtime planning modes.
    - Tally records unified planning result/provenance only.

---

# 7. Architecture Freeze Recommendation

Freeze these boundaries before implementation:

- `closedloop` owns Verify/Repair/Accept/Escalate;
- `workspace` owns execution isolation;
- `gitwriter` owns durable Git writes;
- `changeset` owns candidate identity/lineage;
- `agentruntime` owns coding-runtime lifecycle protocol;
- `engineeringpolicy.Policy` remains runtime policy schema;
- Git source commit remains runtime artifact authority;
- `outcome` keeps current event semantics;
- hidden verification material remains unavailable to execution prompts;
- one task has one primary runtime owner.

Do not freeze yet:

- FrozenIntent exact file path/schema details;
- exact acceptance-guard placement (needs L3 implementation reference);
- DeepSeek headless transport shape (needs spike);
- Rule replay storage/query implementation;
- local integration service API.

---

# 8. Validation Baseline

Attempted focused baseline:

```bash
go test ./internal/contract ./internal/engineeringpolicy ./internal/intake ./internal/closedloop ./internal/agentruntime
```

The current analysis environment could not fetch uncached Go modules from `proxy.golang.org`, so the run was blocked by dependency/network availability rather than a source assertion failure.

Implementation MUST run the baseline in a connected/cached environment before changing production code and record:

- Go version;
- dependency lock identity;
- command;
- pass/fail;
- failing package/test names;
- whether failure is baseline or introduced.

---

# 9. Final L2 Status

**Architecture:** `SUPPORTED_WITH_CHANGES`  
**Implementation readiness:** `READY_AFTER_FIXES`  
**Task pack:** generated  
**L3 required before selected high-risk tasks:** yes  
**Greenfield rewrite:** rejected  
**Brownfield incremental implementation:** recommended
