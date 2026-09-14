# P6-04 — Migration false-negative reproduction

- **Phase:** P6
- **Kind:** spike
- **Wave:** W35
- **Complexity:** XL
- **Execution model:** HIGH
- **Depends on:** A0-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **根因裁决：fixture（trial 组合）为触发器 + verifier 语义为放大器；排除 environment / DB lifecycle / oracle / timeout / state-leakage。** 证据链：(1) 冻结契约（preserved at /tmp/tally-v14-trial/.../.tally/verifications/TASK-9106.yaml）记录 argv 子命令为原始 gate keys（`fresh_database` 等）；(2) `scripts/v14_trial/tally/probe/main.go` 仅接受 `fresh|n1|integrity` → 任何 DB 访问前 exit 2 "unknown subcommand"（内容无关）；(3) `internal/verification/orchestrator.go` executeVisibleBinding（L602-608）将所有非零 exit 映射为 `EvidenceStatusFail` 且无 Err → `FinalFailed`（与 recorded `verification_lineage.status=="failed"` 非 "error" 一致）；(4) attempt-3 候选与冻结参考修复字节一致仍失败，而外部判官接受 5/5；(5) bug_fix 臂组合了合法子命令（repro/after）——解释为何仅 db 任务受影响。
- 复现：单元钉（exit 2 → 伪造 `EvidenceStatusFail` → `FinalFailed`，trial 形冻结契约 + 字节完美修复候选）与 e2e 双层（A 触发 / B 生产 Orchestrator 误报 / C 正确子命令对照三绑定全绿于真实 PostgreSQL），各跑 2 次确定性 PASS。P6-05 修复面：compose.go 做 gate key → probe 子命令映射（**在 P6-05 Allowed Paths 之外，已由 orchestrator 显式授权最小修复并记录**）；orchestrator 增加基础设施 exit-code 通道，使文档化接线错误产出 needs_review（无裁决、无事实）而非受信候选 FAIL；必须保留 exit-1-trusted-FAIL 负路径。
- Verification：`go test ./internal/verification/ -run TestP604 -count=2` PASS；`./internal/verification/...` PASS；`-tags=integration ./test/e2e/ -run TestP604MigrationFalseNegativeReproE2E -count=1 -v` PASS ×2（44.5s/44.1s）；`-run "TestV14Migration"` PASS（79.2s 零回归）；vet/gofmt/`git diff --check` 清洁。Full evidence: `reports/P6-04_migration_false_negative_reproduction.md`。

## Goal

Reproduce and classify the v1.4 experiment disagreement where internal verification failed but external judging accepted migration candidates.


## Allowed / primary paths

- `test/**`
- `internal/verification/**`
- `docs/tasks/v1.4.1/spikes/**`


## Implementation requirements

1. Classify contract/fixture/DB lifecycle/environment/oracle/timeout/setup-teardown/state leakage.
2. Do not patch production verifier before root cause is demonstrated.


## Acceptance criteria

- [ ] At least one reproducible minimal case or a documented evidence-backed inability to reproduce.


## Verification

- Reproduction matrix and logs.

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
- **Reason:** Root-cause diagnosis task; requires discriminating verifier, fixture, DB lifecycle, environment and oracle failures.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests/evidence first: v1.4 trial/experiment migration fixtures, `verification/orchestrator_test.go`, visible/holdout executors, execution_context tests
2. **Contract / Interface:** Interface: exact frozen verification contract + environment identity + run records
3. **Core Implementation:** Core: reproduce before edit; classify fixture, DB lifecycle, environment, oracle, timeout, setup/teardown, state leakage
4. **Failure Handling:** Failure: if not reproducible, report evidence gap; do not 'fix' verifier based on hypothesis

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

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
