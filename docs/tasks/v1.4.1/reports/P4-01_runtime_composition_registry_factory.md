# P4-01 Execution Report — Runtime composition registry/factory

- **Status:** DONE
- **Date:** 2026-09-12
- **Baseline:** A0+P0+P1+P2-01…P2-09+P3 DONE(22/57),master `c6767d7`;Kind: implementation,Execution model: LOW
- **Goal:** 把 v13/v14 组合根里硬编码的 fake/codex 选择 switch,替换为一个小型 runtime 注册表/工厂,产出值仍是 `agentruntime.Runtime` —— 不新增执行核心接口。

## 1. Acceptance criteria

| AC | 证据 | 结果 |
|---|---|---|
| Existing fake/codex paths behave identically through factory | `TestRuntimeFactoryMatchesDirectConstruction`(factory 产物 vs 直接构造:Descriptor name/version/capabilities 逐字段一致,CapabilitySet 按 Sorted 序列比较);`TestRuntimeFactoryPlumbsCodexBin`(CodexBin 确实进入适配器:不存在的显式路径在 Start 处失败,错误与直接构造逐字一致 —— requireFile 存在性检查先于任何进程动作,零 spawn);全部既有 v13/v14 组合回归原样 PASS(含 `TestV13RuntimeSelectionIsConfiguration` 的 codex 显式选择与 `TestV13HeldOutPrerequisitesEnforced` 的 `cfg.Runtime="codex"` 全流程,均经 factory) | PASS |

**Failure Handling(L3 合同:unknown runtime/config 在任务执行前拒绝)** — `TestRuntimeFactoryRejectsUnknownBeforeExecution`:`runV13Task` 配 `Runtime: "magic"` → 类型化拒绝(错误含「未知 runtime」与名字),**零 run 事实**(sink begins/finishes = 0)、**零一次性 worktree 根创建**(选择失败先于 workspace/Create),fail-closed 绝不静默降级到缺省 runtime。v14 路径同一工厂同一拒绝点(选择在第 3 步、编译权威第 2 步之后、任何会话/workspace 之前)。

## 2. Changed files(全部在 Allowed Paths:cmd/worker/** + 本报告)

- `cmd/worker/runtimefactory.go`(新)— runtime 组合注册表:`runtimeBuilder`(v13Config → agentruntime.Runtime)、`runtimeRegistry`(name → builder + 缺省名 fake)、生产实例 `workerRuntimes = newRuntimeRegistry()`(登记 fake.DescriptorName → `fake.New(fake.Config{})`、codex.DescriptorName → `codex.New(codex.Config{BinPath: cfg.CodexBin})`,构造式与原 switch 逐字相同)、`register`(登记缝)、`build`(空名 → 缺省 fake;未登记名字 → 错误,支持集由登记键序推导)。
- `cmd/worker/v13.go`(改,P4-01 是本 wave cmd/worker runtime 组合的独占重构;§10 热点触碰在此声明)— 删除 `v13Runtime` 硬编码 switch 及其小节;步骤 2 改为 `workerRuntimes.build(cfg)`(错误经 `fmt.Errorf("v13: %w", err)` 保留路径前缀);移除 fake/codex import(随 switch 迁往 factory);文件头边界注释更新为注册表表述。
- `cmd/worker/v14.go`(改)— 步骤 3 同一工厂 `workerRuntimes.build(cfg.v13)`(错误前缀 `v14: `);`v14Deps.runtime` 注释更新。
- `cmd/worker/runtimefactory_test.go`(新)— 4 个聚焦测试(见 §1/§4)。
- `cmd/worker/v13_test.go`(改)— `TestV13RuntimeSelectionIsConfiguration` 改指注册表缝(同一矩阵:缺省 fake/显式 fake/显式 codex/未知拒绝,新增「错误自述支持集」断言);顺手修复本文件**基线已有**的 gofmt import 顺序漂移(changeset/candidateverify 两行互换,文件属本任务触碰范围,非行尾 pass)。
- `cmd/worker/v14_test.go`(改)— 仅一处文件头注释更新(v13Runtime → workerRuntimes 注册表),零语义改动。

`cmd/worker/main.go` **未触碰**:工厂是包级组合根实例,v13/v14 handler 装配签名不变,无需穿线。

## 3. Semantic changes

