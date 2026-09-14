# P3-08 — Apply Once vs Propose Rule review action

- **执行者**: 编排者主会话(平台模型请求故障持续,子智能体派发连续 11 次失败;按既定先例接管)
- **日期**: 2026-09-13
- **基线**: master `c938631`(P3-03 之后)

## 1. L3 完成证据

**阅读顺序**: 本地契约 → `AGENTS.md`(DL-19;§10 热点 server.go 声明式最小接缝,P2-03 先例) → DECISION_LOCKS DL-08/09 + BI-01(治理词汇归属) → 参照测试 `v13_http_test.go::TestV13ApproveWritesHumanApprovalEvidenceNotStatus`(决定=append-only 证据非状态;URL/表单/flash 断言惯例) → 既有面 `internal/approval/{types,store}.go`(HumanApprovalEvidence 语义,复用不扩展)、`internal/web/handlers/v13_review.go`(V13Decide 流)、`internal/policygovernance/{proposal,store}.go`(P3-01/02/03:NewDraft 正典校验、Append fail-safe、Reconstruct)。

外部参考: 无实际使用(EXT-GITHUB-RULESETS/EXT-GITLAB-APPROVAL 为模式参考;两动作均落在 Tally 既有事实上,无外部模式可搬)。

**接缝选择理由**: 契约 = 「人可只解决一案,或另立规则提案;一次性决定绝不自动改策略」。最窄接缝 = 评审页两个**独立**表单:(a) 既有 decide 流原样保留,明确标注「仅此一次」(它本来就只写本候选绑定的 HumanApprovalEvidence —— 要求 1 的"既有语义 where applicable"); (b) 新增 propose-rule POST 落 policygovernance 草稿提案,证据引用指向本案,零审批证据写入(要求 2 + "Propose Rule never auto-approves")。两动作互不触发 —— AC 的「绝不自动」由结构保证(不同处理器、不同事实面)。

## 2. 语义变化

- `server.go`(§10 热点,声明式最小接缝):`Server.Governance *policygovernance.Store` 字段(nil → 提议新规按未接入呈现)+ 一行路由 `POST /v13/review/{csid}/propose-rule`。
- `v13_review.go`:`V13ProposeRule` 处理器 —— 候选 YAML 出生即正典解析校验(NewCandidate;治理字段混入即拒,P3-01/03 语义),身份铸造 + 草稿事件 Append(证据引用 changeset:<id> + task:<id>),零审批证据写入;flash 明示「绝不批准本候选;运行时策略不受影响」。
- `v13_review.tmpl`:decide 表单标注「仅此一次」语义;新增独立「提议新规」表单(候选 YAML/提案人/范围说明/取代指针,label[for] 全覆盖),治理未接入时诚实呈现不可用。
- `internal/approval/**`:零改动(既有证据语义复用)。

## 3. Reconstruct 缺口修复(P3-02 交付的改进)

开发中发现 `Reconstruct` 不回填 draft 事件携带的初始证据引用 —— 修复(一行: draft 分支回填 first.Evidence)。这是 P3-02 交付面的补全,非语义变更。

## 4. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test ./internal/web/handlers/ -run 'TestV13ProposeRule|TestV13ApplyOnce' -count=1 -v` | PASS ×4(happy:草稿重建 DRAFT + 候选正典解析 + 双证据引用 + 零审批证据;负例:治理字段污染/缺提案人 fail-closed 零残留;仅此一次:decide 照旧写恰一条审批证据且治理事件流为零;未接入:propose 显式失败而 decide 不受影响) |
| `DATABASE_URL=… go test ./internal/web/... -count=1` | PASS(web 0.27s + handlers 187.0s 全包) |
| `DATABASE_URL=… go test ./internal/policygovernance/ -count=1` | PASS 54.1s(Reconstruct 修复回归) |
| `go test ./test/arch/... -count=1` / `go build ./...` / `go vet` | PASS |
| `gofmt -l` / `git diff --check` | 触及文件同全仓 CRLF checkout 既有条件(P4-01 同型记录,内容层清洁,未做行尾规整)/ 清洁 |

## 5. 基线失败观察

一轮失败为 Reconstruct 证据回填缺口(P3-02 交付面改进,当轮修复);非基线失败。既有 approve 证据测试全断言不变绿。

## 6. 风险

- 治理存储的生产接线(`NewRouter` → `handlers.Server.Governance`)不在本任务路径(internal/web/router.go、cmd/server 不在 Allowed Paths)—— handler 级能力已就绪,nil 语义 fail-safe;接线归后续整合任务(PRODUCTION_REACHABILITY 记 N/A)。
- 提案身份在处理器内铸造(prop-<csid>-<unixnano>):同案可多次提议(各自成案);跨提案去重不适用(同内容重复提案是合法治理形态,P3-03 边界一致)。

## 7. Decision Locks

未触碰、未绕过:DL-08(治理字段混入候选即拒)、DL-09(运行时策略零触碰 —— propose 动作不写 Git、不改激活;decide 动作不触治理面)、BI-01(治理词汇住治理事实侧)。hidden/ 未读(DL-17)。

## 8. Commit

由编排者提交(见 git log P3-08 提交信息)。
