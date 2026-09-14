# 运维手册

自部署、单用户。这份文档回答一件事:**换一台机器,怎么把它跑起来、怎么备份、怎么用 CLI。**

设计和边界不在这一份 —— 架构见 `01-architecture.md`,铁律见 `02-invariants.md`,数据模型见 `03-data-model.md`。

---

## 三个进程

| 进程 | 职责 | 默认端口 | 状态 |
|---|---|---|---|
| `cmd/server` | HTTP 界面(含写路由)、登录/CSRF、入队 jobs | `:8080` | 无 |
| `cmd/worker` | 消费 jobs 队列:拉环境、部署、跑验证、调 LLM | `:8081`(仅 `/healthz`) | 无 |
| `cmd/tally` | 运维 CLI:迁移、重算索引、手动触发 | 无 | 无 |

全部无状态,状态都在 Postgres 和 git 里。任何一个都可以杀掉重启,不丢东西。

`server` 和 `worker` **都要跑**:界面上的动作(拉起/部署/回滚/诊断)由 `server` 入队、`worker` 执行。只跑 `server` 的话,动作入队后没人执行,页面会一直停在「已入队」。

---

## 环境变量

全部配置走环境变量,没有配置文件。缺省值面向本地开发,**自部署时至少要把 `TALLY_PASSWORD` 和 `DATABASE_URL` 改掉**。

### 核心

| 变量 | 缺省 | 谁用 | 说明 |
|---|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/tally?sslmode=disable` | 三进程 | Postgres 连接串 |
| `TALLY_PASSWORD` | `tally-dev` | `server` | 界面登录密码(明文,bcrypt 后比对)。**未设置时启动告警,生产必须设** |
| `TALLY_ADDR` | `:8080` | `server` | HTTP 监听地址 |
| `TALLY_WORKER_ADDR` | `:8081` | `worker` | worker 健康检查监听地址 |
| `TALLY_CONTENT_REPO` | `./tally-content` | `server` + `worker` | 内容仓库路径(git)。**两个进程必须指向同一个** |
| `TALLY_WEBHOOK_TOKEN` | — | `server` | webhook 鉴权 token;**未设置则 `/webhook/build` 整体未启用(一律 503)** |
| `TALLY_LOG_DIR` | `./tally-logs` | `server` | webhook 留存原始日志的目录(`raw_log_ref` 记路径,规则改了能重跑) |
| `TALLY_DIAGNOSE_AUTO` | — | 三进程 | 诊断自动触发开关(T11e):`=1` 时失败录入(webhook / `ingest-build` / 执行端部署环境失败)后按指纹自动入队一次诊断;**默认关**(诊断手动触发,D1) |

### 次级(CLI / LLM)

| 变量 | 缺省 | 谁用 | 说明 |
|---|---|---|---|
| `TALLY_HARVEST_NAME` / `TALLY_HARVEST_EMAIL` | — | `tally harvest` | 抽服务定义时写 `catalog/` 的 commit 署名 |
| `TALLY_SIGN_NAME` / `TALLY_SIGN_EMAIL` | — | `tally req sign` | 需求签署人(可用 `-name`/`-email` flag 覆盖) |
| `TALLY_SIGN_KEY` | — | `tally req sign` | 设置后用该 gpg key 给签署 commit 签名 |
| `TALLY_LLM_API_KEY`(或 `OPENAI_API_KEY`) | — | LLM 调用方 | LLM API key |
| `TALLY_LLM_BASE_URL` | `https://api.openai.com/v1` | LLM 调用方 | LLM API 端点(可指向兼容服务) |
| `TALLY_LLM_MODEL` | `gpt-4o-mini` | LLM 调用方 | 模型名 |

LLM 只在三处被调(铁律 1):`requirement` 问询、`failure` 诊断、`harvest` 抽取。没有 LLM 配置时,这三处的功能不可用,其余照常。

---

## 首次跑起来

```bash
make dev-db     # (可选)用 docker 起一个 Postgres;已有本地 Postgres 可跳过
make migrate    # 建库 + 跑全部迁移(幂等)
make build      # 编出 bin/server bin/worker bin/tally
```

然后:

```bash
export TALLY_PASSWORD='你的密码'
./bin/server    # 界面在 http://localhost:8080
./bin/worker    # 另一个终端,消费动作队列
```

浏览器开 `http://localhost:8080`,跳到 `/login`,用 `TALLY_PASSWORD` 登录。

---

## 登录

