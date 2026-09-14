# 架构

## 三个进程

| 进程 | 职责 | 有状态? |
|---|---|---|
| `cmd/server` | HTTP 服务、界面(**含写路由**)、登录/CSRF、接收 webhook、入队任务 | 无 |
| `cmd/worker` | 执行任务:拉环境、部署、跑验证、调 LLM | 无 |
| `cmd/tally` | 运维 CLI:重算索引、手动触发、导出 | 无 |

全部无状态,状态都在 Postgres 和 git 里。可以杀掉重启,不丢东西。

单机跑,`server` 和 `worker` 可以是同一台机器上的两个进程,也可以合成一个二进制用参数区分。**不做分布式。**

---

## 数据存哪

### git —— 人写的东西

| 内容 | 位置 |
|---|---|
| 需求条款、验收条件、非目标、术语表 | 各项目 repo 的 `.tally/requirements/` |
| 环境定义、健康检查 | 各项目 repo 的 `.tally/environments/` |
| 部署定义(Kamal 配置等) | 各项目 repo,路径由环境定义指定 |
| 已验证的服务定义库 | tally 自己的 repo 的 `catalog/` |
| 已验证的修复 | tally 自己的 repo 的 `knowledge/` |
| **Wiki 笔记层**(可编辑、未验证,v3) | tally 自己的 repo 的 `wiki/` |
| **项目登记**(项目清单、repo 位置,v3) | tally 自己的 repo 的 `projects/` |

**为什么在 git:** 这些东西 agent 要直接读写,文件比数据库记录好用;而且版本、diff、回滚 git 已经给了,不用再造。

### Postgres —— 机器产生的东西

两类,不要混:

**事实面(只追加,不可重建)**

`build_run` `deploy_run` `environment_run` `failure_event` `dependency_change`

发生过就是发生过。只 INSERT。

**索引面(派生,随时可删重算)**

`clause` `clause_version` `commit_clause` `test_clause` `fingerprint_stat` 等

全部可以从 git + 事实面重新算出来。

> **判据:** 这条数据能不能从别处重新算出来?能 → 索引面。不能且是人写的 → git。不能且是机器产生的 → 事实面。

**还有一张运维表(v3):** `session`(登录态)。可随时清空(清掉只是要重新登录),不属事实面、不属索引面。密码哈希放配置(`x/crypto/bcrypt`),不进数据库。

---

## 主流水线

```
                         ┌──────────────┐
   构建失败 ────────────►│              │
   部署失败 ────────────►│   logpipe    │  收集→脱敏→抽取→规范化→指纹
   环境拉起失败 ────────►│  (无 LLM)    │
                         └──────┬───────┘
                                │ 精确指纹 + 模糊指纹
                                ▼
                         ┌──────────────┐
                         │  knowledge   │  查库
                         └──────┬───────┘
                    ┌───────────┼───────────┐
         精确命中 ◄─┘           │           └─► 都没命中
              │            只有模糊命中            │
              ▼                 │                  ▼
       直接返回修复              └─────────►  ┌──────────┐
       (零 LLM 调用)              当线索      │ failure  │
                                              │ LLM 提案 │
                                              └────┬─────┘
                                                   ▼
                                            ┌──────────────┐
                                            │ 验证:重跑    │
                                            │ 红 N 次      │
                                            │ 改           │
                                            │ 绿 N 次      │
                                            │ 其他不受影响 │
                                            └──────┬───────┘
                                          通过 │        │ 不通过
                                               ▼        ▼
                                          写 knowledge  归类存档
                                          (git commit) (五种出口)
```

---

## 各包职责

### `core`

跨包共享的领域类型:`ProjectID` `Fingerprint` `VersionCombo` `ClauseRef` 等。

**不放业务逻辑,不导入任何 internal 包。** 如果一个类型只有一个包用,放那个包里,不要放这。

### `runner`

执行抽象。两种实现:

```
Runner interface {
    Run(ctx, spec) (Result, error)   // Result 含 stdout/stderr/exitCode/duration
}
```

- `DockerRunner` —— shell 调 `docker` / `docker compose`
- `SSHRunner` —— `x/crypto/ssh`

**只负责执行和返回原始结果,不解释结果。** 判断成功失败是调用方的事。

### `environment`

读环境定义 → 调 `runner` 拉起 → 跑健康检查 → 记录版本组合。

失败时把原始输出交给 `logpipe`。

**健康检查是必须的。** 没有健康检查的环境定义拒绝注册,这是硬校验不是警告。

### `harvest`

从 Coolify 等来源抽服务定义。流程:

```
llm 产出候选定义(含健康检查)
  → environment 实际拉起
  → 健康检查过 → 写 catalog/ + 记版本组合
  → 不过 → llm 改,最多三轮 → 还不行就跳过
```

