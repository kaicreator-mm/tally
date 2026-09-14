# P1-08 Execution Report — Compiled-authority no-bypass gate

- **Status:** DONE — **P1 gate: PASS**(带一项显式记录、归属其属主的例外,见 §6)
- **Date:** 2026-09-12
- **Baseline:** A0+P0+P1-01…P1-07 DONE(21/57),HEAD `9578f08`,工作树 clean
- **Kind:** gate — 证明 compilation、prevention context、risk coverage 与 assurance 全部约束真实 acceptance(P1 权威链收官)

## 1. Gate 结论与证据面

任务合同的四个验收标准全部以**真实 Git 仓库 + 真实 PostgreSQL 上的闭环执行**(测试等价组合,非 isolated unit-only 检查)证明:

| AC | 证据 | 结果 |
|---|---|---|
| Same-source invariant tested | `TestV141GateAuthorityReproducibleFromOneCommit`:一次 `clRunTask` 的编译权威,在同一 pinned HEAD 上独立 `LoadBundle`+`CompileAuthority` 重载/重编译,bundle 身份与 Compiled 哈希确定性复现,来源提交 = 观测 HEAD(一个提交,DL-04) | PASS |
| L3-required/L1-achieved blocked | `TestV141GateRequiredL3AchievedL1BlockedE2E`(命中 gate 策略上调 L3,visible-only 候选 FinalVerified 后在 accept 转移被裁决 `assurance_insufficient`,明细逐字 `assurance gap [AC-01 required L3 achieved L1]`,零修复消耗,Prevention 小节已随指令到达运行时)+ `TestV14MigrationClosedLoopE2EDefectFailsThenRepairGuardEscalates`(真实 SQL 执行的迁移闭环:修复候选三类证据齐备 FinalVerified,但 visible-only 达不成 L3 → 守卫升级,运行时裁决与读模型 VERIFIED_LIMITED 投影同源一致) | PASS |
| Held-out content not leaked | `TestV141GateHeldOutNeverLeaksThroughCompiledChannels`(held-out PASS 场景在守卫下合法接受;全部实现可见通道——含 requires/Prevention 渲染路径——对密标/store 根/不透明 ref/内容摘要零出现,INV-V14-07);既有 `TestV14ClosedLoopE2EHeldOutFailureEscalatesWithoutLeak` 在新组合下继续 PASS | PASS |
| Unknown/unmatched required risk is fail-safe | `TestV141GateUnknownUnmatchedRiskFailsafeE2E`(owner 确认的 unknown 信号无任何策略声明:Coverage 双集合逐字保真,必需档 fail-safe 上调 L3,verified 候选照样不普通接受,信号词面进入守卫明细,INV-V14-09) | PASS |

**Failure Handling(合同:蓄意 bypass 夹具必须在接受之前失败)** — `TestV141GateBypassFailsBeforeAcceptance`:
- 拒收策略制品 → `CompileAuthority` 类型化 `ErrPolicyRejections`(绝不带规则缺口执行,DL-09);
- 缺 FrozenIntent → `LoadBundle` 类型化 `ErrIntentNotFound`(缺一角也能跑的同源 bundle 不存在,DL-04);
- 混合来源手工 bundle(旧提交的 task/intent/verification + 新提交的策略快照)→ `ErrSourceMismatch`(绕过装载器的手工构造在装配入口复核被拒,DL-04 第二道)。
  三者均**零 runtime 会话**(`startsCount()==0` 实测)、零一次性 worktree 残留、零候选事实落库。

## 2. Changed files(全部在 Allowed Paths:test/**、reports/**)

