-- db-02 N-1 upgrade path data seed (data setup, NOT a migration).
-- Fixed ids keep every run deterministic. accounts rows 1 and 2 carry the
-- duplicate address that makes revision 0003 fail on the upgraded path.
INSERT INTO accounts (id, email) OVERRIDING SYSTEM VALUE
VALUES (1, 'dup@example.test'),
       (2, 'dup@example.test'),
       (3, 'solo@example.test');

INSERT INTO events (id, account_id, kind) OVERRIDING SYSTEM VALUE
VALUES (10, 1, 'created'),
       (11, 2, 'created'),
       (12, 3, 'created');
