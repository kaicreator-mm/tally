# P1-08 — Compiled-authority no-bypass gate

- **Phase:** P1
- **Kind:** gate
- **Wave:** W14
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P1-05, P1-06, P1-07
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-12) — **P1 gate: PASS**(带一项显式记录、归属其属主的例外:`scripts/v14_trial`,P4-03 所有,见报告 §6)。
- 三项 P1 遗留(明示移交本任务)在 test/** 面闭合:测试等价组合 `clRunTask`(P2-10)、`bfRunTask`(P3-03)、`migRunTask`(P3-04)现与 `cmd/worker runV14Task` 执行前置序逐件一致 —— `artifacts.LoadBundle`(P1-01,DL-04)→ `CompileAuthority` 编译权威第一道门、先于任何 runtime 会话与 workspace(P1-03/04,PRD §16)→ `Compiled.Requires()` 进 ContextPack 与每轮迭代 Prevention 小节(P1-05)→ `LoopConfig.Authority` 必达(P1-06 验收守卫);夹具仓库冻结 FrozenIntent(db_migration 任务类型经 P1-02 确定性派生 schema_migration,迁移场景的权威风险输入回归证据面)。
- AC:Same-source invariant tested — PASS(`TestV141GateAuthorityReproducibleFromOneCommit`:同 pinned HEAD 重载+重编译身份/哈希确定性复现)。L3-required/L1-achieved blocked — PASS(`TestV141GateRequiredL3AchievedL1BlockedE2E` + 迁移闭环 `…RepairGuardEscalates`:FinalVerified 候选被守卫裁决 `assurance_insufficient`,运行时与读模型 VERIFIED_LIMITED 同源一致)。Held-out content not leaked — PASS(编译通道零泄漏扫描)。Unknown/unmatched required risk is fail-safe — PASS(unknown 信号逐字保真 + L3 上调 + 拒绝普通接受)。蓄意 bypass 夹具(拒收策略/缺 intent/混合来源手工 bundle)全部在接受之前类型化失败,零 runtime 会话、零候选事实。
- Repo-wide 机器守卫:`test/arch/v141_no_bypass_gate_test.go` 要求全仓每个 `closedloop.LoopConfig` 字面量显式绑定 `Authority`(别名/包内裸形态不构成绕过),例外清单显式+活性受检(`internal/closedloop` 两处守卫自身语义钉;`scripts/v14_trial` 归 P4-03/P4-07)。
- Verification:全套 integration e2e PASS(365.7s,真实 PostgreSQL 15432 + 真实临时 Git 仓库);`go test ./test/arch/...`、`./test/runtimeconformance/...`、`./internal/...`、`./cmd/worker`、`go build/vet ./...`、`git diff --check` 全 PASS;`make check` NOT_RUN(Git Bash 无 make;构成族逐项运行)。Full evidence: `reports/P1-08_compiled_authority_no_bypass_gate.md`。Decision Locks not bypassed;hidden/ unread(DL-17)。


## Goal

Prove compilation, prevention context, risk coverage and assurance all constrain real acceptance.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Same-source invariant tested.
- [ ] L3-required/L1-achieved blocked.
- [ ] Held-out content not leaked.
- [ ] Unknown/unmatched required risk is fail-safe.


## Verification

- Run focused integration/E2E suite and archive evidence.

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
1. **Tests:** Tests: `test/arch/v14_contract_authority_test.go`, `v14_assurance_failsafe_test.go`, `verification/holdout_path_leak_test.go`
2. **Contract / Interface:** Contract: one source commit + Compiled authority + no held-out leakage + assurance guard
3. **Core Implementation:** Core: E2E through worker/Closed Loop, not isolated unit-only checks
4. **Failure Handling:** Failure: add deliberate bypass fixtures and assert they fail before acceptance

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Fail before runtime start when authority cannot be reproduced from one source snapshot.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not compute policy/risk/assurance from mixed commits or silently downgrade unknowns.
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
