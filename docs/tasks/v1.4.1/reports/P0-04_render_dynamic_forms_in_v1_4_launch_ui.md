# P0-04 — Render dynamic forms in v1.4 Launch UI: Execution Report

- **TASK:** P0-04
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W05
- **Execution/Review model:** LOW / LOW (per task pack; review rule "escalate only on source/API mismatch" — no mismatch encountered)
- **Depends on:** P0-03 (bounded Dynamic Clarification Schema, commit 1677bf1) — satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-11 binding; A-09) → `PRD_AMENDMENTS.md` (A-09 UI = projection, dynamic forms are bounded presentation objects) → `SOURCE_BASELINE.md` → `PATH_OWNERSHIP.md` → current local source → L3 reference pack

## 1. Deliverable

P0-03 的有界 clarification schema（封闭 `ClarificationKind` 词表 / advisory
`Recommended`+`RecommendationReason` / `number` 的 `Min`/`Max`）在 v1.4 Launch
页（`v14_launch.tmpl` + `v14_launch.go`）完成渲染与折叠接线。UI 只是把
schema 呈现为数据与原生表单控件（DL-11 / PRD §3.2 / §8.4），没有发明任何新
状态权威、领域契约或前端框架。

### 每种字段类型的呈现（全部 9 种，无 schema 外形态进入 UI）

| Kind | 控件 | 说明 |
|---|---|---|
| `choice` / `repository` / `branch` / `risk_choice` | radio 单选 | 逐候选一个 radio；推荐项带 `(推荐)` 标注 |
| `multichoice` | checkbox 多选 | 逐候选一个 checkbox；推荐项同样只标注 |
| `boolean` | radio `是` / `否` | 固定规范值 |
| `confirmation` | radio `确认` / `否决` | 固定规范值 |
| `number` | `input type=number` + `min`/`max` 属性 | 声明区间同时渲染为标签提示与 input 约束 |
| `free_text` | `input type=text` | 自然语言逃生舱（PRD §8.1）成为一等字段类型 |
| 旧简单形态（Kind 空） | radio 候选 + 自由作答 | 与 P0-03 前行为一致 |

### Advisory 语义（实现需求 1）

推荐项**只标注、不预选**：选项旁 `(推荐)` 徽标 + 问题底部「推荐 X:理由
(仅供参考,建议不是决定)」提示；模板与渲染结果中没有任何 `checked` 属性，
提交值永远来自 owner 显式选择（测试断言全页无 `checked`）。

### 自由文本逃生舱（实现需求 2）

除 `number`/`free_text`（主输入本就是自由值）外，每种问题都带可选的
「其他(填写后优先于选项)」输入；填写即覆盖结构化选项（与既有折叠语义一致，
输入框有明确标注）。

### 服务端重校验（实现需求 3）

澄清表单以隐藏域携带上一轮渲染的 schema（`v14k.*` kind、`v14c.*` 候选集、
`v14min/v14max.*` 区间、`v14r/v14rr.*` advisory）。提交时
`v14FoldClarifications` 先对 schema 本身做 fail-safe 校验，再按 kind 复核每
个提交值：

- kind 必须在封闭词表内（词表外 → 类型化 400 拒绝，DL-11）；
- 单选/多选值必须 ∈ 渲染时声明的候选集；多选 ≤ 16 项；
- `boolean`/`confirmation` 只接受规范值（是/否、确认/否决）；
- `number` 必须可解析为数字且落在声明区间；携带区间倒挂也拒绝；
- 每个答案 ≤ 500 B（超限拒绝，不截断）；
- 未携带 `v14k.*` 的旧形态表单保持既有折叠行为（自由文本优先，选项原样），
  既有测试零改动通过。

复核失败：400 + 类型化原因，问题与 schema 从隐藏域重建后原样重渲染
（候选/区间/推荐提示不丢，可直接重答）；复核失败绝不产出草案。

### 无 arbitrary HTML/JS（AC: No raw generated HTML is executed）

全部模型来源文本（问题/候选/推荐理由）与提交回填值经 html/template 上下文
转义按数据呈现；代码中无 `template.HTML`、无任何 markup/脚本承载字段。表单
是原生整页 POST（无 `hx-*` 属性），关 JS 完全可用（与
`TestActionFormsArePlainNoJSUsable` 同一合同）。

### 可访问性（AC: Keyboard/basic accessibility）

