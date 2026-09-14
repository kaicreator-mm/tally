# P6-07 — Current documentation synchronization

- **Phase:** P6
- **Kind:** docs
- **Wave:** W38
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P3-09, P4-07, P5-06, P6-02, P6-05, P6-06
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 文档同步落地(`docs/**`;README/CLAUDE/AGENTS 检查后零改动 —— 无过期生产能力声明,最小变更纪律):**CURRENT_CODEBASE.md** — v1.4.1 current-state 头段(56/57,STATUS.md 为 live 权威;各 lane gated;P6-08 留外部 per DL-17)、拓扑补 gitintegration/policygovernance/runtimes/dsh、Coding Runtime 补 DSH canary 行、§5 四条未决 gap 划线引用证据(规则治理 P3、DSH 适配器 P4 LIMITED_READY、Outcome 写入器 P6-01/02、迁移验证分歧 P6-04/05);**CURRENT_IMPLEMENTATION_STATUS.md** — 16 个 area 行更新到当前事实 + overnight-batch 进度注记(42 任务跨 lane、平台故障执行记录、56/57)。AC:无 CURRENT/STATUS 文档声称能力未启动而 source/gates 说已完成(逐行核检);runtime/rules/git/outcome 边界均已记载。
- 验证:docs link/status consistency 核检 PASS(全部 RESOLVED 引用的 lane/报告存在);`go build ./...`/`./test/arch/...` PASS(代码零改动)。PRODUCTION_REACHABILITY: N/A(docs 任务)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P6-07_current_documentation_synchronization.md`。


## Goal

Make current docs describe the actual v1.4.1 system and clearly archive/supersede stale v1.4 status documents.


## Allowed / primary paths

- `README.md`
- `CLAUDE.md`
- `AGENTS.md`
- `docs/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] No CURRENT/STATUS doc says production capability is not started when source/gates say otherwise.
- [ ] Runtime/Rules/Git/Outcome boundaries documented.


## Verification

- Docs link/status consistency check.

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
1. **Tests:** Tests: docs/status consistency checker from A0 patterns
2. **Contract / Interface:** Interface: README/current-status docs reflect gate reports and actual source
3. **Core Implementation:** Core: update only after final feature gates to avoid repeated drift
4. **Failure Handling:** Failure: archive/supersede historical docs instead of silently rewriting history

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