- `test/e2e/v14_closedloop_harness_test.go` — P2-10 测试等价组合编译权威化(P1-04 遗留移交):`clRunTask` 改走 `artifacts.LoadBundle → bundle.CompileAuthority`(替换 `artifacts.Load` 手工扫描),执行前置序与 `cmd/worker runV14Task` 逐件一致(装载 → 项目身份 fail-closed → profile 准入 → **编译权威第一道门** → held-out 前提 → workspace → ContextPack);组合序提炼为 `clComposeTask`(error 返回面,gate 负例断言 typed 失败);`clContextPack`/`clInstruction`/`clCodegenRunner` 增加 `Compiled.Requires()` 通道(P1-05 遗留移交:Prevention 小节与 worker `v14Instruction` 同渲染,纳入 context_hash);`LoopConfig.Authority: authority.Compiled`(P1-06 遗留移交:验收守卫在测试等价组合生效);`clScriptedRuntime` 增加会话计数观测;夹具仓库冻结 FrozenIntent(`clIntentYAML`,恰四字段 schema)并参数化(`clNewFixtureRepo`);`clObservation` 暴露 Authority/Bundle 供 gate 复算断言。
- `test/e2e/v14_bugfix_harness_test.go` — P3-03 组合同样对齐:`bfNewRepo` 冻结 `TASK-2410` 的 FrozenIntent(bug_fix,无风险提示 → 权威风险输入为零信号,束经 REQ-2410 需求轴命中);`bfRunTask` 走 bundle+CompileAuthority+Requires+Authority。
- `test/e2e/v14_migration_harness_test.go` — P3-04 组合对齐:`migNewRepo` 冻结 `TASK-304` 的 FrozenIntent(**db_migration 任务类型 ⇒ P1-02 DeriveRisk 确定性派生 schema_migration**——替换原测试侧 risk 夹具派生,权威风险输入回归 FrozenIntent 证据面);`migRunTask` 删除测试侧 `LoadPolicies+contract.Compile+contract.Bind` 手工装配,统一走 `CompileAuthority`(策略/风险/编译不再可能混源),策略快照与风险信号断言改挂在 bundle/权威上。
- `test/e2e/v14_migration_closedloop_test.go` — 终态期望更新为守卫接线后的真实产品语义(见 §3.2),测试更名 `…RepairAccepted` → `…RepairGuardEscalates`,Prevention 指令断言加强。
- `test/e2e/v141_no_bypass_gate_test.go`(新)— gate E2E 证据面(§1 的 5 个测试)。
- `test/arch/v141_no_bypass_gate_test.go`(新)— repo-wide no-bypass 机器守卫(§3.3)。

## 3. Semantic changes

### 3.1 测试等价组合 = 生产执行前置序(P1-04/P1-05/P1-06 三项遗留在 test/** 面闭合)

P1-04/05/06 的报告均把「e2e 组合的编译权威化 / Prevention 对齐 / repo-wide 门」移交本任务;至此 `clRunTask`、`bfRunTask`、`migRunTask` 与 `cmd/worker runV14Task` 的执行前置序逐件一致:同源 bundle 装载(DL-04)→ 编译权威先于任何 runtime 会话与 workspace(PRD §16 no-bypass)→ `Compiled.Requires()` 进 ContextPack 与每轮迭代指令(PRD §8.3 Prevention)→ `LoopConfig.Authority` 必达(DL-07/A-03 验收守卫)。测试组合不再存在「生产有门、测试无门」的语义分叉。

### 3.2 迁移闭环 E2E 终态更新(有意语义后果,非回归)

P1-06 已记录「policy-matched visible-only 契约确定性升级而非自动接受」;当时 e2e 组合未接线守卫,迁移场景继续表达 v1.4 行为。P1-08 接线后,迁移闭环的真实语义:缺陷候选真实 FAIL → 修复候选真实三类证据齐备 → **FinalVerified 但普通 Accept 被守卫拒绝**(required L3 由 db_migration 风险上调,achieved L1 仅 visible 确定性事实)→ 终态 `assurance_insufficient` 升级,守卫明细 `acceptance guard: assurance gap [AC-304 required L3 achieved L1]`,不携带 basis、不伪装候选失败、修复预算只被真实验证 FAIL 消耗一次。运行时裁决与读模型投影(`migAssertFinalBasis`:VERIFIED_LIMITED,required L3/achieved L1,unmet [AC-304])同源一致——两个面不再各说各话。除终态块外,该测试的全部事实断言(尝试谱系、取代链、双验证运行、逐键证据、门禁求值、append-only 失败保留)原样保留且全部 PASS。

### 3.3 repo-wide no-bypass 机器守卫(P1-06 遗留的收官)

