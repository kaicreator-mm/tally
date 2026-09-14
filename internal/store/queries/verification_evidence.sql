-- verification_evidence / verification_evidence_ruling 查询合同(P1-12)。
-- 事实面只 INSERT/SELECT(INV-FACT-01):evidence 诞生即 valid,失效只能追加裁定行,
-- 任何行都不改写(PRD §8.8.1:合法 FAIL 不被后来的 PASS 抹掉);
-- applicability 由行上精确 scope 引用相对当前 desired 上下文判定(§8.8.2)。

-- name: InsertVerificationEvidence :one
-- 记一条 observed evidence 事实(result + provenance + 精确 scope 引用)。
INSERT INTO verification_evidence (
  fact_id, run_id, project_id, task_id, acceptance_id, binding_id,
  visibility_policy, verdict, recorder_kind, recorder_actor, validator_version,
  command, exit_code, stdout_digest, stderr_digest, raw_output_ref, recorded_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
RETURNING id;

-- name: GetVerificationEvidenceByFactID :one
-- 按证据事实身份取单条(fact_id 唯一且永不复用)。
SELECT id, fact_id, run_id, project_id, task_id, acceptance_id, binding_id,
       visibility_policy, verdict, recorder_kind, recorder_actor, validator_version,
       command, exit_code, stdout_digest, stderr_digest, raw_output_ref, recorded_at
FROM verification_evidence
WHERE project_id = $1 AND fact_id = $2;

-- name: ListVerificationEvidenceForRun :many
-- 一次 Run 产生的全部 evidence(§6.4.3:Run → Evidence 一对多)。
SELECT id, fact_id, run_id, project_id, task_id, acceptance_id, binding_id,
       visibility_policy, verdict, recorder_kind, recorder_actor, validator_version,
       command, exit_code, stdout_digest, stderr_digest, raw_output_ref, recorded_at
FROM verification_evidence
WHERE run_id = $1 AND project_id = $2
ORDER BY recorded_at, id;

-- name: ListApplicableValidEvidence :many
-- 当前 desired 上下文的 gate 输入(§8.8.1/8.8.2):evidence 必须精确落在
-- project/task/acceptance/contract revision/ChangeSet 五维 scope 上(applicability),
-- 且未被任何失效裁定命中(validity)。裁定表只收失效裁定,存在裁定行 = 已失效。
SELECT e.id, e.fact_id, e.run_id, e.project_id, e.task_id, e.acceptance_id, e.binding_id,
       e.visibility_policy, e.verdict, e.recorder_kind, e.recorder_actor, e.validator_version,
       e.command, e.exit_code, e.stdout_digest, e.stderr_digest, e.raw_output_ref, e.recorded_at
FROM verification_evidence AS e
JOIN verification_run AS r ON r.id = e.run_id
WHERE e.project_id = $1
  AND e.task_id = $2
  AND e.acceptance_id = $3
  AND r.contract_revision = $4
  AND r.contract_hash = $5
  AND r.changeset_id = $6
  AND NOT EXISTS (
    SELECT 1 FROM verification_evidence_ruling AS g WHERE g.evidence_id = e.id
  )
ORDER BY e.recorded_at, e.id;

-- name: InsertVerificationEvidenceRuling :one
-- 追加一条失效裁定(带 provenance;不改写被裁定的 evidence 行,§8.8.1)。
INSERT INTO verification_evidence_ruling (evidence_id, project_id, kind, reason, recorder_kind, recorder_actor, ruled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: ListVerificationEvidenceRulings :many
-- 某条 evidence 的裁定序列(append-only,旧到新;validity 由此派生)。
SELECT id, evidence_id, project_id, kind, reason, recorder_kind, recorder_actor, ruled_at
FROM verification_evidence_ruling
WHERE evidence_id = $1 AND project_id = $2
ORDER BY ruled_at, id;
