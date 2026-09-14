# P1-04 — Wire worker to compiled authority

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W11
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P1-03
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Make production v1.4 worker obtain CompiledContract before creating an execution workspace/runtime session.


## Allowed / primary paths

- `cmd/worker/v14.go`
- `internal/artifacts/**`
- `internal/taskstate/**`


## Implementation requirements

1. Fail before execution on compilation authority errors.
2. Record compiled identity/basis in run facts/read model.


## Acceptance criteria

- [ ] Runtime start spy proves no session starts before successful compilation.


## Verification

- Worker tests for compile success/failure and no-runtime-on-failure.

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
1. **Tests:** Tests: `cmd/worker/v14_test.go`, `test/arch/v14_runtime_boundary_test.go`
2. **Contract / Interface:** Interface: worker config/deps + compiled assembler result
3. **Core Implementation:** Core: `cmd/worker/v14.go` composition order before workspace/runtime start
4. **Failure Handling:** Failure: prove runtime/session/worktree is not started on compilation failure

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
- Report: `docs/tasks/v1.4.1/reports/P1-04_wire_worker_to_compiled_authority.md`
- Implemented: `cmd/worker/v14.go` 的 `runV14Task` 接入合同编译权威(PRD §16 no-bypass 成为生产组合事实)——装载换 P1-01 `artifacts.LoadBundle(repo, "HEAD", taskID)`(单次 revision 解析,四类权威制品同一 pinned 提交,DL-04;拒绝 Load+LoadBundle 双解析的混合来源形态),`bundle.CompileAuthority()` 前置为执行第一道门:先于 runtime 选择、held-out 前提强制、workspace manager 创建与任何 runtime 会话;编译权威错误(拒收策略 `ErrPolicyRejections`/无效风险依据/来源不一致/bundle 不完整哨兵)类型化经 `%w` 上抛为执行前置失败——绝不进闭环、绝不伪装 verdict、绝不耗修复预算;编译身份(task/source_commit/bundle/compiled 哈希/risk 信号计数)在任何会话开启前结构化落观测;运行事实坐标(project/task 修订/verification 修订/base 提交)全部改从同一 bundle 取值;`v14CodegenRunner` 增加 `authority` 字段递至 P1-05(Compiled.Requires→context_hash)/P1-06(风险调整验收守卫)。`internal/artifacts/v141_bundle_loader.go`:`ExecutionBundle` 增加同树 `ProjectID` 坐标(复用 `parseManifest`,排在四类制品后装载,P1-01 钉住的类型化拒收序逐字不变;缺席空值由 worker 执行入口 fail-closed)。`cmd/worker/v14_test.go`:bundle 完整 fixture(`newV14Repo`/`newV14HeldRepo`/`newV14RejectedPolicyRepo`)+ `v14StartSpy`;v13_test.go 与 v13 路径零改动,§10 热点(main.go/v13.go/server.go/sqlc/migrations)零接触。
- Acceptance: 「Runtime start spy proves no session starts before successful compilation」— PASS:`TestV14CompileFailureBlocksRuntimeStart`(拒收策略 → `ErrPolicyRejections` + spy 零 Start + 零 codegen 事实 + 零 worktree;缺 intent → `ErrIntentNotFound` + 零 Start)与 `TestV14RunFactsPinCompiledAuthority`(编译成功 → 恰好一次 Start、闭环 Accepted、运行事实坐标与重装载重编译逐项一致)。实现要求 1(编译权威错误先于执行失败)PASS;实现要求 2(编译身份/依据入运行事实)PASS——本任务路径内完整形态:事实坐标全源同源(身份可确定性复算,测试钉住)+ 执行起点身份观测 + 权威随组合递至下游;持久 compiled 专列在 §10 热点(workflow/runstore/sqlc/migrations)内,按 PATH_OWNERSHIP 归 P1-05(context_hash)/P1-07(读模型 basis)。
- Validation: `go build ./...` PASS;`go vet ./cmd/worker/ ./internal/artifacts/` PASS;`go test ./cmd/worker/ -count=1` PASS;`DATABASE_URL=postgresql://postgres:postgres@localhost:15432/postgres?sslmode=disable go test ./cmd/worker/ -run 'TestV14' -v` PASS 17/17(含两个真库持久化测试);`go test ./internal/artifacts/ -count=1` PASS(P1-01/P1-03 合同保持);`go test ./test/arch/...` PASS;`go test ./internal/...` PASS;`go test ./test/runtimeconformance/` PASS;`go test -tags=integration ./test/e2e/` PASS(201.8s);`git diff --check` clean;`make check` NOT_RUN(环境无 make 二进制,构成命令逐条执行)。
- Baseline failures: none。
- Decision Locks touched: none(DL-04 单次解析+同源注册表满足;DL-05 派生值不落库;DL-16 单 runtime owner 未动;DL-17 遵守,hidden/ 未读;DECISION_LOCKS.md 文件未动)。
- Risks: 每运行持久 compiled 槽位按所有权分期(P1-05 context_hash / P1-07 读模型 basis),此前审计者需重装载复算哈希(路径已测试钉住);e2e 测试等价组合 `clRunTask` 的编译权威化归 P1-08(test/** 面);HIGH review 待执行(重点:接缝拒绝项与记录分期边界)。
- Commit: 见本文件 git 历史(`P1-04: …`)。
