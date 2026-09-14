# P6-02 — Remaining existing Outcome event capture

- **Phase:** P6
- **Kind:** implementation
- **Wave:** W36
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P6-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 四类剩余事件的类型化捕获面落地 `internal/outcome/capture.go`：`RevertFact`/`HotfixFact`/`RegressionFact`/`HumanCorrectionFact`（各自 `Validate`：至少一个非空绑定 + 显式 `RecordedAt`）+ `*Store` 四个捕获方法（`RecordReverted`/`RecordHotfixed`/`RecordRegressionFound`/`RecordHumanCorrection`），共享重放安全追加路径（kind+坐标+时刻精确重放抑制，经 `Equal`；不同时刻恒追加——`RecordedAt` 是发生坐标，两次不同时刻的回归 = 两条事实），每 kind 附权威生产者文档与**缺席的负面证据**。
- **接线裁定：N/A(显式裁定，非遗漏)** —— 契约要求 1「源事件不存在处禁止伪造自动检测」；四类在 v1.4.1 均无生产源事件（负面证据见报告 §1：deploy 回滚无 task/changeset 身份、`gitwriter.revert` 是事务清理、无 hotfix 路径、verification 的 regression gate 是验收前门、`human_approval` 是独立领域的方向裁决证据）。四类生产者归后续控制面/验证任务；**P6-03（读侧）与未来生产者任务消费此面**。`accepted` 在结构上被排除出捕获面（`TestCaptureAPINeverExposesAccepted`）—— P6-01 的 Engine 验收接缝仍是唯一验收事实写入者。
- AC：每个受支持事件均有文档化权威生产者 — PASS（每事实类型具名生产者角色 + 声明 v1.4.1 自动 hook 缺席）。Verification：`go test ./internal/outcome/` DB-backed PASS（54.1s，`TestCapture` 6/6）；`./internal/closedloop/`（未动零回归）PASS；`./test/arch/`（DL-12 五类枚举守卫 + P1-08 no-bypass）PASS；vet/build/gofmt/`git diff --check` 清洁，新文件全 LF。**DL-12 五类枚举在基线已存在、本任务未扩张**（orchestrator 核实：types.go 的 KindReverted 等为基线事实）。
- Risks：**文档化生产者 vs 已接线生产者的 gap** —— 生产调用方按设计尚不存在；生产者落地时必须使用这些 API，绝不从 deploy/workflow 事实推断事件（P6-08 终局验证需知悉此 gap）。精确重放幂等为 check-then-append（单写者纪律下正确，无 DB 唯一索引，migrations 在路径外——与 P6-01 同类已裁定限制）。Full evidence: `reports/P6-02_remaining_existing_outcome_event_capture.md`。


## Goal

Wire existing `reverted`, `hotfixed`, `regression_found`, `human_correction` events to real hooks or explicit capture APIs.


## Allowed / primary paths

- `internal/outcome/**`
- `internal/**`
- `cmd/**`


## Implementation requirements

1. Do not fake automatic detection where source events do not exist.
2. Do not add rejected/escalated/integration_failed kinds.


## Acceptance criteria

- [ ] Each supported event has a documented authoritative producer.


## Verification

- Event producer tests.

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
1. **Tests:** Tests: `outcome/types_test.go`, `store_test.go`, architecture closed-enum test
2. **Contract / Interface:** Interface: exactly five existing Outcome kinds
3. **Core Implementation:** Core: map only real authoritative producers; expose explicit capture API if automatic hook does not exist
4. **Failure Handling:** Failure: do not infer regression/hotfix/revert from names or generic workflow failure

## External reference pack
- **EXT-POSTGRES-INSERT — PostgreSQL INSERT / ON CONFLICT**  
  https://www.postgresql.org/docs/current/sql-insert.html  
  License/version: PostgreSQL documentation; current PostgreSQL docs.  
  Use: Reference idempotent insert patterns where a unique semantic key is appropriate; do not mutate append-only events.

## Do

- Wire only authoritative producers and reproduce defects before changing trust/verification semantics.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not broaden Outcome semantics or weaken verification to make a failing experiment green.
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
