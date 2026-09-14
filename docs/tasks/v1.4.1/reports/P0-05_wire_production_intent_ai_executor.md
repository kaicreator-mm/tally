# P0-05 — Wire production Intent AI executor: Execution Report

- **Date:** 2026-09-12
- **Status:** DONE
- **Baseline:** working tree at `64854e3` (P0-04 merged); DECISION_LOCKS frozen at `c0f8498`

## 1. Deliverable

生产组合根（`cmd/server`）现在在启动时把 v1.4 意图入口的推理出口接到真实 AI
Runtime 执行路径：已配置 → 组装 `airuntime.Executor` 并经既有注入缝
`handlers.SetV14ProposeExecutor` 注入生产 handler 链；未配置 → 类型化
`errAIRuntimeNotConfigured` + 显式告警，服务照常启动，意图入口保持既有
「AI Runtime 未接入」显式状态。消除了 SOURCE_BASELINE.md「Known production
gaps」第 3 条（production Intent AI executor injection is incomplete）。

### 选择的缝（pack 要求：解释缝的选择）

组装表达式与 cmd/worker、cmd/tally 组合根完全同型：

```text
aiProposeExecutorFromEnv()                     // 配置判定 + 组装(cmd/server 组合根)
  = airuntime.Validated( llmExecutor{          // 域侧机械闸(既有 seam,airuntime Validated 契约)
      client: llm.NewHTTPClient(llm.ConfigFromEnv()) })   // 遗留 OpenAI 兼容客户端(过渡传输,ADR-0002)
→ handlers.SetV14ProposeExecutor(exec)         // 既有注入缝(v14_launch.go,P3-07 合同)
→ V14ProposeIntent → intake.Propose            // 既有生产 handler 链(P0-03/P0-04 已就位)
```

- 「existing AI Runtime client/transport seam」（实现要求 1）即
  ADR-0002 记录的 SDK 接线缺口过渡适配器：`internal/airuntime.Executor`
  边界 + `airuntime.Validated` 域闸 + 遗留 `internal/llm` 客户端搬运。
  真实 SDK 接线（P0-10/P0-11）落地后本适配器随 worker/tally 的同型适配器
  一起删除。

### 否决的替代方案

1. **在 `internal/airuntime` 内建真实 HTTP/SDK 传输** —— 违反包自身导入边界
   （`test/arch/ai_runtime_boundary_test.go` AC4：airuntime 禁 net/http/外部
   SDK），也是任务 Forbidden boundary 明令禁止项。拒绝。
