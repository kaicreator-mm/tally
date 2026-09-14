# P0-04 — Render dynamic forms in v1.4 Launch UI

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W05
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P0-03
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Render typed LLM-generated clarification choices with familiar form controls.


## Allowed / primary paths

- `internal/web/handlers/v14_launch.go`
- `web/templates/v14_launch.tmpl`
- `web/static/**`


## Implementation requirements

1. Recommended option is advisory.
2. Free-text escape hatch is optional.
3. Server revalidates all submitted values.


## Acceptance criteria

- [ ] No raw generated HTML is executed.
- [ ] Keyboard/basic accessibility works.


## Verification

- Handler/template tests for all supported field kinds.

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
1. **Tests:** Tests: `internal/web/handlers/v14_launch_test.go` clarification/draft cases; `execution_http_test.go::TestActionFormsArePlainNoJSUsable`
2. **Contract / Interface:** Interface: `V14QuestionView`, `v14QuestionViews`, server-side form parsing
3. **Core Implementation:** Core: `internal/web/handlers/v14_launch.go`, `web/templates/v14_launch.tmpl`
4. **Failure Handling:** Failure: render model-originated strings as escaped data; server must revalidate submitted option IDs/values

## External reference pack
- **EXT-GO-HTML — Go html/template**  
  https://pkg.go.dev/html/template  
  License/version: BSD-3-Clause (Go project); Go standard library.  
  Use: Context-aware escaping reference for rendering LLM-originated values as data, never raw HTML.
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
- Report: `docs/tasks/v1.4.1/reports/P0-04_render_dynamic_forms_in_v1_4_launch_ui.md`
- Implemented: P0-03 的有界 clarification schema 在 v1.4 Launch UI 完整渲染与折叠接线。全部 9 种 `ClarificationKind` 按类型渲染为原生控件（choice/repository/branch/risk_choice → radio 单选；multichoice → checkbox；boolean → 是/否；confirmation → 确认/否决；number → `type=number` + min/max 属性与标签提示；free_text → 文本输入；旧简单形态照旧）；advisory 推荐**只标注不预选**（`(推荐)` 徽标 + 理由提示，全页无 `checked`）；除 number/free_text 外每问带可选自由文本逃生舱（填写优先于选项，有标注）。提交侧以隐藏域携带渲染 schema，`v14FoldClarifications` 先 fail-safe 校验 schema（词表外 kind/区间倒挂/坏界值 → 类型化 400）再按 kind 复核每个提交值（候选集成员、规范值、数值可解析且在区间、答案 ≤500B、多选 ≤16 项），复核失败保留问题与 schema 重渲染且绝不产出草案；未携带 kind 的旧形态表单保持既有折叠行为（既有测试零改动）。无 `template.HTML`、无脚本、无 `hx-*`（原生整页 POST，关 JS 可用）；`fieldset/legend` 分组 + label 关联（基础可访问性）；CSS 仅 +4 条规则，无新框架/组件体系。
- Acceptance: No raw generated HTML executed — PASS（`TestV14ClarificationRenderedValuesStayEscapedData`：`<script>`/`onerror` 形态值以转义数据呈现）；Keyboard/basic accessibility — PASS（fieldset/legend、包裹式与显式 label、原生控件，渲染断言在 `TestV14ClarificationFormsRenderAllFieldKinds`）。Implementation requirements 1/2/3 全部落实并有测试。
- Validation: `go test -count=1 -run 'TestV14' ./internal/web/handlers/` PASS；`go test -count=1 ./internal/web/handlers/` PASS；`go test ./test/arch/...` PASS；`go test ./internal/... ./runtimes/...` PASS；`go test -count=1 ./internal/intake/ ./internal/measurement/` PASS；`go build ./...` + `go vet ./...` PASS；gofmt clean（本任务文件）；`git diff --check` clean。`make check` NOT_RUN(reason: integration DB `localhost:5432` 不可达；合同只要求 handler/template 测试，已覆盖)。
- Baseline failures: none（pack 指定测试实现前即绿，实现后全绿；handlers 包其它文件的 gofmt 标记为 baseline 既有）。
- Decision Locks touched: none（DL-11 落实；A-09 遵守；DL-17 遵守）。
- Shared hotspots: `internal/web/handlers/server.go` 零接触（推理出口沿用既有 `SetV14ProposeExecutor` 注入缝）。
