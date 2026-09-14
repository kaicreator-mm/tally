-- +goose Up

-- Job 租约(P0-01 / PRD §17.1 Job Lease / §37 P0 correctness):
-- worker 崩溃后 job 会卡在 running,需要能记录租约持有人、租约到期时间与心跳,
-- 让 stale-running 行可被安全接管。只加列,不动 0010 的既有语义
-- (队列不是事实面,允许 UPDATE,同 0010)。
--
--   worker_id    租约持有人(认领本行的 worker 实例标识)
--   locked_at    最近一次认领时间
--   lease_until  租约到期时间;过期即视为 stale,可被回收谓词接管
--   heartbeat_at 最近一次心跳时间
--
-- 约定:running 且 lease_until 为 NULL(迁移前的存量卡死行)或 lease_until <= now()
-- 视为 stale;活租约(lease_until > now())不被任何回收谓词触碰。
ALTER TABLE job
  ADD COLUMN worker_id    TEXT,
  ADD COLUMN locked_at    TIMESTAMPTZ,
  ADD COLUMN lease_until  TIMESTAMPTZ,
  ADD COLUMN heartbeat_at TIMESTAMPTZ;

-- 回收/接管扫描:按租约到期时间扫 stale-running 行(认领谓词共用)。
CREATE INDEX job_running_lease_idx ON job (lease_until) WHERE status = 'running';

-- +goose Down
DROP INDEX IF EXISTS job_running_lease_idx;
ALTER TABLE job
  DROP COLUMN IF EXISTS heartbeat_at,
  DROP COLUMN IF EXISTS lease_until,
  DROP COLUMN IF EXISTS locked_at,
  DROP COLUMN IF EXISTS worker_id;
