-- db-02 reference repair candidate (new ChangeSet content; append-only
-- supersession of baseline/migrations/0003_email_unique.sql). Verification
-- oracle only — never shipped to implementing agents.
-- Disambiguates the duplicate row BEFORE the constraint, so both evidence
-- paths pass and every account keeps a row (no destructive deletion).
UPDATE accounts SET email = 'dup-2@example.test' WHERE id = 2;

ALTER TABLE accounts
    ADD CONSTRAINT accounts_email_unique UNIQUE (email);
