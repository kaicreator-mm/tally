# P1-05 — Compiled prevention context projection

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W12
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P1-04
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Expose compiled prevention requirements to the coding runtime without exposing held-out verification material.


## Allowed / primary paths

- `internal/contextpack/**`
- `cmd/worker/v14.go`
- `internal/contract/**`


## Implementation requirements

1. Use `Compiled.Requires()` or existing canonical projection.
2. Separate prevention context from verification evidence/holdouts.


## Acceptance criteria

- [ ] Runtime receives required implementation constraints but cannot see held-out content.


## Verification

- Prompt/context snapshot tests and exposure regression tests.

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
1. **Tests:** Tests first: `internal/contextpack/requires_test.go`, `contextpack_test.go::TestHeldOutProjectionExposesStatementAndTypeOnly`, `selector_test.go::TestSelectFiltersHeldOutInternals`
2. **Contract / Interface:** Interface: `contract.Compiled.Requires()`, `contextpack.Input/Build`
3. **Core Implementation:** Core: `contextpack/build.go`, `redact.go`, worker context construction
4. **Failure Handling:** Failure: prevention context may include policy requirements, never held-out store/ref/content

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
- Report: `docs/tasks/v1.4.1/reports/P1-05_compiled_prevention_context_projection.md`
- Implemented: `cmd/worker/v14.go` 把编译权威的执行前 requires 上下文（PRD §8.3 Prevention）投影进 v1.4 闭环的两条确定性路径——① `runV14Task` 的 ContextPack 组装换新增的 `v14ContextPack(..., authority.Compiled.Requires())`（候选面复用 v13.go 既有件零改动;唯一差异是把 `Compiled.Requires()` 注入 `contextpack.SelectInput.Requires` 既有槽位）,使 `RunBegin.ContextPackHash` 成为 compiled-aware 哈希:策略换版/命中集变化必然改变上下文坐标,修复/重放不能在事实面上悄悄换权威;② `v14CodegenRunner.RunCodegen` 每轮迭代指令渲染确定性 Prevention 段（`v14Instruction` 增参,`RequiresContext.Lines()` 行,与 Acceptance/Repair context 分离小节）——runtime 实际收到实现约束;③ `RunCodegen` 入口加编译权威类型化守卫（缺权威拒绝执行,fail-safe 绝不无声开跑）;④ 编译日志增加 `policy_requires` 计数。`internal/contextpack/**` 与 `internal/contract/**` **零改动**——`Compiled.Requires()`/`Input.Requires`/`Pack.PolicyRequires`/INV-V14-07 全部既有权威原样消费(DL-18),v13 路径零改动,§10 热点零接触。
- Acceptance: 「Runtime receives required implementation constraints but cannot see held-out content」— PASS:正向 `TestV14PreventionContextReachesRuntime`(端到端 Accepted,逐迭代指令含确定性快照行 `- policy db_gate (gate): 最终会按这些条件验收: data_integrity, fresh_database — 理由: …`;`ContextPackHash` 与重装载+重编译+`v14ContextPack` 复算逐字节一致,且因 requires 偏离无 requires 组装);暴露回归 `TestV14PreventionExcludesHeldOutAndBindsHash`(held-out 契约 HOLD-1/holdout/hold-1 与命中策略同场:指令与渲染行零泄漏;策略 v2 换版 ⇒ context_hash 必变,而同对提交上无 requires 的哈希不变——差异恰好来自编译权威);实现要求 1(用 `Compiled.Requires()`)与 2(Prevention 与验证证据/holdouts 分面)均 PASS。
- Validation: `go build ./...` PASS;`go vet ./cmd/worker/ ./internal/contextpack/` PASS;`gofmt -l`(touched files) clean;`go test ./cmd/worker/ -run 'TestV14Prevention…\|TestV14ContextPackZeroRequiresMatchesV13\|TestV14RunnerRequiresCompiledAuthority\|TestV14InstructionRendersSafeFeedbackOnly' -v` PASS 5/5;`DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres go test ./cmd/worker/` PASS 全量(48.1s,含两个真库持久化测试 RUN+PASS 非 SKIP);`go test ./internal/contextpack/... ./internal/contract/... ./internal/artifacts/... ./internal/engineeringpolicy/...` PASS;`go test ./test/arch/...` PASS;`go test ./internal/...` PASS;`go test ./test/runtimeconformance/` PASS;`go test -tags=integration ./test/e2e/` PASS;`git diff --check` clean;`make check` NOT_RUN(环境无 make 二进制,构成命令逐条执行)。
- Baseline failures: none。
- Negative tests added: `TestV14RunnerRequiresCompiledAuthority`(缺编译权威 ⇒ 类型化拒绝 + 零迭代)、`TestV14PreventionExcludesHeldOutAndBindsHash`(held-out 三词反断言 + 换版敏感双断言)、`TestV14InstructionRendersSafeFeedbackOnly` 扩展(零值 requires 不渲染 Prevention 段)。
- Decision Locks touched: none(DL-04 同源输入未扰动;DL-09 策略仍以 pinned 快照精确内容消费;DL-17 遵守,hidden/ 未读;DL-18 零重复;DECISION_LOCKS.md 文件未动)。
- Risks: context_hash 覆盖 Prevention 面而非编译身份全量——纯风险信号变化(不动命中集)只改 `Compiled.Hash()` 不改 requires,该差异由 P1-04 已钉住的运行事实坐标(可确定性复算)承载,读模型 `compiled:<hash>` basis 归 P1-07;每轮指令携带全部命中策略告知行,token 预算/裁剪归 Context Engine 后续任务(requires 为必现内容,contextpack 既有不参与预算裁决);test/e2e 测试侧等价组合的 Prevention 对齐归 P1-08;HIGH review 待执行(重点:AC 双路径论证与覆盖边界)。
- Commit: 见本文件 git 历史(`P1-05: …`)。
