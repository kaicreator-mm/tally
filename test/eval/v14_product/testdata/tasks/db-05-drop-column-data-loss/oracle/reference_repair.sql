-- db-05 reference repair candidate (new ChangeSet content; append-only
-- supersession of baseline/migrations/0003_drop_legacy_token.sql).
-- Verification oracle only — never shipped to implementing agents.
-- Expand/contract: the data is preserved into a new column BEFORE the old
-- column is dropped, so data_integrity holds on both evidence paths.
ALTER TABLE accounts
    ADD COLUMN credential TEXT;

UPDATE accounts SET credential = legacy_token;

ALTER TABLE accounts
    DROP COLUMN legacy_token;