1. **唯一名字分支点收拢**:`fake`/`codex` 名字 → 适配器的映射从 v13.go 的硬编码 switch(被 v14 复用)收进 `cmd/worker/runtimefactory.go` 的注册表;它是 worker 内唯一按 runtime 名字分支的点,且只存在于组合边界 —— workflow / closedloop / candidateverify 内核只接收 `agentruntime.Runtime` 值,控制流仍只按能力协商结果分支(PRD §10.3;enforceHeldOutPrerequisites 的 CapabilitySet 裁决原样未动,能力逻辑保持 runtime 中立)。
2. **无新执行核心接口**:工厂产出值就是既有 `agentruntime.Runtime`;无第二 lifecycle 协议(DL-02 / AGENTS.md §3 Runtime lock)。扩展缝 = 在 `newRuntimeRegistry` 登记一个 builder:后续 runtime(如 DSH,P4-04+)登记即被 v13/v14 同时选用,组合流程零改动;Runtime lock 形态(Tally Go → runtimes/<rt> → 外部进程)不受影响。
3. **错误面(有意、cosmetic)**:未知 runtime 消息由注册表自述支持集(排序键),v13 为 `v13: 未知 runtime %q(支持 codex|fake)`(原 `支持 fake|codex`,仅键序差别);v14 从复用 v13 前缀变为自身前缀 `v14: 未知 runtime …`。拒绝语义不变(执行前 fail-closed);测试只钉非 nil,无消息字面断言。
4. **v13 行为等价性**:builder 内构造式与原 switch 逐字相同(`fake.New(fake.Config{})` / `codex.New(codex.Config{BinPath: cfg.CodexBin})`);空名缺省 fake 不变;held-out 前提、worktree、验证、事实面全部原路径。

## 4. Tests / commands executed(exact)

| 命令 | 结果 |
|---|---|
| `go build ./cmd/... ./internal/agentruntime/... ./runtimes/...` | PASS |
| `go vet ./cmd/worker/...` | PASS |
| 基线(改动前)`rtk go test ./internal/agentruntime/... ./runtimes/fake/... ./runtimes/codex/... -count=1` | PASS 102(L3 参考测试集:protocol_test / fake runtime_test / codex runtime_test) |
| 基线 `rtk go test ./test/arch/... -count=1` | PASS 100 |
| 基线 `rtk go test ./cmd/worker/... -count=1` | PASS 45 |
| `rtk go test ./cmd/worker/... -count=1` | PASS 49(基线 45 + 新增 4) |
| `rtk go test ./internal/agentruntime/... ./runtimes/fake/... ./runtimes/codex/... ./test/arch/... -count=1` | PASS 202 |
| `DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres rtk go test ./cmd/worker/... -count=1` | PASS 53(共享开发实例;4 个 DB 组合回归从 SKIP 转 PASS) |
| 终验 `DATABASE_URL=… rtk go test ./cmd/worker/... ./internal/agentruntime/... -count=1` | PASS 70 |
| 终验 `rtk go test ./test/arch/... -count=1` | PASS 100(含 `v141_no_bypass_gate_test.go`:LoopConfig 字面量零触碰,`cmd/worker/v14.go` 组合仍绑定 Authority,编译权威先于 runtime 选择/会话的次序未动) |
| 终验 `rtk go test ./runtimes/fake/... ./runtimes/codex/... -count=1` | PASS 85 |
| `gofmt -l`(触碰文件) | 仅 v13.go/v13_test.go 因**基线已有**的 worktree CRLF 被标记(见 §5);对两文件做去 CR 副本后 `gofmt -l` 均 clean,即内容零格式漂移;新文件 clean |
| `git diff --check` | clean |

不跑 repo-wide sweeps(orchestrator post-commit 负责)。

## 5. Baseline failures observed

None(测试全绿)。两事项为基线既有条件,非本任务引入:

- **worktree CRLF**:`cmd/worker/v13.go`、`v13_test.go` 在 worktree 里是 CRLF(`git ls-files --eol`:`i/lf w/crlf attr/text=auto eol=lf`;未触碰的 `main.go` 与 P2-01 lane 的 `base.tmpl` 同状态,`core.autocrlf=true` 先于 checkout)。按 §16 未做行尾归一 pass —— 提交面 diff 只含本任务内容 hunk(git 经 attributes 归一,`git diff` 实测仅内容 hunk)。
- **v13_test.go import 顺序漂移**(changeset 先于 candidateverify):基线已有;因文件属本任务触碰范围,做了两行互换修复(非行尾、非顺手扩大面)。

## 6. Remaining risks

