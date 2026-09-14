-- db-06 reference repair candidate (new ChangeSet content; append-only
-- supersession of baseline/migrations/0003_narrow_total.sql).
-- Verification oracle only — never shipped to implementing agents.
-- Enforces the intended range invariant with a CHECK constraint instead of
-- an unsafe narrowing; existing wide values keep working on both paths.
ALTER TABLE orders
    ADD CONSTRAINT total_micros_nonneg CHECK (total_micros >= 0);
