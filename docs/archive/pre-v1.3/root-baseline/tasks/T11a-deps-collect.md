# T11a 依赖升级采集(dependency_change 写入方)

## 前置

无(M6 已完成;`dependency_change` 表与查询在 T00 就位,消费方已存在)。

## 目标

给 `dependency_change` 补上写入方:从项目 git 仓库的锁文件 diff 解析依赖变更,
让「失败次数时间线与依赖升级对齐」(PRD 第三部分收集来源、第四部分 UI)和
「上游问题」归类信号(`failure.Handle` 的 `HasRecentDepChange`)有真实数据可吃。

## 已锁定的设计决策

- **D-a1 解析范围:只做 go.mod 与 package-lock.json。** 工具链清单里
  docker/compose/kamal 没有锁文件,无从 diff。两个解析器都是确定性纯函数。
- **D-a2 幂等:唯一索引 + ON CONFLICT DO NOTHING。** CLI 重跑是常态,事实面
  只 INSERT 不 DELETE,靠 `(project_id, commit_sha, package, from_version, to_version)`
  唯一索引(`NULLS NOT DISTINCT`,新增包 from_version 为 NULL 也要去重)保证重跑不产生重复行。
  `INSERT ... ON CONFLICT DO NOTHING` 仍是 INSERT,不触发语句级 UPDATE/DELETE 拦截触发器(INV-5 无碍)。
- **D-a3 扫描方式:逐提交与父提交比锁文件 blob。** go-git 读两个版本的锁文件内容,
  解析成 `map[包名]版本` 后 diff:新增(from 空)/移除(to 空)/版本变化。
  根提交(无父)跳过 —— 全量「新增」是噪声不是升级信号。
- **D-a4 时刻:`changed_at` 取提交的 committer 时间**(升级发生时刻),不取采集时刻。
- **D-a5 go.mod 的 indirect 依赖一并记**:它们同样是升级,对齐信号本来就是粗粒度的。
- **D-a6 package-lock.json 包名取 `node_modules/` 分隔后的最后一段**(嵌套依赖同名时以深处为准)。

## 交付物

```
internal/deps/
  doc.go        包职责:锁文件 diff → dependency_change 采集;允许导入 store(不导入 llm)
  parse.go      ParseGoMod / ParsePackageLock([]byte → map[string]string),Diff(before, after)
  parse_test.go 表驱动:require 块/单行/indirect、lock 的嵌套包、增删改三类 diff
  collect.go    Collect(ctx, store, Options{ProjectID, RepoPath, From, To}) — go-git 遍历提交、读 blob、diff、入库
migrations/0012_dependency_change_unique.sql   唯一索引(NULLS NOT DISTINCT)
internal/store/queries/dependency_change.sql   + InsertDependencyChangeIfAbsent(:execrows, ON CONFLICT DO NOTHING)
internal/store/(sqlc 重新生成)
cmd/tally/deps.go    runDeps:-project P -repo PATH [-from REV] [-to REV]
cmd/tally/main.go    注册子命令 deps + usage
docs/07-operations.md 子命令表加一行
migrations/README.md  登记 0012
tasks/T11a-deps-collect.md  本文件
```

## 验收标准(机器可查)

- [ ] `ParseGoMod`/`ParsePackageLock`/`Diff` 单测全绿(增/删/改/空文件/嵌套包/indirect)
- [ ] 集成测试(真 Postgres,`store.TestDB`):临时 git 仓库两次提交改 go.mod,
      `Collect` 后 `dependency_change` 恰有对应行(package/from/to/commit_sha/changed_at=提交时刻)
- [ ] 重跑同一区间 `Collect` → 行数不变(唯一索引去重)
- [ ] `make check` 等价全套(gofmt/vet/lint/arch/unit)全绿

## 不要做

- 不做 UI(失败页时间线已在消费 `ListDependencyChangesRecent`)
- 不做常驻监听 / 文件 watcher(被动触发,PRD:不做常驻管道)
- 不解析 go.sum / yarn.lock 等其余锁文件(本任务只两个;其余记 BACKLOG)
- 不碰 `failure` 的归类逻辑(消费侧已就位)

## 相关文档

`docs/00-PRD.md` 第三部分「从哪收」、第四部分「界面上要有什么」(时间线对齐);
`docs/03-data-model.md`(dependency_change)。
