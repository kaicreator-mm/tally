# P1-04 — Wire worker to compiled authority — Execution Report

- **Date:** 2026-09-12
- **Status:** DONE
- **Task contract:** `docs/tasks/v1.4.1/tasks/P1/P1-04_wire_worker_to_compiled_authority.md`
- **Depends on:** P1-01 (bundle loader), P1-02 (risk derivation), P1-03 (compilation assembler) — all DONE.

## 1. Deliverable

`runV14Task`（`cmd/worker/v14.go`，P1-04 primary path per PATH_OWNERSHIP）现在按固定前置顺序执行：

```text
artifacts.LoadBundle(repo, "HEAD", taskID)   ← P1-01:一次 revision 解析,四类制品同一 pinned 提交(DL-04)
→ ProjectID fail-closed(同一棵提交树的注册表坐标)
→ profile standard 准入
→ bundle.CompileAuthority()                  ← P1-03:合同编译权威,PRD §16 的执行第一道门
→ (此后才允许)runtime 选择 → held-out 前提 → workspace manager → ContextPack
→ 事实面/闭环组合 → loop.Run
```

PRD §16「Worker 不允许绕过 Compiler」与 Gate C「100% supported production
executions use CompiledContract」从缝级事实（P1-03 收尾时的状态）升级为**生产组合级
事实**：唯一的 worker 执行入口先取 CompiledAuthority,任何 fail-safe 触发都不开工。

## 2. 为什么这是最窄的合法接缝（以及被拒绝的替代方案）

- **接缝选择：`runV14Task` 第 1–2 步换装。** 任务 Goal 的原文是「obtain
  CompiledContract **before creating an execution workspace/runtime session**」——
  组合时序本身就是语义,只有组合根能表达。改动是纯替换：`artifacts.Load` +
  `findV13Task`/`findV13Verification` → 单次 `LoadBundle` + `CompileAuthority`；
  下游全部既有件（`v13Runtime`/`enforceHeldOutPrerequisites`/`workspace.New`/
  `v13ContextPack`/sinks/`closedloop.NewLoop`）零改动、零新抽象。
- **拒绝：保留 `artifacts.Load` 并叠加 `LoadBundle`。** 两次 revision 解析对
  可动的 "HEAD" 可能跨并发提交推进落在两个提交上——恰是 DL-04 禁止的混合来源。
  单次装载是唯一诚实形态,为此 bundle 需要补 ProjectID 坐标（见 §3）。
- **拒绝：为编译身份加 codegen_run 列 / workflow.RunBegin 字段。**
  `internal/workflow`、`internal/runstore`、`internal/store`（sqlc）、`migrations`
  均不在本任务 Allowed Paths,后两者还是 §10 热点。运行事实的记录在既有坐标内
  完成（见 §4）,持久槽位扩展明确归 P1-05（context_hash）/P1-07（读模型 basis）。
- **拒绝：把编译失败引入闭环（例如喂给 FailureRouter）。** AGENTS.md §5/§12：
  编译权威错误是执行前置失败,不是 repairable candidate failure——上抛为 job
  错误,不产 verdict、不耗修复预算、不进谱系（测试钉住：零 begins/finishes/
  changeSets、零会话、零 worktree）。

## 3. 语义变更 / changed files

