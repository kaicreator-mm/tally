# P1-05 — Compiled prevention context projection — Execution Report

- **Date:** 2026-09-12
- **Status:** DONE
- **Task contract:** `docs/tasks/v1.4.1/tasks/P1/P1-05_compiled_prevention_context_projection.md`
- **Depends on:** P1-04 (worker wired to compiled authority) — DONE (9d0f12c)。

## 1. Deliverable

编译权威的执行前 requires 上下文（PRD §8.3 Prevention）现在以两条确定性路径进入
v1.4 闭环（`cmd/worker/v14.go`，P1-05 primary path per PATH_OWNERSHIP）：

```text
bundle.CompileAuthority()                       ← P1-03/P1-04:执行第一道门(不变)
→ v14ContextPack(..., authority.Compiled.Requires())
    → contextpack.Select{Requires: …}           ← P1-05:Prevention 小节纳入 context_hash
    → RunBegin.ContextPackHash                  ← compiled-aware 运行事实坐标
→ v14CodegenRunner.RunCodegen
    → requires := authority.Compiled.Requires().Lines()
    → v14Instruction(..., requires)             ← Prevention 段随每轮迭代指令进入编码上下文
    → session.Execute
```

任务 AC 的两个方向就此都成为运行时事实：**runtime 收到必要的实现约束**（每轮
迭代指令携带命中策略的验收告知行），且**看不到任何 held-out 内容**（投影内容只
来自命中策略的公开声明，`contract.Bind` 的 INV-V14-07 保证 + 本任务组合级回归
测试钉住）；同时 P1-04 移交的 compiled-aware `context_hash` 落地——`Input.Requires`
槽位（`contextpack` 包既有权威，零语义新增）被注入编译权威的 `Requires()`，
权威换版必然改变上下文坐标：**修复/重放不能在事实面上悄悄换权威**。

## 2. 为什么这是最窄的合法接缝（以及被拒绝的替代方案）

- **投影源：`Compiled.Requires()`（合同实现要求 1 的原文指定）。** 它是
  `contract.Bind` 编译出的 Prevention 面，与执行后 Assurance 面（`GateKeys`）
  同源双用途；内容只含命中策略的 id/enforcement/rationale/requires 键。
  `internal/contract/**` 与 `internal/contextpack/**` **零语义改动**——两个包的
  既有权威（`Compiled.Requires()`、`Input.Requires`/`SelectInput.Requires`、
  `Pack.PolicyRequires` 小节、INV-V14-07 测试）全部原样消费，DL-18 的
  extend-not-duplicate 以零新增代码面达成。
- **worker 侧新增 `v14ContextPack`（helper extraction，PATH_OWNERSHIP 预授权）。
  ** `v13ContextPack`（`cmd/worker/v13.go`）不在 Allowed Paths 且 v13 路径不编译
  权威；v14 助手只做一处差异——把 `Requires` 注入 `contextpack.SelectInput`，
  候选面收集逻辑复用 v13.go 的既有件（`gitTreeFiles`/`visibleTestPaths`/
  `underAnyRoot`，零拷贝零改动）。零策略仓库的 v14 组装与 v13 **逐字节同哈希**
  （测试钉住），v13 路径行为完全不变。
- **拒绝：只把 requires 放进包、不改指令。** AC 的主语是 runtime——包/哈希只是
  观测坐标，`session.Execute` 的 Instruction 才是 runtime 实际收到的上下文。
  两条路径都要落，AC 才同时覆盖「收到约束」与「坐标可审计」。
- **拒绝：把 compiled 哈希直接塞进指令或包内容。** 编译身份的持久观测归
  context_hash（本任务）与读模型 basis（P1-07）；把身份字符串注入实现权威可见
  内容会增加哈希输入却不是「实现约束」，超出合同范围。
- **拒绝：缺权威时静默省略 Prevention 段。** P1-04 起 `authority` 是 runner 的
  组合不变量；本任务在 `RunCodegen` 入口加类型化守卫——缺编译权威拒绝执行
  （fail-safe），绝不无声地在未被告知验收条件的情况下开跑。

## 3. 语义变更 / changed files

