# P4-02 Execution Report — Codex runtime regression through factory

- **Status:** DONE
- **Date:** 2026-09-13
- **Baseline:** A0+P0…P3+P4-01+P6-01+P6-05+P5-03 DONE(31/57),master `f7211d3`;Kind: implementation,Execution model: LOW
- **Goal:** 证明 adapter refactor(switch → P4-01 组合注册表)不改变 Codex 的身份、取消、workspace、事件与 ChangeSet 行为。

## 0. 结论(一句话)

**零生产代码改动**:P4-01 已证明 factory 产物与直接构造**结构等价**(描述符逐字段一致、CodexBin 确实进入适配器);本任务补上**行为级**证据 —— 经 factory 拿到的生产 CodexRuntime 通过完整 PRD §35 conformance(13 场景),并在生产注册表路径上跑完 v13 垂直路径产出不可变 ChangeSet 且通过最终验证,失败 claim 零伪造候选。`runtimes/codex/**` 零触碰、无缺陷发现(「regression-only unless defect discovered」的 regression 分支)。

## 1. Acceptance criteria

| AC | 证据 | 结果 |
|---|---|---|
| Existing Codex-focused tests remain green | 既有 Codex 测试面全部原样 PASS:`runtimes/codex`(64,含 L3 指定锚点 `runtime_test.go::TestCodexRuntimeConformance` 与 cancellation/close/structured-output 全套)、`test/runtimeconformance`(67)、`test/arch`(100,P1-08 no-bypass 机器守卫,LoopConfig 字面量零触碰)、`cmd/worker` 既有 62 测试零回归 | PASS |
| No new Git/acceptance authority moves into runtime | 零生产代码改动 —— 无新接口、无新权限;组合回归直接证明验证/验收权威仍在编排侧:codex 路径的 ChangeSet 经 workflow 内核候选固定钩子产生,verified 结论经 candidateverify(替身只产出 claim) | PASS |

## 2. Changed files(全部在 Allowed Paths:cmd/worker/** + 指定报告)

- `cmd/worker/runtimefactory_codex_test.go`(新)— P4-02 回归证明,3 个测试 + 1 个现场构建的最小替身 codex CLI(见 §3)。
- `docs/tasks/v1.4.1/tasks/P4/P4-02_codex_runtime_regression_through_factory.md`(改)— 本任务 Execution status 章节(AGENTS.md §15)。
- 本报告。

`cmd/worker/main.go`、`v13.go`、`v14.go`、`runtimefactory.go`、`runtimes/codex/**`、`internal/agentruntime/**` 全部零触碰。

## 3. Semantic changes

**生产语义变化:无。** 以下均为测试面新增:

