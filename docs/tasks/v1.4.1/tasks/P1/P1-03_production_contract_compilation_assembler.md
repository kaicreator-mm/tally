# P1-03 — Production contract compilation assembler

- **Phase:** P1
- **Kind:** implementation
- **Wave:** W10
- **Complexity:** L
- **Execution model:** HIGH
- **Depends on:** P1-02
- **L3 reference research:** COMPLETE in this pack
- **Initial status:** TODO


## Goal

Create the production seam that compiles and binds the same-source task bundle before runtime execution.


## Allowed / primary paths

- `internal/contract/**`
- `internal/artifacts/**`
- `internal/engineeringpolicy/**`


## Implementation requirements

1. Use existing `contract.Compile/Bind`.
2. Policy loader rejections are fail-safe.
3. Compiled identity/hash exposed to caller.


## Acceptance criteria

- [ ] No supported task starts execution without Compiled contract.
- [ ] Inputs are reproducible from source commit.


## Verification

- Integration tests across policy match/no-match/invalid policy/unknown risk.

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
- **Reason:** Creates the production compilation authority and combines policy/risk/assurance inputs from one source snapshot.
- **Review rule:** HIGH review required because the task touches an authority boundary, release gate, or high-blast-radius projection even when implementation is deterministic.

## Reference order — mandatory

Use evidence in this order. Do not jump directly to an external example before reading Tally's tests/contracts.
1. **Tests:** Tests: `contract/compile_test.go::{TestBindPinsSnapshotAndIsDeterministic,TestBindCompilesRequiresDoubleDuty}`, `test/arch/v14_contract_authority_test.go`
2. **Contract / Interface:** Interface: `contract.Compile`, `contract.Bind`, `Compiled.Hash/Pins/Coverage/Requires/GateKeys`
3. **Core Implementation:** Core: `contract/compile.go`, policy snapshot loader, risk derivation
4. **Failure Handling:** Failure: unpinned policy, rejected policy artifact, source mismatch and invalid risk basis stop execution

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
- Report: `docs/tasks/v1.4.1/reports/P1-03_production_contract_compilation_assembler.md`
- Implemented: 生产合同编译装配缝 `(*artifacts.ExecutionBundle).CompileAuthority() → *artifacts.CompiledAuthority`(新 `internal/artifacts/v141_contract_assembler.go`):从同源执行 bundle(P1-01 `LoadBundle`)到权威已编译合同的唯一生产入口(PRD §16,Worker 不绕过编译器),完全复用既有权威、零平行体系(DL-18)—— 入口复核来源提交一致(`ErrSourceMismatch`,DL-04 对手工拼装 bundle 的防御)、策略装载拒收 fail-safe(`ErrPolicyRejections` 阻断执行 —— P1-01 显式留给本任务的裁决:pinned 快照里存在却解析失败的策略意味着运行可见规则集不完整,带着缺口执行就是策略假阴性)、权威风险推导接线(`DeriveRisk` 类型化错误原样穿透 —— invalid risk basis 停止执行,P1-02 的委托,绝不从非证据推导「无风险」)、`contract.Compile`(内容推导 pin,第二道身份防线)与 `contract.Bind`(既有编译器零改动,DL-04 纯投影不重写权威);unmatched/unknown 风险**不阻断** —— 既有 INV-V14-09 语义原样保留(Coverage 可见 + 全条款 fail-safe 上调 L3);零策略 bundle 合法编译为 `ConfidenceNone`(无信号时 L1 默认地板 —— P1-03 的第二项留裁决,阻断无策略仓库不是任何权威文档的要求,裁决已文档化并由测试钉住);产出暴露任务 id、观测来源提交、bundle 身份与权威 `risk.Result`,连同 `Compiled.Hash/Pins/Coverage/Clauses/FrozenGates/Requires/GateKeys`(实现要求 3:编译身份/哈希暴露给调用方)。`internal/contract/**`、`internal/engineeringpolicy/**`、`internal/intentevidence/**`、`internal/risk/**` 全部零改动。
- Acceptance: 「No supported task starts execution without Compiled contract」— PASS(缝级):`CompileAuthority` 是 bundle → 执行权威的唯一入口,任一 fail-safe 触发(拒收策略/无效风险基础/来源不一致/混合身份/合同聚合与编译拒绝)都不产出 Compiled;Worker 调用时序(编译先于 workspace/runtime 创建、失败不启会话)归 P1-04 的 owned wiring(`cmd/worker/v14.go` 是 §10 热点)。「Inputs are reproducible from source commit」— PASS:装配是 bundle 的纯函数,同 bundle 重复装配、同一提交复装载再装配均得同一编译哈希与 bundle 身份,策略内容换版编译哈希必变,编译 pin 与 bundle 合同逐项相等(编译器不改写权威)。
- Validation(任务 Verification:policy match/no-match/invalid policy/unknown risk 全矩阵,经真实 freeze→commit→load→assemble 生产路径):`TestCompileAuthorityPolicyMatch`(`db_migration` task_type 确定性事实推导 schema_migration 命中 gate 策略、basis 断言、确定性/复现/输入敏感)、`TestCompileAuthorityNoMatchKeepsFailSafe`(未命中已知信号 → 编译成功 + L3 fail-safe;零策略零信号 → L1 地板)、`TestCompileAuthorityUnknownRiskVisible`(枚举外提示保真可见 + L3)、`TestCompileAuthorityRejectsInvalidPolicy`(拒收策略阻断,不误判)以及负例 `TestCompileAuthorityRejectsInvalidRiskBasis` / `TestCompileAuthorityRejectsSourceMismatch` / `TestCompileAuthorityRejectsMixedIdentity`。命令与结果:`go build ./...` PASS;`go vet ./internal/artifacts/` PASS;`go test ./internal/artifacts/ -run 'TestCompileAuthority' -v` 7/7 PASS;`go test ./internal/artifacts/... ./internal/contract/... ./internal/engineeringpolicy/... ./internal/intentevidence/... ./internal/risk/... ./internal/assurance/...` PASS;`go test ./test/arch/...` PASS(DL-04/DL-18 守卫完好);`go test ./internal/...` PASS(46 包);`gofmt -l` 两个新文件 clean;`git diff --check` clean;`make check` NOT_RUN(Git Bash 无 make 二进制,构成命令逐条执行);integration e2e(`-tags=integration ./test/e2e/...`)NOT_RUN(DB-gated opt-in,任务 Verification 矩阵即上列真实 Git 仓库夹具测试)。
- Baseline failures: none。
- Decision Locks touched: none(DL-04 经入口同源复核 + 复用 Compile/Bind 满足;DL-05 经派生值透出、无持久化满足;DL-09/BI-01 经拒收策略 fail-safe 阻断裁决满足;DL-18 零平行体系、纯复用;DL-17 遵守,hidden/ 未读;DECISION_LOCKS.md 文件未动)。
- Risks: 零策略裁决(编译不阻断)是本任务拥有的一项 judgment call,已文档化并由测试钉住,若产品后续要求阻断属本缝一行改动;P1-04 尚未在 worker 调用本缝 —— PRD §16 的 no-bypass 目前是缝级而非组合级事实;`risk.Result` 以派生值随 `CompiledAuthority` 透出(DL-05),下游(P1-06/P1-07)应从此处消费或从 pinned 提交重推导,不得持久化为权威;HIGH review 待执行(重点:§3 fail-safe 裁决表,尤其零策略裁决,与 §2 缝方向)。
- Commit: 见本文件 git 历史(`P1-03: …`)。
