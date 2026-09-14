# P2-01 — Human-centered navigation projection

- **Phase:** P2
- **Kind:** implementation
- **Wave:** W15
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P1-08
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — base.tmpl 新增「研发」导航组（10 行纯插入，零既有行改动）：发起→`/v14/launch`、工作→`/v13/tasks`、需人工→`/v14/attention`；模板注释记录 DL-19 投影纪律及 Runs/Rules 条目延迟至其属主任务（P2-03/P3）的原因 —— 零假占位。Active key 修正两处（`v14`→`v14-launch`、`v14`→`v14-attention`）使高亮真实；任务/变更页维持现状（P2-02 属主）。`server.go`（§10 热点）零触碰。
- AC：现有路由有效、导航只映射真实读模型数据 — PASS，可执行强制：`TestNavLinksResolveToRealRoutes`（每个渲染 nav href 经真实 router GET，404/405/5xx 即失败）+ `TestNavHrefsAreKnownRoutes`；每条目审计到真实读模型支撑路由（launch：结构化意图接入 + 事实存储任务列表；v13/tasks：TaskContract 视图 + taskstate 派生事实；v14/attention：PRD §26.4 Attention 投影）。无支撑路由的 Runs/Rules 不设条目；需人工页标注「需人工」而非 "Reviews"，不暗示无支撑能力。
- Verification：`go test ./internal/web/...` PASS（web 0.295s + handlers 17.192s，7 新导航测试全绿）；`./test/arch/...` PASS；DB-gated 全包 `DATABASE_URL=… go test ./internal/web/handlers/ -count=1` PASS（116.1s）；repo-wide `go build ./...` PASS；vet/gofmt/`git diff --check` 清洁（base.tmpl 零行尾churn，仅 10 行内容）。PRODUCTION_REACHABILITY: PASS（`web.NewRouter` 生产组合根全 handlers 渲染，HTTP 端到端验证含 DB-gated 套件）。Full evidence: `reports/P2-01_human_centered_navigation_projection.md`。Risks：中文人本标签 —— P2-02/03/04 属主可更名/补 Runs 条目（扩展点已在模板注释标记）。


## Goal

Add familiar engineering navigation without changing domain state machine.


## Allowed / primary paths

- `internal/web/**`
- `web/templates/base.tmpl`
- `web/static/**`


## Implementation requirements

1. Work/Runs/Reviews/Rules routes only when backed by real data.
2. Do not introduce fake feature placeholders.


## Acceptance criteria

- [ ] Existing routes remain valid; navigation maps to existing/read-model data.


## Verification

- Routing/template tests.

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
1. **Tests:** Tests: existing handler route/template tests and `v14_task_test.go`
2. **Contract / Interface:** Interface: current task/run/attention read models; no lifecycle writes
3. **Core Implementation:** Core: `internal/web/handlers/server.go`, `views.go`, `web/templates/base.tmpl`
4. **Failure Handling:** Failure: navigation item must not imply unavailable capability or introduce status mutation

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
