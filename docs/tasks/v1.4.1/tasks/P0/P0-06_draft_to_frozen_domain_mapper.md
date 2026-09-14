# P0-06 — Draft-to-frozen domain mapper

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W05
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P0-01, P0-03
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Map a confirmed structured draft to FrozenIntent and TaskContract using pure deterministic code.


## Allowed / primary paths

- `internal/intakefreeze/**`
- `internal/intake/**`
- `internal/contract/**`


## Implementation requirements

1. No Git writes in pure mapper.
2. Stable IDs/revisions under identical input.
3. Reject unsupported task semantics.


## Acceptance criteria

- [ ] Mapper output is deterministic and contains no unresolved presentation-only fields.


## Verification

- Golden tests; invalid drafts fail before artifact commit.

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
1. **Tests:** Tests: `internal/taskcontract/{parse_test,validate_test,types_test}.go`, `verification/freeze_test.go`
2. **Contract / Interface:** Interfaces: `intake.Draft`, `taskcontract.TaskContract`, `verification.Contract`
3. **Core Implementation:** Core: `taskcontract.Parse/Validate/RevisionHash`, `verification.FreezeContract`
4. **Failure Handling:** Failure: pure mapper must reject presentation-only unresolved values and invalid scope/acceptance semantics

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

## Do

- Keep presentation, intent evidence and frozen engineering artifacts explicitly separated.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not let LLM-generated UI/risk text become authority without deterministic validation/freeze.
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

---

## Execution status

- **Status: DONE** (2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P0-06_draft_to_frozen_domain_mapper.md`
- Implemented: 新增纯领域映射包 `internal/intakefreeze`（合同 Allowed Path；P0-01 报告预留、P0-07/P0-08 Allowed Paths 共有的接缝）：`Map(ConfirmedDraft) → Frozen{Intent intentevidence.FrozenIntent, Task taskcontract.TaskContract}`。输入显式化（TaskID/Profile/SourceRef/确认后 Draft/RiskHints）；FrozenIntent 只经 P0-01 唯一冻结入口 `intentevidence.Freeze(Facts)` 产出（Draft→Facts 映射归本任务，P0-01 包文档钦定）；TaskContract 逐字段显式赋值后过既有权威 `Validate` + `ValidateStatic`（invalid drafts fail before artifact commit），哈希按内容重算（taskcontract.Parse 同纪律）。确定性：无时间/随机/Git 写入/存储；切片顺序属排版（Freeze 规范化 + RevisionHash 排序去重消化）；映射不改输入、不共享底层数组。fail-safe 类型化拒绝：`ErrInvalidProfile`（治理输入不设默认）、`ErrUnsupportedTaskType`（显式两臂 switch 翻译，词表漂移即编译失败点）、`ErrUnsupportedDraftSemantics`（non_scope/constraints 在冻结 schema 无归宿——拒绝而非静默丢弃，杜绝「确认了什么」与「生效了什么」分叉）、`ErrInvalidContract`（权威校验失败经 %w 透传）；intentevidence 哨兵可 errors.Is 穿透。Draft.RiskSignals 候选绝不因确认自动升级为证据（DL-06）；验收条款独立性留空按 none 保守默认（治理属性不由映射发明，归 P0-07 准入）。
- Acceptance: 「Mapper output is deterministic and contains no unresolved presentation-only fields」 — PASS（`TestMapHappyPathDeterministicGolden` 同输入两次映射 DeepEqual + 字段逐一对照 + YAML 往返经 `taskcontract.Parse` 重载出同一内容哈希；`TestMapCanonicalizesOrderingVariants` 提示/边界/条款换序与重复提示 → 同一修订；`TestMapFailureOrderIsDeterministic` 校验顺序固定；advisory/不可承载内容零进入产物，见负路径组）。
- Validation: `go test -count=1 -v ./internal/intakefreeze/` PASS（11/11，含 8 组负路径：profile 词表、任务类型大小写变体、non_scope/constraints 三种形态、证据哨兵透传、六种合同语义非法、失败顺序、输入隔离、未确认候选不泄入）；基线 `go test ./internal/taskcontract ./internal/verification ./internal/intake ./internal/intentevidence ./internal/contract ./internal/risk` PASS（改动前）；`go test ./test/arch/...` PASS；`go test -count=1 ./internal/...` PASS（46 包，0 失败）；`go build ./...` + `go vet ./...` PASS；`gofmt -l internal/intakefreeze/` clean；`git diff --check` clean。`make check` NOT_RUN(reason: Git Bash 无 make 可执行文件；其构成的 arch/internal 两条等价命令已逐条真实执行)。integration e2e NOT_RUN(reason: 任务 Verification 只要求 golden tests + 提交前拒绝，集成路径为 opt-in DB-gated)。
- Baseline failures: none。
- Decision Locks touched: none（DL-05/DL-06/DL-11/DL-18 遵守；锁文件未动；hidden/ 未读，DL-17）。
- Risks: non_scope/constraints 拒绝是产品可见的冻结闸（owner 须在确认 UI 把这类内容解析进 objective/scope/acceptance 或删除；若未来要承载属 TaskContract schema 变更，归 P1-03/ADR）；`TBD` 类占位文本启发式检查刻意不在本边界（requirement linter 惯例，归 P0-07 准入）；生产接线归 P0-08，本任务 N/A(owned integration tasks pending)。
- Commit: 见本文件 git 历史（`P0-06: draft-to-frozen domain mapper — ...`）。
