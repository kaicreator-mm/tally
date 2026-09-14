-- 追溯挂钩与视图的查询(T08)。
--
-- clause / clause_version / commit_clause / test_clause / trace_edge 都是索引面
-- (派生数据):可随时 drop 重算,允许全部操作 —— 不受 INV-5 只追加约束。
-- 挂钩一律记 content_hash,不记逻辑编号(INV-4);绑定按当时的当前版本,绑定后不变。
-- commit_clause / test_clause / trace_edge 的身份带 project_id 命名空间(0015,PRD §22):
-- 同 commit_sha / test_name / 边 tuple 在不同项目互不合并。

-- name: InsertClause :exec
-- 登记一个逻辑条款(编号)。已存在则忽略(幂等,reindex 会反复调)。
INSERT INTO clause (project_id, clause_id)
VALUES ($1, $2)
ON CONFLICT (project_id, clause_id) DO NOTHING;

-- name: InsertClauseVersion :exec
-- 登记条款的某一版(不可变)。content_hash 唯一;重复签署同一内容时忽略。
-- 所有追溯边指向这里的 content_hash(INV-4)。
INSERT INTO clause_version (
  project_id, clause_id, content_hash, version_no,
  signed_at, signed_by, signed_commit,
  text, acceptance_criteria, non_goals,
  superseded_by_hash, is_current
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (content_hash) DO NOTHING;

-- name: SetClauseVersionCurrent :exec
-- 把某条款的当前版本切换到指定 content_hash:同一条款下只有命中的那版 is_current=true。
-- 索引面的元数据维护,非状态字段(INV-3 只禁事实面 status 的手改)。
UPDATE clause_version
SET is_current = (content_hash = $3)
WHERE project_id = $1 AND clause_id = $2;

-- name: CurrentClauseVersionHashAt :one
-- 取条款在时刻 $3 的当前版本 content_hash:签署于该时刻之前(含)的最新一版。
-- 这是"按当时当前版本绑定"的实现(INV-4):条款后来升级,旧 commit 仍落回当时那版的哈希。
-- signed_at 为空视为未签署、无法在时间轴上定位,不参与绑定(比较为 NULL 自动落选)。
SELECT content_hash
FROM clause_version
WHERE project_id = $1 AND clause_id = $2 AND signed_at <= $3
ORDER BY signed_at DESC, version_no DESC
LIMIT 1;

-- name: ListCurrentClauseVersions :many
-- 列出项目当前(is_current)的全部条款版本,算条款覆盖率用。
SELECT clause_id, content_hash
FROM clause_version
WHERE project_id = $1 AND is_current
ORDER BY clause_id;

-- name: InsertCommitClause :exec
-- 记一条 commit trailer 挂钩(commit_sha → 当时版本的 content_hash)。幂等。
-- 身份含 project_id:同 commit_sha 在不同项目各自成行,不合并。
INSERT INTO commit_clause (project_id, commit_sha, clause_content_hash)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, commit_sha, clause_content_hash) DO NOTHING;

-- name: InsertTestClause :exec
-- 记一条测试注解挂钩(test_name → 当时版本的 content_hash)。幂等。
-- 身份含 project_id:同 test_name 在不同项目各自成行,不合并。
INSERT INTO test_clause (project_id, test_name, clause_content_hash)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, test_name, clause_content_hash) DO NOTHING;

-- name: ListCommitClauses :many
-- 全部 commit 挂钩(带归属项目),构建追溯图(clause → commit)用。
SELECT project_id, commit_sha, clause_content_hash
FROM commit_clause
ORDER BY project_id, commit_sha, clause_content_hash;

-- name: ListTestClauses :many
-- 全部测试挂钩(带归属项目),算覆盖率用。
SELECT project_id, test_name, clause_content_hash
FROM test_clause
ORDER BY project_id, test_name, clause_content_hash;

-- name: ListCommitClausesForProject :many
-- 某项目的 commit 挂钩:VerifyIndex 的项目内一致性快照用。
-- 0015 起挂钩身份自带 project_id,直接按列收窄,不再借 clause_version 归属
-- (此前靠 join 避免共享库里别项目的挂钩混进本项目的一致性 diff)。
SELECT commit_sha, clause_content_hash
FROM commit_clause
WHERE project_id = $1
ORDER BY commit_sha, clause_content_hash;

