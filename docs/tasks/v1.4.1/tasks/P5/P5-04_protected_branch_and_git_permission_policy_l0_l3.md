# P5-04 — Protected-branch and Git permission policy L0-L3

- **Phase:** P5
- **Kind:** implementation
- **Wave:** W32
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P5-01
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — Git 权限阶梯落地 `internal/gitintegration/permission.go`：`Level` L0-L3（PRD §18.2；L4/L5 不可表示 = out of scope by construction）、`CallerKind`（TallyDomain | AgentRuntime）、纯确定性 `Decider`（NewDecider/IsProtected/Evaluate/Authorize）、`Approval` 事实（protected-branch 策略批准，basis 必填）、不可伪造 `Authorization` 令牌（未导出字段，零值无效）+ 5 个新哨兵。`EnsureIntegrationBranch` 在候选身份之后、任何仓库交互之前强制目标绑定 L3 令牌；`Merge` 在目标 ref 校验后强制。**internal/engineeringpolicy/** 与 internal/contract/** 明确裁定零修改**（无 EngineeringPolicy 模式变化 per DL-08/BI-01；contract L0-L3 保障词汇是独立轴，消费不重定义——受保护集与批准是 gitintegration 内的类型值，非 EngineeringPolicy 字段）。
- AC：Agent runtime 无法经运行时协议请求 L3 操作 — PASS，三层结构各带测试：(1) 运行时协议只承载 Instruction/claim 词汇，无 Git 操作/能力，机器钉 import-scan 测试禁止 `internal/agentruntime`+`runtimes/**` 导入 `gitintegration`；(2) runtime 形调用者的 L3 请求被 `Decider` 以 `ErrRuntimeAuthorityExceeded` 拒绝——即使携带有效批准——且拿不到令牌；(3) `Merge`/`EnsureIntegrationBranch` 在触及任何仓库状态前拒绝零值/错绑令牌（真实临时仓库 + refs/HEAD 不动断言）。受保护分支需策略批准集成（无目标匹配 `Approval` → `ErrProtectedTargetDenied`；端到端批准合并测试）；L4/L5 fail-safe 不可表示。
- Verification：`./internal/gitintegration/` 41 测试（32 前 + 9 新：25 行确定性权限矩阵断言精确 basis + 重求值确定性、结构 AC、令牌生命周期、decider 校验/nil fail-safe、真实仓库负路径、import-scan 边界守卫）PASS；gitwriter/changeset/engineeringpolicy/contract 181 测试 PASS（零改动证明）；`./test/arch/...` 100 PASS；vet/gofmt/`git diff --check` 清洁，新文件 0 CR。PRODUCTION_REACHABILITY: N/A(reason: P5 lane 冻结接缝模式，组合根接线归后续任务；令牌门在操作内部强制，接线落地即生效)。Risks：`NewDecider(nil)` 无声明受保护 ref 时保护为 opt-in（组合根负责声明受保护集）；`Approval` 来源仅校验形状/目标身份，真实性归治理/组合（与 `AcceptanceFact` 同纪律）；运行时调用者上限 L1（DL-03 读法：runtime 无 Git 持久权限），`Integrate`(L2) 故意未动——仍由验收事实流水线把门。Full evidence: `reports/P5-04_protected_branch_and_git_permission_policy_l0_l3.md`。


## Goal

Enforce that runtime execution never receives integration authority and protected branches require policy-approved local integration.


## Allowed / primary paths

- `internal/gitintegration/**`
- `internal/engineeringpolicy/**`
- `internal/contract/**`


## Implementation requirements

1. Keep permission decision in Tally domain/composition.
2. L4 remote and L5 destructive remain out of release scope.


## Acceptance criteria

- [ ] Agent runtime cannot request L3 operation through runtime protocol.


## Verification

- Permission matrix tests.

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
1. **Tests:** Tests: architecture runtime-boundary tests + policy matching tests + new permission matrix
2. **Contract / Interface:** Interface: Git permission decisions belong to Tally policy/compiled contract, not agentruntime
3. **Core Implementation:** Core: L0–L3 enums/decision projection at integration layer
4. **Failure Handling:** Failure: runtime cannot request L3 by adding a tool/capability; L4/L5 remain out of release scope

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
