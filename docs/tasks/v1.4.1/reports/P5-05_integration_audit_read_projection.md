# P5-05 — Integration audit/read projection

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 16 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `f797f42`(P6-03 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(DL-19;DL-03) → DECISION_LOCKS DL-13(验收事实 append-only) → 本 lane 既有面:`internal/gitintegration/branch.go`(P5-02 确定性命名 integrationBranchRef:refs/heads/tally/integration/<sha256-of-ChangeSetID>)、`internal/web/handlers/v13_http_test.go`(审批证据测试惯例)、P2 投影模式(诚实空状态/禁词负测试/声明式接缝)。

外部参考: 无。

**接缝选择理由**: 契约 = 「Show Accepted and Integrated as separate observable states/facts;UI 永不把验收等同于集成」。最窄接缝 = (a) gitintegration 只读列举 `ListIntegrationRefs`(for-each-ref 锁定 refs/heads/tally/integration/ 命名空间,输出 ref 名升序,携带提交 sha/主题/时刻)+ 导出命名对账助手 `IntegrationBranchRefFor`(P5-02 命名规则的导出面); (b) 新页 `/v14/integration` 两分区:Accepted = 既有审批证据事实面(HEAD 快照 standard 任务的最近决定,复用 P3-08 同一事实面),Integrated = Git 集成分支事实。两分区数据来源独立、互不推导 —— AC 由结构保证。

## 2. 语义变化

- `internal/gitintegration/audit.go`(新):`ListIntegrationRefs`(只读 for-each-ref;零写路径)+ `IntegrationBranchRefFor`(命名规则导出面)。
- `internal/web/handlers/v14_integration.go`(新)+ `web/templates/v14_integration.tmpl`(新):双分区投影 —— Accepted 行携带集成状态标注(**经确定性命名对账**:有对应分支 = 已集成:<ref>,否则「尚未集成」;不对应分支的污染被对账拒绝);Integrated 区逐条 ref/commit/subject/date;边界声明区明示「验收 ≠ 集成」。治理命名空间之外的引用不见于集成事实。
- 接线:`RegisterV14Routes` 一行路由;base.tmpl 研发组补 集成 条目;nav known-routes 集合扩展(声明式最小接缝,P2-03/P3-06 同例);v14CA 模板组注册。
- `internal/approval/**`:零改动(Accepted 分区复用既有审批证据)。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/gitintegration/ -count=1 -run 'TestListIntegrationRefs' -v` | PASS(真实临时仓库:P5-02 具名操作落地分支后列举恰 1 条、ref/commit/subject/date 逐项核对、第二条分支升序确定;空仓库零行;非仓库 fail-safe) |
| `DATABASE_URL=… go test ./internal/web/handlers/ -run 'TestV14Integration' -count=1 -v` | PASS ×3(仅验收:Accepted 呈现 + 「尚未集成」标注 + Integrated 空;并存:Integrated 呈现分支事实 + 验收行标注经确定性对账不被无关分支污染;命名空间外引用不可见) |
| `DATABASE_URL=… go test ./internal/web/... -count=1` | PASS(web 0.27s + handlers 245.0s 全包) |
| `go test ./internal/gitintegration/ -count=1` | PASS 23.0s(全包回归:41+2 测试) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` / `gofmt -l` / `git diff --check` | 全 PASS / 清洁 |

## 4. 基线失败观察

无基线失败;一轮自引入 vet 错误(未用 import)与包名笔误当轮修复。

## 5. 风险

- 集成对账只覆盖「有对应集成分支」的方向:P5-03 Merge 推进的目标 ref(合入主线)不在本列举面 —— 合并事实的呈现归后续整合任务(集成分支本身已是 P5-02 定义的持久集成事实)。
- 引用列举无分页:治理命名空间(refs/heads/tally/integration/)规模受治理流程约束,当前规模可接受。

## 6. Decision Locks

未触碰、未绕过:DL-19(全页只读零写路由)、DL-03(列举只读,写面仍归 P5-02/P3-07 具名操作)、DL-13(验收证据 append-only 原样呈现)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P5-05 提交信息)。
