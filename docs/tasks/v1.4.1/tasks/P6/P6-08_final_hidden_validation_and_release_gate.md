# P6-08 — Final hidden validation and release gate

- **Phase:** P6
- **Kind:** gate
- **Wave:** W39
- **Complexity:** XL
- **Execution model:** HIGH
- **Depends on:** P3-09, P4-07, P5-06, P6-03, P6-05, P6-07
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: PENDING_EXTERNAL** (2026-09-13) —— 最终 hidden validation 按其定义由**外部流程**执行:hidden validation pack 外部持有且被扣留(DL-17:实现侧不搜索、不请求、不挂载、不摘要 hidden 材料;候选 HEAD 冻结后才运行)。实现侧**前置条件全部满足**:可见任务 56/57 DONE(仅本任务),三个 lane gate 全部解除(P3-09 split-brain PASS、P4-07 DSH LIMITED_READY、P5-06 Git 控制面 PASS),release blockers 除本任务外全部 RESOLVED,文档同步完成(P6-07),DL-17 的 hidden/ 未读纪律在全部 32 次任务执行中保持。HEAD 冻结点:`2d465b8`(P6-07)—— 外部流程从该点(或其后)执行本门禁。
- 实现侧移交面:全部任务报告在 `docs/tasks/v1.4.1/reports/`;live 进度权威 `docs/tasks/v1.4.1/STATUS.md`;已知限制汇总:DSH LIMITED_READY(P4-07 §3:live 内容产出/R-2/R-3)、Outcome 四类的已文档化生产者与已接线生产者 gap(P6-02)、P3-07 治理库宕机注入夹具未构造(§5)、幂等 check-then-append 非 DB 强制(P6-01/P6-02 同类限制)。


## Goal

Run adversarial and end-to-end release validation across authority bypass, policy, runtime, Git, verification, outcome and docs.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`
- `docs/tasks/v1.4.1/hidden/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Final verdict is PASS / PASS_WITH_KNOWN_LIMITATIONS / FAIL with explicit evidence.
- [ ] No unexplained P0/P1 authority gap remains.


## Verification

- Connected full test suite + hidden scenarios + artifact audit.

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
- **Reason:** Final adversarial release judgment spans all authority boundaries and must detect subtle bypasses.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: all phase gates + `test/arch/*` + hidden/adversarial scenarios
2. **Contract / Interface:** Contract: Decision Locks and PRD Amendments are release authority
3. **Core Implementation:** Core: connected `make check` plus E2E/hidden validation and export audit
4. **Failure Handling:** Failure: verdict must preserve known limitations; no PASS if P0/P1 authority bypass remains

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
