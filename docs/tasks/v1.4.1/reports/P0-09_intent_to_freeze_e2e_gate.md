# P0-09 — Intent-to-Freeze E2E gate · 执行报告

- **Status: DONE**(gate 结论:**PASS**,2026-09-12)
- **Gate 性质:** P0 阶段收官 gate —— 在真实临时 Git 仓库上,经**生产路由原样装配**端到端证明「人类意图 → 动态澄清 → owner 确认 → 不可变冻结 → pinned 提交确定性读回」权威链,作为 STATUS.md「P0 Confirm→Freeze authority」hard blocker 的解除依据。
- **被测纵切(全部为生产组件,无平行实现):** `handlers.Server` + `RegisterV14Routes`(cmd/server 组合根挂载的同一注册函数)→ `POST /v14/launch/intent`(`intake.Propose`,AI Runtime 走生产注入缝 `SetV14ProposeExecutor`)→ 有界澄清问答服务端复核折叠(P0-04)→ 确认表单 → `intakefreeze.Freeze` 冻结准入(P0-07)→ `artifacts.FreezeCommit` 单一期望基线原子提交(P0-08,唯一 Git 写边界 `internal/gitwriter`)→ `artifacts.Load` / `artifacts.LoadIntent` 从 pinned 提交确定性读回(P0-02/P0-08 生产装载器)。

## 1. 实现内容

新文件 `test/e2e/v141_intent_to_freeze_gate_test.go`(`//go:build integration`,package e2e),六个测试函数,零生产代码改动(链路已由 P0-01..P0-08 全部就位;本任务只补端到端证明,符合「gate」任务类别与最小变更纪律):

1. **`TestV141IntentToFreezeGateBugFixEndToEnd`** — bug_fix 场景全链:提案 → 澄清请求(risk_choice,含 advisory 推荐,无草案 —— 含糊不发明验收)→ 澄清答案按有界 schema 服务端复核折叠 → 第二次提案产出草案(第二次 AI 调用的意图文本断言携带折叠后的 Q/A,链路连续)→ owner 从**生产页面渲染值**回收 AI 侧字段并补齐冻结输入(task_id / profile=standard / 逐条款 repo 路径型证据 / 勾选保留风险提示)→ 确认 303 flash 携带冻结提交 sha → 恰一个新提交 → 三件套从 pinned 完整 sha(非 HEAD 符号)读回:TaskContract(profile/scope/验收逐字段与确认内容一致,Hash=内容重算)、冻结 VerificationContract(1 条款 1 绑定,`AC-01-EV01` / pre_commit / visible / 证据定位与 owner 声明一致)、FrozenIntent(bug_fix / SourceRef / Revision=sha256)→ DL-06:冻结意图 RiskHints 恰为 owner 勾选保留的一条,未勾选候选绝不自动晋升 → 身份三元组一致(DL-04)→ 同提交重复装载逐字段相等(确定性)→ 冻结前基线提交 `LoadIntent` = `ErrIntentNotFound`(制品确实来自冻结提交)。
2. **`TestV141IntentToFreezeGateMigrationScenarioFreezesAtomically`** — db_migration 场景(AC1 第二场景):提案直接产草案 → 确认冻结(schema_check 证据,long_lock 风险提示保留)→ 冻结提交相对父提交**恰新增三件套三个文件**(`.tally/{intents,tasks,verifications}/TASK-0902.yaml`,别无其他 —— 原子性/无部分冻结);pinned 读回校验迁移类型与风险提示存活。
3. **`TestV141IntentToFreezeGateAdmissionBlocksFreeze`**(AC2 + 负面)— 缺失 planned evidence → 400 且类型化列出缺失条款(AC-01);证据半声明(有类型无路径)→ 400「必须同时声明」;不可承载语义(确认携带 non_scope)→ 400「无法承载的语义」;全部阻断路径零 Git 副作用(HEAD 不动、树上无任何冻结制品)。
4. **`TestV141IntentToFreezeGateInvalidClarificationBlocksChain`**(负面)— 第一轮合法提案产出澄清(1 次 AI 调用);第二轮答案在声明候选集外 → 400「澄清答案未通过服务端复核…不在候选」,无草案,**无任何新的 AI 调用**(复核失败停在纯确定性层),零提交。
5. **`TestV141IntentToFreezeGateDuplicateConfirmIdempotent`**(负面)— 完全相同的确认再提交:303 kind=ok「未产生新提交」(内容幂等,`ErrNoChanges` 路径),HEAD 不变,幂等后制品仍可从 pinned 完整读回。
6. **`TestV141IntentToFreezeGateStaleBaseConflictFailsSafe`**(负面)— 经生产 HTTP 链冻结 TASK-0907 推进 HEAD 后,以陈旧基线驱动 handler 委托的**同一生产事务** `artifacts.FreezeCommit`(产物经同一准入入口 `intakefreeze.Freeze` 构造):类型化 `ErrBaseRevision`,HEAD 不变,树上零 `TASK-0908` 部分制品;以当前基线重试同一冻结成功且可读回(冲突不损坏仓库,恢复路径可用)。*说明:V14ConfirmDraft 在请求内先读实时 HEAD 再提交,HTTP 层无法无损注入并发推进窗口 —— 与 P0-08 报告同一分区决策;本 gate 把该负面落到真实 e2e 仓库 + 生产事务函数上,而非绕过 handler 伪装 HTTP 场景。*