| 文件 | 语义变更 |
|---|---|
| `cmd/worker/v14.go` | ① 装载换 `LoadBundle`（单次解析,DL-04）；② `CompileAuthority()` 前置为执行第一道门（先于 runtime 选择/held-out/workspace/会话）,错误经 `%w` 类型化穿透（`artifacts.ErrPolicyRejections`/`ErrSourceMismatch`/`intentevidence.ErrInvalidEvidence`/`risk.ErrMalformedSignal`/loader 三哨兵均可 `errors.Is` 判类）；③ 编译身份结构化日志（task/source_commit/bundle/compiled/risk 信号计数）在会话开启前落观测；④ 运行事实坐标（project/task 修订/verification 修订/base 提交）全部改从同一 bundle 取值；⑤ `v14CodegenRunner` 增加 `authority *artifacts.CompiledAuthority` 字段（P1-05 防御上下文、P1-06 验收守卫的取用点,注释明确 runtime/候选侧不得改写）。 |
| `internal/artifacts/v141_bundle_loader.go` | `ExecutionBundle` 增加 `ProjectID` 坐标:从**同一棵提交树**读 `.tally/project.yaml`（复用同包 `parseManifest`,零第二套解析）,排在四类权威制品**之后**装载——P1-01 钉住的类型化拒收序（缺 intent/task/verification）逐字不变；注册表缺席为空串（装载器保持任务作用域,不把 v1.3 全仓纪律强加给 bundle）,空身份由 worker 执行前 fail-closed；注册表存在但非法按 `parseManifest` 严格裁决整体失败。bundle 身份/修订集不变（ProjectID 不进身份——与 P1-01 AC 一致,身份只含任务身份+来源提交+装载修订）。 |
| `cmd/worker/v14_test.go` | 新增 fixture（`newV14Repo`/`newV14HeldRepo`/`newV14RejectedPolicyRepo`/`v14IntentArtifact`——v14 路径的仓库 fixture 必须是 bundle 完整的,即含冻结意图制品）与 `v14StartSpy`；既有 11 处 v14 仓库夹具换 `newV14Repo`/`newV14HeldRepo`（v13_test.go 零改动,v13 路径不受影响）。 |

## 4. 编译身份/依据入运行事实（实现要求 2 的落点与边界）

本任务路径内可动的事实面只有 worker 组合本身。落地形态：

- **运行事实坐标全源同源**：`RunBegin` 携带的 project、task 修订
  （`bundle.Task.Contract.Hash`）、verification 修订（`frozen.Hash()`）、base 提交
  （`bundle.SourceCommit`）与 ContextPack 输入全部来自**同一** bundle——即编译
  权威的全部 pinned 输入。bundle 身份与 `Compiled.Hash()` 是这些坐标的确定性
  纯函数（P1-03 已测试钉住）,新测试 `TestV14RunFactsPinCompiledAuthority` 在
  worker 事实面复验:重装载+重编译逐项复算一致。
- **执行起点结构化观测**：编译成功后、任何会话开启前,`slog.Info` 落 task、
  source_commit、bundle 身份、compiled 哈希、risk 信号计数（含 unknown 计数,
  INV-V14-09 可见性）。
- **权威随组合携带**：`v14CodegenRunner.authority` 供 P1-05（`Compiled.Requires()`
  进 ContextPack → compiled-aware context_hash）与 P1-06（风险调整验收守卫）
  在其 owned 面完成各自的持久化/投影。
- **明确不做**（越界,已留归属）：不改 `workflow.RunBegin`/`StartRunSpec`/schema。
  每次运行的持久 compiled 槽位 = P1-05 的 context_hash（owned slot,`Input.Requires`
  已存在并纳入 context_hash）+ P1-07 的读模型 `compiled:<hash>` basis（owned
  files:`internal/taskstate`/`v14_change.go`）。P1-04 不抢这两块,只把权威递到位。

## 5. Exact source/tests read（L3 evidence）

