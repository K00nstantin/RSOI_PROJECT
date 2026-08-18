-- +goose Up
CREATE TABLE users (
    username TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL,  
    email TEXT,
    role TEXT NOT NULL DEFAULT 'User'
);

-- +goose Down
DROP TABLE users;