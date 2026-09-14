-- db-06 N-1 upgrade path data seed (data setup, NOT a migration).
-- Order 101 carries a value outside the 32-bit integer range.
INSERT INTO accounts (id, email) OVERRIDING SYSTEM VALUE
VALUES (1, 'a@example.test'),
       (2, 'b@example.test');

INSERT INTO orders (id, account_id, total_micros) OVERRIDING SYSTEM VALUE
VALUES (100, 1, 1500000),
       (101, 2, 9999999999);
