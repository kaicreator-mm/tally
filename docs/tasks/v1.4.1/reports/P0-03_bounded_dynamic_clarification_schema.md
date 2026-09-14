# P0-03 — Bounded dynamic clarification schema: Execution Report

- **TASK:** P0-03
- **STATUS:** DONE
- **Date:** 2026-09-12
- **Phase / Wave:** P0 / W03
- **Execution/Review model:** LOW / LOW (per task pack; review rule "escalate only on source/API mismatch" — no mismatch encountered)
- **Depends on:** A0-05 (READY_TO_IMPLEMENT) — satisfied
- **Hidden material:** none read; `docs/tasks/v1.4.1/hidden/` untouched (DL-17)
- **Authority chain read in order:** task contract → `DECISION_LOCKS.md` (DL-11 binding; DL-18 extend-not-duplicate) → `PRD_AMENDMENTS.md` (A-09) → `SOURCE_BASELINE.md` → current local source → L3 reference pack

## 1. Deliverable

`internal/intake` 的 `Clarification` 被扩展为**有界 Dynamic Clarification Schema**
（DL-11 / PRD §3.2 受限 Presentation Schema / PRD §8）——Intent → bounded
Dynamic Form 的 schema 层。没有任何新领域契约：权威字段面仍由 `Draft` 与
既有冻结路径承担；本层只收敛"展示什么、怎么收敛成合法数据"。

### 封闭类型词表（PRD §3.2 全词表，`ClarificationKind`）

`choice` / `multichoice` / `boolean` / `number` / `repository` / `branch` /
`risk_choice` / `confirmation` / `free_text`。

- `free_text` 即自然语言逃生舱（PRD §8.1：Natural Language = Intent Input +
  Escape Hatch；对应 §3.2 的 OptionalFreeText）。
- 词表是封闭枚举、精确匹配：**任何其它 kind 值 → `ErrUnknownClarificationKind`
  fail-safe 拒绝**（AC2；无 untyped custom control，PRD §8.4）。

### 有界结构（bounded presentation schema）

- advisory 推荐项：`Recommended *int`（候选下标，越界拒绝）+
  `RecommendationReason`（必须伴随 `Recommended`；PRD §8.3，建议不是决定）。
- `number` 的 `Min`/`Max` 可选闭区间（倒挂拒绝）。
- 数量/长度上限：问题 ≤ 8 / 提案，候选 ≤ 16 / 问题，问题 ≤ 500 B，候选标签
  ≤ 200 B，推荐理由 ≤ 500 B；超限一律 `ErrInvalidProposal`，不静默截断。

### 旧简单形态的显式迁移（AC1，兼容而非破坏）

`kind` 缺省时按形态归一：带 `candidates` → `choice`；不带 → `free_text`。
旧版 `{question, candidates}` 与旧开放问题逐字段原样保留，仅补齐解析后的
`Kind` —— `TestProposeClarificationOnly`、`TestProposeRejectsInvalidShapes`、
`TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent` 等既有测试零改动通过。

### 严格结构化 AI 输出保持（需求 2）

`decodeStrict` 未放宽：未知 JSON 字段、尾随内容仍拒绝
（`TestClarificationShapeViolationsRejected/未知_JSON_字段仍拒绝`）。
提示词新增第 6 条规则，向推理显式声明封闭词表、迁移规则与 advisory 语义。

### 无 arbitrary HTML/JS（需求 4）

`Clarification` 结构中不存在任何 markup/脚本承载字段；问题、候选、推荐理由
都是纯文本数据，渲染由 html/template 按数据转义（现有 handler 未改动，渲染
面归 P0-04）。

## 2. Changed files

| File | Change |
|---|---|
| `internal/intake/intake.go` | +`ClarificationKind` 封闭词表与 `clarificationKindOf`；`Clarification` 增 `Kind`/`Recommended`/`RecommendationReason`/`Min`/`Max`（advisory/区间，DL-11 注释）；新哨兵 `ErrUnknownClarificationKind` |
| `internal/intake/clarification_schema.go` | **新增**：有界上限常量、`validateClarifications`（迁移 + 词表 + 形状 + 越界校验，fail-safe，返回归一副本） |
| `internal/intake/propose.go` | `toProposal` 澄清分支改为调用 `validateClarifications`（原逐项空白校验并入，错误消息不变）；`wireProposal` 注释更新；`intakePrompt` 增第 6 条规则 |
| `internal/intake/clarification_schema_test.go` | **新增**：本任务全部新行为的聚焦测试（含负例） |

全部变更位于 Allowed Paths `internal/intake/**` 之内；§10 热点零接触。

## 3. Semantic changes

1. 旧候选澄清/开放问题在解析后被赋予解析后的 `Kind`（行为兼容：字段值、
   错误分类、handler 投影 `v14QuestionViews` 均不变——handler 只读
   Question/Candidates，新字段对其透明，渲染扩展归 P0-04）。
2. 词表外 kind、类型不匹配形状（选项类缺候选 / 无候选类带候选）、
   advisory 越界/错位、`min > max`、数量/长度超限：从"静默接受"变为
   类型化拒绝（本任务引入的新行为）。