-- name: ListTestClausesForProject :many
-- 某项目的测试挂钩(同上,直接按归属项目收窄)。
SELECT test_name, clause_content_hash
FROM test_clause
WHERE project_id = $1
ORDER BY test_name, clause_content_hash;

-- name: ListClauseProjects :many
-- 有需求层(登记过条款)的项目列表:覆盖率/视图 A 的项目选择从这里来。
SELECT DISTINCT project_id
FROM clause
ORDER BY project_id;

-- name: ListClauseReferences :many
-- 本项目条款的下游引用(commit trailer + 测试注解),带条款编号与引用时的哈希,
-- 算「依据已变更」用(INV-4:引用的哈希对不上当前版即变更)。
SELECT cv.clause_id, cc.clause_content_hash AS ref_hash, 'commit'::text AS kind
FROM commit_clause cc
JOIN clause_version cv ON cv.content_hash = cc.clause_content_hash
WHERE cc.project_id = $1
UNION ALL
SELECT cv.clause_id, tc.clause_content_hash AS ref_hash, 'test'::text AS kind
FROM test_clause tc
JOIN clause_version cv ON cv.content_hash = tc.clause_content_hash
WHERE tc.project_id = $1
ORDER BY clause_id, ref_hash, kind;

-- name: DeleteCommitClausesForProject :exec
-- reindex 重挂前清掉某项目的所有 commit 挂钩(按归属项目,0015 起直接落列)。
DELETE FROM commit_clause
WHERE project_id = $1;

-- name: DeleteTestClausesForProject :exec
-- reindex 重挂前清掉某项目的所有测试挂钩(按归属项目,0015 起直接落列)。
DELETE FROM test_clause
WHERE project_id = $1;

-- name: ListBuildRunLinks :many
-- 构建链坐标:commit → build → 制品。构建追溯图用。
SELECT id, commit_sha, artifact_digest
FROM build_run
ORDER BY id;

-- name: ListDeployRunLinks :many
-- 部署链坐标:制品 → 部署。构建追溯图用。
SELECT id, artifact_digest
FROM deploy_run
ORDER BY id;

-- name: ListFailureLinks :many
-- 失败链坐标:build/deploy → 精确指纹。构建追溯图用(环境来源的失败无 commit 链,不参与)。
SELECT build_run_id, deploy_run_id, fp_exact
FROM failure_event
ORDER BY id;

-- name: ClearTraceEdges :exec
-- reindex:清空物化追溯图,随后从机器记录重算。
DELETE FROM trace_edge;

-- name: DeleteStaleTraceEdges :exec
-- 删掉不在重算结果集里的追溯边(幂等重算的「清陈旧」一步,配 InsertTraceEdge 的幂等写入)。
-- 按完整 tuple(含 project_id)与现有边逐行比对(NOT EXISTS):此前按四列各自 NOT IN 判定,
-- 每列的存在性来自不同的行,会笛卡尔式误判 —— 该删的边因各列恰好在别的边里出现而被留下。
-- 比起 ClearTraceEdges 全清重写,稳态下不写一行,也让只读检查(VerifyIndex)不再反复重铺全表。
DELETE FROM trace_edge te
WHERE NOT EXISTS (
  SELECT 1
  FROM (SELECT unnest($1::text[]) AS project_id,
               unnest($2::text[]) AS from_type,
               unnest($3::text[]) AS from_key,
               unnest($4::text[]) AS to_type,
               unnest($5::text[]) AS to_key) AS want
  WHERE want.project_id = te.project_id
    AND want.from_type  = te.from_type
    AND want.from_key   = te.from_key
    AND want.to_type    = te.to_type
    AND want.to_key     = te.to_key
);

-- name: InsertTraceEdge :exec
-- 写一条追溯边。幂等(主键冲突忽略)。身份含 project_id(§22):同 tuple 跨项目不合并。
INSERT INTO trace_edge (project_id, from_type, from_key, to_type, to_key)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, from_type, from_key, to_type, to_key) DO NOTHING;

