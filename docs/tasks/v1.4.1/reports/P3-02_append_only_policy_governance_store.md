# P3-02 — Append-only policy governance store

- **执行者**: 编排者主会话(平台模型请求故障持续;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `3fe6fad`(P3-01 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(§10:sqlc 生成文件 internal/store/** 不在本任务路径,存储以消费者定义的最小面直用 pgx 池 —— pgx 在 INV-6 白名单内) → `DECISION_LOCKS.md` BI-01/DL-08/09 + DL-14(append-only 事件捕获先例) → `migrations/0023_v14_outcome_event.sql`(append-only 事件表惯用法:tally_prevent_modify 语句级触发器、CHECK 封闭枚举、显式 occurred_at 无 DB 缺省) → `internal/store/testdb.go` + `internal/outcome/store_test.go`(每测试独立 schema + goose + skip-if-unreachable 的测试基建惯例) → `internal/policygovernance/proposal.go`(本 lane P3-01 的域转移,重放侧的封闭规则)。

外部参考: 无。

**接缝选择理由**: AC「提案历史可重建;不引入可变运行时策略权威」→ 事件表 + 读侧重放是唯一同时满足两者的最小面(不建「当前 ACTIVE」表/布尔列 —— 激活只是事件行,Git 提交才是运行时生效面,P3-07/DL-09)。存储放 policygovernance 包内(消费者定义的 pgx 依赖面),不触碰 §10 热点 internal/store/**(sqlc 生成面);migrations 编号 0024 由本任务保留(PATH_OWNERSHIP)。

## 2. 语义变化

- `migrations/0024_policy_governance_event.sql`: append-only 治理事件表 —— `proposal_id`/`action_kind`(CHECK 封闭:draft|submit|activate|reject|supersede)/`candidate_rev`/`candidate_yaml`(所见即所批,非运行时权威)/`evidence_refs TEXT[]`(kind:ref 串)/`proposed_by`/`approved_by`/`supersedes`/`scope`/`occurred_at`(写入方显式给出);语句级 `tally_prevent_modify()` 触发器(0 行命中的 UPDATE/DELETE 也报错);Down = DROP。
- `internal/policygovernance/store.go`: `Event`/`EventKind`(封闭)+ `Store.Append`(唯一写路径,域侧 fail-safe 校验先行:未知 kind/缺批准人/缺时刻/缺候选内容即拒)+ `EventsForProposal`/`AllEvents`(id 升序 = 发生序)+ `Reconstruct`(按序重放域层封闭转移重建最终提案;空流 → `ErrUnknownProposal`;乱序/候选内容漂移/被污染证据串 → `ErrInconsistentHistory`,绝不猜测修补)。
- **列名裁定**: 治理动作列命名 `action_kind` 而非 `event_kind` —— `TestV141OutcomeKindSetStaysFive` 机械守卫(BI-02:全仓仅 outcome_event 定义 event_kind,禁平行 outcome 事件库)在开发中捕获了初版命名;治理动作语义上也是 action 而非 outcome event,改名使两域词汇正交,守卫原样保留(未改任何测试)。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test ./internal/policygovernance/ -count=1` | PASS 40.9s(5 DB-backed:全链重建 ACTIVE/证据/候选修订核对、reject 链终态、Append 负路径 ×4、读侧证据污染 fail-safe、DB 层 CHECK+UPDATE/DELETE 触发器+空流 ErrUnknownProposal、乱序历史 ErrInconsistentHistory;每测试独立 schema,skip-if-unreachable) |
| `go test ./test/arch/... -count=1` | PASS(1.700s;含 TestV141OutcomeKindSetStaysFive —— 开发中曾 FAIL,以 action_kind 改名裁决后恢复) |
| `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | PASS / 清洁 |

## 4. 基线失败观察

两轮自引入失败当轮修复(夹具假修订哈希被重放核对正确捕获 → 夹具改为真实解析派生;event_kind 列名与 BI-02 守卫冲突 → action_kind 改名)。非基线失败。

## 5. 风险

- `Reconstruct` 的重放校验强度 = 域层转移规则;跨提案的一致性(如 supersedes 指针指向存在且 ACTIVE 的前任)属 P3-04 replay / P3-07 激活链的契约面,本存储不隐式级联。
- 直用 pgx 面(非 sqlc):仅限本包自有表;不触碰 internal/store 的 sqlc 生成面 —— 若未来治理查询需要进 sqlc 契约,须由持有 internal/store/** 的任务执行。
- 无额外索引(README 纪律):治理事件流量极低,现有主键序扫描足够;数据量出现后再议。

## 6. Decision Locks

未触碰、未绕过:DL-08(治理词汇零进 Policy —— P3-01 结构负测试 + 本表分域)、DL-09(无 DB 运行时激活权威 —— 无 ACTIVE 表/列,激活是事件记录,Git 提交才是生效面)、DL-14 同型纪律(append-only 事件,无可变状态列)、BI-02(守卫全绿,词汇正交)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P3-02 提交信息)。
