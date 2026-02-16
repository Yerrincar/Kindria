-- +goose Up
ALTER TABLE books ADD reading_date TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE books DROP COLUMN reading_date;
