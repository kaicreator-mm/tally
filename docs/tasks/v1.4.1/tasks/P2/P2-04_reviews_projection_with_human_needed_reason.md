# P2-04 — Reviews projection with human-needed reason

- **Phase:** P2
- **Kind:** implementation
- **Wave:** W17
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P2-01, P2-02
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 需人工事项聚合投影落地 `/v14/attention`（复用既有 attention 事实面，零新来源）：每条评审项新增固定 **目标动作** `Action`（封闭 per-kind switch —— needs_review→人工裁决评审请求、blocked→解除阻塞/补充依据、failed_run→修复重试/换基线/放弃(不自动重试)、cancelled→发起新任务(终态无解除)、exposure→waiver/换基线/放弃 strict 主张、projection_error→先修事实面），动作文本只复述本页既有边界声明中的人本裁决项；页面新增 **聚合小计** `Summary`（按类型排序位统计非零类别，"聚合 N 项需人工事项:类型 ×count…"）；h1 Attention→需人工评审（meta 保留 PRD §26.4 引用）。`internal/review/**`（hunk 记账语义）与 `internal/approval/**` 零触碰（要求 2）；无第二可变状态、无写路由（DL-19）。
- AC：每条评审项携带 reason/basis/目标动作 — PASS。三面投影测试 `TestV14AttentionReviewItemsCarryReasonBasisAndAction`：no-review（干净主线零条目且**零动作字面**——表头/动作/聚合行只随条目渲染，负断言钉死无中生有）/ exception（终态失败 Run，失败原因 assurance_insufficient 原样 + 固定动作）/ high-risk（在账暴露 + waiver 裁决动作）；聚合小计按封闭词表计数且未发生类别不出现。既有 `TestV14AttentionListsBlockingFactsOnly` 全部断言不变绿（含其禁词表）。
- Verification：DB-gated 全 `./internal/web/...` PASS（170.6s）；`./test/arch/...` PASS；gofmt/`git diff --check` 清洁。PRODUCTION_REACHABILITY: PASS（既有生产路由 `/v14/attention` + 导航条目，HTTP 实测）。Risks：needs_review/blocked/cancelled 分支在 v13 读路径当前不可达（闭环升级的事实面是终态失败 Run；review 请求尚无可读取事实面）——动作映射仍封闭就绪，事实面接入即生效；「需人工」导航标签维持（改名议题裁定为不必要：AC 未要求且无支撑能力暗示）。执行过程：平台模型请求故障持续，由编排者主会话完成（第 4 次子智能体派发失败后）。Full evidence: `reports/P2-04_reviews_projection_with_human_needed_reason.md`。


## Goal

Aggregate exception/high-risk/pending decisions and explain why human input is required.


## Allowed / primary paths

- `internal/web/handlers/**`
- `web/templates/**`
- `internal/approval/**`
- `internal/review/**`


## Implementation requirements

1. Reuse existing approval/review/attention facts.
2. Do not rename existing `internal/review` semantics into generic governance.


## Acceptance criteria

- [ ] Every listed review item includes reason/basis and target action.


## Verification

- Projection tests across exception/high-risk/no-review.

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
1. **Tests:** Tests: `v14_attention_test.go::TestV14AttentionListsBlockingFactsOnly`, `v13_http_test.go::TestV13ApproveWritesHumanApprovalEvidenceNotStatus`
2. **Contract / Interface:** Interface: approval/attention/review evidence already present
3. **Core Implementation:** Core: `internal/approval/*`, `v14_attention.go`, `internal/review/*` only for its existing hunk-accounting meaning
4. **Failure Handling:** Failure: every review item must explain the blocking basis; no generic manual status override

## External reference pack
- **EXT-GO-HTML — Go html/template**  
  https://pkg.go.dev/html/template  
  License/version: BSD-3-Clause (Go project); Go standard library.  
  Use: Context-aware escaping reference for rendering LLM-originated values as data, never raw HTML.

## Do

- Build read-only projections over existing facts/state; preserve no-manual-status discipline.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not create a second mutable task/run/review state just to simplify UI.
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
