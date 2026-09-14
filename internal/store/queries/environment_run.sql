-- environment_run 的查询。事实面只 INSERT 和 SELECT,绝不 UPDATE/DELETE(INV-5)。

-- name: InsertEnvironmentRun :one
-- 记一次环境拉起。失败诊断的集成测试也用它给 failure_event 的来源外键配一个真实行。
INSERT INTO environment_run (project_id, env_name, definition_hash, started_at, status, version_combo)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: InsertEnvironmentRunOutcome :one
-- 记一次环境拉起的完整结果(含结束时刻与健康检查明细)。
-- environment 包 Up 成功/失败都走这里:version_combo 非空(镜像 tag + 运行时版本)。
INSERT INTO environment_run (project_id, env_name, definition_hash, started_at, finished_at, status, health_check_result, version_combo)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id;

-- name: LatestEnvironmentRuns :many
-- 每个环境(project + env_name)最近一次拉起记录:环境页的状态从这里算(INV-3 状态是算出来的)。
-- 事实面只追加,同一环境多次拉起留下多行;取每组 started_at 最大(同刻取 id 最大)的那行。
SELECT DISTINCT ON (project_id, env_name)
  id, project_id, env_name, definition_hash, started_at, finished_at, status, health_check_result, version_combo
FROM environment_run
ORDER BY project_id, env_name, started_at DESC, id DESC;
