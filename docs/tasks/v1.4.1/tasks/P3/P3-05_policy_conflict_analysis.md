# P3-05 — Policy conflict analysis

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W21
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P3-03
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 冲突分析落地 `internal/policygovernance/conflict.go`（新文件，零既有文件改动；engineeringpolicy 零触碰 —— 封闭词汇内消费）：`AnalyzeConflicts` 输出封闭四类发现 —— `duplicate_id`(conflict:同 id 多份,携带全部内容修订)、`shadowed_duplicate`(warning:不同 id 全同语义,完全遮蔽)、`enforcement_mismatch`(warning:同触发面 + 同 requires 键集 + gate/guidance 强度不同 —— 键比较只用 requires,guidance 词汇上不携带 gate_condition)、`stacked_gates`(info:触发轴重叠双 gate 的合法合取叠加,人需知晓叠加后全部键并集);**零自动消解** —— 全部输出型,保守方向(一方零值触发面按重叠上报);确定性 AC — 输出按 kind 词表次序 + ID 升序稳定排序,输入顺序无关。
- AC:已知矛盾/重叠夹具产出显式 conflict/warning — PASS(表驱动 7 行:干净零发现/重复 id/叠加/强度不一致/完全遮蔽/不相交零发现/复合三发现并存;每发现断言非空 Detail + 涉及 ID);确定性(旋转输入 3 轮序列逐字节一致);零值触发面保守上报。开发期两轮自引入失败(AllOf 比较使 gate/guidance 对永不命中;复合夹具与组内去重冲突)由表驱动测试捕获并当轮修正 —— 同 id 组去重是有意设计(身份冲突已上报,不级联)。
- Verification:`DATABASE_URL=… go test ./internal/policygovernance/ -count=1` PASS 54.3s(全包:域/存储/正典化/冲突);arch/build/vet/gofmt/`git diff --check` 全 PASS。PRODUCTION_REACHABILITY: N/A(reason: 分析面;投影归 P3-06,治理链消费归 P3-07/P3-09)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P3-05_policy_conflict_analysis.md`。


## Goal

Detect candidate interactions that require human attention without inventing a generic Boolean rule engine.


## Allowed / primary paths

- `internal/policygovernance/**`
- `internal/engineeringpolicy/**`


## Implementation requirements

1. Use existing trigger/requires/gate/enforcement vocabulary.
2. Prefer conservative warning over unsound auto-resolution.


## Acceptance criteria

- [ ] Known contradictory/overlapping fixtures produce explicit conflict/warning.


## Verification

- Table-driven conflict tests.

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
- **Reason:** Policy conflict semantics are ambiguous and must stay conservative without inventing a new rule engine.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: policy gate/match/parser tests; build explicit overlapping/contradictory fixtures
2. **Contract / Interface:** Interface: existing Trigger/Requires/Gate/Enforcement vocabulary only
3. **Core Implementation:** Core: compare normalized policy effects conservatively; warnings are preferable to automatic resolution
4. **Failure Handling:** Failure: no generic Boolean/rule interpreter; unknown interaction must require review

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
