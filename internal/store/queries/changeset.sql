-- ChangeSet 不可变候选快照与取代事实的查询(P1-11,PRD §15.3/§6.4.2,INV-CHANGE-02)。
--
-- changeset / changeset_superseded_fact 都在事实面:只 INSERT + SELECT,
-- 绝不 UPDATE/DELETE(INV-5;触发器见 migrations/0019_codegen_changeset.sql)。
-- 快照身份 id 由写入方按 internal/changeset 推导("sha256:<hex>"),
-- 同一 Run 的同一候选同 ID,重复写入幂等忽略;后续重做产生新快照 + 新取代事实,
-- superseded_by 一律从取代事实投影派生,快照本体无可变取代字段。

-- name: InsertChangeSet :exec
-- 固定一个候选快照。身份冲突 = 同一快照已存在,幂等忽略(与 clause_version 同策略)。
INSERT INTO changeset (
  id, project_id, base_commit, candidate_commit, patch_hash,
  changed_files, entities, producer_run_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO NOTHING;

-- name: GetChangeSet :one
-- 按 id 取快照本体(全部固定字段,PRD §15.3)。
SELECT id, project_id, base_commit, candidate_commit, patch_hash,
       changed_files, entities, producer_run_id, created_at
FROM changeset
WHERE id = $1;

-- name: ListChangeSetsForTask :many
-- 某 Task 的全部候选快照:经产生它们的 Run 归属到 Task(PRD §6.4.2,
-- Task 是多个 Run/ChangeSet revision 的聚合点),按固定时刻定序。
SELECT cs.id, cs.project_id, cs.base_commit, cs.candidate_commit, cs.patch_hash,
       cs.changed_files, cs.entities, cs.producer_run_id, cs.created_at
FROM changeset cs
JOIN codegen_run cr ON cr.id = cs.producer_run_id
WHERE cr.project_id = $1 AND cr.task_id = $2
ORDER BY cs.created_at, cs.id;

-- name: InsertChangeSetSupersededFact :exec
-- 追加一条取代事实(old 被 new 取代,reason: escalation|rework|spec_defect)。
-- 同一事实(同 old/new/reason/时刻)重复追加幂等忽略;不同时刻各自成行。
INSERT INTO changeset_superseded_fact (old_changeset_id, new_changeset_id, reason, superseded_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (old_changeset_id, new_changeset_id, reason, superseded_at) DO NOTHING;

-- name: ListChangeSetSupersessionsForChangeSet :many
-- 某快照参与的取代事实(作为 old 或 new):派生 superseded_by / 取代链的投影源。
SELECT old_changeset_id, new_changeset_id, reason, superseded_at
FROM changeset_superseded_fact
WHERE old_changeset_id = $1 OR new_changeset_id = $1
ORDER BY superseded_at, id;

-- name: ListChangeSetSupersessions :many
-- 全部取代事实(排序确定):取代投影整体重建/对账用。
SELECT old_changeset_id, new_changeset_id, reason, superseded_at
FROM changeset_superseded_fact
ORDER BY superseded_at, id;
