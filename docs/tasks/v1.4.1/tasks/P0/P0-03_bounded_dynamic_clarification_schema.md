# P0-03 — Bounded dynamic clarification schema

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W03
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** A0-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Extend existing Clarification/Candidates into a typed presentation schema without creating dynamic domain contracts.


## Allowed / primary paths

- `internal/intake/**`


## Implementation requirements

1. Support bounded field types required by v1.4.1.
2. Preserve strict structured AI output.
3. Allow optional natural-language escape hatch.
4. No arbitrary HTML/JS.


## Acceptance criteria

- [ ] Old simple candidate clarification remains compatible or has explicit migration.
- [ ] Unknown field kinds reject.


## Verification

- Schema parsing/validation tests; injection-shaped values rendered as data.

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
1. **Tests:** Tests: `internal/intake/propose_test.go::{TestProposeClarificationOnly,TestProposeRejectsInvalidShapes}`, `v14_launch_test.go::TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent`
2. **Contract / Interface:** Interfaces: `intake.Clarification`, `wireProposal`, strict `decodeStrict`
3. **Core Implementation:** Core: `internal/intake/intake.go`, `propose.go`
4. **Failure Handling:** Failure: reject unknown control kinds/fields; presentation data cannot become authority without mapper validation

## External reference pack
- **EXT-JSONSCHEMA — santhosh-tekuri/jsonschema — bounded declarative validation reference**  
  https://github.com/santhosh-tekuri/jsonschema  
  License/version: Apache-2.0; current repository; use as pattern only.  
  Use: Reference for strict declarative shape validation. Do NOT add this dependency unless dependency whitelist/ADR explicitly approves it.

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
- Report: `docs/tasks/v1.4.1/reports/P0-03_bounded_dynamic_clarification_schema.md`
- Implemented: `internal/intake` 的 `Clarification` 原位扩展为有界 Dynamic Clarification Schema（DL-11 / PRD §3.2/§8）。封闭词表 `ClarificationKind`（choice/multichoice/boolean/number/repository/branch/risk_choice/confirmation/free_text，精确匹配，词表外 → 新哨兵 `ErrUnknownClarificationKind` fail-safe 拒绝）；旧简单形态显式迁移（kind 缺省：带 candidates → `choice`，不带 → `free_text`，即自然语言逃生舱，AC1 兼容）；advisory `Recommended`/`RecommendationReason`（下标越界/错位拒绝，PRD §8.3）与 `number` 的 `Min`/`Max` 闭区间；数量/长度上限（8 问题/提案、16 候选/问题、500B 问题、200B 标签、500B 理由，超限 `ErrInvalidProposal`，不截断）。新文件 `clarification_schema.go` 承载校验/归一；`toProposal` 澄清分支改为调用之（原空白校验并入，消息不变）；`decodeStrict` 未放宽（未知 JSON 字段仍拒绝）；提示词增第 6 条规则声明词表与 advisory。结构中无任何 markup/脚本承载字段（无 arbitrary HTML/JS）。
- No dynamic domain contract was created: 权威字段面仍由 `Draft`/既有冻结路径承担（DL-11/DL-18 遵守，未建平行 form 包）；`confidence`/conditional follow-up（PRD §8.3）刻意未纳入——非字段类型且条件追问引入无消费者的树形结构面。
- Validation: `go test -count=1 ./internal/intake/` PASS（17 tests：4 个新测试文件测试 + 既有 13 个零改动全绿，含 pack 指定的 `TestProposeClarificationOnly`/`TestProposeRejectsInvalidShapes` 与 `TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent` 实现前后各跑一次 PASS）；`go test ./test/arch/...` PASS；`go test ./internal/... ./runtimes/...` PASS；`go build ./...` + `go vet ./...` PASS；`gofmt` clean；`git diff --check` clean。`make check` NOT_RUN(reason: integration DB 依赖，纯域包任务合同未要求)。
- Baseline failures: none.
- Decision Locks touched: none（DL-11 落实；DL-17 遵守）。
- Production reachability: N/A(owned integration tasks pending) — 现有唯一消费方 `internal/web/handlers/v14_launch.go` 的 `v14QuestionViews` 只读 Question/Candidates，对本扩展透明（回归已跑 PASS）；`Kind`/advisory 的渲染与折叠接线归 P0-04（`v14_launch.tmpl`/handler 所有权在该任务），组合根行为本任务未改。
