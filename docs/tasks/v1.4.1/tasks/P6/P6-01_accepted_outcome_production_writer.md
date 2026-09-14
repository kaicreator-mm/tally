# P6-01 — Accepted Outcome production writer

- **Phase:** P6
- **Kind:** implementation
- **Wave:** W35
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P1-08
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **接缝判定：写入器挂在 Engine 验收转移内部（AcceptGuard 通过后、`EventVerified` 前）** —— 绑定后 `PhaseAccepted` 仅在 accepted outcome 事实持久追加后才存在；被拒方案：handler 侧验收后写（崩溃窗口静默丢失不可补的 §23 事实；重试铸造新 changeset = 一次验收两事件）与 verifier 侧权威（制造第二验收权威）。写入器只观测/报告验收；裁决 + P1-06 守卫仍是唯一授权（守卫与裁决语义零触碰）。
- 落地：`internal/closedloop/outcome.go`（`AcceptFact`+`Validate`、`AcceptSink` 端口 append-only 幂等契约、`StoreAcceptSink` 幂等适配器 over `internal/outcome`、`ErrInvalidAcceptSink`）；`engine.go` `Config.AcceptSink` 接缝调用（写用 `context.WithoutCancel`；写前取消保持一档 Cancelled 终态，写后取消不再改写终态）；`result.go` 类型化升级理由 `outcome_unavailable`；`wiring.go` `LoopConfig.AcceptSink` 直通；`cmd/worker/v14.go` 生产绑定（fact store 存在即绑 `NewStoreAcceptSink(outcome.New(queries))` —— 生产恒真）+ `v14Deps.accept` 测试接缝。Nil sink = v1.4 测试组合行为不变。
- AC：一次持久验收恰一个 accepted Outcome 事件绑定 task/changeset — PASS（`TestV14AcceptedOutcomePersistsExactlyOnce`：恰 1 行、kind=accepted、绑 `res.Basis.ChangeSet`；真 DB 重放仍 1 行）；重试/重放零重复 — PASS（幂等 `StoreAcceptSink` 单元 + 真 DB）；失败策略显式 fail-safe — PASS（`outcome_unavailable` 类型化升级、无 Basis、零 Router/预算消耗、零行写入；读失败 ≡ 写失败，绝不折叠为"无事发生"）。`go build ./...`、vet、`./internal/closedloop/... ./internal/outcome/...`、DB-backed `./cmd/worker/...`、`./test/arch/...`（P1-08 no-bypass + DL-12 Outcome kind-set 守卫）全 PASS；gofmt/`git diff --check` 清洁。PRODUCTION_REACHABILITY: PASS（组合根无条件绑定）。
- Risks：幂等为 check-then-append 非 DB 强制（0023 无唯一索引；migrations 在路径外，P3-02 保留编号）—— 单写者 job-lease 纪律下正确（一 lease → 一 one-shot Engine → 每验收一次 sink 调用）；如需 DB 保证，确切 DDL（`ON outcome_event (task_id, changeset_id) WHERE event_kind='accepted'` 部分唯一索引）已记录于报告待仲裁，当前非必需。Full evidence: `reports/P6-01_accepted_outcome_production_writer.md`。


## Goal

Append `outcome.KindAccepted` at the correct durable acceptance seam without creating two acceptance authorities.


## Allowed / primary paths

- `internal/outcome/**`
- `internal/closedloop/**`
- `cmd/worker/**`


## Implementation requirements

1. Identify exact transaction/order semantics before code.
2. No duplicate event on retry/replay.
3. Failure policy for audit append is explicit/fail-safe.


## Acceptance criteria

- [ ] One durable accepted task yields one valid accepted Outcome event tied to task/changeset.


## Verification

- Integration tests incl. retry/idempotency/failure.

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
- **Reason:** Must attach append-only Outcome to the exact durable acceptance seam with correct retry/idempotency behavior.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `closedloop/engine_test.go` accepted path, `outcome/store_test.go`, `test/arch/v14_outcome_event_test.go`
2. **Contract / Interface:** Interface: existing `outcome.KindAccepted`, append-only store, task/changeset binding
3. **Core Implementation:** Core: attach writer immediately after authoritative durable acceptance seam, not runtime self-report
4. **Failure Handling:** Failure: retry/idempotency must not append duplicate semantic accepted event; store failure policy must be explicit

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