| 文件 | 语义变更 |
|---|---|
| `cmd/worker/v14.go` | ① `runV14Task` 第 5 步 ContextPack 组装换 `v14ContextPack(..., authority.Compiled.Requires())`——context_hash 成为编译权威内容的敏感函数；② 新增 `v14ContextPack`（候选面与 v13 同规则 + requires 注入 `contextpack.SelectInput`）；③ `RunCodegen` 入口加编译权威类型化守卫（缺权威拒绝执行），循环前取 `Requires().Lines()` 一次，随每次迭代指令传入；④ `v14Instruction` 增加确定性 Prevention 段（`Prevention requirements (compiled contract):` + 每策略一行，`RequiresContext.Lines()` 渲染），与 Acceptance、Repair context 分离小节；⑤ 编译日志增加 `policy_requires` 计数。 |
| `cmd/worker/v14_test.go` | 新增夹具（`v14PolicyYAML`/`v14PolicyYAMLV2`——命中 schema_migration 的 gate 策略及其换版、`v14IntentArtifactWithHints`、`newV14PolicyRepo`、`newV14HeldPolicyRepo`）与 4 个新测试 + 既有指令测试的两处调用点适配与零值断言；`v13_test.go` 零改动。 |

`internal/contextpack/**`、`internal/contract/**`：零改动（合同允许路径内的权威
已完备；既有测试逐字保持 green——`TestBuildCarriesPolicyRequires`、
`TestPolicyRequiresNotBudgeted`、`TestZeroRequiresLeavesPackUnchanged`、
`TestSelectPassesThroughRequires`、`TestBindRequiresContextExcludesHeldOut`）。

## 4. AC 与验证矩阵的落点

- **AC「Runtime receives required implementation constraints but cannot see
  held-out content」**：
  - 正向（prompt/context snapshot）：`TestV14PreventionContextReachesRuntime` —
    端到端 Accepted 路径，逐迭代断言指令含确定性快照行
    `- policy db_gate (gate): 最终会按这些条件验收: data_integrity, fresh_database — 理由: …`；
    运行事实 `ContextPackHash` 与「重装载 + 重编译 + `v14ContextPack` 复算」
    逐字节一致，且确实因 requires 偏离无 requires 组装（不是候选面的平凡重复）。
  - 回归（exposure）：`TestV14PreventionExcludesHeldOutAndBindsHash` — held-out
    契约（TASK-1050 的 held_out binding HOLD-1/holdout/hold-1）与命中策略同场：
    指令与 requires 渲染行含策略验收告知、不含任何 held-out 线索（三词逐一
    反断言）。
- **修复/重放不能悄悄换权威**：同一测试提交策略 v2（同 id、不同 rationale），
  重装载重编译后 `Compiled.Hash()` 变 ⇒ compiled-aware `context_hash` 变；对照
  断言证明差异**恰好**来自 requires——同一对提交上无 requires 的组装哈希不变
  （候选面与合同内容都没变）。即：若无 P1-05，这次权威换版在上下文坐标上是
  无声的；有了 P1-05，它必然留痕。
- **确定性**：同输入重算同哈希（测试内两次重算逐字节一致）；`Match` 按策略 id
  稳定排序 ⇒ `Lines()` 渲染序确定（engineeringpolicy 既有保证 + contract 编译
  确定性测试）。
- **向后兼容**：`TestV14ContextPackZeroRequiresMatchesV13` — 零策略仓库的 v14
  组装与 v13 同哈希；`TestV14InstructionRendersSafeFeedbackOnly` 扩展断言零值
  requires 不渲染 Prevention 段。
- **负面**：`TestV14RunnerRequiresCompiledAuthority` — 无编译权威的 runner
  拒绝执行、零迭代（no-bypass 在 runner 缝的运行时落点）。

## 5. Exact source/tests read（L3 evidence）

- 任务合同 + L3 Reference Pack 全文;`DECISION_LOCKS.md` 全文（DL-04/09/17/18
  直接约束本任务）;`PRD_AMENDMENTS.md`（A-04 规则治理面）与 `SOURCE_BASELINE.md`
  （合同点名要求）;`PATH_OWNERSHIP.md`（`cmd/worker/v14.go` → P1-05/P1-06 helper
  extraction;`internal/contract/**` 归 P1 lane）。
