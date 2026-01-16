-- +goose Up
CREATE TABLE books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT NOT NULL,
    genders TEXT NOT NULL DEFAULT '[]',
    language TEXT NOT NULL,
    file_name TEXT NOT NULL UNIQUE,
    bookPath TEXT NOT NULL
);

-- +goose Down
DROP TABLE books;
