-- +goose Up

-- 队列:Postgres 表 + 进程内 worker。任务类型是固定枚举(设计期定死,不支持运行时注册)。
-- 队列不是事实面,允许 UPDATE(状态流转 pending→running→done/failed)。
CREATE TABLE job (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  kind        TEXT NOT NULL,               -- 任务类型枚举,见 internal/jobs
  payload     JSONB NOT NULL DEFAULT '{}',
  status      TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending','running','done','failed')),
  attempts    INTEGER NOT NULL DEFAULT 0,
  last_error  TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at  TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);
-- 出队热路径:按状态+创建时间取最早
CREATE INDEX job_pending_idx ON job (status, created_at);

-- +goose Down
DROP TABLE IF EXISTS job;
