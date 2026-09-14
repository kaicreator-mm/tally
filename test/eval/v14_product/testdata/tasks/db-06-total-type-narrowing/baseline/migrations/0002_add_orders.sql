-- db-06 baseline revision 0002. Order totals use the wide type; the N-1
-- seed (baseline/seed/seed.sql) is applied here, before 0003 runs.
CREATE TABLE orders (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id   BIGINT NOT NULL,
    total_micros BIGINT NOT NULL DEFAULT 0
);