- 校验:`server` 启动时把 `TALLY_PASSWORD` 用 bcrypt 哈希,登录时比对;**密码不明文落盘、不进数据库**。
- 会话:登录成功签发 cookie `tally_session`,TTL 7 天,`HttpOnly` + `SameSite=Lax`;会话存 Postgres `session` 表(可随时清空,清了只是要重新登录)。
- 边界(INV-9):所有写路由(POST 等)都挂在登录 + CSRF 中间件之后。公开路由只有 `/login`、`/healthz`、`/static/*`;`/webhook/build` 在登录组外但 token 鉴权(见「webhook 接收构建日志」),未启用/错 token 一律非 2xx。
- 单用户:**没有账号体系、没有注册、没有找回密码**。忘记密码就改 `TALLY_PASSWORD` 重启 `server`,旧会话随之失效。

---

## make 目标

| 目标 | 干什么 |
|---|---|
| `make dev-db` | 用 docker compose 起一个开发 Postgres(已有可跳过) |
| `make migrate` | 跑数据库迁移(幂等) |
| `make build` | 编出三个二进制到 `bin/` |
| `make check` | **提交前必跑**:lint + 架构测试 + 单元测试 + 集成测试 |
| `make reindex` | drop 索引面并从 git + 事实面重算 |
| `make sqlc` | 从 SQL 重新生成 store 代码 |
| `make proto` | 在浏览器看界面原型(`web/prototype/`) |

---

## `cmd/tally` 子命令

全部经 `bin/tally <子命令>`(或 `go run ./cmd/tally <子命令>`)。除标注外都读 `DATABASE_URL`。

| 子命令 | 干什么 |
|---|---|
| `migrate` | 确保库存在 + 跑全部未应用迁移(幂等) |
| `reindex [-repo P -project ID] [-check]` | drop 索引面并从 git + 事实面重算;`-check` 只 diff 不写(CI 自检,发现有人绕过单向数据流) |
| `reindex-fingerprints` | 用留存的原始日志(`failure_event.raw_log_ref`)重算历史指纹(规则版本参与;只读不写事实面) |
| `scan [-toolchain X] <目录>` | 批量扫日志样本,输出碰撞率实验结果(见 `docs/EXPERIMENT-collision.md`) |
| `ingest-build -project P -commit SHA [-toolchain X] [-log <文件>] [-status failed\|passed] [-workdir D] [-repro CMD] [-targets C1,C2]` | 录入一次构建:失败时同时写 `build_run` + `failure_event`(接上生产链:失败 → 指纹 → 诊断);缺省读 stdin 当日志,接 CI 管道。`-workdir/-repro/-targets` 是复现上下文(T11b),诊断 G1/G2/G3 用;`TALLY_DIAGNOSE_AUTO=1` 时失败录入后自动入队诊断 |
| `deps -project P -repo PATH [-from REV] [-to REV] [-content-repo PATH] [-no-retire]` | 采集依赖升级事实:逐提交 diff 锁文件(`go.mod`/`package-lock.json`)写 `dependency_change`;重跑安全(唯一索引去重),供失败时间线与升级对齐。采完顺带跑修复退休检查(T11c):本次变更碰过的已验证修复重跑复现命令(连跑 5 次),不再复现的退休(`RETIRED.md` + 清索引面);`-no-retire` 可关,`-content-repo` 给知识库仓库 |
| `env <up\|down\|rebuild> -def <环境定义.yaml>` | 手动拉起/推倒/重建环境(写 `environment_run`) |
| `deploy -def <部署定义.yaml>` | 部署已构建制品(写 `deploy_run`;调 kamal/ansible/compose) |
| `deploy rollback -def <部署定义.yaml> [-digest D]` | 回滚到指定 digest(调工具自带回滚,写 `deploy_run`) |
| `harvest [-service a,b] [-repo PATH] [-list]` | 抽服务定义:LLM 提案(含健康检查)→ 实际拉起验证 → 写 `catalog/`;`-list` 只列清单 |
| `req lint <file>` | 需求契约五闸检查(结构化问题清单) |
| `req interview <file>` | LLM 把实质缺口转成问题(带候选答案) |
| `req sign <file>` | 签署:产出内容哈希 + git 提交(`TALLY_SIGN_KEY` 时 gpg 签名) |
| `health` | 检查数据库连通性 |

---

## webhook 接收构建日志

`tally ingest-build` 的手动运输层之外的自动入口(T11d):CI 构建跑完,把结果 POST 给
`server` 即完成录入。语义与 CLI 完全一致 —— 失败构建写 `build_run` + `failure_event`,
成功构建只写 `build_run`。

