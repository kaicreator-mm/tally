# P5-05 — Integration audit/read projection

- **Phase:** P5
- **Kind:** implementation
- **Wave:** W33
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P5-03, P5-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 集成审计读投影落地(双分区,数据来源独立互不推导 —— AC 由结构保证):**Accepted 分区** = 既有审批证据事实面(HEAD 快照 standard 任务的最近决定,复用 P3-08 同一事实面;`internal/approval/**` 零改动);**Integrated 分区** = Git 集成分支事实(`internal/gitintegration/audit.go` 新增只读 `ListIntegrationRefs`:for-each-ref 锁定 refs/heads/tally/integration/ 治理命名空间,输出 ref 名升序携带提交 sha/主题/时刻;+ `IntegrationBranchRefFor` 导出 P5-02 确定性命名规则供对账)。验收行的集成状态标注经确定性命名对账(有对应分支 = 已集成:<ref>,否则「尚未集成」;不对应分支的污染被对账拒绝);边界声明区明示「验收 ≠ 集成」。接线:RegisterV14Routes 一行路由 `/v14/integration`、base.tmpl 研发组补 集成 条目、nav known-routes 扩展(声明式最小接缝)。
- AC:UI 永不把候选验收等同于成功集成 — PASS(DB-backed 3 夹具:仅验收 → Accepted 呈现 + 「尚未集成」+ Integrated 空;并存 → Integrated 呈现分支事实且验收行标注经确定性对账不被无关分支污染;命名空间外引用不可见)。`./internal/gitintegration/` 23.0s(列举真实临时仓库逐项核对 + 空仓库零行 + 非仓库 fail-safe);`./internal/web/...` 245.0s 全包;arch/build/vet/gofmt/`git diff --check` 清洁。
- PRODUCTION_REACHABILITY: PASS(RegisterV14Routes 生产装配 + base.tmpl 导航条目,HTTP 实测)。Risks:P5-03 Merge 推进的目标 ref(合入主线)不在本列举面(合并事实呈现归后续整合任务);引用列举无分页(治理命名空间规模受流程约束)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P5-05_integration_audit_read_projection.md`。


## Goal

Show Accepted and Integrated as separate observable engineering states/facts.


## Allowed / primary paths

- `internal/web/handlers/**`
- `web/templates/**`
- `internal/gitintegration/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] UI never equates candidate acceptance with successful integration.


## Verification

- Read projection tests.

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
1. **Tests:** Tests: v1.4 task/change projection test patterns
2. **Contract / Interface:** Interface: Accepted and Integrated are separate derived facts
3. **Core Implementation:** Core: read model over integration audit facts
4. **Failure Handling:** Failure: failed/conflicted integration must never rewrite task acceptance history

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Use semantic local Git operations with CAS/base checks; conflict is an expected typed outcome.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not expose raw arbitrary Git commands to model-controlled execution or automate force/history rewrite.
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
