# P3-01 — Policy governance proposal domain

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W19
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P2-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 治理提案域落地 `internal/policygovernance`（新包，P3 属主；DL-08/BI-01 全遵守）：`Candidate` 不可变候选 —— 构造时经 `engineeringpolicy.Parse` 正典解析（KnownFields 严格模式），原始字节深拷贝隔离、零 setter，内容修订（RevisionHash）出生即固定；`Proposal` 治理身份与生命周期 —— **BI-01 指定封闭词表** `DRAFT/PROPOSED/ACTIVE/SUPERSEDED/REJECTED` + `evidence_refs`（封闭 kind 词表 outcome/changeset/verification/task，全部指向既有事实面）/`approved_by`（Activate 硬性要求）/`supersedes`（append-only 指针）/`scope`（治理说明，非运行时匹配语义）；四个具名封闭转移 `Submit/Activate/Reject/Supersede`，其余一律 `ErrInvalidTransition`，**失败时原样返回接收者**（调用方保留原提案）。要求 2 结构性成立：运行时策略 YAML 面不存在治理词汇 —— 混入 `governance_status` 等治理字段的候选在正典解析处出生即拒绝（负测试钉死）；「runtime consumes ACTIVE」由 P3-07 的 gitwriter 正典 YAML 提交实现（DL-09），本包零落库（P3-02）、零 Git 写、零运行时开关。
- AC：提案引用可正典解析策略候选与证据 — PASS（`NewProposal` 硬性 ≥1 证据 + 候选解析失败即拒）。域验证测试 5 个（AC 正面 / 候选不可变性双面 / 治理字段污染负例 / 构造负路径表 / 全合法 + 9 组非法转移表）：`go test ./internal/policygovernance/ -count=1` PASS（0.687s）；vet/gofmt/`go build ./...`/`./test/arch/...` 全 PASS。
- PRODUCTION_REACHABILITY: N/A(reason: 域包;持久化归 P3-02 append-only store,激活归 P3-07 gitwriter,读投影归 P3-06 —— 本任务契约即域 + 验证)。执行过程:平台模型请求故障持续,编排者主会话完成。Full evidence: `reports/P3-01_policy_governance_proposal_domain.md`。


## Goal

Represent proposed policy content, evidence refs and governance identity without modifying runtime EngineeringPolicy schema.


## Allowed / primary paths

- `internal/policygovernance/**`


## Implementation requirements

1. Candidate policy content/revision immutable.
2. Proposal lifecycle is governance state, not runtime policy state.


## Acceptance criteria

- [ ] Proposal references canonical parseable policy candidate and evidence.


## Verification

- Domain validation tests.

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
1. **Tests:** Tests: `engineeringpolicy/{parse_test,policy_test,hash_test}.go`, `test/arch/v14_runtime_boundary_test.go::TestV14EngineeringPolicyRejectsExecutionPolicyFields`
2. **Contract / Interface:** Interface: canonical `engineeringpolicy.Policy` remains runtime schema
3. **Core Implementation:** Core: new governance proposal wraps/references candidate policy content rather than extending Policy
4. **Failure Handling:** Failure: proposal status/evidence must not appear in runtime policy YAML

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
