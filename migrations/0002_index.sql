-- +goose Up

-- 索引面:派生数据,随时可 drop 重算(make reindex)。
-- 挂钩一律记内容哈希,不记逻辑编号(INV-4)。

-- clause:逻辑条款(编号)
CREATE TABLE clause (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id TEXT NOT NULL,
  clause_id  TEXT NOT NULL,              -- 如 REQ-12
  UNIQUE (project_id, clause_id)
);

-- clause_version:条款的某一版(不可变)。所有边指向 content_hash。
CREATE TABLE clause_version (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id          TEXT NOT NULL,
  clause_id           TEXT NOT NULL,
  content_hash        TEXT NOT NULL UNIQUE,        -- 所有边指向这个(INV-4)
  version_no          INTEGER NOT NULL,
  signed_at           TIMESTAMPTZ,
  signed_by           TEXT,
  signed_commit       TEXT,
  text                TEXT,                         -- 仅展示,权威版本在 git
  acceptance_criteria TEXT,
  non_goals           TEXT,
  superseded_by_hash  TEXT REFERENCES clause_version(content_hash),  -- 变更链,记哈希
  is_current          BOOLEAN NOT NULL DEFAULT false,
  FOREIGN KEY (project_id, clause_id) REFERENCES clause(project_id, clause_id)
);

-- commit_clause:来自 commit trailer `Refs: REQ-12`,按当时 is_current 版本绑定后不变
CREATE TABLE commit_clause (
  commit_sha          TEXT NOT NULL,
  clause_content_hash TEXT NOT NULL REFERENCES clause_version(content_hash),
  PRIMARY KEY (commit_sha, clause_content_hash)
);

-- test_clause:来自测试注解 `@covers REQ-12.2`,用于算条款覆盖率
CREATE TABLE test_clause (
  test_name           TEXT NOT NULL,
  clause_content_hash TEXT NOT NULL REFERENCES clause_version(content_hash),
  PRIMARY KEY (test_name, clause_content_hash)
);

-- fingerprint_stat:派生自 failure_event。
-- fix_ref 只能由 fp_exact 关联(本表主键即 fp_exact,结构上模糊指纹关联不进来)。
CREATE TABLE fingerprint_stat (
  fp_exact           TEXT PRIMARY KEY,
  fp_fuzzy           TEXT NOT NULL,
  first_seen         TIMESTAMPTZ NOT NULL,
  last_seen          TIMESTAMPTZ NOT NULL,
  occurrence_count   BIGINT NOT NULL DEFAULT 1,
  affected_projects  TEXT[] NOT NULL DEFAULT '{}',
  status             TEXT NOT NULL DEFAULT 'unresolved'
    CHECK (status IN ('unresolved','fixed','flaky','upstream','infra','spec_defect')),
  fix_ref            TEXT,                          -- 指向 git 里的已验证修复
  fix_verified_at    TIMESTAMPTZ,
  fix_verified_combo JSONB,
  diagnose_attempts  INTEGER NOT NULL DEFAULT 0     -- 三轮上限用
);

-- trace_edge:物化追溯图,加速视图 A/B。可完全删除重建。
CREATE TABLE trace_edge (
  from_type TEXT NOT NULL,
  from_key  TEXT NOT NULL,
  to_type   TEXT NOT NULL,
  to_key    TEXT NOT NULL,
  PRIMARY KEY (from_type, from_key, to_type, to_key)
);

-- +goose Down
DROP TABLE IF EXISTS trace_edge;
DROP TABLE IF EXISTS fingerprint_stat;
DROP TABLE IF EXISTS test_clause;
DROP TABLE IF EXISTS commit_clause;
DROP TABLE IF EXISTS clause_version;
DROP TABLE IF EXISTS clause;
