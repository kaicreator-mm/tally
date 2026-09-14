-- db-04 N-1 upgrade path data seed (data setup, NOT a migration).
-- Order 102 references account 999, which no seeded account provides.
INSERT INTO accounts (id, email) OVERRIDING SYSTEM VALUE
VALUES (1, 'a@example.test'),
       (2, 'b@example.test');

INSERT INTO orders (id, account_id, note) OVERRIDING SYSTEM VALUE
VALUES (100, 1, 'first'),
       (101, 2, 'second'),
       (102, 999, 'orphaned-by-import');
