# P5-01 — Semantic local Git integration service

- **Phase:** P5
- **Kind:** implementation
- **Wave:** W30
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P1-08
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-12) — 语义集成层落地为 `internal/gitintegration.Service.Integrate`：一个具名类型化操作（校验流水线 + 经 `gitwriter.Writer.Commit` 的只读 CAS 预检 + 确定性提交消息 `tally: integrate changeset <id>`），不暴露原始 argv；哨兵错误全类型化（`ErrCandidateUnidentified` / `ErrNotAccepted` / `ErrAcceptanceMismatch` / `ErrProtectedTarget` / `ErrNoExpectedBase` / `ErrStaleBase`（包装 `gitwriter.ErrBaseRevision`）/ `ErrNoApplicableContent` / `ErrNoRepository`）；`AcceptanceFact` 强制 ChangeSetID + AcceptedAt + 期望基准，空基准 fail-safe 拒绝。`internal/gitwriter/**` 与 `internal/changeset/**` 零改动 —— 纯组合，无核心修改（开放接缝行 "L3 local Git integration service API" 由本任务关闭，接缝判定与被拒替代方案见报告）。
- AC：未接受/未识别候选不可集成 — PASS（空/未识别 `ErrCandidateUnidentified`，验收事实缺失或不完整 `ErrNotAccepted`，身份不匹配 `ErrAcceptanceMismatch`，全部在 HEAD 不动、工作树清洁下失败）；语义化操作 — PASS；期望基准 + 已接受 ChangeSet — PASS。12 个真实临时 Git 仓库测试（gitwriter 约定）全绿。
- Verification：`go test ./internal/gitintegration/...` PASS（orchestrator 复核）；`./internal/gitwriter/... ./internal/changeset/...` PASS（未改动近邻零回归）；`go vet` PASS；`gofmt -l` 清洁；`git diff --check` 清洁。Full evidence: `reports/P5-01_semantic_local_git_integration_service.md`。
- PRODUCTION_REACHABILITY: N/A(reason: 生产组装归属 P5-02/P5-03，其在本接缝之上构建分支创建/合并)。Risks：AcceptanceFact 来源为调用方 —— 后续闭环组装必须仅从验收权威记录构建事实；主工作树保护为路径级，符号链接别名未解析（既有路径保护限制）。

## Goal

Add controlled local integration semantics on top of existing GitWriter/workspace/ChangeSet without exposing raw model-controlled Git.


## Allowed / primary paths

- `internal/gitintegration/**`
- `internal/gitwriter/**`
- `internal/changeset/**`


## Implementation requirements

1. Semantic operations only.
2. Expected-base and accepted ChangeSet required.


## Acceptance criteria

- [ ] Service cannot integrate an unaccepted/unidentified candidate.


## Verification

- Unit tests around validation/CAS.

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
- **Reason:** Defines Git integration authority and semantic API; wrong design can re-expose raw Git or duplicate GitWriter.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: `gitwriter/writer_test.go`, `workspace/manager_test.go`, `changeset/*_test.go`
2. **Contract / Interface:** Interfaces: accepted immutable ChangeSet + ExpectedBase; semantic service, no model-facing raw argv
3. **Core Implementation:** Core: compose GitWriter/workspace/ChangeSet rather than moving their authority
4. **Failure Handling:** Failure: reject unaccepted/unknown changeset, stale base, protected target

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
