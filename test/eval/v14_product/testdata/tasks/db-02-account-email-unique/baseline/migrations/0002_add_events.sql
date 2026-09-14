-- db-02 baseline revision 0002. The N-1 seed (baseline/seed/seed.sql) is
-- applied here, before 0003 runs.
CREATE TABLE events (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT NOT NULL,
    kind       TEXT NOT NULL
);
