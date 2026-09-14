# T10a 统一失败录入 + 构建失败录入(接上生产链断点)

## 前置
T09(业务任务与测试基建均已完成;本任务在其上补生产链的录入层)。

## 目标
新增 `internal/ingest` 编排包:把「失败已算出指纹,却没人写 `failure_event`」的断点补上。
先交付构建入口 `IngestBuild`(先写 `build_run`,失败再接着录 `failure_event`)+ 运维 CLI
`tally ingest-build`,让 PRD 生产链「失败 → logpipe → 指纹 → 查库 → 提案 → 四道闸 → 入库」
从构建侧贯通。部署/环境侧的接入在 T10b(走 exec 编排,业务包不导入本包)。

## 交付物

```
internal/ingest/
  doc.go                 包文档:统一失败录入编排;允许导入 core/store/logpipe(含 toolchain)/failure
  ingest.go              Input、RecordFailure(logpipe 选抽取器或透传 Processed → failure.RecordEvent)
  build.go               BuildInput、IngestBuild(先 INSERT build_run,failed 才录 failure_event)
  ingest_test.go         单测:校验类错误不碰库(nil store 安全)、抽取器选择与确定性
  ingest_integration_test.go   真 Postgres:三面同事务落盘、指纹与直接跑 logpipe 一致、成功不落事件
cmd/tally/
  ingest.go              runIngestBuild:flag 解析 + 读日志(文件或 stdin)→ ingest.IngestBuild
  main.go                注册子命令 ingest-build + usage 行
docs/07-operations.md    cmd/tally 子命令表加一行 ingest-build
tasks/README.md          登记 M6(T10a/b/c)与 T10a 行
```

## 验收标准

- [x] `ingest.RecordFailure(ctx, s, in)`:选工具链抽取器跑 logpipe(空/未知退回通用;
      `in.Processed` 非 nil 时透传不重复处理),组 `failure.Event`(`PipelineVersion = logpipe.RuleVersion`)
      交 `failure.RecordEvent` 同事务写 `failure_event` + 累加 `fingerprint_stat`
- [x] `ingest.IngestBuild(ctx, s, in)`:先 INSERT `build_run`(只追加,INV-5)拿到 runID;
      仅 `status=failed` 时接着录 `failure_event`(`source=build`,`build_run_id=runID`);
      成功构建只落 `build_run`,返回 `eventID=0`
- [x] 校验类错误(缺 project_id/commit_sha/started_at、status 非法、未知来源、来源外键非恰好一个)
      全在碰库前返回——单测传 nil store 即可断言
- [x] 集成测试(真 Postgres,每测试独立 schema):失败录入三面齐落且 `fp_exact` 与直接跑
      go 抽取器一致、`raw_log_bytes = len(raw)`、`fingerprint_stat.occurrence_count = 1`;
      成功录入 `failure_event` 零行;`Processed` 透传指纹不被重算
- [x] CLI `tally ingest-build -project P -commit SHA [-toolchain X] [-log <文件>] [-status failed|passed]
      [-exit N] [-ci-url U] [-artifact D] [-raw-ref R] [-combo k=v,k=v]`:缺省读 stdin,
      管道接 CI 即完成录入;失败/成功路径各输出对应事实记录 id
- [x] `docs/07-operations.md` 子命令表登记 `ingest-build`
- [x] `go test ./...`、`go test -tags=integration ./internal/ingest/...` 全绿;
      `test/arch` 全过(不为此改任何 arch 测试);不新增任何 `go.mod` 依赖;零迁移

## 不要做
- **不让业务包(environment/deploy)导入 ingest** —— 那是 T10b,走 exec 编排层
- 不在录入时自动入队诊断 job(D1:诊断只手动触发)
- 不持久化 Repro/Workdir/OtherTargets(表上无对应列;D3:诊断侧缺复现上下文时跳 G1/G2)
- 不给 `build_run`/`failure_event` 提供任何 UPDATE/DELETE(INV-5)
- 不新增迁移;不改 `failure.RecordEvent` 的语义(只调用)

## 相关文档
`docs/00-PRD.md` 第四部分(生产链)· `docs/01-architecture.md`(`internal/ingest` 一节)·
`docs/07-operations.md`(`tally ingest-build` 一节)· `docs/02-invariants.md`
