# P1-01 — Same-source execution bundle loader

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W08
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P0-09
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Resolve TaskContract, VerificationContract, FrozenIntent and EngineeringPolicies from one pinned source commit.


## Allowed / primary paths

- `internal/artifacts/**`
- `internal/contract/**`
- `internal/intentevidence/**`


## Implementation requirements

1. No mixed working-tree/source-commit reads.
2. Typed rejection for missing/mismatched artifact.


## Acceptance criteria

- [ ] Bundle identity includes one source commit and all loaded revisions.


## Verification

- Tests with intentionally divergent commits.

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
1. **Tests:** Tests: `artifacts/v13_loader_test.go::TestLoadReadsGitSnapshotNotWorkingTree`, `v14_policy_loader_test.go`
2. **Contract / Interface:** Interface: single `sourceCommit` identity on artifact snapshots
3. **Core Implementation:** Core: compose existing task/verification/policy loaders + FrozenIntent loader
4. **Failure Handling:** Failure: mixed-commit artifacts are a hard error, never auto-upgraded to HEAD

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
- Report: `docs/tasks/v1.4.1/reports/P1-01_same_source_execution_bundle_loader.md`
- Implemented: 同源执行 bundle 装载器 `artifacts.LoadBundle(repoPath, rev, taskID)`(新 `internal/artifacts/v141_bundle_loader.go`):把 rev 解析到**恰一个**提交,从该提交树快照一次装载同一任务身份的全部执行权威 —— FrozenIntent(共享 `loadIntentTree`,与 `LoadIntent` 逐字同裁决)、TaskContract(`taskcontract.Parse`+`ValidateStatic`,与 v13 装载同链)、冻结 VerificationContract(`verification.Parse`+`FreezeContract` 零值策略,与 v13 装载同链)、EngineeringPolicies(既有 `loadPolicyTree` 原样,无效策略逐文件 Rejected,DL-09/BI-01 继承);混合来源在构造上不可能(绝不二次解析 revision,绝不自动升级到 HEAD 补读,DL-04);装载顺序 intent→task→verification→policies 固定;六新哨兵 + 三个既有 intent 哨兵全部 `errors.Is` 可判;bundle 身份 `Identity()` = `artifacts/bundle/v1` 规范文本(task id + 观测来源提交 + `Revisions()` 全部装载修订)的确定性 sha256。`LoadIntent` 纯重构到共享 `openCommitTree`/`loadIntentTree`,公共签名与语义逐字不变;`internal/contract` 零改动(`Snapshot`/`Compile`/`Bind` 权威原样,装配归 P1-03)。
- Acceptance: 「Bundle identity includes one source commit and all loaded revisions」— PASS(`TestLoadBundleAtFreezeCommit`:经生产 FreezeCommit 落库后同提交装载,三件套与冻结值逐项相等、与独立装载器相等,Revisions 覆盖全部五类修订,同提交复装载身份确定;`TestLoadBundlePinnedCommitNeverUpgrades`:策略换版→身份变;内容相同、来源提交不同→身份仍变)。实现要求 1(无混合工作区/来源提交读取)与 2(缺失/不匹配制品类型化拒收)由构造与 `TestLoadBundleTypedRejections`(9 子用例)钉住。任务 Verification(divergent commits)由 `TestLoadBundlePinnedCommitNeverUpgrades` 覆盖。
- Validation: `go test ./internal/artifacts/` PASS;`go test -run 'TestLoadBundle' -v ./internal/artifacts/` PASS(4/4 顶层,9/9 子用例);参考 pack 点名测试(`TestLoadReadsGitSnapshotNotWorkingTree` + 全部 `TestLoadPolicies*`)7/7 PASS;`go test ./test/arch/...` PASS;`go build ./...` + `go vet ./...` PASS;`go test ./internal/...` PASS(46 包);`gofmt -l`(三个触碰文件)clean;`git diff --check` clean;`make check` NOT_RUN(无 make,构成命令逐条执行);integration e2e NOT_RUN(DB-gated opt-in,非本任务 Verification 要求)。
- Baseline failures: none。
- Decision Locks touched: none(DL-04 即本任务实现的锁;DL-09/BI-01 经既有策略装载继承;DL-17 遵守,hidden/ 未读;DL-18 遵守,既有域零改动)。
- Risks: 零策略/含拒收策略的 bundle 可装载,是否阻断执行归 P1-03 fail-safe 裁决;go-git `tree.File` 把损坏树读取归为 NotFound 的既有分类学(P0-02 已记录);HIGH review 待执行(重点:单次解析论证与「不越权装配 contract.Snapshot」的缝界)。
- Commit: 见本文件 git 历史(`P1-01: …`)。