每个问题是一个 `fieldset` + `legend` 分组（屏幕阅读器按组播报问题），选项
label 包裹 input（原生键盘可达），number/text 输入用显式 `label for` + `id`
关联，类型以人读标签 `[单选]` 等标注。CSS 仅追加 fieldset/legend/选项标签
四条规则（沿用既有 `.rec` / `.v14-form` 体系，无新组件体系）。

## 2. Changed files

| File | Change |
|---|---|
| `internal/web/handlers/v14_launch.go` | `V14QuestionView` 扩展（Kind/KindLabel/advisory 扁平化/number 区间；指针 → Has* 布尔，模板不按下标取值）；`v14QuestionViews` 完整投影 + `v14KindLabel`；`v14FoldClarifications` 重写为按 kind 的服务端复核 + schema 重建（`v14ClarificationForm`/`v14ClarificationFormOf`/`v14ClarificationAnswer`/`v14BoundedAnswer`/`v14InCandidates`，上限常量 500B/16 项） |
| `web/templates/v14_launch.tmpl` | 澄清区改为 per-kind 渲染：fieldset/legend 分组、radio/checkbox/number/text 控件、advisory 标注（不预选）、number 区间属性、可选自由文本逃生舱、schema 隐藏域携带 |
| `web/static/tally.css` | +4 条规则：`.v14-form fieldset/legend/.opt`（可访问分组 + 选项标签正文观感），无新组件 |
| `internal/web/handlers/v14_launch_test.go` | 新增 4 个测试（含 8 个负例子用例）；既有测试零改动 |

全部变更位于任务 Allowed Paths（`v14_launch.go` / `v14_launch.tmpl` /
`web/static/**`）之内；`server.go` 等共享热点零接触（推理出口沿用既有
`SetV14ProposeExecutor` 包级注入缝）。测试文件是任务合同 Verification 条
（"Handler/template tests for all supported field kinds"）指定的承载处
（reference pack Tests 即该文件），在既有文件内追加、未改既有用例。

## 3. Semantic changes

1. 澄清问题从「问题 + 平铺候选」升级为按封闭词表的类型化表单；advisory
   推荐与 number 区间首次在 UI 可见。
2. 折叠提交从「自由文本原样并回」升级为「schema 服务端复核后才折叠」；
   词表外/越界/超长/非规范值从静默接受变为类型化 400（新行为）。
3. 复核失败重渲染从「只剩问题文本」升级为「问题 + 完整 schema 重建」。
4. 无任何路由/状态权威/冻结语义变化（A-09：UI 是投影）。

## 4. Tests / validation（exact commands and results）

新增测试（`internal/web/handlers/v14_launch_test.go`）：

- `TestV14ClarificationFormsRenderAllFieldKinds` — 全部字段类型的控件/区间/
  advisory/fieldset/逃生舱正例；断言无 `checked`（advisory 不预选）、无
  `hx-*`（关 JS 可用）
- `TestV14ClarificationRenderedValuesStayEscapedData` — 合同 AC 负例：含
  `<script>`/`onerror`/`</fieldset><script>` 形态的问题/候选/理由必须以转义
  数据出现，不得以可执行形态出现
- `TestV14ClarificationFoldRevalidatesAndFoldsPerKind` — 9 问提交（含多选
  多值、number 区间内值、自由文本覆盖选项）逐一复核后折叠进新意图，prompt
  逐条断言
- `TestV14ClarificationFoldRejectsInvalidAnswers` — 8 个负例：数值越界/
  非数字、词表外 kind、单选/多选不在候选集、boolean 非规范值、携带区间倒挂、
  未作答要求重答；另断言复核失败保留问题与 schema 供重答且不产出草案

| Command | Result |
|---|---|
| `go test -count=1 -run 'TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent\|TestV14ProposeRendersEditableDraftAndConfirmBoundary' ./internal/web/handlers/`（实现前 baseline） | PASS |
| `go test -count=1 -run 'TestV14' ./internal/web/handlers/` | PASS |
| `go test -count=1 ./internal/web/handlers/` | PASS（含 DB-gated 全链路用例，Postgres `localhost:5432` 不可达时 SKIP，与本任务改动无关） |
| `go test ./test/arch/...` | PASS |
| `go test ./internal/... ./runtimes/...` | PASS |
| `go test -count=1 ./internal/intake/ ./internal/measurement/`（消费方回归） | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS（无输出） |
| `gofmt -l` on `v14_launch.go` / `v14_launch_test.go` | clean（handlers 包内其它文件的 gofmt 标记为 baseline 既有，未触碰） |
| `git diff --check` | clean |

