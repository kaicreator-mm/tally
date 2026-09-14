# T10b 部署/环境失败经 exec 编排接入统一录入

## 前置
T10a(`internal/ingest` 统一录入编排已就绪)。

## 目标
让部署与环境失败也进 `failure_event`,贯通生产链。按决定 D2:**业务包不导入 ingest** ——
`environment` / `deploy` 只把失败错误结构化(带 runID + logpipe 指纹产出),录入由编排层
(`exec` 执行端 + 运维 CLI)用 `errors.As` 取出后统一调 `ingest.RecordFailure` 完成。

## 交付物

```
internal/environment/lifecycle.go   fail() 改返回 *UpError(RunID/Combo/Processed/RawBytes/Cause,
                                    Error() 文本不变);记录失败行写库出错时仍返回普通 error(无 runID 可带)
internal/deploy/deploy.go           失败改返回 *FailureError(同上 + Status failed/rollback_failed)
internal/exec/
  exec.go                           Config 加 DB *store.Store;ingestEnvFailure/ingestDeployFailure
                                    两个尽力而为的录入帮手;接进 up/rebuild/rollback 失败路径
  doc.go                            依赖方向说明更新(exec → ingest/failure;业务包不导入 ingest)
cmd/worker/main.go                  exec.Config 接上 DB: st(生产失败真正入 failure_event)
cmd/tally/env.go, deploy.go         CLI 手动触发的失败同样录入(与 exec 同一套)
internal/environment/lifecycle_test.go, internal/deploy/deploy_test.go
                                    单测补断言:结构化错误带 runID、指纹与 health_check_result 一致、可 Unwrap
internal/exec/exec_integration_test.go
                                    集成:拉起失败 → failure_event(source=environment)+ 指纹与
                                    environment_run 一致;成功不录;回滚失败 → source=deploy 同理
```

## 验收标准

- [x] `*environment.UpError` / `*deploy.FailureError`:`Error()` 文本与结构化前逐字一致
      (既有四处 `Contains("指纹")` 断言不改而过);`Unwrap()` 回原始 cause
- [x] 结构化错误携带的指纹与同一次失败写进 `health_check_result` 的 logpipe 指纹**同源一致**
- [x] `exec` 编排层:up/rebuild/rollback 失败时经 `ingest.RecordFailure` 写 `failure_event`
      (`source` 分别为 environment/deploy,来源外键指回失败行);`Config.DB == nil`(单测)时静默跳过
- [x] 录入出错不遮盖原始失败(只记日志/提示);非结构化错误(推倒失败、定义校验不过)不录入
- [x] 集成测试(真 Postgres):拉起失败三面关联正确且指纹一致;成功拉起 `failure_event` 零行;
      回滚失败录 source=deploy
- [x] `go test ./...` 与 `go test -tags=integration ./internal/exec/...` 全绿;
      `test/arch` 全过(不改任何 arch 测试);不新增 `go.mod` 依赖;零迁移

## 不要做
- **不让 environment / deploy 导入 ingest**(方向是 exec → ingest → 业务包;由本任务的结构保证)
- 不在录入时自动入队诊断(D1:诊断只手动触发,入队在 T10c 之后另立)
- 不改 `failure_event` 表结构、不给事实面写 UPDATE/DELETE(INV-5)
- 不改 `failure.RecordEvent` 语义;不改既有错误断言测试来迁就实现

## 相关文档
`docs/01-architecture.md`(`internal/ingest` 一节:部署/环境失败的接入方式)·
`docs/00-PRD.md` 第二部分规则 ⑤(失败来源标记)· `tasks/T10a-ingest-build.md`
