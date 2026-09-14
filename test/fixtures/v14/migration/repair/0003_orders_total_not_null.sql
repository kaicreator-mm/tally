-- P3-02 fixture — repair candidate (new ChangeSet content; append-only
-- supersession of the defect migration migrations/0003_orders_total_not_null.sql,
-- which is never edited in place).
--
-- DEFAULT 0 makes the ADD COLUMN valid on both PRD §24.2 evidence paths:
--   * fresh DB: the column is born with the default;
--   * N-1 upgrade: the default backfills every pre-existing row without
--     touching identity or linkage columns (id, account_id, created_at),
--     so the data_integrity evidence key can be satisfied deterministically.
ALTER TABLE orders
    ADD COLUMN total_cents BIGINT NOT NULL DEFAULT 0;
