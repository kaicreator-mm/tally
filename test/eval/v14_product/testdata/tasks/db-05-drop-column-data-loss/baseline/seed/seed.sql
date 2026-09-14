-- db-05 N-1 upgrade path data seed (data setup, NOT a migration).
-- Every account carries a legacy_token value that must not be destroyed.
INSERT INTO accounts (id, email, legacy_token) OVERRIDING SYSTEM VALUE
VALUES (1, 'a@example.test', 'tok-1'),
       (2, 'b@example.test', 'tok-2');

INSERT INTO sessions (id, account_id, token) OVERRIDING SYSTEM VALUE
VALUES (20, 1, 'sess-1'),
       (21, 2, 'sess-2');
