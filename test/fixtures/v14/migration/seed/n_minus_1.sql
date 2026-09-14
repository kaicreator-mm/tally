-- P3-02 fixture — N-1 upgrade path data seed (data setup, NOT a migration).
-- Applied once after revision 0002 so revision 0003 upgrades a database that
-- carries rows. Fixed ids/timestamps keep every run deterministic.
INSERT INTO accounts (id, email) OVERRIDING SYSTEM VALUE
VALUES (1, 'seed-account-1@example.test'),
       (2, 'seed-account-2@example.test');

INSERT INTO orders (id, account_id, created_at) OVERRIDING SYSTEM VALUE
VALUES (100, 1, '2026-01-01 00:00:00+00'),
       (101, 2, '2026-01-02 00:00:00+00'),
       (102, 1, '2026-01-03 00:00:00+00');