`test/arch/v141_no_bypass_gate_test.go` 把「不存在绕过编译权威的闭环组合」钉成机器检查:
- **NB1**:AST 全仓扫描(cmd/**、internal/**、scripts/**、test/**,含测试文件)——每个 `closedloop.LoopConfig` 组合字面量必须显式绑定 `Authority`;选择器名匹配(导入别名不构成绕过),closedloop 包内裸标识符形态同样纳入;例外只允许显式清单(文件+理由),且清单活性被检查(文件删除/重构后遗留的死例外必须移除)。
- **NB2**:守卫自身负例/正例对照(缺 Authority 的合成字面量必须被识别;带 Authority、别名导入、包内裸形态均按口径命中/放行)。

例外清单(显式、在案):
1. `internal/closedloop/acceptguard_test.go` — P1-06 守卫机制的 nil-authority 语义钉(`TestLoopWithoutAuthorityKeepsLegacyAccept` 等);
2. `internal/closedloop/integration_test.go` — P2-08 引擎域集成夹具(域层测试,非业务闭环组合);
3. `scripts/v14_trial/tally/trialrun.go` — P4-03 DSH trial 工具(P1-08 Allowed Paths 之外;其 Authority 对齐归 P4-03/P4-07 gate 所有)。

守卫当前实测:全仓 LoopConfig 字面量 = 生产 `cmd/worker/v14.go`(绑定)+ e2e 三个 harness(绑定,P1-08)+ 上述三个例外。

## 4. Tests / commands executed(exact)

| 命令 | 结果 |
|---|---|
| `go build ./...` | PASS |
| `go vet ./...` / `go vet -tags=integration ./test/e2e/` | PASS |
| `gofmt -l`(全部触碰文件) | 仅 `test/e2e/v14_bugfix_harness_test.go` 标记,且唯一差异是**基线已存在**的 `bfRepairFixture` 结构体对齐(本任务未触碰该 struct;不顺手格式化,不混行尾/格式化 pass) |
| `go test ./test/arch/... -count=1` | PASS(1.3s;含新增 NB1/NB2 守卫与全部既有 v13/v14/v141 守卫) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test -tags=integration ./test/e2e/ -run "TestV141Gate" -count=1 -v` | PASS(39.3s;5 个 gate 测试逐一 PASS) |
| `… -run "TestV14Migration" -count=1` | PASS(76.5s;更新后的迁移闭环 + 迁移探针真实 SQL 执行) |
| `… -run "TestV14ClosedLoop\|TestV14BugFix" -count=1` | PASS(80.2s;P2-10 矩阵 8 场景 + P3-03 bugfix 闭环) |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/... -count=1` | **PASS(365.7s,全套)** + `v14_codex_trust` PASS + `v14_migration_probe` no test files |
| `go test ./test/runtimeconformance/... -count=1` | PASS |
| `go test ./internal/... -count=1` | PASS(全套;仅 core/store 无测试文件) |
| `DATABASE_URL=… go test ./cmd/worker -count=1` | PASS(50.3s;生产组合回归不回退) |
| `git diff --check` | clean |
| `make check` | NOT_RUN(Git Bash 无 make 二进制;构成测试族已逐项运行,与 P1-03…07 同裁决) |

## 5. Baseline failures observed

None。唯一 gofmt 标记为基线已有的 `bfRepairFixture` 对齐( untouched );全部既有 e2e/arch/internal/worker 测试在新组合下 PASS(迁移闭环终态为本任务的有意语义更新,见 §3.2,非 baseline failure)。

## 6. Remaining risks / scope notes

- **`scripts/v14_trial`(显式例外,非放水)**:P4-03 的手工 trial 驱动器自身经 `contract.Compile+Bind` 编译并把 `Requires()` 接进其 ContextPack,但 `LoopConfig` 未绑 Authority(无验收守卫)。本任务 Allowed Paths(`test/**`、`reports/**`)不覆盖 `scripts/**`,不能代改;机器守卫将其列为显式例外(理由在案、活性受检),对齐责任随 PATH_OWNERSHIP 归 P4-03,其发布面裁决在 P4-07(DSH limited-readiness gate)。该工具不在任何生产/发布路径上(cmd/worker 是唯一生产组合根,恒绑定)。
- P2-06 的 `v14_holdout_repair_exposure_test.go` 直接以 `closedloop.NewEngine` 组合(先于 NewLoop 组合根的域测试面),不经 `LoopConfig`,不在 NB1 扫描口径内;其语义(暴露封顶)与守卫正交,由 closedloop 包测试钉住。
- `docs/architecture/CURRENT_CODEBASE.md` 中 P1-06 行尾的「no-bypass gate owned by P1-08」前向引用现已满足;`docs/architecture/**` 不在本任务 Allowed Paths 且生产架构事实未变,未代改,留待下一个 docs 拥有任务刷新。
- gate 判定基于本机真实 PostgreSQL(共享开发实例 15432)与真实临时 Git 仓库;Hidden Validation Pack 未读、未挂载(DL-17),gate 不预支任何 hidden 结论。

## 7. Decision Locks

Not bypassed, none modified。DL-04(同源:bundle 一次解析 + 装配入口复核 + 混合来源负例)、DL-07(守卫在测试组合与生产组合同等生效)、DL-09(拒收策略阻断,bypass 负例)、DL-17(hidden/ 未读)、DL-18(既有域件零改动——全部改动在 test/** 与 reports/**)。DECISION_LOCKS.md、PRD、共享热点文件零触碰。

## 8. Completion report(task contract format)

1. **status:** DONE(gate: PASS)
2. **changed files:** §2(4 个 e2e 文件修改,2 个新测试文件,1 个新报告;全部在 Allowed Paths)
3. **semantic changes:** §3(测试组合生产对齐;迁移闭环终态为守卫语义的有意结果;repo-wide 机器守卫落地)
4. **tests/commands:** §4(全套 integration e2e 365.7s PASS;arch 全套 PASS;internal 全套 PASS;worker PASS)
5. **baseline failures:** none(§5)
6. **remaining risks:** §6(scripts/v14_trial 显式例外归 P4-03/P4-07;NewEngine 域测试面口径说明;CURRENT_CODEBASE 前向引用刷新留待 docs 拥有任务)
7. **Decision Locks touched:** none(§7)
8. **commit:** 见 git log(`P1-08: …`)

## 9. L3 completion evidence(合同 Reference Pack 要求)

- **exact source/tests read:** 合同指定三测试全文(`test/arch/v14_contract_authority_test.go`、`test/arch/v14_assurance_failsafe_test.go`、`internal/verification/holdout_path_leak_test.go` 存在性与其 INV-V14-07 语义经 e2e 泄漏扫描面消费);`cmd/worker/v14.go`、`internal/closedloop/wiring.go`、`internal/closedloop/acceptguard.go`(经 P1-06 报告与 `internal/taskstate/acceptguard.go`)、`internal/artifacts/v141_bundle_loader.go`、`v141_contract_assembler.go`、`internal/intentevidence/artifact.go`、`internal/contract/compile.go`(riskBand/RequiresContext)、`internal/risk/{risk,derive}.go`、`catalog/v14/policies/{bug_fix,database_migration}.yaml`、三个 e2e harness 及其全部消费者(matrix/bugfix/migration closedloop)、`test/arch/{helpers,v141_decision_lock_guards}_test.go`(守卫模式)、P1-04/05/06 报告的移交条目、DECISION_LOCKS/PRD_AMENDMENTS/SOURCE_BASELINE/PATH_OWNERSHIP/STATUS。
- **external reference used:** none(L3 Pack 明示「No external implementation is required」)。
- **why the chosen seam is the narrowest valid seam:** 零生产代码改动——生产组合根自 P1-04/P1-06 起已绑定;本任务的最小合法缝就是把「测试等价组合」对齐到同一生产前置序并让门变成机器检查:组合序提炼(`clComposeTask`)+ 夹具补齐(FrozenIntent)+ 三处参数通道(requires/Authority)+ 一个 AST 守卫 + 一个 gate E2E 文件。未发明平行子系统、未触碰 internal/**。
- **negative tests added:** bypass 三连(拒收策略/缺 intent/混合来源)+ NB2 守卫负例;gate-2/3 的「verified 仍不普通接受」负路径即合同 AC 的失败面。
- **known uncertainty / version sensitivity:** bundle 身份与编译哈希依赖 `artifacts/bundle/v1`、`contract/compiled/v1` 规范版本前缀(版本演进时重算,不跨版本比较);PostgreSQL 行为面依赖本机 15432 共享实例可用性。
- **Decision Locks not bypassed:** 确认(§7)。
