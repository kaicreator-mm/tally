-- human_approval 的读写合同(P1-13)。
--
-- human_approval 是事实面(migrations/0021):Owner 决定只以 HumanApprovalEvidence
-- 入库(§18.1:human approval record 是一种 Evidence,不是
-- `UPDATE task SET status='APPROVED'`),APPROVED 等权威状态由状态引擎从证据
-- 派生(INV-STATE-02)。本文件因此只有 INSERT 和 SELECT:没有任何 UPDATE/DELETE
-- (INV-FACT-01,数据库层另有行级触发器兜底),更没有触碰任务状态的查询(AC2)。
--
-- 行语义(§8.10/§19):一条决定证据 = decision(approve|reject)+ 必填 reason +
-- decided_by/decided_at provenance + 指向的 changeset_id/refs;豁免行另带
-- waiver provenance(waived_independence/waiver_reason_code/waiver_refs)。
-- 历史决定永不改写:同一候选后来追加的新决定是新行;读侧取「最新一行」时,
-- 更早的行仍是可追溯的历史证据(后来 pass 不洗白更早的记录,反之亦然)。
-- strict_satisfied / strict_waived 的区分读行内 waived_independence 原值,
-- 永远逐行携带,报表不得把 strict_waived 行并入 strict_satisfied 呈现(§8.10)。

-- name: InsertHumanApproval :one
-- 追加一条决定证据(AC1)。事实面唯一写路径:INSERT 且只 INSERT。
-- 豁免 provenance 由 0021 的 CHECK 兜底:waived_independence=true 时
-- waiver_reason_code 必填;waiver_refs 记 compromised acceptances/bindings、
-- exposure refs、coverage gap 的原样 JSON 数组。返回新行 id。
INSERT INTO human_approval (
  project_id, task_id, changeset_id,
  decision, reason, decided_by, decided_at,
  waived_independence, waiver_reason_code, waiver_refs, refs
) VALUES (
  $1, $2, $3,
  $4, $5, $6, $7,
  $8, $9, $10, $11
)
RETURNING id;

-- name: GetHumanApproval :one
-- 按 id 取一条决定证据:状态引擎/审计回查单条 provenance 用。
SELECT id, project_id, task_id, changeset_id,
       decision, reason, decided_by, decided_at,
       waived_independence, waiver_reason_code, waiver_refs, refs
FROM human_approval
WHERE id = $1;

-- name: ListHumanApprovalsByTask :many
-- 某任务的全部决定证据(新→旧,decided_at 同刻按 id 定序):状态派生与
-- 审计的时间线视图。历史行全部可见,不覆盖、不瘦身。
SELECT id, project_id, task_id, changeset_id,
       decision, reason, decided_by, decided_at,
       waived_independence, waiver_reason_code, waiver_refs, refs
FROM human_approval
WHERE project_id = $1 AND task_id = $2
ORDER BY decided_at DESC, id DESC;

-- name: GetLatestHumanApprovalByChangeset :one
-- 某 ChangeSet 的最新一条决定(§19:Review 输入是固定 revision 的 ChangeSet):
-- 状态引擎据此派生该候选当前是否带有效 APPROVED 证据。无任何决定行时返回
-- pgx.ErrNoRows(未决定不是错误状态,是证据缺席)。
SELECT id, project_id, task_id, changeset_id,
       decision, reason, decided_by, decided_at,
       waived_independence, waiver_reason_code, waiver_refs, refs
FROM human_approval
WHERE project_id = $1 AND task_id = $2 AND changeset_id = $3
ORDER BY decided_at DESC, id DESC
LIMIT 1;

-- name: ListWaivedHumanApprovalsByProject :many
-- 某项目全部豁免行(新→旧):strict_waived 与 strict_satisfied 的分开统计源
-- (§38.5 independence waiver rate / strict_satisfied vs strict_waived)。
-- 只返回 waived_independence=true 的行;strict_satisfied 行不进本查询。
SELECT id, project_id, task_id, changeset_id,
       decision, reason, decided_by, decided_at,
       waived_independence, waiver_reason_code, waiver_refs, refs
FROM human_approval
WHERE project_id = $1 AND waived_independence
ORDER BY decided_at DESC, id DESC;
