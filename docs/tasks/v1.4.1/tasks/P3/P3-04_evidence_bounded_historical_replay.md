# P3-04 — Evidence-bounded historical replay

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W21
- **Complexity:** XL
- **Execution model:** HIGH
- **Depends on:** P3-02, P3-03
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 证据边界重放落地 `internal/policygovernance/replay.go`（新文件，零既有文件改动；纯函数，同一证据快照重放恒等）：`ReplayCandidate(ReplayInput) ReplayResult` —— 匹配判定复用 `engineeringpolicy.Match`（零第二 DSL），三值封闭：matched/unmatched/**insufficient_evidence 一等**（触发轴未录得时强制 INSUFFICIENT，不猜 matched/unmatched，且零差量输出）；差量 = 要求键集/门禁键集相对「当时命中的历史策略集」（assurance 差量的治理面表达 —— DL-08 下政策不携带等级，被拒方案：伪造等级比较）；逐键结局**只认录得事实**（`RecordedGateResults`:验证绑定结果按 gate 键投影 → would_have_passed/would_have_failed，缺席键 = insufficient_evidence 一等呈现，绝不猜 DL-10）；冲突复用 P3-05（仅涉及候选者）；unknown/unmatched 风险信号原样保真（INV-V14-09）。被拒替代方案（直查事实面破坏确定性/模拟执行违反 DL-10/自行触发判定 = 第二 DSL）全记录于报告 §2。
- 验证矩阵（表驱动 + 负测试 9 项）：matched+差量+passed / k9 无录得 → 一等 insufficient / 录得 failed → would_have_failed / 触发轴未录得 → INSUFFICIENT 且零差量 / unmatched / unknown 保真 / 同 id 冲突可见 / **确定性 AC 双跑恒等** / 无效候选拒绝。`-run TestReplay -count=2` PASS；`DATABASE_URL=… ./internal/policygovernance/` 76.0s 全包；engineeringpolicy/outcome/taskstate 邻包零改动回归 PASS；arch/build/vet/gofmt/`git diff --check` 清洁。
- PRODUCTION_REACHABILITY: N/A(reason: 纯域函数;事实面→快照投影与 UI 消费归后续整合/Rules 页深化)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成(HIGH 接缝判定与被拒替代方案见报告 §2)。Full evidence: `reports/P3-04_evidence_bounded_historical_replay.md`。


## Goal

Replay candidate policy matching/effects against historical facts without fabricating counterfactual execution outcomes.


## Allowed / primary paths

- `internal/policygovernance/**`
- `internal/engineeringpolicy/**`
- `internal/outcome/**`
- `internal/taskstate/**`


## Implementation requirements

1. Output matched/unmatched, requirement/gate/assurance deltas, conflicts, insufficient evidence.
2. Do not claim historical code would pass/fail unless recorded facts make it deterministic.


## Acceptance criteria

- [ ] Insufficient evidence is first-class.
- [ ] Replay is deterministic for same evidence snapshot.


## Verification

- Historical fixture tests incl. missing evidence.

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
- **Reason:** Historical replay must remain evidence-bounded and avoid unsound counterfactual inference.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: `engineeringpolicy/match_test.go`, `contract/compile_test.go`, `outcome/store_test.go`, `taskstate/basis_test.go`
2. **Contract / Interface:** Interface: candidate policy + frozen historical facts → replay result with matched/effect/insufficient-evidence
3. **Core Implementation:** Core: reuse `engineeringpolicy.Match`, `assurance.Select`, stored historical facts/projections
4. **Failure Handling:** Failure: if evidence needed for a counterfactual was not recorded, emit `INSUFFICIENT_EVIDENCE`; never guess pass/fail

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