- 任务合同 + L3 Reference Pack;`DECISION_LOCKS.md` 全文（DL-01/02/03/04/05/09/16/17 直接约束本任务）;`PRD_AMENDMENTS.md` 与 `SOURCE_BASELINE.md`（合同点名要求;SOURCE_BASELINE 的 "`contract.Bind()` 无生产 worker 调用点" 缺口由本任务关闭）;PRD §5/§14/§16/§34/§38.5。
- `PATH_OWNERSHIP.md`（`cmd/worker/v14.go` → P1-04 first;`cmd/worker/**` runtime composition 归 P4-01——本任务零重构）。
- 参考序（合同指定）：`cmd/worker/v14_test.go` → `test/arch/v14_runtime_boundary_test.go` → worker config/deps + assembler 产出 → `cmd/worker/v14.go` 组合序 → 失败路径证明。
- 实现事实：`internal/artifacts/v141_contract_assembler.go`（CompiledAuthority/装配序/fail-safe 哨兵）、`v141_bundle_loader.go`（ExecutionBundle/装载序/类型化哨兵）、`v13_loader.go`（Snapshot/parseManifest）、`internal/intentevidence/riskderive.go`（DeriveRisk 语义）、`internal/risk/derive.go`（空输入合法）、`internal/engineeringpolicy/parse.go`（拒收路径）、`internal/workflow/v13.go`（RunBegin/RunStoreSink）、`internal/runstore/codegen.go`（StartRunSpec）、`migrations/0019_codegen_changeset.sql`（codegen_run 列语义——确认无合法空槽,支撑 §2 的拒绝项）、`internal/taskstate/{state,basis}.go`（`compiled:<hash>` 词表归 P1-07 消费）、`docs/tasks/v1.4.1/reports/P1-03_*.md`（移交注记:调用时序 + 身份记录归 P1-04）。
- 外部参考：未使用（任务 pack 明示不需要）。

## 6. Tests/commands executed (exact results)

全部实际执行（Windows Git Bash,Go 1.x toolchain）：

