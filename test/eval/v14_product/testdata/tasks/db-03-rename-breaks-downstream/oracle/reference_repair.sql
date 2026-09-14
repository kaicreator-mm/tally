-- db-03 reference repair candidate (new ChangeSet content; append-only
-- supersession of baseline/migrations/0003_index_email.sql). Verification
-- oracle only — never shipped to implementing agents.
-- The downstream index follows the renamed column.
CREATE INDEX idx_accounts_email
    ON accounts (email_address);
