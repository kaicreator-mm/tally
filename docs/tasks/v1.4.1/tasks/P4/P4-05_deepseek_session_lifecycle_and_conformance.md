# P4-05 — DeepSeek session lifecycle and conformance

- **Phase:** P4
- **Kind:** implementation
- **Wave:** W28
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P4-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 生命周期/一致性矩阵补全（生产代码零改动 —— P4-04 已交付全部语义;仅测试延伸,路径允许夹具同源）: `TestP405ContextTimeoutLeavesNoOrphan` —— ctx 时限到期触发 Cancel 路径收敛 `RunCancelled`,launcher 子进程(真实进程树)经整树终止后 `waitForGone` 取证**零孤儿残留**(AC;R-1 kill-tree + WaitDelay 的联合效果证明); `TestP405NoOwnerRepoMutation` —— owner 代理目录在 Execute/Close 全生命周期逐字节零变更,替身观测 cwd 钉定 workspace,per-session DSH_HOME 随 Close 删除(隔离零跨运行残留)(AC)。类型化映射已由既有矩阵钉死(崩溃→RunFailed、取消→RunCancelled、成功+无回答→RunFailed fail-safe、Close→RunCancelled)。被拒替代方案:在 test/e2e 复制矩阵(夹具/替身助手第二事实源);模拟执行结局(DL-10 禁)。
- 验证:`-run 'TestP405'` PASS ×2(确定性);`./runtimes/dsh/ -count=1` 9.6s 全包(36 测试;既有崩溃/取消/关闭/13 场景 conformance/能力诚实/身份闸/隔离断言不变绿);arch/build/vet/gofmt/`git diff --check` 清洁。一轮夹具缺失(替身 cwd marker)当轮修复,非基线失败。
- PRODUCTION_REACHABILITY: PASS(矩阵直接驱动生产 dsh.Runtime/Session 公共面 + 真实进程树)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P4-05_deepseek_session_lifecycle_and_conformance.md`。


## Goal

Make DSH process/session lifecycle obey Tally cancellation/result/cleanup semantics.


## Allowed / primary paths

- `runtimes/dsh/**`
- `test/**`


## Implementation requirements

1. cwd is disposable workspace.
2. Cancel/Close leaves no managed child process.
3. Failures map to typed runtime result/error semantics.


## Acceptance criteria

- [ ] No orphan process in cancellation/timeout tests.
- [ ] No owner-repo mutation outside assigned workspace.


## Verification

- Lifecycle/conformance test matrix.

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
- **Reason:** Process lifecycle, cancellation and cleanup across Go↔Node/CLI boundaries are failure-prone and security-sensitive.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: agentruntime cancellation/close tests; Codex runtime process cancellation/close tests
2. **Contract / Interface:** Interface: Execute/Cancel/Result/Close and terminal event stream semantics
3. **Core Implementation:** Core: DSH subprocess/session adapter plus `os/exec` CommandContext/process-group strategy
4. **Failure Handling:** Failure: timeout/cancel/close must leave no managed child; partial output is not a successful claim

## External reference pack
- **EXT-DSH-SAFETY — DeepSeek Harness SAFETY**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md  
  License/version: MIT repository; master; developer preview.  
  Use: Hard constraint: do not treat DSH as sole security control; least privilege and disposable isolation.
- **EXT-DSH-SUBAGENT — DeepSeek Harness subagent seam**  
  https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/subagent.md  
  License/version: MIT; master; pin commit.  
  Use: Named provider registry, capability preflight, child lifecycle and result semantics.
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
