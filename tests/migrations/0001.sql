CREATE TABLE IF NOT EXISTS migration_test_users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO migration_test_users (username, email) VALUES
('alice', 'ALICE@GMAIL.COM'),
('charlie', 'CHARLIE@GMAIL.COM'),
('bob', 'BOB@GMAIL.COM');

CREATE INDEX idx_migration_test_users_email ON migration_test_users (email);