-- +goose Up

-- P0-02:追溯身份的项目命名空间(PRD §22:所有 project-specific node 必须带 project_id)。
--
-- commit_clause / test_clause / trace_edge 都是索引面(派生数据,0002):身份原先不带项目,
-- 同 commit_sha / test_name / 边 tuple 在并行共享库里跨项目合并成一回事。
--
-- 迁移策略(数据安全,各表分开处理):
--   - 挂钩表原地回填 project_id:clause_content_hash 经外键落在 clause_version.content_hash
--     (全局唯一),归属项目有且只有一个,回填必然命中;回填后再 SET NOT NULL,由数据库
--     兜底 —— 任何一行归属不了,迁移显式失败,而不是悄悄写脏。
--   - trace_edge 的键(build/deploy 自增 id、制品摘要、失败指纹)跨项目共享,逐行归属
--     不可判定;整表本就允许随时 drop 重算(0002 定义,INV-STATE-01 可重建),按新身份
--     重建同构表,内容由下一次 reindex 从机器记录全量重铺,不在此伪造归属。
--
-- 新身份(主键全部以 project_id 打头):
--   commit_clause (project_id, commit_sha, clause_content_hash)
--   test_clause   (project_id, test_name, clause_content_hash)
--   trace_edge    (project_id, from_type, from_key, to_type, to_key)

-- 挂钩表:加列 → 回填 → 收紧 NOT NULL → 换主键。content_hash 全局唯一保证回填一对一。
ALTER TABLE commit_clause ADD COLUMN project_id TEXT;
UPDATE commit_clause
SET project_id = (
  SELECT cv.project_id FROM clause_version cv
  WHERE cv.content_hash = commit_clause.clause_content_hash
);
ALTER TABLE commit_clause ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE commit_clause DROP CONSTRAINT commit_clause_pkey;
ALTER TABLE commit_clause ADD PRIMARY KEY (project_id, commit_sha, clause_content_hash);

ALTER TABLE test_clause ADD COLUMN project_id TEXT;
UPDATE test_clause
SET project_id = (
  SELECT cv.project_id FROM clause_version cv
  WHERE cv.content_hash = test_clause.clause_content_hash
);
ALTER TABLE test_clause ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE test_clause DROP CONSTRAINT test_clause_pkey;
ALTER TABLE test_clause ADD PRIMARY KEY (project_id, test_name, clause_content_hash);

-- trace_edge:物化追溯图,派生数据,可完全删除重建(同 0002 的表形状,仅身份加 project_id;
-- 无次级索引,重建不丢任何索引语义)。旧数据不迁移:边的键跨项目共享,归属不可判定。
DROP TABLE trace_edge;
CREATE TABLE trace_edge (
  project_id TEXT NOT NULL,
  from_type  TEXT NOT NULL,
  from_key   TEXT NOT NULL,
  to_type    TEXT NOT NULL,
  to_key     TEXT NOT NULL,
  PRIMARY KEY (project_id, from_type, from_key, to_type, to_key)
);

-- +goose Down

-- 回到无命名空间身份(0015 之前)。0015 之前同身份本就合并为一行,按原样还原;
-- 若 0015 期间写入过同身份不同项目的多行,还原主键会显式报错 —— 由操作者先重算/清空
-- 索引面再回滚(派生数据可随时重建),不做静默去重。
DROP TABLE trace_edge;
CREATE TABLE trace_edge (
  from_type TEXT NOT NULL,
  from_key  TEXT NOT NULL,
  to_type   TEXT NOT NULL,
  to_key    TEXT NOT NULL,
  PRIMARY KEY (from_type, from_key, to_type, to_key)
);

ALTER TABLE test_clause DROP CONSTRAINT test_clause_pkey;
ALTER TABLE test_clause DROP COLUMN project_id;
ALTER TABLE test_clause ADD PRIMARY KEY (test_name, clause_content_hash);

ALTER TABLE commit_clause DROP CONSTRAINT commit_clause_pkey;
ALTER TABLE commit_clause DROP COLUMN project_id;
ALTER TABLE commit_clause ADD PRIMARY KEY (commit_sha, clause_content_hash);
