# P2-02 — Work list and task detail projection

- **Phase:** P2
- **Kind:** implementation
- **Wave:** W16
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P2-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 任务详情页按 PRD §7/§29 人本分区重组（概览/计划/变更/验证/评审/动态），此前展示的每个事实仍展示；新增裁决行 + 四基线维度 + 当前候选覆盖率表 —— 消费既有两条生产读路径：编译权威（`artifacts.LoadBundle → CompileAuthority` 于页面已钉 HEAD hash，P1-03 接缝、DL-04）与当前候选覆盖（`loadV13Change → deriveV13Coverage`），首次 Derive 定位后带 `Snapshot.Compiled` 再派生（构造上纯增量；再派生类型化失败保留首派生 + 显式原因）。Work 列表（v13_tasks）：h1 `P1 任务`→`工作`（PRD §30.1）+ 人本导语 + 每行 TaskID 链接 `/v14/tasks/{task}`（此前 Work→Task 不可达）。Active key `v14`→`v13`（P2-01 移交：任务详情是工作域视图，工作高亮）。`server.go` 零触碰（无路由注册变化）；`internal/taskstate/**` 零改动（P1-06/07 语义冻结，Derive 仍是唯一结论源，DL-19）。
- AC：投影不引入生命周期突变 — PASS（只读、零 setter/状态字段/路由）；compiled/risk/assurance 基线在可用处出现 — PASS（VERIFIED/VERIFIED_LIMITED/NOT_VERIFIED + `compiled:<hash>` + required/achieved assurance + unmatched/unknown 信号 + 四维度；缺 Compiled 时类型化未接入注记，负测试断言禁用字符串零出现）。
- Verification：DB-gated 全 handlers 包 PASS（140.5s）；`TestV14Task|TestV13TaskPage` 7/7（含 3 新：plain VERIFIED / VERIFIED_LIMITED unmatched risk / Compiled-absent 负例）；taskstate/arch 全绿；repo build 仅 runtimes/dsh 在途失败（P4-04 lane 噪声，已按规记录）；gofmt/`git diff --check` 清洁。PRODUCTION_REACHABILITY: PASS（两页面均为登录/CSRF 保护的生产路由）。Risks：陈旧修订候选显式覆盖失配通知 + fail-safe（与 P1-07 变更页同向）；覆盖率表标记在两模板间重复（同一读路径事实）；需人工 rename 归 P2-04。Full evidence: `reports/P2-02_work_list_and_task_detail_projection.md`。


## Goal

Present Task as Overview/Plan/Changes/Verification/Reviews/Activity over existing facts.


## Allowed / primary paths

- `internal/web/handlers/**`
- `web/templates/**`
- `internal/taskstate/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] No lifecycle mutation introduced by projection.
- [ ] Compiled/risk/assurance basis appears where available.


## Verification

- Handler/read-model tests.

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
- **Review:** `LOW`
- **Reason:** The implementation is bounded and deterministic when the supplied tests/contracts/core references are followed; low-cost execution is preferred.
- **Review rule:** LOW review is sufficient if all focused tests and task acceptance criteria pass; escalate only on source/API mismatch.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: `v14_task_test.go`, `v14_change_test.go`, `taskstatus_internal_test.go`
2. **Contract / Interface:** Interface: existing Task/Change/VerifiedBasis projections
3. **Core Implementation:** Core: `v14_task.go`, `v14_change.go`, derived-state helpers
4. **Failure Handling:** Failure: render unknown/limited states explicitly; no duplicate mutable Task state

## External reference pack
- **EXT-GO-HTML — Go html/template**  
  https://pkg.go.dev/html/template  
  License/version: BSD-3-Clause (Go project); Go standard library.  
  Use: Context-aware escaping reference for rendering LLM-originated values as data, never raw HTML.

## Do

- Build read-only projections over existing facts/state; preserve no-manual-status discipline.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not create a second mutable task/run/review state just to simplify UI.
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
