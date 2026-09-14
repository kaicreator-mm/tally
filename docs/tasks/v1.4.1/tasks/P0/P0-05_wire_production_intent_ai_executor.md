# P0-05 — Wire production Intent AI executor

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W04
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** A0-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Replace nil/test-only proposal executor injection with the real configured AI Runtime execution path.


## Source facts

- `SetV14ProposeExecutor` exists; production composition wiring is incomplete.


## Allowed / primary paths

- `cmd/server/**`
- `internal/web/**`
- `internal/airuntime/**`
- `internal/intake/**`


## Forbidden boundary

- Do not add direct provider SDK if project public AI Runtime integration already owns this boundary.


## Implementation requirements

1. Use existing AI Runtime client/transport seam.
2. Typed unavailable/config errors.
3. No provider routing logic in handler.


## Acceptance criteria

- [ ] Server startup establishes executor when configured; unavailable runtime is explicit and non-panicking.


## Verification

- Composition test with fake transport; connected integration where available.

## Mandatory implementation discipline

- Read `../../DECISION_LOCKS.md`, `../../PRD_AMENDMENTS.md`, and `../../SOURCE_BASELINE.md` first.
- Do not broaden scope to “clean up” adjacent architecture.
- Do not weaken fail-safe behavior to make tests pass.
- Preserve source-commit identity and held-out verification boundaries.
- If an expected seam does not exist, stop this task as `BLOCKED_BY_SOURCE_FACT` and report the exact source evidence instead of inventing a parallel subsystem.
- New behavior needs focused tests.
- Existing baseline failures must be distinguished from introduced failures.

## Completion report

Report exactly:

1. status: `DONE | BLOCKED | PARTIAL`;
2. changed files;
3. semantic changes;
4. tests/commands executed with exact results;
5. baseline failures observed;
6. remaining risks;
7. whether any Decision Lock was touched;
8. commit hash (if committed).

---

# L3 Task-Level Implementation Reference Pack

## Model assignment

- **Execution:** `HIGH`
- **Review:** `HIGH`
- **Reason:** Crosses AI Runtime composition and production server wiring; duplicate-provider logic is a silent architecture regression.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `internal/airuntime/executor_test.go`, `request_test.go`, `test/arch/ai_runtime_boundary_test.go`, `ai_runtime_domain_migration_test.go`
2. **Contract / Interface:** Interface: `airuntime.Executor.Execute`, `airuntime.Validated`, `intake.Propose`
3. **Core Implementation:** Core: `internal/airuntime/*`, `internal/web/handlers/v14_launch.go::SetV14ProposeExecutor`, `cmd/server/main.go`
4. **Failure Handling:** Failure: missing/unavailable executor is explicit; handler never adds model/provider routing or legacy `internal/llm` path

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Keep presentation, intent evidence and frozen engineering artifacts explicitly separated.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not let LLM-generated UI/risk text become authority without deterministic validation/freeze.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## High-model execution requirement

This task should be performed by a high-capability model because success requires semantic judgment rather than mechanical editing. The model must explain the chosen seam, enumerate rejected alternatives, and demonstrate that fail-safe behavior remains intact before coding.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.

---

## Execution status

- **Status: DONE** (2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P0-05_wire_production_intent_ai_executor.md`
- Implemented: 生产组合根（`cmd/server`）启动时按环境配置把 v1.4 意图入口的推理出口接到真实 AI Runtime 执行路径。已配置（`TALLY_LLM_API_KEY`/`OPENAI_API_KEY` 凭证，或显式 `TALLY_LLM_BASE_URL`）→ `aiProposeExecutorFromEnv()` 组装 `airuntime.Validated(过渡 llmExecutor)`（与 cmd/worker、cmd/tally 同型适配器同型：遗留 `internal/llm` OpenAI 兼容客户端搬运，ADR-0002 SDK 接线缺口；真实 SDK 接线后三处一起删除）并经既有注入缝 `handlers.SetV14ProposeExecutor` 注入；未配置 → 类型化 `errAIRuntimeNotConfigured` + 启动 Warn，服务照常启动、意图入口保持既有「未接入」显式渲染（不假装可用、不 panic，与 `TALLY_WEBHOOK_TOKEN` 同一装配纪律）。provider/model/端点/凭证只出现在组合根适配文件，handler 与域包零改动（实现要求 1/2/3 全落实）；`internal/web/handlers/server.go` 及全部共享热点零接触。
- Acceptance: 「Server startup establishes executor when configured; unavailable runtime is explicit and non-panicking」 — PASS（`TestAIProposeExecutorWiredEndToEnd`：与 main() 完全相同的生产装配 + 真实 v14 路由 + 假 OpenAI 端点，接入后页面不再渲染未接入防线、POST 意图产出可确认草案；`TestUnconfiguredRuntimeKeepsLaunchExplicitlyUnavailable`：未配置 → 200 + 显式不可用文案，非 5xx/panic；`TestAIProposeExecutorNotConfiguredIsTypedAndNil`：类型化哨兵 + nil）。
- Validation: `go test -run "AIPropose|Unconfigured" -v ./cmd/server/` PASS（7 例：配置判定 3 子例、组合链、传输失败显性化、未配置负路径）；`TestAIProposeExecutorConnectedIntegration` PASS（真实 DeepSeek 端点，凭证仅从 `DEEPSEEK_API_KEY` 环境变量读取；无 Key SKIP）；`go test ./test/arch/` PASS；`go test -count=1 ./internal/web/... ./internal/airuntime/ ./internal/intake/ ./cmd/server/` PASS；`go test ./internal/...` PASS（45 包）；`go test -tags=integration ./...` 58 包 ok，唯一失败 `test/harvestenv` 为基线既有（已在 HEAD `64854e3` 无本任务改动树复现证实）；`go build ./...` + `go vet ./cmd/server/` PASS；gofmt clean；`git diff --check` clean。`make check` NOT_RUN(reason: Git Bash 无 make 可执行文件；其组成的 arch/internal/integration 三条等价命令已逐条真实执行)。
- Baseline failures: `TestHarvestEndToEndIntegration`（`environment_run` 关系不存在）—— 基线既有，非本任务引入（stash 对照证明）。
- Decision Locks touched: none（DL-02/DL-16/DL-17/DL-20、A-10、AGENTS §6 职责边界均遵守；锁文件未动）。
- Risks: 过渡适配器三处并存待 P0-10/P0-11 统一替换；真实模型偶发 `recommended` 字符串类型漂移被 P0-03 冻结契约 fail-safe 拒绝（正确行为，是否加有界归一归 P0-03 契约方裁定）；过渡传输无 requested/actual model 观测，溯源不伪造（§13）。
