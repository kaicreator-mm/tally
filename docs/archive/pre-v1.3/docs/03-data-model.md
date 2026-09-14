# 数据模型

只写**表有哪些、字段为什么这么设计**。建表 SQL 在 `migrations/`,由任务逐个交付。

---

## 分面

| 面 | 表 | 能不能重建 | 允许的操作 |
|---|---|---|---|
| **事实面** | `build_run` `deploy_run` `environment_run` `failure_event` `dependency_change` | 不能 | 只 INSERT |
| **索引面** | `clause` `clause_version` `commit_clause` `test_clause` `fingerprint_stat` `trace_edge` | 能 | 随便 |
| **队列** | `job` | 不需要 | 全部 |
| **运维(v3)** | `session` | 可清空(清了只是重新登录) | 全部 |

**git 里的不进数据库**(需求原文、环境定义、已验证修复内容)。数据库里存的是它们的哈希和位置。

---

## 事实面

### `environment_run`

一次环境拉起。

| 字段 | 说明 |
|---|---|
| `id` | |
| `project_id` | |
| `env_name` | 哪个环境(dev / test / 临时验证环境) |
| `definition_hash` | **环境定义文件的内容哈希**。定义改了,新旧记录自然分开 |
| `started_at` `finished_at` | |
| `status` | `up` / `failed` / `torn_down` |
| `health_check_result` | 结构化结果,哪一项没过 |
| `version_combo` | **JSONB。当时的镜像 tag、包版本、运行时版本、OS** |

**`version_combo` 冗余存在这里,不靠 join 去别处取。** 事实面是历史,历史必须自包含 —— 以后清理旧数据时,老化检测不能跟着一起坏。

### `build_run`

| 字段 | 说明 |
|---|---|
| `id` `project_id` | |
| `commit_sha` | |
| `ci_run_url` | 回链到 CI |
| `started_at` `finished_at` `exit_code` `status` | |
| `toolchain` | 用哪个抽取器解析日志 |
| `artifact_digest` | 成功时产出的制品。**部署只认这个** |
| `version_combo` | JSONB,同上 |

### `deploy_run`

| 字段 | 说明 |
|---|---|
| `id` `project_id` | |
| `artifact_digest` | **必填。** 没有 digest 的部署不允许记录,也就不允许发生 |
| `build_run_id` | 可空(制品可能来自外部) |
| `target_env` | |
| `tool` | `kamal` / `ansible` / `compose` |
| `started_at` `finished_at` `status` `exit_code` | |
| `health_check_result` | |
| `version_combo` | JSONB |

### `failure_event`

三个来源(构建/部署/环境)共用一张表。

| 字段 | 说明 |
|---|---|
| `id` `project_id` | |
| `source` | `build` / `deploy` / `environment` |
| `build_run_id` / `deploy_run_id` / `environment_run_id` | 三选一非空 |
| `raw_log_ref` | 原始日志的存储位置(对象存储或本地路径)。**留着是为了规则改了能重跑** |
| `raw_log_bytes` | 原始大小,用于容量管理 |
| `excerpt` | 抽取后的关键片段(已脱敏) |
| `normalized` | 规范化后的文本 |
| `fp_exact` | **精确指纹。只有它能关联到已验证修复** |
| `fp_fuzzy` | **模糊指纹。只能用于召回线索** |
| `pipeline_version` | 抽取+规范化规则的版本。**参与指纹计算** |
| `occurred_at` | |
| `version_combo` | JSONB,冗余存 |
| `workdir` `repro` `other_targets` | 复现上下文(T11b):录入方携带,诊断时装载跑 G1/G2/G3。`NOT NULL DEFAULT`(空),留空 = 部署/环境失败语义,Handle 跳 G1/G2 |

**`fp_exact` 和 `fp_fuzzy` 是两个独立列,不是一列加类型标记。**

这样"已验证修复"表的外键可以只指向 `fp_exact`。想用模糊指纹直接查修复 —— SQL 写不出来。约束胜过纪律。

