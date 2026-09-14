-- outcome_event 的读写合同(P3-05;migrations/0023,PRD §23 / 决策锁 14)。
--
-- outcome_event 是 outcome 事实面:每个 outcome(accepted / reverted /
-- hotfixed / regression_found / human_correction)落成一条带 kind 的事件行,
-- 不在任务上落任何可变 outcome 布尔/状态列。本文件只有 INSERT 和 SELECT:
-- 没有任何 UPDATE/DELETE(锁 14 / INV-FACT-01,数据库层另有语句级触发器
-- 兜底),outcome 事实落库后只读;「当前 outcome」由事件流在读侧投影,
-- 库内不存在也不允许出现可变投影列(AC5)。
--
-- 绑定口径(P3-05 AC2):task_id / changeset_id 可空 —— 绑定可得时必填,
-- 上下文外的事件(如事故热修)先落事实;未知 kind 的写入被 0023 的 CHECK
-- 在数据库层显式拒绝(fail-safe),不被静默折叠进已知 kind。

-- name: InsertOutcomeEvent :one
-- 追加一条 outcome 事实(AC2):事件、绑定坐标、写入时间一次落成,返回新行
-- id。事实面唯一写路径:INSERT 且只 INSERT。event_kind 由 0023 的 CHECK 限定
-- 为 PRD §23 的五个封闭枚举;task_id / changeset_id 可空(可得时必填);
-- recorded_at 写入方显式给出,无默认。
INSERT INTO outcome_event (task_id, changeset_id, event_kind, recorded_at)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: InsertOutcomeEventOnce :one
-- 幂等捕获路径:同一完整事实坐标在数据库唯一约束下最多一行。冲突时
-- RETURNING 无行(sqlc/pgx => ErrNoRows),由 outcome.Store 解释为成功重放。
INSERT INTO outcome_event (task_id, changeset_id, event_kind, recorded_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT DO NOTHING
RETURNING id;

-- name: ListOutcomeEventsByTask :many
-- 某任务全部 outcome 事件(旧→新,recorded_at 同刻按 id 定序):同一任务的
-- accepted→regression_found→… 完整历史可重放,「当前 outcome」由读侧按事件
-- 流推导,不在库内维护任何可变投影(AC5)。
SELECT id, task_id, changeset_id, event_kind, recorded_at
FROM outcome_event
WHERE task_id = $1
ORDER BY recorded_at, id;

-- name: ListOutcomeEventsByChangeset :many
-- 某个候选 ChangeSet 关联的全部 outcome 事件(旧→新,同刻按 id 定序):
-- 候选维度的 outcome 审计(该候选被接受过没有、后来是否被回退/发现回归)。
SELECT id, task_id, changeset_id, event_kind, recorded_at
FROM outcome_event
WHERE changeset_id = $1
ORDER BY recorded_at, id;
