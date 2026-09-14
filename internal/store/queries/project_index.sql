-- 派生项目索引的读写合同(P0-08)。
--
-- project_index 是索引面(派生数据,migrations/0017):从 Git 项目注册表
-- (.tally/project.yaml)派生、可整体重建,不是作者源(INV-STATE-01)。
-- 事实面五张表历史上接受任意 project_id 文本;本表提供「已知项目」判定:
-- 所有 ingest 落库前先查 ProjectIndexExists,注册表之外的 project_id 一律拒收,
-- 避免 ghost project facts(§33.3)。

-- name: UpsertProjectIndex :exec
-- 单项目派生入库(注册表扫描的增量路径):登记或刷新一行,幂等可反复调。
-- source_commit 记录派生依据的注册表提交;重复派生同一提交不产生副作用,
-- 派生新提交则刷新依据行(derived_at 同步刷新,标记最近一次重算)。
INSERT INTO project_index (project_id, source_commit)
VALUES ($1, $2)
ON CONFLICT (project_id) DO UPDATE SET
  source_commit = EXCLUDED.source_commit,
  derived_at    = now();

-- name: RebuildProjectIndex :exec
-- 全量重建的批量 upsert:以给定注册表快照(项目 id 数组 + 对位 source_commit 数组,
-- 按位置一一对应,同 DeleteStaleTraceEdges 的 unnest 惯例)铺满快照内的全部项目。
-- 配 DeleteStaleProjectIndex 清掉快照之外的项目行,即完成一次整表重建(INV-STATE-01):
-- 重建后内容 = 注册表快照,不残留任何手工/历史行。
INSERT INTO project_index (project_id, source_commit)
SELECT reg.project_id, reg.source_commit
FROM (
  SELECT unnest($1::text[]) AS project_id,
         unnest($2::text[]) AS source_commit
) AS reg
ON CONFLICT (project_id) DO UPDATE SET
  source_commit = EXCLUDED.source_commit,
  derived_at    = now();

-- name: DeleteStaleProjectIndex :exec
-- 重建的「清陈旧」一步:删掉不在给定注册表快照里的项目行。
-- 项目从注册表移除后,其事实不再放行( ProjectIndexExists 落 false),
-- 已落库的历史事实不动 —— 只收口径,不抹历史。
DELETE FROM project_index pi
WHERE NOT EXISTS (
  SELECT 1
  FROM (SELECT unnest($1::text[]) AS project_id) AS want
  WHERE want.project_id = pi.project_id
);

-- name: ListProjectIndex :many
-- 全部已知项目(排序保证确定序):项目列表/选择面与重算 --check 的 diff 用。
SELECT project_id, source_commit, derived_at
FROM project_index
ORDER BY project_id;

-- name: ProjectIndexExists :one
-- ghost-project guard 原语(§33.3):项目在注册表派生索引中即 true。
-- ingest 落库前必须先查这个;false 即未知项目,事实拒收,不落任何事实表。
SELECT EXISTS (
  SELECT 1
  FROM project_index
  WHERE project_id = $1
) AS known;
