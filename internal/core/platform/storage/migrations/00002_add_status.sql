-- +goose Up
ALTER TABLE books ADD status TEXT NOT NULL DEFAULT 'Not defined yet';

-- +goose Down
ALTER TABLE books DROP COLUMN status;
