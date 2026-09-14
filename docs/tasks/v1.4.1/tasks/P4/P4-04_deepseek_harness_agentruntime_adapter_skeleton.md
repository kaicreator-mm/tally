# P4-04 — DeepSeek Harness agentruntime adapter skeleton

- **Phase:** P4
- **Kind:** implementation
- **Wave:** W27
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P4-01, P4-03
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — DSH 适配器骨架落地 `runtimes/dsh/**`（零生产外触碰；`workerRuntimes` 注册归后续组装任务）：DshRuntime 实现 `agentruntime.Runtime`（Descriptor `dsh`/v1；`New(Config)`/`Snapshot`/`Start` 经统一 `CheckStart` 能力协商），**一次 Execute = 一个 `--profile headless` 进程**（spike 实证的一次性形态，与 CodexRuntime `codex exec` 同构）。Spike 绑定条件逐项落地：钉版 `0.1.5-rc.1` + `--version` 现场观察闸（不符 fail-closed 拒绝会话，观察缓存跨会话）；每会话全新临时 `DSH_HOME`（无移交，Close 删除）+ 强制 `DSH_TELEMETRY_MODE=DISABLED` + `DSH_PERMISSION_MODE` 强制（继承 danger-full-access 不得静默升级）；kill-tree 加固（Windows `taskkill /T /F`——零新依赖的文档化整树终止，POSIX `Setpgid`+组 SIGKILL；触发点：ctx 取消/显式 Cancel/Close/pre-cancel 竞态）；**诚实能力集 = 空集**（headless 无 events/resume/fork/structured-output/usage/实际模型观测——`ResumeOf`/`ModelProfile` 在协商处被 `*CapabilityError` 拒绝；usage 恒零值宁缺毋假；`CapSessionIsolation` 仅在隔离金丝雀实证后声明）；期望模型仅经生成的最小 `--patch` YAML overlay（类型化字段封闭渲染 + 值字符集白名单；零值 = 逐字节朴素调用）；`session.v3.jsonl.zstd` 零触碰。身份只报观察（DL-15）：`RuntimeVersion` 来自真实 `--version`；Name/Model/SessionID 恒空不回填。
- **崩溃路径缺陷（收尾修复，根因 + 机制）**：`Config.WaitDelay`/`waitDelay()`/`classifyRun` 的 `ErrWaitDelay` 分支均已存在，但 `execTurn` 从未把 `cmd.WaitDelay` 接上——外部单进程 kill launcher 留下持有输出管道的孤儿子进程时 `cmd.Wait` 无界阻塞，Execute 挂死（TestProcessCrashHandledAndSessionResponsive 22.30s deadline FAIL；conformance crash 60.37s 挂死），孤儿持有临时文件致 t.TempDir 清理失败。修复 = 一行接线 `cmd.WaitDelay = s.runtime.cfg.waitDelay()`；**测试零改动**（收敛/会话存活/孤儿清理断言原样）。证据：22.30s FAIL → 3.09s PASS；crash 子测试 2.33s PASS；全套件两遍 9.152s/9.195s 全绿（34 断言，经 testdata/fakedsh 假 launcher 确定性无钥运行）。
- Verification：`go test ./runtimes/dsh/ -count=1` ×2 PASS；vet/gofmt（全 LF）/`go build`/`git diff --check` 清洁。PRODUCTION_REACHABILITY: N/A(reason: 注册归后续 P4 组装任务)。Risks：`--patch` 合并语义未 live 验证（版本升级必须重验，patch.go 载明）；taskkill 按调用时刻枚举、PATH 缺失时显式错误不静默（WaitDelay 仍保收敛）；身份观察仅版本、输出形态漂移 = 未观察合法态不伪造；R-2 易变性由钉版闸以 Start 显式失败显形。执行过程：原始实现由 general-purpose 子智能体完成，因平台模型请求故障中断后由编排者主会话收尾（修复 + 报告）。Full evidence: `reports/P4-04_deepseek_harness_agentruntime_adapter_skeleton.md`。


## Goal

Implement DSH as an `agentruntime.Runtime/Session` adapter only.


## Allowed / primary paths

- `runtimes/dsh/**`


## Implementation requirements

1. No Git writer, verifier, acceptance or policy ownership.
2. Map runtime identity and supported stable capabilities conservatively.


## Acceptance criteria

- [ ] Adapter compiles against current agentruntime without changing core protocol unless separately approved by ADR.


## Verification

- Contract tests with fake/subprocess stub.

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

- **Execution:** `LOW`
- **Review:** `HIGH`
- **Reason:** The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: `agentruntime/protocol_test.go` conformance style and `runtimes/codex/runtime_test.go` adapter structure
2. **Contract / Interface:** Interface: implement exactly `agentruntime.Runtime/Session`; no new core fields without ADR
3. **Core Implementation:** Core: follow Codex adapter package layout for Descriptor/Capabilities/Start/Session methods
4. **Failure Handling:** Failure: unsupported capability fails at `CheckStart`/adapter preflight rather than silent ignore

## External reference pack
- **EXT-DSH-SAFETY — DeepSeek Harness SAFETY**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md  
  License/version: MIT repository; master; developer preview.  
  Use: Hard constraint: do not treat DSH as sole security control; least privilege and disposable isolation.
- **EXT-DSH-SUBAGENT — DeepSeek Harness subagent seam**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/subagent.md  
  License/version: MIT; master; pin commit.  
  Use: Named provider registry, capability preflight, child lifecycle and result semantics.
- **EXT-DSH-TOOL — DeepSeek Harness tool-subagent implementation**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/subagent/tool-subagent/src/index.ts  
  License/version: MIT; master; pin commit.  
  Use: Current provider/model/reasoning selection surface. Must verify exact installed build because developer-preview behavior has changed.
- **EXT-GO-EXEC — Go os/exec**  
  https://pkg.go.dev/os/exec  
  License/version: BSD-3-Clause (Go project); Go standard library.  
  Use: Subprocess context cancellation, Wait/Process lifecycle and command execution semantics.

## Do

- Implement only the runtime adapter seam; Tally keeps workspace, Git, verification and acceptance.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not add a second runtime lifecycle protocol or grant a runtime durable Git/acceptance authority.
- Do not copy external code wholesale. Extract the pattern and adapt it to Tally's existing interfaces and dependency whitelist.
- Do not add a new dependency unless `test/arch/deps_test.go` and the dependency policy are intentionally updated by an approved architecture change.

## Low-cost model execution recipe

1. Read the exact tests listed above.
2. Read the interface/types used by those tests.
3. Read only the nearest implementation seam, not the whole repository.
4. Implement one bounded change.
5. Run focused tests.
6. Run relevant `test/arch` guards.
7. If expected source facts differ from this pack, stop with `BLOCKED_BY_SOURCE_FACT`; do not redesign the architecture.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.