- 参考序（合同指定）：`internal/contextpack/requires_test.go` →
  `contextpack_test.go::TestHeldOutProjectionExposesStatementAndTypeOnly` →
  `selector_test.go::TestSelectFiltersHeldOutInternals` → 接口
  （`contract.Compiled.Requires()`、`contextpack.Input/Build`/`SelectInput`）→
  核心实现（`contextpack/build.go`、`redact.go`、`cmd/worker/v14.go` worker
  context 构造）→ 失败处理（requires 只含策略公开内容，绝不含 held-out
  store/ref/content）。
- 实现事实：`internal/contract/compile.go`（RequiresContext/Lines/INV-V14-07/
  编译哈希规范形）、`cmd/worker/v13.go`（v13ContextPack 候选面规则、
  gitTreeFiles/visibleTestPaths/underAnyRoot 复用面——只读未改）、
  `internal/artifacts/v141_contract_assembler.go`（CompiledAuthority）、
  `internal/engineeringpolicy/match.go`（稳定排序 ⇒ 渲染序确定）、
  `internal/intentevidence`（RiskHints 夹具语义）、P1-04 报告（移交注记：
  authority 字段与 owned slot 归属）。
- 外部参考：未使用（任务 pack 明示不需要）。

## 6. Tests/commands executed (exact results)

全部实际执行（Windows Git Bash,本地 Docker PostgreSQL 15432 可达）：

