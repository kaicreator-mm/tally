-- db-04 baseline revision 0001 (PostgreSQL dialect). Deterministic;
-- no extensions, no seed data.
CREATE TABLE accounts (
    id    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email TEXT NOT NULL
);
