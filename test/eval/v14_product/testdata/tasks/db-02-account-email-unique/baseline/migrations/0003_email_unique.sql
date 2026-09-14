-- db-02 baseline revision 0003 — THE REPAIRABLE MIGRATION DEFECT.
-- Adds a UNIQUE constraint while the seeded data carries duplicate e-mail
-- values. The two evidence paths diverge deterministically:
--   * fresh DB path: accounts is empty when 0003 runs -> succeeds;
--   * N-1 upgrade path: accounts carries two rows with the same e-mail
--     -> PostgreSQL fails deterministically:
--     ERROR: could not create unique constraint "accounts_email_unique"
--     DETAIL: Key (email)=(dup@example.test) is duplicated.
-- Repair discipline (Decision Lock 8/23): the repair ships as NEW candidate
-- content (oracle/reference_repair.sql) superseding this file append-only;
-- this defect file is never edited in place.
ALTER TABLE accounts
    ADD CONSTRAINT accounts_email_unique UNIQUE (email);
