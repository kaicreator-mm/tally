-- TaskContract 派生索引的读写合同(P1-10)。
--
-- task_index 是索引面(派生数据,migrations/0018):TaskContract 的权威在 Git
-- (.tally/tasks/TASK-*.yaml,INV-ART-01),DB 只存派生索引(PRD §33.3),
-- 可整体删除重建,不是作者源(INV-STATE-01)。行 = 某项目某任务的一份内容修订
-- (revision 即 taskcontract.RevisionHash 内容修订哈希);is_current 标记当前投影,
-- 同一 (project_id, task_id) 至多一行(0018 partial unique index 强制)。
-- 本表无 status 列:Task lifecycle 状态是证据可追溯的派生事实(INV-STATE-02),
-- 不落索引面。
--
-- 重建合同(INV-STATE-01,一个事务内):
--   1. ClearTaskIndexCurrent:逐合同把现任 current 行退位(不然 step 2 撞
--      partial unique index);
--   2. RebuildTaskIndex:批量 upsert Git 快照的全部修订行(含新 current 声明);
--   3. DeleteStaleTaskIndex:删掉快照之外的旧行。
-- 重建后内容 = 快照,不残留手工/历史行。增量路径 = Clear + Upsert(旋转 current 时)。

-- name: UpsertTaskIndex :exec
-- 单合同修订派生入库(增量路径):登记或刷新一行,幂等可反复调。
-- source_commit 记录派生依据的制品提交;重复派生同一提交不产生副作用。
-- 注意:若本行带 is_current=true 而 (project_id, task_id) 已有别的 current 行,
-- partial unique index 会拒绝 —— 旋转 current 必须先 ClearTaskIndexCurrent
-- (或走整表重建三步),不能直接覆盖。
INSERT INTO task_index (project_id, task_id, revision, path, is_current, source_commit)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (project_id, task_id, revision) DO UPDATE SET
  path          = EXCLUDED.path,
  is_current    = EXCLUDED.is_current,
  source_commit = EXCLUDED.source_commit,
  derived_at    = now();

-- name: ClearTaskIndexCurrent :exec
-- 把某合同现任的 current 行全部退位(增量旋转 current 的前置步,重建 step 1)。
-- 只动索引面,不删行、不碰事实面。
UPDATE task_index
SET is_current = false
WHERE project_id = $1 AND task_id = $2 AND is_current;

-- name: RebuildTaskIndex :exec
-- 整体重建的批量 upsert(step 2):以给定 Git 快照铺满全部修订行。
-- 六个数组按位置一一对应(project_id/task_id/revision/path/is_current/source_commit,
-- 同 DeleteStaleTraceEdges / RebuildProjectIndex 的 unnest 惯例)。
-- 前置条件:快照内要旋转 current 的合同已先 ClearTaskIndexCurrent,
-- 否则同一合同的第二个 current 行会被 partial unique index 拒绝。
INSERT INTO task_index (project_id, task_id, revision, path, is_current, source_commit)
SELECT r.project_id, r.task_id, r.revision, r.path, r.is_current, r.source_commit
FROM (
  SELECT unnest($1::text[])   AS project_id,
         unnest($2::text[])   AS task_id,
         unnest($3::text[])   AS revision,
         unnest($4::text[])   AS path,
         unnest($5::boolean[]) AS is_current,
         unnest($6::text[])   AS source_commit
) AS r
ON CONFLICT (project_id, task_id, revision) DO UPDATE SET
  path          = EXCLUDED.path,
  is_current    = EXCLUDED.is_current,
  source_commit = EXCLUDED.source_commit,
  derived_at    = now();

-- name: DeleteStaleTaskIndex :exec
-- 重建的「清陈旧」一步(step 3,项目内):删掉不在给定快照三元组里的行。
-- 项目/合同从快照消失后其行不再出现在索引里;已落库的历史事实不动
-- (事实面各自持有 revision/hash 引用),只收口径,不抹历史。
DELETE FROM task_index ti
WHERE ti.project_id = $1
  AND NOT EXISTS (
    SELECT 1
    FROM (SELECT unnest($2::text[]) AS task_id,
                 unnest($3::text[]) AS revision) AS want
    WHERE want.task_id = ti.task_id
      AND want.revision = ti.revision
  );

-- name: ListTaskIndex :many
-- 全部已索引修订行(排序保证确定序):重算 --check 的 diff 与审计用。
SELECT project_id, task_id, revision, path, is_current, source_commit, derived_at
FROM task_index
ORDER BY project_id, task_id, revision;

-- name: ListCurrentTaskIndex :many
-- 全部当前投影行(排序保证确定序):任务列表/选择面用。
SELECT project_id, task_id, revision, path, is_current, source_commit, derived_at
FROM task_index
WHERE is_current
ORDER BY project_id, task_id;

-- name: GetCurrentTaskIndex :one
-- 某合同当前投影行:执行/验证侧问「该任务现在声明的是哪份合同」用。
-- 无 current 行时返回 pgx.ErrNoRows(未知任务不是错误状态,是索引缺席)。
SELECT project_id, task_id, revision, path, is_current, source_commit, derived_at
FROM task_index
WHERE project_id = $1 AND task_id = $2 AND is_current;

-- name: GetTaskIndexRevision :one
-- 按修订标识取行:事实面(trace/verification run)持有 revision 哈希引用,
-- 回查「当时是哪份合同、哪个路径」用。
SELECT project_id, task_id, revision, path, is_current, source_commit, derived_at
FROM task_index
WHERE project_id = $1 AND task_id = $2 AND revision = $3;
