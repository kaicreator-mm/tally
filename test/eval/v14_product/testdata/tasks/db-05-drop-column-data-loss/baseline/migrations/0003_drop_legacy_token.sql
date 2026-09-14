-- db-05 baseline revision 0003 — THE REPAIRABLE MIGRATION DEFECT.
-- Drops a column that still carries live data. The upgrade SQL itself is
-- valid on both paths — the damage is silent, which is exactly why the
-- policy demands a separate data_integrity evidence key:
--   * fresh DB path: accounts is empty, nothing is lost -> succeeds;
--   * N-1 upgrade path: every seeded account loses its legacy_token value
--     -> upgrade succeeds but seeded data is destroyed.
-- Repair discipline (Decision Lock 8/23): the repair ships as NEW candidate
-- content (oracle/reference_repair.sql) superseding this file append-only.
ALTER TABLE accounts
    DROP COLUMN legacy_token;
