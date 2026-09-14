# T11e 诊断自动触发(可选缺口,默认关)

## 前置

T11d(webhook 是自动触发的主入口);T10c 已让诊断可执行。

## 目标

新失败录入后**可选**自动入队一次诊断(`jobs.KindDiagnose`),把
「失败 → 指纹 → 诊断」从两段手动变成一段手动。**默认关**:D1 已定诊断手动触发,
本任务只提供显式开关(环境变量 `TALLY_DIAGNOSE_AUTO=1`),不改变默认行为。

## 已锁定的设计决策

- **D-e1 默认关、环境变量开。** `TALLY_DIAGNOSE_AUTO=1` 时,录入成功且产生失败事件的
  那些路径(webhook / ingest-build CLI / exec 部署环境失败)顺带 `Enqueue(KindDiagnose)`。
  与 D1 不冲突:D1 拒绝的是「默认自动」,不拒绝「显式打开」。
- **D-e2 预算在 worker 侧守,不在入队侧预检。** worker 的 `Handle` 对终态/超预算指纹
  零 LLM 短路,多入一个队的成本可忽略;入队侧再查一遍预算反而两处逻辑。
- **D-e3 录入指纹要能拿到**:`IngestBuild` / 失败录入函数的返回值需带精确指纹,
  供入队用(改签名,内部调用点同步)。
- **D-e4 自动诊断与手动重诊断同一条 worker 路径**(同一个 handler),不另开执行体。

## 交付物

```
internal/ingest/(签名调整)  录入返回精确指纹
internal/exec/ + cmd/worker/  exec 失败录入后按开关入队诊断
internal/web/webhook.go      webhook 路径同样按开关入队
cmd/tally/ingest.go          ingest-build 同样按开关入队(需要连队列)
jobs 入队的小 helper(避免三处复制 payload 组装)
集成/单测:开关关 → 不入队;开关开 → 失败录入后队里有 diagnose 任务、成功构建不入队
docs/07-operations.md  环境变量说明
```

## 验收标准(机器可查)

- [ ] 默认(未设变量)行为与之前完全一致:录入后队列为空
- [ ] `TALLY_DIAGNOSE_AUTO=1`:失败录入 → 队列出现对应指纹的 diagnose 任务;passed 构建 → 无
- [ ] worker 取出自动入队的诊断任务能正常跑(复用既有 handler)
- [ ] `make check` 等价全套全绿

## 不要做

- 不把默认改成自动(D1)
- 不在入队侧做预算预检(D-e2)
- 不做「仅某些项目/指纹自动」的过滤配置(自用规模用不上)

## 相关文档

`tasks/T10c-diagnose-worker.md`(诊断执行体);D1 决策(T10 计划 §5);
`tasks/BACKLOG.md`「诊断自动触发」。
