UPDATE migration_test_users SET email = 'CHARLIE2@GMAIL.COM' WHERE username = 'charlie';
UPDATE migration_test_users SET email = 'BOB2@GMAIL.COM' WHERE username = 'bob';

DROP INDEX IF EXISTS idx_migration_test_users_email;
