# P6-03 — Outcome evidence projection for Rules

- **Phase:** P6
- **Kind:** implementation
- **Wave:** W37
- **Complexity:** M
- **Execution model:** LOW
- **Depends on:** P3-01, P6-02
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO

## Execution status

- **Status: DONE** (2026-09-13) — Outcome → 治理证据投影桥落地 `internal/policygovernance/outcome_evidence.go`（新文件；`internal/outcome/**` 零改动）:`OutcomeEvidenceRef(outcome.Event) (EvidenceRef, error)` —— 合法事件(kind 五类封闭 + 至少一个绑定 + 显式时刻,outcome 自身 Validate)映射为 `EvidenceOutcome` 引用,引用 = 事件自然键 绑定/kind/UTC 时刻(P6-02 的精确重放抑制键,append-only 日志内唯一、可独立重导出;绑定优先级 task>changeset);`ProposalEvidence` 批次投影保持日志次序、任一不合法整体失败(证据面零静默裁剪)。**AC 结构保证**:治理面只经本桥读 append-only 事件(调用方经 outcome.Store 既有 ForTask/ForChangeset 读面取值,本包零数据源);边界守卫测试静态扫描 engineeringpolicy 全部源文件断言零 outcome 导入 —— 运行时策略匹配面与 Outcome 零耦合。被拒替代方案:outcome.Store 增带 ID 全日志读(需 sqlc 查询,store §10 热点不在路径)→ 改用自然键引用,零 store 改动。
- 验证:`DATABASE_URL=… go test ./internal/policygovernance/ -count=1` PASS 76.3s(全包;新增 4 测试:映射+稳定性+绑定优先级/三类负例+批次 fail-closed/次序保真/边界守卫);arch/build/vet/gofmt/`git diff --check` 清洁。执行过程:平台模型请求故障彻夜持续,编排者主会话完成。Full evidence: `reports/P6-03_outcome_evidence_projection_for_rules.md`。


## Goal

Expose Outcome references as evidence inputs for rule proposals/replay without coupling runtime policy to mutable analytics.


## Allowed / primary paths

- `internal/policygovernance/**`
- `internal/outcome/**`


## Implementation requirements

1. Implement only the minimum change required by this task and preserve all v1.4 invariants.


## Acceptance criteria

- [ ] Governance reads append-only events; runtime policy matching does not query Outcome directly.


## Verification

- Projection/query tests.

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
1. **Tests:** Tests: outcome queries + policy governance replay tests
2. **Contract / Interface:** Interface: read-only Outcome evidence refs into governance
3. **Core Implementation:** Core: projection/query layer, not EngineeringPolicy runtime matcher
4. **Failure Handling:** Failure: analytics/history unavailability cannot alter active runtime policy semantics

## External reference pack

- No external implementation is required. Prefer current Tally source as the canonical implementation reference.

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
