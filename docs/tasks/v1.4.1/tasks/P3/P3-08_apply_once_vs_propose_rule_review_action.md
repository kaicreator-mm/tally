# P3-08 — Apply Once vs Propose Rule review action

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W23
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P2-04, P3-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 评审页双动作落地（`internal/approval/**` 零改动 —— 复用既有审批证据语义，要求 1）：**(a) 仅此一次** = 既有 decide 流原样保留并明确标注（决定只写本候选绑定的 HumanApprovalEvidence，绝不自动创建/修改策略 —— AC 由既有语义保证，`TestV13ApproveWritesHumanApprovalEvidenceNotStatus` 全断言不变绿）；**(b) 提议新规** = 新增 `POST /v13/review/{csid}/propose-rule`（`V13ProposeRule`）：候选 YAML 出生即正典解析校验（治理字段混入即拒），草稿事件落 policygovernance（证据引用 changeset:<id> + task:<id>），**零审批证据写入、零自动批准、零运行时策略触碰**（两动作互不触发，AC 由结构保证 —— 不同处理器、不同事实面）。`server.go`（§10 热点，声明式最小接缝）：`Governance *policygovernance.Store` 字段（nil → 表单按未接入呈现 + POST 显式失败，两层一致）+ 一行路由。顺带修复 P3-02 的 Reconstruct 缺口（draft 事件初始证据引用未回填 —— 一行补全）。
- AC:一次性决定绝不自动改策略 — PASS（DB-backed 4 测试：happy 草稿重建 + 双证据引用 + 零审批证据；治理污染/缺提案人 fail-closed 零残留；仅此一次 decide 恰一条审批证据且治理事件流为零；未接入时 propose 显式失败而 decide 不受影响）。`./internal/web/...` 全包 PASS 187s；policygovernance 54.1s 回归；arch/build/vet 清洁。
- PRODUCTION_REACHABILITY: N/A(reason: `NewRouter`→`Server.Governance` 接线不在本任务路径(router.go/cmd/server)；handler 级能力就绪，nil 语义 fail-safe，接线归后续整合任务)。执行过程:平台模型请求故障持续,编排者主会话完成。Full evidence: `reports/P3-08_apply_once_vs_propose_rule_review_action.md`。


## Goal

Let a human resolve one case or create a separate rule proposal without auto-activating policy.


## Allowed / primary paths

- `internal/web/handlers/**`
- `web/templates/**`
- `internal/approval/**`
- `internal/policygovernance/**`


## Implementation requirements

1. Apply Once uses existing task approval evidence semantics where applicable.
2. Propose Rule creates governance proposal only.


## Acceptance criteria

- [ ] No one-off decision mutates policy automatically.


## Verification

- Review action tests.

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
1. **Tests:** Tests: `v13_http_test.go::TestV13ApproveWritesHumanApprovalEvidenceNotStatus`, approval store tests
2. **Contract / Interface:** Interface: one-off human approval evidence vs independent RuleProposal creation
3. **Core Implementation:** Core: existing approval facts + policygovernance create proposal
4. **Failure Handling:** Failure: `Apply Once` never mutates policy; `Propose Rule` never auto-approves

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
