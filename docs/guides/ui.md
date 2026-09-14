# UI 操作说明

> **Current-source UI guide.** The `/v13/*` pages below are implemented today. v1.4 Launch / Closed Loop / Attention UI is target scope, not current source, until its P3 tasks land.

> 当前源码事实(`internal/web` + `web/templates`)。P1 Trust Core 能力只在任务完成、闸门证据通过后才算上线;本文只描述已存在于代码里的界面行为。

Tally 自带一个 **Go template + HTMX 管理界面**(无前端构建管线,静态资源内嵌二进制),由 `cmd/server` 在 `TALLY_ADDR`(默认 `:8080`)提供。界面原型可在 `web/prototype/`(`make proto`)浏览。

---

## 访问与登录

- 浏览器打开 `http://<host>:8080`,未登录自动跳 `/login`。
- **单用户体系**:无注册、无多账号、无找回密码。密码即部署时设置的 `TALLY_PASSWORD`(bcrypt 哈希后比对,不明文落盘)。
- 登录成功签发 cookie `tally_session`(TTL 7 天,`HttpOnly` + `SameSite=Lax`);忘记密码就改 `TALLY_PASSWORD` 并重启 `server`,旧会话随之失效。
- **写边界(INV-9)**:除 `/login`、`/healthz`、`/static/*` 外的所有路由都在登录 + CSRF 保护之后;`/webhook/build` 在登录组外、用 token 鉴权。

## 核心心智模型:界面只入队,worker 执行

界面上的所有动作按钮(拉起/推倒/重建环境、回滚、重诊断、Patch Review 以外的执行动作)**只向 Postgres jobs 队列写入一条 job**,由 `cmd/worker` 消费执行并写事实面。因此:

- **`server` 和 `worker` 必须都在跑**,否则动作会一直停在「已入队」。
- 页面状态(环境/部署/失败/任务)全部从事实面与索引面**算出、只读**,界面不存任何副本。
- 破坏性动作(推倒、重建、回滚)带警示色 + 二次确认。

---

## 页面清单

### 内容层(写 git 内容仓库 `TALLY_CONTENT_REPO`)

| 页面 | 路径 | 能做什么 |
|---|---|---|
| 首页 | `/` | 总览入口 |
| 项目管理 | `/projects`、`/projects/new`、`/projects/{slug}`、`/projects/{slug}/edit` | 项目登记/查看/编辑;写入内容仓库 `projects/`,每次保存一个 git commit |
| 需求条款 | `/requirements`、`/requirements/{project}/{file}` | 需求条款清单与编辑;写入内容仓库 `.tally/requirements/` |
| 任务状态 | `/tasks` | 从执行链算出的任务状态,**只读,无任何控件** |
| 文档 | `/docs`、`/docs/new`、`/docs/{slug}`、`/docs/{slug}/edit` | 文档查看/编辑;写入内容仓库 `docs/` |
| Wiki 笔记 | `/wiki`、`/wiki/new`、`/wiki/{slug}`、`/wiki/{slug}/edit` | 笔记层,页面标「**笔记·未验证**」;写入内容仓库 `wiki/` |

版本、diff、回滚由 git 管:界面不另存副本,历史与回滚直接在内容仓库里用 git 操作。

### 执行引擎层(状态只读,动作入队)

| 页面 | 路径 | 能做什么 |
|---|---|---|
| 环境 | `/environments` | 环境列表;动作:**拉起(up)/ 推倒(down)/ 重建(rebuild)**,POST 入队 |
| 部署 | `/deploys` | 部署记录;动作:**回滚(rollback)**,POST 入队。直接部署不走界面,用 `tally deploy -def` |
| 失败 | `/failures` | 失败事件列表(事实面) |
| 指纹 | `/fingerprint` | 失败指纹详情(聚合与影响面) |
| 诊断 | `/diagnose` | 诊断状态(第几轮 / 五分类);POST **触发重诊断**(入队,后端守三轮上限 `failure.ConsultBudget`,界面无法绕过) |

### 追溯与知识层(全部只读)

| 页面 | 路径 | 内容 |
|---|---|---|
| 已验证修复库 | `/knowledge` | 已验证修复浏览,**只读,无编辑入口(INV-2)** |
| 条款覆盖率 | `/coverage` | 需求条款的验证覆盖率(无 project 列索引时为总览) |
| 视图 A | `/views/a` | 条款累计失败 |
| 视图 B | `/views/b` | 指纹影响面(关联条款 LEFT JOIN) |

### v1.3 P1 最小界面(PRD §26.2)

| 页面 | 路径 | 内容 |
|---|---|---|
| Tasks | `/v13/tasks` | 标准 TaskContract 任务列表 |
| Change·Run Detail | `/v13/changes/{csid}` | 单个 ChangeSet 的运行详情 |
| Patch Review | `/v13/review/{csid}` | Owner 审阅候选补丁;唯一写动作 **decide**(批准/拒绝)只落 `HumanApprovalEvidence`(append-only 证据),不写任务状态字段——状态由 State Engine 从证据派生 |

数据源是事实面 + Git 制品(`.tally` 契约与 base..candidate diff);二者缺席时页面渲染「未接入」空状态,不报 500。

### 交互约定

- 所有表单 POST 后 303 重定向回列表页并带提示条(`?kind=&flash=`),刷新不会重复提交。
- CSRF token 在每个表单隐藏域中,由中间件注入。
- 页面数据来源(`Queue`/`Exec`/`Knowledge`/`Trace`)任一未接线时,对应页面渲染**空状态 + 行动提示**,而不是报错。
- HTMX 局部刷新 + 内嵌 `tally.css`;无需 Node/前端构建。

---

## 常见界面问题

| 现象 | 原因 |
|---|---|
| 动作一直「已入队」没结果 | `worker` 没在跑,或 `server`/`worker` 的 `TALLY_CONTENT_REPO` 指向不一致 |
| 登录一直跳回 `/login?error=1` | 改了 `TALLY_PASSWORD` 但没重启 `server`(哈希在启动时计算) |
| 诊断/问询/harvest 报 LLM 错 | `TALLY_LLM_API_KEY` / `TALLY_LLM_BASE_URL` 不可用;LLM 只影响这三处,其余功能照常 |
| 页面大片「未接入/空状态」 | 对应数据面尚未迁移或 Postgres 中无事实记录;按页面上的行动提示处理 |
