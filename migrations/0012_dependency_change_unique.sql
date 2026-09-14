-- +goose Up

-- 采集幂等(T11a):dependency_change 是事实面只追加(不 DELETE),CLI 重跑、
-- 区间重叠都不该产生重复行。唯一索引兜底,录入用 ON CONFLICT DO NOTHING 跳过。
-- NULLS NOT DISTINCT:新增/移除的依赖 from_version/to_version 为 NULL,
-- 普通唯一索引里 NULL 互不相等,去重会失效。
CREATE UNIQUE INDEX dependency_change_unique_idx
  ON dependency_change (project_id, commit_sha, package, from_version, to_version)
  NULLS NOT DISTINCT;

-- +goose Down
DROP INDEX IF EXISTS dependency_change_unique_idx;
