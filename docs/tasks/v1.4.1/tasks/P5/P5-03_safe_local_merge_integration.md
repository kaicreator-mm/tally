# P5-03 — Safe local merge integration

- **Phase:** P5
- **Kind:** implementation
- **Wave:** W32
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P5-02
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — `Service.Merge` 落地：P5-02 确定性集成分支到格式受保护目标 ref 的类型化安全合并 —— 只读 `git merge-tree --write-tree -z --name-only` 冲突预检；CAS-only ref 推进（`update <ref> <new> <old>`，无任何 force/delete/reset argv）；`commit-tree` 确定性双亲合并提交（消息 `tally: merge changeset <id>`，显式身份）；`git worktree list --porcelain` 检出目标守卫；类型化哨兵 `ErrMergeConflict`（`*MergeConflictError{Paths}`）/`ErrInvalidTargetRef`/`ErrTargetCheckedOut`/`ErrIntegrationBranchMissing`/`ErrAlreadyMerged`，双侧基准漂移类型化 `ErrStaleBase` + 分支携带候选固定基准的纵深防御检查。分类用 `*exec.ExitError` 退出码，无输出字符串匹配。
- **P5-02 交接风险已解决**：branch.go 的 `runGit`/`sanitizedGitEnv` 重复副本删除，改为一行转发 `gitwriter.RunGit`；gitwriter 授权最小导出（未导出 `runGit` 增加 `stdin` 参数，7 处调用点零行为变化，导出 `RunGit`）—— 单一事实源，无第三副本（P5-02 报告风险注记的既定首选方案，派发时已授权）。
- AC：干净合并成功、冲突返回结构化结果且无部分隐藏状态 — PASS（真合并断言双亲/两侧内容/确定性消息/历史推进；冲突断言 `ErrMergeConflict` + 精确路径列表，两 ref、HEAD、工作树不动，无 `MERGE_HEAD`）；fail-safe / 无 force 无历史改写 / 审计 ref 保留 — PASS（`refs/heads/tally/**` 不可合并且每种结局后断言不变）。12 个真实临时 Git 仓库测试（干净真合并/快进/冲突/目标漂移/分支漂移/重放/检出目标/无基准分支/缺分支/11 例校验表/非仓库/ctx 取消）。`go test ./internal/gitintegration/ ./internal/gitwriter/` PASS（go clean -testcache 后：gitintegration 32 = 13+7+12，gitwriter 签名变化全绿）；`./test/arch/...` PASS；vet/gofmt/`git diff --check` 清洁，触及文件全 LF。PRODUCTION_REACHABILITY: N/A(reason: 冻结接缝上的库级操作，接线归后续组装任务)。Risks：`merge-tree --write-tree` 需 git ≥ 2.38（本机 2.55；旧装机 fail-safe 为普通 `GitError`，设计上无回退）；冲突预检写一个未引用 tree 对象（可 gc）；CAS 终局竞态分类为代码评审覆盖（无注入钩子不可确定性测试）。Full evidence: `reports/P5-03_safe_local_merge_integration.md`。


## Goal

Integrate an accepted ChangeSet locally with typed conflict handling and no autonomous conflict resolution.


## Allowed / primary paths

- `internal/gitintegration/**`
- `internal/gitwriter/**`


## Implementation requirements

1. Fail safe on merge conflict.
2. No force/history rewrite.
3. Preserve audit refs.


## Acceptance criteria

- [ ] Clean merge succeeds; conflict returns structured result without partial hidden state.


## Verification

- Git fixture tests incl. conflict/base drift.

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

- **Execution:** `HIGH`
- **Review:** `HIGH`
- **Reason:** Merge/conflict/base-drift correctness is repository-safety critical.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: conflict/base-drift repo fixtures; no-conflict and rename/file-dir conflict cases
2. **Contract / Interface:** Interface: merge preflight/result + accepted ChangeSet + expected target ref
3. **Core Implementation:** Core: prefer `git merge-tree --write-tree` for preflight where installed Git supports it, then controlled merge/ref update strategy
4. **Failure Handling:** Failure: conflict is a structured non-success; no autonomous resolution, force, or history rewrite

## External reference pack
- **EXT-GIT-MERGE-TREE — Git merge-tree documentation**  
  https://git-scm.com/docs/git-merge-tree  
  License/version: Documentation/reference only; current Git manual.  
  Use: Preflight merge/conflict evaluation without touching working tree/index; inspect exit status and conflict records.
- **EXT-GIT-MERGE — Git merge documentation**  
  https://git-scm.com/docs/git-merge  
  License/version: Documentation/reference only; current Git manual.  
  Use: Authoritative merge/abort/conflict behavior.
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

## High-model execution requirement

This task should be performed by a high-capability model because success requires semantic judgment rather than mechanical editing. The model must explain the chosen seam, enumerate rejected alternatives, and demonstrate that fail-safe behavior remains intact before coding.

## L3 completion evidence required from implementing agent

- exact source/tests read;
- any external reference actually used;
- why the chosen seam is the narrowest valid seam;
- changed files;
- exact test commands/results;
- negative tests added;
- known uncertainty or version sensitivity;
- confirmation that Decision Locks were not bypassed.
