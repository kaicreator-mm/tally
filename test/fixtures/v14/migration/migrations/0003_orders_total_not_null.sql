-- P3-02 fixture — THE REPAIRABLE MIGRATION DEFECT (task P3-02, AC3).
--
-- Adds a NOT NULL column without DEFAULT. The two PRD §24.2 evidence paths
-- diverge deterministically:
--
--   * fresh DB path: orders is empty when 0003 runs -> succeeds. Green here
--     is false confidence: it proves nothing about upgraded databases.
--   * N-1 upgrade path (seed/n_minus_1.sql applied after 0002): orders
--     carries rows -> PostgreSQL fails deterministically:
--     ERROR: column "total_cents" of relation "orders" contains null values.
--
-- Repair discipline (Decision Lock 8/23): a failed candidate is never
-- mutated in place. The repair ships as a NEW candidate ChangeSet content in
-- repair/0003_orders_total_not_null.sql, which supersedes this file
-- append-only; this defect file is kept as evidence.
ALTER TABLE orders
    ADD COLUMN total_cents BIGINT NOT NULL;