2. **抽 internal/** 共享适配包** —— 违反
   `ai_runtime_domain_migration_test.go` AC1（internal/** 禁导入 internal/llm，
   仅 cmd/** 组合根豁免且必须与 airuntime 同现）。拒绝。
3. **顺手动 cmd/worker / cmd/tally 去重** —— 越出本任务 Allowed Paths；
   `PATH_OWNERSHIP.md` 判 cmd/worker 运行时组合归 P4-01 独占。拒绝；三处
   过渡适配器并存至 SDK 接线统一删除（在 aiexecutor.go 包注释中声明）。
4. **在 handler 里读环境/做 provider 配置判断** —— 实现要求 3（handler 无
   provider 路由逻辑）禁止；配置判定收敛在组合根。拒绝。

### 配置语义（AC「establishes executor when configured」）

- **已配置** = 有效凭证（`TALLY_LLM_API_KEY`，回退 `OPENAI_API_KEY`）或显式
  端点覆盖（`TALLY_LLM_BASE_URL`，自建网关/本地推理端点可不设 key）二者其一。
  `TALLY_LLM_MODEL` 有缺省值，不作为可用性信号。
- **未配置** = `(nil, errAIRuntimeNotConfigured)`：main() 记 Warn 并继续启动
  （与 `TALLY_WEBHOOK_TOKEN` 同一装配纪律：明确未启用，不做默认放行）；
  页面渲染既有「AI Runtime 推理出口未接入」防线；提交意图得到 200 + 显式
  不可用文案（既有 handler 行为，非 panic、非 5xx、不假装可用）。

## 2. Changed files

- `cmd/server/aiexecutor.go`（新增）：过渡 llmExecutor 适配器 +
  `aiProposeExecutorFromEnv` 配置判定/组装 + 类型化 `errAIRuntimeNotConfigured`
  哨兵；边界声明（legacy llm 仅经本适配器搬运；provider/model/端点只在本
  文件，绝不进 handler）。
- `cmd/server/main.go`（+18 行）：启动时装配并注入推理出口；三路类型化分支
  （已接入 Info / 未配置 Warn / 其他装配错误 Error+exit 1）。
- `cmd/server/aiexecutor_test.go`（新增）：配置判定、组合链（假传输）、
  传输失败显式性、未配置显式不可用、connected integration（环境门控）。

`internal/web/handlers/`（含共享热点 server.go）、`internal/airuntime/`、
`internal/intake/`、`internal/llm/` 零改动 —— 既有接口/错误映射已完备
（读后复用，未发现需要修的缝）。

## 3. Semantic changes

1. server 启动（`cmd/server/main.go`）新增：AI Runtime 已配置时注入意图
   推理出口 → `/v14/launch/intent` 在生产中真实产出草案/澄清（此前恒为
   「未接入」）；未配置时行为与之前一致（显式不可用），新增启动告警说明
   缺哪个配置。
2. 类型化错误面：`errAIRuntimeNotConfigured`（组合根判定未配置）、
   `errNilLLMClient`（装配错误，调用时暴露）——均为 `errors.Is` 判类哨兵；
   凭证只进 `llm.Config`，不落日志、不进 provenance（AGENTS.md §13）。
3. 无 provider/model 路由进入 handler 或域包；`internal/web/handlers` 的
   错误→文案映射、`intake` 的类型化提案语义原样保留。

## 4. Tests / validation（exact commands and results）

新增测试（`cmd/server/aiexecutor_test.go`，package main，与 cmd/worker
main_test.go 同约定）：

- `TestAIProposeExecutorNotConfiguredIsTypedAndNil` — 负路径：全环境清空 →
  `(nil, errAIRuntimeNotConfigured)`，PASS
- `TestAIProposeExecutorConfiguredDetection` — 显式 key / OPENAI_API_KEY 回退 /
  仅 TALLY_LLM_BASE_URL 三例均建立执行器，PASS
- `TestAIProposeExecutorWiredEndToEnd` — 组合测试（假传输）：环境指向
  httptest 假 OpenAI 兼容端点，走与 main() 完全相同的装配
  （aiProposeExecutorFromEnv → SetV14ProposeExecutor → 真实 v14 路由 →
  intake.Propose → Validated → 过渡传输），断言接入后页面不再渲染未接入
  防线、POST 意图产出可确认草案，PASS
- `TestAIProposeExecutorTransportFailureIsExplicit` — 端点 500 → 400 + 类型化
  失败透出（不编造结果），PASS
- `TestUnconfiguredRuntimeKeepsLaunchExplicitlyUnavailable` — 未配置不注入 →
  提交意图 200 + 显式未接入文案（非 5xx、不 panic），PASS
- `TestAIProposeExecutorConnectedIntegration` — connected integration（AC
  「connected integration where available」）：真实 DeepSeek OpenAI 兼容端点
  （`TALLY_LLM_BASE_URL=https://api.deepseek.com`，凭证仅从 `DEEPSEEK_API_KEY`
  环境变量读取，无 Key SKIP），一次有界真实推理（intake 预算：单次、
  ≤4096 tokens、≤$0.05）必须命中类型化提案一等产物（200 草案/澄清）或
  类型化 fail-safe 拒绝（400 + 明确原因），绝不 5xx/panic/未接入，PASS
  （本机环境 Key 在位，真实运行通过）

验证序列（全部实际执行，无未运行标 PASS）：

| 命令 | 结果 |
|---|---|
| `go test ./cmd/server/ -v -run "AIPropose\|Unconfigured"` | PASS（7 例含子测试） |
| `go test ./test/arch/`（两次，含 `-count=1`） | PASS（守卫套件含 ai_runtime_boundary / ai_runtime_domain_migration / v141_decision_lock_guards） |
| `go test -count=1 ./internal/web/... ./internal/airuntime/ ./internal/intake/ ./cmd/server/` | PASS |
| `go test ./internal/...` | PASS（45 包 ok，0 失败） |
| `export DATABASE_URL=…@localhost:15432/tally…; go test -tags=integration ./...` | 58 包 ok；唯一失败 `test/harvestenv` `TestHarvestEndToEndIntegration` 为既有基线失败（见 §5） |
| `go build ./...` / `go vet ./cmd/server/` | PASS |
| `gofmt -l cmd/server/` | 干净（格式化后） |
| `git diff --check` | clean |

`make check` NOT_RUN(reason: Git Bash 环境无 `make` 可执行文件 —— 与 P0-04
报告同一环境限制)；其组成的等价三条命令（arch / internal / integration）
已如上逐条真实执行。

## 5. Baseline failures observed

- `tally/test/harvestenv` `TestHarvestEndToEndIntegration`：报
  `relation "environment_run" does not exist (SQLSTATE 42P01)`。
  **基线既有失败，非本任务引入** —— 已在未含 P0-05 改动的 HEAD `64854e3`
  基线树（`git stash -u` 后）复现同一失败后 `git stash pop` 还原。该测试包
  自身迁移/环境缺口（缺 `environment_run` 关系），与本任务改动
  （cmd/server 组合根接线）无依赖关系；留待其归属任务/整合者处理。

## 6. Remaining risks

1. **过渡适配器三处并存**（cmd/worker、cmd/tally、cmd/server 同型
   llmExecutor）：刻意的 brownfield 决定（arch AC1/AC2 约束 + P4-01 所有权），
   P0-10/P0-11 真实 SDK 接线时必须三处一起替换 —— 已在 aiexecutor.go 包注释
   与各处注释中互相指认，防止单点遗漏。
2. **真实模型 vs 严格形状契约**：connected integration 中观察到 DeepSeek
   偶发把澄清项 `recommended` 输出为字符串（P0-03 冻结契约要求 int），
   `intake` 严格解码按 `ErrInvalidProposal` fail-safe 拒绝整份提案 —— 这是
   契约的正确行为（拒绝而非猜测修正），本任务未放宽；但意味着真实使用中
   部分提案会被拒后需重试。是否为 `recommended` 增加有界归一（如仅接受
   数字字符串）属 P0-03 契约变更决策，需其归属方裁定，本任务不越权。
3. **request/model 溯源**：过渡传输如实留空 OperationID、不上报成本
   （不伪造 trace）；requested model 与实际模型的一致性在过渡传输上无观测
   （AGENTS.md §13 禁止在无观测时写 requested==actual）。真实 SDK 接线后由
   Runtime 侧补齐。

## 7. Decision Locks touched

None. DL-02/DL-16 未触碰（无新 runtime 协议；意图推理不经 AgentRuntime）；
DL-17 遵守（未读 `hidden/`）；DL-20/A-10 遵守（Tally 不实现 planning，只经
airuntime 声明 operation/intent/input/budget）；AGENTS.md §6 职责边界遵守
（generic inference 归 shared AI Runtime 边界，组合根只接传输）。未修改任何
锁文件；`PATH_OWNERSHIP.md` 边界内施工（Allowed Paths:
cmd/server/**；internal/web、internal/airuntime、internal/intake 零改动）。

## 8. Commit

recorded in the task-file addendum / final message.

## 9. L3 completion evidence（pack 要求项）

- **Exact source/tests read:** `internal/airuntime/`（airuntime.go / executor.go
  / executor_test.go / request.go / request_test.go / result.go / fake.go）、
  `internal/intake/`（intake.go / propose.go / clarification_schema.go）、
  `internal/web/handlers/v14_launch.go`（SetV14ProposeExecutor 注入缝、
  V14ProposeIntent、类型化错误映射）、`internal/web/handlers/v14_launch_test.go`
  （注入/还原测试约定）、`internal/web/handlers/server.go:137`（v14 路由生产
  挂载点，只读确认）、`cmd/server/main.go`（组合根）、`cmd/worker/main.go`
  与 `cmd/tally/req.go`（llmExecutor 同型适配器先例）、`internal/llm/llm.go`
  （ConfigFromEnv/WithRole/Propose 契约）、`test/arch/ai_runtime_boundary_test.go`、
  `test/arch/ai_runtime_domain_migration_test.go`、
  `test/arch/v141_decision_lock_guards_test.go`（守卫集）、
  `web/templates/v14_launch.tmpl`（未接入防线/草案渲染断言依据）。
- **External reference actually used:** none —— 本任务以当前 Tally 源为唯一
  实现参照（pack 明示无需外部实现）；connected integration 端点
  （api.deepseek.com OpenAI 兼容路径）仅作运行环境，未引入任何依赖；
  `test/arch/deps_test.go` 白名单未动。
- **Why the chosen seam is the narrowest valid seam:** 生产缺口恰好是组合根
  一侧的「装配缺失」而非接口缺失（SetV14ProposeExecutor、intake.Propose、
  Validated、llmExecutor 先例全部就位）；最小完整变更 = 在唯一组合根按既有
  同型表达式补装配 + 类型化配置判定。internal/** 共享包与 airuntime 内建
  传输分别被 arch 守卫 AC1/AC4 直接拒绝（见 §1 否决项），故 cmd/server 本地
  适配器是唯一合法且最小的落点。
- **Changed files / exact test commands/results:** 见 §2、§4。
- **Negative tests added:** 未配置类型化拒绝 + 显式不可用（2 例）、传输失败
  显性化（1 例）—— 共 3 条新负路径。
- **Known uncertainty / version sensitivity:** connected integration 依赖外部
  DeepSeek 服务可用性与模型行为（无 Key 时 SKIP，不伪装 PASS）；§6.2 的
  recommended 类型漂移为已观察到的真实模型边界。
- **Decision Locks not bypassed:** confirmed（见 §7）。
