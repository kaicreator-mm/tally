> **Current-source deployment guide.** This deploys the current repository, including the v1.3 Trust Core. v1.4 closed-loop features are added incrementally and must not be assumed present merely because the v1.4 PRD exists.

# 部署文档

> 依据当前源码事实编写(`cmd/server`、`cmd/worker`、`cmd/tally`、`Makefile`、`internal/web`、`migrations/`)。目标场景:**单机自部署、单用户**。运行期日常运维(备份、CLI 全量子命令、webhook 报文格式、排查表)见 `operations.md`,本文只回答「一台新机器,怎么从零跑起来」。

---

## 1. 拓扑与依赖

```text
            ┌──────────────┐        ┌──────────────┐
  浏览器 ──▶│ bin/server   │──入队──▶│  PostgreSQL  │◀──取任务── bin/worker
            │ :8080 界面   │        │  队列+事实面  │          │
            └──────────────┘        └──────────────┘          │ 写 git 内容仓库/事实面
  CI ──POST /webhook/build───────────▶ server                   (TALLY_CONTENT_REPO)
```

| 依赖 | 版本 | 必需性 |
|---|---|---|
| Go | 1.23+ | 仅构建期需要;也可在别处交叉编译后拷贝二进制 |
| PostgreSQL | 16+ | **必需**。唯一状态存储(队列、事实面、索引面、会话) |
| git CLI | 任意较新版本 | **必需**(worker 的仓库根解析与一次性 worktree 走 `git` 命令;内容仓库初始化/提交是 go-git,不依赖 CLI) |
| Docker / SSH / kamal / ansible / compose | — | 可选。仅在真正执行环境拉起/部署动作时按部署定义需要 |
| LLM API(OpenAI 兼容) | — | 可选。仅 `req interview`、失败诊断、`harvest` 三处需要;不配则这三处不可用,其余照常 |

三个二进制全部无状态,可随时杀掉重启。

## 2. 构建

```bash
git clone <本仓库> tally && cd tally
make build        # 产出 bin/server bin/worker bin/tally(Windows 下为 .exe)
```

数据库迁移**内嵌在二进制里**(`migrations/embed.go`),部署机不需要单独带迁移文件。

## 3. 准备数据库

`Makefile` 的 `make dev-db` 引用 `docker-compose.dev.yml`,**该文件当前不在仓库里**,此目标不可用。自备一个 PostgreSQL 实例即可,例如:

```bash
docker run -d --name tally-pg \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=tally \
  -p 5432:5432 postgres:16
```

连串统一用环境变量 `DATABASE_URL`(三个进程都读它):

```bash
export DATABASE_URL='postgres://<用户>:<密码>@<主机>:5432/tally?sslmode=disable'
```

然后跑迁移(幂等,会确保库存在):

```bash
./bin/tally migrate       # 等价 make migrate
./bin/tally health        # 验证连通性
```

## 4. 环境变量

全部配置走环境变量,没有配置文件。**自部署至少要改 `TALLY_PASSWORD` 和 `DATABASE_URL`。**

### 核心

| 变量 | 缺省 | 谁用 | 说明 |
|---|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/tally?sslmode=disable` | 三进程 | Postgres 连接串 |
| `TALLY_PASSWORD` | `tally-dev`(启动时告警) | server | 界面登录密码。**生产必设**;改动后需重启 server 生效 |
| `TALLY_ADDR` | `:8080` | server | HTTP 监听地址 |
| `TALLY_WORKER_ADDR` | `:8081` | worker | worker 健康检查(`GET /healthz`)监听地址 |
| `TALLY_CONTENT_REPO` | `./tally-content` | server + worker | git 内容仓库路径。**两个进程必须指向同一路径**,建议写绝对路径;server 启动时幂等初始化(git init) |

### webhook / 日志 / 诊断

| 变量 | 缺省 | 谁用 | 说明 |
|---|---|---|---|
| `TALLY_WEBHOOK_TOKEN` | —(未设) | server | 设置后启用 `POST /webhook/build`(Bearer token 鉴权);**不设置则该路由整体 503**,这是推荐的默认状态 |
| `TALLY_LOG_DIR` | `./tally-logs` | server | webhook 原始日志留存目录(`raw_log_ref` 记路径,规则改了能重跑) |
| `TALLY_DIAGNOSE_AUTO` | 关 | 三进程 | `=1` 时失败录入后按指纹自动入队一次诊断(由 worker 消费);默认关,诊断手动触发 |

### LLM(可选)

| 变量 | 缺省 | 说明 |
|---|---|---|
| `TALLY_LLM_API_KEY`(或 `OPENAI_API_KEY`) | — | LLM API key |
| `TALLY_LLM_BASE_URL` | `https://api.openai.com/v1` | 可指向任意 OpenAI 兼容端点 |
| `TALLY_LLM_MODEL` | `gpt-4o-mini` | 模型名 |

### worker 执行相关(可选)

| 变量 | 缺省 | 说明 |
|---|---|---|
| `TALLY_WORKTREE_ROOT` | `<系统临时目录>/tally-worktrees` | 诊断候选执行的一次性 worktree 父目录。**必须落在 owner checkout 之外**;建议设为持久磁盘上的绝对路径 |
| `TALLY_JOB_LEASE` / `TALLY_JOB_HEARTBEAT` | 60s / 20s | 任务租约窗口与心跳间隔(Go duration,如 `90s`、`2m`);长构建任务可调大 |
| `TALLY_SIGN_NAME` / `TALLY_SIGN_EMAIL` | `tally-worker` / `tally-worker@localhost` | worker 写内容仓库的提交身份(spec-defect 提案等) |

