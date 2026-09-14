-- evidence_exposure 查询合同(P1-12,PRD §8.6)。
-- Exposure 是 Fact,不是 Contract mutation:只追加,绝不 UPDATE/DELETE(INV-FACT-01),
-- 也不回写 VerificationContract。读取面服务 Review 可见性(P1 Trust Gate:至少一次
-- exposure fact 让 effective information independence 变为 compromised 并在 Review 可见)
-- 与 §38.5 的「exposure events by channel/scope」统计。

-- name: InsertEvidenceExposure :one
-- 追加一条暴露事实(channel + binding 引用 + lineage 引用)。
-- detail 只放引用/坐标类上下文,不得携带 held-out 内容或可识别 verdict(INV-VER-05)。
INSERT INTO evidence_exposure (
  project_id, task_id, contract_revision, contract_hash, binding_id,
  acceptance_id, changeset_id, channel, detail, exposed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: ListEvidenceExposureForBinding :many
-- 某条 held-out binding 的全部暴露事实(§8.10:Waiver 需引用 compromised bindings 的 exposure refs)。
SELECT id, project_id, task_id, contract_revision, contract_hash, binding_id,
       acceptance_id, changeset_id, channel, detail, exposed_at
FROM evidence_exposure
WHERE project_id = $1 AND contract_hash = $2 AND binding_id = $3
ORDER BY exposed_at DESC, id DESC;

-- name: ListEvidenceExposureForChangeSet :many
-- 某候选 lineage 上的暴露事实(selection/content 污染判定,§8.6.2)。
SELECT id, project_id, task_id, contract_revision, contract_hash, binding_id,
       acceptance_id, changeset_id, channel, detail, exposed_at
FROM evidence_exposure
WHERE project_id = $1 AND changeset_id = $2
ORDER BY exposed_at DESC, id DESC;

-- name: ListEvidenceExposureForTask :many
-- 某任务的暴露事实时间线(Review 面按任务展开)。
SELECT id, project_id, task_id, contract_revision, contract_hash, binding_id,
       acceptance_id, changeset_id, channel, detail, exposed_at
FROM evidence_exposure
WHERE project_id = $1 AND task_id = $2
ORDER BY exposed_at DESC, id DESC;
