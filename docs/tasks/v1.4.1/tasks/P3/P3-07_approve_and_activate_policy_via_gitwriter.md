# P3-07 — Approve and activate policy via GitWriter

- **Phase:** P3
- **Kind:** implementation
- **Wave:** W23
- **Complexity:** L
- **Execution model:** LOW
- **Depends on:** P3-03, P3-06
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — 激活链落地 `internal/policygovernance/activate.go`（新文件；gitwriter/artifacts/web-handlers 零改动，纯组合）：`(*Store).Activate` 按不变量顺序执行 —— (1) Reconstruct 核验（PROPOSED + 证据齐备 + 候选修订一致，DRAFT/终态/未知/漂移全部 `ErrInvalidActivation`）；(2) `gitwriter.Writer.Commit` 期望基线 CAS 提交正典 YAML（确定性消息 `tally: activate policy <id> (proposal <pid>)`；陈旧基线 → 类型化失败、治理面保持 PROPOSED、零激活事件、HEAD 不动）；(3) `artifacts.LoadPolicies` 从确切新提交重载，必须观察到同 id 同内容修订（`ErrActivationNotObservable`）；(4) 最后落 activate 治理事件（approved_by 必填，BI-01）—— **治理记录在 Git 可观察之后**：DL-09 下 Git 是运行时权威，事件是批准证据记录；第 4 步失败类型化并携带已生效提交 sha（append-only 补记，绝不回滚 Git）。制品路径由策略 id 推导（路径安全白名单，不做转义器）。**DAG 顺序偏离声明**：本任务 Depends 含 P3-06（UI 投影）；编排裁决按服务核心先行 —— 激活服务与读投影无技术耦合，P3-06 的 Rules UI 将消费本服务与 P3-05 冲突分析，消费面接线随 P3-06/整合任务完成（本任务验证面 = 服务级集成测试，与任务 Verification 节一致）。
- 验证矩阵：success（提交 → 独立 LoadPolicies 复核同 id 同修订 → Reconstruct ACTIVE + approved_by）/ stale base（CAS 拒绝 + 治理面保持 PROPOSED + HEAD 不动）/ invalid candidate（DRAFT 拒绝、缺批准人、未知提案）/ commit failure（CAS 失败面覆盖；事件落库故障路径代码化并文档化，真实注入夹具留待 P3-09）。`DATABASE_URL=… go test ./internal/policygovernance/ -count=1` PASS 76.0s（全包 12 测试）；arch/build/vet/gofmt/`git diff --check` 全 PASS。
- PRODUCTION_REACHABILITY: N/A(reason: 服务级接缝；激活的 UI 入口归 P3-06,组合根接线归后续整合任务——与 P3-08 Governance 字段同一模式)。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P3-07_approve_and_activate_policy_via_gitwriter.md`。


## Goal

Convert a human-approved RuleProposal into canonical `.tally/policies/*.yaml` through expected-base Git commit.


## Allowed / primary paths

- `internal/policygovernance/**`
- `internal/gitwriter/**`
- `internal/artifacts/**`
- `internal/web/handlers/**`


## Implementation requirements

1. Record governance approval evidence.
2. Commit canonical policy through existing GitWriter.
3. Reload committed revision from exact new source commit.
4. Stale base fails safely.


## Acceptance criteria

- [ ] A proposal becomes runtime-active only after committed policy is observable from source snapshot.


## Verification

- Integration test success/stale base/invalid candidate/commit failure.

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
1. **Tests:** Tests: `gitwriter/writer_test.go` stale-base/rollback cases; `v14_policy_loader_test.go`
2. **Contract / Interface:** Interface: approved candidate → `gitwriter.CommitRequest` → reload exact policy revision from new commit
3. **Core Implementation:** Core: existing GitWriter; canonical EngineeringPolicy serialization
4. **Failure Handling:** Failure: stale base, failed commit, reloaded revision mismatch all leave proposal non-active

## External reference pack
- **EXT-GITHUB-RULESETS — GitHub Repository Rulesets — evaluate/rule insights pattern**  
  https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository  
  License/version: Documentation/reference only; current GitHub Docs.  
  Use: Pattern for rule proposal/evaluate-before-enforce and inspecting affected historical/current actions.
- **EXT-GIT-UPDATE-REF — Git update-ref documentation**  
  https://git-scm.com/docs/git-update-ref  
  License/version: Documentation/reference only; current Git manual.  
  Use: Compare-and-swap style ref update semantics using expected old object IDs.

## Do

- Treat Git policy content as runtime authority; governance is evidence and authorization around it.
- Start by running the listed focused tests or the closest available subset.
- Make the smallest semantic change that satisfies this task; keep public surface area narrow.
- Add at least one negative/failure-path test for new behavior.

## Don't

- Do not build a second policy DSL/engine or treat DB governance status as runtime activation.
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
