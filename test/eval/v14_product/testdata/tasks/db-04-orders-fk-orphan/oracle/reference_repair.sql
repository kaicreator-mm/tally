-- db-04 reference repair candidate (new ChangeSet content; append-only
-- supersession of baseline/migrations/0003_orders_account_fk.sql).
-- Verification oracle only — never shipped to implementing agents.
-- Reassigns the orphan order to a real account BEFORE the constraint so
-- both evidence paths pass and the order row is preserved.
UPDATE orders SET account_id = 1 WHERE id = 102;

ALTER TABLE orders
    ADD CONSTRAINT orders_account_fk
    FOREIGN KEY (account_id) REFERENCES accounts (id);
