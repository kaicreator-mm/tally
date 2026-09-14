-- deploy_run 的查询。事实面只 INSERT 和 SELECT,绝不 UPDATE/DELETE(INV-5)。

-- name: InsertDeployRun :one
-- 记一次部署。artifact_digest 必填:没有 digest 的部署不允许发生。
-- 制品 → 部署 的追溯坐标。追溯视图/集成测试也用它给 failure_event 的 deploy_run_id 来源外键配真实行。
INSERT INTO deploy_run (
  project_id, artifact_digest, build_run_id, target_env, tool,
  started_at, finished_at, status, exit_code, health_check_result, version_combo
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9, $10, $11
)
RETURNING id;

-- name: ListDeployRuns :many
-- 部署历史(新→旧,上限 200 行):部署页从这里读;一键回滚的目标 digest 也出自这些行。
SELECT id, project_id, artifact_digest, build_run_id, target_env, tool,
       started_at, finished_at, status, exit_code, health_check_result, version_combo
FROM deploy_run
ORDER BY started_at DESC, id DESC
LIMIT 200;

-- name: TaskStability :many
-- 每个项目最近一次部署 + 部署后是否有失败(算任务「部署/稳定」两栏用,INV-3 状态算出):
-- last_deploy_status 给出部署结果;has_post_deploy_failure 为真 = 部署后又炸了(有复发)。
-- 最近部署取 started_at 最大(同刻取 id 最大);无部署的项目不出现(那两栏落「无记录」)。
SELECT d.project_id,
       d.status AS last_deploy_status,
       EXISTS (
         SELECT 1 FROM failure_event fe
         WHERE fe.project_id = d.project_id AND fe.occurred_at > d.started_at
       ) AS has_post_deploy_failure
FROM (
  SELECT DISTINCT ON (project_id) project_id, status, started_at
  FROM deploy_run
  ORDER BY project_id, started_at DESC, id DESC
) d
ORDER BY d.project_id;
