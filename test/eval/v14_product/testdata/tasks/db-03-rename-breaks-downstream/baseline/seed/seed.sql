-- db-03 N-1 upgrade path data seed (data setup, NOT a migration).
-- Applied after revision 0002; uses the post-rename column name.
INSERT INTO accounts (id, email_address) OVERRIDING SYSTEM VALUE
VALUES (1, 'a@example.test'),
       (2, 'b@example.test');