| 命令 | 结果 |
|---|---|
| `go build ./...` | PASS |
| `go vet ./cmd/worker/ ./internal/artifacts/` | PASS |
| `go test ./cmd/worker/ -run 'TestV14CompileFailureBlocksRuntimeStart\|TestV14RunFactsPinCompiledAuthority' -count=1 -v` | PASS（3/3 子测试） |
| `go test ./cmd/worker/ -count=1` | PASS（全量,15.5s） |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres?sslmode=disable go test ./cmd/worker/ -run 'TestV14' -count=1 -v` | PASS 17/17（含 `TestV14ClosedLoopPersistsFacts`、`TestV14RepairSupersessionPersists` 两个真库持久化测试,本地 Docker PostgreSQL 15432 可达） |
| `go test ./internal/artifacts/ -count=1` | PASS（P1-01/P1-03 既有合同逐字保持） |
| `go test ./test/arch/... -count=1` | PASS（DL 守卫完好） |
| `go test ./internal/... -count=1` | PASS（全部包） |
| `go test ./test/runtimeconformance/ -count=1` | PASS |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -count=1` | PASS（201.8s;e2e 的 `clRunTask` 是测试侧等价组合,不调用 `runV14Task`,基线不受影响;其编译权威化属 P1-08 的 test/** 面） |
| `git diff --check` | clean |
| `make check` | NOT_RUN(Git Bash 环境无 make 二进制;构成命令已逐条执行) |

## 7. Negative/failure-path tests added (all PASS)

- `TestV14CompileFailureBlocksRuntimeStart/拒收策略阻断编译`：坏策略制品 →
  `errors.Is(err, artifacts.ErrPolicyRejections)`；**Start spy 零次**、零 codegen
  事实、一次性根为空——AC「no session starts before successful compilation」的
  负面证明（no-runtime-on-failure）。
- `TestV14CompileFailureBlocksRuntimeStart/缺意图制品阻断装载`：bundle 不完整 →
  `errors.Is(err, artifacts.ErrIntentNotFound)`、零会话——装载哨兵先于一切执行。
- `TestV14RunFactsPinCompiledAuthority`：成功路径恰好一次 Start（先编译后开会话
  的正面证明）+ 运行事实坐标与重装载重编译逐项一致（身份确定性复算）。

## 8. Baseline failures observed

None。全部套件在改动前后均 green;既有失败为零。

## 9. Acceptance criteria

- **AC — Runtime start spy proves no session starts before successful compilation:
  PASS。** `v14StartSpy` 统计 `Runtime.Start`：编译权威失败（拒收策略 / 缺意图）
  ⇒ 零会话、零 codegen 事实、零一次性 worktree;编译成功 ⇒ 恰好一次会话开启、
  闭环 Accepted。`cmd/worker/v14.go` 的组合序（编译在 runtime 选择/held-out/
  workspace/会话之前）是静态事实,spy 测试是其运行时证明。
- 实现要求 1（Fail before execution on compilation authority errors）：PASS——
  见 §1/§2,错误类型化、不伪造候选失败。
- 实现要求 2（Record compiled identity/basis in run facts/read model）：PASS
  （本任务路径内的完整形态,见 §4）——运行事实坐标全源同源 + 执行起点身份观测
  + 权威随组合递至 P1-05/P1-06/P1-07;持久 compiled 专列的扩展在 §10 热点
  （sqlc/migrations/workflow）内,按 PATH_OWNERSHIP 留给拥有对应面的下游任务,
  非本任务可动范围（拒绝越界是任务纪律,不是缺口——见 §11 risks）。

## 10. Production reachability

**PASS。** `v14Handler`/`runV14Task` 是既有生产注册路径（`cmd/worker/main.go`
组合根零改动,§10 热点未触碰）:worker 进程启动即注册 KindV14Task,本任务的编译
门就在该路径的每次 job 执行上生效——PRD §16 no-bypass 从本任务起是生产运行时
事实。落库验证经 `TestV14ClosedLoopPersistsFacts`（真 PostgreSQL schema,迁移、
Run/ChangeSet/VerificationRun/Evidence 全链）。

## 11. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `cmd/worker/v14.go`;`cmd/worker/v14_test.go`;`internal/artifacts/v141_bundle_loader.go`（全部在 Allowed Paths 内）
3. **semantic changes:** 见 §3;核心一条——v1.4 worker 不再可能在没有 CompiledContract 的情况下开工
4. **tests/commands:** 见 §6（全部实际执行,结果如实记录）
5. **baseline failures:** none
6. **remaining risks:** 见 §12
7. **Decision Lock touched:** none(文件未动;DL-04 单次解析+同源 manifest 满足;DL-16 单 runtime owner 未动;DL-17 遵守,hidden/ 未读)
8. **commit:** 见任务文件附录与本仓库 git 历史（`P1-04: …`）

## 12. Remaining risks / notes

- **每运行持久 compiled 槽位按所有权分期落地**：P1-05 落 compiled-aware
  `context_hash`（`contextpack.Input.Requires` 槽位已存在并纳入 context_hash）,
  P1-07 落读模型 `compiled:<hash>` basis。在此之前,编译身份的持久观测是
  「运行事实坐标（可确定性复算）+ 执行起点日志」的组合——复算路径已被测试
  钉住,但一个只看单行的审计者需要重装载才能得到哈希。若产品要求即时刻面,
  需要一个 ADR 来动 workflow/runstore/migrations（均非 P1-04 可动）。
- `ExecutionBundle.ProjectID` 在装载器层是可选坐标（保 P1-01 既有裁决序与测试
  逐字不动）,空值 fail-closed 在 worker 执行入口。真实生产仓库（v1.3 起）恒有
  注册表,该分支是防御边界;P1-01 的既有测试未受影响（全量 PASS）。
- e2e 测试等价组合（`test/e2e/v14_closedloop_harness_test.go` 的 `clRunTask`）
  仍是 P1-04 前的组成形态（无编译门）——`test/**` 不在本任务路径内;P1-08
  （Allowed Paths 含 `test/**`）拥有把 e2e 组合与 no-bypass 门禁对齐的权威。
- HIGH review per MODEL_ASSIGNMENT 待执行;建议重点：§2 拒绝项（尤其单次解析
  的 DL-04 论证）与 §4 的记录分期边界。
- P1-05/P1-06 与本任务共享 `cmd/worker/v14.go`（PATH_OWNERSHIP:P1-04 first;
  P1-05/P1-06 用 helper extraction 并协调）——本任务保持 runner/组合结构不动,
  下游以新增字段/助手方式接入即可。
