# T11b 复现上下文的权威来源(D3 正式答案)

## 前置

无(M6 已完成)。与 T11a 无依赖,可并行;编号先后而已。

## 目标

给「诊断要能真正复现」补上缺的复现上下文:D3 当时零迁移跳过了 G1/G2,
本任务给出正式答案 —— **复现上下文在失败录入时由知道怎么重现的一方(CI/人)随失败一起持久化**
进 `failure_event`,诊断时装载出来跑真的 G1/G2/G3。

依据(PRD 第四部分):只做构建/部署两格失败,正因为「重现一次几乎不要钱」——
构建重现 = checkout 那个提交再 build 一次。重现命令与代码目录是失败自带的属性,
录入时不记,事后无处还原。

## 已锁定的设计决策

- **D-b1 权威来源 = 录入时携带。** `failure_event` 加三列(`workdir` / `repro` / `other_targets`),
  谁录入谁知道:构建失败由 CI/操作者给(workdir=执行机上 checkout 路径,repro=重跑那次构建的命令);
  部署/环境失败本来就没有可复现命令(不复现,靠重新部署),继续留空 → `Handle` 照旧跳过 G1/G2(D3 语义保留)。
- **D-b2 允许一次迁移**(D3 的「零迁移」只是当时那轮的决定)。三列 `NOT NULL DEFAULT`(空串/空数组),
  存量行自动补默认值 —— 旧失败仍走「跳过 G1/G2」路径,行为不回归。
- **D-b3 事实面只加列,不加状态。** 三列是失败的属性(事实),随 INSERT 落库,不提供 UPDATE(INV-5)。
- **D-b4 `OtherTargets` 条目 = 其余构建目标的完整构建命令**(G3 检查「修复不影响别的目标」)。
  `gates.go` 跑 G3 时 `Command` 保持复现命令、`Target` 带目标;worker 执行器约定
  **`Target` 非空时把 `Target` 当完整命令跑**(不对 Command 做拆改手术 —— 脆弱且猜)。
  谁给复现上下文谁知道别的目标怎么建,与 D-b1 同源。接线后 `failureRunner` 不再报「未接线」。
- **D-b5 `tally ingest-build` 增 `-workdir / -repro / -targets` 三个 flag**,把上下文带进来。

## 交付物

```
migrations/0013_failure_repro.sql  failure_event 加 workdir/repro/other_targets 三列(NOT NULL DEFAULT)
internal/store/queries/failure_event.sql  Insert / GetLatest 两查询带上三列
internal/store/(sqlc 重新生成)
internal/failure/budget.go      RecordEvent 写三列(Event 已有字段,落库接通)
internal/exec/diagnose.go       loadDiagnoseEvent 装载三列(不再留空),删 D3 注释
cmd/worker/main.go              failureRunner:Target 非空 → 在该构建目标上跑(G3 接线),删「未接线」报错
internal/ingest/build.go + cmd/tally/ingest.go  ingest-build 加 -workdir/-repro/-targets
internal/failure/(相关集成测试)  带 repro 的失败 → Handle 真跑 G1(fake Runner 计数)
```

## 验收标准(机器可查)

- [ ] 迁移 up/down 可逆;存量行三列取默认值(空),旧诊断行为不变(跳过 G1/G2)
- [ ] 集成测试:带 repro 录入的失败,诊断时装载出非空 `Repro`,`Handle` 走 G1(fake Runner 断言被调)
- [ ] worker `failureRunner` 对 `Target` 非空的 RunSpec 不再报错(G3 可跑)
- [ ] `tally ingest-build -repro ... -workdir ...` 落库可查
- [ ] `make check` 等价全套全绿

## 不要做

- 不做「自动推断复现命令」(从 CI 配置反推等)—— 权威来源是录入方显式给
- 不碰部署/环境失败的复现语义(它们留空是设计,不是缺口)
- 不改四道闸判定逻辑(gates.go 不动)

## 相关文档

`docs/00-PRD.md` 第四部分(重现便宜);`internal/exec/diagnose.go` 的 D3 注释;
`tasks/BACKLOG.md`「复现上下文的权威来源」。
