# P3-03 — Policy candidate canonicalization and validation

- **执行者**: 编排者主会话（平台模型请求故障持续,子智能体派发连续 10 次失败;按既定先例接管）
- **日期**: 2026-09-13
- **基线**: master `5193863`（P3-02 之后）

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md` → DECISION_LOCKS DL-08/09（运行时模式冻结、内容修订语义）→ `internal/engineeringpolicy/{parse,hash}.go`（KnownFields 严格解析;normalize → sortedUnique → RevisionHash/Canonical 的内容修订语义 —— P3-03 的验证对象）→ `internal/engineeringpolicy/{parse_test,hash_test}.go`（金色测试惯例）→ 本 lane 既有面 `internal/policygovernance/{proposal,store}.go`（P3-01 域 + P3-02 存储边界）。

外部参考: 无。

**接缝选择理由**: AC「候选修订 = 实际正典策略内容修订语义」+ 要求「经既有解析器验证;无效/重复身份 fail-closed;不建第二 DSL」。最窄接缝 = (a) `VerifyCandidate`（重放正典解析并核对声称修订 —— 内容可独立验证,不信任调用方自报）; (b) 把该验证前置到存储写入边界（Append 即校验,而非等到读侧重放才暴露）; (c) 重复身份在写入处 fail-closed（身份由首个 draft 铸造）。零新解析器、零新策略字段（engineeringpolicy/** 零改动 —— 运行时模式冻结,消费即可）。

## 2. 语义变化

- `canonical.go`（新）: `VerifyCandidate(raw, claimedRevision)` —— 经 `engineeringpolicy.Parse` 正典解析（不可解析 = `ErrInvalidProposal`）+ 声称修订一致性核对;空声称修订 = 只解析（返回候选）。
- `store.go` Append 增强: (1) 每个事件写入前过 `VerifyCandidate`（修订漂移/不可解析候选在写入处类型化拒绝,不再等到读侧）; (2) 重复身份 fail-closed —— 第二个 `draft` 事件 → `ErrDuplicateProposal`（新哨兵;append-only 流不允许身份重铸）。
- `internal/engineeringpolicy/**`: 零改动。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test ./internal/policygovernance/ -count=1` | PASS 54.2s(9 测试;新增 2 个 DB-backed:修订漂移/不可解析候选写入拒绝、重复身份 fail-closed + 非重复后续事件可落) |
| 金色解析/修订(新增 `canonical_test.go`,非 DB):`TestCanonicalRevisionGolden` | PASS —— 键序/空白/列表顺序三种排版变体修订稳定 + 正典形式逐字节一致;语义内容变化(requires 去一条)修订必变 |
| `TestCandidateRevisionEqualsCanonicalSemantics` | PASS —— 候选修订 = 独立 `engineeringpolicy.Parse → RevisionHash` 语义(AC);VerifyCandidate 一致通过/不一致拒绝/不可解析拒绝 |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

## 4. 基线失败观察

无——本轮全部一次通过。

## 5. 风险

- 金色稳定性测试锚定的是当前 `engineeringpolicy` 修订语义(normalize+sortedUnique);上游若改变哈希输入规范,金色向量会失败提醒重审(设计上可见,非静默)。
- 重复身份判定以「同 proposal_id 的第二个 draft」为准;跨提案的同内容重复(revision 相同的两个不同提案)是合法治理形态(supersede 流),不拒绝 —— 已在测试与文档中如实记录边界。

## 6. Decision Locks

未触碰、未绕过:DL-08（engineeringpolicy 零改动）、DL-09（修订=内容派生语义,可独立验证）、要求 1（零第二 DSL —— 全部复用既有解析器）。hidden/ 未读（DL-17）。

## 7. Commit

由编排者提交(见 git log P3-03 提交信息)。
