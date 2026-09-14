# 范围外发现的问题

## 集成测试共享库并行污染(测试基建债)—— 已由 T09 根治

`go test -tags=integration ./...` 各包测试二进制**并行**跑,共享同一个持久
Postgres(`DATABASE_URL`)。凡读**全表索引面**(`commit_clause` / `trace_edge`)
再按某键归并断言的测试,会被别包/上次运行追加的行污染,表现为「集成连跑两遍」
时全局计数增长而误报。

**根治(已完成,见 [T09](T09-test-isolation.md)):** 每个测试经 `store.TestDB(t)`
(`internal/store/testdb.go`)拿到独立 schema —— 建 `tally_it_<纳秒>` schema、把全部迁移
(含 INV-5 触发器)跑进去、`search_path` 钉到它、`t.Cleanup` 里 `DROP SCHEMA CASCADE`。
全部 `*_integration_test.go` 已改用它,不再共享默认 schema 的表数据。全局/全表断言
(数 `deploy_run`/`job` 全表行数、`VerifyIndex` 全库 diff)随之恢复严格相等;
`internal/web/handlers/execution_integration_test.go` 的动作边界断言也从
「收窄到 `WHERE project_id = $1`」恢复为断全表。连续 5 次 `go test -tags=integration -count=1 ./...` 全绿。

下面保留两处历史记录,说明当初为什么做过临时收窄、以及结构性限制的成因(现均已随隔离消失)。

<details>
<summary>历史:T09 之前的临时收窄与结构性限制(已随 schema 隔离失效)</summary>

曾用「测试数据键带每次运行唯一 run 号」临时隔离(不削弱不变量,严格断言收窄到自己的数据):

- `internal/project/trailer_integration_test.go`(commit sha 带 run)
- `internal/project/trace_integration_test.go`(全局边数改断言单调不降 + 自己的边严格相等)
- `internal/web/handlers/execution_integration_test.go`(动作边界测试用每次运行唯一的
  `forge-env-*` / `forge-rb-*` 项目名,事实面与 job 断言收窄到 `WHERE project_id = $1`)

**T05c 进展(部分缓解,未根治):** T05c 的视图 A/覆盖率测试往 `commit_clause`/`test_clause`
写**无法收窄**的索引行(内容寻址、无 project_id 列),落进 `internal/project` 的
`VerifyIndex` 全局一致性 diff,致 `TestReindexWithGitRepoThenVerify` /
`TestVerifyIndexDetectsHandEdit` 在并行全跑时**时序性**失败(单跑必过)。已做三处加固,
把「测试工具自身制造污染」的几类消除:

- `VerifyIndex` 改 `RepeatableRead`(原 read committed 每语句重读最新已提交行,before 快照/
  重算删除/after 快照截到不一致横截面,产生幻影 diff)。
- commit/test 挂钩快照收窄到本项目(新增 `ListCommitClausesForProject` /
  `ListTestClausesForProject`,经 content_hash 归属),别项目挂钩不进本项目 diff。
- `RebuildTraceEdges` 改幂等(补缺失边 + 删陈旧边,替代 ClearTraceEdges 全清重写):
  稳态下不写一行,只读检查不再反复重铺全表。

**曾存的结构性限制(根治的一部分):** `VerifyIndex` 的 trace_edge 臂是**全局**重建再 diff
(`BuildEdges` 读 live `commit_clause`)。已提交的 `Reindex` 删掉某挂钩时**不级联删**它已写入的
`clause→commit` 边,要等下一次 reindex 才清;并发跑时 verify 读到「挂钩已删、边还在」的中间态,
合法地算出一周期 diff。这是「对可变且无法收窄的挂钩做全局一致性断言」在共享库并行下的固有限制,
非代码缺陷。**已随 T09 根治:** 每测试独立 schema/DB,让 verify 的「全局」收敛为「本测试 schema
的全局」,跨包并发写入天然不可见,幻影 diff 与上述中间态一并消失。

</details>

## raw_log_ref 保留期与语句级 append-only 触发器冲突

PRD 提到 `failure_event.raw_log_ref` 90 天保留期「过期后清空」,这需要对事实面
UPDATE,与 0011 加强后的语句级 append-only 触发器(任何 UPDATE/DELETE 语句都报错)
冲突。未来需设计:触发器豁免特定清理语句,或原始日志存外部对象存储、事实面只记引用。
不属于 T06,留待后续。

## 环境「已停止」状态无从推导(T05b 遗留)—— 已修复

`docs/06-ui-spec.md` ② 列了环境状态「在跑 / 停了 / 上次拉起失败」,但 T01 的
`environment.Manager.Down` 明确**不写事实面**(「environment_run 只记拉起」),
所以推倒之后最近一行仍是 `up`,界面只能展示「上次拉起成功」,展示不出「停了」。
`environment_run.status` 的 CHECK 约束预留了 `torn_down` 取值,UI 侧(T05b)也已
按 `torn_down → 已停止` 渲染,但没有写入方。

**修复(2026-07-27):** 采用「补记一行 `status='torn_down'`」方案。
`Manager.Down` 在 docker/ssh 推倒成功后调 `recordTornDown`,写一行
`status='torn_down'`,`version_combo` 写 `{"kind":"docker-compose"}` 占位
(推倒时不采集镜像版本),`health_check_result` 为 null。`Rebuild` 因此写两行
(torn_down + up),最后一行是 up,状态推导正确。相关注释(exec.go、actions.go、
environments.go)同步更新。`environment` 包新增 `StatusUp/StatusFailed/StatusTornDown`
常量,消除裸字符串。

## T10 后续:生产链闭环之后的缺口 —— 已全部完成

T10a/b/c 把生产链接通:三类失败(构建/部署/环境)录入 `failure_event` + 指纹索引,
worker 注册诊断执行体(`KindDiagnose`)。以下缺口在 T10 计划里**明确排除**,各自立项,
现均由 T11 系列完成:

- **运输层** —— 已完成([T11d](T11d-webhook.md)):`server` 提供 `POST /webhook/build`
  (token 鉴权),CI 构建结果 POST 进来即完成录入;语义与 `tally ingest-build` 一致。
- **依赖升级采集** —— 已完成([T11a](T11a-deps-collect.md)):逐提交 diff 锁文件写
  `dependency_change`;`tally deps` 子命令。
- **修复退休机制** —— 已完成([T11c](T11c-fix-retire.md)):`tally deps` 采集完顺带
  重验本次变更碰过的已验证修复,不再复现的退休(`RETIRED.md` + 清索引面)。
- **复现上下文的权威来源(D3 的正式答案)** —— 已完成([T11b](T11b-repro-context.md)):
  `failure_event` 增 `workdir`/`repro`/`other_targets`,录入方携带、诊断时装载跑
  G1/G2/G3;worker 的 failureRunner 按完整命令执行 Target。
- **诊断自动触发(可选)** —— 已完成([T11e](T11e-auto-diagnose.md),默认关):
  `TALLY_DIAGNOSE_AUTO=1` 时失败录入(webhook / `ingest-build` / 执行端部署环境失败)后
  按指纹自动入队一次诊断,与手动重诊断同一条 worker 路径;默认行为不变(D1)。
