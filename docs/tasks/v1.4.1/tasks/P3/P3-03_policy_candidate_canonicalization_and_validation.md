# P3-03 — Policy candidate canonicalization and validation

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W20
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P3-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 候选正典化与验证落地（`internal/engineeringpolicy/**` 零改动 —— 运行时模式冻结,消费即可;要求 1:零第二 DSL,全部复用既有解析器）: `canonical.go` 新增 `VerifyCandidate(raw, claimedRevision)` —— 经 `engineeringpolicy.Parse` 正典解析 + 声称修订一致性核对（内容修订可独立重导出,不信任调用方自报）; `store.go` Append 增强: 每事件写入前过 VerifyCandidate（修订漂移/不可解析候选写入处类型化拒绝,不再等到读侧重放）+ 重复身份 fail-closed（第二个 draft 事件 → 新哨兵 `ErrDuplicateProposal`;append-only 流不允许身份重铸;非重复后续事件正常可落）。
- AC:候选修订 = 实际正典策略内容修订语义 — PASS（金色测试:独立 `Parse → RevisionHash` 语义核对;键序/空白/列表顺序三种排版变体修订稳定 + 正典形式逐字节一致;语义内容变化修订必变）。要求 2:重复身份 fail-closed — PASS（DB-backed 写入拒绝测试）。`DATABASE_URL=… go test ./internal/policygovernance/ -count=1` PASS 54.2s（9 测试）;`./test/arch/...`/`go build ./...`/vet/gofmt/`git diff --check` 全 PASS。跨提案同内容重复（supersede 流）是合法治理形态,不拒绝（边界已在报告/测试如实记录）。执行过程:平台模型请求故障持续,编排者主会话完成。Full evidence: `reports/P3-03_policy_candidate_canonicalization_and_validation.md`。


## Goal

Validate proposed rule content through the existing EngineeringPolicy parser and content-derived revision semantics.


## Allowed / primary paths

- `internal/policygovernance/**`
- `internal/engineeringpolicy/**`


## Implementation requirements

1. Do not create a second policy DSL.
2. Invalid/duplicate identity fails closed.


## Acceptance criteria

- [ ] Candidate revision equals actual canonical policy content revision semantics.


## Verification

- Golden parse/revision tests.

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
1. **Tests:** Tests first: `engineeringpolicy/parse_test.go`, `hash_test.go`, `policy_test.go`, `v14_policy_loader_test.go`
2. **Contract / Interface:** Interface: `engineeringpolicy.Parse` supplies content-derived Revision
3. **Core Implementation:** Core: candidate validation must call existing parser; no second DSL
4. **Failure Handling:** Failure: duplicate ID, unknown fields, invalid enforcement/gate content reject before replay

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
