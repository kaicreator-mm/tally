-- name: InsertJob :one
INSERT INTO job (kind, payload)
VALUES ($1, $2)
RETURNING id;

-- name: DequeueJob :one
-- 取出最早的一个待处理任务并锁住行;并发 worker 靠 FOR UPDATE SKIP LOCKED 不争抢。
SELECT id, kind, payload, attempts
FROM job
WHERE status = 'pending'
ORDER BY created_at
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: MarkJobRunning :exec
UPDATE job
SET status = 'running', attempts = attempts + 1, started_at = now()
WHERE id = $1;

-- name: MarkJobDone :exec
UPDATE job
SET status = 'done', finished_at = now()
WHERE id = $1;

-- name: MarkJobFailed :exec
UPDATE job
SET status = 'failed', finished_at = now(), last_error = $2
WHERE id = $1;

-- ===== 租约原语(P0-01 / PRD §17.1 Job Lease)=====
--
-- stale 约定(与 migrations/0014_job_lease.sql 一致):
--   running 且 (lease_until IS NULL OR lease_until <= now()) 即 stale;
--   活租约(lease_until > now())不被任何认领/回收谓词触碰。

-- name: ClaimNextJob :one
-- 原子认领最早的一个可运行任务:pending,或 stale 的 running(worker 崩溃遗留)。
-- 单条 UPDATE 内嵌 FOR UPDATE SKIP LOCKED 子查询,并发 worker 不争抢、不重复认领;
-- 认领即续租(写入新租约窗口)。需在事务中调用,与 0010 的 DequeueJob 同一事务语义。
UPDATE job SET
  status       = 'running',
  attempts     = attempts + 1,
  started_at   = now(),
  worker_id    = $1,
  locked_at    = now(),
  lease_until  = now() + $2::interval,
  heartbeat_at = now()
WHERE id = (
  SELECT id FROM job
  WHERE status = 'pending'
     OR (status = 'running' AND (lease_until IS NULL OR lease_until <= now()))
  ORDER BY created_at
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING id, kind, payload, attempts;

-- name: RequeueStaleJobs :exec
-- 回收扫描:把全部 stale 的 running 任务退回 pending,等待重新认领(供后台巡检使用)。
-- 活租约不满足谓词,不会被回收;FOR UPDATE SKIP LOCKED 与并发认领互不阻塞、不重复处理。
-- attempts 不在此处递增:重新认领时由 ClaimNextJob 计数。
UPDATE job SET
  status       = 'pending',
  worker_id    = NULL,
  locked_at    = NULL,
  lease_until  = NULL,
  heartbeat_at = NULL
WHERE id IN (
  SELECT id FROM job
  WHERE status = 'running'
    AND (lease_until IS NULL OR lease_until <= now())
  FOR UPDATE SKIP LOCKED
);

-- name: HeartbeatJob :execrows
-- 心跳续租:仅在任务仍归该 worker 所有、仍为 running 且租约仍存活(lease_until > now())
-- 时延长租约。过期后不允许复活:失联恢复的 worker 心跳命中 0 行(RowsAffected=0),
-- 必须放弃执行,任务由回收路径接管。
UPDATE job
SET heartbeat_at = now(), lease_until = now() + $3::interval
WHERE id = $1 AND worker_id = $2 AND status = 'running' AND lease_until > now();