3. 推理提示词多了第 6 条规则（声明词表与上限）；`OutputShape` 顶层样例
   行为不变（`clarifications: null`，词表契约由提示词承载）。

## 4. Tests / validation（exact commands and results）

新增测试（`internal/intake/clarification_schema_test.go`）：

- `TestClarificationLegacyShapeMigratesCompatibly` — AC1 正例
- `TestClarificationBoundedKindsAccepted` — 有界词表 + advisory + 提示词契约
- `TestClarificationUnknownKindRejected` — AC2 负例（`dropdown`/`html`/
  `CHOICE`/`custom_widget`/尾随空白均拒）
- `TestClarificationShapeViolationsRejected` — 12 个形状负例 + 数量超限
- `TestClarificationInjectionValuesStayData` — 合同验证项：`<script>`/
  `onerror`/SQL 注入形态的值原样作为数据携带，不执行、不改写

Commands（全部实际执行）：

| Command | Result |
|---|---|
| `go test -count=1 -run 'TestProposeClarificationOnly\|TestProposeRejectsInvalidShapes' ./internal/intake/`（实现前 baseline） | PASS |
| `go test -count=1 ./internal/intake/`（实现前 baseline；实现后全包） | PASS / PASS（17 tests, 0 fail） |
| `go test -count=1 -run TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent ./internal/web/handlers/`（实现前） | PASS |
| `go test ./test/arch/...` | PASS |
| `go test -count=1 ./internal/web/handlers/ ./internal/measurement/` | PASS |
| `go test ./internal/... ./runtimes/...` | PASS（仅 `core`/`store` no-test-files 通知） |
| `go build ./...` | PASS |
| `go vet ./...` | PASS（无输出） |
| `gofmt -l internal/intake/` | clean（无输出） |
| `git diff --check` | clean |

`make check`（integration DB 依赖）未运行：NOT_RUN(reason: 本任务为纯域包
schema 变更，任务合同未要求 integration 验证；既有焦点测试 + arch 守卫已覆盖)。

## 5. Baseline failures observed

None. 实现前的参考测试与全包测试全绿；实现后无新增失败。

## 6. Remaining risks

- `Clarification` 的新字段（kind/advisory/区间）已进入 wire 契约
  （`decodeStrict` 现在接受这些字段名）；P0-04 起的渲染与折叠 UI 尚未消费
  `Kind`/`Recommended` —— 这是所有权内的后续接线，不是缺口；在接线前，
  推理若返回纯 `kind` 差异（如 boolean），行为与旧版一致（照常展示问题与
  候选/自由回答）。
- 上限数值（8 问题/16 候选/500B 等）是工程判断，PRD 未固定具体数字；都
  是常量、拒绝而非截断，调整不影响调用方。
- `confidence` 与 conditional follow-up（PRD §8.3 提及）**刻意未纳入**：
  它们不是字段类型，且条件追问会引入树形深度/无消费者的结构面（§11：不为
  单一推测消费者建抽象）。若 P0-04+ 需要再由其所属任务扩展。

## 7. Decision Locks touched

None. DL-11 被直接落实（bounded presentation schema、无 HTML/JS、无动态领域
schema）；DL-18（扩展现有域而非复制）遵守 —— `Clarification` 是原位扩展，
没有新建平行的 form 包。未修改任何锁文件。

## 8. Commit

recorded in the task-file addendum / final message.

## 9. L3 completion evidence（pack 要求项）

- **Exact source/tests read:** `internal/intake/intake.go`、`propose.go`、
  `propose_test.go`（pack 指定的 `TestProposeClarificationOnly` /
  `TestProposeRejectsInvalidShapes`）；`internal/web/handlers/v14_launch.go`
  （pack 指定测试的消费方，确认新字段对现有投影透明）及其
  `TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent`；
  `internal/measurement/v14_metrics.go`（clarification 仅为缺口注释，无消费）。
- **External reference actually used:** none —— EXT-JSONSCHEMA
  (santhosh-tekuri/jsonschema) 仅作为"严格声明式形状校验"的模式参照，
  未复制代码、未引入依赖（pack 明令不得加依赖；std lib `encoding/json`
  的 `DisallowUnknownFields` 已是现有严格模式）。
- **Why this seam is the narrowest valid seam:** pack 指定 Interface 即
  `intake.Clarification` / `wireProposal` / `decodeStrict`；任务 Goal 即
  "Extend existing Clarification/Candidates"。在 `toProposal` 的澄清分支
  （解析后、类型化 Proposal 之前）插入校验/归一，是唯一的必经收敛点：
  wire 严格性保留、领域返回值已归一、handler/存储无需感知。无平行子系统。
- **Negative tests added:** `TestClarificationUnknownKindRejected`、
  `TestClarificationShapeViolationsRejected`（12 形状 + 2 数量超限）、
  注入值正例中的显式迁移断言。
- **Known uncertainty / version sensitivity:** 无新依赖；纯 stdlib。上限
  常量如上（工程判断，PRD 未固定）。
- **Decision Locks not bypassed:** confirmed（见 §7）。
