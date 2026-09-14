# T09 集成测试隔离(每测试独立 schema)

## 前置
T05c(全部业务任务完成;这是测试基建,不改业务语义)。

## 目标
把集成测试从「共享一个持久 Postgres、靠唯一项目名/收窄断言躲污染」改为**每个测试独立 schema**,
落实 `docs/05-test-standards.md` 早就写明的规范(「每个测试自己建 schema,自己清理;用独立 schema
而不是独立数据库,快」),让全表/全局索引面断言在并行全跑下恢复**确定性**严格相等。

## 背景(为什么单独立项)
`go test -tags=integration ./...` 各包测试二进制并行,共享同一持久库。凡读**全表/全局索引面**
再断言的测试会被别包并发写入污染:

- `internal/project` 的 `TestReindexWithGitRepoThenVerify` / `TestVerifyIndexDetectsHandEdit`
  对 `trace_edge`/`commit_clause`/`test_clause` 做**全局**一致性 diff(`VerifyIndex`/`BuildEdges`
  读全表),被 `internal/web/handlers` 的 T05c 测试写入的**无法收窄**挂钩行(内容寻址、无 project_id)
  污染,时序性失败(单跑必过)。
- 既有「唯一项目名 + 收窄断言」只是**规避**,治标;且对无法收窄的索引面无效。

T05c 已做三处生产侧加固(`VerifyIndex` 改 RepeatableRead、挂钩快照按项目收窄、`RebuildTraceEdges`
改幂等)消除了测试工具自身制造的几类污染;但 trace_edge 全局重建 diff 对可变挂钩的**结构性**时序
敏感只能靠隔离根治 —— 每测试一个 schema,verify 的「全局」即收敛为「本测试 schema 的全局」。

## 交付物

```
internal/store/
  testdb.go            测试基建:从 DATABASE_URL 派生独立 schema,跑迁移,返回连到该 schema 的 *Store
                       (t.Cleanup 里 DROP SCHEMA CASCADE;连接 search_path 指向该 schema)
  migrate.go           把「对某连接跑全部 goose 迁移」抽成可复用(供测试基建与 cmd 共用)
internal/*/  *_integration_test.go
                     各包集成测试改从 testdb 取独立 schema,删掉唯一项目名/收窄断言的规避,
                     恢复严格全表断言(其中 internal/project 的两个 Verify 测试回到全局严格 diff)
docs/05-test-standards.md   「每测试独立 schema」一节补上 testdb 的具体用法(规范→实现对齐)
```

> `testdb`/`migrate` 属于测试基建与连接管理,放 `internal/store`(连接层)而非新包;若迁移抽取
> 更适合放 `internal/store/migrate` 子包,按实现时判断,同样登记在本文件。

## 验收标准

- [ ] 提供一个 helper(如 `store.TestDB(t)`):返回连到**本测试独立 schema** 的 `*store.Store`,
      迁移已应用,`t.Cleanup` 自动 DROP;schema 名每次运行唯一(可含包名/随机后缀)
- [ ] 所有 `*_integration_test.go` 改用它,**没有任何测试再共享默认 schema 的表数据**
- [ ] `internal/project` 的 `TestReindexWithGitRepoThenVerify` / `TestVerifyIndexDetectsHandEdit`
      **恢复全局严格 diff**(删掉 T05c 加的收窄兜底),断言与机器记录完全一致
- [ ] 删掉为躲污染而加的运行唯一项目名 / `countProjectRows` 式收窄(不再必要),断言回到直观写法
- [ ] **连续 5 次** `go test -tags=integration -count=1 ./...` **全绿**(原来约每 4–5 次偶发 1 次失败)
- [ ] `make check` 全绿;`test/arch` 全部通过(不为此改任何 arch 测试)
- [ ] 迁移对任意 schema 幂等可重复;`TestDB` 并行安全(并发测试各自 schema 互不可见)
- [ ] 不为本任务新增任何 `go.mod` 依赖(用 pgx + goose,均已在白名单)

## 不要做
- **不改业务逻辑、不改表结构、不加 project_id 列** —— 这是测试隔离,不是数据模型变更
- 不动七条铁律与 `test/arch` 任何断言(只让它们从「时序性」变回「确定性」)
- 不引入 testcontainers 等新框架(沿用「连用户指定 DATABASE_URL + 建独立 schema」的现有模式)
- 不顺手重构无关测试;只为本任务触及的隔离与断言写法
- 不改 `cmd/tally migrate` 的对外行为(迁移抽取保持等价)

## 相关文档
`docs/05-test-standards.md` 集成测试一节 · `tasks/BACKLOG.md`(共享库污染条目)· `docs/02-invariants.md`
