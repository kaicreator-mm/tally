# P0-08 — Atomic Confirm-to-Freeze Git transaction

- **Phase:** P0
- **Kind:** implementation
- **Wave:** W06
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P0-02, P0-04, P0-05, P0-06, P0-07
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Turn user confirmation into immutable Git-backed FrozenIntent + TaskContract + VerificationContract at one expected base.


## Allowed / primary paths

- `internal/web/handlers/v14_launch.go`
- `internal/gitwriter/**`
- `internal/artifacts/**`
- `internal/intakefreeze/**`


## Implementation requirements

1. One expected-base CAS boundary.
2. All artifacts refer to same task/source commit semantics.
3. Failure leaves no partial durable artifact set.
4. PRG-safe handler behavior.


## Acceptance criteria

- [ ] Stale HEAD conflicts instead of silently rebasing.
- [ ] Successful confirm is reloadable from committed artifacts.


## Verification

- Integration tests: success, stale base, partial failure rollback, duplicate confirm idempotency semantics.

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
- **Reason:** Atomic multi-artifact Git transaction with stale-base and rollback semantics; partial success is dangerous.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests first: `gitwriter/writer_test.go::{TestCommitValidatesExpectedBase,...}`, `v14_launch_test.go`, artifact loader tests
2. **Contract / Interface:** Interfaces: `gitwriter.CommitRequest{ExpectedBase,Files}`, immutable artifact loaders
3. **Core Implementation:** Core: `gitwriter.Writer.Commit`, `V14ConfirmDraft`, new mapper outputs
4. **Failure Handling:** Failure: stale HEAD, partial write, retry/idempotency, rollback; one confirm cannot leave a half-frozen artifact set

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

- Keep presentation, intent evidence and frozen engineering artifacts explicitly separated.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not let LLM-generated UI/risk text become authority without deterministic validation/freeze.
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

---

## Execution status

- **Status: DONE** (2026-09-12)
- Report: `docs/tasks/v1.4.1/reports/P0-08_atomic_confirm_to_freeze_git_transaction.md`
- Implemented: 确认→冻结 Git 事务接线。新 `internal/artifacts/v141_freeze_commit.go`(P0-02 装载器的写侧对应物):`FreezeCommit` 把 `intakefreeze.FrozenArtifacts` 三件套渲染为 `.tally/intents/<id>.yaml` + `.tally/tasks/<id>.yaml` + `.tally/verifications/<id>.yaml`,经**恰一次** `gitwriter.Writer.Commit` 以显式 `ExpectedBase` 原子提交(单一 CAS 边界,DL-03 纪律,gitwriter 零改动);提交前任务身份一致性门(`ErrFreezeIdentityMismatch`)fail-closed。handler 接线(`v14_launch.go`,P0-04→P0-08 串行热点归本任务):`V14ConfirmDraft` → 表单冻结输入(task_id/profile 显式选择/risk_hint 勾选保留/逐条款可见证据 kind+path+selector)→ `intakefreeze.Freeze`(P0-07 准入原样)→ `FreezeCommit` → 四种结局全部 PRG 303 flash(冻结完成含 sha / 重复确认幂等 ErrNoChanges / 基线冲突类型化拒绝绝不改基 / 失败如实转述);准入失败 400 退回表单且全部已填内容保留。生产可达无需组合根改动:confirm 路由已在生产挂载,仓库与提交身份来自既有注入的 `Server.Content`;`cmd/server`、`server.go` 零接触。模板(`v14_launch.tmpl`,PATH_OWNERSHIP 于 P0-04 合并后释放,P0-07 报告明确确认表单携带证据声明归 P0-08)增加冻结输入控件;「冻结未接线」的过期文案移除。
- Acceptance: 「Stale HEAD conflicts instead of silently rebasing」— PASS(事务级显式 stale base → 类型化 `ErrBaseRevision`,零提交、零文件、工作区干净;handler 层冲突 flash 如实转述)。「Successful confirm is reloadable from committed artifacts」— PASS(事务级与 handler 级均验证:`artifacts.Load` 在新 HEAD 返回同哈希 TaskContract + 冻结 VerificationContract,`artifacts.LoadIntent` 返回同 Revision/SourceRef/RiskHints 的 FrozenIntent)。
- Validation: `go test -count=1 -run FreezeCommit -v ./internal/artifacts/` PASS(5/5:成功+三件套读回 / stale base / 部分失败回滚(apply 中途失败零残留)/ 重复确认 ErrNoChanges / 身份不一致拒收);`go test -count=1 -run 'TestV14Confirm' -v ./internal/web/handlers/` PASS(4/4:成功+读回 / 重复幂等无新提交 / 准入阻断(缺证据、证据半声明、任务 id 含空白)且 HEAD 不动 / 仓库未接入如实降级);`go test -count=1 ./internal/artifacts/ ./internal/intakefreeze/ ./internal/gitwriter/ ./internal/intentevidence/ ./internal/web/handlers/` PASS;`go test ./test/arch/...` PASS;`go build ./...` + `go vet` PASS;`go test -count=1 ./internal/...` PASS(46 包 0 失败);`DATABASE_URL=…15432… go test -tags=integration ./test/e2e/... ./internal/web/handlers/` PASS(共享本地 Postgres,206s/97s);`gofmt -l`(五个 Go 文件)clean;`git diff --check` clean。`make check` NOT_RUN(reason: 无 make;构成命令已逐条执行)。
- Baseline failures: none(`v14_launch_test.go` 中两条断言钉的是 P0-08 之前「冻结未接线」回执,随行为更新)。
- Decision Locks touched: none(DL-03/DL-04/DL-06/DL-17/DL-18 遵守;hidden/ 未读;held-out 边界保持 —— UI 证据面只声明 repo 路径型可见证据,held-out/validator/protection 仍归程序化 API 面)。
- Risks: 确认 UI 每条款声明一条可见证据(域面支持 N 条/held-out,UI 有界);同任务 id 变内容再确认产生新修订提交(内容哈希为准,恒等内容为幂等空操作,修订簿记归后续 lane);HIGH review 待执行。
- Commit: 见本文件 git 历史(`P0-08: atomic confirm-to-freeze git transaction — …`)。
