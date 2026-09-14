-- P3-02 Database Migration harness fixture — PostgreSQL dialect.
-- Revision 0001: base schema. Deterministic; no extensions, no seed data.
CREATE TABLE accounts (
    id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email  TEXT NOT NULL UNIQUE
);
