-- build_run 的查询。事实面只 INSERT 和 SELECT,绝不 UPDATE/DELETE(INV-5)。

-- name: InsertBuildRun :one
-- 记一次构建。commit_sha 与成功时的 artifact_digest 是追溯链(commit → 制品)的坐标。
-- 追溯视图/集成测试也用它给 failure_event 的 build_run_id 来源外键配一个真实行。
INSERT INTO build_run (
  project_id, commit_sha, ci_run_url, started_at, finished_at,
  exit_code, status, toolchain, artifact_digest, version_combo
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10
)
RETURNING id;

-- name: TaskBuildSummary :many
-- 每个项目最近一次构建(算任务「合并/构建」两栏用,INV-3 状态算出):
-- 有行即该项目有构建记录(代码合了),status/exit_code 给出构建结果。
-- 取 started_at 最大(同刻取 id 最大)的那行。
SELECT DISTINCT ON (project_id)
  project_id, commit_sha, status, exit_code, started_at
FROM build_run
ORDER BY project_id, started_at DESC, id DESC;
