# Tally v2 重构整合与交付路线 R0（Agent-first / 中文）

> **状态：Product/L2 整合提案，非冻结需求、非已执行 Task DAG、非源码授权。** 来源：2026-10-28 用户明确「Tally 用软件工程约束 Vibe Coding；实际开发任务默认由各种 LLM Agent 完成，人只在必要时介入；用户在 Tally 内完成软件整个开发过程」。本文件把该产品方向对接**当前在途** Tally Product PR #77、L2 PR #82、ADS #85、版本 #86、Agent #64、Package B #91、UI #92 和工作流 #145。实际正式 Product/L2 决议须按已采用 ADS 和独立源审核完成。

## 0. 一句话 Product Freeze 目标

**Tally v2 = Agent-native Software Engineering Domain App**：用户输入目标、预算、约束和授权，Tally 用软件工程的 Requirements/Design/Construction/Verification/Maintenance/Release 契约组织**能够真正工作的 LLM Agents**，让他们合法地并行拆需求、写代码、测试、独立审核、修复和准备发布；只向用户呈现真正必须由人决定、授予或物理操作的事项。最终可从项目创建持续到发布和下一个版本，不要求用户每一步手动说「继续推进」。

**业务成功 ≠ Agent 提示词成功、Issue CLOSED、PR merged、模拟 UI、K 对 Fake Agent 执行、CI 测试计数。** 必须证明真实已选 Domain Package→实际 Agent/Host/GitHub effect→真实 Review/CI/外部用户行为→合法发布。

## 1. 已存在基础与当前资格（截至本研究检索的 GitHub 锚点）