**这是 `llm` 的三个合法调用方之一。** 但判定权在 `environment` 的健康检查上,不在 LLM。

### `deploy`

决定部什么 → 调外部工具 → 记账。

**不实现零停机、回滚、健康等待。** 那些是 Kamal / Ansible 的活。这个包做三件事:

1. 解析部署定义,确定制品 digest 和目标环境
2. 调 `runner` 执行外部工具
3. 写 `deploy_run` 记录(digest、目标、结果、版本组合)

**硬校验:输入必须是已存在的制品 digest。** 不接受"拉代码然后构建"的部署定义 —— 目标机上构建会让指纹失去意义。

### `logpipe`

六个阶段,每个阶段是独立可测的纯函数:

```
Collect  → 从 CI API / runner 输出 / 容器日志取原始日志
Redact   → 脱敏(粗)
Extract  → 去 ANSI → 定位标记 → 取片段
Redact   → 脱敏(细)
Normalize→ 洗掉路径/时间戳/PID/地址/临时名
Fingerprint → 算精确指纹 + 模糊指纹
```

**规则版本号参与指纹计算。** 规则一改,新旧指纹自然不会混淆;要对齐历史数据,用留存的原始日志重跑。

**这个包不导入 `llm`,由架构测试强制。**

### `ingest`

统一失败录入编排:把三类失败(构建 / 部署 / 环境)接进 `failure_event` + `fingerprint_stat`,
补上了生产链「失败已算出指纹,却没人写 `failure_event`」的断点。

```
原始日志 → logpipe(按工具链选抽取器)→ failure.RecordEvent(同事务写事实面 + 累加索引面)
```

- 构建入口 `IngestBuild`:先写一行真实 `build_run`(来源外键约束),构建失败再接着录 `failure_event`;成功构建只落 `build_run`,不录失败事件
- 部署 / 环境失败:业务包返回带 runID + logpipe 指纹的结构化错误,由 `exec` 编排层调本包录入(**业务包不导入 ingest**,方向是 `exec` → `ingest` → `failure`)
- 只 INSERT 事实面(INV-5);只录失败;允许导入 `core`、`store`、`logpipe`(含 `toolchain`)、`failure`,反向不依赖

### `failure`

诊断与验证。**验证器在这里,不在 LLM 里。**

四道闸,顺序固定:

| 闸 | 检查 |
|---|---|
| G1 | 连跑 N 次(默认 5)全部失败 —— 确认稳定复现;不稳定 → 归类为偶发 |
| G2 | 应用修复后连跑 N 次全部成功 |
| G3 | 其他项目/目标的构建不受影响 |
| G4 | 修复只改了白名单内的文件 |

G4 白名单:依赖清单、锁文件、Dockerfile、CI 配置、compose/部署清单、资源限制、健康检查定义。

- 改到应用源码 → 暂停,等人确认
- 改到验证用的东西 → **直接拒绝**

五种出口,必须显式归类:`fixed` `flaky` `upstream` `infra` `spec_defect`。

`spec_defect` 要回写到 `requirement`,触发条款失效传播。

**重试上限三轮**,超了归档为"暂无自动修复",以后同指纹命中直接返回该结论,不再调 LLM。

**缺复现上下文时不假装验证**:`failure_event` 不带复现命令/工作目录,诊断装载出来 `Repro` 为空
→ 跳过 G1/G2(不提候选、不验证、零 LLM),次数照计、预算仍三轮归档;知识库精确命中短路不受影响。

### `knowledge`

三元组存储:`(指纹, 复现命令, 已验证修复 + 验证时的版本组合)`。

两种匹配待遇不同:

- **精确指纹命中** → 直接返回修复
- **模糊指纹命中** → 只作为线索交给 `failure` 的提案阶段,**不返回修复**

这个区别要在 API 层面体现:查模糊指纹的方法**不返回** `Fix` 字段。让"用模糊匹配当答案"写不出来。

老化:命中时比对当前版本组合,对不上标记 `stale`,降级为线索。

退休:依赖升级(`tally deps` 采集)碰过验证组合里的包时,重跑该修复的复现命令
(上下文取自事实面最近一次 `failure_event` 的 workdir/repro),连跑 5 次(G1 同标准),
有绿 = 不再复现 → 写 `RETIRED.md` + git commit(记录保留不删),同时清索引面
(`fingerprint_stat` 回 unresolved,同指纹再失败重新走诊断)。退休 ≠ stale:
stale 是版本组合对不上、降级成线索;退休是验证过不再复现、修复退出使用。
已退休条目 `LookupExact` 命中按 stale 处理,不重验不删。判定全在确定性代码,零 LLM。

