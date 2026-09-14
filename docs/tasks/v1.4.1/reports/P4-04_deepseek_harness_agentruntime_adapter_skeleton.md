# P4-04 — DeepSeek Harness agentruntime adapter skeleton

- **执行者**: general-purpose 子智能体（原始实现）+ 编排者主会话（收尾：崩溃路径缺陷修复 + 本报告；原因见 §6）
- **日期**: 2026-09-13
- **基线**: master `d7c69e6`（P6-02 之后）

## 1. L3 证据来源（阅读顺序与判读）

1. 本地契约 `docs/tasks/v1.4.1/tasks/P4/P4-04_deepseek_harness_agentruntime_adapter_skeleton.md`（Allowed paths: `runtimes/dsh/**` 仅此一项；要求 1「无 Git/verifier/acceptance/policy 权属」、要求 2「保守映射身份与受支持的稳定能力」）。
2. `AGENTS.md` Runtime lock（`internal/agentruntime.Runtime/Session` 是唯一核心协议；`Tally Go → runtimes/dsh → external DSH process`；不嵌入 agent loop、不做对等 ExecutionBackend、runtime 产出永非 Verification/Acceptance）。
3. P4-03 spike 裁决与绑定条件：`reports/P4-03_deepseek_harness_integration_spike.md` + `spikes/P4-03_dsh_protocol_spike.md`（全部协议事实以 spike 活体探针观测为准：`--profile headless "<task>"` 一次性执行、stdout=最终回答、exit 0/1/143、cwd=进程 cwd、keyless `MISSING_CREDENTIAL`、无模型旗标、`--patch` 可重复最高优先级、DSH_HOME 默认 `~/.dsh` 跨运行持久、遥测默认 FEEDBACK_ONLY 外送 OTLP）。
4. 现行源码模式参照：`runtimes/codex/**`（进程外适配器先例：每 Execute 一进程、cwd 钉 workspace、env 认证、kill 即取消、能力诚实声明）、`internal/agentruntime/**`（CheckStart 能力协商、IdentityReporter、CapabilitySet）、`internal/command.CommandSpec`（结构化 argv）。

外部仓库未读（pack 未要求）；spike 之外未新增任何 CLI 表面假设——**未观测的旗标/表面一律不使用、不解释**（runtime.go/process.go/isolation.go/patch.go 文件头均载明此纪律）。

## 2. 接缝判定

- **DshRuntime 实现 `agentruntime.Runtime`**（`New(Config) *Runtime`；`Descriptor()`= name `dsh` / adapter version `1`；`Snapshot()` 建立 CLI 身份观察 + 钉版闸；`Start()` 经统一 `agentruntime.CheckStart` 能力协商）。无任何对等执行协议、无 agent loop 内嵌、无 Git/验证/验收/策略权属（契约要求 1）。
- **一次 Execute = 一个 headless 进程**（spike 实证的一次性形态；Start 不 spawn 进程，只做配置校验、二进制解析、钉版身份闸、per-session home 建立）。与 CodexRuntime 的 `codex exec` 模式 1:1 同构。
- **能力映射 = 空集起步**（`Capabilities()` 返回零能力 `CapabilitySet`）：headless 不暴露 events/resume/fork/structured-output/usage/实际模型观测（spike R-3），因此 `StartRequest.ResumeOf`/`ModelProfile` 在协商处被 `*CapabilityError` 拒绝——未声明即拒绝是能力协商的契约方向。`CapSessionIsolation` 仅当 `Config.SessionIsolationVerified=true`（断言共享隔离金丝雀已对真实 dsh + 本隔离实现运行并通过）才声明——per-session home 隔离本身是无条件生产行为，开关只控制声明真伪（与 codex 同一证据门纪律）。
- **身份 = 只报观察**（identity.go，DL-15）：`RuntimeVersion` 来自真实 `dsh --version` 进程观察（Runtime 级缓存）；`RuntimeName`/`Model`/`SessionID` 恒空——CLI 不自报产品名、headless 不报实际模型、无协议层会话句柄，期望配置绝不回填观察字段（requested ≠ observed 永不相等）。

