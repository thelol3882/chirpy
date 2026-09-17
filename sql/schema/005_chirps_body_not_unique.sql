-- +goose Up
ALTER TABLE chirps DROP CONSTRAINT chirps_body_key;

-- +goose Down
ALTER TABLE chirps ADD CONSTRAINT chirps_body_key UNIQUE (body);
