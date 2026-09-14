# P2-03 — Runs projection

- **Phase:** P2
- **Kind:** implementation
- **Wave:** W16
- **Complexity:** S
- **Execution model:** LOW
- **Depends on:** P2-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — `V13Runs` 处理器 + 人本 Runs 页落地：运行行为 `runstore.Run` 事实的逐字投影（状态取自行内 run-control 事实、迭代数、用量 nil 保留为 未上报、escalated-from 链、失败原因逐字、仅 `ResultPatchHash` 设置时给候选链接）；期望 runtime/model 标注 期望坐标，runtime 自报 `observed_identity` 只读解码为 实际…(runtime 自报)，缺失为 实际身份未观察，不可解析快照为显式注记 —— 期望 vs 观测恒不混写；确定性最新优先排序；逐任务读失败显式 flash。`server.go`（§10 热点）声明式最小接缝：一行路由 `r.Get("/v13/runs", s.V13Runs)` + 一个注释词。base.tmpl 研发组新增 运行→`/v13/runs`（key `v13-runs`），闭合 P2-01 预留扩展点（Rules 仍归 P3）；既有条目字节不变。
- AC：运行状态派生自既有 workflow/runtime 事实、无重复运行状态 — PASS（状态经既有 `CodegenRunsForTask` 路径逐字渲染，零新状态词汇（颜色纯展示）、零状态字段/setter/写路由/新 SQL；负测试证明未播种状态（`>cancelled<`）与伪造候选/失败原因永不渲染）。实现期源事实修正：`runstore.Run` 无 profile 列 —— 字段移除而非合成。
- Verification：DB-gated 全 handlers 包 PASS（153.4s）；`TestNav` 5/5 + handlers nav 3/3（含真实 router GET `/v13/runs`）+ `TestV13Runs` 2/2（真实 run fixtures）；workflow+arch 141 PASS；`go build ./...` exit 0；gofmt（触及 LF 文件清洁）/`git diff --check` 清洁。PRODUCTION_REACHABILITY: PASS（路由+导航条目在生产组合根，HTTP 实测）。Risks：HEAD 快照外的任务运行不列出（任务域读路径是唯一枚举面，不碰 `internal/store` §10 热点；页面文案显式声明）；逐任务读 N+1 形态当前规模可接受（list-all sqlc 查询归 store 属主任务）；escalated 展示色为展示判断（文本逐字）。Full evidence: `reports/P2-03_runs_projection.md`。


## Goal

Expose actual execution/run facts separately from Work/Task state.


## Allowed / primary paths

- `internal/web/handlers/**`
- `web/templates/**`
- `internal/workflow/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Run status is derived from existing workflow/runtime facts; no duplicate run state.


## Verification

- Handler tests with real run fixtures.

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
1. **Tests:** Tests: worker/runstore/workflow tests plus web handler projection tests
2. **Contract / Interface:** Interface: existing run facts / runtime identities / workflow outcomes
3. **Core Implementation:** Core: `internal/workflow`, `internal/runstore`, v1.4 handlers
4. **Failure Handling:** Failure: Runs page is read-only derived projection; do not create a second run lifecycle

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
