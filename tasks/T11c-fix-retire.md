# T11c 修复退休机制

## 前置

T11a(依赖变更事实从哪来)、T11b(复现命令与工作目录从哪来)。

## 目标

落实 PRD 第五部分「修复会过期」:依赖升级的时候,重跑相关已验证修复的重现命令,
不再复现的标记为**退休**(记录保留,不删)。退休不是删除,也不是标 stale ——
stale 是「版本组合对不上,降级成线索」,退休是「验证过不再复现,修复退出使用」。

## 已锁定的设计决策

- **D-c1 触发时机:`tally deps` 采集完依赖变更顺带跑退休检查。**
  PRD 的触发条件就是「依赖升级的时候」;采集是唯一写入 `dependency_change` 的入口,顺带触发最贴合。
  不为退休单开守护进程(不做常驻)。
- **D-c2 相关条目的判定:修复验证组合(`verified_combo`)里含本次变更的包(任意版本)。**
  版本比对交给运行时:只要升级碰过修复依赖过的组件,就值得花五次构建去重验。
- **D-c3 退休判据 = 复现不再稳定**(G1 语义复用:`failure.ConfirmRepro` 返回 false)。
  连跑 5 次复现命令,有绿 → 不再复现 → 退休。全红 → 仍复现,不动。
  判定在确定性代码(铁律 1),零 LLM。
- **D-c4 退休的执行上下文取自事实面**:该项目 + 该指纹最近一次 `failure_event` 的
  `workdir`/`repro`(T11b 的列)。没有(旧失败没带上下文)→ 跳过并说明,不猜。
- **D-c5 退休标记 = 知识条目目录写 `RETIRED.md` + git commit**(与 STALE.md 同构,记录保留不删)。
  索引面同步:`fingerprint_stat` 清 fix 相关列(status 回 unresolved、fix_ref/验证边界清空、
  诊断次数归零)—— 修复失效后同指纹再失败应当重新走诊断,而不是撞「终态 fixed」短路。
  这是系统按确定性规则改写索引面(归类/入库同源先例),不是手动改状态(INV-3 不违反)。
- **D-c6 退休后的待遇:`LookupExact` 命中已退休条目 → 按 stale 处理(降级线索)**;
  `ListVerified` 展示退休标记。已退休条目不删不重验(除非再次升级再碰)。

## 交付物

```
internal/knowledge/store.go   Retire(fp, reason) + hasRetiredMarker;LookupExact 退休→Stale;Entry 加 Retired
internal/store/queries/fingerprint_stat.sql + ClearFingerprintStatFix(:exec)
internal/store/(sqlc 重新生成)
internal/deps/retire.go       RetirePass(ctx, ...):变更包 × 已验证修复 → 重跑复现 → 退休
internal/deps/retire_test.go  fake Runner/KB 的表驱动:仍复现不退休 / 不再复现退休 / 缺上下文跳过
internal/deps/retire_integration_test.go  真 Postgres + 真知识仓库 + fake 执行器走全链
cmd/tally/deps.go             采集后调 RetirePass(-no-retire 可关)
docs/01-architecture.md       knowledge 段补退休说明
```

## 验收标准(机器可查)

- [ ] 单测:复现仍稳定(5 红)→ 不退休;不再复现(有绿)→ RETIRED.md 落盘 + git commit + 索引面被清
- [ ] 集成:退休后 `LookupExact` 该条目返回 Stale;`fingerprint_stat.status` 回 unresolved、fix_ref 空
- [ ] 缺 workdir/repro 的条目被跳过(有日志说明),不误退
- [ ] `make check` 等价全套全绿

## 不要做

- 不删知识条目、不重写 verified.json(记录保留,只加标记)
- 不在界面上提供「手动退休」按钮(状态类写入只能来自执行链;界面只读验证库,INV-2)
- 不自动重新诊断被退休的指纹(下次失败录入/手动重诊断自然触发)

## 相关文档

`docs/00-PRD.md` 第五部分「修复会过期」;T11a/T11b 任务文件;
`internal/knowledge/store.go`(MarkStale 同构先例)。
