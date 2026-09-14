# P0-07 — Verification authoring and freeze admission

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W05
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P0-03, P0-06
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Ensure Confirm→Freeze cannot invent or omit required evidence bindings.


## Source facts

- Verification freeze requires acceptance/evidence bindings; held-out evidence has stricter semantics.


## Allowed / primary paths

- `internal/intakefreeze/**`
- `internal/verification/**`
- `internal/intake/**`


## Implementation requirements

1. Map supported acceptance/evidence declarations.
2. If evidence is missing, return a typed clarification/blocker.
3. Never auto-label model-generated evidence as held-out.


## Acceptance criteria

- [ ] Every frozen acceptance has a valid planned evidence binding.
- [ ] Missing evidence prevents execution-ready freeze.


## Verification

- Unit tests for normal, missing-evidence, held-out, invalid binding cases.

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
- **Reason:** Held-out verification and freeze admission are trust-boundary semantics; mistakes can leak or fabricate evidence.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `verification/freeze_test.go`, `types_test.go::TestFrozenImmutableAfterFreeze`, `holdout_path_leak_test.go`, `holdout_executor_test.go`
2. **Contract / Interface:** Interface: `verification.FreezePolicy`, `ValidateFreeze`, `FreezeContract`, binding visibility/authoring provenance
3. **Core Implementation:** Core: `verification/freeze.go`, `types.go`, `coverage_*`
4. **Failure Handling:** Failure: no post-hoc evidence by default; held-out identifiers/content never invented or exposed; missing required evidence blocks freeze

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
- Report: `docs/tasks/v1.4.1/reports/P0-07_verification_authoring_and_freeze_admission.md`
- Implemented: 同包扩展 `internal/intakefreeze`（P0-06 报告预留的接缝）：冻结准入入口 `Freeze(ConfirmedDraft, VerificationPlan) → FrozenArtifacts{Intent, Task, Verification}`。复用 `Map`（P0-06 全部门先行，单一映射路径不分叉）；逐条款证据声明（`EvidenceDeclaration`：条款路由 + 封闭 `verification.EvidenceType` 词表 + visible 定位 Path/Selector + **显式** `HeldOut` + store/ref）作者化为与确认条款 1:1 的 `verification.Contract`，经既有权威 `verification.FreezeContract`（零值 `FreezePolicy`，P1 standard path）冻结；`internal/verification` 零改动（只消费，DL-18）。作者化 provenance 结构上固定 `pre_commit`（确认边界不存在 post_hoc 作者行为，声明形状不可表达）；binding id 由条款 id 机械派生（`<clause>-EV<nn>`）；条款/声明按内容排序后映射（输入换序不改变冻结哈希）；等值重复声明拒绝而非去重。
- Fail-safe admission（固定顺序，全部类型化、零部分产物）：`Map` 门 → 发明防护（孤儿声明 → `ErrUnknownEvidenceTarget`）→ 重复防护（→ `ErrInvalidVerification`）→ 遗漏防护（每条款 ≥1 声明，缺失按条款 id 升序列全 → `ErrMissingEvidence`，缺失证据阻止执行就绪冻结；verification 的 ≥1 binding 规则独立兜底）→ held-out 隔离门（显式声明必须伴随完整 runtime 隔离能力要求，INV-VER-05 → `ErrHeldOutWithoutIsolation`；visible 引用 holdout store 由既有 §8.4.1 校验拒绝）→ 权威校验与冻结（→ `ErrInvalidVerification` 经 %w 透传）。held-out 绝不自动标注：唯一来源是 owner 显式 `HeldOut=true`，映射器无任何推断规则。
- Acceptance: 「Every frozen acceptance has a valid planned evidence binding」— PASS（准入强制每条款 ≥1 声明 + 权威 Validate/freeze 规则；happy-path 断言每条冻结验收携带有效绑定，且冻结哈希等于同内容直接 `verification.FreezeContract` 的哈希）。「Missing evidence prevents execution-ready freeze」— PASS（`ErrMissingEvidence` 类型化阻断 + 零产物；`Map` 文档明确其为非执行就绪的映射跳）。
- Validation: `go test -count=1 -v ./internal/intakefreeze/` PASS（19/19：11 个 P0-06 原测试不动全绿 + 8 个新测试，覆盖 normal/missing-evidence/held-out/invalid binding 四类合同要求用例与负路径：孤儿声明含大小写/无零填充变体、能力要求缺席/不完整、未知/缺失证据类型、visible 引 holdout store、等值与排版变体重复、held-out 携带 repo path、零条款 light 冻结、Map 门先行、输入隔离）；基线（改动前）`go test ./internal/verification ./internal/intakefreeze ./internal/intake ./internal/contract` PASS；`go test -count=1 ./internal/{intakefreeze,intake,intentevidence,taskcontract,verification,contract}/` PASS；`go test ./test/arch/...` PASS；`go test -count=1 ./internal/...` PASS（46 包，0 失败）；`go build ./...` + `go vet ./...` PASS；`gofmt -l internal/intakefreeze/` clean；`git diff --check` clean。`make check` NOT_RUN(reason: Git Bash 无 make；构成命令已逐条真实执行)。integration e2e NOT_RUN(reason: opt-in DB-gated;任务 Verification 只要求单元测试)。
- Baseline failures: none。
- Decision Locks touched: none（DL-05/DL-06/DL-17/DL-18 遵守；锁文件未动；hidden/ 未读；held-out 边界保持，holdout 泄漏套件绿）。
- Risks: 独立性（independence_required）作者化刻意未实现——确认输入没有独立性声明通道，准入冻结保守 none 默认，显式等级需先有确认侧通道（后续任务/产品决策）；protection+baseline_hash 不在声明面（完整性锚点属制品字节，现在接受 owner 手填哈希会邀请发明锚点）；生产确认 UI 尚未采集证据声明、`Freeze` 目前仅测试可达——按设计（`v14_launch.go` 归 P0-04→P0-08 串行，接线归 P0-08）。
- Commit: 见本文件 git 历史（`P0-07: verification authoring and freeze admission — ...`）。