**AI Runtime 环节的如实区分:** gate 的提案环节经生产注入缝挂确定性替身 `airuntime.FakeExecutor`(gate 必须可复现,非确定性输入不进判定);真实端点连通性由 P0-05 的 connected 测试单独覆盖 —— 本次执行 `DEEPSEEK_API_KEY` 已设置,`TestAIProposeExecutorConnectedIntegration` 对真实 DeepSeek 端点 **PASS**(生产 intent 入口对真实 AI 产出一等结局,不 5xx、不编造)。gate 不依赖该结果;两者共同证明「真实入口 → 确定性冻结」。

## 2. Gate 运行证据(2026-09-12 记录;哈希来自当次运行的临时仓库,可用下述命令复现)

```text
GATE bug_fix:     freeze_commit=226df89d6cf7ced5b1342b708af9fe9bb50a94bb
                  task_hash=sha256:321834cc6d93a8284fe5de3cce3ea284d900a7c9b77cb818ae0423653fca3ae2
                  verification_hash=sha256:c7a1e2aa6a422484145e86e33b1af8b9bc69ad232b1af63491d3a93cbbb3ee70
                  intent_revision=sha256:0e4c4323bca2f7febddc3107df0270206aad77d878e35b979d92ca7a6f04bdb3
GATE db_migration: freeze_commit=9a55bdda9a385e1380a4262c174f09bc8ea84efd
                  task_hash=sha256:fdc33fd34a1c8825df479d427799a5c1a04730a55044b54e00b99f9dacfbe587
                  verification_hash=sha256:3f79a531992c1bf3ee95b7ce253af16e6c1fe478aa44dd39ae35b42a53cbc955
                  intent_revision=sha256:4573db5978417590596bb5b187a7749e0bb3ea280d0ae34276a9059dfa81d2e0
GATE stale_base:  conflicted_base=8789c029f768ba2e388acf7627f42eee03defd25
                  recovered_commit=09a10610b4b34693f4ce31d2d4e8985e9e7a0e6a
```

## 3. 验收判据(任务合同 AC)

- **AC「At least bugfix and migration/intake-risk scenario freeze correctly」— PASS。** 两个场景各走完整生产 HTTP 链冻结成功;pinned 读回逐字段一致;DL-06 风险提示以不可变证据存活且候选不晋升。
- **AC「Missing verification evidence blocks freeze」— PASS。** 缺证据 400 类型化列出条款 id,零提交零制品;半声明/不可承载语义同型阻断。
- **AC「All artifacts load from pinned commit」— PASS。** 三件套以完整 sha 经 `artifacts.Load`/`LoadIntent` 装载,`SourceCommit==pinned`,重复装载确定性相等,冻结前基线无 intent 制品。
- **Verification「Run E2E and record commit/artifact hashes」— PASS。** 见第 2 节。
- Reference Pack 负面用例:**missing evidence / stale base / invalid clarification** 三项全部以真实负面用例覆盖(另加重复确认幂等)。

## 4. 执行的命令与结果