1. **替身 codex CLI(`factoryCodexSource`)**:现场 `go build` 的独立进程,只实现本回归驱动的实测表面(`--version`;`exec [flags] [resume|fork <id>] <prompt>`,旗标仅在 exec 层;JSONL:thread.started/turn.started/turn.completed;`-o` last-message 产物为权威完成证据;resume 续用 thread id,fork/new 落新 id;指令含 marker 时阻塞直到被 kill)。行为开关经继承环境注入(`TALLY_FACTORY_CODEX_RECORD/_SCENARIO/_WRITE_PATH/_WRITE_CONTENT`)。**刻意最小**:深 CLI 表面(argv 顺序/clap 拒绝/警告权威化/schema 违约负例)的权威探测仍在 runtimes/codex 的替身与本回归共同运行的 runtimeconformance 裁决里,本替身不重复、不漂移权威 —— 两替身各司其职,任何一方的协议形状退化都会在对应套件里响亮失败。
2. **回归 1 — conformance 经 factory(`TestCodexRuntimeThroughFactoryPassesConformance`)**:runtime 经 `newRuntimeRegistry().build(v13Config{Runtime:"codex", CodexBin:替身})` 构造 —— **与生产 builder 逐字同形**(factory 面唯一生产配置就是 CodexBin,替身行为开关走继承环境,`runner` 把 spec.Env 追加在继承环境之后,零 factory 改动)。对它跑完整 `runtimeconformance.Run`:start / cwd / execute / result / events / cancel / crash / workspace / usage / capability_truth + 已声明能力专项 cap:session.resume / cap:session.fork / cap:output.structured(断言必须真的跑过);未声明 session.isolation → canary 不运行(声明真伪不变)。探针全部指向 factory 产物自身:CrashRun 经 `*codex.Session.RunProcess` 注入外部 kill;**Fork 用 factory 产物自身的 `Fork` 方法**;StructuredOutput 经 `SetOutputSchema` 往返;WorkspaceEffect 读替身调用记录断言 `-C` 值与进程 cwd 都等于提供的 workspace(workspace 钉定经 factory 不变)。身份/取消/workspace/事件四个行为面一次覆盖。
3. **回归 2 — 组合面 ChangeSet(`TestV13CodexRuntimeThroughFactoryProducesChangeSet`)**:走**生产注册表**(`runV13Task` 且 `deps.runtime=nil` → `workerRuntimes.build(cfg)` 的生产选择行),cfg.Runtime="codex"。替身把实现文件写进进程 cwd(= 适配器钉定的一次性 worktree;若 factory 弄丢 workspace 管道,文件落错位置、候选固定失败,本测试即失败)→ 垂直路径与 fake 路径同构:TaskCompleted/1 run → 不可变 ChangeSet(含 `internal/widget/greeting.go`)→ visible binding(go version)verified。附加断言:`RunBegin.RuntimeName == "codex"`(factory 选择的身份如实进入 run 事实/观测面)、owner 主 worktree 零污染、一次性根清空(既有纪律经 codex 路径不变)。
4. **回归 3 — 负例(`TestV13CodexRuntimeFailedClaimNeverFixesCandidate`)**:替身 stderr + exit 7(无 `-o` 产物)→ 适配器 RunFailed claim → 内核按失败迭代收束(2 次失败迭代达升级边界,无 ResolveEscalation 钩子 → `escalation_unavailable`)→ needs_review,**零 ChangeSet、零最终验证、run 终态非 completed**。信任不变量(claim ≠ 事实,失败不伪造候选)经 factory 路径不变。
5. **为什么这是最窄的有效缝**:要证明的对象是「factory 产物 = 生产 codex 路径且行为不变」;factory 的输出就是 `agentruntime.Runtime` 值,唯一可行为验证的方式就是把协议级 conformance 与组合级垂直路径打到该值上。测试全部落在 cmd/worker(factory 所在包),内核/协议包/适配器零改动;不复制 runtimes/codex 测试替身(避免双权威),不新建 test/** helper 包(避免为单一消费者加包)。

## 4. Tests / commands executed(exact)

| 命令 | 结果 |
|---|---|
| 基线(改动前)`rtk go test ./runtimes/codex/... ./test/runtimeconformance/... -count=1` | PASS 131(codex 64 + runtimeconformance 67,分包复测一致) |
| 基线(改动前)`rtk go test ./cmd/worker/... ./test/arch/... -count=1` | PASS,零 FAIL(rtk 聚合读数 149 与分包实测 worker 62 + arch 100 = 162 存在聚合计数偏差;无任何失败,分包数以直测为准) |
| `go vet ./cmd/worker/` | PASS |
| `go test ./cmd/worker/ -run 'TestCodexRuntimeThroughFactory\|TestV13CodexRuntime' -count=1 -v` | PASS 3/3(conformance 13 场景全绿 2.12s;组合 ChangeSet 1.13s;负例 0.73s) |
| 终验 `rtk go test ./cmd/worker/... ./runtimes/codex/... ./test/runtimeconformance/... ./test/arch/... -count=1` | PASS 296(65+64+67+100,分包直测一致;原始 `go test` ok 行逐包确认) |
| 终验 `rtk go test ./cmd/worker/ -count=1` | PASS 65(基线 62 + 新增 3;`env -u DATABASE_URL` 直测) |
| 终验 `rtk go test ./runtimes/codex/ ./test/runtimeconformance/ -count=1` | PASS 131(64+67) |
| 终验 `rtk go test ./test/arch/ -count=1` | PASS 100(含 P1-08 no-bypass:LoopConfig 字面量零触碰) |
| DB 回归 `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres rtk go test ./cmd/worker/... -count=1` | PASS 71(共享开发实例;基线 68 + 新增 3,DB 组合回归从 SKIP 转 PASS) |
| e2e 证据 `DATABASE_URL=… go test -tags=integration ./test/e2e/v14_codex_trust/ -run 'TestP04' -count=1` | SKIP(环境前提缺失:`TALLY_E2E_CODEX_AUTH_FILE` 真实 codex 凭据,任务声明的前提,非失败;且该 e2e 构造 `codex.New` 直连,不经 factory,不构成本任务的 factory 证据) |
| `gofmt -l cmd/worker/runtimefactory_codex_test.go` | clean(gofmt -w 后) |
| `git diff --check` | clean |
| 新文件行尾 | LF only(`.gitattributes` 权威) |

不跑 repo-wide integration suite(orchestrator post-commit 负责)。

## 5. Baseline failures observed

None(基线与终验全绿)。工作树中 `internal/gitintegration/**`(P5-04)、`runtimes/dsh/**`(P4-04)、`scripts/export/` 为并行 lane 在途改动,零触碰、零干扰(e2e 编译含这些 lane 的当前状态,一次通过)。

## 6. Remaining risks

- **替身 CLI 的版本敏感性**:替身钉住的是本回归驱动的最小观测表面;真实 codex CLI 表面变化由 runtimes/codex 的权威替身 + `TALLY_CODEX_LIVE` 门控测试与 P4-03 带凭据再验证负责,不在本任务面内。
- **继承环境注入(t.Setenv)是测试基础设施事实**:它利用 `runner` 把 spec.Env 追加在继承环境之后的既有语义(生产透传 Config.Env 同机制),不改变任何生产行为;若未来 runner 改为封闭环境快照,替身开关需改走 `codex.Config.Env`(届时 factory 面才需要携带 env 配置 —— 本任务判定不该由回归任务先行扩面)。
- **workerRuntimes 生产单例在组合回归中被只读使用**:登记只发生在构造时,测试未登记任何东西;与 P4-01 报告的「登记点唯一」约定一致,无跨测试污染。
- `docs/architecture/CURRENT_CODEBASE.md` 刷新:架构事实未变(零生产改动),无需刷新;本任务文件与报告已按 §15 更新。

## 7. Decision Locks

Not bypassed, none modified。DL-02(无第二 runtime 协议:测试只经既有 `agentruntime.Runtime` 值)、DL-13 / DECISION_LOCKS #13(替身只建模探针记录观测表面,不发明旗标;旗标集合与 buildExecSpec 实测表面一致)、#22(session.isolation 声明门:factory 产物零值配置 → 未声明 → canary 不运行,组合面 held-out 契约仍被 `enforceHeldOutPrerequisites` 拒绝)、AGENTS.md §3 Runtime lock(codex 路径仍 Tally Go → runtimes/codex → 外部 CLI 进程;runtime 不携带 Git/验证/验收权威 —— 回归 2 直接证明了权威边界)。DECISION_LOCKS.md、PRD、PATH_OWNERSHIP、共享热点 main.go/v13.go/v14.go 零触碰。跨 lane 噪音(gitintegration/dsh/scripts)未触碰。

## 8. Completion report(task contract format)

1. **status:** DONE
2. **changed files:** `cmd/worker/runtimefactory_codex_test.go`(新);任务文件 Execution status;本报告(§2)
3. **semantic changes:** 生产零改动;测试面新增三条回归证明(conformance 经 factory / 组合 ChangeSet 经生产注册表 / 失败 claim 负例)+ 最小替身 CLI(§3)
4. **tests/commands:** §4(终验 296 PASS;worker 无 DB 65 / 有 DB 71;codex+conformance 131;vet/build/diff-check/gofmt PASS;integration e2e SKIP-带理由)
5. **baseline failures:** none(§5)
6. **remaining risks:** §6
7. **Decision Lock touched:** none(§7)
8. **commit:** 未提交(orchestrator 统一 commit;工作树含其他 lane 改动,不代提交)

## 9. L3 completion evidence(Reference Pack 要求)

- **exact source/tests read:** 合同指定测试 `runtimes/codex/runtime_test.go`(全文:TestCodexRuntimeConformance、Cancel/ctx-cancel/Close/structured-output/authfail/warnok-warnauthfail/badjson/exitfail/argv 顺序)、`runtimes/codex/runtime.go`(全文)、`runtimes/codex/process.go`(classifyTurn/buildExecSpec:旗标表面、`-o` 产物权威、取消>解析>退出码>产物>error 优先级)、`runtimes/codex/isolation.go`(isolatedSpawnEnv 的 Config.Env 透传 + CODEX_HOME 强制覆盖)、`test/runtimeconformance/harness.go`(全文:13 场景裁决逻辑、Probes 合同、声明门控)、`cmd/worker/runtimefactory.go` + `runtimefactory_test.go` + `v13.go` + `v13_test.go`(fixtures/sinks/instructionFor)、`internal/workflow/v13.go`(候选固定点/失败迭代/升级边界/escalation_unavailable)、`internal/policy/execution.go`(MaxIterations=4/AfterFailedIterations=2)、`internal/runner/runner.go`(Env 追加于继承环境之后)、`test/e2e/v14_codex_trust/harness_test.go`(环境前提与直连构造确认)。
- **external reference used:** none(EXT-CODEX-MULTI / EXT-CODEX-CONFIG 为 Codex 子代理/model 路由参考,与本回归的组合缝/行为面无关,未使用)。
- **why the chosen seam is the narrowest valid seam:** 见 §3.5 —— 被证明对象是 factory 产出的 `agentruntime.Runtime` 值本身;把权威 conformance 裁决与组合级垂直路径打到该值上,不需要任何生产改动、不需要新 helper 包、不复制权威替身。
- **negative tests added:** 失败 claim 零伪造候选(非零退出 → needs_review,零 ChangeSet/零验证);conformance 内建负例(cancel in-flight、外部 kill 崩溃、缺 workspace 拒绝、未知 run 哨兵、capability_truth 未声明方向)全部对 factory 产物运行。
- **known uncertainty / version sensitivity:** §6 前两条(替身表面范围;继承环境注入对 runner Env 语义的依赖)。
- **Decision Locks not bypassed:** 确认(§7)。
