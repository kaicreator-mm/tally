# Tally v1.5.5 源码级功能审计与 DomainHarness v2 重构输入

**审计日期**：2026-10-24  
**仓库**：`kaicreator-mm/tally`  
**审计固定对象**：Release tag `v1.5.5` → commit `b5e7b6e38ea38837e3bbcfc46c36586180705de0`  
**原 v1.5.5 发布候选**：`9c71fa6f35f08bcc5bda4101263929f70f81202a`  
**方法**：GitHub 递归 Git tree、核心模块生产源码/API/关键测试只读检查、发布记录交叉对账。没有本地编译或重新运行测试。**本报告是完整功能域盘点与关键源码路径的静态审计，不宣称对 406 个生产 Go 文件逐行进行漏洞审计。**

## A. 基线与覆盖

- Git tree 文件数：1,570 个 Blob；Go 735 文件，生产 Go 406，Go 测试 329；`internal/` 下一级模块 57 个。
- `v1.5.5` Final RQ 已记录 PASS：Build、full regression、集成（76 packages）、51/51 Hidden、8/8 Critical Journeys、独立审查。证据：[Issue #26](https://github.com/kaicreator-mm/tally/issues/26#issuecomment-5692237137)。这不是本次重新运行的结果。
- `README.md` 和 `docs/STATUS.md` 仍保留 v1.5.0 的历史状态说明，不能作为 v1.5.5 最终功能事实的唯一依据。
- 来源优先级：固定 commit 的具体源码 → 测试 → 固定版本发布记录 → 历史 PRD/Issue。未找到生产调用证据时标为“源码存在，端到端连通性未在本次复验”。

## B. 关键产品决策（v2 必须满足）

1. `task-meta` 是部署/采集体系强依赖，但**执行路径上的故障可隔离**：采集、上报、发布失败均不得阻止新 Task 启动或已有 Task 完成；依赖正常使用时提供 Agent/Task/Run/Attempt/Step/Tool/Model/Cost 等测量。缺失遥测保持 UNKNOWN，不计作零。ADS GitHub Report 是 Tally 的数据读取交界，正常不直读 task-meta 私有库。
2. Local Execution 为基本运行路径；SSH/Docker、runx、远程 Provider 作为可选扩展。缺 runx 的场景必须完整通过单任务和版本闭环。
3. GitHub 是 Issue/PR/Review/CI/报告的事实源；SQLite 用于读侧缓存、搜索索引、同步水位、草稿/outbox 等，以及经 DomainHarness K 管理的 Tally 应用状态，二者不能混淆。缓存写入不等于 GitHub mutation 成功。
4. Tally UI 必须实现简化但完整的 ADS 工作流：Product Evidence→PRD→独立 Product Review/Freeze→L2→独立 L2 Review/Freeze→Task DAG→Task Issue/PR→Execute→Repair/Review/Verify→Version Closure→RQ→Tag/Baseline。
5. Source Compatibility：保留旧版业务语义，不要求 Go/Node 互操作、不迁移旧 Go 源码。DomainHarness K/S 重用优先；工程业务逻辑归 Tally Engineering B，Provider 归 C。

## C. 功能盘点与目标所有权

说明：`CONFIRMED` = 固定 tag 中有相关生产实现和相应测试文件；不自动等于 v2 可运行或本次全链路 PASS。`LEGACY` = 旧代功能仍存在源码，v2 是否保留由业务决定。`ADAPTER` = 应被能力适配取代，不移植原 Go 实现。`GAP` = v2 新要求、现有源码不足以证明具备。

| ID | 既有功能 | 源码定位（tag 相对路径） | 基线 | v2 归属/处置 |
|---|---|---|---|---|
| F01 | Requirements Baseline / 验收条件 | `internal/engineering/model.go` | CONFIRMED | B KEEP |
| F02 | Requirement 澄清/访谈/签署/缺陷 | `internal/requirement/`, `internal/intake/` | LEGACY | B+S ENHANCE，复用意图澄清 |
| F03 | Frozen Intent 与风险信号 | `internal/intentevidence/`, `internal/intakefreeze/` | LEGACY | B+S KEEP SEMANTICS |
| F04 | PlanningRevision / ProjectPlan | `internal/engineering/model.go` | CONFIRMED | B KEEP |
| F05 | ArchitectureRevision / Constraints | `internal/engineering/model.go` | CONFIRMED | B KEEP |
| F06 | TaskDAG Proposal→Active DAG | `internal/engineering/dag.go` | CONFIRMED | B KEEP |
| F07 | 环检测/依赖 Ready/并行候选 | `internal/engineering/dag.go` | CONFIRMED | B KEEP |
| F08 | CompletionContract / 缺失门禁 | `internal/engineering/completion.go` | CONFIRMED | B+S Rule KEEP |
| F09 | Authority / AUTO/SUPERVISED/REQUIRED | `internal/engineering/completion.go`, `internal/approval/` | CONFIRMED | B+S KEEP |
| F10 | 证据来源与独立性 | `internal/evidence/`, `internal/verification/` | CONFIRMED | B+S Evidence KEEP |
| F11 | Risk/Assurance 选择 | `internal/risk/`, `internal/assurance/` | LEGACY | B+S KEEP |
| F12 | Contract 编译/策略匹配 | `internal/contract/`, `internal/engineeringpolicy/` | LEGACY | B+S Rule KEEP |
| F13 | Project Control Plane / 状态推进 | `internal/controlplane/service.go`, `types.go` | CONFIRMED | B + K REIMPLEMENT |
| F14 | API 操作 Lifecycle/DAG/Dispatch | `internal/controlplane/http.go` | CONFIRMED | App API KEEP |
| F15 | Ready 调度/并发/启动恢复 | `internal/controlplane/scheduler.go` | CONFIRMED | B+K KEEP |
| F16 | AgentExecutor 与 Runtime 中立协议 | `internal/taskexecution/`, `internal/agentruntime/` | CONFIRMED | B+C KEEP |
| F17 | Codex / DSH / Fake Agent | `runtimes/codex/`, `runtimes/dsh/`, `runtimes/fake/` | CONFIRMED | C Provider ADAPTER |
| F18 | Direct LLM / AI Runtime | `internal/intelligence/`, `internal/airuntime/`, `internal/llm/` | CONFIRMED | S+C ADAPTER |
| F19 | Local/SSH/Docker/Workspace 能力 | `internal/runner/`, `internal/taskexecution/`, `internal/workspace/` | CONFIRMED | C Execution / Local mandatory |
| F20 | 环境 UP/Down/Health/Deploy/Rollback | `internal/environment/`, `internal/deploy/` | LEGACY | C optional/业务视图视需要保留 |
| F21 | Disposable worktree 与候选冻结 | `internal/changeset/`, `internal/workspace/` | CONFIRMED | B+C KEEP |
| F22 | Run / ChangeSet / Evidence 持久事实 | `internal/runstore/`, `internal/store/` | CONFIRMED | K State + B provenance |
| F23 | Verification Contract / required gates | `internal/verification/` | CONFIRMED | B+S+C KEEP |
| F24 | Hidden/Held-out 执行与隔离 | `internal/holdout/`, `internal/candidateverify/` | CONFIRMED | B+C validation KEEP |
| F25 | Closedloop Execute/Verify/Repair/Accept | `internal/closedloop/` | CONFIRMED | B+S Workflow，K effects |
| F26 | Reviewer 归账与保护文件处理 | `internal/review/` | LEGACY | B+S KEEP |
| F27 | Multiagent role / 独立性 / Repair | `internal/multiagent/`, `internal/controlplane/multiagent.go` | CONFIRMED | B+S KEEP |
| F28 | Git Engineering Authority / GitHub Actions | `internal/gitengineering/`, `internal/engineering/git.go` | CONFIRMED | B+C GitHub KEEP |
| F29 | Local Git 写入/集成与防冲突 | `internal/gitwriter/`, `internal/gitintegration/` | LEGACY | C Git ADAPTER，保留安全约束 |
| F30 | Replan / stale parent / 循环预算 | `internal/replan/`, `internal/controlplane/replan.go` | CONFIRMED | B+S KEEP |
| F31 | Engineering Learning→RuleProposal | `internal/engineeringlearning/` | CONFIRMED | B+S KEEP |
| F32 | Rule 提议/激活/冲突处理 | `internal/policygovernance/` | CONFIRMED | B+S，生产选择经新 BUILD |
| F33 | Context Pack / 经验候选筛选 | `internal/contextpack/` | LEGACY | B+S Context KEEP |
| F34 | 失败分析/修复证据闸门 | `internal/failure/` | LEGACY | B+S Decision KEEP |
| F35 | 日志抽取/脱敏/指纹/归类 | `internal/logpipe/` | LEGACY | S utility / C logs REUSE |
| F36 | 已验证知识 Lookup/Stale/Retire | `internal/knowledge/` | LEGACY | S Memory + B selection |
| F37 | Harvest / Dependency 变更 | `internal/harvest/`, `internal/deps/`, `internal/ingest/` | LEGACY | optional B/C, evaluate ROI |
| F38 | Outcome Event 与再工作指标 | `internal/outcome/`, `internal/measurement/v14_metrics.go` | LEGACY | B+S 投影；观测归 task-meta |
| F39 | 旧版 LLM/Token/Time Measurement | `internal/measurement/` | LEGACY | **由 task-meta 取代采集权威**；缺失不能阻断执行 |
| F40 | SQLite / Postgres 存储接口 | `internal/persistence/` | CONFIRMED | K Host 本地持久化 + SQLite GitHub cache 分离 |
| F41 | Git-backed Project/Trace/Index | `internal/project/` | LEGACY | B projection + C GitHub |
| F42 | Project/Requirement/Document/Wiki UI | `internal/web/handlers/projects.go`, `requirements.go`, `docs.go`, `wiki.go` | CONFIRMED | App UI merge/unify |
| F43 | v14 Launch/Intent/Run/Review/Attention UI | `internal/web/handlers/v14_*.go` | CONFIRMED | App UI reuse interaction concepts |
| F44 | v15 Control Plane 生命周期 UI | `internal/controlplane/ui.go` | CONFIRMED (thin) | **ENHANCE P0** |
| F45 | 安全 Auth/CSRF | `internal/web/auth.go`, `csrf.go`, `router.go` | CONFIRMED | App transport/security |
| F46 | 操作命令与对外 CLI | `cmd/tally/`, `cmd/server/`, `cmd/v15server/`, `cmd/worker/` | CONFIRMED | Node TS CLI/Server rewrite，功能选型 |
| F47 | Release/CI/Hidden/回归质量门禁 | `scripts/validate-v1.5.*.sh`, `test/`, release docs | CONFIRMED | 版本 Closure B + C verifier |

### 旧版代码连通性的特别说明

- F14：`internal/controlplane/http.go` 已注册 `POST requirements/plan/architecture/task-dag/replan/dispatch/multi-agent`，但 `internal/controlplane/ui.go` 只为状态、Authority、Restart、手写 JSON Dispatch 提供直观控件。**API 存在不能推断完整交互 UI 已实现**。
- F39：旧版 `measurement.Recorder.RecordRuntimeUsage` 在 store 为 nil 或写入失败时返回错误。这是旧实现语义，不得照搬成为 v2 Task 强耦合路径；应使用非阻断的 task-meta 采集接口/sidecar，真实 Task completion 不以 telemetry ACK 为条件。
- F38：`internal/outcome/capture.go` 自身注明 `reverted/hotfixed/regression_found/human_correction` 等存在显式 Capture API，但当时缺少可靠的自动事实生产钩子；不要将“有 API”误报为“自动采集闭环”。
- F28：`internal/gitengineering/github.go` 有主要写操作，但从该类公开函数不能推出 v2 已具备 GitHub ADS Report 的全面 source-qualified 读回、分页、修正、版本兼容、缓存失效等能力。
- F40：旧版 SQLite 是业务本地持久化适配，并不是 GitHub source-qualified 只读缓存。v2 新增缓存/索引须另定一致性、TTL/ETag/水位和 mutation 前复核。
- F43：旧 v14 Launch 表单含自然语言 Intent → 澄清 → Draft Confirm → Run，但 Run 入队旧 `jobs`；不可直接视为完整新 Tally DomainHarness 工作流。

## D. v2 目标 Package 边界（按真实功能重组，非一对一翻译）

```text
Tally Engineering Domain App (TS monorepo)
  ├─ Engineering B / Product & Lifecycle
  │    Requirements, PRD, L2, Architecture, Gate, Revision
  ├─ Engineering B / Task & Scheduling
  │    DAG, TaskContract, Ready, Agent matching, Dispatch, Replan
  ├─ Engineering B / Verification & Release
  │    Candidate, Review, Repair, Integration, RQ, Baseline
  ├─ Engineering B / Experience & Decision
  │    ContextPack, Learning, Candidate qualification, Budget
  ├─ selected S Context / Rule / Decision / Workflow / Evidence
  ├─ exactly one DomainHarness K Primary + selected Host
  ├─ C GitHub (native facts + ADS Reports + legal writeback)
  ├─ C LocalExecution + C Agent adapters (required)
  ├─ C SSH/Docker/runx execution adapters (optional)
  └─ task-meta telemetry publisher/agent hooks (strong install dependency,
       non-blocking data plane; GitHub-only aggregate consumer)
```

严禁新增 Tally 自己的第二套 K Journal/Effect/Replay；禁止另造通用 Agent Runtime 平台或第二套 GitHub Task Authority。B 保留工程语义；S 提供可复用决策/Context 机制；K 执行授权、状态、Effect/恢复；C 负责外部 I/O。

## E. 新版最小 ADS 全流程 UI（P0）

| UI 步骤 | 用户可完成 | 对接的业务机制 | 必须有的门禁 |
|---|---|---|---|
| 1 项目与产品 | 项目新建、意图、证据、需求澄清、生成/编辑 PRD | B Product + S Context/Decision | 来源、需求范围、审查结果 |
| 2 Product Freeze | 比较 PRD 修订、发起独立审核、Freeze | B Gate + C GitHub | 合法 ADS pin / Reviewer / exact SHA |
| 3 L2 Architecture | 生成/编辑 L2、设计权威边界、接受修订 | B Architecture + S Rule | L2 独立 Review/Freeze |
| 4 Task DAG | 交互式 Task 拆解、依赖编辑、阻塞检查、Issues 发布 | B Task + C GitHub | DAG cycle、native GitHub readback |
| 5 执行中心 | Agent/Local Provider 选择、启动、取消、重试、Replan | B Scheduling + K + C Local | Claim/Scope/权限、幂等、UNKNOWN |
| 6 修复与审查 | 错误解释、Repair、Reviewer、Verifier、PR 回读 | B Validation + S Decision | 原始证据、独立性、不得自称 PASS |
| 7 版本闭环 | 整合、回归/Hidden/CJ、RQ、Merge、Tag、Baseline | B Release + C GitHub | PR PASS ≠ Release PASS |
| 8 经验反馈 | 查看 Task telemetry 覆盖率、成本、经验 Context | B Experience + S Context | 缺失明确 UNKNOWN、无假零值 |

**简化交互方案**：采用一个“开发流程向导 + 阶段状态卡 + Markdown/结构化编辑器 + 审核/错误/执行任务面板”即可，不需要通用 BPM、复杂可视化 Workflow DSL 或超大前端。每一步生成 GitHub 合法制品，同时在 SQLite 缓存投影。所有 mutation 通过统一工程命令/ K 准入、source revision 检查；UI 不是另一份 authority。

## F. SQLite 一致性方案（建议作为 L2 P0）

区分：
- **GitHub Source Projection**：Issues/Dependencies/PR/CI/Review/ADS Task Reports 的派生缓存，字段包含 source ID、commit/profile、ETag/updated/currentness、水位、上次刷新、fetch coverage，支持按需增量拉取、事件通知、强制复核。
- **K Domain State**：Project App occurrence/state/journal、合法本地用户草稿、部署/调度状态，仅 K 按协议写。它不是 GitHub 原生 Issue/PR 事实的替身。
- **Mutation Outbox**：带幂等键、GitHub 目标、expected revision、待发/发送中/UNKNOWN/已确认状态；不可仅凭 HTTP ACK 或本地 cache 命中宣告成功，必须 readback/reconcile。
- **失败/断连**：只读界面可展示“缓存，截至 T 时刻”；不能允许陈旧 GitHub 权威数据放行新的不可逆写入；GitHub 无法复核时停在 WAIT/UNKNOWN，Local 计算和已授权非 GitHub 任务执行可依项目策略继续。

## G. task-meta 非阻断降级合同（P0）

### 正常
Agent/Task → task-meta Hook/Collector → ADS 合规 GitHub Task Report → Tally C GitHub reader → context/cost/learning/feedback。

### task-meta down
- 不阻断 `TaskContract → C LocalExecution → Agent → Verification → GitHub`，也不阻断已有运行 Task。
- 采集事件 best-effort 或有界缓冲；超时/背压与执行结果分离；断连使用健康指示 `TELEMETRY_UNAVAILABLE`，采集覆盖率为 UNKNOWN/partial，不当零值。
- 允许恢复后合法补发、有界去重、按 source revision 回读；不能凭补发遥测改写已经合法完成的工程事实。
- 经验比较、Agent 成本排名、Package 候选资格在证据不足时停为 `INSUFFICIENT_EVIDENCE`；实际工程 Gate 以本身真实 verifier/GitHub/Host 为准。
- 运行契约必须故障注入证明：collector 进程退出、hook timeout、队列满、GitHub 报告迟到/重复均不改变 Task 执行最终结果。

## H. 高 ROI Scope 选择

**纳入本版 P0**：
1. Full ADS Process UI 的简化向导、修复/Review/RQ 操作。
2. 现有功能等价迁移、Local-only 真 Task E2E。
3. task-meta graceful degradation 和 GitHub 合法报告回读。
4. SQLite source-qualified cache + outbox/reconcile。
5. Agent/Execution capability registry，Local 默认 / runx 可选。
6. 错误诊断 → 下一动作建议（复用现有 failure/logpipe/replan）。

**纳入本版 P1，低成本逐步开启**：
7. 可信 Experience Context Pack 与建议使用反馈。
8. 按有覆盖率的实际数据展示 Token/Cost/Agent performance。
9. 跨项目并发资源与阻塞视图。

**后续版本**：自动 Package 生成/升级、递归优化、多轮 A/B/评测平台、高可用多集群、通用 BPM。

## I. 闭环与自洽 Gate

| Gate | 可执行证明 | 当前重构状态 |
|---|---|---|
| CL-01 Product→PRD→L2→DAG | UI 操作产生合法版本、Review 和 GitHub 事实 | V2 NOT_IMPLEMENTED |
| CL-02 Local-only Task | runx 未配置，真实 Agent/Worktree/Verifier/GitHub 完成 | V2 NOT_RUN |
| CL-03 task-meta down | Collector 故障时 Task success/failure 与无故障路径一致；Telemetry UNKNOWN | V2 NOT_RUN |
| CL-04 GitHub cache consistency | stale cache 与远程 correction/merge/403/429 冲突不假放行 | V2 NOT_RUN |
| CL-05 Repair & independent review | FAIL→Repair→Review→Verify，有独立 exact-SHA 证据 | V2 NOT_RUN |
| CL-06 Version Closure | 多 Task 集成、回归、Hidden/CJ、Release Qualification、Tag | V2 NOT_RUN |
| CL-07 Learning loop | ADS 报告读回→下一 Task 合法 Context→真实结果反馈，缺测不伪造 | V2 NOT_RUN |
| CL-08 Optional runtime | runx 接入或移除均不改变 Tally 的核心工程语义 | V2 NOT_RUN |

**产品闭环结论**：旧版 v1.5.5 具有工程 Task 层的大部分核心行为和发布验证；v2 的完整 ADS 开发 UI、GitHub 双向数据一致性、非阻断 task-meta 采集和 Local-only DomainHarness 真 E2E 尚未证明。V2 PRD 可启动，但不得声称架构闭环已通过。

## J. 待办与证据限制

- 本次已基于固定完整文件树盘点 57 个 internal 模块，逐一检查了主干路径的代表性源码、API、实现和测试，并交叉核对固定版发布账本。未逐文件执行编译、静态调用图/coverage 或重新运行 735 个 Go 文件的全量验证；`LEGACY` 模块生产调用可达性需在详细迁移阶段补证。
- 下一阶段生成逐功能的 `functional_parity.csv`（旧 file/function/test/behavior → 目标 B/S/K/C → E2E case），必要时在本地对 tag 执行 `go list`, `go test`, call graph 与启动拓扑验证，明确 DEAD/KEEP/REPLACE，不把 source-present 等同 production-used。
- 先冻结 v2 Product 以上六项 P0，再将 GitHub source-qualified cache 和 K State 的具体存储协议及公共 DomainHarness ABI 交给独立 L2 审查；不得为 Tally 单独重写 K。

## K. 精确源码索引

- [v1.5.5 Tag](https://github.com/kaicreator-mm/tally/tree/b5e7b6e38ea38837e3bbcfc46c36586180705de0)
- [Engineering DAG](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/engineering/dag.go)
- [Completion](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/engineering/completion.go)
- [Control Plane HTTP](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/controlplane/http.go)
- [Control Plane UI](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/controlplane/ui.go)
- [Multi-Agent](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/multiagent/engine.go)
- [Replan](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/replan/validate.go)
- [GitHub Provider](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/gitengineering/github.go)
- [SQLite](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/persistence/sqlite/store.go)
- [Legacy v14 Launch](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/web/handlers/v14_launch.go)
- [Legacy telemetry](https://github.com/kaicreator-mm/tally/blob/b5e7b6e38ea38837e3bbcfc46c36586180705de0/internal/measurement/recorder.go)
- [Release Qualification](https://github.com/kaicreator-mm/tally/issues/26#issuecomment-5692237137)