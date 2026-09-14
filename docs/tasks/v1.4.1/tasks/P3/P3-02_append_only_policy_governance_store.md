# P3-02 — Append-only policy governance store

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W20
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P3-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — append-only 治理事件面落地:`migrations/0024_policy_governance_event.sql`(action_kind CHECK 封闭 draft|submit|activate|reject|supersede;candidate_yaml 所见即所批 —— 非运行时权威;evidence_refs TEXT[];occurred_at 写入方显式;`tally_prevent_modify()` 语句级触发器,0 行命中也报错;**无任何「当前 ACTIVE」表/布尔列** —— 激活只是事件行,运行时生效 = P3-07 的 gitwriter 正典 YAML 提交,DL-09/BI-01)+ `internal/policygovernance/store.go`(`Store.Append` 唯一写路径 + 域侧 fail-safe 校验先行;`EventsForProposal`/`AllEvents` id 升序;`Reconstruct` 按序重放域层封闭转移重建最终提案 —— 空流 `ErrUnknownProposal`,乱序/候选内容漂移/被污染证据串 `ErrInconsistentHistory`,绝不猜测修补)。列名裁定:治理动作列命名 `action_kind` 而非 `event_kind` —— `TestV141OutcomeKindSetStaysFive`(BI-02 禁平行 outcome 事件库)开发中捕获初版命名,治理动作语义上亦是 action,改名使两域词汇正交,**守卫原样保留**(未改任何测试)。
- AC:提案历史可重建、无可变运行时策略权威 — PASS(DB-backed 5 测试 40.9s:draft→submit→activate 全链重建 ACTIVE + 证据 + 候选修订核对;reject 链终态;Append 负路径 ×4;读侧证据污染 fail-safe;DB 层 CHECK + UPDATE/DELETE 触发器 + 空流类型化错误;乱序历史 ErrInconsistentHistory)。`./test/arch/...` PASS;vet/gofmt/`go build ./...`/`git diff --check` 清洁。internal/store/**(§10 sqlc 热点)零触碰 —— 存储直用 pgx 池(INV-6 白名单)于本包自有表。
- PRODUCTION_REACHABILITY: N/A(reason: 域+存储;读投影归 P3-06,激活链归 P3-07,重放归 P3-04)。执行过程:平台模型请求故障持续,编排者主会话完成。Full evidence: `reports/P3-02_append_only_policy_governance_store.md`。


## Goal

Persist proposal, approval/rejection and replay facts without creating an authoritative ACTIVE policy DB table.


## Allowed / primary paths

- `internal/policygovernance/**`
- `migrations/**`


## Implementation requirements

1. Append-only where practical.
2. Runtime policy activation is not derived solely from DB flag.


## Acceptance criteria

- [ ] Proposal history reconstructable; no mutable runtime policy source-of-truth introduced.


## Verification

- Store/migration tests.

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
1. **Tests:** Tests: append-only patterns in `test/arch/append_only_test.go`, `v14_outcome_event_test.go`
2. **Contract / Interface:** Interface: append-only proposal/review/replay facts
3. **Core Implementation:** Core: existing migration/sqlc conventions under `migrations/`, `internal/store/queries/`
4. **Failure Handling:** Failure: no mutable `active=true` database authority for runtime policy

## External reference pack
- **EXT-GITHUB-RULESETS — GitHub Repository Rulesets — evaluate/rule insights pattern**  
  https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository  
  License/version: Documentation/reference only; current GitHub Docs.  
  Use: Pattern for rule proposal/evaluate-before-enforce and inspecting affected historical/current actions.
- **EXT-GITLAB-APPROVAL — GitLab Merge Request Approval Policies**  
  https://docs.gitlab.com/user/application_security/policies/merge_request_approval_policies/  
  License/version: Documentation/reference only; current GitLab Docs.  
  Use: Pattern for condition-based approval, fail-safe behavior when required evidence/scans are unavailable.

## Do

- Treat Git policy content as runtime authority; governance is evidence and authorization around it.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not build a second policy DSL/engine or treat DB governance status as runtime activation.
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
