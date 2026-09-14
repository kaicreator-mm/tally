# P2-05 — Human-centered UX gate

- **Phase:** P2
- **Kind:** gate
- **Wave:** W18
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P2-02, P2-03, P2-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **门禁裁决:PASS**(零生产代码改动、零缺陷发现)。三条 AC 可执行证明于 `test/e2e/v141_p205_ux_gate_test.go::TestP205HumanCenteredUXGate`(DB-gated,生产 RegisterRoutes 原样装配 + 真实内容仓库 + 真实事实面):AC1 结构化优先 —— launch 唯一入口是有界意图表单（`<label for="v14-intent">意图(一句自然语言,唯一必填项)</label>` + 服务端澄清闭环）,无 chat-first 流;评审面 = 工作/任务详情/需人工聚合页。AC2 —— label[for] 全覆盖、表格 th 表头（静态模板断言）、四页零空文本链接（键盘死角正则负断言）、nav aria-current 复核。AC3 —— 渲染面恰一个静态 htmx script 引用（零内联、零模板插值,四页渲染 + 五模板双断言）,LLM 来源值全部 html/template 转义作数据。门禁目标 —— 导航轨道零生命周期机器词面（NEEDS_REVIEW/VALIDATING/AWAITING_APPROVAL/CODEGEN/CHANGESET 负断言）,人读标签主导（发起/工作/运行/需人工）。
- Verification:门禁测试 PASS 7.43s + 复跑确定性 PASS;`./test/arch/...` PASS;vet/gofmt/`git diff --check` 清洁。首轮 FAIL 为门禁测试期望修正（导航标签实为「需人工」,按 base.tmpl 源事实修正,非放宽断言）。结构化评审清单（逐页 × 结构化优先/词面约束/键盘可达/数据渲染）见报告 §6。执行过程:平台模型请求故障持续,编排者主会话完成。Full evidence: `reports/P2-05_human_centered_ux_gate.md`。


## Goal

Verify structured decision flow is primary and internal state-machine terminology does not dominate user navigation.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Core task can be created/reviewed without Chat-first flow.
- [ ] Keyboard/basic form accessibility checked.
- [ ] No arbitrary LLM-generated UI executes.


## Verification

- UI integration tests + structured review checklist.

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
1. **Tests:** Tests: all P2 handler tests + `execution_http_test.go::TestActionFormsArePlainNoJSUsable`
2. **Contract / Interface:** Contract: Project/Work/Run/Review is navigation projection, not domain state
3. **Core Implementation:** Core: browser-level flow using real derived fixtures
4. **Failure Handling:** Failure: fail if task requires Chat-only interaction or if AI data becomes executable HTML

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
