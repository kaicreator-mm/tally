# P3-06 — Rules UI projections

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W22
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P3-02, P3-04, P3-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — Rules 投影页 `/v14/rules` 落地(`v14_rules.go` + `v14_rules.tmpl`,注册进 RegisterV14Routes 与 v14CA 模板组;base.tmpl 研发组补 规则 条目闭合 P2-01 预留;nav known-routes 集合声明式扩展,路由已真实注册)。四分区全投影:**Active 只从 `artifacts.LoadPolicies(repo, HEAD)` 派生**(DL-09;AC「UI 不得把 DB-only 提案显示为 Active」由数据流方向结构保证 —— Active 不读治理面),每条携带治理一致性标注;**治理分歧区**(治理 ACTIVE 但 HEAD 不可观察:同 id 缺席/修订不一致,注明运行时不受影响);**Proposed**(未激活提案含策略 id/证据数);**Conflicts**(P3-05 只呈现不消解);**History**(事件流倒序截断 + 全长声明;重放失败提案呈现为「治理历史不一致」不静默)。不确定性一等可见:HEAD 有而治理无记录(或修订不一致)的策略逐条不确定标注;治理未接入时记录面诚实呈现、Active(Git 权威面)照常。
- **DAG 顺序偏离声明**:Depends 含 P3-04(未完成,平台通道故障);AC 的 replay evidence/uncertainty 可见性由治理/Git 一致性标注实现(一致性/分歧/不确定三态),P3-04 专用重放服务落地后可深化展示 —— 页面契约(Active 来源 + 不确定性可见)已完整。
- 验证:分歧夹具 5 测试(A 治理领先绝不显示为 Active / B 分歧区 + DL-09 注记 / C 一致标注零分歧 / D 不确定标注 / E 治理未接入 Active 照常)全 PASS;`./internal/web/...` 全包 222.6s;arch/build/vet/gofmt/`git diff --check` 清洁。开发期自引入缺陷(字段名遮蔽 Page.Active 导致导航渲染错误被吞)由测试捕获并当轮修正。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P3-06_rules_ui_projections.md`。


## Goal

Show Proposed/Active/Conflicts/History where Active is derived from actual Git policy snapshot.


## Allowed / primary paths

- `internal/web/handlers/**`
- `web/templates/**`
- `internal/policygovernance/**`
- `internal/artifacts/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] UI cannot show DB-only proposal as Active.
- [ ] Replay evidence and uncertainty are visible.


## Verification

- Handler tests with diverged governance/Git fixtures.

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
1. **Tests:** Tests: `v14_policy_loader_test.go` + new governance projection tests
2. **Contract / Interface:** Interface: Proposed from governance store; Active from exact Git policy snapshot
3. **Core Implementation:** Core: handlers/templates compose both read sources without merging their authority
4. **Failure Handling:** Failure: DB-only proposal must never display as runtime Active

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
