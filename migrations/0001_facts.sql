-- +goose Up

-- 事实面五张表:只 INSERT,不 UPDATE 不 DELETE(INV-5)。
-- 双保险:sqlc 不生成这些表的 UPDATE/DELETE(架构测试 append_only_test 查),
-- 数据库层再加触发器阻止(本文件)。

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION tally_prevent_modify() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'fact table % is append-only: % is not allowed (INV-5)', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- environment_run:一次环境拉起
CREATE TABLE environment_run (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id          TEXT NOT NULL,
  env_name            TEXT NOT NULL,
  definition_hash     TEXT NOT NULL,               -- 环境定义内容哈希(INV-4)
  started_at          TIMESTAMPTZ NOT NULL,
  finished_at         TIMESTAMPTZ,
  status              TEXT NOT NULL CHECK (status IN ('up','failed','torn_down')),
  health_check_result JSONB,
  version_combo       JSONB NOT NULL               -- 当时版本组合,冗余自包含
);
CREATE TRIGGER environment_run_append_only
  BEFORE UPDATE OR DELETE ON environment_run
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- build_run:一次构建
CREATE TABLE build_run (
  id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id      TEXT NOT NULL,
  commit_sha      TEXT NOT NULL,
  ci_run_url      TEXT,
  started_at      TIMESTAMPTZ NOT NULL,
  finished_at     TIMESTAMPTZ,
  exit_code       INTEGER,
  status          TEXT NOT NULL,
  toolchain       TEXT,                            -- 用哪个抽取器解析日志
  artifact_digest TEXT,                            -- 成功时的制品,部署只认这个
  version_combo   JSONB NOT NULL
);
CREATE TRIGGER build_run_append_only
  BEFORE UPDATE OR DELETE ON build_run
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- deploy_run:一次部署
CREATE TABLE deploy_run (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id          TEXT NOT NULL,
  artifact_digest     TEXT NOT NULL,               -- 必填:没有 digest 的部署不允许发生
  build_run_id        BIGINT REFERENCES build_run(id),  -- 可空(制品可能来自外部)
  target_env          TEXT NOT NULL,
  tool                TEXT NOT NULL CHECK (tool IN ('kamal','ansible','compose')),
  started_at          TIMESTAMPTZ NOT NULL,
  finished_at         TIMESTAMPTZ,
  status              TEXT NOT NULL,
  exit_code           INTEGER,
  health_check_result JSONB,
  version_combo       JSONB NOT NULL
);
CREATE TRIGGER deploy_run_append_only
  BEFORE UPDATE OR DELETE ON deploy_run
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- failure_event:三个来源(构建/部署/环境)共用
CREATE TABLE failure_event (
  id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id         TEXT NOT NULL,
  source             TEXT NOT NULL CHECK (source IN ('build','deploy','environment')),
  build_run_id       BIGINT REFERENCES build_run(id),
  deploy_run_id      BIGINT REFERENCES deploy_run(id),
  environment_run_id BIGINT REFERENCES environment_run(id),
  raw_log_ref        TEXT,                         -- 原始日志位置,为规则改了能重跑
  raw_log_bytes      BIGINT,
  excerpt            TEXT,                         -- 抽取后关键片段(已脱敏)
  normalized         TEXT,                         -- 规范化后文本
  fp_exact           TEXT NOT NULL,                -- 只有它能关联已验证修复
  fp_fuzzy           TEXT NOT NULL,                -- 只能用于召回线索
  pipeline_version   TEXT NOT NULL,                -- 规则版本,参与指纹计算
  occurred_at        TIMESTAMPTZ NOT NULL,
  version_combo      JSONB NOT NULL,
  -- 三个来源外键恰好一个非空
  CONSTRAINT failure_event_one_source CHECK (
    (build_run_id IS NOT NULL)::int
    + (deploy_run_id IS NOT NULL)::int
    + (environment_run_id IS NOT NULL)::int = 1
  )
);
CREATE TRIGGER failure_event_append_only
  BEFORE UPDATE OR DELETE ON failure_event
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- dependency_change:从锁文件 diff 解析,失败时间线对齐用
CREATE TABLE dependency_change (
  id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  project_id   TEXT NOT NULL,
  commit_sha   TEXT NOT NULL,
  package      TEXT NOT NULL,
  from_version TEXT,
  to_version   TEXT,
  changed_at   TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER dependency_change_append_only
  BEFORE UPDATE OR DELETE ON dependency_change
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

-- 唯二现在就要的索引(查库热路径 + 列表页),见 docs/03-data-model.md
CREATE INDEX failure_event_fp_exact_idx ON failure_event (fp_exact);
CREATE INDEX failure_event_project_occurred_idx ON failure_event (project_id, occurred_at DESC);

-- +goose Down
DROP TABLE IF EXISTS dependency_change;
DROP TABLE IF EXISTS failure_event;
DROP TABLE IF EXISTS deploy_run;
DROP TABLE IF EXISTS build_run;
DROP TABLE IF EXISTS environment_run;
DROP FUNCTION IF EXISTS tally_prevent_modify();
