-- db-06 baseline revision 0003 — THE REPAIRABLE MIGRATION DEFECT.
-- Narrows total_micros from BIGINT to INTEGER while seeded data exceeds the
-- 32-bit range. The two evidence paths diverge deterministically:
--   * fresh DB path: orders is empty when 0003 runs -> succeeds;
--   * N-1 upgrade path: order 101 carries total_micros = 9999999999
--     -> PostgreSQL fails deterministically:
--     ERROR: integer out of range
-- Repair discipline (Decision Lock 8/23): the repair ships as NEW candidate
-- content (oracle/reference_repair.sql) superseding this file append-only.
ALTER TABLE orders
    ALTER COLUMN total_micros TYPE INTEGER;