`make check`（integration 依赖）NOT_RUN(reason: Postgres 测试库指向
`localhost:5432` 不可达；任务合同只要求 handler/template 测试，已由聚焦
HTTP 测试覆盖，全链路 DB 用例与澄清渲染无关)。

## 5. Baseline failures observed

None. 实现 pack 指定测试（`v14_launch_test.go` clarification/draft 用例）在
实现前即绿；实现后全绿。`gofmt -l` 在 handlers 包列出的其它文件为 baseline
既有状态（stash 验证），非本任务引入。

## 6. Remaining risks

- 提交侧 schema 由隐藏域携带：剔除隐藏域可让问题退回旧形态折叠（值仍是有界
  纯文本、经 intake 再结构化、绝不经 UI 成为权威），没有权限提升面；词表外
  kind 等篡改一律 fail-safe 拒绝。若未来要求更强绑定，需要服务端暂存每轮
  schema（引入会话状态），超出本任务边界。
- 复核失败重渲染只保留失败/未答的问题（与既有「未答重答」行为一致）；已答
  问题的值不回填 —— 多问长表单下重答成本略高，可由后续 UX 任务改进。
- number 以 `step="any"` 渲染（支持小数）；min/max 用 Go 默认浮点文本
  （如 `3.5`），极端量级（`1e+06`）依赖 HTML number 的指数语法，现代浏览器
  均接受。

## 7. Decision Locks touched

None. DL-11 被直接落实（bounded presentation schema 渲染、无 HTML/JS 执行
面、无动态领域 schema）；A-09（UI 是投影、动态表单是 bounded presentation
object）遵守；DL-17 遵守。未修改任何锁文件；`PATH_OWNERSHIP.md` 边界内施工
（`v14_launch.go` P0-04 primary，`v14_launch.tmpl` exclusive，static 允许）。

## 8. Commit

recorded in the task-file addendum / final message.

## 9. L3 completion evidence（pack 要求项）

- **Exact source/tests read:** pack 指定测试 `internal/web/handlers/v14_launch_test.go`
  （clarification/draft 用例）与 `execution_http_test.go::TestActionFormsArePlainNoJSUsable`；
  指定接口/实现 `V14QuestionView`/`v14QuestionViews`/`internal/web/handlers/v14_launch.go`/
  `web/templates/v14_launch.tmpl`；上游 `internal/intake/intake.go`
  （`Clarification`/`ClarificationKind`）与 `clarification_schema.go`
  （上限/归一保证）、`propose.go`（`decodeStrict`/`toProposal`/`hasDraftFields`）；
  `web/static/tally.css`、`web/templates/base.tmpl`（渲染体系）。
- **External reference actually used:** none —— EXT-GO-HTML
  (html/template) 即 stdlib 现有渲染机制（上下文感知转义为既有事实，未引入
  新依赖）；EXT-JSONSCHEMA 仅作声明式校验模式参照，未引入依赖
  （`test/arch/deps_test.go` 白名单未动）。
- **Why this seam is the narrowest valid seam:** pack 指定 Interface 即
  `V14QuestionView`/`v14QuestionViews` + server-side form parsing；渲染必须
  落在唯一既有消费方（v14 launch 页）与其模板，折叠必须落在唯一既有收敛点
  （`v14FoldClarifications`，`V14ProposeIntent` 调 `intake.Propose` 前的必经
  路）。schema 隐藏域是该收敛点上最小可用的复核依据（提交侧无权威 schema
  可查、每轮澄清由推理新生成）；无平行子系统、无新路由、无新框架。
- **Negative tests added:** `TestV14ClarificationFoldRejectsInvalidAnswers`
  （8 例）+ `TestV14ClarificationRenderedValuesStayEscapedData`（注入形态
  数据化）+ advisory 不预选断言（无 `checked`）。
- **Known uncertainty / version sensitivity:** 无新依赖；纯 stdlib +
  既有模板/CSS 体系。风险节所列隐藏域绑定与浮点文本渲染为已知边界。
- **Decision Locks not bypassed:** confirmed（见 §7）。
