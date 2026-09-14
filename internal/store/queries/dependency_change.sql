-- dependency_change 的查询。事实面只 INSERT 和 SELECT,绝不 UPDATE/DELETE(INV-5)。

-- name: InsertDependencyChange :one
-- 记一条从锁文件 diff 解析出的依赖变更(失败时间线对齐用,不靠人填)。
INSERT INTO dependency_change (project_id, commit_sha, package, from_version, to_version, changed_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: InsertDependencyChangeIfAbsent :execrows
-- 幂等录入(T11a):唯一索引(project/commit/包/新旧版本)冲突即跳过 ——
-- CLI 重跑、区间重叠不产生重复行。ON CONFLICT DO NOTHING 仍是 INSERT,
-- 不触发事实面的 UPDATE/DELETE 拦截触发器(INV-5 无碍)。
INSERT INTO dependency_change (project_id, commit_sha, package, from_version, to_version, changed_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING;

-- name: CountDependencyChangesSince :one
-- 某项目自某时刻起的依赖变更数:失败与升级对齐的确定性信号,
-- 供归类判断「上游的问题」(PRD 第四部分规则 ④)。
SELECT count(*)
FROM dependency_change
WHERE project_id = $1 AND changed_at >= $2;

-- name: ListDependencyChangesRecent :many
-- 近 90 天的依赖变更:失败时间线的另一根轴,与按天失败数对齐显示 ——
-- 哪根失败柱子旁边有依赖刻度,哪次升级就可能是元凶。
SELECT id, project_id, commit_sha, package, from_version, to_version, changed_at
FROM dependency_change
WHERE changed_at >= now() - interval '90 days'
ORDER BY changed_at, id;
