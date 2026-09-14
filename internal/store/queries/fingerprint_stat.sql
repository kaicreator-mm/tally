-- fingerprint_stat 的查询。索引面(派生自 failure_event):可随时 drop 重算,
-- 允许全部操作 —— 与事实面不同,不受 INV-5 只追加约束。
-- status 由 failure 包的归类函数与 deps 的退休检查写,不由 HTTP 写
-- (INV-3 的登记例外,见 docs/03-data-model.md)。

-- name: GetFingerprintStat :one
-- 按精确指纹取索引行:诊断预算(三轮上限)、终态短路、跨项目判断都从这里读。
SELECT fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects,
       status, fix_ref, fix_verified_at, fix_verified_combo, diagnose_attempts
FROM fingerprint_stat
WHERE fp_exact = $1;

-- name: UpsertFingerprintStat :one
-- 失败发生时累加索引:新指纹建行,老指纹 +1 并更新 last_seen、并入受影响项目。
-- fix 相关列与 status 不在这里动 —— 那些由归类/入库流程显式写。
INSERT INTO fingerprint_stat (fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects)
VALUES ($1, $2, $3, $3, 1, ARRAY[sqlc.arg(project)::text]::TEXT[])
ON CONFLICT (fp_exact) DO UPDATE SET
  fp_fuzzy = EXCLUDED.fp_fuzzy,
  last_seen = EXCLUDED.last_seen,
  occurrence_count = fingerprint_stat.occurrence_count + 1,
  affected_projects = CASE
      WHEN sqlc.arg(project)::text = ANY(fingerprint_stat.affected_projects)
        THEN fingerprint_stat.affected_projects
      ELSE array_append(fingerprint_stat.affected_projects, sqlc.arg(project)::text)
    END
RETURNING fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects,
          status, fix_ref, fix_verified_at, fix_verified_combo, diagnose_attempts;

-- name: ListFingerprintStatsByFuzzy :many
-- 模糊指纹召回:返回所有同模糊指纹的索引行(各自带精确指纹与状态)。
-- knowledge 用它产出线索 —— 只给坐标和状态,给不出修复内容(修复只能由精确指纹关联)。
SELECT fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects,
       status, fix_ref, fix_verified_at, fix_verified_combo, diagnose_attempts
FROM fingerprint_stat
WHERE fp_fuzzy = $1
ORDER BY last_seen DESC;

-- name: ListFingerprintStats :many
-- 全部指纹索引行(最近出现在前):失败列表按指纹归并的数据源。
-- 修复排行(出现次数 × 影响项目数)在页面侧算 —— 排行是展示逻辑,不落库。
SELECT fp_exact, fp_fuzzy, first_seen, last_seen, occurrence_count, affected_projects,
       status, fix_ref, fix_verified_at, fix_verified_combo, diagnose_attempts
FROM fingerprint_stat
ORDER BY last_seen DESC, fp_exact;

-- name: UpdateFingerprintStatClassification :exec
-- 写归类出口(fixed/flaky/upstream/infra/spec_defect)。仅 failure 的归类函数调用。
UPDATE fingerprint_stat
SET status = $2
WHERE fp_exact = $1;

-- name: UpdateFingerprintStatFix :exec
-- 修复入库后回写:状态 fixed + 指向 git 里已验证修复的引用与验证边界(时刻、版本组合)。
UPDATE fingerprint_stat
SET status = 'fixed', fix_ref = $2, fix_verified_at = $3, fix_verified_combo = $4
WHERE fp_exact = $1;

-- name: ClearFingerprintStatFix :exec
-- 修复退休后清索引(D-c5,PRD 第五部分「修复会过期」):状态回 unresolved、
-- fix_ref / 验证边界清空、诊断次数归零。修复失效后同指纹再失败应重新走诊断,
-- 而不是撞「终态 fixed」短路。系统按确定性规则改写索引面(与归类同源先例),非手动改状态。
UPDATE fingerprint_stat
SET status = 'unresolved', fix_ref = NULL, fix_verified_at = NULL,
    fix_verified_combo = NULL, diagnose_attempts = 0
WHERE fp_exact = $1;

-- name: BumpDiagnoseAttempts :exec
-- 诊断预算:每跑一轮 +1,达到上限后同指纹不再调 LLM(PRD 第四部分规则 ⑤)。
UPDATE fingerprint_stat
SET diagnose_attempts = diagnose_attempts + 1
WHERE fp_exact = $1;