## 3. Spike 绑定条件 → 实现映射

| 绑定条件（P4-03 裁决） | 实现 | 位置 |
| --- | --- | --- |
| 钉死 `@deepseek-ai/dsh@0.1.5-rc.1` + 运行时 `--version` 身份探针 | `PinnedDSHVersion` 常量；`Snapshot`/`Start` 走 `ensureCLIIdentity`（`--version` 现场观察，缓存跨会话）+ `checkVersionPin`（不符即拒绝会话，fail-closed） | runtime.go `PinnedDSHVersion`/`ensureCLIIdentity`/`checkVersionPin` |
| per-run `DSH_HOME` 隔离（默认 `~/.dsh` 跨运行残留会话态） | 每会话全新临时 home（`os.MkdirTemp`），`Close` 删除；无任何移交（认证走 env，与 home 状态分离）；`DSH_HOME` 强制覆盖（env last-wins 压倒继承与 Config.Env） | isolation.go `newSessionHome`/`sessionSpawnEnv` |
| 强制 `DSH_TELEMETRY_MODE=DISABLED` | `sessionSpawnEnv` 无条件写入（默认 FEEDBACK_ONLY 外送 OTLP 被压制） | isolation.go `telemetryDisabled` |
| kill-tree 加固（R-1） | Windows：`taskkill /PID <pid> /T /F`（文档化整树终止；选型注记：Job Objects 需 `golang.org/x/sys/windows`，在测试架构依赖白名单 INV-6 之外，taskkill 为等价设施且零新依赖）；POSIX：`Setpgid` + `kill(-pgid, SIGKILL)`。触发点：ctx 取消（`cmd.Cancel`→`requestCancelRun`→`killTree`）、显式 `Cancel`、`Close`、spawn 前 pre-cancel 竞态 | process_windows.go / process_unix.go / runtime.go `requestCancelRun` |
| 诚实能力集 | 空能力集 + 协商拒绝；usage 恒零值（宁缺毋假，INV-AUTH-01）；空能力下 isolation 金丝雀按未声明门控不运行（conformance `capability_truth` 断言） | runtime.go `Capabilities`；process.go `finishRun` |
| 模型仅经生成的 `--patch` YAML overlay | `renderPatchOverlay`：类型化 Config 字段封闭渲染（仅 provider/model，锚定 spike 观测的 `agent-default-model` 条目 id）；值字符集白名单（拒绝可疑取值，不做转义器）；零值 = 零文件零旗标的逐字节朴素调用 | patch.go |
| 绝不解析 `session.v3.jsonl.zstd` | 代码零出现该文件面；唯一输出契约是 headless stdout 最终回答 | 全包 |

## 4. 崩溃路径缺陷（主会话收尾修复）

**症状**（子智能体中断后由编排者实测）：
```
--- FAIL: TestProcessCrashHandledAndSessionResponsive (22.30s)
    runtime_test.go:854: Execute 未在时限内收敛: context deadline exceeded
    testing.go:1617: TempDir RemoveAll cleanup: ... being used by another process.
--- FAIL: TestDSHRuntimeConformance/crash (60.37s)
    harness.go:261: 崩溃后 Execute 未在时限内收敛(挂死): context deadline exceeded
```

**根因**：`Config.WaitDelay`（默认 `defaultWaitDelay = 5s`）、`waitDelay()` 访问器与 `classifyRun` 的 `exec.ErrWaitDelay` 分类分支均已存在，但 **`execTurn` 从未把 `cmd.WaitDelay` 赋给 `exec.Cmd`**——注释宣称的兜底是断线的。崩溃场景（外部单进程 kill launcher、孤儿子进程持有 stdout/stderr 写端）下 `cmd.Wait` 在管道 EOF 上无界阻塞（`WaitDelay=0` 语义），Execute 挂到测试截止；孤儿同时持有临时目录内文件，导致 `t.TempDir` 清理失败（下游症状）。

