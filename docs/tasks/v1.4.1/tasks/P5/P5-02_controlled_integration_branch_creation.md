# P5-02 — Controlled integration branch creation

- **Phase:** P5
- **Kind:** implementation
- **Wave:** W31
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P5-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — `Service.EnsureIntegrationBranch` 落地：确定性分支名 `refs/heads/tally/integration/<sha256-hex-of-ChangeSetID>`，根植于 ChangeSet 自身固定基准解析出的不可变 OID；经 `git update-ref --stdin` `create` 原语的 CAS 创建（覆盖在结构上不可能）；复用 P5-01 冻结校验流水线（候选已识别、验收事实完整且匹配、非主工作树目标、类型化 `ErrNotRepository`）；同基准已存在 → 幂等（`Created=false`），漂移分支 → `ErrIntegrationBranchExists`（测试断言 ref 保持旧值且 HEAD 不动），目标缺基准 → `ErrBaseNotFound`；ref 名与分支点全推导，无任何调用方/模型可控输入触及 git argv 或 stdin（无第二 Git 写权威）。
- AC：基准漂移为类型化冲突、无静默覆盖 — PASS（7 个真实临时 Git 仓库测试：happy path + 幂等 + 5 条失败路径）。`go test ./internal/gitintegration/...` PASS（19 测试 = 12 P5-01 + 7 新）；`./internal/gitwriter/... ./internal/changeset/...` PASS（近邻零回归）；`./test/arch/...` PASS（依赖策略不变）；vet/gofmt/`git diff --check` 清洁（两新文件 0 CRLF，LF per `.gitattributes`）。Full evidence: `reports/P5-02_controlled_integration_branch_creation.md`。
- PRODUCTION_REACHABILITY: N/A(reason: 冻结 P5-01 接缝上的库级操作；生产接线归 P5 lane 组装任务)。Risks：`runGit`/`sanitizedGitEnv` 复制两个 gitwriter 未导出内部（P5-03 拥有 `internal/gitwriter/**` 路径，可做授权小导出）；并发输竞幂等路径（create 失败→按值读回）结构正确但无注入钩子不可确定性测试。

## Goal

Create an integration branch/reference from the expected accepted base/change using deterministic naming/ownership.


## Allowed / primary paths

- `internal/gitintegration/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Base drift is typed conflict; no silent branch overwrite.


## Verification

- Temporary repo tests.

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
1. **Tests:** Tests: Git fixture style from `gitwriter/writer_test.go` and `workspace/manager_test.go`
2. **Contract / Interface:** Interface: deterministic integration ref/branch identity from expected base/change
3. **Core Implementation:** Core: Git command/plumbing isolated in gitintegration package
4. **Failure Handling:** Failure: existing target mismatch must fail; do not silently reset/overwrite

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
