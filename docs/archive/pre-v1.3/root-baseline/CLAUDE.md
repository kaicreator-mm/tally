# CLAUDE.md

写给 Claude Code 的项目指令。**每次开始工作前先读完这个文件。**

---

## 这是什么

`tally` —— 一个**自部署的项目管理 + 开发环境管理平台**,管作者自己的十来个项目,**带完整的管理界面**。做六件事:

| 包 | 干什么 |
|---|---|
| `environment` | 拉起开发/测试环境(docker 或 ssh),健康检查,记录版本组合 |
| `harvest` | 从 Coolify 等来源抽取服务定义,实际拉起验证后入库 |
| `deploy` | 把制品部到环境上(调 Kamal/Ansible/Compose,不自己实现) |
| `logpipe` | 日志收集 → 脱敏 → 抽取 → 规范化 → 指纹(**全程无 LLM**) |
| `ingest` | 统一失败录入编排:logpipe → `failure_event` + 指纹索引(构建入口 + 部署/环境经 exec 接入) |
| `failure` | 失败诊断:查库 → LLM 提案 → 重跑验证 |
| `knowledge` | 已验证修复知识库(**只收验证过的,铁律 2**;可编辑的 Wiki 笔记是另一层,见 `docs/06-ui-spec.md`) |
| `requirement` | 需求契约、linter、LLM 问询 |
| `project` | 项目、任务、追溯图 |

需求文档在 `docs/00-PRD.md`,**它是唯一权威**。本文件和它冲突时以 PRD 为准。

**用户是作者本人。单用户,自部署,不做多租户,不做产品。**

**v3 方向(2026-07-21):从「只读维护日志本」转为带完整写操作的管理平台。** 界面可以增改用户内容(项目/需求/文档/笔记)、触发动作(拉起/部署/回滚/诊断),但**不改状态、不改事实** —— 状态永远从执行记录算出(铁律 3),事实面永远只追加(铁律 5)。**单用户 + 简单登录**(密码/token),不做账号体系。详见下文「界面的写边界」。

---

## 七条铁律

违反任何一条都是架构性错误,不是可以权衡的取舍。写代码前对照检查。

### 1. LLM 只能提案,不能判定

任何"这个对不对"的问题必须由确定性代码回答。LLM 的输出永远是候选,永远要过验证器。

**机器检查:** `test/arch` 里有导入边界测试 —— `internal/logpipe`、`internal/knowledge`、`internal/store` **不允许导入 `internal/llm`**。改这个测试 = 违规。

### 2. 未验证的东西不入库

知识库、环境定义库只收通过实际验证的条目。不存"抽出来了还没试"的,不存"LLM 分析结论仅供参考"的。

理由:未验证的条目看起来和验证过的一模一样,以后你会信它,而这是自用工具,没人会来纠正。

### 3. 状态是算出来的,不给手动改

任务是否完成,看执行记录(合并了没、构建过了没、部署了没),不看有没有人点按钮。

**不要给任何状态字段提供 UPDATE 接口。**

### 4. 挂钩记内容哈希,不记 ID

需求条款、版本组合、制品这些,关联时存的是**内容的哈希**,不是逻辑编号。这样内容一变旧记录自动对不上,失效传播不需要额外机制。

### 5. 事实面只追加

`build_run`、`deploy_run`、`failure_event` 这类表只 INSERT,不 UPDATE 不 DELETE。派生数据(索引、统计)可以随时重算。

### 6. 不写别人已经写好的东西

不写部署引擎(用 Kamal/Ansible/Compose)、不写制品库(用 registry)、不写 git server、不写日志管道(不装 Loki/ELK)、不写运行期错误分析(那是 Sentry 的事)。

**这个工具的职责是调度 + 记账 + 验证。**

### 7. 只做有机器记录对应的功能

某个**跟踪字段**填不出机器来源,就不要这个字段。工时、指派、优先级、通知、看板、评审流程 —— 全部不做。

**v3 澄清:** 这条限的是*机器填不出的跟踪字段*,**不限用户创作的内容**。需求正文、文档、Wiki 笔记是人写的,界面可以编辑(写进 git),不在此条范围。任务「完成与否」不是用户填的字段,是从执行链算出来的(铁律 3),所以没有手动状态可改。

---

## 界面的写边界(v3 新增)

平台界面有两种「写」,**都合法**:

- **写用户内容** —— 增改项目、需求条款、任务定义、bug 注记、文档、Wiki 笔记。这些落 **git**(markdown/yaml),界面只是它的编辑器,版本/diff/回滚由 git 管。
- **触发动作** —— 一键拉起/推倒/重建环境、部署/回滚、触发诊断。这些**入队 `jobs`**,由 worker 执行,产生**新的事实记录**(INSERT)。

界面**不能**做的两种写:

- **改状态字段**(铁律 3)—— 没有「标记完成」按钮,没有任何 `UPDATE ...status`。状态从执行链算。
- **改事实面**(铁律 5)—— 事实表只由系统在执行动作时 INSERT,界面不碰,也不提供 DELETE。

**机器检查:** `test/arch/no_status_update_test.go` 禁状态 UPDATE 与状态路由;`test/arch/auth_required_test.go` 要求所有写路由经过登录中间件;`test/arch/fk_hash_test.go`、`append_only_test.go` 照旧。

