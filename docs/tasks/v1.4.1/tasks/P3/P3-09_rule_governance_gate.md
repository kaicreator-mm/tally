# P3-09 — Rule-governance gate

- **Phase:** P3
- **Kind:** gate
- **Wave:** W24
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P3-04, P3-05, P3-07, P3-08
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **P3 门禁: PASS**(零生产代码改动;P3-01..08 交付面原样组合于真实临时内容仓库 + 真实治理事件库)。端到端链:提案(出生正典校验 + draft/submit 证据齐备)→ 重放(ReplayCandidate:录得键 would_have_passed;**n1_upgrade 无录得 → insufficient_evidence 一等输出,AC2/DL-10**)→ 人本批准激活(P3-07:CAS 提交 → LoadPolicies 重载验证同 id 同修订 → 治理 ACTIVE + approved_by;**AC1:激活后 HEAD 快照可观察**)。**分裂脑刻意注入**:第二提案只落治理 ACTIVE 事件、无 Git 提交 → 治理面记 ACTIVE 而 HEAD 不可观察,运行时权威跟随 Git(AC1 对立面结构性证明,DL-09/BI-01)。**陈旧激活**(AC4):过期基线类型化拒绝、治理面保持 PROPOSED、零激活事件。**治理字段**(AC3):`governance_status` 混入的策略制品被运行时装载器整体拒绝(Rejected,永不进生效集),合法策略同快照不受影响。P3-08 复用确认:评审页 propose-rule 写入的即本链消费的治理事件面。
- 验证:`-tags=integration ./test/e2e/ -run TestP309RuleGovernanceGate -count=1 -v` PASS 8.14s + 双跑确定性;`./test/arch/...` PASS;build/gofmt/`git diff --check` 清洁。两轮开发期失败均为门禁夹具自身(同 id 歧义语义触发另一条正确守卫;断言方向写反),当轮修正;非基线失败。
- **P3 lane 关闭,P3-09 发布阻塞解除**(release blockers 列表)。剩余风险:split-brain 反向(治理缺记录、Git 已生效)由 P3-07 报告文档化;治理库宕机注入夹具留待演练。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P3-09_rule_governance_gate.md`。


## Goal

Prove rule governance can evaluate, approve and activate without split-brain policy authority.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Git snapshot remains runtime authority.
- [ ] Replay emits insufficient evidence when needed.
- [ ] No EngineeringPolicy governance fields added.
- [ ] Stale activation rejected.


## Verification

- End-to-end rule proposal/replay/activate/reload test.

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
1. **Tests:** Tests: P3 E2E plus `test/arch/v14_runtime_boundary_test.go` policy-boundary guards
2. **Contract / Interface:** Contract: Active runtime policy must be observable from pinned Git commit
3. **Core Implementation:** Core: proposal→replay→human decision→GitWriter→reload
4. **Failure Handling:** Failure: deliberately diverge governance DB and Git snapshot and ensure runtime follows Git

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
