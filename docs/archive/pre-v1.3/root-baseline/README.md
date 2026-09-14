# tally

自部署的开发运维工具。管自己的十来个项目:拉环境、部署、抓失败、记住修过的问题。

**单用户,自部署。不做多租户,不做产品。**

---

## 从哪开始

**如果你是 Claude Code:先读 [`CLAUDE.md`](CLAUDE.md)。**

| | |
|---|---|
| 需求文档(唯一权威) | [`docs/00-PRD.md`](docs/00-PRD.md) |
| 架构 | [`docs/01-architecture.md`](docs/01-architecture.md) |
| 七条铁律与机器检查 | [`docs/02-invariants.md`](docs/02-invariants.md) |
| 数据模型 | [`docs/03-data-model.md`](docs/03-data-model.md) |
| 代码规范 | [`docs/04-code-standards.md`](docs/04-code-standards.md) |
| 测试规范 | [`docs/05-test-standards.md`](docs/05-test-standards.md) |
| 界面规格 | [`docs/06-ui-spec.md`](docs/06-ui-spec.md) |
| 运维手册(配置/进程/备份/CLI) | [`docs/07-operations.md`](docs/07-operations.md) |
| 任务拆解 | [`tasks/README.md`](tasks/README.md) |

界面原型:`make proto` 然后打开 http://localhost:8081

---

## 它干什么

```
环境    读 repo 里的定义,用 docker 或 ssh 拉起开发/测试环境,跑健康检查,记录版本组合
部署    把已构建的制品部上去 —— 调 Kamal / Ansible / Compose,不自己实现
日志    收集 → 脱敏 → 抽取 → 规范化 → 指纹,全程不用 LLM
失败    查库命中就直接给答案;没命中才让 LLM 提方案,过四道闸才算数
知识    只存验证过的修复,带着验证时的版本组合
需求    LLM 拷问你把想法问清楚;状态从执行记录算,不给手动改
```

## 它不干什么

不碰生产环境 · 不写部署引擎 · 不写制品库 · 不写 git server ·
不做常驻日志管道 · 不做运行期错误分析 · 不做团队协作功能

---

## 跑起来

```bash
make dev-db      # 本地 Postgres
make migrate
make build
make check       # 提交前必跑
```

然后起两个进程(界面动作由 server 入队、worker 执行,两个都要跑):

```bash
export TALLY_PASSWORD='你的密码'
./bin/server     # 界面在 http://localhost:8080
./bin/worker     # 消费动作队列
```

浏览器开 `http://localhost:8080`,跳到 `/login`,用 `TALLY_PASSWORD` 登录(缺省 `tally-dev`,自部署务必改掉)。

### 运行配置

全部走环境变量,没有配置文件。核心的几个:

| 变量 | 缺省 | 说明 |
|---|---|---|
| `DATABASE_URL` | 本地 `tally` 库 | Postgres 连接串 |
| `TALLY_PASSWORD` | `tally-dev` | 登录密码,**生产必改** |
| `TALLY_ADDR` / `TALLY_WORKER_ADDR` | `:8080` / `:8081` | 两个进程的监听地址 |
| `TALLY_CONTENT_REPO` | `./tally-content` | 内容仓库(git),server/worker 须一致 |

完整清单(含 LLM、CLI 署名等)见 [`docs/07-operations.md`](docs/07-operations.md)。

### 目录

```
cmd/         三个进程入口(server / worker / tally CLI)
internal/    各业务包(environment deploy logpipe failure knowledge requirement project ...)
migrations/  goose SQL 迁移
web/         界面模板、静态资源、原型
docs/        设计文档(00 需求 → 07 运维)
tasks/       任务拆解,一次做一个
```

各包职责见 [`docs/01-architecture.md`](docs/01-architecture.md)。

---

## 七条铁律

1. LLM 只能提案,不能判定
2. 未验证的东西不入库
3. 状态是算出来的,不给手动改
4. 挂钩记内容哈希,不记 ID
5. 事实面只追加
6. 不写别人已经写好的东西
7. 只做有机器记录对应的功能

每条都有对应的自动检查,见 `docs/02-invariants.md`。**写在文档里的规矩会被忘,写成测试的不会。**
