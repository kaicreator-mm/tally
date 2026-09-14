# 迁移

用 `goose`,文件名 `NNNN_名字.sql`,由任务逐个交付。

| 文件 | 任务 | 内容 |
|---|---|---|
| `0001_facts.sql` | T00 | 事实面表 + 阻止 UPDATE/DELETE 的触发器 |
| `0002_index.sql` | T00 | 索引面表 |
| `0003_env.sql` | T01 | `environment_run` |
| `0004_deploy.sql` | T03 | `deploy_run` |
| `0005_failure.sql` | T04 | `failure_event` |
| `0006_fp.sql` | T06 | `fingerprint_stat`(**未建:T00 已并入 `0002_index.sql`,重建会撞表**) |
| `0007_clause.sql` | T07 | `clause` / `clause_version` |
| `0008_trace.sql` | T08 | `commit_clause` / `test_clause` / `trace_edge` |
| `0009_session.sql` | T00 | 登录 session |
| `0010_jobs.sql` | T00 | 任务队列 |
| `0011_append_only_statement.sql` | T06 | 事实面触发器改语句级(0 行 UPDATE 也拦,INV-5 双保险) |
| `0012_dependency_change_unique.sql` | T11a | `dependency_change` 唯一索引(NULLS NOT DISTINCT),采集幂等去重 |
| `0013_failure_repro.sql` | T11b | `failure_event` 加复现上下文三列(workdir/repro/other_targets,NOT NULL DEFAULT) |

字段设计与理由见 `docs/03-data-model.md`。

## 三条规矩

1. **事实面表要加触发器阻止 UPDATE / DELETE。** 应用层的约束会被绕过,数据库层的不会
2. **引用条款、版本组合、制品的外键,列名以 `_hash` 或 `_digest` 结尾**(INV-4,架构测试会查)
3. **不要出现字段黑名单里的列名**(INV-7):`estimated_hours` `actual_hours` `assignee` `priority` `story_points` `reviewer` `approver` `due_date`

## 索引

先别建。等有真实数据量了拿 `EXPLAIN` 决定。现在只要三个:

```
failure_event(fp_exact)
failure_event(project_id, occurred_at DESC)
clause_version(content_hash)     -- 唯一约束自带
```
