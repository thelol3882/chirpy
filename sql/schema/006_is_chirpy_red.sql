-- +goose Up
ALTER TABLE users ADD COLUMN is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROM COLUMN is_chirpy_red;
