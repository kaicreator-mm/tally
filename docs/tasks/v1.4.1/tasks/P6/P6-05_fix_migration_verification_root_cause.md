# P6-05 — Fix migration verification root cause

- **Phase:** P6
- **Kind:** implementation
- **Wave:** W36
- **Complexity:** XL
- **Execution model:** HIGH
- **Depends on:** P6-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **双面修复落地，P6-04 裁决逐面闭合**：(1) 放大器修复 `internal/verification`：`CommandSpec.InfraExitCodes []int`（yaml `infra_exit_codes`，omitempty）声明文档化接线/基础设施退出码 —— `Validate` 拒绝 <2 与重复（0=pass、1=受信 FAIL 不可重定义为 infra，语义在类型层不可削弱）；`executeVisibleBinding` 命中声明 infra 集 → `Err`（无裁决、无会话记录、无事实）→ `FinalNeedsReview`；`executeGates` 同通道 → `VisibleStatusError`（needs_review），绝不伪造必过门 FAIL；未声明码行为不变（0=pass，其余含 1=受信 FAIL）；**通道 opt-in，既有契约字节等价**。(2) 触发器修复（orchestrator 授权的 compose.go 最小修复）：`migrationProbeSubcommands` 映射（fresh_database→fresh、n_minus_1_upgrade→n1、data_integrity→integrity），未映射键响亮失败；两分支按 probe 文档化 0/1/2 契约声明 `infra_exit_codes: [2]`；scripts/ 其余零触碰（v14_trial LoopConfig 组合未动）。
- P6-04 复现测试翻转为 FIXED 语义（exit 2 → Err + FinalNeedsReview + 零 sink facts）并新增 exit-1 受信 FAIL 与 Validate 结构拒绝测试；e2e 演进：A 不变（gate-key 逐字 argv 仍 exit 2，probe 契约未放松）、B1 trial 形协议 + infra{2} → 三绑定 Err + FinalNeedsReview + 零事实、B2 真 probe + 合法子命令 + 不可达 DSN → 真 exit 1 → FinalFailed 每绑定一 fail fact（回归钉）、C 映射子命令对共享 PostgreSQL 全绿。全部测试语义变化在报告 §2 逐项枚举（无静默削弱，§16）。
- Verification：`go test ./internal/verification/...` PASS；`TestP604 -count=2` 确定性 PASS；`DATABASE_URL=… -tags=integration ./test/e2e/ -run 'TestP604|TestV14Migration'` PASS（P604 43.4s；TestV14Migration 37.0s+40.6s；独立复跑 44.0s）；vet（含 integration tag）/build scripts/gofmt/`git diff --check` 清洁。PRODUCTION_REACHABILITY: PASS（orchestrator 变更在真实终验组合路径，e2e 经 LocalRunner + 真 probe 端到端驱动）。Risks：通道仅在冻结协议声明 infra 码时生效（遗留契约保持旧语义 per INV-VER-02，重定义需新契约修订）；全量 14 迁移绑定实验重跑归 trial 属主（联合因果已在 db-01 e2e 证明）。Full evidence: `reports/P6-05_fix_migration_verification_root_cause.md`。


## Goal

Fix only the demonstrated root cause and add regression coverage.


## Allowed / primary paths

- `internal/verification/**`
- `test/**`
- `docs/**`


## Implementation requirements

1. If root cause is experiment fixture only, do not change production verification semantics.
2. If production bug, preserve held-out/exposure invariants.


## Acceptance criteria

- [ ] Former false-negative fixture behaves correctly without weakening true-negative tests.


## Verification

- Before/after reproduction plus existing verification suite.

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
- **Reason:** Fix depends on diagnosed root cause and must avoid weakening verifier semantics.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: minimal reproduction from P6-04 becomes permanent regression test plus existing true-negative verifier tests
2. **Contract / Interface:** Interface: change only the demonstrated failing seam
3. **Core Implementation:** Core: verifier/fixture/environment code identified by reproduction
4. **Failure Handling:** Failure: fix must not weaken held-out, required gate, stopping-rule or baseline protections

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