### v1.3 执行通道(可选,P1 Codegen/Verify)

| 变量 | 缺省 | 说明 |
|---|---|---|
| `TALLY_V13_RUNTIME` | `fake` | runtime 适配器:`fake` 或 `codex`(决策锁:FakeRuntime 优先) |
| `TALLY_V13_CODEX_BIN` | — | `codex` 适配器的可执行文件路径 |
| `TALLY_V13_WORKTREE_ROOT` | `<系统临时目录>/tally-v13-worktrees` | v1.3 一次性 worktree 父目录 |
| `TALLY_V13_HOLDOUT_ROOT` | —(未设) | held-out 验证通道根。**契约含 held_out binding 而此项未设时,任务被拒绝执行**(INV-VER-05,fail-closed) |
| `TALLY_V13_PROFILE` / `TALLY_V13_MODEL` / `TALLY_V13_BEHAVIOR` / `TALLY_V13_ENV` | `default-coding` / `default` / `v1` / `worker-local` | runtime profile/model/行为版本/环境名 |

### CLI 署名(可选)

| 变量 | 说明 |
|---|---|
| `TALLY_HARVEST_NAME` / `TALLY_HARVEST_EMAIL` | `tally harvest` 写 `catalog/` 的 commit 署名 |
| `TALLY_SIGN_NAME` / `TALLY_SIGN_EMAIL` / `TALLY_SIGN_KEY` | `tally req sign` 署名人;`TALLY_SIGN_KEY` 设 gpg key 时给签署 commit 签名 |

## 5. 进程管理

server 与 worker **都要跑**:界面动作由 server 入队、worker 执行,只跑 server 的话动作会一直停在「已入队」。建议用 systemd 托管(环境变量写在 unit 的 `Environment`/`EnvironmentFile` 里):

```ini
# /etc/systemd/system/tally-server.service
[Unit]
Description=tally server
After=network.target postgresql.service

[Service]
User=tally
WorkingDirectory=/srv/tally
EnvironmentFile=/srv/tally/tally.env
ExecStart=/srv/tally/bin/server
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/tally-worker.service
[Unit]
Description=tally worker
After=network.target postgresql.service

[Service]
User=tally
WorkingDirectory=/srv/tally
EnvironmentFile=/srv/tally/tally.env
ExecStart=/srv/tally/bin/worker
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now tally-server tally-worker
```

要点:

- 两个 unit 用**同一个** `EnvironmentFile`,保证 `DATABASE_URL` 与 `TALLY_CONTENT_REPO` 一致。
- `WorkingDirectory` 决定相对路径缺省值(`./tally-content`、`./tally-logs`)落在哪里;生产建议所有路径环境变量都写绝对路径。
- 运行用户需要对 `TALLY_CONTENT_REPO`、`TALLY_LOG_DIR`、`TALLY_WORKTREE_ROOT`(及 `TALLY_V13_*_ROOT`)可写,且能调用 `git`。
- server 本身是纯 HTTP;对外暴露时建议置于 nginx/caddy 等反代之后做 TLS。

## 6. 首次部署验证清单

按顺序执行,每步通过再走下一步:

```bash
# 1. 数据库连通 + 迁移齐
./bin/tally health && ./bin/tally migrate

# 2. 两个健康检查端点
curl -fsS http://127.0.0.1:8080/healthz    # → ok
curl -fsS http://127.0.0.1:8081/healthz    # → ok

# 3. 浏览器打开 http://<host>:8080 → 跳 /login,用 TALLY_PASSWORD 登录成功
# 4. 从界面(如环境页)触发一个动作,几秒内状态不再停留「已入队」
#    —— 证明 server→队列→worker→事实面 整条链通
```

## 7. 升级

```bash
cd /srv/tally/src && git pull
make build                                  # 或在 CI 构建后分发二进制
cp bin/server bin/worker bin/tally /srv/tally/bin/
/srv/tally/bin/tally migrate                # 迁移幂等,先跑再重启
sudo systemctl restart tally-server tally-worker
```

- 迁移向前只增;二进制先升级、迁移先应用均以「先 migrate 再重启进程」为稳妥顺序。
- 怀疑索引面不一致时:`bin/tally reindex -check` 只 diff 不写;确认后 `make reindex` 从 git + 事实面重算。

## 8. 备份与恢复(摘要)

真正必须定期备的只有两样,其余可重算或可丢弃:

| 数据 | 方式 |
|---|---|
| 事实面表(`build_run`/`deploy_run`/`environment_run`/`failure_event`/`dependency_change` 等) | `pg_dump`,**唯一不可再生数据** |
| 内容仓库(`TALLY_CONTENT_REPO` 指向的 git 仓库) | `git push` 到远端 |

恢复:还原 dump → `bin/tally migrate` 补齐迁移 → `make reindex` 重算索引面 → 内容仓库 clone 回原路径。详表见 `operations.md`「备份与恢复」。

## 9. 已知差异与坑

- `make dev-db` 引用的 `docker-compose.dev.yml` **不在仓库中**,该目标当前不可用(见 §3,自备 Postgres)。
- `TALLY_PASSWORD` 的 bcrypt 哈希在 server **启动时**计算:改密码后必须重启 server,否则登录永远失败。
- 只启动 server 不启动 worker 不是合法部署形态:界面动作入队后无人消费。
- `TALLY_CONTENT_REPO` 两端不一致时,worker 读不到界面写入的定义,动作看似入队却永远不生效。
- 迁移由 `goose` 库经 `cmd/tally` 执行,**不依赖** goose CLI。
