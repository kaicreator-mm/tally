# P2-04 — Reviews projection with human-needed reason

- **执行者**: 编排者主会话（平台模型请求故障持续,第 4 次 general-purpose 子智能体派发失败后按最小阻塞项接管;先例见 P4-04 收尾）
- **日期**: 2026-09-13
- **基线**: master `dab7c48`（P4-04 之后）

## 1. L3 完成证据

**阅读顺序（契约规定）**: 本地契约 → `AGENTS.md`（DL-19 投影纪律）→ 参照测试 → 接口/事实面 → 最近实现接缝。实际读取：

1. 参照测试: `internal/web/handlers/v14_attention_test.go::TestV14AttentionListsBlockingFactsOnly`（阻断事实清单 + 禁词表:未发生的来源标签不得出现）; `internal/web/handlers/v13_http_test.go::TestV13ApproveWritesHumanApprovalEvidenceNotStatus`（审批=证据非状态,本任务未触碰该面,确认边界）。
2. 接口/事实面: `internal/web/handlers/v14_attention.go`（Attention 封闭词表 AC4:needs_review/blocked/failed_run/cancelled/exposure_compromise/projection_error;条目已带 Detail=原因、Refs=依据引用 fact:<id>/exposure:<id>）; `internal/taskstate/state.go`（openReviews 派生 StateNeedsReview;`v13FactsFor` 的 run/cs/vrun 事实映射——**review_requested 不在 v13 读路径映射内**,闭环升级的事实面是终态失败 Run）。
3. 最近实现接缝: `web/templates/v14_attention.tmpl`(边界声明节)、P2-02/P2-03 报告的投影模式(诚实空状态/逐字呈现/负测试钉禁词)。

外部参考: 仅 EXT-GO-HTML(html/template 上下文感知转义,新模板单元格 `{{.Action}}` 全部走数据插值,零 raw HTML)。

**接缝选择理由**: AC「每条评审项携带 reason/basis/目标动作」——reason=既有 Detail、basis=既有 Refs,唯一缺口是 target action 与「聚合」。最窄接缝 = 在 `V14AttentionItem` 增加固定 `Action`(封闭 per-kind switch,文本只复述本页既有边界声明中的人本裁决项)+ 页面 `Summary` 聚合小计(纯派生算术)。零新事实来源(要求 1)、零 `internal/review` 语义更名(要求 2)、零写路由、零第二可变状态。

## 2. 语义变化

- `V14AttentionItem.Action`: 每条评审项的固定目标动作(封闭 switch,无字符串拼接推断)。
- `V14AttentionPage.Summary`: 按类型排序位次序的非零类别计数(`V14AttentionCount{Kind,Label,Count}`)。
- 模板: h1 `Attention`→`需人工评审`(meta 保留 PRD §26.4 引用);表格新增「目标动作」列;条目区新增聚合小计行;边界声明补充「聚合小计与每条的裁决动作都从同一事实派生」。
- 表头/动作/聚合行只随条目渲染(空状态零动作字面)——负测试钉死。

## 3. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test ./internal/web/handlers/ -run 'TestV14Attention' -count=1 -v` | PASS ×2(既有 7.49s + 新增 7.08s;第一轮新测试 FAIL 暴露 meta 行泄漏「目标动作」字面于空状态,修措辞后转绿) |
| `DATABASE_URL=… go test ./internal/web/... -count=1` | PASS(web 0.274s + handlers 170.573s,全包) |
| `go test ./test/arch/... -count=1` | PASS(1.477s) |
| `gofmt -l`(触及文件) / `git diff --check` | 清洁 |

新增负测试面: 干净主线禁止出现 `目标动作`(表头)/`人工裁决后续`/`人工裁决 strict`/`项需人工事项`(聚合行);注入后禁止未发生类别的小计(`需要人工评审 ×`/`被显式阻塞 ×`/`投影失败 ×`)。

## 4. 基线失败观察

无——既有 attention 测试全断言不变绿。第一轮失败为本任务自引入的模板措辞泄漏,非基线。

## 5. 风险

- needs_review/blocked/cancelled 分支在 v13 读路径当前不可达(review 请求尚无可读取事实面;闭环升级的事实面=终态失败 Run,页头边界声明如实记载)——动作映射封闭就绪,事实面接入即生效。
- 「需人工」导航标签维持不变:P2-02 移交的改名议题裁定为不必要(AC 未要求;页面 h1 已人本化,无未支撑能力暗示)。

## 6. Decision Locks

未触碰、未绕过(DL-19 投影只读;hidden/ 未读 per DL-17)。`internal/review/**`/`internal/approval/**` 零改动。

## 7. Commit

由编排者提交(见 git log P2-04 提交信息)。
