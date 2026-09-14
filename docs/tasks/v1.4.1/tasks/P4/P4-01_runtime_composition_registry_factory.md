# P4-01 — Runtime composition registry/factory

- **Phase:** P4
- **Kind:** implementation
- **Wave:** W25
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P1-08
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 组合边界小注册表落地 `cmd/worker/runtimefactory.go`：`runtimeBuilder`（v13Config → agentruntime.Runtime）、`runtimeRegistry`（name → builder，缺省 fake）、生产实例 `workerRuntimes` 注册 fake/codex（构造与旧 switch 字节等价）、`register` 接缝、fail-closed `build`。`v13.go` 硬编码 `v13Runtime` switch 删除，step 2 改调 `workerRuntimes.build(cfg)`（**§10 热点触碰已声明**：fake/codex import 移出，行为等价）；`v14.go` step 3 同工厂（错误前缀 `v14:`）；`main.go` 零改动（包级工厂无需穿线）；`LoopConfig/Authority` 零触碰；无新执行核心接口 —— 仅组合边界 name→adapter 选择，能力逻辑（PRD §10.3）未动，runtime lock 完好（工厂只产出 `agentruntime.Runtime`）。
- AC：fake/codex 经工厂行为同一 — PASS（描述符等价 + CodexBin 管道证明测试 + 全部既有 v13/v14 组合回归不变绿）。`go test ./cmd/worker/... -count=1` PASS（49 = 45 基线 + 4 新；DATABASE_URL 下 53 含 4 个 DB 组合回归实跑）；`./test/arch/...` PASS（100，含 v141 no-bypass 守卫 —— 全仓 LoopConfig 字面量仍绑 Authority）；agentruntime + runtimes/fake + runtimes/codex PASS；repo build/vet/gofmt（新文件清洁；v13.go/v13_test.go 仅预存 worktree CRLF 条件、未做行尾规整pass per §16）/`git diff --check` 清洁。PRODUCTION_REACHABILITY: PASS（两类 worker 作业的生产选择路径；拒绝先于执行：零 run facts、零 worktree）。
- Risks：DSH 接线（P4-04+）必须在 `newRuntimeRegistry` 注册其 builder（文档化接缝）—— 不得在别处新增 name 分支点；未知 runtime 错误文案现由注册表键派生（cosmetic）。Full evidence: `reports/P4-01_runtime_composition_registry_factory.md`。


## Goal

Replace hard-coded fake/codex composition branching with a small runtime factory/registry while preserving `agentruntime.Runtime`.


## Allowed / primary paths

- `cmd/worker/**`
- `internal/agentruntime/**`


## Implementation requirements

1. No new execution core interface.
2. Runtime name only selects adapter at composition boundary.
3. Capability logic remains runtime-neutral.


## Acceptance criteria

- [ ] Existing fake/codex paths behave identically through factory.


## Verification

- Worker composition regression tests.

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
1. **Tests:** Tests: `internal/agentruntime/protocol_test.go`, `runtimes/fake/runtime_test.go`, `runtimes/codex/runtime_test.go`
2. **Contract / Interface:** Interface: existing `agentruntime.Runtime` only
3. **Core Implementation:** Core: worker composition selects runtime implementation; no new lifecycle protocol
4. **Failure Handling:** Failure: unknown runtime/config rejected before task execution

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
