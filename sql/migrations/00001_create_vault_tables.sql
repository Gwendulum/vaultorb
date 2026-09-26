-- +goose Up
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL,
    username TEXT NOT NULL,
    encrypted_password BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL,
    UNIQUE(domain, username)
);

-- +goose Down
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS metadata;