---

## 技术栈

固定,不要替换:

| | |
|---|---|
| Go | 1.23+ |
| HTTP | 标准库 `net/http`(Go 1.22 的 ServeMux)+ `chi` 中间件 |
| 数据库 | PostgreSQL 16+ |
| SQL | **`sqlc`** —— 手写 SQL 生成类型安全的 Go。**不用 ORM** |
| 迁移 | `goose` |
| 模板 | 标准库 `html/template` + `go:embed`。**不引入前端构建步骤** |
| 交互 | HTMX(静态文件嵌入) |
| Git 读取 | `go-git` |
| SSH | `golang.org/x/crypto/ssh` |
| Docker | **shell 调用 `docker` CLI**,不用 SDK |
| LLM | 标准库 `net/http` 直接调 API,不用 SDK |
| 队列 | Postgres 表 + 进程内 worker。**不用 Redis,不用 Temporal** |

最后一条特别说明:任务图是固定的(收集→清洗→指纹→查库→提案→验证),不需要动态编排。引入编排引擎是过度设计。

---

## 目录结构

```
cmd/
  server/     HTTP 服务 + 嵌入的 UI
  worker/     后台任务执行器
  tally/      运维 CLI(手动触发、重算索引)
internal/
  core/       跨包共享的领域类型。保持极小,不放业务逻辑
  project/    项目、任务、追溯图
  requirement/需求契约、linter、问询循环
  environment/环境拉起与健康检查
  harvest/    服务定义抽取与验证
  deploy/     部署调度与记账
  logpipe/    日志流水线(无 LLM)
    toolchain/  按工具链(npm/go/docker/compose/kamal)的日志抽取器
  ingest/     统一失败录入编排(logpipe → failure_event;构建入口,部署/环境经 exec 接入)
  failure/    失败诊断与修复验证
  knowledge/  已验证修复知识库
  llm/        LLM 客户端。只被 requirement、failure、harvest 导入
  runner/     docker / ssh 执行抽象
  store/      sqlc 生成代码 + 仓储层
  jobs/       任务队列
  exec/       worker 侧执行器:把 job 队列接到 environment/deploy,写事实面;失败经 ingest 录入、诊断接 failure
  web/        handler(含写路由)+ 模板 + 登录/CSRF 中间件
    handlers/   各功能域的 HTTP handler(写路由层:内容 CRUD → git;动作 → 入队 jobs)
migrations/   goose SQL 迁移
web/
  prototype/  静态 HTML 原型(**只读参考,不参与构建**)
  templates/  实际使用的 html/template
  static/     CSS、HTMX
test/arch/    架构约束测试(导入边界等)
docs/         设计文档
tasks/        任务拆解。一次做一个
```

## 包依赖方向

```
web ──► project, requirement, environment, deploy, failure, knowledge
           │           │            │          │        │
           └───────────┴────────────┴──────────┴────────┘
                              │
                    ┌─────────┼──────────┐
                    ▼         ▼          ▼
                 store     jobs     runner / logpipe
                    │
                    ▼
                  core

llm ◄── requirement, failure, harvest   (仅此三个)
```

**规则:**
- `core` 不导入任何 internal 包
- `store` 只导入 `core`
- `logpipe` 只导入 `core`(**这是它无 LLM 的结构保证**)
- 业务包之间不互相导入。需要协作时由 `web` 或 `jobs` 编排

---

## 怎么干活

### 一次一个任务

`tasks/` 下每个文件是一个独立任务,有明确的交付物和验收标准。

1. 读 `tasks/README.md` 看依赖关系
2. 挑一个前置已完成的任务
3. 读那个任务文件
4. **先写测试,再写实现**
5. 跑 `make check`,全绿才算完成

### 开始新任务前

```bash
make check    # 确认当前是干净的
```

不绿的话先修,不要在红的基础上加东西。

### 提交信息格式

```
<类型>: <一句话说明>

Task: T04
Refs: REQ-12          # 有对应需求条款时必填
```

类型:`feat` `fix` `refactor` `test` `docs` `chore`

`Refs:` 这行是追溯图的数据来源(见铁律 4),T07 之后强制。

---

## 不要做的事

- **不要为了让测试过而改测试。** 特别是 `test/arch` 下的
- **不要引入新依赖**,除非任务文件里明确写了
- **不要写 mock 数据库。** 集成测试用真实 Postgres(见 `docs/05-test-standards.md`)
- **不要在实现里加 TODO 就交付。** 做不完就把任务拆小
- **不要一次做多个任务。** 一个任务一个提交
- **不要"顺便优化一下"。** 范围外的问题记到 `tasks/BACKLOG.md`

---

## 文档

| 文件 | 内容 |
|---|---|
| `docs/00-PRD.md` | 需求文档(v3,平台化方向)。**唯一权威** |
| `docs/01-architecture.md` | 各包职责、边界、数据流 |
| `docs/02-invariants.md` | 七条铁律的机器检查方式 |
| `docs/03-data-model.md` | 表设计与字段理由 |
| `docs/04-code-standards.md` | Go 代码规范 |
| `docs/05-test-standards.md` | 测试规范 |
| `docs/06-ui-spec.md` | 界面规格与原型说明 |
| `tasks/README.md` | 任务依赖图 |