-- name: ListTraceEdges :many
-- 读回全部追溯边(排序保证确定序),reindex --check 与重算结果 diff 用。
SELECT project_id, from_type, from_key, to_type, to_key
FROM trace_edge
ORDER BY project_id, from_type, from_key, to_type, to_key;

-- name: RebuildFingerprintStats :exec
-- reindex:从 failure_event 重算 fingerprint_stat 的派生统计列(次数、首末见、受影响项目)。
-- 只更新派生列,status / fix_ref / 验证边界 / 诊断次数不在其中 ——
-- 那些由 failure 包的归类/入库流程写(见 docs/03-data-model.md 的登记例外),非 failure_event 可推导,
-- 重算不得抹掉。affected_projects 排序保证确定序(reindex --check 的 diff 稳定)。
INSERT INTO fingerprint_stat (fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects)
SELECT agg.fp_exact, agg.fp_fuzzy, agg.first_seen, agg.last_seen, agg.occurrence_count,
       (SELECT array_agg(p ORDER BY p) FROM unnest(agg.affected_projects) p) AS affected_projects
FROM (
  SELECT fe.fp_exact AS fp_exact,
         min(fe.fp_fuzzy) AS fp_fuzzy,
         min(fe.occurred_at) AS first_seen,
         max(fe.occurred_at) AS last_seen,
         count(*)::bigint AS occurrence_count,
         array_agg(DISTINCT fe.project_id) AS affected_projects
  FROM failure_event fe
  GROUP BY fe.fp_exact
) agg
ON CONFLICT (fp_exact) DO UPDATE SET
  fp_fuzzy = EXCLUDED.fp_fuzzy,
  first_seen = EXCLUDED.first_seen,
  last_seen = EXCLUDED.last_seen,
  occurrence_count = EXCLUDED.occurrence_count,
  affected_projects = EXCLUDED.affected_projects;

-- name: ListFingerprintStatDerived :many
-- 读回 fingerprint_stat 的派生列(排序确定),reindex --check 与 failure_event 重算结果 diff 用。
SELECT fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects
FROM fingerprint_stat
ORDER BY fp_exact;

-- name: ViewClauseImpl :one
-- 视图 A(某条款的实现累计失败,PRD 第六部分):
-- 实现提交数、构建失败数、部署失败数、其中同一指纹的最高复发次数、条款变更次数、首次签署时刻。
-- 链:条款各版 content_hash → commit_clause → build_run →(build 失败 / 制品 → deploy_run → deploy 失败)。
-- 条款年龄由调用方用 first_signed_at 算;失败构成按指纹另查 ViewClauseImplFailures。
WITH clause_hashes AS (
  SELECT cv.content_hash FROM clause_version cv WHERE cv.project_id = $1 AND cv.clause_id = $2
),
impl_commits AS (
  SELECT DISTINCT cc.commit_sha
  FROM commit_clause cc
  WHERE cc.clause_content_hash IN (SELECT content_hash FROM clause_hashes)
),
clause_builds AS (
  SELECT br.id FROM build_run br
  WHERE br.commit_sha IN (SELECT commit_sha FROM impl_commits)
),
build_failures AS (
  SELECT fe.fp_exact FROM failure_event fe
  WHERE fe.build_run_id IN (SELECT id FROM clause_builds)
),
clause_deploys AS (
  SELECT d.id FROM deploy_run d
  WHERE d.artifact_digest IN (
    SELECT br.artifact_digest FROM build_run br
    WHERE br.id IN (SELECT id FROM clause_builds) AND br.artifact_digest IS NOT NULL
  )
),
deploy_failures AS (
  SELECT fe.fp_exact FROM failure_event fe
  WHERE fe.deploy_run_id IN (SELECT id FROM clause_deploys)
),
all_failures AS (
  SELECT fp_exact FROM build_failures
  UNION ALL
  SELECT fp_exact FROM deploy_failures
)
SELECT
  (SELECT count(*) FROM impl_commits)::bigint AS impl_commits,
  (SELECT count(*) FROM build_failures)::bigint AS build_failures,
  (SELECT count(*) FROM deploy_failures)::bigint AS deploy_failures,
  (SELECT coalesce(max(c), 0) FROM (SELECT count(*) c FROM all_failures GROUP BY fp_exact) t)::bigint AS top_fp_recurrence,
  (SELECT count(*) FROM clause_hashes)::bigint AS change_count,
  (SELECT min(cv2.signed_at) FROM clause_version cv2 WHERE cv2.project_id = $1 AND cv2.clause_id = $2)::timestamptz AS first_signed_at;