- 注册表实例是 cmd/worker 包级单例;登记只发生在构造时,测试一律用 `newRuntimeRegistry()` 新实例,无跨测试污染。DSH 接线任务(P4-04+)须在 `newRuntimeRegistry` 登记其 builder(缝已在文件注释中写明),不得另开名字分支点。
- 错误消息键序/前缀变化(§3.3)对任何按字面匹配该消息的外部消费者是可见变化;仓内无此类断言(实测)。
- `docs/architecture/CURRENT_CODEBASE.md` / `CURRENT_IMPLEMENTATION_STATUS.md` 的对应行刷新:docs 不在本任务 Allowed Paths,生产架构事实未变(仍是同一 Runtime 协议、同一组合根),留待 docs 拥有任务(与 P1-08 同判例)。

## 7. Decision Locks

Not bypassed, none modified。DL-02(无第二 runtime 协议:工厂只产出 `agentruntime.Runtime`)、DL-03(builder 不携带 Git/验证/验收权威)、DL-16(v14 仍一 attempt 一 runtime 会话,LoopConfig/Authority 未动)、AGENTS.md §3 Runtime lock(适配器选择仍在组合边界;DSH 扩展路径保持 out-of-process 形态)。DECISION_LOCKS.md、PRD、PATH_OWNERSHIP、共享热点 main.go 零触碰。跨 lane 噪音(internal/web/**、web/templates/base.tmpl 的 P2-01 改动)未触碰。

## 8. Completion report(task contract format)

1. **status:** DONE
2. **changed files:** §2(runtimefactory.go/runtimefactory_test.go 新增;v13.go、v14.go、v13_test.go、v14_test.go 修改;本报告)
3. **semantic changes:** §3(名字分支收拢为组合边界注册表;无新接口;错误面 cosmetic 变化;行为等价)
4. **tests/commands:** §4(arch 100 PASS、worker 49/53 PASS、agentruntime+fake+codex 202/85 PASS、vet/build/diff-check PASS)
5. **baseline failures:** none(§5 两项基线既有条件已说明)
6. **remaining risks:** §6
7. **Decision Lock touched:** none(§7)
8. **commit:** 未提交(orchestrator 统一 commit;工作树含其他 lane 改动,不代提交)

## 9. L3 completion evidence(Reference Pack 要求)

- **exact source/tests read:** 合同指定三测试(`internal/agentruntime/protocol_test.go`、`runtimes/fake/runtime_test.go` 存在性经基线运行 102 PASS 覆盖,`runtimes/codex/runtime_test.go` 含 fakeCodexBin 替身模式);本地现状:`cmd/worker/{main,v13,v14}.go` 全文、`v13_test.go` 全文、`v14_test.go` 结构、`internal/agentruntime/protocol.go`、`runtimes/fake/runtime.go`、`runtimes/codex/runtime.go`(Config/New/Descriptor/Snapshot/ensureBinary)与 `internal/workspace/manager.go`(workspace.New 无根目录副作用,负例断言依据);`DECISION_LOCKS.md` 全表 + BI、`PRD_AMENDMENTS.md`(A-01:不新增平行抽象)、`SOURCE_BASELINE.md`;守卫对照 `test/arch/v14_runtime_boundary_test.go`、`v141_no_bypass_gate_test.go`(经 `git grep LoopConfig` 确认字面量集合未变);P1-08 报告(结构/守卫判例)。
- **external reference used:** none(EXT-CODEX-MULTI / EXT-CODEX-CONFIG 为 Codex 子代理/model 路由参考,与本任务的组合缝改动无关,未使用)。
- **why the chosen seam is the narrowest valid seam:** 唯一要替换的硬编码分支就是 v13.go 的 `v13Runtime` switch(v14 复用它);把 name→builder 收进同包注册表后,v13/v14 调用点各改一行,内核零改动、`agentruntime` 协议包零改动(不能放那里:协议包不得反向依赖 runtimes/*,且 codex 构造需要 worker 配置面 CodexBin)、main.go 零改动(工厂是包级实例,无需穿线)。产出值保持 `agentruntime.Runtime`,不发明任何新接口。
- **negative tests added:** 未知 runtime 组合级拒绝(零 run 事实 + 零 worktree 根,`TestRuntimeFactoryRejectsUnknownBeforeExecution`);未知名字在注册表层拒绝且错误自述支持集(`TestV13RuntimeSelectionIsConfiguration`);CodexBin 丢失配置的可区分失败(`TestRuntimeFactoryPlumbsCodexBin`)。
- **known uncertainty / version sensitivity:** 无版本敏感面(工厂只搬运既有构造式,不触碰 codex/fake 内部);注册表 `register` 为「后登记覆盖」语义,依赖「生产登记点唯一」这一约定(已在注释中钉住)。
- **Decision Locks not bypassed:** 确认(§7)。
