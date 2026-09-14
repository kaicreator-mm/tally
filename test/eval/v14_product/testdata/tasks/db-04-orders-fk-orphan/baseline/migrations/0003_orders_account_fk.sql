-- db-04 baseline revision 0003 — THE REPAIRABLE MIGRATION DEFECT.
-- Adds a foreign key while the seeded orders carry an orphan account_id.
-- The two evidence paths diverge deterministically:
--   * fresh DB path: orders is empty when 0003 runs -> succeeds;
--   * N-1 upgrade path: order 102 references account 999, which does not
--     exist -> PostgreSQL fails deterministically:
--     ERROR: insert or update on table "orders" violates foreign key
--     constraint "orders_account_fk"
-- Repair discipline (Decision Lock 8/23): the repair ships as NEW candidate
-- content (oracle/reference_repair.sql) superseding this file append-only.
ALTER TABLE orders
    ADD CONSTRAINT orders_account_fk
    FOREIGN KEY (account_id) REFERENCES accounts (id);
