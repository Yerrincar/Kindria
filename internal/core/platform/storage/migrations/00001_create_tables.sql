-- +goose Up
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT NOT NULL,
    genres TEXT NOT NULL DEFAULT '[]',
    language TEXT NOT NULL,
    file_name TEXT NOT NULL UNIQUE,
    bookPath TEXT NOT NULL,
    rating REAL
);

-- +goose Down
DROP TABLE books;