| 命令 | 结果 |
|---|---|
| `go test -tags=integration -run 'TestV141IntentToFreezeGate' -v ./test/e2e/` | **PASS**(6/6,含 gate 哈希输出) |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test -tags=integration ./test/e2e/...` | **PASS**(整包 ok,327.8s;既有 e2e 全绿,零基线失败) |
| `go test ./test/arch/...` | **PASS**(cached) |
| `go test -tags=integration -run TestAIProposeExecutorConnectedIntegration ./cmd/server/`(辅助证据,非本任务合同要求) | **PASS**(真实 DeepSeek 端点,2.3s) |
| `go build ./...` | **PASS** |
| `go vet -tags=integration ./test/e2e/` | **PASS**(无 tag 的 `go vet ./test/e2e/` 因 build tag 排除文件报 "build constraints exclude all Go files",属预期,opt-in tag 套件以 tagged vet 为准) |
| `gofmt -l test/e2e/v141_intent_to_freeze_gate_test.go` | **clean** |
| `git diff --check` | **clean** |
| `make check` | NOT_RUN(reason: Git Bash 无 make 二进制;构成命令已逐条执行,与 P0-05/07/08 同一记法) |

## 5. Baseline failures

无。整包 integration e2e(含全部既有 v13/v14 闭环/harness/holdout 用例)与 arch 套件全绿;未观察到任何基线失败。

## 6. 变更文件

- `test/e2e/v141_intent_to_freeze_gate_test.go`(新增,唯一代码变更)
- `docs/tasks/v1.4.1/reports/P0-09_intent_to_freeze_e2e_gate.md`(本报告)
- `docs/tasks/v1.4.1/tasks/P0/P0-09_intent_to_freeze_e2e_gate.md`(追加 Execution status 附录)
- `docs/tasks/v1.4.1/STATUS.md`(进度 13/57 → 14/57;hard blocker #2 解除记法)

## 7. 语义变更

无生产语义变更(零生产文件改动)。测试语义:钉住 Intent→Freeze 权威链的端到端合同 —— 生产路由可达性、有界澄清复核、冻结准入 fail-safe、单一 CAS 原子提交、pinned 确定性读回、幂等与基线冲突恢复。

## 8. L3 completion evidence(Reference Pack 要求)

- **读取的精确源码/测试:** 任务合同;`DECISION_LOCKS.md`、`PRD_AMENDMENTS.md`、`SOURCE_BASELINE.md`、`PATH_OWNERSHIP.md`、`STATUS.md`;Reference Pack 点名测试 `v14_launch_test.go::{TestV14ProposeRendersEditableDraftAndConfirmBoundary, TestV14LaunchRunBlocksWithoutFrozenVerification}` 及同文件澄清/折叠/负面组;`v14_confirm_freeze_test.go`(P0-08 handler 级分区与装配注记);`internal/web/handlers/v14_launch.go`、`server.go`、`content.go`、`v13_http_test.go`(v13Repo 夹具)、`v13_tasks.go`(v13env);`internal/intakefreeze/{intakefreeze,verification}.go`;`internal/artifacts/{v141_freeze_commit,v141_intent_loader,v13_loader}.go`;`internal/gitwriter/writer.go`;`internal/verification/types.go`;`internal/intentevidence/{intentevidence,artifact}.go`;`internal/airuntime/fake.go`;`web/templates/v14_launch.tmpl`;既有 e2e 惯例(`v14_bugfix_{closedloop,harness}_test.go`、`v14_closedloop_harness_test.go`、`v13_changeset_supersession_test.go`)。
- **外部参考:** 无(pack 明示不需要;未读任何外部仓库)。
- **为何这是最窄合法缝:** 零生产改动;唯一新文件放在任务 Allowed path `test/**` 内,复用既有 e2e 惯例(integration tag、真 git 临时仓库、结构化断言),经组合根同款路由注册函数与生产注入缝驱动 —— 不新增抽象、不绕 handler、不复制领域规则。
- **新增负面测试:** 见第 1 节 3–6(缺失证据/半声明/不可承载语义/非法澄清/重复确认幂等/陈旧基线冲突+恢复)。
- **已知不确定性/版本敏感:** go-git `Commit.Parent(i)` 返回 `(Commit, error)`(版本 API 差异,已按签名处理);渲染页字段回收用受控正则,夹具内容不含需 HTML 转义的字符;gate 哈希逐次运行不同(临时仓库),判定依据是断言而非具体哈希值。

## 9. Decision Locks

未触碰(无锁文件改动;未读 `hidden/`,DL-17)。遵守:DL-03(Git 写只走 gitwriter)、DL-04(三件套同一 pinned 提交,e2e 显式断言)、DL-06(仅 owner 确认的风险提示进冻结证据,e2e 显式断言)、DL-11(澄清走有界 schema 服务端复核)、DL-18(只消费既有权威域)。

## 10. 剩余风险

- gate 的 AI 环节为生产注入缝上的确定性替身;真实端点仅由 P0-05 connected 测试覆盖(本次 PASS,但 Key/额度属环境状态,不构成 gate 的可复现部分)。
- 陈旧基线负面落在生产事务函数而非 HTTP 层(handler 在请求内读实时 HEAD,无并发注入窗口)—— 分区决策沿用 P0-08 并在测试头注记。
- 闭环执行侧(v14_task 入队/worker)不在本 gate 合同内,P1/P2 链路证明归后续任务。

## 11. Commit

见本文件 git 历史(`P0-09: intent-to-freeze e2e gate — …`)。