-- name: ViewClauseImplFailures :many
-- 视图 A 的失败构成:按指纹列出本条款实现链上的失败,标注处置分类(fingerprint_stat.status)。
-- 指纹还没被归类时分类落回 unresolved(LEFT JOIN,不因没索引行而丢失败)。
WITH clause_hashes AS (
  SELECT cv.content_hash FROM clause_version cv WHERE cv.project_id = $1 AND cv.clause_id = $2
),
impl_commits AS (
  SELECT DISTINCT cc.commit_sha
  FROM commit_clause cc
  WHERE cc.clause_content_hash IN (SELECT content_hash FROM clause_hashes)
),
clause_builds AS (
  SELECT br.id FROM build_run br
  WHERE br.commit_sha IN (SELECT commit_sha FROM impl_commits)
),
build_failures AS (
  SELECT fe.fp_exact FROM failure_event fe
  WHERE fe.build_run_id IN (SELECT id FROM clause_builds)
),
clause_deploys AS (
  SELECT d.id FROM deploy_run d
  WHERE d.artifact_digest IN (
    SELECT br.artifact_digest FROM build_run br
    WHERE br.id IN (SELECT id FROM clause_builds) AND br.artifact_digest IS NOT NULL
  )
),
deploy_failures AS (
  SELECT fe.fp_exact FROM failure_event fe
  WHERE fe.deploy_run_id IN (SELECT id FROM clause_deploys)
),
all_failures AS (
  SELECT fp_exact FROM build_failures
  UNION ALL
  SELECT fp_exact FROM deploy_failures
)
SELECT af.fp_exact,
       count(*)::bigint AS occurrence,
       coalesce(fs.status, 'unresolved') AS classification
FROM all_failures af
LEFT JOIN fingerprint_stat fs ON fs.fp_exact = af.fp_exact
GROUP BY af.fp_exact, fs.status
ORDER BY occurrence DESC, af.fp_exact;

-- name: ViewFingerprintImpact :many
-- 视图 B(某指纹的影响面,PRD 第六部分):
-- 出现次数、受影响项目数(affected_projects)、处置状态、已验证修复引用与验证边界、是否漂移。
-- 关联条款用 LEFT JOIN ... ON true:没有需求层(推不出条款)时该列为 NULL,行仍在,页面其余照常。
-- 条款来自:指纹的失败 →(build_run / deploy_run→制品→build_run)→ commit → commit_clause → clause_version。
-- drifted:已修复(fixed)但末次出现晚于修复验证时刻 —— 修复后又复发,说明修复可能过期/漂移。
WITH fp_commits AS (
  SELECT DISTINCT br.commit_sha
  FROM failure_event fe
  JOIN build_run br ON br.id = fe.build_run_id
  WHERE fe.fp_exact = $1
  UNION
  SELECT DISTINCT br.commit_sha
  FROM failure_event fe
  JOIN deploy_run dr ON dr.id = fe.deploy_run_id
  JOIN build_run br ON br.artifact_digest = dr.artifact_digest
  WHERE fe.fp_exact = $1
),
related AS (
  SELECT DISTINCT cv.clause_id, cv.content_hash
  FROM fp_commits fc
  JOIN commit_clause cc ON cc.commit_sha = fc.commit_sha
  JOIN clause_version cv ON cv.content_hash = cc.clause_content_hash
)
SELECT fs.fp_exact,
       fs.occurrence_count,
       fs.affected_projects,
       fs.status,
       fs.fix_ref,
       fs.fix_verified_at,
       fs.fix_verified_combo,
       (fs.status = 'fixed' AND fs.fix_verified_at IS NOT NULL AND fs.last_seen > fs.fix_verified_at) AS drifted,
       r.clause_id AS related_clause_id,
       r.content_hash AS related_clause_hash
FROM fingerprint_stat fs
LEFT JOIN related r ON true
WHERE fs.fp_exact = $1
ORDER BY r.clause_id;