| 来源 | 当前可用或待审核内容 | 当前不能宣称 |
|---|---|---|
| [version/v2 #86](https://github.com/kaicreator-mm/tally/issues/86) | 控制器在 2026-10-27 记录 `version/v2@de5b937b...`、已合入该版本分支的 14 个独立审核阶段 PR；已做源码集成，不必重写 | 不是已合 main/v2 正式发布 |
| [C GitHub PR #120](https://github.com/kaicreator-mm/tally/pull/120) | 在版本分支 merged 的真实 selected DomainHarness C→K→SQLite / GitHub 读取能力 | 单纯 source/reader 成功不等于可接受的 Task Claim 或完整 Task |
| [T06a PR #127](https://github.com/kaicreator-mm/tally/pull/127) | 已 merged：真实 GitHub native Issue/blocked_by 读取，能够正确拒绝缺 Claim 的派发 | 并未证明真正 CAS 接受 |
| [T06b PR #129](https://github.com/kaicreator-mm/tally/pull/129) | Draft 未合并，ADS Claim/Dispatch 来源事件读取候选，Node24 局部验证 | 不能将声明/评论视为合法 CAS |
| [T06c PR #131](https://github.com/kaicreator-mm/tally/pull/131) | Draft 未合并，Engineering B 任务依赖/Claim 诊断建议 | 不会自动为 K 派发 Task |
| [T06d PR #133](https://github.com/kaicreator-mm/tally/pull/133) | Draft 未合并，Host SQLite 真 CAS + mock GitHub 正向 publisher/readback，6/6 Node24 | mock publisher 不是真实 GitHub accepted Claim，真实 K Admission/Task E2E 仍未通过 |
| [ADS adoption PR #85](https://github.com/kaicreator-mm/tally/pull/85) | Draft，ADS v4.18 exact-pin adoption candidate | 没合入项目有效 Pin |
| [Product PR #77](https://github.com/kaicreator-mm/tally/pull/77) | Draft HEAD `b80594e2391617d9527f669e6b3d081ab98215fb`；17 个 P0、67 个 V spec、功能保留证据部分核实 | 独立 Product R3 对 47 功能 parity P1 仍 BLOCK；不能当作 frozen |
| [L2 PR #82](https://github.com/kaicreator-mm/tally/pull/82) | Draft HEAD `0a58f2dad881f5634575e2d0cf3107470ec97de2`；R5 exact-source semantic PASS | Product/ADS 真实 Freeze 未全部通过；L2_FREEZE_READY=NO |
| [DH stable](https://github.com/kaicreator-mm/domain-harness/releases) | v0.10.2 已有 immutable tag `cee14399d9d0af4e20e34b9111861b16a465b690`，真实公共 BUILD/K/Host 有部分合格 Consumer 证据；DH RQ CONDITIONAL | 不因 DH 局部 PASS 就说 Tally v2 实际发布或所有 public ABI 已满足 |
| [DH 0.10.3 sync PR #135](https://github.com/kaicreator-mm/tally/pull/135) | 候选 source 兼容性检查，不是已发布的 v0.10.3 升级 | 当前不能强制要求 v2 使用不存在的合格 v0.10.3 release |

**原则**：已经通过当前适用版本的 exact-source 独立审核/真实物理实验应能沿用；候选 PR 不因为本研究自动合并或废弃；不覆盖真实在途 Claim，不修改其他 Agent 的写集。前提是所有适用冻结/变更和安全准入在物理 effect 前均真实满足。

## 2. 必须 Product Owner 正式决定的差异（这才决定能否以该设计完整结束 v2）

| 差异 ID | 当前 Product/L2 精确现状 | 本轮确定的 Agent-first 用户结果 | 最小 Product/L2 纠正及验收 |
|---|---|---|---|
| **D01 P0 Agent-first** | PR #77 §1 `users (with Agents)`，P0-02/13 和 L2 5.2 以 UI operator 的主动提交为中心；配置过 Agent 角色 | 日常 Planner/Architect/Builder/Reviewer/Verifier/Integrator **默认是 LLM Agent**；用户只需描述目标、授权、处理例外 | 在 Product 明确 `AUTONOMOUS_ELIGIBLE_AGENT_NEXT_STEP` 与 `HUMAN_REQUIRED_REASON`；同一真实 CJ 不需人工中转每个 Review/Task；不允许自动授权 |
| **D02 P0 并行目标** | PR #77 §4/P0 Local 与 Product Vector **V63**、L2 §6 **明确将「最多 1 个执行中的 Task Attempt」设为 P0**，并发 P1 | 用户要求多个 LLM Agent 协同、最大安全并行；安全独立 Concern 不应排队 | **产品 Owner 需在 Freeze 前裁定**：建议将 **至少两个合法写集隔离/实际可并行 Agent Builder Attempts** 升为完整 v2 P0 验收；原 max1 保留为先行 Vertical Slice A 的安全阶段，非最终完整交付。仅 UI 显示 parallel-ready 不能满足；不同 Agent 写同一 Claim/写集必须拒绝。若选择继续单执行，应明确 v2 只能叫受限 MVP，不得称 Agent-first 完整交付 |
| **D03 Agent Provider 扩展** | PR #77 P0-06 `Codex CLI mandatory first`；Claude Code optional，尚未把多 Agent Provider 的组合验证作为具体 CJ | 多种基于 LLM 的 Coding Agents，按任务适配 Agent 能力/Host/上下文/成本；Reviewer 与 Builder 真独立 | 保留 Codex 为第一个可执行真实 Provider，**Provider-neutral B Registry/typed C Adapter P0**；增加第二个**独立真实** Agent Provider 的选配/任务角色/能力/拒绝与来源验收（Claude Code 或 ZCode 有真 CLI 能力证明后选），不能用 Fake/echo 充数；是否第二 Provider 纳入 Release P0 由 Product Owner 判定，不强推所有 CLI/模型进入 v2 |
| **D04 自动继续与 Human Inbox** | P0-12 `required user decision where applicable`、P0-13 UI；缺日常长期 Version Driver/Agent→Agent 恢复 P0 验收 | 每个合法 next step 由 B + K/C 继续；仅必要时通知人，不能无界后台任务 | #145 的 `nextActions` 增添 Agent 默认执行、预算/权限/Role/currentness、独立 Agent Reviewer、人工例外理由；Agent/Host 不具资格则 WAIT/ABSTAIN。后台工作必须是实际 Host/Agent 服务，有状态/受控生命周期，不是 ChatGPT 聊天持续任务承诺 |
| **D05 用户验收与 Version 进化** | PR #77 V01–V67 多为单 Task、安全及 UI vector；#92 J01–J08 为后来补充的完整用户目标 | 从用户能力定义到 **真实 selected/installed 行为证明**、发布与下一版保留 | #146 的 Requirement→Task/PR→physical validation→Release 账本及 #92 whole-software CJ 应加入正式 Product 接受标准（复用现有 V，缺的只补增量）；历史安全不变量不可静默退役 |
| **D06 审核职责 ≠ 人工审批** | Product 已定义逻辑独立 Agent/Verifier，但是部分 UI/operator 语句可能被实现成每阶段人工点击 | 独立 Reviewer **默认可为合格 LLM Agent**；小任务减少不必要角色切换 | 定义 `ReviewRoleProvenance` / true independence 的负例：Builder 改 role 标签≠独立 Agent；required independent source Review 和适用 Owner 授权仍必须满足 |
| **D07 P0 安全/成本适用性** | P0-16/17 single-machine Builder/Hidden principal 硬要求；早期版本可能被大量 Host 准备占用 | 风险与价值优先，Human 仅真实物理/权限要求时介入 | **不擅自削弱既有冻结的独立验证/Hidden/P0 授权规则**；审核其适用 Release Profile，并在 Agent-capability-aware Host 安排里用真实 CI/隔离 Runner 优先复用。仅需普通 unit/CI 的 Leaf 不强制 Full/Hidden 或单独人类 Host |
| **D08 历史功能归档 P1** | PR #77 R3 对 retained 31 个 P0 Function parity 尚有部分旧行为未能建立完整函数/测试→当前规范证明 | 用户需真实可用的软件开发流程，不能耗尽精力补证明不再适用的旧 Go 细节 | 独立 Product Owner 在 47 功能矩阵逐项明确 `RETAIN/ADAPT/RETIRE/UNKNOWN`，对 15 已 source 证明的高风险项和安全 P0 追求真实验收，其余真实重要能力补证；如需调整原先「全 P0 legacy reachability」条件必须走明确 Product amendment，不得直接把 P1 当已解决 |
| **D09 进化第一期范围** | #57/#58/#60/#139–#144 是 Product Intake；无全部物理 Package/任务 Claim | task-meta→ADS→Tally 能基于实战改进选 Agent/规则/代码/测试，而不靠 LLM 自评 | v2 P0：**源证据图谱+可复用上下文+工程反馈入口**；复杂的跨项目 Shadow/A-B/自动 Package candidate Promotion 可以 P1/后续（除非 Product 单独决定），不得阻塞真实软件发布 |

**判定**：D01/D02/D03/D04/D05/D06 是否属于 v2 P0 目标必须由 Product #63/#76 在下一合法 Review 中明确处理，随后 #78/#82 L2 同步源协议。尤其 **V63 的单执行限定与 Agent 最大安全并行是明文冲突**，此文件不是静默替换 V63。Product/DAG 未授权改变前，当前获准的阶段任务仍按原 Claim 和旧范围执行。

## 3. Domain Package 软件工程模型：让不同 LLM Agents 真正开发

**第一阶段优先一个合法精选的 `Engineering Business B`**，在内部明确以下 **业务 bounded contexts**（不是七个物理 npm Package，也不是七个内核）：

| Context（B） | Agent 可完成的动作与用户结果 | 可复用依赖 |
|---|---|---|
| Product / Requirements | 分析已有仓库、澄清真实歧义、生成可追踪 Requirement/PRD、建议 Product 范围、负责候选评审/修复 | selected S Context/Decision；GitHub C；#63、#92 |
| Architecture / Planning | 架构方案、技术栈和 Scaffold 适配、**阶段×子需求×并行 Lane** Task DAG、实际能力/预算/写集规划、任务和测试契约 | #91、#141、ADS v4.18 |
| Agent Registry & Role Dispatch | Agent Provider/Model/Host 能力、逻辑身份、角色独立性、任务委派、预算、Attempt/handoff 和 source/Claim pin | #64、#125；已有 K/C |
| Version Journey | 用户目标驱动下候选下一步、合法自动派发、Agent→Agent Review/Repair、仅必要 Human Inbox、重启/未知 Effect 恢复 | #145、#137 |
| Assurance / Release | 找用户验收缺口、选中真实外部消费者/Host 见证、Review Finding、风险比例化修复/复测、RQ/Tag 真证据 | #146、#92；ADS CI/Host |
| Experience / Improvement | 区分数据/流程/SDK/业务缺陷；基于真 Casebook 提出 context、Agent、DAG、测试或新版本改进候选 | #139–#144、#57/#58/#60；task-meta 仅经 GitHub |

**K/S/B/C 真实所有权**：B 的模型建议不授权；唯一选定 DH K 管状态、Effect/Admission/CAS/UNKNOWN；S 提供 Rule→Exact Reuse→Promoted Procedure→Bounded LLM（历史设计需真实 ABI 资格）；C 实际连接 GitHub/多 Coding Agent CLI/CI/Host/可选 runx。各 Agent 不能借其子进程冒用 GitHub Publisher、修改受保护文件或读取 Hidden 机密。GitHub Task 原生事实不放 SQLite 的第二 authoritative Task DB。C task-meta 直连和独立 Agent SDK/Kernel 都不在范围。

## 4. 最小安全并行重构 Workfront（计划候选，**不是已准入的 native DAG**）

```text
                       [Product #77 P1 parity + D01–D09 intake/review]
                                 │           [ADS #85 exact adoption]
             [L2 #82 prior semantic proof + bounded Product delta] ─┘
                                 └──→ Qualified Product/L2/ADS Join
                                            │
                ┌───────────────────────────┼─────────────────────────┐
                ▼                           ▼                         ▼
    [Engineering B Product/Plan] [Agent Registry/C adapters] [UX J01–J08 draft/shell]
    [#91; #141; #143 fixtures]     [#64; eligible C/Host]        [#92; #145 read only]
                │                           │                         │
                └──────────────┬────────────┘                         │
                               ▼                                      │
       [native GitHub Claim / one K legal Effect / REAL Agent]
       [#125: #127 merged + PR #129,#131,#133 review/merge on lawful pins]
                               │                                      │
                               ├──────────────┐                       │
                               ▼              ▼                       │
                      [Builder lane A]  [Builder lane B] <─────────────┘
                      (disjoint source/Claim; physical concurrency only if new P0 approved)
                               └──────┬───────┘
                                      ▼
             [independent Agent Reviewer → scoped Repair → source re-review]
                                      ▼
                    [Host/CI + actual installed user acceptance]
                                      ▼
                    [Version RQ / permitted merge+tag / next version]
```

- **可以并行的设计/只读工作不因后续 Gate 停住**：Product 修订、B 纯协议建议、Agent Provider 能力核对、UI 线框/文案、Casebook Goldens 都可独立 prepared；**受保护 source/Effect** 仍需要法定 Product/L2/DAG/Claim/写集和真实 SDK 资格。
- 已在途 PR #129/#131/#133/#135 继续沿原 Workfront 审核，不因 #147 文档改变原 exact SHA 结论，不“抢占”同一根目录或同时写共享 exporter/manifest。
- 应首轮复用已有 GitHub C/Host/Engineering B，不为接一名 Agent 建一个通用多执行体 OS 服务，也不重造 Provider。
- 推荐 5 个**真实交付里程碑**而非为每个 UI 状态新建 Issue：`M1 产品需求/L2/Agent-role 契约源裁定`; `M2 一个真实 Agent + native Claim selected K`; `M3 两个独立安全 Builder workfront + independent Agent Reviewer`; `M4 full user journey UI/Host/Release`; `M5 下一版和数据驱动评估`。其中 M1 的纯文档准备与 M2 有现行批准的独立工作可以并行，不是完全串行。

## 5. 核心用户验收：必须 Agent 自主，人工只例外

**CJ-APP-01**：用户创建/导入真实软件仓库，输入自然语言产品目标与能力约束；无需自行理解 ADS 或多次点“继续”，Agent 分析→Product/L2/DAG 候选；真正目标模糊时简短询问人；本地 Draft 不会自报冻结。

**CJ-APP-02**：当前有效 ADS/权限、Product/L2、真实 native Issue DAG/Claim 与一个 selected K C/Host readback 齐备后，真实 Codex CLI Agent 开发一个 Task，源 commit+PR 及独立 Review；task-meta 断开不导致 Task 阻塞；丢 ACK UNKNOWN 可恢复。

**CJ-APP-03（完整 Agent-first P0 建议）**：两项真正独立子需求，各有合法 Claim、不同 worktree/write set，由能力证据合格的两个 Agent 同时执行；Reviewer Agent（不同上下文）发现一个真正缺陷，原合法 Builder 执行有界修复，后续 independent reviewer 确认；运行真实 required CI/Host/installed consumer，不凭模型自述 PASS。若 Product Owner 保持原 V63 max1，则明确本 CJ 不属旧 P0、整个 v2 不能宣称达到本次 Agent-first 最大安全并行目标。

**CJ-APP-04**：需要特殊权限/真实设备/冻结 Product 方向/应由人的 Release 批准时，只出现必要 Human Intervention 卡片：`why human, alternatives, delegated authority, source, minimum local steps, expected evidence, resume lane`。正常已授权 Review/修复/CI 不打断人。虚假 Human Required = Negative FAIL。

**CJ-APP-05**：实际 GitHub CI、被选执行路径、外部安装/使用和版本 RQ/Tag 的真实结果完成；新版本从历史行为/安全不变量与实际 Casebook 继续，历史 PASS/Release 不倒写，也不自动升级封闭 Domain Package。

**成本与人为最少干预**：只在 task-meta 经 ADS GitHub 合格发布且覆盖度足够时统计 Agent 活跃耗时、模型成本、Agent→Agent handoff、真正 Human 等待与不必要点击；不能将 Issue 创建到关闭视为 Agent 工作量。追求质量合格情况下更少无效审批与更快交付，而非无条件减少所有 Review。

## 6. 明确结束标准和近期待办

**本研究当前结论：DESIGN_FOUNDATION=READY_FOR_PRODUCT_INTAKE；PRODUCT_FREEZE_READY=NO；L2_FREEZE_READY=NO；NATIVE_DAG_FOR_NEW_SCOPE=NO；TALLY_AGENT_FIRST_END_TO_END=NOT_RUN；RELEASE_READY=NO。**

接下来的操作按顺序但不是强制全串行：
1. **Product**：在原 PR #77 后继合法作者修订中正式处理 D01–D09，特别 V63 与 Agent-first 冲突；记录 #77 现存 31 retained P0 / parity P1 的 Owner 策略。需要独立 exact HEAD Review，不能因这份文章直接宣称解决。
2. **L2**：以 #82 既有 17 P0/67 Vector/N01–N30 语义作为审查基线，增量定义 `AgentWorkPlan/RoleAndCapabilityMatch/TaskOrchestrationDecision/ReviewerSeparationProof/HumanInterventionRequest` 的 B Intent 和真正 Host/ABI，避免五个新物理 B；需要合法变更审查。
3. **Workfront**：由 #86 Controller 对已有 source/Issue/PR 与新需求去重、source writes/concern conflict、genuine GitHub native blocked_by 做授权 DAG 变更；仅未覆盖的独立 concern 才提新实现 Issue。
4. **源实现**：先稳定 #125 native Claim + selected B/K/C real Agent Loop，再把 #92/#145 UX、独立 Agent Reviewer/Repair、#146 installed acceptance 和多 Agent 并行接入。
5. **验收和 Release**：必须实现 CJ-APP-01–05 的最终经裁定向量及 real RQ；对缺乏真实凭据/设备场景给出最小 Host/human 触发步骤，不能伪称已跑或发布。

**对当前 v2 的直接答复：**这些文档**能够成为正式重构基础**，但不能直接替代最新有效 Product/L2、不能一夜之间让未实现的 Agent 软件工厂变成已交付。最合理是**现有 v2 续建 + P0 用户需求增量裁定 + 平行工程切片 + 两个版本真实用户验收**，而不是重新开发 DomainHarness、ADS 或 Tally 的第二 Kernel。
