# P1-02 — Authoritative risk input derivation

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W09
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P1-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Derive risk.Result from confirmed immutable facts plus deterministic repository/task facts, preserving basis references.


## Allowed / primary paths

- `internal/risk/**`
- `internal/intentevidence/**`
- `internal/contract/**`


## Implementation requirements

1. Candidate LLM risk output is not trusted unless confirmed into FrozenIntent.
2. Unknown/unmatched signals remain visible.
3. No persisted mutable RiskProfile.


## Acceptance criteria

- [ ] Known migration/auth/protected-surface facts produce deterministic signals+basis.
- [ ] Missing basis is rejected per risk invariants.


## Verification

- Risk derivation table tests.

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
- **Reason:** Defines the authoritative source-to-risk derivation chain; silent under-classification is worse than loud failure.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `risk/risk_test.go`, `engineeringpolicy/match_test.go`, `contract/compile_test.go`, `test/arch/v14_assurance_failsafe_test.go`
2. **Contract / Interface:** Interfaces: `risk.Result` with basis refs; `engineeringpolicy.MatchInput`; FrozenIntent facts
3. **Core Implementation:** Core: `risk/derive.go`, `engineeringpolicy/match.go`, `contract.Bind` riskBand logic
4. **Failure Handling:** Failure: unknown/unmatched signals stay observable and trigger fail-safe uplift rather than being dropped

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Fail before runtime start when authority cannot be reproduced from one source snapshot.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not compute policy/risk/assurance from mixed commits or silently downgrade unknowns.
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

---

## Execution status

- **Status: DONE** (2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P1-02_authoritative_risk_input_derivation.md`
- Implemented: 权威风险输入推导 `intentevidence.FrozenIntent.DeriveRisk()`(新 `internal/intentevidence/riskderive.go`):从冻结意图证据**唯一**推导权威 `risk.Result` —— 确认风险提示经既有 `RiskSignalCandidates`(basis `"intent:<task-id>"`,P0-01 裁决)+ 冻结 task_type 确定性事实经新 `TaskTypeRiskCandidates`(`db_migration` ⇒ `schema_migration`(PRD §24.2 闭环按定义),basis `"tasktype:<task-id>"`;`bug_fix` ⇒ 零候选),组装候选原样交给既有 `risk.Derive` 分类合并 —— known/unknown 归类完全留在 risk 包(DL-05,扩展不复制),候选 LLM 风险输出没有任何旁路参数(只有冻结证据被读,DL-06/要求 1),unknown/未匹配信号逐字保真可见(INV-V14-09/要求 2),零值意图类型化拒收(`ErrInvalidEvidence`,绝不从非证据推导「无风险」),缺/坏 basis 在 `risk.Derive` 入口按风险不变量**整体**拒收(`risk.ErrMalformedSignal`,无部分结果/验收标准 2);纯函数、零持久化(无 RiskProfile,DL-05/要求 3),同 Revision 证据重算逐字段相同。`internal/risk`、`internal/contract`、`internal/engineeringpolicy`、`internal/artifacts` 全部零改动(DL-18)。
- Acceptance: 「Known migration/auth/protected-surface facts produce deterministic signals+basis」— PASS(`TestDeriveRiskKnownFactsProduceDeterministicSignals` 推导表:提示/任务类型/合并并集 basis/三信号齐备/unknown 保真;`TestDeriveRiskDeterministicAcrossEqualEvidence` 同 Revision 确定性);「Missing basis is rejected per risk invariants」— PASS(`TestDeriveRiskChainRejectsMissingBasis`:缺 basis 候选注入推导入口整体拒收 `ErrMalformedSignal` 且无部分结果 + DeriveRisk 产物重进 `risk.Derive` 恒等重派生;`TestDeriveRiskRejectsZeroValue`:零值拒收)。任务 Verification(风险推导表测试)即 `TestDeriveRiskKnownFactsProduceDeterministicSignals` 六用例表。
- Validation: `gofmt -l internal/intentevidence/` clean;`go test ./internal/intentevidence/ ./internal/risk/` PASS;`go test -run 'TestDeriveRisk|TestTaskTypeRiskCandidates' -v ./internal/intentevidence/` PASS(6/6);`go test ./internal/risk/ ./internal/engineeringpolicy/ ./internal/contract/ ./test/arch/` PASS(reference pack 点名的全部测试族,含 `v14_assurance_failsafe_test.go`);`go build ./...` + `go vet ./...` PASS;`go test ./internal/...` PASS(46 包);`git diff --check` clean;`make check` NOT_RUN(无 make,构成命令逐条执行);integration e2e NOT_RUN(DB-gated opt-in,本任务 Verification 为纯变换表测试)。
- Baseline failures: none。
- Decision Locks touched: none(DL-05 即本任务实现的锁:派生、无持久化;DL-06 经「只读冻结证据」继承;DL-18 遵守,既有域零改动、无平行风险体系;DL-17 遵守,hidden/ 未读)。
- Risks: 唯一确定性任务类型映射为 `db_migration → schema_migration`(扩展映射/信号枚举属任务合同级变更);`tasktype:` 与 `intent:` basis 出处区分靠文档+测试钉住(按设计是不透明字符串);P1-03 必须把 `DeriveRisk` 的类型化错误接入执行阻断(其合同 "invalid risk basis stop execution");HIGH review 待执行(重点:§2 缝选择与零值拒收语义)。
- Commit: 见本文件 git 历史(`P1-02: …`)。
