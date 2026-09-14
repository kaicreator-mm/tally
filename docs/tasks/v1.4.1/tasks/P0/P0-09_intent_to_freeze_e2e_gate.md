# P0-09 — Intent-to-Freeze E2E gate

- **Phase:** P0
- **Kind:** gate
- **Wave:** W07
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P0-08
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Prove the complete human intent → dynamic clarification → immutable freeze path on a real temporary Git repository.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] At least bugfix and migration/intake-risk scenario freeze correctly.
- [ ] Missing verification evidence blocks freeze.
- [ ] All artifacts load from pinned commit.


## Verification

- Run E2E and record commit/artifact hashes.

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
1. **Tests:** Tests: `v14_launch_test.go::{TestV14ProposeRendersEditableDraftAndConfirmBoundary,TestV14LaunchRunBlocksWithoutFrozenVerification}`, `test/e2e/*` patterns
2. **Contract / Interface:** Contract: FrozenIntent + TaskContract + Frozen Verification must all resolve from pinned commit
3. **Core Implementation:** Core: public launch/confirm route and Git-backed loaders
4. **Failure Handling:** Failure: missing evidence, stale base and invalid clarification must be exercised as negative E2E cases

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Keep presentation, intent evidence and frozen engineering artifacts explicitly separated.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not let LLM-generated UI/risk text become authority without deterministic validation/freeze.
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

---

## Execution status

- **Status: DONE**(gate 结论:PASS,2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P0-09_intent_to_freeze_e2e_gate.md`(含 gate 提交/制品哈希记录)
- Implemented: 新 `test/e2e/v141_intent_to_freeze_gate_test.go`(integration tag,零生产改动):六个测试经**生产路由原样装配**(组合根同款 `RegisterV14Routes` + `EnsureRepo` 引导 + `SetV14ProposeExecutor` 注入缝 + 真实临时 Git 仓库)端到端证明 Intent→Confirm→Freeze 权威链 —— bug_fix 全链(提案→澄清→复核折叠→草案→确认→冻结→pinned 读回一致+确定性+DL-06 候选不晋升)、db_migration 场景(冻结提交恰含三件套三个新文件,原子性)、缺失证据阻断(AC2)、非法澄清复核阻断(无草案无新 AI 调用)、重复确认内容幂等、陈旧基线类型化冲突+当前基线恢复。AI 环节用确定性替身(生产注入缝),真实 DeepSeek 端点由 P0-05 connected 测试单独覆盖(本次 PASS,如实区分)。
- Acceptance: 「bugfix 与 migration/intake-risk 场景正确冻结」PASS;「缺失验证证据阻断冻结」PASS(400 类型化列出缺失条款,零提交零制品);「全部制品从 pinned 提交装载」PASS(完整 sha 装载,SourceCommit==pinned,重复装载相等,冻结前基线无 intent)。「Run E2E and record commit/artifact hashes」PASS(报告第 2 节)。
- Validation: `go test -tags=integration -run 'TestV141IntentToFreezeGate' -v ./test/e2e/` PASS(6/6);`DATABASE_URL=…15432… go test -tags=integration ./test/e2e/...` PASS(整包 327.8s,零基线失败);`go test ./test/arch/...` PASS;`go build ./...` PASS;`go vet -tags=integration ./test/e2e/` PASS;`gofmt -l`(新文件)clean;`git diff --check` clean;`make check` NOT_RUN(reason: Git Bash 无 make;构成命令逐条执行)。
- Baseline failures: none。
- Decision Locks touched: none(DL-03/04/06/11/17/18 遵守;hidden/ 未读)。
- Risks: gate 的 AI 环节为生产注入缝上的确定性替身(真实端点连通性归 P0-05 connected 测试);stale-base 负面落在生产事务函数(HTTP 层无并发注入窗口,沿用 P0-08 分区并在测试头注记)。
- Commit: 见本文件 git 历史(`P0-09: intent-to-freeze e2e gate — …`)。