**修复**：`process.go` `execTurn` 增加一行 `cmd.WaitDelay = s.runtime.cfg.waitDelay()`（连同注释归位）。无测试改动——两条测试的收敛/会话存活断言原样保留，测试自身的孤儿清理断言（`waitForGone`）继续成立。

**证据**：`TestProcessCrashHandledAndSessionResponsive` 22.30s FAIL → **3.09s PASS**；`TestDSHRuntimeConformance/crash` 60.37s 挂死 → **2.33s PASS**；全套件两遍 `9.152s` / `9.195s` 全绿（确定性）。

## 5. 验证

- `go test ./runtimes/dsh/ -count=1` ×2：PASS（9.152s / 9.195s；34 测试断言：单测 + 全 13 场景 conformance（start/cwd/execute/result/events/cancel/crash/workspace/usage/capability_truth）+ 能力诚实/取消/关闭/身份/隔离负路径）。测试经 `testdata/fakedsh`（自带最小假 dsh launcher：记录调用、模拟 headless stdout/exit 契约、child-spawn 长驻模式）确定性、无钥运行。
- `go vet ./runtimes/dsh/`：PASS。`gofmt -l runtimes/dsh/`：清洁（全部新文件 LF）。`git diff --check`：清洁。
- `go build ./runtimes/dsh/`：PASS。
- 全仓扫套：编排者提交后统一运行（本任务只跑聚焦面）。

## 6. 执行过程说明（诚实记录）

原始实现由 general-purpose 子智能体完成（2294 行，含 1105 行测试）；其会话在最后阶段因平台模型请求故障中断（连续多次重派同因失败）。编排者主会话接管收尾：实测定位上述单点缺陷（一行接线缺失），修复后全套件两遍验证通过，并基于逐文件阅读撰写本报告。生产注册（`workerRuntimes` 注册 builder）归后续 P4 任务，本任务未触碰 `cmd/worker`。

## 7. 风险

- **`--patch` 合并语义未做 live 验证**（spike §5 已标注）：overlay 按 id 锚定默认配置 dump 观测的条目，patch 合并优先级未在真实 dsh 上验证——live 表面验证由 env-gated 测试与 P4-04+ 的 conformance canary 承担，版本升级必须重验（patch.go 文件头载明）。
- **`taskkill /T /F` 按调用时刻父子关系枚举**：若孤儿在枚举后由存活子进程再产生新孙进程，不保证覆盖；`WaitDelay` 兜底保证 Execute 收敛不受影响，残留孤儿由会话 `Close` 的 best-effort 终止 + 测试取证面覆盖。
- **Windows teardown 依赖外部 taskkill 可用性**（系统 PATH）；缺失时 killTree 返回显式错误（不静默），`WaitDelay` 仍保证收敛。
- **身份观察仅覆盖版本**：`--version` 输出形态变化（未来版本）可能使解析产生空值——空值即「未观察」合法态，fail-safe 不伪造（identity.go 文件头）。
- **DSH developer-preview 易变（R-2）**：钉版闸使任何上游行为漂移以 Start 显式失败显形，不静默降级。

## 8. NOTES

- DL-02/DL-03/DL-15/DL-16 与 AGENTS.md §3 Runtime lock 全程遵守：适配器只产出 `agentruntime.Runtime/Session`；无 Git 持久权限；期望/观察身份分离；认证仅经 env（`DEEPSEEK_API_KEY` 由环境注入，未硬编码、未记录）。
- `PRODUCTION_REACHABILITY`: N/A（reason：`workerRuntimes` 注册属后续 P4 组装任务；本骨架 + conformance 即交付面）。
- 关联报告：`reports/P4-03_deepseek_harness_integration_spike.md`（协议事实与 GO-WITH-CONDITIONS 条件）。
