-- verification_run 查询合同(P1-12,PRD §6.4.3/§8.13)。
-- 事实面只 INSERT/SELECT:历史 Run 全量保留,绝不 UPDATE/DELETE(INV-FACT-01);
-- 同一 (contract revision, ChangeSet) 的多次执行各自成行,禁止 cherry-pick 靠全量留痕(§8.7.2)。

-- name: InsertVerificationRun :one
-- 记一次验证执行(落行时带最终观测结果;行一旦落库不可改)。
INSERT INTO verification_run (
  project_id, task_id, contract_revision, contract_hash, changeset_id,
  env_name, env_definition_hash, version_combo, runtime,
  started_at, finished_at, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id;

-- name: GetVerificationRun :one
-- 按 id 取单条 Run(项目命名空间内)。
SELECT id, project_id, task_id, contract_revision, contract_hash, changeset_id,
       env_name, env_definition_hash, version_combo, runtime,
       started_at, finished_at, status
FROM verification_run
WHERE project_id = $1 AND id = $2;

-- name: ListVerificationRunsForChangeSet :many
-- 某 immutable ChangeSet 上的全部历史 Run(§8.13:保留历史 Run;新到在前)。
SELECT id, project_id, task_id, contract_revision, contract_hash, changeset_id,
       env_name, env_definition_hash, version_combo, runtime,
       started_at, finished_at, status
FROM verification_run
WHERE project_id = $1 AND changeset_id = $2
ORDER BY started_at DESC, id DESC;

-- name: ListVerificationRunsForTask :many
-- 某任务的全部历史 Run(任务页 verification timeline)。
SELECT id, project_id, task_id, contract_revision, contract_hash, changeset_id,
       env_name, env_definition_hash, version_combo, runtime,
       started_at, finished_at, status
FROM verification_run
WHERE project_id = $1 AND task_id = $2
ORDER BY started_at DESC, id DESC;
