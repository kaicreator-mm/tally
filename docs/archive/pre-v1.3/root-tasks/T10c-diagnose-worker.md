# T10c 诊断执行体(worker 注册 KindDiagnose)

## 前置
T10b(失败已经统一录入 `failure_event`,生产链前半段贯通)。

## 目标
把队列里已有的 `jobs.KindDiagnose` 在执行端真正接上:`exec` 新增诊断处理器,按精确指纹
装载最近一条 `failure_event` 重组 `failure.Event`,交给 `failure.Handle` 跑一轮
「查库 → 预算 → 复现 → 提案 → 四道闸 → 归类」;worker 注入真实依赖
(LLM 提案器、复现执行器、知识库)。

按决定 D3:**零迁移、零新表**。`failure_event` 没有复现上下文列,装载时
`Workdir` / `Repro` / `OtherTargets` 留空 —— `Handle` 缺复现上下文时**跳过 G1/G2**
(不提候选、不验证、零 LLM),预算闸门与知识库精确命中短路不受影响。
复现上下文的权威来源另立项;本任务先把链路接通。

按决定 D1:诊断只手动触发(本任务不加入队入口,界面/CLI 触发在后续任务)。

## 交付物

```
internal/failure/
  ports.go                          Conclusion 加 SkippedRepro(缺复现上下文跳过 G1/G2)
  handle.go                         预算计次之后、G1 之前:ev.Repro == "" → 返回 SkippedRepro,
                                    零 LLM、执行器不跑(次数照计,预算仍三轮归档)
  handle_test.go                    补两个单测:无复现上下文跳过且零调用;精确命中仍短路 fixed
internal/store/
  queries/failure_event.sql         GetLatestFailureEventByFingerprint(:one,按 occurred_at 取最近)
  failure_event.sql.go, querier.go  sqlc generate 产出(不改手写 SQL 之外的东西)
internal/exec/
  diagnose.go                       DiagnoseDeps(Proposer/Runner/KB/SpecSink)+ handleDiagnose:
                                    校验 → 装载事件 → failure.Handle → 记结论日志
  exec.go                           Config 加 Diagnose;Handlers 注册 KindDiagnose(共 5 种)
  exec_test.go                      表覆盖断言改 5 种;新增依赖未接线/缺指纹的守卫测试
  diagnose_integration_test.go      集成(真 Postgres + 真 git 知识库):
                                    KB 精确命中 → 成功、零 LLM、索引面 fixed 原样保留;
                                    缺复现上下文三轮 → attempts==3、第四轮归档短路、全程零 LLM;
                                    未知指纹 → 报「无 failure_event」
cmd/worker/main.go                  注入 DiagnoseDeps:llm HTTP 客户端(ConfigFromEnv)、
                                    failureRunner(LocalRunner 适配:照写 Fix 文件后跑命令,
                                    输出经 logpipe.RedactCoarse;G3 目标未接线,遇 Target 报错)、
                                    knowledge.NewStore(与 server 同一内容仓库)
```

## 验收标准

- [x] `exec.Handlers` 注册 `jobs.KindDiagnose`;缺 `fingerprint` 或依赖未接线
      (DB/KB/Proposer/Runner 任一缺)→ 清楚报错,不空跑不 panic
- [x] 装载按精确指纹取**最近一条** `failure_event`;无对应事件 → 报「先录入再诊断」,
      不让诊断凭空跑
- [x] D3 落点:`Handle` 在预算计次之后检查 `ev.Repro`,为空则 `SkippedRepro` 返回 ——
      不提候选、不验证、零 LLM;次数照计(三轮后归档);精确命中短路在入口,不受影响
- [x] worker 注入真实依赖;LLM 只提案(INV-1),判定全在 `failure` 四道闸;
      执行器输出经 `logpipe.RedactCoarse` 再回传(失败执行器的责任)
- [x] 集成测试(真 Postgres、真 git 知识库、`failure.Verify` 真跑四道闸产出证据入库):
      KB 命中零 LLM;三轮预算 + 归档;未知指纹报错
- [x] `go test ./...` 与 `go test -tags=integration ./...` 全绿;
      `test/arch` 全过(不改任何 arch 测试);不新增 `go.mod` 依赖;零迁移

## 不要做
- **不加复现上下文列/表**(D3 零迁移;另立项解决权威来源)
- 不在录入失败时自动入队诊断(D1:手动触发,入队入口后续任务)
- 不让执行器对 `spec.Target` 做任意解释(G3 目标执行未接线,遇之报错)
- 不给事实面写 UPDATE/DELETE(INV-5);诊断只读改索引面(`fingerprint_stat`)
- 不改 `failure.Handle` 的判定语义,只加缺复现上下文的早退分支

## 相关文档
`docs/01-architecture.md`(`failure` / `exec` 两节:诊断链路与执行端)·
`docs/00-PRD.md` 第四部分(失败诊断流程)· `tasks/T10b-failure-ingest.md`
