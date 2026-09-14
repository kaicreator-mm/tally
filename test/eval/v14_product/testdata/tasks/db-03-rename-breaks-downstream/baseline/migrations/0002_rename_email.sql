-- db-03 baseline revision 0002. Renames the contact column; the N-1 seed
-- (baseline/seed/seed.sql) is applied here, before 0003 runs.
ALTER TABLE accounts
    RENAME COLUMN email TO email_address;
