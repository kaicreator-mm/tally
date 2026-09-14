# P4-06 — DeepSeek configuration and canary policy

- **Phase:** P4
- **Kind:** implementation
- **Wave:** W28
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P4-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — DSH 配置与 canary 策略落地(零 TaskContract 字段,要求 3):**缺省禁用**(要求 1)+ **canary 三闸**(runtime=dsh + TALLY_DSH_CANARY=1 + 显式 TALLY_DSH_BIN 任一缺席在任务执行前类型化失败;canary 关闭时 Codex 缺省面零影响,AC):注册表登记 `dsh` builder(runtimefactory.go),`v13Config` 增 DSHCanary/DSHBin/DSHModel/DSHProvider(env 装载,v13.go §10 声明式最小接缝);`runtimes/dsh` 新增 `Config.Validate()` 预检(权限枚举/overlay token 在最早闸拒绝;钉版身份闸仍在 Start)—— builder 构造期即调用的 **Invalid configuration fails before task execution** 闸(AC);canary/bounded-task 策略文档化 `docs/guides/dsh-canary.md`(要求 2:启用三条件/绑定约束[钉版、隔离、kill-tree、诚实能力、认证]/有界任务边界)。期望模型/provider 仅经 --patch overlay(与实际生效模型两回事)。
- 验证:`go test ./cmd/worker/ -count=1 -run 'TestDSH' -v` PASS(缺省禁用 5 组非法配置全拒 + 显式 canary 可构造 + Codex 缺省面零影响 + 未知 runtime 错误列出登记面);`./cmd/worker/` 26.4s 全包、`./runtimes/dsh/` 9.3s、arch/build/vet/gofmt/`git diff --check` 全 PASS。一轮 BAD MODEL 用例失败暴露 Start 期校验过晚 → 催生 Config.Validate 预检;自引入当轮修复,非基线失败。
- PRODUCTION_REACHABILITY: PASS(builder 即生产 workerRuntimes 注册面;env 驱动,缺省禁用即生产现状)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P4-06_deepseek_configuration_and_canary_policy.md`。


## Goal

Configure DSH provider/model/delegation options at adapter/composition level without destabilizing core capability vocabulary.


## Allowed / primary paths

- `runtimes/dsh/**`
- `cmd/worker/**`
- `docs/**`


## Implementation requirements

1. Disabled by default unless explicitly configured.
2. Canary/bounded-task policy documented.
3. Do not add speculative provider-specific fields to core TaskContract.


## Acceptance criteria

- [ ] Invalid configuration fails before task execution.
- [ ] Current Codex default unaffected.


## Verification

- Config tests.

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
1. **Tests:** Tests: agentruntime capability negotiation tests + DSH adapter config tests
2. **Contract / Interface:** Interface: adapter/composition config only; core TaskContract stays provider-neutral
3. **Core Implementation:** Core: explicit canary enablement, provider/model configuration in DSH-owned extension config
4. **Failure Handling:** Failure: invalid/unavailable route fails before execution; Codex default remains unchanged

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
