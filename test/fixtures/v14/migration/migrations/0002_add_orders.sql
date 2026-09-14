-- P3-02 Database Migration harness fixture — PostgreSQL dialect.
-- Revision 0002: orders reference accounts. This is the N-1 revision:
-- the upgrade path seeds real data (seed/n_minus_1.sql) HERE, before 0003 runs.
CREATE TABLE orders (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  BIGINT NOT NULL REFERENCES accounts (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
