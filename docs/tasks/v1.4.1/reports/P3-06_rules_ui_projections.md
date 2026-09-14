# P3-06 — Rules UI projections

- **执行者**: 编排者主会话(平台模型请求故障彻夜持续,子智能体派发连续 14 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `7101ce3`(P3-07 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(DL-19 投影只读;DL-09 运行时权威面) → DECISION_LOCKS BI-01/DL-08/09 → 本 lane 既有面:`internal/policygovernance/{proposal,store,conflict}.go`(提案域/事件存储+Reconstruct/冲突分析 —— 全部只读消费)、`internal/artifacts/v14_policy_loader.go`(LoadPolicies:Active 分区的唯一权威来源)、`internal/web/handlers/v14_attention.go`(v14CA 模板组与诚实呈现惯例)、P2-01..P2-05 投影模式(人本分区/禁词负测试/声明式接缝)。

外部参考: 无。

**接缝选择理由**: 契约 = 「Show Proposed/Active/Conflicts/History,Active 从真实 Git 策略快照派生」。最窄接缝 = 新页 `/v14/rules`(`v14_rules.go` + `v14_rules.tmpl`,注册进 RegisterV14Routes 与 v14CA 模板组)+ base.tmpl 研发组补 规则 条目(闭合 P2-01 预留)。Active 分区**只**从 `artifacts.LoadPolicies(repo, HEAD)` 派生;治理存储只提供 Proposed/History/一致性核对 —— 「UI 不得把 DB-only 提案显示为 Active」由数据流方向结构保证(Active 不读治理面)。

## 2. 语义变化

- `v14_rules.go`(新):四分区投影 —— **Active**(HEAD 快照策略 + 逐条治理一致性标注:提案一致(ACTIVE,携带提案 id)/ 不确定(治理面无对应记录或修订不一致 —— 一等可见));**治理分歧**(治理记 ACTIVE 但 HEAD 不可观察:同 id 缺席/修订不一致,注明运行时不受影响 DL-09);**Proposed**(未激活提案:策略 id/状态/候选修订/提案人/证据数/范围);**Conflicts**(P3-05 AnalyzeConflicts 对 Active 集合,只呈现不消解);**History**(治理事件流倒序截断 20 条 + 全长如实声明;重放失败的提案作为「治理历史不一致」行呈现,不静默)。治理未接入:记录面诚实呈现,Active(Git 权威面)照常。
- `v14_change.go`:v14CA 模板组注册 `v14_rules`(一行)。
- `v14_launch.go`:RegisterV14Routes 增加 `GET /v14/rules`(一行)。
- `base.tmpl`:研发组新增 规则→`/v14/rules`(闭合 P2-01/P2-03 注释预留)。
- `internal/web/nav_test.go`:known-routes 集合加 `/v14/rules`(声明式最小接缝;测试语义 = 新增条目必须先有真实路由 —— 路由已真实注册;P2-03 同例)。
- `internal/policygovernance/**`、`internal/artifacts/**`:零改动。

## 3. 开发期缺陷(自引入,表驱动测试捕获)

初版 `V14RulesPage.Active []V14ActivePolicy` 字段名**遮蔽**了嵌入 `Page.Active`(导航高亮键字符串),base.tmpl 的 `eq .Active "..."` 在渲染时类型错误且被渲染器吞错(响应截断至 495 字节)。调试定位后改名为 `ActivePolicies`;顺带补 Proposed 行的策略 id 列。教训已记录:嵌入 Page 的新页面字段不得复用 Page 既有字段名。

## 4. 测试与命令(分歧夹具逐项)

| 夹具 | 断言 | 结果 |
| --- | --- | --- |
| A 治理领先(PROPOSED + Git 空) | 提案出现在 Proposed 区;Active 区空;策略 id 全页恰 1 次(**绝不显示为 Active**) | PASS |
| B 分歧(治理 ACTIVE + Git 空) | 分歧区呈现「治理面 ACTIVE 但 HEAD 不可观察」+ 原因 + DL-09 注记;Active 区仍空 | PASS |
| C 一致(Git 提交 + 治理 ACTIVE 同修订) | Active 行 + 「治理提案一致(ACTIVE)」标注;零分歧行 | PASS |
| D 不确定(Git 有策略、治理无记录) | Active 行带「不确定:治理面无对应 ACTIVE 提案记录」;Git 权威面照常 | PASS |
| E 治理未接入 | 记录面诚实呈现;Active 照常 | PASS |

`DATABASE_URL=… go test ./internal/web/... -count=1`:PASS(web 0.29s + handlers 222.6s 全包;既有 nav/attention/review 测试不变绿)。`./test/arch/...`/`go build ./...`/vet/gofmt(触及文件)/`git diff --check` 全 PASS。

## 5. 风险

- 治理提案重建为 N+1(每提案一次 Reconstruct):治理面规模(数十)下可接受;AllEvents 无分页 —— 历史截断 20 条 + 全长声明。
- 分歧区的存在意味着治理面与 Git 面可能合法地暂时不一致(激活链第 4 步失败等)—— 页面如实呈现供人裁决,不自动消解(与 P3-05 同纪律)。

## 6. Decision Locks

未触碰、未绕过:DL-09(Active 只从 Git 快照派生 —— AC 的结构保证)、DL-08(engineeringpolicy 零改动)、DL-19(全页只读零写路由)。hidden/ 未读(DL-17)。

## 7. Commit

由编排者提交(见 git log P3-06 提交信息)。