`raw_log_ref` 设计上有保留期(PRD 提 90 天),**但清理未实现**:清空需 UPDATE 事实面,与 `0011` append-only 触发器冲突,留待后续设计(见 `tasks/BACKLOG.md`)。当前 `raw_log_ref` 与 `excerpt` / `normalized` / 指纹一样**永久保留**。

### `dependency_change`

| 字段 | 说明 |
|---|---|
| `id` `project_id` `commit_sha` | |
| `package` `from_version` `to_version` | |
| `changed_at` | |

用途:失败时间线和依赖升级事件对齐 —— 一眼看出哪次升级把东西搞坏了。

从锁文件 diff 解析,不需要人填。

---

## 索引面

### `clause` / `clause_version`

拆两张是因为**边要指向具体某一版,不是逻辑编号**(INV-4)。

`clause`:`project_id` + `clause_id`(如 `REQ-12`)。

`clause_version`:

| 字段 | 说明 |
|---|---|
| `id` | |
| `project_id` `clause_id` | 指向 `clause` |
| `content_hash` | **唯一。所有边指向这个** |
| `version_no` `signed_at` `signed_by` `signed_commit` | 签署信息 |
| `text` `acceptance_criteria` `non_goals` | 仅供展示,权威版本在 git |
| `superseded_by_hash` | 变更链:指向取代本版的 `content_hash`(记哈希不记 id,INV-4) |
| `is_current` | |

条款一改,`content_hash` 变,旧边自动指向旧版本 → 下游全部显示"依据已变更"。**不需要额外的失效传播机制。**

### `commit_clause`

来自 commit trailer `Refs: REQ-12`。

`(commit_sha, clause_content_hash)`。**记的是绑定当时那一版的 `content_hash`,不是逻辑编号**(INV-4)。

**解析时按当时的 `is_current` 版本绑定,绑定后不再变。** 条款后来改了,旧挂钩仍指向旧版本的哈希,下游自动显示「依据已变更」。

### `test_clause`

来自测试注解 `@covers REQ-12.2`。`(test_name, clause_content_hash)`,同样记内容哈希(INV-4)。

用于算**条款覆盖率** —— 签署的条款里有多少条没有任何测试引用。

### `fingerprint_stat`

| 字段 | 说明 |
|---|---|
| `fp_exact` | 主键 |
| `fp_fuzzy` | |
| `first_seen` `last_seen` `occurrence_count` | |
| `affected_projects` | |
| `status` | `unresolved` / `fixed` / `flaky` / `upstream` / `infra` / `spec_defect` |
| `fix_ref` | 指向 git 里的已验证修复;**只能由 `fp_exact` 关联** |
| `fix_verified_at` `fix_verified_combo` | 有效性边界 |
| `diagnose_attempts` | 诊断次数,用于三轮上限 |

派生自 `failure_event`,可以随时 drop 重算。

### `trace_edge`

物化的追溯图,加速视图 A / B。

`(from_type, from_key, to_type, to_key)`,全部从上面的表推导。

**可以完全删掉重建。** 有一个 `make reindex` 命令做这件事,而且 CI 里定期跑一次 —— 重建后和现有数据 diff,不一致就是有人绕过了单向数据流。

---

## 状态字段清单

INV-3 的检查从这里读。这些字段**不允许有 UPDATE 接口**:

```
build_run.status
deploy_run.status
environment_run.status
fingerprint_stat.status      ← 例外:允许由 failure 包的归类函数与 deps 的退休检查写
                               (系统按确定性规则改写,非手动;不允许 HTTP 接口写)
```

任务完成状态**不存表**,每次从执行链算。存了就会有人想改它。

---

## 字段黑名单

INV-7 的检查从这里读。出现即失败:

```
estimated_hours  actual_hours  assignee  priority
story_points     reviewer      approver  due_date
```

---

## 索引

先不建。等有真实数据量了,拿 `EXPLAIN` 决定。

唯一现在就要的:

- `failure_event(fp_exact)` —— 查库是热路径
- `failure_event(project_id, occurred_at DESC)` —— 列表页
- `clause_version(content_hash)` —— 唯一约束自带