**双层分离(v3):** 本包只管**已验证修复库**(只读、gated、守铁律 2,**界面无编辑入口**)。可编辑的 Wiki 笔记层另存 `wiki/` git,**不经本包**,由 `web` 直接读写。两层在界面上醒目标注区分(见 `docs/06-ui-spec.md`)。

**这个包不导入 `llm`,由架构测试强制。**

### `requirement`

- 需求文件解析(EARS 格式)
- **linter 五闸**:条款化、验收条件+检查机制、零未决、非目标非空、术语唯一
- 问询循环:linter 报缺口 → LLM 把缺口转成问题(带候选答案)→ 人回答 → 重跑
- 签署与冻结:内容哈希 + git signed commit

**LLM 只做"缺口 → 问题"的转换,不产出需求内容。**

两类缺口分开处理:

- 结构缺口(linter 报的)→ 硬闸,不清零不许签
- 实质缺口(LLM 提的)→ 只建议,可一键驳回,**不阻塞签署**;驳回要留痕

### `project`

项目、任务、追溯图查询。

追溯链:`条款哈希 → commit → 制品 → 部署 → 失败指纹`,全部从机器记录推导。

两个核心查询:

- **视图 A** 某条款的实现累计失败(需要完整链)
- **视图 B** 某指纹的影响面(条款维度可选,**没有 requirement 也能工作**)

### `llm`

客户端封装。

```
Proposer interface {
    Propose(ctx, prompt, schema) (json.RawMessage, error)
}
```

**只有 Propose,没有 Judge、Verify、Score、Evaluate 之类的方法。** 接口形状本身就是约束。

调用方只有三个:`requirement`、`failure`、`harvest`。

### `jobs`

Postgres 表 + 进程内 worker pool。

任务类型固定枚举,不支持运行时注册新类型 —— 图是设计期定死的。

### `exec`

worker 侧的动作执行端:`jobs` 队列里的环境三件(拉起/推倒/重建)、部署回滚与诊断在这里落到真正的执行。

- **环境 / 部署动作**:按 payload 里的定义路径(内容仓库相对)读最新定义,调 `environment` / `deploy` 执行;
  事实面(`environment_run` / `deploy_run`)由执行端在动作执行时 INSERT(INV-5 的正路:界面只入队)。
- **失败录入编排**:动作失败时业务包返回带 runID + logpipe 指纹的结构化错误
  (`environment.UpError` / `deploy.FailureError`),由本层调 `ingest` 录进 `failure_event`。
  **业务包不导入 ingest**,接入点收敛在本编排层。`TALLY_DIAGNOSE_AUTO=1` 时(T11e,默认关),
  录入返回的精确指纹顺带入队一次 `diagnose`,与界面手动重诊断同一条路径。
- **诊断**:按精确指纹装载最近一条 `failure_event`,重组 `failure.Event` 交 `failure.Handle` 跑一轮;
  LLM 提案器、复现执行器、知识库由 worker 注入。缺依赖时清楚报错,不空跑。

### `store`

`sqlc` 生成的查询 + 薄仓储层。

**不放业务逻辑。** 事实面表只暴露 Insert 和 Select,不暴露 Update/Delete。

### `web`

HTTP 层:路由、模板、登录/CSRF 中间件、全部界面 handler(v3 起含写操作)。

**两种写(见 CLAUDE.md「界面的写边界」):**

- **写用户内容** —— 增改项目登记、需求条款、任务定义、bug 备注、文档、Wiki 笔记。handler 读写 **git**(markdown/yaml)或写**索引面**表。**不碰事实面、不碰状态字段。**
- **触发动作** —— 拉起/推倒/重建环境、部署/回滚、触发诊断。handler 只**入队 `jobs`**,真正执行在 worker,产生新的事实记录。

**硬约束:**

- 非 GET/HEAD 的路由一律挂在登录中间件之后(INV-9),并带 CSRF 双提交 token(手写,不引依赖)。
- 公开路由只有 `/login`、`/logout`、健康检查、静态资源。
- 没有任何 handler 写 `.status` 字段,没有 `PUT/PATCH .../status` 路由(INV-3)。
- `web` 通过调用业务包(requirement/failure/harvest)编排,**不直接导入 `llm`**。

---

## LLM 的三个合法位置

| 位置 | 提什么 | 谁判定 |
|---|---|---|
| `requirement` 问询 | 把 linter 缺口转成问题 | 人(回答/驳回) |
| `failure` 诊断 | 修复方案 | `failure` 的四道闸,二值 |
| `harvest` 抽取 | 服务定义 + 健康检查 | 实际拉起是否通过健康检查,二值 |

三处都是:**LLM 提,机器判。** 没有第四处。
