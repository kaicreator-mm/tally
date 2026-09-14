# P6-06 — Safe project export

- **Phase:** P6
- **Kind:** implementation
- **Wave:** W35
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** A0-05
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — `scripts/export` 落地（零 Go 生产代码触碰）：钉死单一源 commit（`git rev-parse --verify <ref>^{commit}`）→ 流式 `git archive --format=tar <sha>`（仅对象库，绝不碰工作树 —— 闭合 SOURCE_BASELINE 缺口 #7）→ 每个 tar 条目全量 manifest/黑名单扫描 → 干净才原子发布；类型化 `*DeniedPathError`；黑名单 = 任务具名目录（.git/.worktrees/.e2e/.codegraph/.codex/bin）+ 缓存 + 凭证/密钥文件名规则（`.env*` 含文档化 `.env.example` 豁免、id_rsa*、*.pem/.key/…）+ 内容级 PEM 私钥标记扫描；路径安全（symlink/link/特殊类型、穿越、绝对路径、反斜杠、重名）拒绝；manifest 钉死完整 commit sha + 每文件 path/size/sha256。CLI `go run ./scripts/export [-repo -ref -out]` 退出码 0/1/2/3 文档化；Makefile `export-src` 目标（镜像既有 `go run ./cmd/tally migrate` 惯用法）。
- AC：归档含必需源/文档 + 零拒绝路径 — PASS（真实仓库 f7211d3 实跑：exit 0、1313 文件、sha256 461f6b3b7750…、0 denied）；确定性 — PASS（同 commit → 字节等价归档+manifest）；fail-safe — PASS（投毒仓库 exit 2 + 逐路径拒绝 + 空输出目录；密钥内容 fail-safe；未知 ref；合成 tar 游走器含 32KiB 跨界 PEM 标记）。8/8 测试 + `./scripts/...` 邻居全绿；vet/gofmt/新文件全 LF；`git diff --check` 清洁。
- PRODUCTION_REACHABILITY: N/A(reason: scripts/** 下只读操作员工具，经 `make export-src`/`go run ./scripts/export` 实跑可达；导出为只读投影，无生产组合根接线归属)。Risks：黑名单为名字/标记级而非内容 DLP（AWS-key 内容扫描故意省略 —— 被跟踪的 logpipe 脱敏夹具使用 AWS 文档公开示例键，代码+报告已文档化）；确定性范围 = 同 commit + 同 git 版本（未提交在途编辑按设计排除，DL-04 单钉 commit）。Full evidence: `reports/P6-06_safe_project_export.md`。


## Goal

Create deterministic source export for external code review/agent analysis without local runtime state or credentials.


## Allowed / primary paths

- `scripts/**`
- `Makefile`
- `docs/**`


## Implementation requirements

1. Deny `.git`, `.worktrees`, `.e2e`, `.codegraph`, `bin`, caches, credentials/secrets.
2. Archive inspection test.


## Acceptance criteria

- [ ] Generated archive contains required source/docs and zero denied paths.


## Verification

- Automated manifest/denylist scan.

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
1. **Tests:** Tests: archive manifest/denylist scanner; current source archive demonstrates undesired `.git/.e2e/.worktrees` inclusion
2. **Contract / Interface:** Interface: deterministic export command/target
3. **Core Implementation:** Core: `git archive` or explicit allowlist-first exporter plus required untracked fixtures if needed
4. **Failure Handling:** Failure: deny secrets/runtime dirs even if `.gitignore` changes

## External reference pack
- **EXT-GIT-WORKTREE — Git worktree documentation**  
  https://git-scm.com/docs/git-worktree  
  License/version: Documentation/reference only; current Git manual.  
  Use: Disposable linked/detached worktree semantics; no code copying required.

## Do

- Wire only authoritative producers and reproduce defects before changing trust/verification semantics.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not broaden Outcome semantics or weaken verification to make a failing experiment green.
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
