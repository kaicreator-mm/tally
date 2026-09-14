# P4-02 — Codex runtime regression through factory

- **Phase:** P4
- **Kind:** implementation
- **Wave:** W26
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P4-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 零生产代码改动的回归证明:P4-01 factory 的 codex 路径补齐**行为级**等价证据。`cmd/worker/runtimefactory_codex_test.go`(新):现场构建最小替身 codex CLI(进程级真实,深 CLI 表面探测仍归 runtimes/codex),① `TestCodexRuntimeThroughFactoryPassesConformance` 对 factory 产物(构造式与生产 builder 同形,只携带 CodexBin)跑完整 runtimeconformance(13 场景:start/cwd/execute/result/events/cancel/crash/workspace/usage/capability_truth + resume/fork/output.structured;isolation canary 按未声明门控不运行);② `TestV13CodexRuntimeThroughFactoryProducesChangeSet` 走生产注册表(workerRuntimes,deps.runtime=nil)跑完 v13 垂直路径:替身产出经 workspace 钉定落入一次性 worktree → 不可变 ChangeSet → visible 验证 verified,`RunBegin.RuntimeName=="codex"` 身份如实入 run 事实,主 worktree 零污染、一次性根清空;③ 负例 `TestV13CodexRuntimeFailedClaimNeverFixesCandidate`:非零退出 → RunFailed claim → needs_review,零 ChangeSet、零最终验证(claim ≠ 事实经 factory 不变)。替身行为开关经 `t.Setenv` 注入(runner 把 spec.Env 追加到继承环境之后),factory 面(只暴露 CodexBin)零改动。runtimes/codex 零触碰,无缺陷发现,无 Git/验证/验收权威移入 runtime。详见 `docs/tasks/v1.4.1/reports/P4-02_codex_runtime_regression_through_factory.md`。


## Goal

Prove adapter refactor does not alter Codex identity, cancellation, workspace, event or ChangeSet behavior.


## Allowed / primary paths

- `runtimes/codex/**`
- `cmd/worker/**`
- `test/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Existing Codex-focused tests remain green; no new Git/acceptance authority moves into runtime.


## Verification

- Codex conformance/regression suite.

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
1. **Tests:** Tests first: `runtimes/codex/runtime_test.go::TestCodexRuntimeConformance`, cancellation/close/structured-output tests
2. **Contract / Interface:** Interface: `agentruntime.Runtime/Session`, Codex Descriptor/Capabilities
3. **Core Implementation:** Core: existing `runtimes/codex/runtime.go` must remain behaviorally unchanged
4. **Failure Handling:** Failure: factory refactor cannot alter cwd, cancellation, result, identity or ChangeSet behavior

## External reference pack
- **EXT-CODEX-MULTI — OpenAI Codex spawn_agent implementation**  
  https://github.com/openai/codex/blob/main/codex-rs/core/src/tools/handlers/multi_agents_spec.rs  
  License/version: Apache-2.0; main; pin inspected commit before implementation.  
  Use: Reference current child model/reasoning surface; do not assume arbitrary provider routing.
- **EXT-CODEX-CONFIG — OpenAI Codex runtime config**  
  https://github.com/openai/codex/blob/main/codex-rs/core/src/config/mod.rs  
  License/version: Apache-2.0; main; pin inspected commit.  
  Use: Reference current provider map, multi-agent enablement, subagent model/thread config.

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
