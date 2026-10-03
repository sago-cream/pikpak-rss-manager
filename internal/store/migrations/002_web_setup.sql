CREATE TABLE IF NOT EXISTS administrator (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    password_hash TEXT NOT NULL,
    public_url TEXT NOT NULL DEFAULT '',
    allow_private_feeds INTEGER NOT NULL DEFAULT 0
);
PRAGMA user_version=2;