| 命令 | 结果 |
|---|---|
| `go build ./...` | PASS |
| `go vet ./cmd/worker/ ./internal/contextpack/` | PASS |
| `gofmt -l cmd/worker/v14.go cmd/worker/v14_test.go` | clean（无输出） |
| `go test ./cmd/worker/ -run 'TestV14PreventionContextReachesRuntime\|TestV14PreventionExcludesHeldOutAndBindsHash\|TestV14ContextPackZeroRequiresMatchesV13\|TestV14RunnerRequiresCompiledAuthority\|TestV14InstructionRendersSafeFeedbackOnly' -v` | PASS（5/5） |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./cmd/worker/` | PASS（全量,48.1s;含 `TestV14ClosedLoopPersistsFacts`、`TestV14RepairSupersessionPersists` 真库持久化测试,单独 `-v` 复跑确认 RUN+PASS 非 SKIP,7.87s/7.78s） |
| `go test ./internal/contextpack/... ./internal/contract/... ./internal/artifacts/... ./internal/engineeringpolicy/...` | PASS（邻包权威零回归） |
| `go test ./test/arch/...` | PASS（DL 守卫完好） |
| `go test ./internal/...` | PASS（全部包,EXIT=0） |
| `go test ./test/runtimeconformance/ -count=1` | PASS |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -count=1` | PASS（e2e 的 `clRunTask` 为测试侧等价组合、不调用 `runV14Task`,基线不受影响;其编译权威化归 P1-08 的 test/** 面） |
| `git diff --check` | clean |
| `make check` | NOT_RUN(Git Bash 环境无 make 二进制;构成命令已逐条执行) |

## 7. Negative/failure-path tests added (all PASS)

- `TestV14PreventionExcludesHeldOutAndBindsHash`（暴露回归 + 权威换版敏感性）：
  held-out 材料三词反断言（HOLD-1/holdout/hold-1 不在指令与渲染行）；策略 v2
  换版 ⇒ context_hash 必变、无 requires 对照哈希必不变。
- `TestV14RunnerRequiresCompiledAuthority`：缺编译权威 ⇒ 类型化拒绝执行 +
  零迭代（绝不无声开跑）。
- `TestV14InstructionRendersSafeFeedbackOnly`（扩展）：零值 requires 不渲染
  Prevention 段。

## 8. Baseline failures observed

None。全部套件改动前后均 green;既有失败为零。

## 9. Acceptance criteria

- **AC — Runtime receives required implementation constraints but cannot see
  held-out content: PASS。** 正向由 `TestV14PreventionContextReachesRuntime`
  的端到端指令快照证明（每轮迭代携带命中策略验收告知行）;反向由
  `TestV14PreventionExcludesHeldOutAndBindsHash` 的暴露回归证明（held-out 契约
  与命中策略同场时零泄漏）,纵深由 `contract.Bind` 的 INV-V14-07 既有测试与
  contextpack 的 redaction/provenance 测试共同托底。
- 实现要求 1（Use `Compiled.Requires()` or existing canonical projection）：
  PASS——`authority.Compiled.Requires()` 同时进 ContextPack（hash 面）与指令
  （prompt 面）。
- 实现要求 2（Separate prevention context from verification evidence/holdouts）：
  PASS——包内 `PolicyRequires` 与 `Verification`（redacted projection）是分离
  小节（contextpack 既有权威）;指令内 Prevention 段与 Acceptance、Repair
  context 是分离段,渲染输入类型上不含任何验证证据定位（`[]string` 渲染行,
  来源仅命中策略公开字段）。

## 10. Production reachability

**PASS。** 变更落在 `runV14Task`（生产注册路径 `v14Handler` 的执行体,组合根
`cmd/worker/main.go` 零改动）与 `v14CodegenRunner.RunCodegen`（唯一生产
CodegenRunner）:每个 v14 job 的编译权威 requires 从此既进运行事实
（`codegen_run.context_pack_hash`,真库持久化已验证）也进 runtime 指令。
`internal/contextpack`、`internal/contract` 零改动,不产生新生产面。

## 11. Completion report (per task contract format)

1. **status:** DONE
2. **changed files:** `cmd/worker/v14.go`;`cmd/worker/v14_test.go`（全部在
   Allowed Paths 内）
3. **semantic changes:** 见 §3;核心一条——编译权威的执行前 requires 成为
   runtime 上下文与 context_hash 的必现成分,权威换版必然留痕于运行事实
4. **tests/commands:** 见 §6（全部实际执行,结果如实记录）
5. **baseline failures:** none
6. **remaining risks:** 见 §12
7. **Decision Lock touched:** none（文件未动;DL-04 同源输入未被扰动;DL-09
   runtime-active 策略仍以 pinned 快照精确内容消费;DL-17 遵守,hidden/ 未读;
   DL-18 零重复——全部权威既有件原样消费）
8. **commit:** 见任务文件附录与本仓库 git 历史（`P1-05: …`）

## 12. Remaining risks / notes

- **context_hash 覆盖的是 Prevention 面,不是编译身份全量。** `Requires()` 的
  内容（命中策略集 + 其公开声明）对策略内容/命中集变化必然敏感;纯风险信号
  变化（不动命中集）只改 `Compiled.Hash()`（assurance 面）而不改 requires——
  该场景的权威差异仍由 P1-04 已钉住的运行事实坐标（base 提交/task 修订/
  verification 修订,均可确定性复算出编译哈希）承载,读模型 `compiled:<hash>`
  basis 归 P1-07。这是合同指定投影（`Compiled.Requires()`）的固有边界,不是
  缺口。
- **指令体积**：每轮迭代指令携带全部命中策略的告知行;P1 仓库策略数小
  （十数量级),行长有界（策略 id/enforcement/键/rationale）;token 预算与指令
  裁剪归 Context Engine 后续任务,本任务不引入预算机制（requires 是必现内容,
  contextpack 侧已有不参与预算的既有裁决）。
- `test/e2e` 的测试侧等价组合（`clRunTask`）不含本任务的 Prevention 接线——
  test/** 归 P1-08 对齐。
- HIGH review per MODEL_ASSIGNMENT 待执行;建议重点：§2 的接缝拒绝项（尤其
  「两条路径都要落」的 AC 论证）与 §12 第一条的覆盖边界陈述。
- P1-06 与本任务共享 `cmd/worker/v14.go`（PATH_OWNERSHIP:helper extraction 并
  协调）——本任务未动 `authority` 字段的既有注释与 P1-06 预留语义,runner 仅
  新增入口守卫与 requires 取用。
