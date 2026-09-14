-- CodegenRun 过程记录的查询(P1-11,PRD §15)。
--
-- codegen_run 是过程记录,不是事实面:行在运行开始时以 status='running' 写入,
-- 由 workflow kernel 推移状态(§17 「CodegenRun 状态迁移」),因此允许有限的
-- 状态 UPDATE —— 但只经下面两条带谓词的路径:进度更新与终态迁移都要求
-- status='running' 且按主键定位,终态迁移命中 0 行(:execrows)即可判定
-- 「不存在或已终态」,终态一次性,不可回退重写。

-- name: InsertCodegenRun :one
-- 运行开始:写入过程记录,身份坐标(PRD §15.2 可追溯字段)此时即已固定。
-- status 恒以 'running' 起步;escalated_from 仅升级新开的 Run 非空(§11.2.1)。
INSERT INTO codegen_run (
  project_id, task_id, role,
  runtime, runtime_version, model, behavior_revision, cost_plane,
  base_commit, task_revision, verification_revision, context_pack_hash,
  workspace_id, agent_session_id, runtime_capability_snapshot,
  escalated_from, started_at, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, 'running')
RETURNING id;

-- name: GetCodegenRun :one
-- 按 id 取整行(全部 §15.1 字段)。
SELECT id, project_id, task_id, role,
       runtime, runtime_version, model, behavior_revision, cost_plane,
       base_commit, task_revision, verification_revision, context_pack_hash,
       workspace_id, agent_session_id,
       started_at, finished_at, status, iterations, escalated_from,
       input_tokens, output_tokens, cost_micros,
       result_patch_hash, agent_claim, validation_result, failure_reason,
       runtime_capability_snapshot, conformance_evidence_ref
FROM codegen_run
WHERE id = $1;

-- name: ListCodegenRunsForTask :many
-- 某 Task 的全部 Run 时间线(PRD §29 CodegenRun timeline),按开始时刻定序。
SELECT id, project_id, task_id, role,
       runtime, runtime_version, model, behavior_revision, cost_plane,
       base_commit, task_revision, verification_revision, context_pack_hash,
       workspace_id, agent_session_id,
       started_at, finished_at, status, iterations, escalated_from,
       input_tokens, output_tokens, cost_micros,
       result_patch_hash, agent_claim, validation_result, failure_reason,
       runtime_capability_snapshot, conformance_evidence_ref
FROM codegen_run
WHERE project_id = $1 AND task_id = $2
ORDER BY started_at, id;

-- name: UpdateCodegenRunProgress :execrows
-- 运行中进度(iterations 与 usage/USAGE_UPDATED 到达的 token/成本)。
-- 仅对仍处 running 的 Run 生效;命中 0 行 = Run 不存在或已终态,调用方必须放弃。
UPDATE codegen_run
SET iterations = $2, input_tokens = $3, output_tokens = $4, cost_micros = $5
WHERE id = $1 AND status = 'running';

-- name: FinishCodegenRun :execrows
-- 终态迁移(§11.2.1:completed / failed / escalated / cancelled),一次性:
-- 谓词限定 status='running',二次迁移或对未知 id 命中 0 行,不可重写终态。
-- iterations 取终值一并落定;result_patch_hash 仅在有固定 candidate 的 Run 非空。
UPDATE codegen_run
SET status = $2, finished_at = $3, iterations = $4,
    input_tokens = $5, output_tokens = $6, cost_micros = $7,
    result_patch_hash = $8, agent_claim = $9,
    validation_result = $10, failure_reason = $11
WHERE id = $1 AND status = 'running';