**启用**:设置 `TALLY_WEBHOOK_TOKEN` 再启动 `server`。未设置时路由整体返回 503
(明确「未启用」,不做默认放行)。鉴权是单一静态 token,走 `Authorization: Bearer`,
常量时间比较;webhook 不在 CSRF 组内(非浏览器交互,token 即凭据)。

```bash
curl -X POST http://localhost:8080/webhook/build \
  -H "Authorization: Bearer $TALLY_WEBHOOK_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "project": "tally",
    "commit": "'"$CI_COMMIT_SHA"'",
    "toolchain": "go",
    "status": "failed",
    "exit_code": 2,
    "ci_url": "'"$CI_RUN_URL"'",
    "combo": {"go": "1.23.0"},
    "workdir": "/srv/build/tally",
    "repro": "make build",
    "other_targets": ["make worker"],
    "log": "...构建日志全文..."
  }'
```

- 请求体其余字段:`artifact`(成功时的制品 digest)、`raw_ref`(CI 自己留存了原始日志
 时给路径;缺省由 server 落盘到 `TALLY_LOG_DIR/<项目>/<UTC时刻>-<commit 前 12 位>.log`
  并记进 `raw_log_ref`)。
- 尺寸:请求体 12MB 封顶,`log` 单条 10MB(PRD 同限),超了 413 不录入。
- `workdir`/`repro`/`other_targets` 是复现上下文(T11b):带上,诊断 G1/G2/G3 才跑得起来。
- 应答(JSON):`{"build_run": <id>, "failure_event": <id>, "raw_log_ref": "<路径>"}`;
  成功构建 `failure_event` 为 0。
- 不做 CI 平台专属适配(GitHub/GitLab 各自 schema)—— 通用 JSON,CI 侧一行脚本拼装。
- 自动诊断(T11e,可选):`TALLY_DIAGNOSE_AUTO=1` 时,失败构建录入后按失败指纹自动入队
  一次诊断,由 `worker` 消费(与界面「重诊断」同一条路径)。默认关;成功构建永不入队。

---

## 备份与恢复

按 `03-data-model.md` 的分面,**备份策略各不相同**:

| 数据 | 在哪 | 能不能重建 | 怎么备 |
|---|---|---|---|
| **事实面**(`build_run`/`deploy_run`/`environment_run`/`failure_event`/`dependency_change`) | Postgres | **不能** | **必须备**:`pg_dump`。这是唯一不可再生的数据 |
| **索引面**(`clause`/`commit_clause`/`test_clause`/`fingerprint_stat`/`trace_edge`) | Postgres | 能 | **可不备**:`make reindex` 从 git + 事实面重算 |
| **内容仓库**(项目/需求/文档/Wiki 笔记) | git(`TALLY_CONTENT_REPO`) | git 自身 | `git push` 到一个远端即备份 |
| **已验证服务定义 / 修复库** | git(`catalog/`、`knowledge/`,在 tally 自己的 repo) | git 自身 | 随 tally repo 一起备份 |
| `session` 表 | Postgres | 可清空 | 不备(清了只是重新登录) |

**一句话:真正必须定期备的只有两样 —— 事实面表(`pg_dump`)和 git 内容仓库。** 其余要么能重算,要么丢了无所谓。

恢复:还原 Postgres dump → `make migrate` 补齐迁移 → `make reindex` 重算索引面 → git 内容仓库 clone 回 `TALLY_CONTENT_REPO` 指向的路径。

---

## 内容仓库

写用户内容(项目登记、需求条款、文档、Wiki 笔记)落到 `TALLY_CONTENT_REPO` 指向的 git 仓库,每次写一次 commit。`server` 启动时调 `EnsureRepo` 幂等初始化(不是 git 仓库就 `git init`)。

- **版本/diff/回滚由 git 管**,界面不另存副本。要看历史、要回滚某次编辑,直接在这个仓库里用 git。
- **`server` 和 `worker` 必须指向同一路径**(单机自部署),否则 worker 读不到界面写入的定义。

---

## 常见排查

| 现象 | 先看 |
|---|---|
| 界面动作一直「已入队」没结果 | `worker` 是不是没跑;`TALLY_CONTENT_REPO` 两端是否一致 |
| 登录就跳回 `/login?error=1` | `TALLY_PASSWORD` 是否改过后没重启 `server`(哈希在启动时算) |
| 诊断/问询/harvest 报 LLM 错 | `TALLY_LLM_API_KEY` / `TALLY_LLM_BASE_URL` 是否可用 |
| 迁移报错连不上库 | `tally health`;`DATABASE_URL` 指向的 Postgres 是否在跑 |
| 怀疑索引面被改乱 | `tally reindex -check` 只 diff 不写,看是否有人绕过单向数据流 |
