# P6-03 — Outcome evidence projection for rules

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 16 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `8c75afe`(P3-09 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md` → DECISION_LOCKS DL-12(五类封闭枚举)+ DL-10 → `internal/outcome/{types,store}.go`(Event 值语义 + Validate 硬约束:kind 封闭/至少一个绑定/显式时刻;ForTask/ForChangeset 既有 append-only 读面;P6-02 的精确重放键 = kind+坐标+时刻) → 本 lane `internal/policygovernance/{proposal,outcome_evidence}.go`(EvidenceRef/EvidenceOutcome 词汇,P3-01 落地)。

外部参考: 无。

**接缝选择理由**: AC = 「治理读 append-only 事件;运行时策略匹配不直接查询 Outcome」。最窄接缝 = 纯投影桥(新文件):`OutcomeEvidenceRef` 把合法 outcome 事件映射为 `EvidenceOutcome` 证据引用(引用 = 事件自然键 绑定/kind/时刻 —— P6-02 的精确重放抑制键,append-only 日志内唯一、可独立重导出);`ProposalEvidence` 批次投影保持日志次序、任一不合法整体失败(证据面不做静默裁剪)。outcome 事件值经调用方从 `outcome.Store.ForTask/ForChangeset`(既有 append-only 读面)取得 —— 本包零数据源持有、零查询。

**被拒替代方案**: (a) 给 outcome.Store 增加带行 ID 的全日志读取 —— 需要新增 sqlc 查询(internal/store §10 热点不在本任务路径);改用事件自然键(P6-02 已裁定其在日志内唯一)作引用,零 store 改动。(b) 让 engineeringpolicy 感知 outcome —— 直接违反 AC;边界守卫测试钉死。

## 2. 语义变化(新文件)

- `outcome_evidence.go`:`OutcomeEvidenceRef(outcome.Event) (EvidenceRef, error)`(fail-closed:五类封闭 kind + 至少一个绑定 + 显式时刻;引用 = 绑定/kind/UTC RFC3339Nano 自然键);`ProposalEvidence([]outcome.Event) ([]EvidenceRef, error)`(次序保真,整体 fail-closed);`OutcomeImportBoundary` 常量。
- `internal/outcome/**`:零改动。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/policygovernance/ -count=1` | PASS 76.3s(全包;新增 4 测试:引用映射 + 稳定性 + 绑定优先级 task>changeset / 三类负例 + 批次整体 fail-closed / 次序保真 / 边界守卫) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

边界守卫 `TestEngineeringPolicyMustNotImportOutcome`:静态扫描 engineeringpolicy 全部非测试源文件,断言零 `tally/internal/outcome` 导入(运行时匹配面与 Outcome 零耦合 —— AC 的结构证明)。

## 4. 基线失败观察

两轮断言笔误(引用 String() 形态 vs Ref 字段)当轮修正;非基线失败。

## 5. 风险

- 引用自然键含时刻(RFC3339Nano):P6-02 裁定 RecordedAt 为发生坐标,时刻变更即不同事实 —— 引用稳定性以此为界;若未来 outcome 增加显式事件 ID 列,投影可无痛升级(持有 store 的任务执行)。
- 投影不校验事件在日志中的真实存在性(纯值函数)—— 存在性由调用方的读面保证(ForTask/ForChangeset 即日志读)。

## 6. Decision Locks

未触碰、未绕过:DL-12(五类枚举只读消费)、DL-10(重放证据输入面,无推断)、DL-08(engineeringpolicy 零改动 + 边界守卫)、DL-19(投影只读)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P6-03 提交信息)。
