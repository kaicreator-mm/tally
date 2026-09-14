# P2-05 — Human-centered UX gate

- **执行者**: 编排者主会话（平台模型请求故障持续,子智能体派发连续失败;按 P4-04/P2-04 先例接管）
- **日期**: 2026-09-13
- **基线**: master `d39ab4d`（P2-04 之后）

## 1. 门禁裁决

**PASS** —— 三条 AC 全部可执行证明,零生产代码缺陷发现,零生产代码改动（门禁路径纪律:`test/**` + 本报告）。

## 2. 被测面与证据来源

生产路由原样装配（`handlers.Server.RegisterRoutes` —— 组合根同一注册函数,末尾组合 `RegisterV14Routes`;登录/CSRF 组行为由既有 web 层测试覆盖,与 P0-09 门禁同一注记）+ 真实内容仓库（p09Repo 同一生产引导）+ 真实事实面（`store.TestDB` 每测试独立 schema,共享 dev PostgreSQL）。P2 四页读序:P2-01..P2-04 报告 → base.tmpl/v14_launch.tmpl/v13_tasks.tmpl/v13_runs.tmpl/v14_attention.tmpl → v13env/v14Attention 处理器源码。

## 3. AC 判定（可执行证据:`test/e2e/v141_p205_ux_gate_test.go::TestP205HumanCenteredUXGate`,DB-gated,7.43s,复跑确定性 PASS）

### AC1 核心任务可不经 Chat-first 流创建/评审 — PASS
- 创建:launch 页唯一入口是有界结构化表单 `<form method="post" action="/v14/launch/intent">`,主控件为单个有标签 textarea（`<label for="v14-intent">意图(一句自然语言,唯一必填项)</label>` + `id="v14-intent"`）;后续决策全部经服务端有界澄清闭环（P0-03/04 封闭 kind 词表:choice/bool/confirm/number/text + 推荐标注不预选）。无 chat transcript 式自由多轮界面、无无边界的对话流 UI。
- 评审:工作列表（/v13/tasks,TaskContract 视图 + 派生状态）→ 任务详情（/v14/tasks/{task},P2-02 人本六分区 + compiled 基线）;需人工面（/v14/attention,P2-04 聚合:逐条原因/依据/目标动作 + 小计）。全部为结构化读模型,评审决策点带固定人本动作。

### AC2 键盘/基础表单可访问性 — PASS
- 表单控件 label[for] 关联（launch 22 个 label;澄清控件 checkbox/radio/number/text 逐一关联）。
- 表格有 th 表头（工作列表 `<th>Task</th><th>派生状态</th>…`,静态模板断言钉死）。
- 渲染面无空文本链接（`<a …></a>` 无可聚焦文本 = 键盘死角;正则负断言四页全过）。
- 导航条目带 `aria-current="page"`（P2-01 既有,门禁复核确认）。

### AC3 无任意 LLM 生成 UI 执行 — PASS
- 渲染面恰一个 `<script>`:base.tmpl 的静态 `<script src="/static/htmx.min.js" defer></script>`（仓库内静态文件、defer、零模板插值）——四页渲染断言 + 五模板静态断言（base 恰 1、其余 0）。
- LLM 来源值（意图/澄清/失败原因/运行身份）一律经 html/template 上下文感知转义作为数据渲染（P2 各页既有负测试:伪造候选/失败原因不渲染）,不存在内联脚本执行缝。

### 门禁目标:状态机词汇不主导导航 — PASS
导航轨道断言:四页渲染的研发组为人读标签（发起/工作/运行/需人工）,生命周期机器词面（NEEDS_REVIEW/VALIDATING/AWAITING_APPROVAL/CODEGEN/CHANGESET）零出现于 `<nav>` 段;机器词面仅作为页面内容区的 dim 词面小标（事实引用,非导航）。

## 4. 测试与命令

| 命令 | 结果 |
| --- | --- |
| `DATABASE_URL=… go test -tags=integration ./test/e2e/ -run 'TestP205HumanCenteredUXGate' -count=1 -v` | PASS(7.43s;首轮 FAIL 为门禁测试自身的期望修正:导航人读标签实为「需人工」而非页面 h1「需人工评审」——按 base.tmpl 事实修正期望,非放宽断言) |
| 复跑 `-count=1` | PASS(确定性;0.274s cached 后实跑 ok) |
| `go test ./test/arch/... -count=1` | PASS(2.117s) |
| `go vet -tags=integration ./test/e2e/` | PASS |
| `gofmt -l` / `git diff --check` | 清洁(新文件 LF) |

## 5. 基线失败观察

无生产缺陷。唯一失败轮次为门禁测试自身对导航标签文本的期望与 P2-01 实际落地文本的差异（需人工 vs 需人工评审）,以源事实为准修正。

## 6. 结构化评审清单（报告面）

| 页面 | 结构化优先 | 机器词面 confinement | 键盘可达 | 数据渲染 |
| --- | --- | --- | --- | --- |
| /v14/launch | 有界意图表单,唯一入口 | 导航零机器词面 | label[for] 全覆盖 | 转义数据 |
| /v13/tasks | TaskContract 读模型列表 | 派生状态仅内容区 dim 词面 | th 表头 | 转义数据 |
| /v13/runs | Run 事实逐字投影 | 期望/自报分离标注 | th 表头 | 转义数据 |
| /v14/attention | 逐条原因/依据/目标动作 + 小计 | 封闭词表 + 禁词负测试 | 链接有文本 | 转义数据 |

## 7. 风险

- 门禁的空状态面（0 任务仓库）覆盖导航/结构/无障碍/脚本面;带数据的表格行级可访问性由 P2 各页的 DB-gated 测试覆盖（P2-02/03/04 各自渲染断言）。
- htmx 为唯一 JS 依赖且为静态引用;未来若引入动态脚本面,AC3 断言（恰 1 个静态 script）将按设计失败提醒门禁复审。

## 8. Decision Locks

未触碰、未绕过（DL-19 投影只读;hidden/ 未读 per DL-17）。零生产代码改动。
