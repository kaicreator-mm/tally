-- failure_event 的查询。事实面只 INSERT(在 failure 包)和 SELECT,
-- 绝不 UPDATE/DELETE(INV-5 / INV-FACT-01)。
--
-- 复现上下文持久化合同(PRD §21.6 结构化 Repro,0016 起生效):
--   - 新写入只落结构化列:repro_command(internal/command.CommandSpec 的
--     规范 JSON,字段集 argv/cwd/env/timeout/shell)、other_target_commands
--     ([{"target":…,"command":…}] 数组);
--   - 0013 的旧文本三列 workdir/repro/other_targets 不再写入(NOT NULL DEFAULT ''),
--     仅作为存量行的兼容读取来源;
--   - 兼容路径:repro_command 为 JSON null(0016 前的存量行)时,按旧执行语义
--     重建 CommandSpec{Argv: strings.Fields(repro), Cwd: workdir, Shell: false}
--     (旧 worker 正是 strings.Fields(repro) + Dir(workdir) 执行);
--     other_target_commands 为空数组时回退 other_targets 里的目标名(无命令)。

-- name: ListFailureEventsWithRaw :many
-- 取带留存原始日志的失败事件,供规则升级后重算指纹(cmd/tally reindex-fingerprints)。
-- 只选重算所需的列;raw_log_ref 虽可空,但这里过滤掉空值。
SELECT id, project_id, source, raw_log_ref, fp_exact, fp_fuzzy, pipeline_version
FROM failure_event
WHERE raw_log_ref IS NOT NULL
ORDER BY id;

-- name: InsertFailureEvent :one
-- 记一条失败事件(三个来源外键恰好一个非空,由数据库约束兜底)。
-- 事实面只追加:本表只有这一条 INSERT,没有任何修改查询(INV-5)。
-- 复现上下文只落结构化列(§21.6):repro_command 是 CommandSpec 规范 JSON
-- (internal/command 的 MarshalJSON 产物),other_target_commands 是
-- [{"target":…,"command":…}] 数组;旧文本三列不再写入,留默认值。
INSERT INTO failure_event (
  project_id, source, build_run_id, deploy_run_id, environment_run_id,
  raw_log_ref, raw_log_bytes, excerpt, normalized,
  fp_exact, fp_fuzzy, pipeline_version, occurred_at, version_combo,
  repro_command, other_target_commands
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9,
  $10, $11, $12, $13, $14,
  $15, $16
)
RETURNING id;

-- name: ListLatestFailureExcerpts :many
-- 每个精确指纹最近一次的片段与来源:失败列表按指纹归并后,用它给指纹行人读摘要。
-- 摘要是脱敏后的关键片段,只展示;指纹关联修复仍只认 fp_exact。
SELECT DISTINCT ON (fp_exact) fp_exact, source, excerpt, occurred_at
FROM failure_event
ORDER BY fp_exact, occurred_at DESC, id DESC;

-- name: CountFailuresByDay :many
-- 近 90 天按天统计的失败数:失败时间线的一根轴,与依赖变更在同一天刻度上对齐。
SELECT date_trunc('day', occurred_at)::date AS day, count(*)::bigint AS failures
FROM failure_event
WHERE occurred_at >= now() - interval '90 days'
GROUP BY day
ORDER BY day;

-- name: ListFailureEventsByFingerprint :many
-- 某指纹的发生记录(新→旧,上限 100):指纹详情页的执行侧。
-- commit 经 build_run 直连或经 deploy_run→制品→build_run 回推;环境来源的失败没有 commit 链,落空。
-- 关联条款一列由 T05c 用 LEFT JOIN 补,本查询只提供执行侧坐标。
SELECT fe.id, fe.project_id, fe.source, fe.occurred_at, fe.excerpt,
       COALESCE(br.commit_sha, (
         SELECT b2.commit_sha
         FROM deploy_run dr
         JOIN build_run b2 ON b2.artifact_digest = dr.artifact_digest
         WHERE dr.id = fe.deploy_run_id
         ORDER BY b2.id DESC
         LIMIT 1
       ), '') AS commit_sha
FROM failure_event fe
LEFT JOIN build_run br ON br.id = fe.build_run_id
WHERE fe.fp_exact = $1
ORDER BY fe.occurred_at DESC, fe.id DESC
LIMIT 100;

-- name: GetLatestFailureEventByFingerprint :one
-- 某精确指纹最近一次的完整失败事件:诊断执行(T10c)按指纹装载失败、重组诊断输入。
-- 只 SELECT 诊断所需列;不碰事实面(INV-5)。
-- 复现上下文按合同双轨返回:结构化两列 + 兼容三列。repro_command 为 null 的
-- 存量行由装载方按文件头兼容路径从 workdir/repro 重建。
SELECT project_id, source, build_run_id, deploy_run_id, environment_run_id,
       raw_log_ref, raw_log_bytes, excerpt, normalized,
       fp_exact, fp_fuzzy, pipeline_version, occurred_at, version_combo,
       repro_command, other_target_commands,
       workdir, repro, other_targets
FROM failure_event
WHERE fp_exact = $1
ORDER BY occurred_at DESC, id DESC
LIMIT 1;

-- name: GetLatestFailureEventByProjectAndFingerprint :one
-- 某项目 + 某精确指纹最近一次的复现上下文(D-c4,T11c 退休检查用):
-- 结构化 repro_command + 兼容 workdir/repro 双轨;repro_command 为 null 的
-- 存量行按文件头兼容路径重建。无对应事件 → 无上下文,退休检查跳过不猜。
-- 只 SELECT,不碰事实面(INV-5)。
SELECT repro_command, workdir, repro
FROM failure_event
WHERE project_id = $1 AND fp_exact = $2
ORDER BY occurred_at DESC, id DESC
LIMIT 1;
