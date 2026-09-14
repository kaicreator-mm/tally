-- db-03 baseline revision 0003 — THE REPAIRABLE MIGRATION DEFECT.
-- Still references the pre-rename column name, so the statement fails on
-- BOTH evidence paths, deterministically:
--   ERROR: column "email" of relation "accounts" does not exist
-- A rename is not a no-op: downstream objects must follow the new name.
-- Repair discipline (Decision Lock 8/23): the repair ships as NEW candidate
-- content (oracle/reference_repair.sql) superseding this file append-only.
CREATE INDEX idx_accounts_email
    ON accounts (email);
