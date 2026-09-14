-- db-04 baseline revision 0002. Deliberately no FK yet; the N-1 seed
-- (baseline/seed/seed.sql) is applied here, before 0003 runs.
CREATE TABLE orders (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT NOT NULL,
    note       TEXT NOT NULL
);
