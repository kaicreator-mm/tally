# P5-06 — Local Git control-plane gate

- **Phase:** P5
- **Kind:** gate
- **Wave:** W34
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P5-03, P5-04, P5-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — **P5 门禁: PASS**(零生产代码改动;P5-01..05 交付面原样组合于真实临时仓库)。全矩阵(`test/e2e/v141_p506_git_control_plane_gate_test.go`,1.6s,双跑确定性):集成主链(EnsureIntegrationBranch 确定性分支 CAS create → Integrate 候选内容经唯一写边界 CAS 提交到分支 → Merge 确定性合并提交 + 目标 ref CAS 推进,受保护 main 经批准令牌);漂移/并发写败者(过期基线 → `ErrStaleBase`,HEAD 不动,预检 + 锁内二次校验双层 CAS,零部分写入);冲突族(已合并重放 `ErrAlreadyMerged`;分支漂移 `ErrIntegrationBranchExists` —— P5-02 CAS create 使覆盖结构上不可能;真三方可检冲突由 merge_test.go 包级 12 测试覆盖并如实声明);受保护分支(无批准 → `ErrProtectedTargetDenied`;带批准 → 令牌);运行时天花板(RuntimeCaller 请求 L3 → `ErrRuntimeAuthorityExceeded`,即使携带有效批准);审计列举(P5-05 可见治理命名空间分支)。**AC1 零远程**:仓库无任何 remote 配置(git.Remotes 断言空),全部操作离线。**AC2 并发 CAS**:见漂移/并发行。
- 验证:门禁测试 PASS 1.61s + 双跑确定性;`./test/arch/...` PASS;build/gofmt/`git diff --check` 清洁。两轮开发期修正均为夹具问题(PlainInit 默认分支名;分支检出编排),非基线失败。**P5 lane 关闭,P5-06 发布阻塞解除**。
- PRODUCTION_REACHABILITY: PASS(门禁经 e2e 集成标签在生产 CI 面可执行;被测服务全部为已落地生产库)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P5-06_local_git_control_plane_gate.md`。


## Goal

Validate L0–L3 scope under drift, conflict, protected branch and concurrent writer conditions.


## Allowed / primary paths

- `test/**`
- `docs/tasks/v1.4.1/reports/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] No remote credential/push requirement in gate.
- [ ] Concurrent writes respect existing lock/CAS semantics.


## Verification

- Full local Git integration matrix.

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
1. **Tests:** Tests: concurrent/stale lock tests in `gitwriter`, worktree cleanup tests, new merge conflict matrix
2. **Contract / Interface:** Contract: local L0–L3 only
3. **Core Implementation:** Core: E2E on temporary repositories
4. **Failure Handling:** Failure: no remote credentials/network dependency allowed in release gate

## External reference pack
- **EXT-GIT-WORKTREE — Git worktree documentation**  
  https://git-scm.com/docs/git-worktree  
  License/version: Documentation/reference only; current Git manual.  
  Use: Disposable linked/detached worktree semantics; no code copying required.
- **EXT-GIT-UPDATE-REF — Git update-ref documentation**  
  https://git-scm.com/docs/git-update-ref  
  License/version: Documentation/reference only; current Git manual.  
  Use: Compare-and-swap style ref update semantics using expected old object IDs.

## Do

- Use semantic local Git operations with CAS/base checks; conflict is an expected typed outcome.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not expose raw arbitrary Git commands to model-controlled execution or automate force/history rewrite.
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
